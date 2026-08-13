package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendRequestDebugInfoMasksSecretsTruncatesAndRewindsBody(t *testing.T) {
	previousEnabled := common.LogRequestDebugEnabled
	previousMaxBytes := common.LogRequestBodyMaxBytes
	common.LogRequestDebugEnabled = true
	common.LogRequestBodyMaxBytes = 5
	t.Cleanup(func() {
		common.LogRequestDebugEnabled = previousEnabled
		common.LogRequestBodyMaxBytes = previousMaxBytes
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	storage, err := common.CreateBodyStorage([]byte("123456789"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	_, err = storage.Seek(4, io.SeekStart)
	require.NoError(t, err)
	ctx.Set(common.KeyBodyStorage, storage)

	other := map[string]interface{}{}
	appendRequestDebugInfo(ctx, &relaycommon.RelayInfo{RequestHeaders: map[string]string{
		"Authorization":       "Bearer secret",
		"Cookie":              "session=secret",
		"Proxy-Authorization": "Basic secret",
		"X-Api-Key":           "secret",
		"X-Goog-Api-Key":      "secret",
		"X-Custom-Token":      "secret",
		"X-Tenant-ID":         "private-tenant",
		"Content-Type":        "application/json",
		"User-Agent":          "diagnostic-client/1.0",
		"X-Oneapi-Request-Id": "oneapi-request-123",
		"X-Request-ID":        "request-123",
		"Traceparent":         "00-trace-span-01",
	}}, other)

	headers, ok := other["request_headers"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "***", headers["Authorization"])
	assert.Equal(t, "***", headers["Cookie"])
	assert.Equal(t, "***", headers["Proxy-Authorization"])
	assert.Equal(t, "***", headers["X-Api-Key"])
	assert.Equal(t, "***", headers["X-Goog-Api-Key"])
	assert.Equal(t, "***", headers["X-Custom-Token"])
	assert.Equal(t, "***", headers["X-Tenant-ID"])
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "diagnostic-client/1.0", headers["User-Agent"])
	assert.Equal(t, "oneapi-request-123", headers["X-Oneapi-Request-Id"])
	assert.Equal(t, "request-123", headers["X-Request-ID"])
	assert.Equal(t, "00-trace-span-01", headers["Traceparent"])
	assert.Equal(t, "12345\n... (truncated)", other["request_body"])

	position, err := storage.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	assert.Zero(t, position)

	body := make([]byte, 9)
	readBytes, err := storage.Read(body)
	require.NoError(t, err)
	assert.Equal(t, "123456789", string(body[:readBytes]))
}
