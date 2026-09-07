package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func parseFlowQuotaTimeRange(c *gin.Context) (int64, int64, bool) {
	startTimestamp, err := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	if err != nil || startTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid start_timestamp")
		return 0, 0, false
	}
	endTimestamp, err := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if err != nil || endTimestamp <= 0 {
		common.ApiErrorMsg(c, "invalid end_timestamp")
		return 0, 0, false
	}
	if endTimestamp < startTimestamp {
		common.ApiErrorMsg(c, "invalid time range")
		return 0, 0, false
	}
	return startTimestamp, endTimestamp, true
}

func GetAllQuotaDates(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	includeCache, _ := strconv.ParseBool(c.Query("include_cache"))
	dates, err := model.GetAllQuotaDates(startTimestamp, endTimestamp, username, includeCache)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetQuotaDatesByUser(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	includeCache, _ := strconv.ParseBool(c.Query("include_cache"))
	dates, err := model.GetQuotaDataGroupByUser(startTimestamp, endTimestamp, includeCache)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetQuotaDatesByGroup(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	includeCache, _ := strconv.ParseBool(c.Query("include_cache"))
	username := c.Query("username")
	dates, err := model.GetQuotaDataGroupByUseGroup(startTimestamp, endTimestamp, username, 0, includeCache)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetUserQuotaDatesByGroup(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	includeCache, _ := strconv.ParseBool(c.Query("include_cache"))
	dates, err := model.GetQuotaDataGroupByUseGroup(startTimestamp, endTimestamp, "", userId, includeCache)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
}

func GetUserQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 判断时间跨度是否超过 1 个月
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	includeCache, _ := strconv.ParseBool(c.Query("include_cache"))
	dates, err := model.GetQuotaDataByUserId(userId, startTimestamp, endTimestamp, includeCache)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetAllFlowQuotaDates(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	username := c.Query("username")
	dates, err := model.GetFlowQuotaData(startTimestamp, endTimestamp, username, 0, c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetUserFlowQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	dates, err := model.GetFlowQuotaData(startTimestamp, endTimestamp, "", userId, common.RoleCommonUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
	})
	return
}

func GetAllDailyTokenData(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	data, err := model.GetAllDailyTokenData(
		startTimestamp,
		endTimestamp,
		c.Query("username"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetUserDailyTokenData(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	data, err := model.GetDailyTokenDataByUserId(
		c.GetInt("id"),
		startTimestamp,
		endTimestamp,
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetAllDailyModelTokenData(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	data, err := model.GetAllDailyModelTokenData(
		startTimestamp,
		endTimestamp,
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetUserDailyModelTokenData(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	data, err := model.GetDailyModelTokenDataByUserId(
		c.GetInt("id"),
		startTimestamp,
		endTimestamp,
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetAllChannelModelStats(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	data, err := model.GetChannelModelStats(
		startTimestamp,
		endTimestamp,
		c.Query("username"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetSelfChannelModelStats(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	data, err := model.GetSelfChannelModelStats(
		c.GetInt("id"),
		startTimestamp,
		endTimestamp,
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetSelfSubscriptionUsage(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	subscriptionID, _ := strconv.Atoi(c.Query("subscription_id"))
	data, err := model.GetSelfSubscriptionDailyUsage(
		c.GetInt("id"),
		startTimestamp,
		endTimestamp,
		subscriptionID,
		c.Query("model"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetAllSubscriptionUsage(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	subscriptionID, _ := strconv.Atoi(c.Query("subscription_id"))
	data, err := model.GetSubscriptionDailyUsage(
		0,
		startTimestamp,
		endTimestamp,
		subscriptionID,
		c.Query("model"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetSelfSubscriptionModelUsage(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	subscriptionID, _ := strconv.Atoi(c.Query("subscription_id"))
	data, err := model.GetSelfSubscriptionModelUsage(
		c.GetInt("id"),
		startTimestamp,
		endTimestamp,
		subscriptionID,
		c.Query("model"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetAllSubscriptionModelUsage(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseFlowQuotaTimeRange(c)
	if !ok {
		return
	}
	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}
	subscriptionID, _ := strconv.Atoi(c.Query("subscription_id"))
	data, err := model.GetSubscriptionModelUsage(
		0,
		startTimestamp,
		endTimestamp,
		subscriptionID,
		c.Query("model"),
		c.Query("include_cache") != "false",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}
