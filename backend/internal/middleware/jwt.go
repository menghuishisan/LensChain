// jwt.go
// JWT 鉴权中间件
// 从 Authorization: Bearer <token> 头中提取并验证 Access Token
// 将用户信息（UserID、SchoolID、Roles）注入到 gin.Context 中
// 公开接口（登录、SSO回调等）不经过此中间件

package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/response"
)

// Context Key 常量
const (
	ContextKeyUserID   = "user_id"
	ContextKeySchoolID = "school_id"
	ContextKeyRoles    = "roles"
	ContextKeyJTI      = "jti"
)

// JWTAuth JWT 鉴权中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 头提取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Abort(c, errcode.ErrUnauthorized)
			return
		}

		// 格式：Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Abort(c, errcode.ErrTokenInvalid)
			return
		}

		tokenString := parts[1]

		// 解析 Access Token
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
		blacklisted, err := cache.Exists(context.Background(), cache.KeyTokenBlacklist+claims.ID)
		if err == nil && blacklisted {
			response.Abort(c, errcode.ErrTokenBlacklist)
			return
		}

		// 将用户信息注入到 Context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeySchoolID, claims.SchoolID)
		c.Set(ContextKeyRoles, claims.Roles)
		c.Set(ContextKeyJTI, claims.ID)

		c.Next()
	}
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

		c.Set(ContextKeyUserID, claims.UserID)
		c.Next()
	}
}

// GetUserID 从 Context 获取当前用户ID
func GetUserID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeyUserID); exists {
		return v.(int64)
	}
	return 0
}

// GetSchoolID 从 Context 获取当前用户的学校ID
func GetSchoolID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeySchoolID); exists {
		return v.(int64)
	}
	return 0
}

// GetRoles 从 Context 获取当前用户的角色列表
func GetRoles(c *gin.Context) []string {
	if v, exists := c.Get(ContextKeyRoles); exists {
		return v.([]string)
	}
	return nil
}

// GetJTI 从 Context 获取当前 Token 的 JTI
func GetJTI(c *gin.Context) string {
	if v, exists := c.Get(ContextKeyJTI); exists {
		return v.(string)
	}
	return ""
}
