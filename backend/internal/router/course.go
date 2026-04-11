// course.go
// 模块03 — 课程与教学 路由注册
// 注册课程管理、章节课时、选课、作业、提交批改、学习进度、课程表、
// 公告、讨论区、评价、成绩管理、共享课程库、课程统计、学生视角等路由
// 共 76 个端点

package router

import (
	"github.com/gin-gonic/gin"

	coursehandler "github.com/lenschain/backend/internal/handler/course"
	"github.com/lenschain/backend/internal/middleware"
)

// CourseHandlers 模块03（课程与教学）的 Handler 集合
type CourseHandlers struct {
	CourseHandler     *coursehandler.CourseHandler
	AssignmentHandler *coursehandler.AssignmentHandler
	DiscussionHandler *coursehandler.DiscussionHandler
}

// RegisterCourseRoutes 注册课程与教学模块路由
func RegisterCourseRoutes(rg *gin.RouterGroup, ch *CourseHandlers) {
	// ========== 1. 课程管理（教师） ==========
	courses := rg.Group("/courses")
	courses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		// 教师操作
		courses.POST("", middleware.RequireTeacher(), ch.CourseHandler.Create)                          // 创建课程
		courses.GET("", ch.CourseHandler.List)                                                          // 课程列表
		courses.GET("/:id", ch.CourseHandler.GetByID)                                                  // 课程详情
		courses.PUT("/:id", middleware.RequireTeacher(), ch.CourseHandler.Update)                       // 编辑课程信息
		courses.DELETE("/:id", middleware.RequireTeacher(), ch.CourseHandler.Delete)                    // 删除课程（仅草稿）
		courses.POST("/:id/publish", middleware.RequireTeacher(), ch.CourseHandler.Publish)             // 发布课程
		courses.POST("/:id/end", middleware.RequireTeacher(), ch.CourseHandler.End)                     // 结束课程
		courses.POST("/:id/archive", middleware.RequireTeacher(), ch.CourseHandler.Archive)             // 归档课程
		courses.POST("/:id/clone", middleware.RequireTeacher(), ch.CourseHandler.Clone)                 // 克隆课程
		courses.PATCH("/:id/share", middleware.RequireTeacher(), ch.CourseHandler.ToggleShare)          // 设置共享状态
		courses.POST("/:id/invite-code/refresh", middleware.RequireTeacher(), ch.CourseHandler.RefreshInviteCode) // 刷新邀请码

		// ========== 2. 章节与课时 ==========
		courses.GET("/:id/chapters", ch.CourseHandler.ListChapters)                                    // 获取章节列表（含课时）
		courses.POST("/:id/chapters", middleware.RequireTeacher(), ch.CourseHandler.CreateChapter)      // 创建章节
		courses.PUT("/:id/chapters/sort", middleware.RequireTeacher(), ch.CourseHandler.SortChapters)   // 章节排序

		// ========== 3. 选课管理 ==========
		courses.POST("/join", middleware.RequireStudent(), ch.CourseHandler.JoinByInviteCode)           // 通过邀请码加入课程
		courses.POST("/:id/students", middleware.RequireTeacher(), ch.CourseHandler.AddStudent)         // 教师添加学生
		courses.POST("/:id/students/batch", middleware.RequireTeacher(), ch.CourseHandler.BatchAddStudents) // 批量添加学生
		courses.DELETE("/:id/students/:student_id", middleware.RequireTeacher(), ch.CourseHandler.RemoveStudent) // 移除学生
		courses.GET("/:id/students", middleware.RequireTeacher(), ch.CourseHandler.ListStudents)        // 课程学生列表

		// ========== 4. 作业管理 ==========
		courses.POST("/:id/assignments", middleware.RequireTeacher(), ch.AssignmentHandler.CreateAssignment) // 创建作业
		courses.GET("/:id/assignments", ch.AssignmentHandler.ListAssignments)                           // 作业列表

		// ========== 5. 学习进度 ==========
		courses.GET("/:id/my-progress", middleware.RequireStudent(), ch.CourseHandler.GetMyProgress)    // 我的课程学习进度
		courses.GET("/:id/students-progress", middleware.RequireTeacher(), ch.CourseHandler.ListStudentsProgress) // 所有学生学习进度

		// ========== 6. 课程表 ==========
		courses.PUT("/:id/schedules", middleware.RequireTeacher(), ch.CourseHandler.SetSchedule)        // 设置课程表
		courses.GET("/:id/schedules", ch.CourseHandler.GetSchedule)                                    // 获取课程表

		// ========== 7. 公告 ==========
		courses.POST("/:id/announcements", middleware.RequireTeacher(), ch.DiscussionHandler.CreateAnnouncement) // 发布公告
		courses.GET("/:id/announcements", ch.DiscussionHandler.ListAnnouncements)                      // 公告列表

		// ========== 8. 讨论区 ==========
		courses.POST("/:id/discussions", ch.DiscussionHandler.CreateDiscussion)                        // 发帖
		courses.GET("/:id/discussions", ch.DiscussionHandler.ListDiscussions)                           // 帖子列表

		// ========== 9. 课程评价 ==========
		courses.POST("/:id/evaluations", middleware.RequireStudent(), ch.DiscussionHandler.CreateEvaluation) // 提交评价
		courses.GET("/:id/evaluations", ch.DiscussionHandler.ListEvaluations)                          // 评价列表

		// ========== 10. 成绩管理 ==========
		courses.PUT("/:id/grade-config", middleware.RequireTeacher(), ch.DiscussionHandler.SetGradeConfig)   // 配置成绩权重
		courses.GET("/:id/grade-config", middleware.RequireTeacher(), ch.DiscussionHandler.GetGradeConfig)   // 获取成绩权重配置
		courses.GET("/:id/grades", middleware.RequireTeacher(), todo)                                   // 成绩汇总表（模块06聚合）
		courses.PATCH("/:id/grades/:student_id", middleware.RequireTeacher(), todo)                     // 手动调整成绩（模块06聚合）
		courses.GET("/:id/grades/export", middleware.RequireTeacher(), todo)                            // 导出成绩单（模块06聚合）
		courses.GET("/:id/my-grades", middleware.RequireStudent(), todo)                                // 我的成绩（模块06聚合）

		// ========== 11. 课程统计 ==========
		courses.GET("/:id/statistics/overview", middleware.RequireTeacher(), ch.CourseHandler.GetCourseOverview)    // 课程整体统计
		courses.GET("/:id/statistics/assignments", middleware.RequireTeacher(), todo)                   // 作业统计（模块06聚合）
		courses.GET("/:id/statistics/export", middleware.RequireTeacher(), todo)                        // 导出统计报告（模块06聚合）
	}

	// ========== 章节（独立路径） ==========
	chapters := rg.Group("/chapters")
	chapters.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	chapters.Use(middleware.RequireTeacher())
	{
		chapters.PUT("/:id", ch.CourseHandler.UpdateChapter)           // 编辑章节
		chapters.DELETE("/:id", ch.CourseHandler.DeleteChapter)        // 删除章节
		chapters.POST("/:id/lessons", ch.CourseHandler.CreateLesson)   // 创建课时
		chapters.PUT("/:id/lessons/sort", ch.CourseHandler.SortLessons) // 课时排序
	}

	// ========== 课时（独立路径） ==========
	lessons := rg.Group("/lessons")
	lessons.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		lessons.GET("/:id", ch.CourseHandler.GetLesson)                                                // 课时详情
		lessons.PUT("/:id", middleware.RequireTeacher(), ch.CourseHandler.UpdateLesson)                 // 编辑课时
		lessons.DELETE("/:id", middleware.RequireTeacher(), ch.CourseHandler.DeleteLesson)              // 删除课时
		lessons.POST("/:id/attachments", middleware.RequireTeacher(), ch.CourseHandler.UploadAttachment) // 上传课时附件
		lessons.POST("/:id/progress", middleware.RequireStudent(), ch.CourseHandler.UpdateProgress)     // 更新学习进度
	}

	// ========== 课时附件（独立路径） ==========
	lessonAttachments := rg.Group("/lesson-attachments")
	lessonAttachments.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	lessonAttachments.Use(middleware.RequireTeacher())
	{
		lessonAttachments.DELETE("/:id", ch.CourseHandler.DeleteAttachment) // 删除附件
	}

	// ========== 作业（独立路径） ==========
	assignments := rg.Group("/assignments")
	assignments.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		assignments.GET("/:id", ch.AssignmentHandler.GetAssignment)                                    // 作业详情（含题目）
		assignments.PUT("/:id", middleware.RequireTeacher(), ch.AssignmentHandler.UpdateAssignment)     // 编辑作业
		assignments.DELETE("/:id", middleware.RequireTeacher(), ch.AssignmentHandler.DeleteAssignment)  // 删除作业
		assignments.POST("/:id/publish", middleware.RequireTeacher(), ch.AssignmentHandler.PublishAssignment) // 发布作业
		assignments.POST("/:id/questions", middleware.RequireTeacher(), ch.AssignmentHandler.AddQuestion) // 添加题目
		assignments.POST("/:id/submit", middleware.RequireStudent(), ch.AssignmentHandler.SubmitAssignment) // 学生提交作业
		assignments.GET("/:id/my-submissions", middleware.RequireStudent(), ch.AssignmentHandler.ListMySubmissions) // 我的提交记录
		assignments.GET("/:id/submissions", middleware.RequireTeacher(), ch.AssignmentHandler.ListSubmissions) // 所有学生提交列表
	}

	// ========== 作业题目（独立路径） ==========
	assignmentQuestions := rg.Group("/assignment-questions")
	assignmentQuestions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	assignmentQuestions.Use(middleware.RequireTeacher())
	{
		assignmentQuestions.PUT("/:id", ch.AssignmentHandler.UpdateQuestion)   // 编辑题目
		assignmentQuestions.DELETE("/:id", ch.AssignmentHandler.DeleteQuestion) // 删除题目
	}

	// ========== 提交（独立路径） ==========
	submissions := rg.Group("/submissions")
	submissions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		submissions.GET("/:id", ch.AssignmentHandler.GetSubmission)                                    // 提交详情
		submissions.POST("/:id/grade", middleware.RequireTeacher(), ch.AssignmentHandler.GradeSubmission) // 批改提交
	}

	// ========== 讨论区（独立路径） ==========
	discussions := rg.Group("/discussions")
	discussions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		discussions.GET("/:id", ch.DiscussionHandler.GetDiscussion)                                    // 帖子详情（含回复）
		discussions.DELETE("/:id", ch.DiscussionHandler.DeleteDiscussion)                               // 删除帖子
		discussions.PATCH("/:id/pin", middleware.RequireTeacher(), ch.DiscussionHandler.PinDiscussion)  // 置顶/取消置顶
		discussions.POST("/:id/replies", ch.DiscussionHandler.CreateReply)                              // 回复帖子
		discussions.POST("/:id/like", ch.DiscussionHandler.ToggleLike)                                 // 点赞/取消点赞
		discussions.DELETE("/:id/like", ch.DiscussionHandler.ToggleLike)                                // 取消点赞（复用 toggle）
	}

	// ========== 讨论回复（独立路径） ==========
	discussionReplies := rg.Group("/discussion-replies")
	discussionReplies.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		discussionReplies.DELETE("/:id", ch.DiscussionHandler.DeleteReply) // 删除回复
	}

	// ========== 公告（独立路径） ==========
	announcements := rg.Group("/announcements")
	announcements.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	announcements.Use(middleware.RequireTeacher())
	{
		announcements.PUT("/:id", ch.DiscussionHandler.UpdateAnnouncement)    // 编辑公告
		announcements.DELETE("/:id", ch.DiscussionHandler.DeleteAnnouncement) // 删除公告
	}

	// ========== 课程评价（独立路径） ==========
	courseEvaluations := rg.Group("/course-evaluations")
	courseEvaluations.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		courseEvaluations.PUT("/:id", ch.DiscussionHandler.UpdateEvaluation) // 修改评价
	}

	// ========== 共享课程库 ==========
	sharedCourses := rg.Group("/shared-courses")
	sharedCourses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	sharedCourses.Use(middleware.RequireTeacher())
	{
		sharedCourses.GET("", ch.CourseHandler.ListShared)         // 共享课程库列表
		sharedCourses.GET("/:id", ch.CourseHandler.GetSharedDetail) // 共享课程详情
	}

	// ========== 我的课程表 ==========
	mySchedule := rg.Group("/my-schedule")
	mySchedule.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		mySchedule.GET("", ch.CourseHandler.GetMySchedule) // 我的周课程表
	}

	// ========== 学生视角 — 我的课程 ==========
	myCourses := rg.Group("/my-courses")
	myCourses.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	myCourses.Use(middleware.RequireStudent())
	{
		myCourses.GET("", ch.CourseHandler.ListMyCourses) // 我的课程列表
	}
}
