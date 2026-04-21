// response.go
// 该文件封装所有 HTTP 接口必须使用的统一响应结构，负责把成功、分页、校验错误和业务
// 错误按 API 规范输出为同一 JSON 形态。它的职责是统一“怎么返回”，而不是判断“该返回
// 什么业务结果”。

package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/errcode"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// PaginatedData 分页数据结构
type PaginatedData struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// now 获取当前 ISO 8601 时间戳（UTC）
func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Now 获取当前 ISO 8601 时间戳（导出，供中间件使用）
func Now() string {
	return now()
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      data,
		Timestamp: now(),
	})
}

// SuccessWithMsg 成功响应（自定义消息）
func SuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   msg,
		Data:      data,
		Timestamp: now(),
	})
}

// Created 创建成功响应（HTTP 200，业务码200，遵循统一响应格式）
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      data,
		Timestamp: now(),
	})
}

// Paginated 分页响应
func Paginated(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: PaginatedData{
			List: list,
			Pagination: Pagination{
				Page:       page,
				PageSize:   pageSize,
				Total:      total,
				TotalPages: totalPage,
			},
		},
		Timestamp: now(),
	})
}

// Error 业务错误响应
func Error(c *gin.Context, err *errcode.AppError) {
	c.JSON(err.HTTPStatus, Response{
		Code:      err.Code,
		Message:   err.Message,
		Data:      nil,
		Timestamp: now(),
	})
}

// ErrorWithData 业务错误响应（附带数据，如校验错误详情）
func ErrorWithData(c *gin.Context, err *errcode.AppError, data interface{}) {
	c.JSON(err.HTTPStatus, Response{
		Code:      err.Code,
		Message:   err.Message,
		Data:      data,
		Timestamp: now(),
	})
}

// ValidationError 参数校验错误响应
func ValidationError(c *gin.Context, errors interface{}) {
	c.JSON(http.StatusBadRequest, Response{
		Code:      errcode.ErrInvalidParams.Code,
		Message:   errcode.ErrInvalidParams.Message,
		Data:      gin.H{"errors": errors},
		Timestamp: now(),
	})
}

// Abort 中止请求并返回错误（用于中间件）
func Abort(c *gin.Context, err *errcode.AppError) {
	if err == nil {
		err = errcode.ErrInternal
	}
	c.AbortWithStatusJSON(err.HTTPStatus, Response{
		Code:      err.Code,
		Message:   err.Message,
		Data:      nil,
		Timestamp: now(),
	})
}
