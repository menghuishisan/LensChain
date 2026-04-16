package statechannel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造状态通道场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "state-channel",
		Title:        "状态通道",
		Phase:        "开通通道",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   16,
		Stages:       []string{"开通通道", "链下更新", "争议提交", "通道关闭"},
		Nodes: []framework.Node{
			{ID: "party-a", Label: "Party-A", Status: "active", Role: "channel", X: 110, Y: 200},
			{ID: "channel", Label: "Channel", Status: "normal", Role: "channel", X: 300, Y: 110},
			{ID: "party-b", Label: "Party-B", Status: "normal", Role: "channel", X: 300, Y: 310},
			{ID: "adjudicator", Label: "Adjudicator", Status: "normal", Role: "channel", X: 530, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化通道资金、最新状态号和关闭状态。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := channelModel{
		ChannelID:       "channel-001",
		PartyABalance:   40,
		PartyBBalance:   60,
		LockedTotal:     100,
		Version:         0,
		LatestStateHash: buildStateHash(40, 60, 0),
		DisputeProof:    "",
		DisputeOpen:     false,
		ChallengeWindow: 3,
		LatestSigner:    "Party-A|Party-B",
	}
	return rebuildState(state, model, "开通通道")
}

// Step 推进开通、链下更新、争议和关闭流程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "开通通道"))
	switch phase {
	case "链下更新":
		model.Version++
		model.PartyABalance -= 5
		model.PartyBBalance += 5
		model.LatestSigner = fmt.Sprintf("Party-A|Party-B@v%d", model.Version)
		model.LatestStateHash = buildStateHash(model.PartyABalance, model.PartyBBalance, model.Version)
	case "争议提交":
		model.DisputeProof = buildProof(model.LatestStateHash, model.Version)
		model.DisputeOpen = true
		if model.ChallengeWindow > 0 {
			model.ChallengeWindow--
		}
	case "通道关闭":
		model.DisputeOpen = false
		model.ChallengeWindow = 0
		model.SettlementResult = fmt.Sprintf("A:%d B:%d", model.PartyABalance, model.PartyBBalance)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("状态通道进入%s阶段。", phase), toneByChannelPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"address": model.ChannelID,
				"storage": map[string]any{
					"party_a_balance":  model.PartyABalance,
					"party_b_balance":  model.PartyBBalance,
					"version":          model.Version,
					"state_hash":       model.LatestStateHash,
					"challenge_window": model.ChallengeWindow,
				},
			},
			"call_stack": map[string]any{
				"type":   "state-channel",
				"stack":  []string{"Party-A", "Channel", "Party-B"},
				"target": "Adjudicator",
			},
		},
	}, nil
}

// HandleAction 提交争议证明并打开争议窗口。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.DisputeProof = framework.StringValue(input.Params["proof"], "")
	if model.DisputeProof == "" {
		model.DisputeProof = buildProof(model.LatestStateHash, model.Version)
	}
	model.DisputeOpen = true
	model.ChallengeWindow = maxInt(model.ChallengeWindow-1, 0)
	model.LatestSigner = "dispute-submitted"
	if err := rebuildState(state, model, "争议提交"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "提交争议", fmt.Sprintf("已提交争议证明 %s。", framework.Abbreviate(model.DisputeProof, 14)), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"address": model.ChannelID,
				"storage": map[string]any{
					"party_a_balance":  model.PartyABalance,
					"party_b_balance":  model.PartyBBalance,
					"version":          model.Version,
					"state_hash":       model.LatestStateHash,
					"challenge_window": model.ChallengeWindow,
				},
			},
			"call_stack": map[string]any{
				"type":   "dispute",
				"stack":  []string{"Party-A", "Channel", "Adjudicator"},
				"target": "Adjudicator",
			},
		},
	}, nil
}

// BuildRenderState 输出状态通道、争议证明和结算结果。
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

// SyncSharedState 在合约安全组共享合约状态变化后重建状态通道场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedChannelState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// channelModel 保存状态通道资金、版本和争议状态。
type channelModel struct {
	ChannelID        string `json:"channel_id"`
	PartyABalance    int    `json:"party_a_balance"`
	PartyBBalance    int    `json:"party_b_balance"`
	LockedTotal      int    `json:"locked_total"`
	Version          int    `json:"version"`
	LatestStateHash  string `json:"latest_state_hash"`
	DisputeProof     string `json:"dispute_proof"`
	DisputeOpen      bool   `json:"dispute_open"`
	SettlementResult string `json:"settlement_result"`
	ChallengeWindow  int    `json:"challenge_window"`
	LatestSigner     string `json:"latest_signer"`
}

// rebuildState 将状态通道模型映射为节点、消息和指标。
func rebuildState(state *framework.SceneState, model channelModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = "active"
	state.Nodes[0].Load = float64(model.PartyABalance)
	state.Nodes[1].Status = channelStatus(phase)
	state.Nodes[1].Load = float64(model.LockedTotal)
	state.Nodes[2].Status = "active"
	state.Nodes[2].Load = float64(model.PartyBBalance)
	state.Nodes[3].Status = adjudicatorStatus(model, phase)
	state.Nodes[3].Load = float64(model.Version * 20)
	state.Messages = []framework.Message{
		{ID: "funding", Label: fmt.Sprintf("fund:%d", model.LockedTotal), Kind: "call", Status: phase, SourceID: "party-a", TargetID: "channel"},
		{ID: "state-update", Label: fmt.Sprintf("v%d", model.Version), Kind: "call", Status: phase, SourceID: "party-b", TargetID: "channel"},
		{ID: "proof", Label: framework.Abbreviate(model.DisputeProof, 14), Kind: "call", Status: phase, SourceID: "channel", TargetID: "adjudicator"},
	}
	state.Metrics = []framework.Metric{
		{Key: "version", Label: "最新状态号", Value: fmt.Sprintf("%d", model.Version), Tone: "info"},
		{Key: "hash", Label: "状态哈希", Value: framework.Abbreviate(model.LatestStateHash, 14), Tone: "warning"},
		{Key: "window", Label: "争议窗口", Value: fmt.Sprintf("%d", model.ChallengeWindow), Tone: disputeTone(model.DisputeOpen)},
		{Key: "dispute", Label: "争议开启", Value: framework.BoolText(model.DisputeOpen), Tone: disputeTone(model.DisputeOpen)},
		{Key: "settlement", Label: "结算结果", Value: model.SettlementResult, Tone: settlementTone(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "通道编号", Value: model.ChannelID},
		{Label: "争议证明", Value: emptyText(model.DisputeProof, "未提交")},
		{Label: "最近签名", Value: model.LatestSigner},
		{Label: "资金分布", Value: fmt.Sprintf("A:%d / B:%d", model.PartyABalance, model.PartyBBalance)},
	}
	state.Data = map[string]any{
		"phase_name":    phase,
		"state_channel": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟链上开通、链下状态更新、争议提交与最终结算关闭。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复状态通道模型。
func decodeModel(state *framework.SceneState) channelModel {
	entry, ok := state.Data["state_channel"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["state_channel"].(channelModel); ok {
			return typed
		}
		return channelModel{
			ChannelID:        "channel-001",
			PartyABalance:    40,
			PartyBBalance:    60,
			LockedTotal:      100,
			Version:          0,
			LatestStateHash:  buildStateHash(40, 60, 0),
			SettlementResult: "pending",
			ChallengeWindow:  3,
			LatestSigner:     "Party-A|Party-B",
		}
	}
	return channelModel{
		ChannelID:        framework.StringValue(entry["channel_id"], "channel-001"),
		PartyABalance:    int(framework.NumberValue(entry["party_a_balance"], 40)),
		PartyBBalance:    int(framework.NumberValue(entry["party_b_balance"], 60)),
		LockedTotal:      int(framework.NumberValue(entry["locked_total"], 100)),
		Version:          int(framework.NumberValue(entry["version"], 0)),
		LatestStateHash:  framework.StringValue(entry["latest_state_hash"], ""),
		DisputeProof:     framework.StringValue(entry["dispute_proof"], ""),
		DisputeOpen:      framework.BoolValue(entry["dispute_open"], false),
		SettlementResult: framework.StringValue(entry["settlement_result"], "pending"),
		ChallengeWindow:  int(framework.NumberValue(entry["challenge_window"], 3)),
		LatestSigner:     framework.StringValue(entry["latest_signer"], "Party-A|Party-B"),
	}
}

// applySharedChannelState 将共享合约状态映射回状态通道模型。
func applySharedChannelState(model *channelModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		if storage, ok := contractState["storage"].(map[string]any); ok {
			if partyA, ok := storage["party_a_balance"]; ok {
				model.PartyABalance = int(framework.NumberValue(partyA, float64(model.PartyABalance)))
			}
			if partyB, ok := storage["party_b_balance"]; ok {
				model.PartyBBalance = int(framework.NumberValue(partyB, float64(model.PartyBBalance)))
			}
			if version, ok := storage["version"]; ok {
				model.Version = int(framework.NumberValue(version, float64(model.Version)))
			}
			if stateHash, ok := storage["state_hash"].(string); ok && strings.TrimSpace(stateHash) != "" {
				model.LatestStateHash = stateHash
			}
			if challengeWindow, ok := storage["challenge_window"]; ok {
				model.ChallengeWindow = int(framework.NumberValue(challengeWindow, float64(model.ChallengeWindow)))
			}
		}
		if address, ok := contractState["address"].(string); ok && strings.TrimSpace(address) != "" {
			model.ChannelID = address
		}
	}
}

// buildStateHash 为链下状态构造摘要。
func buildStateHash(balanceA int, balanceB int, version int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%d|%d", balanceA, balanceB, version)))
	return hex.EncodeToString(sum[:])
}

// buildProof 构造争议提交用的简化证明。
func buildProof(stateHash string, version int) string {
	sum := sha256.Sum256([]byte(stateHash + fmt.Sprintf("|proof|%d", version)))
	return hex.EncodeToString(sum[:])
}

// nextPhase 返回状态通道的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "开通通道":
		return "链下更新"
	case "链下更新":
		return "争议提交"
	case "争议提交":
		return "通道关闭"
	default:
		return "开通通道"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "开通通道":
		return 0
	case "链下更新":
		return 1
	case "争议提交":
		return 2
	case "通道关闭":
		return 3
	default:
		return 0
	}
}

// channelStatus 返回通道节点状态。
func channelStatus(phase string) string {
	if phase == "通道关闭" {
		return "success"
	}
	if phase == "争议提交" {
		return "warning"
	}
	return "active"
}

// adjudicatorStatus 返回仲裁节点状态。
func adjudicatorStatus(model channelModel, phase string) string {
	if phase == "争议提交" && model.DisputeOpen {
		return "warning"
	}
	if phase == "通道关闭" {
		return "success"
	}
	return "normal"
}

// toneByChannelPhase 返回阶段事件色调。
func toneByChannelPhase(phase string) string {
	if phase == "通道关闭" {
		return "success"
	}
	if phase == "争议提交" {
		return "warning"
	}
	return "info"
}

// disputeTone 返回争议状态指标色调。
func disputeTone(open bool) string {
	if open {
		return "warning"
	}
	return "info"
}

// settlementTone 返回结算指标色调。
func settlementTone(phase string) string {
	if phase == "通道关闭" {
		return "success"
	}
	return "info"
}

// emptyText 在字符串为空时回退默认文案。
func emptyText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// maxInt 返回两个整数中的较大值。
func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
