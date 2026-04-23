// system.go
// 模块08 — 系统管理与监控：请求/响应 DTO 定义。
// 该文件对齐 docs/modules/08-系统管理与监控/03-API接口设计.md，覆盖审计、配置、告警、仪表盘、统计、备份接口。

package dto

// AuditLogListReq 聚合审计日志查询参数。
type AuditLogListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Source     string `form:"source" binding:"omitempty,oneof=login operation experiment"`
	Keyword    string `form:"keyword"`
	OperatorID string `form:"operator_id"`
	Action     string `form:"action"`
	DateFrom   string `form:"date_from"`
	DateTo     string `form:"date_to"`
	IP         string `form:"ip"`
}

// ExportAuditLogReq 导出审计日志查询参数。
type ExportAuditLogReq struct {
	AuditLogListReq
	Format string `form:"format" binding:"omitempty,oneof=excel csv"`
}

// AuditLogItem 审计日志列表项。
type AuditLogItem struct {
	ID           string          `json:"id"`
	Source       string          `json:"source"`
	SourceText   string          `json:"source_text"`
	OperatorID   *string         `json:"operator_id"`
	OperatorName *string         `json:"operator_name"`
	Action       string          `json:"action"`
	ActionText   string          `json:"action_text"`
	Target       *AuditLogTarget `json:"target"`
	Detail       interface{}     `json:"detail"`
	IP           *string         `json:"ip"`
	UserAgent    *string         `json:"user_agent"`
	CreatedAt    string          `json:"created_at"`
}

// AuditLogTarget 审计日志目标对象。
// 该结构对应统一审计响应中 target 的固定字段。
type AuditLogTarget struct {
	Type string  `json:"type"`
	ID   *string `json:"id"`
}

// AuditLogSourceCounts 审计来源统计。
// 该结构对应 source_counts 的固定来源键。
type AuditLogSourceCounts struct {
	Login      int `json:"login"`
	Operation  int `json:"operation"`
	Experiment int `json:"experiment"`
}

// AuditLogListResp 审计日志列表响应。
type AuditLogListResp struct {
	List         []AuditLogItem       `json:"list"`
	Pagination   PaginationResp       `json:"pagination"`
	SourceCounts AuditLogSourceCounts `json:"source_counts"`
}

// SystemConfigGroupItem 系统配置分组项。
type SystemConfigGroupItem struct {
	Group     string             `json:"group"`
	GroupText string             `json:"group_text"`
	Configs   []SystemConfigItem `json:"configs"`
}

// SystemConfigItem 系统配置项。
type SystemConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   int16  `json:"value_type"`
	Description string `json:"description"`
	IsSensitive bool   `json:"is_sensitive"`
	UpdatedAt   string `json:"updated_at"`
}

// SystemConfigListResp 系统配置列表响应。
type SystemConfigListResp struct {
	Groups []SystemConfigGroupItem `json:"groups"`
}

// SystemConfigGroupResp 单个配置分组响应。
type SystemConfigGroupResp struct {
	Group SystemConfigGroupItem `json:"group"`
}

// UpdateSystemConfigReq 更新单个配置请求。
type UpdateSystemConfigReq struct {
	Value     string `json:"value" binding:"required"`
	UpdatedAt string `json:"updated_at" binding:"required"`
}

// BatchUpdateSystemConfigsReq 批量更新分组配置请求。
type BatchUpdateSystemConfigsReq struct {
	Configs []BatchUpdateSystemConfigItem `json:"configs" binding:"required,min=1,dive"`
}

// BatchUpdateSystemConfigItem 批量更新配置项。
type BatchUpdateSystemConfigItem struct {
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value" binding:"required"`
	UpdatedAt string `json:"updated_at" binding:"required"`
}

// ConfigChangeLogListReq 配置变更记录查询参数。
type ConfigChangeLogListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	ConfigGroup string `form:"config_group"`
	ConfigKey   string `form:"config_key"`
	DateFrom    string `form:"date_from"`
	DateTo      string `form:"date_to"`
}

// ConfigChangeLogItem 配置变更记录列表项。
type ConfigChangeLogItem struct {
	ID            string  `json:"id"`
	ConfigGroup   string  `json:"config_group"`
	ConfigKey     string  `json:"config_key"`
	OldValue      *string `json:"old_value"`
	NewValue      string  `json:"new_value"`
	ChangedBy     string  `json:"changed_by"`
	ChangedByName string  `json:"changed_by_name"`
	ChangedAt     string  `json:"changed_at"`
	IP            string  `json:"ip"`
}

// ConfigChangeLogListResp 配置变更记录列表响应。
type ConfigChangeLogListResp struct {
	List       []ConfigChangeLogItem `json:"list"`
	Pagination PaginationResp        `json:"pagination"`
}

// CreateAlertRuleReq 创建告警规则请求。
type CreateAlertRuleReq struct {
	Name          string             `json:"name" binding:"required,max=100"`
	Description   *string            `json:"description" binding:"omitempty,max=500"`
	AlertType     int16              `json:"alert_type" binding:"required,oneof=1 2 3"`
	Level         int16              `json:"level" binding:"required,oneof=1 2 3 4"`
	Condition     AlertRuleCondition `json:"condition" binding:"required"`
	SilencePeriod int                `json:"silence_period" binding:"omitempty,min=0"`
}

// UpdateAlertRuleReq 更新告警规则请求。
type UpdateAlertRuleReq struct {
	Name          *string             `json:"name" binding:"omitempty,max=100"`
	Description   *string             `json:"description" binding:"omitempty,max=500"`
	AlertType     *int16              `json:"alert_type" binding:"omitempty,oneof=1 2 3"`
	Level         *int16              `json:"level" binding:"omitempty,oneof=1 2 3 4"`
	Condition     *AlertRuleCondition `json:"condition"`
	SilencePeriod *int                `json:"silence_period" binding:"omitempty,min=0"`
}

// AlertRuleCondition 告警规则条件。
// 该结构统一承载数据库设计中 3 类条件 JSON 的固定字段，未使用字段保持为空。
type AlertRuleCondition struct {
	Metric         *string                `json:"metric,omitempty"`
	Operator       *string                `json:"operator,omitempty"`
	Value          *float64               `json:"value,omitempty"`
	Duration       *int                   `json:"duration,omitempty"`
	EventSource    *string                `json:"event_source,omitempty"`
	EventFilter    map[string]interface{} `json:"event_filter,omitempty"`
	GroupBy        *string                `json:"group_by,omitempty"`
	CountThreshold *int                   `json:"count_threshold,omitempty"`
	TimeWindow     *int                   `json:"time_window,omitempty"`
	ServiceName    *string                `json:"service_name,omitempty"`
	CheckURL       *string                `json:"check_url,omitempty"`
	CheckInterval  *int                   `json:"check_interval,omitempty"`
	FailThreshold  *int                   `json:"fail_threshold,omitempty"`
}

// ToggleAlertRuleReq 启用/禁用告警规则请求。
type ToggleAlertRuleReq struct {
	IsEnabled bool `json:"is_enabled"`
}

// AlertRuleListReq 告警规则列表查询参数。
type AlertRuleListReq struct {
	Page      int   `form:"page" binding:"omitempty,min=1"`
	PageSize  int   `form:"page_size" binding:"omitempty,min=1,max=100"`
	AlertType int16 `form:"alert_type" binding:"omitempty,oneof=1 2 3"`
	Level     int16 `form:"level" binding:"omitempty,oneof=1 2 3 4"`
	IsEnabled *bool `form:"is_enabled"`
}

// AlertRuleItem 告警规则列表项。
type AlertRuleItem struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   *string            `json:"description"`
	AlertType     int16              `json:"alert_type"`
	AlertTypeText string             `json:"alert_type_text"`
	Level         int16              `json:"level"`
	LevelText     string             `json:"level_text"`
	Condition     AlertRuleCondition `json:"condition"`
	SilencePeriod int                `json:"silence_period"`
	IsEnabled     bool               `json:"is_enabled"`
	CreatedAt     *string            `json:"created_at,omitempty"`
}

// AlertRuleDetailResp 告警规则详情响应。
type AlertRuleDetailResp struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   *string            `json:"description"`
	AlertType     int16              `json:"alert_type"`
	AlertTypeText string             `json:"alert_type_text"`
	Level         int16              `json:"level"`
	LevelText     string             `json:"level_text"`
	Condition     AlertRuleCondition `json:"condition"`
	SilencePeriod int                `json:"silence_period"`
	IsEnabled     bool               `json:"is_enabled"`
	CreatedAt     *string            `json:"created_at,omitempty"`
}

// AlertRuleListResp 告警规则列表响应。
type AlertRuleListResp struct {
	List       []AlertRuleItem `json:"list"`
	Pagination PaginationResp  `json:"pagination"`
}

// AlertEventListReq 告警事件列表查询参数。
type AlertEventListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	RuleID   string `form:"rule_id"`
	Level    int16  `form:"level" binding:"omitempty,oneof=1 2 3 4"`
	Status   int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
}

// AlertEventItem 告警事件列表项。
type AlertEventItem struct {
	ID          string           `json:"id"`
	RuleID      string           `json:"rule_id"`
	RuleName    string           `json:"rule_name"`
	Level       int16            `json:"level"`
	LevelText   string           `json:"level_text"`
	Title       string           `json:"title"`
	Detail      AlertEventDetail `json:"detail"`
	Status      int16            `json:"status"`
	StatusText  string           `json:"status_text"`
	TriggeredAt string           `json:"triggered_at"`
}

// AlertEventDetail 告警事件详情。
// 该结构统一承载数据库设计中的阈值告警和事件告警明细字段。
type AlertEventDetail struct {
	Metric          *string                 `json:"metric,omitempty"`
	CurrentValue    *float64                `json:"current_value,omitempty"`
	Threshold       *float64                `json:"threshold,omitempty"`
	DurationSeconds *int                    `json:"duration_seconds,omitempty"`
	Node            *string                 `json:"node,omitempty"`
	EventSource     *string                 `json:"event_source,omitempty"`
	GroupValue      *string                 `json:"group_value,omitempty"`
	EventCount      *int                    `json:"event_count,omitempty"`
	TimeWindow      *int                    `json:"time_window,omitempty"`
	SampleEvents    []AlertEventSampleEvent `json:"sample_events,omitempty"`
}

// AlertEventSampleEvent 事件告警样本日志项。
type AlertEventSampleEvent struct {
	UserID     *string `json:"user_id,omitempty"`
	FailReason *string `json:"fail_reason,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// AlertEventStatusCounts 告警状态统计。
type AlertEventStatusCounts struct {
	Pending int `json:"pending"`
	Handled int `json:"handled"`
	Ignored int `json:"ignored"`
}

// AlertEventListResp 告警事件列表响应。
type AlertEventListResp struct {
	List         []AlertEventItem       `json:"list"`
	Pagination   PaginationResp         `json:"pagination"`
	StatusCounts AlertEventStatusCounts `json:"status_counts"`
}

// AlertEventDetailResp 告警事件详情响应。
type AlertEventDetailResp struct {
	ID            string           `json:"id"`
	RuleID        string           `json:"rule_id"`
	RuleName      string           `json:"rule_name"`
	Level         int16            `json:"level"`
	LevelText     string           `json:"level_text"`
	Title         string           `json:"title"`
	Detail        AlertEventDetail `json:"detail"`
	Status        int16            `json:"status"`
	StatusText    string           `json:"status_text"`
	HandledBy     *string          `json:"handled_by,omitempty"`
	HandledByName *string          `json:"handled_by_name,omitempty"`
	HandledAt     *string          `json:"handled_at,omitempty"`
	HandleNote    *string          `json:"handle_note,omitempty"`
	TriggeredAt   string           `json:"triggered_at"`
}

// HandleAlertEventReq 处理/忽略告警请求。
type HandleAlertEventReq struct {
	HandleNote string `json:"handle_note" binding:"required"`
}

// DashboardHealthResp 平台健康状态响应。
type DashboardHealthResp struct {
	OverallStatus string                   `json:"overall_status"`
	Services      []DashboardServiceHealth `json:"services"`
	LastCheckAt   string                   `json:"last_check_at"`
}

// DashboardServiceConnections 数据库连接信息。
type DashboardServiceConnections struct {
	Active int `json:"active"`
	Max    int `json:"max"`
}

// DashboardServiceHealth 服务健康项。
// 该结构覆盖健康面板里各类服务的固定字段，未适用字段保持为空。
type DashboardServiceHealth struct {
	Name            string                       `json:"name"`
	Status          string                       `json:"status"`
	LatencyMS       int                          `json:"latency_ms"`
	Uptime          *string                      `json:"uptime,omitempty"`
	Connections     *DashboardServiceConnections `json:"connections,omitempty"`
	MemoryUsedMB    *int                         `json:"memory_used_mb,omitempty"`
	MessagesInQueue *int                         `json:"messages_in_queue,omitempty"`
	StorageUsedGB   *float64                     `json:"storage_used_gb,omitempty"`
	Nodes           *int                         `json:"nodes,omitempty"`
	PodsRunning     *int                         `json:"pods_running,omitempty"`
}

// DashboardResourcesResp 资源使用情况响应。
type DashboardResourcesResp struct {
	CPU     DashboardCPUResource     `json:"cpu"`
	Memory  DashboardMemoryResource  `json:"memory"`
	Storage DashboardStorageResource `json:"storage"`
	K8s     DashboardK8sResource     `json:"k8s"`
}

// DashboardCPUResource CPU 资源使用项。
type DashboardCPUResource struct {
	UsagePercent float64 `json:"usage_percent"`
	CoresTotal   int     `json:"cores_total"`
	CoresUsed    float64 `json:"cores_used"`
}

// DashboardMemoryResource 内存资源使用项。
type DashboardMemoryResource struct {
	UsagePercent float64 `json:"usage_percent"`
	TotalGB      int     `json:"total_gb"`
	UsedGB       int     `json:"used_gb"`
}

// DashboardStorageResource 存储资源使用项。
type DashboardStorageResource struct {
	UsagePercent float64 `json:"usage_percent"`
	TotalGB      int     `json:"total_gb"`
	UsedGB       int     `json:"used_gb"`
}

// DashboardK8sResource K8s 资源使用项。
type DashboardK8sResource struct {
	Nodes       int `json:"nodes"`
	PodsTotal   int `json:"pods_total"`
	PodsRunning int `json:"pods_running"`
	PodsPending int `json:"pods_pending"`
	Namespaces  int `json:"namespaces"`
}

// DashboardRealtimeResp 实时指标响应。
type DashboardRealtimeResp struct {
	OnlineUsers          int                    `json:"online_users"`
	ActiveExperiments    int                    `json:"active_experiments"`
	ActiveCompetitions   int                    `json:"active_competitions"`
	APIRequestsPerMinute int                    `json:"api_requests_per_minute"`
	PendingAlerts        int                    `json:"pending_alerts"`
	RecentAlerts         []DashboardRecentAlert `json:"recent_alerts"`
}

// DashboardRecentAlert 仪表盘最近告警项。
type DashboardRecentAlert struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Level       int16  `json:"level"`
	TriggeredAt string `json:"triggered_at"`
}

// StatisticsTrendReq 趋势数据查询参数。
type StatisticsTrendReq struct {
	Metric string `form:"metric" binding:"required,oneof=active_users new_users experiments api_requests"`
	Period string `form:"period" binding:"omitempty,oneof=7d 30d 90d 365d"`
}

// StatisticsOverviewResp 统计总览响应。
type StatisticsOverviewResp struct {
	TotalUsers        int                    `json:"total_users"`
	TotalSchools      int                    `json:"total_schools"`
	TotalCourses      int                    `json:"total_courses"`
	TotalExperiments  int                    `json:"total_experiments"`
	TotalCompetitions int                    `json:"total_competitions"`
	Today             StatisticsTodaySummary `json:"today"`
}

// StatisticsTodaySummary 今日统计汇总。
type StatisticsTodaySummary struct {
	ActiveUsers        int `json:"active_users"`
	NewUsers           int `json:"new_users"`
	ExperimentsStarted int `json:"experiments_started"`
	APIRequests        int `json:"api_requests"`
}

// StatisticsTrendResp 趋势数据响应。
type StatisticsTrendResp struct {
	Metric     string                 `json:"metric"`
	Period     string                 `json:"period"`
	DataPoints []StatisticsTrendPoint `json:"data_points"`
}

// StatisticsTrendPoint 趋势点位。
type StatisticsTrendPoint struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// SchoolActivityRankResp 学校活跃度排行响应。
type SchoolActivityRankResp struct {
	List []SchoolActivityRankItem `json:"list"`
}

// SchoolActivityRankItem 学校活跃度排行项。
type SchoolActivityRankItem struct {
	Rank          int     `json:"rank"`
	SchoolID      string  `json:"school_id"`
	SchoolName    string  `json:"school_name"`
	ActiveUsers   int     `json:"active_users"`
	TotalUsers    int     `json:"total_users"`
	ActivityScore float64 `json:"activity_score"`
}

// TriggerBackupResp 手动触发备份响应。
type TriggerBackupResp struct {
	ID         string `json:"id"`
	BackupType int16  `json:"backup_type"`
	Status     int16  `json:"status"`
	StatusText string `json:"status_text"`
	StartedAt  string `json:"started_at"`
}

// BackupListReq 备份列表查询参数。
type BackupListReq struct {
	Page     int   `form:"page" binding:"omitempty,min=1"`
	PageSize int   `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   int16 `form:"status" binding:"omitempty,oneof=1 2 3"`
}

// BackupListItem 备份列表项。
type BackupListItem struct {
	ID              string  `json:"id"`
	BackupType      int16   `json:"backup_type"`
	BackupTypeText  string  `json:"backup_type_text"`
	Status          int16   `json:"status"`
	StatusText      string  `json:"status_text"`
	DatabaseName    string  `json:"database_name"`
	FileSize        *int64  `json:"file_size"`
	FileSizeText    *string `json:"file_size_text"`
	StartedAt       string  `json:"started_at"`
	CompletedAt     *string `json:"completed_at"`
	DurationSeconds *int    `json:"duration_seconds"`
	ErrorMessage    *string `json:"error_message,omitempty"`
}

// BackupConfigResp 备份配置响应。
type BackupConfigResp struct {
	AutoEnabled             bool   `json:"auto_enabled"`
	Cron                    string `json:"cron"`
	RetentionCount          int    `json:"retention_count"`
	AutoEnabledUpdatedAt    string `json:"auto_enabled_updated_at"`
	CronUpdatedAt           string `json:"cron_updated_at"`
	RetentionCountUpdatedAt string `json:"retention_count_updated_at"`
}

// BackupListResp 备份列表响应。
type BackupListResp struct {
	List         []BackupListItem `json:"list"`
	Pagination   PaginationResp   `json:"pagination"`
	BackupConfig BackupConfigResp `json:"backup_config"`
}

// UpdateBackupConfigReq 更新备份配置请求。
type UpdateBackupConfigReq struct {
	AutoEnabled             bool   `json:"auto_enabled"`
	Cron                    string `json:"cron" binding:"required"`
	RetentionCount          int    `json:"retention_count" binding:"required,min=1"`
	AutoEnabledUpdatedAt    string `json:"auto_enabled_updated_at" binding:"required"`
	CronUpdatedAt           string `json:"cron_updated_at" binding:"required"`
	RetentionCountUpdatedAt string `json:"retention_count_updated_at" binding:"required"`
}
