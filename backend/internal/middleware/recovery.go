// recovery.go
// 该文件提供 HTTP 请求级的 panic 恢复中间件，负责在出现未处理异常时记录堆栈并返回统一
// 的 500 响应，避免单个请求把整个服务打崩。它属于基础兜底能力，而不是常规错误处理路径。

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
