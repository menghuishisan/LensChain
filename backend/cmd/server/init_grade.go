// init_grade.go
// 模块06 — 评测与成绩：依赖注入初始化。
// 按 repository → service → handler 顺序组装模块06依赖，并通过接口接入跨模块只读能力。

package main

import (
	"context"
	"encoding/json"
	"fmt"

	gradehandler "github.com/lenschain/backend/internal/handler/grade"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	graderepo "github.com/lenschain/backend/internal/repository/grade"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	gradesvc "github.com/lenschain/backend/internal/service/grade"
	notificationsvc "github.com/lenschain/backend/internal/service/notification"
)

// initGradeModule 初始化模块06。
// 返回模块06路由所需的 Handler 集合以及供模块03复用的成绩服务接口。
func initGradeModule(notificationDispatcher notificationsvc.EventDispatcher) (*router.GradeHandlers, gradesvc.Service) {
	db := database.Get()

	semesterRepo := graderepo.NewSemesterRepository(db)
	levelRepo := graderepo.NewGradeLevelConfigRepository(db)
	warningCfgRepo := graderepo.NewWarningConfigRepository(db)
	reviewRepo := graderepo.NewGradeReviewRepository(db)
	gradeRepo := graderepo.NewStudentSemesterGradeRepository(db)
	appealRepo := graderepo.NewGradeAppealRepository(db)
	warningRepo := graderepo.NewAcademicWarningRepository(db)
	transcriptRepo := graderepo.NewTranscriptRecordRepository(db)
	sourceRepo := graderepo.NewCourseGradeSourceRepository(db)
	courseRepo := courserepo.NewCourseRepository(db)

	userRepo := authrepo.NewUserRepository(db)
	opLogRepo := authrepo.NewOperationLogRepository(db)
	schoolRepo := schoolrepo.NewSchoolRepository(db)

	service := gradesvc.NewService(
		db,
		semesterRepo,
		levelRepo,
		warningCfgRepo,
		reviewRepo,
		gradeRepo,
		appealRepo,
		warningRepo,
		transcriptRepo,
		sourceRepo,
		&gradeUserSummaryQuerier{userRepo: userRepo},
		&gradeSchoolSummaryQuerier{schoolRepo: schoolRepo},
		&gradeNotificationPublisherAdapter{
			dispatcher:   notificationDispatcher,
			courseRepo:   courseRepo,
			semesterRepo: semesterRepo,
		},
		&gradeAuditLogger{repo: opLogRepo},
	)

	return &router.GradeHandlers{
		GradeHandler: gradehandler.NewGradeHandler(service),
	}, service
}

type gradeUserSummaryQuerier struct {
	userRepo authrepo.UserRepository
}

// GetUserSummary 获取单个用户摘要，供模块06只读查询复用。
func (a *gradeUserSummaryQuerier) GetUserSummary(ctx context.Context, userID int64) *gradesvc.UserSummary {
	if userID == 0 {
		return nil
	}
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	return &gradesvc.UserSummary{
		UserID:    user.ID,
		Name:      user.Name,
		StudentNo: user.StudentNo,
	}
}

// GetUserSummaries 批量获取用户摘要，避免列表场景重复查库。
func (a *gradeUserSummaryQuerier) GetUserSummaries(ctx context.Context, userIDs []int64) map[int64]*gradesvc.UserSummary {
	result := make(map[int64]*gradesvc.UserSummary)
	users, err := a.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return result
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		result[user.ID] = &gradesvc.UserSummary{
			UserID:    user.ID,
			Name:      user.Name,
			StudentNo: user.StudentNo,
		}
	}
	return result
}

type gradeSchoolSummaryQuerier struct {
	schoolRepo schoolrepo.SchoolRepository
}

// GetSchoolSummary 获取学校摘要，供成绩单等场景读取基础信息。
func (a *gradeSchoolSummaryQuerier) GetSchoolSummary(ctx context.Context, schoolID int64) *gradesvc.SchoolSummary {
	school, err := a.schoolRepo.GetByID(ctx, schoolID)
	if err != nil || school == nil {
		return nil
	}
	return &gradesvc.SchoolSummary{
		SchoolID: school.ID,
		Name:     school.Name,
		LogoURL:  school.LogoURL,
	}
}

type gradeAuditLogger struct {
	repo authrepo.OperationLogRepository
}

// Record 记录模块06关键操作审计日志。
func (a *gradeAuditLogger) Record(ctx context.Context, operatorID int64, action string, targetType string, targetID int64, detail string, ip string) error {
	if a.repo == nil {
		return nil
	}
	rawDetail, _ := json.Marshal(map[string]interface{}{"message": detail})
	return a.repo.Create(ctx, &entity.OperationLog{
		OperatorID: operatorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   &targetID,
		Detail:     rawDetail,
		IP:         ip,
	})
}

type gradeNotificationPublisherAdapter struct {
	dispatcher   notificationsvc.EventDispatcher
	courseRepo   courserepo.CourseRepository
	semesterRepo graderepo.SemesterRepository
}

// Publish 将模块06内部事件转译为模块07统一通知事件。
func (a *gradeNotificationPublisherAdapter) Publish(ctx context.Context, event string, payload map[string]interface{}) error {
	if a == nil || a.dispatcher == nil {
		return nil
	}

	switch event {
	case "grade_review_approved":
		return a.publishReviewApproved(ctx, payload)
	case "grade_review_rejected":
		return a.publishReviewRejected(ctx, payload)
	case "grade_appeal_approved":
		return a.publishAppealHandled(ctx, payload, "申诉已通过")
	case "grade_appeal_rejected":
		return a.publishAppealHandled(ctx, payload, "申诉已驳回")
	case "grade_academic_warning":
		return a.publishAcademicWarning(ctx, payload)
	default:
		return nil
	}
}

// publishReviewApproved 发送成绩审核通过后的教师与学生通知。
func (a *gradeNotificationPublisherAdapter) publishReviewApproved(ctx context.Context, payload map[string]interface{}) error {
	courseID := mapInt64(payload, "course_id")
	course, semesterName := a.loadCourseContext(ctx, courseID, mapInt64(payload, "semester_id"))
	courseName := ""
	if course != nil {
		courseName = course.Title
	}

	if teacherID := mapInt64(payload, "teacher_id"); teacherID > 0 {
		req := &dto.InternalSendNotificationEventReq{
			EventType:    "grade.review_approved",
			ReceiverIDs:  []string{fmt.Sprintf("%d", teacherID)},
			Params:       map[string]interface{}{"course_name": courseName, "semester_name": semesterName, "audience": "teacher"},
			SourceModule: "module_06",
			SourceType:   "grade_review",
			SourceID:     fmt.Sprintf("%d", mapInt64(payload, "review_id")),
		}
		if err := a.dispatcher.DispatchEvent(ctx, req); err != nil {
			return err
		}
	}

	studentIDs := mapInt64Slice(payload, "student_ids")
	if len(studentIDs) == 0 {
		return nil
	}
	return a.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:    "grade.review_approved",
		ReceiverIDs:  stringifyIDs(studentIDs),
		Params:       map[string]interface{}{"course_name": courseName, "semester_name": semesterName},
		SourceModule: "module_06",
		SourceType:   "grade_review",
		SourceID:     fmt.Sprintf("%d", mapInt64(payload, "review_id")),
	})
}

// publishReviewRejected 发送成绩审核驳回通知给课程教师。
func (a *gradeNotificationPublisherAdapter) publishReviewRejected(ctx context.Context, payload map[string]interface{}) error {
	teacherID := mapInt64(payload, "teacher_id")
	if teacherID == 0 {
		return nil
	}
	course, _ := a.loadCourseContext(ctx, mapInt64(payload, "course_id"), mapInt64(payload, "semester_id"))
	courseName := ""
	if course != nil {
		courseName = course.Title
	}
	return a.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:    "grade.review_rejected",
		ReceiverIDs:  []string{fmt.Sprintf("%d", teacherID)},
		Params:       map[string]interface{}{"course_name": courseName, "reason": mapString(payload, "comment")},
		SourceModule: "module_06",
		SourceType:   "grade_review",
		SourceID:     fmt.Sprintf("%d", mapInt64(payload, "review_id")),
	})
}

// publishAppealHandled 发送成绩申诉处理结果给学生。
func (a *gradeNotificationPublisherAdapter) publishAppealHandled(ctx context.Context, payload map[string]interface{}, result string) error {
	studentID := mapInt64(payload, "student_id")
	if studentID == 0 {
		return nil
	}
	course, _ := a.loadCourseContext(ctx, mapInt64(payload, "course_id"), 0)
	courseName := ""
	if course != nil {
		courseName = course.Title
	}
	return a.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:    "grade.appeal_handled",
		ReceiverIDs:  []string{fmt.Sprintf("%d", studentID)},
		Params:       map[string]interface{}{"course_name": courseName, "result": result},
		SourceModule: "module_06",
		SourceType:   "grade_appeal",
		SourceID:     fmt.Sprintf("%d", mapInt64(payload, "appeal_id")),
	})
}

// publishAcademicWarning 发送学业预警通知。
func (a *gradeNotificationPublisherAdapter) publishAcademicWarning(ctx context.Context, payload map[string]interface{}) error {
	studentID := mapInt64(payload, "student_id")
	if studentID == 0 {
		return nil
	}
	return a.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:    "grade.academic_warning",
		ReceiverIDs:  []string{fmt.Sprintf("%d", studentID)},
		Params:       map[string]interface{}{"gpa": payload["gpa"], "threshold": payload["threshold"]},
		SourceModule: "module_06",
		SourceType:   "academic_warning",
		SourceID:     fmt.Sprintf("%d", mapInt64(payload, "warning_id")),
	})
}

// loadCourseContext 加载课程与学期展示名称。
func (a *gradeNotificationPublisherAdapter) loadCourseContext(ctx context.Context, courseID, semesterID int64) (*entity.Course, string) {
	var course *entity.Course
	if a.courseRepo != nil && courseID > 0 {
		course, _ = a.courseRepo.GetByID(ctx, courseID)
	}
	name := ""
	if a.semesterRepo != nil && semesterID > 0 {
		if semester, err := a.semesterRepo.GetByID(ctx, semesterID); err == nil && semester != nil {
			name = semester.Name
		}
	}
	return course, name
}

// mapInt64 从事件载荷中读取 int64 值。
func mapInt64(payload map[string]interface{}, key string) int64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

// mapString 从事件载荷中读取字符串值。
func mapString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

// mapInt64Slice 从事件载荷中读取 int64 切片。
func mapInt64Slice(payload map[string]interface{}, key string) []int64 {
	if payload == nil {
		return []int64{}
	}
	switch values := payload[key].(type) {
	case []int64:
		return values
	case []interface{}:
		result := make([]int64, 0, len(values))
		for _, value := range values {
			switch typed := value.(type) {
			case int64:
				result = append(result, typed)
			case int:
				result = append(result, int64(typed))
			case float64:
				result = append(result, int64(typed))
			}
		}
		return result
	default:
		return []int64{}
	}
}

// stringifyIDs 将 ID 切片转为字符串切片。
func stringifyIDs(values []int64) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		result = append(result, fmt.Sprintf("%d", value))
	}
	return result
}
