package fiftyoneattack

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 51% 算力攻击场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "51-percent-attack",
		Title:        "51% 算力攻击",
		Phase:        "私链积累",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   18,
		Stages:       []string{"私链积累", "追平主链", "公开重组"},
		Nodes: []framework.Node{
			{ID: "honest-chain", Label: "Honest Chain", Status: "normal", Role: "chain", X: 180, Y: 180},
			{ID: "attacker-chain", Label: "Attacker Chain", Status: "attack", Role: "chain", X: 480, Y: 180},
			{ID: "hashrate", Label: "Hashrate", Status: "warning", Role: "metric", X: 330, Y: 320},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化诚实链、攻击链和攻击者算力占比。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := attackState{
		HonestHeight:     3,
		AttackerHeight:   1,
		AttackerHashrate: 0.55,
		HonestHashrate:   0.45,
		Reorganized:      false,
	}
	applySharedAttackState(&model, input.SharedState)
	return rebuildState(state, model, "私链积累")
}

// Step 推进私链积累、追平主链和链重组流程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeState(state)
	applySharedAttackState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "私链积累"))
	switch phase {
	case "私链积累":
		model.AttackerHeight++
	case "追平主链":
		if model.AttackerHeight < model.HonestHeight {
			model.AttackerHeight++
		} else {
			model.HonestHeight++
			model.AttackerHeight++
		}
	case "公开重组":
		if model.AttackerHeight >= model.HonestHeight {
			model.Reorganized = true
			model.HonestHeight = model.AttackerHeight
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("攻击流程进入%s阶段。", phase), phaseTone(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"blockchain": map[string]any{
				"honest_height":   model.HonestHeight,
				"attacker_height": model.AttackerHeight,
				"reorganized":     model.Reorganized,
			},
			"nodes": map[string]any{
				"attacker_hashrate": model.AttackerHashrate,
			},
		},
	}, nil
}

// HandleAction 调整攻击者算力占比，驱动后续分叉竞争结果。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeState(state)
	ratio := framework.NumberValue(input.Params["ratio"], 0.55)
	if ratio < 0.01 {
		ratio = 0.01
	}
	if ratio > 0.99 {
		ratio = 0.99
	}
	model.AttackerHashrate = ratio
	model.HonestHashrate = 1 - ratio
	if err := rebuildState(state, model, "私链积累"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "提升攻击算力", fmt.Sprintf("攻击者算力比例调整为 %.2f。", ratio), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"attacker_hashrate": model.AttackerHashrate,
				"honest_hashrate":   model.HonestHashrate,
			},
		},
	}, nil
}

// BuildRenderState 输出双链高度、算力比例和重组结果。
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

// SyncSharedState 在联动组共享状态变化后重建 51% 攻击场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeState(state)
	applySharedAttackState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// attackState 保存诚实链与攻击链的高度和算力分布。
type attackState struct {
	HonestHeight     int     `json:"honest_height"`
	AttackerHeight   int     `json:"attacker_height"`
	AttackerHashrate float64 `json:"attacker_hashrate"`
	HonestHashrate   float64 `json:"honest_hashrate"`
	Reorganized      bool    `json:"reorganized"`
}

// rebuildState 将攻击模型转成双链可视化状态。
func rebuildState(state *framework.SceneState, model attackState, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes[0].Status = honestStatus(model)
	state.Nodes[0].Load = float64(model.HonestHeight * 20)
	state.Nodes[0].Attributes = map[string]any{"height": model.HonestHeight}
	state.Nodes[1].Status = attackerStatus(model)
	state.Nodes[1].Load = float64(model.AttackerHeight * 20)
	state.Nodes[1].Attributes = map[string]any{"height": model.AttackerHeight}
	state.Nodes[2].Status = "warning"
	state.Nodes[2].Load = model.AttackerHashrate * 100
	state.Nodes[2].Attributes = map[string]any{
		"attacker_hashrate": model.AttackerHashrate,
		"honest_hashrate":   model.HonestHashrate,
	}
	state.Messages = []framework.Message{
		{ID: "honest-tip", Label: "Honest Tip", Kind: "attack", Status: phase, SourceID: "honest-chain"},
		{ID: "attacker-tip", Label: "Private Tip", Kind: "attack", Status: phase, SourceID: "attacker-chain"},
	}
	state.Metrics = []framework.Metric{
		{Key: "honest_height", Label: "诚实链高度", Value: fmt.Sprintf("%d", model.HonestHeight), Tone: "info"},
		{Key: "attacker_height", Label: "攻击链高度", Value: fmt.Sprintf("%d", model.AttackerHeight), Tone: "warning"},
		{Key: "attacker_hashrate", Label: "攻击算力占比", Value: fmt.Sprintf("%.0f%%", model.AttackerHashrate*100), Tone: "warning"},
		{Key: "reorg", Label: "是否重组", Value: framework.BoolText(model.Reorganized), Tone: reorgTone(model.Reorganized)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "双链差值", Value: fmt.Sprintf("%d", model.AttackerHeight-model.HonestHeight)},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"attack":     model,
	}
	state.Extra = map[string]any{
		"description": "该场景实现私链积累、追平主链和公开重组的双链竞争过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeState 从通用 JSON 状态恢复攻击模型。
func decodeState(state *framework.SceneState) attackState {
	entry, ok := state.Data["attack"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["attack"].(attackState); ok {
			return typed
		}
		return attackState{HonestHeight: 3, AttackerHeight: 1, AttackerHashrate: 0.55, HonestHashrate: 0.45}
	}
	return attackState{
		HonestHeight:     int(framework.NumberValue(entry["honest_height"], 3)),
		AttackerHeight:   int(framework.NumberValue(entry["attacker_height"], 1)),
		AttackerHashrate: framework.NumberValue(entry["attacker_hashrate"], 0.55),
		HonestHashrate:   framework.NumberValue(entry["honest_hashrate"], 0.45),
		Reorganized:      framework.BoolValue(entry["reorganized"], false),
	}
}

// nextPhase 返回攻击流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "私链积累":
		return "追平主链"
	case "追平主链":
		return "公开重组"
	default:
		return "私链积累"
	}
}

// phaseIndex 将阶段名称映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "私链积累":
		return 0
	case "追平主链":
		return 1
	case "公开重组":
		return 2
	default:
		return 0
	}
}

// honestStatus 返回诚实链节点状态。
func honestStatus(model attackState) string {
	if model.Reorganized {
		return "warning"
	}
	return "normal"
}

// attackerStatus 返回攻击链节点状态。
func attackerStatus(model attackState) string {
	if model.Reorganized {
		return "success"
	}
	return "attack"
}

// phaseTone 返回阶段事件色调。
func phaseTone(phase string) string {
	if phase == "公开重组" {
		return "warning"
	}
	return "info"
}

// reorgTone 返回重组指标色调。
func reorgTone(flag bool) string {
	if flag {
		return "warning"
	}
	return "info"
}

// applySharedAttackState 将 PoW 联动组中的链高度和算力变化映射回 51% 攻击模型。
func applySharedAttackState(model *attackState, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if blockchain, ok := sharedState["blockchain"].(map[string]any); ok {
		if honestHeight := int(framework.NumberValue(blockchain["honest_height"], float64(model.HonestHeight))); honestHeight > 0 {
			model.HonestHeight = honestHeight
		}
		if attackerHeight := int(framework.NumberValue(blockchain["attacker_height"], float64(model.AttackerHeight))); attackerHeight > 0 {
			model.AttackerHeight = attackerHeight
		}
		if linkedHeight := int(framework.NumberValue(blockchain["height"], float64(model.HonestHeight))); linkedHeight > model.HonestHeight {
			model.HonestHeight = linkedHeight
		}
		model.Reorganized = framework.BoolValue(blockchain["reorganized"], model.Reorganized)
	}
	if nodes, ok := sharedState["nodes"].(map[string]any); ok {
		if attackerHashrate := framework.NumberValue(nodes["attacker_hashrate"], model.AttackerHashrate); attackerHashrate > 0 {
			model.AttackerHashrate = attackerHashrate
			model.HonestHashrate = 1 - attackerHashrate
		}
		if honestHashrate := framework.NumberValue(nodes["honest_hashrate"], model.HonestHashrate); honestHashrate > 0 {
			model.HonestHashrate = honestHashrate
		}
	}
}
