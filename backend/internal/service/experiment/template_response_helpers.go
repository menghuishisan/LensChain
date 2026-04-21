// template_response_helpers.go
// 模块04 — 实验环境：模板响应转换共享辅助
// 负责模板及其子资源的基础 DTO 转换，供 templateService 与 templateSubService 统一复用

package experiment

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// buildTemplateContainerRespBase 构建容器配置的基础响应。
// 模板接口返回的是“教师看到的有效模板配置”，包含镜像默认值与条件变量结果，
// 但不包含仅在实例运行时注入的服务发现变量。
func buildTemplateContainerRespBase(
	c *entity.TemplateContainer,
	version *entity.ImageVersion,
	image *entity.Image,
	allContainers []entity.TemplateContainer,
) *dto.ContainerResp {
	envVars, ports, volumes, cpuLimit, memoryLimit := buildTemplateContainerDisplayConfig(c, version, image, allContainers)
	dependsOn := make([]string, 0)
	if len(c.DependsOn) > 0 {
		_ = json.Unmarshal(c.DependsOn, &dependsOn)
	}

	resp := &dto.ContainerResp{
		ID:             strconv.FormatInt(c.ID, 10),
		TemplateID:     strconv.FormatInt(c.TemplateID, 10),
		ImageVersionID: strconv.FormatInt(c.ImageVersionID, 10),
		ContainerName:  c.ContainerName,
		EnvVars:        envVars,
		Ports:          ports,
		Volumes:        volumes,
		CPULimit:       cpuLimit,
		MemoryLimit:    memoryLimit,
		DependsOn:      dependsOn,
		StartupOrder:   c.StartupOrder,
		IsPrimary:      c.IsPrimary,
	}
	if c.RoleID != nil {
		roleID := strconv.FormatInt(*c.RoleID, 10)
		resp.RoleID = &roleID
	}
	return resp
}

// buildTemplateContainerRespWithImage 构建带镜像版本摘要的容器响应。
func buildTemplateContainerRespWithImage(
	c *entity.TemplateContainer,
	version *entity.ImageVersion,
	image *entity.Image,
	allContainers []entity.TemplateContainer,
) *dto.ContainerResp {
	resp := buildTemplateContainerRespBase(c, version, image, allContainers)
	if version == nil {
		return resp
	}

	resp.ImageVersion = &dto.ContainerImageVersionResp{
		ID:      strconv.FormatInt(version.ID, 10),
		Version: version.Version,
	}
	if image != nil {
		resp.ImageVersion.ImageName = image.Name
		resp.ImageVersion.ImageDisplayName = image.DisplayName
		resp.ImageVersion.IconURL = image.IconURL
	}
	return resp
}

// buildTemplateContainerDisplayConfig 计算模板编辑视角下的有效容器配置。
func buildTemplateContainerDisplayConfig(
	container *entity.TemplateContainer,
	version *entity.ImageVersion,
	image *entity.Image,
	allContainers []entity.TemplateContainer,
) ([]dto.ContainerEnvVarItem, []dto.ContainerPortItem, []dto.ContainerVolumeItem, *string, *string) {
	if container == nil {
		return []dto.ContainerEnvVarItem{}, []dto.ContainerPortItem{}, []dto.ContainerVolumeItem{}, nil, nil
	}

	envMap := make(map[string]string)
	mergeDefaultEnvVars(envMap, image)
	_ = mergeConditionalEnvVars(envMap, image, allContainers)
	for key, value := range parseExplicitEnvVars(json.RawMessage(container.EnvVars)) {
		envMap[key] = value
	}

	envVars := make([]dto.ContainerEnvVarItem, 0, len(envMap))
	envKeys := make([]string, 0, len(envMap))
	for key := range envMap {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		envVars = append(envVars, dto.ContainerEnvVarItem{Key: key, Value: envMap[key]})
	}

	portSpecs := mergePorts(image, json.RawMessage(container.Ports))
	ports := make([]dto.ContainerPortItem, 0, len(portSpecs))
	for _, port := range portSpecs {
		ports = append(ports, dto.ContainerPortItem{
			Container: port.ContainerPort,
			Protocol:  port.Protocol,
		})
	}

	volumeSpecs := mergeVolumes(image, json.RawMessage(container.Volumes))
	volumes := make([]dto.ContainerVolumeItem, 0, len(volumeSpecs))
	for _, volume := range volumeSpecs {
		volumes = append(volumes, dto.ContainerVolumeItem{
			HostPath:      "",
			ContainerPath: volume.MountPath,
		})
	}

	cpuLimit := container.CPULimit
	if cpuLimit == nil && version != nil && version.MinCPU != nil {
		cpuLimit = version.MinCPU
	}
	memoryLimit := container.MemoryLimit
	if memoryLimit == nil && version != nil && version.MinMemory != nil {
		memoryLimit = version.MinMemory
	}

	return envVars, ports, volumes, cpuLimit, memoryLimit
}

// buildTemplateCheckpointResp 构建检查点基础响应。
func buildTemplateCheckpointResp(cp *entity.TemplateCheckpoint) *dto.CheckpointResp {
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
		AssertionConfig: json.RawMessage(cp.AssertionConfig),
		Score:           cp.Score,
		Scope:           cp.Scope,
		ScopeText:       enum.GetCheckpointScopeText(cp.Scope),
		SortOrder:       cp.SortOrder,
	}
}

// buildTemplateInitScriptResp 构建初始化脚本基础响应。
func buildTemplateInitScriptResp(script *entity.TemplateInitScript) *dto.InitScriptResp {
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

// buildTemplateSimSceneRespBase 构建模板仿真场景基础响应，并按统一语义拆分 Config。
func buildTemplateSimSceneRespBase(scene *entity.TemplateSimScene) *dto.TemplateSimSceneResp {
	resp := &dto.TemplateSimSceneResp{
		ID:               strconv.FormatInt(scene.ID, 10),
		TemplateID:       strconv.FormatInt(scene.TemplateID, 10),
		DataSourceConfig: json.RawMessage(scene.DataSourceConfig),
		LayoutPosition:   json.RawMessage(scene.LayoutPosition),
	}
	cfg := decodeSimSceneConfig(json.RawMessage(scene.Config))
	resp.SceneParams = cfg.SceneParams
	resp.InitialState = cfg.InitialState
	resp.DataSourceMode = cfg.DataSourceMode
	resp.DataSourceModeText = enum.GetDataSourceModeText(cfg.DataSourceMode)
	if scene.LinkGroupID != nil {
		linkGroupID := strconv.FormatInt(*scene.LinkGroupID, 10)
		resp.LinkGroupID = &linkGroupID
	}
	return resp
}

// buildTemplateSimSceneRespWithRelations 构建带场景摘要和联动组名称的仿真场景响应。
func buildTemplateSimSceneRespWithRelations(scene *entity.TemplateSimScene, scenario *entity.SimScenario, linkGroup *entity.SimLinkGroup) *dto.TemplateSimSceneResp {
	resp := buildTemplateSimSceneRespBase(scene)
	if linkGroup != nil {
		resp.LinkGroupName = &linkGroup.Name
	}
	if scenario == nil {
		return resp
	}

	brief := &dto.ScenarioBrief{
		ID:              strconv.FormatInt(scenario.ID, 10),
		Name:            scenario.Name,
		Code:            scenario.Code,
		Category:        scenario.Category,
		CategoryText:    enum.GetSceneCategoryText(scenario.Category),
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
	return resp
}

// buildTemplateListItemsWithMetadata 根据模板主表和批量关联数据构建列表项。
func buildTemplateListItemsWithMetadata(
	templates []*entity.ExperimentTemplate,
	containers []*entity.TemplateContainer,
	checkpoints []*entity.TemplateCheckpoint,
	templateTags []*entity.TemplateTag,
	tags []*entity.Tag,
) []*dto.TemplateListItem {
	containerCounts := make(map[int64]int, len(templates))
	for _, container := range containers {
		if container == nil {
			continue
		}
		containerCounts[container.TemplateID]++
	}

	checkpointCounts := make(map[int64]int, len(templates))
	for _, checkpoint := range checkpoints {
		if checkpoint == nil {
			continue
		}
		checkpointCounts[checkpoint.TemplateID]++
	}

	tagMap := make(map[int64]*entity.Tag, len(tags))
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		tagMap[tag.ID] = tag
	}

	templateTagMap := make(map[int64][]dto.TagResp, len(templates))
	for _, relation := range templateTags {
		if relation == nil {
			continue
		}
		tag, ok := tagMap[relation.TagID]
		if !ok {
			continue
		}
		templateTagMap[relation.TemplateID] = append(templateTagMap[relation.TemplateID], *buildTagResp(tag))
	}

	items := make([]*dto.TemplateListItem, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		topologyMode := int16(0)
		if template.TopologyMode != nil {
			topologyMode = *template.TopologyMode
		}
		maxDuration := 0
		if template.MaxDuration != nil {
			maxDuration = *template.MaxDuration
		}

		items = append(items, &dto.TemplateListItem{
			ID:                 strconv.FormatInt(template.ID, 10),
			Title:              template.Title,
			ExperimentType:     template.ExperimentType,
			ExperimentTypeText: enum.GetExperimentTypeText(template.ExperimentType),
			TopologyMode:       topologyMode,
			TopologyModeText:   enum.GetTopologyModeText(topologyMode),
			JudgeMode:          template.JudgeMode,
			JudgeModeText:      enum.GetJudgeModeText(template.JudgeMode),
			TotalScore:         int(template.TotalScore),
			MaxDuration:        maxDuration,
			IsShared:           template.IsShared,
			Status:             template.Status,
			StatusText:         enum.GetTemplateStatusText(template.Status),
			ContainerCount:     containerCounts[template.ID],
			CheckpointCount:    checkpointCounts[template.ID],
			Tags:               templateTagMap[template.ID],
			CreatedAt:          template.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          template.UpdatedAt.Format(time.RFC3339),
		})
	}
	return items
}

// buildTemplateRoleResp 构建角色基础响应。
func buildTemplateRoleResp(role *entity.TemplateRole) *dto.RoleResp {
	return &dto.RoleResp{
		ID:          strconv.FormatInt(role.ID, 10),
		TemplateID:  strconv.FormatInt(role.TemplateID, 10),
		RoleName:    role.RoleName,
		Description: role.Description,
		MaxMembers:  role.MaxMembers,
	}
}

// buildTagResp 构建标签基础响应。
func buildTagResp(tag *entity.Tag) *dto.TagResp {
	return &dto.TagResp{
		ID:       strconv.FormatInt(tag.ID, 10),
		Name:     tag.Name,
		Category: tag.Category,
	}
}
