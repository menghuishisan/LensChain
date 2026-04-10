// security.go
// 模块01 — 用户与认证：安全策略与日志 HTTP 处理器
// 负责安全策略配置管理、登录日志查询、操作日志查询
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	svc "github.com/lenschain/backend/internal/service/auth"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
)

// SecurityHandler 安全策略与日志处理器
// 处理安全策略配置和日志查询请求
type SecurityHandler struct {
	securityService svc.SecurityService
}

// NewSecurityHandler 创建安全策略处理器实例
func NewSecurityHandler(securityService svc.SecurityService) *SecurityHandler {
	return &SecurityHandler{securityService: securityService}
}

// GetSecurityPolicy 获取安全策略配置
// GET /api/v1/security-policies
func (h *SecurityHandler) GetSecurityPolicy(c *gin.Context) {
	policy, err := h.securityService.GetSecurityPolicy(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, policy)
}

// UpdateSecurityPolicy 更新安全策略配置
// PUT /api/v1/security-policies
func (h *SecurityHandler) UpdateSecurityPolicy(c *gin.Context) {
	var req dto.UpdateSecurityPolicyReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	if err := h.securityService.UpdateSecurityPolicy(c.Request.Context(), sc, &req); err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "安全策略更新成功", nil)
}

// ListLoginLogs 登录日志列表
// GET /api/v1/login-logs
func (h *SecurityHandler) ListLoginLogs(c *gin.Context) {
	var req dto.LoginLogListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	items, total, err := h.securityService.ListLoginLogs(c.Request.Context(), sc, &req)
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

// ListOperationLogs 操作日志列表
// GET /api/v1/operation-logs
func (h *SecurityHandler) ListOperationLogs(c *gin.Context) {
	var req dto.OperationLogListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	items, total, err := h.securityService.ListOperationLogs(c.Request.Context(), sc, &req)
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
