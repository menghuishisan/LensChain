package integeroverflow

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// uint8Max 表示场景里演示溢出的目标上限。
	uint8Max = 255
)

// DefaultState 构造整数溢出攻击场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "integer-overflow",
		Title:        "整数溢出攻击",
		Phase:        "逼近上限",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   10,
		Stages:       []string{"逼近上限", "发生回绕", "保护拦截"},
		Nodes: []framework.Node{
			{ID: "unsafe", Label: "UnsafeUint8", Status: "active", Role: "value", X: 120, Y: 210},
			{ID: "boundary", Label: "Boundary", Status: "normal", Role: "value", X: 320, Y: 90},
			{ID: "wrapped", Label: "WrappedValue", Status: "normal", Role: "value", X: 320, Y: 330},
			{ID: "safe", Label: "SafeMath", Status: "normal", Role: "value", X: 540, Y: 210},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化不安全加法、回绕结果和保护状态。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := overflowModel{
		BaseValue:      250,
		Delta:          1,
		UnsafeResult:   251,
		SafeResult:     251,
		Wrapped:        false,
		SafeRejected:   false,
		CriticalPoint:  uint8Max,
		RemainingSpace: 4,
	}
	applySharedOverflowState(&model, input.SharedState)
	return rebuildState(state, model, "逼近上限")
}

// Step 推进逼近上限、发生回绕和 SafeMath 拦截过程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedOverflowState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "逼近上限"))
	switch phase {
	case "发生回绕":
		model.BaseValue = 254
		model.Delta = 3
		model.UnsafeResult = (model.BaseValue + model.Delta) % (uint8Max + 1)
		model.Wrapped = true
		model.RemainingSpace = 0
	case "保护拦截":
		model.SafeRejected = model.BaseValue+model.Delta > model.CriticalPoint
		if model.SafeRejected {
			model.SafeResult = model.BaseValue
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("整数运算进入%s阶段。", phase), toneByOverflow(model, phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"unsafe_value": model.UnsafeResult,
				"safe_value":   model.SafeResult,
				"wrapped":      model.Wrapped,
				"protected":    model.SafeRejected,
				"status":       overflowContractStatus(model),
				"last_event":   overflowEventName(model),
				"storage": map[string]any{
					"unsafe_value": model.UnsafeResult,
					"safe_value":   model.SafeResult,
					"delta":        model.Delta,
				},
			},
		},
	}, nil
}

// HandleAction 对目标值执行加法，演示不安全整数与 SafeMath 的不同结果。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Delta = int(framework.NumberValue(input.Params["delta"], 1))
	model.UnsafeResult = (model.BaseValue + model.Delta) % (uint8Max + 1)
	model.Wrapped = model.BaseValue+model.Delta > model.CriticalPoint
	model.SafeRejected = model.Wrapped
	if model.SafeRejected {
		model.SafeResult = model.BaseValue
	} else {
		model.SafeResult = model.BaseValue + model.Delta
	}
	model.RemainingSpace = maxInt(model.CriticalPoint-model.BaseValue, 0)
	phase := "逼近上限"
	if model.Wrapped {
		phase = "发生回绕"
	}
	if model.SafeRejected {
		phase = "保护拦截"
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "增加数值", fmt.Sprintf("执行 +%d 运算。", model.Delta), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"unsafe_value": model.UnsafeResult,
				"safe_value":   model.SafeResult,
				"wrapped":      model.Wrapped,
				"delta":        model.Delta,
				"protected":    model.SafeRejected,
				"status":       overflowContractStatus(model),
				"last_event":   overflowEventName(model),
				"storage": map[string]any{
					"unsafe_value": model.UnsafeResult,
					"safe_value":   model.SafeResult,
					"delta":        model.Delta,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出不安全结果、临界点和 SafeMath 对比。
func BuildRenderState(state framework.SceneState) framework.RenderEnvelope {
	return framework.RenderEnvelope{
		Nodes:       state.Nodes,
		Messages:    state.Messages,
		Stages:      state.Stages,
		ChangedKeys: state.ChangedKeys,
		Phase:       state.Phase,
		PhaseIndex:  state.PhaseIndex,
		Progress:    state.Progress,
		Data:        framework.CloneMap(state.Data),
		Extra:       framework.CloneMap(state.Extra),
	}
}

// SyncSharedState 在共享合约状态变化后重建整数溢出场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedOverflowState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// overflowModel 保存危险整数运算与保护逻辑对比。
type overflowModel struct {
	BaseValue      int  `json:"base_value"`
	Delta          int  `json:"delta"`
	UnsafeResult   int  `json:"unsafe_result"`
	SafeResult     int  `json:"safe_result"`
	Wrapped        bool `json:"wrapped"`
	SafeRejected   bool `json:"safe_rejected"`
	CriticalPoint  int  `json:"critical_point"`
	RemainingSpace int  `json:"remaining_space"`
}

// rebuildState 将整数模型映射为可视化节点、消息和指标。
func rebuildState(state *framework.SceneState, model overflowModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = unsafeStatus(model, phase)
	state.Nodes[0].Load = float64(model.UnsafeResult)
	state.Nodes[1].Status = "warning"
	state.Nodes[1].Load = float64(model.CriticalPoint)
	state.Nodes[2].Status = wrappedStatus(model)
	state.Nodes[2].Load = float64(model.UnsafeResult)
	state.Nodes[3].Status = safeStatus(model)
	state.Nodes[3].Load = float64(model.SafeResult)
	state.Messages = []framework.Message{
		{ID: "overflow-unsafe", Label: fmt.Sprintf("%d + %d", model.BaseValue, model.Delta), Kind: "attack", Status: phase, SourceID: "unsafe", TargetID: "wrapped"},
		{ID: "overflow-safe", Label: safeMessage(model), Kind: "attack", Status: phase, SourceID: "boundary", TargetID: "safe"},
	}
	state.Metrics = []framework.Metric{
		{Key: "base", Label: "当前值", Value: fmt.Sprintf("%d", model.BaseValue), Tone: "info"},
		{Key: "unsafe", Label: "不安全结果", Value: fmt.Sprintf("%d", model.UnsafeResult), Tone: toneByOverflow(model, phase)},
		{Key: "safe", Label: "SafeMath 结果", Value: fmt.Sprintf("%d", model.SafeResult), Tone: safeTone(model)},
		{Key: "critical", Label: "临界点", Value: fmt.Sprintf("%d", model.CriticalPoint), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "是否回绕", Value: framework.BoolText(model.Wrapped)},
		{Label: "是否拦截", Value: framework.BoolText(model.SafeRejected)},
		{Label: "剩余空间", Value: fmt.Sprintf("%d", model.RemainingSpace)},
	}
	state.Data = map[string]any{
		"phase_name":       phase,
		"integer_overflow": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 uint8 上界附近的整数回绕，以及 SafeMath 风格的边界拦截。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复整数模型。
func decodeModel(state *framework.SceneState) overflowModel {
	entry, ok := state.Data["integer_overflow"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["integer_overflow"].(overflowModel); ok {
			return typed
		}
		return overflowModel{
			BaseValue:      250,
			Delta:          1,
			UnsafeResult:   251,
			SafeResult:     251,
			CriticalPoint:  uint8Max,
			RemainingSpace: 4,
		}
	}
	return overflowModel{
		BaseValue:      int(framework.NumberValue(entry["base_value"], 250)),
		Delta:          int(framework.NumberValue(entry["delta"], 1)),
		UnsafeResult:   int(framework.NumberValue(entry["unsafe_result"], 251)),
		SafeResult:     int(framework.NumberValue(entry["safe_result"], 251)),
		Wrapped:        framework.BoolValue(entry["wrapped"], false),
		SafeRejected:   framework.BoolValue(entry["safe_rejected"], false),
		CriticalPoint:  int(framework.NumberValue(entry["critical_point"], uint8Max)),
		RemainingSpace: int(framework.NumberValue(entry["remaining_space"], 4)),
	}
}

// applySharedOverflowState 将合约安全组中的共享状态映射回整数溢出模型。
func applySharedOverflowState(model *overflowModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		model.UnsafeResult = int(framework.NumberValue(contractState["unsafe_value"], float64(model.UnsafeResult)))
		model.SafeResult = int(framework.NumberValue(contractState["safe_value"], float64(model.SafeResult)))
		model.Wrapped = framework.BoolValue(contractState["wrapped"], model.Wrapped)
		model.Delta = int(framework.NumberValue(contractState["delta"], float64(model.Delta)))
		if framework.BoolValue(contractState["protected"], false) {
			model.SafeRejected = true
		}
	}
}

// nextPhase 返回整数溢出场景的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "逼近上限":
		return "发生回绕"
	case "发生回绕":
		return "保护拦截"
	default:
		return "逼近上限"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "逼近上限":
		return 0
	case "发生回绕":
		return 1
	case "保护拦截":
		return 2
	default:
		return 0
	}
}

// unsafeStatus 返回不安全整数节点状态。
func unsafeStatus(model overflowModel, phase string) string {
	if phase == "发生回绕" || model.Wrapped {
		return "warning"
	}
	return "active"
}

// wrappedStatus 返回回绕结果节点状态。
func wrappedStatus(model overflowModel) string {
	if model.Wrapped {
		return "warning"
	}
	return "normal"
}

// safeStatus 返回 SafeMath 节点状态。
func safeStatus(model overflowModel) string {
	if model.SafeRejected {
		return "success"
	}
	return "normal"
}

// safeTone 返回 SafeMath 指标色调。
func safeTone(model overflowModel) string {
	if model.SafeRejected {
		return "success"
	}
	return "info"
}

// toneByOverflow 返回回绕阶段对应的色调。
func toneByOverflow(model overflowModel, phase string) string {
	if phase == "保护拦截" && model.SafeRejected {
		return "success"
	}
	if model.Wrapped || phase == "发生回绕" {
		return "warning"
	}
	return "info"
}

// safeMessage 返回 SafeMath 面板上展示的处理文案。
func safeMessage(model overflowModel) string {
	if model.SafeRejected {
		return "revert"
	}
	return fmt.Sprintf("ok:%d", model.SafeResult)
}

// maxInt 返回较大的整数值。
func maxInt(value int, fallback int) int {
	if value > fallback {
		return value
	}
	return fallback
}

// overflowContractStatus 返回整数溢出对合约状态机暴露的状态。
func overflowContractStatus(model overflowModel) string {
	if model.SafeRejected {
		return "paused"
	}
	if model.Wrapped {
		return "active"
	}
	return "created"
}

// overflowEventName 返回整数运算对联动场景暴露的最近事件。
func overflowEventName(model overflowModel) string {
	if model.SafeRejected {
		return "pause"
	}
	if model.Wrapped {
		return "activate"
	}
	return "deploy"
}
