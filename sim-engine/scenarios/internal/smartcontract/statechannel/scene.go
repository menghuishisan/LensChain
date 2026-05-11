// 模块：sim-engine/scenarios/internal/smartcontract/statechannel
// 文件职责：SC-05 状态通道（State Channel）场景的完整实现。
//
// SSOT 依据：06.md §4.6.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Generalized State Channel 协议（零外部依赖；keccak 复用 keccak256hash 兄弟包）：
//
//   1. 通道生命周期：CLOSED → OPEN → CHALLENGE → SETTLED / FRAUD-CLOSED
//
//   2. open(partyA, partyB, deposit_a, deposit_b)
//      · 双方在链上锁定保证金 → 链上合约持有 (deposit_a + deposit_b)
//      · 通道状态进入 OPEN，nonce = 0
//
//   3. update(off-chain)
//      · 任意一方提议新状态 (balance_a, balance_b, nonce++)
//      · 计算 stateHash = keccak256(channel_id || nonce || balance_a || balance_b)
//      · 双方签名（教学版用确定性"签名 = keccak256(state_hash || private_key)"）
//      · 签名集 = {sigA, sigB}；只有双签状态才被链上接受
//
//   4. close_cooperative(state)
//      · 双方都在线，最高 nonce 双签状态 → 直接结算 → SETTLED
//
//   5. close_unilateral(state)（"乐观提交"）
//      · 单方把已知最新状态发到链上 → 进入 CHALLENGE 状态
//      · 启动挑战期（challenge_period 个 tick 倒计时）
//      · 挑战期结束未被反驳 → 按提交状态结算 → SETTLED
//
//   6. challenge(higher_nonce_state)
//      · 在挑战期内，任一方可提交更高 nonce 的双签状态作为反驳
//      · 链上比较 nonce：新 > 当前 → 替换当前状态，挑战期重置
//
//   7. fraud_proof(invalid_state)
//      · 攻击者提交"自己单签 + 伪造对方签名"的状态
//      · 链上验证签名 → 失败 → 攻击被拒、攻击者保证金扣除（教学版打 50%）
//
//   8. tick_advance
//      · 在 CHALLENGE 状态：challenge_remaining-- ；归零 → 自动 settle
//
//   9. 教学攻击：
//      · old_state_attack：用过期低 nonce 状态 close_unilateral，期望被高 nonce challenge 击败
//      · forged_sig_attack：伪造对方签名（fraud_proof 演示）

package statechannel

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
	sceneCode     = "state-channel"
	schemaVersion = "v1.0.0"
	algorithmType = "generalized-state-channel"

	chPhaseClosed      = "CLOSED"
	chPhaseOpen        = "OPEN"
	chPhaseChallenge   = "CHALLENGE"
	chPhaseSettled     = "SETTLED"
	chPhaseFraudClosed = "FRAUD-CLOSED"

	defaultChallengePeriod = 10

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "contract.channel"
)

// =====================================================================
// 数据结构
// =====================================================================

type party struct {
	ID         string
	Address    string
	PrivateKey string // 教学版"私钥"：仅用于确定性签名
	OnChain    int64  // 链上余额（未锁入通道）
}

// chState 一份链下状态。
type chState struct {
	Nonce     uint64
	BalanceA  int64
	BalanceB  int64
	StateHash string // hex
	SigA      string
	SigB      string
}

// channel 状态通道。
type channel struct {
	ID              string
	A, B            string
	DepositA        int64 // A 的初始充值
	DepositB        int64 // B 的初始充值
	Locked          int64 // 链上合约持有 (=DepositA+DepositB；通道开启时锁定)
	Phase           string
	Nonce           uint64
	OffChainState   chState // 双方持有的最新双签状态
	OnChainState    chState // 链上当前生效状态（CHALLENGE / SETTLED 时填充）
	ChallengePeriod int
	ChallengeRemain int
	OpenedAtTick    int
	ClosedAtTick    int
	FraudCount      int
	OldStateBlocks  int // 旧状态被挑战拦截次数
	History         []chEvent
	SettlementA     int64
	SettlementB     int64
}

type chEvent struct {
	Tick   int
	Kind   string
	Detail string
	OK     bool
}

type snapState struct {
	Parties         map[string]*party
	Channels        map[string]*channel
	Tick            int
	NextChannel     int
	GlobalFrauds    int
	GlobalOldBlocks int
	LastError       string
}

func defaultSnapState() snapState {
	st := snapState{
		Parties:  map[string]*party{},
		Channels: map[string]*channel{},
	}
	st.Parties["A"] = &party{ID: "A", Address: "0xAAA", PrivateKey: "secret-A", OnChain: 1000}
	st.Parties["B"] = &party{ID: "B", Address: "0xBBB", PrivateKey: "secret-B", OnChain: 1000}
	return st
}

// =====================================================================
// 核心算法
// =====================================================================

// computeStateHash keccak256(channel_id || nonce || balanceA || balanceB)
func computeStateHash(channelID string, nonce uint64, balA, balB int64) string {
	buf := []byte(channelID)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], nonce)
	buf = append(buf, b[:]...)
	binary.BigEndian.PutUint64(b[:], uint64(balA))
	buf = append(buf, b[:]...)
	binary.BigEndian.PutUint64(b[:], uint64(balB))
	buf = append(buf, b[:]...)
	h := keccak256hash.Sum256(buf)
	return hex.EncodeToString(h[:])
}

// sign 教学版签名：sig = keccak256(stateHash || privateKey)
func sign(stateHash, privateKey string) string {
	hashBytes, _ := hex.DecodeString(stateHash)
	buf := append([]byte{}, hashBytes...)
	buf = append(buf, []byte(privateKey)...)
	d := keccak256hash.Sum256(buf)
	return hex.EncodeToString(d[:])
}

// verifySig 教学版签名验证：通过签名者提供的 privateKey 重算签名比对。
// 真实链上会用 ecrecover；这里用确定性 hash 等价检验。
func verifySig(stateHash, expectedPrivateKey, sig string) bool {
	return sign(stateHash, expectedPrivateKey) == sig
}

// signedFully 双方签名都有效。
func signedFully(c *channel, st chState, partyA, partyB *party) bool {
	if !verifySig(st.StateHash, partyA.PrivateKey, st.SigA) {
		return false
	}
	if !verifySig(st.StateHash, partyB.PrivateKey, st.SigB) {
		return false
	}
	return true
}

// open 开通道。
func (st *snapState) open(aID, bID string, dA, dB int64, period int) (*channel, error) {
	a, ok1 := st.Parties[aID]
	b, ok2 := st.Parties[bID]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("当事方不存在: %s/%s", aID, bID)
	}
	if a.OnChain < dA || b.OnChain < dB {
		return nil, errors.New("一方链上余额不足以充值")
	}
	if dA <= 0 || dB <= 0 {
		return nil, errors.New("deposit 必须 > 0")
	}
	st.NextChannel++
	id := fmt.Sprintf("ch-%d", st.NextChannel)
	a.OnChain -= dA
	b.OnChain -= dB
	c := &channel{
		ID: id, A: aID, B: bID,
		DepositA: dA, DepositB: dB,
		Locked: dA + dB, Phase: chPhaseOpen,
		Nonce: 0,
		OffChainState: chState{
			Nonce: 0, BalanceA: dA, BalanceB: dB,
			StateHash: computeStateHash(id, 0, dA, dB),
		},
		ChallengePeriod: period, OpenedAtTick: st.Tick,
	}
	// 初始 nonce=0 状态由两方都默认认可（双签）
	c.OffChainState.SigA = sign(c.OffChainState.StateHash, a.PrivateKey)
	c.OffChainState.SigB = sign(c.OffChainState.StateHash, b.PrivateKey)
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "open",
		Detail: fmt.Sprintf("%s+%s 充值 %d/%d 锁仓 %d", aID, bID, dA, dB, c.Locked), OK: true})
	st.Channels[id] = c
	return c, nil
}

// proposeOffChain 链下双签状态更新。
func (st *snapState) proposeOffChain(channelID string, balA, balB int64) (*chState, error) {
	c, ok := st.Channels[channelID]
	if !ok {
		return nil, fmt.Errorf("通道不存在: %s", channelID)
	}
	if c.Phase != chPhaseOpen {
		return nil, fmt.Errorf("通道阶段 %s 不允许更新", c.Phase)
	}
	if balA+balB != c.Locked {
		return nil, fmt.Errorf("balance 守恒失败: %d+%d != %d", balA, balB, c.Locked)
	}
	if balA < 0 || balB < 0 {
		return nil, errors.New("余额不能为负")
	}
	a := st.Parties[c.A]
	b := st.Parties[c.B]
	c.Nonce++
	hash := computeStateHash(c.ID, c.Nonce, balA, balB)
	newState := chState{
		Nonce: c.Nonce, BalanceA: balA, BalanceB: balB,
		StateHash: hash,
		SigA:      sign(hash, a.PrivateKey),
		SigB:      sign(hash, b.PrivateKey),
	}
	c.OffChainState = newState
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "off-chain-update",
		Detail: fmt.Sprintf("nonce=%d bal=(%d,%d)", c.Nonce, balA, balB), OK: true})
	return &newState, nil
}

// closeCooperative 协作关闭：用最新双签状态直接结算。
func (st *snapState) closeCooperative(channelID string) error {
	c, ok := st.Channels[channelID]
	if !ok {
		return fmt.Errorf("通道不存在: %s", channelID)
	}
	if c.Phase != chPhaseOpen {
		return fmt.Errorf("通道阶段 %s 不允许 close", c.Phase)
	}
	a := st.Parties[c.A]
	b := st.Parties[c.B]
	if !signedFully(c, c.OffChainState, a, b) {
		return errors.New("最终状态签名校验失败")
	}
	st.settle(c, c.OffChainState, false)
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "close-coop",
		Detail: fmt.Sprintf("nonce=%d 结算 (%d,%d)", c.OffChainState.Nonce, c.OffChainState.BalanceA, c.OffChainState.BalanceB), OK: true})
	return nil
}

// closeUnilateral 单方把已知状态发到链上，进入 CHALLENGE。
// useStateNonce: 选择哪个 nonce 的状态（用于演示提交旧状态攻击）；-1 表示用最新。
func (st *snapState) closeUnilateral(channelID, submitterID string, useNonce int64) error {
	c, ok := st.Channels[channelID]
	if !ok {
		return fmt.Errorf("通道不存在: %s", channelID)
	}
	if c.Phase != chPhaseOpen {
		return fmt.Errorf("通道阶段 %s 不允许 close_unilateral", c.Phase)
	}
	state := c.OffChainState
	if useNonce >= 0 && uint64(useNonce) != state.Nonce {
		// 教学：人为构造一个旧状态版本（复用历史中可见的余额；这里简化为
		// "回退到 nonce=useNonce 时余额各占一半"）
		balA := c.DepositA
		balB := c.DepositB
		// 模拟旧状态：用 useNonce 重算 hash + 双签
		state = chState{
			Nonce: uint64(useNonce), BalanceA: balA, BalanceB: balB,
			StateHash: computeStateHash(c.ID, uint64(useNonce), balA, balB),
		}
		state.SigA = sign(state.StateHash, st.Parties[c.A].PrivateKey)
		state.SigB = sign(state.StateHash, st.Parties[c.B].PrivateKey)
	}
	a := st.Parties[c.A]
	b := st.Parties[c.B]
	if !signedFully(c, state, a, b) {
		return errors.New("提交状态签名校验失败")
	}
	c.OnChainState = state
	c.Phase = chPhaseChallenge
	c.ChallengeRemain = c.ChallengePeriod
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "close-unilateral",
		Detail: fmt.Sprintf("由 %s 提交 nonce=%d，挑战期=%d tick", submitterID, state.Nonce, c.ChallengePeriod), OK: true})
	return nil
}

// challenge 在挑战期内，用更高 nonce 的双签状态反驳。
func (st *snapState) challenge(channelID, challengerID string, balA, balB int64, useNonce uint64) error {
	c, ok := st.Channels[channelID]
	if !ok {
		return fmt.Errorf("通道不存在: %s", channelID)
	}
	if c.Phase != chPhaseChallenge {
		return fmt.Errorf("通道阶段 %s 不在挑战期", c.Phase)
	}
	if useNonce <= c.OnChainState.Nonce {
		c.OldStateBlocks++
		st.GlobalOldBlocks++
		c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "challenge",
			Detail: fmt.Sprintf("✗ 提交 nonce=%d ≤ 当前 %d 被拒", useNonce, c.OnChainState.Nonce), OK: false})
		return fmt.Errorf("nonce %d ≤ 当前 %d", useNonce, c.OnChainState.Nonce)
	}
	if balA+balB != c.Locked {
		return errors.New("balance 守恒失败")
	}
	hash := computeStateHash(c.ID, useNonce, balA, balB)
	a := st.Parties[c.A]
	b := st.Parties[c.B]
	newState := chState{
		Nonce: useNonce, BalanceA: balA, BalanceB: balB, StateHash: hash,
		SigA: sign(hash, a.PrivateKey), SigB: sign(hash, b.PrivateKey),
	}
	if !signedFully(c, newState, a, b) {
		return errors.New("挑战状态签名校验失败")
	}
	c.OnChainState = newState
	if newState.Nonce > c.Nonce {
		c.Nonce = newState.Nonce
		c.OffChainState = newState
	}
	c.ChallengeRemain = c.ChallengePeriod // 重置挑战期
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "challenge",
		Detail: fmt.Sprintf("✓ %s 用 nonce=%d 反驳，挑战期重置", challengerID, useNonce), OK: true})
	return nil
}

// fraudProof 提交伪造签名状态 → 应被链上拒绝并惩罚。
func (st *snapState) fraudProof(channelID, attackerID string, balA, balB int64) error {
	c, ok := st.Channels[channelID]
	if !ok {
		return fmt.Errorf("通道不存在: %s", channelID)
	}
	if c.Phase != chPhaseOpen && c.Phase != chPhaseChallenge {
		return fmt.Errorf("通道阶段 %s 不允许 fraud_proof", c.Phase)
	}
	hash := computeStateHash(c.ID, c.Nonce+10, balA, balB)
	// 伪造对方签名（用攻击者自己的 private_key 假装对方签）
	attackerKey := st.Parties[attackerID].PrivateKey
	bogus := chState{
		Nonce: c.Nonce + 10, BalanceA: balA, BalanceB: balB, StateHash: hash,
		SigA: sign(hash, attackerKey),
		SigB: sign(hash, attackerKey),
	}
	a := st.Parties[c.A]
	b := st.Parties[c.B]
	if signedFully(c, bogus, a, b) {
		// 教学版：理论上不可能（除非 attackerKey 同时是 a/b 私钥）
		c.OnChainState = bogus
		c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "fraud-proof",
			Detail: "⚠ 伪造签名通过（不应该）", OK: false})
		return nil
	}
	// 拒绝、惩罚
	c.FraudCount++
	st.GlobalFrauds++
	penalty := int64(0)
	if attackerID == c.A {
		penalty = c.DepositA / 2
		c.DepositA -= penalty
		c.Locked -= penalty
		// 惩罚 → 划给对方
		st.Parties[c.B].OnChain += penalty
	} else if attackerID == c.B {
		penalty = c.DepositB / 2
		c.DepositB -= penalty
		c.Locked -= penalty
		st.Parties[c.A].OnChain += penalty
	}
	c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "fraud-proof",
		Detail: fmt.Sprintf("✓ 伪造签名被拒，%s 被罚 %d", attackerID, penalty), OK: true})
	return nil
}

// tick 推进时间：CHALLENGE 中每 tick 倒计时 1。归零 → settle。
func (st *snapState) advance() {
	st.Tick++
	for _, c := range st.Channels {
		if c.Phase != chPhaseChallenge {
			continue
		}
		c.ChallengeRemain--
		if c.ChallengeRemain <= 0 {
			st.settle(c, c.OnChainState, false)
			c.History = append(c.History, chEvent{Tick: st.Tick, Kind: "auto-settle",
				Detail: fmt.Sprintf("挑战期结束，按 nonce=%d 结算", c.OnChainState.Nonce), OK: true})
		}
	}
}

// settle 结算：把锁定金按状态分给双方。
func (st *snapState) settle(c *channel, finalState chState, fraud bool) {
	c.SettlementA = finalState.BalanceA
	c.SettlementB = finalState.BalanceB
	st.Parties[c.A].OnChain += c.SettlementA
	st.Parties[c.B].OnChain += c.SettlementB
	c.Locked = 0
	if fraud {
		c.Phase = chPhaseFraudClosed
	} else {
		c.Phase = chPhaseSettled
	}
	c.ClosedAtTick = st.Tick
}

// =====================================================================
// 持久化（关键字段；省略 OnChainState/OffChainState 的多余字段，按需重算 hash）
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		Parties:         map[string]*party{},
		Channels:        map[string]*channel{},
		Tick:            fw.MapInt(d, "tick", 0),
		NextChannel:     fw.MapInt(d, "next_ch", 0),
		GlobalFrauds:    fw.MapInt(d, "global_frauds", 0),
		GlobalOldBlocks: fw.MapInt(d, "global_old", 0),
		LastError:       fw.MapStr(d, "last_error", ""),
	}
	if pAny, ok := d["parties"].(map[string]any); ok {
		for id, vAny := range pAny {
			if pm, ok := vAny.(map[string]any); ok {
				st.Parties[id] = &party{
					ID: id, Address: fw.MapStr(pm, "addr", ""),
					PrivateKey: fw.MapStr(pm, "pk", ""),
					OnChain:    int64(fw.MapInt(pm, "on_chain", 0)),
				}
			}
		}
	}
	if len(st.Parties) == 0 {
		return defaultSnapState()
	}
	if cAny, ok := d["channels"].(map[string]any); ok {
		for id, vAny := range cAny {
			if cm, ok := vAny.(map[string]any); ok {
				c := &channel{
					ID: id, A: fw.MapStr(cm, "a", ""), B: fw.MapStr(cm, "b", ""),
					DepositA:        int64(fw.MapInt(cm, "dep_a", 0)),
					DepositB:        int64(fw.MapInt(cm, "dep_b", 0)),
					Locked:          int64(fw.MapInt(cm, "locked", 0)),
					Phase:           fw.MapStr(cm, "phase", chPhaseClosed),
					Nonce:           uint64(fw.MapInt(cm, "nonce", 0)),
					ChallengePeriod: fw.MapInt(cm, "period", defaultChallengePeriod),
					ChallengeRemain: fw.MapInt(cm, "remain", 0),
					OpenedAtTick:    fw.MapInt(cm, "opened", 0),
					ClosedAtTick:    fw.MapInt(cm, "closed", 0),
					FraudCount:      fw.MapInt(cm, "frauds", 0),
					OldStateBlocks:  fw.MapInt(cm, "old_blocks", 0),
					SettlementA:     int64(fw.MapInt(cm, "settle_a", 0)),
					SettlementB:     int64(fw.MapInt(cm, "settle_b", 0)),
				}
				if osAny, ok := cm["off"].(map[string]any); ok {
					c.OffChainState = decodeChState(osAny)
				}
				if onAny, ok := cm["on"].(map[string]any); ok {
					c.OnChainState = decodeChState(onAny)
				}
				if hAny, ok := cm["history"].([]any); ok {
					for _, eAny := range hAny {
						if em, ok := eAny.(map[string]any); ok {
							c.History = append(c.History, chEvent{
								Tick: fw.MapInt(em, "t", 0), Kind: fw.MapStr(em, "k", ""),
								Detail: fw.MapStr(em, "d", ""), OK: fw.MapBool(em, "ok", false),
							})
						}
					}
				}
				st.Channels[id] = c
			}
		}
	}
	return st
}

func decodeChState(m map[string]any) chState {
	return chState{
		Nonce:     uint64(fw.MapInt(m, "n", 0)),
		BalanceA:  int64(fw.MapInt(m, "ba", 0)),
		BalanceB:  int64(fw.MapInt(m, "bb", 0)),
		StateHash: fw.MapStr(m, "h", ""),
		SigA:      fw.MapStr(m, "sa", ""),
		SigB:      fw.MapStr(m, "sb", ""),
	}
}

func encodeChState(s chState) map[string]any {
	return map[string]any{
		"n": int(s.Nonce), "ba": int(s.BalanceA), "bb": int(s.BalanceB),
		"h": s.StateHash, "sa": s.SigA, "sb": s.SigB,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["next_ch"] = st.NextChannel
	s.Data["global_frauds"] = st.GlobalFrauds
	s.Data["global_old"] = st.GlobalOldBlocks
	s.Data["last_error"] = st.LastError
	pAny := map[string]any{}
	for id, p := range st.Parties {
		pAny[id] = map[string]any{"addr": p.Address, "pk": p.PrivateKey, "on_chain": int(p.OnChain)}
	}
	s.Data["parties"] = pAny
	cAny := map[string]any{}
	for id, c := range st.Channels {
		hAny := []any{}
		for _, e := range c.History {
			hAny = append(hAny, map[string]any{"t": e.Tick, "k": e.Kind, "d": e.Detail, "ok": e.OK})
		}
		cAny[id] = map[string]any{
			"a": c.A, "b": c.B, "dep_a": int(c.DepositA), "dep_b": int(c.DepositB),
			"locked": int(c.Locked), "phase": c.Phase, "nonce": int(c.Nonce),
			"period": c.ChallengePeriod, "remain": c.ChallengeRemain,
			"opened": c.OpenedAtTick, "closed": c.ClosedAtTick,
			"frauds": c.FraudCount, "old_blocks": c.OldStateBlocks,
			"settle_a": int(c.SettlementA), "settle_b": int(c.SettlementB),
			"off":     encodeChState(c.OffChainState),
			"on":      encodeChState(c.OnChainState),
			"history": hAny,
		}
	}
	s.Data["channels"] = cAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "状态通道（State Channel）",
		Description:         "演示链下双签状态更新 + 协作关闭 / 单方关闭 + 挑战期 + 旧状态攻击 + 伪造签名 fraud_proof",
		Category:            fw.CategorySmartContract,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupContractSec},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"contract.channel.locked",
			"contract.channel.fraud_count",
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
				ActionCode: "open", Label: "开通道",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "a", Type: fw.FieldEnum, Label: "Party A", Required: true, Default: "A", Options: []any{"A", "B"}},
					{Name: "b", Type: fw.FieldEnum, Label: "Party B", Required: true, Default: "B", Options: []any{"A", "B"}},
					{Name: "deposit_a", Type: fw.FieldNumber, Label: "A 充值", Required: true, Default: 100, Min: 1, Step: 10},
					{Name: "deposit_b", Type: fw.FieldNumber, Label: "B 充值", Required: true, Default: 100, Min: 1, Step: 10},
					{Name: "challenge_period", Type: fw.FieldNumber, Label: "挑战期 (tick)", Required: true, Default: defaultChallengePeriod, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"contract.channel.locked"},
				LinkOwnerFields:   []string{"contract.channel.locked"},
			},
			{
				ActionCode: "off_chain_pay", Label: "链下双签更新",
				Description:   "在通道内调整双方余额（守恒），nonce 自增，双签",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "channel_id", Type: fw.FieldString, Label: "channel_id", Required: true, Default: "ch-1"},
					{Name: "balance_a", Type: fw.FieldNumber, Label: "新 balance_a", Required: true, Default: 70, Min: 0, Step: 10},
					{Name: "balance_b", Type: fw.FieldNumber, Label: "新 balance_b", Required: true, Default: 130, Min: 0, Step: 10},
				},
			},
			{
				ActionCode: "close_cooperative", Label: "协作关闭",
				Description:   "用最新双签状态直接结算",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "channel_id", Type: fw.FieldString, Label: "channel_id", Required: true, Default: "ch-1"},
				},
				WritesOwnedFields: []string{"contract.channel.locked"},
				LinkOwnerFields:   []string{"contract.channel.locked"},
			},
			{
				ActionCode: "close_unilateral", Label: "单方关闭",
				Description:   "提交已知状态进入 CHALLENGE 期；可选 use_nonce 模拟旧状态攻击",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "channel_id", Type: fw.FieldString, Label: "channel_id", Required: true, Default: "ch-1"},
					{Name: "submitter", Type: fw.FieldEnum, Label: "提交者", Required: true, Default: "A", Options: []any{"A", "B"}},
					{Name: "use_nonce", Type: fw.FieldNumber, Label: "use_nonce (-1=最新)", Required: false, Default: -1, Step: 1},
				},
			},
			{
				ActionCode: "challenge", Label: "挑战（更高 nonce）",
				Description:   "在挑战期内提交更高 nonce 的双签状态",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "channel_id", Type: fw.FieldString, Label: "channel_id", Required: true, Default: "ch-1"},
					{Name: "challenger", Type: fw.FieldEnum, Label: "挑战者", Required: true, Default: "B", Options: []any{"A", "B"}},
					{Name: "balance_a", Type: fw.FieldNumber, Label: "balance_a", Required: true, Default: 70, Min: 0, Step: 10},
					{Name: "balance_b", Type: fw.FieldNumber, Label: "balance_b", Required: true, Default: 130, Min: 0, Step: 10},
					{Name: "use_nonce", Type: fw.FieldNumber, Label: "use_nonce", Required: true, Default: 5, Min: 1, Step: 1},
				},
				LinkOwnerFields: []string{"contract.channel.old_blocks"},
			},
			{
				ActionCode: "fraud_proof", Label: "伪造签名攻击",
				Description:   "攻击者用自己的 private_key 伪造对方签名 → 验签失败、攻击者被罚 50%",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "channel_id", Type: fw.FieldString, Label: "channel_id", Required: true, Default: "ch-1"},
					{Name: "attacker", Type: fw.FieldEnum, Label: "attacker", Required: true, Default: "A", Options: []any{"A", "B"}},
					{Name: "balance_a", Type: fw.FieldNumber, Label: "伪造 bal_a", Required: true, Default: 200, Step: 10},
					{Name: "balance_b", Type: fw.FieldNumber, Label: "伪造 bal_b", Required: true, Default: 0, Step: 10},
				},
				WritesOwnedFields: []string{"contract.channel.fraud_count"},
				LinkOwnerFields:   []string{"contract.channel.fraud_count"},
			},
			{
				ActionCode: "tick", Label: "推进 1 tick",
				Description:   "用于挑战期倒计时",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode: "tick_n", Label: "推进 N tick",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "N", Required: true, Default: 5, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_force_revert",
				Label:         "教师强制回滚",
				Description:   "仅教师可用，强制回滚用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师强制回滚"},
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
	st := loadState(state)
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "Channel 场景初始化（A、B 各 1000）", true)
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
	case "open":
		a := fw.MapStr(in.Params, "a", "A")
		b := fw.MapStr(in.Params, "b", "B")
		dA := int64(fw.MapInt(in.Params, "deposit_a", 100))
		dB := int64(fw.MapInt(in.Params, "deposit_b", 100))
		period := fw.MapInt(in.Params, "challenge_period", defaultChallengePeriod)
		c, err := st.open(a, b, dA, dB, period)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "open",
			fmt.Sprintf("✓ 开通道 %s: %s+%s 锁定 %d", c.ID, a, b, c.Locked), false)
		appendOpenMicroSteps(&out.Render, c.ID)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "off_chain_update":
		id := fw.MapStr(in.Params, "channel_id", "ch-1")
		balA := int64(fw.MapInt(in.Params, "balance_a", 0))
		balB := int64(fw.MapInt(in.Params, "balance_b", 0))
		s, err := st.proposeOffChain(id, balA, balB)
		if err != nil {
			saveState(state, st)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "off_chain_update",
			fmt.Sprintf("链下更新 %s nonce=%d (%d,%d)", id, s.Nonce, s.BalanceA, s.BalanceB), false)
		appendOffChainMicroSteps(&out.Render, s)
		return out, nil

	case "close_cooperative":
		id := fw.MapStr(in.Params, "channel_id", "ch-1")
		if err := st.closeCooperative(id); err != nil {
			saveState(state, st)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		c := st.Channels[id]
		out.Render = buildEnvelope(st, "close_cooperative",
			fmt.Sprintf("✓ %s 协作关闭：A=%d, B=%d", id, c.SettlementA, c.SettlementB), false)
		appendSettleMicroSteps(&out.Render, c, "cooperative")
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "close_unilateral":
		id := fw.MapStr(in.Params, "channel_id", "ch-1")
		submitter := fw.MapStr(in.Params, "submitter", "A")
		useNonce := int64(fw.MapInt(in.Params, "use_nonce", -1))
		if err := st.closeUnilateral(id, submitter, useNonce); err != nil {
			saveState(state, st)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		c := st.Channels[id]
		out.Render = buildEnvelope(st, "close_unilateral",
			fmt.Sprintf("⏳ %s 进入挑战期，nonce=%d，剩 %d tick", id, c.OnChainState.Nonce, c.ChallengeRemain), false)
		appendUnilateralMicroSteps(&out.Render, c)
		return out, nil

	case "challenge":
		id := fw.MapStr(in.Params, "channel_id", "ch-1")
		challenger := fw.MapStr(in.Params, "challenger", "B")
		balA := int64(fw.MapInt(in.Params, "balance_a", 0))
		balB := int64(fw.MapInt(in.Params, "balance_b", 0))
		useNonce := uint64(fw.MapInt(in.Params, "use_nonce", 1))
		err := st.challenge(id, challenger, balA, balB, useNonce)
		saveState(state, st)
		c := st.Channels[id]
		summary := fmt.Sprintf("✓ %s 挑战成功，OnChain.nonce=%d", challenger, c.OnChainState.Nonce)
		if err != nil {
			summary = "✗ " + err.Error()
		}
		out.Render = buildEnvelope(st, "challenge", summary, false)
		appendChallengeMicroSteps(&out.Render, c, err == nil)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "fraud_proof":
		id := fw.MapStr(in.Params, "channel_id", "ch-1")
		attacker := fw.MapStr(in.Params, "attacker", "A")
		balA := int64(fw.MapInt(in.Params, "balance_a", 200))
		balB := int64(fw.MapInt(in.Params, "balance_b", 0))
		err := st.fraudProof(id, attacker, balA, balB)
		saveState(state, st)
		summary := fmt.Sprintf("✓ %s 伪造签名被拒，已罚 50%%", attacker)
		if err != nil {
			summary = "✗ " + err.Error()
		}
		out.Render = buildEnvelope(st, "fraud_proof", summary, false)
		appendFraudMicroSteps(&out.Render, attacker, err == nil)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "tick":
		st.advance()
		saveState(state, st)
		out.Render = buildEnvelope(st, "tick", fmt.Sprintf("tick → %d", st.Tick), false)
		appendTickMicroSteps(&out.Render)
		return out, nil

	case "tick_n":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.advance()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "tick_n", fmt.Sprintf("推进 %d tick → %d", n, st.Tick), false)
		appendTickMicroSteps(&out.Render)
		return out, nil

	case "teacher_force_revert":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师强制回滚"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-revert-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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
	prims := make([]fw.Primitive, 0, 50)

	// 1) Party A / Bridge / Party B
	prims = append(prims, fw.PrimNodeAt("party-A",
		fmt.Sprintf("Party A\nonChain=%d\npk=%s", st.Parties["A"].OnChain, st.Parties["A"].PrivateKey),
		"active", "party", 0.15, 0.5, 1.4))
	prims = append(prims, fw.PrimNodeAt("party-B",
		fmt.Sprintf("Party B\nonChain=%d\npk=%s", st.Parties["B"].OnChain, st.Parties["B"].PrivateKey),
		"active", "party", 0.85, 0.5, 1.4))

	// 2) 通道节点（环形）
	chIDs := []string{}
	for id := range st.Channels {
		chIDs = append(chIDs, id)
	}
	sort.Strings(chIDs)

	if len(chIDs) > 0 {
		prims = append(prims, fw.PrimRingLayout("channels-ring", len(chIDs)))
		for _, id := range chIDs {
			c := st.Channels[id]
			role := "channel-" + strings.ToLower(c.Phase)
			label := fmt.Sprintf("%s\n%s\nlocked=%d\nnonce=%d", c.ID, c.Phase, c.Locked, c.Nonce)
			if c.Phase == chPhaseChallenge {
				label += fmt.Sprintf("\nchallenge: %d", c.ChallengeRemain)
			}
			status := "active"
			if c.Phase == chPhaseSettled || c.Phase == chPhaseFraudClosed {
				status = "normal"
			}
			prims = append(prims, fw.PrimNode("ch-"+id, label, status, role))
		}
	}

	// 3) 通道 ↔ 当事方边
	for _, id := range chIDs {
		c := st.Channels[id]
		anim := ""
		if c.Phase == chPhaseOpen || c.Phase == chPhaseChallenge {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge("e-A-"+id, "party-A", "ch-"+id, "solid", anim))
		prims = append(prims, fw.PrimEdge("e-B-"+id, "ch-"+id, "party-B", "solid", anim))
	}

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-state",
		`\text{stateHash} = \mathrm{keccak256}(\text{ch\_id} \,\Vert\, \text{nonce} \,\Vert\, \text{bal}_A \,\Vert\, \text{bal}_B);
		\text{sig}_X = \mathrm{keccak256}(\text{stateHash} \,\Vert\, \text{pk}_X)`, false))
	prims = append(prims, fw.PrimMathFormula("formula-rule",
		`\text{accept iff: } \text{sig}_A,\ \text{sig}_B \text{ valid} \land \text{nonce}_{\text{new}} > \text{nonce}_{\text{on-chain}}`, false))

	// 5) 通道生命周期 phase_progress（针对最新通道）
	phases := []string{chPhaseClosed, chPhaseOpen, chPhaseChallenge, chPhaseSettled}
	curPhaseIdx := 0
	if len(chIDs) > 0 {
		c := st.Channels[chIDs[len(chIDs)-1]]
		for i, p := range phases {
			if p == c.Phase {
				curPhaseIdx = i
			}
		}
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phases, curPhaseIdx, float64(curPhaseIdx)/float64(len(phases)-1)))

	// 6) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d\n通道数 = %d\n伪造签名拒绝 = %d\n旧状态拦截 = %d",
			st.Tick, len(st.Channels), st.GlobalFrauds, st.GlobalOldBlocks),
		"text", nil, 6))

	// 7) 通道详情表
	if len(chIDs) > 0 {
		cLines := []string{"id    phase         nonce  off-chain (a,b)         on-chain (a,b)        locked  challenge  settle(A,B)"}
		for _, id := range chIDs {
			c := st.Channels[id]
			off := fmt.Sprintf("(%d,%d)", c.OffChainState.BalanceA, c.OffChainState.BalanceB)
			on := fmt.Sprintf("(%d,%d)", c.OnChainState.BalanceA, c.OnChainState.BalanceB)
			settle := fmt.Sprintf("(%d,%d)", c.SettlementA, c.SettlementB)
			cLines = append(cLines, fmt.Sprintf("%-5s %-13s %-6d %-22s %-21s %-7d %-10d %s",
				c.ID, c.Phase, c.Nonce, off, on, c.Locked, c.ChallengeRemain, settle))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-channels", strings.Join(cLines, "\n"), "text", nil, 12))
	}

	// 8) 当前选中通道的 stateHash + sigA + sigB（教学：展示 hash 链）
	if len(chIDs) > 0 {
		c := st.Channels[chIDs[len(chIDs)-1]]
		off := c.OffChainState
		hLines := []string{
			fmt.Sprintf("最新通道：%s（%s）", c.ID, c.Phase),
			fmt.Sprintf("  off-chain.nonce  = %d", off.Nonce),
			fmt.Sprintf("  off-chain.balA   = %d  balB = %d", off.BalanceA, off.BalanceB),
			fmt.Sprintf("  off-chain.hash   = 0x%s…", short(off.StateHash, 16)),
			fmt.Sprintf("  off-chain.sigA   = 0x%s…", short(off.SigA, 16)),
			fmt.Sprintf("  off-chain.sigB   = 0x%s…", short(off.SigB, 16)),
		}
		if c.OnChainState.StateHash != "" {
			hLines = append(hLines,
				fmt.Sprintf("  on-chain.nonce   = %d", c.OnChainState.Nonce),
				fmt.Sprintf("  on-chain.balA    = %d  balB = %d", c.OnChainState.BalanceA, c.OnChainState.BalanceB),
				fmt.Sprintf("  on-chain.hash    = 0x%s…", short(c.OnChainState.StateHash, 16)),
			)
		}
		prims = append(prims, fw.PrimCodeBlock("cb-state", strings.Join(hLines, "\n"), "text", nil, 12))
	}

	// 9) 事件日志（合并所有通道的 history 后取最近）
	allEvents := []chEvent{}
	for _, id := range chIDs {
		for _, e := range st.Channels[id].History {
			allEvents = append(allEvents, chEvent{Tick: e.Tick, Kind: id + ":" + e.Kind, Detail: e.Detail, OK: e.OK})
		}
	}
	sort.SliceStable(allEvents, func(a, b int) bool { return allEvents[a].Tick < allEvents[b].Tick })
	if len(allEvents) > 0 {
		eLines := []string{"事件日志：t  ok  kind                 detail"}
		startIdx := 0
		if len(allEvents) > 16 {
			startIdx = len(allEvents) - 16
		}
		for _, e := range allEvents[startIdx:] {
			ok := "✓"
			if !e.OK {
				ok = "✗"
			}
			eLines = append(eLines, fmt.Sprintf("  %-3d %s   %-20s %s", e.Tick, ok, e.Kind, e.Detail))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 18))
	}

	// 10) 进度条 / 计数
	prims = append(prims, fw.PrimBar("bar-fraud", float64(st.GlobalFrauds), 0, "success", "Fraud Sigs Blocked"))
	prims = append(prims, fw.PrimBar("bar-old", float64(st.GlobalOldBlocks), 0, "success", "Old-State Challenges Blocked"))
	if len(chIDs) > 0 {
		c := st.Channels[chIDs[len(chIDs)-1]]
		if c.Phase == chPhaseChallenge {
			prims = append(prims, fw.PrimProgressBar("bar-challenge",
				float64(c.ChallengePeriod-c.ChallengeRemain), float64(c.ChallengePeriod),
				fmt.Sprintf("Challenge progress %d/%d", c.ChallengePeriod-c.ChallengeRemain, c.ChallengePeriod)))
		}
	}

	// 11) 动效
	if len(chIDs) > 0 {
		c := st.Channels[chIDs[len(chIDs)-1]]
		switch c.Phase {
		case chPhaseSettled:
			prims = append(prims, fw.PrimBurst("burst-settle", "ch-"+c.ID, "success", int64(st.Tick), 700))
			prims = append(prims, fw.PrimGlow("glow-settle", "ch-"+c.ID, "success", 0.9))
		case chPhaseFraudClosed:
			prims = append(prims, fw.PrimShake("shake-fraud", "ch-"+c.ID, 0.5, 800))
			prims = append(prims, fw.PrimPulse("pulse-fraud", "ch-"+c.ID, "danger", 1500))
		case chPhaseChallenge:
			prims = append(prims, fw.PrimPulse("pulse-challenge", "ch-"+c.ID, "warning", 1500))
		case chPhaseOpen:
			prims = append(prims, fw.PrimGlow("glow-open", "ch-"+c.ID, "info", 0.7))
		}
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Channel 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"channels":      len(st.Channels),
		"global_frauds": st.GlobalFrauds,
		"global_old":    st.GlobalOldBlocks,
		"tick":          st.Tick,
	}
	totalLocked := int64(0)
	for _, c := range st.Channels {
		totalLocked += c.Locked
	}
	d["locked"] = totalLocked
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendOpenMicroSteps(env *fw.RenderEnvelope, id string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "o-1", Label: "双方在链上锁仓", DurationMs: 400, HighlightIDs: []string{"party-A", "party-B"}},
		{ID: "o-2", Label: fmt.Sprintf("通道 %s OPEN，nonce=0", id), DurationMs: 500, HighlightIDs: []string{"ch-" + id, "phase-progress"}, FirePrimitives: []string{"glow-open"}},
		{ID: "o-3", Label: "初始状态（0,deposit_a,deposit_b）双签", DurationMs: 500, HighlightIDs: []string{"cb-state", "formula-state"}, IsLinkTrigger: true},
	}
}

func appendOffChainMicroSteps(env *fw.RenderEnvelope, s *chState) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "u-1", Label: fmt.Sprintf("提议新状态 nonce=%d (a=%d, b=%d)", s.Nonce, s.BalanceA, s.BalanceB), DurationMs: 400, HighlightIDs: []string{"cb-state"}},
		{ID: "u-2", Label: "计算 stateHash = keccak256(...)", DurationMs: 400, HighlightIDs: []string{"formula-state"}},
		{ID: "u-3", Label: "双方签名 sig_A, sig_B；不上链", DurationMs: 500, HighlightIDs: []string{"cb-channels", "cb-state"}, IsLinkTrigger: true},
	}
}

func appendUnilateralMicroSteps(env *fw.RenderEnvelope, c *channel) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "u1-1", Label: fmt.Sprintf("提交 onChain state nonce=%d", c.OnChainState.Nonce), DurationMs: 400, HighlightIDs: []string{"ch-" + c.ID, "cb-state"}},
		{ID: "u1-2", Label: "链上验证签名", DurationMs: 400, HighlightIDs: []string{"formula-rule"}},
		{ID: "u1-3", Label: fmt.Sprintf("进入 CHALLENGE，倒计时 %d", c.ChallengeRemain), DurationMs: 500, HighlightIDs: []string{"phase-progress", "bar-challenge"}, FirePrimitives: []string{"pulse-challenge"}, IsLinkTrigger: true},
	}
}

func appendChallengeMicroSteps(env *fw.RenderEnvelope, c *channel, ok bool) {
	tail := "✗ 旧 nonce 被拒"
	if ok {
		tail = fmt.Sprintf("✓ 替换 onChain.nonce=%d，挑战期重置", c.OnChainState.Nonce)
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "c-1", Label: "比较 nonce_new vs nonce_on-chain", DurationMs: 400, HighlightIDs: []string{"cb-channels"}},
		{ID: "c-2", Label: "验签 sig_A, sig_B", DurationMs: 400, HighlightIDs: []string{"formula-rule"}},
		{ID: "c-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"phase-progress", "bar-old"}, IsLinkTrigger: true},
	}
}

func appendFraudMicroSteps(env *fw.RenderEnvelope, attacker string, blocked bool) {
	tail := "⚠ 伪造签名通过（不应该）"
	if blocked {
		tail = fmt.Sprintf("✓ 验签失败，%s 罚 50%% 充值", attacker)
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "f-1", Label: attacker + " 用自己的 pk 伪造对方签名", DurationMs: 400, HighlightIDs: []string{"cb-state"}},
		{ID: "f-2", Label: "链上验证 sig_B = keccak256(hash || pk_B)", DurationMs: 500, HighlightIDs: []string{"formula-state", "formula-rule"}},
		{ID: "f-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-fraud"}, FirePrimitives: []string{"shake-fraud", "pulse-fraud"}, IsLinkTrigger: true},
	}
}

func appendSettleMicroSteps(env *fw.RenderEnvelope, c *channel, kind string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: kind + " 关闭：解锁链上保证金", DurationMs: 400, HighlightIDs: []string{"ch-" + c.ID}},
		{ID: "s-2", Label: fmt.Sprintf("结算 A=%d B=%d", c.SettlementA, c.SettlementB), DurationMs: 500, HighlightIDs: []string{"party-A", "party-B"}, FirePrimitives: []string{"glow-settle", "burst-settle"}},
		{ID: "s-3", Label: "通道进入 SETTLED", DurationMs: 400, HighlightIDs: []string{"phase-progress"}, IsLinkTrigger: true},
	}
}

func appendTickMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "t-1", Label: "推进时间", DurationMs: 300, HighlightIDs: []string{"cb-status"}},
		{ID: "t-2", Label: "CHALLENGE 中通道倒计时；归零自动 settle", DurationMs: 500, HighlightIDs: []string{"bar-challenge", "phase-progress"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func channelTotalLocked(st snapState) int64 {
	total := int64(0)
	for _, c := range st.Channels {
		total += c.Locked
	}
	return total
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "channel-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_channel",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"contract.channel.locked", "contract.channel.fraud_count"},
		Payload:        map[string]any{"locked": channelTotalLocked(st), "frauds": st.GlobalFrauds},
		SourceAnchorID: "channel-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "contract.channel.locked", "contract.channel.fraud_count")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	totalLocked := int64(0)
	for _, c := range st.Channels {
		totalLocked += c.Locked
	}
	return map[string]any{
		"contract": map[string]any{
			"channel": map[string]any{
				"channels":    len(st.Channels),
				"locked":      int(totalLocked),
				"fraud_count": st.GlobalFrauds,
				"old_blocks":  st.GlobalOldBlocks,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

