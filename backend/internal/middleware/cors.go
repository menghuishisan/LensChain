// cors.go
// 该文件负责处理平台 HTTP 接口的跨域策略，把配置文件里的允许来源、方法和请求头规则
// 转成标准 CORS 响应头。它只解决浏览器跨域访问边界，不承担鉴权或业务来源校验职责。

package middleware

import (
	"fmt"
	"net/http"
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
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Total-Count")
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", cfg.MaxAge.Seconds()))

		// 预检请求直接返回
		if c.Request.Method == http.MethodOptions {
			if !allowed && origin != "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
