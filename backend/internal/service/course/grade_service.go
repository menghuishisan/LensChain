// grade_service.go
// 模块03 — 课程与教学：单课程成绩与统计业务逻辑
// 负责成绩汇总、手动调分、我的成绩、作业统计等单课程能力

package course

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
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

// CourseOverviewQuerier 课程概览统计查询接口
// 统计导出需要复用课程概览已有实现，避免在成绩服务里复制第二套聚合逻辑。
type CourseOverviewQuerier interface {
	GetCourseOverview(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseOverviewStatsResp, error)
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
	SetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeConfigReq) error
	GetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeConfigResp, error)
	GetGradeSummary(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeSummaryResp, error)
	AdjustGrade(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64, req *dto.AdjustGradeReq) error
	GetMyGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.MyGradesResp, error)
	GetAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.AssignmentStatsResp, error)
	ExportGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error)
	ExportCourseStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error)
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
	courseOverview     CourseOverviewQuerier
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
	courseOverview CourseOverviewQuerier,
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
		courseOverview:     courseOverview,
	}
}

type gradeConfigPayload struct {
	Items []dto.GradeConfigItem `json:"items"`
}

// SetGradeConfig 配置课程成绩权重
func (s *gradeService) SetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeConfigReq) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	if err := ensureCourseWriteAllowed(course); err != nil {
		return err
	}
	if err := s.ensureCourseGradesUnlocked(ctx, courseID); err != nil {
		return err
	}
	if err := s.validateGradeConfigItems(ctx, courseID, req.Items); err != nil {
		return err
	}

	var totalWeight float64
	for _, item := range req.Items {
		totalWeight += item.Weight
	}
	if math.Abs(totalWeight-100) > 0.0001 {
		return errcode.ErrInvalidParams.WithMessage("权重总和必须为100%")
	}

	configJSON, err := json.Marshal(req)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("配置格式错误")
	}

	config := &entity.CourseGradeConfig{
		CourseID: courseID,
		Config:   configJSON,
	}
	return s.gradeConfigRepo.Upsert(ctx, config)
}

// validateGradeConfigItems 校验成绩配置项是否合法。
// 成绩权重只能引用当前课程下的作业，且同一作业在配置中只能出现一次，
// 否则会导致成绩计算与导出出现重复或跨课程污染。
func (s *gradeService) validateGradeConfigItems(ctx context.Context, courseID int64, items []dto.GradeConfigItem) error {
	seenAssignmentIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := seenAssignmentIDs[item.AssignmentID]; exists {
			return errcode.ErrInvalidParams.WithMessage("成绩配置中存在重复作业")
		}
		seenAssignmentIDs[item.AssignmentID] = struct{}{}

		assignmentID, err := snowflake.ParseString(item.AssignmentID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("成绩配置中的作业ID无效")
		}
		assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("成绩配置中的作业不存在")
		}
		if assignment.CourseID != courseID {
			return errcode.ErrInvalidParams.WithMessage("成绩配置中的作业不属于当前课程")
		}
	}
	return nil
}

// GetGradeConfig 获取课程成绩权重配置
func (s *gradeService) GetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeConfigResp, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, err
	}

	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return &dto.GradeConfigResp{Items: config.Items}, nil
}

// GetGradeSummary 获取课程成绩汇总
func (s *gradeService) GetGradeSummary(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeSummaryResp, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, err
	}

	items, err := s.buildGradeSummary(ctx, courseID)
	if err != nil {
		return nil, err
	}
	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}
	students := make([]dto.GradeSummaryItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		students = append(students, *item)
	}
	return &dto.GradeSummaryResp{
		GradeConfig: dto.GradeConfigResp{Items: config.Items},
		Students:    students,
	}, nil
}

// AdjustGrade 手动调整课程最终成绩
func (s *gradeService) AdjustGrade(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64, req *dto.AdjustGradeReq) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	if err := ensureCourseWriteAllowed(course); err != nil {
		return err
	}
	if err := s.ensureCourseGradesUnlocked(ctx, courseID); err != nil {
		return err
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
	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}

	return &dto.MyGradesResp{
		GradeConfig:   dto.GradeConfigResp{Items: config.Items},
		Scores:        record.Scores,
		WeightedTotal: record.WeightedTotal,
		FinalScore:    record.FinalScore,
		IsAdjusted:    record.IsAdjusted,
	}, nil
}

// GetAssignmentStatistics 获取课程作业统计
func (s *gradeService) GetAssignmentStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.AssignmentStatsResp, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, err
	}

	assignments, err := s.listAllAssignmentsByCourse(ctx, courseID)
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

	assignmentIDs := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentIDs = append(assignmentIDs, assignment.ID)
	}

	latestSubmissions, err := s.submissionRepo.ListLatestByAssignments(ctx, assignmentIDs)
	if err != nil {
		return nil, err
	}
	submissionsByAssignment := make(map[int64][]*entity.AssignmentSubmission, len(assignments))
	for _, submission := range latestSubmissions {
		submissionsByAssignment[submission.AssignmentID] = append(submissionsByAssignment[submission.AssignmentID], submission)
	}

	for _, assignment := range assignments {
		submissions := submissionsByAssignment[assignment.ID]

		submitCount := len(submissions)
		sum := 0.0
		maxScore := 0.0
		minScore := 0.0
		scoredCount := 0
		scoreDistribution := buildScoreDistribution()
		hasScore := false
		for _, submission := range submissions {
			score := extractSubmissionScore(submission)
			if score == nil {
				continue
			}
			sum += *score
			scoredCount++
			if !hasScore || *score > maxScore {
				maxScore = *score
			}
			if !hasScore || *score < minScore {
				minScore = *score
			}
			appendScoreDistribution(scoreDistribution, *score)
			hasScore = true
		}

		avgScore := 0.0
		if hasScore && scoredCount > 0 {
			avgScore = sum / float64(scoredCount)
		}

		submitRate := 0.0
		if totalStudents > 0 {
			submitRate = float64(submitCount) / float64(totalStudents) * 100
		}

		resp.Assignments = append(resp.Assignments, dto.AssignmentStatItem{
			ID:                strconv.FormatInt(assignment.ID, 10),
			Title:             assignment.Title,
			SubmitCount:       submitCount,
			TotalStudents:     totalStudents,
			SubmitRate:        round2(submitRate),
			AvgScore:          round2(avgScore),
			MaxScore:          round2(maxScore),
			MinScore:          round2(minScore),
			ScoreDistribution: scoreDistribution,
		})
	}

	return resp, nil
}

// ensureCourseGradesUnlocked 校验课程成绩当前未被模块06锁定。
// 成绩权重修改与手动调分都会改变课程最终成绩结果，锁定后必须统一拒绝。
func (s *gradeService) ensureCourseGradesUnlocked(ctx context.Context, courseID int64) error {
	locked, err := s.gradeLockChecker.IsCourseGradeLocked(ctx, courseID)
	if err != nil {
		return err
	}
	if locked {
		return errcode.ErrGradeLocked.WithMessage("成绩已锁定，如需修改请联系学校管理员解锁")
	}
	return nil
}

// listAllAssignmentsByCourse 分页拉取课程下全部作业，避免统计场景只汇总第一页数据。
func (s *gradeService) listAllAssignmentsByCourse(ctx context.Context, courseID int64) ([]*entity.Assignment, error) {
	const pageSize = 1000

	page := 1
	assignments := make([]*entity.Assignment, 0)
	for {
		pageItems, total, err := s.assignmentRepo.ListByCourseID(ctx, &courserepo.AssignmentListParams{
			CourseID: courseID,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}

		assignments = append(assignments, pageItems...)
		if len(assignments) >= int(total) || len(pageItems) < pageSize {
			break
		}
		page++
	}

	return assignments, nil
}

// ExportGrades 导出课程成绩汇总 Excel
func (s *gradeService) ExportGrades(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error) {
	resp, err := s.GetGradeSummary(ctx, sc, courseID)
	if err != nil {
		return nil, "", err
	}

	headers := buildGradeExportHeaders(resp.GradeConfig.Items)
	rows := make([][]interface{}, 0, len(resp.Students))
	for _, item := range resp.Students {
		studentNo := ""
		if item.StudentNo != nil {
			studentNo = *item.StudentNo
		}
		row := []interface{}{studentNo, item.StudentName}
		for _, configItem := range resp.GradeConfig.Items {
			score, ok := item.Scores[configItem.AssignmentID]
			if ok {
				row = append(row, score)
				continue
			}
			row = append(row, "")
		}
		row = append(row, item.WeightedTotal, item.FinalScore, item.IsAdjusted)
		rows = append(rows, row)
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

// ExportCourseStatistics 导出课程统计报告 Excel
// 按文档要求同时导出课程概览与作业统计，避免统计页下载结果只覆盖其中一个视图。
func (s *gradeService) ExportCourseStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*bytes.Buffer, string, error) {
	overview, err := s.GetCourseOverview(ctx, sc, courseID)
	if err != nil {
		return nil, "", err
	}
	stats, err := s.GetAssignmentStatistics(ctx, sc, courseID)
	if err != nil {
		return nil, "", err
	}

	overviewRows := [][]interface{}{
		{"学生数", overview.StudentCount},
		{"课时数", overview.LessonCount},
		{"作业数", overview.AssignmentCount},
		{"平均进度(%)", overview.AvgProgress},
		{"平均分", overview.AvgScore},
		{"完课率(%)", overview.CompletionRate},
		{"活跃度(%)", overview.ActivityRate},
		{"总学习时长(小时)", round2(overview.TotalStudyHours)},
		{"未开始占比(%)", overview.ProgressDistribution.NotStartedRate},
		{"进行中占比(%)", overview.ProgressDistribution.InProgressRate},
		{"已完成占比(%)", overview.ProgressDistribution.CompletedRate},
	}

	assignmentRows := make([][]interface{}, 0, len(stats.Assignments))
	for _, item := range stats.Assignments {
		assignmentRows = append(assignmentRows, []interface{}{
			item.Title,
			item.SubmitCount,
			item.TotalStudents,
			item.SubmitRate,
			item.AvgScore,
			item.MaxScore,
			item.MinScore,
			formatScoreDistribution(item.ScoreDistribution),
		})
	}

	buf, err := excelpkg.ExportWorkbook([]*excelpkg.SheetConfig{
		{
			SheetName: "课程概览",
			Headers:   []string{"指标", "值"},
			Rows:      overviewRows,
			ColWidths: []float64{24, 18},
		},
		{
			SheetName: "作业统计",
			Headers:   []string{"作业名称", "提交人数", "课程学生数", "提交率(%)", "平均分", "最高分", "最低分", "分数分布"},
			Rows:      assignmentRows,
			ColWidths: []float64{28, 12, 14, 12, 12, 12, 12, 32},
		},
	})
	if err != nil {
		return nil, "", err
	}
	return buf, "课程统计报告.xlsx", nil
}

// GetCourseOverview 获取课程概览统计
// 成绩服务通过接口复用学习进度服务已有统计实现，避免复制聚合逻辑。
func (s *gradeService) GetCourseOverview(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseOverviewStatsResp, error) {
	if s.courseOverview == nil {
		return nil, errcode.ErrInternal
	}
	return s.courseOverview.GetCourseOverview(ctx, sc, courseID)
}

type studentGradeRecord struct {
	Scores        map[string]float64
	WeightedTotal float64
	FinalScore    float64
	IsAdjusted    bool
}

// buildGradeSummary 构建课程成绩汇总列表
// 文档定义成绩汇总表为完整班级成绩表，因此这里必须基于全量选课名单计算，不能隐式分页。
func (s *gradeService) buildGradeSummary(ctx context.Context, courseID int64) ([]*dto.GradeSummaryItem, error) {
	enrollments, err := s.enrollmentRepo.ListAllByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	records, err := s.calculateCourseGradeRecords(ctx, courseID, enrollments)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.GradeSummaryItem, 0, len(enrollments))
	for _, enrollment := range enrollments {
		record := records[enrollment.StudentID]
		if record == nil {
			record = &studentGradeRecord{
				Scores:        map[string]float64{},
				WeightedTotal: 0,
				FinalScore:    0,
				IsAdjusted:    false,
			}
		}
		summary := s.userSummaryQuerier.GetUserSummary(ctx, enrollment.StudentID)
		name := ""
		var studentNo *string
		if summary != nil {
			name = summary.Name
			studentNo = summary.StudentNo
		}

		items = append(items, &dto.GradeSummaryItem{
			StudentID:     strconv.FormatInt(enrollment.StudentID, 10),
			StudentName:   name,
			StudentNo:     studentNo,
			Scores:        record.Scores,
			WeightedTotal: record.WeightedTotal,
			FinalScore:    record.FinalScore,
			IsAdjusted:    record.IsAdjusted,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].StudentID < items[j].StudentID
	})
	return items, nil
}

// calculateCourseGradeRecords 批量计算课程下所有学生的成绩记录。
// 成绩汇总与导出面向整班数据，必须复用仓储层的批量查询能力，避免按学生逐条查询导致 N+1。
func (s *gradeService) calculateCourseGradeRecords(ctx context.Context, courseID int64, enrollments []*entity.CourseEnrollment) (map[int64]*studentGradeRecord, error) {
	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}

	records := make(map[int64]*studentGradeRecord, len(enrollments))
	for _, enrollment := range enrollments {
		records[enrollment.StudentID] = &studentGradeRecord{
			Scores:        map[string]float64{},
			WeightedTotal: 0,
			FinalScore:    0,
			IsAdjusted:    false,
		}
	}
	if len(config.Items) == 0 || len(enrollments) == 0 {
		return records, nil
	}

	assignmentIDs := make([]int64, 0, len(config.Items))
	assignmentIDStrings := make(map[int64]string, len(config.Items))
	for _, item := range config.Items {
		assignmentID, err := snowflake.ParseString(item.AssignmentID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("成绩配置中的作业ID无效")
		}
		assignmentIDs = append(assignmentIDs, assignmentID)
		assignmentIDStrings[assignmentID] = item.AssignmentID
	}

	submissions, err := s.submissionRepo.ListLatestByAssignments(ctx, assignmentIDs)
	if err != nil {
		return nil, err
	}
	for _, submission := range submissions {
		record, exists := records[submission.StudentID]
		if !exists {
			continue
		}
		score := extractSubmissionScore(submission)
		if score == nil {
			continue
		}

		assignmentIDString, ok := assignmentIDStrings[submission.AssignmentID]
		if !ok {
			continue
		}
		record.Scores[assignmentIDString] = round2(*score)
	}

	for _, item := range config.Items {
		for _, enrollment := range enrollments {
			record := records[enrollment.StudentID]
			if score, exists := record.Scores[item.AssignmentID]; exists {
				record.WeightedTotal += score * item.Weight / 100
			}
		}
	}

	overrides, err := s.gradeOverrideRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	overrideMap := make(map[int64]*entity.CourseGradeOverride, len(overrides))
	for _, override := range overrides {
		overrideMap[override.StudentID] = override
	}

	for _, enrollment := range enrollments {
		record := records[enrollment.StudentID]
		record.WeightedTotal = round2(record.WeightedTotal)
		record.FinalScore = record.WeightedTotal
		if override, exists := overrideMap[enrollment.StudentID]; exists {
			record.FinalScore = round2(override.FinalScore)
			record.IsAdjusted = true
		}
	}

	return records, nil
}

// calculateStudentGrade 计算单个学生在某门课程中的成绩结果
// 先按成绩配置汇总各作业分数，再叠加手动调分记录得到最终分。
func (s *gradeService) calculateStudentGrade(ctx context.Context, courseID, studentID int64) (*studentGradeRecord, error) {
	config, err := s.loadGradeConfig(ctx, courseID)
	if err != nil {
		return nil, err
	}

	scores := make(map[string]float64, len(config.Items))
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
		if score != nil {
			scores[item.AssignmentID] = round2(*score)
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
		Scores:        scores,
		WeightedTotal: weightedTotal,
		FinalScore:    finalScore,
		IsAdjusted:    isAdjusted,
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

// buildScoreDistribution 初始化固定分段的分数分布结构。
func buildScoreDistribution() []dto.ScoreDistributionItem {
	return []dto.ScoreDistributionItem{
		{Range: "90-100", Count: 0},
		{Range: "80-89", Count: 0},
		{Range: "70-79", Count: 0},
		{Range: "60-69", Count: 0},
		{Range: "0-59", Count: 0},
	}
}

// appendScoreDistribution 将分数计入固定分段。
func appendScoreDistribution(distribution []dto.ScoreDistributionItem, score float64) {
	switch {
	case score >= 90:
		distribution[0].Count++
	case score >= 80:
		distribution[1].Count++
	case score >= 70:
		distribution[2].Count++
	case score >= 60:
		distribution[3].Count++
	default:
		distribution[4].Count++
	}
}

// formatScoreDistribution 将分数分布转为导出列使用的文本。
func formatScoreDistribution(distribution []dto.ScoreDistributionItem) string {
	result := ""
	for index, item := range distribution {
		if index > 0 {
			result += "；"
		}
		result += item.Range + ":" + strconv.Itoa(item.Count)
	}
	return result
}

// buildGradeExportHeaders 生成成绩导出表头，确保包含各项成绩列。
// 成绩配置中的 name 已是模块03对外暴露的稳定展示名，导出应与其保持一致，避免再回查作业表产生额外耦合。
func buildGradeExportHeaders(items []dto.GradeConfigItem) []string {
	headers := []string{"学号", "姓名"}
	for _, item := range items {
		title := item.Name
		if title == "" {
			title = "作业"
		}
		headers = append(headers, title)
	}
	headers = append(headers, "加权总成绩", "最终成绩", "是否已调整")
	return headers
}
