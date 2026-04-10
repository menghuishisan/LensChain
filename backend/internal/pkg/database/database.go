// database.go
// 数据库连接与 GORM 封装
// PostgreSQL 连接池管理、软删除基类、GORM 日志级别配置
// 所有表统一使用雪花ID主键、软删除、时间戳字段

package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局数据库实例
var db *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) error {
	logLevel := parseLogLevel(cfg.LogLevel)

	var err error
	db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // 使用复数表名（users, courses 等）
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		// 禁用默认事务（提升性能，需要事务时手动开启）
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接池失败: %w", err)
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接验证失败: %w", err)
	}

	logger.L.Info("数据库连接成功",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("dbname", cfg.DBName),
	)

	return nil
}

// Get 获取全局数据库实例
func Get() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// parseLogLevel 解析 GORM 日志级别
func parseLogLevel(level string) gormlogger.LogLevel {
	switch level {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn":
		return gormlogger.Warn
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

// BaseModel 基础模型
// 所有实体必须嵌入此结构体
// 包含：雪花ID主键、创建时间、更新时间、软删除时间
type BaseModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BaseModelNoSoftDelete 无软删除的基础模型
// 用于日志表等不需要软删除的表（如 login_logs, operation_logs）
type BaseModelNoSoftDelete struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// Scopes 通用查询作用域

// WithSchoolID 多租户过滤（按 school_id 隔离）
func WithSchoolID(schoolID int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if schoolID > 0 {
			return db.Where("school_id = ?", schoolID)
		}
		return db
	}
}

// WithKeywordSearch 关键字模糊搜索
// fields 为需要搜索的字段列表
// 使用子查询分组避免 OR 条件破坏已有 WHERE 约束（如多租户 school_id 过滤）
// P3-10 修复：转义 LIKE 通配符（%、_、\），防止用户输入注入
func WithKeywordSearch(keyword string, fields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if keyword == "" || len(fields) == 0 {
			return db
		}
		// P3-10 修复：转义 LIKE 特殊字符
		escaped := escapeLike(keyword)
		// 构建独立的 OR 子查询，避免与外层 AND 条件冲突
		sub := db.Session(&gorm.Session{NewDB: true})
		for i, field := range fields {
			if i == 0 {
				sub = sub.Where(field+" ILIKE ?", "%"+escaped+"%")
			} else {
				sub = sub.Or(field+" ILIKE ?", "%"+escaped+"%")
			}
		}
		return db.Where(sub)
	}
}

// escapeLike 转义 LIKE 通配符（%、_、\）
// 防止用户输入的 % 或 _ 被当作通配符执行
func escapeLike(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%', '_', '\\':
			result = append(result, '\\', s[i])
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// WithStatus 状态过滤
func WithStatus(status int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status > 0 {
			return db.Where("status = ?", status)
		}
		return db
	}
}

// WithDateRange 时间范围过滤
func WithDateRange(field, from, to string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if from != "" {
			db = db.Where(field+" >= ?", from)
		}
		if to != "" {
			db = db.Where(field+" <= ?", to)
		}
		return db
	}
}
