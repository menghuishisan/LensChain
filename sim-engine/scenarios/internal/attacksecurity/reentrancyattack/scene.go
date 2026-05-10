// 模块：sim-engine/scenarios/internal/attacksecurity/reentrancyattack
// 文件职责：ATK-04 重入攻击场景的完整实现（The DAO 历史还原）。
//
// SSOT 依据：06.md §4.7.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 EVM 风格调用栈 + 三种 Vault 合约 + 三种受害者合约对照（零外部依赖）。
//
//   1. Vault 合约（教学：以"函数 + storage + balance"建模）：
//      · vault_vulnerable     : checks-interactions-effects 顺序错误
//          fn withdraw():
//              if balances[msg.sender] >= amt: # check
//                  callEOA(msg.sender, amt)    # interaction（外部调用，给攻击者控制权）
//                  balances[msg.sender] -= amt # effect（先打钱再扣账）
//      · vault_secure_cei     : checks-effects-interactions 正确顺序
//          fn withdraw():
//              if balances[msg.sender] >= amt:
//                  balances[msg.sender] -= amt # effect 先扣账
//                  callEOA(msg.sender, amt)    # 外部调用最后
//      · vault_with_guard     : 带 ReentrancyGuard（OpenZeppelin 风格）
//          fn withdraw() nonReentrant:
//              vault_vulnerable.withdraw()
//
//   2. 攻击者合约 Attacker：
//      · receive() / fallback()：每次收到 ETH 时如果还能提现就再次调用 vault.withdraw()
//      · attackDepth：当前重入深度（最大 maxAttackDepth）
//
//   3. 调用栈模型：
//      · CallFrame { depth, caller, callee, fn, value, msgSender, isStatic }
//      · executeFrame 沿着调用栈递归执行；externalCall 触发 fallback() → 嵌套
//      · stackDepthLimit = 1024（与 EVM 一致）；超出 → REVERT
//
//   4. 数值指标：
//      · vaultBalanceBefore / vaultBalanceAfter
//      · attackerStolen
//      · reentrancyDepthMax
//      · guardBlockedCount（被 ReentrancyGuard 拦截）
//      · ceiBlockedCount（被 CEI 安全顺序拦截）
//
//   5. 教学场景脚本（自动一键演示）：
//      · run_attack_vulnerable : 演示 The DAO 漏洞 → attacker 把整个 vault 抽干
//      · run_attack_cei        : 演示 secure-CEI vault 抗 reentrancy（attacker 只取一次）
//      · run_attack_guard      : 演示带 nonReentrant 的 vault 抗 reentrancy

package reentrancyattack

import (
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "reentrancy-attack"
	schemaVersion = "v1.0.0"
	algorithmType = "reentrancy"

	vaultVulnerable = "vault_vulnerable"
	vaultSecureCEI  = "vault_secure_cei"
	vaultWithGuard  = "vault_with_guard"

	maxStackDepth       = 1024
	maxAttackDepth      = 16 // 教学：避免栈过深
	attackerInitDeposit = 100
	vaultInitTotal      = 1000

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "attack.reentrancy"
)

// =====================================================================
// 数据结构
// =====================================================================

// vault 一份 Vault 合约的状态。
type vault struct {
	Kind     string // vulnerable / secure_cei / with_guard
	Balance  int64
	Deposits map[string]int64 // 用户记账
	Locked   bool             // ReentrancyGuard 标志
}

func newVault(kind string) *vault {
	return &vault{Kind: kind, Deposits: map[string]int64{}}
}

// callFrame 调用栈帧。
type callFrame struct {
	Depth      int
	Caller     string
	Callee     string // vault_xxx / attacker / eoa
	Fn         string // deposit / withdraw / fallback
	Value      int64  // 转账金额
	MsgSender  string
	Reverted   bool
	RevertNote string
	Note       string // 教学说明
}

// runReport 一次攻击演示的报告。
type runReport struct {
	Variant            string
	Trace              []callFrame
	StackDepthMax      int
	ReentrancyDepthMax int
	VaultBefore        int64
	VaultAfter         int64
	AttackerStolen     int64
	GuardBlocks        int
	CEIBlocks          int
	Success            bool
	Detail             string
}

type snapState struct {
	Vaults      map[string]*vault
	EOA         map[string]int64 // 链下账户
	Reports     []runReport
	GuardBlocks int
	CEIBlocks   int
	StolenTotal int64
	Tick        int
	LastError   string
}

func defaultSnapState() snapState {
	st := snapState{
		Vaults: map[string]*vault{},
		EOA:    map[string]int64{"alice": 1000, "attacker": 1000, "victim": 0},
	}
	for _, k := range []string{vaultVulnerable, vaultSecureCEI, vaultWithGuard} {
		v := newVault(k)
		v.Balance = vaultInitTotal
		v.Deposits["alice"] = vaultInitTotal - attackerInitDeposit
		v.Deposits["attacker"] = attackerInitDeposit
		st.Vaults[k] = v
	}
	return st
}

// =====================================================================
// 调用栈执行
// =====================================================================

// executeFrame 执行单个调用帧；返回是否成功（!Reverted）。
// 攻击重入逻辑：当 vault.withdraw 把钱"打"给 attacker 时，
// 触发 attacker.fallback() 再次调用 vault.withdraw() —— 形成栈递归。
func (st *snapState) executeFrame(f *callFrame, attackerCanReenter *bool, report *runReport) {
	if f.Depth > report.StackDepthMax {
		report.StackDepthMax = f.Depth
	}
	if f.Depth > maxStackDepth {
		f.Reverted = true
		f.RevertNote = "stack overflow > 1024"
		return
	}

	switch f.Callee {
	case vaultVulnerable, vaultSecureCEI, vaultWithGuard:
		st.execVault(f, attackerCanReenter, report)
	case "attacker":
		st.execAttacker(f, attackerCanReenter, report)
	default:
		f.Reverted = true
		f.RevertNote = "未知合约: " + f.Callee
	}
}

// execVault 执行 vault.deposit / vault.withdraw 函数。
func (st *snapState) execVault(f *callFrame, attackerCanReenter *bool, report *runReport) {
	v := st.Vaults[f.Callee]
	switch f.Fn {
	case "deposit":
		// EOA 转账给 vault
		if st.EOA[f.MsgSender] < f.Value {
			f.Reverted = true
			f.RevertNote = "EOA 余额不足"
			return
		}
		st.EOA[f.MsgSender] -= f.Value
		v.Balance += f.Value
		v.Deposits[f.MsgSender] += f.Value
		f.Note = fmt.Sprintf("deposit %d → vault.balance=%d", f.Value, v.Balance)

	case "withdraw":
		// 三种实现
		amount := v.Deposits[f.MsgSender]
		switch v.Kind {
		case vaultVulnerable:
			st.withdrawVulnerable(f, v, amount, attackerCanReenter, report)
		case vaultSecureCEI:
			st.withdrawSecureCEI(f, v, amount, attackerCanReenter, report)
		case vaultWithGuard:
			st.withdrawWithGuard(f, v, amount, attackerCanReenter, report)
		}
	}
}

// withdrawVulnerable Checks-Interactions-Effects 错序。
func (st *snapState) withdrawVulnerable(f *callFrame, v *vault, amount int64, attackerCanReenter *bool, report *runReport) {
	// CHECK
	if amount <= 0 {
		f.Reverted = true
		f.RevertNote = "无可提现余额"
		return
	}
	if v.Balance < amount {
		f.Reverted = true
		f.RevertNote = "vault 余额不足"
		return
	}
	// 注：教学版判断每次 withdraw 实际能转出多少：
	// 真实漏洞中 amount = deposits[sender]，外部 call 使 vault.balance 减 amount，
	// 但 deposits[sender] 没扣，攻击者重入时再次以同 amount 拿到钱。
	// 这里建模为：每次 withdraw 转出 min(amount, vault.Balance)。
	transfer := amount
	if v.Balance < transfer {
		transfer = v.Balance
	}
	// INTERACTION（先转账，给攻击者控制权）
	v.Balance -= transfer
	f.Note = fmt.Sprintf("[VULN] interaction 先转 %d → vault=%d", transfer, v.Balance)
	// 触发 callee=attacker fallback
	st.invokeFallback(f, transfer, attackerCanReenter, report)
	// EFFECT（之后才扣账，但攻击已得手）
	if !f.Reverted {
		v.Deposits[f.MsgSender] -= transfer
		// 防止负值
		if v.Deposits[f.MsgSender] < 0 {
			v.Deposits[f.MsgSender] = 0
		}
	}
}

// withdrawSecureCEI Checks-Effects-Interactions 正确顺序。
func (st *snapState) withdrawSecureCEI(f *callFrame, v *vault, amount int64, attackerCanReenter *bool, report *runReport) {
	if amount <= 0 {
		f.Reverted = true
		f.RevertNote = "无可提现余额"
		return
	}
	if v.Balance < amount {
		f.Reverted = true
		f.RevertNote = "vault 余额不足"
		return
	}
	// EFFECT 先扣账
	v.Deposits[f.MsgSender] -= amount
	v.Balance -= amount
	f.Note = fmt.Sprintf("[CEI] effect 先扣 %d → vault=%d deposit[%s]=%d", amount, v.Balance, f.MsgSender, v.Deposits[f.MsgSender])
	// INTERACTION 后转账（重入也无效，因为 deposits 已经清零）
	st.invokeFallback(f, amount, attackerCanReenter, report)
	report.CEIBlocks++
	st.CEIBlocks++
}

// withdrawWithGuard ReentrancyGuard。
func (st *snapState) withdrawWithGuard(f *callFrame, v *vault, amount int64, attackerCanReenter *bool, report *runReport) {
	if v.Locked {
		// nonReentrant 拦截
		f.Reverted = true
		f.RevertNote = "ReentrancyGuard: nonReentrant"
		report.GuardBlocks++
		st.GuardBlocks++
		return
	}
	v.Locked = true
	defer func() { v.Locked = false }()
	if amount <= 0 {
		f.Reverted = true
		f.RevertNote = "无可提现余额"
		return
	}
	if v.Balance < amount {
		f.Reverted = true
		f.RevertNote = "vault 余额不足"
		return
	}
	// 即便顺序错序，也被 guard 保护
	v.Balance -= amount
	f.Note = fmt.Sprintf("[GUARD locked=true] interaction 转 %d", amount)
	st.invokeFallback(f, amount, attackerCanReenter, report)
	if !f.Reverted {
		v.Deposits[f.MsgSender] -= amount
		if v.Deposits[f.MsgSender] < 0 {
			v.Deposits[f.MsgSender] = 0
		}
	}
}

// invokeFallback 模拟 vault 把钱"call"给 attacker（触发其 fallback）。
func (st *snapState) invokeFallback(parent *callFrame, value int64, attackerCanReenter *bool, report *runReport) {
	if parent.MsgSender != "attacker" {
		// 受害方是 alice，则 alice 直接收到钱（没有 fallback 攻击）
		st.EOA[parent.MsgSender] += value
		return
	}
	// 攻击者收到钱
	st.EOA["attacker"] += value
	report.AttackerStolen += value
	st.StolenTotal += value
	// 触发 attacker.fallback()
	child := callFrame{
		Depth: parent.Depth + 1, Caller: parent.Callee, Callee: "attacker",
		Fn: "fallback", Value: value, MsgSender: parent.Callee,
		Note: fmt.Sprintf("attacker.fallback() 收 %d，决定是否再次 withdraw", value),
	}
	if child.Depth > report.ReentrancyDepthMax {
		report.ReentrancyDepthMax = child.Depth
	}
	report.Trace = append(report.Trace, child)
	st.executeFrame(&child, attackerCanReenter, report)
}

// execAttacker fallback：判断是否能再次 reenter；能则发起对 vault.withdraw 的嵌套调用。
func (st *snapState) execAttacker(f *callFrame, attackerCanReenter *bool, report *runReport) {
	if attackerCanReenter == nil || !*attackerCanReenter {
		f.Note = "attacker.fallback() 不再 reenter"
		return
	}
	// 检测 reenter 次数上限
	if f.Depth > maxAttackDepth*2 { // 每个 reenter 涉及两层（vault.withdraw + attacker.fallback）
		f.Note = "fallback 抵达最大重入深度"
		return
	}
	// 检测目标 vault 还有钱可拿
	target := report.Variant // 当前演示的 vault kind
	v := st.Vaults[target]
	if v == nil {
		return
	}
	if v.Balance <= 0 {
		f.Note = "vault 已被掏空，停止重入"
		return
	}
	if v.Deposits["attacker"] <= 0 && v.Kind != vaultVulnerable {
		f.Note = "deposits[attacker]=0，停止"
		return
	}
	// 发起重入
	child := callFrame{
		Depth: f.Depth + 1, Caller: "attacker", Callee: target,
		Fn: "withdraw", Value: 0, MsgSender: "attacker",
		Note: "attacker.fallback() 触发 RE-ENTRY",
	}
	if child.Depth > report.ReentrancyDepthMax {
		report.ReentrancyDepthMax = child.Depth
	}
	report.Trace = append(report.Trace, child)
	st.executeFrame(&child, attackerCanReenter, report)
}

// =====================================================================
// 高层操作：deposit / withdraw / 一键攻击
// =====================================================================

// directDeposit EOA 直接 deposit 进 vault。
func (st *snapState) directDeposit(eoa, vaultName string, amount int64) error {
	v, ok := st.Vaults[vaultName]
	if !ok {
		return fmt.Errorf("未知 vault: %s", vaultName)
	}
	if st.EOA[eoa] < amount {
		return errors.New("EOA 余额不足")
	}
	st.EOA[eoa] -= amount
	v.Balance += amount
	v.Deposits[eoa] += amount
	return nil
}

// runAttack 攻击：attacker.callVault.withdraw() + 重入。
// canReenter=true → 触发 reentry；false → 仅 1 次 withdraw（无重入对照）。
func (st *snapState) runAttack(vaultName string, canReenter bool) runReport {
	st.Tick++
	v := st.Vaults[vaultName]
	report := runReport{
		Variant:     vaultName,
		VaultBefore: v.Balance,
	}
	// 顶层 frame：EOA(attacker) → vault.withdraw
	root := callFrame{
		Depth: 1, Caller: "attacker", Callee: vaultName,
		Fn: "withdraw", Value: 0, MsgSender: "attacker",
		Note: "顶层调用 vault.withdraw()",
	}
	report.ReentrancyDepthMax = 1
	report.StackDepthMax = 1
	report.Trace = append(report.Trace, root)
	flag := canReenter
	st.executeFrame(&root, &flag, &report)
	report.VaultAfter = v.Balance
	report.Success = report.AttackerStolen > attackerInitDeposit
	switch {
	case vaultName == vaultVulnerable && canReenter && report.AttackerStolen >= report.VaultBefore:
		report.Detail = fmt.Sprintf("⚠ DAO 漏洞：vault 被抽干 %d/%d", report.AttackerStolen, report.VaultBefore)
	case vaultName == vaultSecureCEI:
		report.Detail = fmt.Sprintf("✓ CEI 顺序正确：attacker 仅取走自己的 %d", report.AttackerStolen)
	case vaultName == vaultWithGuard && report.GuardBlocks > 0:
		report.Detail = fmt.Sprintf("✓ ReentrancyGuard 拦截 %d 次重入", report.GuardBlocks)
	case !canReenter:
		report.Detail = fmt.Sprintf("无重入对照：attacker 取 %d", report.AttackerStolen)
	default:
		report.Detail = fmt.Sprintf("attacker 取 %d", report.AttackerStolen)
	}
	st.Reports = append(st.Reports, report)
	if len(st.Reports) > 16 {
		st.Reports = st.Reports[len(st.Reports)-16:]
	}
	return report
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
		Vaults:      map[string]*vault{},
		EOA:         map[string]int64{},
		Tick:        fw.MapInt(d, "tick", 0),
		GuardBlocks: fw.MapInt(d, "guard_blocks", 0),
		CEIBlocks:   fw.MapInt(d, "cei_blocks", 0),
		StolenTotal: int64(fw.MapInt(d, "stolen_total", 0)),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if vAny, ok := d["vaults"].(map[string]any); ok {
		for k, vAny2 := range vAny {
			if vm, ok := vAny2.(map[string]any); ok {
				v := &vault{Kind: k, Balance: int64(fw.MapInt(vm, "balance", 0)),
					Locked: fw.MapBool(vm, "locked", false), Deposits: map[string]int64{}}
				if depAny, ok := vm["deposits"].(map[string]any); ok {
					for u, vv := range depAny {
						v.Deposits[u] = int64(intFromAny(vv))
					}
				}
				st.Vaults[k] = v
			}
		}
	}
	if len(st.Vaults) == 0 {
		return defaultSnapState()
	}
	if eAny, ok := d["eoa"].(map[string]any); ok {
		for k, vv := range eAny {
			st.EOA[k] = int64(intFromAny(vv))
		}
	}
	if rAny, ok := d["reports"].([]any); ok {
		for _, x := range rAny {
			if m, ok := x.(map[string]any); ok {
				rp := runReport{
					Variant:            fw.MapStr(m, "variant", ""),
					StackDepthMax:      fw.MapInt(m, "stack_depth", 0),
					ReentrancyDepthMax: fw.MapInt(m, "re_depth", 0),
					VaultBefore:        int64(fw.MapInt(m, "vault_before", 0)),
					VaultAfter:         int64(fw.MapInt(m, "vault_after", 0)),
					AttackerStolen:     int64(fw.MapInt(m, "stolen", 0)),
					GuardBlocks:        fw.MapInt(m, "guard", 0),
					CEIBlocks:          fw.MapInt(m, "cei", 0),
					Success:            fw.MapBool(m, "success", false),
					Detail:             fw.MapStr(m, "detail", ""),
				}
				if tAny, ok := m["trace"].([]any); ok {
					for _, fAny := range tAny {
						if fm, ok := fAny.(map[string]any); ok {
							rp.Trace = append(rp.Trace, callFrame{
								Depth:  fw.MapInt(fm, "depth", 0),
								Caller: fw.MapStr(fm, "caller", ""), Callee: fw.MapStr(fm, "callee", ""),
								Fn: fw.MapStr(fm, "fn", ""), Value: int64(fw.MapInt(fm, "value", 0)),
								MsgSender: fw.MapStr(fm, "sender", ""),
								Reverted:  fw.MapBool(fm, "rev", false), RevertNote: fw.MapStr(fm, "rev_note", ""),
								Note: fw.MapStr(fm, "note", ""),
							})
						}
					}
				}
				st.Reports = append(st.Reports, rp)
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["guard_blocks"] = st.GuardBlocks
	s.Data["cei_blocks"] = st.CEIBlocks
	s.Data["stolen_total"] = int(st.StolenTotal)
	s.Data["last_error"] = st.LastError
	vAny := map[string]any{}
	for k, v := range st.Vaults {
		dep := map[string]any{}
		for u, vv := range v.Deposits {
			dep[u] = int(vv)
		}
		vAny[k] = map[string]any{"balance": int(v.Balance), "locked": v.Locked, "deposits": dep}
	}
	s.Data["vaults"] = vAny
	eAny := map[string]any{}
	for k, vv := range st.EOA {
		eAny[k] = int(vv)
	}
	s.Data["eoa"] = eAny
	rAny := make([]any, len(st.Reports))
	for i, rp := range st.Reports {
		ts := make([]any, len(rp.Trace))
		for j, f := range rp.Trace {
			ts[j] = map[string]any{
				"depth": f.Depth, "caller": f.Caller, "callee": f.Callee,
				"fn": f.Fn, "value": int(f.Value), "sender": f.MsgSender,
				"rev": f.Reverted, "rev_note": f.RevertNote, "note": f.Note,
			}
		}
		rAny[i] = map[string]any{
			"variant": rp.Variant, "stack_depth": rp.StackDepthMax,
			"re_depth": rp.ReentrancyDepthMax, "vault_before": int(rp.VaultBefore),
			"vault_after": int(rp.VaultAfter), "stolen": int(rp.AttackerStolen),
			"guard": rp.GuardBlocks, "cei": rp.CEIBlocks,
			"success": rp.Success, "detail": rp.Detail, "trace": ts,
		}
	}
	s.Data["reports"] = rAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "重入攻击（The DAO）",
		Description:         "演示 vulnerable / CEI / Guard 三种 vault 在重入攻击下的表现；EVM 风格调用栈递归",
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
			"attack.reentrancy.stolen_total",
			"attack.reentrancy.guard_blocks",
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
				ActionCode: "deposit", Label: "EOA → Vault 存款",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "eoa", Type: fw.FieldEnum, Label: "from", Required: true, Default: "alice",
						Options: []any{"alice", "attacker"}},
					{Name: "vault", Type: fw.FieldEnum, Label: "vault", Required: true, Default: vaultVulnerable,
						Options: []any{vaultVulnerable, vaultSecureCEI, vaultWithGuard}},
					{Name: "amount", Type: fw.FieldNumber, Label: "金额", Required: true, Default: 100, Min: 1, Step: 10},
				},
			},
			{
				ActionCode: "run_attack_vulnerable", Label: "演示：攻击 vulnerable vault",
				Description: "DAO 风格 reentrancy → 抽干 vault",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"attack.reentrancy.stolen_total"},
				LinkOwnerFields:   []string{"attack.reentrancy.stolen_total"},
			},
			{
				ActionCode: "run_attack_cei", Label: "演示：攻击 CEI vault",
				Description: "checks-effects-interactions 顺序正确 → 重入失败",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"attack.reentrancy.stolen_total"},
			},
			{
				ActionCode: "run_attack_guard", Label: "演示：攻击 Guard vault",
				Description: "ReentrancyGuard 锁 → 重入被拦截",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"attack.reentrancy.guard_blocks"},
				LinkOwnerFields:   []string{"attack.reentrancy.guard_blocks"},
			},
			{
				ActionCode: "run_no_reenter", Label: "对照：单次 withdraw（无重入）",
				Description: "attacker fallback 不重入；展示正常单次提现",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "vault", Type: fw.FieldEnum, Label: "vault", Required: true, Default: vaultVulnerable,
						Options: []any{vaultVulnerable, vaultSecureCEI, vaultWithGuard}},
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
			{
				ActionCode:    "replay_poc",
				Label:         "重放 PoC（patch-verifier）",
				Description:   "调 patch-verifier 重放重入攻击 PoC",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
	env := buildEnvelope(st, "init", "三 Vault 初始化（each balance=1000；attacker 已 deposit=100）", true)
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
	case "deposit":
		eoa := fw.MapStr(in.Params, "eoa", "alice")
		vname := fw.MapStr(in.Params, "vault", vaultVulnerable)
		amt := int64(fw.MapInt(in.Params, "amount", 100))
		if err := st.directDeposit(eoa, vname, amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "deposit",
			fmt.Sprintf("%s 向 %s 存入 %d", eoa, vname, amt), false)
		appendDepositMicroSteps(&out.Render)
		return out, nil

	case "run_attack_vulnerable":
		rp := st.runAttack(vaultVulnerable, true)
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_attack_vulnerable", rp.Detail, false)
		appendAttackMicroSteps(&out.Render, rp)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "run_attack_cei":
		rp := st.runAttack(vaultSecureCEI, true)
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_attack_cei", rp.Detail, false)
		appendAttackMicroSteps(&out.Render, rp)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "run_attack_guard":
		rp := st.runAttack(vaultWithGuard, true)
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_attack_guard", rp.Detail, false)
		appendAttackMicroSteps(&out.Render, rp)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "run_no_reenter":
		vname := fw.MapStr(in.Params, "vault", vaultVulnerable)
		rp := st.runAttack(vname, false)
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_no_reenter", rp.Detail, false)
		appendAttackMicroSteps(&out.Render, rp)
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

	// 1) 三 Vault 节点 + Attacker
	prims = append(prims, fw.PrimNodeAt("eoa-attacker",
		fmt.Sprintf("attacker\nbal=%d\nstolen_total=%d", st.EOA["attacker"], st.StolenTotal),
		"active", "attacker", 0.5, 0.85, 1.4))
	xs := []float64{0.18, 0.5, 0.82}
	roles := []string{"vault-vulnerable", "vault-cei", "vault-guard"}
	names := []string{vaultVulnerable, vaultSecureCEI, vaultWithGuard}
	for i, n := range names {
		v := st.Vaults[n]
		prims = append(prims, fw.PrimNodeAt("v-"+n,
			fmt.Sprintf("%s\nbalance=%d\ndep[atk]=%d", n, v.Balance, v.Deposits["attacker"]),
			"active", roles[i], xs[i], 0.25, 1.2))
	}
	for _, n := range names {
		prims = append(prims, fw.PrimEdge("e-"+n, "eoa-attacker", "v-"+n, "solid", "flow"))
	}

	// 2) 公式 / 顺序对照
	prims = append(prims, fw.PrimMathFormula("formula-vuln",
		`\textbf{vulnerable: }\underbrace{\text{check}}_{1} \to \underbrace{\text{interaction}}_{2} \to \underbrace{\text{effect}}_{3}\quad\Rightarrow\text{ reenter 时 effect 还没执行}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-cei",
		`\textbf{secure CEI: }\text{check} \to \text{effect} \to \text{interaction}\quad\Rightarrow\text{ reenter 时 deposit=0，提不出钱}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-guard",
		`\textbf{guard: }\text{nonReentrant: } \neg\text{locked} \to \text{locked=true; ... ; locked=false}`, false))

	// 3) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("vault_vulnerable.balance = %d\nvault_secure_cei.balance = %d\nvault_with_guard.balance = %d\nattacker EOA balance = %d\nstolen_total = %d\nguard_blocks = %d\ncei_blocks = %d\nreports = %d",
			st.Vaults[vaultVulnerable].Balance,
			st.Vaults[vaultSecureCEI].Balance,
			st.Vaults[vaultWithGuard].Balance,
			st.EOA["attacker"], st.StolenTotal,
			st.GuardBlocks, st.CEIBlocks, len(st.Reports)),
		"text", nil, 8))

	// 4) Vault 余额对照表 + 饼图
	bLines := []string{"vault             balance  dep[alice]  dep[attacker]  locked"}
	for _, n := range names {
		v := st.Vaults[n]
		bLines = append(bLines, fmt.Sprintf("  %-16s  %-7d  %-10d  %-13d  %v",
			n, v.Balance, v.Deposits["alice"], v.Deposits["attacker"], v.Locked))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-vaults", strings.Join(bLines, "\n"), "text", nil, 6))

	// 5) 余额饼图
	prims = append(prims, fw.PrimPieChart("balance-pie", []map[string]any{
		{"label": "vulnerable", "value": float64(st.Vaults[vaultVulnerable].Balance), "color_role": "danger"},
		{"label": "secure_cei", "value": float64(st.Vaults[vaultSecureCEI].Balance), "color_role": "success"},
		{"label": "with_guard", "value": float64(st.Vaults[vaultWithGuard].Balance), "color_role": "info"},
	}))

	// 6) 最近一次 attack 的调用栈展开
	if len(st.Reports) > 0 {
		rp := st.Reports[len(st.Reports)-1]
		ids := make([]string, len(rp.Trace))
		for i := range rp.Trace {
			ids[i] = fmt.Sprintf("frame-%d", i)
		}
		prims = append(prims, fw.PrimStack("call-stack", ids, "vertical"))
		for i, f := range rp.Trace {
			role := "frame-vault"
			if f.Callee == "attacker" {
				role = "frame-attacker"
			}
			status := "active"
			if f.Reverted {
				status = "error"
			}
			label := fmt.Sprintf("d=%d  %s.%s\nsender=%s value=%d\n%s",
				f.Depth, f.Callee, f.Fn, f.MsgSender, f.Value, f.Note)
			if f.Reverted {
				label += "\nREVERT: " + f.RevertNote
			}
			prims = append(prims, fw.PrimNode(fmt.Sprintf("frame-%d", i), label, status, role))
		}
		// trace 表
		tLines := []string{"depth caller     → callee             fn        sender     value  rev  note"}
		for _, f := range rp.Trace {
			rv := ""
			if f.Reverted {
				rv = "✗ " + f.RevertNote
			}
			tLines = append(tLines, fmt.Sprintf("  %-4d %-10s → %-18s %-9s %-9s %-5d  %-3s  %s",
				f.Depth, f.Caller, f.Callee, f.Fn, f.MsgSender, f.Value, rv, f.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-trace", strings.Join(tLines, "\n"), "text", nil, 18))
	}

	// 7) 报告表
	if len(st.Reports) > 0 {
		rLines := []string{"variant            stolen  vault_before→after  re_depth  guard  cei  success  detail"}
		startIdx := 0
		if len(st.Reports) > 8 {
			startIdx = len(st.Reports) - 8
		}
		for _, rp := range st.Reports[startIdx:] {
			ok := "✗"
			if rp.Success {
				ok = "✓"
			}
			rLines = append(rLines, fmt.Sprintf("  %-17s %-6d  %-5d→%-5d         %-8d  %-5d  %-3d  %s        %s",
				rp.Variant, rp.AttackerStolen, rp.VaultBefore, rp.VaultAfter,
				rp.ReentrancyDepthMax, rp.GuardBlocks, rp.CEIBlocks, ok, rp.Detail))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-reports", strings.Join(rLines, "\n"), "text", nil, 12))
	}

	// 8) 进度条
	prims = append(prims, fw.PrimBar("bar-stolen", float64(st.StolenTotal), 0, "danger", "Stolen Total"))
	prims = append(prims, fw.PrimBar("bar-guard", float64(st.GuardBlocks), 0, "success", "Guard Blocked"))
	prims = append(prims, fw.PrimBar("bar-cei", float64(st.CEIBlocks), 0, "success", "CEI Safety Wins"))
	if len(st.Reports) > 0 {
		rp := st.Reports[len(st.Reports)-1]
		prims = append(prims, fw.PrimProgressBar("bar-depth",
			float64(rp.ReentrancyDepthMax), float64(maxAttackDepth*2),
			fmt.Sprintf("Reentrancy depth %d / %d", rp.ReentrancyDepthMax, maxAttackDepth*2)))
	}

	// 9) 动效
	if st.StolenTotal > 0 {
		prims = append(prims, fw.PrimShake("shake-stolen", "v-"+vaultVulnerable, 0.6, 800))
		prims = append(prims, fw.PrimBurst("burst-stolen", "eoa-attacker", "danger", st.StolenTotal, 700))
		prims = append(prims, fw.PrimPulse("pulse-stolen", "bar-stolen", "danger", 1500))
	}
	if st.GuardBlocks > 0 {
		prims = append(prims, fw.PrimGlow("glow-guard", "v-"+vaultWithGuard, "success", 0.9))
	}
	if st.CEIBlocks > 0 {
		prims = append(prims, fw.PrimGlow("glow-cei", "v-"+vaultSecureCEI, "success", 0.9))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Reentrancy 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "patch-verifier", MetricKey: "reentrancy.stolen_total", Value: st.StolenTotal, TargetPrimitive: "cb-poc", TargetParam: "stolen"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"vault_vulnerable": st.Vaults[vaultVulnerable].Balance,
		"vault_secure_cei": st.Vaults[vaultSecureCEI].Balance,
		"vault_with_guard": st.Vaults[vaultWithGuard].Balance,
		"stolen_total":     st.StolenTotal,
		"guard_blocks":     st.GuardBlocks,
		"cei_blocks":       st.CEIBlocks,
		"reports":          len(st.Reports),
		"tick":             st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendDepositMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "d-1", Label: "EOA → Vault.deposit()", DurationMs: 400, HighlightIDs: []string{"cb-vaults"}},
		{ID: "d-2", Label: "更新 vault.balance + deposits[eoa]", DurationMs: 400, HighlightIDs: []string{"balance-pie", "cb-status"}, IsLinkTrigger: true},
	}
}

func appendAttackMicroSteps(env *fw.RenderEnvelope, rp runReport) {
	header := "attacker.callVault.withdraw()"
	tail := rp.Detail
	highlight := "v-" + rp.Variant
	steps := []fw.MicroStep{
		{ID: "a-1", Label: header, DurationMs: 400, HighlightIDs: []string{highlight, "call-stack", "cb-trace"}},
	}
	switch rp.Variant {
	case vaultVulnerable:
		steps = append(steps,
			fw.MicroStep{ID: "a-2", Label: "[VULN] check → interaction → effect 错序", DurationMs: 500, HighlightIDs: []string{"formula-vuln"}},
			fw.MicroStep{ID: "a-3", Label: "vault 调用 attacker.fallback() 触发 RE-ENTRY", DurationMs: 500, HighlightIDs: []string{"call-stack", "bar-depth"}, FirePrimitives: []string{"shake-stolen"}},
			fw.MicroStep{ID: "a-4", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-stolen", "balance-pie"}, FirePrimitives: []string{"burst-stolen", "pulse-stolen"}, IsLinkTrigger: true},
		)
	case vaultSecureCEI:
		steps = append(steps,
			fw.MicroStep{ID: "a-2", Label: "[CEI] effect 先扣账", DurationMs: 500, HighlightIDs: []string{"formula-cei"}, FirePrimitives: []string{"glow-cei"}},
			fw.MicroStep{ID: "a-3", Label: "interaction 后转账，attacker.fallback 重入", DurationMs: 400, HighlightIDs: []string{"cb-trace"}},
			fw.MicroStep{ID: "a-4", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-cei"}, IsLinkTrigger: true},
		)
	case vaultWithGuard:
		steps = append(steps,
			fw.MicroStep{ID: "a-2", Label: "[GUARD] locked=true（首次 withdraw）", DurationMs: 400, HighlightIDs: []string{"formula-guard"}, FirePrimitives: []string{"glow-guard"}},
			fw.MicroStep{ID: "a-3", Label: "重入时 require(!locked) 失败 → REVERT", DurationMs: 500, HighlightIDs: []string{"call-stack", "cb-trace"}},
			fw.MicroStep{ID: "a-4", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-guard"}, IsLinkTrigger: true},
		)
	default:
		steps = append(steps,
			fw.MicroStep{ID: "a-2", Label: "无重入：attacker 仅取一次", DurationMs: 400, HighlightIDs: []string{"cb-trace"}},
			fw.MicroStep{ID: "a-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-stolen"}, IsLinkTrigger: true},
		)
	}
	env.MicroSteps = steps
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "reentrancy-attack",
		SourceScene:    sceneCode,
		SourceAction:   "run_reenter",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"attack.reentrancy.stolen_total"},
		Payload:        map[string]any{"stolen_total": st.StolenTotal},
		SourceAnchorID: "reentrancy-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"attack": map[string]any{
			"reentrancy": map[string]any{
				"vault_vulnerable": int(st.Vaults[vaultVulnerable].Balance),
				"vault_secure_cei": int(st.Vaults[vaultSecureCEI].Balance),
				"vault_with_guard": int(st.Vaults[vaultWithGuard].Balance),
				"stolen_total":     int(st.StolenTotal),
				"guard_blocks":     st.GuardBlocks,
				"cei_blocks":       st.CEIBlocks,
				"reports":          len(st.Reports),
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
