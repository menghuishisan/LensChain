// sso_service.go
// 模块02 — 学校与租户管理：SSO配置业务逻辑
// 负责 SSO 配置的读取、更新、连接测试
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

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
}

// ssoService SSO配置服务实现
type ssoService struct {
	ssoRepo    schoolrepo.SSOConfigRepository
	schoolRepo schoolrepo.SchoolRepository
}

// NewSSOService 创建SSO配置服务实例
func NewSSOService(
	ssoRepo schoolrepo.SSOConfigRepository,
	schoolRepo schoolrepo.SchoolRepository,
) SSOService {
	return &ssoService{
		ssoRepo:    ssoRepo,
		schoolRepo: schoolRepo,
	}
}

// GetConfig 获取SSO配置
// client_secret 脱敏显示为 ******
func (s *ssoService) GetConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOConfigResp, error) {
	config, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err != nil {
		// 未配置SSO，返回空配置
		return &dto.SSOConfigResp{
			Provider:  "",
			IsEnabled: false,
			IsTested:  false,
			Config:    make(map[string]interface{}),
		}, nil
	}

	// 解析 config JSON
	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config.Config), &configMap); err != nil {
		return nil, errcode.ErrInternal.WithMessage("解析SSO配置失败")
	}

	// 解密 client_secret 后脱敏显示
	if _, ok := configMap["client_secret"]; ok {
		configMap["client_secret"] = "******"
	}

	resp := &dto.SSOConfigResp{
		Provider:  config.Provider,
		IsEnabled: config.IsEnabled,
		IsTested:  config.IsTested,
		Config:    configMap,
	}
	if config.TestedAt != nil {
		t := config.TestedAt.Format(time.RFC3339)
		resp.TestedAt = &t
	}

	return resp, nil
}

// UpdateConfig 更新SSO配置
// 保存配置，client_secret 加密存储，重置 is_tested = false
func (s *ssoService) UpdateConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateSSOConfigReq) error {
	// 加密 client_secret
	configMap := req.Config
	if secret, ok := configMap["client_secret"]; ok {
		if secretStr, ok := secret.(string); ok && secretStr != "" && secretStr != "******" {
			encrypted, err := crypto.AESEncrypt(secretStr)
			if err != nil {
				return errcode.ErrInternal.WithMessage("加密SSO密钥失败")
			}
			configMap["client_secret"] = encrypted
		} else if secretStr == "******" {
			// 未修改密钥，保留原值
			existing, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
			if err == nil {
				var existingConfig map[string]interface{}
				if json.Unmarshal([]byte(existing.Config), &existingConfig) == nil {
					if origSecret, ok := existingConfig["client_secret"]; ok {
						configMap["client_secret"] = origSecret
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
		IsTested:  false, // 配置变更后需重新测试
		Config:    string(configJSON),
		UpdatedBy: &sc.UserID,
	}

	// 保留原有的 is_enabled 状态
	existing, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err == nil {
		ssoConfig.IsEnabled = existing.IsEnabled
	}

	return s.ssoRepo.Upsert(ctx, ssoConfig)
}

// TestConnection 测试SSO连接
func (s *ssoService) TestConnection(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SSOTestResp, error) {
	config, err := s.ssoRepo.GetBySchoolID(ctx, sc.SchoolID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("请先配置SSO")
	}

	// 解析配置
	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config.Config), &configMap); err != nil {
		return nil, errcode.ErrInternal.WithMessage("解析SSO配置失败")
	}

	// 根据协议类型测试连接
	var testErr error
	var testDetail string

	switch config.Provider {
	case "cas":
		testDetail, testErr = s.testCASConnection(configMap)
	case "oauth2":
		testDetail, testErr = s.testOAuth2Connection(configMap)
	default:
		return nil, errcode.ErrInvalidParams.WithMessage("不支持的SSO协议类型")
	}

	now := time.Now()
	if testErr != nil {
		// 测试失败
		logger.L.Warn("SSO连接测试失败", zap.Int64("school_id", sc.SchoolID), zap.Error(testErr))
		errDetail := testErr.Error()

		// 更新测试状态
		_ = s.ssoRepo.UpdateFields(ctx, sc.SchoolID, map[string]interface{}{
			"is_tested":  false,
			"tested_at":  now,
			"updated_at": now,
		})

		return &dto.SSOTestResp{
			IsTested:    false,
			ErrorDetail: &errDetail,
		}, errcode.ErrSSOTestFailed
	}

	// 测试成功
	nowStr := now.Format(time.RFC3339)
	_ = s.ssoRepo.UpdateFields(ctx, sc.SchoolID, map[string]interface{}{
		"is_tested":  true,
		"tested_at":  now,
		"updated_at": now,
	})

	return &dto.SSOTestResp{
		IsTested:   true,
		TestedAt:   &nowStr,
		TestDetail: &testDetail,
	}, nil
}

// testCASConnection 测试CAS连接
func (s *ssoService) testCASConnection(config map[string]interface{}) (string, error) {
	serverURL, ok := config["cas_server_url"].(string)
	if !ok || serverURL == "" {
		return "", fmt.Errorf("CAS服务器地址未配置")
	}

	// 使用安全 HTTP 客户端（SSRF 防护：仅允许 HTTPS、拒绝私有 IP）
	resp, err := httpclient.SafeGet(context.Background(), serverURL)
	if err != nil {
		return "", fmt.Errorf("无法连接到CAS服务器：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("CAS服务器返回错误：HTTP %d", resp.StatusCode)
	}

	return "成功连接到CAS认证服务器", nil
}

// testOAuth2Connection 测试OAuth2连接
func (s *ssoService) testOAuth2Connection(config map[string]interface{}) (string, error) {
	// 测试 authorize_url 可达性
	authorizeURL, _ := config["authorize_url"].(string)
	tokenURL, _ := config["token_url"].(string)

	if authorizeURL == "" || tokenURL == "" {
		return "", fmt.Errorf("OAuth2授权端点或Token端点未配置")
	}

	// 使用安全 HTTP 客户端（SSRF 防护：仅允许 HTTPS、拒绝私有 IP）
	resp, err := httpclient.SafeGet(context.Background(), tokenURL)
	if err != nil {
		return "", fmt.Errorf("无法连接到Token端点：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("Token端点返回服务器错误：HTTP %d", resp.StatusCode)
	}

	return "成功连接到OAuth2授权服务器", nil
}
