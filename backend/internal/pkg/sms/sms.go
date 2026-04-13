// sms.go
// SMS 短信网关接口封装
// 支持 mock（开发环境）/ aliyun / tencent 三种提供商
// 用于：学校审核通知、账号创建通知、密码重置通知、到期提醒等

package sms

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// Sender 短信发送接口
type Sender interface {
	// Send 发送短信
	// phone: 手机号
	// templateCode: 短信模板编码
	// params: 模板参数（键值对）
	Send(phone, templateCode string, params map[string]string) error
}

// 全局短信发送器
var sender Sender

const (
	verificationCodeTTL      = 5 * time.Minute
	verificationCodeCooldown = 1 * time.Minute
)

// Init 初始化短信网关
func Init(cfg *config.SMSConfig) error {
	switch cfg.Provider {
	case "mock":
		sender = &mockSender{}
		logger.L.Info("短信网关初始化完成（Mock模式）")
	case "aliyun":
		sender = &aliyunSender{
			accessKey: cfg.AccessKey,
			secretKey: cfg.SecretKey,
			signName:  cfg.SignName,
			region:    cfg.Region,
		}
		logger.L.Info("短信网关初始化完成（阿里云）")
	case "tencent":
		sender = &tencentSender{
			accessKey: cfg.AccessKey,
			secretKey: cfg.SecretKey,
			signName:  cfg.SignName,
			region:    cfg.Region,
		}
		logger.L.Info("短信网关初始化完成（腾讯云）")
	default:
		return fmt.Errorf("不支持的短信提供商: %s", cfg.Provider)
	}
	return nil
}

// Send 发送短信（全局函数）
func Send(phone, templateCode string, params map[string]string) error {
	if sender == nil {
		return fmt.Errorf("短信网关未初始化")
	}
	return sender.Send(phone, templateCode, params)
}

// VerifyCode 校验短信验证码
// mock 模式下允许固定验证码 123456，便于联调；真实缓存命中后会立即删除，避免重复使用
func VerifyCode(ctx context.Context, phone, code string) error {
	if strings.TrimSpace(phone) == "" || strings.TrimSpace(code) == "" {
		return fmt.Errorf("手机号或验证码不能为空")
	}

	if _, ok := sender.(*mockSender); ok && code == "123456" {
		return nil
	}

	cachedCode, err := cache.GetString(ctx, cache.KeySMSVerification+phone)
	if err != nil {
		return fmt.Errorf("验证码不存在或已过期")
	}
	if strings.TrimSpace(cachedCode) != code {
		return fmt.Errorf("验证码错误")
	}

	_ = cache.Del(ctx, cache.KeySMSVerification+phone)
	return nil
}

// SendVerificationCode 发送短信验证码并写入缓存
// 统一实现验证码缓存与发送冷却，供公开查询/重申场景复用。
func SendVerificationCode(ctx context.Context, phone string) error {
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("手机号不能为空")
	}

	cooldownKey := cache.KeySMSVerificationCooldown + phone
	ok, err := cache.SetNX(ctx, cooldownKey, "1", verificationCodeCooldown)
	if err != nil {
		return fmt.Errorf("设置短信发送冷却失败: %w", err)
	}
	if !ok {
		return fmt.Errorf("短信发送过于频繁")
	}

	code, err := generateVerificationCode()
	if err != nil {
		_ = cache.Del(ctx, cooldownKey)
		return fmt.Errorf("生成验证码失败: %w", err)
	}

	if err := cache.Set(ctx, cache.KeySMSVerification+phone, code, verificationCodeTTL); err != nil {
		_ = cache.Del(ctx, cooldownKey)
		return fmt.Errorf("缓存验证码失败: %w", err)
	}

	if err := Send(phone, TemplateSMSVerification, map[string]string{
		"code": code,
	}); err != nil {
		_ = cache.Del(ctx, cache.KeySMSVerification+phone)
		_ = cache.Del(ctx, cooldownKey)
		return err
	}

	return nil
}

// generateVerificationCode 生成 6 位数字验证码
func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ---- Mock 实现（开发环境） ----

type mockSender struct{}

func (s *mockSender) Send(phone, templateCode string, params map[string]string) error {
	logger.L.Info("【Mock短信】发送成功",
		zap.String("phone", phone),
		zap.String("template", templateCode),
		zap.Any("params", params),
	)
	return nil
}

// ---- 阿里云实现（预留） ----

type aliyunSender struct {
	accessKey string
	secretKey string
	signName  string
	region    string
}

func (s *aliyunSender) Send(phone, templateCode string, params map[string]string) error {
	// TODO: 接入阿里云短信 SDK
	logger.L.Info("【阿里云短信】发送",
		zap.String("phone", phone),
		zap.String("template", templateCode),
	)
	return nil
}

// ---- 腾讯云实现（预留） ----

type tencentSender struct {
	accessKey string
	secretKey string
	signName  string
	region    string
}

func (s *tencentSender) Send(phone, templateCode string, params map[string]string) error {
	// TODO: 接入腾讯云短信 SDK
	logger.L.Info("【腾讯云短信】发送",
		zap.String("phone", phone),
		zap.String("template", templateCode),
	)
	return nil
}

// ---- 短信模板编码常量 ----

const (
	TemplateSchoolApproved  = "school_approved"  // 学校审核通过
	TemplateSchoolRejected  = "school_rejected"  // 学校审核拒绝
	TemplateAccountCreated  = "account_created"  // 账号创建通知
	TemplatePasswordReset   = "password_reset"   // 密码重置通知
	TemplateLicenseExpiring = "license_expiring" // 授权即将到期
	TemplateLicenseExpired  = "license_expired"  // 授权已过期
	TemplateSchoolFrozen    = "school_frozen"    // 学校冻结通知
	TemplateSMSVerification = "sms_verification" // 短信验证码
)
