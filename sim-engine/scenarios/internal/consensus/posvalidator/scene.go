// 模块：sim-engine/scenarios/internal/consensus/posvalidator
// 文件职责：CON-02 权益证明（PoS）验证者选举场景的完整实现。
//
// SSOT 依据：06.md §4.2.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 PoS 加权随机出块者选择 + 验证者奖励 + slashing 扣押 +
// 委托质押（LSD），零第三方加密库；复用 sha256hash.Sum256 提供确定性随机：
//
//   · 出块者选择：seed_e = SHA-256(global_seed || epoch_be64) → 大端整数 mod total_stake
//     落在哪个验证者的 stake 区间即被选中（与抵押权重严格成正比）
//   · 奖励：当选验证者获得固定奖励（block_reward），增加其 stake
//   · Slashing：恶意行为（双签 / 离线）扣除 slash_pct% stake，不参与选举
//   · LSD：用户把 stake 委托给某验证者，按比例分享奖励 / slashing 风险
//
// 教学决策：
//   - ring_layout 把 N 个验证者均匀环形排列
//   - pie_chart 显示 stake 占比
//   - 当前出块者 glow + burst
//   - risk_gauge 显示中心化程度（HHI 指数）

package posvalidator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "pos-validator"
	schemaVersion = "v1.0.0"
	algorithmType = "pos-stake-weighted"

	defaultBlockReward = 1
	maxValidators      = 16
	maxEpochHistory    = 32

	linkGroupPosEconomy = "pos-economy-group"
	linkOwnerSubtree    = "validators.pos"
)

// =====================================================================
// 验证者 / 状态
// =====================================================================

type validator struct {
	ID          string
	Stake       int64
	BlocksMined int
	Slashed     bool
	SlashedAt   int // epoch
	Delegators  []delegator
}

type delegator struct {
	UserID string
	Amount int64
}

type epochRecord struct {
	Epoch    int
	Producer string
	Reward   int64
	Slashed  string
}

type snapState struct {
	Validators   []validator
	GlobalSeed   string
	BlockReward  int64
	SlashPctBp   int // 10000 = 100%
	CurrentEpoch int
	History      []epochRecord
	LastError    string
}

// defaultValidators 4 个验证者，stake 按比例分配（演示中心化程度）。
func defaultValidators() []validator {
	return []validator{
		{ID: "alice", Stake: 100},
		{ID: "bob", Stake: 200},
		{ID: "carol", Stake: 300},
		{ID: "dave", Stake: 400},
	}
}

func defaultSnapState() snapState {
	return snapState{
		Validators:  defaultValidators(),
		GlobalSeed:  "lenschain-pos-2026",
		BlockReward: defaultBlockReward,
		SlashPctBp:  3000, // 默认扣 30%
	}
}

// totalStake 返回所有未 Slash 验证者 stake 总和。
func (st snapState) totalStake() int64 {
	var sum int64
	for _, v := range st.Validators {
		if !v.Slashed {
			sum += v.Stake
		}
	}
	return sum
}

// pickProducer 给定 epoch + global_seed，按 stake 加权选出 producer ID。
// 算法：seed = SHA-256(global_seed || epoch_be64)，把 hash 解读为大整数 mod total_stake，
// 然后累加 stake 找到落在哪个区间的验证者。
func (st snapState) pickProducer(epoch int) (string, int) {
	total := st.totalStake()
	if total <= 0 {
		return "", -1
	}
	var buf []byte
	buf = append(buf, []byte(st.GlobalSeed)...)
	var ep [8]byte
	binary.BigEndian.PutUint64(ep[:], uint64(epoch))
	buf = append(buf, ep[:]...)
	h := sha256hash.Sum256(buf)
	hashInt := new(big.Int).SetBytes(h[:])
	pick := new(big.Int).Mod(hashInt, big.NewInt(total)).Int64()
	cum := int64(0)
	for i, v := range st.Validators {
		if v.Slashed {
			continue
		}
		cum += v.Stake
		if pick < cum {
			return v.ID, i
		}
	}
	// 浮点边界（理论不会触发）
	for i, v := range st.Validators {
		if !v.Slashed {
			return v.ID, i
		}
	}
	return "", -1
}

// hhiIndex 计算 stake 分布的赫芬达尔-赫希曼指数（HHI），衡量中心化（10000 = 完全垄断）。
func (st snapState) hhiIndex() float64 {
	total := float64(st.totalStake())
	if total == 0 {
		return 0
	}
	hhi := 0.0
	for _, v := range st.Validators {
		if v.Slashed {
			continue
		}
		share := float64(v.Stake) / total
		hhi += share * share * 10000
	}
	return hhi
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
		GlobalSeed:   fw.MapStr(d, "global_seed", "lenschain-pos-2026"),
		BlockReward:  int64(fw.MapInt(d, "block_reward", defaultBlockReward)),
		SlashPctBp:   fw.MapInt(d, "slash_pct_bp", 3000),
		CurrentEpoch: fw.MapInt(d, "current_epoch", 0),
		LastError:    fw.MapStr(d, "last_error", ""),
	}
	if vsAny, ok := d["validators"].([]any); ok {
		for _, vAny := range vsAny {
			if vm, ok := vAny.(map[string]any); ok {
				v := validator{
					ID:          fw.MapStr(vm, "id", ""),
					Stake:       int64(fw.MapInt(vm, "stake", 0)),
					BlocksMined: fw.MapInt(vm, "blocks", 0),
					Slashed:     fw.MapBool(vm, "slashed", false),
					SlashedAt:   fw.MapInt(vm, "slashed_at", 0),
				}
				if delsAny, ok := vm["delegators"].([]any); ok {
					for _, dAny := range delsAny {
						if dm, ok := dAny.(map[string]any); ok {
							v.Delegators = append(v.Delegators, delegator{
								UserID: fw.MapStr(dm, "user", ""),
								Amount: int64(fw.MapInt(dm, "amount", 0)),
							})
						}
					}
				}
				st.Validators = append(st.Validators, v)
			}
		}
	}
	if len(st.Validators) == 0 {
		st.Validators = defaultValidators()
	}
	if hist, ok := d["history"].([]any); ok {
		for _, hAny := range hist {
			if hm, ok := hAny.(map[string]any); ok {
				st.History = append(st.History, epochRecord{
					Epoch:    fw.MapInt(hm, "epoch", 0),
					Producer: fw.MapStr(hm, "producer", ""),
					Reward:   int64(fw.MapInt(hm, "reward", 0)),
					Slashed:  fw.MapStr(hm, "slashed", ""),
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
	s.Data["global_seed"] = st.GlobalSeed
	s.Data["block_reward"] = st.BlockReward
	s.Data["slash_pct_bp"] = st.SlashPctBp
	s.Data["current_epoch"] = st.CurrentEpoch
	s.Data["last_error"] = st.LastError
	vs := make([]any, len(st.Validators))
	for i, v := range st.Validators {
		dels := make([]any, len(v.Delegators))
		for j, d := range v.Delegators {
			dels[j] = map[string]any{"user": d.UserID, "amount": d.Amount}
		}
		vs[i] = map[string]any{
			"id":         v.ID,
			"stake":      v.Stake,
			"blocks":     v.BlocksMined,
			"slashed":    v.Slashed,
			"slashed_at": v.SlashedAt,
			"delegators": dels,
		}
	}
	s.Data["validators"] = vs
	hist := make([]any, len(st.History))
	for i, h := range st.History {
		hist[i] = map[string]any{
			"epoch":    h.Epoch,
			"producer": h.Producer,
			"reward":   h.Reward,
			"slashed":  h.Slashed,
		}
	}
	s.Data["history"] = hist
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "权益证明（PoS 验证者）",
		Description:         "演示 PoS 加权随机选举、出块奖励、slashing、委托质押（LSD）",
		Category:            fw.CategoryConsensus,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPosEconomy},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"validators.pos.current_producer",
			"validators.pos.epoch",
			"validators.pos.slashed_set",
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
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data:      map[string]any{},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_validators", Label: "设置验证者集",
				Description: "格式: id1:stake1,id2:stake2,...（≤ 16 个）",
				Category:    fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "validators_csv", Type: fw.FieldString, Label: "验证者 CSV", Required: true,
						Default: "alice:100,bob:200,carol:300,dave:400"},
					{Name: "global_seed", Type: fw.FieldString, Label: "全局随机种子", Required: false, Default: "lenschain-pos-2026"},
				},
			},
			{
				ActionCode: "run_epoch", Label: "推进 1 个 Epoch",
				Description: "按 stake 加权选出 producer，分发奖励",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"validators.pos.current_producer", "validators.pos.epoch"},
				LinkOwnerFields:   []string{"validators.pos.current_producer", "validators.pos.epoch"},
			},
			{
				ActionCode: "run_n_epochs", Label: "推进 N 个 Epoch",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "Epoch 数", Required: true, Default: 10, Min: 1, Max: 1000, Step: 1},
				},
				WritesOwnedFields: []string{"validators.pos.epoch"},
				LinkOwnerFields:   []string{"validators.pos.epoch"},
			},
			{
				ActionCode: "slash_validator", Label: "扣押恶意验证者",
				Description: "对指定验证者执行 slash（按 slash_pct 扣 stake，移出选举）",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "validator_id", Type: fw.FieldString, Label: "验证者 ID", Required: true, Default: "dave"},
				},
				WritesOwnedFields: []string{"validators.pos.slashed_set"},
				LinkOwnerFields:   []string{"validators.pos.slashed_set"},
			},
			{
				ActionCode: "delegate_stake", Label: "委托质押（LSD）",
				Description: "用户把 stake 委托给某验证者",
				Category:    fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "user_id", Type: fw.FieldString, Label: "用户 ID", Required: true, Default: "user-1"},
					{Name: "validator_id", Type: fw.FieldString, Label: "验证者 ID", Required: true, Default: "alice"},
					{Name: "amount", Type: fw.FieldNumber, Label: "委托数量", Required: true, Default: 50, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_inject_fault",
				Label:         "教师注入故障",
				Description:   "仅教师可用，注入故障用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneFault,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "故障描述", Required: false, Default: "教师注入故障"},
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
	state.Phase = "epoch-0"
	env := buildEnvelope(st, "init", "PoS 初始化（4 个默认验证者）", true)
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

	switch in.ActionCode {
	case "set_validators":
		csv := fw.MapStr(in.Params, "validators_csv", "")
		seed := fw.MapStr(in.Params, "global_seed", "lenschain-pos-2026")
		vs, err := parseValidators(csv)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.Validators = vs
		st.GlobalSeed = seed
		st.CurrentEpoch = 0
		st.History = nil
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_validators", fmt.Sprintf("已设置 %d 个验证者", len(vs)), true)
		appendSetValidatorsMicroSteps(&out.Render)
		return out, nil

	case "run_epoch":
		producer, idx := st.pickProducer(st.CurrentEpoch)
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "无可用验证者（全部被 slash 或 stake=0）"}, nil
		}
		st.Validators[idx].Stake += st.BlockReward
		st.Validators[idx].BlocksMined++
		rec := epochRecord{Epoch: st.CurrentEpoch, Producer: producer, Reward: st.BlockReward}
		st.History = append(st.History, rec)
		if len(st.History) > maxEpochHistory {
			st.History = st.History[len(st.History)-maxEpochHistory:]
		}
		st.CurrentEpoch++
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_epoch", fmt.Sprintf("Epoch %d → 出块者 %s（奖励 %d）", rec.Epoch, producer, rec.Reward), false)
		appendRunEpochMicroSteps(&out.Render, producer, idx)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "run_n_epochs":
		n := fw.MapInt(in.Params, "n", 10)
		if n < 1 {
			n = 1
		}
		if n > 1000 {
			n = 1000
		}
		var lastProducer string
		var lastIdx int
		for i := 0; i < n; i++ {
			producer, idx := st.pickProducer(st.CurrentEpoch)
			if idx < 0 {
				return fw.ActionOutput{Success: false, ErrorMessage: "中途无可用验证者"}, nil
			}
			st.Validators[idx].Stake += st.BlockReward
			st.Validators[idx].BlocksMined++
			rec := epochRecord{Epoch: st.CurrentEpoch, Producer: producer, Reward: st.BlockReward}
			st.History = append(st.History, rec)
			st.CurrentEpoch++
			lastProducer = producer
			lastIdx = idx
		}
		if len(st.History) > maxEpochHistory {
			st.History = st.History[len(st.History)-maxEpochHistory:]
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "run_n_epochs", fmt.Sprintf("推进 %d epoch（终态出块者 %s）", n, lastProducer), false)
		appendRunEpochMicroSteps(&out.Render, lastProducer, lastIdx)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "slash_validator":
		vid := fw.MapStr(in.Params, "validator_id", "")
		idx := -1
		for i, v := range st.Validators {
			if v.ID == vid {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到验证者: " + vid}, nil
		}
		if st.Validators[idx].Slashed {
			return fw.ActionOutput{Success: false, ErrorMessage: vid + " 已被 slash"}, nil
		}
		st.Validators[idx].Stake = st.Validators[idx].Stake * int64(10000-st.SlashPctBp) / 10000
		st.Validators[idx].Slashed = true
		st.Validators[idx].SlashedAt = st.CurrentEpoch
		saveState(state, st)
		out.Render = buildEnvelope(st, "slash_validator", fmt.Sprintf("已 slash %s（扣 %.1f%%）", vid, float64(st.SlashPctBp)/100), false)
		appendSlashMicroSteps(&out.Render, vid, idx)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "delegate_stake":
		user := fw.MapStr(in.Params, "user_id", "user-1")
		vid := fw.MapStr(in.Params, "validator_id", "")
		amount := int64(fw.MapInt(in.Params, "amount", 50))
		idx := -1
		for i, v := range st.Validators {
			if v.ID == vid {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到验证者: " + vid}, nil
		}
		if amount <= 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "委托量必须 > 0"}, nil
		}
		st.Validators[idx].Stake += amount
		st.Validators[idx].Delegators = append(st.Validators[idx].Delegators, delegator{UserID: user, Amount: amount})
		saveState(state, st)
		out.Render = buildEnvelope(st, "delegate_stake", fmt.Sprintf("%s 委托 %d 给 %s", user, amount, vid), false)
		appendDelegateMicroSteps(&out.Render, idx)
		return out, nil

	case "teacher_inject_fault":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师注入故障"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-fault-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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

// parseValidators 解析 "id1:stake1,id2:stake2,..." 格式。
func parseValidators(csv string) ([]validator, error) {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil, errors.New("CSV 为空")
	}
	parts := strings.Split(csv, ",")
	if len(parts) > maxValidators {
		return nil, fmt.Errorf("验证者数量 ≤ %d", maxValidators)
	}
	out := make([]validator, 0, len(parts))
	for _, p := range parts {
		kv := strings.Split(strings.TrimSpace(p), ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("格式错误: %s", p)
		}
		stake, err := atoi64(strings.TrimSpace(kv[1]))
		if err != nil || stake < 0 {
			return nil, fmt.Errorf("stake 非法: %s", p)
		}
		out = append(out, validator{ID: strings.TrimSpace(kv[0]), Stake: stake})
	}
	if len(out) == 0 {
		return nil, errors.New("至少 1 个验证者")
	}
	return out, nil
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 30)

	// 1) 环形布局：N 个验证者
	nodeIDs := make([]string, 0, len(st.Validators))
	for _, v := range st.Validators {
		nodeIDs = append(nodeIDs, "node-"+v.ID)
	}
	prims = append(prims, fw.PrimRingLayout("validator-ring", len(st.Validators)))

	// 2) 验证者节点
	currentProducer := ""
	if len(st.History) > 0 {
		currentProducer = st.History[len(st.History)-1].Producer
	}
	for _, v := range st.Validators {
		status := "normal"
		role := "validator"
		if v.Slashed {
			status = "error"
			role = "slashed"
		} else if v.ID == currentProducer {
			status = "active"
			role = "producer"
		}
		label := fmt.Sprintf("%s\nstake=%d\n出块=%d", v.ID, v.Stake, v.BlocksMined)
		if v.Slashed {
			label = fmt.Sprintf("%s\n[SLASHED]\nstake=%d", v.ID, v.Stake)
		}
		prims = append(prims, fw.PrimNode("node-"+v.ID, label, status, role))
	}

	// 3) Stake 占比饼图
	segments := make([]map[string]any, 0, len(st.Validators))
	for _, v := range st.Validators {
		colorRole := "info"
		if v.Slashed {
			colorRole = "danger"
		} else if v.ID == currentProducer {
			colorRole = "success"
		}
		segments = append(segments, map[string]any{
			"label":      v.ID,
			"value":      float64(v.Stake),
			"color_role": colorRole,
		})
	}
	prims = append(prims, fw.PrimPieChart("stake-pie", segments))

	// 4) HHI 中心化仪表
	hhi := st.hhiIndex()
	prims = append(prims, fw.PrimRiskGauge("hhi-gauge", hhi,
		[]map[string]any{
			{"from": 0.0, "to": 1500.0, "color": "success"},    // 高度分散
			{"from": 1500.0, "to": 2500.0, "color": "warning"}, // 中度集中
			{"from": 2500.0, "to": 10000.0, "color": "danger"}, // 高度集中
		},
	))

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-pick",
		`P(\text{producer}=v) = \frac{\mathrm{stake}_v}{\sum_i \mathrm{stake}_i};\ \ \text{seed}_e = \mathrm{SHA256}(\text{global\_seed}\,\|\,e)`, false))

	// 6) Epoch 信息
	prims = append(prims, fw.PrimCodeBlock("cb-epoch",
		fmt.Sprintf("当前 Epoch = %d\n总 stake = %d\n出块者 = %s\nHHI = %.1f / 10000", st.CurrentEpoch, st.totalStake(), currentProducer, hhi),
		"text", nil, 4))

	// 7) 验证者表
	rows := []string{"id        stake    blocks  status      delegators"}
	for _, v := range st.Validators {
		statusStr := "active"
		if v.Slashed {
			statusStr = fmt.Sprintf("SLASHED(%d)", v.SlashedAt)
		}
		rows = append(rows, fmt.Sprintf("%-9s %-7d  %-6d  %-11s %d 个",
			v.ID, v.Stake, v.BlocksMined, statusStr, len(v.Delegators)))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-table", strings.Join(rows, "\n"), "text", nil, 12))

	// 8) Epoch 历史
	histLines := []string{"Epoch  Producer  Reward"}
	startIdx := 0
	if len(st.History) > 12 {
		startIdx = len(st.History) - 12
		histLines = append(histLines, "  …")
	}
	for _, h := range st.History[startIdx:] {
		histLines = append(histLines, fmt.Sprintf("  %-4d  %-8s  +%d", h.Epoch, h.Producer, h.Reward))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(histLines, "\n"), "text", nil, 14))

	// 9) 动效
	if currentProducer != "" {
		prims = append(prims, fw.PrimGlow("glow-producer", "node-"+currentProducer, "success", 0.9))
		prims = append(prims, fw.PrimBurst("burst-producer", "node-"+currentProducer, "success",
			int64(st.CurrentEpoch), 700))
	}
	for _, v := range st.Validators {
		if v.Slashed {
			prims = append(prims, fw.PrimGlow("glow-slash-"+v.ID, "node-"+v.ID, "danger", 0.7))
			prims = append(prims, fw.PrimShake("shake-"+v.ID, "node-"+v.ID, 0.4, 600))
		}
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-pos-econ", linkGroupPosEconomy, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "PoS 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	currentProducer := ""
	if len(st.History) > 0 {
		currentProducer = st.History[len(st.History)-1].Producer
	}
	slashedSet := []string{}
	for _, v := range st.Validators {
		if v.Slashed {
			slashedSet = append(slashedSet, v.ID)
		}
	}
	d := map[string]any{
		"current_epoch":    st.CurrentEpoch,
		"current_producer": currentProducer,
		"validator_count":  len(st.Validators),
		"total_stake":      st.totalStake(),
		"hhi_index":        st.hhiIndex(),
		"slashed_set":      slashedSet,
		"block_reward":     st.BlockReward,
		"history_count":    len(st.History),
		"global_seed":      st.GlobalSeed,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendSetValidatorsMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sv-1", Label: "解析验证者集", DurationMs: 400, HighlightIDs: []string{"validator-ring"}, ParentPhase: "setup"},
		{ID: "sv-2", Label: "计算 stake 占比", DurationMs: 500, HighlightIDs: []string{"stake-pie"}},
		{ID: "sv-3", Label: "评估 HHI 中心化指数", DurationMs: 500, HighlightIDs: []string{"hhi-gauge"}},
	}
}

func appendRunEpochMicroSteps(env *fw.RenderEnvelope, producer string, idx int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "re-1", Label: "seed_e = SHA-256(global_seed || epoch)", DurationMs: 400, HighlightIDs: []string{"formula-pick", "cb-epoch"}},
		{ID: "re-2", Label: "hash mod total_stake → 落入区间", DurationMs: 500, HighlightIDs: []string{"stake-pie"}, FirePrimitives: []string{"glow-producer"}},
		{ID: "re-3", Label: fmt.Sprintf("出块者 = %s（奖励入账）", producer), DurationMs: 600, HighlightIDs: []string{"cb-table", "cb-history"}, FirePrimitives: []string{"burst-producer"}, IsLinkTrigger: true},
	}
}

func appendSlashMicroSteps(env *fw.RenderEnvelope, vid string, idx int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sl-1", Label: "检测 " + vid + " 恶意行为", DurationMs: 400, HighlightIDs: []string{"node-" + vid}, FirePrimitives: []string{"shake-" + vid}},
		{ID: "sl-2", Label: "扣除 stake（slash_pct）", DurationMs: 500, HighlightIDs: []string{"cb-table"}},
		{ID: "sl-3", Label: vid + " 移出选举池", DurationMs: 500, HighlightIDs: []string{"validator-ring", "stake-pie"}, IsLinkTrigger: true},
	}
}

func appendDelegateMicroSteps(env *fw.RenderEnvelope, idx int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "dl-1", Label: "委托人转入 stake", DurationMs: 400, HighlightIDs: []string{"cb-table"}},
		{ID: "dl-2", Label: "更新验证者总 stake", DurationMs: 400, HighlightIDs: []string{"stake-pie"}},
		{ID: "dl-3", Label: "重算 HHI", DurationMs: 400, HighlightIDs: []string{"hhi-gauge"}},
	}
}

// =====================================================================
// 联动
// =====================================================================

// currentProducerOrEmpty 取最近一个 epoch 的出块者；无记录返回空串。
func currentProducerOrEmpty(st snapState) string {
	if len(st.History) == 0 {
		return ""
	}
	return st.History[len(st.History)-1].Producer
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "pos-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_validators",
		LinkGroup:      linkGroupPosEconomy,
		ChangedFields:  []string{"validators.pos.current_producer", "validators.pos.epoch"},
		Payload: map[string]any{
			"epoch":    st.CurrentEpoch,
			"producer": currentProducerOrEmpty(st),
		},
		SourceAnchorID: "pos-output-anchor",
		TargetAnchorID: "economy-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "validators.pos.epoch", "validators.pos.current_producer")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	currentProducer := ""
	if len(st.History) > 0 {
		currentProducer = st.History[len(st.History)-1].Producer
	}
	slashedSet := []string{}
	for _, v := range st.Validators {
		if v.Slashed {
			slashedSet = append(slashedSet, v.ID)
		}
	}
	return map[string]any{
		"validators": map[string]any{
			"pos": map[string]any{
				"epoch":            st.CurrentEpoch,
				"current_producer": currentProducer,
				"total_stake":      st.totalStake(),
				"validator_count":  len(st.Validators),
				"slashed_set":      slashedSet,
				"hhi_index":        st.hhiIndex(),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func atoi64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	var n int64
	sign := int64(1)
	i := 0
	if s[0] == '-' {
		sign = -1
		i = 1
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("非法字符 %c", c)
		}
		n = n*10 + int64(c-'0')
	}
	return n * sign, nil
}
