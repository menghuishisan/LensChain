// sso_service.go
// 模块02 — 学校与租户管理：SSO配置业务逻辑
// 负责 SSO 配置的读取、更新、连接测试
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/httpclient"
	"github.com/lenschain/backend/internal/pkg/logger"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
)

// SSOService SSO配置服务接口
type SSOService interface {
	GetConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOConfigResp, error)
	UpdateConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateSSOConfigReq) error
	TestConnection(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOTestResp, error)
	ToggleEnable(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ToggleSSOEnableReq) error
}

// ssoService SSO配置服务实现
type ssoService struct {
	ssoRepo schoolrepo.SSOConfigRepository
}

// NewSSOService 创建SSO配置服务实例
func NewSSOService(
	ssoRepo schoolrepo.SSOConfigRepository,
) SSOService {
	return &ssoService{
		ssoRepo: ssoRepo,
	}
}

// GetConfig 获取SSO配置
// client_secret 脱敏显示为 ******。
func (s *ssoService) GetConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOConfigResp, error) {
	config, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrInternal.WithMessage("查询SSO配置失败")
		}
		return &dto.SSOConfigResp{
			Provider:  "",
			IsEnabled: false,
			IsTested:  false,
			Config:    &dto.SSOConfig{},
		}, nil
	}

	configMap := make(map[string]interface{})
	if err := json.Unmarshal(config.Config, &configMap); err != nil {
		return nil, errcode.ErrInternal.WithMessage("解析SSO配置失败")
	}

	if _, ok := configMap["client_secret"]; ok {
		configMap["client_secret"] = "******"
	}

	resp := &dto.SSOConfigResp{
		Provider:  config.Provider,
		IsEnabled: config.IsEnabled,
		IsTested:  config.IsTested,
		Config:    buildSSOConfigDTO(configMap),
	}
	if config.TestedAt != nil {
		testedAt := config.TestedAt.Format(time.RFC3339)
		resp.TestedAt = &testedAt
	}
	return resp, nil
}

// UpdateConfig 更新SSO配置
// 保存配置，client_secret 加密存储，配置变更后重置测试状态。
func (s *ssoService) UpdateConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateSSOConfigReq) error {
	if err := validateSSOConfig(req.Provider, req.Config); err != nil {
		return err
	}

	configMap := buildSSOConfigMap(req.Config)
	if secret, ok := configMap["client_secret"].(string); ok {
		switch {
		case secret != "" && secret != "******":
			encrypted, err := crypto.AESEncrypt(secret)
			if err != nil {
				return errcode.ErrInternal.WithMessage("加密SSO密钥失败")
			}
			configMap["client_secret"] = encrypted
		case secret == "******":
			existing, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
			if err == nil {
				var existingConfig map[string]interface{}
				if json.Unmarshal(existing.Config, &existingConfig) == nil {
					if originalSecret, found := existingConfig["client_secret"]; found {
						configMap["client_secret"] = originalSecret
					}
				}
			}
		}
	}

	configJSON, err := json.Marshal(configMap)
	if err != nil {
		return errcode.ErrInternal.WithMessage("序列化SSO配置失败")
	}

	ssoConfig := &entity.SchoolSSOConfig{
		SchoolID:  sc.SchoolID,
		Provider:  req.Provider,
		IsTested:  false,
		Config:    datatypes.JSON(configJSON),
		UpdatedBy: &sc.UserID,
	}

	existing, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err == nil {
		ssoConfig.IsEnabled = existing.IsEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.ErrInternal.WithMessage("查询现有SSO配置失败")
	}

	return s.ssoRepo.Upsert(ctx, ssoConfig)
}

// TestConnection 测试SSO连接
func (s *ssoService) TestConnection(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOTestResp, error) {
	config, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrInternal.WithMessage("查询SSO配置失败")
		}
		return nil, errcode.ErrInvalidParams.WithMessage("请先完善SSO配置")
	}

	configMap := make(map[string]interface{})
	if err := json.Unmarshal(config.Config, &configMap); err != nil {
		return nil, errcode.ErrInternal.WithMessage("解析SSO配置失败")
	}

	var (
		testErr    error
		testDetail string
	)
	switch config.Provider {
	case "cas":
		testDetail, testErr = s.testCASConnection(ctx, configMap)
	case "oauth2":
		testDetail, testErr = s.testOAuth2Connection(ctx, configMap)
	default:
		return nil, errcode.ErrInvalidParams.WithMessage("不支持的SSO协议类型")
	}

	now := time.Now()
	if testErr != nil {
		logger.L.Warn("SSO连接测试失败", zap.Int64("school_id", sc.SchoolID), zap.Error(testErr))
		errorDetail := testErr.Error()
		_ = s.ssoRepo.UpdateTestResult(ctx, sc.SchoolID, false, now)
		return &dto.SSOTestResp{
			IsTested:    false,
			ErrorDetail: &errorDetail,
		}, errcode.ErrSSOTestFailed
	}

	_ = s.ssoRepo.UpdateTestResult(ctx, sc.SchoolID, true, now)
	testedAt := now.Format(time.RFC3339)
	return &dto.SSOTestResp{
		IsTested:   true,
		TestedAt:   &testedAt,
		TestDetail: &testDetail,
	}, nil
}

// ToggleEnable 启用或禁用SSO
// 仅当配置已测试通过时允许启用。
func (s *ssoService) ToggleEnable(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ToggleSSOEnableReq) error {
	config, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrInternal.WithMessage("查询SSO配置失败")
		}
		return errcode.ErrSSOConfigNotFound
	}

	if req.IsEnabled == nil {
		return errcode.ErrInvalidParams.WithMessage("is_enabled 不能为空")
	}

	if *req.IsEnabled && !config.IsTested {
		return errcode.ErrSSONotTested
	}

	return s.ssoRepo.ToggleEnabled(ctx, sc.SchoolID, *req.IsEnabled, sc.UserID)
}

// testCASConnection 测试CAS连接。
// 使用调用链上下文，确保超时、取消信号能透传到底层 HTTP 客户端。
func (s *ssoService) testCASConnection(ctx context.Context, config map[string]interface{}) (string, error) {
	serverURL, ok := config["cas_server_url"].(string)
	if !ok || serverURL == "" {
		return "", fmt.Errorf("请先完善SSO配置")
	}

	resp, err := httpclient.SafeGet(ctx, serverURL)
	if err != nil {
		return "", fmt.Errorf("无法连接到CAS服务器：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("CAS服务器返回错误：HTTP %d", resp.StatusCode)
	}

	return "成功连接到CAS认证服务器", nil
}

// testOAuth2Connection 测试OAuth2连接。
// 使用调用链上下文，确保超时、取消信号能透传到底层 HTTP 客户端。
func (s *ssoService) testOAuth2Connection(ctx context.Context, config map[string]interface{}) (string, error) {
	tokenURL, _ := config["token_url"].(string)
	if tokenURL == "" {
		return "", fmt.Errorf("请先完善SSO配置")
	}

	resp, err := httpclient.SafeGet(ctx, tokenURL)
	if err != nil {
		return "", fmt.Errorf("无法连接到Token端点：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("Token端点返回服务器错误：HTTP %d", resp.StatusCode)
	}

	return "成功连接到OAuth2授权服务器", nil
}

// validateSSOConfig 按协议校验 SSO 配置必填项。
// DTO 只负责字段格式校验，协议级必填规则由 service 按文档统一判断。
func validateSSOConfig(provider string, config *dto.SSOConfig) error {
	if config == nil {
		return errcode.ErrInvalidParams.WithMessage("请先完善SSO配置")
	}

	switch provider {
	case "cas":
		if isBlankString(config.CASServerURL) || isBlankString(config.CASServiceURL) || config.UserIDAttribute == "" {
			return errcode.ErrInvalidParams.WithMessage("请先完善SSO配置")
		}
	case "oauth2":
		if isBlankString(config.AuthorizeURL) ||
			isBlankString(config.TokenURL) ||
			isBlankString(config.UserinfoURL) ||
			isBlankString(config.ClientID) ||
			isBlankString(config.ClientSecret) ||
			isBlankString(config.RedirectURI) ||
			config.UserIDAttribute == "" {
			return errcode.ErrInvalidParams.WithMessage("请先完善SSO配置")
		}
	default:
		return errcode.ErrInvalidParams.WithMessage("不支持的SSO协议类型")
	}

	return nil
}

// isBlankString 判断可选字符串字段是否为空。
func isBlankString(value *string) bool {
	return value == nil || *value == ""
}

// buildSSOConfigDTO 将持久化配置转换为对外响应 DTO。
func buildSSOConfigDTO(configMap map[string]interface{}) *dto.SSOConfig {
	config := &dto.SSOConfig{}
	assignStringField(&config.CASServerURL, configMap["cas_server_url"])
	assignStringField(&config.CASServiceURL, configMap["cas_service_url"])
	assignStringField(&config.CASVersion, configMap["cas_version"])
	assignStringField(&config.AuthorizeURL, configMap["authorize_url"])
	assignStringField(&config.TokenURL, configMap["token_url"])
	assignStringField(&config.UserinfoURL, configMap["userinfo_url"])
	assignStringField(&config.ClientID, configMap["client_id"])
	assignStringField(&config.ClientSecret, configMap["client_secret"])
	assignStringField(&config.RedirectURI, configMap["redirect_uri"])
	assignStringField(&config.Scope, configMap["scope"])
	if value, ok := configMap["user_id_attribute"].(string); ok {
		config.UserIDAttribute = value
	}
	return config
}

// buildSSOConfigMap 将请求 DTO 转换为持久化配置映射，只保留非空字段。
func buildSSOConfigMap(config *dto.SSOConfig) map[string]interface{} {
	result := make(map[string]interface{})
	if config == nil {
		return result
	}

	setOptionalString(result, "cas_server_url", config.CASServerURL)
	setOptionalString(result, "cas_service_url", config.CASServiceURL)
	setOptionalString(result, "cas_version", config.CASVersion)
	setOptionalString(result, "authorize_url", config.AuthorizeURL)
	setOptionalString(result, "token_url", config.TokenURL)
	setOptionalString(result, "userinfo_url", config.UserinfoURL)
	setOptionalString(result, "client_id", config.ClientID)
	setOptionalString(result, "client_secret", config.ClientSecret)
	setOptionalString(result, "redirect_uri", config.RedirectURI)
	setOptionalString(result, "scope", config.Scope)
	if config.UserIDAttribute != "" {
		result["user_id_attribute"] = config.UserIDAttribute
	}
	return result
}

// assignStringField 把持久化配置中的字符串写回 DTO 可选字段。
func assignStringField(target **string, value interface{}) {
	text, ok := value.(string)
	if !ok || text == "" {
		return
	}
	*target = &text
}

// setOptionalString 把 DTO 可选字符串字段写入持久化配置映射。
func setOptionalString(target map[string]interface{}, key string, value *string) {
	if value == nil || *value == "" {
		return
	}
	target[key] = *value
}
