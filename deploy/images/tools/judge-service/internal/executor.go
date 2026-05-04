// executor.go — judge-service 核心执行引擎。
// 实现合约部署、攻击交易执行和初始化交易回放，
// 通过 go-ethereum RPC 客户端与 EVM 链交互。
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ── 合约部署 ───────────────────────────────────────────────────

// DeployContracts 按 deploy_order 在链上部署合约并返回绑定信息。
func DeployContracts(ctx context.Context, rpcURL string, contracts []ContractSpec) ([]ContractBinding, error) {
	client, rpcClient, err := dialChain(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	from, err := getCoinbase(ctx, rpcClient)
	if err != nil {
		return nil, fmt.Errorf("获取 coinbase 账户失败: %w", err)
	}

	sorted := make([]ContractSpec, len(contracts))
	copy(sorted, contracts)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ChallengeID == sorted[j].ChallengeID {
			return sorted[i].DeployOrder < sorted[j].DeployOrder
		}
		return sorted[i].ChallengeID < sorted[j].ChallengeID
	})

	results := make([]ContractBinding, 0, len(sorted))
	for _, spec := range sorted {
		addr, err := deployOneContract(ctx, client, rpcClient, from, &spec)
		if err != nil {
			return nil, fmt.Errorf("部署合约 %s 失败: %w", spec.ContractName, err)
		}
		results = append(results, ContractBinding{
			ChallengeID:  fmtInt64(spec.ChallengeID),
			ContractName: spec.ContractName,
			Address:      addr.Hex(),
			ABIJSON:      spec.ABIJSON,
			PatchVersion: 0,
			IsPatched:    false,
		})
	}
	return results, nil
}

// deployOneContract 在链上部署单个合约并等待回执，返回合约地址。
// 支持自动编码构造参数并拼接到 bytecode 末尾。
func deployOneContract(ctx context.Context, client *ethclient.Client, rpcClient *rpc.Client, from common.Address, spec *ContractSpec) (common.Address, error) {
	bytecodeHex := strings.TrimSpace(spec.Bytecode)
	if !strings.HasPrefix(bytecodeHex, "0x") {
		bytecodeHex = "0x" + bytecodeHex
	}
	bytecodeBytes := common.FromHex(bytecodeHex)

	// 编码构造参数
	if len(spec.ConstructorArgs) > 0 && strings.TrimSpace(spec.ABIJSON) != "" && spec.ABIJSON != "[]" {
		parsedABI, err := abi.JSON(strings.NewReader(spec.ABIJSON))
		if err == nil && parsedABI.Constructor.Inputs != nil {
			packed, err := packArgs(parsedABI.Constructor.Inputs, spec.ConstructorArgs)
			if err != nil {
				return common.Address{}, fmt.Errorf("编码构造参数失败: %w", err)
			}
			bytecodeBytes = append(bytecodeBytes, packed...)
		}
	}

	tx := map[string]interface{}{
		"from": from.Hex(),
		"data": hexutil.Encode(bytecodeBytes),
		"gas":  hexutil.EncodeUint64(12_000_000),
	}
	var txHash common.Hash
	if err := rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", tx); err != nil {
		return common.Address{}, err
	}
	receipt, err := waitReceipt(ctx, client, txHash)
	if err != nil {
		return common.Address{}, err
	}
	return receipt.ContractAddress, nil
}

// ── 攻击执行 ───────────────────────────────────────────────────

// ExecuteAttack 执行攻击交易并验证断言。
func ExecuteAttack(ctx context.Context, req *AttackRequest) *AttackResponse {
	startedAt := time.Now()
	resp := &AttackResponse{Results: []AssertionResult{}}

	client, rpcClient, err := dialChain(ctx, req.RPCURL)
	if err != nil {
		resp.ErrorMessage = stringPtr("连接链 RPC 失败: " + err.Error())
		return resp
	}
	defer client.Close()

	from, err := getCoinbase(ctx, rpcClient)
	if err != nil {
		resp.ErrorMessage = stringPtr("获取 coinbase 账户失败: " + err.Error())
		return resp
	}

	contractMap := buildContractMap(req.Contracts)
	txHash, err := executeSubmission(ctx, client, rpcClient, from, req.Submission, req.DefaultTarget, contractMap)
	if err != nil {
		resp.ErrorMessage = stringPtr(err.Error())
		resp.ExecutionTimeMS = int(time.Since(startedAt).Milliseconds())
		return resp
	}
	if txHash != (common.Hash{}) {
		h := txHash.Hex()
		resp.TxHash = &h
	}

	receipt, _ := waitReceipt(ctx, client, txHash)

	// 执行断言
	for _, assertion := range req.Assertions {
		result, err := EvaluateAssertion(ctx, client, rpcClient, receipt, &assertion, contractMap)
		if err != nil {
			log.Printf("[assertion] %s error: %v", assertion.AssertionType, err)
			resp.Results = append(resp.Results, AssertionResult{
				Type:     assertion.AssertionType,
				Target:   assertion.Target,
				Expected: assertion.ExpectedValue,
				Actual:   "error: " + err.Error(),
				Passed:   false,
			})
			continue
		}
		resp.Results = append(resp.Results, *result)
	}

	allPassed := true
	for _, r := range resp.Results {
		if !r.Passed {
			allPassed = false
			break
		}
	}
	resp.AllPassed = allPassed && resp.ErrorMessage == nil
	resp.ExecutionTimeMS = int(time.Since(startedAt).Milliseconds())
	return resp
}

// executeSubmission 解析并执行攻击提交数据。
// 支持三种格式：
//   - JSON 结构化提交（包含 bytecode 或 to+data）
//   - 纯 hex calldata（发给 defaultTarget）
//   - 纯 hex bytecode（无目标时当作合约部署）
func executeSubmission(ctx context.Context, client *ethclient.Client, rpcClient *rpc.Client, from common.Address, submission, defaultTarget string, contractMap map[string]*ContractBinding) (common.Hash, error) {
	submission = strings.TrimSpace(submission)
	if submission == "" {
		return common.Hash{}, nil
	}

	// 尝试解析为 JSON 结构化提交
	if strings.HasPrefix(submission, "{") {
		var spec struct {
			Bytecode        string        `json:"bytecode"`
			ABI             interface{}   `json:"abi"`
			ConstructorArgs []interface{} `json:"constructor_args"`
			To              string        `json:"to"`
			Data            string        `json:"data"`
			Value           string        `json:"value"`
		}
		if err := json.Unmarshal([]byte(submission), &spec); err == nil {
			if spec.Bytecode != "" {
				// 部署合约型攻击
				return sendRawTx(ctx, rpcClient, from, nil, spec.Bytecode)
			}
			to := resolveAddress(spec.To, contractMap)
			data := spec.Data
			if data == "" {
				data = "0x"
			}
			return sendRawTx(ctx, rpcClient, from, to, data)
		}
	}

	// 纯 hex calldata → 发给 defaultTarget
	if isHex(submission) {
		target := resolveAddress(defaultTarget, contractMap)
		if target != nil {
			return sendRawTx(ctx, rpcClient, from, target, submission)
		}
		// 无目标则当作合约部署
		return sendRawTx(ctx, rpcClient, from, nil, submission)
	}

	return common.Hash{}, fmt.Errorf("无法解析攻击提交数据")
}

// ── 初始化交易 ─────────────────────────────────────────────────

// ExecuteSetupTransactions 依序执行初始化交易。
func ExecuteSetupTransactions(ctx context.Context, req *SetupRequest) (int, error) {
	client, rpcClient, err := dialChain(ctx, req.RPCURL)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	contractMap := buildContractMap(req.Contracts)
	pinnedMap := make(map[string]*PinnedContract)
	for i := range req.PinnedContracts {
		pinnedMap[req.PinnedContracts[i].Name] = &req.PinnedContracts[i]
	}
	accountMap := make(map[string]int)
	for i, acc := range req.Accounts {
		if acc.Name != "" {
			accountMap[acc.Name] = i
		}
	}
	impersonatedSet := make(map[string]bool)
	for _, addr := range req.ImpersonatedAccounts {
		impersonatedSet[strings.ToLower(addr)] = true
	}

	// 获取链上所有账户
	var chainAccounts []common.Address
	_ = rpcClient.CallContext(ctx, &chainAccounts, "eth_accounts")

	applied := 0
	for _, tx := range req.SetupTransactions {
		from, err := resolveSetupFrom(ctx, rpcClient, tx.From, accountMap, chainAccounts, impersonatedSet)
		if err != nil {
			return applied, fmt.Errorf("解析发送方失败: %w", err)
		}
		to := resolveSetupTo(tx.To, contractMap, pinnedMap)
		value := parseValue(tx.Value)

		if tx.Function != "" && to != nil {
			// 合约调用
			binding := findContractBinding(tx.To, contractMap, pinnedMap)
			if binding == nil {
				return applied, fmt.Errorf("找不到合约 %s", tx.To)
			}
			abiJSON := binding.ABIJSON
			if abiJSON == "" {
				abiJSON = "[]"
			}
			parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
			if err != nil {
				return applied, fmt.Errorf("解析 ABI 失败: %w", err)
			}
			method, ok := parsedABI.Methods[tx.Function]
			if !ok {
				return applied, fmt.Errorf("ABI 中找不到方法 %s", tx.Function)
			}
			args := resolveContractArgs(tx.Args, contractMap, pinnedMap)
			packed, err := packArgs(method.Inputs, args)
			if err != nil {
				return applied, fmt.Errorf("编码方法参数失败: %w", err)
			}
			calldata := append(method.ID, packed...)
			txObj := map[string]interface{}{
				"from":  from.Hex(),
				"to":    to.Hex(),
				"data":  hexutil.Encode(calldata),
				"gas":   hexutil.EncodeUint64(12_000_000),
				"value": hexutil.EncodeBig(value),
			}
			var txHash common.Hash
			if err := rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", txObj); err != nil {
				return applied, fmt.Errorf("发送交易失败: %w", err)
			}
			if _, err := waitReceipt(ctx, client, txHash); err != nil {
				return applied, err
			}
		} else {
			// 简单转账
			txObj := map[string]interface{}{
				"from":  from.Hex(),
				"data":  "0x",
				"gas":   hexutil.EncodeUint64(12_000_000),
				"value": hexutil.EncodeBig(value),
			}
			if to != nil {
				txObj["to"] = to.Hex()
			}
			var txHash common.Hash
			if err := rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", txObj); err != nil {
				return applied, err
			}
			if _, err := waitReceipt(ctx, client, txHash); err != nil {
				return applied, err
			}
		}
		applied++
	}
	return applied, nil
}

// ── 链 RPC 工具函数 ────────────────────────────────────────────

// dialChain 连接链节点 RPC，同时返回高级客户端和底层 RPC 客户端。
func dialChain(ctx context.Context, rpcURL string) (*ethclient.Client, *rpc.Client, error) {
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, nil, err
	}
	return ethclient.NewClient(rpcClient), rpcClient, nil
}

// getCoinbase 获取链上第一个可用账户作为交易发送方。
func getCoinbase(ctx context.Context, rpcClient *rpc.Client) (common.Address, error) {
	var accounts []common.Address
	if err := rpcClient.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, err
	}
	if len(accounts) == 0 {
		return common.Address{}, fmt.Errorf("链上无可用账户")
	}
	return accounts[0], nil
}

// waitReceipt 轮询等待交易回执，每 500ms 检查一次，支持 ctx 超时取消。
// 如果超过 30 秒仍未获取到回执，返回超时错误。
func waitReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("等待交易回执超时: %s", txHash.Hex())
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err == nil && receipt != nil {
				return receipt, nil
			}
		}
	}
}

// sendRawTx 发送原始交易，若 to 为 nil 则为合约创建交易。
func sendRawTx(ctx context.Context, rpcClient *rpc.Client, from common.Address, to *common.Address, data string) (common.Hash, error) {
	if !strings.HasPrefix(data, "0x") {
		data = "0x" + data
	}
	tx := map[string]interface{}{
		"from": from.Hex(),
		"data": data,
		"gas":  hexutil.EncodeUint64(12_000_000),
	}
	if to != nil {
		tx["to"] = to.Hex()
	}
	var txHash common.Hash
	if err := rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", tx); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

// ── 辅助函数 ───────────────────────────────────────────────────

// buildContractMap 将合约绑定列表构建为名称→绑定的双向映射（同时支持按地址查找）。
func buildContractMap(contracts []ContractBinding) map[string]*ContractBinding {
	m := make(map[string]*ContractBinding)
	for i := range contracts {
		m[contracts[i].ContractName] = &contracts[i]
		if contracts[i].Address != "" {
			m[strings.ToLower(contracts[i].Address)] = &contracts[i]
		}
	}
	return m
}

type abiHolder struct {
	ABIJSON string
}

// findContractBinding 在合约映射和固定引用映射中查找合约绑定信息。
func findContractBinding(name string, contractMap map[string]*ContractBinding, pinnedMap map[string]*PinnedContract) *ContractBinding {
	if b, ok := contractMap[name]; ok {
		return b
	}
	if p, ok := pinnedMap[name]; ok {
		return &ContractBinding{ContractName: p.Name, Address: p.Address}
	}
	return nil
}

// resolveAddress 将合约名称或 hex 地址解析为链上地址。
func resolveAddress(name string, contractMap map[string]*ContractBinding) *common.Address {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if common.IsHexAddress(name) {
		addr := common.HexToAddress(name)
		return &addr
	}
	if b, ok := contractMap[name]; ok {
		addr := common.HexToAddress(b.Address)
		return &addr
	}
	return nil
}

// resolveSetupTo 解析初始化交易的目标地址，支持合约名称和固定引用。
func resolveSetupTo(name string, contractMap map[string]*ContractBinding, pinnedMap map[string]*PinnedContract) *common.Address {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if common.IsHexAddress(name) {
		addr := common.HexToAddress(name)
		return &addr
	}
	if b, ok := contractMap[name]; ok {
		addr := common.HexToAddress(b.Address)
		return &addr
	}
	if p, ok := pinnedMap[name]; ok {
		addr := common.HexToAddress(p.Address)
		return &addr
	}
	return nil
}

// resolveSetupFrom 解析初始化交易的发送方地址。
// 支持直接传入 hex 地址、账户名称或空值（回退到 coinbase）。
// 对于 impersonated 地址会自动执行 hardhat_impersonateAccount 并充值。
func resolveSetupFrom(ctx context.Context, rpcClient *rpc.Client, from string, accountMap map[string]int, chainAccounts []common.Address, impersonatedSet map[string]bool) (common.Address, error) {
	from = strings.TrimSpace(from)
	if from == "" {
		if len(chainAccounts) > 0 {
			return chainAccounts[0], nil
		}
		return getCoinbase(ctx, rpcClient)
	}
	if common.IsHexAddress(from) {
		addr := common.HexToAddress(from)
		if impersonatedSet[strings.ToLower(from)] {
			_ = impersonateAndFund(ctx, rpcClient, addr)
		}
		return addr, nil
	}
	if idx, ok := accountMap[from]; ok && idx < len(chainAccounts) {
		return chainAccounts[idx], nil
	}
	if len(chainAccounts) > 0 {
		return chainAccounts[0], nil
	}
	return getCoinbase(ctx, rpcClient)
}

// impersonateAndFund 在 Hardhat/Ganache 链上冒充账户并充值，用于初始化交易的账户冒充场景。
func impersonateAndFund(ctx context.Context, rpcClient *rpc.Client, addr common.Address) error {
	var result interface{}
	_ = rpcClient.CallContext(ctx, &result, "hardhat_impersonateAccount", addr)
	_ = rpcClient.CallContext(ctx, &result, "hardhat_setBalance", addr, "0x3635C9ADC5DEA00000")
	return nil
}

// resolveContractArgs 将参数列表中的合约名称替换为实际部署地址。
func resolveContractArgs(args []interface{}, contractMap map[string]*ContractBinding, pinnedMap map[string]*PinnedContract) []interface{} {
	resolved := make([]interface{}, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			s = strings.TrimSpace(s)
			if b, exists := contractMap[s]; exists {
				resolved[i] = b.Address
				continue
			}
			if p, exists := pinnedMap[s]; exists {
				resolved[i] = p.Address
				continue
			}
		}
		resolved[i] = arg
	}
	return resolved
}

// isHex 判断字符串是否为有效的十六进制数据（可带 0x 前缀）。
func isHex(s string) bool {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// parseValue 将交易值字符串解析为 wei 单位的 big.Int。
// 支持纯数字、hex、以及带单位后缀（ether/gwei/wei）的格式。
func parseValue(raw string) *big.Int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return big.NewInt(0)
	}
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, " ether") {
		v := strings.TrimSpace(raw[:len(raw)-6])
		return ethToWei(v)
	}
	if strings.HasSuffix(lower, " gwei") {
		v := strings.TrimSpace(raw[:len(raw)-5])
		return gweiToWei(v)
	}
	if strings.HasSuffix(lower, " wei") {
		v := strings.TrimSpace(raw[:len(raw)-4])
		n, _ := new(big.Int).SetString(v, 10)
		if n == nil {
			return big.NewInt(0)
		}
		return n
	}
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		n, _ := new(big.Int).SetString(raw[2:], 16)
		if n == nil {
			return big.NewInt(0)
		}
		return n
	}
	n, _ := new(big.Int).SetString(raw, 10)
	if n == nil {
		return ethToWei(raw)
	}
	return n
}

// ethToWei 将 ether 单位的字符串转换为 wei。
func ethToWei(s string) *big.Int {
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return big.NewInt(0)
	}
	wei := new(big.Float).Mul(f, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
	result, _ := wei.Int(nil)
	return result
}

// gweiToWei 将 gwei 单位的字符串转换为 wei。
func gweiToWei(s string) *big.Int {
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return big.NewInt(0)
	}
	wei := new(big.Float).Mul(f, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)))
	result, _ := wei.Int(nil)
	return result
}

// packArgs 根据 ABI 参数类型将 JSON 动态值编码为 ABI 字节。
func packArgs(inputs abi.Arguments, args []interface{}) ([]byte, error) {
	converted := make([]interface{}, len(inputs))
	for i, input := range inputs {
		var raw interface{}
		if i < len(args) {
			raw = args[i]
		}
		val, err := convertToABIType(raw, input.Type)
		if err != nil {
			return nil, fmt.Errorf("参数 %d (%s): %w", i, input.Name, err)
		}
		converted[i] = val
	}
	return inputs.Pack(converted...)
}

// convertToABIType 将 JSON 动态值转换为对应 ABI 类型的 Go 值。
func convertToABIType(raw interface{}, abiType abi.Type) (interface{}, error) {
	switch abiType.T {
	case abi.UintTy, abi.IntTy:
		return toBigInt(raw)
	case abi.AddressTy:
		return toAddress(raw)
	case abi.BoolTy:
		return toBool(raw)
	case abi.StringTy:
		return toString(raw)
	case abi.BytesTy:
		return toBytes(raw)
	case abi.FixedBytesTy:
		return toFixedBytes(raw, abiType.Size)
	default:
		return raw, nil
	}
}

// toBigInt 将动态值转换为 big.Int，支持 float64、string、json.Number。
func toBigInt(raw interface{}) (*big.Int, error) {
	switch v := raw.(type) {
	case float64:
		return new(big.Int).SetInt64(int64(v)), nil
	case string:
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
			n, ok := new(big.Int).SetString(v[2:], 16)
			if !ok {
				return nil, fmt.Errorf("无法解析 hex 整数: %s", v)
			}
			return n, nil
		}
		n, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("无法解析整数: %s", v)
		}
		return n, nil
	case json.Number:
		n, ok := new(big.Int).SetString(string(v), 10)
		if !ok {
			return nil, fmt.Errorf("无法解析整数: %s", v)
		}
		return n, nil
	case nil:
		return big.NewInt(0), nil
	default:
		return nil, fmt.Errorf("unsupported type for bigint: %T", raw)
	}
}

// toAddress 将字符串转换为以太坊地址。
func toAddress(raw interface{}) (common.Address, error) {
	s, ok := raw.(string)
	if !ok {
		return common.Address{}, fmt.Errorf("address must be string, got %T", raw)
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("invalid address: %s", s)
	}
	return common.HexToAddress(s), nil
}

// toBool 将动态值转换为布尔值。
func toBool(raw interface{}) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		return strings.ToLower(v) == "true", nil
	default:
		return false, fmt.Errorf("unsupported type for bool: %T", raw)
	}
}

// toString 将动态值转换为字符串。
func toString(raw interface{}) (string, error) {
	if raw == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", raw), nil
}

// toBytes 将 hex 字符串转换为字节切片。
func toBytes(raw interface{}) ([]byte, error) {
	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("bytes must be hex string, got %T", raw)
	}
	return common.FromHex(s), nil
}

// toFixedBytes 将 hex 字符串转换为固定长度字节数组（bytes32/bytes20）。
func toFixedBytes(raw interface{}, size int) (interface{}, error) {
	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("fixed bytes must be hex string, got %T", raw)
	}
	b := common.FromHex(s)
	switch size {
	case 32:
		var arr [32]byte
		copy(arr[:], b)
		return arr, nil
	case 20:
		var arr [20]byte
		copy(arr[:], b)
		return arr, nil
	default:
		var arr [32]byte
		copy(arr[:], b)
		return arr, nil
	}
}
