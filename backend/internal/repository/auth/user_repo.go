// user_repo.go
// 模块01 — 用户与认证：用户数据访问层
// 负责 users 表和 user_profiles 表的 CRUD 操作
// 不包含业务逻辑，仅负责数据库查询构建

package authrepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

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
	BatchCreate(ctx context.Context, users []*entity.User) error
	CountBySchoolID(ctx context.Context, schoolID int64) (int64, error)
	GetIDsBySchoolID(ctx context.Context, schoolID int64) ([]int64, error)
}

// UserListParams 用户列表查询参数
type UserListParams struct {
	SchoolID       int64
	Keyword        string
	Status         int
	Role           string
	College        string
	EducationLevel int
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

// GetByID 根据ID获取用户（含 Profile 和 Roles）
func (r *userRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("Roles").
		Preload("Roles.Role").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户（含 Roles）
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Role").
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

// BatchSoftDelete 批量软删除用户
// 校管只能删除本校用户，通过 schoolID 过滤
func (r *userRepository) BatchSoftDelete(ctx context.Context, ids []int64, schoolID int64) error {
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

	// 排序
	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at":    "created_at",
		"name":          "name",
		"last_login_at": "last_login_at",
		"status":        "status",
	}
	if params.SortBy != "" {
		if field, ok := allowedSortFields[params.SortBy]; ok {
			sortField = field
		}
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	// 预加载关联
	var users []*entity.User
	err := query.
		Preload("Profile").
		Preload("Roles").
		Preload("Roles.Role").
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
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
func (r *userRepository) UpdateLoginInfo(ctx context.Context, id int64, ip string, loginAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at":    loginAt,
			"last_login_ip":    ip,
			"login_fail_count": 0,
			"is_first_login":   false,
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
	var users []*entity.User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// BatchCreate 批量创建用户
func (r *userRepository) BatchCreate(ctx context.Context, users []*entity.User) error {
	return r.db.WithContext(ctx).CreateInBatches(users, 100).Error
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
