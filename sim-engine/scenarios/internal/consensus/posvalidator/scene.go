package posvalidator

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 PoS 验证者选举场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "pos-validator",
		Title:        "PoS 验证者选举",
		Phase:        "质押汇总",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   18,
		Stages:       []string{"质押汇总", "随机抽样", "Epoch 轮转", "奖励结算"},
		Nodes: []framework.Node{
			{ID: "validator-a", Label: "Validator-A", Status: "active", Role: "validator", X: 150, Y: 170, Stake: 120},
			{ID: "validator-b", Label: "Validator-B", Status: "normal", Role: "validator", X: 320, Y: 120, Stake: 260},
			{ID: "validator-c", Label: "Validator-C", Status: "normal", Role: "validator", X: 500, Y: 170, Stake: 180},
			{ID: "validator-d", Label: "Validator-D", Status: "normal", Role: "validator", X: 330, Y: 330, Stake: 90},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化验证者权重和首个 Epoch。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	validators := defaultValidators()
	return rebuildState(state, validators, 1, "质押汇总", "")
}

// Step 执行权重汇总、随机抽样、Epoch 轮转和奖励结算。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	validators := decodeValidators(state)
	epoch := int(framework.NumberValue(state.Data["epoch"], 1))
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "质押汇总"))
	selectedID := framework.StringValue(state.Data["selected_validator"], "")
	epoch, selectedID = applySharedValidatorState(validators, input.SharedState, epoch, selectedID)
	events := make([]framework.TimelineEvent, 0, 1)

	switch phase {
	case "质押汇总":
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "汇总质押", fmt.Sprintf("当前总质押为 %.0f。", totalStake(validators)), "info"))
	case "随机抽样":
		selectedID = weightedSelect(validators, epoch)
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "选出验证者", fmt.Sprintf("Epoch %d 选中 %s。", epoch, selectedID), "success"))
	case "Epoch 轮转":
		epoch++
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Epoch 轮转", fmt.Sprintf("进入 Epoch %d。", epoch), "info"))
	case "奖励结算":
		rewardValidator(validators, selectedID)
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "奖励结算", fmt.Sprintf("%s 获得出块奖励。", selectedID), "success"))
	}

	if err := rebuildState(state, validators, epoch, phase, selectedID); err != nil {
		return framework.StepOutput{}, err
	}
	return framework.StepOutput{
		Events: events,
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":   state.Data["validators"],
				"total_stake":  state.Data["total_stake"],
				"selected":     selectedID,
				"voting_power": totalStake(validators),
			},
			"epoch":    epoch,
			"selected": selectedID,
			"token_supply": map[string]any{
				"total":       state.Data["token_supply"],
				"circulating": state.Data["token_supply"],
				"inflation":   totalReward(validators) / 1000000,
			},
		},
	}, nil
}

// HandleAction 处理追加质押交互并重新计算权重。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	validators := decodeValidators(state)
	targetID := framework.NormalizeDashedID("validator", framework.StringValue(input.Params["resource_id"], "validator-a"), "validator-a")
	stake := framework.NumberValue(input.Params["stake"], 100)
	for index := range validators {
		if validators[index].ID == targetID {
			validators[index].Stake += stake
		}
	}
	epoch := int(framework.NumberValue(state.Data["epoch"], 1))
	selectedID := framework.StringValue(state.Data["selected_validator"], "")
	if err := rebuildState(state, validators, epoch, "质押汇总", selectedID); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "追加质押", fmt.Sprintf("%s 追加质押 %.0f。", targetID, stake), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":   state.Data["validators"],
				"total_stake":  state.Data["total_stake"],
				"selected":     selectedID,
				"voting_power": totalStake(validators),
			},
		},
	}, nil
}

// BuildRenderState 输出质押权重、选举结果和奖励数据。
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

// SyncSharedState 在质押、供应量和治理共享状态变化后重建 PoS 验证者场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	validators := decodeValidators(state)
	epoch := int(framework.NumberValue(state.Data["epoch"], 1))
	selectedID := framework.StringValue(state.Data["selected_validator"], "")
	epoch, selectedID = applySharedValidatorState(validators, sharedState, epoch, selectedID)
	return rebuildState(state, validators, epoch, framework.StringValue(state.Data["phase_name"], state.Phase), selectedID)
}

// validatorState 保存验证者质押、权重、奖励和惩罚状态。
type validatorState struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Stake   float64 `json:"stake"`
	Weight  float64 `json:"weight"`
	Reward  float64 `json:"reward"`
	Slashed bool    `json:"slashed"`
	VRFSeed string  `json:"vrf_seed"`
}

// defaultValidators 创建默认验证者集合。
func defaultValidators() []validatorState {
	return []validatorState{
		{ID: "validator-a", Label: "Validator-A", Stake: 120, VRFSeed: "seed-a"},
		{ID: "validator-b", Label: "Validator-B", Stake: 260, VRFSeed: "seed-b"},
		{ID: "validator-c", Label: "Validator-C", Stake: 180, VRFSeed: "seed-c"},
		{ID: "validator-d", Label: "Validator-D", Stake: 90, VRFSeed: "seed-d"},
	}
}

// rebuildState 根据质押权重重建可视化状态。
func rebuildState(state *framework.SceneState, validators []validatorState, epoch int, phase string, selectedID string) error {
	total := totalStake(validators)
	for index := range validators {
		if total > 0 {
			validators[index].Weight = validators[index].Stake / total
		}
	}
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes = make([]framework.Node, 0, len(validators))
	for index, validator := range validators {
		status := "normal"
		if validator.ID == selectedID {
			status = "success"
		}
		if validator.Slashed {
			status = "slashed"
		}
		state.Nodes = append(state.Nodes, framework.Node{
			ID:     validator.ID,
			Label:  validator.Label,
			Status: status,
			Role:   "validator",
			X:      150 + float64(index%2)*280,
			Y:      150 + float64(index/2)*170,
			Stake:  validator.Stake,
			Load:   validator.Weight * 100,
			Attributes: map[string]any{
				"stake":   validator.Stake,
				"weight":  validator.Weight,
				"reward":  validator.Reward,
				"slashed": validator.Slashed,
			},
		})
	}
	state.Messages = []framework.Message{
		{ID: "pos-randomness", Label: "VRF Randomness", Kind: "vote", Status: phase, SourceID: "validator-a", TargetID: selectedID},
	}
	state.Metrics = []framework.Metric{
		{Key: "epoch", Label: "Epoch", Value: fmt.Sprintf("%d", epoch), Tone: "info"},
		{Key: "total_stake", Label: "总质押", Value: fmt.Sprintf("%.0f", total), Tone: "success"},
		{Key: "selected", Label: "当选验证者", Value: selectedLabel(validators, selectedID), Tone: "warning"},
		{Key: "vrf", Label: "VRF 种子", Value: selectedVRF(validators, selectedID), Tone: "info"},
		{Key: "reward", Label: "累计奖励", Value: fmt.Sprintf("%.1f", totalReward(validators)), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "选择规则", Value: "基于质押权重的确定性随机抽样"},
		{Label: "VRF", Value: selectedVRF(validators, selectedID)},
	}
	state.Data = map[string]any{
		"phase_name":         phase,
		"epoch":              epoch,
		"validators":         validators,
		"selected_validator": selectedID,
		"total_stake":        total,
		"token_supply":       1000000 + totalReward(validators),
	}
	state.Extra = map[string]any{
		"description": "该场景实现质押权重、确定性随机选举、Epoch 轮转和奖励结算。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeValidators 从通用 JSON 状态恢复验证者集合。
func decodeValidators(state *framework.SceneState) []validatorState {
	raw, ok := state.Data["validators"].([]any)
	if !ok {
		if typed, ok := state.Data["validators"].([]validatorState); ok {
			return typed
		}
		return defaultValidators()
	}
	result := make([]validatorState, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, validatorState{
			ID:      framework.StringValue(entry["id"], ""),
			Label:   framework.StringValue(entry["label"], ""),
			Stake:   framework.NumberValue(entry["stake"], 0),
			Weight:  framework.NumberValue(entry["weight"], 0),
			Reward:  framework.NumberValue(entry["reward"], 0),
			Slashed: framework.BoolValue(entry["slashed"], false),
		})
	}
	if len(result) == 0 {
		return defaultValidators()
	}
	return result
}

// weightedSelect 按质押权重执行确定性随机抽样。
func weightedSelect(validators []validatorState, epoch int) string {
	total := totalStake(validators)
	if total == 0 {
		return validators[0].ID
	}
	seed := sha256.Sum256([]byte(fmt.Sprintf("epoch-%d", epoch)))
	point := float64(binary.BigEndian.Uint64(seed[:8])%uint64(total*1000)) / 1000
	acc := 0.0
	for _, validator := range validators {
		acc += validator.Stake
		if point <= acc {
			return validator.ID
		}
	}
	return validators[len(validators)-1].ID
}

// rewardValidator 给当选验证者发放奖励。
func rewardValidator(validators []validatorState, selectedID string) {
	for index := range validators {
		if validators[index].ID == selectedID {
			validators[index].Reward += 2.5
			validators[index].Stake += 2.5
		}
	}
}

// totalStake 统计当前有效总质押。
func totalStake(validators []validatorState) float64 {
	total := 0.0
	for _, validator := range validators {
		if !validator.Slashed {
			total += validator.Stake
		}
	}
	return total
}

// totalReward 统计累计奖励。
func totalReward(validators []validatorState) float64 {
	total := 0.0
	for _, validator := range validators {
		total += validator.Reward
	}
	return total
}

// selectedVRF 返回当选验证者的 VRF 种子展示文本。
func selectedVRF(validators []validatorState, selectedID string) string {
	if selectedID == "" {
		return "未抽样"
	}
	for _, validator := range validators {
		if validator.ID == selectedID {
			if strings.TrimSpace(validator.VRFSeed) != "" {
				return validator.VRFSeed
			}
			return "seed-missing"
		}
	}
	return selectedID
}

// selectedLabel 返回当选验证者展示标签。
func selectedLabel(validators []validatorState, selectedID string) string {
	if selectedID == "" {
		return "未选择"
	}
	for _, validator := range validators {
		if validator.ID == selectedID {
			return validator.Label
		}
	}
	return selectedID
}

// nextPhase 返回 PoS 选举流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "质押汇总":
		return "随机抽样"
	case "随机抽样":
		return "Epoch 轮转"
	case "Epoch 轮转":
		return "奖励结算"
	default:
		return "质押汇总"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "质押汇总":
		return 0
	case "随机抽样":
		return 1
	case "Epoch 轮转":
		return 2
	case "奖励结算":
		return 3
	default:
		return 0
	}
}

// applySharedValidatorState 将 PoS 经济组中的质押与治理共享状态映射回验证者选举场景。
func applySharedValidatorState(validators []validatorState, sharedState map[string]any, epoch int, selectedID string) (int, string) {
	if len(sharedState) == 0 {
		return epoch, selectedID
	}
	if stakes, ok := sharedState["stakes"].(map[string]any); ok {
		if sharedValidators, ok := stakes["validators"].([]any); ok {
			for _, item := range sharedValidators {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				validatorID := framework.StringValue(entry["id"], "")
				for index := range validators {
					if validators[index].ID == validatorID {
						validators[index].Stake = framework.NumberValue(entry["stake"], validators[index].Stake)
						validators[index].Reward = framework.NumberValue(entry["reward"], validators[index].Reward)
						validators[index].Slashed = framework.BoolValue(entry["slashed"], validators[index].Slashed)
					}
				}
			}
		}
	}
	if sharedEpoch := int(framework.NumberValue(sharedState["epoch"], float64(epoch))); sharedEpoch > epoch {
		epoch = sharedEpoch
	}
	if sharedSelected := framework.StringValue(sharedState["selected"], ""); sharedSelected != "" {
		selectedID = sharedSelected
	}
	return epoch, selectedID
}
