// school_repo.go
// 模块02 — 学校与租户管理：学校数据访问层
// 负责学校主表的 CRUD 操作
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

package schoolrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

const licenseExpiringWindowDays = 7

// SchoolRepository 学校数据访问接口
type SchoolRepository interface {
	Create(ctx context.Context, school *entity.School) error
	GetByID(ctx context.Context, id int64) (*entity.School, error)
	GetByIDIncludingDeleted(ctx context.Context, id int64) (*entity.School, error)
	GetByName(ctx context.Context, name string) (*entity.School, error)
	GetByCode(ctx context.Context, code string) (*entity.School, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	List(ctx context.Context, params *SchoolListParams) ([]*entity.School, int64, error)
	ListByStatus(ctx context.Context, status int16) ([]*entity.School, error)
	ListExpiredActive(ctx context.Context, before time.Time) ([]*entity.School, error)
	ListExpiringSoon(ctx context.Context, before time.Time) ([]*entity.School, error)
	ListBufferingExpired(ctx context.Context, before time.Time) ([]*entity.School, error)
	GetSSOEnabledSchools(ctx context.Context) ([]*entity.School, error)
}

// SchoolListParams 学校列表查询参数
type SchoolListParams struct {
	Keyword         string
	Status          int16
	LicenseExpiring bool
	SortBy          string
	SortOrder       string
	Page            int
	PageSize        int
}

// schoolRepository 学校数据访问实现
type schoolRepository struct {
	db *gorm.DB
}

// NewSchoolRepository 创建学校数据访问实例
func NewSchoolRepository(db *gorm.DB) SchoolRepository {
	return &schoolRepository{db: db}
}

// Create 创建学校
func (r *schoolRepository) Create(ctx context.Context, school *entity.School) error {
	if school.ID == 0 {
		school.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(school).Error
}

// GetByID 根据ID获取学校
func (r *schoolRepository) GetByID(ctx context.Context, id int64) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).First(&school, id).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// GetByIDIncludingDeleted 根据ID获取学校，包含已软删除记录
// 学校恢复流程需要先读取已注销学校，普通详情查询不得使用该方法。
func (r *schoolRepository) GetByIDIncludingDeleted(ctx context.Context, id int64) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).Unscoped().First(&school, id).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// GetByName 根据名称获取学校（用于唯一性校验）
// 业务规则要求学校名称全局唯一，因此这里包含已注销学校。
func (r *schoolRepository) GetByName(ctx context.Context, name string) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).Unscoped().Where("name = ?", name).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// GetByCode 根据编码获取学校（用于唯一性校验）
// 业务规则要求学校编码全局唯一，因此这里包含已注销学校。
func (r *schoolRepository) GetByCode(ctx context.Context, code string) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).Unscoped().Where("code = ?", code).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// UpdateFields 更新学校指定字段
func (r *schoolRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.School{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除学校
func (r *schoolRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.School{}, id).Error
}

// Restore 恢复已注销学校（清除 deleted_at）
func (r *schoolRepository) Restore(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Unscoped().Model(&entity.School{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"updated_at": time.Now(),
		}).Error
}

// List 学校列表查询
// 后台列表需要包含已注销学校，便于超管执行恢复操作，因此这里使用 Unscoped 查询。
func (r *schoolRepository) List(ctx context.Context, params *SchoolListParams) ([]*entity.School, int64, error) {
	query := r.db.WithContext(ctx).Unscoped().Model(&entity.School{})

	// 关键字搜索（学校名称、编码）
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "code"))
	}

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 7天内即将到期筛选
	if params.LicenseExpiring {
		now := time.Now()
		expiringDeadline := now.AddDate(0, 0, licenseExpiringWindowDays)
		query = query.Where(
			"status = ? AND license_end_at IS NOT NULL AND license_end_at > ? AND license_end_at <= ?",
			enum.SchoolStatusActive,
			now,
			expiringDeadline,
		)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":     "created_at",
		"name":           "name",
		"status":         "status",
		"license_end_at": "license_end_at",
	}
	pageQuery := pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    normalizeSchoolSortBy(params.SortBy),
		SortOrder: params.SortOrder,
	}
	query = pageQuery.ApplyToGORM(query, allowedSortFields)

	var schools []*entity.School
	if err := query.Find(&schools).Error; err != nil {
		return nil, 0, err
	}

	return schools, total, nil
}

// normalizeSchoolSortBy 统一学校列表默认排序字段。
func normalizeSchoolSortBy(sortBy string) string {
	switch sortBy {
	case "name", "status", "license_end_at":
		return sortBy
	default:
		return "created_at"
	}
}

// ListByStatus 根据状态查询学校列表
func (r *schoolRepository) ListByStatus(ctx context.Context, status int16) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&schools).Error
	return schools, err
}

// ListExpiredActive 查询已到期且仍处于激活状态的学校
// 到期转缓冲期任务使用该查询，避免把冻结或注销学校重复转态。
func (r *schoolRepository) ListExpiredActive(ctx context.Context, before time.Time) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).
		Where("status = ? AND license_end_at IS NOT NULL AND license_end_at <= ?",
			enum.SchoolStatusActive, before).
		Find(&schools).Error
	return schools, err
}

// ListExpiringSoon 查询即将到期的学校（7天内到期且状态为已激活）
func (r *schoolRepository) ListExpiringSoon(ctx context.Context, before time.Time) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).
		Where("status = ? AND license_end_at IS NOT NULL AND license_end_at <= ? AND license_end_at > ?",
			enum.SchoolStatusActive, before, time.Now()).
		Find(&schools).Error
	return schools, err
}

// ListBufferingExpired 查询缓冲期已满7天的学校
func (r *schoolRepository) ListBufferingExpired(ctx context.Context, before time.Time) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).
		Where("status = ? AND license_end_at IS NOT NULL AND license_end_at <= ?",
			enum.SchoolStatusBuffering, before).
		Find(&schools).Error
	return schools, err
}

// GetSSOEnabledSchools 获取可出现在登录页 SSO 学校列表中的学校
// 仅返回状态为已激活、授权未过期，且 SSO 已启用并通过测试的学校。
func (r *schoolRepository) GetSSOEnabledSchools(ctx context.Context) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).
		Where("status = ?", enum.SchoolStatusActive).
		Where("(license_end_at IS NULL OR license_end_at > ?)", time.Now()).
		Where("id IN (?)",
			r.db.Model(&entity.SchoolSSOConfig{}).
				Select("school_id").
				Where("is_enabled = ? AND is_tested = ?", true, true),
		).
		Find(&schools).Error
	return schools, err
}
