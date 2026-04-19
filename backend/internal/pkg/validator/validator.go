// validator.go
// 该文件注册项目通用的自定义校验规则，例如手机号格式、密码复杂度和雪花 ID 合法性。
// DTO 层只声明标签，不重复写校验实现；真正的规则定义集中维护在这里。

package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// 正则表达式
var (
	// 中国大陆手机号：1开头，第二位3-9，共11位
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

	// 密码复杂度：至少8位，包含大写字母、小写字母和数字
	passwordUpperRegex = regexp.MustCompile(`[A-Z]`)
	passwordLowerRegex = regexp.MustCompile(`[a-z]`)
	passwordDigitRegex = regexp.MustCompile(`[0-9]`)

	// 雪花ID：纯数字字符串
	snowflakeRegex = regexp.MustCompile(`^\d{15,20}$`)
)

// RegisterCustomValidators 注册所有自定义校验规则到 Gin 的 validator 实例
func RegisterCustomValidators(v *validator.Validate) error {
	// 手机号校验
	if err := v.RegisterValidation("phone", validatePhone); err != nil {
		return err
	}

	// 密码复杂度校验
	if err := v.RegisterValidation("password", validatePassword); err != nil {
		return err
	}

	// 雪花ID校验
	if err := v.RegisterValidation("snowflake_id", validateSnowflakeID); err != nil {
		return err
	}

	return nil
}

// validatePhone 手机号校验
func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

// validatePassword 密码复杂度校验
// 规则：至少8位，包含大写字母、小写字母和数字
func validatePassword(fl validator.FieldLevel) bool {
	pwd := fl.Field().String()
	if len(pwd) < 8 {
		return false
	}
	if !passwordUpperRegex.MatchString(pwd) {
		return false
	}
	if !passwordLowerRegex.MatchString(pwd) {
		return false
	}
	if !passwordDigitRegex.MatchString(pwd) {
		return false
	}
	return true
}

// validateSnowflakeID 雪花ID格式校验
func validateSnowflakeID(fl validator.FieldLevel) bool {
	return snowflakeRegex.MatchString(fl.Field().String())
}
