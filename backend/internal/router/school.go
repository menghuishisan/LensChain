// school.go
// 模块02 — 学校与租户管理 路由注册
// 注册入驻申请、入驻审核、学校管理、学校配置、公开接口等路由
// 共 23 个端点
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package router

import (
	"github.com/gin-gonic/gin"

	handler "github.com/lenschain/backend/internal/handler/school"
	"github.com/lenschain/backend/internal/middleware"
)

// RegisterSchoolRoutes 注册学校与租户管理模块路由
func RegisterSchoolRoutes(
	rg *gin.RouterGroup,
	appH *handler.ApplicationHandler,
	schoolH *handler.SchoolHandler,
	ssoH *handler.SSOHandler,
) {
	// ========== 1. 入驻申请（公开，无需认证） ==========
	applications := rg.Group("/school-applications")
	{
		applications.POST("", appH.Submit)             // 提交入驻申请
		applications.GET("/query", appH.Query)         // 查询申请状态
		applications.POST("/:id/reapply", appH.Reapply) // 重新申请
	}

	// ========== 2. 入驻审核（超管） ==========
	adminApps := rg.Group("/admin/school-applications")
	adminApps.Use(middleware.JWTAuth())
	adminApps.Use(middleware.RequireSuperAdmin())
	{
		adminApps.GET("", appH.List)                   // 申请列表
		adminApps.GET("/:id", appH.GetByID)            // 申请详情
		adminApps.POST("/:id/approve", appH.Approve)   // 审核通过
		adminApps.POST("/:id/reject", appH.Reject)     // 审核拒绝
	}

	// ========== 3. 学校管理（超管） ==========
	adminSchools := rg.Group("/admin/schools")
	adminSchools.Use(middleware.JWTAuth())
	adminSchools.Use(middleware.RequireSuperAdmin())
	{
		adminSchools.GET("", schoolH.List)                  // 学校列表
		adminSchools.POST("", schoolH.Create)               // 后台直接创建学校
		adminSchools.GET("/:id", schoolH.GetByID)           // 学校详情
		adminSchools.PUT("/:id", schoolH.Update)            // 编辑学校信息
		adminSchools.PATCH("/:id/license", schoolH.SetLicense) // 设置有效期
		adminSchools.POST("/:id/freeze", schoolH.Freeze)    // 冻结学校
		adminSchools.POST("/:id/unfreeze", schoolH.Unfreeze) // 解冻学校
		adminSchools.POST("/:id/cancel", schoolH.Cancel)    // 注销学校
		adminSchools.POST("/:id/restore", schoolH.Restore)  // 恢复已注销学校
	}

	// ========== 4. 学校配置（校管） ==========
	school := rg.Group("/school")
	school.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	school.Use(middleware.RequireSchoolAdmin())
	{
		school.GET("/profile", schoolH.GetProfile)          // 获取本校信息
		school.PUT("/profile", schoolH.UpdateProfile)       // 编辑本校信息
		school.GET("/sso-config", ssoH.GetConfig)           // 获取SSO配置
		school.PUT("/sso-config", ssoH.UpdateConfig)        // 更新SSO配置
		school.POST("/sso-config/test", ssoH.TestConnection) // 测试SSO连接
		school.GET("/license", schoolH.GetLicenseStatus)    // 查看授权状态
	}

	// ========== 5. 公开接口（无需认证） ==========
	publicSchools := rg.Group("/schools")
	{
		publicSchools.GET("/sso-list", schoolH.GetSSOSchoolList) // 获取已配置SSO的学校列表
	}
}
