// context.go
// 服务层上下文
// 用于 handler → service 层的参数传递
// service 层不依赖 *gin.Context，通过 ServiceContext 获取用户信息
// 遵循分层架构规范：service 层不引用任何 HTTP 相关类型

package context

import (
	"context"
)

// ServiceContext 服务层上下文
// handler 层从 gin.Context 提取信息后构建此结构体传给 service 层
type ServiceContext struct {
	Ctx      context.Context // 标准 context（用于超时控制、取消等）
	UserID   int64           // 当前用户ID
	SchoolID int64           // 当前用户所属学校ID（超管为0）
	Roles    []string        // 当前用户角色列表
	ClientIP string          // 客户端IP（用于审计日志）
}

// NewServiceContext 创建服务层上下文
func NewServiceContext(ctx context.Context, userID, schoolID int64, roles []string) *ServiceContext {
	return &ServiceContext{
		Ctx:      ctx,
		UserID:   userID,
		SchoolID: schoolID,
		Roles:    roles,
	}
}

// WithClientIP 设置客户端IP
func (sc *ServiceContext) WithClientIP(ip string) *ServiceContext {
	sc.ClientIP = ip
	return sc
}

// IsSuperAdmin 判断是否为超级管理员
func (sc *ServiceContext) IsSuperAdmin() bool {
	for _, role := range sc.Roles {
		if role == "super_admin" {
			return true
		}
	}
	return false
}

// IsSchoolAdmin 判断是否为学校管理员
func (sc *ServiceContext) IsSchoolAdmin() bool {
	for _, role := range sc.Roles {
		if role == "school_admin" {
			return true
		}
	}
	return false
}

// IsTeacher 判断是否为教师
func (sc *ServiceContext) IsTeacher() bool {
	for _, role := range sc.Roles {
		if role == "teacher" {
			return true
		}
	}
	return false
}

// IsStudent 判断是否为学生
func (sc *ServiceContext) IsStudent() bool {
	for _, role := range sc.Roles {
		if role == "student" {
			return true
		}
	}
	return false
}

// HasRole 判断是否拥有指定角色
func (sc *ServiceContext) HasRole(role string) bool {
	for _, r := range sc.Roles {
		if r == role {
			return true
		}
	}
	return false
}
