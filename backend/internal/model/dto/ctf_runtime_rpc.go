// ctf_runtime_rpc.go
// 模块05 — CTF竞赛：judge-service 与 patch-verifier 微服务 HTTP 通信 DTO。
// 该文件定义后端调用 judge-service 和 patch-verifier 微服务时使用的请求/响应结构体。
// 这些类型是微服务 API 契约的客户端映射，不应在 handler 层直接暴露给前端。

package dto

// ── judge-service 合约部署 ────────────────────────────────────────

// JudgeContractBinding judge-service 返回的合约绑定信息。
type JudgeContractBinding struct {
	ChallengeID  string `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	ABIJSON      string `json:"abi_json,omitempty"`
	PatchVersion int    `json:"patch_version"`
	IsPatched    bool   `json:"is_patched"`
}

// JudgeDeployContractSpec judge-service 合约部署规格。
type JudgeDeployContractSpec struct {
	ChallengeID     int64         `json:"challenge_id"`
	ContractName    string        `json:"contract_name"`
	ABIJSON         string        `json:"abi_json"`
	Bytecode        string        `json:"bytecode"`
	ConstructorArgs []interface{} `json:"constructor_args"`
	DeployOrder     int           `json:"deploy_order"`
}

// JudgeDeployRequest judge-service 部署请求。
type JudgeDeployRequest struct {
	RPCURL    string                    `json:"rpc_url"`
	Contracts []JudgeDeployContractSpec `json:"contracts"`
}

// JudgeDeployResponse judge-service 部署响应。
type JudgeDeployResponse struct {
	Contracts []JudgeContractBinding `json:"contracts"`
}

// ── judge-service 攻击执行 ────────────────────────────────────────

// JudgeAssertionSpec judge-service 断言规格。
type JudgeAssertionSpec struct {
	AssertionType string                 `json:"assertion_type"`
	Target        string                 `json:"target,omitempty"`
	Operator      string                 `json:"operator,omitempty"`
	ExpectedValue string                 `json:"expected_value,omitempty"`
	ExtraParams   map[string]interface{} `json:"extra_params,omitempty"`
}

// JudgeAttackRequest judge-service 攻击执行请求。
type JudgeAttackRequest struct {
	RPCURL        string                 `json:"rpc_url"`
	Submission    string                 `json:"submission"`
	Contracts     []JudgeContractBinding `json:"contracts"`
	Assertions    []JudgeAssertionSpec   `json:"assertions"`
	DefaultTarget string                 `json:"default_target"`
}

// JudgeAttackResponse judge-service 攻击执行响应。
type JudgeAttackResponse struct {
	AllPassed       bool                           `json:"all_passed"`
	Results         []VerificationAssertionResult  `json:"results"`
	ExecutionTimeMS *int                           `json:"execution_time_ms,omitempty"`
	TxHash          *string                        `json:"tx_hash,omitempty"`
	ErrorMessage    *string                        `json:"error_message,omitempty"`
}

// ── patch-verifier 补丁验证 ───────────────────────────────────────

// VerifierContractSpec patch-verifier 原始合约。
type VerifierContractSpec struct {
	ChallengeID  int64  `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	ABIJSON      string `json:"abi_json"`
	Bytecode     string `json:"bytecode"`
	DeployOrder  int    `json:"deploy_order"`
}

// VerifierContractBinding patch-verifier 已部署合约。
type VerifierContractBinding struct {
	ChallengeID  string `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	PatchVersion int    `json:"patch_version"`
	IsPatched    bool   `json:"is_patched"`
}

// VerifierRequest patch-verifier 补丁验证请求。
type VerifierRequest struct {
	RPCURL            string                    `json:"rpc_url"`
	ChallengeID       int64                     `json:"challenge_id"`
	ChallengeTitle    string                    `json:"challenge_title"`
	PatchSourceCode   string                    `json:"patch_source_code"`
	OriginalContracts []VerifierContractSpec    `json:"original_contracts"`
	TargetContracts   []VerifierContractBinding `json:"target_contracts"`
	Assertions        []JudgeAssertionSpec      `json:"assertions"`
	OfficialPoc       string                    `json:"official_poc"`
}

// VerifierResponse patch-verifier 补丁验证响应。
type VerifierResponse struct {
	FunctionalityPassed bool                      `json:"functionality_passed"`
	VulnerabilityFixed  bool                      `json:"vulnerability_fixed"`
	RejectionReason     *string                   `json:"rejection_reason,omitempty"`
	PatchedContracts    []VerifierContractBinding `json:"patched_contracts"`
}
