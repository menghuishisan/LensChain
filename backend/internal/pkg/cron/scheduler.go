// scheduler.go
// 定时任务调度器
// 基于 robfig/cron/v3 封装
// 管理文档中定义的 12+ 定时任务

package cron

import (
	cronlib "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局调度器
var scheduler *cronlib.Cron

// Init 初始化定时任务调度器
func Init() {
	scheduler = cronlib.New(
		cronlib.WithSeconds(), // 支持秒级精度
		cronlib.WithChain(
			cronlib.Recover(cronlib.DefaultLogger), // 任务 panic 恢复
		),
	)
	logger.L.Info("定时任务调度器已初始化")
}

// GetScheduler 获取调度器实例
func GetScheduler() *cronlib.Cron {
	return scheduler
}

// AddTask 注册定时任务
// spec 为 cron 表达式（支持秒级：秒 分 时 日 月 周）
// name 为任务名称（用于日志）
// fn 为任务函数
func AddTask(spec, name string, fn func()) (cronlib.EntryID, error) {
	id, err := scheduler.AddFunc(spec, func() {
		logger.L.Info("定时任务开始执行", zap.String("task", name))
		fn()
		logger.L.Info("定时任务执行完成", zap.String("task", name))
	})
	if err != nil {
		logger.L.Error("注册定时任务失败",
			zap.String("task", name),
			zap.String("spec", spec),
			zap.Error(err),
		)
		return 0, err
	}

	logger.L.Info("定时任务已注册",
		zap.String("task", name),
		zap.String("spec", spec),
	)
	return id, nil
}

// Start 启动调度器
func Start() {
	scheduler.Start()
	logger.L.Info("定时任务调度器已启动")
}

// Stop 停止调度器
func Stop() {
	if scheduler != nil {
		ctx := scheduler.Stop()
		<-ctx.Done()
		logger.L.Info("定时任务调度器已停止")
	}
}

// ---- 定时任务 cron 表达式常量 ----
// 对照 docs/数据库表总览.md 中的定时任务定义

const (
	// 模块03 — 课程与教学
	CronCourseStatusTransition = "0 0 * * * *" // 每小时 — 课程状态自动转换

	// 模块02 — 学校与租户管理
	CronSchoolExpiryCheck  = "0 0 1 * * *"  // 每天凌晨1:00 — 到期提醒检查
	CronSchoolBufferFreeze = "0 0 2 * * *"  // 每天凌晨2:00 — 缓冲期转冻结
	CronSchoolExpireBuffer = "0 30 0 * * *" // 每天凌晨0:30 — 到期转缓冲期

	// 模块04 — 实验环境
	CronExpIdleReclaim    = "0 */5 * * * *" // 每5分钟 — 空闲实验回收
	CronExpExpiredCleanup = "0 0 * * * *"   // 每小时 — 过期实验清理

	// 模块06 — 评测与成绩
	CronGPACacheRefresh   = "0 0 3 * * *" // 每天凌晨3:00 — GPA缓存刷新
	CronTranscriptCleanup = "0 0 4 * * *" // 每天凌晨4:00 — 成绩单文件清理

	// 模块07 — 通知与消息
	CronNotificationScan    = "0 * * * * *" // 每分钟 — 定时通知调度
	CronNotificationCleanup = "0 0 5 * * *" // 每天凌晨5:00 — 过期通知归档

	// 模块08 — 系统管理与监控
	CronAlertThreshold   = "0 * * * * *"    // 每分钟 — 阈值告警检测
	CronAlertEvent       = "0 */5 * * * *"  // 每5分钟 — 事件告警检测
	CronHealthCheck      = "*/30 * * * * *" // 每30秒 — 服务健康检查
	CronStatsAggregation = "0 0 1 * * *"    // 每天凌晨1:00 — 平台统计聚合
	CronAutoBackup       = "0 0 2 * * *"    // 每天凌晨2:00 — 自动备份
	CronBackupCleanup    = "0 0 3 * * *"    // 每天凌晨3:00 — 备份清理
	CronStatsDataCleanup = "0 0 4 1 * *"    // 每月1号凌晨4:00 — 统计数据清理
)
