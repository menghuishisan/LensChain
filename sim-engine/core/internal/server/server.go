// Package server 提供 SimEngine Core 的 HTTP、WebSocket 与 gRPC 服务适配层。
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	"github.com/lenschain/sim-engine/core/internal/app"
	"github.com/lenschain/sim-engine/core/internal/collector"
	simws "github.com/lenschain/sim-engine/core/internal/ws"
)

// TokenValidator 定义 SimEngine WebSocket 鉴权器。
type TokenValidator interface {
	Validate(sessionID string, token string) (AccessGrant, error)
}

// AccessGrant 表示一次 WebSocket 鉴权成功后授予的访问能力。
type AccessGrant struct {
	ReadOnly bool
}

// NewHandler 创建 SimEngine Core HTTP 处理器。
func NewHandler(engine *app.Engine) http.Handler {
	return NewHandlerWithValidator(engine, NewDefaultTokenValidator(engine, "", "", ""))
}

// NewHandlerWithValidator 创建带自定义鉴权器的 SimEngine Core HTTP 处理器。
func NewHandlerWithValidator(engine *app.Engine, validator TokenValidator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.Handle("/api/v1/ws/sim-engine/", websocket.Handler(func(conn *websocket.Conn) {
		handleSimEngineWebSocket(engine, validator, conn)
	}))
	mux.Handle("/api/v1/ws/collector/", websocket.Handler(func(conn *websocket.Conn) {
		handleCollectorWebSocket(engine, conn)
	}))
	return mux
}

// handleHealth 返回 SimEngine Core 健康状态。
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleSimEngineWebSocket 处理 SimEngine 会话的数据通道连接。
func handleSimEngineWebSocket(engine *app.Engine, validator TokenValidator, conn *websocket.Conn) {
	if engine == nil {
		_ = conn.Close()
		return
	}

	sessionID := strings.TrimPrefix(conn.Request().URL.Path, "/api/v1/ws/sim-engine/")
	if strings.TrimSpace(sessionID) == "" || !engine.SessionExists(sessionID) {
		_ = conn.Close()
		return
	}
	if validator == nil {
		validator = NewDefaultTokenValidator(engine, "", "", "")
	}
	grant, err := validator.Validate(sessionID, conn.Request().URL.Query().Get("token"))
	if err != nil {
		_ = conn.Close()
		return
	}

	subscription := engine.SubscribeMessages(sessionID)
	defer subscription.Close()
	defer conn.Close()

	_ = engine.RecoverLatestTickSnapshot(sessionID)
	for _, msg := range engine.CurrentMessages(sessionID) {
		if err := websocket.JSON.Send(conn, msg); err != nil {
			return
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for message := range subscription.C {
			if err := websocket.JSON.Send(conn, message); err != nil {
				return
			}
		}
	}()

	for {
		var message simws.Message
		if err := websocket.JSON.Receive(conn, &message); err != nil {
			return
		}
		switch message.Type {
		case simws.MessageTypeControl:
			if grant.ReadOnly {
				return
			}
			command, params := decodeControlPayload(message.PayloadJSON)
			_ = engine.ControlTime(sessionID, command, params)
		case simws.MessageTypeAction:
			if grant.ReadOnly {
				return
			}
			actionCode, actorID, roleKey, params := decodeActionPayload(message.PayloadJSON)
			_, _ = engine.SendInteraction(
				conn.Request().Context(),
				sessionID,
				message.SceneCode,
				actionCode,
				params,
				app.InteractionContext{
					ActorID: actorID,
					RoleKey: roleKey,
				},
			)
		case simws.MessageTypeRewindTo:
			if grant.ReadOnly {
				return
			}
			_ = engine.ControlTime(sessionID, "rewind_to", message.PayloadJSON)
		}

		select {
		case <-done:
			return
		default:
		}
	}
}

// handleCollectorWebSocket 处理 Collector sidecar 的事件注入连接。
func handleCollectorWebSocket(engine *app.Engine, conn *websocket.Conn) {
	if engine == nil {
		_ = conn.Close()
		return
	}

	sessionID := strings.TrimPrefix(conn.Request().URL.Path, "/api/v1/ws/collector/")
	if strings.TrimSpace(sessionID) == "" || !engine.SessionExists(sessionID) {
		_ = conn.Close()
		return
	}
	defer conn.Close()

	for {
		var message struct {
			Source      string          `json:"source"`
			TimestampMS int64           `json:"timestamp"`
			DataType    string          `json:"data_type"`
			Payload     json.RawMessage `json:"payload"`
		}
		if err := websocket.JSON.Receive(conn, &message); err != nil {
			return
		}
		timestampMS := message.TimestampMS
		if timestampMS == 0 {
			timestampMS = time.Now().UTC().UnixMilli()
		}
		if err := engine.InjectCollectionEvent(sessionID, collector.Event{
			Source:      strings.TrimSpace(message.Source),
			TimestampMS: timestampMS,
			DataType:    strings.TrimSpace(message.DataType),
			PayloadJSON: message.Payload,
		}); err != nil {
			return
		}
	}
}

// decodeControlPayload 解析时间控制消息负载。
func decodeControlPayload(payload []byte) (string, []byte) {
	var data struct {
		Command string  `json:"command"`
		Value   float64 `json:"value"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", payload
	}
	return data.Command, payload
}

// decodeActionPayload 解析交互动作消息负载。
func decodeActionPayload(payload []byte) (string, string, string, []byte) {
	var data struct {
		ActionCode string          `json:"action_code"`
		ActorID    string          `json:"actor_id"`
		RoleKey    string          `json:"role_key"`
		Params     json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", "", "", payload
	}
	return data.ActionCode, data.ActorID, data.RoleKey, data.Params
}
