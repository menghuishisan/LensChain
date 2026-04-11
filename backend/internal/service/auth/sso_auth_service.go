// sso_auth_service.go
// 模块01 — 用户与认证：SSO登录业务逻辑
// 负责构建SSO跳转地址、处理CAS/OAuth2回调、绑定账号并签发Token

package auth

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/httpclient"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

type casAuthResponse struct {
	AuthenticationSuccess *struct {
		User       string         `xml:"user"`
		Attributes []casAttribute `xml:"attributes>*"`
	} `xml:"authenticationSuccess"`
}

type casAttribute struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// SSOLoginURL 构建学校SSO登录地址
func (s *authService) SSOLoginURL(ctx context.Context, schoolID int64) (string, error) {
	config, err := s.getEnabledSchoolSSOConfig(ctx, schoolID)
	if err != nil {
		return "", err
	}

	switch config.Provider {
	case "cas":
		return s.buildCASLoginURL(schoolID, config.Config)
	case "oauth2":
		return s.buildOAuth2LoginURL(ctx, schoolID, config.Config)
	default:
		return "", errcode.ErrSSOAuthFailed.WithMessage("学校SSO配置类型不受支持")
	}
}

// SSOCallback 处理SSO回调并完成登录
func (s *authService) SSOCallback(ctx context.Context, schoolID int64, query map[string]string, ip, userAgent string) (*LoginResult, error) {
	config, err := s.getEnabledSchoolSSOConfig(ctx, schoolID)
	if err != nil {
		return nil, err
	}

	var ssoUserID string
	switch config.Provider {
	case "cas":
		ssoUserID, err = s.handleCASCallback(ctx, schoolID, query, config.Config)
	case "oauth2":
		ssoUserID, err = s.handleOAuth2Callback(ctx, schoolID, query, config.Config)
	default:
		return nil, errcode.ErrSSOAuthFailed.WithMessage("学校SSO配置类型不受支持")
	}
	if err != nil {
		return nil, err
	}

	user, err := s.matchSSOUser(ctx, schoolID, config.Provider, ssoUserID)
	if err != nil {
		return nil, err
	}

	return s.completeSSOLogin(ctx, user, config.Provider, ip, userAgent)
}

func (s *authService) getEnabledSchoolSSOConfig(ctx context.Context, schoolID int64) (*SchoolSSOConfig, error) {
	if s.schoolSSOQuerier == nil {
		return nil, errcode.ErrSSOAuthFailed.WithMessage("学校SSO配置查询器未初始化")
	}

	config, err := s.schoolSSOQuerier.GetSchoolSSOConfig(ctx, schoolID)
	if err != nil {
		return nil, errcode.ErrSSOAuthFailed.WithMessage("学校未配置SSO")
	}
	if !config.IsEnabled {
		return nil, errcode.ErrSSOAuthFailed.WithMessage("学校尚未启用SSO登录")
	}
	if !config.IsTested {
		return nil, errcode.ErrSSOAuthFailed.WithMessage("学校SSO配置尚未通过测试")
	}
	return config, nil
}

func (s *authService) buildCASLoginURL(schoolID int64, config map[string]interface{}) (string, error) {
	serverURL, _ := config["cas_server_url"].(string)
	serviceURL, _ := config["cas_service_url"].(string)
	if serverURL == "" || serviceURL == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS配置不完整")
	}

	callbackURL, err := appendURLQuery(serviceURL, map[string]string{"school_id": strconv.FormatInt(schoolID, 10)})
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS回调地址配置错误")
	}

	return appendURLQuery(strings.TrimRight(serverURL, "/")+"/login", map[string]string{
		"service": callbackURL,
	})
}

func (s *authService) buildOAuth2LoginURL(ctx context.Context, schoolID int64, config map[string]interface{}) (string, error) {
	authorizeURL, _ := config["authorize_url"].(string)
	clientID, _ := config["client_id"].(string)
	redirectURI, _ := config["redirect_uri"].(string)
	scope, _ := config["scope"].(string)
	if authorizeURL == "" || clientID == "" || redirectURI == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("OAuth2配置不完整")
	}

	state := fmt.Sprintf("sso_%d_%d", schoolID, snowflake.Generate())
	if err := cache.Set(ctx, cache.KeySSOState+state, strconv.FormatInt(schoolID, 10), 10*time.Minute); err != nil {
		logger.L.Warn("缓存OAuth2 state失败", zap.Error(err))
		return "", errcode.ErrSSOAuthFailed.WithMessage("SSO状态初始化失败")
	}

	return appendURLQuery(authorizeURL, map[string]string{
		"response_type": "code",
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
		"scope":         scope,
		"state":         state,
	})
}

func (s *authService) handleCASCallback(ctx context.Context, schoolID int64, query map[string]string, config map[string]interface{}) (string, error) {
	ticket := strings.TrimSpace(query["ticket"])
	if ticket == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("缺少CAS ticket")
	}

	serverURL, _ := config["cas_server_url"].(string)
	serviceURL, _ := config["cas_service_url"].(string)
	userIDAttr, _ := config["user_id_attribute"].(string)
	if serverURL == "" || serviceURL == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS配置不完整")
	}

	callbackURL, err := appendURLQuery(serviceURL, map[string]string{"school_id": strconv.FormatInt(schoolID, 10)})
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS回调地址配置错误")
	}

	validateURL, err := appendURLQuery(strings.TrimRight(serverURL, "/")+"/serviceValidate", map[string]string{
		"ticket":  ticket,
		"service": callbackURL,
	})
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS校验地址生成失败")
	}

	resp, err := httpclient.SafeGet(ctx, validateURL)
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("学校认证服务暂时不可用，请使用手机号密码登录")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("读取CAS响应失败")
	}

	var parsed casAuthResponse
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("解析CAS响应失败")
	}
	if parsed.AuthenticationSuccess == nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS认证失败")
	}

	if userIDAttr != "" {
		for _, attr := range parsed.AuthenticationSuccess.Attributes {
			if attr.XMLName.Local == userIDAttr && strings.TrimSpace(attr.Value) != "" {
				return strings.TrimSpace(attr.Value), nil
			}
		}
	}

	userID := strings.TrimSpace(parsed.AuthenticationSuccess.User)
	if userID == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("CAS返回的用户标识为空")
	}
	return userID, nil
}

func (s *authService) handleOAuth2Callback(ctx context.Context, schoolID int64, query map[string]string, config map[string]interface{}) (string, error) {
	code := strings.TrimSpace(query["code"])
	state := strings.TrimSpace(query["state"])
	if code == "" || state == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("缺少OAuth2回调参数")
	}

	cachedSchoolID, err := cache.GetString(ctx, cache.KeySSOState+state)
	if err != nil || cachedSchoolID != strconv.FormatInt(schoolID, 10) {
		return "", errcode.ErrSSOAuthFailed.WithMessage("SSO状态校验失败")
	}
	_ = cache.Del(ctx, cache.KeySSOState+state)

	tokenURL, _ := config["token_url"].(string)
	userInfoURL, _ := config["userinfo_url"].(string)
	clientID, _ := config["client_id"].(string)
	clientSecretCipher, _ := config["client_secret"].(string)
	redirectURI, _ := config["redirect_uri"].(string)
	userIDAttr, _ := config["user_id_attribute"].(string)
	if tokenURL == "" || userInfoURL == "" || clientID == "" || clientSecretCipher == "" || redirectURI == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("OAuth2配置不完整")
	}

	clientSecret, err := crypto.AESDecrypt(clientSecretCipher)
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("OAuth2密钥解密失败")
	}

	tokenResp, err := postFormJSON(ctx, tokenURL, url.Values{
		"grant_type":    []string{"authorization_code"},
		"code":          []string{code},
		"client_id":     []string{clientID},
		"client_secret": []string{clientSecret},
		"redirect_uri":  []string{redirectURI},
	})
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("获取OAuth2访问令牌失败")
	}

	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("OAuth2未返回访问令牌")
	}

	userInfo, err := getJSONWithBearer(ctx, userInfoURL, accessToken)
	if err != nil {
		return "", errcode.ErrSSOAuthFailed.WithMessage("获取OAuth2用户信息失败")
	}

	ssoUserID := extractJSONValue(userInfo, userIDAttr)
	if ssoUserID == "" {
		return "", errcode.ErrSSOAuthFailed.WithMessage("OAuth2返回的用户标识为空")
	}
	return ssoUserID, nil
}

func (s *authService) matchSSOUser(ctx context.Context, schoolID int64, provider, ssoUserID string) (*entity.User, error) {
	if s.ssoBindingRepo != nil {
		binding, err := s.ssoBindingRepo.GetBySchoolAndSSOUserID(ctx, schoolID, ssoUserID)
		if err == nil && binding != nil {
			user, userErr := s.userRepo.GetByID(ctx, binding.UserID)
			if userErr == nil {
				now := time.Now()
				_ = s.ssoBindingRepo.UpdateLastLoginAt(ctx, binding.ID, now)
				return user, nil
			}
		}
	}

	user, err := s.userRepo.GetBySchoolAndStudentNo(ctx, schoolID, ssoUserID)
	if err != nil {
		return nil, errcode.ErrSSOAccountNotFound.WithMessage("账号未开通，请联系管理员")
	}

	fullUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, errcode.ErrSSOAccountNotFound.WithMessage("账号未开通，请联系管理员")
	}

	if s.ssoBindingRepo != nil {
		now := time.Now()
		_ = s.ssoBindingRepo.Upsert(ctx, &entity.UserSSOBinding{
			UserID:      fullUser.ID,
			SchoolID:    schoolID,
			SSOProvider: provider,
			SSOUserID:   ssoUserID,
			LastLoginAt: &now,
		})
	}

	return fullUser, nil
}

func (s *authService) completeSSOLogin(ctx context.Context, user *entity.User, provider, ip, userAgent string) (*LoginResult, error) {
	if s.schoolStatusChecker != nil {
		if err := s.schoolStatusChecker.CheckLoginAllowed(ctx, user.SchoolID); err != nil {
			return nil, err
		}
	}

	switch user.Status {
	case enum.UserStatusDisabled:
		return nil, errcode.ErrAccountDisabled
	case enum.UserStatusArchived:
		return nil, errcode.ErrAccountArchived
	}

	loginMethod := enum.LoginMethodSSOOAuth
	if provider == "cas" {
		loginMethod = enum.LoginMethodSSOCAS
	}

	if user.IsFirstLogin {
		tempToken, err := jwtpkg.GenerateTempToken(user.ID)
		if err != nil {
			return nil, errcode.ErrInternal.WithMessage("生成临时Token失败")
		}
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionSuccess, loginMethod, ip, userAgent, "SSO首次登录，待改密")
		return &LoginResult{
			IsFirstLogin: true,
			ForceResp: &dto.ForceChangePasswordResp{
				ForceChangePassword: true,
				TempToken:           tempToken,
				TempTokenExpiresIn:  300,
			},
		}, nil
	}

	roleCodes := s.extractRoleCodes(user)
	tokenPair, err := jwtpkg.GenerateTokenPair(user.ID, user.SchoolID, roleCodes)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("生成Token失败")
	}

	s.storeSession(ctx, user.ID, tokenPair.RefreshToken, ip)
	now := time.Now()
	_ = s.userRepo.UpdateLoginInfo(ctx, user.ID, ip, now)
	asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionSuccess, loginMethod, ip, userAgent, "")

	schoolName := ""
	if s.schoolNameQuerier != nil {
		schoolName = s.schoolNameQuerier.GetSchoolName(ctx, user.SchoolID)
	}

	return &LoginResult{
		IsFirstLogin: false,
		TokenResp: &dto.LoginResp{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresIn:    tokenPair.ExpiresIn,
			TokenType:    "Bearer",
			User: dto.LoginUser{
				ID:           strconv.FormatInt(user.ID, 10),
				Name:         user.Name,
				Phone:        user.Phone,
				Roles:        roleCodes,
				SchoolID:     strconv.FormatInt(user.SchoolID, 10),
				SchoolName:   schoolName,
				IsFirstLogin: false,
			},
		},
	}, nil
}

func appendURLQuery(rawURL string, params map[string]string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	for key, value := range params {
		if value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func postFormJSON(ctx context.Context, rawURL string, form url.Values) (map[string]interface{}, error) {
	resp, err := httpclient.SafePostForm(ctx, rawURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func getJSONWithBearer(ctx context.Context, rawURL, token string) (map[string]interface{}, error) {
	resp, err := httpclient.SafeGetWithBearer(ctx, rawURL, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func extractJSONValue(data map[string]interface{}, key string) string {
	if key == "" {
		key = "sub"
	}
	value, ok := data[key]
	if !ok {
		return ""
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}
