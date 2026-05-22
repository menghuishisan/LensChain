// 模块：sim-engine/scenarios/internal/consensus/raftelection
// 文件职责：CON-04 Raft 选举与日志复制场景的完整实现。
//
// SSOT 依据：06.md §4.2.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现 Raft 三角色状态机（Follower / Candidate / Leader），含：
//
//   · Term 单调递增，三角色规则严格按 Raft 论文（Diego Ongaro 2014）：
//     · Follower：election_timeout 到 → 转 Candidate；
//     · Candidate：自增 term，给自己投票，向所有节点 RequestVote；
//                   收到多数票 → Leader；遇更高 term 或同 term 更高 log → 退回 Follower；
//     · Leader：周期发 AppendEntries 心跳；接收客户端日志 → 复制 → 多数确认即 commit；
//   · RequestVote RPC：先比较 term，再比较 (lastLogTerm, lastLogIndex) 决定是否投票；
//   · AppendEntries RPC：term 检查 + 日志一致性匹配；
//   · 网络分区：把节点分到两个 partition，分区内消息可达，跨分区消息丢弃；
//     · 大多数派可选出新 leader，少数派 stuck；恢复后 term 较高的 leader 接管。
//
// 教学决策：
//   - ring_layout 把 N 个节点环形排列
//   - vote_matrix 展示投票（行=投票者，列=候选人）
//   - phase_progress 当前 election / heartbeat 状态
//   - partition_zone 网络分区遮罩
//   - 节点 status：normal=Follower / active=Candidate / 高亮=Leader / error=down

package raftelection

import (
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "raft-election"
	schemaVersion = "v1.0.0"
	algorithmType = "raft"

	defaultNodeCount  = 5
	maxNodeCount      = 9
	defaultTimeoutMin = 4 // election_timeout 随机区间下限（tick 数）
	defaultTimeoutMax = 8
	heartbeatInterval = 2

	linkGroupRaftFault = "raft-fault-group"
	linkOwnerSubtree   = "consensus.raft"
)

// =====================================================================
// 节点状态
// =====================================================================

type role string

const (
	roleFollower  role = "follower"
	roleCandidate role = "candidate"
	roleLeader    role = "leader"
	roleDown      role = "down"
)

type logEntry struct {
	Term    int
	Command string
}

type raftNode struct {
	ID             string
	Role           role
	CurrentTerm    int
	VotedFor       string // 当前 term 投给谁；"" 表示未投
	Log            []logEntry
	CommitIndex    int
	ElectionTimer  int // 当前剩余 tick；0 时触发 election
	HeartbeatTimer int // leader 用：剩余 tick 到下一次心跳
	VotesGotten    map[string]bool
	Partition      int // 0 = 默认分区；1/2 = 网络分区时的子分区
}

func (n *raftNode) lastLogIndex() int { return len(n.Log) - 1 }
func (n *raftNode) lastLogTerm() int {
	if len(n.Log) == 0 {
		return 0
	}
	return n.Log[len(n.Log)-1].Term
}

// =====================================================================
// 集群状态
// =====================================================================

type clusterState struct {
	Nodes       []raftNode
	Tick        int
	TimeoutMin  int
	TimeoutMax  int
	Partitioned bool
	VoteMatrix  map[string]map[string]bool // voter → candidate → true
	EventLog    []string                   // 最近 N 条事件
	LastError   string
}

func defaultCluster() clusterState {
	cs := clusterState{
		TimeoutMin: defaultTimeoutMin,
		TimeoutMax: defaultTimeoutMax,
		VoteMatrix: map[string]map[string]bool{},
	}
	for i := 0; i < defaultNodeCount; i++ {
		cs.Nodes = append(cs.Nodes, raftNode{
			ID:            fmt.Sprintf("n%d", i+1),
			Role:          roleFollower,
			CurrentTerm:   0,
			ElectionTimer: defaultTimeoutMin + (i*2)%(defaultTimeoutMax-defaultTimeoutMin+1),
			Partition:     0,
			VotesGotten:   map[string]bool{},
		})
	}
	return cs
}

// activeNodes 返回未 down 的节点数。
func (cs clusterState) activeNodes() int {
	n := 0
	for _, x := range cs.Nodes {
		if x.Role != roleDown {
			n++
		}
	}
	return n
}

// majority 多数派阈值（包括自己）。
func (cs clusterState) majority() int { return cs.activeNodes()/2 + 1 }

// findLeader 当前 leader（如果有）。
func (cs clusterState) findLeader() string {
	maxTerm := -1
	leader := ""
	for _, n := range cs.Nodes {
		if n.Role == roleLeader && n.CurrentTerm > maxTerm {
			leader = n.ID
			maxTerm = n.CurrentTerm
		}
	}
	return leader
}

// canCommunicate 两节点是否在同一分区（可通信）。
func (cs clusterState) canCommunicate(a, b raftNode) bool {
	if !cs.Partitioned {
		return true
	}
	return a.Partition == b.Partition
}

// pushEvent 追加一条人类可读事件（最近 16 条）。
func (cs *clusterState) pushEvent(evt string) {
	cs.EventLog = append(cs.EventLog, fmt.Sprintf("[t=%d] %s", cs.Tick, evt))
	if len(cs.EventLog) > 16 {
		cs.EventLog = cs.EventLog[len(cs.EventLog)-16:]
	}
}

// =====================================================================
// Raft 状态机推进
// =====================================================================

// stepTick 推进 1 tick：递减计时器，处理 election / heartbeat。
func (cs *clusterState) stepTick() {
	cs.Tick++
	for i := range cs.Nodes {
		n := &cs.Nodes[i]
		if n.Role == roleDown {
			continue
		}
		switch n.Role {
		case roleFollower, roleCandidate:
			n.ElectionTimer--
			if n.ElectionTimer <= 0 {
				cs.startElection(i)
			}
		case roleLeader:
			n.HeartbeatTimer--
			if n.HeartbeatTimer <= 0 {
				cs.sendHeartbeats(i)
				n.HeartbeatTimer = heartbeatInterval
			}
		}
	}
}

// resetElectionTimer 给定 i 节点重置选举计时器到 [TimeoutMin, TimeoutMax] 之间，
// 用 (tick + i) 派生确定性"伪随机"，便于教学复现。
func (cs *clusterState) resetElectionTimer(i int) {
	span := cs.TimeoutMax - cs.TimeoutMin + 1
	if span < 1 {
		span = 1
	}
	cs.Nodes[i].ElectionTimer = cs.TimeoutMin + ((cs.Tick + i*7) % span)
}

// startElection 节点 i 转为 Candidate，自增 term，给自己投票，向所有可达节点请求投票。
func (cs *clusterState) startElection(i int) {
	n := &cs.Nodes[i]
	n.Role = roleCandidate
	n.CurrentTerm++
	n.VotedFor = n.ID
	n.VotesGotten = map[string]bool{n.ID: true}
	cs.pushEvent(fmt.Sprintf("%s 发起选举 (term=%d)", n.ID, n.CurrentTerm))
	cs.recordVote(n.ID, n.ID)
	cs.resetElectionTimer(i)
	// 向所有可达 + 未 down 节点请求投票
	for j := range cs.Nodes {
		if j == i {
			continue
		}
		other := &cs.Nodes[j]
		if other.Role == roleDown || !cs.canCommunicate(*n, *other) {
			continue
		}
		// RequestVote(term, candidateId, lastLogIndex, lastLogTerm)
		cs.handleRequestVote(j, *n)
	}
	// 选举后立即统计是否当选
	if len(n.VotesGotten) >= cs.majority() {
		cs.becomeLeader(i)
	}
}

// handleRequestVote 接收方处理 RequestVote RPC（Raft 论文 §5.4.1）。
func (cs *clusterState) handleRequestVote(receiverIdx int, candidate raftNode) {
	r := &cs.Nodes[receiverIdx]
	// 1) term 较小直接拒绝
	if candidate.CurrentTerm < r.CurrentTerm {
		return
	}
	// 2) 收到更高 term 强制转 Follower 并清空 votedFor
	if candidate.CurrentTerm > r.CurrentTerm {
		r.CurrentTerm = candidate.CurrentTerm
		r.Role = roleFollower
		r.VotedFor = ""
	}
	// 3) 同 term：仅当 votedFor 为空 / 已投本人，且候选人 log 至少和自己一样新时投票
	logUpToDate := candidate.lastLogTerm() > r.lastLogTerm() ||
		(candidate.lastLogTerm() == r.lastLogTerm() && candidate.lastLogIndex() >= r.lastLogIndex())
	if (r.VotedFor == "" || r.VotedFor == candidate.ID) && logUpToDate {
		r.VotedFor = candidate.ID
		// 找 candidate 节点更新 VotesGotten
		for ci := range cs.Nodes {
			if cs.Nodes[ci].ID == candidate.ID {
				cs.Nodes[ci].VotesGotten[r.ID] = true
				cs.recordVote(r.ID, candidate.ID)
				cs.pushEvent(fmt.Sprintf("%s 投票给 %s (term=%d)", r.ID, candidate.ID, candidate.CurrentTerm))
				cs.resetElectionTimer(receiverIdx)
				break
			}
		}
	}
}

// recordVote 记录到 VoteMatrix 用于前端展示。
func (cs *clusterState) recordVote(voter, candidate string) {
	if cs.VoteMatrix == nil {
		cs.VoteMatrix = map[string]map[string]bool{}
	}
	if cs.VoteMatrix[voter] == nil {
		cs.VoteMatrix[voter] = map[string]bool{}
	}
	cs.VoteMatrix[voter][candidate] = true
}

// becomeLeader 节点 i 当选 Leader：发送初始心跳，重置心跳计时器。
func (cs *clusterState) becomeLeader(i int) {
	n := &cs.Nodes[i]
	n.Role = roleLeader
	n.HeartbeatTimer = heartbeatInterval
	cs.pushEvent(fmt.Sprintf("%s 当选 Leader (term=%d)", n.ID, n.CurrentTerm))
	cs.sendHeartbeats(i)
}

// sendHeartbeats Leader 向所有可达 follower 发送 AppendEntries（空 entries = heartbeat）。
func (cs *clusterState) sendHeartbeats(leaderIdx int) {
	leader := cs.Nodes[leaderIdx]
	for j := range cs.Nodes {
		if j == leaderIdx {
			continue
		}
		f := &cs.Nodes[j]
		if f.Role == roleDown || !cs.canCommunicate(leader, *f) {
			continue
		}
		// AppendEntries(term, leaderId)
		if leader.CurrentTerm < f.CurrentTerm {
			// 我们的 term 落后，不能心跳；leader 应退位
			cs.Nodes[leaderIdx].Role = roleFollower
			cs.Nodes[leaderIdx].VotedFor = ""
			cs.resetElectionTimer(leaderIdx)
			cs.pushEvent(fmt.Sprintf("%s 收到更高 term → 退回 Follower", leader.ID))
			return
		}
		if leader.CurrentTerm > f.CurrentTerm {
			f.CurrentTerm = leader.CurrentTerm
			f.VotedFor = ""
		}
		f.Role = roleFollower
		cs.resetElectionTimer(j)
		// 简化：直接同步 leader.Log[..f.Log 之后] 到 follower（不实现完整 prevLogIndex 一致性回退）
		if len(f.Log) < len(leader.Log) {
			f.Log = append(f.Log, leader.Log[len(f.Log):]...)
		}
		// 推进 commit_index 到多数派已复制位置
		// （简化：只要 Leader 有，follower 就立刻同步 log，commit 由后面的 majority 检查决定）
	}
	// Leader 计算自己的 commit_index：最高的 N，使得多数派 followers 的 log 长度 ≥ N
	cs.advanceCommitIndex(leaderIdx)
}

// advanceCommitIndex 计算大多数派复制的最高位置。
func (cs *clusterState) advanceCommitIndex(leaderIdx int) {
	leader := &cs.Nodes[leaderIdx]
	maj := cs.majority()
	// 对 leader log 的每个位置，统计复制数量
	for n := len(leader.Log); n > leader.CommitIndex+1; n-- {
		count := 0
		for _, x := range cs.Nodes {
			if x.Role == roleDown {
				continue
			}
			if !cs.canCommunicate(*leader, x) {
				continue
			}
			if len(x.Log) >= n && (n == 0 || x.Log[n-1].Term == leader.Log[n-1].Term) {
				count++
			}
		}
		if count >= maj {
			leader.CommitIndex = n - 1
			cs.pushEvent(fmt.Sprintf("%s 提交 log 到 #%d", leader.ID, leader.CommitIndex))
			break
		}
	}
}

// proposeLog Leader 接收客户端命令，追加到本地 log。
func (cs *clusterState) proposeLog(cmd string) error {
	leaderID := cs.findLeader()
	if leaderID == "" {
		return errors.New("当前无 Leader")
	}
	for i := range cs.Nodes {
		if cs.Nodes[i].ID == leaderID {
			cs.Nodes[i].Log = append(cs.Nodes[i].Log, logEntry{Term: cs.Nodes[i].CurrentTerm, Command: cmd})
			cs.pushEvent(fmt.Sprintf("%s (Leader) 接收命令: %s", leaderID, cmd))
			cs.sendHeartbeats(i)
			return nil
		}
	}
	return errors.New("Leader 节点丢失")
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) clusterState {
	if s == nil || s.Data == nil {
		return defaultCluster()
	}
	d := s.Data
	cs := clusterState{
		Tick:        fw.MapInt(d, "tick", 0),
		TimeoutMin:  fw.MapInt(d, "timeout_min", defaultTimeoutMin),
		TimeoutMax:  fw.MapInt(d, "timeout_max", defaultTimeoutMax),
		Partitioned: fw.MapBool(d, "partitioned", false),
		LastError:   fw.MapStr(d, "last_error", ""),
		VoteMatrix:  map[string]map[string]bool{},
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				n := raftNode{
					ID:             fw.MapStr(nm, "id", ""),
					Role:           role(fw.MapStr(nm, "role", "follower")),
					CurrentTerm:    fw.MapInt(nm, "term", 0),
					VotedFor:       fw.MapStr(nm, "voted_for", ""),
					CommitIndex:    fw.MapInt(nm, "commit_index", -1),
					ElectionTimer:  fw.MapInt(nm, "election_timer", defaultTimeoutMin),
					HeartbeatTimer: fw.MapInt(nm, "heartbeat_timer", 0),
					Partition:      fw.MapInt(nm, "partition", 0),
					VotesGotten:    map[string]bool{},
				}
				if logAny, ok := nm["log"].([]any); ok {
					for _, eAny := range logAny {
						if em, ok := eAny.(map[string]any); ok {
							n.Log = append(n.Log, logEntry{
								Term:    fw.MapInt(em, "term", 0),
								Command: fw.MapStr(em, "cmd", ""),
							})
						}
					}
				}
				if vAny, ok := nm["votes_gotten"].([]any); ok {
					for _, v := range vAny {
						if s, ok := v.(string); ok {
							n.VotesGotten[s] = true
						}
					}
				}
				cs.Nodes = append(cs.Nodes, n)
			}
		}
	}
	if len(cs.Nodes) == 0 {
		cs.Nodes = defaultCluster().Nodes
	}
	if vmAny, ok := d["vote_matrix"].(map[string]any); ok {
		for k, v := range vmAny {
			cs.VoteMatrix[k] = map[string]bool{}
			if listAny, ok := v.([]any); ok {
				for _, c := range listAny {
					if s, ok := c.(string); ok {
						cs.VoteMatrix[k][s] = true
					}
				}
			}
		}
	}
	if logAny, ok := d["event_log"].([]any); ok {
		for _, e := range logAny {
			if s, ok := e.(string); ok {
				cs.EventLog = append(cs.EventLog, s)
			}
		}
	}
	return cs
}

func saveState(s *fw.SceneState, cs clusterState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = cs.Tick
	s.Data["timeout_min"] = cs.TimeoutMin
	s.Data["timeout_max"] = cs.TimeoutMax
	s.Data["partitioned"] = cs.Partitioned
	s.Data["last_error"] = cs.LastError
	nodesAny := make([]any, len(cs.Nodes))
	for i, n := range cs.Nodes {
		logAny := make([]any, len(n.Log))
		for j, e := range n.Log {
			logAny[j] = map[string]any{"term": e.Term, "cmd": e.Command}
		}
		votes := make([]any, 0, len(n.VotesGotten))
		for k := range n.VotesGotten {
			votes = append(votes, k)
		}
		nodesAny[i] = map[string]any{
			"id":              n.ID,
			"role":            string(n.Role),
			"term":            n.CurrentTerm,
			"voted_for":       n.VotedFor,
			"commit_index":    n.CommitIndex,
			"election_timer":  n.ElectionTimer,
			"heartbeat_timer": n.HeartbeatTimer,
			"partition":       n.Partition,
			"log":             logAny,
			"votes_gotten":    votes,
		}
	}
	s.Data["nodes"] = nodesAny
	vmAny := map[string]any{}
	for voter, cs := range cs.VoteMatrix {
		list := []any{}
		for c := range cs {
			list = append(list, c)
		}
		vmAny[voter] = list
	}
	s.Data["vote_matrix"] = vmAny
	logAny := make([]any, len(cs.EventLog))
	for i, e := range cs.EventLog {
		logAny[i] = e
	}
	s.Data["event_log"] = logAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "Raft 选举与日志复制",
		Description:         "演示 Raft 三角色状态机：Follower → Candidate → Leader；选举超时 / 心跳 / 日志复制 / 网络分区",
		Category:            fw.CategoryConsensus,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupRaftFault},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"consensus.raft.term",
			"consensus.raft.leader",
			"consensus.raft.commit_index",
			"consensus.raft.partitioned",
		},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultStateFw,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultStateFw() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "follower",
		Data:      map[string]any{},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_cluster", Label: "设置集群规模",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "node_count", Type: fw.FieldNumber, Label: "节点数（奇数最佳）", Required: true, Default: defaultNodeCount, Min: 3, Max: maxNodeCount, Step: 2},
					{Name: "timeout_min", Type: fw.FieldNumber, Label: "election_timeout 下限 tick", Required: true, Default: defaultTimeoutMin, Min: 2, Max: 20, Step: 1},
					{Name: "timeout_max", Type: fw.FieldNumber, Label: "election_timeout 上限 tick", Required: true, Default: defaultTimeoutMax, Min: 3, Max: 30, Step: 1},
				},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Description: "递减所有 timer，触发 election / heartbeat",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"consensus.raft.term", "consensus.raft.leader"},
				LinkOwnerFields:   []string{"consensus.raft.term", "consensus.raft.leader"},
			},
			{
				ActionCode: "step_n_ticks", Label: "推进 N tick",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 10, Min: 1, Max: 100, Step: 1},
				},
				WritesOwnedFields: []string{"consensus.raft.term"},
				LinkOwnerFields:   []string{"consensus.raft.term"},
			},
			{
				ActionCode: "force_election", Label: "强制启动选举",
				Description:   "把指定节点立即转为 Candidate",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n1"},
				},
			},
			{
				ActionCode: "propose_log", Label: "提交日志",
				Description:   "客户端向 Leader 提交一条命令",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "command", Type: fw.FieldString, Label: "命令", Required: true, Default: "set x=1"},
				},
				WritesOwnedFields: []string{"consensus.raft.commit_index"},
				LinkOwnerFields:   []string{"consensus.raft.commit_index"},
			},
			{
				ActionCode: "partition_network", Label: "网络分区",
				Description: "把节点拆成两个分区（前一半 / 后一半）",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.InterveneAttack,
				WritesOwnedFields: []string{"consensus.raft.partitioned"},
				LinkOwnerFields:   []string{"consensus.raft.partitioned"},
			},
			{
				ActionCode: "recover_partition", Label: "恢复分区",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode: "crash_node", Label: "节点崩溃",
				Description:   "把指定节点置为 down",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n1"},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_inject_fault",
				Label:         "教师注入故障",
				Description:   "仅教师可用，注入故障用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneFault,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "故障描述", Required: false, Default: "教师注入故障"},
				},
			},
			fw.BroadcastHintAction(),
		},
	}
}

// =====================================================================
// 钩子
// =====================================================================

func initScene(state *fw.SceneState, in fw.InitInput) (fw.RenderEnvelope, error) {
	state.Seed = in.Seed
	cs := loadState(state)
	saveState(state, cs)
	state.Phase = "follower"
	env := buildEnvelope(cs, "init", "Raft 初始化（5 节点全 Follower）", true)
	publishOwnerSubtree(&env, cs)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	cs := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(cs, "tick", "", false)
	return fw.StepOutput{Render: env}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	cs := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "set_cluster":
		cnt := fw.MapInt(in.Params, "node_count", defaultNodeCount)
		if cnt < 3 {
			cnt = 3
		}
		if cnt > maxNodeCount {
			cnt = maxNodeCount
		}
		cs = clusterState{
			TimeoutMin: fw.MapInt(in.Params, "timeout_min", defaultTimeoutMin),
			TimeoutMax: fw.MapInt(in.Params, "timeout_max", defaultTimeoutMax),
			VoteMatrix: map[string]map[string]bool{},
		}
		for i := 0; i < cnt; i++ {
			cs.Nodes = append(cs.Nodes, raftNode{
				ID:            fmt.Sprintf("n%d", i+1),
				Role:          roleFollower,
				ElectionTimer: cs.TimeoutMin + (i*2)%(cs.TimeoutMax-cs.TimeoutMin+1),
				CommitIndex:   -1,
				VotesGotten:   map[string]bool{},
			})
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "set_cluster", fmt.Sprintf("已重置为 %d 节点集群", cnt), true)
		return out, nil

	case "step_tick":
		cs.stepTick()
		saveState(state, cs)
		summary := fmt.Sprintf("tick=%d  leader=%s", cs.Tick, displayLeader(cs))
		out.Render = buildEnvelope(cs, "step_tick", summary, false)
		appendStepTickMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 10)
		for i := 0; i < n; i++ {
			cs.stepTick()
		}
		saveState(state, cs)
		summary := fmt.Sprintf("推进 %d tick → leader=%s", n, displayLeader(cs))
		out.Render = buildEnvelope(cs, "step_n_ticks", summary, false)
		appendStepTickMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "force_election":
		nid := fw.MapStr(in.Params, "node_id", "n1")
		idx := -1
		for i, n := range cs.Nodes {
			if n.ID == nid {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到节点: " + nid}, nil
		}
		if cs.Nodes[idx].Role == roleDown {
			return fw.ActionOutput{Success: false, ErrorMessage: nid + " 已 down"}, nil
		}
		cs.startElection(idx)
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "force_election", fmt.Sprintf("%s 强制发起选举 (term=%d)", nid, cs.Nodes[idx].CurrentTerm), false)
		appendElectionMicroSteps(&out.Render, nid)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "propose_log":
		cmd := fw.MapStr(in.Params, "command", "set x=1")
		if err := cs.proposeLog(cmd); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "propose_log", "Leader 接收命令并复制到 followers", false)
		appendProposeMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "partition_network":
		cs.Partitioned = true
		half := len(cs.Nodes) / 2
		for i := range cs.Nodes {
			if i < half {
				cs.Nodes[i].Partition = 1
			} else {
				cs.Nodes[i].Partition = 2
			}
		}
		cs.pushEvent("⚠ 网络分区：前半 / 后半 各成一区")
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "partition_network", "网络分区生效", false)
		appendPartitionMicroSteps(&out.Render, true)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "recover_partition":
		cs.Partitioned = false
		for i := range cs.Nodes {
			cs.Nodes[i].Partition = 0
		}
		cs.pushEvent("✓ 网络恢复")
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "recover_partition", "网络已恢复", false)
		appendPartitionMicroSteps(&out.Render, false)
		return out, nil

	case "crash_node":
		nid := fw.MapStr(in.Params, "node_id", "n1")
		for i := range cs.Nodes {
			if cs.Nodes[i].ID == nid {
				cs.Nodes[i].Role = roleDown
				cs.pushEvent(nid + " 节点崩溃")
				break
			}
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "crash_node", nid+" 已崩溃", false)
		return out, nil

	case "teacher_inject_fault":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师注入故障"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-fault-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		cs = defaultCluster()
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

func displayLeader(cs clusterState) string {
	id := cs.findLeader()
	if id == "" {
		return "(无)"
	}
	return id
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(cs clusterState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 40)

	// 1) 环形布局（按 cs.Nodes 顺序声明 ring 成员）
	nodeIDs := make([]string, len(cs.Nodes))
	for i, n := range cs.Nodes {
		nodeIDs[i] = "node-" + n.ID
	}
	prims = append(prims, fw.PrimRingLayout("node-ring", nodeIDs))

	// 2) 节点
	leaderID := cs.findLeader()
	for _, n := range cs.Nodes {
		status := "normal"
		role := "follower"
		switch n.Role {
		case roleCandidate:
			status = "active"
			role = "candidate"
		case roleLeader:
			status = "active"
			role = "leader"
		case roleDown:
			status = "error"
			role = "down"
		}
		label := fmt.Sprintf("%s\n%s\nterm=%d", n.ID, n.Role, n.CurrentTerm)
		if n.Role == roleLeader {
			label = fmt.Sprintf("👑 %s\nLEADER\nterm=%d", n.ID, n.CurrentTerm)
		}
		if cs.Partitioned {
			label += fmt.Sprintf("\n[P%d]", n.Partition)
		}
		prims = append(prims, fw.PrimNode("node-"+n.ID, label, status, role))
	}

	// 3) 节点之间的连接（任意 2 节点边，跨分区时 dashed）
	for i := 0; i < len(cs.Nodes); i++ {
		for j := i + 1; j < len(cs.Nodes); j++ {
			style := "solid"
			anim := ""
			if cs.Partitioned && cs.Nodes[i].Partition != cs.Nodes[j].Partition {
				style = "dashed"
			}
			if cs.Nodes[i].Role == roleLeader || cs.Nodes[j].Role == roleLeader {
				anim = "flow"
			}
			prims = append(prims, fw.PrimEdge(
				fmt.Sprintf("edge-%s-%s", cs.Nodes[i].ID, cs.Nodes[j].ID),
				"node-"+cs.Nodes[i].ID, "node-"+cs.Nodes[j].ID, style, anim))
		}
	}

	// 4) 投票矩阵
	N := len(cs.Nodes)
	cells := make([]map[string]any, 0, N*N)
	for i, voter := range cs.Nodes {
		for j, cand := range cs.Nodes {
			val := ""
			color := "muted"
			if cs.VoteMatrix[voter.ID][cand.ID] {
				val = "✓"
				color = "success"
			}
			cells = append(cells, map[string]any{
				"row": i, "col": j, "value": val, "color_role": color,
			})
		}
	}
	prims = append(prims, fw.PrimVoteMatrix("vote-matrix", N, N, cells))

	// 5) 当前 term / leader / 多数派阈值
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d\n当前 leader = %s\n活跃节点 = %d / %d\n多数派 ≥ %d\n网络分区 = %v",
			cs.Tick, displayLeader(cs), cs.activeNodes(), len(cs.Nodes), cs.majority(), cs.Partitioned),
		"text", nil, 6))

	// 6) 节点详情
	rows := []string{"id   role        term  voted  log  commit  timer"}
	for _, n := range cs.Nodes {
		rows = append(rows, fmt.Sprintf("%-4s %-10s  %-4d  %-5s  %-3d  %-6d  %d",
			n.ID, string(n.Role), n.CurrentTerm, dashOr(n.VotedFor), len(n.Log), n.CommitIndex, n.ElectionTimer))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-nodes", strings.Join(rows, "\n"), "text", nil, 14))

	// 7) 事件日志
	if len(cs.EventLog) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(cs.EventLog, "\n"), "text", nil, 16))
	}

	// 8) 公式（核心规则）
	prims = append(prims, fw.PrimMathFormula("formula-vote",
		`\text{投票条件}:\ T_c \ge T_r \land (\text{voted\_for} \in \{\bot, c\}) \land \text{log}_c \ge \text{log}_r`, false))

	// 9) 分区遮罩
	if cs.Partitioned {
		prims = append(prims, fw.PrimPartitionZone("partition-zone-1",
			[]map[string]float64{{"x": 0.0, "y": 0.0}, {"x": 0.5, "y": 0.0}, {"x": 0.5, "y": 1.0}, {"x": 0.0, "y": 1.0}},
			"分区 1"))
		prims = append(prims, fw.PrimPartitionZone("partition-zone-2",
			[]map[string]float64{{"x": 0.5, "y": 0.0}, {"x": 1.0, "y": 0.0}, {"x": 1.0, "y": 1.0}, {"x": 0.5, "y": 1.0}},
			"分区 2"))
	}

	// 10) 动效
	if leaderID != "" {
		prims = append(prims, fw.PrimGlow("glow-leader", "node-"+leaderID, "success", 0.9))
		prims = append(prims, fw.PrimPulse("pulse-leader", "node-"+leaderID, "success", 1500))
	}
	for _, n := range cs.Nodes {
		if n.Role == roleCandidate {
			prims = append(prims, fw.PrimGlow("glow-cand-"+n.ID, "node-"+n.ID, "warning", 0.7))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-raft", linkGroupRaftFault, "idle", ""))

	if cs.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Raft 错误", cs.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(cs, summary),
	}
}

func buildSidePanelData(cs clusterState, summary string) map[string]any {
	leaderID := cs.findLeader()
	maxTerm := 0
	commitIdx := -1
	for _, n := range cs.Nodes {
		if n.CurrentTerm > maxTerm {
			maxTerm = n.CurrentTerm
		}
		if n.Role == roleLeader && n.CommitIndex > commitIdx {
			commitIdx = n.CommitIndex
		}
	}
	d := map[string]any{
		"tick":         cs.Tick,
		"node_count":   len(cs.Nodes),
		"active_nodes": cs.activeNodes(),
		"majority":     cs.majority(),
		"current_term": maxTerm,
		"leader":       leaderID,
		"commit_index": commitIdx,
		"partitioned":  cs.Partitioned,
		"timeout_min":  cs.TimeoutMin,
		"timeout_max":  cs.TimeoutMax,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendStepTickMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tk-1", Label: "递减所有节点 election_timer", DurationMs: 300, HighlightIDs: []string{"cb-nodes"}, ParentPhase: "follower"},
		{ID: "tk-2", Label: "Leader 发心跳；Follower 收到则重置 timer", DurationMs: 400, HighlightIDs: []string{"cb-events"}, FirePrimitives: []string{"pulse-leader"}},
		{ID: "tk-3", Label: "timer 到 0 的节点转 Candidate", DurationMs: 400, HighlightIDs: []string{"vote-matrix", "cb-status"}, IsLinkTrigger: true},
	}
}

func appendElectionMicroSteps(env *fw.RenderEnvelope, candID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "el-1", Label: candID + " 转 Candidate，term++", DurationMs: 400, HighlightIDs: []string{"node-" + candID, "cb-status"}, FirePrimitives: []string{"glow-cand-" + candID}},
		{ID: "el-2", Label: "向所有节点发 RequestVote", DurationMs: 500, HighlightIDs: []string{"vote-matrix", "formula-vote"}},
		{ID: "el-3", Label: "收到多数票即当选 Leader", DurationMs: 500, HighlightIDs: []string{"cb-events"}, FirePrimitives: []string{"pulse-leader"}, IsLinkTrigger: true},
	}
}

func appendProposeMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "pr-1", Label: "Leader 追加 log", DurationMs: 400, HighlightIDs: []string{"cb-nodes"}},
		{ID: "pr-2", Label: "AppendEntries 复制到 followers", DurationMs: 500, HighlightIDs: []string{"cb-events"}, FirePrimitives: []string{"pulse-leader"}},
		{ID: "pr-3", Label: "多数派复制成功 → commit_index++", DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendPartitionMicroSteps(env *fw.RenderEnvelope, partition bool) {
	tail := "网络已恢复，term 较高的 leader 接管"
	if partition {
		tail = "少数派分区无法选出新 leader"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "pn-1", Label: "前半节点 → 分区 1，后半 → 分区 2", DurationMs: 500, HighlightIDs: []string{"node-ring"}},
		{ID: "pn-2", Label: "跨分区消息丢弃", DurationMs: 500, HighlightIDs: []string{"cb-events"}},
		{ID: "pn-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

// raftMaxTerm 取所有节点中最高的 CurrentTerm。
func raftMaxTerm(cs clusterState) int {
	maxT := 0
	for _, n := range cs.Nodes {
		if n.CurrentTerm > maxT {
			maxT = n.CurrentTerm
		}
	}
	return maxT
}

func publishOwnerSubtree(env *fw.RenderEnvelope, cs clusterState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "raft-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_consensus",
		LinkGroup:      linkGroupRaftFault,
		ChangedFields:  []string{"consensus.raft.term", "consensus.raft.leader"},
		Payload: map[string]any{
			"term":   raftMaxTerm(cs),
			"leader": cs.findLeader(),
		},
		SourceAnchorID: "raft-output-anchor",
		TargetAnchorID: "fault-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "consensus.raft.term", "consensus.raft.leader")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(cs clusterState) map[string]any {
	leaderID := cs.findLeader()
	maxTerm := 0
	commitIdx := -1
	for _, n := range cs.Nodes {
		if n.CurrentTerm > maxTerm {
			maxTerm = n.CurrentTerm
		}
		if n.Role == roleLeader && n.CommitIndex > commitIdx {
			commitIdx = n.CommitIndex
		}
	}
	return map[string]any{
		"consensus": map[string]any{
			"raft": map[string]any{
				"term":         maxTerm,
				"leader":       leaderID,
				"commit_index": commitIdx,
				"node_count":   len(cs.Nodes),
				"active_nodes": cs.activeNodes(),
				"partitioned":  cs.Partitioned,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func dashOr(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
