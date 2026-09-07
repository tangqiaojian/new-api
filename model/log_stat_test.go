package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSumUsedQuotaPreservesQuotaWhenRpmTpmScanSeparately(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	oldCreatedAt := now - 7*24*3600

	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           1,
		Username:         "alice",
		CreatedAt:        oldCreatedAt,
		Type:             LogTypeConsume,
		ModelName:        "gpt-4",
		Quota:            1500,
		PromptTokens:     100,
		CompletionTokens: 50,
		TokenName:        "tok",
		ChannelId:        1,
		Group:            "default",
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           1,
		Username:         "alice",
		CreatedAt:        oldCreatedAt + 10,
		Type:             LogTypeConsume,
		ModelName:        "gpt-4",
		Quota:            500,
		PromptTokens:     40,
		CompletionTokens: 20,
		TokenName:        "tok",
		ChannelId:        1,
		Group:            "default",
	}).Error)
	// Recent row within the 60s rpm/tpm window.
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           1,
		Username:         "alice",
		CreatedAt:        now - 10,
		Type:             LogTypeConsume,
		ModelName:        "gpt-4",
		Quota:            100,
		PromptTokens:     10,
		CompletionTokens: 5,
		TokenName:        "tok",
		ChannelId:        1,
		Group:            "default",
	}).Error)

	stat, err := SumUsedQuota(LogTypeConsume, oldCreatedAt-1, now+1, "", "alice", "", 0, "", false)
	require.NoError(t, err)
	assert.Equal(t, 2100, stat.Quota, "range quota must not be wiped by rpm/tpm Scan")
	assert.Equal(t, 1, stat.Rpm, "rpm counts only last 60s")
	assert.Equal(t, 15, stat.Tpm)
}

func TestSumUsedQuotaKeepsQuotaWhenNoRecentRpm(t *testing.T) {
	truncateTables(t)

	oldCreatedAt := time.Now().Add(-48 * time.Hour).Unix()
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           2,
		Username:         "bob",
		CreatedAt:        oldCreatedAt,
		Type:             LogTypeConsume,
		ModelName:        "gpt-4",
		Quota:            999,
		PromptTokens:     200,
		CompletionTokens: 100,
		TokenName:        "tok",
		ChannelId:        1,
		Group:            "default",
	}).Error)

	stat, err := SumUsedQuota(LogTypeConsume, oldCreatedAt-1, oldCreatedAt+1, "", "bob", "", 0, "", false)
	require.NoError(t, err)
	assert.Equal(t, 999, stat.Quota)
	assert.Equal(t, 0, stat.Rpm)
	assert.Equal(t, 0, stat.Tpm)
}
