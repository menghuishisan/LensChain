// password.go
// 该文件提供平台统一的密码哈希与校验能力，专门负责用户登录密码的安全存储与比对。它的
// 目标是把“密码绝不能明文存储、绝不能自己拼接摘要”的规则固化为公共能力，让认证相关
// service 只关注业务流程，不再分散处理密码算法细节。

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
