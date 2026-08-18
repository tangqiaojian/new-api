package service

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Upstream token fields are untrusted: a hostile or buggy upstream can report
// negative sub-category counts. Every dimension must clamp at zero before it
// reaches the billing expression, otherwise a negative term lowers (or
// inverts) the charge and settlement can turn into a wallet credit.
func TestBuildTieredTokenParamsClampsNegativeUpstreamFields(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: -5,
			ImageTokens:  -4_000_000,
			AudioTokens:  -10,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ImageTokens: -7,
			AudioTokens: -3,
		},
		ClaudeCacheCreation5mTokens: -100,
		ClaudeCacheCreation1hTokens: -50,
		UsageSemantic:               "anthropic",
	}
	usedVars := map[string]bool{"cr": true, "cc": true, "cc1h": true, "img": true, "ai": true, "img_o": true, "ao": true}

	params := BuildTieredTokenParams(usage, true, usedVars)

	assert.Zero(t, params.CR)
	assert.Zero(t, params.CC)
	assert.Zero(t, params.CC1h)
	assert.Zero(t, params.Img)
	assert.Zero(t, params.AI)
	assert.Zero(t, params.ImgO)
	assert.Zero(t, params.AO)
	assert.Equal(t, 1000.0, params.P, "clamped sub-categories must not inflate the base remainder")
	assert.Equal(t, 500.0, params.C)
	assert.Equal(t, 1000.0, params.Len)
}

// Legacy Claude-derived OpenAI usage (OpenAI relay format, no semantic tag,
// Claude 5m/1h split fields set) carries cache-creation tokens that are NOT
// part of prompt_tokens. They must be billed at their cc/cc1h rates — the
// ratio path bills them, so the tiered path must too — and must not be
// subtracted from p.
func TestBuildTieredTokenParamsLegacyClaudeDerived(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 5,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}
	usedVars := map[string]bool{"cr": true, "cc": true, "cc1h": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	assert.Equal(t, 10.0, params.CC, "legacy 5m cache creation must be billed at the cc rate")
	assert.Equal(t, 20.0, params.CC1h, "legacy 1h cache creation must be billed at the cc1h rate")
	assert.Equal(t, 5.0, params.CR)
	assert.Equal(t, 1000.0, params.P, "legacy Claude cache tokens are not part of prompt_tokens; nothing to subtract")
	assert.Equal(t, 1000.0, params.Len)
}

// Standard OpenAI-format usage still excludes separately-priced sub-categories
// from p after the clamp changes.
func TestBuildTieredTokenParamsOpenAIExclusionUnchanged(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         200,
			CachedCreationTokens: 30,
			ImageTokens:          100,
		},
	}
	usedVars := map[string]bool{"cr": true, "cc": true, "img": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	assert.Equal(t, 670.0, params.P, "p = 1000 - 200 cr - 30 cc - 100 img")
	assert.Equal(t, 200.0, params.CR)
	assert.Equal(t, 30.0, params.CC)
	assert.Equal(t, 100.0, params.Img)
	assert.Equal(t, 1000.0, params.Len, "len is never reduced by sub-category exclusion")
}

// CalcOpenRouterCacheCreateTokens infers a cache-creation count from an
// upstream-reported cost. Upstream cost is untrusted and the inference
// divides by a price difference that can be near zero, so the result must be
// bounded before conversion to int; anomalies return -1 ("cannot infer").
func TestCalcOpenRouterCacheCreateTokensBoundsInference(t *testing.T) {
	priceData := types.PriceData{
		ModelRatio:         2.5,
		CompletionRatio:    1,
		CacheRatio:         0.1,
		CacheCreationRatio: 1.25,
	}
	// quotaPrice = 2.5/5e5 = 5e-6; hidden cache-create = 3000 tokens.
	// cost = 5000*5e-6 + 2000*5e-7 + 3000*6.25e-6 + 1000*5e-6 = 0.04975
	usage := dto.Usage{
		PromptTokens:     10000,
		CompletionTokens: 1000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2000,
		},
		Cost: 0.04975,
	}
	assert.Equal(t, 3000, CalcOpenRouterCacheCreateTokens(usage, priceData), "well-formed cost infers the hidden cache-create count")

	hugeCost := usage
	hugeCost.Cost = 1e20
	assert.Equal(t, -1, CalcOpenRouterCacheCreateTokens(hugeCost, priceData), "absurd upstream cost must not convert to int")

	nearOneRatio := priceData
	nearOneRatio.CacheCreationRatio = 1.0000001
	assert.Equal(t, -1, CalcOpenRouterCacheCreateTokens(usage, nearOneRatio), "near-zero denominator must not amplify float noise into tokens")

	cheaperCreate := priceData
	cheaperCreate.CacheCreationRatio = 0.5
	assert.Equal(t, -1, CalcOpenRouterCacheCreateTokens(usage, cheaperCreate), "cache creation priced below base rate has no usable solution")

	assert.Equal(t, 0, CalcOpenRouterCacheCreateTokens(usage, types.PriceData{ModelRatio: 2.5, CacheCreationRatio: 1}),
		"ratio == 1 keeps the legacy early-out")
}

// TryTieredSettle must fall back to the pre-consumed estimate (not 0, not a
// negative number) when the expression result is negative or non-finite.
func TestTryTieredSettleFallsBackOnGuardedResults(t *testing.T) {
	exprStr := `tier("base", p * 3 + c * 15 / p)`
	snap := makeSnapshot(exprStr, 1.0, 1000, 100)
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: snap,
		FinalPreConsumedQuota: 4242,
	}

	// p = 0 at settlement (all input cached) makes c / p = +Inf.
	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 0, C: 100, Len: 100})
	require.True(t, ok)
	assert.Nil(t, result, "guarded settlement reports no tiered result")
	assert.Equal(t, 4242, quota, "settlement must fall back to the pre-consumed estimate")
}
