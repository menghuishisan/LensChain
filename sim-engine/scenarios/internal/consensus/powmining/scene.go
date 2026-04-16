package powmining

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// defaultTarget 是初始难度目标，矿工哈希前 64 bit 小于该值即视为命中。
	defaultTarget uint64 = 0x000fffffffffffff
	// attemptsPerUnit 决定单位算力每个 tick 可尝试的 Nonce 数量。
	attemptsPerUnit int = 64
	// attackerMinerID 约定 PoW 攻击联动组中的攻击者矿工。
	attackerMinerID = "miner-d"
)

// DefaultState 构造 PoW 挖矿竞争场景的基础状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "pow-mining",
		Title:        "PoW 挖矿竞争",
		Phase:        "分配算力",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   24,
		Stages:       []string{"分配算力", "Nonce 搜索", "命中目标", "出块广播"},
		Nodes: []framework.Node{
			{ID: "miner-a", Label: "Miner-A", Status: "active", Role: "miner", X: 120, Y: 180, HashRate: 1.0},
			{ID: "miner-b", Label: "Miner-B", Status: "normal", Role: "miner", X: 280, Y: 120, HashRate: 1.2},
			{ID: "miner-c", Label: "Miner-C", Status: "normal", Role: "miner", X: 440, Y: 180, HashRate: 1.6},
			{ID: "miner-d", Label: "Miner-D", Status: "normal", Role: "miner", X: 600, Y: 120, HashRate: 0.8},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化真实挖矿轮次状态。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	return rebuildState(state, "")
}

// Step 执行一次真实的 Nonce 搜索批次。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	miners := decodeMiners(state)
	miners = applySharedMiningState(miners, input.SharedState)
	target := uint64(framework.NumberValue(state.Data["target"], float64(defaultTarget)))
	if sharedTarget := uint64(framework.NumberValue(input.SharedState["target"], 0)); sharedTarget != 0 {
		target = sharedTarget
	}
	height := int(framework.NumberValue(state.Data["height"], 1))
	header := framework.StringValue(state.Data["header"], buildHeader(height))
	winner := ""
	bestHash := ""
	events := make([]framework.TimelineEvent, 0, 1)
	for index := range miners {
		attempts := int(math.Max(1, miners[index].HashRate*float64(attemptsPerUnit)))
		for attempt := 0; attempt < attempts; attempt++ {
			miners[index].Nonce++
			hashHex, score := mineHash(header, miners[index].ID, miners[index].Nonce)
			miners[index].Attempts++
			miners[index].LastHash = hashHex
			if score < miners[index].BestScore || miners[index].BestScore == 0 {
				miners[index].BestScore = score
			}
			if score < target {
				winner = miners[index].ID
				bestHash = hashHex
				miners[index].Blocks++
				miners[index].Revenue += 6.25
				break
			}
		}
		if winner != "" {
			break
		}
	}
	if winner != "" {
		height++
		header = buildHeader(height)
		target = adjustTarget(target)
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "命中目标", fmt.Sprintf("%s 挖出新区块 %s。", winner, bestHash[:16]), "success"))
		for index := range miners {
			miners[index].Nonce = 0
			miners[index].BestScore = 0
		}
	} else {
		events = append(events, framework.NewEvent(state.SceneCode, state.Tick, "继续搜索 Nonce", "本轮尚未命中目标值，所有矿工继续并行计算。", "info"))
	}
	updateState(state, miners, height, header, target, winner, bestHash)
	attackerRatio, honestRatio := splitHashrateRatio(miners)
	return framework.StepOutput{
		Events: events,
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"attacker_hashrate": attackerRatio,
				"honest_hashrate":   honestRatio,
			},
			"blockchain": map[string]any{
				"height": height,
			},
			"network": map[string]any{
				"winner": winner,
				"target": target,
			},
		},
	}, nil
}

// HandleAction 调整指定矿工算力，驱动真实尝试次数变化。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	miners := decodeMiners(state)
	targetID := framework.StringValue(input.Params["resource_id"], miners[0].ID)
	hashRate := framework.NumberValue(input.Params["hashrate"], 1.0)
	for index := range miners {
		if miners[index].ID == targetID {
			miners[index].HashRate = math.Max(0.1, hashRate)
			break
		}
	}
	updateState(
		state,
		miners,
		int(framework.NumberValue(state.Data["height"], 1)),
		framework.StringValue(state.Data["header"], buildHeader(1)),
		uint64(framework.NumberValue(state.Data["target"], float64(defaultTarget))),
		framework.StringValue(state.Data["winner"], ""),
		framework.StringValue(state.Data["winning_hash"], ""),
	)
	attackerRatio, honestRatio := splitHashrateRatio(miners)
	event := framework.NewEvent(state.SceneCode, state.Tick, "调整算力", fmt.Sprintf("%s 的算力调整为 %.2f。", targetID, hashRate), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"nodes": map[string]any{
				"attacker_hashrate": attackerRatio,
				"honest_hashrate":   honestRatio,
			},
		},
	}, nil
}

// BuildRenderState 返回包含矿工统计、难度目标和赢家信息的渲染载荷。
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

// SyncSharedState 在联动共享状态更新后重建 PoW 挖矿场景的当前渲染态。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	miners := applySharedMiningState(decodeMiners(state), sharedState)
	target := uint64(framework.NumberValue(state.Data["target"], float64(defaultTarget)))
	if network, ok := sharedState["network"].(map[string]any); ok {
		if sharedTarget := uint64(framework.NumberValue(network["target"], 0)); sharedTarget != 0 {
			target = sharedTarget
		}
	}
	updateState(
		state,
		miners,
		int(framework.NumberValue(state.Data["height"], 1)),
		framework.StringValue(state.Data["header"], buildHeader(1)),
		target,
		framework.StringValue(state.Data["winner"], ""),
		framework.StringValue(state.Data["winning_hash"], ""),
	)
	return nil
}

// minerState 保存单个矿工在当前挖矿轮次中的实时统计。
type minerState struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	HashRate  float64 `json:"hashrate"`
	Nonce     uint64  `json:"nonce"`
	Attempts  uint64  `json:"attempts"`
	BestScore uint64  `json:"best_score"`
	LastHash  string  `json:"last_hash"`
	Blocks    uint64  `json:"blocks"`
	Revenue   float64 `json:"revenue"`
}

// rebuildState 用默认矿工集合初始化一轮新的 PoW 竞争状态。
func rebuildState(state *framework.SceneState, winner string) error {
	miners := make([]minerState, 0, len(state.Nodes))
	for _, node := range state.Nodes {
		miners = append(miners, minerState{ID: node.ID, Label: node.Label, HashRate: maxFloat(node.HashRate, 1)})
	}
	updateState(state, miners, 1, buildHeader(1), defaultTarget, winner, "")
	return nil
}

// updateState 将矿工统计、目标值与赢家信息同步回统一渲染状态。
func updateState(state *framework.SceneState, miners []minerState, height int, header string, target uint64, winner string, winningHash string) {
	phaseIndex, phase := powPhase(winner)
	state.PhaseIndex = phaseIndex
	state.Phase = phase
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	leaderID, leaderAttempts := leadingMiner(miners)
	state.Messages = []framework.Message{
		{ID: "pow-search", Label: "并行哈希搜索", Kind: "vote", Status: phase, SourceID: leaderID, TargetID: "network"},
		{ID: "pow-target", Label: "难度目标线", Kind: "vote", Status: phase},
	}
	state.Metrics = []framework.Metric{
		{Key: "height", Label: "区块高度", Value: strconv.Itoa(height), Tone: "info"},
		{Key: "target", Label: "目标阈值", Value: fmt.Sprintf("%016x", target), Tone: "warning"},
		{Key: "winner", Label: "当前赢家", Value: displayWinner(miners, winner), Tone: "success"},
		{Key: "revenue", Label: "攻击者收益", Value: fmt.Sprintf("%.2f", attackerRevenue(miners)), Tone: "warning"},
		{Key: "attempts", Label: "领先矿工尝试数", Value: fmt.Sprintf("%d", leaderAttempts), Tone: "info"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "区块头", Value: header},
		{Label: "目标阈值", Value: fmt.Sprintf("%016x", target)},
		{Label: "赢家", Value: displayWinner(miners, winner)},
	}
	for index := range state.Nodes {
		node := &state.Nodes[index]
		node.Status = "normal"
		for _, miner := range miners {
			if miner.ID != node.ID {
				continue
			}
			node.HashRate = miner.HashRate
			node.Load = float64(miner.Attempts)
			node.Attributes = map[string]any{
				"nonce":      miner.Nonce,
				"attempts":   miner.Attempts,
				"best_score": fmt.Sprintf("%016x", miner.BestScore),
				"last_hash":  miner.LastHash,
				"blocks":     miner.Blocks,
			}
			if node.ID == winner {
				node.Status = "success"
			} else if node.ID == leaderID {
				node.Status = "active"
			}
		}
	}
	state.Data = map[string]any{
		"height":       height,
		"header":       header,
		"target":       target,
		"winner":       winner,
		"winning_hash": winningHash,
		"miners":       miners,
	}
	state.Extra = map[string]any{
		"description": "该场景使用真实 SHA-256 哈希结果比较 target，按矿工算力分配尝试次数。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
}

// decodeMiners 从通用 JSON 状态中恢复矿工统计结构。
func decodeMiners(state *framework.SceneState) []minerState {
	rawMiners, ok := state.Data["miners"].([]any)
	if !ok {
		typed, ok := state.Data["miners"].([]minerState)
		if ok {
			return typed
		}
		miners := make([]minerState, 0, len(state.Nodes))
		for _, node := range state.Nodes {
			miners = append(miners, minerState{ID: node.ID, Label: node.Label, HashRate: maxFloat(node.HashRate, 1)})
		}
		return miners
	}
	result := make([]minerState, 0, len(rawMiners))
	for _, item := range rawMiners {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, minerState{
			ID:        framework.StringValue(entry["id"], ""),
			Label:     framework.StringValue(entry["label"], ""),
			HashRate:  framework.NumberValue(entry["hashrate"], 1),
			Nonce:     uint64(framework.NumberValue(entry["nonce"], 0)),
			Attempts:  uint64(framework.NumberValue(entry["attempts"], 0)),
			BestScore: uint64(framework.NumberValue(entry["best_score"], 0)),
			LastHash:  framework.StringValue(entry["last_hash"], ""),
			Blocks:    uint64(framework.NumberValue(entry["blocks"], 0)),
		})
	}
	if len(result) > 0 {
		return result
	}
	return []minerState{}
}

// buildHeader 生成当前区块高度对应的简化区块头文本。
func buildHeader(height int) string {
	return fmt.Sprintf("height=%d|prev=0000000000000000|merkle=4d65726b6c65", height)
}

// mineHash 对指定矿工和 Nonce 执行一次真实 SHA-256 哈希尝试。
func mineHash(header string, minerID string, nonce uint64) (string, uint64) {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, nonce)
	sum := sha256.Sum256(append(append([]byte(header+"|"+minerID+"|"), buffer...), byte(len(minerID))))
	return hex.EncodeToString(sum[:]), binary.BigEndian.Uint64(sum[:8])
}

// adjustTarget 在成功出块后按简化规则提升难度，避免每轮都过快命中。
func adjustTarget(target uint64) uint64 {
	next := target - target/32
	if next < 0x0000ffffffffffff {
		return 0x0000ffffffffffff
	}
	return next
}

// powPhase 根据是否已出块决定当前渲染阶段。
func powPhase(winner string) (int, string) {
	if winner != "" {
		return 2, "命中目标"
	}
	return 1, "Nonce 搜索"
}

// leadingMiner 返回当前尝试次数最多的矿工，用于突出显示竞争领先者。
func leadingMiner(miners []minerState) (string, uint64) {
	var leader string
	var attempts uint64
	for _, miner := range miners {
		if miner.Attempts >= attempts {
			leader = miner.ID
			attempts = miner.Attempts
		}
	}
	return leader, attempts
}

// displayWinner 将赢家节点标识转换为更适合展示的标签。
func displayWinner(miners []minerState, winner string) string {
	if winner == "" {
		return "暂无"
	}
	for _, miner := range miners {
		if miner.ID == winner {
			return miner.Label
		}
	}
	return winner
}

// maxFloat 保证算力等参数不会退化为无效零值。
func maxFloat(value float64, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

// applySharedMiningState 将 PoW 攻击组中的攻击者算力比例映射回挖矿竞争场景。
func applySharedMiningState(miners []minerState, sharedState map[string]any) []minerState {
	if len(sharedState) == 0 {
		return miners
	}
	nodes, ok := sharedState["nodes"].(map[string]any)
	if !ok {
		return miners
	}
	attackerRatio := framework.NumberValue(nodes["attacker_hashrate"], -1)
	if attackerRatio <= 0 || attackerRatio >= 1 {
		return miners
	}
	return rebalanceMinersByAttackRatio(miners, attackerRatio)
}

// rebalanceMinersByAttackRatio 按攻击者占比重分配矿工算力，保持总算力规模不变。
func rebalanceMinersByAttackRatio(miners []minerState, attackerRatio float64) []minerState {
	if len(miners) == 0 {
		return miners
	}
	total := totalHashrate(miners)
	if total <= 0 {
		total = 4.6
	}
	honestBase := map[string]float64{
		"miner-a": 1.0,
		"miner-b": 1.2,
		"miner-c": 1.6,
	}
	honestBaseTotal := 0.0
	for _, value := range honestBase {
		honestBaseTotal += value
	}
	attackerHashrate := total * attackerRatio
	honestHashrate := total - attackerHashrate
	for index := range miners {
		if miners[index].ID == attackerMinerID {
			miners[index].HashRate = attackerHashrate
			continue
		}
		weight, ok := honestBase[miners[index].ID]
		if !ok || honestBaseTotal == 0 {
			continue
		}
		miners[index].HashRate = honestHashrate * weight / honestBaseTotal
	}
	return miners
}

// splitHashrateRatio 计算攻击者和诚实矿工的算力占比。
func splitHashrateRatio(miners []minerState) (float64, float64) {
	total := totalHashrate(miners)
	if total <= 0 {
		return 0, 0
	}
	attacker := 0.0
	for _, miner := range miners {
		if miner.ID == attackerMinerID {
			attacker = miner.HashRate / total
			break
		}
	}
	return attacker, 1 - attacker
}

// attackerRevenue 返回攻击者矿工当前累计收益。
func attackerRevenue(miners []minerState) float64 {
	for _, miner := range miners {
		if miner.ID == attackerMinerID {
			return miner.Revenue
		}
	}
	return 0
}

// totalHashrate 统计当前所有矿工的总算力。
func totalHashrate(miners []minerState) float64 {
	total := 0.0
	for _, miner := range miners {
		total += miner.HashRate
	}
	return total
}
