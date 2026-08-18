package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedChannelPoolUsage(t *testing.T, channelID int, includeCache bool) *model.ChannelSubscriptionPool {
	t.Helper()
	pool := &model.ChannelSubscriptionPool{
		Name: "usage", PlanId: 1, PlanTitle: "shared",
		AmountTotal: 10000, TokensTotal: 10000, IncludeCacheTokens: includeCache,
		StartTime: time.Now().Add(-time.Minute).Unix(), EndTime: time.Now().Add(time.Hour).Unix(), Status: model.ChannelPoolStatusActive,
	}
	require.NoError(t, model.DB.Create(pool).Error)
	require.NoError(t, model.DB.Create(&model.ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: channelID}).Error)
	return pool
}

func getChannelPoolUsage(t *testing.T, poolID int) *model.ChannelSubscriptionPool {
	t.Helper()
	pool, err := model.GetChannelSubscriptionPool(poolID)
	require.NoError(t, err)
	return pool
}

func TestSettleChannelPoolActualUsageAppliesIndependentCachePolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		includeCache bool
		wantTokens   int64
	}{
		{name: "exclude cache", includeCache: false, wantTokens: 150},
		{name: "include cache", includeCache: true, wantTokens: 190},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			channelID := 8100
			seedChannel(t, channelID)
			pool := seedChannelPoolUsage(t, channelID, tc.includeCache)

			err := SettleChannelPoolActualUsage(context.Background(), channelID, "request:usage", 75, 100, 50, 40)

			require.NoError(t, err)
			got := getChannelPoolUsage(t, pool.Id)
			assert.EqualValues(t, 75, got.AmountUsed)
			assert.Equal(t, tc.wantTokens, got.TokensUsed)
		})
	}
}

func TestSettleUserSubscriptionActualUsageAppliesCachePolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		includeCache bool
		wantTokens   int64
	}{
		{name: "exclude cache", includeCache: false, wantTokens: 150},
		{name: "include cache", includeCache: true, wantTokens: 190},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			seedUser(t, 9101, 10000)
			sub := &model.UserSubscription{
				UserId:             9101,
				AmountTotal:        10000,
				TokensTotal:        10000,
				IncludeCacheTokens: tc.includeCache,
				Status:             "active",
				StartTime:          time.Now().Unix() - 60,
				EndTime:            time.Now().Add(time.Hour).Unix(),
			}
			require.NoError(t, model.DB.Create(sub).Error)

			info := &relaycommon.RelayInfo{
				BillingSource:                  BillingSourceSubscription,
				SubscriptionId:                 sub.Id,
				SubscriptionIncludeCacheTokens: tc.includeCache,
			}
			require.NoError(t, SettleUserSubscriptionActualUsage(info, 100, 50, 40))

			var got model.UserSubscription
			require.NoError(t, model.DB.First(&got, sub.Id).Error)
			assert.EqualValues(t, tc.wantTokens, got.TokensUsed)
			assert.Zero(t, got.AmountUsed)
		})
	}
}

func TestSettleUserSubscriptionActualUsageSkipsWalletAndMissingId(t *testing.T) {
	truncate(t)
	seedUser(t, 9102, 10000)
	sub := &model.UserSubscription{
		UserId:      9102,
		AmountTotal: 10000,
		TokensTotal: 10000,
		Status:      "active",
		StartTime:   time.Now().Unix() - 60,
		EndTime:     time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, model.DB.Create(sub).Error)

	require.NoError(t, SettleUserSubscriptionActualUsage(&relaycommon.RelayInfo{
		BillingSource:  BillingSourceWallet,
		SubscriptionId: sub.Id,
	}, 100, 50, 40))
	require.NoError(t, SettleUserSubscriptionActualUsage(&relaycommon.RelayInfo{
		BillingSource: BillingSourceSubscription,
	}, 100, 50, 40))
	require.NoError(t, SettleUserSubscriptionActualUsage(nil, 100, 50, 40))

	var got model.UserSubscription
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.Zero(t, got.TokensUsed)
}

func TestSettleChannelPoolActualUsageWithoutPoolIsNoop(t *testing.T) {
	truncate(t)
	channelID := 8150
	seedChannel(t, channelID)

	require.NoError(t, SettleChannelPoolActualUsage(context.Background(), channelID, "request:no-pool", 75, 100, 50, 40))

	var settlements int64
	require.NoError(t, model.DB.Model(&model.ChannelPoolSettlement{}).Count(&settlements).Error)
	assert.Zero(t, settlements)
}

type fixedTaskBillingAdaptor struct {
	actualQuota int
}

func (a fixedTaskBillingAdaptor) Init(*relaycommon.RelayInfo) {}
func (a fixedTaskBillingAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	return nil, nil
}
func (a fixedTaskBillingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return nil, nil
}
func (a fixedTaskBillingAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return a.actualQuota
}

func TestTaskCompletionSettlesActualTokensNotQuota(t *testing.T) {
	truncate(t)
	channelID := 8250
	seedChannel(t, channelID)
	pool := seedChannelPoolUsage(t, channelID, false)
	task := makeTask(1, channelID, 60, 0, BillingSourceWallet, 0)
	task.TaskID = "task_terminal_tokens"
	seedUser(t, task.UserId, 10000)
	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 60, 0))

	settleTaskBillingOnComplete(context.Background(), fixedTaskBillingAdaptor{actualQuota: 90}, task, &relaycommon.TaskInfo{TotalTokens: 321})

	got := getChannelPoolUsage(t, pool.Id)
	assert.EqualValues(t, 90, got.AmountUsed)
	assert.EqualValues(t, 321, got.TokensUsed)
}

func TestSettleTaskChannelPoolUsageIsCumulativeAndRefundable(t *testing.T) {
	truncate(t)
	channelID := 8200
	seedChannel(t, channelID)
	pool := seedChannelPoolUsage(t, channelID, false)
	task := makeTask(1, channelID, 60, 0, BillingSourceWallet, 0)
	task.TaskID = "task_stable_pool_usage"

	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 60, 0))
	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 90, 300))
	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 90, 300))
	got := getChannelPoolUsage(t, pool.Id)
	assert.EqualValues(t, 90, got.AmountUsed)
	assert.EqualValues(t, 300, got.TokensUsed)

	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 0, 0))
	require.NoError(t, SettleTaskChannelPoolUsage(context.Background(), task, 0, 0))
	got = getChannelPoolUsage(t, pool.Id)
	assert.Zero(t, got.AmountUsed)
	assert.Zero(t, got.TokensUsed)

	var settlements int64
	require.NoError(t, model.DB.Model(&model.ChannelPoolSettlement{}).Where("settlement_key = ?", "task:"+task.TaskID).Count(&settlements).Error)
	assert.EqualValues(t, 1, settlements)
}

func seedUserSubscriptionForUsage(t *testing.T, userID int, includeCache bool) *model.UserSubscription {
	t.Helper()
	sub := &model.UserSubscription{
		UserId:             userID,
		AmountTotal:        10000,
		TokensTotal:        10000,
		IncludeCacheTokens: includeCache,
		Status:             "active",
		StartTime:          time.Now().Unix() - 60,
		EndTime:            time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, model.DB.Create(sub).Error)
	return sub
}

func TestSettleTaskUserSubscriptionUsageIsIdempotentByTaskKey(t *testing.T) {
	truncate(t)
	const userID = 9301
	seedUser(t, userID, 10000)
	sub := seedUserSubscriptionForUsage(t, userID, false)
	task := makeTask(userID, 1, 60, 0, BillingSourceSubscription, sub.Id)
	task.TaskID = "task_user_sub_idempotent"

	require.NoError(t, SettleTaskUserSubscriptionUsage(context.Background(), task, 321))
	require.NoError(t, SettleTaskUserSubscriptionUsage(context.Background(), task, 321))
	var got model.UserSubscription
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.EqualValues(t, 321, got.TokensUsed)

	require.NoError(t, SettleTaskUserSubscriptionUsage(context.Background(), task, 0))
	require.NoError(t, SettleTaskUserSubscriptionUsage(context.Background(), task, 0))
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.Zero(t, got.TokensUsed)

	var rows int64
	require.NoError(t, model.DB.Model(&model.UserSubscriptionSettlement{}).Where("settlement_key = ?", "task:"+task.TaskID).Count(&rows).Error)
	assert.EqualValues(t, 1, rows)
}

func TestTaskCompletionSettlesUserSubscriptionTokensIdempotently(t *testing.T) {
	truncate(t)
	const userID = 9302
	channelID := 8260
	seedUser(t, userID, 10000)
	seedChannel(t, channelID)
	sub := seedUserSubscriptionForUsage(t, userID, false)
	task := makeTask(userID, channelID, 60, 0, BillingSourceSubscription, sub.Id)
	task.TaskID = "task_user_sub_complete"

	settleTaskBillingOnComplete(context.Background(), fixedTaskBillingAdaptor{actualQuota: 90}, task, &relaycommon.TaskInfo{TotalTokens: 321})
	settleTaskBillingOnComplete(context.Background(), fixedTaskBillingAdaptor{actualQuota: 90}, task, &relaycommon.TaskInfo{TotalTokens: 321})

	var got model.UserSubscription
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.EqualValues(t, 321, got.TokensUsed)
}

func TestSettleUserSubscriptionActualUsageIsIdempotentByRequestId(t *testing.T) {
	truncate(t)
	const userID = 9303
	seedUser(t, userID, 10000)
	sub := seedUserSubscriptionForUsage(t, userID, false)
	info := &relaycommon.RelayInfo{
		BillingSource:                  BillingSourceSubscription,
		SubscriptionId:                 sub.Id,
		SubscriptionIncludeCacheTokens: false,
		RequestId:                      "req-user-sub-idempotent",
	}

	require.NoError(t, SettleUserSubscriptionActualUsage(info, 100, 50, 40))
	require.NoError(t, SettleUserSubscriptionActualUsage(info, 100, 50, 40))

	var got model.UserSubscription
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.EqualValues(t, 150, got.TokensUsed)
	assert.Zero(t, got.AmountUsed)

	var rows int64
	require.NoError(t, model.DB.Model(&model.UserSubscriptionSettlement{}).Where("settlement_key = ?", info.RequestId).Count(&rows).Error)
	assert.EqualValues(t, 1, rows)
}

func TestSettleUserSubscriptionActualUsageWssCacheFollowsIncludeCacheTokens(t *testing.T) {
	for _, tc := range []struct {
		name         string
		includeCache bool
		wantTokens   int64
	}{
		{name: "exclude cache", includeCache: false, wantTokens: 150},
		{name: "include cache", includeCache: true, wantTokens: 190},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			const userID = 9304
			seedUser(t, userID, 10000)
			sub := seedUserSubscriptionForUsage(t, userID, tc.includeCache)
			usage := &dto.RealtimeUsage{
				InputTokens:       100,
				OutputTokens:      50,
				InputTokenDetails: dto.InputTokenDetails{CachedTokens: 40},
			}
			info := &relaycommon.RelayInfo{
				BillingSource:                  BillingSourceSubscription,
				SubscriptionId:                 sub.Id,
				SubscriptionIncludeCacheTokens: tc.includeCache,
				RequestId:                      "req-wss-cache-" + tc.name,
			}

			require.NoError(t, SettleUserSubscriptionActualUsage(info, usage.InputTokens, usage.OutputTokens, usage.InputTokenDetails.CachedTokens))

			var got model.UserSubscription
			require.NoError(t, model.DB.First(&got, sub.Id).Error)
			assert.Equal(t, tc.wantTokens, got.TokensUsed)
		})
	}
}
