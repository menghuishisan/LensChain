// Package main 启动混合实验 Collector Agent sidecar。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lenschain/sim-engine/core/internal/collectoragent"
)

// main 从环境变量加载 sidecar 配置并启动采集循环。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := collectoragent.ParseConfig(
		os.Getenv("COLLECTOR_TARGET_CONTAINER"),
		os.Getenv("COLLECTOR_ECOSYSTEM"),
		os.Getenv("COLLECTOR_SESSION_ID"),
		os.Getenv("COLLECTOR_CORE_WS_URL"),
		os.Getenv("COLLECTOR_CONFIG_JSON"),
	)
	if err != nil {
		log.Fatalf("parse collector config: %v", err)
	}

	agent, err := collectoragent.New(cfg)
	if err != nil {
		log.Fatalf("create collector agent: %v", err)
	}
	if err := agent.Run(ctx); err != nil {
		log.Fatalf("collector agent stopped: %v", err)
	}
}
