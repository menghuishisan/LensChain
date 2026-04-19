// request_context.go
// 该文件统一封装 HTTP 请求上下文里的认证、租户和会话字段读写规则，目的是让中间件、
// handler 和少量必须读取请求身份的基础工具使用同一套上下文键名，避免项目里出现多套
// `user_id` / `school_id` / `roles` 读写方式并存。

package requestctx

import "github.com/gin-gonic/gin"

const (
	// ContextKeyUserID 当前登录用户 ID。
	ContextKeyUserID = "user_id"
	// ContextKeySchoolID 当前登录用户所属学校 ID。
	ContextKeySchoolID = "school_id"
	// ContextKeyRoles 当前登录用户角色列表。
	ContextKeyRoles = "roles"
	// ContextKeyJTI 当前 Access Token / Sim WS Token 的 JTI。
	ContextKeyJTI = "jti"
	// ContextKeyTenantSchoolID 当前请求的租户学校 ID。
	ContextKeyTenantSchoolID = "tenant_school_id"
)

// AuthContext 表示认证中间件注入的请求身份信息。
type AuthContext struct {
	UserID   int64
	SchoolID int64
	Roles    []string
	JTI      string
}

// SetAuth 将认证信息写入 Gin Context。
func SetAuth(c *gin.Context, auth AuthContext) {
	c.Set(ContextKeyUserID, auth.UserID)
	c.Set(ContextKeySchoolID, auth.SchoolID)
	c.Set(ContextKeyRoles, auth.Roles)
	c.Set(ContextKeyJTI, auth.JTI)
}

// SetTempAuth 将临时令牌中的最小身份信息写入 Gin Context。
func SetTempAuth(c *gin.Context, userID int64) {
	c.Set(ContextKeyUserID, userID)
}

// SetTenantSchoolID 写入当前请求的租户学校 ID。
func SetTenantSchoolID(c *gin.Context, schoolID int64) {
	c.Set(ContextKeyTenantSchoolID, schoolID)
}

// GetUserID 获取当前用户 ID。
func GetUserID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeyUserID); exists {
		if userID, ok := v.(int64); ok {
			return userID
		}
	}
	return 0
}

// GetSchoolID 获取当前用户所属学校 ID。
func GetSchoolID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeySchoolID); exists {
		if schoolID, ok := v.(int64); ok {
			return schoolID
		}
	}
	return 0
}

// GetRoles 获取当前用户角色列表。
func GetRoles(c *gin.Context) []string {
	if v, exists := c.Get(ContextKeyRoles); exists {
		if roles, ok := v.([]string); ok {
			return roles
		}
	}
	return nil
}

// GetJTI 获取当前令牌的 JTI。
func GetJTI(c *gin.Context) string {
	if v, exists := c.Get(ContextKeyJTI); exists {
		if jti, ok := v.(string); ok {
			return jti
		}
	}
	return ""
}

// GetTenantSchoolID 获取当前请求的租户学校 ID。
// 返回 0 表示当前请求不受学校租户限制。
func GetTenantSchoolID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeyTenantSchoolID); exists {
		if schoolID, ok := v.(int64); ok {
			return schoolID
		}
	}
	return 0
}
