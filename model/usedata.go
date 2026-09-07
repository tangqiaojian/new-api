package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id               int    `json:"id"`
	UserID           int    `json:"user_id" gorm:"index"`
	Username         string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName        string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	UseGroup         string `json:"use_group" gorm:"index;size:64;default:''"`
	TokenID          int    `json:"token_id" gorm:"index;default:0"`
	ChannelID        int    `json:"channel_id" gorm:"index;default:0"`
	NodeName         string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed        int    `json:"token_used" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	CacheReadTokens  int    `json:"cache_read_tokens" gorm:"default:0"`
	CacheWriteTokens int    `json:"cache_write_tokens" gorm:"default:0"`
	SuccessCount     int    `json:"success_count" gorm:"default:0"`
	ErrorCount       int    `json:"error_count" gorm:"default:0"`
	Count            int    `json:"count" gorm:"default:0"`
	Quota            int    `json:"quota" gorm:"default:0"`
}

type QuotaDataLogParams struct {
	UserID           int
	Username         string
	ModelName        string
	Quota            int
	CreatedAt        int64
	TokenUsed        int
	PromptTokens     int
	CompletionTokens int
	CacheReadTokens  int
	CacheWriteTokens int
	Success          bool
	UseGroup         string
	TokenID          int
	ChannelID        int
	NodeName         string
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%s\x00%d\x00%d\x00%s",
		quotaData.UserID,
		quotaData.Username,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.TokenID,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	count := quotaData.Count
	quota := quotaData.Quota
	tokenUsed := quotaData.TokenUsed
	promptTokens := quotaData.PromptTokens
	completionTokens := quotaData.CompletionTokens
	cacheReadTokens := quotaData.CacheReadTokens
	cacheWriteTokens := quotaData.CacheWriteTokens
	successCount := quotaData.SuccessCount
	errorCount := quotaData.ErrorCount
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		cachedQuotaData.Count += count
		cachedQuotaData.Quota += quota
		cachedQuotaData.TokenUsed += tokenUsed
		cachedQuotaData.PromptTokens += promptTokens
		cachedQuotaData.CompletionTokens += completionTokens
		cachedQuotaData.CacheReadTokens += cacheReadTokens
		cachedQuotaData.CacheWriteTokens += cacheWriteTokens
		cachedQuotaData.SuccessCount += successCount
		cachedQuotaData.ErrorCount += errorCount
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	successCount := 0
	errorCount := 0
	if params.Success {
		successCount = 1
	} else {
		errorCount = 1
	}
	quotaData := &QuotaData{
		UserID:           params.UserID,
		Username:         params.Username,
		ModelName:        params.ModelName,
		CreatedAt:        createdAt,
		UseGroup:         params.UseGroup,
		TokenID:          params.TokenID,
		ChannelID:        params.ChannelID,
		NodeName:         params.NodeName,
		Count:            1,
		Quota:            params.Quota,
		TokenUsed:        params.TokenUsed,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		CacheReadTokens:  params.CacheReadTokens,
		CacheWriteTokens: params.CacheWriteTokens,
		SuccessCount:     successCount,
		ErrorCount:       errorCount,
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":             gorm.Expr("count + ?", quotaData.Count),
			"quota":             gorm.Expr("quota + ?", quotaData.Quota),
			"token_used":        gorm.Expr("token_used + ?", quotaData.TokenUsed),
			"prompt_tokens":     gorm.Expr("prompt_tokens + ?", quotaData.PromptTokens),
			"completion_tokens": gorm.Expr("completion_tokens + ?", quotaData.CompletionTokens),
			"cache_read_tokens": gorm.Expr("cache_read_tokens + ?", quotaData.CacheReadTokens),
			"cache_write_tokens": gorm.Expr("cache_write_tokens + ?", quotaData.CacheWriteTokens),
			"success_count":     gorm.Expr("success_count + ?", quotaData.SuccessCount),
			"error_count":       gorm.Expr("error_count + ?", quotaData.ErrorCount),
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func quotaTokenUsedSumExpression(includeCache bool) string {
	expression := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		expression += " + " + logCacheTokensSumExpr()
	}
	return expression
}

func logCacheWriteTokensPerRowExpr() string {
	writeExpr := logJSONIntExpression("cache_write_tokens")
	creationExpr := logJSONIntExpression("cache_creation_tokens")
	// Prefer cache_write_tokens; fall back to cache_creation_tokens for older rows.
	return "CASE WHEN COALESCE(" + writeExpr + ", 0) > 0 THEN " + writeExpr + " ELSE COALESCE(" + creationExpr + ", 0) END"
}

func quotaDataAggregateSelect(includeCache bool) string {
	cacheReadExpr := "COALESCE(SUM(" + logJSONIntExpression("cache_tokens") + "), 0)"
	cacheWriteExpr := "COALESCE(SUM(" + logCacheWriteTokensPerRowExpr() + "), 0)"
	successExpr := "COALESCE(SUM(" + logStreamSuccessExpression() + "), 0)"
	return "COUNT(*) AS count, " +
		"COALESCE(SUM(quota), 0) AS quota, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		cacheReadExpr + " AS cache_read_tokens, " +
		cacheWriteExpr + " AS cache_write_tokens, " +
		successExpr + " AS success_count, " +
		"COUNT(*) - " + successExpr + " AS error_count, " +
		quotaTokenUsedSumExpression(includeCache) + " AS token_used"
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64, includeCache bool) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = LOG_DB.Table("logs").
		Select("user_id, username, model_name, created_at - (created_at % 3600) AS created_at, "+quotaDataAggregateSelect(includeCache)).
		Where("username = ? AND type = ? AND created_at >= ? AND created_at <= ?", username, LogTypeConsume, startTime, endTime).
		Group("user_id, username, model_name, created_at - (created_at % 3600)").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64, includeCache bool) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = LOG_DB.Table("logs").
		Select("user_id, username, model_name, created_at - (created_at % 3600) AS created_at, "+quotaDataAggregateSelect(includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Group("user_id, username, model_name, created_at - (created_at % 3600)").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64, includeCache bool) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = LOG_DB.Table("logs").
		Select("username, created_at - (created_at % 3600) AS created_at, "+quotaDataAggregateSelect(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Group("username, created_at - (created_at % 3600)").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string, includeCache bool) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime, includeCache)
	}
	var quotaDatas []*QuotaData
	err = LOG_DB.Table("logs").
		Select("model_name, created_at - (created_at % 3600) AS created_at, "+quotaDataAggregateSelect(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Group("model_name, created_at - (created_at % 3600)").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

// GetQuotaDataGroupByUseGroup aggregates consume logs by billing group for
// Sub2API-style group/channel mini cards (today / range totals).
func GetQuotaDataGroupByUseGroup(startTime int64, endTime int64, username string, userId int, includeCache bool) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	groupExpr := logGroupCol + " AS use_group"
	tx := LOG_DB.Table("logs").
		Select(groupExpr+", "+quotaDataAggregateSelect(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	// Group by the selected alias so GORM does not double-quote the reserved
	// `group` column name (breaks SQLite).
	err = tx.Group("use_group").Find(&quotaDatas).Error
	return quotaDatas, err
}
