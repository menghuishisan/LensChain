// transaction.go
// 数据库事务辅助工具
// 为 service 层提供事务管理封装
// service 层跨 repository 调用时必须使用事务

package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TxFunc 事务执行函数类型
// 接收事务 *gorm.DB，返回错误时自动回滚
type TxFunc func(tx *gorm.DB) error

// Transaction 执行数据库事务
// 自动处理 Begin/Commit/Rollback
// 如果 fn 返回 error 或 panic，事务自动回滚
func Transaction(fn TxFunc) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// TransactionWithContext 带 context 的事务执行
func TransactionWithContext(ctx context.Context, fn TxFunc) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// TransactionWithDB 使用指定数据库实例执行事务
// 适用于已通过依赖注入持有 *gorm.DB 的 service/repository，避免直接重复编写事务模板。
func TransactionWithDB(ctx context.Context, baseDB *gorm.DB, fn TxFunc) error {
	return baseDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// TransactionNested 嵌套事务（SavePoint）
// 在已有事务中创建保存点，支持部分回滚
func TransactionNested(tx *gorm.DB, fn TxFunc) error {
	return tx.Transaction(func(nested *gorm.DB) error {
		return fn(nested)
	})
}

// GetDB 获取数据库实例（供 repository 层使用）
// 如果传入的 tx 不为 nil，使用事务实例；否则使用全局实例
func GetDB(tx ...*gorm.DB) *gorm.DB {
	if len(tx) > 0 && tx[0] != nil {
		return tx[0]
	}
	return db
}

// Paginate 通用分页查询
// 返回数据列表和总数
func Paginate(db *gorm.DB, page, pageSize int, dest interface{}) (int64, error) {
	var total int64

	// 先查总数
	if err := db.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("查询总数失败: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	// 再查分页数据
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return 0, fmt.Errorf("查询分页数据失败: %w", err)
	}

	return total, nil
}
