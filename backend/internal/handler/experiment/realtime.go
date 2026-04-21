// realtime.go
// 模块04 — 实验环境：WebSocket 处理层
// 负责实例状态、组内消息、教师监控、终端只读流和 SimEngine 代理通道

package experiment

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/lenschain/backend/internal/model/dto"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
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
	Command   string `json:"command"`
}

// terminalOutputMessage 终端输出消息结构。
type terminalOutputMessage struct {
	Type      string `json:"type"`
	Container string `json:"container"`
	Data      string `json:"data"`
	Timestamp string `json:"timestamp"`
}

// terminalCommandOutputMessage 终端命令执行结果消息结构。
type terminalCommandOutputMessage struct {
	Type      string `json:"type"`
	Container string `json:"container"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
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

// ServeStudentTerminalWS 建立学生 Web 终端命令执行通道。
// GET /api/v1/experiment-instances/:id/terminal
func (h *InstanceHandler) ServeStudentTerminalWS(c *gin.Context) {
	instanceID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	containerName := strings.TrimSpace(c.Query("container"))
	sc := handlerctx.BuildServiceContext(c)
	initialOutput, err := h.instanceService.GetStudentTerminalOutput(c.Request.Context(), sc, instanceID, containerName, terminalTailLines)
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
	h.readStudentTerminalLoop(c, client, manager, instanceID, sc, initialOutput.Container)
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

	upstreamURL := normalizeWebSocketURL(target.TargetURL)
	clientConn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	upstreamConn, _, err := websocket.DefaultDialer.DialContext(c.Request.Context(), upstreamURL, nil)
	if err != nil {
		_ = clientConn.WriteJSON(gin.H{
			"type": "control_ack",
			"data": gin.H{
				"success": false,
				"message": "连接 SimEngine Core 失败",
			},
		})
		_ = clientConn.Close()
		return
	}

	proxyDone := make(chan struct{}, 2)
	callCtx := websocketServiceContext(c)
	go proxyWebSocket(upstreamConn, clientConn, proxyDone, nil, nil)
	go proxyWebSocket(clientConn, upstreamConn, proxyDone, func() {
		h.instanceService.TouchActivity(callCtx, target.InstanceID)
	}, func(messageType int, payload []byte) {
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			return
		}
		h.instanceService.RecordSimEngineOperation(callCtx, sc, target.InstanceID, payload)
	})
	<-proxyDone
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

// readStudentTerminalLoop 处理学生终端命令输入并回传执行结果。
func (h *InstanceHandler) readStudentTerminalLoop(c *gin.Context, client *wsmanager.Client, manager *wsmanager.Manager, instanceID int64, sc *svcctx.ServiceContext, defaultContainer string) {
	defer func() {
		manager.Unregister(client)
		_ = client.Conn.Close()
	}()

	client.Conn.SetReadLimit(8192)
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
		if inbound.Type != "terminal_command" || strings.TrimSpace(inbound.Command) == "" {
			continue
		}

		containerName := strings.TrimSpace(inbound.Container)
		if containerName == "" {
			containerName = defaultContainer
		}
		result, err := h.instanceService.ExecuteTerminalCommand(callCtx, sc, instanceID, containerName, strings.TrimSpace(inbound.Command))
		if err != nil {
			continue
		}
		h.pushTerminalCommandOutput(client, result)
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

// pushTerminalCommandOutput 推送终端命令执行结果。
func (h *InstanceHandler) pushTerminalCommandOutput(client *wsmanager.Client, output *svc.TerminalCommandOutput) {
	data, err := json.Marshal(terminalCommandOutputMessage{
		Type:      "terminal_output",
		Container: output.Container,
		Command:   output.Command,
		ExitCode:  output.ExitCode,
		Stdout:    output.Stdout,
		Stderr:    output.Stderr,
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
