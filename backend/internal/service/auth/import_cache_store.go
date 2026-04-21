// import_cache_store.go
// 模块01 — 用户与认证：导入缓存访问能力。
// 统一封装导入预览与失败明细的缓存读写，避免 service 中散落 Redis Key 细节。

package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lenschain/backend/internal/pkg/cache"
)

// importFailureMeta 导入失败明细元数据
// 记录批次归属学校，供下载失败明细时做租户隔离校验。
type importFailureMeta struct {
	ImportID string `json:"import_id"`
	SchoolID int64  `json:"school_id"`
}

type importFailureCachePayload struct {
	Meta *importFailureMeta `json:"meta"`
	Rows []*importRow       `json:"rows"`
}

// importCacheStore 导入缓存访问接口
// 便于在 service 内统一复用，并在测试中替换。
type importCacheStore interface {
	GetImport(ctx context.Context, importID string) (*importCache, error)
	SetImport(ctx context.Context, data *importCache, expiration time.Duration) error
	DeleteImport(ctx context.Context, importID string) error
	AcquireExecution(ctx context.Context, importID string, expiration time.Duration) (bool, error)
	ReleaseExecution(ctx context.Context, importID string) error
	SetImportFailures(ctx context.Context, importID string, rows []*importRow, meta *importFailureMeta, expiration time.Duration) error
	GetImportFailures(ctx context.Context, importID string) ([]*importRow, *importFailureMeta, error)
}

type redisImportCacheStore struct{}

func importCacheKey(importID string) string {
	return "import:" + importID
}

func importExecutionKey(importID string) string {
	return "import_execute:" + importID
}

func importFailureKey(importID string) string {
	return "import_failures:" + importID
}

// GetImport 获取导入预览缓存
func (s *redisImportCacheStore) GetImport(ctx context.Context, importID string) (*importCache, error) {
	cacheJSON, err := cache.GetString(ctx, importCacheKey(importID))
	if err != nil {
		return nil, err
	}

	var cacheData importCache
	if err := json.Unmarshal([]byte(cacheJSON), &cacheData); err != nil {
		return nil, err
	}
	return &cacheData, nil
}

// SetImport 保存导入预览缓存
func (s *redisImportCacheStore) SetImport(ctx context.Context, data *importCache, expiration time.Duration) error {
	cacheJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return cache.Set(ctx, importCacheKey(data.ImportID), string(cacheJSON), expiration)
}

// DeleteImport 删除导入预览缓存
func (s *redisImportCacheStore) DeleteImport(ctx context.Context, importID string) error {
	return cache.Del(ctx, importCacheKey(importID))
}

// AcquireExecution 获取导入执行锁，避免同一批次被并发重复执行。
func (s *redisImportCacheStore) AcquireExecution(ctx context.Context, importID string, expiration time.Duration) (bool, error) {
	return cache.SetNX(ctx, importExecutionKey(importID), "1", expiration)
}

// ReleaseExecution 释放导入执行锁。
// 仅在执行失败时调用，成功执行后保留锁到过期时间，阻止同批次再次执行。
func (s *redisImportCacheStore) ReleaseExecution(ctx context.Context, importID string) error {
	return cache.Del(ctx, importExecutionKey(importID))
}

// SetImportFailures 保存导入失败明细与批次元数据
func (s *redisImportCacheStore) SetImportFailures(ctx context.Context, importID string, rows []*importRow, meta *importFailureMeta, expiration time.Duration) error {
	payloadJSON, err := json.Marshal(&importFailureCachePayload{
		Meta: meta,
		Rows: rows,
	})
	if err != nil {
		return err
	}
	return cache.Set(ctx, importFailureKey(importID), string(payloadJSON), expiration)
}

// GetImportFailures 获取导入失败明细与批次元数据
func (s *redisImportCacheStore) GetImportFailures(ctx context.Context, importID string) ([]*importRow, *importFailureMeta, error) {
	failJSON, err := cache.GetString(ctx, importFailureKey(importID))
	if err != nil {
		return nil, nil, err
	}

	var payload importFailureCachePayload
	if err := json.Unmarshal([]byte(failJSON), &payload); err != nil {
		return nil, nil, err
	}
	return payload.Rows, payload.Meta, nil
}
