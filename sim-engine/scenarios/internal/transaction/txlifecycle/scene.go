package txlifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造交易生命周期场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "tx-lifecycle",
		Title:        "交易生命周期",
		Phase:        "创建",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   18,
		Stages:       []string{"创建", "签名", "广播", "内存池", "打包", "确认"},
		Nodes: []framework.Node{
			{ID: "wallet", Label: "Wallet", Status: "active", Role: "wallet", X: 80, Y: 200},
			{ID: "signer", Label: "Signer", Status: "normal", Role: "signer", X: 220, Y: 200},
			{ID: "peer", Label: "Peer", Status: "normal", Role: "peer", X: 360, Y: 200},
			{ID: "mempool", Label: "Mempool", Status: "normal", Role: "mempool", X: 520, Y: 200},
			{ID: "miner", Label: "Miner", Status: "normal", Role: "miner", X: 680, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化第一笔待处理交易。
func Init(state *framework.SceneState, input framework.InitInput) error {
	txCode := framework.StringValue(input.Params["tx"], "tx-1")
	tx := newTransaction(txCode)
	applySharedTransactionState(&tx, input.SharedState, state.LinkGroup)
	return rebuildState(state, tx, "创建")
}

// Step 按文档定义推进交易生命周期六阶段。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	tx := decodeTransaction(state)
	applySharedTransactionState(&tx, input.SharedState, state.LinkGroup)
	nextPhase := nextPhase(framework.StringValue(state.Data["phase_name"], "创建"))
	switch nextPhase {
	case "签名":
		tx.Signature = signTransaction(tx)
	case "广播":
		tx.BroadcastPeers = []string{"peer-a", "peer-b", "peer-c"}
		tx.EventLog = append(tx.EventLog, "Broadcasted")
	case "内存池":
		tx.InMempool = true
		tx.EventLog = append(tx.EventLog, "Pending")
	case "打包":
		tx.BlockHash = blockHash(tx)
		tx.GasUsed = 21000
		tx.EventLog = append(tx.EventLog, "Included")
	case "确认":
		tx.Confirmations = 1
		tx.Status = "confirmed"
		tx.EventLog = append(tx.EventLog, "TransferConfirmed")
	}
	if err := rebuildState(state, tx, nextPhase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, nextPhase, fmt.Sprintf("交易 %s 进入%s阶段。", tx.ID, nextPhase), "info")
	return framework.StepOutput{
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, tx),
	}, nil
}

// HandleAction 创建一笔新的交易并重置流程。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	txCode := framework.StringValue(input.Params["tx"], "tx-new")
	tx := newTransaction(txCode)
	if err := rebuildState(state, tx, "创建"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "创建新交易", fmt.Sprintf("已创建新交易 %s。", txCode), "success")
	return framework.ActionOutput{
		Success:    true,
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, tx),
	}, nil
}

// BuildRenderState 将交易状态、阶段和关键指标输出给渲染层。
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

// SyncSharedState 在交易处理组共享状态变化后重建交易生命周期场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	tx := decodeTransaction(state)
	applySharedTransactionState(&tx, sharedState, state.LinkGroup)
	return rebuildState(state, tx, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// transactionState 保存一笔交易在各生命周期阶段的业务状态。
type transactionState struct {
	ID             string   `json:"id"`
	Nonce          int      `json:"nonce"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	Value          int      `json:"value"`
	PayloadHash    string   `json:"payload_hash"`
	Signature      string   `json:"signature"`
	BroadcastPeers []string `json:"broadcast_peers"`
	InMempool      bool     `json:"in_mempool"`
	BlockHash      string   `json:"block_hash"`
	Confirmations  int      `json:"confirmations"`
	GasUsed        int      `json:"gas_used"`
	Status         string   `json:"status"`
	EventLog       []string `json:"event_log"`
}

// newTransaction 创建一笔完整但尚未签名和广播的交易对象。
func newTransaction(code string) transactionState {
	payload := fmt.Sprintf("%s|alice|bob|25|0", code)
	hash := sha256.Sum256([]byte(payload))
	return transactionState{
		ID:          code,
		Nonce:       0,
		From:        "alice",
		To:          "bob",
		Value:       25,
		PayloadHash: hex.EncodeToString(hash[:]),
		Status:      "created",
		EventLog:    []string{"TransferInitiated"},
	}
}

// rebuildState 按当前交易状态重建阶段、消息流、指标与可视化数据。
func rebuildState(state *framework.SceneState, tx transactionState, phase string) error {
	tx.Status = phaseStatus(phase)
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Messages = buildMessages(tx, phase)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		if index == state.PhaseIndex {
			state.Nodes[index].Status = "active"
		}
		if phase == "确认" && state.Nodes[index].ID == "miner" {
			state.Nodes[index].Status = "success"
		}
	}
	state.Metrics = []framework.Metric{
		{Key: "tx_id", Label: "交易标识", Value: tx.ID, Tone: "info"},
		{Key: "status", Label: "当前状态", Value: tx.Status, Tone: statusTone(tx.Status)},
		{Key: "mempool", Label: "是否入池", Value: framework.BoolText(tx.InMempool), Tone: "warning"},
		{Key: "gas", Label: "Gas Used", Value: fmt.Sprintf("%d", tx.GasUsed), Tone: "warning"},
		{Key: "confirmations", Label: "确认数", Value: fmt.Sprintf("%d", tx.Confirmations), Tone: "success"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "交易哈希", Value: tx.PayloadHash[:16] + "..."},
		{Label: "签名", Value: framework.AbbreviateOr(tx.Signature, 16, "未生成")},
		{Label: "区块哈希", Value: framework.AbbreviateOr(tx.BlockHash, 16, "未生成")},
		{Label: "事件日志", Value: strings.Join(tx.EventLog, ", ")},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"transaction": tx,
	}
	state.Extra = map[string]any{
		"description": "该场景使用真实交易对象、签名摘要、广播节点列表、入池状态和确认状态推进流程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeTransaction 从通用 JSON 状态中恢复交易对象。
func decodeTransaction(state *framework.SceneState) transactionState {
	entry, ok := state.Data["transaction"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["transaction"].(transactionState); ok {
			return typed
		}
		return newTransaction("tx-1")
	}
	return transactionState{
		ID:             framework.StringValue(entry["id"], "tx-1"),
		Nonce:          int(framework.NumberValue(entry["nonce"], 0)),
		From:           framework.StringValue(entry["from"], "alice"),
		To:             framework.StringValue(entry["to"], "bob"),
		Value:          int(framework.NumberValue(entry["value"], 25)),
		PayloadHash:    framework.StringValue(entry["payload_hash"], ""),
		Signature:      framework.StringValue(entry["signature"], ""),
		BroadcastPeers: framework.ToStringSlice(entry["broadcast_peers"]),
		InMempool:      framework.BoolValue(entry["in_mempool"], false),
		BlockHash:      framework.StringValue(entry["block_hash"], ""),
		Confirmations:  int(framework.NumberValue(entry["confirmations"], 0)),
		GasUsed:        int(framework.NumberValue(entry["gas_used"], 0)),
		Status:         framework.StringValue(entry["status"], "created"),
		EventLog:       framework.ToStringSlice(entry["event_log"]),
	}
}

// buildMessages 依据阶段生成交易在管道中的流转消息。
func buildMessages(tx transactionState, phase string) []framework.Message {
	switch phase {
	case "创建":
		return []framework.Message{{ID: tx.ID + "-draft", Label: "Draft", Kind: "transaction", Status: phase, SourceID: "wallet", TargetID: "signer"}}
	case "签名":
		return []framework.Message{{ID: tx.ID + "-signature", Label: "SignedTx", Kind: "transaction", Status: phase, SourceID: "signer", TargetID: "peer"}}
	case "广播":
		return []framework.Message{{ID: tx.ID + "-broadcast", Label: "Broadcast", Kind: "transaction", Status: phase, SourceID: "peer", TargetID: "mempool"}}
	case "内存池":
		return []framework.Message{{ID: tx.ID + "-mempool", Label: "Mempool", Kind: "transaction", Status: phase, SourceID: "peer", TargetID: "mempool"}}
	case "打包":
		return []framework.Message{{ID: tx.ID + "-block", Label: "Block Include", Kind: "transaction", Status: phase, SourceID: "mempool", TargetID: "miner"}}
	case "确认":
		return []framework.Message{{ID: tx.ID + "-confirm", Label: "Confirmed", Kind: "transaction", Status: phase, SourceID: "miner", TargetID: "wallet"}}
	default:
		return []framework.Message{}
	}
}

// phaseStatus 将阶段名称映射为交易状态文案。
func phaseStatus(phase string) string {
	switch phase {
	case "创建":
		return "created"
	case "签名":
		return "signed"
	case "广播":
		return "broadcast"
	case "内存池":
		return "mempool"
	case "打包":
		return "included"
	case "确认":
		return "confirmed"
	default:
		return "created"
	}
}

// nextPhase 返回交易生命周期的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "创建":
		return "签名"
	case "签名":
		return "广播"
	case "广播":
		return "内存池"
	case "内存池":
		return "打包"
	case "打包":
		return "确认"
	default:
		return "确认"
	}
}

// phaseIndex 将阶段名称映射为管道索引。
func phaseIndex(phase string) int {
	switch phase {
	case "创建":
		return 0
	case "签名":
		return 1
	case "广播":
		return 2
	case "内存池":
		return 3
	case "打包":
		return 4
	case "确认":
		return 4
	default:
		return 0
	}
}

// signTransaction 基于交易摘要构造简化签名结果。
func signTransaction(tx transactionState) string {
	sum := sha256.Sum256([]byte(tx.PayloadHash + "|signer"))
	return hex.EncodeToString(sum[:])
}

// blockHash 基于交易签名生成打包后的区块哈希。
func blockHash(tx transactionState) string {
	sum := sha256.Sum256([]byte(tx.Signature + "|block"))
	return hex.EncodeToString(sum[:])
}

// statusTone 根据交易状态选择指标色调。
func statusTone(status string) string {
	switch status {
	case "confirmed":
		return "success"
	case "included":
		return "warning"
	default:
		return "info"
	}
}

// boolToInt 将布尔值转换为 0/1，便于联动共享状态输出。
func boolToInt(flag bool) int {
	if flag {
		return 1
	}
	return 0
}

// buildSharedDiff 按联动组边界输出共享状态，避免不同组的语义混发。
func buildSharedDiff(linkGroup string, tx transactionState) map[string]any {
	switch linkGroup {
	case "tx-processing-group":
		return map[string]any{
			"mempool": map[string]any{
				"size":          boolToInt(tx.InMempool),
				"latest_tx_id":  tx.ID,
				"latest_status": tx.Status,
			},
			"gas": map[string]any{
				"latest_tx_id": tx.ID,
				"used":         tx.GasUsed,
			},
			"balances": map[string]any{
				tx.From: balanceByRole(tx, tx.From),
				tx.To:   balanceByRole(tx, tx.To),
			},
		}
	case "blockchain-integrity-group":
		return map[string]any{
			"blocks":       buildIntegrityBlocks(tx),
			"transactions": buildIntegrityTransactions(tx),
			"merkle_root":  integrityMerkleRoot(tx),
		}
	default:
		return nil
	}
}

// buildIntegrityBlocks 为区块链完整性组输出最小但完整的区块摘要。
func buildIntegrityBlocks(tx transactionState) []map[string]any {
	if tx.BlockHash == "" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"id":           "block-" + tx.ID,
			"height":       1,
			"hash":         tx.BlockHash,
			"prev_hash":    "",
			"transactions": []string{tx.ID},
			"branch":       "main",
			"is_longest":   true,
		},
	}
}

// buildIntegrityTransactions 输出区块链完整性组可复用的交易索引。
func buildIntegrityTransactions(tx transactionState) map[string]any {
	return map[string]any{
		tx.ID: map[string]any{
			"status":        tx.Status,
			"confirmations": tx.Confirmations,
			"block_hash":    tx.BlockHash,
			"from":          tx.From,
			"to":            tx.To,
			"value":         tx.Value,
		},
	}
}

// integrityMerkleRoot 使用交易摘要和区块哈希构造稳定的 Merkle Root 语义。
func integrityMerkleRoot(tx transactionState) string {
	sum := sha256.Sum256([]byte(tx.PayloadHash + "|" + tx.BlockHash))
	return hex.EncodeToString(sum[:])
}

// balanceByRole 返回交易处理组中的账户余额快照。
func balanceByRole(tx transactionState, account string) int {
	switch account {
	case tx.From:
		return 100 - tx.Value
	case tx.To:
		return 50 + tx.Value
	default:
		return 0
	}
}

// applySharedTransactionState 将当前联动组的共享结果映射回交易生命周期。
func applySharedTransactionState(tx *transactionState, sharedState map[string]any, linkGroup string) {
	if len(sharedState) == 0 {
		return
	}
	switch linkGroup {
	case "tx-processing-group":
		if mempool, ok := sharedState["mempool"].(map[string]any); ok && int(framework.NumberValue(mempool["size"], 0)) > 0 {
			tx.InMempool = true
			tx.Status = framework.StringValue(mempool["latest_status"], tx.Status)
		}
		if gas, ok := sharedState["gas"].(map[string]any); ok {
			tx.GasUsed = int(framework.NumberValue(gas["used"], float64(tx.GasUsed)))
		}
	case "blockchain-integrity-group":
		if transactions, ok := sharedState["transactions"].(map[string]any); ok {
			if linked, ok := transactions[tx.ID].(map[string]any); ok {
				tx.Status = framework.StringValue(linked["status"], tx.Status)
				tx.Confirmations = int(framework.NumberValue(linked["confirmations"], float64(tx.Confirmations)))
				tx.BlockHash = framework.StringValue(linked["block_hash"], tx.BlockHash)
			}
		}
		if sharedRoot := framework.StringValue(sharedState["merkle_root"], ""); sharedRoot != "" && tx.BlockHash == "" {
			tx.BlockHash = sharedRoot
		}
	}
}
