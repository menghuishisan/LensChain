package collectoragent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

// Event 表示发送给 SimEngine Core 的标准化采集事件。
type Event struct {
	Source      string          `json:"source"`
	TimestampMS int64           `json:"timestamp"`
	DataType    string          `json:"data_type"`
	Payload     json.RawMessage `json:"payload"`
}

// Agent 负责按固定周期采集实验容器状态并推送到 SimEngine Core。
type Agent struct {
	cfg     *Config
	client  *http.Client
	adapter adapter
}

// adapter 定义不同链生态采集器需要实现的统一轮询接口。
type adapter interface {
	Poll(ctx context.Context) ([]Event, error)
}

// rpcAdapter 负责通过 JSON-RPC 采集链节点区块与节点连接状态。
type rpcAdapter struct {
	cfg                   *Config
	client                *http.Client
	lastBlockNumber       uint64
	lastPeerSnapshot      string
	lastConsensusSnapshot string
}

// tcpAdapter 负责通过端口探测采集链节点基础可达性状态。
type tcpAdapter struct {
	cfg           *Config
	lastReachable *bool
}

// New 创建 Collector Agent 运行实例。
func New(cfg *Config) (*Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("collector config is required")
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	agent := &Agent{
		cfg:    cfg,
		client: httpClient,
	}
	switch cfg.Ecosystem {
	case "ethereum", "fisco":
		agent.adapter = &rpcAdapter{cfg: cfg, client: httpClient}
	case "fabric", "chainmaker":
		agent.adapter = &tcpAdapter{cfg: cfg}
	default:
		return nil, fmt.Errorf("unsupported collector ecosystem: %s", cfg.Ecosystem)
	}
	return agent, nil
}

// Run 启动 Collector Agent 主循环。
func (a *Agent) Run(ctx context.Context) error {
	if a == nil || a.cfg == nil || a.adapter == nil {
		return fmt.Errorf("collector agent is not initialized")
	}

	conn, err := websocket.Dial(a.cfg.CoreWebSocketURL, "", "http://localhost/")
	if err != nil {
		return fmt.Errorf("connect collector websocket: %w", err)
	}
	defer conn.Close()

	ticker := time.NewTicker(time.Duration(a.cfg.PollIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		events, err := a.adapter.Poll(ctx)
		if err != nil {
			return err
		}
		for _, event := range events {
			if err := websocket.JSON.Send(conn, event); err != nil {
				return fmt.Errorf("send collector event: %w", err)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Poll 采集基于 JSON-RPC 的链节点运行状态。
func (a *rpcAdapter) Poll(ctx context.Context) ([]Event, error) {
	if a.cfg == nil || a.cfg.RPCEndpoint == "" {
		return nil, fmt.Errorf("collector rpc endpoint is required")
	}
	events := make([]Event, 0, 8)

	blockNumber, err := a.currentBlockNumber(ctx)
	if err != nil {
		return nil, err
	}
	if blockNumber > 0 {
		start := blockNumber
		if a.lastBlockNumber > 0 && a.lastBlockNumber < blockNumber {
			start = a.lastBlockNumber + 1
		}
		for current := start; current <= blockNumber; current++ {
			blockEvents, err := a.collectBlockEvents(ctx, current)
			if err != nil {
				return nil, err
			}
			events = append(events, blockEvents...)
		}
		a.lastBlockNumber = blockNumber
	}

	if shouldCollect(a.cfg.CollectEvents, "peer_connected") || shouldCollect(a.cfg.CollectEvents, "peer_disconnected") {
		peerEvent, snapshot, err := a.collectPeerEvent(ctx)
		if err != nil {
			return nil, err
		}
		if peerEvent != nil && snapshot != a.lastPeerSnapshot {
			events = append(events, *peerEvent)
			a.lastPeerSnapshot = snapshot
		}
	}

	if a.cfg.Ecosystem == "fisco" && shouldCollect(a.cfg.CollectEvents, "consensus_status") {
		consensusEvent, snapshot, err := a.collectConsensusEvent(ctx)
		if err != nil {
			return nil, err
		}
		if consensusEvent != nil && snapshot != a.lastConsensusSnapshot {
			events = append(events, *consensusEvent)
			a.lastConsensusSnapshot = snapshot
		}
	}

	return events, nil
}

// Poll 采集基于端口可达性的链节点健康状态。
func (a *tcpAdapter) Poll(ctx context.Context) ([]Event, error) {
	address := tcpProbeAddress(a.cfg)
	if address == "" {
		return nil, fmt.Errorf("collector probe address is required")
	}
	reachable := false
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err == nil {
		reachable = true
		_ = conn.Close()
	}
	if a.lastReachable != nil && *a.lastReachable == reachable {
		return nil, nil
	}
	a.lastReachable = &reachable

	payload, _ := json.Marshal(map[string]any{
		"container_name": a.cfg.TargetContainer,
		"reachable":      reachable,
		"address":        address,
	})
	return []Event{{
		Source:      a.cfg.Ecosystem,
		TimestampMS: time.Now().UTC().UnixMilli(),
		DataType:    "peer",
		Payload:     payload,
	}}, nil
}

// callRPC 调用目标容器暴露的 JSON-RPC 接口并返回字符串结果。
func (a *rpcAdapter) callRPC(ctx context.Context, method string, params any) (string, error) {
	result, err := a.callRPCResult(ctx, method, params)
	if err != nil {
		return "", err
	}
	return stringifyRPCResult(result), nil
}

// callRPCResult 调用目标容器暴露的 JSON-RPC 接口并返回原始 result 字段。
func (a *rpcAdapter) callRPCResult(ctx context.Context, method string, params any) (json.RawMessage, error) {
	requestBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  paramsOrEmpty(params),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal collector rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.RPCEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("build collector rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call collector rpc: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read collector rpc response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("collector rpc status %d", resp.StatusCode)
	}

	var payload struct {
		Result json.RawMessage `json:"result"`
		Error  any             `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse collector rpc response: %w", err)
	}
	if payload.Error != nil {
		return nil, fmt.Errorf("collector rpc returned error")
	}
	return payload.Result, nil
}

// shouldCollect 判断当前事件类型是否在采集范围内。
func shouldCollect(events []string, target string) bool {
	if len(events) == 0 {
		return true
	}
	for _, item := range events {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

// currentBlockNumber 读取当前链节点的最新块高。
func (a *rpcAdapter) currentBlockNumber(ctx context.Context) (uint64, error) {
	raw, err := a.callRPCResult(ctx, blockNumberMethod(a.cfg.Ecosystem), blockNumberParams(a.cfg.Ecosystem))
	if err != nil {
		return 0, err
	}
	return parseBlockNumber(raw)
}

// collectBlockEvents 根据块高拉取区块详情、交易和合约事件。
func (a *rpcAdapter) collectBlockEvents(ctx context.Context, blockNumber uint64) ([]Event, error) {
	block, err := a.getBlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, 8)
	now := time.Now().UTC().UnixMilli()
	blockNumberText := encodeBlockNumber(a.cfg.Ecosystem, blockNumber)

	if shouldCollect(a.cfg.CollectEvents, "new_block") {
		payload, _ := json.Marshal(map[string]any{
			"container_name": a.cfg.TargetContainer,
			"block_number":   blockNumberText,
			"block":          block,
			"endpoint":       a.cfg.RPCEndpoint,
		})
		events = append(events, Event{
			Source:      a.cfg.Ecosystem,
			TimestampMS: now,
			DataType:    "block",
			Payload:     payload,
		})
	}

	transactions := extractTransactions(block)
	if shouldCollect(a.cfg.CollectEvents, "new_transaction") {
		for _, tx := range transactions {
			payload, _ := json.Marshal(map[string]any{
				"container_name": a.cfg.TargetContainer,
				"block_number":   blockNumberText,
				"transaction":    tx,
				"endpoint":       a.cfg.RPCEndpoint,
			})
			events = append(events, Event{
				Source:      a.cfg.Ecosystem,
				TimestampMS: now,
				DataType:    "transaction",
				Payload:     payload,
			})
		}
	}

	if shouldCollect(a.cfg.CollectEvents, "contract_event") {
		contractEvents, err := a.collectContractEvents(ctx, blockNumberText, transactions)
		if err != nil {
			return nil, err
		}
		events = append(events, contractEvents...)
	}

	return events, nil
}

// collectPeerEvent 读取节点连接状态并在状态变化时输出标准 peer 事件。
func (a *rpcAdapter) collectPeerEvent(ctx context.Context) (*Event, string, error) {
	raw, err := a.callRPCResult(ctx, peerCountMethod(a.cfg.Ecosystem), peerCountParams(a.cfg.Ecosystem))
	if err != nil {
		return nil, "", err
	}
	snapshot := strings.TrimSpace(string(raw))
	payload, _ := json.Marshal(map[string]any{
		"container_name": a.cfg.TargetContainer,
		"peer_state":     decodeRPCResult(raw),
		"endpoint":       a.cfg.RPCEndpoint,
	})
	return &Event{
		Source:      a.cfg.Ecosystem,
		TimestampMS: time.Now().UTC().UnixMilli(),
		DataType:    "peer",
		Payload:     payload,
	}, snapshot, nil
}

// collectConsensusEvent 读取 FISCO BCOS 共识状态并在变化时输出标准 consensus 事件。
func (a *rpcAdapter) collectConsensusEvent(ctx context.Context) (*Event, string, error) {
	raw, err := a.callRPCResult(ctx, consensusMethod(a.cfg.Ecosystem), consensusParams(a.cfg.Ecosystem))
	if err != nil {
		return nil, "", err
	}
	snapshot := strings.TrimSpace(string(raw))
	payload, _ := json.Marshal(map[string]any{
		"container_name":   a.cfg.TargetContainer,
		"consensus_status": decodeRPCResult(raw),
		"endpoint":         a.cfg.RPCEndpoint,
	})
	return &Event{
		Source:      a.cfg.Ecosystem,
		TimestampMS: time.Now().UTC().UnixMilli(),
		DataType:    "consensus",
		Payload:     payload,
	}, snapshot, nil
}

// getBlockByNumber 根据不同生态的 RPC 协议获取完整区块详情。
func (a *rpcAdapter) getBlockByNumber(ctx context.Context, blockNumber uint64) (map[string]any, error) {
	raw, err := a.callRPCResult(ctx, blockDetailMethod(a.cfg.Ecosystem), blockDetailParams(a.cfg.Ecosystem, blockNumber))
	if err != nil {
		return nil, err
	}
	block := make(map[string]any)
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, fmt.Errorf("parse block detail: %w", err)
	}
	return block, nil
}

// collectContractEvents 读取交易回执中的日志并转换为标准合约事件。
func (a *rpcAdapter) collectContractEvents(ctx context.Context, blockNumber string, transactions []map[string]any) ([]Event, error) {
	events := make([]Event, 0)
	for _, tx := range transactions {
		txHash := extractTransactionHash(tx)
		if txHash == "" {
			continue
		}
		logs, err := a.getTransactionLogs(ctx, txHash)
		if err != nil {
			return nil, err
		}
		for _, logItem := range logs {
			payload, _ := json.Marshal(map[string]any{
				"container_name": a.cfg.TargetContainer,
				"block_number":   blockNumber,
				"tx_hash":        txHash,
				"log":            logItem,
				"endpoint":       a.cfg.RPCEndpoint,
			})
			events = append(events, Event{
				Source:      a.cfg.Ecosystem,
				TimestampMS: time.Now().UTC().UnixMilli(),
				DataType:    "contract_event",
				Payload:     payload,
			})
		}
	}
	return events, nil
}

// getTransactionLogs 获取交易回执中的日志列表。
func (a *rpcAdapter) getTransactionLogs(ctx context.Context, txHash string) ([]map[string]any, error) {
	raw, err := a.callRPCResult(ctx, transactionReceiptMethod(a.cfg.Ecosystem), transactionReceiptParams(a.cfg.Ecosystem, txHash))
	if err != nil {
		return nil, err
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("parse transaction receipt: %w", err)
	}
	logs := make([]map[string]any, 0)
	switch typed := receipt["logs"].(type) {
	case []any:
		for _, item := range typed {
			if asMap, ok := item.(map[string]any); ok {
				logs = append(logs, asMap)
			}
		}
	}
	return logs, nil
}

// blockNumberMethod 返回不同生态查询区块高度的方法名。
func blockNumberMethod(ecosystem string) string {
	switch ecosystem {
	case "fisco":
		return "getBlockNumber"
	default:
		return "eth_blockNumber"
	}
}

// blockNumberParams 返回查询当前块高时的标准参数。
func blockNumberParams(ecosystem string) any {
	if ecosystem == "fisco" {
		return []any{1}
	}
	return nil
}

// blockDetailMethod 返回查询完整区块详情的方法名。
func blockDetailMethod(ecosystem string) string {
	switch ecosystem {
	case "fisco":
		return "getBlockByNumber"
	default:
		return "eth_getBlockByNumber"
	}
}

// blockDetailParams 返回查询完整区块详情的参数列表。
func blockDetailParams(ecosystem string, blockNumber uint64) any {
	switch ecosystem {
	case "fisco":
		return []any{1, encodeBlockNumber(ecosystem, blockNumber), true}
	default:
		return []any{encodeBlockNumber(ecosystem, blockNumber), true}
	}
}

// peerCountMethod 返回不同生态查询节点连接数的方法名。
func peerCountMethod(ecosystem string) string {
	switch ecosystem {
	case "fisco":
		return "getPeers"
	default:
		return "net_peerCount"
	}
}

// peerCountParams 返回查询节点连接状态的标准参数。
func peerCountParams(ecosystem string) any {
	if ecosystem == "fisco" {
		return []any{}
	}
	return nil
}

// consensusMethod 返回查询共识状态的方法名。
func consensusMethod(ecosystem string) string {
	if ecosystem == "fisco" {
		return "getConsensusStatus"
	}
	return ""
}

// consensusParams 返回查询共识状态的标准参数。
func consensusParams(ecosystem string) any {
	if ecosystem == "fisco" {
		return []any{1}
	}
	return nil
}

// transactionReceiptMethod 返回查询交易回执的方法名。
func transactionReceiptMethod(ecosystem string) string {
	switch ecosystem {
	case "fisco":
		return "getTransactionReceipt"
	default:
		return "eth_getTransactionReceipt"
	}
}

// transactionReceiptParams 返回查询交易回执的标准参数。
func transactionReceiptParams(ecosystem string, txHash string) any {
	switch ecosystem {
	case "fisco":
		return []any{1, txHash}
	default:
		return []any{txHash}
	}
}

// paramsOrEmpty 保证 JSON-RPC 请求中的 params 字段始终为数组。
func paramsOrEmpty(params any) any {
	if params == nil {
		return []any{}
	}
	return params
}

// parseBlockNumber 将十六进制或十进制块高字符串转换为无符号整数。
func parseBlockNumber(raw json.RawMessage) (uint64, error) {
	value := stringifyRPCResult(raw)
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		return strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X"), 16, 64)
	}
	return strconv.ParseUint(value, 10, 64)
}

// encodeBlockNumber 根据链生态要求输出块高参数。
func encodeBlockNumber(ecosystem string, blockNumber uint64) string {
	if ecosystem == "fisco" {
		return fmt.Sprintf("0x%x", blockNumber)
	}
	return fmt.Sprintf("0x%x", blockNumber)
}

// stringifyRPCResult 将 RPC result 字段转换为字符串。
func stringifyRPCResult(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(string(raw))
}

// decodeRPCResult 将 RPC result 字段解码为通用 JSON 结构。
func decodeRPCResult(raw json.RawMessage) any {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return stringifyRPCResult(raw)
	}
	return payload
}

// extractTransactions 从区块详情中提取交易列表。
func extractTransactions(block map[string]any) []map[string]any {
	items, ok := block["transactions"].([]any)
	if !ok {
		return nil
	}
	transactions := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if tx, ok := item.(map[string]any); ok {
			transactions = append(transactions, tx)
		}
	}
	return transactions
}

// extractTransactionHash 提取交易哈希字段。
func extractTransactionHash(tx map[string]any) string {
	if tx == nil {
		return ""
	}
	for _, key := range []string{"hash", "txHash", "transactionHash"} {
		if value, ok := tx[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// tcpProbeAddress 根据采集配置选择端口探测地址。
func tcpProbeAddress(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	switch {
	case cfg.RPCEndpoint != "":
		return strings.TrimPrefix(strings.TrimPrefix(cfg.RPCEndpoint, "http://"), "https://")
	case cfg.WSEndpoint != "":
		return strings.TrimPrefix(strings.TrimPrefix(cfg.WSEndpoint, "ws://"), "wss://")
	default:
		return ""
	}
}
