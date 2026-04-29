// query_service.go
// 模块06 — 评测与成绩：成绩查询、申诉、预警、成绩单和分析能力。
// 该文件承载模块06的读取侧与后续处理链路，和管理主流程分开以保持职责清晰。

package grade

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/pdf"
	snow "github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/storage"
	graderepo "github.com/lenschain/backend/internal/repository/grade"
)

// GetStudentSemesterGrades 获取学生学期成绩。
func (s *service) GetStudentSemesterGrades(ctx context.Context, sc *svcctx.ServiceContext, studentID int64, req *dto.SemesterGradesReq) (*dto.SemesterGradesResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc.IsStudent() && studentID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	semesterID, err := s.resolveSemesterID(ctx, sc.SchoolID, req.SemesterID)
	if err != nil {
		return nil, err
	}
	grades, err := s.gradeRepo.ListByStudent(ctx, sc.SchoolID, studentID, []int64{semesterID})
	if err != nil {
		return nil, err
	}
	if err := s.ensureTeacherCanAccessStudentGrades(ctx, sc, grades); err != nil {
		return nil, err
	}
	semester, _ := s.semesterRepo.GetByID(ctx, semesterID)
	teacherMap := make(map[int64]*UserSummary)
	items := make([]dto.SemesterGradeItem, 0, len(grades))
	passedCount := 0
	totalCredits := 0.0
	for _, grade := range grades {
		course, err := s.sourceRepo.GetCourse(ctx, grade.CourseID)
		if err != nil || course == nil {
			continue
		}
		if teacherMap[course.TeacherID] == nil && s.userQuerier != nil {
			teacherMap[course.TeacherID] = s.userQuerier.GetUserSummary(ctx, course.TeacherID)
		}
		reviewStatus := "approved"
		reviewStatusText := "已审核"
		if grade.ReviewID > 0 {
			if review, err := s.reviewRepo.GetByID(ctx, grade.ReviewID); err == nil && review != nil {
				reviewStatus = reviewStatusKey(review.Status)
				reviewStatusText = enum.GetGradeReviewStatusText(review.Status)
			}
		}
		if grade.FinalScore >= 60 {
			passedCount++
		}
		totalCredits += grade.Credits
		items = append(items, dto.SemesterGradeItem{
			GradeID:          fmt.Sprintf("%d", grade.ID),
			CourseID:         fmt.Sprintf("%d", grade.CourseID),
			CourseName:       course.Title,
			TeacherName:      userName(teacherMap[course.TeacherID]),
			Credits:          grade.Credits,
			FinalScore:       grade.FinalScore,
			GradeLevel:       grade.GradeLevel,
			GPAPoint:         grade.GPAPoint,
			IsAdjusted:       grade.IsAdjusted,
			ReviewStatus:     reviewStatus,
			ReviewStatusText: reviewStatusText,
		})
	}
	stats, err := s.gradeRepo.CalculateSemesterGPA(ctx, sc.SchoolID, studentID, semesterID)
	if err != nil {
		return nil, err
	}
	resp := &dto.SemesterGradesResp{
		Semester: &dto.SemesterInfo{
			ID:   fmt.Sprintf("%d", semesterID),
			Name: semesterName(semester),
			Code: semesterCode(semester),
		},
		Grades: items,
		Summary: &dto.SemesterGradeSummary{
			TotalCredits: totalCredits,
			SemesterGPA:  stats.GPA,
			CourseCount:  len(items),
			PassedCount:  passedCount,
			FailedCount:  len(items) - passedCount,
		},
	}
	return resp, nil
}

// GetStudentGPA 获取学生 GPA。
func (s *service) GetStudentGPA(ctx context.Context, sc *svcctx.ServiceContext, studentID int64) (*dto.GPAResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc.IsStudent() && studentID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	grades, err := s.gradeRepo.ListByStudent(ctx, sc.SchoolID, studentID, nil)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTeacherCanAccessStudentGrades(ctx, sc, grades); err != nil {
		return nil, err
	}
	grouped := make(map[int64][]*entity.StudentSemesterGrade)
	for _, grade := range grades {
		if grade == nil {
			continue
		}
		grouped[grade.SemesterID] = append(grouped[grade.SemesterID], grade)
	}
	semesterIDs := make([]int64, 0, len(grouped))
	for semesterID := range grouped {
		semesterIDs = append(semesterIDs, semesterID)
	}
	semesters, _ := s.semesterRepo.ListByIDs(ctx, sc.SchoolID, semesterIDs)
	semesterMap := make(map[int64]*entity.Semester, len(semesters))
	for _, semester := range semesters {
		if semester != nil {
			semesterMap[semester.ID] = semester
		}
	}
	cumulative, err := s.gradeRepo.CalculateCumulativeGPA(ctx, sc.SchoolID, studentID)
	if err != nil {
		return nil, err
	}
	semesterList := make([]dto.GPASemesterItem, 0, len(semesterIDs))
	trend := make([]float64, 0, len(semesterIDs))
	for _, semester := range semesters {
		stats, err := s.gradeRepo.CalculateSemesterGPA(ctx, sc.SchoolID, studentID, semester.ID)
		if err != nil {
			return nil, err
		}
		semesterList = append(semesterList, dto.GPASemesterItem{
			SemesterID:   fmt.Sprintf("%d", semester.ID),
			SemesterName: semester.Name,
			GPA:          stats.GPA,
			Credits:      stats.TotalCredits,
		})
		trend = append(trend, stats.GPA)
	}
	return &dto.GPAResp{
		CumulativeGPA:     cumulative.GPA,
		CumulativeCredits: cumulative.TotalCredits,
		SemesterList:      semesterList,
		GPATrend:          trend,
	}, nil
}

// GetLearningOverview 获取当前学生学习概览。
func (s *service) GetLearningOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.LearningOverviewResp, error) {
	if sc == nil || !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	stats, err := s.sourceRepo.GetLearningOverview(ctx, sc.UserID)
	if err != nil {
		return nil, err
	}
	return &dto.LearningOverviewResp{
		CourseCount:      int(stats.CourseCount),
		ExperimentCount:  int(stats.ExperimentCount),
		CompetitionCount: int(stats.CompetitionCount),
		TotalStudyHours:  round2(float64(stats.TotalStudySeconds) / 3600),
	}, nil
}

// CreateAppeal 提交成绩申诉。
func (s *service) CreateAppeal(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateGradeAppealReq) (*dto.GradeAppealDetailResp, error) {
	if sc == nil || !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	gradeID, err := snow.ParseString(req.GradeID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	grade, err := s.gradeRepo.GetByID(ctx, gradeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotFound
		}
		return nil, err
	}
	if grade.StudentID != sc.UserID || grade.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	review, err := s.reviewRepo.GetByID(ctx, grade.ReviewID)
	if err != nil || review == nil || review.Status != enum.GradeReviewStatusApproved {
		return nil, errcode.ErrInvalidParams.WithMessage("成绩尚未审核通过，不可申诉")
	}
	if review.ReviewedAt != nil && time.Since(*review.ReviewedAt) > 30*24*time.Hour {
		return nil, errcode.ErrInvalidParams.WithMessage("已超过申诉时效（30天）")
	}
	if len([]rune(req.AppealReason)) < 20 {
		return nil, errcode.ErrAppealReasonTooShort
	}
	if _, err := s.appealRepo.GetByStudentCourseSemester(ctx, grade.StudentID, grade.CourseID, grade.SemesterID); err == nil {
		return nil, errcode.ErrAppealAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	appeal := &entity.GradeAppeal{
		StudentID:     grade.StudentID,
		SchoolID:      grade.SchoolID,
		SemesterID:    grade.SemesterID,
		CourseID:      grade.CourseID,
		GradeID:       grade.ID,
		OriginalScore: grade.FinalScore,
		AppealReason:  req.AppealReason,
		Status:        enum.GradeAppealStatusPending,
	}
	if err := s.appealRepo.Create(ctx, appeal); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "grade_appeal_created", map[string]interface{}{
		"appeal_id":  appeal.ID,
		"student_id": appeal.StudentID,
		"course_id":  appeal.CourseID,
		"teacher_id": review.SubmittedBy,
	})
	return s.GetAppeal(ctx, sc, appeal.ID)
}

// ListAppeals 获取申诉列表。
func (s *service) ListAppeals(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GradeAppealListReq) (*dto.GradeAppealListResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc == nil || sc.IsSchoolAdmin() || sc.IsSuperAdmin() || (!sc.IsStudent() && !sc.IsTeacher()) {
		return nil, errcode.ErrForbidden
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	params := &graderepo.GradeAppealListParams{
		SchoolID: sc.SchoolID,
		Status:   req.Status,
		Page:     page,
		PageSize: pageSize,
	}
	if sc.IsStudent() {
		params.StudentID = sc.UserID
	} else if sc.IsTeacher() {
		params.TeacherID = sc.UserID
	}
	if req.CourseID != "" {
		if courseID, err := snow.ParseString(req.CourseID); err == nil {
			params.CourseID = courseID
		}
	}
	items, total, err := s.appealRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	list := make([]dto.GradeAppealItem, 0, len(items))
	userIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item != nil {
			userIDs = append(userIDs, item.StudentID)
		}
	}
	userMap := map[int64]*UserSummary{}
	if s.userQuerier != nil {
		userMap = s.userQuerier.GetUserSummaries(ctx, uniqueInt64s(userIDs))
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		course, _ := s.sourceRepo.GetCourse(ctx, item.CourseID)
		semester, _ := s.semesterRepo.GetByID(ctx, item.SemesterID)
		list = append(list, dto.GradeAppealItem{
			ID:            fmt.Sprintf("%d", item.ID),
			StudentID:     fmt.Sprintf("%d", item.StudentID),
			StudentName:   userName(userMap[item.StudentID]),
			CourseID:      fmt.Sprintf("%d", item.CourseID),
			CourseName:    courseName(course),
			SemesterID:    fmt.Sprintf("%d", item.SemesterID),
			SemesterName:  semesterName(semester),
			OriginalScore: item.OriginalScore,
			Status:        item.Status,
			StatusText:    enum.GetGradeAppealStatusText(item.Status),
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &dto.GradeAppealListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetAppeal 获取申诉详情。
func (s *service) GetAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GradeAppealDetailResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc == nil || sc.IsSchoolAdmin() || sc.IsSuperAdmin() || (!sc.IsStudent() && !sc.IsTeacher()) {
		return nil, errcode.ErrForbidden
	}
	appeal, err := s.appealRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAppealNotFound
		}
		return nil, err
	}
	if appeal.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	if sc.IsStudent() && appeal.StudentID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	if sc.IsTeacher() {
		course, courseErr := s.sourceRepo.GetCourse(ctx, appeal.CourseID)
		if courseErr != nil {
			return nil, courseErr
		}
		if course == nil || course.TeacherID != sc.UserID {
			return nil, errcode.ErrForbidden
		}
	}
	student := s.userQuerier.GetUserSummary(ctx, appeal.StudentID)
	course, _ := s.sourceRepo.GetCourse(ctx, appeal.CourseID)
	semester, _ := s.semesterRepo.GetByID(ctx, appeal.SemesterID)
	var handledBy *UserSummary
	if appeal.HandledBy != nil {
		handledBy = s.userQuerier.GetUserSummary(ctx, *appeal.HandledBy)
	}
	return &dto.GradeAppealDetailResp{
		ID:            fmt.Sprintf("%d", appeal.ID),
		StudentID:     fmt.Sprintf("%d", appeal.StudentID),
		StudentName:   userName(student),
		CourseID:      fmt.Sprintf("%d", appeal.CourseID),
		CourseName:    courseName(course),
		SemesterID:    fmt.Sprintf("%d", appeal.SemesterID),
		SemesterName:  semesterName(semester),
		GradeID:       fmt.Sprintf("%d", appeal.GradeID),
		OriginalScore: appeal.OriginalScore,
		AppealReason:  appeal.AppealReason,
		Status:        appeal.Status,
		StatusText:    enum.GetGradeAppealStatusText(appeal.Status),
		HandledBy:     optionalIDString(appeal.HandledBy),
		HandledByName: optionalUserName(handledBy),
		HandledAt:     formatDateTime(appeal.HandledAt),
		NewScore:      appeal.NewScore,
		HandleComment: appeal.HandleComment,
		CreatedAt:     appeal.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     appeal.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ApproveAppeal 同意申诉。
func (s *service) ApproveAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ApproveGradeAppealReq) error {
	if sc == nil || !sc.IsTeacher() {
		return errcode.ErrForbidden
	}
	appeal, err := s.appealRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAppealNotFound
		}
		return err
	}
	course, err := s.sourceRepo.GetCourse(ctx, appeal.CourseID)
	if err != nil {
		return err
	}
	if course.TeacherID != sc.UserID {
		return errcode.ErrForbidden
	}
	if appeal.Status != enum.GradeAppealStatusPending {
		return errcode.ErrAppealNotPending
	}
	levelConfigs, err := s.levelRepo.ListBySchool(ctx, appeal.SchoolID)
	if err != nil {
		return err
	}
	if len(levelConfigs) == 0 {
		levelConfigs = buildDefaultLevelConfigs(appeal.SchoolID)
	}
	level := matchLevel(levelConfigs, req.NewScore)
	if level == nil {
		return errcode.ErrLevelConfigNotCovered
	}
	now := time.Now()
	if err := s.appealRepo.Approve(ctx, id, sc.UserID, req.NewScore, stringPtr(req.HandleComment), now); err != nil {
		return err
	}
	if err := s.gradeRepo.UpdateAfterAppeal(ctx, appeal.GradeID, req.NewScore, level.LevelName, level.GPAPoint); err != nil {
		return err
	}
	if err := s.refreshWarningsForSemester(ctx, appeal.SchoolID, appeal.SemesterID, levelConfigs); err != nil {
		return err
	}
	s.publishEvent(ctx, "grade_appeal_approved", map[string]interface{}{
		"appeal_id":  appeal.ID,
		"student_id": appeal.StudentID,
		"course_id":  appeal.CourseID,
	})
	return nil
}

// RejectAppeal 驳回申诉。
func (s *service) RejectAppeal(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.RejectGradeAppealReq) error {
	if sc == nil || !sc.IsTeacher() {
		return errcode.ErrForbidden
	}
	appeal, err := s.appealRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAppealNotFound
		}
		return err
	}
	course, err := s.sourceRepo.GetCourse(ctx, appeal.CourseID)
	if err != nil {
		return err
	}
	if course.TeacherID != sc.UserID {
		return errcode.ErrForbidden
	}
	if appeal.Status != enum.GradeAppealStatusPending {
		return errcode.ErrAppealNotPending
	}
	if err := s.appealRepo.Reject(ctx, id, sc.UserID, stringPtr(req.HandleComment), time.Now()); err != nil {
		return err
	}
	s.publishEvent(ctx, "grade_appeal_rejected", map[string]interface{}{
		"appeal_id":  appeal.ID,
		"student_id": appeal.StudentID,
		"course_id":  appeal.CourseID,
	})
	return nil
}

// ListWarnings 获取预警列表。
func (s *service) ListWarnings(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AcademicWarningListReq) (*dto.AcademicWarningListResp, error) {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	params := &graderepo.AcademicWarningListParams{
		SchoolID:    sc.SchoolID,
		WarningType: req.WarningType,
		Status:      req.Status,
		Keyword:     req.Keyword,
		Page:        page,
		PageSize:    pageSize,
	}
	if req.SemesterID != "" {
		if semesterID, err := snow.ParseString(req.SemesterID); err == nil {
			params.SemesterID = semesterID
		}
	}
	items, total, err := s.warningRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	list := make([]dto.AcademicWarningItem, 0, len(items))
	userIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item != nil {
			userIDs = append(userIDs, item.StudentID)
		}
	}
	userMap := map[int64]*UserSummary{}
	if s.userQuerier != nil {
		userMap = s.userQuerier.GetUserSummaries(ctx, uniqueInt64s(userIDs))
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		semester, _ := s.semesterRepo.GetByID(ctx, item.SemesterID)
		detail, _ := decodeWarningDetail(item.Detail)
		list = append(list, dto.AcademicWarningItem{
			ID:              fmt.Sprintf("%d", item.ID),
			StudentID:       fmt.Sprintf("%d", item.StudentID),
			StudentName:     userName(userMap[item.StudentID]),
			StudentNo:       studentNo(userMap[item.StudentID]),
			SemesterName:    semesterName(semester),
			WarningType:     item.WarningType,
			WarningTypeText: enum.GetAcademicWarningTypeText(item.WarningType),
			Detail:          detail,
			Status:          item.Status,
			StatusText:      enum.GetAcademicWarningStatusText(item.Status),
			CreatedAt:       item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &dto.AcademicWarningListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetWarning 获取预警详情。
func (s *service) GetWarning(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AcademicWarningDetailResp, error) {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	warning, err := s.warningRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrWarningNotFound
		}
		return nil, err
	}
	if !sc.IsSuperAdmin() && warning.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	student := s.userQuerier.GetUserSummary(ctx, warning.StudentID)
	semester, _ := s.semesterRepo.GetByID(ctx, warning.SemesterID)
	detail, _ := decodeWarningDetail(warning.Detail)
	var handledBy *UserSummary
	if warning.HandledBy != nil {
		handledBy = s.userQuerier.GetUserSummary(ctx, *warning.HandledBy)
	}
	return &dto.AcademicWarningDetailResp{
		ID:              fmt.Sprintf("%d", warning.ID),
		StudentID:       fmt.Sprintf("%d", warning.StudentID),
		StudentName:     userName(student),
		StudentNo:       studentNo(student),
		SemesterID:      fmt.Sprintf("%d", warning.SemesterID),
		SemesterName:    semesterName(semester),
		WarningType:     warning.WarningType,
		WarningTypeText: enum.GetAcademicWarningTypeText(warning.WarningType),
		Detail:          detail,
		Status:          warning.Status,
		StatusText:      enum.GetAcademicWarningStatusText(warning.Status),
		HandledBy:       optionalIDString(warning.HandledBy),
		HandledByName:   optionalUserName(handledBy),
		HandledAt:       formatDateTime(warning.HandledAt),
		HandleNote:      warning.HandleNote,
		CreatedAt:       warning.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       warning.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// HandleWarning 处理预警。
func (s *service) HandleWarning(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAcademicWarningReq) error {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	warning, err := s.warningRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrWarningNotFound
		}
		return err
	}
	if !sc.IsSuperAdmin() && warning.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	return s.warningRepo.Handle(ctx, id, sc.UserID, stringPtr(req.HandleNote), time.Now())
}

// GetWarningConfig 获取预警配置。
func (s *service) GetWarningConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.WarningConfigResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	config, err := s.warningCfgRepo.GetBySchool(ctx, sc.SchoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = &entity.WarningConfig{SchoolID: sc.SchoolID, GPAThreshold: 2, FailCountThreshold: 2, IsEnabled: true}
		} else {
			return nil, err
		}
	}
	return &dto.WarningConfigResp{
		SchoolID:           fmt.Sprintf("%d", config.SchoolID),
		GPAThreshold:       config.GPAThreshold,
		FailCountThreshold: config.FailCountThreshold,
		IsEnabled:          config.IsEnabled,
	}, nil
}

// UpdateWarningConfig 更新预警配置。
func (s *service) UpdateWarningConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateWarningConfigReq) (*dto.WarningConfigResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	config := &entity.WarningConfig{
		SchoolID:           sc.SchoolID,
		GPAThreshold:       req.GPAThreshold,
		FailCountThreshold: req.FailCountThreshold,
		IsEnabled:          req.IsEnabled,
	}
	if err := s.warningCfgRepo.Upsert(ctx, config); err != nil {
		return nil, err
	}
	return s.GetWarningConfig(ctx, sc)
}

// GenerateTranscript 生成成绩单。
func (s *service) GenerateTranscript(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GenerateTranscriptReq) (*dto.TranscriptResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc == nil || sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	studentID, err := s.resolveTranscriptStudentID(sc, req)
	if err != nil {
		return nil, err
	}
	semesterIDs := make([]int64, 0, len(req.SemesterIDs))
	for _, item := range req.SemesterIDs {
		id, err := snow.ParseString(item)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		semesterIDs = append(semesterIDs, id)
	}
	if err := s.ensureTranscriptAccess(ctx, sc, studentID, semesterIDs); err != nil {
		return nil, err
	}
	transcriptData, err := s.buildTranscriptData(ctx, sc.SchoolID, studentID, semesterIDs)
	if err != nil {
		return nil, err
	}
	buf, err := pdf.GenerateTranscript(transcriptData)
	if err != nil {
		return nil, err
	}
	objectName := pdf.SuggestObjectName(transcriptData)
	if _, err := storage.UploadFile(ctx, objectName, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "application/pdf"); err != nil {
		return nil, err
	}
	record := &entity.TranscriptRecord{
		SchoolID:         sc.SchoolID,
		StudentID:        studentID,
		GeneratedBy:      sc.UserID,
		FileURL:          objectName,
		FileSize:         int64(buf.Len()),
		IncludeSemesters: mustJSON(semesterIDs),
		GeneratedAt:      time.Now(),
	}
	if err := s.transcriptRepo.Create(ctx, record); err != nil {
		return nil, err
	}
	return &dto.TranscriptResp{
		ID:          fmt.Sprintf("%d", record.ID),
		FileURL:     fmt.Sprintf("/api/v1/grades/transcripts/%d/download", record.ID),
		GeneratedAt: record.GeneratedAt.UTC().Format(time.RFC3339),
		StudentID:   stringPtr(fmt.Sprintf("%d", studentID)),
		FileSize:    &record.FileSize,
	}, nil
}

// ListTranscripts 获取成绩单列表。
func (s *service) ListTranscripts(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TranscriptListReq) (*dto.TranscriptListResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if sc == nil || sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	params := &graderepo.TranscriptRecordListParams{
		SchoolID: sc.SchoolID,
		Page:     page,
		PageSize: pageSize,
	}
	if sc.IsStudent() {
		params.StudentID = sc.UserID
	} else if sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		params.GeneratedBy = sc.UserID
		if req.StudentID != "" {
			studentID, err := snow.ParseString(req.StudentID)
			if err != nil {
				return nil, errcode.ErrInvalidID
			}
			if err := s.ensureTranscriptAccess(ctx, sc, studentID, nil); err != nil {
				return nil, err
			}
			params.StudentID = studentID
		}
	} else if req.StudentID != "" {
		studentID, err := snow.ParseString(req.StudentID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.StudentID = studentID
	}
	items, total, err := s.transcriptRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item != nil {
			userIDs = append(userIDs, item.StudentID)
		}
	}
	userMap := map[int64]*UserSummary{}
	if s.userQuerier != nil {
		userMap = s.userQuerier.GetUserSummaries(ctx, uniqueInt64s(userIDs))
	}
	list := make([]dto.TranscriptListItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		var semesterIDs []int64
		_ = json.Unmarshal(item.IncludeSemesters, &semesterIDs)
		includeSemesters := make([]string, 0, len(semesterIDs))
		for _, semesterID := range semesterIDs {
			includeSemesters = append(includeSemesters, fmt.Sprintf("%d", semesterID))
		}
		list = append(list, dto.TranscriptListItem{
			ID:               fmt.Sprintf("%d", item.ID),
			StudentID:        fmt.Sprintf("%d", item.StudentID),
			StudentName:      userName(userMap[item.StudentID]),
			FileURL:          fmt.Sprintf("/api/v1/grades/transcripts/%d/download", item.ID),
			FileSize:         item.FileSize,
			IncludeSemesters: includeSemesters,
			GeneratedAt:      item.GeneratedAt.UTC().Format(time.RFC3339),
			ExpiresAt:        formatDateTime(item.ExpiresAt),
		})
	}
	return &dto.TranscriptListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetTranscriptDownloadURL 获取成绩单下载 URL。
func (s *service) GetTranscriptDownloadURL(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return "", err
	}
	if sc == nil || sc.IsSuperAdmin() {
		return "", errcode.ErrForbidden
	}
	record, err := s.transcriptRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if sc.IsStudent() && record.StudentID != sc.UserID {
		return "", errcode.ErrForbidden
	}
	if err := s.ensureTranscriptAccess(ctx, sc, record.StudentID, nil); err != nil {
		return "", err
	}
	return storage.GetFileURL(ctx, record.FileURL, time.Hour)
}

// GetCourseAnalytics 获取课程成绩分析。
func (s *service) GetCourseAnalytics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseGradeAnalyticsResp, error) {
	if sc == nil || !sc.IsTeacher() {
		return nil, errcode.ErrForbidden
	}
	course, err := s.sourceRepo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course.TeacherID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	stats, err := s.gradeRepo.CourseAnalytics(ctx, course.SchoolID, courseID)
	if err != nil {
		return nil, err
	}
	grades, err := s.gradeRepo.CourseGradeDistribution(ctx, course.SchoolID, courseID)
	if err != nil {
		return nil, err
	}
	scoreDist, err := s.gradeRepo.CourseScoreDistribution(ctx, course.SchoolID, courseID)
	if err != nil {
		return nil, err
	}
	levelDist := make(map[string]int, len(grades))
	for _, item := range grades {
		if item != nil {
			levelDist[item.GradeLevel] = int(item.Count)
		}
	}
	semesterNameValue := ""
	if course.SemesterID != nil {
		if semester, err := s.semesterRepo.GetByID(ctx, *course.SemesterID); err == nil && semester != nil {
			semesterNameValue = semester.Name
		}
	}
	resp := &dto.CourseGradeAnalyticsResp{
		CourseID:          fmt.Sprintf("%d", courseID),
		CourseName:        course.Title,
		SemesterName:      semesterNameValue,
		StudentCount:      int(stats.StudentCount),
		AverageScore:      stats.AverageScore,
		MedianScore:       stats.MedianScore,
		MaxScore:          stats.HighestScore,
		MinScore:          stats.LowestScore,
		PassRate:          stats.PassRate,
		GradeDistribution: levelDist,
		ScoreDistribution: make([]dto.ScoreDistributionItem, 0, len(scoreDist)),
	}
	for _, item := range scoreDist {
		if item != nil {
			resp.ScoreDistribution = append(resp.ScoreDistribution, dto.ScoreDistributionItem{Range: item.Range, Count: int(item.Count)})
		}
	}
	return resp, nil
}

// GetSchoolAnalytics 获取全校成绩分析。
func (s *service) GetSchoolAnalytics(ctx context.Context, sc *svcctx.ServiceContext, semesterID int64) (*dto.SchoolGradeAnalyticsResp, error) {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	stats, err := s.gradeRepo.SchoolAnalytics(ctx, sc.SchoolID, semesterID)
	if err != nil {
		return nil, err
	}
	failRate, err := s.gradeRepo.SchoolFailRate(ctx, sc.SchoolID, semesterID)
	if err != nil {
		return nil, err
	}
	gpaDist, err := s.gradeRepo.SchoolGPADistribution(ctx, sc.SchoolID, semesterID)
	if err != nil {
		return nil, err
	}
	topCourses, err := s.gradeRepo.SchoolCoursePerformance(ctx, sc.SchoolID, semesterID, false, 5)
	if err != nil {
		return nil, err
	}
	bottomCourses, err := s.gradeRepo.SchoolCoursePerformance(ctx, sc.SchoolID, semesterID, true, 5)
	if err != nil {
		return nil, err
	}
	semester, _ := s.semesterRepo.GetByID(ctx, semesterID)
	resp := &dto.SchoolGradeAnalyticsResp{
		SemesterName:    semesterName(semester),
		TotalStudents:   int(stats.StudentCount),
		TotalCourses:    int(stats.CourseCount),
		AverageGPA:      stats.AverageGPA,
		GPADistribution: make([]dto.ScoreDistributionItem, 0, len(gpaDist)),
		FailRate:        failRate,
		WarningCount:    int(stats.WarningCount),
		TopCourses:      make([]dto.CoursePerformanceItem, 0, len(topCourses)),
		BottomCourses:   make([]dto.CoursePerformanceItem, 0, len(bottomCourses)),
	}
	for _, item := range gpaDist {
		if item != nil {
			resp.GPADistribution = append(resp.GPADistribution, dto.ScoreDistributionItem{Range: item.Range, Count: int(item.Count)})
		}
	}
	for _, item := range topCourses {
		if item != nil {
			resp.TopCourses = append(resp.TopCourses, dto.CoursePerformanceItem{
				CourseName:   item.CourseName,
				AverageScore: item.AverageScore,
				PassRate:     item.PassRate,
			})
		}
	}
	for _, item := range bottomCourses {
		if item != nil {
			resp.BottomCourses = append(resp.BottomCourses, dto.CoursePerformanceItem{
				CourseName:   item.CourseName,
				AverageScore: item.AverageScore,
				PassRate:     item.PassRate,
			})
		}
	}
	return resp, nil
}

// GetPlatformAnalytics 获取平台成绩总览。
func (s *service) GetPlatformAnalytics(ctx context.Context, sc *svcctx.ServiceContext, semesterID int64) (*dto.PlatformGradeAnalyticsResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	stats, err := s.gradeRepo.PlatformAnalytics(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	comparison, err := s.gradeRepo.PlatformSchoolComparison(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	resp := &dto.PlatformGradeAnalyticsResp{
		TotalSchools:       int(stats.SchoolCount),
		TotalStudents:      int(stats.StudentCount),
		PlatformAverageGPA: stats.AverageGPA,
		SchoolComparison:   make([]dto.SchoolComparisonItem, 0, len(comparison)),
	}
	for _, item := range comparison {
		if item != nil {
			resp.SchoolComparison = append(resp.SchoolComparison, dto.SchoolComparisonItem{
				SchoolName:   item.SchoolName,
				StudentCount: int(item.StudentCount),
				AverageGPA:   item.AverageGPA,
			})
		}
	}
	return resp, nil
}

// resolveSemesterID 解析查询学期，未指定时回退当前学期。
func (s *service) resolveSemesterID(ctx context.Context, schoolID int64, rawID string) (int64, error) {
	if rawID != "" {
		return snow.ParseString(rawID)
	}
	current, err := s.semesterRepo.GetCurrent(ctx, schoolID)
	if err != nil {
		return 0, errcode.ErrSemesterNotFound
	}
	return current.ID, nil
}

// buildTranscriptData 构建成绩单渲染数据。
func (s *service) buildTranscriptData(ctx context.Context, schoolID, studentID int64, semesterIDs []int64) (*pdf.TranscriptData, error) {
	student := s.userQuerier.GetUserSummary(ctx, studentID)
	if student == nil {
		return nil, errcode.ErrNotFound
	}
	var school *SchoolSummary
	if s.schoolQuerier != nil {
		school = s.schoolQuerier.GetSchoolSummary(ctx, schoolID)
	}
	grades, err := s.gradeRepo.ListByStudent(ctx, schoolID, studentID, semesterIDs)
	if err != nil {
		return nil, err
	}
	grouped := make(map[int64][]entity.StudentSemesterGrade)
	for _, grade := range grades {
		if grade == nil {
			continue
		}
		grouped[grade.SemesterID] = append(grouped[grade.SemesterID], *grade)
	}
	semesters, _ := s.semesterRepo.ListByIDs(ctx, schoolID, semesterIDs)
	pdfSemesters := make([]pdf.SemesterGrades, 0, len(semesters))
	totalCredits := 0.0
	for _, semester := range semesters {
		if semester == nil {
			continue
		}
		rows := grouped[semester.ID]
		courses := make([]pdf.CourseGrade, 0, len(rows))
		for _, row := range rows {
			course, _ := s.sourceRepo.GetCourse(ctx, row.CourseID)
			courses = append(courses, pdf.CourseGrade{
				CourseName: courseName(course),
				Credits:    row.Credits,
				Score:      row.FinalScore,
				GradeLevel: row.GradeLevel,
				GPAPoint:   row.GPAPoint,
			})
			totalCredits += row.Credits
		}
		stats, err := s.gradeRepo.CalculateSemesterGPA(ctx, schoolID, studentID, semester.ID)
		if err != nil {
			return nil, err
		}
		pdfSemesters = append(pdfSemesters, pdf.SemesterGrades{
			SemesterName: semester.Name,
			Courses:      courses,
			SemesterGPA:  stats.GPA,
		})
	}
	cumulative, err := s.gradeRepo.CalculateCumulativeGPA(ctx, schoolID, studentID)
	if err != nil {
		return nil, err
	}
	var schoolLogoReader io.Reader
	if school != nil && school.LogoURL != nil && *school.LogoURL != "" {
		if logoReader, err := storage.DownloadFile(ctx, *school.LogoURL); err == nil {
			defer logoReader.Close()
			if logoBytes, readErr := io.ReadAll(logoReader); readErr == nil && len(logoBytes) > 0 {
				schoolLogoReader = bytes.NewReader(logoBytes)
			}
		}
	}
	return &pdf.TranscriptData{
		SchoolName:    schoolName(school),
		SchoolLogo:    schoolLogoReader,
		StudentName:   userName(student),
		StudentNo:     studentNo(student),
		College:       "-",
		Major:         "-",
		Semesters:     pdfSemesters,
		CumulativeGPA: cumulative.GPA,
		TotalCredits:  totalCredits,
		GeneratedAt:   time.Now().UTC().Format("2006-01-02 15:04:05"),
		SerialNumber:  fmt.Sprintf("TRANSCRIPT-%d-%d", schoolID, time.Now().Unix()),
	}, nil
}

// resolveTranscriptStudentID 解析成绩单目标学生。
// 学生仅允许生成自己的成绩单；教师和管理员必须显式指定 student_id。
func (s *service) resolveTranscriptStudentID(sc *svcctx.ServiceContext, req *dto.GenerateTranscriptReq) (int64, error) {
	if sc == nil {
		return 0, errcode.ErrUnauthorized
	}
	if sc.IsStudent() {
		if req.StudentID != nil && *req.StudentID != "" {
			parsedID, err := snow.ParseString(*req.StudentID)
			if err != nil {
				return 0, errcode.ErrInvalidID
			}
			if parsedID != sc.UserID {
				return 0, errcode.ErrForbidden
			}
		}
		return sc.UserID, nil
	}
	if req.StudentID == nil || *req.StudentID == "" {
		return 0, errcode.ErrInvalidParams.WithMessage("教师或管理员生成成绩单时必须指定学生")
	}
	studentID, err := snow.ParseString(*req.StudentID)
	if err != nil {
		return 0, errcode.ErrInvalidID
	}
	return studentID, nil
}

// ensureTranscriptAccess 校验成绩单读取权限。
// 非管理教师只能访问自己授课学生的成绩单，避免看到同校无关学生的完整学业记录。
func (s *service) ensureTranscriptAccess(ctx context.Context, sc *svcctx.ServiceContext, studentID int64, semesterIDs []int64) error {
	if sc == nil {
		return errcode.ErrUnauthorized
	}
	if sc.IsStudent() {
		if studentID != sc.UserID {
			return errcode.ErrForbidden
		}
		return nil
	}
	if sc.IsSchoolAdmin() {
		if s.userQuerier == nil {
			return errcode.ErrForbidden
		}
		schoolID, err := s.userQuerier.GetUserSchoolID(ctx, studentID)
		if err != nil {
			return err
		}
		if schoolID != sc.SchoolID {
			return errcode.ErrForbidden
		}
		return nil
	}
	if !sc.IsTeacher() {
		return errcode.ErrForbidden
	}
	grades, err := s.gradeRepo.ListByStudent(ctx, sc.SchoolID, studentID, semesterIDs)
	if err != nil {
		return err
	}
	return s.ensureTeacherCanAccessStudentGrades(ctx, sc, grades)
}

// ensureTeacherCanAccessStudentGrades 校验教师是否有权查看指定学生成绩。
func (s *service) ensureTeacherCanAccessStudentGrades(ctx context.Context, sc *svcctx.ServiceContext, grades []*entity.StudentSemesterGrade) error {
	if sc == nil {
		return errcode.ErrUnauthorized
	}
	if !sc.IsTeacher() || sc.IsSchoolAdmin() || sc.IsSuperAdmin() {
		return nil
	}
	if s.sourceRepo == nil {
		return errcode.ErrForbidden
	}
	for _, grade := range grades {
		if grade == nil {
			continue
		}
		course, err := s.sourceRepo.GetCourse(ctx, grade.CourseID)
		if err == nil && course != nil && course.TeacherID == sc.UserID {
			return nil
		}
	}
	return errcode.ErrForbidden
}

// semesterCode 提取学期编码，统一处理空学期兜底。
func semesterCode(semester *entity.Semester) string {
	if semester == nil {
		return ""
	}
	return semester.Code
}

// reviewStatusKey 将审核状态枚举转换为文档约定的英文键名。
func reviewStatusKey(status int16) string {
	switch status {
	case enum.GradeReviewStatusApproved:
		return "approved"
	case enum.GradeReviewStatusPending:
		return "pending"
	case enum.GradeReviewStatusRejected:
		return "rejected"
	default:
		return "not_submitted"
	}
}

// studentNo 提取学生学号，统一处理空摘要场景。
func studentNo(user *UserSummary) string {
	if user == nil || user.StudentNo == nil {
		return ""
	}
	return *user.StudentNo
}

// schoolName 提取学校名称，统一处理空学校摘要场景。
func schoolName(school *SchoolSummary) string {
	if school == nil {
		return ""
	}
	return school.Name
}

// decodeWarningDetail 解码预警明细 JSON，避免在列表和详情里重复解析逻辑。
func decodeWarningDetail(raw []byte) (dto.AcademicWarningDetail, error) {
	var detail dto.AcademicWarningDetail
	err := json.Unmarshal(raw, &detail)
	return detail, err
}

// computeFailRate 基于课程通过率近似推导学校不及格率。
// 这里先复用现有课程表现聚合结果，避免在首轮实现里再增加一套重复 SQL。
func computeFailRate(topCourses []*graderepo.CoursePerformanceItem, bottomCourses []*graderepo.CoursePerformanceItem) float64 {
	totalPassRate := 0.0
	count := 0
	for _, item := range append(topCourses, bottomCourses...) {
		if item == nil {
			continue
		}
		totalPassRate += item.PassRate
		count++
	}
	if count == 0 {
		return 0
	}
	return 1 - totalPassRate/float64(count)
}

// mustJSON 将值序列化为 JSON 字节，用于存储 JSONB 字段的最小封装。
func mustJSON(value interface{}) []byte {
	raw, _ := json.Marshal(value)
	return raw
}
