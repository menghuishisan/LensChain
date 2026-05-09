// jwt.go
// 该文件实现平台的 JWT 鉴权中间件，负责从请求头或受控的 query token 中解析访问令牌，
// 校验黑名单和失效基线，并把当前登录用户信息写入请求上下文。它只负责认证与身份注入，
// 不负责编排具体业务权限判断。

package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/requestctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/tokenstate"
)

// JWTAuth JWT 鉴权中间件
//
// 仅接受 Access Token。SimWS Token（jwtpkg.SimWSClaims）由 backend 自己签发并仅
// 用于 backend 代理 → SimEngine Core 一跳，永远不会从外部回到 backend，故不在此
// 中间件解析。这样既消除了"低权限 SimWS token 被误用为通用 HTTP 鉴权"的攻击面，
// 也避免了维护两套 token 解析路径的复杂度。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// extractBearerToken 内部已经处理：
		//   1) Authorization: Bearer xxx 头部（默认路径）
		//   2) WebSocket 握手退化为 ?token=xxx query 参数（浏览器 WebSocket API 不能发送
		//      Authorization 头，必须用 query token；由 allowQueryToken 守卫，仅 WS 路径或带
		//      Connection: upgrade + Upgrade: websocket 的 GET 才被允许，避免泛化为通用入口）
		// 失败统一 401，不做二次重复读取。
		tokenString, ok := extractBearerToken(c)
		if !ok {
			response.Abort(c, errcode.ErrUnauthorized)
			return
		}

		claims, err := jwtpkg.ParseAccessToken(tokenString)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				response.Abort(c, errcode.ErrTokenExpired)
			} else {
				response.Abort(c, errcode.ErrTokenInvalid)
			}
			return
		}

		// 检查 Token 是否在黑名单中（被踢下线的 Token）
		blacklisted, blackErr := tokenstate.IsTokenBlacklisted(c.Request.Context(), claims.ID)
		if blackErr == nil && blacklisted {
			response.Abort(c, errcode.ErrTokenBlacklist)
			return
		}

		var issuedAt time.Time
		hasIssuedAt := false
		if claims.IssuedAt != nil {
			issuedAt = claims.IssuedAt.Time
			hasIssuedAt = true
		}
		validAfter, vaErr := tokenstate.ResolveTokenValidAfter(c.Request.Context(), claims.UserID)
		if vaErr == nil && validAfter != nil && hasIssuedAt && issuedAt.Before(*validAfter) {
			response.Abort(c, errcode.ErrRefreshTokenInvalid.WithMessage("账号已在其他设备登录或会话已失效"))
			return
		}

		// 将用户信息注入到 Context
		requestctx.SetAuth(c, requestctx.AuthContext{
			UserID:   claims.UserID,
			SchoolID: claims.SchoolID,
			Roles:    claims.Roles,
			JTI:      claims.ID,
		})

		c.Next()
	}
}

// extractBearerToken 提取请求中的 JWT。
// 常规 HTTP 接口使用 Authorization 头，WebSocket 握手允许使用 query token。
func extractBearerToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return "", false
		}
		return parts[1], true
	}

	if !allowQueryToken(c) {
		return "", false
	}

	queryToken := strings.TrimSpace(c.Query("token"))
	if queryToken == "" {
		return "", false
	}
	return queryToken, true
}

// allowQueryToken 仅在受控的 WebSocket 握手场景允许通过 query 传递 token。
// 普通 HTTP 接口必须使用 Authorization 头，避免把 query token 放大成通用认证入口。
func allowQueryToken(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/api/v1/ws/") {
		return true
	}

	connectionHeader := strings.ToLower(c.GetHeader("Connection"))
	upgradeHeader := strings.ToLower(c.GetHeader("Upgrade"))
	return strings.Contains(connectionHeader, "upgrade") && upgradeHeader == "websocket" && c.Request.Method == http.MethodGet
}

// TempTokenAuth 临时Token鉴权中间件（首次登录改密专用）
func TempTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Abort(c, errcode.ErrUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Abort(c, errcode.ErrTokenInvalid)
			return
		}

		claims, err := jwtpkg.ParseTempToken(parts[1])
		if err != nil {
			response.Abort(c, errcode.ErrTokenInvalid)
			return
		}

		requestctx.SetTempAuth(c, claims.UserID)
		c.Next()
	}
}
