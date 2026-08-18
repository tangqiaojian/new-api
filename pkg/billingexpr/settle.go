package billingexpr

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
)

// quotaConversion converts raw expression output to quota based on the
// expression version. This is the central dispatch point for future versions
// that may use a different conversion formula.
func quotaConversion(exprOutput float64, snap *BillingSnapshot) float64 {
	switch snap.ExprVersion {
	default: // v1: coefficients are $/1M tokens prices
		return exprOutput / 1_000_000 * snap.QuotaPerUnit
	}
}

// ComputeTieredQuota runs the Expr from a frozen BillingSnapshot against
// actual token counts and returns the settlement result.
func ComputeTieredQuota(snap *BillingSnapshot, params TokenParams) (TieredResult, error) {
	return ComputeTieredQuotaWithRequest(snap, params, RequestInput{})
}

func ComputeTieredQuotaWithRequest(snap *BillingSnapshot, params TokenParams, request RequestInput) (TieredResult, error) {
	cost, trace, err := RunExprByHashWithRequest(snap.ExprString, snap.ExprHash, params, request)
	if err != nil {
		return TieredResult{}, err
	}

	quotaBeforeGroup := quotaConversion(cost, snap)
	// The conversion multiply can overflow to +Inf even when cost is finite,
	// and a negative result (negative coefficients, hostile upstream token
	// fields) must never become a credit. Both are settlement errors: callers
	// fall back to the pre-consumed estimate instead of charging 0/MaxInt32.
	if math.IsNaN(quotaBeforeGroup) || math.IsInf(quotaBeforeGroup, 0) {
		return TieredResult{}, fmt.Errorf("tiered quota conversion is not finite (cost=%v, quotaPerUnit=%v)", cost, snap.QuotaPerUnit)
	}
	if quotaBeforeGroup < 0 {
		common.SysError(fmt.Sprintf("tiered billing expression produced negative quota %v (cost=%v, expr hash %s)", quotaBeforeGroup, cost, snap.ExprHash))
		return TieredResult{}, fmt.Errorf("tiered quota conversion produced negative quota: %v", quotaBeforeGroup)
	}
	afterGroup, clamp := common.QuotaRoundChecked(quotaBeforeGroup * snap.GroupRatio)
	crossed := trace.MatchedTier != snap.EstimatedTier

	return TieredResult{
		ActualQuotaBeforeGroup: quotaBeforeGroup,
		ActualQuotaAfterGroup:  afterGroup,
		MatchedTier:            trace.MatchedTier,
		RequestRules:           trace.RequestRules,
		CrossedTier:            crossed,
		Clamp:                  clamp,
	}, nil
}
