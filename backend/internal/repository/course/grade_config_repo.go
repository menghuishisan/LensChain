// grade_config_repo.go
// 模块03 — 课程与教学：课程成绩配置数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// GradeConfigRepository 成绩配置数据访问接口
type GradeConfigRepository interface {
	Upsert(ctx context.Context, config *entity.CourseGradeConfig) error
	GetByCourseID(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error)
}

type gradeConfigRepository struct {
	db *gorm.DB
}

// NewGradeConfigRepository 创建成绩配置数据访问实例
func NewGradeConfigRepository(db *gorm.DB) GradeConfigRepository {
	return &gradeConfigRepository{db: db}
}

// Upsert 创建或更新课程成绩配置
func (r *gradeConfigRepository) Upsert(ctx context.Context, config *entity.CourseGradeConfig) error {
	existing, err := r.GetByCourseID(ctx, config.CourseID)
	if err != nil {
		if config.ID == 0 {
			config.ID = snowflake.Generate()
		}
		return r.db.WithContext(ctx).Create(config).Error
	}
	return r.db.WithContext(ctx).Model(&entity.CourseGradeConfig{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"config":     config.Config,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

// GetByCourseID 根据课程 ID 获取成绩配置
func (r *gradeConfigRepository) GetByCourseID(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error) {
	var config entity.CourseGradeConfig
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}
