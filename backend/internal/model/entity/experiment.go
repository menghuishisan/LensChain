// experiment.go
// 模块04 — 实验环境：数据库实体结构体（模板与配置部分）
// 对照 docs/modules/04-实验环境/02-数据库设计.md
// 包含 14 张表的 GORM 映射结构体：镜像、模板、仿真场景、标签、角色相关

package entity

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 2.1 镜像分类
// ---------------------------------------------------------------------------

// ImageCategory 镜像分类表
// 对应 image_categories 表，7 个字段
type ImageCategory struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name        string    `gorm:"type:varchar(50);not null" json:"name"`
	Code        string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (ImageCategory) TableName() string {
	return "image_categories"
}

// ---------------------------------------------------------------------------
// 2.2 镜像主表
// ---------------------------------------------------------------------------

// Image 镜像主表
// 对应 images 表，25 个字段（含软删除）
type Image struct {
	ID                     int64            `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CategoryID             int64            `gorm:"not null;index" json:"category_id,string"`
	Name                   string           `gorm:"type:varchar(100);not null" json:"name"`
	DisplayName            string           `gorm:"type:varchar(100);not null" json:"display_name"`
	Description            *string          `gorm:"type:text" json:"description,omitempty"`
	IconURL                *string          `gorm:"type:varchar(500)" json:"icon_url,omitempty"`
	Ecosystem              *string          `gorm:"type:varchar(50)" json:"ecosystem,omitempty"`
	SourceType             int              `gorm:"type:smallint;not null;default:1" json:"source_type"`
	UploadedBy             *int64           `gorm:"" json:"uploaded_by,omitempty,string"`
	SchoolID               *int64           `gorm:"index" json:"school_id,omitempty,string"`
	Status                 int              `gorm:"type:smallint;not null;default:1" json:"status"`
	ReviewComment          *string          `gorm:"type:varchar(500)" json:"review_comment,omitempty"`
	ReviewedBy             *int64           `gorm:"" json:"reviewed_by,omitempty,string"`
	ReviewedAt             *time.Time       `gorm:"" json:"reviewed_at,omitempty"`
	DefaultPorts           json.RawMessage  `gorm:"type:jsonb" json:"default_ports,omitempty"`
	DefaultEnvVars         json.RawMessage  `gorm:"type:jsonb" json:"default_env_vars,omitempty"`
	DefaultVolumes         json.RawMessage  `gorm:"type:jsonb" json:"default_volumes,omitempty"`
	TypicalCompanions      json.RawMessage  `gorm:"type:jsonb" json:"typical_companions,omitempty"`
	RequiredDependencies   json.RawMessage  `gorm:"type:jsonb" json:"required_dependencies,omitempty"`
	ResourceRecommendation json.RawMessage  `gorm:"type:jsonb" json:"resource_recommendation,omitempty"`
	DocumentationURL       *string          `gorm:"type:varchar(500)" json:"documentation_url,omitempty"`
	UsageCount             int              `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt              time.Time        `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt              time.Time        `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt              gorm.DeletedAt   `gorm:"index" json:"-"`

	// 关联（非数据库字段，用于预加载）
	Versions []ImageVersion `gorm:"foreignKey:ImageID" json:"versions,omitempty"`
}

// TableName 指定表名
func (Image) TableName() string {
	return "images"
}

// ---------------------------------------------------------------------------
// 2.3 镜像版本
// ---------------------------------------------------------------------------

// ImageVersion 镜像版本表
// 对应 image_versions 表，15 个字段
type ImageVersion struct {
	ID          int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ImageID     int64           `gorm:"not null;index" json:"image_id,string"`
	Version     string          `gorm:"type:varchar(50);not null" json:"version"`
	RegistryURL string          `gorm:"type:varchar(500);not null" json:"registry_url"`
	ImageSize   *int64          `gorm:"" json:"image_size,omitempty,string"`
	Digest      *string         `gorm:"type:varchar(200)" json:"digest,omitempty"`
	MinCPU      *string         `gorm:"type:varchar(20)" json:"min_cpu,omitempty"`
	MinMemory   *string         `gorm:"type:varchar(20)" json:"min_memory,omitempty"`
	MinDisk     *string         `gorm:"type:varchar(20)" json:"min_disk,omitempty"`
	IsDefault   bool            `gorm:"not null;default:false" json:"is_default"`
	Status      int             `gorm:"type:smallint;not null;default:1" json:"status"`
	ScanResult  json.RawMessage `gorm:"type:jsonb" json:"scan_result,omitempty"`
	ScannedAt   *time.Time      `gorm:"" json:"scanned_at,omitempty"`
	CreatedAt   time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (ImageVersion) TableName() string {
	return "image_versions"
}

// ---------------------------------------------------------------------------
// 2.4 实验模板主表
// ---------------------------------------------------------------------------

// ExperimentTemplate 实验模板主表
// 对应 experiment_templates 表，28 个字段（含软删除）
type ExperimentTemplate struct {
	ID            int64            `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID      int64            `gorm:"not null;index" json:"school_id,string"`
	TeacherID     int64            `gorm:"not null;index" json:"teacher_id,string"`
	Title         string           `gorm:"type:varchar(200);not null" json:"title"`
	Description   *string          `gorm:"type:text" json:"description,omitempty"`
	Objectives    *string          `gorm:"type:text" json:"objectives,omitempty"`
	Instructions  *string          `gorm:"type:text" json:"instructions,omitempty"`
	References    *string          `gorm:"column:references;type:text" json:"references,omitempty"`
	ExpType       int              `gorm:"column:experiment_type;type:smallint;not null;default:2" json:"experiment_type"`
	TopologyMode  *int             `gorm:"type:smallint" json:"topology_mode,omitempty"`
	JudgeMode     int              `gorm:"type:smallint;not null;default:1" json:"judge_mode"`
	AutoWeight    *float64         `gorm:"type:decimal(5,2)" json:"auto_weight,omitempty"`
	ManualWeight  *float64         `gorm:"type:decimal(5,2)" json:"manual_weight,omitempty"`
	TotalScore    float64          `gorm:"type:decimal(6,2);not null;default:100" json:"total_score"`
	MaxDuration   *int             `gorm:"" json:"max_duration,omitempty"`
	IdleTimeout   int              `gorm:"not null;default:30" json:"idle_timeout"`
	CPULimit      *string          `gorm:"type:varchar(20)" json:"cpu_limit,omitempty"`
	MemoryLimit   *string          `gorm:"type:varchar(20)" json:"memory_limit,omitempty"`
	DiskLimit     *string          `gorm:"type:varchar(20)" json:"disk_limit,omitempty"`
	ScoreStrategy int              `gorm:"type:smallint;not null;default:1" json:"score_strategy"`
	IsShared      bool             `gorm:"not null;default:false" json:"is_shared"`
	ClonedFromID  *int64           `gorm:"" json:"cloned_from_id,omitempty,string"`
	Status        int              `gorm:"type:smallint;not null;default:1" json:"status"`
	SimLayout     json.RawMessage  `gorm:"type:jsonb" json:"sim_layout,omitempty"`
	K8sConfig     json.RawMessage  `gorm:"type:jsonb" json:"k8s_config,omitempty"`
	NetworkConfig json.RawMessage  `gorm:"type:jsonb" json:"network_config,omitempty"`
	CreatedAt     time.Time        `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt   `gorm:"index" json:"-"`

	// 关联（非数据库字段，用于预加载）
	Containers  []TemplateContainer  `gorm:"foreignKey:TemplateID" json:"containers,omitempty"`
	Checkpoints []TemplateCheckpoint `gorm:"foreignKey:TemplateID" json:"checkpoints,omitempty"`
	InitScripts []TemplateInitScript `gorm:"foreignKey:TemplateID" json:"init_scripts,omitempty"`
	SimScenes   []TemplateSimScene   `gorm:"foreignKey:TemplateID" json:"sim_scenes,omitempty"`
	Tags        []TemplateTag        `gorm:"foreignKey:TemplateID" json:"tags,omitempty"`
	Roles       []TemplateRole       `gorm:"foreignKey:TemplateID" json:"roles,omitempty"`
}

// TableName 指定表名
func (ExperimentTemplate) TableName() string {
	return "experiment_templates"
}

// ---------------------------------------------------------------------------
// 2.5 模板容器配置
// ---------------------------------------------------------------------------

// TemplateContainer 模板容器配置表
// 对应 template_containers 表，16 个字段
type TemplateContainer struct {
	ID             int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID     int64           `gorm:"not null;index" json:"template_id,string"`
	ImageVersionID int64           `gorm:"not null;index" json:"image_version_id,string"`
	ContainerName  string          `gorm:"type:varchar(100);not null" json:"container_name"`
	RoleID         *int64          `gorm:"" json:"role_id,omitempty,string"`
	EnvVars        json.RawMessage `gorm:"type:jsonb" json:"env_vars,omitempty"`
	Ports          json.RawMessage `gorm:"type:jsonb" json:"ports,omitempty"`
	Volumes        json.RawMessage `gorm:"type:jsonb" json:"volumes,omitempty"`
	CPULimit       *string         `gorm:"type:varchar(20)" json:"cpu_limit,omitempty"`
	MemoryLimit    *string         `gorm:"type:varchar(20)" json:"memory_limit,omitempty"`
	DependsOn      json.RawMessage `gorm:"type:jsonb" json:"depends_on,omitempty"`
	StartupOrder   int             `gorm:"not null;default:0" json:"startup_order"`
	IsPrimary      bool            `gorm:"not null;default:false" json:"is_primary"`
	SortOrder      int             `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt      time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (TemplateContainer) TableName() string {
	return "template_containers"
}

// ---------------------------------------------------------------------------
// 2.6 检查点定义
// ---------------------------------------------------------------------------

// TemplateCheckpoint 检查点定义表
// 对应 template_checkpoints 表，14 个字段
type TemplateCheckpoint struct {
	ID              int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID      int64           `gorm:"not null;index" json:"template_id,string"`
	Title           string          `gorm:"type:varchar(200);not null" json:"title"`
	Description     *string         `gorm:"type:text" json:"description,omitempty"`
	CheckType       int             `gorm:"type:smallint;not null" json:"check_type"`
	ScriptContent   *string         `gorm:"type:text" json:"script_content,omitempty"`
	ScriptLanguage  *string         `gorm:"type:varchar(20)" json:"script_language,omitempty"`
	TargetContainer *string         `gorm:"type:varchar(100)" json:"target_container,omitempty"`
	AssertionConfig json.RawMessage `gorm:"type:jsonb" json:"assertion_config,omitempty"`
	Score           float64         `gorm:"type:decimal(6,2);not null" json:"score"`
	Scope           int             `gorm:"type:smallint;not null;default:1" json:"scope"`
	SortOrder       int             `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (TemplateCheckpoint) TableName() string {
	return "template_checkpoints"
}

// ---------------------------------------------------------------------------
// 2.7 初始化脚本
// ---------------------------------------------------------------------------

// TemplateInitScript 初始化脚本表
// 对应 template_init_scripts 表，9 个字段
type TemplateInitScript struct {
	ID              int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID      int64     `gorm:"not null;index" json:"template_id,string"`
	TargetContainer string    `gorm:"type:varchar(100);not null" json:"target_container"`
	ScriptContent   string    `gorm:"type:text;not null" json:"script_content"`
	ScriptLanguage  string    `gorm:"type:varchar(20);not null;default:'bash'" json:"script_language"`
	ExecutionOrder  int       `gorm:"not null;default:0" json:"execution_order"`
	Timeout         int       `gorm:"not null;default:300" json:"timeout"`
	CreatedAt       time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (TemplateInitScript) TableName() string {
	return "template_init_scripts"
}

// ---------------------------------------------------------------------------
// 2.8 仿真场景库
// ---------------------------------------------------------------------------

// SimScenario 仿真场景库表
// 对应 sim_scenarios 表，27 个字段（含软删除）
type SimScenario struct {
	ID                  int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name                string          `gorm:"type:varchar(100);not null" json:"name"`
	Code                string          `gorm:"type:varchar(100);not null" json:"code"`
	Category            string          `gorm:"type:varchar(50);not null" json:"category"`
	Description         *string         `gorm:"type:text" json:"description,omitempty"`
	IconURL             *string         `gorm:"type:varchar(500)" json:"icon_url,omitempty"`
	ThumbnailURL        *string         `gorm:"type:varchar(500)" json:"thumbnail_url,omitempty"`
	SourceType          int             `gorm:"type:smallint;not null;default:1" json:"source_type"`
	UploadedBy          *int64          `gorm:"" json:"uploaded_by,omitempty,string"`
	SchoolID            *int64          `gorm:"" json:"school_id,omitempty,string"`
	Status              int             `gorm:"type:smallint;not null;default:1" json:"status"`
	ReviewComment       *string         `gorm:"type:varchar(500)" json:"review_comment,omitempty"`
	ReviewedBy          *int64          `gorm:"" json:"reviewed_by,omitempty,string"`
	ReviewedAt          *time.Time      `gorm:"" json:"reviewed_at,omitempty"`
	AlgorithmType       string          `gorm:"type:varchar(100);not null" json:"algorithm_type"`
	TimeControlMode     string          `gorm:"type:varchar(20);not null;default:'process'" json:"time_control_mode"`
	ContainerImageURL   *string         `gorm:"type:varchar(500)" json:"container_image_url,omitempty"`
	ContainerImageSize  *int64          `gorm:"" json:"container_image_size,omitempty,string"`
	DefaultParams       json.RawMessage `gorm:"type:jsonb" json:"default_params,omitempty"`
	InteractionSchema   json.RawMessage `gorm:"type:jsonb" json:"interaction_schema,omitempty"`
	DataSourceMode      int             `gorm:"type:smallint;not null;default:1" json:"data_source_mode"`
	DefaultSize         json.RawMessage `gorm:"type:jsonb" json:"default_size,omitempty"`
	DeliveryPhase       int             `gorm:"type:smallint;not null;default:1" json:"delivery_phase"`
	Version             string          `gorm:"type:varchar(50);not null;default:'1.0.0'" json:"version"`
	CreatedAt           time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time       `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `gorm:"index" json:"-"`
}

// TableName 指定表名
func (SimScenario) TableName() string {
	return "sim_scenarios"
}

// ---------------------------------------------------------------------------
// 2.9 联动组定义
// ---------------------------------------------------------------------------

// SimLinkGroup 联动组定义表
// 对应 sim_link_groups 表，7 个字段
type SimLinkGroup struct {
	ID                int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name              string          `gorm:"type:varchar(100);not null" json:"name"`
	Code              string          `gorm:"type:varchar(100);not null" json:"code"`
	Description       *string         `gorm:"type:text" json:"description,omitempty"`
	SharedStateSchema json.RawMessage `gorm:"type:jsonb;not null" json:"shared_state_schema"`
	CreatedAt         time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"not null;default:now()" json:"updated_at"`

	// 关联（非数据库字段，用于预加载）
	Scenes []SimLinkGroupScene `gorm:"foreignKey:LinkGroupID" json:"scenes,omitempty"`
}

// TableName 指定表名
func (SimLinkGroup) TableName() string {
	return "sim_link_groups"
}

// ---------------------------------------------------------------------------
// 2.10 联动组场景关联
// ---------------------------------------------------------------------------

// SimLinkGroupScene 联动组场景关联表
// 对应 sim_link_group_scenes 表，6 个字段（只有 created_at，无 updated_at）
type SimLinkGroupScene struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	LinkGroupID int64     `gorm:"not null;index" json:"link_group_id,string"`
	ScenarioID  int64     `gorm:"not null;index" json:"scenario_id,string"`
	RoleInGroup *string   `gorm:"type:varchar(50)" json:"role_in_group,omitempty"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (SimLinkGroupScene) TableName() string {
	return "sim_link_group_scenes"
}

// ---------------------------------------------------------------------------
// 2.11 模板仿真场景配置
// ---------------------------------------------------------------------------

// TemplateSimScene 模板仿真场景配置表
// 对应 template_sim_scenes 表，9 个字段
type TemplateSimScene struct {
	ID               int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID       int64           `gorm:"not null;index" json:"template_id,string"`
	ScenarioID       int64           `gorm:"not null;index" json:"scenario_id,string"`
	LinkGroupID      *int64          `gorm:"index" json:"link_group_id,omitempty,string"`
	Config           json.RawMessage `gorm:"type:jsonb" json:"config,omitempty"`
	LayoutPosition   json.RawMessage `gorm:"type:jsonb" json:"layout_position,omitempty"`
	DataSourceConfig json.RawMessage `gorm:"type:jsonb" json:"data_source_config,omitempty"`
	SortOrder        int             `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt        time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (TemplateSimScene) TableName() string {
	return "template_sim_scenes"
}

// ---------------------------------------------------------------------------
// 2.12 标签
// ---------------------------------------------------------------------------

// Tag 标签表
// 对应 tags 表，6 个字段
type Tag struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name      string    `gorm:"type:varchar(50);not null" json:"name"`
	Category  string    `gorm:"type:varchar(20);not null" json:"category"`
	IsSystem  bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// ---------------------------------------------------------------------------
// 2.13 模板标签关联
// ---------------------------------------------------------------------------

// TemplateTag 模板标签关联表
// 对应 template_tags 表，4 个字段（只有 created_at，无 updated_at）
type TemplateTag struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID int64     `gorm:"not null;index" json:"template_id,string"`
	TagID      int64     `gorm:"not null;index" json:"tag_id,string"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (TemplateTag) TableName() string {
	return "template_tags"
}

// ---------------------------------------------------------------------------
// 2.14 多人实验角色定义
// ---------------------------------------------------------------------------

// TemplateRole 多人实验角色定义表
// 对应 template_roles 表，8 个字段
type TemplateRole struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID  int64     `gorm:"not null;index" json:"template_id,string"`
	RoleName    string    `gorm:"type:varchar(50);not null" json:"role_name"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	MaxMembers  int       `gorm:"not null;default:1" json:"max_members"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (TemplateRole) TableName() string {
	return "template_roles"
}
