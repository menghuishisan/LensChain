// auth.go
// 模块01 — 用户与认证 路由注册
// 注册认证、用户管理、用户导入、个人中心、安全策略、日志等路由
// 共 26 个端点

package router

import (
	"github.com/gin-gonic/gin"

	handler "github.com/lenschain/backend/internal/handler/auth"
	"github.com/lenschain/backend/internal/middleware"
)

// RegisterAuthRoutes 注册用户与认证模块路由
// 接收 handler 实例，将路由绑定到实际处理方法
func RegisterAuthRoutes(rg *gin.RouterGroup, authH *handler.AuthHandler, userH *handler.UserHandler, securityH *handler.SecurityHandler) {
	// ========== 1. 认证（无需鉴权） ==========
	auth := rg.Group("/auth")
	{
		auth.POST("/login", authH.Login)                  // 手机号+密码登录
		auth.POST("/token/refresh", authH.RefreshToken)   // 刷新Token
		auth.GET("/sso/:school_id/login", authH.SSOLogin) // SSO登录跳转
		auth.GET("/sso/callback", authH.SSOCallback)      // SSO回调
	}

	// 认证（需要登录）
	authProtected := rg.Group("/auth")
	authProtected.Use(middleware.JWTAuth())
	{
		authProtected.POST("/logout", authH.Logout)                  // 登出
		authProtected.POST("/change-password", authH.ChangePassword) // 修改密码
	}

	// 首次登录强制改密（临时Token）
	authTemp := rg.Group("/auth")
	authTemp.Use(middleware.TempTokenAuth())
	{
		authTemp.POST("/force-change-password", authH.ForceChangePassword) // 首次登录强制改密
	}

	// ========== 2. 用户管理（超管/校管） ==========
	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	users.Use(middleware.RequireSchoolAdminOrSuperAdmin())
	{
		users.GET("", userH.List)                              // 用户列表
		users.GET("/:id", userH.Get)                           // 用户详情
		users.POST("", userH.Create)                           // 手动创建用户
		users.PUT("/:id", userH.Update)                        // 更新用户信息
		users.DELETE("/:id", userH.Delete)                     // 删除用户（软删除）
		users.PATCH("/:id/status", userH.UpdateStatus)         // 变更账号状态
		users.POST("/:id/reset-password", userH.ResetPassword) // 重置用户密码
		users.POST("/:id/unlock", userH.Unlock)                // 解锁账号
		users.POST("/batch-delete", userH.BatchDelete)         // 批量删除
	}

	// ========== 3. 用户导入（仅校管） ==========
	imports := rg.Group("/user-imports")
	imports.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	imports.Use(middleware.RequireSchoolAdmin())
	{
		imports.GET("/template", userH.DownloadTemplate)   // 下载导入模板
		imports.POST("/preview", userH.ImportPreview)      // 上传文件预览
		imports.POST("/execute", userH.ImportExecute)      // 确认执行导入
		imports.GET("/:id/failures", userH.ImportFailures) // 下载失败明细
	}

	// ========== 4. 个人中心（已登录用户） ==========
	profile := rg.Group("/profile")
	profile.Use(middleware.JWTAuth())
	{
		profile.GET("", userH.GetProfile)    // 获取个人信息
		profile.PUT("", userH.UpdateProfile) // 更新个人信息
	}

	// ========== 5. 安全策略（仅超管） ==========
	security := rg.Group("/security-policies")
	security.Use(middleware.JWTAuth())
	security.Use(middleware.RequireSuperAdmin())
	{
		security.GET("", securityH.GetSecurityPolicy)    // 获取安全策略配置
		security.PUT("", securityH.UpdateSecurityPolicy) // 更新安全策略配置
	}

	// ========== 6. 日志（超管/校管） ==========
	logs := rg.Group("")
	logs.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	logs.Use(middleware.RequireSchoolAdminOrSuperAdmin())
	{
		logs.GET("/login-logs", securityH.ListLoginLogs)         // 登录日志列表
		logs.GET("/operation-logs", securityH.ListOperationLogs) // 操作日志列表
	}
}
