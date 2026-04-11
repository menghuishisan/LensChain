// enrollment_repo.go
// 模块03 — 课程与教学：选课与学习进度数据访问层
// 负责选课记录、学习进度的 CRUD 操作
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// EnrollmentRepository 选课数据访问接口
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *entity.CourseEnrollment) error
	GetByStudentAndCourse(ctx context.Context, studentID, courseID int64) (*entity.CourseEnrollment, error)
	Remove(ctx context.Context, courseID, studentID int64) error
	List(ctx context.Context, params *EnrollmentListParams) ([]*entity.CourseEnrollment, int64, error)
	BatchCreate(ctx context.Context, enrollments []*entity.CourseEnrollment) error
	IsEnrolled(ctx context.Context, studentID, courseID int64) (bool, error)
}

// ProgressRepository 学习进度数据访问接口
type ProgressRepository interface {
	Upsert(ctx context.Context, progress *entity.LearningProgress) error
	GetByStudentAndLesson(ctx context.Context, studentID, courseID, lessonID int64) (*entity.LearningProgress, error)
	ListByStudentAndCourse(ctx context.Context, studentID, courseID int64) ([]*entity.LearningProgress, error)
	ListByCourse(ctx context.Context, courseID int64) ([]*entity.LearningProgress, error)
	CountCompletedByStudent(ctx context.Context, studentID, courseID int64) (int, error)
	SumStudyDurationByStudent(ctx context.Context, studentID, courseID int64) (int, error)
	SumStudyDurationByCourse(ctx context.Context, courseID int64) (int, error)
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

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	query = query.Order("joined_at desc").Offset((page - 1) * pageSize).Limit(pageSize)

	var enrollments []*entity.CourseEnrollment
	if err := query.Find(&enrollments).Error; err != nil {
		return nil, 0, err
	}
	return enrollments, total, nil
}

// BatchCreate 批量创建选课记录
func (r *enrollmentRepository) BatchCreate(ctx context.Context, enrollments []*entity.CourseEnrollment) error {
	for i := range enrollments {
		if enrollments[i].ID == 0 {
			enrollments[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(enrollments, 100).Error
}

// IsEnrolled 检查学生是否已选课
func (r *enrollmentRepository) IsEnrolled(ctx context.Context, studentID, courseID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("student_id = ? AND course_id = ? AND removed_at IS NULL", studentID, courseID).
		Count(&count).Error
	return count > 0, err
}

// ========== Progress 实现 ==========

type progressRepository struct {
	db *gorm.DB
}

// NewProgressRepository 创建学习进度数据访问实例
func NewProgressRepository(db *gorm.DB) ProgressRepository {
	return &progressRepository{db: db}
}

// Upsert 更新或创建学习进度
func (r *progressRepository) Upsert(ctx context.Context, progress *entity.LearningProgress) error {
	existing, err := r.GetByStudentAndLesson(ctx, progress.StudentID, progress.CourseID, progress.LessonID)
	if err != nil {
		// 不存在，创建
		if progress.ID == 0 {
			progress.ID = snowflake.Generate()
		}
		return r.db.WithContext(ctx).Create(progress).Error
	}

	// 已存在，更新
	fields := map[string]interface{}{
		"status":           progress.Status,
		"video_progress":   progress.VideoProgress,
		"study_duration":   gorm.Expr("study_duration + ?", progress.StudyDuration),
		"last_accessed_at": progress.LastAccessedAt,
		"updated_at":       gorm.Expr("now()"),
	}
	if progress.CompletedAt != nil {
		fields["completed_at"] = progress.CompletedAt
	}
	return r.db.WithContext(ctx).Model(&entity.LearningProgress{}).
		Where("id = ?", existing.ID).Updates(fields).Error
}

// GetByStudentAndLesson 获取学生某课时的学习进度
func (r *progressRepository) GetByStudentAndLesson(ctx context.Context, studentID, courseID, lessonID int64) (*entity.LearningProgress, error) {
	var progress entity.LearningProgress
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ? AND lesson_id = ?", studentID, courseID, lessonID).
		First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

// ListByStudentAndCourse 获取学生某课程的所有学习进度
func (r *progressRepository) ListByStudentAndCourse(ctx context.Context, studentID, courseID int64) ([]*entity.LearningProgress, error) {
	var progresses []*entity.LearningProgress
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Find(&progresses).Error
	return progresses, err
}

// ListByCourse 获取课程所有学习进度
func (r *progressRepository) ListByCourse(ctx context.Context, courseID int64) ([]*entity.LearningProgress, error) {
	var progresses []*entity.LearningProgress
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Find(&progresses).Error
	return progresses, err
}

// CountCompletedByStudent 统计学生已完成的课时数
func (r *progressRepository) CountCompletedByStudent(ctx context.Context, studentID, courseID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.LearningProgress{}).
		Where("student_id = ? AND course_id = ? AND status = 3", studentID, courseID).
		Count(&count).Error
	return int(count), err
}

// SumStudyDurationByStudent 统计学生某课程的总学习时长（秒）
func (r *progressRepository) SumStudyDurationByStudent(ctx context.Context, studentID, courseID int64) (int, error) {
	var sum *int
	err := r.db.WithContext(ctx).Model(&entity.LearningProgress{}).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Select("COALESCE(SUM(study_duration), 0)").
		Scan(&sum).Error
	if err != nil || sum == nil {
		return 0, err
	}
	return *sum, nil
}

// SumStudyDurationByCourse 统计课程总学习时长（秒）
func (r *progressRepository) SumStudyDurationByCourse(ctx context.Context, courseID int64) (int, error) {
	var sum *int
	err := r.db.WithContext(ctx).Model(&entity.LearningProgress{}).
		Where("course_id = ?", courseID).
		Select(fmt.Sprintf("COALESCE(SUM(study_duration), 0)")).
		Scan(&sum).Error
	if err != nil || sum == nil {
		return 0, err
	}
	return *sum, nil
}
