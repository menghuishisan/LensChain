// user.go
// 模块01 — 用户与认证：数据库实体结构体
// 对照 docs/modules/01-用户与认证/02-数据库设计.md
// 包含 9 张表的 GORM 映射结构体

package entity

import (
	"time"

	"gorm.io/gorm"
)

// User 用户主表
// 对应 users 表，17个字段
type User struct {
	ID             int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Phone          string         `gorm:"type:varchar(20);not null" json:"phone"`
	PasswordHash   string         `gorm:"type:varchar(255);not null" json:"-"`
	Name           string         `gorm:"type:varchar(50);not null" json:"name"`
	SchoolID       int64          `gorm:"not null;index" json:"school_id,string"`
	StudentNo      *string        `gorm:"type:varchar(50)" json:"student_no,omitempty"`
	Status         int            `gorm:"type:smallint;not null;default:1" json:"status"`
	IsFirstLogin   bool           `gorm:"not null;default:true" json:"is_first_login"`
	IsSchoolAdmin  bool           `gorm:"not null;default:false" json:"is_school_admin"`
	LoginFailCount int            `gorm:"type:smallint;not null;default:0" json:"-"`
	LockedUntil    *time.Time     `gorm:"" json:"-"`
	LastLoginAt    *time.Time     `gorm:"" json:"last_login_at,omitempty"`
	LastLoginIP    *string        `gorm:"type:varchar(45)" json:"-"`
	CreatedAt      time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	CreatedBy      *int64         `gorm:"" json:"created_by,omitempty,string"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联（非数据库字段，用于预加载）
	Profile  *UserProfile `gorm:"foreignKey:UserID" json:"profile,omitempty"`
	Roles    []UserRole   `gorm:"foreignKey:UserID" json:"roles,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserProfile 用户扩展信息表
// 对应 user_profiles 表，14个字段
type UserProfile struct {
	ID             int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID         int64     `gorm:"not null;uniqueIndex" json:"user_id,string"`
	AvatarURL      *string   `gorm:"type:varchar(500)" json:"avatar_url,omitempty"`
	Nickname       *string   `gorm:"type:varchar(50)" json:"nickname,omitempty"`
	Email          *string   `gorm:"type:varchar(100)" json:"email,omitempty"`
	College        *string   `gorm:"type:varchar(100)" json:"college,omitempty"`
	Major          *string   `gorm:"type:varchar(100)" json:"major,omitempty"`
	ClassName      *string   `gorm:"type:varchar(50)" json:"class_name,omitempty"`
	EnrollmentYear *int      `gorm:"type:smallint" json:"enrollment_year,omitempty"`
	EducationLevel *int      `gorm:"type:smallint" json:"education_level,omitempty"`
	Grade          *int      `gorm:"type:smallint" json:"grade,omitempty"`
	Remark         *string   `gorm:"type:text" json:"remark,omitempty"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}

// Role 角色表
// 对应 roles 表，7个字段
type Role struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Code        string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name        string    `gorm:"type:varchar(50);not null" json:"name"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}

// UserRole 用户角色关联表
// 对应 user_roles 表，4个字段
type UserRole struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID    int64     `gorm:"not null" json:"user_id,string"`
	RoleID    int64     `gorm:"not null" json:"role_id,string"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// 关联（用于预加载角色信息）
	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}

// Permission 权限表
// 对应 permissions 表，6个字段
type Permission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Code        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Module      string    `gorm:"type:varchar(50);not null;index" json:"module"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}

// RolePermission 角色权限关联表
// 对应 role_permissions 表，4个字段
type RolePermission struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	RoleID       int64     `gorm:"not null" json:"role_id,string"`
	PermissionID int64     `gorm:"not null" json:"permission_id,string"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserSSOBinding SSO绑定记录表
// 对应 user_sso_bindings 表，7个字段
type UserSSOBinding struct {
	ID          int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID      int64      `gorm:"not null;index" json:"user_id,string"`
	SchoolID    int64      `gorm:"not null" json:"school_id,string"`
	SSOProvider string     `gorm:"type:varchar(20);not null" json:"sso_provider"`
	SSOUserID   string     `gorm:"type:varchar(100);not null" json:"sso_user_id"`
	BoundAt     time.Time  `gorm:"not null;default:now()" json:"bound_at"`
	LastLoginAt *time.Time `gorm:"" json:"last_login_at,omitempty"`
}

// TableName 指定表名
func (UserSSOBinding) TableName() string {
	return "user_sso_bindings"
}

// LoginLog 登录日志表
// 对应 login_logs 表，8个字段
// 审计日志红线：只插入，不更新，不删除
type LoginLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID      int64     `gorm:"not null;index" json:"user_id,string"`
	Action      int       `gorm:"type:smallint;not null" json:"action"`
	LoginMethod *int      `gorm:"type:smallint" json:"login_method,omitempty"`
	IP          string    `gorm:"type:varchar(45);not null" json:"ip"`
	UserAgent   *string   `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	FailReason  *string   `gorm:"type:varchar(200)" json:"fail_reason,omitempty"`
	CreatedAt   time.Time `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定表名
func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationLog 操作日志表
// 对应 operation_logs 表，8个字段
// 审计日志红线：只插入，不更新，不删除
type OperationLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	OperatorID int64     `gorm:"not null;index" json:"operator_id,string"`
	Action     string    `gorm:"type:varchar(50);not null;index" json:"action"`
	TargetType string    `gorm:"type:varchar(50);not null" json:"target_type"`
	TargetID   *int64    `gorm:"" json:"target_id,omitempty,string"`
	Detail     *string   `gorm:"type:jsonb" json:"detail,omitempty"`
	IP         string    `gorm:"type:varchar(45);not null" json:"ip"`
	CreatedAt  time.Time `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}
