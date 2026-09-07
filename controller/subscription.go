package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

type SubscriptionBalancePayRequest struct {
	PlanId int `json:"plan_id"`
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiSuccess(c, []SubscriptionPlanDTO{})
		return
	}

	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ? AND (plan_type = ? OR plan_type = ? OR plan_type = '')", true, model.SubscriptionPlanTypeUser, model.SubscriptionPlanTypeBoth).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubscriptions, // all active subscriptions
		"all_subscriptions":  allSubscriptions,    // all subscriptions including expired
	})
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	if err := model.UpdateUserSetting(user.Id, current); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

func SubscriptionRequestBalancePay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	userId := c.GetInt("id")
	var req SubscriptionBalancePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := model.PurchaseSubscriptionWithBalance(userId, req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func validateSubscriptionPlanSettings(plan *model.SubscriptionPlan) error {
	planType, err := model.NormalizeSubscriptionPlanType(plan.PlanType)
	if err != nil {
		return err
	}
	plan.PlanType = planType
	plan.QuotaResetTimezone = strings.TrimSpace(plan.QuotaResetTimezone)
	plan.TokenResetTimezone = strings.TrimSpace(plan.TokenResetTimezone)
	for _, anchor := range []*int64{plan.QuotaResetAnchor, plan.TokenResetAnchor} {
		if anchor != nil && (*anchor < -62135596800 || *anchor > 253402300799) {
			return errors.New("reset anchor is outside the supported Unix range")
		}
	}
	for _, timezone := range []string{plan.QuotaResetTimezone, plan.TokenResetTimezone} {
		if timezone != "" {
			if _, err := time.LoadLocation(timezone); err != nil {
				return fmt.Errorf("invalid reset timezone: %w", err)
			}
		}
	}
	return nil
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Plan.Id = 0
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.AllowBalancePay == nil {
		req.Plan.AllowBalancePay = common.GetPointer(true)
	}
	if req.Plan.AllowWalletOverflow == nil {
		req.Plan.AllowWalletOverflow = common.GetPointer(true)
	}
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	// Token quota reset period (independent; empty = same as QuotaResetPeriod)
	if strings.TrimSpace(req.Plan.TokenResetPeriod) != "" {
		req.Plan.TokenResetPeriod = model.NormalizeResetPeriod(req.Plan.TokenResetPeriod)
		if req.Plan.TokenResetPeriod == model.SubscriptionResetCustom && req.Plan.TokenResetCustomSeconds <= 0 {
			common.ApiErrorMsg(c, "自定义 token 重置周期需大于0秒")
			return
		}
	}
	if req.Plan.TotalTokens < 0 {
		common.ApiErrorMsg(c, "token 总额度不能为负数")
		return
	}
	if err := validateSubscriptionPlanSettings(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	err := model.DB.Create(&req.Plan).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(req.Plan.Id)
	common.ApiSuccess(c, req.Plan)
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	req.Plan.Id = id
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	// Token quota reset period (independent; empty = same as QuotaResetPeriod)
	tokenResetPeriod := ""
	if strings.TrimSpace(req.Plan.TokenResetPeriod) != "" {
		tokenResetPeriod = model.NormalizeResetPeriod(req.Plan.TokenResetPeriod)
		if tokenResetPeriod == model.SubscriptionResetCustom && req.Plan.TokenResetCustomSeconds <= 0 {
			common.ApiErrorMsg(c, "自定义 token 重置周期需大于0秒")
			return
		}
	}
	if req.Plan.TotalTokens < 0 {
		common.ApiErrorMsg(c, "token 总额度不能为负数")
		return
	}

	if err := validateSubscriptionPlanSettings(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	now := model.GetDBTimestamp()
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]interface{}{
			"title":                      req.Plan.Title,
			"subtitle":                   req.Plan.Subtitle,
			"price_amount":               req.Plan.PriceAmount,
			"currency":                   req.Plan.Currency,
			"duration_unit":              req.Plan.DurationUnit,
			"duration_value":             req.Plan.DurationValue,
			"custom_seconds":             req.Plan.CustomSeconds,
			"enabled":                    req.Plan.Enabled,
			"sort_order":                 req.Plan.SortOrder,
			"stripe_price_id":            req.Plan.StripePriceId,
			"creem_product_id":           req.Plan.CreemProductId,
			"waffo_pancake_product_id":   req.Plan.WaffoPancakeProductId,
			"max_purchase_per_user":      req.Plan.MaxPurchasePerUser,
			"total_amount":               req.Plan.TotalAmount,
			"upgrade_group":              req.Plan.UpgradeGroup,
			"downgrade_group":            req.Plan.DowngradeGroup,
			"quota_reset_period":         req.Plan.QuotaResetPeriod,
			"quota_reset_custom_seconds": req.Plan.QuotaResetCustomSeconds,
			"total_tokens":               req.Plan.TotalTokens,
			"include_cache_tokens":       req.Plan.IncludeCacheTokens,
			"token_reset_period":         tokenResetPeriod,
			"token_reset_custom_seconds": req.Plan.TokenResetCustomSeconds,
			"plan_type":                  req.Plan.PlanType,
			"quota_reset_anchor":         req.Plan.QuotaResetAnchor,
			"quota_reset_timezone":       req.Plan.QuotaResetTimezone,
			"token_reset_anchor":         req.Plan.TokenResetAnchor,
			"token_reset_timezone":       req.Plan.TokenResetTimezone,
			"updated_at":                 common.GetTimestamp(),
		}
		if req.Plan.AllowBalancePay != nil {
			updateMap["allow_balance_pay"] = *req.Plan.AllowBalancePay
		}
		if req.Plan.AllowWalletOverflow != nil {
			updateMap["allow_wallet_overflow"] = *req.Plan.AllowWalletOverflow
		}
		if err := tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
			return err
		}
		// include_cache_tokens is a billing rule, not a balance snapshot — keep
		// active user subscriptions in sync when the plan toggle changes so the
		// setting takes effect without rebinding every user.
		if err := tx.Model(&model.UserSubscription{}).
			Where("plan_id = ? AND status = ?", id, "active").
			Updates(map[string]interface{}{
				"include_cache_tokens": req.Plan.IncludeCacheTokens,
				"updated_at":           common.GetTimestamp(),
			}).Error; err != nil {
			return err
		}
		if err := model.RecalculatePlanResetTimesTx(tx, id, now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	subs, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subs)
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId             int     `json:"plan_id"`
	QuotaResetAnchor   *int64  `json:"quota_reset_anchor"`
	QuotaResetTimezone *string `json:"quota_reset_timezone"`
	TokenResetAnchor   *int64  `json:"token_reset_anchor"`
	TokenResetTimezone *string `json:"token_reset_timezone"`
}

type AdminUpdateUserSubscriptionRequest struct {
	QuotaResetAnchor   *int64  `json:"quota_reset_anchor"`
	QuotaResetTimezone *string `json:"quota_reset_timezone"`
	TokenResetAnchor   *int64  `json:"token_reset_anchor"`
	TokenResetTimezone *string `json:"token_reset_timezone"`
}

type AdminResetSubscriptionRequest struct {
	PlanId           int    `json:"plan_id"`
	ResetScope       string `json:"reset_scope"`
	AdvanceResetTime *bool  `json:"advance_reset_time"`
}

func requireResetScope(scope string) (string, error) {
	if strings.TrimSpace(scope) == "" {
		return "", errors.New("reset_scope is required")
	}
	return model.NormalizeResetScope(scope)
}

func resetAdvanceFlag(req AdminResetSubscriptionRequest) bool {
	// Default false: manual usage reset stamps last_reset_time without rolling
	// the billing cycle unless the admin explicitly opts in.
	if req.AdvanceResetTime == nil {
		return false
	}
	return *req.AdvanceResetTime
}

func recordSubscriptionResetUserLogs(result *model.SubscriptionResetResult, adminInfo map[string]interface{}) {
	if result == nil || result.ResetCount == 0 {
		return
	}
	content := fmt.Sprintf("管理员重置订阅套餐 %s（ID: %d）额度", result.PlanTitle, result.PlanId)
	for _, userId := range result.AffectedUserIds {
		model.RecordLogWithAdminInfo(userId, model.LogTypeManage, content, adminInfo)
	}
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	override := &model.UserSubscriptionResetOverride{
		QuotaResetAnchor:   req.QuotaResetAnchor,
		QuotaResetTimezone: req.QuotaResetTimezone,
		TokenResetAnchor:   req.TokenResetAnchor,
		TokenResetTimezone: req.TokenResetTimezone,
	}
	if override.QuotaResetAnchor == nil && override.QuotaResetTimezone == nil &&
		override.TokenResetAnchor == nil && override.TokenResetTimezone == nil {
		override = nil
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "", override)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminUpdateUserSubscription updates optional instance-level reset anchor overrides.
func AdminUpdateUserSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	var req AdminUpdateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.AdminUpdateUserSubscriptionResetOverrides(subId, model.UserSubscriptionResetOverride{
		QuotaResetAnchor:   req.QuotaResetAnchor,
		QuotaResetTimezone: req.QuotaResetTimezone,
		TokenResetAnchor:   req.TokenResetAnchor,
		TokenResetTimezone: req.TokenResetTimezone,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.user_subscription_update_reset_override", map[string]interface{}{
		"user_subscription_id": subId,
	})
	common.ApiSuccess(c, nil)
}

func AdminResetUserSubscriptionsByPlan(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	resetScope, err := requireResetScope(req.ResetScope)
	if err != nil {
		common.ApiErrorMsg(c, "重置范围必须为 quota、tokens 或 both")
		return
	}
	result, err := model.AdminResetUserSubscriptionsByPlan(userId, req.PlanId, resetAdvanceFlag(req), resetScope)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(result, auditOperatorInfo(c))
	recordManageAuditFor(c, userId, "subscription.user_plan_reset", map[string]interface{}{
		"target_user_id": userId,
		"plan_id":        result.PlanId,
		"plan_title":     result.PlanTitle,
		"reset_count":    result.ResetCount,
		"user_count":     result.UserCount,
		"reset_scope":    result.ResetScope,
	})
	common.ApiSuccess(c, result)
}

func AdminResetPlanSubscriptions(c *gin.Context) {
	planId, _ := strconv.Atoi(c.Param("id"))
	if planId <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	resetScope, err := requireResetScope(req.ResetScope)
	if err != nil {
		common.ApiErrorMsg(c, "重置范围必须为 quota、tokens 或 both")
		return
	}
	result, err := model.AdminResetPlanSubscriptions(planId, resetAdvanceFlag(req), resetScope)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(result, auditOperatorInfo(c))
	common.SysLog(fmt.Sprintf("admin reset subscription plan %d scope=%s: reset_count=%d user_count=%d",
		result.PlanId, result.ResetScope, result.ResetCount, result.UserCount))
	recordManageAudit(c, "subscription.plan_reset", map[string]interface{}{
		"plan_id":     result.PlanId,
		"plan_title":  result.PlanTitle,
		"reset_count": result.ResetCount,
		"user_count":  result.UserCount,
		"reset_scope": result.ResetScope,
	})
	common.ApiSuccess(c, result)
}

// AdminInvalidateUserSubscription cancels a user subscription immediately.
func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminDeleteUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: all-subscriptions listing & per-subscription management ----

// AdminListAllSubscriptions lists all user subscriptions joined with the owning
// username and plan title, with pagination and optional username / status filters.
// Query: username (fuzzy), status (active|expired|cancelled).
func AdminListAllSubscriptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	username := c.Query("username")
	status := c.Query("status")
	details, total, err := model.AdminListAllUserSubscriptions(pageInfo.GetPage(), pageInfo.GetPageSize(), username, status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(details)
	common.ApiSuccess(c, pageInfo)
}

type AdminAdjustSubscriptionRequest struct {
	AmountDelta int64 `json:"amount_delta"`
	TokenDelta  int64 `json:"token_delta"`
}

// AdminAdjustSubscription adds quota (amount and/or tokens) to a single user
// subscription.
func AdminAdjustSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	var req AdminAdjustSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.AmountDelta < 0 || req.TokenDelta < 0 {
		common.ApiErrorMsg(c, "额度调整值不能为负数")
		return
	}
	if err := model.AdminAdjustUserSubscription(subId, req.AmountDelta, req.TokenDelta); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.adjust", map[string]interface{}{
		"user_subscription_id": subId,
		"amount_delta":         req.AmountDelta,
		"token_delta":          req.TokenDelta,
	})
	common.ApiSuccess(c, nil)
}

// AdminResetSingleSubscription resets a single user subscription's usage and
// recalculates its next reset times.
func AdminResetSingleSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	resetScope, err := requireResetScope(req.ResetScope)
	if err != nil {
		common.ApiErrorMsg(c, "重置范围必须为 quota、tokens 或 both")
		return
	}
	if err := model.AdminResetSingleUserSubscription(subId, resetScope, resetAdvanceFlag(req)); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.single_reset", map[string]interface{}{
		"user_subscription_id": subId,
	})
	common.ApiSuccess(c, nil)
}
