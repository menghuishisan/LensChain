// logger.go
// 请求日志中间件
// 记录每个 HTTP 请求的方法、路径、状态码、耗时、客户端IP等信息
// 使用 Zap 结构化日志

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/pkg/logger"
)

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 构建日志字段
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// 如果有用户ID，记录到日志
		if userID := GetUserID(c); userID > 0 {
			fields = append(fields, zap.Int64("user_id", userID))
		}

		// 如果有错误，记录错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			logger.L.Error("HTTP请求", fields...)
		case statusCode >= 400:
			logger.L.Warn("HTTP请求", fields...)
		default:
			logger.L.Info("HTTP请求", fields...)
		}
	}
}
