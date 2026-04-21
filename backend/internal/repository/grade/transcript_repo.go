// transcript_repo.go
// 模块06 — 评测与成绩：成绩单生成记录数据访问层。
// 负责 transcript_records 表的创建、查询、列表和过期记录清理查询。

package graderepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// TranscriptRecordRepository 成绩单生成记录数据访问接口。
type TranscriptRecordRepository interface {
	Create(ctx context.Context, record *entity.TranscriptRecord) error
	GetByID(ctx context.Context, id int64) (*entity.TranscriptRecord, error)
	List(ctx context.Context, params *TranscriptRecordListParams) ([]*entity.TranscriptRecord, int64, error)
	ListExpired(ctx context.Context, before time.Time, limit int) ([]*entity.TranscriptRecord, error)
	Delete(ctx context.Context, id int64) error
}

// TranscriptRecordListParams 成绩单记录列表查询参数。
type TranscriptRecordListParams struct {
	SchoolID    int64
	StudentID   int64
	GeneratedBy int64
	From        *time.Time
	To          *time.Time
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
}

type transcriptRecordRepository struct {
	db *gorm.DB
}

// NewTranscriptRecordRepository 创建成绩单记录数据访问实例。
func NewTranscriptRecordRepository(db *gorm.DB) TranscriptRecordRepository {
	return &transcriptRecordRepository{db: db}
}

// Create 创建成绩单生成记录。
func (r *transcriptRecordRepository) Create(ctx context.Context, record *entity.TranscriptRecord) error {
	if record.ID == 0 {
		record.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByID 根据 ID 获取成绩单生成记录。
func (r *transcriptRecordRepository) GetByID(ctx context.Context, id int64) (*entity.TranscriptRecord, error) {
	var record entity.TranscriptRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// List 查询成绩单生成记录列表。
func (r *transcriptRecordRepository) List(ctx context.Context, params *TranscriptRecordListParams) ([]*entity.TranscriptRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.TranscriptRecord{}).
		Scopes(database.WithSchoolID(params.SchoolID))
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}
	if params.GeneratedBy > 0 {
		query = query.Where("generated_by = ?", params.GeneratedBy)
	}
	if params.From != nil {
		query = query.Where("generated_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("generated_at <= ?", *params.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "generated_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, map[string]string{
		"generated_at": "generated_at",
		"created_at":   "created_at",
		"file_size":    "file_size",
	})

	var records []*entity.TranscriptRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// ListExpired 查询已过期成绩单记录，供上层清理对象存储文件。
func (r *transcriptRecordRepository) ListExpired(ctx context.Context, before time.Time, limit int) ([]*entity.TranscriptRecord, error) {
	var records []*entity.TranscriptRecord
	err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", before).
		Order("expires_at asc").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// Delete 删除成绩单生成记录。
func (r *transcriptRecordRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.TranscriptRecord{}, id).Error
}
