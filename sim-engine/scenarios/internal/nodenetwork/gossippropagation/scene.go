package gossippropagation

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 Gossip 消息传播场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "gossip-propagation",
		Title:        "Gossip 消息传播",
		Phase:        "源节点广播",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   15,
		Stages:       []string{"源节点广播", "邻居转发", "全网覆盖"},
		Nodes: []framework.Node{
			{ID: "seed", Label: "Seed", Status: "active", Role: "relay", X: 130, Y: 200},
			{ID: "relay-1", Label: "Relay-1", Status: "normal", Role: "relay", X: 300, Y: 100},
			{ID: "relay-2", Label: "Relay-2", Status: "normal", Role: "relay", X: 300, Y: 300},
			{ID: "relay-3", Label: "Relay-3", Status: "normal", Role: "relay", X: 500, Y: 120},
			{ID: "relay-4", Label: "Relay-4", Status: "normal", Role: "relay", X: 500, Y: 280},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化消息种子、拓扑图和已覆盖节点集合。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := propagationModel{
		MessageLabel: framework.StringValue(input.Params["message_label"], "TX-001"),
		SeenBy: map[string]bool{
			"seed": true,
		},
		Frontier: []string{"seed"},
		Topology: map[string][]string{
			"seed":    {"relay-1", "relay-2"},
			"relay-1": {"seed", "relay-3"},
			"relay-2": {"seed", "relay-4"},
			"relay-3": {"relay-1", "relay-4"},
			"relay-4": {"relay-2", "relay-3"},
		},
		Hops:         0,
		CoverageRate: 0.2,
	}
	return rebuildState(state, model, "源节点广播")
}

// Step 按 Gossip 扩散规则将 frontier 消息转发给相邻节点。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedPropagationState(&model, input.SharedState)
	newFrontier := make([]string, 0)
	for _, source := range model.Frontier {
		for _, target := range model.Topology[source] {
			if model.SeenBy[target] {
				continue
			}
			model.SeenBy[target] = true
			newFrontier = append(newFrontier, target)
		}
	}
	if len(newFrontier) == 0 {
		newFrontier = append(newFrontier, model.Frontier...)
	}
	model.Frontier = uniqueStrings(newFrontier)
	model.Hops++
	model.CoverageRate = float64(len(model.SeenBy)) / float64(len(model.Topology))
	phase := determinePhase(model)
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("消息 %s 已覆盖 %.0f%% 节点。", model.MessageLabel, model.CoverageRate*100), toneByCoverage(model.CoverageRate))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology": map[string]any{
				"coverage_rate": model.CoverageRate,
				"frontier":      model.Frontier,
			},
			"routing_table": map[string]any{
				"gossip_seen": len(model.SeenBy),
			},
			"load": map[string]any{
				"relay_hops": model.Hops,
			},
		},
	}, nil
}

// HandleAction 注入新消息并重新从源节点开始扩散。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.MessageLabel = framework.StringValue(input.Params["message_label"], "TX-001")
	model.SeenBy = map[string]bool{"seed": true}
	model.Frontier = []string{"seed"}
	model.Hops = 0
	model.CoverageRate = 1.0 / float64(len(model.Topology))
	if err := rebuildState(state, model, "源节点广播"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "投放消息", fmt.Sprintf("已从 Seed 注入消息 %s。", model.MessageLabel), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology": map[string]any{
				"message_label": model.MessageLabel,
				"frontier":      model.Frontier,
			},
		},
	}, nil
}

// BuildRenderState 输出当前覆盖节点、传播路径和覆盖率。
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

// SyncSharedState 在拓扑与传播前沿联动变化后重建 Gossip 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedPropagationState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// propagationModel 保存 Gossip 拓扑、已覆盖节点和传播前沿。
type propagationModel struct {
	MessageLabel string              `json:"message_label"`
	SeenBy       map[string]bool     `json:"seen_by"`
	Frontier     []string            `json:"frontier"`
	Topology     map[string][]string `json:"topology"`
	Hops         int                 `json:"hops"`
	CoverageRate float64             `json:"coverage_rate"`
}

// rebuildState 将传播模型映射到节点高亮、消息路径和指标。
func rebuildState(state *framework.SceneState, model propagationModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		nodeID := state.Nodes[index].ID
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
		if model.SeenBy[nodeID] {
			state.Nodes[index].Status = "success"
			state.Nodes[index].Load = float64(model.Hops + 1)
		}
		if contains(model.Frontier, nodeID) {
			state.Nodes[index].Status = "active"
		}
	}
	state.Messages = buildMessages(model, phase)
	state.Metrics = []framework.Metric{
		{Key: "coverage", Label: "覆盖率", Value: fmt.Sprintf("%.0f%%", model.CoverageRate*100), Tone: toneByCoverage(model.CoverageRate)},
		{Key: "frontier", Label: "当前前沿", Value: strings.Join(model.Frontier, ", "), Tone: "info"},
		{Key: "hops", Label: "传播跳数", Value: fmt.Sprintf("%d", model.Hops), Tone: "warning"},
		{Key: "seen", Label: "已接收节点", Value: fmt.Sprintf("%d/%d", len(model.SeenBy), len(model.Topology)), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "消息标签", Value: model.MessageLabel},
		{Label: "阶段", Value: phase},
		{Label: "源节点", Value: "seed"},
	}
	state.Data = map[string]any{
		"phase_name":         phase,
		"gossip_propagation": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 Gossip 前沿扩散、邻居转发和全网覆盖过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 Gossip 传播模型。
func decodeModel(state *framework.SceneState) propagationModel {
	entry, ok := state.Data["gossip_propagation"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["gossip_propagation"].(propagationModel); ok {
			return typed
		}
		return propagationModel{
			MessageLabel: "TX-001",
			SeenBy:       map[string]bool{"seed": true},
			Frontier:     []string{"seed"},
			Topology: map[string][]string{
				"seed":    {"relay-1", "relay-2"},
				"relay-1": {"seed", "relay-3"},
				"relay-2": {"seed", "relay-4"},
				"relay-3": {"relay-1", "relay-4"},
				"relay-4": {"relay-2", "relay-3"},
			},
			CoverageRate: 0.2,
		}
	}
	return propagationModel{
		MessageLabel: framework.StringValue(entry["message_label"], "TX-001"),
		SeenBy:       decodeSeenBy(entry["seen_by"]),
		Frontier:     framework.ToStringSlice(entry["frontier"]),
		Topology:     decodeTopology(entry["topology"]),
		Hops:         int(framework.NumberValue(entry["hops"], 0)),
		CoverageRate: framework.NumberValue(entry["coverage_rate"], 0.2),
	}
}

// buildMessages 为当前 frontier 构造传播路径。
func buildMessages(model propagationModel, phase string) []framework.Message {
	messages := make([]framework.Message, 0, len(model.Frontier))
	for _, source := range model.Frontier {
		for _, target := range model.Topology[source] {
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("%s-%s", source, target),
				Label:    model.MessageLabel,
				Kind:     "packet",
				Status:   phase,
				SourceID: source,
				TargetID: target,
			})
		}
	}
	return messages
}

// determinePhase 根据覆盖率返回当前传播阶段。
func determinePhase(model propagationModel) string {
	if model.CoverageRate >= 1 {
		return "全网覆盖"
	}
	if model.Hops == 0 {
		return "源节点广播"
	}
	return "邻居转发"
}

// phaseIndex 将传播阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "源节点广播":
		return 0
	case "邻居转发":
		return 1
	case "全网覆盖":
		return 2
	default:
		return 0
	}
}

// toneByCoverage 根据覆盖率选择事件和指标色调。
func toneByCoverage(rate float64) string {
	if rate >= 1 {
		return "success"
	}
	if rate >= 0.5 {
		return "warning"
	}
	return "info"
}

// contains 判断节点列表中是否存在指定节点。
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// uniqueStrings 对前沿节点去重，避免重复扩散。
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

// decodeSeenBy 恢复已接收节点集合。
func decodeSeenBy(value any) map[string]bool {
	entry, ok := value.(map[string]any)
	if !ok {
		return map[string]bool{"seed": true}
	}
	result := make(map[string]bool, len(entry))
	for key, raw := range entry {
		result[key] = framework.BoolValue(raw, false)
	}
	if len(result) == 0 {
		result["seed"] = true
	}
	return result
}

// decodeTopology 恢复节点拓扑结构。
func decodeTopology(value any) map[string][]string {
	entry, ok := value.(map[string]any)
	if !ok {
		return map[string][]string{
			"seed":    {"relay-1", "relay-2"},
			"relay-1": {"seed", "relay-3"},
			"relay-2": {"seed", "relay-4"},
			"relay-3": {"relay-1", "relay-4"},
			"relay-4": {"relay-2", "relay-3"},
		}
	}
	result := make(map[string][]string, len(entry))
	for key, raw := range entry {
		result[key] = framework.ToStringSlice(raw)
	}
	if len(result) == 0 {
		return map[string][]string{
			"seed":    {"relay-1", "relay-2"},
			"relay-1": {"seed", "relay-3"},
			"relay-2": {"seed", "relay-4"},
			"relay-3": {"relay-1", "relay-4"},
			"relay-4": {"relay-2", "relay-3"},
		}
	}
	return result
}


// applySharedPropagationState 将网络基础组中的拓扑变化和已传播前沿映射回 Gossip 场景。
func applySharedPropagationState(model *propagationModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if topology, ok := sharedState["topology"].(map[string]any); ok {
		if linkedMessage := framework.StringValue(topology["message_label"], ""); linkedMessage != "" {
			model.MessageLabel = linkedMessage
		}
		if frontier := framework.ToStringSlice(topology["frontier"]); len(frontier) > 0 {
			model.Frontier = frontier
		}
	}
	if routing, ok := sharedState["routing_table"].(map[string]any); ok {
		if seenCount := int(framework.NumberValue(routing["gossip_seen"], 0)); seenCount > len(model.SeenBy) {
			model.CoverageRate = float64(seenCount) / float64(len(model.Topology))
		}
	}
}
