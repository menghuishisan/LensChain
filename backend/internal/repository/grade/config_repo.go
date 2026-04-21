// config_repo.go
// 模块06 — 评测与成绩：成绩等级映射与学业预警配置数据访问层。
// 负责 grade_level_configs、warning_configs 的读取、替换和学校级配置保存。

package graderepo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// GradeLevelConfigRepository 成绩等级映射配置数据访问接口。
type GradeLevelConfigRepository interface {
	ListBySchool(ctx context.Context, schoolID int64) ([]*entity.GradeLevelConfig, error)
	GetMatchedLevel(ctx context.Context, schoolID int64, score float64) (*entity.GradeLevelConfig, error)
	ReplaceBySchool(ctx context.Context, schoolID int64, configs []*entity.GradeLevelConfig) error
	DeleteBySchool(ctx context.Context, schoolID int64) error
}

type gradeLevelConfigRepository struct {
	db *gorm.DB
}

// NewGradeLevelConfigRepository 创建成绩等级映射配置数据访问实例。
func NewGradeLevelConfigRepository(db *gorm.DB) GradeLevelConfigRepository {
	return &gradeLevelConfigRepository{db: db}
}

// ListBySchool 查询学校等级映射配置。
func (r *gradeLevelConfigRepository) ListBySchool(ctx context.Context, schoolID int64) ([]*entity.GradeLevelConfig, error) {
	var configs []*entity.GradeLevelConfig
	err := r.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("sort_order asc, min_score desc").
		Find(&configs).Error
	return configs, err
}

// GetMatchedLevel 根据百分制成绩匹配等级配置。
func (r *gradeLevelConfigRepository) GetMatchedLevel(ctx context.Context, schoolID int64, score float64) (*entity.GradeLevelConfig, error) {
	var config entity.GradeLevelConfig
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND min_score <= ? AND max_score >= ?", schoolID, score, score).
		Order("sort_order asc").
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ReplaceBySchool 替换学校等级映射配置。
func (r *gradeLevelConfigRepository) ReplaceBySchool(ctx context.Context, schoolID int64, configs []*entity.GradeLevelConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("school_id = ?", schoolID).Delete(&entity.GradeLevelConfig{}).Error; err != nil {
			return err
		}
		if len(configs) == 0 {
			return nil
		}
		for i := range configs {
			if configs[i].ID == 0 {
				configs[i].ID = snowflake.Generate()
			}
			configs[i].SchoolID = schoolID
		}
		return tx.CreateInBatches(configs, 100).Error
	})
}

// DeleteBySchool 删除学校等级映射配置。
func (r *gradeLevelConfigRepository) DeleteBySchool(ctx context.Context, schoolID int64) error {
	return r.db.WithContext(ctx).Where("school_id = ?", schoolID).Delete(&entity.GradeLevelConfig{}).Error
}

// WarningConfigRepository 学业预警配置数据访问接口。
type WarningConfigRepository interface {
	GetBySchool(ctx context.Context, schoolID int64) (*entity.WarningConfig, error)
	Upsert(ctx context.Context, config *entity.WarningConfig) error
}

type warningConfigRepository struct {
	db *gorm.DB
}

// NewWarningConfigRepository 创建学业预警配置数据访问实例。
func NewWarningConfigRepository(db *gorm.DB) WarningConfigRepository {
	return &warningConfigRepository{db: db}
}

// GetBySchool 获取学校学业预警配置。
func (r *warningConfigRepository) GetBySchool(ctx context.Context, schoolID int64) (*entity.WarningConfig, error) {
	var config entity.WarningConfig
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Upsert 保存学校学业预警配置。
func (r *warningConfigRepository) Upsert(ctx context.Context, config *entity.WarningConfig) error {
	if config.ID == 0 {
		config.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "school_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"gpa_threshold",
			"fail_count_threshold",
			"is_enabled",
			"updated_at",
		}),
	}).Create(config).Error
}
