package contractstatemachine

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造智能合约状态机场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "contract-state-machine",
		Title:        "智能合约状态机",
		Phase:        "状态触发",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1000,
		TotalTicks:   10,
		Stages:       []string{"状态触发", "状态迁移", "存储更新"},
		Nodes: []framework.Node{
			{ID: "created", Label: "Created", Status: "active", Role: "state", X: 120, Y: 200},
			{ID: "active", Label: "Active", Status: "normal", Role: "state", X: 300, Y: 120},
			{ID: "paused", Label: "Paused", Status: "normal", Role: "state", X: 300, Y: 280},
			{ID: "closed", Label: "Closed", Status: "normal", Role: "state", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化当前状态、允许事件和存储槽。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := machineModel{
		CurrentState:  "created",
		LastEvent:     "deploy",
		StorageSlots:  map[string]string{"owner": "teacher", "status": "created", "counter": "0", "guard": "disabled"},
		AllowedEvents: []string{"activate", "pause", "close"},
		GuardEnabled:  false,
		Version:       0,
	}
	applySharedMachineState(&model, input.SharedState)
	return rebuildState(state, model, "状态触发")
}

// Step 推进状态触发、状态迁移和存储更新。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedMachineState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "状态触发"))
	switch phase {
	case "状态迁移":
		transitionState(&model, model.LastEvent)
		model.Version++
	case "存储更新":
		model.StorageSlots["status"] = model.CurrentState
		model.StorageSlots["counter"] = fmt.Sprintf("%d", int(framework.NumberValue(model.StorageSlots["counter"], 0))+1)
		if model.GuardEnabled {
			model.StorageSlots["guard"] = "enabled"
		} else {
			model.StorageSlots["guard"] = "disabled"
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("合约状态机进入%s阶段。", phase), toneByState(model.CurrentState))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"status":     model.CurrentState,
				"last_event": model.LastEvent,
				"protected":  model.GuardEnabled,
				"storage": map[string]any{
					"owner":   model.StorageSlots["owner"],
					"status":  model.StorageSlots["status"],
					"counter": model.StorageSlots["counter"],
					"guard":   model.StorageSlots["guard"],
					"version": model.Version,
				},
			},
		},
	}, nil
}

// HandleAction 处理外部事件并触发新的状态迁移。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.LastEvent = framework.StringValue(input.Params["event"], "activate")
	if model.LastEvent == "pause" {
		model.GuardEnabled = true
	}
	if model.LastEvent == "activate" {
		model.GuardEnabled = false
	}
	transitionState(&model, model.LastEvent)
	model.Version++
	model.StorageSlots["status"] = model.CurrentState
	if model.GuardEnabled {
		model.StorageSlots["guard"] = "enabled"
	}
	if err := rebuildState(state, model, "状态迁移"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "触发事件", fmt.Sprintf("已触发事件 %s。", model.LastEvent), toneByState(model.CurrentState))
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"status":     model.CurrentState,
				"last_event": model.LastEvent,
				"protected":  model.GuardEnabled,
				"storage": map[string]any{
					"owner":   model.StorageSlots["owner"],
					"status":  model.StorageSlots["status"],
					"counter": model.StorageSlots["counter"],
					"guard":   model.StorageSlots["guard"],
					"version": model.Version,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出状态节点、事件流和存储槽。
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

// SyncSharedState 在合约安全组共享状态变化后重建状态机场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedMachineState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// machineModel 保存状态机当前状态、触发事件和存储槽。
type machineModel struct {
	CurrentState  string            `json:"current_state"`
	LastEvent     string            `json:"last_event"`
	StorageSlots  map[string]string `json:"storage_slots"`
	AllowedEvents []string          `json:"allowed_events"`
	GuardEnabled  bool              `json:"guard_enabled"`
	Version       int               `json:"version"`
}

// rebuildState 将状态机模型映射为状态节点、消息和指标。
func rebuildState(state *framework.SceneState, model machineModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		if state.Nodes[index].ID == model.CurrentState {
			state.Nodes[index].Status = "success"
		}
		if phase == "状态触发" && state.Nodes[index].ID == model.CurrentState {
			state.Nodes[index].Status = "active"
		}
	}
	state.Messages = []framework.Message{
		{ID: "event-flow", Label: model.LastEvent, Kind: "call", Status: phase, SourceID: model.CurrentState, TargetID: nextStatePreview(model.CurrentState, model.LastEvent)},
	}
	state.Metrics = []framework.Metric{
		{Key: "state", Label: "当前状态", Value: model.CurrentState, Tone: toneByState(model.CurrentState)},
		{Key: "event", Label: "最后事件", Value: model.LastEvent, Tone: "info"},
		{Key: "version", Label: "状态版本", Value: fmt.Sprintf("%d", model.Version), Tone: "warning"},
		{Key: "counter", Label: "计数器", Value: model.StorageSlots["counter"], Tone: "warning"},
		{Key: "guard", Label: "保护开关", Value: model.StorageSlots["guard"], Tone: toneByGuard(model.GuardEnabled)},
		{Key: "owner", Label: "Owner", Value: model.StorageSlots["owner"], Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "状态槽", Value: model.StorageSlots["status"]},
		{Label: "可触发事件", Value: strings.Join(model.AllowedEvents, ", ")},
		{Label: "保护状态", Value: model.StorageSlots["guard"]},
		{Label: "阶段", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":             phase,
		"contract_state_machine": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟合约状态机中的事件触发、状态迁移和存储槽更新。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复状态机模型。
func decodeModel(state *framework.SceneState) machineModel {
	entry, ok := state.Data["contract_state_machine"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["contract_state_machine"].(machineModel); ok {
			return typed
		}
		return machineModel{
			CurrentState:  "created",
			LastEvent:     "deploy",
			StorageSlots:  map[string]string{"owner": "teacher", "status": "created", "counter": "0", "guard": "disabled"},
			AllowedEvents: []string{"activate", "pause", "close"},
			GuardEnabled:  false,
			Version:       0,
		}
	}
	return machineModel{
		CurrentState:  framework.StringValue(entry["current_state"], "created"),
		LastEvent:     framework.StringValue(entry["last_event"], "deploy"),
		StorageSlots:  decodeStringMap(entry["storage_slots"]),
		AllowedEvents: framework.ToStringSlice(entry["allowed_events"]),
		GuardEnabled:  framework.BoolValue(entry["guard_enabled"], framework.StringValue(entry["guard"], "disabled") == "enabled"),
		Version:       int(framework.NumberValue(entry["version"], 0)),
	}
}

// applySharedMachineState 将合约安全组中的共享状态映射回状态机模型。
func applySharedMachineState(model *machineModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		status := framework.StringValue(contractState["status"], model.CurrentState)
		if status != "" {
			model.CurrentState = status
			model.StorageSlots["status"] = status
		}
		model.LastEvent = framework.StringValue(contractState["last_event"], model.LastEvent)
		if storage, ok := contractState["storage"].(map[string]any); ok {
			if version, ok := storage["version"]; ok {
				model.Version = int(framework.NumberValue(version, float64(model.Version)))
			}
		}
		if framework.BoolValue(contractState["protected"], false) {
			model.GuardEnabled = true
			model.StorageSlots["guard"] = "enabled"
		} else if _, ok := contractState["protected"]; ok {
			model.GuardEnabled = false
			model.StorageSlots["guard"] = "disabled"
		}
	}
	if callState, ok := sharedState["call_stack"].(map[string]any); ok {
		opcode := framework.StringValue(callState["opcode"], "")
		if opcode != "" {
			model.LastEvent = strings.ToLower(opcode)
		}
	}
}

// transitionState 根据事件执行状态迁移。
func transitionState(model *machineModel, event string) {
	switch model.CurrentState {
	case "created":
		if event == "activate" {
			model.CurrentState = "active"
		}
	case "active":
		if event == "pause" {
			model.CurrentState = "paused"
		}
		if event == "close" {
			model.CurrentState = "closed"
		}
	case "paused":
		if event == "activate" {
			model.CurrentState = "active"
		}
		if event == "close" {
			model.CurrentState = "closed"
		}
	}
}

// nextStatePreview 计算事件触发后的预期状态。
func nextStatePreview(current string, event string) string {
	model := machineModel{CurrentState: current}
	transitionState(&model, event)
	return model.CurrentState
}

// nextPhase 返回状态机流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "状态触发":
		return "状态迁移"
	case "状态迁移":
		return "存储更新"
	default:
		return "状态触发"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "状态触发":
		return 0
	case "状态迁移":
		return 1
	case "存储更新":
		return 2
	default:
		return 0
	}
}

// toneByState 返回合约状态对应的色调。
func toneByState(state string) string {
	switch state {
	case "active":
		return "success"
	case "paused":
		return "warning"
	case "closed":
		return "info"
	default:
		return "info"
	}
}

// toneByGuard 返回保护开关对应的色调。
func toneByGuard(enabled bool) string {
	if enabled {
		return "success"
	}
	return "info"
}

// decodeStringMap 恢复字符串映射。
func decodeStringMap(value any) map[string]string {
	entry, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]string); ok {
			return typed
		}
		return map[string]string{"owner": "teacher", "status": "created", "counter": "0"}
	}
	result := make(map[string]string, len(entry))
	for key, raw := range entry {
		result[key] = framework.StringValue(raw, "")
	}
	return result
}


// scalarText 将共享状态中的标量值统一转换为字符串，兼容数字与布尔输入。
func scalarText(value any, fallback string) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fallback
	}
}
