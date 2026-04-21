// context_helper.go
// 模块04 — 实验环境：Service 层上下文辅助
// 负责为请求链路派生的异步任务提供“保留上下文值、去除取消信号”的上下文构造，避免直接丢失调用链信息

package experiment

import "context"

// detachContext 从当前调用上下文派生异步任务上下文。
// 请求触发的异步环境编排需要保留上游上下文中的租户、链路追踪等值，但不能受原请求取消影响。
func detachContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}
