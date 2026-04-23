// runtime_adapter.go
// 模块05 — CTF竞赛：运行时编排适配接口。
// 该文件声明模块05对外部环境编排能力的最小依赖，避免 service 直接依赖模块04具体实现，
// 同时为后续在 init 层注入 K8s/实验环境适配器预留接口。

package ctf

import (
	"context"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// NamespaceProvisioner 定义 CTF 模块需要的最小命名空间编排能力。
type NamespaceProvisioner interface {
	CreateNamespace(ctx context.Context, name string, labels map[string]string) error
	DeleteNamespace(ctx context.Context, name string) error
}

// RuntimeClusterOperator 定义模块05运行时编排所需的最小 K8s 能力。
// 该接口屏蔽模块04具体实现与请求模型，确保模块05 service 只依赖本模块抽象。
type RuntimeClusterOperator interface {
	CreateNamespace(ctx context.Context, name string, labels map[string]string) error
	DeleteNamespace(ctx context.Context, name string) error
	DeployPod(ctx context.Context, req *RuntimeDeployPodRequest) (*RuntimeDeployPodResponse, error)
	ExecInPod(ctx context.Context, namespace, podName, container, command string) (*RuntimeExecResult, error)
	GetPodStatus(ctx context.Context, namespace, podName string) (*RuntimePodStatus, error)
}

// ChallengeEnvironmentProvisioner 定义解题赛题目环境编排接口。
// 该接口负责创建真实题目运行时，并返回链地址、容器状态和部署后的合约元数据。
type ChallengeEnvironmentProvisioner interface {
	ProvisionChallengeEnvironment(ctx context.Context, spec *ChallengeEnvironmentSpec) (*ChallengeEnvironmentResult, error)
}

// ChallengeSubmissionExecutor 定义解题赛提交执行接口。
// 该接口负责在选手题目环境中执行攻击交易，并依据断言返回真实验证结果。
type ChallengeSubmissionExecutor interface {
	ExecuteChallengeSubmission(ctx context.Context, spec *ChallengeSubmissionSpec) (*ChallengeSubmissionResult, error)
}

// ADRuntimeProvisioner 定义攻防赛链运行时编排接口。
// 该接口负责真实创建裁判链、队伍链并返回链访问地址与已部署合约信息。
type ADRuntimeProvisioner interface {
	CreateADGroupRuntime(ctx context.Context, spec *ADRuntimeGroupSpec) (*ADRuntimeGroupResult, error)
	DeleteADGroupRuntime(ctx context.Context, namespace string) error
}

// ADAttackExecutor 定义攻防赛攻击代理执行接口。
// 该接口负责在目标队伍链执行攻击交易，并返回断言校验结果。
type ADAttackExecutor interface {
	ExecuteADAttack(ctx context.Context, spec *ADAttackExecutionSpec) (*ADAttackExecutionResult, error)
}

// ADPatchVerifier 定义攻防赛补丁验证接口。
// 该接口负责在隔离运行时中编译补丁、校验 ABI/功能兼容性并执行官方 PoC 回放。
type ADPatchVerifier interface {
	VerifyADPatch(ctx context.Context, spec *ADPatchVerificationSpec) (*ADPatchVerificationResult, error)
}

// ChallengeEnvironmentSpec 描述解题赛题目环境编排输入。
type ChallengeEnvironmentSpec struct {
	CompetitionID      int64
	ChallengeID        int64
	TeamID             int64
	Namespace          string
	RuntimeProfileName string
	RuntimeMode        int16
	ChainConfig        *dto.ChallengeChainConfig
	SetupTransactions  []dto.ChallengeSetupTransaction
	EnvironmentConfig  *dto.ChallengeEnvironmentConfig
	Contracts          []ChallengeRuntimeContractSpec
}

// ChallengeRuntimeContractSpec 描述解题赛/验证运行时中的单个题目合约。
type ChallengeRuntimeContractSpec struct {
	ChallengeID     int64
	ContractName    string
	ABIJSON         string
	Bytecode        string
	ConstructorArgs []interface{}
	DeployOrder     int
}

// ChallengeRuntimeContractBinding 描述部署后的题目合约绑定信息。
type ChallengeRuntimeContractBinding struct {
	ChallengeID  int64
	ContractName string
	Address      string
	ABIJSON      string
}

// ChallengeEnvironmentResult 描述题目环境编排完成后的真实资源结果。
type ChallengeEnvironmentResult struct {
	ChainRPCURL     *string
	ContainerStatus map[string]dto.ChallengeEnvironmentContainerState
	Contracts       []ChallengeRuntimeContractBinding
}

// ChallengeSubmissionSpec 描述解题赛链上提交执行输入。
type ChallengeSubmissionSpec struct {
	Namespace      string
	ChainRPCURL    string
	SubmissionData string
	Contracts      []ChallengeRuntimeContractBinding
	Accounts       []dto.ChallengeChainAccount
	Assertions     []ChallengeAssertionSpec
}

// ChallengeAssertionSpec 描述运行时执行时使用的断言定义。
type ChallengeAssertionSpec struct {
	AssertionType string
	Target        string
	Operator      string
	ExpectedValue string
	ExtraParams   map[string]interface{}
}

// ChallengeSubmissionResult 描述链上提交执行结果。
type ChallengeSubmissionResult struct {
	IsCorrect        bool
	AssertionResults *dto.VerificationAssertionResults
	ErrorMessage     *string
}

// ADRuntimeGroupSpec 描述一个攻防赛分组所需的链运行时编排输入。
type ADRuntimeGroupSpec struct {
	CompetitionID    int64
	GroupID          int64
	Namespace        string
	JudgeChainImage  string
	TeamChainImage   string
	RuntimeToolImage string
	Teams            []ADRuntimeTeamSpec
}

// ADRuntimeTeamSpec 描述单支队伍链初始化所需的部署数据。
type ADRuntimeTeamSpec struct {
	TeamID    int64
	TeamName  string
	Contracts []ADRuntimeContractSpec
}

// ADRuntimeContractSpec 描述队伍链上的单个合约部署任务。
type ADRuntimeContractSpec struct {
	ChallengeID     int64
	ChallengeTitle  string
	ContractName    string
	ABIJSON         string
	Bytecode        string
	ConstructorArgs []interface{}
	DeployOrder     int
}

// ADRuntimeGroupResult 描述分组运行时编排完成后的真实资源结果。
type ADRuntimeGroupResult struct {
	JudgeChainURL        *string
	JudgeContractAddress *string
	Teams                []ADRuntimeTeamResult
}

// ADRuntimeTeamResult 描述单支队伍链的真实访问地址与合约部署结果。
type ADRuntimeTeamResult struct {
	TeamID              int64
	ChainRPCURL         *string
	ChainWSURL          *string
	DeployedContracts   []dto.TeamChainContractItem
	CurrentPatchVersion int
	Status              int16
}

// ADAttackExecutionSpec 描述攻防赛攻击代理执行输入。
type ADAttackExecutionSpec struct {
	Namespace    string
	ChainRPCURL  string
	AttackTxData string
	Contracts    []dto.TeamChainContractItem
	Assertions   []ChallengeAssertionSpec
}

// ADAttackExecutionResult 描述攻防赛攻击代理执行结果。
type ADAttackExecutionResult struct {
	IsSuccessful     bool
	AssertionResults *dto.VerificationAssertionResults
	ErrorMessage     *string
}

// ADPatchVerificationSpec 描述攻防赛补丁验证输入。
type ADPatchVerificationSpec struct {
	Namespace          string
	ChallengeID        int64
	ChallengeTitle     string
	ChainRPCURL        string
	PatchSourceCode    string
	OriginalContracts  []ChallengeRuntimeContractSpec
	TargetContracts    []dto.TeamChainContractItem
	Assertions         []ChallengeAssertionSpec
	OfficialPocContent string
}

// ADPatchVerificationResult 描述攻防赛补丁验证输出。
type ADPatchVerificationResult struct {
	FunctionalityPassed bool
	VulnerabilityFixed  bool
	RejectionReason     *string
	PatchedContracts    []dto.TeamChainContractItem
}

// RuntimeDeployPodRequest 描述模块05运行时 Pod 部署输入。
type RuntimeDeployPodRequest struct {
	Namespace  string
	PodName    string
	Containers []RuntimeContainerSpec
	Labels     map[string]string
}

// RuntimeContainerSpec 描述模块05运行时使用的容器规格。
type RuntimeContainerSpec struct {
	Name        string
	Image       string
	Ports       []RuntimePortSpec
	EnvVars     map[string]string
	CPULimit    string
	MemoryLimit string
	Command     []string
}

// RuntimePortSpec 描述运行时 Service 暴露端口。
type RuntimePortSpec struct {
	ContainerPort int
	Protocol      string
	ServicePort   int
}

// RuntimeDeployPodResponse 描述运行时 Pod 部署结果。
type RuntimeDeployPodResponse struct {
	PodName    string
	Namespace  string
	InternalIP string
	Status     string
}

// RuntimeExecResult 描述容器命令执行结果。
type RuntimeExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// RuntimePodStatus 描述运行时 Pod 状态。
type RuntimePodStatus struct {
	PodName    string
	Namespace  string
	NodeName   string
	Status     string
	Reason     string
	Message    string
	InternalIP string
}

// buildRuntimeAssertions 把题目断言实体转换为运行时执行规格。
func buildRuntimeAssertions(assertions []*entity.ChallengeAssertion) []ChallengeAssertionSpec {
	items := make([]ChallengeAssertionSpec, 0, len(assertions))
	for _, assertion := range assertions {
		extra := map[string]interface{}{}
		_ = decodeJSON(assertion.ExtraParams, &extra)
		items = append(items, ChallengeAssertionSpec{
			AssertionType: assertion.AssertionType,
			Target:        assertion.Target,
			Operator:      assertion.Operator,
			ExpectedValue: assertion.ExpectedValue,
			ExtraParams:   extra,
		})
	}
	return items
}

// buildChallengeEnvironmentSpec 根据题目配置和合约列表构建运行时环境规格。
func buildChallengeEnvironmentSpec(challenge *entity.Challenge, competitionID, teamID int64, namespace string, contracts []*entity.ChallengeContract) *ChallengeEnvironmentSpec {
	spec := &ChallengeEnvironmentSpec{
		CompetitionID: competitionID,
		ChallengeID:   challenge.ID,
		TeamID:        teamID,
		Namespace:     namespace,
		RuntimeMode:   challenge.RuntimeMode,
		Contracts:     buildChallengeRuntimeContractSpecs(contracts),
	}
	if challenge != nil {
		if chainCfg := loadChallengeChainConfig(challenge); chainCfg != nil {
			spec.ChainConfig = chainCfg
		}
		if len(challenge.SetupTransactions) > 0 {
			var setupTransactions []dto.ChallengeSetupTransaction
			if err := decodeJSON(challenge.SetupTransactions, &setupTransactions); err == nil {
				spec.SetupTransactions = setupTransactions
			}
		}
	}
	if challenge != nil && len(challenge.EnvironmentConfig) > 0 {
		var envCfg dto.ChallengeEnvironmentConfig
		if err := decodeJSON(challenge.EnvironmentConfig, &envCfg); err == nil {
			spec.EnvironmentConfig = &envCfg
		}
	}
	return spec
}

// loadChallengeAccounts 解析题目链预置账户，供运行时脚本识别账户别名。
func loadChallengeAccounts(challenge *entity.Challenge) []dto.ChallengeChainAccount {
	if challenge == nil || len(challenge.ChainConfig) == 0 {
		return nil
	}
	var cfg dto.ChallengeChainConfig
	if err := decodeJSON(challenge.ChainConfig, &cfg); err != nil {
		return nil
	}
	return cfg.Accounts
}

// listChallengeRuntimeContracts 读取题目合约并转换为运行时部署规格。
func listChallengeRuntimeContracts(ctx context.Context, contractRepo ctfrepo.ChallengeContractRepository, challengeID int64) ([]ChallengeRuntimeContractSpec, error) {
	contracts, err := contractRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	return buildChallengeRuntimeContractSpecs(contracts), nil
}

// buildChallengeRuntimeContractSpecs 将题目合约实体映射为运行时部署规格。
func buildChallengeRuntimeContractSpecs(contracts []*entity.ChallengeContract) []ChallengeRuntimeContractSpec {
	items := make([]ChallengeRuntimeContractSpec, 0, len(contracts))
	for _, contract := range contracts {
		constructorArgs := []interface{}{}
		_ = decodeJSON(contract.ConstructorArgs, &constructorArgs)
		items = append(items, ChallengeRuntimeContractSpec{
			ChallengeID:     contract.ChallengeID,
			ContractName:    contract.Name,
			ABIJSON:         string(contract.ABI),
			Bytecode:        contract.Bytecode,
			ConstructorArgs: constructorArgs,
			DeployOrder:     contract.DeployOrder,
		})
	}
	return items
}

// challengeEnvironmentRuntimeState 表示题目环境运行时快照。
type challengeEnvironmentRuntimeState struct {
	Containers map[string]dto.ChallengeEnvironmentContainerState `json:"containers"`
	Contracts  []ChallengeRuntimeContractBinding                 `json:"contracts"`
}

// buildChallengeEnvironmentRuntimeState 构建可持久化的题目环境运行时快照。
func buildChallengeEnvironmentRuntimeState(result *ChallengeEnvironmentResult) challengeEnvironmentRuntimeState {
	state := challengeEnvironmentRuntimeState{
		Containers: map[string]dto.ChallengeEnvironmentContainerState{},
		Contracts:  []ChallengeRuntimeContractBinding{},
	}
	if result == nil {
		return state
	}
	if result.ContainerStatus != nil {
		state.Containers = result.ContainerStatus
	}
	if result.Contracts != nil {
		state.Contracts = result.Contracts
	}
	return state
}

// decodeChallengeEnvironmentRuntimeState 解析题目环境运行时快照。
func decodeChallengeEnvironmentRuntimeState(environment *entity.ChallengeEnvironment) challengeEnvironmentRuntimeState {
	state := challengeEnvironmentRuntimeState{
		Containers: map[string]dto.ChallengeEnvironmentContainerState{},
		Contracts:  []ChallengeRuntimeContractBinding{},
	}
	if environment == nil || len(environment.ContainerStatus) == 0 {
		return state
	}
	_ = decodeJSON(environment.ContainerStatus, &state)
	return state
}
