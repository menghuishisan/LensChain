// system.go
// 模块08 — 系统管理与监控：统一维护配置、告警、备份等领域枚举。
// 聚合层所有状态值都集中定义，便于后续监控与运维接口共享。

package enum

const (
	SystemConfigValueTypeString = 1 // 字符串
	SystemConfigValueTypeNumber = 2 // 数字
	SystemConfigValueTypeBool   = 3 // 布尔
	SystemConfigValueTypeJSON   = 4 // JSON
)

// SystemConfigValueTypeText 配置值类型文本映射。
var SystemConfigValueTypeText = map[int16]string{
	SystemConfigValueTypeString: "字符串",
	SystemConfigValueTypeNumber: "数字",
	SystemConfigValueTypeBool:   "布尔",
	SystemConfigValueTypeJSON:   "JSON",
}

// GetSystemConfigValueTypeText 获取配置值类型文本。
func GetSystemConfigValueTypeText(value int16) string {
	if text, ok := SystemConfigValueTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidSystemConfigValueType 校验配置值类型是否合法。
func IsValidSystemConfigValueType(value int16) bool {
	_, ok := SystemConfigValueTypeText[value]
	return ok
}

const (
	AlertTypeThreshold = 1 // 阈值告警
	AlertTypeEvent     = 2 // 事件告警
	AlertTypeService   = 3 // 服务状态告警
)

// AlertTypeText 告警类型文本映射。
var AlertTypeText = map[int16]string{
	AlertTypeThreshold: "阈值告警",
	AlertTypeEvent:     "事件告警",
	AlertTypeService:   "服务状态告警",
}

// GetAlertTypeText 获取告警类型文本。
func GetAlertTypeText(value int16) string {
	if text, ok := AlertTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAlertType 校验告警类型是否合法。
func IsValidAlertType(value int16) bool {
	_, ok := AlertTypeText[value]
	return ok
}

const (
	AlertLevelInfo     = 1 // 信息
	AlertLevelWarning  = 2 // 警告
	AlertLevelCritical = 3 // 严重
	AlertLevelUrgent   = 4 // 紧急
)

// AlertLevelText 告警级别文本映射。
var AlertLevelText = map[int16]string{
	AlertLevelInfo:     "信息",
	AlertLevelWarning:  "警告",
	AlertLevelCritical: "严重",
	AlertLevelUrgent:   "紧急",
}

// GetAlertLevelText 获取告警级别文本。
func GetAlertLevelText(value int16) string {
	if text, ok := AlertLevelText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAlertLevel 校验告警级别是否合法。
func IsValidAlertLevel(value int16) bool {
	_, ok := AlertLevelText[value]
	return ok
}

const (
	AlertEventStatusPending = 1 // 待处理
	AlertEventStatusHandled = 2 // 已处理
	AlertEventStatusIgnored = 3 // 已忽略
)

// AlertEventStatusText 告警事件状态文本映射。
var AlertEventStatusText = map[int16]string{
	AlertEventStatusPending: "待处理",
	AlertEventStatusHandled: "已处理",
	AlertEventStatusIgnored: "已忽略",
}

// GetAlertEventStatusText 获取告警事件状态文本。
func GetAlertEventStatusText(value int16) string {
	if text, ok := AlertEventStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAlertEventStatus 校验告警事件状态是否合法。
func IsValidAlertEventStatus(value int16) bool {
	_, ok := AlertEventStatusText[value]
	return ok
}

const (
	BackupTypeAuto   = 1 // 自动备份
	BackupTypeManual = 2 // 手动备份
)

// BackupTypeText 备份类型文本映射。
var BackupTypeText = map[int16]string{
	BackupTypeAuto:   "自动备份",
	BackupTypeManual: "手动备份",
}

// GetBackupTypeText 获取备份类型文本。
func GetBackupTypeText(value int16) string {
	if text, ok := BackupTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidBackupType 校验备份类型是否合法。
func IsValidBackupType(value int16) bool {
	_, ok := BackupTypeText[value]
	return ok
}

const (
	BackupStatusRunning = 1 // 进行中
	BackupStatusSuccess = 2 // 成功
	BackupStatusFailed  = 3 // 失败
)

// BackupStatusText 备份状态文本映射。
var BackupStatusText = map[int16]string{
	BackupStatusRunning: "进行中",
	BackupStatusSuccess: "成功",
	BackupStatusFailed:  "失败",
}

// GetBackupStatusText 获取备份状态文本。
func GetBackupStatusText(value int16) string {
	if text, ok := BackupStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidBackupStatus 校验备份状态是否合法。
func IsValidBackupStatus(value int16) bool {
	_, ok := BackupStatusText[value]
	return ok
}
