// tool_proxy_auth.go
// 工具反代专用鉴权中间件。
//
// 设计动机：浏览器 iframe 加载工具页面（code-server / blockscout / VNC / monitor）时，
// 主资源与所有子资源（CSS/JS/图片/WS upgrade）都由 iframe 内部触发，无法附加 Authorization
// 头；只能依赖 cookie。直接复用 access token cookie 风险大（作用域宽 + 可访问全 API），
// 因此本中间件只接受 jwtpkg.ToolProxyClaims（TokenType=tool_proxy）专用 cookie：
//
//   1) cookie 路径作用域 = /instance/<instance_id>/<tool_kind>/，浏览器只在该路径下回传，
//      不会污染主站或其他工具；
//   2) cookie 由 IssueToolProxyCookie 端点（带 Bearer access token 校验）签发；
//   3) 本中间件解析 token 后再次校验 (instance_id, tool_kind) 与 URL 路径完全一致，防止
//      cookie 被复制到其他路径绕过浏览器的 path 隔离；
//   4) 不做 tokenstate 黑名单 / validAfter 检查——反代 token TTL 短（30min），强制下线
//      场景由 IssueToolProxyCookie 端点的下次刷新拦截即可，避免每个静态资源都打 redis。

package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/requestctx"
	"github.com/lenschain/backend/internal/pkg/response"
)

// ToolProxyCookieName 是工具反代 cookie 名。所有 tool kind 共用同一名称，由 cookie path
// 作用域（/instance/<id>/<kind>/）实现隔离——浏览器的 path-scoped cookie 自动按路径
// 选择回传，不会跨工具污染。
const ToolProxyCookieName = "lc_tool_proxy"

// ToolProxyContextKey 用于在 gin.Context 中暂存反代 claims，handler 直接读取 (Namespace,
// PodName, Port) 进行 SPDY 拨号，避免再次走 service 层 DB 校验。
const ToolProxyContextKey = "lc_tool_proxy_claims"

// ToolProxyAuth 工具反代鉴权中间件。
func ToolProxyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(ToolProxyCookieName)
		if err != nil || cookie == "" {
			response.Abort(c, errcode.ErrUnauthorized)
			return
		}
		claims, err := jwtpkg.ParseToolProxyToken(cookie)
		if err != nil {
			response.Abort(c, errcode.ErrTokenInvalid)
			return
		}

		// URL 上的 instance_id / tool_kind 必须与 token 完全一致：
		// path-scoped cookie 在浏览器层已经做了一次过滤；服务端再校一次防止 cookie 被以
		// 非浏览器方式（如手工拷贝、Postman）跨路径滥用。
		urlInstanceID, parseErr := strconv.ParseInt(c.Param("instance_id"), 10, 64)
		if parseErr != nil || urlInstanceID != claims.InstanceID {
			response.Abort(c, errcode.ErrForbidden)
			return
		}
		if c.Param("tool_kind") != claims.ToolKind {
			response.Abort(c, errcode.ErrForbidden)
			return
		}

		// 注入身份与反代 claims，handler 复用。注意：此中间件下 requestctx 中的身份 = 学生
		// 本人（IssueToolProxyCookie 已校验），与正常 JWTAuth 注入语义一致。
		requestctx.SetAuth(c, requestctx.AuthContext{
			UserID:   claims.UserID,
			SchoolID: claims.SchoolID,
			Roles:    []string{"student"},
			JTI:      claims.ID,
		})
		c.Set(ToolProxyContextKey, claims)
		c.Next()
	}
}
