// application.go
// 模块02 — 学校与租户管理：入驻申请 HTTP 处理器
// 负责提交申请、查询状态、重新申请、申请列表/详情、审核通过/拒绝
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"github.com/gin-gonic/gin"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/sms"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/school"
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
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "申请提交成功", resp)
}

// SendSMSCode 发送查询验证码
// POST /api/v1/school-applications/send-sms-code
// 公开接口，用于查询申请状态和重新申请前的验证码发送。
func (h *ApplicationHandler) SendSMSCode(c *gin.Context) {
	var req dto.SendSMSCodeReq
	if !validator.BindJSON(c, &req) {
		return
	}

	if err := h.appService.SendSMSCode(c.Request.Context(), req.Phone); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "验证码发送成功", nil)
}

// Query 查询申请状态
// GET /api/v1/school-applications/query
// 公开接口，通过手机号+短信验证码查询
func (h *ApplicationHandler) Query(c *gin.Context) {
	var req dto.QueryApplicationReq
	if !validator.BindQuery(c, &req) {
		return
	}

	if err := sms.VerifyCode(c.Request.Context(), req.Phone, req.SMSCode); err != nil {
		handlerctx.HandleError(c, errcode.ErrInvalidParams.WithMessage("短信验证码错误或已过期"))
		return
	}

	resp, err := h.appService.Query(c.Request.Context(), req.Phone)
	if err != nil {
		handlerctx.HandleError(c, err)
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

	if err := sms.VerifyCode(c.Request.Context(), req.ContactPhone, req.SMSCode); err != nil {
		handlerctx.HandleError(c, errcode.ErrInvalidParams.WithMessage("短信验证码错误或已过期"))
		return
	}

	resp, err := h.appService.Reapply(c.Request.Context(), id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "重新申请已提交", resp)
}

// List 申请列表
// GET /api/v1/admin/school-applications
// 超管接口
func (h *ApplicationHandler) List(c *gin.Context) {
	var req dto.ApplicationListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.appService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
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

	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.appService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
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

	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.appService.Approve(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "审核通过", resp)
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

	sc := handlerctx.BuildServiceContext(c)
	if err := h.appService.Reject(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "已拒绝该申请", nil)
}
