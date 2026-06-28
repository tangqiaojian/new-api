package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	weeklyQuotaResetTickInterval = 5 * time.Minute
	weeklyQuotaResetBatchSize    = 500
)

var (
	weeklyQuotaResetOnce    sync.Once
	weeklyQuotaResetRunning atomic.Bool
)

// StartWeeklyQuotaResetTask 启动周额度定时重置任务。
// 每隔几分钟检查一次是否有用户的周额度需要重置（到期后重置已用量为0）。
func StartWeeklyQuotaResetTask() {
	weeklyQuotaResetOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("weekly quota reset task started: tick=%s", weeklyQuotaResetTickInterval))
			ticker := time.NewTicker(weeklyQuotaResetTickInterval)
			defer ticker.Stop()

			runWeeklyQuotaResetOnce()
			for range ticker.C {
				runWeeklyQuotaResetOnce()
			}
		})
	})
}

func runWeeklyQuotaResetOnce() {
	if !weeklyQuotaResetRunning.CompareAndSwap(false, true) {
		return
	}
	defer weeklyQuotaResetRunning.Store(false)

	ctx := context.Background()
	totalReset := 0
	for {
		n, err := model.ResetDueWeeklyQuotas(weeklyQuotaResetBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("weekly quota reset task failed: %v", err))
			return
		}
		if n == 0 {
			break
		}
		totalReset += n
		if n < weeklyQuotaResetBatchSize {
			break
		}
	}
	if common.DebugEnabled && totalReset > 0 {
		logger.LogDebug(ctx, "weekly quota reset: count=%d", totalReset)
	}
}
