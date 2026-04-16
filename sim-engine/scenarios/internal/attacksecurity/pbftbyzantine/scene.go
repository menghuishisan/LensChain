package pbftbyzantine

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// replicaCount 是拜占庭攻击场景中的 PBFT 副本数。
	replicaCount = 4
	// faultLimit 是当前副本规模下允许的最大拜占庭节点数。
	faultLimit = 1
)

// DefaultState 构造 PBFT 拜占庭攻击场景的初始状态。
func DefaultState() framework.SceneState {
	nodes := make([]framework.Node, 0, replicaCount)
	for index := 0; index < replicaCount; index++ {
		nodes = append(nodes, framework.Node{
			ID:     fmt.Sprintf("replica-%d", index),
			Label:  fmt.Sprintf("Replica-%d", index),
			Status: "normal",
			Role:   "replica",
			X:      140 + float64(index)*160,
			Y:      220,
		})
	}
	return framework.SceneState{
		SceneCode:    "pbft-byzantine",
		Title:        "PBFT 拜占庭攻击",
		Phase:        "恶意消息注入",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   18,
		Stages:       []string{"恶意消息注入", "投票偏差", "触发视图切换"},
		Nodes:        nodes,
		ChangedKeys:  []string{"nodes", "data", "metrics"},
		Data:         map[string]any{},
		Extra:        map[string]any{},
	}
}

// Init 初始化副本状态、法定票阈值和默认恶意节点。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := attackModel{
		View:          0,
		Sequence:      1,
		Quorum:        3,
		ByzantineNode: "replica-2",
		VoteBias:      0,
		ViewChanged:   false,
	}
	return rebuildState(state, model, "恶意消息注入")
}

// Step 推进恶意消息注入、投票偏差和视图切换流程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedAttackState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "恶意消息注入"))
	switch phase {
	case "恶意消息注入":
		model.VoteBias = 1
	case "投票偏差":
		model.VoteBias = 2
	case "触发视图切换":
		model.View++
		model.ViewChanged = true
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("异常共识流程进入%s阶段。", phase), phaseTone(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"byzantine": model.ByzantineNode,
				"isolated":  isolatedReplicaIDs(model.Partitioned),
			},
			"messages": state.Data["message_flow"],
			"view":     model.View,
		},
	}, nil
}

// HandleAction 允许指定拜占庭副本伪造投票。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.ByzantineNode = framework.NormalizeDashedID("replica", framework.StringValue(input.Params["resource_id"], model.ByzantineNode), model.ByzantineNode)
	model.VoteBias = 2
	if err := rebuildState(state, model, "投票偏差"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "伪造投票", fmt.Sprintf("%s 已发送冲突投票。", model.ByzantineNode), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes":    map[string]any{"byzantine": model.ByzantineNode, "isolated": isolatedReplicaIDs(model.Partitioned)},
			"messages": state.Data["message_flow"],
		},
	}, nil
}

// BuildRenderState 输出异常消息流、容错阈值和视图切换结果。
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

// SyncSharedState 在 PBFT 攻击组共享状态变化后重建拜占庭攻击场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedAttackState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// attackModel 保存拜占庭节点、投票偏差和视图切换状态。
type attackModel struct {
	View          int    `json:"view"`
	Sequence      int    `json:"sequence"`
	Quorum        int    `json:"quorum"`
	ByzantineNode string `json:"byzantine_node"`
	VoteBias      int    `json:"vote_bias"`
	ViewChanged   bool   `json:"view_changed"`
	Partitioned   bool   `json:"partitioned"`
}

// rebuildState 将攻击模型转为副本状态、异常消息流和容错指标。
func rebuildState(state *framework.SceneState, model attackModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Messages = buildMessages(model, phase)
	for index := range state.Nodes {
		node := &state.Nodes[index]
		node.Status = "normal"
		node.Role = "replica"
		if node.ID == model.ByzantineNode {
			node.Status = "byzantine"
			node.Role = "faulty"
		}
		if phase == "触发视图切换" && node.ID == "replica-0" {
			node.Status = "warning"
		}
		node.Attributes = map[string]any{
			"view":         model.View,
			"sequence":     model.Sequence,
			"vote_bias":    model.VoteBias,
			"is_byzantine": node.ID == model.ByzantineNode,
			"partitioned":  model.Partitioned,
		}
	}
	state.Metrics = []framework.Metric{
		{Key: "view", Label: "当前视图", Value: fmt.Sprintf("%d", model.View), Tone: "info"},
		{Key: "quorum", Label: "法定票数", Value: fmt.Sprintf("%d (f=%d)", model.Quorum, faultLimit), Tone: "warning"},
		{Key: "bias", Label: "异常票偏差", Value: fmt.Sprintf("%d", model.VoteBias), Tone: "warning"},
		{Key: "view_change", Label: "是否切换视图", Value: framework.BoolText(model.ViewChanged), Tone: viewTone(model.ViewChanged)},
		{Key: "partitioned", Label: "网络分区", Value: framework.BoolText(model.Partitioned), Tone: viewTone(model.Partitioned)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "拜占庭节点", Value: model.ByzantineNode},
		{Label: "阶段", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":   phase,
		"attack_model": model,
		"message_flow": state.Messages,
	}
	state.Extra = map[string]any{
		"description": "该场景实现恶意副本伪造投票、法定票偏差和视图切换。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复攻击模型。
func decodeModel(state *framework.SceneState) attackModel {
	entry, ok := state.Data["attack_model"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["attack_model"].(attackModel); ok {
			return typed
		}
		return attackModel{View: 0, Sequence: 1, Quorum: 3, ByzantineNode: "replica-2"}
	}
	return attackModel{
		View:          int(framework.NumberValue(entry["view"], 0)),
		Sequence:      int(framework.NumberValue(entry["sequence"], 1)),
		Quorum:        int(framework.NumberValue(entry["quorum"], 3)),
		ByzantineNode: framework.StringValue(entry["byzantine_node"], "replica-2"),
		VoteBias:      int(framework.NumberValue(entry["vote_bias"], 0)),
		ViewChanged:   framework.BoolValue(entry["view_changed"], false),
		Partitioned:   framework.BoolValue(entry["partitioned"], false),
	}
}

// buildMessages 生成正常投票与异常投票的消息流。
func buildMessages(model attackModel, phase string) []framework.Message {
	messages := make([]framework.Message, 0, 4)
	for index := 0; index < replicaCount; index++ {
		targetID := fmt.Sprintf("replica-%d", index)
		if targetID == model.ByzantineNode {
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("faulty-%d", index),
				Label:    "conflicting-vote",
				Kind:     "attack",
				Status:   phase,
				SourceID: model.ByzantineNode,
				TargetID: "replica-0",
			})
			continue
		}
		if targetID == "replica-0" {
			continue
		}
		messages = append(messages, framework.Message{
			ID:       fmt.Sprintf("vote-%d", index),
			Label:    "prepare-vote",
			Kind:     "attack",
			Status:   phase,
			SourceID: targetID,
			TargetID: "replica-0",
		})
	}
	return messages
}

// nextPhase 返回拜占庭攻击流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "恶意消息注入":
		return "投票偏差"
	case "投票偏差":
		return "触发视图切换"
	default:
		return "恶意消息注入"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "恶意消息注入":
		return 0
	case "投票偏差":
		return 1
	case "触发视图切换":
		return 2
	default:
		return 0
	}
}

// phaseTone 返回阶段事件色调。
func phaseTone(phase string) string {
	if phase == "触发视图切换" {
		return "warning"
	}
	return "info"
}

// viewTone 返回视图切换指标色调。
func viewTone(flag bool) string {
	if flag {
		return "warning"
	}
	return "info"
}

// applySharedAttackState 将共享状态中的拜占庭节点与视图切换结果映射回攻击场景。
// 这样 PBFT 共识面板中的操作会同步影响攻击面板展示。
func applySharedAttackState(model *attackModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if sharedView := int(framework.NumberValue(sharedState["view"], float64(model.View))); sharedView > model.View {
		model.View = sharedView
		model.ViewChanged = true
	}
	if nodes, ok := sharedState["nodes"].(map[string]any); ok {
		sharedByzantine := framework.NormalizeDashedID("replica", framework.StringValue(nodes["byzantine"], ""), "")
		if sharedByzantine != "" {
			model.ByzantineNode = sharedByzantine
		}
		model.Partitioned = len(framework.ToStringSlice(nodes["isolated"])) > 0
		if model.Partitioned {
			model.VoteBias = 2
			model.ViewChanged = true
		}
	}
}

// isolatedReplicaIDs 在 PBFT 攻击组中统一输出隔离副本列表。
func isolatedReplicaIDs(partitioned bool) []string {
	if !partitioned {
		return []string{}
	}
	return []string{"replica-3"}
}
