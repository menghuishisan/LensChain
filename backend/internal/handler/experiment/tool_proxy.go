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
//     - HTTP 走 httputil.ReverseProxy + SPDY DialContext
//     - WebSocket 走 hijack + 双向 io.Copy
//     - cookie 已携 (Namespace, PodName, Port)，本端点无 DB 命中
//
// 不允许直拨 Pod IP / Service ClusterIP / NodePort，所有上游连接强制走 K8s API 的 SPDY
// portforward 隧道（详见 docs/modules/09-部署与运维/02-基础设施设计.md §2.4）。

package experiment

import (
	"context"
	"fmt"
	"io"
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

	if isWebSocketUpgrade(c.Request) {
		h.serveToolProxyWebSocket(c, claims, upstreamPath)
		return
	}
	h.serveToolProxyHTTP(c, claims, upstreamPath)
}

// serveToolProxyHTTP 处理普通 HTTP 反代。每个请求新建 SPDY 隧道（无法跨请求复用 SPDY 流）。
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
}

// serveToolProxyWebSocket 处理 WebSocket upgrade 请求。
//
// httputil.ReverseProxy 不支持 WS upgrade（它只复用 HTTP RoundTrip 语义），所以走 hijack：
//  1. 通过 SPDY 隧道拿到一条裸 net.Conn 通向 Pod:Port
//  2. 把客户端 HTTP/1.1 Upgrade 请求按字节级写到 upstream
//  3. 上游回 101 Switching Protocols 后 hijack client TCP，做双向 io.Copy
//
// 这种实现对协议无感（IDE 自定义 WS / VNC websockify / blockscout 实时事件）都能正确转发。
func (h *InstanceHandler) serveToolProxyWebSocket(c *gin.Context, claims *jwtpkg.ToolProxyClaims, upstreamPath string) {
	dialCtx, cancel := context.WithTimeout(c.Request.Context(), toolProxyDialTimeout)
	defer cancel()
	upstreamConn, err := h.instanceService.DialPodPort(dialCtx, claims.Namespace, claims.PodName, claims.Port)
	if err != nil {
		c.Error(err)
		response.Error(c, errcode.ErrInternal.WithMessage("连接工具上游失败: "+err.Error()))
		return
	}
	defer upstreamConn.Close()

	// 重写请求行：去掉 /instance/<id>/<kind> 前缀，否则上游应用收到不识别的路径。
	// 同时清掉反代 cookie，避免泄漏给上游进程；保留 Authorization / Upgrade 等头。
	outReq := c.Request.Clone(c.Request.Context())
	outReq.URL.Path = upstreamPath
	outReq.URL.RawPath = ""
	outReq.URL.Scheme = ""
	outReq.URL.Host = ""
	outReq.Host = fmt.Sprintf("%s:%d", claims.PodName, claims.Port)
	outReq.RequestURI = ""
	outReq.Header.Del("Cookie")

	// hijack 客户端连接
	hj, ok := c.Writer.(http.Hijacker)
	if !ok {
		response.Error(c, errcode.ErrInternal.WithMessage("ResponseWriter 不支持 Hijack"))
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		c.Error(err)
		return
	}
	defer clientConn.Close()

	// 把客户端的 Upgrade 请求按字节级写到上游 SPDY 流。
	if err := outReq.Write(upstreamConn); err != nil {
		c.Error(err)
		return
	}

	// 双向字节级转发，任何一侧关闭就退出。
	doneCh := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstreamConn, clientConn)
		doneCh <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(clientConn, upstreamConn)
		doneCh <- struct{}{}
	}()
	<-doneCh
	// 刷新活跃时间——iframe 持续打开 = 学生在使用工具
	h.instanceService.TouchActivity(context.WithoutCancel(c.Request.Context()), claims.InstanceID)
}

// isWebSocketUpgrade 判断当前 HTTP 请求是否为 WebSocket 升级请求。
func isWebSocketUpgrade(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}

