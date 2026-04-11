// init_course.go
// 模块03 — 课程与教学：依赖注入初始化
// 按照 repository → adapter → service → handler 的顺序组装模块03的依赖
// 每个模块独立一个 init 文件，避免 main.go 膨胀为单体

package main

import (
	"context"

	handler "github.com/lenschain/backend/internal/handler/course"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	svc "github.com/lenschain/backend/internal/service/course"
)

// initCourseModule 初始化模块03（课程与教学）的 Handler
// 按照 repository → adapter → service → handler 的顺序组装依赖
func initCourseModule() *router.CourseHandlers {
	db := database.Get()

	// ========== Repository 层 ==========
	courseRepo := courserepo.NewCourseRepository(db)
	chapterRepo := courserepo.NewChapterRepository(db)
	lessonRepo := courserepo.NewLessonRepository(db)
	attachmentRepo := courserepo.NewAttachmentRepository(db)
	enrollmentRepo := courserepo.NewEnrollmentRepository(db)
	progressRepo := courserepo.NewProgressRepository(db)
	assignmentRepo := courserepo.NewAssignmentRepository(db)
	questionRepo := courserepo.NewQuestionRepository(db)
	submissionRepo := courserepo.NewSubmissionRepository(db)
	answerRepo := courserepo.NewAnswerRepository(db)
	discussionRepo := courserepo.NewDiscussionRepository(db)
	replyRepo := courserepo.NewReplyRepository(db)
	likeRepo := courserepo.NewLikeRepository(db)
	announcementRepo := courserepo.NewAnnouncementRepository(db)
	evaluationRepo := courserepo.NewEvaluationRepository(db)
	gradeConfigRepo := courserepo.NewGradeConfigRepository(db)
	scheduleRepo := courserepo.NewScheduleRepository(db)

	// ========== 跨模块 Adapter ==========
	// 用户名查询（模块01 → 模块03）
	userRepo := authrepo.NewUserRepository(db)
	userNameQuerier := &userNameQuerierAdapter{userRepo: userRepo}

	// 学校名称查询（模块02 → 模块03）
	// 复用 init_school.go 中定义的 newSchoolNameQuerier
	schoolRepo := schoolrepo.NewSchoolRepository(db)
	schoolNameQuerier := newSchoolNameQuerier(schoolRepo)

	// ========== Service 层 ==========
	courseService := svc.NewCourseService(
		db, courseRepo, chapterRepo, lessonRepo, enrollmentRepo,
		assignmentRepo, questionRepo, progressRepo, evaluationRepo,
		userNameQuerier, schoolNameQuerier,
	)
	contentService := svc.NewContentService(
		courseRepo, chapterRepo, lessonRepo, attachmentRepo,
		enrollmentRepo, progressRepo, userNameQuerier,
	)
	assignmentService := svc.NewAssignmentService(
		courseRepo, assignmentRepo, questionRepo, submissionRepo,
		answerRepo, enrollmentRepo, userNameQuerier, userNameQuerier,
	)
	discussionService := svc.NewDiscussionService(
		courseRepo, discussionRepo, replyRepo, likeRepo,
		announcementRepo, evaluationRepo, enrollmentRepo,
		gradeConfigRepo, userNameQuerier,
	)
	progressService := svc.NewProgressService(
		courseRepo, lessonRepo, chapterRepo, enrollmentRepo,
		progressRepo, assignmentRepo, submissionRepo,
		scheduleRepo, userNameQuerier, userNameQuerier,
	)

	// ========== Handler 层 ==========
	courseHandler := handler.NewCourseHandler(courseService, contentService, progressService)
	assignmentHandler := handler.NewAssignmentHandler(assignmentService)
	discussionHandler := handler.NewDiscussionHandler(discussionService)

	// ========== 定时任务注册 ==========
	scheduler := svc.NewCourseScheduler(courseRepo)
	cronpkg.AddTask(cronpkg.CronCourseStatusTransition, "课程已发布转进行中", scheduler.RunPublishedToActive)
	cronpkg.AddTask(cronpkg.CronCourseStatusTransition, "课程进行中转已结束", scheduler.RunActiveToEnded)

	return &router.CourseHandlers{
		CourseHandler:     courseHandler,
		AssignmentHandler: assignmentHandler,
		DiscussionHandler: discussionHandler,
	}
}

// ========== UserNameQuerier Adapter ==========

// userNameQuerierAdapter 跨模块 adapter：查询用户名称
// 实现 course.UserNameQuerier 接口，内部调用模块01的 repo 层
type userNameQuerierAdapter struct {
	userRepo authrepo.UserRepository
}

// GetUserName 根据用户ID查询用户名称
// 查询失败时返回空字符串，不影响主流程
func (a *userNameQuerierAdapter) GetUserName(ctx context.Context, userID int64) string {
	summary := a.GetUserSummary(ctx, userID)
	if summary == nil {
		return ""
	}
	return summary.Name
}

// GetUserSummary 根据用户ID查询课程模块所需的用户摘要
// 查询失败时返回 nil，不阻断主业务流程。
func (a *userNameQuerierAdapter) GetUserSummary(ctx context.Context, userID int64) *svc.CourseUserSummary {
	if userID == 0 {
		return nil
	}
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil
	}
	summary := &svc.CourseUserSummary{
		Name:      user.Name,
		StudentNo: user.StudentNo,
	}
	if user.Profile != nil {
		summary.College = user.Profile.College
		summary.Major = user.Profile.Major
		summary.ClassName = user.Profile.ClassName
	}
	return summary
}
