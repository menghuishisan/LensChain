package gasmarket

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// targetUtilization 表示 EIP-1559 调整基准所使用的目标利用率。
	targetUtilization = 0.50
)

// DefaultState 构造 Gas 费市场场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "gas-market",
		Title:        "Gas 费市场（EIP-1559）",
		Phase:        "采样利用率",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   16,
		Stages:       []string{"采样利用率", "调整基础费", "统计燃烧"},
		Nodes: []framework.Node{
			{ID: "base-fee", Label: "BaseFee", Status: "normal", Role: "fee", X: 120, Y: 200},
			{ID: "tip", Label: "Tip", Status: "normal", Role: "fee", X: 300, Y: 120},
			{ID: "burn", Label: "Burn", Status: "normal", Role: "fee", X: 300, Y: 280},
			{ID: "utilization", Label: "BlockUtilization", Status: "normal", Role: "fee", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化基础费、小费、区块利用率和累计燃烧量。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := marketModel{
		BaseFee:     30,
		TipMedian:   2,
		Utilization: 0.48,
		BurnedTotal: 0,
		DemandSpike: 1,
		BlockGas:    15000000,
	}
	return rebuildState(state, model, "采样利用率")
}

// Step 根据 EIP-1559 规则推进基础费、利用率和燃烧量。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "采样利用率"))
	switch phase {
	case "调整基础费":
		model.Utilization = framework.Clamp(model.Utilization+0.08*model.DemandSpike, 0.1, 1)
		adjust := (model.Utilization - targetUtilization) * 12
		model.BaseFee = framework.Clamp(model.BaseFee+adjust, 1, 400)
	case "统计燃烧":
		burnThisBlock := model.BaseFee * (model.BlockGas / 1000000) * model.Utilization
		model.BurnedTotal += burnThisBlock
		model.TipMedian = framework.Clamp(model.TipMedian+0.4*model.DemandSpike, 1, 20)
	default:
		model.Utilization = framework.Clamp(model.Utilization*0.92, 0.1, 1)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Gas 市场进入%s阶段。", phase), toneByMarket(model))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 允许注入短期需求高峰，抬升基础费和小费分布。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.DemandSpike = framework.Clamp(framework.NumberValue(input.Params["demand"], 2), 1, 4)
	model.Utilization = framework.Clamp(model.Utilization+0.12*model.DemandSpike, 0.1, 1)
	if err := rebuildState(state, model, "调整基础费"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "提升需求", fmt.Sprintf("已注入 %.1fx 的交易需求高峰。", model.DemandSpike), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出基础费、小费、利用率和燃烧量。
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

// SyncSharedState 在交易处理组共享 Gas 与交易池变化后重建 Gas 市场场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedGasMarketState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// marketModel 保存 EIP-1559 市场中的价格和利用率状态。
type marketModel struct {
	BaseFee     float64 `json:"base_fee"`
	TipMedian   float64 `json:"tip_median"`
	Utilization float64 `json:"utilization"`
	BurnedTotal float64 `json:"burned_total"`
	DemandSpike float64 `json:"demand_spike"`
	BlockGas    float64 `json:"block_gas"`
}

// rebuildState 将 Gas 市场状态映射为节点、消息和指标。
func rebuildState(state *framework.SceneState, model marketModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Load = model.BaseFee
	state.Nodes[1].Load = model.TipMedian
	state.Nodes[2].Load = model.BurnedTotal
	state.Nodes[3].Load = model.Utilization * 100
	state.Nodes[0].Status = "active"
	if phase == "统计燃烧" {
		state.Nodes[2].Status = "warning"
	}
	if model.Utilization > targetUtilization {
		state.Nodes[3].Status = "warning"
	}
	state.Messages = []framework.Message{
		{ID: "base-fee-adjust", Label: "Base Fee Update", Kind: "proposal", Status: phase, SourceID: "utilization", TargetID: "base-fee"},
		{ID: "burn-stats", Label: "Burn Stats", Kind: "proposal", Status: phase, SourceID: "base-fee", TargetID: "burn"},
	}
	state.Metrics = []framework.Metric{
		{Key: "base_fee", Label: "基础费", Value: framework.MetricValue(model.BaseFee, " gwei"), Tone: toneByMarket(model)},
		{Key: "tip", Label: "小费中位数", Value: framework.MetricValue(model.TipMedian, " gwei"), Tone: "info"},
		{Key: "utilization", Label: "区块利用率", Value: fmt.Sprintf("%.0f%%", model.Utilization*100), Tone: toneByUtilization(model.Utilization)},
		{Key: "burn", Label: "累计燃烧", Value: framework.MetricValue(model.BurnedTotal, " ETH"), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "目标利用率", Value: "50%"},
		{Label: "需求倍率", Value: fmt.Sprintf("%.1fx", model.DemandSpike)},
		{Label: "区块 Gas 上限", Value: fmt.Sprintf("%.0f", model.BlockGas)},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"gas_market": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 EIP-1559 中基础费调整、区块利用率采样和燃烧统计。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 Gas 市场模型。
func decodeModel(state *framework.SceneState) marketModel {
	entry, ok := state.Data["gas_market"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["gas_market"].(marketModel); ok {
			return typed
		}
		return marketModel{BaseFee: 30, TipMedian: 2, Utilization: 0.48, DemandSpike: 1, BlockGas: 15000000}
	}
	return marketModel{
		BaseFee:     framework.NumberValue(entry["base_fee"], 30),
		TipMedian:   framework.NumberValue(entry["tip_median"], 2),
		Utilization: framework.NumberValue(entry["utilization"], 0.48),
		BurnedTotal: framework.NumberValue(entry["burned_total"], 0),
		DemandSpike: framework.NumberValue(entry["demand_spike"], 1),
		BlockGas:    framework.NumberValue(entry["block_gas"], 15000000),
	}
}

// applySharedGasMarketState 将交易处理组共享 Gas 与 mempool 状态映射回 Gas 市场模型。
func applySharedGasMarketState(model *marketModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if gas, ok := sharedState["gas"].(map[string]any); ok {
		if baseFee, ok := gas["base_fee"]; ok {
			model.BaseFee = framework.NumberValue(baseFee, model.BaseFee)
		}
		if tipMedian, ok := gas["tip_median"]; ok {
			model.TipMedian = framework.NumberValue(tipMedian, model.TipMedian)
		}
		if utilization, ok := gas["utilization"]; ok {
			model.Utilization = framework.Clamp(framework.NumberValue(utilization, model.Utilization), 0.1, 1)
		}
	}
	if mempool, ok := sharedState["mempool"].(map[string]any); ok {
		if ordered, ok := mempool["ordered"].([]any); ok {
			model.DemandSpike = framework.Clamp(float64(len(ordered))/2, 1, 4)
		}
	}
}

// nextPhase 返回 Gas 市场的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "采样利用率":
		return "调整基础费"
	case "调整基础费":
		return "统计燃烧"
	default:
		return "采样利用率"
	}
}

// phaseIndex 将阶段映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "采样利用率":
		return 0
	case "调整基础费":
		return 1
	case "统计燃烧":
		return 2
	default:
		return 0
	}
}

// toneByMarket 根据基础费和利用率选择色调。
func toneByMarket(model marketModel) string {
	if model.Utilization > 0.8 || model.BaseFee > 80 {
		return "warning"
	}
	if model.BaseFee < 20 {
		return "success"
	}
	return "info"
}

// toneByUtilization 根据区块利用率返回展示色调。
func toneByUtilization(rate float64) string {
	if rate > 0.8 {
		return "warning"
	}
	if rate < 0.4 {
		return "success"
	}
	return "info"
}
