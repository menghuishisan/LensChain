// school.go
// 模块02 学校与租户管理实体定义。
// 该文件只保留学校、申请、SSO 配置和学校通知四张表的字段映射。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// School 表示 schools 表。
// 该实体保存学校租户主数据、授权状态和联系人信息。
type School struct {
	ID             int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name           string         `gorm:"type:varchar(100);not null" json:"name"`
	Code           string         `gorm:"type:varchar(50);not null" json:"code"`
	LogoURL        *string        `gorm:"column:logo_url;type:varchar(500)" json:"logo_url,omitempty"`
	Address        *string        `gorm:"type:varchar(200)" json:"address,omitempty"`
	Website        *string        `gorm:"type:varchar(200)" json:"website,omitempty"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Status         int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	LicenseStartAt *time.Time     `gorm:"column:license_start_at" json:"license_start_at,omitempty"`
	LicenseEndAt   *time.Time     `gorm:"column:license_end_at" json:"license_end_at,omitempty"`
	FrozenAt       *time.Time     `gorm:"column:frozen_at" json:"frozen_at,omitempty"`
	FrozenReason   *string        `gorm:"column:frozen_reason;type:varchar(200)" json:"frozen_reason,omitempty"`
	ContactName    string         `gorm:"column:contact_name;type:varchar(50);not null" json:"contact_name"`
	ContactPhone   string         `gorm:"column:contact_phone;type:varchar(20);not null" json:"contact_phone"`
	ContactEmail   *string        `gorm:"column:contact_email;type:varchar(100)" json:"contact_email,omitempty"`
	ContactTitle   *string        `gorm:"column:contact_title;type:varchar(100)" json:"contact_title,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	CreatedBy      *int64         `gorm:"column:created_by" json:"created_by,omitempty,string"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 schools 表名。
func (School) TableName() string {
	return "schools"
}

// SchoolApplication 表示 school_applications 表。
// 该实体记录学校入驻申请、审核和复提交流程。
type SchoolApplication struct {
	ID                    int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolName            string     `gorm:"column:school_name;type:varchar(100);not null" json:"school_name"`
	SchoolCode            string     `gorm:"column:school_code;type:varchar(50);not null" json:"school_code"`
	SchoolAddress         *string    `gorm:"column:school_address;type:varchar(200)" json:"school_address,omitempty"`
	SchoolWebsite         *string    `gorm:"column:school_website;type:varchar(200)" json:"school_website,omitempty"`
	SchoolLogoURL         *string    `gorm:"column:school_logo_url;type:varchar(500)" json:"school_logo_url,omitempty"`
	ContactName           string     `gorm:"column:contact_name;type:varchar(50);not null" json:"contact_name"`
	ContactPhone          string     `gorm:"column:contact_phone;type:varchar(20);not null" json:"contact_phone"`
	ContactEmail          *string    `gorm:"column:contact_email;type:varchar(100)" json:"contact_email,omitempty"`
	ContactTitle          *string    `gorm:"column:contact_title;type:varchar(100)" json:"contact_title,omitempty"`
	Status                int16      `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	ReviewerID            *int64     `gorm:"column:reviewer_id" json:"reviewer_id,omitempty,string"`
	ReviewedAt            *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	RejectReason          *string    `gorm:"column:reject_reason;type:varchar(500)" json:"reject_reason,omitempty"`
	SchoolID              *int64     `gorm:"column:school_id" json:"school_id,omitempty,string"`
	PreviousApplicationID *int64     `gorm:"column:previous_application_id" json:"previous_application_id,omitempty,string"`
	CreatedAt             time.Time  `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

// TableName 返回 school_applications 表名。
func (SchoolApplication) TableName() string {
	return "school_applications"
}

// SchoolSSOConfig 表示 school_sso_configs 表。
// 该实体保存学校级 SSO 参数和测试状态。
type SchoolSSOConfig struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID  int64          `gorm:"column:school_id;not null;uniqueIndex" json:"school_id,string"`
	Provider  string         `gorm:"type:varchar(20);not null" json:"provider"`
	IsEnabled bool           `gorm:"column:is_enabled;not null;default:false" json:"is_enabled"`
	IsTested  bool           `gorm:"column:is_tested;not null;default:false" json:"is_tested"`
	Config    datatypes.JSON `gorm:"column:config;type:jsonb;not null" json:"config"`
	TestedAt  *time.Time     `gorm:"column:tested_at" json:"tested_at,omitempty"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	UpdatedBy *int64         `gorm:"column:updated_by" json:"updated_by,omitempty,string"`
}

// TableName 返回 school_sso_configs 表名。
func (SchoolSSOConfig) TableName() string {
	return "school_sso_configs"
}

// SchoolNotification 表示 school_notifications 表。
// 该实体只记录已有学校租户的生命周期通知发送流水，如到期提醒、缓冲期、冻结和审核通过。
type SchoolNotification struct {
	ID          int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID    int64      `gorm:"column:school_id;not null;index" json:"school_id,string"`
	Type        int16      `gorm:"column:type;type:smallint;not null" json:"type"`
	Title       string     `gorm:"type:varchar(200);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	IsSent      bool       `gorm:"column:is_sent;not null;default:false" json:"is_sent"`
	SentAt      *time.Time `gorm:"column:sent_at" json:"sent_at,omitempty"`
	TargetPhone *string    `gorm:"column:target_phone;type:varchar(20)" json:"target_phone,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

// TableName 返回 school_notifications 表名。
func (SchoolNotification) TableName() string {
	return "school_notifications"
}
