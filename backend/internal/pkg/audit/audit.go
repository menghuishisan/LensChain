// audit.go
// 该文件提供平台统一的操作审计写入能力，负责把用户操作、目标资源、来源 IP 和变更详情
// 组织成标准审计记录并写入 `operation_logs`。它的定位是 service 层可直接复用的基础工具，
// 用来保证所有模块的审计格式一致、字段口径一致，避免各模块各自拼装日志导致统一审计中心
// 无法聚合查询。这里约束的是“如何记录审计”，不承担业务动作本身的判断与编排。

package audit

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// OperationLog 操作日志结构体
// 严格对照 operation_logs 表定义。
type OperationLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false"`
	OperatorID int64     `gorm:"column:operator_id;not null;index"` // 操作人ID
	Action     string    `gorm:"type:varchar(50);not null;index"`   // 操作类型
	TargetType string    `gorm:"type:varchar(50);not null"`         // 操作对象类型
	TargetID   *int64    `gorm:""`                                  // 操作对象ID（可空）
	Detail     *string   `gorm:"type:jsonb"`                        // 操作详情 JSON（可空）
	IP         string    `gorm:"type:varchar(45);not null"`         // 操作人IP
	CreatedAt  time.Time `gorm:"not null;default:now();index"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}

// LogEntry 审计日志条目（用于构建日志）
type LogEntry struct {
	OperatorID int64       // 操作人ID
	Action     string      // 操作类型（如 create_user, delete_course）
	TargetType string      // 操作对象类型（如 user, course）
	TargetID   int64       // 操作对象ID（0 表示无特定对象）
	IP         string      // 客户端IP
	Detail     interface{} // 操作详情（会被 JSON 序列化）
}

// Record 异步记录操作日志
// 不阻塞业务流程，写入失败仅记录错误日志
func Record(db *gorm.DB, entry *LogEntry) {
	if db == nil || entry == nil {
		return
	}
	cronpkg.RunAsync("操作审计日志写入", func(ctx context.Context) {
		log := &OperationLog{
			ID:         snowflake.Generate(),
			OperatorID: entry.OperatorID,
			Action:     entry.Action,
			TargetType: entry.TargetType,
			IP:         entry.IP,
			CreatedAt:  time.Now().UTC(),
		}
		if entry.TargetID > 0 {
			log.TargetID = &entry.TargetID
		}
		if entry.Detail != nil {
			detailJSON, err := json.Marshal(entry.Detail)
			if err != nil {
				logger.L.Error("序列化操作日志详情失败",
					zap.Error(err),
					zap.String("action", entry.Action),
					zap.Int64("operator_id", entry.OperatorID),
				)
				return
			}
			detailStr := string(detailJSON)
			log.Detail = &detailStr
		}

		if err := db.WithContext(ctx).Create(log).Error; err != nil {
			logger.L.Error("写入操作日志失败",
				zap.Error(err),
				zap.String("action", entry.Action),
				zap.Int64("operator_id", entry.OperatorID),
			)
		}
	})
}

// RecordSync 同步记录操作日志
// 用于需要确保日志写入成功的场景
func RecordSync(db *gorm.DB, entry *LogEntry) error {
	if db == nil || entry == nil {
		return nil
	}
	log := &OperationLog{
		ID:         snowflake.Generate(),
		OperatorID: entry.OperatorID,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		IP:         entry.IP,
		CreatedAt:  time.Now().UTC(),
	}
	if entry.TargetID > 0 {
		log.TargetID = &entry.TargetID
	}
	if entry.Detail != nil {
		detailJSON, err := json.Marshal(entry.Detail)
		if err != nil {
			return err
		}
		detailStr := string(detailJSON)
		log.Detail = &detailStr
	}

	return db.Create(log).Error
}

// RecordFromContext 从 ServiceContext 风格参数异步记录操作日志。
// 这是当前审计包对外保留的统一便捷入口，service 层应统一复用它，避免再增加同职责别名。
func RecordFromContext(db *gorm.DB, operatorID int64, clientIP, action, targetType string, targetID int64, detail interface{}) {
	Record(db, &LogEntry{
		OperatorID: operatorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         clientIP,
		Detail:     detail,
	})
}
