// ctf_runtime_adapter.go
// 模块05 — CTF竞赛：K8s 运行时实现。
// 负责在 service/ctf 内部承接模块04的 K8s 编排能力，
// 提供模块05所需的命名空间、攻防赛分组链初始化与合约部署能力。

package ctf

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
)

const (
	ctfJudgeRuntimePodName      = "judge-runtime"
	ctfJudgeChainContainer      = "judge-chain"
	ctfJudgeToolsContainer      = "judge-tools"
	ctfJudgeServiceContainer    = "judge-service"
	ctfPatchVerifierContainer   = "patch-verifier"
	ctfJudgeChainRPCPort        = 8545
	ctfJudgeChainWSPort         = 8546
	ctfTeamChainRPCPort         = 8545
	ctfRuntimeReadyPoll         = 2 * time.Second
	ctfRuntimeReadyTimeout      = 90 * time.Second
	ctfDefaultRuntimeImage      = "ctf-blockchain:latest"
	ctfDefaultJudgeImage        = "geth-dev:latest"
	ctfDefaultTeamChainImage    = "ganache:latest"
	ctfDefaultForkChainImage    = "hardhat-node:latest"
	ctfDefaultJudgeSvcImage     = "judge-service:latest"
	ctfDefaultPatchVerifierImage = "patch-verifier:latest"
)

// RuntimeProvisionerAdapter 复用模块04 K8sService 提供模块05运行时能力。
type RuntimeProvisionerAdapter struct {
	k8sSvc RuntimeClusterOperator
}

// NewRuntimeProvisionerAdapter 创建模块05运行时适配器。
func NewRuntimeProvisionerAdapter(k8sSvc RuntimeClusterOperator) *RuntimeProvisionerAdapter {
	return &RuntimeProvisionerAdapter{k8sSvc: k8sSvc}
}

// CreateNamespace 创建 CTF 命名空间。
func (a *RuntimeProvisionerAdapter) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	err := a.k8sSvc.CreateNamespace(ctx, name, labels)
	if err != nil && isAlreadyExistsError(err) {
		return nil
	}
	return err
}

// DeleteNamespace 删除 CTF 命名空间。
func (a *RuntimeProvisionerAdapter) DeleteNamespace(ctx context.Context, name string) error {
	err := a.k8sSvc.DeleteNamespace(ctx, name)
	if err != nil && isNotFoundError(err) {
		return nil
	}
	return err
}

// CreateADGroupRuntime 创建攻防赛分组所需的裁判链与队伍链运行时。
func (a *RuntimeProvisionerAdapter) CreateADGroupRuntime(ctx context.Context, spec *ADRuntimeGroupSpec) (*ADRuntimeGroupResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("攻防赛运行时规格不能为空")
	}
	labels := map[string]string{
		"module":         "ctf",
		"competition_id": fmt.Sprintf("%d", spec.CompetitionID),
		"group_id":       fmt.Sprintf("%d", spec.GroupID),
		"runtime":        "attack-defense",
	}
	if err := a.CreateNamespace(ctx, spec.Namespace, labels); err != nil {
		return nil, err
	}
	if err := a.deployJudgeRuntime(ctx, spec); err != nil {
		return nil, err
	}
	judgeRPCURL := buildClusterServiceURL(ctfJudgeChainContainer, spec.Namespace, ctfJudgeChainRPCPort, false)
	judgeContractAddress, err := a.deployJudgeSettlementContract(ctx, spec.Namespace, judgeRPCURL)
	if err != nil {
		return nil, err
	}

	teamResults := make([]ADRuntimeTeamResult, 0, len(spec.Teams))
	for _, teamSpec := range spec.Teams {
		teamResult, err := a.deployTeamRuntime(ctx, spec, &teamSpec)
		if err != nil {
			return nil, err
		}
		teamResults = append(teamResults, *teamResult)
	}

	return &ADRuntimeGroupResult{
		JudgeChainURL:        stringPtr(judgeRPCURL),
		JudgeContractAddress: stringPtr(judgeContractAddress),
		Teams:                teamResults,
	}, nil
}

// DeleteADGroupRuntime 删除攻防赛分组运行时占用的整组命名空间资源。
func (a *RuntimeProvisionerAdapter) DeleteADGroupRuntime(ctx context.Context, namespace string) error {
	return a.DeleteNamespace(ctx, namespace)
}

// deployJudgeRuntime 部署裁判链、judge-service 和 patch-verifier 容器。
func (a *RuntimeProvisionerAdapter) deployJudgeRuntime(ctx context.Context, spec *ADRuntimeGroupSpec) error {
	judgeImage := strings.TrimSpace(spec.JudgeChainImage)
	if judgeImage == "" {
		judgeImage = ctfDefaultJudgeImage
	}
	judgeRPCURL := buildClusterServiceURL(ctfJudgeChainContainer, spec.Namespace, ctfJudgeChainRPCPort, false)
	_, err := a.k8sSvc.DeployPod(ctx, &RuntimeDeployPodRequest{
		Namespace: spec.Namespace,
		PodName:   ctfJudgeRuntimePodName,
		Labels: map[string]string{
			"module":   "ctf",
			"runtime":  "judge-chain",
			"group_id": fmt.Sprintf("%d", spec.GroupID),
		},
		Containers: []RuntimeContainerSpec{
			{
				Name:  ctfJudgeChainContainer,
				Image: judgeImage,
				Ports: []RuntimePortSpec{
					{ContainerPort: ctfJudgeChainRPCPort, Protocol: "TCP", ServicePort: ctfJudgeChainRPCPort},
					{ContainerPort: ctfJudgeChainWSPort, Protocol: "TCP", ServicePort: ctfJudgeChainWSPort},
				},
			},
			{
				Name:  ctfJudgeServiceContainer,
				Image: ctfDefaultJudgeSvcImage,
				Ports: []RuntimePortSpec{
					{ContainerPort: judgeServicePort, Protocol: "TCP", ServicePort: judgeServicePort},
				},
				EnvVars: map[string]string{
					"LISTEN_ADDR": fmt.Sprintf(":%d", judgeServicePort),
					"DEFAULT_RPC": judgeRPCURL,
				},
			},
			{
				Name:  ctfPatchVerifierContainer,
				Image: ctfDefaultPatchVerifierImage,
				Ports: []RuntimePortSpec{
					{ContainerPort: patchVerifierPort, Protocol: "TCP", ServicePort: patchVerifierPort},
				},
				EnvVars: map[string]string{
					"LISTEN_ADDR": fmt.Sprintf(":%d", patchVerifierPort),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return a.waitForPodRunning(ctx, spec.Namespace, ctfJudgeRuntimePodName)
}

// deployTeamRuntime 部署单支队伍的链节点、工具容器，并完成题目合约初始化部署。
func (a *RuntimeProvisionerAdapter) deployTeamRuntime(ctx context.Context, groupSpec *ADRuntimeGroupSpec, teamSpec *ADRuntimeTeamSpec) (*ADRuntimeTeamResult, error) {
	teamImage := strings.TrimSpace(groupSpec.TeamChainImage)
	if teamImage == "" {
		teamImage = ctfDefaultTeamChainImage
	}
	toolImage := strings.TrimSpace(groupSpec.RuntimeToolImage)
	if toolImage == "" {
		toolImage = ctfDefaultRuntimeImage
	}
	podName := buildTeamRuntimePodName(teamSpec.TeamID)
	chainContainer := buildTeamChainContainerName(teamSpec.TeamID)
	toolsContainer := buildTeamToolsContainerName(teamSpec.TeamID)
	_, err := a.k8sSvc.DeployPod(ctx, &RuntimeDeployPodRequest{
		Namespace: groupSpec.Namespace,
		PodName:   podName,
		Labels: map[string]string{
			"module":   "ctf",
			"runtime":  "team-chain",
			"group_id": fmt.Sprintf("%d", groupSpec.GroupID),
			"team_id":  fmt.Sprintf("%d", teamSpec.TeamID),
		},
		Containers: []RuntimeContainerSpec{
			{
				Name:  chainContainer,
				Image: teamImage,
				Ports: []RuntimePortSpec{
					{ContainerPort: ctfTeamChainRPCPort, Protocol: "TCP", ServicePort: ctfTeamChainRPCPort},
				},
			},
			{
				Name:  toolsContainer,
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
	if err := a.waitForPodRunning(ctx, groupSpec.Namespace, podName); err != nil {
		return nil, err
	}
	rpcURL := buildClusterServiceURL(chainContainer, groupSpec.Namespace, ctfTeamChainRPCPort, false)
	wsURL := buildClusterServiceURL(chainContainer, groupSpec.Namespace, ctfTeamChainRPCPort, true)
	deployedContracts, err := a.deployTeamContracts(ctx, groupSpec.Namespace, rpcURL, teamSpec.Contracts)
	if err != nil {
		return nil, err
	}
	return &ADRuntimeTeamResult{
		TeamID:              teamSpec.TeamID,
		ChainRPCURL:         stringPtr(rpcURL),
		ChainWSURL:          stringPtr(wsURL),
		DeployedContracts:   deployedContracts,
		CurrentPatchVersion: 0,
		Status:              2,
	}, nil
}

// deployJudgeSettlementContract 通过 judge-service 在裁判链部署最小结算锚点合约。
func (a *RuntimeProvisionerAdapter) deployJudgeSettlementContract(ctx context.Context, namespace, rpcURL string) (string, error) {
	payload := dto.JudgeDeployRequest{
		RPCURL: rpcURL,
		Contracts: []dto.JudgeDeployContractSpec{
			{
				ChallengeID:     0,
				ContractName:    "JudgeSettlementAnchor",
				ABIJSON:         "[]",
				Bytecode:        "0x60006000f3",
				ConstructorArgs: []interface{}{},
				DeployOrder:     1,
			},
		},
	}
	judgeURL := buildJudgeServiceURL(namespace)
	var result dto.JudgeDeployResponse
	if err := postJSON(ctx, judgeURL+"/api/v1/contracts/deploy", payload, &result); err != nil {
		return "", err
	}
	if len(result.Contracts) == 0 || strings.TrimSpace(result.Contracts[0].Address) == "" {
		return "", fmt.Errorf("裁判链结算锚点部署结果为空")
	}
	return result.Contracts[0].Address, nil
}

// deployTeamContracts 通过 judge-service 在队伍链部署合约。
func (a *RuntimeProvisionerAdapter) deployTeamContracts(
	ctx context.Context,
	namespace, rpcURL string,
	contracts []ADRuntimeContractSpec,
) ([]dto.TeamChainContractItem, error) {
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
	items := make([]dto.TeamChainContractItem, 0, len(result.Contracts))
	for _, c := range result.Contracts {
		items = append(items, dto.TeamChainContractItem{
			ChallengeID:  c.ChallengeID,
			ContractName: c.ContractName,
			Address:      c.Address,
			PatchVersion: c.PatchVersion,
			IsPatched:    c.IsPatched,
		})
	}
	return items, nil
}

// waitForPodRunning 等待运行时 Pod 进入 Running，避免在链尚未启动时执行部署脚本。
func (a *RuntimeProvisionerAdapter) waitForPodRunning(ctx context.Context, namespace, podName string) error {
	deadline := time.Now().Add(ctfRuntimeReadyTimeout)
	for time.Now().Before(deadline) {
		status, err := a.k8sSvc.GetPodStatus(ctx, namespace, podName)
		if err == nil && status != nil && strings.EqualFold(status.Status, "Running") {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ctfRuntimeReadyPoll):
		}
	}
	return fmt.Errorf("等待运行时 Pod 就绪超时: %s/%s", namespace, podName)
}

// buildTeamRuntimePodName 构建队伍运行时 Pod 名称。
func buildTeamRuntimePodName(teamID int64) string {
	return fmt.Sprintf("team-%d-runtime", teamID)
}

// buildTeamChainContainerName 构建队伍链容器名称。
func buildTeamChainContainerName(teamID int64) string {
	return fmt.Sprintf("team-%d-chain", teamID)
}

// buildTeamToolsContainerName 构建队伍工具容器名称。
func buildTeamToolsContainerName(teamID int64) string {
	return fmt.Sprintf("team-%d-tools", teamID)
}

// buildClusterServiceURL 构建集群内 Service DNS 地址。
func buildClusterServiceURL(serviceName, namespace string, port int, useWS bool) string {
	scheme := "http"
	if useWS {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", scheme, serviceName, namespace, port)
}

// isAlreadyExistsError 判断 Kubernetes 创建操作是否因资源已存在而失败。
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already exists")
}

// isNotFoundError 判断 Kubernetes 删除动作是否因为资源已不存在而失败。
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "not found") || strings.Contains(err.Error(), "不存在")
}
