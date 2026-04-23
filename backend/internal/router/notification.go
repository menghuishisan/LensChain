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
func RegisterNotificationRoutes(rg *gin.RouterGroup, handlers *NotificationHandlers) {
	if handlers == nil || handlers.NotificationHandler == nil {
		return
	}
	notifications := rg.Group("/notifications")
	notifications.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 站内信（登录用户） ==========
	inbox := notifications.Group("/inbox")
	{
		inbox.GET("", handlers.NotificationHandler.ListInbox)                     // 收件箱列表
		inbox.GET("/:id", handlers.NotificationHandler.GetInboxDetail)            // 消息详情
		inbox.PATCH("/:id/read", handlers.NotificationHandler.MarkInboxRead)      // 标记已读
		inbox.POST("/batch-read", handlers.NotificationHandler.BatchReadInbox)    // 批量标记已读
		inbox.POST("/read-all", handlers.NotificationHandler.ReadAllInbox)        // 全部标记已读
		inbox.DELETE("/:id", handlers.NotificationHandler.DeleteInbox)            // 删除消息
		inbox.GET("/unread-count", handlers.NotificationHandler.GetUnreadCount)   // 未读消息计数
	}

	// ========== 2. 系统公告 ==========
	announcements := notifications.Group("/announcements")
	{
		announcements.POST("", middleware.RequireSuperAdmin(), handlers.NotificationHandler.CreateAnnouncement)              // 创建公告
		announcements.GET("", handlers.NotificationHandler.ListAnnouncements)                                                // 公告列表
		announcements.GET("/:id", handlers.NotificationHandler.GetAnnouncement)                                              // 公告详情
		announcements.PUT("/:id", middleware.RequireSuperAdmin(), handlers.NotificationHandler.UpdateAnnouncement)           // 编辑公告
		announcements.POST("/:id/publish", middleware.RequireSuperAdmin(), handlers.NotificationHandler.PublishAnnouncement) // 发布公告
		announcements.POST("/:id/unpublish", middleware.RequireSuperAdmin(), handlers.NotificationHandler.UnpublishAnnouncement) // 下架公告
		announcements.DELETE("/:id", middleware.RequireSuperAdmin(), handlers.NotificationHandler.DeleteAnnouncement)        // 删除公告
	}

	// ========== 3. 定向通知（管理员/教师） ==========
	notifications.POST("/send", middleware.RequireAdminOrTeacher(), handlers.NotificationHandler.SendDirectNotification) // 发送定向通知

	// ========== 4. 通知偏好（登录用户） ==========
	preferences := notifications.Group("/preferences")
	{
		preferences.GET("", handlers.NotificationHandler.GetPreferences)    // 获取通知偏好
		preferences.PUT("", handlers.NotificationHandler.UpdatePreferences)  // 更新通知偏好
	}

	// ========== 5. 消息模板（超管） ==========
	templates := notifications.Group("/templates")
	templates.Use(middleware.RequireSuperAdmin())
	{
		templates.GET("", handlers.NotificationHandler.ListTemplates)          // 模板列表
		templates.GET("/:id", handlers.NotificationHandler.GetTemplate)        // 模板详情
		templates.PUT("/:id", handlers.NotificationHandler.UpdateTemplate)     // 更新模板
		templates.POST("/:id/preview", handlers.NotificationHandler.PreviewTemplate) // 预览模板
	}

	// ========== 6. 统计（超管） ==========
	notifications.GET("/statistics", middleware.RequireSuperAdmin(), handlers.NotificationHandler.GetStatistics) // 消息统计
}
