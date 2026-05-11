// 模块：sim-engine/scenarios/internal/attacksecurity/selfishmining
// 文件职责：ATK-06 自私挖矿场景的完整实现（Eyal & Sirer 2014）。
//
// SSOT 依据：06.md §4.7.6 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Selfish Mining 决策机（零外部依赖）。
//
//   1. 模型：
//      · α = attacker hashpower / total
//      · γ = 网络分裂参数（攻击者私链与诚实链同时发布时，γ 比例的 honest 跟随 attacker）
//      · 块由 (Bernoulli α) 决定 attacker / honest 出块
//
//   2. 状态：
//      · lead = privateChainLen - publicChainLen （attacker 领先块数）
//      · 状态空间：lead ∈ {-1, 0, 0', 1, 2, ≥3}，0' 是一个特殊"分叉同高"状态
//
//   3. 攻击者决策（Eyal & Sirer 算法）：
//      事件 = attackerMines / honestMines；当前 lead 决定决策：
//
//      A. attackerMines（attacker 出块）：
//         · lead = 0   →  lead = 1，私藏（不广播）
//         · lead = 0'  →  发布 lead+1 块，状态归 0；attacker 拿到 2 块奖励
//         · lead ≥ 1   →  lead++ ，私藏
//
//      B. honestMines（honest 出块）：
//         · lead = 0   →  lead = 0（honest 块进主链；标准追赶）
//         · lead = 1   →  状态变 0'（attacker 立即发布唯一私藏块，与 honest 块同高竞争）
//         · lead = 2   →  发布 2 块 → 主链替换；attacker 拿到 2 块奖励，状态 0
//         · lead ≥ 3   →  发布 1 块（让对方追赶不上），lead--；保留剩余领先优势
//
//      C. 0' 同高度竞争（race）后的下一块：
//         · attackerMines      → attacker 链获胜（再加 1 块），attacker 收 3 奖励，状态 0
//         · honestMines on attacker fork (γ)   → attacker 链获胜，收 2 奖励，状态 0
//         · honestMines on honest fork (1−γ)   → honest 链获胜，收 1 奖励，状态 0
//
//   4. 收益分析：
//      · attackerReward / honestReward / attackerOrphaned / honestOrphaned
//      · attackerRevenueShare = attackerReward / (attackerReward + honestReward)
//      · 理论值（Eyal & Sirer）：
//          R(α, γ) = ((1 − γ) α(1 − α)² − α³) / (1 − α(1 + (2 − γ)α))
//      · 当 R(α, γ) > α 时，selfish mining 比诚实更有利

package selfishmining

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
	sceneCode     = "selfish-mining"
	schemaVersion = "v1.0.0"
	algorithmType = "selfish-mining"

	stateLeadM1  = "-1"
	stateLead0   = "0"
	stateLead0P  = "0'" // 同高度分叉
	stateLead1   = "1"
	stateLead2   = "2"
	stateLeadGE3 = "≥3"

	linkGroupPoWAttack = "pow-attack-group"
	linkOwnerSubtree   = "attack.selfish_mining"
)

// =====================================================================
// 数据结构
// =====================================================================

type miningEvent struct {
	Tick     int
	Roll     float64
	WhoMines string // attacker / honest
	State    string // 处理后的状态
	Action   string // 处理后的动作描述
	PubLen   int
	PriLen   int
	AtkRew   int
	HonRew   int
}

type chainBlock struct {
	Height int
	Owner  string // attacker / honest
	Hash   string
	Tick   int
}

type snapState struct {
	Alpha       float64 // [0, 1]
	Gamma       float64 // [0, 1]
	Tick        int
	Seed        string
	State       string
	Public      []chainBlock
	Private     []chainBlock
	AtkReward   int
	HonReward   int
	AtkOrphaned int // 攻击者块作废（譬如 0' race 输了）
	HonOrphaned int
	Events      []miningEvent
	LastError   string
}

func defaultSnapState() snapState {
	st := snapState{
		Alpha: 0.33, Gamma: 0.5, Seed: "lenschain-selfish",
		State: stateLead0,
	}
	g := chainBlock{Height: 0, Owner: "honest", Hash: "GENESIS", Tick: 0}
	st.Public = []chainBlock{g}
	st.Private = []chainBlock{g}
	return st
}

func (st snapState) leadDelta() int {
	return len(st.Private) - len(st.Public)
}

// =====================================================================
// 核心：rng + hash
// =====================================================================

func (st snapState) rng(salt string, tick int) float64 {
	buf := []byte(st.Seed)
	buf = append(buf, []byte(salt)...)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(tick))
	buf = append(buf, b[:]...)
	d := keccak256hash.Sum256(buf)
	v := binary.BigEndian.Uint64(d[:8])
	return float64(v%1_000_000_000) / 1_000_000_000.0
}

func (st snapState) blockHash(b chainBlock) string {
	buf := []byte(st.Seed)
	var num [8]byte
	binary.BigEndian.PutUint64(num[:], uint64(b.Height))
	buf = append(buf, num[:]...)
	buf = append(buf, []byte(b.Owner)...)
	binary.BigEndian.PutUint64(num[:], uint64(b.Tick))
	buf = append(buf, num[:]...)
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:6])
}

// =====================================================================
// 决策机
// =====================================================================

// step 一次出块 + Eyal-Sirer 决策。
func (st *snapState) step() miningEvent {
	st.Tick++
	roll := st.rng("miner", st.Tick)
	who := "honest"
	if roll < st.Alpha {
		who = "attacker"
	}
	ev := miningEvent{Tick: st.Tick, Roll: roll, WhoMines: who}
	if who == "attacker" {
		st.onAttackerMines(&ev)
	} else {
		st.onHonestMines(&ev)
	}
	ev.State = st.State
	ev.PubLen = len(st.Public) - 1
	ev.PriLen = len(st.Private) - 1
	ev.AtkRew = st.AtkReward
	ev.HonRew = st.HonReward
	st.Events = append(st.Events, ev)
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
	return ev
}

// newAttackerBlock 在 private chain 末端追加 attacker 块。
func (st *snapState) newAttackerBlock() chainBlock {
	parent := st.Private[len(st.Private)-1]
	b := chainBlock{Height: parent.Height + 1, Owner: "attacker", Tick: st.Tick}
	b.Hash = st.blockHash(b)
	st.Private = append(st.Private, b)
	return b
}

// newHonestBlock 在 public chain 末端追加 honest 块。
func (st *snapState) newHonestBlock() chainBlock {
	parent := st.Public[len(st.Public)-1]
	b := chainBlock{Height: parent.Height + 1, Owner: "honest", Tick: st.Tick}
	b.Hash = st.blockHash(b)
	st.Public = append(st.Public, b)
	return b
}

// publishPrivate 把 privateChain 的领先部分公开（替换 publicChain 的相应高度），
// 并将原 publicChain 的相同高度块作废。返回 attacker 收到的奖励数与 honest orphaned 数。
func (st *snapState) publishPrivate() (int, int) {
	priLen := len(st.Private)
	pubLen := len(st.Public)
	if priLen <= pubLen {
		return 0, 0
	}
	// 找到分叉点（第一个不同高度的 owner+hash）：教学版假设 private 与 public 都从 genesis 出发，
	// 私链是攻击者从 lead=0 开始独自挖出的；分叉点 = min(pubLen, priLen) 处。
	forkIdx := pubLen // 高度从 forkIdx 开始替换
	// 公链 fork 后的块都被作废
	orphaned := 0
	for i := forkIdx; i < pubLen; i++ {
		if st.Public[i].Owner == "honest" {
			orphaned++
			st.HonOrphaned++
		}
	}
	// 把私链 forkIdx 起的块替换进公链
	st.Public = append(st.Public[:forkIdx], st.Private[forkIdx:]...)
	// 攻击者奖励
	atkRewards := 0
	for i := forkIdx; i < len(st.Public); i++ {
		if st.Public[i].Owner == "attacker" {
			atkRewards++
		}
	}
	return atkRewards, orphaned
}

// onAttackerMines 攻击者出块。
func (st *snapState) onAttackerMines(ev *miningEvent) {
	st.newAttackerBlock()
	switch st.State {
	case stateLead0:
		st.State = stateLead1
		ev.Action = "attacker 出块 → 私藏 (lead=1)"
	case stateLead0P:
		// 0' 状态下出块 → 攻击链确定胜出
		atkR, _ := st.publishPrivate()
		st.AtkReward += atkR
		st.State = stateLead0
		ev.Action = fmt.Sprintf("0' 状态下出块 → 发布 → +%d 给 attacker", atkR)
	case stateLead1:
		st.State = stateLead2
		ev.Action = "attacker 私藏 → lead=2"
	case stateLead2:
		st.State = stateLeadGE3
		ev.Action = "attacker 私藏 → lead=3"
	case stateLeadGE3:
		ev.Action = "attacker 私藏 → lead 继续累加"
	}
}

// onHonestMines honest 出块（含 0' race）。
func (st *snapState) onHonestMines(ev *miningEvent) {
	switch st.State {
	case stateLead0:
		st.newHonestBlock()
		// 私链此时也应在末端追加（教学：保持 private == public）
		st.syncPrivateToPublic()
		ev.Action = "honest 出块 → 主链增长 (lead=0)"

	case stateLead0P:
		// 0' state → honest 出第二块；按 γ 决定 honest 跟谁
		// 先生成 honest 块
		hb := st.newHonestBlock()
		_ = hb
		// γ 概率：honest 跟随 attacker fork（attacker fork 胜出）
		choose := st.rng("gamma", st.Tick)
		if choose < st.Gamma {
			// attacker 胜：把 attacker 私链发布替换 honest 块
			// 撤销刚才的 honest 块
			lastH := st.Public[len(st.Public)-1]
			st.Public = st.Public[:len(st.Public)-1]
			st.HonOrphaned++
			_ = lastH
			atkR, _ := st.publishPrivate()
			st.AtkReward += atkR
			st.HonReward += 1 // 跟随的 honest 块也有奖励（本就在 attacker fork 上下一块）
			ev.Action = fmt.Sprintf("0' race + honest mines → γ=%.2f attacker fork wins (+%d to atk; +1 honest follower)",
				st.Gamma, atkR)
			st.State = stateLead0
		} else {
			// honest 胜：放弃 attacker 私链
			// attacker 唯一私藏块作废（lead=1 的那块）
			diff := len(st.Private) - len(st.Public)
			if diff > 0 {
				st.AtkOrphaned += diff
			}
			st.Private = append([]chainBlock{}, st.Public...)
			st.HonReward += 2 // honest 之前的块 + 当前块
			ev.Action = fmt.Sprintf("0' race + honest mines → 1−γ=%.2f honest fork wins (+2 honest)",
				1-st.Gamma)
			st.State = stateLead0
		}

	case stateLead1:
		// lead=1 时 honest 出块 → 进入 0'（attacker 立即广播私藏 1 块，触发竞争）
		st.newHonestBlock()
		ev.Action = "honest 出块 → 0' 同高竞争状态（attacker 释放第 1 块）"
		st.State = stateLead0P

	case stateLead2:
		// lead=2 honest 出块 → attacker 发布全部私链，主链替换；attacker 收 2，honest orphaned
		st.newHonestBlock()
		atkR, _ := st.publishPrivate()
		st.AtkReward += atkR
		ev.Action = fmt.Sprintf("lead=2 honest 出块 → 发布私链 → +%d to attacker，state=0", atkR)
		st.State = stateLead0

	case stateLeadGE3:
		// lead≥3 honest 出块 → 发布 1 块，lead--
		st.newHonestBlock()
		st.publishOneBlock()
		ev.Action = "lead≥3 honest 出块 → 发布 1 块（保持领先），state 仍 ≥1"
		// 状态调整：根据当前 lead
		switch st.leadDelta() {
		case 0:
			st.State = stateLead0
		case 1:
			st.State = stateLead1
		case 2:
			st.State = stateLead2
		default:
			st.State = stateLeadGE3
		}
	}
}

// publishOneBlock 把 privateChain 的下一个块"挤"进 publicChain（一次释放 1 块）。
func (st *snapState) publishOneBlock() {
	pubLen := len(st.Public)
	priLen := len(st.Private)
	if priLen <= pubLen {
		return
	}
	// 第 pubLen 高度的私链块发布
	if st.Public[pubLen-1].Owner == "honest" && pubLen < priLen {
		nextPriv := st.Private[pubLen]
		// 替换 honest tip：本来 publicTip 已经被 honest 出，要先回滚一格再追加 attacker 块
		// 教学简化：让 attacker 块直接覆盖
		oldTip := st.Public[pubLen-1]
		_ = oldTip
		// 把当前 honest tip 作废
		if st.Public[pubLen-1].Owner == "honest" {
			st.HonOrphaned++
			st.Public = st.Public[:pubLen-1]
		}
		// 追加 attacker 私链对应位置的块（不动 private）
		st.Public = append(st.Public, nextPriv)
		// 给 attacker 奖励
		st.AtkReward++
	}
}

func (st *snapState) syncPrivateToPublic() {
	if len(st.Private) < len(st.Public) {
		// 私链落后：把公链尾部追加到私链（攻击者跟上诚实链）
		for i := len(st.Private); i < len(st.Public); i++ {
			st.Private = append(st.Private, st.Public[i])
		}
	}
	// honest 在 lead=0 出的块本来要给 honest 奖励
	st.HonReward++
}

// =====================================================================
// 理论收益计算
// =====================================================================

// theoreticalRevenue Eyal & Sirer 2014 公式
// R(α, γ) = ((1 - γ) α(1 - α)² - α³) / (1 - α(1 + (2 - γ)α))
func theoreticalRevenue(alpha, gamma float64) float64 {
	num := (1-gamma)*alpha*(1-alpha)*(1-alpha) - alpha*alpha*alpha
	den := 1 - alpha*(1+(2-gamma)*alpha)
	if den == 0 {
		return 0
	}
	return num / den
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
		Alpha:       floatOr(d, "alpha", 0.33),
		Gamma:       floatOr(d, "gamma", 0.5),
		Tick:        fw.MapInt(d, "tick", 0),
		Seed:        fw.MapStr(d, "seed", "lenschain-selfish"),
		State:       fw.MapStr(d, "state", stateLead0),
		AtkReward:   fw.MapInt(d, "atk_reward", 0),
		HonReward:   fw.MapInt(d, "hon_reward", 0),
		AtkOrphaned: fw.MapInt(d, "atk_orphan", 0),
		HonOrphaned: fw.MapInt(d, "hon_orphan", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if pcAny, ok := d["public"].([]any); ok {
		for _, x := range pcAny {
			if m, ok := x.(map[string]any); ok {
				st.Public = append(st.Public, decodeBlk(m))
			}
		}
	}
	if priAny, ok := d["private"].([]any); ok {
		for _, x := range priAny {
			if m, ok := x.(map[string]any); ok {
				st.Private = append(st.Private, decodeBlk(m))
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, miningEvent{
					Tick: fw.MapInt(m, "tick", 0), Roll: floatOr(m, "roll", 0),
					WhoMines: fw.MapStr(m, "who", ""),
					State:    fw.MapStr(m, "state", ""), Action: fw.MapStr(m, "action", ""),
					PubLen: fw.MapInt(m, "pub", 0), PriLen: fw.MapInt(m, "pri", 0),
					AtkRew: fw.MapInt(m, "ar", 0), HonRew: fw.MapInt(m, "hr", 0),
				})
			}
		}
	}
	if len(st.Public) == 0 || len(st.Private) == 0 {
		return defaultSnapState()
	}
	return st
}

func decodeBlk(m map[string]any) chainBlock {
	return chainBlock{
		Height: fw.MapInt(m, "h", 0), Owner: fw.MapStr(m, "owner", ""),
		Hash: fw.MapStr(m, "hash", ""), Tick: fw.MapInt(m, "tick", 0),
	}
}

func encodeBlk(b chainBlock) map[string]any {
	return map[string]any{"h": b.Height, "owner": b.Owner, "hash": b.Hash, "tick": b.Tick}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["alpha"] = st.Alpha
	s.Data["gamma"] = st.Gamma
	s.Data["tick"] = st.Tick
	s.Data["seed"] = st.Seed
	s.Data["state"] = st.State
	s.Data["atk_reward"] = st.AtkReward
	s.Data["hon_reward"] = st.HonReward
	s.Data["atk_orphan"] = st.AtkOrphaned
	s.Data["hon_orphan"] = st.HonOrphaned
	s.Data["last_error"] = st.LastError
	pcAny := make([]any, len(st.Public))
	for i, b := range st.Public {
		pcAny[i] = encodeBlk(b)
	}
	s.Data["public"] = pcAny
	priAny := make([]any, len(st.Private))
	for i, b := range st.Private {
		priAny[i] = encodeBlk(b)
	}
	s.Data["private"] = priAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{
			"tick": ev.Tick, "roll": ev.Roll, "who": ev.WhoMines,
			"state": ev.State, "action": ev.Action,
			"pub": ev.PubLen, "pri": ev.PriLen,
			"ar": ev.AtkRew, "hr": ev.HonRew,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "自私挖矿（Selfish Mining）",
		Description:         "演示 Eyal & Sirer 2014 决策机：lead 状态空间、0' 同高度 race、γ 网络分裂、收益对照",
		Category:            fw.CategoryAttackSecurity,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPoWAttack},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"attack.selfish_mining.attacker_revenue_share",
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
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: "ready", Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode: sceneCode, SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_params", Label: "设置 α / γ",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "alpha", Type: fw.FieldNumber, Label: "α (attacker share)", Required: true, Default: 0.33, Min: 0, Max: 1, Step: 0.01},
					{Name: "gamma", Type: fw.FieldNumber, Label: "γ (network split)", Required: true, Default: 0.5, Min: 0, Max: 1, Step: 0.05},
				},
			},
			{
				ActionCode: "step", Label: "出块 1 次",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"attack.selfish_mining.attacker_revenue_share"},
			},
			{
				ActionCode: "step_n", Label: "出块 N 次",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "块数", Required: true, Default: 100, Min: 1, Step: 10},
				},
				WritesOwnedFields: []string{"attack.selfish_mining.attacker_revenue_share"},
				LinkOwnerFields:   []string{"attack.selfish_mining.attacker_revenue_share"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
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
	env := buildEnvelope(st, "init", "Selfish Mining 初始化（α=0.33, γ=0.5）", true)
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
	case "set_params":
		st.Alpha = floatOr(in.Params, "alpha", 0.33)
		st.Gamma = floatOr(in.Params, "gamma", 0.5)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("α=%.2f γ=%.2f", st.Alpha, st.Gamma), false)
		return out, nil

	case "step":
		ev := st.step()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step",
			fmt.Sprintf("tick=%d %s → %s | %s", ev.Tick, ev.WhoMines, ev.State, ev.Action), false)
		appendStepMicroSteps(&out.Render, ev)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n":
		n := fw.MapInt(in.Params, "n", 100)
		var lastEv miningEvent
		for i := 0; i < n; i++ {
			lastEv = st.step()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n",
			fmt.Sprintf("出 %d 块；attackerReward=%d / honestReward=%d (share=%.3f)",
				n, st.AtkReward, st.HonReward, attackerShare(st)), false)
		appendStepMicroSteps(&out.Render, lastEv)
		out.SharedStateDiff = ownerDiff(st)
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

func attackerShare(st snapState) float64 {
	tot := st.AtkReward + st.HonReward
	if tot == 0 {
		return 0
	}
	return float64(st.AtkReward) / float64(tot)
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 60)

	// 1) 状态机节点
	states := []string{stateLead0, stateLead0P, stateLead1, stateLead2, stateLeadGE3}
	sIDs := []string{"st-0", "st-0p", "st-1", "st-2", "st-3"}
	prims = append(prims, fw.PrimStack("state-stack", sIDs, "horizontal"))
	for i, s := range states {
		role := "fsm-state"
		status := "normal"
		if s == st.State {
			status = "active"
			role = "fsm-current"
		}
		prims = append(prims, fw.PrimNode(sIDs[i], "lead="+s, status, role))
	}

	// 2) 状态转移说明（核心边）
	prims = append(prims, fw.PrimEdge("e-01", "st-0", "st-1", "solid", "flow"))
	prims = append(prims, fw.PrimEdge("e-12", "st-1", "st-2", "solid", ""))
	prims = append(prims, fw.PrimEdge("e-23", "st-2", "st-3", "solid", ""))
	prims = append(prims, fw.PrimEdge("e-1-0p", "st-1", "st-0p", "dashed", ""))
	prims = append(prims, fw.PrimEdge("e-0p-0", "st-0p", "st-0", "dashed", ""))

	// 3) 公开链 / 私有链
	pubIDs := []string{}
	for _, b := range st.Public {
		pubIDs = append(pubIDs, "pub-"+b.Hash)
	}
	priIDs := []string{}
	for _, b := range st.Private {
		priIDs = append(priIDs, "pri-"+b.Hash)
	}
	prims = append(prims, fw.PrimStack("pub-stack", pubIDs, "horizontal"))
	prims = append(prims, fw.PrimStack("pri-stack", priIDs, "horizontal"))
	for _, b := range st.Public {
		role := "block-honest"
		if b.Owner == "attacker" {
			role = "block-attacker"
		}
		prims = append(prims, fw.PrimNode("pub-"+b.Hash,
			fmt.Sprintf("#%d\n%s\n%s", b.Height, b.Owner, b.Hash), "active", role))
	}
	for _, b := range st.Private {
		role := "block-private"
		if b.Owner == "honest" {
			role = "block-honest-shadow"
		}
		prims = append(prims, fw.PrimNode("pri-"+b.Hash,
			fmt.Sprintf("#%d\n%s\n%s", b.Height, b.Owner, b.Hash), "active", role))
	}
	for i := 0; i+1 < len(st.Public); i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("pe-%d", i),
			"pub-"+st.Public[i].Hash, "pub-"+st.Public[i+1].Hash, "solid", "flow"))
	}
	for i := 0; i+1 < len(st.Private); i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ie-%d", i),
			"pri-"+st.Private[i].Hash, "pri-"+st.Private[i+1].Hash, "dashed", ""))
	}

	// 4) 公式 + 理论收益曲线
	prims = append(prims, fw.PrimMathFormula("formula-rev",
		`R(\alpha,\gamma) = \dfrac{(1-\gamma)\alpha(1-\alpha)^2 - \alpha^3}{1 - \alpha(1+(2-\gamma)\alpha)}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-fair",
		`\text{selfish 优于 honest} \iff R(\alpha,\gamma) > \alpha`, false))

	// 5) 收益曲线（α 在 [0, 0.5]）
	curvePts := []map[string]float64{}
	for i := 0; i <= 50; i++ {
		a := float64(i) / 100.0
		curvePts = append(curvePts, map[string]float64{"x": a, "y": theoreticalRevenue(a, st.Gamma)})
	}
	prims = append(prims, fw.PrimCurve("curve-revenue", "R(α, γ) — 理论收益占比", curvePts, "solid"))
	// honest 基线 R = α
	honPts := []map[string]float64{}
	for i := 0; i <= 50; i++ {
		a := float64(i) / 100.0
		honPts = append(honPts, map[string]float64{"x": a, "y": a})
	}
	prims = append(prims, fw.PrimCurve("curve-honest", "honest baseline R = α", honPts, "dashed"))

	// 6) 状态参数 / 收益指标
	share := attackerShare(st)
	thR := theoreticalRevenue(st.Alpha, st.Gamma)
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("α = %.3f  γ = %.3f  state = %s  tick = %d\nR(α,γ) 理论 = %.4f  vs α = %.4f （selfish %s）\nattackerReward = %d  honestReward = %d  实测 share = %.4f\nattackerOrphaned = %d  honestOrphaned = %d\npublic_len = %d  private_len = %d  lead = %d",
			st.Alpha, st.Gamma, st.State, st.Tick,
			thR, st.Alpha, ifThenStr(thR > st.Alpha, "更优", "不优"),
			st.AtkReward, st.HonReward, share,
			st.AtkOrphaned, st.HonOrphaned,
			len(st.Public)-1, len(st.Private)-1, st.leadDelta()),
		"text", nil, 10))

	// 7) 进度条
	prims = append(prims, fw.PrimProgressBar("bar-share", share*100, 100,
		fmt.Sprintf("attacker share %.2f%% (理论 %.2f%%)", share*100, thR*100)))
	prims = append(prims, fw.PrimBar("bar-orphan-h", float64(st.HonOrphaned), 0, "warning", "Honest blocks orphaned"))
	prims = append(prims, fw.PrimBar("bar-orphan-a", float64(st.AtkOrphaned), 0, "info", "Attacker blocks orphaned"))

	// 8) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"tick  who       state  action                                              pub  pri  atkR  honR"}
		startIdx := 0
		if len(st.Events) > 18 {
			startIdx = len(st.Events) - 18
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-4d  %-9s %-5s  %-50s  %-3d  %-3d  %-4d  %d",
				ev.Tick, ev.WhoMines, ev.State, ev.Action, ev.PubLen, ev.PriLen, ev.AtkRew, ev.HonRew))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 20))
	}

	// 9) 收益饼图
	prims = append(prims, fw.PrimPieChart("rev-pie", []map[string]any{
		{"label": "Attacker", "value": float64(st.AtkReward), "color_role": "danger"},
		{"label": "Honest", "value": float64(st.HonReward), "color_role": "success"},
	}))

	// 10) 动效
	if share > st.Alpha {
		prims = append(prims, fw.PrimPulse("pulse-win", "bar-share", "danger", 1500))
		prims = append(prims, fw.PrimGlow("glow-win", "bar-share", "danger", 0.9))
	}
	prims = append(prims, fw.PrimGlow("glow-state", "st-"+stateKey(st.State), "info", 0.7))

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pow", linkGroupPoWAttack, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Selfish 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func stateKey(s string) string {
	switch s {
	case stateLead0:
		return "0"
	case stateLead0P:
		return "0p"
	case stateLead1:
		return "1"
	case stateLead2:
		return "2"
	case stateLeadGE3:
		return "3"
	}
	return "0"
}

func ifThenStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	share := attackerShare(st)
	d := map[string]any{
		"alpha":                  st.Alpha,
		"gamma":                  st.Gamma,
		"state":                  st.State,
		"public_len":             len(st.Public) - 1,
		"private_len":            len(st.Private) - 1,
		"lead":                   st.leadDelta(),
		"attacker_reward":        st.AtkReward,
		"honest_reward":          st.HonReward,
		"attacker_revenue_share": share,
		"theoretical_revenue":    theoreticalRevenue(st.Alpha, st.Gamma),
		"honest_orphaned":        st.HonOrphaned,
		"attacker_orphaned":      st.AtkOrphaned,
		"tick":                   st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendStepMicroSteps(env *fw.RenderEnvelope, ev miningEvent) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: fmt.Sprintf("rng=%.3f → %s 出块", ev.Roll, ev.WhoMines), DurationMs: 400, HighlightIDs: []string{"cb-events"}},
		{ID: "s-2", Label: ev.Action, DurationMs: 500, HighlightIDs: []string{"state-stack", "pub-stack", "pri-stack"}, FirePrimitives: []string{"glow-state"}},
		{ID: "s-3", Label: fmt.Sprintf("奖励：atk=%d honest=%d", ev.AtkRew, ev.HonRew), DurationMs: 400, HighlightIDs: []string{"rev-pie", "bar-share"}, FirePrimitives: []string{"glow-win", "pulse-win"}, IsLinkTrigger: true},
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
		ID:             "selfish-mining-attack",
		SourceScene:    sceneCode,
		SourceAction:   "step_n",
		LinkGroup:      linkGroupPoWAttack,
		ChangedFields:  []string{"attack.selfish_mining.attacker_revenue_share"},
		Payload:        map[string]any{"atk_reward": st.AtkReward, "hon_reward": st.HonReward},
		SourceAnchorID: "selfish-mining-anchor",
		TargetAnchorID: "pow-chain-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	share := attackerShare(st)
	return map[string]any{
		"attack": map[string]any{
			"selfish_mining": map[string]any{
				"alpha":                  st.Alpha,
				"gamma":                  st.Gamma,
				"state":                  st.State,
				"attacker_reward":        st.AtkReward,
				"honest_reward":          st.HonReward,
				"attacker_revenue_share": share,
				"lead":                   st.leadDelta(),
				"honest_orphaned":        st.HonOrphaned,
				"attacker_orphaned":      st.AtkOrphaned,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func floatOr(m map[string]any, k string, def float64) float64 {
	if m == nil {
		return def
	}
	if v, ok := m[k]; ok {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		case int64:
			return float64(t)
		}
	}
	return def
}
