// management_service.go
// 模块06 — 评测与成绩：学期、等级映射和成绩审核主流程。
// 该文件承载模块06最核心的管理类能力，避免骨架实现散落在多个小文件中。

package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	snow "github.com/lenschain/backend/internal/pkg/snowflake"
	graderepo "github.com/lenschain/backend/internal/repository/grade"
)

// CreateSemester 创建学期。
func (s *service) CreateSemester(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SemesterReq) (*dto.GradeSemesterResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	if err := validateSemesterReq(req); err != nil {
		return nil, err
	}
	if _, err := s.semesterRepo.GetByCode(ctx, sc.SchoolID, req.Code); err == nil {
		return nil, errcode.ErrDuplicateSemesterCode
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	semester := &entity.Semester{
		SchoolID:  sc.SchoolID,
		Name:      req.Name,
		Code:      req.Code,
		StartDate: startDate,
		EndDate:   endDate,
		IsCurrent: false,
	}
	if err := s.semesterRepo.Create(ctx, semester); err != nil {
		return nil, err
	}
	return buildSemesterResp(semester, nil, nil), nil
}

// ListSemesters 获取学期列表。
func (s *service) ListSemesters(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SemesterListReq) (*dto.GradeSemesterListResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	items, total, err := s.semesterRepo.List(ctx, &graderepo.SemesterListParams{
		SchoolID:  sc.SchoolID,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	list := make([]dto.GradeSemesterResp, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		courseCount, summary, _ := s.buildSemesterStats(ctx, sc.SchoolID, item.ID)
		list = append(list, *buildSemesterResp(item, courseCount, summary))
	}
	return &dto.GradeSemesterListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// UpdateSemester 更新学期。
func (s *service) UpdateSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SemesterReq) error {
	if err := ensureSchoolScope(sc); err != nil {
		return err
	}
	if err := validateSemesterReq(req); err != nil {
		return err
	}
	semester, err := s.semesterRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSemesterNotFound
		}
		return err
	}
	if semester.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	existing, err := s.semesterRepo.GetByCode(ctx, sc.SchoolID, req.Code)
	if err == nil && existing != nil && existing.ID != id {
		return errcode.ErrDuplicateSemesterCode
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	return s.semesterRepo.Update(ctx, id, map[string]interface{}{
		"name":       req.Name,
		"code":       req.Code,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

// DeleteSemester 删除学期。
func (s *service) DeleteSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if err := ensureSchoolScope(sc); err != nil {
		return err
	}
	semester, err := s.semesterRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSemesterNotFound
		}
		return err
	}
	if semester.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	return s.semesterRepo.Delete(ctx, id)
}

// SetCurrentSemester 设为当前学期。
func (s *service) SetCurrentSemester(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if err := ensureSchoolScope(sc); err != nil {
		return err
	}
	semester, err := s.semesterRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSemesterNotFound
		}
		return err
	}
	if semester.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	return s.semesterRepo.SetCurrent(ctx, id)
}

// GetLevelConfigs 获取等级映射。
func (s *service) GetLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.GradeLevelConfigResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	configs, err := s.levelRepo.ListBySchool(ctx, sc.SchoolID)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		configs = buildDefaultLevelConfigs(sc.SchoolID)
	}
	return buildLevelConfigResp(sc.SchoolID, configs), nil
}

// UpdateLevelConfigs 更新等级映射。
func (s *service) UpdateLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateGradeLevelConfigsReq) (*dto.GradeLevelConfigResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	configs, err := validateAndBuildLevelConfigs(sc.SchoolID, req)
	if err != nil {
		return nil, err
	}
	if err := s.levelRepo.ReplaceBySchool(ctx, sc.SchoolID, configs); err != nil {
		return nil, err
	}
	return buildLevelConfigResp(sc.SchoolID, configs), nil
}

// ResetDefaultLevelConfigs 重置默认等级映射。
func (s *service) ResetDefaultLevelConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.GradeLevelConfigResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	configs := buildDefaultLevelConfigs(sc.SchoolID)
	if err := s.levelRepo.ReplaceBySchool(ctx, sc.SchoolID, configs); err != nil {
		return nil, err
	}
	return buildLevelConfigResp(sc.SchoolID, configs), nil
}

// SubmitReview 提交成绩审核。
func (s *service) SubmitReview(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SubmitGradeReviewReq) (*dto.GradeReviewStatusResp, error) {
	if sc == nil || !sc.IsTeacher() {
		return nil, errcode.ErrForbidden
	}
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	courseID, err := snow.ParseString(req.CourseID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	semesterID, err := snow.ParseString(req.SemesterID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	course, err := s.sourceRepo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course.SchoolID != sc.SchoolID || course.TeacherID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	if course.Credits == nil || *course.Credits <= 0 {
		return nil, errcode.ErrInvalidParams.WithMessage("请先在课程设置中配置学分")
	}
	if course.SemesterID == nil || *course.SemesterID == 0 {
		return nil, errcode.ErrInvalidParams.WithMessage("请先关联课程所属学期")
	}
	if *course.SemesterID != semesterID {
		return nil, errcode.ErrInvalidParams.WithMessage("课程所属学期与提交学期不一致")
	}
	existing, err := s.reviewRepo.GetByCourseSemester(ctx, courseID, semesterID)
	if err == nil && existing != nil {
		if existing.Status == enum.GradeReviewStatusPending || existing.Status == enum.GradeReviewStatusApproved {
			return nil, errcode.ErrReviewAlreadyExists
		}
		now := time.Now()
		if err := s.reviewRepo.UpdateSubmitInfo(ctx, existing.ID, sc.UserID, req.SubmitNote, now); err != nil {
			return nil, err
		}
		return &dto.GradeReviewStatusResp{
			ID:          fmt.Sprintf("%d", existing.ID),
			CourseID:    req.CourseID,
			SemesterID:  req.SemesterID,
			Status:      enum.GradeReviewStatusPending,
			StatusText:  enum.GetGradeReviewStatusText(enum.GradeReviewStatusPending),
			SubmittedAt: formatDateTime(&now),
		}, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if _, _, missingCount, err := s.buildApprovedGradeRows(ctx, sc.SchoolID, 0, course); err != nil {
		return nil, err
	} else if missingCount > 0 {
		return nil, errcode.ErrGradesIncomplete.WithMessage(fmt.Sprintf("课程成绩未全部计算完成，尚有%d名学生未出成绩", missingCount))
	}
	now := time.Now()
	review := &entity.GradeReview{
		CourseID:    courseID,
		SchoolID:    sc.SchoolID,
		SemesterID:  semesterID,
		SubmittedBy: sc.UserID,
		Status:      enum.GradeReviewStatusPending,
		SubmitNote:  req.SubmitNote,
		SubmittedAt: &now,
	}
	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "grade_review_submitted", map[string]interface{}{
		"review_id":   review.ID,
		"course_id":   courseID,
		"semester_id": semesterID,
		"teacher_id":  sc.UserID,
		"school_id":   sc.SchoolID,
	})
	return &dto.GradeReviewStatusResp{
		ID:          fmt.Sprintf("%d", review.ID),
		CourseID:    req.CourseID,
		SemesterID:  req.SemesterID,
		Status:      enum.GradeReviewStatusPending,
		StatusText:  enum.GetGradeReviewStatusText(enum.GradeReviewStatusPending),
		SubmittedAt: formatDateTime(review.SubmittedAt),
	}, nil
}

// ListReviews 获取审核列表。
func (s *service) ListReviews(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GradeReviewListReq) (*dto.GradeReviewListResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	params := &graderepo.GradeReviewListParams{
		SchoolID: sc.SchoolID,
		Status:   req.Status,
		Page:     page,
		PageSize: pageSize,
	}
	if req.SemesterID != "" {
		if semesterID, err := snow.ParseString(req.SemesterID); err == nil {
			params.SemesterID = semesterID
		}
	}
	if req.CourseID != "" {
		if courseID, err := snow.ParseString(req.CourseID); err == nil {
			params.CourseID = courseID
		}
	}
	if sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		params.SubmittedBy = sc.UserID
	}
	reviews, total, err := s.reviewRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	courseMap, semesterMap, userMap := s.buildReviewLookupMaps(ctx, reviews)
	list := make([]dto.GradeReviewItem, 0, len(reviews))
	for _, review := range reviews {
		if review == nil {
			continue
		}
		course := courseMap[review.CourseID]
		semester := semesterMap[review.SemesterID]
		user := userMap[review.SubmittedBy]
		list = append(list, dto.GradeReviewItem{
			ID:              fmt.Sprintf("%d", review.ID),
			CourseID:        fmt.Sprintf("%d", review.CourseID),
			CourseName:      courseName(course),
			SemesterID:      fmt.Sprintf("%d", review.SemesterID),
			SemesterName:    semesterName(semester),
			SubmittedBy:     fmt.Sprintf("%d", review.SubmittedBy),
			SubmittedByName: userName(user),
			Status:          review.Status,
			StatusText:      enum.GetGradeReviewStatusText(review.Status),
			SubmittedAt:     formatDateTime(review.SubmittedAt),
			ReviewedAt:      formatDateTime(review.ReviewedAt),
			IsLocked:        review.IsLocked,
		})
	}
	return &dto.GradeReviewListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetReview 获取审核详情。
func (s *service) GetReview(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GradeReviewDetailResp, error) {
	if err := ensureSchoolScope(sc); err != nil {
		return nil, err
	}
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrReviewNotFound
		}
		return nil, err
	}
	if review.SchoolID != sc.SchoolID && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	courseMap, semesterMap, userMap := s.buildReviewLookupMaps(ctx, []*entity.GradeReview{review})
	submittedBy := userMap[review.SubmittedBy]
	var reviewedBy *UserSummary
	if review.ReviewedBy != nil {
		reviewedBy = s.userQuerier.GetUserSummary(ctx, *review.ReviewedBy)
	}
	return &dto.GradeReviewDetailResp{
		ID:              fmt.Sprintf("%d", review.ID),
		CourseID:        fmt.Sprintf("%d", review.CourseID),
		CourseName:      courseName(courseMap[review.CourseID]),
		SemesterID:      fmt.Sprintf("%d", review.SemesterID),
		SemesterName:    semesterName(semesterMap[review.SemesterID]),
		SubmittedBy:     fmt.Sprintf("%d", review.SubmittedBy),
		SubmittedByName: userName(submittedBy),
		Status:          review.Status,
		StatusText:      enum.GetGradeReviewStatusText(review.Status),
		SubmitNote:      review.SubmitNote,
		SubmittedAt:     formatDateTime(review.SubmittedAt),
		ReviewedBy:      optionalIDString(review.ReviewedBy),
		ReviewedByName:  optionalUserName(reviewedBy),
		ReviewedAt:      formatDateTime(review.ReviewedAt),
		ReviewComment:   review.ReviewComment,
		IsLocked:        review.IsLocked,
		LockedAt:        formatDateTime(review.LockedAt),
		UnlockedBy:      optionalIDString(review.UnlockedBy),
		UnlockedAt:      formatDateTime(review.UnlockedAt),
		UnlockReason:    review.UnlockReason,
	}, nil
}

// ApproveReview 审核通过。
func (s *service) ApproveReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewHandleReq) error {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrReviewNotFound
		}
		return err
	}
	if !sc.IsSuperAdmin() && review.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	if review.Status != enum.GradeReviewStatusPending {
		return errcode.ErrReviewNotPending
	}
	course, err := s.sourceRepo.GetCourse(ctx, review.CourseID)
	if err != nil {
		return err
	}
	levelConfigs, gradeRows, missingCount, err := s.buildApprovedGradeRows(ctx, review.SchoolID, review.ID, course)
	if err != nil {
		return err
	}
	if missingCount > 0 {
		return errcode.ErrGradesIncomplete.WithMessage(fmt.Sprintf("课程成绩未全部计算完成，尚有%d名学生未出成绩", missingCount))
	}
	now := time.Now()
	if err := s.reviewRepo.Approve(ctx, id, sc.UserID, stringPtr(req.ReviewComment), now); err != nil {
		return err
	}
	if err := s.gradeRepo.DeleteByReview(ctx, review.ID); err != nil {
		return err
	}
	if err := s.gradeRepo.BatchUpsert(ctx, gradeRows); err != nil {
		return err
	}
	if err := s.refreshWarningsForSemester(ctx, review.SchoolID, review.SemesterID, levelConfigs); err != nil {
		return err
	}
	studentIDs := make([]int64, 0, len(gradeRows))
	for _, row := range gradeRows {
		if row == nil {
			continue
		}
		studentIDs = append(studentIDs, row.StudentID)
	}
	s.publishEvent(ctx, "grade_review_approved", map[string]interface{}{
		"review_id":   review.ID,
		"course_id":   review.CourseID,
		"teacher_id":  review.SubmittedBy,
		"school_id":   review.SchoolID,
		"student_ids": uniqueInt64s(studentIDs),
	})
	return nil
}

// RejectReview 审核驳回。
func (s *service) RejectReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewHandleReq) error {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrReviewNotFound
		}
		return err
	}
	if !sc.IsSuperAdmin() && review.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	if review.Status != enum.GradeReviewStatusPending {
		return errcode.ErrReviewNotPending
	}
	now := time.Now()
	if err := s.reviewRepo.Reject(ctx, id, sc.UserID, stringPtr(req.ReviewComment), now); err != nil {
		return err
	}
	s.publishEvent(ctx, "grade_review_rejected", map[string]interface{}{
		"review_id":  review.ID,
		"teacher_id": review.SubmittedBy,
		"school_id":  review.SchoolID,
		"comment":    req.ReviewComment,
	})
	return nil
}

// UnlockReview 解锁审核记录。
func (s *service) UnlockReview(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UnlockGradeReviewReq) error {
	if sc == nil || !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	review, err := s.reviewRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrReviewNotFound
		}
		return err
	}
	if !sc.IsSuperAdmin() && review.SchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	if !review.IsLocked || review.Status != enum.GradeReviewStatusApproved {
		return errcode.ErrGradeLocked.WithMessage("当前审核记录未处于已锁定状态")
	}
	now := time.Now()
	if err := s.reviewRepo.Unlock(ctx, id, sc.UserID, req.UnlockReason, now); err != nil {
		return err
	}
	s.recordAudit(ctx, sc, "grade_review_unlock", "grade_review", review.ID, req.UnlockReason)
	return nil
}

// IsCourseGradeLocked 查询课程成绩是否被锁定。
func (s *service) IsCourseGradeLocked(ctx context.Context, courseID int64) (bool, error) {
	course, err := s.sourceRepo.GetCourse(ctx, courseID)
	if err != nil {
		return false, err
	}
	if course.SemesterID == nil {
		return false, nil
	}
	review, err := s.reviewRepo.GetByCourseSemester(ctx, courseID, *course.SemesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return review.IsLocked, nil
}

// validateSemesterReq 校验学期请求。
func validateSemesterReq(req *dto.SemesterReq) error {
	if req == nil {
		return errcode.ErrInvalidParams
	}
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return errcode.ErrSemesterDateInvalid
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return errcode.ErrSemesterDateInvalid
	}
	if !startDate.Before(endDate) {
		return errcode.ErrSemesterDateInvalid
	}
	return nil
}

// validateAndBuildLevelConfigs 校验并构建等级映射实体。
func validateAndBuildLevelConfigs(schoolID int64, req *dto.UpdateGradeLevelConfigsReq) ([]*entity.GradeLevelConfig, error) {
	if req == nil || len(req.Levels) < 2 {
		return nil, errcode.ErrInvalidParams
	}
	items := make([]dto.GradeLevelItem, 0, len(req.Levels))
	items = append(items, req.Levels...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].MinScore == items[j].MinScore {
			return items[i].MaxScore > items[j].MaxScore
		}
		return items[i].MinScore < items[j].MinScore
	})
	configs := make([]*entity.GradeLevelConfig, 0, len(items))
	currentMin := 0.0
	for idx, item := range items {
		if item.GPAPoint < 0 || item.GPAPoint > 4 {
			return nil, errcode.ErrLevelConfigGPARange
		}
		if idx == 0 && item.MinScore > 0 {
			return nil, errcode.ErrLevelConfigNotCovered
		}
		if item.MinScore > item.MaxScore {
			return nil, errcode.ErrLevelConfigOverlap
		}
		if idx > 0 && item.MinScore > currentMin+0.02 {
			return nil, errcode.ErrLevelConfigNotCovered
		}
		if idx > 0 && item.MinScore < currentMin-0.001 {
			return nil, errcode.ErrLevelConfigOverlap
		}
		currentMin = item.MaxScore + 0.01
		configs = append(configs, &entity.GradeLevelConfig{
			SchoolID:  schoolID,
			LevelName: item.LevelName,
			MinScore:  item.MinScore,
			MaxScore:  item.MaxScore,
			GPAPoint:  item.GPAPoint,
			SortOrder: idx + 1,
		})
	}
	if items[len(items)-1].MaxScore < 100 {
		return nil, errcode.ErrLevelConfigNotCovered
	}
	return configs, nil
}

// buildSemesterResp 构建学期响应。
func buildSemesterResp(semester *entity.Semester, courseCount *int, summary *dto.ReviewStatusSummaryCounts) *dto.GradeSemesterResp {
	if semester == nil {
		return nil
	}
	resp := &dto.GradeSemesterResp{
		ID:        fmt.Sprintf("%d", semester.ID),
		SchoolID:  stringPtr(fmt.Sprintf("%d", semester.SchoolID)),
		Name:      semester.Name,
		Code:      semester.Code,
		StartDate: formatDate(semester.StartDate),
		EndDate:   formatDate(semester.EndDate),
		IsCurrent: semester.IsCurrent,
		CreatedAt: formatDateTime(&semester.CreatedAt),
	}
	resp.CourseCount = courseCount
	resp.ReviewStatusSummary = summary
	return resp
}

// buildLevelConfigResp 构建等级映射响应。
func buildLevelConfigResp(schoolID int64, configs []*entity.GradeLevelConfig) *dto.GradeLevelConfigResp {
	resp := &dto.GradeLevelConfigResp{
		SchoolID: fmt.Sprintf("%d", schoolID),
		Levels:   make([]dto.GradeLevelItem, 0, len(configs)),
	}
	for _, config := range configs {
		if config == nil {
			continue
		}
		resp.Levels = append(resp.Levels, dto.GradeLevelItem{
			ID:        stringPtr(fmt.Sprintf("%d", config.ID)),
			LevelName: config.LevelName,
			MinScore:  config.MinScore,
			MaxScore:  config.MaxScore,
			GPAPoint:  config.GPAPoint,
			SortOrder: intPtr(config.SortOrder),
		})
	}
	return resp
}

// buildApprovedGradeRows 读取模块03来源数据并生成汇总成绩。
func (s *service) buildApprovedGradeRows(ctx context.Context, schoolID int64, reviewID int64, course *entity.Course) ([]*entity.GradeLevelConfig, []*entity.StudentSemesterGrade, int, error) {
	levelConfigs, err := s.levelRepo.ListBySchool(ctx, schoolID)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(levelConfigs) == 0 {
		levelConfigs = buildDefaultLevelConfigs(schoolID)
	}
	studentIDs, err := s.sourceRepo.ListEnrolledStudentIDs(ctx, course.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	config, err := s.sourceRepo.GetGradeConfig(ctx, course.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	var payload dto.GradeConfigReq
	if err := json.Unmarshal([]byte(config.Config), &payload); err != nil {
		return nil, nil, 0, errcode.ErrInvalidFormat.WithMessage("课程成绩配置解析失败")
	}
	submissions, err := s.sourceRepo.ListLatestGradedSubmissions(ctx, course.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	overrides, err := s.sourceRepo.ListGradeOverrides(ctx, course.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	scoreMap := make(map[int64]map[int64]float64)
	for _, submission := range submissions {
		if submission == nil {
			continue
		}
		if _, ok := scoreMap[submission.StudentID]; !ok {
			scoreMap[submission.StudentID] = make(map[int64]float64)
		}
		scoreMap[submission.StudentID][submission.AssignmentID] = submission.FinalScore
	}
	overrideMap := make(map[int64]*entity.CourseGradeOverride, len(overrides))
	for _, override := range overrides {
		if override == nil {
			continue
		}
		overrideMap[override.StudentID] = override
	}
	rows := make([]*entity.StudentSemesterGrade, 0, len(studentIDs))
	missingCount := 0
	for _, studentID := range studentIDs {
		finalScore, complete := calculateWeightedScore(scoreMap[studentID], payload.Items)
		if !complete {
			missingCount++
			continue
		}
		isAdjusted := false
		if override, ok := overrideMap[studentID]; ok {
			finalScore = override.FinalScore
			isAdjusted = true
		}
		level := matchLevel(levelConfigs, finalScore)
		if level == nil {
			return nil, nil, 0, errcode.ErrLevelConfigNotCovered
		}
		rows = append(rows, &entity.StudentSemesterGrade{
			StudentID:  studentID,
			SchoolID:   schoolID,
			SemesterID: *course.SemesterID,
			CourseID:   course.ID,
			FinalScore: finalScore,
			GradeLevel: level.LevelName,
			GPAPoint:   level.GPAPoint,
			Credits:    *course.Credits,
			IsAdjusted: isAdjusted,
			ReviewID:   reviewID,
		})
	}
	return levelConfigs, rows, missingCount, nil
}

// calculateWeightedScore 计算学生课程加权成绩。
func calculateWeightedScore(scores map[int64]float64, items []dto.GradeConfigItem) (float64, bool) {
	if len(items) == 0 {
		return 0, false
	}
	total := 0.0
	for _, item := range items {
		assignmentID, err := snow.ParseString(item.AssignmentID)
		if err != nil {
			return 0, false
		}
		score, ok := scores[assignmentID]
		if !ok {
			return 0, false
		}
		total += score * item.Weight / 100
	}
	return math.Round(total*100) / 100, true
}

// matchLevel 匹配等级映射。
func matchLevel(configs []*entity.GradeLevelConfig, score float64) *entity.GradeLevelConfig {
	for _, config := range configs {
		if config == nil {
			continue
		}
		if score >= config.MinScore && score <= config.MaxScore {
			return config
		}
	}
	return nil
}

// publishEvent 发送模块07事件。
func (s *service) publishEvent(ctx context.Context, event string, payload map[string]interface{}) {
	if s.eventPublisher == nil {
		return
	}
	_ = s.eventPublisher.Publish(ctx, event, payload)
}

// recordAudit 记录模块01操作日志。
func (s *service) recordAudit(ctx context.Context, sc *svcctx.ServiceContext, action string, targetType string, targetID int64, detail string) {
	if s.auditLogger == nil || sc == nil {
		return
	}
	_ = s.auditLogger.Record(ctx, sc.UserID, action, targetType, targetID, detail, sc.ClientIP)
}

// buildSemesterStats 汇总学期列表展示统计。
func (s *service) buildSemesterStats(ctx context.Context, schoolID, semesterID int64) (*int, *dto.ReviewStatusSummaryCounts, error) {
	courses, err := s.sourceRepo.ListCoursesBySemester(ctx, schoolID, semesterID)
	if err != nil {
		return nil, nil, err
	}
	count := len(courses)
	summary := &dto.ReviewStatusSummaryCounts{}
	for _, course := range courses {
		if course == nil {
			continue
		}
		review, err := s.reviewRepo.GetByCourseSemester(ctx, course.ID, semesterID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				summary.NotSubmitted++
				continue
			}
			return &count, nil, err
		}
		switch review.Status {
		case enum.GradeReviewStatusPending:
			summary.Pending++
		case enum.GradeReviewStatusApproved:
			summary.Approved++
		case enum.GradeReviewStatusRejected:
			summary.Rejected++
		default:
			summary.NotSubmitted++
		}
	}
	return &count, summary, nil
}

// buildReviewLookupMaps 构建审核详情查询所需的课程、学期和用户映射。
func (s *service) buildReviewLookupMaps(ctx context.Context, reviews []*entity.GradeReview) (map[int64]*entity.Course, map[int64]*entity.Semester, map[int64]*UserSummary) {
	courseMap := make(map[int64]*entity.Course)
	semesterMap := make(map[int64]*entity.Semester)
	userMap := make(map[int64]*UserSummary)
	courseIDs := make([]int64, 0, len(reviews))
	semesterIDs := make([]int64, 0, len(reviews))
	userIDs := make([]int64, 0, len(reviews)*2)
	for _, review := range reviews {
		if review == nil {
			continue
		}
		courseIDs = append(courseIDs, review.CourseID)
		semesterIDs = append(semesterIDs, review.SemesterID)
		userIDs = append(userIDs, review.SubmittedBy)
		if review.ReviewedBy != nil {
			userIDs = append(userIDs, *review.ReviewedBy)
		}
	}
	for _, courseID := range uniqueInt64s(courseIDs) {
		course, err := s.sourceRepo.GetCourse(ctx, courseID)
		if err == nil && course != nil {
			courseMap[courseID] = course
		}
	}
	for _, semesterID := range uniqueInt64s(semesterIDs) {
		semester, err := s.semesterRepo.GetByID(ctx, semesterID)
		if err == nil && semester != nil {
			semesterMap[semesterID] = semester
		}
	}
	if s.userQuerier != nil {
		for id, summary := range s.userQuerier.GetUserSummaries(ctx, uniqueInt64s(userIDs)) {
			userMap[id] = summary
		}
	}
	return courseMap, semesterMap, userMap
}

// refreshWarningsForSemester 重新检测学业预警。
func (s *service) refreshWarningsForSemester(ctx context.Context, schoolID, semesterID int64, levelConfigs []*entity.GradeLevelConfig) error {
	config, err := s.warningCfgRepo.GetBySchool(ctx, schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = &entity.WarningConfig{SchoolID: schoolID, GPAThreshold: 2, FailCountThreshold: 2, IsEnabled: true}
		} else {
			return err
		}
	}
	if !config.IsEnabled {
		return nil
	}
	activeWarnings, err := s.warningRepo.ListActiveBySemester(ctx, schoolID, semesterID)
	if err != nil {
		return err
	}
	grades, _, err := s.gradeRepo.List(ctx, &graderepo.StudentGradeListParams{
		SchoolID:   schoolID,
		SemesterID: semesterID,
		Page:       1,
		PageSize:   10000,
	})
	if err != nil {
		return err
	}
	grouped := make(map[int64][]*entity.StudentSemesterGrade)
	for _, grade := range grades {
		if grade == nil {
			continue
		}
		grouped[grade.StudentID] = append(grouped[grade.StudentID], grade)
	}
	triggered := make(map[string]struct{})
	for studentID, rows := range grouped {
		stats, err := s.gradeRepo.CalculateSemesterGPA(ctx, schoolID, studentID, semesterID)
		if err != nil {
			return err
		}
		lowGPA := stats.GPA < config.GPAThreshold
		failRows := make([]dto.AcademicWarningFailedCourse, 0)
		courseScores := make([]dto.AcademicWarningCourseScore, 0, len(rows))
		for _, row := range rows {
			course, _ := s.sourceRepo.GetCourse(ctx, row.CourseID)
			name := courseName(course)
			courseScores = append(courseScores, dto.AcademicWarningCourseScore{
				CourseID:   fmt.Sprintf("%d", row.CourseID),
				CourseName: name,
				Score:      row.FinalScore,
				Grade:      row.GradeLevel,
				Credits:    row.Credits,
			})
			if row.GPAPoint <= 0 {
				failRows = append(failRows, dto.AcademicWarningFailedCourse{
					CourseID:   fmt.Sprintf("%d", row.CourseID),
					CourseName: name,
					Score:      row.FinalScore,
					Semester:   fmt.Sprintf("%d", semesterID),
				})
			}
		}
		if lowGPA {
			triggered[warningKey(studentID, enum.AcademicWarningTypeLowGPA)] = struct{}{}
			if err := s.upsertWarning(ctx, schoolID, studentID, semesterID, enum.AcademicWarningTypeLowGPA, dto.AcademicWarningDetail{
				CurrentGPA:      &stats.GPA,
				Threshold:       config.GPAThreshold,
				SemesterCourses: courseScores,
				FailedCourses:   failRows,
			}); err != nil {
				return err
			}
		}
		if len(failRows) >= config.FailCountThreshold {
			failCount := len(failRows)
			triggered[warningKey(studentID, enum.AcademicWarningTypeConsecutiveFail)] = struct{}{}
			if err := s.upsertWarning(ctx, schoolID, studentID, semesterID, enum.AcademicWarningTypeConsecutiveFail, dto.AcademicWarningDetail{
				FailCount:     &failCount,
				Threshold:     float64(config.FailCountThreshold),
				FailedCourses: failRows,
			}); err != nil {
				return err
			}
		}
	}
	for _, warning := range activeWarnings {
		if warning == nil {
			continue
		}
		if _, ok := triggered[warningKey(warning.StudentID, warning.WarningType)]; ok {
			continue
		}
		if err := s.warningRepo.Resolve(ctx, warning.ID); err != nil {
			return err
		}
	}
	return nil
}

// upsertWarning 创建或更新学业预警。
func (s *service) upsertWarning(ctx context.Context, schoolID, studentID, semesterID int64, warningType int16, detail dto.AcademicWarningDetail) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	existing, err := s.warningRepo.GetExisting(ctx, schoolID, studentID, semesterID, warningType)
	if err == nil && existing != nil {
		return s.warningRepo.UpdateDetail(ctx, existing.ID, datatypes.JSON(raw))
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	warning := &entity.AcademicWarning{
		StudentID:   studentID,
		SchoolID:    schoolID,
		SemesterID:  semesterID,
		WarningType: warningType,
		Detail:      datatypes.JSON(raw),
		Status:      enum.AcademicWarningStatusPending,
	}
	if err := s.warningRepo.Create(ctx, warning); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"warning_id":   warning.ID,
		"student_id":   studentID,
		"warning_type": warningType,
		"threshold":    detail.Threshold,
	}
	if detail.CurrentGPA != nil {
		payload["gpa"] = *detail.CurrentGPA
	}
	s.publishEvent(ctx, "grade_academic_warning", payload)
	return nil
}

// warningKey 构造“学生+预警类型”的唯一键，用于本轮预警重算去重。
func warningKey(studentID int64, warningType int16) string {
	return fmt.Sprintf("%d:%d", studentID, warningType)
}

// courseName 提取课程名称，统一处理空课程兜底。
func courseName(course *entity.Course) string {
	if course == nil {
		return ""
	}
	return course.Title
}

// semesterName 提取学期名称，统一处理空学期兜底。
func semesterName(semester *entity.Semester) string {
	if semester == nil {
		return ""
	}
	return semester.Name
}

// userName 提取用户名称，统一处理空用户兜底。
func userName(user *UserSummary) string {
	if user == nil {
		return ""
	}
	return user.Name
}

// optionalIDString 将可选 ID 转成字符串指针，供 DTO 输出复用。
func optionalIDString(value *int64) *string {
	if value == nil {
		return nil
	}
	return stringPtr(fmt.Sprintf("%d", *value))
}

// optionalUserName 将可选用户摘要转换为名称指针，供 DTO 输出复用。
func optionalUserName(user *UserSummary) *string {
	if user == nil || user.Name == "" {
		return nil
	}
	return &user.Name
}
