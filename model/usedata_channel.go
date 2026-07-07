package model

import (
	"github.com/QuantumNous/new-api/common"
)

// ChannelModelStats represents aggregated statistics per channel + model combination
// from the logs table, grouped by channel_id and model_name.
type ChannelModelStats struct {
	ChannelID        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	ModelName        string  `json:"model_name"`
	RequestCount     int     `json:"request_count"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	AvgFirstByteMs  float64 `json:"avg_first_byte_ms"`
	AvgSpeedTokPerS float64 `json:"avg_speed_tok_per_s"`
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	SuccessRate     float64 `json:"success_rate"`
	TotalTokens     int     `json:"total_tokens"`
	Quota           int     `json:"quota"`
}

// channelStatsOtherCol returns the quoted column name for `other` based on log database dialect.
func channelStatsOtherCol() string {
	if common.UsingLogDatabase(common.DatabaseTypePostgreSQL) {
		return `"other"`
	}
	return "`other`"
}

// LogOtherCol returns the quoted column name for the logs `other` column based on
// the log database dialect. It is the exported form of channelStatsOtherCol, reused
// by all log-table aggregations that need to extract fields from the `other` JSON.
func LogOtherCol() string {
	return channelStatsOtherCol()
}

// channelStatsJsonExtractInt builds a dialect-specific SQL expression to extract an
// integer value from the JSON `other` column at the given key path.
func channelStatsJsonExtractInt(key string) string {
	otherCol := channelStatsOtherCol()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		// ClickHouse: JSONExtractInt extracts numeric values; cache_tokens is an integer
		return "JSONExtractInt(" + otherCol + ", '" + key + "')"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		// PostgreSQL: other is text, must cast to jsonb first
		return "CAST((" + otherCol + "::jsonb)->>'" + key + "' AS INTEGER)"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		// SQLite: CAST(json_extract(other, '$.key') AS INTEGER)
		return "CAST(json_extract(" + otherCol + ", '$." + key + "') AS INTEGER)"
	default:
		// MySQL: CAST(JSON_EXTRACT(other, '$.key') AS SIGNED)
		return "CAST(JSON_EXTRACT(" + otherCol + ", '$." + key + "') AS SIGNED)"
	}
}

// LogJsonExtractInt is the exported form of channelStatsJsonExtractInt. It builds a
// dialect-specific SQL expression to extract an integer value from the logs `other`
// JSON column at the given key. Reused by daily/model/channel/log aggregations.
func LogJsonExtractInt(key string) string {
	return channelStatsJsonExtractInt(key)
}

// logCacheTokensSumExpr returns a SQL expression that sums cache_tokens extracted
// from the `other` JSON column, wrapped in COALESCE so missing keys yield 0.
func logCacheTokensSumExpr() string {
	return "COALESCE(SUM(" + channelStatsJsonExtractInt("cache_tokens") + "), 0)"
}

// channelStatsJsonExtractFloat builds a dialect-specific SQL expression to extract a
// float/double value from the JSON `other` column at the given key path.
func channelStatsJsonExtractFloat(key string) string {
	otherCol := channelStatsOtherCol()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		// ClickHouse: JSONExtractFloat extracts numeric values; frt is a float64
		return "JSONExtractFloat(" + otherCol + ", '" + key + "')"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		// PostgreSQL: other is text, must cast to jsonb first
		return "CAST((" + otherCol + "::jsonb)->>'" + key + "' AS DOUBLE PRECISION)"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		// SQLite: CAST(json_extract(other, '$.key') AS REAL)
		return "CAST(json_extract(" + otherCol + ", '$." + key + "') AS REAL)"
	default:
		// MySQL: CAST(JSON_EXTRACT(other, '$.key') AS DECIMAL(20, 4))
		return "CAST(JSON_EXTRACT(" + otherCol + ", '$." + key + "') AS DECIMAL(20, 4))"
	}
}

// channelStatsBoolExpr builds a dialect-specific SQL expression that evaluates to 1
// when the stream_status in `other` indicates success, and 0 otherwise.
// The `other` JSON field contains an optional `stream_status.status` key which is "ok"
// for normal completions and "error" for failures (only present for streaming requests).
// For non-stream requests, success is always 1 (no stream_status field means success).
func channelStatsSuccessExpr() string {
	otherCol := channelStatsOtherCol()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		// ClickHouse: status == 'ok' or field not present (non-stream)
		return "CASE WHEN JSONExtractString(" + otherCol + ", 'stream_status', 'status') = 'error' THEN 0 ELSE 1 END"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		// PostgreSQL: other is text, must cast to jsonb for JSON path operators
		return "CASE WHEN (" + otherCol + "::jsonb)->'stream_status'->>'status' = 'error' THEN 0 ELSE 1 END"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		// SQLite: json_extract(other, '$.stream_status.status') = 'error' → failure
		return "CASE WHEN json_extract(" + otherCol + ", '$.stream_status.status') = 'error' THEN 0 ELSE 1 END"
	default:
		// MySQL: JSON_EXTRACT(other, '$.stream_status.status') = 'error' → failure
		return "CASE WHEN JSON_EXTRACT(" + otherCol + ", '$.stream_status.status') = 'error' THEN 0 ELSE 1 END"
	}
}

// channelStatsUseTimeCol returns the quoted column name for `use_time` based on
// log database dialect.
func channelStatsUseTimeCol() string {
	if common.UsingLogDatabase(common.DatabaseTypePostgreSQL) {
		return `"use_time"`
	}
	return "`use_time`"
}

// channelStatsSelectColumns builds the SELECT clause for channel+model aggregation
// from the logs table. It extracts key metrics from the `other` JSON column.
//
// Grouped columns: channel_id, model_name
// Aggregated: request_count, prompt_tokens, completion_tokens, cached_tokens,
//
//	first_byte_ms (avg), speed (avg), cache_hit_ratio, success_rate,
//	quota, total_tokens
//
// When includeCache is true, total_tokens adds cached_tokens on top of
// prompt_tokens + completion_tokens, matching the upstream dashboard's view.
func channelStatsSelectColumns(includeCache bool) string {
	cachedTokensExpr := channelStatsJsonExtractInt("cache_tokens")
	frtExpr := channelStatsJsonExtractFloat("frt")
	successExpr := channelStatsSuccessExpr()
	useTimeCol := channelStatsUseTimeCol()

	// total_tokens = prompt_tokens + completion_tokens (already computed per row)
	//   When includeCache is true, cached_tokens are added so the total matches the
	//   upstream provider's dashboard (which counts cache-read input tokens).
	// avg_speed = completion_tokens / use_time (output tokens per second)
	//   NOTE: prompt_tokens includes cache_tokens (cached hits need no computation),
	//   so using total_tokens would hugely overstate speed. Output speed is the
	//   standard industry metric for tok/s.
	// cache_hit_ratio = cached_tokens / prompt_tokens
	// success_rate = success_count / total_count

	totalTokensExpr := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		totalTokensExpr += " + " + logCacheTokensSumExpr()
	}

	return "channel_id, model_name, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
		totalTokensExpr + " as total_tokens, " +
		"COALESCE(SUM(" + cachedTokensExpr + "), 0) as cached_tokens, " +
		"COALESCE(AVG(" + frtExpr + "), 0.0) as avg_first_byte_ms, " +
		// avg_speed = SUM(completion_tokens) / SUM(use_time) — output token generation speed
		"COALESCE(1.0 * COALESCE(SUM(completion_tokens), 0) / " +
		"  NULLIF(COALESCE(SUM(" + useTimeCol + "), 0), 0), 0.0) as avg_speed_tok_per_s, " +
		"COALESCE(1.0 * COALESCE(SUM(" + cachedTokensExpr + "), 0) / " +
		"  NULLIF(COALESCE(SUM(prompt_tokens), 0), 0), 0.0) as cache_hit_ratio, " +
		"COALESCE(1.0 * COALESCE(SUM(" + successExpr + "), 0) / " +
		"  NULLIF(COUNT(*), 0), 0.0) as success_rate, " +
		"COALESCE(SUM(quota), 0) as quota"
}

// GetChannelModelStats returns aggregated statistics grouped by channel_id and model_name
// for all users within the given time range. When username is non-empty, results are
// filtered to that user. When includeCache is true, cached_tokens are included in
// total_tokens.
func GetChannelModelStats(startTime int64, endTime int64, username string, includeCache bool) ([]*ChannelModelStats, error) {
	var data []*ChannelModelStats

	query := LOG_DB.Table("logs").
		Select(channelStatsSelectColumns(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)

	if username != "" {
		query = query.Where("username = ?", username)
	}

	err := query.
		Group("channel_id, model_name").
		Order("request_count DESC").
		Find(&data).Error

	if err != nil {
		return nil, err
	}

	// Resolve channel names from the channel cache or database
	resolveChannelNames(data)

	return data, nil
}

// GetSelfChannelModelStats returns aggregated statistics grouped by channel_id and
// model_name for a specific user within the given time range. When includeCache is
// true, cached_tokens are included in total_tokens.
func GetSelfChannelModelStats(userId int, startTime int64, endTime int64, includeCache bool) ([]*ChannelModelStats, error) {
	var data []*ChannelModelStats

	err := LOG_DB.Table("logs").
		Select(channelStatsSelectColumns(includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Group("channel_id, model_name").
		Order("request_count DESC").
		Find(&data).Error

	if err != nil {
		return nil, err
	}

	resolveChannelNames(data)

	return data, nil
}

// resolveChannelNames populates the ChannelName field for each stats item by looking up
// the channel_id in the channel cache (if available) or the database via a single batch query.
func resolveChannelNames(data []*ChannelModelStats) {
	if len(data) == 0 {
		return
	}

	// Collect unique channel IDs
	channelIds := make(map[int]bool)
	for _, item := range data {
		if item.ChannelID > 0 {
			channelIds[item.ChannelID] = true
		}
	}

	if len(channelIds) == 0 {
		return
	}

	// Build channel name map
	channelNameMap := make(map[int]string, len(channelIds))

	// First pass: try to resolve from memory cache
	uncachedIds := make([]int, 0, len(channelIds))
	for channelId := range channelIds {
		if common.MemoryCacheEnabled {
			if cached, err := CacheGetChannel(channelId); err == nil {
				channelNameMap[channelId] = cached.Name
				continue
			}
		}
		uncachedIds = append(uncachedIds, channelId)
	}

	// Second pass: bulk query from main DB for all uncached IDs
	if len(uncachedIds) > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := DB.Table("channels").Select("id, name").Where("id IN ?", uncachedIds).Find(&channels).Error; err == nil {
			for _, ch := range channels {
				channelNameMap[ch.Id] = ch.Name
			}
		}
	}

	// Assign channel names
	for _, item := range data {
		if item.ChannelID > 0 {
			if name, ok := channelNameMap[item.ChannelID]; ok {
				item.ChannelName = name
			}
		}
	}
}