package raftelection

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// majorityVotes 是 5 节点 Raft 集群选主所需的多数票。
	majorityVotes = 3
)

// DefaultState 构造 Raft 领导选举场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "raft-election",
		Title:        "Raft 领导选举",
		Phase:        "Follower 超时",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   20,
		Stages:       []string{"Follower 超时", "Candidate 拉票", "Leader 当选", "日志复制"},
		Nodes: []framework.Node{
			{ID: "node-a", Label: "Node-A", Status: "normal", Role: "follower", X: 120, Y: 200},
			{ID: "node-b", Label: "Node-B", Status: "normal", Role: "follower", X: 260, Y: 120},
			{ID: "node-c", Label: "Node-C", Status: "normal", Role: "follower", X: 420, Y: 120},
			{ID: "node-d", Label: "Node-D", Status: "normal", Role: "follower", X: 560, Y: 200},
			{ID: "node-e", Label: "Node-E", Status: "normal", Role: "follower", X: 340, Y: 320},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 Raft 集群任期、日志索引和节点角色。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	nodes := defaultNodes()
	return rebuildState(state, nodes, 1, "node-a", "Follower 超时")
}

// Step 推进超时、拉票、当选和日志复制流程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	nodes := decodeNodes(state)
	term := int(framework.NumberValue(state.Data["term"], 1))
	leaderID := framework.StringValue(state.Data["leader_id"], "node-a")
	term, leaderID = applySharedRaftState(nodes, input.SharedState, term, leaderID)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "Follower 超时"))
	events := make([]framework.TimelineEvent, 0, 1)

	switch phase {
	case "Follower 超时":
		term++
		for index := range nodes {
			nodes[index].Role = "follower"
			nodes[index].VotedFor = ""
			if nodes[index].ID == leaderID {
				nodes[index].Role = "candidate"
				nodes[index].VotedFor = nodes[index].ID
			}
		}
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "选举超时", fmt.Sprintf("%s 在任期 %d 转为 Candidate。", leaderID, term), "warning"))
	case "Candidate 拉票":
		for index := range nodes {
			if !nodes[index].Failed {
				nodes[index].VotedFor = leaderID
			}
		}
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "请求投票", fmt.Sprintf("%s 收集到 %d 张有效票。", leaderID, countVotes(nodes, leaderID)), "info"))
	case "Leader 当选":
		for index := range nodes {
			if nodes[index].ID == leaderID {
				nodes[index].Role = "leader"
			} else if !nodes[index].Failed {
				nodes[index].Role = "follower"
			}
		}
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "Leader 当选", fmt.Sprintf("%s 获得多数票并成为 Leader。", leaderID), "success"))
	case "日志复制":
		for index := range nodes {
			if !nodes[index].Failed {
				nodes[index].LogIndex++
				nodes[index].CommitIndex = nodes[index].LogIndex
			}
		}
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "复制日志", "Leader 发送 AppendEntries，Follower 追平日志索引。", "success"))
	}

	if err := rebuildState(state, nodes, term, leaderID, phase); err != nil {
		return framework.StepOutput{}, err
	}
	return framework.StepOutput{
		Events: events,
		SharedDiff: map[string]any{
			"nodes": raftSharedNodes(nodes),
			"terms": map[string]any{"current_term": term, "leader": leaderID, "leader_failed": false},
			"logs":  raftSharedLogs(nodes),
		},
	}, nil
}

// HandleAction 处理 Leader 故障，触发新一轮选举。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	nodes := decodeNodes(state)
	leaderID := framework.NormalizeDashedID("node", framework.StringValue(input.Params["resource_id"], framework.StringValue(state.Data["leader_id"], "node-a")), "node-a")
	for index := range nodes {
		if nodes[index].ID == leaderID {
			nodes[index].Failed = true
			nodes[index].Role = "failed"
		}
	}
	nextLeader := firstAlive(nodes)
	term := int(framework.NumberValue(state.Data["term"], 1))
	if err := rebuildState(state, nodes, term, nextLeader, "Follower 超时"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "Leader 故障", fmt.Sprintf("%s 已故障，下一候选人为 %s。", leaderID, nextLeader), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes": raftSharedNodes(nodes),
			"terms": map[string]any{"current_term": term, "leader": nextLeader, "leader_failed": true},
			"logs":  raftSharedLogs(nodes),
		},
	}, nil
}

// BuildRenderState 输出 Raft 任期、投票和日志复制状态。
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

// SyncSharedState 在任期、日志或分区联动更新后重建 Raft 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	nodes := decodeNodes(state)
	term := int(framework.NumberValue(state.Data["term"], 1))
	leaderID := framework.StringValue(state.Data["leader_id"], "node-a")
	phase := framework.StringValue(state.Data["phase_name"], state.Phase)
	term, leaderID = applySharedRaftState(nodes, sharedState, term, leaderID)
	return rebuildState(state, nodes, term, leaderID, phase)
}

// raftNode 保存 Raft 节点角色、投票和日志复制进度。
type raftNode struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Role            string `json:"role"`
	VotedFor        string `json:"voted_for"`
	LogIndex        int    `json:"log_index"`
	CommitIndex     int    `json:"commit_index"`
	TermStartIndex  int    `json:"term_start_index"`
	ElectionTimeout int    `json:"election_timeout"`
	Failed          bool   `json:"failed"`
}

// defaultNodes 创建默认 5 节点 Raft 集群。
func defaultNodes() []raftNode {
	labels := []string{"Node-A", "Node-B", "Node-C", "Node-D", "Node-E"}
	nodes := make([]raftNode, 0, len(labels))
	for index, label := range labels {
		nodes = append(nodes, raftNode{
			ID:          fmt.Sprintf("node-%c", 'a'+rune(index)),
			Label:       label,
			Role:        "follower",
			LogIndex:    1,
			CommitIndex: 1,
		})
	}
	return nodes
}

// rebuildState 将 Raft 内部状态转为可渲染节点、消息和指标。
func rebuildState(state *framework.SceneState, nodes []raftNode, term int, leaderID string, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes = make([]framework.Node, 0, len(nodes))
	for index, node := range nodes {
		status := "normal"
		switch {
		case node.Failed:
			status = "failed"
		case node.Role == "leader":
			status = "success"
		case node.Role == "candidate":
			status = "active"
		}
		state.Nodes = append(state.Nodes, framework.Node{
			ID:     node.ID,
			Label:  node.Label,
			Status: status,
			Role:   node.Role,
			X:      130 + float64(index%3)*190,
			Y:      150 + float64(index/3)*170,
			Load:   float64(node.CommitIndex * 20),
			Attributes: map[string]any{
				"term":             term,
				"voted_for":        node.VotedFor,
				"log_index":        node.LogIndex,
				"commit_index":     node.CommitIndex,
				"term_start_index": node.TermStartIndex,
				"election_timeout": node.ElectionTimeout,
				"failed":           node.Failed,
			},
		})
	}
	state.Messages = buildMessages(nodes, leaderID, phase, term)
	state.Metrics = []framework.Metric{
		{Key: "term", Label: "当前任期", Value: fmt.Sprintf("%d", term), Tone: "info"},
		{Key: "leader", Label: "Leader", Value: leaderLabel(nodes, leaderID), Tone: "success"},
		{Key: "votes", Label: "投票数", Value: fmt.Sprintf("%d / %d", countVotes(nodes, leaderID), majorityVotes), Tone: voteTone(countVotes(nodes, leaderID))},
		{Key: "timeout", Label: "最小超时", Value: fmt.Sprintf("%d", minTimeout(nodes)), Tone: "warning"},
		{Key: "commit", Label: "最高提交索引", Value: fmt.Sprintf("%d", maxCommit(nodes)), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "任期", Value: fmt.Sprintf("%d", term)},
		{Label: "多数票", Value: fmt.Sprintf("%d", majorityVotes)},
		{Label: "任期起始索引", Value: fmt.Sprintf("%d", termStart(nodes))},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"term":       term,
		"leader_id":  leaderID,
		"raft_nodes": nodes,
		"votes":      countVotes(nodes, leaderID),
	}
	state.Extra = map[string]any{
		"description": "该场景实现 Raft 超时、候选拉票、Leader 当选和 AppendEntries 日志复制。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeNodes 从通用 JSON 状态恢复 Raft 节点集合。
func decodeNodes(state *framework.SceneState) []raftNode {
	raw, ok := state.Data["raft_nodes"].([]any)
	if !ok {
		if typed, ok := state.Data["raft_nodes"].([]raftNode); ok {
			return typed
		}
		return defaultNodes()
	}
	nodes := make([]raftNode, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		nodes = append(nodes, raftNode{
			ID:          framework.StringValue(entry["id"], ""),
			Label:       framework.StringValue(entry["label"], ""),
			Role:        framework.StringValue(entry["role"], "follower"),
			VotedFor:    framework.StringValue(entry["voted_for"], ""),
			LogIndex:    int(framework.NumberValue(entry["log_index"], 1)),
			CommitIndex: int(framework.NumberValue(entry["commit_index"], 1)),
			Failed:      framework.BoolValue(entry["failed"], false),
		})
	}
	if len(nodes) == 0 {
		return defaultNodes()
	}
	return nodes
}

// buildMessages 根据阶段生成 RequestVote、Heartbeat 或 AppendEntries 消息。
func buildMessages(nodes []raftNode, leaderID string, phase string, term int) []framework.Message {
	messages := make([]framework.Message, 0)
	for _, node := range nodes {
		if node.ID == leaderID || node.Failed {
			continue
		}
		label := fmt.Sprintf("RequestVote(term=%d)", term)
		source := leaderID
		target := node.ID
		if phase == "日志复制" {
			label = "AppendEntries"
		}
		if phase == "Candidate 拉票" {
			source = leaderID
		}
		messages = append(messages, framework.Message{
			ID:       fmt.Sprintf("%s-%s-%s", phase, source, target),
			Label:    label,
			Kind:     "vote",
			Status:   phase,
			SourceID: source,
			TargetID: target,
		})
	}
	return messages
}

// nextPhase 返回 Raft 过程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "Follower 超时":
		return "Candidate 拉票"
	case "Candidate 拉票":
		return "Leader 当选"
	case "Leader 当选":
		return "日志复制"
	default:
		return "Follower 超时"
	}
}

// phaseIndex 将阶段名映射到时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "Follower 超时":
		return 0
	case "Candidate 拉票":
		return 1
	case "Leader 当选":
		return 2
	case "日志复制":
		return 3
	default:
		return 0
	}
}

// countVotes 统计投给目标候选人的有效票。
func countVotes(nodes []raftNode, candidateID string) int {
	votes := 0
	for _, node := range nodes {
		if !node.Failed && node.VotedFor == candidateID {
			votes++
		}
	}
	return votes
}

// firstAlive 返回第一个未故障节点作为下一候选人。
func firstAlive(nodes []raftNode) string {
	for _, node := range nodes {
		if !node.Failed {
			return node.ID
		}
	}
	return "node-a"
}

// leaderLabel 返回 Leader 的展示名称。
func leaderLabel(nodes []raftNode, leaderID string) string {
	for _, node := range nodes {
		if node.ID == leaderID {
			return node.Label
		}
	}
	return leaderID
}

// maxCommit 返回集群中最高提交日志索引。
func maxCommit(nodes []raftNode) int {
	maxValue := 0
	for _, node := range nodes {
		if node.CommitIndex > maxValue {
			maxValue = node.CommitIndex
		}
	}
	return maxValue
}

// minTimeout 返回当前存活节点中的最小选举超时。
func minTimeout(nodes []raftNode) int {
	minValue := 0
	for _, node := range nodes {
		if node.Failed {
			continue
		}
		if minValue == 0 || node.ElectionTimeout < minValue {
			minValue = node.ElectionTimeout
		}
	}
	return minValue
}

// termStart 返回当前任期的最小起始日志索引。
func termStart(nodes []raftNode) int {
	minValue := 0
	for _, node := range nodes {
		if node.Failed {
			continue
		}
		if minValue == 0 || node.TermStartIndex < minValue {
			minValue = node.TermStartIndex
		}
	}
	return minValue
}

// voteTone 根据投票是否达到多数派返回指标色调。
func voteTone(votes int) string {
	if votes >= majorityVotes {
		return "success"
	}
	return "warning"
}

// applySharedRaftState 将联动组中的任期、Leader 故障和网络分区状态映射回 Raft 集群。
func applySharedRaftState(nodes []raftNode, sharedState map[string]any, term int, leaderID string) (int, string) {
	if len(sharedState) == 0 {
		return term, leaderID
	}
	if terms, ok := sharedState["terms"].(map[string]any); ok {
		if sharedTerm := int(framework.NumberValue(terms["current_term"], float64(term))); sharedTerm > term {
			term = sharedTerm
		}
		if sharedLeader := framework.NormalizeDashedID("node", framework.StringValue(terms["leader"], leaderID), leaderID); sharedLeader != "" {
			leaderID = sharedLeader
		}
		if framework.BoolValue(terms["leader_failed"], false) {
			for index := range nodes {
				if nodes[index].ID == leaderID {
					nodes[index].Failed = true
					nodes[index].Role = "failed"
				}
			}
			leaderID = firstAlive(nodes)
		}
	}
	if logs, ok := sharedState["logs"].(map[string]any); ok {
		if replicaIndex, ok := logs["replica_index"].(map[string]any); ok {
			for index := range nodes {
				if value, exists := replicaIndex[nodes[index].ID]; exists {
					nodes[index].LogIndex = int(framework.NumberValue(value, float64(nodes[index].LogIndex)))
					nodes[index].CommitIndex = nodes[index].LogIndex
				}
			}
		}
	}
	return term, leaderID
}

// raftSharedNodes 输出 Raft 容错组统一使用的节点共享结构。
func raftSharedNodes(nodes []raftNode) map[string]any {
	result := make(map[string]any, len(nodes))
	for _, node := range nodes {
		result[node.ID] = map[string]any{
			"role":   node.Role,
			"failed": node.Failed,
		}
	}
	return result
}

// raftSharedLogs 输出 Raft 容错组统一使用的日志共享结构。
func raftSharedLogs(nodes []raftNode) map[string]any {
	replicaIndex := make(map[string]any, len(nodes))
	maxIndex := 0
	for _, node := range nodes {
		replicaIndex[node.ID] = node.LogIndex
		if node.CommitIndex > maxIndex {
			maxIndex = node.CommitIndex
		}
	}
	return map[string]any{
		"replica_index": replicaIndex,
		"commit_index":  maxIndex,
	}
}
