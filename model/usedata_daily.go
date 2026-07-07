package model

import (
	"github.com/QuantumNous/new-api/common"
)

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

	return data, err
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

	return data, err
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

	return data, err
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

	return data, err
}
