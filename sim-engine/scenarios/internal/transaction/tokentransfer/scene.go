package tokentransfer

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 Token 转账流转场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "token-transfer",
		Title:        "Token 转账流转",
		Phase:        "扣减余额",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1300,
		TotalTicks:   12,
		Stages:       []string{"扣减余额", "事件广播", "接收到账"},
		Nodes: []framework.Node{
			{ID: "sender", Label: "Sender", Status: "active", Role: "account", X: 120, Y: 200},
			{ID: "token", Label: "Token", Status: "normal", Role: "contract", X: 340, Y: 200},
			{ID: "receiver", Label: "Receiver", Status: "normal", Role: "account", X: 560, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化账户余额、转账数量和事件日志。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := transferModel{
		TokenSymbol:      "LENS",
		SenderBalance:    120,
		ReceiverBalance:  45,
		Amount:           int(framework.NumberValue(input.Params["amount"], 10)),
		TxID:             "erc20-transfer-1",
		EventDispatched:  false,
		ReceiverCredited: false,
		EventLog:         []string{"TransferInitiated"},
	}
	applySharedTransferState(&model, input.SharedState)
	return rebuildState(state, model, "扣减余额")
}

// Step 推进余额扣减、Transfer 事件广播和到账确认。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedTransferState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "扣减余额"))
	switch phase {
	case "事件广播":
		model.SenderBalance -= model.Amount
		model.EventDispatched = true
		model.EventLog = append(model.EventLog, "TransferEvent")
	case "接收到账":
		if !model.EventDispatched {
			model.SenderBalance -= model.Amount
			model.EventDispatched = true
			model.EventLog = append(model.EventLog, "TransferEvent")
		}
		model.ReceiverBalance += model.Amount
		model.ReceiverCredited = true
		model.EventLog = append(model.EventLog, "ReceiverCredited")
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Token 转账进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"balances": map[string]any{
				"sender":   model.SenderBalance,
				"receiver": model.ReceiverBalance,
			},
			"transactions": map[string]any{
				model.TxID: map[string]any{
					"amount":  model.Amount,
					"token":   model.TokenSymbol,
					"settled": model.ReceiverCredited,
					"emitted": model.EventDispatched,
					"events":  model.EventLog,
				},
			},
		},
	}, nil
}

// HandleAction 允许重新发起新的 Token 转账并重置阶段状态。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Amount = int(framework.NumberValue(input.Params["amount"], 10))
	model.TxID = fmt.Sprintf("erc20-transfer-%d", state.Tick+1)
	model.EventDispatched = false
	model.ReceiverCredited = false
	model.EventLog = []string{"TransferInitiated"}
	if err := rebuildState(state, model, "扣减余额"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "发起转账", fmt.Sprintf("发起 %d %s 转账。", model.Amount, model.TokenSymbol), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"transactions": map[string]any{
				model.TxID: map[string]any{
					"amount": model.Amount,
					"token":  model.TokenSymbol,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出余额变化、Transfer 事件和到账状态。
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

// SyncSharedState 在余额、交易池和打包状态联动变化后重建 Token 转账场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedTransferState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// transferModel 保存 ERC-20 转账中余额和事件状态。
type transferModel struct {
	TokenSymbol      string   `json:"token_symbol"`
	SenderBalance    int      `json:"sender_balance"`
	ReceiverBalance  int      `json:"receiver_balance"`
	Amount           int      `json:"amount"`
	TxID             string   `json:"tx_id"`
	EventDispatched  bool     `json:"event_dispatched"`
	ReceiverCredited bool     `json:"receiver_credited"`
	EventLog         []string `json:"event_log"`
}

// rebuildState 将账户余额和事件状态映射为前端可视化结构。
func rebuildState(state *framework.SceneState, model transferModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes[0].Status = "normal"
	state.Nodes[1].Status = "normal"
	state.Nodes[2].Status = "normal"
	state.Nodes[0].Load = float64(model.SenderBalance)
	state.Nodes[1].Load = float64(model.Amount)
	state.Nodes[2].Load = float64(model.ReceiverBalance)
	switch phase {
	case "扣减余额":
		state.Nodes[0].Status = "active"
	case "事件广播":
		state.Nodes[1].Status = "active"
	case "接收到账":
		state.Nodes[2].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: model.TxID + "-transfer", Label: fmt.Sprintf("%s %d", model.TokenSymbol, model.Amount), Kind: "transaction", Status: phase, SourceID: "sender", TargetID: "token"},
		{ID: model.TxID + "-event", Label: "Transfer Event", Kind: "transaction", Status: eventStatus(model), SourceID: "token", TargetID: "receiver"},
	}
	state.Metrics = []framework.Metric{
		{Key: "sender_balance", Label: "发送方余额", Value: fmt.Sprintf("%d", model.SenderBalance), Tone: "info"},
		{Key: "receiver_balance", Label: "接收方余额", Value: fmt.Sprintf("%d", model.ReceiverBalance), Tone: "success"},
		{Key: "amount", Label: "转账数量", Value: fmt.Sprintf("%d %s", model.Amount, model.TokenSymbol), Tone: "warning"},
		{Key: "event", Label: "事件状态", Value: framework.BoolText(model.EventDispatched), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "交易标识", Value: model.TxID},
		{Label: "是否到账", Value: framework.BoolText(model.ReceiverCredited)},
		{Label: "Token", Value: model.TokenSymbol},
		{Label: "事件日志", Value: strings.Join(model.EventLog, ", ")},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"token_transfer": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实展示 ERC-20 转账中的扣减余额、Transfer 事件广播和接收到账。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 Token 转账状态。
func decodeModel(state *framework.SceneState) transferModel {
	entry, ok := state.Data["token_transfer"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["token_transfer"].(transferModel); ok {
			return typed
		}
		return transferModel{TokenSymbol: "LENS", SenderBalance: 120, ReceiverBalance: 45, Amount: 10, TxID: "erc20-transfer-1"}
	}
	return transferModel{
		TokenSymbol:      framework.StringValue(entry["token_symbol"], "LENS"),
		SenderBalance:    int(framework.NumberValue(entry["sender_balance"], 120)),
		ReceiverBalance:  int(framework.NumberValue(entry["receiver_balance"], 45)),
		Amount:           int(framework.NumberValue(entry["amount"], 10)),
		TxID:             framework.StringValue(entry["tx_id"], "erc20-transfer-1"),
		EventDispatched:  framework.BoolValue(entry["event_dispatched"], false),
		ReceiverCredited: framework.BoolValue(entry["receiver_credited"], false),
		EventLog:         framework.ToStringSlice(entry["event_log"]),
	}
}

// applySharedTransferState 将交易处理组中的余额、交易池和打包状态映射回 Token 转账场景。
func applySharedTransferState(model *transferModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if balances, ok := sharedState["balances"].(map[string]any); ok {
		sender := int(framework.NumberValue(balances["sender"], float64(model.SenderBalance)))
		receiver := int(framework.NumberValue(balances["receiver"], float64(model.ReceiverBalance)))
		if sender != 0 || receiver != 0 {
			model.SenderBalance = sender
			model.ReceiverBalance = receiver
			model.EventDispatched = true
		}
	}
	if transactions, ok := sharedState["transactions"].(map[string]any); ok {
		if linked, ok := transactions[model.TxID].(map[string]any); ok {
			model.EventDispatched = framework.BoolValue(linked["emitted"], model.EventDispatched)
			model.ReceiverCredited = framework.BoolValue(linked["settled"], model.ReceiverCredited)
			model.Amount = int(framework.NumberValue(linked["amount"], float64(model.Amount)))
			model.TokenSymbol = framework.StringValue(linked["token"], model.TokenSymbol)
		}
	}
	if mempool, ok := sharedState["mempool"].(map[string]any); ok {
		if included, ok := mempool["included"].([]any); ok && len(included) > 0 {
			model.ReceiverCredited = true
		}
	}
}

// nextPhase 返回 Token 转账的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "扣减余额":
		return "事件广播"
	case "事件广播":
		return "接收到账"
	default:
		return "接收到账"
	}
}

// phaseIndex 将阶段映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "扣减余额":
		return 0
	case "事件广播":
		return 1
	case "接收到账":
		return 2
	default:
		return 0
	}
}

// eventStatus 返回事件消息的当前状态。
func eventStatus(model transferModel) string {
	if model.ReceiverCredited {
		return "settled"
	}
	if model.EventDispatched {
		return "emitted"
	}
	return "pending"
}

// toneByPhase 返回阶段对应的展示色调。
func toneByPhase(phase string) string {
	switch phase {
	case "接收到账":
		return "success"
	case "事件广播":
		return "warning"
	default:
		return "info"
	}
}
