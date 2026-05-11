// 模块：sim-engine/scenarios/internal/transaction/txorderingmev
// 文件职责：TX-04 交易排序与 MEV 攻击场景的完整实现。
//
// SSOT 依据：06.md §4.5.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现简化 AMM (x·y=k 恒定乘积) + mempool 按 gas 排序 + MEV 攻击：
//   · 资金池 (X, Y)，初始 X=1000 Y=1000 → k=1_000_000
//   · swap_in_X(dx) → dy = Y - k/(X+dx)，pool.X += dx, pool.Y -= dy
//   · 价格 = Y/X
//   · 用户提交 swap → mempool；矿工按 gas 降序排序打包
//   · MEV 攻击：
//     · Front-run：攻击者用更高 gas 在 user tx 前注入相同方向 swap
//     · Sandwich：在 user tx 前 buy（推高价）+ 后 sell（吃滑点）
//     · 数值化攻击者利润 = sell_received - buy_paid

package txorderingmev

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "tx-ordering-mev"
	schemaVersion = "v1.0.0"
	algorithmType = "amm-mev"

	defaultPoolX = 10000
	defaultPoolY = 10000

	dirBuyY  = "buy_y"  // 用 X 换 Y（dx in, dy out）
	dirSellY = "sell_y" // 用 Y 换 X（dy in, dx out）

	linkGroupTxProc  = "tx-processing-group"
	linkOwnerSubtree = "tx.mev"
)

type swapTx struct {
	ID           string
	Submitter    string
	Direction    string // buy_y / sell_y
	AmountIn     int64
	MinOut       int64 // 最小可接受输出（滑点保护）
	Gas          int
	Tag          string // user / attacker-frontrun / attacker-sandwich-pre / sandwich-post
	Mined        bool
	BlockHeight  int
	ActualOut    int64
	Slippage     float64 // 实际滑点（vs no-front-run）
	FailedReason string
}

type bblock struct {
	Height int
	TxIDs  []string
}

type snapState struct {
	PoolX          int64
	PoolY          int64
	K              int64
	Mempool        []swapTx
	History        []swapTx
	Blocks         []bblock
	BlockHeight    int
	AttackerProfit int64 // 累计 X 单位
	LastError      string
}

func defaultSnapState() snapState {
	return snapState{
		PoolX: defaultPoolX, PoolY: defaultPoolY,
		K: int64(defaultPoolX) * int64(defaultPoolY),
	}
}

// price 当前 Y/X 价格。
func (st snapState) price() float64 {
	if st.PoolX == 0 {
		return 0
	}
	return float64(st.PoolY) / float64(st.PoolX)
}

// applyBuyY 用 dx 单位 X 换出 dy 单位 Y。
func (st *snapState) applyBuyY(dx int64) (int64, error) {
	if dx <= 0 {
		return 0, errors.New("dx 必须 > 0")
	}
	if st.PoolX+dx == 0 {
		return 0, errors.New("pool 异常")
	}
	newX := st.PoolX + dx
	newY := st.K / newX
	dy := st.PoolY - newY
	if dy <= 0 {
		return 0, errors.New("流动性不足")
	}
	st.PoolX = newX
	st.PoolY = newY
	return dy, nil
}

// applySellY 用 dy 单位 Y 换出 dx 单位 X。
func (st *snapState) applySellY(dy int64) (int64, error) {
	if dy <= 0 {
		return 0, errors.New("dy 必须 > 0")
	}
	newY := st.PoolY + dy
	newX := st.K / newY
	dx := st.PoolX - newX
	if dx <= 0 {
		return 0, errors.New("流动性不足")
	}
	st.PoolX = newX
	st.PoolY = newY
	return dx, nil
}

// estimateOut 预估某方向某数量的输出（不修改池）。
func (st snapState) estimateOut(dir string, amountIn int64) int64 {
	switch dir {
	case dirBuyY:
		newX := st.PoolX + amountIn
		if newX == 0 {
			return 0
		}
		return st.PoolY - st.K/newX
	case dirSellY:
		newY := st.PoolY + amountIn
		if newY == 0 {
			return 0
		}
		return st.PoolX - st.K/newY
	}
	return 0
}

// submitTx 提交一笔 swap 进入 mempool。
func (st *snapState) submitTx(submitter, dir string, amountIn, minOut int64, gas int, tag string) (*swapTx, error) {
	if dir != dirBuyY && dir != dirSellY {
		return nil, fmt.Errorf("direction 必须是 %s 或 %s", dirBuyY, dirSellY)
	}
	t := swapTx{
		ID:        fmt.Sprintf("tx%d", len(st.History)+len(st.Mempool)),
		Submitter: submitter, Direction: dir,
		AmountIn: amountIn, MinOut: minOut,
		Gas: gas, Tag: tag,
	}
	st.Mempool = append(st.Mempool, t)
	return &st.Mempool[len(st.Mempool)-1], nil
}

// mineBlock 按 gas 降序排序 mempool，逐笔执行。
// 返回 user tx 在无前置攻击下应得的 noFrontOut（用于 slippage 对比）。
func (st *snapState) mineBlock() (bblock, []swapTx) {
	sort.Slice(st.Mempool, func(a, b int) bool {
		return st.Mempool[a].Gas > st.Mempool[b].Gas
	})
	blk := bblock{Height: st.BlockHeight}
	executed := []swapTx{}

	// 先记录每笔在排序前 mempool 顺序下的"无前置攻击"基线 out（教学：以快照池估算）
	// 这里简化：用 user tx 的"无前置时直接 swap"输出作为 baseline
	baselineOut := map[string]int64{}
	{
		// 复制 pool
		px, py := st.PoolX, st.PoolY
		for _, t := range st.Mempool {
			if t.Tag != "user" {
				continue
			}
			// 用快照池估算
			snap := snapState{PoolX: px, PoolY: py, K: st.K}
			baselineOut[t.ID] = snap.estimateOut(t.Direction, t.AmountIn)
		}
	}

	for i := range st.Mempool {
		t := &st.Mempool[i]
		var out int64
		var err error
		switch t.Direction {
		case dirBuyY:
			out, err = st.applyBuyY(t.AmountIn)
		case dirSellY:
			out, err = st.applySellY(t.AmountIn)
		}
		if err != nil {
			t.FailedReason = err.Error()
			t.Mined = true
			t.BlockHeight = blk.Height
			executed = append(executed, *t)
			blk.TxIDs = append(blk.TxIDs, t.ID)
			continue
		}
		if out < t.MinOut {
			// 滑点保护触发，回滚
			switch t.Direction {
			case dirBuyY:
				st.PoolX -= t.AmountIn
				st.PoolY += out
			case dirSellY:
				st.PoolY -= t.AmountIn
				st.PoolX += out
			}
			t.FailedReason = fmt.Sprintf("slippage：实际 %d < min_out %d", out, t.MinOut)
			t.Mined = true
			t.BlockHeight = blk.Height
			executed = append(executed, *t)
			blk.TxIDs = append(blk.TxIDs, t.ID)
			continue
		}
		t.ActualOut = out
		// 计算用户实际滑点
		if t.Tag == "user" {
			if base := baselineOut[t.ID]; base > 0 {
				t.Slippage = float64(base-out) / float64(base)
			}
		}
		t.Mined = true
		t.BlockHeight = blk.Height
		executed = append(executed, *t)
		blk.TxIDs = append(blk.TxIDs, t.ID)
	}
	st.History = append(st.History, executed...)
	if len(st.History) > 32 {
		st.History = st.History[len(st.History)-32:]
	}
	st.Mempool = nil
	st.Blocks = append(st.Blocks, blk)
	if len(st.Blocks) > 16 {
		st.Blocks = st.Blocks[len(st.Blocks)-16:]
	}
	st.BlockHeight++
	st.calculateAttackerProfit(executed)
	return blk, executed
}

// calculateAttackerProfit 三明治攻击利润 = sandwich-post 输出（X）- sandwich-pre 输入（X）。
func (st *snapState) calculateAttackerProfit(executed []swapTx) {
	pre, post := int64(0), int64(0)
	prePresent, postPresent := false, false
	for _, t := range executed {
		if t.Tag == "attacker-sandwich-pre" && t.ActualOut > 0 {
			// pre 是 buy_y：花了 AmountIn (X) 得到 ActualOut (Y)
			pre = t.AmountIn
			prePresent = true
		}
		if t.Tag == "attacker-sandwich-post" && t.ActualOut > 0 {
			// post 是 sell_y：花了 AmountIn (Y) 得到 ActualOut (X)
			post = t.ActualOut
			postPresent = true
		}
	}
	if prePresent && postPresent {
		st.AttackerProfit += (post - pre)
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
		PoolX:          int64(fw.MapInt(d, "pool_x", defaultPoolX)),
		PoolY:          int64(fw.MapInt(d, "pool_y", defaultPoolY)),
		K:              int64(fw.MapInt(d, "k", defaultPoolX*defaultPoolY)),
		BlockHeight:    fw.MapInt(d, "block_height", 0),
		AttackerProfit: int64(fw.MapInt(d, "attacker_profit", 0)),
		LastError:      fw.MapStr(d, "last_error", ""),
	}
	if mpAny, ok := d["mempool"].([]any); ok {
		for _, tAny := range mpAny {
			if tm, ok := tAny.(map[string]any); ok {
				st.Mempool = append(st.Mempool, decodeSwapTx(tm))
			}
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, tAny := range hAny {
			if tm, ok := tAny.(map[string]any); ok {
				st.History = append(st.History, decodeSwapTx(tm))
			}
		}
	}
	if bsAny, ok := d["blocks"].([]any); ok {
		for _, bAny := range bsAny {
			if bm, ok := bAny.(map[string]any); ok {
				blk := bblock{Height: fw.MapInt(bm, "h", 0)}
				if txAny, ok := bm["txs"].([]any); ok {
					for _, t := range txAny {
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

func decodeSwapTx(tm map[string]any) swapTx {
	sl := 0.0
	if v, ok := tm["slip"].(float64); ok {
		sl = v
	}
	return swapTx{
		ID:           fw.MapStr(tm, "id", ""),
		Submitter:    fw.MapStr(tm, "submitter", ""),
		Direction:    fw.MapStr(tm, "dir", ""),
		AmountIn:     int64(fw.MapInt(tm, "in", 0)),
		MinOut:       int64(fw.MapInt(tm, "min_out", 0)),
		Gas:          fw.MapInt(tm, "gas", 0),
		Tag:          fw.MapStr(tm, "tag", ""),
		Mined:        fw.MapBool(tm, "mined", false),
		BlockHeight:  fw.MapInt(tm, "block", 0),
		ActualOut:    int64(fw.MapInt(tm, "out", 0)),
		Slippage:     sl,
		FailedReason: fw.MapStr(tm, "fail", ""),
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["pool_x"] = int(st.PoolX)
	s.Data["pool_y"] = int(st.PoolY)
	s.Data["k"] = int(st.K)
	s.Data["block_height"] = st.BlockHeight
	s.Data["attacker_profit"] = int(st.AttackerProfit)
	s.Data["last_error"] = st.LastError
	mpAny := make([]any, len(st.Mempool))
	for i, t := range st.Mempool {
		mpAny[i] = encodeSwapTx(t)
	}
	s.Data["mempool"] = mpAny
	hAny := make([]any, len(st.History))
	for i, t := range st.History {
		hAny[i] = encodeSwapTx(t)
	}
	s.Data["history"] = hAny
	bsAny := make([]any, len(st.Blocks))
	for i, b := range st.Blocks {
		txs := make([]any, len(b.TxIDs))
		for j, id := range b.TxIDs {
			txs[j] = id
		}
		bsAny[i] = map[string]any{"h": b.Height, "txs": txs}
	}
	s.Data["blocks"] = bsAny
}

func encodeSwapTx(t swapTx) map[string]any {
	return map[string]any{
		"id": t.ID, "submitter": t.Submitter, "dir": t.Direction,
		"in": int(t.AmountIn), "min_out": int(t.MinOut),
		"gas": t.Gas, "tag": t.Tag, "mined": t.Mined,
		"block": t.BlockHeight, "out": int(t.ActualOut),
		"slip": t.Slippage, "fail": t.FailedReason,
	}
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "交易排序与 MEV",
		Description:         "演示 AMM 滑点 + mempool 按 gas 排序 + Front-running + Sandwich 攻击 + 攻击者利润",
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
			"tx.mev.attacker_profit",
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
				ActionCode: "set_pool", Label: "设置流动性池",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "x", Type: fw.FieldNumber, Label: "X 储备", Required: true, Default: defaultPoolX, Min: 100, Step: 100},
					{Name: "y", Type: fw.FieldNumber, Label: "Y 储备", Required: true, Default: defaultPoolY, Min: 100, Step: 100},
				},
			},
			{
				ActionCode: "submit_user_swap", Label: "用户提交 swap",
				Description:   "正常用户提交 swap，进 mempool 等待打包",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "direction", Type: fw.FieldEnum, Label: "方向", Required: true, Default: dirBuyY,
						Options: []any{dirBuyY, dirSellY}},
					{Name: "amount_in", Type: fw.FieldNumber, Label: "input 数量", Required: true, Default: 100, Min: 1, Step: 1},
					{Name: "min_out", Type: fw.FieldNumber, Label: "min_out（滑点保护）", Required: true, Default: 80, Min: 0, Step: 1},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: 50, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "inject_frontrun", Label: "注入 front-run 攻击",
				Description:   "攻击者用更高 gas 复制 user 同向 swap",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "direction", Type: fw.FieldEnum, Label: "方向", Required: true, Default: dirBuyY,
						Options: []any{dirBuyY, dirSellY}},
					{Name: "amount_in", Type: fw.FieldNumber, Label: "input", Required: true, Default: 100, Min: 1, Step: 1},
					{Name: "gas", Type: fw.FieldNumber, Label: "高 gas", Required: true, Default: 200, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.mev.attacker_profit"},
				LinkOwnerFields:   []string{"tx.mev.attacker_profit"},
			},
			{
				ActionCode: "inject_sandwich", Label: "注入三明治攻击",
				Description:   "攻击者前后包夹 user tx：pre buy + post sell",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "amount_in", Type: fw.FieldNumber, Label: "pre buy 数量", Required: true, Default: 200, Min: 1, Step: 1},
					{Name: "high_gas", Type: fw.FieldNumber, Label: "pre 高 gas", Required: true, Default: 250, Min: 1, Step: 1},
					{Name: "low_gas", Type: fw.FieldNumber, Label: "post 低 gas", Required: true, Default: 30, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.mev.attacker_profit"},
				LinkOwnerFields:   []string{"tx.mev.attacker_profit"},
			},
			{
				ActionCode: "mine_block", Label: "出块（按 gas 排序）",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"tx.mev.attacker_profit"},
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
	env := buildEnvelope(st, "init", "MEV 初始化（pool=10000:10000）", true)
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
	case "set_pool":
		x := int64(fw.MapInt(in.Params, "x", defaultPoolX))
		y := int64(fw.MapInt(in.Params, "y", defaultPoolY))
		st = snapState{PoolX: x, PoolY: y, K: x * y}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_pool", fmt.Sprintf("pool=%d:%d k=%d", x, y, st.K), true)
		return out, nil

	case "submit_user_swap":
		dir := fw.MapStr(in.Params, "direction", dirBuyY)
		amt := int64(fw.MapInt(in.Params, "amount_in", 100))
		minOut := int64(fw.MapInt(in.Params, "min_out", 80))
		gas := fw.MapInt(in.Params, "gas", 50)
		t, err := st.submitTx("user", dir, amt, minOut, gas, "user")
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "submit_user_swap",
			fmt.Sprintf("user 提交 %s amount=%d gas=%d", dir, amt, gas), false)
		appendSubmitMicroSteps(&out.Render, t.ID, "user")
		return out, nil

	case "inject_frontrun":
		dir := fw.MapStr(in.Params, "direction", dirBuyY)
		amt := int64(fw.MapInt(in.Params, "amount_in", 100))
		gas := fw.MapInt(in.Params, "gas", 200)
		t, err := st.submitTx("attacker", dir, amt, 0, gas, "attacker-frontrun")
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "inject_frontrun",
			fmt.Sprintf("attacker front-run %s amount=%d gas=%d", dir, amt, gas), false)
		appendSubmitMicroSteps(&out.Render, t.ID, "attacker-frontrun")
		return out, nil

	case "inject_sandwich":
		amt := int64(fw.MapInt(in.Params, "amount_in", 200))
		highGas := fw.MapInt(in.Params, "high_gas", 250)
		lowGas := fw.MapInt(in.Params, "low_gas", 30)
		// Pre：buy_y 高 gas
		st.submitTx("attacker", dirBuyY, amt, 0, highGas, "attacker-sandwich-pre")
		// Post：sell_y 低 gas（必须放在 user 之后；矿工按 gas 排序时 user>30 即可在 post 之前）
		// 教学简化：post 输入 = 上一笔 pre 估算的产出（保守）
		expectedY := st.estimateOut(dirBuyY, amt)
		st.submitTx("attacker", dirSellY, expectedY, 0, lowGas, "attacker-sandwich-post")
		saveState(state, st)
		out.Render = buildEnvelope(st, "inject_sandwich",
			fmt.Sprintf("attacker sandwich pre+post（pre gas=%d, post gas=%d）", highGas, lowGas), false)
		appendSandwichMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_block":
		blk, executed := st.mineBlock()
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_block",
			fmt.Sprintf("出块 #%d，处理 %d 笔 swap", blk.Height, len(executed)), false)
		appendMineMicroSteps(&out.Render, len(executed))
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

	// 1) Pool 节点
	prims = append(prims, fw.PrimNodeAt("pool",
		fmt.Sprintf("AMM Pool\nX=%d Y=%d\nk=%d\nprice=%.3f", st.PoolX, st.PoolY, st.K, st.price()),
		"active", "amm-pool", 0.5, 0.3, 1.5))

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-amm",
		`x \cdot y = k;\quad dy = y - \tfrac{k}{x + dx};\quad \text{滑点}_{user} = \tfrac{out_{base} - out_{actual}}{out_{base}}`, false))

	// 3) Mempool 节点（按提交顺序展示，矿工出块时按 gas 排序）
	if len(st.Mempool) > 0 {
		mIDs := []string{}
		for _, t := range st.Mempool {
			mIDs = append(mIDs, "mp-"+t.ID)
		}
		prims = append(prims, fw.PrimStack("mempool-stack", mIDs, "horizontal"))
		for _, t := range st.Mempool {
			role := "user-tx"
			status := "normal"
			switch t.Tag {
			case "attacker-frontrun":
				role = "attacker-frontrun"
				status = "warning"
			case "attacker-sandwich-pre":
				role = "attacker-pre"
				status = "warning"
			case "attacker-sandwich-post":
				role = "attacker-post"
				status = "warning"
			case "user":
				status = "active"
			}
			label := fmt.Sprintf("%s\n%s\nin=%d gas=%d", t.ID, t.Direction, t.AmountIn, t.Gas)
			prims = append(prims, fw.PrimNode("mp-"+t.ID, label, status, role))
		}
	}

	// 4) 状态
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("pool X = %d\npool Y = %d\nk = %d\nprice = Y/X = %.6f\nblock_height = %d\nmempool = %d 笔\nattacker_profit (X) = %d",
			st.PoolX, st.PoolY, st.K, st.price(), st.BlockHeight, len(st.Mempool), st.AttackerProfit),
		"text", nil, 8))

	// 5) Mempool 表
	if len(st.Mempool) > 0 {
		mRows := []string{"id    submitter  tag                   dir       in    gas"}
		// 按 gas 降序展示（与矿工排序一致）
		mp := append([]swapTx{}, st.Mempool...)
		sort.Slice(mp, func(a, b int) bool { return mp[a].Gas > mp[b].Gas })
		for _, t := range mp {
			mRows = append(mRows, fmt.Sprintf("%-5s %-9s  %-22s %-9s %-5d %d",
				t.ID, t.Submitter, t.Tag, t.Direction, t.AmountIn, t.Gas))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-mempool", strings.Join(mRows, "\n"), "text", nil, 14))
	}

	// 6) 历史表
	if len(st.History) > 0 {
		hRows := []string{"id    tag                   dir       in    out    slip%   fail"}
		startIdx := 0
		if len(st.History) > 16 {
			startIdx = len(st.History) - 16
		}
		for _, t := range st.History[startIdx:] {
			fail := ""
			if t.FailedReason != "" {
				fail = "FAIL: " + t.FailedReason
			}
			slip := fmt.Sprintf("%.2f", t.Slippage*100)
			if t.Tag != "user" {
				slip = "-"
			}
			hRows = append(hRows, fmt.Sprintf("%-5s %-22s %-9s %-5d %-5d  %-7s %s",
				t.ID, t.Tag, t.Direction, t.AmountIn, t.ActualOut, slip, fail))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(hRows, "\n"), "text", nil, 18))
	}

	// 7) 进度条：攻击者利润
	if st.AttackerProfit != 0 {
		col := "warning"
		if st.AttackerProfit > 0 {
			col = "danger"
		}
		_ = col
		prims = append(prims, fw.PrimBar("bar-profit",
			float64(st.AttackerProfit), 0, "danger", "Attacker Profit"))
	}

	// 8) 价格曲线
	if len(st.History) > 0 {
		// 取每个块结束后的价格
		points := []map[string]float64{{"x": 0, "y": float64(defaultPoolY) / float64(defaultPoolX)}}
		// 用 history 中按块累计模拟（教学简化）
		for i, b := range st.Blocks {
			_ = b
			points = append(points, map[string]float64{
				"x": float64(i + 1),
				"y": st.price(),
			})
		}
		prims = append(prims, fw.PrimCurve("price-curve", "y = Y/X over blocks", points, "solid"))
	}

	// 9) 动效
	prims = append(prims, fw.PrimGlow("glow-pool", "pool", "info", 0.7))
	if st.AttackerProfit > 0 {
		prims = append(prims, fw.PrimShake("shake-mev", "bar-profit", 0.4, 700))
		prims = append(prims, fw.PrimPulse("pulse-mev", "bar-profit", "danger", 1500))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "MEV 错误", st.LastError, "scene", "请检查参数", true))
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
		"pool_x":          st.PoolX,
		"pool_y":          st.PoolY,
		"k":               st.K,
		"price":           st.price(),
		"block_height":    st.BlockHeight,
		"mempool_count":   len(st.Mempool),
		"history_count":   len(st.History),
		"attacker_profit": st.AttackerProfit,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendSubmitMicroSteps(env *fw.RenderEnvelope, txID, tag string) {
	tail := txID + " 进入 mempool"
	switch tag {
	case "attacker-frontrun":
		tail = txID + " 用更高 gas 抢占执行顺序"
	case "attacker-sandwich-pre":
		tail = "Sandwich pre：高 gas 推高价"
	case "attacker-sandwich-post":
		tail = "Sandwich post：低 gas 在 user 之后吃滑点"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "tx 进入 mempool", DurationMs: 400, HighlightIDs: []string{"mp-" + txID, "cb-mempool"}},
		{ID: "s-2", Label: "等待出块（按 gas 排序）", DurationMs: 400, HighlightIDs: []string{"cb-mempool"}},
		{ID: "s-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"formula-amm"}, IsLinkTrigger: true},
	}
}

func appendSandwichMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sw-1", Label: "攻击者提交 pre buy（高 gas，前置）", DurationMs: 400, HighlightIDs: []string{"cb-mempool"}, FirePrimitives: []string{"shake-mev"}},
		{ID: "sw-2", Label: "提交 post sell（低 gas，后置）", DurationMs: 400, HighlightIDs: []string{"cb-mempool"}},
		{ID: "sw-3", Label: "等矿工按 gas 排序：pre → user → post", DurationMs: 500, HighlightIDs: []string{"formula-amm"}, IsLinkTrigger: true},
	}
}

func appendMineMicroSteps(env *fw.RenderEnvelope, n int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "m-1", Label: "mempool 按 gas 降序排序", DurationMs: 400, HighlightIDs: []string{"cb-mempool"}},
		{ID: "m-2", Label: fmt.Sprintf("依次执行 %d 笔 swap，更新 pool", n), DurationMs: 500, HighlightIDs: []string{"pool", "formula-amm"}, FirePrimitives: []string{"glow-pool"}},
		{ID: "m-3", Label: "user 实际滑点 vs baseline；攻击者利润累计", DurationMs: 500, HighlightIDs: []string{"cb-history", "bar-profit"}, FirePrimitives: []string{"pulse-mev"}, IsLinkTrigger: true},
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
		ID:             "mev-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_mev",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"tx.mev.attacker_profit"},
		Payload:        map[string]any{"profit": st.AttackerProfit},
		SourceAnchorID: "mev-anchor",
		TargetAnchorID: "tx-proc-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "tx.mev.attacker_profit")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"tx": map[string]any{
			"mev": map[string]any{
				"pool_x":          int(st.PoolX),
				"pool_y":          int(st.PoolY),
				"price":           st.price(),
				"block_height":    st.BlockHeight,
				"mempool_count":   len(st.Mempool),
				"attacker_profit": int(st.AttackerProfit),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

