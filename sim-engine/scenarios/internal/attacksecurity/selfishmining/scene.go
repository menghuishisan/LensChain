package selfishmining

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造自私挖矿场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "selfish-mining",
		Title:        "自私挖矿",
		Phase:        "私链积累",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   16,
		Stages:       []string{"私链积累", "条件公开", "收益比较"},
		Nodes: []framework.Node{
			{ID: "honest", Label: "Honest", Status: "normal", Role: "miner", X: 140, Y: 200},
			{ID: "selfish", Label: "Selfish", Status: "normal", Role: "miner", X: 340, Y: 120},
			{ID: "public", Label: "Public", Status: "normal", Role: "miner", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化诚实链、私链长度和收益情况。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := miningModel{
		HonestLength:    5,
		PrivateLength:   6,
		PublishedLength: 5,
		HonestReward:    5,
		SelfishReward:   6,
		Strategy:        "withhold",
	}
	return rebuildState(state, model, "私链积累")
}

// Step 推进私链积累、公开时机和收益比较。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "私链积累"))
	switch phase {
	case "条件公开":
		model.PublishedLength = model.PrivateLength
		model.Strategy = "publish"
	case "收益比较":
		if model.PrivateLength >= model.HonestLength {
			model.SelfishReward += 2
		} else {
			model.HonestReward++
		}
		model.Strategy = "compare"
	default:
		model.PrivateLength++
		model.HonestLength++
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("自私挖矿进入%s阶段。", phase), toneByRewards(model))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 允许提前公开私链以触发链竞争。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	blocks := int(framework.NumberValue(input.Params["blocks"], float64(model.PrivateLength-model.PublishedLength)))
	if blocks < 1 {
		blocks = 1
	}
	model.PublishedLength += blocks
	if model.PublishedLength > model.PrivateLength {
		model.PublishedLength = model.PrivateLength
	}
	model.Strategy = "manual_publish"
	if err := rebuildState(state, model, "条件公开"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "公开私链", fmt.Sprintf("已公开 %d 个私有区块。", blocks), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出诚实链、私链和收益对比。
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

// SyncSharedState 在 PoW 攻击组共享区块链状态变化后重建自私挖矿场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedSelfishMiningState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// miningModel 保存自私挖矿中的链长度、策略和收益。
type miningModel struct {
	HonestLength    int    `json:"honest_length"`
	PrivateLength   int    `json:"private_length"`
	PublishedLength int    `json:"published_length"`
	HonestReward    int    `json:"honest_reward"`
	SelfishReward   int    `json:"selfish_reward"`
	Strategy        string `json:"strategy"`
}

// rebuildState 将自私挖矿模型映射为节点、链路和指标。
func rebuildState(state *framework.SceneState, model miningModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes[0].Load = float64(model.HonestLength)
	state.Nodes[1].Load = float64(model.PrivateLength)
	state.Nodes[2].Load = float64(model.PublishedLength)
	state.Nodes[0].Status = "normal"
	state.Nodes[1].Status = "warning"
	state.Nodes[2].Status = "normal"
	if model.PublishedLength >= model.HonestLength {
		state.Nodes[2].Status = "active"
	}
	if model.SelfishReward > model.HonestReward {
		state.Nodes[1].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "private-chain", Label: fmt.Sprintf("Private-%d", model.PrivateLength), Kind: "attack", Status: phase, SourceID: "selfish", TargetID: "public"},
		{ID: "honest-chain", Label: fmt.Sprintf("Honest-%d", model.HonestLength), Kind: "attack", Status: phase, SourceID: "honest", TargetID: "public"},
	}
	state.Metrics = []framework.Metric{
		{Key: "private_length", Label: "私链长度", Value: fmt.Sprintf("%d", model.PrivateLength), Tone: "warning"},
		{Key: "honest_length", Label: "诚实链长度", Value: fmt.Sprintf("%d", model.HonestLength), Tone: "info"},
		{Key: "selfish_reward", Label: "自私矿工收益", Value: fmt.Sprintf("%d", model.SelfishReward), Tone: toneByRewards(model)},
		{Key: "honest_reward", Label: "诚实矿工收益", Value: fmt.Sprintf("%d", model.HonestReward), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "当前策略", Value: model.Strategy},
		{Label: "公开链长度", Value: fmt.Sprintf("%d", model.PublishedLength)},
		{Label: "收益领先方", Value: leadingSide(model)},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"selfish_mining": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟私有链积累、选择性公开和自私挖矿收益比较。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复自私挖矿模型。
func decodeModel(state *framework.SceneState) miningModel {
	entry, ok := state.Data["selfish_mining"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["selfish_mining"].(miningModel); ok {
			return typed
		}
		return miningModel{HonestLength: 5, PrivateLength: 6, PublishedLength: 5, HonestReward: 5, SelfishReward: 6, Strategy: "withhold"}
	}
	return miningModel{
		HonestLength:    int(framework.NumberValue(entry["honest_length"], 5)),
		PrivateLength:   int(framework.NumberValue(entry["private_length"], 6)),
		PublishedLength: int(framework.NumberValue(entry["published_length"], 5)),
		HonestReward:    int(framework.NumberValue(entry["honest_reward"], 5)),
		SelfishReward:   int(framework.NumberValue(entry["selfish_reward"], 6)),
		Strategy:        framework.StringValue(entry["strategy"], "withhold"),
	}
}

// applySharedSelfishMiningState 将 PoW 攻击组共享链高度映射到自私挖矿模型。
func applySharedSelfishMiningState(model *miningModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if blockchain, ok := sharedState["blockchain"].(map[string]any); ok {
		if honestHeight, ok := blockchain["honest_height"]; ok {
			model.HonestLength = int(framework.NumberValue(honestHeight, float64(model.HonestLength)))
		}
		if attackerHeight, ok := blockchain["attacker_height"]; ok {
			model.PrivateLength = int(framework.NumberValue(attackerHeight, float64(model.PrivateLength)))
		}
	}
}

// nextPhase 返回自私挖矿流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "私链积累":
		return "条件公开"
	case "条件公开":
		return "收益比较"
	default:
		return "私链积累"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "私链积累":
		return 0
	case "条件公开":
		return 1
	case "收益比较":
		return 2
	default:
		return 0
	}
}

// toneByRewards 根据收益比较结果返回色调。
func toneByRewards(model miningModel) string {
	if model.SelfishReward > model.HonestReward {
		return "warning"
	}
	return "success"
}

// leadingSide 返回当前收益领先方。
func leadingSide(model miningModel) string {
	if model.SelfishReward > model.HonestReward {
		return "Selfish"
	}
	if model.SelfishReward < model.HonestReward {
		return "Honest"
	}
	return "Tie"
}
