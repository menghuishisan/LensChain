// notification.go
// 模块07 — 通知与消息：数据库实体结构体。
// 该文件覆盖站内信、系统公告、消息模板、通知偏好、公告阅读状态五张表映射。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Notification 站内信主表。
type Notification struct {
	ID           int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ReceiverID   int64      `gorm:"not null;index:idx_notifications_receiver_id;index:idx_notifications_receiver_read;index:idx_notifications_receiver_category" json:"receiver_id,string"`
	SchoolID     *int64     `gorm:"index" json:"school_id,omitempty,string"`
	Category     int16      `gorm:"column:category;type:smallint;not null" json:"category"`
	EventType    string     `gorm:"type:varchar(50);not null;index" json:"event_type"`
	Title        string     `gorm:"type:varchar(200);not null" json:"title"`
	Content      string     `gorm:"type:text;not null" json:"content"`
	SourceModule string     `gorm:"type:varchar(20);not null" json:"source_module"`
	SourceID     *int64     `gorm:"" json:"source_id,omitempty,string"`
	SourceType   *string    `gorm:"type:varchar(50)" json:"source_type,omitempty"`
	IsRead       bool       `gorm:"not null;default:false" json:"is_read"`
	ReadAt       *time.Time `gorm:"" json:"read_at,omitempty"`
	IsDeleted    bool       `gorm:"not null;default:false" json:"is_deleted"`
	DeletedAt    *time.Time `gorm:"" json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定站内信主表表名。
func (Notification) TableName() string {
	return "notifications"
}

// SystemAnnouncement 系统公告表。
type SystemAnnouncement struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Title         string         `gorm:"type:varchar(200);not null" json:"title"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	PublishedBy   int64          `gorm:"not null" json:"published_by,string"`
	Status        int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	IsPinned      bool           `gorm:"not null;default:true" json:"is_pinned"`
	PublishedAt   *time.Time     `gorm:"index" json:"published_at,omitempty"`
	ScheduledAt   *time.Time     `gorm:"" json:"scheduled_at,omitempty"`
	UnpublishedAt *time.Time     `gorm:"" json:"unpublished_at,omitempty"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定系统公告表表名。
func (SystemAnnouncement) TableName() string {
	return "system_announcements"
}

// NotificationTemplate 消息模板表。
type NotificationTemplate struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	EventType       string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"event_type"`
	Category        int16          `gorm:"column:category;type:smallint;not null" json:"category"`
	TitleTemplate   string         `gorm:"type:varchar(200);not null" json:"title_template"`
	ContentTemplate string         `gorm:"type:text;not null" json:"content_template"`
	Variables       datatypes.JSON `gorm:"column:variables;type:jsonb;not null" json:"variables"`
	IsEnabled       bool           `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定消息模板表表名。
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// UserNotificationPreference 用户通知偏好表。
type UserNotificationPreference struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	UserID    int64     `gorm:"not null;uniqueIndex:uk_user_notification_prefs" json:"user_id,string"`
	Category  int16     `gorm:"column:category;type:smallint;not null;uniqueIndex:uk_user_notification_prefs" json:"category"`
	IsEnabled bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定用户通知偏好表表名。
func (UserNotificationPreference) TableName() string {
	return "user_notification_preferences"
}

// AnnouncementReadStatus 公告阅读状态表。
type AnnouncementReadStatus struct {
	ID             int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	AnnouncementID int64     `gorm:"not null;uniqueIndex:uk_announcement_read" json:"announcement_id,string"`
	UserID         int64     `gorm:"not null;uniqueIndex:uk_announcement_read;index:idx_announcement_read_user" json:"user_id,string"`
	ReadAt         time.Time `gorm:"not null;default:now()" json:"read_at"`
}

// TableName 指定公告阅读状态表表名。
func (AnnouncementReadStatus) TableName() string {
	return "announcement_read_status"
}
