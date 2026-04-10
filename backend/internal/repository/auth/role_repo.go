// role_repo.go
// 模块01 — 用户与认证：角色与权限数据访问层
// 负责 roles、user_roles、permissions、role_permissions 表的操作

package authrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// RoleRepository 角色数据访问接口
type RoleRepository interface {
	GetByCode(ctx context.Context, code string) (*entity.Role, error)
	GetByID(ctx context.Context, id int64) (*entity.Role, error)
	GetAll(ctx context.Context) ([]*entity.Role, error)

	// 用户角色关联
	AssignRole(ctx context.Context, userID, roleID int64) error
	RemoveRole(ctx context.Context, userID, roleID int64) error
	GetUserRoles(ctx context.Context, userID int64) ([]*entity.Role, error)
	GetUserRoleCodes(ctx context.Context, userID int64) ([]string, error)
	BatchAssignRole(ctx context.Context, userIDs []int64, roleID int64) error
}

// roleRepository 角色数据访问实现
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色数据访问实例
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// GetByCode 根据角色编码获取角色
func (r *roleRepository) GetByCode(ctx context.Context, code string) (*entity.Role, error) {
	var role entity.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByID 根据ID获取角色
func (r *roleRepository) GetByID(ctx context.Context, id int64) (*entity.Role, error) {
	var role entity.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetAll 获取所有角色
func (r *roleRepository) GetAll(ctx context.Context) ([]*entity.Role, error) {
	var roles []*entity.Role
	err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error
	return roles, err
}

// AssignRole 为用户分配角色
func (r *roleRepository) AssignRole(ctx context.Context, userID, roleID int64) error {
	userRole := &entity.UserRole{
		ID:     snowflake.Generate(),
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}

// RemoveRole 移除用户角色
func (r *roleRepository) RemoveRole(ctx context.Context, userID, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&entity.UserRole{}).Error
}

// GetUserRoles 获取用户的所有角色
func (r *roleRepository) GetUserRoles(ctx context.Context, userID int64) ([]*entity.Role, error) {
	var roles []*entity.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// GetUserRoleCodes 获取用户的角色编码列表
func (r *roleRepository) GetUserRoleCodes(ctx context.Context, userID int64) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Model(&entity.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Pluck("code", &codes).Error
	return codes, err
}

// BatchAssignRole 批量为用户分配角色
func (r *roleRepository) BatchAssignRole(ctx context.Context, userIDs []int64, roleID int64) error {
	userRoles := make([]*entity.UserRole, 0, len(userIDs))
	for _, userID := range userIDs {
		userRoles = append(userRoles, &entity.UserRole{
			ID:     snowflake.Generate(),
			UserID: userID,
			RoleID: roleID,
		})
	}
	return r.db.WithContext(ctx).CreateInBatches(userRoles, 100).Error
}
