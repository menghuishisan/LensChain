package nodeloadbalance

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造节点负载均衡场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "node-load-balance",
		Title:        "节点负载均衡",
		Phase:        "采集负载",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   14,
		Stages:       []string{"采集负载", "评估热点", "迁移流量"},
		Nodes: []framework.Node{
			{ID: "lb-1", Label: "LB-1", Status: "active", Role: "gateway", X: 120, Y: 150},
			{ID: "lb-2", Label: "LB-2", Status: "normal", Role: "gateway", X: 320, Y: 110},
			{ID: "lb-3", Label: "LB-3", Status: "normal", Role: "gateway", X: 320, Y: 300},
			{ID: "lb-4", Label: "LB-4", Status: "normal", Role: "gateway", X: 540, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化节点负载、热点节点和迁移目标。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := balanceModel{
		Loads:         map[string]float64{"lb-1": 74, "lb-2": 43, "lb-3": 36, "lb-4": 58},
		HotNode:       "lb-1",
		TargetNode:    "lb-2",
		MigrationRate: 0,
		RequestRate:   210,
	}
	return rebuildState(state, model, "采集负载")
}

// Step 推进负载采集、热点判断和流量迁移过程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedBalanceState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "采集负载"))
	switch phase {
	case "评估热点":
		model.HotNode, model.TargetNode = evaluateHotspot(model.Loads)
	case "迁移流量":
		model.MigrationRate = 22
		shiftLoad(model.Loads, model.HotNode, model.TargetNode, model.MigrationRate)
		model.HotNode, model.TargetNode = evaluateHotspot(model.Loads)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("负载均衡进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology": map[string]any{
				"loads":          framework.CloneMap(toAnyMap(model.Loads)),
				"hot_node":       model.HotNode,
				"migration_to":   model.TargetNode,
				"migration_rate": model.MigrationRate,
			},
		},
	}, nil
}

// HandleAction 手动将部分流量迁移到指定节点。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.TargetNode = strings.ToLower(framework.StringValue(input.Params["resource_id"], "LB-2"))
	model.HotNode, _ = evaluateHotspot(model.Loads)
	model.MigrationRate = 18
	shiftLoad(model.Loads, model.HotNode, model.TargetNode, model.MigrationRate)
	model.HotNode, model.TargetNode = evaluateHotspot(model.Loads)
	if err := rebuildState(state, model, "迁移流量"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "迁移流量", fmt.Sprintf("已将流量迁移到 %s。", strings.ToUpper(model.TargetNode)), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology": map[string]any{
				"loads":          framework.CloneMap(toAnyMap(model.Loads)),
				"hot_node":       model.HotNode,
				"migration_to":   model.TargetNode,
				"migration_rate": model.MigrationRate,
			},
		},
	}, nil
}

// BuildRenderState 输出节点热力、流量迁移和热点评估结果。
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

// SyncSharedState 在共享拓扑负载变化后重建负载均衡场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedBalanceState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// balanceModel 保存各节点负载、热点节点和迁移速率。
type balanceModel struct {
	Loads         map[string]float64 `json:"loads"`
	HotNode       string             `json:"hot_node"`
	TargetNode    string             `json:"target_node"`
	MigrationRate float64            `json:"migration_rate"`
	RequestRate   float64            `json:"request_rate"`
}

// rebuildState 将负载模型映射为节点热力图、迁移消息和指标。
func rebuildState(state *framework.SceneState, model balanceModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		node := &state.Nodes[index]
		load := model.Loads[node.ID]
		node.Load = load
		node.Status = "normal"
		if node.ID == model.HotNode {
			node.Status = "warning"
		}
		if node.ID == model.TargetNode {
			node.Status = "success"
		}
		if phase == "采集负载" && index == 0 {
			node.Status = "active"
		}
		node.Attributes = map[string]any{
			"load_percent": load,
			"request_rate": model.RequestRate,
		}
	}
	state.Messages = []framework.Message{
		{ID: "traffic-scan", Label: fmt.Sprintf("qps:%.0f", model.RequestRate), Kind: "packet", Status: phase, SourceID: "lb-1", TargetID: model.HotNode},
		{ID: "traffic-shift", Label: fmt.Sprintf("shift:%.0f%%", model.MigrationRate), Kind: "packet", Status: phase, SourceID: model.HotNode, TargetID: model.TargetNode},
	}
	state.Metrics = []framework.Metric{
		{Key: "hot", Label: "热点节点", Value: strings.ToUpper(model.HotNode), Tone: "warning"},
		{Key: "target", Label: "迁移目标", Value: strings.ToUpper(model.TargetNode), Tone: "success"},
		{Key: "migration", Label: "迁移比例", Value: fmt.Sprintf("%.0f%%", model.MigrationRate), Tone: "info"},
		{Key: "spread", Label: "负载离散度", Value: fmt.Sprintf("%.1f", loadSpread(model.Loads)), Tone: spreadTone(model.Loads)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "LB-1", Value: fmt.Sprintf("%.0f%%", model.Loads["lb-1"])},
		{Label: "LB-2", Value: fmt.Sprintf("%.0f%%", model.Loads["lb-2"])},
		{Label: "LB-3", Value: fmt.Sprintf("%.0f%%", model.Loads["lb-3"])},
		{Label: "LB-4", Value: fmt.Sprintf("%.0f%%", model.Loads["lb-4"])},
	}
	state.Data = map[string]any{
		"phase_name":        phase,
		"node_load_balance": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟节点负载采集、热点识别和流量迁移后的热力变化。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复负载模型。
func decodeModel(state *framework.SceneState) balanceModel {
	entry, ok := state.Data["node_load_balance"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["node_load_balance"].(balanceModel); ok {
			return typed
		}
		return balanceModel{
			Loads:         map[string]float64{"lb-1": 74, "lb-2": 43, "lb-3": 36, "lb-4": 58},
			HotNode:       "lb-1",
			TargetNode:    "lb-2",
			MigrationRate: 0,
			RequestRate:   210,
		}
	}
	return balanceModel{
		Loads:         decodeFloatMap(entry["loads"]),
		HotNode:       framework.StringValue(entry["hot_node"], "lb-1"),
		TargetNode:    framework.StringValue(entry["target_node"], "lb-2"),
		MigrationRate: framework.NumberValue(entry["migration_rate"], 0),
		RequestRate:   framework.NumberValue(entry["request_rate"], 210),
	}
}

// evaluateHotspot 选出当前负载最高的热点节点和最低的承接节点。
func evaluateHotspot(loads map[string]float64) (string, string) {
	hotNode := ""
	targetNode := ""
	hotLoad := -1.0
	lowLoad := 101.0
	for nodeID, load := range loads {
		if load > hotLoad {
			hotLoad = load
			hotNode = nodeID
		}
		if load < lowLoad {
			lowLoad = load
			targetNode = nodeID
		}
	}
	return hotNode, targetNode
}

// shiftLoad 将一部分负载从热点节点迁移到目标节点。
func shiftLoad(loads map[string]float64, from string, to string, rate float64) {
	if from == to {
		return
	}
	transfer := framework.Clamp(rate, 0, loads[from])
	loads[from] = framework.Clamp(loads[from]-transfer, 0, 100)
	loads[to] = framework.Clamp(loads[to]+transfer, 0, 100)
}

// loadSpread 计算简单负载离散度。
func loadSpread(loads map[string]float64) float64 {
	if len(loads) == 0 {
		return 0
	}
	total := 0.0
	for _, load := range loads {
		total += load
	}
	mean := total / float64(len(loads))
	variance := 0.0
	for _, load := range loads {
		diff := load - mean
		variance += diff * diff
	}
	return variance / float64(len(loads))
}

// nextPhase 返回负载均衡流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "采集负载":
		return "评估热点"
	case "评估热点":
		return "迁移流量"
	default:
		return "采集负载"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "采集负载":
		return 0
	case "评估热点":
		return 1
	case "迁移流量":
		return 2
	default:
		return 0
	}
}

// toneByPhase 返回负载均衡阶段色调。
func toneByPhase(phase string) string {
	if phase == "迁移流量" {
		return "success"
	}
	if phase == "评估热点" {
		return "warning"
	}
	return "info"
}

// spreadTone 根据负载离散度选择指标色调。
func spreadTone(loads map[string]float64) string {
	if loadSpread(loads) > 200 {
		return "warning"
	}
	return "info"
}

// decodeFloatMap 将通用 JSON 映射恢复为浮点映射。
func decodeFloatMap(value any) map[string]float64 {
	entry, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]float64); ok {
			return typed
		}
		return map[string]float64{"lb-1": 74, "lb-2": 43, "lb-3": 36, "lb-4": 58}
	}
	result := make(map[string]float64, len(entry))
	for key, raw := range entry {
		result[key] = framework.NumberValue(raw, 0)
	}
	return result
}

// toAnyMap 将浮点映射转换为通用映射，供共享状态输出。
func toAnyMap(value map[string]float64) map[string]any {
	result := make(map[string]any, len(value))
	for key, load := range value {
		result[key] = load
	}
	return result
}

// applySharedBalanceState 将网络基础组中的共享拓扑负载映射回负载均衡场景。
func applySharedBalanceState(model *balanceModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if topology, ok := sharedState["topology"].(map[string]any); ok {
		if loads := decodeFloatMap(topology["loads"]); len(loads) > 0 {
			model.Loads = loads
		}
		model.HotNode = framework.StringValue(topology["hot_node"], model.HotNode)
		model.TargetNode = framework.StringValue(topology["migration_to"], model.TargetNode)
		model.MigrationRate = framework.NumberValue(topology["migration_rate"], model.MigrationRate)
	}
	if load, ok := sharedState["load"].(map[string]any); ok {
		model.RequestRate = 210 + framework.NumberValue(load["relay_hops"], 0)*12
	}
}
