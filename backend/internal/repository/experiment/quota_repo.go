// quota_repo.go
// 模块04 — 实验环境：资源配额数据访问层
// 负责资源配额的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// QuotaRepository 资源配额数据访问接口
type QuotaRepository interface {
	Create(ctx context.Context, quota *entity.ResourceQuota) error
	GetByID(ctx context.Context, id int64) (*entity.ResourceQuota, error)
	GetBySchoolID(ctx context.Context, schoolID int64) (*entity.ResourceQuota, error)
	GetByCourseID(ctx context.Context, courseID int64) (*entity.ResourceQuota, error)
	GetBySchoolAndCourse(ctx context.Context, schoolID int64, courseID *int64) (*entity.ResourceQuota, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params *QuotaListParams) ([]*entity.ResourceQuota, int64, error)
	ListBySchoolID(ctx context.Context, schoolID int64) ([]*entity.ResourceQuota, error)
	IncrUsedConcurrency(ctx context.Context, id int64, delta int) error
	DecrUsedConcurrency(ctx context.Context, id int64, delta int) error
}

// QuotaListParams 配额列表查询参数
type QuotaListParams struct {
	SchoolID   int64
	QuotaLevel int
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// quotaRepository 资源配额数据访问实现
type quotaRepository struct {
	db *gorm.DB
}

// NewQuotaRepository 创建资源配额数据访问实例
func NewQuotaRepository(db *gorm.DB) QuotaRepository {
	return &quotaRepository{db: db}
}

// Create 创建资源配额
func (r *quotaRepository) Create(ctx context.Context, quota *entity.ResourceQuota) error {
	if quota.ID == 0 {
		quota.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(quota).Error
}

// GetByID 根据ID获取资源配额
func (r *quotaRepository) GetByID(ctx context.Context, id int64) (*entity.ResourceQuota, error) {
	var quota entity.ResourceQuota
	err := r.db.WithContext(ctx).First(&quota, id).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// GetBySchoolID 获取学校级别配额
func (r *quotaRepository) GetBySchoolID(ctx context.Context, schoolID int64) (*entity.ResourceQuota, error) {
	var quota entity.ResourceQuota
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND quota_level = 1 AND course_id IS NULL", schoolID).
		First(&quota).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// GetByCourseID 获取课程级别配额
func (r *quotaRepository) GetByCourseID(ctx context.Context, courseID int64) (*entity.ResourceQuota, error) {
	var quota entity.ResourceQuota
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND quota_level = 2", courseID).
		First(&quota).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// GetBySchoolAndCourse 获取指定学校和课程的配额
func (r *quotaRepository) GetBySchoolAndCourse(ctx context.Context, schoolID int64, courseID *int64) (*entity.ResourceQuota, error) {
	var quota entity.ResourceQuota
	query := r.db.WithContext(ctx).Where("school_id = ?", schoolID)
	if courseID != nil {
		query = query.Where("course_id = ?", *courseID)
	} else {
		query = query.Where("course_id IS NULL")
	}
	err := query.First(&quota).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// UpdateFields 更新资源配额指定字段
func (r *quotaRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ResourceQuota{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除资源配额
func (r *quotaRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ResourceQuota{}, id).Error
}

// List 配额列表查询
func (r *quotaRepository) List(ctx context.Context, params *QuotaListParams) ([]*entity.ResourceQuota, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ResourceQuota{})

	if params.SchoolID > 0 {
		query = query.Where("school_id = ?", params.SchoolID)
	}
	if params.QuotaLevel > 0 {
		query = query.Where("quota_level = ?", params.QuotaLevel)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at":  "created_at",
		"school_id":   "school_id",
		"quota_level": "quota_level",
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

	var quotas []*entity.ResourceQuota
	if err := query.Find(&quotas).Error; err != nil {
		return nil, 0, err
	}
	return quotas, total, nil
}

// ListBySchoolID 获取学校的所有配额（含课程级别）
func (r *quotaRepository) ListBySchoolID(ctx context.Context, schoolID int64) ([]*entity.ResourceQuota, error) {
	var quotas []*entity.ResourceQuota
	err := r.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("quota_level asc, created_at asc").
		Find(&quotas).Error
	return quotas, err
}

// IncrUsedConcurrency 增加已用并发数
func (r *quotaRepository) IncrUsedConcurrency(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.ResourceQuota{}).
		Where("id = ?", id).
		UpdateColumn("used_concurrency", gorm.Expr("used_concurrency + ?", delta)).Error
}

// DecrUsedConcurrency 减少已用并发数
func (r *quotaRepository) DecrUsedConcurrency(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.ResourceQuota{}).
		Where("id = ? AND used_concurrency >= ?", id, delta).
		UpdateColumn("used_concurrency", gorm.Expr("used_concurrency - ?", delta)).Error
}
