// rbac.go
// 该文件实现平台基础 RBAC 角色校验中间件，负责根据请求上下文中已注入的角色集合判断某个
// 路由是否允许访问。它只做通用角色门禁，不在这里放接口特有的业务条件或资源级权限判断。

package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/requestctx"
	"github.com/lenschain/backend/internal/pkg/response"
)

// 角色常量
const (
	RoleSuperAdmin  = "super_admin"
	RoleSchoolAdmin = "school_admin"
	RoleTeacher     = "teacher"
	RoleStudent     = "student"
)

// RequireRoles 要求用户拥有指定角色之一
// 传入多个角色表示"或"关系，用户拥有其中任一角色即可通过
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := requestctx.GetRoles(c)
		if len(userRoles) == 0 {
			response.Abort(c, errcode.ErrForbidden)
			return
		}

		// 检查用户是否拥有所需角色之一
		for _, required := range roles {
			for _, userRole := range userRoles {
				if userRole == required {
					c.Next()
					return
				}
			}
		}

		response.Abort(c, errcode.ErrForbidden)
	}
}

// RequireSuperAdmin 要求超级管理员权限
func RequireSuperAdmin() gin.HandlerFunc {
	return RequireRoles(RoleSuperAdmin)
}

// RequireSchoolAdmin 要求学校管理员权限
func RequireSchoolAdmin() gin.HandlerFunc {
	return RequireRoles(RoleSchoolAdmin)
}

// RequireTeacher 要求教师权限
func RequireTeacher() gin.HandlerFunc {
	return RequireRoles(RoleTeacher)
}

// RequireStudent 要求学生权限
func RequireStudent() gin.HandlerFunc {
	return RequireRoles(RoleStudent)
}

// RequireAdminOrTeacher 要求管理员或教师权限
func RequireAdminOrTeacher() gin.HandlerFunc {
	return RequireRoles(RoleSuperAdmin, RoleSchoolAdmin, RoleTeacher)
}

// RequireSchoolAdminOrSuperAdmin 要求学校管理员或超级管理员权限
func RequireSchoolAdminOrSuperAdmin() gin.HandlerFunc {
	return RequireRoles(RoleSuperAdmin, RoleSchoolAdmin)
}

// IsSuperAdmin 判断当前用户是否为超级管理员
func IsSuperAdmin(c *gin.Context) bool {
	return hasRole(c, RoleSuperAdmin)
}

// IsSchoolAdmin 判断当前用户是否为学校管理员
func IsSchoolAdmin(c *gin.Context) bool {
	return hasRole(c, RoleSchoolAdmin)
}

// IsTeacher 判断当前用户是否为教师
func IsTeacher(c *gin.Context) bool {
	return hasRole(c, RoleTeacher)
}

// IsStudent 判断当前用户是否为学生
func IsStudent(c *gin.Context) bool {
	return hasRole(c, RoleStudent)
}

// hasRole 检查用户是否拥有指定角色
func hasRole(c *gin.Context, role string) bool {
	roles := requestctx.GetRoles(c)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
