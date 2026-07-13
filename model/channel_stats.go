package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ChannelStatsData 按渠道聚合的统计数据
type ChannelStatsData struct {
	ChannelID       int     `json:"channel_id"`
	ChannelName     string  `json:"channel_name"`
	RequestCount    int64   `json:"request_count"`
	SuccessCount    int64   `json:"success_count"`
	ErrorCount      int64   `json:"error_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgUseTime      float64 `json:"avg_use_time"`
	AvgFirstByte    float64 `json:"avg_first_byte"` // 平均首字节时间(ms)，从 Other.frt 提取
	TotalTokens     int64   `json:"total_tokens"`
	CachedTokens    int64   `json:"cached_tokens"` // 缓存命中 Token 数
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	UsedQuota       int64   `json:"used_quota"`
}

// ChannelStatsSummary 渠道统计汇总
type ChannelStatsSummary struct {
	TotalRequests    int64             `json:"total_requests"`
	TotalSuccess     int64             `json:"total_success"`
	TotalErrors      int64             `json:"total_errors"`
	OverallSuccessRate float64         `json:"overall_success_rate"`
	AvgUseTime       float64           `json:"avg_use_time"`
	AvgFirstByte     float64           `json:"avg_first_byte"`
	TotalTokens      int64             `json:"total_tokens"`
	TotalCachedTokens int64            `json:"total_cached_tokens"`
	OverallCacheHitRatio float64       `json:"overall_cache_hit_ratio"`
	TotalUsedQuota   int64             `json:"total_used_quota"`
	TopChannels      []ChannelStatsData `json:"top_channels"`
	AllChannels      []ChannelStatsData `json:"all_channels"`
}

// GetChannelStats 按渠道聚合统计数据
// startTime/endTime 为 unix 秒
// includeCache 为 false 时，缓存命中的 token 不计入 total_tokens，缓存命中率相应调整
func GetChannelStats(startTime, endTime int64, includeCache bool) (*ChannelStatsSummary, error) {
	// 使用日志数据库查询
	var logs []struct {
		ChannelID    int    `gorm:"column:channel_id"`
		Type         int    `gorm:"column:type"`
		UseTime      int    `gorm:"column:use_time"`
		Quota        int    `gorm:"column:quota"`
		PromptTokens int    `gorm:"column:prompt_tokens"`
		CompletionTokens int `gorm:"column:completion_tokens"`
		Other        string `gorm:"column:other"`
	}

	err := LOG_DB.Table("logs").
		Select("channel_id, type, use_time, quota, prompt_tokens, completion_tokens, other").
		Where("channel_id > 0 AND created_at >= ? AND created_at <= ?", startTime, endTime).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	// 按渠道聚合
	channelMap := make(map[int]*ChannelStatsData)
	for _, log := range logs {
		ch, ok := channelMap[log.ChannelID]
		if !ok {
			ch = &ChannelStatsData{ChannelID: log.ChannelID}
			channelMap[log.ChannelID] = ch
		}
		ch.RequestCount++
		ch.UsedQuota += int64(log.Quota)
		ch.TotalTokens += int64(log.PromptTokens + log.CompletionTokens)

		if log.Type == LogTypeConsume {
			ch.SuccessCount++
		} else if log.Type == LogTypeError {
			ch.ErrorCount++
		}

		if log.UseTime > 0 {
			ch.AvgUseTime += float64(log.UseTime)
		}

		// 从 Other JSON 提取 frt 和 cache_tokens
		if log.Other != "" {
			frt, cacheTokens := parseOtherForStats(log.Other)
			if frt > 0 {
				ch.AvgFirstByte += frt
			}
			ch.CachedTokens += int64(cacheTokens)
		}
	}

	// 不含缓存模式：从 total_tokens 中扣除缓存命中的 token，并清零缓存相关指标
	if !includeCache {
		for _, ch := range channelMap {
			ch.TotalTokens -= ch.CachedTokens
			ch.CachedTokens = 0
		}
	}

	// 计算平均值并组装结果
	summary := &ChannelStatsSummary{}
	allChannels := make([]ChannelStatsData, 0, len(channelMap))

	for chID, ch := range channelMap {
		// 获取渠道名称
		if name, err := getChannelNameByID(chID); err == nil && name != "" {
			ch.ChannelName = name
		} else {
			ch.ChannelName = "Unknown"
		}

		// 计算成功率
		if ch.RequestCount > 0 {
			ch.SuccessRate = float64(ch.SuccessCount) / float64(ch.RequestCount) * 100
		}

		// 计算平均响应时间
		if ch.SuccessCount > 0 {
			ch.AvgUseTime = ch.AvgUseTime / float64(ch.SuccessCount)
			ch.AvgFirstByte = ch.AvgFirstByte / float64(ch.SuccessCount)
		} else {
			ch.AvgUseTime = 0
			ch.AvgFirstByte = 0
		}

		// 计算缓存命中率
		if ch.TotalTokens > 0 {
			ch.CacheHitRatio = float64(ch.CachedTokens) / float64(ch.TotalTokens) * 100
		}

		summary.TotalRequests += ch.RequestCount
		summary.TotalSuccess += ch.SuccessCount
		summary.TotalErrors += ch.ErrorCount
		summary.TotalTokens += ch.TotalTokens
		summary.TotalCachedTokens += ch.CachedTokens
		summary.TotalUsedQuota += ch.UsedQuota
		summary.AvgUseTime += ch.AvgUseTime * float64(ch.RequestCount)
		summary.AvgFirstByte += ch.AvgFirstByte * float64(ch.RequestCount)

		allChannels = append(allChannels, *ch)
	}

	// 计算汇总平均值
	if summary.TotalRequests > 0 {
		summary.OverallSuccessRate = float64(summary.TotalSuccess) / float64(summary.TotalRequests) * 100
		summary.AvgUseTime = summary.AvgUseTime / float64(summary.TotalRequests)
		summary.AvgFirstByte = summary.AvgFirstByte / float64(summary.TotalRequests)
	}
	if summary.TotalTokens > 0 {
		summary.OverallCacheHitRatio = float64(summary.TotalCachedTokens) / float64(summary.TotalTokens) * 100
	}

	// Top 渠道按 used_quota 排序
	summary.AllChannels = allChannels
	summary.TopChannels = getTopChannels(allChannels, 10)

	return summary, nil
}

// parseOtherForStats 从 Other JSON 字符串提取 frt（首字节时间ms）和 cache_tokens
func parseOtherForStats(otherStr string) (frt float64, cacheTokens int) {
	if otherStr == "" {
		return 0, 0
	}
	m := make(map[string]interface{})
	if err := common.Unmarshal([]byte(otherStr), &m); err != nil {
		return 0, 0
	}
	if v, ok := m["frt"]; ok {
		if f, ok := v.(float64); ok {
			frt = f
		}
	}
	if v, ok := m["cache_tokens"]; ok {
		switch t := v.(type) {
		case float64:
			cacheTokens = int(t)
		case int:
			cacheTokens = t
		}
	}
	return frt, cacheTokens
}

// getChannelNameByID 通过 ID 获取渠道名称（优先从缓存）
func getChannelNameByID(channelID int) (string, error) {
	if channel, err := CacheGetChannel(channelID); err == nil && channel != nil {
		return channel.Name, nil
	}
	var name string
	err := DB.Model(&Channel{}).Where("id = ?", channelID).Select("name").Find(&name).Error
	return name, err
}

// getTopChannels 按已用额度排序取前 N 个渠道
func getTopChannels(channels []ChannelStatsData, limit int) []ChannelStatsData {
	if len(channels) <= limit {
		// 按 UsedQuota 降序排序
		return sortChannelsByQuota(channels)
	}
	sorted := sortChannelsByQuota(channels)
	return sorted[:limit]
}

// sortChannelsByQuota 按 UsedQuota 降序排序
func sortChannelsByQuota(channels []ChannelStatsData) []ChannelStatsData {
	// 简单选择排序（渠道数量通常不大）
	result := make([]ChannelStatsData, len(channels))
	copy(result, channels)
	for i := 0; i < len(result)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(result); j++ {
			if result[j].UsedQuota > result[maxIdx].UsedQuota {
				maxIdx = j
			}
		}
		result[i], result[maxIdx] = result[maxIdx], result[i]
	}
	return result
}

// GetChannelCacheHitRatios 批量获取多个渠道的缓存命中率（最近 days 天）。
// 返回 channelID -> cacheHitRatio(%) 映射，供渠道列表展示用。
func GetChannelCacheHitRatios(channelIDs []int, days int) (map[int]float64, error) {
	if len(channelIDs) == 0 {
		return map[int]float64{}, nil
	}
	now := time.Now().Unix()
	startTime := now - int64(days)*86400

	type aggRow struct {
		ChannelID       int   `gorm:"column:channel_id"`
		TotalTokens     int64 `gorm:"column:total_tokens"`
		CachedTokens    int64 `gorm:"column:cached_tokens"`
	}

	// 用子查询从 other JSON 提取 cache_tokens
	// 兼容 SQLite/MySQL/PostgreSQL：在 Go 层聚合（避免 JSON 函数方言差异）
	var logs []struct {
		ChannelID       int    `gorm:"column:channel_id"`
		PromptTokens    int    `gorm:"column:prompt_tokens"`
		CompletionTokens int   `gorm:"column:completion_tokens"`
		Other           string `gorm:"column:other"`
	}

	err := LOG_DB.Table("logs").
		Select("channel_id, prompt_tokens, completion_tokens, other").
		Where("channel_id IN ? AND created_at >= ? AND created_at <= ?", channelIDs, startTime, now).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	type acc struct {
		totalTokens  int64
		cachedTokens int64
	}
	accMap := make(map[int]*acc)
	for _, log := range logs {
		a, ok := accMap[log.ChannelID]
		if !ok {
			a = &acc{}
			accMap[log.ChannelID] = a
		}
		a.totalTokens += int64(log.PromptTokens + log.CompletionTokens)
		if log.Other != "" {
			_, cacheTokens := parseOtherForStats(log.Other)
			a.cachedTokens += int64(cacheTokens)
		}
	}

	result := make(map[int]float64, len(accMap))
	for chID, a := range accMap {
		if a.totalTokens > 0 {
			result[chID] = float64(a.cachedTokens) / float64(a.totalTokens) * 100
		} else {
			result[chID] = 0
		}
	}
	return result, nil
}
