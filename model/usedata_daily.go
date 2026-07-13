package model

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
)

// cacheKeyIntStr 构造分组 key（int|string 格式）
func cacheKeyIntStr(id int, date string) string {
	return strconv.Itoa(id) + "|" + date
}

// cacheKeyStrStr 构造分组 key（string|string 格式）
func cacheKeyStrStr(name, date string) string {
	return name + "|" + date
}

// DailyTokenData represents daily token usage statistics per user
type DailyTokenData struct {
	UserID           int    `json:"user_id"`
	Username         string `json:"username"`
	Date             string `json:"date"` // YYYY-MM-DD format
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// DailyModelTokenData represents daily token usage statistics per model
type DailyModelTokenData struct {
	ModelName        string `json:"model_name"`
	Date             string `json:"date"` // YYYY-MM-DD format
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// dailyTokenDateExpression returns the SQL expression that extracts a
// YYYY-MM-DD date string from the logs.created_at unix timestamp, for the
// dialect currently used to store logs (SQLite, MySQL, PostgreSQL, or
// ClickHouse).
func dailyTokenDateExpression() string {
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		// ClickHouse: toDate(created_at) -> 'YYYY-MM-DD'
		return "toString(toDate(created_at))"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD')"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "strftime('%Y-%m-%d', created_at, 'unixepoch')"
	default:
		// MySQL
		return "DATE(FROM_UNIXTIME(created_at))"
	}
}

// dailyTokenSelectColumns builds the SELECT clause for daily token aggregation.
// The date column is aliased to "date" so it maps to DailyTokenData.Date.
// When includeCache is true, total_tokens adds cached_tokens (extracted from the
// `other` JSON) on top of prompt_tokens + completion_tokens.
func dailyTokenSelectColumns(dateExpr string, includeCache bool) string {
	totalTokensExpr := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		totalTokensExpr += " + " + logCacheTokensSumExpr()
	}
	return "user_id, username, " + dateExpr + " as date, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
		totalTokensExpr + " as total_tokens, " +
		logCacheTokensSumExpr() + " as cached_tokens, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(quota), 0) as quota"
}

// dailyModelTokenSelectColumns builds the SELECT clause for daily model token aggregation.
// When includeCache is true, total_tokens adds cached_tokens (extracted from the
// `other` JSON) on top of prompt_tokens + completion_tokens.
func dailyModelTokenSelectColumns(dateExpr string, includeCache bool) string {
	totalTokensExpr := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		totalTokensExpr += " + " + logCacheTokensSumExpr()
	}
	return "model_name, " + dateExpr + " as date, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
		totalTokensExpr + " as total_tokens, " +
		logCacheTokensSumExpr() + " as cached_tokens, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(quota), 0) as quota"
}

// fillDailyTokenCacheTokens 为每日 token 统计数据填充 cache_tokens（从 other JSON 提取）。
// keyFn 返回每条日志的分组 key（如 "userId|date"），用于匹配 data 中的条目。
func fillDailyTokenCacheTokens(data []*DailyTokenData, startTime, endTime int64, whereClauses []string, whereArgs []interface{}) {
	if len(data) == 0 {
		return
	}
	dateExpr := dailyTokenDateExpression()
	query := LOG_DB.Table("logs").
		Select("user_id, username, " + dateExpr + " as date, other").
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)
	for i := 0; i < len(whereClauses); i++ {
		query = query.Where(whereClauses[i], whereArgs[i])
	}

	var rows []struct {
		UserID   int    `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
		Date     string `gorm:"column:date"`
		Other    string `gorm:"column:other"`
	}
	query.Find(&rows)

	cacheMap := make(map[string]int) // "userId|date" -> cachedTokens
	for _, r := range rows {
		if r.Other == "" {
			continue
		}
		_, cacheTokens := parseOtherForStats(r.Other)
		key := cacheKeyIntStr(r.UserID, r.Date)
		cacheMap[key] += cacheTokens
	}

	for _, d := range data {
		key := cacheKeyIntStr(d.UserID, d.Date)
		d.CachedTokens = cacheMap[key]
	}
}

// fillDailyModelTokenCacheTokens 为每日模型 token 统计数据填充 cache_tokens。
func fillDailyModelTokenCacheTokens(data []*DailyModelTokenData, startTime, endTime int64, whereClauses []string, whereArgs []interface{}) {
	if len(data) == 0 {
		return
	}
	dateExpr := dailyTokenDateExpression()
	query := LOG_DB.Table("logs").
		Select("model_name, " + dateExpr + " as date, other").
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)
	for i := 0; i < len(whereClauses); i++ {
		query = query.Where(whereClauses[i], whereArgs[i])
	}

	var rows []struct {
		ModelName string `gorm:"column:model_name"`
		Date      string `gorm:"column:date"`
		Other     string `gorm:"column:other"`
	}
	query.Find(&rows)

	cacheMap := make(map[string]int) // "model|date" -> cachedTokens
	for _, r := range rows {
		if r.Other == "" {
			continue
		}
		_, cacheTokens := parseOtherForStats(r.Other)
		key := cacheKeyStrStr(r.ModelName, r.Date)
		cacheMap[key] += cacheTokens
	}

	for _, d := range data {
		key := cacheKeyStrStr(d.ModelName, d.Date)
		d.CachedTokens = cacheMap[key]
	}
}

// applyExcludeCache 从 token 统计里扣除缓存 token（不含缓存模式），并清零缓存字段
func applyExcludeCacheDailyToken(data []*DailyTokenData) {
	for _, d := range data {
		d.PromptTokens -= d.CachedTokens
		d.TotalTokens -= d.CachedTokens
		d.CachedTokens = 0
	}
}

func applyExcludeCacheDailyModelToken(data []*DailyModelTokenData) {
	for _, d := range data {
		d.PromptTokens -= d.CachedTokens
		d.TotalTokens -= d.CachedTokens
		d.CachedTokens = 0
	}
}

// GetDailyTokenDataByUserId returns daily token usage for a specific user.
// When includeCache is true, cached_tokens are included in total_tokens.
func GetDailyTokenDataByUserId(userId int, startTime int64, endTime int64, includeCache bool) ([]*DailyTokenData, error) {
	var data []*DailyTokenData
	dateExpr := dailyTokenDateExpression()

	err := LOG_DB.Table("logs").
		Select(dailyTokenSelectColumns(dateExpr, includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Group("user_id, username, " + dateExpr).
		Order("date DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	fillDailyTokenCacheTokens(data, startTime, endTime,
		[]string{"user_id = ?"}, []interface{}{userId})

	if !includeCache {
		applyExcludeCacheDailyToken(data)
	}

	return data, nil
}

// GetAllDailyTokenData returns daily token usage for all users (admin only).
// When username is non-empty, results are filtered to that user.
// When includeCache is true, cached_tokens are included in total_tokens.
func GetAllDailyTokenData(startTime int64, endTime int64, username string, includeCache bool) ([]*DailyTokenData, error) {
	var data []*DailyTokenData
	dateExpr := dailyTokenDateExpression()

	query := LOG_DB.Table("logs").
		Select(dailyTokenSelectColumns(dateExpr, includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)

	if username != "" {
		query = query.Where("username = ?", username)
	}

	err := query.
		Group("user_id, username, " + dateExpr).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	whereClauses := []string{}
	whereArgs := []interface{}{}
	if username != "" {
		whereClauses = append(whereClauses, "username = ?")
		whereArgs = append(whereArgs, username)
	}
	fillDailyTokenCacheTokens(data, startTime, endTime, whereClauses, whereArgs)

	if !includeCache {
		applyExcludeCacheDailyToken(data)
	}

	return data, nil
}

// GetDailyModelTokenData returns daily token usage grouped by model for a specific user.
// When includeCache is true, cached_tokens are included in total_tokens.
func GetDailyModelTokenDataByUserId(userId int, startTime int64, endTime int64, includeCache bool) ([]*DailyModelTokenData, error) {
	var data []*DailyModelTokenData
	dateExpr := dailyTokenDateExpression()

	err := LOG_DB.Table("logs").
		Select(dailyModelTokenSelectColumns(dateExpr, includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Group("model_name, " + dateExpr).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	fillDailyModelTokenCacheTokens(data, startTime, endTime,
		[]string{"user_id = ?"}, []interface{}{userId})

	if !includeCache {
		applyExcludeCacheDailyModelToken(data)
	}

	return data, nil
}

// GetAllDailyModelTokenData returns daily token usage grouped by model for all users (admin only).
// When includeCache is true, cached_tokens are included in total_tokens.
func GetAllDailyModelTokenData(startTime int64, endTime int64, includeCache bool) ([]*DailyModelTokenData, error) {
	var data []*DailyModelTokenData
	dateExpr := dailyTokenDateExpression()

	err := LOG_DB.Table("logs").
		Select(dailyModelTokenSelectColumns(dateExpr, includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Group("model_name, " + dateExpr).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	fillDailyModelTokenCacheTokens(data, startTime, endTime, nil, nil)

	if !includeCache {
		applyExcludeCacheDailyModelToken(data)
	}

	return data, nil
}
