// template_repo.go
// 模块04 — 实验环境：实验模板数据访问层
// 负责实验模板主表的 CRUD 操作
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

// TemplateRepository 实验模板数据访问接口
type TemplateRepository interface {
	Create(ctx context.Context, template *entity.ExperimentTemplate) error
	GetByID(ctx context.Context, id int64) (*entity.ExperimentTemplate, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *TemplateListParams) ([]*entity.ExperimentTemplate, int64, error)
	ListShared(ctx context.Context, params *SharedTemplateListParams) ([]*entity.ExperimentTemplate, int64, error)
	CountByTeacherID(ctx context.Context, teacherID int64) (int64, error)
	HasInstances(ctx context.Context, templateID int64) (bool, error)
	HasCourseReferences(ctx context.Context, templateID int64) (bool, error)
}

// TemplateListParams 模板列表查询参数
type TemplateListParams struct {
	SchoolID       int64
	TeacherID      int64
	Keyword        string
	Status         int16
	ExperimentType int16
	Ecosystem      string
	TagID          int64
	SortBy         string
	SortOrder      string
	Page           int
	PageSize       int
}

// SharedTemplateListParams 共享模板列表查询参数
type SharedTemplateListParams struct {
	Keyword   string
	Ecosystem string
	TagID     int64
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

// templateRepository 实验模板数据访问实现
type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建实验模板数据访问实例
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

// Create 创建实验模板
func (r *templateRepository) Create(ctx context.Context, template *entity.ExperimentTemplate) error {
	if template.ID == 0 {
		template.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(template).Error
}

// GetByID 根据ID获取实验模板（不含关联）
func (r *templateRepository) GetByID(ctx context.Context, id int64) (*entity.ExperimentTemplate, error) {
	var template entity.ExperimentTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateFields 更新实验模板指定字段
func (r *templateRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ExperimentTemplate{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除实验模板
func (r *templateRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ExperimentTemplate{}, id).Error
}

// List 教师模板列表查询
func (r *templateRepository) List(ctx context.Context, params *TemplateListParams) ([]*entity.ExperimentTemplate, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentTemplate{})

	// 多租户隔离
	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}

	// 教师筛选
	if params.TeacherID > 0 {
		query = query.Where("teacher_id = ?", params.TeacherID)
	}

	// 关键字搜索
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title", "description"))
	}

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 实验类型筛选
	if params.ExperimentType > 0 {
		query = query.Where("experiment_type = ?", params.ExperimentType)
	}

	// 生态筛选（根据模板容器引用的镜像生态过滤）
	if params.Ecosystem != "" {
		query = query.Where("id IN (?)",
			r.db.Table("template_containers").
				Select("DISTINCT template_containers.template_id").
				Joins("JOIN image_versions ON image_versions.id = template_containers.image_version_id").
				Joins("JOIN images ON images.id = image_versions.image_id").
				Where("images.ecosystem = ?", params.Ecosystem),
		)
	}

	// 标签筛选（通过子查询）
	if params.TagID > 0 {
		query = query.Where("id IN (?)",
			r.db.Model(&entity.TemplateTag{}).
				Select("template_id").
				Where("tag_id = ?", params.TagID),
		)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"title":      "title",
		"status":     "status",
		"updated_at": "updated_at",
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

	var templates []*entity.ExperimentTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

// ListShared 共享模板列表查询
func (r *templateRepository) ListShared(ctx context.Context, params *SharedTemplateListParams) ([]*entity.ExperimentTemplate, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentTemplate{}).
		Where("is_shared = ? AND status = ?", true, enum.TemplateStatusPublished)

	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title", "description"))
	}

	if params.Ecosystem != "" {
		query = query.Where("id IN (?)",
			r.db.Table("template_containers").
				Select("DISTINCT template_containers.template_id").
				Joins("JOIN image_versions ON image_versions.id = template_containers.image_version_id").
				Joins("JOIN images ON images.id = image_versions.image_id").
				Where("images.ecosystem = ?", params.Ecosystem),
		)
	}

	if params.TagID > 0 {
		query = query.Where("id IN (?)",
			r.db.Model(&entity.TemplateTag{}).
				Select("template_id").
				Where("tag_id = ?", params.TagID),
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"title":      "title",
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

	var templates []*entity.ExperimentTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

// CountByTeacherID 统计教师创建的模板数量
func (r *templateRepository) CountByTeacherID(ctx context.Context, teacherID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentTemplate{}).
		Where("teacher_id = ?", teacherID).
		Count(&count).Error
	return count, err
}

// HasInstances 检查模板是否有关联的实验实例
func (r *templateRepository) HasInstances(ctx context.Context, templateID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("template_id = ?", templateID).
		Count(&count).Error
	return count > 0, err
}

// HasCourseReferences 检查模板是否被课时或课程独立实验引用。
// 模板编辑/删除规则以“是否被课程侧引用”为准，不能只检查运行实例。
func (r *templateRepository) HasCourseReferences(ctx context.Context, templateID int64) (bool, error) {
	var lessonCount int64
	if err := r.db.WithContext(ctx).
		Table("lessons").
		Where("experiment_id = ? AND deleted_at IS NULL", templateID).
		Count(&lessonCount).Error; err != nil {
		return false, err
	}
	if lessonCount > 0 {
		return true, nil
	}

	var courseExperimentCount int64
	err := r.db.WithContext(ctx).
		Table("course_experiments").
		Where("experiment_id = ?", templateID).
		Count(&courseExperimentCount).Error
	return courseExperimentCount > 0, err
}
