// system.go
// 模块08 — 系统管理与监控：数据库实体结构体。
// 该文件覆盖系统配置、配置变更、告警规则、告警事件、平台统计、备份记录六张表映射。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SystemConfig 系统配置表。
type SystemConfig struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ConfigGroup string    `gorm:"type:varchar(50);not null;uniqueIndex:uk_system_configs_group_key;index" json:"config_group"`
	ConfigKey   string    `gorm:"type:varchar(100);not null;uniqueIndex:uk_system_configs_group_key" json:"config_key"`
	ConfigValue string    `gorm:"type:text;not null" json:"config_value"`
	ValueType   int16     `gorm:"column:value_type;type:smallint;not null;default:1" json:"value_type"`
	Description *string   `gorm:"type:varchar(200)" json:"description,omitempty"`
	IsSensitive bool      `gorm:"not null;default:false" json:"is_sensitive"`
	UpdatedBy   *int64    `gorm:"" json:"updated_by,omitempty,string"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定系统配置表表名。
func (SystemConfig) TableName() string {
	return "system_configs"
}

// ConfigChangeLog 配置变更记录表。
type ConfigChangeLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ConfigGroup string    `gorm:"type:varchar(50);not null;index:idx_config_change_logs_group_key" json:"config_group"`
	ConfigKey   string    `gorm:"type:varchar(100);not null;index:idx_config_change_logs_group_key" json:"config_key"`
	OldValue    *string   `gorm:"type:text" json:"old_value,omitempty"`
	NewValue    string    `gorm:"type:text;not null" json:"new_value"`
	ChangedBy   int64     `gorm:"not null;index" json:"changed_by,string"`
	ChangedAt   time.Time `gorm:"not null;default:now();index" json:"changed_at"`
	IP          string    `gorm:"type:varchar(45);not null" json:"ip"`
}

// TableName 指定配置变更记录表表名。
func (ConfigChangeLog) TableName() string {
	return "config_change_logs"
}

// AlertRule 告警规则表。
type AlertRule struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	Description   *string        `gorm:"type:varchar(500)" json:"description,omitempty"`
	AlertType     int16          `gorm:"column:alert_type;type:smallint;not null;index" json:"alert_type"`
	Level         int16          `gorm:"column:level;type:smallint;not null;default:2" json:"level"`
	Condition     datatypes.JSON `gorm:"column:condition;type:jsonb;not null" json:"condition"`
	SilencePeriod int            `gorm:"not null;default:1800" json:"silence_period"`
	IsEnabled     bool           `gorm:"not null;default:true;index" json:"is_enabled"`
	CreatedBy     int64          `gorm:"not null" json:"created_by,string"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定告警规则表表名。
func (AlertRule) TableName() string {
	return "alert_rules"
}

// AlertEvent 告警事件表。
type AlertEvent struct {
	ID          int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	RuleID      int64          `gorm:"not null;index" json:"rule_id,string"`
	Level       int16          `gorm:"column:level;type:smallint;not null;index" json:"level"`
	Title       string         `gorm:"type:varchar(200);not null" json:"title"`
	Detail      datatypes.JSON `gorm:"column:detail;type:jsonb;not null" json:"detail"`
	Status      int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	HandledBy   *int64         `gorm:"" json:"handled_by,omitempty,string"`
	HandledAt   *time.Time     `gorm:"" json:"handled_at,omitempty"`
	HandleNote  *string        `gorm:"type:text" json:"handle_note,omitempty"`
	TriggeredAt time.Time      `gorm:"not null;default:now();index" json:"triggered_at"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定告警事件表表名。
func (AlertEvent) TableName() string {
	return "alert_events"
}

// PlatformStatistic 平台统计日表。
type PlatformStatistic struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	StatDate           time.Time `gorm:"type:date;not null;uniqueIndex" json:"stat_date"`
	ActiveUsers        int       `gorm:"not null;default:0" json:"active_users"`
	NewUsers           int       `gorm:"not null;default:0" json:"new_users"`
	TotalUsers         int       `gorm:"not null;default:0" json:"total_users"`
	TotalSchools       int       `gorm:"not null;default:0" json:"total_schools"`
	TotalCourses       int       `gorm:"not null;default:0" json:"total_courses"`
	ActiveCourses      int       `gorm:"not null;default:0" json:"active_courses"`
	TotalExperiments   int       `gorm:"not null;default:0" json:"total_experiments"`
	TotalCompetitions  int       `gorm:"not null;default:0" json:"total_competitions"`
	ActiveCompetitions int       `gorm:"not null;default:0" json:"active_competitions"`
	StorageUsedGB      float64   `gorm:"type:decimal(10,2);not null;default:0" json:"storage_used_gb"`
	APIRequestCount    int64     `gorm:"not null;default:0" json:"api_request_count"`
	CreatedAt          time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定平台统计日表表名。
func (PlatformStatistic) TableName() string {
	return "platform_statistics"
}

// BackupRecord 备份记录表。
type BackupRecord struct {
	ID           int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	BackupType   int16      `gorm:"column:backup_type;type:smallint;not null" json:"backup_type"`
	Status       int16      `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	FilePath     *string    `gorm:"type:varchar(500)" json:"file_path,omitempty"`
	FileSize     *int64     `gorm:"" json:"file_size,omitempty"`
	DatabaseName string     `gorm:"type:varchar(100);not null" json:"database_name"`
	StartedAt    time.Time  `gorm:"not null;default:now();index" json:"started_at"`
	CompletedAt  *time.Time `gorm:"" json:"completed_at,omitempty"`
	ErrorMessage *string    `gorm:"type:text" json:"error_message,omitempty"`
	TriggeredBy  *int64     `gorm:"" json:"triggered_by,omitempty,string"`
	CreatedAt    time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定备份记录表表名。
func (BackupRecord) TableName() string {
	return "backup_records"
}
