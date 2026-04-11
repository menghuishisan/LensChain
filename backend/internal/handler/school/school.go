// school.go
// 模块02 — 学校与租户管理：学校管理 HTTP 处理器
// 负责学校 CRUD、冻结/解冻、注销/恢复、有效期设置、本校信息、授权状态、SSO学校列表
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"github.com/gin-gonic/gin"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/school"
)

// SchoolHandler 学校管理处理器
// 处理 /api/v1/admin/schools 和 /api/v1/school 路径下的请求
type SchoolHandler struct {
	schoolService svc.SchoolService
}

// NewSchoolHandler 创建学校管理处理器实例
func NewSchoolHandler(schoolService svc.SchoolService) *SchoolHandler {
	return &SchoolHandler{schoolService: schoolService}
}

// ========== 超管接口 ==========

// List 学校列表
// GET /api/v1/admin/schools
func (h *SchoolHandler) List(c *gin.Context) {
	var req dto.SchoolListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.schoolService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// Create 后台直接创建学校
// POST /api/v1/admin/schools
func (h *SchoolHandler) Create(c *gin.Context) {
	var req dto.CreateSchoolReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.schoolService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Created(c, resp)
}

// GetByID 学校详情
// GET /api/v1/admin/schools/:id
func (h *SchoolHandler) GetByID(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.schoolService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, detail)
}

// Update 编辑学校信息
// PUT /api/v1/admin/schools/:id
func (h *SchoolHandler) Update(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateSchoolReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "更新成功", nil)
}

// SetLicense 设置有效期
// PATCH /api/v1/admin/schools/:id/license
func (h *SchoolHandler) SetLicense(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.SetLicenseReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.SetLicense(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "有效期设置成功", nil)
}

// Freeze 冻结学校
// POST /api/v1/admin/schools/:id/freeze
func (h *SchoolHandler) Freeze(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.FreezeSchoolReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.Freeze(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "学校已冻结", nil)
}

// Unfreeze 解冻学校
// POST /api/v1/admin/schools/:id/unfreeze
func (h *SchoolHandler) Unfreeze(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.Unfreeze(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "学校已解冻", nil)
}

// Cancel 注销学校
// POST /api/v1/admin/schools/:id/cancel
func (h *SchoolHandler) Cancel(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.CancelSchoolReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.Cancel(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "学校已注销", nil)
}

// Restore 恢复已注销学校
// POST /api/v1/admin/schools/:id/restore
func (h *SchoolHandler) Restore(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.Restore(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "学校已恢复", nil)
}

// ========== 校管接口 ==========

// GetProfile 获取本校信息
// GET /api/v1/school/profile
func (h *SchoolHandler) GetProfile(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.schoolService.GetProfile(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, resp)
}

// UpdateProfile 编辑本校信息
// PUT /api/v1/school/profile
// 校管仅可修改 logo/description/address/website
func (h *SchoolHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateSchoolProfileReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.schoolService.UpdateProfile(c.Request.Context(), sc, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "更新成功", nil)
}

// GetLicenseStatus 查看授权状态
// GET /api/v1/school/license
func (h *SchoolHandler) GetLicenseStatus(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.schoolService.GetLicenseStatus(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, resp)
}

// ========== 公开接口 ==========

// GetSSOSchoolList 获取已配置SSO的学校列表
// GET /api/v1/schools/sso-list
// 公开接口，无需认证
func (h *SchoolHandler) GetSSOSchoolList(c *gin.Context) {
	items, err := h.schoolService.GetSSOSchoolList(c.Request.Context())
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"list": items})
}
