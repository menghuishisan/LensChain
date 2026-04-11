// submission_service.go
// 模块03 — 课程与教学：作业提交与批改业务逻辑
// 从 assignment_service.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// Submit 学生提交作业（含客观题自动批改）
func (s *assignmentService) Submit(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SubmitAssignmentReq) (*dto.SubmitAssignmentResp, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}
	course, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID)
	if err != nil {
		return nil, err
	}
	if course.Status == enum.CourseStatusEnded || course.Status == enum.CourseStatusArchived {
		return nil, errcode.ErrInvalidParams.WithMessage("课程已结束，无法提交作业")
	}
	if !assignment.IsPublished {
		return nil, errcode.ErrInvalidParams.WithMessage("作业未发布")
	}

	// 检查提交次数
	count, _ := s.submissionRepo.CountByStudentAndAssignment(ctx, sc.UserID, assignmentID)
	if count >= assignment.MaxSubmissions {
		return nil, errcode.ErrSubmissionExceedMax.WithMessage("已达最大提交次数")
	}

	// 检查截止时间和迟交策略
	now := time.Now()
	isLate := false
	lateDays := 0
	if assignment.DeadlineAt != nil && now.After(*assignment.DeadlineAt) {
		if assignment.LatePolicy == enum.LatePolicyNotAllowed {
			return nil, errcode.ErrAssignmentDeadline.WithMessage("作业已截止且不允许迟交")
		}
		isLate = true
		lateDays = int(math.Ceil(now.Sub(*assignment.DeadlineAt).Hours() / 24))
	}

	// 获取题目列表用于自动批改
	questions, _ := s.questionRepo.ListByAssignmentID(ctx, assignmentID)
	questionMap := make(map[int64]*entity.AssignmentQuestion)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// 创建提交记录
	submission := &entity.AssignmentSubmission{
		AssignmentID: assignmentID, StudentID: sc.UserID,
		SubmissionNo: count + 1, Status: enum.SubmissionStatusSubmitted,
		IsLate: isLate, SubmittedAt: now,
	}
	if isLate {
		submission.LateDays = &lateDays
	}

	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, err
	}

	// 创建答案并自动批改客观题
	answers := make([]*entity.SubmissionAnswer, 0, len(req.Answers))
	var totalScore float64
	var totalObjectiveScore float64
	allObjective := true
	feedbackDetails := make([]dto.SubmitFeedbackDetail, 0, len(req.Answers))

	for _, a := range req.Answers {
		qID, err := snowflake.ParseString(a.QuestionID)
		if err != nil {
			continue
		}
		answer := &entity.SubmissionAnswer{
			SubmissionID: submission.ID, QuestionID: qID,
			AnswerContent: a.AnswerContent, AnswerFileURL: a.AnswerFileURL,
		}

		// 客观题自动批改
		if q, ok := questionMap[qID]; ok && enum.IsObjectiveQuestion(q.QuestionType) {
			totalObjectiveScore += q.Score
			correct := a.AnswerContent != nil && q.CorrectAnswer != nil && *a.AnswerContent == *q.CorrectAnswer
			answer.IsCorrect = &correct
			if correct {
				answer.Score = &q.Score
				totalScore += q.Score
			} else {
				zero := 0.0
				answer.Score = &zero
			}
			feedbackDetails = append(feedbackDetails, dto.SubmitFeedbackDetail{
				QuestionID: a.QuestionID,
				IsCorrect:  answer.IsCorrect,
				Score:      answer.Score,
			})
		} else {
			allObjective = false
			status := "pending_review"
			if q, ok := questionMap[qID]; ok && q.QuestionType == enum.QuestionTypeCoding {
				status = "judging"
			}
			feedbackDetails = append(feedbackDetails, dto.SubmitFeedbackDetail{
				QuestionID: a.QuestionID,
				Status:     status,
			})
		}
		answers = append(answers, answer)
	}

	if err := s.answerRepo.BatchCreate(ctx, answers); err != nil {
		return nil, err
	}

	// 如果全部为客观题，直接标记为已批改
	if allObjective {
		s.autoGradeSubmission(ctx, submission.ID, totalScore, isLate, lateDays, assignment)
	}

	return &dto.SubmitAssignmentResp{
		SubmissionID:         strconv.FormatInt(submission.ID, 10),
		SubmissionNo:         submission.SubmissionNo,
		RemainingSubmissions: assignment.MaxSubmissions - submission.SubmissionNo,
		IsLate:               submission.IsLate,
		InstantFeedback: dto.SubmitFeedbackSummary{
			AutoGradedScore: totalScore,
			AutoGradedTotal: totalObjectiveScore,
			Details:         feedbackDetails,
		},
	}, nil
}

// autoGradeSubmission 全客观题自动批改（含迟交扣分）
func (s *assignmentService) autoGradeSubmission(ctx context.Context, submissionID int64, totalScore float64, isLate bool, lateDays int, assignment *entity.Assignment) {
	now := time.Now()
	fields := map[string]interface{}{
		"status": enum.SubmissionStatusGraded, "total_score": totalScore,
		"score_before_deduction": totalScore, "graded_at": now,
	}
	// 迟交扣分
	if isLate && assignment.LatePolicy == enum.LatePolicyWithDeduction && assignment.LateDeductionPerDay != nil {
		deduction := float64(lateDays) * *assignment.LateDeductionPerDay / 100 * totalScore
		afterDeduction := math.Max(0, totalScore-deduction)
		fields["score_after_deduction"] = afterDeduction
		fields["total_score"] = afterDeduction
	} else {
		fields["score_after_deduction"] = totalScore
	}
	_ = s.submissionRepo.UpdateFields(ctx, submissionID, fields)
}

// GetSubmission 获取提交详情
func (s *assignmentService) GetSubmission(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmissionDetailResp, error) {
	submission, err := s.submissionRepo.GetByIDWithAnswers(ctx, id)
	if err != nil {
		return nil, errcode.ErrSubmissionNotFound
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, submission.AssignmentID)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}

	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, assignment.CourseID); err != nil {
		if !errors.Is(err, errcode.ErrNotCourseTeacher) {
			return nil, err
		}
		if submission.StudentID != sc.UserID {
			return nil, errcode.ErrForbidden
		}
		if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID); err != nil {
			return nil, err
		}
	}

	studentName := s.userNameQuerier.GetUserName(ctx, submission.StudentID)
	questions, _ := s.questionRepo.ListByAssignmentID(ctx, submission.AssignmentID)
	questionMap := make(map[int64]*entity.AssignmentQuestion)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	resp := &dto.SubmissionDetailResp{
		ID:           strconv.FormatInt(submission.ID, 10),
		AssignmentID: strconv.FormatInt(submission.AssignmentID, 10),
		StudentID:    strconv.FormatInt(submission.StudentID, 10),
		StudentName:  studentName, SubmissionNo: submission.SubmissionNo,
		Status: submission.Status, StatusText: enum.GetSubmissionStatusText(submission.Status),
		TotalScore: submission.TotalScore, IsLate: submission.IsLate,
		LateDays:             submission.LateDays,
		ScoreBeforeDeduction: submission.ScoreBeforeDeduction,
		ScoreAfterDeduction:  submission.ScoreAfterDeduction,
		TeacherComment:       submission.TeacherComment,
		SubmittedAt:          submission.SubmittedAt.Format(time.RFC3339),
	}
	if submission.GradedAt != nil {
		g := submission.GradedAt.Format(time.RFC3339)
		resp.GradedAt = &g
	}

	resp.Answers = make([]dto.SubmissionAnswerItem, 0, len(submission.Answers))
	for _, a := range submission.Answers {
		item := dto.SubmissionAnswerItem{
			ID:            strconv.FormatInt(a.ID, 10),
			QuestionID:    strconv.FormatInt(a.QuestionID, 10),
			AnswerContent: a.AnswerContent, AnswerFileURL: a.AnswerFileURL,
			IsCorrect: a.IsCorrect, Score: a.Score,
			TeacherComment: a.TeacherComment, AutoJudgeResult: a.AutoJudgeResult,
		}
		if q, ok := questionMap[a.QuestionID]; ok {
			item.QuestionTitle = q.Title
			item.QuestionType = q.QuestionType
		}
		resp.Answers = append(resp.Answers, item)
	}
	return resp, nil
}

// ListSubmissions 教师查看作业提交列表
func (s *assignmentService) ListSubmissions(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SubmissionListReq) ([]*dto.SubmissionListItem, int64, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, 0, errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, assignment.CourseID); err != nil {
		return nil, 0, err
	}

	submissions, total, err := s.submissionRepo.ListByAssignment(ctx, &courserepo.SubmissionListParams{
		AssignmentID: assignmentID, Status: req.Status,
		Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.SubmissionListItem, 0, len(submissions))
	for _, sub := range submissions {
		summary := s.userSummaryQuerier.GetUserSummary(ctx, sub.StudentID)
		name := ""
		if summary != nil {
			name = summary.Name
		}
		items = append(items, &dto.SubmissionListItem{
			ID:           strconv.FormatInt(sub.ID, 10),
			StudentID:    strconv.FormatInt(sub.StudentID, 10),
			StudentName:  name,
			StudentNo:    getSummaryStudentNo(summary),
			SubmissionNo: sub.SubmissionNo,
			Status:       sub.Status, StatusText: enum.GetSubmissionStatusText(sub.Status),
			TotalScore: sub.TotalScore, IsLate: sub.IsLate,
			SubmittedAt: sub.SubmittedAt.Format(time.RFC3339),
		})
	}
	return items, total, nil
}

// ListMySubmissions 学生查看自己的提交列表
func (s *assignmentService) ListMySubmissions(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64) (*dto.MySubmissionsResp, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}
	if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID); err != nil {
		return nil, err
	}

	submissions, err := s.submissionRepo.ListByStudentAndAssignment(ctx, sc.UserID, assignmentID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.MySubmissionItem, 0, len(submissions))
	for _, sub := range submissions {
		items = append(items, dto.MySubmissionItem{
			ID: strconv.FormatInt(sub.ID, 10), SubmissionNo: sub.SubmissionNo,
			Status: sub.Status, StatusText: enum.GetSubmissionStatusText(sub.Status),
			TotalScore: sub.TotalScore, IsLate: sub.IsLate,
			SubmittedAt: sub.SubmittedAt.Format(time.RFC3339),
		})
	}
	return &dto.MySubmissionsResp{Submissions: items}, nil
}

// GradeSubmission 教师批改提交
func (s *assignmentService) GradeSubmission(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.GradeSubmissionReq) error {
	submission, err := s.submissionRepo.GetByIDWithAnswers(ctx, id)
	if err != nil {
		return errcode.ErrSubmissionNotFound
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, submission.AssignmentID)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, assignment.CourseID); err != nil {
		return err
	}

	// 更新每道题的分数
	answerMap := make(map[int64]*entity.SubmissionAnswer)
	for i := range submission.Answers {
		answerMap[submission.Answers[i].QuestionID] = &submission.Answers[i]
	}
	questionMap := make(map[int64]*entity.AssignmentQuestion)
	questions, _ := s.questionRepo.ListByAssignmentID(ctx, submission.AssignmentID)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	var totalScore float64
	for _, ga := range req.Answers {
		qID, err := snowflake.ParseString(ga.QuestionID)
		if err != nil {
			continue
		}
		if a, ok := answerMap[qID]; ok {
			if q, exists := questionMap[qID]; exists && ga.Score > q.Score {
				return errcode.ErrInvalidParams.WithMessage("给分不能超过题目分值")
			}
			fields := map[string]interface{}{"score": ga.Score}
			if ga.TeacherComment != nil {
				fields["teacher_comment"] = *ga.TeacherComment
			}
			_ = s.answerRepo.UpdateFields(ctx, a.ID, fields)
			totalScore += ga.Score
		}
	}

	// 更新提交记录
	now := time.Now()
	fields := map[string]interface{}{
		"status": enum.SubmissionStatusGraded, "total_score": totalScore,
		"score_before_deduction": totalScore,
		"graded_by":              sc.UserID, "graded_at": now,
	}
	if req.TeacherComment != nil {
		fields["teacher_comment"] = *req.TeacherComment
	}

	// 迟交扣分
	if submission.IsLate && assignment.LatePolicy == enum.LatePolicyWithDeduction && assignment.LateDeductionPerDay != nil {
		days := 0
		if submission.LateDays != nil {
			days = *submission.LateDays
		}
		deduction := float64(days) * *assignment.LateDeductionPerDay / 100 * totalScore
		afterDeduction := math.Max(0, totalScore-deduction)
		fields["score_after_deduction"] = afterDeduction
		fields["total_score"] = afterDeduction
	} else {
		fields["score_after_deduction"] = totalScore
	}

	return s.submissionRepo.UpdateFields(ctx, id, fields)
}
