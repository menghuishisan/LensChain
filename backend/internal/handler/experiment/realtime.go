// realtime.go
// 模块04 — 实验环境：WebSocket 处理层
// 负责实例状态、组内消息、教师监控、终端 PTY 代理、终端只读流和 SimEngine 代理通道

package experiment

import (
	"context"
	"encoding/json"
	"io"
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
//
// 通过 K8s Pod exec subresource 直接在目标容器内拉起 PTY（kubectl exec / Lens / Rancher /
// OpenShift Web Terminal 一致路径）。任何 Running 容器都可作为终端目标，工具按容器自然
// 就绪（redis-cli 在 redis 容器、psql 在 postgres 容器、geth attach 在 geth 容器）。
// query.container 留空则使用实例首个就绪容器（按 template_containers.sort_order）。
//
// GET /api/v1/experiment-instances/:id/terminal?container=xxx
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
		response.Abort(c, errcode.ErrInvalidParams.WithMessage("实例没有就绪容器可作为终端目标"))
		return
	}
	h.serveTerminalExecPTY(c, sc, target)
}

// serveTerminalExecPTY 在客户端 WebSocket 与 K8s exec PTY 流之间建立全双工桥接。
//
// 协议（前后端唯一约定）：
//   - 客户端 → 服务端：WS Text/Binary 帧。
//     * Text 帧若为 JSON {"type":"resize","cols":N,"rows":N}，作为 SIGWINCH 处理。
//     * 其余字节（含 Text/Binary）一律视为 PTY 输入，直接写入容器 stdin。
//   - 服务端 → 客户端：容器 stdout 字节流（TTY 模式 stderr 已合并）以 WS BinaryMessage 下发。
//     PTY 数据是字节流（含 ANSI 转义、回车 \r、二进制控制字符），不能按行读取。
//   - 连接初始化：服务端立即发一帧 {"type":"terminal_init","data":{"mode":"pty","container":...}}，
//     前端据此判定通道就绪。
//
// 生命周期：客户端断开 → 关闭 stdinWriter → exec stdin EOF → PTY 进程退出 → 关闭
// stdoutWriter → stdout→WS goroutine 退出。任一端关闭都会引导另一端自然解开。
func (h *InstanceHandler) serveTerminalExecPTY(c *gin.Context, _ *svcctx.ServiceContext, target *svc.TerminalProxyTarget) {
	clientConn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	// 上行 stdin：客户端 → io.Pipe → K8s exec。Pipe 端关闭即向 exec 发 EOF。
	stdinReader, stdinWriter := io.Pipe()
	// 下行 stdout：K8s exec → io.Pipe → 客户端 WS。
	stdoutReader, stdoutWriter := io.Pipe()
	resizeCh := make(chan svc.TerminalSize, 4)

	// ExecPodPTY 阻塞到 PTY 进程退出 / ctx 取消。生命周期独立于 HTTP 请求 ctx，
	// 使用 WithoutCancel 让 K8s exec 在 WS 升级完成后不被 gin 的 ctx 提前取消。
	callCtx := websocketServiceContext(c)
	execCtx, execCancel := context.WithCancel(callCtx)
	defer execCancel()

	// 协调三个 goroutine 的退出：execDone（PTY 退出）、wsReadDone（客户端断开）。
	// 任一发生都触发关闭：stdinWriter/stdoutWriter 关闭 → 另一侧 io.Pipe 读取返回 EOF。
	execDone := make(chan error, 1)
	go func() {
		execDone <- h.instanceService.ExecTerminalPTY(execCtx, target, stdinReader, stdoutWriter, resizeCh)
		// PTY 退出后必须关闭 stdoutWriter，否则 stdout→WS goroutine 永远阻塞在 Read。
		_ = stdoutWriter.Close()
	}()

	// 发送 PTY 就绪指示，前端据此切换到键击转发状态。
	_ = clientConn.WriteJSON(map[string]interface{}{
		"type": "terminal_init",
		"data": map[string]interface{}{"mode": "pty", "container": target.ContainerName},
	})

	// goroutine 1：客户端 → stdin。读取 WS 消息，识别 resize JSON，其余原样写 stdin。
	go func() {
		defer func() {
			// 关闭 stdinWriter 让 exec stdin 收到 EOF，触发 PTY 进程退出（sh 默认 exit）。
			_ = stdinWriter.Close()
			// 关闭 resize 通道让 client-go 的 ptySizeQueue.Next 返回 nil 停止 SIGWINCH。
			close(resizeCh)
		}()
		for {
			messageType, payload, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			// 仅 Text 帧尝试解析 resize；二进制帧总是直接转 stdin。
			if messageType == websocket.TextMessage {
				if cols, rows, ok := tryParseTerminalResize(payload); ok {
					select {
					case resizeCh <- svc.TerminalSize{Width: uint16(cols), Height: uint16(rows)}:
					default:
						// 通道满（4 容量足够，理论上不会到这），丢弃最旧的 resize。
					}
					h.instanceService.TouchActivity(callCtx, target.InstanceID)
					continue
				}
			}
			if _, werr := stdinWriter.Write(payload); werr != nil {
				return
			}
			h.instanceService.TouchActivity(callCtx, target.InstanceID)
		}
	}()

	// goroutine 2（当前 goroutine）：stdout → 客户端。PTY 数据按字节流读，固定大小缓冲。
	buf := make([]byte, 4*1024)
	for {
		n, rerr := stdoutReader.Read(buf)
		if n > 0 {
			if werr := clientConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
				break
			}
		}
		if rerr != nil {
			break
		}
	}

	// 等 ExecPodPTY 真正退出，把错误（如有）记录到 gin 请求日志，便于运维定位。
	if err := <-execDone; err != nil && c.Request.Context().Err() == nil {
		c.Error(err)
		// 错误时尝试通知客户端：连接已断，前端会看到 mode=error 并自动重连。
		_ = clientConn.WriteJSON(map[string]interface{}{
			"type": "terminal_init",
			"data": map[string]interface{}{
				"mode":            "error",
				"message":         "终端会话已结束",
				"upstream_target": target.Namespace + "/" + target.PodName + "/" + target.ContainerName,
				"upstream_reason": err.Error(),
			},
		})
	}
}

// tryParseTerminalResize 尝试把 WS Text 帧解析为终端 resize 控制消息。
//
// 控制消息格式：{"type":"resize","cols":C,"rows":R}。命中时返回 (cols,rows,true)，
// 调用方应把消息送入 resize 通道；解析失败（非 JSON、type 不是 resize、字段缺失等）
// 返回 ok=false，调用方应把 payload 当作普通 PTY 输入字节流透传到 stdin。
//
// 之所以放在 handler 层而不是引一个通用 codec 包：终端通道只有这一种控制消息，
// 增加一层封装反而隐藏协议；将来若引入更多控制帧（如 detach / signal）再考虑抽取。
func tryParseTerminalResize(payload []byte) (cols, rows int, ok bool) {
	// 快速预筛：必须以 '{' 开头。避免对所有键击都跑 JSON parser。
	if len(payload) < 2 || payload[0] != '{' {
		return 0, 0, false
	}
	var msg struct {
		Type string `json:"type"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return 0, 0, false
	}
	if msg.Type != "resize" || msg.Cols <= 0 || msg.Rows <= 0 {
		return 0, 0, false
	}
	return msg.Cols, msg.Rows, true
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
