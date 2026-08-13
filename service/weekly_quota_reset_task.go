package service

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/model"
)

const (
	weeklyQuotaResetInterval  = 5 * time.Minute
	weeklyQuotaResetBatchSize = 500
)

type weeklyQuotaResetHandler struct{}

type WeeklyQuotaResetResult struct {
	ResetCount int `json:"reset_count"`
}

func (weeklyQuotaResetHandler) Type() string { return model.SystemTaskTypeWeeklyQuotaReset }

func (weeklyQuotaResetHandler) Enabled() bool { return true }

func (weeklyQuotaResetHandler) Interval() time.Duration { return weeklyQuotaResetInterval }

func (weeklyQuotaResetHandler) NewPayload() any { return nil }

func (weeklyQuotaResetHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	totalReset := 0
	for {
		select {
		case <-ctx.Done():
			failSystemTask(task, runnerID, ctx.Err())
			return
		default:
		}

		resetCount, err := model.ResetDueWeeklyQuotas(weeklyQuotaResetBatchSize)
		if err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		totalReset += resetCount
		if resetCount < weeklyQuotaResetBatchSize {
			break
		}
	}

	result := WeeklyQuotaResetResult{ResetCount: totalReset}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, result, ""); err != nil {
		logSystemTaskLockError(ctx, task, err)
	}
}

func init() {
	RegisterSystemTaskHandler(weeklyQuotaResetHandler{})
}
