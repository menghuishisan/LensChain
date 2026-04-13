// token_state.go
// Token生效时间基线缓存工具
// 统一封装用户 Access Token 失效基线的缓存读写，供认证服务、中间件和跨模块强制下线场景复用

package tokenstate

import (
	"context"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/pkg/cache"
)

const tokenValidAfterTTL = 7 * 24 * time.Hour

// SetTokenValidAfter 缓存用户Token生效时间基线
// 任何早于该时间签发的 Access Token 都应视为失效。
func SetTokenValidAfter(ctx context.Context, userID int64, validAfter time.Time) error {
	return cache.Set(ctx, cache.KeyTokenValidAfter+strconv.FormatInt(userID, 10), validAfter.Format(time.RFC3339Nano), tokenValidAfterTTL)
}

// GetTokenValidAfter 读取用户Token生效时间基线缓存
// 缓存不存在时返回 nil，不把 miss 视为错误。
func GetTokenValidAfter(ctx context.Context, userID int64) (*time.Time, error) {
	value, err := cache.GetString(ctx, cache.KeyTokenValidAfter+strconv.FormatInt(userID, 10))
	if err != nil {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
