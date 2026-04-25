// experiment.go
// 模块04 — 实验环境：请求/响应 DTO 定义（镜像管理 + 实验模板 + 仿真场景 + 标签 + 角色）
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package dto

import "encoding/json"

// ========== 镜像管理 DTO ==========

// CreateImageReq 创建/上传镜像请求
// POST /api/v1/images
type CreateImageReq struct {
	CategoryID             string                        `json:"category_id" binding:"required"`
	Name                   string                        `json:"name" binding:"required,max=100"`
	DisplayName            string                        `json:"display_name" binding:"required,max=100"`
	Description            *string                       `json:"description"`
	IconURL                *string                       `json:"icon_url" binding:"omitempty,url,max=500"`
	Ecosystem              *string                       `json:"ecosystem" binding:"omitempty,max=50"`
	DefaultPorts           []ImagePortItem               `json:"default_ports"`
	DefaultEnvVars         []ImageEnvVarItem             `json:"default_env_vars"`
	DefaultVolumes         []ImageVolumeItem             `json:"default_volumes"`
	TypicalCompanions      ImageTypicalCompanions        `json:"typical_companions"`
	RequiredDependencies   []ImageDependencyItem         `json:"required_dependencies"`
	ResourceRecommendation ImageResourceRecommendation   `json:"resource_recommendation"`
	DocumentationURL       *string                       `json:"documentation_url" binding:"omitempty,max=500"`
	Versions               []CreateImageVersionInlineReq `json:"versions" binding:"required,min=1,dive"`
}

// CreateImageVersionInlineReq 创建镜像时内联的版本信息
type CreateImageVersionInlineReq struct {
	Version     string  `json:"version" binding:"required,max=50"`
	RegistryURL string  `json:"registry_url" binding:"required,max=500"`
	MinCPU      *string `json:"min_cpu"`
	MinMemory   *string `json:"min_memory"`
	MinDisk     *string `json:"min_disk"`
	IsDefault   bool    `json:"is_default"`
}

// UpdateImageReq 编辑镜像信息请求
// PUT /api/v1/images/:id
type UpdateImageReq struct {
	DisplayName            *string                      `json:"display_name" binding:"omitempty,max=100"`
	Description            *string                      `json:"description"`
	IconURL                *string                      `json:"icon_url" binding:"omitempty,url,max=500"`
	Ecosystem              *string                      `json:"ecosystem" binding:"omitempty,max=50"`
	DefaultPorts           []ImagePortItem              `json:"default_ports"`
	DefaultEnvVars         []ImageEnvVarItem            `json:"default_env_vars"`
	DefaultVolumes         []ImageVolumeItem            `json:"default_volumes"`
	TypicalCompanions      *ImageTypicalCompanions      `json:"typical_companions"`
	RequiredDependencies   []ImageDependencyItem        `json:"required_dependencies"`
	ResourceRecommendation *ImageResourceRecommendation `json:"resource_recommendation"`
	DocumentationURL       *string                      `json:"documentation_url" binding:"omitempty,max=500"`
}

// ReviewImageReq 审核镜像请求
// POST /api/v1/images/:id/review
type ReviewImageReq struct {
	Action  string  `json:"action" binding:"required,oneof=approve reject"`
	Comment *string `json:"comment" binding:"omitempty,max=500"`
}

// ImageListReq 镜像列表查询参数
// GET /api/v1/images
type ImageListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword    string `form:"keyword"`
	CategoryID string `form:"category_id"`
	Ecosystem  string `form:"ecosystem"`
	SourceType int16  `form:"source_type" binding:"omitempty,oneof=1 2"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ImageResp 镜像详情响应
type ImageResp struct {
	ID                     string                      `json:"id"`
	CategoryID             string                      `json:"category_id"`
	CategoryName           string                      `json:"category_name"`
	Name                   string                      `json:"name"`
	DisplayName            string                      `json:"display_name"`
	Description            *string                     `json:"description"`
	IconURL                *string                     `json:"icon_url"`
	Ecosystem              *string                     `json:"ecosystem"`
	SourceType             int16                       `json:"source_type"`
	SourceTypeText         string                      `json:"source_type_text"`
	UploadedBy             *string                     `json:"uploaded_by"`
	UploaderName           *string                     `json:"uploader_name"`
	Status                 int16                       `json:"status"`
	StatusText             string                      `json:"status_text"`
	ReviewComment          *string                     `json:"review_comment"`
	DefaultPorts           []ImagePortItem             `json:"default_ports"`
	DefaultEnvVars         []ImageEnvVarItem           `json:"default_env_vars"`
	DefaultVolumes         []ImageVolumeItem           `json:"default_volumes"`
	TypicalCompanions      ImageTypicalCompanions      `json:"typical_companions"`
	RequiredDependencies   []ImageDependencyItem       `json:"required_dependencies"`
	ResourceRecommendation ImageResourceRecommendation `json:"resource_recommendation"`
	DocumentationURL       *string                     `json:"documentation_url"`
	UsageCount             int                         `json:"usage_count"`
	Versions               []ImageVersionResp          `json:"versions"`
	CreatedAt              string                      `json:"created_at"`
	UpdatedAt              string                      `json:"updated_at"`
}

// ImagePortItem 镜像默认端口项
type ImagePortItem struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
}

// ImageEnvVarItem 镜像默认环境变量项。
// 条件变量示例由 Conditions 描述，普通变量该字段为空。
type ImageEnvVarItem struct {
	Key        string                 `json:"key"`
	Value      string                 `json:"value"`
	Desc       *string                `json:"desc,omitempty"`
	Conditions []ImageEnvVarCondition `json:"conditions"`
}

// ImageEnvVarCondition 条件环境变量规则
type ImageEnvVarCondition struct {
	When       string           `json:"when"`
	Value      string           `json:"value"`
	InjectVars []ImageInjectVar `json:"inject_vars"`
}

// ImageInjectVar 条件命中后附加注入的环境变量
type ImageInjectVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ImageVolumeItem 镜像默认数据卷项
type ImageVolumeItem struct {
	Path string  `json:"path"`
	Desc *string `json:"desc,omitempty"`
}

// ImageTypicalCompanions 镜像典型搭配建议
type ImageTypicalCompanions struct {
	Required    []ImageDependencyItem `json:"required"`
	Recommended []ImageDependencyItem `json:"recommended"`
	Optional    []ImageDependencyItem `json:"optional"`
}

// ImageDependencyItem 镜像依赖或搭配项
type ImageDependencyItem struct {
	Image  string `json:"image"`
	Reason string `json:"reason"`
}

// ImageResourceRecommendation 镜像资源建议
type ImageResourceRecommendation struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}

// ImageListItem 镜像列表项
type ImageListItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	DisplayName    string  `json:"display_name"`
	IconURL        *string `json:"icon_url"`
	Ecosystem      *string `json:"ecosystem"`
	CategoryName   string  `json:"category_name"`
	SourceType     int16   `json:"source_type"`
	SourceTypeText string  `json:"source_type_text"`
	Status         int16   `json:"status"`
	StatusText     string  `json:"status_text"`
	VersionCount   int     `json:"version_count"`
	UsageCount     int     `json:"usage_count"`
	CreatedAt      string  `json:"created_at"`
}

// CreateImageResp 创建镜像响应
type CreateImageResp struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Status      int16               `json:"status"`
	StatusText  string              `json:"status_text"`
	Versions    []ImageVersionBrief `json:"versions"`
}

// ImageVersionBrief 镜像版本简要信息（创建响应用）
type ImageVersionBrief struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	IsDefault bool   `json:"is_default"`
}

// ========== 镜像版本 DTO ==========

// CreateImageVersionReq 添加镜像版本请求
// POST /api/v1/images/:id/versions
type CreateImageVersionReq struct {
	Version     string  `json:"version" binding:"required,max=50"`
	RegistryURL string  `json:"registry_url" binding:"required,max=500"`
	ImageSize   *int64  `json:"image_size"`
	Digest      *string `json:"digest" binding:"omitempty,max=200"`
	MinCPU      *string `json:"min_cpu"`
	MinMemory   *string `json:"min_memory"`
	MinDisk     *string `json:"min_disk"`
	IsDefault   bool    `json:"is_default"`
}

// UpdateImageVersionReq 编辑镜像版本请求
// PUT /api/v1/image-versions/:id
type UpdateImageVersionReq struct {
	RegistryURL *string `json:"registry_url" binding:"omitempty,max=500"`
	MinCPU      *string `json:"min_cpu"`
	MinMemory   *string `json:"min_memory"`
	MinDisk     *string `json:"min_disk"`
}

// ImageVersionResp 镜像版本详情响应
type ImageVersionResp struct {
	ID          string          `json:"id"`
	ImageID     string          `json:"image_id"`
	Version     string          `json:"version"`
	RegistryURL string          `json:"registry_url"`
	ImageSize   *int64          `json:"image_size"`
	Digest      *string         `json:"digest"`
	MinCPU      *string         `json:"min_cpu"`
	MinMemory   *string         `json:"min_memory"`
	MinDisk     *string         `json:"min_disk"`
	IsDefault   bool            `json:"is_default"`
	Status      int16           `json:"status"`
	StatusText  string          `json:"status_text"`
	ScanResult  json.RawMessage `json:"scan_result"`
	ScannedAt   *string         `json:"scanned_at"`
	CreatedAt   string          `json:"created_at"`
}

// ========== 镜像分类 DTO ==========

// CreateImageCategoryReq 创建镜像分类请求
// POST /api/v1/image-categories
type CreateImageCategoryReq struct {
	Name        string  `json:"name" binding:"required,max=50"`
	Code        string  `json:"code" binding:"required,max=50"`
	Description *string `json:"description" binding:"omitempty,max=200"`
	SortOrder   int     `json:"sort_order"`
}

// UpdateImageCategoryReq 更新镜像分类请求
// PUT /api/v1/image-categories/:id
type UpdateImageCategoryReq struct {
	Name        *string `json:"name" binding:"omitempty,max=50"`
	Description *string `json:"description" binding:"omitempty,max=200"`
	SortOrder   *int    `json:"sort_order"`
}

// ImageCategoryResp 镜像分类响应
type ImageCategoryResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
}

// ========== 镜像配置模板 DTO ==========

// ImageConfigTemplateResp 镜像配置模板响应
// GET /api/v1/images/:id/config-template
type ImageConfigTemplateResp struct {
	ImageID                string                      `json:"image_id"`
	Name                   string                      `json:"name"`
	DisplayName            string                      `json:"display_name"`
	Ecosystem              *string                     `json:"ecosystem"`
	DefaultPorts           []ImagePortItem             `json:"default_ports"`
	DefaultEnvVars         []ImageEnvVarItem           `json:"default_env_vars"`
	DefaultVolumes         []ImageVolumeItem           `json:"default_volumes"`
	TypicalCompanions      ImageTypicalCompanions      `json:"typical_companions"`
	RequiredDependencies   []ImageDependencyItem       `json:"required_dependencies"`
	ResourceRecommendation ImageResourceRecommendation `json:"resource_recommendation"`
	ConditionalEnvVars     []ConditionalEnvVarExample  `json:"conditional_env_vars_example"`
}

// ConditionalEnvVarExample 条件环境变量示例
type ConditionalEnvVarExample struct {
	Key          string                 `json:"key"`
	DefaultValue string                 `json:"default_value"`
	Conditions   []ImageEnvVarCondition `json:"conditions"`
	Description  string                 `json:"description"`
}

// ImageDocumentationResp 镜像结构化文档响应
// GET /api/v1/images/:id/documentation
type ImageDocumentationResp struct {
	ImageID     string                     `json:"image_id"`
	Name        string                     `json:"name"`
	DisplayName string                     `json:"display_name"`
	Sections    ImageDocumentationSections `json:"sections"`
}

// ImageDocumentationSections 镜像文档各章节
type ImageDocumentationSections struct {
	Overview          string `json:"overview"`
	VersionNotes      string `json:"version_notes"`
	DefaultConfig     string `json:"default_config"`
	TypicalCompanions string `json:"typical_companions"`
	EnvVarsReference  string `json:"env_vars_reference"`
	UsageExamples     string `json:"usage_examples"`
	Notes             string `json:"notes"`
}

// ========== 实验模板 DTO ==========

// CreateTemplateReq 创建实验模板请求
// POST /api/v1/experiment-templates
type CreateTemplateReq struct {
	Title              string   `json:"title" binding:"required,max=200"`
	Description        *string  `json:"description"`
	Objectives         *string  `json:"objectives"`
	Instructions       *string  `json:"instructions"`
	ReferenceMaterials *string  `json:"reference_materials"`
	ExperimentType     int16    `json:"experiment_type" binding:"required,oneof=1 2 3"`
	TopologyMode       int16    `json:"topology_mode" binding:"required,oneof=1 2 3 4"`
	JudgeMode          int16    `json:"judge_mode" binding:"required,oneof=1 2 3"`
	AutoWeight         *float64 `json:"auto_weight" binding:"omitempty,min=0,max=100"`
	ManualWeight       *float64 `json:"manual_weight" binding:"omitempty,min=0,max=100"`
	TotalScore         int      `json:"total_score" binding:"required,min=1,max=1000"`
	MaxDuration        int      `json:"max_duration" binding:"required,min=1"`
	IdleTimeout        *int     `json:"idle_timeout" binding:"omitempty,min=1"`
	CPULimit           *string  `json:"cpu_limit"`
	MemoryLimit        *string  `json:"memory_limit"`
	DiskLimit          *string  `json:"disk_limit"`
	ScoreStrategy      int16    `json:"score_strategy" binding:"required,oneof=1 2"`
}

// UpdateTemplateReq 编辑实验模板请求
// PUT /api/v1/experiment-templates/:id
type UpdateTemplateReq struct {
	Title              *string  `json:"title" binding:"omitempty,max=200"`
	Description        *string  `json:"description"`
	Objectives         *string  `json:"objectives"`
	Instructions       *string  `json:"instructions"`
	ReferenceMaterials *string  `json:"reference_materials"`
	ExperimentType     *int16   `json:"experiment_type" binding:"omitempty,oneof=1 2 3"`
	TopologyMode       *int16   `json:"topology_mode" binding:"omitempty,oneof=1 2 3 4"`
	JudgeMode          *int16   `json:"judge_mode" binding:"omitempty,oneof=1 2 3"`
	AutoWeight         *float64 `json:"auto_weight" binding:"omitempty,min=0,max=100"`
	ManualWeight       *float64 `json:"manual_weight" binding:"omitempty,min=0,max=100"`
	TotalScore         *int     `json:"total_score" binding:"omitempty,min=1,max=1000"`
	MaxDuration        *int     `json:"max_duration" binding:"omitempty,min=1"`
	IdleTimeout        *int     `json:"idle_timeout" binding:"omitempty,min=1"`
	CPULimit           *string  `json:"cpu_limit"`
	MemoryLimit        *string  `json:"memory_limit"`
	DiskLimit          *string  `json:"disk_limit"`
	ScoreStrategy      *int16   `json:"score_strategy" binding:"omitempty,oneof=1 2"`
}

// TemplateListReq 实验模板列表查询参数
// GET /api/v1/experiment-templates
type TemplateListReq struct {
	Page           int    `form:"page" binding:"omitempty,min=1"`
	PageSize       int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword        string `form:"keyword"`
	ExperimentType int16  `form:"experiment_type" binding:"omitempty,oneof=1 2 3"`
	Status         int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	TagID          string `form:"tag_id"`
	SortBy         string `form:"sort_by"`
	SortOrder      string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ShareTemplateReq 设置共享状态请求
// PATCH /api/v1/experiment-templates/:id/share
type ShareTemplateReq struct {
	IsShared bool `json:"is_shared"`
}

// K8sConfigReq 微调K8s编排配置请求
// POST /api/v1/experiment-templates/:id/k8s-config
type K8sConfigReq struct {
	K8sConfig json.RawMessage `json:"k8s_config" binding:"required"`
}

// ValidateTemplateReq 模板配置验证请求
// POST /api/v1/experiment-templates/:id/validate
type ValidateTemplateReq struct {
	Levels []int `json:"levels"`
}

// TemplateResp 实验模板详情响应
type TemplateResp struct {
	ID                 string                 `json:"id"`
	Title              string                 `json:"title"`
	Description        *string                `json:"description"`
	Objectives         *string                `json:"objectives"`
	Instructions       *string                `json:"instructions"`
	ReferenceMaterials *string                `json:"reference_materials"`
	ExperimentType     int16                  `json:"experiment_type"`
	ExperimentTypeText string                 `json:"experiment_type_text"`
	TopologyMode       int16                  `json:"topology_mode"`
	TopologyModeText   string                 `json:"topology_mode_text"`
	JudgeMode          int16                  `json:"judge_mode"`
	JudgeModeText      string                 `json:"judge_mode_text"`
	AutoWeight         *float64               `json:"auto_weight"`
	ManualWeight       *float64               `json:"manual_weight"`
	TotalScore         int                    `json:"total_score"`
	MaxDuration        int                    `json:"max_duration"`
	IdleTimeout        *int                   `json:"idle_timeout"`
	CPULimit           *string                `json:"cpu_limit"`
	MemoryLimit        *string                `json:"memory_limit"`
	DiskLimit          *string                `json:"disk_limit"`
	ScoreStrategy      int16                  `json:"score_strategy"`
	IsShared           bool                   `json:"is_shared"`
	Status             int16                  `json:"status"`
	StatusText         string                 `json:"status_text"`
	Teacher            *SimpleUserResp        `json:"teacher"`
	Containers         []ContainerResp        `json:"containers"`
	Checkpoints        []CheckpointResp       `json:"checkpoints"`
	InitScripts        []InitScriptResp       `json:"init_scripts"`
	SimScenes          []TemplateSimSceneResp `json:"sim_scenes"`
	Tags               []TagResp              `json:"tags"`
	Roles              []RoleResp             `json:"roles"`
	K8sConfig          json.RawMessage        `json:"k8s_config,omitempty"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

// SimpleUserResp 简要用户信息（嵌套用）
type SimpleUserResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TemplateListItem 实验模板列表项
type TemplateListItem struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	ExperimentType     int16     `json:"experiment_type"`
	ExperimentTypeText string    `json:"experiment_type_text"`
	TopologyMode       int16     `json:"topology_mode"`
	TopologyModeText   string    `json:"topology_mode_text"`
	JudgeMode          int16     `json:"judge_mode"`
	JudgeModeText      string    `json:"judge_mode_text"`
	TotalScore         int       `json:"total_score"`
	MaxDuration        int       `json:"max_duration"`
	IsShared           bool      `json:"is_shared"`
	Status             int16     `json:"status"`
	StatusText         string    `json:"status_text"`
	ContainerCount     int       `json:"container_count"`
	CheckpointCount    int       `json:"checkpoint_count"`
	Tags               []TagResp `json:"tags"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// CreateTemplateResp 创建实验模板响应
type CreateTemplateResp struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	ExperimentType     int16  `json:"experiment_type"`
	ExperimentTypeText string `json:"experiment_type_text"`
	Status             int16  `json:"status"`
	StatusText         string `json:"status_text"`
	TopologyMode       int16  `json:"topology_mode"`
	TopologyModeText   string `json:"topology_mode_text"`
	JudgeMode          int16  `json:"judge_mode"`
	JudgeModeText      string `json:"judge_mode_text"`
}

// K8sConfigResp K8s编排配置响应
// GET /api/v1/experiment-templates/:id/k8s-config
type K8sConfigResp struct {
	TemplateID string          `json:"template_id"`
	K8sConfig  json.RawMessage `json:"k8s_config"`
}

// ValidateTemplateResp 模板配置验证响应
// POST /api/v1/experiment-templates/:id/validate
type ValidateTemplateResp struct {
	TemplateID    string                  `json:"template_id"`
	IsPublishable bool                    `json:"is_publishable"`
	Summary       ValidationSummary       `json:"summary"`
	Results       []ValidationLevelResult `json:"results"`
}

// ValidationSummary 验证结果汇总
type ValidationSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Hints    int `json:"hints"`
	Infos    int `json:"infos"`
}

// ValidationLevelResult 单层验证结果
type ValidationLevelResult struct {
	Level     int               `json:"level"`
	LevelName string            `json:"level_name"`
	Severity  string            `json:"severity"`
	Passed    bool              `json:"passed"`
	Issues    []ValidationIssue `json:"issues"`
}

// ValidationIssue 验证问题项
type ValidationIssue struct {
	Code              string                `json:"code"`
	Message           string                `json:"message"`
	SourceContainer   *string               `json:"source_container,omitempty"`
	MissingDependency *string               `json:"missing_dependency,omitempty"`
	Suggestion        *ValidationSuggestion `json:"suggestion,omitempty"`
	CurrentTotalCPU   *string               `json:"current_total_cpu,omitempty"`
	QuotaLimitCPU     *string               `json:"quota_limit_cpu,omitempty"`
	ToolImage         *string               `json:"tool_image,omitempty"`
	ExpectedEcosystem *string               `json:"expected_ecosystem,omitempty"`
	CurrentEcosystems []string              `json:"current_ecosystems,omitempty"`
}

// ValidationSuggestion 模板验证问题的建议修复动作
type ValidationSuggestion struct {
	Action string `json:"action"`
	Image  string `json:"image"`
	Reason string `json:"reason"`
}

// ========== 模板容器配置 DTO ==========

// CreateContainerReq 添加容器配置请求
// POST /api/v1/experiment-templates/:id/containers
type CreateContainerReq struct {
	ImageVersionID string                `json:"image_version_id" binding:"required"`
	ContainerName  string                `json:"container_name" binding:"required,max=100"`
	DeploymentScope int16                `json:"deployment_scope" binding:"required,oneof=1 2"`
	RoleID         *string               `json:"role_id"`
	EnvVars        []ContainerEnvVarItem `json:"env_vars"`
	Ports          []ContainerPortItem   `json:"ports"`
	Volumes        []ContainerVolumeItem `json:"volumes"`
	CPULimit       *string               `json:"cpu_limit"`
	MemoryLimit    *string               `json:"memory_limit"`
	DependsOn      []string              `json:"depends_on"`
	StartupOrder   int                   `json:"startup_order"`
	IsPrimary      bool                  `json:"is_primary"`
}

// UpdateContainerReq 编辑容器配置请求
// PUT /api/v1/template-containers/:id
type UpdateContainerReq struct {
	ImageVersionID *string               `json:"image_version_id"`
	ContainerName  *string               `json:"container_name" binding:"omitempty,max=100"`
	DeploymentScope *int16               `json:"deployment_scope" binding:"omitempty,oneof=1 2"`
	RoleID         *string               `json:"role_id"`
	EnvVars        []ContainerEnvVarItem `json:"env_vars"`
	Ports          []ContainerPortItem   `json:"ports"`
	Volumes        []ContainerVolumeItem `json:"volumes"`
	CPULimit       *string               `json:"cpu_limit"`
	MemoryLimit    *string               `json:"memory_limit"`
	DependsOn      []string              `json:"depends_on"`
	StartupOrder   *int                  `json:"startup_order"`
	IsPrimary      *bool                 `json:"is_primary"`
}

// ContainerResp 容器配置响应
type ContainerResp struct {
	ID             string                     `json:"id"`
	TemplateID     string                     `json:"template_id"`
	ImageVersionID string                     `json:"image_version_id"`
	ImageVersion   *ContainerImageVersionResp `json:"image_version,omitempty"`
	ContainerName  string                     `json:"container_name"`
	DeploymentScope int16                     `json:"deployment_scope"`
	DeploymentScopeText string                `json:"deployment_scope_text"`
	RoleID         *string                    `json:"role_id"`
	EnvVars        []ContainerEnvVarItem      `json:"env_vars"`
	Ports          []ContainerPortItem        `json:"ports"`
	Volumes        []ContainerVolumeItem      `json:"volumes"`
	CPULimit       *string                    `json:"cpu_limit"`
	MemoryLimit    *string                    `json:"memory_limit"`
	DependsOn      []string                   `json:"depends_on"`
	StartupOrder   int                        `json:"startup_order"`
	IsPrimary      bool                       `json:"is_primary"`
}

// ContainerEnvVarItem 模板容器环境变量项
type ContainerEnvVarItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ContainerPortItem 模板容器端口映射项
type ContainerPortItem struct {
	Container int    `json:"container"`
	Protocol  string `json:"protocol"`
}

// ContainerVolumeItem 模板容器挂载卷项
type ContainerVolumeItem struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
}

// ContainerImageVersionResp 容器关联的镜像版本信息
type ContainerImageVersionResp struct {
	ID               string  `json:"id"`
	ImageName        string  `json:"image_name"`
	ImageDisplayName string  `json:"image_display_name"`
	Version          string  `json:"version"`
	IconURL          *string `json:"icon_url"`
}

// ========== 检查点 DTO ==========

// CreateCheckpointReq 添加检查点请求
// POST /api/v1/experiment-templates/:id/checkpoints
type CreateCheckpointReq struct {
	Title           string          `json:"title" binding:"required,max=200"`
	Description     *string         `json:"description"`
	CheckType       int16           `json:"check_type" binding:"required,oneof=1 2 3"`
	ScriptContent   *string         `json:"script_content"`
	ScriptLanguage  *string         `json:"script_language" binding:"omitempty,max=20"`
	TargetContainer *string         `json:"target_container" binding:"omitempty,max=100"`
	AssertionConfig json.RawMessage `json:"assertion_config"`
	Score           float64         `json:"score" binding:"required,min=0"`
	Scope           int16           `json:"scope" binding:"required,oneof=1 2"`
	SortOrder       int             `json:"sort_order"`
}

// UpdateCheckpointReq 编辑检查点请求
// PUT /api/v1/template-checkpoints/:id
type UpdateCheckpointReq struct {
	Title           *string         `json:"title" binding:"omitempty,max=200"`
	Description     *string         `json:"description"`
	CheckType       *int16          `json:"check_type" binding:"omitempty,oneof=1 2 3"`
	ScriptContent   *string         `json:"script_content"`
	ScriptLanguage  *string         `json:"script_language" binding:"omitempty,max=20"`
	TargetContainer *string         `json:"target_container" binding:"omitempty,max=100"`
	AssertionConfig json.RawMessage `json:"assertion_config"`
	Score           *float64        `json:"score" binding:"omitempty,min=0"`
	Scope           *int16          `json:"scope" binding:"omitempty,oneof=1 2"`
	SortOrder       *int            `json:"sort_order"`
}

// CheckpointResp 检查点响应
type CheckpointResp struct {
	ID              string          `json:"id"`
	TemplateID      string          `json:"template_id"`
	Title           string          `json:"title"`
	Description     *string         `json:"description"`
	CheckType       int16           `json:"check_type"`
	CheckTypeText   string          `json:"check_type_text"`
	ScriptContent   *string         `json:"script_content"`
	ScriptLanguage  *string         `json:"script_language"`
	TargetContainer *string         `json:"target_container"`
	AssertionConfig json.RawMessage `json:"assertion_config"`
	Score           float64         `json:"score"`
	Scope           int16           `json:"scope"`
	ScopeText       string          `json:"scope_text"`
	SortOrder       int             `json:"sort_order"`
}

// ========== 初始化脚本 DTO ==========

// CreateInitScriptReq 添加初始化脚本请求
// POST /api/v1/experiment-templates/:id/init-scripts
type CreateInitScriptReq struct {
	TargetContainer string `json:"target_container" binding:"required,max=100"`
	ScriptContent   string `json:"script_content" binding:"required"`
	ScriptLanguage  string `json:"script_language" binding:"required,max=20"`
	ExecutionOrder  int    `json:"execution_order"`
	Timeout         *int   `json:"timeout" binding:"omitempty,min=1"`
}

// UpdateInitScriptReq 编辑初始化脚本请求
// PUT /api/v1/template-init-scripts/:id
type UpdateInitScriptReq struct {
	TargetContainer *string `json:"target_container" binding:"omitempty,max=100"`
	ScriptContent   *string `json:"script_content"`
	ScriptLanguage  *string `json:"script_language" binding:"omitempty,max=20"`
	ExecutionOrder  *int    `json:"execution_order"`
	Timeout         *int    `json:"timeout" binding:"omitempty,min=1"`
}

// InitScriptResp 初始化脚本响应
type InitScriptResp struct {
	ID              string `json:"id"`
	TemplateID      string `json:"template_id"`
	TargetContainer string `json:"target_container"`
	ScriptContent   string `json:"script_content"`
	ScriptLanguage  string `json:"script_language"`
	ExecutionOrder  int    `json:"execution_order"`
	Timeout         int    `json:"timeout"`
}

// ========== 仿真场景库 DTO ==========

// CreateScenarioReq 上传自定义仿真场景请求
// POST /api/v1/sim-scenarios
type CreateScenarioReq struct {
	Name               string          `json:"name" binding:"required,max=100"`
	Code               string          `json:"code" binding:"required,max=100"`
	Description        *string         `json:"description"`
	Category           string          `json:"category" binding:"required,max=50"`
	AlgorithmType      string          `json:"algorithm_type" binding:"required,max=100"`
	TimeControlMode    string          `json:"time_control_mode" binding:"required,oneof=process reactive continuous"`
	ContainerImageURL  string          `json:"container_image_url" binding:"required,max=500"`
	ContainerImageSize *string         `json:"container_image_size" binding:"omitempty,max=20"`
	DefaultParams      json.RawMessage `json:"default_params"`
	InteractionSchema  json.RawMessage `json:"interaction_schema"`
	DataSourceMode     int16           `json:"data_source_mode" binding:"required,oneof=1 2 3"`
	DefaultSize        json.RawMessage `json:"default_size"`
}

// UpdateScenarioReq 编辑场景信息请求
// PUT /api/v1/sim-scenarios/:id
type UpdateScenarioReq struct {
	Name               *string         `json:"name" binding:"omitempty,max=100"`
	Description        *string         `json:"description"`
	Category           *string         `json:"category" binding:"omitempty,max=50"`
	AlgorithmType      *string         `json:"algorithm_type" binding:"omitempty,max=100"`
	TimeControlMode    *string         `json:"time_control_mode" binding:"omitempty,oneof=process reactive continuous"`
	ContainerImageURL  *string         `json:"container_image_url" binding:"omitempty,max=500"`
	ContainerImageSize *string         `json:"container_image_size" binding:"omitempty,max=20"`
	DefaultParams      json.RawMessage `json:"default_params"`
	InteractionSchema  json.RawMessage `json:"interaction_schema"`
	DataSourceMode     *int16          `json:"data_source_mode" binding:"omitempty,oneof=1 2 3"`
	DefaultSize        json.RawMessage `json:"default_size"`
}

// ReviewScenarioReq 审核场景请求
// POST /api/v1/sim-scenarios/:id/review
type ReviewScenarioReq struct {
	Action  string  `json:"action" binding:"required,oneof=approve reject"`
	Comment *string `json:"comment" binding:"omitempty,max=500"`
}

// ScenarioListReq 仿真场景列表查询参数
// GET /api/v1/sim-scenarios
type ScenarioListReq struct {
	Page            int    `form:"page" binding:"omitempty,min=1"`
	PageSize        int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword         string `form:"keyword"`
	Category        string `form:"category"`
	AlgorithmType   string `form:"algorithm_type"`
	SourceType      int16  `form:"source_type" binding:"omitempty,oneof=1 2"`
	Status          int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
	TimeControlMode string `form:"time_control_mode" binding:"omitempty,oneof=process reactive continuous"`
	DataSourceMode  int16  `form:"data_source_mode" binding:"omitempty,oneof=1 2 3"`
	SortBy          string `form:"sort_by"`
	SortOrder       string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ScenarioResp 仿真场景详情响应
type ScenarioResp struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Code               string          `json:"code"`
	Description        *string         `json:"description"`
	Category           string          `json:"category"`
	CategoryText       string          `json:"category_text"`
	AlgorithmType      string          `json:"algorithm_type"`
	SourceType         int16           `json:"source_type"`
	SourceTypeText     string          `json:"source_type_text"`
	UploadedBy         *string         `json:"uploaded_by"`
	UploaderName       *string         `json:"uploader_name"`
	Status             int16           `json:"status"`
	StatusText         string          `json:"status_text"`
	TimeControlMode    string          `json:"time_control_mode"`
	ContainerImageURL  string          `json:"container_image_url"`
	ContainerImageSize *string         `json:"container_image_size"`
	DefaultParams      json.RawMessage `json:"default_params"`
	InteractionSchema  json.RawMessage `json:"interaction_schema"`
	DataSourceMode     int16           `json:"data_source_mode"`
	DataSourceModeText string          `json:"data_source_mode_text"`
	DefaultSize        json.RawMessage `json:"default_size"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

// ScenarioListItem 仿真场景列表项
type ScenarioListItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Code            string `json:"code"`
	Category        string `json:"category"`
	CategoryText    string `json:"category_text"`
	AlgorithmType   string `json:"algorithm_type"`
	SourceType      int16  `json:"source_type"`
	SourceTypeText  string `json:"source_type_text"`
	Status          int16  `json:"status"`
	StatusText      string `json:"status_text"`
	TimeControlMode string `json:"time_control_mode"`
	DataSourceMode  int16  `json:"data_source_mode"`
	CreatedAt       string `json:"created_at"`
}

// ========== 联动组 DTO ==========

// LinkGroupResp 联动组详情响应
// GET /api/v1/sim-link-groups/:id
type LinkGroupResp struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	Scenes      []LinkGroupSceneResp `json:"scenes"`
}

// LinkGroupSceneResp 联动组关联场景
type LinkGroupSceneResp struct {
	ID         string `json:"id"`
	ScenarioID string `json:"scenario_id"`
	SceneName  string `json:"scene_name"`
	SceneCode  string `json:"scene_code"`
	LinkRole   string `json:"link_role"`
	SortOrder  int    `json:"sort_order"`
}

// LinkGroupListItem 联动组列表项
// GET /api/v1/sim-link-groups
type LinkGroupListItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SceneCount  int     `json:"scene_count"`
}

// ========== 模板仿真场景配置 DTO ==========

// CreateTemplateSimSceneReq 添加仿真场景到模板请求
// POST /api/v1/experiment-templates/:id/sim-scenes
type CreateTemplateSimSceneReq struct {
	ScenarioID       string          `json:"scenario_id" binding:"required"`
	LinkGroupID      *string         `json:"link_group_id"`
	SceneParams      json.RawMessage `json:"scene_params"`
	InitialState     json.RawMessage `json:"initial_state"`
	DataSourceMode   int16           `json:"data_source_mode" binding:"required,oneof=1 2 3"`
	DataSourceConfig json.RawMessage `json:"data_source_config"`
	LayoutPosition   json.RawMessage `json:"layout_position"`
}

// UpdateTemplateSimSceneReq 编辑仿真场景配置请求
// PUT /api/v1/template-sim-scenes/:id
type UpdateTemplateSimSceneReq struct {
	SceneParams      json.RawMessage `json:"scene_params"`
	InitialState     json.RawMessage `json:"initial_state"`
	DataSourceMode   *int16          `json:"data_source_mode" binding:"omitempty,oneof=1 2 3"`
	DataSourceConfig json.RawMessage `json:"data_source_config"`
	LayoutPosition   json.RawMessage `json:"layout_position"`
}

// UpdateSimSceneLayoutReq 更新仿真场景布局请求
// PUT /api/v1/experiment-templates/:id/sim-scenes/layout
type UpdateSimSceneLayoutReq struct {
	Items []SimSceneLayoutItem `json:"items" binding:"required,dive"`
}

// SimSceneLayoutItem 仿真场景布局项
type SimSceneLayoutItem struct {
	SimSceneID     string          `json:"sim_scene_id" binding:"required"`
	LayoutPosition json.RawMessage `json:"layout_position" binding:"required"`
}

// TemplateSimSceneResp 模板仿真场景配置响应
type TemplateSimSceneResp struct {
	ID                 string          `json:"id"`
	TemplateID         string          `json:"template_id"`
	Scenario           *ScenarioBrief  `json:"scenario"`
	LinkGroupID        *string         `json:"link_group_id"`
	LinkGroupName      *string         `json:"link_group_name"`
	SceneParams        json.RawMessage `json:"scene_params"`
	InitialState       json.RawMessage `json:"initial_state"`
	DataSourceMode     int16           `json:"data_source_mode"`
	DataSourceModeText string          `json:"data_source_mode_text"`
	DataSourceConfig   json.RawMessage `json:"data_source_config"`
	LayoutPosition     json.RawMessage `json:"layout_position"`
}

// ScenarioBrief 场景简要信息（嵌套用）
type ScenarioBrief struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Code               string  `json:"code"`
	Category           string  `json:"category"`
	CategoryText       string  `json:"category_text"`
	TimeControlMode    string  `json:"time_control_mode"`
	ContainerImageURL  string  `json:"container_image_url"`
	ContainerImageSize *string `json:"container_image_size"`
}

// ========== 标签 DTO ==========

// CreateTagReq 创建自定义标签请求
// POST /api/v1/tags
type CreateTagReq struct {
	Name     string `json:"name" binding:"required,max=50"`
	Category string `json:"category" binding:"required,oneof=ecosystem type difficulty custom"`
}

// TagListReq 标签列表查询参数
// GET /api/v1/tags
type TagListReq struct {
	Category string `form:"category" binding:"omitempty,oneof=ecosystem type difficulty custom"`
	Keyword  string `form:"keyword"`
}

// SetTemplateTagsReq 设置模板标签请求
// PUT /api/v1/experiment-templates/:id/tags
type SetTemplateTagsReq struct {
	TagIDs []string `json:"tag_ids" binding:"required"`
}

// TagResp 标签响应
type TagResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// ========== 多人实验角色 DTO ==========

// CreateRoleReq 添加角色请求
// POST /api/v1/experiment-templates/:id/roles
type CreateRoleReq struct {
	RoleName    string          `json:"role_name" binding:"required,max=50"`
	Description *string         `json:"description"`
	MaxMembers  int             `json:"max_members" binding:"required,min=1"`
	Permissions json.RawMessage `json:"permissions"`
}

// UpdateRoleReq 编辑角色请求
// PUT /api/v1/template-roles/:id
type UpdateRoleReq struct {
	RoleName    *string         `json:"role_name" binding:"omitempty,max=50"`
	Description *string         `json:"description"`
	MaxMembers  *int            `json:"max_members" binding:"omitempty,min=1"`
	Permissions json.RawMessage `json:"permissions"`
}

// RoleResp 角色响应
type RoleResp struct {
	ID          string          `json:"id"`
	TemplateID  string          `json:"template_id"`
	RoleName    string          `json:"role_name"`
	Description *string         `json:"description"`
	MaxMembers  int             `json:"max_members"`
	Permissions json.RawMessage `json:"permissions"`
}

// ========== 镜像预拉取管理 DTO ==========

// ImagePullStatusListReq 镜像预拉取状态列表查询参数
// GET /api/v1/admin/image-pull-status
type ImagePullStatusListReq struct {
	NodeName  string `form:"node_name"`
	ImageName string `form:"image_name"`
	Status    int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ImagePullStatusListResp 镜像预拉取状态列表响应
// GET /api/v1/admin/image-pull-status
type ImagePullStatusListResp struct {
	Summary    ImagePullStatusSummary `json:"summary"`
	Items      []ImagePullStatusItem  `json:"items"`
	Pagination ListPagination         `json:"pagination"`
}

// ImagePullStatusSummary 镜像预拉取状态汇总
type ImagePullStatusSummary struct {
	TotalImages     int     `json:"total_images"`
	TotalNodes      int     `json:"total_nodes"`
	FullyPulled     int     `json:"fully_pulled"`
	PartiallyPulled int     `json:"partially_pulled"`
	NotPulled       int     `json:"not_pulled"`
	CompletionRate  float64 `json:"completion_rate"`
}

// ImagePullStatusItem 单个镜像版本的预拉取状态
type ImagePullStatusItem struct {
	ImageName      string                `json:"image_name"`
	ImageVersion   string                `json:"image_version"`
	RegistryURL    string                `json:"registry_url"`
	SourceType     int16                 `json:"source_type"`
	SourceTypeText string                `json:"source_type_text"`
	Nodes          []ImagePullNodeStatus `json:"nodes"`
}

// ImagePullNodeStatus 单个节点上的镜像拉取状态
type ImagePullNodeStatus struct {
	NodeName      string  `json:"node_name"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	PulledAt      *string `json:"pulled_at"`
	NodeCacheSize string  `json:"node_cache_size"`
}

// ListPagination 列表分页信息。
// 该结构仅表达当前接口文档定义的分页字段，不扩展额外元数据。
type ListPagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// TriggerImagePullReq 触发镜像预拉取请求
// POST /api/v1/admin/image-pull
type TriggerImagePullReq struct {
	ImageIDs    []string `json:"image_ids"`
	TargetNodes []string `json:"target_nodes"`
	Force       bool     `json:"force"`
}

// TriggerImagePullResp 触发镜像预拉取响应
// POST /api/v1/admin/image-pull
type TriggerImagePullResp struct {
	TaskID      string   `json:"task_id"`
	TotalJobs   int      `json:"total_jobs"`
	Images      []string `json:"images"`
	TargetNodes []string `json:"target_nodes"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
}
