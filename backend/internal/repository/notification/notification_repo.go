// notification_repo.go
// 模块07 — 通知与消息：站内信与消息统计数据访问层。
// 负责 notifications 表的收件箱查询、已读/删除更新、未读统计、批量写入和归档清理查询。

package notificationrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// NotificationRepository 站内信数据访问接口。
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	BatchCreate(ctx context.Context, notifications []*entity.Notification) error
	GetByID(ctx context.Context, id int64) (*entity.Notification, error)
	GetByReceiver(ctx context.Context, id, receiverID int64) (*entity.Notification, error)
	ListInbox(ctx context.Context, params *InboxListParams) ([]*entity.Notification, int64, error)
	ListRecent(ctx context.Context, receiverID int64, limit int) ([]*entity.Notification, error)
	MarkRead(ctx context.Context, id, receiverID int64, readAt time.Time) error
	BatchMarkRead(ctx context.Context, receiverID int64, ids []int64, readAt time.Time) (int64, error)
	MarkAllRead(ctx context.Context, receiverID int64, readAt time.Time) (int64, error)
	SoftDelete(ctx context.Context, id, receiverID int64, deletedAt time.Time) error
	UnreadCount(ctx context.Context, receiverID int64) (int64, error)
	UnreadCountByCategory(ctx context.Context, receiverID int64) ([]*UnreadCountByCategoryItem, error)
	ExistsByEvent(ctx context.Context, receiverID int64, eventType string, sourceID *int64) (bool, error)
	ListExpiredRead(ctx context.Context, before time.Time, limit int) ([]*entity.Notification, error)
	DeleteByIDs(ctx context.Context, ids []int64) error
	Statistics(ctx context.Context, params *NotificationStatisticsParams) (*NotificationStatistics, error)
	DailyTrend(ctx context.Context, params *NotificationStatisticsParams) ([]*DailyTrendItem, error)
	CategoryStatistics(ctx context.Context, params *NotificationStatisticsParams) ([]*CategoryStatisticsItem, error)
}

// InboxListParams 收件箱列表查询参数。
type InboxListParams struct {
	ReceiverID int64
	Category   int16
	IsRead     *bool
	Keyword    string
	Page       int
	PageSize   int
}

// UnreadCountByCategoryItem 未读分类统计项。
type UnreadCountByCategoryItem struct {
	Category int16 `gorm:"column:category"`
	Count    int64 `gorm:"column:count"`
}

// NotificationStatisticsParams 消息统计查询参数。
type NotificationStatisticsParams struct {
	SchoolID *int64
	From     *time.Time
	To       *time.Time
}

// NotificationStatistics 消息统计总览。
type NotificationStatistics struct {
	TotalSent int64   `gorm:"column:total_sent"`
	TotalRead int64   `gorm:"column:total_read"`
	ReadRate  float64 `gorm:"column:read_rate"`
}

// DailyTrendItem 每日消息趋势项。
type DailyTrendItem struct {
	Date string `gorm:"column:date"`
	Sent int64  `gorm:"column:sent"`
	Read int64  `gorm:"column:read"`
}

// CategoryStatisticsItem 分类统计项。
type CategoryStatisticsItem struct {
	Category int16   `gorm:"column:category"`
	Sent     int64   `gorm:"column:sent"`
	Read     int64   `gorm:"column:read"`
	ReadRate float64 `gorm:"column:read_rate"`
}

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建站内信数据访问实例。
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

// Create 创建站内信。
func (r *notificationRepository) Create(ctx context.Context, notification *entity.Notification) error {
	if notification.ID == 0 {
		notification.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(notification).Error
}

// BatchCreate 批量创建站内信。
func (r *notificationRepository) BatchCreate(ctx context.Context, notifications []*entity.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	for i := range notifications {
		if notifications[i].ID == 0 {
			notifications[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(notifications, 100).Error
}

// GetByID 根据 ID 获取站内信。
func (r *notificationRepository) GetByID(ctx context.Context, id int64) (*entity.Notification, error) {
	var notification entity.Notification
	err := r.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetByReceiver 获取指定用户的站内信。
func (r *notificationRepository) GetByReceiver(ctx context.Context, id, receiverID int64) (*entity.Notification, error) {
	var notification entity.Notification
	err := r.db.WithContext(ctx).
		Where("id = ? AND receiver_id = ? AND is_deleted = ?", id, receiverID, false).
		First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// ListInbox 查询收件箱列表。
func (r *notificationRepository) ListInbox(ctx context.Context, params *InboxListParams) ([]*entity.Notification, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("receiver_id = ? AND is_deleted = ?", params.ReceiverID, false).
		Scopes(database.WithKeywordSearch(params.Keyword, "title"))
	if params.Category > 0 {
		query = query.Where("category = ?", params.Category)
	}
	if params.IsRead != nil {
		query = query.Where("is_read = ?", *params.IsRead)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var notifications []*entity.Notification
	err := query.Order("created_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

// ListRecent 查询最近站内信预览。
func (r *notificationRepository) ListRecent(ctx context.Context, receiverID int64, limit int) ([]*entity.Notification, error) {
	var notifications []*entity.Notification
	err := r.db.WithContext(ctx).
		Where("receiver_id = ? AND is_deleted = ?", receiverID, false).
		Order("created_at desc").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

// MarkRead 标记单条消息为已读。
func (r *notificationRepository) MarkRead(ctx context.Context, id, receiverID int64, readAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("id = ? AND receiver_id = ? AND is_read = ? AND is_deleted = ?", id, receiverID, false, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": readAt,
		}).Error
}

// BatchMarkRead 批量标记消息为已读。
func (r *notificationRepository) BatchMarkRead(ctx context.Context, receiverID int64, ids []int64, readAt time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("receiver_id = ? AND id IN ? AND is_read = ? AND is_deleted = ?", receiverID, ids, false, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": readAt,
		})
	return result.RowsAffected, result.Error
}

// MarkAllRead 标记用户全部消息为已读。
func (r *notificationRepository) MarkAllRead(ctx context.Context, receiverID int64, readAt time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("receiver_id = ? AND is_read = ? AND is_deleted = ?", receiverID, false, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": readAt,
		})
	return result.RowsAffected, result.Error
}

// SoftDelete 软删除消息。
func (r *notificationRepository) SoftDelete(ctx context.Context, id, receiverID int64, deletedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("id = ? AND receiver_id = ? AND is_deleted = ?", id, receiverID, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": deletedAt,
		}).Error
}

// UnreadCount 统计未读消息总数。
func (r *notificationRepository) UnreadCount(ctx context.Context, receiverID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("receiver_id = ? AND is_read = ? AND is_deleted = ?", receiverID, false, false).
		Count(&count).Error
	return count, err
}

// UnreadCountByCategory 按分类统计未读消息数。
func (r *notificationRepository) UnreadCountByCategory(ctx context.Context, receiverID int64) ([]*UnreadCountByCategoryItem, error) {
	var items []*UnreadCountByCategoryItem
	err := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Select("category, COUNT(*) AS count").
		Where("receiver_id = ? AND is_read = ? AND is_deleted = ?", receiverID, false, false).
		Group("category").
		Order("category asc").
		Find(&items).Error
	return items, err
}

// ExistsByEvent 判断同事件是否已发送给用户。
func (r *notificationRepository) ExistsByEvent(ctx context.Context, receiverID int64, eventType string, sourceID *int64) (bool, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("receiver_id = ? AND event_type = ?", receiverID, eventType)
	if sourceID == nil {
		query = query.Where("source_id IS NULL")
	} else {
		query = query.Where("source_id = ?", *sourceID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListExpiredRead 查询超过保留期的已读消息。
func (r *notificationRepository) ListExpiredRead(ctx context.Context, before time.Time, limit int) ([]*entity.Notification, error) {
	var notifications []*entity.Notification
	err := r.db.WithContext(ctx).
		Where("is_read = ? AND created_at <= ?", true, before).
		Order("created_at asc").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

// DeleteByIDs 物理删除站内信。
func (r *notificationRepository) DeleteByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&entity.Notification{}).Error
}

// Statistics 查询消息统计总览。
func (r *notificationRepository) Statistics(ctx context.Context, params *NotificationStatisticsParams) (*NotificationStatistics, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{})
	if params.SchoolID != nil && *params.SchoolID > 0 {
		query = query.Where("school_id = ?", *params.SchoolID)
	}
	if params.From != nil {
		query = query.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("created_at <= ?", *params.To)
	}

	var stats NotificationStatistics
	err := query.Select(`
		COUNT(*) AS total_sent,
		COALESCE(SUM(CASE WHEN is_read THEN 1 ELSE 0 END), 0) AS total_read,
		COALESCE(AVG(CASE WHEN is_read THEN 1.0 ELSE 0 END), 0) AS read_rate
	`).Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// DailyTrend 查询消息日趋势。
func (r *notificationRepository) DailyTrend(ctx context.Context, params *NotificationStatisticsParams) ([]*DailyTrendItem, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{})
	if params.SchoolID != nil && *params.SchoolID > 0 {
		query = query.Where("school_id = ?", *params.SchoolID)
	}
	if params.From != nil {
		query = query.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("created_at <= ?", *params.To)
	}

	var items []*DailyTrendItem
	err := query.Select(`
		TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date,
		COUNT(*) AS sent,
		COALESCE(SUM(CASE WHEN is_read THEN 1 ELSE 0 END), 0) AS read
	`).Group("created_at::date").
		Order("created_at::date asc").
		Find(&items).Error
	return items, err
}

// CategoryStatistics 查询按分类聚合的消息统计。
func (r *notificationRepository) CategoryStatistics(ctx context.Context, params *NotificationStatisticsParams) ([]*CategoryStatisticsItem, error) {
	query := r.db.WithContext(ctx).Model(&entity.Notification{})
	if params.SchoolID != nil && *params.SchoolID > 0 {
		query = query.Where("school_id = ?", *params.SchoolID)
	}
	if params.From != nil {
		query = query.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("created_at <= ?", *params.To)
	}

	var items []*CategoryStatisticsItem
	err := query.Select(`
		category,
		COUNT(*) AS sent,
		COALESCE(SUM(CASE WHEN is_read THEN 1 ELSE 0 END), 0) AS read,
		COALESCE(AVG(CASE WHEN is_read THEN 1.0 ELSE 0 END), 0) AS read_rate
	`).Group("category").
		Order("category asc").
		Find(&items).Error
	return items, err
}
