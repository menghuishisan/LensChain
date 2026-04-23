// core.go
// 模块08 — 系统管理与监控：核心类型、依赖装配与公共辅助函数。
// 该文件统一定义 service 接口、跨模块依赖和通用映射逻辑，避免各功能文件重复声明。

package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cronlib "github.com/robfig/cron/v3"
	"gorm.io/datatypes"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	systemrepo "github.com/lenschain/backend/internal/repository/system"
)

const (
	systemConfigCacheTTL         = time.Hour
	systemConfigCachePrefix      = "system_config:"
	defaultBackupRetentionCount  = 30
	defaultBackupCron            = "0 0 2 * * *"
	defaultBackupObjectPrefix    = "system/backups"
	defaultRecentAlertListSize   = 5
	defaultSchoolActivityTopSize = 20
	maxAuditExportRows           = 100000
	alertWindowStatePrefix       = "alert_rule:window:"
	serviceFailCountPrefix       = "service_health:fail_count:"
)

const optimisticLockTimeLayout = time.RFC3339Nano

var systemConfigGroupKeys = map[string][]string{
	"platform": {
		"name",
		"logo_url",
		"icp_number",
		"copyright",
		"description",
	},
	"storage": {
		"default_school_quota_gb",
		"max_upload_size_mb",
	},
	"security": {
		"session_timeout_hours",
		"max_login_fail_count",
		"lock_duration_minutes",
		"password_min_length",
		"password_require_uppercase",
		"password_require_lowercase",
		"password_require_digit",
		"password_require_special_char",
	},
	"backup": {
		"auto_backup_enabled",
		"auto_backup_cron",
		"backup_retention_count",
	},
}

// ServiceFile 表示模块08导出下载类接口的文件结果。
type ServiceFile struct {
	FileName    string
	ContentType string
	Content     []byte
	RedirectURL string
}

// Service 模块08统一服务接口。
type Service interface {
	ListAuditLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AuditLogListReq) (*dto.AuditLogListResp, error)
	ExportAuditLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ExportAuditLogReq) (*ServiceFile, error)

	GetConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SystemConfigListResp, error)
	GetConfigGroup(ctx context.Context, sc *svcctx.ServiceContext, group string) (*dto.SystemConfigGroupResp, error)
	UpdateConfig(ctx context.Context, sc *svcctx.ServiceContext, group, key string, req *dto.UpdateSystemConfigReq) error
	BatchUpdateConfigs(ctx context.Context, sc *svcctx.ServiceContext, group string, req *dto.BatchUpdateSystemConfigsReq) error
	ListConfigChangeLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ConfigChangeLogListReq) (*dto.ConfigChangeLogListResp, error)

	CreateAlertRule(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateAlertRuleReq) (*dto.AlertRuleDetailResp, error)
	ListAlertRules(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AlertRuleListReq) (*dto.AlertRuleListResp, error)
	GetAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AlertRuleDetailResp, error)
	UpdateAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAlertRuleReq) error
	ToggleAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ToggleAlertRuleReq) error
	DeleteAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	ListAlertEvents(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AlertEventListReq) (*dto.AlertEventListResp, error)
	GetAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AlertEventDetailResp, error)
	HandleAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAlertEventReq) error
	IgnoreAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAlertEventReq) error

	GetDashboardHealth(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardHealthResp, error)
	GetDashboardResources(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardResourcesResp, error)
	GetDashboardRealtime(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardRealtimeResp, error)

	GetStatisticsOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.StatisticsOverviewResp, error)
	GetStatisticsTrend(ctx context.Context, sc *svcctx.ServiceContext, req *dto.StatisticsTrendReq) (*dto.StatisticsTrendResp, error)
	GetSchoolStatistics(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolActivityRankResp, error)

	TriggerBackup(ctx context.Context, sc *svcctx.ServiceContext) (*dto.TriggerBackupResp, error)
	ListBackups(ctx context.Context, sc *svcctx.ServiceContext, req *dto.BackupListReq) (*dto.BackupListResp, error)
	DownloadBackup(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*ServiceFile, error)
	UpdateBackupConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateBackupConfigReq) (*dto.BackupConfigResp, error)
	GetBackupConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.BackupConfigResp, error)
}

// NotificationEventDispatcher 提供给模块08的最小通知发送接口。
type NotificationEventDispatcher interface {
	DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error
}

// RuntimeSecurityConfigSyncer 提供给模块08的运行时安全策略同步接口。
type RuntimeSecurityConfigSyncer interface {
	SyncRuntimeSecurityConfig(ctx context.Context, config RuntimeSecurityConfig) error
}

// BackupScheduleSyncer 提供给模块08的自动备份调度同步接口。
type BackupScheduleSyncer interface {
	SyncAutoBackupTask(ctx context.Context) error
}

// RuntimeSecurityConfig 表示模块08向模块01同步的安全配置子集。
type RuntimeSecurityConfig struct {
	SessionTimeoutHours        int
	MaxLoginFailCount          int
	LockDurationMinutes        int
	PasswordMinLength          int
	PasswordRequireUppercase   bool
	PasswordRequireLowercase   bool
	PasswordRequireDigit       bool
	PasswordRequireSpecialChar bool
}

// ClusterStatusProvider 提供模块08所需的最小 K8s 集群状态能力。
type ClusterStatusProvider interface {
	GetClusterStatus(ctx context.Context) (*ClusterStatusSnapshot, error)
}

// ClusterStatusSnapshot 表示仪表盘与告警复用的 K8s 集群快照。
type ClusterStatusSnapshot struct {
	TotalNodes   int
	ReadyNodes   int
	TotalCPU     string
	UsedCPU      string
	TotalMemory  string
	UsedMemory   string
	TotalStorage string
	UsedStorage  string
	TotalPods    int
	RunningPods  int
	PendingPods  int
	FailedPods   int
	Namespaces   int
}

// service 模块08服务实现。
type service struct {
	cfg                  *config.Config
	auditRepo            systemrepo.AuditRepository
	configRepo           systemrepo.SystemConfigRepository
	configChangeLogRepo  systemrepo.ConfigChangeLogRepository
	alertRuleRepo        systemrepo.AlertRuleRepository
	alertEventRepo       systemrepo.AlertEventRepository
	statRepo             systemrepo.PlatformStatisticRepository
	backupRepo           systemrepo.BackupRecordRepository
	userRepo             authrepo.UserRepository
	notification         NotificationEventDispatcher
	securitySyncer       RuntimeSecurityConfigSyncer
	backupScheduleSyncer BackupScheduleSyncer
	clusterProvider      ClusterStatusProvider
}

// NewService 创建模块08服务实例。
func NewService(
	cfg *config.Config,
	auditRepo systemrepo.AuditRepository,
	configRepo systemrepo.SystemConfigRepository,
	configChangeLogRepo systemrepo.ConfigChangeLogRepository,
	alertRuleRepo systemrepo.AlertRuleRepository,
	alertEventRepo systemrepo.AlertEventRepository,
	statRepo systemrepo.PlatformStatisticRepository,
	backupRepo systemrepo.BackupRecordRepository,
	userRepo authrepo.UserRepository,
	notification NotificationEventDispatcher,
	securitySyncer RuntimeSecurityConfigSyncer,
	clusterProvider ClusterStatusProvider,
) Service {
	return &service{
		cfg:                 cfg,
		auditRepo:           auditRepo,
		configRepo:          configRepo,
		configChangeLogRepo: configChangeLogRepo,
		alertRuleRepo:       alertRuleRepo,
		alertEventRepo:      alertEventRepo,
		statRepo:            statRepo,
		backupRepo:          backupRepo,
		userRepo:            userRepo,
		notification:        notification,
		securitySyncer:      securitySyncer,
		clusterProvider:     clusterProvider,
	}
}

// BindBackupScheduleSyncer 绑定自动备份调度同步器，供配置更新后重载 cron 使用。
func BindBackupScheduleSyncer(svc Service, syncer BackupScheduleSyncer) {
	impl, ok := svc.(*service)
	if !ok || impl == nil {
		return
	}
	impl.backupScheduleSyncer = syncer
}

// ensureSuperAdmin 校验模块08仅允许超级管理员访问。
func ensureSuperAdmin(sc *svcctx.ServiceContext) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	return nil
}

// formatTime 将时间格式化为 RFC3339 字符串指针。
func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	text := value.UTC().Format(time.RFC3339)
	return &text
}

// int64String 将雪花 ID 转为字符串。
func int64String(value int64) string {
	return strconv.FormatInt(value, 10)
}

// optionalInt64String 将可选雪花 ID 转为字符串指针。
func optionalInt64String(value *int64) *string {
	if value == nil || *value == 0 {
		return nil
	}
	text := strconv.FormatInt(*value, 10)
	return &text
}

// parseSnowflakeID 解析雪花 ID 字符串。
func parseSnowflakeID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, errcode.ErrInvalidID
	}
	return value, nil
}

// buildPaginationResp 构建模块08通用分页结构。
func buildPaginationResp(page, pageSize int, total int64) dto.PaginationResp {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return dto.PaginationResp{
		Page:       page,
		PageSize:   pageSize,
		Total:      int(total),
		TotalPages: totalPages,
	}
}

// decodeJSONToMap 解析 JSONB 为通用对象。
func decodeJSONToMap(raw datatypes.JSON) interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]interface{}{}
	}
	return value
}

// decodeAlertRuleCondition 解析告警规则条件 JSON。
func decodeAlertRuleCondition(raw datatypes.JSON) dto.AlertRuleCondition {
	var condition dto.AlertRuleCondition
	_ = json.Unmarshal(raw, &condition)
	return condition
}

// decodeAlertEventDetail 解析告警事件详情 JSON。
func decodeAlertEventDetail(raw datatypes.JSON) dto.AlertEventDetail {
	var detail dto.AlertEventDetail
	_ = json.Unmarshal(raw, &detail)
	return detail
}

// getConfigGroupText 返回配置分组中文名。
func getConfigGroupText(group string) string {
	switch group {
	case "platform":
		return "平台基本信息"
	case "storage":
		return "存储配置"
	case "security":
		return "安全配置"
	case "backup":
		return "备份配置"
	default:
		return group
	}
}

// validateConfigValueType 校验配置值是否符合声明的类型。
func validateConfigValueType(value string, valueType int16) error {
	switch valueType {
	case enum.SystemConfigValueTypeString:
		return nil
	case enum.SystemConfigValueTypeNumber:
		if _, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err != nil {
			return errcode.ErrInvalidParams.WithMessage("配置值类型错误")
		}
		return nil
	case enum.SystemConfigValueTypeBool:
		if _, err := strconv.ParseBool(strings.TrimSpace(value)); err != nil {
			return errcode.ErrInvalidParams.WithMessage("配置值类型错误")
		}
		return nil
	case enum.SystemConfigValueTypeJSON:
		var payload interface{}
		if err := json.Unmarshal([]byte(value), &payload); err != nil {
			return errcode.ErrInvalidParams.WithMessage("配置值类型错误")
		}
		return nil
	default:
		return errcode.ErrInvalidParams.WithMessage("配置值类型错误")
	}
}

// validateBackupCronSpec 校验自动备份 cron 必须使用平台统一的 6 段秒级表达式。
func validateBackupCronSpec(value string) error {
	spec := strings.TrimSpace(value)
	if len(strings.Fields(spec)) != 6 {
		return errcode.ErrInvalidParams.WithMessage("自动备份cron表达式必须为6段秒级格式")
	}
	parser := cronlib.NewParser(cronlib.Second | cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
	if _, err := parser.Parse(spec); err != nil {
		return errcode.ErrInvalidParams.WithMessage("自动备份cron表达式格式错误")
	}
	return nil
}

// maskSensitiveValue 脱敏敏感配置值。
func maskSensitiveValue(item *entity.SystemConfig) string {
	if item == nil {
		return ""
	}
	if item.IsSensitive {
		return "******"
	}
	return item.ConfigValue
}

// describeAuditAction 返回统一审计动作中文名。
func describeAuditAction(source, action string) string {
	switch source {
	case "login":
		switch action {
		case "login_success":
			return "登录成功"
		case "login_fail":
			return "登录失败"
		case "logout":
			return "登出"
		case "kicked":
			return "踢下线"
		case "locked":
			return "账号锁定"
		}
	case "experiment":
		switch action {
		case "terminal_command":
			return "终端命令"
		case "start":
			return "启动实验"
		case "stop":
			return "停止实验"
		}
	}
	return action
}

// describeAuditSource 返回统一审计来源中文名。
func describeAuditSource(source string) string {
	switch source {
	case "login":
		return "登录日志"
	case "operation":
		return "操作日志"
	case "experiment":
		return "实验操作日志"
	default:
		return source
	}
}

// mapAuditItem 将仓储层审计项映射为响应 DTO。
func mapAuditItem(item *systemrepo.AuditLogItem) dto.AuditLogItem {
	resp := dto.AuditLogItem{
		ID:           int64String(item.ID),
		Source:       item.Source,
		SourceText:   describeAuditSource(item.Source),
		OperatorID:   optionalInt64String(int64Ptr(item.OperatorID)),
		OperatorName: item.OperatorName,
		Action:       item.Action,
		ActionText:   describeAuditAction(item.Source, item.Action),
		Detail:       decodeJSONToMap(item.Detail),
		IP:           item.IP,
		UserAgent:    item.UserAgent,
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.TargetType != nil {
		resp.Target = &dto.AuditLogTarget{
			Type: *item.TargetType,
			ID:   optionalInt64String(item.TargetID),
		}
	}
	return resp
}

// int64Ptr 返回整型指针，便于 DTO 构建复用。
func int64Ptr(value int64) *int64 {
	return &value
}

// buildBackupConfigResp 从配置表结果构建备份配置响应。
func buildBackupConfigResp(configs []*entity.SystemConfig) *dto.BackupConfigResp {
	configMap := make(map[string]string, len(configs))
	updatedAtMap := make(map[string]string, len(configs))
	for _, item := range configs {
		if item == nil {
			continue
		}
		configMap[item.ConfigKey] = item.ConfigValue
		updatedAtMap[item.ConfigKey] = item.UpdatedAt.UTC().Format(optimisticLockTimeLayout)
	}
	autoEnabled, _ := strconv.ParseBool(configMap["auto_backup_enabled"])
	retentionCount, _ := strconv.Atoi(configMap["backup_retention_count"])
	if retentionCount <= 0 {
		retentionCount = defaultBackupRetentionCount
	}
	cron := strings.TrimSpace(configMap["auto_backup_cron"])
	if cron == "" {
		cron = defaultBackupCron
	}
	return &dto.BackupConfigResp{
		AutoEnabled:             autoEnabled,
		Cron:                    cron,
		RetentionCount:          retentionCount,
		AutoEnabledUpdatedAt:    updatedAtMap["auto_backup_enabled"],
		CronUpdatedAt:           updatedAtMap["auto_backup_cron"],
		RetentionCountUpdatedAt: updatedAtMap["backup_retention_count"],
	}
}

// buildBackupConfigMap 将备份配置 DTO 反向转为配置键值。
func buildBackupConfigMap(resp *dto.BackupConfigResp) map[string]string {
	if resp == nil {
		return map[string]string{}
	}
	return map[string]string{
		"auto_backup_enabled":    strconv.FormatBool(resp.AutoEnabled),
		"auto_backup_cron":       resp.Cron,
		"backup_retention_count": strconv.Itoa(resp.RetentionCount),
	}
}

// buildDefaultBackupFileName 生成模块08备份文件名。
func buildDefaultBackupFileName(now time.Time, backupType int16) string {
	prefix := "manual"
	if backupType == enum.BackupTypeAuto {
		prefix = "auto"
	}
	return fmt.Sprintf("lenschain_%s_%s.sql", prefix, now.UTC().Format("20060102_150405"))
}
