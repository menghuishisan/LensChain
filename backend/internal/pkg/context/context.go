// context.go
// 该文件定义 service 层内部使用的服务上下文对象，用来承接 handler 从 HTTP 请求中提取出的
// 用户身份、租户信息和客户端来源。它的目的不是替代标准 context.Context，而是把 service
// 层真正需要的业务身份信息从 Gin 上下文中剥离出来，保证 service 不依赖任何 HTTP 类型，
// 同时又能拿到当前操作者与租户边界。

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
