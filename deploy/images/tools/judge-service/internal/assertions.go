// assertions.go — judge-service 断言评估引擎。
// 支持 balance_check、token_balance_check、storage_check、owner_check、
// event_check、code_check、custom_script 七种断言类型，为 CTF 攻防赛攻击验证提供判定依据。
// custom_script 通过 goja（纯 Go ECMAScript 引擎）在沙箱中执行自定义验证脚本。
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// EvaluateAssertion 执行单个断言并返回结果。
func EvaluateAssertion(
	ctx context.Context,
	client *ethclient.Client,
	rpcClient *rpc.Client,
	receipt *types.Receipt,
	assertion *AssertionSpec,
	contractMap map[string]*ContractBinding,
) (*AssertionResult, error) {
	target := resolveAssertionTarget(assertion.Target, contractMap)
	expected := parseExpectedValue(assertion.ExpectedValue)
	extra := assertion.ExtraParams
	if extra == nil {
		extra = map[string]interface{}{}
	}
	operator := strings.ToLower(strings.TrimSpace(assertion.Operator))
	if operator == "" {
		operator = "eq"
	}

	var actual interface{}
	var passed bool

	switch strings.ToLower(strings.TrimSpace(assertion.AssertionType)) {
	case "balance_check":
		if target == nil {
			return nil, fmt.Errorf("balance_check 缺少目标合约")
		}
		balance, err := client.BalanceAt(ctx, *target, nil)
		if err != nil {
			return nil, err
		}
		actual = balance
		passed = compareBigInt(balance, expected, operator)

	case "token_balance_check":
		if target == nil {
			return nil, fmt.Errorf("token_balance_check 缺少代币合约")
		}
		binding := findInContractMap(*target, contractMap)
		if binding == nil || binding.ABIJSON == "" {
			return nil, fmt.Errorf("token_balance_check 缺少代币合约 ABI")
		}
		owner := getExtraString(extra, "owner", "account", "holder")
		if owner == "" {
			// 未指定 owner，回退到链上 coinbase 账户
			owner = getFirstAccount(ctx, rpcClient)
		}
		balance, err := callUint256(ctx, client, *target, binding.ABIJSON, "balanceOf", common.HexToAddress(owner))
		if err != nil {
			return nil, err
		}
		actual = balance
		passed = compareBigInt(balance, expected, operator)

	case "storage_check":
		if target == nil {
			return nil, fmt.Errorf("storage_check 缺少目标合约")
		}
		slot := getExtraString(extra, "slot")
		if slot == "" {
			slot = "0x0"
		}
		slotHash := common.HexToHash(slot)
		value, err := client.StorageAt(ctx, *target, slotHash, nil)
		if err != nil {
			return nil, err
		}
		valueHex := common.Bytes2Hex(value)
		actual = "0x" + valueHex
		expectedStr := strings.ToLower(strings.TrimSpace(assertion.ExpectedValue))
		actualStr := strings.ToLower("0x" + valueHex)
		passed = compareString(actualStr, expectedStr, operator)

	case "owner_check":
		if target == nil {
			return nil, fmt.Errorf("owner_check 缺少目标合约")
		}
		binding := findInContractMap(*target, contractMap)
		if binding == nil || binding.ABIJSON == "" {
			return nil, fmt.Errorf("owner_check 缺少合约 ABI")
		}
		fnName := getExtraString(extra, "function")
		if fnName == "" {
			fnName = "owner"
		}
		ownerAddr, err := callAddress(ctx, client, *target, binding.ABIJSON, fnName)
		if err != nil {
			return nil, err
		}
		actual = strings.ToLower(ownerAddr.Hex())
		expectedAddr := strings.ToLower(strings.TrimSpace(assertion.ExpectedValue))
		passed = compareString(actual.(string), expectedAddr, operator)

	case "event_check":
		var targetAddr string
		if target != nil {
			targetAddr = strings.ToLower(target.Hex())
		}
		topic0 := strings.ToLower(getExtraString(extra, "topic0", "event_signature"))
		logCount := countMatchingLogs(receipt, targetAddr, topic0)
		actual = big.NewInt(int64(logCount))
		if operator == "" {
			operator = "gt"
		}
		passed = compareBigInt(big.NewInt(int64(logCount)), expected, operator)

	case "code_check":
		if target == nil {
			return nil, fmt.Errorf("code_check 缺少目标合约")
		}
		code, err := client.CodeAt(ctx, *target, nil)
		if err != nil {
			return nil, err
		}
		codeSize := big.NewInt(int64(len(code)))
		actual = codeSize
		passed = compareBigInt(codeSize, expected, operator)

	case "custom_script":
		script := getExtraString(extra, "script", "code")
		if script == "" {
			return nil, fmt.Errorf("custom_script 缺少 script 参数")
		}
		scriptResult, err := executeCustomScript(ctx, client, script, contractMap, receipt)
		if err != nil {
			return nil, fmt.Errorf("custom_script 执行失败: %w", err)
		}
		actual = scriptResult.Actual
		passed = scriptResult.Passed

	default:
		return nil, fmt.Errorf("不支持的断言类型: %s", assertion.AssertionType)
	}

	return &AssertionResult{
		Type:     assertion.AssertionType,
		Target:   assertion.Target,
		Expected: assertion.ExpectedValue,
		Actual:   formatActual(actual),
		Passed:   passed,
	}, nil
}

// ── 合约调用 ───────────────────────────────────────────────────

// callUint256 调用合约的只读方法并返回 uint256 结果。
// 适用于 balanceOf、totalSupply 等返回单一 uint256 的方法。
func callUint256(ctx context.Context, client *ethclient.Client, addr common.Address, abiJSON, method string, args ...interface{}) (*big.Int, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	m, ok := parsedABI.Methods[method]
	if !ok {
		return nil, fmt.Errorf("ABI 中找不到方法 %s", method)
	}
	packed, err := packArgs(m.Inputs, args)
	if err != nil {
		return nil, err
	}
	data := append(m.ID, packed...)
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	if len(result) < 32 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(result[:32]), nil
}

// callAddress 调用合约的只读方法并返回 address 结果。
// 适用于 owner()、admin() 等返回单一地址的方法。
func callAddress(ctx context.Context, client *ethclient.Client, addr common.Address, abiJSON, method string) (common.Address, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return common.Address{}, err
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

// getFirstAccount 通过 eth_accounts RPC 获取链上第一个可用账户地址。
// 用于 token_balance_check 在未指定 owner 时作为默认持有者。
func getFirstAccount(ctx context.Context, rpcClient *rpc.Client) string {
	if rpcClient == nil {
		return ""
	}
	var accounts []common.Address
	if err := rpcClient.CallContext(ctx, &accounts, "eth_accounts"); err != nil || len(accounts) == 0 {
		return ""
	}
	return accounts[0].Hex()
}

// ── 比较与格式化 ───────────────────────────────────────────────

// parseExpectedValue 将断言期望值字符串解析为 big.Int。
// 支持纯数字、hex、以及带单位后缀（ether/gwei/wei）的格式。
func parseExpectedValue(s string) *big.Int {
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
	default: // eq
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

// formatActual 将断言实际值格式化为字符串用于响应输出。
func formatActual(v interface{}) string {
	switch val := v.(type) {
	case *big.Int:
		return val.String()
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ── 日志匹配 ───────────────────────────────────────────────────

// countMatchingLogs 统计交易回执中匹配目标地址和 topic0 的日志条数。
// targetAddr 和 topic0 为空时表示不过滤对应条件。
func countMatchingLogs(receipt *types.Receipt, targetAddr, topic0 string) int {
	if receipt == nil {
		return 0
	}
	count := 0
	for _, log := range receipt.Logs {
		if targetAddr != "" && strings.ToLower(log.Address.Hex()) != targetAddr {
			continue
		}
		if topic0 != "" {
			if len(log.Topics) == 0 || strings.ToLower(log.Topics[0].Hex()) != topic0 {
				continue
			}
		}
		count++
	}
	return count
}

// ── 目标解析 ───────────────────────────────────────────────────

// resolveAssertionTarget 将断言目标解析为链上地址。
// 支持直接传入 hex 地址或通过合约名称从 contractMap 查找。
func resolveAssertionTarget(target string, contractMap map[string]*ContractBinding) *common.Address {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	if common.IsHexAddress(target) {
		addr := common.HexToAddress(target)
		return &addr
	}
	if b, ok := contractMap[target]; ok {
		addr := common.HexToAddress(b.Address)
		return &addr
	}
	return nil
}

// findInContractMap 在合约映射中按地址查找合约绑定信息。
func findInContractMap(addr common.Address, contractMap map[string]*ContractBinding) *ContractBinding {
	addrLower := strings.ToLower(addr.Hex())
	for _, b := range contractMap {
		if strings.ToLower(b.Address) == addrLower {
			return b
		}
	}
	return nil
}

// getExtraString 从断言的 extra_params 中按优先级顺序查找字符串值。
// 支持传入多个候选 key，返回第一个非空匹配。
func getExtraString(extra map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := extra[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// ── custom_script 沙箱执行引擎 ─────────────────────────────────

// customScriptResult 自定义脚本的执行结果。
type customScriptResult struct {
	Passed bool
	Actual string
}

// customScriptTimeout 自定义脚本最大执行时长。
const customScriptTimeout = 10 * time.Second

// executeCustomScript 在 goja JS 沙箱中执行自定义验证脚本。
//
// 脚本可使用以下注入函数查询链上状态：
//   - getBalance(address) → string：查询账户余额（wei）
//   - getStorageAt(address, slot) → string：查询合约存储槽
//   - getCode(address) → string：查询合约字节码
//   - callContract(to, data) → string：执行只读合约调用
//   - getContractAddress(name) → string：按名称查找已部署合约地址
//   - getTxHash() → string：获取攻击交易的哈希
//   - getLogCount() → number：获取交易回执中的日志数量
//
// 脚本必须返回一个对象：{ passed: boolean, actual: string }
func executeCustomScript(
	ctx context.Context,
	client *ethclient.Client,
	script string,
	contractMap map[string]*ContractBinding,
	receipt *types.Receipt,
) (*customScriptResult, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	// 注入超时中断：启动定时器，超时后中断 VM 执行。
	timer := time.AfterFunc(customScriptTimeout, func() {
		vm.Interrupt("script execution timeout")
	})
	defer timer.Stop()

	// 注入 getBalance(address) → string
	_ = vm.Set("getBalance", func(call goja.FunctionCall) goja.Value {
		addr := call.Argument(0).String()
		if !common.IsHexAddress(addr) {
			return vm.ToValue("0")
		}
		balance, err := client.BalanceAt(ctx, common.HexToAddress(addr), nil)
		if err != nil {
			return vm.ToValue("0")
		}
		return vm.ToValue(balance.String())
	})

	// 注入 getStorageAt(address, slot) → string
	_ = vm.Set("getStorageAt", func(call goja.FunctionCall) goja.Value {
		addr := call.Argument(0).String()
		slot := call.Argument(1).String()
		if !common.IsHexAddress(addr) {
			return vm.ToValue("")
		}
		value, err := client.StorageAt(ctx, common.HexToAddress(addr), common.HexToHash(slot), nil)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue("0x" + common.Bytes2Hex(value))
	})

	// 注入 getCode(address) → string
	_ = vm.Set("getCode", func(call goja.FunctionCall) goja.Value {
		addr := call.Argument(0).String()
		if !common.IsHexAddress(addr) {
			return vm.ToValue("")
		}
		code, err := client.CodeAt(ctx, common.HexToAddress(addr), nil)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue("0x" + common.Bytes2Hex(code))
	})

	// 注入 callContract(to, data) → string
	_ = vm.Set("callContract", func(call goja.FunctionCall) goja.Value {
		toAddr := call.Argument(0).String()
		dataHex := call.Argument(1).String()
		if !common.IsHexAddress(toAddr) {
			return vm.ToValue("")
		}
		to := common.HexToAddress(toAddr)
		callData := common.FromHex(dataHex)
		result, err := client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: callData}, nil)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue("0x" + common.Bytes2Hex(result))
	})

	// 注入 getContractAddress(name) → string
	_ = vm.Set("getContractAddress", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if b, ok := contractMap[name]; ok {
			return vm.ToValue(b.Address)
		}
		return vm.ToValue("")
	})

	// 注入 getTxHash() → string
	_ = vm.Set("getTxHash", func(call goja.FunctionCall) goja.Value {
		if receipt != nil {
			return vm.ToValue(receipt.TxHash.Hex())
		}
		return vm.ToValue("")
	})

	// 注入 getLogCount() → number
	_ = vm.Set("getLogCount", func(call goja.FunctionCall) goja.Value {
		if receipt != nil {
			return vm.ToValue(len(receipt.Logs))
		}
		return vm.ToValue(0)
	})

	// 将脚本包装为立即执行函数以获取返回值
	wrappedScript := "(function() {\n" + script + "\n})()"
	val, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("脚本执行错误: %w", err)
	}

	// 解析返回值
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil, fmt.Errorf("脚本未返回结果，需返回 {passed: boolean, actual: string}")
	}

	obj := val.ToObject(vm)
	passedVal := obj.Get("passed")
	actualVal := obj.Get("actual")

	result := &customScriptResult{
		Passed: false,
		Actual: "",
	}
	if passedVal != nil && !goja.IsUndefined(passedVal) {
		result.Passed = passedVal.ToBoolean()
	}
	if actualVal != nil && !goja.IsUndefined(actualVal) {
		result.Actual = actualVal.String()
	}

	return result, nil
}
