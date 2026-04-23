// image_service.go
// 模块04 — 实验环境：镜像管理业务逻辑
// 负责镜像分类CRUD、镜像CRUD、审核、版本管理、配置模板、文档
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
	"go.uber.org/zap"
)

// ImageService 镜像管理服务接口
type ImageService interface {
	// 镜像分类
	ListCategories(ctx context.Context, sc *svcctx.ServiceContext) ([]*dto.ImageCategoryResp, error)
	CreateCategory(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateImageCategoryReq) (*dto.ImageCategoryResp, error)
	UpdateCategory(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateImageCategoryReq) error
	DeleteCategory(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	// 镜像CRUD
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateImageReq) (string, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateImageReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImageListReq) ([]*dto.ImageListItem, int64, error)
	ListSchoolImages(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SchoolImageListReq) ([]*dto.ImageListItem, int64, error)

	// 审核
	Review(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewImageReq) error

	// 版本管理
	CreateVersion(ctx context.Context, sc *svcctx.ServiceContext, imageID int64, req *dto.CreateImageVersionReq) (string, error)
	UpdateVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64, req *dto.UpdateImageVersionReq) error
	DeleteVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64) error
	SetDefaultVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64) error

	// 配置模板与文档
	GetConfigTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageConfigTemplateResp, error)
	GetDocumentation(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageDocumentationResp, error)
	GetImagePullStatus(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImagePullStatusListReq) (*dto.ImagePullStatusListResp, int64, error)
	TriggerImagePull(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TriggerImagePullReq) (*dto.TriggerImagePullResp, error)
}

// imageService 镜像管理服务实现
type imageService struct {
	db              *gorm.DB
	categoryRepo    experimentrepo.ImageCategoryRepository
	imageRepo       experimentrepo.ImageRepository
	versionRepo     experimentrepo.ImageVersionRepository
	userNameQuerier UserNameQuerier
	k8sSvc          K8sService
}

// NewImageService 创建镜像管理服务实例
func NewImageService(
	db *gorm.DB,
	categoryRepo experimentrepo.ImageCategoryRepository,
	imageRepo experimentrepo.ImageRepository,
	versionRepo experimentrepo.ImageVersionRepository,
	userNameQuerier UserNameQuerier,
	k8sSvc K8sService,
) ImageService {
	return &imageService{
		db: db, categoryRepo: categoryRepo,
		imageRepo: imageRepo, versionRepo: versionRepo,
		userNameQuerier: userNameQuerier,
		k8sSvc:          k8sSvc,
	}
}

// ---------------------------------------------------------------------------
// 镜像分类
// ---------------------------------------------------------------------------

// ListCategories 获取所有镜像分类
func (s *imageService) ListCategories(ctx context.Context, sc *svcctx.ServiceContext) ([]*dto.ImageCategoryResp, error) {
	if err := ensureImageCatalogAccess(sc); err != nil {
		return nil, err
	}
	categories, err := s.categoryRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.ImageCategoryResp, 0, len(categories))
	for _, c := range categories {
		result = append(result, s.toCategoryResp(c))
	}
	return result, nil
}

// CreateCategory 创建镜像分类
func (s *imageService) CreateCategory(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateImageCategoryReq) (*dto.ImageCategoryResp, error) {
	category := &entity.ImageCategory{
		ID:          snowflake.Generate(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}
	return s.toCategoryResp(category), nil
}

// UpdateCategory 更新镜像分类
func (s *imageService) UpdateCategory(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateImageCategoryReq) error {
	_, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageCategoryNotFound
	}
	fields := make(map[string]interface{})
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if len(fields) == 0 {
		return nil
	}
	return s.categoryRepo.UpdateFields(ctx, id, fields)
}

// DeleteCategory 删除镜像分类
func (s *imageService) DeleteCategory(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	_, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageCategoryNotFound
	}
	count, err := s.imageRepo.CountByCategoryID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errcode.ErrImageCategoryInUse
	}
	return s.categoryRepo.Delete(ctx, id)
}

// ---------------------------------------------------------------------------
// 镜像 CRUD
// ---------------------------------------------------------------------------

// Create 创建镜像
func (s *imageService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateImageReq) (string, error) {
	categoryID, _ := snowflake.ParseString(req.CategoryID)
	_, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return "", errcode.ErrImageCategoryNotFound
	}

	defaultPorts, err := marshalImageJSON(req.DefaultPorts, "默认端口配置格式错误")
	if err != nil {
		return "", err
	}
	defaultEnvVars, err := marshalImageJSON(req.DefaultEnvVars, "默认环境变量配置格式错误")
	if err != nil {
		return "", err
	}
	defaultVolumes, err := marshalImageJSON(req.DefaultVolumes, "默认挂载卷配置格式错误")
	if err != nil {
		return "", err
	}
	typicalCompanions, err := marshalImageJSON(req.TypicalCompanions, "典型搭配配置格式错误")
	if err != nil {
		return "", err
	}
	requiredDependencies, err := marshalImageJSON(req.RequiredDependencies, "依赖镜像配置格式错误")
	if err != nil {
		return "", err
	}
	resourceRecommendation, err := marshalImageJSON(req.ResourceRecommendation, "资源建议配置格式错误")
	if err != nil {
		return "", err
	}

	image := &entity.Image{
		ID:                     snowflake.Generate(),
		CategoryID:             categoryID,
		Name:                   req.Name,
		DisplayName:            req.DisplayName,
		Description:            req.Description,
		IconURL:                req.IconURL,
		Ecosystem:              req.Ecosystem,
		DefaultPorts:           defaultPorts,
		DefaultEnvVars:         defaultEnvVars,
		DefaultVolumes:         defaultVolumes,
		TypicalCompanions:      typicalCompanions,
		RequiredDependencies:   requiredDependencies,
		ResourceRecommendation: resourceRecommendation,
		DocumentationURL:       req.DocumentationURL,
		SourceType:             enum.ImageSourceTypeOfficial,
		Status:                 enum.ImageStatusNormal,
	}

	// 非超管创建的镜像视为教师自定义，需要审核
	if !sc.IsSuperAdmin() {
		image.SourceType = enum.ImageSourceTypeCustom
		image.UploadedBy = &sc.UserID
		if sc.SchoolID > 0 {
			image.SchoolID = &sc.SchoolID
		}
		image.Status = enum.ImageStatusPending
	}

	if err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txImageRepo := experimentrepo.NewImageRepository(tx)
		txVersionRepo := experimentrepo.NewImageVersionRepository(tx)
		if err := txImageRepo.Create(ctx, image); err != nil {
			return err
		}
		hasDefault := false
		for _, item := range req.Versions {
			if item.IsDefault {
				hasDefault = true
				break
			}
		}
		for index, item := range req.Versions {
			version := &entity.ImageVersion{
				ID:          snowflake.Generate(),
				ImageID:     image.ID,
				Version:     item.Version,
				RegistryURL: item.RegistryURL,
				MinCPU:      item.MinCPU,
				MinMemory:   item.MinMemory,
				MinDisk:     item.MinDisk,
				IsDefault:   item.IsDefault || (!hasDefault && index == 0),
				Status:      enum.ImageVersionStatusNormal,
			}
			if err := txVersionRepo.Create(ctx, version); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return "", err
	}
	return strconv.FormatInt(image.ID, 10), nil
}

// GetByID 获取镜像详情
func (s *imageService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageResp, error) {
	if err := ensureImageCatalogAccess(sc); err != nil {
		return nil, err
	}
	image, versions, err := s.getImageWithVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toImageResp(ctx, image, versions), nil
}

// Update 更新镜像
func (s *imageService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateImageReq) error {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageNotFound
	}
	if err := ensureImageWriteAccess(sc, image); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.IconURL != nil {
		fields["icon_url"] = *req.IconURL
	}
	if req.Ecosystem != nil {
		fields["ecosystem"] = *req.Ecosystem
	}
	if req.DefaultPorts != nil {
		defaultPorts, err := marshalImageJSON(req.DefaultPorts, "默认端口配置格式错误")
		if err != nil {
			return err
		}
		fields["default_ports"] = defaultPorts
	}
	if req.DefaultEnvVars != nil {
		defaultEnvVars, err := marshalImageJSON(req.DefaultEnvVars, "默认环境变量配置格式错误")
		if err != nil {
			return err
		}
		fields["default_env_vars"] = defaultEnvVars
	}
	if req.DefaultVolumes != nil {
		defaultVolumes, err := marshalImageJSON(req.DefaultVolumes, "默认挂载卷配置格式错误")
		if err != nil {
			return err
		}
		fields["default_volumes"] = defaultVolumes
	}
	if req.TypicalCompanions != nil {
		typicalCompanions, err := marshalImageJSON(req.TypicalCompanions, "典型搭配配置格式错误")
		if err != nil {
			return err
		}
		fields["typical_companions"] = typicalCompanions
	}
	if req.RequiredDependencies != nil {
		requiredDependencies, err := marshalImageJSON(req.RequiredDependencies, "依赖镜像配置格式错误")
		if err != nil {
			return err
		}
		fields["required_dependencies"] = requiredDependencies
	}
	if req.ResourceRecommendation != nil {
		resourceRecommendation, err := marshalImageJSON(req.ResourceRecommendation, "资源建议配置格式错误")
		if err != nil {
			return err
		}
		fields["resource_recommendation"] = resourceRecommendation
	}
	if req.DocumentationURL != nil {
		fields["documentation_url"] = *req.DocumentationURL
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.imageRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除镜像
func (s *imageService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageNotFound
	}
	referenceCount, err := s.imageRepo.CountTemplateReferences(ctx, image.ID)
	if err != nil {
		return err
	}
	if referenceCount > 0 {
		return errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("该镜像被%d个实验模板引用，不可下架", referenceCount))
	}
	if image.Status == enum.ImageStatusOffShelf {
		return nil
	}
	return s.imageRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":     enum.ImageStatusOffShelf,
		"updated_at": time.Now(),
	})
}

// List 镜像列表
func (s *imageService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImageListReq) ([]*dto.ImageListItem, int64, error) {
	if err := ensureImageCatalogAccess(sc); err != nil {
		return nil, 0, err
	}
	categoryID, _ := snowflake.ParseString(req.CategoryID)
	params := &experimentrepo.ImageListParams{
		Keyword:    req.Keyword,
		CategoryID: categoryID,
		Ecosystem:  req.Ecosystem,
		SourceType: req.SourceType,
		Status:     req.Status,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	images, total, err := s.imageRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.ImageListItem, 0, len(images))
	for _, img := range images {
		items = append(items, s.toImageListItem(ctx, img))
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// 审核
// ---------------------------------------------------------------------------

// Review 审核镜像
func (s *imageService) Review(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewImageReq) error {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageNotFound
	}
	if image.Status != enum.ImageStatusPending {
		return errcode.ErrImagePendingReview.WithMessage("镜像不在待审核状态")
	}

	now := time.Now()
	newStatus := enum.ImageStatusNormal
	if req.Action == "reject" {
		newStatus = enum.ImageStatusRejected
	}
	fields := map[string]interface{}{
		"status":         newStatus,
		"review_comment": req.Comment,
		"reviewed_by":    sc.UserID,
		"reviewed_at":    now,
		"updated_at":     now,
	}
	if err := s.imageRepo.UpdateFields(ctx, id, fields); err != nil {
		return err
	}
	if req.Action != "approve" {
		return nil
	}

	// 审核通过后只负责触发异步预拉取，不把节点侧拉取结果并入审核事务。
	// 这样可以保持“镜像状态变为正常”和“预拉取任务后台执行”两个业务节点语义清晰，
	// 也与模块04关于状态页查询进度、离线节点后续补拉的文档约束一致。
	cronpkg.RunAsync("模块04审核通过镜像预拉取", func(asyncCtx context.Context) {
		s.enqueueApprovedImagePrePull(detachContext(ctx), image.ID)
	})
	return nil
}

// enqueueApprovedImagePrePull 在自定义镜像审核通过后异步触发各版本预拉取。
// 失败只记录日志，由预拉取状态页与定时对账任务继续收口，不回滚已完成的审核状态流转。
func (s *imageService) enqueueApprovedImagePrePull(ctx context.Context, imageID int64) {
	if s.versionRepo == nil || s.k8sSvc == nil || imageID == 0 {
		return
	}

	versions, err := s.versionRepo.ListByImageID(ctx, imageID)
	if err != nil {
		logger.L.Warn("审核通过后触发镜像预拉取失败：查询版本列表失败",
			zap.Int64("image_id", imageID),
			zap.Error(err),
		)
		return
	}
	for _, version := range versions {
		if version == nil || version.RegistryURL == "" {
			continue
		}
		if err := s.k8sSvc.PrePullImage(ctx, version.RegistryURL, nil); err != nil {
			logger.L.Warn("审核通过后触发镜像预拉取失败",
				zap.Int64("image_id", imageID),
				zap.Int64("version_id", version.ID),
				zap.String("registry_url", version.RegistryURL),
				zap.Error(err),
			)
		}
	}
}

// ---------------------------------------------------------------------------
// 版本管理
// ---------------------------------------------------------------------------

// CreateVersion 创建镜像版本
func (s *imageService) CreateVersion(ctx context.Context, sc *svcctx.ServiceContext, imageID int64, req *dto.CreateImageVersionReq) (string, error) {
	image, err := s.imageRepo.GetByID(ctx, imageID)
	if err != nil {
		return "", errcode.ErrImageNotFound
	}
	if err := ensureImageWriteAccess(sc, image); err != nil {
		return "", err
	}

	version := &entity.ImageVersion{
		ID:          snowflake.Generate(),
		ImageID:     imageID,
		Version:     req.Version,
		RegistryURL: req.RegistryURL,
		ImageSize:   req.ImageSize,
		Digest:      req.Digest,
		MinCPU:      req.MinCPU,
		MinMemory:   req.MinMemory,
		MinDisk:     req.MinDisk,
		IsDefault:   req.IsDefault,
		Status:      enum.ImageVersionStatusNormal,
	}

	// 如果设为默认，先清除其他默认标记
	if req.IsDefault {
		if err := s.versionRepo.ClearDefault(ctx, imageID); err != nil {
			return "", err
		}
	}

	if err := s.versionRepo.Create(ctx, version); err != nil {
		return "", err
	}
	return strconv.FormatInt(version.ID, 10), nil
}

// UpdateVersion 编辑镜像版本。
func (s *imageService) UpdateVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64, req *dto.UpdateImageVersionReq) error {
	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return errcode.ErrImageVersionNotFound
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil {
		return errcode.ErrImageNotFound
	}
	if err := ensureImageWriteAccess(sc, image); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.RegistryURL != nil {
		fields["registry_url"] = *req.RegistryURL
	}
	if req.MinCPU != nil {
		fields["min_cpu"] = *req.MinCPU
	}
	if req.MinMemory != nil {
		fields["min_memory"] = *req.MinMemory
	}
	if req.MinDisk != nil {
		fields["min_disk"] = *req.MinDisk
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.versionRepo.UpdateFields(ctx, versionID, fields)
}

// DeleteVersion 删除镜像版本
func (s *imageService) DeleteVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64) error {
	if _, err := s.versionRepo.GetByID(ctx, versionID); err != nil {
		return errcode.ErrImageVersionNotFound
	}
	inUse, err := s.versionRepo.IsVersionInUse(ctx, versionID)
	if err != nil {
		return err
	}
	if inUse {
		return errcode.ErrImageVersionInUse
	}
	return s.versionRepo.Delete(ctx, versionID)
}

// SetDefaultVersion 设置默认版本
func (s *imageService) SetDefaultVersion(ctx context.Context, sc *svcctx.ServiceContext, versionID int64) error {
	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return errcode.ErrImageVersionNotFound
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil {
		return errcode.ErrImageNotFound
	}
	if err := ensureImageWriteAccess(sc, image); err != nil {
		return err
	}
	if err := s.versionRepo.ClearDefault(ctx, version.ImageID); err != nil {
		return err
	}
	return s.versionRepo.SetDefault(ctx, versionID)
}

// ---------------------------------------------------------------------------
// 配置模板与文档
// ---------------------------------------------------------------------------

// GetConfigTemplate 获取镜像配置模板
func (s *imageService) GetConfigTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageConfigTemplateResp, error) {
	if err := ensureTeacherScopeAccess(sc); err != nil {
		return nil, err
	}
	image, _, err := s.getImageWithVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	defaultPorts, err := unmarshalImageJSON[[]dto.ImagePortItem](image.DefaultPorts, "默认端口配置数据损坏")
	if err != nil {
		return nil, err
	}
	defaultEnvVars, err := unmarshalImageJSON[[]dto.ImageEnvVarItem](image.DefaultEnvVars, "默认环境变量配置数据损坏")
	if err != nil {
		return nil, err
	}
	defaultVolumes, err := unmarshalImageJSON[[]dto.ImageVolumeItem](image.DefaultVolumes, "默认挂载卷配置数据损坏")
	if err != nil {
		return nil, err
	}
	typicalCompanions, err := unmarshalImageJSON[dto.ImageTypicalCompanions](image.TypicalCompanions, "典型搭配配置数据损坏")
	if err != nil {
		return nil, err
	}
	requiredDependencies, err := unmarshalImageJSON[[]dto.ImageDependencyItem](image.RequiredDependencies, "依赖镜像配置数据损坏")
	if err != nil {
		return nil, err
	}
	resourceRecommendation, err := unmarshalImageJSON[dto.ImageResourceRecommendation](image.ResourceRecommendation, "资源建议配置数据损坏")
	if err != nil {
		return nil, err
	}
	resp := &dto.ImageConfigTemplateResp{
		ImageID:                strconv.FormatInt(image.ID, 10),
		Name:                   image.Name,
		DisplayName:            image.DisplayName,
		DefaultPorts:           defaultPorts,
		DefaultEnvVars:         defaultEnvVars,
		DefaultVolumes:         defaultVolumes,
		TypicalCompanions:      typicalCompanions,
		RequiredDependencies:   requiredDependencies,
		ResourceRecommendation: resourceRecommendation,
		ConditionalEnvVars:     buildConditionalEnvVarExamples(json.RawMessage(image.DefaultEnvVars)),
	}
	if image.Ecosystem != nil {
		resp.Ecosystem = image.Ecosystem
	}
	return resp, nil
}

// GetDocumentation 获取镜像文档
func (s *imageService) GetDocumentation(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageDocumentationResp, error) {
	if err := ensureTeacherScopeAccess(sc); err != nil {
		return nil, err
	}
	image, versions, err := s.getImageWithVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &dto.ImageDocumentationResp{
		ImageID:     strconv.FormatInt(image.ID, 10),
		Name:        image.Name,
		DisplayName: image.DisplayName,
	}
	sections := dto.ImageDocumentationSections{
		Overview:          strings.TrimSpace(derefString(image.Description)),
		VersionNotes:      s.buildImageVersionNotes(versions),
		DefaultConfig:     s.buildImageDefaultConfig(image),
		TypicalCompanions: s.buildImageCompanionDoc(image.TypicalCompanions),
		EnvVarsReference:  s.buildImageEnvVarReference(image.DefaultEnvVars),
		UsageExamples:     s.buildImageUsageExamples(image),
		Notes:             s.buildImageNotes(image),
	}
	resp.Sections = sections
	return resp, nil
}

// buildImageVersionNotes 构建镜像版本章节说明。
func (s *imageService) buildImageVersionNotes(versions []*entity.ImageVersion) string {
	if len(versions) == 0 {
		return "暂无版本说明。"
	}

	sorted := make([]*entity.ImageVersion, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].IsDefault == sorted[j].IsDefault {
			return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		}
		return sorted[i].IsDefault
	})

	lines := make([]string, 0, len(sorted))
	for _, version := range sorted {
		label := fmt.Sprintf("- %s", version.Version)
		if version.IsDefault {
			label += "（默认推荐）"
		}
		if version.Digest != nil && *version.Digest != "" {
			label += fmt.Sprintf("，Digest: `%s`", *version.Digest)
		}
		lines = append(lines, label)
	}
	return strings.Join(lines, "\n")
}

// buildImageDefaultConfig 构建镜像默认配置章节。
func (s *imageService) buildImageDefaultConfig(image *entity.Image) string {
	parts := make([]string, 0, 4)
	if config := formatJSONDoc("默认端口", image.DefaultPorts); config != "" {
		parts = append(parts, config)
	}
	if config := formatJSONDoc("默认环境变量", image.DefaultEnvVars); config != "" {
		parts = append(parts, config)
	}
	if config := formatJSONDoc("默认卷挂载", image.DefaultVolumes); config != "" {
		parts = append(parts, config)
	}
	if config := formatJSONDoc("资源建议", image.ResourceRecommendation); config != "" {
		parts = append(parts, config)
	}
	if len(parts) == 0 {
		return "暂无默认配置说明。"
	}
	return strings.Join(parts, "\n\n")
}

// buildImageCompanionDoc 构建典型搭配章节。
func (s *imageService) buildImageCompanionDoc(payload datatypes.JSON) string {
	text := formatJSONDoc("", payload)
	if text == "" {
		return "暂无典型搭配说明。"
	}
	return text
}

// buildImageEnvVarReference 构建环境变量参考章节。
func (s *imageService) buildImageEnvVarReference(payload datatypes.JSON) string {
	text := formatJSONDoc("", payload)
	if text == "" {
		return "暂无环境变量参考。"
	}
	return text
}

// buildImageUsageExamples 构建使用示例章节。
func (s *imageService) buildImageUsageExamples(image *entity.Image) string {
	parts := make([]string, 0, 3)
	if image.Ecosystem != nil && *image.Ecosystem != "" {
		parts = append(parts, fmt.Sprintf("### 适用生态\n推荐用于 `%s` 教学实验与容器编排。", *image.Ecosystem))
	}
	if usage := s.buildImageCompanionDoc(image.TypicalCompanions); usage != "暂无典型搭配说明。" {
		parts = append(parts, "### 典型搭配\n"+usage)
	}
	if deps := formatJSONDoc("依赖镜像", image.RequiredDependencies); deps != "" {
		parts = append(parts, deps)
	}
	if len(parts) == 0 {
		return "暂无使用示例。"
	}
	return strings.Join(parts, "\n\n")
}

// buildImageNotes 构建注意事项章节。
func (s *imageService) buildImageNotes(image *entity.Image) string {
	notes := make([]string, 0, 3)
	if image.DocumentationURL != nil && *image.DocumentationURL != "" {
		notes = append(notes, "详细文档请参考: "+*image.DocumentationURL)
	}
	if image.SourceType == enum.ImageSourceTypeCustom {
		notes = append(notes, "该镜像为教师自定义镜像，使用前请关注审核状态与版本适配要求。")
	}
	if len(notes) == 0 {
		return "暂无额外注意事项。"
	}
	return strings.Join(notes, "\n")
}

// getImageWithVersions 获取镜像主表与版本列表，并在 service 层完成聚合。
func (s *imageService) getImageWithVersions(ctx context.Context, id int64) (*entity.Image, []*entity.ImageVersion, error) {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, errcode.ErrImageNotFound
	}
	versions, err := s.versionRepo.ListByImageID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return image, versions, nil
}

// ensureImageCatalogAccess 校验镜像库查询权限。
func ensureImageCatalogAccess(sc *svcctx.ServiceContext) error {
	if sc == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() || sc.IsSchoolAdmin() || sc.IsTeacher() {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureTeacherScopeAccess 校验教师及以上访问权限。
func ensureTeacherScopeAccess(sc *svcctx.ServiceContext) error {
	if sc == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() || sc.IsTeacher() {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureImageWriteAccess 校验镜像上传者或超级管理员的写权限。
func ensureImageWriteAccess(sc *svcctx.ServiceContext, image *entity.Image) error {
	if sc == nil || image == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if image.UploadedBy != nil && *image.UploadedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// marshalImageJSON 将镜像 DTO 字段编码为 JSONB，统一由 service 层承担模型转换职责。
func marshalImageJSON(value interface{}, message string) (datatypes.JSON, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage(message)
	}
	if string(payload) == "null" {
		return datatypes.JSON(nil), nil
	}
	return datatypes.JSON(payload), nil
}

// unmarshalImageJSON 将镜像实体中的 JSONB 字段解码为 DTO 字段，避免 handler 直接处理持久化格式。
func unmarshalImageJSON[T any](payload datatypes.JSON, message string) (T, error) {
	var result T
	if len(payload) == 0 || string(payload) == "null" {
		return result, nil
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return result, errcode.ErrInternal.WithMessage(message)
	}
	return result, nil
}

// formatJSONDoc 将 JSON 元数据格式化为 Markdown 代码块说明。
func formatJSONDoc(title string, payload datatypes.JSON) string {
	if len(payload) == 0 || string(payload) == "null" {
		return ""
	}
	formatted := &strings.Builder{}
	if title != "" {
		formatted.WriteString("### ")
		formatted.WriteString(title)
		formatted.WriteString("\n")
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		return ""
	}
	formatted.WriteString("```json\n")
	formatted.Write(pretty.Bytes())
	formatted.WriteString("\n```")
	return formatted.String()
}

// derefString 读取可空字符串值。
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ---------------------------------------------------------------------------
// 转换方法
// ---------------------------------------------------------------------------

// toCategoryResp 转换镜像分类为响应
func (s *imageService) toCategoryResp(c *entity.ImageCategory) *dto.ImageCategoryResp {
	resp := &dto.ImageCategoryResp{
		ID:        strconv.FormatInt(c.ID, 10),
		Name:      c.Name,
		Code:      c.Code,
		SortOrder: c.SortOrder,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
	if c.Description != nil {
		resp.Description = *c.Description
	}
	return resp
}

// toImageListItem 转换镜像为列表项
func (s *imageService) toImageListItem(ctx context.Context, img *entity.Image) *dto.ImageListItem {
	item := &dto.ImageListItem{
		ID:             strconv.FormatInt(img.ID, 10),
		Name:           img.Name,
		DisplayName:    img.DisplayName,
		IconURL:        img.IconURL,
		Ecosystem:      img.Ecosystem,
		SourceType:     img.SourceType,
		SourceTypeText: enum.GetImageSourceTypeText(img.SourceType),
		Status:         img.Status,
		StatusText:     enum.GetImageStatusText(img.Status),
		UsageCount:     img.UsageCount,
		CreatedAt:      img.CreatedAt.Format(time.RFC3339),
	}
	if category, err := s.categoryRepo.GetByID(ctx, img.CategoryID); err == nil && category != nil {
		item.CategoryName = category.Name
	}
	if versions, err := s.versionRepo.ListByImageID(ctx, img.ID); err == nil {
		item.VersionCount = len(versions)
	}
	return item
}

// toImageResp 转换镜像为详情响应
func (s *imageService) toImageResp(ctx context.Context, img *entity.Image, versions []*entity.ImageVersion) *dto.ImageResp {
	defaultPorts, _ := unmarshalImageJSON[[]dto.ImagePortItem](img.DefaultPorts, "默认端口配置数据损坏")
	defaultEnvVars, _ := unmarshalImageJSON[[]dto.ImageEnvVarItem](img.DefaultEnvVars, "默认环境变量配置数据损坏")
	defaultVolumes, _ := unmarshalImageJSON[[]dto.ImageVolumeItem](img.DefaultVolumes, "默认挂载卷配置数据损坏")
	typicalCompanions, _ := unmarshalImageJSON[dto.ImageTypicalCompanions](img.TypicalCompanions, "典型搭配配置数据损坏")
	requiredDependencies, _ := unmarshalImageJSON[[]dto.ImageDependencyItem](img.RequiredDependencies, "依赖镜像配置数据损坏")
	resourceRecommendation, _ := unmarshalImageJSON[dto.ImageResourceRecommendation](img.ResourceRecommendation, "资源建议配置数据损坏")

	resp := &dto.ImageResp{
		ID:                     strconv.FormatInt(img.ID, 10),
		CategoryID:             strconv.FormatInt(img.CategoryID, 10),
		Name:                   img.Name,
		DisplayName:            img.DisplayName,
		Description:            img.Description,
		IconURL:                img.IconURL,
		Ecosystem:              img.Ecosystem,
		SourceType:             img.SourceType,
		SourceTypeText:         enum.GetImageSourceTypeText(img.SourceType),
		Status:                 img.Status,
		StatusText:             enum.GetImageStatusText(img.Status),
		ReviewComment:          img.ReviewComment,
		DefaultPorts:           defaultPorts,
		DefaultEnvVars:         defaultEnvVars,
		DefaultVolumes:         defaultVolumes,
		TypicalCompanions:      typicalCompanions,
		RequiredDependencies:   requiredDependencies,
		ResourceRecommendation: resourceRecommendation,
		DocumentationURL:       img.DocumentationURL,
		UsageCount:             img.UsageCount,
		CreatedAt:              img.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              img.UpdatedAt.Format(time.RFC3339),
	}
	if category, err := s.categoryRepo.GetByID(ctx, img.CategoryID); err == nil && category != nil {
		resp.CategoryName = category.Name
	}
	if img.UploadedBy != nil {
		uploadedBy := strconv.FormatInt(*img.UploadedBy, 10)
		uploaderName := s.userNameQuerier.GetUserName(ctx, *img.UploadedBy)
		resp.UploadedBy = &uploadedBy
		resp.UploaderName = &uploaderName
	}

	// 版本列表
	if len(versions) > 0 {
		resp.Versions = make([]dto.ImageVersionResp, 0, len(versions))
		for _, version := range versions {
			resp.Versions = append(resp.Versions, s.toVersionResp(version))
		}
	}
	return resp
}

// toVersionResp 转换镜像版本为响应
func (s *imageService) toVersionResp(v *entity.ImageVersion) dto.ImageVersionResp {
	resp := dto.ImageVersionResp{
		ID:          strconv.FormatInt(v.ID, 10),
		ImageID:     strconv.FormatInt(v.ImageID, 10),
		Version:     v.Version,
		RegistryURL: v.RegistryURL,
		ImageSize:   v.ImageSize,
		Digest:      v.Digest,
		MinCPU:      v.MinCPU,
		MinMemory:   v.MinMemory,
		MinDisk:     v.MinDisk,
		IsDefault:   v.IsDefault,
		Status:      v.Status,
		StatusText:  enum.GetImageVersionStatusText(v.Status),
		ScanResult:  json.RawMessage(v.ScanResult),
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
	}
	if v.ScannedAt != nil {
		t := v.ScannedAt.Format(time.RFC3339)
		resp.ScannedAt = &t
	}
	return resp
}
