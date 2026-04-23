// core.go
// 模块07 — 通知与消息：核心类型、依赖装配与公共辅助函数。
// 该文件集中定义通知服务接口、跨层依赖和通用构造逻辑，避免各功能文件重复声明。

package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/pagination"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
)

var placeholderPattern = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`)

const notificationBroadcastRoom = "notification:broadcast"

// Service 模块07统一服务接口。
type Service interface {
	ListInbox(ctx context.Context, sc *svcctx.ServiceContext, req *dto.InboxListReq) (*dto.InboxListResp, error)
	GetInboxDetail(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.NotificationInboxItem, error)
	MarkInboxRead(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	BatchReadInbox(ctx context.Context, sc *svcctx.ServiceContext, req *dto.BatchReadNotificationsReq) error
	ReadAllInbox(ctx context.Context, sc *svcctx.ServiceContext) error
	DeleteInbox(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	GetUnreadCount(ctx context.Context, sc *svcctx.ServiceContext) (*dto.UnreadCountResp, error)

	CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSystemAnnouncementReq) (*dto.SystemAnnouncementDetailResp, error)
	ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, req *dto.NotificationAnnouncementListReq) (*dto.SystemAnnouncementListResp, error)
	GetAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SystemAnnouncementDetailResp, error)
	UpdateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateSystemAnnouncementReq) error
	PublishAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	UnpublishAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	SendDirectNotification(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SendNotificationReq) error

	GetPreferences(ctx context.Context, sc *svcctx.ServiceContext) (*dto.NotificationPreferencesResp, error)
	UpdatePreferences(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateNotificationPreferencesReq) (*dto.NotificationPreferencesResp, error)

	ListTemplates(ctx context.Context, sc *svcctx.ServiceContext) (*dto.NotificationTemplateListResp, error)
	GetTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.NotificationTemplateDetailResp, error)
	UpdateTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateNotificationTemplateReq) error
	PreviewTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.PreviewNotificationTemplateReq) (*dto.PreviewNotificationTemplateResp, error)

	GetStatistics(ctx context.Context, sc *svcctx.ServiceContext, req *dto.NotificationStatisticsReq) (*dto.NotificationStatisticsResp, error)

	DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error
	ProcessEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error
	SyncUnreadCountForUser(ctx context.Context, userID int64) error
}

// EventDispatcher 提供给其他模块的最小通知发送接口。
type EventDispatcher interface {
	DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error
}

// service 模块07服务实现。
type service struct {
	db                   *gorm.DB
	notificationRepo     notificationrepo.NotificationRepository
	announcementRepo     notificationrepo.SystemAnnouncementRepository
	announcementReadRepo notificationrepo.AnnouncementReadStatusRepository
	templateRepo         notificationrepo.NotificationTemplateRepository
	preferenceRepo       notificationrepo.UserNotificationPreferenceRepository
	sourceRepo           notificationrepo.NotificationSourceRepository
	eventBus             EventBus
}

// NewService 创建模块07服务实例。
func NewService(
	db *gorm.DB,
	notificationRepo notificationrepo.NotificationRepository,
	announcementRepo notificationrepo.SystemAnnouncementRepository,
	announcementReadRepo notificationrepo.AnnouncementReadStatusRepository,
	templateRepo notificationrepo.NotificationTemplateRepository,
	preferenceRepo notificationrepo.UserNotificationPreferenceRepository,
	sourceRepo notificationrepo.NotificationSourceRepository,
) Service {
	return &service{
		db:                   db,
		notificationRepo:     notificationRepo,
		announcementRepo:     announcementRepo,
		announcementReadRepo: announcementReadRepo,
		templateRepo:         templateRepo,
		preferenceRepo:       preferenceRepo,
		sourceRepo:           sourceRepo,
	}
}

// normalizePagination 统一分页默认值。
func normalizePagination(page, pageSize int) (int, int) {
	return pagination.NormalizeValues(page, pageSize)
}

// buildPaginationResp 构建模块07通用分页结构。
func buildPaginationResp(page, pageSize int, total int64) dto.PaginationResp {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return dto.PaginationResp{
		Page:       page,
		PageSize:   pageSize,
		Total:      int(total),
		TotalPages: totalPages,
	}
}

// ensureAuthenticated 校验模块07的登录上下文。
func ensureAuthenticated(sc *svcctx.ServiceContext) error {
	if sc == nil || sc.UserID == 0 {
		return errcode.ErrUnauthorized
	}
	return nil
}

// formatTime 将时间格式化为 RFC3339 字符串指针。
func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

// int64String 将雪花 ID 转为字符串。
func int64String(value int64) string {
	return strconv.FormatInt(value, 10)
}

// optionalInt64String 将可选雪花 ID 转为字符串指针。
func optionalInt64String(value *int64) *string {
	if value == nil || *value == 0 {
		return nil
	}
	text := strconv.FormatInt(*value, 10)
	return &text
}

// parseSnowflakeID 解析字符串雪花 ID。
func parseSnowflakeID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, errcode.ErrInvalidID
	}
	return value, nil
}

// parseSnowflakeIDs 批量解析雪花 ID。
func parseSnowflakeIDs(rawIDs []string) ([]int64, error) {
	ids := make([]int64, 0, len(rawIDs))
	seen := make(map[int64]struct{}, len(rawIDs))
	for _, rawID := range rawIDs {
		id, err := parseSnowflakeID(rawID)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// isForcedCategory 判断分类是否属于强制接收通知。
func isForcedCategory(category int16) bool {
	return category == enum.NotificationCategorySystem || category == enum.NotificationCategoryGrade
}

// renderTextTemplate 渲染消息模板文本，并对变量值做 HTML 转义。
func renderTextTemplate(template string, params map[string]interface{}) string {
	result := template
	for key, value := range params {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, html.EscapeString(fmt.Sprint(value)))
	}
	return placeholderPattern.ReplaceAllString(result, "")
}

// inferCategoryFromEventType 根据事件编码推断通知分类。
func inferCategoryFromEventType(eventType string) int16 {
	switch {
	case strings.HasPrefix(eventType, "assignment.") || strings.HasPrefix(eventType, "course."):
		return enum.NotificationCategoryCourse
	case strings.HasPrefix(eventType, "experiment."):
		return enum.NotificationCategoryExperiment
	case strings.HasPrefix(eventType, "competition."):
		return enum.NotificationCategoryCompetition
	case strings.HasPrefix(eventType, "grade."):
		return enum.NotificationCategoryGrade
	default:
		return enum.NotificationCategorySystem
	}
}

// decodeTemplateVariables 解析模板变量定义。
func decodeTemplateVariables(raw datatypes.JSON) []dto.NotificationTemplateVariable {
	if len(raw) == 0 {
		return []dto.NotificationTemplateVariable{}
	}
	var variables []dto.NotificationTemplateVariable
	if err := json.Unmarshal(raw, &variables); err != nil {
		return []dto.NotificationTemplateVariable{}
	}
	return variables
}

// buildNotificationInboxItem 构建收件箱响应项。
func buildNotificationInboxItem(item *entity.Notification) dto.NotificationInboxItem {
	return dto.NotificationInboxItem{
		ID:           int64String(item.ID),
		Category:     item.Category,
		CategoryText: enum.GetNotificationCategoryText(item.Category),
		Title:        item.Title,
		Content:      item.Content,
		SourceModule: item.SourceModule,
		SourceType:   item.SourceType,
		SourceID:     optionalInt64String(item.SourceID),
		IsRead:       item.IsRead,
		ReadAt:       formatTime(item.ReadAt),
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// buildAnnouncementItem 构建公告列表项。
func buildAnnouncementItem(item *entity.SystemAnnouncement, isAdmin bool, isRead *bool) dto.SystemAnnouncementItem {
	resp := dto.SystemAnnouncementItem{
		ID:          int64String(item.ID),
		Title:       item.Title,
		Content:     item.Content,
		IsPinned:    item.IsPinned,
		IsRead:      isRead,
		PublishedAt: formatTime(item.PublishedAt),
	}
	if isAdmin {
		statusText := enum.GetSystemAnnouncementStatusText(item.Status)
		resp.Status = &item.Status
		resp.StatusText = &statusText
		createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
		resp.CreatedAt = &createdAt
	}
	return resp
}

// buildAnnouncementDetail 构建公告详情响应。
func buildAnnouncementDetail(item *entity.SystemAnnouncement, isAdmin bool, isRead *bool) *dto.SystemAnnouncementDetailResp {
	resp := &dto.SystemAnnouncementDetailResp{
		ID:            int64String(item.ID),
		Title:         item.Title,
		Content:       item.Content,
		IsPinned:      item.IsPinned,
		IsRead:        isRead,
		PublishedAt:   formatTime(item.PublishedAt),
		ScheduledAt:   formatTime(item.ScheduledAt),
		UnpublishedAt: formatTime(item.UnpublishedAt),
	}
	if isAdmin {
		statusText := enum.GetSystemAnnouncementStatusText(item.Status)
		resp.Status = &item.Status
		resp.StatusText = &statusText
		createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
		updatedAt := item.UpdatedAt.UTC().Format(time.RFC3339)
		resp.CreatedAt = &createdAt
		resp.UpdatedAt = &updatedAt
	}
	return resp
}

// buildPreferenceResp 构建用户通知偏好响应。
func buildPreferenceResp(preferences map[int16]bool) *dto.NotificationPreferencesResp {
	categories := []int16{
		enum.NotificationCategorySystem,
		enum.NotificationCategoryCourse,
		enum.NotificationCategoryExperiment,
		enum.NotificationCategoryCompetition,
		enum.NotificationCategoryGrade,
	}
	items := make([]dto.NotificationPreferenceItem, 0, len(categories))
	for _, category := range categories {
		enabled := preferences[category]
		if !enabled && !isForcedCategory(category) {
			// 非强制分类且未配置时默认开启。
			if _, exists := preferences[category]; !exists {
				enabled = true
			}
		}
		if isForcedCategory(category) {
			enabled = true
		}
		items = append(items, dto.NotificationPreferenceItem{
			Category:     category,
			CategoryText: enum.GetNotificationCategoryText(category),
			IsEnabled:    enabled,
			IsForced:     isForcedCategory(category),
		})
	}
	return &dto.NotificationPreferencesResp{Preferences: items}
}

// buildTemplateItem 构建模板列表项。
func buildTemplateItem(item *entity.NotificationTemplate) dto.NotificationTemplateListItem {
	return dto.NotificationTemplateListItem{
		ID:              int64String(item.ID),
		EventType:       item.EventType,
		Category:        item.Category,
		CategoryText:    enum.GetNotificationCategoryText(item.Category),
		TitleTemplate:   item.TitleTemplate,
		ContentTemplate: item.ContentTemplate,
		Variables:       decodeTemplateVariables(item.Variables),
		IsEnabled:       item.IsEnabled,
	}
}

// buildTemplateDetail 构建模板详情响应。
func buildTemplateDetail(item *entity.NotificationTemplate) *dto.NotificationTemplateDetailResp {
	resp := &dto.NotificationTemplateDetailResp{
		ID:              int64String(item.ID),
		EventType:       item.EventType,
		Category:        item.Category,
		CategoryText:    enum.GetNotificationCategoryText(item.Category),
		TitleTemplate:   item.TitleTemplate,
		ContentTemplate: item.ContentTemplate,
		Variables:       decodeTemplateVariables(item.Variables),
		IsEnabled:       item.IsEnabled,
	}
	createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
	updatedAt := item.UpdatedAt.UTC().Format(time.RFC3339)
	resp.CreatedAt = &createdAt
	resp.UpdatedAt = &updatedAt
	return resp
}
