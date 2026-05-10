// 模块：sim-engine/framework
// 文件职责：SimEngine 协议契约 SSOT 的 Go 类型定义。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.2 / §3.3 / §3.4 / §6.2 / §6.3。
//
// 设计要点：
// 1. framework 是独立 Go module（`module github.com/lenschain/sim-engine/framework`），
//    被 sdk / scenarios / core 三方共同 import；不依赖 sim-engine 内任何其它 module。
// 2. 本文件类型与 proto/lenschain/sim_scenario/v1/sim_scenario.proto、
//    renderers/shared/types.ts 三方 1:1 对齐；任一方变更必须三端同步。
// 3. JSON 字段统一 snake_case，时间统一 Unix 毫秒时间戳。
// 4. 场景容器无状态：所有跨 Step 状态通过 SceneState 序列化往返。
// 5. SharedState 字段写入受 owner 模型约束（详 §8.3），由 Core LinkEngine 校验。

package framework

// =====================================================================
// 47 原语 / Layer / 时间控制 / 类目 等枚举常量
// =====================================================================

// PrimitiveType 47 个原语 type 值（详 06.md §3.2）。
type PrimitiveType string

const (
	// 几何类（8 个）
	PrimGeometryNode     PrimitiveType = "node"
	PrimGeometryEdge     PrimitiveType = "edge"
	PrimGeometryBar      PrimitiveType = "bar"
	PrimGeometryCurve    PrimitiveType = "curve"
	PrimGeometryPolygon  PrimitiveType = "polygon"
	PrimGeometryArea     PrimitiveType = "area"
	PrimGeometryGridCell PrimitiveType = "grid_cell"
	PrimGeometryRing     PrimitiveType = "ring"

	// 动效类（7 个）
	PrimEffectParticleStream PrimitiveType = "particle_stream"
	PrimEffectBurst          PrimitiveType = "burst"
	PrimEffectPulse          PrimitiveType = "pulse"
	PrimEffectTrail          PrimitiveType = "trail"
	PrimEffectGlow           PrimitiveType = "glow"
	PrimEffectShake          PrimitiveType = "shake"
	PrimEffectShiftAnimation PrimitiveType = "shift_animation"

	// 布局类（6 个）
	PrimLayoutHorizontalLane PrimitiveType = "horizontal_lane"
	PrimLayoutStack          PrimitiveType = "stack"
	PrimLayoutRing           PrimitiveType = "ring_layout"
	PrimLayoutTree           PrimitiveType = "tree_layout"
	PrimLayoutGraph          PrimitiveType = "graph_layout"
	PrimLayoutMatrix         PrimitiveType = "matrix_layout"

	// 数据展示类（7 个）
	PrimDataLabel        PrimitiveType = "label"
	PrimDataTooltip      PrimitiveType = "tooltip"
	PrimDataAnnotation   PrimitiveType = "annotation"
	PrimDataRegisterRow  PrimitiveType = "register_row"
	PrimDataMathPipeline PrimitiveType = "math_pipeline"
	PrimDataCodeBlock    PrimitiveType = "code_block"
	PrimDataMathFormula  PrimitiveType = "math_formula"

	// 状态指示类（8 个）
	PrimStatePhaseProgress       PrimitiveType = "phase_progress"
	PrimStateProgressBar         PrimitiveType = "progress_bar"
	PrimStateTargetZone          PrimitiveType = "target_zone"
	PrimStateLinkIndicator       PrimitiveType = "link_indicator"
	PrimStateExternalEventMarker PrimitiveType = "external_event_marker"
	PrimStateErrorOverlay        PrimitiveType = "error_overlay"
	PrimStateVerifyPathHighlight PrimitiveType = "verify_path_highlight"
	PrimStateRiskGauge           PrimitiveType = "risk_gauge"

	// 领域复合类（11 个）
	PrimDomainVoteMatrix    PrimitiveType = "vote_matrix"
	PrimDomainDualTrack     PrimitiveType = "dual_track"
	PrimDomainTimeWheel     PrimitiveType = "time_wheel"
	PrimDomainPieChart      PrimitiveType = "pie_chart"
	PrimDomainSankeyFlow    PrimitiveType = "sankey_flow"
	PrimDomainHeatMap       PrimitiveType = "heat_map"
	PrimDomainMempoolSlot   PrimitiveType = "mempool_slot"
	PrimDomainBridgeTrack   PrimitiveType = "bridge_track"
	PrimDomainCodeMarker    PrimitiveType = "code_marker"
	PrimDomainPartitionZone PrimitiveType = "partition_zone"
	PrimDomainCurvePoint    PrimitiveType = "curve_point"
)

// PrimitiveLayer 原语分层（详 06.md §3.3）。同层按数组顺序绘制（先入底）。
type PrimitiveLayer string

const (
	LayerBackground PrimitiveLayer = "background"
	LayerContent    PrimitiveLayer = "content"
	LayerEffect     PrimitiveLayer = "effect"
	LayerOverlay    PrimitiveLayer = "overlay"
)

// TimeControlMode 三种时间控制模式（详 06.md §五）。
type TimeControlMode string

const (
	TimeControlProcess    TimeControlMode = "process"
	TimeControlReactive   TimeControlMode = "reactive"
	TimeControlContinuous TimeControlMode = "continuous"
)

// DataSourceMode 场景数据源模式。
type DataSourceMode string

const (
	DataSourceSimulation DataSourceMode = "simulation"
	DataSourceCollection DataSourceMode = "collection"
	DataSourceDual       DataSourceMode = "dual"
)

// SceneCategory 场景类目（皮肤选择依据，9 选 1，详 06.md §3.5）。
type SceneCategory string

const (
	CategoryNodeNetwork    SceneCategory = "node_network"
	CategoryConsensus      SceneCategory = "consensus"
	CategoryCryptography   SceneCategory = "cryptography"
	CategoryDataStructure  SceneCategory = "data_structure"
	CategoryTransaction    SceneCategory = "transaction"
	CategorySmartContract  SceneCategory = "smart_contract"
	CategoryAttackSecurity SceneCategory = "attack_security"
	CategoryEconomic       SceneCategory = "economic"
	CategoryGeneric        SceneCategory = "generic" // 教师 fallback；走 L3 自定义渲染器
)

// ActionCategory ActionDef 业务语义分类（详 06.md §6.3）。
type ActionCategory string

const (
	ActionParamTune     ActionCategory = "param_tune"
	ActionAttackInject  ActionCategory = "attack_inject"
	ActionPrimary       ActionCategory = "primary"
	ActionObserve       ActionCategory = "observe"
)

// ActionTrigger ActionDef 触发方式（详 06.md §6.3）。
type ActionTrigger string

const (
	TriggerSubmit    ActionTrigger = "submit"
	TriggerImmediate ActionTrigger = "immediate"
	TriggerHold      ActionTrigger = "hold"
)

// FieldType FieldDef 字段类型（详 06.md §6.3）。
type FieldType string

const (
	FieldString      FieldType = "string"
	FieldNumber      FieldType = "number"
	FieldBoolean     FieldType = "boolean"
	FieldSelect      FieldType = "select"
	FieldEnum        FieldType = "enum"
	FieldRange       FieldType = "range"
	FieldJSON        FieldType = "json"
	FieldMultiSelect FieldType = "multi_select"
)

// HybridChannel 混合实验下 ActionDef 执行通道。
type HybridChannel string

const (
	HybridChannelSim       HybridChannel = "sim"
	HybridChannelContainer HybridChannel = "container"
)

// UserRole 调用 ActionDef 时的用户角色。
type UserRole string

const (
	RoleStudent UserRole = "student"
	RoleTeacher UserRole = "teacher"
)

// RolesAll / RolesStudentOnly / RolesTeacherOnly 是 ActionDef.Roles 的常用集合常量。
//
// 任何 ActionDef.Roles 字段都可直接复用本组常量，避免分散重复 `[]UserRole{...}` 字面量。
// 教师专属 ActionDef 必须使用 RolesTeacherOnly（详 §0.7.5 / §0.7.8）。
var (
	RolesAll          = []UserRole{RoleStudent, RoleTeacher}
	RolesStudentOnly  = []UserRole{RoleStudent}
	RolesTeacherOnly  = []UserRole{RoleTeacher}
)

// ExtensionLevel 场景扩展层级（详 §6.1 catalog 元信息要求）。
//
//	L1：平台内置（catalog 注册的 43 场景）
//	L2：平台扩展（平台二次开发，进 sim_scenarios 表 extension_level=2）
//	L3：教师自定义（教师上传，category=generic + custom_renderer_package）
type ExtensionLevel string

const (
	ExtensionL1 ExtensionLevel = "L1"
	ExtensionL2 ExtensionLevel = "L2"
	ExtensionL3 ExtensionLevel = "L3"
)

// InterveneType 教师干预类型（与 06.md §10.9 / 06.2 §9.7 teacher_intervene_logs 10 类对齐）。
//
// 凡 Roles 含 RoleTeacher 的 ActionDef 都必须填本字段；Core 写审计日志时按本字段分类。
type InterveneType string

const (
	InterveneHint        InterveneType = "hint"          // 教学提示 / broadcast_hint
	InterveneFault       InterveneType = "fault"         // 注入故障
	InterveneAttack      InterveneType = "attack"        // 启用攻击
	IntervenePhase       InterveneType = "phase"         // 强制相位
	InterveneTopology    InterveneType = "topology"      // 改拓扑
	InterveneState       InterveneType = "state"         // 直接改状态 / set_storage
	InterveneReset       InterveneType = "reset"         // 重置 / unjail
	InterveneEpoch       InterveneType = "epoch"         // 强制 epoch 推进
	InterveneRevert      InterveneType = "revert"        // 强制回滚 / force_revert
	InterveneFreeze      InterveneType = "freeze"        // 冻结 / freeze_mempool
)

// =====================================================================
// 协议数据结构（与 06.md §6.2 / §6.3 1:1 对齐）
// =====================================================================

// Primitive 单个原语描述（详 06.md §6.2 Primitive）。
// Params 字段需按 §3.2 各原语 schema 提供，平台不强校验类型，但运行时基类会按字段名取值。
type Primitive struct {
	ID           string         `json:"id"`
	Type         PrimitiveType  `json:"type"`
	Layer        PrimitiveLayer `json:"layer"`
	Params       map[string]any `json:"params"`
	Clickable    bool           `json:"clickable,omitempty"`
	HoverTooltip string         `json:"hover_tooltip,omitempty"`
	ClickAction  string         `json:"click_action,omitempty"` // 绑定 ActionDef.ActionCode
}

// MicroStep 微动画步骤（详 06.md §6.2 MicroStep / §5.3）。
// DurationMs 必须 ≥ 本步骤内 fire_primitives 最长动画时长 + 200ms buffer（详 §3.10.1）。
type MicroStep struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	DurationMs     int      `json:"duration_ms"`
	HighlightIDs   []string `json:"highlight_ids,omitempty"`
	FirePrimitives []string `json:"fire_primitives,omitempty"`
	IsLinkTrigger  bool     `json:"is_link_trigger,omitempty"`
	LinkSource     string   `json:"link_source,omitempty"`
	ParentPhase    string   `json:"parent_phase,omitempty"`
}

// LinkTrigger 跨场景联动事件（详 06.md §6.2 LinkTrigger / §8.5.2 / §9.4）。
// SourceAnchorID / TargetAnchorID 用于 M8 跨画布弧线的起终点定位。
type LinkTrigger struct {
	ID             string         `json:"id"`
	SourceScene    string         `json:"source_scene"`
	SourceAction   string         `json:"source_action"`
	LinkGroup      string         `json:"link_group"`
	ChangedFields  []string       `json:"changed_fields"`
	Payload        map[string]any `json:"payload"`
	Timestamp      int64          `json:"ts"`
	SourceAnchorID string         `json:"source_anchor_id,omitempty"`
	TargetAnchorID string         `json:"target_anchor_id,omitempty"`
}

// ContainerMetric 混合实验容器采集数据（详 06.md §6.2 ContainerMetric / §十）。
type ContainerMetric struct {
	SourceContainer string `json:"source_container"`
	MetricKey       string `json:"metric_key"`
	Value           any    `json:"value"`
	Timestamp       int64  `json:"ts"`
	TargetPrimitive string `json:"target_primitive,omitempty"`
	TargetParam     string `json:"target_param,omitempty"`
}

// RenderEnvelope 场景输出给前端的标准渲染载荷（详 06.md §6.2）。
// 替代旧 {Nodes, Messages, Data} 模型；旧字段已废弃。
type RenderEnvelope struct {
	Primitives    []Primitive       `json:"primitives"`
	MicroSteps    []MicroStep       `json:"micro_steps,omitempty"`
	LinkTriggers  []LinkTrigger     `json:"link_triggers,omitempty"`
	ContainerData []ContainerMetric `json:"container_data,omitempty"`

	ChangedKeys    []string `json:"changed_keys,omitempty"`
	IsFullSnapshot bool     `json:"is_full_snapshot,omitempty"`

	// Data 仅承载侧栏指标 / 文字面板用数据（非原语参数）。
	Data map[string]any `json:"data,omitempty"`
}

// =====================================================================
// 场景内部状态（仅场景与 Core 之间序列化往返，前端不可见）
// =====================================================================

// SceneState 场景算法容器内部维护的状态。
// 对场景作者透明（使用 Data 字段自由组织业务字段），平台只关注序列化能力。
// 跨 Step 持有任何状态都必须放在本结构内由 Core 持有，场景容器不得在 goroutine / 全局变量持有。
type SceneState struct {
	SceneCode string         `json:"scene_code"`
	Tick      int64          `json:"tick"`
	Seed      int64          `json:"seed"`
	Phase     string         `json:"phase,omitempty"`
	StartedAt int64          `json:"started_at_ms,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// =====================================================================
// 交互定义（InteractionDefinition / ActionDef / FieldDef，详 06.md §6.3）
// =====================================================================

// InteractionDefinition 场景对外暴露的全部 ActionDef。
type InteractionDefinition struct {
	SceneCode     string      `json:"scene_code"`
	SchemaVersion string      `json:"schema_version"`
	Actions       []ActionDef `json:"actions"`
}

// ActionDef 单个学生 / 教师可执行交互动作。
// 协议层 LinkOwnerFields 必须 ⊆ 本场景在对应 LinkGroup 的 owner 字段集（详 §8.3）。
type ActionDef struct {
	ActionCode  string     `json:"action_code"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	Category    ActionCategory `json:"category"`
	Trigger     ActionTrigger  `json:"trigger"`
	Fields      []FieldDef `json:"fields"`

	Roles      []UserRole `json:"roles"`
	CooldownMs int        `json:"cooldown_ms,omitempty"`

	// LinkOwnerFields 仅作为该 action 的"owner 写入声明"显式列表（详 §8.3）。
	// Core LinkEngine 据此校验场景写入是否合法；非 owner 字段直接拒绝。
	LinkOwnerFields []string `json:"link_owner_fields,omitempty"`

	// WritesOwnedFields 列出本动作会修改的 owner 字段路径（必须 ⊆ Definition.OwnedFieldPaths）。
	// Core 按 DB sim_link_group_scenes 反查 field_path → [target_link_groups]，
	// 自动 fan-out 联动（详 §0.7.6 联动可扩展性）。
	// 本字段是"本动作触发哪些联动组"的唯一来源；旧的 TriggersLinkGroups 字段已弃用并删除。
	WritesOwnedFields []string `json:"writes_owned_fields,omitempty"`

	// Reversible 标识本动作是否可参与 process 模式回退栈（详 §0.7.7）。
	// container-channel 与有外部副作用的动作必须为 false；纯 SceneState 写入动作为 true。
	// 默认空指针处理逻辑由 Core 保守处理为 false（"不知道是否可逆 → 视为不可逆"）。
	Reversible bool `json:"reversible,omitempty"`

	// InterveneType 仅当 Roles 含 RoleTeacher 时必填，用于 Core 写审计日志（详 §10.9）。
	InterveneType InterveneType `json:"intervene_type,omitempty"`

	HybridChannel HybridChannel `json:"hybrid_channel,omitempty"`
	ContainerCmd  string        `json:"container_cmd,omitempty"`
}

// FieldDef 单个输入字段。
type FieldDef struct {
	Name        string    `json:"name"`
	Type        FieldType `json:"type"`
	Label       string    `json:"label"`
	Required    bool      `json:"required,omitempty"`
	Default     any       `json:"default,omitempty"`
	Min         any       `json:"min,omitempty"`
	Max         any       `json:"max,omitempty"`
	Step        any       `json:"step,omitempty"`
	Options     []any     `json:"options,omitempty"`
	OptionsFrom string    `json:"options_from,omitempty"`
}

// =====================================================================
// 场景定义（场景作者实现的钩子集合）
// =====================================================================

// Definition 描述一个场景的元信息与算法行为钩子。
// 场景代码通过 catalog 注册 Definition；launcher 据此启动 gRPC 服务。
type Definition struct {
	// 元信息（与 ScenarioMeta proto 对齐）
	Code            string
	Name            string
	Description     string
	Category        SceneCategory
	AlgorithmType   string
	Version         string
	TimeControlMode TimeControlMode
	DataSourceMode  DataSourceMode

	// 默认值
	DefaultParams func() map[string]any
	DefaultState  func() SceneState

	// 联动声明
	SupportedLinkGroups []string

	// OwnedFieldPaths 本场景能产出的全部 owner 字段路径（与 link_group 解耦）。
	// 例：["consensus.pbft.view", "consensus.pbft.committed", "consensus.pbft.byzantine_set"]。
	// Core 按 DB `sim_link_group_scenes` 反查路由表，新增联动组无需重编译镜像（详 §0.7.6）。
	OwnedFieldPaths []string

	// LinkGroupVersion 显式锁定本场景使用的 LinkGroup schema 版本（owner-based v0.5+）。
	// LinkGroup schema 不可演进；版本号由 docs SSOT 决定（详 §10.7）。
	LinkGroupVersion string

	// SupportsMultiActor 标识本场景是否支持多 actor 共享会话（详 §0.7.3）。
	// 多 actor 桶约定：SceneState.Data["actor_states"] = map[ActorID]map[string]any。
	// 单人场景置 false；ActorID == "" 时回退到 "default" 桶。
	SupportsMultiActor bool

	// ExtensionLevel 标识场景所属层级（L1 平台内置 / L2 平台扩展 / L3 教师自定义）。
	ExtensionLevel ExtensionLevel

	// 交互 schema 提供器
	Interaction func() InteractionDefinition

	// 算法钩子
	Init         func(state *SceneState, input InitInput) (RenderEnvelope, error)
	Step         func(state *SceneState, input StepInput) (StepOutput, error)
	HandleAction func(state *SceneState, input ActionInput) (ActionOutput, error)
}

// InitInput Init 调用输入。
type InitInput struct {
	SessionID   string
	InstanceID  string
	StudentID   string
	Seed        int64
	Params      map[string]any
	SharedState map[string]any
}

// StepInput Step 调用输入。
type StepInput struct {
	Tick                 int64
	SharedState          map[string]any
	IncomingLinkTriggers []LinkTrigger // Core fan-out 进来的联动事件，本 tick 内可视化

	// IncomingContainerMetrics 是 Core Collector 上一周期采集到的容器指标（详 §0.7.9 Core 1）。
	// 混合实验场景在本字段中读取真实容器数据驱动可视化（绑定到对应原语 Param）。
	IncomingContainerMetrics []ContainerMetric
}

// StepOutput Step 调用结果。
type StepOutput struct {
	Render          RenderEnvelope
	SharedStateDiff map[string]any // 仅本场景 owner 字段（详 §8.3）
}

// ActionInput HandleAction 调用输入。
type ActionInput struct {
	Tick        int64
	ActionCode  string
	Params      map[string]any
	ActorID     string
	UserRole    UserRole
	SharedState map[string]any
}

// ActionOutput HandleAction 调用结果。
type ActionOutput struct {
	Success         bool
	ErrorMessage    string
	Render          RenderEnvelope
	SharedStateDiff map[string]any // 仅本场景 owner 字段
}
