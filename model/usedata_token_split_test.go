package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllQuotaDatesReturnsTokenSplitAndSuccessCounts(t *testing.T) {
	truncateTables(t)

	start := int64(1_704_067_200)
	require.NoError(t, LOG_DB.Create([]Log{
		{
			UserId: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: start + 10,
			Type: LogTypeConsume, PromptTokens: 100, CompletionTokens: 40, Quota: 50,
			Other: `{"cache_tokens":30,"cache_write_tokens":5,"stream_status":{"status":"ok"}}`,
		},
		{
			UserId: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: start + 20,
			Type: LogTypeConsume, PromptTokens: 50, CompletionTokens: 10, Quota: 20,
			Other: `{"cache_tokens":10,"stream_status":{"status":"error"}}`,
		},
		{
			UserId: 2, Username: "bob", ModelName: "gpt-b", CreatedAt: start + 30,
			Type: LogTypeConsume, PromptTokens: 20, CompletionTokens: 5, Quota: 8,
			Other: `{"cache_tokens":2,"cache_creation_tokens":7}`,
		},
	}).Error)

	rows, err := GetAllQuotaDates(start, start+3600, "", false)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	var prompt, completion, cacheRead, cacheWrite, success, errors, count, quota int
	for _, row := range rows {
		prompt += row.PromptTokens
		completion += row.CompletionTokens
		cacheRead += row.CacheReadTokens
		cacheWrite += row.CacheWriteTokens
		success += row.SuccessCount
		errors += row.ErrorCount
		count += row.Count
		quota += row.Quota
	}

	assert.Equal(t, 170, prompt)
	assert.Equal(t, 55, completion)
	assert.Equal(t, 42, cacheRead)
	assert.Equal(t, 12, cacheWrite, "cache_write_tokens + cache_creation_tokens fallback")
	assert.Equal(t, 2, success)
	assert.Equal(t, 1, errors)
	assert.Equal(t, 3, count)
	assert.Equal(t, 78, quota)
}

func TestGetQuotaDataGroupByUserReturnsPerUserTokenSplit(t *testing.T) {
	truncateTables(t)

	start := int64(1_704_067_200)
	require.NoError(t, LOG_DB.Create([]Log{
		{
			UserId: 1, Username: "alice", ModelName: "gpt-a", CreatedAt: start + 10,
			Type: LogTypeConsume, PromptTokens: 80, CompletionTokens: 20, Quota: 30,
			Other: `{"cache_tokens":40,"stream_status":{"status":"ok"}}`,
		},
		{
			UserId: 2, Username: "bob", ModelName: "gpt-a", CreatedAt: start + 20,
			Type: LogTypeConsume, PromptTokens: 10, CompletionTokens: 5, Quota: 4,
			Other: `{"cache_tokens":1}`,
		},
	}).Error)

	rows, err := GetQuotaDataGroupByUser(start, start+3600, false)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byUser := map[string]*QuotaData{}
	for _, row := range rows {
		byUser[row.Username] = row
	}
	require.Contains(t, byUser, "alice")
	require.Contains(t, byUser, "bob")
	assert.Equal(t, 80, byUser["alice"].PromptTokens)
	assert.Equal(t, 40, byUser["alice"].CacheReadTokens)
	assert.Equal(t, 1, byUser["alice"].SuccessCount)
	assert.Equal(t, 10, byUser["bob"].PromptTokens)
	assert.Equal(t, 1, byUser["bob"].CacheReadTokens)
}

func TestLogQuotaDataPersistsTokenSplitFields(t *testing.T) {
	truncateTables(t)

	now := int64(1_704_067_200)
	LogQuotaData(QuotaDataLogParams{
		UserID:           9,
		Username:         "carol",
		ModelName:        "gpt-c",
		Quota:            11,
		CreatedAt:        now,
		TokenUsed:        130,
		PromptTokens:     100,
		CompletionTokens: 30,
		CacheReadTokens:  25,
		CacheWriteTokens: 8,
		Success:          true,
	})
	SaveQuotaDataCache()

	var row QuotaData
	require.NoError(t, DB.Table("quota_data").Where("username = ?", "carol").First(&row).Error)
	assert.Equal(t, 100, row.PromptTokens)
	assert.Equal(t, 30, row.CompletionTokens)
	assert.Equal(t, 25, row.CacheReadTokens)
	assert.Equal(t, 8, row.CacheWriteTokens)
	assert.Equal(t, 1, row.SuccessCount)
	assert.Equal(t, 0, row.ErrorCount)
	assert.Equal(t, 1, row.Count)
}
