package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type resetChannelSubscriptionPoolRequest struct {
	Scope string `json:"scope"`
}

func AdminCreateChannelSubscriptionPool(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req model.CreateChannelSubscriptionPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pool, err := service.ChannelPools.Create(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.channel_pool_create", map[string]interface{}{"pool_id": pool.Id, "plan_id": pool.PlanId, "channel_count": len(pool.Channels)})
	common.ApiSuccess(c, pool)
}

func AdminListChannelSubscriptionPools(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	pools, total, err := service.ChannelPools.List(pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(pools)
	common.ApiSuccess(c, pageInfo)
}

func AdminListChannelPoolOccupancies(c *gin.Context) {
	items, err := model.ListActiveChannelPoolOccupancies()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func AdminGetChannelSubscriptionPool(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的订阅池ID")
		return
	}
	pool, err := service.ChannelPools.Get(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, pool)
}

func AdminUpdateChannelSubscriptionPool(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的订阅池ID")
		return
	}
	var req model.UpdateChannelSubscriptionPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pool, err := service.ChannelPools.Update(id, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.channel_pool_update", map[string]interface{}{"pool_id": id, "channel_count": len(pool.Channels)})
	common.ApiSuccess(c, pool)
}

func AdminCancelChannelSubscriptionPool(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的订阅池ID")
		return
	}
	if err := service.ChannelPools.Cancel(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.channel_pool_cancel", map[string]interface{}{"pool_id": id})
	common.ApiSuccess(c, nil)
}

func AdminResetChannelSubscriptionPool(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的订阅池ID")
		return
	}
	var req resetChannelSubscriptionPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Scope == "" {
		common.ApiErrorMsg(c, "重置范围必须为 quota、tokens 或 both")
		return
	}
	scope, err := model.NormalizeResetScope(req.Scope)
	if err != nil {
		common.ApiErrorMsg(c, "重置范围必须为 quota、tokens 或 both")
		return
	}
	if err := service.ChannelPools.Reset(id, scope); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.channel_pool_reset", map[string]interface{}{"pool_id": id, "scope": scope})
	common.ApiSuccess(c, nil)
}
