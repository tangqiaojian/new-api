package service

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type weeklyQuotaFundingStub struct {
	settleCalls int
	settleDelta int
}

func (*weeklyQuotaFundingStub) Source() string       { return BillingSourceWallet }
func (*weeklyQuotaFundingStub) PreConsume(int) error { return nil }
func (f *weeklyQuotaFundingStub) Settle(delta int) error {
	f.settleCalls++
	f.settleDelta += delta
	return nil
}
func (*weeklyQuotaFundingStub) Refund() error { return nil }

func createServiceWeeklyQuotaUser(t *testing.T, id int, walletQuota int, weeklyLimit int, weeklyUsed int, resetAt int64) {
	t.Helper()
	user := &model.User{
		Id:                 id,
		Username:           fmt.Sprintf("service-weekly-%d", id),
		AffCode:            fmt.Sprintf("service-weekly-%d", id),
		Status:             common.UserStatusEnabled,
		Quota:              walletQuota,
		WeeklyQuota:        weeklyLimit,
		WeeklyQuotaUsed:    weeklyUsed,
		WeeklyQuotaResetAt: resetAt,
	}
	require.NoError(t, model.DB.Create(user).Error)
}

func serviceWeeklyQuotaUsed(t *testing.T, id int) int {
	t.Helper()
	var used int
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", id).Pluck("weekly_quota_used", &used).Error)
	return used
}

func TestBillingSessionSettlesWeeklyQuotaWithActualUsageOnce(t *testing.T) {
	truncate(t)
	const userID = 9201
	createServiceWeeklyQuotaUser(t, userID, 1_000, 500, 0, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 50)
	require.NoError(t, err)

	funding := &weeklyQuotaFundingStub{}
	session := &BillingSession{
		relayInfo:        &relaycommon.RelayInfo{UserId: userID, IsPlayground: true},
		funding:          funding,
		preConsumedQuota: 50,
		weeklyQuota:      weeklyQuota,
	}

	require.NoError(t, session.Settle(80))
	require.NoError(t, session.Settle(80))
	assert.Equal(t, 1, funding.settleCalls)
	assert.Equal(t, 80, serviceWeeklyQuotaUsed(t, userID))
	snapshot := session.WeeklyQuotaReservation()
	require.NotNil(t, snapshot)
	assert.Equal(t, 80, snapshot.Amount)
}

func TestBillingSessionWeeklyCounterSaturationDoesNotBlockFundingSettlement(t *testing.T) {
	truncate(t)
	const userID = 9206
	createServiceWeeklyQuotaUser(t, userID, common.MaxQuota, common.MaxQuota, common.MaxQuota-10, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 5)
	require.NoError(t, err)

	funding := &weeklyQuotaFundingStub{}
	session := &BillingSession{
		relayInfo:        &relaycommon.RelayInfo{UserId: userID, IsPlayground: true},
		funding:          funding,
		preConsumedQuota: 5,
		weeklyQuota:      weeklyQuota,
	}

	require.NoError(t, session.Settle(20))
	assert.Equal(t, 1, funding.settleCalls)
	assert.Equal(t, 15, funding.settleDelta)
	assert.Equal(t, common.MaxQuota, serviceWeeklyQuotaUsed(t, userID))
}

func TestBillingSessionRefundReleasesWeeklyQuotaReservationSynchronously(t *testing.T) {
	truncate(t)
	const userID = 9202
	createServiceWeeklyQuotaUser(t, userID, 1_000, 500, 0, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 50)
	require.NoError(t, err)

	session := &BillingSession{
		relayInfo:   &relaycommon.RelayInfo{UserId: userID, IsPlayground: true},
		funding:     &weeklyQuotaFundingStub{},
		weeklyQuota: weeklyQuota,
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	session.Refund(ctx)
	assert.Equal(t, 0, serviceWeeklyQuotaUsed(t, userID))
}

func TestBillingSessionReserveEnforcesWeeklyQuotaBeforeFundingTopUp(t *testing.T) {
	truncate(t)
	const userID = 9203
	createServiceWeeklyQuotaUser(t, userID, 1_000, 75, 0, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 50)
	require.NoError(t, err)

	funding := &weeklyQuotaFundingStub{}
	session := &BillingSession{
		relayInfo:        &relaycommon.RelayInfo{UserId: userID, IsPlayground: true},
		funding:          funding,
		preConsumedQuota: 50,
		weeklyQuota:      weeklyQuota,
	}

	err = session.Reserve(80)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrWeeklyQuotaExceeded))
	assert.Equal(t, 0, funding.settleCalls)
	assert.Equal(t, 50, serviceWeeklyQuotaUsed(t, userID))
}

func TestNewBillingSessionReleasesWeeklyQuotaWhenFundingFails(t *testing.T) {
	truncate(t)
	const userID = 9204
	createServiceWeeklyQuotaUser(t, userID, 0, 100, 0, model.CalcNextWeeklyResetTime())
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:       userID,
		IsPlayground: true,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}

	session, apiErr := NewBillingSession(ctx, info, 50)
	assert.Nil(t, session)
	require.NotNil(t, apiErr)
	assert.Equal(t, 0, serviceWeeklyQuotaUsed(t, userID))
}

func TestRefundTaskQuotaReleasesPersistedWeeklyReservation(t *testing.T) {
	truncate(t)
	const userID = 9207
	createServiceWeeklyQuotaUser(t, userID, 1_000, 500, 0, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 50)
	require.NoError(t, err)

	task := makeTask(userID, 0, 50, 0, BillingSourceWallet, 0)
	task.PrivateData.WeeklyQuotaReservation = &weeklyQuota
	require.NoError(t, model.DB.Create(task).Error)

	assert.True(t, RefundTaskQuota(context.Background(), task, "upstream failed"))
	assert.Equal(t, 0, serviceWeeklyQuotaUsed(t, userID))

	var persisted model.Task
	require.NoError(t, model.DB.First(&persisted, task.ID).Error)
	assert.Zero(t, persisted.Quota)
	require.NotNil(t, persisted.PrivateData.WeeklyQuotaReservation)
	assert.Zero(t, persisted.PrivateData.WeeklyQuotaReservation.Amount)
}

func TestRecalculateTaskQuotaReconcilesPersistedWeeklyReservation(t *testing.T) {
	truncate(t)
	const userID = 9208
	createServiceWeeklyQuotaUser(t, userID, 1_000, 500, 0, model.CalcNextWeeklyResetTime())
	weeklyQuota, err := model.ReserveWeeklyQuota(userID, 50)
	require.NoError(t, err)

	task := makeTask(userID, 0, 50, 0, BillingSourceWallet, 0)
	task.PrivateData.WeeklyQuotaReservation = &weeklyQuota
	require.NoError(t, model.DB.Create(task).Error)

	RecalculateTaskQuota(context.Background(), task, 80, "actual task usage")
	assert.Equal(t, 80, serviceWeeklyQuotaUsed(t, userID))

	var persisted model.Task
	require.NoError(t, model.DB.First(&persisted, task.ID).Error)
	assert.Equal(t, 80, persisted.Quota)
	require.NotNil(t, persisted.PrivateData.WeeklyQuotaReservation)
	assert.Equal(t, 80, persisted.PrivateData.WeeklyQuotaReservation.Amount)
}

func TestWeeklyQuotaResetHandlerRunsThroughSystemTaskLease(t *testing.T) {
	truncate(t)
	const userID = 9205
	createServiceWeeklyQuotaUser(t, userID, 1_000, 100, 90, time.Now().Add(-time.Minute).Unix())

	task, err := model.CreateSystemTask(model.SystemTaskTypeWeeklyQuotaReset, nil, nil)
	require.NoError(t, err)
	claimed, ok, err := model.ClaimSystemTask(task.ID, task.Type, "weekly-runner", common.GetTimestamp()+60)
	require.NoError(t, err)
	require.True(t, ok)

	weeklyQuotaResetHandler{}.Run(context.Background(), claimed, "weekly-runner")
	reloaded, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, reloaded)
	assert.Equal(t, model.SystemTaskStatusSucceeded, reloaded.Status)
	assert.Equal(t, 0, serviceWeeklyQuotaUsed(t, userID))

	result := WeeklyQuotaResetResult{}
	require.NoError(t, common.UnmarshalJsonStr(reloaded.Result, &result))
	assert.Equal(t, 1, result.ResetCount)
}
