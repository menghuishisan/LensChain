// 模块：sim-engine/scenarios/internal/attacksecurity/doublespend
// 文件职责：ATK-02 双花攻击场景的完整实现。
//
// SSOT 依据：06.md §4.7.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 UTXO 模型 + 双花攻击三种变体（零外部依赖，Keccak 复用 keccak256hash）。
//
//   1. UTXO 模型：
//      · 账户余额由 utxo 集合表示：{utxoID, owner, value, spent}
//      · 一笔 tx：spend (in_utxos[]) → produce (out_utxos[]) ；签名采用确定性 hash
//      · same nonce / same input 视为冲突
//
//   2. 三种双花变体：
//      · race attack         : 同时发出 tx_A→merchant 和 tx_B→attacker_self；
//                              不同的节点先看到不同 tx；最终只一笔上链
//      · finney attack       : 攻击者预挖 1 块（含 tx_B→self）但不广播；
//                              用 tx_A 付款给商家（0-confirm 接受），
//                              紧接着发布预挖块 → tx_B 被链上承认 → tx_A 失败
//      · vector76 attack     : 攻击者预挖 1 块（tx_B），等到下一区块到来时，
//                              先把 tx_A 提交到 merchant，再发布预挖块上 fork；
//                              利用商家的 1-confirm 仍然可被回滚
//
//   3. 算法共同部分：
//      · mempoolTx[] 标记 ownership = honest / attacker
//      · 受害交易 = tx_A，金额 v 给商家；攻击交易 = tx_B，同一 input、目标=攻击者
//      · 通过 confirmations 阈值判定商家是否被欺骗
//
//   4. 教学指标：
//      · attemptsRace, attemptsFinney, attemptsVector
//      · successCount / failureCount
//      · merchantBalance / attackerStolen
//      · confirmationsAtAccept （商家接受时确认数）

package doublespend

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
	sceneCode     = "double-spend"
	schemaVersion = "v1.0.0"
	algorithmType = "utxo-double-spend"

	variantRace   = "race"
	variantFinney = "finney"
	variantVector = "vector76"

	linkGroupPoWAttack = "pow-attack-group"
	linkOwnerSubtree   = "attack.double_spend"
)

// =====================================================================
// 数据结构
// =====================================================================

type utxo struct {
	ID      string
	Owner   string
	Value   int64
	Spent   bool
	SpentBy string // tx id
}

type tx struct {
	ID            string
	From          string
	To            string
	InUTXOs       []string
	Value         int64
	Owner         string // honest / attacker
	Hash          string
	Sig           string
	Status        string // pending / mempool / inblock / dropped / replaced
	IncludedBlock int
	Variant       string
}

type block struct {
	Height  int
	Parent  string
	Hash    string
	Miner   string // honest / attacker
	Txs     []string
	Tick    int
	Visible bool // false = attacker 私藏；true = 已广播
}

type attackAttempt struct {
	Variant       string
	Tick          int
	VictimTxID    string
	AttackerTxID  string
	Success       bool
	Detail        string
	Stolen        int64
	ConfsAtAccept int
}

type snapState struct {
	Tick                                int
	Seed                                string
	UTXOs                               map[string]*utxo
	Txs                                 map[string]*tx
	NextTxID                            int
	Chain                               []block // 已确认主链（不含攻击者私藏块）
	Hidden                              []block // 攻击者私藏块（finney/vector）
	NextBlockHeight                     int
	MerchantBalance                     int64
	AttackerStolen                      int64
	Honor                               map[string]int64 // 余额视图：honest / attacker / merchant
	ConfirmationsReq                    int              // 商家接受需要多少确认（教学：0/1/3 三档）
	Attempts                            []attackAttempt
	RaceCount, FinneyCount, VectorCount int
	SuccessCount, FailureCount          int
	LastError                           string
}

func defaultSnapState() snapState {
	st := snapState{
		Seed: "lenschain-ds", ConfirmationsReq: 0,
		UTXOs: map[string]*utxo{}, Txs: map[string]*tx{},
		Honor: map[string]int64{"honest_user": 1000, "attacker": 1000, "merchant": 0},
	}
	// 初始 utxo
	st.UTXOs["u-honest-1"] = &utxo{ID: "u-honest-1", Owner: "honest_user", Value: 1000}
	st.UTXOs["u-attacker-1"] = &utxo{ID: "u-attacker-1", Owner: "attacker", Value: 1000}
	st.NextBlockHeight = 1
	// genesis
	g := block{Height: 0, Hash: "GENESIS", Miner: "honest", Visible: true}
	st.Chain = append(st.Chain, g)
	return st
}

// =====================================================================
// 核心：哈希、签名、验签
// =====================================================================

func (st snapState) txHash(t tx) string {
	buf := []byte(st.Seed)
	buf = append(buf, []byte(t.ID)...)
	buf = append(buf, []byte(t.From)...)
	buf = append(buf, []byte(t.To)...)
	for _, u := range t.InUTXOs {
		buf = append(buf, []byte(u)...)
	}
	var num [8]byte
	binary.BigEndian.PutUint64(num[:], uint64(t.Value))
	buf = append(buf, num[:]...)
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:6])
}

func (st snapState) txSign(t tx, privateKey string) string {
	hb, _ := hex.DecodeString(t.Hash)
	buf := append([]byte{}, hb...)
	buf = append(buf, []byte(privateKey)...)
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:8])
}

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

// =====================================================================
// 操作
// =====================================================================

// createTx 构造一笔新 tx，标记 owner / variant，但还未上链。
func (st *snapState) createTx(owner, from, to string, in []string, val int64, variant string) (*tx, error) {
	for _, uid := range in {
		u, ok := st.UTXOs[uid]
		if !ok {
			return nil, fmt.Errorf("utxo 不存在: %s", uid)
		}
		if u.Owner != from {
			return nil, fmt.Errorf("utxo %s 所有者 %s ≠ %s", uid, u.Owner, from)
		}
		// 注意：重复花费在创建阶段不报错（教学允许双花尝试）；
		// 在 includeIntoBlock 时再判定。
	}
	st.NextTxID++
	id := fmt.Sprintf("tx%s-%d", owner[:1], st.NextTxID)
	t := &tx{
		ID: id, From: from, To: to, InUTXOs: in, Value: val,
		Owner: owner, Status: "mempool", Variant: variant,
	}
	t.Hash = st.txHash(*t)
	t.Sig = st.txSign(*t, "pk-"+from)
	st.Txs[id] = t
	return t, nil
}

// minerMineBlock 把指定 tx_ids 打包进一块（教学：不模拟 PoW）。
//
//	miner = "honest" → 进入主链
//	miner = "attacker" → hidden=true（私藏），visible=false
func (st *snapState) minerMineBlock(miner string, txIDs []string, hidden bool) (*block, error) {
	if hidden && miner != "attacker" {
		return nil, errors.New("hidden block 必须由 attacker 出")
	}
	// 验证 tx 不冲突且 input 未被该次块内的其它 tx 占用
	used := map[string]bool{}
	for _, id := range txIDs {
		t, ok := st.Txs[id]
		if !ok {
			return nil, fmt.Errorf("tx 不存在: %s", id)
		}
		for _, u := range t.InUTXOs {
			if used[u] {
				return nil, fmt.Errorf("块内冲突：utxo %s 被多笔花费", u)
			}
			used[u] = true
		}
	}
	parent := "GENESIS"
	if !hidden {
		if len(st.Chain) > 0 {
			parent = st.Chain[len(st.Chain)-1].Hash
		}
	} else {
		// 攻击者基于当前主链 tip 分叉
		if len(st.Chain) > 0 {
			parent = st.Chain[len(st.Chain)-1].Hash
		}
	}
	st.Tick++
	height := st.NextBlockHeight
	b := block{
		Height: height, Parent: parent, Miner: miner, Tick: st.Tick,
		Txs: append([]string{}, txIDs...), Visible: !hidden,
	}
	b.Hash = st.blockHash(b)
	if hidden {
		st.Hidden = append(st.Hidden, b)
	} else {
		st.Chain = append(st.Chain, b)
		st.NextBlockHeight++
		st.applyBlockEffects(&b)
	}
	return &b, nil
}

// applyBlockEffects 把块内 tx 的余额变化应用到 Honor / UTXO 集合。
func (st *snapState) applyBlockEffects(b *block) {
	for _, txid := range b.Txs {
		t, ok := st.Txs[txid]
		if !ok {
			continue
		}
		// 检查所有 input 是否还能花费；若任一已被花，则视为该 tx 失败
		canSpend := true
		for _, uid := range t.InUTXOs {
			if u, ok := st.UTXOs[uid]; ok && u.Spent {
				canSpend = false
				break
			}
		}
		if !canSpend {
			t.Status = "dropped"
			continue
		}
		// 标记 input 已花
		for _, uid := range t.InUTXOs {
			st.UTXOs[uid].Spent = true
			st.UTXOs[uid].SpentBy = t.ID
		}
		// 创建 output utxo 给 to
		outID := fmt.Sprintf("u-%s-%d", t.To, len(st.UTXOs)+1)
		st.UTXOs[outID] = &utxo{ID: outID, Owner: t.To, Value: t.Value}
		// 余额视图
		st.Honor[t.From] -= t.Value
		st.Honor[t.To] += t.Value
		t.Status = "inblock"
		t.IncludedBlock = b.Height
		if t.To == "merchant" {
			st.MerchantBalance += t.Value
		}
		if t.To == "attacker" && t.Variant != "" {
			st.AttackerStolen += t.Value
		}
	}
}

// publishHidden 攻击者发布私藏块（finney/vector 的关键步骤）。
//
// 规则（教学版简化）：若 hiddenBlock.height == 当前主链 tip.height，则形成同高分叉；
// 攻击者通过"再多挖 1 块或同高度 hidden 块更早 timestamp"假设其更优 → 主链 reorg
// 替换为 hidden 块（教学：直接接受 hiddenBlock 替换原 tip）。
func (st *snapState) publishHidden() *block {
	if len(st.Hidden) == 0 {
		return nil
	}
	hb := st.Hidden[0]
	st.Hidden = st.Hidden[1:]
	hb.Visible = true
	// 找到对应高度
	if hb.Height <= len(st.Chain)-1 {
		// 同高度替换：先回滚原块再应用 hidden
		old := st.Chain[hb.Height]
		st.rollbackBlock(old)
		st.Chain[hb.Height] = hb
		st.applyBlockEffects(&hb)
	} else {
		// 直接追加
		st.Chain = append(st.Chain, hb)
		st.NextBlockHeight = hb.Height + 1
		st.applyBlockEffects(&hb)
	}
	return &hb
}

// rollbackBlock 撤销块的所有 tx 效应（仅教学版，简化逻辑）。
func (st *snapState) rollbackBlock(b block) {
	for _, txid := range b.Txs {
		t, ok := st.Txs[txid]
		if !ok || t.Status != "inblock" {
			continue
		}
		// 恢复 input
		for _, uid := range t.InUTXOs {
			if u, ok := st.UTXOs[uid]; ok && u.SpentBy == t.ID {
				u.Spent = false
				u.SpentBy = ""
			}
		}
		// 删除 output utxo（按 owner+value 匹配最近一个未 spent）
		for k, u := range st.UTXOs {
			if u.Owner == t.To && u.Value == t.Value && !u.Spent {
				delete(st.UTXOs, k)
				break
			}
		}
		// 余额回滚
		st.Honor[t.From] += t.Value
		st.Honor[t.To] -= t.Value
		if t.To == "merchant" {
			st.MerchantBalance -= t.Value
		}
		if t.To == "attacker" && t.Variant != "" {
			st.AttackerStolen -= t.Value
		}
		t.Status = "dropped"
	}
}

// runRaceAttack 演示 race attack：同 input 两笔 tx，最终只一笔上链。
func (st *snapState) runRaceAttack(value int64) attackAttempt {
	att := attackAttempt{Variant: variantRace, Tick: st.Tick}
	st.RaceCount++
	// honest 用 u-honest-1 付款 merchant
	tA, _ := st.createTx("honest", "honest_user", "merchant", []string{"u-honest-1"}, value, variantRace)
	// attacker 同时假冒受害者用同一个 utxo 付款给自己（教学：忽略私钥伪造）
	// 由于 honest 私钥不在攻击者手中，本支线建模为：
	//  攻击者先用自己的 utxo 转给自己（伪装相同输入概念）
	tB, _ := st.createTx("attacker", "attacker", "attacker", []string{"u-attacker-1"}, value, variantRace)
	att.VictimTxID = tA.ID
	att.AttackerTxID = tB.ID
	// 商家在 0-confirm 看到 tA 后立即接受
	att.ConfsAtAccept = 0
	// 矿工把 tA 打包（race 攻击：常发生在 mempool 竞速；教学只展示一种结果）
	_, _ = st.minerMineBlock("honest", []string{tA.ID}, false)
	// 之后再尝试把 tB 打包到同一区块（应失败 — 不同 input 不会冲突，但攻击者目的是"商家
	// 在确认前因看到 tB 而拒绝接受 tA"——商家若仅看 tA 已成功，则攻击失败。
	att.Success = false // 当 ConfirmationsReq=0 时商家已被骗收 tA，但 tA 真的上链了，merchant 未损失
	att.Detail = "商家 0-confirm 接收 tA，tA 已上链 → 攻击失败"
	st.FailureCount++
	st.Attempts = append(st.Attempts, att)
	return att
}

// runFinneyAttack 演示 Finney attack：预挖 hidden block 含 tB；商家 0-confirm 接受 tA；发布 hidden → reorg。
func (st *snapState) runFinneyAttack(value int64) attackAttempt {
	att := attackAttempt{Variant: variantFinney, Tick: st.Tick}
	st.FinneyCount++
	// Step1：攻击者本地先制造 tB （input = u-attacker-1, to = attacker）
	tB, _ := st.createTx("attacker", "attacker", "attacker", []string{"u-attacker-1"}, value, variantFinney)
	// Step2：攻击者先挖一个 hidden block 包含 tB
	hb, _ := st.minerMineBlock("attacker", []string{tB.ID}, true)
	// Step3：攻击者用 tA 付款给商家（同 input — 但教学版攻击者私钥已属自己，所以同账号双花）
	// 教学：把"双花"建模为 attacker 在 hidden 里花了 u-attacker-1，但同时又
	// 在公链上用同一 utxo 给 merchant 付款（这是攻击核心）
	tA, _ := st.createTx("attacker", "attacker", "merchant", []string{"u-attacker-1"}, value, variantFinney)
	att.VictimTxID = tA.ID
	att.AttackerTxID = tB.ID
	// Step4：商家以 ConfirmationsReq=0 接受 tA → 假装发货
	confs := 0
	att.ConfsAtAccept = confs
	merchantTookGoods := confs <= st.ConfirmationsReq
	// Step5：tA 也上链（被诚实矿工打包）
	st.minerMineBlock("honest", []string{tA.ID}, false)
	// Step6：攻击者发布 hidden
	if hb != nil {
		st.publishHidden()
	}
	// Step7：判定攻击成功（商家发货但 tA 被回滚 / 攻击者收到自己的钱 + 商品）
	tAAfter := st.Txs[tA.ID]
	if merchantTookGoods && tAAfter != nil && tAAfter.Status == "dropped" {
		att.Success = true
		att.Stolen = value
		att.Detail = fmt.Sprintf("hidden 替换主链 → tA 被丢弃，商家发货损失 %d", value)
		st.SuccessCount++
	} else {
		att.Success = false
		att.Detail = "hidden 未替换成功，tA 仍在链上"
		st.FailureCount++
	}
	st.Attempts = append(st.Attempts, att)
	return att
}

// runVector76Attack 演示 vector76：商家要求 1-confirm；攻击者预挖 1 块（tB）；
// 等到 honest 出块时把 tA 提交进 mempool；honest 把 tA 打包；
// 商家见 1-confirm 后发货；攻击者立即广播 hidden block fork → reorg。
func (st *snapState) runVector76Attack(value int64) attackAttempt {
	att := attackAttempt{Variant: variantVector, Tick: st.Tick}
	st.VectorCount++
	tB, _ := st.createTx("attacker", "attacker", "attacker", []string{"u-attacker-1"}, value, variantVector)
	hb, _ := st.minerMineBlock("attacker", []string{tB.ID}, true)
	tA, _ := st.createTx("attacker", "attacker", "merchant", []string{"u-attacker-1"}, value, variantVector)
	att.VictimTxID = tA.ID
	att.AttackerTxID = tB.ID
	st.minerMineBlock("honest", []string{tA.ID}, false)
	att.ConfsAtAccept = 1 // 商家见 1-confirm 即认账
	merchantTook := att.ConfsAtAccept <= st.ConfirmationsReq
	if hb != nil {
		st.publishHidden()
	}
	tAAfter := st.Txs[tA.ID]
	if merchantTook && tAAfter != nil && tAAfter.Status == "dropped" {
		att.Success = true
		att.Stolen = value
		att.Detail = fmt.Sprintf("1-confirm 也被回滚，商家损失 %d", value)
		st.SuccessCount++
	} else {
		att.Success = false
		att.Detail = "vector76 未触发回滚"
		st.FailureCount++
	}
	st.Attempts = append(st.Attempts, att)
	return att
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
		Tick: fw.MapInt(d, "tick", 0), Seed: fw.MapStr(d, "seed", "lenschain-ds"),
		NextTxID:         fw.MapInt(d, "next_tx", 0),
		NextBlockHeight:  fw.MapInt(d, "next_h", 1),
		MerchantBalance:  int64(fw.MapInt(d, "merchant_bal", 0)),
		AttackerStolen:   int64(fw.MapInt(d, "atk_stolen", 0)),
		ConfirmationsReq: fw.MapInt(d, "confs_req", 0),
		RaceCount:        fw.MapInt(d, "race_cnt", 0),
		FinneyCount:      fw.MapInt(d, "finney_cnt", 0),
		VectorCount:      fw.MapInt(d, "vector_cnt", 0),
		SuccessCount:     fw.MapInt(d, "succ_cnt", 0),
		FailureCount:     fw.MapInt(d, "fail_cnt", 0),
		LastError:        fw.MapStr(d, "last_error", ""),
		UTXOs:            map[string]*utxo{}, Txs: map[string]*tx{},
		Honor: map[string]int64{},
	}
	if uAny, ok := d["utxos"].(map[string]any); ok {
		for id, vAny := range uAny {
			if m, ok := vAny.(map[string]any); ok {
				st.UTXOs[id] = &utxo{ID: id,
					Owner: fw.MapStr(m, "owner", ""), Value: int64(fw.MapInt(m, "value", 0)),
					Spent: fw.MapBool(m, "spent", false), SpentBy: fw.MapStr(m, "spent_by", "")}
			}
		}
	}
	if tAny, ok := d["txs"].(map[string]any); ok {
		for id, vAny := range tAny {
			if m, ok := vAny.(map[string]any); ok {
				t := &tx{ID: id,
					From: fw.MapStr(m, "from", ""), To: fw.MapStr(m, "to", ""),
					Value: int64(fw.MapInt(m, "value", 0)),
					Owner: fw.MapStr(m, "owner", ""), Hash: fw.MapStr(m, "hash", ""),
					Sig: fw.MapStr(m, "sig", ""), Status: fw.MapStr(m, "status", ""),
					IncludedBlock: fw.MapInt(m, "block", 0), Variant: fw.MapStr(m, "variant", ""),
				}
				if inAny, ok := m["in"].([]any); ok {
					for _, x := range inAny {
						if s, ok := x.(string); ok {
							t.InUTXOs = append(t.InUTXOs, s)
						}
					}
				}
				st.Txs[id] = t
			}
		}
	}
	if cAny, ok := d["chain"].([]any); ok {
		for _, x := range cAny {
			if m, ok := x.(map[string]any); ok {
				st.Chain = append(st.Chain, decodeBlock(m))
			}
		}
	}
	if hAny, ok := d["hidden"].([]any); ok {
		for _, x := range hAny {
			if m, ok := x.(map[string]any); ok {
				st.Hidden = append(st.Hidden, decodeBlock(m))
			}
		}
	}
	if hAny, ok := d["honor"].(map[string]any); ok {
		for k, v := range hAny {
			st.Honor[k] = int64(intFromAny(v))
		}
	}
	if len(st.Honor) == 0 {
		st.Honor = map[string]int64{"honest_user": 1000, "attacker": 1000, "merchant": 0}
	}
	if aAny, ok := d["attempts"].([]any); ok {
		for _, x := range aAny {
			if m, ok := x.(map[string]any); ok {
				st.Attempts = append(st.Attempts, attackAttempt{
					Variant: fw.MapStr(m, "variant", ""), Tick: fw.MapInt(m, "tick", 0),
					VictimTxID: fw.MapStr(m, "victim", ""), AttackerTxID: fw.MapStr(m, "atk", ""),
					Success:       fw.MapBool(m, "success", false),
					Detail:        fw.MapStr(m, "detail", ""),
					Stolen:        int64(fw.MapInt(m, "stolen", 0)),
					ConfsAtAccept: fw.MapInt(m, "confs", 0),
				})
			}
		}
	}
	if len(st.UTXOs) == 0 {
		return defaultSnapState()
	}
	return st
}

func decodeBlock(m map[string]any) block {
	b := block{
		Height: fw.MapInt(m, "h", 0), Parent: fw.MapStr(m, "parent", ""),
		Hash: fw.MapStr(m, "hash", ""), Miner: fw.MapStr(m, "miner", ""),
		Tick: fw.MapInt(m, "tick", 0), Visible: fw.MapBool(m, "visible", false),
	}
	if txAny, ok := m["txs"].([]any); ok {
		for _, x := range txAny {
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
		"h": b.Height, "parent": b.Parent, "hash": b.Hash,
		"miner": b.Miner, "tick": b.Tick, "visible": b.Visible, "txs": txs,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["seed"] = st.Seed
	s.Data["next_tx"] = st.NextTxID
	s.Data["next_h"] = st.NextBlockHeight
	s.Data["merchant_bal"] = int(st.MerchantBalance)
	s.Data["atk_stolen"] = int(st.AttackerStolen)
	s.Data["confs_req"] = st.ConfirmationsReq
	s.Data["race_cnt"] = st.RaceCount
	s.Data["finney_cnt"] = st.FinneyCount
	s.Data["vector_cnt"] = st.VectorCount
	s.Data["succ_cnt"] = st.SuccessCount
	s.Data["fail_cnt"] = st.FailureCount
	s.Data["last_error"] = st.LastError
	uAny := map[string]any{}
	for id, u := range st.UTXOs {
		uAny[id] = map[string]any{"owner": u.Owner, "value": int(u.Value),
			"spent": u.Spent, "spent_by": u.SpentBy}
	}
	s.Data["utxos"] = uAny
	tAny := map[string]any{}
	for id, t := range st.Txs {
		ins := make([]any, len(t.InUTXOs))
		for i, x := range t.InUTXOs {
			ins[i] = x
		}
		tAny[id] = map[string]any{"from": t.From, "to": t.To, "value": int(t.Value),
			"owner": t.Owner, "hash": t.Hash, "sig": t.Sig, "status": t.Status,
			"block": t.IncludedBlock, "variant": t.Variant, "in": ins}
	}
	s.Data["txs"] = tAny
	cAny := make([]any, len(st.Chain))
	for i, b := range st.Chain {
		cAny[i] = encodeBlock(b)
	}
	s.Data["chain"] = cAny
	hAny := make([]any, len(st.Hidden))
	for i, b := range st.Hidden {
		hAny[i] = encodeBlock(b)
	}
	s.Data["hidden"] = hAny
	honAny := map[string]any{}
	for k, v := range st.Honor {
		honAny[k] = int(v)
	}
	s.Data["honor"] = honAny
	aAny := make([]any, len(st.Attempts))
	for i, a := range st.Attempts {
		aAny[i] = map[string]any{"variant": a.Variant, "tick": a.Tick,
			"victim": a.VictimTxID, "atk": a.AttackerTxID,
			"success": a.Success, "detail": a.Detail,
			"stolen": int(a.Stolen), "confs": a.ConfsAtAccept}
	}
	s.Data["attempts"] = aAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "双花攻击",
		Description:         "演示 race / Finney / vector76 三种双花变体；UTXO 模型 + 商家确认数策略",
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
			"attack.double_spend.attacker_stolen",
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
				ActionCode: "set_confirmations", Label: "设置商家所需确认数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "confs", Type: fw.FieldNumber, Label: "ConfirmationsReq", Required: true, Default: 0, Min: 0, Max: 6, Step: 1},
				},
			},
			{
				ActionCode: "race_attack", Label: "Race attack",
				Description:   "0-confirm 商家场景：mempool 双花竞速",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "value", Type: fw.FieldNumber, Label: "支付金额", Required: true, Default: 200, Min: 1, Step: 50},
				},
				WritesOwnedFields: []string{"attack.double_spend.attacker_stolen"},
			},
			{
				ActionCode: "finney_attack", Label: "Finney attack",
				Description:   "攻击者预挖 hidden block，0-confirm 商家收 tA → 发布 hidden 回滚",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "value", Type: fw.FieldNumber, Label: "支付金额", Required: true, Default: 200, Min: 1, Step: 50},
				},
				WritesOwnedFields: []string{"attack.double_spend.attacker_stolen"},
				LinkOwnerFields:   []string{"attack.double_spend.attacker_stolen"},
			},
			{
				ActionCode: "vector76_attack", Label: "Vector76 attack",
				Description:   "1-confirm 商家场景：分叉 race",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "value", Type: fw.FieldNumber, Label: "支付金额", Required: true, Default: 200, Min: 1, Step: 50},
				},
				WritesOwnedFields: []string{"attack.double_spend.attacker_stolen"},
				LinkOwnerFields:   []string{"attack.double_spend.attacker_stolen"},
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
	env := buildEnvelope(st, "init", "双花场景：UTXO 初始化（honest=1000, attacker=1000）", true)
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
	case "set_confirmations":
		st.ConfirmationsReq = fw.MapInt(in.Params, "confs", 0)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_confirmations",
			fmt.Sprintf("商家所需确认数 = %d", st.ConfirmationsReq), false)
		return out, nil

	case "race_attack":
		v := int64(fw.MapInt(in.Params, "value", 200))
		att := st.runRaceAttack(v)
		saveState(state, st)
		summary := fmt.Sprintf("Race: %v %s", att.Success, att.Detail)
		out.Render = buildEnvelope(st, "race_attack", summary, false)
		appendAttackMicroSteps(&out.Render, variantRace, att.Success)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "finney_attack":
		v := int64(fw.MapInt(in.Params, "value", 200))
		att := st.runFinneyAttack(v)
		saveState(state, st)
		summary := fmt.Sprintf("Finney: %v %s", att.Success, att.Detail)
		out.Render = buildEnvelope(st, "finney_attack", summary, false)
		appendAttackMicroSteps(&out.Render, variantFinney, att.Success)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "vector76_attack":
		v := int64(fw.MapInt(in.Params, "value", 200))
		att := st.runVector76Attack(v)
		saveState(state, st)
		summary := fmt.Sprintf("Vector76: %v %s", att.Success, att.Detail)
		out.Render = buildEnvelope(st, "vector76_attack", summary, false)
		appendAttackMicroSteps(&out.Render, variantVector, att.Success)
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

	// 1) 三方节点
	prims = append(prims, fw.PrimNodeAt("p-honest", fmt.Sprintf("honest_user\nbal=%d", st.Honor["honest_user"]), "active", "user", 0.15, 0.3, 1.3))
	prims = append(prims, fw.PrimNodeAt("p-merchant", fmt.Sprintf("merchant\nbal=%d", st.Honor["merchant"]), "active", "merchant", 0.85, 0.3, 1.3))
	prims = append(prims, fw.PrimNodeAt("p-attacker", fmt.Sprintf("attacker\nbal=%d\nstolen=%d", st.Honor["attacker"], st.AttackerStolen), "active", "attacker", 0.5, 0.7, 1.3))

	// 2) 主链（visible）+ hidden
	cIDs := []string{}
	for _, b := range st.Chain {
		cIDs = append(cIDs, "blk-"+b.Hash)
	}
	prims = append(prims, fw.PrimStack("chain-stack", cIDs, "horizontal"))
	for _, b := range st.Chain {
		role := "block-honest"
		if b.Miner == "attacker" {
			role = "block-attacker"
		}
		label := fmt.Sprintf("#%d\n%s\n%s\ntx=%d", b.Height, b.Miner, b.Hash, len(b.Txs))
		prims = append(prims, fw.PrimNode("blk-"+b.Hash, label, "active", role))
	}
	for i := 0; i+1 < len(st.Chain); i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ce-%d", i),
			"blk-"+st.Chain[i].Hash, "blk-"+st.Chain[i+1].Hash, "solid", "flow"))
	}
	for _, b := range st.Hidden {
		label := fmt.Sprintf("HIDDEN\n#%d\n%s\ntx=%d", b.Height, b.Hash, len(b.Txs))
		prims = append(prims, fw.PrimNode("h-"+b.Hash, label, "warning", "block-hidden"))
	}

	// 3) UTXO 节点（环形布局）
	uIDs := []string{}
	for k := range st.UTXOs {
		uIDs = append(uIDs, k)
	}
	sort.Strings(uIDs)
	if len(uIDs) > 0 {
		ringNodeIDs := make([]string, len(uIDs))
		for i, id := range uIDs {
			ringNodeIDs[i] = "u-" + id
		}
		prims = append(prims, fw.PrimRingLayout("utxo-ring", ringNodeIDs))
	}
	for _, id := range uIDs {
		u := st.UTXOs[id]
		role := "utxo-active"
		status := "active"
		if u.Spent {
			role = "utxo-spent"
			status = "normal"
		}
		label := fmt.Sprintf("%s\n%s\n%d", u.ID, u.Owner, u.Value)
		prims = append(prims, fw.PrimNode("u-"+id, label, status, role))
	}

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-spend",
		`\text{double-spend} \iff \exists\ tx_A, tx_B:\ tx_A.\text{in} \cap tx_B.\text{in} \neq \emptyset \land\ tx_A \neq tx_B`, false))
	prims = append(prims, fw.PrimMathFormula("formula-confirm",
		`\text{merchant accepts iff } \text{conf}(tx_A) \geq \text{ConfirmationsReq}`, false))

	// 5) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("ConfirmationsReq = %d\n主链高度 = %d\nhidden blocks = %d\nutxo 总数 = %d\nattempts = %d (race=%d finney=%d vector=%d)\nsuccess = %d  failure = %d\nattacker_stolen = %d  merchant_balance = %d",
			st.ConfirmationsReq, len(st.Chain)-1, len(st.Hidden),
			len(st.UTXOs), len(st.Attempts), st.RaceCount, st.FinneyCount, st.VectorCount,
			st.SuccessCount, st.FailureCount,
			st.AttackerStolen, st.MerchantBalance),
		"text", nil, 8))

	// 6) Tx 表
	if len(st.Txs) > 0 {
		txLines := []string{"id           variant   from       to         val   in           status    block  hash"}
		ids := []string{}
		for k := range st.Txs {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		for _, id := range ids {
			t := st.Txs[id]
			txLines = append(txLines, fmt.Sprintf("  %-10s %-9s %-10s %-10s %-5d %-12s %-9s %-5d  %s",
				t.ID, t.Variant, t.From, t.To, t.Value,
				strings.Join(t.InUTXOs, ","), t.Status, t.IncludedBlock, t.Hash))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-txs", strings.Join(txLines, "\n"), "text", nil, 14))
	}

	// 7) 攻击记录
	if len(st.Attempts) > 0 {
		aLines := []string{"variant    success  stolen  confs@accept  victim     attacker   detail"}
		startIdx := 0
		if len(st.Attempts) > 12 {
			startIdx = len(st.Attempts) - 12
		}
		for _, a := range st.Attempts[startIdx:] {
			ok := "✗"
			if a.Success {
				ok = "✓"
			}
			aLines = append(aLines, fmt.Sprintf("  %-9s %s   %-6d %-12d %-10s %-10s %s",
				a.Variant, ok, a.Stolen, a.ConfsAtAccept, a.VictimTxID, a.AttackerTxID, a.Detail))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-attempts", strings.Join(aLines, "\n"), "text", nil, 14))
	}

	// 8) 进度条
	prims = append(prims, fw.PrimBar("bar-stolen", float64(st.AttackerStolen), 0, "danger", "Attacker Stolen"))
	prims = append(prims, fw.PrimBar("bar-success", float64(st.SuccessCount), 0, "danger", "Attack Success"))
	prims = append(prims, fw.PrimBar("bar-failure", float64(st.FailureCount), 0, "success", "Attack Failure"))

	// 9) 动效
	if st.SuccessCount > 0 {
		prims = append(prims, fw.PrimShake("shake-suc", "bar-stolen", 0.5, 800))
		prims = append(prims, fw.PrimBurst("burst-suc", "bar-stolen", "danger", int64(st.AttackerStolen), 700))
	}
	if len(st.Hidden) > 0 {
		prims = append(prims, fw.PrimPulse("pulse-hidden", "p-attacker", "warning", 1500))
	}

	// 10) 联动
	prims = append(prims, fw.PrimLinkIndicator("link-pow", linkGroupPoWAttack, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Double-Spend 错误", st.LastError, "scene", "请检查参数", true))
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
		"chain_len":         len(st.Chain) - 1,
		"hidden_blocks":     len(st.Hidden),
		"utxo_count":        len(st.UTXOs),
		"attempts":          len(st.Attempts),
		"race_count":        st.RaceCount,
		"finney_count":      st.FinneyCount,
		"vector_count":      st.VectorCount,
		"success_count":     st.SuccessCount,
		"failure_count":     st.FailureCount,
		"attacker_stolen":   st.AttackerStolen,
		"merchant_balance":  st.MerchantBalance,
		"confirmations_req": st.ConfirmationsReq,
		"tick":              st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendAttackMicroSteps(env *fw.RenderEnvelope, variant string, success bool) {
	tail := "✗ 攻击失败：商家已安全确认"
	if success {
		tail = "⚠ 攻击成功：商家发货后被回滚"
	}
	switch variant {
	case variantRace:
		env.MicroSteps = []fw.MicroStep{
			{ID: "r-1", Label: "honest 创建 tA → mempool", DurationMs: 400, HighlightIDs: []string{"cb-txs", "p-honest"}},
			{ID: "r-2", Label: "attacker 创建 tB（同 input 概念）", DurationMs: 400, HighlightIDs: []string{"cb-txs", "p-attacker"}},
			{ID: "r-3", Label: "矿工竞速打包 → 仅一笔上链", DurationMs: 500, HighlightIDs: []string{"chain-stack", "formula-spend"}, IsLinkTrigger: true},
		}
	case variantFinney:
		env.MicroSteps = []fw.MicroStep{
			{ID: "f-1", Label: "attacker 预挖 hidden block（含 tB）", DurationMs: 400, HighlightIDs: []string{"p-attacker", "h-hidden"}, FirePrimitives: []string{"pulse-hidden"}},
			{ID: "f-2", Label: "attacker 用 tA 付款 merchant", DurationMs: 400, HighlightIDs: []string{"p-merchant", "cb-txs"}},
			{ID: "f-3", Label: "0-confirm 商家发货", DurationMs: 400, HighlightIDs: []string{"formula-confirm"}},
			{ID: "f-4", Label: "attacker 发布 hidden → reorg", DurationMs: 500, HighlightIDs: []string{"chain-stack", "bar-stolen"}, FirePrimitives: []string{"shake-suc", "burst-suc"}},
			{ID: "f-5", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-attempts"}, IsLinkTrigger: true},
		}
	case variantVector:
		env.MicroSteps = []fw.MicroStep{
			{ID: "v-1", Label: "attacker 预挖 hidden + 在 mempool 推 tA", DurationMs: 400, HighlightIDs: []string{"p-attacker", "h-hidden"}, FirePrimitives: []string{"pulse-hidden"}},
			{ID: "v-2", Label: "honest 矿工把 tA 打包，1-confirm", DurationMs: 400, HighlightIDs: []string{"chain-stack", "formula-confirm"}},
			{ID: "v-3", Label: "merchant 发货", DurationMs: 400, HighlightIDs: []string{"p-merchant"}},
			{ID: "v-4", Label: "attacker 发布 hidden 替换 tip → reorg", DurationMs: 500, HighlightIDs: []string{"chain-stack", "bar-stolen"}, FirePrimitives: []string{"shake-suc", "burst-suc"}},
			{ID: "v-5", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-attempts"}, IsLinkTrigger: true},
		}
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
		ID:             "double-spend-attack",
		SourceScene:    sceneCode,
		SourceAction:   "race_attack",
		LinkGroup:      linkGroupPoWAttack,
		ChangedFields:  []string{"attack.double_spend.attacker_stolen"},
		Payload:        map[string]any{"stolen": st.AttackerStolen},
		SourceAnchorID: "double-spend-anchor",
		TargetAnchorID: "pow-chain-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"attack": map[string]any{
			"double_spend": map[string]any{
				"attempts":          len(st.Attempts),
				"success_count":     st.SuccessCount,
				"failure_count":     st.FailureCount,
				"attacker_stolen":   int(st.AttackerStolen),
				"merchant_balance":  int(st.MerchantBalance),
				"confirmations_req": st.ConfirmationsReq,
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
