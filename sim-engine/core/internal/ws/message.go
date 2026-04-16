package ws

import (
	"encoding/json"
	"sync"
)

// MessageType 是 SimEngine WebSocket 标准消息类型。
type MessageType string

const (
	// MessageTypeStateDiff 表示 tick 增量状态推送。
	MessageTypeStateDiff MessageType = "state_diff"
	// MessageTypeStateFull 表示初始化或恢复时的完整状态推送。
	MessageTypeStateFull MessageType = "state_full"
	// MessageTypeEvent 表示过程事件通知。
	MessageTypeEvent MessageType = "event"
	// MessageTypeLinkUpdate 表示联动共享状态更新通知。
	MessageTypeLinkUpdate MessageType = "link_update"
	// MessageTypeControlAck 表示时间控制命令确认。
	MessageTypeControlAck MessageType = "control_ack"
	// MessageTypeSnapshot 表示快照创建或恢复通知。
	MessageTypeSnapshot MessageType = "snapshot"
	// MessageTypeAction 表示前端发起的场景交互消息。
	MessageTypeAction MessageType = "action"
	// MessageTypeControl 表示前端发起的时间控制消息。
	MessageTypeControl MessageType = "control"
	// MessageTypeRewindTo 表示前端发起的定点回退消息。
	MessageTypeRewindTo MessageType = "rewind_to"
	// MessageTypeTeacherSummary 表示教师监控摘要推送。
	MessageTypeTeacherSummary MessageType = "teacher_summary"
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
