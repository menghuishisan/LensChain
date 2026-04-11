// school_repo.go
// 模块02 — 学校与租户管理：学校数据访问层
// 负责学校主表的 CRUD 操作
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

package schoolrepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SchoolRepository 学校数据访问接口
type SchoolRepository interface {
	Create(ctx context.Context, school *entity.School) error
	GetByID(ctx context.Context, id int64) (*entity.School, error)
	GetByName(ctx context.Context, name string) (*entity.School, error)
	GetByCode(ctx context.Context, code string) (*entity.School, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	List(ctx context.Context, params *SchoolListParams) ([]*entity.School, int64, error)
	ListByStatus(ctx context.Context, status int) ([]*entity.School, error)
	ListExpiringSoon(ctx context.Context, before time.Time) ([]*entity.School, error)
	ListBufferingExpired(ctx context.Context, before time.Time) ([]*entity.School, error)
	GetSSOEnabledSchools(ctx context.Context) ([]*entity.School, error)
}

// SchoolListParams 学校列表查询参数
type SchoolListParams struct {
	Keyword         string
	Status          int
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

// GetByName 根据名称获取学校（用于唯一性校验）
func (r *schoolRepository) GetByName(ctx context.Context, name string) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&school).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// GetByCode 根据编码获取学校（用于唯一性校验）
func (r *schoolRepository) GetByCode(ctx context.Context, code string) (*entity.School, error) {
	var school entity.School
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&school).Error
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
func (r *schoolRepository) List(ctx context.Context, params *SchoolListParams) ([]*entity.School, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.School{})

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
		sevenDaysLater := now.AddDate(0, 0, 7)
		query = query.Where("license_end_at IS NOT NULL AND license_end_at > ? AND license_end_at <= ?", now, sevenDaysLater)
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
		"created_at":     "created_at",
		"name":           "name",
		"status":         "status",
		"license_end_at": "license_end_at",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var schools []*entity.School
	if err := query.Find(&schools).Error; err != nil {
		return nil, 0, err
	}

	return schools, total, nil
}

// ListByStatus 根据状态查询学校列表
func (r *schoolRepository) ListByStatus(ctx context.Context, status int) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&schools).Error
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

// GetSSOEnabledSchools 获取已启用SSO且已通过测试的学校
func (r *schoolRepository) GetSSOEnabledSchools(ctx context.Context) ([]*entity.School, error) {
	var schools []*entity.School
	err := r.db.WithContext(ctx).
		Where("status = ?", 2).
		Where("id IN (?)",
			r.db.Model(&entity.SchoolSSOConfig{}).
				Select("school_id").
				Where("is_enabled = ? AND is_tested = ?", true, true),
		).
		Find(&schools).Error
	return schools, err
}
