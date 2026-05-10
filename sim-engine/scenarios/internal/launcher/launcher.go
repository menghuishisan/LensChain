// 模块：sim-engine/scenarios/launcher
// 文件职责：通用场景容器启动器 — 根据 scene_code 从 catalog 取 Definition，
//          通过 sdk.NewRuntimeScenario 适配为 Scenario，再调用 sdk.Run 启动 gRPC 服务。
//
// 职责约束（详 sim-engine/AGENTS.md §6.3）：
//   - 不感知具体场景业务，所有场景都通过 catalog 注册机制接入；
//   - 启动参数（场景 code、监听端口）从环境变量读取，不在代码中硬编码业务规则。

package launcher

import (
	"context"
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/catalog"
	sdkscenario "github.com/lenschain/sim-engine/sdk/go/scenario"
)

// RunByCode 根据场景编码启动对应的场景算法容器服务。
//
// 监听地址由 sdk.Run 内部按 SCENARIO_LISTEN_ADDR 环境变量与 :8080 默认值解析（详 sdk.Run）。
func RunByCode(ctx context.Context, sceneCode string) error {
	registry := catalog.NewRegistry()
	definition, err := registry.Get(sceneCode)
	if err != nil {
		return err
	}
	runtime, err := sdkscenario.NewRuntimeScenario(definition)
	if err != nil {
		return err
	}
	return sdkscenario.Run(ctx, runtime)
}

// MustRunByCode 根据场景编码启动服务，失败时返回带场景编码的错误。
func MustRunByCode(ctx context.Context, sceneCode string) error {
	if err := RunByCode(ctx, sceneCode); err != nil {
		return fmt.Errorf("启动场景 %s 失败: %w", sceneCode, err)
	}
	return nil
}
