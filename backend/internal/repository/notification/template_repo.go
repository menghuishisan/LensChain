// template_repo.go
// 模块07 — 通知与消息：消息模板与用户通知偏好数据访问层。
// 负责 notification_templates、user_notification_preferences 的查询、更新和默认偏好读取支撑。

package notificationrepo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// NotificationTemplateRepository 消息模板数据访问接口。
type NotificationTemplateRepository interface {
	GetByID(ctx context.Context, id int64) (*entity.NotificationTemplate, error)
	GetByEventType(ctx context.Context, eventType string) (*entity.NotificationTemplate, error)
	List(ctx context.Context) ([]*entity.NotificationTemplate, error)
	Update(ctx context.Context, id int64, values map[string]interface{}) error
}

type notificationTemplateRepository struct {
	db *gorm.DB
}

// NewNotificationTemplateRepository 创建消息模板数据访问实例。
func NewNotificationTemplateRepository(db *gorm.DB) NotificationTemplateRepository {
	return &notificationTemplateRepository{db: db}
}

// GetByID 根据 ID 获取消息模板。
func (r *notificationTemplateRepository) GetByID(ctx context.Context, id int64) (*entity.NotificationTemplate, error) {
	var template entity.NotificationTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetByEventType 根据事件类型获取消息模板。
func (r *notificationTemplateRepository) GetByEventType(ctx context.Context, eventType string) (*entity.NotificationTemplate, error) {
	var template entity.NotificationTemplate
	err := r.db.WithContext(ctx).Where("event_type = ?", eventType).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// List 查询消息模板列表。
func (r *notificationTemplateRepository) List(ctx context.Context) ([]*entity.NotificationTemplate, error) {
	var templates []*entity.NotificationTemplate
	err := r.db.WithContext(ctx).Order("event_type asc").Find(&templates).Error
	return templates, err
}

// Update 更新消息模板字段。
func (r *notificationTemplateRepository) Update(ctx context.Context, id int64, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.NotificationTemplate{}).
		Where("id = ?", id).
		Updates(values).Error
}

// UserNotificationPreferenceRepository 用户通知偏好数据访问接口。
type UserNotificationPreferenceRepository interface {
	GetByUserAndCategory(ctx context.Context, userID int64, category int16) (*entity.UserNotificationPreference, error)
	ListByUser(ctx context.Context, userID int64) ([]*entity.UserNotificationPreference, error)
	Upsert(ctx context.Context, preference *entity.UserNotificationPreference) error
	IsEnabled(ctx context.Context, userID int64, category int16) (bool, error)
}

type userNotificationPreferenceRepository struct {
	db *gorm.DB
}

// NewUserNotificationPreferenceRepository 创建用户通知偏好数据访问实例。
func NewUserNotificationPreferenceRepository(db *gorm.DB) UserNotificationPreferenceRepository {
	return &userNotificationPreferenceRepository{db: db}
}

// GetByUserAndCategory 获取用户分类通知偏好。
func (r *userNotificationPreferenceRepository) GetByUserAndCategory(ctx context.Context, userID int64, category int16) (*entity.UserNotificationPreference, error) {
	var preference entity.UserNotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND category = ?", userID, category).
		First(&preference).Error
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

// ListByUser 查询用户通知偏好列表。
func (r *userNotificationPreferenceRepository) ListByUser(ctx context.Context, userID int64) ([]*entity.UserNotificationPreference, error) {
	var preferences []*entity.UserNotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("category asc").
		Find(&preferences).Error
	return preferences, err
}

// Upsert 保存用户通知偏好。
func (r *userNotificationPreferenceRepository) Upsert(ctx context.Context, preference *entity.UserNotificationPreference) error {
	if preference.ID == 0 {
		preference.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "category"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"is_enabled",
			"updated_at",
		}),
	}).Create(preference).Error
}

// IsEnabled 判断用户是否开启某分类通知。
// 系统通知和成绩通知属于强制分类，不允许关闭。
func (r *userNotificationPreferenceRepository) IsEnabled(ctx context.Context, userID int64, category int16) (bool, error) {
	if category == enum.NotificationCategorySystem || category == enum.NotificationCategoryGrade {
		return true, nil
	}
	preference, err := r.GetByUserAndCategory(ctx, userID, category)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}
	return preference.IsEnabled, nil
}
