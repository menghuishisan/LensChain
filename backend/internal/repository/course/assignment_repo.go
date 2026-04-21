// assignment_repo.go
// 模块03 — 课程与教学：作业与提交数据访问层
// 负责作业、题目、提交记录、答案明细的 CRUD 操作
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// AssignmentRepository 作业数据访问接口
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *entity.Assignment) error
	GetByID(ctx context.Context, id int64) (*entity.Assignment, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	ListByCourseID(ctx context.Context, params *AssignmentListParams) ([]*entity.Assignment, int64, error)
	CountByCourseID(ctx context.Context, courseID int64) (int, error)
	HasSubmissions(ctx context.Context, assignmentID int64) (bool, error)
}

// QuestionRepository 题目数据访问接口
type QuestionRepository interface {
	Create(ctx context.Context, question *entity.AssignmentQuestion) error
	GetByID(ctx context.Context, id int64) (*entity.AssignmentQuestion, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByAssignmentID(ctx context.Context, assignmentID int64) ([]*entity.AssignmentQuestion, error)
	ListByAssignmentIDs(ctx context.Context, assignmentIDs []int64) ([]*entity.AssignmentQuestion, error)
}

// SubmissionRepository 提交数据访问接口
type SubmissionRepository interface {
	Create(ctx context.Context, submission *entity.AssignmentSubmission) error
	GetByID(ctx context.Context, id int64) (*entity.AssignmentSubmission, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	CountByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (int, error)
	ListByAssignment(ctx context.Context, params *SubmissionListParams) ([]*entity.AssignmentSubmission, int64, error)
	ListByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) ([]*entity.AssignmentSubmission, error)
	CountByAssignment(ctx context.Context, assignmentID int64) (int, error)
	GetLatestByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (*entity.AssignmentSubmission, error)
	ListLatestByAssignments(ctx context.Context, assignmentIDs []int64) ([]*entity.AssignmentSubmission, error)
}

// DraftRepository 作答草稿数据访问接口
type DraftRepository interface {
	Upsert(ctx context.Context, draft *entity.AssignmentDraft) error
	GetByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (*entity.AssignmentDraft, error)
	DeleteByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) error
}

// AnswerRepository 答案数据访问接口
type AnswerRepository interface {
	BatchCreate(ctx context.Context, answers []*entity.SubmissionAnswer) error
	ListBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionAnswer, error)
	ListBySubmissionIDs(ctx context.Context, submissionIDs []int64) ([]*entity.SubmissionAnswer, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
}

// AssignmentListParams 作业列表查询参数
type AssignmentListParams struct {
	CourseID       int64
	AssignmentType int16
	OnlyPublished  bool
	Page           int
	PageSize       int
}

// SubmissionListParams 提交列表查询参数
type SubmissionListParams struct {
	AssignmentID int64
	Status       int16
	Keyword      string
	Page         int
	PageSize     int
}

// ========== Assignment 实现 ==========

type assignmentRepository struct {
	db *gorm.DB
}

// NewAssignmentRepository 创建作业数据访问实例
func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db: db}
}

// Create 创建作业
func (r *assignmentRepository) Create(ctx context.Context, assignment *entity.Assignment) error {
	if assignment.ID == 0 {
		assignment.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(assignment).Error
}

// GetByID 根据ID获取作业
func (r *assignmentRepository) GetByID(ctx context.Context, id int64) (*entity.Assignment, error) {
	var assignment entity.Assignment
	err := r.db.WithContext(ctx).First(&assignment, id).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// UpdateFields 更新作业指定字段
func (r *assignmentRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Assignment{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除作业
func (r *assignmentRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Assignment{}, id).Error
}

// ListByCourseID 课程作业列表
func (r *assignmentRepository) ListByCourseID(ctx context.Context, params *AssignmentListParams) ([]*entity.Assignment, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Assignment{}).
		Where("course_id = ?", params.CourseID)

	if params.AssignmentType > 0 {
		query = query.Where("assignment_type = ?", params.AssignmentType)
	}
	if params.OnlyPublished {
		query = query.Where("is_published = ?", true)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("sort_order asc, created_at desc").
		Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var assignments []*entity.Assignment
	if err := query.Find(&assignments).Error; err != nil {
		return nil, 0, err
	}
	return assignments, total, nil
}

// CountByCourseID 统计课程作业数
func (r *assignmentRepository) CountByCourseID(ctx context.Context, courseID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Assignment{}).
		Where("course_id = ?", courseID).Count(&count).Error
	return int(count), err
}

// HasSubmissions 检查作业是否有提交记录
func (r *assignmentRepository) HasSubmissions(ctx context.Context, assignmentID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).
		Where("assignment_id = ?", assignmentID).Count(&count).Error
	return count > 0, err
}

// ========== Question 实现 ==========

type questionRepository struct {
	db *gorm.DB
}

// NewQuestionRepository 创建题目数据访问实例
func NewQuestionRepository(db *gorm.DB) QuestionRepository {
	return &questionRepository{db: db}
}

// Create 创建题目
func (r *questionRepository) Create(ctx context.Context, question *entity.AssignmentQuestion) error {
	if question.ID == 0 {
		question.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(question).Error
}

// GetByID 根据ID获取题目
func (r *questionRepository) GetByID(ctx context.Context, id int64) (*entity.AssignmentQuestion, error) {
	var question entity.AssignmentQuestion
	err := r.db.WithContext(ctx).First(&question, id).Error
	if err != nil {
		return nil, err
	}
	return &question, nil
}

// UpdateFields 更新题目指定字段
func (r *questionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AssignmentQuestion{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 硬删除题目
func (r *questionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.AssignmentQuestion{}, id).Error
}

// ListByAssignmentID 获取作业下所有题目
func (r *questionRepository) ListByAssignmentID(ctx context.Context, assignmentID int64) ([]*entity.AssignmentQuestion, error) {
	var questions []*entity.AssignmentQuestion
	err := r.db.WithContext(ctx).
		Where("assignment_id = ?", assignmentID).
		Order("sort_order asc, created_at asc").
		Find(&questions).Error
	return questions, err
}

// ListByAssignmentIDs 批量获取多个作业下的题目
func (r *questionRepository) ListByAssignmentIDs(ctx context.Context, assignmentIDs []int64) ([]*entity.AssignmentQuestion, error) {
	if len(assignmentIDs) == 0 {
		return []*entity.AssignmentQuestion{}, nil
	}
	var questions []*entity.AssignmentQuestion
	err := r.db.WithContext(ctx).
		Where("assignment_id IN ?", assignmentIDs).
		Order("assignment_id asc, sort_order asc, created_at asc").
		Find(&questions).Error
	return questions, err
}

// ========== Submission 实现 ==========

type submissionRepository struct {
	db *gorm.DB
}

// NewSubmissionRepository 创建提交数据访问实例
func NewSubmissionRepository(db *gorm.DB) SubmissionRepository {
	return &submissionRepository{db: db}
}

// Create 创建提交记录
func (r *submissionRepository) Create(ctx context.Context, submission *entity.AssignmentSubmission) error {
	if submission.ID == 0 {
		submission.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(submission).Error
}

// GetByID 根据ID获取提交记录
func (r *submissionRepository) GetByID(ctx context.Context, id int64) (*entity.AssignmentSubmission, error) {
	var submission entity.AssignmentSubmission
	err := r.db.WithContext(ctx).First(&submission, id).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// UpdateFields 更新提交记录指定字段
func (r *submissionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).Where("id = ?", id).Updates(fields).Error
}

// CountByStudentAndAssignment 统计学生某作业的提交次数
func (r *submissionRepository) CountByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).
		Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).
		Count(&count).Error
	return int(count), err
}

// ListByAssignment 作业提交列表
func (r *submissionRepository) ListByAssignment(ctx context.Context, params *SubmissionListParams) ([]*entity.AssignmentSubmission, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).
		Where("assignment_id = ?", params.AssignmentID)

	if params.Status > 0 {
		query = query.Where("status = ?", params.Status)
	}

	if params.Keyword != "" {
		query = query.Where("student_id IN (?)",
			r.db.Model(&entity.User{}).
				Select("id").
				Scopes(database.WithKeywordSearch(params.Keyword, "name", "student_no")),
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("submitted_at desc").
		Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var submissions []*entity.AssignmentSubmission
	if err := query.Find(&submissions).Error; err != nil {
		return nil, 0, err
	}
	return submissions, total, nil
}

// ListByStudentAndAssignment 获取学生某作业的所有提交
func (r *submissionRepository) ListByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) ([]*entity.AssignmentSubmission, error) {
	var submissions []*entity.AssignmentSubmission
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).
		Order("submission_no desc").
		Find(&submissions).Error
	return submissions, err
}

// CountByAssignment 统计作业提交人数（去重学生）
func (r *submissionRepository) CountByAssignment(ctx context.Context, assignmentID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).
		Where("assignment_id = ?", assignmentID).
		Distinct("student_id").
		Count(&count).Error
	return int(count), err
}

// GetLatestByStudentAndAssignment 获取学生某作业的最新提交
func (r *submissionRepository) GetLatestByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (*entity.AssignmentSubmission, error) {
	var submission entity.AssignmentSubmission
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).
		Order("submission_no desc").
		First(&submission).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// ListLatestByAssignments 批量获取多个作业下每个学生的最新提交
// PostgreSQL DISTINCT ON 与 submission_no 倒序配合，保证成绩汇总使用每个学生最后一次提交。
func (r *submissionRepository) ListLatestByAssignments(ctx context.Context, assignmentIDs []int64) ([]*entity.AssignmentSubmission, error) {
	if len(assignmentIDs) == 0 {
		return []*entity.AssignmentSubmission{}, nil
	}
	var submissions []*entity.AssignmentSubmission
	err := r.db.WithContext(ctx).Model(&entity.AssignmentSubmission{}).
		Select("DISTINCT ON (assignment_id, student_id) *").
		Where("assignment_id IN ?", assignmentIDs).
		Order("assignment_id asc, student_id asc, submission_no desc").
		Find(&submissions).Error
	return submissions, err
}

// ========== Draft 实现 ==========

type draftRepository struct {
	db *gorm.DB
}

// NewDraftRepository 创建作答草稿数据访问实例
func NewDraftRepository(db *gorm.DB) DraftRepository {
	return &draftRepository{db: db}
}

// Upsert 创建或覆盖当前学生在作业下的最新草稿
func (r *draftRepository) Upsert(ctx context.Context, draft *entity.AssignmentDraft) error {
	if draft.ID == 0 {
		draft.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "assignment_id"},
				{Name: "student_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"answers":    draft.Answers,
				"saved_at":   draft.SavedAt,
				"updated_at": draft.UpdatedAt,
			}),
		}).
		Create(draft).Error
}

// GetByStudentAndAssignment 获取学生在指定作业下的当前草稿
func (r *draftRepository) GetByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) (*entity.AssignmentDraft, error) {
	var draft entity.AssignmentDraft
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).
		First(&draft).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// DeleteByStudentAndAssignment 删除学生在指定作业下的草稿
func (r *draftRepository) DeleteByStudentAndAssignment(ctx context.Context, studentID, assignmentID int64) error {
	return r.db.WithContext(ctx).
		Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).
		Delete(&entity.AssignmentDraft{}).Error
}

// ========== Answer 实现 ==========

type answerRepository struct {
	db *gorm.DB
}

// NewAnswerRepository 创建答案数据访问实例
func NewAnswerRepository(db *gorm.DB) AnswerRepository {
	return &answerRepository{db: db}
}

// BatchCreate 批量创建答案
func (r *answerRepository) BatchCreate(ctx context.Context, answers []*entity.SubmissionAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	for i := range answers {
		if answers[i].ID == 0 {
			answers[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(answers, courseRepositoryBatchSize).Error
}

// ListBySubmissionID 获取提交下所有答案
func (r *answerRepository) ListBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionAnswer, error) {
	var answers []*entity.SubmissionAnswer
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Find(&answers).Error
	return answers, err
}

// ListBySubmissionIDs 批量获取多个提交下的答案
func (r *answerRepository) ListBySubmissionIDs(ctx context.Context, submissionIDs []int64) ([]*entity.SubmissionAnswer, error) {
	if len(submissionIDs) == 0 {
		return []*entity.SubmissionAnswer{}, nil
	}
	var answers []*entity.SubmissionAnswer
	err := r.db.WithContext(ctx).
		Where("submission_id IN ?", submissionIDs).
		Order("submission_id asc, created_at asc").
		Find(&answers).Error
	return answers, err
}

// UpdateFields 更新答案指定字段
func (r *answerRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SubmissionAnswer{}).Where("id = ?", id).Updates(fields).Error
}
