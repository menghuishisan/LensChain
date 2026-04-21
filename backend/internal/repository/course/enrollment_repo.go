// enrollment_repo.go
// 模块03 — 课程与教学：选课数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// EnrollmentRepository 选课数据访问接口
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *entity.CourseEnrollment) error
	GetByStudentAndCourse(ctx context.Context, studentID, courseID int64) (*entity.CourseEnrollment, error)
	Remove(ctx context.Context, courseID, studentID int64) error
	List(ctx context.Context, params *EnrollmentListParams) ([]*entity.CourseEnrollment, int64, error)
	ListAllByCourse(ctx context.Context, courseID int64) ([]*entity.CourseEnrollment, error)
	BatchCreate(ctx context.Context, enrollments []*entity.CourseEnrollment) error
	IsEnrolled(ctx context.Context, studentID, courseID int64) (bool, error)
}

// EnrollmentListParams 选课列表查询参数
type EnrollmentListParams struct {
	CourseID int64
	Keyword  string
	Page     int
	PageSize int
}

// ========== Enrollment 实现 ==========

type enrollmentRepository struct {
	db *gorm.DB
}

// NewEnrollmentRepository 创建选课数据访问实例
func NewEnrollmentRepository(db *gorm.DB) EnrollmentRepository {
	return &enrollmentRepository{db: db}
}

// Create 创建选课记录
func (r *enrollmentRepository) Create(ctx context.Context, enrollment *entity.CourseEnrollment) error {
	if enrollment.ID == 0 {
		enrollment.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(enrollment).Error
}

// GetByStudentAndCourse 根据学生和课程获取选课记录
func (r *enrollmentRepository) GetByStudentAndCourse(ctx context.Context, studentID, courseID int64) (*entity.CourseEnrollment, error) {
	var enrollment entity.CourseEnrollment
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ? AND removed_at IS NULL", studentID, courseID).
		First(&enrollment).Error
	if err != nil {
		return nil, err
	}
	return &enrollment, nil
}

// Remove 移除学生（设置 removed_at）
func (r *enrollmentRepository) Remove(ctx context.Context, courseID, studentID int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.CourseEnrollment{}).
		Where("course_id = ? AND student_id = ? AND removed_at IS NULL", courseID, studentID).
		Update("removed_at", gorm.Expr("now()")).Error
}

// List 选课列表查询（含学生信息关联查询）
func (r *enrollmentRepository) List(ctx context.Context, params *EnrollmentListParams) ([]*entity.CourseEnrollment, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("course_id = ? AND removed_at IS NULL", params.CourseID)

	// 关键字搜索需要关联 users 表
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
	query = query.Order("joined_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var enrollments []*entity.CourseEnrollment
	if err := query.Find(&enrollments).Error; err != nil {
		return nil, 0, err
	}
	return enrollments, total, nil
}

// ListAllByCourse 获取课程下所有未移除的选课记录。
// 成绩汇总与成绩导出必须基于完整班级名单，不能复用分页查询截断学生数据。
func (r *enrollmentRepository) ListAllByCourse(ctx context.Context, courseID int64) ([]*entity.CourseEnrollment, error) {
	var enrollments []*entity.CourseEnrollment
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND removed_at IS NULL", courseID).
		Order("joined_at desc").
		Find(&enrollments).Error
	return enrollments, err
}

// BatchCreate 批量创建选课记录
func (r *enrollmentRepository) BatchCreate(ctx context.Context, enrollments []*entity.CourseEnrollment) error {
	if len(enrollments) == 0 {
		return nil
	}
	for i := range enrollments {
		if enrollments[i].ID == 0 {
			enrollments[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(enrollments, courseRepositoryBatchSize).Error
}

// IsEnrolled 检查学生是否已选课
func (r *enrollmentRepository) IsEnrolled(ctx context.Context, studentID, courseID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("student_id = ? AND course_id = ? AND removed_at IS NULL", studentID, courseID).
		Count(&count).Error
	return count > 0, err
}
