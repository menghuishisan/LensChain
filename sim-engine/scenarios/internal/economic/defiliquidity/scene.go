package defiliquidity

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 DeFi 流动性池场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "defi-liquidity",
		Title:        "DeFi 流动性池",
		Phase:        "池子初始化",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   12,
		Stages:       []string{"池子初始化", "交易撮合", "价格偏移", "无常损失"},
		Nodes: []framework.Node{
			{ID: "pool-x", Label: "Pool-X", Status: "active", Role: "amm", X: 120, Y: 170},
			{ID: "pool-y", Label: "Pool-Y", Status: "normal", Role: "amm", X: 320, Y: 170},
			{ID: "lp", Label: "LP", Status: "normal", Role: "amm", X: 220, Y: 330},
			{ID: "trader", Label: "Trader", Status: "normal", Role: "amm", X: 520, Y: 170},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 AMM 池子储备、价格和 LP 净值。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := liquidityModel{
		ReserveX:        1000,
		ReserveY:        1000,
		KValue:          1000000,
		TradeAmount:     20,
		OutputAmount:    19.61,
		SpotPrice:       1,
		LPShareValue:    2000,
		HODLValue:       2000,
		ImpermanentLoss: 0,
	}
	return rebuildState(state, model, "池子初始化")
}

// Step 推进撮合、价格偏移与无常损失计算。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "池子初始化"))
	switch phase {
	case "交易撮合":
		model.OutputAmount = swapOutput(model.ReserveX, model.ReserveY, model.TradeAmount)
		model.ReserveX += model.TradeAmount
		model.ReserveY -= model.OutputAmount
		model.KValue = model.ReserveX * model.ReserveY
	case "价格偏移":
		model.SpotPrice = model.ReserveY / model.ReserveX
	case "无常损失":
		model.LPShareValue = 2 * sqrt(model.ReserveX*model.ReserveY)
		model.HODLValue = 1000 + 1000*model.SpotPrice
		model.ImpermanentLoss = calculateImpermanentLoss(model.LPShareValue, model.HODLValue)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("流动性池进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 执行新的资产兑换，并立即重算池子状态。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.TradeAmount = framework.NumberValue(input.Params["amount"], 20)
	model.OutputAmount = swapOutput(model.ReserveX, model.ReserveY, model.TradeAmount)
	model.ReserveX += model.TradeAmount
	model.ReserveY -= model.OutputAmount
	model.KValue = model.ReserveX * model.ReserveY
	model.SpotPrice = model.ReserveY / model.ReserveX
	model.LPShareValue = 2 * sqrt(model.ReserveX*model.ReserveY)
	model.HODLValue = 1000 + 1000*model.SpotPrice
	model.ImpermanentLoss = calculateImpermanentLoss(model.LPShareValue, model.HODLValue)
	if err := rebuildState(state, model, "交易撮合"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "执行兑换", fmt.Sprintf("已向池子注入 %.2f 单位资产。", model.TradeAmount), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出流动性池储备、价格和无常损失。
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

// SyncSharedState 在交易处理组共享价格与余额变化后重建流动性池场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedLiquidityState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// liquidityModel 保存 AMM 池储备、价格和 LP 收益对比。
type liquidityModel struct {
	ReserveX        float64 `json:"reserve_x"`
	ReserveY        float64 `json:"reserve_y"`
	KValue          float64 `json:"k_value"`
	TradeAmount     float64 `json:"trade_amount"`
	OutputAmount    float64 `json:"output_amount"`
	SpotPrice       float64 `json:"spot_price"`
	LPShareValue    float64 `json:"lp_share_value"`
	HODLValue       float64 `json:"hodl_value"`
	ImpermanentLoss float64 `json:"impermanent_loss"`
}

// rebuildState 将 AMM 模型映射为池子图元、交易消息和指标。
func rebuildState(state *framework.SceneState, model liquidityModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = "active"
	state.Nodes[0].Load = model.ReserveX
	state.Nodes[1].Status = "active"
	state.Nodes[1].Load = model.ReserveY
	state.Nodes[2].Status = lpStatus(phase)
	state.Nodes[2].Load = model.LPShareValue
	state.Nodes[3].Status = traderStatus(phase)
	state.Nodes[3].Load = model.TradeAmount
	state.Messages = []framework.Message{
		{ID: "swap-in", Label: fmt.Sprintf("in:%.2f", model.TradeAmount), Kind: "proposal", Status: phase, SourceID: "trader", TargetID: "pool-x"},
		{ID: "swap-out", Label: fmt.Sprintf("out:%.2f", model.OutputAmount), Kind: "proposal", Status: phase, SourceID: "pool-y", TargetID: "trader"},
		{ID: "lp-value", Label: fmt.Sprintf("IL:%.2f%%", model.ImpermanentLoss*100), Kind: "proposal", Status: phase, SourceID: "pool-x", TargetID: "lp"},
	}
	state.Metrics = []framework.Metric{
		{Key: "price", Label: "即时价格", Value: fmt.Sprintf("%.4f", model.SpotPrice), Tone: "info"},
		{Key: "k", Label: "常数积 k", Value: fmt.Sprintf("%.0f", model.KValue), Tone: "warning"},
		{Key: "slippage", Label: "滑点输出", Value: fmt.Sprintf("%.2f", model.OutputAmount), Tone: "success"},
		{Key: "loss", Label: "无常损失", Value: fmt.Sprintf("%.2f%%", model.ImpermanentLoss*100), Tone: lossTone(model.ImpermanentLoss)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "LP 持仓价值", Value: fmt.Sprintf("%.2f", model.LPShareValue)},
		{Label: "HODL 价值", Value: fmt.Sprintf("%.2f", model.HODLValue)},
		{Label: "储备比", Value: fmt.Sprintf("%.2f : %.2f", model.ReserveX, model.ReserveY)},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"defi_liquidity": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 x*y=k 池子、兑换滑点、价格偏移与 LP 无常损失。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 AMM 模型。
func decodeModel(state *framework.SceneState) liquidityModel {
	entry, ok := state.Data["defi_liquidity"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["defi_liquidity"].(liquidityModel); ok {
			return typed
		}
		return liquidityModel{
			ReserveX:     1000,
			ReserveY:     1000,
			KValue:       1000000,
			TradeAmount:  20,
			OutputAmount: 19.61,
			SpotPrice:    1,
			LPShareValue: 2000,
			HODLValue:    2000,
		}
	}
	return liquidityModel{
		ReserveX:        framework.NumberValue(entry["reserve_x"], 1000),
		ReserveY:        framework.NumberValue(entry["reserve_y"], 1000),
		KValue:          framework.NumberValue(entry["k_value"], 1000000),
		TradeAmount:     framework.NumberValue(entry["trade_amount"], 20),
		OutputAmount:    framework.NumberValue(entry["output_amount"], 19.61),
		SpotPrice:       framework.NumberValue(entry["spot_price"], 1),
		LPShareValue:    framework.NumberValue(entry["lp_share_value"], 2000),
		HODLValue:       framework.NumberValue(entry["hodl_value"], 2000),
		ImpermanentLoss: framework.NumberValue(entry["impermanent_loss"], 0),
	}
}

// applySharedLiquidityState 将共享余额与交易池状态映射回流动性池模型。
func applySharedLiquidityState(model *liquidityModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if balances, ok := sharedState["balances"].(map[string]any); ok {
		if sender, ok := balances["sender"]; ok {
			model.ReserveX = framework.NumberValue(sender, model.ReserveX)
		}
		if receiver, ok := balances["receiver"]; ok {
			model.ReserveY = framework.NumberValue(receiver, model.ReserveY)
		}
	}
	if gas, ok := sharedState["gas"].(map[string]any); ok {
		if spotPrice, ok := gas["effective_price"]; ok {
			model.SpotPrice = framework.NumberValue(spotPrice, model.SpotPrice)
		}
	}
	model.KValue = model.ReserveX * model.ReserveY
	model.LPShareValue = 2 * sqrt(model.ReserveX*model.ReserveY)
	model.HODLValue = 1000 + 1000*model.SpotPrice
	model.ImpermanentLoss = calculateImpermanentLoss(model.LPShareValue, model.HODLValue)
}

// swapOutput 根据常数积 AMM 公式计算输出数量。
func swapOutput(reserveIn float64, reserveOut float64, amountIn float64) float64 {
	amountInWithFee := amountIn * 0.997
	return (amountInWithFee * reserveOut) / (reserveIn + amountInWithFee)
}

// calculateImpermanentLoss 计算 LP 相对持币的损失比例。
func calculateImpermanentLoss(lpValue float64, hodlValue float64) float64 {
	if hodlValue == 0 {
		return 0
	}
	return framework.Clamp((hodlValue-lpValue)/hodlValue, -1, 1)
}

// sqrt 返回浮点平方根。
func sqrt(value float64) float64 {
	if value <= 0 {
		return 0
	}
	x := value
	for index := 0; index < 8; index++ {
		x = 0.5 * (x + value/x)
	}
	return x
}

// nextPhase 返回流动性池的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "池子初始化":
		return "交易撮合"
	case "交易撮合":
		return "价格偏移"
	case "价格偏移":
		return "无常损失"
	default:
		return "池子初始化"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "池子初始化":
		return 0
	case "交易撮合":
		return 1
	case "价格偏移":
		return 2
	case "无常损失":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回阶段事件色调。
func toneByPhase(phase string) string {
	if phase == "无常损失" {
		return "warning"
	}
	if phase == "交易撮合" {
		return "success"
	}
	return "info"
}

// lpStatus 返回 LP 节点状态。
func lpStatus(phase string) string {
	if phase == "无常损失" {
		return "warning"
	}
	return "normal"
}

// traderStatus 返回交易者节点状态。
func traderStatus(phase string) string {
	if phase == "交易撮合" || phase == "价格偏移" {
		return "active"
	}
	return "normal"
}

// lossTone 返回无常损失指标色调。
func lossTone(loss float64) string {
	if loss > 0.03 {
		return "warning"
	}
	return "info"
}
