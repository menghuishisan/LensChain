package framework

import "time"

// Node 表示场景中的节点或参与者状态。
type Node struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Status     string         `json:"status"`
	X          float64        `json:"x,omitempty"`
	Y          float64        `json:"y,omitempty"`
	Role       string         `json:"role,omitempty"`
	Load       float64        `json:"load,omitempty"`
	HashRate   float64        `json:"hashrate,omitempty"`
	Stake      float64        `json:"stake,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Message 表示场景中的消息、交易或数据流对象。
type Message struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Kind       string         `json:"kind"`
	Status     string         `json:"status"`
	SourceID   string         `json:"source_id,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	X          float64        `json:"x,omitempty"`
	Y          float64        `json:"y,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Metric 表示嵌入可视化画面的关键指标。
type Metric struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone,omitempty"`
}

// TooltipEntry 表示悬停详情中的单行数据。
type TooltipEntry struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// TimelineEvent 表示前端事件时间线中的单项事件。
type TimelineEvent struct {
	ID          string `json:"id"`
	Tick        int64  `json:"tick"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Tone        string `json:"tone,omitempty"`
}

// SceneState 是场景算法容器内部维护的通用状态模型。
type SceneState struct {
	SceneCode    string          `json:"scene_code"`
	Title        string          `json:"title"`
	Tick         int64           `json:"tick"`
	Seed         int64           `json:"seed"`
	Phase        string          `json:"phase"`
	PhaseIndex   int             `json:"phase_index"`
	Progress     float64         `json:"progress"`
	StartedAt    time.Time       `json:"started_at"`
	Nodes        []Node          `json:"nodes,omitempty"`
	Messages     []Message       `json:"messages,omitempty"`
	Metrics      []Metric        `json:"metrics,omitempty"`
	Tooltip      []TooltipEntry  `json:"tooltip,omitempty"`
	Timeline     []TimelineEvent `json:"timeline,omitempty"`
	Stages       []string        `json:"stages,omitempty"`
	ChangedKeys  []string        `json:"changed_keys,omitempty"`
	Linked       bool            `json:"linked,omitempty"`
	LinkGroup    string          `json:"link_group,omitempty"`
	TotalTicks   int64           `json:"total_ticks,omitempty"`
	StepDuration int64           `json:"step_duration_ms,omitempty"`
	Data         map[string]any  `json:"data,omitempty"`
	Extra        map[string]any  `json:"extra,omitempty"`
}

// RenderEnvelope 是场景输出给前端渲染层的标准载荷。
type RenderEnvelope struct {
	Nodes       []Node         `json:"nodes,omitempty"`
	Messages    []Message      `json:"messages,omitempty"`
	Stages      []string       `json:"stages,omitempty"`
	ChangedKeys []string       `json:"changed_keys,omitempty"`
	Phase       string         `json:"phase,omitempty"`
	PhaseIndex  int            `json:"phase_index,omitempty"`
	Progress    float64        `json:"progress,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// Definition 描述一个内置场景的元信息和算法行为钩子。
type Definition struct {
	Code             string
	Name             string
	Description      string
	CategoryCode     string
	AlgorithmType    string
	Version          string
	TimeControlMode  string
	DataSourceMode   string
	DefaultParams    map[string]any
	DefaultState     func() SceneState
	SupportedLinks   []string
	Interaction      func() InteractionDefinition
	Init             func(state *SceneState, input InitInput) error
	Step             func(state *SceneState, input StepInput) (StepOutput, error)
	HandleAction     func(state *SceneState, input ActionInput) (ActionOutput, error)
	SyncSharedState  func(state *SceneState, sharedState map[string]any) error
	BuildRenderState func(state SceneState) RenderEnvelope
}

// InitInput 表示一次初始化需要的全部输入。
type InitInput struct {
	SessionID    string
	Params       map[string]any
	InitialState map[string]any
	SharedState  map[string]any
	Seed         int64
}

// StepInput 表示一步仿真推进的输入。
type StepInput struct {
	SharedState map[string]any
}

// StepOutput 表示一步推进返回的结果。
type StepOutput struct {
	Events     []TimelineEvent
	SharedDiff map[string]any
}

// ActionInput 表示一次交互操作的输入。
type ActionInput struct {
	ActionCode  string
	Params      map[string]any
	ActorID     string
	RoleKey     string
	SharedState map[string]any
}

// ActionOutput 表示一次交互操作的结果。
type ActionOutput struct {
	Success    bool
	Error      string
	Events     []TimelineEvent
	SharedDiff map[string]any
}

// InteractionDefinition 定义场景专属交互面板。
type InteractionDefinition struct {
	SceneCode string
	Actions   []InteractionActionDefinition
}

// InteractionActionDefinition 定义单个操作。
type InteractionActionDefinition struct {
	ActionCode   string
	Label        string
	Description  string
	Trigger      string
	Fields       []InteractionFieldDefinition
	UISchemaJSON []byte
}

// InteractionFieldDefinition 定义单个输入字段。
type InteractionFieldDefinition struct {
	Key            string
	Label          string
	Type           string
	Required       bool
	DefaultValue   string
	Options        []InteractionOptionDefinition
	ValidationJSON []byte
}

// InteractionOptionDefinition 定义选择项。
type InteractionOptionDefinition struct {
	Value string
	Label string
}
