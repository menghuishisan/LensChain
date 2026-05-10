// 模块：sim-engine/scenarios/internal/economic/gasmarket
// 文件职责：ECO-05 Gas 市场（宏观视角）场景的完整实现。
//
// SSOT 依据：06.md §4.8.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以"用户群体 - 矿工市场"为视角的 Gas 市场长期演化模型（零外部依赖）。
//
// 与 TX-02 (GasCalculation) 的区别：
//   · TX-02 聚焦微观：单个 tx 的 base_fee + priority + 打包逻辑
//   · ECO-05 聚焦宏观：群体需求曲线、长期均衡、拒绝率、收入构成
//
//   1. 用户群（多用户类）：
//      · userClass { name, arrivalRate, wtp_mean, wtp_std, gasLimit }
//      · 每 tick 按 arrivalRate（泊松-like）生成 tx；wtp（愿付 gasPrice）服从正态分布
//
//   2. 用户决策：
//      · effectiveGasPrice ≥ wtp ? 不发起 (rejected)
//        否则发起 → mempool；priority = wtp - base_fee
//
//   3. 矿工 / 验证人：
//      · gasTarget / gasLimit
//      · 按 priority 降序打包，直到 sum(gasUsed) ≥ gasLimit
//
//   4. EIP-1559 base_fee 自适应：
//      · base_fee_{n+1} = base_fee_n × (1 + 1/8 × (gasUsed - target)/target)
//      · burn = sum(gasUsed × base_fee)；矿工奖励 = sum(gasUsed × priority)
//
//   5. 教学指标：
//      · congestionRate     : gasUsed / gasLimit
//      · rejectedRate       : rejected / arrived
//      · acceptedRate
//      · avgWaitDuration    : 入 mempool 至打包的等待 tick
//      · totalBurned / totalMinerRevenue
//      · base_fee 历史曲线 / 拥堵历史 / 拒绝率历史

package gasmarket

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
)

const (
	sceneCode     = "gas-market"
	schemaVersion = "v1.0.0"
	algorithmType = "gas-market-eip1559"

	defaultGasTarget = 15_000_000
	defaultGasLimit  = 30_000_000
	defaultBaseFee   = 10.0 // gwei

	linkGroupTxProc  = "tx-processing-group"
	linkOwnerSubtree = "economic.gas_market"
)

// =====================================================================
// 数据结构
// =====================================================================

type userClass struct {
	Name        string
	ArrivalRate float64 // tx/tick 期望
	WtpMean     float64 // 愿付 gasPrice 均值（gwei）
	WtpStd      float64 // 标准差
	GasLimit    int64   // 单 tx 的 gas 用量
}

type tx struct {
	ID          int
	Class       string
	Wtp         float64
	GasUsed     int64
	ArriveTick  int
	IncludeTick int
	Priority    float64 // wtp - baseFee
	Status      string  // mempool / included / rejected
	Wait        int
}

type epochSnapshot struct {
	Block             int
	BaseFee           float64
	GasUsed           int64
	GasTarget         int64
	GasLimit          int64
	CongestionRate    float64
	ArrivedThisBlock  int
	IncludedThisBlock int
	RejectedThisBlock int
	BurnThisBlock     float64
	MinerRevThisBlock float64
	MempoolSize       int
}

type marketEvent struct {
	Tick int
	Kind string
	Note string
}

type snapState struct {
	Tick        int
	Block       int
	BaseFee     float64
	GasTarget   int64
	GasLimit    int64
	Seed        string
	UserClasses map[string]*userClass

	NextTxID  int
	Mempool   []tx
	History   []tx
	Snapshots []epochSnapshot
	Events    []marketEvent

	TotalArrived  int
	TotalIncluded int
	TotalRejected int
	TotalBurned   float64
	TotalMinerRev float64
	LastError     string
}

func defaultSnapState() snapState {
	st := snapState{
		BaseFee:   defaultBaseFee,
		GasTarget: defaultGasTarget, GasLimit: defaultGasLimit,
		Seed: "lenschain-gas",
		UserClasses: map[string]*userClass{
			"retail": {Name: "retail", ArrivalRate: 30, WtpMean: 15, WtpStd: 5, GasLimit: 21000},
			"defi":   {Name: "defi", ArrivalRate: 20, WtpMean: 30, WtpStd: 10, GasLimit: 200000},
			"whale":  {Name: "whale", ArrivalRate: 5, WtpMean: 100, WtpStd: 30, GasLimit: 500000},
			"bot":    {Name: "bot", ArrivalRate: 10, WtpMean: 50, WtpStd: 15, GasLimit: 100000},
		},
	}
	return st
}

// rng 确定性伪随机 [0, 1)。
func (st snapState) rng(salt string, tick int, seq int) float64 {
	buf := []byte(st.Seed)
	buf = append(buf, []byte(salt)...)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(tick))
	buf = append(buf, b[:]...)
	binary.BigEndian.PutUint64(b[:], uint64(seq))
	buf = append(buf, b[:]...)
	d := keccak256hash.Sum256(buf)
	v := binary.BigEndian.Uint64(d[:8])
	return float64(v%1_000_000_000) / 1_000_000_000.0
}

// gaussian Box-Muller 法生成 N(0,1) 高斯。
func (st snapState) gaussian(salt string, tick int, seq int) float64 {
	u1 := math.Max(1e-9, st.rng(salt+"-u1", tick, seq))
	u2 := st.rng(salt+"-u2", tick, seq)
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// =====================================================================
// 核心：到达 / 决策 / 打包 / base_fee 调整
// =====================================================================

// generateArrivals 在当前 tick 生成新 tx：每 class 按泊松-like。
// 教学版用"期望 N + uniform 抖动"近似（不依赖全局随机状态，使用 rand.New 确定性 RNG）。
func (st *snapState) generateArrivals() (arrived, rejected int) {
	classes := []string{}
	for k := range st.UserClasses {
		classes = append(classes, k)
	}
	sort.Strings(classes)
	seq := 0
	for _, name := range classes {
		c := st.UserClasses[name]
		// poisson approx：用 rng 的 [-0.5, 0.5] 抖动
		jitter := (st.rng("arrival-"+name, st.Tick, seq) - 0.5) * 0.4
		count := int(math.Round(c.ArrivalRate * (1 + jitter)))
		seq++
		for i := 0; i < count; i++ {
			arrived++
			st.NextTxID++
			// 生成 wtp ~ N(WtpMean, WtpStd)，clamp ≥ 0
			z := st.gaussian("wtp-"+name, st.Tick, st.NextTxID)
			wtp := c.WtpMean + z*c.WtpStd
			if wtp < 0 {
				wtp = 0
			}
			t := tx{
				ID: st.NextTxID, Class: name,
				Wtp: wtp, GasUsed: c.GasLimit,
				ArriveTick: st.Tick,
				Priority:   wtp - st.BaseFee,
			}
			// 用户决策：effectiveGasPrice = base_fee + priority；若 wtp < base_fee → 拒绝（priority < 0）
			if wtp < st.BaseFee {
				t.Status = "rejected"
				rejected++
				st.History = append(st.History, t)
				continue
			}
			t.Status = "mempool"
			st.Mempool = append(st.Mempool, t)
		}
	}
	st.TotalArrived += arrived
	st.TotalRejected += rejected
	if rejected > 0 {
		st.recordEvent("arrival",
			fmt.Sprintf("到达=%d 拒绝=%d (base_fee=%.2f)", arrived, rejected, st.BaseFee))
	}
	return
}

// mineBlock 把 mempool 按 priority 降序打包到 gasLimit。
func (st *snapState) mineBlock() epochSnapshot {
	st.Tick++
	st.Block++
	arrived, rejected := st.generateArrivals()
	// 按 priority 降序排序
	sort.SliceStable(st.Mempool, func(a, b int) bool {
		return st.Mempool[a].Priority > st.Mempool[b].Priority
	})
	included := []tx{}
	gasUsed := int64(0)
	burnSum := 0.0
	minerSum := 0.0
	remain := []tx{}
	for _, t := range st.Mempool {
		if gasUsed+t.GasUsed <= st.GasLimit {
			t.IncludeTick = st.Tick
			t.Wait = t.IncludeTick - t.ArriveTick
			t.Status = "included"
			gasUsed += t.GasUsed
			burnSum += float64(t.GasUsed) * st.BaseFee
			// priority 实际取走的部分 = min(priority, wtp - baseFee)（教学）
			priorityActual := t.Priority
			if priorityActual < 0 {
				priorityActual = 0
			}
			minerSum += float64(t.GasUsed) * priorityActual
			included = append(included, t)
		} else {
			// 不能装下 → 留在 mempool
			remain = append(remain, t)
		}
	}
	st.Mempool = remain
	st.History = append(st.History, included...)
	if len(st.History) > 256 {
		st.History = st.History[len(st.History)-256:]
	}
	st.TotalIncluded += len(included)
	st.TotalBurned += burnSum
	st.TotalMinerRev += minerSum

	// EIP-1559 base_fee 调整
	delta := float64(gasUsed-st.GasTarget) / float64(st.GasTarget)
	st.BaseFee = st.BaseFee * (1 + delta/8)
	if st.BaseFee < 0.001 {
		st.BaseFee = 0.001
	}

	congestion := float64(gasUsed) / float64(st.GasLimit)
	snap := epochSnapshot{
		Block: st.Block, BaseFee: st.BaseFee,
		GasUsed: gasUsed, GasTarget: st.GasTarget, GasLimit: st.GasLimit,
		CongestionRate:   congestion,
		ArrivedThisBlock: arrived, IncludedThisBlock: len(included),
		RejectedThisBlock: rejected,
		BurnThisBlock:     burnSum, MinerRevThisBlock: minerSum,
		MempoolSize: len(st.Mempool),
	}
	st.Snapshots = append(st.Snapshots, snap)
	if len(st.Snapshots) > 512 {
		st.Snapshots = st.Snapshots[len(st.Snapshots)-512:]
	}
	st.recordEvent("mine_block",
		fmt.Sprintf("block=%d gasUsed=%d (%.2f%%) included=%d burn=%.2f minerRev=%.2f next_base_fee=%.4f",
			st.Block, gasUsed, congestion*100, len(included), burnSum, minerSum, st.BaseFee))
	return snap
}

func (st *snapState) recordEvent(kind, note string) {
	st.Events = append(st.Events, marketEvent{Tick: st.Tick, Kind: kind, Note: note})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

// addUserClass 教学：动态修改用户群体（注入或调整）。
func (st *snapState) addUserClass(c userClass) error {
	if c.Name == "" {
		return errors.New("name 不能空")
	}
	st.UserClasses[c.Name] = &c
	st.recordEvent("user_class",
		fmt.Sprintf("class %s arrival=%.2f wtp=%.2f±%.2f gas=%d",
			c.Name, c.ArrivalRate, c.WtpMean, c.WtpStd, c.GasLimit))
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
		Tick: fw.MapInt(d, "tick", 0), Block: fw.MapInt(d, "block", 0),
		BaseFee:       floatOr(d, "base_fee", defaultBaseFee),
		GasTarget:     int64(fw.MapInt(d, "gas_target", defaultGasTarget)),
		GasLimit:      int64(fw.MapInt(d, "gas_limit", defaultGasLimit)),
		Seed:          fw.MapStr(d, "seed", "lenschain-gas"),
		UserClasses:   map[string]*userClass{},
		NextTxID:      fw.MapInt(d, "next_tx", 0),
		TotalArrived:  fw.MapInt(d, "arrived", 0),
		TotalIncluded: fw.MapInt(d, "included", 0),
		TotalRejected: fw.MapInt(d, "rejected", 0),
		TotalBurned:   floatOr(d, "burned", 0),
		TotalMinerRev: floatOr(d, "miner_rev", 0),
		LastError:     fw.MapStr(d, "last_error", ""),
	}
	if cAny, ok := d["classes"].(map[string]any); ok {
		for name, x := range cAny {
			if m, ok := x.(map[string]any); ok {
				st.UserClasses[name] = &userClass{
					Name:        name,
					ArrivalRate: floatOr(m, "rate", 0),
					WtpMean:     floatOr(m, "mean", 0),
					WtpStd:      floatOr(m, "std", 0),
					GasLimit:    int64(fw.MapInt(m, "gas", 0)),
				}
			}
		}
	}
	if len(st.UserClasses) == 0 {
		return defaultSnapState()
	}
	if mpAny, ok := d["mempool"].([]any); ok {
		for _, x := range mpAny {
			if m, ok := x.(map[string]any); ok {
				st.Mempool = append(st.Mempool, decodeTx(m))
			}
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, x := range hAny {
			if m, ok := x.(map[string]any); ok {
				st.History = append(st.History, decodeTx(m))
			}
		}
	}
	if sAny, ok := d["snaps"].([]any); ok {
		for _, x := range sAny {
			if m, ok := x.(map[string]any); ok {
				st.Snapshots = append(st.Snapshots, epochSnapshot{
					Block:             fw.MapInt(m, "block", 0),
					BaseFee:           floatOr(m, "base_fee", 0),
					GasUsed:           int64(fw.MapInt(m, "gas_used", 0)),
					GasTarget:         int64(fw.MapInt(m, "gas_target", 0)),
					GasLimit:          int64(fw.MapInt(m, "gas_limit", 0)),
					CongestionRate:    floatOr(m, "congestion", 0),
					ArrivedThisBlock:  fw.MapInt(m, "arr", 0),
					IncludedThisBlock: fw.MapInt(m, "inc", 0),
					RejectedThisBlock: fw.MapInt(m, "rej", 0),
					BurnThisBlock:     floatOr(m, "burn", 0),
					MinerRevThisBlock: floatOr(m, "rev", 0),
					MempoolSize:       fw.MapInt(m, "mp", 0),
				})
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, marketEvent{
					Tick: fw.MapInt(m, "tick", 0),
					Kind: fw.MapStr(m, "kind", ""),
					Note: fw.MapStr(m, "note", ""),
				})
			}
		}
	}
	return st
}

func decodeTx(m map[string]any) tx {
	return tx{
		ID: fw.MapInt(m, "id", 0), Class: fw.MapStr(m, "class", ""),
		Wtp: floatOr(m, "wtp", 0), GasUsed: int64(fw.MapInt(m, "gas", 0)),
		ArriveTick: fw.MapInt(m, "arr", 0), IncludeTick: fw.MapInt(m, "inc", 0),
		Priority: floatOr(m, "prio", 0),
		Status:   fw.MapStr(m, "status", ""),
		Wait:     fw.MapInt(m, "wait", 0),
	}
}

func encodeTx(t tx) map[string]any {
	return map[string]any{
		"id": t.ID, "class": t.Class,
		"wtp": t.Wtp, "gas": int(t.GasUsed),
		"arr": t.ArriveTick, "inc": t.IncludeTick,
		"prio":   t.Priority,
		"status": t.Status, "wait": t.Wait,
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["block"] = st.Block
	s.Data["base_fee"] = st.BaseFee
	s.Data["gas_target"] = int(st.GasTarget)
	s.Data["gas_limit"] = int(st.GasLimit)
	s.Data["seed"] = st.Seed
	s.Data["next_tx"] = st.NextTxID
	s.Data["arrived"] = st.TotalArrived
	s.Data["included"] = st.TotalIncluded
	s.Data["rejected"] = st.TotalRejected
	s.Data["burned"] = st.TotalBurned
	s.Data["miner_rev"] = st.TotalMinerRev
	s.Data["last_error"] = st.LastError
	cAny := map[string]any{}
	for name, c := range st.UserClasses {
		cAny[name] = map[string]any{
			"rate": c.ArrivalRate, "mean": c.WtpMean,
			"std": c.WtpStd, "gas": int(c.GasLimit),
		}
	}
	s.Data["classes"] = cAny
	mpAny := make([]any, len(st.Mempool))
	for i, t := range st.Mempool {
		mpAny[i] = encodeTx(t)
	}
	s.Data["mempool"] = mpAny
	hAny := make([]any, len(st.History))
	for i, t := range st.History {
		hAny[i] = encodeTx(t)
	}
	s.Data["history"] = hAny
	sAny := make([]any, len(st.Snapshots))
	for i, sn := range st.Snapshots {
		sAny[i] = map[string]any{
			"block": sn.Block, "base_fee": sn.BaseFee,
			"gas_used": int(sn.GasUsed), "gas_target": int(sn.GasTarget),
			"gas_limit":  int(sn.GasLimit),
			"congestion": sn.CongestionRate,
			"arr":        sn.ArrivedThisBlock, "inc": sn.IncludedThisBlock,
			"rej":  sn.RejectedThisBlock,
			"burn": sn.BurnThisBlock, "rev": sn.MinerRevThisBlock,
			"mp": sn.MempoolSize,
		}
	}
	s.Data["snaps"] = sAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{
			"tick": ev.Tick, "kind": ev.Kind, "note": ev.Note,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "Gas 市场（宏观）",
		Description:         "演示用户群体 wtp 分布 + EIP-1559 长期均衡 + 拒绝率 + 拥堵 + 矿工收入构成",
		Category:            fw.CategoryEconomic,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupTxProc},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"economic.gas_market.base_fee",
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
				ActionCode: "set_block_params", Label: "区块参数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "gas_target", Type: fw.FieldNumber, Label: "GasTarget", Required: true, Default: defaultGasTarget, Min: 1000, Step: 1000000},
					{Name: "gas_limit", Type: fw.FieldNumber, Label: "GasLimit", Required: true, Default: defaultGasLimit, Min: 1000, Step: 1000000},
					{Name: "base_fee", Type: fw.FieldNumber, Label: "BaseFee init (gwei)", Required: true, Default: defaultBaseFee, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "set_user_class", Label: "新增/修改用户群体",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "name", Type: fw.FieldString, Label: "类别名称", Required: true, Default: "newbie"},
					{Name: "rate", Type: fw.FieldNumber, Label: "ArrivalRate", Required: true, Default: 5, Min: 0, Step: 1},
					{Name: "mean", Type: fw.FieldNumber, Label: "WtpMean (gwei)", Required: true, Default: 12, Min: 0, Step: 1},
					{Name: "std", Type: fw.FieldNumber, Label: "WtpStd", Required: true, Default: 4, Min: 0, Step: 1},
					{Name: "gas", Type: fw.FieldNumber, Label: "GasLimit/tx", Required: true, Default: 21000, Min: 1, Step: 1000},
				},
			},
			{
				ActionCode: "mine_block", Label: "出 1 块",
				Description: "生成到达 → 排序 → 打包 → 调整 base_fee",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"economic.gas_market.base_fee"},
				LinkOwnerFields:   []string{"economic.gas_market.base_fee"},
			},
			{
				ActionCode: "mine_n", Label: "出 N 块",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "块数", Required: true, Default: 50, Min: 1, Step: 5},
				},
				WritesOwnedFields: []string{"economic.gas_market.base_fee"},
			},
			{
				ActionCode: "demand_shock", Label: "需求冲击（×N）",
				Description: "把所有 user class 的 ArrivalRate × multiplier",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "multiplier", Type: fw.FieldNumber, Label: "倍数", Required: true, Default: 3, Min: 0.1, Max: 100, Step: 0.5},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
	env := buildEnvelope(st, "init", "Gas 市场初始化（4 个用户群体）", true)
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
	case "set_block_params":
		st.GasTarget = int64(fw.MapInt(in.Params, "gas_target", defaultGasTarget))
		st.GasLimit = int64(fw.MapInt(in.Params, "gas_limit", defaultGasLimit))
		st.BaseFee = floatOr(in.Params, "base_fee", defaultBaseFee)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_block_params", "区块参数已更新", false)
		return out, nil

	case "set_user_class":
		c := userClass{
			Name:        fw.MapStr(in.Params, "name", "newbie"),
			ArrivalRate: floatOr(in.Params, "rate", 5),
			WtpMean:     floatOr(in.Params, "mean", 12),
			WtpStd:      floatOr(in.Params, "std", 4),
			GasLimit:    int64(fw.MapInt(in.Params, "gas", 21000)),
		}
		if err := st.addUserClass(c); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_user_class",
			fmt.Sprintf("user class %s 已配置", c.Name), false)
		return out, nil

	case "mine_block":
		snap := st.mineBlock()
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_block",
			fmt.Sprintf("block=%d gasUsed=%d (%.2f%%) base_fee→%.4f", snap.Block, snap.GasUsed, snap.CongestionRate*100, st.BaseFee), false)
		appendMineMicroSteps(&out.Render, snap)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_n":
		n := fw.MapInt(in.Params, "n", 50)
		var snap epochSnapshot
		for i := 0; i < n; i++ {
			snap = st.mineBlock()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_n",
			fmt.Sprintf("出 %d 块；block=%d base_fee→%.4f burn=%.2f miner=%.2f reject=%d",
				n, snap.Block, st.BaseFee, st.TotalBurned, st.TotalMinerRev, st.TotalRejected), false)
		appendMineMicroSteps(&out.Render, snap)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "demand_shock":
		mult := floatOr(in.Params, "multiplier", 3)
		for _, c := range st.UserClasses {
			c.ArrivalRate *= mult
		}
		st.recordEvent("demand_shock", fmt.Sprintf("ArrivalRate × %.2f", mult))
		saveState(state, st)
		out.Render = buildEnvelope(st, "demand_shock",
			fmt.Sprintf("需求冲击 ×%.2f", mult), false)
		appendShockMicroSteps(&out.Render, mult)
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

	// 1) 用户群体节点（环形）
	cNames := []string{}
	for k := range st.UserClasses {
		cNames = append(cNames, k)
	}
	sort.Strings(cNames)
	prims = append(prims, fw.PrimRingLayout("user-ring", len(cNames)+1))
	for _, n := range cNames {
		c := st.UserClasses[n]
		role := "user-class"
		if c.WtpMean > 50 {
			role = "user-class-whale"
		}
		label := fmt.Sprintf("%s\nrate=%.1f\nwtp=%.1f±%.1f\ngas=%d", n, c.ArrivalRate, c.WtpMean, c.WtpStd, c.GasLimit)
		prims = append(prims, fw.PrimNode("c-"+n, label, "active", role))
	}
	prims = append(prims, fw.PrimNode("market", fmt.Sprintf("Gas Market\nbase_fee=%.4f\ngasTarget=%d\ngasLimit=%d",
		st.BaseFee, st.GasTarget, st.GasLimit), "active", "market"))
	for _, n := range cNames {
		prims = append(prims, fw.PrimEdge("ce-"+n, "c-"+n, "market", "solid", "flow"))
	}

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-decision",
		`\text{user accepts iff } \text{wtp} \ge \text{base\_fee};\quad \text{priority} = \text{wtp} - \text{base\_fee}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-basefee",
		`\text{base\_fee}_{n+1} = \text{base\_fee}_n \cdot \left(1 + \dfrac{1}{8} \cdot \dfrac{\text{gasUsed} - \text{target}}{\text{target}}\right)`, false))
	prims = append(prims, fw.PrimMathFormula("formula-revenue",
		`\text{burn} = \sum \text{gasUsed} \cdot \text{base\_fee};\quad \text{minerRev} = \sum \text{gasUsed} \cdot \text{priority}`, false))

	// 3) 状态参数
	rejectRate := 0.0
	if st.TotalArrived > 0 {
		rejectRate = float64(st.TotalRejected) / float64(st.TotalArrived)
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("block = %d  tick = %d\nbase_fee = %.6f gwei\ngasTarget = %d  gasLimit = %d\nmempool = %d\nuser classes = %d\nTotalArrived = %d  Included = %d  Rejected = %d (%.2f%%)\nTotalBurned = %.4f  TotalMinerRev = %.4f",
			st.Block, st.Tick, st.BaseFee, st.GasTarget, st.GasLimit,
			len(st.Mempool), len(st.UserClasses),
			st.TotalArrived, st.TotalIncluded, st.TotalRejected, rejectRate*100,
			st.TotalBurned, st.TotalMinerRev),
		"text", nil, 10))

	// 4) User class 表
	if len(cNames) > 0 {
		uLines := []string{"name      arrivalRate  wtpMean   wtpStd   gas"}
		for _, n := range cNames {
			c := st.UserClasses[n]
			uLines = append(uLines, fmt.Sprintf("  %-9s %-11.2f  %-7.2f  %-7.2f  %d",
				n, c.ArrivalRate, c.WtpMean, c.WtpStd, c.GasLimit))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-classes", strings.Join(uLines, "\n"), "text", nil, 8))
	}

	// 5) 最近 swap / mempool 表
	if len(st.Mempool) > 0 {
		mLines := []string{"id      class    wtp       priority   gas       wait"}
		startIdx := 0
		if len(st.Mempool) > 12 {
			startIdx = len(st.Mempool) - 12
		}
		mp := append([]tx{}, st.Mempool...)
		sort.SliceStable(mp, func(a, b int) bool { return mp[a].Priority > mp[b].Priority })
		if len(mp) > 12 {
			mp = mp[:12]
		}
		for _, t := range mp {
			mLines = append(mLines, fmt.Sprintf("  %-6d  %-7s  %-8.4f  %-9.4f  %-8d  %d",
				t.ID, t.Class, t.Wtp, t.Priority, t.GasUsed, st.Tick-t.ArriveTick))
		}
		_ = startIdx
		prims = append(prims, fw.PrimCodeBlock("cb-mempool", strings.Join(mLines, "\n"), "text", nil, 14))
	}

	// 6) 曲线：base_fee / 拥堵率 / 拒绝率 / burn 收入
	if len(st.Snapshots) > 0 {
		bfPts := []map[string]float64{}
		ctPts := []map[string]float64{}
		rjPts := []map[string]float64{}
		burnPts := []map[string]float64{}
		revPts := []map[string]float64{}
		for _, sn := range st.Snapshots {
			x := float64(sn.Block)
			bfPts = append(bfPts, map[string]float64{"x": x, "y": sn.BaseFee})
			ctPts = append(ctPts, map[string]float64{"x": x, "y": sn.CongestionRate})
			rj := 0.0
			if sn.ArrivedThisBlock > 0 {
				rj = float64(sn.RejectedThisBlock) / float64(sn.ArrivedThisBlock)
			}
			rjPts = append(rjPts, map[string]float64{"x": x, "y": rj})
			burnPts = append(burnPts, map[string]float64{"x": x, "y": sn.BurnThisBlock})
			revPts = append(revPts, map[string]float64{"x": x, "y": sn.MinerRevThisBlock})
		}
		prims = append(prims, fw.PrimCurve("curve-bf", "base_fee 演化", bfPts, "solid"))
		prims = append(prims, fw.PrimCurve("curve-congestion", "拥堵率 (gasUsed/limit)", ctPts, "dashed"))
		prims = append(prims, fw.PrimCurve("curve-reject", "block 拒绝率", rjPts, "dotted"))
		prims = append(prims, fw.PrimCurve("curve-burn", "block burn 收入", burnPts, "solid"))
		prims = append(prims, fw.PrimCurve("curve-rev", "block miner 收入", revPts, "dashed"))
	}

	// 7) 进度条
	if len(st.Snapshots) > 0 {
		last := st.Snapshots[len(st.Snapshots)-1]
		prims = append(prims, fw.PrimProgressBar("bar-congestion",
			float64(last.GasUsed), float64(last.GasLimit),
			fmt.Sprintf("Last block congestion %.2f%%", last.CongestionRate*100)))
	}
	prims = append(prims, fw.PrimBar("bar-reject", rejectRate*100, 100, "warning",
		fmt.Sprintf("Total reject rate %.2f%%", rejectRate*100)))
	prims = append(prims, fw.PrimBar("bar-burn", st.TotalBurned, 0, "danger", fmt.Sprintf("Total burned %.2f", st.TotalBurned)))
	prims = append(prims, fw.PrimBar("bar-miner", st.TotalMinerRev, 0, "success", fmt.Sprintf("Total miner rev %.2f", st.TotalMinerRev)))

	// 8) 收入构成饼图
	prims = append(prims, fw.PrimPieChart("rev-pie", []map[string]any{
		{"label": "Burned (协议)", "value": st.TotalBurned, "color_role": "danger"},
		{"label": "Miner reward", "value": st.TotalMinerRev, "color_role": "success"},
	}))

	// 9) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"tick   kind            note"}
		startIdx := 0
		if len(st.Events) > 14 {
			startIdx = len(st.Events) - 14
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-5d  %-13s  %s", ev.Tick, ev.Kind, ev.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 16))
	}

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-market", "market", "info", 0.7))
	if len(st.Snapshots) > 0 {
		last := st.Snapshots[len(st.Snapshots)-1]
		if last.CongestionRate > 0.95 {
			prims = append(prims, fw.PrimShake("shake-cong", "bar-congestion", 0.5, 800))
			prims = append(prims, fw.PrimPulse("pulse-cong", "bar-congestion", "danger", 1500))
		}
	}

	// 11) 联动
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "GasMarket 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	rate := 0.0
	if st.TotalArrived > 0 {
		rate = float64(st.TotalRejected) / float64(st.TotalArrived)
	}
	d := map[string]any{
		"block":          st.Block,
		"base_fee":       st.BaseFee,
		"gas_target":     st.GasTarget,
		"gas_limit":      st.GasLimit,
		"mempool_size":   len(st.Mempool),
		"total_arrived":  st.TotalArrived,
		"total_included": st.TotalIncluded,
		"total_rejected": st.TotalRejected,
		"reject_rate":    rate,
		"total_burned":   st.TotalBurned,
		"total_miner":    st.TotalMinerRev,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendMineMicroSteps(env *fw.RenderEnvelope, snap epochSnapshot) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: fmt.Sprintf("生成到达 = %d 笔；用户决策 (%d 拒绝)", snap.ArrivedThisBlock, snap.RejectedThisBlock), DurationMs: 400, HighlightIDs: []string{"user-ring", "formula-decision"}},
		{ID: "m-2", Label: "按 priority 降序打包至 GasLimit", DurationMs: 500, HighlightIDs: []string{"market", "cb-mempool"}, FirePrimitives: []string{"glow-market"}},
		{ID: "m-3", Label: fmt.Sprintf("调整 base_fee（congestion %.2f%%）", snap.CongestionRate*100), DurationMs: 500, HighlightIDs: []string{"formula-basefee", "curve-bf"}},
		{ID: "m-4", Label: "结算 burn / minerRev", DurationMs: 400, HighlightIDs: []string{"formula-revenue", "rev-pie"}, IsLinkTrigger: true},
	}
}

func appendShockMicroSteps(env *fw.RenderEnvelope, mult float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sh-1", Label: fmt.Sprintf("ArrivalRate × %.2f", mult), DurationMs: 400, HighlightIDs: []string{"user-ring", "cb-classes"}},
		{ID: "sh-2", Label: "下次出块时拥堵率应飙升 → base_fee 上行", DurationMs: 500, HighlightIDs: []string{"curve-congestion", "curve-bf"}, IsLinkTrigger: true},
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
		ID:             "gas-market-update",
		SourceScene:    sceneCode,
		SourceAction:   "produce_block",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"economic.gas_market.base_fee"},
		Payload:        map[string]any{"base_fee": st.BaseFee, "block": st.Block},
		SourceAnchorID: "gas-market-anchor",
		TargetAnchorID: "tx-proc-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	rate := 0.0
	if st.TotalArrived > 0 {
		rate = float64(st.TotalRejected) / float64(st.TotalArrived)
	}
	return map[string]any{
		"economic": map[string]any{
			"gas_market": map[string]any{
				"block":          st.Block,
				"base_fee":       st.BaseFee,
				"mempool_size":   len(st.Mempool),
				"total_arrived":  st.TotalArrived,
				"total_included": st.TotalIncluded,
				"total_rejected": st.TotalRejected,
				"reject_rate":    rate,
				"total_burned":   st.TotalBurned,
				"total_miner":    st.TotalMinerRev,
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
