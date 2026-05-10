// Package scenario 是 LensChain SimEngine 的场景算法 SDK。
//
// 本包面向两类使用者：
//
//  1. 教师自定义场景作者：通过实现 Scenario 接口，把任意区块链算法封装成
//     gRPC 微服务镜像；按平台规范打包后即可上传到平台场景库给学生使用。
//  2. 平台内部场景：通过 framework.Definition 声明算法元信息与三个钩子
//     （Init / Step / HandleAction），由 NewRuntimeScenario 包装成 Scenario
//     后挂上 sdk.Server 启动。
//
// # 协议契约
//
// sdk 通过 type alias 真正重导出 sim-engine/framework 模块的协议契约（Primitive /
// MicroStep / RenderEnvelope / SceneState / ActionDef / FieldDef /
// InteractionDefinition / Definition / 47 原语类型常量 / Layer / 时间控制模式 /
// 类目 / 角色 / 触发方式 / 字段类型 / 混合通道）。
//
// 这意味着教师自定义场景与平台内部场景使用同一份 Go 类型，避免双套同步带来的协议腐化。
// 协议字段含义详见 docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.2 / §6.2 / §6.3。
//
// # 最简场景示例
//
//	package main
//
//	import (
//		"context"
//		"os"
//		"os/signal"
//
//		"github.com/lenschain/sim-engine/sdk/go/scenario"
//	)
//
//	type myScenario struct{}
//
//	func (myScenario) Meta(ctx context.Context) (scenario.Meta, error) {
//		return scenario.Meta{
//			Code:            "demo-scene",
//			Name:            "示例场景",
//			AlgorithmType:   "demo",
//			Version:         "1.0.0",
//			Category:        scenario.CategoryGeneric,
//			TimeControlMode: scenario.TimeControlReactive,
//			DataSourceMode:  scenario.DataSourceSimulation,
//		}, nil
//	}
//
//	func (myScenario) InteractionSchema(ctx context.Context) (scenario.InteractionDefinition, error) {
//		return scenario.InteractionDefinition{
//			SceneCode:     "demo-scene",
//			SchemaVersion: "1.0.0",
//			Actions:       []scenario.ActionDef{},
//		}, nil
//	}
//
//	func (myScenario) Init(ctx context.Context, req scenario.InitRequest) (scenario.InitResult, error) {
//		envelope := scenario.RenderEnvelope{
//			Primitives: []scenario.Primitive{
//				scenario.PrimLabel("hello", "", 100, 100, "Hello SimEngine", ""),
//			},
//			IsFullSnapshot: true,
//		}
//		// ... 序列化 envelope 与 scene_state 到 InitResult，省略
//		return scenario.InitResult{}, nil
//	}
//
//	func (myScenario) Step(ctx context.Context, req scenario.StepRequest) (scenario.StepResult, error) {
//		return scenario.StepResult{}, nil
//	}
//
//	func (myScenario) HandleAction(ctx context.Context, req scenario.ActionRequest) (scenario.ActionResult, error) {
//		return scenario.ActionResult{Success: true}, nil
//	}
//
//	func main() {
//		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//		defer cancel()
//		_ = scenario.Run(ctx, myScenario{})
//	}
//
// # 平台内部场景接入示例
//
// 平台内部 43 场景通过 framework.Definition 声明算法，并由 NewRuntimeScenario 适配：
//
//	import (
//		fw "github.com/lenschain/sim-engine/framework"
//		"github.com/lenschain/sim-engine/sdk/go/scenario"
//	)
//
//	func runPBFT(ctx context.Context) error {
//		def := fw.Definition{
//			Code: "pbft-consensus",
//			// ... Init / Step / HandleAction / Interaction / DefaultState ...
//		}
//		runtime, err := scenario.NewRuntimeScenario(def)
//		if err != nil {
//			return err
//		}
//		return scenario.Run(ctx, runtime)
//	}
//
// # 模块边界（详 sim-engine/AGENTS.md §八）
//
// sdk 不依赖 scenarios/internal/* 的具体算法；所有共享逻辑位于 framework module。
// sdk 不暴露 core 内部类型；所有外部 API 必须保持稳定，版本化升级。
//
// # 协议三方对齐
//
// sdk 类型与 proto/lenschain/sim_scenario/v1/sim_scenario.proto、
// renderers/shared/types.ts 三方 1:1 对齐；任何变更必须三端同步。
package scenario
