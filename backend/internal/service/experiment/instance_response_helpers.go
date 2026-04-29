// instance_response_helpers.go
// 模块04 — 实验环境：实例响应转换共享辅助
// 负责实例详情中的学生摘要和容器运行态 DTO 组装，保持 service 层聚合逻辑集中可复用。

package experiment

import (
	"context"
	"strconv"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// getInstanceUserSummary 获取实例详情所需的学生摘要。
// 实例详情必须通过模块01注入的摘要接口拿到姓名和学号，避免同时保留多条用户信息来源。
func (s *instanceService) getInstanceUserSummary(ctx context.Context, userID int64) ExperimentUserSummary {
	if s.userSummaryQuerier != nil {
		if summary := s.userSummaryQuerier.GetUserSummary(ctx, userID); summary != nil {
			return *summary
		}
	}
	return ExperimentUserSummary{
		UserID: userID,
	}
}

// buildInstanceContainerItems 构建实例详情容器列表。
// 运行态容器只保存模板容器ID，镜像名称和版本必须通过模板容器、镜像版本、镜像主表逐级补齐。
func (s *instanceService) buildInstanceContainerItems(
	ctx context.Context,
	containers []*entity.InstanceContainer,
	templateContainers []*entity.TemplateContainer,
) []dto.InstanceContainerItem {
	templateContainerMap := make(map[int64]*entity.TemplateContainer, len(templateContainers))
	for _, templateContainer := range templateContainers {
		if templateContainer == nil {
			continue
		}
		templateContainerMap[templateContainer.ID] = templateContainer
	}

	versionMap := s.loadImageVersionMap(ctx, templateContainerMap)
	imageMap := s.loadImageMap(ctx, versionMap)

	items := make([]dto.InstanceContainerItem, 0, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		item := buildInstanceContainerItemBase(container)
		if templateContainer := templateContainerMap[container.TemplateContainerID]; templateContainer != nil {
			if version := versionMap[templateContainer.ImageVersionID]; version != nil {
				item.ImageVersion = version.Version
				if image := imageMap[version.ImageID]; image != nil {
					item.ImageName = image.Name
				}
			}
		}
		items = append(items, item)
	}
	return items
}

// buildInstanceContainerItemBase 构建实例容器运行态基础字段。
func buildInstanceContainerItemBase(container *entity.InstanceContainer) dto.InstanceContainerItem {
	return dto.InstanceContainerItem{
		ID:            strconv.FormatInt(container.ID, 10),
		ContainerName: container.ContainerName,
		Status:        container.Status,
		StatusText:    enum.GetContainerStatusText(container.Status),
		InternalIP:    container.InternalIP,
		CPUUsage:      container.CPUUsage,
		MemoryUsage:   container.MemoryUsage,
		ToolKind:      container.ToolKind,
	}
}

// buildInstanceToolItems 构建实例详情工具列表（从容器中筛选 tool_kind 非空且 proxy_url 已签发的）。
func (s *instanceService) buildInstanceToolItems(
	_ context.Context,
	containers []*entity.InstanceContainer,
) []dto.InstanceToolItem {
	items := make([]dto.InstanceToolItem, 0)
	for _, container := range containers {
		if container == nil || container.ToolKind == nil || *container.ToolKind == "" {
			continue
		}
		if container.ProxyURL == nil || *container.ProxyURL == "" {
			continue
		}
		items = append(items, dto.InstanceToolItem{
			Kind:          *container.ToolKind,
			ContainerID:   strconv.FormatInt(container.ID, 10),
			ContainerName: container.ContainerName,
			ProxyURL:      *container.ProxyURL,
			Status:        container.Status,
			StatusText:    enum.GetContainerStatusText(container.Status),
		})
	}
	return items
}

// loadImageVersionMap 批量按需加载模板容器引用的镜像版本。
func (s *instanceService) loadImageVersionMap(ctx context.Context, templateContainers map[int64]*entity.TemplateContainer) map[int64]*entity.ImageVersion {
	versionMap := make(map[int64]*entity.ImageVersion, len(templateContainers))
	if s.imageVersionRepo == nil {
		return versionMap
	}
	for _, templateContainer := range templateContainers {
		if templateContainer == nil || templateContainer.ImageVersionID == 0 {
			continue
		}
		if _, exists := versionMap[templateContainer.ImageVersionID]; exists {
			continue
		}
		version, err := s.imageVersionRepo.GetByID(ctx, templateContainer.ImageVersionID)
		if err != nil || version == nil {
			continue
		}
		versionMap[templateContainer.ImageVersionID] = version
	}
	return versionMap
}

// loadImageMap 批量按需加载镜像版本关联的镜像主表信息。
func (s *instanceService) loadImageMap(ctx context.Context, versions map[int64]*entity.ImageVersion) map[int64]*entity.Image {
	imageMap := make(map[int64]*entity.Image, len(versions))
	if s.imageRepo == nil {
		return imageMap
	}
	for _, version := range versions {
		if version == nil || version.ImageID == 0 {
			continue
		}
		if _, exists := imageMap[version.ImageID]; exists {
			continue
		}
		image, err := s.imageRepo.GetByID(ctx, version.ImageID)
		if err != nil || image == nil {
			continue
		}
		imageMap[version.ImageID] = image
	}
	return imageMap
}
