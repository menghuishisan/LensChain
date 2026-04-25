// token_state.go
// 该文件封装令牌失效基线与黑名单状态的统一读写能力，用来支撑“强制下线、其他设备登录
// 失效、踢人退出”这类认证安全场景。认证服务和 JWT 中间件通过它共享同一套令牌状态判断。

package tokenstate

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/database"
)

const tokenValidAfterTTL = 7 * 24 * time.Hour

// normalizeTokenValidAfter 将 Token 生效基线统一到秒级精度。
// JWT iat 默认按秒序列化，若这里保留纳秒精度，会导致刚签发的 token 被误判为“早于有效基线”。
func normalizeTokenValidAfter(validAfter time.Time) time.Time {
	return validAfter.UTC().Truncate(time.Second)
}

// SetTokenValidAfter 缓存用户Token生效时间基线
// 任何早于该时间签发的 Access Token 都应视为失效。
func SetTokenValidAfter(ctx context.Context, userID int64, validAfter time.Time) error {
	normalized := normalizeTokenValidAfter(validAfter)
	return cache.Set(ctx, cache.KeyTokenValidAfter+strconv.FormatInt(userID, 10), normalized.Format(time.RFC3339Nano), tokenValidAfterTTL)
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

// ResolveTokenValidAfter 获取用户 Token 生效时间基线。
// 优先读取缓存，缓存未命中时回源数据库并自动回填缓存。
func ResolveTokenValidAfter(ctx context.Context, userID int64) (*time.Time, error) {
	validAfter, err := GetTokenValidAfter(ctx, userID)
	if err != nil {
		return nil, err
	}
	if validAfter != nil {
		return validAfter, nil
	}

	var user entity.User
	err = database.Get().
		WithContext(ctx).
		Select("token_valid_after").
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	if user.TokenValidAfter.IsZero() {
		return nil, nil
	}

	_ = SetTokenValidAfter(ctx, userID, user.TokenValidAfter)
	return &user.TokenValidAfter, nil
}

// IsTokenBlacklisted 判断访问令牌是否已加入黑名单。
func IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	return cache.Exists(ctx, cache.KeyTokenBlacklist+jti)
}

// BlacklistToken 将访问令牌加入黑名单。
// 调用方应传入该令牌的剩余有效期，避免黑名单键长期残留。
func BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	return cache.Set(ctx, cache.KeyTokenBlacklist+jti, "1", ttl)
}
