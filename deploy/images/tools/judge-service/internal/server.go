// Package internal 实现 judge-service 的 HTTP 服务器与请求处理。
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

// ── 配置 ──────────────────────────────────────────────────────

// Config 服务配置。
type Config struct {
	ListenAddr     string
	AuthToken      string
	RequestTimeout time.Duration
}

// LoadConfig 从环境变量加载配置。
func LoadConfig() *Config {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8090"
	}
	timeout := 60 * time.Second
	if v := os.Getenv("REQUEST_TIMEOUT_SECONDS"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil && d > 0 {
			timeout = d
		}
	}
	return &Config{
		ListenAddr:     addr,
		AuthToken:      os.Getenv("AUTH_TOKEN"),
		RequestTimeout: timeout,
	}
}

// ── 服务器 ─────────────────────────────────────────────────────

// Server judge-service HTTP 服务器。
type Server struct {
	cfg *Config
}

// NewServer 创建服务实例。
func NewServer(cfg *Config) *Server {
	return &Server{cfg: cfg}
}

// Handler 返回路由 HTTP Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/v1/contracts/deploy", s.withMiddleware(s.handleDeployContracts))
	mux.HandleFunc("POST /api/v1/attacks/execute", s.withMiddleware(s.handleExecuteAttack))
	mux.HandleFunc("POST /api/v1/transactions/setup", s.withMiddleware(s.handleSetupTransactions))
	return mux
}

// ── 中间件 ─────────────────────────────────────────────────────

// withMiddleware 为处理函数添加认证检查、请求体大小限制和请求超时控制。
func (s *Server) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 认证
		if s.cfg.AuthToken != "" {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+s.cfg.AuthToken {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
		}
		// 请求体大小限制
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		// 超时
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
		defer cancel()
		next(w, r.WithContext(ctx))
	}
}

// ── 请求/响应类型 ──────────────────────────────────────────────

// ContractSpec 合约部署规格。
type ContractSpec struct {
	ChallengeID     int64         `json:"challenge_id"`
	ContractName    string        `json:"contract_name"`
	ABIJSON         string        `json:"abi_json"`
	Bytecode        string        `json:"bytecode"`
	ConstructorArgs []interface{} `json:"constructor_args"`
	DeployOrder     int           `json:"deploy_order"`
}

// ContractBinding 已部署合约绑定信息。
type ContractBinding struct {
	ChallengeID  string `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	ABIJSON      string `json:"abi_json,omitempty"`
	PatchVersion int    `json:"patch_version"`
	IsPatched    bool   `json:"is_patched"`
}

// DeployRequest 合约部署请求。
type DeployRequest struct {
	RPCURL    string         `json:"rpc_url"`
	Contracts []ContractSpec `json:"contracts"`
}

// DeployResponse 合约部署响应。
type DeployResponse struct {
	Contracts []ContractBinding `json:"contracts"`
}

// AssertionSpec 断言规格。
type AssertionSpec struct {
	AssertionType string                 `json:"assertion_type"`
	Target        string                 `json:"target,omitempty"`
	Operator      string                 `json:"operator,omitempty"`
	ExpectedValue string                 `json:"expected_value,omitempty"`
	ExtraParams   map[string]interface{} `json:"extra_params,omitempty"`
}

// AssertionResult 断言执行结果。
type AssertionResult struct {
	Type     string `json:"type"`
	Target   string `json:"target,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// AttackRequest 攻击执行请求。
type AttackRequest struct {
	RPCURL        string            `json:"rpc_url"`
	Submission    string            `json:"submission"`
	Contracts     []ContractBinding `json:"contracts"`
	Assertions    []AssertionSpec   `json:"assertions"`
	DefaultTarget string            `json:"default_target"`
}

// AttackResponse 攻击执行响应。
type AttackResponse struct {
	AllPassed       bool             `json:"all_passed"`
	Results         []AssertionResult `json:"results"`
	ExecutionTimeMS int              `json:"execution_time_ms"`
	TxHash          *string          `json:"tx_hash"`
	ErrorMessage    *string          `json:"error_message"`
}

// SetupTx 初始化交易。
type SetupTx struct {
	From     string        `json:"from,omitempty"`
	To       string        `json:"to,omitempty"`
	Function string        `json:"function,omitempty"`
	Args     []interface{} `json:"args,omitempty"`
	Value    string        `json:"value,omitempty"`
}

// SetupRequest 初始化交易请求。
type SetupRequest struct {
	RPCURL                string            `json:"rpc_url"`
	RuntimeMode           int16             `json:"runtime_mode"`
	Accounts              []AccountSpec     `json:"accounts"`
	Contracts             []ContractBinding `json:"contracts"`
	SetupTransactions     []SetupTx         `json:"setup_transactions"`
	ImpersonatedAccounts  []string          `json:"impersonated_accounts"`
	PinnedContracts       []PinnedContract  `json:"pinned_contracts"`
}

// AccountSpec 账户。
type AccountSpec struct {
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
}

// PinnedContract 外部合约固定引用。
type PinnedContract struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// SetupResponse 初始化交易响应。
type SetupResponse struct {
	Applied int `json:"applied"`
}

// ── Handlers ───────────────────────────────────────────────────

// handleHealthz 健康检查端点，返回服务状态。
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

// handleDeployContracts 处理合约部署请求。
func (s *Server) handleDeployContracts(w http.ResponseWriter, r *http.Request) {
	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.RPCURL == "" {
		writeError(w, http.StatusBadRequest, "rpc_url is required")
		return
	}
	result, err := DeployContracts(r.Context(), req.RPCURL, req.Contracts)
	if err != nil {
		log.Printf("[deploy] error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, &DeployResponse{Contracts: result})
}

// handleExecuteAttack 处理攻击执行请求，执行攻击交易并验证断言。
func (s *Server) handleExecuteAttack(w http.ResponseWriter, r *http.Request) {
	var req AttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.RPCURL == "" {
		writeError(w, http.StatusBadRequest, "rpc_url is required")
		return
	}
	resp := ExecuteAttack(r.Context(), &req)
	writeJSON(w, http.StatusOK, resp)
}

// handleSetupTransactions 处理初始化交易回放请求。
func (s *Server) handleSetupTransactions(w http.ResponseWriter, r *http.Request) {
	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.RPCURL == "" {
		writeError(w, http.StatusBadRequest, "rpc_url is required")
		return
	}
	applied, err := ExecuteSetupTransactions(r.Context(), &req)
	if err != nil {
		log.Printf("[setup] error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, &SetupResponse{Applied: applied})
}

// ── 响应辅助 ───────────────────────────────────────────────────

// writeJSON 将响应以 JSON 格式写入 HTTP 响应。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将错误信息以 JSON 格式写入 HTTP 响应。
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// stringPtr 返回字符串指针。
func stringPtr(s string) *string { return &s }

// fmtInt64 int64 转字符串。
func fmtInt64(v int64) string { return fmt.Sprintf("%d", v) }
