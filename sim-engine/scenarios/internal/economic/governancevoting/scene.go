package governancevoting

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// quorumVotes 是提案通过前需要达到的最低加权投票数。
	quorumVotes = 60.0
)

// DefaultState 构造链上治理投票场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "governance-voting",
		Title:        "链上治理投票",
		Phase:        "提出提案",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   18,
		Stages:       []string{"提出提案", "投票期", "达到法定人数", "执行提案"},
		Nodes: []framework.Node{
			{ID: "proposal", Label: "Proposal", Status: "active", Role: "proposal", X: 150, Y: 180},
			{ID: "voter-a", Label: "Voter-A", Status: "normal", Role: "voter", X: 320, Y: 120},
			{ID: "voter-b", Label: "Voter-B", Status: "normal", Role: "voter", X: 520, Y: 120},
			{ID: "executor", Label: "Executor", Status: "normal", Role: "executor", X: 340, Y: 320},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化提案和默认投票权重。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := votingModel{
		ProposalID:    "proposal-1",
		Status:        "draft",
		Quorum:        quorumVotes,
		YesVotes:      0,
		NoVotes:       0,
		Executed:      false,
		Voters:        []voter{{ID: "voter-a", Label: "Voter-A", Weight: 35}, {ID: "voter-b", Label: "Voter-B", Weight: 30}, {ID: "delegator", Label: "Delegator", Weight: 20}},
		ExecutionNote: "",
	}
	applySharedGovernanceState(&model, input.SharedState)
	return rebuildState(state, model, "提出提案")
}

// Step 推进提案、投票、法定人数判断与执行。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedGovernanceState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "提出提案"))
	switch phase {
	case "提出提案":
		model.Status = "voting"
	case "投票期":
		if model.YesVotes == 0 && model.NoVotes == 0 {
			model.YesVotes += model.Voters[0].Weight
			model.NoVotes += model.Voters[1].Weight
		}
	case "达到法定人数":
		model.YesVotes += model.Voters[2].Weight
		if model.YesVotes >= model.Quorum {
			model.Status = "quorum_reached"
		}
	case "执行提案":
		model.Executed = model.YesVotes > model.NoVotes && model.YesVotes >= model.Quorum
		if model.Executed {
			model.Status = "executed"
			model.ExecutionNote = "Treasury payout approved"
		} else {
			model.Status = "rejected"
			model.ExecutionNote = "Proposal rejected by governance"
		}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("治理流程进入%s阶段。", phase), toneByStatus(model.Status))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"proposals": map[string]any{
				model.ProposalID: map[string]any{
					"status":   model.Status,
					"yes":      model.YesVotes,
					"no":       model.NoVotes,
					"executed": model.Executed,
				},
			},
			"stakes": map[string]any{
				"voting_power": totalVotes(model),
			},
		},
	}, nil
}

// HandleAction 处理显式投票操作。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	choice := strings.ToLower(framework.StringValue(input.Params["choice"], "yes"))
	voterID := framework.StringValue(input.ActorID, "voter-a")
	weight := voterWeight(model, voterID)
	if weight == 0 {
		weight = model.Voters[0].Weight
	}
	if choice == "yes" {
		model.YesVotes += weight
	} else {
		model.NoVotes += weight
	}
	model.Status = "voting"
	if err := rebuildState(state, model, "投票期"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "投票", fmt.Sprintf("%s 投出 %s，权重 %.0f。", voterID, choice, weight), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"proposals": map[string]any{
				model.ProposalID: map[string]any{
					"yes": model.YesVotes,
					"no":  model.NoVotes,
				},
			},
		},
	}, nil
}

// BuildRenderState 输出提案状态、投票分布和执行结果。
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

// SyncSharedState 在治理提案、投票权和供应量联动变化后重建治理投票场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedGovernanceState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// votingModel 保存提案状态、加权投票和执行结果。
type votingModel struct {
	ProposalID    string  `json:"proposal_id"`
	Status        string  `json:"status"`
	Quorum        float64 `json:"quorum"`
	YesVotes      float64 `json:"yes_votes"`
	NoVotes       float64 `json:"no_votes"`
	Executed      bool    `json:"executed"`
	ExecutionNote string  `json:"execution_note"`
	Voters        []voter `json:"voters"`
}

// voter 保存治理参与者及其投票权重。
type voter struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
}

// rebuildState 将治理模型转换为节点、投票消息和指标。
func rebuildState(state *framework.SceneState, model votingModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[0].Status = proposalStatus(model)
	state.Nodes[0].Load = model.YesVotes + model.NoVotes
	if len(model.Voters) > 0 {
		state.Nodes[1].Load = model.Voters[0].Weight
	}
	if len(model.Voters) > 1 {
		state.Nodes[2].Load = model.Voters[1].Weight
	}
	state.Nodes[3].Status = executorStatus(model)
	state.Messages = []framework.Message{
		{ID: "gov-vote-yes", Label: "Yes Votes", Kind: "proposal", Status: phase, SourceID: "voter-a", TargetID: "proposal"},
		{ID: "gov-vote-no", Label: "No Votes", Kind: "proposal", Status: phase, SourceID: "voter-b", TargetID: "proposal"},
	}
	state.Metrics = []framework.Metric{
		{Key: "yes", Label: "赞成票", Value: fmt.Sprintf("%.0f", model.YesVotes), Tone: "success"},
		{Key: "no", Label: "反对票", Value: fmt.Sprintf("%.0f", model.NoVotes), Tone: "warning"},
		{Key: "quorum", Label: "法定人数", Value: fmt.Sprintf("%.0f", model.Quorum), Tone: "info"},
		{Key: "status", Label: "提案状态", Value: model.Status, Tone: toneByStatus(model.Status)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "提案", Value: model.ProposalID},
		{Label: "执行说明", Value: fallbackText(model.ExecutionNote, "尚未执行")},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"governance": model,
	}
	state.Extra = map[string]any{
		"description": "该场景实现提案创建、加权投票、法定人数判断和执行结果。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复治理模型。
func decodeModel(state *framework.SceneState) votingModel {
	entry, ok := state.Data["governance"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["governance"].(votingModel); ok {
			return typed
		}
		return votingModel{ProposalID: "proposal-1", Status: "draft", Quorum: quorumVotes, Voters: []voter{{ID: "voter-a", Label: "Voter-A", Weight: 35}, {ID: "voter-b", Label: "Voter-B", Weight: 30}, {ID: "delegator", Label: "Delegator", Weight: 20}}}
	}
	return votingModel{
		ProposalID:    framework.StringValue(entry["proposal_id"], "proposal-1"),
		Status:        framework.StringValue(entry["status"], "draft"),
		Quorum:        framework.NumberValue(entry["quorum"], quorumVotes),
		YesVotes:      framework.NumberValue(entry["yes_votes"], 0),
		NoVotes:       framework.NumberValue(entry["no_votes"], 0),
		Executed:      framework.BoolValue(entry["executed"], false),
		ExecutionNote: framework.StringValue(entry["execution_note"], ""),
		Voters:        decodeVoters(entry["voters"]),
	}
}

// decodeVoters 恢复投票者列表。
func decodeVoters(value any) []voter {
	raw, ok := value.([]any)
	if !ok {
		return []voter{{ID: "voter-a", Label: "Voter-A", Weight: 35}, {ID: "voter-b", Label: "Voter-B", Weight: 30}, {ID: "delegator", Label: "Delegator", Weight: 20}}
	}
	result := make([]voter, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, voter{
			ID:     framework.StringValue(entry["id"], ""),
			Label:  framework.StringValue(entry["label"], ""),
			Weight: framework.NumberValue(entry["weight"], 0),
		})
	}
	if len(result) == 0 {
		return []voter{{ID: "voter-a", Label: "Voter-A", Weight: 35}, {ID: "voter-b", Label: "Voter-B", Weight: 30}, {ID: "delegator", Label: "Delegator", Weight: 20}}
	}
	return result
}

// voterWeight 返回指定投票者权重。
func voterWeight(model votingModel, voterID string) float64 {
	for _, current := range model.Voters {
		if current.ID == voterID {
			return current.Weight
		}
	}
	return 0
}

// totalVotes 返回已配置投票权总和。
func totalVotes(model votingModel) float64 {
	total := 0.0
	for _, current := range model.Voters {
		total += current.Weight
	}
	return total
}

// nextPhase 返回治理流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "提出提案":
		return "投票期"
	case "投票期":
		return "达到法定人数"
	case "达到法定人数":
		return "执行提案"
	default:
		return "提出提案"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "提出提案":
		return 0
	case "投票期":
		return 1
	case "达到法定人数":
		return 2
	case "执行提案":
		return 3
	default:
		return 0
	}
}

// proposalStatus 返回提案节点状态。
func proposalStatus(model votingModel) string {
	switch model.Status {
	case "executed":
		return "success"
	case "rejected":
		return "warning"
	default:
		return "active"
	}
}

// executorStatus 返回执行节点状态。
func executorStatus(model votingModel) string {
	if model.Executed {
		return "success"
	}
	return "normal"
}

// toneByStatus 返回提案状态对应的指标色调。
func toneByStatus(status string) string {
	switch status {
	case "executed", "quorum_reached":
		return "success"
	case "rejected":
		return "warning"
	default:
		return "info"
	}
}

// fallbackText 在文本为空时返回替代文案。
func fallbackText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// applySharedGovernanceState 将 PoS 经济组中的质押权与提案状态映射回治理投票场景。
func applySharedGovernanceState(model *votingModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if stakes, ok := sharedState["stakes"].(map[string]any); ok {
		if votingPower := framework.NumberValue(stakes["voting_power"], 0); votingPower > 0 {
			model.Quorum = framework.Clamp(votingPower*0.7, quorumVotes, votingPower)
		}
	}
	if proposals, ok := sharedState["proposals"].(map[string]any); ok {
		if proposal, ok := proposals[model.ProposalID].(map[string]any); ok {
			model.Status = framework.StringValue(proposal["status"], model.Status)
			model.YesVotes = framework.NumberValue(proposal["yes"], model.YesVotes)
			model.NoVotes = framework.NumberValue(proposal["no"], model.NoVotes)
			model.Executed = framework.BoolValue(proposal["executed"], model.Executed)
		}
	}
}
