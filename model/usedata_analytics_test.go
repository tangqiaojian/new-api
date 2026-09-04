package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const analyticsTestStart int64 = 1704067200

func analyticsLog(userID int, username string, modelName string, channelID int, createdAt int64, promptTokens int, completionTokens int, useTime int, quota int, other map[string]interface{}) Log {
	otherJSON := ""
	if other != nil {
		otherJSON = common.MapToJsonStr(other)
	}
	return Log{
		UserId:           userID,
		Username:         username,
		ModelName:        modelName,
		ChannelId:        channelID,
		CreatedAt:        createdAt,
		Type:             LogTypeConsume,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		UseTime:          useTime,
		Quota:            quota,
		Other:            otherJSON,
	}
}

func seedAnalyticsLogs(t *testing.T) {
	t.Helper()
	truncateTables(t)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	require.NoError(t, DB.Create(&Channel{Id: 10, Name: "alpha"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 20, Name: "beta"}).Error)

	subscription := func(cacheTokens int, firstByte float64, status string, subscriptionID int) map[string]interface{} {
		return map[string]interface{}{
			"billing_source":          "subscription",
			"cache_tokens":            cacheTokens,
			"frt":                     firstByte,
			"stream_status":           map[string]interface{}{"status": status},
			"subscription_id":         subscriptionID,
			"subscription_plan_id":    7,
			"subscription_plan_title": "Pro",
		}
	}

	logs := []Log{
		analyticsLog(1, "alice", "gpt-a", 10, analyticsTestStart+100, 100, 20, 2, 30, subscription(40, 120.5, "ok", 5)),
		analyticsLog(1, "alice", "gpt-a", 10, analyticsTestStart+200, 50, 10, 1, 20, subscription(10, 60, "error", 5)),
		// A legacy row with an empty `other` value must not break JSON aggregation.
		analyticsLog(2, "bob", "gpt-b", 20, analyticsTestStart+300, 30, 5, 1, 9, nil),
		analyticsLog(1, "alice", "gpt-a", 10, analyticsTestStart+500, 20, 2, 1, 4, map[string]interface{}{
			"billing_source": "wallet",
			"cache_tokens":   5,
		}),
		analyticsLog(1, "alice", "gpt-b", 20, analyticsTestStart+600, 10, 1, 1, 2, subscription(3, 30, "ok", 50)),
		// Outside the requested interval.
		analyticsLog(1, "alice", "gpt-a", 10, analyticsTestStart+90000, 999, 999, 1, 999, subscription(999, 1, "ok", 5)),
	}
	errorLog := analyticsLog(1, "alice", "gpt-a", 10, analyticsTestStart+400, 500, 500, 1, 500, subscription(500, 1, "error", 5))
	errorLog.Type = LogTypeError
	logs = append(logs, errorLog)
	require.NoError(t, LOG_DB.Create(&logs).Error)
}

func TestDailyTokenAnalyticsAggregateConsumeLogsAndCache(t *testing.T) {
	seedAnalyticsLogs(t)
	endTime := analyticsTestStart + 3600

	withCache, err := GetDailyTokenDataByUserId(1, analyticsTestStart, endTime, true)
	require.NoError(t, err)
	require.Len(t, withCache, 2)
	assert.Equal(t, "gpt-a", withCache[0].ModelName)
	assert.Equal(t, "2024-01-01", withCache[0].Date)
	assert.Equal(t, 170, withCache[0].PromptTokens)
	assert.Equal(t, 32, withCache[0].CompletionTokens)
	assert.Equal(t, 55, withCache[0].CachedTokens)
	assert.Equal(t, 257, withCache[0].TotalTokens)
	assert.Equal(t, 3, withCache[0].RequestCount)
	assert.Equal(t, 54, withCache[0].Quota)
	assert.Equal(t, "gpt-b", withCache[1].ModelName)
	assert.Equal(t, 14, withCache[1].TotalTokens)
	assert.Equal(t, 3, withCache[1].CachedTokens)
	assert.Equal(t, 1, withCache[1].RequestCount)

	withoutCache, err := GetDailyTokenDataByUserId(1, analyticsTestStart, endTime, false)
	require.NoError(t, err)
	require.Len(t, withoutCache, 2)
	assert.Equal(t, "gpt-a", withoutCache[0].ModelName)
	assert.Equal(t, 55, withoutCache[0].CachedTokens)
	assert.Equal(t, 202, withoutCache[0].TotalTokens)
	assert.Equal(t, 11, withoutCache[1].TotalTokens)

	allUsers, err := GetAllDailyTokenData(analyticsTestStart, endTime, "", true)
	require.NoError(t, err)
	require.Len(t, allUsers, 3)
	assert.Equal(t, "alice", allUsers[0].Username)
	assert.Equal(t, "gpt-a", allUsers[0].ModelName)
	assert.Equal(t, "bob", allUsers[1].Username)
	assert.Equal(t, "gpt-b", allUsers[1].ModelName)
	assert.Equal(t, 35, allUsers[1].TotalTokens)
	assert.Zero(t, allUsers[1].CachedTokens)
	assert.Equal(t, "alice", allUsers[2].Username)
	assert.Equal(t, "gpt-b", allUsers[2].ModelName)
	assert.Equal(t, 14, allUsers[2].TotalTokens)

	models, err := GetAllDailyModelTokenData(analyticsTestStart, endTime, true)
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "gpt-a", models[0].ModelName)
	assert.Equal(t, 257, models[0].TotalTokens)
	assert.Equal(t, 55, models[0].CachedTokens)
	assert.Equal(t, "gpt-b", models[1].ModelName)
	assert.Equal(t, 49, models[1].TotalTokens)
}

func TestDailyTokenAnalyticsGroupByUserModelAndDate(t *testing.T) {
	seedAnalyticsLogs(t)
	endTime := analyticsTestStart + 3600

	allUsers, err := GetAllDailyTokenData(analyticsTestStart, endTime, "", true)
	require.NoError(t, err)
	require.Len(t, allUsers, 3)

	findRow := func(username string, modelName string) *DailyTokenData {
		t.Helper()
		for _, row := range allUsers {
			if row.Username == username && row.ModelName == modelName {
				return row
			}
		}
		return nil
	}

	aliceA := findRow("alice", "gpt-a")
	require.NotNil(t, aliceA)
	assert.Equal(t, 1, aliceA.UserID)
	assert.Equal(t, "2024-01-01", aliceA.Date)
	assert.Equal(t, 170, aliceA.PromptTokens)
	assert.Equal(t, 32, aliceA.CompletionTokens)
	assert.Equal(t, 55, aliceA.CachedTokens)
	assert.Equal(t, 257, aliceA.TotalTokens)
	assert.Equal(t, 3, aliceA.RequestCount)
	assert.Equal(t, 54, aliceA.Quota)

	aliceB := findRow("alice", "gpt-b")
	require.NotNil(t, aliceB)
	assert.Equal(t, 1, aliceB.UserID)
	assert.Equal(t, 10, aliceB.PromptTokens)
	assert.Equal(t, 1, aliceB.CompletionTokens)
	assert.Equal(t, 3, aliceB.CachedTokens)
	assert.Equal(t, 14, aliceB.TotalTokens)
	assert.Equal(t, 1, aliceB.RequestCount)
	assert.Equal(t, 2, aliceB.Quota)

	bobB := findRow("bob", "gpt-b")
	require.NotNil(t, bobB)
	assert.Equal(t, 2, bobB.UserID)
	assert.Equal(t, 35, bobB.TotalTokens)
	assert.Equal(t, 1, bobB.RequestCount)

	aliceOnly, err := GetAllDailyTokenData(analyticsTestStart, endTime, "alice", true)
	require.NoError(t, err)
	require.Len(t, aliceOnly, 2)
	assert.Equal(t, "gpt-a", aliceOnly[0].ModelName)
	assert.Equal(t, "gpt-b", aliceOnly[1].ModelName)
}

func TestChannelAnalyticsScopeMetricsAndResolveNames(t *testing.T) {
	seedAnalyticsLogs(t)
	endTime := analyticsTestStart + 3600

	data, err := GetSelfChannelModelStats(1, analyticsTestStart, endTime, true)
	require.NoError(t, err)
	require.Len(t, data, 2)

	first := data[0]
	assert.Equal(t, 10, first.ChannelID)
	assert.Equal(t, "alpha", first.ChannelName)
	assert.Equal(t, "gpt-a", first.ModelName)
	assert.Equal(t, 3, first.RequestCount)
	assert.Equal(t, 170, first.PromptTokens)
	assert.Equal(t, 32, first.CompletionTokens)
	assert.Equal(t, 55, first.CachedTokens)
	assert.Equal(t, 257, first.TotalTokens)
	assert.InDelta(t, 90.25, first.AvgFirstByteMs, 0.0001)
	assert.InDelta(t, 8, first.AvgSpeedTokPerS, 0.0001)
	assert.InDelta(t, float64(55)/170, first.CacheHitRatio, 0.0001)
	assert.InDelta(t, float64(2)/3, first.SuccessRate, 0.0001)
	assert.Equal(t, 54, first.Quota)

	bobData, err := GetChannelModelStats(analyticsTestStart, endTime, "bob", true)
	require.NoError(t, err)
	require.Len(t, bobData, 1)
	assert.Equal(t, "beta", bobData[0].ChannelName)
	assert.Equal(t, 35, bobData[0].TotalTokens)
	assert.InDelta(t, 1, bobData[0].SuccessRate, 0.0001)
}

// 平均首字节必须只统计真实记录到首字节的请求。历史日志里非流式请求写过
// -1000 哨兵（FirstResponseTime 初始为 startTime-1s），ClickHouse 对缺失键
// 返回 0，这些值都必须被过滤，否则平均值被拖低甚至拖负。
func TestChannelAnalyticsAvgFirstByteFiltersSentinelAndZero(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 30, Name: "frt-channel"}).Error)

	frtLog := func(createdAt int64, other map[string]interface{}) Log {
		return analyticsLog(1, "carol", "gpt-frt", 30, createdAt, 10, 5, 1, 3, other)
	}
	logs := []Log{
		frtLog(analyticsTestStart+100, map[string]interface{}{"frt": -1000.0}),    // 非流式哨兵
		frtLog(analyticsTestStart+200, map[string]interface{}{"frt": 0.0}),        // ClickHouse 缺失键等效值
		frtLog(analyticsTestStart+300, map[string]interface{}{"cache_tokens": 1}), // 无 frt
		frtLog(analyticsTestStart+400, map[string]interface{}{"frt": 100.0}),
		frtLog(analyticsTestStart+500, map[string]interface{}{"frt": 200.0}),
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	data, err := GetChannelModelStats(analyticsTestStart, analyticsTestStart+3600, "carol", true)
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 5, data[0].RequestCount)
	assert.InDelta(t, 150.0, data[0].AvgFirstByteMs, 0.0001,
		"AVG 只应覆盖 frt>0 的两行：(100+200)/2")
}

func TestSubscriptionAnalyticsUseStructuredFilters(t *testing.T) {
	seedAnalyticsLogs(t)
	endTime := analyticsTestStart + 3600

	daily, err := GetSelfSubscriptionDailyUsage(1, analyticsTestStart, endTime, 5, "", true)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	assert.Equal(t, 5, daily[0].SubscriptionID)
	assert.Equal(t, 7, daily[0].PlanID)
	assert.Equal(t, "Pro", daily[0].PlanTitle)
	assert.Equal(t, 150, daily[0].PromptTokens)
	assert.Equal(t, 30, daily[0].CompletionTokens)
	assert.Equal(t, 50, daily[0].CachedTokens)
	assert.Equal(t, 230, daily[0].TotalTokens)
	assert.Equal(t, 2, daily[0].RequestCount)
	assert.Equal(t, 50, daily[0].Quota)

	withoutCache, err := GetSelfSubscriptionDailyUsage(1, analyticsTestStart, endTime, 5, "gpt-a", false)
	require.NoError(t, err)
	require.Len(t, withoutCache, 1)
	assert.Equal(t, 180, withoutCache[0].TotalTokens)
	assert.Equal(t, 50, withoutCache[0].CachedTokens)

	models, err := GetSubscriptionModelUsage(0, analyticsTestStart, endTime, 5, "", true)
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "gpt-a", models[0].ModelName)
	assert.Equal(t, 230, models[0].TotalTokens)
}

func TestQuotaDashboardReadsAuthoritativeLogsWhenExportCacheIsEmpty(t *testing.T) {
	seedAnalyticsLogs(t)
	endTime := analyticsTestStart + 3600

	rows, err := GetAllQuotaDates(analyticsTestStart, endTime, "alice", true)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	var gptARow *QuotaData
	for _, row := range rows {
		if row.ModelName == "gpt-a" {
			gptARow = row
			break
		}
	}
	require.NotNil(t, gptARow)
	assert.Equal(t, 3, gptARow.Count)
	assert.Equal(t, 54, gptARow.Quota)
	assert.Equal(t, 257, gptARow.TokenUsed)
	assert.Equal(t, analyticsTestStart, gptARow.CreatedAt)
}

func TestDailyTokenDateExpressionUsesUTCForEveryLogDatabase(t *testing.T) {
	originalMainType := common.MainDatabaseType()
	originalLogType := common.LogDatabaseType()
	t.Cleanup(func() { common.SetDatabaseTypes(originalMainType, originalLogType) })

	tests := []struct {
		name       string
		database   common.DatabaseType
		expression string
	}{
		{
			name:       "sqlite",
			database:   common.DatabaseTypeSQLite,
			expression: "strftime('%Y-%m-%d', created_at, 'unixepoch')",
		},
		{
			name:       "mysql",
			database:   common.DatabaseTypeMySQL,
			expression: "DATE_FORMAT(DATE_ADD('1970-01-01 00:00:00', INTERVAL created_at SECOND), '%Y-%m-%d')",
		},
		{
			name:       "postgresql",
			database:   common.DatabaseTypePostgreSQL,
			expression: "TO_CHAR(TO_TIMESTAMP(created_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD')",
		},
		{
			name:       "clickhouse",
			database:   common.DatabaseTypeClickHouse,
			expression: "toString(toDate(fromUnixTimestamp(created_at, 'UTC')))",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.SetDatabaseTypes(originalMainType, test.database)
			assert.Equal(t, test.expression, dailyTokenDateExpression())
		})
	}
}
