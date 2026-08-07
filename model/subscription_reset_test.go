package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubscriptionResetPlan(t *testing.T, plan *SubscriptionPlan) {
	t.Helper()
	require.NoError(t, DB.Create(plan).Error)
}

func seedSubscriptionResetSub(t *testing.T, sub *UserSubscription) {
	t.Helper()
	require.NoError(t, DB.Create(sub).Error)
}

func getSubscriptionResetSub(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

func TestAdminResetUserSubscriptionsByPlanResetsAllActiveMatchesAndAdvancesTime(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9101,
		Title:            "Pro",
		PriceAmount:      10,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1000,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	otherPlan := &SubscriptionPlan{
		Id:               9102,
		Title:            "Basic",
		PriceAmount:      1,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      100,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	seedSubscriptionResetPlan(t, plan)
	seedSubscriptionResetPlan(t, otherPlan)

	activeEnd := now + 30*24*3600
	expiredEnd := now - 1
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9201, UserId: 101, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 300, StartTime: now - 3600, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 120})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9202, UserId: 101, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 500, StartTime: now - 3600, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 120})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9203, UserId: 101, PlanId: otherPlan.Id, AmountTotal: 100, AmountUsed: 60, StartTime: now - 3600, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 120})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9204, UserId: 101, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 700, StartTime: now - 7200, EndTime: expiredEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now - 10})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9205, UserId: 102, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 800, StartTime: now - 3600, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 120})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9206, UserId: 101, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 900, StartTime: now - 3600, EndTime: activeEnd, Status: "cancelled", LastResetTime: now - 3600, NextResetTime: now + 120})

	beforeReset := GetDBTimestamp()
	result, err := AdminResetUserSubscriptionsByPlan(101, plan.Id, true)
	afterReset := GetDBTimestamp()

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.Equal(t, 2, result.MatchedCount)
	assert.Equal(t, 2, result.ResetCount)
	assert.Equal(t, 1, result.UserCount)
	assert.Equal(t, []int{101}, result.AffectedUserIds)
	assert.True(t, result.AdvanceResetTime)

	for _, id := range []int{9201, 9202} {
		sub := getSubscriptionResetSub(t, id)
		assert.Zero(t, sub.AmountUsed)
		assert.GreaterOrEqual(t, sub.LastResetTime, beforeReset)
		assert.LessOrEqual(t, sub.LastResetTime, afterReset)
		assert.Equal(t, calcNextResetTime(time.Unix(sub.LastResetTime, 0), plan, sub.EndTime), sub.NextResetTime)
	}
	assert.EqualValues(t, 60, getSubscriptionResetSub(t, 9203).AmountUsed)
	assert.EqualValues(t, 700, getSubscriptionResetSub(t, 9204).AmountUsed)
	assert.EqualValues(t, 800, getSubscriptionResetSub(t, 9205).AmountUsed)
	assert.EqualValues(t, 900, getSubscriptionResetSub(t, 9206).AmountUsed)
}

func TestAdminResetUserSubscriptionsByPlanKeepsResetTimes(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9301,
		Title:            "Team",
		PriceAmount:      20,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      2000,
		QuotaResetPeriod: SubscriptionResetMonthly,
	}
	seedSubscriptionResetPlan(t, plan)

	lastReset := now - 86400
	nextReset := now + 86400
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9302, UserId: 201, PlanId: plan.Id, AmountTotal: 2000, AmountUsed: 1200, StartTime: now - 172800, EndTime: now + 30*24*3600, Status: "active", LastResetTime: lastReset, NextResetTime: nextReset})

	result, err := AdminResetUserSubscriptionsByPlan(201, plan.Id, false)

	require.NoError(t, err)
	assert.False(t, result.AdvanceResetTime)
	sub := getSubscriptionResetSub(t, 9302)
	assert.Zero(t, sub.AmountUsed)
	assert.Equal(t, lastReset, sub.LastResetTime)
	assert.Equal(t, nextReset, sub.NextResetTime)
}

func TestAdminResetUserSubscriptionsByPlanNoActiveMatchReturnsError(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:            9401,
		Title:         "Expired",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   1000,
	}
	seedSubscriptionResetPlan(t, plan)
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9402, UserId: 301, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 500, StartTime: now - 7200, EndTime: now - 1, Status: "active"})

	result, err := AdminResetUserSubscriptionsByPlan(301, plan.Id, true)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, strings.Contains(err.Error(), "该用户没有有效的此套餐订阅"))
}

func TestAdminResetPlanSubscriptionsResetsAllActiveUsers(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9501,
		Title:            "Business",
		PriceAmount:      30,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      3000,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	seedSubscriptionResetPlan(t, plan)

	activeEnd := now + 30*24*3600
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9502, UserId: 401, PlanId: plan.Id, AmountTotal: 3000, AmountUsed: 1000, StartTime: now - 3600, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 10})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9503, UserId: 401, PlanId: plan.Id, AmountTotal: 3000, AmountUsed: 1100, StartTime: now - 3500, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 10})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9504, UserId: 402, PlanId: plan.Id, AmountTotal: 3000, AmountUsed: 1200, StartTime: now - 3400, EndTime: activeEnd, Status: "active", LastResetTime: now - 3600, NextResetTime: now + 10})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9505, UserId: 403, PlanId: plan.Id, AmountTotal: 3000, AmountUsed: 1300, StartTime: now - 7200, EndTime: now - 1, Status: "active", LastResetTime: now - 3600, NextResetTime: now - 10})
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9506, UserId: 404, PlanId: plan.Id, AmountTotal: 3000, AmountUsed: 1400, StartTime: now - 3600, EndTime: activeEnd, Status: "cancelled", LastResetTime: now - 3600, NextResetTime: now + 10})

	result, err := AdminResetPlanSubscriptions(plan.Id, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.MatchedCount)
	assert.Equal(t, 3, result.ResetCount)
	assert.Equal(t, 2, result.UserCount)
	assert.Equal(t, []int{401, 402}, result.AffectedUserIds)
	for _, id := range []int{9502, 9503, 9504} {
		sub := getSubscriptionResetSub(t, id)
		assert.Zero(t, sub.AmountUsed)
		assert.Zero(t, sub.LastResetTime)
		assert.Zero(t, sub.NextResetTime)
	}
	assert.EqualValues(t, 1300, getSubscriptionResetSub(t, 9505).AmountUsed)
	assert.EqualValues(t, 1400, getSubscriptionResetSub(t, 9506).AmountUsed)
}

func TestAnchoredSubscriptionResetScheduleUsesTimezoneDSTMonthEndAndStrictExpiry(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	dailyAnchor := time.Date(2026, time.March, 7, 9, 30, 0, 0, newYork).Unix()
	monthlyAnchor := time.Date(2026, time.January, 31, 8, 15, 0, 0, newYork).Unix()
	tests := []struct {
		name    string
		base    time.Time
		plan    SubscriptionPlan
		endUnix int64
		want    int64
	}{
		{
			name: "daily keeps wall clock across DST",
			base: time.Date(2026, time.March, 7, 10, 0, 0, 0, newYork),
			plan: SubscriptionPlan{QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &dailyAnchor, QuotaResetTimezone: "America/New_York"},
			want: time.Date(2026, time.March, 8, 9, 30, 0, 0, newYork).Unix(),
		},
		{
			name: "monthly clamps anchor to month end",
			base: time.Date(2026, time.February, 1, 0, 0, 0, 0, newYork),
			plan: SubscriptionPlan{QuotaResetPeriod: SubscriptionResetMonthly, QuotaResetAnchor: &monthlyAnchor, QuotaResetTimezone: "America/New_York"},
			want: time.Date(2026, time.February, 28, 8, 15, 0, 0, newYork).Unix(),
		},
		{
			name:    "reset at exact expiry is suppressed",
			base:    time.Date(2026, time.March, 7, 10, 0, 0, 0, newYork),
			plan:    SubscriptionPlan{QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &dailyAnchor, QuotaResetTimezone: "America/New_York"},
			endUnix: time.Date(2026, time.March, 8, 9, 30, 0, 0, newYork).Unix(),
			want:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calcNextResetTime(tt.base, &tt.plan, tt.endUnix))
		})
	}
}

func TestAnchoredWallTimeDSTPolicyAndCustomBounds(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	gap := anchoredWallTime(2026, time.March, 8, 2, 30, 0, location)
	assert.Equal(t, time.Date(2026, time.March, 8, 3, 0, 0, 0, location), gap)

	fold := anchoredWallTime(2026, time.November, 1, 1, 30, 0, location)
	_, offset := fold.Zone()
	assert.Equal(t, -4*3600, offset, "fold policy uses the first occurrence")

	minAnchor := minSupportedResetUnix
	maxAnchor := maxSupportedResetUnix
	assert.Zero(t, calcNextResetSchedule(time.Unix(maxSupportedResetUnix, 0), SubscriptionResetCustom, 1, &minAnchor, "UTC", 0))
	assert.Zero(t, calcNextResetSchedule(time.Unix(maxSupportedResetUnix, 0), SubscriptionResetCustom, 1, &maxAnchor, "UTC", 0))
}

func TestTokenResetScheduleUsesExplicitAnchorAndCustomAnchorArithmetic(t *testing.T) {
	quotaAnchor := int64(1_700_000_000)
	tokenAnchor := quotaAnchor + 1800
	plan := &SubscriptionPlan{
		QuotaResetPeriod:        SubscriptionResetCustom,
		QuotaResetCustomSeconds: 3600,
		QuotaResetAnchor:        &quotaAnchor,
		TokenResetPeriod:        SubscriptionResetCustom,
		TokenResetCustomSeconds: 7200,
		TokenResetAnchor:        &tokenAnchor,
	}

	assert.Equal(t, quotaAnchor+3600, calcNextResetTime(time.Unix(quotaAnchor, 0), plan, 0))
	assert.Equal(t, tokenAnchor, calcNextTokenResetTime(time.Unix(quotaAnchor, 0), plan, 0))
	assert.Equal(t, quotaAnchor+4*3600, calcNextResetTime(time.Unix(quotaAnchor+3*3600+1, 0), plan, 0))
	assert.Equal(t, tokenAnchor+2*7200, calcNextTokenResetTime(time.Unix(tokenAnchor+7200+1, 0), plan, 0))
}

func TestTokenResetScheduleCompletelyInheritsQuotaSchedule(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	anchor := time.Date(2026, time.January, 31, 6, 0, 0, 0, location).Unix()
	plan := &SubscriptionPlan{
		QuotaResetPeriod:   SubscriptionResetMonthly,
		QuotaResetAnchor:   &anchor,
		QuotaResetTimezone: "Asia/Shanghai",
		TokenResetPeriod:   "",
	}
	base := time.Date(2026, time.February, 1, 0, 0, 0, 0, location)
	assert.Equal(t, calcNextResetTime(base, plan, 0), calcNextTokenResetTime(base, plan, 0))
}

func TestAdminResetSingleSubscriptionHonorsScopeAndKeepsEndTime(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9701, Title: "Scoped", PriceAmount: 1, DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 100, TotalTokens: 200, QuotaResetPeriod: SubscriptionResetDaily}
	seedSubscriptionResetPlan(t, plan)
	endTime := now + 10*24*3600
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9702, UserId: 501, PlanId: plan.Id, AmountTotal: 100, AmountUsed: 40, TokensTotal: 200, TokensUsed: 80, StartTime: now - 3600, EndTime: endTime, Status: "active"})

	require.NoError(t, AdminResetSingleUserSubscription(9702, SubscriptionResetScopeTokens))
	sub := getSubscriptionResetSub(t, 9702)
	assert.EqualValues(t, 40, sub.AmountUsed)
	assert.Zero(t, sub.TokensUsed)
	assert.Zero(t, sub.LastResetTime)
	assert.Greater(t, sub.TokenLastResetTime, int64(0))
	assert.Equal(t, endTime, sub.EndTime)
}

func TestRecalculatePlanResetTimesUpdatesInheritingActiveSchedules(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	anchor := now - 3*24*3600
	plan := &SubscriptionPlan{Id: 9801, Title: "Inherited", PriceAmount: 1, DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 100, TotalTokens: 200, QuotaResetPeriod: SubscriptionResetDaily, QuotaResetAnchor: &anchor, QuotaResetTimezone: "UTC"}
	seedSubscriptionResetPlan(t, plan)
	seedSubscriptionResetSub(t, &UserSubscription{Id: 9802, UserId: 601, PlanId: plan.Id, AmountTotal: 100, TokensTotal: 200, StartTime: now - 3600, EndTime: now + 10*24*3600, Status: "active"})

	require.NoError(t, RecalculatePlanResetTimes(plan.Id))
	sub := getSubscriptionResetSub(t, 9802)
	assert.Greater(t, sub.NextResetTime, now)
	assert.Greater(t, sub.TokenNextResetTime, now)
	assert.Equal(t, sub.NextResetTime, sub.TokenNextResetTime)
}

func TestAdminResetPlanSubscriptionsNoMatchSucceeds(t *testing.T) {
	truncateTables(t)

	plan := &SubscriptionPlan{
		Id:            9601,
		Title:         "Empty",
		PriceAmount:   10,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   1000,
	}
	seedSubscriptionResetPlan(t, plan)

	result, err := AdminResetPlanSubscriptions(plan.Id, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Zero(t, result.MatchedCount)
	assert.Zero(t, result.ResetCount)
	assert.Zero(t, result.UserCount)
	assert.Empty(t, result.AffectedUserIds)
}

func TestUserSubscriptionResetOverrideAppliesAndClearsInheritance(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	planAnchor := now - 2*24*3600
	plan := &SubscriptionPlan{
		Id:                 9401,
		Title:              "Override",
		PriceAmount:        1,
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		TotalAmount:        100,
		TotalTokens:        200,
		QuotaResetPeriod:   SubscriptionResetDaily,
		QuotaResetAnchor:   &planAnchor,
		QuotaResetTimezone: "UTC",
		PlanType:           SubscriptionPlanTypeUser,
		Enabled:            true,
	}
	seedSubscriptionResetPlan(t, plan)
	require.NoError(t, DB.Exec("DELETE FROM users WHERE id = ?", 701).Error)
	require.NoError(t, DB.Create(&User{Id: 701, Username: "override-user", Password: "x", Role: 1, Status: 1}).Error)

	sub, err := CreateUserSubscriptionFromPlanTx(DB, 701, plan, "admin")
	require.NoError(t, err)
	require.Nil(t, sub.QuotaResetAnchor)
	assert.Greater(t, sub.NextResetTime, now)

	instanceAnchor := now - 3600
	tz := "Asia/Shanghai"
	applyUserSubscriptionResetOverride(sub, plan, &UserSubscriptionResetOverride{
		QuotaResetAnchor:   &instanceAnchor,
		QuotaResetTimezone: &tz,
	}, now)
	require.NoError(t, DB.Save(sub).Error)

	created := getSubscriptionResetSub(t, sub.Id)
	require.NotNil(t, created.QuotaResetAnchor)
	assert.Equal(t, instanceAnchor, *created.QuotaResetAnchor)
	require.NotNil(t, created.QuotaResetTimezone)
	assert.Equal(t, tz, *created.QuotaResetTimezone)
	assert.Nil(t, created.TokenResetAnchor)
	assert.Greater(t, created.NextResetTime, now)

	// Clear overrides: instance should fully inherit plan schedule again.
	applyUserSubscriptionResetOverride(&created, plan, &UserSubscriptionResetOverride{}, now)
	require.NoError(t, DB.Save(&created).Error)
	updated := getSubscriptionResetSub(t, sub.Id)
	assert.Nil(t, updated.QuotaResetAnchor)
	assert.Nil(t, updated.QuotaResetTimezone)
	assert.Greater(t, updated.NextResetTime, now)
	assert.Equal(t, updated.NextResetTime, updated.TokenNextResetTime)
}
