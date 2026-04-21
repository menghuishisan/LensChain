// school.go
// 模块02 — 学校与租户管理：枚举常量定义
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md
// 包含学校状态、申请状态、通知类型、SSO协议类型等枚举

package enum

// ========== 学校状态（schools.status） ==========

const (
	SchoolStatusPending   = 1 // 待审核
	SchoolStatusActive    = 2 // 已激活
	SchoolStatusBuffering = 3 // 缓冲期
	SchoolStatusFrozen    = 4 // 已冻结
	SchoolStatusCancelled = 5 // 已注销
	SchoolStatusRejected  = 6 // 已拒绝
)

// SchoolStatusText 学校状态文本映射
var SchoolStatusText = map[int16]string{
	SchoolStatusPending:   "待审核",
	SchoolStatusActive:    "已激活",
	SchoolStatusBuffering: "缓冲期",
	SchoolStatusFrozen:    "已冻结",
	SchoolStatusCancelled: "已注销",
	SchoolStatusRejected:  "已拒绝",
}

// GetSchoolStatusText 获取学校状态文本
func GetSchoolStatusText(status int16) string {
	if text, ok := SchoolStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidSchoolStatus 校验学校状态是否合法
func IsValidSchoolStatus(status int16) bool {
	_, ok := SchoolStatusText[status]
	return ok
}

// ========== 入驻申请状态（school_applications.status） ==========

const (
	ApplicationStatusPending  = 1 // 待审核
	ApplicationStatusApproved = 2 // 已通过
	ApplicationStatusRejected = 3 // 已拒绝
)

// ApplicationStatusText 申请状态文本映射
var ApplicationStatusText = map[int16]string{
	ApplicationStatusPending:  "待审核",
	ApplicationStatusApproved: "已通过",
	ApplicationStatusRejected: "已拒绝",
}

// GetApplicationStatusText 获取申请状态文本
func GetApplicationStatusText(status int16) string {
	if text, ok := ApplicationStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidApplicationStatus 校验入驻申请状态是否合法
func IsValidApplicationStatus(status int16) bool {
	_, ok := ApplicationStatusText[status]
	return ok
}

// ========== 学校通知类型（school_notifications.type） ==========

const (
	SchoolNotifyExpiring  = 1 // 到期提醒
	SchoolNotifyBuffering = 2 // 缓冲期通知
	SchoolNotifyFrozen    = 3 // 冻结通知
	SchoolNotifyApproved  = 4 // 审核通过
)

// SchoolNotifyText 通知类型文本映射
var SchoolNotifyText = map[int16]string{
	SchoolNotifyExpiring:  "到期提醒",
	SchoolNotifyBuffering: "缓冲期通知",
	SchoolNotifyFrozen:    "冻结通知",
	SchoolNotifyApproved:  "审核通过",
}

// GetSchoolNotifyText 获取学校通知类型文本
func GetSchoolNotifyText(t int16) string {
	if text, ok := SchoolNotifyText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidSchoolNotify 校验学校通知类型是否合法
func IsValidSchoolNotify(t int16) bool {
	_, ok := SchoolNotifyText[t]
	return ok
}

// ========== SSO协议类型（school_sso_configs.provider） ==========

const (
	SSOProviderCAS    = "cas"    // CAS
	SSOProviderOAuth2 = "oauth2" // OAuth2
)

// SSOProviderText SSO协议类型文本映射
var SSOProviderText = map[string]string{
	SSOProviderCAS:    "CAS",
	SSOProviderOAuth2: "OAuth2",
}

// GetSSOProviderText 获取SSO协议类型文本
func GetSSOProviderText(provider string) string {
	if text, ok := SSOProviderText[provider]; ok {
		return text
	}
	return "未知"
}

// IsValidSSOProvider 校验SSO协议类型是否合法
func IsValidSSOProvider(provider string) bool {
	_, ok := SSOProviderText[provider]
	return ok
}
