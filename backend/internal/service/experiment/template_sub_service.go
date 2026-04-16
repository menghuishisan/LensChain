// template_sub_service.go
// 模块04 — 实验环境：模板子资源业务逻辑
// 负责容器配置、检查点、初始化脚本、仿真场景配置、标签、角色的 CRUD
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// stringPtr 返回字符串指针。
func stringPtr(value string) *string {
	return &value
}

// mustMarshalRawJSON 将结构体编码为 JSONB 存储格式。
func mustMarshalRawJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

// refreshTemplateContainers 在容器增删改后统一重算默认配置、条件变量与服务发现变量。
func (s *templateSubService) refreshTemplateContainers(ctx context.Context, templateID int64) error {
	containers, err := s.containerRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return err
	}
	allContainers := make([]entity.TemplateContainer, 0, len(containers))
	for _, container := range containers {
		allContainers = append(allContainers, *container)
	}
	for _, container := range containers {
		version, err := s.imageVersionRepo.GetByID(ctx, container.ImageVersionID)
		if err != nil {
			return errcode.ErrImageVersionNotFound
		}
		image, err := s.imageRepo.GetByID(ctx, version.ImageID)
		if err != nil {
			return errcode.ErrImageNotFound
		}
		merged, err := mergeContainerConfig(*container, image, version, allContainers)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("容器配置刷新失败: %v", err))
		}

		fields := map[string]interface{}{
			"env_vars":   mustMarshalRawJSON(merged.EnvVars),
			"ports":      mustMarshalRawJSON(merged.Ports),
			"volumes":    mustMarshalRawJSON(merged.Volumes),
			"updated_at": time.Now(),
		}
		if container.CPULimit == nil && merged.CPULimit != "" {
			fields["cpu_limit"] = merged.CPULimit
		}
		if container.MemoryLimit == nil && merged.MemoryLimit != "" {
			fields["memory_limit"] = merged.MemoryLimit
		}
		if err := s.containerRepo.UpdateFields(ctx, container.ID, fields); err != nil {
			return err
		}
	}
	return nil
}

// TemplateSubService 模板子资源服务接口
type TemplateSubService interface {
	// 容器配置
	CreateContainer(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateContainerReq) (*dto.ContainerResp, error)
	UpdateContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64, req *dto.UpdateContainerReq) error
	DeleteContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64) error
	ListContainers(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.ContainerResp, error)
	SortContainers(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SortReq) error

	// 检查点
	CreateCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateCheckpointReq) (*dto.CheckpointResp, error)
	UpdateCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, checkpointID int64, req *dto.UpdateCheckpointReq) error
	DeleteCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, checkpointID int64) error
	ListCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.CheckpointResp, error)
	SortCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SortReq) error

	// 初始化脚本
	CreateInitScript(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateInitScriptReq) (*dto.InitScriptResp, error)
	UpdateInitScript(ctx context.Context, sc *svcctx.ServiceContext, scriptID int64, req *dto.UpdateInitScriptReq) error
	DeleteInitScript(ctx context.Context, sc *svcctx.ServiceContext, scriptID int64) error
	ListInitScripts(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.InitScriptResp, error)

	// 仿真场景配置
	CreateSimScene(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateTemplateSimSceneReq) (*dto.TemplateSimSceneResp, error)
	UpdateSimScene(ctx context.Context, sc *svcctx.ServiceContext, simSceneID int64, req *dto.UpdateTemplateSimSceneReq) error
	DeleteSimScene(ctx context.Context, sc *svcctx.ServiceContext, simSceneID int64) error
	ListSimScenes(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.TemplateSimSceneResp, error)
	UpdateSimSceneLayout(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.UpdateSimSceneLayoutReq) error

	// 标签
	CreateTag(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateTagReq) (*dto.TagResp, error)
	DeleteTag(ctx context.Context, sc *svcctx.ServiceContext, tagID int64) error
	ListTags(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TagListReq) ([]*dto.TagResp, error)
	SetTemplateTags(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SetTemplateTagsReq) error
	ListTemplateTags(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.TagResp, error)

	// 角色
	CreateRole(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateRoleReq) (*dto.RoleResp, error)
	UpdateRole(ctx context.Context, sc *svcctx.ServiceContext, roleID int64, req *dto.UpdateRoleReq) error
	DeleteRole(ctx context.Context, sc *svcctx.ServiceContext, roleID int64) error
	ListRoles(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.RoleResp, error)
}

// templateSubService 模板子资源服务实现
type templateSubService struct {
	templateRepo     experimentrepo.TemplateRepository
	containerRepo    experimentrepo.ContainerRepository
	checkpointRepo   experimentrepo.CheckpointRepository
	initScriptRepo   experimentrepo.InitScriptRepository
	simSceneRepo     experimentrepo.SimSceneRepository
	tagRepo          experimentrepo.TagRepository
	templateTagRepo  experimentrepo.TemplateTagRepository
	roleRepo         experimentrepo.RoleRepository
	imageVersionRepo experimentrepo.ImageVersionRepository
	imageRepo        experimentrepo.ImageRepository
	scenarioRepo     experimentrepo.ScenarioRepository
	linkGroupRepo    experimentrepo.LinkGroupRepository
}

// NewTemplateSubService 创建模板子资源服务实例
func NewTemplateSubService(
	templateRepo experimentrepo.TemplateRepository,
	containerRepo experimentrepo.ContainerRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	initScriptRepo experimentrepo.InitScriptRepository,
	simSceneRepo experimentrepo.SimSceneRepository,
	tagRepo experimentrepo.TagRepository,
	templateTagRepo experimentrepo.TemplateTagRepository,
	roleRepo experimentrepo.RoleRepository,
	imageVersionRepo experimentrepo.ImageVersionRepository,
	imageRepo experimentrepo.ImageRepository,
	scenarioRepo experimentrepo.ScenarioRepository,
	linkGroupRepo experimentrepo.LinkGroupRepository,
) TemplateSubService {
	return &templateSubService{
		templateRepo: templateRepo, containerRepo: containerRepo,
		checkpointRepo: checkpointRepo, initScriptRepo: initScriptRepo,
		simSceneRepo: simSceneRepo, tagRepo: tagRepo,
		templateTagRepo: templateTagRepo, roleRepo: roleRepo,
		imageVersionRepo: imageVersionRepo, imageRepo: imageRepo,
		scenarioRepo: scenarioRepo, linkGroupRepo: linkGroupRepo,
	}
}

// ensureDraft 检查模板是否为草稿状态
func (s *templateSubService) ensureDraft(ctx context.Context, templateID int64) error {
	tpl, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if tpl.Status != enum.TemplateStatusDraft {
		return errcode.ErrTemplateNotDraft
	}
	return nil
}

// ---------------------------------------------------------------------------
// 容器配置
// ---------------------------------------------------------------------------

// CreateContainer 添加容器配置
func (s *templateSubService) CreateContainer(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateContainerReq) (*dto.ContainerResp, error) {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return nil, err
	}

	imageVersionID, _ := snowflake.ParseString(req.ImageVersionID)
	version, err := s.imageVersionRepo.GetByID(ctx, imageVersionID)
	if err != nil {
		return nil, errcode.ErrImageVersionNotFound
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil {
		return nil, errcode.ErrImageNotFound
	}
	existingContainers, err := s.containerRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	container := &entity.TemplateContainer{
		ID:             snowflake.Generate(),
		TemplateID:     templateID,
		ImageVersionID: imageVersionID,
		ContainerName:  req.ContainerName,
		EnvVars:        req.EnvVars,
		Ports:          req.Ports,
		Volumes:        req.Volumes,
		CPULimit:       req.CPULimit,
		MemoryLimit:    req.MemoryLimit,
		DependsOn:      req.DependsOn,
		StartupOrder:   req.StartupOrder,
		IsPrimary:      req.IsPrimary,
	}
	if req.RoleID != nil {
		roleID, _ := snowflake.ParseString(*req.RoleID)
		container.RoleID = &roleID
	}

	allContainers := make([]entity.TemplateContainer, 0, len(existingContainers)+1)
	for _, existing := range existingContainers {
		allContainers = append(allContainers, *existing)
	}
	allContainers = append(allContainers, *container)
	mergedSpec, err := mergeContainerConfig(*container, image, version, allContainers)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("容器配置合并失败: %v", err))
	}
	container.EnvVars = mustMarshalRawJSON(mergedSpec.EnvVars)
	container.Ports = mustMarshalRawJSON(mergedSpec.Ports)
	container.Volumes = mustMarshalRawJSON(mergedSpec.Volumes)
	if container.CPULimit == nil && mergedSpec.CPULimit != "" {
		container.CPULimit = stringPtr(mergedSpec.CPULimit)
	}
	if container.MemoryLimit == nil && mergedSpec.MemoryLimit != "" {
		container.MemoryLimit = stringPtr(mergedSpec.MemoryLimit)
	}

	if err := s.containerRepo.Create(ctx, container); err != nil {
		return nil, err
	}
	return s.toContainerResp(ctx, container), nil
}

// UpdateContainer 编辑容器配置
func (s *templateSubService) UpdateContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64, req *dto.UpdateContainerReq) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return errcode.ErrContainerNotFound
	}
	if err := s.ensureDraft(ctx, container.TemplateID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.ImageVersionID != nil {
		versionID, _ := snowflake.ParseString(*req.ImageVersionID)
		if _, err := s.imageVersionRepo.GetByID(ctx, versionID); err != nil {
			return errcode.ErrImageVersionNotFound
		}
		fields["image_version_id"] = versionID
	}
	if req.ContainerName != nil {
		fields["container_name"] = *req.ContainerName
	}
	if req.RoleID != nil {
		roleID, _ := snowflake.ParseString(*req.RoleID)
		fields["role_id"] = roleID
	}
	if req.EnvVars != nil {
		fields["env_vars"] = req.EnvVars
	}
	if req.Ports != nil {
		fields["ports"] = req.Ports
	}
	if req.Volumes != nil {
		fields["volumes"] = req.Volumes
	}
	if req.CPULimit != nil {
		fields["cpu_limit"] = *req.CPULimit
	}
	if req.MemoryLimit != nil {
		fields["memory_limit"] = *req.MemoryLimit
	}
	if req.DependsOn != nil {
		fields["depends_on"] = req.DependsOn
	}
	if req.StartupOrder != nil {
		fields["startup_order"] = *req.StartupOrder
	}
	if req.IsPrimary != nil {
		fields["is_primary"] = *req.IsPrimary
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	if err := s.containerRepo.UpdateFields(ctx, containerID, fields); err != nil {
		return err
	}
	return s.refreshTemplateContainers(ctx, container.TemplateID)
}

// DeleteContainer 删除容器配置
func (s *templateSubService) DeleteContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return errcode.ErrContainerNotFound
	}
	if err := s.ensureDraft(ctx, container.TemplateID); err != nil {
		return err
	}
	if err := s.containerRepo.Delete(ctx, containerID); err != nil {
		return err
	}
	return s.refreshTemplateContainers(ctx, container.TemplateID)
}

// ListContainers 获取模板的所有容器配置
func (s *templateSubService) ListContainers(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.ContainerResp, error) {
	containers, err := s.containerRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.ContainerResp, 0, len(containers))
	for _, c := range containers {
		result = append(result, s.toContainerResp(ctx, c))
	}
	return result, nil
}

// SortContainers 按请求顺序更新模板容器排序。
func (s *templateSubService) SortContainers(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SortReq) error {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return err
	}
	now := time.Now()
	for _, item := range req.Items {
		containerID, err := snowflake.ParseString(item.ID)
		if err != nil {
			return errcode.ErrContainerNotFound
		}
		container, err := s.containerRepo.GetByID(ctx, containerID)
		if err != nil {
			return errcode.ErrContainerNotFound
		}
		if container.TemplateID != templateID {
			return errcode.ErrForbidden.WithMessage("存在不属于当前模板的容器配置")
		}
		if err := s.containerRepo.UpdateFields(ctx, containerID, map[string]interface{}{
			"sort_order":    item.SortOrder,
			"startup_order": item.SortOrder,
			"updated_at":    now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 检查点
// ---------------------------------------------------------------------------

// CreateCheckpoint 添加检查点
func (s *templateSubService) CreateCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateCheckpointReq) (*dto.CheckpointResp, error) {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return nil, err
	}

	checkpoint := &entity.TemplateCheckpoint{
		ID:              snowflake.Generate(),
		TemplateID:      templateID,
		Title:           req.Title,
		Description:     req.Description,
		CheckType:       req.CheckType,
		ScriptContent:   req.ScriptContent,
		ScriptLanguage:  req.ScriptLanguage,
		TargetContainer: req.TargetContainer,
		AssertionConfig: req.AssertionConfig,
		Score:           req.Score,
		Scope:           req.Scope,
		SortOrder:       req.SortOrder,
	}

	if err := s.checkpointRepo.Create(ctx, checkpoint); err != nil {
		return nil, err
	}
	return s.toCheckpointResp(checkpoint), nil
}

// UpdateCheckpoint 编辑检查点
func (s *templateSubService) UpdateCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, checkpointID int64, req *dto.UpdateCheckpointReq) error {
	cp, err := s.checkpointRepo.GetByID(ctx, checkpointID)
	if err != nil {
		return errcode.ErrCheckpointNotFound
	}
	if err := s.ensureDraft(ctx, cp.TemplateID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.CheckType != nil {
		fields["check_type"] = *req.CheckType
	}
	if req.ScriptContent != nil {
		fields["script_content"] = *req.ScriptContent
	}
	if req.ScriptLanguage != nil {
		fields["script_language"] = *req.ScriptLanguage
	}
	if req.TargetContainer != nil {
		fields["target_container"] = *req.TargetContainer
	}
	if req.AssertionConfig != nil {
		fields["assertion_config"] = req.AssertionConfig
	}
	if req.Score != nil {
		fields["score"] = *req.Score
	}
	if req.Scope != nil {
		fields["scope"] = *req.Scope
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.checkpointRepo.UpdateFields(ctx, checkpointID, fields)
}

// DeleteCheckpoint 删除检查点
func (s *templateSubService) DeleteCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, checkpointID int64) error {
	cp, err := s.checkpointRepo.GetByID(ctx, checkpointID)
	if err != nil {
		return errcode.ErrCheckpointNotFound
	}
	if err := s.ensureDraft(ctx, cp.TemplateID); err != nil {
		return err
	}
	return s.checkpointRepo.Delete(ctx, checkpointID)
}

// ListCheckpoints 获取模板的所有检查点
func (s *templateSubService) ListCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.CheckpointResp, error) {
	checkpoints, err := s.checkpointRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.CheckpointResp, 0, len(checkpoints))
	for _, cp := range checkpoints {
		result = append(result, s.toCheckpointResp(cp))
	}
	return result, nil
}

// SortCheckpoints 按请求顺序更新模板检查点排序。
func (s *templateSubService) SortCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SortReq) error {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return err
	}
	now := time.Now()
	for _, item := range req.Items {
		checkpointID, err := snowflake.ParseString(item.ID)
		if err != nil {
			return errcode.ErrCheckpointNotFound
		}
		checkpoint, err := s.checkpointRepo.GetByID(ctx, checkpointID)
		if err != nil {
			return errcode.ErrCheckpointNotFound
		}
		if checkpoint.TemplateID != templateID {
			return errcode.ErrForbidden.WithMessage("存在不属于当前模板的检查点")
		}
		if err := s.checkpointRepo.UpdateFields(ctx, checkpointID, map[string]interface{}{
			"sort_order": item.SortOrder,
			"updated_at": now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 初始化脚本
// ---------------------------------------------------------------------------

// CreateInitScript 添加初始化脚本
func (s *templateSubService) CreateInitScript(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateInitScriptReq) (*dto.InitScriptResp, error) {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return nil, err
	}

	script := &entity.TemplateInitScript{
		ID:              snowflake.Generate(),
		TemplateID:      templateID,
		TargetContainer: req.TargetContainer,
		ScriptContent:   req.ScriptContent,
		ScriptLanguage:  req.ScriptLanguage,
		ExecutionOrder:  req.ExecutionOrder,
	}
	if req.Timeout != nil {
		script.Timeout = *req.Timeout
	} else {
		script.Timeout = 300
	}

	if err := s.initScriptRepo.Create(ctx, script); err != nil {
		return nil, err
	}
	resp := s.toInitScriptResp(script)
	// Description 是 DTO-only 字段，不持久化到数据库
	resp.Description = req.Description
	return resp, nil
}

// UpdateInitScript 编辑初始化脚本
func (s *templateSubService) UpdateInitScript(ctx context.Context, sc *svcctx.ServiceContext, scriptID int64, req *dto.UpdateInitScriptReq) error {
	script, err := s.initScriptRepo.GetByID(ctx, scriptID)
	if err != nil {
		return errcode.ErrInitScriptNotFound
	}
	if err := s.ensureDraft(ctx, script.TemplateID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.TargetContainer != nil {
		fields["target_container"] = *req.TargetContainer
	}
	if req.ScriptContent != nil {
		fields["script_content"] = *req.ScriptContent
	}
	if req.ScriptLanguage != nil {
		fields["script_language"] = *req.ScriptLanguage
	}
	if req.ExecutionOrder != nil {
		fields["execution_order"] = *req.ExecutionOrder
	}
	if req.Timeout != nil {
		fields["timeout"] = *req.Timeout
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.initScriptRepo.UpdateFields(ctx, scriptID, fields)
}

// DeleteInitScript 删除初始化脚本
func (s *templateSubService) DeleteInitScript(ctx context.Context, sc *svcctx.ServiceContext, scriptID int64) error {
	script, err := s.initScriptRepo.GetByID(ctx, scriptID)
	if err != nil {
		return errcode.ErrInitScriptNotFound
	}
	if err := s.ensureDraft(ctx, script.TemplateID); err != nil {
		return err
	}
	return s.initScriptRepo.Delete(ctx, scriptID)
}

// ListInitScripts 获取模板的所有初始化脚本
func (s *templateSubService) ListInitScripts(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.InitScriptResp, error) {
	scripts, err := s.initScriptRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.InitScriptResp, 0, len(scripts))
	for _, script := range scripts {
		result = append(result, s.toInitScriptResp(script))
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 仿真场景配置
// ---------------------------------------------------------------------------

// simSceneConfig 仿真场景 Config JSONB 内部结构
// entity 的 Config 字段存储合并后的 JSON，DTO 拆分为 SceneParams + InitialState + DataSourceMode
type simSceneConfig struct {
	SceneParams    json.RawMessage `json:"scene_params,omitempty"`
	InitialState   json.RawMessage `json:"initial_state,omitempty"`
	DataSourceMode int             `json:"data_source_mode"`
}

// CreateSimScene 添加仿真场景到模板
func (s *templateSubService) CreateSimScene(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateTemplateSimSceneReq) (*dto.TemplateSimSceneResp, error) {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return nil, err
	}

	scenarioID, _ := snowflake.ParseString(req.ScenarioID)
	_, err := s.scenarioRepo.GetByID(ctx, scenarioID)
	if err != nil {
		return nil, errcode.ErrScenarioNotFound
	}

	// 合并 SceneParams + InitialState + DataSourceMode 到 Config JSONB
	cfg := simSceneConfig{
		SceneParams:    req.SceneParams,
		InitialState:   req.InitialState,
		DataSourceMode: req.DataSourceMode,
	}
	configBytes, _ := json.Marshal(cfg)

	scene := &entity.TemplateSimScene{
		ID:               snowflake.Generate(),
		TemplateID:       templateID,
		ScenarioID:       scenarioID,
		Config:           configBytes,
		DataSourceConfig: req.DataSourceConfig,
		LayoutPosition:   req.LayoutPosition,
	}
	if req.LinkGroupID != nil {
		lgID, _ := snowflake.ParseString(*req.LinkGroupID)
		scene.LinkGroupID = &lgID
	}

	if err := s.simSceneRepo.Create(ctx, scene); err != nil {
		return nil, err
	}
	return s.toSimSceneResp(ctx, scene), nil
}

// UpdateSimScene 编辑仿真场景配置
func (s *templateSubService) UpdateSimScene(ctx context.Context, sc *svcctx.ServiceContext, simSceneID int64, req *dto.UpdateTemplateSimSceneReq) error {
	scene, err := s.simSceneRepo.GetByID(ctx, simSceneID)
	if err != nil {
		return errcode.ErrSimSceneNotFound
	}
	if err := s.ensureDraft(ctx, scene.TemplateID); err != nil {
		return err
	}

	fields := make(map[string]interface{})

	// 解析现有 Config，合并更新
	var existingCfg simSceneConfig
	if scene.Config != nil {
		_ = json.Unmarshal(scene.Config, &existingCfg)
	}
	configChanged := false
	if req.SceneParams != nil {
		existingCfg.SceneParams = req.SceneParams
		configChanged = true
	}
	if req.InitialState != nil {
		existingCfg.InitialState = req.InitialState
		configChanged = true
	}
	if req.DataSourceMode != nil {
		existingCfg.DataSourceMode = *req.DataSourceMode
		configChanged = true
	}
	if configChanged {
		configBytes, _ := json.Marshal(existingCfg)
		fields["config"] = configBytes
	}

	if req.DataSourceConfig != nil {
		fields["data_source_config"] = req.DataSourceConfig
	}
	if req.LayoutPosition != nil {
		fields["layout_position"] = req.LayoutPosition
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.simSceneRepo.UpdateFields(ctx, simSceneID, fields)
}

// DeleteSimScene 删除仿真场景配置
func (s *templateSubService) DeleteSimScene(ctx context.Context, sc *svcctx.ServiceContext, simSceneID int64) error {
	scene, err := s.simSceneRepo.GetByID(ctx, simSceneID)
	if err != nil {
		return errcode.ErrSimSceneNotFound
	}
	if err := s.ensureDraft(ctx, scene.TemplateID); err != nil {
		return err
	}
	return s.simSceneRepo.Delete(ctx, simSceneID)
}

// ListSimScenes 获取模板的所有仿真场景配置
func (s *templateSubService) ListSimScenes(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.TemplateSimSceneResp, error) {
	scenes, err := s.simSceneRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.TemplateSimSceneResp, 0, len(scenes))
	for _, scene := range scenes {
		result = append(result, s.toSimSceneResp(ctx, scene))
	}
	return result, nil
}

// UpdateSimSceneLayout 批量更新仿真场景布局
func (s *templateSubService) UpdateSimSceneLayout(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.UpdateSimSceneLayoutReq) error {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return err
	}
	now := time.Now()
	for _, item := range req.Items {
		sceneID, _ := snowflake.ParseString(item.SimSceneID)
		scene, err := s.simSceneRepo.GetByID(ctx, sceneID)
		if err != nil {
			return errcode.ErrSimSceneNotFound
		}
		if scene.TemplateID != templateID {
			return errcode.ErrForbidden.WithMessage("存在不属于当前模板的仿真场景配置")
		}
		fields := map[string]interface{}{
			"layout_position": item.LayoutPosition,
			"updated_at":      now,
		}
		if err := s.simSceneRepo.UpdateFields(ctx, sceneID, fields); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 标签
// ---------------------------------------------------------------------------

// CreateTag 创建自定义标签
func (s *templateSubService) CreateTag(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateTagReq) (*dto.TagResp, error) {
	tag := &entity.Tag{
		ID:       snowflake.Generate(),
		Name:     req.Name,
		Category: req.Category,
		IsSystem: false,
	}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return s.toTagResp(tag), nil
}

// DeleteTag 删除标签
func (s *templateSubService) DeleteTag(ctx context.Context, sc *svcctx.ServiceContext, tagID int64) error {
	_, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return errcode.ErrTagNotFound
	}
	inUse, err := s.tagRepo.IsTagInUse(ctx, tagID)
	if err != nil {
		return err
	}
	if inUse {
		return errcode.ErrTagInUse
	}
	return s.tagRepo.Delete(ctx, tagID)
}

// ListTags 获取标签列表
func (s *templateSubService) ListTags(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TagListReq) ([]*dto.TagResp, error) {
	var tags []*entity.Tag
	var err error
	if req.Category != "" {
		tags, err = s.tagRepo.ListByCategory(ctx, req.Category)
	} else {
		tags, err = s.tagRepo.ListAll(ctx)
	}
	if err != nil {
		return nil, err
	}
	result := make([]*dto.TagResp, 0, len(tags))
	for _, t := range tags {
		if req.Keyword != "" {
			// 简单关键词过滤（标签数量有限，内存过滤即可）
			if !containsKeyword(t.Name, req.Keyword) {
				continue
			}
		}
		result = append(result, s.toTagResp(t))
	}
	return result, nil
}

// SetTemplateTags 设置模板标签（全量替换）
func (s *templateSubService) SetTemplateTags(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.SetTemplateTagsReq) error {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return err
	}
	// 删除现有关联
	if err := s.templateTagRepo.DeleteByTemplateID(ctx, templateID); err != nil {
		return err
	}
	// 批量创建新关联
	if len(req.TagIDs) == 0 {
		return nil
	}
	tags := make([]*entity.TemplateTag, 0, len(req.TagIDs))
	for _, idStr := range req.TagIDs {
		tagID, _ := snowflake.ParseString(idStr)
		tags = append(tags, &entity.TemplateTag{
			ID:         snowflake.Generate(),
			TemplateID: templateID,
			TagID:      tagID,
		})
	}
	return s.templateTagRepo.BatchCreate(ctx, tags)
}

// ListTemplateTags 获取模板的标签列表
func (s *templateSubService) ListTemplateTags(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.TagResp, error) {
	templateTags, err := s.templateTagRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.TagResp, 0, len(templateTags))
	for _, tt := range templateTags {
		tag, err := s.tagRepo.GetByID(ctx, tt.TagID)
		if err != nil {
			continue
		}
		result = append(result, s.toTagResp(tag))
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 角色
// ---------------------------------------------------------------------------

// CreateRole 添加角色
func (s *templateSubService) CreateRole(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateRoleReq) (*dto.RoleResp, error) {
	if err := s.ensureDraft(ctx, templateID); err != nil {
		return nil, err
	}

	role := &entity.TemplateRole{
		ID:          snowflake.Generate(),
		TemplateID:  templateID,
		RoleName:    req.RoleName,
		Description: req.Description,
		MaxMembers:  req.MaxMembers,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	resp := s.toRoleResp(role)
	// Permissions 是 DTO-only 字段，不持久化到数据库
	resp.Permissions = req.Permissions
	return resp, nil
}

// UpdateRole 编辑角色
func (s *templateSubService) UpdateRole(ctx context.Context, sc *svcctx.ServiceContext, roleID int64, req *dto.UpdateRoleReq) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errcode.ErrRoleNotFound
	}
	if err := s.ensureDraft(ctx, role.TemplateID); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.RoleName != nil {
		fields["role_name"] = *req.RoleName
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.MaxMembers != nil {
		fields["max_members"] = *req.MaxMembers
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.roleRepo.UpdateFields(ctx, roleID, fields)
}

// DeleteRole 删除角色
func (s *templateSubService) DeleteRole(ctx context.Context, sc *svcctx.ServiceContext, roleID int64) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errcode.ErrRoleNotFound
	}
	if err := s.ensureDraft(ctx, role.TemplateID); err != nil {
		return err
	}
	return s.roleRepo.Delete(ctx, roleID)
}

// ListRoles 获取模板的所有角色
func (s *templateSubService) ListRoles(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.RoleResp, error) {
	roles, err := s.roleRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.RoleResp, 0, len(roles))
	for _, r := range roles {
		result = append(result, s.toRoleResp(r))
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 转换方法
// ---------------------------------------------------------------------------

// toContainerResp 转换容器实体为响应（含镜像版本信息）
func (s *templateSubService) toContainerResp(ctx context.Context, c *entity.TemplateContainer) *dto.ContainerResp {
	resp := &dto.ContainerResp{
		ID:             strconv.FormatInt(c.ID, 10),
		TemplateID:     strconv.FormatInt(c.TemplateID, 10),
		ImageVersionID: strconv.FormatInt(c.ImageVersionID, 10),
		ContainerName:  c.ContainerName,
		EnvVars:        c.EnvVars,
		Ports:          c.Ports,
		Volumes:        c.Volumes,
		CPULimit:       c.CPULimit,
		MemoryLimit:    c.MemoryLimit,
		DependsOn:      c.DependsOn,
		StartupOrder:   c.StartupOrder,
		IsPrimary:      c.IsPrimary,
	}
	if c.RoleID != nil {
		roleIDStr := strconv.FormatInt(*c.RoleID, 10)
		resp.RoleID = &roleIDStr
	}

	// 关联镜像版本信息
	version, err := s.imageVersionRepo.GetByID(ctx, c.ImageVersionID)
	if err == nil {
		ivResp := &dto.ContainerImageVersionResp{
			ID:      strconv.FormatInt(version.ID, 10),
			Version: version.Version,
		}
		image, err := s.imageRepo.GetByID(ctx, version.ImageID)
		if err == nil {
			ivResp.ImageName = image.Name
			ivResp.ImageDisplayName = image.DisplayName
			ivResp.IconURL = image.IconURL
		}
		resp.ImageVersion = ivResp
	}
	return resp
}

// toCheckpointResp 转换检查点实体为响应
func (s *templateSubService) toCheckpointResp(cp *entity.TemplateCheckpoint) *dto.CheckpointResp {
	return &dto.CheckpointResp{
		ID:              strconv.FormatInt(cp.ID, 10),
		TemplateID:      strconv.FormatInt(cp.TemplateID, 10),
		Title:           cp.Title,
		Description:     cp.Description,
		CheckType:       cp.CheckType,
		CheckTypeText:   enum.GetCheckTypeText(cp.CheckType),
		ScriptContent:   cp.ScriptContent,
		ScriptLanguage:  cp.ScriptLanguage,
		TargetContainer: cp.TargetContainer,
		AssertionConfig: cp.AssertionConfig,
		Score:           cp.Score,
		Scope:           cp.Scope,
		ScopeText:       enum.GetCheckpointScopeText(cp.Scope),
		SortOrder:       cp.SortOrder,
	}
}

// toInitScriptResp 转换初始化脚本实体为响应
func (s *templateSubService) toInitScriptResp(script *entity.TemplateInitScript) *dto.InitScriptResp {
	return &dto.InitScriptResp{
		ID:              strconv.FormatInt(script.ID, 10),
		TemplateID:      strconv.FormatInt(script.TemplateID, 10),
		TargetContainer: script.TargetContainer,
		ScriptContent:   script.ScriptContent,
		ScriptLanguage:  script.ScriptLanguage,
		ExecutionOrder:  script.ExecutionOrder,
		Timeout:         script.Timeout,
	}
}

// toSimSceneResp 转换仿真场景配置实体为响应
func (s *templateSubService) toSimSceneResp(ctx context.Context, scene *entity.TemplateSimScene) *dto.TemplateSimSceneResp {
	resp := &dto.TemplateSimSceneResp{
		ID:               strconv.FormatInt(scene.ID, 10),
		TemplateID:       strconv.FormatInt(scene.TemplateID, 10),
		DataSourceConfig: scene.DataSourceConfig,
		LayoutPosition:   scene.LayoutPosition,
	}

	// 拆分 Config JSONB → SceneParams + InitialState + DataSourceMode
	if scene.Config != nil {
		var cfg simSceneConfig
		if err := json.Unmarshal(scene.Config, &cfg); err == nil {
			resp.SceneParams = cfg.SceneParams
			resp.InitialState = cfg.InitialState
			resp.DataSourceMode = cfg.DataSourceMode
			resp.DataSourceModeText = enum.GetDataSourceModeText(cfg.DataSourceMode)
		}
	}

	// 联动组信息
	if scene.LinkGroupID != nil {
		lgIDStr := strconv.FormatInt(*scene.LinkGroupID, 10)
		resp.LinkGroupID = &lgIDStr
		lg, err := s.linkGroupRepo.GetByID(ctx, *scene.LinkGroupID)
		if err == nil {
			resp.LinkGroupName = &lg.Name
		}
	}

	// 场景简要信息
	scenario, err := s.scenarioRepo.GetByID(ctx, scene.ScenarioID)
	if err == nil {
		brief := &dto.ScenarioBrief{
			ID:              strconv.FormatInt(scenario.ID, 10),
			Name:            scenario.Name,
			Code:            scenario.Code,
			Category:        scenario.Category,
			CategoryText:    enum.GetScenarioCategoryText(scenario.Category),
			TimeControlMode: scenario.TimeControlMode,
		}
		if scenario.ContainerImageURL != nil {
			brief.ContainerImageURL = *scenario.ContainerImageURL
		}
		if scenario.ContainerImageSize != nil {
			sizeStr := strconv.FormatInt(*scenario.ContainerImageSize, 10)
			brief.ContainerImageSize = &sizeStr
		}
		resp.Scenario = brief
	}

	return resp
}

// toTagResp 转换标签实体为响应
func (s *templateSubService) toTagResp(t *entity.Tag) *dto.TagResp {
	return &dto.TagResp{
		ID:       strconv.FormatInt(t.ID, 10),
		Name:     t.Name,
		Category: t.Category,
	}
}

// toRoleResp 转换角色实体为响应
func (s *templateSubService) toRoleResp(r *entity.TemplateRole) *dto.RoleResp {
	return &dto.RoleResp{
		ID:          strconv.FormatInt(r.ID, 10),
		TemplateID:  strconv.FormatInt(r.TemplateID, 10),
		RoleName:    r.RoleName,
		Description: r.Description,
		MaxMembers:  r.MaxMembers,
	}
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// containsKeyword 简单关键词匹配（不区分大小写）
func containsKeyword(s, keyword string) bool {
	return len(keyword) > 0 && len(s) > 0 &&
		(strings.Contains(strings.ToLower(s), strings.ToLower(keyword)))
}
