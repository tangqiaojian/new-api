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
	require.Len(t, withCache, 1)
	assert.Equal(t, "2024-01-01", withCache[0].Date)
	assert.Equal(t, 180, withCache[0].PromptTokens)
	assert.Equal(t, 33, withCache[0].CompletionTokens)
	assert.Equal(t, 58, withCache[0].CachedTokens)
	assert.Equal(t, 271, withCache[0].TotalTokens)
	assert.Equal(t, 4, withCache[0].RequestCount)
	assert.Equal(t, 56, withCache[0].Quota)

	withoutCache, err := GetDailyTokenDataByUserId(1, analyticsTestStart, endTime, false)
	require.NoError(t, err)
	require.Len(t, withoutCache, 1)
	assert.Equal(t, 58, withoutCache[0].CachedTokens)
	assert.Equal(t, 213, withoutCache[0].TotalTokens)

	allUsers, err := GetAllDailyTokenData(analyticsTestStart, endTime, "", true)
	require.NoError(t, err)
	require.Len(t, allUsers, 2)
	assert.Equal(t, "alice", allUsers[0].Username)
	assert.Equal(t, "bob", allUsers[1].Username)
	assert.Equal(t, 35, allUsers[1].TotalTokens)
	assert.Zero(t, allUsers[1].CachedTokens)

	models, err := GetAllDailyModelTokenData(analyticsTestStart, endTime, true)
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "gpt-a", models[0].ModelName)
	assert.Equal(t, 257, models[0].TotalTokens)
	assert.Equal(t, 55, models[0].CachedTokens)
	assert.Equal(t, "gpt-b", models[1].ModelName)
	assert.Equal(t, 49, models[1].TotalTokens)
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
