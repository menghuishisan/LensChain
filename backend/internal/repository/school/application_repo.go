// application_repo.go
// 模块02 — 学校与租户管理：入驻申请数据访问层
// 负责入驻申请记录的 CRUD 操作
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

package schoolrepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ApplicationRepository 入驻申请数据访问接口
type ApplicationRepository interface {
	Create(ctx context.Context, app *entity.SchoolApplication) error
	GetByID(ctx context.Context, id int64) (*entity.SchoolApplication, error)
	GetPendingByPhone(ctx context.Context, phone string) (*entity.SchoolApplication, error)
	ListByPhone(ctx context.Context, phone string) ([]*entity.SchoolApplication, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	List(ctx context.Context, params *ApplicationListParams) ([]*entity.SchoolApplication, int64, error)
}

// ApplicationListParams 申请列表查询参数
type ApplicationListParams struct {
	Status    int
	Keyword   string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

// applicationRepository 入驻申请数据访问实现
type applicationRepository struct {
	db *gorm.DB
}

// NewApplicationRepository 创建入驻申请数据访问实例
func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

// Create 创建申请记录
func (r *applicationRepository) Create(ctx context.Context, app *entity.SchoolApplication) error {
	if app.ID == 0 {
		app.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(app).Error
}

// GetByID 根据ID获取申请记录
func (r *applicationRepository) GetByID(ctx context.Context, id int64) (*entity.SchoolApplication, error) {
	var app entity.SchoolApplication
	err := r.db.WithContext(ctx).First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetPendingByPhone 根据手机号查询待审核的申请（用于重复提交检测）
func (r *applicationRepository) GetPendingByPhone(ctx context.Context, phone string) (*entity.SchoolApplication, error) {
	var app entity.SchoolApplication
	err := r.db.WithContext(ctx).
		Where("contact_phone = ? AND status = ?", phone, 1).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// ListByPhone 根据手机号查询所有申请记录（按时间倒序）
func (r *applicationRepository) ListByPhone(ctx context.Context, phone string) ([]*entity.SchoolApplication, error) {
	var apps []*entity.SchoolApplication
	err := r.db.WithContext(ctx).
		Where("contact_phone = ?", phone).
		Order("created_at DESC").
		Find(&apps).Error
	return apps, err
}

// UpdateFields 更新申请记录指定字段
func (r *applicationRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SchoolApplication{}).Where("id = ?", id).Updates(fields).Error
}

// List 申请列表查询
func (r *applicationRepository) List(ctx context.Context, params *ApplicationListParams) ([]*entity.SchoolApplication, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.SchoolApplication{})

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 关键字搜索（学校名称、联系人姓名、手机号）
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "school_name", "contact_name", "contact_phone"))
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
		"created_at":  "created_at",
		"school_name": "school_name",
		"status":      "status",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// 分页
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	var apps []*entity.SchoolApplication
	if err := query.Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}
