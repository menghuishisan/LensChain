// system.go
// 模块08 — 系统管理与监控 路由注册
// 注册统一审计、全局配置、告警规则、告警事件、运维仪表盘、
// 平台统计、数据备份等路由
// 共 28 个端点（全部仅超级管理员可访问）

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterSystemRoutes 注册系统管理与监控模块路由
func RegisterSystemRoutes(rg *gin.RouterGroup, h *SystemHandlers) {
	if h == nil || h.SystemHandler == nil {
		return
	}
	system := rg.Group("/system")
	system.Use(middleware.JWTAuth())
	system.Use(middleware.RequireSuperAdmin())

	// ========== 1. 统一审计 ==========
	audit := system.Group("/audit")
	{
		audit.GET("/logs", h.SystemHandler.ListAuditLogs)          // 聚合审计日志查询
		audit.GET("/logs/export", h.SystemHandler.ExportAuditLogs) // 导出审计日志
	}

	// ========== 2. 全局配置 ==========
	configs := system.Group("/configs")
	{
		configs.GET("", h.SystemHandler.GetConfigs)                       // 获取配置列表
		configs.GET("/change-logs", h.SystemHandler.ListConfigChangeLogs) // 配置变更记录
		configs.GET("/:group", h.SystemHandler.GetConfigGroup)            // 获取某分组配置
		configs.PUT("/:group/:key", h.SystemHandler.UpdateConfig)         // 更新单个配置
		configs.PUT("/:group", h.SystemHandler.BatchUpdateConfigs)        // 批量更新分组配置
	}

	// ========== 3. 告警规则 ==========
	alertRules := system.Group("/alert-rules")
	{
		alertRules.POST("", h.SystemHandler.CreateAlertRule)             // 创建告警规则
		alertRules.GET("", h.SystemHandler.ListAlertRules)               // 告警规则列表
		alertRules.GET("/:id", h.SystemHandler.GetAlertRule)             // 告警规则详情
		alertRules.PUT("/:id", h.SystemHandler.UpdateAlertRule)          // 更新告警规则
		alertRules.PATCH("/:id/toggle", h.SystemHandler.ToggleAlertRule) // 启用/禁用规则
		alertRules.DELETE("/:id", h.SystemHandler.DeleteAlertRule)       // 删除告警规则
	}

	// ========== 4. 告警事件 ==========
	alertEvents := system.Group("/alert-events")
	{
		alertEvents.GET("", h.SystemHandler.ListAlertEvents)              // 告警事件列表
		alertEvents.GET("/:id", h.SystemHandler.GetAlertEvent)            // 告警事件详情
		alertEvents.POST("/:id/handle", h.SystemHandler.HandleAlertEvent) // 处理告警
		alertEvents.POST("/:id/ignore", h.SystemHandler.IgnoreAlertEvent) // 忽略告警
	}

	// ========== 5. 运维仪表盘 ==========
	dashboard := system.Group("/dashboard")
	{
		dashboard.GET("/health", h.SystemHandler.GetDashboardHealth)       // 平台健康状态
		dashboard.GET("/resources", h.SystemHandler.GetDashboardResources) // 资源使用情况
		dashboard.GET("/realtime", h.SystemHandler.GetDashboardRealtime)   // 实时指标
	}

	// ========== 6. 平台统计 ==========
	statistics := system.Group("/statistics")
	{
		statistics.GET("/overview", h.SystemHandler.GetStatisticsOverview) // 统计总览
		statistics.GET("/trend", h.SystemHandler.GetStatisticsTrend)       // 趋势数据
		statistics.GET("/schools", h.SystemHandler.GetSchoolStatistics)    // 学校活跃度排行
	}

	// ========== 7. 数据备份 ==========
	backups := system.Group("/backups")
	{
		backups.POST("/trigger", h.SystemHandler.TriggerBackup)      // 手动触发备份
		backups.GET("", h.SystemHandler.ListBackups)                 // 备份列表
		backups.GET("/:id/download", h.SystemHandler.DownloadBackup) // 下载备份文件
		backups.PUT("/config", h.SystemHandler.UpdateBackupConfig)   // 更新备份配置
		backups.GET("/config", h.SystemHandler.GetBackupConfig)      // 获取备份配置
	}
}
