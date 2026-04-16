package scenario

import (
	"context"
	"errors"
	"net"
	"strings"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
	"google.golang.org/grpc"
)

// ServeConfig 描述 Go 场景开发包服务的启动参数。
type ServeConfig struct {
	ListenAddr string
}

// Serve 启动场景远程过程调用服务，并在上下文取消时优雅停止。
func Serve(ctx context.Context, scenario Scenario, config ServeConfig) error {
	if scenario == nil {
		return errors.New("scenario is required")
	}
	if strings.TrimSpace(config.ListenAddr) == "" {
		return errors.New("listen address is required")
	}

	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := grpc.NewServer()
	simscenariov1.RegisterSimScenarioServiceServer(server, NewServer(scenario))

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
