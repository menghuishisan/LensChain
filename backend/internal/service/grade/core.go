// core.go
// 模块06 — 评测与成绩：聚合层核心类型与依赖装配。
// 该文件集中定义服务接口、跨模块查询契约和公共辅助结构，避免在各功能文件里重复声明。

package grade

import (
	"context"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/pagination"
	graderepo "github.com/lenschain/backend/internal/repository/grade"
)

// UserSummary 跨模块用户摘要。
type UserSummary struct {
	UserID    int64
	Name      string
	StudentNo *string
}

// SchoolSummary 跨模块学校摘要。
type SchoolSummary struct {
	SchoolID int64
	Name     string
	LogoURL  *string
}

// UserSummaryQuerier 提供最小用户摘要查询能力。
type UserSummaryQuerier interface {
	GetUserSummary(ctx context.Context, userID int64) *UserSummary
	GetUserSummaries(ctx context.Context, userIDs []int64) map[int64]*UserSummary
	GetUserSchoolID(ctx context.Context, userID int64) (int64, error)
}

// SchoolSummaryQuerier 提供最小学校摘要查询能力。
type SchoolSummaryQuerier interface {
	GetSchoolSummary(ctx context.Context, schoolID int64) *SchoolSummary
}

// GradeEventPublisher 提供模块06到模块07的通知解耦接口。
type GradeEventPublisher interface {
	Publish(ctx context.Context, event string, payload map[string]interface{}) error
}

// AuditLogger 提供模块06到模块01操作日志的解耦接口。
type AuditLogger interface {
	Record(ctx context.Context, operatorID int64, action string, targetType string, targetID int64, detail string, ip string) error
}

// Service 模块06统一服务接口。
type Service interface {
	CreateSemester(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SemesterReq) (*dto.GradeSemesterResp, error)
	ListSemesters(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SemesterListReq) (*dto.GradeSemesterListResp, error)
	UpdateSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SemesterReq) error
	DeleteSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SetCurrentSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	GetLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.GradeLevelConfigResp, error)
	UpdateLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateGradeLevelConfigsReq) (*dto.GradeLevelConfigResp, error)
	ResetDefaultLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.GradeLevelConfigResp, error)

	SubmitReview(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SubmitGradeReviewReq) (*dto.GradeReviewStatusResp, error)
	ListReviews(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GradeReviewListReq) (*dto.GradeReviewListResp, error)
	GetReview(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GradeReviewDetailResp, error)
	ApproveReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewHandleReq) error
	RejectReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewHandleReq) error
	UnlockReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UnlockGradeReviewReq) error
	IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error)

	GetStudentSemesterGrades(ctx context.Context, sc *svcctx.ServiceContext, studentID int64, req *dto.SemesterGradesReq) (*dto.SemesterGradesResp, error)
	GetStudentGPA(ctx context.Context, sc *svcctx.ServiceContext, studentID int64) (*dto.GPAResp, error)
	GetLearningOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.LearningOverviewResp, error)

	CreateAppeal(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateGradeAppealReq) (*dto.GradeAppealDetailResp, error)
	ListAppeals(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GradeAppealListReq) (*dto.GradeAppealListResp, error)
	GetAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GradeAppealDetailResp, error)
	ApproveAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ApproveGradeAppealReq) error
	RejectAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.RejectGradeAppealReq) error

	ListWarnings(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AcademicWarningListReq) (*dto.AcademicWarningListResp, error)
	GetWarning(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AcademicWarningDetailResp, error)
	HandleWarning(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAcademicWarningReq) error
	GetWarningConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.WarningConfigResp, error)
	UpdateWarningConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateWarningConfigReq) (*dto.WarningConfigResp, error)

	GenerateTranscript(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GenerateTranscriptReq) (*dto.TranscriptResp, error)
	ListTranscripts(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TranscriptListReq) (*dto.TranscriptListResp, error)
	GetTranscriptDownloadURL(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error)

	GetCourseAnalytics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseGradeAnalyticsResp, error)
	GetSchoolAnalytics(ctx context.Context, sc *svcctx.ServiceContext, semesterID int64) (*dto.SchoolGradeAnalyticsResp, error)
	GetPlatformAnalytics(ctx context.Context, sc *svcctx.ServiceContext, semesterID int64) (*dto.PlatformGradeAnalyticsResp, error)
}

type service struct {
	db             *gorm.DB
	semesterRepo   graderepo.SemesterRepository
	levelRepo      graderepo.GradeLevelConfigRepository
	warningCfgRepo graderepo.WarningConfigRepository
	reviewRepo     graderepo.GradeReviewRepository
	gradeRepo      graderepo.StudentSemesterGradeRepository
	appealRepo     graderepo.GradeAppealRepository
	warningRepo    graderepo.AcademicWarningRepository
	transcriptRepo graderepo.TranscriptRecordRepository
	sourceRepo     graderepo.CourseGradeSourceRepository
	userQuerier    UserSummaryQuerier
	schoolQuerier  SchoolSummaryQuerier
	eventPublisher GradeEventPublisher
	auditLogger    AuditLogger
}

// NewService 创建模块06服务实例。
func NewService(
	db *gorm.DB,
	semesterRepo graderepo.SemesterRepository,
	levelRepo graderepo.GradeLevelConfigRepository,
	warningCfgRepo graderepo.WarningConfigRepository,
	reviewRepo graderepo.GradeReviewRepository,
	gradeRepo graderepo.StudentSemesterGradeRepository,
	appealRepo graderepo.GradeAppealRepository,
	warningRepo graderepo.AcademicWarningRepository,
	transcriptRepo graderepo.TranscriptRecordRepository,
	sourceRepo graderepo.CourseGradeSourceRepository,
	userQuerier UserSummaryQuerier,
	schoolQuerier SchoolSummaryQuerier,
	eventPublisher GradeEventPublisher,
	auditLogger AuditLogger,
) Service {
	return &service{
		db:             db,
		semesterRepo:   semesterRepo,
		levelRepo:      levelRepo,
		warningCfgRepo: warningCfgRepo,
		reviewRepo:     reviewRepo,
		gradeRepo:      gradeRepo,
		appealRepo:     appealRepo,
		warningRepo:    warningRepo,
		transcriptRepo: transcriptRepo,
		sourceRepo:     sourceRepo,
		userQuerier:    userQuerier,
		schoolQuerier:  schoolQuerier,
		eventPublisher: eventPublisher,
		auditLogger:    auditLogger,
	}
}

// normalizePagination 统一分页默认值。
func normalizePagination(page, pageSize int) (int, int) {
	return pagination.NormalizeValues(page, pageSize)
}

// buildPaginationResp 构建模块06通用分页结构。
func buildPaginationResp(page, pageSize int, total int64) dto.PaginationResp {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return dto.PaginationResp{
		Page:       page,
		PageSize:   pageSize,
		Total:      int(total),
		TotalPages: totalPages,
	}
}

// formatDate 将时间格式化为日期字符串。
func formatDate(value time.Time) string {
	return value.Format("2006-01-02")
}

// formatDateTime 将时间格式化为 RFC3339 字符串。
func formatDateTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

// stringPtr 将字符串转为指针。
func stringPtr(value string) *string {
	return &value
}

// intPtr 将整数转为指针。
func intPtr(value int) *int {
	return &value
}

// round2 将浮点数保留两位小数。
func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// uniqueInt64s 对 ID 切片去重并排序。
func uniqueInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	set := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// ensureSchoolScope 校验模块06的学校边界。
func ensureSchoolScope(sc *svcctx.ServiceContext) error {
	if sc == nil {
		return errcode.ErrUnauthorized
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if sc.SchoolID == 0 {
		return errcode.ErrForbidden
	}
	return nil
}

// buildDefaultLevelConfigs 构建默认五级制。
func buildDefaultLevelConfigs(schoolID int64) []*entity.GradeLevelConfig {
	return []*entity.GradeLevelConfig{
		{SchoolID: schoolID, LevelName: "A", MinScore: 90, MaxScore: 100, GPAPoint: 4, SortOrder: 1},
		{SchoolID: schoolID, LevelName: "B", MinScore: 80, MaxScore: 89.99, GPAPoint: 3, SortOrder: 2},
		{SchoolID: schoolID, LevelName: "C", MinScore: 70, MaxScore: 79.99, GPAPoint: 2, SortOrder: 3},
		{SchoolID: schoolID, LevelName: "D", MinScore: 60, MaxScore: 69.99, GPAPoint: 1, SortOrder: 4},
		{SchoolID: schoolID, LevelName: "F", MinScore: 0, MaxScore: 59.99, GPAPoint: 0, SortOrder: 5},
	}
}
