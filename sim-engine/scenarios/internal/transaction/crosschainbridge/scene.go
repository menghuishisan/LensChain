package crosschainbridge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造跨链桥接通信场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "cross-chain-bridge",
		Title:        "跨链桥接通信",
		Phase:        "源链锁定",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   16,
		Stages:       []string{"源链锁定", "证明生成", "中继提交", "目标链铸造"},
		Nodes: []framework.Node{
			{ID: "source", Label: "Source", Status: "active", Role: "bridge", X: 100, Y: 200},
			{ID: "vault", Label: "LockVault", Status: "normal", Role: "bridge", X: 270, Y: 110},
			{ID: "relay", Label: "Relay", Status: "normal", Role: "bridge", X: 430, Y: 200},
			{ID: "target", Label: "Target", Status: "normal", Role: "bridge", X: 600, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化双链资产余额、锁定凭证和桥接状态。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := bridgeModel{
		Asset:         "USDT",
		Amount:        100,
		SourceBalance: 1000,
		TargetBalance: 300,
		LockedAmount:  0,
		ProofHash:     "",
		RelayAccepted: false,
		MintedAmount:  0,
		BridgeStatus:  "idle",
		RelayMessage:  "pending",
	}
	return rebuildState(state, model, "源链锁定")
}

// Step 推进锁定、证明生成、中继提交与目标链铸造流程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "源链锁定"))
	switch phase {
	case "证明生成":
		model.LockedAmount = model.Amount
		model.SourceBalance -= model.Amount
		model.ProofHash = buildProofHash(model.Asset, model.Amount, model.SourceBalance)
		model.BridgeStatus = "proof_ready"
	case "中继提交":
		model.RelayAccepted = true
		model.RelayMessage = "relay-accepted"
		model.BridgeStatus = "relayed"
	case "目标链铸造":
		model.TargetBalance += model.Amount
		model.MintedAmount = model.Amount
		model.BridgeStatus = "minted"
		model.RelayMessage = "mint-confirmed"
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("跨链桥接进入%s阶段。", phase), toneByBridgePhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 发起新的桥接资产请求并重置桥接状态。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Asset = framework.StringValue(input.Params["asset"], "USDT")
	model.Amount = inferAmount(model.Asset)
	model.LockedAmount = 0
	model.ProofHash = ""
	model.RelayAccepted = false
	model.MintedAmount = 0
	model.BridgeStatus = "locking"
	model.RelayMessage = "pending"
	if err := rebuildState(state, model, "源链锁定"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "桥接资产", fmt.Sprintf("发起 %s 跨链桥接。", model.Asset), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出桥接流程、锁定资产和目标链铸造状态。
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

// SyncSharedState 在区块链完整性组共享状态变化后重建跨链桥接场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedBridgeState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// bridgeModel 保存双链资产锁定、证明和铸造状态。
type bridgeModel struct {
	Asset         string `json:"asset"`
	Amount        int    `json:"amount"`
	SourceBalance int    `json:"source_balance"`
	TargetBalance int    `json:"target_balance"`
	LockedAmount  int    `json:"locked_amount"`
	ProofHash     string `json:"proof_hash"`
	RelayAccepted bool   `json:"relay_accepted"`
	MintedAmount  int    `json:"minted_amount"`
	BridgeStatus  string `json:"bridge_status"`
	RelayMessage  string `json:"relay_message"`
}

// rebuildState 将桥接模型映射为节点、消息和内嵌指标。
func rebuildState(state *framework.SceneState, model bridgeModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = "active"
	state.Nodes[0].Load = float64(model.SourceBalance)
	state.Nodes[1].Status = vaultStatus(phase)
	state.Nodes[1].Load = float64(model.LockedAmount)
	state.Nodes[2].Status = relayStatus(model)
	state.Nodes[2].Load = float64(model.Amount)
	state.Nodes[3].Status = targetStatus(phase)
	state.Nodes[3].Load = float64(model.TargetBalance)
	state.Messages = []framework.Message{
		{ID: "lock", Label: fmt.Sprintf("%s:%d", model.Asset, model.Amount), Kind: "transaction", Status: phase, SourceID: "source", TargetID: "vault"},
		{ID: "proof", Label: framework.AbbreviateOr(model.ProofHash, 14, "未生成"), Kind: "transaction", Status: phase, SourceID: "vault", TargetID: "relay"},
		{ID: "mint", Label: fmt.Sprintf("mint:%d", model.MintedAmount), Kind: "transaction", Status: phase, SourceID: "relay", TargetID: "target"},
	}
	state.Metrics = []framework.Metric{
		{Key: "asset", Label: "桥接资产", Value: model.Asset, Tone: "info"},
		{Key: "locked", Label: "锁定数量", Value: fmt.Sprintf("%d", model.LockedAmount), Tone: "warning"},
		{Key: "proof", Label: "证明摘要", Value: framework.AbbreviateOr(model.ProofHash, 14, "未生成"), Tone: "info"},
		{Key: "minted", Label: "目标链铸造", Value: fmt.Sprintf("%d", model.MintedAmount), Tone: settlementTone(model.BridgeStatus)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "源链余额", Value: fmt.Sprintf("%d", model.SourceBalance)},
		{Label: "目标链余额", Value: fmt.Sprintf("%d", model.TargetBalance)},
		{Label: "中继消息", Value: model.RelayMessage},
		{Label: "桥接状态", Value: model.BridgeStatus},
	}
	state.Data = map[string]any{
		"phase_name":         phase,
		"cross_chain_bridge": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟源链锁定、证明生成、中继提交和目标链铸造的双链桥接流程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复桥接模型。
func decodeModel(state *framework.SceneState) bridgeModel {
	entry, ok := state.Data["cross_chain_bridge"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["cross_chain_bridge"].(bridgeModel); ok {
			return typed
		}
		return bridgeModel{
			Asset:         "USDT",
			Amount:        100,
			SourceBalance: 1000,
			TargetBalance: 300,
			BridgeStatus:  "idle",
		}
	}
	return bridgeModel{
		Asset:         framework.StringValue(entry["asset"], "USDT"),
		Amount:        int(framework.NumberValue(entry["amount"], 100)),
		SourceBalance: int(framework.NumberValue(entry["source_balance"], 1000)),
		TargetBalance: int(framework.NumberValue(entry["target_balance"], 300)),
		LockedAmount:  int(framework.NumberValue(entry["locked_amount"], 0)),
		ProofHash:     framework.StringValue(entry["proof_hash"], ""),
		RelayAccepted: framework.BoolValue(entry["relay_accepted"], false),
		MintedAmount:  int(framework.NumberValue(entry["minted_amount"], 0)),
		BridgeStatus:  framework.StringValue(entry["bridge_status"], "idle"),
		RelayMessage:  framework.StringValue(entry["relay_message"], "pending"),
	}
}

// applySharedBridgeState 将共享区块链结构信息映射到跨链桥接模型。
func applySharedBridgeState(model *bridgeModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if blockchain, ok := sharedState["blockchain"].(map[string]any); ok {
		if merkleRoot, ok := blockchain["merkle_root"].(string); ok && strings.TrimSpace(merkleRoot) != "" {
			model.ProofHash = merkleRoot
		}
	}
}

// buildProofHash 构造桥接证明摘要。
func buildProofHash(asset string, amount int, sourceBalance int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", asset, amount, sourceBalance)))
	return hex.EncodeToString(sum[:])
}

// inferAmount 根据资产名称生成教学用默认桥接数量。
func inferAmount(asset string) int {
	switch strings.ToUpper(strings.TrimSpace(asset)) {
	case "ETH":
		return 2
	case "BTC":
		return 1
	default:
		return 100
	}
}

// nextPhase 返回跨链桥接的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "源链锁定":
		return "证明生成"
	case "证明生成":
		return "中继提交"
	case "中继提交":
		return "目标链铸造"
	default:
		return "源链锁定"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "源链锁定":
		return 0
	case "证明生成":
		return 1
	case "中继提交":
		return 2
	case "目标链铸造":
		return 3
	default:
		return 0
	}
}

// toneByBridgePhase 返回桥接阶段的事件色调。
func toneByBridgePhase(phase string) string {
	if phase == "目标链铸造" {
		return "success"
	}
	if phase == "中继提交" {
		return "warning"
	}
	return "info"
}

// vaultStatus 返回锁定金库节点状态。
func vaultStatus(phase string) string {
	if phase == "证明生成" || phase == "中继提交" {
		return "warning"
	}
	if phase == "目标链铸造" {
		return "success"
	}
	return "normal"
}

// relayStatus 返回中继节点状态。
func relayStatus(model bridgeModel) string {
	if model.RelayAccepted {
		return "warning"
	}
	return "normal"
}

// targetStatus 返回目标链节点状态。
func targetStatus(phase string) string {
	if phase == "目标链铸造" {
		return "success"
	}
	return "normal"
}

// settlementTone 返回目标链铸造指标色调。
func settlementTone(status string) string {
	if status == "minted" {
		return "success"
	}
	return "info"
}
