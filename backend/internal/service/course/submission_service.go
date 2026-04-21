// submission_service.go
// 模块03 — 课程与教学：作业提交与批改业务逻辑
// 从 assignment_service.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"encoding/json"
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

// SaveDraft 保存学生当前作答草稿。
// 草稿与正式提交分离建模，不计入提交次数，也不参与批改与成绩统计。
func (s *assignmentService) SaveDraft(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SaveAssignmentDraftReq) (*dto.SaveAssignmentDraftResp, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}
	course, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID)
	if err != nil {
		return nil, err
	}
	if err := ensureCourseSubmissionAllowed(course); err != nil {
		return nil, err
	}
	if !assignment.IsPublished {
		return nil, errcode.ErrInvalidParams.WithMessage("作业未发布")
	}
	if err := ensureDraftWritable(assignment, time.Now()); err != nil {
		return nil, err
	}
	if err := validateAssignmentAnswers(ctx, req.Answers, assignmentID, s.questionRepo); err != nil {
		return nil, err
	}

	answerBytes, err := json.Marshal(req.Answers)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	now := time.Now()
	draft := &entity.AssignmentDraft{
		AssignmentID: assignmentID,
		StudentID:    sc.UserID,
		Answers:      answerBytes,
		SavedAt:      now,
		UpdatedAt:    now,
	}
	if err := s.draftRepo.Upsert(ctx, draft); err != nil {
		return nil, err
	}
	return &dto.SaveAssignmentDraftResp{
		AssignmentID: strconv.FormatInt(assignmentID, 10),
		SavedAt:      now.UTC().Format(time.RFC3339),
		AnswerCount:  len(req.Answers),
	}, nil
}

// ensureDraftWritable 校验当前作业是否仍允许写入新的服务端草稿。
// 与正式提交不同，草稿仅在文档允许的时间窗口内可写入。
func ensureDraftWritable(assignment *entity.Assignment, now time.Time) error {
	if !assignment.DeadlineAt.IsZero() && now.After(assignment.DeadlineAt) && assignment.LatePolicy == enum.LatePolicyNotAllowed {
		return errcode.ErrAssignmentDeadline.WithMessage("作业已截止且不允许迟交")
	}
	return nil
}

// GetDraft 获取学生当前作答草稿。
// 无草稿时返回 nil，由上层统一输出 data = null。
func (s *assignmentService) GetDraft(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64) (*dto.AssignmentDraftResp, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}
	if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID); err != nil {
		return nil, err
	}
	if !assignment.IsPublished {
		return nil, errcode.ErrAssignmentNotFound
	}

	draft, err := s.draftRepo.GetByStudentAndAssignment(ctx, sc.UserID, assignmentID)
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, nil
	}

	answers := make([]dto.SubmitAnswerReq, 0)
	if len(draft.Answers) > 0 {
		if err := json.Unmarshal(draft.Answers, &answers); err != nil {
			return nil, errcode.ErrInternal
		}
	}
	return &dto.AssignmentDraftResp{
		AssignmentID: strconv.FormatInt(assignmentID, 10),
		SavedAt:      draft.SavedAt.UTC().Format(time.RFC3339),
		Answers:      answers,
	}, nil
}

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
	if err := ensureCourseSubmissionAllowed(course); err != nil {
		return nil, err
	}
	if !assignment.IsPublished {
		return nil, errcode.ErrInvalidParams.WithMessage("作业未发布")
	}

	// 检查提交次数
	count, err := s.submissionRepo.CountByStudentAndAssignment(ctx, sc.UserID, assignmentID)
	if err != nil {
		return nil, err
	}
	if count >= assignment.MaxSubmissions {
		return nil, errcode.ErrSubmissionExceedMax.WithMessage("已达最大提交次数")
	}

	// 检查截止时间和迟交策略
	now := time.Now()
	isLate := false
	lateDays := 0
	if !assignment.DeadlineAt.IsZero() && now.After(assignment.DeadlineAt) {
		if assignment.LatePolicy == enum.LatePolicyNotAllowed {
			return nil, errcode.ErrAssignmentDeadline.WithMessage("作业已截止且不允许迟交")
		}
		isLate = true
		lateDays = int(math.Ceil(now.Sub(assignment.DeadlineAt).Hours() / 24))
	}

	// 获取题目列表用于自动批改
	questions, err := s.questionRepo.ListByAssignmentID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	questionMap := make(map[int64]*entity.AssignmentQuestion)
	for _, q := range questions {
		questionMap[q.ID] = q
	}
	parsedAnswers, err := normalizeAssignmentAnswers(req.Answers, questionMap)
	if err != nil {
		return nil, err
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
	answers := make([]*entity.SubmissionAnswer, 0, len(parsedAnswers))
	var totalScore float64
	var totalObjectiveScore float64
	allObjective := true
	feedbackDetails := make([]dto.SubmitFeedbackDetail, 0, len(parsedAnswers))

	for _, a := range parsedAnswers {
		qID := a.QuestionID
		answer := &entity.SubmissionAnswer{
			SubmissionID: submission.ID, QuestionID: qID,
			AnswerContent: a.Answer.AnswerContent, AnswerFileURL: a.Answer.AnswerFileURL,
		}

		// 客观题自动批改
		if q, ok := questionMap[qID]; ok && enum.IsObjectiveQuestion(q.QuestionType) {
			totalObjectiveScore += q.Score
			correct := a.Answer.AnswerContent != nil && q.CorrectAnswer != nil && *a.Answer.AnswerContent == *q.CorrectAnswer
			answer.IsCorrect = &correct
			if correct {
				answer.Score = &q.Score
				totalScore += q.Score
			} else {
				zero := 0.0
				answer.Score = &zero
			}
			feedbackDetails = append(feedbackDetails, dto.SubmitFeedbackDetail{
				QuestionID: a.Answer.QuestionID,
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
				QuestionID: a.Answer.QuestionID,
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
		if err := s.autoGradeSubmission(ctx, submission.ID, totalScore, isLate, lateDays, assignment); err != nil {
			return nil, err
		}
	}
	if err := s.draftRepo.DeleteByStudentAndAssignment(ctx, sc.UserID, assignmentID); err != nil {
		return nil, err
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

type parsedAssignmentAnswer struct {
	QuestionID int64
	Answer     dto.SubmitAnswerReq
}

// validateAssignmentAnswers 校验草稿或提交中的答题项，确保题目ID合法且属于当前作业。
func validateAssignmentAnswers(ctx context.Context, answers []dto.SubmitAnswerReq, assignmentID int64, questionRepo courserepo.QuestionRepository) error {
	questions, err := questionRepo.ListByAssignmentID(ctx, assignmentID)
	if err != nil {
		return err
	}
	questionMap := make(map[int64]*entity.AssignmentQuestion, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}
	_, err = normalizeAssignmentAnswers(answers, questionMap)
	return err
}

// normalizeAssignmentAnswers 统一解析并校验题目ID，避免提交和草稿写入出现非法题目或重复题目。
func normalizeAssignmentAnswers(answers []dto.SubmitAnswerReq, questionMap map[int64]*entity.AssignmentQuestion) ([]parsedAssignmentAnswer, error) {
	normalized := make([]parsedAssignmentAnswer, 0, len(answers))
	seen := make(map[int64]struct{}, len(answers))
	for _, answer := range answers {
		questionID, err := snowflake.ParseString(answer.QuestionID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("题目ID格式错误")
		}
		if _, ok := questionMap[questionID]; !ok {
			return nil, errcode.ErrInvalidParams.WithMessage("题目不存在或不属于当前作业")
		}
		if _, ok := seen[questionID]; ok {
			return nil, errcode.ErrInvalidParams.WithMessage("题目不可重复作答")
		}
		seen[questionID] = struct{}{}
		normalized = append(normalized, parsedAssignmentAnswer{
			QuestionID: questionID,
			Answer:     answer,
		})
	}
	return normalized, nil
}

// autoGradeSubmission 全客观题自动批改（含迟交扣分）
func (s *assignmentService) autoGradeSubmission(ctx context.Context, submissionID int64, totalScore float64, isLate bool, lateDays int, assignment *entity.Assignment) error {
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
	return s.submissionRepo.UpdateFields(ctx, submissionID, fields)
}

// GetSubmission 获取提交详情
func (s *assignmentService) GetSubmission(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmissionDetailResp, error) {
	submission, err := s.submissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrSubmissionNotFound
	}
	answers, err := s.answerRepo.ListBySubmissionID(ctx, submission.ID)
	if err != nil {
		return nil, err
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
		if !assignment.IsPublished {
			return nil, errcode.ErrAssignmentNotFound
		}
	}

	studentName := s.userNameQuerier.GetUserName(ctx, submission.StudentID)
	questions, err := s.questionRepo.ListByAssignmentID(ctx, submission.AssignmentID)
	if err != nil {
		return nil, err
	}
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

	resp.Answers = make([]dto.SubmissionAnswerItem, 0, len(answers))
	for _, a := range answers {
		item := dto.SubmissionAnswerItem{
			ID:            strconv.FormatInt(a.ID, 10),
			QuestionID:    strconv.FormatInt(a.QuestionID, 10),
			AnswerContent: a.AnswerContent, AnswerFileURL: a.AnswerFileURL,
			IsCorrect: a.IsCorrect, Score: a.Score,
			TeacherComment: a.TeacherComment, AutoJudgeResult: stringifyOptionalJSON(a.AutoJudgeResult),
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
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, assignment.CourseID); err != nil {
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
			SubmissionNo: sub.SubmissionNo,
			Status:       sub.Status, StatusText: enum.GetSubmissionStatusText(sub.Status),
			TotalScore: sub.TotalScore, IsLate: sub.IsLate,
			SubmittedAt: sub.SubmittedAt.Format(time.RFC3339),
		})
		if summary != nil {
			items[len(items)-1].StudentNo = summary.StudentNo
		}
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
	if !assignment.IsPublished {
		return nil, errcode.ErrAssignmentNotFound
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
	submission, err := s.submissionRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSubmissionNotFound
	}
	answers, err := s.answerRepo.ListBySubmissionID(ctx, submission.ID)
	if err != nil {
		return err
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, submission.AssignmentID)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, assignment.CourseID)
	if err != nil {
		return err
	}
	if err := ensureCourseGradingAllowed(course); err != nil {
		return err
	}

	// 更新每道题的分数
	answerMap := make(map[int64]*entity.SubmissionAnswer)
	for _, answer := range answers {
		answerMap[answer.QuestionID] = answer
	}
	questionMap := make(map[int64]*entity.AssignmentQuestion)
	questions, err := s.questionRepo.ListByAssignmentID(ctx, submission.AssignmentID)
	if err != nil {
		return err
	}
	for _, q := range questions {
		questionMap[q.ID] = q
	}

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
			if err := s.answerRepo.UpdateFields(ctx, a.ID, fields); err != nil {
				return err
			}
			a.Score = &ga.Score
			if ga.TeacherComment != nil {
				a.TeacherComment = ga.TeacherComment
			}
		}
	}

	// 批改后总分需要基于本次提交的全部答案重新汇总，避免覆盖已自动批改的客观题得分。
	var totalScore float64
	for _, answer := range answers {
		if answer.Score != nil {
			totalScore += *answer.Score
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
