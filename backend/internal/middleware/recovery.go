// recovery.go
// 异常恢复中间件
// 捕获 handler 中的 panic，防止服务崩溃
// 记录错误堆栈到日志，返回500错误响应

package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/response"
)

// Recovery 异常恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录错误堆栈
				logger.L.Error("请求处理 panic",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
					zap.String("stack", string(debug.Stack())),
				)

				// 使用统一响应格式返回500错误
				response.Abort(c, errcode.ErrInternal)
			}
		}()
		c.Next()
	}
}
