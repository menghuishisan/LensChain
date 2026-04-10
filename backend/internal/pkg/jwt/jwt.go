// jwt.go
// JWT Token 工具
// 实现双Token机制：Access Token（30分钟）+ Refresh Token（7天）
// 支持临时Token（首次登录强制改密，5分钟有效）
// Token 中的 JTI 用于黑名单机制（踢人下线）

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
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeTemp    TokenType = "temp"
)

// Claims Access/Refresh Token 载荷
type Claims struct {
	UserID    int64    `json:"user_id"`
	SchoolID  int64    `json:"school_id"`
	Roles     []string `json:"roles"`
	TokenType TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// TempClaims 临时Token载荷（首次登录改密）
type TempClaims struct {
	UserID    int64     `json:"user_id"`
	TokenType TokenType `json:"token_type"`
	jwtv5.RegisteredClaims
}

// TokenPair Access + Refresh Token 对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token 过期时间（秒）
}

// GenerateTokenPair 生成 Access + Refresh Token 对
func GenerateTokenPair(userID, schoolID int64, roles []string) (*TokenPair, error) {
	cfg := config.Get().JWT

	// 生成 Access Token
	accessClaims := &Claims{
		UserID:    userID,
		SchoolID:  schoolID,
		Roles:     roles,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(cfg.AccessExpire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			ID:        uuid.New().String(), // JTI，用于黑名单
		},
	}
	accessToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, accessClaims).
		SignedString([]byte(cfg.AccessSecret))
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
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(cfg.RefreshExpire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			ID:        uuid.New().String(),
		},
	}
	refreshToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, refreshClaims).
		SignedString([]byte(cfg.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("生成Refresh Token失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(cfg.AccessExpire.Seconds()),
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
