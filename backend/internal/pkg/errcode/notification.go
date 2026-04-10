// notification.go
// 模块07 — 通知与消息模块错误码
// 对照 docs/modules/07-通知与消息/03-API接口设计.md

package errcode

import "net/http"

var (
	ErrNotificationNotFound = New(40425, http.StatusNotFound, "通知不存在")
	ErrAnnouncementNotFound = New(40426, http.StatusNotFound, "公告不存在")
	ErrTemplateNotFoundNtf  = New(40427, http.StatusNotFound, "通知模板不存在")
	ErrDuplicateEventCode   = New(40944, http.StatusConflict, "事件编码已存在")
)
