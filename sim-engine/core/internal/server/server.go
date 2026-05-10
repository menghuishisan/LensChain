// Package server 提供 SimEngine Core 的 HTTP、WebSocket 与 gRPC 服务适配层。
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

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
	mux.HandleFunc("/api/v1/ws/sim-engine/", func(w http.ResponseWriter, r *http.Request) {
		handleSimEngineWS(engine, validator, w, r)
	})
	mux.HandleFunc("/api/v1/ws/collector/", func(w http.ResponseWriter, r *http.Request) {
		handleCollectorWS(engine, w, r)
	})
	return mux
}

// handleHealth 返回 SimEngine Core 健康状态。
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// upgrader WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，由后端CORS中间件控制
	},
}

// handleSimEngineWS 处理 SimEngine 会话的数据通道连接。
//
// 重要：所有错误路径必须写入 HTTP 状态码并 return，禁止裸 return。
// 裸 return 在 net/http 下会被默认填充为 200 OK 空响应，上游后端代理（gorilla
// DialContext）只会接受 101 Switching Protocols，非 101 一律报 "websocket: bad
// handshake"。让 Core 的鉴权 / 路径错误以正确状态码可观测，是排错的前提。
func handleSimEngineWS(engine *app.Engine, validator TokenValidator, w http.ResponseWriter, r *http.Request) {
	if engine == nil {
		http.Error(w, "sim-engine not initialized", http.StatusServiceUnavailable)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/ws/sim-engine/")
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if !engine.SessionExists(sessionID) {
		// 业务侧最常见的原因：Core 重启导致内存中的会话丢失，前端仍持有旧 sessionID。
		// 必须以 404 暴露给上游，方便代理日志与运维定位，而不是吞成 200。
		slog.Warn("sim-engine ws: session not found", "session_id", sessionID, "remote", r.RemoteAddr)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if validator == nil {
		validator = NewDefaultTokenValidator(engine, "", "", "")
	}
	grant, err := validator.Validate(sessionID, r.URL.Query().Get("token"))
	if err != nil {
		slog.Warn("sim-engine ws: token validation failed", "session_id", sessionID, "err", err.Error())
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// gorilla 已经在 Upgrade 内部写入了 4xx 响应。
		slog.Warn("sim-engine ws: upgrade failed", "session_id", sessionID, "err", err.Error())
		return
	}
	defer conn.Close()

	subscription := engine.SubscribeMessages(sessionID)
	defer subscription.Close()

	_ = engine.RecoverLatestTickSnapshot(sessionID)
	for _, msg := range engine.CurrentMessages(sessionID) {
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for message := range subscription.C {
			if err := conn.WriteJSON(message); err != nil {
				return
			}
		}
	}()

	for {
		var message simws.Message
		if err := conn.ReadJSON(&message); err != nil {
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
			actionCode, actorID, userRole, params := decodeActionPayload(message.PayloadJSON)
			_, _ = engine.SendInteraction(
				r.Context(),
				sessionID,
				message.SceneCode,
				actionCode,
				params,
				app.InteractionContext{
					ActorID:  actorID,
					UserRole: userRole,
				},
			)
		case simws.MessageTypeStepBack:
			if grant.ReadOnly {
				return
			}
			// step_back 打包为 control 语义交给 ControlTime（仅单场景 process 有效，上层会校验）。
			_ = engine.StepBack(sessionID)
		}

		select {
		case <-done:
			return
		default:
		}
	}
}

// handleCollectorWS 处理 Collector sidecar 的事件注入连接。
// 错误路径同样必须写入状态码（详见 handleSimEngineWS 注释）。
func handleCollectorWS(engine *app.Engine, w http.ResponseWriter, r *http.Request) {
	if engine == nil {
		http.Error(w, "sim-engine not initialized", http.StatusServiceUnavailable)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/ws/collector/")
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if !engine.SessionExists(sessionID) {
		slog.Warn("sim-engine collector: session not found", "session_id", sessionID, "remote", r.RemoteAddr)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("sim-engine collector: upgrade failed", "session_id", sessionID, "err", err.Error())
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
		if err := conn.ReadJSON(&message); err != nil {
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
		UserRole   string          `json:"user_role"`
		Params     json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", "", "", payload
	}
	return data.ActionCode, data.ActorID, data.UserRole, data.Params
}
