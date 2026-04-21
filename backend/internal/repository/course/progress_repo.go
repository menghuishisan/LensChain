// progress_repo.go
// 模块03 — 课程与教学：学习进度数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ProgressRepository 学习进度数据访问接口
type ProgressRepository interface {
	Upsert(ctx context.Context, progress *entity.LearningProgress) error
	GetByStudentAndLesson(ctx context.Context, studentID, courseID, lessonID int64) (*entity.LearningProgress, error)
	ListByStudentAndCourse(ctx context.Context, studentID, courseID int64) ([]*entity.LearningProgress, error)
	ListByCourse(ctx context.Context, courseID int64) ([]*entity.LearningProgress, error)
	CountCompletedByStudent(ctx context.Context, studentID, courseID int64) (int, error)
	SumStudyDurationByStudent(ctx context.Context, studentID, courseID int64) (int, error)
	SumStudyDurationByCourse(ctx context.Context, courseID int64) (int, error)
	DeleteByLessonID(ctx context.Context, lessonID int64) error
}

type progressRepository struct {
	db *gorm.DB
}

// NewProgressRepository 创建学习进度数据访问实例
func NewProgressRepository(db *gorm.DB) ProgressRepository {
	return &progressRepository{db: db}
}

// Upsert 更新或创建学习进度
func (r *progressRepository) Upsert(ctx context.Context, progress *entity.LearningProgress) error {
	if progress.ID == 0 {
		progress.ID = snowflake.Generate()
	}

	updateFields := map[string]interface{}{
		"status":           progress.Status,
		"video_progress":   progress.VideoProgress,
		"study_duration":   gorm.Expr("learning_progresses.study_duration + ?", progress.StudyDuration),
		"last_accessed_at": progress.LastAccessedAt,
		"updated_at":       gorm.Expr("now()"),
	}
	if progress.CompletedAt != nil {
		updateFields["completed_at"] = progress.CompletedAt
	}

	// 课程、学生、课时存在唯一索引，使用数据库原子 upsert 避免并发上报时重复插入。
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "course_id"},
			{Name: "student_id"},
			{Name: "lesson_id"},
		},
		DoUpdates: clause.Assignments(updateFields),
	}).Create(progress).Error
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
		Where("student_id = ? AND course_id = ? AND status = ?", studentID, courseID, enum.LearningStatusCompleted).
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
		Select("COALESCE(SUM(study_duration), 0)").
		Scan(&sum).Error
	if err != nil || sum == nil {
		return 0, err
	}
	return *sum, nil
}

// DeleteByLessonID 删除指定课时关联的全部学习进度。
// 课时删除后其学习记录不再有业务意义，需一并清理避免悬挂引用。
func (r *progressRepository) DeleteByLessonID(ctx context.Context, lessonID int64) error {
	return r.db.WithContext(ctx).
		Where("lesson_id = ?", lessonID).
		Delete(&entity.LearningProgress{}).Error
}
