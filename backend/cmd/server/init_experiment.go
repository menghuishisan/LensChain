// init_experiment.go
// 模块04 — 实验环境：依赖注入初始化
// 按照 repository → adapter → service → handler 的顺序组装模块04依赖
// 统一在此处装配跨模块查询适配器，避免业务层直接耦合其他模块实现

package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/config"
	experimenthandler "github.com/lenschain/backend/internal/handler/experiment"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	svc "github.com/lenschain/backend/internal/service/experiment"
)

// initExperimentModule 初始化模块04（实验环境）的 Handler。
// 统一装配镜像、模板、实例、分组、监控、配额与 SimEngine/K8s 依赖。
func initExperimentModule() *router.ExperimentHandlers {
	db := database.Get()

	// ========== Repository 层 ==========
	imageCategoryRepo := experimentrepo.NewImageCategoryRepository(db)
	imageRepo := experimentrepo.NewImageRepository(db)
	imageVersionRepo := experimentrepo.NewImageVersionRepository(db)

	templateRepo := experimentrepo.NewTemplateRepository(db)
	containerRepo := experimentrepo.NewContainerRepository(db)
	checkpointRepo := experimentrepo.NewCheckpointRepository(db)
	initScriptRepo := experimentrepo.NewInitScriptRepository(db)
	simSceneRepo := experimentrepo.NewSimSceneRepository(db)
	tagRepo := experimentrepo.NewTagRepository(db)
	templateTagRepo := experimentrepo.NewTemplateTagRepository(db)
	roleRepo := experimentrepo.NewRoleRepository(db)

	scenarioRepo := experimentrepo.NewScenarioRepository(db)
	linkGroupRepo := experimentrepo.NewLinkGroupRepository(db)
	linkGroupSceneRepo := experimentrepo.NewLinkGroupSceneRepository(db)

	instanceRepo := experimentrepo.NewInstanceRepository(db)
	instanceContainerRepo := experimentrepo.NewInstanceContainerRepository(db)
	checkpointResultRepo := experimentrepo.NewCheckpointResultRepository(db)
	snapshotRepo := experimentrepo.NewSnapshotRepository(db)
	opLogRepo := experimentrepo.NewOperationLogRepository(db)
	reportRepo := experimentrepo.NewReportRepository(db)
	groupRepo := experimentrepo.NewGroupRepository(db)
	groupMemberRepo := experimentrepo.NewGroupMemberRepository(db)
	groupMessageRepo := experimentrepo.NewGroupMessageRepository(db)
	quotaRepo := experimentrepo.NewQuotaRepository(db)

	// ========== 跨模块 Repository ==========
	userRepo := authrepo.NewUserRepository(db)
	courseRepo := courserepo.NewCourseRepository(db)
	lessonRepo := courserepo.NewLessonRepository(db)
	courseExperimentRepo := courserepo.NewCourseExperimentRepository(db)
	assignmentRepo := courserepo.NewAssignmentRepository(db)
	submissionRepo := courserepo.NewSubmissionRepository(db)
	enrollmentRepo := courserepo.NewEnrollmentRepository(db)
	schoolRepo := schoolrepo.NewSchoolRepository(db)

	// ========== 跨模块 Adapter ==========
	userQuerier := &experimentUserQuerierAdapter{userRepo: userRepo}
	courseQuerier := &experimentCourseQuerierAdapter{courseRepo: courseRepo}
	courseTemplateQuerier := &experimentCourseTemplateQuerierAdapter{
		lessonRepo:           lessonRepo,
		courseExperimentRepo: courseExperimentRepo,
	}
	enrollmentChecker := &experimentEnrollmentCheckerAdapter{enrollmentRepo: enrollmentRepo}
	courseGradeSyncer := &experimentCourseGradeSyncerAdapter{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
	}
	endedCourseQuerier := &experimentEndedCourseQuerierAdapter{courseRepo: courseRepo}
	courseRosterQuerier := &experimentCourseRosterQuerierAdapter{
		enrollmentRepo: enrollmentRepo,
		userRepo:       userRepo,
	}
	schoolNameQuerier := &experimentSchoolNameQuerierAdapter{schoolRepo: schoolRepo}
	// 模块07内部通知发送器暂不在此处注入。
	// 原因：当前模块07的 /internal/send-event 仍未完成 service/handler 闭环，
	// 先保持模块04的跨模块边界清晰，待模块07补齐后在此处按接口注入，不在模块04 service 内写跨模块写入逻辑。

	// ========== 基础服务 ==========
	k8sSvc, err := svc.NewK8sService(config.Get().K8s)
	if err != nil {
		logger.L.Fatal("初始化模块04 K8s 服务失败", zap.Error(err))
	}

	simEngineSvc, err := svc.NewSimEngineService(config.Get().SimEngine)
	if err != nil {
		logger.L.Fatal("初始化模块04 SimEngine 服务失败", zap.Error(err))
	}

	// ========== Service 层 ==========
	imageService := svc.NewImageService(
		db,
		imageCategoryRepo,
		imageRepo,
		imageVersionRepo,
		userQuerier,
		k8sSvc,
	)
	templateService := svc.NewTemplateService(
		db,
		templateRepo,
		containerRepo,
		checkpointRepo,
		initScriptRepo,
		simSceneRepo,
		imageRepo,
		imageVersionRepo,
		scenarioRepo,
		linkGroupRepo,
		tagRepo,
		templateTagRepo,
		roleRepo,
		userQuerier,
	)
	templateSubService := svc.NewTemplateSubService(
		templateRepo,
		containerRepo,
		checkpointRepo,
		initScriptRepo,
		simSceneRepo,
		tagRepo,
		templateTagRepo,
		roleRepo,
		imageVersionRepo,
		imageRepo,
		scenarioRepo,
		linkGroupRepo,
	)
	scenarioService := svc.NewScenarioService(
		scenarioRepo,
		linkGroupRepo,
		linkGroupSceneRepo,
		userQuerier,
		k8sSvc,
	)
	instanceService := svc.NewInstanceService(
		db,
		instanceRepo,
		instanceContainerRepo,
		templateRepo,
		containerRepo,
		imageRepo,
		imageVersionRepo,
		checkpointRepo,
		checkpointResultRepo,
		groupRepo,
		groupMemberRepo,
		snapshotRepo,
		opLogRepo,
		reportRepo,
		quotaRepo,
		initScriptRepo,
		simSceneRepo,
		scenarioRepo,
		linkGroupRepo,
		linkGroupSceneRepo,
		k8sSvc,
		simEngineSvc,
		userQuerier,
		userQuerier,
		schoolNameQuerier,
		courseQuerier,
		courseGradeSyncer,
		enrollmentChecker,
	)
	groupService := svc.NewGroupService(
		db,
		groupRepo,
		groupMemberRepo,
		groupMessageRepo,
		roleRepo,
		templateRepo,
		checkpointRepo,
		checkpointResultRepo,
		instanceRepo,
		userQuerier,
		courseQuerier,
		courseRosterQuerier,
	)
	monitorService := svc.NewMonitorService(
		instanceRepo,
		instanceContainerRepo,
		templateRepo,
		imageRepo,
		scenarioRepo,
		quotaRepo,
		checkpointRepo,
		checkpointResultRepo,
		courseQuerier,
		courseTemplateQuerier,
		courseRosterQuerier,
		userQuerier,
		userQuerier,
		schoolNameQuerier,
		k8sSvc,
	)
	quotaService := svc.NewQuotaService(
		quotaRepo,
		instanceRepo,
		schoolNameQuerier,
		courseQuerier,
	)
	experimentScheduler := svc.NewExperimentScheduler(
		instanceService,
		instanceRepo,
		templateRepo,
		endedCourseQuerier,
		imageRepo,
		imageVersionRepo,
		k8sSvc,
	)

	// ========== Handler 层 ==========
	templateHandler := experimenthandler.NewTemplateHandler(
		imageService,
		templateService,
		templateSubService,
		scenarioService,
	)
	instanceHandler := experimenthandler.NewInstanceHandler(
		instanceService,
		groupService,
		monitorService,
		quotaService,
		imageService,
	)

	// ========== 定时任务注册 ==========
	cronpkg.AddTask(cronpkg.CronExpAutoSnapshot, "实验自动快照", experimentScheduler.RunAutoSnapshot)
	cronpkg.AddTask(cronpkg.CronExpIdleReclaim, "实验空闲回收", experimentScheduler.RunIdleReclaim)
	cronpkg.AddTask(cronpkg.CronExpExpiredCleanup, "实验超时与课程结束回收", experimentScheduler.RunExpiredCleanup)
	cronpkg.AddTask(cronpkg.CronExpRuntimeHealth, "实验运行时健康检查", experimentScheduler.RunRuntimeHealthCheck)
	cronpkg.AddTask(cronpkg.CronExpImagePrePullSync, "实验镜像预拉取对账", experimentScheduler.RunImagePrePullReconcile)

	return &router.ExperimentHandlers{
		TemplateHandler: templateHandler,
		InstanceHandler: instanceHandler,
	}
}

// experimentUserQuerierAdapter 跨模块适配器：提供模块04所需的用户名称与摘要查询。
type experimentUserQuerierAdapter struct {
	userRepo authrepo.UserRepository
}

// GetUserName 根据用户ID查询姓名。
func (a *experimentUserQuerierAdapter) GetUserName(ctx context.Context, userID int64) string {
	summary := a.GetUserSummary(ctx, userID)
	if summary == nil {
		return ""
	}
	return summary.Name
}

// GetUserSummary 根据用户ID查询实验环境模块所需的用户摘要。
func (a *experimentUserQuerierAdapter) GetUserSummary(ctx context.Context, userID int64) *svc.ExperimentUserSummary {
	if userID == 0 {
		return nil
	}
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil
	}
	studentNo := ""
	if user.StudentNo != nil {
		studentNo = *user.StudentNo
	}
	return &svc.ExperimentUserSummary{
		UserID:    user.ID,
		Name:      user.Name,
		StudentNo: studentNo,
	}
}

// experimentCourseQuerierAdapter 跨模块适配器：提供课程标题、学校和教师查询。
type experimentCourseQuerierAdapter struct {
	courseRepo courserepo.CourseRepository
}

// GetCourseTitle 根据课程ID查询标题。
func (a *experimentCourseQuerierAdapter) GetCourseTitle(ctx context.Context, courseID int64) string {
	course, err := a.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return ""
	}
	return course.Title
}

// GetCourseSchoolID 根据课程ID查询所属学校。
func (a *experimentCourseQuerierAdapter) GetCourseSchoolID(ctx context.Context, courseID int64) (int64, error) {
	course, err := a.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return 0, err
	}
	return course.SchoolID, nil
}

// GetCourseTeacherID 根据课程ID查询授课教师。
func (a *experimentCourseQuerierAdapter) GetCourseTeacherID(ctx context.Context, courseID int64) (int64, error) {
	course, err := a.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return 0, err
	}
	return course.TeacherID, nil
}

// experimentCourseTemplateQuerierAdapter 跨模块适配器：查询课程已关联的实验模板ID。
// 统一收敛 lessons.experiment_id 与 course_experiments.experiment_id 两种课程侧来源，
// 避免模块04在 service 内直接依赖模块03表结构。
type experimentCourseTemplateQuerierAdapter struct {
	lessonRepo           courserepo.LessonRepository
	courseExperimentRepo courserepo.CourseExperimentRepository
}

// ListCourseTemplateIDs 返回课程已配置的实验模板ID集合。
func (a *experimentCourseTemplateQuerierAdapter) ListCourseTemplateIDs(ctx context.Context, courseID int64) ([]int64, error) {
	templateIDSet := make(map[int64]struct{})

	if a.lessonRepo != nil {
		lessons, err := a.lessonRepo.ListByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}
		for _, lesson := range lessons {
			if lesson == nil || lesson.ExperimentID == nil || *lesson.ExperimentID == 0 {
				continue
			}
			templateIDSet[*lesson.ExperimentID] = struct{}{}
		}
	}

	if a.courseExperimentRepo != nil {
		courseExperiments, err := a.courseExperimentRepo.ListByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}
		for _, courseExperiment := range courseExperiments {
			if courseExperiment == nil || courseExperiment.ExperimentID == 0 {
				continue
			}
			templateIDSet[courseExperiment.ExperimentID] = struct{}{}
		}
	}

	templateIDs := make([]int64, 0, len(templateIDSet))
	for templateID := range templateIDSet {
		templateIDs = append(templateIDs, templateID)
	}
	sort.Slice(templateIDs, func(i, j int) bool {
		return templateIDs[i] < templateIDs[j]
	})
	return templateIDs, nil
}

// experimentCourseGradeSyncerAdapter 跨模块适配器：将实验最终成绩同步到课程作业提交。
type experimentCourseGradeSyncerAdapter struct {
	assignmentRepo courserepo.AssignmentRepository
	submissionRepo courserepo.SubmissionRepository
}

// SyncExperimentScore 将实验最终成绩写回课程成绩体系。
func (a *experimentCourseGradeSyncerAdapter) SyncExperimentScore(
	ctx context.Context,
	courseID int64,
	assignmentID int64,
	studentID int64,
	templateTitle string,
	score float64,
	submittedAt time.Time,
	teacherComment *string,
) error {
	assignment, err := a.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return err
	}
	if assignment.CourseID != courseID {
		return fmt.Errorf("assignment %d does not belong to course %d", assignmentID, courseID)
	}

	comment := buildExperimentScoreComment(templateTitle, teacherComment)
	gradedAt := time.Now()
	latest, err := a.submissionRepo.GetLatestByStudentAndAssignment(ctx, studentID, assignmentID)
	if err == nil && latest != nil {
		return a.submissionRepo.UpdateFields(ctx, latest.ID, map[string]interface{}{
			"status":                 enum.SubmissionStatusGraded,
			"total_score":            score,
			"score_before_deduction": score,
			"score_after_deduction":  score,
			"teacher_comment":        comment,
			"submitted_at":           submittedAt,
			"graded_at":              gradedAt,
		})
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	count, err := a.submissionRepo.CountByStudentAndAssignment(ctx, studentID, assignmentID)
	if err != nil {
		return err
	}

	submission := &entity.AssignmentSubmission{
		ID:                   snowflake.Generate(),
		AssignmentID:         assignmentID,
		StudentID:            studentID,
		SubmissionNo:         count + 1,
		Status:               enum.SubmissionStatusGraded,
		TotalScore:           float64Ptr(score),
		ScoreBeforeDeduction: float64Ptr(score),
		ScoreAfterDeduction:  float64Ptr(score),
		TeacherComment:       &comment,
		SubmittedAt:          submittedAt,
		GradedAt:             &gradedAt,
	}
	return a.submissionRepo.Create(ctx, submission)
}

// buildExperimentScoreComment 构建实验成绩回传到课程体系时使用的说明文字。
func buildExperimentScoreComment(templateTitle string, teacherComment *string) string {
	base := "实验成绩自动回传"
	if templateTitle != "" {
		base = fmt.Sprintf("实验成绩自动回传：%s", templateTitle)
	}
	if teacherComment != nil && *teacherComment != "" {
		return fmt.Sprintf("%s；教师评语：%s", base, *teacherComment)
	}
	return base
}

// float64Ptr 返回 float64 指针，便于组装模块03提交实体。
func float64Ptr(v float64) *float64 {
	return &v
}

// experimentEnrollmentCheckerAdapter 跨模块适配器：检查学生选课状态。
type experimentEnrollmentCheckerAdapter struct {
	enrollmentRepo courserepo.EnrollmentRepository
}

// IsEnrolled 判断学生是否已加入指定课程。
func (a *experimentEnrollmentCheckerAdapter) IsEnrolled(ctx context.Context, courseID, studentID int64) (bool, error) {
	return a.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
}

// experimentCourseRosterQuerierAdapter 跨模块适配器：查询课程学生名单。
type experimentCourseRosterQuerierAdapter struct {
	enrollmentRepo courserepo.EnrollmentRepository
	userRepo       authrepo.UserRepository
}

// ListCourseStudents 获取课程学生摘要列表。
func (a *experimentCourseRosterQuerierAdapter) ListCourseStudents(ctx context.Context, courseID int64) ([]svc.CourseStudentSummary, error) {
	enrollments, _, err := a.enrollmentRepo.List(ctx, &courserepo.EnrollmentListParams{
		CourseID: courseID,
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(enrollments))
	for _, enrollment := range enrollments {
		userIDs = append(userIDs, enrollment.StudentID)
	}
	users, err := a.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]string, len(users))
	userNoMap := make(map[int64]string, len(users))
	for _, user := range users {
		userMap[user.ID] = user.Name
		if user.StudentNo != nil {
			userNoMap[user.ID] = *user.StudentNo
		}
	}

	result := make([]svc.CourseStudentSummary, 0, len(enrollments))
	for _, enrollment := range enrollments {
		result = append(result, svc.CourseStudentSummary{
			StudentID: enrollment.StudentID,
			Name:      userMap[enrollment.StudentID],
			StudentNo: userNoMap[enrollment.StudentID],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].StudentNo == result[j].StudentNo {
			return result[i].StudentID < result[j].StudentID
		}
		return result[i].StudentNo < result[j].StudentNo
	})
	return result, nil
}

// experimentEndedCourseQuerierAdapter 跨模块适配器：查询已结束或已归档课程。
type experimentEndedCourseQuerierAdapter struct {
	courseRepo courserepo.CourseRepository
}

// ListEndedCourseIDs 获取所有已结束或已归档课程ID。
func (a *experimentEndedCourseQuerierAdapter) ListEndedCourseIDs(ctx context.Context) ([]int64, error) {
	courses, _, err := a.courseRepo.List(ctx, &courserepo.CourseListParams{
		Statuses: []int16{enum.CourseStatusEnded, enum.CourseStatusArchived},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(courses))
	for _, course := range courses {
		result = append(result, course.ID)
	}
	return result, nil
}

// ListCourseIDsEndingWithin 获取指定时间窗口内即将结束的课程及结束时间。
func (a *experimentEndedCourseQuerierAdapter) ListCourseIDsEndingWithin(ctx context.Context, within time.Duration) (map[int64]time.Time, error) {
	courses, _, err := a.courseRepo.List(ctx, &courserepo.CourseListParams{
		Statuses: []int16{enum.CourseStatusActive},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	deadline := now.Add(within)
	result := make(map[int64]time.Time)
	for _, course := range courses {
		if course.EndAt == nil {
			continue
		}
		if course.EndAt.After(now) && !course.EndAt.After(deadline) {
			result[course.ID] = *course.EndAt
		}
	}
	return result, nil
}

// experimentSchoolNameQuerierAdapter 跨模块适配器：查询学校名称。
type experimentSchoolNameQuerierAdapter struct {
	schoolRepo schoolrepo.SchoolRepository
}

// GetSchoolName 根据学校ID查询学校名称。
func (a *experimentSchoolNameQuerierAdapter) GetSchoolName(ctx context.Context, schoolID int64) string {
	if schoolID == 0 {
		return ""
	}
	school, err := a.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		return ""
	}
	return school.Name
}
