// checker.go
// 该文件提供系统管理模块需要的服务健康检查基础能力，负责统一探测 PostgreSQL、Redis、
// NATS 和 MinIO 的可达性与延迟，并把结果整理成可直接用于仪表盘和告警判断的结构化数据。
// 它解决的是“基础设施健康状态怎么测、怎么统一表达”的问题，不承担告警规则触发本身。

package healthcheck

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/mq"
	"github.com/lenschain/backend/internal/pkg/storage"
)

// Status 表示基础设施服务的健康状态。
type Status string

const (
	// StatusHealthy 表示服务在允许时间内成功响应。
	StatusHealthy Status = "healthy"
	// StatusUnhealthy 表示服务明确不可达或调用失败。
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown 表示检查超时或依赖未初始化，当前无法给出稳定结论。
	StatusUnknown Status = "unknown"
)

// Result 描述一次服务健康检查结果。
// 系统仪表盘、服务状态告警和定时巡检都可以直接复用这个结构。
type Result struct {
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// CheckCoreServices 统一检查平台核心依赖服务。
// 当前覆盖 PostgreSQL、Redis、NATS 和 MinIO，顺序与运维面板展示口径保持一致。
func CheckCoreServices(ctx context.Context) []Result {
	return []Result{
		CheckPostgres(ctx),
		CheckRedis(ctx),
		CheckNATS(ctx),
		CheckMinIO(ctx),
	}
}

// CheckPostgres 检查 PostgreSQL 的连通性与响应时间。
func CheckPostgres(ctx context.Context) Result {
	db := database.Get()
	if db == nil {
		return unknown("PostgreSQL", "数据库客户端未初始化")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return unhealthy("PostgreSQL", 0, err.Error())
	}

	startedAt := time.Now()
	if err := sqlDB.PingContext(ctx); err != nil {
		if ctx.Err() != nil {
			return unknown("PostgreSQL", "健康检查超时")
		}
		return unhealthy("PostgreSQL", time.Since(startedAt), err.Error())
	}
	return healthy("PostgreSQL", time.Since(startedAt))
}

// CheckRedis 检查 Redis 的连通性与响应时间。
func CheckRedis(ctx context.Context) Result {
	client := cache.Get()
	if client == nil {
		return unknown("Redis", "Redis 客户端未初始化")
	}

	startedAt := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		if ctx.Err() != nil {
			return unknown("Redis", "健康检查超时")
		}
		return unhealthy("Redis", time.Since(startedAt), err.Error())
	}
	return healthy("Redis", time.Since(startedAt))
}

// CheckNATS 检查 NATS 连接状态。
// 这里优先使用当前客户端连接状态判断，避免仅靠本地对象存在就误判为可用。
func CheckNATS(ctx context.Context) Result {
	conn := mq.GetConn()
	if conn == nil {
		return unknown("NATS", "NATS 客户端未初始化")
	}

	startedAt := time.Now()
	switch conn.Status() {
	case nats.CONNECTED:
		// 连接状态显示为已连接时，再向服务端发一次 Flush 确认，避免本地状态滞后导致误报。
		if err := conn.FlushWithContext(ctx); err != nil {
			if ctx.Err() != nil {
				return unknown("NATS", "健康检查超时")
			}
			return unhealthy("NATS", time.Since(startedAt), err.Error())
		}
		return healthy("NATS", time.Since(startedAt))
	case nats.CLOSED:
		return unhealthy("NATS", time.Since(startedAt), "连接已关闭")
	case nats.RECONNECTING:
		return unhealthy("NATS", time.Since(startedAt), "连接重连中")
	case nats.DISCONNECTED:
		return unhealthy("NATS", time.Since(startedAt), "连接已断开")
	default:
		if conn.IsClosed() {
			return unhealthy("NATS", time.Since(startedAt), "连接已关闭")
		}
		return unknown("NATS", "连接状态未知")
	}
}

// CheckMinIO 检查 MinIO 的可达性与响应时间。
// 通过列举 1 个对象的方式验证桶访问可用性，避免仅凭客户端对象存在误判。
func CheckMinIO(ctx context.Context) Result {
	client := storage.GetClient()
	if client == nil {
		return unknown("MinIO", "MinIO 客户端未初始化")
	}

	startedAt := time.Now()
	if _, err := storage.ListObjects(ctx, "", false); err != nil {
		if ctx.Err() != nil {
			return unknown("MinIO", "健康检查超时")
		}
		return unhealthy("MinIO", time.Since(startedAt), err.Error())
	}
	return healthy("MinIO", time.Since(startedAt))
}

func healthy(name string, latency time.Duration) Result {
	return Result{
		Name:      name,
		Status:    StatusHealthy,
		LatencyMS: latency.Milliseconds(),
		CheckedAt: time.Now().UTC(),
	}
}

func unhealthy(name string, latency time.Duration, detail string) Result {
	return Result{
		Name:      name,
		Status:    StatusUnhealthy,
		LatencyMS: latency.Milliseconds(),
		Detail:    detail,
		CheckedAt: time.Now().UTC(),
	}
}

func unknown(name string, detail string) Result {
	return Result{
		Name:      name,
		Status:    StatusUnknown,
		LatencyMS: 0,
		Detail:    detail,
		CheckedAt: time.Now().UTC(),
	}
}
