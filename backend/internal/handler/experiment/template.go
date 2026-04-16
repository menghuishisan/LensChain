// template.go
// 模块04 — 实验环境：镜像、模板、场景与标签 HTTP 处理层
// 负责镜像管理、模板管理、模板子资源、仿真场景库、联动组、标签与共享模板接口
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/experiment"
)

// TemplateHandler 实验模板域处理器。
// 统一处理镜像、模板、模板子资源、场景库、标签和共享模板相关接口。
type TemplateHandler struct {
	imageService       svc.ImageService
	templateService    svc.TemplateService
	templateSubService svc.TemplateSubService
	scenarioService    svc.ScenarioService
}

// NewTemplateHandler 创建实验模板域处理器。
func NewTemplateHandler(
	imageService svc.ImageService,
	templateService svc.TemplateService,
	templateSubService svc.TemplateSubService,
	scenarioService svc.ScenarioService,
) *TemplateHandler {
	return &TemplateHandler{
		imageService:       imageService,
		templateService:    templateService,
		templateSubService: templateSubService,
		scenarioService:    scenarioService,
	}
}

// ListImages 获取镜像列表。
// GET /api/v1/images
func (h *TemplateHandler) ListImages(c *gin.Context) {
	var req dto.ImageListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.imageService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// CreateImage 创建或上传镜像。
// POST /api/v1/images
func (h *TemplateHandler) CreateImage(c *gin.Context) {
	var req dto.CreateImageReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.imageService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	imageID, parseErr := snowflake.ParseString(id)
	parseOK := parseErr == nil && imageID > 0
	if !parseOK {
		response.Created(c, gin.H{"id": id})
		return
	}
	detail, err := h.imageService.GetByID(c.Request.Context(), sc, imageID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, detail)
}

// GetImage 获取镜像详情。
// GET /api/v1/images/:id
func (h *TemplateHandler) GetImage(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.imageService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateImage 编辑镜像信息。
// PUT /api/v1/images/:id
func (h *TemplateHandler) UpdateImage(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateImageReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteImage 删除镜像。
// DELETE /api/v1/images/:id
func (h *TemplateHandler) DeleteImage(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ReviewImage 审核镜像。
// POST /api/v1/images/:id/review
func (h *TemplateHandler) ReviewImage(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewImageReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.Review(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "审核成功", nil)
}

// ListImageVersions 获取镜像版本列表。
// GET /api/v1/images/:id/versions
func (h *TemplateHandler) ListImageVersions(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.imageService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail.Versions)
}

// CreateImageVersion 添加镜像版本。
// POST /api/v1/images/:id/versions
func (h *TemplateHandler) CreateImageVersion(c *gin.Context) {
	imageID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateImageVersionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.imageService.CreateVersion(c.Request.Context(), sc, imageID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// UpdateImageVersion 编辑镜像版本。
// PUT /api/v1/image-versions/:id
func (h *TemplateHandler) UpdateImageVersion(c *gin.Context) {
	versionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateImageVersionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.UpdateVersion(c.Request.Context(), sc, versionID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteImageVersion 删除镜像版本。
// DELETE /api/v1/image-versions/:id
func (h *TemplateHandler) DeleteImageVersion(c *gin.Context) {
	versionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.DeleteVersion(c.Request.Context(), sc, versionID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// SetDefaultImageVersion 设置默认镜像版本。
// PATCH /api/v1/image-versions/:id/default
func (h *TemplateHandler) SetDefaultImageVersion(c *gin.Context) {
	versionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.imageService.SetDefaultVersion(c.Request.Context(), sc, versionID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// GetImageConfigTemplate 获取镜像配置模板。
// GET /api/v1/images/:id/config-template
func (h *TemplateHandler) GetImageConfigTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.imageService.GetConfigTemplate(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetImageDocumentation 获取镜像结构化文档。
// GET /api/v1/images/:id/documentation
func (h *TemplateHandler) GetImageDocumentation(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.imageService.GetDocumentation(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// ListImageCategories 获取镜像分类列表。
// GET /api/v1/image-categories
func (h *TemplateHandler) ListImageCategories(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.imageService.ListCategories(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplate 创建实验模板。
// POST /api/v1/experiment-templates
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req dto.CreateTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// ListTemplates 获取实验模板列表。
// GET /api/v1/experiment-templates
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	var req dto.TemplateListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.templateService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetTemplate 获取实验模板详情。
// GET /api/v1/experiment-templates/:id
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.templateService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateTemplate 编辑实验模板。
// PUT /api/v1/experiment-templates/:id
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplate 删除实验模板。
// DELETE /api/v1/experiment-templates/:id
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// PublishTemplate 发布实验模板。
// POST /api/v1/experiment-templates/:id/publish
func (h *TemplateHandler) PublishTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateService.Publish(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", nil)
}

// CloneTemplate 克隆实验模板。
// POST /api/v1/experiment-templates/:id/clone
func (h *TemplateHandler) CloneTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateService.Clone(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// ShareTemplate 设置模板共享状态。
// PATCH /api/v1/experiment-templates/:id/share
func (h *TemplateHandler) ShareTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ShareTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateService.Share(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// GetTemplateK8sConfig 获取模板 K8s 编排配置。
// GET /api/v1/experiment-templates/:id/k8s-config
func (h *TemplateHandler) GetTemplateK8sConfig(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateService.GetK8sConfig(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SetTemplateK8sConfig 设置模板 K8s 编排配置。
// POST /api/v1/experiment-templates/:id/k8s-config
func (h *TemplateHandler) SetTemplateK8sConfig(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.K8sConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateService.SetK8sConfig(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "保存成功", nil)
}

// ValidateTemplate 校验实验模板配置。
// POST /api/v1/experiment-templates/:id/validate
func (h *TemplateHandler) ValidateTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ValidateTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateService.Validate(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListTemplateContainers 获取模板容器配置列表。
// GET /api/v1/experiment-templates/:id/containers
func (h *TemplateHandler) ListTemplateContainers(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListContainers(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplateContainer 添加模板容器配置。
// POST /api/v1/experiment-templates/:id/containers
func (h *TemplateHandler) CreateTemplateContainer(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateContainerReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateContainer(c.Request.Context(), sc, templateID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// SortTemplateContainers 调整模板容器排序。
// PUT /api/v1/experiment-templates/:id/containers/sort
func (h *TemplateHandler) SortTemplateContainers(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SortReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.SortContainers(c.Request.Context(), sc, templateID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序成功", nil)
}

// UpdateTemplateContainer 编辑模板容器配置。
// PUT /api/v1/template-containers/:id
func (h *TemplateHandler) UpdateTemplateContainer(c *gin.Context) {
	containerID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateContainerReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateContainer(c.Request.Context(), sc, containerID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplateContainer 删除模板容器配置。
// DELETE /api/v1/template-containers/:id
func (h *TemplateHandler) DeleteTemplateContainer(c *gin.Context) {
	containerID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.DeleteContainer(c.Request.Context(), sc, containerID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListTemplateCheckpoints 获取模板检查点列表。
// GET /api/v1/experiment-templates/:id/checkpoints
func (h *TemplateHandler) ListTemplateCheckpoints(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListCheckpoints(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplateCheckpoint 添加模板检查点。
// POST /api/v1/experiment-templates/:id/checkpoints
func (h *TemplateHandler) CreateTemplateCheckpoint(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateCheckpointReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateCheckpoint(c.Request.Context(), sc, templateID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// SortTemplateCheckpoints 调整模板检查点排序。
// PUT /api/v1/experiment-templates/:id/checkpoints/sort
func (h *TemplateHandler) SortTemplateCheckpoints(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SortReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.SortCheckpoints(c.Request.Context(), sc, templateID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序成功", nil)
}

// UpdateTemplateCheckpoint 编辑模板检查点。
// PUT /api/v1/template-checkpoints/:id
func (h *TemplateHandler) UpdateTemplateCheckpoint(c *gin.Context) {
	checkpointID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCheckpointReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateCheckpoint(c.Request.Context(), sc, checkpointID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplateCheckpoint 删除模板检查点。
// DELETE /api/v1/template-checkpoints/:id
func (h *TemplateHandler) DeleteTemplateCheckpoint(c *gin.Context) {
	checkpointID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.DeleteCheckpoint(c.Request.Context(), sc, checkpointID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListTemplateInitScripts 获取模板初始化脚本列表。
// GET /api/v1/experiment-templates/:id/init-scripts
func (h *TemplateHandler) ListTemplateInitScripts(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListInitScripts(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplateInitScript 添加模板初始化脚本。
// POST /api/v1/experiment-templates/:id/init-scripts
func (h *TemplateHandler) CreateTemplateInitScript(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateInitScriptReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateInitScript(c.Request.Context(), sc, templateID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// UpdateTemplateInitScript 编辑模板初始化脚本。
// PUT /api/v1/template-init-scripts/:id
func (h *TemplateHandler) UpdateTemplateInitScript(c *gin.Context) {
	scriptID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateInitScriptReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateInitScript(c.Request.Context(), sc, scriptID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplateInitScript 删除模板初始化脚本。
// DELETE /api/v1/template-init-scripts/:id
func (h *TemplateHandler) DeleteTemplateInitScript(c *gin.Context) {
	scriptID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.DeleteInitScript(c.Request.Context(), sc, scriptID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListTemplateSimScenes 获取模板仿真场景配置列表。
// GET /api/v1/experiment-templates/:id/sim-scenes
func (h *TemplateHandler) ListTemplateSimScenes(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListSimScenes(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplateSimScene 添加模板仿真场景配置。
// POST /api/v1/experiment-templates/:id/sim-scenes
func (h *TemplateHandler) CreateTemplateSimScene(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateTemplateSimSceneReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateSimScene(c.Request.Context(), sc, templateID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// UpdateTemplateSimSceneLayout 批量更新模板仿真场景布局。
// PUT /api/v1/experiment-templates/:id/sim-scenes/layout
func (h *TemplateHandler) UpdateTemplateSimSceneLayout(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateSimSceneLayoutReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateSimSceneLayout(c.Request.Context(), sc, templateID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// UpdateTemplateSimScene 编辑模板仿真场景配置。
// PUT /api/v1/template-sim-scenes/:id
func (h *TemplateHandler) UpdateTemplateSimScene(c *gin.Context) {
	simSceneID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTemplateSimSceneReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateSimScene(c.Request.Context(), sc, simSceneID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplateSimScene 删除模板仿真场景配置。
// DELETE /api/v1/template-sim-scenes/:id
func (h *TemplateHandler) DeleteTemplateSimScene(c *gin.Context) {
	simSceneID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.DeleteSimScene(c.Request.Context(), sc, simSceneID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// SetTemplateTags 设置模板标签。
// PUT /api/v1/experiment-templates/:id/tags
func (h *TemplateHandler) SetTemplateTags(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SetTemplateTagsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.SetTemplateTags(c.Request.Context(), sc, templateID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// ListTemplateTags 获取模板标签列表。
// GET /api/v1/experiment-templates/:id/tags
func (h *TemplateHandler) ListTemplateTags(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListTemplateTags(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ListTemplateRoles 获取模板角色列表。
// GET /api/v1/experiment-templates/:id/roles
func (h *TemplateHandler) ListTemplateRoles(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListRoles(c.Request.Context(), sc, templateID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTemplateRole 添加模板角色。
// POST /api/v1/experiment-templates/:id/roles
func (h *TemplateHandler) CreateTemplateRole(c *gin.Context) {
	templateID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateRoleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateRole(c.Request.Context(), sc, templateID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// UpdateTemplateRole 编辑模板角色。
// PUT /api/v1/template-roles/:id
func (h *TemplateHandler) UpdateTemplateRole(c *gin.Context) {
	roleID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateRoleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.UpdateRole(c.Request.Context(), sc, roleID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteTemplateRole 删除模板角色。
// DELETE /api/v1/template-roles/:id
func (h *TemplateHandler) DeleteTemplateRole(c *gin.Context) {
	roleID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.templateSubService.DeleteRole(c.Request.Context(), sc, roleID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListScenarios 获取仿真场景列表。
// GET /api/v1/sim-scenarios
func (h *TemplateHandler) ListScenarios(c *gin.Context) {
	var req dto.ScenarioListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.scenarioService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// CreateScenario 创建自定义仿真场景。
// POST /api/v1/sim-scenarios
func (h *TemplateHandler) CreateScenario(c *gin.Context) {
	var req dto.CreateScenarioReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.scenarioService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// GetScenario 获取仿真场景详情。
// GET /api/v1/sim-scenarios/:id
func (h *TemplateHandler) GetScenario(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.scenarioService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateScenario 编辑仿真场景。
// PUT /api/v1/sim-scenarios/:id
func (h *TemplateHandler) UpdateScenario(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateScenarioReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.scenarioService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteScenario 删除仿真场景。
// DELETE /api/v1/sim-scenarios/:id
func (h *TemplateHandler) DeleteScenario(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.scenarioService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ReviewScenario 审核仿真场景。
// POST /api/v1/sim-scenarios/:id/review
func (h *TemplateHandler) ReviewScenario(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewScenarioReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.scenarioService.Review(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "审核成功", nil)
}

// ListLinkGroups 获取联动组列表。
// GET /api/v1/sim-link-groups
func (h *TemplateHandler) ListLinkGroups(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.scenarioService.ListLinkGroups(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// GetLinkGroup 获取联动组详情。
// GET /api/v1/sim-link-groups/:id
func (h *TemplateHandler) GetLinkGroup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.scenarioService.GetLinkGroup(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListTags 获取标签列表。
// GET /api/v1/tags
func (h *TemplateHandler) ListTags(c *gin.Context) {
	var req dto.TagListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.templateSubService.ListTags(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateTag 创建自定义标签。
// POST /api/v1/tags
func (h *TemplateHandler) CreateTag(c *gin.Context) {
	var req dto.CreateTagReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.templateSubService.CreateTag(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// ListSharedTemplates 获取共享实验模板列表。
// GET /api/v1/shared-experiment-templates
func (h *TemplateHandler) ListSharedTemplates(c *gin.Context) {
	var req dto.SharedTemplateListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.templateService.ListShared(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetSharedTemplate 获取共享实验模板详情。
// GET /api/v1/shared-experiment-templates/:id
func (h *TemplateHandler) GetSharedTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.templateService.GetSharedByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListSchoolImages 获取学校管理员视角的本校镜像列表。
// GET /api/v1/school/images
func (h *TemplateHandler) ListSchoolImages(c *gin.Context) {
	var req dto.SchoolImageListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.imageService.ListSchoolImages(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}
