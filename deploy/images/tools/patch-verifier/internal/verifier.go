// verifier.go — patch-verifier 核心验证引擎。
// 实现 Solidity 补丁编译、ABI 兼容性校验、补丁合约部署和官方 PoC 回放，
// 为 CTF 攻防赛补丁提交提供完整的漏洞修复验证能力。
// 断言验证支持 balance_check、storage_check、owner_check、code_check、event_check。
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// VerifyPatch 编译补丁、校验 ABI 兼容性、部署补丁合约并回放官方 PoC。
func VerifyPatch(ctx context.Context, cfg *Config, req *VerifyRequest) (*VerifyResponse, error) {
	// 1. 编译补丁源码
	compiled, err := compileSolidity(ctx, cfg.SolcPath, req.PatchSourceCode)
	if err != nil {
		return &VerifyResponse{
			FunctionalityPassed: false,
			VulnerabilityFixed:  false,
			RejectionReason:     stringPtr("补丁编译失败: " + err.Error()),
			PatchedContracts:    []ContractBinding{},
		}, nil
	}
	if len(compiled) == 0 {
		return &VerifyResponse{
			FunctionalityPassed: false,
			VulnerabilityFixed:  false,
			RejectionReason:     stringPtr("补丁编译结果为空"),
			PatchedContracts:    []ContractBinding{},
		}, nil
	}

	// 2. 匹配原始合约名称
	var original *ContractSpec
	if len(req.OriginalContracts) > 0 {
		original = &req.OriginalContracts[0]
	}
	originalName := ""
	if original != nil {
		originalName = original.ContractName
	}
	candidate := findBestMatch(compiled, originalName)

	// 3. ABI 兼容性校验
	if original != nil && original.ABIJSON != "" {
		oldSigs := extractPublicSignatures(original.ABIJSON)
		newSigs := extractPublicSignatures(candidate.ABIJSON)
		missing := findMissingSignatures(oldSigs, newSigs)
		if len(missing) > 0 {
			return &VerifyResponse{
				FunctionalityPassed: false,
				VulnerabilityFixed:  false,
				RejectionReason:     stringPtr("功能完整性检查未通过：缺少接口 " + strings.Join(missing, ", ")),
				PatchedContracts:    []ContractBinding{},
			}, nil
		}
	}

	// 4. 连接链并部署补丁合约
	rpcClient, err := rpc.DialContext(ctx, req.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("连接链 RPC 失败: %w", err)
	}
	client := ethclient.NewClient(rpcClient)
	defer client.Close()

	from, err := getCoinbase(ctx, rpcClient)
	if err != nil {
		return nil, fmt.Errorf("获取 coinbase 失败: %w", err)
	}

	patchedAddr, err := deployContract(ctx, client, rpcClient, from, candidate.Bytecode, nil)
	if err != nil {
		return &VerifyResponse{
			FunctionalityPassed: false,
			VulnerabilityFixed:  false,
			RejectionReason:     stringPtr("补丁合约部署失败: " + err.Error()),
			PatchedContracts:    []ContractBinding{},
		}, nil
	}

	// 5. 回放官方 PoC
	pocExecuted := false
	if strings.TrimSpace(req.OfficialPoc) != "" {
		_, err := sendTx(ctx, rpcClient, from, &patchedAddr, req.OfficialPoc, big.NewInt(0))
		if err == nil {
			pocExecuted = true
		}
	}

	// 6. 漏洞修复断言验证
	// 补丁验证的断言逻辑与攻击验证相反：断言条件仍然满足意味着漏洞未被修复。
	vulnerabilityFixed := true
	for _, assertion := range req.Assertions {
		assertType := strings.ToLower(strings.TrimSpace(assertion.AssertionType))
		op := strings.ToLower(strings.TrimSpace(assertion.Operator))
		if op == "" {
			op = "eq"
		}
		expected := parseExpected(assertion.ExpectedValue)

		switch assertType {
		case "balance_check":
			balance, err := client.BalanceAt(ctx, patchedAddr, nil)
			if err != nil {
				continue
			}
			// 余额仍满足漏洞条件 → 漏洞未修复
			if compareBigInt(balance, expected, op) {
				vulnerabilityFixed = false
			}

		case "storage_check":
			slot := getExtraString(assertion.ExtraParams, "slot")
			if slot == "" {
				slot = "0x0"
			}
			value, err := client.StorageAt(ctx, patchedAddr, common.HexToHash(slot), nil)
			if err != nil {
				continue
			}
			actualHex := strings.ToLower("0x" + common.Bytes2Hex(value))
			expectedHex := strings.ToLower(strings.TrimSpace(assertion.ExpectedValue))
			if compareString(actualHex, expectedHex, op) {
				vulnerabilityFixed = false
			}

		case "owner_check":
			// 查找对应的原始合约 ABI 用于调用 owner()
			abiJSON := findOriginalABI(req.OriginalContracts, candidate.ContractName, originalName)
			if abiJSON == "" {
				continue
			}
			fnName := getExtraString(assertion.ExtraParams, "function")
			if fnName == "" {
				fnName = "owner"
			}
			ownerAddr, err := callAddress(ctx, client, patchedAddr, abiJSON, fnName)
			if err != nil {
				continue
			}
			actualAddr := strings.ToLower(ownerAddr.Hex())
			expectedAddr := strings.ToLower(strings.TrimSpace(assertion.ExpectedValue))
			if compareString(actualAddr, expectedAddr, op) {
				vulnerabilityFixed = false
			}

		case "code_check":
			code, err := client.CodeAt(ctx, patchedAddr, nil)
			if err != nil {
				continue
			}
			codeSize := big.NewInt(int64(len(code)))
			if compareBigInt(codeSize, expected, op) {
				vulnerabilityFixed = false
			}

		case "event_check":
			// PoC 成功执行意味着漏洞未修复
			if pocExecuted {
				vulnerabilityFixed = false
			}
		}
	}

	// 7. 构建补丁合约结果
	patchedContracts := make([]ContractBinding, 0, len(req.TargetContracts))
	for _, tc := range req.TargetContracts {
		if tc.ContractName == candidate.ContractName || tc.ContractName == originalName {
			patchedContracts = append(patchedContracts, ContractBinding{
				ChallengeID:  tc.ChallengeID,
				ContractName: tc.ContractName,
				Address:      patchedAddr.Hex(),
				PatchVersion: tc.PatchVersion + 1,
				IsPatched:    true,
			})
		} else {
			patchedContracts = append(patchedContracts, tc)
		}
	}

	rejectionReason := (*string)(nil)
	if !vulnerabilityFixed {
		rejectionReason = stringPtr("漏洞修复验证失败：官方PoC仍可成功执行")
	}

	return &VerifyResponse{
		FunctionalityPassed: true,
		VulnerabilityFixed:  vulnerabilityFixed,
		RejectionReason:     rejectionReason,
		PatchedContracts:    patchedContracts,
	}, nil
}

// ── Solidity 编译 ──────────────────────────────────────────────

// compiledContract 表示 solc 编译后的单个合约产物。
type compiledContract struct {
	ContractName string
	ABIJSON      string
	Bytecode     string
}

// compileSolidity 调用 solc 编译器编译 Solidity 源码，返回编译后的合约列表。
// 使用临时目录写入源文件，通过 --combined-json 获取 ABI 和字节码。
func compileSolidity(ctx context.Context, solcPath, source string) ([]compiledContract, error) {
	tmpDir, err := os.MkdirTemp("", "patch-verify-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "Patch.sol")
	if err := os.WriteFile(srcFile, []byte(source), 0644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, solcPath,
		"--combined-json", "abi,bin",
		"--optimize",
		"--optimize-runs", "200",
		"--base-path", tmpDir,
		"--allow-paths", tmpDir,
		srcFile,
	)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var result struct {
		Contracts map[string]struct {
			ABI json.RawMessage `json:"abi"`
			Bin string          `json:"bin"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析 solc 输出失败: %w", err)
	}

	contracts := make([]compiledContract, 0, len(result.Contracts))
	for key, val := range result.Contracts {
		if val.Bin == "" {
			continue
		}
		parts := strings.Split(key, ":")
		name := parts[len(parts)-1]
		abiStr := string(val.ABI)
		if abiStr == "" {
			abiStr = "[]"
		}
		contracts = append(contracts, compiledContract{
			ContractName: name,
			ABIJSON:      abiStr,
			Bytecode:     val.Bin,
		})
	}
	return contracts, nil
}

// ── ABI 兼容性 ─────────────────────────────────────────────────

// extractPublicSignatures 从 ABI JSON 中提取所有公共方法签名。
// 返回排序后的签名列表，格式为 "methodName(type1,type2)"。
func extractPublicSignatures(abiJSON string) []string {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil
	}
	sigs := make([]string, 0, len(parsedABI.Methods))
	for _, method := range parsedABI.Methods {
		sig := method.Name + "("
		inputs := make([]string, len(method.Inputs))
		for i, input := range method.Inputs {
			inputs[i] = input.Type.String()
		}
		sig += strings.Join(inputs, ",") + ")"
		sigs = append(sigs, sig)
	}
	sort.Strings(sigs)
	return sigs
}

// findMissingSignatures 查找旧 ABI 中存在但新 ABI 中缺失的方法签名。
func findMissingSignatures(oldSigs, newSigs []string) []string {
	newSet := make(map[string]bool, len(newSigs))
	for _, s := range newSigs {
		newSet[s] = true
	}
	var missing []string
	for _, s := range oldSigs {
		if !newSet[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

// findBestMatch 从编译结果中找到与原始合约名称最匹配的合约。
// 若未匹配到则返回第一个编译结果。
func findBestMatch(compiled []compiledContract, originalName string) *compiledContract {
	if originalName != "" {
		for i := range compiled {
			if compiled[i].ContractName == originalName {
				return &compiled[i]
			}
		}
	}
	return &compiled[0]
}

// ── 链交互 ─────────────────────────────────────────────────────

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

// deployContract 在链上部署合约并等待回执，返回合约地址。
// constructorData 为可选的已编码构造参数。
func deployContract(ctx context.Context, client *ethclient.Client, rpcClient *rpc.Client, from common.Address, bytecodeHex string, constructorData []byte) (common.Address, error) {
	if !strings.HasPrefix(bytecodeHex, "0x") {
		bytecodeHex = "0x" + bytecodeHex
	}
	data := common.FromHex(bytecodeHex)
	if len(constructorData) > 0 {
		data = append(data, constructorData...)
	}
	tx := map[string]interface{}{
		"from": from.Hex(),
		"data": hexutil.Encode(data),
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

// sendTx 发送交易到链上，若 to 为 nil 则为合约创建交易。
func sendTx(ctx context.Context, rpcClient *rpc.Client, from common.Address, to *common.Address, dataHex string, value *big.Int) (common.Hash, error) {
	if !strings.HasPrefix(dataHex, "0x") {
		dataHex = "0x" + dataHex
	}
	tx := map[string]interface{}{
		"from":  from.Hex(),
		"data":  dataHex,
		"gas":   hexutil.EncodeUint64(12_000_000),
		"value": hexutil.EncodeBig(value),
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

// ── 比较辅助 ───────────────────────────────────────────────────

// parseExpected 将断言期望值字符串解析为 big.Int。
// 支持纯数字、hex、以及带单位后缀（ether/gwei/wei）的格式。
func parseExpected(s string) *big.Int {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0)
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, " ether") {
		return ethToWei(strings.TrimSpace(s[:len(s)-6]))
	}
	if strings.HasSuffix(lower, " gwei") {
		return gweiToWei(strings.TrimSpace(s[:len(s)-5]))
	}
	if strings.HasSuffix(lower, " wei") {
		n, _ := new(big.Int).SetString(strings.TrimSpace(s[:len(s)-4]), 10)
		if n != nil {
			return n
		}
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, _ := new(big.Int).SetString(s[2:], 16)
		if n != nil {
			return n
		}
	}
	n, _ := new(big.Int).SetString(s, 10)
	if n != nil {
		return n
	}
	return big.NewInt(0)
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

// compareBigInt 根据运算符比较两个 big.Int 值。
// 支持 eq（默认）、lt、le、gt、ge、ne 六种运算符。
func compareBigInt(actual, expected *big.Int, operator string) bool {
	if actual == nil {
		actual = big.NewInt(0)
	}
	if expected == nil {
		expected = big.NewInt(0)
	}
	cmp := actual.Cmp(expected)
	switch operator {
	case "lt":
		return cmp < 0
	case "le":
		return cmp <= 0
	case "gt":
		return cmp > 0
	case "ge":
		return cmp >= 0
	case "ne":
		return cmp != 0
	default:
		return cmp == 0
	}
}

// compareString 根据运算符比较两个字符串。
// 支持 eq（默认）、ne、contains 三种运算符。
func compareString(actual, expected, operator string) bool {
	switch operator {
	case "contains":
		return strings.Contains(actual, expected)
	case "ne":
		return actual != expected
	default:
		return actual == expected
	}
}

// getExtraString 从断言的 extra_params 中按优先级顺序查找字符串值。
// 支持传入多个候选 key，返回第一个非空匹配。
func getExtraString(extra map[string]interface{}, keys ...string) string {
	if extra == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// findOriginalABI 在原始合约列表中按名称查找 ABI JSON。
// 用于 owner_check 等需要调用合约方法的断言。
func findOriginalABI(originals []ContractSpec, candidateName, originalName string) string {
	for _, c := range originals {
		if c.ContractName == candidateName || c.ContractName == originalName {
			return c.ABIJSON
		}
	}
	return ""
}

// callAddress 调用合约的只读方法并返回 address 结果。
// 适用于 owner()、admin() 等返回单一地址的方法。
func callAddress(ctx context.Context, client *ethclient.Client, addr common.Address, abiJSON, method string) (common.Address, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return common.Address{}, fmt.Errorf("解析 ABI 失败: %w", err)
	}
	m, ok := parsedABI.Methods[method]
	if !ok {
		return common.Address{}, fmt.Errorf("ABI 中找不到方法 %s", method)
	}
	data := m.ID
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return common.Address{}, err
	}
	if len(result) < 32 {
		return common.Address{}, nil
	}
	return common.BytesToAddress(result[12:32]), nil
}
