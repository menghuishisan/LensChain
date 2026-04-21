// sso.go
// 模块02 — 学校与租户管理：SSO配置 HTTP 处理器
// 负责 SSO 配置的读取、更新、连接测试
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"github.com/gin-gonic/gin"
	"github.com/lenschain/backend/internal/pkg/handlerctx"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/school"
)

// SSOHandler SSO配置处理器
// 处理 /api/v1/school/sso-config 路径下的请求
type SSOHandler struct {
	ssoService svc.SSOService
}

// NewSSOHandler 创建SSO配置处理器实例
func NewSSOHandler(ssoService svc.SSOService) *SSOHandler {
	return &SSOHandler{ssoService: ssoService}
}

// GetConfig 获取SSO配置
// GET /api/v1/school/sso-config
// 校管接口，client_secret 脱敏显示为 ******
func (h *SSOHandler) GetConfig(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.ssoService.GetConfig(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, resp)
}

// UpdateConfig 更新SSO配置
// PUT /api/v1/school/sso-config
// 校管接口，client_secret 加密存储，更新后重置 is_tested = false
func (h *SSOHandler) UpdateConfig(c *gin.Context) {
	var req dto.UpdateSSOConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.ssoService.UpdateConfig(c.Request.Context(), sc, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "SSO配置已更新", nil)
}

// TestConnection 测试SSO连接
// POST /api/v1/school/sso-config/test
// 校管接口，根据协议类型（CAS/OAuth2）测试连接可达性
func (h *SSOHandler) TestConnection(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.ssoService.TestConnection(c.Request.Context(), sc)
	if err != nil {
		// 测试失败时按文档返回错误码，同时在 data 中附带错误详情
		if appErr, ok := errcode.IsAppError(err); ok && appErr.Is(errcode.ErrSSOTestFailed) && resp != nil {
			response.ErrorWithData(c, appErr, resp)
			return
		}
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "SSO连接测试成功", resp)
}

// ToggleEnable 启用或禁用SSO
// POST /api/v1/school/sso-config/enable
// 校管接口，仅允许启用已通过测试的SSO配置。
func (h *SSOHandler) ToggleEnable(c *gin.Context) {
	var req dto.ToggleSSOEnableReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.ssoService.ToggleEnable(c.Request.Context(), sc, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	if req.IsEnabled != nil && *req.IsEnabled {
		response.SuccessWithMsg(c, "SSO已启用", nil)
		return
	}
	response.SuccessWithMsg(c, "SSO已禁用", nil)
}
