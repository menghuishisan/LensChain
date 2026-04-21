// notification_repo.go
// 模块02 — 学校与租户管理：学校通知数据访问层
// 负责学校通知记录的 CRUD 操作
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

package schoolrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// NotificationRepository 学校通知数据访问接口
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.SchoolNotification) error
	MarkSent(ctx context.Context, id int64) error
	ExistsBySchoolAndType(ctx context.Context, schoolID int64, notifyType int16) (bool, error)
}

// notificationRepository 学校通知数据访问实现
type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建学校通知数据访问实例
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

// Create 创建通知记录
func (r *notificationRepository) Create(ctx context.Context, notification *entity.SchoolNotification) error {
	if notification.ID == 0 {
		notification.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(notification).Error
}

// MarkSent 标记通知为已发送
func (r *notificationRepository) MarkSent(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.SchoolNotification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_sent": true,
			"sent_at": now,
		}).Error
}

// ExistsBySchoolAndType 检查指定学校和类型的通知是否已存在。
// 到期提醒按学校维度只允许发送一次，因此这里不再引入时间窗口。
func (r *notificationRepository) ExistsBySchoolAndType(ctx context.Context, schoolID int64, notifyType int16) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.SchoolNotification{}).
		Where("school_id = ? AND type = ?", schoolID, notifyType).
		Count(&count).Error
	return count > 0, err
}
