// assignment_service.go
// 模块03 — 课程与教学：作业管理业务逻辑（CRUD + 题目管理）
// 提交与批改逻辑拆分至 submission_service.go
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"gorm.io/datatypes"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/contentsafety"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/timeutil"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// AssignmentService 作业管理服务接口
type AssignmentService interface {
	// 作业
	Create(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateAssignmentReq) (string, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AssignmentDetailResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAssignmentReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	List(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AssignmentListReq) ([]*dto.AssignmentListItem, int64, error)
	Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	// 题目
	AddQuestion(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.AddQuestionReq) (string, error)
	UpdateQuestion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateQuestionReq) error
	DeleteQuestion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	// 提交
	SaveDraft(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SaveAssignmentDraftReq) (*dto.SaveAssignmentDraftResp, error)
	GetDraft(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64) (*dto.AssignmentDraftResp, error)
	Submit(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SubmitAssignmentReq) (*dto.SubmitAssignmentResp, error)
	GetSubmission(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmissionDetailResp, error)
	ListSubmissions(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.SubmissionListReq) ([]*dto.SubmissionListItem, int64, error)
	ListMySubmissions(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64) (*dto.MySubmissionsResp, error)
	GradeSubmission(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.GradeSubmissionReq) error
}

type assignmentService struct {
	courseRepo         courserepo.CourseRepository
	assignmentRepo     courserepo.AssignmentRepository
	questionRepo       courserepo.QuestionRepository
	submissionRepo     courserepo.SubmissionRepository
	draftRepo          courserepo.DraftRepository
	answerRepo         courserepo.AnswerRepository
	enrollmentRepo     courserepo.EnrollmentRepository
	userNameQuerier    UserNameQuerier
	userSummaryQuerier UserSummaryQuerier
}

// NewAssignmentService 创建作业管理服务实例
func NewAssignmentService(
	courseRepo courserepo.CourseRepository,
	assignmentRepo courserepo.AssignmentRepository,
	questionRepo courserepo.QuestionRepository,
	submissionRepo courserepo.SubmissionRepository,
	draftRepo courserepo.DraftRepository,
	answerRepo courserepo.AnswerRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	userNameQuerier UserNameQuerier,
	userSummaryQuerier UserSummaryQuerier,
) AssignmentService {
	return &assignmentService{
		courseRepo: courseRepo, assignmentRepo: assignmentRepo,
		questionRepo: questionRepo, submissionRepo: submissionRepo,
		draftRepo: draftRepo, answerRepo: answerRepo, enrollmentRepo: enrollmentRepo,
		userNameQuerier: userNameQuerier, userSummaryQuerier: userSummaryQuerier,
	}
}

// ========== 作业管理 ==========

func (s *assignmentService) Create(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateAssignmentReq) (string, error) {
	if err := s.verifyCourseTeacherForContent(ctx, sc, courseID); err != nil {
		return "", err
	}

	assignment := &entity.Assignment{
		CourseID:            courseID,
		Title:               req.Title,
		Description:         contentsafety.SanitizeOptionalMarkdown(req.Description),
		AssignmentType:      req.AssignmentType,
		TotalScore:          0,
		LatePolicy:          req.LatePolicy,
		LateDeductionPerDay: req.LateDeductionPerDay,
		MaxSubmissions:      1,
	}
	if req.DeadlineAt == nil {
		return "", errcode.ErrInvalidParams.WithMessage("截止时间不能为空")
	}
	if req.ChapterID != nil {
		chapterID, err := parseOptionalSnowflakeID(req.ChapterID, "章节ID格式错误")
		if err != nil {
			return "", err
		}
		assignment.ChapterID = chapterID
	}
	t, err := timeutil.ParseRFC3339(*req.DeadlineAt)
	if err != nil {
		return "", errcode.ErrInvalidParams.WithMessage("截止时间格式错误")
	}
	assignment.DeadlineAt = *t
	if req.MaxSubmissions != nil {
		assignment.MaxSubmissions = *req.MaxSubmissions
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return "", err
	}
	return strconv.FormatInt(assignment.ID, 10), nil
}

// GetByID 获取作业详情
// 教师可查看自己课程下的全部作业，学生仅可查看自己已加入课程且已发布的作业
func (s *assignmentService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AssignmentDetailResp, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrAssignmentNotFound
	}
	questions, err := s.questionRepo.ListByAssignmentID(ctx, assignment.ID)
	if err != nil {
		return nil, err
	}
	teacherView := true

	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, assignment.CourseID); err != nil {
		if !errors.Is(err, errcode.ErrNotCourseTeacher) {
			return nil, err
		}
		if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, assignment.CourseID); err != nil {
			return nil, err
		}
		if !assignment.IsPublished {
			return nil, errcode.ErrAssignmentNotFound
		}
		teacherView = false
	}

	return s.buildAssignmentDetail(assignment, questions, teacherView), nil
}

// Update 更新作业基础信息
func (s *assignmentService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAssignmentReq) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = contentsafety.SanitizeMarkdown(*req.Description)
	}
	if req.AssignmentType != nil {
		fields["assignment_type"] = *req.AssignmentType
	}
	if req.DeadlineAt != nil {
		t, err := timeutil.ParseRFC3339(*req.DeadlineAt)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("截止时间格式错误")
		}
		fields["deadline_at"] = t
	}
	if req.MaxSubmissions != nil {
		fields["max_submissions"] = *req.MaxSubmissions
	}
	if req.LatePolicy != nil {
		fields["late_policy"] = *req.LatePolicy
	}
	if req.LateDeductionPerDay != nil {
		fields["late_deduction_per_day"] = *req.LateDeductionPerDay
	}
	if req.ChapterID != nil {
		chapterID, err := parseOptionalSnowflakeID(req.ChapterID, "章节ID格式错误")
		if err != nil {
			return err
		}
		if chapterID == nil {
			fields["chapter_id"] = nil
		} else {
			fields["chapter_id"] = *chapterID
		}
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.assignmentRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除作业
// 仅课程教师可删除，且已有提交记录时禁止删除
func (s *assignmentService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return err
	}
	// 已有提交的作业不可删除
	has, err := s.assignmentRepo.HasSubmissions(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return errcode.ErrInvalidParams.WithMessage("该作业已有学生提交，不可删除")
	}
	return s.assignmentRepo.SoftDelete(ctx, id)
}

// List 获取作业列表
// 教师可查看课程全部作业，学生仅可查看已发布作业
func (s *assignmentService) List(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AssignmentListReq) ([]*dto.AssignmentListItem, int64, error) {
	teacherView := true
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		if !errors.Is(err, errcode.ErrNotCourseTeacher) {
			return nil, 0, err
		}
		if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
			return nil, 0, err
		}
		teacherView = false
	}

	assignments, total, err := s.assignmentRepo.ListByCourseID(ctx, &courserepo.AssignmentListParams{
		CourseID: courseID, AssignmentType: req.AssignmentType,
		OnlyPublished: !teacherView,
		Page:          req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	studentCount, err := s.courseRepo.CountStudents(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.AssignmentListItem, 0, len(assignments))
	for _, a := range assignments {
		submitCount, err := s.submissionRepo.CountByAssignment(ctx, a.ID)
		if err != nil {
			return nil, 0, err
		}
		item := &dto.AssignmentListItem{
			ID: strconv.FormatInt(a.ID, 10), Title: a.Title,
			AssignmentType:     a.AssignmentType,
			AssignmentTypeText: enum.GetAssignmentTypeText(a.AssignmentType),
			TotalScore:         a.TotalScore, IsPublished: a.IsPublished,
			SubmitCount: submitCount, TotalStudents: studentCount,
			SortOrder: a.SortOrder,
		}
		if !a.DeadlineAt.IsZero() {
			d := a.DeadlineAt.Format(time.RFC3339)
			item.DeadlineAt = &d
		}
		items = append(items, item)
	}
	return items, total, nil
}

// Publish 发布作业
func (s *assignmentService) Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return err
	}
	if assignment.IsPublished {
		return errcode.ErrInvalidParams.WithMessage("作业已发布")
	}
	questions, err := s.questionRepo.ListByAssignmentID(ctx, id)
	if err != nil {
		return err
	}
	if len(questions) == 0 {
		return errcode.ErrInvalidParams.WithMessage("请至少添加一道题目后再发布")
	}
	return s.assignmentRepo.UpdateFields(ctx, id, map[string]interface{}{
		"is_published": true, "updated_at": time.Now(),
	})
}

// ========== 题目管理 ==========

func (s *assignmentService) AddQuestion(ctx context.Context, sc *svcctx.ServiceContext, assignmentID int64, req *dto.AddQuestionReq) (string, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return "", errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return "", err
	}

	question := &entity.AssignmentQuestion{
		AssignmentID: assignmentID, QuestionType: req.QuestionType,
		Title:         contentsafety.SanitizeMarkdown(req.Title),
		CorrectAnswer: req.CorrectAnswer, ReferenceAnswer: contentsafety.SanitizeOptionalMarkdown(req.ReferenceAnswer),
		Score: req.Score,
	}
	options, err := parseOptionalJSON(req.Options, "题目选项格式错误")
	if err != nil {
		return "", err
	}
	judgeConfig, err := parseOptionalJSON(req.JudgeConfig, "判题配置格式错误")
	if err != nil {
		return "", err
	}
	question.Options = options
	question.JudgeConfig = judgeConfig
	if err := s.questionRepo.Create(ctx, question); err != nil {
		return "", err
	}
	if err := s.syncAssignmentTotalScore(ctx, assignmentID); err != nil {
		return "", err
	}
	return strconv.FormatInt(question.ID, 10), nil
}

// UpdateQuestion 更新作业题目
func (s *assignmentService) UpdateQuestion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateQuestionReq) error {
	question, err := s.questionRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("题目不存在")
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, question.AssignmentID)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.QuestionType != nil {
		fields["question_type"] = *req.QuestionType
	}
	if req.Title != nil {
		fields["title"] = contentsafety.SanitizeMarkdown(*req.Title)
	}
	if req.Options != nil {
		options, err := parseOptionalJSON(req.Options, "题目选项格式错误")
		if err != nil {
			return err
		}
		fields["options"] = options
	}
	if req.CorrectAnswer != nil {
		fields["correct_answer"] = *req.CorrectAnswer
	}
	if req.ReferenceAnswer != nil {
		fields["reference_answer"] = contentsafety.SanitizeMarkdown(*req.ReferenceAnswer)
	}
	if req.Score != nil {
		fields["score"] = *req.Score
	}
	if req.JudgeConfig != nil {
		judgeConfig, err := parseOptionalJSON(req.JudgeConfig, "判题配置格式错误")
		if err != nil {
			return err
		}
		fields["judge_config"] = judgeConfig
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	if err := s.questionRepo.UpdateFields(ctx, id, fields); err != nil {
		return err
	}
	return s.syncAssignmentTotalScore(ctx, question.AssignmentID)
}

// DeleteQuestion 删除作业题目
func (s *assignmentService) DeleteQuestion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	question, err := s.questionRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("题目不存在")
	}
	assignment, err := s.assignmentRepo.GetByID(ctx, question.AssignmentID)
	if err != nil {
		return errcode.ErrAssignmentNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, assignment.CourseID); err != nil {
		return err
	}
	hasSubmissions, err := s.assignmentRepo.HasSubmissions(ctx, assignment.ID)
	if err != nil {
		return err
	}
	if hasSubmissions {
		return errcode.ErrInvalidParams.WithMessage("作业已有学生提交，不能删除题目")
	}
	if err := s.questionRepo.Delete(ctx, id); err != nil {
		return err
	}
	return s.syncAssignmentTotalScore(ctx, assignment.ID)
}

// ========== 辅助方法 ==========

// verifyCourseTeacherForContent 校验课程教师身份，并确保课程仍允许编辑教学内容。
func (s *assignmentService) verifyCourseTeacherForContent(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	return ensureCourseContentEditable(course)
}

// syncAssignmentTotalScore 根据当前题目分值重算并回写作业总分，确保总分口径与题目保持一致。
func (s *assignmentService) syncAssignmentTotalScore(ctx context.Context, assignmentID int64) error {
	questions, err := s.questionRepo.ListByAssignmentID(ctx, assignmentID)
	if err != nil {
		return err
	}

	totalScore := 0.0
	for _, question := range questions {
		totalScore += question.Score
	}

	return s.assignmentRepo.UpdateFields(ctx, assignmentID, map[string]interface{}{
		"total_score": totalScore,
		"updated_at":  time.Now(),
	})
}

// buildAssignmentDetail 构建作业详情响应。
// 学生视角不返回标准答案与参考答案，避免通过详情接口提前获知题解。
func (s *assignmentService) buildAssignmentDetail(
	a *entity.Assignment,
	questions []*entity.AssignmentQuestion,
	includeAnswers bool,
) *dto.AssignmentDetailResp {
	resp := &dto.AssignmentDetailResp{
		ID:       strconv.FormatInt(a.ID, 10),
		CourseID: strconv.FormatInt(a.CourseID, 10),
		Title:    a.Title, Description: a.Description,
		AssignmentType:     a.AssignmentType,
		AssignmentTypeText: enum.GetAssignmentTypeText(a.AssignmentType),
		TotalScore:         a.TotalScore, MaxSubmissions: a.MaxSubmissions,
		LatePolicy: a.LatePolicy, LatePolicyText: enum.GetLatePolicyText(a.LatePolicy),
		LateDeductionPerDay: a.LateDeductionPerDay, IsPublished: a.IsPublished,
	}
	if a.ChapterID != nil {
		cid := strconv.FormatInt(*a.ChapterID, 10)
		resp.ChapterID = &cid
	}
	if !a.DeadlineAt.IsZero() {
		d := a.DeadlineAt.Format(time.RFC3339)
		resp.DeadlineAt = &d
	}

	resp.Questions = make([]dto.QuestionDetailItem, 0, len(questions))
	for _, q := range questions {
		item := dto.QuestionDetailItem{
			ID: strconv.FormatInt(q.ID, 10), QuestionType: q.QuestionType,
			QuestionTypeText: enum.GetQuestionTypeText(q.QuestionType),
			Title:            q.Title, Options: stringifyOptionalJSON(q.Options),
			Score: q.Score, SortOrder: q.SortOrder,
		}
		if includeAnswers {
			item.CorrectAnswer = q.CorrectAnswer
			item.ReferenceAnswer = q.ReferenceAnswer
			item.JudgeConfig = stringifyOptionalJSON(q.JudgeConfig)
		}
		resp.Questions = append(resp.Questions, item)
	}
	return resp
}

// parseOptionalJSON 将 DTO 中的 JSON 字符串转为仓储层持久化所需的 JSONB 数据。
func parseOptionalJSON(raw *string, message string) (datatypes.JSON, error) {
	if raw == nil || *raw == "" {
		return datatypes.JSON(nil), nil
	}
	if !json.Valid([]byte(*raw)) {
		return nil, errcode.ErrInvalidParams.WithMessage(message)
	}
	return datatypes.JSON([]byte(*raw)), nil
}

// stringifyOptionalJSON 将 JSONB 内容转回字符串，保持 DTO 与 API 文档一致。
func stringifyOptionalJSON(raw datatypes.JSON) *string {
	if len(raw) == 0 {
		return nil
	}
	text := string(raw)
	return &text
}
