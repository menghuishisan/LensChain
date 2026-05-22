// instance_response_helpers.go
// 模块04 — 实验环境：实例响应转换共享辅助
// 负责实例详情中的学生摘要和容器运行态 DTO 组装，保持 service 层聚合逻辑集中可复用。

package experiment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// deriveContainerToolMeta 按 docs/modules/04-实验环境/02-数据库设计.md §2.16 派生
// instance_containers.tool_kind 与 proxy_url：
//   - tool_kind：沿 template_container.image_version_id → image_versions.image_id → images.tool_kind
//     直接读取，service 层不做任何镜像名→kind 的硬编码映射；
//   - proxy_url：仅工具镜像（tool_kind 非空）签发，格式 /instance/{id}/{tool_kind}/。
//     **始终为相对路径，绝不嵌入 host**——origin 由前端 `NEXT_PUBLIC_TOOL_PROXY_BASE_URL`
//     解析（services/experimentToolProxy.ts::resolveToolProxyURL）：生产留空走 Ingress
//     同源；本地开发设为 http://localhost:8080，iframe / WS 直连后端，绕开 Next dev
//     rewrite 对 trailingSlash 与 WS upgrade 的路径归一化限制。`localhost:3000` 与
//     `localhost:8080` 是 same-site（eTLD+1=localhost），SameSite=Lax cookie 在跨源
//     same-site iframe / WS 中仍会被发送，鉴权链路与生产保持一致。
//   - 末尾斜杠保证 router 路由 *proxy_path 在裸路径下也能命中根 "/"。token 不放 URL，
//     前端必须先调 POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie
//     拿 cookie，由 ToolProxyAuth 中间件鉴权（handler/experiment/tool_proxy.go）。
//
// 任一环节失败或非工具镜像，返回 (nil, nil)。
func (s *instanceService) deriveContainerToolMeta(ctx context.Context, imageVersionID int64, instanceID int64) (*string, *string) {
	if imageVersionID == 0 || s.imageVersionRepo == nil || s.imageRepo == nil {
		return nil, nil
	}
	version, err := s.imageVersionRepo.GetByID(ctx, imageVersionID)
	if err != nil || version == nil {
		return nil, nil
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil || image == nil {
		return nil, nil
	}
	if image.ToolKind == nil || strings.TrimSpace(*image.ToolKind) == "" {
		return nil, nil
	}
	kind := strings.TrimSpace(*image.ToolKind)
	url := fmt.Sprintf("/instance/%d/%s/", instanceID, kind)
	return &kind, &url
}

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
