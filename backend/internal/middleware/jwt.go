// jwt.go
// 该文件实现平台的 JWT 鉴权中间件，负责从请求头或受控的 query token 中解析访问令牌，
// 校验黑名单和失效基线，并把当前登录用户信息写入请求上下文。它只负责认证与身份注入，
// 不负责编排具体业务权限判断。

package middleware

import (
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
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从 Authorization 头提取 Token，WebSocket 场景允许退化为 query token。
		tokenString, hasToken := extractBearerToken(c)
		if !hasToken {
			response.Abort(c, errcode.ErrUnauthorized)
			return
		}

		// 解析 Access Token；SimEngine WebSocket 场景允许使用会话级 sim_ws token。
		claims, simClaims, err := parseJWTClaims(tokenString)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				response.Abort(c, errcode.ErrTokenExpired)
			} else {
				response.Abort(c, errcode.ErrTokenInvalid)
			}
			return
		}

		userID := claims.UserID
		schoolID := claims.SchoolID
		roles := claims.Roles
		jti := claims.ID
		if simClaims != nil {
			userID = simClaims.UserID
			schoolID = simClaims.SchoolID
			roles = simClaims.Roles
			jti = simClaims.ID
		}

		// 检查 Token 是否在黑名单中（被踢下线的 Token）
		blacklisted, err := tokenstate.IsTokenBlacklisted(c.Request.Context(), jti)
		if err == nil && blacklisted {
			response.Abort(c, errcode.ErrTokenBlacklist)
			return
		}

		var issuedAt time.Time
		var hasIssuedAt bool
		if claims != nil {
			if claims.IssuedAt != nil {
				issuedAt = claims.IssuedAt.Time
				hasIssuedAt = true
			}
		}
		if simClaims != nil {
			if simClaims.IssuedAt != nil {
				issuedAt = simClaims.IssuedAt.Time
				hasIssuedAt = true
			}
		}
		validAfter, err := tokenstate.ResolveTokenValidAfter(c.Request.Context(), userID)
		if err == nil && validAfter != nil && hasIssuedAt && issuedAt.Before(*validAfter) {
			response.Abort(c, errcode.ErrRefreshTokenInvalid.WithMessage("账号已在其他设备登录或会话已失效"))
			return
		}

		// 将用户信息注入到 Context
		requestctx.SetAuth(c, requestctx.AuthContext{
			UserID:   userID,
			SchoolID: schoolID,
			Roles:    roles,
			JTI:      jti,
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

	queryToken := strings.TrimSpace(c.Query("token"))
	if queryToken == "" {
		return "", false
	}
	return queryToken, true
}

// parseJWTClaims 解析访问令牌或 SimEngine WebSocket 会话令牌。
func parseJWTClaims(tokenString string) (*jwtpkg.Claims, *jwtpkg.SimWSClaims, error) {
	claims, err := jwtpkg.ParseAccessToken(tokenString)
	if err == nil {
		return claims, nil, nil
	}

	simClaims, simErr := jwtpkg.ParseSimWSToken(tokenString)
	if simErr == nil {
		return nil, simClaims, nil
	}

	return nil, nil, err
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
