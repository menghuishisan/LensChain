// template_sub_repo.go
// 模块04 — 实验环境：模板子资源数据访问层
// 负责模板容器、检查点、初始化脚本、仿真场景配置、标签、角色的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 模板容器 Repository
// ---------------------------------------------------------------------------

// ContainerRepository 模板容器数据访问接口
type ContainerRepository interface {
	Create(ctx context.Context, container *entity.TemplateContainer) error
	GetByID(ctx context.Context, id int64) (*entity.TemplateContainer, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateContainer, error)
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	BatchCreate(ctx context.Context, containers []*entity.TemplateContainer) error
	CountByTemplateID(ctx context.Context, templateID int64) (int64, error)
}

// containerRepository 模板容器数据访问实现
type containerRepository struct {
	db *gorm.DB
}

// NewContainerRepository 创建模板容器数据访问实例
func NewContainerRepository(db *gorm.DB) ContainerRepository {
	return &containerRepository{db: db}
}

// Create 创建模板容器
func (r *containerRepository) Create(ctx context.Context, container *entity.TemplateContainer) error {
	if container.ID == 0 {
		container.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(container).Error
}

// GetByID 根据ID获取模板容器
func (r *containerRepository) GetByID(ctx context.Context, id int64) (*entity.TemplateContainer, error) {
	var container entity.TemplateContainer
	err := r.db.WithContext(ctx).First(&container, id).Error
	if err != nil {
		return nil, err
	}
	return &container, nil
}

// UpdateFields 更新模板容器指定字段
func (r *containerRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.TemplateContainer{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除模板容器
func (r *containerRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateContainer{}, id).Error
}

// ListByTemplateID 获取模板的所有容器配置
func (r *containerRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateContainer, error) {
	var containers []*entity.TemplateContainer
	err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("sort_order asc, startup_order asc").
		Find(&containers).Error
	return containers, err
}

// DeleteByTemplateID 删除模板的所有容器配置
func (r *containerRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateContainer{}).Error
}

// BatchCreate 批量创建模板容器
func (r *containerRepository) BatchCreate(ctx context.Context, containers []*entity.TemplateContainer) error {
	for i := range containers {
		if containers[i].ID == 0 {
			containers[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(containers, 50).Error
}

// CountByTemplateID 统计模板容器数量
func (r *containerRepository) CountByTemplateID(ctx context.Context, templateID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TemplateContainer{}).
		Where("template_id = ?", templateID).
		Count(&count).Error
	return count, err
}

// ---------------------------------------------------------------------------
// 检查点 Repository
// ---------------------------------------------------------------------------

// CheckpointRepository 检查点数据访问接口
type CheckpointRepository interface {
	Create(ctx context.Context, checkpoint *entity.TemplateCheckpoint) error
	GetByID(ctx context.Context, id int64) (*entity.TemplateCheckpoint, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateCheckpoint, error)
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	BatchCreate(ctx context.Context, checkpoints []*entity.TemplateCheckpoint) error
	SumScoreByTemplateID(ctx context.Context, templateID int64) (float64, error)
}

// checkpointRepository 检查点数据访问实现
type checkpointRepository struct {
	db *gorm.DB
}

// NewCheckpointRepository 创建检查点数据访问实例
func NewCheckpointRepository(db *gorm.DB) CheckpointRepository {
	return &checkpointRepository{db: db}
}

// Create 创建检查点
func (r *checkpointRepository) Create(ctx context.Context, checkpoint *entity.TemplateCheckpoint) error {
	if checkpoint.ID == 0 {
		checkpoint.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(checkpoint).Error
}

// GetByID 根据ID获取检查点
func (r *checkpointRepository) GetByID(ctx context.Context, id int64) (*entity.TemplateCheckpoint, error) {
	var checkpoint entity.TemplateCheckpoint
	err := r.db.WithContext(ctx).First(&checkpoint, id).Error
	if err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

// UpdateFields 更新检查点指定字段
func (r *checkpointRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.TemplateCheckpoint{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除检查点
func (r *checkpointRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateCheckpoint{}, id).Error
}

// ListByTemplateID 获取模板的所有检查点
func (r *checkpointRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateCheckpoint, error) {
	var checkpoints []*entity.TemplateCheckpoint
	err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("sort_order asc").
		Find(&checkpoints).Error
	return checkpoints, err
}

// DeleteByTemplateID 删除模板的所有检查点
func (r *checkpointRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateCheckpoint{}).Error
}

// BatchCreate 批量创建检查点
func (r *checkpointRepository) BatchCreate(ctx context.Context, checkpoints []*entity.TemplateCheckpoint) error {
	for i := range checkpoints {
		if checkpoints[i].ID == 0 {
			checkpoints[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(checkpoints, 50).Error
}

// SumScoreByTemplateID 统计模板检查点总分
func (r *checkpointRepository) SumScoreByTemplateID(ctx context.Context, templateID int64) (float64, error) {
	var sum float64
	err := r.db.WithContext(ctx).Model(&entity.TemplateCheckpoint{}).
		Where("template_id = ?", templateID).
		Select("COALESCE(SUM(score), 0)").
		Scan(&sum).Error
	return sum, err
}

// ---------------------------------------------------------------------------
// 初始化脚本 Repository
// ---------------------------------------------------------------------------

// InitScriptRepository 初始化脚本数据访问接口
type InitScriptRepository interface {
	Create(ctx context.Context, script *entity.TemplateInitScript) error
	GetByID(ctx context.Context, id int64) (*entity.TemplateInitScript, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateInitScript, error)
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	BatchCreate(ctx context.Context, scripts []*entity.TemplateInitScript) error
}

// initScriptRepository 初始化脚本数据访问实现
type initScriptRepository struct {
	db *gorm.DB
}

// NewInitScriptRepository 创建初始化脚本数据访问实例
func NewInitScriptRepository(db *gorm.DB) InitScriptRepository {
	return &initScriptRepository{db: db}
}

// Create 创建初始化脚本
func (r *initScriptRepository) Create(ctx context.Context, script *entity.TemplateInitScript) error {
	if script.ID == 0 {
		script.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(script).Error
}

// GetByID 根据ID获取初始化脚本
func (r *initScriptRepository) GetByID(ctx context.Context, id int64) (*entity.TemplateInitScript, error) {
	var script entity.TemplateInitScript
	err := r.db.WithContext(ctx).First(&script, id).Error
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// UpdateFields 更新初始化脚本指定字段
func (r *initScriptRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.TemplateInitScript{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除初始化脚本
func (r *initScriptRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateInitScript{}, id).Error
}

// ListByTemplateID 获取模板的所有初始化脚本
func (r *initScriptRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateInitScript, error) {
	var scripts []*entity.TemplateInitScript
	err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("execution_order asc").
		Find(&scripts).Error
	return scripts, err
}

// DeleteByTemplateID 删除模板的所有初始化脚本
func (r *initScriptRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateInitScript{}).Error
}

// BatchCreate 批量创建初始化脚本
func (r *initScriptRepository) BatchCreate(ctx context.Context, scripts []*entity.TemplateInitScript) error {
	for i := range scripts {
		if scripts[i].ID == 0 {
			scripts[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(scripts, 50).Error
}

// ---------------------------------------------------------------------------
// 模板仿真场景配置 Repository
// ---------------------------------------------------------------------------

// SimSceneRepository 模板仿真场景配置数据访问接口
type SimSceneRepository interface {
	Create(ctx context.Context, scene *entity.TemplateSimScene) error
	GetByID(ctx context.Context, id int64) (*entity.TemplateSimScene, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateSimScene, error)
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	BatchCreate(ctx context.Context, scenes []*entity.TemplateSimScene) error
}

// simSceneRepository 模板仿真场景配置数据访问实现
type simSceneRepository struct {
	db *gorm.DB
}

// NewSimSceneRepository 创建模板仿真场景配置数据访问实例
func NewSimSceneRepository(db *gorm.DB) SimSceneRepository {
	return &simSceneRepository{db: db}
}

// Create 创建模板仿真场景配置
func (r *simSceneRepository) Create(ctx context.Context, scene *entity.TemplateSimScene) error {
	if scene.ID == 0 {
		scene.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(scene).Error
}

// GetByID 根据ID获取模板仿真场景配置
func (r *simSceneRepository) GetByID(ctx context.Context, id int64) (*entity.TemplateSimScene, error) {
	var scene entity.TemplateSimScene
	err := r.db.WithContext(ctx).First(&scene, id).Error
	if err != nil {
		return nil, err
	}
	return &scene, nil
}

// UpdateFields 更新模板仿真场景配置指定字段
func (r *simSceneRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.TemplateSimScene{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除模板仿真场景配置
func (r *simSceneRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateSimScene{}, id).Error
}

// ListByTemplateID 获取模板的所有仿真场景配置
func (r *simSceneRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateSimScene, error) {
	var scenes []*entity.TemplateSimScene
	err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("sort_order asc").
		Find(&scenes).Error
	return scenes, err
}

// DeleteByTemplateID 删除模板的所有仿真场景配置
func (r *simSceneRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateSimScene{}).Error
}

// BatchCreate 批量创建模板仿真场景配置
func (r *simSceneRepository) BatchCreate(ctx context.Context, scenes []*entity.TemplateSimScene) error {
	for i := range scenes {
		if scenes[i].ID == 0 {
			scenes[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(scenes, 50).Error
}

// ---------------------------------------------------------------------------
// 标签 Repository
// ---------------------------------------------------------------------------

// TagRepository 标签数据访问接口
type TagRepository interface {
	Create(ctx context.Context, tag *entity.Tag) error
	GetByID(ctx context.Context, id int64) (*entity.Tag, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*entity.Tag, error)
	ListByCategory(ctx context.Context, category string) ([]*entity.Tag, error)
	GetByName(ctx context.Context, name string) (*entity.Tag, error)
	IsTagInUse(ctx context.Context, tagID int64) (bool, error)
}

// TemplateTagRepository 模板标签关联数据访问接口
type TemplateTagRepository interface {
	Create(ctx context.Context, tt *entity.TemplateTag) error
	Delete(ctx context.Context, id int64) error
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	DeleteByTemplateAndTag(ctx context.Context, templateID, tagID int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateTag, error)
	BatchCreate(ctx context.Context, tags []*entity.TemplateTag) error
}

// tagRepository 标签数据访问实现
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签数据访问实例
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// Create 创建标签
func (r *tagRepository) Create(ctx context.Context, tag *entity.Tag) error {
	if tag.ID == 0 {
		tag.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(tag).Error
}

// GetByID 根据ID获取标签
func (r *tagRepository) GetByID(ctx context.Context, id int64) (*entity.Tag, error) {
	var tag entity.Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// UpdateFields 更新标签指定字段
func (r *tagRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Tag{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除标签
func (r *tagRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Tag{}, id).Error
}

// ListAll 获取所有标签
func (r *tagRepository) ListAll(ctx context.Context) ([]*entity.Tag, error) {
	var tags []*entity.Tag
	err := r.db.WithContext(ctx).Order("category asc, name asc").Find(&tags).Error
	return tags, err
}

// ListByCategory 按分类获取标签
func (r *tagRepository) ListByCategory(ctx context.Context, category string) ([]*entity.Tag, error) {
	var tags []*entity.Tag
	err := r.db.WithContext(ctx).Where("category = ?", category).Order("name asc").Find(&tags).Error
	return tags, err
}

// GetByName 根据名称获取标签
func (r *tagRepository) GetByName(ctx context.Context, name string) (*entity.Tag, error) {
	var tag entity.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// IsTagInUse 检查标签是否被模板引用
func (r *tagRepository) IsTagInUse(ctx context.Context, tagID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TemplateTag{}).
		Where("tag_id = ?", tagID).
		Count(&count).Error
	return count > 0, err
}

// templateTagRepository 模板标签关联数据访问实现
type templateTagRepository struct {
	db *gorm.DB
}

// NewTemplateTagRepository 创建模板标签关联数据访问实例
func NewTemplateTagRepository(db *gorm.DB) TemplateTagRepository {
	return &templateTagRepository{db: db}
}

// Create 创建模板标签关联
func (r *templateTagRepository) Create(ctx context.Context, tt *entity.TemplateTag) error {
	if tt.ID == 0 {
		tt.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(tt).Error
}

// Delete 删除模板标签关联
func (r *templateTagRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateTag{}, id).Error
}

// DeleteByTemplateID 删除模板的所有标签关联
func (r *templateTagRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateTag{}).Error
}

// DeleteByTemplateAndTag 删除指定模板和标签的关联
func (r *templateTagRepository) DeleteByTemplateAndTag(ctx context.Context, templateID, tagID int64) error {
	return r.db.WithContext(ctx).
		Where("template_id = ? AND tag_id = ?", templateID, tagID).
		Delete(&entity.TemplateTag{}).Error
}

// ListByTemplateID 获取模板的所有标签关联
func (r *templateTagRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateTag, error) {
	var tags []*entity.TemplateTag
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&tags).Error
	return tags, err
}

// BatchCreate 批量创建模板标签关联
func (r *templateTagRepository) BatchCreate(ctx context.Context, tags []*entity.TemplateTag) error {
	for i := range tags {
		if tags[i].ID == 0 {
			tags[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(tags, 50).Error
}

// ---------------------------------------------------------------------------
// 角色 Repository
// ---------------------------------------------------------------------------

// RoleRepository 模板角色数据访问接口
type RoleRepository interface {
	Create(ctx context.Context, role *entity.TemplateRole) error
	GetByID(ctx context.Context, id int64) (*entity.TemplateRole, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateRole, error)
	DeleteByTemplateID(ctx context.Context, templateID int64) error
	BatchCreate(ctx context.Context, roles []*entity.TemplateRole) error
}

// roleRepository 模板角色数据访问实现
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建模板角色数据访问实例
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// Create 创建模板角色
func (r *roleRepository) Create(ctx context.Context, role *entity.TemplateRole) error {
	if role.ID == 0 {
		role.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(role).Error
}

// GetByID 根据ID获取模板角色
func (r *roleRepository) GetByID(ctx context.Context, id int64) (*entity.TemplateRole, error) {
	var role entity.TemplateRole
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// UpdateFields 更新模板角色指定字段
func (r *roleRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.TemplateRole{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除模板角色
func (r *roleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TemplateRole{}, id).Error
}

// ListByTemplateID 获取模板的所有角色
func (r *roleRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*entity.TemplateRole, error) {
	var roles []*entity.TemplateRole
	err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("sort_order asc").
		Find(&roles).Error
	return roles, err
}

// DeleteByTemplateID 删除模板的所有角色
func (r *roleRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&entity.TemplateRole{}).Error
}

// BatchCreate 批量创建模板角色
func (r *roleRepository) BatchCreate(ctx context.Context, roles []*entity.TemplateRole) error {
	for i := range roles {
		if roles[i].ID == 0 {
			roles[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(roles, 50).Error
}
