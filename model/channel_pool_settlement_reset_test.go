package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReserveUserSubscriptionAmountCapsAtRemaining(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	limited := &UserSubscription{
		UserId: 1, PlanId: 1, AmountTotal: 1000, AmountUsed: 800,
		Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}
	require.NoError(t, DB.Create(limited).Error)
	unlimited := &UserSubscription{
		UserId: 1, PlanId: 1, AmountTotal: 0, AmountUsed: 50,
		Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}
	require.NoError(t, DB.Create(unlimited).Error)

	applied, err := ReserveUserSubscriptionAmount(limited.Id, 400)
	require.NoError(t, err)
	assert.EqualValues(t, 200, applied, "only the remaining room may be applied")
	assert.EqualValues(t, 1000, getDeltaSubscription(t, limited.Id).AmountUsed)

	applied, err = ReserveUserSubscriptionAmount(limited.Id, 50)
	require.NoError(t, err)
	assert.Zero(t, applied, "an exhausted subscription has no room")
	assert.EqualValues(t, 1000, getDeltaSubscription(t, limited.Id).AmountUsed)

	applied, err = ReserveUserSubscriptionAmount(unlimited.Id, 400)
	require.NoError(t, err)
	assert.EqualValues(t, 400, applied, "unlimited subscriptions apply the full delta")
	assert.EqualValues(t, 450, getDeltaSubscription(t, unlimited.Id).AmountUsed)

	_, err = ReserveUserSubscriptionAmount(987653, 10)
	require.Error(t, err, "missing subscription must not be a silent success")
	_, err = ReserveUserSubscriptionAmount(limited.Id, 0)
	require.Error(t, err, "non-positive reserve delta is invalid")
}

// Deleting old-cycle settlement rows on reset prevents a late settle-to-zero
// on an old key from eating fresh usage.
func TestResetChannelSubscriptionPoolDeletesSettlements(t *testing.T) {
	setupChannelPoolTest(t)
	now := time.Now().Unix()
	pool := &ChannelSubscriptionPool{
		Name: "reset-settlements", PlanId: 1, AmountTotal: 10000, TokensTotal: 10000,
		StartTime: now - 60, EndTime: now + 3600, Status: ChannelPoolStatusActive,
	}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 9501}).Error)

	_, err := SettleChannelPoolUsage(9501, "keyA", 100, 10)
	require.NoError(t, err)

	require.NoError(t, ResetChannelSubscriptionPool(pool.Id, SubscriptionResetScopeBoth))
	reloaded, err := GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.Zero(t, reloaded.AmountUsed)
	assert.Zero(t, reloaded.TokensUsed)
	var settlementCount int64
	require.NoError(t, DB.Model(&ChannelPoolSettlement{}).Where("pool_id = ?", pool.Id).Count(&settlementCount).Error)
	assert.Zero(t, settlementCount, "reset must drop old-cycle settlement rows")

	// A late settle-to-zero on the old key must be a no-op, not a subtraction.
	_, err = SettleChannelPoolUsage(9501, "keyA", 0, 0)
	require.NoError(t, err)
	_, err = SettleChannelPoolUsage(9501, "keyB", 50, 5)
	require.NoError(t, err)
	_, err = SettleChannelPoolUsage(9501, "keyA", 0, 0)
	require.NoError(t, err)
	reloaded, err = GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 50, reloaded.AmountUsed, "old key must not eat new-cycle usage")
	assert.EqualValues(t, 5, reloaded.TokensUsed)
}

// The automatic recovery path (due reset detected during settle) must drop
// settlement rows the same way the admin reset does.
func TestRecoverChannelPoolDueDeletesSettlements(t *testing.T) {
	setupChannelPoolTest(t)
	now := time.Now().Unix()
	pool := &ChannelSubscriptionPool{
		Name: "recover-settlements", PlanId: 1, AmountTotal: 10000, TokensTotal: 10000,
		StartTime: now - 3600, EndTime: now + 3600, Status: ChannelPoolStatusActive,
		QuotaResetPeriod: SubscriptionResetDaily, QuotaNextResetTime: now + 3600,
	}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 9502}).Error)

	_, err := SettleChannelPoolUsage(9502, "keyA", 100, 0)
	require.NoError(t, err)

	// Make the quota reset due; the next settle triggers recovery first.
	require.NoError(t, DB.Model(&ChannelSubscriptionPool{}).Where("id = ?", pool.Id).
		Update("quota_next_reset_time", now-10).Error)

	_, err = SettleChannelPoolUsage(9502, "keyB", 50, 0)
	require.NoError(t, err)
	reloaded, err := GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 50, reloaded.AmountUsed, "recovery must zero usage before the new settle")

	var count int64
	require.NoError(t, DB.Model(&ChannelPoolSettlement{}).Where("pool_id = ? AND settlement_key = ?", pool.Id, "keyA").Count(&count).Error)
	assert.Zero(t, count, "recovery must drop old-cycle settlement rows")

	_, err = SettleChannelPoolUsage(9502, "keyA", 0, 0)
	require.NoError(t, err)
	reloaded, err = GetChannelSubscriptionPool(pool.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 50, reloaded.AmountUsed)
}
