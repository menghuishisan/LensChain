package contractinteraction

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造合约间调用场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "contract-interaction",
		Title:        "合约间调用",
		Phase:        "进入调用",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1300,
		TotalTicks:   14,
		Stages:       []string{"进入调用", "上下文切换", "返回值传播"},
		Nodes: []framework.Node{
			{ID: "caller", Label: "Caller", Status: "active", Role: "contract", X: 100, Y: 200},
			{ID: "library", Label: "Library", Status: "normal", Role: "contract", X: 280, Y: 100},
			{ID: "vault", Label: "Vault", Status: "normal", Role: "contract", X: 280, Y: 300},
			{ID: "receiver", Label: "Receiver", Status: "normal", Role: "contract", X: 500, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化调用类型、调用栈和返回值。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := interactionModel{
		CallType:      "call",
		CallStack:     []string{"Caller"},
		CurrentTarget: "Vault",
		ReturnValue:   "pending",
		ContextOwner:  "Caller",
		StorageWrite:  "Vault.balance",
		GasUsed:       700,
	}
	return rebuildState(state, model, "进入调用")
}

// Step 推进进入调用、上下文切换和返回值传播。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "进入调用"))
	switch phase {
	case "上下文切换":
		model.CallStack = append(model.CallStack, model.CurrentTarget)
		model.ContextOwner = resolveContextOwner(model.CallType, model.CurrentTarget)
		model.GasUsed += gasCostByCallType(model.CallType)
	case "返回值传播":
		model.ReturnValue = model.CallType + "-ok"
		model.StorageWrite = resolveStorageWrite(model.CallType, model.CurrentTarget)
		if len(model.CallStack) > 1 {
			model.CallStack = model.CallStack[:len(model.CallStack)-1]
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("合约调用进入%s阶段。", phase), toneByCallType(model.CallType))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"call_stack": map[string]any{
				"type":   model.CallType,
				"stack":  model.CallStack,
				"target": model.CurrentTarget,
				"opcode": strings.ToUpper(model.CallType),
			},
			"contract_state": map[string]any{
				"return_value": model.ReturnValue,
				"storage": map[string]any{
					"context_owner": model.ContextOwner,
					"storage_write": model.StorageWrite,
					"gas_used":      model.GasUsed,
				},
			},
		},
	}, nil
}

// HandleAction 切换为 delegatecall 或其他调用模式。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	target := framework.StringValue(input.Params["resource_id"], "Library")
	model.CurrentTarget = target
	if strings.EqualFold(target, "Library") {
		model.CallType = "delegatecall"
	} else {
		model.CallType = "call"
	}
	model.CallStack = []string{"Caller", target}
	model.ContextOwner = resolveContextOwner(model.CallType, target)
	model.StorageWrite = resolveStorageWrite(model.CallType, target)
	model.GasUsed += gasCostByCallType(model.CallType)
	if err := rebuildState(state, model, "上下文切换"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "触发调用", fmt.Sprintf("已触发 %s 到 %s。", model.CallType, target), toneByCallType(model.CallType))
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"call_stack": map[string]any{
				"type":   model.CallType,
				"stack":  model.CallStack,
				"target": model.CurrentTarget,
				"opcode": strings.ToUpper(model.CallType),
			},
			"contract_state": map[string]any{
				"storage": map[string]any{
					"context_owner": model.ContextOwner,
					"storage_write": model.StorageWrite,
					"gas_used":      model.GasUsed,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出调用栈、上下文切换和返回值。
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

// SyncSharedState 在合约安全组共享调用栈和返回值变化后重建合约间调用场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedInteractionState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// interactionModel 保存调用类型、调用栈和返回值。
type interactionModel struct {
	CallType      string   `json:"call_type"`
	CallStack     []string `json:"call_stack"`
	CurrentTarget string   `json:"current_target"`
	ReturnValue   string   `json:"return_value"`
	ContextOwner  string   `json:"context_owner"`
	StorageWrite  string   `json:"storage_write"`
	GasUsed       int      `json:"gas_used"`
}

// rebuildState 将调用模型映射为节点、调用消息和指标。
func rebuildState(state *framework.SceneState, model interactionModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[nodeIndexForPhase(state.PhaseIndex)].Status = "active"
	if phase == "返回值传播" {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "call-path", Label: model.CallType, Kind: "call", Status: phase, SourceID: "caller", TargetID: framework.NormalizeSlug(model.CurrentTarget, "vault")},
		{ID: "return-path", Label: model.ReturnValue, Kind: "call", Status: phase, SourceID: framework.NormalizeSlug(model.CurrentTarget, "vault"), TargetID: "receiver"},
	}
	state.Metrics = []framework.Metric{
		{Key: "call_type", Label: "调用类型", Value: model.CallType, Tone: toneByCallType(model.CallType)},
		{Key: "target", Label: "目标合约", Value: model.CurrentTarget, Tone: "info"},
		{Key: "context", Label: "上下文归属", Value: model.ContextOwner, Tone: "warning"},
		{Key: "gas", Label: "调用 Gas", Value: fmt.Sprintf("%d", model.GasUsed), Tone: "warning"},
		{Key: "stack_depth", Label: "调用栈深度", Value: fmt.Sprintf("%d", len(model.CallStack)), Tone: "warning"},
		{Key: "return", Label: "返回值", Value: model.ReturnValue, Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "调用栈", Value: strings.Join(model.CallStack, " -> ")},
		{Label: "上下文", Value: model.CallType},
		{Label: "存储写入", Value: model.StorageWrite},
		{Label: "阶段", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":           phase,
		"contract_interaction": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 call 与 delegatecall 的进入调用、上下文切换和返回值传播。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// nodeIndexForPhase 返回当前四节点图中的可用高亮索引。
func nodeIndexForPhase(phaseIndex int) int {
	if phaseIndex > 3 {
		return 3
	}
	if phaseIndex < 0 {
		return 0
	}
	return phaseIndex
}

// decodeModel 从通用 JSON 状态恢复调用模型。
func decodeModel(state *framework.SceneState) interactionModel {
	entry, ok := state.Data["contract_interaction"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["contract_interaction"].(interactionModel); ok {
			return typed
		}
		return interactionModel{
			CallType:      "call",
			CallStack:     []string{"Caller"},
			CurrentTarget: "Vault",
			ReturnValue:   "pending",
			ContextOwner:  "Caller",
			StorageWrite:  "Vault.balance",
			GasUsed:       700,
		}
	}
	return interactionModel{
		CallType:      framework.StringValue(entry["call_type"], "call"),
		CallStack:     framework.ToStringSliceOr(entry["call_stack"], []string{"Caller"}),
		CurrentTarget: framework.StringValue(entry["current_target"], "Vault"),
		ReturnValue:   framework.StringValue(entry["return_value"], "pending"),
		ContextOwner:  framework.StringValue(entry["context_owner"], "Caller"),
		StorageWrite:  framework.StringValue(entry["storage_write"], "Vault.balance"),
		GasUsed:       int(framework.NumberValue(entry["gas_used"], 700)),
	}
}

// applySharedInteractionState 将共享调用栈和返回值映射到合约交互模型。
func applySharedInteractionState(model *interactionModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if callStack, ok := sharedState["call_stack"].(map[string]any); ok {
		if stack := framework.ToStringSlice(callStack["stack"]); len(stack) > 0 {
			model.CallStack = stack
		}
		if target, ok := callStack["target"].(string); ok && strings.TrimSpace(target) != "" {
			model.CurrentTarget = target
		}
		if callType, ok := callStack["type"].(string); ok && strings.TrimSpace(callType) != "" {
			model.CallType = callType
		}
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		if returnValue, ok := contractState["return_value"].(string); ok && strings.TrimSpace(returnValue) != "" {
			model.ReturnValue = returnValue
		}
		if storage, ok := contractState["storage"].(map[string]any); ok {
			if contextOwner, ok := storage["context_owner"].(string); ok && strings.TrimSpace(contextOwner) != "" {
				model.ContextOwner = contextOwner
			}
			if storageWrite, ok := storage["storage_write"].(string); ok && strings.TrimSpace(storageWrite) != "" {
				model.StorageWrite = storageWrite
			}
			if gasUsed, ok := storage["gas_used"]; ok {
				model.GasUsed = int(framework.NumberValue(gasUsed, float64(model.GasUsed)))
			}
		}
	}
}

// nextPhase 返回合约交互流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "进入调用":
		return "上下文切换"
	case "上下文切换":
		return "返回值传播"
	default:
		return "进入调用"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "进入调用":
		return 0
	case "上下文切换":
		return 1
	case "返回值传播":
		return 3
	default:
		return 0
	}
}

// toneByCallType 返回调用类型对应的色调。
func toneByCallType(callType string) string {
	if callType == "delegatecall" {
		return "warning"
	}
	return "success"
}

// gasCostByCallType 返回不同调用语义的近似 Gas 成本。
func gasCostByCallType(callType string) int {
	switch callType {
	case "delegatecall":
		return 700
	case "staticcall":
		return 40
	default:
		return 700
	}
}

// resolveContextOwner 返回当前执行上下文归属。
func resolveContextOwner(callType string, target string) string {
	if callType == "delegatecall" {
		return "Caller"
	}
	return target
}

// resolveStorageWrite 返回本次调用写入的主要存储位置。
func resolveStorageWrite(callType string, target string) string {
	if callType == "delegatecall" {
		return "Caller.storage->" + target
	}
	return target + ".storage"
}
