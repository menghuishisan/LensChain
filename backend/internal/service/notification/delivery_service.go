// delivery_service.go
// 模块07 — 通知与消息：缓存与实时推送辅助能力。
// 该文件集中处理未读计数缓存、WebSocket 推送和公告广播，避免把交付逻辑散落到各业务文件。

package notification

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/logger"
	wsmanager "github.com/lenschain/backend/internal/pkg/ws"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
)

// loadUnreadCount 优先从缓存读取未读计数，不命中时回退数据库并回填缓存。
func (s *service) loadUnreadCount(ctx context.Context, userID int64) (*dto.UnreadCountResp, error) {
	if userID == 0 {
		return &dto.UnreadCountResp{ByCategory: dto.NotificationUnreadByCategory{}}, nil
	}
	if cache.Get() != nil {
		if raw, err := cache.GetString(ctx, cache.KeyNotifyUnread+strconv.FormatInt(userID, 10)); err == nil && raw != "" {
			var resp dto.UnreadCountResp
			if jsonErr := json.Unmarshal([]byte(raw), &resp); jsonErr == nil {
				return &resp, nil
			}
		}
	}
	return s.syncUnreadCount(ctx, userID)
}

// syncUnreadCount 从数据库重算单个用户未读计数、更新缓存并推送实时消息。
func (s *service) syncUnreadCount(ctx context.Context, userID int64) (*dto.UnreadCountResp, error) {
	total, err := s.notificationRepo.UnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	categoryItems, err := s.notificationRepo.UnreadCountByCategory(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := buildUnreadCountResp(total, categoryItems)
	s.writeUnreadCountCache(ctx, userID, resp)
	s.pushUnreadCount(userID, resp)
	return resp, nil
}

// syncUnreadCounts 批量刷新多个接收者的未读计数和前端铃铛状态。
func (s *service) syncUnreadCounts(ctx context.Context, userIDs []int64) {
	for _, userID := range uniqueReceiverIDs(userIDs) {
		if _, err := s.syncUnreadCount(ctx, userID); err != nil {
			logger.L.Warn("刷新通知未读计数失败", zap.Int64("user_id", userID), zap.Error(err))
		}
	}
}

// afterNotificationsCreated 在通知入库后统一处理缓存和 WebSocket 推送。
func (s *service) afterNotificationsCreated(ctx context.Context, notifications []*entity.Notification) {
	if len(notifications) == 0 {
		return
	}
	userIDs := make([]int64, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		userIDs = append(userIDs, notification.ReceiverID)
		s.pushNewNotification(notification)
	}
	s.syncUnreadCounts(ctx, userIDs)
}

// writeUnreadCountCache 回写用户未读计数缓存。
func (s *service) writeUnreadCountCache(ctx context.Context, userID int64, resp *dto.UnreadCountResp) {
	if userID == 0 || resp == nil || cache.Get() == nil {
		return
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		logger.L.Warn("序列化未读计数缓存失败", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	if err := cache.Set(ctx, cache.KeyNotifyUnread+strconv.FormatInt(userID, 10), string(raw), 0); err != nil {
		logger.L.Warn("写入未读计数缓存失败", zap.Int64("user_id", userID), zap.Error(err))
	}
}

// pushUnreadCount 推送未读计数更新消息。
func (s *service) pushUnreadCount(userID int64, resp *dto.UnreadCountResp) {
	manager := wsmanager.GetManager()
	if manager == nil || userID == 0 || resp == nil {
		return
	}
	if err := manager.SendToUser(userID, &wsmanager.Message{
		Type:    "unread_count_update",
		Channel: "notification",
		Data:    resp,
	}); err != nil {
		logger.L.Warn("推送未读计数失败", zap.Int64("user_id", userID), zap.Error(err))
	}
}

// pushNewNotification 推送新站内信预览。
func (s *service) pushNewNotification(notification *entity.Notification) {
	manager := wsmanager.GetManager()
	if manager == nil || notification == nil {
		return
	}
	payload := map[string]interface{}{
		"id":         int64String(notification.ID),
		"title":      notification.Title,
		"category":   notification.Category,
		"created_at": notification.CreatedAt.UTC().Format(time.RFC3339),
	}
	if err := manager.SendToUser(notification.ReceiverID, &wsmanager.Message{
		Type:    "new_notification",
		Channel: "notification",
		Data:    payload,
	}); err != nil {
		logger.L.Warn("推送新通知预览失败", zap.Int64("user_id", notification.ReceiverID), zap.Error(err))
	}
}

// broadcastAnnouncement 向所有在线用户广播新公告提醒。
func (s *service) broadcastAnnouncement(announcement *entity.SystemAnnouncement) {
	manager := wsmanager.GetManager()
	if manager == nil || announcement == nil || announcement.PublishedAt == nil {
		return
	}
	if err := manager.BroadcastToRoom(notificationBroadcastRoom, &wsmanager.Message{
		Type:    "new_announcement",
		Channel: "notification",
		Data: map[string]interface{}{
			"id":           int64String(announcement.ID),
			"title":        announcement.Title,
			"published_at": announcement.PublishedAt.UTC().Format(time.RFC3339),
		},
	}); err != nil {
		logger.L.Warn("广播系统公告失败", zap.Int64("announcement_id", announcement.ID), zap.Error(err))
	}
}

// uniqueReceiverIDs 对接收者 ID 去重并过滤零值。
func uniqueReceiverIDs(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// buildUnreadCountFromRepoResults 兼容定时或批量场景按仓储结果快速构建未读计数。
func buildUnreadCountFromRepoResults(total int64, categoryItems []*notificationrepo.UnreadCountByCategoryItem) *dto.UnreadCountResp {
	return buildUnreadCountResp(total, categoryItems)
}

// SyncUnreadCountForUser 对外暴露单个用户未读计数重算入口，供模块07定时任务对账复用。
func (s *service) SyncUnreadCountForUser(ctx context.Context, userID int64) error {
	_, err := s.syncUnreadCount(ctx, userID)
	return err
}

// BroadcastAnnouncementPublished 对外暴露系统公告广播入口，供模块07定时发布场景复用。
func (s *service) BroadcastAnnouncementPublished(announcement *entity.SystemAnnouncement) {
	s.broadcastAnnouncement(announcement)
}
