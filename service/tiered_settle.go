package service

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// TieredResultWrapper wraps billingexpr.TieredResult for use at the service layer.
type TieredResultWrapper = billingexpr.TieredResult

// BuildTieredTokenParams constructs billingexpr.TokenParams from a dto.Usage,
// normalizing P and C so they mean "tokens not separately priced by the
// expression". Sub-categories (cache, image, audio) are only subtracted
// when the expression references them via their own variable.
//
// GPT-format APIs report prompt_tokens / completion_tokens as totals that
// include all sub-categories (cache, image, audio). Claude-format APIs
// report them as text-only. This function normalizes to text-only when
// sub-categories are separately priced.
func BuildTieredTokenParams(usage *dto.Usage, isClaudeUsageSemantic bool, usedVars map[string]bool) billingexpr.TokenParams {
	p := float64(usage.PromptTokens)
	c := float64(usage.CompletionTokens)
	cr := float64(usage.PromptTokensDetails.CachedTokens)
	cc5m := float64(usage.PromptTokensDetails.CacheCreationTokensTotal())
	cc1h := float64(0)

	// Legacy Claude-derived OpenAI usage carries the 5m/1h cache-creation
	// split without an anthropic semantic tag (mirrors the ratio path's
	// isLegacyClaudeDerivedOpenAIUsage). Those tokens are not part of
	// prompt_tokens, so they must be billed at their own cc/cc1h rates and
	// must NOT be subtracted from p below.
	legacyClaudeDerived := !isClaudeUsageSemantic &&
		usage.UsageSource == "" && usage.UsageSemantic == "" &&
		(usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0)

	if usage.UsageSemantic == "anthropic" || legacyClaudeDerived {
		cc1h = float64(usage.ClaudeCacheCreation1hTokens)
		cc5m = float64(usage.ClaudeCacheCreation5mTokens)
	}

	img := float64(usage.PromptTokensDetails.ImageTokens)
	ai := float64(usage.PromptTokensDetails.AudioTokens)
	imgO := float64(usage.CompletionTokenDetails.ImageTokens)
	ao := float64(usage.CompletionTokenDetails.AudioTokens)

	// Upstream token fields are untrusted: a hostile or buggy upstream can
	// report negative counts, and a negative term would lower (or invert) the
	// charge. Clamp every dimension at zero before it reaches the expression.
	if p < 0 {
		p = 0
	}
	if c < 0 {
		c = 0
	}
	if cr < 0 {
		cr = 0
	}
	if cc5m < 0 {
		cc5m = 0
	}
	if cc1h < 0 {
		cc1h = 0
	}
	if img < 0 {
		img = 0
	}
	if ai < 0 {
		ai = 0
	}
	if imgO < 0 {
		imgO = 0
	}
	if ao < 0 {
		ao = 0
	}

	// len = total input context length for tier condition evaluation.
	// Non-Claude: prompt_tokens already includes everything.
	// Claude: input_tokens is text-only, so add cache read + cache creation.
	inputLen := p
	if isClaudeUsageSemantic {
		inputLen = p + cr + cc5m + cc1h
	}

	if !isClaudeUsageSemantic {
		if !legacyClaudeDerived {
			if usedVars["cr"] {
				p -= cr
			}
			if usedVars["cc"] {
				p -= cc5m
			}
			if usedVars["cc1h"] {
				p -= cc1h
			}
		}
		if usedVars["img"] {
			p -= img
		}
		if usedVars["ai"] {
			p -= ai
		}
		if usedVars["img_o"] {
			c -= imgO
		}
		if usedVars["ao"] {
			c -= ao
		}
	}

	// OpenAI cache-write usage reports unadjusted prefix counts, so cr + cc can
	// exceed the prompt and drive the remainder negative. Clamp at zero.
	if p < 0 {
		p = 0
	}
	if c < 0 {
		c = 0
	}

	return billingexpr.TokenParams{
		P:    p,
		C:    c,
		Len:  inputLen,
		CR:   cr,
		CC:   cc5m,
		CC1h: cc1h,
		Img:  img,
		ImgO: imgO,
		AI:   ai,
		AO:   ao,
	}
}

func refreshTieredBillingGroup(relayInfo *relaycommon.RelayInfo) (*billingexpr.BillingSnapshot, error) {
	if relayInfo == nil {
		return nil, nil
	}
	snap := relayInfo.TieredBillingSnapshot
	if snap == nil || snap.BillingMode != "tiered_expr" {
		return nil, nil
	}

	groupRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	if snap.GroupRatio == groupRatio {
		return snap, nil
	}

	estimatedQuotaAfterGroup := snap.EstimatedQuotaBeforeGroup * groupRatio
	estimatedQuota, err := billingexpr.QuotaRoundStrict(estimatedQuotaAfterGroup)
	if err != nil {
		return nil, err
	}
	snap.GroupRatio = groupRatio
	snap.EstimatedQuotaAfterGroup = estimatedQuota
	return snap, nil
}

// PrepareTieredBillingForSelectedGroup refreshes routing-dependent billing
// state before an upstream attempt. An existing session reserves any higher
// estimate before sending. If the initial group was free and skipped
// pre-consume, switching to a paid group creates the session at that point.
func PrepareTieredBillingForSelectedGroup(c *gin.Context, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
	snap, err := refreshTieredBillingGroup(relayInfo)
	if err != nil {
		return types.NewErrorWithStatusCode(
			err,
			types.ErrorCodeModelPriceError,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}
	if snap == nil {
		return nil
	}
	if snap.GroupRatio == 0 {
		// Paid-to-free keeps FreeModel as-is: FreeModel means "pre-consume was
		// skipped", which is not true once a session exists, and settlement
		// already yields 0 for a zero group ratio.
		return nil
	}

	// The selected group is paid; clear a FreeModel flag frozen when the
	// initial group was free so downstream state stays consistent.
	relayInfo.PriceData.FreeModel = false

	if relayInfo.Billing == nil {
		return PreConsumeBilling(c, snap.EstimatedQuotaAfterGroup, relayInfo)
	}
	if err := relayInfo.Billing.Reserve(snap.EstimatedQuotaAfterGroup); err != nil {
		if errors.Is(err, model.ErrWeeklyQuotaExceeded) {
			return types.NewErrorWithStatusCode(
				err,
				types.ErrorCodeInsufficientUserQuota,
				http.StatusForbidden,
				types.ErrOptionWithSkipRetry(),
				types.ErrOptionWithNoRecordErrorLog(),
			)
		}
		return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}
	relayInfo.FinalPreConsumedQuota = relayInfo.Billing.GetPreConsumedQuota()
	return nil
}

// TryTieredSettle checks if the request uses tiered_expr billing and, if so,
// computes the actual quota using the captured BillingSnapshot. Returns:
//   - ok=true, quota, result  when tiered billing applies
//   - ok=false, 0, nil        when it doesn't (caller should fall through to existing logic)
func TryTieredSettle(relayInfo *relaycommon.RelayInfo, params billingexpr.TokenParams) (ok bool, quota int, result *billingexpr.TieredResult) {
	snap := relayInfo.TieredBillingSnapshot
	if snap == nil || snap.BillingMode != "tiered_expr" {
		return false, 0, nil
	}

	requestInput := billingexpr.RequestInput{}
	if relayInfo.BillingRequestInput != nil {
		requestInput = *relayInfo.BillingRequestInput
	}

	tr, err := billingexpr.ComputeTieredQuotaWithRequest(snap, params, requestInput)
	if err != nil {
		quota = relayInfo.FinalPreConsumedQuota
		if quota <= 0 {
			quota = snap.EstimatedQuotaAfterGroup
		}
		return true, quota, nil
	}

	// Surface any int32 saturation from settlement onto RelayInfo so the
	// consume log records it under admin_info, regardless of which caller
	// (text, audio, WSS) consumes the returned quota. First non-nil wins.
	noteQuotaClamp(relayInfo, tr.Clamp)

	return true, tr.ActualQuotaAfterGroup, &tr
}
