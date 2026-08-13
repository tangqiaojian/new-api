package xai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestXAIStreamHandlerPreservesUsageCacheFields(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","created":1710000000,"model":"grok-test","choices":[{"index":0,"delta":{"content":"hello"}}]}`,
		`data: {"id":"chatcmpl_test","object":"chat.completion.chunk","created":1710000000,"model":"grok-test","choices":[],"usage":{"prompt_tokens":100,"completion_tokens":0,"total_tokens":130,"prompt_cache_hit_tokens":20,"prompt_tokens_details":{"cached_tokens":20,"cache_write_tokens":5,"text_tokens":75},"completion_tokens_details":{"reasoning_tokens":2}}}`,
		`data: [DONE]`,
		``,
	}, "\n")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(common.RequestIdKey, "xai-stream-cache-test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "grok-test"},
		RelayFormat: types.RelayFormatOpenAI,
		DisablePing: true,
	}

	usage, err := xAIStreamHandler(c, info, resp)
	require.Nil(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 100, usage.PromptTokens)
	require.Equal(t, 30, usage.CompletionTokens)
	require.Equal(t, 130, usage.TotalTokens)
	require.Equal(t, 20, usage.PromptCacheHitTokens)
	require.Equal(t, 20, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 5, usage.PromptTokensDetails.CacheWriteTokens)
	require.Equal(t, 75, usage.PromptTokensDetails.TextTokens)
	require.Equal(t, 2, usage.CompletionTokenDetails.ReasoningTokens)

	output := recorder.Body.String()
	require.Contains(t, output, `"prompt_tokens_details":{"cached_tokens":20`)
	require.Contains(t, output, `"completion_tokens":30`)
	require.Contains(t, output, `data: [DONE]`)
}
