// internal_only.go
// 模块公共中间件：内部接口访问控制。
// 仅允许来自环回或私有网段的请求访问标记为“内部接口”的路由，
// 避免把模块间调用入口直接暴露给公网。

package middleware

import (
	"crypto/subtle"
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
)

// InternalOnly 限制内部接口仅允许内网来源访问。
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isInternalRequestIP(c.ClientIP()) {
			response.Abort(c, errcode.ErrForbidden.WithMessage("内部接口仅允许内网访问"))
			return
		}
		if !hasValidInternalToken(c.GetHeader("X-Internal-Token")) {
			response.Abort(c, errcode.ErrForbidden.WithMessage("内部接口鉴权失败"))
			return
		}
		c.Next()
	}
}

// isInternalRequestIP 判断请求来源 IP 是否属于环回或私有网段。
func isInternalRequestIP(rawIP string) bool {
	ip := parseRequestIP(rawIP)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// parseRequestIP 解析 Gin 提供的来源 IP，兼容附带端口的写法。
func parseRequestIP(rawIP string) net.IP {
	rawIP = strings.TrimSpace(rawIP)
	if rawIP == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(rawIP); err == nil {
		rawIP = host
	}
	return net.ParseIP(rawIP)
}

func hasValidInternalToken(raw string) bool {
	cfg := config.Get()
	if cfg == nil {
		return false
	}
	expected := strings.TrimSpace(cfg.Internal.APIToken)
	if expected == "" {
		return true
	}
	provided := strings.TrimSpace(raw)
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
