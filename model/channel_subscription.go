package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ChannelPoolStatusActive    = "active"
	ChannelPoolStatusExpired   = "expired"
	ChannelPoolStatusCancelled = "cancelled"
)

var (
	ErrChannelPoolNotFound       = errors.New("channel subscription pool not found")
	ErrChannelPoolMembership     = errors.New("channel already belongs to a subscription pool")
	ErrChannelPoolPlanNotAllowed = errors.New("subscription plan does not allow channels")
)

type ChannelSubscriptionPool struct {
	Id        int    `json:"id"`
	Name      string `json:"name" gorm:"type:varchar(128);not null"`
	PlanId    int    `json:"plan_id" gorm:"index;not null"`
	PlanTitle string `json:"plan_title" gorm:"type:varchar(128);not null"`

	AmountTotal int64 `json:"amount_total" gorm:"type:bigint;not null;default:0"`
	AmountUsed  int64 `json:"amount_used" gorm:"type:bigint;not null;default:0"`
	TokensTotal int64 `json:"tokens_total" gorm:"type:bigint;not null;default:0"`
	TokensUsed  int64 `json:"tokens_used" gorm:"type:bigint;not null;default:0"`

	IncludeCacheTokens bool   `json:"include_cache_tokens"`
	StartTime          int64  `json:"start_time" gorm:"type:bigint;index"`
	EndTime            int64  `json:"end_time" gorm:"type:bigint;index"`
	Status             string `json:"status" gorm:"type:varchar(32);index"`

	QuotaResetPeriod        string  `json:"quota_reset_period" gorm:"type:varchar(16);default:'never'"`
	QuotaResetCustomSeconds int64   `json:"quota_reset_custom_seconds" gorm:"type:bigint;default:0"`
	QuotaResetAnchor        *int64  `json:"quota_reset_anchor" gorm:"type:bigint"`
	QuotaResetTimezone      *string `json:"quota_reset_timezone" gorm:"type:varchar(64)"`
	QuotaResetOverridden    bool    `json:"quota_reset_overridden"`
	QuotaLastResetTime      int64   `json:"quota_last_reset_time" gorm:"type:bigint;default:0"`
	QuotaNextResetTime      int64   `json:"quota_next_reset_time" gorm:"type:bigint;default:0;index"`

	TokenResetPeriod        string  `json:"token_reset_period" gorm:"type:varchar(16);default:''"`
	TokenResetCustomSeconds int64   `json:"token_reset_custom_seconds" gorm:"type:bigint;default:0"`
	TokenResetAnchor        *int64  `json:"token_reset_anchor" gorm:"type:bigint"`
	TokenResetTimezone      *string `json:"token_reset_timezone" gorm:"type:varchar(64)"`
	TokenResetOverridden    bool    `json:"token_reset_overridden"`
	TokenLastResetTime      int64   `json:"token_last_reset_time" gorm:"type:bigint;default:0"`
	TokenNextResetTime      int64   `json:"token_next_reset_time" gorm:"type:bigint;default:0;index"`

	CreatedAt int64                            `json:"created_at" gorm:"type:bigint"`
	UpdatedAt int64                            `json:"updated_at" gorm:"type:bigint"`
	Channels  []ChannelSubscriptionPoolChannel `json:"channels,omitempty" gorm:"foreignKey:PoolId"`
}

func (p *ChannelSubscriptionPool) BeforeCreate(*gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *ChannelSubscriptionPool) BeforeUpdate(*gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

type ChannelSubscriptionPoolChannel struct {
	Id        int   `json:"id"`
	PoolId    int   `json:"pool_id" gorm:"index;not null"`
	ChannelId int   `json:"channel_id" gorm:"uniqueIndex;not null"`
	CreatedAt int64 `json:"created_at" gorm:"type:bigint"`
}

func (m *ChannelSubscriptionPoolChannel) BeforeCreate(*gorm.DB) error {
	m.CreatedAt = common.GetTimestamp()
	return nil
}

type ChannelPoolSettlement struct {
	Id            int    `json:"id"`
	PoolId        int    `json:"pool_id" gorm:"uniqueIndex:idx_channel_pool_settlement,priority:1;index;not null"`
	ChannelId     int    `json:"channel_id" gorm:"index;not null"`
	SettlementKey string `json:"settlement_key" gorm:"uniqueIndex:idx_channel_pool_settlement,priority:2;type:varchar(191);not null"`
	AmountSettled int64  `json:"amount_settled" gorm:"type:bigint;not null;default:0"`
	TokensSettled int64  `json:"tokens_settled" gorm:"type:bigint;not null;default:0"`
	CreatedAt     int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt     int64  `json:"updated_at" gorm:"type:bigint;index"`
}

func (s *ChannelPoolSettlement) BeforeCreate(*gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *ChannelPoolSettlement) BeforeUpdate(*gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

type CreateChannelSubscriptionPoolRequest struct {
	Name               string  `json:"name"`
	PlanId             int     `json:"plan_id"`
	ChannelIds         []int   `json:"channel_ids"`
	QuotaResetAnchor   *int64  `json:"quota_reset_anchor"`
	QuotaResetTimezone *string `json:"quota_reset_timezone"`
	TokenResetAnchor   *int64  `json:"token_reset_anchor"`
	TokenResetTimezone *string `json:"token_reset_timezone"`
}

type UpdateChannelSubscriptionPoolRequest struct {
	Name               string  `json:"name"`
	ChannelIds         []int   `json:"channel_ids"`
	QuotaResetAnchor   *int64  `json:"quota_reset_anchor"`
	QuotaResetTimezone *string `json:"quota_reset_timezone"`
	TokenResetAnchor   *int64  `json:"token_reset_anchor"`
	TokenResetTimezone *string `json:"token_reset_timezone"`
}

func poolNextQuotaReset(base time.Time, pool *ChannelSubscriptionPool) int64 {
	if pool == nil {
		return 0
	}
	timezone := ""
	if pool.QuotaResetTimezone != nil {
		timezone = *pool.QuotaResetTimezone
	}
	return calcNextResetSchedule(base, pool.QuotaResetPeriod, pool.QuotaResetCustomSeconds, pool.QuotaResetAnchor, timezone, pool.EndTime)
}

func poolNextTokenReset(base time.Time, pool *ChannelSubscriptionPool) int64 {
	if pool == nil {
		return 0
	}
	period := pool.TokenResetPeriod
	customSeconds := pool.TokenResetCustomSeconds
	anchor := pool.TokenResetAnchor
	timezone := ""
	if pool.TokenResetTimezone != nil {
		timezone = *pool.TokenResetTimezone
	}
	if strings.TrimSpace(period) == "" {
		period = pool.QuotaResetPeriod
		customSeconds = pool.QuotaResetCustomSeconds
		if anchor == nil {
			anchor = pool.QuotaResetAnchor
		}
		if pool.TokenResetTimezone == nil && pool.QuotaResetTimezone != nil {
			timezone = *pool.QuotaResetTimezone
		}
	}
	return calcNextResetSchedule(base, period, customSeconds, anchor, timezone, pool.EndTime)
}

func validatePoolTimezone(value *string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	if _, err := time.LoadLocation(strings.TrimSpace(*value)); err != nil {
		return fmt.Errorf("invalid reset timezone: %w", err)
	}
	return nil
}

func uniquePositiveChannelIds(ids []int) ([]int, error) {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("channel ids must be positive")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	if len(result) == 0 {
		return nil, errors.New("at least one channel is required")
	}
	return result, nil
}

func CreateChannelSubscriptionPool(req CreateChannelSubscriptionPoolRequest) (*ChannelSubscriptionPool, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("pool name is required")
	}
	if len(name) > 128 {
		return nil, errors.New("pool name is too long")
	}
	channelIds, err := uniquePositiveChannelIds(req.ChannelIds)
	if err != nil {
		return nil, err
	}
	if err := validatePoolTimezone(req.QuotaResetTimezone); err != nil {
		return nil, err
	}
	if err := validatePoolTimezone(req.TokenResetTimezone); err != nil {
		return nil, err
	}
	plan, err := GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		return nil, fmt.Errorf("get subscription plan: %w", err)
	}
	planType, err := NormalizeSubscriptionPlanType(plan.PlanType)
	if err != nil || (planType != SubscriptionPlanTypeChannel && planType != SubscriptionPlanTypeBoth) {
		return nil, ErrChannelPoolPlanNotAllowed
	}
	now := GetDBTimestamp()
	start := now
	end, err := calcPlanEndTime(time.Unix(start, 0), plan)
	if err != nil {
		return nil, fmt.Errorf("calculate pool end time: %w", err)
	}
	quotaAnchor := req.QuotaResetAnchor
	quotaTimezone := req.QuotaResetTimezone
	if quotaAnchor == nil {
		quotaAnchor = plan.QuotaResetAnchor
	}
	if quotaTimezone == nil && strings.TrimSpace(plan.QuotaResetTimezone) != "" {
		value := plan.QuotaResetTimezone
		quotaTimezone = &value
	}
	tokenAnchor := req.TokenResetAnchor
	tokenTimezone := req.TokenResetTimezone
	if strings.TrimSpace(plan.TokenResetPeriod) == "" {
		if tokenAnchor == nil {
			tokenAnchor = quotaAnchor
		}
		if tokenTimezone == nil {
			tokenTimezone = quotaTimezone
		}
	} else {
		if tokenAnchor == nil {
			tokenAnchor = plan.TokenResetAnchor
		}
		if tokenTimezone == nil && strings.TrimSpace(plan.TokenResetTimezone) != "" {
			value := plan.TokenResetTimezone
			tokenTimezone = &value
		}
	}
	pool := &ChannelSubscriptionPool{
		Name: name, PlanId: plan.Id, PlanTitle: plan.Title, AmountTotal: plan.TotalAmount, TokensTotal: plan.TotalTokens,
		IncludeCacheTokens: plan.IncludeCacheTokens, StartTime: start, EndTime: end, Status: ChannelPoolStatusActive,
		QuotaResetPeriod: plan.QuotaResetPeriod, QuotaResetCustomSeconds: plan.QuotaResetCustomSeconds,
		QuotaResetAnchor: quotaAnchor, QuotaResetTimezone: quotaTimezone,
		QuotaResetOverridden: req.QuotaResetAnchor != nil || req.QuotaResetTimezone != nil,
		TokenResetPeriod:     plan.TokenResetPeriod, TokenResetCustomSeconds: plan.TokenResetCustomSeconds,
		TokenResetAnchor: tokenAnchor, TokenResetTimezone: tokenTimezone,
		TokenResetOverridden: req.TokenResetAnchor != nil || req.TokenResetTimezone != nil,
	}
	pool.QuotaNextResetTime = poolNextQuotaReset(time.Unix(start, 0), pool)
	pool.TokenNextResetTime = poolNextTokenReset(time.Unix(start, 0), pool)
	if pool.QuotaNextResetTime > 0 {
		pool.QuotaLastResetTime = start
	}
	if pool.TokenNextResetTime > 0 {
		pool.TokenLastResetTime = start
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&Channel{}).Where("id IN ?", channelIds).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(channelIds)) {
			return errors.New("one or more channels do not exist")
		}
		if err := tx.Model(&ChannelSubscriptionPoolChannel{}).Where("channel_id IN ?", channelIds).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrChannelPoolMembership
		}
		if err := tx.Create(pool).Error; err != nil {
			return err
		}
		members := make([]ChannelSubscriptionPoolChannel, 0, len(channelIds))
		for _, channelId := range channelIds {
			members = append(members, ChannelSubscriptionPoolChannel{PoolId: pool.Id, ChannelId: channelId})
		}
		return tx.Create(&members).Error
	})
	if err != nil {
		return nil, err
	}
	return GetChannelSubscriptionPool(pool.Id)
}

func ListChannelSubscriptionPools(page, pageSize int) ([]ChannelSubscriptionPool, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var pools []ChannelSubscriptionPool
	var total int64
	if err := DB.Model(&ChannelSubscriptionPool{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Preload("Channels").Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&pools).Error
	return pools, total, err
}

func GetChannelSubscriptionPool(id int) (*ChannelSubscriptionPool, error) {
	var pool ChannelSubscriptionPool
	if err := DB.Preload("Channels").First(&pool, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChannelPoolNotFound
		}
		return nil, err
	}
	return &pool, nil
}

func UpdateChannelSubscriptionPool(id int, req UpdateChannelSubscriptionPoolRequest) (*ChannelSubscriptionPool, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("pool name is required")
	}
	if len(name) > 128 {
		return nil, errors.New("pool name is too long")
	}
	// Reject an empty channel list: updating a pool to zero members would leave an
	// active pool that no channel is routed into. Use Cancel to remove a pool instead.
	channelIds := make([]int, 0, len(req.ChannelIds))
	for _, id := range req.ChannelIds {
		if id > 0 {
			channelIds = append(channelIds, id)
		}
	}
	if len(channelIds) == 0 {
		return nil, errors.New("at least one channel is required")
	}
	if err := validatePoolTimezone(req.QuotaResetTimezone); err != nil {
		return nil, err
	}
	if err := validatePoolTimezone(req.TokenResetTimezone); err != nil {
		return nil, err
	}
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		var pool ChannelSubscriptionPool
		if err := lockForUpdate(tx).First(&pool, id).Error; err != nil {
			return err
		}
		if pool.Status != ChannelPoolStatusActive && pool.Status != ChannelPoolStatusCancelled {
			return errors.New("only active or cancelled pools can be updated")
		}
		var count int64
		if err := tx.Model(&Channel{}).Where("id IN ?", channelIds).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(channelIds)) {
			return errors.New("one or more channels do not exist")
		}
		if err := tx.Model(&ChannelSubscriptionPoolChannel{}).Where("channel_id IN ? AND pool_id <> ?", channelIds, id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrChannelPoolMembership
		}
		updates := map[string]interface{}{"name": name, "quota_reset_anchor": req.QuotaResetAnchor, "quota_reset_timezone": req.QuotaResetTimezone, "quota_reset_overridden": req.QuotaResetAnchor != nil || req.QuotaResetTimezone != nil, "token_reset_anchor": req.TokenResetAnchor, "token_reset_timezone": req.TokenResetTimezone, "token_reset_overridden": req.TokenResetAnchor != nil || req.TokenResetTimezone != nil, "updated_at": common.GetTimestamp()}
		updatedPool := pool
		updatedPool.QuotaResetAnchor = req.QuotaResetAnchor
		updatedPool.QuotaResetTimezone = req.QuotaResetTimezone
		updatedPool.TokenResetAnchor = req.TokenResetAnchor
		updatedPool.TokenResetTimezone = req.TokenResetTimezone
		updates["quota_next_reset_time"] = poolNextQuotaReset(time.Unix(now, 0), &updatedPool)
		updates["token_next_reset_time"] = poolNextTokenReset(time.Unix(now, 0), &updatedPool)
		if err := tx.Model(&pool).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("pool_id = ?", id).Delete(&ChannelSubscriptionPoolChannel{}).Error; err != nil {
			return err
		}
		members := make([]ChannelSubscriptionPoolChannel, 0, len(channelIds))
		for _, channelId := range channelIds {
			members = append(members, ChannelSubscriptionPoolChannel{PoolId: id, ChannelId: channelId})
		}
		return tx.Create(&members).Error
	})
	if err != nil {
		return nil, err
	}
	return GetChannelSubscriptionPool(id)
}

func CancelChannelSubscriptionPool(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&ChannelSubscriptionPool{}).Where("id = ? AND status = ?", id, ChannelPoolStatusActive).Updates(map[string]interface{}{"status": ChannelPoolStatusCancelled, "updated_at": common.GetTimestamp()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrChannelPoolNotFound
		}
		return tx.Where("pool_id = ?", id).Delete(&ChannelSubscriptionPoolChannel{}).Error
	})
}

func ResetChannelSubscriptionPool(id int, scope string) error {
	scope, err := NormalizeResetScope(scope)
	if err != nil {
		return err
	}
	now := GetDBTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var pool ChannelSubscriptionPool
		if err := lockForUpdate(tx).First(&pool, id).Error; err != nil {
			return err
		}
		if pool.Status != ChannelPoolStatusActive {
			return errors.New("only active pools can be reset")
		}
		updates := map[string]interface{}{"updated_at": common.GetTimestamp()}
		if scope != SubscriptionResetScopeTokens {
			nextQuota := poolNextQuotaReset(time.Unix(now, 0), &pool)
			updates["amount_used"] = 0
			updates["quota_next_reset_time"] = nextQuota
			if nextQuota > 0 {
				updates["quota_last_reset_time"] = now
			} else {
				updates["quota_last_reset_time"] = 0
			}
		}
		if scope != SubscriptionResetScopeQuota {
			nextToken := poolNextTokenReset(time.Unix(now, 0), &pool)
			updates["tokens_used"] = 0
			updates["token_next_reset_time"] = nextToken
			if nextToken > 0 {
				updates["token_last_reset_time"] = now
			} else {
				updates["token_last_reset_time"] = 0
			}
		}
		// Usage counters restart from zero, so settlement contributions from the
		// previous cycle must go too: a late settle-to-zero on an old key would
		// otherwise subtract from fresh usage. Deleting is safe for pools — a
		// re-settled old key over-counts, which only throttles earlier.
		if err := tx.Where("pool_id = ?", id).Delete(&ChannelPoolSettlement{}).Error; err != nil {
			return err
		}
		return tx.Model(&pool).Updates(updates).Error
	})
}

// ChannelPoolChannelOccupancy describes a channel currently bound to an active pool.
type ChannelPoolChannelOccupancy struct {
	ChannelId int    `json:"channel_id"`
	PoolId    int    `json:"pool_id"`
	PoolName  string `json:"pool_name"`
}

// ListActiveChannelPoolOccupancies returns active pool memberships for admin UI filtering.
func ListActiveChannelPoolOccupancies() ([]ChannelPoolChannelOccupancy, error) {
	now := GetDBTimestamp()
	var rows []ChannelPoolChannelOccupancy
	err := DB.Table("channel_subscription_pool_channels AS memberships").
		Select("memberships.channel_id AS channel_id, memberships.pool_id AS pool_id, channel_subscription_pools.name AS pool_name").
		Joins("JOIN channel_subscription_pools ON channel_subscription_pools.id = memberships.pool_id").
		Where("channel_subscription_pools.status = ? AND channel_subscription_pools.end_time > ?", ChannelPoolStatusActive, now).
		Order("memberships.channel_id").
		Scan(&rows).Error
	return rows, err
}

// IsChannelSubscriptionPoolRoutable reports whether an active pool can accept new traffic.
// Both totals zero means unlimited. Exhausted pools are not routable.
func IsChannelSubscriptionPoolRoutable(pool *ChannelSubscriptionPool) bool {
	if pool == nil || pool.Status != ChannelPoolStatusActive {
		return false
	}
	now := GetDBTimestamp()
	if pool.StartTime > now || pool.EndTime <= now {
		return false
	}
	if pool.AmountTotal > 0 && pool.AmountUsed >= pool.AmountTotal {
		return false
	}
	if pool.TokensTotal > 0 && pool.TokensUsed >= pool.TokensTotal {
		return false
	}
	return true
}

func recoverChannelPoolDueTx(tx *gorm.DB, pool *ChannelSubscriptionPool, now int64) error {
	updates := map[string]interface{}{}
	if pool.QuotaNextResetTime > 0 && pool.QuotaNextResetTime <= now {
		base := pool.QuotaNextResetTime
		next := poolNextQuotaReset(time.Unix(base, 0), pool)
		for next > 0 && next <= now {
			base = next
			next = poolNextQuotaReset(time.Unix(base, 0), pool)
		}
		updates["amount_used"] = 0
		updates["quota_last_reset_time"] = base
		updates["quota_next_reset_time"] = next
	}
	if pool.TokenNextResetTime > 0 && pool.TokenNextResetTime <= now {
		base := pool.TokenNextResetTime
		next := poolNextTokenReset(time.Unix(base, 0), pool)
		for next > 0 && next <= now {
			base = next
			next = poolNextTokenReset(time.Unix(base, 0), pool)
		}
		updates["tokens_used"] = 0
		updates["token_last_reset_time"] = base
		updates["token_next_reset_time"] = next
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = common.GetTimestamp()
	// See ResetChannelSubscriptionPool: old-cycle settlement rows must not eat
	// the fresh counters this recovery just zeroed.
	if err := tx.Where("pool_id = ?", pool.Id).Delete(&ChannelPoolSettlement{}).Error; err != nil {
		return err
	}
	if err := tx.Model(pool).Updates(updates).Error; err != nil {
		return err
	}
	return tx.First(pool, pool.Id).Error
}

func getEligibleChannelPoolTx(tx *gorm.DB, channelId int, now int64, lock bool) (*ChannelSubscriptionPool, error) {
	query := tx.Model(&ChannelSubscriptionPool{}).Joins("JOIN channel_subscription_pool_channels memberships ON memberships.pool_id = channel_subscription_pools.id").Where("memberships.channel_id = ? AND channel_subscription_pools.status = ? AND channel_subscription_pools.start_time <= ? AND channel_subscription_pools.end_time > ?", channelId, ChannelPoolStatusActive, now, now)
	if lock {
		query = lockForUpdate(query)
	}
	var pool ChannelSubscriptionPool
	if err := query.First(&pool).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := recoverChannelPoolDueTx(tx, &pool, now); err != nil {
		return nil, err
	}
	return &pool, nil
}

func GetEligibleChannelSubscriptionPool(channelId int) (*ChannelSubscriptionPool, error) {
	now := GetDBTimestamp()
	var pool *ChannelSubscriptionPool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		pool, err = getEligibleChannelPoolTx(tx, channelId, now, true)
		return err
	})
	return pool, err
}

func GetEligibleChannelSubscriptionPools(channelIds []int) (map[int]*ChannelSubscriptionPool, error) {
	result := make(map[int]*ChannelSubscriptionPool, len(channelIds))
	uniqueIds, err := uniquePositiveChannelIds(channelIds)
	if err != nil {
		if len(channelIds) == 0 {
			return result, nil
		}
		return nil, err
	}
	for _, channelId := range uniqueIds {
		result[channelId] = nil
	}
	now := GetDBTimestamp()
	type poolMembership struct {
		ChannelId int `gorm:"column:channel_id"`
		ChannelSubscriptionPool
	}
	var rows []poolMembership
	err = DB.Model(&ChannelSubscriptionPool{}).
		Select("memberships.channel_id, channel_subscription_pools.*").
		Joins("JOIN channel_subscription_pool_channels memberships ON memberships.pool_id = channel_subscription_pools.id").
		Where("memberships.channel_id IN ? AND channel_subscription_pools.status = ? AND channel_subscription_pools.start_time <= ? AND channel_subscription_pools.end_time > ?", uniqueIds, ChannelPoolStatusActive, now, now).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	poolById := make(map[int]*ChannelSubscriptionPool, len(rows))
	for i := range rows {
		pool := rows[i].ChannelSubscriptionPool
		poolById[pool.Id] = &pool
	}
	for poolId, pool := range poolById {
		if pool.QuotaNextResetTime <= 0 || pool.QuotaNextResetTime > now {
			if pool.TokenNextResetTime <= 0 || pool.TokenNextResetTime > now {
				continue
			}
		}
		var recovered *ChannelSubscriptionPool
		err := DB.Transaction(func(tx *gorm.DB) error {
			var locked ChannelSubscriptionPool
			if err := lockForUpdate(tx).First(&locked, poolId).Error; err != nil {
				return err
			}
			if err := recoverChannelPoolDueTx(tx, &locked, now); err != nil {
				return err
			}
			recovered = &locked
			return nil
		})
		if err != nil {
			return nil, err
		}
		poolById[poolId] = recovered
	}
	for i := range rows {
		result[rows[i].ChannelId] = poolById[rows[i].Id]
	}
	return result, nil
}

func SettleChannelPoolUsage(channelId int, key string, desiredAmount, desiredTokens int64) (*ChannelSubscriptionPool, error) {
	if channelId <= 0 || strings.TrimSpace(key) == "" {
		return nil, errors.New("invalid channel pool settlement")
	}
	if desiredAmount < 0 || desiredTokens < 0 {
		return nil, errors.New("desired settlement totals cannot be negative")
	}
	var result *ChannelSubscriptionPool
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		pool, err := getEligibleChannelPoolTx(tx, channelId, now, true)
		if err != nil {
			return err
		}
		if pool == nil {
			return ErrChannelPoolNotFound
		}
		var settlement ChannelPoolSettlement
		err = tx.Where("pool_id = ? AND settlement_key = ?", pool.Id, strings.TrimSpace(key)).First(&settlement).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			settlement = ChannelPoolSettlement{PoolId: pool.Id, ChannelId: channelId, SettlementKey: strings.TrimSpace(key)}
			if err := tx.Create(&settlement).Error; err != nil {
				// A concurrent settle with the same key committed first; fall
				// through to the stored row instead of failing the whole settle.
				if err2 := tx.Where("pool_id = ? AND settlement_key = ?", pool.Id, strings.TrimSpace(key)).First(&settlement).Error; err2 != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(&settlement, settlement.Id).Error; err != nil {
			return err
		}
		amountDelta := desiredAmount - settlement.AmountSettled
		tokenDelta := desiredTokens - settlement.TokensSettled
		amountUsed := pool.AmountUsed + amountDelta
		tokensUsed := pool.TokensUsed + tokenDelta
		if amountUsed < 0 {
			amountUsed = 0
		}
		if tokensUsed < 0 {
			tokensUsed = 0
		}
		if err := tx.Model(pool).Updates(map[string]interface{}{"amount_used": amountUsed, "tokens_used": tokensUsed, "updated_at": common.GetTimestamp()}).Error; err != nil {
			return err
		}
		if err := tx.Model(&settlement).Updates(map[string]interface{}{"amount_settled": desiredAmount, "tokens_settled": desiredTokens, "updated_at": common.GetTimestamp()}).Error; err != nil {
			return err
		}
		pool.AmountUsed = amountUsed
		pool.TokensUsed = tokensUsed
		result = pool
		return nil
	})
	return result, err
}

func recalculateChannelPoolResetTimesTx(tx *gorm.DB, plan *SubscriptionPlan, now int64) error {
	var pools []ChannelSubscriptionPool
	if err := lockForUpdate(tx).Where("plan_id = ? AND status = ? AND end_time > ?", plan.Id, ChannelPoolStatusActive, now).Find(&pools).Error; err != nil {
		return err
	}
	for i := range pools {
		pool := &pools[i]
		updates := map[string]interface{}{}
		if !pool.QuotaResetOverridden {
			pool.QuotaResetPeriod = plan.QuotaResetPeriod
			pool.QuotaResetCustomSeconds = plan.QuotaResetCustomSeconds
			pool.QuotaResetAnchor = plan.QuotaResetAnchor
			pool.QuotaResetTimezone = nil
			if strings.TrimSpace(plan.QuotaResetTimezone) != "" {
				value := plan.QuotaResetTimezone
				pool.QuotaResetTimezone = &value
			}
			updates["quota_reset_period"] = pool.QuotaResetPeriod
			updates["quota_reset_custom_seconds"] = pool.QuotaResetCustomSeconds
			updates["quota_reset_anchor"] = pool.QuotaResetAnchor
			updates["quota_reset_timezone"] = pool.QuotaResetTimezone
			updates["quota_next_reset_time"] = poolNextQuotaReset(time.Unix(now, 0), pool)
		}
		if !pool.TokenResetOverridden {
			pool.TokenResetPeriod = plan.TokenResetPeriod
			pool.TokenResetCustomSeconds = plan.TokenResetCustomSeconds
			if strings.TrimSpace(plan.TokenResetPeriod) == "" {
				pool.TokenResetAnchor = pool.QuotaResetAnchor
				pool.TokenResetTimezone = pool.QuotaResetTimezone
			} else {
				pool.TokenResetAnchor = plan.TokenResetAnchor
				pool.TokenResetTimezone = nil
				if strings.TrimSpace(plan.TokenResetTimezone) != "" {
					value := plan.TokenResetTimezone
					pool.TokenResetTimezone = &value
				}
			}
			updates["token_reset_period"] = pool.TokenResetPeriod
			updates["token_reset_custom_seconds"] = pool.TokenResetCustomSeconds
			updates["token_reset_anchor"] = pool.TokenResetAnchor
			updates["token_reset_timezone"] = pool.TokenResetTimezone
			updates["token_next_reset_time"] = poolNextTokenReset(time.Unix(now, 0), pool)
		}
		if len(updates) > 0 {
			updates["updated_at"] = common.GetTimestamp()
			if err := tx.Model(pool).Updates(updates).Error; err != nil {
				return fmt.Errorf("recalculate channel pool %d reset times: %w", pool.Id, err)
			}
		}
	}
	return nil
}

func ExpireDueChannelSubscriptionPools(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var ids []int
	if err := DB.Model(&ChannelSubscriptionPool{}).Where("status = ? AND end_time <= ?", ChannelPoolStatusActive, now).Order("end_time").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ChannelSubscriptionPool{}).Where("id IN ?", ids).Updates(map[string]interface{}{"status": ChannelPoolStatusExpired, "updated_at": common.GetTimestamp()}).Error; err != nil {
			return err
		}
		return tx.Where("pool_id IN ?", ids).Delete(&ChannelSubscriptionPoolChannel{}).Error
	})
	return len(ids), err
}

func ResetDueChannelSubscriptionPools(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var ids []int
	if err := DB.Model(&ChannelSubscriptionPool{}).Where("status = ? AND ((quota_next_reset_time > 0 AND quota_next_reset_time <= ?) OR (token_next_reset_time > 0 AND token_next_reset_time <= ?))", ChannelPoolStatusActive, now, now).Limit(limit).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	count := 0
	for _, id := range ids {
		err := DB.Transaction(func(tx *gorm.DB) error {
			var pool ChannelSubscriptionPool
			if err := lockForUpdate(tx).First(&pool, id).Error; err != nil {
				return err
			}
			return recoverChannelPoolDueTx(tx, &pool, now)
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func CleanupChannelPoolSettlements(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		return 0, errors.New("olderThanSeconds must be positive")
	}
	result := DB.Where("updated_at < ?", GetDBTimestamp()-olderThanSeconds).Delete(&ChannelPoolSettlement{})
	return result.RowsAffected, result.Error
}
