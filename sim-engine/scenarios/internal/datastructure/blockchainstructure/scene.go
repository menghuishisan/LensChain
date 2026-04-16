package blockchainstructure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造区块链结构与分叉场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "blockchain-structure",
		Title:        "区块链结构与分叉",
		Phase:        "创世块",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   16,
		Stages:       []string{"创世块", "追加区块", "产生分叉", "最长链选择"},
		ChangedKeys:  []string{"nodes", "data", "metrics"},
		Data:         map[string]any{},
		Extra:        map[string]any{},
	}
}

// Init 初始化包含创世块和两个后续区块的基础链。
func Init(state *framework.SceneState, input framework.InitInput) error {
	blocks := []blockState{
		newBlock("block-0", 0, "", []string{"genesis"}, "main"),
	}
	blocks = append(blocks, newBlock("block-1", 1, blocks[0].Hash, []string{"tx-a"}, "main"))
	blocks = append(blocks, newBlock("block-2", 2, blocks[1].Hash, []string{"tx-b"}, "main"))
	blocks = applySharedChainState(blocks, input.SharedState, state.LinkGroup)
	return rebuildState(state, blocks, "创世块")
}

// Step 推进追加区块、产生分叉和最长链选择流程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	blocks := decodeBlocks(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "创世块"))
	if sharedBlocks, ok := input.SharedState["blocks"].([]any); ok && len(sharedBlocks) > len(blocks) {
		blocks = decodeSharedBlocks(sharedBlocks, blocks)
	}
	blocks = applySharedChainState(blocks, input.SharedState, state.LinkGroup)
	switch phase {
	case "追加区块":
		parent := tipOfBranch(blocks, "main")
		blocks = append(blocks, newBlock(fmt.Sprintf("block-%d", len(blocks)), parent.Height+1, parent.Hash, []string{fmt.Sprintf("tx-%d", len(blocks))}, "main"))
	case "产生分叉":
		parent := blockByHeight(blocks, 1)
		blocks = append(blocks, newBlock(fmt.Sprintf("fork-%d", len(blocks)), parent.Height+1, parent.Hash, []string{"tx-fork"}, "fork"))
	case "最长链选择":
		markLongestBranch(blocks)
	}
	if err := rebuildState(state, blocks, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("链结构进入%s阶段。", phase), "info")
	return framework.StepOutput{
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, blocks),
	}, nil
}

// HandleAction 支持追加区块和制造分叉两类结构操作。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	blocks := decodeBlocks(state)
	phase := "追加区块"
	if input.ActionCode == "fork_chain" {
		phase = "产生分叉"
		parent := blockByHeight(blocks, int(framework.NumberValue(input.Params["height"], 1)))
		blocks = append(blocks, newBlock(fmt.Sprintf("fork-%d", len(blocks)), parent.Height+1, parent.Hash, []string{"manual-fork"}, "fork"))
	} else {
		parent := tipOfBranch(blocks, "main")
		blocks = append(blocks, newBlock(fmt.Sprintf("block-%d", len(blocks)), parent.Height+1, parent.Hash, []string{"manual-tx"}, "main"))
	}
	if err := rebuildState(state, blocks, phase); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "更新链结构", fmt.Sprintf("已执行 %s。", phase), "success")
	return framework.ActionOutput{
		Success:    true,
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, blocks),
	}, nil
}

// BuildRenderState 返回区块链结构、分叉和最长链标记。
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

// SyncSharedState 在共享链状态变化后重建区块链结构场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	blocks := applySharedChainState(decodeBlocks(state), sharedState, state.LinkGroup)
	return rebuildState(state, blocks, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// blockState 保存区块哈希指针、交易列表和分叉归属。
type blockState struct {
	ID           string   `json:"id"`
	Height       int      `json:"height"`
	Hash         string   `json:"hash"`
	PrevHash     string   `json:"prev_hash"`
	Transactions []string `json:"transactions"`
	Branch       string   `json:"branch"`
	IsLongest    bool     `json:"is_longest"`
}

// newBlock 使用父哈希和交易列表生成稳定区块哈希。
func newBlock(id string, height int, prevHash string, transactions []string, branch string) blockState {
	payload := fmt.Sprintf("%s|%d|%s|%s|%s", id, height, prevHash, strings.Join(transactions, ","), branch)
	hash := sha256.Sum256([]byte(payload))
	return blockState{
		ID:           id,
		Height:       height,
		Hash:         hex.EncodeToString(hash[:]),
		PrevHash:     prevHash,
		Transactions: transactions,
		Branch:       branch,
		IsLongest:    branch == "main",
	}
}

// rebuildState 将链结构转成节点、哈希指针消息和可视化指标。
func rebuildState(state *framework.SceneState, blocks []blockState, phase string) error {
	markLongestBranch(blocks)
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes = make([]framework.Node, 0, len(blocks))
	for index, block := range blocks {
		y := 180.0
		if block.Branch == "fork" {
			y = 310
		}
		status := "normal"
		if block.IsLongest {
			status = "success"
		}
		if block.Branch == "fork" && !block.IsLongest {
			status = "fork"
		}
		state.Nodes = append(state.Nodes, framework.Node{
			ID:     block.ID,
			Label:  fmt.Sprintf("Block-%d", block.Height),
			Status: status,
			Role:   "block",
			X:      120 + float64(index)*130,
			Y:      y,
			Load:   float64(len(block.Transactions) * 20),
			Attributes: map[string]any{
				"hash":         block.Hash,
				"prev_hash":    block.PrevHash,
				"height":       block.Height,
				"transactions": block.Transactions,
				"branch":       block.Branch,
				"is_longest":   block.IsLongest,
			},
		})
	}
	state.Messages = buildMessages(blocks, phase)
	state.Metrics = []framework.Metric{
		{Key: "blocks", Label: "区块数", Value: fmt.Sprintf("%d", len(blocks)), Tone: "info"},
		{Key: "height", Label: "最长链高度", Value: fmt.Sprintf("%d", longestHeight(blocks)), Tone: "success"},
		{Key: "forks", Label: "分叉数", Value: fmt.Sprintf("%d", forkCount(blocks)), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "最长链", Value: longestBranch(blocks)},
	}
	state.Data = map[string]any{
		"phase_name":   phase,
		"blocks":       blocks,
		"transactions": transactionIndex(blocks),
		"longest":      longestBranch(blocks),
	}
	state.Extra = map[string]any{
		"description": "该场景实现区块哈希指针、分叉树和最长链选择。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeBlocks 从通用 JSON 状态恢复区块列表。
func decodeBlocks(state *framework.SceneState) []blockState {
	raw, ok := state.Data["blocks"].([]any)
	if !ok {
		if typed, ok := state.Data["blocks"].([]blockState); ok {
			return typed
		}
		blocks := []blockState{newBlock("block-0", 0, "", []string{"genesis"}, "main")}
		return append(blocks, newBlock("block-1", 1, blocks[0].Hash, []string{"tx-a"}, "main"))
	}
	return decodeSharedBlocks(raw, nil)
}

// decodeSharedBlocks 从联动共享状态或内部状态中恢复区块列表。
func decodeSharedBlocks(raw []any, fallback []blockState) []blockState {
	result := make([]blockState, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, blockState{
			ID:           framework.StringValue(entry["id"], ""),
			Height:       int(framework.NumberValue(entry["height"], 0)),
			Hash:         framework.StringValue(entry["hash"], ""),
			PrevHash:     framework.StringValue(entry["prev_hash"], ""),
			Transactions: framework.ToStringSlice(entry["transactions"]),
			Branch:       framework.StringValue(entry["branch"], "main"),
			IsLongest:    framework.BoolValue(entry["is_longest"], false),
		})
	}
	if len(result) == 0 && len(fallback) > 0 {
		return fallback
	}
	return result
}

// buildMessages 生成区块之间的哈希指针消息。
func buildMessages(blocks []blockState, phase string) []framework.Message {
	messages := make([]framework.Message, 0, len(blocks))
	for _, block := range blocks {
		if block.PrevHash == "" {
			continue
		}
		parent := parentBlock(blocks, block.PrevHash)
		messages = append(messages, framework.Message{
			ID:       fmt.Sprintf("%s-pointer", block.ID),
			Label:    "prev_hash",
			Kind:     "pointer",
			Status:   phase,
			SourceID: block.ID,
			TargetID: parent.ID,
		})
	}
	return messages
}

// markLongestBranch 根据分支高度标记最长链。
func markLongestBranch(blocks []blockState) {
	branch := longestBranch(blocks)
	for index := range blocks {
		blocks[index].IsLongest = blocks[index].Branch == branch
	}
}

// tipOfBranch 返回指定分支的最高区块。
func tipOfBranch(blocks []blockState, branch string) blockState {
	tip := blocks[0]
	for _, block := range blocks {
		if block.Branch == branch && block.Height >= tip.Height {
			tip = block
		}
	}
	return tip
}

// blockByHeight 返回主链上指定高度的区块。
func blockByHeight(blocks []blockState, height int) blockState {
	for _, block := range blocks {
		if block.Height == height && block.Branch == "main" {
			return block
		}
	}
	return blocks[0]
}

// parentBlock 根据前序哈希查找父区块。
func parentBlock(blocks []blockState, prevHash string) blockState {
	for _, block := range blocks {
		if block.Hash == prevHash {
			return block
		}
	}
	return blocks[0]
}

// longestBranch 返回当前最高分支名称。
func longestBranch(blocks []blockState) string {
	heights := map[string]int{}
	for _, block := range blocks {
		if block.Height > heights[block.Branch] {
			heights[block.Branch] = block.Height
		}
	}
	result := "main"
	best := -1
	for branch, height := range heights {
		if height > best {
			result = branch
			best = height
		}
	}
	return result
}

// longestHeight 返回当前最长链高度。
func longestHeight(blocks []blockState) int {
	height := 0
	for _, block := range blocks {
		if block.IsLongest && block.Height > height {
			height = block.Height
		}
	}
	return height
}

// forkCount 统计非主链区块数量。
func forkCount(blocks []blockState) int {
	total := 0
	for _, block := range blocks {
		if block.Branch != "main" {
			total++
		}
	}
	return total
}

// branchHeight 返回指定分支的最高高度。
func branchHeight(blocks []blockState, branch string) int {
	height := 0
	for _, block := range blocks {
		if block.Branch == branch && block.Height > height {
			height = block.Height
		}
	}
	return height
}

// transactionIndex 按区块索引交易列表，供联动组读取。
func transactionIndex(blocks []blockState) map[string]any {
	result := map[string]any{}
	for _, block := range blocks {
		result[block.ID] = block.Transactions
	}
	return result
}

// nextPhase 返回链结构演化的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "创世块":
		return "追加区块"
	case "追加区块":
		return "产生分叉"
	case "产生分叉":
		return "最长链选择"
	default:
		return "追加区块"
	}
}

// phaseIndex 将阶段名映射到时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "创世块":
		return 0
	case "追加区块":
		return 1
	case "产生分叉":
		return 2
	case "最长链选择":
		return 3
	default:
		return 0
	}
}

// buildSharedDiff 按联动组输出链结构场景允许共享的状态。
func buildSharedDiff(linkGroup string, blocks []blockState) map[string]any {
	switch linkGroup {
	case "pow-attack-group":
		return map[string]any{
			"nodes": map[string]any{
				"honest_height":   branchHeight(blocks, "main"),
				"attacker_height": branchHeight(blocks, "fork"),
			},
			"blockchain": map[string]any{
				"height":          longestHeight(blocks),
				"fork_detected":   forkCount(blocks) > 0,
				"reorganized":     longestBranch(blocks) == "fork",
				"attacker_height": branchHeight(blocks, "fork"),
				"honest_height":   branchHeight(blocks, "main"),
			},
			"network": map[string]any{
				"fork_detected": forkCount(blocks) > 0,
			},
		}
	case "blockchain-integrity-group":
		return map[string]any{
			"blocks":       blocksToShared(blocks),
			"transactions": transactionIndex(blocks),
			"merkle_root":  blocksMerkleRoot(blocks),
		}
	default:
		return nil
	}
}

// blocksToShared 将内部区块结构转换为共享状态中的数组对象。
func blocksToShared(blocks []blockState) []map[string]any {
	result := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, map[string]any{
			"id":           block.ID,
			"height":       block.Height,
			"hash":         block.Hash,
			"prev_hash":    block.PrevHash,
			"transactions": block.Transactions,
			"branch":       block.Branch,
			"is_longest":   block.IsLongest,
		})
	}
	return result
}

// blocksMerkleRoot 根据最长链顶端区块生成完整性组使用的 Merkle Root 语义。
func blocksMerkleRoot(blocks []blockState) string {
	tip := tipOfBranch(blocks, longestBranch(blocks))
	if tip.Hash == "" {
		return ""
	}
	return tip.Hash
}

// applySharedChainState 将当前联动组的共享状态映射回链结构场景。
func applySharedChainState(blocks []blockState, sharedState map[string]any, linkGroup string) []blockState {
	if len(sharedState) == 0 {
		return blocks
	}
	if linkGroup == "blockchain-integrity-group" {
		if sharedBlocks, ok := sharedState["blocks"].([]any); ok && len(sharedBlocks) > 0 {
			return decodeSharedBlocks(sharedBlocks, blocks)
		}
		return blocks
	}
	if linkGroup != "pow-attack-group" {
		return blocks
	}
	blockchain, ok := sharedState["blockchain"].(map[string]any)
	if !ok {
		return blocks
	}
	honestHeight := int(framework.NumberValue(blockchain["honest_height"], float64(branchHeight(blocks, "main"))))
	attackerHeight := int(framework.NumberValue(blockchain["attacker_height"], float64(branchHeight(blocks, "fork"))))
	if linkedHeight := int(framework.NumberValue(blockchain["height"], float64(honestHeight))); linkedHeight > honestHeight {
		honestHeight = linkedHeight
	}
	reorganized := framework.BoolValue(blockchain["reorganized"], false)
	if honestHeight == 0 && attackerHeight == 0 {
		return blocks
	}
	return rebuildBranchesFromHeights(honestHeight, attackerHeight, reorganized)
}

// rebuildBranchesFromHeights 按共享态中的主链和攻击链高度重建分叉结构。
func rebuildBranchesFromHeights(honestHeight int, attackerHeight int, reorganized bool) []blockState {
	if honestHeight < 0 {
		honestHeight = 0
	}
	if attackerHeight < 0 {
		attackerHeight = 0
	}
	blocks := make([]blockState, 0, honestHeight+attackerHeight+1)
	genesis := newBlock("block-0", 0, "", []string{"genesis"}, "main")
	blocks = append(blocks, genesis)
	parent := genesis
	for height := 1; height <= honestHeight; height++ {
		block := newBlock(fmt.Sprintf("block-%d", height), height, parent.Hash, []string{fmt.Sprintf("tx-%d", height)}, "main")
		blocks = append(blocks, block)
		parent = block
	}
	forkBase := genesis
	if honestHeight >= 1 && len(blocks) > 1 {
		forkBase = blocks[1]
	}
	forkParent := forkBase
	for height := 2; height <= attackerHeight; height++ {
		block := newBlock(fmt.Sprintf("fork-%d", height), height, forkParent.Hash, []string{fmt.Sprintf("fork-tx-%d", height)}, "fork")
		blocks = append(blocks, block)
		forkParent = block
	}
	if reorganized && attackerHeight > honestHeight {
		for index := range blocks {
			blocks[index].IsLongest = blocks[index].Branch == "fork" || blocks[index].Height <= 1
		}
		return blocks
	}
	markLongestBranch(blocks)
	return blocks
}
