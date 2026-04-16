package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	// StatusRunning 表示仿真会话正在运行。
	StatusRunning = "running"
	// StatusPaused 表示仿真会话已暂停。
	StatusPaused = "paused"
	// StatusError 表示仿真会话处于异常状态。
	StatusError = "error"
)

// SceneConfig 是单个仿真场景的会话配置。
type SceneConfig struct {
	SceneCode string
}

// CreateRequest 是会话创建请求。
type CreateRequest struct {
	InstanceID string
	StudentID  string
	Scenes     []SceneConfig
}

// Session 表示一个学生实验实例对应的仿真会话。
type Session struct {
	SessionID        string
	InstanceID       string
	StudentID        string
	Status           string
	ActiveSceneCodes []string
	CreatedAt        time.Time
}

// Manager 管理 SimEngine 会话生命周期。
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

// NewManager 创建会话管理器。
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]Session),
	}
}

// Create 创建并保存会话。
func (m *Manager) Create(req CreateRequest) (Session, error) {
	if len(req.Scenes) == 0 {
		return Session{}, errors.New("session must contain at least one scene")
	}

	sessionID, err := newSessionID()
	if err != nil {
		return Session{}, err
	}

	sceneCodes := make([]string, 0, len(req.Scenes))
	for _, scene := range req.Scenes {
		sceneCodes = append(sceneCodes, scene.SceneCode)
	}

	session := Session{
		SessionID:        sessionID,
		InstanceID:       req.InstanceID,
		StudentID:        req.StudentID,
		Status:           StatusRunning,
		ActiveSceneCodes: sceneCodes,
		CreatedAt:        time.Now().UTC(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = session
	return session, nil
}

// Get 按会话 ID 获取会话。
func (m *Manager) Get(sessionID string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[sessionID]
	return session, ok
}

// Destroy 删除会话。
func (m *Manager) Destroy(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[sessionID]; !ok {
		return errors.New("session not found")
	}
	delete(m.sessions, sessionID)
	return nil
}

// newSessionID 生成新的仿真会话 ID。
func newSessionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "sim-" + hex.EncodeToString(raw[:]), nil
}
