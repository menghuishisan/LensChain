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
var UserStatusText = map[int]string{
	UserStatusActive:   "正常",
	UserStatusDisabled: "禁用",
	UserStatusArchived: "归档",
}

// GetUserStatusText 获取用户状态文本
func GetUserStatusText(status int) string {
	if text, ok := UserStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidUserStatus 校验用户状态是否合法
func IsValidUserStatus(status int) bool {
	_, ok := UserStatusText[status]
	return ok
}

// ========== 学业层次（user_profiles.education_level） ==========

const (
	EduLevelUndergraduate = 1 // 本科
	EduLevelMaster        = 2 // 硕士
	EduLevelDoctor        = 3 // 博士
)

// EduLevelText 学业层次文本映射
var EduLevelText = map[int]string{
	EduLevelUndergraduate: "本科",
	EduLevelMaster:        "硕士",
	EduLevelDoctor:        "博士",
}

// GetEduLevelText 获取学业层次文本
func GetEduLevelText(level int) string {
	if text, ok := EduLevelText[level]; ok {
		return text
	}
	return "未知"
}

// ParseEduLevel 从中文文本解析学业层次枚举值（用于Excel导入）
func ParseEduLevel(text string) int {
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
var LoginActionText = map[int]string{
	LoginActionSuccess: "登录成功",
	LoginActionFail:    "登录失败",
	LoginActionLogout:  "主动登出",
	LoginActionKicked:  "被踢下线",
	LoginActionLocked:  "账号锁定",
}

// GetLoginActionText 获取登录操作类型文本
func GetLoginActionText(action int) string {
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
var LoginMethodText = map[int]string{
	LoginMethodPassword: "密码登录",
	LoginMethodSSOCAS:   "SSO-CAS",
	LoginMethodSSOOAuth: "SSO-OAuth2",
}

// GetLoginMethodText 获取登录方式文本
func GetLoginMethodText(method int) string {
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

// ========== 导入类型 ==========

const (
	ImportTypeStudent = "student" // 学生导入
	ImportTypeTeacher = "teacher" // 教师导入
)

// ========== 导入行状态 ==========

const (
	ImportRowValid    = "valid"    // 有效
	ImportRowInvalid  = "invalid"  // 无效
	ImportRowConflict = "conflict" // 冲突
)

// ========== 冲突处理策略 ==========

const (
	ConflictStrategySkip      = "skip"      // 跳过
	ConflictStrategyOverwrite = "overwrite"  // 覆盖
)
