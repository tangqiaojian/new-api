package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPreConsumePlan(t *testing.T, id int, period string) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:               id,
		Title:            "pre-consume-plan",
		PriceAmount:      1,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1000,
		QuotaResetPeriod: period,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

// A repeated pre-consume with the same requestId must return the stored
// reservation instead of charging again.
func TestPreConsumeUserSubscriptionIdempotentHitReturnsStoredSub(t *testing.T) {
	truncateTables(t)
	plan := seedPreConsumePlan(t, 9301, "")
	user := User{Username: "preconsume-idem-user", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	now := time.Now().Unix()
	sub := &UserSubscription{
		UserId: user.Id, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 0,
		Status: "active", StartTime: now - 60, EndTime: now + 30*24*3600,
	}
	require.NoError(t, DB.Create(sub).Error)

	first, err := PreConsumeUserSubscription("req-idem-1", user.Id, "model", 0, 100, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, sub.Id, first.UserSubscriptionId)
	assert.EqualValues(t, 100, first.PreConsumed)

	second, err := PreConsumeUserSubscription("req-idem-1", user.Id, "model", 0, 100, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, sub.Id, second.UserSubscriptionId)
	assert.EqualValues(t, 100, second.PreConsumed)

	got := getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 100, got.AmountUsed, "idempotent retry must not double-charge")
}

// Refund restores usage exactly once; a repeated refund is a no-op.
func TestRefundSubscriptionPreConsumeRestoresUsageOnce(t *testing.T) {
	truncateTables(t)
	plan := seedPreConsumePlan(t, 9302, "")
	user := User{Username: "refund-once-user", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	now := time.Now().Unix()
	sub := &UserSubscription{
		UserId: user.Id, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 900,
		Status: "active", StartTime: now - 60, EndTime: now + 30*24*3600,
	}
	require.NoError(t, DB.Create(sub).Error)

	_, err := PreConsumeUserSubscription("req-refund-1", user.Id, "model", 0, 100, 0, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1000, getDeltaSubscription(t, sub.Id).AmountUsed)

	require.NoError(t, RefundSubscriptionPreConsume("req-refund-1"))
	assert.EqualValues(t, 900, getDeltaSubscription(t, sub.Id).AmountUsed)

	require.NoError(t, RefundSubscriptionPreConsume("req-refund-1"))
	assert.EqualValues(t, 900, getDeltaSubscription(t, sub.Id).AmountUsed, "second refund must be a no-op")

	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "req-refund-1").First(&record).Error)
	assert.Equal(t, "refunded", record.Status)

	_, err = PreConsumeUserSubscription("req-refund-1", user.Id, "model", 0, 100, 0, 0)
	require.Error(t, err, "refunded requestId must not pre-consume again")
}

// A cycle reset voids outstanding pre-consume records: a later refund must not
// eat quota reserved in the new cycle.
func TestResetVoidsPreConsumeRecordAndRefundSkipsNewCycle(t *testing.T) {
	truncateTables(t)
	plan := seedPreConsumePlan(t, 9303, SubscriptionResetDaily)
	user := User{Username: "reset-void-user", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	now := time.Now().Unix()
	sub := &UserSubscription{
		UserId: user.Id, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 0,
		Status: "active", StartTime: now - 48*3600, EndTime: now + 30*24*3600,
		LastResetTime: now - 24*3600, NextResetTime: now + 3600,
	}
	require.NoError(t, DB.Create(sub).Error)

	_, err := PreConsumeUserSubscription("req-old-cycle", user.Id, "model", 0, 100, 0, 0)
	require.NoError(t, err)
	require.EqualValues(t, 100, getDeltaSubscription(t, sub.Id).AmountUsed)

	// Make the quota reset due and run the scheduler.
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).
		Update("next_reset_time", now-10).Error)
	reset, err := ResetDueSubscriptions(10)
	require.NoError(t, err)
	require.Equal(t, 1, reset)
	assert.Zero(t, getDeltaSubscription(t, sub.Id).AmountUsed, "reset must wipe usage")

	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "req-old-cycle").First(&record).Error)
	assert.Equal(t, "reset", record.Status, "reset must void the outstanding pre-consume record")

	// New cycle: another request reserves quota.
	_, err = PreConsumeUserSubscription("req-new-cycle", user.Id, "model", 0, 50, 0, 0)
	require.NoError(t, err)
	require.EqualValues(t, 50, getDeltaSubscription(t, sub.Id).AmountUsed)

	// The old request finally fails and refunds: the new cycle must be untouched.
	require.NoError(t, RefundSubscriptionPreConsume("req-old-cycle"))
	assert.EqualValues(t, 50, getDeltaSubscription(t, sub.Id).AmountUsed,
		"refunding a reset-voided record must not subtract from the new cycle")
}

// Refunding a record whose subscription row was deleted closes the record
// instead of failing forever.
func TestRefundSubscriptionPreConsumeToleratesMissingSubscription(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId: "req-orphan-sub", UserId: 1, UserSubscriptionId: 987650,
		PreConsumed: 10, Status: "consumed",
	}).Error)

	require.NoError(t, RefundSubscriptionPreConsume("req-orphan-sub"))
	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "req-orphan-sub").First(&record).Error)
	assert.Equal(t, "refunded", record.Status)
}

func TestAdminAdjustUserSubscriptionRejectsUnlimitedAndMissing(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	unlimited := &UserSubscription{
		UserId: 1, PlanId: 1, AmountTotal: 0, TokensTotal: 0,
		Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}
	require.NoError(t, DB.Create(unlimited).Error)
	limited := &UserSubscription{
		UserId: 1, PlanId: 1, AmountTotal: 100, TokensTotal: 200,
		Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}
	require.NoError(t, DB.Create(limited).Error)

	require.Error(t, AdminAdjustUserSubscription(unlimited.Id, 50, 0),
		"adding quota to an unlimited subscription must fail instead of capping it")
	require.Error(t, AdminAdjustUserSubscription(unlimited.Id, 0, 50))
	require.Error(t, AdminAdjustUserSubscription(987651, 50, 0), "missing id must not be a silent no-op")

	require.NoError(t, AdminAdjustUserSubscription(limited.Id, 50, 60))
	got := getDeltaSubscription(t, limited.Id)
	assert.EqualValues(t, 150, got.AmountTotal)
	assert.EqualValues(t, 50, got.AdminAmountExtra)
	assert.EqualValues(t, 260, got.TokensTotal)
	assert.EqualValues(t, 60, got.AdminTokensExtra)
}

// Expiry must only apply group policy from subscriptions expired in this run;
// a historical expired row must not override a later admin group change.
func TestExpireDueSubscriptionsIgnoresHistoricalGroupRows(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	user := User{Username: "expire-history-user", Password: "password", Group: "special"}
	require.NoError(t, DB.Create(&user).Error)
	// Historical row: expired last month with a downgrade policy.
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: user.Id, PlanId: 1, Status: "expired",
		StartTime: now - 60*24*3600, EndTime: now - 30*24*3600,
		UpgradeGroup: "pro", PrevUserGroup: "free", DowngradeGroup: "free",
	}).Error)
	// Due row without any group policy.
	due := &UserSubscription{
		UserId: user.Id, PlanId: 1, Status: "active",
		StartTime: now - 10*24*3600, EndTime: now - 10,
	}
	require.NoError(t, DB.Create(due).Error)

	expired, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)
	assert.Equal(t, "expired", getDeltaSubscription(t, due.Id).Status)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, "special", reloaded.Group,
		"historical expired rows must not drive group transitions")
}

// Expiry reverts the group to the pre-purchase snapshot when the just-expired
// subscription was the one that elevated the user.
func TestExpireDueSubscriptionsRevertsGroupFromJustExpiredSub(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	user := User{Username: "expire-revert-user", Password: "password", Group: "pro"}
	require.NoError(t, DB.Create(&user).Error)
	due := &UserSubscription{
		UserId: user.Id, PlanId: 1, Status: "active",
		StartTime: now - 10*24*3600, EndTime: now - 10,
		UpgradeGroup: "pro", PrevUserGroup: "free",
	}
	require.NoError(t, DB.Create(due).Error)

	expired, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, "free", reloaded.Group)
}

// Expiring an orphan subscription (user already hard-deleted) flips the status
// without failing on the missing user row.
func TestExpireDueSubscriptionsSkipsGroupForOrphanUser(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	orphan := &UserSubscription{
		UserId: 987652, PlanId: 1, Status: "active",
		StartTime: now - 10*24*3600, EndTime: now - 10,
		UpgradeGroup: "pro", PrevUserGroup: "free",
	}
	require.NoError(t, DB.Create(orphan).Error)

	expired, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expired)
	assert.Equal(t, "expired", getDeltaSubscription(t, orphan.Id).Status)
}
