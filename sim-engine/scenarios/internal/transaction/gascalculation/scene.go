package gascalculation

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 Gas 计算与优化场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "gas-calculation",
		Title:        "Gas 计算与优化",
		Phase:        "Opcode 分析",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   10,
		Stages:       []string{"Opcode 分析", "Gas 汇总", "优化建议"},
		Nodes: []framework.Node{
			{ID: "opcode", Label: "Opcode", Status: "active", Role: "gas", X: 110, Y: 200},
			{ID: "gas", Label: "Gas", Status: "normal", Role: "gas", X: 290, Y: 110},
			{ID: "limit", Label: "Limit", Status: "normal", Role: "gas", X: 290, Y: 290},
			{ID: "advice", Label: "Advice", Status: "normal", Role: "gas", X: 510, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化默认操作码路径、Gas 用量和上限。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := gasModel{
		ActiveOpcode: "SSTORE",
		OpcodeGas: map[string]int{
			"SLOAD":  100,
			"SSTORE": 20000,
			"CALL":   700,
			"LOG1":   750,
		},
		Sequence: []string{"PUSH1", "PUSH1", "SSTORE", "LOG1"},
		GasLimit: 50000,
		Refund:   4800,
	}
	applySharedGasState(&model, input.SharedState)
	return rebuildState(state, model, "Opcode 分析")
}

// Step 推进操作码分析、Gas 汇总和优化建议阶段。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedGasState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "Opcode 分析"))
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Gas 计算进入%s阶段。", phase), toneByUsage(model))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"gas": map[string]any{
				"active_opcode": model.ActiveOpcode,
				"used":          totalGas(model),
				"limit":         model.GasLimit,
				"refund":        model.Refund,
			},
		},
	}, nil
}

// HandleAction 切换操作码路径并重新计算总 Gas。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.ActiveOpcode = framework.StringValue(input.Params["opcode"], "SSTORE")
	switch model.ActiveOpcode {
	case "SLOAD":
		model.Sequence = []string{"PUSH1", "SLOAD", "ADD"}
	case "CALL":
		model.Sequence = []string{"PUSH1", "CALL", "LOG1"}
	default:
		model.Sequence = []string{"PUSH1", "PUSH1", "SSTORE", "LOG1"}
	}
	if err := rebuildState(state, model, "Opcode 分析"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "切换操作码", fmt.Sprintf("已切换为 %s 路径。", model.ActiveOpcode), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"gas": map[string]any{
				"active_opcode": model.ActiveOpcode,
				"used":          totalGas(model),
				"limit":         model.GasLimit,
				"refund":        model.Refund,
			},
		},
	}, nil
}

// BuildRenderState 输出操作码、Gas 用量和优化建议。
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

// SyncSharedState 在共享 Gas、交易池和余额变化后重建 Gas 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedGasState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// gasModel 保存操作码路径、单项 Gas 和上限。
type gasModel struct {
	ActiveOpcode string         `json:"active_opcode"`
	OpcodeGas    map[string]int `json:"opcode_gas"`
	Sequence     []string       `json:"sequence"`
	GasLimit     int            `json:"gas_limit"`
	MempoolSize  int            `json:"mempool_size"`
	IncludedTxs  int            `json:"included_txs"`
	Refund       int            `json:"refund"`
}

// rebuildState 将 Gas 模型映射为节点、消息和指标。
func rebuildState(state *framework.SceneState, model gasModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	used := totalGas(model)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	state.Nodes[1].Load = float64(used)
	state.Nodes[2].Load = float64(model.GasLimit)
	if phase == "优化建议" {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "opcode-gas", Label: model.ActiveOpcode, Kind: "transaction", Status: phase, SourceID: "opcode", TargetID: "gas"},
		{ID: "gas-advice", Label: optimizationAdvice(model), Kind: "transaction", Status: phase, SourceID: "gas", TargetID: "advice"},
	}
	state.Metrics = []framework.Metric{
		{Key: "opcode", Label: "核心操作码", Value: model.ActiveOpcode, Tone: "info"},
		{Key: "used", Label: "Gas Used", Value: fmt.Sprintf("%d", used), Tone: toneByUsage(model)},
		{Key: "limit", Label: "Gas Limit", Value: fmt.Sprintf("%d", model.GasLimit), Tone: "warning"},
		{Key: "refund", Label: "Gas Refund", Value: fmt.Sprintf("%d", model.Refund), Tone: "info"},
		{Key: "advice", Label: "优化建议", Value: optimizationAdvice(model), Tone: "success"},
		{Key: "mempool", Label: "共享交易池", Value: fmt.Sprintf("%d", model.MempoolSize), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "执行路径", Value: strings.Join(model.Sequence, " -> ")},
		{Label: "余量", Value: fmt.Sprintf("%d", model.GasLimit-used+model.Refund)},
		{Label: "高耗点", Value: model.ActiveOpcode},
		{Label: "已纳入区块", Value: fmt.Sprintf("%d", model.IncludedTxs)},
	}
	state.Data = map[string]any{
		"phase_name":      phase,
		"gas_calculation": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟操作码级别的 Gas 累加、上限对比和优化建议输出。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 Gas 模型。
func decodeModel(state *framework.SceneState) gasModel {
	entry, ok := state.Data["gas_calculation"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["gas_calculation"].(gasModel); ok {
			return typed
		}
		return gasModel{
			ActiveOpcode: "SSTORE",
			OpcodeGas:    map[string]int{"SLOAD": 100, "SSTORE": 20000, "CALL": 700, "LOG1": 750},
			Sequence:     []string{"PUSH1", "PUSH1", "SSTORE", "LOG1"},
			GasLimit:     50000,
		}
	}
	return gasModel{
		ActiveOpcode: framework.StringValue(entry["active_opcode"], "SSTORE"),
		OpcodeGas:    framework.ToIntMapOr(entry["opcode_gas"], map[string]int{"SLOAD": 100, "SSTORE": 20000, "CALL": 700, "LOG1": 750}),
		Sequence:     framework.ToStringSliceOr(entry["sequence"], []string{"PUSH1", "PUSH1", "SSTORE", "LOG1"}),
		GasLimit:     int(framework.NumberValue(entry["gas_limit"], 50000)),
		MempoolSize:  int(framework.NumberValue(entry["mempool_size"], 0)),
		IncludedTxs:  int(framework.NumberValue(entry["included_txs"], 0)),
		Refund:       int(framework.NumberValue(entry["refund"], 4800)),
	}
}

// totalGas 统计当前执行路径的总 Gas。
func totalGas(model gasModel) int {
	total := 0
	for _, opcode := range model.Sequence {
		if cost, ok := model.OpcodeGas[opcode]; ok {
			total += cost
			continue
		}
		total += 3
	}
	return total
}

// optimizationAdvice 返回当前路径的优化建议。
func optimizationAdvice(model gasModel) string {
	switch model.ActiveOpcode {
	case "SSTORE":
		if model.MempoolSize > 2 {
			return "拥堵时减少重复写存储"
		}
		return "减少重复写存储"
	case "CALL":
		return "合并外部调用"
	default:
		return "复用缓存降低读写"
	}
}

// nextPhase 返回 Gas 计算的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "Opcode 分析":
		return "Gas 汇总"
	case "Gas 汇总":
		return "优化建议"
	default:
		return "Opcode 分析"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "Opcode 分析":
		return 0
	case "Gas 汇总":
		return 1
	case "优化建议":
		return 3
	default:
		return 0
	}
}

// toneByUsage 根据 Gas 用量返回色调。
func toneByUsage(model gasModel) string {
	if totalGas(model) > model.GasLimit*8/10 {
		return "warning"
	}
	return "success"
}

// applySharedGasState 将交易处理组中的交易池、打包结果和余额变化映射回 Gas 场景。
func applySharedGasState(model *gasModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if gasState, ok := sharedState["gas"].(map[string]any); ok {
		model.ActiveOpcode = framework.StringValue(gasState["active_opcode"], model.ActiveOpcode)
		model.GasLimit = int(framework.NumberValue(gasState["limit"], float64(model.GasLimit)))
	}
	if mempoolState, ok := sharedState["mempool"].(map[string]any); ok {
		model.MempoolSize = int(framework.NumberValue(mempoolState["size"], float64(model.MempoolSize)))
		if ordered, ok := mempoolState["ordered"].([]any); ok {
			model.MempoolSize = len(ordered)
			if containsBotTx(ordered) {
				model.ActiveOpcode = "CALL"
				model.Sequence = []string{"PUSH1", "CALL", "CALL", "LOG1"}
			} else if len(ordered) > 1 {
				model.ActiveOpcode = "SSTORE"
				model.Sequence = []string{"PUSH1", "SLOAD", "SSTORE", "LOG1"}
			}
		}
		if included, ok := mempoolState["included"].([]any); ok {
			model.IncludedTxs = len(included)
		}
	}
	if balances, ok := sharedState["balances"].(map[string]any); ok {
		sender := int(framework.NumberValue(balances["sender"], 0))
		receiver := int(framework.NumberValue(balances["receiver"], 0))
		if sender > 0 || receiver > 0 {
			model.ActiveOpcode = "LOG1"
			model.Sequence = []string{"PUSH1", "SLOAD", "LOG1"}
		}
	}
}

// containsBotTx 判断共享交易池中是否已经出现机器人插单。
func containsBotTx(values []any) bool {
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if framework.StringValue(entry["sender"], "") == "bot" {
			return true
		}
	}
	return false
}
