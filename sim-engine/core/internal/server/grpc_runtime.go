package server

import (
	"net"

	simenginev1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_engine/v1"
	"google.golang.org/grpc"

	"github.com/lenschain/sim-engine/core/internal/app"
	"github.com/lenschain/sim-engine/core/internal/grpcserver"
)

// GRPCRuntime 封装 SimEngine Core 的远程过程调用服务监听器。
type GRPCRuntime struct {
	server *grpc.Server
}

// NewGRPCRuntime 创建远程过程调用运行时并注册 SimEngine 控制面服务。
func NewGRPCRuntime(engine *app.Engine, publicBase string) *GRPCRuntime {
	srv := grpc.NewServer()
	simenginev1.RegisterSimEngineServiceServer(srv, grpcserver.NewSimEngineServer(engine, publicBase))
	return &GRPCRuntime{server: srv}
}

// Serve 启动远程过程调用服务。
func (r *GRPCRuntime) Serve(listener net.Listener) error {
	return r.server.Serve(listener)
}

// Stop 停止远程过程调用服务。
func (r *GRPCRuntime) Stop() {
	r.server.Stop()
}
