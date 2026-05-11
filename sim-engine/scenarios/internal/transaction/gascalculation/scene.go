// 模块：sim-engine/scenarios/internal/transaction/gascalculation
// 文件职责：TX-02 Gas 费用计算（EIP-1559）场景的完整实现。
//
// SSOT 依据：06.md §4.5.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以太坊 EIP-1559 费用机制：
//   · 每笔 tx 提交时附带：max_fee_per_gas / max_priority_fee_per_gas / gas_limit
//   · effective_gas_price = min(max_fee, base_fee + priority_fee)
//   · 真实 priority_fee = effective - base_fee（≤ max_priority_fee）
//   · 矿工选 tx：按 effective_priority_fee 降序选 max_block_size 笔
//   · 块出后 base_fee 自适应：
//     · base_fee_next = base_fee * (1 + 1/8 * (gas_used - target) / target)
//   · 总费用 = base_fee × gas_used（销毁）+ priority_fee × gas_used（给矿工）

package gascalculation

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "gas-calculation"
	schemaVersion = "v1.0.0"
	algorithmType = "eip-1559"

	defaultBaseFee     = 100
	defaultGasTarget   = 200
	defaultGasLimit    = 400 // 单块最大 gas
	defaultMaxBlockTxs = 4
	maxTxs             = 32

	linkGroupTxProc  = "tx-processing-group"
	linkGroupGasMkt  = "gas-market"
	linkOwnerSubtree = "tx.gas"
)

type tx struct {
	ID             string
	From           string
	GasLimit       int
	MaxFee         int // 用户愿意付的总最高 gas price
	MaxPriority    int // 给矿工的最高小费
	EffectivePrice int // 实际计算出的 effective_gas_price (mined 后填)
	GasUsed        int // 实际消耗（教学：直接 = GasLimit）
	BaseFeePaid    int // 销毁
	PriorityPaid   int // 给矿工
	Mined          bool
	BlockHeight    int
}

type bblock struct {
	Height       int
	BaseFee      int
	GasUsed      int
	GasTarget    int
	TxIDs        []string
	MinerReward  int
	BaseFeeBurnt int
}

type snapState struct {
	BaseFee     int
	GasTarget   int
	GasLimit    int
	MaxBlockTxs int
	BlockHeight int
	Tick        int
	Txs         []tx
	Blocks      []bblock
	TotalBurnt  int
	TotalMined  int
	LastError   string
}

func defaultSnapState() snapState {
	return snapState{
		BaseFee:     defaultBaseFee,
		GasTarget:   defaultGasTarget,
		GasLimit:    defaultGasLimit,
		MaxBlockTxs: defaultMaxBlockTxs,
	}
}

// effectiveGasPrice 计算单笔 tx 在当前 base_fee 下的 effective_gas_price 和真实 priority_fee。
func effectiveGasPrice(maxFee, maxPriority, baseFee int) (effective int, priority int) {
	wanted := baseFee + maxPriority
	effective = wanted
	if effective > maxFee {
		effective = maxFee
	}
	priority = effective - baseFee
	if priority < 0 {
		priority = 0
	}
	return
}

// canBeIncluded 一笔 tx 是否可以被本块包含（max_fee >= base_fee）。
func canBeIncluded(t tx, baseFee int) bool {
	return t.MaxFee >= baseFee
}

// adjustBaseFee EIP-1559 base_fee 自适应：
//
//	base_fee_next = base_fee * (1 + 1/8 * (gas_used - target) / target)
//	控制变化范围 ≤ ±12.5%
func adjustBaseFee(baseFee, gasUsed, target int) int {
	if target == 0 {
		return baseFee
	}
	delta := baseFee * (gasUsed - target) / target / 8
	next := baseFee + delta
	if next < 1 {
		next = 1
	}
	return next
}

// submitTx 提交一笔新 tx。
func (st *snapState) submitTx(from string, gasLimit, maxFee, maxPriority int) (*tx, error) {
	if len(st.Txs) >= maxTxs {
		return nil, fmt.Errorf("tx 数 ≥ %d", maxTxs)
	}
	if maxPriority > maxFee {
		return nil, fmt.Errorf("max_priority (%d) 不能 > max_fee (%d)", maxPriority, maxFee)
	}
	t := tx{
		ID:   fmt.Sprintf("tx%d", len(st.Txs)),
		From: from, GasLimit: gasLimit,
		MaxFee: maxFee, MaxPriority: maxPriority,
	}
	st.Txs = append(st.Txs, t)
	return &st.Txs[len(st.Txs)-1], nil
}

// mineBlock 出块：从未 mined 的 tx 中按 effective_priority 降序选 max_block_txs 笔（gas_limit 内）。
func (st *snapState) mineBlock() (bblock, error) {
	type cand struct {
		Idx       int
		Effective int
		Priority  int
	}
	cands := []cand{}
	for i, t := range st.Txs {
		if t.Mined {
			continue
		}
		if !canBeIncluded(t, st.BaseFee) {
			continue
		}
		eff, pr := effectiveGasPrice(t.MaxFee, t.MaxPriority, st.BaseFee)
		cands = append(cands, cand{Idx: i, Effective: eff, Priority: pr})
	}
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].Priority != cands[b].Priority {
			return cands[a].Priority > cands[b].Priority
		}
		return cands[a].Effective > cands[b].Effective
	})
	blk := bblock{Height: st.BlockHeight, BaseFee: st.BaseFee, GasTarget: st.GasTarget}
	gasUsed := 0
	for _, c := range cands {
		t := &st.Txs[c.Idx]
		if gasUsed+t.GasLimit > st.GasLimit {
			continue
		}
		if len(blk.TxIDs) >= st.MaxBlockTxs {
			break
		}
		// 包含
		t.Mined = true
		t.BlockHeight = blk.Height
		t.EffectivePrice = c.Effective
		t.GasUsed = t.GasLimit
		t.BaseFeePaid = st.BaseFee * t.GasUsed
		t.PriorityPaid = c.Priority * t.GasUsed
		blk.TxIDs = append(blk.TxIDs, t.ID)
		gasUsed += t.GasUsed
		blk.MinerReward += t.PriorityPaid
		blk.BaseFeeBurnt += t.BaseFeePaid
	}
	blk.GasUsed = gasUsed
	st.Blocks = append(st.Blocks, blk)
	if len(st.Blocks) > 24 {
		st.Blocks = st.Blocks[len(st.Blocks)-24:]
	}
	st.TotalBurnt += blk.BaseFeeBurnt
	st.TotalMined += blk.MinerReward
	st.BlockHeight++
	st.BaseFee = adjustBaseFee(st.BaseFee, gasUsed, st.GasTarget)
	return blk, nil
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
		BaseFee:     fw.MapInt(d, "base_fee", defaultBaseFee),
		GasTarget:   fw.MapInt(d, "gas_target", defaultGasTarget),
		GasLimit:    fw.MapInt(d, "gas_limit", defaultGasLimit),
		MaxBlockTxs: fw.MapInt(d, "max_block_txs", defaultMaxBlockTxs),
		BlockHeight: fw.MapInt(d, "block_height", 0),
		Tick:        fw.MapInt(d, "tick", 0),
		TotalBurnt:  fw.MapInt(d, "total_burnt", 0),
		TotalMined:  fw.MapInt(d, "total_mined", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if txsAny, ok := d["txs"].([]any); ok {
		for _, tAny := range txsAny {
			if tm, ok := tAny.(map[string]any); ok {
				st.Txs = append(st.Txs, tx{
					ID: fw.MapStr(tm, "id", ""), From: fw.MapStr(tm, "from", ""),
					GasLimit: fw.MapInt(tm, "gas_limit", 0),
					MaxFee:   fw.MapInt(tm, "max_fee", 0), MaxPriority: fw.MapInt(tm, "max_pri", 0),
					EffectivePrice: fw.MapInt(tm, "eff", 0),
					GasUsed:        fw.MapInt(tm, "gas_used", 0),
					BaseFeePaid:    fw.MapInt(tm, "base_paid", 0),
					PriorityPaid:   fw.MapInt(tm, "pri_paid", 0),
					Mined:          fw.MapBool(tm, "mined", false),
					BlockHeight:    fw.MapInt(tm, "block", 0),
				})
			}
		}
	}
	if bsAny, ok := d["blocks"].([]any); ok {
		for _, bAny := range bsAny {
			if bm, ok := bAny.(map[string]any); ok {
				blk := bblock{
					Height: fw.MapInt(bm, "h", 0), BaseFee: fw.MapInt(bm, "base", 0),
					GasUsed: fw.MapInt(bm, "used", 0), GasTarget: fw.MapInt(bm, "target", 0),
					MinerReward: fw.MapInt(bm, "reward", 0), BaseFeeBurnt: fw.MapInt(bm, "burnt", 0),
				}
				if txsA, ok := bm["txs"].([]any); ok {
					for _, t := range txsA {
						if s, ok := t.(string); ok {
							blk.TxIDs = append(blk.TxIDs, s)
						}
					}
				}
				st.Blocks = append(st.Blocks, blk)
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["base_fee"] = st.BaseFee
	s.Data["gas_target"] = st.GasTarget
	s.Data["gas_limit"] = st.GasLimit
	s.Data["max_block_txs"] = st.MaxBlockTxs
	s.Data["block_height"] = st.BlockHeight
	s.Data["tick"] = st.Tick
	s.Data["total_burnt"] = st.TotalBurnt
	s.Data["total_mined"] = st.TotalMined
	s.Data["last_error"] = st.LastError
	txsAny := make([]any, len(st.Txs))
	for i, t := range st.Txs {
		txsAny[i] = map[string]any{
			"id": t.ID, "from": t.From,
			"gas_limit": t.GasLimit, "max_fee": t.MaxFee, "max_pri": t.MaxPriority,
			"eff": t.EffectivePrice, "gas_used": t.GasUsed,
			"base_paid": t.BaseFeePaid, "pri_paid": t.PriorityPaid,
			"mined": t.Mined, "block": t.BlockHeight,
		}
	}
	s.Data["txs"] = txsAny
	bsAny := make([]any, len(st.Blocks))
	for i, b := range st.Blocks {
		txs := make([]any, len(b.TxIDs))
		for j, id := range b.TxIDs {
			txs[j] = id
		}
		bsAny[i] = map[string]any{
			"h": b.Height, "base": b.BaseFee, "used": b.GasUsed,
			"target": b.GasTarget, "txs": txs,
			"reward": b.MinerReward, "burnt": b.BaseFeeBurnt,
		}
	}
	s.Data["blocks"] = bsAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "Gas 费用计算（EIP-1559）",
		Description:         "演示 EIP-1559：base_fee 自适应 + priority_fee 拍卖 + base_fee 销毁 + 矿工小费",
		Category:            fw.CategoryTransaction,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupTxProc, linkGroupGasMkt},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"tx.gas.pending_count",
			"tx.gas.base_fee",
			"tx.gas.block_height",
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
				ActionCode: "set_params", Label: "设置费用参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "base_fee", Type: fw.FieldNumber, Label: "初始 base_fee", Required: true, Default: defaultBaseFee, Min: 1, Step: 1},
					{Name: "gas_target", Type: fw.FieldNumber, Label: "目标 gas/块", Required: true, Default: defaultGasTarget, Min: 10, Step: 10},
					{Name: "gas_limit", Type: fw.FieldNumber, Label: "块 gas_limit (最大)", Required: true, Default: defaultGasLimit, Min: 10, Step: 10},
					{Name: "max_block_txs", Type: fw.FieldNumber, Label: "每块 tx 上限", Required: true, Default: defaultMaxBlockTxs, Min: 1, Max: 16, Step: 1},
				},
			},
			{
				ActionCode: "submit_tx", Label: "提交交易（EIP-1559 字段）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "alice"},
					{Name: "gas_limit", Type: fw.FieldNumber, Label: "gas_limit", Required: true, Default: 100, Min: 1, Step: 1},
					{Name: "max_fee", Type: fw.FieldNumber, Label: "max_fee_per_gas", Required: true, Default: 150, Min: 1, Step: 1},
					{Name: "max_priority", Type: fw.FieldNumber, Label: "max_priority_fee_per_gas", Required: true, Default: 30, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"tx.gas.pending_count"},
				LinkOwnerFields:   []string{"tx.gas.pending_count"},
			},
			{
				ActionCode: "mine_block", Label: "出块",
				Description: "选 priority 最高的 tx，更新 base_fee",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"tx.gas.base_fee", "tx.gas.block_height"},
				LinkOwnerFields:   []string{"tx.gas.base_fee", "tx.gas.block_height"},
			},
			{
				ActionCode: "mine_n_blocks", Label: "连续出 N 块",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "块数", Required: true, Default: 5, Min: 1, Max: 50, Step: 1},
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
			{
				ActionCode:    "estimate_gas",
				Label:         "估算 Gas（真实链）",
				Description:   "调 geth eth_estimateGas 估算交易 gas",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{"from":"{{from}}","to":"{{to}}","data":"{{data}}"}],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from address", Required: true, Default: "0x0000000000000000000000000000000000000001"},
					{Name: "to", Type: fw.FieldString, Label: "to address", Required: false, Default: ""},
					{Name: "data", Type: fw.FieldString, Label: "calldata (hex)", Required: false, Default: "0x"},
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
	env := buildEnvelope(st, "init", "Gas 计算初始化（base=100, target=200）", true)
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
	case "set_params":
		st.BaseFee = fw.MapInt(in.Params, "base_fee", defaultBaseFee)
		st.GasTarget = fw.MapInt(in.Params, "gas_target", defaultGasTarget)
		st.GasLimit = fw.MapInt(in.Params, "gas_limit", defaultGasLimit)
		st.MaxBlockTxs = fw.MapInt(in.Params, "max_block_txs", defaultMaxBlockTxs)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("base=%d target=%d limit=%d max_txs=%d",
				st.BaseFee, st.GasTarget, st.GasLimit, st.MaxBlockTxs), false)
		return out, nil

	case "submit_tx":
		from := fw.MapStr(in.Params, "from", "alice")
		gl := fw.MapInt(in.Params, "gas_limit", 100)
		mf := fw.MapInt(in.Params, "max_fee", 150)
		mp := fw.MapInt(in.Params, "max_priority", 30)
		t, err := st.submitTx(from, gl, mf, mp)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "submit_tx",
			fmt.Sprintf("%s 提交 (max_fee=%d, max_priority=%d, gas=%d)", t.ID, mf, mp, gl), false)
		appendSubmitMicroSteps(&out.Render, t.ID)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_block":
		blk, err := st.mineBlock()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_block",
			fmt.Sprintf("出块 #%d: %d 笔, gas=%d/%d, base→%d", blk.Height, len(blk.TxIDs), blk.GasUsed, st.GasTarget, st.BaseFee), false)
		appendMineMicroSteps(&out.Render, blk.GasUsed > st.GasTarget)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_n_blocks":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.mineBlock()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_n_blocks",
			fmt.Sprintf("连续出 %d 块 → base_fee = %d", n, st.BaseFee), false)
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

	// 1) 公式
	prims = append(prims, fw.PrimMathFormula("formula-eff",
		`\text{effective} = \min(\text{max\_fee}, \text{base\_fee} + \text{priority});\quad \text{priority\_paid} = \text{effective} - \text{base\_fee}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-base",
		`\text{base}_{n+1} = \text{base}_n \cdot \left(1 + \tfrac{1}{8}\cdot\tfrac{\text{used} - \text{target}}{\text{target}}\right)`, false))

	// 2) 状态参数
	pending := 0
	for _, t := range st.Txs {
		if !t.Mined {
			pending++
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("base_fee = %d\ngas_target = %d\ngas_limit (块) = %d\nmax_block_txs = %d\nblock_height = %d\npending tx = %d\ntotal mined = %d\ntotal burnt = %d",
			st.BaseFee, st.GasTarget, st.GasLimit, st.MaxBlockTxs,
			st.BlockHeight, pending, st.TotalMined, st.TotalBurnt),
		"text", nil, 8))

	// 3) base_fee 历史 area / curve
	if len(st.Blocks) > 0 {
		points := make([]map[string]float64, 0, len(st.Blocks))
		for i, b := range st.Blocks {
			points = append(points, map[string]float64{
				"x": float64(i),
				"y": float64(b.BaseFee),
			})
		}
		prims = append(prims, fw.PrimCurve("base-fee-curve",
			"y = base_fee per block", points, "solid"))
		// gas_used / target 双轨
		usedPoints := make([]map[string]float64, 0, len(st.Blocks))
		for i, b := range st.Blocks {
			usedPoints = append(usedPoints, map[string]float64{
				"x": float64(i),
				"y": float64(b.GasUsed),
			})
		}
		prims = append(prims, fw.PrimCurve("gas-used-curve",
			"y = gas_used per block", usedPoints, "dashed"))
		prims = append(prims, fw.PrimTargetZone("target-line",
			float64(st.GasTarget), "gas_target", "y"))
	}

	// 4) tx 表
	rows := []string{"id    from   gas    max_fee  max_pri  eff  pri_paid  base_paid  block"}
	startIdx := 0
	if len(st.Txs) > 16 {
		startIdx = len(st.Txs) - 16
	}
	for _, t := range st.Txs[startIdx:] {
		blk := "-"
		if t.Mined {
			blk = fmt.Sprintf("%d", t.BlockHeight)
		}
		rows = append(rows, fmt.Sprintf("%-5s %-6s %-6d %-8d %-8d %-4d %-9d %-10d %s",
			t.ID, t.From, t.GasLimit, t.MaxFee, t.MaxPriority,
			t.EffectivePrice, t.PriorityPaid, t.BaseFeePaid, blk))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-txs", strings.Join(rows, "\n"), "text", nil, 18))

	// 5) 区块表
	if len(st.Blocks) > 0 {
		bRows := []string{"#    base_fee  gas_used/target  reward  burnt   txs"}
		startBI := 0
		if len(st.Blocks) > 12 {
			startBI = len(st.Blocks) - 12
		}
		for _, b := range st.Blocks[startBI:] {
			bRows = append(bRows, fmt.Sprintf("%-4d %-9d %-4d/%-10d %-7d %-7d [%s]",
				b.Height, b.BaseFee, b.GasUsed, b.GasTarget,
				b.MinerReward, b.BaseFeeBurnt, strings.Join(b.TxIDs, ", ")))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-blocks", strings.Join(bRows, "\n"), "text", nil, 14))
	}

	// 6) bar 累计销毁/铸造
	prims = append(prims, fw.PrimBar("bar-burnt", float64(st.TotalBurnt), 0, "danger", "Total Burnt"))
	prims = append(prims, fw.PrimBar("bar-mined", float64(st.TotalMined), 0, "success", "Total Mined Reward"))

	// 7) 进度条：当前块 gas_used 占比
	if len(st.Blocks) > 0 {
		last := st.Blocks[len(st.Blocks)-1]
		prims = append(prims, fw.PrimProgressBar("bar-block-gas",
			float64(last.GasUsed), float64(st.GasLimit),
			fmt.Sprintf("最近块 gas %d/%d", last.GasUsed, st.GasLimit)))
	}

	// 8) 动效
	if len(st.Blocks) > 0 {
		last := st.Blocks[len(st.Blocks)-1]
		col := "info"
		if last.GasUsed > st.GasTarget {
			col = "warning"
		} else if last.GasUsed < st.GasTarget {
			col = "success"
		}
		prims = append(prims, fw.PrimPulse("pulse-base", "cb-status", col, 1500))
	}

	// 9) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-gas", linkGroupGasMkt, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Gas 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "gas.base_fee", Value: st.BaseFee, TargetPrimitive: "cb-gas", TargetParam: "base_fee"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	pending := 0
	for _, t := range st.Txs {
		if !t.Mined {
			pending++
		}
	}
	d := map[string]any{
		"base_fee":      st.BaseFee,
		"gas_target":    st.GasTarget,
		"gas_limit":     st.GasLimit,
		"block_height":  st.BlockHeight,
		"max_block_txs": st.MaxBlockTxs,
		"pending_count": pending,
		"total_tx":      len(st.Txs),
		"total_burnt":   st.TotalBurnt,
		"total_mined":   st.TotalMined,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendSubmitMicroSteps(env *fw.RenderEnvelope, txID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sb-1", Label: txID + " 进入 mempool", DurationMs: 400, HighlightIDs: []string{"cb-txs"}},
		{ID: "sb-2", Label: "矿工按 effective_priority 排序", DurationMs: 400, HighlightIDs: []string{"formula-eff"}},
		{ID: "sb-3", Label: "等待出块 → 实付 = effective × gas_used", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendMineMicroSteps(env *fw.RenderEnvelope, gasUp bool) {
	tail := "gas_used < target → base_fee 下降"
	if gasUp {
		tail = "gas_used > target → base_fee 上升"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "mn-1", Label: "选 priority 最高的 tx", DurationMs: 400, HighlightIDs: []string{"cb-txs", "formula-eff"}},
		{ID: "mn-2", Label: "区块包含 + base_fee 销毁 + 小费给矿工", DurationMs: 500, HighlightIDs: []string{"cb-blocks", "bar-burnt", "bar-mined"}},
		{ID: "mn-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"formula-base", "base-fee-curve"}, FirePrimitives: []string{"pulse-base"}, IsLinkTrigger: true},
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
		ID:             "gas-calc-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_gas",
		LinkGroup:      linkGroupGasMkt,
		ChangedFields:  []string{"tx.gas.base_fee", "tx.gas.block_height"},
		Payload:        map[string]any{"base_fee": st.BaseFee, "height": st.BlockHeight},
		SourceAnchorID: "gas-calc-anchor",
		TargetAnchorID: "gas-market-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "tx.gas.base_fee", "tx.gas.block_height")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	pending := 0
	for _, t := range st.Txs {
		if !t.Mined {
			pending++
		}
	}
	return map[string]any{
		"tx": map[string]any{
			"gas": map[string]any{
				"base_fee":      st.BaseFee,
				"gas_target":    st.GasTarget,
				"block_height":  st.BlockHeight,
				"pending_count": pending,
				"total_burnt":   st.TotalBurnt,
				"total_mined":   st.TotalMined,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

