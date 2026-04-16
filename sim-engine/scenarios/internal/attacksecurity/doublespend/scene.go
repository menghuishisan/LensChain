package doublespend

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// defaultRiskThreshold 表示商家认为已经相对安全的确认数阈值。
	defaultRiskThreshold = 3
)

// DefaultState 构造双花攻击场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "double-spend",
		Title:        "双花攻击",
		Phase:        "发送商家交易",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   16,
		Stages:       []string{"发送商家交易", "秘密构造冲突交易", "确认数竞赛", "替换已确认交易"},
		Nodes: []framework.Node{
			{ID: "merchant", Label: "Merchant", Status: "normal", Role: "merchant", X: 120, Y: 180},
			{ID: "attacker", Label: "Attacker", Status: "normal", Role: "attacker", X: 320, Y: 120},
			{ID: "miner", Label: "Miner", Status: "normal", Role: "miner", X: 520, Y: 180},
			{ID: "wallet", Label: "Wallet", Status: "normal", Role: "wallet", X: 320, Y: 320},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化诚实交易、冲突交易和风险评估状态。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := spendModel{
		MerchantTxID:    "merchant-tx-1",
		ConflictTxID:    "conflict-tx-1",
		MerchantAmount:  12,
		ConflictAmount:  12,
		HonestHeight:    100,
		PrivateHeight:   99,
		Confirmations:   0,
		RiskScore:       0.95,
		AcceptedTx:      "merchant-tx-1",
		MerchantSettled: false,
		AttackSucceeded: false,
	}
	return rebuildState(state, model, "发送商家交易")
}

// Step 推进两笔冲突交易的确认竞赛和链替换过程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "发送商家交易"))
	switch phase {
	case "秘密构造冲突交易":
		model.PrivateHeight++
		model.RiskScore = 0.78
	case "确认数竞赛":
		model.HonestHeight++
		model.PrivateHeight++
		model.Confirmations++
		model.MerchantSettled = model.Confirmations >= defaultRiskThreshold
		model.RiskScore = calculateRisk(model.Confirmations, model.PrivateHeight-model.HonestHeight)
	case "替换已确认交易":
		model.PrivateHeight += 2
		if model.PrivateHeight >= model.HonestHeight {
			model.AcceptedTx = model.ConflictTxID
			model.AttackSucceeded = true
			model.RiskScore = 1
		} else {
			model.AttackSucceeded = false
			model.RiskScore = 0.4
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("双花流程进入%s阶段。", phase), toneByOutcome(model.AttackSucceeded, phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 允许手动发起新的冲突交易，模拟攻击者重新组织双花路径。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.ConflictTxID = framework.StringValue(input.Params["tx"], "conflict-tx-manual")
	model.PrivateHeight++
	model.RiskScore = calculateRisk(model.Confirmations, model.PrivateHeight-model.HonestHeight)
	if err := rebuildState(state, model, "秘密构造冲突交易"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "构造冲突交易", fmt.Sprintf("攻击者广播新的冲突交易 %s。", model.ConflictTxID), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出冲突交易路径、确认进度和风险状态。
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

// SyncSharedState 在交易处理组共享交易池变化后重建双花攻击场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedDoubleSpendState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// spendModel 保存诚实链、私有链和商家风险的核心状态。
type spendModel struct {
	MerchantTxID    string  `json:"merchant_tx_id"`
	ConflictTxID    string  `json:"conflict_tx_id"`
	MerchantAmount  int     `json:"merchant_amount"`
	ConflictAmount  int     `json:"conflict_amount"`
	HonestHeight    int     `json:"honest_height"`
	PrivateHeight   int     `json:"private_height"`
	Confirmations   int     `json:"confirmations"`
	RiskScore       float64 `json:"risk_score"`
	AcceptedTx      string  `json:"accepted_tx"`
	MerchantSettled bool    `json:"merchant_settled"`
	AttackSucceeded bool    `json:"attack_succeeded"`
}

// rebuildState 将双花模型映射到节点、消息、指标和提示信息。
func rebuildState(state *framework.SceneState, model spendModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = merchantStatus(model)
	state.Nodes[0].Load = float64(model.Confirmations)
	state.Nodes[1].Status = attackerStatus(model, phase)
	state.Nodes[1].Load = float64(model.PrivateHeight - model.HonestHeight + 1)
	state.Nodes[2].Status = "active"
	state.Nodes[2].Load = float64(model.HonestHeight)
	if model.AttackSucceeded {
		state.Nodes[3].Status = "warning"
	}
	state.Messages = []framework.Message{
		{ID: "merchant-tx", Label: model.MerchantTxID, Kind: "attack", Status: merchantMessageStatus(model), SourceID: "wallet", TargetID: "merchant"},
		{ID: "conflict-tx", Label: model.ConflictTxID, Kind: "attack", Status: conflictMessageStatus(model), SourceID: "attacker", TargetID: "miner"},
	}
	state.Metrics = []framework.Metric{
		{Key: "confirmations", Label: "确认数", Value: fmt.Sprintf("%d", model.Confirmations), Tone: "info"},
		{Key: "risk", Label: "商家风险", Value: fmt.Sprintf("%.0f%%", model.RiskScore*100), Tone: riskTone(model.RiskScore)},
		{Key: "accepted", Label: "当前接受交易", Value: model.AcceptedTx, Tone: toneByOutcome(model.AttackSucceeded, phase)},
		{Key: "private_lead", Label: "私链领先", Value: fmt.Sprintf("%d", model.PrivateHeight-model.HonestHeight), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "商家是否放货", Value: framework.BoolText(model.MerchantSettled)},
		{Label: "攻击是否成功", Value: framework.BoolText(model.AttackSucceeded)},
		{Label: "诚实链高度", Value: fmt.Sprintf("%d", model.HonestHeight)},
	}
	state.Data = map[string]any{
		"phase_name":   phase,
		"double_spend": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实展示诚实交易、秘密冲突交易、确认数竞赛与重组替换结果。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复双花模型。
func decodeModel(state *framework.SceneState) spendModel {
	entry, ok := state.Data["double_spend"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["double_spend"].(spendModel); ok {
			return typed
		}
		return spendModel{
			MerchantTxID:  "merchant-tx-1",
			ConflictTxID:  "conflict-tx-1",
			HonestHeight:  100,
			PrivateHeight: 99,
			RiskScore:     0.95,
			AcceptedTx:    "merchant-tx-1",
		}
	}
	return spendModel{
		MerchantTxID:    framework.StringValue(entry["merchant_tx_id"], "merchant-tx-1"),
		ConflictTxID:    framework.StringValue(entry["conflict_tx_id"], "conflict-tx-1"),
		MerchantAmount:  int(framework.NumberValue(entry["merchant_amount"], 12)),
		ConflictAmount:  int(framework.NumberValue(entry["conflict_amount"], 12)),
		HonestHeight:    int(framework.NumberValue(entry["honest_height"], 100)),
		PrivateHeight:   int(framework.NumberValue(entry["private_height"], 99)),
		Confirmations:   int(framework.NumberValue(entry["confirmations"], 0)),
		RiskScore:       framework.NumberValue(entry["risk_score"], 0.95),
		AcceptedTx:      framework.StringValue(entry["accepted_tx"], "merchant-tx-1"),
		MerchantSettled: framework.BoolValue(entry["merchant_settled"], false),
		AttackSucceeded: framework.BoolValue(entry["attack_succeeded"], false),
	}
}

// applySharedDoubleSpendState 将交易池共享状态映射到双花攻击场景。
func applySharedDoubleSpendState(model *spendModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if mempool, ok := sharedState["mempool"].(map[string]any); ok {
		if included, ok := mempool["included"].([]any); ok && len(included) > 0 {
			if tx, ok := included[0].(string); ok && strings.TrimSpace(tx) != "" {
				model.AcceptedTx = tx
				model.AttackSucceeded = tx == model.ConflictTxID
			}
		}
	}
}

// nextPhase 返回双花攻击流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "发送商家交易":
		return "秘密构造冲突交易"
	case "秘密构造冲突交易":
		return "确认数竞赛"
	case "确认数竞赛":
		return "替换已确认交易"
	default:
		return "替换已确认交易"
	}
}

// phaseIndex 将阶段文案映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "发送商家交易":
		return 0
	case "秘密构造冲突交易":
		return 1
	case "确认数竞赛":
		return 2
	case "替换已确认交易":
		return 3
	default:
		return 0
	}
}

// calculateRisk 根据确认数和私链领先程度估算商家风险。
func calculateRisk(confirmations int, lead int) float64 {
	base := 0.95 - float64(confirmations)*0.18 + float64(lead)*0.22
	return framework.Clamp(base, 0.05, 1)
}

// merchantStatus 返回商家节点当前状态。
func merchantStatus(model spendModel) string {
	if model.AttackSucceeded {
		return "warning"
	}
	if model.MerchantSettled {
		return "success"
	}
	return "active"
}

// attackerStatus 返回攻击者节点当前状态。
func attackerStatus(model spendModel, phase string) string {
	if model.AttackSucceeded {
		return "success"
	}
	if phase == "秘密构造冲突交易" || phase == "确认数竞赛" {
		return "warning"
	}
	return "normal"
}

// merchantMessageStatus 返回商家交易的显示状态。
func merchantMessageStatus(model spendModel) string {
	if model.AcceptedTx == model.MerchantTxID {
		return "confirmed"
	}
	return "replaced"
}

// conflictMessageStatus 返回冲突交易的显示状态。
func conflictMessageStatus(model spendModel) string {
	if model.AcceptedTx == model.ConflictTxID {
		return "confirmed"
	}
	return "private"
}

// riskTone 根据风险分值选择色调。
func riskTone(risk float64) string {
	if risk >= 0.7 {
		return "warning"
	}
	if risk <= 0.3 {
		return "success"
	}
	return "info"
}

// toneByOutcome 返回攻击结果对应的事件色调。
func toneByOutcome(success bool, phase string) string {
	if success {
		return "warning"
	}
	if phase == "确认数竞赛" {
		return "info"
	}
	return "success"
}
