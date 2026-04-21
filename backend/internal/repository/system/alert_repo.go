// alert_repo.go
// 模块08 — 系统管理与监控：告警规则与告警事件数据访问层。
// 负责 alert_rules、alert_events 的 CRUD、列表、状态流转和静默期查询。

package systemrepo

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

// AlertRuleRepository 告警规则数据访问接口。
type AlertRuleRepository interface {
	Create(ctx context.Context, rule *entity.AlertRule) error
	GetByID(ctx context.Context, id int64) (*entity.AlertRule, error)
	List(ctx context.Context, params *AlertRuleListParams) ([]*entity.AlertRule, int64, error)
	ListEnabled(ctx context.Context) ([]*entity.AlertRule, error)
	Update(ctx context.Context, id int64, values map[string]interface{}) error
	Toggle(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
}

// AlertRuleListParams 告警规则列表查询参数。
type AlertRuleListParams struct {
	AlertType int16
	Level     int16
	IsEnabled *bool
	Page      int
	PageSize  int
}

type alertRuleRepository struct {
	db *gorm.DB
}

// NewAlertRuleRepository 创建告警规则数据访问实例。
func NewAlertRuleRepository(db *gorm.DB) AlertRuleRepository {
	return &alertRuleRepository{db: db}
}

// Create 创建告警规则。
func (r *alertRuleRepository) Create(ctx context.Context, rule *entity.AlertRule) error {
	if rule.ID == 0 {
		rule.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 获取告警规则详情。
func (r *alertRuleRepository) GetByID(ctx context.Context, id int64) (*entity.AlertRule, error) {
	var rule entity.AlertRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List 查询告警规则列表。
func (r *alertRuleRepository) List(ctx context.Context, params *AlertRuleListParams) ([]*entity.AlertRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AlertRule{})
	if params.AlertType > 0 {
		query = query.Where("alert_type = ?", params.AlertType)
	}
	if params.Level > 0 {
		query = query.Where("level = ?", params.Level)
	}
	if params.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *params.IsEnabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var rules []*entity.AlertRule
	err := query.Order("created_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&rules).Error
	if err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

// ListEnabled 查询启用中的告警规则。
func (r *alertRuleRepository) ListEnabled(ctx context.Context) ([]*entity.AlertRule, error) {
	var rules []*entity.AlertRule
	err := r.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("level desc, created_at asc").
		Find(&rules).Error
	return rules, err
}

// Update 更新告警规则。
func (r *alertRuleRepository) Update(ctx context.Context, id int64, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AlertRule{}).Where("id = ?", id).Updates(values).Error
}

// Toggle 启用/禁用告警规则。
func (r *alertRuleRepository) Toggle(ctx context.Context, id int64, enabled bool) error {
	return r.db.WithContext(ctx).Model(&entity.AlertRule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_enabled": enabled,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

// Delete 软删除告警规则。
func (r *alertRuleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.AlertRule{}, id).Error
}

// AlertEventRepository 告警事件数据访问接口。
type AlertEventRepository interface {
	Create(ctx context.Context, event *entity.AlertEvent) error
	GetByID(ctx context.Context, id int64) (*entity.AlertEvent, error)
	List(ctx context.Context, params *AlertEventListParams) ([]*entity.AlertEvent, int64, error)
	GetLatestPendingByRule(ctx context.Context, ruleID int64) (*entity.AlertEvent, error)
	Handle(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error
	Ignore(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error
	StatusCounts(ctx context.Context, params *AlertEventListParams) (*AlertEventStatusCounts, error)
	ListRecentPending(ctx context.Context, limit int) ([]*entity.AlertEvent, error)
}

// AlertEventListParams 告警事件列表查询参数。
type AlertEventListParams struct {
	RuleID   int64
	Level    int16
	Status   int16
	DateFrom string
	DateTo   string
	Page     int
	PageSize int
}

// AlertEventStatusCounts 告警状态统计。
type AlertEventStatusCounts struct {
	Pending int64 `gorm:"column:pending"`
	Handled int64 `gorm:"column:handled"`
	Ignored int64 `gorm:"column:ignored"`
}

type alertEventRepository struct {
	db *gorm.DB
}

// NewAlertEventRepository 创建告警事件数据访问实例。
func NewAlertEventRepository(db *gorm.DB) AlertEventRepository {
	return &alertEventRepository{db: db}
}

// Create 创建告警事件。
func (r *alertEventRepository) Create(ctx context.Context, event *entity.AlertEvent) error {
	if event.ID == 0 {
		event.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(event).Error
}

// GetByID 获取告警事件详情。
func (r *alertEventRepository) GetByID(ctx context.Context, id int64) (*entity.AlertEvent, error) {
	var event entity.AlertEvent
	err := r.db.WithContext(ctx).First(&event, id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// List 查询告警事件列表。
func (r *alertEventRepository) List(ctx context.Context, params *AlertEventListParams) ([]*entity.AlertEvent, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AlertEvent{}).Scopes(database.WithStatus(params.Status))
	if params.RuleID > 0 {
		query = query.Where("rule_id = ?", params.RuleID)
	}
	if params.Level > 0 {
		query = query.Where("level = ?", params.Level)
	}
	query = query.Scopes(database.WithDateRange("triggered_at", params.DateFrom, params.DateTo))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var events []*entity.AlertEvent
	err := query.Order("triggered_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

// GetLatestPendingByRule 获取规则最近一条未处理告警。
func (r *alertEventRepository) GetLatestPendingByRule(ctx context.Context, ruleID int64) (*entity.AlertEvent, error) {
	var event entity.AlertEvent
	err := r.db.WithContext(ctx).
		Where("rule_id = ? AND status = ?", ruleID, enum.AlertEventStatusPending).
		Order("triggered_at desc").
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// Handle 标记告警为已处理。
func (r *alertEventRepository) Handle(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.AlertEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      enum.AlertEventStatusHandled,
			"handled_by":  handledBy,
			"handled_at":  handledAt,
			"handle_note": note,
		}).Error
}

// Ignore 标记告警为已忽略。
func (r *alertEventRepository) Ignore(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.AlertEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      enum.AlertEventStatusIgnored,
			"handled_by":  handledBy,
			"handled_at":  handledAt,
			"handle_note": note,
		}).Error
}

// StatusCounts 查询告警状态统计。
func (r *alertEventRepository) StatusCounts(ctx context.Context, params *AlertEventListParams) (*AlertEventStatusCounts, error) {
	query := r.db.WithContext(ctx).Model(&entity.AlertEvent{})
	if params.RuleID > 0 {
		query = query.Where("rule_id = ?", params.RuleID)
	}
	if params.Level > 0 {
		query = query.Where("level = ?", params.Level)
	}
	query = query.Scopes(database.WithDateRange("triggered_at", params.DateFrom, params.DateTo))
	var counts AlertEventStatusCounts
	err := query.Select(`
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS pending,
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS handled,
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS ignored
	`,
		enum.AlertEventStatusPending,
		enum.AlertEventStatusHandled,
		enum.AlertEventStatusIgnored,
	).Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	return &counts, nil
}

// ListRecentPending 查询最近待处理告警。
func (r *alertEventRepository) ListRecentPending(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", enum.AlertEventStatusPending).
		Order("triggered_at desc").
		Limit(limit).
		Find(&events).Error
	return events, err
}
