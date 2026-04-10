// grade.go
// 模块06 — 评测与成绩 路由注册
// 注册学期管理、等级映射、成绩审核、成绩查询、成绩申诉、
// 学业预警、成绩单、成绩分析等路由
// 共 34 个端点

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterGradeRoutes 注册评测与成绩模块路由
func RegisterGradeRoutes(rg *gin.RouterGroup) {
	grades := rg.Group("/grades")
	grades.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 学期管理（校管） ==========
	semesters := grades.Group("/semesters")
	{
		semesters.GET("", todo)                                                       // 学期列表（登录用户）
		semesters.POST("", middleware.RequireSchoolAdmin(), todo)                      // 创建学期
		semesters.PUT("/:id", middleware.RequireSchoolAdmin(), todo)                   // 更新学期
		semesters.DELETE("/:id", middleware.RequireSchoolAdmin(), todo)                // 删除学期
		semesters.PATCH("/:id/set-current", middleware.RequireSchoolAdmin(), todo)     // 设为当前学期
	}

	// ========== 2. 等级映射（校管） ==========
	levelConfigs := grades.Group("/level-configs")
	levelConfigs.Use(middleware.RequireSchoolAdmin())
	{
		levelConfigs.GET("", todo)                      // 获取等级映射配置
		levelConfigs.PUT("", todo)                      // 更新等级映射配置
		levelConfigs.POST("/reset-default", todo)       // 重置为默认配置
	}

	// ========== 3. 成绩审核 ==========
	reviews := grades.Group("/reviews")
	{
		reviews.POST("", middleware.RequireTeacher(), todo)                            // 提交成绩审核
		reviews.GET("", middleware.RequireAdminOrTeacher(), todo)                      // 审核列表
		reviews.GET("/:id", middleware.RequireAdminOrTeacher(), todo)                  // 审核详情
		reviews.POST("/:id/approve", middleware.RequireSchoolAdmin(), todo)            // 审核通过
		reviews.POST("/:id/reject", middleware.RequireSchoolAdmin(), todo)             // 审核驳回
		reviews.POST("/:id/unlock", middleware.RequireSchoolAdmin(), todo)             // 解锁成绩
	}

	// ========== 4. 成绩查询 ==========
	grades.GET("/students/:id/semester-grades", todo)                                 // 学生学期成绩
	grades.GET("/students/:id/gpa", todo)                                             // 学生GPA
	grades.GET("/my/semester-grades", middleware.RequireStudent(), todo)               // 我的学期成绩
	grades.GET("/my/gpa", middleware.RequireStudent(), todo)                           // 我的GPA

	// ========== 5. 成绩申诉 ==========
	appeals := grades.Group("/appeals")
	{
		appeals.POST("", middleware.RequireStudent(), todo)                            // 提交申诉
		appeals.GET("", todo)                                                         // 申诉列表
		appeals.GET("/:id", todo)                                                     // 申诉详情
		appeals.POST("/:id/approve", middleware.RequireTeacher(), todo)                // 同意申诉
		appeals.POST("/:id/reject", middleware.RequireTeacher(), todo)                 // 驳回申诉
	}

	// ========== 6. 学业预警（校管） ==========
	warnings := grades.Group("/warnings")
	warnings.Use(middleware.RequireSchoolAdmin())
	{
		warnings.GET("", todo)                          // 预警列表
		warnings.GET("/:id", todo)                      // 预警详情
		warnings.POST("/:id/handle", todo)              // 处理预警
	}

	warningConfigs := grades.Group("/warning-configs")
	warningConfigs.Use(middleware.RequireSchoolAdmin())
	{
		warningConfigs.GET("", todo)                    // 获取预警配置
		warningConfigs.PUT("", todo)                    // 更新预警配置
	}

	// ========== 7. 成绩单 ==========
	transcripts := grades.Group("/transcripts")
	{
		transcripts.POST("/generate", todo)             // 生成成绩单
		transcripts.GET("", todo)                       // 成绩单列表
		transcripts.GET("/:id/download", todo)          // 下载成绩单
	}

	// ========== 8. 成绩分析 ==========
	analytics := grades.Group("/analytics")
	{
		analytics.GET("/course/:id", middleware.RequireTeacher(), todo)                // 课程成绩分析
		analytics.GET("/school", middleware.RequireSchoolAdmin(), todo)                // 全校成绩分析
		analytics.GET("/platform", middleware.RequireSuperAdmin(), todo)               // 平台成绩总览
	}
}
