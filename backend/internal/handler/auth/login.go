// login.go
// 模块01 — 用户与认证：认证相关 HTTP 处理器
// 负责登录、登出、Token刷新、密码修改等接口
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/auth"
)

// AuthHandler 认证处理器
// 处理 /api/v1/auth 路径下的认证相关请求
type AuthHandler struct {
	authService svc.AuthService
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService svc.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 手机号+密码登录
// POST /api/v1/auth/login
// 验证手机号和密码，返回 Access Token 和 Refresh Token
// 首次登录返回临时 Token，需先强制改密
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if !validator.BindJSON(c, &req) {
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.authService.Login(c.Request.Context(), req.Phone, req.Password, ip, userAgent)
	if err != nil {
		handleError(c, err)
		return
	}

	// 判断返回类型：首次登录强制改密 or 正常登录
	if result.IsFirstLogin {
		response.SuccessWithMsg(c, "首次登录，请修改密码", result.ForceResp)
	} else {
		response.SuccessWithMsg(c, "登录成功", result.TokenResp)
	}
}

// Logout 登出
// POST /api/v1/auth/logout
// 将当前 Access Token 加入黑名单，删除 Session
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	jti := middleware.GetJTI(c)
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	if err := h.authService.Logout(c.Request.Context(), userID, jti, ip, userAgent); err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "登出成功", nil)
}

// RefreshToken 刷新Token
// POST /api/v1/auth/token/refresh
// 使用 Refresh Token 获取新的 Token 对
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenReq
	if !validator.BindJSON(c, &req) {
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, resp)
}

// ChangePassword 修改密码
// POST /api/v1/auth/change-password
// 已登录用户修改自己的密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordReq
	if !validator.BindJSON(c, &req) {
		return
	}

	userID := middleware.GetUserID(c)
	ip := c.ClientIP()

	if err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword, ip); err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "密码修改成功", nil)
}

// ForceChangePassword 首次登录强制改密
// POST /api/v1/auth/force-change-password
// 使用临时 Token 认证，改密成功后返回正式的双 Token
func (h *AuthHandler) ForceChangePassword(c *gin.Context) {
	var req dto.ForceChangePasswordReq
	if !validator.BindJSON(c, &req) {
		return
	}

	userID := middleware.GetUserID(c)
	ip := c.ClientIP()

	result, err := h.authService.ForceChangePassword(c.Request.Context(), userID, req.NewPassword, ip)
	if err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "密码修改成功", result.TokenResp)
}

// SSOLogin SSO登录跳转
// GET /api/v1/auth/sso/:school_id/login
// 重定向到学校SSO登录页面
func (h *AuthHandler) SSOLogin(c *gin.Context) {
	schoolID, ok := validator.ParsePathID(c, "school_id")
	if !ok {
		return
	}

	loginURL, err := h.authService.SSOLoginURL(c.Request.Context(), schoolID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Redirect(http.StatusFound, loginURL)
}

// SSOCallback SSO回调
// GET /api/v1/auth/sso/callback
// 处理SSO系统回调，验证ticket/code，完成登录
func (h *AuthHandler) SSOCallback(c *gin.Context) {
	schoolID := validator.ParseQueryInt64(c, "school_id", 0)
	if schoolID <= 0 {
		handleError(c, errcode.ErrInvalidParams.WithMessage("缺少学校ID"))
		return
	}

	query := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}

	result, err := h.authService.SSOCallback(c.Request.Context(), schoolID, query, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		handleError(c, err)
		return
	}

	if result.IsFirstLogin {
		response.SuccessWithMsg(c, "首次登录，请修改密码", result.ForceResp)
		return
	}
	response.SuccessWithMsg(c, "登录成功", result.TokenResp)
}
