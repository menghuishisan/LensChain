package mpttrie

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造状态树场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "mpt-trie",
		Title:        "状态树（MPT）",
		Phase:        "路径定位",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   12,
		Stages:       []string{"路径定位", "分支展开", "哈希回写"},
		Nodes: []framework.Node{
			{ID: "root", Label: "Root", Status: "active", Role: "trie", X: 120, Y: 200},
			{ID: "branch-a", Label: "Branch-A", Status: "normal", Role: "trie", X: 300, Y: 120},
			{ID: "branch-b", Label: "Branch-B", Status: "normal", Role: "trie", X: 300, Y: 280},
			{ID: "leaf-x", Label: "Leaf-X", Status: "normal", Role: "trie", X: 520, Y: 120},
			{ID: "leaf-y", Label: "Leaf-Y", Status: "normal", Role: "trie", X: 520, Y: 280},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化账户路径和值。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := trieModel{
		AccountKey: "0xabc",
		LeafValues: map[string]string{
			"leaf-x": "balance:10",
			"leaf-y": "balance:20",
		},
	}
	model.RootHash = recalculateRoot(model)
	return rebuildState(state, model, "路径定位")
}

// Step 推进路径定位、分支展开和根哈希回写。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "路径定位"))
	if phase == "哈希回写" {
		model.RootHash = recalculateRoot(model)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("MPT 状态树进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 更新账户值并重新计算根哈希。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.AccountKey = framework.StringValue(input.Params["account"], "0xabc")
	model.LeafValues["leaf-x"] = "balance:42"
	model.RootHash = recalculateRoot(model)
	if err := rebuildState(state, model, "哈希回写"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "更新账户", fmt.Sprintf("已更新账户 %s 的状态。", model.AccountKey), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出路径节点、叶子值和根哈希。
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

// trieModel 保存路径键、叶子节点值和根哈希。
type trieModel struct {
	AccountKey string            `json:"account_key"`
	LeafValues map[string]string `json:"leaf_values"`
	RootHash   string            `json:"root_hash"`
}

// rebuildState 将状态树模型映射为节点高亮、路径消息和指标。
func rebuildState(state *framework.SceneState, model trieModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	if phase == "哈希回写" {
		state.Nodes[0].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "root-branch", Label: model.AccountKey, Kind: "pointer", Status: phase, SourceID: "root", TargetID: "branch-a"},
		{ID: "branch-leaf", Label: framework.Abbreviate(model.RootHash, 12), Kind: "pointer", Status: phase, SourceID: "branch-a", TargetID: "leaf-x"},
	}
	state.Metrics = []framework.Metric{
		{Key: "account", Label: "账户键", Value: model.AccountKey, Tone: "info"},
		{Key: "leaf_x", Label: "Leaf-X", Value: model.LeafValues["leaf-x"], Tone: "warning"},
		{Key: "leaf_y", Label: "Leaf-Y", Value: model.LeafValues["leaf-y"], Tone: "success"},
		{Key: "root", Label: "根哈希", Value: framework.Abbreviate(model.RootHash, 12), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "RootHash", Value: model.RootHash},
		{Label: "Leaf-X", Value: model.LeafValues["leaf-x"]},
		{Label: "Leaf-Y", Value: model.LeafValues["leaf-y"]},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"mpt_trie":   model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 MPT 中路径定位、分支展开和根哈希回写。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复状态树模型。
func decodeModel(state *framework.SceneState) trieModel {
	entry, ok := state.Data["mpt_trie"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["mpt_trie"].(trieModel); ok {
			return typed
		}
		model := trieModel{
			AccountKey: "0xabc",
			LeafValues: map[string]string{"leaf-x": "balance:10", "leaf-y": "balance:20"},
		}
		model.RootHash = recalculateRoot(model)
		return model
	}
	return trieModel{
		AccountKey: framework.StringValue(entry["account_key"], "0xabc"),
		LeafValues: decodeStringMap(entry["leaf_values"]),
		RootHash:   framework.StringValue(entry["root_hash"], ""),
	}
}

// recalculateRoot 根据两个叶子值重新计算根哈希。
func recalculateRoot(model trieModel) string {
	left := framework.HashText(model.LeafValues["leaf-x"])
	right := framework.HashText(model.LeafValues["leaf-y"])
	return framework.HashText(left + right)
}

// nextPhase 返回状态树的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "路径定位":
		return "分支展开"
	case "分支展开":
		return "哈希回写"
	default:
		return "路径定位"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "路径定位":
		return 0
	case "分支展开":
		return 1
	case "哈希回写":
		return 0
	default:
		return 0
	}
}

// toneByPhase 返回状态树阶段色调。
func toneByPhase(phase string) string {
	if phase == "哈希回写" {
		return "success"
	}
	return "info"
}

// decodeStringMap 恢复字符串映射。
func decodeStringMap(value any) map[string]string {
	entry, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]string); ok {
			return typed
		}
		return map[string]string{"leaf-x": "balance:10", "leaf-y": "balance:20"}
	}
	result := make(map[string]string, len(entry))
	for key, raw := range entry {
		result[key] = framework.StringValue(raw, "")
	}
	return result
}
