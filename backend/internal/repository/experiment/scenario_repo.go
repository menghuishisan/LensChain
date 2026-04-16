// scenario_repo.go
// 模块04 — 实验环境：仿真场景库数据访问层
// 负责仿真场景库、联动组的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 仿真场景库 Repository
// ---------------------------------------------------------------------------

// ScenarioRepository 仿真场景库数据访问接口
type ScenarioRepository interface {
	Create(ctx context.Context, scenario *entity.SimScenario) error
	GetByID(ctx context.Context, id int64) (*entity.SimScenario, error)
	GetByCode(ctx context.Context, code string) (*entity.SimScenario, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *ScenarioListParams) ([]*entity.SimScenario, int64, error)
	HasReferences(ctx context.Context, scenarioID int64) (bool, error)
}

// ScenarioListParams 仿真场景列表查询参数
type ScenarioListParams struct {
	Keyword         string
	Category        string
	SourceType      int
	Status          int
	TimeControlMode string
	DataSourceMode  int
	SortBy          string
	SortOrder       string
	Page            int
	PageSize        int
}

// scenarioRepository 仿真场景库数据访问实现
type scenarioRepository struct {
	db *gorm.DB
}

// NewScenarioRepository 创建仿真场景库数据访问实例
func NewScenarioRepository(db *gorm.DB) ScenarioRepository {
	return &scenarioRepository{db: db}
}

// Create 创建仿真场景
func (r *scenarioRepository) Create(ctx context.Context, scenario *entity.SimScenario) error {
	if scenario.ID == 0 {
		scenario.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(scenario).Error
}

// GetByID 根据ID获取仿真场景
func (r *scenarioRepository) GetByID(ctx context.Context, id int64) (*entity.SimScenario, error) {
	var scenario entity.SimScenario
	err := r.db.WithContext(ctx).First(&scenario, id).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

// GetByCode 根据编码获取仿真场景
func (r *scenarioRepository) GetByCode(ctx context.Context, code string) (*entity.SimScenario, error) {
	var scenario entity.SimScenario
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&scenario).Error
	if err != nil {
		return nil, err
	}
	return &scenario, nil
}

// UpdateFields 更新仿真场景指定字段
func (r *scenarioRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SimScenario{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除仿真场景
func (r *scenarioRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.SimScenario{}, id).Error
}

// List 仿真场景列表查询
func (r *scenarioRepository) List(ctx context.Context, params *ScenarioListParams) ([]*entity.SimScenario, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.SimScenario{})

	// 关键字搜索
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "code", "description"))
	}

	// 领域分类筛选
	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	// 来源类型筛选
	if params.SourceType > 0 {
		query = query.Where("source_type = ?", params.SourceType)
	}

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 时间控制模式筛选
	if params.TimeControlMode != "" {
		query = query.Where("time_control_mode = ?", params.TimeControlMode)
	}

	// 数据源模式筛选
	if params.DataSourceMode > 0 {
		query = query.Where("data_source_mode = ?", params.DataSourceMode)
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
		"created_at": "created_at",
		"name":       "name",
		"category":   "category",
		"status":     "status",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// 分页
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var scenarios []*entity.SimScenario
	if err := query.Find(&scenarios).Error; err != nil {
		return nil, 0, err
	}
	return scenarios, total, nil
}

// HasReferences 检查仿真场景是否被模板引用
func (r *scenarioRepository) HasReferences(ctx context.Context, scenarioID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TemplateSimScene{}).
		Where("scenario_id = ?", scenarioID).
		Count(&count).Error
	return count > 0, err
}

// ---------------------------------------------------------------------------
// 联动组 Repository
// ---------------------------------------------------------------------------

// LinkGroupRepository 联动组数据访问接口
type LinkGroupRepository interface {
	Create(ctx context.Context, group *entity.SimLinkGroup) error
	GetByID(ctx context.Context, id int64) (*entity.SimLinkGroup, error)
	GetByIDWithScenes(ctx context.Context, id int64) (*entity.SimLinkGroup, error)
	GetByCode(ctx context.Context, code string) (*entity.SimLinkGroup, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*entity.SimLinkGroup, error)
	ListByScenarioID(ctx context.Context, scenarioID int64) ([]*entity.SimLinkGroup, error)
}

// LinkGroupSceneRepository 联动组场景关联数据访问接口
type LinkGroupSceneRepository interface {
	Create(ctx context.Context, scene *entity.SimLinkGroupScene) error
	Delete(ctx context.Context, id int64) error
	DeleteByLinkGroupID(ctx context.Context, linkGroupID int64) error
	ListByLinkGroupID(ctx context.Context, linkGroupID int64) ([]*entity.SimLinkGroupScene, error)
	BatchCreate(ctx context.Context, scenes []*entity.SimLinkGroupScene) error
}

// linkGroupRepository 联动组数据访问实现
type linkGroupRepository struct {
	db *gorm.DB
}

// NewLinkGroupRepository 创建联动组数据访问实例
func NewLinkGroupRepository(db *gorm.DB) LinkGroupRepository {
	return &linkGroupRepository{db: db}
}

// Create 创建联动组
func (r *linkGroupRepository) Create(ctx context.Context, group *entity.SimLinkGroup) error {
	if group.ID == 0 {
		group.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(group).Error
}

// GetByID 根据ID获取联动组
func (r *linkGroupRepository) GetByID(ctx context.Context, id int64) (*entity.SimLinkGroup, error) {
	var group entity.SimLinkGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByIDWithScenes 根据ID获取联动组（含场景关联）
func (r *linkGroupRepository) GetByIDWithScenes(ctx context.Context, id int64) (*entity.SimLinkGroup, error) {
	var group entity.SimLinkGroup
	err := r.db.WithContext(ctx).
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByCode 根据编码获取联动组
func (r *linkGroupRepository) GetByCode(ctx context.Context, code string) (*entity.SimLinkGroup, error) {
	var group entity.SimLinkGroup
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateFields 更新联动组指定字段
func (r *linkGroupRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SimLinkGroup{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除联动组
func (r *linkGroupRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.SimLinkGroup{}, id).Error
}

// ListAll 获取所有联动组
func (r *linkGroupRepository) ListAll(ctx context.Context) ([]*entity.SimLinkGroup, error) {
	var groups []*entity.SimLinkGroup
	err := r.db.WithContext(ctx).
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Order("created_at desc").
		Find(&groups).Error
	return groups, err
}

// ListByScenarioID 根据场景ID查询所属联动组
func (r *linkGroupRepository) ListByScenarioID(ctx context.Context, scenarioID int64) ([]*entity.SimLinkGroup, error) {
	var groups []*entity.SimLinkGroup
	err := r.db.WithContext(ctx).
		Where("id IN (?)",
			r.db.Model(&entity.SimLinkGroupScene{}).
				Select("link_group_id").
				Where("scenario_id = ?", scenarioID),
		).
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Find(&groups).Error
	return groups, err
}

// linkGroupSceneRepository 联动组场景关联数据访问实现
type linkGroupSceneRepository struct {
	db *gorm.DB
}

// NewLinkGroupSceneRepository 创建联动组场景关联数据访问实例
func NewLinkGroupSceneRepository(db *gorm.DB) LinkGroupSceneRepository {
	return &linkGroupSceneRepository{db: db}
}

// Create 创建联动组场景关联
func (r *linkGroupSceneRepository) Create(ctx context.Context, scene *entity.SimLinkGroupScene) error {
	if scene.ID == 0 {
		scene.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(scene).Error
}

// Delete 删除联动组场景关联
func (r *linkGroupSceneRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.SimLinkGroupScene{}, id).Error
}

// DeleteByLinkGroupID 删除联动组的所有场景关联
func (r *linkGroupSceneRepository) DeleteByLinkGroupID(ctx context.Context, linkGroupID int64) error {
	return r.db.WithContext(ctx).Where("link_group_id = ?", linkGroupID).Delete(&entity.SimLinkGroupScene{}).Error
}

// ListByLinkGroupID 获取联动组的所有场景关联
func (r *linkGroupSceneRepository) ListByLinkGroupID(ctx context.Context, linkGroupID int64) ([]*entity.SimLinkGroupScene, error) {
	var scenes []*entity.SimLinkGroupScene
	err := r.db.WithContext(ctx).
		Where("link_group_id = ?", linkGroupID).
		Order("sort_order asc").
		Find(&scenes).Error
	return scenes, err
}

// BatchCreate 批量创建联动组场景关联
func (r *linkGroupSceneRepository) BatchCreate(ctx context.Context, scenes []*entity.SimLinkGroupScene) error {
	for i := range scenes {
		if scenes[i].ID == 0 {
			scenes[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(scenes, 50).Error
}
