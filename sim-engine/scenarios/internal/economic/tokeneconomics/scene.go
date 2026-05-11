// 模块：sim-engine/scenarios/internal/economic/tokeneconomics
// 文件职责：ECO-01 代币经济场景的完整实现。
//
// SSOT 依据：06.md §4.8.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现完整代币供应模型（零外部依赖）。
//
//   1. 发行模型（同时支持 Bitcoin 风格 halving 和固定通胀）：
//        · MaxSupply        : 上限（0 表示无上限）
//        · BlockReward      : 当前每块发放
//        · HalvingInterval  : 每 N 块奖励减半
//        · FixedInflation   : 若 > 0，则按"年通胀率"线性发行（不走 halving）
//
//   2. 销毁模型：
//        · BurnRate         : 每笔交易费的 burn 比例（EIP-1559 基础）
//        · BuybackPool      : 协议回购销毁池
//        · ManualBurn       : 教学手工 burn
//
//   3. 状态：
//        · TotalSupply      : 累计发行 - 累计销毁
//        · Circulating      : 实际可流通 = TotalSupply - Locked
//        · Locked           : 锁仓（团队/基金会/质押）
//        · TotalIssued      : 历史累计铸造
//        · TotalBurned      : 历史累计销毁
//        · InflationRate    : 当前年化通胀（基于最近 epoch 测算）
//
//   4. 教学操作：
//        · mine_blocks      : 推进 N 块；按当前 reward 铸造
//        · burn_fees        : 模拟一段交易期内累计 fee + burn
//        · halving_now      : 立即执行 halving
//        · lock_tokens      : 锁仓
//        · unlock_tokens    : 解锁
//        · manual_burn      : 协议买回销毁
//
//   5. 教学指标：
//        · supplyCurve      : (block, totalSupply) 曲线
//        · burnRatio        : TotalBurned / TotalIssued
//        · inflationByEpoch : 每 epoch 的发行率

package tokeneconomics

import (
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "token-economics"
	schemaVersion = "v1.0.0"
	algorithmType = "supply-emission-burn"

	defaultMaxSupply       = 21_000_000
	defaultInitialReward   = 50.0
	defaultHalvingInterval = 1000
	defaultBurnRate        = 0.5 // 50% fee burn (EIP-1559)
	defaultBlocksPerYear   = 5_256_000

	linkGroupContractSec = "contract-security-group"
	linkOwnerSubtree     = "economic.token"
)

// =====================================================================
// 数据结构
// =====================================================================

type epochSnap struct {
	BlockHeight   int
	TotalSupply   float64
	Circulating   float64
	Locked        float64
	TotalIssued   float64
	TotalBurned   float64
	BlockReward   float64
	InflationRate float64 // 基于该 epoch 的年化估算
	BurnRatio     float64
}

type supplyEvent struct {
	Tick   int
	Kind   string // mine / burn / halving / lock / unlock / manual_burn
	Amount float64
	Note   string
}

type snapState struct {
	MaxSupply       float64
	HalvingInterval int
	BlockReward     float64
	FixedInflation  float64 // 0 = 走 halving；>0 = 固定年通胀率
	BurnRate        float64
	BlocksPerYear   int

	BlockHeight  int
	TotalIssued  float64
	TotalBurned  float64
	Locked       float64
	HalvingCount int
	Snapshots    []epochSnap
	Events       []supplyEvent
	Tick         int
	LastError    string
}

func (st snapState) totalSupply() float64 { return st.TotalIssued - st.TotalBurned }
func (st snapState) circulating() float64 {
	c := st.totalSupply() - st.Locked
	if c < 0 {
		return 0
	}
	return c
}
func (st snapState) burnRatio() float64 {
	if st.TotalIssued == 0 {
		return 0
	}
	return st.TotalBurned / st.TotalIssued
}

// inflationRate 当前块的年化通胀率（基于 reward 与 totalSupply）。
func (st snapState) inflationRate() float64 {
	supply := st.totalSupply()
	if supply <= 0 {
		return 0
	}
	annualEmit := st.BlockReward * float64(st.BlocksPerYear)
	return annualEmit / supply
}

func defaultSnapState() snapState {
	return snapState{
		MaxSupply:       defaultMaxSupply,
		HalvingInterval: defaultHalvingInterval,
		BlockReward:     defaultInitialReward,
		BurnRate:        defaultBurnRate,
		BlocksPerYear:   defaultBlocksPerYear,
	}
}

// =====================================================================
// 核心操作
// =====================================================================

// mineBlocks 推进 n 个块，每块按 BlockReward 铸造，按 HalvingInterval 减半。
// 若 FixedInflation > 0，则改用"年通胀线性发行"模式：每块发行 = supply * FixedInflation / blocksPerYear。
func (st *snapState) mineBlocks(n int) (float64, []supplyEvent) {
	st.Tick++
	emitted := 0.0
	evs := []supplyEvent{}
	for i := 0; i < n; i++ {
		st.BlockHeight++
		var reward float64
		if st.FixedInflation > 0 {
			reward = st.totalSupply() * st.FixedInflation / float64(st.BlocksPerYear)
		} else {
			reward = st.BlockReward
		}
		// MaxSupply 限制
		if st.MaxSupply > 0 {
			room := st.MaxSupply - st.totalSupply()
			if room <= 0 {
				st.recordEvent("mine", 0, fmt.Sprintf("block %d: MaxSupply 已达，停发", st.BlockHeight))
				continue
			}
			if reward > room {
				reward = room
			}
		}
		st.TotalIssued += reward
		emitted += reward
		evs = append(evs, supplyEvent{Tick: st.Tick, Kind: "mine", Amount: reward,
			Note: fmt.Sprintf("block %d, reward=%.4f", st.BlockHeight, reward)})

		// halving 检查（仅 halving 模式）
		if st.FixedInflation == 0 && st.HalvingInterval > 0 && st.BlockHeight%st.HalvingInterval == 0 {
			st.BlockReward /= 2
			st.HalvingCount++
			ev := supplyEvent{Tick: st.Tick, Kind: "halving", Amount: 0,
				Note: fmt.Sprintf("block %d: halving #%d, reward → %.4f", st.BlockHeight, st.HalvingCount, st.BlockReward)}
			evs = append(evs, ev)
		}
	}
	st.commitEvents(evs)
	st.captureSnapshot()
	return emitted, evs
}

// burnFees 一段时期累计的交易费按 BurnRate 销毁。
func (st *snapState) burnFees(totalFees float64) (float64, supplyEvent) {
	st.Tick++
	burn := totalFees * st.BurnRate
	st.TotalBurned += burn
	ev := supplyEvent{Tick: st.Tick, Kind: "burn", Amount: burn,
		Note: fmt.Sprintf("fees=%.2f burnRate=%.2f → burn=%.4f", totalFees, st.BurnRate, burn)}
	st.commitEvents([]supplyEvent{ev})
	st.captureSnapshot()
	return burn, ev
}

// halvingNow 立即触发一次 halving。
func (st *snapState) halvingNow() supplyEvent {
	st.Tick++
	st.BlockReward /= 2
	st.HalvingCount++
	ev := supplyEvent{Tick: st.Tick, Kind: "halving", Amount: 0,
		Note: fmt.Sprintf("manual halving #%d → reward=%.4f", st.HalvingCount, st.BlockReward)}
	st.commitEvents([]supplyEvent{ev})
	st.captureSnapshot()
	return ev
}

// lockTokens / unlockTokens 调整锁仓量。
func (st *snapState) lockTokens(amount float64) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if st.Locked+amount > st.totalSupply() {
		return fmt.Errorf("锁仓 %.2f 超出 totalSupply %.2f", st.Locked+amount, st.totalSupply())
	}
	st.Tick++
	st.Locked += amount
	st.commitEvents([]supplyEvent{{Tick: st.Tick, Kind: "lock", Amount: amount,
		Note: fmt.Sprintf("lock=+%.2f total locked=%.2f", amount, st.Locked)}})
	st.captureSnapshot()
	return nil
}

func (st *snapState) unlockTokens(amount float64) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if amount > st.Locked {
		return fmt.Errorf("解锁 %.2f 超出锁仓 %.2f", amount, st.Locked)
	}
	st.Tick++
	st.Locked -= amount
	st.commitEvents([]supplyEvent{{Tick: st.Tick, Kind: "unlock", Amount: amount,
		Note: fmt.Sprintf("unlock=-%.2f remaining locked=%.2f", amount, st.Locked)}})
	st.captureSnapshot()
	return nil
}

// manualBurn 协议手动销毁。
func (st *snapState) manualBurn(amount float64, reason string) error {
	if amount <= 0 {
		return errors.New("amount 必须 > 0")
	}
	if amount > st.totalSupply() {
		return fmt.Errorf("销毁 %.2f 超出供应 %.2f", amount, st.totalSupply())
	}
	st.Tick++
	st.TotalBurned += amount
	st.commitEvents([]supplyEvent{{Tick: st.Tick, Kind: "manual_burn", Amount: amount,
		Note: fmt.Sprintf("manual burn -%.2f (%s)", amount, reason)}})
	st.captureSnapshot()
	return nil
}

func (st *snapState) recordEvent(kind string, amount float64, note string) {
	st.commitEvents([]supplyEvent{{Tick: st.Tick, Kind: kind, Amount: amount, Note: note}})
}

func (st *snapState) commitEvents(evs []supplyEvent) {
	st.Events = append(st.Events, evs...)
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

func (st *snapState) captureSnapshot() {
	snap := epochSnap{
		BlockHeight:   st.BlockHeight,
		TotalSupply:   st.totalSupply(),
		Circulating:   st.circulating(),
		Locked:        st.Locked,
		TotalIssued:   st.TotalIssued,
		TotalBurned:   st.TotalBurned,
		BlockReward:   st.BlockReward,
		InflationRate: st.inflationRate(),
		BurnRatio:     st.burnRatio(),
	}
	// 同 BlockHeight 的快照只保留最后一个
	if n := len(st.Snapshots); n > 0 && st.Snapshots[n-1].BlockHeight == snap.BlockHeight {
		st.Snapshots[n-1] = snap
		return
	}
	st.Snapshots = append(st.Snapshots, snap)
	if len(st.Snapshots) > 256 {
		st.Snapshots = st.Snapshots[len(st.Snapshots)-256:]
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
		MaxSupply:       floatOr(d, "max_supply", defaultMaxSupply),
		HalvingInterval: fw.MapInt(d, "halving", defaultHalvingInterval),
		BlockReward:     floatOr(d, "block_reward", defaultInitialReward),
		FixedInflation:  floatOr(d, "fixed_inflation", 0),
		BurnRate:        floatOr(d, "burn_rate", defaultBurnRate),
		BlocksPerYear:   fw.MapInt(d, "blocks_year", defaultBlocksPerYear),
		BlockHeight:     fw.MapInt(d, "height", 0),
		TotalIssued:     floatOr(d, "total_issued", 0),
		TotalBurned:     floatOr(d, "total_burned", 0),
		Locked:          floatOr(d, "locked", 0),
		HalvingCount:    fw.MapInt(d, "halving_count", 0),
		Tick:            fw.MapInt(d, "tick", 0),
		LastError:       fw.MapStr(d, "last_error", ""),
	}
	if snAny, ok := d["snaps"].([]any); ok {
		for _, x := range snAny {
			if m, ok := x.(map[string]any); ok {
				st.Snapshots = append(st.Snapshots, epochSnap{
					BlockHeight:   fw.MapInt(m, "h", 0),
					TotalSupply:   floatOr(m, "ts", 0),
					Circulating:   floatOr(m, "circ", 0),
					Locked:        floatOr(m, "locked", 0),
					TotalIssued:   floatOr(m, "iss", 0),
					TotalBurned:   floatOr(m, "brn", 0),
					BlockReward:   floatOr(m, "rw", 0),
					InflationRate: floatOr(m, "inf", 0),
					BurnRatio:     floatOr(m, "br", 0),
				})
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, supplyEvent{
					Tick:   fw.MapInt(m, "tick", 0),
					Kind:   fw.MapStr(m, "kind", ""),
					Amount: floatOr(m, "amount", 0),
					Note:   fw.MapStr(m, "note", ""),
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
	s.Data["max_supply"] = st.MaxSupply
	s.Data["halving"] = st.HalvingInterval
	s.Data["block_reward"] = st.BlockReward
	s.Data["fixed_inflation"] = st.FixedInflation
	s.Data["burn_rate"] = st.BurnRate
	s.Data["blocks_year"] = st.BlocksPerYear
	s.Data["height"] = st.BlockHeight
	s.Data["total_issued"] = st.TotalIssued
	s.Data["total_burned"] = st.TotalBurned
	s.Data["locked"] = st.Locked
	s.Data["halving_count"] = st.HalvingCount
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	snAny := make([]any, len(st.Snapshots))
	for i, sn := range st.Snapshots {
		snAny[i] = map[string]any{
			"h": sn.BlockHeight, "ts": sn.TotalSupply, "circ": sn.Circulating,
			"locked": sn.Locked, "iss": sn.TotalIssued, "brn": sn.TotalBurned,
			"rw": sn.BlockReward, "inf": sn.InflationRate, "br": sn.BurnRatio,
		}
	}
	s.Data["snaps"] = snAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{"tick": ev.Tick, "kind": ev.Kind,
			"amount": ev.Amount, "note": ev.Note}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "代币经济",
		Description:         "演示发行 / halving / 销毁 / 锁仓 + supply 曲线 + 通胀率随时间衰减",
		Category:            fw.CategoryEconomic,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths:    []string{},

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
				ActionCode: "set_params", Label: "设置参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "max_supply", Type: fw.FieldNumber, Label: "MaxSupply (0=∞)", Required: true, Default: defaultMaxSupply, Min: 0, Step: 1000000},
					{Name: "halving", Type: fw.FieldNumber, Label: "Halving Interval (块)", Required: true, Default: defaultHalvingInterval, Min: 1, Step: 100},
					{Name: "block_reward", Type: fw.FieldNumber, Label: "Initial Block Reward", Required: true, Default: defaultInitialReward, Min: 0, Step: 1},
					{Name: "fixed_inflation", Type: fw.FieldNumber, Label: "FixedInflation (0=halving 模式)", Required: false, Default: 0, Min: 0, Max: 1, Step: 0.01},
					{Name: "burn_rate", Type: fw.FieldNumber, Label: "BurnRate", Required: true, Default: defaultBurnRate, Min: 0, Max: 1, Step: 0.05},
					{Name: "blocks_year", Type: fw.FieldNumber, Label: "BlocksPerYear", Required: true, Default: defaultBlocksPerYear, Min: 1000, Step: 100000},
				},
			},
			{
				ActionCode: "mine_blocks", Label: "出块（铸造）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "块数", Required: true, Default: 100, Min: 1, Step: 10},
				},
			},
			{
				ActionCode: "burn_fees", Label: "burn 一笔费用",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "fees", Type: fw.FieldNumber, Label: "总 fees", Required: true, Default: 100, Min: 0, Step: 10},
				},
			},
			{
				ActionCode: "halving_now", Label: "立即 halving",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
			},
			{
				ActionCode: "lock_tokens", Label: "锁仓",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "amount", Type: fw.FieldNumber, Label: "数量", Required: true, Default: 1000, Min: 1, Step: 100},
				},
			},
			{
				ActionCode: "unlock_tokens", Label: "解锁",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "amount", Type: fw.FieldNumber, Label: "数量", Required: true, Default: 500, Min: 1, Step: 50},
				},
			},
			{
				ActionCode: "manual_burn", Label: "协议销毁",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "amount", Type: fw.FieldNumber, Label: "数量", Required: true, Default: 100, Min: 1, Step: 10},
					{Name: "reason", Type: fw.FieldString, Label: "原因", Required: false, Default: "buyback"},
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
	env := buildEnvelope(st, "init", "Token economics 初始化（Bitcoin-like）", true)
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
		st.MaxSupply = floatOr(in.Params, "max_supply", defaultMaxSupply)
		st.HalvingInterval = fw.MapInt(in.Params, "halving", defaultHalvingInterval)
		st.BlockReward = floatOr(in.Params, "block_reward", defaultInitialReward)
		st.FixedInflation = floatOr(in.Params, "fixed_inflation", 0)
		st.BurnRate = floatOr(in.Params, "burn_rate", defaultBurnRate)
		st.BlocksPerYear = fw.MapInt(in.Params, "blocks_year", defaultBlocksPerYear)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", "参数已更新", true)
		return out, nil

	case "mine_blocks":
		n := fw.MapInt(in.Params, "n", 100)
		emitted, evs := st.mineBlocks(n)
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_blocks",
			fmt.Sprintf("出 %d 块, 共铸造 %.4f", n, emitted), false)
		appendMineMicroSteps(&out.Render, n, emitted, len(evs))
		return out, nil

	case "burn_fees":
		fees := floatOr(in.Params, "fees", 100)
		burned, _ := st.burnFees(fees)
		saveState(state, st)
		out.Render = buildEnvelope(st, "burn_fees",
			fmt.Sprintf("fees=%.2f → burn=%.4f", fees, burned), false)
		appendBurnMicroSteps(&out.Render, burned)
		return out, nil

	case "halving_now":
		ev := st.halvingNow()
		saveState(state, st)
		out.Render = buildEnvelope(st, "halving_now", ev.Note, false)
		appendHalvingMicroSteps(&out.Render)
		return out, nil

	case "lock_tokens":
		amt := floatOr(in.Params, "amount", 1000)
		if err := st.lockTokens(amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "lock_tokens",
			fmt.Sprintf("lock %.2f, total locked=%.2f", amt, st.Locked), false)
		appendLockMicroSteps(&out.Render, "lock", amt)
		return out, nil

	case "unlock_tokens":
		amt := floatOr(in.Params, "amount", 500)
		if err := st.unlockTokens(amt); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "unlock_tokens",
			fmt.Sprintf("unlock %.2f, locked=%.2f", amt, st.Locked), false)
		appendLockMicroSteps(&out.Render, "unlock", amt)
		return out, nil

	case "manual_burn":
		amt := floatOr(in.Params, "amount", 100)
		reason := fw.MapStr(in.Params, "reason", "buyback")
		if err := st.manualBurn(amt, reason); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "manual_burn",
			fmt.Sprintf("manual burn %.2f (%s)", amt, reason), false)
		appendBurnMicroSteps(&out.Render, amt)
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

	// 1) 三块状态卡：Issued / Burned / Circulating
	prims = append(prims, fw.PrimNodeAt("card-issued",
		fmt.Sprintf("Total Issued\n%.2f", st.TotalIssued), "active", "supply-issued", 0.2, 0.2, 1.4))
	prims = append(prims, fw.PrimNodeAt("card-burned",
		fmt.Sprintf("Total Burned\n%.2f", st.TotalBurned), "active", "supply-burned", 0.5, 0.2, 1.4))
	prims = append(prims, fw.PrimNodeAt("card-circ",
		fmt.Sprintf("Circulating\n%.2f", st.circulating()), "active", "supply-circ", 0.8, 0.2, 1.4))

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-supply",
		`\text{TotalSupply} = \text{TotalIssued} - \text{TotalBurned}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-halving",
		`\text{reward}_{i+1} = \text{reward}_i / 2 \quad \text{每 } H \text{ 块}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-inflation",
		`\text{inflation}_{年} = \dfrac{\text{reward} \cdot \text{blocks\_per\_year}}{\text{TotalSupply}}`, false))

	// 3) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("BlockHeight = %d\nMaxSupply = %.0f  HalvingInterval = %d\nBlockReward = %.6f  HalvingCount = %d\nFixedInflation = %.4f  BurnRate = %.2f\nTotalIssued = %.4f  TotalBurned = %.4f\nTotalSupply = %.4f  Circulating = %.4f\nLocked = %.4f\nInflationRate = %.6f  BurnRatio = %.4f",
			st.BlockHeight, st.MaxSupply, st.HalvingInterval,
			st.BlockReward, st.HalvingCount,
			st.FixedInflation, st.BurnRate,
			st.TotalIssued, st.TotalBurned,
			st.totalSupply(), st.circulating(),
			st.Locked,
			st.inflationRate(), st.burnRatio()),
		"text", nil, 14))

	// 4) Supply 饼图（Circulating / Locked / Burned）
	prims = append(prims, fw.PrimPieChart("supply-pie", []map[string]any{
		{"label": "Circulating", "value": st.circulating(), "color_role": "success"},
		{"label": "Locked", "value": st.Locked, "color_role": "info"},
		{"label": "Burned", "value": st.TotalBurned, "color_role": "danger"},
	}))

	// 5) 进度条
	if st.MaxSupply > 0 {
		prims = append(prims, fw.PrimProgressBar("bar-cap",
			st.totalSupply(), st.MaxSupply,
			fmt.Sprintf("Supply %.2f / %.0f (%.2f%%)",
				st.totalSupply(), st.MaxSupply,
				100*st.totalSupply()/st.MaxSupply)))
	}
	prims = append(prims, fw.PrimBar("bar-burnratio", st.burnRatio()*100, 100, "warning",
		fmt.Sprintf("Burn ratio %.2f%%", st.burnRatio()*100)))

	// 6) 曲线：TotalSupply vs BlockHeight
	if len(st.Snapshots) > 0 {
		supplyPts := []map[string]float64{}
		issuedPts := []map[string]float64{}
		burnedPts := []map[string]float64{}
		rewardPts := []map[string]float64{}
		for _, sn := range st.Snapshots {
			x := float64(sn.BlockHeight)
			supplyPts = append(supplyPts, map[string]float64{"x": x, "y": sn.TotalSupply})
			issuedPts = append(issuedPts, map[string]float64{"x": x, "y": sn.TotalIssued})
			burnedPts = append(burnedPts, map[string]float64{"x": x, "y": sn.TotalBurned})
			rewardPts = append(rewardPts, map[string]float64{"x": x, "y": sn.BlockReward})
		}
		prims = append(prims, fw.PrimCurve("curve-supply", "TotalSupply over height", supplyPts, "solid"))
		prims = append(prims, fw.PrimCurve("curve-issued", "TotalIssued (累计铸造)", issuedPts, "dashed"))
		prims = append(prims, fw.PrimCurve("curve-burned", "TotalBurned (累计销毁)", burnedPts, "dotted"))
		prims = append(prims, fw.PrimCurve("curve-reward", "BlockReward (随 halving 衰减)", rewardPts, "solid"))
	}

	// 7) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"tick   kind          amount       note"}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-5d  %-12s  %-10.4f  %s",
				ev.Tick, ev.Kind, ev.Amount, ev.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 18))
	}

	// 8) 动效
	if len(st.Events) > 0 {
		last := st.Events[len(st.Events)-1]
		switch last.Kind {
		case "halving":
			prims = append(prims, fw.PrimBurst("burst-halving", "card-issued", "warning", int64(st.HalvingCount), 700))
			prims = append(prims, fw.PrimPulse("pulse-halving", "formula-halving", "warning", 1500))
		case "burn", "manual_burn":
			prims = append(prims, fw.PrimGlow("glow-burn", "card-burned", "danger", 0.8))
		case "mine":
			prims = append(prims, fw.PrimGlow("glow-mine", "card-issued", "success", 0.7))
		}
	}

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "TokenEconomics 错误", st.LastError, "scene", "请检查参数", true))
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
		"block_height":   st.BlockHeight,
		"total_supply":   st.totalSupply(),
		"circulating":    st.circulating(),
		"locked":         st.Locked,
		"total_issued":   st.TotalIssued,
		"total_burned":   st.TotalBurned,
		"block_reward":   st.BlockReward,
		"halving_count":  st.HalvingCount,
		"inflation_rate": st.inflationRate(),
		"burn_ratio":     st.burnRatio(),
		"tick":           st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendMineMicroSteps(env *fw.RenderEnvelope, n int, emitted float64, evCount int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: fmt.Sprintf("出 %d 块", n), DurationMs: 400, HighlightIDs: []string{"card-issued", "cb-status"}},
		{ID: "m-2", Label: fmt.Sprintf("铸造 %.4f；可能触发 halving", emitted), DurationMs: 500, HighlightIDs: []string{"formula-halving", "curve-reward"}},
		{ID: "m-3", Label: "更新 supply curve", DurationMs: 400, HighlightIDs: []string{"curve-supply", "supply-pie"}, IsLinkTrigger: true},
	}
}

func appendBurnMicroSteps(env *fw.RenderEnvelope, burned float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "b-1", Label: "fees × burnRate = burn 数额", DurationMs: 400, HighlightIDs: []string{"formula-supply"}},
		{ID: "b-2", Label: fmt.Sprintf("销毁 %.4f", burned), DurationMs: 500, HighlightIDs: []string{"card-burned", "supply-pie", "bar-burnratio"}, FirePrimitives: []string{"glow-burn"}, IsLinkTrigger: true},
	}
}

func appendHalvingMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "h-1", Label: "blockReward /= 2", DurationMs: 400, HighlightIDs: []string{"formula-halving"}},
		{ID: "h-2", Label: "halving_count++", DurationMs: 400, HighlightIDs: []string{"cb-status", "curve-reward"}, FirePrimitives: []string{"burst-halving", "pulse-halving"}, IsLinkTrigger: true},
	}
}

func appendLockMicroSteps(env *fw.RenderEnvelope, kind string, amount float64) {
	tag := "锁仓"
	if kind == "unlock" {
		tag = "解锁"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "l-1", Label: fmt.Sprintf("%s %.2f", tag, amount), DurationMs: 400, HighlightIDs: []string{"supply-pie"}},
		{ID: "l-2", Label: "更新 Circulating / Locked", DurationMs: 400, HighlightIDs: []string{"card-circ", "cb-status"}, IsLinkTrigger: true},
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
		ID:             "token-econ-update",
		SourceScene:    sceneCode,
		SourceAction:   "advance_epoch",
		LinkGroup:      linkGroupContractSec,
		ChangedFields:  []string{"economic.token.total_supply", "economic.token.circulating"},
		Payload:        map[string]any{"total_supply": st.totalSupply(), "circulating": st.circulating()},
		SourceAnchorID: "token-econ-anchor",
		TargetAnchorID: "econ-group-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
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
