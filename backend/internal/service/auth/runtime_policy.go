// runtime_policy.go
// 模块01 — 用户与认证：运行时安全策略辅助能力
// 统一封装密码复杂度校验与 Token 时效解析，避免各服务重复实现

package auth

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

var specialCharRegex = regexp.MustCompile(`[^A-Za-z0-9]`)
var passwordUpperRegex = regexp.MustCompile(`[A-Z]`)
var passwordLowerRegex = regexp.MustCompile(`[a-z]`)
var passwordDigitRegex = regexp.MustCompile(`[0-9]`)

// runtimeSecurityPolicy 运行时安全策略
// 从缓存配置解包后供 auth 模块内部使用。
type runtimeSecurityPolicy struct {
	LoginFailMaxCount        int
	LoginLockDurationMinutes int
	PasswordMinLength        int
	PasswordRequireUppercase bool
	PasswordRequireLowercase bool
	PasswordRequireDigit     bool
	PasswordRequireSpecial   bool
	AccessTokenExpireMinutes int
	RefreshTokenExpireDays   int
}

// runtimePolicyProvider 安全策略读取接口
// 便于在 service 层复用与测试替换。
type runtimePolicyProvider interface {
	GetRuntimeSecurityPolicy(ctx context.Context) (*runtimeSecurityPolicy, error)
}

type cacheRuntimePolicyProvider struct{}

// GetRuntimeSecurityPolicy 获取运行时安全策略
// 缓存缺失或解析失败时回落到默认策略，避免影响认证主流程。
func (p *cacheRuntimePolicyProvider) GetRuntimeSecurityPolicy(ctx context.Context) (*runtimeSecurityPolicy, error) {
	data, err := cache.GetString(ctx, cache.KeySecurityPolicy)
	if err != nil || data == "" {
		return defaultRuntimeSecurityPolicy(), nil
	}

	var resp dto.SecurityPolicyResp
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return defaultRuntimeSecurityPolicy(), nil
	}

	return &runtimeSecurityPolicy{
		LoginFailMaxCount:        resp.LoginFailMaxCount,
		LoginLockDurationMinutes: resp.LoginLockDurationMinutes,
		PasswordMinLength:        resp.PasswordMinLength,
		PasswordRequireUppercase: resp.PasswordRequireUppercase,
		PasswordRequireLowercase: resp.PasswordRequireLowercase,
		PasswordRequireDigit:     resp.PasswordRequireDigit,
		PasswordRequireSpecial:   resp.PasswordRequireSpecialChar,
		AccessTokenExpireMinutes: resp.AccessTokenExpireMinutes,
		RefreshTokenExpireDays:   resp.RefreshTokenExpireDays,
	}, nil
}

// defaultRuntimeSecurityPolicy 返回默认运行时安全策略
func defaultRuntimeSecurityPolicy() *runtimeSecurityPolicy {
	return &runtimeSecurityPolicy{
		LoginFailMaxCount:        5,
		LoginLockDurationMinutes: 15,
		PasswordMinLength:        8,
		PasswordRequireUppercase: true,
		PasswordRequireLowercase: true,
		PasswordRequireDigit:     true,
		PasswordRequireSpecial:   false,
		AccessTokenExpireMinutes: 30,
		RefreshTokenExpireDays:   7,
	}
}

// getRuntimeSecurityPolicyOrDefault 获取运行时安全策略。
// 读取失败时回落到默认策略，避免认证主流程被配置读取失败阻断。
func getRuntimeSecurityPolicyOrDefault(ctx context.Context, provider runtimePolicyProvider) *runtimeSecurityPolicy {
	if provider == nil {
		return defaultRuntimeSecurityPolicy()
	}
	policy, err := provider.GetRuntimeSecurityPolicy(ctx)
	if err != nil || policy == nil {
		return defaultRuntimeSecurityPolicy()
	}
	return policy
}

// resolveAccessTokenTTLByProvider 获取当前生效的 Access Token 时长。
func resolveAccessTokenTTLByProvider(ctx context.Context, provider runtimePolicyProvider) time.Duration {
	cfg := config.Get().JWT
	policy := getRuntimeSecurityPolicyOrDefault(ctx, provider)
	accessExpire, _ := resolveJWTDurations(&cfg, policy)
	return accessExpire
}

// validatePasswordWithPolicy 使用运行时策略校验密码复杂度
func validatePasswordWithPolicy(password string, policy *runtimeSecurityPolicy) error {
	if policy == nil {
		policy = defaultRuntimeSecurityPolicy()
	}

	if len(password) < policy.PasswordMinLength {
		return errcode.ErrPasswordComplexity
	}
	if policy.PasswordRequireUppercase && !passwordUpperRegex.MatchString(password) {
		return errcode.ErrPasswordComplexity
	}
	if policy.PasswordRequireLowercase && !passwordLowerRegex.MatchString(password) {
		return errcode.ErrPasswordComplexity
	}
	if policy.PasswordRequireDigit && !passwordDigitRegex.MatchString(password) {
		return errcode.ErrPasswordComplexity
	}
	if policy.PasswordRequireSpecial && !specialCharRegex.MatchString(password) {
		return errcode.ErrPasswordComplexity
	}

	return nil
}

// resolveJWTDurations 根据运行时策略解析 Token 时效
func resolveJWTDurations(cfg *config.JWTConfig, policy *runtimeSecurityPolicy) (time.Duration, time.Duration) {
	accessExpire := cfg.AccessExpire
	refreshExpire := cfg.RefreshExpire

	if policy != nil && policy.AccessTokenExpireMinutes > 0 {
		accessExpire = time.Duration(policy.AccessTokenExpireMinutes) * time.Minute
	}
	if policy != nil && policy.RefreshTokenExpireDays > 0 {
		refreshExpire = time.Duration(policy.RefreshTokenExpireDays) * 24 * time.Hour
	}

	return accessExpire, refreshExpire
}
