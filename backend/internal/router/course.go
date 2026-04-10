// course.go
// 模块03 — 课程与教学 路由注册
// 注册课程管理、章节课时、选课、作业、提交批改、学习进度、课程表、
// 公告、讨论区、评价、成绩管理、共享课程库、课程统计、学生视角等路由
// 共 76 个端点

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterCourseRoutes 注册课程与教学模块路由
func RegisterCourseRoutes(rg *gin.RouterGroup) {
	// ========== 1. 课程管理（教师） ==========
	courses := rg.Group("/courses")
	courses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		// 教师操作
		courses.POST("", middleware.RequireTeacher(), todo)                          // 创建课程
		courses.GET("", todo)                                                        // 课程列表
		courses.GET("/:id", todo)                                                    // 课程详情
		courses.PUT("/:id", middleware.RequireTeacher(), todo)                        // 编辑课程信息
		courses.DELETE("/:id", middleware.RequireTeacher(), todo)                     // 删除课程（仅草稿）
		courses.POST("/:id/publish", middleware.RequireTeacher(), todo)               // 发布课程
		courses.POST("/:id/end", middleware.RequireTeacher(), todo)                   // 结束课程
		courses.POST("/:id/archive", middleware.RequireTeacher(), todo)               // 归档课程
		courses.POST("/:id/clone", middleware.RequireTeacher(), todo)                 // 克隆课程
		courses.PATCH("/:id/share", middleware.RequireTeacher(), todo)                // 设置共享状态
		courses.POST("/:id/invite-code/refresh", middleware.RequireTeacher(), todo)   // 刷新邀请码

		// ========== 2. 章节与课时 ==========
		courses.GET("/:id/chapters", todo)                                           // 获取章节列表（含课时）
		courses.POST("/:id/chapters", middleware.RequireTeacher(), todo)              // 创建章节
		courses.PUT("/:id/chapters/sort", middleware.RequireTeacher(), todo)          // 章节排序

		// ========== 3. 选课管理 ==========
		courses.POST("/join", middleware.RequireStudent(), todo)                      // 通过邀请码加入课程
		courses.POST("/:id/students", middleware.RequireTeacher(), todo)              // 教师添加学生
		courses.POST("/:id/students/batch", middleware.RequireTeacher(), todo)        // 批量添加学生
		courses.DELETE("/:id/students/:student_id", middleware.RequireTeacher(), todo) // 移除学生
		courses.GET("/:id/students", middleware.RequireTeacher(), todo)               // 课程学生列表

		// ========== 4. 作业管理 ==========
		courses.POST("/:id/assignments", middleware.RequireTeacher(), todo)           // 创建作业
		courses.GET("/:id/assignments", todo)                                        // 作业列表

		// ========== 5. 学习进度 ==========
		courses.GET("/:id/my-progress", middleware.RequireStudent(), todo)            // 我的课程学习进度
		courses.GET("/:id/students-progress", middleware.RequireTeacher(), todo)      // 所有学生学习进度

		// ========== 6. 课程表 ==========
		courses.PUT("/:id/schedules", middleware.RequireTeacher(), todo)              // 设置课程表
		courses.GET("/:id/schedules", todo)                                          // 获取课程表

		// ========== 7. 公告 ==========
		courses.POST("/:id/announcements", middleware.RequireTeacher(), todo)         // 发布公告
		courses.GET("/:id/announcements", todo)                                      // 公告列表

		// ========== 8. 讨论区 ==========
		courses.POST("/:id/discussions", todo)                                       // 发帖
		courses.GET("/:id/discussions", todo)                                        // 帖子列表

		// ========== 9. 课程评价 ==========
		courses.POST("/:id/evaluations", middleware.RequireStudent(), todo)           // 提交评价
		courses.GET("/:id/evaluations", todo)                                        // 评价列表

		// ========== 10. 成绩管理 ==========
		courses.PUT("/:id/grade-config", middleware.RequireTeacher(), todo)           // 配置成绩权重
		courses.GET("/:id/grade-config", middleware.RequireTeacher(), todo)           // 获取成绩权重配置
		courses.GET("/:id/grades", middleware.RequireTeacher(), todo)                 // 成绩汇总表
		courses.PATCH("/:id/grades/:student_id", middleware.RequireTeacher(), todo)   // 手动调整成绩
		courses.GET("/:id/grades/export", middleware.RequireTeacher(), todo)          // 导出成绩单
		courses.GET("/:id/my-grades", middleware.RequireStudent(), todo)              // 我的成绩

		// ========== 11. 课程统计 ==========
		courses.GET("/:id/statistics/overview", middleware.RequireTeacher(), todo)    // 课程整体统计
		courses.GET("/:id/statistics/assignments", middleware.RequireTeacher(), todo) // 作业统计
		courses.GET("/:id/statistics/export", middleware.RequireTeacher(), todo)      // 导出统计报告
	}

	// ========== 章节（独立路径） ==========
	chapters := rg.Group("/chapters")
	chapters.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	chapters.Use(middleware.RequireTeacher())
	{
		chapters.PUT("/:id", todo)                      // 编辑章节
		chapters.DELETE("/:id", todo)                   // 删除章节
		chapters.POST("/:id/lessons", todo)             // 创建课时
		chapters.PUT("/:id/lessons/sort", todo)         // 课时排序
	}

	// ========== 课时（独立路径） ==========
	lessons := rg.Group("/lessons")
	lessons.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		lessons.GET("/:id", todo)                                                    // 课时详情
		lessons.PUT("/:id", middleware.RequireTeacher(), todo)                        // 编辑课时
		lessons.DELETE("/:id", middleware.RequireTeacher(), todo)                     // 删除课时
		lessons.POST("/:id/attachments", middleware.RequireTeacher(), todo)           // 上传课时附件
		lessons.POST("/:id/progress", middleware.RequireStudent(), todo)              // 更新学习进度
	}

	// ========== 课时附件（独立路径） ==========
	lessonAttachments := rg.Group("/lesson-attachments")
	lessonAttachments.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	lessonAttachments.Use(middleware.RequireTeacher())
	{
		lessonAttachments.DELETE("/:id", todo)           // 删除附件
	}

	// ========== 作业（独立路径） ==========
	assignments := rg.Group("/assignments")
	assignments.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		assignments.GET("/:id", todo)                                                // 作业详情（含题目）
		assignments.PUT("/:id", middleware.RequireTeacher(), todo)                    // 编辑作业
		assignments.DELETE("/:id", middleware.RequireTeacher(), todo)                 // 删除作业
		assignments.POST("/:id/publish", middleware.RequireTeacher(), todo)           // 发布作业
		assignments.POST("/:id/questions", middleware.RequireTeacher(), todo)         // 添加题目
		assignments.POST("/:id/submit", middleware.RequireStudent(), todo)            // 学生提交作业
		assignments.GET("/:id/my-submissions", middleware.RequireStudent(), todo)     // 我的提交记录
		assignments.GET("/:id/submissions", middleware.RequireTeacher(), todo)        // 所有学生提交列表
	}

	// ========== 作业题目（独立路径） ==========
	assignmentQuestions := rg.Group("/assignment-questions")
	assignmentQuestions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	assignmentQuestions.Use(middleware.RequireTeacher())
	{
		assignmentQuestions.PUT("/:id", todo)            // 编辑题目
		assignmentQuestions.DELETE("/:id", todo)         // 删除题目
	}

	// ========== 提交（独立路径） ==========
	submissions := rg.Group("/submissions")
	submissions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		submissions.GET("/:id", todo)                                                // 提交详情
		submissions.POST("/:id/grade", middleware.RequireTeacher(), todo)             // 批改提交
	}

	// ========== 讨论区（独立路径） ==========
	discussions := rg.Group("/discussions")
	discussions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		discussions.GET("/:id", todo)                                                // 帖子详情（含回复）
		discussions.DELETE("/:id", todo)                                              // 删除帖子
		discussions.PATCH("/:id/pin", middleware.RequireTeacher(), todo)              // 置顶/取消置顶
		discussions.POST("/:id/replies", todo)                                       // 回复帖子
		discussions.POST("/:id/like", todo)                                          // 点赞
		discussions.DELETE("/:id/like", todo)                                        // 取消点赞
	}

	// ========== 讨论回复（独立路径） ==========
	discussionReplies := rg.Group("/discussion-replies")
	discussionReplies.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		discussionReplies.DELETE("/:id", todo)           // 删除回复
	}

	// ========== 公告（独立路径） ==========
	announcements := rg.Group("/announcements")
	announcements.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	announcements.Use(middleware.RequireTeacher())
	{
		announcements.PUT("/:id", todo)                 // 编辑公告
		announcements.DELETE("/:id", todo)              // 删除公告
	}

	// ========== 课程评价（独立路径） ==========
	courseEvaluations := rg.Group("/course-evaluations")
	courseEvaluations.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		courseEvaluations.PUT("/:id", todo)              // 修改评价
	}

	// ========== 共享课程库 ==========
	sharedCourses := rg.Group("/shared-courses")
	sharedCourses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	sharedCourses.Use(middleware.RequireTeacher())
	{
		sharedCourses.GET("", todo)                     // 共享课程库列表
		sharedCourses.GET("/:id", todo)                 // 共享课程详情
	}

	// ========== 我的课程表 ==========
	mySchedule := rg.Group("/my-schedule")
	mySchedule.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		mySchedule.GET("", todo)                        // 我的周课程表
	}

	// ========== 学生视角 — 我的课程 ==========
	myCourses := rg.Group("/my-courses")
	myCourses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	myCourses.Use(middleware.RequireStudent())
	{
		myCourses.GET("", todo)                         // 我的课程列表
	}
}
