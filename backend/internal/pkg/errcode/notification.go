// notification.go
// 该文件定义模块07“通知与消息”的错误码，包括站内信、公告、模板、偏好设置和内部事件
// 派发等场景，供通知模块和调用内部通知接口的上层模块统一复用。

package errcode

import "net/http"

var (
	ErrNotificationNotFound = New(40425, http.StatusNotFound, "通知不存在")
	ErrAnnouncementNotFound = New(40426, http.StatusNotFound, "公告不存在")
	ErrTemplateNotFoundNtf  = New(40427, http.StatusNotFound, "通知模板不存在")
	ErrDuplicateEventCode   = New(40944, http.StatusConflict, "事件编码已存在")
)
