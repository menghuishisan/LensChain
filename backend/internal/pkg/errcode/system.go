// system.go
// 该文件定义模块08“系统管理与监控”的错误码，主要覆盖统一审计、系统配置、告警规则、
// 告警事件、备份任务与运维面板相关能力，供系统模块聚合接口统一返回。

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
