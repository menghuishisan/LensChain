// Package main 启动 SimEngine Core 运行时服务。
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/lenschain/sim-engine/core/internal/app"
	"github.com/lenschain/sim-engine/core/internal/config"
	"github.com/lenschain/sim-engine/core/internal/scene"
	"github.com/lenschain/sim-engine/core/internal/server"
)

// main 启动 SimEngine Core 的 HTTP 与远程过程调用服务。
// 配置统一从 yaml 文件加载（路径可由 -config 参数或 LENSCHAIN_SIM_* 环境变量覆盖），
// 与 backend 的配置加载方式保持一致。
func main() {
	configPath := flag.String("config", "", "配置文件路径，留空时按默认搜索路径定位 configs/config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storageCfg := app.ObjectStorageConfig{
		Endpoint:        cfg.ObjectStorage.Endpoint,
		AccessKey:       cfg.ObjectStorage.AccessKey,
		SecretKey:       cfg.ObjectStorage.SecretKey,
		UseSSL:          cfg.ObjectStorage.UseSSL,
		Bucket:          cfg.ObjectStorage.Bucket,
		Region:          cfg.ObjectStorage.Region,
		ObjectPrefix:    cfg.ObjectStorage.ObjectPrefix,
		EncryptionKey:   cfg.ObjectStorage.EncryptionKey,
		PresignDuration: cfg.ObjectStorage.PresignDuration,
	}
	if err := app.ValidateObjectStorageConfig(storageCfg); err != nil {
		log.Fatalf("invalid object_storage config: %v", err)
	}

	engine := app.NewEngine(scene.NewGRPCClientFactory(cfg.Scene.Endpoints))
	storageCtx, cancel := context.WithTimeout(context.Background(), cfg.Snapshot.InitTimeout)
	defer cancel()
	store, err := app.NewMinIOSnapshotStore(storageCtx, storageCfg)
	if err != nil {
		log.Fatalf("create snapshot store: %v", err)
	}
	engine.SetSnapshotStore(store)
	go engine.StartClockLoop(ctx, cfg.Loop.ClockInterval)
	go engine.StartTeacherSummaryLoop(ctx, cfg.Loop.TeacherSummaryInterval)
	go engine.StartAutoSnapshotLoop(ctx, cfg.Loop.AutoSnapshotInterval)

	validator := server.NewDefaultTokenValidator(engine, cfg.Auth.WSJWTSecret, cfg.Auth.WSJWTIssuer, cfg.Auth.WSJWTAudience)

	srv := &http.Server{
		Addr:              cfg.Server.HTTPAddr,
		Handler:           server.NewHandlerWithValidator(engine, validator),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	grpcListener, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}
	grpcRuntime := server.NewGRPCRuntime(engine, cfg.Server.PublicBase)

	go func() {
		log.Printf("SimEngine Core gRPC listening on %s", cfg.Server.GRPCAddr)
		if err := grpcRuntime.Serve(grpcListener); err != nil {
			log.Fatalf("SimEngine Core gRPC stopped: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		grpcRuntime.Stop()
	}()

	if len(cfg.Scene.Endpoints) == 0 {
		log.Printf("warning: scene.endpoints 为空，引擎已启动但任何场景请求都将返回 'scene endpoint is not configured'")
	}
	log.Printf("SimEngine Core HTTP listening on %s", cfg.Server.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("SimEngine Core HTTP stopped: %v", err)
	}
}
