// notification.go
// 模块07 — 通知与消息 路由注册
// 注册站内信、系统公告、定向通知、通知偏好、消息模板、统计等路由
// 共 22 个外部端点 + 1 个内部端点 + 1 个 WebSocket 端点

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterNotificationRoutes 注册通知与消息模块路由
func RegisterNotificationRoutes(rg *gin.RouterGroup) {
	notifications := rg.Group("/notifications")
	notifications.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 站内信（登录用户） ==========
	inbox := notifications.Group("/inbox")
	{
		inbox.GET("", todo)                             // 收件箱列表
		inbox.GET("/:id", todo)                         // 消息详情
		inbox.PATCH("/:id/read", todo)                  // 标记已读
		inbox.POST("/batch-read", todo)                 // 批量标记已读
		inbox.POST("/read-all", todo)                   // 全部标记已读
		inbox.DELETE("/:id", todo)                      // 删除消息
		inbox.GET("/unread-count", todo)                // 未读消息计数
	}

	// ========== 2. 系统公告 ==========
	announcements := notifications.Group("/announcements")
	{
		announcements.POST("", middleware.RequireSuperAdmin(), todo)                   // 创建公告
		announcements.GET("", todo)                                                   // 公告列表
		announcements.GET("/:id", todo)                                               // 公告详情
		announcements.PUT("/:id", middleware.RequireSuperAdmin(), todo)                // 编辑公告
		announcements.POST("/:id/publish", middleware.RequireSuperAdmin(), todo)       // 发布公告
		announcements.POST("/:id/unpublish", middleware.RequireSuperAdmin(), todo)     // 下架公告
		announcements.DELETE("/:id", middleware.RequireSuperAdmin(), todo)             // 删除公告
	}

	// ========== 3. 定向通知（管理员/教师） ==========
	notifications.POST("/send", middleware.RequireAdminOrTeacher(), todo)              // 发送定向通知

	// ========== 4. 通知偏好（登录用户） ==========
	preferences := notifications.Group("/preferences")
	{
		preferences.GET("", todo)                       // 获取通知偏好
		preferences.PUT("", todo)                       // 更新通知偏好
	}

	// ========== 5. 消息模板（超管） ==========
	templates := notifications.Group("/templates")
	templates.Use(middleware.RequireSuperAdmin())
	{
		templates.GET("", todo)                         // 模板列表
		templates.GET("/:id", todo)                     // 模板详情
		templates.PUT("/:id", todo)                     // 更新模板
		templates.POST("/:id/preview", todo)            // 预览模板
	}

	// ========== 6. 统计（超管） ==========
	notifications.GET("/statistics", middleware.RequireSuperAdmin(), todo)             // 消息统计
}
