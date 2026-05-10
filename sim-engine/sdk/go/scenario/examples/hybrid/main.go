// 模块：sim-engine/sdk/go/scenario/examples/hybrid
// 文件职责：教师 L3 自定义场景示例 — 演示混合实验 HybridChannelContainer + ContainerData。
//
// 演示能力点（详 AGENTS.md §0.7.4 / §0.7.7）：
//   1. ActionDef.HybridChannel = HybridChannelContainer：标识本动作走真链容器侧（geth RPC 等）
//   2. ActionDef.Reversible = false：容器侧动作有外部副作用，不可参与回退栈
//   3. RenderEnvelope.ContainerData：场景把上一周期的容器指标透传给前端用于驱动原语
//   4. StepInput.IncomingContainerMetrics：Core Collector 把容器指标注入到 Step
//
// 运行方式：
//
//	SCENARIO_LISTEN_ADDR=:50100 go run ./examples/hybrid

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/sdk/go/scenario"
)

const sceneCode = "demo-hybrid"

func definition() scenario.Definition {
	return scenario.Definition{
		Code:            sceneCode,
		Name:            "示例：混合实验容器通道",
		Description:     "演示 HybridChannelContainer + ContainerData 混合实验集成。",
		Category:        scenario.CategoryGeneric,
		AlgorithmType:   "demo-hybrid",
		Version:         "v1.0.0",
		TimeControlMode: scenario.TimeControlContinuous,
		DataSourceMode:  scenario.DataSourceDual,
		ExtensionLevel:  scenario.ExtensionL3,

		LinkGroupVersion: "v0.5.0",
		OwnedFieldPaths:  []string{"demo.hybrid.last_block"},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState: func() scenario.SceneState {
			return scenario.SceneState{SceneCode: sceneCode, Tick: 0, Data: map[string]any{}}
		},
		Interaction: func() scenario.InteractionDefinition {
			return scenario.InteractionDefinition{
				SceneCode:     sceneCode,
				SchemaVersion: "v1.0.0",
				Actions: []scenario.ActionDef{
					{
						ActionCode:    "fetch_block",
						Label:         "拉取真实区块",
						Description:   "通过 geth RPC 获取最新区块（容器通道）。",
						Category:      scenario.ActionPrimary,
						Trigger:       scenario.TriggerImmediate,
						Roles:         scenario.RolesAll,
						HybridChannel: scenario.HybridChannelContainer,
						ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}' http://geth:8545`,
						Reversible:    false, // §0.7.7 容器侧不可回退
					},
					fw.BroadcastHintAction(),
				},
			}
		},
		Init: func(state *scenario.SceneState, _ scenario.InitInput) (scenario.RenderEnvelope, error) {
			return scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt("status", 0.5, 0.4, "等待真链区块...", ""),
					scenario.PrimLabelAt("block-num", 0.5, 0.55, "block=?", ""),
				},
				IsFullSnapshot: true,
			}, nil
		},
		Step: func(state *scenario.SceneState, in scenario.StepInput) (scenario.StepOutput, error) {
			// 把 Core Collector 注入的容器指标透传到 RenderEnvelope.ContainerData。
			env := scenario.RenderEnvelope{
				ContainerData: in.IncomingContainerMetrics,
				ChangedKeys:   []string{"display_round"},
			}
			return scenario.StepOutput{Render: env}, nil
		},
		HandleAction: handleAction,
	}
}

func handleAction(state *scenario.SceneState, in scenario.ActionInput) (scenario.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}
	switch in.ActionCode {
	case "fetch_block":
		// 场景层不实际跑 RPC — Core 按 ContainerCmd 走 K8sOrchestrator 转发到 geth 容器；
		// 本场景仅返回"已下发"指示，等下个 Step 收到 ContainerData 时再可视化。
		state.Tick++
		return scenario.ActionOutput{
			Success: true,
			Render: scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt("status", 0.5, 0.4, fmt.Sprintf("已派发到容器 (tick=%d)，等待回填...", state.Tick), ""),
				},
				ChangedKeys: []string{"status"},
			},
		}, nil
	}
	return scenario.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode"}, errors.New("unknown action")
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	rt, err := scenario.NewRuntimeScenario(definition())
	if err != nil {
		log.Fatalf("Definition 校验失败: %v", err)
	}
	if err := scenario.Run(ctx, rt); err != nil {
		log.Fatalf("hybrid example 启动失败: %v", err)
	}
}
