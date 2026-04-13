// grade_override_repo.go
// 模块03 — 课程与教学：课程成绩调整记录数据访问层
// 负责 course_grade_overrides 表的查询与写入

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// GradeOverrideRepository 成绩调整记录数据访问接口
// 提供按课程+学生读取和写入手动调分结果的能力。
type GradeOverrideRepository interface {
	GetByCourseAndStudent(ctx context.Context, courseID, studentID int64) (*entity.CourseGradeOverride, error)
	Upsert(ctx context.Context, override *entity.CourseGradeOverride) error
}

type gradeOverrideRepository struct {
	db *gorm.DB
}

// NewGradeOverrideRepository 创建成绩调整记录数据访问实例
func NewGradeOverrideRepository(db *gorm.DB) GradeOverrideRepository {
	return &gradeOverrideRepository{db: db}
}

// GetByCourseAndStudent 获取课程下某学生的成绩调整记录
func (r *gradeOverrideRepository) GetByCourseAndStudent(ctx context.Context, courseID, studentID int64) (*entity.CourseGradeOverride, error) {
	var override entity.CourseGradeOverride
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND student_id = ?", courseID, studentID).
		First(&override).Error
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// Upsert 保存课程成绩调整记录
// 若课程下该学生已存在调整记录则更新，否则创建新记录。
func (r *gradeOverrideRepository) Upsert(ctx context.Context, override *entity.CourseGradeOverride) error {
	existing, err := r.GetByCourseAndStudent(ctx, override.CourseID, override.StudentID)
	if err != nil {
		if override.ID == 0 {
			override.ID = snowflake.Generate()
		}
		return r.db.WithContext(ctx).Create(override).Error
	}

	return r.db.WithContext(ctx).Model(&entity.CourseGradeOverride{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"weighted_total": override.WeightedTotal,
			"final_score":    override.FinalScore,
			"adjust_reason":  override.AdjustReason,
			"adjusted_by":    override.AdjustedBy,
			"adjusted_at":    override.AdjustedAt,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}
