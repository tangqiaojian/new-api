package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRelayInfoTestContext(path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return ctx
}

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestGenRelayInfoOpenAICapturesAPIReasoningEffort(t *testing.T) {
	ctx := newRelayInfoTestContext("/v1/chat/completions")
	request := &dto.GeneralOpenAIRequest{ReasoningEffort: "high"}

	info := GenRelayInfoOpenAI(ctx, request)

	require.Equal(t, "high", info.APIReasoningEffort)
}

func TestGenRelayInfoOpenAICapturesOpenRouterReasoningEffort(t *testing.T) {
	ctx := newRelayInfoTestContext("/v1/chat/completions")
	request := &dto.GeneralOpenAIRequest{
		Reasoning: json.RawMessage(`{"effort":"medium"}`),
	}

	info := GenRelayInfoOpenAI(ctx, request)

	require.Equal(t, "medium", info.APIReasoningEffort)
}

func TestGenRelayInfoResponsesCapturesAPIReasoningEffort(t *testing.T) {
	ctx := newRelayInfoTestContext("/v1/responses")
	request := &dto.OpenAIResponsesRequest{
		Reasoning: &dto.Reasoning{Effort: "low"},
	}

	info := GenRelayInfoResponses(ctx, request)

	require.Equal(t, "low", info.APIReasoningEffort)
}
