// Package main 启动 SimEngine Core 运行时服务。
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lenschain/sim-engine/core/internal/app"
	"github.com/lenschain/sim-engine/core/internal/scene"
	"github.com/lenschain/sim-engine/core/internal/server"
)

// main 启动 SimEngine Core 的 HTTP 与远程过程调用服务。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpAddr := os.Getenv("SIM_ENGINE_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8090"
	}
	grpcAddr := os.Getenv("SIM_ENGINE_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":9090"
	}
	publicBase := os.Getenv("SIM_ENGINE_PUBLIC_BASE")
	if publicBase == "" {
		publicBase = "ws://127.0.0.1:8090"
	}
	wsJWTSecret := os.Getenv("SIM_ENGINE_WS_JWT_SECRET")
	wsJWTIssuer := os.Getenv("SIM_ENGINE_WS_JWT_ISSUER")
	wsJWTAudience := os.Getenv("SIM_ENGINE_WS_JWT_AUDIENCE")
	endpointsConfig := os.Getenv("SIM_ENGINE_SCENE_ENDPOINTS")
	endpoints, err := scene.ParseEndpointsConfig(endpointsConfig)
	if err != nil {
		log.Fatalf("parse scene endpoints: %v", err)
	}
	storageConfig, err := app.ParseObjectStorageConfigFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("parse object storage config: %v", err)
	}

	engine := app.NewEngine(scene.NewGRPCClientFactory(endpoints))
	storageCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := app.NewMinIOSnapshotStore(storageCtx, storageConfig)
	if err != nil {
		log.Fatalf("create snapshot store: %v", err)
	}
	engine.SetSnapshotStore(store)
	go engine.StartClockLoop(ctx, 100*time.Millisecond)
	go engine.StartTeacherSummaryLoop(ctx, 5*time.Second)
	go engine.StartAutoSnapshotLoop(ctx, 5*time.Minute)
	validator := server.NewDefaultTokenValidator(engine, wsJWTSecret, wsJWTIssuer, wsJWTAudience)

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           server.NewHandlerWithValidator(engine, validator),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}
	grpcRuntime := server.NewGRPCRuntime(engine, publicBase)

	go func() {
		log.Printf("SimEngine Core gRPC listening on %s", grpcAddr)
		if err := grpcRuntime.Serve(grpcListener); err != nil {
			log.Fatalf("SimEngine Core gRPC stopped: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		grpcRuntime.Stop()
	}()

	log.Printf("SimEngine Core HTTP listening on %s", httpAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("SimEngine Core HTTP stopped: %v", err)
	}
}
