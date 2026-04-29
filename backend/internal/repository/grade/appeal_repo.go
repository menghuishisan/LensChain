// appeal_repo.go
// 模块06 — 评测与成绩：成绩申诉数据访问层。
// 负责 grade_appeals 表的创建、查询、列表和处理结果保存。

package graderepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// GradeAppealRepository 成绩申诉数据访问接口。
type GradeAppealRepository interface {
	Create(ctx context.Context, appeal *entity.GradeAppeal) error
	GetByID(ctx context.Context, id int64) (*entity.GradeAppeal, error)
	GetByStudentCourseSemester(ctx context.Context, studentID, courseID, semesterID int64) (*entity.GradeAppeal, error)
	List(ctx context.Context, params *GradeAppealListParams) ([]*entity.GradeAppeal, int64, error)
	Approve(ctx context.Context, id, handledBy int64, newScore float64, comment *string, handledAt time.Time) error
	Reject(ctx context.Context, id, handledBy int64, comment *string, handledAt time.Time) error
	CountBySemester(ctx context.Context, schoolID, semesterID int64) (int64, error)
}

// GradeAppealListParams 成绩申诉列表查询参数。
type GradeAppealListParams struct {
	SchoolID   int64
	StudentID  int64
	SemesterID int64
	CourseID   int64
	TeacherID  int64
	Status     int16
	From       *time.Time
	To         *time.Time
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

type gradeAppealRepository struct {
	db *gorm.DB
}

// NewGradeAppealRepository 创建成绩申诉数据访问实例。
func NewGradeAppealRepository(db *gorm.DB) GradeAppealRepository {
	return &gradeAppealRepository{db: db}
}

// Create 创建成绩申诉。
func (r *gradeAppealRepository) Create(ctx context.Context, appeal *entity.GradeAppeal) error {
	if appeal.ID == 0 {
		appeal.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(appeal).Error
}

// GetByID 根据 ID 获取成绩申诉。
func (r *gradeAppealRepository) GetByID(ctx context.Context, id int64) (*entity.GradeAppeal, error) {
	var appeal entity.GradeAppeal
	err := r.db.WithContext(ctx).First(&appeal, id).Error
	if err != nil {
		return nil, err
	}
	return &appeal, nil
}

// GetByStudentCourseSemester 获取学生某课程某学期申诉记录。
func (r *gradeAppealRepository) GetByStudentCourseSemester(ctx context.Context, studentID, courseID, semesterID int64) (*entity.GradeAppeal, error) {
	var appeal entity.GradeAppeal
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ? AND semester_id = ?", studentID, courseID, semesterID).
		First(&appeal).Error
	if err != nil {
		return nil, err
	}
	return &appeal, nil
}

// List 查询成绩申诉列表。
func (r *gradeAppealRepository) List(ctx context.Context, params *GradeAppealListParams) ([]*entity.GradeAppeal, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.GradeAppeal{})
	if params.SchoolID > 0 {
		query = query.Where("grade_appeals.school_id = ?", params.SchoolID)
	}
	if params.Status > 0 {
		query = query.Where("grade_appeals.status = ?", params.Status)
	}
	if params.StudentID > 0 {
		query = query.Where("grade_appeals.student_id = ?", params.StudentID)
	}
	if params.SemesterID > 0 {
		query = query.Where("grade_appeals.semester_id = ?", params.SemesterID)
	}
	if params.CourseID > 0 {
		query = query.Where("grade_appeals.course_id = ?", params.CourseID)
	}
	if params.TeacherID > 0 {
		query = query.Joins("JOIN courses c ON c.id = grade_appeals.course_id").
			Where("c.teacher_id = ?", params.TeacherID)
	}
	if params.From != nil {
		query = query.Where("grade_appeals.created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("grade_appeals.created_at <= ?", *params.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, map[string]string{
		"created_at": "grade_appeals.created_at",
		"handled_at": "grade_appeals.handled_at",
	})

	var appeals []*entity.GradeAppeal
	if err := query.Find(&appeals).Error; err != nil {
		return nil, 0, err
	}
	return appeals, total, nil
}

// Approve 保存申诉同意处理结果。
func (r *gradeAppealRepository) Approve(ctx context.Context, id, handledBy int64, newScore float64, comment *string, handledAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeAppeal{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         enum.GradeAppealStatusApproved,
			"handled_by":     handledBy,
			"handled_at":     handledAt,
			"new_score":      newScore,
			"handle_comment": comment,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}

// Reject 保存申诉驳回处理结果。
func (r *gradeAppealRepository) Reject(ctx context.Context, id, handledBy int64, comment *string, handledAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeAppeal{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         enum.GradeAppealStatusRejected,
			"handled_by":     handledBy,
			"handled_at":     handledAt,
			"handle_comment": comment,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}

// CountBySemester 统计学校学期内成绩申诉数量。
func (r *gradeAppealRepository) CountBySemester(ctx context.Context, schoolID, semesterID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GradeAppeal{}).
		Scopes(database.WithSchoolID(schoolID)).
		Where("semester_id = ?", semesterID).
		Count(&count).Error
	return count, err
}
