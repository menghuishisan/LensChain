package txbroadcast

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造交易广播与打包场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "tx-broadcast",
		Title:        "交易广播与打包",
		Phase:        "创建交易",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   16,
		Stages:       []string{"创建交易", "网络广播", "内存池堆积", "矿工收集"},
		Nodes: []framework.Node{
			{ID: "wallet", Label: "Wallet", Status: "normal", Role: "node", X: 100, Y: 200},
			{ID: "peer-1", Label: "Peer-1", Status: "normal", Role: "node", X: 260, Y: 100},
			{ID: "peer-2", Label: "Peer-2", Status: "normal", Role: "node", X: 260, Y: 300},
			{ID: "miner", Label: "Miner", Status: "normal", Role: "node", X: 500, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化交易、广播覆盖范围和矿工收集状态。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := broadcastModel{
		TxLabel:        framework.StringValue(input.Params["tx_label"], "tx-1001"),
		SeenNodes:      []string{"wallet"},
		MempoolSize:    0,
		MinerCollected: false,
	}
	return rebuildState(state, model, "创建交易")
}

// Step 推进交易从钱包创建到矿工收集的传播过程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "创建交易"))
	switch phase {
	case "网络广播":
		model.SeenNodes = uniqueStrings([]string{"wallet", "peer-1", "peer-2"})
	case "内存池堆积":
		model.SeenNodes = uniqueStrings([]string{"wallet", "peer-1", "peer-2", "miner"})
		model.MempoolSize = len(model.SeenNodes) - 1
	case "矿工收集":
		model.MinerCollected = true
		model.MempoolSize = 0
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("交易 %s 进入%s阶段。", model.TxLabel, phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 注入一笔新的交易并重新开始传播。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.TxLabel = framework.StringValue(input.Params["tx_label"], "tx-1001")
	model.SeenNodes = []string{"wallet"}
	model.MempoolSize = 0
	model.MinerCollected = false
	if err := rebuildState(state, model, "创建交易"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "发起交易", fmt.Sprintf("已发起新的广播交易 %s。", model.TxLabel), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出交易广播路径和内存池状态。
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

// SyncSharedState 在交易处理组共享交易池变化后重建交易广播场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedBroadcastState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// broadcastModel 保存广播交易在网络中的扩散和收集状态。
type broadcastModel struct {
	TxLabel        string   `json:"tx_label"`
	SeenNodes      []string `json:"seen_nodes"`
	MempoolSize    int      `json:"mempool_size"`
	MinerCollected bool     `json:"miner_collected"`
}

// rebuildState 将广播模型映射为节点状态、消息路径和指标。
func rebuildState(state *framework.SceneState, model broadcastModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
		if contains(model.SeenNodes, state.Nodes[index].ID) {
			state.Nodes[index].Status = "active"
			state.Nodes[index].Load = float64(len(model.SeenNodes))
		}
	}
	if model.MinerCollected {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "wallet-peer1", Label: model.TxLabel, Kind: "packet", Status: phase, SourceID: "wallet", TargetID: "peer-1"},
		{ID: "wallet-peer2", Label: model.TxLabel, Kind: "packet", Status: phase, SourceID: "wallet", TargetID: "peer-2"},
		{ID: "peers-miner", Label: model.TxLabel, Kind: "packet", Status: phase, SourceID: "peer-1", TargetID: "miner"},
	}
	state.Metrics = []framework.Metric{
		{Key: "seen", Label: "已覆盖节点", Value: fmt.Sprintf("%d", len(model.SeenNodes)), Tone: "info"},
		{Key: "mempool", Label: "内存池交易数", Value: fmt.Sprintf("%d", model.MempoolSize), Tone: "warning"},
		{Key: "miner", Label: "矿工已收集", Value: framework.BoolText(model.MinerCollected), Tone: toneByPhase(phase)},
		{Key: "tx", Label: "交易标识", Value: model.TxLabel, Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "传播节点", Value: strings.Join(model.SeenNodes, ", ")},
		{Label: "矿工状态", Value: framework.BoolText(model.MinerCollected)},
	}
	state.Data = map[string]any{
		"phase_name":   phase,
		"tx_broadcast": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟交易从钱包广播到对等节点，再进入内存池并被矿工收集的过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复交易广播模型。
func decodeModel(state *framework.SceneState) broadcastModel {
	entry, ok := state.Data["tx_broadcast"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["tx_broadcast"].(broadcastModel); ok {
			return typed
		}
		return broadcastModel{TxLabel: "tx-1001", SeenNodes: []string{"wallet"}}
	}
	return broadcastModel{
		TxLabel:        framework.StringValue(entry["tx_label"], "tx-1001"),
		SeenNodes:      framework.ToStringSlice(entry["seen_nodes"]),
		MempoolSize:    int(framework.NumberValue(entry["mempool_size"], 0)),
		MinerCollected: framework.BoolValue(entry["miner_collected"], false),
	}
}

// applySharedBroadcastState 将交易处理组共享交易池状态映射到广播场景。
func applySharedBroadcastState(model *broadcastModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if mempool, ok := sharedState["mempool"].(map[string]any); ok {
		if ordered, ok := mempool["ordered"].([]any); ok {
			model.MempoolSize = len(ordered)
		}
		if included, ok := mempool["included"].([]any); ok && len(included) > 0 {
			model.MinerCollected = true
		}
	}
}

// nextPhase 返回交易广播流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "创建交易":
		return "网络广播"
	case "网络广播":
		return "内存池堆积"
	case "内存池堆积":
		return "矿工收集"
	default:
		return "创建交易"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "创建交易":
		return 0
	case "网络广播":
		return 1
	case "内存池堆积":
		return 2
	case "矿工收集":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回交易广播阶段的色调。
func toneByPhase(phase string) string {
	switch phase {
	case "矿工收集":
		return "success"
	case "内存池堆积":
		return "warning"
	default:
		return "info"
	}
}

// contains 判断切片中是否包含指定值。
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// uniqueStrings 对字符串切片去重。
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
