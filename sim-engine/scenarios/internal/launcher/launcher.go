package launcher

import (
	"context"
	"fmt"
	"os"

	"github.com/lenschain/sim-engine/scenarios/internal/catalog"
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
	sdkscenario "github.com/lenschain/sim-engine/sdk/go/scenario"
)

// RunByCode 根据场景编码启动对应的场景算法容器服务。
func RunByCode(ctx context.Context, sceneCode string) error {
	registry := catalog.NewRegistry()
	definition, err := registry.Get(sceneCode)
	if err != nil {
		return err
	}
	runtime, err := framework.NewRuntimeScenario(definition)
	if err != nil {
		return err
	}
	listenAddr := os.Getenv("SCENARIO_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}
	return sdkscenario.Serve(ctx, runtime, sdkscenario.ServeConfig{ListenAddr: listenAddr})
}

// MustRunByCode 根据场景编码启动服务，失败时返回带场景编码的错误。
func MustRunByCode(ctx context.Context, sceneCode string) error {
	if err := RunByCode(ctx, sceneCode); err != nil {
		return fmt.Errorf("启动场景 %s 失败: %w", sceneCode, err)
	}
	return nil
}
