// 模块：sim-engine/sdk/go/scenario
// 文件职责：场景容器进程启动入口 — 把 Scenario 实例挂上 gRPC 服务并监听端口。
// 协议依据：proto/lenschain/sim_scenario/v1/sim_scenario.proto。
//
// 教师写场景的最简流程（详 doc.go）：
//
//   func main() {
//       ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//       defer cancel()
//       _ = scenario.Run(ctx, MyScenario{})
//   }
//
// 平台内部场景通过 framework.Definition 接入：
//
//   runtime, _ := scenario.NewRuntimeScenario(myDefinition)
//   _ = scenario.Run(ctx, runtime)

package scenario

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
	"google.golang.org/grpc"
)

// 默认监听地址 — 与 deploy/docker/scenario-base.Dockerfile 的 ENV / EXPOSE 50100 一致。
//
// 场景容器在 K8s 内通过 ClusterIP Service + NetworkPolicy 与 sim-engine Core 隔离通讯；
// 默认值仅在容器外开发调试时生效，生产环境一律由 scenario-base 镜像注入 SCENARIO_LISTEN_ADDR=":50100"。
const defaultListenAddr = ":50100"

// 监听地址环境变量名（与 deploy/docker/scenario-base.Dockerfile ENV 同名）。
const envListenAddr = "SCENARIO_LISTEN_ADDR"

// ServeConfig 描述 sdk 启动 gRPC 服务的参数。
type ServeConfig struct {
	// ListenAddr 是 gRPC 服务监听地址，例如 ":50100"。
	// 不调用 Run 而直接调用 Serve 时由调用方提供；为空将返回错误。
	ListenAddr string
}

// Run 是 sdk 推荐的启动入口：用默认配置启动场景 gRPC 服务并监听 ctx 取消信号。
//
// 内部按以下规则解析监听地址：
//  1. 环境变量 SCENARIO_LISTEN_ADDR（部署侧由 scenario-base.Dockerfile ENV 注入 ":50100"）
//  2. defaultListenAddr（":50100"，与 scenario-base 镜像 EXPOSE 一致）
//
// 任意失败立即返回错误；ctx 取消时优雅停止 gRPC 服务并返回 nil。
func Run(ctx context.Context, scenario Scenario) error {
	addr := strings.TrimSpace(os.Getenv(envListenAddr))
	if addr == "" {
		addr = defaultListenAddr
	}
	return Serve(ctx, scenario, ServeConfig{ListenAddr: addr})
}

// Serve 启动场景 gRPC 服务并在 ctx 取消时优雅停止。
//
// 与 Run 的区别：调用方显式提供 ListenAddr，跳过环境变量回退；适合需要自定义监听地址的场景。
//
// 同时启动内置健康检查 HTTP 服务（/healthz / /readyz）：
//   - 监听地址由环境变量 SCENARIO_HEALTH_ADDR 控制，默认 :50101（详 §0.7.1 C24）
//   - 在 gRPC 监听绑定成功后立即 MarkReady（保证 §0.7.1 C39 ≤ 10s ready）
func Serve(ctx context.Context, scenario Scenario, config ServeConfig) error {
	if scenario == nil {
		return errors.New("scenario 不能为空")
	}
	addr := strings.TrimSpace(config.ListenAddr)
	if addr == "" {
		return errors.New("ListenAddr 不能为空")
	}

	// 启动健康检查 HTTP 服务（独立端口，与 gRPC 错开）。
	health := newHealthServer(resolveHealthAddr())
	if err := health.Start(ctx); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := grpc.NewServer()
	simscenariov1.RegisterSimScenarioServiceServer(server, NewServer(scenario))

	// gRPC 监听绑定成功 → 标记 ready（K8s readinessProbe 立刻通过）。
	health.MarkReady()

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		server.GracefulStop()
	}()

	err = server.Serve(listener)
	if ctx.Err() != nil {
		<-stopped
		return nil
	}
	return err
}
