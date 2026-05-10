// 模块：sim-engine/scenarios/internal/smartcontract/contractstatemachine
// 文件职责：SC-01 智能合约状态机场景的完整实现。
//
// SSOT 依据：06.md §4.6.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现一个含完整状态机 + 权限控制 + 暂停机制 + 防重入的合约：
//   · 状态：Idle → Proposed → Voting → Executed / Rejected
//   · 修饰符：onlyOwner / whenNotPaused / nonReentrant
//   · 投票：支持 / 反对 票统计；阈值 ≥ 2/3 即通过
//   · 权限失败 / 状态错误 / 暂停 / 重入攻击 等异常分支可视化

package contractstatemachine

import (
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "contract-state-machine"
	schemaVersion = "v1.0.0"
	algorithmType = "contract-fsm"

	stateIdle     = "Idle"
	stateProposed = "Proposed"
	stateVoting   = "Voting"
	stateExecuted = "Executed"
	stateRejected = "Rejected"

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "contract.fsm"
)

var stateOrder = []string{stateIdle, stateProposed, stateVoting, stateExecuted, stateRejected}

type proposal struct {
	ID          int
	Description string
	Yes         []string
	No          []string
	Status      string // active / executed / rejected
	CreatedTick int
}

type event struct {
	Tick   int
	Action string
	Caller string
	OK     bool
	Reason string
}

type snapState struct {
	Owner              string
	Voters             []string
	Paused             bool
	State              string
	CurrentProposal    *proposal
	HistoryProps       []proposal
	NextID             int
	InCall             bool // 重入保护标志
	ReentrancyAttempts int
	Tick               int
	Events             []event
	LastError          string
}

func defaultSnapState() snapState {
	return snapState{
		Owner:  "alice",
		Voters: []string{"alice", "bob", "carol"},
		State:  stateIdle,
	}
}

func (st *snapState) pushEvent(act, caller string, ok bool, reason string) {
	st.Events = append(st.Events, event{Tick: st.Tick, Action: act, Caller: caller, OK: ok, Reason: reason})
	if len(st.Events) > 32 {
		st.Events = st.Events[len(st.Events)-32:]
	}
}

// onlyOwner 修饰符。
func (st *snapState) onlyOwner(caller string) error {
	if caller != st.Owner {
		return fmt.Errorf("onlyOwner: %s 不是 owner（owner=%s）", caller, st.Owner)
	}
	return nil
}

// whenNotPaused 修饰符。
func (st *snapState) whenNotPaused() error {
	if st.Paused {
		return errors.New("whenNotPaused: 合约已暂停")
	}
	return nil
}

// nonReentrant 修饰符（伪 Solidity ReentrancyGuard）。
func (st *snapState) nonReentrant() error {
	if st.InCall {
		st.ReentrancyAttempts++
		return errors.New("ReentrancyGuard: reentrant call detected")
	}
	st.InCall = true
	return nil
}

func (st *snapState) endNonReentrant() {
	st.InCall = false
}

// isVoter 检查 caller 是否为投票者。
func (st *snapState) isVoter(caller string) bool {
	for _, v := range st.Voters {
		if v == caller {
			return true
		}
	}
	return false
}

// transferOwnership 改变 owner。
func (st *snapState) transferOwnership(caller, newOwner string) error {
	if err := st.onlyOwner(caller); err != nil {
		st.pushEvent("transferOwnership", caller, false, err.Error())
		return err
	}
	old := st.Owner
	st.Owner = newOwner
	st.pushEvent("transferOwnership", caller, true, fmt.Sprintf("%s → %s", old, newOwner))
	return nil
}

// setPaused 切换暂停状态（仅 owner）。
func (st *snapState) setPaused(caller string, paused bool) error {
	if err := st.onlyOwner(caller); err != nil {
		st.pushEvent("setPaused", caller, false, err.Error())
		return err
	}
	st.Paused = paused
	tag := "unpaused"
	if paused {
		tag = "paused"
	}
	st.pushEvent("setPaused", caller, true, tag)
	return nil
}

// propose 创建新提案 (Idle → Proposed)。
func (st *snapState) propose(caller, desc string) error {
	if err := st.whenNotPaused(); err != nil {
		st.pushEvent("propose", caller, false, err.Error())
		return err
	}
	if !st.isVoter(caller) {
		err := fmt.Errorf("%s 不是投票者", caller)
		st.pushEvent("propose", caller, false, err.Error())
		return err
	}
	if st.State != stateIdle && st.State != stateExecuted && st.State != stateRejected {
		err := fmt.Errorf("当前状态 %s，不能 propose", st.State)
		st.pushEvent("propose", caller, false, err.Error())
		return err
	}
	st.NextID++
	st.CurrentProposal = &proposal{
		ID: st.NextID, Description: desc,
		Status: "active", CreatedTick: st.Tick,
	}
	st.State = stateProposed
	st.pushEvent("propose", caller, true, fmt.Sprintf("#%d %q", st.NextID, desc))
	return nil
}

// startVoting Proposed → Voting。
func (st *snapState) startVoting(caller string) error {
	if err := st.onlyOwner(caller); err != nil {
		st.pushEvent("startVoting", caller, false, err.Error())
		return err
	}
	if err := st.whenNotPaused(); err != nil {
		st.pushEvent("startVoting", caller, false, err.Error())
		return err
	}
	if st.State != stateProposed {
		err := fmt.Errorf("当前 %s，不能 startVoting", st.State)
		st.pushEvent("startVoting", caller, false, err.Error())
		return err
	}
	st.State = stateVoting
	st.pushEvent("startVoting", caller, true, "")
	return nil
}

// vote 在 Voting 阶段投票。
func (st *snapState) vote(caller string, support bool) error {
	if err := st.whenNotPaused(); err != nil {
		st.pushEvent("vote", caller, false, err.Error())
		return err
	}
	if !st.isVoter(caller) {
		err := fmt.Errorf("%s 不是投票者", caller)
		st.pushEvent("vote", caller, false, err.Error())
		return err
	}
	if st.State != stateVoting || st.CurrentProposal == nil {
		err := fmt.Errorf("当前 %s，不能 vote", st.State)
		st.pushEvent("vote", caller, false, err.Error())
		return err
	}
	// 防重复投票
	for _, v := range st.CurrentProposal.Yes {
		if v == caller {
			err := errors.New("已投赞成票")
			st.pushEvent("vote", caller, false, err.Error())
			return err
		}
	}
	for _, v := range st.CurrentProposal.No {
		if v == caller {
			err := errors.New("已投反对票")
			st.pushEvent("vote", caller, false, err.Error())
			return err
		}
	}
	if support {
		st.CurrentProposal.Yes = append(st.CurrentProposal.Yes, caller)
	} else {
		st.CurrentProposal.No = append(st.CurrentProposal.No, caller)
	}
	st.pushEvent("vote", caller, true, fmt.Sprintf("support=%v", support))
	return nil
}

// execute 关闭 Voting → Executed/Rejected。带 nonReentrant 保护。
func (st *snapState) execute(caller string) error {
	if err := st.onlyOwner(caller); err != nil {
		st.pushEvent("execute", caller, false, err.Error())
		return err
	}
	if err := st.whenNotPaused(); err != nil {
		st.pushEvent("execute", caller, false, err.Error())
		return err
	}
	if err := st.nonReentrant(); err != nil {
		st.pushEvent("execute", caller, false, err.Error())
		return err
	}
	defer st.endNonReentrant()
	if st.State != stateVoting || st.CurrentProposal == nil {
		err := fmt.Errorf("当前 %s，不能 execute", st.State)
		st.pushEvent("execute", caller, false, err.Error())
		return err
	}
	yes := len(st.CurrentProposal.Yes)
	total := len(st.Voters)
	if yes*3 >= 2*total {
		st.State = stateExecuted
		st.CurrentProposal.Status = "executed"
		st.pushEvent("execute", caller, true, fmt.Sprintf("通过 %d/%d", yes, total))
	} else {
		st.State = stateRejected
		st.CurrentProposal.Status = "rejected"
		st.pushEvent("execute", caller, true, fmt.Sprintf("否决 %d/%d", yes, total))
	}
	st.HistoryProps = append(st.HistoryProps, *st.CurrentProposal)
	if len(st.HistoryProps) > 16 {
		st.HistoryProps = st.HistoryProps[len(st.HistoryProps)-16:]
	}
	return nil
}

// reentrantAttack 模拟在 execute 中再次 execute（应被 nonReentrant 拦截）。
func (st *snapState) reentrantAttack(caller string) error {
	if err := st.nonReentrant(); err != nil {
		st.pushEvent("execute(re-entry)", caller, false, err.Error())
		return err
	}
	defer st.endNonReentrant()
	// 嵌套调用
	if err := st.nonReentrant(); err != nil {
		st.pushEvent("execute(re-entry)", caller, false, err.Error())
		return err
	}
	st.endNonReentrant()
	st.pushEvent("execute(re-entry)", caller, true, "（不应该到这）")
	return nil
}

// reset 重置场景。
func (st *snapState) reset() {
	*st = defaultSnapState()
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
		Owner:              fw.MapStr(d, "owner", "alice"),
		Paused:             fw.MapBool(d, "paused", false),
		State:              fw.MapStr(d, "state", stateIdle),
		NextID:             fw.MapInt(d, "next_id", 0),
		InCall:             fw.MapBool(d, "in_call", false),
		ReentrancyAttempts: fw.MapInt(d, "re_attempts", 0),
		Tick:               fw.MapInt(d, "tick", 0),
		LastError:          fw.MapStr(d, "last_error", ""),
	}
	if vsAny, ok := d["voters"].([]any); ok {
		for _, v := range vsAny {
			if s, ok := v.(string); ok {
				st.Voters = append(st.Voters, s)
			}
		}
	}
	if len(st.Voters) == 0 {
		st.Voters = []string{"alice", "bob", "carol"}
	}
	if cpAny, ok := d["current_prop"].(map[string]any); ok {
		st.CurrentProposal = decodeProp(cpAny)
	}
	if hpAny, ok := d["history_props"].([]any); ok {
		for _, p := range hpAny {
			if pm, ok := p.(map[string]any); ok {
				st.HistoryProps = append(st.HistoryProps, *decodeProp(pm))
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, evAny := range eAny {
			if em, ok := evAny.(map[string]any); ok {
				st.Events = append(st.Events, event{
					Tick: fw.MapInt(em, "tick", 0), Action: fw.MapStr(em, "action", ""),
					Caller: fw.MapStr(em, "caller", ""), OK: fw.MapBool(em, "ok", false),
					Reason: fw.MapStr(em, "reason", ""),
				})
			}
		}
	}
	return st
}

func decodeProp(m map[string]any) *proposal {
	p := &proposal{
		ID:          fw.MapInt(m, "id", 0),
		Description: fw.MapStr(m, "desc", ""),
		Status:      fw.MapStr(m, "status", ""),
		CreatedTick: fw.MapInt(m, "created", 0),
	}
	if yAny, ok := m["yes"].([]any); ok {
		for _, v := range yAny {
			if s, ok := v.(string); ok {
				p.Yes = append(p.Yes, s)
			}
		}
	}
	if nAny, ok := m["no"].([]any); ok {
		for _, v := range nAny {
			if s, ok := v.(string); ok {
				p.No = append(p.No, s)
			}
		}
	}
	return p
}

func encodeProp(p *proposal) map[string]any {
	yes := make([]any, len(p.Yes))
	for i, v := range p.Yes {
		yes[i] = v
	}
	no := make([]any, len(p.No))
	for i, v := range p.No {
		no[i] = v
	}
	return map[string]any{
		"id": p.ID, "desc": p.Description, "status": p.Status,
		"created": p.CreatedTick, "yes": yes, "no": no,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["owner"] = st.Owner
	s.Data["paused"] = st.Paused
	s.Data["state"] = st.State
	s.Data["next_id"] = st.NextID
	s.Data["in_call"] = st.InCall
	s.Data["re_attempts"] = st.ReentrancyAttempts
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	vs := make([]any, len(st.Voters))
	for i, v := range st.Voters {
		vs[i] = v
	}
	s.Data["voters"] = vs
	if st.CurrentProposal != nil {
		s.Data["current_prop"] = encodeProp(st.CurrentProposal)
	} else {
		delete(s.Data, "current_prop")
	}
	hp := make([]any, len(st.HistoryProps))
	for i, p := range st.HistoryProps {
		hp[i] = encodeProp(&p)
	}
	s.Data["history_props"] = hp
	eAny := make([]any, len(st.Events))
	for i, e := range st.Events {
		eAny[i] = map[string]any{
			"tick": e.Tick, "action": e.Action, "caller": e.Caller,
			"ok": e.OK, "reason": e.Reason,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "合约状态机",
		Description:         "演示状态机：Idle/Proposed/Voting/Executed/Rejected + onlyOwner / whenNotPaused / nonReentrant",
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
			"contract.fsm.state",
			"contract.fsm.reentrancy_attempts",
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
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: "Idle", Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "transfer_ownership", Label: "转移 Owner",
				Description: "onlyOwner",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
					{Name: "new_owner", Type: fw.FieldString, Label: "new_owner", Required: true, Default: "bob"},
				},
			},
			{
				ActionCode: "set_paused", Label: "暂停 / 恢复",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
					{Name: "paused", Type: fw.FieldBoolean, Label: "暂停？", Required: true, Default: true},
				},
			},
			{
				ActionCode: "propose", Label: "创建提案",
				Description: "Idle/Executed/Rejected → Proposed",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: true, Default: "升级合约逻辑 V2"},
				},
				WritesOwnedFields: []string{"contract.fsm.state"},
				LinkOwnerFields:   []string{"contract.fsm.state"},
			},
			{
				ActionCode: "start_voting", Label: "开启投票",
				Description: "Proposed → Voting (onlyOwner)",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
				},
			},
			{
				ActionCode: "vote", Label: "投票",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "bob"},
					{Name: "support", Type: fw.FieldBoolean, Label: "support？", Required: true, Default: true},
				},
			},
			{
				ActionCode: "execute", Label: "执行结果",
				Description: "Voting → Executed/Rejected (onlyOwner + nonReentrant)",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "alice"},
				},
				WritesOwnedFields: []string{"contract.fsm.state"},
				LinkOwnerFields:   []string{"contract.fsm.state"},
			},
			{
				ActionCode: "reentrant_attack", Label: "重入攻击",
				Description: "嵌套 nonReentrant 调用，应被拦截",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "caller", Type: fw.FieldString, Label: "caller", Required: true, Default: "attacker"},
				},
				WritesOwnedFields: []string{"contract.fsm.reentrancy_attempts"},
				LinkOwnerFields:   []string{"contract.fsm.reentrancy_attempts"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
				ActionCode:    "read_storage",
				Label:         "读取存储槽（真实链）",
				Description:   "调 geth eth_getStorageAt 读取合约状态",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["{{contract}}","{{slot}}","latest"],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "contract", Type: fw.FieldString, Label: "contract address", Required: true, Default: "0x"},
					{Name: "slot", Type: fw.FieldString, Label: "storage slot (hex)", Required: true, Default: "0x0"},
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
	state.Phase = st.State
	env := buildEnvelope(st, "init", "FSM 初始化（Idle, owner=alice）", true)
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
	case "transfer_ownership":
		caller := fw.MapStr(in.Params, "caller", "alice")
		newOwner := fw.MapStr(in.Params, "new_owner", "bob")
		if err := st.transferOwnership(caller, newOwner); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "transfer_ownership", err.Error(), false)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "transfer_ownership",
			fmt.Sprintf("owner: %s → %s", caller, newOwner), false)
		appendOwnerMicroSteps(&out.Render)
		return out, nil

	case "set_paused":
		caller := fw.MapStr(in.Params, "caller", "alice")
		paused := fw.MapBool(in.Params, "paused", true)
		if err := st.setPaused(caller, paused); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "set_paused", err.Error(), false)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		tag := "已恢复"
		if paused {
			tag = "已暂停"
		}
		out.Render = buildEnvelope(st, "set_paused", tag, false)
		return out, nil

	case "propose":
		caller := fw.MapStr(in.Params, "caller", "alice")
		desc := fw.MapStr(in.Params, "description", "升级合约逻辑 V2")
		if err := st.propose(caller, desc); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "propose", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "propose",
			fmt.Sprintf("提案 #%d: %s → Proposed", st.NextID, desc), false)
		appendProposeMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "start_voting":
		caller := fw.MapStr(in.Params, "caller", "alice")
		if err := st.startVoting(caller); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "start_voting", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "start_voting", "Proposed → Voting", false)
		appendStartVotingMicroSteps(&out.Render)
		return out, nil

	case "vote":
		caller := fw.MapStr(in.Params, "caller", "bob")
		support := fw.MapBool(in.Params, "support", true)
		if err := st.vote(caller, support); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "vote", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "vote",
			fmt.Sprintf("%s 投 %v", caller, support), false)
		appendVoteMicroSteps(&out.Render, caller, support)
		return out, nil

	case "execute":
		caller := fw.MapStr(in.Params, "caller", "alice")
		if err := st.execute(caller); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "execute", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		summary := fmt.Sprintf("结算：%s", st.State)
		out.Render = buildEnvelope(st, "execute", summary, false)
		appendExecuteMicroSteps(&out.Render, st.State)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "reentrant_attack":
		caller := fw.MapStr(in.Params, "caller", "attacker")
		err := st.reentrantAttack(caller)
		saveState(state, st)
		if err != nil {
			out.Render = buildEnvelope(st, "reentrant_attack",
				fmt.Sprintf("✓ 重入被拦截：%s", err.Error()), false)
			appendReentrantMicroSteps(&out.Render, true)
			return fw.ActionOutput{Success: true, Render: out.Render}, nil
		}
		out.Render = buildEnvelope(st, "reentrant_attack", "⚠ 重入未被拦截", false)
		appendReentrantMicroSteps(&out.Render, false)
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
		st.reset()
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

	// 1) 状态节点（5 个）
	stateIDs := make([]string, len(stateOrder))
	for i, s := range stateOrder {
		stateIDs[i] = "st-" + s
	}
	prims = append(prims, fw.PrimStack("state-stack", stateIDs, "horizontal"))
	for i, s := range stateOrder {
		role := "state"
		status := "normal"
		if s == st.State {
			status = "active"
			role = "current-state"
		}
		if s == stateExecuted && st.State == stateExecuted {
			role = "success-state"
		}
		if s == stateRejected && st.State == stateRejected {
			role = "fail-state"
		}
		prims = append(prims, fw.PrimNode(stateIDs[i], s, status, role))
	}
	// 2) 状态转移边
	transitions := []struct{ from, to string }{
		{stateIdle, stateProposed},
		{stateProposed, stateVoting},
		{stateVoting, stateExecuted},
		{stateVoting, stateRejected},
		{stateExecuted, stateProposed}, // 新提案
		{stateRejected, stateProposed},
	}
	for i, t := range transitions {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("trans-%d", i), "st-"+t.from, "st-"+t.to, "solid", ""))
	}

	// 3) phase_progress
	idx := 0
	for i, s := range stateOrder {
		if s == st.State {
			idx = i
		}
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", stateOrder, idx, float64(idx)/float64(len(stateOrder)-1)))

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-modifier",
		`\text{onlyOwner: } caller = owner;\quad \text{whenNotPaused: } \neg paused;\quad \text{nonReentrant: } \neg inCall`, false))

	// 5) 状态参数
	pausedTag := "no"
	if st.Paused {
		pausedTag = "YES"
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("owner = %s\nvoters = %s\npaused = %s\nstate = %s\nnext_id = %d\nin_call = %v\nreentrancy_attempts = %d",
			st.Owner, strings.Join(st.Voters, ", "),
			pausedTag, st.State, st.NextID, st.InCall, st.ReentrancyAttempts),
		"text", nil, 8))

	// 6) 当前提案详情
	if st.CurrentProposal != nil {
		p := st.CurrentProposal
		yes, no := len(p.Yes), len(p.No)
		quorum := (len(st.Voters)*2 + 2) / 3
		propLines := []string{
			fmt.Sprintf("提案 #%d: %s", p.ID, p.Description),
			fmt.Sprintf("status = %s", p.Status),
			fmt.Sprintf("yes = %d ([%s])", yes, strings.Join(p.Yes, ", ")),
			fmt.Sprintf("no  = %d ([%s])", no, strings.Join(p.No, ", ")),
			fmt.Sprintf("门槛 ≥ 2/3 (= %d)", quorum),
		}
		prims = append(prims, fw.PrimCodeBlock("cb-prop", strings.Join(propLines, "\n"), "text", nil, 8))

		// 投票饼图
		if yes+no > 0 {
			segs := []map[string]any{
				{"label": "Yes", "value": float64(yes), "color_role": "success"},
				{"label": "No", "value": float64(no), "color_role": "danger"},
			}
			prims = append(prims, fw.PrimPieChart("vote-pie", segs))
		}
	}

	// 7) 历史提案表
	if len(st.HistoryProps) > 0 {
		histLines := []string{"id  status      yes/no  description"}
		startIdx := 0
		if len(st.HistoryProps) > 8 {
			startIdx = len(st.HistoryProps) - 8
		}
		for _, p := range st.HistoryProps[startIdx:] {
			histLines = append(histLines, fmt.Sprintf("%-3d %-10s  %d/%d   %s",
				p.ID, p.Status, len(p.Yes), len(p.No), p.Description))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(histLines, "\n"), "text", nil, 12))
	}

	// 8) 事件日志
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
			line := fmt.Sprintf("  t=%d  %s [%s] caller=%s", e.Tick, ok, e.Action, e.Caller)
			if e.Reason != "" {
				line += " // " + e.Reason
			}
			eLines = append(eLines, line)
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 16))
	}

	// 9) 动效
	prims = append(prims, fw.PrimGlow("glow-state", "st-"+st.State, "info", 0.9))
	if st.Paused {
		prims = append(prims, fw.PrimShake("shake-paused", "cb-status", 0.3, 600))
		prims = append(prims, fw.PrimPulse("pulse-paused", "cb-status", "warning", 1500))
	}
	if st.ReentrancyAttempts > 0 {
		prims = append(prims, fw.PrimShake("shake-re", "cb-status", 0.4, 700))
	}
	if st.State == stateExecuted {
		prims = append(prims, fw.PrimBurst("burst-exec", "st-"+stateExecuted, "success", int64(st.NextID), 700))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-sec", linkGroupContractSec, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "FSM 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "fsm.current_state", Value: st.State, TargetPrimitive: "cb-state", TargetParam: "state"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	yes, no := 0, 0
	if st.CurrentProposal != nil {
		yes = len(st.CurrentProposal.Yes)
		no = len(st.CurrentProposal.No)
	}
	d := map[string]any{
		"owner":               st.Owner,
		"paused":              st.Paused,
		"state":               st.State,
		"voters_count":        len(st.Voters),
		"current_yes":         yes,
		"current_no":          no,
		"reentrancy_attempts": st.ReentrancyAttempts,
		"history_count":       len(st.HistoryProps),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendOwnerMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "o-1", Label: "onlyOwner 检查", DurationMs: 400, HighlightIDs: []string{"formula-modifier", "cb-status"}},
		{ID: "o-2", Label: "更新 owner 字段", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendProposeMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "p-1", Label: "whenNotPaused 检查", DurationMs: 400, HighlightIDs: []string{"formula-modifier"}},
		{ID: "p-2", Label: "状态机：Idle → Proposed", DurationMs: 500, HighlightIDs: []string{"st-Proposed", "phase-progress"}, FirePrimitives: []string{"glow-state"}},
		{ID: "p-3", Label: "记录新提案", DurationMs: 400, HighlightIDs: []string{"cb-prop"}, IsLinkTrigger: true},
	}
}

func appendStartVotingMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sv-1", Label: "onlyOwner + whenNotPaused", DurationMs: 400, HighlightIDs: []string{"formula-modifier"}},
		{ID: "sv-2", Label: "Proposed → Voting", DurationMs: 500, HighlightIDs: []string{"st-Voting"}, FirePrimitives: []string{"glow-state"}, IsLinkTrigger: true},
	}
}

func appendVoteMicroSteps(env *fw.RenderEnvelope, caller string, support bool) {
	supTag := "yes"
	if !support {
		supTag = "no"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "v-1", Label: "isVoter 检查", DurationMs: 400, HighlightIDs: []string{"cb-status"}},
		{ID: "v-2", Label: caller + " 投 " + supTag + " 票", DurationMs: 500, HighlightIDs: []string{"cb-prop", "vote-pie"}},
		{ID: "v-3", Label: "更新 yes/no 列表", DurationMs: 400, HighlightIDs: []string{"vote-pie"}, IsLinkTrigger: true},
	}
}

func appendExecuteMicroSteps(env *fw.RenderEnvelope, finalState string) {
	tail := "Voting → Rejected"
	if finalState == stateExecuted {
		tail = "Voting → Executed (≥ 2/3)"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "e-1", Label: "onlyOwner + whenNotPaused + nonReentrant", DurationMs: 400, HighlightIDs: []string{"formula-modifier"}},
		{ID: "e-2", Label: "统计 yes/no 票", DurationMs: 500, HighlightIDs: []string{"vote-pie"}},
		{ID: "e-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"st-" + finalState}, FirePrimitives: []string{"burst-exec"}, IsLinkTrigger: true},
	}
}

func appendReentrantMicroSteps(env *fw.RenderEnvelope, blocked bool) {
	tail := "重入未被拦截"
	if blocked {
		tail = "✓ ReentrancyGuard 拦截了重入"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "re-1", Label: "进入 nonReentrant 调用 (in_call=true)", DurationMs: 400, HighlightIDs: []string{"cb-status"}, FirePrimitives: []string{"shake-re"}},
		{ID: "re-2", Label: "嵌套调用 nonReentrant", DurationMs: 400, HighlightIDs: []string{"formula-modifier"}},
		{ID: "re-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-events"}, IsLinkTrigger: true},
	}
}

func appendFailMicroSteps(env *fw.RenderEnvelope, reason string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "f-1", Label: "调用失败：" + reason, DurationMs: 500, HighlightIDs: []string{"cb-events"}},
		{ID: "f-2", Label: "状态未变", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
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
		ID:             "fsm-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_fsm",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"contract.fsm.state"},
		Payload:        map[string]any{"state": st.State},
		SourceAnchorID: "fsm-anchor",
		TargetAnchorID: "contract-sec-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "contract.fsm.state")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"contract": map[string]any{
			"fsm": map[string]any{
				"state":               st.State,
				"owner":               st.Owner,
				"paused":              st.Paused,
				"voters_count":        len(st.Voters),
				"reentrancy_attempts": st.ReentrancyAttempts,
				"history_count":       len(st.HistoryProps),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

