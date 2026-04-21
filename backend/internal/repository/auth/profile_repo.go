// profile_repo.go
// 模块01 — 用户与认证：用户扩展信息数据访问层
// 负责 user_profiles 表的 CRUD 操作

package authrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ProfileRepository 用户扩展信息数据访问接口
type ProfileRepository interface {
	Create(ctx context.Context, profile *entity.UserProfile) error
	GetByUserID(ctx context.Context, userID int64) (*entity.UserProfile, error)
	GetByUserIDs(ctx context.Context, userIDs []int64) ([]*entity.UserProfile, error)
	Update(ctx context.Context, profile *entity.UserProfile) error
	UpdateFields(ctx context.Context, userID int64, fields map[string]interface{}) error
	BatchCreate(ctx context.Context, profiles []*entity.UserProfile) error
	DeleteByUserID(ctx context.Context, userID int64) error
}

// profileRepository 用户扩展信息数据访问实现
type profileRepository struct {
	db *gorm.DB
}

// NewProfileRepository 创建用户扩展信息数据访问实例
func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

// Create 创建用户扩展信息
func (r *profileRepository) Create(ctx context.Context, profile *entity.UserProfile) error {
	if profile.ID == 0 {
		profile.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(profile).Error
}

// GetByUserID 根据用户ID获取扩展信息
func (r *profileRepository) GetByUserID(ctx context.Context, userID int64) (*entity.UserProfile, error) {
	var profile entity.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetByUserIDs 根据用户ID列表批量获取扩展信息
// 用于用户列表、个人信息等场景统一补齐 profile 数据，避免上层逐条查询。
func (r *profileRepository) GetByUserIDs(ctx context.Context, userIDs []int64) ([]*entity.UserProfile, error) {
	if len(userIDs) == 0 {
		return []*entity.UserProfile{}, nil
	}

	var profiles []*entity.UserProfile
	err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&profiles).Error
	return profiles, err
}

// Update 更新用户扩展信息
func (r *profileRepository) Update(ctx context.Context, profile *entity.UserProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

// UpdateFields 更新指定字段
func (r *profileRepository) UpdateFields(ctx context.Context, userID int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&entity.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(fields).Error
}

// BatchCreate 批量创建用户扩展信息
func (r *profileRepository) BatchCreate(ctx context.Context, profiles []*entity.UserProfile) error {
	if len(profiles) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).CreateInBatches(profiles, repositoryBatchSize).Error
}

// DeleteByUserID 根据用户ID删除扩展信息
func (r *profileRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.UserProfile{}).Error
}
