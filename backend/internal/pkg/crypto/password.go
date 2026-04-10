// password.go
// 密码加密工具
// 基于 bcrypt 算法，用于用户密码的加密存储和验证
// 所有密码操作必须通过此包，禁止明文存储

package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost bcrypt 默认计算成本
const DefaultCost = 12

// HashPassword 加密密码
// 使用 bcrypt 算法，返回加密后的哈希字符串
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 验证密码
// 比较明文密码与哈希值是否匹配
// 返回 true 表示匹配
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
