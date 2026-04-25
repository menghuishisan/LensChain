// auth.go
// 模块01 — 用户与认证：请求/响应 DTO 定义
// 对照 docs/modules/01-用户与认证/03-API接口设计.md
// 所有 26 个端点的请求体和响应体结构

package dto

// ========== 认证接口 DTO ==========

// LoginReq 登录请求
// POST /api/v1/auth/login
type LoginReq struct {
	Phone    string `json:"phone" binding:"required,phone"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResp 登录成功响应（正常登录）
type LoginResp struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	User         LoginUser `json:"user"`
}

// LoginUser 登录响应中的用户信息
type LoginUser struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Phone        string   `json:"phone"`
	Roles        []string `json:"roles"`
	SchoolID     string   `json:"school_id"`
	SchoolName   string   `json:"school_name"`
	IsFirstLogin bool     `json:"is_first_login"`
}

// ForceChangePasswordResp 首次登录强制改密响应
type ForceChangePasswordResp struct {
	ForceChangePassword bool   `json:"force_change_password"`
	TempToken           string `json:"temp_token"`
	TempTokenExpiresIn  int64  `json:"temp_token_expires_in"`
}

// RefreshTokenReq 刷新Token请求
// POST /api/v1/auth/token/refresh
type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResp 刷新Token响应
type RefreshTokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// SSOCallbackReq SSO回调查询参数。
// CAS 与 OAuth2 回调参数不同，因此这里只表达平台明确消费的公共参数和两类协议的标准凭证。
type SSOCallbackReq struct {
	SchoolID string `form:"school_id"`
	Ticket   string `form:"ticket"`
	Code     string `form:"code"`
	State    string `form:"state"`
}

// SSOSchoolListReq SSO 学校列表查询参数
// GET /api/v1/auth/sso-schools
type SSOSchoolListReq struct {
	Keyword string `form:"keyword"`
}

// ========== 密码接口 DTO ==========

// ChangePasswordReq 修改密码请求
// POST /api/v1/auth/change-password
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,password"`
}

// ForceChangePasswordReq 首次登录强制改密请求
// POST /api/v1/auth/force-change-password
type ForceChangePasswordReq struct {
	NewPassword string `json:"new_password" binding:"required,password"`
}

// ========== 用户管理接口 DTO ==========

// UserListReq 用户列表查询参数
// GET /api/v1/users
type UserListReq struct {
	Page           int    `form:"page" binding:"omitempty,min=1"`
	PageSize       int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword        string `form:"keyword"`
	Status         int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	Role           string `form:"role" binding:"omitempty,oneof=teacher student"`
	College        string `form:"college"`
	EducationLevel int16  `form:"education_level" binding:"omitempty,oneof=1 2 3 4"`
	SortBy         string `form:"sort_by" binding:"omitempty"`
	SortOrder      string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// UserListItem 用户列表项
type UserListItem struct {
	ID                 string   `json:"id"`
	Phone              string   `json:"phone"`
	Name               string   `json:"name"`
	StudentNo          *string  `json:"student_no"`
	Status             int16    `json:"status"`
	StatusText         string   `json:"status_text"`
	Roles              []string `json:"roles"`
	College            *string  `json:"college"`
	Major              *string  `json:"major"`
	ClassName          *string  `json:"class_name"`
	EducationLevel     *int16   `json:"education_level"`
	EducationLevelText *string  `json:"education_level_text"`
	LastLoginAt        *string  `json:"last_login_at"`
	CreatedAt          string   `json:"created_at"`
}

// UserListResp 用户列表响应。
type UserListResp struct {
	List       []UserListItem `json:"list"`
	Pagination PaginationResp `json:"pagination"`
}

// CreateUserReq 创建用户请求
// POST /api/v1/users
type CreateUserReq struct {
	Phone          string  `json:"phone" binding:"required,phone"`
	Name           string  `json:"name" binding:"required,min=2,max=50"`
	Password       string  `json:"password" binding:"required,password"`
	Role           string  `json:"role" binding:"required,oneof=teacher student"`
	StudentNo      *string `json:"student_no" binding:"omitempty,max=50"`
	College        *string `json:"college" binding:"omitempty,max=100"`
	Major          *string `json:"major" binding:"omitempty,max=100"`
	ClassName      *string `json:"class_name" binding:"omitempty,max=50"`
	EducationLevel *int16  `json:"education_level" binding:"omitempty,oneof=1 2 3 4"`
	Email          *string `json:"email" binding:"omitempty,email"`
	Remark         *string `json:"remark"`
}

// CreateUserResp 创建用户响应
type CreateUserResp struct {
	ID string `json:"id"`
}

// CreateSuperAdminReq 创建超级管理员请求
// POST /api/v1/users/super-admins
type CreateSuperAdminReq struct {
	Phone    string  `json:"phone" binding:"required,phone"`
	Name     string  `json:"name" binding:"required,min=2,max=50"`
	Password string  `json:"password" binding:"required,password"`
	SchoolID *string `json:"school_id" binding:"omitempty"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Remark   *string `json:"remark"`
}

// UserDetailResp 用户详情响应
// GET /api/v1/users/:id
type UserDetailResp struct {
	ID                 string   `json:"id"`
	Phone              string   `json:"phone"`
	Name               string   `json:"name"`
	StudentNo          *string  `json:"student_no"`
	Status             int16    `json:"status"`
	StatusText         string   `json:"status_text"`
	IsFirstLogin       bool     `json:"is_first_login"`
	IsSchoolAdmin      bool     `json:"is_school_admin"`
	SchoolID           string   `json:"school_id"`
	Roles              []string `json:"roles"`
	AvatarURL          *string  `json:"avatar_url"`
	Nickname           *string  `json:"nickname"`
	Email              *string  `json:"email"`
	College            *string  `json:"college"`
	Major              *string  `json:"major"`
	ClassName          *string  `json:"class_name"`
	EnrollmentYear     *int16   `json:"enrollment_year"`
	EducationLevel     *int16   `json:"education_level"`
	EducationLevelText *string  `json:"education_level_text"`
	Grade              *int16   `json:"grade"`
	Remark             *string  `json:"remark"`
	LastLoginAt        *string  `json:"last_login_at"`
	CreatedAt          string   `json:"created_at"`
}

// UpdateUserReq 更新用户请求
// PUT /api/v1/users/:id
type UpdateUserReq struct {
	Name           *string `json:"name" binding:"omitempty,min=2,max=50"`
	StudentNo      *string `json:"student_no" binding:"omitempty,max=50"`
	College        *string `json:"college" binding:"omitempty,max=100"`
	Major          *string `json:"major" binding:"omitempty,max=100"`
	ClassName      *string `json:"class_name" binding:"omitempty,max=50"`
	EnrollmentYear *int16  `json:"enrollment_year"`
	EducationLevel *int16  `json:"education_level" binding:"omitempty,oneof=1 2 3 4"`
	Grade          *int16  `json:"grade"`
	Email          *string `json:"email" binding:"omitempty,email"`
	Remark         *string `json:"remark"`
}

// UpdateStatusReq 变更账号状态请求
// PATCH /api/v1/users/:id/status
type UpdateStatusReq struct {
	Status int16  `json:"status" binding:"required,oneof=1 2 3"`
	Reason string `json:"reason" binding:"omitempty,max=200"`
}

// ResetPasswordReq 重置密码请求
// POST /api/v1/users/:id/reset-password
type ResetPasswordReq struct {
	NewPassword string `json:"new_password" binding:"required,password"`
}

// BatchDeleteReq 批量删除请求
// POST /api/v1/users/batch-delete
type BatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// ========== 用户导入接口 DTO ==========

// ImportTemplateReq 用户导入模板下载查询参数。
// type 决定生成学生模板还是教师模板，模板文件本身由文件下载响应承载。
type ImportTemplateReq struct {
	Type string `form:"type" binding:"required,oneof=student teacher"`
}

// ImportPreviewReq 用户导入预览表单参数。
// 上传文件由 multipart 文件流承载，DTO 只表达接口文档中的业务表单字段。
type ImportPreviewReq struct {
	Type string `form:"type" binding:"required,oneof=student teacher"`
}

// ImportPreviewResp 导入预览响应
// POST /api/v1/user-imports/preview
type ImportPreviewResp struct {
	ImportID    string             `json:"import_id"`
	Total       int                `json:"total"`
	Valid       int                `json:"valid"`
	Invalid     int                `json:"invalid"`
	Conflict    int                `json:"conflict"`
	PreviewList []ImportPreviewRow `json:"preview_list"`
}

// ImportPreviewRow 导入预览行
type ImportPreviewRow struct {
	Row       int     `json:"row"`
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	StudentNo string  `json:"student_no"`
	Status    string  `json:"status"`
	Message   *string `json:"message"`
}

// ImportExecuteReq 确认执行导入请求
// POST /api/v1/user-imports/execute
type ImportExecuteReq struct {
	ImportID          string   `json:"import_id" binding:"required"`
	ConflictStrategy  string   `json:"conflict_strategy" binding:"required,oneof=skip overwrite"`
	ConflictOverrides []string `json:"conflict_overrides"`
}

// ImportExecuteResp 导入执行结果响应
type ImportExecuteResp struct {
	ImportID       string `json:"import_id"`
	SuccessCount   int    `json:"success_count"`
	FailCount      int    `json:"fail_count"`
	SkipCount      int    `json:"skip_count"`
	OverwriteCount int    `json:"overwrite_count"`
}

// ImportFailureRow 导入失败行明细
// GET /api/v1/user-imports/:id/failures
type ImportFailureRow struct {
	Row            int    `json:"row"`
	Name           string `json:"name"`
	Phone          string `json:"phone"`
	StudentNo      string `json:"student_no"`
	College        string `json:"college"`
	Major          string `json:"major"`
	ClassName      string `json:"class_name"`
	EnrollmentYear string `json:"enrollment_year"`
	EducationLevel string `json:"education_level"`
	Grade          string `json:"grade"`
	Email          string `json:"email"`
	Remark         string `json:"remark"`
	FailReason     string `json:"fail_reason"`
}

// ========== 个人中心接口 DTO ==========

// ProfileResp 个人信息响应
// GET /api/v1/profile
type ProfileResp struct {
	ID                 string   `json:"id"`
	Phone              string   `json:"phone"`
	Name               string   `json:"name"`
	Nickname           *string  `json:"nickname"`
	AvatarURL          *string  `json:"avatar_url"`
	Email              *string  `json:"email"`
	StudentNo          *string  `json:"student_no"`
	SchoolName         string   `json:"school_name"`
	College            *string  `json:"college"`
	Major              *string  `json:"major"`
	ClassName          *string  `json:"class_name"`
	EducationLevel     *int16   `json:"education_level"`
	EducationLevelText *string  `json:"education_level_text"`
	Roles              []string `json:"roles"`
}

// UpdateProfileReq 更新个人信息请求
// PUT /api/v1/profile
type UpdateProfileReq struct {
	Nickname  *string `json:"nickname" binding:"omitempty,max=50"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url,max=500"`
	Email     *string `json:"email" binding:"omitempty,email"`
}

// ========== 安全策略接口 DTO ==========

// SecurityPolicyResp 安全策略响应
// GET /api/v1/security-policies
type SecurityPolicyResp struct {
	LoginFailMaxCount          int  `json:"login_fail_max_count"`
	LoginLockDurationMinutes   int  `json:"login_lock_duration_minutes"`
	PasswordMinLength          int  `json:"password_min_length"`
	PasswordRequireUppercase   bool `json:"password_require_uppercase"`
	PasswordRequireLowercase   bool `json:"password_require_lowercase"`
	PasswordRequireDigit       bool `json:"password_require_digit"`
	PasswordRequireSpecialChar bool `json:"password_require_special_char"`
	AccessTokenExpireMinutes   int  `json:"access_token_expire_minutes"`
	RefreshTokenExpireDays     int  `json:"refresh_token_expire_days"`
}

// UpdateSecurityPolicyReq 更新安全策略请求
// PUT /api/v1/security-policies（支持部分更新）
type UpdateSecurityPolicyReq struct {
	LoginFailMaxCount          *int  `json:"login_fail_max_count" binding:"omitempty,min=1,max=20"`
	LoginLockDurationMinutes   *int  `json:"login_lock_duration_minutes" binding:"omitempty,min=1,max=1440"`
	PasswordMinLength          *int  `json:"password_min_length" binding:"omitempty,min=6,max=32"`
	PasswordRequireUppercase   *bool `json:"password_require_uppercase"`
	PasswordRequireLowercase   *bool `json:"password_require_lowercase"`
	PasswordRequireDigit       *bool `json:"password_require_digit"`
	PasswordRequireSpecialChar *bool `json:"password_require_special_char"`
	AccessTokenExpireMinutes   *int  `json:"access_token_expire_minutes" binding:"omitempty,min=5,max=1440"`
	RefreshTokenExpireDays     *int  `json:"refresh_token_expire_days" binding:"omitempty,min=1,max=30"`
}

// ========== 日志接口 DTO ==========

// LoginLogListReq 登录日志列表查询参数
// GET /api/v1/login-logs
type LoginLogListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	UserID      string `form:"user_id"`
	Action      int16  `form:"action" binding:"omitempty,oneof=1 2 3 4 5"`
	CreatedFrom string `form:"created_from"`
	CreatedTo   string `form:"created_to"`
}

// LoginLogItem 登录日志列表项
type LoginLogItem struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	UserName        string  `json:"user_name"`
	Action          int16   `json:"action"`
	ActionText      string  `json:"action_text"`
	LoginMethod     *int16  `json:"login_method"`
	LoginMethodText *string `json:"login_method_text"`
	IP              string  `json:"ip"`
	UserAgent       *string `json:"user_agent"`
	FailReason      *string `json:"fail_reason"`
	CreatedAt       string  `json:"created_at"`
}

// LoginLogListResp 登录日志列表响应。
type LoginLogListResp struct {
	List       []LoginLogItem `json:"list"`
	Pagination PaginationResp `json:"pagination"`
}

// OperationLogListReq 操作日志列表查询参数
// GET /api/v1/operation-logs
type OperationLogListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	OperatorID  string `form:"operator_id"`
	Action      string `form:"action"`
	TargetType  string `form:"target_type"`
	CreatedFrom string `form:"created_from"`
	CreatedTo   string `form:"created_to"`
}

// OperationLogItem 操作日志列表项
type OperationLogItem struct {
	ID           string  `json:"id"`
	OperatorID   string  `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
	Action       string  `json:"action"`
	TargetType   string  `json:"target_type"`
	TargetID     *string `json:"target_id"`
	Detail       *string `json:"detail"`
	IP           string  `json:"ip"`
	CreatedAt    string  `json:"created_at"`
}

// OperationLogListResp 操作日志列表响应。
type OperationLogListResp struct {
	List       []OperationLogItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}
