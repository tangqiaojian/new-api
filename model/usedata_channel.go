package model

import "github.com/QuantumNous/new-api/common"

// ChannelModelStats contains usage aggregated by channel and model.
type ChannelModelStats struct {
	ChannelID        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	ModelName        string  `json:"model_name"`
	RequestCount     int     `json:"request_count"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CachedTokens     int     `json:"cached_tokens"`
	AvgFirstByteMs   float64 `json:"avg_first_byte_ms"`
	AvgSpeedTokPerS  float64 `json:"avg_speed_tok_per_s"`
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
	SuccessRate      float64 `json:"success_rate"`
	TotalTokens      int     `json:"total_tokens"`
	Quota            int     `json:"quota"`
}

func logOtherColumn() string {
	if common.UsingLogDatabase(common.DatabaseTypePostgreSQL) {
		return `"other"`
	}
	return "`other`"
}

func logJSONIntExpression(key string) string {
	otherColumn := logOtherColumn()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "JSONExtractInt(" + otherColumn + ", '" + key + "')"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "CAST((NULLIF(btrim(" + otherColumn + "), '')::jsonb)->>'" + key + "' AS BIGINT)"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "CAST(json_extract(CASE WHEN json_valid(" + otherColumn + ") THEN " + otherColumn + " ELSE '{}' END, '$." + key + "') AS INTEGER)"
	default:
		return "CAST(JSON_EXTRACT(CASE WHEN JSON_VALID(" + otherColumn + ") THEN " + otherColumn + " ELSE JSON_OBJECT() END, '$." + key + "') AS SIGNED)"
	}
}

func logJSONStringExpression(key string) string {
	otherColumn := logOtherColumn()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "JSONExtractString(" + otherColumn + ", '" + key + "')"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "COALESCE((NULLIF(btrim(" + otherColumn + "), '')::jsonb)->>'" + key + "', '')"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "COALESCE(json_extract(CASE WHEN json_valid(" + otherColumn + ") THEN " + otherColumn + " ELSE '{}' END, '$." + key + "'), '')"
	default:
		return "COALESCE(JSON_UNQUOTE(JSON_EXTRACT(CASE WHEN JSON_VALID(" + otherColumn + ") THEN " + otherColumn + " ELSE JSON_OBJECT() END, '$." + key + "')), '')"
	}
}

func logJSONFloatExpression(key string) string {
	otherColumn := logOtherColumn()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "JSONExtractFloat(" + otherColumn + ", '" + key + "')"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "CAST((NULLIF(btrim(" + otherColumn + "), '')::jsonb)->>'" + key + "' AS DOUBLE PRECISION)"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "CAST(json_extract(CASE WHEN json_valid(" + otherColumn + ") THEN " + otherColumn + " ELSE '{}' END, '$." + key + "') AS REAL)"
	default:
		return "CAST(JSON_EXTRACT(CASE WHEN JSON_VALID(" + otherColumn + ") THEN " + otherColumn + " ELSE JSON_OBJECT() END, '$." + key + "') AS DECIMAL(20, 4))"
	}
}

func logStreamSuccessExpression() string {
	otherColumn := logOtherColumn()
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "CASE WHEN JSONExtractString(" + otherColumn + ", 'stream_status', 'status') = 'error' THEN 0 ELSE 1 END"
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		return "CASE WHEN (NULLIF(btrim(" + otherColumn + "), '')::jsonb)->'stream_status'->>'status' = 'error' THEN 0 ELSE 1 END"
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		return "CASE WHEN json_extract(CASE WHEN json_valid(" + otherColumn + ") THEN " + otherColumn + " ELSE '{}' END, '$.stream_status.status') = 'error' THEN 0 ELSE 1 END"
	default:
		return "CASE WHEN JSON_UNQUOTE(JSON_EXTRACT(CASE WHEN JSON_VALID(" + otherColumn + ") THEN " + otherColumn + " ELSE JSON_OBJECT() END, '$.stream_status.status')) = 'error' THEN 0 ELSE 1 END"
	}
}

func logUseTimeColumn() string {
	if common.UsingLogDatabase(common.DatabaseTypePostgreSQL) {
		return `"use_time"`
	}
	return "`use_time`"
}

func channelStatsSelectColumns(includeCache bool) string {
	cachedTokensExpression := logJSONIntExpression("cache_tokens")
	totalTokensExpression := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		totalTokensExpression += " + " + logCacheTokensSumExpr()
	}

	return "channel_id, model_name, " +
		"COUNT(*) AS request_count, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		totalTokensExpression + " AS total_tokens, " +
		"COALESCE(SUM(" + cachedTokensExpression + "), 0) AS cached_tokens, " +
		"COALESCE(AVG(" + logJSONFloatExpression("frt") + "), 0.0) AS avg_first_byte_ms, " +
		"COALESCE(1.0 * COALESCE(SUM(completion_tokens), 0) / NULLIF(COALESCE(SUM(" + logUseTimeColumn() + "), 0), 0), 0.0) AS avg_speed_tok_per_s, " +
		"COALESCE(1.0 * COALESCE(SUM(" + cachedTokensExpression + "), 0) / NULLIF(COALESCE(SUM(prompt_tokens), 0), 0), 0.0) AS cache_hit_ratio, " +
		"COALESCE(1.0 * COALESCE(SUM(" + logStreamSuccessExpression() + "), 0) / NULLIF(COUNT(*), 0), 0.0) AS success_rate, " +
		"COALESCE(SUM(quota), 0) AS quota"
}

// GetChannelModelStats returns channel/model statistics for all users.
func GetChannelModelStats(startTime int64, endTime int64, username string, includeCache bool) ([]*ChannelModelStats, error) {
	data := make([]*ChannelModelStats, 0)
	query := LOG_DB.Table("logs").
		Select(channelStatsSelectColumns(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime)
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if err := query.
		Group("channel_id, model_name").
		Order("request_count DESC, channel_id ASC, model_name ASC").
		Find(&data).Error; err != nil {
		return nil, err
	}
	return data, resolveChannelModelNames(data)
}

// GetSelfChannelModelStats returns channel/model statistics scoped to one user.
func GetSelfChannelModelStats(userID int, startTime int64, endTime int64, includeCache bool) ([]*ChannelModelStats, error) {
	data := make([]*ChannelModelStats, 0)
	if err := LOG_DB.Table("logs").
		Select(channelStatsSelectColumns(includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userID, LogTypeConsume, startTime, endTime).
		Group("channel_id, model_name").
		Order("request_count DESC, channel_id ASC, model_name ASC").
		Find(&data).Error; err != nil {
		return nil, err
	}
	return data, resolveChannelModelNames(data)
}

func resolveChannelModelNames(data []*ChannelModelStats) error {
	channelIDs := make([]int, 0)
	seen := make(map[int]struct{})
	channelNames := make(map[int]string)
	for _, item := range data {
		if item.ChannelID <= 0 {
			continue
		}
		if _, ok := seen[item.ChannelID]; ok {
			continue
		}
		seen[item.ChannelID] = struct{}{}
		if common.MemoryCacheEnabled {
			if channel, err := CacheGetChannel(item.ChannelID); err == nil && channel != nil {
				channelNames[item.ChannelID] = channel.Name
				continue
			}
		}
		channelIDs = append(channelIDs, item.ChannelID)
	}

	if len(channelIDs) > 0 {
		var channels []struct {
			ID   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return err
		}
		for _, channel := range channels {
			channelNames[channel.ID] = channel.Name
		}
	}

	for _, item := range data {
		item.ChannelName = channelNames[item.ChannelID]
	}
	return nil
}
