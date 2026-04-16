package posstaking

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 PoS 质押经济场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "pos-staking",
		Title:        "PoS 质押经济",
		Phase:        "汇总质押",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   16,
		Stages:       []string{"汇总质押", "分发奖励", "罚没 Slash"},
		Nodes: []framework.Node{
			{ID: "validator-a", Label: "Validator-A", Status: "normal", Role: "staking", X: 140, Y: 200},
			{ID: "validator-b", Label: "Validator-B", Status: "normal", Role: "staking", X: 320, Y: 120},
			{ID: "validator-c", Label: "Validator-C", Status: "normal", Role: "staking", X: 320, Y: 280},
			{ID: "delegator", Label: "Delegator", Status: "normal", Role: "staking", X: 540, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化验证者质押、委托和收益率。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := stakingModel{
		Validators: []validatorStake{
			{ID: "validator-a", Label: "Validator-A", Stake: 80, Reward: 0, Slashed: false},
			{ID: "validator-b", Label: "Validator-B", Stake: 70, Reward: 0, Slashed: false},
			{ID: "validator-c", Label: "Validator-C", Stake: 50, Reward: 0, Slashed: false},
		},
		DelegatorStake:  40,
		SlashCount:      0,
		RewardPerEpoch:  6,
		AnnualYieldRate: 0.12,
	}
	applySharedStakingState(&model, input.SharedState)
	return rebuildState(state, model, "汇总质押")
}

// Step 推进质押汇总、奖励发放与 Slash 罚没过程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedStakingState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "汇总质押"))
	switch phase {
	case "分发奖励":
		distributeRewards(&model)
	case "罚没 Slash":
		applySlash(&model, "validator-a")
	case "汇总质押":
		model.DelegatorStake += 2
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("PoS 质押经济进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":      validatorsShared(model),
				"delegator_stake": model.DelegatorStake,
				"slash_count":     model.SlashCount,
			},
			"token_supply": map[string]any{
				"reward_emission": totalRewards(model),
			},
		},
	}, nil
}

// HandleAction 允许手动对某个验证者执行 Slash。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	target := framework.StringValue(input.Params["resource_id"], "Validator-A")
	applySlash(&model, framework.NormalizeSlug(target, "validator-a"))
	if err := rebuildState(state, model, "罚没 Slash"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "执行 Slash", fmt.Sprintf("已对 %s 执行 Slash。", target), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":  validatorsShared(model),
				"slash_count": model.SlashCount,
			},
		},
	}, nil
}

// BuildRenderState 输出质押量、收益和罚没结果。
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

// SyncSharedState 在质押经济共享状态变化后重建 PoS 质押场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedStakingState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// stakingModel 保存质押经济中的验证者、委托人与收益状态。
type stakingModel struct {
	Validators      []validatorStake `json:"validators"`
	DelegatorStake  float64          `json:"delegator_stake"`
	SlashCount      int              `json:"slash_count"`
	RewardPerEpoch  float64          `json:"reward_per_epoch"`
	AnnualYieldRate float64          `json:"annual_yield_rate"`
}

// validatorStake 保存单个验证者的质押量、奖励和罚没标记。
type validatorStake struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Stake   float64 `json:"stake"`
	Reward  float64 `json:"reward"`
	Slashed bool    `json:"slashed"`
}

// rebuildState 将质押模型映射为节点、奖励流和指标。
func rebuildState(state *framework.SceneState, model stakingModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	for index, validator := range model.Validators {
		state.Nodes[index].Load = validator.Stake
		state.Nodes[index].Stake = validator.Stake
		state.Nodes[index].Status = "active"
		if validator.Slashed {
			state.Nodes[index].Status = "warning"
		}
		if phase == "分发奖励" && !validator.Slashed {
			state.Nodes[index].Status = "success"
		}
	}
	state.Nodes[3].Load = model.DelegatorStake
	state.Nodes[3].Status = "normal"
	state.Messages = buildMessages(model, phase)
	state.Metrics = []framework.Metric{
		{Key: "total_stake", Label: "总质押量", Value: framework.MetricValue(totalStake(model), ""), Tone: "info"},
		{Key: "delegator", Label: "委托质押", Value: framework.MetricValue(model.DelegatorStake, ""), Tone: "warning"},
		{Key: "epoch_reward", Label: "轮次奖励", Value: framework.MetricValue(model.RewardPerEpoch, ""), Tone: "success"},
		{Key: "slash_count", Label: "Slash 次数", Value: fmt.Sprintf("%d", model.SlashCount), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "年化收益率", Value: fmt.Sprintf("%.1f%%", model.AnnualYieldRate*100)},
		{Label: "已发奖励", Value: framework.MetricValue(totalRewards(model), "")},
		{Label: "当前阶段", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"pos_staking": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 PoS 质押汇总、按权重发放奖励与 Slash 罚没。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复质押经济模型。
func decodeModel(state *framework.SceneState) stakingModel {
	entry, ok := state.Data["pos_staking"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["pos_staking"].(stakingModel); ok {
			return typed
		}
		return stakingModel{
			Validators: []validatorStake{
				{ID: "validator-a", Label: "Validator-A", Stake: 80},
				{ID: "validator-b", Label: "Validator-B", Stake: 70},
				{ID: "validator-c", Label: "Validator-C", Stake: 50},
			},
			DelegatorStake: 40, RewardPerEpoch: 6, AnnualYieldRate: 0.12,
		}
	}
	return stakingModel{
		Validators:      decodeValidators(entry["validators"]),
		DelegatorStake:  framework.NumberValue(entry["delegator_stake"], 40),
		SlashCount:      int(framework.NumberValue(entry["slash_count"], 0)),
		RewardPerEpoch:  framework.NumberValue(entry["reward_per_epoch"], 6),
		AnnualYieldRate: framework.NumberValue(entry["annual_yield_rate"], 0.12),
	}
}

// buildMessages 为当前阶段生成奖励与罚没消息。
func buildMessages(model stakingModel, phase string) []framework.Message {
	messages := make([]framework.Message, 0, len(model.Validators))
	for _, validator := range model.Validators {
		label := "Stake"
		status := phase
		if phase == "分发奖励" {
			label = "Reward"
		}
		if validator.Slashed && phase == "罚没 Slash" {
			label = "Slash"
			status = "warning"
		}
		messages = append(messages, framework.Message{
			ID:       validator.ID + "-" + phase,
			Label:    label,
			Kind:     "proposal",
			Status:   status,
			SourceID: "delegator",
			TargetID: validator.ID,
		})
	}
	return messages
}

// distributeRewards 按质押权重为验证者发放奖励。
func distributeRewards(model *stakingModel) {
	total := totalStake(*model)
	if total == 0 {
		return
	}
	for index := range model.Validators {
		share := model.Validators[index].Stake / total
		model.Validators[index].Reward += model.RewardPerEpoch * share
		model.Validators[index].Stake += model.RewardPerEpoch * share
	}
}

// applySlash 对指定验证者执行罚没，并记录 Slash 次数。
func applySlash(model *stakingModel, target string) {
	for index := range model.Validators {
		if model.Validators[index].ID != target {
			continue
		}
		model.Validators[index].Stake = framework.Clamp(model.Validators[index].Stake*0.92, 0, model.Validators[index].Stake)
		model.Validators[index].Slashed = true
		model.SlashCount++
		return
	}
}

// validatorsShared 生成共享状态所需的验证者列表。
func validatorsShared(model stakingModel) []map[string]any {
	result := make([]map[string]any, 0, len(model.Validators))
	for _, validator := range model.Validators {
		result = append(result, map[string]any{
			"id":      validator.ID,
			"stake":   validator.Stake,
			"reward":  validator.Reward,
			"slashed": validator.Slashed,
		})
	}
	return result
}

// decodeValidators 恢复验证者切片。
func decodeValidators(value any) []validatorStake {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]validatorStake); ok {
			return append([]validatorStake(nil), typed...)
		}
		return []validatorStake{
			{ID: "validator-a", Label: "Validator-A", Stake: 80},
			{ID: "validator-b", Label: "Validator-B", Stake: 70},
			{ID: "validator-c", Label: "Validator-C", Stake: 50},
		}
	}
	result := make([]validatorStake, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, validatorStake{
			ID:      framework.StringValue(entry["id"], ""),
			Label:   framework.StringValue(entry["label"], ""),
			Stake:   framework.NumberValue(entry["stake"], 0),
			Reward:  framework.NumberValue(entry["reward"], 0),
			Slashed: framework.BoolValue(entry["slashed"], false),
		})
	}
	if len(result) == 0 {
		return []validatorStake{
			{ID: "validator-a", Label: "Validator-A", Stake: 80},
			{ID: "validator-b", Label: "Validator-B", Stake: 70},
			{ID: "validator-c", Label: "Validator-C", Stake: 50},
		}
	}
	return result
}

// totalStake 统计全网质押总量。
func totalStake(model stakingModel) float64 {
	total := model.DelegatorStake
	for _, validator := range model.Validators {
		total += validator.Stake
	}
	return total
}

// totalRewards 统计累计奖励。
func totalRewards(model stakingModel) float64 {
	total := 0.0
	for _, validator := range model.Validators {
		total += validator.Reward
	}
	return total
}

// nextPhase 返回 PoS 质押经济的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "汇总质押":
		return "分发奖励"
	case "分发奖励":
		return "罚没 Slash"
	default:
		return "汇总质押"
	}
}

// phaseIndex 将阶段映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "汇总质押":
		return 0
	case "分发奖励":
		return 1
	case "罚没 Slash":
		return 2
	default:
		return 0
	}
}

// toneByPhase 返回质押阶段对应的展示色调。
func toneByPhase(phase string) string {
	switch phase {
	case "分发奖励":
		return "success"
	case "罚没 Slash":
		return "warning"
	default:
		return "info"
	}
}

// applySharedStakingState 将 PoS 经济组中的验证者质押和代币供应变化映射回质押经济场景。
func applySharedStakingState(model *stakingModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if stakes, ok := sharedState["stakes"].(map[string]any); ok {
		if sharedValidators, ok := stakes["validators"].([]any); ok {
			model.Validators = decodeValidators(sharedValidators)
		}
		model.DelegatorStake = framework.NumberValue(stakes["delegator_stake"], model.DelegatorStake)
		model.SlashCount = int(framework.NumberValue(stakes["slash_count"], float64(model.SlashCount)))
		model.AnnualYieldRate = framework.Clamp(0.08+float64(model.SlashCount)*0.01, 0.01, 0.2)
	}
}
