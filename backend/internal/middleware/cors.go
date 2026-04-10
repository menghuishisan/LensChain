// cors.go
// 跨域中间件
// 根据配置文件设置 CORS 响应头

package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/config"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	cfg := config.Get().CORS

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查来源是否在允许列表中
		allowed := false
		for _, o := range cfg.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Total-Count")
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", cfg.MaxAge.Seconds()))

		// 预检请求直接返回
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
