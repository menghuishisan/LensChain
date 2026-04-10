// sms.go
// SMS 短信网关接口封装
// 支持 mock（开发环境）/ aliyun / tencent 三种提供商
// 用于：学校审核通知、账号创建通知、密码重置通知、到期提醒等

package sms

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
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
	TemplateSchoolApproved    = "school_approved"     // 学校审核通过
	TemplateSchoolRejected    = "school_rejected"     // 学校审核拒绝
	TemplateAccountCreated    = "account_created"     // 账号创建通知
	TemplatePasswordReset     = "password_reset"      // 密码重置通知
	TemplateLicenseExpiring   = "license_expiring"    // 授权即将到期
	TemplateLicenseExpired    = "license_expired"     // 授权已过期
	TemplateSchoolFrozen      = "school_frozen"       // 学校冻结通知
	TemplateSMSVerification   = "sms_verification"    // 短信验证码
)
