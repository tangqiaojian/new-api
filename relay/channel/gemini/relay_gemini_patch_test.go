package gemini

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression: the zero-completion patch used to replace BillingUsage with a
// prompt+completion-only estimate, dropping cache/image/audio metadata, so a
// client abort mid-stream billed those tokens at the plain text rate and
// dropped them from the log. The patch must preserve the original metadata
// and only fill in the missing completion.
func TestPatchGeminiZeroCompletionPreservesCacheAndModalities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3-flash-preview",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3-flash-preview",
		},
	}
	usage := &dto.Usage{
		PromptTokens: 1000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 400,
			ImageTokens:  200,
		},
		BillingUsage: &dto.BillingUsage{
			Source:   dto.BillingUsageSourceGeminiChat,
			Semantic: dto.BillingUsageSemanticGemini,
			GeminiUsageMetadata: &dto.GeminiUsageMetadata{
				PromptTokenCount:        1000,
				TotalTokenCount:         1000,
				CachedContentTokenCount: 400,
				PromptTokensDetails: []dto.GeminiPromptTokensDetails{
					{Modality: "TEXT", TokenCount: 800},
					{Modality: "IMAGE", TokenCount: 200},
				},
			},
		},
	}

	patchGeminiZeroCompletionUsage(c, info, usage, "partial streamed answer before disconnect", 0)

	require.Greater(t, usage.CompletionTokens, 0, "completion is estimated from the streamed text")
	metadata := usage.BillingUsage.GeminiUsageMetadata
	require.NotNil(t, metadata)
	assert.Equal(t, 400, metadata.CachedContentTokenCount, "cache metadata must survive the completion patch")
	assert.Len(t, metadata.PromptTokensDetails, 2, "modality details must survive the completion patch")
	assert.Equal(t, usage.CompletionTokens, metadata.CandidatesTokenCount)
	assert.Equal(t, metadata.PromptTokenCount+metadata.ToolUsePromptTokenCount+metadata.CandidatesTokenCount, metadata.TotalTokenCount)
	assert.True(t, usage.BillingUsage.Estimated)
}
