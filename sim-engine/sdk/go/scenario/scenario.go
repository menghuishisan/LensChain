// 模块：sim-engine/sdk/go/scenario
// 文件职责：sdk 对外稳定 API — 通过 type alias 真正重导出 framework 协议契约 +
//          定义 sdk 自己的 Scenario 接口与 gRPC 适配契约 + 提供校验函数。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §6.2 / §6.3 / §6.4。
//
// 与 framework 的关系：
//   - sdk 是教师自定义场景使用的稳定 API；framework 是协议契约 SSOT。
//   - 本文件用 `type X = framework.X` 把 framework 的协议类型与常量真正 alias 重导出，
//     底层 Go 类型只有一份，避免双套同步带来的协议腐化。
//   - sdk 自有的 Meta / InitRequest / InitResult / StepRequest / StepResult /
//     ActionRequest / ActionResult / Scenario 接口属于"sdk 与 gRPC server 适配的中间层"，
//     不入 framework（framework 是协议契约，不管教师如何写场景接口）。

package scenario

import (
	"context"
	"errors"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

// =====================================================================
// 协议类型重导出（真 type alias，与 framework 同一类型）
// =====================================================================

// 协议数据结构（详 06.md §6.2）。
type (
	Primitive       = fw.Primitive
	MicroStep       = fw.MicroStep
	LinkTrigger     = fw.LinkTrigger
	ContainerMetric = fw.ContainerMetric
	RenderEnvelope  = fw.RenderEnvelope
	SceneState      = fw.SceneState
)

// 交互定义（详 06.md §6.3）。
type (
	InteractionDefinition = fw.InteractionDefinition
	ActionDef             = fw.ActionDef
	FieldDef              = fw.FieldDef
)

// 场景定义钩子（详 06.md §6.4）。
type (
	Definition   = fw.Definition
	InitInput    = fw.InitInput
	StepInput    = fw.StepInput
	StepOutput   = fw.StepOutput
	ActionInput  = fw.ActionInput
	ActionOutput = fw.ActionOutput
)

// 几何工具（详 framework/layout.go）。
type Point = fw.Point

// =====================================================================
// 枚举常量重导出
// =====================================================================

// PrimitiveType 与 PrimitiveLayer。
type (
	PrimitiveType  = fw.PrimitiveType
	PrimitiveLayer = fw.PrimitiveLayer
)

// 47 原语 type 常量。
const (
	PrimGeometryNode         = fw.PrimGeometryNode
	PrimGeometryEdge         = fw.PrimGeometryEdge
	PrimGeometryBar          = fw.PrimGeometryBar
	PrimGeometryCurve        = fw.PrimGeometryCurve
	PrimGeometryPolygon      = fw.PrimGeometryPolygon
	PrimGeometryArea         = fw.PrimGeometryArea
	PrimGeometryGridCell     = fw.PrimGeometryGridCell
	PrimGeometryRing         = fw.PrimGeometryRing
	PrimEffectParticleStream = fw.PrimEffectParticleStream
	PrimEffectBurst          = fw.PrimEffectBurst
	PrimEffectPulse          = fw.PrimEffectPulse
	PrimEffectTrail          = fw.PrimEffectTrail
	PrimEffectGlow           = fw.PrimEffectGlow
	PrimEffectShake          = fw.PrimEffectShake
	PrimEffectShiftAnimation = fw.PrimEffectShiftAnimation
	PrimLayoutHorizontalLane = fw.PrimLayoutHorizontalLane
	PrimLayoutStack          = fw.PrimLayoutStack
	PrimLayoutRing           = fw.PrimLayoutRing
	PrimLayoutTree           = fw.PrimLayoutTree
	PrimLayoutGraph          = fw.PrimLayoutGraph
	PrimLayoutMatrix         = fw.PrimLayoutMatrix
	PrimDataLabel            = fw.PrimDataLabel
	PrimDataTooltip          = fw.PrimDataTooltip
	PrimDataAnnotation       = fw.PrimDataAnnotation
	PrimDataRegisterRow      = fw.PrimDataRegisterRow
	PrimDataMathPipeline     = fw.PrimDataMathPipeline
	PrimDataCodeBlock        = fw.PrimDataCodeBlock
	PrimDataMathFormula      = fw.PrimDataMathFormula

	PrimStatePhaseProgress       = fw.PrimStatePhaseProgress
	PrimStateProgressBar         = fw.PrimStateProgressBar
	PrimStateTargetZone          = fw.PrimStateTargetZone
	PrimStateLinkIndicator       = fw.PrimStateLinkIndicator
	PrimStateExternalEventMarker = fw.PrimStateExternalEventMarker
	PrimStateErrorOverlay        = fw.PrimStateErrorOverlay
	PrimStateVerifyPathHighlight = fw.PrimStateVerifyPathHighlight
	PrimStateRiskGauge           = fw.PrimStateRiskGauge

	PrimDomainVoteMatrix    = fw.PrimDomainVoteMatrix
	PrimDomainDualTrack     = fw.PrimDomainDualTrack
	PrimDomainTimeWheel     = fw.PrimDomainTimeWheel
	PrimDomainPieChart      = fw.PrimDomainPieChart
	PrimDomainSankeyFlow    = fw.PrimDomainSankeyFlow
	PrimDomainHeatMap       = fw.PrimDomainHeatMap
	PrimDomainMempoolSlot   = fw.PrimDomainMempoolSlot
	PrimDomainBridgeTrack   = fw.PrimDomainBridgeTrack
	PrimDomainCodeMarker    = fw.PrimDomainCodeMarker
	PrimDomainPartitionZone = fw.PrimDomainPartitionZone
	PrimDomainCurvePoint    = fw.PrimDomainCurvePoint
)

// 4 Layer 常量。
const (
	LayerBackground = fw.LayerBackground
	LayerContent    = fw.LayerContent
	LayerEffect     = fw.LayerEffect
	LayerOverlay    = fw.LayerOverlay
)

// 时间控制 / 数据源 / 类目 / 角色 / 触发 / 字段类型 / 混合通道 枚举。
type (
	TimeControlMode = fw.TimeControlMode
	DataSourceMode  = fw.DataSourceMode
	Category        = fw.SceneCategory
	ActionCategory  = fw.ActionCategory
	ActionTrigger   = fw.ActionTrigger
	FieldType       = fw.FieldType
	HybridChannel   = fw.HybridChannel
	UserRole        = fw.UserRole
)

const (
	TimeControlProcess    = fw.TimeControlProcess
	TimeControlReactive   = fw.TimeControlReactive
	TimeControlContinuous = fw.TimeControlContinuous

	DataSourceSimulation = fw.DataSourceSimulation
	DataSourceCollection = fw.DataSourceCollection
	DataSourceDual       = fw.DataSourceDual

	CategoryNodeNetwork    = fw.CategoryNodeNetwork
	CategoryConsensus      = fw.CategoryConsensus
	CategoryCryptography   = fw.CategoryCryptography
	CategoryDataStructure  = fw.CategoryDataStructure
	CategoryTransaction    = fw.CategoryTransaction
	CategorySmartContract  = fw.CategorySmartContract
	CategoryAttackSecurity = fw.CategoryAttackSecurity
	CategoryEconomic       = fw.CategoryEconomic
	CategoryGeneric        = fw.CategoryGeneric

	ActionParamTune    = fw.ActionParamTune
	ActionAttackInject = fw.ActionAttackInject
	ActionPrimary      = fw.ActionPrimary
	ActionObserve      = fw.ActionObserve

	TriggerSubmit    = fw.TriggerSubmit
	TriggerImmediate = fw.TriggerImmediate
	TriggerHold      = fw.TriggerHold

	FieldString      = fw.FieldString
	FieldNumber      = fw.FieldNumber
	FieldBoolean     = fw.FieldBoolean
	FieldSelect      = fw.FieldSelect
	FieldEnum        = fw.FieldEnum
	FieldRange       = fw.FieldRange
	FieldJSON        = fw.FieldJSON
	FieldMultiSelect = fw.FieldMultiSelect

	HybridChannelSim       = fw.HybridChannelSim
	HybridChannelContainer = fw.HybridChannelContainer

	RoleStudent = fw.RoleStudent
	RoleTeacher = fw.RoleTeacher
)

// v0.5 新增类型与常量重导出（AGENTS.md §0.7.1 C27 / C29 / C32 / C37）。
type (
	ExtensionLevel = fw.ExtensionLevel
	InterveneType  = fw.InterveneType
)

const (
	ExtensionL1 = fw.ExtensionL1
	ExtensionL2 = fw.ExtensionL2
	ExtensionL3 = fw.ExtensionL3

	InterveneHint     = fw.InterveneHint
	InterveneFault    = fw.InterveneFault
	InterveneAttack   = fw.InterveneAttack
	IntervenePhase    = fw.IntervenePhase
	InterveneTopology = fw.InterveneTopology
	InterveneState    = fw.InterveneState
	InterveneReset    = fw.InterveneReset
	InterveneEpoch    = fw.InterveneEpoch
	InterveneRevert   = fw.InterveneRevert
	InterveneFreeze   = fw.InterveneFreeze
)

// 角色集合常量（详 framework）。
var (
	RolesAll         = fw.RolesAll
	RolesStudentOnly = fw.RolesStudentOnly
	RolesTeacherOnly = fw.RolesTeacherOnly
)

// =====================================================================
// sdk 自有：Scenario 接口与 gRPC 适配中间类型
// =====================================================================

// Meta 是场景对外元信息（gRPC 适配层使用）。
//
// 教师写场景时通过 Scenario.Meta(ctx) 返回该结构；底层 sdk.Server 把它转换为 proto.ScenarioMeta。
type Meta struct {
	Code                    string
	Name                    string
	Description             string
	Category                Category
	AlgorithmType           string
	Version                 string
	TimeControlMode         TimeControlMode
	DataSourceMode          DataSourceMode
	DefaultParams           []byte // JSON
	DefaultState            []byte // JSON
	CustomRendererPackage   string // L3 npm 包名（仅 generic）
	SupportedLinkGroupCodes []string

	// v0.5 新增（详 AGENTS.md §0.7.1 C10 / C29 / C37）
	ExtensionLevel     ExtensionLevel
	LinkGroupVersion   string
	SupportsMultiActor bool
	OwnedFieldPaths    []string
}

// InitRequest 场景初始化 gRPC 请求。
type InitRequest struct {
	SessionID       string
	SceneCode       string
	InstanceID      string
	StudentID       string
	Seed            int64
	ParamsJSON      []byte
	SharedStateJSON []byte
}

// InitResult 场景初始化结果。
type InitResult struct {
	Tick                int64
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// StepRequest 单 tick 推进请求。
type StepRequest struct {
	SessionID                string
	SceneCode                string
	Tick                     int64
	SceneStateJSON           []byte
	SharedStateJSON          []byte
	IncomingLinkTriggers     []LinkTrigger
	IncomingContainerMetrics []ContainerMetric // v0.5 新增（详 §0.7.1 C1）
}

// StepResult 单 tick 推进结果。
type StepResult struct {
	Tick                int64
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// ActionRequest 交互请求。
type ActionRequest struct {
	SessionID       string
	SceneCode       string
	Tick            int64
	SceneStateJSON  []byte
	SharedStateJSON []byte
	ActionCode      string
	ParamsJSON      []byte
	ActorID         string
	UserRole        UserRole
}

// ActionResult 交互结果。
type ActionResult struct {
	Success             bool
	ErrorMessage        string
	Tick                int64
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// Scenario 是场景算法容器必须实现的最小接口。
//
// 教师自定义场景必须实现以下方法；返回 RenderEnvelope 须严格遵循协议字段（详 06.md §6.2）。
// 平台内部场景通常通过 framework.Definition + sdk.NewRuntimeScenario 间接实现 Scenario。
type Scenario interface {
	Meta(ctx context.Context) (Meta, error)
	InteractionSchema(ctx context.Context) (InteractionDefinition, error)
	Init(ctx context.Context, req InitRequest) (InitResult, error)
	Step(ctx context.Context, req StepRequest) (StepResult, error)
	HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error)
}

// =====================================================================
// 校验函数
// =====================================================================

// ValidateMeta 校验场景元信息是否满足平台上架要求。
func ValidateMeta(meta Meta) error {
	if strings.TrimSpace(meta.Code) == "" {
		return errors.New("场景编码不能为空")
	}
	if strings.TrimSpace(meta.Name) == "" {
		return errors.New("场景名称不能为空")
	}
	if !validCategory(meta.Category) {
		return errors.New("场景类目不合法")
	}
	if strings.TrimSpace(meta.AlgorithmType) == "" {
		return errors.New("算法类型不能为空")
	}
	if strings.TrimSpace(meta.Version) == "" {
		return errors.New("场景版本不能为空")
	}
	if !validTimeControlMode(meta.TimeControlMode) {
		return errors.New("时间控制模式不合法")
	}
	if !validDataSourceMode(meta.DataSourceMode) {
		return errors.New("数据源模式不合法")
	}
	return nil
}

// ValidateInteractionDefinition 校验交互定义。
func ValidateInteractionDefinition(def InteractionDefinition) error {
	if strings.TrimSpace(def.SceneCode) == "" {
		return errors.New("交互定义的场景编码不能为空")
	}
	for _, action := range def.Actions {
		if strings.TrimSpace(action.ActionCode) == "" {
			return errors.New("ActionDef.action_code 不能为空")
		}
		if strings.TrimSpace(action.Label) == "" {
			return errors.New("ActionDef.label 不能为空")
		}
		if !validActionCategory(action.Category) {
			return errors.New("ActionDef.category 不合法")
		}
		if !validActionTrigger(action.Trigger) {
			return errors.New("ActionDef.trigger 不合法")
		}
		for _, field := range action.Fields {
			if strings.TrimSpace(field.Name) == "" {
				return errors.New("FieldDef.name 不能为空")
			}
			if !validFieldType(field.Type) {
				return errors.New("FieldDef.type 不合法")
			}
		}
	}
	return nil
}

func validCategory(c Category) bool {
	switch c {
	case CategoryNodeNetwork, CategoryConsensus, CategoryCryptography, CategoryDataStructure,
		CategoryTransaction, CategorySmartContract, CategoryAttackSecurity, CategoryEconomic, CategoryGeneric:
		return true
	}
	return false
}

func validTimeControlMode(m TimeControlMode) bool {
	switch m {
	case TimeControlProcess, TimeControlReactive, TimeControlContinuous:
		return true
	}
	return false
}

func validDataSourceMode(m DataSourceMode) bool {
	switch m {
	case DataSourceSimulation, DataSourceCollection, DataSourceDual:
		return true
	}
	return false
}

func validActionCategory(c ActionCategory) bool {
	switch c {
	case ActionParamTune, ActionAttackInject, ActionPrimary, ActionObserve:
		return true
	}
	return false
}

func validActionTrigger(t ActionTrigger) bool {
	switch t {
	case TriggerSubmit, TriggerImmediate, TriggerHold:
		return true
	}
	return false
}

func validFieldType(t FieldType) bool {
	switch t {
	case FieldString, FieldNumber, FieldBoolean, FieldSelect,
		FieldEnum, FieldRange, FieldJSON, FieldMultiSelect:
		return true
	}
	return false
}
