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

	// 权限点与角色权限关联
	GetPermissionByCode(ctx context.Context, code string) (*entity.Permission, error)
	GetPermissionsByModule(ctx context.Context, module string) ([]*entity.Permission, error)
	GetRolePermissions(ctx context.Context, roleID int64) ([]*entity.Permission, error)
	GetPermissionCodesByRoleCodes(ctx context.Context, roleCodes []string) ([]string, error)
	AssignPermission(ctx context.Context, roleID, permissionID int64) error
	RemovePermission(ctx context.Context, roleID, permissionID int64) error

	// 用户角色关联
	AssignRole(ctx context.Context, userID, roleID int64) error
	RemoveRole(ctx context.Context, userID, roleID int64) error
	GetUserRoles(ctx context.Context, userID int64) ([]*entity.Role, error)
	GetUserRoleCodes(ctx context.Context, userID int64) ([]string, error)
	GetUserRoleCodesMap(ctx context.Context, userIDs []int64) (map[int64][]string, error)
	CountActiveUsersByRoleCode(ctx context.Context, roleCode string) (int64, error)
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

// GetPermissionByCode 根据权限编码获取权限点
func (r *roleRepository) GetPermissionByCode(ctx context.Context, code string) (*entity.Permission, error) {
	var permission entity.Permission
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetPermissionsByModule 根据模块获取权限点列表
func (r *roleRepository) GetPermissionsByModule(ctx context.Context, module string) ([]*entity.Permission, error) {
	var permissions []*entity.Permission
	err := r.db.WithContext(ctx).
		Where("module = ?", module).
		Order("code ASC").
		Find(&permissions).Error
	return permissions, err
}

// GetRolePermissions 获取角色已绑定的权限点列表
func (r *roleRepository) GetRolePermissions(ctx context.Context, roleID int64) ([]*entity.Permission, error) {
	var permissions []*entity.Permission
	err := r.db.WithContext(ctx).
		Model(&entity.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.code ASC").
		Find(&permissions).Error
	return permissions, err
}

// GetPermissionCodesByRoleCodes 根据角色编码批量获取权限编码
// JWT 和中间件当前以角色门禁为主，该方法为后续细粒度权限校验提供统一数据入口。
func (r *roleRepository) GetPermissionCodesByRoleCodes(ctx context.Context, roleCodes []string) ([]string, error) {
	if len(roleCodes) == 0 {
		return []string{}, nil
	}

	var codes []string
	err := r.db.WithContext(ctx).
		Model(&entity.Permission{}).
		Distinct("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Where("roles.code IN ?", roleCodes).
		Order("permissions.code ASC").
		Pluck("permissions.code", &codes).Error
	return codes, err
}

// AssignPermission 为角色绑定权限点
func (r *roleRepository) AssignPermission(ctx context.Context, roleID, permissionID int64) error {
	rolePermission := &entity.RolePermission{
		ID:           snowflake.Generate(),
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	return r.db.WithContext(ctx).Create(rolePermission).Error
}

// RemovePermission 移除角色权限点
func (r *roleRepository) RemovePermission(ctx context.Context, roleID, permissionID int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&entity.RolePermission{}).Error
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

// userRoleCode 用户角色编码查询结果
// 该结构只承载 roles 与 user_roles 关联查询结果，不作为 API 响应 DTO 使用。
type userRoleCode struct {
	UserID int64  `gorm:"column:user_id"`
	Code   string `gorm:"column:code"`
}

// GetUserRoleCodesMap 批量获取用户角色编码映射
// 用于列表和详情装配阶段避免逐个用户查询角色。
func (r *roleRepository) GetUserRoleCodesMap(ctx context.Context, userIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var rows []userRoleCode
	if err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("user_roles.user_id, roles.code").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id IN ?", userIDs).
		Order("user_roles.user_id ASC, roles.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row.Code)
	}
	return result, nil
}

// CountActiveUsersByRoleCode 统计指定角色下处于正常状态且未删除的用户数量。
// 用于“至少保留一个超级管理员”这类边界校验，不承载业务判断本身。
func (r *roleRepository) CountActiveUsersByRoleCode(ctx context.Context, roleCode string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("roles.code = ?", roleCode).
		Where("users.status = ?", 1).
		Where("users.deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// BatchAssignRole 批量为用户分配角色
func (r *roleRepository) BatchAssignRole(ctx context.Context, userIDs []int64, roleID int64) error {
	if len(userIDs) == 0 {
		return nil
	}

	userRoles := make([]*entity.UserRole, 0, len(userIDs))
	for _, userID := range userIDs {
		userRoles = append(userRoles, &entity.UserRole{
			ID:     snowflake.Generate(),
			UserID: userID,
			RoleID: roleID,
		})
	}
	return r.db.WithContext(ctx).CreateInBatches(userRoles, repositoryBatchSize).Error
}
