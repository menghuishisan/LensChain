// bind.go
// 该文件为 handler 层提供统一的参数绑定与校验入口，负责把 JSON、Query、Path 参数解析
// 成 DTO，并把校验错误转成符合接口规范的错误响应。它的目标是统一 handler 的输入处理
// 方式，避免各接口自己拼错误信息或直接暴露框架原始报错。

package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// BindAndValidate 绑定请求参数并校验
// 如果校验失败，自动返回 400 错误响应并返回 false
// handler 层使用方式：
//
//	var req dto.CreateUserReq
//	if !validator.BindAndValidate(c, &req) { return }
func BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBind(obj); err != nil {
		handleValidationError(c, err)
		return false
	}
	return true
}

// BindJSON 绑定 JSON 请求体并校验
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		handleValidationError(c, err)
		return false
	}
	return true
}

// BindQuery 绑定查询参数并校验
func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		handleValidationError(c, err)
		return false
	}
	return true
}

// BindURI 绑定 URI 参数并校验
func BindURI(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		handleValidationError(c, err)
		return false
	}
	return true
}

// ParsePathID 解析路径参数中的 ID（雪花ID字符串 → int64）
// 参数 param 为路径参数名，如 "id"、"student_id"
// 解析失败自动返回 400 错误响应并返回 0, false
func ParsePathID(c *gin.Context, param string) (int64, bool) {
	idStr := c.Param(param)
	if idStr == "" {
		response.Error(c, errcode.ErrInvalidID.WithMessage(fmt.Sprintf("缺少路径参数: %s", param)))
		return 0, false
	}

	id, err := snowflake.ParseString(idStr)
	if err != nil || id <= 0 {
		response.Error(c, errcode.ErrInvalidID.WithMessage(fmt.Sprintf("无效的ID: %s", idStr)))
		return 0, false
	}

	return id, true
}

// ParseIDList 解析字符串 ID 列表（雪花ID字符串 → int64 列表）
// 解析失败自动返回 400 错误响应并返回 nil, false。
func ParseIDList(c *gin.Context, ids []string) ([]int64, bool) {
	result := make([]int64, 0, len(ids))
	for _, idStr := range ids {
		id, err := snowflake.ParseString(idStr)
		if err != nil || id <= 0 {
			response.Error(c, errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("无效的ID: %s", idStr)))
			return nil, false
		}
		result = append(result, id)
	}
	return result, true
}

// ParseQueryInt 解析查询参数中的整数值
// 如果参数不存在返回 defaultVal
func ParseQueryInt(c *gin.Context, key string, defaultVal int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// ParseQueryInt64 解析查询参数中的 int64 值
func ParseQueryInt64(c *gin.Context, key string, defaultVal int64) int64 {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// handleValidationError 处理校验错误
// 将 validator 错误翻译为中文友好的错误信息
func handleValidationError(c *gin.Context, err error) {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		errors := make([]map[string]string, 0, len(validationErrors))
		for _, e := range validationErrors {
			errors = append(errors, map[string]string{
				"field":   toSnakeCase(e.Field()),
				"message": translateValidationError(e),
			})
		}
		response.ValidationError(c, errors)
		return
	}

	// 非校验错误（如 JSON 解析失败）
	response.Error(c, errcode.ErrInvalidParams.WithMessage("请求参数格式错误"))
}

// translateValidationError 翻译校验错误为中文
func translateValidationError(e validator.FieldError) string {
	field := toSnakeCase(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s 不能为空", field)
	case "min":
		return fmt.Sprintf("%s 长度不能小于 %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能大于 %s", field, e.Param())
	case "email":
		return fmt.Sprintf("%s 格式不正确", field)
	case "phone":
		return fmt.Sprintf("%s 手机号格式不正确", field)
	case "password":
		return fmt.Sprintf("%s 不满足密码复杂度要求（至少8位，包含大小写字母和数字）", field)
	case "snowflake_id":
		return fmt.Sprintf("%s ID格式不正确", field)
	case "oneof":
		return fmt.Sprintf("%s 值必须是 [%s] 之一", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s 不能小于 %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s 不能大于 %s", field, e.Param())
	case "url":
		return fmt.Sprintf("%s URL格式不正确", field)
	default:
		return fmt.Sprintf("%s 校验失败 (%s)", field, e.Tag())
	}
}

// toSnakeCase 将驼峰命名转为蛇形命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + 32) // 转小写
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
