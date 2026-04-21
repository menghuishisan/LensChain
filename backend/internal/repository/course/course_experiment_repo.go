// course_experiment_repo.go
// 模块03 — 课程与教学：课程实验关联数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// CourseExperimentRepository 课程实验关联数据访问接口
type CourseExperimentRepository interface {
	Create(ctx context.Context, exp *entity.CourseExperiment) error
	Delete(ctx context.Context, id int64) error
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseExperiment, error)
}

type courseExperimentRepository struct {
	db *gorm.DB
}

// NewCourseExperimentRepository 创建课程实验关联数据访问实例
func NewCourseExperimentRepository(db *gorm.DB) CourseExperimentRepository {
	return &courseExperimentRepository{db: db}
}

// Create 创建课程实验关联
func (r *courseExperimentRepository) Create(ctx context.Context, exp *entity.CourseExperiment) error {
	if exp.ID == 0 {
		exp.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(exp).Error
}

// Delete 删除课程实验关联
func (r *courseExperimentRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CourseExperiment{}, id).Error
}

// ListByCourseID 获取课程下的实验关联列表
func (r *courseExperimentRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseExperiment, error) {
	var exps []*entity.CourseExperiment
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order asc, created_at asc").
		Find(&exps).Error
	return exps, err
}
