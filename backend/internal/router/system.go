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
func RegisterSystemRoutes(rg *gin.RouterGroup) {
	system := rg.Group("/system")
	system.Use(middleware.JWTAuth())
	system.Use(middleware.RequireSuperAdmin())

	// ========== 1. 统一审计 ==========
	audit := system.Group("/audit")
	{
		audit.GET("/logs", todo)                        // 聚合审计日志查询
		audit.GET("/logs/export", todo)                 // 导出审计日志
	}

	// ========== 2. 全局配置 ==========
	configs := system.Group("/configs")
	{
		configs.GET("", todo)                           // 获取配置列表
		configs.GET("/:group", todo)                    // 获取某分组配置
		configs.PUT("/:group/:key", todo)               // 更新单个配置
		configs.PUT("/:group", todo)                    // 批量更新分组配置
		configs.GET("/change-logs", todo)               // 配置变更记录
	}

	// ========== 3. 告警规则 ==========
	alertRules := system.Group("/alert-rules")
	{
		alertRules.POST("", todo)                       // 创建告警规则
		alertRules.GET("", todo)                        // 告警规则列表
		alertRules.GET("/:id", todo)                    // 告警规则详情
		alertRules.PUT("/:id", todo)                    // 更新告警规则
		alertRules.PATCH("/:id/toggle", todo)           // 启用/禁用规则
		alertRules.DELETE("/:id", todo)                 // 删除告警规则
	}

	// ========== 4. 告警事件 ==========
	alertEvents := system.Group("/alert-events")
	{
		alertEvents.GET("", todo)                       // 告警事件列表
		alertEvents.GET("/:id", todo)                   // 告警事件详情
		alertEvents.POST("/:id/handle", todo)           // 处理告警
		alertEvents.POST("/:id/ignore", todo)           // 忽略告警
	}

	// ========== 5. 运维仪表盘 ==========
	dashboard := system.Group("/dashboard")
	{
		dashboard.GET("/health", todo)                  // 平台健康状态
		dashboard.GET("/resources", todo)               // 资源使用情况
		dashboard.GET("/realtime", todo)                // 实时指标
	}

	// ========== 6. 平台统计 ==========
	statistics := system.Group("/statistics")
	{
		statistics.GET("/overview", todo)               // 统计总览
		statistics.GET("/trend", todo)                  // 趋势数据
		statistics.GET("/schools", todo)                // 学校活跃度排行
	}

	// ========== 7. 数据备份 ==========
	backups := system.Group("/backups")
	{
		backups.POST("/trigger", todo)                  // 手动触发备份
		backups.GET("", todo)                           // 备份列表
		backups.GET("/:id/download", todo)              // 下载备份文件
		backups.PUT("/config", todo)                    // 更新备份配置
		backups.GET("/config", todo)                    // 获取备份配置
	}
}
