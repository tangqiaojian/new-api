package model

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createWeeklyQuotaUser(t *testing.T, id int, limit int, used int, resetAt int64) {
	t.Helper()
	user := &User{
		Id:                 id,
		Username:           fmt.Sprintf("weekly-quota-%d", id),
		AffCode:            fmt.Sprintf("weekly-%d", id),
		Status:             common.UserStatusEnabled,
		WeeklyQuota:        limit,
		WeeklyQuotaUsed:    used,
		WeeklyQuotaResetAt: resetAt,
	}
	require.NoError(t, DB.Create(user).Error)
}

func weeklyQuotaUsed(t *testing.T, id int) (int, int64) {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("weekly_quota_used", "weekly_quota_reset_at").First(&user, id).Error)
	return user.WeeklyQuotaUsed, user.WeeklyQuotaResetAt
}

func TestAdjustWeeklyQuotaReservationReconcilesActualUsageExactlyOnce(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	resetAt := calcNextWeeklyResetTime(now)
	createWeeklyQuotaUser(t, 9101, 100, 10, resetAt)

	reserved, err := adjustWeeklyQuotaReservationAt(9101, WeeklyQuotaReservation{}, 60, true, now)
	require.NoError(t, err)
	assert.Equal(t, WeeklyQuotaReservation{Amount: 60, ResetAt: resetAt, Version: 0, Enabled: true}, reserved)
	used, _ := weeklyQuotaUsed(t, 9101)
	assert.Equal(t, 70, used)

	settled, err := adjustWeeklyQuotaReservationAt(9101, reserved, 40, false, now)
	require.NoError(t, err)
	used, _ = weeklyQuotaUsed(t, 9101)
	assert.Equal(t, 50, used)

	// Passing the returned reservation makes a repeated reconciliation a no-op.
	settled, err = adjustWeeklyQuotaReservationAt(9101, settled, 40, false, now)
	require.NoError(t, err)
	used, _ = weeklyQuotaUsed(t, 9101)
	assert.Equal(t, 50, used)

	_, err = adjustWeeklyQuotaReservationAt(9101, settled, 0, false, now)
	require.NoError(t, err)
	used, _ = weeklyQuotaUsed(t, 9101)
	assert.Equal(t, 10, used)
}

func TestAdjustWeeklyQuotaReservationRejectsOverLimitWithoutMutation(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	createWeeklyQuotaUser(t, 9102, 100, 25, calcNextWeeklyResetTime(now))

	_, err := adjustWeeklyQuotaReservationAt(9102, WeeklyQuotaReservation{}, 76, true, now)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrWeeklyQuotaExceeded))
	used, _ := weeklyQuotaUsed(t, 9102)
	assert.Equal(t, 25, used)
}

func TestAdjustWeeklyQuotaReservationSaturatesUnrepresentableSettledUsage(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	createWeeklyQuotaUser(t, 9104, common.MaxQuota, common.MaxQuota-10, calcNextWeeklyResetTime(now))

	settled, err := adjustWeeklyQuotaReservationAt(9104, WeeklyQuotaReservation{}, 11, false, now)
	require.NoError(t, err)
	used, _ := weeklyQuotaUsed(t, 9104)
	assert.Equal(t, common.MaxQuota, used)
	assert.Equal(t, 10, settled.Amount)

	// Replacing the returned reservation remains stable at the storage ceiling.
	settled, err = adjustWeeklyQuotaReservationAt(9104, settled, 11, false, now)
	require.NoError(t, err)
	used, _ = weeklyQuotaUsed(t, 9104)
	assert.Equal(t, common.MaxQuota, used)
	assert.Equal(t, 10, settled.Amount)
}

func TestAdjustWeeklyQuotaReservationDoesNotSubtractAcrossResetCycles(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	oldResetAt := now.Add(-time.Minute).Unix()
	createWeeklyQuotaUser(t, 9103, 100, 90, oldResetAt)

	staleReservation := WeeklyQuotaReservation{Amount: 30, ResetAt: oldResetAt, Enabled: true}
	reserved, err := adjustWeeklyQuotaReservationAt(9103, staleReservation, 25, true, now)
	require.NoError(t, err)

	used, resetAt := weeklyQuotaUsed(t, 9103)
	assert.Equal(t, 25, used)
	assert.Equal(t, calcNextWeeklyResetTime(now), resetAt)
	assert.Equal(t, resetAt, reserved.ResetAt)
	assert.Equal(t, int64(1), reserved.Version)
}

func TestAdminResetInvalidatesInflightReservationsWithinSameCalendarCycle(t *testing.T) {
	truncateTables(t)
	now := time.Now()
	resetAt := calcNextWeeklyResetTime(now)
	createWeeklyQuotaUser(t, 9105, 100, 0, resetAt)

	requestA, err := adjustWeeklyQuotaReservationAt(9105, WeeklyQuotaReservation{}, 60, true, now)
	require.NoError(t, err)
	require.NoError(t, setUserWeeklyQuotaAt(9105, 100, now))

	requestB, err := adjustWeeklyQuotaReservationAt(9105, WeeklyQuotaReservation{}, 20, true, now)
	require.NoError(t, err)
	assert.NotEqual(t, requestA.Version, requestB.Version)
	assert.Equal(t, requestA.ResetAt, requestB.ResetAt)

	// A was reserved before the administrator reset. Its settlement must add
	// its actual usage without subtracting B's post-reset reservation.
	requestA, err = adjustWeeklyQuotaReservationAt(9105, requestA, 40, false, now)
	require.NoError(t, err)
	used, _ := weeklyQuotaUsed(t, 9105)
	assert.Equal(t, 60, used)

	_, err = adjustWeeklyQuotaReservationAt(9105, requestB, 10, false, now)
	require.NoError(t, err)
	used, _ = weeklyQuotaUsed(t, 9105)
	assert.Equal(t, 50, used)
	assert.Equal(t, requestB.Version, requestA.Version)
}

func TestResetDueWeeklyQuotasHonorsBatchLimit(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	for id := 9110; id < 9113; id++ {
		createWeeklyQuotaUser(t, id, 100, id-9100, now.Add(-time.Minute).Unix())
	}

	count, err := resetDueWeeklyQuotasAt(2, now)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	var due int64
	require.NoError(t, DB.Model(&User{}).
		Where("id BETWEEN ? AND ? AND weekly_quota_reset_at <= ?", 9110, 9112, now.Unix()).
		Count(&due).Error)
	assert.Equal(t, int64(1), due)

	count, err = resetDueWeeklyQuotasAt(2, now)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestResetDueWeeklyQuotasIncludesZeroResetAt(t *testing.T) {
	truncateTables(t)
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	createWeeklyQuotaUser(t, 9120, 100, 55, 0)

	count, err := resetDueWeeklyQuotasAt(10, now)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	used, resetAt := weeklyQuotaUsed(t, 9120)
	assert.Equal(t, 0, used)
	assert.Equal(t, calcNextWeeklyResetTime(now), resetAt)
}
