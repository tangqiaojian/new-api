package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedDeltaSubscription(t *testing.T, tokensTotal, tokensUsed, amountTotal, amountUsed int64) *UserSubscription {
	t.Helper()
	sub := &UserSubscription{
		UserId:      9401,
		AmountTotal: amountTotal,
		AmountUsed:  amountUsed,
		TokensTotal: tokensTotal,
		TokensUsed:  tokensUsed,
		Status:      "active",
		StartTime:   time.Now().Unix() - 60,
		EndTime:     time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, DB.Create(sub).Error)
	return sub
}

func getDeltaSubscription(t *testing.T, id int) UserSubscription {
	t.Helper()
	var got UserSubscription
	require.NoError(t, DB.First(&got, id).Error)
	return got
}

func TestPostConsumeUserSubscriptionDeltaClampsTokensInSQL(t *testing.T) {
	truncateTables(t)
	sub := seedDeltaSubscription(t, 100, 80, 1000, 10)

	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, 0, 50))
	got := getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 100, got.TokensUsed, "TokensUsed must not exceed TokensTotal")
	assert.EqualValues(t, 10, got.AmountUsed, "token-only delta must not rewrite amount")

	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, 0, -1000))
	got = getDeltaSubscription(t, sub.Id)
	assert.Zero(t, got.TokensUsed, "TokensUsed must not go negative")
	assert.EqualValues(t, 10, got.AmountUsed)
}

func TestPostConsumeUserSubscriptionDeltaClampsAmountInSQL(t *testing.T) {
	truncateTables(t)
	sub := seedDeltaSubscription(t, 1000, 5, 100, 90)

	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, 50, 0))
	got := getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 100, got.AmountUsed, "AmountUsed must not exceed AmountTotal")
	assert.EqualValues(t, 5, got.TokensUsed)

	require.NoError(t, PostConsumeUserSubscriptionDelta(sub.Id, -1000, 0))
	got = getDeltaSubscription(t, sub.Id)
	assert.Zero(t, got.AmountUsed, "AmountUsed must not go negative")
	assert.EqualValues(t, 5, got.TokensUsed)
}

func TestPostConsumeUserSubscriptionDeltaMissingRowErrors(t *testing.T) {
	truncateTables(t)
	err := PostConsumeUserSubscriptionDelta(9_999_001, 1, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSettleUserSubscriptionUsageIsIdempotentAndClamps(t *testing.T) {
	truncateTables(t)
	sub := seedDeltaSubscription(t, 100, 0, 1000, 0)

	_, err := SettleUserSubscriptionUsage(sub.Id, "request:same", 80)
	require.NoError(t, err)
	_, err = SettleUserSubscriptionUsage(sub.Id, "request:same", 80)
	require.NoError(t, err)
	got := getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 80, got.TokensUsed)

	_, err = SettleUserSubscriptionUsage(sub.Id, "request:overflow", 50)
	require.NoError(t, err)
	got = getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 100, got.TokensUsed, "second key must clamp against TokensTotal")

	_, err = SettleUserSubscriptionUsage(sub.Id, "request:overflow", 0)
	require.NoError(t, err)
	got = getDeltaSubscription(t, sub.Id)
	assert.EqualValues(t, 80, got.TokensUsed, "refund of a clamped key must only reverse what it applied")

	var rows int64
	require.NoError(t, DB.Model(&UserSubscriptionSettlement{}).Where("user_subscription_id = ?", sub.Id).Count(&rows).Error)
	assert.EqualValues(t, 2, rows)
}
