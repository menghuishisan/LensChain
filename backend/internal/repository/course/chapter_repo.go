// chapter_repo.go
// 模块03 — 课程与教学：章节与课时数据访问层
// 负责章节、课时、课时附件的 CRUD 操作
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ChapterRepository 章节数据访问接口
type ChapterRepository interface {
	Create(ctx context.Context, chapter *entity.Chapter) error
	GetByID(ctx context.Context, id int64) (*entity.Chapter, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.Chapter, error)
	UpdateSortOrders(ctx context.Context, items []SortItem) error
	CountByCourseID(ctx context.Context, courseID int64) (int, error)
}

// LessonRepository 课时数据访问接口
type LessonRepository interface {
	Create(ctx context.Context, lesson *entity.Lesson) error
	GetByID(ctx context.Context, id int64) (*entity.Lesson, error)
	GetByIDWithAttachments(ctx context.Context, id int64) (*entity.Lesson, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	ListByChapterID(ctx context.Context, chapterID int64) ([]*entity.Lesson, error)
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.Lesson, error)
	UpdateSortOrders(ctx context.Context, items []SortItem) error
	CountByCourseID(ctx context.Context, courseID int64) (int, error)
}

// AttachmentRepository 课时附件数据访问接口
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *entity.LessonAttachment) error
	GetByID(ctx context.Context, id int64) (*entity.LessonAttachment, error)
	Delete(ctx context.Context, id int64) error
	ListByLessonID(ctx context.Context, lessonID int64) ([]*entity.LessonAttachment, error)
}

// SortItem 排序项
type SortItem struct {
	ID        int64
	SortOrder int
}

// ========== Chapter 实现 ==========

type chapterRepository struct {
	db *gorm.DB
}

// NewChapterRepository 创建章节数据访问实例
func NewChapterRepository(db *gorm.DB) ChapterRepository {
	return &chapterRepository{db: db}
}

// Create 创建章节
func (r *chapterRepository) Create(ctx context.Context, chapter *entity.Chapter) error {
	if chapter.ID == 0 {
		chapter.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(chapter).Error
}

// GetByID 根据ID获取章节
func (r *chapterRepository) GetByID(ctx context.Context, id int64) (*entity.Chapter, error) {
	var chapter entity.Chapter
	err := r.db.WithContext(ctx).First(&chapter, id).Error
	if err != nil {
		return nil, err
	}
	return &chapter, nil
}

// UpdateFields 更新章节指定字段
func (r *chapterRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Chapter{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除章节
func (r *chapterRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Chapter{}, id).Error
}

// ListByCourseID 获取课程下所有章节（按排序）
func (r *chapterRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.Chapter, error) {
	var chapters []*entity.Chapter
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order asc, created_at asc").
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc, created_at asc")
		}).
		Find(&chapters).Error
	return chapters, err
}

// UpdateSortOrders 批量更新章节排序
func (r *chapterRepository) UpdateSortOrders(ctx context.Context, items []SortItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&entity.Chapter{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CountByCourseID 统计课程下章节数
func (r *chapterRepository) CountByCourseID(ctx context.Context, courseID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Chapter{}).
		Where("course_id = ?", courseID).Count(&count).Error
	return int(count), err
}

// ========== Lesson 实现 ==========

type lessonRepository struct {
	db *gorm.DB
}

// NewLessonRepository 创建课时数据访问实例
func NewLessonRepository(db *gorm.DB) LessonRepository {
	return &lessonRepository{db: db}
}

// Create 创建课时
func (r *lessonRepository) Create(ctx context.Context, lesson *entity.Lesson) error {
	if lesson.ID == 0 {
		lesson.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(lesson).Error
}

// GetByID 根据ID获取课时
func (r *lessonRepository) GetByID(ctx context.Context, id int64) (*entity.Lesson, error) {
	var lesson entity.Lesson
	err := r.db.WithContext(ctx).First(&lesson, id).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// GetByIDWithAttachments 获取课时（含附件）
func (r *lessonRepository) GetByIDWithAttachments(ctx context.Context, id int64) (*entity.Lesson, error) {
	var lesson entity.Lesson
	err := r.db.WithContext(ctx).
		Preload("Attachments", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		First(&lesson, id).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

// UpdateFields 更新课时指定字段
func (r *lessonRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Lesson{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除课时
func (r *lessonRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Lesson{}, id).Error
}

// ListByChapterID 获取章节下所有课时（按排序）
func (r *lessonRepository) ListByChapterID(ctx context.Context, chapterID int64) ([]*entity.Lesson, error) {
	var lessons []*entity.Lesson
	err := r.db.WithContext(ctx).
		Where("chapter_id = ?", chapterID).
		Order("sort_order asc, created_at asc").
		Find(&lessons).Error
	return lessons, err
}

// ListByCourseID 获取课程下所有课时
func (r *lessonRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.Lesson, error) {
	var lessons []*entity.Lesson
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order asc, created_at asc").
		Find(&lessons).Error
	return lessons, err
}

// UpdateSortOrders 批量更新课时排序
func (r *lessonRepository) UpdateSortOrders(ctx context.Context, items []SortItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&entity.Lesson{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CountByCourseID 统计课程下课时数
func (r *lessonRepository) CountByCourseID(ctx context.Context, courseID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Lesson{}).
		Where("course_id = ?", courseID).Count(&count).Error
	return int(count), err
}

// ========== Attachment 实现 ==========

type attachmentRepository struct {
	db *gorm.DB
}

// NewAttachmentRepository 创建附件数据访问实例
func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

// Create 创建附件
func (r *attachmentRepository) Create(ctx context.Context, attachment *entity.LessonAttachment) error {
	if attachment.ID == 0 {
		attachment.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(attachment).Error
}

// GetByID 根据ID获取附件
func (r *attachmentRepository) GetByID(ctx context.Context, id int64) (*entity.LessonAttachment, error) {
	var attachment entity.LessonAttachment
	err := r.db.WithContext(ctx).First(&attachment, id).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// Delete 硬删除附件
func (r *attachmentRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.LessonAttachment{}, id).Error
}

// ListByLessonID 获取课时下所有附件
func (r *attachmentRepository) ListByLessonID(ctx context.Context, lessonID int64) ([]*entity.LessonAttachment, error) {
	var attachments []*entity.LessonAttachment
	err := r.db.WithContext(ctx).
		Where("lesson_id = ?", lessonID).
		Order("sort_order asc, created_at asc").
		Find(&attachments).Error
	return attachments, err
}
