// school.go
// 模块02 — 学校与租户管理：数据库实体结构体
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md
// 包含 4 张表的 GORM 映射结构体

package entity

import (
	"time"

	"gorm.io/gorm"
)

// School 学校主表
// 对应 schools 表，20个字段
type School struct {
	ID             int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name           string         `gorm:"type:varchar(100);not null" json:"name"`
	Code           string         `gorm:"type:varchar(50);not null" json:"code"`
	LogoURL        *string        `gorm:"type:varchar(500)" json:"logo_url,omitempty"`
	Address        *string        `gorm:"type:varchar(200)" json:"address,omitempty"`
	Website        *string        `gorm:"type:varchar(200)" json:"website,omitempty"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Status         int            `gorm:"type:smallint;not null;default:1" json:"status"`
	LicenseStartAt *time.Time     `gorm:"" json:"license_start_at,omitempty"`
	LicenseEndAt   *time.Time     `gorm:"" json:"license_end_at,omitempty"`
	FrozenAt       *time.Time     `gorm:"" json:"frozen_at,omitempty"`
	FrozenReason   *string        `gorm:"type:varchar(200)" json:"frozen_reason,omitempty"`
	ContactName    string         `gorm:"type:varchar(50);not null" json:"contact_name"`
	ContactPhone   string         `gorm:"type:varchar(20);not null" json:"contact_phone"`
	ContactEmail   *string        `gorm:"type:varchar(100)" json:"contact_email,omitempty"`
	ContactTitle   *string        `gorm:"type:varchar(100)" json:"contact_title,omitempty"`
	CreatedAt      time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	CreatedBy      *int64         `gorm:"" json:"created_by,omitempty,string"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联（非数据库字段，用于预加载）
	SSOConfig *SchoolSSOConfig `gorm:"foreignKey:SchoolID" json:"sso_config,omitempty"`
}

// TableName 指定表名
func (School) TableName() string {
	return "schools"
}

// SchoolApplication 入驻申请记录表
// 对应 school_applications 表，17个字段
type SchoolApplication struct {
	ID                    int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolName            string     `gorm:"type:varchar(100);not null" json:"school_name"`
	SchoolCode            string     `gorm:"type:varchar(50);not null" json:"school_code"`
	SchoolAddress         *string    `gorm:"type:varchar(200)" json:"school_address,omitempty"`
	SchoolWebsite         *string    `gorm:"type:varchar(200)" json:"school_website,omitempty"`
	SchoolLogoURL         *string    `gorm:"type:varchar(500)" json:"school_logo_url,omitempty"`
	ContactName           string     `gorm:"type:varchar(50);not null" json:"contact_name"`
	ContactPhone          string     `gorm:"type:varchar(20);not null" json:"contact_phone"`
	ContactEmail          *string    `gorm:"type:varchar(100)" json:"contact_email,omitempty"`
	ContactTitle          *string    `gorm:"type:varchar(100)" json:"contact_title,omitempty"`
	Status                int        `gorm:"type:smallint;not null;default:1" json:"status"`
	ReviewerID            *int64     `gorm:"" json:"reviewer_id,omitempty,string"`
	ReviewedAt            *time.Time `gorm:"" json:"reviewed_at,omitempty"`
	RejectReason          *string    `gorm:"type:varchar(500)" json:"reject_reason,omitempty"`
	SchoolID              *int64     `gorm:"" json:"school_id,omitempty,string"`
	PreviousApplicationID *int64     `gorm:"" json:"previous_application_id,omitempty,string"`
	CreatedAt             time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (SchoolApplication) TableName() string {
	return "school_applications"
}

// SchoolSSOConfig SSO配置表
// 对应 school_sso_configs 表，10个字段
type SchoolSSOConfig struct {
	ID        int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID  int64      `gorm:"not null;uniqueIndex" json:"school_id,string"`
	Provider  string     `gorm:"type:varchar(20);not null" json:"provider"`
	IsEnabled bool       `gorm:"not null;default:false" json:"is_enabled"`
	IsTested  bool       `gorm:"not null;default:false" json:"is_tested"`
	Config    string     `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	TestedAt  *time.Time `gorm:"" json:"tested_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	UpdatedBy *int64     `gorm:"" json:"updated_by,omitempty,string"`
}

// TableName 指定表名
func (SchoolSSOConfig) TableName() string {
	return "school_sso_configs"
}

// SchoolNotification 学校通知记录表
// 对应 school_notifications 表，9个字段
type SchoolNotification struct {
	ID          int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID    int64      `gorm:"not null;index" json:"school_id,string"`
	Type        int        `gorm:"type:smallint;not null" json:"type"`
	Title       string     `gorm:"type:varchar(200);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	IsSent      bool       `gorm:"not null;default:false" json:"is_sent"`
	SentAt      *time.Time `gorm:"" json:"sent_at,omitempty"`
	TargetPhone *string    `gorm:"type:varchar(20)" json:"target_phone,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (SchoolNotification) TableName() string {
	return "school_notifications"
}
