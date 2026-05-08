// Package main 是链镜平台 43 个内置仿真场景的共享运行时入口。
//
// 设计说明：
//   - 所有内置场景共享同一个二进制（`scenarios/runtime:v1.0.0`）。
//   - 启动时通过 SCENE_CODE 环境变量选择要运行哪个场景定义，
//     由 SimEngine SceneManager（K8sOrchestrator）在创建场景 Pod 时注入。
//   - 监听地址通过 SCENARIO_LISTEN_ADDR 注入（容器内默认 :50100）。
//
// 教师自定义场景应通过 FROM registry.lianjing.com/lenschain/scenario-base:v1.0.0
// 构建独立镜像，可以直接复用或借鉴本入口的实现模式。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

func main() {
	sceneCode := strings.TrimSpace(os.Getenv("SCENE_CODE"))
	if sceneCode == "" {
		log.Fatal("启动失败：环境变量 SCENE_CODE 未设置（平台会在场景 Pod spec 中注入）")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("scenario runtime starting: scene_code=%s", sceneCode)
	if err := launcher.MustRunByCode(ctx, sceneCode); err != nil {
		log.Fatalf("scenario runtime stopped: %v", err)
	}
}
