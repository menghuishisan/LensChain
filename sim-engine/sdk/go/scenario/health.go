// 模块：sim-engine/sdk/go/scenario
// 文件职责：场景容器内置 HTTP 健康检查端点（/healthz / /readyz）。
// 协议依据：AGENTS.md §0.7.1 C24 / C39；06.md §17 性能预算（≤ 10s ready）。
//
// 设计要点：
// 1. /healthz：进程存活探针；HTTP 服务起来即 200。K8s livenessProbe 用。
// 2. /readyz：就绪探针；只有 sdk.Run 内部 gRPC 服务监听成功后才 200。K8s readinessProbe 用。
// 3. 监听端口由环境变量 SCENARIO_HEALTH_ADDR 控制，默认 :50101（与 gRPC :50100 错开）。
// 4. 不依赖任何第三方 HTTP 框架；仅 net/http stdlib，零额外开销。

package scenario

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// 默认健康检查 HTTP 监听地址（与 gRPC :50100 错开）。
const defaultHealthAddr = ":50101"

// 监听地址环境变量名（与 deploy/docker/scenario-base.Dockerfile ENV 同名）。
const envHealthAddr = "SCENARIO_HEALTH_ADDR"

// healthState 是 healthServer 的状态；ready 由 sdk.Serve 在 gRPC 监听成功后置 1。
type healthState struct {
	ready atomic.Bool // 0 = starting, 1 = ready
}

// healthServer 提供 /healthz / /readyz 两个端点。
type healthServer struct {
	state  *healthState
	server *http.Server
}

// newHealthServer 构造 healthServer 但不启动；调用方需调用 Start 启动。
func newHealthServer(addr string) *healthServer {
	state := &healthState{}
	mux := http.NewServeMux()
	// /healthz 仅检查进程存活（HTTP 服务起来 = 进程活）。
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	// /readyz 检查 gRPC 监听是否已就绪。
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if state.ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"starting"}`))
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	return &healthServer{state: state, server: srv}
}

// Start 在后台启动 HTTP 服务；ctx 取消时优雅停止。
func (h *healthServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		err := h.server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.server.Shutdown(shutdownCtx)
	}()
	// 给 ListenAndServe 一点时间立刻报错（如端口被占）。
	select {
	case err := <-errCh:
		return fmt.Errorf("健康检查 HTTP 服务启动失败: %w", err)
	case <-time.After(80 * time.Millisecond):
		return nil
	}
}

// MarkReady 把 readiness 置为 ready；sdk.Serve 在 gRPC 监听成功后调用。
func (h *healthServer) MarkReady() {
	h.state.ready.Store(true)
}

// resolveHealthAddr 按环境变量 / 默认值解析健康检查地址。
//
// 监听地址留空（""）视为禁用健康检查（开发场景）。
func resolveHealthAddr() string {
	addr := strings.TrimSpace(os.Getenv(envHealthAddr))
	if addr == "" {
		addr = defaultHealthAddr
	}
	return addr
}
