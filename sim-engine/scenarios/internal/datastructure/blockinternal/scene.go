package blockinternal

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造区块内部结构场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "block-internal",
		Title:        "区块内部结构",
		Phase:        "展开 Header",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   10,
		Stages:       []string{"展开 Header", "展开 Body", "计算 Merkle Root"},
		Nodes: []framework.Node{
			{ID: "header", Label: "Header", Status: "active", Role: "field", X: 120, Y: 200},
			{ID: "body", Label: "Body", Status: "normal", Role: "field", X: 320, Y: 120},
			{ID: "nonce", Label: "Nonce", Status: "normal", Role: "field", X: 320, Y: 280},
			{ID: "merkle-root", Label: "MerkleRoot", Status: "normal", Role: "field", X: 540, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化区块头字段、交易体和 Merkle 根。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := blockModel{
		Version:      1,
		PreviousHash: "000000abc123",
		Timestamp:    "2026-04-14T10:00:00Z",
		Nonce:        1024,
		Transactions: []string{"tx-a", "tx-b", "tx-c", "tx-d"},
	}
	applySharedBlockState(&model, input.SharedState)
	model.MerkleRoot = calculateMerkleRoot(model.Transactions)
	return rebuildState(state, model, "展开 Header")
}

// Step 推进头部展开、交易体展开和 Merkle 根计算。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedBlockState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "展开 Header"))
	if phase == "计算 Merkle Root" {
		model.MerkleRoot = calculateMerkleRoot(model.Transactions)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("区块内部结构进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"blocks":       buildSharedBlocks(model),
			"transactions": buildSharedTransactions(model.Transactions),
			"merkle_root":  model.MerkleRoot,
		},
	}, nil
}

// HandleAction 展开指定字段，并在修改 nonce 后重算头部摘要。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	field := framework.StringValue(input.Params["field"], "header")
	if field == "nonce" {
		model.Nonce++
	}
	if err := rebuildState(state, model, phaseByField(field)); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "展开字段", fmt.Sprintf("已展开字段 %s。", field), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"blocks":       buildSharedBlocks(model),
			"transactions": buildSharedTransactions(model.Transactions),
			"merkle_root":  model.MerkleRoot,
		},
	}, nil
}

// BuildRenderState 输出区块头、区块体和 Merkle 根。
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

// SyncSharedState 在区块完整性联动变化后重建区块内部结构场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedBlockState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// blockModel 保存区块头、交易体和 Merkle 根。
type blockModel struct {
	Version      int      `json:"version"`
	PreviousHash string   `json:"previous_hash"`
	Timestamp    string   `json:"timestamp"`
	Nonce        int      `json:"nonce"`
	Transactions []string `json:"transactions"`
	MerkleRoot   string   `json:"merkle_root"`
}

// rebuildState 将区块模型映射为节点、字段消息和指标。
func rebuildState(state *framework.SceneState, model blockModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	state.Nodes[2].Load = float64(model.Nonce)
	state.Messages = []framework.Message{
		{ID: "header-body", Label: "Header->Body", Kind: "pointer", Status: phase, SourceID: "header", TargetID: "body"},
		{ID: "body-root", Label: framework.Abbreviate(model.MerkleRoot, 12), Kind: "pointer", Status: phase, SourceID: "body", TargetID: "merkle-root"},
	}
	state.Metrics = []framework.Metric{
		{Key: "version", Label: "区块版本", Value: fmt.Sprintf("%d", model.Version), Tone: "info"},
		{Key: "nonce", Label: "Nonce", Value: fmt.Sprintf("%d", model.Nonce), Tone: "warning"},
		{Key: "tx_count", Label: "交易数", Value: fmt.Sprintf("%d", len(model.Transactions)), Tone: "success"},
		{Key: "merkle_root", Label: "Merkle Root", Value: framework.Abbreviate(model.MerkleRoot, 12), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "PreviousHash", Value: model.PreviousHash},
		{Label: "Timestamp", Value: model.Timestamp},
		{Label: "Transactions", Value: strings.Join(model.Transactions, ", ")},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"block_internal": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟区块头字段展开、交易体展开和 Merkle Root 计算。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复区块模型。
func decodeModel(state *framework.SceneState) blockModel {
	entry, ok := state.Data["block_internal"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["block_internal"].(blockModel); ok {
			return typed
		}
		model := blockModel{
			Version:      1,
			PreviousHash: "000000abc123",
			Timestamp:    "2026-04-14T10:00:00Z",
			Nonce:        1024,
			Transactions: []string{"tx-a", "tx-b", "tx-c", "tx-d"},
		}
		model.MerkleRoot = calculateMerkleRoot(model.Transactions)
		return model
	}
	return blockModel{
		Version:      int(framework.NumberValue(entry["version"], 1)),
		PreviousHash: framework.StringValue(entry["previous_hash"], "000000abc123"),
		Timestamp:    framework.StringValue(entry["timestamp"], "2026-04-14T10:00:00Z"),
		Nonce:        int(framework.NumberValue(entry["nonce"], 1024)),
		Transactions: framework.ToStringSliceOr(entry["transactions"], []string{"tx-a", "tx-b", "tx-c", "tx-d"}),
		MerkleRoot:   framework.StringValue(entry["merkle_root"], ""),
	}
}

// calculateMerkleRoot 计算交易列表的简化 Merkle Root。
func calculateMerkleRoot(transactions []string) string {
	if len(transactions) == 0 {
		return ""
	}
	hashes := make([]string, 0, len(transactions))
	for _, tx := range transactions {
		hashes = append(hashes, framework.HashText(tx))
	}
	for len(hashes) > 1 {
		next := make([]string, 0, (len(hashes)+1)/2)
		for index := 0; index < len(hashes); index += 2 {
			right := index + 1
			if right >= len(hashes) {
				right = index
			}
			next = append(next, framework.HashText(hashes[index]+hashes[right]))
		}
		hashes = next
	}
	return hashes[0]
}

// nextPhase 返回区块内部结构的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "展开 Header":
		return "展开 Body"
	case "展开 Body":
		return "计算 Merkle Root"
	default:
		return "展开 Header"
	}
}

// phaseByField 根据字段名选择展示阶段。
func phaseByField(field string) string {
	switch field {
	case "body":
		return "展开 Body"
	case "nonce":
		return "计算 Merkle Root"
	default:
		return "展开 Header"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "展开 Header":
		return 0
	case "展开 Body":
		return 1
	case "计算 Merkle Root":
		return 2
	default:
		return 0
	}
}

// toneByPhase 返回区块内部结构阶段色调。
func toneByPhase(phase string) string {
	if phase == "计算 Merkle Root" {
		return "success"
	}
	return "info"
}

// buildSharedBlocks 输出区块完整性组统一使用的 blocks 数组语义。
func buildSharedBlocks(model blockModel) []map[string]any {
	return []map[string]any{
		{
			"id":           "block-internal",
			"height":       1,
			"hash":         framework.HashText(model.PreviousHash + "|" + model.MerkleRoot),
			"prev_hash":    model.PreviousHash,
			"transactions": append([]string(nil), model.Transactions...),
			"branch":       "main",
			"is_longest":   true,
			"header": map[string]any{
				"version":       model.Version,
				"previous_hash": model.PreviousHash,
				"timestamp":     model.Timestamp,
				"nonce":         model.Nonce,
				"merkle_root":   model.MerkleRoot,
			},
		},
	}
}

// buildSharedTransactions 输出区块完整性组统一使用的 transactions 索引语义。
func buildSharedTransactions(transactions []string) map[string]any {
	result := make(map[string]any, len(transactions))
	for _, txID := range transactions {
		result[txID] = map[string]any{
			"payload": txID,
			"hash":    framework.HashText(txID),
		}
	}
	return result
}

// applySharedBlockState 将区块完整性组中的交易列表与 Merkle Root 映射回区块内部结构。
func applySharedBlockState(model *blockModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if sharedRoot := framework.StringValue(sharedState["merkle_root"], ""); sharedRoot != "" {
		model.MerkleRoot = sharedRoot
	}
	if blocks, ok := sharedState["blocks"].([]any); ok && len(blocks) > 0 {
		first, ok := blocks[0].(map[string]any)
		if !ok {
			return
		}
		if transactions := framework.ToStringSlice(first["transactions"]); len(transactions) > 0 {
			model.Transactions = transactions
		}
		if header, ok := first["header"].(map[string]any); ok {
			model.PreviousHash = framework.StringValue(header["previous_hash"], model.PreviousHash)
			model.Timestamp = framework.StringValue(header["timestamp"], model.Timestamp)
			model.Nonce = int(framework.NumberValue(header["nonce"], float64(model.Nonce)))
			if headerRoot := framework.StringValue(header["merkle_root"], ""); headerRoot != "" {
				model.MerkleRoot = headerRoot
			}
		}
	}
}
