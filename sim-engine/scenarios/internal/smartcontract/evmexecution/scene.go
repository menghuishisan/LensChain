package evmexecution

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 EVM 执行步进场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "evm-execution",
		Title:        "EVM 执行步进",
		Phase:        "取指",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1300,
		TotalTicks:   20,
		Stages:       []string{"取指", "执行", "栈更新", "存储写回"},
		Nodes: []framework.Node{
			{ID: "pc", Label: "PC", Status: "active", Role: "vm", X: 120, Y: 200},
			{ID: "stack", Label: "Stack", Status: "normal", Role: "vm", X: 300, Y: 100},
			{ID: "memory", Label: "Memory", Status: "normal", Role: "vm", X: 300, Y: 300},
			{ID: "storage", Label: "Storage", Status: "normal", Role: "vm", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化操作码序列、程序计数器和栈内存状态。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := vmModel{
		ProgramCounter: 0,
		Opcodes:        []string{"PUSH1", "PUSH1", "ADD", "SSTORE"},
		Stack:          []int{2, 3},
		MemoryWords:    []int{0, 0, 0},
		StorageSlots:   map[string]int{"0x00": 0},
		LastOpcode:     "PUSH1",
		GasUsed:        3,
		MemorySize:     96,
	}
	applySharedExecutionState(&model, input.SharedState)
	return rebuildState(state, model, "取指")
}

// Step 推进取指、执行、栈更新和存储写回。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedExecutionState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "取指"))
	switch phase {
	case "执行":
		model.LastOpcode = currentOpcode(model)
	case "栈更新":
		applyOpcode(&model)
	case "存储写回":
		if model.LastOpcode == "SSTORE" && len(model.Stack) > 0 {
			model.StorageSlots["0x00"] = model.Stack[len(model.Stack)-1]
		}
		model.ProgramCounter = (model.ProgramCounter + 1) % len(model.Opcodes)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("EVM 执行进入%s阶段。", phase), toneByOpcode(model.LastOpcode))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"call_stack": map[string]any{
				"pc":     model.ProgramCounter,
				"opcode": model.LastOpcode,
				"stack":  model.Stack,
			},
			"contract_state": map[string]any{
				"storage": map[string]any{
					"0x00":     model.StorageSlots["0x00"],
					"gas_used": model.GasUsed,
				},
			},
		},
	}, nil
}

// HandleAction 手动推动一个操作码执行。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	opcode := framework.StringValue(input.Params["opcode"], currentOpcode(model))
	model.LastOpcode = opcode
	applyOpcodeWithName(&model, opcode)
	if err := rebuildState(state, model, "执行"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "执行单步", fmt.Sprintf("已手动执行操作码 %s。", opcode), toneByOpcode(opcode))
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"call_stack": map[string]any{
				"pc":     model.ProgramCounter,
				"opcode": opcode,
				"stack":  model.Stack,
			},
			"contract_state": map[string]any{
				"storage": map[string]any{
					"0x00":     model.StorageSlots["0x00"],
					"gas_used": model.GasUsed,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出 PC、栈、内存和存储状态。
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

// SyncSharedState 在调用栈与存储槽联动变化后重建 EVM 执行场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedExecutionState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// vmModel 保存 EVM 执行过程中的程序计数器、操作码和栈内存状态。
type vmModel struct {
	ProgramCounter int            `json:"program_counter"`
	Opcodes        []string       `json:"opcodes"`
	Stack          []int          `json:"stack"`
	MemoryWords    []int          `json:"memory_words"`
	StorageSlots   map[string]int `json:"storage_slots"`
	LastOpcode     string         `json:"last_opcode"`
	GasUsed        int            `json:"gas_used"`
	MemorySize     int            `json:"memory_size"`
}

// rebuildState 将虚拟机模型映射为执行面板节点、消息和指标。
func rebuildState(state *framework.SceneState, model vmModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes[0].Load = float64(model.ProgramCounter)
	state.Nodes[1].Load = float64(len(model.Stack))
	state.Nodes[2].Load = float64(len(model.MemoryWords))
	state.Nodes[3].Load = float64(model.StorageSlots["0x00"])
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	if phase == "存储写回" {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "opcode-flow", Label: model.LastOpcode, Kind: "call", Status: phase, SourceID: "pc", TargetID: "stack"},
		{ID: "storage-flow", Label: "slot 0x00", Kind: "call", Status: phase, SourceID: "stack", TargetID: "storage"},
	}
	state.Metrics = []framework.Metric{
		{Key: "pc", Label: "PC", Value: fmt.Sprintf("%d", model.ProgramCounter), Tone: "info"},
		{Key: "opcode", Label: "当前操作码", Value: model.LastOpcode, Tone: toneByOpcode(model.LastOpcode)},
		{Key: "gas", Label: "累计 Gas", Value: fmt.Sprintf("%d", model.GasUsed), Tone: "warning"},
		{Key: "memory_size", Label: "Memory Size", Value: fmt.Sprintf("%d", model.MemorySize), Tone: "info"},
		{Key: "stack_depth", Label: "栈深度", Value: fmt.Sprintf("%d", len(model.Stack)), Tone: "warning"},
		{Key: "slot", Label: "存储槽 0x00", Value: fmt.Sprintf("%d", model.StorageSlots["0x00"]), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "Stack", Value: formatIntSlice(model.Stack)},
		{Label: "Memory", Value: formatIntSlice(model.MemoryWords)},
		{Label: "Opcodes", Value: strings.Join(model.Opcodes, ", ")},
	}
	state.Data = map[string]any{
		"phase_name":    phase,
		"evm_execution": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 EVM 中取指、执行、栈变化和存储写回过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复虚拟机模型。
func decodeModel(state *framework.SceneState) vmModel {
	entry, ok := state.Data["evm_execution"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["evm_execution"].(vmModel); ok {
			return typed
		}
		return vmModel{
			ProgramCounter: 0,
			Opcodes:        []string{"PUSH1", "PUSH1", "ADD", "SSTORE"},
			Stack:          []int{2, 3},
			MemoryWords:    []int{0, 0, 0},
			StorageSlots:   map[string]int{"0x00": 0},
			LastOpcode:     "PUSH1",
			GasUsed:        3,
			MemorySize:     96,
		}
	}
	return vmModel{
		ProgramCounter: int(framework.NumberValue(entry["program_counter"], 0)),
		Opcodes:        framework.ToStringSliceOr(entry["opcodes"], []string{"PUSH1", "PUSH1", "ADD", "SSTORE"}),
		Stack:          framework.ToIntSlice(entry["stack"]),
		MemoryWords:    framework.ToIntSlice(entry["memory_words"]),
		StorageSlots:   framework.ToIntMapOr(entry["storage_slots"], map[string]int{"0x00": 0}),
		LastOpcode:     framework.StringValue(entry["last_opcode"], "PUSH1"),
		GasUsed:        int(framework.NumberValue(entry["gas_used"], 3)),
		MemorySize:     int(framework.NumberValue(entry["memory_size"], 96)),
	}
}

// applySharedExecutionState 将合约安全组中的调用栈和存储槽共享状态映射回 EVM 执行模型。
func applySharedExecutionState(model *vmModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if callState, ok := sharedState["call_stack"].(map[string]any); ok {
		model.ProgramCounter = int(framework.NumberValue(callState["pc"], float64(model.ProgramCounter)))
		model.LastOpcode = framework.StringValue(callState["opcode"], model.LastOpcode)
		if stack := framework.ToIntSlice(callState["stack"]); len(stack) > 0 {
			model.Stack = stack
		}
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		if storageRaw, ok := contractState["storage"].(map[string]any); ok {
			storage := framework.ToIntMapOr(storageRaw, map[string]int{"0x00": 0})
			if slotValue, ok := storageRaw["0x00"]; ok {
				storage["0x00"] = int(framework.NumberValue(slotValue, float64(storage["0x00"])))
			}
			if len(storage) > 0 {
				model.StorageSlots = storage
			}
			if gasUsed, ok := storageRaw["gas_used"]; ok {
				model.GasUsed = int(framework.NumberValue(gasUsed, float64(model.GasUsed)))
			}
		}
	}
}

// currentOpcode 返回当前程序计数器指向的操作码。
func currentOpcode(model vmModel) string {
	if len(model.Opcodes) == 0 {
		return "STOP"
	}
	index := model.ProgramCounter % len(model.Opcodes)
	return model.Opcodes[index]
}

// applyOpcode 按当前程序计数器执行操作码。
func applyOpcode(model *vmModel) {
	applyOpcodeWithName(model, currentOpcode(*model))
}

// applyOpcodeWithName 按给定操作码执行简化 EVM 逻辑。
func applyOpcodeWithName(model *vmModel, opcode string) {
	switch opcode {
	case "PUSH1":
		model.Stack = append(model.Stack, len(model.Stack)+1)
		model.GasUsed += 3
	case "ADD":
		if len(model.Stack) >= 2 {
			last := model.Stack[len(model.Stack)-1]
			prev := model.Stack[len(model.Stack)-2]
			model.Stack = model.Stack[:len(model.Stack)-2]
			model.Stack = append(model.Stack, last+prev)
		}
		model.GasUsed += 3
	case "SSTORE":
		if len(model.Stack) > 0 {
			model.StorageSlots["0x00"] = model.Stack[len(model.Stack)-1]
		}
		model.GasUsed += 20000
	case "SLOAD":
		model.Stack = append(model.Stack, model.StorageSlots["0x00"])
		model.GasUsed += 100
	}
	model.MemorySize = len(model.MemoryWords) * 32
}

// nextPhase 返回 EVM 执行流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "取指":
		return "执行"
	case "执行":
		return "栈更新"
	case "栈更新":
		return "存储写回"
	default:
		return "取指"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "取指":
		return 0
	case "执行":
		return 1
	case "栈更新":
		return 2
	case "存储写回":
		return 3
	default:
		return 0
	}
}

// toneByOpcode 返回操作码对应的展示色调。
func toneByOpcode(opcode string) string {
	switch opcode {
	case "ADD":
		return "success"
	case "SSTORE":
		return "warning"
	default:
		return "info"
	}
}

// formatIntSlice 格式化整数切片。
func formatIntSlice(values []int) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
