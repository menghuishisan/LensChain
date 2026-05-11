// 模块：sim-engine/scenarios/internal/attacksecurity/integeroverflow
// 文件职责：ATK-05 整数溢出/下溢攻击场景的完整实现。
//
// SSOT 依据：06.md §4.7.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现多位宽无符号整数算术 + 三种检查模式（零外部依赖）。
//
//   1. 位宽（Solidity 风格 unsigned）：
//        uint8  : 模 2^8
//        uint16 : 模 2^16
//        uint32 : 模 2^32
//        uint64 : 模 2^64
//        uint256: 模 2^256（教学版用 [4]uint64 表示，可与 evmexecution 兼容）
//
//   2. 三种检查模式：
//        legacy_unchecked : 静默 wrap-around（Solidity < 0.8.x 默认行为）
//        checked          : 任何 over/under flow → REVERT（Solidity ≥ 0.8.x 默认）
//        unchecked_block  : 用 unchecked { ... } 显式跳过检查
//
//   3. 操作：add / sub / mul / div / pow（小指数）
//
//   4. 攻击演示（典型 ERC-20 漏洞）：
//        · transfer 下溢：balances[from] - amount，其中 amount > balance
//          → uint256 wrap → balance 变巨大 → 攻击者可无限转账
//        · totalSupply 上溢：mint 时不检查，totalSupply + amount 溢出
//        · array index 上溢：length-- 在 length=0 时 wrap 到 max → OOB
//
//   5. 教学指标：
//        · overflowDetected / underflowDetected
//        · checkedRevertCount / unchecked_silentWraps
//        · attackerStolenViaOverflow

package integeroverflow

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "integer-overflow"
	schemaVersion = "v1.0.0"
	algorithmType = "uint-overflow"

	bitsU8   = 8
	bitsU16  = 16
	bitsU32  = 32
	bitsU64  = 64
	bitsU256 = 256

	modeLegacy    = "legacy_unchecked"
	modeChecked   = "checked"
	modeUnchecked = "unchecked_block"

	opAdd = "add"
	opSub = "sub"
	opMul = "mul"
	opDiv = "div"
	opPow = "pow"

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "attack.overflow"
)

// =====================================================================
// uint256（与 evmexecution 同源；本场景独立实现以避免 internal 包跨用）
// =====================================================================

type word256 [4]uint64 // big-endian: [0]=高位 ... [3]=低位

func u256FromUint64(v uint64) word256 { return word256{0, 0, 0, v} }

func u256IsZero(a word256) bool { return a[0] == 0 && a[1] == 0 && a[2] == 0 && a[3] == 0 }

// u256Add 返回 (sum, overflow)。
func u256Add(a, b word256) (word256, bool) {
	var r word256
	var carry uint64
	for i := 3; i >= 0; i-- {
		sum := a[i] + b[i] + carry
		newCarry := uint64(0)
		if sum < a[i] || (carry == 1 && sum == a[i]) {
			newCarry = 1
		}
		r[i] = sum
		carry = newCarry
	}
	return r, carry == 1
}

// u256Sub 返回 (diff, underflow)。
func u256Sub(a, b word256) (word256, bool) {
	var r word256
	var borrow uint64
	for i := 3; i >= 0; i-- {
		bi := b[i] + borrow
		newBorrow := uint64(0)
		if a[i] < bi || (borrow == 1 && b[i] == ^uint64(0)) {
			newBorrow = 1
		}
		r[i] = a[i] - bi
		borrow = newBorrow
	}
	return r, borrow == 1
}

// u256Mul 返回 (low, overflow)，其中 overflow 表示高 256-bit 非 0。
func u256Mul(a, b word256) (word256, bool) {
	var prod [8]uint64
	for i := 3; i >= 0; i-- {
		var carry uint64 = 0
		for j := 3; j >= 0; j-- {
			hi, lo := mul64(a[i], b[j])
			pos := i + j + 1
			sum := prod[pos] + lo + carry
			c := uint64(0)
			if sum < prod[pos] || (carry == 1 && sum == prod[pos]) {
				c = 1
			}
			prod[pos] = sum
			carry = hi + c
		}
		if i >= 0 {
			prod[i] += carry
		}
	}
	low := word256{prod[4], prod[5], prod[6], prod[7]}
	overflow := prod[0] != 0 || prod[1] != 0 || prod[2] != 0 || prod[3] != 0
	return low, overflow
}

func mul64(a, b uint64) (hi, lo uint64) {
	const halfShift = 32
	const halfMask = uint64(0xFFFFFFFF)
	a0, a1 := a&halfMask, a>>halfShift
	b0, b1 := b&halfMask, b>>halfShift
	w0 := a0 * b0
	t := a1*b0 + (w0 >> halfShift)
	w1 := t & halfMask
	w2 := t >> halfShift
	w1 += a0 * b1
	hi = a1*b1 + w2 + (w1 >> halfShift)
	lo = (w1 << halfShift) | (w0 & halfMask)
	return
}

// u256Div 整除（b=0 → (0, true) 标记错误）。教学：这里使用位级长除法。
func u256Div(a, b word256) (word256, bool) {
	if u256IsZero(b) {
		return word256{}, true
	}
	if cmpLess(a, b) {
		return word256{}, false
	}
	var q word256
	var r word256
	for i := 255; i >= 0; i-- {
		// r << 1
		var carry uint64
		for j := 3; j >= 0; j-- {
			next := (r[j] << 1) | carry
			carry = r[j] >> 63
			r[j] = next
		}
		bit := getBit(a, i)
		r[3] |= bit
		if !cmpLess(r, b) {
			r, _ = u256Sub(r, b)
			setBit(&q, i, 1)
		}
	}
	return q, false
}

func cmpLess(a, b word256) bool {
	for i := 0; i < 4; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func getBit(a word256, i int) uint64 {
	w := 3 - i/64
	b := uint(i % 64)
	return (a[w] >> b) & 1
}
func setBit(a *word256, i int, v uint64) {
	w := 3 - i/64
	b := uint(i % 64)
	if v == 1 {
		a[w] |= 1 << b
	} else {
		a[w] &^= 1 << b
	}
}

func u256ToHex(a word256) string {
	buf := make([]byte, 32)
	for i := 0; i < 4; i++ {
		binary.BigEndian.PutUint64(buf[i*8:], a[i])
	}
	// 去除前导 0
	i := 0
	for i < 31 && buf[i] == 0 {
		i++
	}
	return fmt.Sprintf("0x%x", buf[i:])
}

func u256FromHex(s string) (word256, error) {
	s = strings.TrimPrefix(s, "0x")
	for len(s) < 64 {
		s = "0" + s
	}
	if len(s) > 64 {
		return word256{}, fmt.Errorf("超过 256 bit")
	}
	var w word256
	for i := 0; i < 4; i++ {
		var v uint64
		_, err := fmt.Sscanf(s[i*16:i*16+16], "%x", &v)
		if err != nil {
			return word256{}, err
		}
		w[i] = v
	}
	return w, nil
}

// =====================================================================
// 通用 N-bit 包装：把 word256 截断到指定位宽。
// =====================================================================

func maskNBits(a word256, bits int) word256 {
	if bits >= 256 {
		return a
	}
	var mask word256
	for i := 0; i < bits; i++ {
		setBit(&mask, i, 1)
	}
	return word256{a[0] & mask[0], a[1] & mask[1], a[2] & mask[2], a[3] & mask[3]}
}

// nBitMax 返回 2^bits - 1 作为 word256。
func nBitMax(bits int) word256 {
	var w word256
	for i := 0; i < bits; i++ {
		setBit(&w, i, 1)
	}
	return w
}

// =====================================================================
// 算术（带模式）
// =====================================================================

type arithResult struct {
	A, B, Result word256
	Op           string
	Bits         int
	Mode         string
	Overflow     bool
	Underflow    bool
	Reverted     bool
	RevertNote   string
	Wrapped      bool // 静默 wrap（legacy / unchecked block 才会出现）
}

// computeArith 在指定位宽和模式下做一次运算。
func computeArith(a, b word256, op string, bits int, mode string) arithResult {
	res := arithResult{A: a, B: b, Op: op, Bits: bits, Mode: mode}

	// 全 256-bit 计算
	var raw word256
	var fullOver, fullUnder bool
	switch op {
	case opAdd:
		raw, fullOver = u256Add(a, b)
	case opSub:
		raw, fullUnder = u256Sub(a, b)
	case opMul:
		raw, fullOver = u256Mul(a, b)
	case opDiv:
		var divErr bool
		raw, divErr = u256Div(a, b)
		if divErr {
			res.Reverted = true
			res.RevertNote = "division by zero"
			return res
		}
	case opPow:
		// b 视作小指数（取低 64 bit）
		exp := b[3]
		if exp > 64 {
			exp = 64
		}
		raw = u256FromUint64(1)
		for i := uint64(0); i < exp; i++ {
			var ov bool
			raw, ov = u256Mul(raw, a)
			if ov {
				fullOver = true
			}
		}
	}

	// 截断到指定位宽并检测 wrap
	trunc := maskNBits(raw, bits)
	wrap := false
	if op == opAdd || op == opMul || op == opPow {
		// 上溢：原结果 ≠ 截断结果（256-bit 未溢出但 N-bit 溢出）
		if !equalU256(raw, trunc) || fullOver {
			wrap = true
		}
	}
	if op == opSub && fullUnder {
		wrap = true
	}

	res.Result = trunc
	switch {
	case op == opSub && fullUnder:
		res.Underflow = true
	case (op == opAdd || op == opMul || op == opPow) && wrap:
		res.Overflow = true
	}

	switch mode {
	case modeChecked:
		if res.Overflow || res.Underflow {
			res.Reverted = true
			res.RevertNote = fmt.Sprintf("Solidity 0.8.x checked: %s", overflowTag(res))
			res.Result = word256{}
			return res
		}
	case modeLegacy, modeUnchecked:
		if res.Overflow || res.Underflow {
			res.Wrapped = true
		}
	}
	return res
}

func overflowTag(r arithResult) string {
	if r.Overflow {
		return "overflow"
	}
	if r.Underflow {
		return "underflow"
	}
	return "ok"
}

func equalU256(a, b word256) bool {
	return a == b
}

// =====================================================================
// ERC-20 漏洞演示状态
// =====================================================================

type erc20State struct {
	TotalSupply word256
	Balances    map[string]word256
	Bits        int
	Mode        string
	History     []arithResult
}

func newERC20(bits int, mode string) *erc20State {
	es := &erc20State{Bits: bits, Mode: mode, Balances: map[string]word256{}}
	// 初始：alice 100，attacker 0，victim 0；total = 100
	es.Balances["alice"] = u256FromUint64(100)
	es.Balances["attacker"] = u256FromUint64(0)
	es.Balances["victim"] = u256FromUint64(50)
	es.TotalSupply = u256FromUint64(150)
	return es
}

// erc20Transfer from→to amount，带模式。
func (es *erc20State) transfer(from, to string, amount word256) (arithResult, error) {
	if _, ok := es.Balances[from]; !ok {
		return arithResult{}, fmt.Errorf("from=%s 不存在", from)
	}
	if _, ok := es.Balances[to]; !ok {
		es.Balances[to] = word256{}
	}
	bal := es.Balances[from]
	res := computeArith(bal, amount, opSub, es.Bits, es.Mode)
	es.History = append(es.History, res)
	if res.Reverted {
		return res, nil
	}
	es.Balances[from] = res.Result
	// to 加
	addRes := computeArith(es.Balances[to], amount, opAdd, es.Bits, es.Mode)
	es.History = append(es.History, addRes)
	if addRes.Reverted {
		// 回滚 from
		es.Balances[from] = bal
		return addRes, nil
	}
	es.Balances[to] = addRes.Result
	return addRes, nil
}

// erc20Mint 给 to 增发（演示 totalSupply 溢出）。
func (es *erc20State) mint(to string, amount word256) arithResult {
	res := computeArith(es.TotalSupply, amount, opAdd, es.Bits, es.Mode)
	es.History = append(es.History, res)
	if res.Reverted {
		return res
	}
	es.TotalSupply = res.Result
	es.Balances[to] = maskNBits(addRaw(es.Balances[to], amount), es.Bits)
	return res
}

func addRaw(a, b word256) word256 {
	r, _ := u256Add(a, b)
	return r
}

// =====================================================================
// 场景全局状态
// =====================================================================

type snapState struct {
	Bits              int
	Mode              string
	Calculations      []arithResult
	ERC20             *erc20State
	OverflowCnt       int
	UnderflowCnt      int
	CheckedReverts    int
	SilentWraps       int
	StolenViaOverflow word256
	Tick              int
	LastError         string
}

func defaultSnapState() snapState {
	return snapState{
		Bits: bitsU8, Mode: modeLegacy,
		ERC20: newERC20(bitsU8, modeLegacy),
	}
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
		Bits: fw.MapInt(d, "bits", bitsU8), Mode: fw.MapStr(d, "mode", modeLegacy),
		OverflowCnt:    fw.MapInt(d, "overflow_cnt", 0),
		UnderflowCnt:   fw.MapInt(d, "underflow_cnt", 0),
		CheckedReverts: fw.MapInt(d, "checked_reverts", 0),
		SilentWraps:    fw.MapInt(d, "silent_wraps", 0),
		Tick:           fw.MapInt(d, "tick", 0),
		LastError:      fw.MapStr(d, "last_error", ""),
	}
	if cAny, ok := d["calcs"].([]any); ok {
		for _, x := range cAny {
			if m, ok := x.(map[string]any); ok {
				st.Calculations = append(st.Calculations, decodeArith(m))
			}
		}
	}
	if eAny, ok := d["erc20"].(map[string]any); ok {
		es := &erc20State{Bits: st.Bits, Mode: st.Mode, Balances: map[string]word256{}}
		if tsHex, ok := eAny["total_supply"].(string); ok {
			ts, _ := u256FromHex(tsHex)
			es.TotalSupply = ts
		}
		if balAny, ok := eAny["balances"].(map[string]any); ok {
			for k, vAny := range balAny {
				if vs, ok := vAny.(string); ok {
					b, _ := u256FromHex(vs)
					es.Balances[k] = b
				}
			}
		}
		if hAny, ok := eAny["history"].([]any); ok {
			for _, x := range hAny {
				if m, ok := x.(map[string]any); ok {
					es.History = append(es.History, decodeArith(m))
				}
			}
		}
		st.ERC20 = es
	}
	if st.ERC20 == nil {
		st.ERC20 = newERC20(st.Bits, st.Mode)
	}
	if stolenHex, ok := d["stolen"].(string); ok {
		st.StolenViaOverflow, _ = u256FromHex(stolenHex)
	}
	return st
}

func decodeArith(m map[string]any) arithResult {
	a, _ := u256FromHex(fw.MapStr(m, "a", "0x0"))
	b, _ := u256FromHex(fw.MapStr(m, "b", "0x0"))
	r, _ := u256FromHex(fw.MapStr(m, "r", "0x0"))
	return arithResult{
		A: a, B: b, Result: r,
		Op: fw.MapStr(m, "op", ""), Bits: fw.MapInt(m, "bits", 0), Mode: fw.MapStr(m, "mode", ""),
		Overflow:   fw.MapBool(m, "over", false),
		Underflow:  fw.MapBool(m, "under", false),
		Reverted:   fw.MapBool(m, "rev", false),
		RevertNote: fw.MapStr(m, "note", ""),
		Wrapped:    fw.MapBool(m, "wrap", false),
	}
}

func encodeArith(r arithResult) map[string]any {
	return map[string]any{
		"a": u256ToHex(r.A), "b": u256ToHex(r.B), "r": u256ToHex(r.Result),
		"op": r.Op, "bits": r.Bits, "mode": r.Mode,
		"over": r.Overflow, "under": r.Underflow,
		"rev": r.Reverted, "note": r.RevertNote, "wrap": r.Wrapped,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["bits"] = st.Bits
	s.Data["mode"] = st.Mode
	s.Data["overflow_cnt"] = st.OverflowCnt
	s.Data["underflow_cnt"] = st.UnderflowCnt
	s.Data["checked_reverts"] = st.CheckedReverts
	s.Data["silent_wraps"] = st.SilentWraps
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	cAny := make([]any, len(st.Calculations))
	for i, r := range st.Calculations {
		cAny[i] = encodeArith(r)
	}
	s.Data["calcs"] = cAny
	if st.ERC20 != nil {
		balAny := map[string]any{}
		for k, v := range st.ERC20.Balances {
			balAny[k] = u256ToHex(v)
		}
		hAny := make([]any, len(st.ERC20.History))
		for i, r := range st.ERC20.History {
			hAny[i] = encodeArith(r)
		}
		s.Data["erc20"] = map[string]any{
			"total_supply": u256ToHex(st.ERC20.TotalSupply),
			"balances":     balAny,
			"history":      hAny,
		}
	}
	s.Data["stolen"] = u256ToHex(st.StolenViaOverflow)
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "整数溢出/下溢攻击",
		Description:         "演示 uint8/16/32/64/256 算术 + legacy / checked / unchecked 三模式 + ERC-20 transfer 下溢漏洞",
		Category:            fw.CategoryAttackSecurity,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupContractSec},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"attack.overflow.stolen_via_overflow",
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
				ActionCode: "set_mode", Label: "设置位宽与模式",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "bits", Type: fw.FieldEnum, Label: "位宽", Required: true, Default: bitsU8,
						Options: []any{bitsU8, bitsU16, bitsU32, bitsU64, bitsU256}},
					{Name: "mode", Type: fw.FieldEnum, Label: "检查模式", Required: true, Default: modeLegacy,
						Options: []any{modeLegacy, modeChecked, modeUnchecked}},
				},
			},
			{
				ActionCode: "compute", Label: "算一次（add/sub/mul/div/pow）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "a", Type: fw.FieldString, Label: "a (hex)", Required: true, Default: "0xff"},
					{Name: "b", Type: fw.FieldString, Label: "b (hex)", Required: true, Default: "0x01"},
					{Name: "op", Type: fw.FieldEnum, Label: "操作", Required: true, Default: opAdd,
						Options: []any{opAdd, opSub, opMul, opDiv, opPow}},
				},
			},
			{
				ActionCode: "erc20_transfer", Label: "ERC-20 转账",
				Description:   "演示 amount > balance → 下溢漏洞",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldEnum, Label: "from", Required: true, Default: "alice",
						Options: []any{"alice", "attacker", "victim"}},
					{Name: "to", Type: fw.FieldEnum, Label: "to", Required: true, Default: "victim",
						Options: []any{"alice", "attacker", "victim"}},
					{Name: "amount", Type: fw.FieldString, Label: "amount (hex)", Required: true, Default: "0x32"},
				},
			},
			{
				ActionCode: "erc20_mint", Label: "ERC-20 增发",
				Description:   "演示 totalSupply 上溢",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "attacker"},
					{Name: "amount", Type: fw.FieldString, Label: "amount (hex)", Required: true, Default: "0xff"},
				},
			},
			{
				ActionCode: "attack_underflow", Label: "演示：transfer 下溢",
				Description: "attacker 在 legacy/unchecked 模式下 transfer 100，但余额=0 → 下溢得到 max_uint",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.InterveneAttack,
				WritesOwnedFields: []string{"attack.overflow.stolen_via_overflow"},
				LinkOwnerFields:   []string{"attack.overflow.stolen_via_overflow"},
			},
			{
				ActionCode: "reset_erc20", Label: "重置 ERC-20",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode: "reset", Label: "重置全部",
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
			{
				ActionCode:    "replay_poc",
				Label:         "重放 PoC（patch-verifier）",
				Description:   "调 patch-verifier 重放溢出攻击 PoC",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"contract":"{{contract}}","patch":"{{patch}}"}' http://patch-verifier:8091/api/v1/patches/verify`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "contract", Type: fw.FieldString, Label: "vulnerable contract", Required: true, Default: ""},
					{Name: "patch", Type: fw.FieldString, Label: "patch source", Required: true, Default: ""},
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
	env := buildEnvelope(st, "init", "uint8 / legacy 模式初始化", true)
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
	st.Tick++

	switch in.ActionCode {
	case "set_mode":
		st.Bits = fw.MapInt(in.Params, "bits", bitsU8)
		st.Mode = fw.MapStr(in.Params, "mode", modeLegacy)
		st.ERC20 = newERC20(st.Bits, st.Mode)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_mode",
			fmt.Sprintf("bits=%d, mode=%s", st.Bits, st.Mode), true)
		return out, nil

	case "compute":
		aHex := fw.MapStr(in.Params, "a", "0xff")
		bHex := fw.MapStr(in.Params, "b", "0x01")
		op := fw.MapStr(in.Params, "op", opAdd)
		a, err := u256FromHex(aHex)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "a 无效: " + err.Error()}, nil
		}
		b, err := u256FromHex(bHex)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "b 无效: " + err.Error()}, nil
		}
		// 截断到当前位宽（教学：保证输入也在位宽内）
		a = maskNBits(a, st.Bits)
		b = maskNBits(b, st.Bits)
		res := computeArith(a, b, op, st.Bits, st.Mode)
		st.Calculations = append(st.Calculations, res)
		if len(st.Calculations) > 32 {
			st.Calculations = st.Calculations[len(st.Calculations)-32:]
		}
		st.tally(res)
		saveState(state, st)
		summary := summarizeArith(res)
		out.Render = buildEnvelope(st, "compute", summary, false)
		appendComputeMicroSteps(&out.Render, res)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "erc20_transfer":
		from := fw.MapStr(in.Params, "from", "alice")
		to := fw.MapStr(in.Params, "to", "victim")
		amtHex := fw.MapStr(in.Params, "amount", "0x32")
		amt, err := u256FromHex(amtHex)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		amt = maskNBits(amt, st.Bits)
		st.ERC20.Bits = st.Bits
		st.ERC20.Mode = st.Mode
		res, err := st.ERC20.transfer(from, to, amt)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.tally(res)
		saveState(state, st)
		summary := fmt.Sprintf("transfer %s→%s amount=%s | %s",
			from, to, u256ToHex(amt), summarizeArith(res))
		out.Render = buildEnvelope(st, "erc20_transfer", summary, false)
		appendTransferMicroSteps(&out.Render, res)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "erc20_mint":
		to := fw.MapStr(in.Params, "to", "attacker")
		amtHex := fw.MapStr(in.Params, "amount", "0xff")
		amt, err := u256FromHex(amtHex)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		amt = maskNBits(amt, st.Bits)
		st.ERC20.Bits = st.Bits
		st.ERC20.Mode = st.Mode
		res := st.ERC20.mint(to, amt)
		st.tally(res)
		saveState(state, st)
		summary := fmt.Sprintf("mint %s += %s | %s", to, u256ToHex(amt), summarizeArith(res))
		out.Render = buildEnvelope(st, "erc20_mint", summary, false)
		appendMintMicroSteps(&out.Render, res)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "attack_underflow":
		// attacker.balance = 0；transfer 100 → 下溢
		st.ERC20.Bits = st.Bits
		st.ERC20.Mode = st.Mode
		bigAmount := u256FromUint64(100)
		bigAmount = maskNBits(bigAmount, st.Bits)
		preBal := st.ERC20.Balances["attacker"]
		res, _ := st.ERC20.transfer("attacker", "victim", bigAmount)
		st.tally(res)
		// 计算 attacker 余额增量 = 后 - 前（教学：attacker 在 legacy 下被"赋予"巨额）
		postBal := st.ERC20.Balances["attacker"]
		gained, _ := u256Sub(postBal, preBal)
		// 在 wrap 情形 gained 极大；记入 stolen
		if res.Wrapped {
			st.StolenViaOverflow, _ = u256Add(st.StolenViaOverflow, gained)
		}
		saveState(state, st)
		summary := summarizeArith(res)
		out.Render = buildEnvelope(st, "attack_underflow",
			"Underflow demo: "+summary, false)
		appendUnderflowMicroSteps(&out.Render, res)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "reset_erc20":
		st.ERC20 = newERC20(st.Bits, st.Mode)
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset_erc20", "ERC-20 已重置", false)
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
		out.Render = buildEnvelope(st, "reset", "全部已重置", true)
		return out, nil
	}
	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode"}, errors.New("unknown action")
}

// tally 累计统计计数器。
func (st *snapState) tally(r arithResult) {
	if r.Overflow {
		st.OverflowCnt++
	}
	if r.Underflow {
		st.UnderflowCnt++
	}
	if r.Reverted && (r.Overflow || r.Underflow || strings.Contains(r.RevertNote, "checked")) {
		st.CheckedReverts++
	}
	if r.Wrapped {
		st.SilentWraps++
	}
}

func summarizeArith(r arithResult) string {
	if r.Reverted {
		return fmt.Sprintf("✗ REVERT (%s, %s)", r.RevertNote, overflowTag(r))
	}
	tag := ""
	if r.Wrapped {
		tag = " ⚠ WRAPPED"
	}
	return fmt.Sprintf("%s %s %s = %s [%dbit / %s]%s",
		u256ToHex(r.A), r.Op, u256ToHex(r.B), u256ToHex(r.Result), r.Bits, r.Mode, tag)
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 60)

	// 1) 模式开关流水线
	modes := []string{modeLegacy, modeChecked, modeUnchecked}
	mIDs := []string{"m-legacy", "m-checked", "m-unchecked"}
	prims = append(prims, fw.PrimStack("mode-stack", mIDs, "horizontal"))
	for i, m := range modes {
		role := "mode-" + m
		status := "normal"
		if m == st.Mode {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(mIDs[i], m, status, role))
	}

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-mod",
		fmt.Sprintf(`\text{uint%d 算术：} (a \,\text{op}\, b) \bmod 2^{%d}`, st.Bits, st.Bits), false))
	prims = append(prims, fw.PrimMathFormula("formula-checked",
		`\text{checked: } a \pm b \to \text{REVERT if overflow/underflow}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-unchecked",
		`\text{unchecked / legacy: } \text{silently wrap to } (a \,\text{op}\, b) \bmod 2^N`, false))

	// 3) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("位宽 = %d  模式 = %s\n2^N - 1 = %s\n累计 overflow = %d  underflow = %d\nchecked REVERT = %d  silent wraps = %d\n stolen via overflow = %s",
			st.Bits, st.Mode, u256ToHex(nBitMax(st.Bits)),
			st.OverflowCnt, st.UnderflowCnt,
			st.CheckedReverts, st.SilentWraps, u256ToHex(st.StolenViaOverflow)),
		"text", nil, 8))

	// 4) 算术历史
	if len(st.Calculations) > 0 {
		cLines := []string{"a              op    b              result          bits  mode             flags"}
		startIdx := 0
		if len(st.Calculations) > 12 {
			startIdx = len(st.Calculations) - 12
		}
		for _, r := range st.Calculations[startIdx:] {
			flags := ""
			if r.Overflow {
				flags += "OVER "
			}
			if r.Underflow {
				flags += "UNDER "
			}
			if r.Wrapped {
				flags += "WRAP "
			}
			if r.Reverted {
				flags += "REVERT"
			}
			cLines = append(cLines, fmt.Sprintf("  %-13s %-5s %-13s %-15s %-5d %-16s %s",
				u256ToHex(r.A), r.Op, u256ToHex(r.B), u256ToHex(r.Result),
				r.Bits, r.Mode, flags))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-calcs", strings.Join(cLines, "\n"), "text", nil, 14))
	}

	// 5) ERC-20 状态
	if st.ERC20 != nil {
		bLines := []string{"holder      balance"}
		keys := []string{}
		for k := range st.ERC20.Balances {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			bLines = append(bLines, fmt.Sprintf("  %-9s  %s", k, u256ToHex(st.ERC20.Balances[k])))
		}
		bLines = append(bLines, fmt.Sprintf("totalSupply = %s", u256ToHex(st.ERC20.TotalSupply)))
		prims = append(prims, fw.PrimCodeBlock("cb-erc20", strings.Join(bLines, "\n"), "text", nil, 8))
	}

	// 6) ERC-20 历史
	if st.ERC20 != nil && len(st.ERC20.History) > 0 {
		hLines := []string{"a              op   b              result          bits  mode             flags"}
		startIdx := 0
		if len(st.ERC20.History) > 12 {
			startIdx = len(st.ERC20.History) - 12
		}
		for _, r := range st.ERC20.History[startIdx:] {
			flags := ""
			if r.Overflow {
				flags += "OVER "
			}
			if r.Underflow {
				flags += "UNDER "
			}
			if r.Wrapped {
				flags += "WRAP "
			}
			if r.Reverted {
				flags += "REVERT"
			}
			hLines = append(hLines, fmt.Sprintf("  %-13s %-4s %-13s %-15s %-5d %-16s %s",
				u256ToHex(r.A), r.Op, u256ToHex(r.B), u256ToHex(r.Result),
				r.Bits, r.Mode, flags))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-erc20-history", strings.Join(hLines, "\n"), "text", nil, 14))
	}

	// 7) 进度条
	prims = append(prims, fw.PrimBar("bar-over", float64(st.OverflowCnt), 0, "warning", "Overflow Detected"))
	prims = append(prims, fw.PrimBar("bar-under", float64(st.UnderflowCnt), 0, "warning", "Underflow Detected"))
	prims = append(prims, fw.PrimBar("bar-revert", float64(st.CheckedReverts), 0, "success", "Checked REVERT"))
	prims = append(prims, fw.PrimBar("bar-wrap", float64(st.SilentWraps), 0, "danger", "Silent WRAP (UNSAFE)"))

	// 8) 动效
	if st.SilentWraps > 0 {
		prims = append(prims, fw.PrimShake("shake-wrap", "bar-wrap", 0.5, 800))
		prims = append(prims, fw.PrimPulse("pulse-wrap", "bar-wrap", "danger", 1500))
	}
	if st.CheckedReverts > 0 {
		prims = append(prims, fw.PrimGlow("glow-checked", "m-checked", "success", 0.9))
	}

	// 9) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Overflow 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "patch-verifier", MetricKey: "overflow.overflow_cnt", Value: st.OverflowCnt, TargetPrimitive: "cb-poc", TargetParam: "count"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"bits":            st.Bits,
		"mode":            st.Mode,
		"overflow_cnt":    st.OverflowCnt,
		"underflow_cnt":   st.UnderflowCnt,
		"checked_reverts": st.CheckedReverts,
		"silent_wraps":    st.SilentWraps,
		"stolen_overflow": u256ToHex(st.StolenViaOverflow),
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

func appendComputeMicroSteps(env *fw.RenderEnvelope, r arithResult) {
	tail := summarizeArith(r)
	steps := []fw.MicroStep{
		{ID: "c-1", Label: "执行 256-bit 全宽运算", DurationMs: 400, HighlightIDs: []string{"formula-mod"}},
		{ID: "c-2", Label: fmt.Sprintf("截断到 uint%d", r.Bits), DurationMs: 400, HighlightIDs: []string{"cb-status"}},
	}
	if r.Reverted {
		steps = append(steps, fw.MicroStep{ID: "c-3", Label: "checked 模式 → REVERT", DurationMs: 500,
			HighlightIDs: []string{"formula-checked", "bar-revert"}, FirePrimitives: []string{"glow-checked"}, IsLinkTrigger: true})
	} else if r.Wrapped {
		steps = append(steps, fw.MicroStep{ID: "c-3", Label: "legacy / unchecked → 静默 WRAP", DurationMs: 500,
			HighlightIDs: []string{"formula-unchecked", "bar-wrap"}, FirePrimitives: []string{"shake-wrap", "pulse-wrap"}, IsLinkTrigger: true})
	} else {
		steps = append(steps, fw.MicroStep{ID: "c-3", Label: "无溢出: " + tail, DurationMs: 400,
			HighlightIDs: []string{"cb-calcs"}, IsLinkTrigger: true})
	}
	env.MicroSteps = steps
}

func appendTransferMicroSteps(env *fw.RenderEnvelope, r arithResult) {
	tail := summarizeArith(r)
	steps := []fw.MicroStep{
		{ID: "t-1", Label: "from -= amount → 检查下溢", DurationMs: 400, HighlightIDs: []string{"cb-erc20", "formula-mod"}},
	}
	if r.Reverted {
		steps = append(steps, fw.MicroStep{ID: "t-2", Label: "REVERT，余额不变", DurationMs: 500,
			HighlightIDs: []string{"bar-revert"}, FirePrimitives: []string{"glow-checked"}, IsLinkTrigger: true})
	} else if r.Wrapped {
		steps = append(steps, fw.MicroStep{ID: "t-2", Label: "⚠ legacy 下下溢: from balance 变为巨大值", DurationMs: 500,
			HighlightIDs: []string{"bar-wrap", "cb-erc20"}, FirePrimitives: []string{"shake-wrap", "pulse-wrap"}, IsLinkTrigger: true})
	} else {
		steps = append(steps, fw.MicroStep{ID: "t-2", Label: "正常转账: " + tail, DurationMs: 400,
			HighlightIDs: []string{"cb-erc20"}, IsLinkTrigger: true})
	}
	env.MicroSteps = steps
}

func appendMintMicroSteps(env *fw.RenderEnvelope, r arithResult) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: "totalSupply += amount", DurationMs: 400, HighlightIDs: []string{"cb-erc20"}},
		{ID: "m-2", Label: summarizeArith(r), DurationMs: 500, HighlightIDs: []string{"bar-over", "bar-wrap"}, IsLinkTrigger: true},
	}
}

func appendUnderflowMicroSteps(env *fw.RenderEnvelope, r arithResult) {
	tail := "✓ checked 模式拦截"
	if r.Wrapped {
		tail = "⚠ 攻击成功：attacker.balance 变为 max_uint，可无限转账"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "u-1", Label: "attacker.balance(0) − 100 = ?", DurationMs: 400, HighlightIDs: []string{"formula-mod", "cb-erc20"}},
		{ID: "u-2", Label: "256-bit subtract underflow", DurationMs: 400, HighlightIDs: []string{"formula-unchecked"}},
		{ID: "u-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-wrap", "cb-erc20"}, FirePrimitives: []string{"shake-wrap", "pulse-wrap"}, IsLinkTrigger: true},
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
		ID:             "overflow-attack",
		SourceScene:    sceneCode,
		SourceAction:   "trigger_overflow",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"attack.overflow.overflow_cnt", "attack.overflow.underflow_cnt"},
		Payload:        map[string]any{"overflow_cnt": st.OverflowCnt, "underflow_cnt": st.UnderflowCnt},
		SourceAnchorID: "overflow-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"attack": map[string]any{
			"overflow": map[string]any{
				"bits":                st.Bits,
				"mode":                st.Mode,
				"overflow_cnt":        st.OverflowCnt,
				"underflow_cnt":       st.UnderflowCnt,
				"checked_reverts":     st.CheckedReverts,
				"silent_wraps":        st.SilentWraps,
				"stolen_via_overflow": u256ToHex(st.StolenViaOverflow),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

