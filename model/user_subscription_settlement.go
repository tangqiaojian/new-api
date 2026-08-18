package model

import (
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// UserSubscriptionSettlement records the tokens one settlement key has applied
// to a user subscription. Retrying the same key only applies the delta from the
// previously stored contribution (channel-pool desired-total style).
type UserSubscriptionSettlement struct {
	Id                 int    `json:"id"`
	UserSubscriptionId int    `json:"user_subscription_id" gorm:"uniqueIndex:idx_user_sub_settlement,priority:1;index;not null"`
	SettlementKey      string `json:"settlement_key" gorm:"uniqueIndex:idx_user_sub_settlement,priority:2;type:varchar(191);not null"`
	TokensSettled      int64  `json:"tokens_settled" gorm:"type:bigint;not null;default:0"`
	CreatedAt          int64  `json:"created_at" gorm:"type:bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"type:bigint;index"`
}

func (s *UserSubscriptionSettlement) BeforeCreate(*gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *UserSubscriptionSettlement) BeforeUpdate(*gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

// SettleUserSubscriptionUsage applies desiredTokens for settlementKey onto the
// user subscription. Re-entry with the same key only applies the difference
// from TokensSettled. The stored contribution is the amount actually applied
// after [0, TokensTotal] clamp, so a later refund of 0 reverses only what this
// key added.
func SettleUserSubscriptionUsage(userSubscriptionId int, key string, desiredTokens int64) (*UserSubscription, error) {
	if userSubscriptionId <= 0 || strings.TrimSpace(key) == "" {
		return nil, errors.New("invalid user subscription settlement")
	}
	if desiredTokens < 0 {
		return nil, errors.New("desired settlement totals cannot be negative")
	}
	key = strings.TrimSpace(key)
	var result *UserSubscription
	err := DB.Transaction(func(tx *gorm.DB) error {
		var settlement UserSubscriptionSettlement
		err := tx.Where("user_subscription_id = ? AND settlement_key = ?", userSubscriptionId, key).First(&settlement).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			settlement = UserSubscriptionSettlement{UserSubscriptionId: userSubscriptionId, SettlementKey: key}
			if err := tx.Create(&settlement).Error; err != nil {
				// A concurrent settle with the same key committed first; fall
				// through to the stored row instead of failing the whole settle.
				if err2 := tx.Where("user_subscription_id = ? AND settlement_key = ?", userSubscriptionId, key).First(&settlement).Error; err2 != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(&settlement, settlement.Id).Error; err != nil {
			return err
		}
		var sub UserSubscription
		if err := lockForUpdate(tx).First(&sub, userSubscriptionId).Error; err != nil {
			return err
		}
		tokenDelta := desiredTokens - settlement.TokensSettled
		if tokenDelta > 0 {
			room := int64(math.MaxInt64)
			if sub.TokensUsed < room {
				room = room - sub.TokensUsed
			} else {
				room = 0
			}
			if sub.TokensTotal > 0 {
				if sub.TokensUsed >= sub.TokensTotal {
					room = 0
				} else {
					room = sub.TokensTotal - sub.TokensUsed
				}
			}
			if tokenDelta > room {
				tokenDelta = room
			}
		} else if tokenDelta < 0 {
			if sub.TokensUsed == 0 {
				tokenDelta = 0
			} else if tokenDelta < -sub.TokensUsed {
				tokenDelta = -sub.TokensUsed
			}
		}
		if err := applyUserSubscriptionDeltaTx(tx, userSubscriptionId, 0, tokenDelta); err != nil {
			return err
		}
		applied := settlement.TokensSettled + tokenDelta
		if err := tx.Model(&settlement).Updates(map[string]interface{}{
			"tokens_settled": applied,
			"updated_at":     common.GetTimestamp(),
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&sub, userSubscriptionId).Error; err != nil {
			return err
		}
		result = &sub
		return nil
	})
	return result, err
}

func CleanupUserSubscriptionSettlements(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		return 0, errors.New("olderThanSeconds must be positive")
	}
	result := DB.Where("updated_at < ?", GetDBTimestamp()-olderThanSeconds).Delete(&UserSubscriptionSettlement{})
	return result.RowsAffected, result.Error
}
