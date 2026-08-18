package jimeng

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 回归：火山引擎 CVSync2AsyncGetResult 的 generating 中间态曾被忽略，空状态
// 在轮询侧被当作失败并触发退款（上游仍在正常生成）。not_found/expired 是上游
// 明确终态，必须判负；失败响应（code != 10000）不得被 Data.Status 覆盖回成功。
func TestParseTaskResultStatusMapping(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		wantStatus   string
		wantProgress string
	}{
		{"queued", `{"code":10000,"data":{"status":"in_queue"}}`, model.TaskStatusQueued, "10%"},
		{"generating", `{"code":10000,"data":{"status":"generating"}}`, model.TaskStatusInProgress, "50%"},
		{"done", `{"code":10000,"data":{"status":"done","video_url":"http://x/v.mp4"}}`, model.TaskStatusSuccess, "100%"},
		{"not_found", `{"code":10000,"data":{"status":"not_found"}}`, model.TaskStatusFailure, "100%"},
		{"expired", `{"code":10000,"data":{"status":"expired"}}`, model.TaskStatusFailure, "100%"},
	}

	adaptor := &TaskAdaptor{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := adaptor.ParseTaskResult([]byte(tc.body))
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, result.Status)
			assert.Equal(t, tc.wantProgress, result.Progress)
		})
	}
}

func TestParseTaskResultFailureNotOverriddenByDataStatus(t *testing.T) {
	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(`{"code":50000,"message":"boom","data":{"status":"done"}}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, result.Status, "失败响应不得被 Data.Status 覆盖")
	assert.Equal(t, "boom", result.Reason)
}
