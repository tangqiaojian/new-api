package billing_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Expressions are the billing contract: they must be compiled and
// smoke-tested before storage (expr.md §Storage). A bad expression must fail
// the save instead of surfacing as a settlement error on user requests.
func TestValidateBillingExprMapJSON(t *testing.T) {
	require.NoError(t, ValidateBillingExprMapJSON(`{"gpt-5":"tier(\"base\", p * 2.5 + c * 15)"}`),
		"well-formed expression map passes")

	require.NoError(t, ValidateBillingExprMapJSON(`{"gpt-5":""}`),
		"empty expressions are skipped")

	err := ValidateBillingExprMapJSON(`not-json`)
	require.Error(t, err, "malformed JSON is rejected")

	err = ValidateBillingExprMapJSON(`{"gpt-5":"p *"}`)
	require.Error(t, err, "uncompilable expression is rejected")

	err = ValidateBillingExprMapJSON(`{"gpt-5":"tier(\"base\", p * -1)"}`)
	require.Error(t, err, "negative-result expression is rejected")

	err = ValidateBillingExprMapJSON(`{"gpt-5":"tier(\"base\", p / c)"}`)
	require.Error(t, err, "expression that divides by zero on smoke vectors is rejected")

	assert.Error(t, ValidateBillingExprMapJSON(`{"a":"tier(\"base\", p * 2)","b":"tier(\"base\", 0 / 0)"}`),
		"one bad entry fails the whole map")
}
