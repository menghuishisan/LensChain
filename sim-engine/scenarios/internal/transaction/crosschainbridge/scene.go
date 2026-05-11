// 模块：sim-engine/scenarios/internal/transaction/crosschainbridge
// 文件职责：TX-05 跨链桥（Lock-Mint / Burn-Release）场景的完整实现。
//
// SSOT 依据：06.md §4.5.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现跨链桥（Bridge）协议（零外部依赖）：
//   · 两条链：source / target
//   · 锁仓铸造：user 在 source 上 lock(token, amount) → 桥事件
//                → bridge validators 收集 ≥ 2f+1 签名 → target 上 mint(amount)
//   · 销毁释放：user 在 target 上 burn(amount) → bridge 收集 ≥ 2f+1 签名 → source.release(amount)
//   · 防重放：每个 message 有唯一 nonce + processed 集合；已 processed 的 message 不能再次中继
//   · 攻击：
//     · replay：把同一笔 lock 事件提交到 target 两次（应被 nonce 拦截）
//     · double-mint：≤ 2f 签名也尝试 mint（应被 quorum 拦截）

package crosschainbridge

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "cross-chain-bridge"
	schemaVersion = "v1.0.0"
	algorithmType = "bridge-lock-mint"

	defaultValidators = 5
	defaultBalance    = 1000
	maxValidators     = 9

	directionLockMint    = "lock-mint"
	directionBurnRelease = "burn-release"

	linkGroupTxProc  = "tx-processing-group"
	linkOwnerSubtree = "tx.bridge"
)

// validator 桥的验证者节点。
type validator struct {
	ID          string
	IsByzantine bool
	IsDown      bool
}

// message 跨链桥消息。
type message struct {
	Nonce         int
	Direction     string // lock-mint / burn-release
	User          string
	Amount        int64
	SourceChainTx string
	Signatures    []string // 已签名的 validator ID
	Status        string   // pending / executed / replay-rejected / quorum-failed
	ExecutedAt    int      // tick
	Tag           string   // user-flow / replay-attack / quorum-attack
}

type snapState struct {
	Validators        []validator
	Threshold         int // f：拜占庭容忍数；quorum = 2f+1
	SourceLocked      int64
	TargetMinted      int64 // 桥代币（wrapped token）
	UserSourceBalance map[string]int64
	UserTargetBalance map[string]int64
	NextNonce         int
	Messages          []message
	Processed         map[int]bool // 已 processed 的 nonce
	Tick              int
	BlockedReplays    int
	BlockedQuorum     int
	LastError         string
}

func defaultSnapState() snapState {
	st := snapState{
		Threshold:         1, // f=1, quorum=3
		UserSourceBalance: map[string]int64{"alice": defaultBalance, "bob": defaultBalance / 2},
		UserTargetBalance: map[string]int64{"alice": 0, "bob": 0},
		Processed:         map[int]bool{},
	}
	for i := 0; i < defaultValidators; i++ {
		st.Validators = append(st.Validators, validator{ID: fmt.Sprintf("v%d", i)})
	}
	return st
}

func (st snapState) quorum() int { return 2*st.Threshold + 1 }

func (st snapState) activeValidators() []*validator {
	out := []*validator{}
	for i := range st.Validators {
		v := &st.Validators[i]
		if !v.IsDown {
			out = append(out, v)
		}
	}
	return out
}

// lock 用户在 source 锁定 amount，创建 lock-mint 消息。
func (st *snapState) lock(user string, amount int64) (*message, error) {
	if amount <= 0 {
		return nil, errors.New("amount 必须 > 0")
	}
	if st.UserSourceBalance[user] < amount {
		return nil, fmt.Errorf("%s 在 source 余额不足", user)
	}
	st.UserSourceBalance[user] -= amount
	st.SourceLocked += amount
	m := message{
		Nonce: st.NextNonce, Direction: directionLockMint,
		User: user, Amount: amount,
		SourceChainTx: fmt.Sprintf("src-tx%d", st.NextNonce),
		Status:        "pending",
		Tag:           "user-flow",
	}
	st.NextNonce++
	st.Messages = append(st.Messages, m)
	return &st.Messages[len(st.Messages)-1], nil
}

// burn 用户在 target 销毁 amount，创建 burn-release 消息。
func (st *snapState) burn(user string, amount int64) (*message, error) {
	if amount <= 0 {
		return nil, errors.New("amount 必须 > 0")
	}
	if st.UserTargetBalance[user] < amount {
		return nil, fmt.Errorf("%s 在 target 余额不足", user)
	}
	st.UserTargetBalance[user] -= amount
	st.TargetMinted -= amount
	m := message{
		Nonce: st.NextNonce, Direction: directionBurnRelease,
		User: user, Amount: amount,
		SourceChainTx: fmt.Sprintf("tgt-tx%d", st.NextNonce),
		Status:        "pending",
		Tag:           "user-flow",
	}
	st.NextNonce++
	st.Messages = append(st.Messages, m)
	return &st.Messages[len(st.Messages)-1], nil
}

// signMessage validator 给指定 nonce 的消息签名。
func (st *snapState) signMessage(nonce int, validatorID string) error {
	m := st.findMessage(nonce)
	if m == nil {
		return fmt.Errorf("未找到 nonce=%d 的消息", nonce)
	}
	if m.Status != "pending" {
		return fmt.Errorf("nonce=%d 状态 %s 不允许签名", nonce, m.Status)
	}
	var v *validator
	for i := range st.Validators {
		if st.Validators[i].ID == validatorID {
			v = &st.Validators[i]
			break
		}
	}
	if v == nil {
		return fmt.Errorf("未找到 validator: %s", validatorID)
	}
	if v.IsDown {
		return fmt.Errorf("%s 已下线", validatorID)
	}
	if v.IsByzantine {
		return fmt.Errorf("%s 是拜占庭节点（拒绝签名）", validatorID)
	}
	for _, s := range m.Signatures {
		if s == validatorID {
			return nil // 已签
		}
	}
	m.Signatures = append(m.Signatures, validatorID)
	return nil
}

// signByAll 让所有非拜占庭非 down validator 签名（教学便捷）。
func (st *snapState) signByAll(nonce int) int {
	m := st.findMessage(nonce)
	if m == nil {
		return 0
	}
	cnt := 0
	for i := range st.Validators {
		v := &st.Validators[i]
		if v.IsDown || v.IsByzantine {
			continue
		}
		seen := false
		for _, s := range m.Signatures {
			if s == v.ID {
				seen = true
				break
			}
		}
		if !seen {
			m.Signatures = append(m.Signatures, v.ID)
			cnt++
		}
	}
	return cnt
}

// executeMessage 执行已收齐 quorum 的消息（mint 或 release）。
func (st *snapState) executeMessage(nonce int) error {
	m := st.findMessage(nonce)
	if m == nil {
		return fmt.Errorf("未找到 nonce=%d 的消息", nonce)
	}
	if m.Status != "pending" {
		return fmt.Errorf("nonce=%d 状态 %s 不允许执行", nonce, m.Status)
	}
	if st.Processed[nonce] {
		m.Status = "replay-rejected"
		st.BlockedReplays++
		return fmt.Errorf("nonce=%d 已 processed（重放被拒）", nonce)
	}
	if len(m.Signatures) < st.quorum() {
		m.Status = "quorum-failed"
		st.BlockedQuorum++
		return fmt.Errorf("签名 %d < quorum %d", len(m.Signatures), st.quorum())
	}
	switch m.Direction {
	case directionLockMint:
		st.UserTargetBalance[m.User] += m.Amount
		st.TargetMinted += m.Amount
	case directionBurnRelease:
		st.UserSourceBalance[m.User] += m.Amount
		st.SourceLocked -= m.Amount
	}
	m.Status = "executed"
	m.ExecutedAt = st.Tick
	st.Processed[nonce] = true
	return nil
}

func (st *snapState) findMessage(nonce int) *message {
	for i := range st.Messages {
		if st.Messages[i].Nonce == nonce {
			return &st.Messages[i]
		}
	}
	return nil
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
		Threshold:         fw.MapInt(d, "threshold", 1),
		SourceLocked:      int64(fw.MapInt(d, "source_locked", 0)),
		TargetMinted:      int64(fw.MapInt(d, "target_minted", 0)),
		NextNonce:         fw.MapInt(d, "next_nonce", 0),
		Tick:              fw.MapInt(d, "tick", 0),
		BlockedReplays:    fw.MapInt(d, "blocked_replays", 0),
		BlockedQuorum:     fw.MapInt(d, "blocked_quorum", 0),
		LastError:         fw.MapStr(d, "last_error", ""),
		UserSourceBalance: map[string]int64{},
		UserTargetBalance: map[string]int64{},
		Processed:         map[int]bool{},
	}
	if vsAny, ok := d["validators"].([]any); ok {
		for _, vAny := range vsAny {
			if vm, ok := vAny.(map[string]any); ok {
				st.Validators = append(st.Validators, validator{
					ID:          fw.MapStr(vm, "id", ""),
					IsByzantine: fw.MapBool(vm, "byz", false),
					IsDown:      fw.MapBool(vm, "down", false),
				})
			}
		}
	}
	if len(st.Validators) == 0 {
		return defaultSnapState()
	}
	if mAny, ok := d["messages"].([]any); ok {
		for _, msgAny := range mAny {
			if mm, ok := msgAny.(map[string]any); ok {
				m := message{
					Nonce:         fw.MapInt(mm, "nonce", 0),
					Direction:     fw.MapStr(mm, "dir", ""),
					User:          fw.MapStr(mm, "user", ""),
					Amount:        int64(fw.MapInt(mm, "amount", 0)),
					SourceChainTx: fw.MapStr(mm, "src_tx", ""),
					Status:        fw.MapStr(mm, "status", "pending"),
					ExecutedAt:    fw.MapInt(mm, "exec_at", 0),
					Tag:           fw.MapStr(mm, "tag", ""),
				}
				if sAny, ok := mm["sigs"].([]any); ok {
					for _, x := range sAny {
						if s, ok := x.(string); ok {
							m.Signatures = append(m.Signatures, s)
						}
					}
				}
				st.Messages = append(st.Messages, m)
			}
		}
	}
	if usAny, ok := d["us_balance"].(map[string]any); ok {
		for k, v := range usAny {
			st.UserSourceBalance[k] = int64(intFromAny(v))
		}
	}
	if utAny, ok := d["ut_balance"].(map[string]any); ok {
		for k, v := range utAny {
			st.UserTargetBalance[k] = int64(intFromAny(v))
		}
	}
	if pAny, ok := d["processed"].([]any); ok {
		for _, p := range pAny {
			st.Processed[intFromAny(p)] = true
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["threshold"] = st.Threshold
	s.Data["source_locked"] = int(st.SourceLocked)
	s.Data["target_minted"] = int(st.TargetMinted)
	s.Data["next_nonce"] = st.NextNonce
	s.Data["tick"] = st.Tick
	s.Data["blocked_replays"] = st.BlockedReplays
	s.Data["blocked_quorum"] = st.BlockedQuorum
	s.Data["last_error"] = st.LastError
	vsAny := make([]any, len(st.Validators))
	for i, v := range st.Validators {
		vsAny[i] = map[string]any{"id": v.ID, "byz": v.IsByzantine, "down": v.IsDown}
	}
	s.Data["validators"] = vsAny
	mAny := make([]any, len(st.Messages))
	for i, m := range st.Messages {
		sigs := make([]any, len(m.Signatures))
		for j, x := range m.Signatures {
			sigs[j] = x
		}
		mAny[i] = map[string]any{
			"nonce": m.Nonce, "dir": m.Direction, "user": m.User,
			"amount": int(m.Amount), "src_tx": m.SourceChainTx,
			"status": m.Status, "exec_at": m.ExecutedAt, "tag": m.Tag,
			"sigs": sigs,
		}
	}
	s.Data["messages"] = mAny
	us := map[string]any{}
	for k, v := range st.UserSourceBalance {
		us[k] = int(v)
	}
	s.Data["us_balance"] = us
	ut := map[string]any{}
	for k, v := range st.UserTargetBalance {
		ut[k] = int(v)
	}
	s.Data["ut_balance"] = ut
	pAny := []any{}
	for k := range st.Processed {
		pAny = append(pAny, k)
	}
	s.Data["processed"] = pAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "跨链桥（Lock-Mint / Burn-Release）",
		Description:         "演示 lock-mint 和 burn-release + validator 多签 + 防重放 / 防双花攻击",
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
			"tx.bridge.source_locked",
			"tx.bridge.target_minted",
			"tx.bridge.blocked_replays",
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
				ActionCode: "set_validators", Label: "设置 validators",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "validator 数 (n=3f+1)", Required: true, Default: defaultValidators, Min: 4, Max: maxValidators, Step: 3},
				},
			},
			{
				ActionCode: "lock", Label: "Source 锁仓",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "user", Type: fw.FieldString, Label: "user", Required: true, Default: "alice"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 100, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.bridge.source_locked"},
				LinkOwnerFields:   []string{"tx.bridge.source_locked"},
			},
			{
				ActionCode: "burn", Label: "Target 销毁",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "user", Type: fw.FieldString, Label: "user", Required: true, Default: "alice"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 50, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.bridge.source_locked"},
				LinkOwnerFields:   []string{"tx.bridge.source_locked"},
			},
			{
				ActionCode: "sign_all", Label: "所有 validator 签名",
				Description:   "让所有非拜占庭、非 down 的 validator 给指定 nonce 签名",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce", Required: true, Default: 0, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "execute", Label: "执行（mint/release）",
				Description:   "若签名 ≥ quorum 则执行；若 nonce 已 processed 则拒绝",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce", Required: true, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"tx.bridge.target_minted"},
				LinkOwnerFields:   []string{"tx.bridge.target_minted"},
			},
			{
				ActionCode: "replay_attack", Label: "重放攻击",
				Description:   "尝试再次执行已 processed 的消息（应被 nonce 拦截）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce", Required: true, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"tx.bridge.blocked_replays"},
				LinkOwnerFields:   []string{"tx.bridge.blocked_replays"},
			},
			{
				ActionCode: "quorum_attack", Label: "Quorum 不足执行",
				Description:   "签名 < 2f+1 时强制执行（应被 quorum 拦截）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce", Required: true, Default: 0, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "byzantine_validator", Label: "标记 validator 拜占庭",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "validator_id", Type: fw.FieldString, Label: "ID", Required: true, Default: "v0"},
				},
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
	env := buildEnvelope(st, "init", "Bridge 初始化（5 validators, f=1）", true)
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
	case "set_validators":
		n := fw.MapInt(in.Params, "n", defaultValidators)
		if n < 4 {
			n = 4
		}
		if n > maxValidators {
			n = maxValidators
		}
		st = defaultSnapState()
		st.Validators = nil
		for i := 0; i < n; i++ {
			st.Validators = append(st.Validators, validator{ID: fmt.Sprintf("v%d", i)})
		}
		st.Threshold = (n - 1) / 3
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_validators",
			fmt.Sprintf("validators=%d, f=%d, quorum=%d", n, st.Threshold, st.quorum()), true)
		return out, nil

	case "lock":
		user := fw.MapStr(in.Params, "user", "alice")
		amt := int64(fw.MapInt(in.Params, "amount", 100))
		m, err := st.lock(user, amt)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "lock",
			fmt.Sprintf("source.lock(%s, %d) → 创建 nonce=%d (lock-mint)", user, amt, m.Nonce), false)
		appendLockMicroSteps(&out.Render, m.Nonce)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "burn":
		user := fw.MapStr(in.Params, "user", "alice")
		amt := int64(fw.MapInt(in.Params, "amount", 50))
		m, err := st.burn(user, amt)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "burn",
			fmt.Sprintf("target.burn(%s, %d) → 创建 nonce=%d (burn-release)", user, amt, m.Nonce), false)
		appendBurnMicroSteps(&out.Render, m.Nonce)
		return out, nil

	case "sign_all":
		nonce := fw.MapInt(in.Params, "nonce", 0)
		cnt := st.signByAll(nonce)
		saveState(state, st)
		out.Render = buildEnvelope(st, "sign_all",
			fmt.Sprintf("收集 %d 个新签名（nonce=%d）", cnt, nonce), false)
		appendSignMicroSteps(&out.Render, nonce)
		return out, nil

	case "execute":
		nonce := fw.MapInt(in.Params, "nonce", 0)
		err := st.executeMessage(nonce)
		saveState(state, st)
		if err != nil {
			out.Render = buildEnvelope(st, "execute", err.Error(), false)
			appendFailMicroSteps(&out.Render, err.Error())
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		m := st.findMessage(nonce)
		summary := fmt.Sprintf("✓ 执行 nonce=%d (%s, %s, %d)", nonce, m.Direction, m.User, m.Amount)
		out.Render = buildEnvelope(st, "execute", summary, false)
		appendExecuteMicroSteps(&out.Render, m.Direction)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "replay_attack":
		nonce := fw.MapInt(in.Params, "nonce", 0)
		err := st.executeMessage(nonce)
		saveState(state, st)
		if err != nil {
			out.Render = buildEnvelope(st, "replay_attack",
				fmt.Sprintf("✓ 重放被拒：%s", err.Error()), false)
			appendReplayMicroSteps(&out.Render, nonce, true)
			return fw.ActionOutput{Success: true, Render: out.Render}, nil
		}
		out.Render = buildEnvelope(st, "replay_attack", "⚠ 重放成功（不应该）", false)
		appendReplayMicroSteps(&out.Render, nonce, false)
		return out, nil

	case "quorum_attack":
		nonce := fw.MapInt(in.Params, "nonce", 0)
		err := st.executeMessage(nonce)
		saveState(state, st)
		if err != nil {
			out.Render = buildEnvelope(st, "quorum_attack",
				fmt.Sprintf("✓ Quorum 不足拒绝：%s", err.Error()), false)
			appendQuorumMicroSteps(&out.Render, nonce, true)
			return fw.ActionOutput{Success: true, Render: out.Render}, nil
		}
		out.Render = buildEnvelope(st, "quorum_attack", "⚠ 签名足够，已执行", false)
		appendQuorumMicroSteps(&out.Render, nonce, false)
		return out, nil

	case "byzantine_validator":
		vid := fw.MapStr(in.Params, "validator_id", "v0")
		for i := range st.Validators {
			if st.Validators[i].ID == vid {
				st.Validators[i].IsByzantine = true
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "byzantine_validator", vid+" 已标记为拜占庭（拒签）", false)
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

	// 1) 双链布局
	prims = append(prims, fw.PrimNodeAt("source-chain", "Source Chain\nlocked="+fmt.Sprintf("%d", st.SourceLocked), "active", "chain", 0.15, 0.5, 1.5))
	prims = append(prims, fw.PrimNodeAt("target-chain", "Target Chain\nminted="+fmt.Sprintf("%d", st.TargetMinted), "active", "chain", 0.85, 0.5, 1.5))
	prims = append(prims, fw.PrimNodeAt("bridge", fmt.Sprintf("Bridge\nN=%d  f=%d  quorum=%d", len(st.Validators), st.Threshold, st.quorum()), "active", "bridge", 0.5, 0.5, 1.3))

	// 2) Validator 节点
	prims = append(prims, fw.PrimRingLayout("validator-ring", len(st.Validators)))
	for _, v := range st.Validators {
		role := "validator"
		status := "normal"
		if v.IsDown {
			role = "down"
			status = "error"
		} else if v.IsByzantine {
			role = "byzantine"
			status = "warning"
		}
		prims = append(prims, fw.PrimNode("v-"+v.ID, v.ID+"\n"+role, status, role))
	}

	// 3) 双链之间 Bridge 流动方向
	prims = append(prims, fw.PrimEdge("flow-1", "source-chain", "bridge", "solid", "flow"))
	prims = append(prims, fw.PrimEdge("flow-2", "bridge", "target-chain", "solid", "flow"))

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-bridge",
		`\text{lock-mint}: src.\text{lock}(v) \to bridge.\text{quorum} \ge 2f{+}1 \to tgt.\text{mint}(v)\\
\text{burn-release}: tgt.\text{burn}(v) \to bridge.\text{quorum} \to src.\text{release}(v)`, false))

	// 5) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("validators = %d  f = %d  quorum = %d\nsource_locked = %d\ntarget_minted = %d\n消息总数 = %d\nblocked_replays = %d\nblocked_quorum = %d",
			len(st.Validators), st.Threshold, st.quorum(),
			st.SourceLocked, st.TargetMinted,
			len(st.Messages), st.BlockedReplays, st.BlockedQuorum),
		"text", nil, 8))

	// 6) 用户余额表
	balRows := []string{"user      source     target"}
	users := map[string]bool{}
	for k := range st.UserSourceBalance {
		users[k] = true
	}
	for k := range st.UserTargetBalance {
		users[k] = true
	}
	keys := []string{}
	for k := range users {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, u := range keys {
		balRows = append(balRows, fmt.Sprintf("  %-8s  %-8d  %d", u, st.UserSourceBalance[u], st.UserTargetBalance[u]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-balances", strings.Join(balRows, "\n"), "text", nil, 10))

	// 7) 消息表
	msgRows := []string{"nonce dir            user     amount sigs/quorum status         tag"}
	startIdx := 0
	if len(st.Messages) > 16 {
		startIdx = len(st.Messages) - 16
	}
	for _, m := range st.Messages[startIdx:] {
		msgRows = append(msgRows, fmt.Sprintf("%-5d %-14s %-8s %-6d %-2d/%-9d %-14s %s",
			m.Nonce, m.Direction, m.User, m.Amount,
			len(m.Signatures), st.quorum(),
			m.Status, m.Tag))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-messages", strings.Join(msgRows, "\n"), "text", nil, 18))

	// 8) 投票矩阵：行 = message，列 = validator
	if len(st.Messages) > 0 && len(st.Validators) > 0 {
		showMsgs := st.Messages
		if len(showMsgs) > 8 {
			showMsgs = showMsgs[len(showMsgs)-8:]
		}
		cells := make([]map[string]any, 0, len(showMsgs)*len(st.Validators))
		for i, m := range showMsgs {
			signed := map[string]bool{}
			for _, s := range m.Signatures {
				signed[s] = true
			}
			for j, v := range st.Validators {
				val := ""
				color := "muted"
				if signed[v.ID] {
					val = "✓"
					color = "success"
				} else if v.IsByzantine {
					color = "warning"
				} else if v.IsDown {
					color = "danger"
				}
				cells = append(cells, map[string]any{
					"row": i, "col": j, "value": val, "color_role": color,
				})
			}
		}
		prims = append(prims, fw.PrimVoteMatrix("sig-matrix", len(showMsgs), len(st.Validators), cells))
	}

	// 9) 进度条：blocked attacks
	prims = append(prims, fw.PrimBar("bar-replays", float64(st.BlockedReplays), 0, "success", "Blocked Replays"))
	prims = append(prims, fw.PrimBar("bar-quorum", float64(st.BlockedQuorum), 0, "success", "Blocked Quorum"))

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-bridge", "bridge", "info", 0.7))
	if len(st.Messages) > 0 {
		last := st.Messages[len(st.Messages)-1]
		switch last.Status {
		case "executed":
			prims = append(prims, fw.PrimBurst("burst-exec", "bridge", "success", int64(last.Nonce), 700))
		case "replay-rejected", "quorum-failed":
			prims = append(prims, fw.PrimShake("shake-block", "bridge", 0.4, 700))
			prims = append(prims, fw.PrimPulse("pulse-block", "bridge", "warning", 1500))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Bridge 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	exec := 0
	for _, m := range st.Messages {
		if m.Status == "executed" {
			exec++
		}
	}
	d := map[string]any{
		"validators":      len(st.Validators),
		"threshold":       st.Threshold,
		"quorum":          st.quorum(),
		"source_locked":   st.SourceLocked,
		"target_minted":   st.TargetMinted,
		"messages":        len(st.Messages),
		"executed":        exec,
		"blocked_replays": st.BlockedReplays,
		"blocked_quorum":  st.BlockedQuorum,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendLockMicroSteps(env *fw.RenderEnvelope, nonce int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "l-1", Label: "user 在 source 链锁仓", DurationMs: 400, HighlightIDs: []string{"source-chain"}},
		{ID: "l-2", Label: fmt.Sprintf("创建跨链消息 nonce=%d", nonce), DurationMs: 400, HighlightIDs: []string{"bridge", "cb-messages"}},
		{ID: "l-3", Label: "等待 validator 签名", DurationMs: 400, HighlightIDs: []string{"validator-ring"}, IsLinkTrigger: true},
	}
}

func appendBurnMicroSteps(env *fw.RenderEnvelope, nonce int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "b-1", Label: "user 在 target 链销毁 wrapped token", DurationMs: 400, HighlightIDs: []string{"target-chain"}},
		{ID: "b-2", Label: fmt.Sprintf("创建 burn-release 消息 nonce=%d", nonce), DurationMs: 400, HighlightIDs: []string{"bridge", "cb-messages"}},
		{ID: "b-3", Label: "等待 validator 签名 → release", DurationMs: 400, HighlightIDs: []string{"validator-ring"}, IsLinkTrigger: true},
	}
}

func appendSignMicroSteps(env *fw.RenderEnvelope, nonce int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "validator 观察 source 链事件", DurationMs: 400, HighlightIDs: []string{"validator-ring"}},
		{ID: "s-2", Label: "签名加入消息", DurationMs: 500, HighlightIDs: []string{"sig-matrix", "cb-messages"}},
		{ID: "s-3", Label: "≥ 2f+1 → 可执行", DurationMs: 500, HighlightIDs: []string{"formula-bridge"}, IsLinkTrigger: true},
	}
}

func appendExecuteMicroSteps(env *fw.RenderEnvelope, dir string) {
	tail := "target.mint(amount)"
	if dir == directionBurnRelease {
		tail = "source.release(amount)"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "e-1", Label: "验证签名数 ≥ quorum", DurationMs: 400, HighlightIDs: []string{"sig-matrix"}},
		{ID: "e-2", Label: "检查 nonce 未 processed", DurationMs: 400, HighlightIDs: []string{"cb-messages"}},
		{ID: "e-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"target-chain", "source-chain"}, FirePrimitives: []string{"burst-exec"}, IsLinkTrigger: true},
	}
}

func appendReplayMicroSteps(env *fw.RenderEnvelope, nonce int, blocked bool) {
	tail := "重放成功（不应该）"
	if blocked {
		tail = "✓ 重放被 nonce 拦截"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "rp-1", Label: fmt.Sprintf("再次提交 nonce=%d", nonce), DurationMs: 400, HighlightIDs: []string{"cb-messages"}},
		{ID: "rp-2", Label: "检查 processed 集合", DurationMs: 400, HighlightIDs: []string{"bridge"}, FirePrimitives: []string{"shake-block"}},
		{ID: "rp-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-replays"}, FirePrimitives: []string{"pulse-block"}, IsLinkTrigger: true},
	}
}

func appendQuorumMicroSteps(env *fw.RenderEnvelope, nonce int, blocked bool) {
	tail := "签名足够，已执行"
	if blocked {
		tail = "✓ 签名 < quorum，被拒"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "qa-1", Label: fmt.Sprintf("提交 nonce=%d 但签名不足", nonce), DurationMs: 400, HighlightIDs: []string{"sig-matrix"}},
		{ID: "qa-2", Label: "执行前 quorum 检查", DurationMs: 400, HighlightIDs: []string{"formula-bridge"}, FirePrimitives: []string{"shake-block"}},
		{ID: "qa-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"bar-quorum"}, FirePrimitives: []string{"pulse-block"}, IsLinkTrigger: true},
	}
}

func appendFailMicroSteps(env *fw.RenderEnvelope, reason string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "f-1", Label: "执行失败：" + reason, DurationMs: 500, HighlightIDs: []string{"cb-messages"}, FirePrimitives: []string{"shake-block"}},
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
		ID:             "bridge-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_bridge",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"tx.bridge.source_locked", "tx.bridge.target_minted"},
		Payload:        map[string]any{"locked": st.SourceLocked, "minted": st.TargetMinted},
		SourceAnchorID: "bridge-anchor",
		TargetAnchorID: "tx-proc-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "tx.bridge.source_locked", "tx.bridge.target_minted")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"tx": map[string]any{
			"bridge": map[string]any{
				"validators":      len(st.Validators),
				"threshold":       st.Threshold,
				"quorum":          st.quorum(),
				"source_locked":   int(st.SourceLocked),
				"target_minted":   int(st.TargetMinted),
				"messages":        len(st.Messages),
				"blocked_replays": st.BlockedReplays,
				"blocked_quorum":  st.BlockedQuorum,
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
