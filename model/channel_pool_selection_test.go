package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelPoolSelectionTest(t *testing.T, memoryCache bool) {
	t.Helper()
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = memoryCache
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &ChannelSubscriptionPool{}, &ChannelSubscriptionPoolChannel{}, &ChannelPoolSettlement{}))
	for _, table := range []string{"channel_pool_settlements", "channel_subscription_pool_channels", "channel_subscription_pools", "abilities", "channels"} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
	t.Cleanup(func() {
		for _, table := range []string{"channel_pool_settlements", "channel_subscription_pool_channels", "channel_subscription_pools", "abilities", "channels"} {
			DB.Exec("DELETE FROM " + table)
		}
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		InitChannelCache()
	})
}

func seedChannelPoolSelectionCandidate(t *testing.T, id int, priority int64) {
	t.Helper()
	weight := uint(100)
	channel := &Channel{
		Id:       id,
		Name:     fmt.Sprintf("channel-%d", id),
		Key:      fmt.Sprintf("key-%d", id),
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "pool-selection-model",
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{Group: "default", Model: "pool-selection-model", ChannelId: id, Enabled: true, Priority: &priority, Weight: 100}).Error)
}

func TestChannelSelectionSkipsExhaustedSharedPoolBeforePriority(t *testing.T) {
	for _, memoryCache := range []bool{false, true} {
		t.Run(fmt.Sprintf("memory_cache_%t", memoryCache), func(t *testing.T) {
			setupChannelPoolSelectionTest(t, memoryCache)
			seedChannelPoolSelectionCandidate(t, 7101, 100)
			seedChannelPoolSelectionCandidate(t, 7102, 100)
			seedChannelPoolSelectionCandidate(t, 7103, 10)

			now := GetDBTimestamp()
			pool := &ChannelSubscriptionPool{
				Name: "exhausted", PlanId: 1, PlanTitle: "shared",
				AmountTotal: 50, AmountUsed: 50, TokensTotal: 1000, TokensUsed: 20,
				StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive,
			}
			require.NoError(t, DB.Create(pool).Error)
			require.NoError(t, DB.Create(&[]ChannelSubscriptionPoolChannel{{PoolId: pool.Id, ChannelId: 7101}, {PoolId: pool.Id, ChannelId: 7102}}).Error)
			if memoryCache {
				InitChannelCache()
			}

			selected, err := GetRandomSatisfiedChannel("default", "pool-selection-model", 0, "")

			require.NoError(t, err)
			require.NotNil(t, selected)
			assert.Equal(t, 7103, selected.Id)
			for _, id := range []int{7101, 7102, 7103} {
				var got Channel
				require.NoError(t, DB.First(&got, id).Error)
				assert.Equal(t, common.ChannelStatusEnabled, got.Status, "pool exhaustion must not mutate channel status")
			}
		})
	}
}

func TestChannelSelectionSkipsTokenExhaustedPool(t *testing.T) {
	setupChannelPoolSelectionTest(t, false)
	seedChannelPoolSelectionCandidate(t, 7201, 100)
	seedChannelPoolSelectionCandidate(t, 7202, 10)
	now := GetDBTimestamp()
	pool := &ChannelSubscriptionPool{
		Name: "token-exhausted", PlanId: 1, PlanTitle: "shared",
		TokensTotal: 100, TokensUsed: 100,
		StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive,
	}
	require.NoError(t, DB.Create(pool).Error)
	require.NoError(t, DB.Create(&ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: 7201}).Error)

	selected, err := GetRandomSatisfiedChannel("default", "pool-selection-model", 0, "")

	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 7202, selected.Id)
}

// When every candidate channel belongs to an exhausted pool, selection must
// degrade to "no channel" (nil, nil) instead of panicking or returning a
// stale pool member.
func TestChannelSelectionAllCandidatesInExhaustedPoolsReturnsNil(t *testing.T) {
	for _, memoryCache := range []bool{false, true} {
		t.Run(fmt.Sprintf("memory_cache_%t", memoryCache), func(t *testing.T) {
			setupChannelPoolSelectionTest(t, memoryCache)
			seedChannelPoolSelectionCandidate(t, 7301, 100)
			seedChannelPoolSelectionCandidate(t, 7302, 10)
			now := GetDBTimestamp()
			pool := &ChannelSubscriptionPool{
				Name: "all-exhausted", PlanId: 1, PlanTitle: "shared",
				AmountTotal: 50, AmountUsed: 50, TokensTotal: 100, TokensUsed: 100,
				StartTime: now - 10, EndTime: now + 3600, Status: ChannelPoolStatusActive,
			}
			require.NoError(t, DB.Create(pool).Error)
			require.NoError(t, DB.Create(&[]ChannelSubscriptionPoolChannel{{PoolId: pool.Id, ChannelId: 7301}, {PoolId: pool.Id, ChannelId: 7302}}).Error)
			if memoryCache {
				InitChannelCache()
			}

			selected, err := GetRandomSatisfiedChannel("default", "pool-selection-model", 0, "")

			require.NoError(t, err)
			assert.Nil(t, selected, "all candidates in exhausted pools must yield no channel")
		})
	}
}
