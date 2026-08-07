package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const responsesUsageWithReasoning = `{"id":"resp_1","object":"response","status":"completed","usage":{"input_tokens":10,"output_tokens":25,"total_tokens":35,"completion_tokens_details":{"reasoning_tokens":15}}}`

// The Responses API usage payload carries completion_tokens_details.reasoning_tokens;
// it must survive parsing so the usage log can display the thinking token count.
func TestOaiResponsesHandlerExtractsReasoningTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "responses-rt-test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(responsesUsageWithReasoning)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5"},
		IsStream:    false,
	}

	usage, err := OaiResponsesHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 25, usage.CompletionTokens)
	assert.Equal(t, 35, usage.TotalTokens)
	assert.Equal(t, 15, usage.CompletionTokenDetails.ReasoningTokens)
}

func TestOaiResponsesStreamHandlerExtractsReasoningTokens(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-test","created_at":1710000000}}`,
		`data: {"type":"response.output_text.delta","delta":"hello"}`,
		`data: {"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":10,"output_tokens":25,"total_tokens":35,"completion_tokens_details":{"reasoning_tokens":15}}}}`,
		`data: [DONE]`,
		``,
	}, "\n")

	c, recorder, resp, info := newResponsesChatTestContext(t, body, true)

	usage, err := OaiResponsesStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 25, usage.CompletionTokens)
	assert.Equal(t, 15, usage.CompletionTokenDetails.ReasoningTokens)

	got := recorder.Body.String()
	require.Contains(t, got, `"reasoning_tokens":15`)
}

// Responses usage without completion_tokens_details must stay zero instead of
// picking up stale values.
func TestOaiResponsesHandlerWithoutReasoningTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "responses-rt-test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"id":"resp_1","object":"response","status":"completed","usage":{"input_tokens":10,"output_tokens":25,"total_tokens":35}}`)),
	}

	usage, err := OaiResponsesHandler(c, nil, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, 0, usage.CompletionTokenDetails.ReasoningTokens)
	assert.Equal(t, 0, usage.PromptTokensDetails.CachedTokens)
}
