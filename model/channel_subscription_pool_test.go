package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelPoolTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&ChannelSubscriptionPool{}, &ChannelSubscriptionPoolChannel{}, &ChannelPoolSettlement{}))
	require.NoError(t, DB.Exec("DELETE FROM channel_pool_settlements").Error)
	require.NoError(t, DB.Exec("DELETE FROM channel_subscription_pool_channels").Error)
	require.NoError(t, DB.Exec("DELETE FROM channel_subscription_pools").Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM channel_pool_settlements")
		DB.Exec("DELETE FROM channel_subscription_pool_channels")
		DB.Exec("DELETE FROM channel_subscription_pools")
	})
}

func TestCreateChannelSubscriptionPoolSnapshotsPlanAndInheritsResetDefaults(t *testing.T) {
	setupChannelPoolTest(t)
	anchor := time.Date(2026, time.January, 31, 8, 0, 0, 0, time.UTC).Unix()
	plan := &SubscriptionPlan{
		Id: 9901, Title: "Channel plan", PlanType: SubscriptionPlanTypeChannel,
		DurationUnit: SubscriptionDurationDay, DurationValue: 2,
		TotalAmount: 500, TotalTokens: 5000, IncludeCacheTokens: true,
		QuotaResetPeriod: SubscriptionResetMonthly, QuotaResetAnchor: &anchor, QuotaResetTimezone: "UTC",
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&Channel{Id: 9902, Name: "A", Key: "key"}).Error)
	before := GetDBTimestamp()

	pool, err := CreateChannelSubscriptionPool(CreateChannelSubscriptionPoolRequest{Name: "Shared", PlanId: plan.Id, ChannelIds: []int{9902}})

	require.NoError(t, err)
	assert.Equal(t, "Shared", pool.Name)
	assert.GreaterOrEqual(t, pool.StartTime, before)
	assert.Equal(t, pool.StartTime+2*24*3600, pool.EndTime)
	assert.EqualValues(t, 500, pool.AmountTotal)
	assert.EqualValues(t, 5000, pool.TokensTotal)
	assert.True(t, pool.IncludeCacheTokens)
	require.NotNil(t, pool.QuotaResetAnchor)
	assert.Equal(t, anchor, *pool.QuotaResetAnchor)
	require.NotNil(t, pool.TokenResetAnchor)
	assert.Equal(t, anchor, *pool.TokenResetAnchor, "empty token period inherits complete quota schedule")
	require.NotNil(t, pool.TokenResetTimezone)
	assert.Equal(t, "UTC", *pool.TokenResetTimezone)
}

func TestPlanResetRecalculationResyncsOnlyInheritedPoolWithoutMutatingUsage(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	oldAnchor := now + 60
	newAnchor := now + 3600
	plan := &SubscriptionPlan{Id: 9910, Title: "Plan", PlanType: SubscriptionPlanTypeChannel, DurationUnit: SubscriptionDurationDay, DurationValue: 2, QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &newAnchor, QuotaResetTimezone: "UTC"}
	require.NoError(t, DB.Create(plan).Error)
	inherited := &ChannelSubscriptionPool{Name: "Inherited", PlanId: plan.Id, PlanTitle: plan.Title, AmountUsed: 77, TokensUsed: 88, StartTime: now - 10, EndTime: now + 7200, Status: ChannelPoolStatusActive, QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &oldAnchor}
	overridden := &ChannelSubscriptionPool{Name: "Override", PlanId: plan.Id, PlanTitle: plan.Title, AmountUsed: 11, TokensUsed: 22, StartTime: now - 10, EndTime: now + 7200, Status: ChannelPoolStatusActive, QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &oldAnchor, QuotaResetOverridden: true}
	require.NoError(t, DB.Create(inherited).Error)
	require.NoError(t, DB.Create(overridden).Error)

	require.NoError(t, RecalculatePlanResetTimes(plan.Id))

	gotInherited, err := GetChannelSubscriptionPool(inherited.Id)
	require.NoError(t, err)
	require.NotNil(t, gotInherited.QuotaResetAnchor)
	assert.Equal(t, newAnchor, *gotInherited.QuotaResetAnchor)
	assert.EqualValues(t, 77, gotInherited.AmountUsed)
	assert.EqualValues(t, 88, gotInherited.TokensUsed)
	gotOverride, err := GetChannelSubscriptionPool(overridden.Id)
	require.NoError(t, err)
	require.NotNil(t, gotOverride.QuotaResetAnchor)
	assert.Equal(t, oldAnchor, *gotOverride.QuotaResetAnchor)
	assert.EqualValues(t, 11, gotOverride.AmountUsed)
	assert.EqualValues(t, 22, gotOverride.TokensUsed)
}

func TestChannelSubscriptionPoolSharedUsageAndMembershipUniqueness(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{Name: "Shared", PlanId: 1, PlanTitle: "Channel", AmountTotal: 100, TokensTotal: 1000, StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&[]ChannelSubscriptionPoolChannel{{PoolId: pool.Id, ChannelId: 101}, {PoolId: pool.Id, ChannelId: 102}}).Error)

	result, err := SettleChannelPoolUsage(101, "request-1", 70, 700)
	require.NoError(t, err)
	assert.EqualValues(t, 70, result.AmountUsed)
	assert.EqualValues(t, 700, result.TokensUsed)

	result, err = SettleChannelPoolUsage(102, "request-2", 50, 500)
	require.NoError(t, err)
	assert.EqualValues(t, 120, result.AmountUsed, "tail overage must be preserved")
	assert.EqualValues(t, 1200, result.TokensUsed)

	other := &ChannelSubscriptionPool{Name: "Other", PlanId: 1, PlanTitle: "Other", StartTime: now, EndTime: now + 3600, Status: ChannelPoolStatusActive}
	require.NoError(t, DB.Create(other).Error)
	assert.Error(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: other.Id, ChannelId: 101}).Error)
}

func TestGetEligibleChannelSubscriptionPoolsReturnsSharedRecoveredPoolForBatch(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{
		Name: "Batch", PlanId: 1, PlanTitle: "Channel", AmountUsed: 99,
		StartTime: now - 7200, EndTime: now + 7200, Status: ChannelPoolStatusActive,
		QuotaResetPeriod: SubscriptionResetCustom, QuotaResetCustomSeconds: 3600,
		QuotaLastResetTime: now - 7200, QuotaNextResetTime: now - 10,
	}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&[]ChannelSubscriptionPoolChannel{{PoolId: pool.Id, ChannelId: 401}, {PoolId: pool.Id, ChannelId: 402}}).Error)

	eligible, err := GetEligibleChannelSubscriptionPools([]int{401, 402, 999})

	require.NoError(t, err)
	require.NotNil(t, eligible[401])
	require.NotNil(t, eligible[402])
	assert.Equal(t, pool.Id, eligible[401].Id)
	assert.Equal(t, pool.Id, eligible[402].Id)
	assert.Zero(t, eligible[401].AmountUsed)
	assert.Zero(t, eligible[402].AmountUsed)
	assert.Nil(t, eligible[999])
}

func TestSettleChannelPoolUsageUsesDesiredCumulativeTotals(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{Name: "Desired", PlanId: 1, PlanTitle: "Channel", AmountTotal: 10, TokensTotal: 10, StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 201}).Error)

	for _, desired := range []struct{ amount, tokens int64 }{{8, 9}, {8, 9}, {13, 15}, {4, 3}} {
		_, err := SettleChannelPoolUsage(201, "stable-key", desired.amount, desired.tokens)
		require.NoError(t, err)
	}
	got, err := GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 4, got.AmountUsed, "desired total decrease must refund without going negative")
	assert.EqualValues(t, 3, got.TokensUsed)
}

// Repeated settlement with the same key must never grow the settlements table:
// the (pool_id, settlement_key) unique index makes the whole flow idempotent,
// including zero-value calibrations from refund paths.
func TestSettleChannelPoolUsageKeepsSingleSettlementRow(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{Name: "Idempotent", PlanId: 1, PlanTitle: "Channel", AmountTotal: 100, TokensTotal: 1000, StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 202}).Error)

	for _, desired := range []struct{ amount, tokens int64 }{{30, 300}, {30, 300}, {0, 0}, {25, 250}, {25, 250}} {
		_, err := SettleChannelPoolUsage(202, "request-key", desired.amount, desired.tokens)
		require.NoError(t, err)
	}
	var rows int64
	require.NoError(t, DB.Model(&ChannelPoolSettlement{}).Where("pool_id = ? AND settlement_key = ?", pool.Id, "request-key").Count(&rows).Error)
	assert.EqualValues(t, 1, rows, "same settlement key must be upserted, never duplicated")
	var settlement ChannelPoolSettlement
	require.NoError(t, DB.Where("pool_id = ? AND settlement_key = ?", pool.Id, "request-key").First(&settlement).Error)
	// settlement stores the desired cumulative totals: zero calibrations from
	// refund paths are persisted, and later non-zero values overwrite them.
	assert.EqualValues(t, 25, settlement.AmountSettled)
	assert.EqualValues(t, 250, settlement.TokensSettled)
	got, err := GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 25, got.AmountUsed, "usage tracks the last desired totals, not the sum of deltas")
	assert.EqualValues(t, 250, got.TokensUsed)
}

func TestResetChannelSubscriptionPoolIsScopedAndRecoversDuePool(t *testing.T) {
	setupChannelPoolTest(t)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{
		Name:   "Reset",
		PlanId: 1, PlanTitle: "Channel", AmountTotal: 100, AmountUsed: 90, TokensTotal: 1000, TokensUsed: 900,
		StartTime: now - 7200, EndTime: now + 7200, Status: ChannelPoolStatusActive,
		QuotaResetPeriod: SubscriptionResetCustom, QuotaResetCustomSeconds: 3600, QuotaLastResetTime: now - 7200, QuotaNextResetTime: now - 10,
		TokenResetPeriod: SubscriptionResetCustom, TokenResetCustomSeconds: 3600, TokenLastResetTime: now - 100, TokenNextResetTime: now + 3500,
	}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 301}).Error)

	eligible, err := GetEligibleChannelSubscriptionPool(301)
	require.NoError(t, err)
	require.NotNil(t, eligible)
	assert.Zero(t, eligible.AmountUsed)
	assert.EqualValues(t, 900, eligible.TokensUsed)
	assert.Greater(t, eligible.QuotaNextResetTime, now)

	require.NoError(t, ResetChannelSubscriptionPool(pool.Id, SubscriptionResetScopeTokens))
	got, err := GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.Zero(t, got.AmountUsed)
	assert.Zero(t, got.TokensUsed)

	got.EndTime = time.Now().Unix() - 1
	require.NoError(t, DB.Model(got).Update("end_time", got.EndTime).Error)
	eligible, err = GetEligibleChannelSubscriptionPool(301)
	require.NoError(t, err)
	assert.Nil(t, eligible)
}
