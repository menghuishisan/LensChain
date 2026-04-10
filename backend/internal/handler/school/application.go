// application.go
// 模块02 — 学校与租户管理：入驻申请 HTTP 处理器
// 负责提交申请、查询状态、重新申请、申请列表/详情、审核通过/拒绝
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	svc "github.com/lenschain/backend/internal/service/school"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
)

// ApplicationHandler 入驻申请处理器
// 处理 /api/v1/school-applications 和 /api/v1/admin/school-applications 路径下的请求
type ApplicationHandler struct {
	appService svc.ApplicationService
}

// NewApplicationHandler 创建入驻申请处理器实例
func NewApplicationHandler(appService svc.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{appService: appService}
}

// Submit 提交入驻申请
// POST /api/v1/school-applications
// 公开接口，无需认证
func (h *ApplicationHandler) Submit(c *gin.Context) {
	var req dto.SubmitApplicationReq
	if !validator.BindJSON(c, &req) {
		return
	}

	resp, err := h.appService.Submit(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, resp)
}

// Query 查询申请状态
// GET /api/v1/school-applications/query
// 公开接口，通过手机号+短信验证码查询
func (h *ApplicationHandler) Query(c *gin.Context) {
	var req dto.QueryApplicationReq
	if !validator.BindQuery(c, &req) {
		return
	}

	// TODO: 验证短信验证码（SMS验证逻辑在此处完成）
	// 当前阶段跳过短信验证，直接查询

	resp, err := h.appService.Query(c.Request.Context(), req.Phone)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, resp)
}

// Reapply 重新申请
// POST /api/v1/school-applications/:id/reapply
// 公开接口，仅已拒绝的申请可重新提交
func (h *ApplicationHandler) Reapply(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.ReapplyReq
	if !validator.BindJSON(c, &req) {
		return
	}

	// TODO: 验证短信验证码

	resp, err := h.appService.Reapply(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, resp)
}

// List 申请列表
// GET /api/v1/admin/school-applications
// 超管接口
func (h *ApplicationHandler) List(c *gin.Context) {
	var req dto.ApplicationListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	items, total, err := h.appService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	// 规范化分页参数
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	response.Paginated(c, items, total, page, pageSize)
}

// GetByID 申请详情
// GET /api/v1/admin/school-applications/:id
// 超管接口
func (h *ApplicationHandler) GetByID(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := buildServiceContext(c)
	detail, err := h.appService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, detail)
}

// Approve 审核通过
// POST /api/v1/admin/school-applications/:id/approve
// 超管接口：通过申请 → 创建学校 → 创建首个校管 → 发送短信
func (h *ApplicationHandler) Approve(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.ApproveApplicationReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	resp, err := h.appService.Approve(c.Request.Context(), sc, id, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, resp)
}

// Reject 审核拒绝
// POST /api/v1/admin/school-applications/:id/reject
// 超管接口
func (h *ApplicationHandler) Reject(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.RejectApplicationReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	if err := h.appService.Reject(c.Request.Context(), sc, id, &req); err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "已拒绝该申请", nil)
}
