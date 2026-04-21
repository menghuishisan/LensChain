// session_state.go
// 模块01 — 用户与认证：会话状态编排
// 负责统一处理用户会话失效、Token 生效基线刷新等认证态变更

package auth

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/tokenstate"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
)

type authSession struct {
	RefreshToken string `json:"refresh_token"`
	AccessJTI    string `json:"access_jti"`
	DeviceInfo   string `json:"device_info"`
	LoginAt      int64  `json:"login_at"`
}

// invalidateUserSession 使指定用户的历史令牌和在线会话立即失效。
// 用于禁用、归档、重置密码、批量导入覆盖等需要强制重新登录的场景。
func invalidateUserSession(ctx context.Context, userRepo authrepo.UserRepository, userID int64, accessTokenTTL time.Duration) {
	if session, err := getAuthSession(ctx, userID); err == nil {
		_ = tokenstate.BlacklistToken(ctx, session.AccessJTI, accessTokenTTL)
	}

	now := time.Now()
	_ = userRepo.UpdateTokenValidAfter(ctx, userID, now)
	_ = tokenstate.SetTokenValidAfter(ctx, userID, now)
	sessionKey := cache.KeySession + strconv.FormatInt(userID, 10)
	_ = cache.Del(ctx, sessionKey)
}

// getAuthSession 读取当前用户的在线会话信息。
func getAuthSession(ctx context.Context, userID int64) (*authSession, error) {
	sessionKey := cache.KeySession + strconv.FormatInt(userID, 10)
	value, err := cache.GetString(ctx, sessionKey)
	if err != nil || value == "" {
		return nil, err
	}

	var session authSession
	if err := json.Unmarshal([]byte(value), &session); err != nil {
		return nil, err
	}
	return &session, nil
}
