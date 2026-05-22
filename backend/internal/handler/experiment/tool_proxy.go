// tool_proxy.go
// 模块04 — 实验环境：工具容器反向代理。
//
// 学生侧 IDE / 区块链浏览器 / VNC 桌面 / 监控仪表盘等工具镜像在 iframe 中加载时，
// 主资源与所有子资源（CSS/JS/图片/IDE 内部 WS 等）必须经由 backend 反代到 K8s 集群内
// Pod 的工具端口。本文件实现两个相关端点：
//
//  1. IssueToolProxyCookie  POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie
//     - 走全局 JWTAuth（Bearer access token），完成业务校验后签 cookie
//     - 校验路径：ResolveToolProxyTarget（学生本人 + 实例 Running + 该 kind 容器存在）
//
//  2. ServeToolProxy        ANY  /instance/:instance_id/:tool_kind/*proxy_path
//     - 走 ToolProxyAuth（cookie 鉴权），不走 JWTAuth
//     - HTTP 与 WebSocket 共用 httputil.ReverseProxy（Go 1.20+ 原生支持 Connection: Upgrade
//       的 101 透传与双向流桥接），上游连接由 SPDY DialContext 提供
//     - cookie 已携 (Namespace, PodName, Port)，本端点无 DB 命中
//
// iframe / WS origin 解耦：浏览器看到的 origin 不强制等于 backend origin。
//   - 生产环境：Ingress 兜底，frontend / backend 同源，前端 iframe src 用相对路径；
//   - 本地开发：浏览器直接连 backend :8080（`NEXT_PUBLIC_TOOL_PROXY_BASE_URL`），
//     绕开 Next dev rewrite 对 trailingSlash 与 WS upgrade 的路径归一化限制。
//   两种部署对 backend 完全透明——cookie 发到哪个 origin / iframe src 拼成什么形态，
//   由前端 services/experimentToolProxy.ts::resolveToolProxyURL 决定，详见该处注释。
//
// 不允许直拨 Pod IP / Service ClusterIP / NodePort，所有上游连接强制走 K8s API 的 SPDY
// portforward 隧道（详见 docs/modules/09-部署与运维/02-基础设施设计.md §2.4）。

package experiment

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
)

// toolProxyDialTimeout SPDY 隧道拨号超时——上游 K8s API / Pod 不可达时不能让 HTTP 请求挂死。
const toolProxyDialTimeout = 8 * time.Second

// IssueToolProxyCookie 签发工具反代 cookie。
// POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie
//
// 职责边界：handler 只做 HTTP 层面的工作（参数解析 / 写 Set-Cookie / 拼响应）；
// "签什么 token / TTL 多少 / cookie path 怎么算"是业务决策，由 service.IssueToolProxyAccess
// 完成。这与 ServeSimEngineWS 中"上游 token 由 GetSimEngineProxyTarget 现签"模式一致。
func (h *InstanceHandler) IssueToolProxyCookie(c *gin.Context) {
	instanceID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	toolKind := strings.TrimSpace(c.Param("kind"))
	if toolKind == "" {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("tool_kind 不能为空"))
		return
	}
	sc := handlerctx.BuildServiceContext(c)

	access, err := h.instanceService.IssueToolProxyAccess(c.Request.Context(), sc, instanceID, toolKind)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	// SameSite=Lax：iframe src 与 cookie 同站时可正常携带；跨站部署需切 None+Secure，此处
	// 通过 X-Forwarded-Proto / TLS 自动判定 Secure 标志（生产 HTTPS 必为 true）。
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.ToolProxyCookieName,
		Value:    access.Token,
		Path:     access.CookiePath,
		MaxAge:   int(access.ExpiresIn.Seconds()),
		HttpOnly: true,
		Secure:   toolProxyCookieSecure(c),
		SameSite: http.SameSiteLaxMode,
	})

	response.Success(c, dto.IssueToolProxyCookieResp{
		ProxyPath: access.ProxyPath,
		ExpiresIn: int64(access.ExpiresIn.Seconds()),
	})
}

// toolProxyCookieSecure 根据请求协议决定 cookie Secure 标志。
//
// 浏览器在 HTTP 上拒绝接收 Secure cookie；HTTPS 下必须打开。生产部署一定走 HTTPS（Ingress
// TLS 终止），dev 本地调试走 HTTP（端口 8080 / 3000）。
func toolProxyCookieSecure(c *gin.Context) bool {
	// 优先读 X-Forwarded-Proto（Ingress 透传），无则看 TLS。
	if proto := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))); proto != "" {
		return proto == "https"
	}
	return c.Request.TLS != nil
}

// ServeToolProxy 工具反代主入口。
// ANY /instance/:instance_id/:tool_kind/*proxy_path
//
// 请求已被 ToolProxyAuth 中间件鉴权，gin.Context 中已注入 ToolProxyClaims。
func (h *InstanceHandler) ServeToolProxy(c *gin.Context) {
	raw, exists := c.Get(middleware.ToolProxyContextKey)
	if !exists {
		response.Abort(c, errcode.ErrUnauthorized)
		return
	}
	claims, ok := raw.(*jwtpkg.ToolProxyClaims)
	if !ok || claims == nil {
		response.Abort(c, errcode.ErrUnauthorized)
		return
	}

	// 路径剥离：上游 Pod 内部不知道 /instance/<id>/<kind> 前缀，必须去掉再转发。
	prefix := fmt.Sprintf("/instance/%d/%s", claims.InstanceID, claims.ToolKind)
	upstreamPath := strings.TrimPrefix(c.Request.URL.Path, prefix)
	if upstreamPath == "" {
		upstreamPath = "/"
	}

	h.serveToolProxyHTTP(c, claims, upstreamPath)
}

// serveToolProxyHTTP 处理 HTTP 与 WebSocket 反代。
//
// Go 1.20+ 的 httputil.ReverseProxy 原生识别 `Connection: Upgrade` + `Upgrade: websocket`
// 头并完成 101 Switching Protocols 透传与双向字节流桥接，无需在 handler 层判 WS 走单独
// hijack 实现——单实现覆盖 HTML/CSS/JS/字体/IDE 内部 WS / VNC websockify / blockscout
// 实时事件等所有协议形态。
//
// 每个请求新建 SPDY 隧道（无法跨请求复用 SPDY 流）。
func (h *InstanceHandler) serveToolProxyHTTP(c *gin.Context, claims *jwtpkg.ToolProxyClaims, upstreamPath string) {
	upstreamHost := fmt.Sprintf("%s:%d", claims.PodName, claims.Port)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = upstreamHost
			req.URL.Path = upstreamPath
			req.URL.RawPath = ""
			req.Host = upstreamHost
			// 不向上游泄漏反代 cookie——这是 backend 与浏览器之间的凭证，与上游无关。
			req.Header.Del("Cookie")
			// 同源/CSRF 重写 Origin：上游工具（code-server / VS Code Web、jupyter、grafana
			// 等）在 WebSocket / 敏感 HTTP 端点会比较 Origin 与自身 Host，不一致直接 403。
			// 浏览器 → backend 之间走 backend origin（localhost:8080 / Ingress 域名），上游
			// Pod 完全不感知该外部域名，从上游视角"请求来自自己" 才是事实。这与 nginx
			// `proxy_set_header Origin $scheme://$proxy_host` 的做法等价，是反代 WS 的
			// 正解，不是临时绕过。同样处理 Referer，避免上游 Referer 检查复现该问题。
			if req.Header.Get("Origin") != "" {
				req.Header.Set("Origin", "http://"+upstreamHost)
			}
			if req.Header.Get("Referer") != "" {
				req.Header.Set("Referer", "http://"+upstreamHost+"/")
			}
			// X-Forwarded-* 让上游应用知道实际外部协议 / 主机，便于生成正确的链接。
			if req.Header.Get("X-Forwarded-Proto") == "" {
				if c.Request.TLS != nil {
					req.Header.Set("X-Forwarded-Proto", "https")
				} else {
					req.Header.Set("X-Forwarded-Proto", "http")
				}
			}
			req.Header.Set("X-Forwarded-Prefix", fmt.Sprintf("/instance/%d/%s", claims.InstanceID, claims.ToolKind))
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialCtx, cancel := context.WithTimeout(ctx, toolProxyDialTimeout)
				defer cancel()
				return h.instanceService.DialPodPort(dialCtx, claims.Namespace, claims.PodName, claims.Port)
			},
			// SPDY 流是单向字节流，无法跨请求复用做不同 HTTP 请求。每请求新建 SPDY 流。
			DisableKeepAlives: true,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			c.Error(err)
			http.Error(w, "工具反代上游不可达: "+err.Error(), http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(c.Writer, c.Request)
	// 刷新活跃时间——iframe 持续打开 = 学生在使用工具。
	// HTTP 与 WS（被 ReverseProxy 升级后阻塞到流关闭）都会在此返回时触达。
	h.instanceService.TouchActivity(context.WithoutCancel(c.Request.Context()), claims.InstanceID)
}

