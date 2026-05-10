// 模块：sim-engine/scenarios/internal/economic/defiliquidity
// 文件职责：ECO-04 DeFi 流动性挖矿场景的完整实现。
//
// SSOT 依据：06.md §4.8.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Uniswap V2 风格 CFMM + LP token + 无常损失（IL）+ 流动性挖矿奖励。
//
//   1. CFMM 模型（恒定乘积）：
//      · 池：(reserveX, reserveY)，不变量 k = reserveX * reserveY
//      · addLiquidity: 按当前比例存入 (dx, dy)；
//          首次 LP：lpMinted = sqrt(dx * dy)；
//          后续 LP：lpMinted = min(dx/X * S, dy/Y * S)，S = totalLP
//      · removeLiquidity(lp): 取出 (lp/S * X, lp/S * Y)，烧毁 lp。
//      · swap(dxIn, x→y): dy = Y - k/(X+dx)，扣 fee（默认 0.3%）
//
//   2. 无常损失（Impermanent Loss）：
//      · LP 在价格 p₀ 进入；当前价格 p₁
//      · 持币策略价值 = dx * p₁ + dy   （固定数量 dx, dy 计价于 token Y）
//      · LP 持仓价值   = (lp/S) * (X * p₁ + Y)
//      · IL = (LP 价值 - 持币价值) / 持币价值（始终 ≤ 0）
//      · 公式：IL(p) = 2*sqrt(p)/(1+p) - 1，p = p₁/p₀
//
//   3. 流动性挖矿（Liquidity Mining）：
//      · 池配额 RewardRatePerEpoch（每 epoch 给所有 LP 分发）
//      · 用户 reward = (lpHeld / totalLP) * rewardPerEpoch
//      · 每次 advanceEpoch 累计 user.PendingRewards
//
//   4. 教学指标：
//      · 实测 totalReturn = LP 收到的 swap fee + farm reward
//      · annualPercentageYield (APY)
//      · netProfitVsHODL = totalReturn - IL_loss

package defiliquidity

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "defi-liquidity"
	schemaVersion = "v1.0.0"
	algorithmType = "uniswap-v2-cfmm"

	defaultFeeRate        = 0.003 // 0.3%
	defaultRewardPerEpoch = 100.0
	defaultEpochsPerYear  = 365.0
	tokenX                = "X"
	tokenY                = "Y"

	linkGroupPosEcon  = "pos-economy-group"
	linkOwnerSubtree  = "economic.liquidity"
)

// =====================================================================
// 数据结构
// =====================================================================

type lpPosition struct {
	Owner          string
	LPTokens       float64
	EntryPrice     float64 // 进场时价格 P0 = Y/X
	EntryX, EntryY float64 // 投入数量
	PendingRewards float64 // 流动性挖矿累积奖励
	FeeShare       float64 // 累积 swap fee 收入
}

type swapRecord struct {
	Tick       int
	Trader     string
	Direction  string // x_to_y / y_to_x
	AmountIn   float64
	AmountOut  float64
	Fee        float64
	PriceAfter float64
}

type epochSnapshot struct {
	Epoch        int
	ReserveX     float64
	ReserveY     float64
	K            float64
	Price        float64 // Y/X
	TotalLP      float64
	TotalFees    float64
	TotalRewards float64
}

type lpEvent struct {
	Tick   int
	Kind   string
	Note   string
	Amount float64
}

type snapState struct {
	ReserveX       float64
	ReserveY       float64
	TotalLP        float64
	Positions      map[string]*lpPosition // owner → position（教学：每地址 1 个）
	FeeRate        float64
	RewardPerEpoch float64
	EpochsPerYear  float64

	Tick            int
	Epoch           int
	TotalFeesCum    float64
	TotalRewardsCum float64
	Swaps           []swapRecord
	Snapshots       []epochSnapshot
	Events          []lpEvent
	LastError       string
}

func defaultSnapState() snapState {
	st := snapState{
		Positions:      map[string]*lpPosition{},
		FeeRate:        defaultFeeRate,
		RewardPerEpoch: defaultRewardPerEpoch,
		EpochsPerYear:  defaultEpochsPerYear,
	}
	// 初始空池
	return st
}

func (st snapState) k() float64 { return st.ReserveX * st.ReserveY }
func (st snapState) price() float64 {
	if st.ReserveX == 0 {
		return 0
	}
	return st.ReserveY / st.ReserveX
}

// =====================================================================
// 核心：addLiquidity / removeLiquidity / swap
// =====================================================================

// addLiquidity 投入 (dx, dy)。首 LP 任意比例，后续按池比例。
func (st *snapState) addLiquidity(owner string, dx, dy float64) (float64, error) {
	if dx <= 0 || dy <= 0 {
		return 0, errors.New("dx,dy 必须 > 0")
	}
	st.Tick++
	var lpMinted float64
	if st.TotalLP == 0 {
		// 首次：直接 sqrt(dx * dy)
		lpMinted = math.Sqrt(dx * dy)
	} else {
		// 按比例：取较小者，避免比例倾斜
		share1 := dx / st.ReserveX * st.TotalLP
		share2 := dy / st.ReserveY * st.TotalLP
		lpMinted = math.Min(share1, share2)
	}
	st.ReserveX += dx
	st.ReserveY += dy
	st.TotalLP += lpMinted
	pos, ok := st.Positions[owner]
	if !ok {
		pos = &lpPosition{Owner: owner, EntryPrice: st.price(), EntryX: dx, EntryY: dy}
		st.Positions[owner] = pos
	} else {
		// 更新加权进场价
		newLP := pos.LPTokens + lpMinted
		if newLP > 0 {
			pos.EntryPrice = (pos.EntryPrice*pos.LPTokens + st.price()*lpMinted) / newLP
		}
		pos.EntryX += dx
		pos.EntryY += dy
	}
	pos.LPTokens += lpMinted
	st.recordEvent("add_liquidity",
		fmt.Sprintf("%s 投入 dx=%.4f dy=%.4f → 铸造 LP %.4f", owner, dx, dy, lpMinted),
		lpMinted)
	return lpMinted, nil
}

// removeLiquidity 销毁 lp，按当前比例返还 (X, Y)。
func (st *snapState) removeLiquidity(owner string, lpAmount float64) (float64, float64, error) {
	pos, ok := st.Positions[owner]
	if !ok || pos.LPTokens < lpAmount || lpAmount <= 0 {
		return 0, 0, fmt.Errorf("lp 余额不足: %.4f < %.4f", lpAmountOrZero(pos), lpAmount)
	}
	if st.TotalLP <= 0 {
		return 0, 0, errors.New("池为空")
	}
	st.Tick++
	share := lpAmount / st.TotalLP
	dx := st.ReserveX * share
	dy := st.ReserveY * share
	st.ReserveX -= dx
	st.ReserveY -= dy
	st.TotalLP -= lpAmount
	pos.LPTokens -= lpAmount
	if pos.LPTokens <= 0 {
		// 仓位归零
		pos.EntryX = 0
		pos.EntryY = 0
	}
	st.recordEvent("remove_liquidity",
		fmt.Sprintf("%s 取出 LP %.4f → dx=%.4f dy=%.4f", owner, lpAmount, dx, dy), lpAmount)
	return dx, dy, nil
}

func lpAmountOrZero(p *lpPosition) float64 {
	if p == nil {
		return 0
	}
	return p.LPTokens
}

// swap dxIn → dyOut（x_to_y）或反向。
func (st *snapState) swap(trader, direction string, amountIn float64) (swapRecord, error) {
	if amountIn <= 0 {
		return swapRecord{}, errors.New("amountIn 必须 > 0")
	}
	if st.ReserveX <= 0 || st.ReserveY <= 0 {
		return swapRecord{}, errors.New("池为空")
	}
	st.Tick++
	rec := swapRecord{Tick: st.Tick, Trader: trader, Direction: direction, AmountIn: amountIn}
	feeAmount := amountIn * st.FeeRate
	netIn := amountIn - feeAmount
	rec.Fee = feeAmount

	switch direction {
	case "x_to_y":
		newX := st.ReserveX + netIn
		newY := st.k() / newX
		rec.AmountOut = st.ReserveY - newY
		st.ReserveX = newX + feeAmount // fee 留池中（按 V2 风格）
		st.ReserveY = newY
	case "y_to_x":
		newY := st.ReserveY + netIn
		newX := st.k() / newY
		rec.AmountOut = st.ReserveX - newX
		st.ReserveY = newY + feeAmount
		st.ReserveX = newX
	default:
		return rec, fmt.Errorf("未知方向: %s", direction)
	}
	rec.PriceAfter = st.price()
	st.TotalFeesCum += feeAmount
	st.Swaps = append(st.Swaps, rec)
	if len(st.Swaps) > 32 {
		st.Swaps = st.Swaps[len(st.Swaps)-32:]
	}
	// 把 fee 按 LP 持仓比例分配（教学版直接累加到 FeeShare）
	if st.TotalLP > 0 {
		for _, pos := range st.Positions {
			pos.FeeShare += feeAmount * pos.LPTokens / st.TotalLP
		}
	}
	st.recordEvent("swap",
		fmt.Sprintf("%s %s in=%.4f out=%.4f fee=%.4f price=%.6f",
			trader, direction, amountIn, rec.AmountOut, feeAmount, rec.PriceAfter),
		amountIn)
	return rec, nil
}

// advanceEpoch 推进 1 epoch：分发 farming reward。
func (st *snapState) advanceEpoch() epochSnapshot {
	st.Tick++
	st.Epoch++
	if st.TotalLP > 0 {
		for _, pos := range st.Positions {
			r := st.RewardPerEpoch * pos.LPTokens / st.TotalLP
			pos.PendingRewards += r
			st.TotalRewardsCum += r
		}
	}
	st.recordEvent("advance_epoch",
		fmt.Sprintf("epoch=%d distributed=%.4f", st.Epoch, st.RewardPerEpoch), st.RewardPerEpoch)
	return st.captureSnapshot()
}

// withdrawRewards 用户提取 farming 累积奖励。
func (st *snapState) withdrawRewards(owner string) (float64, error) {
	pos, ok := st.Positions[owner]
	if !ok {
		return 0, fmt.Errorf("仓位不存在: %s", owner)
	}
	r := pos.PendingRewards
	pos.PendingRewards = 0
	st.recordEvent("withdraw_rewards",
		fmt.Sprintf("%s 提取 %.4f rewards", owner, r), r)
	return r, nil
}

// =====================================================================
// 无常损失计算
// =====================================================================

// impermanentLoss 计算单个 LP 的无常损失：
// IL(p) = 2*sqrt(p)/(1+p) - 1，p = currentPrice / entryPrice
func (st snapState) impermanentLoss(pos *lpPosition) float64 {
	if pos.EntryPrice <= 0 || pos.LPTokens <= 0 || st.TotalLP <= 0 {
		return 0
	}
	p := st.price() / pos.EntryPrice
	if p <= 0 {
		return 0
	}
	il := 2*math.Sqrt(p)/(1+p) - 1
	return il
}

// hodlValue 持币策略当前价值（以 Y 计价）：x * price + y
func hodlValue(x, y, price float64) float64 {
	return x*price + y
}

// lpValue 当前 LP 仓位价值（以 Y 计价）。
func (st snapState) lpValue(pos *lpPosition) float64 {
	if st.TotalLP <= 0 {
		return 0
	}
	share := pos.LPTokens / st.TotalLP
	return share*st.ReserveX*st.price() + share*st.ReserveY
}

// netReturn 总回报 = (lpValue + pendingRewards + 已收 fee) - 持币基线
func (st snapState) netReturn(pos *lpPosition) float64 {
	hodl := hodlValue(pos.EntryX, pos.EntryY, st.price())
	current := st.lpValue(pos) + pos.PendingRewards + pos.FeeShare
	return current - hodl
}

// estimatedAPY 简化年化估算：基于最近 10 epoch 的平均 reward + fee。
func (st snapState) estimatedAPY() float64 {
	if len(st.Snapshots) < 2 || st.TotalLP <= 0 {
		return 0
	}
	// 用 totalRewards + totalFees 增量估算
	last := st.Snapshots[len(st.Snapshots)-1]
	first := st.Snapshots[0]
	deltaR := last.TotalRewards - first.TotalRewards
	deltaF := last.TotalFees - first.TotalFees
	deltaEpoch := float64(last.Epoch - first.Epoch)
	if deltaEpoch <= 0 {
		return 0
	}
	// LP 的"基础价值"近似 = (X * price + Y) ≈ 2 * Y
	tvl := 2 * last.ReserveY
	if tvl <= 0 {
		return 0
	}
	annualized := (deltaR + deltaF) * st.EpochsPerYear / deltaEpoch
	return annualized / tvl
}

// =====================================================================
// 持久化
// =====================================================================

func (st *snapState) recordEvent(kind, note string, amount float64) {
	st.Events = append(st.Events, lpEvent{Tick: st.Tick, Kind: kind, Note: note, Amount: amount})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

func (st *snapState) captureSnapshot() epochSnapshot {
	snap := epochSnapshot{
		Epoch: st.Epoch, ReserveX: st.ReserveX, ReserveY: st.ReserveY,
		K: st.k(), Price: st.price(),
		TotalLP: st.TotalLP, TotalFees: st.TotalFeesCum, TotalRewards: st.TotalRewardsCum,
	}
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
		Positions:       map[string]*lpPosition{},
		ReserveX:        floatOr(d, "rx", 0),
		ReserveY:        floatOr(d, "ry", 0),
		TotalLP:         floatOr(d, "lp", 0),
		FeeRate:         floatOr(d, "fee", defaultFeeRate),
		RewardPerEpoch:  floatOr(d, "reward", defaultRewardPerEpoch),
		EpochsPerYear:   floatOr(d, "epoch_year", defaultEpochsPerYear),
		Tick:            fw.MapInt(d, "tick", 0),
		Epoch:           fw.MapInt(d, "epoch", 0),
		TotalFeesCum:    floatOr(d, "fees_cum", 0),
		TotalRewardsCum: floatOr(d, "rewards_cum", 0),
		LastError:       fw.MapStr(d, "last_error", ""),
	}
	if pAny, ok := d["positions"].(map[string]any); ok {
		for owner, x := range pAny {
			if m, ok := x.(map[string]any); ok {
				st.Positions[owner] = &lpPosition{
					Owner:      owner,
					LPTokens:   floatOr(m, "lp", 0),
					EntryPrice: floatOr(m, "entry_p", 0),
					EntryX:     floatOr(m, "ex", 0), EntryY: floatOr(m, "ey", 0),
					PendingRewards: floatOr(m, "pending", 0),
					FeeShare:       floatOr(m, "fee_share", 0),
				}
			}
		}
	}
	if sAny, ok := d["swaps"].([]any); ok {
		for _, x := range sAny {
			if m, ok := x.(map[string]any); ok {
				st.Swaps = append(st.Swaps, swapRecord{
					Tick: fw.MapInt(m, "tick", 0), Trader: fw.MapStr(m, "trader", ""),
					Direction: fw.MapStr(m, "dir", ""),
					AmountIn:  floatOr(m, "in", 0), AmountOut: floatOr(m, "out", 0),
					Fee: floatOr(m, "fee", 0), PriceAfter: floatOr(m, "p_after", 0),
				})
			}
		}
	}
	if snAny, ok := d["snaps"].([]any); ok {
		for _, x := range snAny {
			if m, ok := x.(map[string]any); ok {
				st.Snapshots = append(st.Snapshots, epochSnapshot{
					Epoch:    fw.MapInt(m, "epoch", 0),
					ReserveX: floatOr(m, "rx", 0), ReserveY: floatOr(m, "ry", 0),
					K: floatOr(m, "k", 0), Price: floatOr(m, "p", 0),
					TotalLP:      floatOr(m, "lp", 0),
					TotalFees:    floatOr(m, "fees", 0),
					TotalRewards: floatOr(m, "rewards", 0),
				})
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, lpEvent{
					Tick: fw.MapInt(m, "tick", 0), Kind: fw.MapStr(m, "kind", ""),
					Note: fw.MapStr(m, "note", ""), Amount: floatOr(m, "amount", 0),
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
	s.Data["rx"] = st.ReserveX
	s.Data["ry"] = st.ReserveY
	s.Data["lp"] = st.TotalLP
	s.Data["fee"] = st.FeeRate
	s.Data["reward"] = st.RewardPerEpoch
	s.Data["epoch_year"] = st.EpochsPerYear
	s.Data["tick"] = st.Tick
	s.Data["epoch"] = st.Epoch
	s.Data["fees_cum"] = st.TotalFeesCum
	s.Data["rewards_cum"] = st.TotalRewardsCum
	s.Data["last_error"] = st.LastError
	pAny := map[string]any{}
	for owner, p := range st.Positions {
		pAny[owner] = map[string]any{
			"lp": p.LPTokens, "entry_p": p.EntryPrice,
			"ex": p.EntryX, "ey": p.EntryY,
			"pending": p.PendingRewards, "fee_share": p.FeeShare,
		}
	}
	s.Data["positions"] = pAny
	sAny := make([]any, len(st.Swaps))
	for i, r := range st.Swaps {
		sAny[i] = map[string]any{
			"tick": r.Tick, "trader": r.Trader, "dir": r.Direction,
			"in": r.AmountIn, "out": r.AmountOut,
			"fee": r.Fee, "p_after": r.PriceAfter,
		}
	}
	s.Data["swaps"] = sAny
	snAny := make([]any, len(st.Snapshots))
	for i, sn := range st.Snapshots {
		snAny[i] = map[string]any{
			"epoch": sn.Epoch, "rx": sn.ReserveX, "ry": sn.ReserveY,
			"k": sn.K, "p": sn.Price, "lp": sn.TotalLP,
			"fees": sn.TotalFees, "rewards": sn.TotalRewards,
		}
	}
	s.Data["snaps"] = snAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{
			"tick": ev.Tick, "kind": ev.Kind,
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
		Code: sceneCode, Name: "DeFi 流动性挖矿",
		Description:         "演示 Uniswap V2 CFMM + LP token + swap fee + 流动性挖矿 + 无常损失计算",
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
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "fee_rate", Type: fw.FieldNumber, Label: "FeeRate", Required: true, Default: defaultFeeRate, Min: 0, Max: 0.1, Step: 0.001},
					{Name: "reward", Type: fw.FieldNumber, Label: "RewardPerEpoch", Required: true, Default: defaultRewardPerEpoch, Min: 0, Step: 10},
					{Name: "epoch_year", Type: fw.FieldNumber, Label: "EpochsPerYear", Required: true, Default: defaultEpochsPerYear, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "add_liquidity", Label: "添加流动性",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "owner", Type: fw.FieldString, Label: "owner", Required: true, Default: "alice"},
					{Name: "dx", Type: fw.FieldNumber, Label: "ΔX", Required: true, Default: 1000, Min: 0, Step: 100},
					{Name: "dy", Type: fw.FieldNumber, Label: "ΔY", Required: true, Default: 1000, Min: 0, Step: 100},
				},
			},
			{
				ActionCode: "remove_liquidity", Label: "移除流动性",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "owner", Type: fw.FieldString, Label: "owner", Required: true, Default: "alice"},
					{Name: "lp", Type: fw.FieldNumber, Label: "LP 数", Required: true, Default: 100, Min: 0, Step: 10},
				},
			},
			{
				ActionCode: "swap", Label: "swap",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "trader", Type: fw.FieldString, Label: "trader", Required: true, Default: "trader1"},
					{Name: "direction", Type: fw.FieldEnum, Label: "方向", Required: true, Default: "x_to_y",
						Options: []any{"x_to_y", "y_to_x"}},
					{Name: "amount_in", Type: fw.FieldNumber, Label: "amountIn", Required: true, Default: 100, Min: 0, Step: 10},
				},
			},
			{
				ActionCode: "advance_epoch", Label: "推进 epoch（分发 farming）",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode: "advance_n", Label: "推进 N epoch",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "n", Required: true, Default: 30, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "withdraw_rewards", Label: "提取 farming 奖励",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "owner", Type: fw.FieldString, Label: "owner", Required: true, Default: "alice"},
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
	env := buildEnvelope(st, "init", "DeFi 池初始化（空池，需先 add_liquidity）", true)
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
		st.FeeRate = floatOr(in.Params, "fee_rate", defaultFeeRate)
		st.RewardPerEpoch = floatOr(in.Params, "reward", defaultRewardPerEpoch)
		st.EpochsPerYear = floatOr(in.Params, "epoch_year", defaultEpochsPerYear)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", "参数已更新", false)
		return out, nil

	case "add_liquidity":
		owner := fw.MapStr(in.Params, "owner", "alice")
		dx := floatOr(in.Params, "dx", 1000)
		dy := floatOr(in.Params, "dy", 1000)
		lp, err := st.addLiquidity(owner, dx, dy)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "add_liquidity",
			fmt.Sprintf("%s 投入 (%.2f,%.2f) → LP %.4f", owner, dx, dy, lp), false)
		appendAddLiquidityMicroSteps(&out.Render, owner, lp)
		return out, nil

	case "remove_liquidity":
		owner := fw.MapStr(in.Params, "owner", "alice")
		lp := floatOr(in.Params, "lp", 100)
		dx, dy, err := st.removeLiquidity(owner, lp)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "remove_liquidity",
			fmt.Sprintf("%s 取出 LP %.4f → (%.2f, %.2f)", owner, lp, dx, dy), false)
		appendRemoveLiquidityMicroSteps(&out.Render, dx, dy)
		return out, nil

	case "swap":
		trader := fw.MapStr(in.Params, "trader", "trader1")
		dir := fw.MapStr(in.Params, "direction", "x_to_y")
		amt := floatOr(in.Params, "amount_in", 100)
		rec, err := st.swap(trader, dir, amt)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.captureSnapshot()
		saveState(state, st)
		out.Render = buildEnvelope(st, "swap",
			fmt.Sprintf("%s %s in=%.2f out=%.4f fee=%.4f price→%.6f",
				trader, dir, amt, rec.AmountOut, rec.Fee, rec.PriceAfter), false)
		appendSwapMicroSteps(&out.Render, rec)
		return out, nil

	case "advance_epoch":
		snap := st.advanceEpoch()
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_epoch",
			fmt.Sprintf("epoch=%d rewards distributed=%.4f", snap.Epoch, st.RewardPerEpoch), false)
		appendAdvanceMicroSteps(&out.Render)
		return out, nil

	case "advance_n":
		n := fw.MapInt(in.Params, "n", 30)
		for i := 0; i < n; i++ {
			st.advanceEpoch()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_n",
			fmt.Sprintf("推进 %d epoch → epoch=%d totalRewards=%.4f", n, st.Epoch, st.TotalRewardsCum), false)
		appendAdvanceMicroSteps(&out.Render)
		return out, nil

	case "withdraw_rewards":
		owner := fw.MapStr(in.Params, "owner", "alice")
		r, err := st.withdrawRewards(owner)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "withdraw_rewards",
			fmt.Sprintf("%s 提取 farming reward %.4f", owner, r), false)
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

	// 1) Pool 节点（中心）
	prims = append(prims, fw.PrimNodeAt("pool",
		fmt.Sprintf("AMM Pool\nX=%.2f Y=%.2f\nk=%.2f\nprice=Y/X=%.6f",
			st.ReserveX, st.ReserveY, st.k(), st.price()),
		"active", "amm-pool", 0.5, 0.3, 1.6))

	// 2) LP 节点
	owners := []string{}
	for o := range st.Positions {
		owners = append(owners, o)
	}
	sort.Strings(owners)
	for i, o := range owners {
		pos := st.Positions[o]
		x := 0.15 + float64(i)*0.7/float64(maxInt(len(owners)-1, 1))
		il := st.impermanentLoss(pos)
		role := "lp-position"
		if il < -0.01 {
			role = "lp-position-loss"
		}
		label := fmt.Sprintf("%s\nLP=%.2f\nentry P0=%.4f\nIL=%.4f\nfees=%.2f\nfarm=%.2f",
			o, pos.LPTokens, pos.EntryPrice, il, pos.FeeShare, pos.PendingRewards)
		prims = append(prims, fw.PrimNodeAt("lp-"+o, label, "active", role, x, 0.7, 1.2))
		prims = append(prims, fw.PrimEdge("le-"+o, "lp-"+o, "pool", "solid", "flow"))
	}

	// 3) 公式
	prims = append(prims, fw.PrimMathFormula("formula-cfmm",
		`x \cdot y = k;\quad dy = y - \dfrac{k}{x + dx_{net}}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-lp",
		`\text{首次 LP: } L = \sqrt{dx \cdot dy};\quad \text{后续: } L = \min\!\left(\dfrac{dx}{X}S,\ \dfrac{dy}{Y}S\right)`, false))
	prims = append(prims, fw.PrimMathFormula("formula-il",
		`\text{IL}(p) = \dfrac{2\sqrt{p}}{1+p} - 1,\quad p = \dfrac{P_1}{P_0}`, false))

	// 4) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("ReserveX=%.4f ReserveY=%.4f\nk=%.4f price=%.6f\nTotalLP=%.4f  feeRate=%.4f\nRewardPerEpoch=%.2f  epoch=%d\nTotalFeesCum=%.4f  TotalRewardsCum=%.4f\nestimatedAPY=%.4f",
			st.ReserveX, st.ReserveY, st.k(), st.price(),
			st.TotalLP, st.FeeRate,
			st.RewardPerEpoch, st.Epoch,
			st.TotalFeesCum, st.TotalRewardsCum,
			st.estimatedAPY()),
		"text", nil, 10))

	// 5) Position 表 + IL
	if len(owners) > 0 {
		pLines := []string{"owner    LP        entry_X    entry_Y    entryP0    IL          lpValue     hodl       net      fees+farm"}
		for _, o := range owners {
			p := st.Positions[o]
			il := st.impermanentLoss(p)
			lpVal := st.lpValue(p)
			hod := hodlValue(p.EntryX, p.EntryY, st.price())
			net := st.netReturn(p)
			pLines = append(pLines, fmt.Sprintf("  %-7s  %-8.4f  %-8.2f  %-8.2f  %-8.4f  %-10.4f  %-10.4f  %-9.4f  %-7.4f  %.4f",
				o, p.LPTokens, p.EntryX, p.EntryY, p.EntryPrice,
				il, lpVal, hod, net, p.FeeShare+p.PendingRewards))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-positions", strings.Join(pLines, "\n"), "text", nil, 14))
	}

	// 6) Swap 表
	if len(st.Swaps) > 0 {
		sLines := []string{"tick  trader    dir       in       out       fee      price_after"}
		startIdx := 0
		if len(st.Swaps) > 12 {
			startIdx = len(st.Swaps) - 12
		}
		for _, r := range st.Swaps[startIdx:] {
			sLines = append(sLines, fmt.Sprintf("  %-4d  %-9s  %-9s %-7.2f  %-8.4f  %-7.4f  %.6f",
				r.Tick, r.Trader, r.Direction, r.AmountIn, r.AmountOut, r.Fee, r.PriceAfter))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-swaps", strings.Join(sLines, "\n"), "text", nil, 14))
	}

	// 7) 曲线：price / k / TVL / IL
	if len(st.Snapshots) > 0 {
		pPts := []map[string]float64{}
		kPts := []map[string]float64{}
		tvlPts := []map[string]float64{}
		feesPts := []map[string]float64{}
		for _, sn := range st.Snapshots {
			x := float64(sn.Epoch)
			pPts = append(pPts, map[string]float64{"x": x, "y": sn.Price})
			kPts = append(kPts, map[string]float64{"x": x, "y": sn.K})
			tvlPts = append(tvlPts, map[string]float64{"x": x, "y": 2 * sn.ReserveY})
			feesPts = append(feesPts, map[string]float64{"x": x, "y": sn.TotalFees})
		}
		prims = append(prims, fw.PrimCurve("curve-price", "price = Y/X", pPts, "solid"))
		prims = append(prims, fw.PrimCurve("curve-k", "k = X·Y", kPts, "dashed"))
		prims = append(prims, fw.PrimCurve("curve-tvl", "TVL ≈ 2·Y", tvlPts, "dotted"))
		prims = append(prims, fw.PrimCurve("curve-fees", "TotalFees", feesPts, "dotted"))
	}

	// 8) 进度条
	prims = append(prims, fw.PrimBar("bar-fees", st.TotalFeesCum, 0, "success",
		fmt.Sprintf("Total swap fees %.4f", st.TotalFeesCum)))
	prims = append(prims, fw.PrimBar("bar-rewards", st.TotalRewardsCum, 0, "info",
		fmt.Sprintf("Total farming rewards %.4f", st.TotalRewardsCum)))

	// 9) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"tick  kind                  amount       note"}
		startIdx := 0
		if len(st.Events) > 14 {
			startIdx = len(st.Events) - 14
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-4d  %-20s  %-10.4f  %s",
				ev.Tick, ev.Kind, ev.Amount, ev.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 16))
	}

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-pool", "pool", "info", 0.7))
	if len(st.Swaps) > 0 {
		prims = append(prims, fw.PrimPulse("pulse-swap", "pool", "info", 1500))
	}
	for _, o := range owners {
		il := st.impermanentLoss(st.Positions[o])
		if il < -0.05 {
			prims = append(prims, fw.PrimShake("shake-il-"+o, "lp-"+o, 0.5, 800))
			prims = append(prims, fw.PrimPulse("pulse-il-"+o, "lp-"+o, "danger", 1500))
		}
	}

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "DeFi 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"reserve_x":      st.ReserveX,
		"reserve_y":      st.ReserveY,
		"k":              st.k(),
		"price":          st.price(),
		"total_lp":       st.TotalLP,
		"total_fees":     st.TotalFeesCum,
		"total_rewards":  st.TotalRewardsCum,
		"estimated_apy":  st.estimatedAPY(),
		"position_count": len(st.Positions),
		"epoch":          st.Epoch,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendAddLiquidityMicroSteps(env *fw.RenderEnvelope, owner string, lp float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "a-1", Label: fmt.Sprintf("%s 注入 (dx, dy)", owner), DurationMs: 400, HighlightIDs: []string{"lp-" + owner, "pool"}},
		{ID: "a-2", Label: fmt.Sprintf("铸造 LP %.4f", lp), DurationMs: 500, HighlightIDs: []string{"formula-lp", "cb-positions"}, IsLinkTrigger: true},
	}
}

func appendRemoveLiquidityMicroSteps(env *fw.RenderEnvelope, dx, dy float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "r-1", Label: "按比例计算返还", DurationMs: 400, HighlightIDs: []string{"formula-lp"}},
		{ID: "r-2", Label: fmt.Sprintf("返还 (%.2f, %.2f)，烧 LP", dx, dy), DurationMs: 500, HighlightIDs: []string{"pool", "cb-positions"}, IsLinkTrigger: true},
	}
}

func appendSwapMicroSteps(env *fw.RenderEnvelope, rec swapRecord) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "扣 fee 后净 input", DurationMs: 400, HighlightIDs: []string{"formula-cfmm"}},
		{ID: "s-2", Label: fmt.Sprintf("dy = Y - k/(X+dx) = %.4f", rec.AmountOut), DurationMs: 500, HighlightIDs: []string{"pool", "cb-swaps"}, FirePrimitives: []string{"pulse-swap"}},
		{ID: "s-3", Label: "fee 留池中（按 LP 比例分给所有 LP）", DurationMs: 400, HighlightIDs: []string{"cb-positions", "bar-fees"}, IsLinkTrigger: true},
	}
}

func appendAdvanceMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "ae-1", Label: "epoch++", DurationMs: 300, HighlightIDs: []string{"cb-status"}},
		{ID: "ae-2", Label: "按 LP 比例分发 farming reward", DurationMs: 500, HighlightIDs: []string{"cb-positions", "bar-rewards"}, IsLinkTrigger: true},
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
		ID:             "defi-liquidity-update",
		SourceScene:    sceneCode,
		SourceAction:   "swap",
		LinkGroup:      linkGroupPosEcon,
		ChangedFields:  []string{"economic.liquidity.total_liquidity"},
		Payload:        map[string]any{"reserve_x": st.ReserveX, "reserve_y": st.ReserveY},
		SourceAnchorID: "defi-liquidity-anchor",
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
