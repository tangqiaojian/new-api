package billingexpr_test

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IEEE float division never errors inside the evaluator: `x / 0` is +Inf and
// `0 / 0` is NaN. A non-finite expression result must surface as an error so
// settlement falls back to the pre-consumed estimate, instead of saturating
// to MaxInt32 (overcharge) or flooring to 0 (free request).
func TestRunExprRejectsNonFiniteResults(t *testing.T) {
	_, _, err := billingexpr.RunExpr(`p / c`, billingexpr.TokenParams{P: 1000, C: 0})
	require.Error(t, err, "+Inf result must be rejected")

	_, _, err = billingexpr.RunExpr(`p / c`, billingexpr.TokenParams{P: 0, C: 0})
	require.Error(t, err, "NaN result must be rejected")

	cost, _, err := billingexpr.RunExpr(`p / c`, billingexpr.TokenParams{P: 1000, C: 4})
	require.NoError(t, err)
	assert.Equal(t, 250.0, cost, "finite division still works")
}

// A negative expression result (negative coefficients or hostile upstream
// token fields) must never reach quota conversion: it would become a wallet
// credit downstream. Settlement rejects it so the caller falls back to the
// pre-consumed estimate.
func TestComputeTieredQuotaRejectsNegativeCost(t *testing.T) {
	exprStr := `tier("base", p * -1)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	_, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1000})
	require.Error(t, err, "negative cost must be a settlement error, not a credit")
}

// quotaConversion multiplies cost by QuotaPerUnit; an absurd (but admin-set)
// QuotaPerUnit can overflow a finite cost to +Inf. That must be rejected
// before QuotaRoundChecked saturates it into a MaxInt32 debit, and before
// callers hand ActualQuotaBeforeGroup to decimal.NewFromFloat, which panics
// on non-finite input.
func TestComputeTieredQuotaRejectsConversionOverflow(t *testing.T) {
	exprStr := `tier("base", p * 1e200)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 1e300,
	}

	_, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1e108})
	require.Error(t, err, "conversion overflow to +Inf must be a settlement error")
}

// UsedVars drives sub-category token exclusion at settlement. It must return
// the compiled entry's variables directly — re-reading the shared cache after
// compiling raced with cache eviction and could return nil, silently
// disabling exclusion and double-billing cache tokens.
func TestUsedVarsReturnsSubCategoryVars(t *testing.T) {
	exprStr := `p * 3 + c * 15 + cr * 0.3 + cc * 3.75`

	vars := billingexpr.UsedVars(exprStr)
	require.NotNil(t, vars)
	assert.True(t, vars["p"])
	assert.True(t, vars["c"])
	assert.True(t, vars["cr"])
	assert.True(t, vars["cc"])
	assert.False(t, vars["img"], "unused variables must not be reported")

	// A cache wipe between calls must not change the result: the second call
	// recompiles and still returns the full variable set.
	billingexpr.InvalidateCache()
	vars = billingexpr.UsedVars(exprStr)
	require.NotNil(t, vars)
	assert.True(t, vars["cr"])
	assert.True(t, vars["cc"])

	assert.Nil(t, billingexpr.UsedVars(""), "empty expression has no variables")
	assert.Nil(t, billingexpr.UsedVars(`p * * 3`), "uncompilable expression has no variables")
}

// Time-based request probes (hour/weekday/day/...) must evaluate against the
// clock frozen at pre-consume, not the live clock at settlement — otherwise a
// request crossing a time boundary settles with a different multiplier than
// it reserved.
func TestTimeProbesUseFrozenEvalTime(t *testing.T) {
	const frozen = int64(1755417600) // arbitrary fixed instant
	expected := time.Unix(frozen, 0).UTC()
	expectedCost := float64(expected.Day()*1000 + int(expected.Weekday()))

	cost, _, err := billingexpr.RunExprWithRequest(`day("UTC") * 1000 + weekday("UTC")`,
		billingexpr.TokenParams{}, billingexpr.RequestInput{NowUnix: frozen})
	require.NoError(t, err)
	assert.Equal(t, expectedCost, cost, "frozen clock drives time probes")

	again, _, err := billingexpr.RunExprWithRequest(`day("UTC") * 1000 + weekday("UTC")`,
		billingexpr.TokenParams{}, billingexpr.RequestInput{NowUnix: frozen})
	require.NoError(t, err)
	assert.Equal(t, cost, again, "same frozen input is deterministic")

	live, _, err := billingexpr.RunExprWithRequest(`day("UTC") * 1000`,
		billingexpr.TokenParams{}, billingexpr.RequestInput{})
	require.NoError(t, err)
	assert.Equal(t, float64(time.Now().UTC().Day())*1000, live, "zero NowUnix keeps the live clock")
}
