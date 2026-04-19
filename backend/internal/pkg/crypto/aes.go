// aes.go
// 该文件提供平台统一的对称加密能力，主要用于保护数据库或配置中不应明文保存的敏感值，
// 例如 SSO 密钥、第三方接入凭据和对象存储访问令牌。这里统一采用 AES-256-GCM，既负责
// 加密也负责完整性校验，避免各模块自行选算法或写出不安全的加密实现。

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"

	"github.com/lenschain/backend/internal/pkg/logger"
)

// aesKey 全局 AES 密钥（32字节 = AES-256）
var (
	aesKey     []byte
	aesKeyErr  error
	aesKeyOnce sync.Once
)

const devAESKeyBase64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// initAESKey 从环境变量加载 AES 密钥。
// 密钥必须是 base64 编码的 32 字节数据；开发环境允许回落到固定开发密钥。
func initAESKey() {
	keyStr := os.Getenv("AES_ENCRYPTION_KEY")
	if keyStr == "" {
		if os.Getenv("GIN_MODE") == "release" {
			aesKeyErr = errors.New("生产环境必须配置 AES_ENCRYPTION_KEY 环境变量")
			return
		}
		// 开发环境使用默认密钥并打印警告
		logger.L.Warn("AES_ENCRYPTION_KEY 未配置，使用开发环境默认密钥（请勿用于生产环境）")
		keyStr = devAESKeyBase64
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		aesKeyErr = fmt.Errorf("AES_ENCRYPTION_KEY base64 解码失败: %w", err)
		return
	}

	// 严格要求 32 字节（AES-256），不做 padding
	if len(key) != 32 {
		aesKeyErr = fmt.Errorf("AES_ENCRYPTION_KEY 必须为 32 字节（当前 %d 字节）", len(key))
		return
	}

	aesKey = key
}

// getAESKey 获取 AES 密钥（懒加载）。
func getAESKey() ([]byte, error) {
	aesKeyOnce.Do(initAESKey)
	if aesKeyErr != nil {
		return nil, aesKeyErr
	}
	return aesKey, nil
}

// AESEncrypt 使用 AES-256-GCM 加密明文
// 返回 base64 编码的密文（包含 nonce）
func AESEncrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := getAESKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密（nonce + ciphertext 拼接）
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESEncryptBytes 使用 AES-256-GCM 加密二进制内容。
// 该函数适用于备份文件、导出文件等非文本数据场景，返回值仍为“随机 nonce + 密文”的字节流。
func AESEncryptBytes(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	key, err := getAESKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt 使用 AES-256-GCM 解密密文
// 输入为 base64 编码的密文（包含 nonce）
func AESDecrypt(cipherBase64 string) (string, error) {
	if cipherBase64 == "" {
		return "", nil
	}

	key, err := getAESKey()
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", errors.New("密文格式错误")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文长度不足")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("解密失败")
	}

	return string(plaintext), nil
}

// AESDecryptBytes 使用 AES-256-GCM 解密二进制内容。
// 输入必须是 AESEncryptBytes 输出的原始字节流格式。
func AESDecryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}

	key, err := getAESKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("密文格式错误")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	payload := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

// GenerateRandomPassword 生成随机密码
// 用于学校审核通过后为首个校管生成初始密码
// 使用 crypto/rand.Int 消除 modulo bias
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$"
	charsetLen := big.NewInt(int64(len(charset)))

	b := make([]byte, length)
	for i := range b {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("生成随机密码失败: %w", err)
		}
		b[i] = charset[idx.Int64()]
	}

	// 确保密码包含至少一个大写、一个小写、一个数字、一个特殊字符
	password := string(b)
	if !isPasswordComplex(password) {
		// 强制在固定位置插入各类字符
		runes := []rune(password)
		runes[0] = pickRandom("ABCDEFGHJKLMNPQRSTUVWXYZ")
		runes[1] = pickRandom("abcdefghijkmnpqrstuvwxyz")
		runes[2] = pickRandom("23456789")
		runes[3] = pickRandom("!@#$")
		password = string(runes)
	}

	return password, nil
}

// isPasswordComplex 检查密码是否满足复杂度要求
func isPasswordComplex(s string) bool {
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// pickRandom 从字符集中随机选择一个字符
func pickRandom(charset string) rune {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return rune(charset[0])
	}
	return rune(charset[idx.Int64()])
}
