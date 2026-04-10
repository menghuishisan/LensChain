// system.go
// 模块08 — 系统管理与监控模块错误码
// 对照 docs/modules/08-系统管理与监控/03-API接口设计.md

package errcode

import "net/http"

var (
	ErrConfigNotFound     = New(40428, http.StatusNotFound, "配置项不存在")
	ErrAlertRuleNotFound  = New(40429, http.StatusNotFound, "告警规则不存在")
	ErrAlertEventNotFound = New(40430, http.StatusNotFound, "告警事件不存在")
	ErrBackupNotFound     = New(40431, http.StatusNotFound, "备份记录不存在")
	ErrBackupInProgress   = New(40945, http.StatusConflict, "已有备份任务正在执行")
	ErrConfigSensitive    = New(40025, http.StatusBadRequest, "敏感配置不可直接读取")
)
