// notification.go
// 模块07 — 通知与消息：HTTP 处理层。
// 该文件负责参数绑定、权限上下文透传和统一响应，不承载通知业务规则。

package notification

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	wsmanager "github.com/lenschain/backend/internal/pkg/ws"
	svc "github.com/lenschain/backend/internal/service/notification"
)

// NotificationHandler 模块07统一处理器。
type NotificationHandler struct {
	service svc.Service
}

// NewNotificationHandler 创建模块07处理器。
func NewNotificationHandler(service svc.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// ListInbox 获取收件箱列表。
func (h *NotificationHandler) ListInbox(c *gin.Context) {
	var req dto.InboxListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListInbox(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetInboxDetail 获取消息详情。
func (h *NotificationHandler) GetInboxDetail(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetInboxDetail(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// MarkInboxRead 标记单条消息已读。
func (h *NotificationHandler) MarkInboxRead(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkInboxRead(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "标记成功", nil)
}

// BatchReadInbox 批量标记消息已读。
func (h *NotificationHandler) BatchReadInbox(c *gin.Context) {
	var req dto.BatchReadNotificationsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.BatchReadInbox(c.Request.Context(), handlerctx.BuildServiceContext(c), &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "标记成功", nil)
}

// ReadAllInbox 标记全部消息已读。
func (h *NotificationHandler) ReadAllInbox(c *gin.Context) {
	if err := h.service.ReadAllInbox(c.Request.Context(), handlerctx.BuildServiceContext(c)); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "标记成功", nil)
}

// DeleteInbox 删除消息。
func (h *NotificationHandler) DeleteInbox(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteInbox(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// GetUnreadCount 获取未读消息计数。
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	respData, err := h.service.GetUnreadCount(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateAnnouncement 创建公告。
func (h *NotificationHandler) CreateAnnouncement(c *gin.Context) {
	var req dto.CreateSystemAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.CreateAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// ListAnnouncements 获取公告列表。
func (h *NotificationHandler) ListAnnouncements(c *gin.Context) {
	var req dto.NotificationAnnouncementListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListAnnouncements(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAnnouncement 获取公告详情。
func (h *NotificationHandler) GetAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateAnnouncement 更新公告。
func (h *NotificationHandler) UpdateAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateSystemAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// PublishAnnouncement 发布公告。
func (h *NotificationHandler) PublishAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.PublishAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", nil)
}

// UnpublishAnnouncement 下架公告。
func (h *NotificationHandler) UnpublishAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.UnpublishAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "下架成功", nil)
}

// DeleteAnnouncement 删除公告。
func (h *NotificationHandler) DeleteAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteAnnouncement(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// SendDirectNotification 发送定向通知。
func (h *NotificationHandler) SendDirectNotification(c *gin.Context) {
	var req dto.SendNotificationReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.SendDirectNotification(c.Request.Context(), handlerctx.BuildServiceContext(c), &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发送成功", nil)
}

// GetPreferences 获取通知偏好。
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	respData, err := h.service.GetPreferences(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdatePreferences 更新通知偏好。
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	var req dto.UpdateNotificationPreferencesReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.UpdatePreferences(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListTemplates 获取模板列表。
func (h *NotificationHandler) ListTemplates(c *gin.Context) {
	respData, err := h.service.ListTemplates(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetTemplate 获取模板详情。
func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetTemplate(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateTemplate 更新模板。
func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateNotificationTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateTemplate(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// PreviewTemplate 预览模板。
func (h *NotificationHandler) PreviewTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.PreviewNotificationTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.PreviewTemplate(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetStatistics 获取消息统计。
func (h *NotificationHandler) GetStatistics(c *gin.Context) {
	var req dto.NotificationStatisticsReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.GetStatistics(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SendInternalEvent 处理内部通知事件。
func (h *NotificationHandler) SendInternalEvent(c *gin.Context) {
	var req dto.InternalSendNotificationEventReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.DispatchEvent(c.Request.Context(), &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发送成功", nil)
}

// ServeWS 建立通知 WebSocket 连接。
func (h *NotificationHandler) ServeWS(c *gin.Context) {
	conn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	manager := wsmanager.GetManager()
	if manager == nil {
		_ = conn.Close()
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	client := wsmanager.NewClient(sc.UserID, conn)
	manager.Register(client)
	manager.JoinRoom(client, "notification:broadcast")

	go client.WritePump()
	h.pushInitialWSPayload(context.WithoutCancel(c.Request.Context()), client, sc)
	client.ReadPump(manager)
}

// pushInitialWSPayload 推送 WebSocket 首次连接后的未读计数。
func (h *NotificationHandler) pushInitialWSPayload(ctx context.Context, client *wsmanager.Client, sc *svcctx.ServiceContext) {
	if client == nil || sc == nil {
		return
	}
	respData, err := h.service.GetUnreadCount(ctx, sc)
	if err != nil {
		return
	}
	client.Send <- mustWSMessage("unread_count_update", respData)
}

// mustWSMessage 构建模块07 WebSocket 标准消息。
func mustWSMessage(messageType string, data interface{}) []byte {
	payload, _ := json.Marshal(&wsmanager.Message{
		Type:    messageType,
		Channel: "notification",
		Data:    data,
	})
	return payload
}
