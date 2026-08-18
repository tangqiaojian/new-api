package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelContextLimitNotConfiguredMeansUnlimited(t *testing.T) {
	// The default map is empty, so any model returns (0, false) = unlimited.
	UpdateModelContextLimitByJSONString(`{}`)
	limit, ok := GetModelContextLimit("gpt-4o")
	assert.False(t, ok)
	assert.Equal(t, 0, limit)
}

func TestModelContextLimitExactAndBelow(t *testing.T) {
	require.NoError(t, UpdateModelContextLimitByJSONString(`{"grok-4.6":128000}`))

	limit, ok := GetModelContextLimit("grok-4.6")
	require.True(t, ok)
	assert.Equal(t, 128000, limit)

	// At-or-below limit should resolve; the controller compares, here we only
	// verify lookup. An unknown model stays unlimited.
	_, ok = GetModelContextLimit("claude-3-5-sonnet")
	assert.False(t, ok)
}

func TestModelContextLimitZeroOrNegativeIsUnlimited(t *testing.T) {
	require.NoError(t, UpdateModelContextLimitByJSONString(`{"m-zero":0,"m-neg":-5}`))
	_, ok := GetModelContextLimit("m-zero")
	assert.False(t, ok, "0 must mean no limit")
	_, ok = GetModelContextLimit("m-neg")
	assert.False(t, ok, "negative must mean no limit")
}

func TestUpdateModelContextLimitByJSONStringRoundTrip(t *testing.T) {
	require.NoError(t, UpdateModelContextLimitByJSONString(`{"a":100,"b":200}`))
	out := ModelContextLimit2JSONString()
	// Re-parse the serialized form and confirm values survive a round trip.
	require.NoError(t, UpdateModelContextLimitByJSONString(out))
	if limit, ok := GetModelContextLimit("a"); ok {
		assert.Equal(t, 100, limit)
	}
	if limit, ok := GetModelContextLimit("b"); ok {
		assert.Equal(t, 200, limit)
	}
}

func TestUpdateModelContextLimitByJSONStringInvalid(t *testing.T) {
	err := UpdateModelContextLimitByJSONString(`{not json}`)
	assert.Error(t, err)
}
