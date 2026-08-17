package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

type ChannelSubscriptionPoolService struct{}

var ChannelPools ChannelSubscriptionPoolService

func (ChannelSubscriptionPoolService) Create(req model.CreateChannelSubscriptionPoolRequest) (*model.ChannelSubscriptionPool, error) {
	return model.CreateChannelSubscriptionPool(req)
}

func (ChannelSubscriptionPoolService) List(page, pageSize int) ([]model.ChannelSubscriptionPool, int64, error) {
	return model.ListChannelSubscriptionPools(page, pageSize)
}

func (ChannelSubscriptionPoolService) ListOccupancies() ([]model.ChannelPoolChannelOccupancy, error) {
	return model.ListActiveChannelPoolOccupancies()
}

func (ChannelSubscriptionPoolService) Get(id int) (*model.ChannelSubscriptionPool, error) {
	return model.GetChannelSubscriptionPool(id)
}

func (ChannelSubscriptionPoolService) Update(id int, req model.UpdateChannelSubscriptionPoolRequest) (*model.ChannelSubscriptionPool, error) {
	return model.UpdateChannelSubscriptionPool(id, req)
}

func (ChannelSubscriptionPoolService) Cancel(id int) error {
	return model.CancelChannelSubscriptionPool(id)
}

func (ChannelSubscriptionPoolService) Reset(id int, scope string) error {
	return model.ResetChannelSubscriptionPool(id, scope)
}

func SettleChannelPoolActualUsage(ctx context.Context, channelId int, settlementKey string, actualQuota int, promptTokens, completionTokens, cacheTokens int) error {
	if channelId <= 0 || strings.TrimSpace(settlementKey) == "" {
		return errors.New("channel pool settlement requires channel and key")
	}
	pool, err := model.GetEligibleChannelSubscriptionPool(channelId)
	if err != nil {
		return fmt.Errorf("get channel subscription pool: %w", err)
	}
	if pool == nil {
		return nil
	}
	desiredTokens := subscriptionTokenQuotaUsage(promptTokens, completionTokens, cacheTokens, pool.IncludeCacheTokens)
	if _, err := model.SettleChannelPoolUsage(channelId, settlementKey, int64(actualQuota), desiredTokens); err != nil {
		return fmt.Errorf("settle channel subscription pool usage: %w", err)
	}
	return nil
}

// SettleUserSubscriptionActualUsage charges the selected user subscription's
// token quota after money/quota settle. Token pre-consume stays 0; this is the
// only place user-plan TokensUsed is incremented on the relay path.
func SettleUserSubscriptionActualUsage(relayInfo *relaycommon.RelayInfo, promptTokens, completionTokens, cacheTokens int) error {
	if relayInfo == nil || relayInfo.SubscriptionId <= 0 || relayInfo.BillingSource != BillingSourceSubscription {
		return nil
	}
	desiredTokens := subscriptionTokenQuotaUsage(promptTokens, completionTokens, cacheTokens, relayInfo.SubscriptionIncludeCacheTokens)
	if desiredTokens <= 0 {
		return nil
	}
	if err := model.PostConsumeUserSubscriptionDelta(relayInfo.SubscriptionId, 0, desiredTokens); err != nil {
		return fmt.Errorf("settle user subscription token usage: %w", err)
	}
	return nil
}

func SettleTaskChannelPoolUsage(ctx context.Context, task *model.Task, actualQuota int, actualTokens int64) error {
	if task == nil {
		return errors.New("task is required")
	}
	if task.ChannelId <= 0 || strings.TrimSpace(task.TaskID) == "" {
		return errors.New("task channel pool settlement requires channel and task id")
	}
	_, err := model.SettleChannelPoolUsage(task.ChannelId, "task:"+task.TaskID, int64(actualQuota), actualTokens)
	if errors.Is(err, model.ErrChannelPoolNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("settle task channel subscription pool usage: %w", err)
	}
	return nil
}
