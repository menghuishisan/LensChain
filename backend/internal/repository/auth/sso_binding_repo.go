// sso_binding_repo.go
// 模块01 — 用户与认证：SSO绑定数据访问层
// 负责 user_sso_bindings 表的查询与写入

package authrepo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SSOBindingRepository SSO绑定数据访问接口
type SSOBindingRepository interface {
	GetBySchoolAndSSOUserID(ctx context.Context, schoolID int64, ssoUserID string) (*entity.UserSSOBinding, error)
	Upsert(ctx context.Context, binding *entity.UserSSOBinding) error
	UpdateLastLoginAt(ctx context.Context, id int64, loginAt time.Time) error
}

type ssoBindingRepository struct {
	db *gorm.DB
}

// NewSSOBindingRepository 创建 SSO 绑定仓储
func NewSSOBindingRepository(db *gorm.DB) SSOBindingRepository {
	return &ssoBindingRepository{db: db}
}

// GetBySchoolAndSSOUserID 按学校与SSO用户标识查询绑定
func (r *ssoBindingRepository) GetBySchoolAndSSOUserID(ctx context.Context, schoolID int64, ssoUserID string) (*entity.UserSSOBinding, error) {
	var binding entity.UserSSOBinding
	if err := r.db.WithContext(ctx).
		Where("school_id = ? AND sso_user_id = ?", schoolID, ssoUserID).
		First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

// Upsert 创建或更新SSO绑定
func (r *ssoBindingRepository) Upsert(ctx context.Context, binding *entity.UserSSOBinding) error {
	if binding.ID == 0 {
		binding.ID = snowflake.Generate()
	}
	if binding.BoundAt.IsZero() {
		binding.BoundAt = time.Now()
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "school_id"}, {Name: "sso_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "sso_provider", "last_login_at",
		}),
	}).Create(binding).Error
}

// UpdateLastLoginAt 更新最后SSO登录时间
func (r *ssoBindingRepository) UpdateLastLoginAt(ctx context.Context, id int64, loginAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.UserSSOBinding{}).
		Where("id = ?", id).
		Update("last_login_at", loginAt).
		Error
}
