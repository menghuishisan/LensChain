// 模块：sim-engine/scenarios/internal/attacksecurity/fiftyoneattack
// 文件职责：ATK-01 51% 攻击场景的完整实现。
//
// SSOT 依据：06.md §4.7.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 PoW 双链竞争 + 算力切片 + 重组（零外部依赖）。
//
//   1. 网络模型：
//      · totalHash = honestHash + attackerHash（默认 60% / 40%）
//      · attackerShare = attackerHash / totalHash
//
//   2. 链结构：
//      · publicChain  : 全网公开主链（含 honest + attacker 公开出的块）
//      · privateChain : 攻击者私下挖的链（不广播）
//      · 每个块: height, parent, miner ∈ {honest, attacker}, txHash, time
//
//   3. 出块过程（每 tick 推进一次）：
//      · 用确定性伪随机：rng = keccak256(seed || tick) 取低 64 位
//      · 概率 attackerShare → 攻击者出块；否则诚实节点出块
//      · 攻击者出块默认进 privateChain；诚实节点出块进 publicChain
//      · isPublishing 时攻击者把 privateChain 的最早未公开块同步进 publicChain
//
//   4. 重组规则（Nakamoto 最长链规则）：
//      · 攻击者发布 privateChain 后，若 len(privateChain) > len(publicChain)，
//        节点切换到 attacker 链，发生 reorg
//      · 重组时回滚 publicChain 的尾部块（含 confirmedTx），导致 doubleSpendSuccess++
//
//   5. 教学攻击：
//      · sendVictimTx        : 受害方在 publicChain 末端付款
//      · attackerSecretSpend : 攻击者在 privateChain 同 nonce 写"反向"花费
//      · publishPrivateChain : 揭示 privateChain，触发可能的 reorg
//
//   6. 数据指标：
//      · publicLength / privateLength
//      · leadGap = privateLength - publicLength
//      · attackerBlocksInPublic / attackerBlocksInPrivate
//      · doubleSpendSuccess（成功撤销受害交易次数）

package fiftyoneattack

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
)

const (
	sceneCode     = "51-percent-attack"
	schemaVersion = "v1.0.0"
	algorithmType = "pow-double-chain"

	minerHonest   = "honest"
	minerAttacker = "attacker"

	linkGroupPoWAttack = "pow-attack-group"
	linkOwnerSubtree   = "attack.fifty_one"
)

// =====================================================================
// 数据结构
// =====================================================================

type block struct {
	Height int
	Parent string
	Miner  string
	Hash   string
	Txs    []string
	Tick   int
}

func (b block) shortHash() string {
	if len(b.Hash) <= 8 {
		return b.Hash
	}
	return b.Hash[:8]
}

type pendingTx struct {
	ID         string
	From       string
	To         string
	Amount     int64
	OnChain    string // public / private / -
	IncludedAt int    // 块高度
}

type reorgEvent struct {
	Tick             int
	OldLen           int
	NewLen           int
	BlocksRolledBack int
	DoubleSpentTxIDs []string
}

type snapState struct {
	HonestHash       int // 默认 60
	AttackerHash     int // 默认 40
	Tick             int
	Seed             string // 伪随机种子
	PublicChain      []block
	PrivateChain     []block
	NextTxID         int
	Txs              map[string]*pendingTx
	ReorgEvents      []reorgEvent
	AttackerBlocks   int // 累计：攻击者挖的所有块（含未发布的）
	AttackerInPublic int // 攻击者在 publicChain 上的块数
	DoubleSpends     int
	LastError        string
}

func defaultSnapState() snapState {
	st := snapState{
		HonestHash: 60, AttackerHash: 40, Seed: "lenschain-51",
		Txs: map[string]*pendingTx{},
	}
	// genesis
	g := block{Height: 0, Parent: "", Miner: minerHonest, Tick: 0, Hash: "GENESIS00"}
	st.PublicChain = append(st.PublicChain, g)
	st.PrivateChain = append(st.PrivateChain, g)
	return st
}

func (st snapState) attackerShare() float64 {
	tot := st.HonestHash + st.AttackerHash
	if tot <= 0 {
		return 0
	}
	return float64(st.AttackerHash) / float64(tot)
}

// rngBit 给 (tick, salt) 生成确定性 [0, 1) 浮点。
func (st snapState) rngFloat(tick int, salt string) float64 {
	buf := []byte(st.Seed)
	buf = append(buf, []byte(salt)...)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(tick))
	buf = append(buf, b[:]...)
	d := keccak256hash.Sum256(buf)
	v := binary.BigEndian.Uint64(d[:8])
	return float64(v%1_000_000) / 1_000_000.0
}

// blockHash 自实现块哈希：keccak256(seed || height || parent || miner || tick || txs)
func (st snapState) blockHash(b block) string {
	buf := []byte(st.Seed)
	var num [8]byte
	binary.BigEndian.PutUint64(num[:], uint64(b.Height))
	buf = append(buf, num[:]...)
	buf = append(buf, []byte(b.Parent)...)
	buf = append(buf, []byte(b.Miner)...)
	binary.BigEndian.PutUint64(num[:], uint64(b.Tick))
	buf = append(buf, num[:]...)
	for _, t := range b.Txs {
		buf = append(buf, []byte(t)...)
	}
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:6])
}

// mineTick 一次出块：根据算力比例决定 miner，块加入相应链。
func (st *snapState) mineTick(announceAttacker bool) (block, error) {
	st.Tick++
	roll := st.rngFloat(st.Tick, "miner")
	miner := minerHonest
	if roll < st.attackerShare() {
		miner = minerAttacker
	}
	parentChain := st.PublicChain
	if miner == minerAttacker && !announceAttacker {
		parentChain = st.PrivateChain
	}
	parent := parentChain[len(parentChain)-1]
	newBlk := block{
		Height: parent.Height + 1, Parent: parent.Hash, Miner: miner, Tick: st.Tick,
	}
	// 把 mempool 中未上链的交易尽量上链
	if miner == minerHonest {
		for _, t := range st.txsByOnChain("-") {
			newBlk.Txs = append(newBlk.Txs, t.ID)
		}
	} else {
		// 攻击者拒绝把受害交易写入私链（挑选无 'victim' 标签的）
		for _, t := range st.txsByOnChain("-") {
			if !strings.Contains(t.ID, "victim") {
				newBlk.Txs = append(newBlk.Txs, t.ID)
			}
		}
	}
	newBlk.Hash = st.blockHash(newBlk)

	if miner == minerAttacker {
		st.AttackerBlocks++
		if announceAttacker {
			st.PublicChain = append(st.PublicChain, newBlk)
			st.AttackerInPublic++
			st.markTxsOnChain(newBlk.Txs, "public", newBlk.Height)
		} else {
			st.PrivateChain = append(st.PrivateChain, newBlk)
			st.markTxsOnChain(newBlk.Txs, "private", newBlk.Height)
		}
	} else {
		st.PublicChain = append(st.PublicChain, newBlk)
		st.markTxsOnChain(newBlk.Txs, "public", newBlk.Height)
		// 私链尾部要继续延伸（攻击者并非每 tick 都能跟进；
		// 简化：攻击者总是 fork 自己最新块，等下一 tick 再扩展）
	}
	return newBlk, nil
}

func (st snapState) txsByOnChain(tag string) []*pendingTx {
	out := []*pendingTx{}
	keys := []string{}
	for k := range st.Txs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if st.Txs[k].OnChain == tag {
			out = append(out, st.Txs[k])
		}
	}
	return out
}

func (st *snapState) markTxsOnChain(ids []string, tag string, height int) {
	for _, id := range ids {
		if t, ok := st.Txs[id]; ok {
			t.OnChain = tag
			t.IncludedAt = height
		}
	}
}

// publishPrivateChain 攻击者把 privateChain 揭示出来：若长度更长 → reorg。
func (st *snapState) publishPrivateChain() reorgEvent {
	pubLen := len(st.PublicChain)
	priLen := len(st.PrivateChain)
	ev := reorgEvent{Tick: st.Tick, OldLen: pubLen, NewLen: priLen}
	if priLen <= pubLen {
		// 不足以重组，攻击者所有私链块作废（教学：直接清空 privateChain，回到 publicChain 末尾）
		// privateChain 的攻击者块就此放弃；公链不变
		st.PrivateChain = append([]block{}, st.PublicChain...)
		return ev
	}
	// 计算被回滚的 public block 与受害交易（已 confirmed 但因 reorg 撤销）
	rolledBack := []block{}
	for _, b := range st.PublicChain {
		// 任何不在 privateChain 上的块都被回滚
		matched := false
		for _, pb := range st.PrivateChain {
			if pb.Hash == b.Hash {
				matched = true
				break
			}
		}
		if !matched {
			rolledBack = append(rolledBack, b)
		}
	}
	ev.BlocksRolledBack = len(rolledBack)
	for _, b := range rolledBack {
		for _, txid := range b.Txs {
			if t, ok := st.Txs[txid]; ok {
				if strings.HasPrefix(txid, "victim") {
					st.DoubleSpends++
					ev.DoubleSpentTxIDs = append(ev.DoubleSpentTxIDs, txid)
				}
				t.OnChain = "-" // 回到 mempool
				t.IncludedAt = 0
			}
		}
	}
	// 切换主链
	// 重新统计 attackerInPublic
	st.PublicChain = append([]block{}, st.PrivateChain...)
	cnt := 0
	for _, b := range st.PublicChain {
		if b.Miner == minerAttacker {
			cnt++
		}
	}
	st.AttackerInPublic = cnt
	st.ReorgEvents = append(st.ReorgEvents, ev)
	if len(st.ReorgEvents) > 16 {
		st.ReorgEvents = st.ReorgEvents[len(st.ReorgEvents)-16:]
	}
	return ev
}

// addTx 添加一笔交易到 mempool。
func (st *snapState) addTx(prefix, from, to string, amount int64) *pendingTx {
	st.NextTxID++
	id := fmt.Sprintf("%s-%d", prefix, st.NextTxID)
	t := &pendingTx{ID: id, From: from, To: to, Amount: amount, OnChain: "-"}
	st.Txs[id] = t
	return t
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
		HonestHash:       fw.MapInt(d, "honest_hash", 60),
		AttackerHash:     fw.MapInt(d, "attacker_hash", 40),
		Tick:             fw.MapInt(d, "tick", 0),
		Seed:             fw.MapStr(d, "seed", "lenschain-51"),
		NextTxID:         fw.MapInt(d, "next_tx", 0),
		AttackerBlocks:   fw.MapInt(d, "atk_blocks", 0),
		AttackerInPublic: fw.MapInt(d, "atk_in_public", 0),
		DoubleSpends:     fw.MapInt(d, "double_spends", 0),
		LastError:        fw.MapStr(d, "last_error", ""),
		Txs:              map[string]*pendingTx{},
	}
	if pcAny, ok := d["public_chain"].([]any); ok {
		for _, x := range pcAny {
			if m, ok := x.(map[string]any); ok {
				st.PublicChain = append(st.PublicChain, decodeBlock(m))
			}
		}
	}
	if priAny, ok := d["private_chain"].([]any); ok {
		for _, x := range priAny {
			if m, ok := x.(map[string]any); ok {
				st.PrivateChain = append(st.PrivateChain, decodeBlock(m))
			}
		}
	}
	if len(st.PublicChain) == 0 || len(st.PrivateChain) == 0 {
		return defaultSnapState()
	}
	if txsAny, ok := d["txs"].(map[string]any); ok {
		for id, vAny := range txsAny {
			if tm, ok := vAny.(map[string]any); ok {
				st.Txs[id] = &pendingTx{
					ID:   id,
					From: fw.MapStr(tm, "from", ""), To: fw.MapStr(tm, "to", ""),
					Amount:     int64(fw.MapInt(tm, "amount", 0)),
					OnChain:    fw.MapStr(tm, "on_chain", "-"),
					IncludedAt: fw.MapInt(tm, "included_at", 0),
				}
			}
		}
	}
	if rAny, ok := d["reorgs"].([]any); ok {
		for _, x := range rAny {
			if m, ok := x.(map[string]any); ok {
				ev := reorgEvent{
					Tick:   fw.MapInt(m, "tick", 0),
					OldLen: fw.MapInt(m, "old", 0), NewLen: fw.MapInt(m, "new", 0),
					BlocksRolledBack: fw.MapInt(m, "rolled", 0),
				}
				if dsAny, ok := m["ds"].([]any); ok {
					for _, y := range dsAny {
						if s, ok := y.(string); ok {
							ev.DoubleSpentTxIDs = append(ev.DoubleSpentTxIDs, s)
						}
					}
				}
				st.ReorgEvents = append(st.ReorgEvents, ev)
			}
		}
	}
	return st
}

func decodeBlock(m map[string]any) block {
	b := block{
		Height: fw.MapInt(m, "h", 0),
		Parent: fw.MapStr(m, "parent", ""),
		Miner:  fw.MapStr(m, "miner", ""),
		Hash:   fw.MapStr(m, "hash", ""),
		Tick:   fw.MapInt(m, "tick", 0),
	}
	if txsAny, ok := m["txs"].([]any); ok {
		for _, x := range txsAny {
			if s, ok := x.(string); ok {
				b.Txs = append(b.Txs, s)
			}
		}
	}
	return b
}

func encodeBlock(b block) map[string]any {
	txs := make([]any, len(b.Txs))
	for i, t := range b.Txs {
		txs[i] = t
	}
	return map[string]any{
		"h": b.Height, "parent": b.Parent, "miner": b.Miner,
		"hash": b.Hash, "tick": b.Tick, "txs": txs,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["honest_hash"] = st.HonestHash
	s.Data["attacker_hash"] = st.AttackerHash
	s.Data["tick"] = st.Tick
	s.Data["seed"] = st.Seed
	s.Data["next_tx"] = st.NextTxID
	s.Data["atk_blocks"] = st.AttackerBlocks
	s.Data["atk_in_public"] = st.AttackerInPublic
	s.Data["double_spends"] = st.DoubleSpends
	s.Data["last_error"] = st.LastError
	pcAny := make([]any, len(st.PublicChain))
	for i, b := range st.PublicChain {
		pcAny[i] = encodeBlock(b)
	}
	s.Data["public_chain"] = pcAny
	priAny := make([]any, len(st.PrivateChain))
	for i, b := range st.PrivateChain {
		priAny[i] = encodeBlock(b)
	}
	s.Data["private_chain"] = priAny
	tAny := map[string]any{}
	for id, t := range st.Txs {
		tAny[id] = map[string]any{"from": t.From, "to": t.To,
			"amount": int(t.Amount), "on_chain": t.OnChain, "included_at": t.IncludedAt}
	}
	s.Data["txs"] = tAny
	rAny := make([]any, len(st.ReorgEvents))
	for i, e := range st.ReorgEvents {
		ds := make([]any, len(e.DoubleSpentTxIDs))
		for j, id := range e.DoubleSpentTxIDs {
			ds[j] = id
		}
		rAny[i] = map[string]any{"tick": e.Tick, "old": e.OldLen,
			"new": e.NewLen, "rolled": e.BlocksRolledBack, "ds": ds}
	}
	s.Data["reorgs"] = rAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "51% 攻击",
		Description:         "演示 PoW 公开链 vs 攻击者私链 + 算力切片 + 攻击者发布 → 重组 → 双花",
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
			"attack.fifty_one.double_spends",
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
				ActionCode: "set_hashrate", Label: "设置算力分布",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "honest", Type: fw.FieldNumber, Label: "诚实算力", Required: true, Default: 60, Min: 0, Step: 5},
					{Name: "attacker", Type: fw.FieldNumber, Label: "攻击者算力", Required: true, Default: 40, Min: 0, Step: 5},
				},
			},
			{
				ActionCode: "send_victim_tx", Label: "受害交易",
				Description:   "受害方付款给商家，进入 mempool",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "amount", Type: fw.FieldNumber, Label: "金额", Required: true, Default: 100, Min: 1, Step: 10},
				},
			},
			{
				ActionCode: "mine_tick", Label: "出块 1 tick",
				Description: "按算力比例随机选 miner，攻击者出块默认进 privateChain",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"attack.fifty_one.double_spends"},
			},
			{
				ActionCode: "mine_n", Label: "出块 N tick",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 10, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "publish_private", Label: "发布私链",
				Description: "若 priv > pub → reorg → 双花",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.InterveneAttack,
				WritesOwnedFields: []string{"attack.fifty_one.double_spends"},
				LinkOwnerFields:   []string{"attack.fifty_one.double_spends"},
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
	env := buildEnvelope(st, "init", "51% 攻击场景：60% honest / 40% attacker", true)
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
	case "set_hashrate":
		h := fw.MapInt(in.Params, "honest", 60)
		a := fw.MapInt(in.Params, "attacker", 40)
		if h+a == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "算力不能全为 0"}, nil
		}
		st.HonestHash = h
		st.AttackerHash = a
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_hashrate",
			fmt.Sprintf("算力 honest=%d attacker=%d (share=%.2f%%)", h, a, st.attackerShare()*100), true)
		return out, nil

	case "send_victim_tx":
		amt := int64(fw.MapInt(in.Params, "amount", 100))
		t := st.addTx("victim", "alice", "merchant", amt)
		saveState(state, st)
		out.Render = buildEnvelope(st, "send_victim_tx",
			fmt.Sprintf("受害交易 %s 入 mempool", t.ID), false)
		appendVictimTxMicroSteps(&out.Render, t.ID)
		return out, nil

	case "mine_tick":
		blk, _ := st.mineTick(false)
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_tick",
			fmt.Sprintf("tick=%d miner=%s 块 #%d", st.Tick, blk.Miner, blk.Height), false)
		appendMineMicroSteps(&out.Render, blk)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_n":
		n := fw.MapInt(in.Params, "n", 10)
		var last block
		for i := 0; i < n; i++ {
			last, _ = st.mineTick(false)
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_n",
			fmt.Sprintf("出 %d 块；公链 %d；私链 %d", n, len(st.PublicChain)-1, len(st.PrivateChain)-1), false)
		appendMineMicroSteps(&out.Render, last)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "publish_private":
		ev := st.publishPrivateChain()
		saveState(state, st)
		summary := fmt.Sprintf("发布私链：oldLen=%d newLen=%d 回滚=%d 双花=%d",
			ev.OldLen, ev.NewLen, ev.BlocksRolledBack, len(ev.DoubleSpentTxIDs))
		if ev.NewLen <= ev.OldLen {
			summary = fmt.Sprintf("私链 %d ≤ 公链 %d，攻击作废", ev.NewLen, ev.OldLen)
		}
		out.Render = buildEnvelope(st, "publish_private", summary, false)
		appendPublishMicroSteps(&out.Render, ev)
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

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 60)

	// 1) 上下两条链：public（上）、private（下）
	pubIDs := make([]string, len(st.PublicChain))
	for i, b := range st.PublicChain {
		pubIDs[i] = "pub-" + b.Hash
	}
	priIDs := make([]string, len(st.PrivateChain))
	for i, b := range st.PrivateChain {
		priIDs[i] = "pri-" + b.Hash
	}
	prims = append(prims, fw.PrimStack("public-stack", pubIDs, "horizontal"))
	prims = append(prims, fw.PrimStack("private-stack", priIDs, "horizontal"))

	for _, b := range st.PublicChain {
		role := "block-honest"
		if b.Miner == minerAttacker {
			role = "block-attacker-public"
		}
		label := fmt.Sprintf("#%d\n%s\n%s\ntx=%d", b.Height, b.Miner, b.shortHash(), len(b.Txs))
		prims = append(prims, fw.PrimNode("pub-"+b.Hash, label, "active", role))
	}
	for _, b := range st.PrivateChain {
		role := "block-attacker"
		if b.Miner == minerHonest {
			role = "block-honest-shadow"
		}
		label := fmt.Sprintf("#%d\n%s\n%s\ntx=%d", b.Height, b.Miner, b.shortHash(), len(b.Txs))
		prims = append(prims, fw.PrimNode("pri-"+b.Hash, label, "active", role))
	}

	// 2) 链边
	for i := 0; i+1 < len(st.PublicChain); i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("pub-e-%d", i),
			"pub-"+st.PublicChain[i].Hash, "pub-"+st.PublicChain[i+1].Hash, "solid", "flow"))
	}
	for i := 0; i+1 < len(st.PrivateChain); i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("pri-e-%d", i),
			"pri-"+st.PrivateChain[i].Hash, "pri-"+st.PrivateChain[i+1].Hash, "dashed", ""))
	}

	// 3) 算力饼图 + 进度条
	prims = append(prims, fw.PrimPieChart("hash-pie", []map[string]any{
		{"label": "Honest", "value": float64(st.HonestHash), "color_role": "success"},
		{"label": "Attacker", "value": float64(st.AttackerHash), "color_role": "danger"},
	}))
	prims = append(prims, fw.PrimProgressBar("bar-share", st.attackerShare()*100, 100,
		fmt.Sprintf("Attacker share %.1f%%", st.attackerShare()*100)))

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-share",
		`P_{\text{attacker block}} = \frac{H_{\text{attacker}}}{H_{\text{honest}} + H_{\text{attacker}}};\quad
		\text{reorg iff } |\text{priv}| > |\text{pub}|`, false))

	// 5) leadGap 图：私链领先长度
	leadGap := len(st.PrivateChain) - len(st.PublicChain)
	leadColor := "warning"
	if leadGap > 0 {
		leadColor = "danger"
	}
	prims = append(prims, fw.PrimBar("bar-lead", float64(leadGap), 0, leadColor,
		fmt.Sprintf("Private − Public = %d", leadGap)))
	prims = append(prims, fw.PrimBar("bar-double", float64(st.DoubleSpends), 0, "danger", "Double-Spend Success"))
	prims = append(prims, fw.PrimBar("bar-attacker-pub", float64(st.AttackerInPublic), 0, "warning", "Attacker Blocks in Public Chain"))

	// 6) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d  seed = %s\nhonest_hash = %d  attacker_hash = %d  share = %.2f%%\npublic_len = %d  private_len = %d  lead = %d\nattacker_blocks (total) = %d\nattacker_in_public = %d\ndouble_spends = %d  reorgs = %d",
			st.Tick, st.Seed, st.HonestHash, st.AttackerHash, st.attackerShare()*100,
			len(st.PublicChain)-1, len(st.PrivateChain)-1, leadGap,
			st.AttackerBlocks, st.AttackerInPublic, st.DoubleSpends, len(st.ReorgEvents)),
		"text", nil, 8))

	// 7) 公链表
	pubLines := []string{"public chain（高度 / miner / hash / txs）"}
	startIdx := 0
	if len(st.PublicChain) > 12 {
		startIdx = len(st.PublicChain) - 12
	}
	for _, b := range st.PublicChain[startIdx:] {
		pubLines = append(pubLines, fmt.Sprintf("  #%-3d %-9s %-10s tx=%d", b.Height, b.Miner, b.shortHash(), len(b.Txs)))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-public", strings.Join(pubLines, "\n"), "text", nil, 14))

	// 8) 私链表
	priLines := []string{"private chain（攻击者隐藏）"}
	startIdx2 := 0
	if len(st.PrivateChain) > 12 {
		startIdx2 = len(st.PrivateChain) - 12
	}
	for _, b := range st.PrivateChain[startIdx2:] {
		priLines = append(priLines, fmt.Sprintf("  #%-3d %-9s %-10s tx=%d", b.Height, b.Miner, b.shortHash(), len(b.Txs)))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-private", strings.Join(priLines, "\n"), "text", nil, 14))

	// 9) 交易表
	if len(st.Txs) > 0 {
		txLines := []string{"id            from→to             amount  on_chain  height"}
		ids := []string{}
		for k := range st.Txs {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		for _, id := range ids {
			t := st.Txs[id]
			txLines = append(txLines, fmt.Sprintf("  %-12s %-7s→%-7s %-6d  %-8s  %d",
				t.ID, t.From, t.To, t.Amount, t.OnChain, t.IncludedAt))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-txs", strings.Join(txLines, "\n"), "text", nil, 12))
	}

	// 10) 重组事件表
	if len(st.ReorgEvents) > 0 {
		rLines := []string{"reorgs：tick  oldLen  newLen  rolledBack  doubleSpentTxs"}
		for _, e := range st.ReorgEvents {
			rLines = append(rLines, fmt.Sprintf("        %-4d  %-6d  %-6d  %-10d  [%s]",
				e.Tick, e.OldLen, e.NewLen, e.BlocksRolledBack,
				strings.Join(e.DoubleSpentTxIDs, ", ")))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-reorgs", strings.Join(rLines, "\n"), "text", nil, 10))
	}

	// 11) 动效
	if len(st.PublicChain) > 0 {
		last := st.PublicChain[len(st.PublicChain)-1]
		prims = append(prims, fw.PrimGlow("glow-tip", "pub-"+last.Hash, "info", 0.7))
	}
	if leadGap > 0 {
		prims = append(prims, fw.PrimPulse("pulse-priv", "private-stack", "warning", 1500))
	}
	if st.DoubleSpends > 0 {
		prims = append(prims, fw.PrimShake("shake-ds", "bar-double", 0.5, 800))
		prims = append(prims, fw.PrimBurst("burst-ds", "bar-double", "danger", int64(st.DoubleSpends), 700))
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pow", linkGroupPoWAttack, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "51% 攻击错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"public_len":         len(st.PublicChain) - 1,
		"private_len":        len(st.PrivateChain) - 1,
		"lead_gap":           len(st.PrivateChain) - len(st.PublicChain),
		"attacker_share":     st.attackerShare(),
		"attacker_blocks":    st.AttackerBlocks,
		"attacker_in_public": st.AttackerInPublic,
		"double_spends":      st.DoubleSpends,
		"reorg_count":        len(st.ReorgEvents),
		"tick":               st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendVictimTxMicroSteps(env *fw.RenderEnvelope, txID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "v-1", Label: "alice → merchant 发起付款", DurationMs: 400, HighlightIDs: []string{"cb-txs"}},
		{ID: "v-2", Label: "进入 mempool（pending）", DurationMs: 400, HighlightIDs: []string{"cb-txs"}},
		{ID: "v-3", Label: "等下一块打包到 publicChain", DurationMs: 400, HighlightIDs: []string{"public-stack"}, IsLinkTrigger: true},
	}
}

func appendMineMicroSteps(env *fw.RenderEnvelope, blk block) {
	chain := "publicChain"
	if blk.Miner == minerAttacker {
		chain = "privateChain（隐藏）"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: "rng < attacker_share？", DurationMs: 400, HighlightIDs: []string{"hash-pie", "formula-share"}},
		{ID: "m-2", Label: fmt.Sprintf("%s 出块 #%d", blk.Miner, blk.Height), DurationMs: 500, HighlightIDs: []string{"public-stack", "private-stack"}, FirePrimitives: []string{"glow-tip"}},
		{ID: "m-3", Label: "块加入 " + chain, DurationMs: 400, HighlightIDs: []string{"cb-public", "cb-private", "bar-lead"}, IsLinkTrigger: true},
	}
}

func appendPublishMicroSteps(env *fw.RenderEnvelope, ev reorgEvent) {
	tail := fmt.Sprintf("private %d ≤ public %d，攻击作废", ev.NewLen, ev.OldLen)
	if ev.NewLen > ev.OldLen {
		tail = fmt.Sprintf("private %d > public %d → reorg，回滚 %d 块，双花 %d 笔",
			ev.NewLen, ev.OldLen, ev.BlocksRolledBack, len(ev.DoubleSpentTxIDs))
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "p-1", Label: "攻击者揭示 privateChain", DurationMs: 400, HighlightIDs: []string{"private-stack"}, FirePrimitives: []string{"pulse-priv"}},
		{ID: "p-2", Label: "节点比较两条链长度（最长链规则）", DurationMs: 500, HighlightIDs: []string{"formula-share", "bar-lead"}},
		{ID: "p-3", Label: tail, DurationMs: 600, HighlightIDs: []string{"cb-reorgs", "bar-double"}, FirePrimitives: []string{"shake-ds", "burst-ds"}, IsLinkTrigger: true},
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
		ID:             "fifty-one-attack",
		SourceScene:    sceneCode,
		SourceAction:   "publish_private",
		LinkGroup:      linkGroupPoWAttack,
		ChangedFields:  []string{"attack.fifty_one.double_spends", "attack.fifty_one.lead_gap"},
		Payload:        map[string]any{"double_spends": st.DoubleSpends},
		SourceAnchorID: "fifty-one-anchor",
		TargetAnchorID: "pow-chain-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"attack": map[string]any{
			"fifty_one": map[string]any{
				"public_len":         len(st.PublicChain) - 1,
				"private_len":        len(st.PrivateChain) - 1,
				"lead_gap":           len(st.PrivateChain) - len(st.PublicChain),
				"attacker_share":     st.attackerShare(),
				"attacker_in_public": st.AttackerInPublic,
				"double_spends":      st.DoubleSpends,
				"reorg_count":        len(st.ReorgEvents),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

