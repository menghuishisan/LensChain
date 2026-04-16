package scenario

import (
	"context"
	"errors"
	"strings"
)

// Category 表示场景所属的前端领域渲染器。
// 除平台内置 8 个领域外，也允许教师声明新的自定义领域编码。
type Category string

const (
	// CategoryNodeNetwork 表示节点与网络领域。
	CategoryNodeNetwork Category = "node_network"
	// CategoryConsensus 表示共识过程领域。
	CategoryConsensus Category = "consensus"
	// CategoryCryptography 表示密码学运算领域。
	CategoryCryptography Category = "cryptography"
	// CategoryDataStructure 表示数据结构领域。
	CategoryDataStructure Category = "data_structure"
	// CategoryTransaction 表示交易生命周期领域。
	CategoryTransaction Category = "transaction"
	// CategorySmartContract 表示智能合约领域。
	CategorySmartContract Category = "smart_contract"
	// CategoryAttackSecurity 表示攻击与安全领域。
	CategoryAttackSecurity Category = "attack_security"
	// CategoryEconomic 表示经济模型领域。
	CategoryEconomic Category = "economic"
)

// TimeControlMode 表示场景时间控制模式。
type TimeControlMode string

const (
	// TimeControlModeProcess 表示带播放、单步、回退能力的过程化模式。
	TimeControlModeProcess TimeControlMode = "process"
	// TimeControlModeReactive 表示输入即响应的交互响应式模式。
	TimeControlModeReactive TimeControlMode = "reactive"
	// TimeControlModeContinuous 表示持续演化观察的连续运行模式。
	TimeControlModeContinuous TimeControlMode = "continuous"
)

// DataSourceMode 表示场景数据源模式。
type DataSourceMode string

const (
	// DataSourceModeSimulation 表示状态完全来自仿真算法。
	DataSourceModeSimulation DataSourceMode = "simulation"
	// DataSourceModeCollection 表示状态完全来自外部采集。
	DataSourceModeCollection DataSourceMode = "collection"
	// DataSourceModeDual 表示同时支持仿真与采集两种输入。
	DataSourceModeDual DataSourceMode = "dual"
)

// Meta 是场景算法容器上报给平台的元信息。
type Meta struct {
	Code                    string
	Name                    string
	Category                Category
	AlgorithmType           string
	Description             string
	Version                 string
	TimeControlMode         TimeControlMode
	DataSourceMode          DataSourceMode
	DefaultParams           []byte
	DefaultState            []byte
	SupportedLinkGroupCodes []string
}

// InitRequest 是场景初始化请求。
type InitRequest struct {
	SceneCode        string
	InstanceID       string
	StudentID        string
	Seed             int64
	SessionID        string
	ParamsJSON       []byte
	InitialStateJSON []byte
	SharedStateJSON  []byte
}

// State 是场景完整状态和可渲染状态。
type State struct {
	Tick            int64
	StateJSON       []byte
	RenderStateJSON []byte
	SharedStateJSON []byte
}

// StepRequest 是仿真时钟步推进请求。
type StepRequest struct {
	SessionID       string
	SceneCode       string
	Tick            int64
	StateJSON       []byte
	SharedStateJSON []byte
}

// StepResult 是仿真时钟步推进结果。
type StepResult struct {
	Tick                int64
	StateJSON           []byte
	RenderStateJSON     []byte
	Events              []Event
	SharedStateDiffJSON []byte
}

// ActionRequest 是场景专属交互请求。
type ActionRequest struct {
	SessionID       string
	SceneCode       string
	ActionCode      string
	ParamsJSON      []byte
	StateJSON       []byte
	SharedStateJSON []byte
	Tick            int64
	ActorID         string
	RoleKey         string
}

// ActionResult 是场景专属交互结果。
type ActionResult struct {
	Success         bool
	ErrorMessage    string
	StateJSON       []byte
	RenderStateJSON []byte
	Events          []Event
	SharedStateDiff []byte
}

// RenderStateRequest 是场景按当前共享状态重建渲染态时的输入。
type RenderStateRequest struct {
	SessionID       string
	SceneCode       string
	Tick            int64
	StateJSON       []byte
	SharedStateJSON []byte
}

// Event 表示场景算法产生的过程事件。
type Event struct {
	EventID     string
	EventType   string
	SceneCode   string
	Tick        int64
	TimestampMS int64
	PayloadJSON []byte
}

// InteractionSchema 是前端动态生成场景专属操作面板的依据。
type InteractionSchema struct {
	SceneCode string
	Actions   []InteractionAction
}

// InteractionAction 表示一个可执行的场景操作。
type InteractionAction struct {
	ActionCode   string
	Label        string
	Description  string
	Trigger      InteractionTrigger
	Fields       []InteractionField
	UISchemaJSON []byte
}

// InteractionField 表示操作面板中的一个输入字段。
type InteractionField struct {
	Key            string
	Label          string
	Type           InteractionFieldType
	Required       bool
	DefaultValue   string
	Options        []InteractionOption
	ValidationJSON []byte
}

// InteractionOption 表示选择类字段的候选项。
type InteractionOption struct {
	Value string
	Label string
}

// InteractionFieldType 定义动态交互面板字段类型。
type InteractionFieldType string

const (
	// InteractionFieldTypeString 表示字符串输入字段。
	InteractionFieldTypeString InteractionFieldType = "string"
	// InteractionFieldTypeNumber 表示数值输入字段。
	InteractionFieldTypeNumber InteractionFieldType = "number"
	// InteractionFieldTypeBoolean 表示布尔开关字段。
	InteractionFieldTypeBoolean InteractionFieldType = "boolean"
	// InteractionFieldTypeSelect 表示单选下拉字段。
	InteractionFieldTypeSelect InteractionFieldType = "select"
	// InteractionFieldTypeNodeRef 表示节点引用字段。
	InteractionFieldTypeNodeRef InteractionFieldType = "node_ref"
	// InteractionFieldTypeRange 表示范围滑块字段。
	InteractionFieldTypeRange InteractionFieldType = "range"
	// InteractionFieldTypeJSON 表示原始 JSON 输入字段。
	InteractionFieldTypeJSON InteractionFieldType = "json"
)

// InteractionTrigger 定义交互操作的触发方式。
type InteractionTrigger string

const (
	// InteractionTriggerClick 表示点击触发。
	InteractionTriggerClick InteractionTrigger = "click"
	// InteractionTriggerFormSubmit 表示表单提交触发。
	InteractionTriggerFormSubmit InteractionTrigger = "form_submit"
	// InteractionTriggerDrag 表示拖拽触发。
	InteractionTriggerDrag InteractionTrigger = "drag"
	// InteractionTriggerCanvasSelect 表示画布框选触发。
	InteractionTriggerCanvasSelect InteractionTrigger = "canvas_select"
)

// InteractiveScenario 表示支持场景专属交互的扩展接口。
type InteractiveScenario interface {
	HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error)
}

// InteractionSchemaProvider 表示支持动态交互面板定义的扩展接口。
type InteractionSchemaProvider interface {
	InteractionSchema(ctx context.Context) (InteractionSchema, error)
}

// RenderStateProvider 表示场景支持基于共享状态即时重建渲染态。
type RenderStateProvider interface {
	RenderState(ctx context.Context, req RenderStateRequest) (State, error)
}

// Scenario 是教师自定义场景需要实现的最小接口。
type Scenario interface {
	Meta(ctx context.Context) (Meta, error)
	Init(ctx context.Context, req InitRequest) (State, error)
	Step(ctx context.Context, req StepRequest) (StepResult, error)
}

// FuncScenario 用函数快速组装一个场景，适合示例和轻量场景。
type FuncScenario struct {
	MetaValue Meta
	InitFunc  func(context.Context, InitRequest) (State, error)
	StepFunc  func(context.Context, StepRequest) (StepResult, error)
}

// Meta 返回场景元信息。
func (s FuncScenario) Meta(context.Context) (Meta, error) {
	if err := ValidateMeta(s.MetaValue); err != nil {
		return Meta{}, err
	}
	return s.MetaValue, nil
}

// Init 调用函数式初始化逻辑。
func (s FuncScenario) Init(ctx context.Context, req InitRequest) (State, error) {
	if s.InitFunc == nil {
		return State{}, errors.New("必须提供初始化函数")
	}
	return s.InitFunc(ctx, req)
}

// Step 调用函数式仿真时钟步推进逻辑。
func (s FuncScenario) Step(ctx context.Context, req StepRequest) (StepResult, error) {
	if s.StepFunc == nil {
		return StepResult{}, errors.New("必须提供推进函数")
	}
	return s.StepFunc(ctx, req)
}

// ValidateMeta 校验场景元信息是否满足平台上架要求。
func ValidateMeta(meta Meta) error {
	if strings.TrimSpace(meta.Code) == "" {
		return errors.New("场景编码不能为空")
	}
	if strings.TrimSpace(meta.Name) == "" {
		return errors.New("场景名称不能为空")
	}
	if !validCategory(meta.Category) {
		return errors.New("场景领域类型不合法")
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

// validCategory 校验领域类型是否合法。
// 文档允许教师为全新领域同时上传自定义渲染器，因此这里不能把领域编码锁死为内置枚举。
func validCategory(category Category) bool {
	if strings.TrimSpace(string(category)) == "" {
		return false
	}
	switch category {
	case CategoryNodeNetwork, CategoryConsensus, CategoryCryptography, CategoryDataStructure,
		CategoryTransaction, CategorySmartContract, CategoryAttackSecurity, CategoryEconomic:
		return true
	default:
		return true
	}
}

// validTimeControlMode 校验时间控制模式是否合法。
func validTimeControlMode(mode TimeControlMode) bool {
	switch mode {
	case TimeControlModeProcess, TimeControlModeReactive, TimeControlModeContinuous:
		return true
	default:
		return false
	}
}

// validDataSourceMode 校验数据源模式是否合法。
func validDataSourceMode(mode DataSourceMode) bool {
	switch mode {
	case DataSourceModeSimulation, DataSourceModeCollection, DataSourceModeDual:
		return true
	default:
		return false
	}
}

// ValidateInteractionSchema 校验动态交互面板定义。
func ValidateInteractionSchema(schema InteractionSchema) error {
	if strings.TrimSpace(schema.SceneCode) == "" {
		return errors.New("交互面板的场景编码不能为空")
	}
	if len(schema.Actions) == 0 {
		return errors.New("交互面板至少要包含一个操作")
	}
	for _, action := range schema.Actions {
		if strings.TrimSpace(action.ActionCode) == "" {
			return errors.New("交互操作编码不能为空")
		}
		if strings.TrimSpace(action.Label) == "" {
			return errors.New("交互操作标题不能为空")
		}
		if !validInteractionTrigger(action.Trigger) {
			return errors.New("交互触发方式不合法")
		}
		for _, field := range action.Fields {
			if strings.TrimSpace(field.Key) == "" {
				return errors.New("交互字段键不能为空")
			}
			if strings.TrimSpace(field.Label) == "" {
				return errors.New("交互字段标题不能为空")
			}
			if !validInteractionFieldType(field.Type) {
				return errors.New("交互字段类型不合法")
			}
		}
	}
	return nil
}

// validInteractionFieldType 校验交互字段类型是否合法。
func validInteractionFieldType(fieldType InteractionFieldType) bool {
	switch fieldType {
	case InteractionFieldTypeString, InteractionFieldTypeNumber, InteractionFieldTypeBoolean,
		InteractionFieldTypeSelect, InteractionFieldTypeNodeRef, InteractionFieldTypeRange, InteractionFieldTypeJSON:
		return true
	default:
		return false
	}
}

// validInteractionTrigger 校验交互触发方式是否合法。
func validInteractionTrigger(trigger InteractionTrigger) bool {
	switch trigger {
	case InteractionTriggerClick, InteractionTriggerFormSubmit, InteractionTriggerDrag, InteractionTriggerCanvasSelect:
		return true
	default:
		return false
	}
}
