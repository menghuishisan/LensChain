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

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
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
	GetImagePullStatus(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImagePullStatusReq) (*dto.ImagePullStatusResp, int64, error)
	TriggerImagePull(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImagePullReq) (*dto.ImagePullResp, error)
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

	image := &entity.Image{
		ID:                     snowflake.Generate(),
		CategoryID:             categoryID,
		Name:                   req.Name,
		DisplayName:            req.DisplayName,
		Description:            req.Description,
		IconURL:                req.IconURL,
		Ecosystem:              req.Ecosystem,
		DefaultPorts:           req.DefaultPorts,
		DefaultEnvVars:         req.DefaultEnvVars,
		DefaultVolumes:         req.DefaultVolumes,
		TypicalCompanions:      req.TypicalCompanions,
		RequiredDependencies:   req.RequiredDependencies,
		ResourceRecommendation: req.ResourceRecommendation,
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

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if image.ID == 0 {
			image.ID = snowflake.Generate()
		}
		if err := tx.Create(image).Error; err != nil {
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
			if err := tx.Create(version).Error; err != nil {
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
	image, err := s.imageRepo.GetByIDWithVersions(ctx, id)
	if err != nil {
		return nil, errcode.ErrImageNotFound
	}
	return s.toImageResp(ctx, image), nil
}

// Update 更新镜像
func (s *imageService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateImageReq) error {
	_, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrImageNotFound
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
		fields["default_ports"] = req.DefaultPorts
	}
	if req.DefaultEnvVars != nil {
		fields["default_env_vars"] = req.DefaultEnvVars
	}
	if req.DefaultVolumes != nil {
		fields["default_volumes"] = req.DefaultVolumes
	}
	if req.TypicalCompanions != nil {
		fields["typical_companions"] = req.TypicalCompanions
	}
	if req.RequiredDependencies != nil {
		fields["required_dependencies"] = req.RequiredDependencies
	}
	if req.ResourceRecommendation != nil {
		fields["resource_recommendation"] = req.ResourceRecommendation
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
	// 检查是否有版本被引用
	versions, err := s.versionRepo.ListByImageID(ctx, image.ID)
	if err != nil {
		return err
	}
	for _, v := range versions {
		inUse, err := s.versionRepo.IsVersionInUse(ctx, v.ID)
		if err != nil {
			return err
		}
		if inUse {
			return errcode.ErrImageHasReferences
		}
	}
	return s.imageRepo.SoftDelete(ctx, id)
}

// List 镜像列表
func (s *imageService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImageListReq) ([]*dto.ImageListItem, int64, error) {
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
	versions, err := s.versionRepo.ListByImageID(ctx, image.ID)
	if err != nil {
		return err
	}
	for _, version := range versions {
		if version.RegistryURL == "" {
			continue
		}
		if err := s.k8sSvc.PrePullImage(ctx, version.RegistryURL, nil); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 版本管理
// ---------------------------------------------------------------------------

// CreateVersion 创建镜像版本
func (s *imageService) CreateVersion(ctx context.Context, sc *svcctx.ServiceContext, imageID int64, req *dto.CreateImageVersionReq) (string, error) {
	_, err := s.imageRepo.GetByID(ctx, imageID)
	if err != nil {
		return "", errcode.ErrImageNotFound
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
	_, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return errcode.ErrImageVersionNotFound
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
	image, err := s.imageRepo.GetByIDWithVersions(ctx, id)
	if err != nil {
		return nil, errcode.ErrImageNotFound
	}
	resp := &dto.ImageConfigTemplateResp{
		ImageID:                strconv.FormatInt(image.ID, 10),
		Name:                   image.Name,
		DisplayName:            image.DisplayName,
		DefaultPorts:           image.DefaultPorts,
		DefaultEnvVars:         image.DefaultEnvVars,
		DefaultVolumes:         image.DefaultVolumes,
		TypicalCompanions:      image.TypicalCompanions,
		RequiredDependencies:   image.RequiredDependencies,
		ResourceRecommendation: image.ResourceRecommendation,
	}
	if image.Ecosystem != nil {
		resp.Ecosystem = image.Ecosystem
	}
	return resp, nil
}

// GetDocumentation 获取镜像文档
func (s *imageService) GetDocumentation(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ImageDocumentationResp, error) {
	image, err := s.imageRepo.GetByIDWithVersions(ctx, id)
	if err != nil {
		return nil, errcode.ErrImageNotFound
	}
	resp := &dto.ImageDocumentationResp{
		ImageID:     strconv.FormatInt(image.ID, 10),
		Name:        image.Name,
		DisplayName: image.DisplayName,
	}
	sections := dto.ImageDocumentationSections{
		Overview:          strings.TrimSpace(derefString(image.Description)),
		VersionNotes:      s.buildImageVersionNotes(image.Versions),
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
func (s *imageService) buildImageVersionNotes(versions []entity.ImageVersion) string {
	if len(versions) == 0 {
		return "暂无版本说明。"
	}

	sorted := make([]entity.ImageVersion, len(versions))
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
func (s *imageService) buildImageCompanionDoc(payload json.RawMessage) string {
	text := formatJSONDoc("", payload)
	if text == "" {
		return "暂无典型搭配说明。"
	}
	return text
}

// buildImageEnvVarReference 构建环境变量参考章节。
func (s *imageService) buildImageEnvVarReference(payload json.RawMessage) string {
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
		notes = append(notes, "该镜像为教师自定义镜像，使用前请关注审核状态与版本兼容性。")
	}
	if len(notes) == 0 {
		return "暂无额外注意事项。"
	}
	return strings.Join(notes, "\n")
}

// formatJSONDoc 将 JSON 元数据格式化为 Markdown 代码块说明。
func formatJSONDoc(title string, payload json.RawMessage) string {
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
func (s *imageService) toImageResp(ctx context.Context, img *entity.Image) *dto.ImageResp {
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
		DefaultPorts:           img.DefaultPorts,
		DefaultEnvVars:         img.DefaultEnvVars,
		DefaultVolumes:         img.DefaultVolumes,
		TypicalCompanions:      img.TypicalCompanions,
		RequiredDependencies:   img.RequiredDependencies,
		ResourceRecommendation: img.ResourceRecommendation,
		DocumentationURL:       img.DocumentationURL,
		UsageCount:             img.UsageCount,
		CreatedAt:              img.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              img.UpdatedAt.Format(time.RFC3339),
	}
	if img.UploadedBy != nil {
		uploadedBy := strconv.FormatInt(*img.UploadedBy, 10)
		uploaderName := s.userNameQuerier.GetUserName(ctx, *img.UploadedBy)
		resp.UploadedBy = &uploadedBy
		resp.UploaderName = &uploaderName
	}

	// 版本列表
	if len(img.Versions) > 0 {
		resp.Versions = make([]dto.ImageVersionResp, 0, len(img.Versions))
		for _, v := range img.Versions {
			resp.Versions = append(resp.Versions, s.toVersionResp(&v))
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
		ScanResult:  v.ScanResult,
		CreatedAt:   v.CreatedAt.Format(time.RFC3339),
	}
	if v.ScannedAt != nil {
		t := v.ScannedAt.Format(time.RFC3339)
		resp.ScannedAt = &t
	}
	return resp
}
