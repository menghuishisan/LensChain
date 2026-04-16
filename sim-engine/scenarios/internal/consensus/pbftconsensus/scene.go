package pbftconsensus

import (
	"fmt"
	"sort"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// replicaCount 是 PBFT Phase 1 场景的固定副本数量，n=4, f=1。
	replicaCount = 4
	// faultTolerance 是当前副本规模下可容忍的最大拜占庭节点数。
	faultTolerance = 1
	// quorumSize 是 Prepare/Commit 阶段需要达到的 2f+1 法定票数。
	quorumSize = 3
)

// DefaultState 构造 PBFT 三阶段共识场景的初始状态。
func DefaultState() framework.SceneState {
	nodes := make([]framework.Node, 0, replicaCount)
	for index := 0; index < replicaCount; index++ {
		role := "replica"
		if index == 0 {
			role = "primary"
		}
		nodes = append(nodes, framework.Node{
			ID:     fmt.Sprintf("replica-%d", index),
			Label:  fmt.Sprintf("Replica-%d", index),
			Status: "normal",
			Role:   role,
			X:      140 + float64(index)*160,
			Y:      220,
		})
	}
	return framework.SceneState{
		SceneCode:    "pbft-consensus",
		Title:        "PBFT 三阶段共识",
		Phase:        "Pre-prepare",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1600,
		TotalTicks:   25,
		Stages:       []string{"Pre-prepare", "Prepare", "Commit", "Checkpoint", "View Change"},
		Nodes:        nodes,
		ChangedKeys:  []string{"nodes", "data", "metrics"},
		Data:         map[string]any{},
		Extra:        map[string]any{},
	}
}

// Init 初始化 PBFT 运行期状态，包括视图、序号、检查点和水位线。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	replicas := defaultReplicas()
	return rebuildState(state, replicas, 0, 1, "Pre-prepare", false, false, "client-request")
}

// Step 推进 PBFT 共识状态机，覆盖三阶段、检查点和视图切换。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	replicas := decodeReplicas(state)
	view := int(framework.NumberValue(state.Data["view"], 0))
	sequence := int(framework.NumberValue(state.Data["sequence"], 1))
	phase := framework.StringValue(state.Data["phase_name"], "Pre-prepare")
	viewChangePending := framework.BoolValue(state.Data["view_change_pending"], false)
	checkpointReady := framework.BoolValue(state.Data["checkpoint_ready"], false)
	requestID := framework.StringValue(state.Data["request_id"], "client-request")
	var partitioned bool
	view, requestID, partitioned = applySharedConsensusState(replicas, input.SharedState, view, requestID)
	if partitioned && phase != "View Change" {
		viewChangePending = true
	}

	events := make([]framework.TimelineEvent, 0, 2)
	switch phase {
	case "Pre-prepare":
		for index := range replicas {
			replicas[index].ViewNumber = view
		}
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "主节点广播预准备", fmt.Sprintf("Primary 在视图 %d 为序号 %d 广播预准备消息。", view, sequence), "info"))
		phase = "Prepare"
	case "Prepare":
		prepareVotes := collectVotes(replicas, sequence, false)
		if prepareVotes >= quorumSize {
			events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Prepare 达到法定票数", fmt.Sprintf("收到 %d 份 Prepare，进入 Commit。", prepareVotes), "success"))
			phase = "Commit"
		} else {
			viewChangePending = true
			incrementTimeouts(replicas)
			events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Prepare 票数不足", fmt.Sprintf("仅收到 %d 份 Prepare，触发视图切换。", prepareVotes), "warning"))
			phase = "View Change"
		}
	case "Commit":
		commitVotes := collectVotes(replicas, sequence, true)
		if commitVotes >= quorumSize {
			checkpointReady = sequence%2 == 0
			if checkpointReady {
				phase = "Checkpoint"
				events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Commit 完成", fmt.Sprintf("序号 %d 已提交，准备广播检查点。", sequence), "success"))
			} else {
				sequence++
				phase = "Pre-prepare"
				events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Commit 完成", fmt.Sprintf("序号 %d 已提交，进入下一笔请求。", sequence-1), "success"))
			}
		} else {
			viewChangePending = true
			incrementTimeouts(replicas)
			phase = "View Change"
			events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Commit 票数不足", fmt.Sprintf("仅收到 %d 份 Commit，切换视图。", commitVotes), "warning"))
		}
	case "Checkpoint":
		checkpointReady = false
		sequence++
		markCheckpoint(replicas, sequence-1)
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "建立检查点", fmt.Sprintf("序号 %d 建立稳定检查点，并更新水位线。", sequence-1), "success"))
		phase = "Pre-prepare"
	case "View Change":
		view++
		viewChangePending = false
		rotatePrimary(replicas, view)
		resetTimeouts(replicas)
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "完成视图切换", fmt.Sprintf("进入新视图 %d，新主节点为 %s。", view, currentPrimary(replicas).Label), "warning"))
		phase = "Pre-prepare"
	default:
		phase = "Pre-prepare"
	}

	if err := rebuildState(state, replicas, view, sequence, phase, viewChangePending, checkpointReady, requestID); err != nil {
		return framework.StepOutput{}, err
	}
	return framework.StepOutput{
		Events: events,
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"byzantine": byzantineReplicaID(replicas),
				"isolated":  isolatedReplicaIDs(replicas),
			},
			"messages": state.Messages,
			"view":     view,
		},
	}, nil
}

// HandleAction 处理拜占庭注入和手动视图切换两类文档定义交互。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	replicas := decodeReplicas(state)
	view := int(framework.NumberValue(state.Data["view"], 0))
	sequence := int(framework.NumberValue(state.Data["sequence"], 1))
	phase := framework.StringValue(state.Data["phase_name"], "Pre-prepare")
	viewChangePending := framework.BoolValue(state.Data["view_change_pending"], false)
	checkpointReady := framework.BoolValue(state.Data["checkpoint_ready"], false)
	requestID := framework.StringValue(state.Data["request_id"], "client-request")

	var event framework.TimelineEvent
	switch input.ActionCode {
	case "inject_byzantine_node":
		targetID := framework.NormalizeDashedID("replica", framework.StringValue(input.Params["resource_id"], "replica-1"), "replica-1")
		for index := range replicas {
			if replicas[index].ID == targetID {
				replicas[index].Byzantine = true
				replicas[index].Status = "byzantine"
			}
		}
		event = framework.NewEvent(state.SceneCode, state.Tick, "注入拜占庭副本", fmt.Sprintf("%s 已切换为拜占庭节点。", targetID), "warning")
	case "trigger_view_change":
		viewChangePending = true
		phase = "View Change"
		nextView := int(framework.NumberValue(input.Params["view"], float64(view+1)))
		if nextView > view {
			view = nextView - 1
		}
		incrementTimeouts(replicas)
		event = framework.NewEvent(state.SceneCode, state.Tick, "手动触发视图切换", fmt.Sprintf("请求从视图 %d 切换到新主节点。", view), "warning")
	default:
		event = framework.NewEvent(state.SceneCode, state.Tick, input.ActionCode, "执行 PBFT 场景操作。", "info")
	}

	if err := rebuildState(state, replicas, view, sequence, phase, viewChangePending, checkpointReady, requestID); err != nil {
		return framework.ActionOutput{}, err
	}
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"byzantine": byzantineReplicaID(replicas),
				"isolated":  isolatedReplicaIDs(replicas),
			},
			"view": view,
		},
	}, nil
}

// BuildRenderState 返回 PBFT 场景对渲染层需要的完整过程化载荷。
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

// SyncSharedState 在共享副本、视图和网络状态变更后重建 PBFT 共识场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	replicas := decodeReplicas(state)
	view := int(framework.NumberValue(state.Data["view"], 0))
	sequence := int(framework.NumberValue(state.Data["sequence"], 1))
	phase := framework.StringValue(state.Data["phase_name"], "Pre-prepare")
	viewChangePending := framework.BoolValue(state.Data["view_change_pending"], false)
	checkpointReady := framework.BoolValue(state.Data["checkpoint_ready"], false)
	requestID := framework.StringValue(state.Data["request_id"], "client-request")
	var partitioned bool
	view, requestID, partitioned = applySharedConsensusState(replicas, sharedState, view, requestID)
	if partitioned && phase != "View Change" {
		viewChangePending = true
	}
	return rebuildState(state, replicas, view, sequence, phase, viewChangePending, checkpointReady, requestID)
}

// replicaState 保存 PBFT 副本在当前视图中的一致性状态。
type replicaState struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	IsPrimary       bool     `json:"is_primary"`
	Byzantine       bool     `json:"is_byzantine"`
	Partitioned     bool     `json:"is_partitioned"`
	PreparedSeqs    []int    `json:"prepared_seqs"`
	CommittedSeqs   []int    `json:"committed_seqs"`
	LastCheckpoint  int      `json:"last_checkpoint"`
	LowWatermark    int      `json:"low_watermark"`
	HighWatermark   int      `json:"high_watermark"`
	ViewNumber      int      `json:"view_number"`
	TimeoutCounter  int      `json:"timeout_counter"`
	CurrentMessages []string `json:"current_messages"`
	Status          string   `json:"status"`
}

// rebuildState 依据 PBFT 内部副本状态重建节点、消息、指标和事件面板数据。
func rebuildState(state *framework.SceneState, replicas []replicaState, view int, sequence int, phase string, viewChangePending bool, checkpointReady bool, requestID string) error {
	phaseIndex := stageIndex(phase)
	state.Phase = phase
	state.PhaseIndex = phaseIndex
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes = make([]framework.Node, 0, len(replicas))
	messages := buildMessages(replicas, phase, sequence, view)
	for index := range replicas {
		replicas[index].ViewNumber = view
	}
	for index, replica := range replicas {
		role := "replica"
		if replica.IsPrimary {
			role = "primary"
		}
		status := "normal"
		switch {
		case replica.Byzantine:
			status = "byzantine"
		case replica.Partitioned:
			status = "partitioned"
		case replica.IsPrimary:
			status = "active"
		case phase == "Commit" && containsInt(replica.CommittedSeqs, sequence):
			status = "success"
		case phase == "Prepare" && containsInt(replica.PreparedSeqs, sequence):
			status = "active"
		}
		state.Nodes = append(state.Nodes, framework.Node{
			ID:     replica.ID,
			Label:  replica.Label,
			Status: status,
			Role:   role,
			X:      140 + float64(index)*160,
			Y:      220,
			Load:   float64(len(replica.CurrentMessages) * 10),
			Attributes: map[string]any{
				"prepared_seqs":   replica.PreparedSeqs,
				"committed_seqs":  replica.CommittedSeqs,
				"checkpoint":      replica.LastCheckpoint,
				"low_watermark":   replica.LowWatermark,
				"high_watermark":  replica.HighWatermark,
				"view_number":     replica.ViewNumber,
				"timeout_counter": replica.TimeoutCounter,
				"is_byzantine":    replica.Byzantine,
				"is_partitioned":  replica.Partitioned,
				"current_message": replica.CurrentMessages,
			},
		})
	}
	prepareVotes := countVotes(replicas, sequence, false)
	commitVotes := countVotes(replicas, sequence, true)
	checkpoint := stableCheckpoint(replicas)
	state.Messages = messages
	state.Metrics = []framework.Metric{
		{Key: "view", Label: "当前视图", Value: fmt.Sprintf("%d", view), Tone: "info"},
		{Key: "sequence", Label: "请求序号", Value: fmt.Sprintf("%d", sequence), Tone: "info"},
		{Key: "prepare_votes", Label: "Prepare 票数", Value: fmt.Sprintf("%d / %d", prepareVotes, quorumSize), Tone: voteTone(prepareVotes)},
		{Key: "commit_votes", Label: "Commit 票数", Value: fmt.Sprintf("%d / %d", commitVotes, quorumSize), Tone: voteTone(commitVotes)},
		{Key: "checkpoint", Label: "稳定检查点", Value: fmt.Sprintf("%d", checkpoint), Tone: "success"},
		{Key: "timeouts", Label: "超时计数", Value: fmt.Sprintf("%d", totalTimeouts(replicas)), Tone: "warning"},
		{Key: "fault_tolerance", Label: "容错阈值", Value: fmt.Sprintf("f=%d", faultTolerance), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "当前阶段", Value: phase},
		{Label: "主节点", Value: currentPrimary(replicas).Label},
		{Label: "检查点", Value: fmt.Sprintf("%d", checkpoint)},
		{Label: "水位线", Value: fmt.Sprintf("[%d, %d]", replicas[0].LowWatermark, replicas[0].HighWatermark)},
	}
	state.Data = map[string]any{
		"view":                view,
		"sequence":            sequence,
		"phase_name":          phase,
		"request_id":          requestID,
		"replicas":            replicas,
		"prepare_votes":       prepareVotes,
		"commit_votes":        commitVotes,
		"quorum_size":         quorumSize,
		"fault_tolerance":     faultTolerance,
		"stable_checkpoint":   checkpoint,
		"low_watermark":       replicas[0].LowWatermark,
		"high_watermark":      replicas[0].HighWatermark,
		"view_change_pending": viewChangePending,
		"checkpoint_ready":    checkpointReady,
	}
	state.Extra = map[string]any{
		"description": "该场景实现 Pre-prepare/Prepare/Commit、检查点、水位线和视图切换，不再是阶段轮播模板。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// defaultReplicas 创建 PBFT 场景默认副本集合。
func defaultReplicas() []replicaState {
	result := make([]replicaState, 0, replicaCount)
	for index := 0; index < replicaCount; index++ {
		result = append(result, replicaState{
			ID:             fmt.Sprintf("replica-%d", index),
			Label:          fmt.Sprintf("Replica-%d", index),
			IsPrimary:      index == 0,
			PreparedSeqs:   []int{},
			CommittedSeqs:  []int{},
			LastCheckpoint: 0,
			LowWatermark:   0,
			HighWatermark:  50,
			ViewNumber:     0,
			TimeoutCounter: 0,
			Status:         "normal",
		})
	}
	return result
}

// decodeReplicas 从通用状态对象恢复副本内部结构。
func decodeReplicas(state *framework.SceneState) []replicaState {
	raw, ok := state.Data["replicas"].([]any)
	if !ok {
		if typed, ok := state.Data["replicas"].([]replicaState); ok {
			return typed
		}
		return defaultReplicas()
	}
	result := make([]replicaState, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, replicaState{
			ID:              framework.StringValue(entry["id"], ""),
			Label:           framework.StringValue(entry["label"], ""),
			IsPrimary:       framework.BoolValue(entry["is_primary"], false),
			Byzantine:       framework.BoolValue(entry["is_byzantine"], false),
			Partitioned:     framework.BoolValue(entry["is_partitioned"], false),
			PreparedSeqs:    toIntSlice(entry["prepared_seqs"]),
			CommittedSeqs:   toIntSlice(entry["committed_seqs"]),
			LastCheckpoint:  int(framework.NumberValue(entry["last_checkpoint"], 0)),
			LowWatermark:    int(framework.NumberValue(entry["low_watermark"], 0)),
			HighWatermark:   int(framework.NumberValue(entry["high_watermark"], 50)),
			ViewNumber:      int(framework.NumberValue(entry["view_number"], 0)),
			TimeoutCounter:  int(framework.NumberValue(entry["timeout_counter"], 0)),
			CurrentMessages: framework.ToStringSlice(entry["current_messages"]),
			Status:          framework.StringValue(entry["status"], "normal"),
		})
	}
	if len(result) == 0 {
		return defaultReplicas()
	}
	return result
}

// buildMessages 按当前阶段生成 PBFT 消息流，供前端时序图和消息粒子渲染。
func buildMessages(replicas []replicaState, phase string, sequence int, view int) []framework.Message {
	primary := currentPrimary(replicas)
	messages := make([]framework.Message, 0, len(replicas)*2)
	switch phase {
	case "Pre-prepare":
		for _, replica := range replicas {
			if replica.ID == primary.ID {
				continue
			}
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("preprepare-%s-%d", replica.ID, sequence),
				Label:    fmt.Sprintf("PrePrepare(v=%d,n=%d)", view, sequence),
				Kind:     "vote",
				Status:   phase,
				SourceID: primary.ID,
				TargetID: replica.ID,
			})
		}
	case "Prepare":
		for _, replica := range replicas {
			if replica.Byzantine {
				continue
			}
			for _, target := range replicas {
				if target.ID == replica.ID {
					continue
				}
				messages = append(messages, framework.Message{
					ID:       fmt.Sprintf("prepare-%s-%s-%d", replica.ID, target.ID, sequence),
					Label:    fmt.Sprintf("Prepare(%d)", sequence),
					Kind:     "vote",
					Status:   phase,
					SourceID: replica.ID,
					TargetID: target.ID,
				})
			}
		}
	case "Commit":
		for _, replica := range replicas {
			if replica.Byzantine {
				continue
			}
			for _, target := range replicas {
				if target.ID == replica.ID {
					continue
				}
				messages = append(messages, framework.Message{
					ID:       fmt.Sprintf("commit-%s-%s-%d", replica.ID, target.ID, sequence),
					Label:    fmt.Sprintf("Commit(%d)", sequence),
					Kind:     "vote",
					Status:   phase,
					SourceID: replica.ID,
					TargetID: target.ID,
				})
			}
		}
	case "Checkpoint":
		for _, replica := range replicas {
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("checkpoint-%s-%d", replica.ID, sequence),
				Label:    fmt.Sprintf("Checkpoint(%d)", sequence),
				Kind:     "vote",
				Status:   phase,
				SourceID: replica.ID,
			})
		}
	case "View Change":
		for _, replica := range replicas {
			if replica.IsPrimary {
				continue
			}
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("view-change-%s-%d", replica.ID, view+1),
				Label:    fmt.Sprintf("ViewChange(%d)", view+1),
				Kind:     "vote",
				Status:   phase,
				SourceID: replica.ID,
				TargetID: primary.ID,
			})
		}
	}
	return messages
}

// collectVotes 统计当前阶段的有效票数，拜占庭节点不会贡献法定票。
func collectVotes(replicas []replicaState, sequence int, commit bool) int {
	votes := 0
	for index := range replicas {
		if replicas[index].Byzantine || replicas[index].Partitioned {
			replicas[index].CurrentMessages = []string{"faulty"}
			continue
		}
		votes++
		if commit {
			replicas[index].CommittedSeqs = appendUniqueInt(replicas[index].CommittedSeqs, sequence)
			replicas[index].CurrentMessages = []string{"commit"}
		} else {
			replicas[index].PreparedSeqs = appendUniqueInt(replicas[index].PreparedSeqs, sequence)
			replicas[index].CurrentMessages = []string{"prepare"}
		}
	}
	return votes
}

// countVotes 只统计票数，不修改副本内部状态，供渲染指标读取。
func countVotes(replicas []replicaState, sequence int, commit bool) int {
	votes := 0
	for _, replica := range replicas {
		if replica.Byzantine {
			continue
		}
		if commit && containsInt(replica.CommittedSeqs, sequence) {
			votes++
		}
		if !commit && containsInt(replica.PreparedSeqs, sequence) {
			votes++
		}
	}
	return votes
}

// rotatePrimary 根据视图编号轮转主节点。
func rotatePrimary(replicas []replicaState, view int) {
	target := view % len(replicas)
	for index := range replicas {
		replicas[index].IsPrimary = index == target
		replicas[index].ViewNumber = view
	}
}

// currentPrimary 返回当前主节点；若状态异常则回退到第一个副本。
func currentPrimary(replicas []replicaState) replicaState {
	for _, replica := range replicas {
		if replica.IsPrimary {
			return replica
		}
	}
	return replicas[0]
}

// stableCheckpoint 计算当前稳定检查点，并同步更新水位线。
func stableCheckpoint(replicas []replicaState) int {
	stable := 0
	for index := range replicas {
		if len(replicas[index].CommittedSeqs) == 0 {
			continue
		}
		sort.Ints(replicas[index].CommittedSeqs)
		last := replicas[index].CommittedSeqs[len(replicas[index].CommittedSeqs)-1]
		if last%2 == 0 && last > stable {
			stable = last
		}
		replicas[index].LastCheckpoint = stable
		replicas[index].LowWatermark = stable
		replicas[index].HighWatermark = stable + 50
	}
	return stable
}

// incrementTimeouts 增加所有存活副本的超时计数。
func incrementTimeouts(replicas []replicaState) {
	for index := range replicas {
		if replicas[index].Partitioned || replicas[index].Byzantine {
			replicas[index].TimeoutCounter++
		}
	}
}

// resetTimeouts 在完成视图切换后清空超时计数。
func resetTimeouts(replicas []replicaState) {
	for index := range replicas {
		replicas[index].TimeoutCounter = 0
	}
}

// markCheckpoint 显式记录稳定检查点。
func markCheckpoint(replicas []replicaState, sequence int) {
	for index := range replicas {
		replicas[index].LastCheckpoint = sequence
		replicas[index].LowWatermark = sequence
		replicas[index].HighWatermark = sequence + 50
	}
}

// totalTimeouts 返回当前累计超时次数。
func totalTimeouts(replicas []replicaState) int {
	total := 0
	for _, replica := range replicas {
		total += replica.TimeoutCounter
	}
	return total
}

// voteTone 根据票数是否达到法定阈值选择指标色调。
func voteTone(votes int) string {
	if votes >= quorumSize {
		return "success"
	}
	return "warning"
}

// stageIndex 将阶段名称映射为前端时间线索引。
func stageIndex(phase string) int {
	switch phase {
	case "Pre-prepare":
		return 0
	case "Prepare":
		return 1
	case "Commit":
		return 2
	case "Checkpoint":
		return 3
	case "View Change":
		return 4
	default:
		return 0
	}
}

// appendUniqueInt 在序号集合中追加新值，并保证不重复。
func appendUniqueInt(values []int, target int) []int {
	if containsInt(values, target) {
		return values
	}
	return append(values, target)
}

// containsInt 判断目标序号是否已存在。
func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// toIntSlice 将通用 JSON 列表恢复为整型切片。
func toIntSlice(value any) []int {
	raw, ok := value.([]any)
	if !ok {
		return []int{}
	}
	result := make([]int, 0, len(raw))
	for _, item := range raw {
		result = append(result, int(framework.NumberValue(item, 0)))
	}
	return result
}

// applySharedConsensusState 将联动组共享状态映射回 PBFT 共识模型。
// 该逻辑用于让拜占庭攻击场景和网络扰动场景对当前共识过程产生自然影响。
func applySharedConsensusState(replicas []replicaState, sharedState map[string]any, view int, requestID string) (int, string, bool) {
	if len(sharedState) == 0 {
		return view, requestID, false
	}
	partitioned := false
	if sharedView := int(framework.NumberValue(sharedState["view"], float64(view))); sharedView > view {
		view = sharedView
	}
	if nodes, ok := sharedState["nodes"].(map[string]any); ok {
		byzantineID := framework.NormalizeDashedID("replica", framework.StringValue(nodes["byzantine"], ""), "")
		if byzantineID != "" {
			for index := range replicas {
				if replicas[index].ID == byzantineID {
					replicas[index].Byzantine = true
					replicas[index].Status = "byzantine"
				}
			}
		}
		if isolated := framework.ToStringSlice(nodes["isolated"]); len(isolated) > 0 {
			partitioned = true
			for index := range replicas {
				replicas[index].Partitioned = containsString(isolated, replicas[index].ID)
				if replicas[index].Partitioned {
					replicas[index].CurrentMessages = []string{"partitioned"}
				}
			}
		}
	}
	if messages, ok := sharedState["messages"].([]any); ok && len(messages) > 0 {
		requestID = fmt.Sprintf("linked-%d", len(messages))
	}
	if !partitioned {
		for index := range replicas {
			replicas[index].Partitioned = false
		}
	}
	return view, requestID, partitioned
}

// byzantineReplicaID 返回当前被标记为拜占庭的副本标识。
func byzantineReplicaID(replicas []replicaState) string {
	for _, replica := range replicas {
		if replica.Byzantine {
			return replica.ID
		}
	}
	return ""
}

// isolatedReplicaIDs 返回当前被网络隔离的副本列表。
func isolatedReplicaIDs(replicas []replicaState) []string {
	result := make([]string, 0)
	for _, replica := range replicas {
		if replica.Partitioned {
			result = append(result, replica.ID)
		}
	}
	return result
}

// containsString 判断字符串列表中是否存在目标值。
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
