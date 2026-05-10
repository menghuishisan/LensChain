// 模块：sim-engine/core/internal/ws
// 文件职责：定义 SimEngine Core ↔ 前端 WebSocket 协议消息与会话级广播 Hub。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §七 / 03-API §四。
//
// 关键约束：
// 1. 唯一渲染数据通道为 `render`；统一承载首帧（is_full_snapshot=true）与增量帧。
// 2. 单步回退使用 `step_back` 顶层消息，不复用 `control.command`，对齐 §7.4。
// 3. 教师监控摘要走 `event` 携带 `event=teacher_summary`，不再单独保留独立消息类型。

package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// MessageType 是 SimEngine WebSocket 标准消息类型。
type MessageType string

const (
	// 后端 → 前端

	// MessageTypeRender 表示渲染数据下发（每 tick / Action 响应 / 重连首帧）。
	// 载荷为 RenderEnvelope（含 primitives / micro_steps / link_triggers /
	// container_data / changed_keys / is_full_snapshot）。
	MessageTypeRender MessageType = "render"
	// MessageTypeEvent 表示仿真事件通知（侧栏日志 / 提示）。
	MessageTypeEvent MessageType = "event"
	// MessageTypeControlAck 表示控制指令确认。
	MessageTypeControlAck MessageType = "control_ack"
	// MessageTypeSchemaInvalidated 表示 ActionDef schema 失效（教师调试发布）。
	MessageTypeSchemaInvalidated MessageType = "schema_invalidated"

	// 前端 → 后端

	// MessageTypeAction 表示用户交互操作（ActionDef 调用）。
	MessageTypeAction MessageType = "action"
	// MessageTypeControl 表示仿真时间控制（play / pause / step / set_speed / reset / resume）。
	MessageTypeControl MessageType = "control"
	// MessageTypeStepBack 表示单步回退（仅单场景 process 模式有效）。
	MessageTypeStepBack MessageType = "step_back"
)

// Message 是前端与 SimEngine Core 之间的数据通道消息。
type Message struct {
	Type        MessageType     `json:"type"`
	SceneCode   string          `json:"scene_code,omitempty"`
	Tick        int64           `json:"tick"`
	TimestampMS int64           `json:"timestamp"`
	PayloadJSON json.RawMessage `json:"payload"`
}

// Encode 编码 WebSocket 消息。
func Encode(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// Decode 解码 WebSocket 消息。
func Decode(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

// =====================================================================
// 标准载荷结构（对照 06.md §7.3 / §7.4）
// =====================================================================

// ActionPayload 是前端 → 后端 action 消息载荷。
type ActionPayload struct {
	ActionCode string                 `json:"action_code"`
	Params     map[string]any         `json:"params"`
	ActorID    string                 `json:"actor_id,omitempty"`
	UserRole   string                 `json:"user_role,omitempty"`
}

// ControlPayload 是前端 → 后端 control 消息载荷。
// command 取值：play / pause / step / set_speed / reset / resume。
// step_back 通过独立 MessageTypeStepBack 表达，不在此处。
type ControlPayload struct {
	Command string  `json:"command"`
	Value   float64 `json:"value,omitempty"`
}

// EventPayload 是后端 → 前端 event 消息载荷。
type EventPayload struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data,omitempty"`
}

// ControlAckPayload 是后端 → 前端 control_ack 消息载荷。
type ControlAckPayload struct {
	Command string `json:"command"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SchemaInvalidatedPayload 是后端 → 前端 schema_invalidated 消息载荷。
type SchemaInvalidatedPayload struct {
	Reason string `json:"reason"`
}

// =====================================================================
// 会话级广播 Hub
// =====================================================================

// Subscription 表示一个会话订阅。
type Subscription struct {
	C       <-chan Message
	closeFn func()
}

// Close 关闭订阅。
func (s Subscription) Close() {
	if s.closeFn != nil {
		s.closeFn()
	}
}

// Hub 管理会话级消息广播。
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Message]struct{}
}

// NewHub 创建 WebSocket 会话消息中心。
func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[chan Message]struct{})}
}

// Subscribe 订阅指定会话消息。
func (h *Hub) Subscribe(sessionID string) Subscription {
	ch := make(chan Message, 8)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan Message]struct{})
	}
	h.subscribers[sessionID][ch] = struct{}{}
	return Subscription{
		C: ch,
		closeFn: func() {
			h.unsubscribe(sessionID, ch)
		},
	}
}

// Publish 向会话内所有订阅者广播消息。
func (h *Hub) Publish(sessionID string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers[sessionID] {
		select {
		case ch <- msg:
		default:
			log.Printf("[ws.Hub] message dropped: session=%s type=%s scene=%s tick=%d (subscriber channel full)",
				sessionID, msg.Type, msg.SceneCode, msg.Tick)
		}
	}
}

// unsubscribe 取消单个会话订阅并在无订阅者时清理会话项。
func (h *Hub) unsubscribe(sessionID string, ch chan Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscribers[sessionID] == nil {
		return
	}
	delete(h.subscribers[sessionID], ch)
	close(ch)
	if len(h.subscribers[sessionID]) == 0 {
		delete(h.subscribers, sessionID)
	}
}
