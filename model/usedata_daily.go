package model

import "github.com/QuantumNous/new-api/common"

// DailyTokenData represents daily token usage statistics per user.
type DailyTokenData struct {
	UserID           int    `json:"user_id"`
	Username         string `json:"username"`
	Date             string `json:"date"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// DailyModelTokenData represents daily token usage statistics per model.
type DailyModelTokenData struct {
	ModelName        string `json:"model_name"`
	Date             string `json:"date"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

func dailyTokenDateExpression() string {
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "toString(toDate(fromUnixTimestamp(created_at, 'UTC')))"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "TO_CHAR(TO_TIMESTAMP(created_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD')"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "strftime('%Y-%m-%d', created_at, 'unixepoch')"
	default:
		// Adding Unix seconds to a DATETIME literal avoids FROM_UNIXTIME's
		// dependence on the MySQL session time zone.
		return "DATE_FORMAT(DATE_ADD('1970-01-01 00:00:00', INTERVAL created_at SECOND), '%Y-%m-%d')"
	}
}

func dailyTotalTokensExpression(includeCache bool) string {
	expression := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		expression += " + " + logCacheTokensSumExpr()
	}
	return expression
}

func dailyTokenSelectColumns(dateExpression string, includeCache bool) string {
	return "user_id, username, " + dateExpression + " AS date, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		dailyTotalTokensExpression(includeCache) + " AS total_tokens, " +
		logCacheTokensSumExpr() + " AS cached_tokens, " +
		"COUNT(*) AS request_count, " +
		"COALESCE(SUM(quota), 0) AS quota"
}

func dailyModelTokenSelectColumns(dateExpression string, includeCache bool) string {
	return "model_name, " + dateExpression + " AS date, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		dailyTotalTokensExpression(includeCache) + " AS total_tokens, " +
		logCacheTokensSumExpr() + " AS cached_tokens, " +
		"COUNT(*) AS request_count, " +
		"COALESCE(SUM(quota), 0) AS quota"
}

// GetDailyTokenDataByUserId returns daily usage for one user.
func GetDailyTokenDataByUserId(userID int, startTime int64, endTime int64, includeCache bool) ([]*DailyTokenData, error) {
	data := make([]*DailyTokenData, 0)
	dateExpression := dailyTokenDateExpression()
	err := LOG_DB.Table("logs").
		Select(dailyTokenSelectColumns(dateExpression, includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userID, LogTypeConsume, startTime, endTime).
		Group("user_id, username, " + dateExpression).
		Order("date DESC").
		Find(&data).Error
	return data, err
}

// GetAllDailyTokenData returns daily usage for all users, optionally filtered by username.
func GetAllDailyTokenData(startTime int64, endTime int64, username string, includeCache bool) ([]*DailyTokenData, error) {
	data := make([]*DailyTokenData, 0)
	dateExpression := dailyTokenDateExpression()
	query := LOG_DB.Table("logs").
		Select(dailyTokenSelectColumns(dateExpression, includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)
	if username != "" {
		query = query.Where("username = ?", username)
	}
	err := query.
		Group("user_id, username, " + dateExpression).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	return data, err
}

// GetDailyModelTokenDataByUserId returns daily usage grouped by model for one user.
func GetDailyModelTokenDataByUserId(userID int, startTime int64, endTime int64, includeCache bool) ([]*DailyModelTokenData, error) {
	data := make([]*DailyModelTokenData, 0)
	dateExpression := dailyTokenDateExpression()
	err := LOG_DB.Table("logs").
		Select(dailyModelTokenSelectColumns(dateExpression, includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userID, LogTypeConsume, startTime, endTime).
		Group("model_name, " + dateExpression).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	return data, err
}

// GetAllDailyModelTokenData returns daily usage grouped by model for all users.
func GetAllDailyModelTokenData(startTime int64, endTime int64, includeCache bool) ([]*DailyModelTokenData, error) {
	data := make([]*DailyModelTokenData, 0)
	dateExpression := dailyTokenDateExpression()
	err := LOG_DB.Table("logs").
		Select(dailyModelTokenSelectColumns(dateExpression, includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Group("model_name, " + dateExpression).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	return data, err
}
