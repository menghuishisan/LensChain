// user_repo.go
// 模块01 — 用户与认证：用户数据访问层
// 负责 users 表的 CRUD、账号状态、登录安全字段和批量租户用户操作
// 不包含业务逻辑，仅负责数据库查询构建

package authrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

const repositoryBatchSize = 100

// UserRepository 用户数据访问接口
type UserRepository interface {
	// 用户 CRUD
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id int64) (*entity.User, error)
	GetByPhone(ctx context.Context, phone string) (*entity.User, error)
	GetBySchoolAndStudentNo(ctx context.Context, schoolID int64, studentNo string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	SoftDeleteBySchoolID(ctx context.Context, schoolID int64) error
	RestoreBySchoolID(ctx context.Context, schoolID int64) error
	BatchSoftDelete(ctx context.Context, ids []int64, schoolID int64) error
	List(ctx context.Context, params *UserListParams) ([]*entity.User, int64, error)

	// 登录相关
	IncrLoginFailCount(ctx context.Context, id int64) error
	ResetLoginFailCount(ctx context.Context, id int64) error
	UpdateLoginInfo(ctx context.Context, id int64, ip string, loginAt time.Time) error
	LockUser(ctx context.Context, id int64, lockedUntil time.Time) error
	UnlockUser(ctx context.Context, id int64) error
	UpdateTokenValidAfter(ctx context.Context, id int64, validAfter time.Time) error
	BatchUpdateTokenValidAfterBySchool(ctx context.Context, schoolID int64, validAfter time.Time) error

	// 批量操作
	GetByIDs(ctx context.Context, ids []int64) ([]*entity.User, error)
	GetByIDsIncludingDeleted(ctx context.Context, ids []int64) ([]*entity.User, error)
	BatchCreate(ctx context.Context, users []*entity.User) error
	CountBySchoolID(ctx context.Context, schoolID int64) (int64, error)
	GetIDsBySchoolID(ctx context.Context, schoolID int64) ([]int64, error)
	ListAdminPhonesBySchoolID(ctx context.Context, schoolID int64) ([]string, error)
}

// UserListParams 用户列表查询参数
type UserListParams struct {
	SchoolID       int64
	Keyword        string
	Status         int16
	Role           string
	College        string
	EducationLevel int16
	SortBy         string
	SortOrder      string
	Page           int
	PageSize       int
}

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if user.ID == 0 {
		user.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据ID获取用户主表记录
// 用户扩展信息和角色已按 entity 职责拆分，调用方需要时应使用 ProfileRepository 和 RoleRepository 查询。
func (r *userRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户主表记录
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("phone = ?", phone).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetBySchoolAndStudentNo 根据学校ID和学号获取用户
func (r *userRepository) GetBySchoolAndStudentNo(ctx context.Context, schoolID int64, studentNo string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND student_no = ?", schoolID, studentNo).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户（全量更新）
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateFields 更新用户指定字段
func (r *userRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除用户
func (r *userRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.User{}, id).Error
}

// SoftDeleteBySchoolID 软删除指定学校下的全部用户
// 学校注销时使用，保留用户历史数据但阻止继续登录。
func (r *userRepository) SoftDeleteBySchoolID(ctx context.Context, schoolID int64) error {
	if schoolID <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Delete(&entity.User{}).Error
}

// RestoreBySchoolID 恢复指定学校下已软删除的全部用户
// 学校从注销状态恢复时使用，清除 deleted_at 并刷新 updated_at。
func (r *userRepository) RestoreBySchoolID(ctx context.Context, schoolID int64) error {
	if schoolID <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Unscoped().
		Model(&entity.User{}).
		Where("school_id = ?", schoolID).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"updated_at": time.Now(),
		}).Error
}

// BatchSoftDelete 批量软删除用户
// 校管只能删除本校用户，通过 schoolID 过滤
func (r *userRepository) BatchSoftDelete(ctx context.Context, ids []int64, schoolID int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := r.db.WithContext(ctx).Where("id IN ?", ids)
	if schoolID > 0 {
		query = query.Where("school_id = ?", schoolID)
	}
	return query.Delete(&entity.User{}).Error
}

// List 用户列表查询（带分页、筛选、排序）
func (r *userRepository) List(ctx context.Context, params *UserListParams) ([]*entity.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.User{})

	// 多租户过滤
	query = query.Scopes(database.WithSchoolID(params.SchoolID))

	// 关键字搜索（姓名、手机号、学号）
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "phone", "student_no"))
	}

	// 状态过滤
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 角色过滤（通过子查询）
	if params.Role != "" {
		query = query.Where("id IN (?)",
			r.db.Model(&entity.UserRole{}).
				Select("user_id").
				Joins("JOIN roles ON roles.id = user_roles.role_id").
				Where("roles.code = ?", params.Role),
		)
	}

	// 学院过滤（通过 user_profiles 子查询）
	if params.College != "" {
		query = query.Where("id IN (?)",
			r.db.Model(&entity.UserProfile{}).
				Select("user_id").
				Where("college = ?", params.College),
		)
	}

	// 学业层次过滤
	if params.EducationLevel > 0 {
		query = query.Where("id IN (?)",
			r.db.Model(&entity.UserProfile{}).
				Select("user_id").
				Where("education_level = ?", params.EducationLevel),
		)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":    "created_at",
		"name":          "name",
		"last_login_at": "last_login_at",
		"status":        "status",
	}
	pageQuery := pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    normalizeUserSortBy(params.SortBy),
		SortOrder: params.SortOrder,
	}
	query = pageQuery.ApplyToGORM(query, allowedSortFields)

	var users []*entity.User
	err := query.Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// normalizeUserSortBy 统一用户列表默认排序字段。
// pkg/pagination 负责排序白名单应用，这里只补齐模块01文档要求的默认 created_at。
func normalizeUserSortBy(sortBy string) string {
	switch sortBy {
	case "name", "last_login_at", "status":
		return sortBy
	default:
		return "created_at"
	}
}

// IncrLoginFailCount 增加登录失败次数
func (r *userRepository) IncrLoginFailCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		UpdateColumn("login_fail_count", gorm.Expr("login_fail_count + 1")).
		Error
}

// ResetLoginFailCount 重置登录失败次数
func (r *userRepository) ResetLoginFailCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"login_fail_count": 0,
			"locked_until":     nil,
		}).Error
}

// UpdateLoginInfo 更新最后登录信息
// 该方法只维护登录时间、登录IP和失败计数，首次登录状态由 service 层按业务流程显式更新。
func (r *userRepository) UpdateLoginInfo(ctx context.Context, id int64, ip string, loginAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at":    loginAt,
			"last_login_ip":    ip,
			"login_fail_count": 0,
		}).Error
}

// LockUser 锁定用户
func (r *userRepository) LockUser(ctx context.Context, id int64, lockedUntil time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Update("locked_until", lockedUntil).Error
}

// UnlockUser 解锁用户
func (r *userRepository) UnlockUser(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"locked_until":     nil,
			"login_fail_count": 0,
		}).Error
}

// UpdateTokenValidAfter 更新用户Token生效时间基线
// 用于强制历史 Access Token 失效。
func (r *userRepository) UpdateTokenValidAfter(ctx context.Context, id int64, validAfter time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Update("token_valid_after", validAfter).Error
}

// BatchUpdateTokenValidAfterBySchool 批量更新学校下所有用户的Token生效时间基线
// 用于学校冻结、注销等需要统一强制下线的场景。
func (r *userRepository) BatchUpdateTokenValidAfterBySchool(ctx context.Context, schoolID int64, validAfter time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("school_id = ? AND deleted_at IS NULL", schoolID).
		Update("token_valid_after", validAfter).Error
}

// GetByIDs 根据ID列表批量获取用户
func (r *userRepository) GetByIDs(ctx context.Context, ids []int64) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}

	var users []*entity.User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// GetByIDsIncludingDeleted 根据ID列表批量获取用户，包含已软删除记录
// 日志表需要保留历史用户名称时使用，避免软删除账号后历史日志无法补齐操作人信息。
func (r *userRepository) GetByIDsIncludingDeleted(ctx context.Context, ids []int64) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}

	var users []*entity.User
	err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// BatchCreate 批量创建用户
func (r *userRepository) BatchCreate(ctx context.Context, users []*entity.User) error {
	if len(users) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(users, repositoryBatchSize).Error
}

// CountBySchoolID 统计学校用户数
func (r *userRepository) CountBySchoolID(ctx context.Context, schoolID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).Where("school_id = ?", schoolID).Count(&count).Error
	return count, err
}

// GetIDsBySchoolID 获取学校所有用户ID
// 用于跨模块踢出学校所有用户的 Session
func (r *userRepository) GetIDsBySchoolID(ctx context.Context, schoolID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("school_id = ?", schoolID).
		Pluck("id", &ids).Error
	return ids, err
}

// ListAdminPhonesBySchoolID 获取学校管理员手机号列表。
// 仅返回未软删除且标记为学校管理员的账号手机号，用于学校模块发送授权提醒。
func (r *userRepository) ListAdminPhonesBySchoolID(ctx context.Context, schoolID int64) ([]string, error) {
	var phones []string
	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("school_id = ? AND is_school_admin = ? AND deleted_at IS NULL", schoolID, true).
		Distinct().
		Pluck("phone", &phones).Error
	return phones, err
}
