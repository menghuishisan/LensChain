// user.go
// 模块01 — 用户与认证：枚举常量定义
// 对照 docs/modules/01-用户与认证/02-数据库设计.md
// 包含用户状态、学业层次、登录操作类型、登录方式等枚举

package enum

// ========== 用户状态（users.status） ==========

const (
	UserStatusActive   = 1 // 正常
	UserStatusDisabled = 2 // 禁用
	UserStatusArchived = 3 // 归档
)

// UserStatusText 用户状态文本映射
var UserStatusText = map[int16]string{
	UserStatusActive:   "正常",
	UserStatusDisabled: "禁用",
	UserStatusArchived: "归档",
}

// GetUserStatusText 获取用户状态文本
func GetUserStatusText(status int16) string {
	if text, ok := UserStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidUserStatus 校验用户状态是否合法
func IsValidUserStatus(status int16) bool {
	_, ok := UserStatusText[status]
	return ok
}

// ========== 学业层次（user_profiles.education_level） ==========

const (
	EduLevelJuniorCollege = 1 // 专科
	EduLevelUndergraduate = 2 // 本科
	EduLevelMaster        = 3 // 硕士
	EduLevelDoctor        = 4 // 博士
)

// EduLevelText 学业层次文本映射
var EduLevelText = map[int16]string{
	EduLevelJuniorCollege: "专科",
	EduLevelUndergraduate: "本科",
	EduLevelMaster:        "硕士",
	EduLevelDoctor:        "博士",
}

// GetEduLevelText 获取学业层次文本
func GetEduLevelText(level int16) string {
	if text, ok := EduLevelText[level]; ok {
		return text
	}
	return "未知"
}

// ParseEduLevel 从中文文本解析学业层次枚举值（用于Excel导入）
func ParseEduLevel(text string) int16 {
	for k, v := range EduLevelText {
		if v == text {
			return k
		}
	}
	return 0
}

// ========== 登录操作类型（login_logs.action） ==========

const (
	LoginActionSuccess = 1 // 登录成功
	LoginActionFail    = 2 // 登录失败
	LoginActionLogout  = 3 // 主动登出
	LoginActionKicked  = 4 // 被踢下线
	LoginActionLocked  = 5 // 账号锁定
)

// LoginActionText 登录操作类型文本映射
var LoginActionText = map[int16]string{
	LoginActionSuccess: "登录成功",
	LoginActionFail:    "登录失败",
	LoginActionLogout:  "主动登出",
	LoginActionKicked:  "被踢下线",
	LoginActionLocked:  "账号锁定",
}

// GetLoginActionText 获取登录操作类型文本
func GetLoginActionText(action int16) string {
	if text, ok := LoginActionText[action]; ok {
		return text
	}
	return "未知"
}

// ========== 登录方式（login_logs.login_method） ==========

const (
	LoginMethodPassword = 1 // 密码登录
	LoginMethodSSOCAS   = 2 // SSO-CAS
	LoginMethodSSOOAuth = 3 // SSO-OAuth2
)

// LoginMethodText 登录方式文本映射
var LoginMethodText = map[int16]string{
	LoginMethodPassword: "密码登录",
	LoginMethodSSOCAS:   "SSO-CAS",
	LoginMethodSSOOAuth: "SSO-OAuth2",
}

// GetLoginMethodText 获取登录方式文本
func GetLoginMethodText(method int16) string {
	if text, ok := LoginMethodText[method]; ok {
		return text
	}
	return "未知"
}

// ========== 角色编码常量 ==========

const (
	RoleSuperAdmin  = "super_admin"  // 超级管理员
	RoleSchoolAdmin = "school_admin" // 学校管理员
	RoleTeacher     = "teacher"      // 教师
	RoleStudent     = "student"      // 学生
)

// RoleText 角色编码文本映射
var RoleText = map[string]string{
	RoleSuperAdmin:  "超级管理员",
	RoleSchoolAdmin: "学校管理员",
	RoleTeacher:     "教师",
	RoleStudent:     "学生",
}

// GetRoleText 获取角色编码文本
func GetRoleText(role string) string {
	if text, ok := RoleText[role]; ok {
		return text
	}
	return "未知"
}

// IsValidRole 校验角色编码是否合法
func IsValidRole(role string) bool {
	_, ok := RoleText[role]
	return ok
}

// ========== 导入类型 ==========

const (
	ImportTypeStudent = "student" // 学生导入
	ImportTypeTeacher = "teacher" // 教师导入
)

// ImportTypeText 导入类型文本映射
var ImportTypeText = map[string]string{
	ImportTypeStudent: "学生导入",
	ImportTypeTeacher: "教师导入",
}

// GetImportTypeText 获取导入类型文本
func GetImportTypeText(t string) string {
	if text, ok := ImportTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidImportType 校验导入类型是否合法
func IsValidImportType(t string) bool {
	_, ok := ImportTypeText[t]
	return ok
}

// ========== 导入行状态 ==========

const (
	ImportRowValid    = "valid"    // 有效
	ImportRowInvalid  = "invalid"  // 无效
	ImportRowConflict = "conflict" // 冲突
)

// ImportRowStatusText 导入行状态文本映射
var ImportRowStatusText = map[string]string{
	ImportRowValid:    "有效",
	ImportRowInvalid:  "无效",
	ImportRowConflict: "冲突",
}

// GetImportRowStatusText 获取导入行状态文本
func GetImportRowStatusText(status string) string {
	if text, ok := ImportRowStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidImportRowStatus 校验导入行状态是否合法
func IsValidImportRowStatus(status string) bool {
	_, ok := ImportRowStatusText[status]
	return ok
}

// ========== 冲突处理策略 ==========

const (
	ConflictStrategySkip      = "skip"      // 跳过
	ConflictStrategyOverwrite = "overwrite" // 覆盖
)

// ConflictStrategyText 冲突处理策略文本映射
var ConflictStrategyText = map[string]string{
	ConflictStrategySkip:      "跳过",
	ConflictStrategyOverwrite: "覆盖",
}

// GetConflictStrategyText 获取冲突处理策略文本
func GetConflictStrategyText(strategy string) string {
	if text, ok := ConflictStrategyText[strategy]; ok {
		return text
	}
	return "未知"
}

// IsValidConflictStrategy 校验冲突处理策略是否合法
func IsValidConflictStrategy(strategy string) bool {
	_, ok := ConflictStrategyText[strategy]
	return ok
}
