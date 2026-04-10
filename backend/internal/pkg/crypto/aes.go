// aes.go
// AES-256 加密/解密工具
// 用于 SSO client_secret 等敏感配置的加密存储
// 使用 AES-256-GCM 模式，提供认证加密（AEAD）

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
	"go.uber.org/zap"
)

// aesKey 全局 AES 密钥（32字节 = AES-256）
var (
	aesKey     []byte
	aesKeyOnce sync.Once
)

// initAESKey 从环境变量加载 AES 密钥
// 密钥必须是 base64 编码的 32 字节数据
// 生产环境（GIN_MODE=release）缺少密钥时直接 panic
func initAESKey() {
	keyStr := os.Getenv("AES_ENCRYPTION_KEY")
	if keyStr == "" {
		if os.Getenv("GIN_MODE") == "release" {
			panic("生产环境必须配置 AES_ENCRYPTION_KEY 环境变量")
		}
		// 开发环境使用默认密钥并打印警告
		logger.L.Warn("AES_ENCRYPTION_KEY 未配置，使用开发环境默认密钥（请勿用于生产环境）")
		keyStr = "bGVuc2NoYWluLWFlcy0yNTYta2V5LWRldiE=" // base64("lenschain-aes-256-key-dev!")
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		panic(fmt.Sprintf("AES_ENCRYPTION_KEY base64 解码失败: %v", err))
	}

	// 严格要求 32 字节（AES-256），不做 padding
	if len(key) != 32 {
		panic(fmt.Sprintf("AES_ENCRYPTION_KEY 必须为 32 字节（当前 %d 字节）", len(key)))
	}

	aesKey = key
}

// getAESKey 获取 AES 密钥（懒加载）
func getAESKey() []byte {
	aesKeyOnce.Do(initAESKey)
	return aesKey
}

// AESEncrypt 使用 AES-256-GCM 加密明文
// 返回 base64 编码的密文（包含 nonce）
func AESEncrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key := getAESKey()
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

// AESDecrypt 使用 AES-256-GCM 解密密文
// 输入为 base64 编码的密文（包含 nonce）
func AESDecrypt(cipherBase64 string) (string, error) {
	if cipherBase64 == "" {
		return "", nil
	}

	key := getAESKey()
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

// MustGenerateRandomPassword 生成随机密码，失败时记录日志并 panic
// 仅在密码生成是关键路径时使用（如创建校管账号）
func MustGenerateRandomPassword(length int) string {
	pwd, err := GenerateRandomPassword(length)
	if err != nil {
		logger.L.Error("生成随机密码失败", zap.Error(err))
		panic("生成随机密码失败: " + err.Error())
	}
	return pwd
}
