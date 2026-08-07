package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newShouldRetryTestContext(t *testing.T, specificChannel bool) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if specificChannel {
		c.Set(string(constant.ContextKeyTokenSpecificChannelId), "1")
	}
	return c
}

// A token pinned to a specific channel must still retry when the pinned channel
// itself fails (channel error) — the pin suppresses failover to other channels,
// not the retry on the pinned channel itself.
func TestShouldRetrySpecificChannelAllowsChannelErrorRetry(t *testing.T) {
	c := newShouldRetryTestContext(t, true)
	channelErr := types.NewErrorWithStatusCode(fmt.Errorf("pinned channel failed"), types.ErrorCodeChannelNoAvailableKey, http.StatusBadGateway)
	assert.True(t, shouldRetry(c, channelErr, 3))
}

// A pinned channel must not be retried for non-channel errors: with the pin set,
// a retry would re-select the same pinned channel, so fail fast instead of
// hammering it (legacy behavior preserved).
func TestShouldRetrySpecificChannelSuppressesNonChannelError(t *testing.T) {
	c := newShouldRetryTestContext(t, true)
	serverErr := types.NewErrorWithStatusCode(fmt.Errorf("upstream 500"), types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	assert.False(t, shouldRetry(c, serverErr, 3))
}

func TestShouldRetryWithoutSpecificChannelAllowsChannelErrorRetry(t *testing.T) {
	c := newShouldRetryTestContext(t, false)
	channelErr := types.NewErrorWithStatusCode(fmt.Errorf("channel failed"), types.ErrorCodeChannelNoAvailableKey, http.StatusBadGateway)
	assert.True(t, shouldRetry(c, channelErr, 3))
}

func TestShouldRetryWithoutSpecificChannelAllowsServerErrorRetry(t *testing.T) {
	c := newShouldRetryTestContext(t, false)
	serverErr := types.NewErrorWithStatusCode(fmt.Errorf("upstream 500"), types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	assert.True(t, shouldRetry(c, serverErr, 3))
}

// retryTimes <= 0 suppresses retry for non-channel errors; channel errors are
// retried unconditionally (the outer relay loop caps the attempt count).
func TestShouldRetryExhaustedRetryTimesNeverRetries(t *testing.T) {
	for _, specificChannel := range []bool{true, false} {
		c := newShouldRetryTestContext(t, specificChannel)
		serverErr := types.NewErrorWithStatusCode(fmt.Errorf("upstream 500"), types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
		assert.False(t, shouldRetry(c, serverErr, 0))
	}
}

func TestShouldRetryChannelErrorIgnoresRetryTimes(t *testing.T) {
	for _, specificChannel := range []bool{true, false} {
		c := newShouldRetryTestContext(t, specificChannel)
		channelErr := types.NewErrorWithStatusCode(fmt.Errorf("channel failed"), types.ErrorCodeChannelNoAvailableKey, http.StatusBadGateway)
		assert.True(t, shouldRetry(c, channelErr, 0))
	}
}

func TestShouldRetryNilErrorNeverRetries(t *testing.T) {
	c := newShouldRetryTestContext(t, true)
	require.False(t, shouldRetry(c, nil, 3))
}
