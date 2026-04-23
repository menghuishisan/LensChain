// secret_codec.go
// 模块05 — CTF竞赛：敏感字段编解码辅助。
// 负责静态 Flag、动态 Flag 密钥和教师 PoC 的统一加解密，
// 避免在多个 service 文件中各自散写安全字段处理逻辑。

package ctf

import (
	"strings"

	cryptopkg "github.com/lenschain/backend/internal/pkg/crypto"
)

// encryptSensitiveText 对模块05中的敏感明文进行 AES 加密。
// 空字符串保持为空，避免把“未填写”写成无意义密文。
func encryptSensitiveText(plain *string) (*string, error) {
	if plain == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*plain)
	if trimmed == "" {
		empty := ""
		return &empty, nil
	}
	cipherText, err := cryptopkg.AESEncrypt(trimmed)
	if err != nil {
		return nil, err
	}
	return &cipherText, nil
}

// decryptSensitiveText 读取模块05中的敏感字段明文。
// 解密失败时返回空值，避免把未加密明文继续当作合法敏感数据使用。
func decryptSensitiveText(cipherText *string) *string {
	if cipherText == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*cipherText)
	if trimmed == "" {
		empty := ""
		return &empty
	}
	plain, err := cryptopkg.AESDecrypt(trimmed)
	if err != nil {
		empty := ""
		return &empty
	}
	return &plain
}

// decryptSensitiveTextValue 返回敏感字段的明文字符串值，便于运行时执行器直接使用。
func decryptSensitiveTextValue(cipherText *string) string {
	plain := decryptSensitiveText(cipherText)
	if plain == nil {
		return ""
	}
	return *plain
}
