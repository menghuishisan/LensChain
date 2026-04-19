// user.go
// 模块01 用户与认证实体定义。
// 该文件只保留模块01数据库表的字段映射，不承载任何预加载关系或业务方法。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User 表示 users 表。
// 该实体保存用户登录凭证、基础身份和租户归属信息。
type User struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Phone           string         `gorm:"type:varchar(20);not null" json:"phone"`
	PasswordHash    string         `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Name            string         `gorm:"type:varchar(50);not null" json:"name"`
	SchoolID        int64          `gorm:"column:school_id;not null;index" json:"school_id,string"`
	StudentNo       *string        `gorm:"column:student_no;type:varchar(50)" json:"student_no,omitempty"`
	Status          int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	IsFirstLogin    bool           `gorm:"column:is_first_login;not null;default:true" json:"is_first_login"`
	IsSchoolAdmin   bool           `gorm:"column:is_school_admin;not null;default:false" json:"is_school_admin"`
	LoginFailCount  int16          `gorm:"column:login_fail_count;type:smallint;not null;default:0" json:"login_fail_count"`
	LockedUntil     *time.Time     `gorm:"column:locked_until" json:"locked_until,omitempty"`
	TokenValidAfter time.Time      `gorm:"column:token_valid_after;not null;default:now()" json:"token_valid_after"`
	LastLoginAt     *time.Time     `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	LastLoginIP     *string        `gorm:"column:last_login_ip;type:varchar(45)" json:"last_login_ip,omitempty"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	CreatedBy       *int64         `gorm:"column:created_by" json:"created_by,omitempty,string"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 users 表名。
func (User) TableName() string {
	return "users"
}

// UserProfile 表示 user_profiles 表。
// 该实体保存头像、院系、班级和备注等扩展资料。
type UserProfile struct {
	ID             int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID         int64     `gorm:"column:user_id;not null;uniqueIndex" json:"user_id,string"`
	AvatarURL      *string   `gorm:"column:avatar_url;type:varchar(500)" json:"avatar_url,omitempty"`
	Nickname       *string   `gorm:"type:varchar(50)" json:"nickname,omitempty"`
	Email          *string   `gorm:"type:varchar(100)" json:"email,omitempty"`
	College        *string   `gorm:"type:varchar(100)" json:"college,omitempty"`
	Major          *string   `gorm:"type:varchar(100)" json:"major,omitempty"`
	ClassName      *string   `gorm:"column:class_name;type:varchar(50)" json:"class_name,omitempty"`
	EnrollmentYear *int16    `gorm:"column:enrollment_year;type:smallint" json:"enrollment_year,omitempty"`
	EducationLevel *int16    `gorm:"column:education_level;type:smallint" json:"education_level,omitempty"`
	Grade          *int16    `gorm:"column:grade;type:smallint" json:"grade,omitempty"`
	Remark         *string   `gorm:"type:text" json:"remark,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
}

// TableName 返回 user_profiles 表名。
func (UserProfile) TableName() string {
	return "user_profiles"
}

// Role 表示 roles 表。
// 该实体定义平台角色和学校可分配角色。
type Role struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Code        string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name        string    `gorm:"type:varchar(50);not null" json:"name"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	IsSystem    bool      `gorm:"column:is_system;not null;default:false" json:"is_system"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
}

// TableName 返回 roles 表名。
func (Role) TableName() string {
	return "roles"
}

// Permission 表示 permissions 表。
// 该实体定义后端接口和操作动作的授权点。
type Permission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Code        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Module      string    `gorm:"type:varchar(50);not null;index" json:"module"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

// TableName 返回 permissions 表名。
func (Permission) TableName() string {
	return "permissions"
}

// UserRole 表示 user_roles 表。
// 该实体建立用户与角色的多对多授权关系。
type UserRole struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID    int64     `gorm:"column:user_id;not null" json:"user_id,string"`
	RoleID    int64     `gorm:"column:role_id;not null" json:"role_id,string"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

// TableName 返回 user_roles 表名。
func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 表示 role_permissions 表。
// 该实体建立角色与权限点的多对多关系。
type RolePermission struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	RoleID       int64     `gorm:"column:role_id;not null" json:"role_id,string"`
	PermissionID int64     `gorm:"column:permission_id;not null" json:"permission_id,string"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

// TableName 返回 role_permissions 表名。
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserSSOBinding 表示 user_sso_bindings 表。
// 该实体记录平台账号与学校身份提供方账号的绑定关系。
type UserSSOBinding struct {
	ID          int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID      int64      `gorm:"column:user_id;not null;index" json:"user_id,string"`
	SchoolID    int64      `gorm:"column:school_id;not null" json:"school_id,string"`
	SSOProvider string     `gorm:"column:sso_provider;type:varchar(20);not null" json:"sso_provider"`
	SSOUserID   string     `gorm:"column:sso_user_id;type:varchar(100);not null" json:"sso_user_id"`
	BoundAt     time.Time  `gorm:"column:bound_at;not null;default:now()" json:"bound_at"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
}

// TableName 返回 user_sso_bindings 表名。
func (UserSSOBinding) TableName() string {
	return "user_sso_bindings"
}

// LoginLog 表示 login_logs 表。
// 该实体记录登录成功、失败、登出和锁定等认证事件。
type LoginLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID      int64     `gorm:"column:user_id;not null;index" json:"user_id,string"`
	Action      int16     `gorm:"column:action;type:smallint;not null" json:"action"`
	LoginMethod *int16    `gorm:"column:login_method;type:smallint" json:"login_method,omitempty"`
	IP          string    `gorm:"type:varchar(45);not null" json:"ip"`
	UserAgent   *string   `gorm:"column:user_agent;type:varchar(500)" json:"user_agent,omitempty"`
	FailReason  *string   `gorm:"column:fail_reason;type:varchar(200)" json:"fail_reason,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:now();index" json:"created_at"`
}

// TableName 返回 login_logs 表名。
func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationLog 表示 operation_logs 表。
// 该实体记录用户管理相关关键操作和批量导入结果。
type OperationLog struct {
	ID         int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	OperatorID int64          `gorm:"column:operator_id;not null;index" json:"operator_id,string"`
	Action     string         `gorm:"type:varchar(50);not null;index" json:"action"`
	TargetType string         `gorm:"column:target_type;type:varchar(50);not null" json:"target_type"`
	TargetID   *int64         `gorm:"column:target_id" json:"target_id,omitempty,string"`
	Detail     datatypes.JSON `gorm:"column:detail;type:jsonb" json:"detail,omitempty"`
	IP         string         `gorm:"type:varchar(45);not null" json:"ip"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;default:now();index" json:"created_at"`
}

// TableName 返回 operation_logs 表名。
func (OperationLog) TableName() string {
	return "operation_logs"
}
