// sms.go
// 该文件实现平台统一的短信网关抽象，负责验证码发送、短信模板参数组装和不同云厂商的
// 请求签名。学校入驻申请、审核通知、账号创建通知、授权到期提醒等所有短信能力，都应
// 通过这里接入，后续只替换配置与模板编码，不再改上层业务代码。

package sms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// Sender 短信发送接口。
type Sender interface {
	// Send 发送短信。
	// phone: 国内 11 位手机号
	// templateCode: 模板编码；阿里云为模板 Code，腾讯云为模板 ID
	// params: 模板参数。阿里云按命名参数序列化为 JSON；腾讯云按键名排序后转为参数列表。
	Send(phone, templateCode string, params map[string]string) error
}

const (
	providerMock    = "mock"
	providerAliyun  = "aliyun"
	providerTencent = "tencent"
)

var (
	// ErrProviderNotSupported 表示不支持的短信提供商。
	ErrProviderNotSupported = errors.New("不支持的短信提供商")
	// ErrGatewayNotReady 表示短信网关尚未初始化。
	ErrGatewayNotReady = errors.New("短信网关未初始化")
	// ErrTemplateCodeRequired 表示模板编码不能为空。
	ErrTemplateCodeRequired = errors.New("短信模板编码不能为空")
	mainlandPhonePattern    = regexp.MustCompile(`^1[3-9]\d{9}$`)
	sender                  Sender
	currentProvider         = providerMock
)

const (
	verificationCodeTTL      = 5 * time.Minute
	verificationCodeCooldown = 1 * time.Minute
)

// Init 初始化短信网关。
func Init(cfg *config.SMSConfig) error {
	if cfg == nil {
		return fmt.Errorf("短信配置不能为空")
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = providerMock
	}

	switch provider {
	case providerMock:
		sender = &mockSender{}
		currentProvider = providerMock
		logger.L.Info("短信网关初始化完成（Mock模式）")
		return nil
	case providerAliyun:
		aliyun, err := newAliyunSender(cfg)
		if err != nil {
			return err
		}
		sender = aliyun
		currentProvider = providerAliyun
		logger.L.Info("短信网关初始化完成（阿里云）",
			zap.String("endpoint", aliyun.endpoint),
			zap.String("region", aliyun.region),
		)
		return nil
	case providerTencent:
		tencent, err := newTencentSender(cfg)
		if err != nil {
			return err
		}
		sender = tencent
		currentProvider = providerTencent
		logger.L.Info("短信网关初始化完成（腾讯云）",
			zap.String("endpoint", tencent.endpoint),
			zap.String("region", tencent.region),
		)
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrProviderNotSupported, cfg.Provider)
	}
}

// Send 发送短信（全局函数）。
func Send(phone, templateCode string, params map[string]string) error {
	if sender == nil {
		return ErrGatewayNotReady
	}
	if err := validatePhone(phone); err != nil {
		return err
	}
	if strings.TrimSpace(templateCode) == "" {
		return ErrTemplateCodeRequired
	}
	return sender.Send(phone, templateCode, params)
}

// VerifyCode 校验短信验证码。
// mock 模式下允许固定验证码 123456，便于联调；真实缓存命中后会立即删除，避免重复使用。
func VerifyCode(ctx context.Context, phone, code string) error {
	if strings.TrimSpace(phone) == "" || strings.TrimSpace(code) == "" {
		return fmt.Errorf("手机号或验证码不能为空")
	}
	if err := validatePhone(phone); err != nil {
		return err
	}

	if currentProvider == providerMock && code == "123456" {
		return nil
	}

	cachedCode, err := cache.GetString(ctx, cache.KeySMSVerification+phone)
	if err != nil {
		return fmt.Errorf("验证码不存在或已过期")
	}
	if strings.TrimSpace(cachedCode) != strings.TrimSpace(code) {
		return fmt.Errorf("验证码错误")
	}

	_ = cache.Del(ctx, cache.KeySMSVerification+phone)
	return nil
}

// SendVerificationCode 发送短信验证码并写入缓存。
// 统一实现验证码缓存与发送冷却，供公开查询/重申场景复用。
func SendVerificationCode(ctx context.Context, phone string) error {
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("手机号不能为空")
	}
	if err := validatePhone(phone); err != nil {
		return err
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

	if err := Send(phone, TemplateSMSVerification, map[string]string{"code": code}); err != nil {
		_ = cache.Del(ctx, cache.KeySMSVerification+phone)
		_ = cache.Del(ctx, cooldownKey)
		return err
	}

	return nil
}

// generateVerificationCode 生成 6 位数字验证码。
func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

type mockSender struct{}

func (s *mockSender) Send(phone, templateCode string, params map[string]string) error {
	logger.L.Info("【Mock短信】发送成功",
		zap.String("phone", phone),
		zap.String("template", templateCode),
		zap.Any("params", params),
	)
	return nil
}

type aliyunSender struct {
	accessKey string
	secretKey string
	signName  string
	region    string
	endpoint  string
	client    *http.Client
}

func newAliyunSender(cfg *config.SMSConfig) (*aliyunSender, error) {
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("阿里云短信 access_key 和 secret_key 不能为空")
	}
	if strings.TrimSpace(cfg.SignName) == "" {
		return nil, fmt.Errorf("阿里云短信 sign_name 不能为空")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com"
	}
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "cn-hangzhou"
	}

	return &aliyunSender{
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		signName:  cfg.SignName,
		region:    region,
		endpoint:  endpoint,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Send 按阿里云短信 OpenAPI 的公共参数规则组装并签名请求。
// 这里统一把模板参数序列化为 JSON，保证模块层只传业务参数，不接触云厂商签名细节。
func (s *aliyunSender) Send(phone, templateCode string, params map[string]string) error {
	body := url.Values{
		"Action":           {"SendSms"},
		"AccessKeyId":      {s.accessKey},
		"Format":           {"JSON"},
		"RegionId":         {s.region},
		"SignName":         {s.signName},
		"SignatureMethod":  {"HMAC-SHA1"},
		"SignatureNonce":   {uuid.NewString()},
		"SignatureVersion": {"1.0"},
		"Timestamp":        {time.Now().UTC().Format("2006-01-02T15:04:05Z")},
		"TemplateCode":     {templateCode},
		"Version":          {"2017-05-25"},
		"PhoneNumbers":     {phone},
	}

	if len(params) > 0 {
		payload, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("序列化阿里云短信模板参数失败: %w", err)
		}
		body.Set("TemplateParam", string(payload))
	}

	body.Set("Signature", s.signAliyun(body))

	req, err := http.NewRequest(http.MethodPost, s.endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("创建阿里云短信请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送阿里云短信请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取阿里云短信响应失败: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("阿里云短信请求失败，HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析阿里云短信响应失败: %w", err)
	}
	if strings.ToUpper(result.Code) != "OK" {
		return fmt.Errorf("阿里云短信发送失败[%s]: %s", result.Code, strings.TrimSpace(result.Message))
	}

	logger.L.Info("阿里云短信发送成功",
		zap.String("phone", phone),
		zap.String("template", templateCode),
		zap.String("request_id", result.RequestID),
	)
	return nil
}

// signAliyun 生成阿里云短信接口要求的 HMAC-SHA1 签名。
// 签名必须基于排序后的查询参数计算，否则服务端会直接拒绝请求。
func (s *aliyunSender) signAliyun(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "Signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, aliyunPercentEncode(key)+"="+aliyunPercentEncode(values.Get(key)))
	}
	stringToSign := "POST&%2F&" + aliyunPercentEncode(strings.Join(parts, "&"))

	mac := hmac.New(sha1.New, []byte(s.secretKey+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// aliyunPercentEncode 实现阿里云签名规范要求的特殊 URL 编码规则。
func aliyunPercentEncode(s string) string {
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

type tencentSender struct {
	accessKey string
	secretKey string
	signName  string
	region    string
	endpoint  string
	sdkAppID  string
	client    *http.Client
}

func newTencentSender(cfg *config.SMSConfig) (*tencentSender, error) {
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("腾讯云短信 access_key 和 secret_key 不能为空")
	}
	if strings.TrimSpace(cfg.SignName) == "" {
		return nil, fmt.Errorf("腾讯云短信 sign_name 不能为空")
	}
	if strings.TrimSpace(cfg.SDKAppID) == "" {
		return nil, fmt.Errorf("腾讯云短信 sdk_app_id 不能为空")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://sms.tencentcloudapi.com"
	}
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "ap-guangzhou"
	}

	return &tencentSender{
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		signName:  cfg.SignName,
		region:    region,
		endpoint:  endpoint,
		sdkAppID:  cfg.SDKAppID,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Send 按腾讯云 TC3-HMAC-SHA256 规范组装短信请求。
// 模板参数在这里转换成有序数组，避免上层业务感知不同云厂商的参数格式差异。
func (s *tencentSender) Send(phone, templateCode string, params map[string]string) error {
	payload := map[string]interface{}{
		"PhoneNumberSet":   []string{formatTencentPhone(phone)},
		"SmsSdkAppId":      s.sdkAppID,
		"SignName":         s.signName,
		"TemplateId":       templateCode,
		"TemplateParamSet": buildTencentTemplateParams(params),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化腾讯云短信请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建腾讯云短信请求失败: %w", err)
	}

	host := req.URL.Host
	timestamp := time.Now().UTC().Unix()
	req.Header.Set("Authorization", s.buildTencentAuthorization(host, body, timestamp))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", "SendSms")
	req.Header.Set("X-TC-Version", "2021-01-11")
	req.Header.Set("X-TC-Region", s.region)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送腾讯云短信请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取腾讯云短信响应失败: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("腾讯云短信请求失败，HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Response struct {
			RequestID     string `json:"RequestId"`
			Error         *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			SendStatusSet []struct {
				Code        string `json:"Code"`
				Message     string `json:"Message"`
				PhoneNumber string `json:"PhoneNumber"`
				SerialNo    string `json:"SerialNo"`
			} `json:"SendStatusSet"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析腾讯云短信响应失败: %w", err)
	}
	if result.Response.Error != nil {
		return fmt.Errorf("腾讯云短信发送失败[%s]: %s", result.Response.Error.Code, result.Response.Error.Message)
	}
	if len(result.Response.SendStatusSet) == 0 {
		return fmt.Errorf("腾讯云短信返回空发送结果")
	}

	status := result.Response.SendStatusSet[0]
	if !strings.EqualFold(status.Code, "Ok") {
		return fmt.Errorf("腾讯云短信发送失败[%s]: %s", status.Code, status.Message)
	}

	logger.L.Info("腾讯云短信发送成功",
		zap.String("phone", phone),
		zap.String("template", templateCode),
		zap.String("request_id", result.Response.RequestID),
		zap.String("serial_no", status.SerialNo),
	)
	return nil
}

// buildTencentAuthorization 生成腾讯云短信接口所需的 Authorization 头。
// 它把请求体摘要、请求时间和服务名组合成 TC3 签名，保证后续只改配置即可切换到真实发送。
func (s *tencentSender) buildTencentAuthorization(host string, body []byte, timestamp int64) string {
	const (
		service       = "sms"
		algorithm     = "TC3-HMAC-SHA256"
		requestTarget = "tc3_request"
	)

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	hashedPayload := sha256.Sum256(body)
	canonicalHeaders := "content-type:application/json; charset=utf-8\nhost:" + host + "\n"
	signedHeaders := "content-type;host"
	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		hex.EncodeToString(hashedPayload[:]),
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s", date, service, requestTarget)
	hashedCanonicalRequest := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		algorithm,
		strconv.FormatInt(timestamp, 10),
		credentialScope,
		hex.EncodeToString(hashedCanonicalRequest[:]),
	}, "\n")

	secretDate := hmacSHA256([]byte("TC3"+s.secretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, requestTarget)
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	return fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		s.accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)
}

// hmacSHA256 是腾讯云 TC3 签名流程复用的 HMAC-SHA256 计算函数。
func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

// buildTencentTemplateParams 把模板参数转换成腾讯云要求的有序字符串数组。
// 如果上层传入的是数字键名，会优先按数字顺序排列，方便直接表达模板占位顺序。
func buildTencentTemplateParams(params map[string]string) []string {
	if len(params) == 0 {
		return []string{}
	}

	type pair struct {
		key   string
		value string
		order int
	}
	pairs := make([]pair, 0, len(params))
	for key, value := range params {
		if strings.HasPrefix(key, "_") {
			continue
		}
		order := 1<<31 - 1
		if n, err := strconv.Atoi(key); err == nil {
			order = n
		}
		pairs = append(pairs, pair{
			key:   key,
			value: value,
			order: order,
		})
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].order != pairs[j].order {
			return pairs[i].order < pairs[j].order
		}
		return pairs[i].key < pairs[j].key
	})

	result := make([]string, 0, len(pairs))
	for _, item := range pairs {
		result = append(result, item.value)
	}
	return result
}

// formatTencentPhone 将国内 11 位手机号转换成腾讯云要求的 E.164 格式。
func formatTencentPhone(phone string) string {
	if strings.HasPrefix(phone, "+") {
		return phone
	}
	return "+86" + phone
}

const (
	// TemplateSchoolApproved 学校审核通过通知。
	TemplateSchoolApproved = "school_approved"
	// TemplateSchoolRejected 学校审核拒绝通知。
	TemplateSchoolRejected = "school_rejected"
	// TemplateAccountCreated 账号创建通知。
	TemplateAccountCreated = "account_created"
	// TemplatePasswordReset 密码重置通知。
	TemplatePasswordReset = "password_reset"
	// TemplateLicenseExpiring 授权即将到期通知。
	TemplateLicenseExpiring = "license_expiring"
	// TemplateLicenseExpired 授权已过期通知。
	TemplateLicenseExpired = "license_expired"
	// TemplateSchoolFrozen 学校冻结通知。
	TemplateSchoolFrozen = "school_frozen"
	// TemplateSMSVerification 短信验证码。
	TemplateSMSVerification = "sms_verification"
)

// validatePhone 校验短信手机号格式。
func validatePhone(phone string) error {
	if !mainlandPhonePattern.MatchString(strings.TrimSpace(phone)) {
		return fmt.Errorf("手机号格式不正确")
	}
	return nil
}
