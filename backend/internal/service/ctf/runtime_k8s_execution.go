// ctf_runtime_execution.go
// 模块05 — CTF竞赛：解题赛与攻防赛运行时执行器。
// 负责在 service/ctf 内部把题目环境编排、链上攻击验证和补丁验证
// 落到真实容器执行，供业务服务通过接口调用。

package ctf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
)

const (
	ctfChallengeRuntimePodName = "challenge-runtime"
	ctfChallengeChainContainer = "challenge-chain"
	ctfChallengeToolsContainer = "challenge-tools"
)

// ProvisionChallengeEnvironment 创建解题赛题目环境，并返回真实环境元数据。
func (a *RuntimeProvisionerAdapter) ProvisionChallengeEnvironment(ctx context.Context, spec *ChallengeEnvironmentSpec) (*ChallengeEnvironmentResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("题目环境规格不能为空")
	}
	labels := map[string]string{
		"module":         "ctf",
		"runtime":        "challenge-environment",
		"competition_id": fmt.Sprintf("%d", spec.CompetitionID),
		"challenge_id":   fmt.Sprintf("%d", spec.ChallengeID),
		"team_id":        fmt.Sprintf("%d", spec.TeamID),
	}
	if err := a.CreateNamespace(ctx, spec.Namespace, labels); err != nil {
		return nil, err
	}
	if spec.ChainConfig != nil || len(spec.Contracts) > 0 {
		return a.provisionOnChainChallengeEnvironment(ctx, spec)
	}
	return a.provisionContainerChallengeEnvironment(ctx, spec)
}

// ExecuteChallengeSubmission 在选手题目环境中执行链上攻击并校验断言。
func (a *RuntimeProvisionerAdapter) ExecuteChallengeSubmission(ctx context.Context, spec *ChallengeSubmissionSpec) (*ChallengeSubmissionResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("题目提交执行规格不能为空")
	}
	payload := map[string]interface{}{
		"rpc_url":        spec.ChainRPCURL,
		"submission":     spec.SubmissionData,
		"contracts":      spec.Contracts,
		"accounts":       spec.Accounts,
		"assertions":     spec.Assertions,
		"default_target": defaultRuntimeTarget(spec.Assertions, spec.Contracts),
	}
	result := struct {
		AllPassed       bool                              `json:"all_passed"`
		Results         []dto.VerificationAssertionResult `json:"results"`
		ExecutionTimeMS *int                              `json:"execution_time_ms,omitempty"`
		TxHash          *string                           `json:"tx_hash,omitempty"`
		ErrorMessage    *string                           `json:"error_message,omitempty"`
	}{}
	if err := a.executeJSONRuntime(ctx, spec.Namespace, ctfChallengeRuntimePodName, ctfChallengeToolsContainer, payload, runtimeExecutionScript, &result); err != nil {
		return nil, err
	}
	return &ChallengeSubmissionResult{
		IsCorrect: result.AllPassed,
		AssertionResults: &dto.VerificationAssertionResults{
			AllPassed:       result.AllPassed,
			Results:         result.Results,
			ExecutionTimeMS: result.ExecutionTimeMS,
			TxHash:          result.TxHash,
		},
		ErrorMessage: result.ErrorMessage,
	}, nil
}

// ExecuteADAttack 在目标队伍链执行攻击交易，并返回真实断言校验结果。
func (a *RuntimeProvisionerAdapter) ExecuteADAttack(ctx context.Context, spec *ADAttackExecutionSpec) (*ADAttackExecutionResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("攻防赛攻击执行规格不能为空")
	}
	payload := map[string]interface{}{
		"rpc_url":        spec.ChainRPCURL,
		"submission":     spec.AttackTxData,
		"contracts":      spec.Contracts,
		"assertions":     spec.Assertions,
		"default_target": defaultRuntimeTarget(spec.Assertions, nil),
	}
	result := struct {
		AllPassed       bool                              `json:"all_passed"`
		Results         []dto.VerificationAssertionResult `json:"results"`
		ExecutionTimeMS *int                              `json:"execution_time_ms,omitempty"`
		TxHash          *string                           `json:"tx_hash,omitempty"`
		ErrorMessage    *string                           `json:"error_message,omitempty"`
	}{}
	if err := a.executeJSONRuntime(ctx, spec.Namespace, ctfJudgeRuntimePodName, ctfJudgeToolsContainer, payload, runtimeExecutionScript, &result); err != nil {
		return nil, err
	}
	return &ADAttackExecutionResult{
		IsSuccessful: result.AllPassed,
		AssertionResults: &dto.VerificationAssertionResults{
			AllPassed:       result.AllPassed,
			Results:         result.Results,
			ExecutionTimeMS: result.ExecutionTimeMS,
			TxHash:          result.TxHash,
		},
		ErrorMessage: result.ErrorMessage,
	}, nil
}

// VerifyADPatch 在隔离运行时中编译补丁并回放官方 PoC。
func (a *RuntimeProvisionerAdapter) VerifyADPatch(ctx context.Context, spec *ADPatchVerificationSpec) (*ADPatchVerificationResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("补丁验证规格不能为空")
	}
	payload := map[string]interface{}{
		"rpc_url":            spec.ChainRPCURL,
		"challenge_id":       spec.ChallengeID,
		"challenge_title":    spec.ChallengeTitle,
		"patch_source_code":  spec.PatchSourceCode,
		"original_contracts": spec.OriginalContracts,
		"target_contracts":   spec.TargetContracts,
		"assertions":         spec.Assertions,
		"official_poc":       spec.OfficialPocContent,
	}
	result := struct {
		FunctionalityPassed bool                        `json:"functionality_passed"`
		VulnerabilityFixed  bool                        `json:"vulnerability_fixed"`
		RejectionReason     *string                     `json:"rejection_reason,omitempty"`
		PatchedContracts    []dto.TeamChainContractItem `json:"patched_contracts"`
	}{}
	if err := a.executeJSONRuntime(ctx, spec.Namespace, ctfJudgeRuntimePodName, ctfJudgeToolsContainer, payload, runtimePatchVerificationScript, &result); err != nil {
		return nil, err
	}
	return &ADPatchVerificationResult{
		FunctionalityPassed: result.FunctionalityPassed,
		VulnerabilityFixed:  result.VulnerabilityFixed,
		RejectionReason:     result.RejectionReason,
		PatchedContracts:    result.PatchedContracts,
	}, nil
}

// provisionOnChainChallengeEnvironment 创建智能合约题目的独立链环境。
func (a *RuntimeProvisionerAdapter) provisionOnChainChallengeEnvironment(ctx context.Context, spec *ChallengeEnvironmentSpec) (*ChallengeEnvironmentResult, error) {
	chainImage, chainCommand := resolveChallengeChainRuntime(spec)
	toolImage := ctfDefaultRuntimeImage
	_, err := a.k8sSvc.DeployPod(ctx, &RuntimeDeployPodRequest{
		Namespace: spec.Namespace,
		PodName:   ctfChallengeRuntimePodName,
		Labels: map[string]string{
			"module":  "ctf",
			"runtime": "challenge-chain",
		},
		Containers: []RuntimeContainerSpec{
			{
				Name:    ctfChallengeChainContainer,
				Image:   chainImage,
				Command: chainCommand,
				Ports: []RuntimePortSpec{
					{ContainerPort: ctfTeamChainRPCPort, Protocol: "TCP", ServicePort: ctfTeamChainRPCPort},
				},
			},
			{
				Name:  ctfChallengeToolsContainer,
				Image: toolImage,
				Command: []string{
					"/bin/sh",
					"-c",
					"while true; do sleep 3600; done",
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := a.waitForPodRunning(ctx, spec.Namespace, ctfChallengeRuntimePodName); err != nil {
		return nil, err
	}
	rpcURL := buildClusterServiceURL(ctfChallengeChainContainer, spec.Namespace, ctfTeamChainRPCPort, false)
	bindings, err := a.deployChallengeContracts(ctx, spec.Namespace, ctfChallengeRuntimePodName, ctfChallengeToolsContainer, rpcURL, spec.Contracts)
	if err != nil {
		return nil, err
	}
	if err := a.executeChallengeSetupTransactions(ctx, spec.Namespace, ctfChallengeRuntimePodName, ctfChallengeToolsContainer, rpcURL, spec, bindings); err != nil {
		return nil, err
	}
	return &ChallengeEnvironmentResult{
		ChainRPCURL: stringPtr(rpcURL),
		ContainerStatus: map[string]dto.ChallengeEnvironmentContainerState{
			ctfChallengeChainContainer: {Status: "running", Image: chainImage},
			ctfChallengeToolsContainer: {Status: "running", Image: toolImage},
		},
		Contracts: bindings,
	}, nil
}

// resolveChallengeChainRuntime 根据题目运行时模式选择链镜像与启动命令。
func resolveChallengeChainRuntime(spec *ChallengeEnvironmentSpec) (string, []string) {
	if spec != nil && spec.RuntimeMode == enum.RuntimeModeForked && spec.ChainConfig != nil && spec.ChainConfig.Fork != nil {
		forkCfg := spec.ChainConfig.Fork
		command := []string{
			"npx", "hardhat", "node",
			"--hostname", "0.0.0.0",
			"--port", fmt.Sprintf("%d", ctfTeamChainRPCPort),
			"--fork", strings.TrimSpace(forkCfg.RPCURL),
			"--fork-block-number", fmt.Sprintf("%d", forkCfg.BlockNumber),
		}
		return ctfDefaultForkChainImage, command
	}
	return ctfDefaultTeamChainImage, nil
}

// executeChallengeSetupTransactions 在题目环境完成部署后回放初始化交易。
func (a *RuntimeProvisionerAdapter) executeChallengeSetupTransactions(
	ctx context.Context,
	namespace, podName, container, rpcURL string,
	spec *ChallengeEnvironmentSpec,
	bindings []ChallengeRuntimeContractBinding,
) error {
	if spec == nil || len(spec.SetupTransactions) == 0 {
		return nil
	}
	payload := map[string]interface{}{
		"rpc_url":               rpcURL,
		"runtime_mode":          spec.RuntimeMode,
		"accounts":              nil,
		"contracts":             bindings,
		"setup_transactions":    spec.SetupTransactions,
		"impersonated_accounts": []string{},
		"pinned_contracts":      []dto.ChallengePinnedContract{},
	}
	if spec.ChainConfig != nil {
		payload["accounts"] = spec.ChainConfig.Accounts
		if spec.ChainConfig.Fork != nil {
			payload["impersonated_accounts"] = spec.ChainConfig.Fork.ImpersonatedAccounts
			payload["pinned_contracts"] = spec.ChainConfig.Fork.PinnedContracts
		}
	}
	result := struct {
		Applied int `json:"applied"`
	}{}
	return a.executeJSONRuntime(ctx, namespace, podName, container, payload, runtimeSetupScript, &result)
}

// provisionContainerChallengeEnvironment 创建非合约题目的普通容器环境。
func (a *RuntimeProvisionerAdapter) provisionContainerChallengeEnvironment(ctx context.Context, spec *ChallengeEnvironmentSpec) (*ChallengeEnvironmentResult, error) {
	if spec.EnvironmentConfig == nil || len(spec.EnvironmentConfig.Images) == 0 {
		return &ChallengeEnvironmentResult{
			ContainerStatus: map[string]dto.ChallengeEnvironmentContainerState{},
		}, nil
	}
	containers := make([]RuntimeContainerSpec, 0, len(spec.EnvironmentConfig.Images))
	containerStatus := make(map[string]dto.ChallengeEnvironmentContainerState, len(spec.EnvironmentConfig.Images))
	for idx, imageCfg := range spec.EnvironmentConfig.Images {
		containerName := fmt.Sprintf("challenge-app-%d", idx+1)
		imageRef := strings.TrimSpace(imageCfg.Image)
		if version := strings.TrimSpace(imageCfg.Version); version != "" && !strings.Contains(imageRef, ":") {
			imageRef = imageRef + ":" + version
		}
		portSpecs := make([]RuntimePortSpec, 0, len(imageCfg.Ports))
		for _, port := range imageCfg.Ports {
			portSpecs = append(portSpecs, RuntimePortSpec{
				ContainerPort: port.Port,
				ServicePort:   port.Port,
				Protocol:      strings.ToUpper(strings.TrimSpace(port.Protocol)),
			})
		}
		envVars := make(map[string]string, len(imageCfg.EnvVars))
		for _, envVar := range imageCfg.EnvVars {
			envVars[envVar.Key] = envVar.Value
		}
		containers = append(containers, RuntimeContainerSpec{
			Name:        containerName,
			Image:       imageRef,
			Ports:       portSpecs,
			EnvVars:     envVars,
			CPULimit:    imageCfg.CPULimit,
			MemoryLimit: imageCfg.MemoryLimit,
		})
		containerStatus[containerName] = dto.ChallengeEnvironmentContainerState{
			Status: "running",
			Image:  imageRef,
		}
	}
	_, err := a.k8sSvc.DeployPod(ctx, &RuntimeDeployPodRequest{
		Namespace:  spec.Namespace,
		PodName:    ctfChallengeRuntimePodName,
		Labels:     map[string]string{"module": "ctf", "runtime": "challenge-container"},
		Containers: containers,
	})
	if err != nil {
		return nil, err
	}
	if err := a.waitForPodRunning(ctx, spec.Namespace, ctfChallengeRuntimePodName); err != nil {
		return nil, err
	}
	if spec.EnvironmentConfig.InitScript != nil && strings.TrimSpace(*spec.EnvironmentConfig.InitScript) != "" && len(containers) > 0 {
		if _, err := a.k8sSvc.ExecInPod(ctx, spec.Namespace, ctfChallengeRuntimePodName, containers[0].Name, *spec.EnvironmentConfig.InitScript); err != nil {
			return nil, err
		}
	}
	return &ChallengeEnvironmentResult{
		ContainerStatus: containerStatus,
		Contracts:       []ChallengeRuntimeContractBinding{},
	}, nil
}

// deployChallengeContracts 在题目环境工具容器中部署漏洞合约，并返回部署绑定信息。
func (a *RuntimeProvisionerAdapter) deployChallengeContracts(
	ctx context.Context,
	namespace, podName, container, rpcURL string,
	contracts []ChallengeRuntimeContractSpec,
) ([]ChallengeRuntimeContractBinding, error) {
	if len(contracts) == 0 {
		return []ChallengeRuntimeContractBinding{}, nil
	}
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].DeployOrder == contracts[j].DeployOrder {
			return contracts[i].ContractName < contracts[j].ContractName
		}
		return contracts[i].DeployOrder < contracts[j].DeployOrder
	})
	payload := map[string]interface{}{
		"rpc_url":   rpcURL,
		"contracts": contracts,
	}
	result := struct {
		Contracts []ChallengeRuntimeContractBinding `json:"contracts"`
	}{}
	if err := a.executeJSONRuntime(ctx, namespace, podName, container, payload, runtimeDeployScript, &result); err != nil {
		return nil, err
	}
	return result.Contracts, nil
}

// executeJSONRuntime 在指定容器中执行 Node 脚本，并把 JSON 输出解析到目标结构。
func (a *RuntimeProvisionerAdapter) executeJSONRuntime(
	ctx context.Context,
	namespace, podName, container string,
	payload map[string]interface{},
	script string,
	target interface{},
) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	command := fmt.Sprintf(
		"PAYLOAD_BASE64='%s' node <<'NODE'\n%s\nNODE",
		base64.StdEncoding.EncodeToString(payloadBytes),
		script,
	)
	result, err := a.k8sSvc.ExecInPod(ctx, namespace, podName, container, command)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		errText := strings.TrimSpace(result.Stderr)
		if errText == "" {
			errText = strings.TrimSpace(result.Stdout)
		}
		return fmt.Errorf("运行时执行失败: %s", errText)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), target); err != nil {
		return fmt.Errorf("解析运行时执行结果失败: %w", err)
	}
	return nil
}

// defaultRuntimeTarget 根据断言和合约绑定推断默认攻击目标。
func defaultRuntimeTarget(assertions []ChallengeAssertionSpec, contracts []ChallengeRuntimeContractBinding) string {
	for _, assertion := range assertions {
		if strings.TrimSpace(assertion.Target) != "" {
			return assertion.Target
		}
	}
	if len(contracts) > 0 {
		return contracts[0].ContractName
	}
	return ""
}

// runtimeDeployScript 负责在链运行时中完成统一合约部署。
const runtimeDeployScript = `
const { ethers } = require('ethers');

async function main() {
  const payload = JSON.parse(Buffer.from(process.env.PAYLOAD_BASE64, 'base64').toString('utf8'));
  const provider = new ethers.JsonRpcProvider(payload.rpc_url);
  const signer = await provider.getSigner(0);
  const items = [];
  for (const item of (payload.contracts || [])) {
    const abi = item.abi_json ? JSON.parse(item.abi_json) : [];
    const args = Array.isArray(item.constructor_args) ? item.constructor_args : [];
    const factory = new ethers.ContractFactory(abi, item.bytecode, signer);
    const contract = await factory.deploy(...args);
    await contract.waitForDeployment();
    items.push({
      challenge_id: Number(item.challenge_id || 0),
      contract_name: item.contract_name,
      address: await contract.getAddress(),
      abi_json: item.abi_json || '[]'
    });
  }
  process.stdout.write(JSON.stringify({ contracts: items }));
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`

// runtimeSetupScript 负责执行题目初始化交易，把环境推进到可利用状态。
const runtimeSetupScript = `
const { ethers } = require('ethers');

function normalizeMap(items, keyField, valueMapper) {
  const map = new Map();
  for (const item of (items || [])) {
    const key = String(item && item[keyField] ? item[keyField] : '').trim();
    if (!key) continue;
    map.set(key, valueMapper(item));
  }
  return map;
}

async function resolveSigner(provider, tx, accountMap, impersonatedSet) {
  const from = String(tx.from || '').trim();
  if (!from) {
    return await provider.getSigner(0);
  }
  if (ethers.isAddress(from)) {
    if (impersonatedSet.has(from.toLowerCase())) {
      await provider.send('hardhat_impersonateAccount', [from]);
      await provider.send('hardhat_setBalance', [from, '0x3635C9ADC5DEA00000']);
      return await provider.getSigner(from);
    }
    return await provider.getSigner(from);
  }
  const account = accountMap.get(from);
  if (!account) {
    return await provider.getSigner(0);
  }
  if (ethers.isAddress(account.name) && impersonatedSet.has(account.name.toLowerCase())) {
    await provider.send('hardhat_impersonateAccount', [account.name]);
    await provider.send('hardhat_setBalance', [account.name, '0x3635C9ADC5DEA00000']);
    return await provider.getSigner(account.name);
  }
  return await provider.getSigner(account.index);
}

function resolveTarget(tx, contractMap, pinnedMap) {
  const target = String(tx.to || '').trim();
  if (!target) return null;
  if (ethers.isAddress(target)) return target;
  if (contractMap.has(target)) return contractMap.get(target).address;
  if (pinnedMap.has(target)) return pinnedMap.get(target).address;
  return target;
}

function normalizeArgs(args, contractMap, pinnedMap) {
  return (Array.isArray(args) ? args : []).map((item) => {
    if (typeof item !== 'string') return item;
    const raw = item.trim();
    if (!raw) return raw;
    if (contractMap.has(raw)) return contractMap.get(raw).address;
    if (pinnedMap.has(raw)) return pinnedMap.get(raw).address;
    return item;
  });
}

function parseValue(raw) {
  const value = String(raw || '').trim();
  if (!value) return 0n;
  const lower = value.toLowerCase();
  if (lower.endsWith(' ether')) return ethers.parseEther(value.slice(0, -6).trim());
  if (lower.endsWith(' gwei')) return ethers.parseUnits(value.slice(0, -5).trim(), 'gwei');
  if (lower.endsWith(' wei')) return BigInt(value.slice(0, -4).trim());
  if (/^0x[0-9a-f]+$/i.test(value)) return BigInt(value);
  if (/^-?\d+$/.test(value)) return BigInt(value);
  return ethers.parseEther(value);
}

async function main() {
  const payload = JSON.parse(Buffer.from(process.env.PAYLOAD_BASE64, 'base64').toString('utf8'));
  const provider = new ethers.JsonRpcProvider(payload.rpc_url);
  const contractMap = normalizeMap(payload.contracts, 'contract_name', (item) => item);
  const pinnedMap = normalizeMap(payload.pinned_contracts, 'name', (item) => item);
  const accountMap = new Map();
  (payload.accounts || []).forEach((item, index) => {
    if (item && item.name) accountMap.set(String(item.name).trim(), { ...item, index });
  });
  const impersonatedSet = new Set((payload.impersonated_accounts || []).map((item) => String(item).toLowerCase()));

  let applied = 0;
  for (const tx of (payload.setup_transactions || [])) {
    const signer = await resolveSigner(provider, tx, accountMap, impersonatedSet);
    const target = resolveTarget(tx, contractMap, pinnedMap);
    const value = parseValue(tx.value);
    const fn = String(tx.function || '').trim();
    if (fn) {
      const contractInfo = contractMap.get(String(tx.to || '').trim()) || pinnedMap.get(String(tx.to || '').trim());
      const abi = contractInfo && contractInfo.abi_json ? JSON.parse(contractInfo.abi_json) : [];
      const contract = new ethers.Contract(target, abi, signer);
      const call = await contract[fn](...normalizeArgs(tx.args, contractMap, pinnedMap), { value });
      await call.wait();
    } else {
      const sent = await signer.sendTransaction({ to: target, value, data: '0x' });
      await sent.wait();
    }
    applied += 1;
  }
  process.stdout.write(JSON.stringify({ applied }));
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`

// runtimeExecutionScript 负责执行攻击交易并依据断言返回真实判定结果。
const runtimeExecutionScript = `
const { ethers } = require('ethers');

function parseJSONBase64() {
  return JSON.parse(Buffer.from(process.env.PAYLOAD_BASE64, 'base64').toString('utf8'));
}

function parseExpected(value) {
  if (value === null || value === undefined) return null;
  if (typeof value !== 'string') return value;
  const raw = value.trim();
  if (!raw) return raw;
  const lower = raw.toLowerCase();
  if (lower.endsWith(' ether')) return ethers.parseEther(raw.slice(0, -6).trim());
  if (lower.endsWith(' gwei')) return ethers.parseUnits(raw.slice(0, -5).trim(), 'gwei');
  if (lower.endsWith(' wei')) return BigInt(raw.slice(0, -4).trim());
  if (/^0x[0-9a-f]+$/i.test(raw)) return raw.toLowerCase();
  if (/^-?\d+$/.test(raw)) return BigInt(raw);
  if (lower === 'true' || lower === 'false') return lower === 'true';
  return raw;
}

function compareValue(actual, expected, operator) {
  const op = (operator || 'eq').toLowerCase();
  if (typeof actual === 'bigint' || typeof expected === 'bigint') {
    const left = typeof actual === 'bigint' ? actual : BigInt(actual);
    const right = typeof expected === 'bigint' ? expected : BigInt(expected);
    if (op === 'lt') return left < right;
    if (op === 'le') return left <= right;
    if (op === 'gt') return left > right;
    if (op === 'ge') return left >= right;
    if (op === 'ne') return left !== right;
    return left === right;
  }
  if (op === 'contains') return String(actual).includes(String(expected));
  if (op === 'ne') return String(actual) !== String(expected);
  return String(actual) === String(expected);
}

function formatActual(value) {
  if (typeof value === 'bigint') return value.toString();
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (value === null || value === undefined) return '';
  return String(value);
}

function normalizeContracts(contracts) {
  const byName = new Map();
  const byAddress = new Map();
  for (const item of (contracts || [])) {
    if (item.contract_name) byName.set(item.contract_name, item);
    if (item.address) byAddress.set(String(item.address).toLowerCase(), item);
  }
  return { byName, byAddress };
}

function resolveTarget(assertion, contractMaps) {
  const target = (assertion.target || '').trim();
  if (!target) return null;
  if (/^0x[0-9a-f]{40}$/i.test(target)) return { address: target, binding: contractMaps.byAddress.get(target.toLowerCase()) };
  const binding = contractMaps.byName.get(target);
  if (binding) return { address: binding.address, binding };
  return null;
}

async function executeSubmission(provider, signer, payload, contractMaps) {
  const submission = String(payload.submission || '').trim();
  if (!submission) {
    return { txHash: null };
  }
  let spec = null;
  if (submission.startsWith('{')) {
    spec = JSON.parse(submission);
  }
  if (spec && spec.bytecode) {
    const abi = Array.isArray(spec.abi) ? spec.abi : [];
    const factory = new ethers.ContractFactory(abi, spec.bytecode, signer);
    const contract = await factory.deploy(...(Array.isArray(spec.constructor_args) ? spec.constructor_args : []));
    await contract.waitForDeployment();
    const tx = contract.deploymentTransaction();
    return { txHash: tx ? tx.hash : null };
  }
  const defaultTarget = (payload.default_target || '').trim();
  const targetBinding = defaultTarget ? (contractMaps.byName.get(defaultTarget) || contractMaps.byAddress.get(defaultTarget.toLowerCase())) : null;
  const txRequest = {
    to: spec && spec.to ? (contractMaps.byName.get(spec.to)?.address || spec.to) : (targetBinding ? targetBinding.address : null),
    data: spec && spec.data ? spec.data : submission,
    value: spec && spec.value ? parseExpected(spec.value) : 0n,
    gasLimit: 12000000
  };
  try {
    const tx = await signer.sendTransaction(txRequest);
    await tx.wait();
    return { txHash: tx.hash };
  } catch (sendError) {
    if (/^0x[0-9a-f]+$/i.test(submission)) {
      const deployTx = await signer.sendTransaction({ data: submission, gasLimit: 12000000 });
      await deployTx.wait();
      return { txHash: deployTx.hash };
    }
    throw sendError;
  }
}

async function evaluateAssertion(provider, receipt, assertion, contractMaps) {
  const resolved = resolveTarget(assertion, contractMaps);
  const expected = parseExpected(assertion.expected_value);
  const extra = assertion.extra_params || {};
  let actual = '';
  let passed = false;
  switch ((assertion.assertion_type || '').toLowerCase()) {
    case 'balance_check': {
      if (!resolved) throw new Error('balance_check 缺少目标合约');
      const balance = await provider.getBalance(resolved.address);
      actual = balance;
      passed = compareValue(balance, expected, assertion.operator);
      break;
    }
    case 'token_balance_check': {
      if (!resolved || !resolved.binding) throw new Error('token_balance_check 缺少代币合约绑定');
      const owner = extra.owner || extra.account || extra.holder || await (await provider.getSigner(0)).getAddress();
      const abi = JSON.parse(resolved.binding.abi_json || '[]');
      const contract = new ethers.Contract(resolved.address, abi, provider);
      const value = await contract.balanceOf(owner);
      actual = value;
      passed = compareValue(value, expected, assertion.operator);
      break;
    }
    case 'storage_check': {
      if (!resolved) throw new Error('storage_check 缺少目标合约');
      const slot = extra.slot !== undefined ? extra.slot : '0x0';
      const value = await provider.getStorage(resolved.address, slot);
      actual = value;
      passed = compareValue(String(value).toLowerCase(), String(expected).toLowerCase(), assertion.operator);
      break;
    }
    case 'owner_check': {
      if (!resolved || !resolved.binding) throw new Error('owner_check 缺少目标合约绑定');
      const abi = JSON.parse(resolved.binding.abi_json || '[]');
      const fn = extra.function || 'owner';
      const contract = new ethers.Contract(resolved.address, abi, provider);
      const value = await contract[fn]();
      actual = String(value).toLowerCase();
      passed = compareValue(actual, String(expected).toLowerCase(), assertion.operator);
      break;
    }
    case 'event_check': {
      const targetAddress = resolved ? resolved.address.toLowerCase() : '';
      const topic0 = String(extra.topic0 || extra.event_signature || '').toLowerCase();
      const logs = receipt && Array.isArray(receipt.logs) ? receipt.logs : [];
      const logCount = logs.filter((log) => {
        if (targetAddress && String(log.address || '').toLowerCase() !== targetAddress) return false;
        if (topic0 && (!Array.isArray(log.topics) || String(log.topics[0] || '').toLowerCase() !== topic0)) return false;
        return true;
      }).length;
      actual = BigInt(logCount);
      passed = compareValue(BigInt(logCount), expected === null ? 0n : expected, assertion.operator || 'gt');
      break;
    }
    case 'code_check': {
      if (!resolved) throw new Error('code_check 缺少目标合约');
      const code = await provider.getCode(resolved.address);
      const size = BigInt(Math.max((code.length - 2) / 2, 0));
      actual = size;
      passed = compareValue(size, expected, assertion.operator);
      break;
    }
    case 'custom_script': {
      const script = String(extra.script || extra.javascript || '').trim();
      if (!script) throw new Error('custom_script 缺少脚本内容');
      const fn = new Function('ethers', 'provider', 'receipt', 'contracts', 'assertion', script);
      const value = await fn(ethers, provider, receipt, Object.fromEntries(contractMaps.byName.entries()), assertion);
      if (value && typeof value === 'object' && Object.prototype.hasOwnProperty.call(value, 'passed')) {
        actual = value.actual !== undefined ? value.actual : '';
        passed = Boolean(value.passed);
      } else {
        actual = value;
        passed = Boolean(value);
      }
      break;
    }
    default:
      throw new Error('不支持的断言类型: ' + assertion.assertion_type);
  }
  return {
    type: assertion.assertion_type,
    target: assertion.target || '',
    expected: assertion.expected_value || '',
    actual: formatActual(actual),
    passed
  };
}

async function main() {
  const startedAt = Date.now();
  const payload = parseJSONBase64();
  const provider = new ethers.JsonRpcProvider(payload.rpc_url);
  const signer = await provider.getSigner(0);
  const contractMaps = normalizeContracts(payload.contracts);
  let txHash = null;
  let receipt = null;
  let errorMessage = null;
  try {
    const execResult = await executeSubmission(provider, signer, payload, contractMaps);
    txHash = execResult.txHash;
    if (txHash) {
      receipt = await provider.getTransactionReceipt(txHash);
    }
  } catch (error) {
    errorMessage = error && error.shortMessage ? error.shortMessage : (error && error.message ? error.message : String(error));
  }
  const results = [];
  if (!errorMessage) {
    for (const assertion of (payload.assertions || [])) {
      results.push(await evaluateAssertion(provider, receipt, assertion, contractMaps));
    }
  }
  const allPassed = !errorMessage && results.every((item) => item.passed);
  const executionTimeMS = Date.now() - startedAt;
  process.stdout.write(JSON.stringify({
    all_passed: allPassed,
    results,
    execution_time_ms: executionTimeMS,
    tx_hash: txHash,
    error_message: errorMessage
  }));
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`

// runtimePatchVerificationScript 负责补丁编译、ABI兼容校验和官方 PoC 回放。
const runtimePatchVerificationScript = `
const { ethers } = require('ethers');
let solc = null;
try { solc = require('solc'); } catch (error) { solc = null; }

function compilePatch(source) {
  if (!solc) {
    throw new Error('补丁编译依赖 solc 未安装');
  }
  const input = {
    language: 'Solidity',
    sources: { 'Patch.sol': { content: source } },
    settings: {
      optimizer: { enabled: false, runs: 200 },
      outputSelection: { '*': { '*': ['abi', 'evm.bytecode.object'] } }
    }
  };
  const output = JSON.parse(solc.compile(JSON.stringify(input)));
  if (Array.isArray(output.errors)) {
    const fatal = output.errors.filter((item) => item.severity === 'error');
    if (fatal.length > 0) {
      throw new Error(fatal.map((item) => item.formattedMessage || item.message).join('\n'));
    }
  }
  const contracts = output.contracts['Patch.sol'] || {};
  return Object.entries(contracts).map(([name, item]) => ({
    name,
    abi: item.abi || [],
    bytecode: item.evm && item.evm.bytecode ? item.evm.bytecode.object || '' : ''
  })).filter((item) => item.bytecode);
}

function publicSignatures(abi) {
  return (abi || [])
    .filter((item) => item.type === 'function')
    .map((item) => item.name + '(' + (item.inputs || []).map((input) => input.type).join(',') + ')')
    .sort();
}

async function replayPoc(provider, signer, targetAddress, poc) {
  const raw = String(poc || '').trim();
  if (!raw) return null;
  const tx = await signer.sendTransaction({ to: targetAddress, data: raw, gasLimit: 12000000 });
  await tx.wait();
  return tx.hash;
}

async function main() {
  const payload = JSON.parse(Buffer.from(process.env.PAYLOAD_BASE64, 'base64').toString('utf8'));
  const provider = new ethers.JsonRpcProvider(payload.rpc_url);
  const signer = await provider.getSigner(0);
  const compiled = compilePatch(payload.patch_source_code || '');
  if (compiled.length === 0) {
    throw new Error('补丁编译结果为空');
  }
  const original = (payload.original_contracts || [])[0];
  const originalName = original && original.contract_name ? original.contract_name : null;
  const candidate = compiled.find((item) => item.name === originalName) || compiled[0];
  const originalAbi = original && original.abi_json ? JSON.parse(original.abi_json) : [];
  const oldSigs = publicSignatures(originalAbi);
  const newSigs = publicSignatures(candidate.abi);
  const missing = oldSigs.filter((sig) => !newSigs.includes(sig));
  if (missing.length > 0) {
    process.stdout.write(JSON.stringify({
      functionality_passed: false,
      vulnerability_fixed: false,
      rejection_reason: '功能完整性检查未通过：缺少接口 ' + missing.join(', '),
      patched_contracts: []
    }));
    return;
  }
  const factory = new ethers.ContractFactory(candidate.abi, '0x' + candidate.bytecode.replace(/^0x/, ''), signer);
  const contract = await factory.deploy();
  await contract.waitForDeployment();
  const patchedAddress = await contract.getAddress();
  const txHash = payload.official_poc ? await replayPoc(provider, signer, patchedAddress, payload.official_poc) : null;
  const receipt = txHash ? await provider.getTransactionReceipt(txHash) : null;
  const balance = await provider.getBalance(patchedAddress);
  const assertions = payload.assertions || [];
  let allPassed = true;
  for (const assertion of assertions) {
    if ((assertion.assertion_type || '').toLowerCase() === 'balance_check') {
      const expectedRaw = String(assertion.expected_value || '').trim();
      const expected = expectedRaw.toLowerCase().endsWith(' ether')
        ? ethers.parseEther(expectedRaw.slice(0, -6).trim())
        : BigInt(expectedRaw || '0');
      const passed = (assertion.operator || 'lt').toLowerCase() === 'lt' ? balance < expected : balance === expected;
      if (passed) allPassed = false;
    }
    if ((assertion.assertion_type || '').toLowerCase() === 'event_check' && receipt && Array.isArray(receipt.logs) && receipt.logs.length > 0) {
      allPassed = false;
    }
  }
  process.stdout.write(JSON.stringify({
    functionality_passed: true,
    vulnerability_fixed: allPassed,
    rejection_reason: allPassed ? null : '漏洞修复验证失败：官方PoC仍可成功执行',
    patched_contracts: (payload.target_contracts || []).map((item) => {
      if (item.contract_name === candidate.name || item.contract_name === originalName) {
        return {
          challenge_id: item.challenge_id,
          contract_name: item.contract_name,
          address: patchedAddress,
          patch_version: Number(item.patch_version || 0) + 1,
          is_patched: true
        };
      }
      return item;
    })
  }));
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
`
