// instance_repo.go
// 模块04 — 实验环境：实验实例数据访问层
// 负责实验实例、实例容器的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 实验实例 Repository
// ---------------------------------------------------------------------------

// InstanceRepository 实验实例数据访问接口
type InstanceRepository interface {
	Create(ctx context.Context, instance *entity.ExperimentInstance) error
	GetByID(ctx context.Context, id int64) (*entity.ExperimentInstance, error)
	GetByIDWithAll(ctx context.Context, id int64) (*entity.ExperimentInstance, error)
	GetBySimSessionID(ctx context.Context, sessionID string) (*entity.ExperimentInstance, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	List(ctx context.Context, params *InstanceListParams) ([]*entity.ExperimentInstance, int64, error)
	ListByStudentID(ctx context.Context, studentID int64, params *StudentInstanceListParams) ([]*entity.ExperimentInstance, int64, error)
	ListByTemplateAndStudent(ctx context.Context, templateID, studentID int64) ([]*entity.ExperimentInstance, error)
	CountRunningByStudent(ctx context.Context, studentID int64) (int64, error)
	CountRunningBySchool(ctx context.Context, schoolID int64) (int64, error)
	CountRunningByCourse(ctx context.Context, courseID int64) (int64, error)
	CountByTemplateID(ctx context.Context, templateID int64) (int64, error)
	GetMaxAttemptNo(ctx context.Context, templateID, studentID int64) (int, error)
	ListIdleInstances(ctx context.Context, idleSince time.Time) ([]*entity.ExperimentInstance, error)
	ListExpiredInstances(ctx context.Context, now time.Time) ([]*entity.ExperimentInstance, error)
	ListByGroupID(ctx context.Context, groupID int64) ([]*entity.ExperimentInstance, error)
	UpdateLastActiveAt(ctx context.Context, id int64, t time.Time) error

	// 管理员视角
	ListAdmin(ctx context.Context, params *AdminInstanceListParams) ([]*entity.ExperimentInstance, int64, error)

	// 统计
	CountByStatus(ctx context.Context, schoolID int64) (map[int]int64, error)
	CountByTemplateAndStatus(ctx context.Context, templateID int64) (map[int]int64, error)
}

// InstanceListParams 实例列表查询参数（教师视角）
type InstanceListParams struct {
	SchoolID   int64
	TemplateID int64
	CourseID   int64
	StudentID  int64
	Status     int
	Statuses   []int
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// StudentInstanceListParams 学生实例列表查询参数
type StudentInstanceListParams struct {
	TemplateID int64
	CourseID   int64
	Status     int
	Page       int
	PageSize   int
}

// AdminInstanceListParams 管理员实例列表查询参数
type AdminInstanceListParams struct {
	SchoolID   int64
	TemplateID int64
	Status     int
	Keyword    string
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// instanceRepository 实验实例数据访问实现
type instanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository 创建实验实例数据访问实例
func NewInstanceRepository(db *gorm.DB) InstanceRepository {
	return &instanceRepository{db: db}
}

// Create 创建实验实例
func (r *instanceRepository) Create(ctx context.Context, instance *entity.ExperimentInstance) error {
	if instance.ID == 0 {
		instance.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(instance).Error
}

// GetByID 根据ID获取实验实例（不含关联）
func (r *instanceRepository) GetByID(ctx context.Context, id int64) (*entity.ExperimentInstance, error) {
	var instance entity.ExperimentInstance
	err := r.db.WithContext(ctx).First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByIDWithAll 根据ID获取实验实例（含容器、检查点结果、快照）
func (r *instanceRepository) GetByIDWithAll(ctx context.Context, id int64) (*entity.ExperimentInstance, error) {
	var instance entity.ExperimentInstance
	err := r.db.WithContext(ctx).
		Preload("Containers").
		Preload("CheckpointResults").
		Preload("Snapshots", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at desc")
		}).
		First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetBySimSessionID 根据仿真会话ID获取实验实例。
func (r *instanceRepository) GetBySimSessionID(ctx context.Context, sessionID string) (*entity.ExperimentInstance, error) {
	var instance entity.ExperimentInstance
	err := r.db.WithContext(ctx).
		Where("sim_session_id = ?", sessionID).
		First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// UpdateFields 更新实验实例指定字段
func (r *instanceRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).Where("id = ?", id).Updates(fields).Error
}

// List 教师视角实例列表查询
func (r *instanceRepository) List(ctx context.Context, params *InstanceListParams) ([]*entity.ExperimentInstance, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{})

	// 多租户隔离
	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}

	// 模板筛选
	if params.TemplateID > 0 {
		query = query.Where("template_id = ?", params.TemplateID)
	}

	// 课程筛选
	if params.CourseID > 0 {
		query = query.Where("course_id = ?", params.CourseID)
	}

	// 学生筛选
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}

	// 状态筛选
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	} else if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at":   "created_at",
		"status":       "status",
		"total_score":  "total_score",
		"submitted_at": "submitted_at",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// 分页
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var instances []*entity.ExperimentInstance
	if err := query.Find(&instances).Error; err != nil {
		return nil, 0, err
	}
	return instances, total, nil
}

// ListByStudentID 学生实例列表查询
func (r *instanceRepository) ListByStudentID(ctx context.Context, studentID int64, params *StudentInstanceListParams) ([]*entity.ExperimentInstance, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("student_id = ?", studentID)

	if params.TemplateID > 0 {
		query = query.Where("template_id = ?", params.TemplateID)
	}
	if params.CourseID > 0 {
		query = query.Where("course_id = ?", params.CourseID)
	}
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var instances []*entity.ExperimentInstance
	if err := query.Find(&instances).Error; err != nil {
		return nil, 0, err
	}
	return instances, total, nil
}

// ListByTemplateAndStudent 获取学生在某模板下的所有实例
func (r *instanceRepository) ListByTemplateAndStudent(ctx context.Context, templateID, studentID int64) ([]*entity.ExperimentInstance, error) {
	var instances []*entity.ExperimentInstance
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND student_id = ?", templateID, studentID).
		Order("attempt_no desc").
		Find(&instances).Error
	return instances, err
}

// CountRunningByStudent 统计学生正在运行的实例数
func (r *instanceRepository) CountRunningByStudent(ctx context.Context, studentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("student_id = ? AND status IN ?", studentID, []int{1, 2, 8, 9}).
		Count(&count).Error
	return count, err
}

// CountRunningBySchool 统计学校正在运行的实例数
func (r *instanceRepository) CountRunningBySchool(ctx context.Context, schoolID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("school_id = ? AND status IN ?", schoolID, []int{1, 2, 8, 9}).
		Count(&count).Error
	return count, err
}

// CountRunningByCourse 统计课程正在运行的实例数
func (r *instanceRepository) CountRunningByCourse(ctx context.Context, courseID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("course_id = ? AND status IN ?", courseID, []int{1, 2, 8, 9}).
		Count(&count).Error
	return count, err
}

// CountByTemplateID 统计模板下的实例数
func (r *instanceRepository) CountByTemplateID(ctx context.Context, templateID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("template_id = ?", templateID).
		Count(&count).Error
	return count, err
}

// GetMaxAttemptNo 获取学生在某模板下的最大尝试次数
func (r *instanceRepository) GetMaxAttemptNo(ctx context.Context, templateID, studentID int64) (int, error) {
	var max int
	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("template_id = ? AND student_id = ?", templateID, studentID).
		Select("COALESCE(MAX(attempt_no), 0)").
		Scan(&max).Error
	return max, err
}

// ListIdleInstances 查询空闲超时的实例
func (r *instanceRepository) ListIdleInstances(ctx context.Context, idleSince time.Time) ([]*entity.ExperimentInstance, error) {
	var instances []*entity.ExperimentInstance
	err := r.db.WithContext(ctx).
		Where("status = 2 AND last_active_at IS NOT NULL AND last_active_at < ?", idleSince).
		Find(&instances).Error
	return instances, err
}

// ListExpiredInstances 查询超过最大时长的实例
func (r *instanceRepository) ListExpiredInstances(ctx context.Context, now time.Time) ([]*entity.ExperimentInstance, error) {
	var instances []*entity.ExperimentInstance
	// 通过 JOIN 模板获取 max_duration，筛选超时实例
	err := r.db.WithContext(ctx).
		Joins("JOIN experiment_templates ON experiment_templates.id = experiment_instances.template_id").
		Where("experiment_instances.status = 2").
		Where("experiment_templates.max_duration IS NOT NULL").
		Where("experiment_instances.started_at IS NOT NULL").
		Where("experiment_instances.started_at + (experiment_templates.max_duration || ' minutes')::interval < ?", now).
		Find(&instances).Error
	return instances, err
}

// ListByGroupID 获取分组下的所有实例
func (r *instanceRepository) ListByGroupID(ctx context.Context, groupID int64) ([]*entity.ExperimentInstance, error) {
	var instances []*entity.ExperimentInstance
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("created_at asc").
		Find(&instances).Error
	return instances, err
}

// UpdateLastActiveAt 更新实例最后活跃时间
func (r *instanceRepository) UpdateLastActiveAt(ctx context.Context, id int64, t time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Where("id = ?", id).
		Update("last_active_at", t).Error
}

// ListAdmin 管理员视角实例列表查询
func (r *instanceRepository) ListAdmin(ctx context.Context, params *AdminInstanceListParams) ([]*entity.ExperimentInstance, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{})

	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}
	if params.TemplateID > 0 {
		query = query.Where("template_id = ?", params.TemplateID)
	}
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"status":     "status",
		"school_id":  "school_id",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var instances []*entity.ExperimentInstance
	if err := query.Find(&instances).Error; err != nil {
		return nil, 0, err
	}
	return instances, total, nil
}

// CountByStatus 按状态统计实例数量
func (r *instanceRepository) CountByStatus(ctx context.Context, schoolID int64) (map[int]int64, error) {
	type result struct {
		Status int   `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}
	var results []result

	query := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Select("status, COUNT(*) as count").
		Group("status")

	if schoolID > 0 {
		query = query.Where("school_id = ?", schoolID)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	m := make(map[int]int64)
	for _, r := range results {
		m[r.Status] = r.Count
	}
	return m, nil
}

// CountByTemplateAndStatus 按模板和状态统计实例数量
func (r *instanceRepository) CountByTemplateAndStatus(ctx context.Context, templateID int64) (map[int]int64, error) {
	type result struct {
		Status int   `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}
	var results []result

	err := r.db.WithContext(ctx).Model(&entity.ExperimentInstance{}).
		Select("status, COUNT(*) as count").
		Where("template_id = ?", templateID).
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	m := make(map[int]int64)
	for _, r := range results {
		m[r.Status] = r.Count
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// 实例容器 Repository
// ---------------------------------------------------------------------------

// InstanceContainerRepository 实例容器数据访问接口
type InstanceContainerRepository interface {
	Create(ctx context.Context, container *entity.InstanceContainer) error
	GetByID(ctx context.Context, id int64) (*entity.InstanceContainer, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceContainer, error)
	BatchCreate(ctx context.Context, containers []*entity.InstanceContainer) error
	DeleteByInstanceID(ctx context.Context, instanceID int64) error
}

// instanceContainerRepository 实例容器数据访问实现
type instanceContainerRepository struct {
	db *gorm.DB
}

// NewInstanceContainerRepository 创建实例容器数据访问实例
func NewInstanceContainerRepository(db *gorm.DB) InstanceContainerRepository {
	return &instanceContainerRepository{db: db}
}

// Create 创建实例容器
func (r *instanceContainerRepository) Create(ctx context.Context, container *entity.InstanceContainer) error {
	if container.ID == 0 {
		container.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(container).Error
}

// GetByID 根据ID获取实例容器
func (r *instanceContainerRepository) GetByID(ctx context.Context, id int64) (*entity.InstanceContainer, error) {
	var container entity.InstanceContainer
	err := r.db.WithContext(ctx).First(&container, id).Error
	if err != nil {
		return nil, err
	}
	return &container, nil
}

// UpdateFields 更新实例容器指定字段
func (r *instanceContainerRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.InstanceContainer{}).Where("id = ?", id).Updates(fields).Error
}

// ListByInstanceID 获取实例的所有容器
func (r *instanceContainerRepository) ListByInstanceID(ctx context.Context, instanceID int64) ([]*entity.InstanceContainer, error) {
	var containers []*entity.InstanceContainer
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Find(&containers).Error
	return containers, err
}

// BatchCreate 批量创建实例容器
func (r *instanceContainerRepository) BatchCreate(ctx context.Context, containers []*entity.InstanceContainer) error {
	for i := range containers {
		if containers[i].ID == 0 {
			containers[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(containers, 50).Error
}

// DeleteByInstanceID 删除实例的所有容器
func (r *instanceContainerRepository) DeleteByInstanceID(ctx context.Context, instanceID int64) error {
	return r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Delete(&entity.InstanceContainer{}).Error
}
