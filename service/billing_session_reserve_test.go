package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A subscription reserve that hits the plan cap applies only the remaining
// room; the session must track the applied amount so Settle reverses exactly
// what was reserved instead of refunding the clamped-away difference.
func TestBillingSessionReserveSubscriptionPartialApplySettlesExactly(t *testing.T) {
	truncate(t)
	user := &model.User{Username: "reserve-partial-user", Password: "password"}
	require.NoError(t, model.DB.Create(user).Error)
	now := time.Now().Unix()
	sub := &model.UserSubscription{
		UserId: user.Id, PlanId: 1, AmountTotal: 1000, AmountUsed: 800,
		Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}
	require.NoError(t, model.DB.Create(sub).Error)

	session := &BillingSession{
		relayInfo: &relaycommon.RelayInfo{
			UserId: user.Id, RequestId: "reserve-partial-req", IsPlayground: true,
		},
		funding:          &SubscriptionFunding{requestId: "reserve-partial-req", userId: user.Id, subscriptionId: sub.Id, preConsumed: 800},
		preConsumedQuota: 800,
		tokenConsumed:    800,
	}

	require.NoError(t, session.Reserve(1200))
	assert.Equal(t, 1000, session.GetPreConsumedQuota(),
		"session must track the applied 200, not the requested 400")

	require.NoError(t, session.Settle(900))
	var got model.UserSubscription
	require.NoError(t, model.DB.First(&got, sub.Id).Error)
	assert.EqualValues(t, 900, got.AmountUsed,
		"settle must land on the actual usage, not refund the capped difference")
}

// A negative actual quota is an upstream billing defect; Settle must clamp it
// to zero instead of producing a credit larger than the pre-consume.
func TestBillingSessionSettleClampsNegativeActual(t *testing.T) {
	truncate(t)
	user := &model.User{Username: "settle-negative-user", Password: "password", Quota: 10_000}
	require.NoError(t, model.DB.Create(user).Error)
	require.NoError(t, model.DecreaseUserQuota(user.Id, 800, false))

	session := &BillingSession{
		relayInfo: &relaycommon.RelayInfo{
			UserId: user.Id, RequestId: "settle-negative-req", IsPlayground: true,
		},
		funding:          &WalletFunding{userId: user.Id, consumed: 800},
		preConsumedQuota: 800,
		tokenConsumed:    800,
	}

	require.NoError(t, session.Settle(-50))
	quota, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 10_000, quota,
		"negative actual must clamp to zero: exactly the pre-consumed 800 is refunded")
}
