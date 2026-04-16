// jwt.go
// JWT 鉴权中间件
// 从 Authorization: Bearer <token> 头中提取并验证 Access Token
// 将用户信息（UserID、SchoolID、Roles）注入到 gin.Context 中
// 公开接口（登录、SSO回调等）不经过此中间件

package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/tokenstate"
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
		blacklisted, err := cache.Exists(context.Background(), cache.KeyTokenBlacklist+jti)
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
		validAfter, err := loadTokenValidAfter(context.Background(), userID)
		if err == nil && validAfter != nil && hasIssuedAt && issuedAt.Before(*validAfter) {
			response.Abort(c, errcode.ErrRefreshTokenInvalid.WithMessage("账号已在其他设备登录或会话已失效"))
			return
		}

		// 将用户信息注入到 Context
		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeySchoolID, schoolID)
		c.Set(ContextKeyRoles, roles)
		c.Set(ContextKeyJTI, jti)

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

// loadTokenValidAfter 加载用户Token生效时间基线
// 优先读取缓存，缓存不存在时回源数据库并回填缓存。
func loadTokenValidAfter(ctx context.Context, userID int64) (*time.Time, error) {
	validAfter, err := tokenstate.GetTokenValidAfter(ctx, userID)
	if err != nil {
		return nil, err
	}
	if validAfter != nil {
		return validAfter, nil
	}

	var user entity.User
	err = database.Get().WithContext(ctx).
		Select("token_valid_after").
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}

	_ = tokenstate.SetTokenValidAfter(ctx, userID, user.TokenValidAfter)
	return &user.TokenValidAfter, nil
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
