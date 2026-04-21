// instance_sub_repo.go
// 模块04 — 实验环境：实例子资源数据访问层
// 负责检查点结果、快照、操作日志、实验报告的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 检查点结果 Repository
// ---------------------------------------------------------------------------

// CheckpointResultRepository 检查点结果数据访问接口
type CheckpointResultRepository interface {
	Create(ctx context.Context, result *entity.CheckpointResult) error
	GetByID(ctx context.Context, id int64) (*entity.CheckpointResult, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.CheckpointResult, error)
	ListByInstanceIDs(ctx context.Context, instanceIDs []int64) ([]*entity.CheckpointResult, error)
	GetByInstanceAndCheckpoint(ctx context.Context, instanceID, checkpointID int64) (*entity.CheckpointResult, error)
	CountPassedByInstanceID(ctx context.Context, instanceID int64) (int64, error)
	SumScoreByInstanceID(ctx context.Context, instanceID int64) (float64, error)
	ListByStudentAndTemplate(ctx context.Context, studentID, templateID int64) ([]*entity.CheckpointResult, error)
	ListCommonFailedByCourse(ctx context.Context, courseID int64, limit int) ([]*CommonCheckpointIssue, error)
}

// CommonCheckpointIssue 课程实验统计中的常见失败检查点聚合结果。
type CommonCheckpointIssue struct {
	TemplateID      int64  `gorm:"column:template_id"`
	CheckpointID    int64  `gorm:"column:checkpoint_id"`
	CheckpointTitle string `gorm:"column:checkpoint_title"`
	FailedCount     int64  `gorm:"column:failed_count"`
}

// checkpointResultRepository 检查点结果数据访问实现
type checkpointResultRepository struct {
	db *gorm.DB
}

// NewCheckpointResultRepository 创建检查点结果数据访问实例
func NewCheckpointResultRepository(db *gorm.DB) CheckpointResultRepository {
	return &checkpointResultRepository{db: db}
}

// Create 创建检查点结果
func (r *checkpointResultRepository) Create(ctx context.Context, result *entity.CheckpointResult) error {
	if result.ID == 0 {
		result.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(result).Error
}

// GetByID 根据ID获取检查点结果
func (r *checkpointResultRepository) GetByID(ctx context.Context, id int64) (*entity.CheckpointResult, error) {
	var result entity.CheckpointResult
	err := r.db.WithContext(ctx).First(&result, id).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateFields 更新检查点结果指定字段
func (r *checkpointResultRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CheckpointResult{}).Where("id = ?", id).Updates(fields).Error
}

// ListByInstanceID 获取实例的所有检查点结果
func (r *checkpointResultRepository) ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.CheckpointResult, error) {
	var results []*entity.CheckpointResult
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("checkpoint_id asc").
		Find(&results).Error
	return results, err
}

// ListByInstanceIDs 批量获取多个实例的检查点结果。
func (r *checkpointResultRepository) ListByInstanceIDs(ctx context.Context, instanceIDs []int64) ([]*entity.CheckpointResult, error) {
	if len(instanceIDs) == 0 {
		return []*entity.CheckpointResult{}, nil
	}
	var results []*entity.CheckpointResult
	err := r.db.WithContext(ctx).
		Where("instance_id IN ?", instanceIDs).
		Order("instance_id asc, checkpoint_id asc").
		Find(&results).Error
	return results, err
}

// GetByInstanceAndCheckpoint 获取实例某检查点的结果
func (r *checkpointResultRepository) GetByInstanceAndCheckpoint(ctx context.Context, instanceID, checkpointID int64) (*entity.CheckpointResult, error) {
	var result entity.CheckpointResult
	err := r.db.WithContext(ctx).
		Where("instance_id = ? AND checkpoint_id = ?", instanceID, checkpointID).
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CountPassedByInstanceID 统计实例已通过的检查点数
func (r *checkpointResultRepository) CountPassedByInstanceID(ctx context.Context, instanceID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CheckpointResult{}).
		Where("instance_id = ? AND is_passed = true", instanceID).
		Count(&count).Error
	return count, err
}

// SumScoreByInstanceID 统计实例检查点总得分
func (r *checkpointResultRepository) SumScoreByInstanceID(ctx context.Context, instanceID int64) (float64, error) {
	var sum float64
	err := r.db.WithContext(ctx).Model(&entity.CheckpointResult{}).
		Where("instance_id = ?", instanceID).
		Select("COALESCE(SUM(score), 0)").
		Scan(&sum).Error
	return sum, err
}

// ListByStudentAndTemplate 获取学生在某模板下的所有检查点结果
func (r *checkpointResultRepository) ListByStudentAndTemplate(ctx context.Context, studentID, templateID int64) ([]*entity.CheckpointResult, error) {
	var results []*entity.CheckpointResult
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND instance_id IN (?)",
			studentID,
			r.db.Model(&entity.ExperimentInstance{}).
				Select("id").
				Where("template_id = ?", templateID),
		).
		Order("checkpoint_id asc").
		Find(&results).Error
	return results, err
}

// ListCommonFailedByCourse 聚合课程下失败次数最多的检查点，用于实验统计“常见错误检查点”。
func (r *checkpointResultRepository) ListCommonFailedByCourse(ctx context.Context, courseID int64, limit int) ([]*CommonCheckpointIssue, error) {
	var issues []*CommonCheckpointIssue
	query := r.db.WithContext(ctx).Model(&entity.CheckpointResult{}).
		Select(`
			experiment_instances.template_id AS template_id,
			checkpoint_results.checkpoint_id AS checkpoint_id,
			template_checkpoints.title AS checkpoint_title,
			COUNT(*) AS failed_count
		`).
		Joins("JOIN experiment_instances ON experiment_instances.id = checkpoint_results.instance_id").
		Joins("JOIN template_checkpoints ON template_checkpoints.id = checkpoint_results.checkpoint_id").
		Where("experiment_instances.course_id = ?", courseID).
		Where("checkpoint_results.is_passed = false").
		Group("experiment_instances.template_id, checkpoint_results.checkpoint_id, template_checkpoints.title").
		Order("failed_count desc, checkpoint_results.checkpoint_id asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&issues).Error
	return issues, err
}

// ---------------------------------------------------------------------------
// 实例快照 Repository
// ---------------------------------------------------------------------------

// SnapshotRepository 实例快照数据访问接口
type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *entity.InstanceSnapshot) error
	GetByID(ctx context.Context, id int64) (*entity.InstanceSnapshot, error)
	GetLatestByInstanceID(ctx context.Context, instanceID int64) (*entity.InstanceSnapshot, error)
	ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceSnapshot, error)
	ListByInstanceIDs(ctx context.Context, instanceIDs []int64) ([]*entity.InstanceSnapshot, error)
	Delete(ctx context.Context, id int64) error
	DeleteByInstanceID(ctx context.Context, instanceID int64) error
	DeleteOldByInstanceID(ctx context.Context, instanceID int64, keepCount int) error
	CountByInstanceID(ctx context.Context, instanceID int64) (int64, error)
}

// snapshotRepository 实例快照数据访问实现
type snapshotRepository struct {
	db *gorm.DB
}

// NewSnapshotRepository 创建实例快照数据访问实例
func NewSnapshotRepository(db *gorm.DB) SnapshotRepository {
	return &snapshotRepository{db: db}
}

// Create 创建实例快照
func (r *snapshotRepository) Create(ctx context.Context, snapshot *entity.InstanceSnapshot) error {
	if snapshot.ID == 0 {
		snapshot.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(snapshot).Error
}

// GetByID 根据ID获取实例快照
func (r *snapshotRepository) GetByID(ctx context.Context, id int64) (*entity.InstanceSnapshot, error) {
	var snapshot entity.InstanceSnapshot
	err := r.db.WithContext(ctx).First(&snapshot, id).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// GetLatestByInstanceID 获取实例最新快照，供恢复接口未指定 snapshot_id 时使用。
func (r *snapshotRepository) GetLatestByInstanceID(ctx context.Context, instanceID int64) (*entity.InstanceSnapshot, error) {
	var snapshot entity.InstanceSnapshot
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at desc").
		First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// ListByInstanceID 获取实例的所有快照
func (r *snapshotRepository) ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceSnapshot, error) {
	var snapshots []*entity.InstanceSnapshot
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at desc").
		Find(&snapshots).Error
	return snapshots, err
}

// ListByInstanceIDs 批量获取多个实例的快照列表。
func (r *snapshotRepository) ListByInstanceIDs(ctx context.Context, instanceIDs []int64) ([]*entity.InstanceSnapshot, error) {
	if len(instanceIDs) == 0 {
		return []*entity.InstanceSnapshot{}, nil
	}
	var snapshots []*entity.InstanceSnapshot
	err := r.db.WithContext(ctx).
		Where("instance_id IN ?", instanceIDs).
		Order("instance_id asc, created_at desc").
		Find(&snapshots).Error
	return snapshots, err
}

// Delete 删除快照
func (r *snapshotRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.InstanceSnapshot{}, id).Error
}

// DeleteByInstanceID 删除实例的全部快照。
func (r *snapshotRepository) DeleteByInstanceID(ctx context.Context, instanceID int64) error {
	return r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Delete(&entity.InstanceSnapshot{}).Error
}

// DeleteOldByInstanceID 删除实例的旧快照（保留最新 keepCount 条）
func (r *snapshotRepository) DeleteOldByInstanceID(ctx context.Context, instanceID int64, keepCount int) error {
	// 子查询获取需要保留的快照ID
	subQuery := r.db.Model(&entity.InstanceSnapshot{}).
		Select("id").
		Where("instance_id = ?", instanceID).
		Order("created_at desc").
		Limit(keepCount)

	return r.db.WithContext(ctx).
		Where("instance_id = ? AND id NOT IN (?)", instanceID, subQuery).
		Delete(&entity.InstanceSnapshot{}).Error
}

// CountByInstanceID 统计实例快照数量
func (r *snapshotRepository) CountByInstanceID(ctx context.Context, instanceID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.InstanceSnapshot{}).
		Where("instance_id = ?", instanceID).
		Count(&count).Error
	return count, err
}

// ---------------------------------------------------------------------------
// 实例操作日志 Repository
// ---------------------------------------------------------------------------

// OperationLogRepository 实例操作日志数据访问接口
type OperationLogRepository interface {
	Create(ctx context.Context, log *entity.InstanceOperationLog) error
	List(ctx context.Context, params *OperationLogListParams) ([]*entity.InstanceOperationLog, int64, error)
	ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceOperationLog, error)
}

// OperationLogListParams 操作日志列表查询参数
type OperationLogListParams struct {
	InstanceID      int64
	StudentID       int64
	Action          string
	TargetContainer string
	DateFrom        string
	DateTo          string
	SortBy          string
	SortOrder       string
	Page            int
	PageSize        int
}

// operationLogRepository 实例操作日志数据访问实现
type operationLogRepository struct {
	db *gorm.DB
}

// NewOperationLogRepository 创建实例操作日志数据访问实例
func NewOperationLogRepository(db *gorm.DB) OperationLogRepository {
	return &operationLogRepository{db: db}
}

// Create 创建操作日志（只插入不更新不删除）
func (r *operationLogRepository) Create(ctx context.Context, log *entity.InstanceOperationLog) error {
	if log.ID == 0 {
		log.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// List 操作日志列表查询
func (r *operationLogRepository) List(ctx context.Context, params *OperationLogListParams) ([]*entity.InstanceOperationLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.InstanceOperationLog{})

	if params.InstanceID > 0 {
		query = query.Where("instance_id = ?", params.InstanceID)
	}
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}
	if params.TargetContainer != "" {
		query = query.Where("target_container = ?", params.TargetContainer)
	}
	if params.DateFrom != "" || params.DateTo != "" {
		query = query.Scopes(database.WithDateRange("created_at", params.DateFrom, params.DateTo))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"action":     "action",
	}
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, allowedSortFields)

	var logs []*entity.InstanceOperationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ListByInstanceID 获取实例的所有操作日志
func (r *operationLogRepository) ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceOperationLog, error) {
	var logs []*entity.InstanceOperationLog
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at desc").
		Find(&logs).Error
	return logs, err
}

// ---------------------------------------------------------------------------
// 实验报告 Repository
// ---------------------------------------------------------------------------

// ReportRepository 实验报告数据访问接口
type ReportRepository interface {
	Create(ctx context.Context, report *entity.ExperimentReport) error
	GetByID(ctx context.Context, id int64) (*entity.ExperimentReport, error)
	GetByInstanceID(ctx context.Context, instanceID int64) (*entity.ExperimentReport, error)
	GetByInstanceAndStudent(ctx context.Context, instanceID, studentID int64) (*entity.ExperimentReport, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
}

// reportRepository 实验报告数据访问实现
type reportRepository struct {
	db *gorm.DB
}

// NewReportRepository 创建实验报告数据访问实例
func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

// Create 创建实验报告
func (r *reportRepository) Create(ctx context.Context, report *entity.ExperimentReport) error {
	if report.ID == 0 {
		report.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(report).Error
}

// GetByID 根据ID获取实验报告
func (r *reportRepository) GetByID(ctx context.Context, id int64) (*entity.ExperimentReport, error) {
	var report entity.ExperimentReport
	err := r.db.WithContext(ctx).First(&report, id).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// GetByInstanceID 根据实例ID获取实验报告
func (r *reportRepository) GetByInstanceID(ctx context.Context, instanceID int64) (*entity.ExperimentReport, error) {
	var report entity.ExperimentReport
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// GetByInstanceAndStudent 根据实例ID和学生ID获取实验报告
func (r *reportRepository) GetByInstanceAndStudent(ctx context.Context, instanceID, studentID int64) (*entity.ExperimentReport, error) {
	var report entity.ExperimentReport
	err := r.db.WithContext(ctx).
		Where("instance_id = ? AND student_id = ?", instanceID, studentID).
		First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// UpdateFields 更新实验报告指定字段
func (r *reportRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ExperimentReport{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除实验报告
func (r *reportRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ExperimentReport{}, id).Error
}
