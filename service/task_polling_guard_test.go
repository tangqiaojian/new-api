package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 渠道查询失败不得判负（回归：CacheGetChannel 抖动曾批量判负且不退款）
// ---------------------------------------------------------------------------

func seedInProgressTask(t *testing.T, userID, channelID, quota int, taskID string) *model.Task {
	t.Helper()
	seedUser(t, userID, 10000)
	task := makeTask(userID, channelID, quota, 0, BillingSourceWallet, 0)
	task.TaskID = taskID
	require.NoError(t, model.DB.Create(task).Error)
	return task
}

func getTaskStatusAndQuota(t *testing.T, taskID string) (string, int) {
	t.Helper()
	var reloaded model.Task
	require.NoError(t, model.DB.Where("task_id = ?", taskID).First(&reloaded).Error)
	return string(reloaded.Status), reloaded.Quota
}

func TestUpdateSunoTasksChannelLookupFailureKeepsTask(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const badChannelID = 990001
	task := seedInProgressTask(t, 41, badChannelID, 1000, "task_suno_channel_missing")

	err := updateSunoTasks(ctx, badChannelID, []string{task.GetUpstreamTaskID()}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	})
	require.Error(t, err, "missing channel must surface an error")

	status, quota := getTaskStatusAndQuota(t, task.TaskID)
	assert.Equal(t, model.TaskStatusInProgress, status, "任务不得被渠道抖动判负")
	assert.Equal(t, 1000, quota, "quota 必须保留，等待后续轮询或超时清扫退款")
	assert.Equal(t, int64(0), countLogs(t), "不得产生退款/消费日志")
}

func TestUpdateVideoTasksChannelLookupFailureKeepsTask(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const badChannelID = 990002
	task := seedInProgressTask(t, 42, badChannelID, 1000, "task_video_channel_missing")

	err := updateVideoTasks(ctx, constant.TaskPlatform("jimeng"), badChannelID, []string{task.GetUpstreamTaskID()}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	})
	require.Error(t, err, "missing channel must surface an error")

	status, quota := getTaskStatusAndQuota(t, task.TaskID)
	assert.Equal(t, model.TaskStatusInProgress, status, "任务不得被渠道抖动判负")
	assert.Equal(t, 1000, quota)
	assert.Equal(t, int64(0), countLogs(t))
}

// ---------------------------------------------------------------------------
// Suno 成功终态补齐渠道池用量（回归：Suno 成功从不结算池/订阅 token 用量）
// ---------------------------------------------------------------------------

type sunoSuccessAdaptor struct{}

func (a sunoSuccessAdaptor) Init(*relaycommon.RelayInfo) {}
func (a sunoSuccessAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	body := `{"code":"success","message":"","data":[{"task_id":"task_suno_pool_settle","status":"SUCCESS","data":{}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}
func (a sunoSuccessAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return nil, nil
}
func (a sunoSuccessAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return 0
}

func TestUpdateSunoTasksSuccessSettlesPoolUsage(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const channelID = 8300
	baseURL := "http://localhost"
	ch := &model.Channel{Id: channelID, Name: "suno-ch", Key: "sk-test", Status: common.ChannelStatusEnabled, BaseURL: &baseURL}
	require.NoError(t, model.DB.Create(ch).Error)
	pool := seedChannelPoolUsage(t, channelID, false)
	task := seedInProgressTask(t, 43, channelID, 60, "task_suno_pool_settle")

	GetTaskAdaptorFunc = func(platform constant.TaskPlatform) TaskPollingAdaptor {
		return sunoSuccessAdaptor{}
	}
	t.Cleanup(func() { GetTaskAdaptorFunc = nil })

	require.NoError(t, updateSunoTasks(ctx, channelID, []string{task.GetUpstreamTaskID()}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	status, _ := getTaskStatusAndQuota(t, task.TaskID)
	assert.Equal(t, model.TaskStatusSuccess, status)
	assert.EqualValues(t, 60, getChannelPoolUsage(t, pool.Id).AmountUsed,
		"按次计费的 Suno 任务成功后必须把预扣额度落到渠道池用量")
}

// ---------------------------------------------------------------------------
// 落库失败补偿退款（回归：响应已下发 + 结算已完成 + Insert 失败 = 资金孤儿）
// RefundTaskQuota 必须能在 task 行不存在的情况下完成资金退还。
// ---------------------------------------------------------------------------

func TestRefundTaskQuotaWithoutPersistedRow(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID = 44
	seedUser(t, userID, 10000)
	// 模拟 submit 预扣 + 结算已生效：钱包已扣 60，用量统计已加 60
	require.NoError(t, model.DecreaseUserQuota(userID, 60, false))
	model.UpdateUserUsedQuota(userID, 60)

	// 故意不入库 —— 模拟 task.Insert() 失败后的补偿路径
	task := makeTask(userID, 0, 60, 0, BillingSourceWallet, 0)
	task.TaskID = "task_insert_failed_compensate"

	ok := RefundTaskQuota(ctx, task, "task insert failed, refunding settled quota")
	assert.True(t, ok, "资金来源退还应成功（UpdateBillingState 失败不影响退款）")
	assert.Equal(t, 10000, getUserQuota(t, userID), "钱包必须退回已结算额度")
	usedQuota, _ := getUserUsageAccounting(t, userID)
	assert.Equal(t, 0, usedQuota, "用量统计必须回减")

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
	assert.Equal(t, 60, log.Quota)
}

// ---------------------------------------------------------------------------
// 用户主动 fetch 触发终态迁移的计费收尾（回归：Gemini/Vertex tryRealtimeFetch
// CAS 到终态但不结算/不退款，任务退出轮询队列后费用永远挂账）
// ---------------------------------------------------------------------------

func TestSettleTerminalTaskBillingFailureRefunds(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, channelID = 45, 8450
	seedUser(t, userID, 9000) // 已预扣 1000 后的余额
	seedChannel(t, channelID)
	seedChargedAccounting(t, userID, channelID, 0, 1000, 1)

	task := makeTask(userID, channelID, 1000, 0, BillingSourceWallet, 0)
	task.TaskID = "task_terminal_fetch_fail"
	require.NoError(t, model.DB.Create(task).Error)

	// 模拟 tryRealtimeFetch：CAS 获胜后任务已到失败终态
	task.Status = model.TaskStatusFailure
	task.FailReason = "upstream generation failed"
	SettleTerminalTaskBilling(ctx, task, &relaycommon.TaskInfo{Status: model.TaskStatusFailure, Reason: task.FailReason})

	assert.Equal(t, 10000, getUserQuota(t, userID), "失败终态必须退还预扣费")
	assert.Equal(t, 0, task.Quota)
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, 0, reloaded.Quota)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}

func TestSettleTerminalTaskBillingSuccessSettles(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, channelID = 46, 8460
	seedUser(t, userID, 9940) // 已预扣 60 后的余额
	seedChannel(t, channelID)
	pool := seedChannelPoolUsage(t, channelID, false)
	seedChargedAccounting(t, userID, channelID, 0, 60, 1)

	task := makeTask(userID, channelID, 60, 0, BillingSourceWallet, 0)
	task.TaskID = "task_terminal_fetch_success"
	task.Platform = constant.TaskPlatform("gemini")
	require.NoError(t, model.DB.Create(task).Error)

	// adaptor 完成时把实际额度调整为 90（补扣 30）
	GetTaskAdaptorFunc = func(platform constant.TaskPlatform) TaskPollingAdaptor {
		return fixedTaskBillingAdaptor{actualQuota: 90}
	}
	t.Cleanup(func() { GetTaskAdaptorFunc = nil })

	task.Status = model.TaskStatusSuccess
	SettleTerminalTaskBilling(ctx, task, &relaycommon.TaskInfo{Status: model.TaskStatusSuccess})

	assert.Equal(t, 9910, getUserQuota(t, userID), "成功终态按 adaptor 实际额度补扣差额")
	assert.Equal(t, 90, task.Quota)
	assert.EqualValues(t, 90, getChannelPoolUsage(t, pool.Id).AmountUsed,
		"成功终态必须把最终额度落到渠道池用量")
}
