// 模块：sim-engine/scenarios/internal/transaction/tokentransfer
// 文件职责：TX-03 代币转账（ERC-20 标准）场景的完整实现。
//
// SSOT 依据：06.md §4.5.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 ERC-20 状态机（无外部依赖）：
//   · balanceOf[address]：用户余额
//   · allowance[owner][spender]：授权额度
//   · totalSupply：总供应量
//   · 5 操作：mint / transfer / approve / transferFrom / burn
//   · 异常分支：余额不足、allowance 不足、转账给零地址等

package tokentransfer

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "token-transfer"
	schemaVersion = "v1.0.0"
	algorithmType = "erc20"

	maxAccounts = 12
	maxEvents   = 32

	linkGroupTxProc  = "tx-processing-group"
	linkOwnerSubtree = "tx.token"
)

type event struct {
	Tick    int
	Type    string // mint / transfer / approve / transfer_from / burn / fail
	From    string
	To      string
	Spender string
	Value   int64
	OK      bool
	Reason  string
}

type snapState struct {
	TokenName   string
	TokenSymbol string
	Decimals    int
	TotalSupply int64
	Balances    map[string]int64
	Allowances  map[string]map[string]int64 // owner → spender → amount
	Tick        int
	Owner       string
	Events      []event
	LastError   string
}

func defaultSnapState() snapState {
	st := snapState{
		TokenName:   "LensCoin",
		TokenSymbol: "LENS",
		Decimals:    18,
		Owner:       "owner",
		Balances:    map[string]int64{},
		Allowances:  map[string]map[string]int64{},
	}
	// 初始默认账户
	for _, a := range []string{"owner", "alice", "bob", "carol", "dave"} {
		st.Balances[a] = 0
	}
	return st
}

// mint owner 铸造代币给 to。
func (st *snapState) mint(to string, amount int64) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if to == "" {
		return errors.New("不能 mint 给零地址")
	}
	st.Balances[to] += amount
	st.TotalSupply += amount
	st.pushEvent(event{
		Tick: st.Tick, Type: "mint", From: "0x0", To: to, Value: amount, OK: true,
	})
	return nil
}

// transfer from → to。
func (st *snapState) transfer(from, to string, amount int64) error {
	if from == "" || to == "" {
		return errors.New("不能转账到/从零地址")
	}
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if st.Balances[from] < amount {
		st.pushEvent(event{
			Tick: st.Tick, Type: "fail", From: from, To: to, Value: amount,
			OK: false, Reason: fmt.Sprintf("余额不足 (%d < %d)", st.Balances[from], amount),
		})
		return fmt.Errorf("%s 余额不足 (%d < %d)", from, st.Balances[from], amount)
	}
	st.Balances[from] -= amount
	st.Balances[to] += amount
	st.pushEvent(event{
		Tick: st.Tick, Type: "transfer", From: from, To: to, Value: amount, OK: true,
	})
	return nil
}

// approve owner 授权 spender allowance amount。
func (st *snapState) approve(owner, spender string, amount int64) error {
	if owner == "" || spender == "" {
		return errors.New("零地址")
	}
	if st.Allowances[owner] == nil {
		st.Allowances[owner] = map[string]int64{}
	}
	st.Allowances[owner][spender] = amount
	st.pushEvent(event{
		Tick: st.Tick, Type: "approve", From: owner, Spender: spender, Value: amount, OK: true,
	})
	return nil
}

// transferFrom spender 把 owner 的 token 转给 to（消耗 allowance 与 owner 的 balance）。
func (st *snapState) transferFrom(spender, owner, to string, amount int64) error {
	if spender == "" || owner == "" || to == "" {
		return errors.New("零地址")
	}
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	allow := int64(0)
	if a, ok := st.Allowances[owner]; ok {
		allow = a[spender]
	}
	if allow < amount {
		st.pushEvent(event{
			Tick: st.Tick, Type: "fail", From: owner, To: to, Spender: spender, Value: amount,
			OK: false, Reason: fmt.Sprintf("allowance 不足 (%d < %d)", allow, amount),
		})
		return fmt.Errorf("allowance 不足 (%d < %d)", allow, amount)
	}
	if st.Balances[owner] < amount {
		st.pushEvent(event{
			Tick: st.Tick, Type: "fail", From: owner, To: to, Spender: spender, Value: amount,
			OK: false, Reason: fmt.Sprintf("owner 余额不足"),
		})
		return fmt.Errorf("%s 余额不足", owner)
	}
	st.Allowances[owner][spender] -= amount
	st.Balances[owner] -= amount
	st.Balances[to] += amount
	st.pushEvent(event{
		Tick: st.Tick, Type: "transfer_from", From: owner, To: to, Spender: spender, Value: amount, OK: true,
	})
	return nil
}

// burn 持有者销毁自己的 token。
func (st *snapState) burn(holder string, amount int64) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if st.Balances[holder] < amount {
		st.pushEvent(event{
			Tick: st.Tick, Type: "fail", From: holder, To: "0x0", Value: amount,
			OK: false, Reason: "余额不足",
		})
		return fmt.Errorf("%s 余额不足", holder)
	}
	st.Balances[holder] -= amount
	st.TotalSupply -= amount
	st.pushEvent(event{
		Tick: st.Tick, Type: "burn", From: holder, To: "0x0", Value: amount, OK: true,
	})
	return nil
}

func (st *snapState) pushEvent(e event) {
	st.Events = append(st.Events, e)
	if len(st.Events) > maxEvents {
		st.Events = st.Events[len(st.Events)-maxEvents:]
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
		TokenName:   fw.MapStr(d, "name", "LensCoin"),
		TokenSymbol: fw.MapStr(d, "symbol", "LENS"),
		Decimals:    fw.MapInt(d, "decimals", 18),
		TotalSupply: int64(fw.MapInt(d, "total_supply", 0)),
		Owner:       fw.MapStr(d, "owner", "owner"),
		Tick:        fw.MapInt(d, "tick", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
		Balances:    map[string]int64{},
		Allowances:  map[string]map[string]int64{},
	}
	if balsAny, ok := d["balances"].(map[string]any); ok {
		for k, v := range balsAny {
			st.Balances[k] = int64(intFromAny(v))
		}
	}
	if allAny, ok := d["allowances"].(map[string]any); ok {
		for owner, sm := range allAny {
			if smm, ok := sm.(map[string]any); ok {
				inner := map[string]int64{}
				for spender, v := range smm {
					inner[spender] = int64(intFromAny(v))
				}
				st.Allowances[owner] = inner
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, evAny := range eAny {
			if em, ok := evAny.(map[string]any); ok {
				st.Events = append(st.Events, event{
					Tick: fw.MapInt(em, "tick", 0),
					Type: fw.MapStr(em, "type", ""),
					From: fw.MapStr(em, "from", ""), To: fw.MapStr(em, "to", ""),
					Spender: fw.MapStr(em, "spender", ""),
					Value:   int64(fw.MapInt(em, "value", 0)),
					OK:      fw.MapBool(em, "ok", false),
					Reason:  fw.MapStr(em, "reason", ""),
				})
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["name"] = st.TokenName
	s.Data["symbol"] = st.TokenSymbol
	s.Data["decimals"] = st.Decimals
	s.Data["total_supply"] = int(st.TotalSupply)
	s.Data["owner"] = st.Owner
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	balsAny := map[string]any{}
	for k, v := range st.Balances {
		balsAny[k] = int(v)
	}
	s.Data["balances"] = balsAny
	allAny := map[string]any{}
	for owner, sm := range st.Allowances {
		inner := map[string]any{}
		for spender, v := range sm {
			inner[spender] = int(v)
		}
		allAny[owner] = inner
	}
	s.Data["allowances"] = allAny
	eAny := make([]any, len(st.Events))
	for i, e := range st.Events {
		eAny[i] = map[string]any{
			"tick": e.Tick, "type": e.Type,
			"from": e.From, "to": e.To, "spender": e.Spender,
			"value": int(e.Value), "ok": e.OK, "reason": e.Reason,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "代币转账（ERC-20）",
		Description:         "演示 ERC-20 标准：balanceOf / allowance / transfer / approve / transferFrom / mint / burn",
		Category:            fw.CategoryTransaction,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupTxProc},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"tx.token.total_supply",
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
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "mint", Label: "mint（铸造）",
				Description:   "owner 铸造 amount 给 to（增加 totalSupply）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "alice"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 1000, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.token.total_supply"},
				LinkOwnerFields:   []string{"tx.token.total_supply"},
			},
			{
				ActionCode: "transfer", Label: "transfer",
				Description:   "from 直接把 amount 转给 to",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "alice"},
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "bob"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 100, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.token.total_supply"},
				LinkOwnerFields:   []string{"tx.token.total_supply"},
			},
			{
				ActionCode: "approve", Label: "approve（授权）",
				Description:   "owner 授权 spender 一定额度",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "owner", Type: fw.FieldString, Label: "owner", Required: true, Default: "alice"},
					{Name: "spender", Type: fw.FieldString, Label: "spender", Required: true, Default: "bob"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 200, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "transfer_from", Label: "transferFrom（用授权）",
				Description:   "spender 调用，把 owner 的 token 转给 to（消耗 allowance + owner 余额）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "spender", Type: fw.FieldString, Label: "spender", Required: true, Default: "bob"},
					{Name: "owner", Type: fw.FieldString, Label: "owner", Required: true, Default: "alice"},
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "carol"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 50, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "burn", Label: "burn（销毁）",
				Description:   "持有者销毁自己的 token",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "holder", Type: fw.FieldString, Label: "holder", Required: true, Default: "alice"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 10, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.token.total_supply"},
				LinkOwnerFields:   []string{"tx.token.total_supply"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_freeze_mempool",
				Label:         "教师冻结内存池",
				Description:   "仅教师可用，冻结内存池用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师冻结内存池"},
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
	env := buildEnvelope(st, "init", "ERC-20 LensCoin 初始化（5 账户，0 余额）", true)
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
	st.Tick++

	switch in.ActionCode {
	case "mint":
		to := fw.MapStr(in.Params, "to", "alice")
		amt := int64(fw.MapInt(in.Params, "amount", 1000))
		if err := st.mint(to, amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mint", fmt.Sprintf("mint(%s, %d) → totalSupply=%d", to, amt, st.TotalSupply), false)
		appendMintMicroSteps(&out.Render, to, amt)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "transfer":
		from := fw.MapStr(in.Params, "from", "alice")
		to := fw.MapStr(in.Params, "to", "bob")
		amt := int64(fw.MapInt(in.Params, "amount", 100))
		if err := st.transfer(from, to, amt); err != nil {
			saveState(state, st) // 失败也保存 event
			out.Render = buildEnvelope(st, "transfer", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "transfer",
			fmt.Sprintf("%s → %s : %d %s", from, to, amt, st.TokenSymbol), false)
		appendTransferMicroSteps(&out.Render, from, to)
		return out, nil

	case "approve":
		owner := fw.MapStr(in.Params, "owner", "alice")
		spender := fw.MapStr(in.Params, "spender", "bob")
		amt := int64(fw.MapInt(in.Params, "amount", 200))
		if err := st.approve(owner, spender, amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "approve",
			fmt.Sprintf("approve(%s ↗ %s, %d)", owner, spender, amt), false)
		appendApproveMicroSteps(&out.Render, owner, spender)
		return out, nil

	case "transfer_from":
		spender := fw.MapStr(in.Params, "spender", "bob")
		owner := fw.MapStr(in.Params, "owner", "alice")
		to := fw.MapStr(in.Params, "to", "carol")
		amt := int64(fw.MapInt(in.Params, "amount", 50))
		if err := st.transferFrom(spender, owner, to, amt); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "transfer_from", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "transfer_from",
			fmt.Sprintf("transferFrom(spender=%s, %s → %s, %d)", spender, owner, to, amt), false)
		appendTransferFromMicroSteps(&out.Render, owner, to, spender)
		return out, nil

	case "burn":
		h := fw.MapStr(in.Params, "holder", "alice")
		amt := int64(fw.MapInt(in.Params, "amount", 10))
		if err := st.burn(h, amt); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "burn", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "burn",
			fmt.Sprintf("burn(%s, %d) → totalSupply=%d", h, amt, st.TotalSupply), false)
		appendBurnMicroSteps(&out.Render, h)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "teacher_freeze_mempool":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师冻结内存池"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-freeze-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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
	prims := make([]fw.Primitive, 0, 30)

	// 1) 合约 + 账户布局（环形）
	accs := []string{}
	for k := range st.Balances {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	prims = append(prims, fw.PrimRingLayout("account-ring", len(accs)))

	// 2) 合约节点
	prims = append(prims, fw.PrimNodeAt("contract",
		fmt.Sprintf("ERC-20 %s\nname=%s\nsupply=%d", st.TokenSymbol, st.TokenName, st.TotalSupply),
		"active", "contract", 0.5, 0.5, 1.5))

	// 3) 账户节点
	for _, a := range accs {
		role := "account"
		status := "normal"
		if a == st.Owner {
			role = "owner"
			status = "active"
		}
		bal := st.Balances[a]
		if bal > 0 {
			status = "active"
		}
		label := fmt.Sprintf("%s\n%d %s", a, bal, st.TokenSymbol)
		prims = append(prims, fw.PrimNode("acc-"+a, label, status, role))
	}

	// 4) 最近事件画 from→to 边
	if len(st.Events) > 0 {
		last := st.Events[len(st.Events)-1]
		if last.OK && last.From != "" && last.To != "" && last.From != "0x0" && last.To != "0x0" {
			anim := "flow"
			style := "solid"
			fromID := "acc-" + last.From
			toID := "acc-" + last.To
			if last.From == "0x0" {
				fromID = "contract"
			}
			if last.To == "0x0" {
				toID = "contract"
			}
			prims = append(prims, fw.PrimEdge("flow-edge", fromID, toID, style, anim))
		}
	}

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-erc20",
		`\text{transfer}(f,t,v):\ b[f]\ge v;\ b[f]\!-\!=v;\ b[t]\!+\!=v\\
\text{transferFrom}(s,o,t,v):\ a[o][s]\ge v\land b[o]\ge v`, false))

	// 6) 状态
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("name = %s\nsymbol = %s\ndecimals = %d\ntotalSupply = %d\nowner = %s\n账户数 = %d",
			st.TokenName, st.TokenSymbol, st.Decimals, st.TotalSupply, st.Owner, len(accs)),
		"text", nil, 8))

	// 7) 余额表
	balRows := []string{"账户       余额"}
	for _, a := range accs {
		balRows = append(balRows, fmt.Sprintf("  %-9s %d", a, st.Balances[a]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-balances", strings.Join(balRows, "\n"), "text", nil, 12))

	// 8) Allowance 表
	allRows := []string{"owner          spender         amount"}
	for owner, sm := range st.Allowances {
		spenders := make([]string, 0, len(sm))
		for sp := range sm {
			spenders = append(spenders, sp)
		}
		sort.Strings(spenders)
		for _, sp := range spenders {
			if sm[sp] > 0 {
				allRows = append(allRows, fmt.Sprintf("  %-12s   %-12s   %d", owner, sp, sm[sp]))
			}
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-allowances", strings.Join(allRows, "\n"), "text", nil, 12))

	// 9) Pie chart 余额分布
	if st.TotalSupply > 0 {
		segs := []map[string]any{}
		for _, a := range accs {
			if st.Balances[a] > 0 {
				segs = append(segs, map[string]any{
					"label":      a,
					"value":      float64(st.Balances[a]),
					"color_role": "info",
				})
			}
		}
		if len(segs) > 0 {
			prims = append(prims, fw.PrimPieChart("balance-pie", segs))
		}
	}

	// 10) 事件
	if len(st.Events) > 0 {
		eLines := []string{"事件日志（最近 16）："}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		for _, e := range st.Events[startIdx:] {
			ok := "✓"
			if !e.OK {
				ok = "✗"
			}
			line := fmt.Sprintf("  t=%d  %s [%s]  %s → %s : %d", e.Tick, ok, e.Type, e.From, e.To, e.Value)
			if e.Spender != "" {
				line += " (spender=" + e.Spender + ")"
			}
			if !e.OK {
				line += " // " + e.Reason
			}
			eLines = append(eLines, line)
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 18))
	}

	// 11) 动效
	prims = append(prims, fw.PrimGlow("glow-contract", "contract", "info", 0.7))
	if len(st.Events) > 0 && st.Events[len(st.Events)-1].OK {
		prims = append(prims, fw.PrimBurst("burst-tx", "contract", "success", int64(st.Tick), 700))
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Token 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	holders := 0
	for _, b := range st.Balances {
		if b > 0 {
			holders++
		}
	}
	d := map[string]any{
		"name":          st.TokenName,
		"symbol":        st.TokenSymbol,
		"decimals":      st.Decimals,
		"total_supply":  st.TotalSupply,
		"holders":       holders,
		"account_count": len(st.Balances),
		"event_count":   len(st.Events),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendMintMicroSteps(env *fw.RenderEnvelope, to string, amt int64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: "mint 调用", DurationMs: 400, HighlightIDs: []string{"contract"}, FirePrimitives: []string{"glow-contract"}},
		{ID: "m-2", Label: fmt.Sprintf("balance[%s] += %d", to, amt), DurationMs: 400, HighlightIDs: []string{"acc-" + to, "cb-balances"}},
		{ID: "m-3", Label: "totalSupply += amount", DurationMs: 400, HighlightIDs: []string{"cb-status", "balance-pie"}, FirePrimitives: []string{"burst-tx"}, IsLinkTrigger: true},
	}
}

func appendTransferMicroSteps(env *fw.RenderEnvelope, from, to string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "t-1", Label: "检查 balance[from] ≥ amount", DurationMs: 400, HighlightIDs: []string{"acc-" + from, "formula-erc20"}},
		{ID: "t-2", Label: "balance[from] -= amount", DurationMs: 400, HighlightIDs: []string{"acc-" + from}},
		{ID: "t-3", Label: "balance[to] += amount", DurationMs: 400, HighlightIDs: []string{"acc-" + to, "cb-balances"}, FirePrimitives: []string{"burst-tx"}, IsLinkTrigger: true},
	}
}

func appendApproveMicroSteps(env *fw.RenderEnvelope, owner, spender string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "a-1", Label: "owner 调用 approve", DurationMs: 400, HighlightIDs: []string{"acc-" + owner}},
		{ID: "a-2", Label: fmt.Sprintf("allowance[%s][%s] = amount", owner, spender), DurationMs: 500, HighlightIDs: []string{"cb-allowances"}},
		{ID: "a-3", Label: "spender 之后可用 transferFrom", DurationMs: 400, HighlightIDs: []string{"acc-" + spender}, IsLinkTrigger: true},
	}
}

func appendTransferFromMicroSteps(env *fw.RenderEnvelope, owner, to, spender string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tf-1", Label: "spender 调用 transferFrom", DurationMs: 400, HighlightIDs: []string{"acc-" + spender, "formula-erc20"}},
		{ID: "tf-2", Label: "检查 allowance[owner][spender] ≥ amount", DurationMs: 400, HighlightIDs: []string{"cb-allowances"}},
		{ID: "tf-3", Label: "扣 allowance + 扣 owner.balance + 加 to.balance", DurationMs: 500, HighlightIDs: []string{"acc-" + owner, "acc-" + to}, FirePrimitives: []string{"burst-tx"}, IsLinkTrigger: true},
	}
}

func appendBurnMicroSteps(env *fw.RenderEnvelope, holder string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "b-1", Label: holder + " 调用 burn", DurationMs: 400, HighlightIDs: []string{"acc-" + holder}},
		{ID: "b-2", Label: "balance -= amount; totalSupply -= amount", DurationMs: 500, HighlightIDs: []string{"cb-balances", "cb-status"}, FirePrimitives: []string{"burst-tx"}},
		{ID: "b-3", Label: "代币永久销毁", DurationMs: 400, HighlightIDs: []string{"contract"}, IsLinkTrigger: true},
	}
}

func appendFailMicroSteps(env *fw.RenderEnvelope, reason string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "f-1", Label: "调用失败：" + reason, DurationMs: 500, HighlightIDs: []string{"cb-events"}},
		{ID: "f-2", Label: "状态未变", DurationMs: 400, HighlightIDs: []string{"cb-balances"}, IsLinkTrigger: true},
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
		ID:             "token-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_token",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"tx.token.total_supply"},
		Payload:        map[string]any{"total_supply": st.TotalSupply},
		SourceAnchorID: "token-anchor",
		TargetAnchorID: "tx-proc-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "tx.token.total_supply")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"tx": map[string]any{
			"token": map[string]any{
				"symbol":       st.TokenSymbol,
				"total_supply": int(st.TotalSupply),
				"event_count":  len(st.Events),
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
