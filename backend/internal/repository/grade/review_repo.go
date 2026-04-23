// review_repo.go
// 模块06 — 评测与成绩：成绩审核数据访问层。
// 负责 grade_reviews 表的提交记录、审核状态、锁定状态和列表筛选。

package graderepo

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

// GradeReviewRepository 成绩审核数据访问接口。
type GradeReviewRepository interface {
	Create(ctx context.Context, review *entity.GradeReview) error
	GetByID(ctx context.Context, id int64) (*entity.GradeReview, error)
	GetByCourseSemester(ctx context.Context, courseID, semesterID int64) (*entity.GradeReview, error)
	List(ctx context.Context, params *GradeReviewListParams) ([]*entity.GradeReview, int64, error)
	ListApprovedBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.GradeReview, error)
	UpdateSubmitInfo(ctx context.Context, id int64, submittedBy int64, submitNote *string, submittedAt time.Time) error
	Approve(ctx context.Context, id, reviewerID int64, comment *string, reviewedAt time.Time) error
	Reject(ctx context.Context, id, reviewerID int64, comment *string, reviewedAt time.Time) error
	Unlock(ctx context.Context, id, unlockedBy int64, reason string, unlockedAt time.Time) error
	SetLocked(ctx context.Context, id int64, lockedAt time.Time) error
}

// GradeReviewListParams 成绩审核列表查询参数。
type GradeReviewListParams struct {
	SchoolID    int64
	SemesterID  int64
	CourseID    int64
	SubmittedBy int64
	Status      int16
	IsLocked    *bool
	From        *time.Time
	To          *time.Time
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
}

type gradeReviewRepository struct {
	db *gorm.DB
}

// NewGradeReviewRepository 创建成绩审核数据访问实例。
func NewGradeReviewRepository(db *gorm.DB) GradeReviewRepository {
	return &gradeReviewRepository{db: db}
}

// Create 创建成绩审核记录。
func (r *gradeReviewRepository) Create(ctx context.Context, review *entity.GradeReview) error {
	if review.ID == 0 {
		review.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(review).Error
}

// GetByID 根据 ID 获取成绩审核记录。
func (r *gradeReviewRepository) GetByID(ctx context.Context, id int64) (*entity.GradeReview, error) {
	var review entity.GradeReview
	err := r.db.WithContext(ctx).First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetByCourseSemester 获取课程在指定学期的审核记录。
func (r *gradeReviewRepository) GetByCourseSemester(ctx context.Context, courseID, semesterID int64) (*entity.GradeReview, error) {
	var review entity.GradeReview
	err := r.db.WithContext(ctx).
		Where("course_id = ? AND semester_id = ?", courseID, semesterID).
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// List 查询成绩审核列表。
func (r *gradeReviewRepository) List(ctx context.Context, params *GradeReviewListParams) ([]*entity.GradeReview, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Scopes(database.WithSchoolID(params.SchoolID), database.WithStatus(params.Status))
	if params.SemesterID > 0 {
		query = query.Where("semester_id = ?", params.SemesterID)
	}
	if params.CourseID > 0 {
		query = query.Where("course_id = ?", params.CourseID)
	}
	if params.SubmittedBy > 0 {
		query = query.Where("submitted_by = ?", params.SubmittedBy)
	}
	if params.IsLocked != nil {
		query = query.Where("is_locked = ?", *params.IsLocked)
	}
	if params.From != nil {
		query = query.Where("submitted_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("submitted_at <= ?", *params.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "submitted_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, map[string]string{
		"submitted_at": "submitted_at",
		"reviewed_at":  "reviewed_at",
		"created_at":   "created_at",
	})

	var reviews []*entity.GradeReview
	if err := query.Find(&reviews).Error; err != nil {
		return nil, 0, err
	}
	return reviews, total, nil
}

// ListApprovedBySemester 查询学期内已通过审核记录。
func (r *gradeReviewRepository) ListApprovedBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.GradeReview, error) {
	var reviews []*entity.GradeReview
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("semester_id = ? AND status = ?", semesterID, enum.GradeReviewStatusApproved).
		Order("reviewed_at asc").
		Find(&reviews).Error
	return reviews, err
}

// UpdateSubmitInfo 更新提交审核信息。
func (r *gradeReviewRepository) UpdateSubmitInfo(ctx context.Context, id int64, submittedBy int64, submitNote *string, submittedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"submitted_by": submittedBy,
			"submit_note":  submitNote,
			"submitted_at": submittedAt,
			"status":       enum.GradeReviewStatusPending,
			"is_locked":    false,
			"updated_at":   gorm.Expr("now()"),
		}).Error
}

// Approve 将成绩审核记录标记为通过。
func (r *gradeReviewRepository) Approve(ctx context.Context, id, reviewerID int64, comment *string, reviewedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         enum.GradeReviewStatusApproved,
			"reviewed_by":    reviewerID,
			"reviewed_at":    reviewedAt,
			"review_comment": comment,
			"is_locked":      true,
			"locked_at":      reviewedAt,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}

// Reject 将成绩审核记录标记为驳回。
func (r *gradeReviewRepository) Reject(ctx context.Context, id, reviewerID int64, comment *string, reviewedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         enum.GradeReviewStatusRejected,
			"reviewed_by":    reviewerID,
			"reviewed_at":    reviewedAt,
			"review_comment": comment,
			"is_locked":      false,
			"updated_at":     gorm.Expr("now()"),
		}).Error
}

// Unlock 记录成绩解锁信息。
func (r *gradeReviewRepository) Unlock(ctx context.Context, id, unlockedBy int64, reason string, unlockedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_locked":     false,
			"unlocked_by":   unlockedBy,
			"unlocked_at":   unlockedAt,
			"unlock_reason": reason,
			"status":        enum.GradeReviewStatusNotSubmitted,
			"updated_at":    gorm.Expr("now()"),
		}).Error
}

// SetLocked 更新审核记录锁定状态。
func (r *gradeReviewRepository) SetLocked(ctx context.Context, id int64, lockedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_locked":  true,
			"locked_at":  lockedAt,
			"updated_at": gorm.Expr("now()"),
		}).Error
}
