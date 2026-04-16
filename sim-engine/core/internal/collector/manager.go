package collector

import (
	"errors"
	"sync"
	"time"
)

// Event 表示混合实验采集链路的标准化事件。
type Event struct {
	Source      string
	TimestampMS int64
	DataType    string
	PayloadJSON []byte
}

// Session 表示一个采集会话的运行状态。
type Session struct {
	SessionID        string
	Running          bool
	Mode             string
	ConfigJSON       []byte
	LastEvent        *Event
	LastErrorMessage string
	UpdatedAt        time.Time
}

// Manager 管理混合实验的数据采集会话生命周期。
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

// NewManager 创建采集会话管理器。
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]Session),
	}
}

// Start 标记采集会话已启动并保存配置。
func (m *Manager) Start(sessionID string, mode string, configJSON []byte) error {
	if sessionID == "" {
		return errors.New("collector session id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = Session{
		SessionID:  sessionID,
		Running:    true,
		Mode:       mode,
		ConfigJSON: cloneBytes(configJSON),
		UpdatedAt:  time.Now().UTC(),
	}
	return nil
}

// Stop 标记采集会话已停止。
func (m *Manager) Stop(sessionID string) error {
	if sessionID == "" {
		return errors.New("collector session id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return errors.New("collector session not found")
	}
	session.Running = false
	session.UpdatedAt = time.Now().UTC()
	m.sessions[sessionID] = session
	return nil
}

// RecordEvent 记录最近一次成功采集的事件。
func (m *Manager) RecordEvent(sessionID string, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return errors.New("collector session not found")
	}
	if !session.Running {
		return errors.New("collector session is not running")
	}
	copied := event
	copied.PayloadJSON = cloneBytes(event.PayloadJSON)
	session.LastEvent = &copied
	session.LastErrorMessage = ""
	session.UpdatedAt = time.Now().UTC()
	m.sessions[sessionID] = session
	return nil
}

// RecordError 记录最近一次采集错误。
func (m *Manager) RecordError(sessionID string, err error) error {
	if err == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return errors.New("collector session not found")
	}
	session.LastErrorMessage = err.Error()
	session.UpdatedAt = time.Now().UTC()
	m.sessions[sessionID] = session
	return nil
}

// Get 返回采集会话快照。
func (m *Manager) Get(sessionID string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return Session{}, false
	}
	session.ConfigJSON = cloneBytes(session.ConfigJSON)
	if session.LastEvent != nil {
		copied := *session.LastEvent
		copied.PayloadJSON = cloneBytes(copied.PayloadJSON)
		session.LastEvent = &copied
	}
	return session, true
}

// Delete 删除采集会话。
func (m *Manager) Delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
