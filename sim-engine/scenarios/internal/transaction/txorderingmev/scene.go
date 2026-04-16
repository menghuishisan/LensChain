package txorderingmev

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造交易排序与 MEV 场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "tx-ordering-mev",
		Title:        "交易排序与 MEV",
		Phase:        "交易入池",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   16,
		Stages:       []string{"交易入池", "按费率排序", "夹击插单", "打包确认"},
		Nodes: []framework.Node{
			{ID: "user-tx", Label: "User-Tx", Status: "active", Role: "mempool", X: 100, Y: 200},
			{ID: "bot-tx", Label: "Bot-Tx", Status: "normal", Role: "mempool", X: 280, Y: 110},
			{ID: "miner", Label: "Miner", Status: "normal", Role: "mempool", X: 500, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化交易池、费率和矿工排序。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := mevModel{
		Transactions: []mempoolTx{
			{ID: "user-buy", Sender: "user", FeeRate: 28, Value: 100},
			{ID: "user-sell", Sender: "user", FeeRate: 22, Value: 95},
		},
		MEVProfit: 0,
	}
	applySharedMEVState(&model, input.SharedState)
	sortPool(model.Transactions)
	return rebuildState(state, model, "交易入池")
}

// Step 推进交易入池、费率排序、夹击插单和打包确认。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedMEVState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "交易入池"))
	switch phase {
	case "按费率排序":
		sortPool(model.Transactions)
	case "夹击插单":
		model.Transactions = append(model.Transactions,
			mempoolTx{ID: "bot-front", Sender: "bot", FeeRate: 40, Value: 10},
			mempoolTx{ID: "bot-back", Sender: "bot", FeeRate: 38, Value: 10},
		)
		model.MEVProfit += 12
		sortPool(model.Transactions)
	case "打包确认":
		model.Included = topIDs(model.Transactions, 3)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("MEV 排序进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"mempool": map[string]any{
				"ordered":    txList(model.Transactions),
				"included":   model.Included,
				"mev_profit": model.MEVProfit,
			},
		},
	}, nil
}

// HandleAction 注入新的 MEV 机器人交易。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	botID := framework.StringValue(input.Params["bot"], "bot-1")
	model.Transactions = append(model.Transactions, mempoolTx{
		ID:      botID + "-front",
		Sender:  "bot",
		FeeRate: 45,
		Value:   12,
	})
	sortPool(model.Transactions)
	if err := rebuildState(state, model, "夹击插单"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "注入机器人", fmt.Sprintf("已注入 %s 抢跑交易。", botID), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"mempool": map[string]any{
				"ordered":    txList(model.Transactions),
				"mev_profit": model.MEVProfit,
			},
		},
	}, nil
}

// BuildRenderState 输出排序后的交易池、机器人交易和打包结果。
func BuildRenderState(state framework.SceneState) framework.RenderEnvelope {
	return framework.RenderEnvelope{
		Nodes:       state.Nodes,
		Messages:    state.Messages,
		Stages:      state.Stages,
		ChangedKeys: state.ChangedKeys,
		Phase:       state.Phase,
		PhaseIndex:  state.PhaseIndex,
		Progress:    state.Progress,
		Data:        framework.CloneMap(state.Data),
		Extra:       framework.CloneMap(state.Extra),
	}
}

// SyncSharedState 在交易池排序、Gas 与余额变化后重建 MEV 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedMEVState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// mevModel 保存交易池排序和已打包交易列表。
type mevModel struct {
	Transactions []mempoolTx `json:"transactions"`
	Included     []string    `json:"included"`
	MEVProfit    float64     `json:"mev_profit"`
}

// mempoolTx 保存交易池中的单笔交易。
type mempoolTx struct {
	ID      string  `json:"id"`
	Sender  string  `json:"sender"`
	FeeRate float64 `json:"fee_rate"`
	Value   float64 `json:"value"`
}

// rebuildState 将交易池模型映射为节点、交易流和指标。
func rebuildState(state *framework.SceneState, model mevModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	sortPool(model.Transactions)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[0].Load = float64(len(model.Transactions))
	state.Nodes[1].Load = float64(countBots(model.Transactions))
	state.Nodes[2].Load = float64(len(model.Included))
	state.Nodes[nodeIndexForPhase(state.PhaseIndex)].Status = "active"
	if phase == "打包确认" {
		state.Nodes[2].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "tx-order", Label: topLabel(model.Transactions), Kind: "transaction", Status: phase, SourceID: "user-tx", TargetID: "miner"},
		{ID: "bot-order", Label: topBotLabel(model.Transactions), Kind: "transaction", Status: phase, SourceID: "bot-tx", TargetID: "miner"},
	}
	state.Metrics = []framework.Metric{
		{Key: "pool", Label: "交易池数量", Value: fmt.Sprintf("%d", len(model.Transactions)), Tone: "info"},
		{Key: "top_fee", Label: "最高费率", Value: fmt.Sprintf("%.0f", topFee(model.Transactions)), Tone: "warning"},
		{Key: "bot_count", Label: "机器人交易", Value: fmt.Sprintf("%d", countBots(model.Transactions)), Tone: "warning"},
		{Key: "profit", Label: "MEV 收益", Value: fmt.Sprintf("%.0f", model.MEVProfit), Tone: "warning"},
		{Key: "included", Label: "已打包交易", Value: strings.Join(model.Included, ", "), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "排序结果", Value: strings.Join(topIDs(model.Transactions, len(model.Transactions)), ", ")},
		{Label: "Top Sender", Value: topSender(model.Transactions)},
		{Label: "MEV 收益", Value: fmt.Sprintf("%.0f", model.MEVProfit)},
		{Label: "阶段", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":      phase,
		"tx_ordering_mev": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟交易池按费率排序、MEV 机器人夹击插单和矿工打包确认。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// nodeIndexForPhase 返回当前三节点图中可用的高亮索引。
func nodeIndexForPhase(phaseIndex int) int {
	if phaseIndex > 2 {
		return 2
	}
	if phaseIndex < 0 {
		return 0
	}
	return phaseIndex
}

// decodeModel 从通用 JSON 状态恢复 MEV 交易池模型。
func decodeModel(state *framework.SceneState) mevModel {
	entry, ok := state.Data["tx_ordering_mev"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["tx_ordering_mev"].(mevModel); ok {
			return typed
		}
		return mevModel{
			Transactions: []mempoolTx{
				{ID: "user-buy", Sender: "user", FeeRate: 28, Value: 100},
				{ID: "user-sell", Sender: "user", FeeRate: 22, Value: 95},
			},
		}
	}
	return mevModel{
		Transactions: decodeTxs(entry["transactions"]),
		Included:     framework.ToStringSlice(entry["included"]),
		MEVProfit:    framework.NumberValue(entry["mev_profit"], 0),
	}
}

// applySharedMEVState 将交易处理组的共享交易池、Gas 与余额变化映射回排序模型。
func applySharedMEVState(model *mevModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if mempool, ok := sharedState["mempool"].(map[string]any); ok {
		if ordered := decodeSharedMempoolTxs(mempool["ordered"]); len(ordered) > 0 {
			model.Transactions = ordered
		}
		if included := framework.ToStringSlice(mempool["included"]); len(included) > 0 {
			model.Included = included
		}
	}
	if balances, ok := sharedState["balances"].(map[string]any); ok {
		sender := framework.NumberValue(balances["sender"], 0)
		receiver := framework.NumberValue(balances["receiver"], 0)
		if sender > 0 || receiver > 0 {
			model.Transactions = upsertTransaction(model.Transactions, mempoolTx{
				ID:      "token-transfer",
				Sender:  "user",
				FeeRate: 26,
				Value:   receiver,
			})
		}
	}
	if gasState, ok := sharedState["gas"].(map[string]any); ok {
		used := framework.NumberValue(gasState["used"], 0)
		if used > 15000 {
			model.Transactions = upsertTransaction(model.Transactions, mempoolTx{
				ID:      "gas-sensitive",
				Sender:  "bot",
				FeeRate: 42,
				Value:   used,
			})
		}
	}
	sortPool(model.Transactions)
}

// sortPool 按费率从高到低排序交易池。
func sortPool(values []mempoolTx) {
	sort.Slice(values, func(i int, j int) bool {
		return values[i].FeeRate > values[j].FeeRate
	})
}

// topIDs 返回前 N 个交易标识。
func topIDs(values []mempoolTx, limit int) []string {
	size := limit
	if len(values) < size {
		size = len(values)
	}
	result := make([]string, 0, size)
	for index := 0; index < size; index++ {
		result = append(result, values[index].ID)
	}
	return result
}

// txList 生成共享状态所需的排序结果。
func txList(values []mempoolTx) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, tx := range values {
		result = append(result, map[string]any{
			"id":       tx.ID,
			"sender":   tx.Sender,
			"fee_rate": tx.FeeRate,
		})
	}
	return result
}

// countBots 统计机器人交易数量。
func countBots(values []mempoolTx) int {
	total := 0
	for _, tx := range values {
		if tx.Sender == "bot" {
			total++
		}
	}
	return total
}

// topFee 返回最高费率。
func topFee(values []mempoolTx) float64 {
	if len(values) == 0 {
		return 0
	}
	return values[0].FeeRate
}

// topLabel 返回排序首位交易标识。
func topLabel(values []mempoolTx) string {
	if len(values) == 0 {
		return "empty"
	}
	return values[0].ID
}

// topBotLabel 返回排序最靠前的机器人交易标识。
func topBotLabel(values []mempoolTx) string {
	for _, tx := range values {
		if tx.Sender == "bot" {
			return tx.ID
		}
	}
	return "no-bot"
}

// topSender 返回当前排序首位的发送方。
func topSender(values []mempoolTx) string {
	if len(values) == 0 {
		return "none"
	}
	return values[0].Sender
}

// decodeTxs 恢复交易池切片。
func decodeTxs(value any) []mempoolTx {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]mempoolTx); ok {
			return append([]mempoolTx(nil), typed...)
		}
		return []mempoolTx{
			{ID: "user-buy", Sender: "user", FeeRate: 28, Value: 100},
			{ID: "user-sell", Sender: "user", FeeRate: 22, Value: 95},
		}
	}
	result := make([]mempoolTx, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mempoolTx{
			ID:      framework.StringValue(entry["id"], ""),
			Sender:  framework.StringValue(entry["sender"], ""),
			FeeRate: framework.NumberValue(entry["fee_rate"], 0),
			Value:   framework.NumberValue(entry["value"], 0),
		})
	}
	return result
}

// decodeSharedMempoolTxs 将共享状态中的交易池排序结果恢复为内部交易结构。
func decodeSharedMempoolTxs(value any) []mempoolTx {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]mempoolTx, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mempoolTx{
			ID:      framework.StringValue(entry["id"], ""),
			Sender:  framework.StringValue(entry["sender"], ""),
			FeeRate: framework.NumberValue(entry["fee_rate"], 0),
			Value:   framework.NumberValue(entry["value"], 0),
		})
	}
	return result
}

// upsertTransaction 用共享状态中的交易更新或插入本地交易池。
func upsertTransaction(values []mempoolTx, next mempoolTx) []mempoolTx {
	for index := range values {
		if values[index].ID == next.ID {
			values[index] = next
			return values
		}
	}
	return append(values, next)
}

// nextPhase 返回 MEV 排序流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "交易入池":
		return "按费率排序"
	case "按费率排序":
		return "夹击插单"
	case "夹击插单":
		return "打包确认"
	default:
		return "交易入池"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "交易入池":
		return 0
	case "按费率排序":
		return 1
	case "夹击插单":
		return 1
	case "打包确认":
		return 2
	default:
		return 0
	}
}

// toneByPhase 返回 MEV 阶段色调。
func toneByPhase(phase string) string {
	switch phase {
	case "夹击插单":
		return "warning"
	case "打包确认":
		return "success"
	default:
		return "info"
	}
}

