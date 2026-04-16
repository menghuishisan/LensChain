package networkpartition

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造网络分区与恢复场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "network-partition",
		Title:        "网络分区与恢复",
		Phase:        "正常通信",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1600,
		TotalTicks:   20,
		Stages:       []string{"正常通信", "形成分区", "隔离维持", "恢复同步"},
		Nodes: []framework.Node{
			{ID: "zone-a1", Label: "Zone-A1", Status: "normal", Role: "router", X: 150, Y: 170},
			{ID: "zone-a2", Label: "Zone-A2", Status: "normal", Role: "router", X: 290, Y: 120},
			{ID: "zone-b1", Label: "Zone-B1", Status: "normal", Role: "router", X: 470, Y: 120},
			{ID: "zone-b2", Label: "Zone-B2", Status: "normal", Role: "router", X: 610, Y: 170},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化全连通网络与默认链路状态。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := partitionModel{
		LinkAtoB:     true,
		Partitioned:  false,
		SyncedHeight: 12,
		PendingDiff:  0,
	}
	applySharedNetworkState(&model, input.SharedState, state.LinkGroup)
	return rebuildState(state, model, "正常通信")
}

// Step 推进形成分区、隔离维持和恢复同步。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedNetworkState(&model, input.SharedState, state.LinkGroup)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "正常通信"))
	switch phase {
	case "形成分区":
		model.LinkAtoB = false
		model.Partitioned = true
		model.PendingDiff = 2
	case "隔离维持":
		model.PendingDiff += 2
	case "恢复同步":
		model.LinkAtoB = true
		model.Partitioned = false
		model.SyncedHeight += model.PendingDiff
		model.PendingDiff = 0
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("网络状态进入%s阶段。", phase), phaseTone(phase))
	return framework.StepOutput{
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// HandleAction 处理链路切断和恢复操作。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	phase := "形成分区"
	if input.ActionCode == "restore_link" {
		model.LinkAtoB = true
		model.Partitioned = false
		model.SyncedHeight += model.PendingDiff
		model.PendingDiff = 0
		phase = "恢复同步"
	} else {
		model.LinkAtoB = false
		model.Partitioned = true
		model.PendingDiff += 2
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, input.ActionCode, fmt.Sprintf("已执行 %s。", input.ActionCode), phaseTone(phase))
	return framework.ActionOutput{
		Success:    true,
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// BuildRenderState 输出隔离区域、链路状态和恢复同步结果。
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

// SyncSharedState 在共享网络状态变化后重建网络分区场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedNetworkState(&model, sharedState, state.LinkGroup)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// partitionModel 保存链路、分区和同步差量状态。
type partitionModel struct {
	LinkAtoB     bool `json:"link_a_b"`
	Partitioned  bool `json:"partitioned"`
	SyncedHeight int  `json:"synced_height"`
	PendingDiff  int  `json:"pending_diff"`
}

// rebuildState 将分区模型转为节点、链路消息和指标。
func rebuildState(state *framework.SceneState, model partitionModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		if model.Partitioned && strings.HasPrefix(state.Nodes[index].ID, "zone-a") {
			state.Nodes[index].Status = "isolated-a"
		}
		if model.Partitioned && strings.HasPrefix(state.Nodes[index].ID, "zone-b") {
			state.Nodes[index].Status = "isolated-b"
		}
		if !model.Partitioned && phase == "恢复同步" {
			state.Nodes[index].Status = "success"
		}
	}
	state.Messages = []framework.Message{
		{ID: "link-a", Label: linkLabel(model.LinkAtoB), Kind: "packet", Status: phase, SourceID: "zone-a2", TargetID: "zone-b1"},
	}
	state.Metrics = []framework.Metric{
		{Key: "partitioned", Label: "是否分区", Value: framework.BoolText(model.Partitioned), Tone: phaseTone(phase)},
		{Key: "link", Label: "跨区链路", Value: linkLabel(model.LinkAtoB), Tone: phaseTone(phase)},
		{Key: "pending", Label: "待同步差量", Value: fmt.Sprintf("%d", model.PendingDiff), Tone: "warning"},
		{Key: "height", Label: "同步高度", Value: fmt.Sprintf("%d", model.SyncedHeight), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "隔离边界", Value: "Zone-A / Zone-B"},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"partition":  model,
	}
	state.Extra = map[string]any{
		"description": "该场景实现跨区链路切断、隔离维持和恢复后的同步回放。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复分区模型。
func decodeModel(state *framework.SceneState) partitionModel {
	entry, ok := state.Data["partition"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["partition"].(partitionModel); ok {
			return typed
		}
		return partitionModel{LinkAtoB: true, Partitioned: false, SyncedHeight: 12}
	}
	return partitionModel{
		LinkAtoB:     framework.BoolValue(entry["link_a_b"], true),
		Partitioned:  framework.BoolValue(entry["partitioned"], false),
		SyncedHeight: int(framework.NumberValue(entry["synced_height"], 12)),
		PendingDiff:  int(framework.NumberValue(entry["pending_diff"], 0)),
	}
}

// nextPhase 返回网络分区流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "正常通信":
		return "形成分区"
	case "形成分区":
		return "隔离维持"
	case "隔离维持":
		return "恢复同步"
	default:
		return "正常通信"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "正常通信":
		return 0
	case "形成分区":
		return 1
	case "隔离维持":
		return 2
	case "恢复同步":
		return 3
	default:
		return 0
	}
}

// linkLabel 返回跨区链路展示文案。
func linkLabel(open bool) string {
	if open {
		return "connected"
	}
	return "cut"
}

// phaseTone 返回网络状态阶段色调。
func phaseTone(phase string) string {
	if phase == "形成分区" || phase == "隔离维持" {
		return "warning"
	}
	return "info"
}

// buildSharedDiff 按联动组输出网络分区场景的共享状态。
func buildSharedDiff(linkGroup string, model partitionModel) map[string]any {
	switch linkGroup {
	case "pbft-attack-group":
		return map[string]any{
			"nodes": map[string]any{
				"isolated": pbftIsolatedNodes(model),
			},
			"messages": []map[string]any{
				{"id": "partition-link", "status": linkLabel(model.LinkAtoB), "pending": model.PendingDiff},
			},
			"view": model.PendingDiff,
		}
	case "raft-fault-group":
		return map[string]any{
			"nodes": map[string]any{
				"leader":    ternary(model.Partitioned, "isolated", "healthy"),
				"followers": ternary(model.Partitioned, "split", "replicated"),
			},
			"terms": map[string]any{
				"leader_failed": model.Partitioned,
				"current_term":  model.SyncedHeight,
				"leader":        "node-a",
			},
			"logs": map[string]any{
				"pending_diff": model.PendingDiff,
			},
		}
	default:
		return nil
	}
}

// pbftIsolatedNodes 构造 PBFT 攻击组共享的隔离副本列表。
func pbftIsolatedNodes(model partitionModel) []string {
	if !model.Partitioned {
		return []string{}
	}
	return []string{"replica-3"}
}

// ternary 返回简化的字符串分支结果。
func ternary(flag bool, left string, right string) string {
	if flag {
		return left
	}
	return right
}

// applySharedNetworkState 根据联动组共享状态更新网络分区模型。
func applySharedNetworkState(model *partitionModel, sharedState map[string]any, linkGroup string) {
	if len(sharedState) == 0 {
		return
	}
	switch linkGroup {
	case "pbft-attack-group":
		if messages, ok := sharedState["messages"].([]any); ok && len(messages) > 0 {
			if len(messages) > model.PendingDiff {
				model.PendingDiff = len(messages)
			}
		}
		if sharedView := int(framework.NumberValue(sharedState["view"], 0)); sharedView > 0 {
			model.PendingDiff = sharedView
			model.Partitioned = true
			model.LinkAtoB = false
		}
	case "raft-fault-group":
		if terms, ok := sharedState["terms"].(map[string]any); ok {
			if framework.BoolValue(terms["leader_failed"], false) {
				model.Partitioned = true
				model.LinkAtoB = false
			}
			if synced := int(framework.NumberValue(terms["current_term"], float64(model.SyncedHeight))); synced > 0 {
				model.SyncedHeight = synced
			}
		}
		if logs, ok := sharedState["logs"].(map[string]any); ok {
			model.PendingDiff = int(framework.NumberValue(logs["pending_diff"], float64(model.PendingDiff)))
		}
	}
}
