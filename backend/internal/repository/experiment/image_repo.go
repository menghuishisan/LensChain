// image_repo.go
// 模块04 — 实验环境：镜像数据访问层
// 负责镜像分类、镜像主表、镜像版本的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 镜像分类 Repository
// ---------------------------------------------------------------------------

// ImageCategoryRepository 镜像分类数据访问接口
type ImageCategoryRepository interface {
	Create(ctx context.Context, category *entity.ImageCategory) error
	GetByID(ctx context.Context, id int64) (*entity.ImageCategory, error)
	GetByCode(ctx context.Context, code string) (*entity.ImageCategory, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*entity.ImageCategory, error)
}

// imageCategoryRepository 镜像分类数据访问实现
type imageCategoryRepository struct {
	db *gorm.DB
}

// NewImageCategoryRepository 创建镜像分类数据访问实例
func NewImageCategoryRepository(db *gorm.DB) ImageCategoryRepository {
	return &imageCategoryRepository{db: db}
}

// Create 创建镜像分类
func (r *imageCategoryRepository) Create(ctx context.Context, category *entity.ImageCategory) error {
	if category.ID == 0 {
		category.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(category).Error
}

// GetByID 根据ID获取镜像分类
func (r *imageCategoryRepository) GetByID(ctx context.Context, id int64) (*entity.ImageCategory, error) {
	var category entity.ImageCategory
	err := r.db.WithContext(ctx).First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetByCode 根据编码获取镜像分类
func (r *imageCategoryRepository) GetByCode(ctx context.Context, code string) (*entity.ImageCategory, error) {
	var category entity.ImageCategory
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// UpdateFields 更新镜像分类指定字段
func (r *imageCategoryRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ImageCategory{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除镜像分类（硬删除）
func (r *imageCategoryRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ImageCategory{}, id).Error
}

// ListAll 获取所有镜像分类（按 sort_order 排序）
func (r *imageCategoryRepository) ListAll(ctx context.Context) ([]*entity.ImageCategory, error) {
	var categories []*entity.ImageCategory
	err := r.db.WithContext(ctx).Order("sort_order asc, id asc").Find(&categories).Error
	return categories, err
}

// ---------------------------------------------------------------------------
// 镜像主表 Repository
// ---------------------------------------------------------------------------

// ImageRepository 镜像数据访问接口
type ImageRepository interface {
	Create(ctx context.Context, image *entity.Image) error
	GetByID(ctx context.Context, id int64) (*entity.Image, error)
	GetByName(ctx context.Context, name string) (*entity.Image, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *ImageListParams) ([]*entity.Image, int64, error)
	ListBySchoolID(ctx context.Context, params *SchoolImageListParams) ([]*entity.Image, int64, error)
	CountByCategoryID(ctx context.Context, categoryID int64) (int64, error)
	CountTemplateReferences(ctx context.Context, imageID int64) (int64, error)
	IncrUsageCount(ctx context.Context, id int64) error
}

// ImageListParams 镜像列表查询参数
type ImageListParams struct {
	Keyword    string
	CategoryID int64
	Ecosystem  string
	SourceType int16
	Status     int16
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// SchoolImageListParams 本校镜像列表查询参数
type SchoolImageListParams struct {
	SchoolID   int64
	Keyword    string
	CategoryID int64
	Status     int16
	Page       int
	PageSize   int
}

// imageRepository 镜像数据访问实现
type imageRepository struct {
	db *gorm.DB
}

// NewImageRepository 创建镜像数据访问实例
func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

// Create 创建镜像
func (r *imageRepository) Create(ctx context.Context, image *entity.Image) error {
	if image.ID == 0 {
		image.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(image).Error
}

// GetByID 根据ID获取镜像
func (r *imageRepository) GetByID(ctx context.Context, id int64) (*entity.Image, error) {
	var image entity.Image
	err := r.db.WithContext(ctx).First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// GetByName 根据镜像名称获取镜像，供创建/更新时做唯一性校验。
func (r *imageRepository) GetByName(ctx context.Context, name string) (*entity.Image, error) {
	var image entity.Image
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// UpdateFields 更新镜像指定字段
func (r *imageRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Image{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除镜像
func (r *imageRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Image{}, id).Error
}

// List 镜像列表查询（全平台视角）
func (r *imageRepository) List(ctx context.Context, params *ImageListParams) ([]*entity.Image, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Image{})

	// 关键字搜索
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "display_name", "description"))
	}

	// 分类筛选
	if params.CategoryID > 0 {
		query = query.Where("category_id = ?", params.CategoryID)
	}

	// 生态筛选
	if params.Ecosystem != "" {
		query = query.Where("ecosystem = ?", params.Ecosystem)
	}

	// 来源类型筛选
	if params.SourceType > 0 {
		query = query.Where("source_type = ?", params.SourceType)
	}

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":  "created_at",
		"name":        "name",
		"usage_count": "usage_count",
		"status":      "status",
	}
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, allowedSortFields)

	var images []*entity.Image
	if err := query.Find(&images).Error; err != nil {
		return nil, 0, err
	}
	return images, total, nil
}

// ListBySchoolID 本校镜像列表查询
func (r *imageRepository) ListBySchoolID(ctx context.Context, params *SchoolImageListParams) ([]*entity.Image, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Image{}).
		Where("school_id = ? OR source_type = ?", params.SchoolID, enum.ImageSourceTypeOfficial)

	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "display_name"))
	}
	if params.CategoryID > 0 {
		query = query.Where("category_id = ?", params.CategoryID)
	}
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var images []*entity.Image
	if err := query.Find(&images).Error; err != nil {
		return nil, 0, err
	}
	return images, total, nil
}

// CountByCategoryID 统计分类下的镜像数量
func (r *imageRepository) CountByCategoryID(ctx context.Context, categoryID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Image{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error
	return count, err
}

// CountTemplateReferences 统计镜像被模板容器引用的次数，支撑“被引用镜像不可下架/删除”规则。
func (r *imageRepository) CountTemplateReferences(ctx context.Context, imageID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TemplateContainer{}).
		Joins("JOIN image_versions ON image_versions.id = template_containers.image_version_id").
		Where("image_versions.image_id = ?", imageID).
		Count(&count).Error
	return count, err
}

// IncrUsageCount 增加镜像使用次数
func (r *imageRepository) IncrUsageCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.Image{}).
		Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}

// ---------------------------------------------------------------------------
// 镜像版本 Repository
// ---------------------------------------------------------------------------

// ImageVersionRepository 镜像版本数据访问接口
type ImageVersionRepository interface {
	Create(ctx context.Context, version *entity.ImageVersion) error
	GetByID(ctx context.Context, id int64) (*entity.ImageVersion, error)
	GetByImageAndVersion(ctx context.Context, imageID int64, version string) (*entity.ImageVersion, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByImageID(ctx context.Context, imageID int64) ([]*entity.ImageVersion, error)
	ListByImageIDs(ctx context.Context, imageIDs []int64) ([]*entity.ImageVersion, error)
	GetDefaultByImageID(ctx context.Context, imageID int64) (*entity.ImageVersion, error)
	ClearDefault(ctx context.Context, imageID int64) error
	SetDefault(ctx context.Context, id int64) error
	CountByImageID(ctx context.Context, imageID int64) (int64, error)
	IsVersionInUse(ctx context.Context, versionID int64) (bool, error)
}

// imageVersionRepository 镜像版本数据访问实现
type imageVersionRepository struct {
	db *gorm.DB
}

// NewImageVersionRepository 创建镜像版本数据访问实例
func NewImageVersionRepository(db *gorm.DB) ImageVersionRepository {
	return &imageVersionRepository{db: db}
}

// Create 创建镜像版本
func (r *imageVersionRepository) Create(ctx context.Context, version *entity.ImageVersion) error {
	if version.ID == 0 {
		version.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(version).Error
}

// GetByID 根据ID获取镜像版本
func (r *imageVersionRepository) GetByID(ctx context.Context, id int64) (*entity.ImageVersion, error) {
	var version entity.ImageVersion
	err := r.db.WithContext(ctx).First(&version, id).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// GetByImageAndVersion 根据镜像ID和版本号获取版本记录，供版本重复校验使用。
func (r *imageVersionRepository) GetByImageAndVersion(ctx context.Context, imageID int64, version string) (*entity.ImageVersion, error) {
	var imageVersion entity.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_id = ? AND version = ?", imageID, version).
		First(&imageVersion).Error
	if err != nil {
		return nil, err
	}
	return &imageVersion, nil
}

// UpdateFields 更新镜像版本指定字段
func (r *imageVersionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ImageVersion{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除镜像版本（硬删除）
func (r *imageVersionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ImageVersion{}, id).Error
}

// ListByImageID 获取镜像的所有版本
func (r *imageVersionRepository) ListByImageID(ctx context.Context, imageID int64) ([]*entity.ImageVersion, error) {
	var versions []*entity.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_id = ?", imageID).
		Order("is_default desc, created_at desc").
		Find(&versions).Error
	return versions, err
}

// ListByImageIDs 批量获取多个镜像的版本列表，供列表页组装版本信息时避免逐条查询。
func (r *imageVersionRepository) ListByImageIDs(ctx context.Context, imageIDs []int64) ([]*entity.ImageVersion, error) {
	if len(imageIDs) == 0 {
		return []*entity.ImageVersion{}, nil
	}
	var versions []*entity.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_id IN ?", imageIDs).
		Order("image_id asc, is_default desc, created_at desc").
		Find(&versions).Error
	return versions, err
}

// GetDefaultByImageID 获取镜像的默认版本
func (r *imageVersionRepository) GetDefaultByImageID(ctx context.Context, imageID int64) (*entity.ImageVersion, error) {
	var version entity.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_id = ? AND is_default = true", imageID).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// ClearDefault 清除镜像的默认版本标记
func (r *imageVersionRepository) ClearDefault(ctx context.Context, imageID int64) error {
	return r.db.WithContext(ctx).Model(&entity.ImageVersion{}).
		Where("image_id = ? AND is_default = true", imageID).
		Update("is_default", false).Error
}

// SetDefault 设置镜像版本为默认
func (r *imageVersionRepository) SetDefault(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.ImageVersion{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}

// CountByImageID 统计镜像版本数量
func (r *imageVersionRepository) CountByImageID(ctx context.Context, imageID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ImageVersion{}).
		Where("image_id = ?", imageID).
		Count(&count).Error
	return count, err
}

// IsVersionInUse 检查镜像版本是否被模板容器引用
func (r *imageVersionRepository) IsVersionInUse(ctx context.Context, versionID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TemplateContainer{}).
		Where("image_version_id = ?", versionID).
		Count(&count).Error
	return count > 0, err
}
