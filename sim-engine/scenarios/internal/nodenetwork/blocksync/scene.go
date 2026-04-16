package blocksync

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造区块同步与传播场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "block-sync",
		Title:        "区块同步与传播",
		Phase:        "出块",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   20,
		Stages:       []string{"出块", "广播", "高度追平", "分叉检测"},
		Nodes: []framework.Node{
			{ID: "producer", Label: "Producer", Status: "normal", Role: "validator", X: 110, Y: 200},
			{ID: "follower-1", Label: "Follower-1", Status: "normal", Role: "validator", X: 300, Y: 100},
			{ID: "follower-2", Label: "Follower-2", Status: "normal", Role: "validator", X: 300, Y: 300},
			{ID: "follower-3", Label: "Follower-3", Status: "normal", Role: "validator", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化新区块高度、各节点高度和分叉标记。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := syncModel{
		BlockHeight:    10,
		NodeHeights:    map[string]int{"producer": 9, "follower-1": 9, "follower-2": 8, "follower-3": 8},
		ForkDetected:   false,
		BroadcastPeers: []string{},
	}
	applySharedSyncState(&model, input.SharedState, state.LinkGroup)
	return rebuildState(state, model, "出块")
}

// Step 推进新区块出块、广播、追平和分叉检测。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedSyncState(&model, input.SharedState, state.LinkGroup)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "出块"))
	switch phase {
	case "广播":
		model.NodeHeights["producer"] = model.BlockHeight
		model.BroadcastPeers = []string{"follower-1", "follower-2"}
	case "高度追平":
		model.NodeHeights["follower-1"] = model.BlockHeight
		model.NodeHeights["follower-2"] = model.BlockHeight
		model.NodeHeights["follower-3"] = model.BlockHeight - 1
	case "分叉检测":
		model.ForkDetected = model.NodeHeights["follower-3"] != model.BlockHeight
	default:
		model.BlockHeight++
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("区块同步进入%s阶段。", phase), toneByPhase(phase, model.ForkDetected))
	return framework.StepOutput{
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// HandleAction 注入更高新区块并重新触发同步。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.BlockHeight = int(framework.NumberValue(input.Params["block_height"], float64(model.BlockHeight+1)))
	model.ForkDetected = false
	model.BroadcastPeers = []string{}
	if err := rebuildState(state, model, "出块"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "注入新区块", fmt.Sprintf("已注入高度 %d 的新区块。", model.BlockHeight), "info")
	return framework.ActionOutput{
		Success:    true,
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// BuildRenderState 输出各节点高度、广播路径和分叉状态。
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

// SyncSharedState 在链高度或网络联动变化后重建区块同步场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedSyncState(&model, sharedState, state.LinkGroup)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// syncModel 保存区块同步中的链高度、广播目标和分叉状态。
type syncModel struct {
	BlockHeight    int            `json:"block_height"`
	NodeHeights    map[string]int `json:"node_heights"`
	ForkDetected   bool           `json:"fork_detected"`
	BroadcastPeers []string       `json:"broadcast_peers"`
}

// rebuildState 将同步模型转换为节点负载、消息路径和指标。
func rebuildState(state *framework.SceneState, model syncModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		nodeID := state.Nodes[index].ID
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = float64(model.NodeHeights[nodeID])
	}
	state.Nodes[0].Status = "active"
	if model.ForkDetected {
		state.Nodes[3].Status = "warning"
	}
	state.Messages = []framework.Message{
		{ID: "producer-f1", Label: fmt.Sprintf("Block-%d", model.BlockHeight), Kind: "packet", Status: phase, SourceID: "producer", TargetID: "follower-1"},
		{ID: "producer-f2", Label: fmt.Sprintf("Block-%d", model.BlockHeight), Kind: "packet", Status: phase, SourceID: "producer", TargetID: "follower-2"},
		{ID: "sync-f3", Label: fmt.Sprintf("Height-%d", model.BlockHeight), Kind: "packet", Status: phase, SourceID: "follower-2", TargetID: "follower-3"},
	}
	state.Metrics = []framework.Metric{
		{Key: "height", Label: "新区块高度", Value: fmt.Sprintf("%d", model.BlockHeight), Tone: "info"},
		{Key: "broadcast", Label: "已广播节点", Value: strings.Join(model.BroadcastPeers, ", "), Tone: "warning"},
		{Key: "fork", Label: "发现分叉", Value: framework.BoolText(model.ForkDetected), Tone: toneByPhase(phase, model.ForkDetected)},
		{Key: "lag", Label: "最大落后高度", Value: fmt.Sprintf("%d", maxLag(model)), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "节点高度", Value: formatHeights(model.NodeHeights)},
		{Label: "分叉状态", Value: framework.BoolText(model.ForkDetected)},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"block_sync": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟新区块出块后在网络中的传播、节点追平和分叉检测。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复区块同步模型。
func decodeModel(state *framework.SceneState) syncModel {
	entry, ok := state.Data["block_sync"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["block_sync"].(syncModel); ok {
			return typed
		}
		return syncModel{
			BlockHeight:  10,
			NodeHeights:  map[string]int{"producer": 9, "follower-1": 9, "follower-2": 8, "follower-3": 8},
			ForkDetected: false,
		}
	}
	return syncModel{
		BlockHeight:    int(framework.NumberValue(entry["block_height"], 10)),
		NodeHeights:    framework.ToIntMapOr(entry["node_heights"], map[string]int{"producer": 9, "follower-1": 9, "follower-2": 8, "follower-3": 8}),
		ForkDetected:   framework.BoolValue(entry["fork_detected"], false),
		BroadcastPeers: framework.ToStringSlice(entry["broadcast_peers"]),
	}
}

// nextPhase 返回区块同步流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "出块":
		return "广播"
	case "广播":
		return "高度追平"
	case "高度追平":
		return "分叉检测"
	default:
		return "出块"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "出块":
		return 0
	case "广播":
		return 1
	case "高度追平":
		return 2
	case "分叉检测":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回区块同步阶段的色调。
func toneByPhase(phase string, fork bool) string {
	if fork {
		return "warning"
	}
	if phase == "高度追平" {
		return "success"
	}
	return "info"
}

// maxLag 计算节点相对于最新高度的最大落后值。
func maxLag(model syncModel) int {
	lag := 0
	for _, height := range model.NodeHeights {
		if diff := model.BlockHeight - height; diff > lag {
			lag = diff
		}
	}
	return lag
}

// formatHeights 将各节点高度拼接为简洁文案。
func formatHeights(values map[string]int) string {
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, fmt.Sprintf("%s=%d", key, value))
	}
	return strings.Join(parts, ", ")
}

// buildSharedDiff 按联动组输出区块同步场景的共享键。
func buildSharedDiff(linkGroup string, model syncModel) map[string]any {
	switch linkGroup {
	case "pow-attack-group":
		return map[string]any{
			"blockchain": map[string]any{
				"height":        model.BlockHeight,
				"node_heights":  model.NodeHeights,
				"fork_detected": model.ForkDetected,
			},
			"network": map[string]any{
				"broadcast_peers": model.BroadcastPeers,
			},
		}
	case "raft-fault-group":
		return map[string]any{
			"nodes": raftNodes(model),
			"terms": map[string]any{
				"current_term":  model.BlockHeight,
				"leader":        "producer",
				"leader_failed": model.ForkDetected,
			},
			"logs": map[string]any{
				"commit_index":  model.BlockHeight - 1,
				"replica_index": model.NodeHeights,
			},
		}
	default:
		return nil
	}
}

// raftNodes 构造 Raft 容错组所需的节点视图。
func raftNodes(model syncModel) []map[string]any {
	result := make([]map[string]any, 0, len(model.NodeHeights))
	for nodeID, height := range model.NodeHeights {
		status := "follower"
		if nodeID == "producer" {
			status = "leader"
		}
		if model.ForkDetected && nodeID == "follower-3" {
			status = "lagging"
		}
		result = append(result, map[string]any{
			"id":     nodeID,
			"status": status,
			"height": height,
		})
	}
	return result
}

// applySharedSyncState 将当前联动组中的共享状态映射回区块同步模型。
func applySharedSyncState(model *syncModel, sharedState map[string]any, linkGroup string) {
	if len(sharedState) == 0 {
		return
	}
	switch linkGroup {
	case "pow-attack-group":
		if blockchain, ok := sharedState["blockchain"].(map[string]any); ok {
			if linkedHeight := int(framework.NumberValue(blockchain["height"], float64(model.BlockHeight))); linkedHeight > 0 {
				model.BlockHeight = linkedHeight
			}
			if nodeHeights := framework.ToIntMapOr(blockchain["node_heights"], map[string]int{"producer": 9, "follower-1": 9, "follower-2": 8, "follower-3": 8}); len(nodeHeights) > 0 {
				model.NodeHeights = nodeHeights
			}
			model.ForkDetected = framework.BoolValue(blockchain["fork_detected"], model.ForkDetected)
		}
		if network, ok := sharedState["network"].(map[string]any); ok {
			if peers := framework.ToStringSlice(network["broadcast_peers"]); len(peers) > 0 {
				model.BroadcastPeers = peers
			}
			if framework.BoolValue(network["partitioned"], false) {
				model.ForkDetected = true
			}
		}
	case "raft-fault-group":
		if terms, ok := sharedState["terms"].(map[string]any); ok {
			if linkedTerm := int(framework.NumberValue(terms["current_term"], float64(model.BlockHeight))); linkedTerm > 0 {
				model.BlockHeight = linkedTerm
			}
			model.ForkDetected = framework.BoolValue(terms["leader_failed"], model.ForkDetected)
		}
		if logs, ok := sharedState["logs"].(map[string]any); ok {
			if replica := framework.ToIntMapOr(logs["replica_index"], map[string]int{"producer": 9, "follower-1": 9, "follower-2": 8, "follower-3": 8}); len(replica) > 0 {
				model.NodeHeights = replica
			}
		}
	}
}
