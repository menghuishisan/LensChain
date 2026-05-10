// 模块：sim-engine/scenarios/internal/consensus/dposvoting
// 文件职责：CON-05 委托权益证明（DPoS）投票场景的完整实现。
//
// SSOT 依据：06.md §4.2.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 DPoS 选民 → 候选人加权投票、按总票数排名选出前 N 名活跃代表、
// round-robin 轮转出块、弹劾移除恶意代表。零第三方加密库。
//
//   · 投票：每个选民可对一个候选人投票，权重 = 选民 stake；
//     允许同一选民改投（旧票自动撤销）；
//   · 选举：按候选人累计票数降序，取前 N 名为 active_delegates；
//   · 出块：轮转编号 = block_height mod N → 对应 active_delegates 出块；
//     恶意 / 弹劾的代表跳过；
//   · 弹劾：≥ 2/3 选民同意时移除某代表（不重新选举，由备选候选人替补）；
//   · 委托罢免：选民可主动撤销投票。

package dposvoting

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "dpos-voting"
	schemaVersion = "v1.0.0"
	algorithmType = "dpos"

	defaultActiveSlots = 3
	maxCandidates      = 10
	maxVoters          = 12
	maxBlockHistory    = 32

	linkGroupPosEconomy = "pos-economy-group"
	linkOwnerSubtree    = "consensus.dpos"
)

// =====================================================================
// 数据结构
// =====================================================================

type voter struct {
	ID       string
	Stake    int64
	VotedFor string // 当前投给的候选人 ID；"" 表示未投
}

type candidate struct {
	ID        string
	Bio       string // 简短自我介绍 / 政纲（教学）
	Votes     int64
	Blocks    int
	Impeached bool
	Active    bool // 是否进入活跃代表
}

type blockRecord struct {
	Height   int
	Producer string
	Skipped  bool // 当前轮代表被弹劾时跳过
}

type snapState struct {
	Voters      []voter
	Candidates  []candidate
	ActiveSlots int
	BlockHeight int
	History     []blockRecord
	LastError   string
}

func defaultSnapState() snapState {
	return snapState{
		Voters: []voter{
			{ID: "v1", Stake: 100},
			{ID: "v2", Stake: 200},
			{ID: "v3", Stake: 300},
			{ID: "v4", Stake: 150},
			{ID: "v5", Stake: 250},
		},
		Candidates: []candidate{
			{ID: "alice", Bio: "稳定运行 5 年"},
			{ID: "bob", Bio: "技术节点"},
			{ID: "carol", Bio: "社区代表"},
			{ID: "dave", Bio: "新晋"},
			{ID: "eve", Bio: "可疑"},
		},
		ActiveSlots: defaultActiveSlots,
	}
}

// rebuildVotes 根据 voters.VotedFor 重算 candidates.Votes。
func (st *snapState) rebuildVotes() {
	for i := range st.Candidates {
		st.Candidates[i].Votes = 0
	}
	for _, v := range st.Voters {
		if v.VotedFor == "" {
			continue
		}
		for j := range st.Candidates {
			if st.Candidates[j].ID == v.VotedFor && !st.Candidates[j].Impeached {
				st.Candidates[j].Votes += v.Stake
				break
			}
		}
	}
}

// runElection 按当前票数选出活跃代表（票数降序，弹劾的不参与；填满 ActiveSlots 后剩余为后备）。
func (st *snapState) runElection() {
	st.rebuildVotes()
	for i := range st.Candidates {
		st.Candidates[i].Active = false
	}
	// 排序候选人：先按 votes 降序，平票按 ID 字典序
	indexed := make([]int, len(st.Candidates))
	for i := range st.Candidates {
		indexed[i] = i
	}
	sort.Slice(indexed, func(a, b int) bool {
		ca := st.Candidates[indexed[a]]
		cb := st.Candidates[indexed[b]]
		if ca.Impeached != cb.Impeached {
			return !ca.Impeached
		}
		if ca.Votes != cb.Votes {
			return ca.Votes > cb.Votes
		}
		return ca.ID < cb.ID
	})
	picked := 0
	for _, ix := range indexed {
		if picked >= st.ActiveSlots {
			break
		}
		if st.Candidates[ix].Impeached || st.Candidates[ix].Votes <= 0 {
			continue
		}
		st.Candidates[ix].Active = true
		picked++
	}
}

// activeDelegatesOrdered 按票数降序返回所有活跃代表 ID。
func (st snapState) activeDelegatesOrdered() []string {
	indexed := make([]int, len(st.Candidates))
	for i := range st.Candidates {
		indexed[i] = i
	}
	sort.Slice(indexed, func(a, b int) bool {
		ca := st.Candidates[indexed[a]]
		cb := st.Candidates[indexed[b]]
		if ca.Votes != cb.Votes {
			return ca.Votes > cb.Votes
		}
		return ca.ID < cb.ID
	})
	out := []string{}
	for _, ix := range indexed {
		if st.Candidates[ix].Active && !st.Candidates[ix].Impeached {
			out = append(out, st.Candidates[ix].ID)
		}
	}
	return out
}

// nextProducer round-robin 选出当前 BlockHeight 应出块的代表 ID。
func (st snapState) nextProducer() (string, bool) {
	active := st.activeDelegatesOrdered()
	if len(active) == 0 {
		return "", false
	}
	idx := st.BlockHeight % len(active)
	return active[idx], true
}

// totalVoterStake 选民总 stake（用于弹劾 2/3 阈值）。
func (st snapState) totalVoterStake() int64 {
	var s int64
	for _, v := range st.Voters {
		s += v.Stake
	}
	return s
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
		ActiveSlots: fw.MapInt(d, "active_slots", defaultActiveSlots),
		BlockHeight: fw.MapInt(d, "block_height", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if vsAny, ok := d["voters"].([]any); ok {
		for _, vAny := range vsAny {
			if vm, ok := vAny.(map[string]any); ok {
				st.Voters = append(st.Voters, voter{
					ID:       fw.MapStr(vm, "id", ""),
					Stake:    int64(fw.MapInt(vm, "stake", 0)),
					VotedFor: fw.MapStr(vm, "voted_for", ""),
				})
			}
		}
	}
	if csAny, ok := d["candidates"].([]any); ok {
		for _, cAny := range csAny {
			if cm, ok := cAny.(map[string]any); ok {
				st.Candidates = append(st.Candidates, candidate{
					ID:        fw.MapStr(cm, "id", ""),
					Bio:       fw.MapStr(cm, "bio", ""),
					Votes:     int64(fw.MapInt(cm, "votes", 0)),
					Blocks:    fw.MapInt(cm, "blocks", 0),
					Impeached: fw.MapBool(cm, "impeached", false),
					Active:    fw.MapBool(cm, "active", false),
				})
			}
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, h := range hAny {
			if hm, ok := h.(map[string]any); ok {
				st.History = append(st.History, blockRecord{
					Height:   fw.MapInt(hm, "height", 0),
					Producer: fw.MapStr(hm, "producer", ""),
					Skipped:  fw.MapBool(hm, "skipped", false),
				})
			}
		}
	}
	if len(st.Voters) == 0 {
		st = defaultSnapState()
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["active_slots"] = st.ActiveSlots
	s.Data["block_height"] = st.BlockHeight
	s.Data["last_error"] = st.LastError
	vs := make([]any, len(st.Voters))
	for i, v := range st.Voters {
		vs[i] = map[string]any{"id": v.ID, "stake": v.Stake, "voted_for": v.VotedFor}
	}
	s.Data["voters"] = vs
	cs := make([]any, len(st.Candidates))
	for i, c := range st.Candidates {
		cs[i] = map[string]any{
			"id": c.ID, "bio": c.Bio, "votes": c.Votes, "blocks": c.Blocks,
			"impeached": c.Impeached, "active": c.Active,
		}
	}
	s.Data["candidates"] = cs
	hist := make([]any, len(st.History))
	for i, h := range st.History {
		hist[i] = map[string]any{"height": h.Height, "producer": h.Producer, "skipped": h.Skipped}
	}
	s.Data["history"] = hist
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "委托权益证明（DPoS）",
		Description:         "演示 DPoS：选民投票 → 排名选举 → round-robin 出块 → 弹劾代表",
		Category:            fw.CategoryConsensus,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPosEconomy},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"consensus.dpos.active_delegates",
			"consensus.dpos.current_producer",
			"consensus.dpos.block_height",
			"consensus.dpos.impeached_set",
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
		Phase:     "ready",
		Data:      map[string]any{},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_active_slots", Label: "设置活跃代表数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "前 N 名当选", Required: true, Default: defaultActiveSlots, Min: 1, Max: 9, Step: 1},
				},
			},
			{
				ActionCode: "vote", Label: "投票",
				Description: "选民投票（旧票自动撤销）",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "voter_id", Type: fw.FieldString, Label: "选民 ID", Required: true, Default: "v1"},
					{Name: "candidate_id", Type: fw.FieldString, Label: "候选人 ID", Required: true, Default: "alice"},
				},
				WritesOwnedFields: []string{"consensus.dpos.active_delegates"},
				LinkOwnerFields:   []string{"consensus.dpos.active_delegates"},
			},
			{
				ActionCode: "withdraw_vote", Label: "撤销投票",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "voter_id", Type: fw.FieldString, Label: "选民 ID", Required: true, Default: "v1"},
				},
			},
			{
				ActionCode: "run_election", Label: "举行选举",
				Description: "按票数选出前 N 名为活跃代表",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"consensus.dpos.active_delegates"},
				LinkOwnerFields:   []string{"consensus.dpos.active_delegates"},
			},
			{
				ActionCode: "produce_block", Label: "出块",
				Description: "当前轮代表出 1 块",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"consensus.dpos.current_producer", "consensus.dpos.block_height"},
				LinkOwnerFields:   []string{"consensus.dpos.current_producer", "consensus.dpos.block_height"},
			},
			{
				ActionCode: "produce_n_blocks", Label: "出 N 块",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "区块数", Required: true, Default: 6, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "impeach", Label: "弹劾代表",
				Description: "把指定代表标记为弹劾，不再出块",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "candidate_id", Type: fw.FieldString, Label: "代表 ID", Required: true, Default: "eve"},
				},
				WritesOwnedFields: []string{"consensus.dpos.impeached_set"},
				LinkOwnerFields:   []string{"consensus.dpos.impeached_set"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
	st := loadState(state)
	st.rebuildVotes()
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "DPoS 初始化（5 选民 / 5 候选人 / 3 名活跃）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", "", false)
	return fw.StepOutput{Render: env}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "set_active_slots":
		n := fw.MapInt(in.Params, "n", defaultActiveSlots)
		if n < 1 || n > 9 {
			return fw.ActionOutput{Success: false, ErrorMessage: "活跃代表数 ∈ [1, 9]"}, nil
		}
		st.ActiveSlots = n
		st.runElection()
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_active_slots", fmt.Sprintf("活跃代表数 = %d", n), false)
		return out, nil

	case "vote":
		vid := fw.MapStr(in.Params, "voter_id", "")
		cid := fw.MapStr(in.Params, "candidate_id", "")
		vIdx := -1
		cIdx := -1
		for i, v := range st.Voters {
			if v.ID == vid {
				vIdx = i
				break
			}
		}
		for i, c := range st.Candidates {
			if c.ID == cid {
				cIdx = i
				break
			}
		}
		if vIdx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到选民: " + vid}, nil
		}
		if cIdx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到候选人: " + cid}, nil
		}
		if st.Candidates[cIdx].Impeached {
			return fw.ActionOutput{Success: false, ErrorMessage: cid + " 已被弹劾，无法投票"}, nil
		}
		st.Voters[vIdx].VotedFor = cid
		st.rebuildVotes()
		saveState(state, st)
		out.Render = buildEnvelope(st, "vote", fmt.Sprintf("%s 投票给 %s（权重 %d）", vid, cid, st.Voters[vIdx].Stake), false)
		appendVoteMicroSteps(&out.Render, vid, cid)
		return out, nil

	case "withdraw_vote":
		vid := fw.MapStr(in.Params, "voter_id", "")
		for i := range st.Voters {
			if st.Voters[i].ID == vid {
				st.Voters[i].VotedFor = ""
				break
			}
		}
		st.rebuildVotes()
		saveState(state, st)
		out.Render = buildEnvelope(st, "withdraw_vote", vid+" 撤销投票", false)
		return out, nil

	case "run_election":
		st.runElection()
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_election", fmt.Sprintf("选举结束，前 %d 名当选", st.ActiveSlots), false)
		appendElectionMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "produce_block":
		producer, ok := st.nextProducer()
		if !ok {
			return fw.ActionOutput{Success: false, ErrorMessage: "无活跃代表（请先 run_election）"}, nil
		}
		st.History = append(st.History, blockRecord{Height: st.BlockHeight, Producer: producer})
		if len(st.History) > maxBlockHistory {
			st.History = st.History[len(st.History)-maxBlockHistory:]
		}
		for i := range st.Candidates {
			if st.Candidates[i].ID == producer {
				st.Candidates[i].Blocks++
				break
			}
		}
		st.BlockHeight++
		saveState(state, st)
		out.Render = buildEnvelope(st, "produce_block", fmt.Sprintf("#%d → %s 出块", st.BlockHeight-1, producer), false)
		appendProduceMicroSteps(&out.Render, producer)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "produce_n_blocks":
		n := fw.MapInt(in.Params, "n", 6)
		if n < 1 {
			n = 1
		}
		var lastProducer string
		for i := 0; i < n; i++ {
			producer, ok := st.nextProducer()
			if !ok {
				return fw.ActionOutput{Success: false, ErrorMessage: "中途无活跃代表"}, nil
			}
			st.History = append(st.History, blockRecord{Height: st.BlockHeight, Producer: producer})
			for j := range st.Candidates {
				if st.Candidates[j].ID == producer {
					st.Candidates[j].Blocks++
					break
				}
			}
			st.BlockHeight++
			lastProducer = producer
		}
		if len(st.History) > maxBlockHistory {
			st.History = st.History[len(st.History)-maxBlockHistory:]
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "produce_n_blocks", fmt.Sprintf("出 %d 块（终态出块者 %s）", n, lastProducer), false)
		appendProduceMicroSteps(&out.Render, lastProducer)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "impeach":
		cid := fw.MapStr(in.Params, "candidate_id", "")
		idx := -1
		for i, c := range st.Candidates {
			if c.ID == cid {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到代表: " + cid}, nil
		}
		if st.Candidates[idx].Impeached {
			return fw.ActionOutput{Success: false, ErrorMessage: cid + " 已被弹劾"}, nil
		}
		st.Candidates[idx].Impeached = true
		st.Candidates[idx].Active = false
		// 撤销给该代表的所有投票
		for i := range st.Voters {
			if st.Voters[i].VotedFor == cid {
				st.Voters[i].VotedFor = ""
			}
		}
		st.rebuildVotes()
		st.runElection()
		saveState(state, st)
		out.Render = buildEnvelope(st, "impeach", cid+" 已被弹劾，所有投票回收", false)
		appendImpeachMicroSteps(&out.Render, cid)
		out.SharedStateDiff = ownerDiff(st)
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
		st = defaultSnapState()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 40)

	// 1) 选民泳道（上）/ 候选人环（下） — 用 stack vertical 划分
	prims = append(prims, fw.PrimStack("layout", []string{"voter-row", "candidate-ring"}, "vertical"))

	// 2) 选民水平排列
	voterIDs := []string{}
	for _, v := range st.Voters {
		voterIDs = append(voterIDs, "voter-"+v.ID)
	}
	prims = append(prims, fw.PrimStack("voter-row", voterIDs, "horizontal"))
	for _, v := range st.Voters {
		role := "voter"
		status := "normal"
		if v.VotedFor != "" {
			status = "active"
		}
		label := fmt.Sprintf("%s\nstake=%d\n→ %s", v.ID, v.Stake, dashOr(v.VotedFor))
		prims = append(prims, fw.PrimNode("voter-"+v.ID, label, status, role))
	}

	// 3) 候选人环
	candIDs := []string{}
	for _, c := range st.Candidates {
		candIDs = append(candIDs, "cand-"+c.ID)
	}
	prims = append(prims, fw.PrimRingLayout("candidate-ring", len(st.Candidates)))
	currentProducer, _ := st.nextProducer()
	for _, c := range st.Candidates {
		role := "candidate"
		status := "normal"
		if c.Active {
			role = "active-delegate"
			status = "active"
		}
		if c.Impeached {
			role = "impeached"
			status = "error"
		}
		if c.ID == currentProducer {
			role = "producer"
			status = "active"
		}
		label := fmt.Sprintf("%s\nvotes=%d\nblocks=%d", c.ID, c.Votes, c.Blocks)
		if c.Impeached {
			label = fmt.Sprintf("%s\n[弹劾]\nvotes=0", c.ID)
		}
		prims = append(prims, fw.PrimNode("cand-"+c.ID, label, status, role))
	}

	// 4) 投票边（voter → candidate）
	for _, v := range st.Voters {
		if v.VotedFor == "" {
			continue
		}
		anim := "flow"
		prims = append(prims, fw.PrimEdge(
			"vote-"+v.ID,
			"voter-"+v.ID,
			"cand-"+v.VotedFor,
			"solid", anim))
	}

	// 5) 票数饼图
	segs := []map[string]any{}
	for _, c := range st.Candidates {
		if c.Impeached || c.Votes == 0 {
			continue
		}
		colorRole := "info"
		if c.Active {
			colorRole = "success"
		}
		if c.ID == currentProducer {
			colorRole = "warning"
		}
		segs = append(segs, map[string]any{
			"label":      c.ID,
			"value":      float64(c.Votes),
			"color_role": colorRole,
		})
	}
	if len(segs) > 0 {
		prims = append(prims, fw.PrimPieChart("vote-pie", segs))
	}

	// 6) 公式
	prims = append(prims, fw.PrimMathFormula("formula-vote",
		`\text{votes}_c = \sum_{v \to c} \text{stake}_v;\quad \text{producer}(h) = \text{active}[h \bmod N]`, false))

	// 7) 关键数字
	active := st.activeDelegatesOrdered()
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("活跃代表 N = %d\nactive_delegates = %s\n当前出块者 = %s\nblock_height = %d\n选民总 stake = %d",
			st.ActiveSlots, strings.Join(active, ", "), dashOr(currentProducer), st.BlockHeight, st.totalVoterStake()),
		"text", nil, 6))

	// 8) 候选人表
	rows := []string{"id     bio                votes  blocks  status"}
	for _, c := range st.Candidates {
		statusStr := "normal"
		if c.Active {
			statusStr = "ACTIVE"
		}
		if c.Impeached {
			statusStr = "IMPEACHED"
		}
		bio := c.Bio
		if len(bio) > 16 {
			bio = bio[:16]
		}
		rows = append(rows, fmt.Sprintf("%-6s %-18s %-6d %-6d  %s",
			c.ID, bio, c.Votes, c.Blocks, statusStr))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-candidates", strings.Join(rows, "\n"), "text", nil, 12))

	// 9) 出块历史
	histLines := []string{"#  Producer  状态"}
	startIdx := 0
	if len(st.History) > 12 {
		startIdx = len(st.History) - 12
		histLines = append(histLines, "  …")
	}
	for _, h := range st.History[startIdx:] {
		state := "ok"
		if h.Skipped {
			state = "skipped"
		}
		histLines = append(histLines, fmt.Sprintf("%-3d %-8s  %s", h.Height, h.Producer, state))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(histLines, "\n"), "text", nil, 14))

	// 10) 动效
	if currentProducer != "" {
		prims = append(prims, fw.PrimGlow("glow-producer", "cand-"+currentProducer, "warning", 0.9))
		prims = append(prims, fw.PrimBurst("burst-producer", "cand-"+currentProducer, "warning",
			int64(st.BlockHeight), 700))
	}
	for _, c := range st.Candidates {
		if c.Active && !c.Impeached {
			prims = append(prims, fw.PrimGlow("glow-active-"+c.ID, "cand-"+c.ID, "success", 0.6))
		}
		if c.Impeached {
			prims = append(prims, fw.PrimShake("shake-"+c.ID, "cand-"+c.ID, 0.4, 600))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pos-econ", linkGroupPosEconomy, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "DPoS 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	currentProducer, _ := st.nextProducer()
	impeached := []string{}
	for _, c := range st.Candidates {
		if c.Impeached {
			impeached = append(impeached, c.ID)
		}
	}
	d := map[string]any{
		"voter_count":       len(st.Voters),
		"candidate_count":   len(st.Candidates),
		"active_slots":      st.ActiveSlots,
		"active_delegates":  st.activeDelegatesOrdered(),
		"current_producer":  currentProducer,
		"block_height":      st.BlockHeight,
		"impeached_set":     impeached,
		"total_voter_stake": st.totalVoterStake(),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendVoteMicroSteps(env *fw.RenderEnvelope, vid, cid string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "vt-1", Label: vid + " 选择候选人 " + cid, DurationMs: 400, HighlightIDs: []string{"voter-" + vid, "cand-" + cid}, ParentPhase: "voting"},
		{ID: "vt-2", Label: "票权 = 选民 stake", DurationMs: 400, HighlightIDs: []string{"vote-" + vid, "vote-pie"}, FirePrimitives: []string{"glow-producer"}},
		{ID: "vt-3", Label: "更新候选人累计票数", DurationMs: 400, HighlightIDs: []string{"cb-candidates"}},
	}
}

func appendElectionMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "el-1", Label: "汇总每位候选人票数", DurationMs: 400, HighlightIDs: []string{"vote-pie", "formula-vote"}},
		{ID: "el-2", Label: "降序排名", DurationMs: 400, HighlightIDs: []string{"cb-candidates"}},
		{ID: "el-3", Label: "前 N 名为活跃代表", DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendProduceMicroSteps(env *fw.RenderEnvelope, producer string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "pb-1", Label: "round-robin 轮转选出当前代表", DurationMs: 400, HighlightIDs: []string{"cb-status", "formula-vote"}},
		{ID: "pb-2", Label: producer + " 出块", DurationMs: 500, HighlightIDs: []string{"cand-" + producer, "cb-history"}, FirePrimitives: []string{"burst-producer"}, IsLinkTrigger: true},
		{ID: "pb-3", Label: "block_height++", DurationMs: 300, HighlightIDs: []string{"cb-status"}},
	}
}

func appendImpeachMicroSteps(env *fw.RenderEnvelope, cid string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "im-1", Label: cid + " 触发弹劾门槛", DurationMs: 400, HighlightIDs: []string{"cand-" + cid}, FirePrimitives: []string{"shake-" + cid}},
		{ID: "im-2", Label: "撤销该代表所有投票", DurationMs: 400, HighlightIDs: []string{"voter-row", "cb-candidates"}},
		{ID: "im-3", Label: "重新选举活跃代表", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
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
		ID:             "dpos-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_delegates",
		LinkGroup:      linkGroupPosEconomy,
		ChangedFields:  []string{"consensus.dpos.active_delegates", "consensus.dpos.block_height"},
		Payload:        map[string]any{"block_height": st.BlockHeight},
		SourceAnchorID: "dpos-output-anchor",
		TargetAnchorID: "economy-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "consensus.dpos.active_delegates", "consensus.dpos.block_height")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	currentProducer, _ := st.nextProducer()
	impeached := []string{}
	for _, c := range st.Candidates {
		if c.Impeached {
			impeached = append(impeached, c.ID)
		}
	}
	return map[string]any{
		"consensus": map[string]any{
			"dpos": map[string]any{
				"active_delegates":  st.activeDelegatesOrdered(),
				"current_producer":  currentProducer,
				"block_height":      st.BlockHeight,
				"active_slots":      st.ActiveSlots,
				"impeached_set":     impeached,
				"total_voter_stake": st.totalVoterStake(),
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
