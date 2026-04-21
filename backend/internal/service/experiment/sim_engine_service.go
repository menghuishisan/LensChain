// sim_engine_service.go
// 模块04 — 实验环境：SimEngine 适配契约
// 定义模块04与 SimEngine Core 通信所需的最小接口和数据结构，供 service 层编排仿真会话
// 该文件只声明外部系统适配契约，不承载实验业务规则

package experiment

import (
	"context"
	"encoding/json"
)

// SimEngineService SimEngine 通信服务接口
// 封装所有与 SimEngine Core 的交互操作
type SimEngineService interface {
	// 会话管理
	CreateSession(ctx context.Context, req *CreateSimSessionRequest) (*SimSession, error)
	DestroySession(ctx context.Context, sessionID string) error
	GetSessionState(ctx context.Context, sessionID string) (*SimSessionState, error)

	// 交互操作
	SendInteraction(ctx context.Context, sessionID string, interaction *SimInteraction) (*SimInteractionResult, error)
	GetInteractionSchema(ctx context.Context, sessionID string, sceneCode string) (*SimInteractionSchema, error)

	// 时间控制
	ControlTime(ctx context.Context, sessionID string, action string, params json.RawMessage) error

	// 快照
	CreateSnapshot(ctx context.Context, sessionID string) (*SimSnapshot, error)
	RestoreSnapshot(ctx context.Context, sessionID string, snapshotID string) error

	// 数据采集（混合实验用）
	StartDataCollection(ctx context.Context, sessionID string, config json.RawMessage) error
	StopDataCollection(ctx context.Context, sessionID string) error
}

// CreateSimSessionRequest SimEngine 会话创建请求
type CreateSimSessionRequest struct {
	InstanceID     int64
	StudentID      int64
	LinkageEnabled bool
	SessionConfig  json.RawMessage
	Scenes         []SimSceneConfig
}

// SimSceneConfig SimEngine 单场景配置
type SimSceneConfig struct {
	SceneCode        string
	ScenarioID       string
	LinkGroupID      string
	LinkGroupCode    string
	Params           json.RawMessage
	InitialState     json.RawMessage
	DataSourceConfig json.RawMessage
	LayoutPosition   json.RawMessage
	DataSourceMode   string
	SharedState      json.RawMessage
}

// SimSession SimEngine 会话信息
type SimSession struct {
	SessionID        string
	InstanceID       string
	Status           string
	WebSocketURL     string
	ActiveSceneCodes []string
}

// SimSessionState SimEngine 会话状态
type SimSessionState struct {
	SessionID        string
	InstanceID       string
	Status           string
	SimTick          int64
	SimTimeSeconds   float64
	Speed            float64
	ActiveSceneCodes []string
	LinkGroupCodes   []string
	SceneState       json.RawMessage
	LastAction       string
	UpdatedAtMS      int64
}

// SimInteraction SimEngine 交互指令
type SimInteraction struct {
	SceneCode  string          `json:"scene_code"`
	ActionCode string          `json:"action_code"`
	Params     json.RawMessage `json:"params"`
	ActorID    string          `json:"actor_id"`
	RoleKey    string          `json:"role_key"`
}

// SimInteractionResult SimEngine 交互结果
type SimInteractionResult struct {
	Success      bool            `json:"success"`
	Data         json.RawMessage `json:"data"`
	ErrorMessage string          `json:"error_message"`
}

// SimInteractionSchema SimEngine 场景交互面板定义
type SimInteractionSchema struct {
	SceneCode string          `json:"scene_code"`
	Actions   json.RawMessage `json:"actions"`
}

// SimSnapshot SimEngine 快照
type SimSnapshot struct {
	SnapshotID   string
	SessionID    string
	SnapshotType string
	SimTick      int64
	ObjectURL    string
	CreatedAtMS  int64
}

// 真实实现见 sim_engine_client.go（基于 gRPC）
