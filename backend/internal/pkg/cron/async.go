// async.go
// 内部公共后台任务入口。
// 该文件为非周期性的业务后台任务提供统一 goroutine 启动、日志和 panic 兜底能力，
// 避免各模块直接散落裸 go func，导致异常恢复和执行日志不一致。

package cron

import (
	"context"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/pkg/logger"
)

// RunAsync 启动一次性后台任务。
// 适用于短信发送、审计日志、异步恢复等不需要 cron 表达式的后台动作。
func RunAsync(name string, fn func(ctx context.Context)) {
	if fn == nil {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.L.Error("后台任务发生panic",
					zap.String("task", name),
					zap.Any("panic", recovered),
				)
			}
		}()
		logger.L.Info("后台任务开始执行", zap.String("task", name))
		fn(context.Background())
		logger.L.Info("后台任务执行完成", zap.String("task", name))
	}()
}
