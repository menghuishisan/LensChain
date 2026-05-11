// 模块：sim-engine/scenarios/internal/smartcontract/contractinteraction
// 文件职责：SC-03 合约间调用场景的完整实现（call / delegatecall / staticcall）。
//
// SSOT 依据：06.md §4.6.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 EVM 风格合约间调用模型（零外部依赖）。
//
//   1. 三种调用语义：
//      · call         : 目标合约在 **callee** 上下文执行；
//                       msg.sender = caller；可修改 callee.storage；可携带 value。
//      · delegatecall : 目标合约 **代码** 在 **caller** 上下文执行；
//                       msg.sender / msg.value 保持外层调用的值；修改 **caller**.storage。
//      · staticcall   : 目标合约只读执行；任何 SSTORE / 余额变动 → REVERT；
//                       由其发起的所有嵌套调用都"传染"为只读。
//
//   2. 调用栈：完整 frame，含 depth / msg.sender / msg.value / tx.origin /
//      static_flag / storage_ctx；最大深度 1024（EVM 规则）；超出 → REVERT。
//
//   3. 合约模型：
//      · 多函数 ABI：setX / addX / readX / transfer(to, amt) /
//                    withdraw(amt) / forward(target, type, fn, args)
//        — forward 是教学版"代理函数"，可发起子调用，从而构成嵌套链。
//      · 余额转账：msg.value 在 call 中真实从 caller→callee 转账；
//                  delegatecall / staticcall 不允许 value > 0。
//      · 全局只读传染：进入 staticcall 后，任何嵌套调用都强制 static。
//
//   4. 教学对照：
//      · A.call(B.setX(42))         → B.x 被改、A.x 不变
//      · A.delegatecall(B.setX(42)) → A.x 被改、B.x 不变
//      · A.staticcall(B.setX(42))   → REVERT（不许 SSTORE）
//      · A.staticcall(B.forward(C.setX(7))) → 也 REVERT（传染）

package contractinteraction

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "contract-interaction"
	schemaVersion = "v1.0.0"
	algorithmType = "contract-call"

	callTypeCall         = "call"
	callTypeDelegateCall = "delegatecall"
	callTypeStaticCall   = "staticcall"

	maxCallDepth = 1024 // 与 EVM 一致

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "contract.interaction"
)

// =====================================================================
// 数据结构
// =====================================================================

// contractObj 一个合约账户。
type contractObj struct {
	Address string
	// Code 简化为函数白名单：合约支持哪些函数。
	// 真实 EVM 是字节码，这里教学版用"函数集"模拟。
	Code    map[string]bool
	Storage map[string]int64
	Balance int64
}

// callFrame 一次调用的栈帧。
type callFrame struct {
	Depth      int
	CallType   string // call / delegatecall / staticcall
	CodeAddr   string // 执行代码的合约（callee）
	StorageCtx string // 修改 storage 的合约（call/static→callee, delegate→上层 storage_ctx）
	MsgSender  string // 当前 frame 看到的 msg.sender
	MsgValue   int64  // 当前 frame 看到的 msg.value
	TxOrigin   string // 整个调用链最外层的发起者（EOA）
	IsStatic   bool   // 是否处于 staticcall 上下文（含传染）
	FuncName   string
	Args       string

	// 调用结果（execute 完后回填）
	ReturnValue  int64
	Reverted     bool
	RevertReason string
	GasUsed      int
}

// snapState 场景持久化状态。
type snapState struct {
	Contracts     map[string]*contractObj
	EOABalance    map[string]int64 // 外部账户余额（用于发起 tx）
	History       []callFrame      // 顶层调用历史（含其下的嵌套子帧）
	CurrentTrace  []callFrame      // 最近一次调用展开后的全部 frame（含子帧）
	StaticAttacks int              // staticcall 拦截 SSTORE 次数
	DepthExceeded int              // 深度超限次数
	Tick          int
	LastError     string
}

func defaultSnapState() snapState {
	st := snapState{
		Contracts:  map[string]*contractObj{},
		EOABalance: map[string]int64{"eoa": 100000},
	}
	allFns := map[string]bool{
		"setX": true, "addX": true, "readX": true,
		"transfer": true, "withdraw": true, "forward": true,
	}
	st.Contracts["A"] = &contractObj{Address: "A", Code: copyMap(allFns), Storage: map[string]int64{"x": 100}, Balance: 1000}
	st.Contracts["B"] = &contractObj{Address: "B", Code: copyMap(allFns), Storage: map[string]int64{"x": 200}, Balance: 500}
	st.Contracts["C"] = &contractObj{Address: "C", Code: copyMap(allFns), Storage: map[string]int64{"x": 999}, Balance: 0}
	return st
}

func copyMap(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// =====================================================================
// 调用执行核心
// =====================================================================

// invokeFromEOA 由 EOA 发起一次顶层调用，构造完整 trace。
func (st *snapState) invokeFromEOA(origin, callee, ct, fn string, args []int64, value int64) ([]callFrame, error) {
	if _, ok := st.Contracts[callee]; !ok {
		return nil, fmt.Errorf("合约不存在: %s", callee)
	}
	if ct != callTypeCall && ct != callTypeDelegateCall && ct != callTypeStaticCall {
		return nil, fmt.Errorf("未知调用类型: %s", ct)
	}
	// 顶层调用的 storage 上下文规则
	storageCtx := callee
	if ct == callTypeDelegateCall {
		// EOA → delegatecall 的 storage_ctx 应是 EOA，但 EOA 没 storage；
		// 教学简化：拒绝 EOA 直接 delegatecall（与 Solidity 高级 call 一致）。
		return nil, errors.New("EOA 不能直接 delegatecall（无 storage 上下文）")
	}
	if ct == callTypeStaticCall && value != 0 {
		return nil, errors.New("staticcall 不允许 value > 0")
	}
	// 余额预检
	if ct == callTypeCall && value > 0 {
		if st.EOABalance[origin] < value {
			return nil, fmt.Errorf("EOA 余额不足: %d < %d", st.EOABalance[origin], value)
		}
	}

	trace := []callFrame{}
	frame := callFrame{
		Depth: 1, CallType: ct, CodeAddr: callee, StorageCtx: storageCtx,
		MsgSender: origin, MsgValue: value, TxOrigin: origin,
		IsStatic: ct == callTypeStaticCall,
		FuncName: fn, Args: argsToStr(fn, args),
	}
	// 真正转账（call 路径）
	if ct == callTypeCall && value > 0 {
		st.EOABalance[origin] -= value
		st.Contracts[callee].Balance += value
	}
	st.executeFrame(&frame, args, &trace)
	// 回滚处理
	if frame.Reverted {
		// call 顶层 REVERT：退还转账
		if ct == callTypeCall && value > 0 {
			st.EOABalance[origin] += value
			st.Contracts[callee].Balance -= value
		}
	}
	trace = append([]callFrame{frame}, trace...) // 顶层在前
	st.History = append(st.History, frame)
	if len(st.History) > 16 {
		st.History = st.History[len(st.History)-16:]
	}
	st.CurrentTrace = trace
	return trace, nil
}

// executeFrame 在给定 frame 上下文中执行函数，可能嵌套发起子调用。
// 子帧追加进 trace。
func (st *snapState) executeFrame(f *callFrame, args []int64, trace *[]callFrame) {
	codeC, ok := st.Contracts[f.CodeAddr]
	if !ok {
		f.Reverted = true
		f.RevertReason = "code 合约不存在"
		return
	}
	if !codeC.Code[f.FuncName] {
		f.Reverted = true
		f.RevertReason = "未知函数: " + f.FuncName
		return
	}
	storageOwner := st.Contracts[f.StorageCtx]
	if storageOwner == nil {
		f.Reverted = true
		f.RevertReason = "storage_ctx 合约不存在: " + f.StorageCtx
		return
	}
	f.GasUsed = 100 // 基础 gas

	switch f.FuncName {
	case "setX":
		if f.IsStatic {
			f.Reverted = true
			f.RevertReason = "staticcall 中禁止 SSTORE (setX)"
			st.StaticAttacks++
			return
		}
		if len(args) < 1 {
			f.Reverted = true
			f.RevertReason = "setX 需要 1 个参数"
			return
		}
		storageOwner.Storage["x"] = args[0]
		f.GasUsed += 20000
		f.ReturnValue = args[0]

	case "addX":
		if f.IsStatic {
			f.Reverted = true
			f.RevertReason = "staticcall 中禁止 SSTORE (addX)"
			st.StaticAttacks++
			return
		}
		if len(args) < 1 {
			f.Reverted = true
			f.RevertReason = "addX 需要 1 个参数"
			return
		}
		storageOwner.Storage["x"] += args[0]
		f.GasUsed += 5000
		f.ReturnValue = storageOwner.Storage["x"]

	case "readX":
		f.GasUsed += 200
		f.ReturnValue = storageOwner.Storage["x"]

	case "transfer":
		if f.IsStatic {
			f.Reverted = true
			f.RevertReason = "staticcall 中禁止余额变动 (transfer)"
			st.StaticAttacks++
			return
		}
		if len(args) < 2 {
			f.Reverted = true
			f.RevertReason = "transfer(to, amount) 需 2 个参数"
			return
		}
		toAddr := fmt.Sprintf("acc%d", args[0]) // 简化：参数 0 → 账户 ID
		amount := args[1]
		// transfer 是从 storage_ctx 合约的 balance 转账（与 EVM 中 address(this).balance 一致）
		bal := storageOwner.Balance
		if amount <= 0 {
			f.Reverted = true
			f.RevertReason = "transfer amount 必须 > 0"
			return
		}
		if bal < amount {
			f.Reverted = true
			f.RevertReason = fmt.Sprintf("余额不足: %d < %d", bal, amount)
			return
		}
		storageOwner.Balance -= amount
		// 教学简化：toAddr 进入 EOA pool
		st.EOABalance[toAddr] += amount
		f.GasUsed += 9000
		f.ReturnValue = amount

	case "withdraw":
		if f.IsStatic {
			f.Reverted = true
			f.RevertReason = "staticcall 中禁止余额变动 (withdraw)"
			st.StaticAttacks++
			return
		}
		if len(args) < 1 {
			f.Reverted = true
			f.RevertReason = "withdraw(amount) 需 1 个参数"
			return
		}
		amount := args[0]
		if storageOwner.Balance < amount {
			f.Reverted = true
			f.RevertReason = "余额不足"
			return
		}
		storageOwner.Balance -= amount
		st.EOABalance[f.MsgSender] += amount
		f.GasUsed += 9000
		f.ReturnValue = amount

	case "forward":
		// forward(targetIdx, callTypeIdx, funcIdx, arg) — 教学：把数字参数映射成调用
		if len(args) < 4 {
			f.Reverted = true
			f.RevertReason = "forward(target, ctype, fnIdx, arg) 需 4 个参数"
			return
		}
		targetAddrs := []string{"A", "B", "C"}
		ctypes := []string{callTypeCall, callTypeDelegateCall, callTypeStaticCall}
		fns := []string{"setX", "addX", "readX"}
		ti := safeIdx(int(args[0]), len(targetAddrs))
		ci := safeIdx(int(args[1]), len(ctypes))
		fi := safeIdx(int(args[2]), len(fns))
		subTarget := targetAddrs[ti]
		subCT := ctypes[ci]
		subFn := fns[fi]
		subArg := args[3]

		// 计算子帧上下文
		if f.Depth+1 > maxCallDepth {
			f.Reverted = true
			f.RevertReason = fmt.Sprintf("call depth > %d", maxCallDepth)
			st.DepthExceeded++
			return
		}

		// staticcall 传染规则
		subStatic := f.IsStatic || subCT == callTypeStaticCall

		// 子帧 storage 上下文规则
		var subStorageCtx string
		var subSender string
		var subValue int64
		switch subCT {
		case callTypeCall, callTypeStaticCall:
			subStorageCtx = subTarget
			subSender = f.CodeAddr // 当前合约 = 子帧 msg.sender
			subValue = 0           // 教学简化：forward 子调用 value=0
		case callTypeDelegateCall:
			subStorageCtx = f.StorageCtx // 沿用上层 storage_ctx
			subSender = f.MsgSender      // delegatecall 保留 msg.sender
			subValue = f.MsgValue        // 保留 msg.value
		}

		sub := callFrame{
			Depth: f.Depth + 1, CallType: subCT, CodeAddr: subTarget,
			StorageCtx: subStorageCtx, MsgSender: subSender, MsgValue: subValue,
			TxOrigin: f.TxOrigin, IsStatic: subStatic,
			FuncName: subFn, Args: argsToStr(subFn, []int64{subArg}),
		}
		st.executeFrame(&sub, []int64{subArg}, trace)
		*trace = append(*trace, sub)
		if sub.Reverted {
			// 子帧 revert，整体 revert（教学版 forward 不做 try/catch）
			f.Reverted = true
			f.RevertReason = "嵌套 REVERT: " + sub.RevertReason
			return
		}
		f.GasUsed += 1000 + sub.GasUsed
		f.ReturnValue = sub.ReturnValue
	}
}

func safeIdx(i, n int) int {
	if n == 0 {
		return 0
	}
	if i < 0 {
		i = 0
	}
	return i % n
}

func argsToStr(fn string, args []int64) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%d", a)
	}
	return fn + "(" + strings.Join(parts, ", ") + ")"
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
		Contracts:     map[string]*contractObj{},
		EOABalance:    map[string]int64{},
		Tick:          fw.MapInt(d, "tick", 0),
		StaticAttacks: fw.MapInt(d, "static_attacks", 0),
		DepthExceeded: fw.MapInt(d, "depth_exceeded", 0),
		LastError:     fw.MapStr(d, "last_error", ""),
	}
	if csAny, ok := d["contracts"].(map[string]any); ok {
		for addr, cAny := range csAny {
			if cm, ok := cAny.(map[string]any); ok {
				c := &contractObj{Address: addr, Code: map[string]bool{}, Storage: map[string]int64{}, Balance: int64(fw.MapInt(cm, "balance", 0))}
				if codeAny, ok := cm["code"].([]any); ok {
					for _, x := range codeAny {
						if s, ok := x.(string); ok {
							c.Code[s] = true
						}
					}
				}
				if stoAny, ok := cm["storage"].(map[string]any); ok {
					for k, v := range stoAny {
						c.Storage[k] = int64(intFromAny(v))
					}
				}
				st.Contracts[addr] = c
			}
		}
	}
	if len(st.Contracts) == 0 {
		return defaultSnapState()
	}
	if eAny, ok := d["eoa_balance"].(map[string]any); ok {
		for k, v := range eAny {
			st.EOABalance[k] = int64(intFromAny(v))
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, fAny := range hAny {
			if fm, ok := fAny.(map[string]any); ok {
				st.History = append(st.History, decodeFrame(fm))
			}
		}
	}
	if tAny, ok := d["trace"].([]any); ok {
		for _, fAny := range tAny {
			if fm, ok := fAny.(map[string]any); ok {
				st.CurrentTrace = append(st.CurrentTrace, decodeFrame(fm))
			}
		}
	}
	return st
}

func decodeFrame(fm map[string]any) callFrame {
	return callFrame{
		Depth: fw.MapInt(fm, "depth", 0), CallType: fw.MapStr(fm, "type", ""),
		CodeAddr: fw.MapStr(fm, "code", ""), StorageCtx: fw.MapStr(fm, "storage_ctx", ""),
		MsgSender: fw.MapStr(fm, "sender", ""), MsgValue: int64(fw.MapInt(fm, "value", 0)),
		TxOrigin: fw.MapStr(fm, "origin", ""), IsStatic: fw.MapBool(fm, "static", false),
		FuncName: fw.MapStr(fm, "fn", ""), Args: fw.MapStr(fm, "args", ""),
		ReturnValue: int64(fw.MapInt(fm, "ret", 0)),
		Reverted:    fw.MapBool(fm, "rev", false), RevertReason: fw.MapStr(fm, "reason", ""),
		GasUsed: fw.MapInt(fm, "gas", 0),
	}
}

func encodeFrame(f callFrame) map[string]any {
	return map[string]any{
		"depth": f.Depth, "type": f.CallType,
		"code": f.CodeAddr, "storage_ctx": f.StorageCtx,
		"sender": f.MsgSender, "value": int(f.MsgValue),
		"origin": f.TxOrigin, "static": f.IsStatic,
		"fn": f.FuncName, "args": f.Args,
		"ret": int(f.ReturnValue),
		"rev": f.Reverted, "reason": f.RevertReason,
		"gas": f.GasUsed,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["static_attacks"] = st.StaticAttacks
	s.Data["depth_exceeded"] = st.DepthExceeded
	s.Data["last_error"] = st.LastError
	csAny := map[string]any{}
	for addr, c := range st.Contracts {
		fns := []any{}
		for k := range c.Code {
			fns = append(fns, k)
		}
		stoAny := map[string]any{}
		for k, v := range c.Storage {
			stoAny[k] = int(v)
		}
		csAny[addr] = map[string]any{"balance": int(c.Balance), "code": fns, "storage": stoAny}
	}
	s.Data["contracts"] = csAny
	eAny := map[string]any{}
	for k, v := range st.EOABalance {
		eAny[k] = int(v)
	}
	s.Data["eoa_balance"] = eAny
	hAny := make([]any, len(st.History))
	for i, f := range st.History {
		hAny[i] = encodeFrame(f)
	}
	s.Data["history"] = hAny
	tAny := make([]any, len(st.CurrentTrace))
	for i, f := range st.CurrentTrace {
		tAny[i] = encodeFrame(f)
	}
	s.Data["trace"] = tAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "合约间调用",
		Description:         "演示 call / delegatecall / staticcall + 嵌套调用栈 + msg.sender / msg.value / tx.origin + 余额转账 + staticcall 传染",
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
			"contract.interaction.last_call",
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
				ActionCode: "invoke", Label: "EOA 调用合约",
				Description:   "顶层 tx：origin → callee.fn(args) （call 可带 value）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "origin", Type: fw.FieldString, Label: "origin (EOA)", Required: true, Default: "eoa"},
					{Name: "callee", Type: fw.FieldEnum, Label: "callee", Required: true, Default: "A",
						Options: []any{"A", "B", "C"}},
					{Name: "call_type", Type: fw.FieldEnum, Label: "call type", Required: true, Default: callTypeCall,
						Options: []any{callTypeCall, callTypeStaticCall}},
					{Name: "func_name", Type: fw.FieldEnum, Label: "function", Required: true, Default: "setX",
						Options: []any{"setX", "addX", "readX", "transfer", "withdraw", "forward"}},
					{Name: "arg0", Type: fw.FieldNumber, Label: "arg0", Required: true, Default: 42, Step: 1},
					{Name: "arg1", Type: fw.FieldNumber, Label: "arg1 (transfer 用)", Required: false, Default: 0, Step: 1},
					{Name: "arg2", Type: fw.FieldNumber, Label: "arg2 (forward 用)", Required: false, Default: 0, Step: 1},
					{Name: "arg3", Type: fw.FieldNumber, Label: "arg3 (forward 用)", Required: false, Default: 0, Step: 1},
					{Name: "value", Type: fw.FieldNumber, Label: "value", Required: false, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"contract.interaction.last_call"},
				LinkOwnerFields:   []string{"contract.interaction.last_call"},
			},
			{
				ActionCode: "demo_call_vs_delegate", Label: "对照演示：call vs delegatecall",
				Description:   "用 forward 演示 A.call(B.setX) vs A.delegatecall(B.setX) 的 storage 区别",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "value", Type: fw.FieldNumber, Label: "setX 写入值", Required: true, Default: 777, Step: 1},
				},
			},
			{
				ActionCode: "demo_static_infect", Label: "演示：staticcall 传染",
				Description:     "EOA staticcall A.forward(B.setX(7))；嵌套 setX 应 REVERT",
				Category:        fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:   fw.InterveneAttack,
				LinkOwnerFields: []string{"contract.interaction.static_attacks"},
			},
			{
				ActionCode: "deep_chain", Label: "演示：调用链深度",
				Description:   "构造 N 层 forward 嵌套；超过 1024 应 REVERT",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "depth", Type: fw.FieldNumber, Label: "目标深度", Required: true, Default: 5, Min: 1, Max: 100, Step: 1},
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
			{
				ActionCode:    "invoke_method",
				Label:         "调用合约方法（真实链）",
				Description:   "调 geth eth_call / eth_sendTransaction",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"{{contract}}","data":"{{calldata}}"},"latest"],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "contract", Type: fw.FieldString, Label: "contract address", Required: true, Default: "0x"},
					{Name: "calldata", Type: fw.FieldString, Label: "calldata (hex)", Required: true, Default: "0x"},
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
	env := buildEnvelope(st, "init", "三合约 A/B/C + EOA 初始化", true)
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
	case "invoke":
		origin := fw.MapStr(in.Params, "origin", "eoa")
		callee := fw.MapStr(in.Params, "callee", "A")
		ct := fw.MapStr(in.Params, "call_type", callTypeCall)
		fn := fw.MapStr(in.Params, "func_name", "setX")
		args := []int64{
			int64(fw.MapInt(in.Params, "arg0", 0)),
			int64(fw.MapInt(in.Params, "arg1", 0)),
			int64(fw.MapInt(in.Params, "arg2", 0)),
			int64(fw.MapInt(in.Params, "arg3", 0)),
		}
		value := int64(fw.MapInt(in.Params, "value", 0))
		_, err := st.invokeFromEOA(origin, callee, ct, fn, args, value)
		if err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "invoke", err.Error(), false)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		top := st.History[len(st.History)-1]
		summary := summarizeInvocation(top)
		out.Render = buildEnvelope(st, "invoke", summary, false)
		appendInvokeMicroSteps(&out.Render, top)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "demo_call_vs_delegate":
		val := int64(fw.MapInt(in.Params, "value", 777))
		// Step 1: A.call(B.setX(val))   → forward(target=B(1), ctype=call(0), fn=setX(0), arg=val)
		st.invokeFromEOA("eoa", "A", callTypeCall, "forward", []int64{1, 0, 0, val}, 0)
		// Step 2: A.delegatecall(B.setX(val)) → 这里要从 EOA 调 A.forward 改 storage_ctx；
		// 由于 forward 的子调用 ctype=delegatecall 时会保留外层 storage_ctx=A，
		// 所以 setX 改的是 A.x（这就是教学要点）。
		st.invokeFromEOA("eoa", "A", callTypeCall, "forward", []int64{1, 1, 0, val}, 0)
		saveState(state, st)
		out.Render = buildEnvelope(st, "demo_call_vs_delegate",
			fmt.Sprintf("已演示 A.call(B.setX(%d)) vs A.delegatecall(B.setX(%d))", val, val), false)
		appendDemoMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "demo_static_infect":
		// EOA staticcall A.forward(B.setX(7))
		_, err := st.invokeFromEOA("eoa", "A", callTypeStaticCall, "forward", []int64{1, 0, 0, 7}, 0)
		_ = err
		saveState(state, st)
		out.Render = buildEnvelope(st, "demo_static_infect",
			"EOA staticcall A.forward(B.setX(7)) → setX 嵌套被传染拦截", false)
		appendStaticInfectMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "deep_chain":
		depth := fw.MapInt(in.Params, "depth", 5)
		// 构造 depth 层嵌套：通过反复触发 forward(target=A, ctype=call, fn=forward, arg=...) 是递归的，
		// 但 EVM 教学版需要静态展开。我们用直接 frame 模拟：
		st.simulateDeepChain("eoa", depth)
		saveState(state, st)
		summary := fmt.Sprintf("已模拟深度 = %d 的调用链", depth)
		if depth > maxCallDepth {
			summary = fmt.Sprintf("深度 %d 超过 %d → REVERT", depth, maxCallDepth)
		}
		out.Render = buildEnvelope(st, "deep_chain", summary, false)
		appendDeepChainMicroSteps(&out.Render, depth, depth > maxCallDepth)
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

// simulateDeepChain 直接生成一条 N 层 trace 用于教学（不真正执行 setX）。
func (st *snapState) simulateDeepChain(origin string, depth int) {
	trace := []callFrame{}
	for i := 1; i <= depth && i <= maxCallDepth; i++ {
		f := callFrame{
			Depth: i, CallType: callTypeCall, CodeAddr: pickAddr(i),
			StorageCtx: pickAddr(i), MsgSender: pickAddr(i - 1),
			MsgValue: 0, TxOrigin: origin, IsStatic: false,
			FuncName: "forward", Args: "forward(...)",
			ReturnValue: 0, Reverted: false, GasUsed: 1000,
		}
		if i-1 == 0 {
			f.MsgSender = origin
		}
		trace = append(trace, f)
	}
	if depth > maxCallDepth {
		// 超限帧
		f := callFrame{
			Depth: maxCallDepth + 1, CallType: callTypeCall,
			CodeAddr: pickAddr(maxCallDepth + 1), StorageCtx: pickAddr(maxCallDepth + 1),
			MsgSender: pickAddr(maxCallDepth), MsgValue: 0, TxOrigin: origin,
			FuncName: "forward", Args: "forward(...)",
			Reverted: true, RevertReason: fmt.Sprintf("call depth > %d", maxCallDepth),
		}
		trace = append(trace, f)
		st.DepthExceeded++
	}
	st.CurrentTrace = trace
	if len(trace) > 0 {
		st.History = append(st.History, trace[0])
		if len(st.History) > 16 {
			st.History = st.History[len(st.History)-16:]
		}
	}
}

func pickAddr(i int) string {
	addrs := []string{"A", "B", "C"}
	if i <= 0 {
		return addrs[0]
	}
	return addrs[i%len(addrs)]
}

func summarizeInvocation(top callFrame) string {
	tag := "✓"
	if top.Reverted {
		tag = "✗ REVERT: " + top.RevertReason
	}
	return fmt.Sprintf("%s.%s(%s) value=%d → ret=%d %s (storage→%s)",
		top.MsgSender, top.CallType+":"+top.CodeAddr, top.Args, top.MsgValue, top.ReturnValue, tag, top.StorageCtx)
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 50)

	// 1) 合约节点（环形）
	addrs := []string{}
	for a := range st.Contracts {
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)
	prims = append(prims, fw.PrimRingLayout("contract-ring", len(addrs)+1))
	prims = append(prims, fw.PrimNode("eoa", "EOA\nbal="+fmt.Sprintf("%d", st.EOABalance["eoa"]), "active", "eoa"))
	for _, a := range addrs {
		c := st.Contracts[a]
		label := fmt.Sprintf("%s\nbal=%d\nx=%d", a, c.Balance, c.Storage["x"])
		prims = append(prims, fw.PrimNode("c-"+a, label, "active", "contract"))
	}

	// 2) 调用栈（最近一次 trace）
	if len(st.CurrentTrace) > 0 {
		// 调用链流水线（按 depth 排序）
		ordered := append([]callFrame{}, st.CurrentTrace...)
		sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Depth < ordered[b].Depth })
		ids := []string{"frame-eoa"}
		for _, f := range ordered {
			ids = append(ids, fmt.Sprintf("frame-%d", f.Depth))
		}
		prims = append(prims, fw.PrimStack("call-stack", ids, "horizontal"))
		prims = append(prims, fw.PrimNode("frame-eoa", "EOA\norigin", "active", "eoa-frame"))
		for _, f := range ordered {
			role := "call-frame"
			status := "active"
			switch f.CallType {
			case callTypeDelegateCall:
				role = "delegate-frame"
			case callTypeStaticCall:
				role = "static-frame"
			}
			if f.Reverted {
				status = "error"
			}
			label := fmt.Sprintf("d=%d\n%s\n%s.%s\nctx=%s", f.Depth, f.CallType, f.CodeAddr, f.FuncName, f.StorageCtx)
			prims = append(prims, fw.PrimNode(fmt.Sprintf("frame-%d", f.Depth), label, status, role))
		}
		// 边
		prevID := "frame-eoa"
		for _, f := range ordered {
			id := fmt.Sprintf("frame-%d", f.Depth)
			style := "solid"
			anim := "flow"
			if f.Reverted {
				style = "dashed"
				anim = ""
			}
			prims = append(prims, fw.PrimEdge("e-"+id, prevID, id, style, anim))
			prevID = id
		}
	}

	// 3) 公式（语义对照）
	prims = append(prims, fw.PrimMathFormula("formula-call",
		`\begin{aligned}
\text{call}: &\ \text{ctx} = \text{callee},\ \text{sender} = \text{caller},\ \text{value 真实转账}\\
\text{delegatecall}: &\ \text{ctx} = \text{caller},\ \text{sender / value 沿用上层}\\
\text{staticcall}: &\ \text{ctx} = \text{callee},\ \text{SSTORE / 转账 \to REVERT (传染)}
\end{aligned}`, false))

	// 4) 合约状态表
	cLines := []string{"address  balance   storage.x   functions"}
	for _, a := range addrs {
		c := st.Contracts[a]
		fns := []string{}
		for k := range c.Code {
			fns = append(fns, k)
		}
		sort.Strings(fns)
		cLines = append(cLines, fmt.Sprintf("  %-6s  %-8d  %-10d  [%s]", a, c.Balance, c.Storage["x"], strings.Join(fns, ", ")))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-contracts", strings.Join(cLines, "\n"), "text", nil, 10))

	// 5) EOA 余额表
	eLines := []string{"EOA / receiver       balance"}
	keys := []string{}
	for k := range st.EOABalance {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		eLines = append(eLines, fmt.Sprintf("  %-18s  %d", k, st.EOABalance[k]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-eoa", strings.Join(eLines, "\n"), "text", nil, 10))

	// 6) 当前 trace 表
	if len(st.CurrentTrace) > 0 {
		ordered := append([]callFrame{}, st.CurrentTrace...)
		sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Depth < ordered[b].Depth })
		tLines := []string{"depth type           code   ctx    sender  value origin static fn(args)                ret    rev"}
		for _, f := range ordered {
			rev := ""
			if f.Reverted {
				rev = "REVERT: " + f.RevertReason
			}
			tLines = append(tLines, fmt.Sprintf("%-5d %-14s %-6s %-6s %-7s %-5d %-6s %-6v %-23s %-6d %s",
				f.Depth, f.CallType, f.CodeAddr, f.StorageCtx, f.MsgSender, f.MsgValue,
				f.TxOrigin, f.IsStatic, f.Args, f.ReturnValue, rev))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-trace", strings.Join(tLines, "\n"), "text", nil, 16))
	}

	// 7) 历史顶层调用
	if len(st.History) > 0 {
		hLines := []string{"# origin        call_type      callee.fn(args)       value  ret    storage_ctx  rev"}
		startIdx := 0
		if len(st.History) > 12 {
			startIdx = len(st.History) - 12
		}
		for i, f := range st.History[startIdx:] {
			rev := ""
			if f.Reverted {
				rev = "REVERT"
			}
			hLines = append(hLines, fmt.Sprintf("%-2d %-13s %-14s %-21s %-6d %-6d %-12s %s",
				i+startIdx, f.TxOrigin, f.CallType, f.CodeAddr+"."+f.Args,
				f.MsgValue, f.ReturnValue, f.StorageCtx, rev))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(hLines, "\n"), "text", nil, 14))
	}

	// 8) 状态参数 + 计数器
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("contracts = %d\ntrace_depth = %d\nstatic_attacks 拦截 = %d\ndepth_exceeded = %d\nhistory = %d",
			len(st.Contracts), len(st.CurrentTrace), st.StaticAttacks, st.DepthExceeded, len(st.History)),
		"text", nil, 6))

	// 9) 进度条
	prims = append(prims, fw.PrimBar("bar-static", float64(st.StaticAttacks), 0, "success", "Static-Call Blocked"))
	prims = append(prims, fw.PrimBar("bar-depth", float64(st.DepthExceeded), 0, "warning", "Depth Exceeded"))
	if len(st.CurrentTrace) > 0 {
		max := 0
		for _, f := range st.CurrentTrace {
			if f.Depth > max {
				max = f.Depth
			}
		}
		prims = append(prims, fw.PrimProgressBar("bar-cur-depth", float64(max), float64(maxCallDepth),
			fmt.Sprintf("Current call depth %d / %d", max, maxCallDepth)))
	}

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-eoa", "eoa", "info", 0.6))
	if len(st.CurrentTrace) > 0 {
		last := st.CurrentTrace[len(st.CurrentTrace)-1]
		if last.Reverted {
			prims = append(prims, fw.PrimShake("shake-rev", "cb-trace", 0.4, 700))
			prims = append(prims, fw.PrimPulse("pulse-rev", fmt.Sprintf("frame-%d", last.Depth), "danger", 1500))
		} else {
			prims = append(prims, fw.PrimBurst("burst-ok", "c-"+last.StorageCtx, "success", int64(st.Tick), 700))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Interaction 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "interaction.trace_depth", Value: len(st.CurrentTrace), TargetPrimitive: "cb-result", TargetParam: "depth"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"contract_count": len(st.Contracts),
		"trace_depth":    len(st.CurrentTrace),
		"static_attacks": st.StaticAttacks,
		"depth_exceeded": st.DepthExceeded,
		"history_count":  len(st.History),
		"tick":           st.Tick,
	}
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		d["last_call"] = summarizeInvocation(last)
		d["last_reverted"] = last.Reverted
		d["last_storage_ctx"] = last.StorageCtx
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendInvokeMicroSteps(env *fw.RenderEnvelope, top callFrame) {
	tail := fmt.Sprintf("修改 %s 的 storage / 余额", top.StorageCtx)
	if top.Reverted {
		tail = "REVERT，状态回滚"
	} else if top.FuncName == "readX" {
		tail = "只读，不修改 storage"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "i-1", Label: fmt.Sprintf("EOA %s 发起 %s", top.MsgSender, top.CallType), DurationMs: 400, HighlightIDs: []string{"eoa", "frame-eoa", "frame-1"}},
		{ID: "i-2", Label: fmt.Sprintf("跳入 %s.%s（msg.sender=%s, msg.value=%d, static=%v）",
			top.CodeAddr, top.FuncName, top.MsgSender, top.MsgValue, top.IsStatic), DurationMs: 500,
			HighlightIDs: []string{"formula-call", "cb-trace"}},
		{ID: "i-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-contracts"}, FirePrimitives: []string{"burst-ok", "pulse-rev"}, IsLinkTrigger: true},
	}
}

func appendDemoMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "d-1", Label: "Step 1: A.call(B.setX(v)) → 改的是 B.x", DurationMs: 500, HighlightIDs: []string{"c-B", "cb-contracts"}},
		{ID: "d-2", Label: "Step 2: A.delegatecall(B.setX(v)) → 改的是 A.x", DurationMs: 500, HighlightIDs: []string{"c-A", "cb-contracts"}, FirePrimitives: []string{"burst-ok"}},
		{ID: "d-3", Label: "对照：同样是 B 的 setX 代码，但 storage_ctx 不同", DurationMs: 500, HighlightIDs: []string{"formula-call", "cb-history"}, IsLinkTrigger: true},
	}
}

func appendStaticInfectMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "si-1", Label: "EOA staticcall A → frame.IsStatic=true", DurationMs: 400, HighlightIDs: []string{"frame-1"}},
		{ID: "si-2", Label: "A.forward 子调用继承 IsStatic（传染）", DurationMs: 500, HighlightIDs: []string{"frame-2"}, FirePrimitives: []string{"shake-rev"}},
		{ID: "si-3", Label: "嵌套 setX 触发 SSTORE → REVERT", DurationMs: 500, HighlightIDs: []string{"cb-trace", "bar-static"}, FirePrimitives: []string{"pulse-rev"}, IsLinkTrigger: true},
	}
}

func appendDeepChainMicroSteps(env *fw.RenderEnvelope, depth int, exceeded bool) {
	tail := fmt.Sprintf("调用链构造完成，深度 = %d", depth)
	if exceeded {
		tail = fmt.Sprintf("深度 %d > %d → 最深一层 REVERT", depth, maxCallDepth)
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "dc-1", Label: "依次推入 frame", DurationMs: 400, HighlightIDs: []string{"call-stack"}},
		{ID: "dc-2", Label: "msg.sender 沿调用链传递", DurationMs: 400, HighlightIDs: []string{"cb-trace"}},
		{ID: "dc-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-cur-depth", "bar-depth"}, IsLinkTrigger: true},
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
		ID:             "interaction-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_interaction",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"contract.interaction.last_call"},
		Payload:        map[string]any{"call_count": len(st.History)},
		SourceAnchorID: "interaction-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "contract.interaction.last_call")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	d := map[string]any{
		"contract_count": len(st.Contracts),
		"trace_depth":    len(st.CurrentTrace),
		"static_attacks": st.StaticAttacks,
		"depth_exceeded": st.DepthExceeded,
		"history_count":  len(st.History),
	}
	if len(st.History) > 0 {
		last := st.History[len(st.History)-1]
		d["last_call"] = fmt.Sprintf("%s.%s(%s)", last.CodeAddr, last.FuncName, last.Args)
		d["last_reverted"] = last.Reverted
		d["last_storage_ctx"] = last.StorageCtx
	}
	return map[string]any{"contract": map[string]any{"interaction": d}}
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
