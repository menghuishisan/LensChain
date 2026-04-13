// grade_service.go
// 模块03 — 课程与教学：单课程成绩与统计业务逻辑
// 负责成绩汇总、手动调分、我的成绩、作业统计等单课程能力

package course

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	excelpkg "github.com/lenschain/backend/internal/pkg/excel"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// GradeLockChecker 成绩锁定检查接口
// 用于查询课程成绩是否已被模块06审核锁定。
// 在模块06尚未接入时，模块03通过默认空实现保持单模块能力可用。
type GradeLockChecker interface {
	IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error)
}

type noopGradeLockChecker struct{}

// IsCourseGradeLocked 默认返回未锁定
// 作为模块06接入前的兜底实现，避免模块03直接依赖聚合层。
func (noopGradeLockChecker) IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error) {
	return false, nil
}

// GradeService 单课程成绩服务接口
// 负责课程内成绩汇总、手动调分、学生查看本人课程成绩、作业统计等能力。
type GradeService interface {
	GetGradeSummary(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeSummaryReq) ([]*dto.GradeSummaryItem, int64, error)
	AdjustGrade(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64, req *dto.AdjustGradeReq) error
	GetMyGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.MyGradesResp, error)
	GetAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.AssignmentStatsResp, error)
	ExportGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error)
	ExportAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error)
}

type gradeService struct {
	courseRepo         courserepo.CourseRepository
	enrollmentRepo     courserepo.EnrollmentRepository
	assignmentRepo     courserepo.AssignmentRepository
	submissionRepo     courserepo.SubmissionRepository
	gradeConfigRepo    courserepo.GradeConfigRepository
	gradeOverrideRepo  courserepo.GradeOverrideRepository
	userSummaryQuerier UserSummaryQuerier
	gradeLockChecker   GradeLockChecker
}

// NewGradeService 创建单课程成绩服务实例
// 组合课程、选课、作业、提交、成绩配置、调分记录等依赖，提供模块03成绩管理能力。
func NewGradeService(
	courseRepo courserepo.CourseRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	assignmentRepo courserepo.AssignmentRepository,
	submissionRepo courserepo.SubmissionRepository,
	gradeConfigRepo courserepo.GradeConfigRepository,
	gradeOverrideRepo courserepo.GradeOverrideRepository,
	userSummaryQuerier UserSummaryQuerier,
	gradeLockChecker GradeLockChecker,
) GradeService {
	if gradeLockChecker == nil {
		gradeLockChecker = noopGradeLockChecker{}
	}
	return &gradeService{
		courseRepo:         courseRepo,
		enrollmentRepo:     enrollmentRepo,
		assignmentRepo:     assignmentRepo,
		submissionRepo:     submissionRepo,
		gradeConfigRepo:    gradeConfigRepo,
		gradeOverrideRepo:  gradeOverrideRepo,
		userSummaryQuerier: userSummaryQuerier,
		gradeLockChecker:   gradeLockChecker,
	}
}

type gradeConfigPayload struct {
	Items []dto.GradeConfigItem `json:"items"`
}

// GetGradeSummary 获取课程成绩汇总
func (s *gradeService) GetGradeSummary(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeSummaryReq) ([]*dto.GradeSummaryItem, int64, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, 0, err
	}

	items, total, err := s.buildGradeSummary(ctx, courseID, req)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// AdjustGrade 手动调整课程最终成绩
func (s *gradeService) AdjustGrade(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64, req *dto.AdjustGradeReq) error {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return err
	}

	locked, err := s.gradeLockChecker.IsCourseGradeLocked(ctx, courseID)
	if err != nil {
		return err
	}
	if locked {
		return errcode.ErrForbidden.WithMessage("成绩已锁定，如需修改请联系学校管理员解锁")
	}

	enrolled, err := s.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
	if err != nil {
		return err
	}
	if !enrolled {
		return errcode.ErrNotCourseStudent
	}

	record, err := s.calculateStudentGrade(ctx, courseID, studentID)
	if err != nil {
		return err
	}

	override := &entity.CourseGradeOverride{
		CourseID:      courseID,
		StudentID:     studentID,
		WeightedTotal: record.WeightedTotal,
		FinalScore:    req.FinalScore,
		AdjustReason:  req.Reason,
		AdjustedBy:    sc.UserID,
		AdjustedAt:    time.Now(),
	}
	return s.gradeOverrideRepo.Upsert(ctx, override)
}

// GetMyGrades 获取学生在课程内的成绩
func (s *gradeService) GetMyGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.MyGradesResp, error) {
	if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, err
	}

	record, err := s.calculateStudentGrade(ctx, courseID, sc.UserID)
	if err != nil {
		return nil, err
	}

	return &dto.MyGradesResp{
		AssignmentScores: record.AssignmentScores,
		WeightedTotal:    record.WeightedTotal,
		FinalScore:       record.FinalScore,
		IsAdjusted:       record.IsAdjusted,
	}, nil
}

// GetAssignmentStatistics 获取课程作业统计
func (s *gradeService) GetAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.AssignmentStatsResp, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, err
	}

	assignments, _, err := s.assignmentRepo.ListByCourseID(ctx, &courserepo.AssignmentListParams{
		CourseID: courseID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}

	totalStudents, err := s.courseRepo.CountStudents(ctx, courseID)
	if err != nil {
		return nil, err
	}

	resp := &dto.AssignmentStatsResp{
		Assignments: make([]dto.AssignmentStatItem, 0, len(assignments)),
	}

	for _, assignment := range assignments {
		submissions, _, err := s.submissionRepo.ListByAssignment(ctx, &courserepo.SubmissionListParams{
			AssignmentID: assignment.ID,
			Page:         1,
			PageSize:     1000,
		})
		if err != nil {
			return nil, err
		}

		submitCount := len(submissions)
		sum := 0.0
		maxScore := 0.0
		minScore := 0.0
		hasScore := false
		for _, submission := range submissions {
			score := extractSubmissionScore(submission)
			if score == nil {
				continue
			}
			sum += *score
			if !hasScore || *score > maxScore {
				maxScore = *score
			}
			if !hasScore || *score < minScore {
				minScore = *score
			}
			hasScore = true
		}

		avgScore := 0.0
		if hasScore && submitCount > 0 {
			avgScore = sum / float64(submitCount)
		}

		submitRate := 0.0
		if totalStudents > 0 {
			submitRate = float64(submitCount) / float64(totalStudents) * 100
		}

		resp.Assignments = append(resp.Assignments, dto.AssignmentStatItem{
			ID:            strconv.FormatInt(assignment.ID, 10),
			Title:         assignment.Title,
			SubmitCount:   submitCount,
			TotalStudents: totalStudents,
			SubmitRate:    round2(submitRate),
			AvgScore:      round2(avgScore),
			MaxScore:      round2(maxScore),
			MinScore:      round2(minScore),
		})
	}

	return resp, nil
}

// ExportGrades 导出课程成绩汇总 Excel
func (s *gradeService) ExportGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error) {
	items, _, err := s.GetGradeSummary(ctx, sc, courseID, &dto.GradeSummaryReq{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, "", err
	}

	headers := []string{"学号", "姓名", "加权总成绩", "最终成绩", "是否已调整"}
	rows := make([][]interface{}, 0, len(items))
	for _, item := range items {
		studentNo := ""
		if item.StudentNo != nil {
			studentNo = *item.StudentNo
		}
		rows = append(rows, []interface{}{
			studentNo,
			item.StudentName,
			item.WeightedTotal,
			item.FinalScore,
			item.IsAdjusted,
		})
	}

	buf, err := excelpkg.Export(&excelpkg.ExportConfig{
		SheetName: "课程成绩",
		Headers:   headers,
	}, rows)
	if err != nil {
		return nil, "", err
	}
	return buf, "课程成绩单.xlsx", nil
}

// ExportAssignmentStatistics 导出课程作业统计 Excel
func (s *gradeService) ExportAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error) {
	stats, err := s.GetAssignmentStatistics(ctx, sc, courseID)
	if err != nil {
		return nil, "", err
	}

	headers := []string{"作业名称", "提交人数", "课程学生数", "提交率(%)", "平均分", "最高分", "最低分"}
	rows := make([][]interface{}, 0, len(stats.Assignments))
	for _, item := range stats.Assignments {
		rows = append(rows, []interface{}{
			item.Title,
			item.SubmitCount,
			item.TotalStudents,
			item.SubmitRate,
			item.AvgScore,
			item.MaxScore,
			item.MinScore,
		})
	}

	buf, err := excelpkg.Export(&excelpkg.ExportConfig{
		SheetName: "作业统计",
		Headers:   headers,
	}, rows)
	if err != nil {
		return nil, "", err
	}
	return buf, "课程作业统计.xlsx", nil
}

type studentGradeRecord struct {
	AssignmentScores []dto.AssignmentScoreItem
	WeightedTotal    float64
	FinalScore       float64
	IsAdjusted       bool
}

// buildGradeSummary 构建课程成绩汇总列表
// 基于选课学生列表逐个计算单课程成绩，保持模块03只处理课程内数据。
func (s *gradeService) buildGradeSummary(ctx context.Context, courseID int64, req *dto.GradeSummaryReq) ([]*dto.GradeSummaryItem, int64, error) {
	enrollments, total, err := s.enrollmentRepo.List(ctx, &courserepo.EnrollmentListParams{
		CourseID: courseID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.GradeSummaryItem, 0, len(enrollments))
	for _, enrollment := range enrollments {
		record, err := s.calculateStudentGrade(ctx, courseID, enrollment.StudentID)
		if err != nil {
			return nil, 0, err
		}
		summary := s.userSummaryQuerier.GetUserSummary(ctx, enrollment.StudentID)
		name := ""
		var studentNo *string
		if summary != nil {
			name = summary.Name
			studentNo = summary.StudentNo
		}

		items = append(items, &dto.GradeSummaryItem{
			StudentID:        strconv.FormatInt(enrollment.StudentID, 10),
			StudentName:      name,
			StudentNo:        studentNo,
			AssignmentScores: record.AssignmentScores,
			WeightedTotal:    record.WeightedTotal,
			FinalScore:       record.FinalScore,
			IsAdjusted:       record.IsAdjusted,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].StudentID < items[j].StudentID
	})
	return items, total, nil
}

// calculateStudentGrade 计算单个学生在某门课程中的成绩结果
// 先按成绩配置汇总各作业分数，再叠加手动调分记录得到最终分。
func (s *gradeService) calculateStudentGrade(ctx context.Context, courseID, studentID int64) (*studentGradeRecord, error) {
	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}

	assignmentScores := make([]dto.AssignmentScoreItem, 0, len(config.Items))
	weightedTotal := 0.0
	for _, item := range config.Items {
		assignmentID, err := snowflake.ParseString(item.AssignmentID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("成绩配置中的作业ID无效")
		}

		latest, err := s.submissionRepo.GetLatestByStudentAndAssignment(ctx, studentID, assignmentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		score := extractSubmissionScore(latest)
		assignmentScores = append(assignmentScores, dto.AssignmentScoreItem{
			AssignmentID: item.AssignmentID,
			Name:         item.Name,
			Score:        score,
			Weight:       item.Weight,
		})

		if score != nil {
			weightedTotal += *score * item.Weight / 100
		}
	}

	weightedTotal = round2(weightedTotal)
	finalScore := weightedTotal
	isAdjusted := false

	override, err := s.gradeOverrideRepo.GetByCourseAndStudent(ctx, courseID, studentID)
	if err == nil && override != nil {
		finalScore = round2(override.FinalScore)
		isAdjusted = true
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &studentGradeRecord{
		AssignmentScores: assignmentScores,
		WeightedTotal:    weightedTotal,
		FinalScore:       finalScore,
		IsAdjusted:       isAdjusted,
	}, nil
}

// loadGradeConfig 加载课程成绩权重配置
// 未配置时返回空配置，避免调用方再处理 nil 分支。
func (s *gradeService) loadGradeConfig(ctx context.Context, courseID int64) (*gradeConfigPayload, error) {
	config, err := s.gradeConfigRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &gradeConfigPayload{Items: []dto.GradeConfigItem{}}, nil
		}
		return nil, err
	}

	payload := &gradeConfigPayload{}
	if err := json.Unmarshal([]byte(config.Config), payload); err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("成绩配置解析失败")
	}
	return payload, nil
}

// extractSubmissionScore 提取提交记录中的最终有效得分
// 优先使用扣分后的最终得分，若不存在则回退到总得分。
func extractSubmissionScore(submission *entity.AssignmentSubmission) *float64 {
	if submission == nil {
		return nil
	}
	if submission.ScoreAfterDeduction != nil {
		score := round2(*submission.ScoreAfterDeduction)
		return &score
	}
	if submission.TotalScore != nil {
		score := round2(*submission.TotalScore)
		return &score
	}
	return nil
}

// round2 保留两位小数
// 统一成绩计算结果的展示与导出精度。
func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
