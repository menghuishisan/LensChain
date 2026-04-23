// config_repo.go
// 模块08 — 系统管理与监控：系统配置与配置变更记录数据访问层。
// 负责 system_configs、config_change_logs 的查询、更新、批量读取和变更历史检索。

package systemrepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SystemConfigRepository 系统配置数据访问接口。
type SystemConfigRepository interface {
	GetByGroupAndKey(ctx context.Context, group, key string) (*entity.SystemConfig, error)
	List(ctx context.Context) ([]*entity.SystemConfig, error)
	ListByGroup(ctx context.Context, group string) ([]*entity.SystemConfig, error)
	Upsert(ctx context.Context, config *entity.SystemConfig) error
	UpdateValuesWithChangeLogs(ctx context.Context, updates []SystemConfigValueUpdate) (bool, error)
}

// SystemConfigValueUpdate 表示一次配置值更新及其审计日志写入请求。
type SystemConfigValueUpdate struct {
	Group             string
	Key               string
	Value             string
	UpdatedBy         *int64
	ExpectedUpdatedAt time.Time
	ChangeLog         *entity.ConfigChangeLog
}

type systemConfigRepository struct {
	db *gorm.DB
}

// NewSystemConfigRepository 创建系统配置数据访问实例。
func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
	return &systemConfigRepository{db: db}
}

// GetByGroupAndKey 获取单个系统配置。
func (r *systemConfigRepository) GetByGroupAndKey(ctx context.Context, group, key string) (*entity.SystemConfig, error) {
	var config entity.SystemConfig
	err := r.db.WithContext(ctx).Where("config_group = ? AND config_key = ?", group, key).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// List 查询全部系统配置。
func (r *systemConfigRepository) List(ctx context.Context) ([]*entity.SystemConfig, error) {
	var configs []*entity.SystemConfig
	err := r.db.WithContext(ctx).
		Order("config_group asc, config_key asc").
		Find(&configs).Error
	return configs, err
}

// ListByGroup 查询指定分组系统配置。
func (r *systemConfigRepository) ListByGroup(ctx context.Context, group string) ([]*entity.SystemConfig, error) {
	var configs []*entity.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_group = ?", group).
		Order("config_key asc").
		Find(&configs).Error
	return configs, err
}

// Upsert 保存系统配置。
func (r *systemConfigRepository) Upsert(ctx context.Context, config *entity.SystemConfig) error {
	if config.ID == 0 {
		config.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "config_group"}, {Name: "config_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"config_value",
			"value_type",
			"description",
			"is_sensitive",
			"updated_by",
			"updated_at",
		}),
	}).Create(config).Error
}

var errSystemConfigConflict = errors.New("system config optimistic lock conflict")

// UpdateValuesWithChangeLogs 在同一事务中批量更新配置值并写入变更日志。
func (r *systemConfigRepository) UpdateValuesWithChangeLogs(ctx context.Context, updates []SystemConfigValueUpdate) (bool, error) {
	if len(updates) == 0 {
		return true, nil
	}
	var updated bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range updates {
			result := tx.Model(&entity.SystemConfig{}).
				Where("config_group = ? AND config_key = ? AND updated_at = ?", item.Group, item.Key, item.ExpectedUpdatedAt.UTC()).
				Updates(map[string]interface{}{
					"config_value": item.Value,
					"updated_by":   item.UpdatedBy,
					"updated_at":   gorm.Expr("now()"),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				updated = false
				return errSystemConfigConflict
			}
			if item.ChangeLog != nil {
				if item.ChangeLog.ID == 0 {
					item.ChangeLog.ID = snowflake.Generate()
				}
				if err := tx.Create(item.ChangeLog).Error; err != nil {
					return err
				}
			}
		}
		updated = true
		return nil
	})
	if err != nil {
		if errors.Is(err, errSystemConfigConflict) {
			return false, nil
		}
		return false, err
	}
	return updated, nil
}

// ConfigChangeLogRepository 配置变更记录数据访问接口。
type ConfigChangeLogRepository interface {
	Create(ctx context.Context, log *entity.ConfigChangeLog) error
	List(ctx context.Context, params *ConfigChangeLogListParams) ([]*ConfigChangeLogItem, int64, error)
}

// ConfigChangeLogListParams 配置变更记录列表查询参数。
type ConfigChangeLogListParams struct {
	ConfigGroup string
	ConfigKey   string
	DateFrom    string
	DateTo      string
	Page        int
	PageSize    int
}

// ConfigChangeLogItem 配置变更记录列表项。
// 该结构在仓储层完成用户名称投影，供上层直接映射接口响应。
type ConfigChangeLogItem struct {
	ID            int64     `gorm:"column:id"`
	ConfigGroup   string    `gorm:"column:config_group"`
	ConfigKey     string    `gorm:"column:config_key"`
	OldValue      *string   `gorm:"column:old_value"`
	NewValue      string    `gorm:"column:new_value"`
	ChangedBy     int64     `gorm:"column:changed_by"`
	ChangedByName *string   `gorm:"column:changed_by_name"`
	ChangedAt     time.Time `gorm:"column:changed_at"`
	IP            string    `gorm:"column:ip"`
}

type configChangeLogRepository struct {
	db *gorm.DB
}

// NewConfigChangeLogRepository 创建配置变更记录数据访问实例。
func NewConfigChangeLogRepository(db *gorm.DB) ConfigChangeLogRepository {
	return &configChangeLogRepository{db: db}
}

// Create 创建配置变更记录。
func (r *configChangeLogRepository) Create(ctx context.Context, log *entity.ConfigChangeLog) error {
	if log.ID == 0 {
		log.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// List 查询配置变更记录列表。
func (r *configChangeLogRepository) List(ctx context.Context, params *ConfigChangeLogListParams) ([]*ConfigChangeLogItem, int64, error) {
	query := r.db.WithContext(ctx).Table("config_change_logs AS ccl").
		Joins("LEFT JOIN users u ON u.id = ccl.changed_by").
		Select(`
			ccl.id,
			ccl.config_group,
			ccl.config_key,
			ccl.old_value,
			ccl.new_value,
			ccl.changed_by,
			u.name AS changed_by_name,
			ccl.changed_at,
			ccl.ip
		`)
	if params.ConfigGroup != "" {
		query = query.Where("ccl.config_group = ?", params.ConfigGroup)
	}
	if params.ConfigKey != "" {
		query = query.Where("ccl.config_key = ?", params.ConfigKey)
	}
	if params.DateFrom != "" {
		query = query.Where("ccl.changed_at >= ?", params.DateFrom)
	}
	if params.DateTo != "" {
		query = query.Where("ccl.changed_at <= ?", params.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	var logs []*ConfigChangeLogItem
	err := query.Order("ccl.changed_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
