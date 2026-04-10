// audit.go
// 操作审计日志工具
// 提供统一的操作日志记录函数
// 审计日志只插入不更新不删除（红线规则）
// 供各模块 service 层调用，避免各模块重复实现操作日志记录
//
// 结构体字段严格对照 migrations/009_create_operation_logs.up.sql

package audit

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// OperationLog 操作日志结构体
// 严格对照 operation_logs 表定义（migrations/009）
type OperationLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false"`
	OperatorID int64     `gorm:"column:operator_id;not null;index"`  // 操作人ID
	Action     string    `gorm:"type:varchar(50);not null;index"`    // 操作类型
	TargetType string    `gorm:"type:varchar(50);not null"`          // 操作对象类型
	TargetID   *int64    `gorm:""`                                   // 操作对象ID（可空）
	Detail     *string   `gorm:"type:jsonb"`                         // 操作详情 JSON（可空）
	IP         string    `gorm:"type:varchar(45);not null"`          // 操作人IP
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
	go func() {
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
			detailJSON, _ := json.Marshal(entry.Detail)
			detailStr := string(detailJSON)
			log.Detail = &detailStr
		}

		if err := db.Create(log).Error; err != nil {
			logger.L.Error("写入操作日志失败",
				zap.Error(err),
				zap.String("action", entry.Action),
				zap.Int64("operator_id", entry.OperatorID),
			)
		}
	}()
}

// RecordSync 同步记录操作日志
// 用于需要确保日志写入成功的场景
func RecordSync(db *gorm.DB, entry *LogEntry) error {
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
		detailJSON, _ := json.Marshal(entry.Detail)
		detailStr := string(detailJSON)
		log.Detail = &detailStr
	}

	return db.Create(log).Error
}

// RecordFromContext 从 ServiceContext 风格参数异步记录操作日志
// 便捷方法，避免每次手动构建 LogEntry
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

// RecordFromCtx 从 ServiceContext 异步记录操作日志
// 接受 context.Context 以保持接口一致性（当前未使用 ctx，预留扩展）
func RecordFromCtx(_ context.Context, db *gorm.DB, operatorID int64, clientIP, action, targetType string, targetID int64, detail interface{}) {
	RecordFromContext(db, operatorID, clientIP, action, targetType, targetID, detail)
}
