package service

import (
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 平均首字节（渠道统计 avg_first_byte_ms）只应统计真实记录到首字节的请求。
// FirstResponseTime 初始化为 startTime-1s 哨兵：非流式响应从不记录首字节，
// 若无条件写出会得到 frt=-1000 混进 AVG，把平均值拖低甚至拖负。
func TestGenerateTextOtherInfoFrtOnlyWhenFirstResponseRecorded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	start := time.Unix(1700000000, 0)
	info := &relaycommon.RelayInfo{
		StartTime:         start,
		FirstResponseTime: start.Add(-time.Second), // 哨兵：未记录首字节
		ChannelMeta:       &relaycommon.ChannelMeta{},
	}
	other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 1, 0, 1)
	_, hasFrt := other["frt"]
	assert.False(t, hasFrt, "未记录首字节时不得写 frt（哨兵值会污染渠道统计平均值）")

	// 记录到首字节（流式首个 chunk）→ 写入正的 frt
	info.FirstResponseTime = start.Add(250 * time.Millisecond)
	other = GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 1, 0, 1)
	frt, ok := other["frt"].(float64)
	require.True(t, ok, "记录到首字节时必须写 frt")
	assert.InDelta(t, 250, frt, 0.0001)
}
