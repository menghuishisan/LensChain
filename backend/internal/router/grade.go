// grade.go
// 模块06 — 评测与成绩 路由注册。
// 路由层只负责路径和中间件绑定，不承载任何业务判断。

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterGradeRoutes 注册评测与成绩模块路由。
func RegisterGradeRoutes(rg *gin.RouterGroup, h *GradeHandlers) {
	if h == nil || h.GradeHandler == nil {
		return
	}
	grades := rg.Group("/grades")
	grades.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 学期管理（校管） ==========
	semesters := grades.Group("/semesters")
	{
		// 列出当前学校学期及审核汇总统计。
		semesters.GET("", h.GradeHandler.ListSemesters)
		// 创建新学期。
		semesters.POST("", middleware.RequireSchoolAdmin(), h.GradeHandler.CreateSemester)
		// 更新学期基础信息。
		semesters.PUT("/:id", middleware.RequireSchoolAdmin(), h.GradeHandler.UpdateSemester)
		// 删除学期。
		semesters.DELETE("/:id", middleware.RequireSchoolAdmin(), h.GradeHandler.DeleteSemester)
		// 将指定学期设置为当前学期。
		semesters.PATCH("/:id/set-current", middleware.RequireSchoolAdmin(), h.GradeHandler.SetCurrentSemester)
	}

	// ========== 2. 等级映射（校管） ==========
	levelConfigs := grades.Group("/level-configs")
	levelConfigs.Use(middleware.RequireSchoolAdmin())
	{
		// 获取学校等级映射配置。
		levelConfigs.GET("", h.GradeHandler.GetLevelConfigs)
		// 更新学校等级映射配置。
		levelConfigs.PUT("", h.GradeHandler.UpdateLevelConfigs)
		// 重置为系统默认五级制配置。
		levelConfigs.POST("/reset-default", h.GradeHandler.ResetDefaultLevelConfigs)
	}

	// ========== 3. 成绩审核 ==========
	reviews := grades.Group("/reviews")
	{
		// 教师提交课程成绩审核。
		reviews.POST("", middleware.RequireTeacher(), h.GradeHandler.SubmitReview)
		// 查询成绩审核列表。
		reviews.GET("", middleware.RequireAdminOrTeacher(), h.GradeHandler.ListReviews)
		// 查看单条成绩审核详情。
		reviews.GET("/:id", middleware.RequireAdminOrTeacher(), h.GradeHandler.GetReview)
		// 学校管理员审核通过成绩。
		reviews.POST("/:id/approve", middleware.RequireSchoolAdmin(), h.GradeHandler.ApproveReview)
		// 学校管理员驳回成绩审核。
		reviews.POST("/:id/reject", middleware.RequireSchoolAdmin(), h.GradeHandler.RejectReview)
		// 学校管理员解锁已锁定成绩。
		reviews.POST("/:id/unlock", middleware.RequireSchoolAdmin(), h.GradeHandler.UnlockReview)
	}

	// ========== 4. 成绩查询 ==========
	// 查看指定学生的学期成绩。
	grades.GET("/students/:id/semester-grades", h.GradeHandler.GetStudentSemesterGrades)
	// 查看指定学生的 GPA 汇总。
	grades.GET("/students/:id/gpa", h.GradeHandler.GetStudentGPA)
	// 学生查看自己的学期成绩。
	grades.GET("/my/semester-grades", middleware.RequireStudent(), h.GradeHandler.GetMySemesterGrades)
	// 学生查看自己的 GPA 汇总。
	grades.GET("/my/gpa", middleware.RequireStudent(), h.GradeHandler.GetMyGPA)
	// 学生查看自己的学习概览。
	grades.GET("/my/learning-overview", middleware.RequireStudent(), h.GradeHandler.GetMyLearningOverview)

	// ========== 5. 成绩申诉 ==========
	appeals := grades.Group("/appeals")
	{
		// 学生提交成绩申诉。
		appeals.POST("", middleware.RequireStudent(), h.GradeHandler.CreateAppeal)
		// 查询成绩申诉列表。
		appeals.GET("", h.GradeHandler.ListAppeals)
		// 查看成绩申诉详情。
		appeals.GET("/:id", h.GradeHandler.GetAppeal)
		// 教师同意成绩申诉并调整成绩。
		appeals.POST("/:id/approve", middleware.RequireTeacher(), h.GradeHandler.ApproveAppeal)
		// 教师驳回成绩申诉。
		appeals.POST("/:id/reject", middleware.RequireTeacher(), h.GradeHandler.RejectAppeal)
	}

	// ========== 6. 学业预警（校管） ==========
	warnings := grades.Group("/warnings")
	warnings.Use(middleware.RequireSchoolAdmin())
	{
		// 查询学业预警列表。
		warnings.GET("", h.GradeHandler.ListWarnings)
		// 查看单条学业预警详情。
		warnings.GET("/:id", h.GradeHandler.GetWarning)
		// 处理学业预警。
		warnings.POST("/:id/handle", h.GradeHandler.HandleWarning)
	}

	// ========== 7. 预警配置（校管） ==========
	warningConfigs := grades.Group("/warning-configs")
	warningConfigs.Use(middleware.RequireSchoolAdmin())
	{
		// 获取学校预警阈值配置。
		warningConfigs.GET("", h.GradeHandler.GetWarningConfig)
		// 更新学校预警阈值配置。
		warningConfigs.PUT("", h.GradeHandler.UpdateWarningConfig)
	}

	// ========== 8. 成绩单 ==========
	transcripts := grades.Group("/transcripts")
	{
		// 生成成绩单 PDF。
		transcripts.POST("/generate", h.GradeHandler.GenerateTranscript)
		// 查询成绩单生成记录列表。
		transcripts.GET("", h.GradeHandler.ListTranscripts)
		// 下载指定成绩单。
		transcripts.GET("/:id/download", h.GradeHandler.DownloadTranscript)
	}

	// ========== 9. 成绩分析 ==========
	analytics := grades.Group("/analytics")
	{
		// 查询课程成绩分析。
		analytics.GET("/course/:id", middleware.RequireTeacher(), h.GradeHandler.GetCourseAnalytics)
		// 查询学校维度成绩分析。
		analytics.GET("/school", middleware.RequireSchoolAdmin(), h.GradeHandler.GetSchoolAnalytics)
		// 查询平台维度成绩总览。
		analytics.GET("/platform", middleware.RequireSuperAdmin(), h.GradeHandler.GetPlatformAnalytics)
	}
}
