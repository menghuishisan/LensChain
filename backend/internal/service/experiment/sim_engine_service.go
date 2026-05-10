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

	// 教师干预（对齐 proto PublishTeacherIntervention，06.md §14.5）
	PublishTeacherIntervention(ctx context.Context, req *SimTeacherInterventionRequest) (*SimTeacherInterventionResult, error)
}

// CreateSimSessionRequest SimEngine 会话创建请求
type CreateSimSessionRequest struct {
	InstanceID     int64
	StudentID      int64
	LinkageEnabled bool
	SessionConfig  json.RawMessage
	Scenes         []SimSceneConfig
}

// SimSceneConfig SimEngine 单场景配置（对齐 proto SceneConfig 消息）
type SimSceneConfig struct {
	SceneCode        string
	ScenarioID       string
	ScenarioVersion  string // 显式锁定的场景版本
	LinkGroupID      string
	LinkGroupCode    string
	LinkGroupVersion string // 显式锁定的联动组版本
	LayoutRole       string // primary | secondary | auxiliary
	DisplayMode      string // single | split-2 | split-3 | grid-4
	LinkToPrimary    bool
	DefaultVisible   bool
	Params           json.RawMessage
	SharedState      json.RawMessage
	DataSourceMode   string
	DataSourceConfig json.RawMessage
	LayoutPosition   json.RawMessage
	// ContainerImageURL 场景算法容器镜像地址（来自 sim_scenarios.container_image_url，必填）。
	// SimEngine SceneManager 据此通过 K8s 按需启动场景 Pod。
	ContainerImageURL string
	// ResourceRequestCPU 场景容器的 CPU 资源请求（K8s 格式，例如 "100m"）。为空则使用 SimEngine 默认值。
	ResourceRequestCPU string
	// ResourceRequestMemory 场景容器的内存资源请求（K8s 格式，例如 "128Mi"）。为空则使用 SimEngine 默认值。
	ResourceRequestMemory string
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

// SimInteraction SimEngine 交互指令（对齐 proto SendInteractionRequest）
type SimInteraction struct {
	SceneCode  string          `json:"scene_code"`
	ActionCode string          `json:"action_code"`
	Params     json.RawMessage `json:"params"`
	ActorID    string          `json:"actor_id"`
	UserRole   string          `json:"user_role"` // 对齐 proto user_role 字段
}

// SimInteractionResult SimEngine 交互结果（对齐 proto SendInteractionResponse）
type SimInteractionResult struct {
	Success             bool            `json:"success"`
	RenderEnvelopeJSON  json.RawMessage `json:"render_envelope_json"` // 场景执行后产生的最新 RenderEnvelope
	ErrorMessage        string          `json:"error_message"`
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

// SimTeacherInterventionRequest 教师干预请求（对齐 proto PublishTeacherInterventionRequest，06.md §14.5）
// 10 类 action_code: broadcast_message / pause_all / resume_all / force_step /
// force_reset / force_link_state / unlock_link_sync / kick_student 等
type SimTeacherInterventionRequest struct {
	InstanceID       string          `json:"instance_id"`
	TeacherID        string          `json:"teacher_id"`
	ActionCode       string          `json:"action_code"`
	TargetSessionIDs []string        `json:"target_session_ids,omitempty"`
	TargetSceneCodes []string        `json:"target_scene_codes,omitempty"`
	TargetLinkGroup  string          `json:"target_link_group,omitempty"`
	Params           json.RawMessage `json:"params,omitempty"`
}

// SimTeacherInterventionResult 教师干预结果（对齐 proto PublishTeacherInterventionResponse）
type SimTeacherInterventionResult struct {
	Success            bool            `json:"success"`
	ErrorMessage       string          `json:"error_message,omitempty"`
	AffectedSessionIDs []string        `json:"affected_session_ids,omitempty"`
	Result             json.RawMessage `json:"result,omitempty"`
}

// 真实实现见 sim_engine_client.go（基于 gRPC）
