// inbox_service.go
// 模块07 — 通知与消息：收件箱、偏好、模板与统计相关业务。
// 该文件承载用户侧消息中心的核心能力，避免把列表、已读、偏好和模板管理散落在多个小文件里。

package notification

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
)

// ListInbox 获取收件箱列表。
func (s *service) ListInbox(ctx context.Context, sc *svcctx.ServiceContext, req *dto.InboxListReq) (*dto.InboxListResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	items, total, err := s.notificationRepo.ListInbox(ctx, &notificationrepo.InboxListParams{
		ReceiverID: sc.UserID,
		Category:   req.Category,
		IsRead:     req.IsRead,
		Keyword:    req.Keyword,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, err
	}
	list := make([]dto.NotificationInboxItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, buildNotificationInboxItem(item))
	}
	unreadCount, err := s.loadUnreadCount(ctx, sc.UserID)
	if err != nil {
		return nil, err
	}
	return &dto.InboxListResp{
		List:        list,
		Pagination:  buildPaginationResp(page, pageSize, total),
		UnreadCount: unreadCount.Total,
	}, nil
}

// GetInboxDetail 获取消息详情，并自动标记已读。
func (s *service) GetInboxDetail(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.NotificationInboxItem, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	item, err := s.notificationRepo.GetByReceiver(ctx, id, sc.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotificationNotFound
		}
		return nil, err
	}
	if !item.IsRead {
		now := time.Now()
		if err := s.notificationRepo.MarkRead(ctx, id, sc.UserID, now); err != nil {
			return nil, err
		}
		s.syncUnreadCount(ctx, sc.UserID)
		item.IsRead = true
		item.ReadAt = &now
	}
	resp := buildNotificationInboxItem(item)
	return &resp, nil
}

// MarkInboxRead 标记单条消息已读。
func (s *service) MarkInboxRead(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if err := ensureAuthenticated(sc); err != nil {
		return err
	}
	item, err := s.notificationRepo.GetByReceiver(ctx, id, sc.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrNotificationNotFound
		}
		return err
	}
	if item.IsRead {
		return nil
	}
	if err := s.notificationRepo.MarkRead(ctx, id, sc.UserID, time.Now()); err != nil {
		return err
	}
	s.syncUnreadCount(ctx, sc.UserID)
	return nil
}

// BatchReadInbox 批量标记消息已读。
func (s *service) BatchReadInbox(ctx context.Context, sc *svcctx.ServiceContext, req *dto.BatchReadNotificationsReq) error {
	if err := ensureAuthenticated(sc); err != nil {
		return err
	}
	ids, err := parseSnowflakeIDs(req.IDs)
	if err != nil {
		return err
	}
	if _, err = s.notificationRepo.BatchMarkRead(ctx, sc.UserID, ids, time.Now()); err != nil {
		return err
	}
	s.syncUnreadCount(ctx, sc.UserID)
	return nil
}

// ReadAllInbox 标记当前用户全部消息为已读。
func (s *service) ReadAllInbox(ctx context.Context, sc *svcctx.ServiceContext) error {
	if err := ensureAuthenticated(sc); err != nil {
		return err
	}
	if _, err := s.notificationRepo.MarkAllRead(ctx, sc.UserID, time.Now()); err != nil {
		return err
	}
	s.syncUnreadCount(ctx, sc.UserID)
	return nil
}

// DeleteInbox 删除用户自己的站内信。
func (s *service) DeleteInbox(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if err := ensureAuthenticated(sc); err != nil {
		return err
	}
	_, err := s.notificationRepo.GetByReceiver(ctx, id, sc.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrNotificationNotFound
		}
		return err
	}
	if err := s.notificationRepo.SoftDelete(ctx, id, sc.UserID, time.Now()); err != nil {
		return err
	}
	s.syncUnreadCount(ctx, sc.UserID)
	return nil
}

// GetUnreadCount 获取未读计数与分类统计。
func (s *service) GetUnreadCount(ctx context.Context, sc *svcctx.ServiceContext) (*dto.UnreadCountResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	resp, err := s.loadUnreadCount(ctx, sc.UserID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPreferences 获取当前用户通知偏好。
func (s *service) GetPreferences(ctx context.Context, sc *svcctx.ServiceContext) (*dto.NotificationPreferencesResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	rawPreferences, err := s.preferenceRepo.ListByUser(ctx, sc.UserID)
	if err != nil {
		return nil, err
	}
	preferences := make(map[int16]bool, len(rawPreferences))
	for _, preference := range rawPreferences {
		if preference == nil {
			continue
		}
		preferences[preference.Category] = preference.IsEnabled
	}
	return buildPreferenceResp(preferences), nil
}

// UpdatePreferences 更新当前用户通知偏好。
func (s *service) UpdatePreferences(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateNotificationPreferencesReq) (*dto.NotificationPreferencesResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	for _, item := range req.Preferences {
		if isForcedCategory(item.Category) {
			continue
		}
		if err := s.preferenceRepo.Upsert(ctx, &entity.UserNotificationPreference{
			UserID:    sc.UserID,
			Category:  item.Category,
			IsEnabled: item.IsEnabled,
			UpdatedAt: time.Now(),
		}); err != nil {
			return nil, err
		}
	}
	return s.GetPreferences(ctx, sc)
}

// ListTemplates 获取模板列表。
func (s *service) ListTemplates(ctx context.Context, sc *svcctx.ServiceContext) (*dto.NotificationTemplateListResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	items, err := s.templateRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]dto.NotificationTemplateListItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, buildTemplateItem(item))
	}
	return &dto.NotificationTemplateListResp{List: list}, nil
}

// GetTemplate 获取模板详情。
func (s *service) GetTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.NotificationTemplateDetailResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	item, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrTemplateNotFoundNtf
		}
		return nil, err
	}
	return buildTemplateDetail(item), nil
}

// UpdateTemplate 更新消息模板。
func (s *service) UpdateTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateNotificationTemplateReq) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	_, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrTemplateNotFoundNtf
		}
		return err
	}
	return s.templateRepo.Update(ctx, id, map[string]interface{}{
		"title_template":   req.TitleTemplate,
		"content_template": req.ContentTemplate,
		"is_enabled":       req.IsEnabled,
		"updated_at":       time.Now(),
	})
}

// PreviewTemplate 预览模板渲染结果。
func (s *service) PreviewTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.PreviewNotificationTemplateReq) (*dto.PreviewNotificationTemplateResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrTemplateNotFoundNtf
		}
		return nil, err
	}
	return &dto.PreviewNotificationTemplateResp{
		Title:   renderTextTemplate(template.TitleTemplate, req.Params),
		Content: renderTextTemplate(template.ContentTemplate, req.Params),
	}, nil
}

// GetStatistics 获取消息统计。
func (s *service) GetStatistics(ctx context.Context, sc *svcctx.ServiceContext, req *dto.NotificationStatisticsReq) (*dto.NotificationStatisticsResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	params := &notificationrepo.NotificationStatisticsParams{}
	if req.DateFrom != "" {
		from, err := time.Parse("2006-01-02", req.DateFrom)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("date_from 格式错误")
		}
		params.From = &from
	}
	if req.DateTo != "" {
		to, err := time.Parse("2006-01-02", req.DateTo)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("date_to 格式错误")
		}
		endOfDay := to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		params.To = &endOfDay
	}
	stats, err := s.notificationRepo.Statistics(ctx, params)
	if err != nil {
		return nil, err
	}
	dailyTrend, err := s.notificationRepo.DailyTrend(ctx, params)
	if err != nil {
		return nil, err
	}
	categoryStats, err := s.notificationRepo.CategoryStatistics(ctx, params)
	if err != nil {
		return nil, err
	}
	resp := &dto.NotificationStatisticsResp{
		TotalSent:  int(stats.TotalSent),
		TotalRead:  int(stats.TotalRead),
		ReadRate:   stats.ReadRate,
		ByCategory: make([]dto.NotificationCategoryStat, 0, len(categoryStats)),
		DailyTrend: make([]dto.NotificationDailyTrendItem, 0, len(dailyTrend)),
	}
	for _, item := range categoryStats {
		if item == nil {
			continue
		}
		resp.ByCategory = append(resp.ByCategory, dto.NotificationCategoryStat{
			Category: enum.GetNotificationCategoryText(item.Category),
			Sent:     int(item.Sent),
			Read:     int(item.Read),
			ReadRate: item.ReadRate,
		})
	}
	for _, item := range dailyTrend {
		if item == nil {
			continue
		}
		resp.DailyTrend = append(resp.DailyTrend, dto.NotificationDailyTrendItem{
			Date: item.Date,
			Sent: int(item.Sent),
			Read: int(item.Read),
		})
	}
	return resp, nil
}

// canReceiveCategory 判断用户是否开启指定分类通知。
func (s *service) canReceiveCategory(ctx context.Context, userID int64, category int16) bool {
	enabled, err := s.preferenceRepo.IsEnabled(ctx, userID, category)
	return err == nil && enabled
}

// buildUnreadCountResp 从数据库聚合结果构建未读计数响应。
func buildUnreadCountResp(total int64, categoryItems []*notificationrepo.UnreadCountByCategoryItem) *dto.UnreadCountResp {
	resp := &dto.UnreadCountResp{
		Total:      int(total),
		ByCategory: dto.NotificationUnreadByCategory{},
	}
	for _, item := range categoryItems {
		if item == nil {
			continue
		}
		switch item.Category {
		case enum.NotificationCategorySystem:
			resp.ByCategory.System = int(item.Count)
		case enum.NotificationCategoryCourse:
			resp.ByCategory.Course = int(item.Count)
		case enum.NotificationCategoryExperiment:
			resp.ByCategory.Experiment = int(item.Count)
		case enum.NotificationCategoryCompetition:
			resp.ByCategory.Competition = int(item.Count)
		case enum.NotificationCategoryGrade:
			resp.ByCategory.Grade = int(item.Count)
		}
	}
	return resp
}

// buildSourceIDPointer 将可选来源 ID 字符串转为指针。
func buildSourceIDPointer(raw string) *int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return nil
	}
	return &value
}
