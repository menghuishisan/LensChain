// backup_repo.go
// 模块08 — 系统管理与监控：备份记录数据访问层。
// 负责 backup_records 的创建、列表、状态更新和按保留策略查询待清理备份。

package systemrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// BackupRecordRepository 备份记录数据访问接口。
type BackupRecordRepository interface {
	Create(ctx context.Context, record *entity.BackupRecord) error
	GetByID(ctx context.Context, id int64) (*entity.BackupRecord, error)
	List(ctx context.Context, params *BackupRecordListParams) ([]*entity.BackupRecord, int64, error)
	Update(ctx context.Context, id int64, values map[string]interface{}) error
	ListOldSuccessful(ctx context.Context, keepCount int) ([]*entity.BackupRecord, error)
}

// BackupRecordListParams 备份记录列表查询参数。
type BackupRecordListParams struct {
	Status   int16
	Page     int
	PageSize int
}

type backupRecordRepository struct {
	db *gorm.DB
}

// NewBackupRecordRepository 创建备份记录数据访问实例。
func NewBackupRecordRepository(db *gorm.DB) BackupRecordRepository {
	return &backupRecordRepository{db: db}
}

// Create 创建备份记录。
func (r *backupRecordRepository) Create(ctx context.Context, record *entity.BackupRecord) error {
	if record.ID == 0 {
		record.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByID 获取备份记录详情。
func (r *backupRecordRepository) GetByID(ctx context.Context, id int64) (*entity.BackupRecord, error) {
	var record entity.BackupRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// List 查询备份记录列表。
func (r *backupRecordRepository) List(ctx context.Context, params *BackupRecordListParams) ([]*entity.BackupRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.BackupRecord{}).Scopes(database.WithStatus(params.Status))
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var records []*entity.BackupRecord
	err := query.Order("started_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// Update 更新备份记录字段。
func (r *backupRecordRepository) Update(ctx context.Context, id int64, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.BackupRecord{}).Where("id = ?", id).Updates(values).Error
}

// ListOldSuccessful 查询超出保留份数的成功备份。
func (r *backupRecordRepository) ListOldSuccessful(ctx context.Context, keepCount int) ([]*entity.BackupRecord, error) {
	if keepCount <= 0 {
		keepCount = 30
	}
	sub := r.db.WithContext(ctx).Model(&entity.BackupRecord{}).
		Select("id").
		Where("status = ?", enum.BackupStatusSuccess).
		Order("started_at desc").
		Limit(keepCount)

	var records []*entity.BackupRecord
	err := r.db.WithContext(ctx).
		Where("status = ? AND id NOT IN (?)", enum.BackupStatusSuccess, sub).
		Order("started_at asc").
		Find(&records).Error
	return records, err
}
