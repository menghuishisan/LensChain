// log_repo.go
// 模块01 — 用户与认证：日志数据访问层
// 负责 login_logs 和 operation_logs 表的操作
// 审计日志红线：只插入，不更新，不删除

package authrepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// LoginLogRepository 登录日志数据访问接口
type LoginLogRepository interface {
	Create(ctx context.Context, log *entity.LoginLog) error
	List(ctx context.Context, params *LoginLogListParams) ([]*entity.LoginLog, int64, error)
}

// LoginLogListParams 登录日志列表查询参数
type LoginLogListParams struct {
	SchoolID    int64
	UserID      int64
	Action      int
	CreatedFrom string
	CreatedTo   string
	Page        int
	PageSize    int
}

// loginLogRepository 登录日志数据访问实现
type loginLogRepository struct {
	db *gorm.DB
}

// NewLoginLogRepository 创建登录日志数据访问实例
func NewLoginLogRepository(db *gorm.DB) LoginLogRepository {
	return &loginLogRepository{db: db}
}

// Create 创建登录日志
func (r *loginLogRepository) Create(ctx context.Context, log *entity.LoginLog) error {
	if log.ID == 0 {
		log.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// List 登录日志列表查询
func (r *loginLogRepository) List(ctx context.Context, params *LoginLogListParams) ([]*entity.LoginLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.LoginLog{})

	// 多租户过滤：通过 users 表关联过滤学校
	if params.SchoolID > 0 {
		query = query.Where("user_id IN (?)",
			r.db.Model(&entity.User{}).Select("id").Where("school_id = ?", params.SchoolID),
		)
	}

	// 用户ID过滤
	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}

	// 操作类型过滤
	if params.Action > 0 {
		query = query.Where("action = ?", params.Action)
	}

	// 时间范围过滤
	if params.CreatedFrom != "" {
		query = query.Where("created_at >= ?", params.CreatedFrom)
	}
	if params.CreatedTo != "" {
		query = query.Where("created_at <= ?", params.CreatedTo)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	offset := (params.Page - 1) * params.PageSize

	var logs []*entity.LoginLog
	err := query.Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// OperationLogRepository 操作日志数据访问接口
type OperationLogRepository interface {
	Create(ctx context.Context, log *entity.OperationLog) error
	List(ctx context.Context, params *OperationLogListParams) ([]*entity.OperationLog, int64, error)
}

// OperationLogListParams 操作日志列表查询参数
type OperationLogListParams struct {
	SchoolID    int64
	OperatorID  int64
	Action      string
	TargetType  string
	CreatedFrom string
	CreatedTo   string
	Page        int
	PageSize    int
}

// operationLogRepository 操作日志数据访问实现
type operationLogRepository struct {
	db *gorm.DB
}

// NewOperationLogRepository 创建操作日志数据访问实例
func NewOperationLogRepository(db *gorm.DB) OperationLogRepository {
	return &operationLogRepository{db: db}
}

// Create 创建操作日志
func (r *operationLogRepository) Create(ctx context.Context, log *entity.OperationLog) error {
	if log.ID == 0 {
		log.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// List 操作日志列表查询
func (r *operationLogRepository) List(ctx context.Context, params *OperationLogListParams) ([]*entity.OperationLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.OperationLog{})

	// 多租户过滤：通过 users 表关联过滤学校
	if params.SchoolID > 0 {
		query = query.Where("operator_id IN (?)",
			r.db.Model(&entity.User{}).Select("id").Where("school_id = ?", params.SchoolID),
		)
	}

	// 操作人过滤
	if params.OperatorID > 0 {
		query = query.Where("operator_id = ?", params.OperatorID)
	}

	// 操作类型过滤
	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}

	// 操作对象类型过滤
	if params.TargetType != "" {
		query = query.Where("target_type = ?", params.TargetType)
	}

	// 时间范围过滤
	if params.CreatedFrom != "" {
		query = query.Where("created_at >= ?", params.CreatedFrom)
	}
	if params.CreatedTo != "" {
		query = query.Where("created_at <= ?", params.CreatedTo)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计操作日志总数失败: %w", err)
	}

	// 分页
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	offset := (params.Page - 1) * params.PageSize

	var logs []*entity.OperationLog
	err := query.Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
