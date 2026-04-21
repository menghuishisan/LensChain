// discussion_repo.go
// 模块03 — 课程与教学：讨论、回复、点赞、公告数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"github.com/lenschain/backend/internal/pkg/pagination"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// DiscussionRepository 讨论数据访问接口
type DiscussionRepository interface {
	Create(ctx context.Context, discussion *entity.CourseDiscussion) error
	GetByID(ctx context.Context, id int64) (*entity.CourseDiscussion, error)
	SoftDelete(ctx context.Context, id int64) error
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	List(ctx context.Context, params *DiscussionListParams) ([]*entity.CourseDiscussion, int64, error)
	IncrReplyCount(ctx context.Context, id int64, delta int) error
	IncrLikeCount(ctx context.Context, id int64, delta int) error
}

// ReplyRepository 回复数据访问接口
type ReplyRepository interface {
	Create(ctx context.Context, reply *entity.DiscussionReply) error
	GetByID(ctx context.Context, id int64) (*entity.DiscussionReply, error)
	SoftDelete(ctx context.Context, id int64) error
	ListByDiscussionID(ctx context.Context, discussionID int64) ([]*entity.DiscussionReply, error)
}

// LikeRepository 点赞数据访问接口
type LikeRepository interface {
	Create(ctx context.Context, like *entity.DiscussionLike) error
	Delete(ctx context.Context, discussionID, userID int64) error
	Exists(ctx context.Context, discussionID, userID int64) (bool, error)
	ListByUserAndDiscussions(ctx context.Context, userID int64, discussionIDs []int64) ([]int64, error)
}

// AnnouncementRepository 公告数据访问接口
type AnnouncementRepository interface {
	Create(ctx context.Context, announcement *entity.CourseAnnouncement) error
	GetByID(ctx context.Context, id int64) (*entity.CourseAnnouncement, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, courseID int64, page, pageSize int) ([]*entity.CourseAnnouncement, int64, error)
}

// DiscussionListParams 讨论列表查询参数
type DiscussionListParams struct {
	CourseID int64
	Page     int
	PageSize int
}

// ========== Discussion 实现 ==========

type discussionRepository struct {
	db *gorm.DB
}

// NewDiscussionRepository 创建讨论数据访问实例
func NewDiscussionRepository(db *gorm.DB) DiscussionRepository {
	return &discussionRepository{db: db}
}

func (r *discussionRepository) Create(ctx context.Context, discussion *entity.CourseDiscussion) error {
	if discussion.ID == 0 {
		discussion.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(discussion).Error
}

func (r *discussionRepository) GetByID(ctx context.Context, id int64) (*entity.CourseDiscussion, error) {
	var d entity.CourseDiscussion
	err := r.db.WithContext(ctx).First(&d, id).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *discussionRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CourseDiscussion{}, id).Error
}

func (r *discussionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CourseDiscussion{}).Where("id = ?", id).Updates(fields).Error
}

func (r *discussionRepository) List(ctx context.Context, params *DiscussionListParams) ([]*entity.CourseDiscussion, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.CourseDiscussion{}).
		Where("course_id = ?", params.CourseID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序固定为：置顶优先，其余按最新回复时间倒序。
	// 这是模块03文档定义的唯一排序规则，不保留未文档化的自定义排序入口。
	query = query.Order("is_pinned desc, COALESCE(last_replied_at, created_at) desc")

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var discussions []*entity.CourseDiscussion
	if err := query.Find(&discussions).Error; err != nil {
		return nil, 0, err
	}
	return discussions, total, nil
}

func (r *discussionRepository) IncrReplyCount(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.CourseDiscussion{}).
		Where("id = ?", id).
		UpdateColumn("reply_count", gorm.Expr("reply_count + ?", delta)).Error
}

func (r *discussionRepository) IncrLikeCount(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.CourseDiscussion{}).
		Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// ========== Reply 实现 ==========

type replyRepository struct {
	db *gorm.DB
}

// NewReplyRepository 创建回复数据访问实例
func NewReplyRepository(db *gorm.DB) ReplyRepository {
	return &replyRepository{db: db}
}

func (r *replyRepository) Create(ctx context.Context, reply *entity.DiscussionReply) error {
	if reply.ID == 0 {
		reply.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(reply).Error
}

func (r *replyRepository) GetByID(ctx context.Context, id int64) (*entity.DiscussionReply, error) {
	var reply entity.DiscussionReply
	err := r.db.WithContext(ctx).First(&reply, id).Error
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

func (r *replyRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.DiscussionReply{}, id).Error
}

func (r *replyRepository) ListByDiscussionID(ctx context.Context, discussionID int64) ([]*entity.DiscussionReply, error) {
	var replies []*entity.DiscussionReply
	err := r.db.WithContext(ctx).
		Where("discussion_id = ?", discussionID).
		Order("created_at asc").
		Find(&replies).Error
	return replies, err
}

// ========== Like 实现 ==========

type likeRepository struct {
	db *gorm.DB
}

// NewLikeRepository 创建点赞数据访问实例
func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) Create(ctx context.Context, like *entity.DiscussionLike) error {
	if like.ID == 0 {
		like.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(like).Error
}

func (r *likeRepository) Delete(ctx context.Context, discussionID, userID int64) error {
	return r.db.WithContext(ctx).
		Where("discussion_id = ? AND user_id = ?", discussionID, userID).
		Delete(&entity.DiscussionLike{}).Error
}

func (r *likeRepository) Exists(ctx context.Context, discussionID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.DiscussionLike{}).
		Where("discussion_id = ? AND user_id = ?", discussionID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *likeRepository) ListByUserAndDiscussions(ctx context.Context, userID int64, discussionIDs []int64) ([]int64, error) {
	if len(discussionIDs) == 0 {
		return []int64{}, nil
	}
	var ids []int64
	err := r.db.WithContext(ctx).Model(&entity.DiscussionLike{}).
		Where("user_id = ? AND discussion_id IN ?", userID, discussionIDs).
		Pluck("discussion_id", &ids).Error
	return ids, err
}

// ========== Announcement 实现 ==========

type announcementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository 创建公告数据访问实例
func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return &announcementRepository{db: db}
}

func (r *announcementRepository) Create(ctx context.Context, announcement *entity.CourseAnnouncement) error {
	if announcement.ID == 0 {
		announcement.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(announcement).Error
}

func (r *announcementRepository) GetByID(ctx context.Context, id int64) (*entity.CourseAnnouncement, error) {
	var a entity.CourseAnnouncement
	err := r.db.WithContext(ctx).First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *announcementRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CourseAnnouncement{}).Where("id = ?", id).Updates(fields).Error
}

func (r *announcementRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CourseAnnouncement{}, id).Error
}

func (r *announcementRepository) List(ctx context.Context, courseID int64, page, pageSize int) ([]*entity.CourseAnnouncement, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.CourseAnnouncement{}).
		Where("course_id = ?", courseID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize = pagination.NormalizeValues(page, pageSize)
	query = query.Order("is_pinned desc, created_at desc").
		Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var announcements []*entity.CourseAnnouncement
	if err := query.Find(&announcements).Error; err != nil {
		return nil, 0, err
	}
	return announcements, total, nil
}
