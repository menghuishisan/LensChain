// image_manifest_sync.go
// 模块04 — 实验环境：镜像清单同步
//
// 设计目标：让 deploy/images/<category>/<name>/manifest.yaml 成为镜像元数据的
// 单一真相源（Documentation-Driven Development）。本文件提供解析 manifest yaml、
// 按 (name) / (image_id, version) 业务键幂等写入 images / image_versions 的能力，
// 同时被以下两条入口共享：
//   1. cmd/seed-manifests CLI —— 部署期 bootstrap，无需启动 backend 即可灌库；
//   2. POST /api/v1/admin/images/sync —— 运行期管理员上传 manifest 增量更新。
//
// 业务规则：
//   - images.name 是 upsert 业务键（已由 009_images_unique_constraints 保证唯一）；
//   - image_versions(image_id, version) 是 upsert 业务键；
//   - manifest.category 通过 image_categories.code 反查 category_id，避免硬编码 ID；
//   - registry_url 由 registry.lianjing.com/<registry_project>/<name>:<tag> 拼装，
//     与 deploy/scripts 构建/推送脚本约定保持一致。

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// manifestRegistryHost 平台镜像仓库主机名，与 deploy/scripts/bash/*.sh 中
// REGISTRY 默认值保持一致；如需多环境覆盖应通过 config.env 注入并在调用处传入。
const manifestRegistryHost = "registry.lianjing.com"

// imageManifest manifest.yaml 顶层结构（仅声明同步所需字段，未列出的字段
// 由构建/部署链路使用，本同步流程透明跳过）。
type imageManifest struct {
	Name            string                  `yaml:"name"`
	DisplayName     string                  `yaml:"display_name"`
	Category        string                  `yaml:"category"`
	Ecosystem       string                  `yaml:"ecosystem"`
	Description     string                  `yaml:"description"`
	RegistryProject string                  `yaml:"registry_project"`
	ToolKind        string                  `yaml:"tool_kind"`
	DocumentationURL string                 `yaml:"documentation_url"`

	Versions []imageManifestVersion `yaml:"versions"`

	DefaultPorts         []map[string]any `yaml:"default_ports"`
	DefaultEnvVars       []map[string]any `yaml:"default_env_vars"`
	ConditionalEnvVars   []map[string]any `yaml:"conditional_env_vars"`
	DefaultVolumes       []map[string]any `yaml:"default_volumes"`
	TypicalCompanions    map[string]any   `yaml:"typical_companions"`
	RequiredDependencies []string         `yaml:"required_dependencies"`
}

// imageManifestVersion manifest.versions 数组成员。
type imageManifestVersion struct {
	Version          string `yaml:"version"`
	Tag              string `yaml:"tag"`
	MinCPU           string `yaml:"min_cpu"`
	MinMemory        string `yaml:"min_memory"`
	MinDisk          string `yaml:"min_disk"`
	RecommendedCPU   string `yaml:"recommended_cpu"`
	RecommendedMemory string `yaml:"recommended_memory"`
	IsDefault        bool   `yaml:"is_default"`
}

// SyncImageFromManifest 解析 manifest yaml 字节并 upsert 到 images / image_versions 表。
// 调用方：cmd/seed-manifests CLI、admin handler。
//
// 幂等性：
//   - 镜像存在（按 name）→ 更新元数据字段；
//   - 镜像不存在 → 雪花 ID 新建；
//   - 版本存在（按 image_id, version）→ 更新 registry_url / 资源字段 / is_default；
//   - 版本不存在 → 雪花 ID 新建；
//
// 注意：本方法不暴露给前端教师角色，调用前由 handler / CLI 完成权限校验。
func (s *imageService) SyncImageFromManifest(ctx context.Context, raw []byte) (*dto.ImageManifestSyncResp, error) {
	var m imageManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("manifest 解析失败: %v", err))
	}
	if err := validateManifest(&m); err != nil {
		return nil, err
	}

	category, err := s.categoryRepo.GetByCode(ctx, m.Category)
	if err != nil {
		return nil, errcode.ErrImageCategoryNotFound.WithMessage(fmt.Sprintf("manifest.category=%q 在 image_categories 中不存在", m.Category))
	}

	result := &dto.ImageManifestSyncResp{Name: m.Name}

	if err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txImageRepo := experimentrepo.NewImageRepository(tx)
		txVersionRepo := experimentrepo.NewImageVersionRepository(tx)

		image, err := upsertImageFromManifest(ctx, txImageRepo, &m, category.ID, result)
		if err != nil {
			return err
		}

		hasDefault := false
		for _, v := range m.Versions {
			if v.IsDefault {
				hasDefault = true
				break
			}
		}
		for idx, v := range m.Versions {
			isDefault := v.IsDefault || (!hasDefault && idx == 0)
			if err := upsertVersionFromManifest(ctx, txVersionRepo, image, &m, &v, isDefault, result); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// validateManifest 检查 manifest 必填字段。
func validateManifest(m *imageManifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return errcode.ErrInvalidParams.WithMessage("manifest.name 不能为空")
	}
	if strings.TrimSpace(m.Category) == "" {
		return errcode.ErrInvalidParams.WithMessage("manifest.category 不能为空")
	}
	if len(m.Versions) == 0 {
		return errcode.ErrInvalidParams.WithMessage("manifest.versions 不能为空，至少声明一个版本")
	}
	for i, v := range m.Versions {
		if strings.TrimSpace(v.Version) == "" {
			return errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("manifest.versions[%d].version 不能为空", i))
		}
		if strings.TrimSpace(v.Tag) == "" {
			return errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("manifest.versions[%d].tag 不能为空", i))
		}
	}
	return nil
}

// upsertImageFromManifest 按 name upsert images 行。
func upsertImageFromManifest(
	ctx context.Context,
	repo experimentrepo.ImageRepository,
	m *imageManifest,
	categoryID int64,
	result *dto.ImageManifestSyncResp,
) (*entity.Image, error) {
	defaultPortsJSON, err := jsonOrNil(m.DefaultPorts)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("manifest.default_ports 序列化失败")
	}
	envVars := append([]map[string]any{}, m.DefaultEnvVars...)
	envVars = append(envVars, m.ConditionalEnvVars...)
	defaultEnvVarsJSON, err := jsonOrNil(envVars)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("manifest.default_env_vars 序列化失败")
	}
	defaultVolumesJSON, err := jsonOrNil(m.DefaultVolumes)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("manifest.default_volumes 序列化失败")
	}
	companionsJSON, err := jsonOrNil(m.TypicalCompanions)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("manifest.typical_companions 序列化失败")
	}
	requiredDepsJSON, err := jsonOrNil(m.RequiredDependencies)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("manifest.required_dependencies 序列化失败")
	}

	displayName := strings.TrimSpace(m.DisplayName)
	if displayName == "" {
		displayName = m.Name
	}
	var (
		ecosystemPtr   *string
		descriptionPtr *string
		toolKindPtr    *string
		docURLPtr      *string
	)
	if v := strings.TrimSpace(m.Ecosystem); v != "" {
		ecosystemPtr = &v
	}
	if v := strings.TrimSpace(m.Description); v != "" {
		descriptionPtr = &v
	}
	if v := strings.TrimSpace(m.ToolKind); v != "" {
		toolKindPtr = &v
	}
	if v := strings.TrimSpace(m.DocumentationURL); v != "" {
		docURLPtr = &v
	}

	existing, err := repo.GetByName(ctx, m.Name)
	if err == nil && existing != nil {
		fields := map[string]any{
			"category_id":           categoryID,
			"display_name":          displayName,
			"description":           descriptionPtr,
			"ecosystem":             ecosystemPtr,
			"tool_kind":             toolKindPtr,
			"documentation_url":     docURLPtr,
			"default_ports":         defaultPortsJSON,
			"default_env_vars":      defaultEnvVarsJSON,
			"default_volumes":       defaultVolumesJSON,
			"typical_companions":    companionsJSON,
			"required_dependencies": requiredDepsJSON,
		}
		if err := repo.UpdateFields(ctx, existing.ID, fields); err != nil {
			return nil, err
		}
		result.ImageUpdated = true
		// 重新加载以拿到最新字段（部分调用方依赖返回值）
		return repo.GetByID(ctx, existing.ID)
	}

	image := &entity.Image{
		ID:                   snowflake.Generate(),
		CategoryID:           categoryID,
		Name:                 m.Name,
		DisplayName:          displayName,
		Description:          descriptionPtr,
		Ecosystem:            ecosystemPtr,
		ToolKind:             toolKindPtr,
		DocumentationURL:     docURLPtr,
		DefaultPorts:         defaultPortsJSON,
		DefaultEnvVars:       defaultEnvVarsJSON,
		DefaultVolumes:       defaultVolumesJSON,
		TypicalCompanions:    companionsJSON,
		RequiredDependencies: requiredDepsJSON,
		SourceType:           enum.ImageSourceTypeOfficial,
		Status:               enum.ImageStatusNormal,
	}
	if err := repo.Create(ctx, image); err != nil {
		return nil, err
	}
	result.ImageCreated = true
	return image, nil
}

// upsertVersionFromManifest 按 (image_id, version) upsert image_versions 行。
func upsertVersionFromManifest(
	ctx context.Context,
	repo experimentrepo.ImageVersionRepository,
	image *entity.Image,
	m *imageManifest,
	v *imageManifestVersion,
	isDefault bool,
	result *dto.ImageManifestSyncResp,
) error {
	registryURL := buildManifestRegistryURL(m, v)

	var (
		minCPUPtr    *string
		minMemPtr    *string
		minDiskPtr   *string
	)
	if val := strings.TrimSpace(v.MinCPU); val != "" {
		minCPUPtr = &val
	}
	if val := strings.TrimSpace(v.MinMemory); val != "" {
		minMemPtr = &val
	}
	if val := strings.TrimSpace(v.MinDisk); val != "" {
		minDiskPtr = &val
	}

	existing, err := repo.GetByImageAndVersion(ctx, image.ID, v.Version)
	if err == nil && existing != nil {
		// 已存在版本：仅在内容真有差异时更新，避免无谓的 UpdatedAt 抖动
		if existing.RegistryURL == registryURL &&
			ptrEqual(existing.MinCPU, minCPUPtr) &&
			ptrEqual(existing.MinMemory, minMemPtr) &&
			ptrEqual(existing.MinDisk, minDiskPtr) &&
			existing.IsDefault == isDefault {
			result.VersionsSkipped++
			return nil
		}
		fields := map[string]any{
			"registry_url": registryURL,
			"min_cpu":      minCPUPtr,
			"min_memory":   minMemPtr,
			"min_disk":     minDiskPtr,
			"is_default":   isDefault,
		}
		if err := repo.UpdateFields(ctx, existing.ID, fields); err != nil {
			return err
		}
		result.VersionsUpdated++
		return nil
	}

	version := &entity.ImageVersion{
		ID:          snowflake.Generate(),
		ImageID:     image.ID,
		Version:     v.Version,
		RegistryURL: registryURL,
		MinCPU:      minCPUPtr,
		MinMemory:   minMemPtr,
		MinDisk:     minDiskPtr,
		IsDefault:   isDefault,
		Status:      enum.ImageVersionStatusNormal,
	}
	if err := repo.Create(ctx, version); err != nil {
		return err
	}
	result.VersionsCreated++
	return nil
}

// buildManifestRegistryURL 拼装镜像完整 registry URL，格式与 deploy/scripts 中
// REGISTRY 推送路径保持一致：<host>/<registry_project>/<name>:<tag>
func buildManifestRegistryURL(m *imageManifest, v *imageManifestVersion) string {
	project := strings.TrimSpace(m.RegistryProject)
	if project == "" {
		project = strings.TrimSpace(m.Category)
	}
	return fmt.Sprintf("%s/%s/%s:%s", manifestRegistryHost, project, m.Name, v.Tag)
}

// jsonOrNil 把任意 map / slice 序列化为 datatypes.JSON。空值返回 nil 让 GORM 写入 NULL。
func jsonOrNil(v any) (datatypes.JSON, error) {
	if v == nil {
		return nil, nil
	}
	switch typed := v.(type) {
	case []map[string]any:
		if len(typed) == 0 {
			return datatypes.JSON([]byte("[]")), nil
		}
	case []string:
		if len(typed) == 0 {
			return datatypes.JSON([]byte("[]")), nil
		}
	case map[string]any:
		if len(typed) == 0 {
			return datatypes.JSON([]byte("{}")), nil
		}
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(bytes), nil
}

// ptrEqual 比较两个 *string 是否表示相同内容（含同为 nil）。
func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
