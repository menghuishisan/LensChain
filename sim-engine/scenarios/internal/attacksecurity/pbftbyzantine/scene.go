// 模块：sim-engine/scenarios/internal/attacksecurity/pbftbyzantine
// 文件职责：ATK-03 PBFT 拜占庭攻击场景的完整实现。
//
// SSOT 依据：06.md §4.7.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 PBFT 三阶段共识 + 拜占庭节点行为（零外部依赖；keccak 复用 keccak256hash）。
//
//   1. 节点模型：N = 3f+1 个节点（默认 N=4，f=1）；编号 0..N-1
//      每个节点：role ∈ {primary, backup}；fault ∈ {honest, equivocating, silent, lying}
//      view = 当前视图编号；primary = view mod N
//
//   2. PBFT 三阶段（针对单个 request）：
//        a) PRE-PREPARE  : primary 广播 (view, seq, digest(req))；
//                          equivocating primary 可向不同子集发送不同 digest
//        b) PREPARE      : 每个 backup 收到 PRE-PREPARE 后广播 PREPARE(view, seq, digest)；
//                          需要 2f 个 PREPARE（不含自己） → "prepared"
//        c) COMMIT       : 进入 prepared 后广播 COMMIT(view, seq, digest)；
//                          收齐 2f+1 个 COMMIT → "committed-local" → 应用到状态机
//
//   3. 拜占庭故障类型：
//      · honest        : 按协议执行
//      · equivocating  : 同 view 同 seq 给不同节点发送冲突 digest（双花视图）
//      · silent        : 不发送任何消息（等价崩溃）
//      · lying         : 发送随机错误的 digest
//
//   4. 安全性：
//      · 容忍 f = (N-1)/3 个拜占庭节点
//      · 若 byzantine ≤ f → safety + liveness 都成立
//      · 若 byzantine > f → 可能出现 fork（不同诚实节点 commit 不同 digest）→ 安全性破坏
//
//   5. View Change（教学版）：
//      · 节点检测到 primary 故障（超时/equivocate）→ 触发 view + 1
//      · 新 primary = (view+1) mod N，从 prepared 集合中重发未完成的 request
//
//   6. 攻击演示：
//      · attack_equivocate     : 设置 primary 为 equivocating
//      · attack_silent_primary : primary 静默 → 触发 view-change
//      · attack_more_than_f    : 把 byzantine 数提到 f+1 → 演示安全性失败

package pbftbyzantine

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
)

const (
	sceneCode     = "pbft-byzantine"
	schemaVersion = "v1.0.0"
	algorithmType = "pbft-with-faults"

	faultHonest       = "honest"
	faultEquivocating = "equivocating"
	faultSilent       = "silent"
	faultLying        = "lying"

	phaseIdle       = "idle"
	phasePrePrepare = "pre-prepare"
	phasePrepare    = "prepare"
	phaseCommit     = "commit"
	phaseDone       = "committed"
	phaseViewChange = "view-change"
	phaseFork       = "FORK"

	linkGroupPbftAttack = "pbft-attack-group"
	linkOwnerSubtree    = "attack.pbft"
)

// =====================================================================
// 数据结构
// =====================================================================

type pbftNode struct {
	ID    int
	Fault string
	View  uint64
	// 收到的消息（按 (view, seq, fromNode) 索引）
	PrePrepares map[string]string   // key = (view, seq) → digest
	Prepares    map[string][]string // key = (view, seq, digest) → list of from
	Commits     map[string][]string

	Prepared  map[string]bool   // (view, seq, digest) → 已 prepared
	Committed map[string]string // (view, seq) → digest（本节点 commit 的）

	Inbox []msgEnvelope // 当前 tick 待处理消息
	Log   []nodeLogEntry
}

type nodeLogEntry struct {
	Tick  int
	Phase string
	Note  string
}

type request struct {
	Seq    uint64
	Op     string // 客户端请求内容
	Digest string
}

type msgKind int

const (
	msgPrePrepare msgKind = iota
	msgPrepare
	msgCommit
	msgViewChange
)

type msgEnvelope struct {
	Tick   int
	From   int
	To     int
	Kind   msgKind
	View   uint64
	Seq    uint64
	Digest string
}

type forkEvidence struct {
	View    uint64
	Seq     uint64
	Digests map[string][]int // digest → 哪些 honest 节点 commit 了它
}

type snapState struct {
	N             int
	F             int
	Tick          int
	Seed          string
	View          uint64
	NextSeq       uint64
	Nodes         []*pbftNode
	Requests      []request
	Messages      []msgEnvelope // 历史日志
	ViewChanges   int
	ForkCount     int
	Forks         []forkEvidence
	HonestCommits map[string]map[string]int // (view,seq) → digest → count of honest committers
	LastError     string
}

func defaultSnapState() snapState {
	return newSnapState(4, faultHonest)
}

func newSnapState(n int, primaryFault string) snapState {
	st := snapState{
		N: n, F: (n - 1) / 3, Seed: "lenschain-pbft",
		HonestCommits: map[string]map[string]int{},
	}
	for i := 0; i < n; i++ {
		nd := &pbftNode{ID: i, Fault: faultHonest,
			PrePrepares: map[string]string{},
			Prepares:    map[string][]string{},
			Commits:     map[string][]string{},
			Prepared:    map[string]bool{},
			Committed:   map[string]string{},
		}
		st.Nodes = append(st.Nodes, nd)
	}
	if n > 0 {
		st.Nodes[0].Fault = primaryFault
	}
	return st
}

func (st snapState) primary() int { return int(st.View % uint64(st.N)) }

// digestOp 模拟客户端请求的 digest = keccak256(seed||seq||op)。
func (st snapState) digestOp(seq uint64, op string) string {
	buf := []byte(st.Seed)
	var num [8]byte
	binary.BigEndian.PutUint64(num[:], seq)
	buf = append(buf, num[:]...)
	buf = append(buf, []byte(op)...)
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:6])
}

// =====================================================================
// 核心：消息广播 + 三阶段
// =====================================================================

// startRequest 客户端发起请求 → primary 广播 PRE-PREPARE。
func (st *snapState) startRequest(op string) (*request, error) {
	st.NextSeq++
	r := request{Seq: st.NextSeq, Op: op}
	r.Digest = st.digestOp(r.Seq, r.Op)
	st.Requests = append(st.Requests, r)
	primary := st.Nodes[st.primary()]
	primary.Log = append(primary.Log, nodeLogEntry{Tick: st.Tick, Phase: phasePrePrepare,
		Note: fmt.Sprintf("primary 广播 PRE-PREPARE(v=%d, seq=%d, %s)", st.View, r.Seq, r.Digest)})

	switch primary.Fault {
	case faultSilent:
		primary.Log = append(primary.Log, nodeLogEntry{Tick: st.Tick, Phase: phasePrePrepare, Note: "✗ primary 静默"})
		// 触发视图变换
		st.triggerViewChange("primary 静默")
	case faultEquivocating:
		// 给前 N/2 节点发 digestA，其余发 digestA' （把 op 加 "_evil"）
		evilDigest := st.digestOp(r.Seq, r.Op+"_evil")
		for i := range st.Nodes {
			if i == primary.ID {
				continue
			}
			d := r.Digest
			if i >= st.N/2 {
				d = evilDigest
			}
			st.deliver(msgEnvelope{Tick: st.Tick, From: primary.ID, To: i, Kind: msgPrePrepare,
				View: st.View, Seq: r.Seq, Digest: d})
		}
	case faultLying:
		bad := st.digestOp(r.Seq, "wrong-"+op)
		for i := range st.Nodes {
			if i == primary.ID {
				continue
			}
			st.deliver(msgEnvelope{Tick: st.Tick, From: primary.ID, To: i, Kind: msgPrePrepare,
				View: st.View, Seq: r.Seq, Digest: bad})
		}
	default:
		for i := range st.Nodes {
			if i == primary.ID {
				continue
			}
			st.deliver(msgEnvelope{Tick: st.Tick, From: primary.ID, To: i, Kind: msgPrePrepare,
				View: st.View, Seq: r.Seq, Digest: r.Digest})
		}
	}
	return &r, nil
}

// deliver 立即把消息加入目标节点 inbox 并记录全局历史。
func (st *snapState) deliver(m msgEnvelope) {
	if m.To < 0 || m.To >= st.N {
		return
	}
	st.Nodes[m.To].Inbox = append(st.Nodes[m.To].Inbox, m)
	st.Messages = append(st.Messages, m)
	if len(st.Messages) > 256 {
		st.Messages = st.Messages[len(st.Messages)-256:]
	}
}

// processInbox 让所有节点处理当前 inbox（一轮 round）。
func (st *snapState) processInbox() {
	st.Tick++
	// 单轮处理：每个节点把 inbox 清空，根据 fault 决定是否广播 PREPARE / COMMIT
	for _, nd := range st.Nodes {
		batch := nd.Inbox
		nd.Inbox = nil
		for _, m := range batch {
			st.handleMessage(nd, m)
		}
	}
	// 二轮：处理本轮新入 inbox（PREPARE / COMMIT 反馈）
	for _, nd := range st.Nodes {
		batch := nd.Inbox
		nd.Inbox = nil
		for _, m := range batch {
			st.handleMessage(nd, m)
		}
	}
	st.detectForks()
}

func (st *snapState) handleMessage(nd *pbftNode, m msgEnvelope) {
	if nd.Fault == faultSilent {
		nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: "drop", Note: "节点静默，丢弃所有消息"})
		return
	}
	switch m.Kind {
	case msgPrePrepare:
		st.handlePrePrepare(nd, m)
	case msgPrepare:
		st.handlePrepare(nd, m)
	case msgCommit:
		st.handleCommit(nd, m)
	case msgViewChange:
		st.handleViewChange(nd, m)
	}
}

func keyVS(v, s uint64) string { return fmt.Sprintf("%d/%d", v, s) }
func keyVSD(v, s uint64, d string) string {
	return fmt.Sprintf("%d/%d/%s", v, s, d)
}

func (st *snapState) handlePrePrepare(nd *pbftNode, m msgEnvelope) {
	k := keyVS(m.View, m.Seq)
	prev, exists := nd.PrePrepares[k]
	if exists && prev != m.Digest {
		nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phasePrePrepare,
			Note: fmt.Sprintf("⚠ 检测到 equivocate: %s ≠ %s", prev, m.Digest)})
	}
	nd.PrePrepares[k] = m.Digest
	nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phasePrePrepare,
		Note: fmt.Sprintf("收到 PRE-PREPARE(v=%d, s=%d, %s)，广播 PREPARE", m.View, m.Seq, m.Digest)})
	// 广播 PREPARE
	d := m.Digest
	if nd.Fault == faultLying {
		d = st.digestOp(m.Seq, fmt.Sprintf("lie-from-%d", nd.ID))
	}
	for i := range st.Nodes {
		if i == nd.ID {
			continue
		}
		st.deliver(msgEnvelope{Tick: st.Tick, From: nd.ID, To: i, Kind: msgPrepare,
			View: m.View, Seq: m.Seq, Digest: d})
	}
}

func (st *snapState) handlePrepare(nd *pbftNode, m msgEnvelope) {
	k := keyVSD(m.View, m.Seq, m.Digest)
	// 去重
	for _, f := range nd.Prepares[k] {
		if f == fmt.Sprintf("n%d", m.From) {
			return
		}
	}
	nd.Prepares[k] = append(nd.Prepares[k], fmt.Sprintf("n%d", m.From))
	// 加上自己（若自己已 PrePrepare 同 digest）
	count := len(nd.Prepares[k])
	// 如果 prePrepare matches，自己也算 1
	pp, ok := nd.PrePrepares[keyVS(m.View, m.Seq)]
	if ok && pp == m.Digest {
		count++ // 包含自己的 PREPARE
	}
	if !nd.Prepared[k] && count >= 2*st.F {
		nd.Prepared[k] = true
		nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phasePrepare,
			Note: fmt.Sprintf("✓ prepared (v=%d, s=%d, %s)，广播 COMMIT", m.View, m.Seq, m.Digest)})
		// 广播 COMMIT
		for i := range st.Nodes {
			if i == nd.ID {
				continue
			}
			st.deliver(msgEnvelope{Tick: st.Tick, From: nd.ID, To: i, Kind: msgCommit,
				View: m.View, Seq: m.Seq, Digest: m.Digest})
		}
	}
}

func (st *snapState) handleCommit(nd *pbftNode, m msgEnvelope) {
	k := keyVSD(m.View, m.Seq, m.Digest)
	for _, f := range nd.Commits[k] {
		if f == fmt.Sprintf("n%d", m.From) {
			return
		}
	}
	nd.Commits[k] = append(nd.Commits[k], fmt.Sprintf("n%d", m.From))
	count := len(nd.Commits[k])
	if nd.Prepared[k] {
		count++ // 自己也算
	}
	if count >= 2*st.F+1 {
		seqK := keyVS(m.View, m.Seq)
		if _, already := nd.Committed[seqK]; !already {
			nd.Committed[seqK] = m.Digest
			nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phaseCommit,
				Note: fmt.Sprintf("✓ committed-local (v=%d, s=%d, %s)", m.View, m.Seq, m.Digest)})
			// 全局统计 honest commits
			if nd.Fault == faultHonest {
				if _, ok := st.HonestCommits[seqK]; !ok {
					st.HonestCommits[seqK] = map[string]int{}
				}
				st.HonestCommits[seqK][m.Digest]++
			}
		}
	}
}

func (st *snapState) handleViewChange(nd *pbftNode, m msgEnvelope) {
	nd.View = m.View
	nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phaseViewChange,
		Note: fmt.Sprintf("接受 view-change → view=%d，新 primary=%d", m.View, int(m.View)%st.N)})
}

// triggerViewChange 触发视图变换：所有 honest 节点广播 VC。
func (st *snapState) triggerViewChange(reason string) {
	st.View++
	st.ViewChanges++
	for _, nd := range st.Nodes {
		nd.Log = append(nd.Log, nodeLogEntry{Tick: st.Tick, Phase: phaseViewChange,
			Note: fmt.Sprintf("view-change → %d (%s)", st.View, reason)})
	}
	// 广播 VC
	for from, ndF := range st.Nodes {
		if ndF.Fault == faultSilent {
			continue
		}
		for to := range st.Nodes {
			if to == from {
				continue
			}
			st.deliver(msgEnvelope{Tick: st.Tick, From: from, To: to, Kind: msgViewChange, View: st.View})
		}
	}
}

// detectForks 检查同 (view, seq) 下，是否有 honest 节点 commit 了不同 digest。
func (st *snapState) detectForks() {
	for k, m := range st.HonestCommits {
		if len(m) >= 2 {
			// fork
			parts := strings.Split(k, "/")
			var v, s uint64
			fmt.Sscanf(parts[0], "%d", &v)
			fmt.Sscanf(parts[1], "%d", &s)
			// 把 digests 中每个 → 哪些诚实节点 commit
			ev := forkEvidence{View: v, Seq: s, Digests: map[string][]int{}}
			for _, nd := range st.Nodes {
				if nd.Fault != faultHonest {
					continue
				}
				if d, ok := nd.Committed[k]; ok {
					ev.Digests[d] = append(ev.Digests[d], nd.ID)
				}
			}
			if !st.alreadyHasFork(ev) {
				st.Forks = append(st.Forks, ev)
				st.ForkCount++
			}
		}
	}
}

func (st snapState) alreadyHasFork(ev forkEvidence) bool {
	for _, f := range st.Forks {
		if f.View == ev.View && f.Seq == ev.Seq {
			return true
		}
	}
	return false
}

// configureFault 设置某节点的 fault 行为。
func (st *snapState) configureFault(nodeID int, fault string) error {
	if nodeID < 0 || nodeID >= st.N {
		return fmt.Errorf("nodeID 越界: %d", nodeID)
	}
	switch fault {
	case faultHonest, faultEquivocating, faultSilent, faultLying:
	default:
		return fmt.Errorf("未知 fault 类型: %s", fault)
	}
	st.Nodes[nodeID].Fault = fault
	return nil
}

func (st snapState) byzantineCount() int {
	cnt := 0
	for _, nd := range st.Nodes {
		if nd.Fault != faultHonest {
			cnt++
		}
	}
	return cnt
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		N: fw.MapInt(d, "n", 4), F: fw.MapInt(d, "f", 1),
		Tick: fw.MapInt(d, "tick", 0), Seed: fw.MapStr(d, "seed", "lenschain-pbft"),
		View: uint64(fw.MapInt(d, "view", 0)), NextSeq: uint64(fw.MapInt(d, "next_seq", 0)),
		ViewChanges:   fw.MapInt(d, "vc_count", 0),
		ForkCount:     fw.MapInt(d, "fork_count", 0),
		LastError:     fw.MapStr(d, "last_error", ""),
		HonestCommits: map[string]map[string]int{},
	}
	if nAny, ok := d["nodes"].([]any); ok {
		for i, x := range nAny {
			if m, ok := x.(map[string]any); ok {
				nd := &pbftNode{ID: i,
					Fault:       fw.MapStr(m, "fault", faultHonest),
					View:        uint64(fw.MapInt(m, "view", 0)),
					PrePrepares: map[string]string{},
					Prepares:    map[string][]string{},
					Commits:     map[string][]string{},
					Prepared:    map[string]bool{},
					Committed:   map[string]string{},
				}
				st.Nodes = append(st.Nodes, nd)
			}
		}
	}
	if len(st.Nodes) != st.N {
		return defaultSnapState()
	}
	if rsAny, ok := d["requests"].([]any); ok {
		for _, x := range rsAny {
			if m, ok := x.(map[string]any); ok {
				st.Requests = append(st.Requests, request{
					Seq:    uint64(fw.MapInt(m, "seq", 0)),
					Op:     fw.MapStr(m, "op", ""),
					Digest: fw.MapStr(m, "digest", ""),
				})
			}
		}
	}
	if hcAny, ok := d["honest_commits"].(map[string]any); ok {
		for k, vAny := range hcAny {
			if mm, ok := vAny.(map[string]any); ok {
				cnt := map[string]int{}
				for d, c := range mm {
					cnt[d] = intFromAny(c)
				}
				st.HonestCommits[k] = cnt
			}
		}
	}
	if fAny, ok := d["forks"].([]any); ok {
		for _, x := range fAny {
			if m, ok := x.(map[string]any); ok {
				ev := forkEvidence{
					View:    uint64(fw.MapInt(m, "view", 0)),
					Seq:     uint64(fw.MapInt(m, "seq", 0)),
					Digests: map[string][]int{},
				}
				if dAny, ok := m["digests"].(map[string]any); ok {
					for d, lAny := range dAny {
						if list, ok := lAny.([]any); ok {
							for _, idAny := range list {
								ev.Digests[d] = append(ev.Digests[d], intFromAny(idAny))
							}
						}
					}
				}
				st.Forks = append(st.Forks, ev)
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["n"] = st.N
	s.Data["f"] = st.F
	s.Data["tick"] = st.Tick
	s.Data["seed"] = st.Seed
	s.Data["view"] = int(st.View)
	s.Data["next_seq"] = int(st.NextSeq)
	s.Data["vc_count"] = st.ViewChanges
	s.Data["fork_count"] = st.ForkCount
	s.Data["last_error"] = st.LastError
	nAny := make([]any, len(st.Nodes))
	for i, nd := range st.Nodes {
		nAny[i] = map[string]any{"fault": nd.Fault, "view": int(nd.View)}
	}
	s.Data["nodes"] = nAny
	rAny := make([]any, len(st.Requests))
	for i, r := range st.Requests {
		rAny[i] = map[string]any{"seq": int(r.Seq), "op": r.Op, "digest": r.Digest}
	}
	s.Data["requests"] = rAny
	hcAny := map[string]any{}
	for k, m := range st.HonestCommits {
		mm := map[string]any{}
		for d, c := range m {
			mm[d] = c
		}
		hcAny[k] = mm
	}
	s.Data["honest_commits"] = hcAny
	fAny := make([]any, len(st.Forks))
	for i, ev := range st.Forks {
		dAny := map[string]any{}
		for d, ids := range ev.Digests {
			list := make([]any, len(ids))
			for j, id := range ids {
				list[j] = id
			}
			dAny[d] = list
		}
		fAny[i] = map[string]any{"view": int(ev.View), "seq": int(ev.Seq), "digests": dAny}
	}
	s.Data["forks"] = fAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "PBFT 拜占庭攻击",
		Description:         "演示 PBFT 三阶段共识 + 4 种拜占庭故障 + view-change + fork 检测",
		Category:            fw.CategoryAttackSecurity,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPbftAttack},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths:    []string{},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultStateFw,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultStateFw() fw.SceneState {
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: "ready", Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode: sceneCode, SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "configure", Label: "配置网络（N、节点故障）",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "节点数 (3f+1)", Required: true, Default: 4, Min: 4, Max: 13, Step: 3},
					{Name: "primary_fault", Type: fw.FieldEnum, Label: "primary 故障", Required: true, Default: faultHonest,
						Options: []any{faultHonest, faultEquivocating, faultSilent, faultLying}},
				},
			},
			{
				ActionCode: "set_node_fault", Label: "设置节点 fault",
				Category: fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldNumber, Label: "节点 ID", Required: true, Default: 1, Min: 0, Step: 1},
					{Name: "fault", Type: fw.FieldEnum, Label: "故障类型", Required: true, Default: faultEquivocating,
						Options: []any{faultHonest, faultEquivocating, faultSilent, faultLying}},
				},
				LinkOwnerFields: []string{"attack.pbft.byzantine_count"},
			},
			{
				ActionCode: "send_request", Label: "客户端请求",
				Description: "primary 启动一个 PBFT 实例（PRE-PREPARE）",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "op", Type: fw.FieldString, Label: "请求内容", Required: true, Default: "transfer 100"},
				},
				WritesOwnedFields: []string{},
			},
			{
				ActionCode: "process_messages", Label: "处理消息（推进共识）",
				Description: "所有节点处理 inbox 中的 PREPARE / COMMIT / VC",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{},
			},
			{
				ActionCode: "trigger_view_change", Label: "强制 view-change",
				Description: "演示 view-change",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "reason", Type: fw.FieldString, Label: "原因", Required: true, Default: "primary 超时"},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_enable_attack",
				Label:         "教师启用攻击演示",
				Description:   "仅教师可用，启用攻击演示用于教学展示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneFault,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师启用攻击演示"},
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
	st := loadState(state)
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "PBFT 4 节点初始化（f=1）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	return fw.StepOutput{Render: buildEnvelope(st, "tick", "", false)}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "configure":
		n := fw.MapInt(in.Params, "n", 4)
		pf := fw.MapStr(in.Params, "primary_fault", faultHonest)
		st = newSnapState(n, pf)
		saveState(state, st)
		out.Render = buildEnvelope(st, "configure",
			fmt.Sprintf("N=%d f=%d primary fault=%s", st.N, st.F, pf), true)
		return out, nil

	case "set_node_fault":
		id := fw.MapInt(in.Params, "node_id", 1)
		fault := fw.MapStr(in.Params, "fault", faultEquivocating)
		if err := st.configureFault(id, fault); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_node_fault",
			fmt.Sprintf("节点 %d → %s（拜占庭=%d, 阈值 f=%d）", id, fault, st.byzantineCount(), st.F), false)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "send_request":
		op := fw.MapStr(in.Params, "op", "transfer 100")
		r, err := st.startRequest(op)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "send_request",
			fmt.Sprintf("seq=%d, digest=%s", r.Seq, r.Digest), false)
		appendRequestMicroSteps(&out.Render, r.Seq)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "process_messages":
		st.processInbox()
		saveState(state, st)
		out.Render = buildEnvelope(st, "process_messages",
			fmt.Sprintf("tick=%d 推进；fork_count=%d", st.Tick, st.ForkCount), false)
		appendProcessMicroSteps(&out.Render, st.ForkCount > 0)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "trigger_view_change":
		reason := fw.MapStr(in.Params, "reason", "primary 超时")
		st.triggerViewChange(reason)
		saveState(state, st)
		out.Render = buildEnvelope(st, "trigger_view_change",
			fmt.Sprintf("view → %d，新 primary=%d (%s)", st.View, st.primary(), reason), false)
		appendViewChangeMicroSteps(&out.Render, st.primary())
		return out, nil

	case "teacher_enable_attack":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师启用攻击演示"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-attack-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		st = defaultSnapState()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}
	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode"}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 60)

	// 1) 节点环形布局
	prims = append(prims, fw.PrimRingLayout("node-ring", st.N))
	for _, nd := range st.Nodes {
		role := "node-honest"
		switch nd.Fault {
		case faultEquivocating:
			role = "node-equivocating"
		case faultSilent:
			role = "node-silent"
		case faultLying:
			role = "node-lying"
		}
		status := "active"
		if nd.ID == st.primary() {
			role = role + "-primary"
		}
		label := fmt.Sprintf("n%d\n%s", nd.ID, nd.Fault)
		if nd.ID == st.primary() {
			label = fmt.Sprintf("PRIMARY\nn%d\n%s", nd.ID, nd.Fault)
		}
		prims = append(prims, fw.PrimNode(fmt.Sprintf("node-%d", nd.ID), label, status, role))
	}

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-pbft",
		`\text{prepared}: |\text{PREPARE}| \geq 2f;\quad \text{committed}: |\text{COMMIT}| \geq 2f{+}1;\quad N=3f{+}1`, false))
	prims = append(prims, fw.PrimMathFormula("formula-safety",
		`\text{safety holds iff } |\text{byzantine}| \leq f`, false))

	// 3) 阶段流水线
	phaseIDs := []string{"ph-pp", "ph-prep", "ph-com", "ph-done"}
	prims = append(prims, fw.PrimStack("phase-stack", phaseIDs, "horizontal"))
	for i, p := range []string{"PRE-PREPARE", "PREPARE", "COMMIT", "COMMITTED"} {
		role := strings.ToLower(p)
		status := "normal"
		if len(st.Requests) > 0 {
			// 估算当前最高 seq 的进度
			lastSeq := st.NextSeq
			pri := st.primary()
			pp, _ := st.Nodes[pri].PrePrepares[keyVS(st.View, lastSeq)]
			seqK := keyVS(st.View, lastSeq)
			honestProgress := 0
			for _, nd := range st.Nodes {
				if nd.Fault != faultHonest {
					continue
				}
				if pp != "" && nd.PrePrepares[seqK] != "" {
					honestProgress = max(honestProgress, 1)
				}
				for k := range nd.Prepared {
					if strings.HasPrefix(k, seqK+"/") {
						honestProgress = max(honestProgress, 2)
					}
				}
				if _, ok := nd.Committed[seqK]; ok {
					honestProgress = max(honestProgress, 3)
				}
			}
			if i <= honestProgress {
				status = "active"
			}
		}
		prims = append(prims, fw.PrimNode(phaseIDs[i], p, status, role))
	}
	for i := 0; i < 3; i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ph-edge-%d", i), phaseIDs[i], phaseIDs[i+1], "solid", "flow"))
	}

	// 4) 投票矩阵：行 = 节点，列 = 阶段
	if len(st.Requests) > 0 {
		seqK := keyVS(st.View, st.NextSeq)
		cells := []map[string]any{}
		for ni, nd := range st.Nodes {
			cellsForNode := []string{
				ifThenStr(nd.PrePrepares[seqK] != "", "PP", ""),
				ifThenStr(hasPrepared(nd, seqK), "P", ""),
				ifThenStr(nd.Committed[seqK] != "", "C", ""),
			}
			for col, val := range cellsForNode {
				color := "muted"
				if val != "" {
					color = "success"
					if nd.Fault != faultHonest {
						color = "warning"
					}
				}
				cells = append(cells, map[string]any{"row": ni, "col": col, "value": val, "color_role": color})
			}
		}
		prims = append(prims, fw.PrimVoteMatrix("vote-matrix", st.N, 3, cells))
	}

	// 5) 状态参数
	bzCount := st.byzantineCount()
	safety := "✓"
	if bzCount > st.F {
		safety = "✗"
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("N=%d f=%d byzantine=%d safety=%s\nview=%d primary=n%d\nrequests=%d view_changes=%d fork_count=%d",
			st.N, st.F, bzCount, safety, st.View, st.primary(),
			len(st.Requests), st.ViewChanges, st.ForkCount),
		"text", nil, 6))

	// 6) 节点故障表
	nLines := []string{"id  fault          view  prePrepares  prepared  committed"}
	for _, nd := range st.Nodes {
		nLines = append(nLines, fmt.Sprintf("  n%d  %-13s  %-4d  %-11d  %-8d  %d",
			nd.ID, nd.Fault, nd.View, len(nd.PrePrepares), len(nd.Prepared), len(nd.Committed)))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-nodes", strings.Join(nLines, "\n"), "text", nil, 8))

	// 7) Requests 表
	if len(st.Requests) > 0 {
		rLines := []string{"seq  op                 digest"}
		for _, r := range st.Requests {
			rLines = append(rLines, fmt.Sprintf("  %-3d  %-18s %s", r.Seq, r.Op, r.Digest))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-requests", strings.Join(rLines, "\n"), "text", nil, 8))
	}

	// 8) 节点最近 log（取 primary + 1 个 honest backup 拼接）
	logLines := []string{}
	for _, nd := range st.Nodes {
		startIdx := 0
		if len(nd.Log) > 4 {
			startIdx = len(nd.Log) - 4
		}
		for _, e := range nd.Log[startIdx:] {
			logLines = append(logLines, fmt.Sprintf("n%d t=%d [%s] %s", nd.ID, e.Tick, e.Phase, e.Note))
		}
	}
	if len(logLines) > 14 {
		logLines = logLines[len(logLines)-14:]
	}
	if len(logLines) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-logs", strings.Join(logLines, "\n"), "text", nil, 16))
	}

	// 9) Fork evidence 表
	if len(st.Forks) > 0 {
		fLines := []string{"view  seq  分歧 digest（被 honest 节点 commit 的不同摘要）"}
		for _, ev := range st.Forks {
			parts := []string{}
			for d, ids := range ev.Digests {
				idsStr := []string{}
				for _, id := range ids {
					idsStr = append(idsStr, fmt.Sprintf("n%d", id))
				}
				parts = append(parts, fmt.Sprintf("%s={%s}", d, strings.Join(idsStr, ",")))
			}
			fLines = append(fLines, fmt.Sprintf("  %-4d  %-3d  %s", ev.View, ev.Seq, strings.Join(parts, "  |  ")))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-forks", strings.Join(fLines, "\n"), "text", nil, 10))
	}

	// 10) 进度条
	prims = append(prims, fw.PrimBar("bar-byz", float64(bzCount), float64(st.F), "warning", fmt.Sprintf("byzantine %d / f=%d", bzCount, st.F)))
	prims = append(prims, fw.PrimBar("bar-fork", float64(st.ForkCount), 0, "danger", "Fork Events"))
	prims = append(prims, fw.PrimBar("bar-vc", float64(st.ViewChanges), 0, "info", "View Changes"))

	// 11) 动效
	if st.ForkCount > 0 {
		prims = append(prims, fw.PrimShake("shake-fork", "bar-fork", 0.5, 800))
		prims = append(prims, fw.PrimPulse("pulse-fork", "cb-forks", "danger", 1500))
	}
	prims = append(prims, fw.PrimGlow("glow-primary", fmt.Sprintf("node-%d", st.primary()), "info", 0.8))

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pbft", linkGroupPbftAttack, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "PBFT 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func ifThenStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func hasPrepared(nd *pbftNode, seqK string) bool {
	for k := range nd.Prepared {
		if strings.HasPrefix(k, seqK+"/") {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"n":               st.N,
		"f":               st.F,
		"view":            st.View,
		"primary":         st.primary(),
		"byzantine_count": st.byzantineCount(),
		"requests":        len(st.Requests),
		"view_changes":    st.ViewChanges,
		"fork_count":      st.ForkCount,
		"tick":            st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendRequestMicroSteps(env *fw.RenderEnvelope, seq uint64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "r-1", Label: fmt.Sprintf("client → primary：op (seq=%d)", seq), DurationMs: 400, HighlightIDs: []string{"ph-pp", "cb-requests"}},
		{ID: "r-2", Label: "primary 计算 digest 并广播 PRE-PREPARE", DurationMs: 500, HighlightIDs: []string{"node-ring", "vote-matrix"}, FirePrimitives: []string{"glow-primary"}},
		{ID: "r-3", Label: "等节点处理（process_messages）", DurationMs: 400, HighlightIDs: []string{"phase-stack"}, IsLinkTrigger: true},
	}
}

func appendProcessMicroSteps(env *fw.RenderEnvelope, hasFork bool) {
	tail := "✓ 共识达成（无 fork）"
	if hasFork {
		tail = "⚠ 检测到 fork：byzantine > f → safety 破坏"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "p-1", Label: "节点处理 inbox 中的 PRE-PREPARE → 广播 PREPARE", DurationMs: 400, HighlightIDs: []string{"vote-matrix", "ph-prep"}},
		{ID: "p-2", Label: "≥ 2f PREPARE → prepared → 广播 COMMIT", DurationMs: 400, HighlightIDs: []string{"formula-pbft", "ph-com"}},
		{ID: "p-3", Label: "≥ 2f+1 COMMIT → committed-local", DurationMs: 500, HighlightIDs: []string{"ph-done"}, FirePrimitives: []string{"glow-primary"}},
		{ID: "p-4", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-fork", "cb-forks"}, FirePrimitives: []string{"shake-fork", "pulse-fork"}, IsLinkTrigger: true},
	}
}

func appendViewChangeMicroSteps(env *fw.RenderEnvelope, newPrimary int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "v-1", Label: "honest 节点广播 VIEW-CHANGE", DurationMs: 400, HighlightIDs: []string{"node-ring", "bar-vc"}},
		{ID: "v-2", Label: "≥ 2f+1 VC → 切换", DurationMs: 400, HighlightIDs: []string{"formula-pbft"}},
		{ID: "v-3", Label: fmt.Sprintf("新 primary = n%d", newPrimary), DurationMs: 500, HighlightIDs: []string{fmt.Sprintf("node-%d", newPrimary)}, FirePrimitives: []string{"glow-primary"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "pbft-byzantine-attack",
		SourceScene:    sceneCode,
		SourceAction:   "inject_byzantine",
		LinkGroup:      linkGroupPbftAttack,
		ChangedFields:  []string{"attack.pbft.byzantine_count", "attack.pbft.view"},
		Payload:        map[string]any{"view": st.View, "byzantine_count": st.byzantineCount()},
		SourceAnchorID: "pbft-byz-anchor",
		TargetAnchorID: "pbft-consensus-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"attack": map[string]any{
			"pbft": map[string]any{
				"n":               st.N,
				"f":               st.F,
				"byzantine_count": st.byzantineCount(),
				"view":            int(st.View),
				"primary":         st.primary(),
				"view_changes":    st.ViewChanges,
				"fork_count":      st.ForkCount,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}
