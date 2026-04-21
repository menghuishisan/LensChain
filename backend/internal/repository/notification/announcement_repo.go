// announcement_repo.go
// 模块07 — 通知与消息：系统公告与公告阅读状态数据访问层。
// 负责 system_announcements、announcement_read_status 的创建、发布、列表、已读记录和定时发布查询。

package notificationrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SystemAnnouncementRepository 系统公告数据访问接口。
type SystemAnnouncementRepository interface {
	Create(ctx context.Context, announcement *entity.SystemAnnouncement) error
	GetByID(ctx context.Context, id int64) (*entity.SystemAnnouncement, error)
	List(ctx context.Context, params *AnnouncementListParams) ([]*entity.SystemAnnouncement, int64, error)
	ListPublished(ctx context.Context, page, pageSize int) ([]*entity.SystemAnnouncement, int64, error)
	ListPublishDue(ctx context.Context, now time.Time, limit int) ([]*entity.SystemAnnouncement, error)
	Update(ctx context.Context, id int64, values map[string]interface{}) error
	Publish(ctx context.Context, id int64, publishedAt time.Time) error
	Unpublish(ctx context.Context, id int64, unpublishedAt time.Time) error
	Delete(ctx context.Context, id int64) error
}

// AnnouncementListParams 公告列表查询参数。
type AnnouncementListParams struct {
	Status   int16
	Page     int
	PageSize int
}

type systemAnnouncementRepository struct {
	db *gorm.DB
}

// NewSystemAnnouncementRepository 创建系统公告数据访问实例。
func NewSystemAnnouncementRepository(db *gorm.DB) SystemAnnouncementRepository {
	return &systemAnnouncementRepository{db: db}
}

// Create 创建系统公告。
func (r *systemAnnouncementRepository) Create(ctx context.Context, announcement *entity.SystemAnnouncement) error {
	if announcement.ID == 0 {
		announcement.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(announcement).Error
}

// GetByID 根据 ID 获取系统公告。
func (r *systemAnnouncementRepository) GetByID(ctx context.Context, id int64) (*entity.SystemAnnouncement, error) {
	var announcement entity.SystemAnnouncement
	err := r.db.WithContext(ctx).First(&announcement, id).Error
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

// List 查询系统公告列表。
func (r *systemAnnouncementRepository) List(ctx context.Context, params *AnnouncementListParams) ([]*entity.SystemAnnouncement, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.SystemAnnouncement{})
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var announcements []*entity.SystemAnnouncement
	err := query.Order("is_pinned desc, created_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&announcements).Error
	if err != nil {
		return nil, 0, err
	}
	return announcements, total, nil
}

// ListPublished 查询已发布公告。
func (r *systemAnnouncementRepository) ListPublished(ctx context.Context, page, pageSize int) ([]*entity.SystemAnnouncement, int64, error) {
	return r.List(ctx, &AnnouncementListParams{
		Status:   enum.SystemAnnouncementStatusPublished,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListPublishDue 查询到达发布时间的草稿公告。
func (r *systemAnnouncementRepository) ListPublishDue(ctx context.Context, now time.Time, limit int) ([]*entity.SystemAnnouncement, error) {
	var announcements []*entity.SystemAnnouncement
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", enum.SystemAnnouncementStatusDraft, now).
		Order("scheduled_at asc").
		Limit(limit).
		Find(&announcements).Error
	return announcements, err
}

// Update 更新系统公告字段。
func (r *systemAnnouncementRepository) Update(ctx context.Context, id int64, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.SystemAnnouncement{}).
		Where("id = ?", id).
		Updates(values).Error
}

// Publish 发布系统公告。
func (r *systemAnnouncementRepository) Publish(ctx context.Context, id int64, publishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.SystemAnnouncement{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       enum.SystemAnnouncementStatusPublished,
			"published_at": publishedAt,
			"updated_at":   gorm.Expr("now()"),
		}).Error
}

// Unpublish 下架系统公告。
func (r *systemAnnouncementRepository) Unpublish(ctx context.Context, id int64, unpublishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.SystemAnnouncement{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         enum.SystemAnnouncementStatusUnpublished,
			"unpublished_at": unpublishedAt,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}

// Delete 软删除系统公告。
func (r *systemAnnouncementRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.SystemAnnouncement{}, id).Error
}

// AnnouncementReadStatusRepository 公告阅读状态数据访问接口。
type AnnouncementReadStatusRepository interface {
	Create(ctx context.Context, status *entity.AnnouncementReadStatus) error
	GetByAnnouncementAndUser(ctx context.Context, announcementID, userID int64) (*entity.AnnouncementReadStatus, error)
	HasRead(ctx context.Context, announcementID, userID int64) (bool, error)
	ListReadAnnouncementIDs(ctx context.Context, userID int64, ids []int64) ([]int64, error)
}

type announcementReadStatusRepository struct {
	db *gorm.DB
}

// NewAnnouncementReadStatusRepository 创建公告阅读状态数据访问实例。
func NewAnnouncementReadStatusRepository(db *gorm.DB) AnnouncementReadStatusRepository {
	return &announcementReadStatusRepository{db: db}
}

// Create 创建公告阅读状态。
func (r *announcementReadStatusRepository) Create(ctx context.Context, status *entity.AnnouncementReadStatus) error {
	if status.ID == 0 {
		status.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(status).Error
}

// GetByAnnouncementAndUser 获取用户公告阅读状态。
func (r *announcementReadStatusRepository) GetByAnnouncementAndUser(ctx context.Context, announcementID, userID int64) (*entity.AnnouncementReadStatus, error) {
	var status entity.AnnouncementReadStatus
	err := r.db.WithContext(ctx).
		Where("announcement_id = ? AND user_id = ?", announcementID, userID).
		First(&status).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// HasRead 判断用户是否已读公告。
func (r *announcementReadStatusRepository) HasRead(ctx context.Context, announcementID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AnnouncementReadStatus{}).
		Where("announcement_id = ? AND user_id = ?", announcementID, userID).
		Count(&count).Error
	return count > 0, err
}

// ListReadAnnouncementIDs 批量查询用户已读公告 ID。
func (r *announcementReadStatusRepository) ListReadAnnouncementIDs(ctx context.Context, userID int64, ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	var readIDs []int64
	err := r.db.WithContext(ctx).Model(&entity.AnnouncementReadStatus{}).
		Where("user_id = ? AND announcement_id IN ?", userID, ids).
		Pluck("announcement_id", &readIDs).Error
	return readIDs, err
}
