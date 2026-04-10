// manager.go
// WebSocket 连接管理器
// 用于实时通信：实验环境状态推送（模块04）、CTF排行榜/公告（模块05）、通知推送（模块07）
// 支持按用户ID、房间（竞赛/课程/实验组）进行消息推送
// 支持同一用户多设备连接（每个连接独立管理）

package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/pkg/logger"
)

// Upgrader WebSocket 升级器
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应限制来源
	},
}

// connIDCounter 全局连接ID计数器（用于区分同一用户的多个连接）
var connIDCounter uint64

// Message WebSocket 消息结构
type Message struct {
	Type    string      `json:"type"`    // 消息类型
	Channel string      `json:"channel"` // 频道（如 leaderboard, notification）
	Data    interface{} `json:"data"`    // 消息数据
}

// Client WebSocket 客户端连接
type Client struct {
	UserID int64           // 用户ID
	ConnID uint64          // 连接唯一ID（区分同一用户多设备）
	Conn   *websocket.Conn // WebSocket 连接
	Send   chan []byte      // 发送消息通道
	Rooms  map[string]bool  // 已加入的房间
	mu     sync.Mutex
}

// Manager WebSocket 连接管理器
type Manager struct {
	clients    map[uint64]*Client              // 连接ID -> 客户端
	userConns  map[int64]map[uint64]*Client     // 用户ID -> 连接ID集合（支持多设备）
	rooms      map[string]map[uint64]*Client    // 房间名 -> 客户端集合
	register   chan *Client
	unregister chan *Client
	broadcast  chan *RoomMessage
	mu         sync.RWMutex
}

// RoomMessage 房间消息
type RoomMessage struct {
	Room    string
	Message []byte
}

// 全局管理器实例
var manager *Manager

// Init 初始化 WebSocket 管理器
func Init() {
	manager = &Manager{
		clients:    make(map[uint64]*Client),
		userConns:  make(map[int64]map[uint64]*Client),
		rooms:      make(map[string]map[uint64]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *RoomMessage, 256),
	}
	go manager.run()
	logger.L.Info("WebSocket管理器已启动")
}

// GetManager 获取全局管理器
func GetManager() *Manager {
	return manager
}

// run 管理器主循环
func (m *Manager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.ConnID] = client
			if _, ok := m.userConns[client.UserID]; !ok {
				m.userConns[client.UserID] = make(map[uint64]*Client)
			}
			m.userConns[client.UserID][client.ConnID] = client
			m.mu.Unlock()
			logger.L.Debug("WebSocket客户端已连接",
				zap.Int64("user_id", client.UserID),
				zap.Uint64("conn_id", client.ConnID),
			)

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client.ConnID]; ok {
				// 从所有房间移除
				for room := range client.Rooms {
					if roomClients, ok := m.rooms[room]; ok {
						delete(roomClients, client.ConnID)
						if len(roomClients) == 0 {
							delete(m.rooms, room)
						}
					}
				}
				// 从用户连接集合移除
				if conns, ok := m.userConns[client.UserID]; ok {
					delete(conns, client.ConnID)
					if len(conns) == 0 {
						delete(m.userConns, client.UserID)
					}
				}
				delete(m.clients, client.ConnID)
				close(client.Send)
			}
			m.mu.Unlock()
			logger.L.Debug("WebSocket客户端已断开",
				zap.Int64("user_id", client.UserID),
				zap.Uint64("conn_id", client.ConnID),
			)

		case msg := <-m.broadcast:
			m.mu.RLock()
			if roomClients, ok := m.rooms[msg.Room]; ok {
				for connID, client := range roomClients {
					select {
					case client.Send <- msg.Message:
					default:
						// 发送缓冲区满，关闭连接
						close(client.Send)
						delete(roomClients, connID)
					}
				}
			}
			m.mu.RUnlock()
		}
	}
}

// Register 注册客户端
func (m *Manager) Register(client *Client) {
	m.register <- client
}

// Unregister 注销客户端
func (m *Manager) Unregister(client *Client) {
	m.unregister <- client
}

// JoinRoom 加入房间
func (m *Manager) JoinRoom(client *Client, room string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.rooms[room]; !ok {
		m.rooms[room] = make(map[uint64]*Client)
	}
	m.rooms[room][client.ConnID] = client
	client.mu.Lock()
	client.Rooms[room] = true
	client.mu.Unlock()
}

// LeaveRoom 离开房间
func (m *Manager) LeaveRoom(client *Client, room string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if roomClients, ok := m.rooms[room]; ok {
		delete(roomClients, client.ConnID)
		if len(roomClients) == 0 {
			delete(m.rooms, room)
		}
	}
	client.mu.Lock()
	delete(client.Rooms, room)
	client.mu.Unlock()
}

// SendToUser 向指定用户的所有连接发送消息
func (m *Manager) SendToUser(userID int64, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	m.mu.RLock()
	conns, ok := m.userConns[userID]
	m.mu.RUnlock()

	if ok {
		for _, client := range conns {
			select {
			case client.Send <- data:
			default:
				// 缓冲区满，跳过该连接
			}
		}
	}
	return nil
}

// BroadcastToRoom 向房间广播消息
func (m *Manager) BroadcastToRoom(room string, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	m.broadcast <- &RoomMessage{Room: room, Message: data}
	return nil
}

// GetOnlineUserCount 获取在线用户数
func (m *Manager) GetOnlineUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.userConns)
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conns, ok := m.userConns[userID]
	return ok && len(conns) > 0
}

// WritePump 客户端写入协程
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump 客户端读取协程
func (c *Client) ReadPump(m *Manager) {
	defer func() {
		m.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// NewClient 创建新的 WebSocket 客户端
// 自动分配唯一连接ID，支持同一用户多设备连接
func NewClient(userID int64, conn *websocket.Conn) *Client {
	connID := atomic.AddUint64(&connIDCounter, 1)
	return &Client{
		UserID: userID,
		ConnID: connID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Rooms:  make(map[string]bool),
	}
}
