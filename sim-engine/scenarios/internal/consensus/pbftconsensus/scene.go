// 模块：sim-engine/scenarios/internal/consensus/pbftconsensus
// 文件职责：CON-03 实用拜占庭容错（PBFT）共识场景的完整实现。
//
// SSOT 依据：06.md §4.2.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现经典 Castro & Liskov PBFT（OSDI'99）；零第三方加密库；
// 复用 sha256hash.Sum256 计算 request digest：
//
//   · n 个副本（默认 n=4 → f=1，2f+1=3 quorum）；
//   · 5 阶段消息时序（Request → Pre-Prepare → Prepare → Commit → Reply）；
//   · primary = view mod n；
//   · 视图变更（View-Change）：副本怀疑 primary → 广播 ViewChange；
//     新 primary 收 2f+1 后广播 NewView 进入下一视图；
//   · 拜占庭注入：标记副本为 byzantine 后，它在 Prepare/Commit 阶段拒绝发出消息
//     （or 反向投票），用于演示安全性 / 活性边界；
//   · Quorum 校验：Prepare 收 2f+1 后 prepared；Commit 收 2f+1 后 committed-local；
//   · 仅 committed-local 的 request 被 reply 给客户端。
//
// 教学决策：
//   - ring_layout 排列 n 副本，primary 高亮
//   - vote_matrix 展示 prepare / commit 投票矩阵（行=副本, 列=phase）
//   - phase_progress 5 阶段
//   - 拜占庭副本 status=warning + role=byzantine

package pbftconsensus

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "pbft-consensus"
	schemaVersion = "v1.0.0"
	algorithmType = "pbft"

	defaultReplicaCount = 4 // f=1
	maxReplicaCount     = 7 // f=2

	linkGroupPbftAttack = "pbft-attack-group"
	linkOwnerSubtree    = "consensus.pbft"
)

// 5 阶段（与 phaseLabels / phaseIdx* 对齐）
const (
	phaseIdle       = 0
	phasePrePrepare = 1
	phasePrepare    = 2
	phaseCommit     = 3
	phaseReply      = 4
)

var phaseLabels = []string{"Request", "Pre-Prepare", "Prepare", "Commit", "Reply"}
var phaseRoles = []string{"idle", "pre-prepare", "prepare", "commit", "reply"}

// =====================================================================
// 副本 / 集群状态
// =====================================================================

// replica 单个 PBFT 副本的本地状态。
type replica struct {
	ID          string
	View        int
	IsByzantine bool
	IsDown      bool

	// 当前正在处理的 request（PrepareSet/CommitSet 的快照）
	CurrentDigest   string
	CurrentSequence int
	PrepareReceived map[string]bool // sender_id → true
	CommitReceived  map[string]bool
	Prepared        bool
	Committed       bool
	Replied         bool

	// 历史 committed log
	CommittedLog []committedEntry
}

type committedEntry struct {
	Sequence int
	Digest   string
	Request  string
	View     int
}

type clusterState struct {
	Replicas        []replica
	View            int    // 全局当前 view（最高的 valid view）
	Sequence        int    // 全局下一个 sequence 号
	CurrentRequest  string // 客户端请求内容（待处理）
	CurrentDigest   string // 当前 request 的 SHA-256 摘要 hex
	CurrentPhase    int    // 0..4
	EventLog        []string
	ViewChangeVotes map[int]map[string]bool // target_view → voter_id → true
	LastError       string
}

func defaultCluster(n int) clusterState {
	if n < 4 {
		n = 4
	}
	if n > maxReplicaCount {
		n = maxReplicaCount
	}
	cs := clusterState{
		Replicas:        make([]replica, 0, n),
		View:            0,
		Sequence:        0,
		ViewChangeVotes: map[int]map[string]bool{},
	}
	for i := 0; i < n; i++ {
		cs.Replicas = append(cs.Replicas, replica{
			ID:              fmt.Sprintf("r%d", i),
			View:            0,
			PrepareReceived: map[string]bool{},
			CommitReceived:  map[string]bool{},
		})
	}
	return cs
}

// faultThreshold f 计算（n = 3f + 1 → f = (n-1)/3）。
func (cs clusterState) faultThreshold() int { return (len(cs.Replicas) - 1) / 3 }

// quorum 2f + 1。
func (cs clusterState) quorum() int { return 2*cs.faultThreshold() + 1 }

// primaryID 当前 view 下的 primary ID（view mod n）。
func (cs clusterState) primaryID() string {
	if len(cs.Replicas) == 0 {
		return ""
	}
	idx := cs.View % len(cs.Replicas)
	return cs.Replicas[idx].ID
}

// activeReplicas 未 down 的副本数。
func (cs clusterState) activeReplicas() int {
	n := 0
	for _, r := range cs.Replicas {
		if !r.IsDown {
			n++
		}
	}
	return n
}

// pushEvent 追加一行人类可读事件。
func (cs *clusterState) pushEvent(evt string) {
	cs.EventLog = append(cs.EventLog, fmt.Sprintf("[v=%d, s=%d] %s", cs.View, cs.Sequence, evt))
	if len(cs.EventLog) > 24 {
		cs.EventLog = cs.EventLog[len(cs.EventLog)-24:]
	}
}

// =====================================================================
// 5 阶段推进（PBFT 状态机）
// =====================================================================

// proposeRequest 客户端发起请求，由 primary 进入 Pre-Prepare。
func (cs *clusterState) proposeRequest(req string) error {
	if cs.CurrentPhase != phaseIdle && cs.CurrentPhase != phaseReply {
		return errors.New("当前已有未完成请求")
	}
	cs.Sequence++
	digest := sha256hash.Sum256([]byte(req))
	cs.CurrentRequest = req
	cs.CurrentDigest = hex.EncodeToString(digest[:])
	cs.CurrentPhase = phaseIdle
	// 重置所有副本的当前请求状态
	for i := range cs.Replicas {
		cs.Replicas[i].CurrentDigest = ""
		cs.Replicas[i].CurrentSequence = 0
		cs.Replicas[i].PrepareReceived = map[string]bool{}
		cs.Replicas[i].CommitReceived = map[string]bool{}
		cs.Replicas[i].Prepared = false
		cs.Replicas[i].Committed = false
		cs.Replicas[i].Replied = false
	}
	cs.pushEvent("客户端请求: " + req + " (digest=" + cs.CurrentDigest[:12] + ")")
	return nil
}

// stepPhase 推进 1 个 PBFT 阶段。
func (cs *clusterState) stepPhase() {
	switch cs.CurrentPhase {
	case phaseIdle:
		cs.runPrePrepare()
	case phasePrePrepare:
		cs.runPrepare()
	case phasePrepare:
		cs.runCommit()
	case phaseCommit:
		cs.runReply()
	case phaseReply:
		cs.pushEvent("已完成 Reply，等待新 request")
	}
}

// runPrePrepare primary 把 (view, sequence, digest) 广播给所有 backup。
func (cs *clusterState) runPrePrepare() {
	if cs.CurrentRequest == "" {
		cs.pushEvent("⚠ 没有 pending request")
		return
	}
	primaryID := cs.primaryID()
	primaryIdx := -1
	for i, r := range cs.Replicas {
		if r.ID == primaryID {
			primaryIdx = i
			break
		}
	}
	if primaryIdx < 0 {
		cs.pushEvent("⚠ primary 不存在")
		return
	}
	if cs.Replicas[primaryIdx].IsDown {
		cs.pushEvent("⚠ primary " + primaryID + " 已 down，需要 view change")
		return
	}
	if cs.Replicas[primaryIdx].IsByzantine {
		// 教学：拜占庭 primary 仍发 PrePrepare，但摘要可能错（演示视图变更）
		cs.pushEvent("⚠ primary " + primaryID + " 是拜占庭，仍发 PrePrepare")
	}
	for i := range cs.Replicas {
		r := &cs.Replicas[i]
		if r.IsDown {
			continue
		}
		r.CurrentDigest = cs.CurrentDigest
		r.CurrentSequence = cs.Sequence
	}
	cs.CurrentPhase = phasePrePrepare
	cs.pushEvent(primaryID + " (primary) 广播 Pre-Prepare(v=" + fmt.Sprintf("%d, s=%d, d=%s)", cs.View, cs.Sequence, cs.CurrentDigest[:12]))
}

// runPrepare 每个非拜占庭 backup 广播 Prepare；statistics 收到 2f+1 后置 prepared。
func (cs *clusterState) runPrepare() {
	q := cs.quorum()
	// 阶段一：每个副本（含 primary）发 Prepare
	for i := range cs.Replicas {
		s := &cs.Replicas[i]
		if s.IsDown || s.CurrentDigest == "" {
			continue
		}
		if s.IsByzantine {
			// 拜占庭副本拒绝发 Prepare（或发错误摘要）—— 这里教学化为完全拒绝
			cs.pushEvent(s.ID + " 拜占庭：拒绝发 Prepare")
			continue
		}
		// 广播给所有副本（包括自己）
		for j := range cs.Replicas {
			r := &cs.Replicas[j]
			if r.IsDown || r.CurrentDigest == "" {
				continue
			}
			if r.CurrentDigest != s.CurrentDigest {
				continue // 教学：digest 不一致直接拒收
			}
			r.PrepareReceived[s.ID] = true
		}
	}
	// 阶段二：统计每个副本的 prepared 状态
	for i := range cs.Replicas {
		r := &cs.Replicas[i]
		if r.IsDown {
			continue
		}
		if len(r.PrepareReceived) >= q {
			r.Prepared = true
		}
	}
	cs.CurrentPhase = phasePrepare
	prepared := cs.countPrepared()
	cs.pushEvent(fmt.Sprintf("Prepare 阶段：%d 副本 prepared（quorum=%d）", prepared, q))
}

// runCommit 已 prepared 的非拜占庭副本广播 Commit；统计 2f+1 后 committed-local。
func (cs *clusterState) runCommit() {
	q := cs.quorum()
	for i := range cs.Replicas {
		s := &cs.Replicas[i]
		if s.IsDown || !s.Prepared {
			continue
		}
		if s.IsByzantine {
			cs.pushEvent(s.ID + " 拜占庭：拒绝发 Commit")
			continue
		}
		for j := range cs.Replicas {
			r := &cs.Replicas[j]
			if r.IsDown || r.CurrentDigest == "" {
				continue
			}
			if r.CurrentDigest != s.CurrentDigest {
				continue
			}
			r.CommitReceived[s.ID] = true
		}
	}
	for i := range cs.Replicas {
		r := &cs.Replicas[i]
		if r.IsDown {
			continue
		}
		if len(r.CommitReceived) >= q {
			r.Committed = true
		}
	}
	cs.CurrentPhase = phaseCommit
	committed := cs.countCommitted()
	cs.pushEvent(fmt.Sprintf("Commit 阶段：%d 副本 committed-local（quorum=%d）", committed, q))
}

// runReply committed-local 的副本执行 request 并 reply 客户端。
func (cs *clusterState) runReply() {
	for i := range cs.Replicas {
		r := &cs.Replicas[i]
		if !r.Committed || r.IsDown {
			continue
		}
		r.Replied = true
		r.CommittedLog = append(r.CommittedLog, committedEntry{
			Sequence: r.CurrentSequence,
			Digest:   r.CurrentDigest,
			Request:  cs.CurrentRequest,
			View:     cs.View,
		})
		if len(r.CommittedLog) > 16 {
			r.CommittedLog = r.CommittedLog[len(r.CommittedLog)-16:]
		}
	}
	cs.CurrentPhase = phaseReply
	replied := cs.countReplied()
	cs.pushEvent(fmt.Sprintf("Reply 阶段：%d 副本 reply 客户端", replied))
}

// runFullRound 一次性把当前 request 跑完 5 阶段。
func (cs *clusterState) runFullRound() {
	if cs.CurrentRequest == "" {
		return
	}
	for cs.CurrentPhase < phaseReply {
		cs.stepPhase()
	}
}

func (cs clusterState) countPrepared() int {
	n := 0
	for _, r := range cs.Replicas {
		if r.Prepared {
			n++
		}
	}
	return n
}

func (cs clusterState) countCommitted() int {
	n := 0
	for _, r := range cs.Replicas {
		if r.Committed {
			n++
		}
	}
	return n
}

func (cs clusterState) countReplied() int {
	n := 0
	for _, r := range cs.Replicas {
		if r.Replied {
			n++
		}
	}
	return n
}

// triggerViewChange 强制视图变更：每个非拜占庭非 down 副本投票 view+1；
// 新 primary 收 2f+1 票后广播 NewView 进入下一视图。
func (cs *clusterState) triggerViewChange() {
	target := cs.View + 1
	if cs.ViewChangeVotes[target] == nil {
		cs.ViewChangeVotes[target] = map[string]bool{}
	}
	for _, r := range cs.Replicas {
		if r.IsDown || r.IsByzantine {
			continue
		}
		cs.ViewChangeVotes[target][r.ID] = true
	}
	cs.pushEvent(fmt.Sprintf("ViewChange 提案 → view %d，收到 %d 票（quorum=%d）",
		target, len(cs.ViewChangeVotes[target]), cs.quorum()))
	if len(cs.ViewChangeVotes[target]) >= cs.quorum() {
		cs.View = target
		for i := range cs.Replicas {
			cs.Replicas[i].View = target
			cs.Replicas[i].PrepareReceived = map[string]bool{}
			cs.Replicas[i].CommitReceived = map[string]bool{}
			cs.Replicas[i].Prepared = false
			cs.Replicas[i].Committed = false
			cs.Replicas[i].Replied = false
			cs.Replicas[i].CurrentDigest = ""
		}
		cs.CurrentPhase = phaseIdle
		cs.pushEvent(fmt.Sprintf("✓ NewView v=%d，新 primary = %s", target, cs.primaryID()))
	}
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) clusterState {
	if s == nil || s.Data == nil {
		return defaultCluster(defaultReplicaCount)
	}
	d := s.Data
	cs := clusterState{
		View:            fw.MapInt(d, "view", 0),
		Sequence:        fw.MapInt(d, "sequence", 0),
		CurrentRequest:  fw.MapStr(d, "current_request", ""),
		CurrentDigest:   fw.MapStr(d, "current_digest", ""),
		CurrentPhase:    fw.MapInt(d, "current_phase", phaseIdle),
		LastError:       fw.MapStr(d, "last_error", ""),
		ViewChangeVotes: map[int]map[string]bool{},
	}
	if rsAny, ok := d["replicas"].([]any); ok {
		for _, rAny := range rsAny {
			if rm, ok := rAny.(map[string]any); ok {
				r := replica{
					ID:              fw.MapStr(rm, "id", ""),
					View:            fw.MapInt(rm, "view", 0),
					IsByzantine:     fw.MapBool(rm, "byzantine", false),
					IsDown:          fw.MapBool(rm, "down", false),
					CurrentDigest:   fw.MapStr(rm, "cur_digest", ""),
					CurrentSequence: fw.MapInt(rm, "cur_seq", 0),
					Prepared:        fw.MapBool(rm, "prepared", false),
					Committed:       fw.MapBool(rm, "committed", false),
					Replied:         fw.MapBool(rm, "replied", false),
					PrepareReceived: map[string]bool{},
					CommitReceived:  map[string]bool{},
				}
				if pAny, ok := rm["prepare_received"].([]any); ok {
					for _, p := range pAny {
						if s, ok := p.(string); ok {
							r.PrepareReceived[s] = true
						}
					}
				}
				if cAny, ok := rm["commit_received"].([]any); ok {
					for _, c := range cAny {
						if s, ok := c.(string); ok {
							r.CommitReceived[s] = true
						}
					}
				}
				if logAny, ok := rm["committed_log"].([]any); ok {
					for _, l := range logAny {
						if lm, ok := l.(map[string]any); ok {
							r.CommittedLog = append(r.CommittedLog, committedEntry{
								Sequence: fw.MapInt(lm, "seq", 0),
								Digest:   fw.MapStr(lm, "digest", ""),
								Request:  fw.MapStr(lm, "req", ""),
								View:     fw.MapInt(lm, "view", 0),
							})
						}
					}
				}
				cs.Replicas = append(cs.Replicas, r)
			}
		}
	}
	if len(cs.Replicas) == 0 {
		cs = defaultCluster(defaultReplicaCount)
	}
	if vcAny, ok := d["vc_votes"].(map[string]any); ok {
		for k, v := range vcAny {
			tv := atoi(k)
			cs.ViewChangeVotes[tv] = map[string]bool{}
			if listAny, ok := v.([]any); ok {
				for _, x := range listAny {
					if s, ok := x.(string); ok {
						cs.ViewChangeVotes[tv][s] = true
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
	s.Data["view"] = cs.View
	s.Data["sequence"] = cs.Sequence
	s.Data["current_request"] = cs.CurrentRequest
	s.Data["current_digest"] = cs.CurrentDigest
	s.Data["current_phase"] = cs.CurrentPhase
	s.Data["last_error"] = cs.LastError
	rs := make([]any, len(cs.Replicas))
	for i, r := range cs.Replicas {
		prepared := []any{}
		for k := range r.PrepareReceived {
			prepared = append(prepared, k)
		}
		committed := []any{}
		for k := range r.CommitReceived {
			committed = append(committed, k)
		}
		log := make([]any, len(r.CommittedLog))
		for j, l := range r.CommittedLog {
			log[j] = map[string]any{"seq": l.Sequence, "digest": l.Digest, "req": l.Request, "view": l.View}
		}
		rs[i] = map[string]any{
			"id":               r.ID,
			"view":             r.View,
			"byzantine":        r.IsByzantine,
			"down":             r.IsDown,
			"cur_digest":       r.CurrentDigest,
			"cur_seq":          r.CurrentSequence,
			"prepared":         r.Prepared,
			"committed":        r.Committed,
			"replied":          r.Replied,
			"prepare_received": prepared,
			"commit_received":  committed,
			"committed_log":    log,
		}
	}
	s.Data["replicas"] = rs
	vc := map[string]any{}
	for tv, voters := range cs.ViewChangeVotes {
		list := []any{}
		for v := range voters {
			list = append(list, v)
		}
		vc[fmt.Sprintf("%d", tv)] = list
	}
	s.Data["vc_votes"] = vc
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
		Name:                "实用拜占庭容错（PBFT）",
		Description:         "演示 PBFT 5 阶段（Request → Pre-Prepare → Prepare → Commit → Reply）+ View Change + 拜占庭注入",
		Category:            fw.CategoryConsensus,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPbftAttack},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"consensus.pbft.phase",
			"consensus.pbft.committed",
			"consensus.pbft.byzantine_set",
			"consensus.pbft.view",
			"consensus.pbft.primary",
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
		Phase:     "idle",
		Data:      map[string]any{},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_replicas", Label: "设置副本数",
				Description: "n = 3f + 1（4=最小，f=1；7→f=2）",
				Category:    fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "副本数（4 或 7）", Required: true, Default: defaultReplicaCount, Min: 4, Max: maxReplicaCount, Step: 3},
				},
			},
			{
				ActionCode: "propose_request", Label: "提交客户端请求",
				Description:   "primary 接到请求，进入 Pre-Prepare 阶段（推进 step_phase）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "request", Type: fw.FieldString, Label: "请求内容", Required: true, Default: "set x=1"},
				},
				WritesOwnedFields: []string{"consensus.pbft.phase"},
				LinkOwnerFields:   []string{"consensus.pbft.phase"},
			},
			{
				ActionCode: "step_phase", Label: "推进 1 阶段",
				Description: "推进当前 PBFT 阶段（5 阶段循环）",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"consensus.pbft.phase", "consensus.pbft.committed"},
				LinkOwnerFields:   []string{"consensus.pbft.phase", "consensus.pbft.committed"},
			},
			{
				ActionCode: "run_full_round", Label: "完整跑完一轮",
				Description: "一次性推进 5 阶段直到 Reply",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"consensus.pbft.committed"},
				LinkOwnerFields:   []string{"consensus.pbft.committed"},
			},
			{
				ActionCode: "inject_byzantine", Label: "注入拜占庭副本",
				Description:   "把指定副本标记为 byzantine（拒绝发 Prepare/Commit）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "replica_id", Type: fw.FieldString, Label: "副本 ID", Required: true, Default: "r0"},
				},
				WritesOwnedFields: []string{"consensus.pbft.byzantine_set"},
				LinkOwnerFields:   []string{"consensus.pbft.byzantine_set"},
			},
			{
				ActionCode: "recover_byzantine", Label: "恢复拜占庭副本",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
				Fields: []fw.FieldDef{
					{Name: "replica_id", Type: fw.FieldString, Label: "副本 ID", Required: true, Default: "r0"},
				},
			},
			{
				ActionCode: "view_change", Label: "强制视图变更",
				Description: "切换到 view + 1，新 primary = (view+1) mod n",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"consensus.pbft.view", "consensus.pbft.primary"},
				LinkOwnerFields:   []string{"consensus.pbft.view", "consensus.pbft.primary"},
			},
			{
				ActionCode: "crash_replica", Label: "副本崩溃",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "replica_id", Type: fw.FieldString, Label: "副本 ID", Required: true, Default: "r1"},
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
				Description:   "仅教师可用，注入拜占庭故障用于教学演示",
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
	state.Phase = "idle"
	env := buildEnvelope(cs, "init", "PBFT 初始化（4 副本，f=1，quorum=3）", true)
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
	case "set_replicas":
		n := fw.MapInt(in.Params, "n", defaultReplicaCount)
		cs = defaultCluster(n)
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "set_replicas", fmt.Sprintf("已重置为 %d 副本（f=%d, quorum=%d）", n, cs.faultThreshold(), cs.quorum()), true)
		return out, nil

	case "propose_request":
		req := fw.MapStr(in.Params, "request", "set x=1")
		if err := cs.proposeRequest(req); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "propose_request", "客户端请求已接收", false)
		appendProposeMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "step_phase":
		cs.stepPhase()
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "step_phase",
			fmt.Sprintf("推进到 %s 阶段", phaseLabels[cs.CurrentPhase]), false)
		appendStepPhaseMicroSteps(&out.Render, cs.CurrentPhase)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "run_full_round":
		cs.runFullRound()
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "run_full_round", "一轮共识完成", false)
		appendFullRoundMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "inject_byzantine":
		rid := fw.MapStr(in.Params, "replica_id", "r0")
		idx := -1
		for i, r := range cs.Replicas {
			if r.ID == rid {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到副本: " + rid}, nil
		}
		cs.Replicas[idx].IsByzantine = true
		cs.pushEvent("⚠ " + rid + " 标记为拜占庭")
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "inject_byzantine", rid+" 注入拜占庭", false)
		appendByzantineMicroSteps(&out.Render, rid)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "recover_byzantine":
		rid := fw.MapStr(in.Params, "replica_id", "r0")
		for i := range cs.Replicas {
			if cs.Replicas[i].ID == rid {
				cs.Replicas[i].IsByzantine = false
				cs.pushEvent("✓ " + rid + " 恢复正常")
				break
			}
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "recover_byzantine", rid+" 恢复正常", false)
		return out, nil

	case "view_change":
		cs.triggerViewChange()
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "view_change",
			fmt.Sprintf("ViewChange → view = %d, primary = %s", cs.View, cs.primaryID()), false)
		appendViewChangeMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(cs)
		return out, nil

	case "crash_replica":
		rid := fw.MapStr(in.Params, "replica_id", "r1")
		for i := range cs.Replicas {
			if cs.Replicas[i].ID == rid {
				cs.Replicas[i].IsDown = true
				cs.pushEvent("⚠ " + rid + " 节点崩溃")
				break
			}
		}
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "crash_replica", rid+" 已崩溃", false)
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
		cs = defaultCluster(defaultReplicaCount)
		saveState(state, cs)
		out.Render = buildEnvelope(cs, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(cs clusterState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 50)

	// 1) 流水线 5 阶段
	phaseNodeIDs := make([]string, 5)
	for i := range phaseNodeIDs {
		phaseNodeIDs[i] = "phase-" + phaseRoles[i]
	}
	prims = append(prims, fw.PrimStack("phases", phaseNodeIDs, "horizontal"))
	for i, id := range phaseNodeIDs {
		status := "normal"
		if i == cs.CurrentPhase {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(id, phaseLabels[i], status, phaseRoles[i]))
	}
	for i := 0; i < 4; i++ {
		anim := ""
		if i < cs.CurrentPhase {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-ph-%d", i), phaseNodeIDs[i], phaseNodeIDs[i+1], "solid", anim))
	}

	// 2) 阶段进度条
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, cs.CurrentPhase, float64(cs.CurrentPhase)/4.0))

	// 3) 副本环（按 cs.Replicas 顺序声明 ring 成员，渲染器据此把 PrimNode 落到 slot）
	replicaNodeIDs := make([]string, len(cs.Replicas))
	for i, r := range cs.Replicas {
		replicaNodeIDs[i] = "rep-" + r.ID
	}
	prims = append(prims, fw.PrimRingLayout("replica-ring", replicaNodeIDs))
	primaryID := cs.primaryID()
	for _, r := range cs.Replicas {
		status := "normal"
		role := "backup"
		if r.IsDown {
			status = "error"
			role = "down"
		} else if r.IsByzantine {
			status = "warning"
			role = "byzantine"
		} else if r.ID == primaryID {
			status = "active"
			role = "primary"
		}
		stage := "idle"
		if r.Replied {
			stage = "replied"
		} else if r.Committed {
			stage = "committed"
		} else if r.Prepared {
			stage = "prepared"
		} else if r.CurrentDigest != "" {
			stage = "pre-prepared"
		}
		label := fmt.Sprintf("%s\n%s\nv=%d\n%s", r.ID, role, r.View, stage)
		if r.ID == primaryID {
			label = fmt.Sprintf("👑 %s\nPRIMARY\nv=%d\n%s", r.ID, r.View, stage)
		}
		prims = append(prims, fw.PrimNode("rep-"+r.ID, label, status, role))
	}

	// 4) 副本互连边（primary → backup 高亮 flow）
	for i := 0; i < len(cs.Replicas); i++ {
		for j := i + 1; j < len(cs.Replicas); j++ {
			a, b := cs.Replicas[i], cs.Replicas[j]
			anim := ""
			style := "solid"
			if a.IsDown || b.IsDown {
				style = "dashed"
			}
			if cs.CurrentPhase == phasePrePrepare && (a.ID == primaryID || b.ID == primaryID) {
				anim = "flow"
			} else if cs.CurrentPhase == phasePrepare || cs.CurrentPhase == phaseCommit {
				anim = "flow"
			}
			prims = append(prims, fw.PrimEdge(
				fmt.Sprintf("edge-%s-%s", a.ID, b.ID),
				"rep-"+a.ID, "rep-"+b.ID, style, anim))
		}
	}

	// 5) Prepare / Commit 投票矩阵：行=副本，列=Prepare 来源（每个副本作为 sender 是否被本人接收）
	N := len(cs.Replicas)
	cells := make([]map[string]any, 0, N*N*2)
	// Prepare 矩阵（左半）
	for i, r := range cs.Replicas {
		for j, s := range cs.Replicas {
			val := ""
			color := "muted"
			if r.PrepareReceived[s.ID] {
				val = "P"
				color = "info"
			}
			if r.CommitReceived[s.ID] {
				val = "C"
				color = "success"
			}
			if r.PrepareReceived[s.ID] && r.CommitReceived[s.ID] {
				val = "P+C"
				color = "success"
			}
			cells = append(cells, map[string]any{
				"row": i, "col": j, "value": val, "color_role": color,
			})
		}
	}
	prims = append(prims, fw.PrimVoteMatrix("vote-matrix", N, N, cells))

	// 6) 公式
	prims = append(prims, fw.PrimMathFormula("formula-quorum",
		`n = 3f + 1,\quad \mathrm{quorum} = 2f + 1`, false))
	prims = append(prims, fw.PrimMathFormula("formula-primary",
		`\mathrm{primary}(v) = v \bmod n`, false))

	// 7) 关键状态
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("view = %d\nsequence = %d\nprimary = %s\nphase = %s\nactive replicas = %d / %d\nf = %d, quorum = %d\ndigest = %s\nrequest = %s",
			cs.View, cs.Sequence, primaryID, phaseLabels[cs.CurrentPhase],
			cs.activeReplicas(), len(cs.Replicas),
			cs.faultThreshold(), cs.quorum(),
			truncStr(cs.CurrentDigest, 16), cs.CurrentRequest),
		"text", nil, 10))

	// 8) 副本表
	rows := []string{"id   role       view  prep  comm  reply  prep_recv  comm_recv  log"}
	for _, r := range cs.Replicas {
		role := "backup"
		if r.IsDown {
			role = "DOWN"
		} else if r.IsByzantine {
			role = "BYZANTINE"
		} else if r.ID == primaryID {
			role = "primary"
		}
		rows = append(rows, fmt.Sprintf("%-4s %-10s %-4d  %-4v  %-4v  %-5v  %-9d  %-9d  %d",
			r.ID, role, r.View, r.Prepared, r.Committed, r.Replied,
			len(r.PrepareReceived), len(r.CommitReceived), len(r.CommittedLog)))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-replicas", strings.Join(rows, "\n"), "text", nil, 14))

	// 9) 事件日志
	if len(cs.EventLog) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(cs.EventLog, "\n"), "text", nil, 24))
	}

	// 10) 已 commit log（任取首个非拜占庭副本展示）
	commitLines := []string{"已确认请求（committed log）："}
	for _, r := range cs.Replicas {
		if r.IsDown || r.IsByzantine {
			continue
		}
		for _, l := range r.CommittedLog {
			commitLines = append(commitLines, fmt.Sprintf("  s=%d v=%d  digest=%s  req=%s",
				l.Sequence, l.View, truncStr(l.Digest, 12), l.Request))
		}
		break
	}
	prims = append(prims, fw.PrimCodeBlock("cb-commit-log", strings.Join(commitLines, "\n"), "text", nil, 12))

	// 11) 动效
	if primaryID != "" {
		prims = append(prims, fw.PrimGlow("glow-primary", "rep-"+primaryID, "success", 0.9))
	}
	for _, r := range cs.Replicas {
		if r.IsByzantine && !r.IsDown {
			prims = append(prims, fw.PrimGlow("glow-byz-"+r.ID, "rep-"+r.ID, "warning", 0.8))
			prims = append(prims, fw.PrimShake("shake-byz-"+r.ID, "rep-"+r.ID, 0.3, 600))
		}
	}
	if cs.CurrentPhase >= phasePrePrepare && cs.CurrentPhase <= phaseCommit {
		prims = append(prims, fw.PrimPulse("pulse-phase", phaseNodeIDs[cs.CurrentPhase], "info", 1500))
	}
	if cs.CurrentPhase == phaseReply {
		prims = append(prims, fw.PrimBurst("burst-reply", phaseNodeIDs[phaseReply], "success",
			int64(cs.Sequence), 800))
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pbft", linkGroupPbftAttack, "idle", ""))

	if cs.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "PBFT 错误", cs.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(cs, summary),
	}
}

func buildSidePanelData(cs clusterState, summary string) map[string]any {
	byzantine := []string{}
	down := []string{}
	for _, r := range cs.Replicas {
		if r.IsByzantine {
			byzantine = append(byzantine, r.ID)
		}
		if r.IsDown {
			down = append(down, r.ID)
		}
	}
	d := map[string]any{
		"view":          cs.View,
		"sequence":      cs.Sequence,
		"primary":       cs.primaryID(),
		"phase":         phaseLabels[cs.CurrentPhase],
		"phase_index":   cs.CurrentPhase,
		"replica_count": len(cs.Replicas),
		"f":             cs.faultThreshold(),
		"quorum":        cs.quorum(),
		"prepared":      cs.countPrepared(),
		"committed":     cs.countCommitted(),
		"replied":       cs.countReplied(),
		"byzantine_set": byzantine,
		"down_set":      down,
		"digest":        cs.CurrentDigest,
		"request":       cs.CurrentRequest,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendProposeMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "pp-1", Label: "客户端发送请求到 primary", DurationMs: 400, HighlightIDs: []string{"phase-idle", "cb-status"}, ParentPhase: "pre-prepare"},
		{ID: "pp-2", Label: "计算 SHA-256 摘要", DurationMs: 400, HighlightIDs: []string{"cb-status"}, FirePrimitives: []string{"glow-primary"}},
		{ID: "pp-3", Label: "primary 准备 Pre-Prepare 广播", DurationMs: 400, HighlightIDs: []string{"phase-pre-prepare"}, IsLinkTrigger: true},
	}
}

func appendStepPhaseMicroSteps(env *fw.RenderEnvelope, phase int) {
	switch phase {
	case phasePrePrepare:
		env.MicroSteps = []fw.MicroStep{
			{ID: "pp-1", Label: "primary 广播 (v, s, digest)", DurationMs: 500, HighlightIDs: []string{"phase-pre-prepare", "replica-ring"}, FirePrimitives: []string{"pulse-phase", "glow-primary"}},
			{ID: "pp-2", Label: "所有 backup 接收", DurationMs: 500, HighlightIDs: []string{"cb-replicas"}},
			{ID: "pp-3", Label: "进入 Pre-Prepared 状态", DurationMs: 400, HighlightIDs: []string{"vote-matrix"}, IsLinkTrigger: true},
		}
	case phasePrepare:
		env.MicroSteps = []fw.MicroStep{
			{ID: "pr-1", Label: "每个非拜占庭副本广播 Prepare", DurationMs: 500, HighlightIDs: []string{"phase-prepare"}, FirePrimitives: []string{"pulse-phase"}},
			{ID: "pr-2", Label: "统计每副本收到的 Prepare 数", DurationMs: 500, HighlightIDs: []string{"vote-matrix"}},
			{ID: "pr-3", Label: "≥ 2f+1 → prepared", DurationMs: 500, HighlightIDs: []string{"formula-quorum", "cb-replicas"}, IsLinkTrigger: true},
		}
	case phaseCommit:
		env.MicroSteps = []fw.MicroStep{
			{ID: "cm-1", Label: "已 prepared 副本广播 Commit", DurationMs: 500, HighlightIDs: []string{"phase-commit"}, FirePrimitives: []string{"pulse-phase"}},
			{ID: "cm-2", Label: "统计 Commit 数 ≥ 2f+1", DurationMs: 500, HighlightIDs: []string{"vote-matrix", "formula-quorum"}},
			{ID: "cm-3", Label: "committed-local → 可执行", DurationMs: 500, HighlightIDs: []string{"cb-replicas"}, IsLinkTrigger: true},
		}
	case phaseReply:
		env.MicroSteps = []fw.MicroStep{
			{ID: "rp-1", Label: "执行 request 写入 log", DurationMs: 500, HighlightIDs: []string{"cb-commit-log"}, FirePrimitives: []string{"burst-reply"}},
			{ID: "rp-2", Label: "Reply 给客户端", DurationMs: 500, HighlightIDs: []string{"phase-reply"}, IsLinkTrigger: true},
			{ID: "rp-3", Label: "等待新 request", DurationMs: 400, HighlightIDs: []string{"cb-status"}},
		}
	}
}

func appendFullRoundMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "fr-1", Label: "Pre-Prepare 广播", DurationMs: 400, HighlightIDs: []string{"phase-pre-prepare"}, FirePrimitives: []string{"glow-primary"}},
		{ID: "fr-2", Label: "Prepare 收集 quorum", DurationMs: 400, HighlightIDs: []string{"phase-prepare", "vote-matrix"}, FirePrimitives: []string{"pulse-phase"}},
		{ID: "fr-3", Label: "Commit 收集 quorum", DurationMs: 400, HighlightIDs: []string{"phase-commit", "formula-quorum"}},
		{ID: "fr-4", Label: "Reply 客户端", DurationMs: 500, HighlightIDs: []string{"phase-reply", "cb-commit-log"}, FirePrimitives: []string{"burst-reply"}, IsLinkTrigger: true},
	}
}

func appendByzantineMicroSteps(env *fw.RenderEnvelope, rid string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "bz-1", Label: rid + " 标记为拜占庭", DurationMs: 400, HighlightIDs: []string{"rep-" + rid}, FirePrimitives: []string{"shake-byz-" + rid}},
		{ID: "bz-2", Label: "在 Prepare/Commit 阶段拒绝发消息", DurationMs: 500, HighlightIDs: []string{"cb-events", "vote-matrix"}},
		{ID: "bz-3", Label: "若 byzantine ≤ f，仍能达成共识", DurationMs: 500, HighlightIDs: []string{"formula-quorum"}, IsLinkTrigger: true},
	}
}

func appendViewChangeMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "vc-1", Label: "副本怀疑 primary 不可用", DurationMs: 400, HighlightIDs: []string{"cb-events"}},
		{ID: "vc-2", Label: "广播 ViewChange(target=v+1)", DurationMs: 500, HighlightIDs: []string{"vote-matrix"}, FirePrimitives: []string{"pulse-phase"}},
		{ID: "vc-3", Label: "新 primary 收 2f+1 票 → NewView", DurationMs: 500, HighlightIDs: []string{"cb-status", "formula-primary"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, cs clusterState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "pbft-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_consensus",
		LinkGroup:      linkGroupPbftAttack,
		ChangedFields:  []string{"consensus.pbft.phase", "consensus.pbft.view"},
		Payload: map[string]any{
			"view":    cs.View,
			"phase":   phaseLabels[cs.CurrentPhase],
			"primary": cs.primaryID(),
		},
		SourceAnchorID: "pbft-output-anchor",
		TargetAnchorID: "attack-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "consensus.pbft.phase", "consensus.pbft.view")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(cs clusterState) map[string]any {
	byzantine := []string{}
	for _, r := range cs.Replicas {
		if r.IsByzantine {
			byzantine = append(byzantine, r.ID)
		}
	}
	committedSeq := -1
	for _, r := range cs.Replicas {
		if r.IsByzantine || r.IsDown {
			continue
		}
		if len(r.CommittedLog) > 0 {
			last := r.CommittedLog[len(r.CommittedLog)-1].Sequence
			if last > committedSeq {
				committedSeq = last
			}
		}
	}
	return map[string]any{
		"consensus": map[string]any{
			"pbft": map[string]any{
				"view":          cs.View,
				"sequence":      cs.Sequence,
				"primary":       cs.primaryID(),
				"phase":         phaseLabels[cs.CurrentPhase],
				"phase_index":   cs.CurrentPhase,
				"committed":     committedSeq,
				"byzantine_set": byzantine,
				"replica_count": len(cs.Replicas),
				"f":             cs.faultThreshold(),
				"quorum":        cs.quorum(),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func atoi(s string) int {
	n := 0
	sign := 1
	i := 0
	if len(s) > 0 && s[0] == '-' {
		sign = -1
		i = 1
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n * sign
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
