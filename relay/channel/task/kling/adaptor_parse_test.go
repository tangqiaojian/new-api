package kling

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 回归：FinalUnitDeduction 是上游扣费单元数而非 token。它曾填入
// TotalTokens/CompletionTokens 参与按倍率重算——对倍率计费的 Kling 模型，
// 单元数被当作 token 乘以模型倍率，预扣费几乎全额退还（交付了视频却接近免费）。
func TestParseTaskResultSucceedIgnoresUnitDeduction(t *testing.T) {
	body := `{"code":0,"message":"ok","data":{"task_id":"abc","task_status":"succeed","task_result":{"videos":[{"id":"v1","url":"http://x/v.mp4","duration":"5"}]},"final_unit_deduction":"10"}}`

	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(body))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "http://x/v.mp4", result.Url)
	assert.Zero(t, result.TotalTokens, "FinalUnitDeduction 不得进入 token 重算")
	assert.Zero(t, result.CompletionTokens)
}
