// ctf_runtime_execution.go
// 模块05 — CTF竞赛：解题赛与攻防赛运行时执行器。
// 通过 HTTP API 调用 judge-service 和 patch-verifier 微服务完成
// 链上攻击验证、补丁验证和合约部署，替代旧的 kubectl exec + 内嵌脚本方案。

package ctf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
)

// judgeServicePort judge-service 监听端口。
const judgeServicePort = 8090

// patchVerifierPort patch-verifier 监听端口。
const patchVerifierPort = 8091

// httpClient 共享的 HTTP 客户端，复用连接池。
var httpClient = &http.Client{Timeout: 90 * time.Second}

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

// ExecuteChallengeSubmission 通过 judge-service 执行链上攻击并校验断言。
func (a *RuntimeProvisionerAdapter) ExecuteChallengeSubmission(ctx context.Context, spec *ChallengeSubmissionSpec) (*ChallengeSubmissionResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("题目提交执行规格不能为空")
	}
	contracts := make([]dto.JudgeContractBinding, 0, len(spec.Contracts))
	for _, c := range spec.Contracts {
		contracts = append(contracts, dto.JudgeContractBinding{
			ChallengeID:  fmt.Sprintf("%d", c.ChallengeID),
			ContractName: c.ContractName,
			Address:      c.Address,
			ABIJSON:      c.ABIJSON,
		})
	}
	assertions := buildJudgeAssertionSpecs(spec.Assertions)
	payload := dto.JudgeAttackRequest{
		RPCURL:        spec.ChainRPCURL,
		Submission:    spec.SubmissionData,
		Contracts:     contracts,
		Assertions:    assertions,
		DefaultTarget: defaultRuntimeTarget(spec.Assertions, spec.Contracts),
	}
	judgeURL := buildJudgeServiceURL(spec.Namespace)
	var result dto.JudgeAttackResponse
	if err := postJSON(ctx, judgeURL+"/api/v1/attacks/execute", payload, &result); err != nil {
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

// ExecuteADAttack 通过 judge-service 在目标队伍链执行攻击交易并校验断言。
func (a *RuntimeProvisionerAdapter) ExecuteADAttack(ctx context.Context, spec *ADAttackExecutionSpec) (*ADAttackExecutionResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("攻防赛攻击执行规格不能为空")
	}
	contracts := make([]dto.JudgeContractBinding, 0, len(spec.Contracts))
	for _, c := range spec.Contracts {
		contracts = append(contracts, dto.JudgeContractBinding{
			ChallengeID:  c.ChallengeID,
			ContractName: c.ContractName,
			Address:      c.Address,
		})
	}
	payload := dto.JudgeAttackRequest{
		RPCURL:        spec.ChainRPCURL,
		Submission:    spec.AttackTxData,
		Contracts:     contracts,
		Assertions:    buildJudgeAssertionSpecs(spec.Assertions),
		DefaultTarget: defaultRuntimeTarget(spec.Assertions, nil),
	}
	judgeURL := buildJudgeServiceURL(spec.Namespace)
	var result dto.JudgeAttackResponse
	if err := postJSON(ctx, judgeURL+"/api/v1/attacks/execute", payload, &result); err != nil {
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

// VerifyADPatch 通过 patch-verifier 编译补丁并回放官方 PoC。
func (a *RuntimeProvisionerAdapter) VerifyADPatch(ctx context.Context, spec *ADPatchVerificationSpec) (*ADPatchVerificationResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("补丁验证规格不能为空")
	}
	originalContracts := make([]dto.VerifierContractSpec, 0, len(spec.OriginalContracts))
	for _, c := range spec.OriginalContracts {
		originalContracts = append(originalContracts, dto.VerifierContractSpec{
			ChallengeID:  c.ChallengeID,
			ContractName: c.ContractName,
			ABIJSON:      c.ABIJSON,
			Bytecode:     c.Bytecode,
			DeployOrder:  c.DeployOrder,
		})
	}
	targetContracts := make([]dto.VerifierContractBinding, 0, len(spec.TargetContracts))
	for _, c := range spec.TargetContracts {
		targetContracts = append(targetContracts, dto.VerifierContractBinding{
			ChallengeID:  c.ChallengeID,
			ContractName: c.ContractName,
			Address:      c.Address,
			PatchVersion: c.PatchVersion,
			IsPatched:    c.IsPatched,
		})
	}
	assertions := buildJudgeAssertionSpecs(spec.Assertions)
	payload := dto.VerifierRequest{
		RPCURL:            spec.ChainRPCURL,
		ChallengeID:       spec.ChallengeID,
		ChallengeTitle:    spec.ChallengeTitle,
		PatchSourceCode:   spec.PatchSourceCode,
		OriginalContracts: originalContracts,
		TargetContracts:   targetContracts,
		Assertions:        assertions,
		OfficialPoc:       spec.OfficialPocContent,
	}
	verifierURL := buildPatchVerifierURL(spec.Namespace)
	var result dto.VerifierResponse
	if err := postJSON(ctx, verifierURL+"/api/v1/patches/verify", payload, &result); err != nil {
		return nil, err
	}
	patchedContracts := make([]dto.TeamChainContractItem, 0, len(result.PatchedContracts))
	for _, c := range result.PatchedContracts {
		patchedContracts = append(patchedContracts, dto.TeamChainContractItem{
			ChallengeID:  c.ChallengeID,
			ContractName: c.ContractName,
			Address:      c.Address,
			PatchVersion: c.PatchVersion,
			IsPatched:    c.IsPatched,
		})
	}
	return &ADPatchVerificationResult{
		FunctionalityPassed: result.FunctionalityPassed,
		VulnerabilityFixed:  result.VulnerabilityFixed,
		RejectionReason:     result.RejectionReason,
		PatchedContracts:    patchedContracts,
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
	bindings, err := a.deployChallengeContracts(ctx, spec.Namespace, rpcURL, spec.Contracts)
	if err != nil {
		return nil, err
	}
	if err := a.executeChallengeSetupTransactions(ctx, spec.Namespace, rpcURL, spec, bindings); err != nil {
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

// executeChallengeSetupTransactions 通过 judge-service 在题目环境完成部署后回放初始化交易。
func (a *RuntimeProvisionerAdapter) executeChallengeSetupTransactions(
	ctx context.Context,
	namespace, rpcURL string,
	spec *ChallengeEnvironmentSpec,
	bindings []ChallengeRuntimeContractBinding,
) error {
	if spec == nil || len(spec.SetupTransactions) == 0 {
		return nil
	}
	contracts := make([]dto.JudgeContractBinding, 0, len(bindings))
	for _, b := range bindings {
		contracts = append(contracts, dto.JudgeContractBinding{
			ChallengeID:  fmt.Sprintf("%d", b.ChallengeID),
			ContractName: b.ContractName,
			Address:      b.Address,
			ABIJSON:      b.ABIJSON,
		})
	}
	payload := map[string]interface{}{
		"rpc_url":               rpcURL,
		"runtime_mode":          spec.RuntimeMode,
		"accounts":              nil,
		"contracts":             contracts,
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
	judgeURL := buildJudgeServiceURL(namespace)
	var result struct {
		Applied int `json:"applied"`
	}
	return postJSON(ctx, judgeURL+"/api/v1/transactions/setup", payload, &result)
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

// deployChallengeContracts 通过 judge-service 部署漏洞合约并返回绑定信息。
func (a *RuntimeProvisionerAdapter) deployChallengeContracts(
	ctx context.Context,
	namespace, rpcURL string,
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
	specs := make([]dto.JudgeDeployContractSpec, 0, len(contracts))
	for _, c := range contracts {
		specs = append(specs, dto.JudgeDeployContractSpec{
			ChallengeID:     c.ChallengeID,
			ContractName:    c.ContractName,
			ABIJSON:         c.ABIJSON,
			Bytecode:        c.Bytecode,
			ConstructorArgs: c.ConstructorArgs,
			DeployOrder:     c.DeployOrder,
		})
	}
	payload := dto.JudgeDeployRequest{
		RPCURL:    rpcURL,
		Contracts: specs,
	}
	judgeURL := buildJudgeServiceURL(namespace)
	var result dto.JudgeDeployResponse
	if err := postJSON(ctx, judgeURL+"/api/v1/contracts/deploy", payload, &result); err != nil {
		return nil, err
	}
	bindings := make([]ChallengeRuntimeContractBinding, 0, len(result.Contracts))
	for _, c := range result.Contracts {
		var cid int64
		fmt.Sscanf(c.ChallengeID, "%d", &cid)
		bindings = append(bindings, ChallengeRuntimeContractBinding{
			ChallengeID:  cid,
			ContractName: c.ContractName,
			Address:      c.Address,
			ABIJSON:      c.ABIJSON,
		})
	}
	return bindings, nil
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

// ── HTTP 通信辅助 ──────────────────────────────────────────────

// buildJudgeServiceURL 构建 judge-service 集群内 URL。
func buildJudgeServiceURL(namespace string) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		ctfJudgeServiceContainer, namespace, judgeServicePort)
}

// buildPatchVerifierURL 构建 patch-verifier 集群内 URL。
func buildPatchVerifierURL(namespace string) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		ctfPatchVerifierContainer, namespace, patchVerifierPort)
}

// buildJudgeAssertionSpecs 将 ChallengeAssertionSpec 切片转换为 judge-service 断言 DTO 格式。
func buildJudgeAssertionSpecs(assertions []ChallengeAssertionSpec) []dto.JudgeAssertionSpec {
	result := make([]dto.JudgeAssertionSpec, 0, len(assertions))
	for _, a := range assertions {
		result = append(result, dto.JudgeAssertionSpec{
			AssertionType: a.AssertionType,
			Target:        a.Target,
			Operator:      a.Operator,
			ExpectedValue: a.ExpectedValue,
			ExtraParams:   a.ExtraParams,
		})
	}
	return result
}

// postJSON 向目标 URL 发送 JSON POST 请求并解析响应。
func postJSON(ctx context.Context, url string, body interface{}, target interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 %s 失败: %w", url, err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("服务返回错误 %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}
	if err := json.Unmarshal(respBytes, target); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	return nil
}
