// notification.go
// 模块07 — 通知与消息：请求/响应 DTO 定义。
// 该文件对齐 docs/modules/07-通知与消息/03-API接口设计.md，覆盖收件箱、公告、定向通知、偏好、模板、内部事件和统计接口。

package dto

// InboxListReq 收件箱列表查询参数。
type InboxListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Category int16  `form:"category" binding:"omitempty,oneof=1 2 3 4 5"`
	IsRead   *bool  `form:"is_read"`
	Keyword  string `form:"keyword"`
}

// NotificationInboxItem 收件箱列表项。
type NotificationInboxItem struct {
	ID           string  `json:"id"`
	Category     int16   `json:"category"`
	CategoryText string  `json:"category_text"`
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	SourceModule string  `json:"source_module"`
	SourceType   *string `json:"source_type"`
	SourceID     *string `json:"source_id"`
	IsRead       bool    `json:"is_read"`
	ReadAt       *string `json:"read_at"`
	CreatedAt    string  `json:"created_at"`
}

// InboxListResp 收件箱列表响应。
type InboxListResp struct {
	List        []NotificationInboxItem `json:"list"`
	Pagination  PaginationResp          `json:"pagination"`
	UnreadCount int                     `json:"unread_count"`
}

// BatchReadNotificationsReq 批量标记已读请求。
type BatchReadNotificationsReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// UnreadCountResp 未读消息计数响应。
type UnreadCountResp struct {
	Total      int                          `json:"total"`
	ByCategory NotificationUnreadByCategory `json:"by_category"`
}

// NotificationUnreadByCategory 未读消息分类统计。
// 该结构对应未读计数接口的固定分类键，避免使用无约束 map。
type NotificationUnreadByCategory struct {
	System      int `json:"system"`
	Course      int `json:"course"`
	Experiment  int `json:"experiment"`
	Competition int `json:"competition"`
	Grade       int `json:"grade"`
}

// CreateSystemAnnouncementReq 创建公告请求。
type CreateSystemAnnouncementReq struct {
	Title       string  `json:"title" binding:"required,max=200"`
	Content     string  `json:"content" binding:"required"`
	ScheduledAt *string `json:"scheduled_at"`
}

// UpdateSystemAnnouncementReq 编辑公告请求。
type UpdateSystemAnnouncementReq struct {
	Title       *string `json:"title" binding:"omitempty,max=200"`
	Content     *string `json:"content"`
	ScheduledAt *string `json:"scheduled_at"`
	IsPinned    *bool   `json:"is_pinned"`
}

// SystemAnnouncementItem 公告列表项。
type SystemAnnouncementItem struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPinned    bool    `json:"is_pinned"`
	IsRead      *bool   `json:"is_read,omitempty"`
	Status      *int16  `json:"status,omitempty"`
	StatusText  *string `json:"status_text,omitempty"`
	PublishedAt *string `json:"published_at"`
	CreatedAt   *string `json:"created_at,omitempty"`
}

// SystemAnnouncementListResp 公告列表响应。
type SystemAnnouncementListResp struct {
	List       []SystemAnnouncementItem `json:"list"`
	Pagination PaginationResp           `json:"pagination"`
}

// SystemAnnouncementDetailResp 公告详情响应。
type SystemAnnouncementDetailResp struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	IsPinned      bool    `json:"is_pinned"`
	IsRead        *bool   `json:"is_read,omitempty"`
	Status        *int16  `json:"status,omitempty"`
	StatusText    *string `json:"status_text,omitempty"`
	PublishedAt   *string `json:"published_at,omitempty"`
	ScheduledAt   *string `json:"scheduled_at,omitempty"`
	UnpublishedAt *string `json:"unpublished_at,omitempty"`
	CreatedAt     *string `json:"created_at,omitempty"`
	UpdatedAt     *string `json:"updated_at,omitempty"`
}

// NotificationAnnouncementListReq 公告列表查询参数。
type NotificationAnnouncementListReq struct {
	Page     int   `form:"page" binding:"omitempty,min=1"`
	PageSize int   `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   int16 `form:"status" binding:"omitempty,oneof=1 2 3"`
}

// SendNotificationReq 发送定向通知请求。
type SendNotificationReq struct {
	Title      string `json:"title" binding:"required,max=200"`
	Content    string `json:"content" binding:"required"`
	TargetType string `json:"target_type" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
	Category   int16  `json:"category" binding:"required,oneof=1 2 3 4 5"`
}

// NotificationPreferenceItem 通知偏好项。
type NotificationPreferenceItem struct {
	Category     int16  `json:"category"`
	CategoryText string `json:"category_text"`
	IsEnabled    bool   `json:"is_enabled"`
	IsForced     bool   `json:"is_forced"`
}

// NotificationPreferencesResp 通知偏好响应。
type NotificationPreferencesResp struct {
	Preferences []NotificationPreferenceItem `json:"preferences"`
}

// UpdateNotificationPreferencesReq 更新通知偏好请求。
type UpdateNotificationPreferencesReq struct {
	Preferences []UpdateNotificationPreferenceItem `json:"preferences" binding:"required,min=1,dive"`
}

// UpdateNotificationPreferenceItem 更新通知偏好项。
type UpdateNotificationPreferenceItem struct {
	Category  int16 `json:"category" binding:"required,oneof=1 2 3 4 5"`
	IsEnabled bool  `json:"is_enabled"`
}

// NotificationTemplateListItem 模板列表项。
type NotificationTemplateListItem struct {
	ID              string                         `json:"id"`
	EventType       string                         `json:"event_type"`
	Category        int16                          `json:"category"`
	CategoryText    string                         `json:"category_text"`
	TitleTemplate   string                         `json:"title_template"`
	ContentTemplate string                         `json:"content_template"`
	Variables       []NotificationTemplateVariable `json:"variables"`
	IsEnabled       bool                           `json:"is_enabled"`
}

// NotificationTemplateDetailResp 模板详情响应。
type NotificationTemplateDetailResp struct {
	ID              string                         `json:"id"`
	EventType       string                         `json:"event_type"`
	Category        int16                          `json:"category"`
	CategoryText    string                         `json:"category_text"`
	TitleTemplate   string                         `json:"title_template"`
	ContentTemplate string                         `json:"content_template"`
	Variables       []NotificationTemplateVariable `json:"variables"`
	IsEnabled       bool                           `json:"is_enabled"`
	CreatedAt       *string                        `json:"created_at,omitempty"`
	UpdatedAt       *string                        `json:"updated_at,omitempty"`
}

// NotificationTemplateVariable 模板变量定义。
// 该结构对应 notification_templates.variables 的固定字段。
type NotificationTemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// NotificationTemplateListResp 模板列表响应。
type NotificationTemplateListResp struct {
	List []NotificationTemplateListItem `json:"list"`
}

// UpdateNotificationTemplateReq 更新模板请求。
type UpdateNotificationTemplateReq struct {
	TitleTemplate   string `json:"title_template" binding:"required,max=200"`
	ContentTemplate string `json:"content_template" binding:"required"`
	IsEnabled       bool   `json:"is_enabled"`
}

// PreviewNotificationTemplateReq 预览模板请求。
type PreviewNotificationTemplateReq struct {
	Params map[string]interface{} `json:"params" binding:"required"`
}

// PreviewNotificationTemplateResp 预览模板响应。
type PreviewNotificationTemplateResp struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// InternalSendNotificationEventReq 内部通知事件请求。
type InternalSendNotificationEventReq struct {
	EventType    string                 `json:"event_type" binding:"required"`
	ReceiverIDs  []string               `json:"receiver_ids" binding:"required,min=1"`
	Params       map[string]interface{} `json:"params" binding:"required"`
	SourceModule string                 `json:"source_module" binding:"required,max=20"`
	SourceType   string                 `json:"source_type" binding:"required,max=50"`
	SourceID     string                 `json:"source_id" binding:"required"`
}

// NotificationStatisticsReq 消息统计查询参数。
type NotificationStatisticsReq struct {
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
}

// NotificationStatisticsResp 消息统计响应。
type NotificationStatisticsResp struct {
	TotalSent  int                          `json:"total_sent"`
	TotalRead  int                          `json:"total_read"`
	ReadRate   float64                      `json:"read_rate"`
	ByCategory []NotificationCategoryStat   `json:"by_category"`
	DailyTrend []NotificationDailyTrendItem `json:"daily_trend"`
}

// NotificationCategoryStat 分类消息统计项。
type NotificationCategoryStat struct {
	Category string  `json:"category"`
	Sent     int     `json:"sent"`
	Read     int     `json:"read"`
	ReadRate float64 `json:"read_rate"`
}

// NotificationDailyTrendItem 每日消息趋势项。
type NotificationDailyTrendItem struct {
	Date string `json:"date"`
	Sent int    `json:"sent"`
	Read int    `json:"read"`
}
