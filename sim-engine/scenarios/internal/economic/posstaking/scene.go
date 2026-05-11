// 模块：sim-engine/scenarios/internal/economic/posstaking
// 文件职责：ECO-02 PoS 质押经济场景的完整实现。
//
// SSOT 依据：06.md §4.8.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 PoS 质押与 slashing 经济模型（零外部依赖）。
//
//   1. 验证人 / 委托人模型：
//      · validator { id, selfStake, commissionRate, status, jailed, slashedAmount, ... }
//      · delegation { delegator, validator, amount, pendingRewards, unbondingEntries[] }
//      · delegationShares 不实现 share/exchange-rate 抽象，按教学版直接记录金额
//
//   2. 经济参数：
//      · BaseInflation        : 基础年通胀（教学：6%）
//      · TargetStakingRate    : 目标质押率（教学：67%）
//      · MaxStakingRate       : 上限（100%）
//      · APR(stakingRate) 公式：
//          if r < target: APR = baseInflation / r       （奖励高，吸引质押）
//          if r ≥ target: APR = baseInflation * target² / r²  （收益快速衰减）
//      · CommissionRate       : 验证人抽成（默认 10%）
//      · UnbondingDuration    : 21 epoch（教学）
//      · SlashFractionDouble  : double-sign 5%
//      · SlashFractionDowntime: 下线 0.1%
//      · MinJailDuration      : jail 时长 8 epoch
//
//   3. 操作流：
//      · register_validator
//      · delegate / undelegate（产生 unbondingEntry，到期才返还）
//      · advance_epoch（推进 1 epoch，分发 reward，处理 unbond 到期）
//      · slash_validator（double-sign / downtime）
//      · jail / unjail
//      · withdraw_rewards / restake
//
//   4. 教学指标：
//      · totalStake / totalRewardsDistributed / totalSlashed
//      · activeStakingRate（质押 / 总供应量；总供应量教学固定 1_000_000）
//      · validatorAPR / delegatorAPR（扣完佣金）

package posstaking

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "pos-staking"
	schemaVersion = "v1.0.0"
	algorithmType = "pos-staking-slashing"

	defaultTotalSupply       = 1_000_000.0
	defaultBaseInflation     = 0.06
	defaultTargetStakingRate = 0.67
	defaultUnbondingDuration = 21
	defaultMinJailDuration   = 8
	defaultSlashDoubleSign   = 0.05
	defaultSlashDowntime     = 0.001
	defaultCommissionRate    = 0.10

	statusActive   = "active"
	statusInactive = "inactive"
	statusJailed   = "jailed"

	linkGroupPosEcon = "pos-economy-group"
	linkOwnerSubtree = "economic.staking"
)

// =====================================================================
// 数据结构
// =====================================================================

type validator struct {
	ID             string
	SelfStake      float64
	DelegatedStake float64
	Commission     float64
	Status         string
	Jailed         bool
	JailUntilEpoch int
	SlashedAmount  float64
	AccRewardCum   float64 // 验证人累计佣金
	DowntimeStreak int     // 连续未参与 epoch 数
	DoubleSignCnt  int
	DowntimeCnt    int
}

func (v validator) totalStake() float64 { return v.SelfStake + v.DelegatedStake }

type unbondingEntry struct {
	Amount        float64
	CompleteEpoch int
}

type delegation struct {
	Delegator      string
	Validator      string
	Amount         float64
	PendingRewards float64
	Unbondings     []unbondingEntry
}

func (d delegation) totalUnbonding() float64 {
	s := 0.0
	for _, u := range d.Unbondings {
		s += u.Amount
	}
	return s
}

type epochSnapshot struct {
	Epoch           int
	TotalStake      float64
	StakingRate     float64
	APR             float64
	RewardThisEpoch float64
	TotalRewards    float64
	TotalSlashed    float64
	ValidatorCount  int
	ActiveCount     int
}

type stakingEvent struct {
	Tick   int
	Epoch  int
	Kind   string
	Note   string
	Amount float64
}

type snapState struct {
	TotalSupply       float64
	BaseInflation     float64
	TargetStakingRate float64
	UnbondingDuration int
	MinJailDuration   int
	SlashDoubleSign   float64
	SlashDowntime     float64

	Epoch           int
	Tick            int
	Validators      map[string]*validator
	Delegations     []*delegation
	TotalRewardsCum float64
	TotalSlashedCum float64
	Snapshots       []epochSnapshot
	Events          []stakingEvent
	LastError       string
}

func defaultSnapState() snapState {
	st := snapState{
		TotalSupply:       defaultTotalSupply,
		BaseInflation:     defaultBaseInflation,
		TargetStakingRate: defaultTargetStakingRate,
		UnbondingDuration: defaultUnbondingDuration,
		MinJailDuration:   defaultMinJailDuration,
		SlashDoubleSign:   defaultSlashDoubleSign,
		SlashDowntime:     defaultSlashDowntime,
		Validators:        map[string]*validator{},
	}
	// 默认 3 个 validator，alice / bob 各 delegate 一些
	st.registerValidator("v1", 10000, 0.10)
	st.registerValidator("v2", 8000, 0.05)
	st.registerValidator("v3", 5000, 0.20)
	st.delegate("alice", "v1", 5000)
	st.delegate("alice", "v2", 3000)
	st.delegate("bob", "v1", 2000)
	st.captureSnapshot()
	return st
}

// =====================================================================
// 核心：注册、委托、解委托
// =====================================================================

func (st *snapState) registerValidator(id string, selfStake, commission float64) error {
	if _, ok := st.Validators[id]; ok {
		return fmt.Errorf("validator %s 已存在", id)
	}
	if selfStake <= 0 {
		return errors.New("selfStake 必须 > 0")
	}
	if commission < 0 || commission > 1 {
		return errors.New("commission 必须 ∈ [0,1]")
	}
	v := &validator{ID: id, SelfStake: selfStake, Commission: commission, Status: statusActive}
	st.Validators[id] = v
	st.recordEvent("register_validator",
		fmt.Sprintf("v=%s self=%.2f commission=%.2f", id, selfStake, commission), selfStake)
	return nil
}

// findDelegation 找指定 (delegator, validator) 的委托记录。
func (st *snapState) findDelegation(d, v string) *delegation {
	for _, x := range st.Delegations {
		if x.Delegator == d && x.Validator == v {
			return x
		}
	}
	return nil
}

func (st *snapState) delegate(d, v string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	val, ok := st.Validators[v]
	if !ok {
		return fmt.Errorf("validator %s 不存在", v)
	}
	if val.Jailed {
		return fmt.Errorf("validator %s 处于 jail 状态", v)
	}
	val.DelegatedStake += amount
	dele := st.findDelegation(d, v)
	if dele == nil {
		dele = &delegation{Delegator: d, Validator: v}
		st.Delegations = append(st.Delegations, dele)
	}
	dele.Amount += amount
	st.recordEvent("delegate",
		fmt.Sprintf("%s → %s amount=%.2f", d, v, amount), amount)
	return nil
}

func (st *snapState) undelegate(d, v string, amount float64) error {
	dele := st.findDelegation(d, v)
	if dele == nil {
		return fmt.Errorf("没有 %s → %s 的委托", d, v)
	}
	if amount <= 0 || amount > dele.Amount {
		return fmt.Errorf("amount 越界（active=%.2f）", dele.Amount)
	}
	val := st.Validators[v]
	val.DelegatedStake -= amount
	dele.Amount -= amount
	complete := st.Epoch + st.UnbondingDuration
	dele.Unbondings = append(dele.Unbondings, unbondingEntry{Amount: amount, CompleteEpoch: complete})
	st.recordEvent("undelegate",
		fmt.Sprintf("%s → %s amount=%.2f, 解锁 epoch=%d", d, v, amount, complete), amount)
	return nil
}

// =====================================================================
// 经济模型
// =====================================================================

// totalActiveStake 全网活跃质押。
func (st snapState) totalActiveStake() float64 {
	s := 0.0
	for _, v := range st.Validators {
		if v.Status == statusActive && !v.Jailed {
			s += v.totalStake()
		}
	}
	return s
}

// stakingRate 当前质押率。
func (st snapState) stakingRate() float64 {
	if st.TotalSupply <= 0 {
		return 0
	}
	return st.totalActiveStake() / st.TotalSupply
}

// currentAPR 根据 staking rate 计算 APR。
func (st snapState) currentAPR() float64 {
	r := st.stakingRate()
	if r <= 0 {
		return 0
	}
	if r < st.TargetStakingRate {
		return st.BaseInflation / r
	}
	t := st.TargetStakingRate
	return st.BaseInflation * (t * t) / (r * r)
}

// epochReward 每 epoch 总奖励 = 全网 active stake × APR / 12（教学：1 epoch = 1 月）。
func (st snapState) epochReward() float64 {
	return st.totalActiveStake() * st.currentAPR() / 12.0
}

// advanceEpoch 推进一个 epoch：分发奖励、处理 unbonding、检测 jail 解除、清理 downtime。
func (st *snapState) advanceEpoch() epochSnapshot {
	st.Tick++
	st.Epoch++
	rewardPool := st.epochReward()
	totalActive := st.totalActiveStake()

	if rewardPool > 0 && totalActive > 0 {
		// 按 active validator 的 stake 比例分配
		for _, v := range st.Validators {
			if v.Status != statusActive || v.Jailed {
				continue
			}
			share := v.totalStake() / totalActive
			vReward := rewardPool * share
			// 验证人佣金
			commission := vReward * v.Commission
			v.AccRewardCum += commission
			delegPool := vReward - commission
			// 按 selfStake / totalStake 分给 validator-self；剩余按 delegation amount 比例
			selfShare := v.SelfStake / v.totalStake()
			selfReward := delegPool * selfShare
			delegReward := delegPool - selfReward
			// validator self 奖励算到 accRewardCum
			v.AccRewardCum += selfReward
			// 其余 delegPool 按 delegation 比例分配
			if v.DelegatedStake > 0 {
				for _, d := range st.Delegations {
					if d.Validator == v.ID && d.Amount > 0 {
						r := delegReward * d.Amount / v.DelegatedStake
						d.PendingRewards += r
					}
				}
			}
			st.TotalRewardsCum += vReward
		}
	}

	// 处理 unbonding 到期
	for _, d := range st.Delegations {
		remain := []unbondingEntry{}
		for _, u := range d.Unbondings {
			if u.CompleteEpoch <= st.Epoch {
				st.recordEvent("unbond_complete",
					fmt.Sprintf("%s ← %s 释放 %.2f", d.Delegator, d.Validator, u.Amount), u.Amount)
				continue
			}
			remain = append(remain, u)
		}
		d.Unbondings = remain
	}

	// 处理 jail 解除
	for _, v := range st.Validators {
		if v.Jailed && st.Epoch >= v.JailUntilEpoch {
			v.Jailed = false
			v.Status = statusActive
			st.recordEvent("auto_unjail",
				fmt.Sprintf("v=%s 自动解除 jail（epoch=%d ≥ until=%d）", v.ID, st.Epoch, v.JailUntilEpoch), 0)
		}
	}

	st.recordEvent("advance_epoch",
		fmt.Sprintf("epoch=%d reward=%.4f totalActive=%.2f apr=%.4f rate=%.4f",
			st.Epoch, rewardPool, totalActive, st.currentAPR(), st.stakingRate()), rewardPool)
	return st.captureSnapshot()
}

// slash 切罚某 validator（按 fraction 削减 selfStake + DelegatedStake，按比例从 delegations 扣）。
func (st *snapState) slash(vID string, fraction float64, reason string) error {
	v, ok := st.Validators[vID]
	if !ok {
		return fmt.Errorf("validator %s 不存在", vID)
	}
	if fraction <= 0 || fraction > 1 {
		return errors.New("fraction 必须 ∈ (0,1]")
	}
	cutSelf := v.SelfStake * fraction
	cutDeleg := v.DelegatedStake * fraction
	v.SelfStake -= cutSelf
	v.DelegatedStake -= cutDeleg
	v.SlashedAmount += (cutSelf + cutDeleg)
	st.TotalSlashedCum += cutSelf + cutDeleg
	if reason == "double-sign" {
		v.DoubleSignCnt++
	} else {
		v.DowntimeCnt++
	}
	// 按比例扣每个 delegation
	for _, d := range st.Delegations {
		if d.Validator != vID {
			continue
		}
		cut := d.Amount * fraction
		d.Amount -= cut
	}
	// jail
	v.Jailed = true
	v.Status = statusJailed
	v.JailUntilEpoch = st.Epoch + st.MinJailDuration
	st.recordEvent("slash",
		fmt.Sprintf("v=%s reason=%s fraction=%.4f cut=%.4f → jail until=%d",
			vID, reason, fraction, cutSelf+cutDeleg, v.JailUntilEpoch), cutSelf+cutDeleg)
	return nil
}

// withdrawRewards 提取某 delegation 的累积奖励（设置为 0，返回 amount）。
func (st *snapState) withdrawRewards(delegator, vID string) (float64, error) {
	d := st.findDelegation(delegator, vID)
	if d == nil {
		return 0, fmt.Errorf("没有 %s → %s 的委托", delegator, vID)
	}
	r := d.PendingRewards
	d.PendingRewards = 0
	st.recordEvent("withdraw_rewards",
		fmt.Sprintf("%s ← %s rewards=%.4f", delegator, vID, r), r)
	return r, nil
}

// restakeRewards 把 PendingRewards 直接复利重新 delegate。
func (st *snapState) restakeRewards(delegator, vID string) (float64, error) {
	d := st.findDelegation(delegator, vID)
	if d == nil {
		return 0, fmt.Errorf("没有 %s → %s 的委托", delegator, vID)
	}
	if d.PendingRewards <= 0 {
		return 0, errors.New("没有可复利的奖励")
	}
	r := d.PendingRewards
	d.PendingRewards = 0
	v := st.Validators[vID]
	v.DelegatedStake += r
	d.Amount += r
	st.recordEvent("restake",
		fmt.Sprintf("%s 复利 %.4f → %s", delegator, r, vID), r)
	return r, nil
}

// =====================================================================
// 持久化
// =====================================================================

func (st *snapState) recordEvent(kind, note string, amount float64) {
	st.Events = append(st.Events, stakingEvent{Tick: st.Tick, Epoch: st.Epoch,
		Kind: kind, Note: note, Amount: amount})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

func (st *snapState) captureSnapshot() epochSnapshot {
	active := 0
	for _, v := range st.Validators {
		if v.Status == statusActive && !v.Jailed {
			active++
		}
	}
	snap := epochSnapshot{
		Epoch:           st.Epoch,
		TotalStake:      st.totalActiveStake(),
		StakingRate:     st.stakingRate(),
		APR:             st.currentAPR(),
		RewardThisEpoch: st.epochReward(),
		TotalRewards:    st.TotalRewardsCum,
		TotalSlashed:    st.TotalSlashedCum,
		ValidatorCount:  len(st.Validators),
		ActiveCount:     active,
	}
	// 同 epoch 覆盖最后一条
	if n := len(st.Snapshots); n > 0 && st.Snapshots[n-1].Epoch == snap.Epoch {
		st.Snapshots[n-1] = snap
		return snap
	}
	st.Snapshots = append(st.Snapshots, snap)
	if len(st.Snapshots) > 256 {
		st.Snapshots = st.Snapshots[len(st.Snapshots)-256:]
	}
	return snap
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		TotalSupply:       floatOr(d, "total_supply", defaultTotalSupply),
		BaseInflation:     floatOr(d, "base_inflation", defaultBaseInflation),
		TargetStakingRate: floatOr(d, "target_rate", defaultTargetStakingRate),
		UnbondingDuration: fw.MapInt(d, "unbonding", defaultUnbondingDuration),
		MinJailDuration:   fw.MapInt(d, "jail_dur", defaultMinJailDuration),
		SlashDoubleSign:   floatOr(d, "slash_ds", defaultSlashDoubleSign),
		SlashDowntime:     floatOr(d, "slash_dt", defaultSlashDowntime),
		Epoch:             fw.MapInt(d, "epoch", 0),
		Tick:              fw.MapInt(d, "tick", 0),
		TotalRewardsCum:   floatOr(d, "rewards_cum", 0),
		TotalSlashedCum:   floatOr(d, "slashed_cum", 0),
		LastError:         fw.MapStr(d, "last_error", ""),
		Validators:        map[string]*validator{},
	}
	if vAny, ok := d["validators"].(map[string]any); ok {
		for id, x := range vAny {
			if m, ok := x.(map[string]any); ok {
				st.Validators[id] = &validator{
					ID:             id,
					SelfStake:      floatOr(m, "self", 0),
					DelegatedStake: floatOr(m, "deleg", 0),
					Commission:     floatOr(m, "commission", 0),
					Status:         fw.MapStr(m, "status", statusActive),
					Jailed:         fw.MapBool(m, "jailed", false),
					JailUntilEpoch: fw.MapInt(m, "jail_until", 0),
					SlashedAmount:  floatOr(m, "slashed", 0),
					AccRewardCum:   floatOr(m, "acc_reward", 0),
					DowntimeStreak: fw.MapInt(m, "downtime_streak", 0),
					DoubleSignCnt:  fw.MapInt(m, "double_sign", 0),
					DowntimeCnt:    fw.MapInt(m, "downtime", 0),
				}
			}
		}
	}
	if dAny, ok := d["delegations"].([]any); ok {
		for _, x := range dAny {
			if m, ok := x.(map[string]any); ok {
				dele := &delegation{
					Delegator:      fw.MapStr(m, "d", ""),
					Validator:      fw.MapStr(m, "v", ""),
					Amount:         floatOr(m, "amt", 0),
					PendingRewards: floatOr(m, "pending", 0),
				}
				if uAny, ok := m["unbond"].([]any); ok {
					for _, y := range uAny {
						if um, ok := y.(map[string]any); ok {
							dele.Unbondings = append(dele.Unbondings, unbondingEntry{
								Amount:        floatOr(um, "amount", 0),
								CompleteEpoch: fw.MapInt(um, "epoch", 0),
							})
						}
					}
				}
				st.Delegations = append(st.Delegations, dele)
			}
		}
	}
	if sAny, ok := d["snaps"].([]any); ok {
		for _, x := range sAny {
			if m, ok := x.(map[string]any); ok {
				st.Snapshots = append(st.Snapshots, epochSnapshot{
					Epoch:           fw.MapInt(m, "epoch", 0),
					TotalStake:      floatOr(m, "ts", 0),
					StakingRate:     floatOr(m, "rate", 0),
					APR:             floatOr(m, "apr", 0),
					RewardThisEpoch: floatOr(m, "reward", 0),
					TotalRewards:    floatOr(m, "tr", 0),
					TotalSlashed:    floatOr(m, "tslash", 0),
					ValidatorCount:  fw.MapInt(m, "vc", 0),
					ActiveCount:     fw.MapInt(m, "ac", 0),
				})
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, stakingEvent{
					Tick: fw.MapInt(m, "tick", 0), Epoch: fw.MapInt(m, "epoch", 0),
					Kind: fw.MapStr(m, "kind", ""), Note: fw.MapStr(m, "note", ""),
					Amount: floatOr(m, "amount", 0),
				})
			}
		}
	}
	if len(st.Validators) == 0 {
		return defaultSnapState()
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["total_supply"] = st.TotalSupply
	s.Data["base_inflation"] = st.BaseInflation
	s.Data["target_rate"] = st.TargetStakingRate
	s.Data["unbonding"] = st.UnbondingDuration
	s.Data["jail_dur"] = st.MinJailDuration
	s.Data["slash_ds"] = st.SlashDoubleSign
	s.Data["slash_dt"] = st.SlashDowntime
	s.Data["epoch"] = st.Epoch
	s.Data["tick"] = st.Tick
	s.Data["rewards_cum"] = st.TotalRewardsCum
	s.Data["slashed_cum"] = st.TotalSlashedCum
	s.Data["last_error"] = st.LastError
	vAny := map[string]any{}
	for id, v := range st.Validators {
		vAny[id] = map[string]any{
			"self": v.SelfStake, "deleg": v.DelegatedStake,
			"commission": v.Commission, "status": v.Status,
			"jailed": v.Jailed, "jail_until": v.JailUntilEpoch,
			"slashed": v.SlashedAmount, "acc_reward": v.AccRewardCum,
			"downtime_streak": v.DowntimeStreak,
			"double_sign":     v.DoubleSignCnt, "downtime": v.DowntimeCnt,
		}
	}
	s.Data["validators"] = vAny
	dAny := make([]any, len(st.Delegations))
	for i, d := range st.Delegations {
		uAny := make([]any, len(d.Unbondings))
		for j, u := range d.Unbondings {
			uAny[j] = map[string]any{"amount": u.Amount, "epoch": u.CompleteEpoch}
		}
		dAny[i] = map[string]any{
			"d": d.Delegator, "v": d.Validator,
			"amt": d.Amount, "pending": d.PendingRewards, "unbond": uAny,
		}
	}
	s.Data["delegations"] = dAny
	sAny := make([]any, len(st.Snapshots))
	for i, sn := range st.Snapshots {
		sAny[i] = map[string]any{
			"epoch": sn.Epoch, "ts": sn.TotalStake, "rate": sn.StakingRate,
			"apr": sn.APR, "reward": sn.RewardThisEpoch,
			"tr": sn.TotalRewards, "tslash": sn.TotalSlashed,
			"vc": sn.ValidatorCount, "ac": sn.ActiveCount,
		}
	}
	s.Data["snaps"] = sAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{
			"tick": ev.Tick, "epoch": ev.Epoch, "kind": ev.Kind,
			"note": ev.Note, "amount": ev.Amount,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "PoS 质押经济",
		Description:         "演示 validator/delegation + APR 函数 + unbonding period + slashing + commission + 复利",
		Category:            fw.CategoryEconomic,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupPosEcon},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"economic.staking.staking_rate",
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
				ActionCode: "set_params", Label: "经济参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "total_supply", Type: fw.FieldNumber, Label: "TotalSupply", Required: true, Default: defaultTotalSupply, Min: 1000, Step: 1000},
					{Name: "base_inflation", Type: fw.FieldNumber, Label: "BaseInflation", Required: true, Default: defaultBaseInflation, Min: 0, Max: 1, Step: 0.01},
					{Name: "target_rate", Type: fw.FieldNumber, Label: "TargetStakingRate", Required: true, Default: defaultTargetStakingRate, Min: 0, Max: 1, Step: 0.01},
					{Name: "unbonding", Type: fw.FieldNumber, Label: "UnbondingDuration (epoch)", Required: true, Default: defaultUnbondingDuration, Min: 1, Step: 1},
					{Name: "jail_dur", Type: fw.FieldNumber, Label: "MinJailDuration", Required: true, Default: defaultMinJailDuration, Min: 1, Step: 1},
					{Name: "slash_ds", Type: fw.FieldNumber, Label: "Slash double-sign", Required: true, Default: defaultSlashDoubleSign, Min: 0, Max: 1, Step: 0.01},
					{Name: "slash_dt", Type: fw.FieldNumber, Label: "Slash downtime", Required: true, Default: defaultSlashDowntime, Min: 0, Max: 1, Step: 0.001},
				},
			},
			{
				ActionCode: "register_validator", Label: "注册 validator",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "id", Type: fw.FieldString, Label: "validator id", Required: true, Default: "v4"},
					{Name: "self_stake", Type: fw.FieldNumber, Label: "selfStake", Required: true, Default: 5000, Min: 1, Step: 100},
					{Name: "commission", Type: fw.FieldNumber, Label: "commission", Required: true, Default: defaultCommissionRate, Min: 0, Max: 1, Step: 0.01},
				},
			},
			{
				ActionCode: "delegate", Label: "委托",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "delegator", Type: fw.FieldString, Label: "delegator", Required: true, Default: "alice"},
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v1"},
					{Name: "amount", Type: fw.FieldNumber, Label: "金额", Required: true, Default: 1000, Min: 1, Step: 100},
				},
			},
			{
				ActionCode: "undelegate", Label: "解委托",
				Description:   "进入 unbonding queue，到期才返还",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "delegator", Type: fw.FieldString, Label: "delegator", Required: true, Default: "alice"},
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v1"},
					{Name: "amount", Type: fw.FieldNumber, Label: "金额", Required: true, Default: 500, Min: 1, Step: 100},
				},
			},
			{
				ActionCode: "advance_epoch", Label: "推进 epoch",
				Description: "分发 reward + 处理 unbonding 到期 + 自动 unjail",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.InterveneEpoch,
				WritesOwnedFields: []string{"economic.staking.staking_rate"},
				LinkOwnerFields:   []string{"economic.staking.staking_rate"},
			},
			{
				ActionCode: "advance_n", Label: "推进 N epoch",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneEpoch,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "epoch 数", Required: true, Default: 12, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "slash_double_sign", Label: "切罚（double-sign）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v1"},
				},
				LinkOwnerFields: []string{"economic.staking.total_slashed"},
			},
			{
				ActionCode: "slash_downtime", Label: "切罚（downtime）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v2"},
				},
				LinkOwnerFields: []string{"economic.staking.total_slashed"},
			},
			{
				ActionCode: "withdraw_rewards", Label: "提取奖励",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "delegator", Type: fw.FieldString, Label: "delegator", Required: true, Default: "alice"},
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v1"},
				},
			},
			{
				ActionCode: "restake", Label: "复利重新质押",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "delegator", Type: fw.FieldString, Label: "delegator", Required: true, Default: "alice"},
					{Name: "validator", Type: fw.FieldString, Label: "validator", Required: true, Default: "v1"},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_force_epoch",
				Label:         "教师强制纪元推进",
				Description:   "仅教师可用，强制纪元推进用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师强制纪元推进"},
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
	env := buildEnvelope(st, "init", "PoS 初始化（3 validators + 3 delegations）", true)
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
	case "set_params":
		st.TotalSupply = floatOr(in.Params, "total_supply", defaultTotalSupply)
		st.BaseInflation = floatOr(in.Params, "base_inflation", defaultBaseInflation)
		st.TargetStakingRate = floatOr(in.Params, "target_rate", defaultTargetStakingRate)
		st.UnbondingDuration = fw.MapInt(in.Params, "unbonding", defaultUnbondingDuration)
		st.MinJailDuration = fw.MapInt(in.Params, "jail_dur", defaultMinJailDuration)
		st.SlashDoubleSign = floatOr(in.Params, "slash_ds", defaultSlashDoubleSign)
		st.SlashDowntime = floatOr(in.Params, "slash_dt", defaultSlashDowntime)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", "参数已更新", false)
		return out, nil

	case "register_validator":
		id := fw.MapStr(in.Params, "id", "v4")
		self := floatOr(in.Params, "self_stake", 5000)
		comm := floatOr(in.Params, "commission", defaultCommissionRate)
		if err := st.registerValidator(id, self, comm); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "register_validator",
			fmt.Sprintf("注册 %s self=%.2f commission=%.2f", id, self, comm), false)
		appendRegisterMicroSteps(&out.Render, id)
		return out, nil

	case "delegate":
		d := fw.MapStr(in.Params, "delegator", "alice")
		v := fw.MapStr(in.Params, "validator", "v1")
		amt := floatOr(in.Params, "amount", 1000)
		if err := st.delegate(d, v, amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "delegate",
			fmt.Sprintf("%s → %s amount=%.2f", d, v, amt), false)
		appendDelegateMicroSteps(&out.Render, d, v, amt)
		return out, nil

	case "undelegate":
		d := fw.MapStr(in.Params, "delegator", "alice")
		v := fw.MapStr(in.Params, "validator", "v1")
		amt := floatOr(in.Params, "amount", 500)
		if err := st.undelegate(d, v, amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "undelegate",
			fmt.Sprintf("%s ← %s amount=%.2f, 等待 %d epoch 解锁", d, v, amt, st.UnbondingDuration), false)
		appendUndelegateMicroSteps(&out.Render)
		return out, nil

	case "advance_epoch":
		snap := st.advanceEpoch()
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_epoch",
			fmt.Sprintf("epoch=%d reward=%.4f apr=%.4f rate=%.4f", snap.Epoch, snap.RewardThisEpoch, snap.APR, snap.StakingRate), false)
		appendAdvanceMicroSteps(&out.Render, snap)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "advance_n":
		n := fw.MapInt(in.Params, "n", 12)
		var snap epochSnapshot
		for i := 0; i < n; i++ {
			snap = st.advanceEpoch()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_n",
			fmt.Sprintf("推进 %d epoch；最终 epoch=%d totalRewards=%.4f", n, snap.Epoch, st.TotalRewardsCum), false)
		appendAdvanceMicroSteps(&out.Render, snap)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "slash_double_sign":
		v := fw.MapStr(in.Params, "validator", "v1")
		if err := st.slash(v, st.SlashDoubleSign, "double-sign"); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "slash_double_sign",
			fmt.Sprintf("✗ slash %s double-sign %.2f%%", v, st.SlashDoubleSign*100), false)
		appendSlashMicroSteps(&out.Render, v, "double-sign", st.SlashDoubleSign)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "slash_downtime":
		v := fw.MapStr(in.Params, "validator", "v2")
		if err := st.slash(v, st.SlashDowntime, "downtime"); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "slash_downtime",
			fmt.Sprintf("✗ slash %s downtime %.3f%%", v, st.SlashDowntime*100), false)
		appendSlashMicroSteps(&out.Render, v, "downtime", st.SlashDowntime)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "withdraw_rewards":
		d := fw.MapStr(in.Params, "delegator", "alice")
		v := fw.MapStr(in.Params, "validator", "v1")
		r, err := st.withdrawRewards(d, v)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "withdraw_rewards",
			fmt.Sprintf("%s 提取 %.4f from %s", d, r, v), false)
		appendWithdrawMicroSteps(&out.Render, r)
		return out, nil

	case "restake":
		d := fw.MapStr(in.Params, "delegator", "alice")
		v := fw.MapStr(in.Params, "validator", "v1")
		r, err := st.restakeRewards(d, v)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "restake",
			fmt.Sprintf("%s 复利 %.4f → %s", d, r, v), false)
		appendRestakeMicroSteps(&out.Render, r)
		return out, nil

	case "teacher_force_epoch":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师强制纪元推进"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-epoch-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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

	// 1) Validator 节点（环形）
	vIDs := []string{}
	for k := range st.Validators {
		vIDs = append(vIDs, k)
	}
	sort.Strings(vIDs)
	prims = append(prims, fw.PrimRingLayout("validator-ring", len(vIDs)))
	for _, id := range vIDs {
		v := st.Validators[id]
		role := "validator-active"
		status := "active"
		if v.Jailed {
			role = "validator-jailed"
			status = "error"
		} else if v.Status == statusInactive {
			role = "validator-inactive"
			status = "normal"
		}
		label := fmt.Sprintf("%s\nself=%.0f\ndeleg=%.0f\nc=%.0f%%", id, v.SelfStake, v.DelegatedStake, v.Commission*100)
		if v.Jailed {
			label += fmt.Sprintf("\nJAIL until %d", v.JailUntilEpoch)
		}
		prims = append(prims, fw.PrimNode("v-"+id, label, status, role))
	}

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-apr",
		`\text{APR}(r) = \begin{cases}\dfrac{\text{baseInflation}}{r} & r < \text{target}\\ \text{baseInflation}\cdot\dfrac{\text{target}^2}{r^2} & r \ge \text{target}\end{cases}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-reward",
		`\text{epochReward} = \text{totalActiveStake} \times \text{APR} / 12`, false))
	prims = append(prims, fw.PrimMathFormula("formula-slash",
		`\text{slashed} = \text{stake} \times \text{slashFraction};\quad \text{jail until epoch} + \text{minJailDuration}`, false))

	// 3) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("epoch = %d  tick = %d\nTotalSupply = %.0f  BaseInflation = %.4f  Target = %.4f\nUnbondingDuration = %d  MinJailDuration = %d\nSlash double-sign = %.4f  downtime = %.4f\ntotalActive = %.4f  StakingRate = %.4f  APR = %.4f\nTotalRewards = %.4f  TotalSlashed = %.4f",
			st.Epoch, st.Tick,
			st.TotalSupply, st.BaseInflation, st.TargetStakingRate,
			st.UnbondingDuration, st.MinJailDuration,
			st.SlashDoubleSign, st.SlashDowntime,
			st.totalActiveStake(), st.stakingRate(), st.currentAPR(),
			st.TotalRewardsCum, st.TotalSlashedCum),
		"text", nil, 12))

	// 4) Validator 表
	if len(vIDs) > 0 {
		vLines := []string{"id    self     deleg     total     comm   status     jailed  ds  dt  slashed       acc_rew"}
		for _, id := range vIDs {
			v := st.Validators[id]
			vLines = append(vLines, fmt.Sprintf("  %-4s  %-7.2f  %-8.2f  %-8.2f  %-5.2f  %-9s  %-5v   %-2d  %-2d  %-12.4f  %.4f",
				v.ID, v.SelfStake, v.DelegatedStake, v.totalStake(),
				v.Commission, v.Status, v.Jailed,
				v.DoubleSignCnt, v.DowntimeCnt, v.SlashedAmount, v.AccRewardCum))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-validators", strings.Join(vLines, "\n"), "text", nil, 14))
	}

	// 5) Delegation 表
	if len(st.Delegations) > 0 {
		dLines := []string{"delegator  validator  amount    pending    unbonding"}
		for _, d := range st.Delegations {
			dLines = append(dLines, fmt.Sprintf("  %-9s  %-9s  %-8.2f  %-9.4f  %.4f",
				d.Delegator, d.Validator, d.Amount, d.PendingRewards, d.totalUnbonding()))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-delegations", strings.Join(dLines, "\n"), "text", nil, 14))
	}

	// 6) 进度条
	prims = append(prims, fw.PrimProgressBar("bar-staking",
		st.stakingRate()*100, 100,
		fmt.Sprintf("Staking rate %.2f%% (target %.2f%%)",
			st.stakingRate()*100, st.TargetStakingRate*100)))
	prims = append(prims, fw.PrimBar("bar-rewards", st.TotalRewardsCum, 0, "success", fmt.Sprintf("Total Rewards %.4f", st.TotalRewardsCum)))
	prims = append(prims, fw.PrimBar("bar-slashed", st.TotalSlashedCum, 0, "danger", fmt.Sprintf("Total Slashed %.4f", st.TotalSlashedCum)))

	// 7) 曲线：staking rate / APR / cumulative rewards over epoch
	if len(st.Snapshots) > 0 {
		ratePts := []map[string]float64{}
		aprPts := []map[string]float64{}
		rewPts := []map[string]float64{}
		for _, sn := range st.Snapshots {
			x := float64(sn.Epoch)
			ratePts = append(ratePts, map[string]float64{"x": x, "y": sn.StakingRate})
			aprPts = append(aprPts, map[string]float64{"x": x, "y": sn.APR})
			rewPts = append(rewPts, map[string]float64{"x": x, "y": sn.TotalRewards})
		}
		prims = append(prims, fw.PrimCurve("curve-rate", "Staking rate (随 epoch)", ratePts, "solid"))
		prims = append(prims, fw.PrimCurve("curve-apr", "APR (随 epoch)", aprPts, "dashed"))
		prims = append(prims, fw.PrimCurve("curve-rewards", "Cumulative Rewards", rewPts, "dotted"))
	}

	// 8) 饼图
	piesegs := []map[string]any{}
	for _, id := range vIDs {
		v := st.Validators[id]
		piesegs = append(piesegs, map[string]any{
			"label":      id,
			"value":      v.totalStake(),
			"color_role": ifThenStr(v.Jailed, "warning", "success"),
		})
	}
	prims = append(prims, fw.PrimPieChart("pie-stake", piesegs))

	// 9) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"epoch tick   kind                amount         note"}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-5d %-5d  %-18s  %-13.4f  %s",
				ev.Epoch, ev.Tick, ev.Kind, ev.Amount, ev.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 18))
	}

	// 10) 动效
	if st.TotalSlashedCum > 0 {
		prims = append(prims, fw.PrimShake("shake-slash", "bar-slashed", 0.5, 800))
		prims = append(prims, fw.PrimPulse("pulse-slash", "bar-slashed", "danger", 1500))
	}
	if st.TotalRewardsCum > 0 {
		prims = append(prims, fw.PrimGlow("glow-rewards", "bar-rewards", "success", 0.7))
	}

	// 11) 联动
	prims = append(prims, fw.PrimLinkIndicator("link-pos", linkGroupPosEcon, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "PoS Staking 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func ifThenStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"epoch":           st.Epoch,
		"validator_count": len(st.Validators),
		"delegations":     len(st.Delegations),
		"total_stake":     st.totalActiveStake(),
		"staking_rate":    st.stakingRate(),
		"apr":             st.currentAPR(),
		"epoch_reward":    st.epochReward(),
		"total_rewards":   st.TotalRewardsCum,
		"total_slashed":   st.TotalSlashedCum,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendRegisterMicroSteps(env *fw.RenderEnvelope, id string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "r-1", Label: "validator " + id + " 创建", DurationMs: 400, HighlightIDs: []string{"validator-ring", "v-" + id}},
		{ID: "r-2", Label: "更新总质押", DurationMs: 400, HighlightIDs: []string{"cb-status", "pie-stake"}, IsLinkTrigger: true},
	}
}

func appendDelegateMicroSteps(env *fw.RenderEnvelope, d, v string, amt float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "d-1", Label: fmt.Sprintf("%s 委托 %.2f → %s", d, amt, v), DurationMs: 400, HighlightIDs: []string{"v-" + v, "cb-delegations"}},
		{ID: "d-2", Label: "更新 staking rate / APR", DurationMs: 500, HighlightIDs: []string{"formula-apr", "bar-staking"}, IsLinkTrigger: true},
	}
}

func appendUndelegateMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "u-1", Label: "构造 unbondingEntry", DurationMs: 400, HighlightIDs: []string{"cb-delegations"}},
		{ID: "u-2", Label: "等待 UnbondingDuration epoch 后释放", DurationMs: 500, HighlightIDs: []string{"cb-events"}, IsLinkTrigger: true},
	}
}

func appendAdvanceMicroSteps(env *fw.RenderEnvelope, snap epochSnapshot) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "a-1", Label: fmt.Sprintf("epoch=%d 计算 APR=%.4f", snap.Epoch, snap.APR), DurationMs: 400, HighlightIDs: []string{"formula-apr", "curve-apr"}},
		{ID: "a-2", Label: fmt.Sprintf("分发 reward=%.4f", snap.RewardThisEpoch), DurationMs: 500, HighlightIDs: []string{"formula-reward", "cb-validators", "cb-delegations"}, FirePrimitives: []string{"glow-rewards"}},
		{ID: "a-3", Label: "处理 unbonding 到期 + 自动 unjail", DurationMs: 400, HighlightIDs: []string{"cb-events"}, IsLinkTrigger: true},
	}
}

func appendSlashMicroSteps(env *fw.RenderEnvelope, v, kind string, fraction float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: fmt.Sprintf("检测 %s 行为：%s", v, kind), DurationMs: 400, HighlightIDs: []string{"v-" + v}},
		{ID: "s-2", Label: fmt.Sprintf("按 %.4f 切罚 stake", fraction), DurationMs: 500, HighlightIDs: []string{"formula-slash", "cb-validators"}, FirePrimitives: []string{"shake-slash", "pulse-slash"}},
		{ID: "s-3", Label: fmt.Sprintf("jail until epoch+%d", defaultMinJailDuration), DurationMs: 400, HighlightIDs: []string{"bar-slashed"}, IsLinkTrigger: true},
	}
}

func appendWithdrawMicroSteps(env *fw.RenderEnvelope, r float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "w-1", Label: fmt.Sprintf("提取 %.4f", r), DurationMs: 400, HighlightIDs: []string{"cb-delegations"}},
		{ID: "w-2", Label: "PendingRewards → 0", DurationMs: 400, HighlightIDs: []string{"cb-events"}, IsLinkTrigger: true},
	}
}

func appendRestakeMicroSteps(env *fw.RenderEnvelope, r float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "rs-1", Label: fmt.Sprintf("把 %.4f 重新委托", r), DurationMs: 400, HighlightIDs: []string{"cb-delegations"}},
		{ID: "rs-2", Label: "delegated stake 增加 → 提升下一 epoch reward", DurationMs: 500, HighlightIDs: []string{"formula-reward", "curve-rate"}, IsLinkTrigger: true},
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
		ID:             "pos-staking-update",
		SourceScene:    sceneCode,
		SourceAction:   "epoch_advance",
		LinkGroup:      linkGroupPosEcon,
		ChangedFields:  []string{"economic.staking.total_staked", "economic.staking.epoch"},
		Payload:        map[string]any{"epoch": st.Epoch, "total_staked": st.totalActiveStake()},
		SourceAnchorID: "pos-staking-anchor",
		TargetAnchorID: "econ-group-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"economic": map[string]any{
			"staking": map[string]any{
				"epoch":           st.Epoch,
				"validator_count": len(st.Validators),
				"total_active":    st.totalActiveStake(),
				"staking_rate":    st.stakingRate(),
				"apr":             st.currentAPR(),
				"epoch_reward":    st.epochReward(),
				"total_rewards":   st.TotalRewardsCum,
				"total_slashed":   st.TotalSlashedCum,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func floatOr(m map[string]any, k string, def float64) float64 {
	if m == nil {
		return def
	}
	if v, ok := m[k]; ok {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		case int64:
			return float64(t)
		}
	}
	return def
}
