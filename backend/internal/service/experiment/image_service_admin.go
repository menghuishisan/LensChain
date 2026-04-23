// image_service_admin.go
// 模块04 — 实验环境：学校镜像与镜像预拉取业务逻辑
// 负责学校管理员镜像列表、镜像预拉取状态查询与触发

package experiment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// ListSchoolImages 获取本校镜像列表。
func (s *imageService) ListSchoolImages(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SchoolImageListReq) ([]*dto.ImageListItem, int64, error) {
	if !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, 0, errcode.ErrForbidden
	}

	categoryID, _ := snowflake.ParseString(req.CategoryID)
	images, total, err := s.imageRepo.ListBySchoolID(ctx, &experimentrepo.SchoolImageListParams{
		SchoolID:   sc.SchoolID,
		Keyword:    req.Keyword,
		CategoryID: categoryID,
		Status:     req.Status,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.ImageListItem, 0, len(images))
	for _, image := range images {
		items = append(items, s.toImageListItem(ctx, image))
	}
	return items, total, nil
}

// GetImagePullStatus 获取镜像预拉取状态。
func (s *imageService) GetImagePullStatus(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImagePullStatusListReq) (*dto.ImagePullStatusListResp, int64, error) {
	if !sc.IsSuperAdmin() {
		return nil, 0, errcode.ErrForbidden
	}
	if s.versionRepo == nil || s.imageRepo == nil || s.k8sSvc == nil {
		return nil, 0, errcode.ErrInternal.WithMessage("镜像预拉取服务未初始化")
	}

	images, _, err := s.imageRepo.List(ctx, &experimentrepo.ImageListParams{
		Keyword:    req.ImageName,
		SourceType: enum.ImageSourceTypeOfficial,
		Status:     enum.ImageStatusNormal,
		Page:       1,
		PageSize:   10000,
		SortBy:     "created_at",
		SortOrder:  "desc",
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.ImagePullStatusItem, 0)
	nodeSet := make(map[string]struct{})
	fullyPulled := 0
	partiallyPulled := 0
	notPulled := 0
	totalNodeEntries := 0
	pulledNodeEntries := 0

	for _, image := range images {
		versions, versionErr := s.versionRepo.ListByImageID(ctx, image.ID)
		if versionErr != nil {
			return nil, 0, versionErr
		}
		if len(versions) == 0 {
			continue
		}

		for _, version := range versions {
			nodeStatuses, statusErr := s.getImagePullNodeStatuses(ctx, version.RegistryURL)
			if statusErr != nil {
				return nil, 0, statusErr
			}
			filteredNodes := make([]dto.ImagePullNodeStatus, 0, len(nodeStatuses))
			pulledCount := 0
			for _, nodeStatus := range nodeStatuses {
				if req.NodeName != "" && nodeStatus.NodeName != req.NodeName {
					continue
				}
				if req.Status > 0 && nodeStatus.Status != req.Status {
					continue
				}
				nodeSet[nodeStatus.NodeName] = struct{}{}
				totalNodeEntries++
				if nodeStatus.Status == 1 {
					pulledCount++
					pulledNodeEntries++
				}
				filteredNodes = append(filteredNodes, nodeStatus)
			}
			if len(filteredNodes) == 0 {
				continue
			}

			switch {
			case pulledCount == len(filteredNodes):
				fullyPulled++
			case pulledCount == 0:
				notPulled++
			default:
				partiallyPulled++
			}

			items = append(items, dto.ImagePullStatusItem{
				ImageName:      image.Name,
				ImageVersion:   version.Version,
				RegistryURL:    version.RegistryURL,
				SourceType:     image.SourceType,
				SourceTypeText: enum.GetImageSourceTypeText(image.SourceType),
				Nodes:          filteredNodes,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ImageName == items[j].ImageName {
			return items[i].ImageVersion < items[j].ImageVersion
		}
		return items[i].ImageName < items[j].ImageName
	})

	totalItems := len(items)
	total := int64(totalItems)
	page, pageSize := normalizePage(req.Page, req.PageSize)
	items = paginatePullStatusItems(items, page, pageSize)

	return &dto.ImagePullStatusListResp{
		Summary: dto.ImagePullStatusSummary{
			TotalImages:     totalItems,
			TotalNodes:      len(nodeSet),
			FullyPulled:     fullyPulled,
			PartiallyPulled: partiallyPulled,
			NotPulled:       notPulled,
			CompletionRate:  calculatePullCompletionRate(pulledNodeEntries, totalNodeEntries),
		},
		Items: items,
	}, total, nil
}

// TriggerImagePull 触发镜像预拉取任务。
func (s *imageService) TriggerImagePull(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TriggerImagePullReq) (*dto.TriggerImagePullResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	if s.versionRepo == nil || s.k8sSvc == nil {
		return nil, errcode.ErrInternal.WithMessage("镜像预拉取服务未初始化")
	}

	targetNodes, err := s.collectTargetNodes(ctx, req.TargetNodes)
	if err != nil {
		return nil, err
	}
	versions, labels, err := s.collectPrePullVersions(ctx, req.ImageIDs)
	if err != nil {
		return nil, err
	}

	totalJobs := 0
	for _, version := range versions {
		if !req.Force {
			remainingNodes, remainingErr := s.filterNodesNeedingPull(ctx, version.RegistryURL, targetNodes)
			if remainingErr != nil {
				return nil, remainingErr
			}
			if len(remainingNodes) == 0 {
				continue
			}
			targetNodesForVersion := remainingNodes
			if err := s.k8sSvc.PrePullImage(ctx, version.RegistryURL, targetNodesForVersion); err != nil {
				return nil, err
			}
			totalJobs += len(targetNodesForVersion)
			continue
		}
		if err := s.k8sSvc.PrePullImage(ctx, version.RegistryURL, targetNodes); err != nil {
			return nil, err
		}
		totalJobs += len(targetNodes)
	}

	return &dto.TriggerImagePullResp{
		TaskID:      fmt.Sprintf("pull-task-%s-%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%1000),
		TotalJobs:   totalJobs,
		Images:      labels,
		TargetNodes: targetNodes,
		Status:      "running",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// getImagePullNodeStatuses 查询单个镜像版本在各节点的拉取状态。
func (s *imageService) getImagePullNodeStatuses(ctx context.Context, imageURL string) ([]dto.ImagePullNodeStatus, error) {
	if s.k8sSvc == nil {
		return []dto.ImagePullNodeStatus{}, nil
	}

	statuses, err := s.k8sSvc.GetImagePullStatus(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	resp := make([]dto.ImagePullNodeStatus, 0, len(statuses))
	for _, status := range statuses {
		var pulledAt *string
		if status.PulledAt != nil {
			value := status.PulledAt.UTC().Format(time.RFC3339)
			pulledAt = &value
		}
		resp = append(resp, dto.ImagePullNodeStatus{
			NodeName:      status.NodeName,
			Status:        mapPullStatusCode(status.Status, status.Progress, status.Error),
			StatusText:    mapPullStatusText(status.Status, status.Progress, status.Error),
			PulledAt:      pulledAt,
			NodeCacheSize: status.NodeCacheSize,
		})
	}
	return resp, nil
}

// calculatePullCompletionRate 计算镜像预拉取整体完成率。
func calculatePullCompletionRate(pulled, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int((float64(pulled)/float64(total))*10000+0.5)) / 100
}

// collectPrePullVersions 收集需要预拉取的镜像默认版本。
func (s *imageService) collectPrePullVersions(ctx context.Context, imageIDs []string) ([]*entity.ImageVersion, []string, error) {
	versions := make([]*entity.ImageVersion, 0)
	labels := make([]string, 0)

	if len(imageIDs) == 0 {
		images, _, err := s.imageRepo.List(ctx, &experimentrepo.ImageListParams{
			SourceType: enum.ImageSourceTypeOfficial,
			Status:     enum.ImageStatusNormal,
			Page:       1,
			PageSize:   10000,
			SortBy:     "created_at",
			SortOrder:  "desc",
		})
		if err != nil {
			return nil, nil, err
		}
		for _, image := range images {
			version, versionErr := s.pickDefaultVersion(ctx, image.ID)
			if versionErr != nil {
				return nil, nil, versionErr
			}
			if version == nil {
				continue
			}
			versions = append(versions, version)
			labels = append(labels, image.Name+":"+version.Version)
		}
		return versions, labels, nil
	}

	for _, rawID := range imageIDs {
		imageID, err := snowflake.ParseString(rawID)
		if err != nil {
			return nil, nil, errcode.ErrInvalidParams.WithMessage("镜像ID无效")
		}
		image, err := s.imageRepo.GetByID(ctx, imageID)
		if err != nil {
			return nil, nil, errcode.ErrImageNotFound
		}
		if image.SourceType != enum.ImageSourceTypeOfficial {
			return nil, nil, errcode.ErrInvalidParams.WithMessage("手动预拉取仅支持官方镜像")
		}
		if image.Status != enum.ImageStatusNormal {
			return nil, nil, errcode.ErrInvalidParams.WithMessage("仅正常状态的官方镜像可执行预拉取")
		}
		version, versionErr := s.pickDefaultVersion(ctx, image.ID)
		if versionErr != nil {
			return nil, nil, versionErr
		}
		if version == nil {
			continue
		}
		versions = append(versions, version)
		labels = append(labels, image.Name+":"+version.Version)
	}
	return versions, labels, nil
}

// pickDefaultVersion 选择镜像默认版本，没有默认版本时回退到首个版本。
func (s *imageService) pickDefaultVersion(ctx context.Context, imageID int64) (*entity.ImageVersion, error) {
	version, err := s.versionRepo.GetDefaultByImageID(ctx, imageID)
	if err == nil && version != nil {
		return version, nil
	}
	versions, listErr := s.versionRepo.ListByImageID(ctx, imageID)
	if listErr != nil {
		return nil, listErr
	}
	if len(versions) == 0 {
		return nil, nil
	}
	return versions[0], nil
}

// filterNodesNeedingPull 过滤指定镜像仍需预拉取的节点。
func (s *imageService) filterNodesNeedingPull(ctx context.Context, imageURL string, targetNodes []string) ([]string, error) {
	nodeStatuses, err := s.getImagePullNodeStatuses(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	pulledNodes := make(map[string]struct{})
	for _, status := range nodeStatuses {
		if status.Status == 1 {
			pulledNodes[status.NodeName] = struct{}{}
		}
	}

	result := make([]string, 0, len(targetNodes))
	for _, node := range targetNodes {
		if _, ok := pulledNodes[node]; ok {
			continue
		}
		result = append(result, node)
	}
	return result, nil
}

// mapPullStatusCode 将 K8s 拉取状态映射为接口定义的状态码。
func mapPullStatusCode(status, progress, errText string) int16 {
	switch {
	case errText != "":
		return 3
	case progress != "":
		return 2
	case strings.EqualFold(status, "pulled"):
		return 1
	default:
		return 4
	}
}

// mapPullStatusText 将 K8s 拉取状态映射为接口定义的状态文案。
func mapPullStatusText(status, progress, errText string) string {
	switch mapPullStatusCode(status, progress, errText) {
	case 1:
		return "已拉取"
	case 2:
		return "拉取中"
	case 3:
		return "拉取失败"
	default:
		return "未拉取"
	}
}

// paginatePullStatusItems 对预拉取状态条目进行分页。
func paginatePullStatusItems(items []dto.ImagePullStatusItem, page, pageSize int) []dto.ImagePullStatusItem {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []dto.ImagePullStatusItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// normalizePage 规范化分页参数。
func normalizePage(page, pageSize int) (int, int) {
	return pagination.NormalizeValues(page, pageSize)
}

// collectTargetNodes 收集预拉取目标节点。
func (s *imageService) collectTargetNodes(ctx context.Context, requested []string) ([]string, error) {
	if s.k8sSvc == nil {
		return nil, errcode.ErrInternal.WithMessage("K8s 服务未初始化")
	}
	if len(requested) > 0 {
		return requested, nil
	}
	nodes, err := s.k8sSvc.GetNodeStatus(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, node.Name)
	}
	sort.Strings(result)
	return result, nil
}
