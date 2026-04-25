// template_sub_service.go
// 模块04 — 实验环境：模板子资源业务逻辑
// 负责容器配置、检查点、初始化脚本、仿真场景配置、标签、角色的 CRUD
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/datatypes"

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

// resolveContainerRoleID 解析并校验容器绑定的角色。
// 角色只允许引用当前模板下已定义的角色，避免把跨模板角色或无效角色写入容器配置。
func (s *templateSubService) resolveContainerRoleID(ctx context.Context, templateID int64, roleIDText *string) (*int64, error) {
	if roleIDText == nil {
		return nil, nil
	}
	roleID, err := snowflake.ParseString(*roleIDText)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("角色ID无效")
	}
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("角色不存在")
	}
	if role.TemplateID != templateID {
		return nil, errcode.ErrInvalidParams.WithMessage("角色不属于当前实验模板")
	}
	return &roleID, nil
}

// ensureCollaborativeTemplate 校验模板是否为多人协作组网拓扑。
// 角色定义仅服务于多人协作实验，其他拓扑不允许创建或保留角色配置。
func (s *templateSubService) ensureCollaborativeTemplate(ctx context.Context, templateID int64) error {
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.TopologyMode == nil || *template.TopologyMode != enum.TopologyModeCollaborate {
		return errcode.ErrInvalidParams.WithMessage("仅多人协作组网模板允许定义角色")
	}
	return nil
}

// validateContainerDeploymentScope 校验容器部署范围与模板拓扑的组合是否合法。
func validateContainerDeploymentScope(template *entity.ExperimentTemplate, deploymentScope int16) error {
	if !enum.IsValidContainerDeploymentScope(deploymentScope) {
		return errcode.ErrInvalidParams.WithMessage("容器部署范围不合法")
	}
	if template == nil || template.TopologyMode == nil {
		return errcode.ErrInvalidParams.WithMessage("当前模板未配置拓扑模式")
	}
	if *template.TopologyMode == enum.TopologyModeShared {
		return nil
	}
	if deploymentScope != enum.ContainerDeploymentScopeInstance {
		return errcode.ErrInvalidParams.WithMessage("仅共享基础设施拓扑允许配置共享容器")
	}
	return nil
}

// ensureTemplateSupportsContainerConfig 校验模板类型是否允许配置容器。
// 纯仿真实验完全由 SimEngine 驱动，不允许保留任何容器、工具或初始化脚本配置。
func ensureTemplateSupportsContainerConfig(template *entity.ExperimentTemplate) error {
	if template == nil {
		return errcode.ErrTemplateNotFound
	}
	switch template.ExperimentType {
	case enum.ExperimentTypeReal, enum.ExperimentTypeMixed:
		return nil
	default:
		return errcode.ErrInvalidParams.WithMessage("纯仿真实验不允许配置容器镜像")
	}
}

// ensureTemplateSupportsInitScripts 校验模板类型是否允许配置初始化脚本。
func ensureTemplateSupportsInitScripts(template *entity.ExperimentTemplate) error {
	if template == nil {
		return errcode.ErrTemplateNotFound
	}
	switch template.ExperimentType {
	case enum.ExperimentTypeReal, enum.ExperimentTypeMixed:
		return nil
	default:
		return errcode.ErrInvalidParams.WithMessage("纯仿真实验不允许配置初始化脚本")
	}
}

// ensureTemplateSupportsSimScenes 校验模板类型是否允许配置仿真场景。
func ensureTemplateSupportsSimScenes(template *entity.ExperimentTemplate) error {
	if template == nil {
		return errcode.ErrTemplateNotFound
	}
	switch template.ExperimentType {
	case enum.ExperimentTypeSimulation, enum.ExperimentTypeMixed:
		return nil
	default:
		return errcode.ErrInvalidParams.WithMessage("真实环境实验不允许配置仿真场景")
	}
}

// ensureCheckpointTypeAllowed 校验检查点类型与实验类型的匹配关系。
// 文档要求纯仿真仅允许 SimEngine 状态断言和手动评分，真实环境仅允许脚本检查点和手动评分，混合实验允许全部类型。
func ensureCheckpointTypeAllowed(experimentType int16, checkType int16) error {
	switch experimentType {
	case enum.ExperimentTypeSimulation:
		if checkType == enum.CheckTypeScript {
			return errcode.ErrInvalidParams.WithMessage("纯仿真实验不允许使用脚本检查点")
		}
	case enum.ExperimentTypeReal:
		if checkType == enum.CheckTypeSimAssert {
			return errcode.ErrInvalidParams.WithMessage("真实环境实验不允许使用SimEngine状态断言检查点")
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

// ensureStructureEditable 检查模板是否允许修改结构性编排配置。
// 容器、初始化脚本、仿真场景、标签、角色等都会影响模板结构，
// 仅允许草稿模板或“已发布但尚未被课时引用”的模板修改。
func (s *templateSubService) ensureStructureEditable(ctx context.Context, templateID int64) error {
	tpl, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	switch tpl.Status {
	case enum.TemplateStatusDraft:
		return nil
	case enum.TemplateStatusPublished:
		hasCourseReferences, refErr := s.templateRepo.HasCourseReferences(ctx, templateID)
		if refErr != nil {
			return refErr
		}
		if !hasCourseReferences {
			return nil
		}
		return errcode.ErrInvalidParams.WithMessage("该模板已被课时引用，不允许修改容器、脚本、场景、标签或角色等结构配置")
	default:
		return errcode.ErrTemplateNotDraft
	}
}

// ensureCheckpointEditable 检查模板是否允许修改检查点。
// 文档要求已发布模板即使被课时引用，也允许继续调整检查点以优化后续教学，不影响已启动实例。
func (s *templateSubService) ensureCheckpointEditable(ctx context.Context, templateID int64) error {
	tpl, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	switch tpl.Status {
	case enum.TemplateStatusDraft, enum.TemplateStatusPublished:
		return nil
	default:
		return errcode.ErrTemplateNotDraft
	}
}

// validateUsableImageVersion 校验模板编排时引用的镜像版本是否可用。
// 教师只能选择状态为“正常”的镜像；待审核、已下架、审核拒绝都不能进入模板编排。
func (s *templateSubService) validateUsableImageVersion(ctx context.Context, imageVersionID int64) (*entity.ImageVersion, *entity.Image, error) {
	version, err := s.imageVersionRepo.GetByID(ctx, imageVersionID)
	if err != nil {
		return nil, nil, errcode.ErrImageVersionNotFound
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil {
		return nil, nil, errcode.ErrImageNotFound
	}
	if image.Status != enum.ImageStatusNormal {
		return nil, nil, errcode.ErrInvalidParams.WithMessage("当前镜像未处于正常状态，不可用于实验编排")
	}
	return version, image, nil
}

// ---------------------------------------------------------------------------
// 容器配置
// ---------------------------------------------------------------------------

// CreateContainer 添加容器配置
func (s *templateSubService) CreateContainer(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateContainerReq) (*dto.ContainerResp, error) {
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
		return nil, err
	}
	if err := ensureTemplateSupportsContainerConfig(template); err != nil {
		return nil, err
	}

	imageVersionID, _ := snowflake.ParseString(req.ImageVersionID)
	version, image, err := s.validateUsableImageVersion(ctx, imageVersionID)
	if err != nil {
		return nil, err
	}

	container := &entity.TemplateContainer{
		ID:              snowflake.Generate(),
		TemplateID:      templateID,
		ImageVersionID:  imageVersionID,
		ContainerName:   req.ContainerName,
		DeploymentScope: req.DeploymentScope,
		EnvVars:         datatypes.JSON(mustMarshalRawJSON(req.EnvVars)),
		Ports:           datatypes.JSON(mustMarshalRawJSON(req.Ports)),
		Volumes:         datatypes.JSON(mustMarshalRawJSON(req.Volumes)),
		CPULimit:        req.CPULimit,
		MemoryLimit:     req.MemoryLimit,
		DependsOn:       datatypes.JSON(mustMarshalRawJSON(req.DependsOn)),
		StartupOrder:    req.StartupOrder,
		IsPrimary:       req.IsPrimary,
	}
	if err := validateContainerDeploymentScope(template, req.DeploymentScope); err != nil {
		return nil, err
	}
	if req.RoleID != nil {
		if req.DeploymentScope == enum.ContainerDeploymentScopeShared {
			return nil, errcode.ErrInvalidParams.WithMessage("共享容器不允许绑定协作角色")
		}
		roleID, err := s.resolveContainerRoleID(ctx, templateID, req.RoleID)
		if err != nil {
			return nil, err
		}
		container.RoleID = roleID
	}

	if err := s.containerRepo.Create(ctx, container); err != nil {
		return nil, err
	}
	allContainers := []entity.TemplateContainer{*container}
	return buildTemplateContainerRespWithImage(container, version, image, allContainers), nil
}

// UpdateContainer 编辑容器配置
func (s *templateSubService) UpdateContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64, req *dto.UpdateContainerReq) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return errcode.ErrContainerNotFound
	}
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, container.TemplateID)
	if err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, container.TemplateID); err != nil {
		return err
	}
	if err := ensureTemplateSupportsContainerConfig(template); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.ImageVersionID != nil {
		versionID, _ := snowflake.ParseString(*req.ImageVersionID)
		if _, _, err := s.validateUsableImageVersion(ctx, versionID); err != nil {
			return err
		}
		fields["image_version_id"] = versionID
	}
	if req.ContainerName != nil {
		fields["container_name"] = *req.ContainerName
	}
	deploymentScope := container.DeploymentScope
	if req.DeploymentScope != nil {
		deploymentScope = *req.DeploymentScope
		if err := validateContainerDeploymentScope(template, deploymentScope); err != nil {
			return err
		}
		fields["deployment_scope"] = deploymentScope
	}
	if req.RoleID != nil {
		if deploymentScope == enum.ContainerDeploymentScopeShared {
			return errcode.ErrInvalidParams.WithMessage("共享容器不允许绑定协作角色")
		}
		roleID, err := s.resolveContainerRoleID(ctx, container.TemplateID, req.RoleID)
		if err != nil {
			return err
		}
		fields["role_id"] = roleID
	}
	if req.EnvVars != nil {
		fields["env_vars"] = datatypes.JSON(mustMarshalRawJSON(req.EnvVars))
	}
	if req.Ports != nil {
		fields["ports"] = datatypes.JSON(mustMarshalRawJSON(req.Ports))
	}
	if req.Volumes != nil {
		fields["volumes"] = datatypes.JSON(mustMarshalRawJSON(req.Volumes))
	}
	if req.CPULimit != nil {
		fields["cpu_limit"] = *req.CPULimit
	}
	if req.MemoryLimit != nil {
		fields["memory_limit"] = *req.MemoryLimit
	}
	if req.DependsOn != nil {
		fields["depends_on"] = datatypes.JSON(mustMarshalRawJSON(req.DependsOn))
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
	return nil
}

// DeleteContainer 删除容器配置
func (s *templateSubService) DeleteContainer(ctx context.Context, sc *svcctx.ServiceContext, containerID int64) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return errcode.ErrContainerNotFound
	}
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, container.TemplateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, container.TemplateID); err != nil {
		return err
	}
	if err := s.containerRepo.Delete(ctx, containerID); err != nil {
		return err
	}
	return nil
}

// ListContainers 获取模板的所有容器配置
func (s *templateSubService) ListContainers(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.ContainerResp, error) {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCheckpointEditable(ctx, templateID); err != nil {
		return nil, err
	}
	if err := ensureCheckpointTypeAllowed(template.ExperimentType, req.CheckType); err != nil {
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
		AssertionConfig: datatypes.JSON(req.AssertionConfig),
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, cp.TemplateID)
	if err != nil {
		return err
	}
	if err := s.ensureCheckpointEditable(ctx, cp.TemplateID); err != nil {
		return err
	}
	checkType := cp.CheckType
	if req.CheckType != nil {
		checkType = *req.CheckType
	}
	if err := ensureCheckpointTypeAllowed(template.ExperimentType, checkType); err != nil {
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
		fields["assertion_config"] = datatypes.JSON(req.AssertionConfig)
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, cp.TemplateID); err != nil {
		return err
	}
	if err := s.ensureCheckpointEditable(ctx, cp.TemplateID); err != nil {
		return err
	}
	return s.checkpointRepo.Delete(ctx, checkpointID)
}

// ListCheckpoints 获取模板的所有检查点
func (s *templateSubService) ListCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.CheckpointResp, error) {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return err
	}
	if err := s.ensureCheckpointEditable(ctx, templateID); err != nil {
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
		return nil, err
	}
	if err := ensureTemplateSupportsInitScripts(template); err != nil {
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
	return s.toInitScriptResp(script), nil
}

// UpdateInitScript 编辑初始化脚本
func (s *templateSubService) UpdateInitScript(ctx context.Context, sc *svcctx.ServiceContext, scriptID int64, req *dto.UpdateInitScriptReq) error {
	script, err := s.initScriptRepo.GetByID(ctx, scriptID)
	if err != nil {
		return errcode.ErrInitScriptNotFound
	}
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, script.TemplateID)
	if err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, script.TemplateID); err != nil {
		return err
	}
	if err := ensureTemplateSupportsInitScripts(template); err != nil {
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, script.TemplateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, script.TemplateID); err != nil {
		return err
	}
	return s.initScriptRepo.Delete(ctx, scriptID)
}

// ListInitScripts 获取模板的所有初始化脚本
func (s *templateSubService) ListInitScripts(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.InitScriptResp, error) {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	DataSourceMode int16           `json:"data_source_mode"`
}

// CreateSimScene 添加仿真场景到模板
func (s *templateSubService) CreateSimScene(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateTemplateSimSceneReq) (*dto.TemplateSimSceneResp, error) {
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
		return nil, err
	}
	if err := ensureTemplateSupportsSimScenes(template); err != nil {
		return nil, err
	}

	scenarioID, _ := snowflake.ParseString(req.ScenarioID)
	_, err = s.scenarioRepo.GetByID(ctx, scenarioID)
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
		Config:           datatypes.JSON(configBytes),
		DataSourceConfig: datatypes.JSON(req.DataSourceConfig),
		LayoutPosition:   datatypes.JSON(req.LayoutPosition),
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, scene.TemplateID)
	if err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, scene.TemplateID); err != nil {
		return err
	}
	if err := ensureTemplateSupportsSimScenes(template); err != nil {
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
		fields["config"] = datatypes.JSON(configBytes)
	}

	if req.DataSourceConfig != nil {
		fields["data_source_config"] = datatypes.JSON(req.DataSourceConfig)
	}
	if req.LayoutPosition != nil {
		fields["layout_position"] = datatypes.JSON(req.LayoutPosition)
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, scene.TemplateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, scene.TemplateID); err != nil {
		return err
	}
	return s.simSceneRepo.Delete(ctx, simSceneID)
}

// ListSimScenes 获取模板的所有仿真场景配置
func (s *templateSubService) ListSimScenes(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.TemplateSimSceneResp, error) {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID)
	if err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
		return err
	}
	if err := ensureTemplateSupportsSimScenes(template); err != nil {
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
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
	if _, err := ensureTemplateReadAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
	if err := s.ensureStructureEditable(ctx, templateID); err != nil {
		return nil, err
	}
	if err := s.ensureCollaborativeTemplate(ctx, templateID); err != nil {
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, role.TemplateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, role.TemplateID); err != nil {
		return err
	}
	if err := s.ensureCollaborativeTemplate(ctx, role.TemplateID); err != nil {
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, role.TemplateID); err != nil {
		return err
	}
	if err := s.ensureStructureEditable(ctx, role.TemplateID); err != nil {
		return err
	}
	hasReferences, err := s.roleRepo.HasReferences(ctx, roleID)
	if err != nil {
		return err
	}
	if hasReferences {
		return errcode.ErrInvalidParams.WithMessage("该角色已被容器配置或分组成员引用，不能删除")
	}
	return s.roleRepo.Delete(ctx, roleID)
}

// ListRoles 获取模板的所有角色
func (s *templateSubService) ListRoles(ctx context.Context, sc *svcctx.ServiceContext, templateID int64) ([]*dto.RoleResp, error) {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, templateID); err != nil {
		return nil, err
	}
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
	var version *entity.ImageVersion
	if loadedVersion, err := s.imageVersionRepo.GetByID(ctx, c.ImageVersionID); err == nil {
		version = loadedVersion
	}
	var image *entity.Image
	if version != nil {
		if loadedImage, imageErr := s.imageRepo.GetByID(ctx, version.ImageID); imageErr == nil {
			image = loadedImage
		}
	}
	allContainers := make([]entity.TemplateContainer, 0)
	if containers, err := s.containerRepo.ListByTemplateID(ctx, c.TemplateID); err == nil {
		allContainers = make([]entity.TemplateContainer, 0, len(containers))
		for _, item := range containers {
			if item == nil {
				continue
			}
			allContainers = append(allContainers, *item)
		}
	}
	return buildTemplateContainerRespWithImage(c, version, image, allContainers)
}

// toCheckpointResp 转换检查点实体为响应
func (s *templateSubService) toCheckpointResp(cp *entity.TemplateCheckpoint) *dto.CheckpointResp {
	return buildTemplateCheckpointResp(cp)
}

// toInitScriptResp 转换初始化脚本实体为响应
func (s *templateSubService) toInitScriptResp(script *entity.TemplateInitScript) *dto.InitScriptResp {
	return buildTemplateInitScriptResp(script)
}

// toSimSceneResp 转换仿真场景配置实体为响应
func (s *templateSubService) toSimSceneResp(ctx context.Context, scene *entity.TemplateSimScene) *dto.TemplateSimSceneResp {
	var linkGroup *entity.SimLinkGroup
	if scene.LinkGroupID != nil {
		if loadedLinkGroup, err := s.linkGroupRepo.GetByID(ctx, *scene.LinkGroupID); err == nil {
			linkGroup = loadedLinkGroup
		}
	}

	var scenario *entity.SimScenario
	if loadedScenario, err := s.scenarioRepo.GetByID(ctx, scene.ScenarioID); err == nil {
		scenario = loadedScenario
	}

	return buildTemplateSimSceneRespWithRelations(scene, scenario, linkGroup)
}

// toTagResp 转换标签实体为响应
func (s *templateSubService) toTagResp(t *entity.Tag) *dto.TagResp {
	return buildTagResp(t)
}

// toRoleResp 转换角色实体为响应
func (s *templateSubService) toRoleResp(r *entity.TemplateRole) *dto.RoleResp {
	return buildTemplateRoleResp(r)
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// containsKeyword 简单关键词匹配（不区分大小写）
func containsKeyword(s, keyword string) bool {
	return len(keyword) > 0 && len(s) > 0 &&
		(strings.Contains(strings.ToLower(s), strings.ToLower(keyword)))
}
