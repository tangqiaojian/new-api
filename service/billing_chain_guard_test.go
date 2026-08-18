package service

import (
	"math"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression: mid-session realtime charges used to bypass the BillingSession
// via PostConsumeQuota while the final PostWssConsumeQuota settled the full
// usage through the session — charging the user roughly twice. Increments
// must extend the session reservation so the final settle reconciles to
// exactly the actual usage.
func TestPreWssConsumeQuotaReservesThroughSession(t *testing.T) {
	truncate(t)
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"test-wss-model": 1}`))
	user := &model.User{Username: "wss-reserve-user", Password: "password", Quota: 1_000_000}
	require.NoError(t, model.DB.Create(user).Error)
	// Simulate the initial pre-consume done by PreConsumeBilling.
	require.NoError(t, model.DecreaseUserQuota(user.Id, 1000, false))

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	relayInfo := &relaycommon.RelayInfo{
		UserId:          user.Id,
		RequestId:       "wss-reserve-req",
		OriginModelName: "test-wss-model",
		UsingGroup:      "default",
		IsPlayground:    true, // skip API-key token accounting
		PriceData: hosttypes.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	session := &BillingSession{
		relayInfo:        relayInfo,
		funding:          &WalletFunding{userId: user.Id, consumed: 1000},
		preConsumedQuota: 1000,
		tokenConsumed:    1000,
	}
	relayInfo.Billing = session

	event := &dto.RealtimeUsage{InputTokens: 2000, OutputTokens: 3000, TotalTokens: 5000}
	event.InputTokenDetails.TextTokens = 2000
	event.OutputTokenDetails.TextTokens = 3000

	require.NoError(t, PreWssConsumeQuota(ctx, relayInfo, event))
	assert.Equal(t, 6000, session.GetPreConsumedQuota(),
		"mid-session increment must extend the session reservation, not consume beside it")

	require.NoError(t, session.Settle(5000))
	quota, err := model.GetUserQuota(user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, 995_000, quota,
		"user pays exactly the actual 5000: pre-consume 1000 + increment 5000 - settle refund 1000")
}

// buildRealtimeTieredTokenParams must mirror BuildTieredTokenParams: opt-in
// sub-category exclusion and non-negative clamps on untrusted upstream fields.
func TestBuildRealtimeTieredTokenParams(t *testing.T) {
	usage := &dto.RealtimeUsage{InputTokens: 1000, OutputTokens: 500}
	usage.InputTokenDetails.CachedTokens = 200
	usage.InputTokenDetails.AudioTokens = 400
	usage.OutputTokenDetails.AudioTokens = 100

	params := buildRealtimeTieredTokenParams(usage, map[string]bool{"cr": true, "ai": true, "ao": true})
	assert.Equal(t, 400.0, params.P, "p = 1000 - 200 cr - 400 ai")
	assert.Equal(t, 400.0, params.C, "c = 500 - 100 ao")
	assert.Equal(t, 200.0, params.CR)
	assert.Equal(t, 400.0, params.AI)
	assert.Equal(t, 100.0, params.AO)
	assert.Equal(t, 1000.0, params.Len, "len is never reduced by sub-category exclusion")

	params = buildRealtimeTieredTokenParams(usage, nil)
	assert.Equal(t, 1000.0, params.P, "unused variables keep their tokens in p")
	assert.Equal(t, 500.0, params.C)

	negative := &dto.RealtimeUsage{InputTokens: -5, OutputTokens: -3}
	negative.InputTokenDetails.CachedTokens = -10
	negative.InputTokenDetails.AudioTokens = -10
	negative.OutputTokenDetails.AudioTokens = -1
	params = buildRealtimeTieredTokenParams(negative, map[string]bool{"cr": true, "ai": true, "ao": true})
	assert.Zero(t, params.P)
	assert.Zero(t, params.C)
	assert.Zero(t, params.CR)
	assert.Zero(t, params.AI)
	assert.Zero(t, params.AO)
}

// OpenRouter serves Claude-format requests with OpenAI-shaped usage
// (prompt_tokens includes cache). The ratio path remaps the prompt; the
// tiered path must apply the same OpenAI exclusion semantics or cache tokens
// are billed twice (inside p and again via cr).
func TestOpenRouterClaudeTieredUsesOpenAIExclusion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	relayInfo := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "anthropic/claude-3.7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: hosttypes.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2432,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	require.True(t, summary.IsClaudeUsageSemantic)
	require.True(t, isOpenRouterClaudeBilling(relayInfo, &summary))

	tieredClaudeSemantic := summary.IsClaudeUsageSemantic && !isOpenRouterClaudeBilling(relayInfo, &summary)
	require.False(t, tieredClaudeSemantic, "OpenAI-shaped usage must not use Claude no-subtract semantics")

	params := BuildTieredTokenParams(usage, tieredClaudeSemantic, map[string]bool{"cr": true})
	assert.Equal(t, 172.0, params.P, "tiered p must match the ratio remap: 2604 - 2432")
	assert.Equal(t, 2432.0, params.CR)
	assert.Equal(t, 2604.0, params.Len, "len keeps the full OpenAI-shaped prompt total")
}

// The OpenRouter remap subtracts cache from the prompt for the log and
// subscription token settlement; untrusted cache counts larger than the
// prompt must be capped, never driving the remapped prompt negative.
func TestCalculateTextQuotaSummaryOpenRouterCapsCacheAtPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	relayInfo := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "anthropic/claude-3.7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: hosttypes.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 200,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	assert.Equal(t, 0, summary.PromptTokens, "remapped prompt must not go negative")
	assert.Equal(t, 100, summary.CacheTokens, "cache is capped at the tokens the prompt actually held")
	assert.Equal(t, 10, summary.Quota, "quota = 100 capped cache * 0.1")
}

// Upstream usage fields are untrusted: negative counts must never reduce a
// charge, and an absurd prompt+completion sum must not overflow TotalTokens
// into "not billable" (which would refund the pre-consume on a delivered
// response).
func TestCalculateTextQuotaSummaryClampsNegativeAndOverflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	newRelayInfo := func() *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{
			OriginModelName: "test-model",
			PriceData: hosttypes.PriceData{
				ModelRatio:      1,
				CompletionRatio: 1,
				GroupRatioInfo:  hosttypes.GroupRatioInfo{GroupRatio: 1},
			},
		}
	}

	overflowUsage := &dto.Usage{PromptTokens: math.MaxInt64, CompletionTokens: 1}
	summary := calculateTextQuotaSummary(ctx, newRelayInfo(), overflowUsage)
	assert.Equal(t, math.MaxInt, summary.TotalTokens, "overflowing sum saturates instead of wrapping negative")
	assert.True(t, summary.hasBillableUsage(), "a huge upstream usage must stay billable")
	assert.Equal(t, math.MaxInt32, summary.Quota, "money math saturates at the int32 quota bound")

	negativeCompletion := &dto.Usage{PromptTokens: 1000, CompletionTokens: -200}
	summary = calculateTextQuotaSummary(ctx, newRelayInfo(), negativeCompletion)
	assert.Equal(t, 0, summary.CompletionTokens)
	assert.Equal(t, 1000, summary.Quota, "negative completion must not offset the prompt charge")

	negativeCache := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 10,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: -50,
		},
	}
	summary = calculateTextQuotaSummary(ctx, newRelayInfo(), negativeCache)
	assert.Equal(t, 0, summary.CacheTokens)
	assert.Equal(t, 110, summary.Quota, "negative cache must not inflate the base charge")
}

// Gemini metadata with total < prompt must not derive a negative completion.
func TestUsageFromGeminiBillingUsageClampsDerivedCompletion(t *testing.T) {
	usage := usageFromGeminiBillingUsage(&dto.BillingUsage{
		GeminiUsageMetadata: &dto.GeminiUsageMetadata{
			PromptTokenCount: 800,
			TotalTokenCount:  500,
		},
	})
	require.NotNil(t, usage)
	assert.Equal(t, 0, usage.CompletionTokens, "derived completion clamps at zero")
}

// Violation-fee quota conversion must saturate at the int32 quota bound
// instead of bare IntPart/int casts on unbounded admin settings.
func TestCalcViolationFeeQuotaSaturates(t *testing.T) {
	assert.Equal(t, int32(math.MaxInt32), int32(calcViolationFeeQuota(1e10, 1e10)),
		"absurd fee settings saturate at the quota bound")
	assert.Equal(t, 500_000, calcViolationFeeQuota(1.0, 1.0), "$1 at 5e5 quota/unit")
	assert.Zero(t, calcViolationFeeQuota(-1, 1))
	assert.Zero(t, calcViolationFeeQuota(1, 0))
}
