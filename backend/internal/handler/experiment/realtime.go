// realtime.go
// 模块04 — 实验环境：WebSocket 处理层
// 负责实例状态、组内消息、教师监控、终端 PTY 代理、终端只读流和 SimEngine 代理通道

package experiment

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/lenschain/backend/internal/model/dto"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	wsmanager "github.com/lenschain/backend/internal/pkg/ws"
	svc "github.com/lenschain/backend/internal/service/experiment"
)

const terminalTailLines = 120

// realtimeInboundMessage WebSocket 客户端入站消息。
type realtimeInboundMessage struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Container string `json:"container"`
}

// terminalOutputMessage 终端输出消息结构。
type terminalOutputMessage struct {
	Type      string `json:"type"`
	Container string `json:"container"`
	Data      string `json:"data"`
	Timestamp string `json:"timestamp"`
}

// ServeInstanceWS 建立实验实例状态推送通道。
// GET /api/v1/ws/experiment-instances/:id
func (h *InstanceHandler) ServeInstanceWS(c *gin.Context) {
	instanceID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if _, err := h.instanceService.GetByID(c.Request.Context(), sc, instanceID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeExperimentWS(c)
	if !ok {
		return
	}
	// 必须 JoinRoom 才能接收 service 层 BroadcastToRoom 的状态变更 / 检查点完成 / 实例异常 / 容器状态推送。
	// 之前缺这一步导致前端订阅了 WS 但永远收不到任何业务消息，UI 状态永远停留在初始值。
	manager.JoinRoom(client, svc.ExperimentInstanceRoom(instanceID))
	go client.WritePump()
	client.ReadPump(manager)
}

// ServeGroupChatWS 建立组内实时消息通道。
// GET /api/v1/ws/experiment-groups/:id/chat
func (h *InstanceHandler) ServeGroupChatWS(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if _, err := h.groupService.GetByID(c.Request.Context(), sc, groupID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeExperimentWS(c)
	if !ok {
		return
	}
	manager.JoinRoom(client, svc.ExperimentGroupRoom(groupID))
	go client.WritePump()
	h.readGroupChatLoop(c, client, manager, groupID, sc)
}

// ServeCourseMonitorWS 建立教师监控实时推送通道。
// GET /api/v1/ws/courses/:id/experiment-monitor
func (h *InstanceHandler) ServeCourseMonitorWS(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.MonitorPanelReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if _, err := h.monitorService.GetCourseMonitor(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeExperimentWS(c)
	if !ok {
		return
	}
	manager.JoinRoom(client, svc.CourseMonitorRoom(courseID, req.TemplateID))
	go client.WritePump()
	client.ReadPump(manager)
}

// ServeTerminalStreamWS 建立教师远程只读终端查看通道。
// GET /api/v1/experiment-instances/:id/terminal-stream
func (h *InstanceHandler) ServeTerminalStreamWS(c *gin.Context) {
	instanceID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	initialOutput, err := h.instanceService.GetTerminalOutput(c.Request.Context(), sc, instanceID, terminalTailLines)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeExperimentWS(c)
	if !ok {
		return
	}
	go client.WritePump()
	h.pushTerminalOutput(client, initialOutput)
	h.readTerminalLoop(c, client, manager, instanceID, sc, initialOutput.Data)
}

// ServeStudentTerminalWS 建立学生 Web 终端 PTY 通道。
// 通过 xterm-server WebSocket 代理提供真 PTY 终端体验。
// 实验实例必须挂载 xterm-server 工具容器，否则返回错误。
// GET /api/v1/experiment-instances/:id/terminal
func (h *InstanceHandler) ServeStudentTerminalWS(c *gin.Context) {
	instanceID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	containerName := strings.TrimSpace(c.Query("container"))
	sc := handlerctx.BuildServiceContext(c)

	target, err := h.instanceService.ResolveTerminalProxyTarget(c.Request.Context(), sc, instanceID, containerName)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	if target == nil {
		response.Abort(c, errcode.ErrInvalidParams.WithMessage("实验实例未挂载 xterm-server 终端服务"))
		return
	}
	h.serveTerminalPTYProxy(c, sc, target)
}

// serveTerminalPTYProxy 建立到 xterm-server 的 WebSocket 双向代理，提供真 PTY 终端体验。
//
// 上游 xterm-server 跑在实验 Pod 内（端口 3000），其网络可达性必须经由 K8s API 的 SPDY
// portforward 隧道，而不是直拨 Pod IP / Service ClusterIP / NodePort。设计依据见
// docs/modules/09-部署与运维/02-基础设施设计.md §2.4。
//
// 实现路径：通过 K8sService.DialPodPort 拿到等价于 Pod 内 localhost:3000 的 net.Conn，
// 把它注入 gorilla/websocket Dialer 的 NetDialContext，让 WebSocket 握手在 SPDY 隧道
// 之上进行。隧道生命周期与 WS 一致：上下游任一关闭，net.Conn.Close → SPDY stream 释放。
func (h *InstanceHandler) serveTerminalPTYProxy(c *gin.Context, sc *svcctx.ServiceContext, target *svc.TerminalProxyTarget) {
	clientConn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 拨号必须有超时——SPDY 上游或 K8s API 异常时不能让 net.Conn 阻塞；同时必须启用 ctx，
	// 由 ServeSimEngineWS 模块同步引入的 8 秒上限对终端代理同样适用。
	dialCtx, dialCancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer dialCancel()

	// 通过 service 层 SPDY 隧道拨号上游 xterm-server。handler 不直接依赖 K8sService，
	// 所有 (ns, pod, port) 解析与隧道建立逻辑封装在 instanceService.DialPodPort 内。
	// 业务边界（本人 / Running / 该实例容器）已在 ResolveTerminalProxyTarget 完成校验。
	tunnelConn, dialErr := h.instanceService.DialPodPort(dialCtx, target.Namespace, target.PodName, target.Port)
	if dialErr != nil {
		c.Error(dialErr)
		_ = clientConn.WriteJSON(map[string]interface{}{
			"type": "terminal_init",
			"data": map[string]interface{}{
				"mode":            "error",
				"message":         "连接终端服务失败",
				"upstream_target": target.Namespace + "/" + target.PodName,
				"upstream_reason": dialErr.Error(),
			},
		})
		_ = clientConn.Close()
		return
	}

	// 把 SPDY 隧道当作 net.Conn，让 gorilla 在它上面跑标准 WebSocket 握手。
	// NetDialContext 会被 gorilla 调用一次，返回我们的 SPDY net.Conn；之后 gorilla 在该
	// conn 上写 HTTP/1.1 Upgrade 请求，xterm-server 回 101，gorilla 把 conn 升级为
	// websocket.Conn。所有权随之转移给 upstreamWS，关闭路径只有一处（upstreamWS.Close）。
	tunnelDialer := websocket.Dialer{
		NetDialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return tunnelConn, nil
		},
		HandshakeTimeout: 8 * time.Second,
	}
	upstreamWSURL := "ws://" + target.PodName + ":" + strconv.Itoa(target.Port) + target.WebSocketPath
	upstreamConn, upstreamResp, wsErr := tunnelDialer.DialContext(dialCtx, upstreamWSURL, nil)
	if wsErr != nil {
		upstreamStatus := 0
		upstreamReason := wsErr.Error()
		if upstreamResp != nil {
			upstreamStatus = upstreamResp.StatusCode
			_ = upstreamResp.Body.Close()
		}
		_ = tunnelConn.Close()
		c.Error(wsErr)
		_ = clientConn.WriteJSON(map[string]interface{}{
			"type": "terminal_init",
			"data": map[string]interface{}{
				"mode":            "error",
				"message":         "连接终端服务失败",
				"upstream_target": target.Namespace + "/" + target.PodName,
				"upstream_status": upstreamStatus,
				"upstream_reason": upstreamReason,
			},
		})
		_ = clientConn.Close()
		return
	}

	// 发送 PTY 模式指示
	_ = clientConn.WriteJSON(map[string]interface{}{
		"type": "terminal_init",
		"data": map[string]interface{}{"mode": "pty", "container": target.ContainerName},
	})

	proxyDone := make(chan struct{}, 2)
	callCtx := websocketServiceContext(c)

	// upstream → client: PTY 输出原样转发
	go proxyWebSocket(upstreamConn, clientConn, proxyDone, nil, nil)
	// client → upstream: 用户输入原样转发，同时刷新活跃时间
	go proxyWebSocket(clientConn, upstreamConn, proxyDone, func() {
		h.instanceService.TouchActivity(callCtx, target.InstanceID)
	}, nil)

	<-proxyDone
	_ = clientConn.Close()
	_ = upstreamConn.Close()
}

// ServeGroupMemberTerminalStreamWS 建立组员只读终端查看通道。
// GET /api/v1/experiment-groups/:id/members/:student_id/terminal-stream
func (h *InstanceHandler) ServeGroupMemberTerminalStreamWS(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := validator.ParsePathID(c, "student_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	initialOutput, err := h.instanceService.GetGroupMemberTerminalOutput(c.Request.Context(), sc, groupID, studentID, terminalTailLines)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeExperimentWS(c)
	if !ok {
		return
	}
	go client.WritePump()
	h.pushTerminalOutput(client, initialOutput)
	h.readGroupMemberTerminalLoop(c, client, manager, groupID, studentID, sc, initialOutput.Data)
}

// ServeSimEngineWS 建立 SimEngine WebSocket 代理通道。
// GET /api/v1/ws/sim-engine/:session_id
func (h *InstanceHandler) ServeSimEngineWS(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		response.Abort(c, errcode.ErrInvalidParams.WithMessage("session_id 不能为空"))
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	target, err := h.instanceService.GetSimEngineProxyTarget(c.Request.Context(), sc, sessionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	// SimEngine Core 通过 ?token= 校验 JWT（sim-engine/core/internal/server/server.go::handleSimEngineWebSocket）。
	// 必须使用 service 层为本次代理现签的 SimWS token：它绑定 (UserID, SessionID, InstanceID,
	// Audience=sim-engine)，与学生的 access token 互不影响。
	// 不能透传学生 access token——其 claims 不带 session_id/instance_id/aud，过不了 Core 的
	// validateJWTClaims（详见 sim-engine/core/internal/server/jwt_validator.go）。
	// 共享密钥见 sim-engine/core/configs/config.yaml::auth.ws_jwt_secret，与 backend.jwt.access_secret 一致。
	upstreamURL := appendTokenQueryParam(normalizeWebSocketURL(target.TargetURL), target.UpstreamToken)
	logger.S.Infow("[ServeSimEngineWS] entered", "session", sessionID, "upstream", target.TargetURL, "instance", target.InstanceID)
	clientConn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.S.Errorw("[ServeSimEngineWS] upgrade client failed", "session", sessionID, "err", err)
		c.Error(err)
		return
	}
	logger.S.Infow("[ServeSimEngineWS] client upgraded; dialing upstream", "session", sessionID)

	// 同终端代理一样，拨号 SimEngine Core 必须有超时，否则上游不可达时连接挂死。
	simDialCtx, simDialCancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer simDialCancel()
	upstreamConn, upstreamResp, err := websocket.DefaultDialer.DialContext(simDialCtx, upstreamURL, nil)
	if err != nil {
		// 把上游 SimEngine Core 返回的 HTTP 状态与错误描述透传给客户端，便于排错：
		// - 401: token 校验失败（ws_jwt_secret 不一致 / token 过期 / session 归属不符）
		// - 404: Core 重启导致 session 丢失
		// - 503: Core 未初始化
		// - 拨号超时: Core 网络不可达
		// 之前统一吞成"连接 SimEngine Core 失败"导致前端只能看见模糊错误。
		upstreamStatus := 0
		upstreamReason := err.Error()
		if upstreamResp != nil {
			upstreamStatus = upstreamResp.StatusCode
			_ = upstreamResp.Body.Close()
		}
		logger.S.Errorw("[ServeSimEngineWS] upstream dial failed", "session", sessionID, "upstreamStatus", upstreamStatus, "err", err)
		c.Error(err)
		_ = clientConn.WriteJSON(gin.H{
			"type": "control_ack",
			"data": gin.H{
				"success":         false,
				"message":         "连接 SimEngine Core 失败",
				"upstream_status": upstreamStatus,
				"upstream_reason": upstreamReason,
			},
		})
		_ = clientConn.Close()
		return
	}
	logger.S.Infow("[ServeSimEngineWS] upstream connected, starting bidir proxy", "session", sessionID)

	proxyDone := make(chan struct{}, 2)
	callCtx := websocketServiceContext(c)
	// 这两条 Debug 级日志只在 LOG_LEVEL=debug 时打印；生产环境关闭，避免每秒数十帧
	// 渲染消息淹没日志（一次会话每 tick 4 个场景 ≈ 4-8 条/s）。开发环境调试 WS 代理
	// 链路时再开启。
	go proxyWebSocket(upstreamConn, clientConn, proxyDone, nil, func(mt int, payload []byte) {
		logger.S.Debugw("[SimWS] upstream->client", "session", sessionID, "mt", mt, "size", len(payload), "head", safeHeadString(payload))
	})
	go proxyWebSocket(clientConn, upstreamConn, proxyDone, func() {
		h.instanceService.TouchActivity(callCtx, target.InstanceID)
	}, func(messageType int, payload []byte) {
		logger.S.Debugw("[SimWS] client->upstream", "session", sessionID, "mt", messageType, "size", len(payload), "head", safeHeadString(payload))
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			return
		}
		h.instanceService.RecordSimEngineOperation(callCtx, sc, target.InstanceID, payload)
	})
	<-proxyDone
	logger.S.Infow("[ServeSimEngineWS] proxy ended", "session", sessionID)
	_ = clientConn.Close()
	_ = upstreamConn.Close()
}

// upgradeExperimentWS 执行 WebSocket 升级并注册到全局管理器。
func upgradeExperimentWS(c *gin.Context) (*wsmanager.Client, *wsmanager.Manager, bool) {
	conn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, nil, false
	}
	manager := wsmanager.GetManager()
	client := wsmanager.NewClient(handlerctx.BuildServiceContext(c).UserID, conn)
	manager.Register(client)
	return client, manager, true
}

// websocketServiceContext 返回 WebSocket 生命周期内调用 service 使用的上下文。
// WebSocket 升级后 HTTP 请求上下文可能随握手结束而取消，因此保留请求值但去除取消信号。
func websocketServiceContext(c *gin.Context) context.Context {
	return context.WithoutCancel(c.Request.Context())
}

// readGroupChatLoop 读取组内消息客户端输入并转交 service 层处理。
func (h *InstanceHandler) readGroupChatLoop(c *gin.Context, client *wsmanager.Client, manager *wsmanager.Manager, groupID int64, sc *svcctx.ServiceContext) {
	defer func() {
		manager.LeaveRoom(client, svc.ExperimentGroupRoom(groupID))
		manager.Unregister(client)
		_ = client.Conn.Close()
	}()

	client.Conn.SetReadLimit(4096)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	callCtx := websocketServiceContext(c)
	for {
		_, payload, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}
		var inbound realtimeInboundMessage
		if err := json.Unmarshal(payload, &inbound); err != nil {
			continue
		}
		if inbound.Type != "chat_message" || strings.TrimSpace(inbound.Content) == "" {
			continue
		}
		req := &dto.SendGroupMessageReq{Content: strings.TrimSpace(inbound.Content)}
		if err := h.groupService.SendMessage(callCtx, sc, groupID, req); err != nil {
			continue
		}
	}
}

// readTerminalLoop 轮询终端输出并处理教师指导消息。
func (h *InstanceHandler) readTerminalLoop(c *gin.Context, client *wsmanager.Client, manager *wsmanager.Manager, instanceID int64, sc *svcctx.ServiceContext, initialData string) {
	defer func() {
		manager.Unregister(client)
		_ = client.Conn.Close()
	}()

	lastOutput := initialData
	callCtx := websocketServiceContext(c)
	clientInputDone := make(chan struct{}, 1)
	go func() {
		defer func() { clientInputDone <- struct{}{} }()
		client.Conn.SetReadLimit(4096)
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.Conn.SetPongHandler(func(string) error {
			client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			_, payload, err := client.Conn.ReadMessage()
			if err != nil {
				return
			}
			var inbound realtimeInboundMessage
			if err := json.Unmarshal(payload, &inbound); err != nil {
				continue
			}
			if inbound.Type != "guidance_message" || strings.TrimSpace(inbound.Content) == "" {
				continue
			}
			req := &dto.SendGuidanceReq{Content: strings.TrimSpace(inbound.Content)}
			_ = h.instanceService.SendGuidance(callCtx, sc, instanceID, req)
		}
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-clientInputDone:
			return
		case <-ticker.C:
			current, err := h.instanceService.GetTerminalOutput(callCtx, sc, instanceID, terminalTailLines)
			if err != nil {
				return
			}
			if current.Data == lastOutput {
				continue
			}
			lastOutput = current.Data
			h.pushTerminalOutput(client, current)
		}
	}
}

// readGroupMemberTerminalLoop 轮询组员终端输出并以只读方式推送。
func (h *InstanceHandler) readGroupMemberTerminalLoop(c *gin.Context, client *wsmanager.Client, manager *wsmanager.Manager, groupID, studentID int64, sc *svcctx.ServiceContext, initialData string) {
	defer func() {
		manager.Unregister(client)
		_ = client.Conn.Close()
	}()

	lastOutput := initialData
	callCtx := websocketServiceContext(c)
	clientInputDone := make(chan struct{}, 1)
	go func() {
		defer func() { clientInputDone <- struct{}{} }()
		client.Conn.SetReadLimit(4096)
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.Conn.SetPongHandler(func(string) error {
			client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			if _, _, err := client.Conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-clientInputDone:
			return
		case <-ticker.C:
			current, err := h.instanceService.GetGroupMemberTerminalOutput(callCtx, sc, groupID, studentID, terminalTailLines)
			if err != nil {
				return
			}
			if current.Data == lastOutput {
				continue
			}
			lastOutput = current.Data
			h.pushTerminalOutput(client, current)
		}
	}
}

// pushTerminalOutput 推送终端输出消息。
func (h *InstanceHandler) pushTerminalOutput(client *wsmanager.Client, output *svc.TerminalOutput) {
	data, err := json.Marshal(terminalOutputMessage{
		Type:      "terminal_output",
		Container: output.Container,
		Data:      output.Data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	client.Send <- data
}

// normalizeWebSocketURL 归一化上游 SimEngine 地址为 ws/wss。
func normalizeWebSocketURL(raw string) string {
	if strings.HasPrefix(raw, "http://") {
		return "ws://" + strings.TrimPrefix(raw, "http://")
	}
	if strings.HasPrefix(raw, "https://") {
		return "wss://" + strings.TrimPrefix(raw, "https://")
	}
	return raw
}

// appendTokenQueryParam 把鉴权 token 透传到上游 WebSocket URL。
// SimEngine Core 通过 ?token= 校验 JWT 归属，后端代理必须保留同一 token。
func appendTokenQueryParam(rawURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return rawURL
	}
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
	}
	return rawURL + separator + "token=" + token
}

// safeHeadString 截取 payload 头部用于日志，避免长消息撑爆日志。
func safeHeadString(b []byte) string {
	const maxLen = 200
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "..."
}

// proxyWebSocket 负责单向转发 WebSocket 消息。
func proxyWebSocket(src, dst *websocket.Conn, done chan<- struct{}, onMessage func(), onPayload func(messageType int, payload []byte)) {
	defer func() { done <- struct{}{} }()
	for {
		messageType, payload, err := src.ReadMessage()
		if err != nil {
			return
		}
		if onMessage != nil {
			onMessage()
		}
		if onPayload != nil {
			onPayload(messageType, payload)
		}
		if err := dst.WriteMessage(messageType, payload); err != nil {
			return
		}
	}
}
