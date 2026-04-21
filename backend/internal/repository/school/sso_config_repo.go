// sso_config_repo.go
// 模块02 — 学校与租户管理：SSO配置数据访问层
// 负责 SSO 配置的 CRUD 操作
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

package schoolrepo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SSOConfigRepository SSO配置数据访问接口
type SSOConfigRepository interface {
	Create(ctx context.Context, config *entity.SchoolSSOConfig) error
	GetBySchoolID(ctx context.Context, schoolID int64) (*entity.SchoolSSOConfig, error)
	UpdateFields(ctx context.Context, schoolID int64, fields map[string]interface{}) error
	Upsert(ctx context.Context, config *entity.SchoolSSOConfig) error
	UpdateTestResult(ctx context.Context, schoolID int64, isTested bool, testedAt time.Time) error
	ToggleEnabled(ctx context.Context, schoolID int64, isEnabled bool, updatedBy int64) error
}

// ssoConfigRepository SSO配置数据访问实现
type ssoConfigRepository struct {
	db *gorm.DB
}

// NewSSOConfigRepository 创建SSO配置数据访问实例
func NewSSOConfigRepository(db *gorm.DB) SSOConfigRepository {
	return &ssoConfigRepository{db: db}
}

// Create 创建SSO配置
func (r *ssoConfigRepository) Create(ctx context.Context, config *entity.SchoolSSOConfig) error {
	if config.ID == 0 {
		config.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(config).Error
}

// GetBySchoolID 根据学校ID获取SSO配置
func (r *ssoConfigRepository) GetBySchoolID(ctx context.Context, schoolID int64) (*entity.SchoolSSOConfig, error) {
	var config entity.SchoolSSOConfig
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateFields 更新SSO配置指定字段
func (r *ssoConfigRepository) UpdateFields(ctx context.Context, schoolID int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SchoolSSOConfig{}).
		Where("school_id = ?", schoolID).
		Updates(fields).Error
}

// Upsert 创建或更新SSO配置（原子操作，避免竞态条件）
// 使用 PostgreSQL ON CONFLICT 实现原子 upsert
func (r *ssoConfigRepository) Upsert(ctx context.Context, config *entity.SchoolSSOConfig) error {
	if config.ID == 0 {
		config.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "school_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"provider", "is_enabled", "is_tested", "config", "tested_at", "updated_at", "updated_by",
		}),
	}).Create(config).Error
}

// UpdateTestResult 更新SSO连接测试结果
// 测试成功和失败都会记录 tested_at，供启用前校验和后台展示使用。
func (r *ssoConfigRepository) UpdateTestResult(ctx context.Context, schoolID int64, isTested bool, testedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.SchoolSSOConfig{}).
		Where("school_id = ?", schoolID).
		Updates(map[string]interface{}{
			"is_tested":  isTested,
			"tested_at":  testedAt,
			"updated_at": time.Now(),
		}).Error
}

// ToggleEnabled 启用或禁用学校SSO配置
// 只更新启用状态和修改人，是否允许启用由 service 层根据测试状态判断。
func (r *ssoConfigRepository) ToggleEnabled(ctx context.Context, schoolID int64, isEnabled bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Model(&entity.SchoolSSOConfig{}).
		Where("school_id = ?", schoolID).
		Updates(map[string]interface{}{
			"is_enabled": isEnabled,
			"updated_at": time.Now(),
			"updated_by": updatedBy,
		}).Error
}
