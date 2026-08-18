package model

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedAdminListSubscription(t *testing.T, userId int, status string, endTime int64) *UserSubscription {
	t.Helper()
	sub := &UserSubscription{
		UserId:    userId,
		PlanId:    1,
		Status:    status,
		StartTime: time.Now().Unix() - 3600,
		EndTime:   endTime,
	}
	require.NoError(t, DB.Create(sub).Error)
	return sub
}

func adminListById(details []AdminUserSubscriptionDetail) map[int]AdminUserSubscriptionDetail {
	byId := make(map[int]AdminUserSubscriptionDetail, len(details))
	for _, d := range details {
		byId[d.Id] = d
	}
	return byId
}

func TestAdminListAllUserSubscriptionsShowsOrphanWithEmptyUsername(t *testing.T) {
	truncateTables(t)
	user := User{Username: "admin-list-user", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	future := time.Now().Add(time.Hour).Unix()
	normal := seedAdminListSubscription(t, user.Id, "active", future)
	// 孤儿订阅：users 表中没有对应行（用户已被硬删除）。
	orphan := seedAdminListSubscription(t, 987654, "active", future)

	details, total, err := AdminListAllUserSubscriptions(1, 10, "", "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	byId := adminListById(details)
	require.Len(t, byId, 2)
	assert.Equal(t, "admin-list-user", byId[normal.Id].Username)
	assert.Equal(t, "", byId[orphan.Id].Username, "orphan subscription must keep empty username for the frontend placeholder")
	assert.Equal(t, 987654, byId[orphan.Id].UserId)

	// 按用户名搜不到孤儿（名字已随用户删除），纯数字过滤可兼匹配 user_id。
	_, total, err = AdminListAllUserSubscriptions(1, 10, "no-such-user", "")
	require.NoError(t, err)
	assert.Zero(t, total)
	details, total, err = AdminListAllUserSubscriptions(1, 10, strconv.Itoa(orphan.UserId), "")
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	assert.Equal(t, orphan.Id, details[0].Id)
}

func TestAdminListAllUserSubscriptionsEffectiveStatus(t *testing.T) {
	truncateTables(t)
	user := User{Username: "admin-list-status-user", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()
	activeValid := seedAdminListSubscription(t, user.Id, "active", future)
	activeDue := seedAdminListSubscription(t, user.Id, "active", past) // 有效状态已过期
	activeNoEnd := seedAdminListSubscription(t, user.Id, "active", 0)  // 无到期时间
	storedExpired := seedAdminListSubscription(t, user.Id, "expired", past)
	storedCancelled := seedAdminListSubscription(t, user.Id, "cancelled", past)

	details, total, err := AdminListAllUserSubscriptions(1, 10, "", "")
	require.NoError(t, err)
	assert.EqualValues(t, 5, total)
	byId := adminListById(details)
	assert.Equal(t, "active", byId[activeValid.Id].Status)
	assert.Equal(t, "active", byId[activeNoEnd.Id].Status, "end_time=0 means no expiry")
	assert.Equal(t, "expired", byId[activeDue.Id].Status, "due active row must be presented as expired")
	assert.Equal(t, "expired", byId[storedExpired.Id].Status)
	assert.Equal(t, "cancelled", byId[storedCancelled.Id].Status, "cancelled must win over expiry")

	// 库列不被改写（slave 节点由 master 任务负责翻转）。
	var stored UserSubscription
	require.NoError(t, DB.First(&stored, activeDue.Id).Error)
	assert.Equal(t, "active", stored.Status)

	// 有效状态排序：有效 active 排在过期/取消之前。
	seenInactive := false
	for _, d := range details {
		effectiveActive := d.Status == "active"
		if !effectiveActive {
			seenInactive = true
		}
		assert.False(t, seenInactive && effectiveActive, "effective-active rows must sort before inactive rows")
	}

	// 过滤按有效状态。
	_, total, err = AdminListAllUserSubscriptions(1, 10, "", "active")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total, "active filter must exclude due-but-unflipped rows")
	_, total, err = AdminListAllUserSubscriptions(1, 10, "", "expired")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total, "expired filter must include stored expired plus due active rows")
	_, total, err = AdminListAllUserSubscriptions(1, 10, "", "cancelled")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	// 美式拼写归一化。
	_, total, err = AdminListAllUserSubscriptions(1, 10, "", "canceled")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
}
