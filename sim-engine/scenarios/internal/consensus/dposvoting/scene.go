package dposvoting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 DPoS 委托投票场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "dpos-voting",
		Title:        "DPoS 委托投票",
		Phase:        "委托投票",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   18,
		Stages:       []string{"委托投票", "权重汇总", "超级节点排名", "轮次出块"},
		Nodes: []framework.Node{
			{ID: "delegate-1", Label: "Delegate-1", Status: "normal", Role: "delegate", X: 120, Y: 130},
			{ID: "delegate-2", Label: "Delegate-2", Status: "normal", Role: "delegate", X: 120, Y: 270},
			{ID: "delegate-3", Label: "Delegate-3", Status: "normal", Role: "delegate", X: 320, Y: 100},
			{ID: "delegate-4", Label: "Delegate-4", Status: "normal", Role: "delegate", X: 320, Y: 300},
			{ID: "delegate-5", Label: "Delegate-5", Status: "normal", Role: "delegate", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化代表票权和超级节点候选状态。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := votingModel{
		Delegates: []delegate{
			{ID: "delegate-1", Label: "Delegate-1", Votes: 38, Support: 0.38},
			{ID: "delegate-2", Label: "Delegate-2", Votes: 26, Support: 0.26},
			{ID: "delegate-3", Label: "Delegate-3", Votes: 31, Support: 0.31},
			{ID: "delegate-4", Label: "Delegate-4", Votes: 22, Support: 0.22},
			{ID: "delegate-5", Label: "Delegate-5", Votes: 18, Support: 0.18},
		},
		ProducerSlots: []string{},
		RoundIndex:    0,
	}
	return rebuildState(state, model, "委托投票")
}

// Step 推进权重汇总、排名和轮次出块。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "委托投票"))
	switch phase {
	case "权重汇总":
		sortDelegates(model.Delegates)
	case "超级节点排名":
		sortDelegates(model.Delegates)
		model.ProducerSlots = topDelegates(model.Delegates, 3)
	case "轮次出块":
		if len(model.ProducerSlots) == 0 {
			model.ProducerSlots = topDelegates(model.Delegates, 3)
		}
		model.RoundIndex = (model.RoundIndex + 1) % len(model.ProducerSlots)
		recordProducerBlock(&model)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("DPoS 流程进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":   delegateList(model),
				"total_stake":  totalVotes(model),
				"selected":     currentProducer(model),
				"voting_power": totalVotes(model),
			},
			"proposals": map[string]any{
				"active":   len(model.ProducerSlots) > 0,
				"rankings": delegateList(model),
			},
		},
	}, nil
}

// HandleAction 允许将票权重新委托给某个代表。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	target := framework.NormalizeSlug(framework.StringValue(input.Params["resource_id"], "Delegate-1"), "delegate-1")
	for index := range model.Delegates {
		if model.Delegates[index].ID == target {
			model.Delegates[index].Votes += 8
			model.Delegates[index].Support = supportRatio(model.Delegates[index].Votes)
		}
	}
	sortDelegates(model.Delegates)
	model.ProducerSlots = topDelegates(model.Delegates, 3)
	if err := rebuildState(state, model, "权重汇总"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "重新投票", fmt.Sprintf("已将新增票权委托给 %s。", target), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"stakes": map[string]any{
				"validators":   delegateList(model),
				"total_stake":  totalVotes(model),
				"selected":     currentProducer(model),
				"voting_power": totalVotes(model),
			},
		},
	}, nil
}

// BuildRenderState 输出票权排名、超级节点和当前出块槽位。
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

// SyncSharedState 在 PoS 经济组共享投票权变化后重建 DPoS 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedDPOSState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// votingModel 保存代表票数、超级节点列表和出块轮次。
type votingModel struct {
	Delegates     []delegate `json:"delegates"`
	ProducerSlots []string   `json:"producer_slots"`
	RoundIndex    int        `json:"round_index"`
}

// delegate 保存单个代表节点的票权。
type delegate struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Votes   float64 `json:"votes"`
	Support float64 `json:"support"`
	Blocks  int     `json:"blocks"`
}

// rebuildState 将投票模型映射为排名、节点高亮和指标。
func rebuildState(state *framework.SceneState, model votingModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	sortDelegates(model.Delegates)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = model.Delegates[index].Votes
	}
	for index, producer := range model.ProducerSlots {
		for nodeIndex := range state.Nodes {
			if state.Nodes[nodeIndex].ID != producer {
				continue
			}
			state.Nodes[nodeIndex].Status = "success"
			if phase == "轮次出块" && index == model.RoundIndex {
				state.Nodes[nodeIndex].Status = "active"
			}
		}
	}
	state.Messages = buildMessages(model, phase)
	state.Metrics = []framework.Metric{
		{Key: "top_delegate", Label: "最高票代表", Value: model.Delegates[0].Label, Tone: "success"},
		{Key: "top_votes", Label: "最高票数", Value: framework.MetricValue(model.Delegates[0].Votes, ""), Tone: "info"},
		{Key: "support", Label: "最高支持率", Value: fmt.Sprintf("%.0f%%", model.Delegates[0].Support*100), Tone: "warning"},
		{Key: "producers", Label: "超级节点", Value: strings.Join(model.ProducerSlots, ", "), Tone: "warning"},
		{Key: "round_index", Label: "当前轮次槽位", Value: fmt.Sprintf("%d", model.RoundIndex+1), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "超级节点数量", Value: fmt.Sprintf("%d", len(model.ProducerSlots))},
		{Label: "当前出块者", Value: currentProducer(model)},
		{Label: "票权总量", Value: framework.MetricValue(totalVotes(model), "")},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"dpos_voting": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 DPoS 中票权委托、超级节点选出和轮次出块。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 DPoS 投票模型。
func decodeModel(state *framework.SceneState) votingModel {
	entry, ok := state.Data["dpos_voting"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["dpos_voting"].(votingModel); ok {
			return typed
		}
		return votingModel{
			Delegates: []delegate{
				{ID: "delegate-1", Label: "Delegate-1", Votes: 38, Support: 0.38},
				{ID: "delegate-2", Label: "Delegate-2", Votes: 26, Support: 0.26},
				{ID: "delegate-3", Label: "Delegate-3", Votes: 31, Support: 0.31},
				{ID: "delegate-4", Label: "Delegate-4", Votes: 22, Support: 0.22},
				{ID: "delegate-5", Label: "Delegate-5", Votes: 18, Support: 0.18},
			},
		}
	}
	return votingModel{
		Delegates:     decodeDelegates(entry["delegates"]),
		ProducerSlots: framework.ToStringSlice(entry["producer_slots"]),
		RoundIndex:    int(framework.NumberValue(entry["round_index"], 0)),
	}
}

// applySharedDPOSState 将 PoS 经济组共享质押和治理结果映射到 DPoS 票权模型。
func applySharedDPOSState(model *votingModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if stakes, ok := sharedState["stakes"].(map[string]any); ok {
		if votingPower, ok := stakes["voting_power"]; ok {
			total := framework.NumberValue(votingPower, totalVotes(*model))
			if len(model.Delegates) > 0 {
				share := total / float64(len(model.Delegates))
				for index := range model.Delegates {
					model.Delegates[index].Votes = framework.Clamp(model.Delegates[index].Votes+share*0.05, 1, total)
					model.Delegates[index].Support = supportRatio(model.Delegates[index].Votes)
				}
			}
		}
	}
	if proposals, ok := sharedState["proposals"].(map[string]any); ok && len(proposals) > 0 {
		sortDelegates(model.Delegates)
		model.ProducerSlots = topDelegates(model.Delegates, 3)
	}
}

// buildMessages 构造票权流向和超级节点出块消息。
func buildMessages(model votingModel, phase string) []framework.Message {
	messages := make([]framework.Message, 0, len(model.Delegates))
	for _, current := range model.Delegates {
		messages = append(messages, framework.Message{
			ID:       current.ID + "-" + phase,
			Label:    "VoteWeight",
			Kind:     "vote",
			Status:   phase,
			SourceID: current.ID,
			TargetID: current.ID,
		})
	}
	if phase == "轮次出块" && len(model.ProducerSlots) > 0 {
		current := model.ProducerSlots[model.RoundIndex]
		messages = append(messages, framework.Message{
			ID:       "producer-slot",
			Label:    "Produce Block",
			Kind:     "vote",
			Status:   "active",
			SourceID: current,
			TargetID: current,
		})
	}
	return messages
}

// sortDelegates 按票权从高到低排序。
func sortDelegates(values []delegate) {
	sort.Slice(values, func(i int, j int) bool {
		return values[i].Votes > values[j].Votes
	})
}

// topDelegates 返回票权前 N 名的代表标识。
func topDelegates(values []delegate, limit int) []string {
	size := limit
	if len(values) < size {
		size = len(values)
	}
	result := make([]string, 0, size)
	for index := 0; index < size; index++ {
		result = append(result, values[index].ID)
	}
	return result
}

// delegateList 生成共享状态所需的票权列表。
func delegateList(model votingModel) []map[string]any {
	result := make([]map[string]any, 0, len(model.Delegates))
	for _, current := range model.Delegates {
		result = append(result, map[string]any{
			"id":      current.ID,
			"votes":   current.Votes,
			"support": current.Support,
			"blocks":  current.Blocks,
		})
	}
	return result
}

// totalVotes 返回总票权。
func totalVotes(model votingModel) float64 {
	total := 0.0
	for _, current := range model.Delegates {
		total += current.Votes
	}
	return total
}

// decodeDelegates 恢复代表票权切片。
func decodeDelegates(value any) []delegate {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]delegate); ok {
			return append([]delegate(nil), typed...)
		}
		return []delegate{
			{ID: "delegate-1", Label: "Delegate-1", Votes: 38, Support: 0.38},
			{ID: "delegate-2", Label: "Delegate-2", Votes: 26, Support: 0.26},
			{ID: "delegate-3", Label: "Delegate-3", Votes: 31, Support: 0.31},
			{ID: "delegate-4", Label: "Delegate-4", Votes: 22, Support: 0.22},
			{ID: "delegate-5", Label: "Delegate-5", Votes: 18, Support: 0.18},
		}
	}
	result := make([]delegate, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, delegate{
			ID:      framework.StringValue(entry["id"], ""),
			Label:   framework.StringValue(entry["label"], ""),
			Votes:   framework.NumberValue(entry["votes"], 0),
			Support: framework.NumberValue(entry["support"], 0),
			Blocks:  int(framework.NumberValue(entry["blocks"], 0)),
		})
	}
	if len(result) == 0 {
		return []delegate{
			{ID: "delegate-1", Label: "Delegate-1", Votes: 38, Support: 0.38},
			{ID: "delegate-2", Label: "Delegate-2", Votes: 26, Support: 0.26},
			{ID: "delegate-3", Label: "Delegate-3", Votes: 31, Support: 0.31},
			{ID: "delegate-4", Label: "Delegate-4", Votes: 22, Support: 0.22},
			{ID: "delegate-5", Label: "Delegate-5", Votes: 18, Support: 0.18},
		}
	}
	return result
}


// nextPhase 返回 DPoS 投票流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "委托投票":
		return "权重汇总"
	case "权重汇总":
		return "超级节点排名"
	case "超级节点排名":
		return "轮次出块"
	default:
		return "委托投票"
	}
}

// phaseIndex 将阶段映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "委托投票":
		return 0
	case "权重汇总":
		return 1
	case "超级节点排名":
		return 2
	case "轮次出块":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回 DPoS 不同阶段的展示色调。
func toneByPhase(phase string) string {
	switch phase {
	case "轮次出块":
		return "success"
	case "超级节点排名":
		return "warning"
	default:
		return "info"
	}
}

// supportRatio 返回票数对应的近似支持率。
func supportRatio(votes float64) float64 {
	if votes <= 0 {
		return 0
	}
	return framework.Clamp(votes/100, 0, 1)
}

// currentProducer 返回当前轮次出块代表。
func currentProducer(model votingModel) string {
	if len(model.ProducerSlots) == 0 {
		return ""
	}
	return model.ProducerSlots[model.RoundIndex%len(model.ProducerSlots)]
}

// recordProducerBlock 为当前轮次出块代表增加区块计数。
func recordProducerBlock(model *votingModel) {
	current := currentProducer(*model)
	for index := range model.Delegates {
		if model.Delegates[index].ID == current {
			model.Delegates[index].Blocks++
			break
		}
	}
}
