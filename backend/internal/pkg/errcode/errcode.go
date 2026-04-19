// errcode.go
// 该文件是整套业务错误码体系的基础定义，负责声明应用错误结构、错误码包装方式以及错误
// 向统一响应结构转换的规则。各模块错误码文件都建立在这里的 AppError 之上。

package errcode

import (
	"fmt"
	"net/http"
)

// AppError 业务错误结构体
type AppError struct {
	Code       int    `json:"code"`       // 5位业务错误码
	Message    string `json:"message"`    // 错误描述
	HTTPStatus int    `json:"-"`          // HTTP状态码（不序列化到JSON）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	return fmt.Sprintf("错误码: %d, 消息: %s", e.Code, e.Message)
}

// New 创建业务错误
func New(code int, httpStatus int, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// WithMessage 创建带自定义消息的错误（基于已有错误码）
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    msg,
		HTTPStatus: e.HTTPStatus,
	}
}

// WithMessagef 创建带格式化消息的错误
func (e *AppError) WithMessagef(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: e.HTTPStatus,
	}
}

// IsCode 判断是否为指定错误码
func (e *AppError) IsCode(code int) bool {
	return e.Code == code
}

// Is 实现 errors.Is 接口
// 支持 errors.Is(err, target) 判断
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}
	return false
}

// IsAppError 判断 error 是否为 AppError 类型
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// FromError 将普通 error 转换为 AppError
// 如果已经是 AppError 则直接返回，否则包装为内部错误
func FromError(err error) *AppError {
	if appErr, ok := IsAppError(err); ok {
		return appErr
	}
	return New(50000, http.StatusInternalServerError, err.Error())
}
