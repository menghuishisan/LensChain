// auth.go
// 该文件集中定义模块01“用户与认证”在 API 文档中约定的业务错误码，供 handler/service
// 直接返回统一错误响应。把认证错误单独放在这里，是为了保证登录、登出、改密、导入导出
// 等能力在不同接口中复用同一套错误语义。

package errcode

import "net/http"

var (
	// 认证相关
	ErrWrongCredentials   = New(40101, http.StatusUnauthorized, "用户名或密码错误")
	ErrAccountDisabled    = New(40102, http.StatusUnauthorized, "账号已被禁用")
	ErrAccountArchived    = New(40103, http.StatusUnauthorized, "账号已被归档")
	ErrAccountLocked      = New(40104, http.StatusUnauthorized, "账号已被锁定，请稍后再试")
	ErrLoginAttemptsLeft  = New(40105, http.StatusUnauthorized, "密码错误，剩余尝试次数不足")
	ErrRefreshTokenExpired = New(40106, http.StatusUnauthorized, "Refresh Token已过期，请重新登录")
	ErrRefreshTokenInvalid = New(40107, http.StatusUnauthorized, "Refresh Token无效或已被其他设备替换")
	ErrSSOAuthFailed      = New(40108, http.StatusUnauthorized, "SSO认证失败")
	ErrSSOAccountNotFound = New(40109, http.StatusUnauthorized, "SSO账号未在平台中绑定")

	// 密码相关
	ErrWrongOldPassword   = New(40011, http.StatusBadRequest, "原密码错误")
	ErrPasswordComplexity = New(40012, http.StatusBadRequest, "密码不满足复杂度要求")
	ErrPasswordSameAsCurrent = New(40013, http.StatusBadRequest, "新密码不能与当前密码相同")

	// 用户管理
	ErrUserNotFound       = New(40401, http.StatusNotFound, "用户不存在")
	ErrDuplicatePhone     = New(40901, http.StatusConflict, "手机号已存在")
	ErrDuplicateStudentNo = New(40902, http.StatusConflict, "学号已存在")
)
