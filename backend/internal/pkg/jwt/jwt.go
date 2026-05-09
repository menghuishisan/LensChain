// jwt.go
// 该文件集中实现平台的 JWT 令牌签发与解析规则，包括 Access Token、Refresh Token、
// 首次登录改密用的临时 Token，以及实验仿真 WebSocket 会话令牌。认证模块和需要做
// WebSocket query token 鉴权的场景，都应复用这里而不是自行拼装 claims。

package jwt

import (
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/lenschain/backend/internal/config"
)

// TokenType Token类型
type TokenType string

const (
	TokenTypeAccess    TokenType = "access"
	TokenTypeRefresh   TokenType = "refresh"
	TokenTypeTemp      TokenType = "temp"
	TokenTypeSimWS     TokenType = "sim_ws"
	TokenTypeToolProxy TokenType = "tool_proxy"
)

// Claims Access/Refresh Token 载荷
type Claims struct {
	UserID    int64     `json:"user_id"`
	SchoolID  int64     `json:"school_id"`
	Roles     []string  `json:"roles"`
	TokenType TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// TempClaims 临时Token载荷（首次登录改密）
type TempClaims struct {
	UserID    int64     `json:"user_id"`
	TokenType TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// SimWSClaims SimEngine WebSocket 专用载荷
type SimWSClaims struct {
	UserID     int64     `json:"user_id"`
	SchoolID   int64     `json:"school_id"`
	Roles      []string  `json:"roles"`
	SessionID  string    `json:"session_id"`
	InstanceID string    `json:"instance_id"`
	AccessMode string    `json:"access_mode"`
	TokenType  TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// ToolProxyClaims 工具反代 cookie 专用载荷。
//
// 设计目标：iframe 加载工具页面（code-server / blockscout / VNC 等）需要鉴权，但浏览器
// iframe 无法携带 Authorization 头；只能用 cookie。直接把 access token 写进 cookie 风险大
// （HttpOnly 防 XSS，但作用域宽——cookie 一旦泄漏可访问所有 API），因此为反代单独签一个
// 能力收敛的 token：
//   - 仅在 path=/instance/<id>/<kind>/ 下被浏览器发送（path 作用域限制）
//   - 携带 (UserID, InstanceID, ToolKind) 三元组，middleware 校验三者与 URL 路径完全一致
//   - 携带 (Namespace, PodName, Port) 让 ServeToolProxy 0 DB 转发，Pod 重启后凭据失效
//     （PodName 不再匹配 K8s 实际状态），用户重新加载即重签
//   - TTL 与 access token 一致（30min），cookie maxAge 同步
//   - 类型 = TokenTypeToolProxy，与 access token 严格隔离，middleware 不混用
type ToolProxyClaims struct {
	UserID     int64     `json:"user_id"`
	SchoolID   int64     `json:"school_id"`
	InstanceID int64     `json:"instance_id"`
	ToolKind   string    `json:"tool_kind"`
	Namespace  string    `json:"ns"`
	PodName    string    `json:"pod"`
	Port       int       `json:"port"`
	TokenType  TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// TokenPair Access + Refresh Token 对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccessJTI    string `json:"-"`
	RefreshJTI   string `json:"-"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token 过期时间（秒）
}

// GenerateTokenPair 生成 Access + Refresh Token 对
func GenerateTokenPair(userID, schoolID int64, roles []string) (*TokenPair, error) {
	cfg := config.Get().JWT
	return GenerateTokenPairWithExpiry(
		userID,
		schoolID,
		roles,
		cfg.AccessSecret,
		cfg.RefreshSecret,
		cfg.Issuer,
		cfg.AccessExpire,
		cfg.RefreshExpire,
	)
}

// GenerateTokenPairWithExpiry 使用指定密钥和时效生成 Access + Refresh Token 对
// 用于安全策略动态覆盖 Token 有效期，避免调用方重复实现签名逻辑。
func GenerateTokenPairWithExpiry(
	userID, schoolID int64,
	roles []string,
	accessSecret, refreshSecret, issuer string,
	accessExpire, refreshExpire time.Duration,
) (*TokenPair, error) {

	// 生成 Access Token
	accessClaims := &Claims{
		UserID:    userID,
		SchoolID:  schoolID,
		Roles:     roles,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(accessExpire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    issuer,
			ID:        uuid.New().String(), // JTI，用于黑名单
		},
	}
	accessToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, accessClaims).
		SignedString([]byte(accessSecret))
	if err != nil {
		return nil, fmt.Errorf("生成Access Token失败: %w", err)
	}

	// 生成 Refresh Token
	refreshClaims := &Claims{
		UserID:    userID,
		SchoolID:  schoolID,
		Roles:     roles,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(refreshExpire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    issuer,
			ID:        uuid.New().String(),
		},
	}
	refreshToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, refreshClaims).
		SignedString([]byte(refreshSecret))
	if err != nil {
		return nil, fmt.Errorf("生成Refresh Token失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessJTI:    accessClaims.ID,
		RefreshJTI:   refreshClaims.ID,
		ExpiresIn:    int64(accessExpire.Seconds()),
	}, nil
}

// GenerateTempToken 生成临时Token（首次登录强制改密，5分钟有效）
func GenerateTempToken(userID int64) (string, error) {
	cfg := config.Get().JWT

	claims := &TempClaims{
		UserID:    userID,
		TokenType: TokenTypeTemp,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(cfg.TempExpire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			ID:        uuid.New().String(),
		},
	}

	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).
		SignedString([]byte(cfg.AccessSecret))
}

// GenerateSimWSToken 生成 SimEngine WebSocket 会话级 token。
func GenerateSimWSToken(
	userID, schoolID int64,
	roles []string,
	sessionID, instanceID, accessMode string,
	expire time.Duration,
) (string, error) {
	cfg := config.Get().JWT
	if expire <= 0 {
		expire = cfg.AccessExpire
	}

	// SimEngine Core 配置 ws_jwt_audience: "sim-engine"，校验 token aud 必须严格匹配。
	// 详见 sim-engine/core/internal/server/jwt_validator.go::matchAudience。
	// 该 token 只用于 backend 代理 → SimEngine Core 一跳，受众严格收敛到 sim-engine。
	claims := &SimWSClaims{
		UserID:     userID,
		SchoolID:   schoolID,
		Roles:      roles,
		SessionID:  sessionID,
		InstanceID: instanceID,
		AccessMode: accessMode,
		TokenType:  TokenTypeSimWS,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  jwtv5.ClaimStrings{"sim-engine"},
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			ID:        uuid.New().String(),
		},
	}

	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).
		SignedString([]byte(cfg.AccessSecret))
}

// GenerateToolProxyToken 为工具反代签发 cookie token。
//
// 必须在调用方完成业务校验（学生本人 / 实例运行 / 该 toolKind 容器存在）之后再签。本函数
// 仅做 token 字符串编码，不做权限判断。expire <= 0 时使用 cfg.AccessExpire 默认 30 分钟。
func GenerateToolProxyToken(
	userID, schoolID, instanceID int64,
	toolKind, namespace, podName string,
	port int,
	expire time.Duration,
) (string, error) {
	cfg := config.Get().JWT
	if expire <= 0 {
		expire = cfg.AccessExpire
	}
	claims := &ToolProxyClaims{
		UserID:     userID,
		SchoolID:   schoolID,
		InstanceID: instanceID,
		ToolKind:   toolKind,
		Namespace:  namespace,
		PodName:    podName,
		Port:       port,
		TokenType:  TokenTypeToolProxy,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			ID:        uuid.New().String(),
		},
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(cfg.AccessSecret))
}

// ParseToolProxyToken 解析工具反代 cookie token。
//
// 严格校验 token_type=tool_proxy，与 access token 不可互换：access token 不能当反代凭证用，
// 反代凭证也不能当 API access token 用（中间件分别校验）。
func ParseToolProxyToken(tokenString string) (*ToolProxyClaims, error) {
	cfg := config.Get().JWT

	token, err := jwtv5.ParseWithClaims(tokenString, &ToolProxyClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
		}
		return []byte(cfg.AccessSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析工具反代Token失败: %w", err)
	}

	claims, ok := token.Claims.(*ToolProxyClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的工具反代Token")
	}
	if claims.TokenType != TokenTypeToolProxy {
		return nil, fmt.Errorf("Token类型不匹配，期望tool_proxy")
	}
	return claims, nil
}

// ParseAccessToken 解析 Access Token
func ParseAccessToken(tokenString string) (*Claims, error) {
	cfg := config.Get().JWT
	return parseToken(tokenString, cfg.AccessSecret, TokenTypeAccess)
}

// ParseRefreshToken 解析 Refresh Token
func ParseRefreshToken(tokenString string) (*Claims, error) {
	cfg := config.Get().JWT
	return parseToken(tokenString, cfg.RefreshSecret, TokenTypeRefresh)
}

// ParseTempToken 解析临时Token
func ParseTempToken(tokenString string) (*TempClaims, error) {
	cfg := config.Get().JWT

	token, err := jwtv5.ParseWithClaims(tokenString, &TempClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
		}
		return []byte(cfg.AccessSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析临时Token失败: %w", err)
	}

	claims, ok := token.Claims.(*TempClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的临时Token")
	}

	if claims.TokenType != TokenTypeTemp {
		return nil, fmt.Errorf("Token类型不匹配，期望temp")
	}

	return claims, nil
}

// parseToken 通用Token解析
func parseToken(tokenString, secret string, expectedType TokenType) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &Claims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析Token失败: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的Token")
	}

	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("Token类型不匹配，期望%s", expectedType)
	}

	return claims, nil
}
