// init_course.go
// 模块03 — 课程与教学：依赖注入初始化
// 按照 repository → adapter → service → handler 的顺序组装模块03的依赖
// 每个模块独立一个 init 文件，避免 main.go 膨胀为单体

package main

import (
	"context"

	handler "github.com/lenschain/backend/internal/handler/course"
	"github.com/lenschain/backend/internal/model/dto"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	svc "github.com/lenschain/backend/internal/service/course"
	notificationsvc "github.com/lenschain/backend/internal/service/notification"
)

// gradeCourseLockService 定义模块03接入模块06时所需的最小锁定查询契约。
type gradeCourseLockService interface {
	IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error)
}

// initCourseModule 初始化模块03（课程与教学）的 Handler
// 按照 repository → adapter → service → handler 的顺序组装依赖
func initCourseModule(gradeService gradeCourseLockService, notificationDispatcher notificationsvc.EventDispatcher) *router.CourseHandlers {
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
	draftRepo := courserepo.NewDraftRepository(db)
	answerRepo := courserepo.NewAnswerRepository(db)
	discussionRepo := courserepo.NewDiscussionRepository(db)
	replyRepo := courserepo.NewReplyRepository(db)
	likeRepo := courserepo.NewLikeRepository(db)
	announcementRepo := courserepo.NewAnnouncementRepository(db)
	evaluationRepo := courserepo.NewEvaluationRepository(db)
	gradeConfigRepo := courserepo.NewGradeConfigRepository(db)
	gradeOverrideRepo := courserepo.NewGradeOverrideRepository(db)
	scheduleRepo := courserepo.NewScheduleRepository(db)

	// ========== 跨模块 Adapter ==========
	// 用户名查询（模块01 → 模块03）
	userRepo := authrepo.NewUserRepository(db)
	profileRepo := authrepo.NewProfileRepository(db)
	roleRepo := authrepo.NewRoleRepository(db)
	userNameQuerier := &userNameQuerierAdapter{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		roleRepo:    roleRepo,
	}

	// 学校名称查询（模块02 → 模块03）
	// 复用 init_school.go 中定义的 newSchoolNameQuerier
	schoolRepo := schoolrepo.NewSchoolRepository(db)
	schoolNameQuerier := newSchoolNameQuerier(schoolRepo)
	gradeLockChecker := newCourseGradeLockChecker(gradeService)
	courseNotificationDispatcher := newCourseNotificationDispatcher(notificationDispatcher)

	// ========== Service 层 ==========
	courseService := svc.NewCourseService(
		db, courseRepo, chapterRepo, lessonRepo, attachmentRepo, enrollmentRepo,
		assignmentRepo, questionRepo, progressRepo, evaluationRepo,
		userNameQuerier, schoolNameQuerier,
	)
	contentService := svc.NewContentService(
		courseRepo, chapterRepo, lessonRepo, attachmentRepo,
		enrollmentRepo, progressRepo, userNameQuerier, userNameQuerier,
	)
	assignmentService := svc.NewAssignmentService(
		courseRepo, assignmentRepo, questionRepo, submissionRepo,
		draftRepo, answerRepo, enrollmentRepo, userNameQuerier, userNameQuerier, courseNotificationDispatcher,
	)
	discussionService := svc.NewDiscussionService(
		courseRepo, discussionRepo, replyRepo, likeRepo,
		announcementRepo, evaluationRepo, enrollmentRepo,
		userNameQuerier,
	)
	progressService := svc.NewProgressService(
		courseRepo, lessonRepo, chapterRepo, enrollmentRepo,
		progressRepo, assignmentRepo, submissionRepo,
		gradeConfigRepo, gradeOverrideRepo,
		scheduleRepo, userNameQuerier, userNameQuerier,
	)
	courseGradeService := svc.NewGradeService(
		courseRepo, enrollmentRepo, assignmentRepo, submissionRepo,
		gradeConfigRepo, gradeOverrideRepo, userNameQuerier, gradeLockChecker, progressService,
	)

	// ========== Handler 层 ==========
	courseHandler := handler.NewCourseHandler(courseService, courseGradeService, contentService, progressService)
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

// courseNotificationDispatcherAdapter 跨模块适配器：转发模块03产生的通知事件到模块07。
type courseNotificationDispatcherAdapter struct {
	dispatcher notificationsvc.EventDispatcher
}

// newCourseNotificationDispatcher 创建模块03使用的通知事件分发器。
func newCourseNotificationDispatcher(dispatcher notificationsvc.EventDispatcher) svc.NotificationEventDispatcher {
	if dispatcher == nil {
		return nil
	}
	return &courseNotificationDispatcherAdapter{dispatcher: dispatcher}
}

// DispatchEvent 将模块03内部事件转交给模块07统一生成站内信。
func (a *courseNotificationDispatcherAdapter) DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error {
	if a == nil || a.dispatcher == nil || req == nil {
		return nil
	}
	return a.dispatcher.DispatchEvent(ctx, req)
}

// courseGradeLockCheckerAdapter 跨模块 adapter：查询模块06中的成绩锁定状态。
// 模块03只依赖本地接口，避免直接感知聚合层实现细节。
type courseGradeLockCheckerAdapter struct {
	gradeService gradeCourseLockService
}

// newCourseGradeLockChecker 创建模块03使用的成绩锁定检查器。
// 当模块06未注入时返回 nil，模块03会回退到默认空实现。
func newCourseGradeLockChecker(gradeService gradeCourseLockService) svc.GradeLockChecker {
	if gradeService == nil {
		return nil
	}
	return &courseGradeLockCheckerAdapter{gradeService: gradeService}
}

// IsCourseGradeLocked 调用模块06服务判断课程成绩是否已被锁定。
func (a *courseGradeLockCheckerAdapter) IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error) {
	if a == nil || a.gradeService == nil {
		return false, nil
	}
	return a.gradeService.IsCourseGradeLocked(ctx, courseID)
}

// ========== UserNameQuerier Adapter ==========

// userNameQuerierAdapter 跨模块 adapter：查询用户名称
// 实现 course.UserNameQuerier 接口，内部调用模块01的 repo 层
type userNameQuerierAdapter struct {
	userRepo    authrepo.UserRepository
	profileRepo authrepo.ProfileRepository
	roleRepo    authrepo.RoleRepository
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
	if a.profileRepo == nil {
		return summary
	}
	profile, err := a.profileRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return summary
	}
	summary.College = profile.College
	summary.Major = profile.Major
	summary.ClassName = profile.ClassName
	return summary
}

// GetUserSchoolID 根据用户ID查询所属学校
// 查询失败时返回错误，供模块03做租户隔离校验。
func (a *userNameQuerierAdapter) GetUserSchoolID(ctx context.Context, userID int64) (int64, error) {
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return user.SchoolID, nil
}

// HasRole 判断用户是否具备指定角色
// 查询失败时返回错误，避免课程模块自行解析用户表结构。
func (a *userNameQuerierAdapter) HasRole(ctx context.Context, userID int64, role string) (bool, error) {
	if a.roleRepo == nil {
		return false, nil
	}
	codes, err := a.roleRepo.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, code := range codes {
		if code == role {
			return true, nil
		}
	}
	return false, nil
}
