// Package server 提供 SimEngine WebSocket JWT 鉴权能力。
package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/lenschain/sim-engine/core/internal/app"
)

// DefaultTokenValidator 按文档要求校验 WebSocket JWT、会话归属和实例归属。
type DefaultTokenValidator struct {
	engine   *app.Engine
	secret   []byte
	issuer   string
	audience string
	now      func() time.Time
}

// jwtHeader 是 WebSocket JWT 头部结构。
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtClaims 是 SimEngine WebSocket 鉴权需要消费的声明集合。
type jwtClaims struct {
	Sub        string      `json:"sub"`
	UserID     json.Number `json:"user_id"`
	SchoolID   json.Number `json:"school_id"`
	Roles      []string    `json:"roles"`
	SessionID  string      `json:"session_id"`
	InstanceID string      `json:"instance_id"`
	AccessMode string      `json:"access_mode"`
	Iss        string      `json:"iss"`
	Aud        any         `json:"aud"`
	Exp        json.Number `json:"exp"`
	Nbf        json.Number `json:"nbf"`
	Iat        json.Number `json:"iat"`
}

// NewDefaultTokenValidator 创建默认 JWT 鉴权器。
func NewDefaultTokenValidator(engine *app.Engine, secret string, issuer string, audience string) DefaultTokenValidator {
	return DefaultTokenValidator{
		engine:   engine,
		secret:   []byte(strings.TrimSpace(secret)),
		issuer:   strings.TrimSpace(issuer),
		audience: strings.TrimSpace(audience),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Validate 校验 JWT 签名、时效性，以及 token 与会话/实例/权限归属关系。
func (v DefaultTokenValidator) Validate(sessionID string, token string) (AccessGrant, error) {
	if v.engine == nil {
		return AccessGrant{}, errors.New("sim-engine auth validator is not configured")
	}
	if strings.TrimSpace(token) == "" {
		return AccessGrant{}, errors.New("token is required")
	}
	if len(v.secret) == 0 {
		return AccessGrant{}, errors.New("sim-engine ws jwt secret is not configured")
	}

	binding, ok := v.engine.SessionBinding(sessionID)
	if !ok {
		return AccessGrant{}, errors.New("session binding is not found")
	}

	header, claims, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return AccessGrant{}, err
	}
	if !strings.EqualFold(header.Alg, "HS256") {
		return AccessGrant{}, errors.New("unsupported jwt alg")
	}

	expected := signHS256(signingInput, v.secret)
	if !hmac.Equal(signature, expected) {
		return AccessGrant{}, errors.New("jwt signature is invalid")
	}
	grant, err := validateJWTClaims(claims, binding, v.issuer, v.audience, v.now())
	if err != nil {
		return AccessGrant{}, err
	}
	return grant, nil
}

// parseJWT 解析 JWT 并返回头部、声明与签名信息。
func parseJWT(token string) (jwtHeader, jwtClaims, string, []byte, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt header encoding")
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt claims encoding")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt signature encoding")
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt header json")
	}

	decoder := json.NewDecoder(strings.NewReader(string(claimsBytes)))
	decoder.UseNumber()
	var claims jwtClaims
	if err := decoder.Decode(&claims); err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, errors.New("invalid jwt claims json")
	}

	return header, claims, parts[0] + "." + parts[1], signature, nil
}

// signHS256 计算 HS256 JWT 签名。
func signHS256(signingInput string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}

// validateJWTClaims 校验声明时效性与资源归属关系。
func validateJWTClaims(
	claims jwtClaims,
	binding app.SessionBinding,
	expectedIssuer string,
	expectedAudience string,
	now time.Time,
) (AccessGrant, error) {
	if strings.TrimSpace(claims.SessionID) == "" || claims.SessionID != binding.SessionID {
		return AccessGrant{}, errors.New("jwt session_id does not match")
	}
	if strings.TrimSpace(claims.InstanceID) == "" || claims.InstanceID != binding.InstanceID {
		return AccessGrant{}, errors.New("jwt instance_id does not match")
	}
	if expectedIssuer != "" && strings.TrimSpace(claims.Iss) != expectedIssuer {
		return AccessGrant{}, errors.New("jwt issuer does not match")
	}
	if expectedAudience != "" && !matchAudience(claims.Aud, expectedAudience) {
		return AccessGrant{}, errors.New("jwt audience does not match")
	}
	if err := validateJWTTimeClaim(claims.Exp, now, func(value int64) bool {
		return now.Unix() < value
	}, "jwt is expired"); err != nil {
		return AccessGrant{}, err
	}
	if err := validateJWTTimeClaim(claims.Nbf, now, func(value int64) bool {
		return now.Unix() >= value
	}, "jwt is not active yet"); err != nil {
		return AccessGrant{}, err
	}
	if err := validateJWTTimeClaim(claims.Iat, now, func(value int64) bool {
		return now.Unix() >= value
	}, "jwt issued_at is in the future"); err != nil {
		return AccessGrant{}, err
	}

	subject := strings.TrimSpace(claims.UserID.String())
	if subject == "" {
		subject = strings.TrimSpace(claims.Sub)
	}
	if subject != "" && subject == binding.StudentID {
		return AccessGrant{ReadOnly: false}, nil
	}

	accessMode := strings.ToLower(strings.TrimSpace(claims.AccessMode))
	if hasObserverRole(claims.Roles) && isReadOnlyAccessMode(accessMode) {
		return AccessGrant{ReadOnly: true}, nil
	}
	return AccessGrant{}, errors.New("jwt subject or observer permission does not match session access policy")
}

// validateJWTTimeClaim 校验数值型时间声明。
func validateJWTTimeClaim(claim json.Number, now time.Time, allow func(value int64) bool, errMessage string) error {
	if strings.TrimSpace(claim.String()) == "" {
		return nil
	}
	value, err := claim.Int64()
	if err != nil {
		return errors.New("invalid jwt time claim")
	}
	if !allow(value) {
		return errors.New(errMessage)
	}
	return nil
}

// matchAudience 判断 aud 声明是否包含预期受众。
func matchAudience(aud any, expected string) bool {
	switch value := aud.(type) {
	case string:
		return strings.TrimSpace(value) == expected
	case []any:
		for _, item := range value {
			text, ok := item.(string)
			if ok && strings.TrimSpace(text) == expected {
				return true
			}
		}
	}
	return false
}

// hasObserverRole 判断 token 角色集合中是否包含教师/管理员只读观察角色。
func hasObserverRole(roles []string) bool {
	for _, role := range roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "teacher", "school_admin", "super_admin":
			return true
		}
	}
	return false
}

// isReadOnlyAccessMode 判断 token 是否显式声明为只读观察访问。
func isReadOnlyAccessMode(mode string) bool {
	switch mode {
	case "readonly", "read_only", "observe", "observer":
		return true
	default:
		return false
	}
}
