// Package internal 实现 patch-verifier 的 HTTP 服务器与请求处理。
package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

const maxRequestBodyBytes = 2 << 20 // 2 MB

// ── 配置 ──────────────────────────────────────────────────────

// Config 服务配置。
type Config struct {
	ListenAddr     string
	AuthToken      string
	SolcPath       string
	RequestTimeout time.Duration
}

// LoadConfig 从环境变量加载配置。
func LoadConfig() *Config {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8091"
	}
	solcPath := os.Getenv("SOLC_PATH")
	if solcPath == "" {
		solcPath = "/usr/local/bin/solc"
	}
	timeout := 120 * time.Second
	if v := os.Getenv("REQUEST_TIMEOUT_SECONDS"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil && d > 0 {
			timeout = d
		}
	}
	return &Config{
		ListenAddr:     addr,
		AuthToken:      os.Getenv("AUTH_TOKEN"),
		SolcPath:       solcPath,
		RequestTimeout: timeout,
	}
}

// ── 服务器 ─────────────────────────────────────────────────────

// Server patch-verifier HTTP 服务器。
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
	mux.HandleFunc("POST /api/v1/patches/verify", s.withMiddleware(s.handleVerifyPatch))
	return mux
}

// ── 中间件 ─────────────────────────────────────────────────────

// withMiddleware 为处理函数添加认证检查、请求体大小限制和请求超时控制。
func (s *Server) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.AuthToken != "" {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+s.cfg.AuthToken {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
		defer cancel()
		next(w, r.WithContext(ctx))
	}
}

// ── 请求/响应类型 ──────────────────────────────────────────────

// ContractSpec 原始合约。
type ContractSpec struct {
	ChallengeID  int64  `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	ABIJSON      string `json:"abi_json"`
	Bytecode     string `json:"bytecode"`
	DeployOrder  int    `json:"deploy_order"`
}

// ContractBinding 已部署合约。
type ContractBinding struct {
	ChallengeID  string `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	PatchVersion int    `json:"patch_version"`
	IsPatched    bool   `json:"is_patched"`
}

// AssertionSpec 断言。
type AssertionSpec struct {
	AssertionType string                 `json:"assertion_type"`
	Target        string                 `json:"target,omitempty"`
	Operator      string                 `json:"operator,omitempty"`
	ExpectedValue string                 `json:"expected_value,omitempty"`
	ExtraParams   map[string]interface{} `json:"extra_params,omitempty"`
}

// VerifyRequest 补丁验证请求。
type VerifyRequest struct {
	RPCURL            string            `json:"rpc_url"`
	ChallengeID       int64             `json:"challenge_id"`
	ChallengeTitle    string            `json:"challenge_title"`
	PatchSourceCode   string            `json:"patch_source_code"`
	OriginalContracts []ContractSpec    `json:"original_contracts"`
	TargetContracts   []ContractBinding `json:"target_contracts"`
	Assertions        []AssertionSpec   `json:"assertions"`
	OfficialPoc       string            `json:"official_poc"`
}

// VerifyResponse 补丁验证响应。
type VerifyResponse struct {
	FunctionalityPassed bool              `json:"functionality_passed"`
	VulnerabilityFixed  bool              `json:"vulnerability_fixed"`
	RejectionReason     *string           `json:"rejection_reason"`
	PatchedContracts    []ContractBinding `json:"patched_contracts"`
}

// ── Handlers ───────────────────────────────────────────────────

// handleHealthz 健康检查端点，返回服务状态。
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

// handleVerifyPatch 处理补丁验证请求，调用 VerifyPatch 执行完整验证流程。
func (s *Server) handleVerifyPatch(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.RPCURL == "" || req.PatchSourceCode == "" {
		writeError(w, http.StatusBadRequest, "rpc_url and patch_source_code are required")
		return
	}
	resp, err := VerifyPatch(r.Context(), s.cfg, &req)
	if err != nil {
		log.Printf("[verify] error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
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
