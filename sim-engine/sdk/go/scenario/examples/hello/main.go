// 模块：sim-engine/sdk/go/scenario/examples/hello
// 文件职责：教师自定义场景的最简可运行示例 — 一个静态 hello-world 场景。
//
// 运行方式：
//
//	go run ./examples/hello
//
// 运行后该进程会监听 :8080 并实现 SimScenarioService gRPC，可被 SimEngine Core 拉起当作
// 场景容器的算法实现使用。

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/lenschain/sim-engine/sdk/go/scenario"
)

// helloScene 是一个最简 reactive 模式静态场景：始终展示一个 hello label。
type helloScene struct{}

// Meta 返回场景元信息。
func (helloScene) Meta(ctx context.Context) (scenario.Meta, error) {
	defaultParams, _ := json.Marshal(map[string]any{})
	defaultState, _ := json.Marshal(map[string]any{"counter": 0})
	return scenario.Meta{
		Code:            "demo-hello",
		Name:            "示例：Hello SimEngine",
		Description:     "教师自定义场景最简示例。",
		Category:        scenario.CategoryGeneric,
		AlgorithmType:   "demo",
		Version:         "1.0.0",
		TimeControlMode: scenario.TimeControlReactive,
		DataSourceMode:  scenario.DataSourceSimulation,
		DefaultParams:   defaultParams,
		DefaultState:    defaultState,
	}, nil
}

// InteractionSchema 返回学生可执行的 ActionDef 列表（这里仅一个）。
func (helloScene) InteractionSchema(ctx context.Context) (scenario.InteractionDefinition, error) {
	return scenario.InteractionDefinition{
		SceneCode:     "demo-hello",
		SchemaVersion: "1.0.0",
		Actions: []scenario.ActionDef{{
			ActionCode:  "say_hi",
			Label:       "打招呼",
			Description: "在 label 文本后面追加一个感叹号。",
			Category:    scenario.ActionPrimary,
			Trigger:     scenario.TriggerImmediate,
			Roles:       []scenario.UserRole{scenario.RoleStudent},
		}},
	}, nil
}

// Init 返回首帧 RenderEnvelope（仅一个 label 原语）。
func (helloScene) Init(ctx context.Context, req scenario.InitRequest) (scenario.InitResult, error) {
	state, _ := json.Marshal(map[string]any{"text": "Hello SimEngine"})
	envelope := scenario.RenderEnvelope{
		Primitives: []scenario.Primitive{
			scenario.PrimLabelAt("greeting", 0.5, 0.5, "Hello SimEngine", ""),
		},
		IsFullSnapshot: true,
	}
	envelopeJSON, _ := json.Marshal(envelope)
	return scenario.InitResult{
		Tick:               0,
		SceneStateJSON:     state,
		RenderEnvelopeJSON: envelopeJSON,
	}, nil
}

// Step reactive 模式下每个 tick 不变化（直接返回上一帧的 state 与一个最小 envelope）。
func (helloScene) Step(ctx context.Context, req scenario.StepRequest) (scenario.StepResult, error) {
	envelope := scenario.RenderEnvelope{
		Primitives: []scenario.Primitive{
			scenario.PrimLabelAt("greeting", 0.5, 0.5, "Hello SimEngine", ""),
		},
	}
	envelopeJSON, _ := json.Marshal(envelope)
	return scenario.StepResult{
		Tick:               req.Tick,
		SceneStateJSON:     req.SceneStateJSON,
		RenderEnvelopeJSON: envelopeJSON,
	}, nil
}

// HandleAction 处理 say_hi：在文案后追加感叹号。
func (helloScene) HandleAction(ctx context.Context, req scenario.ActionRequest) (scenario.ActionResult, error) {
	var state struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(req.SceneStateJSON, &state)
	if state.Text == "" {
		state.Text = "Hello SimEngine"
	}
	if req.ActionCode == "say_hi" {
		state.Text += "!"
	}
	stateJSON, _ := json.Marshal(state)

	envelope := scenario.RenderEnvelope{
		Primitives: []scenario.Primitive{
			scenario.PrimLabelAt("greeting", 0.5, 0.5, state.Text, ""),
		},
	}
	envelopeJSON, _ := json.Marshal(envelope)
	return scenario.ActionResult{
		Success:            true,
		Tick:               req.Tick,
		SceneStateJSON:     stateJSON,
		RenderEnvelopeJSON: envelopeJSON,
	}, nil
}

// main 启动最简场景容器；监听 :8080（可通过环境变量 SCENARIO_LISTEN_ADDR 覆盖）。
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := scenario.Run(ctx, helloScene{}); err != nil {
		log.Fatalf("hello scenario 启动失败: %v", err)
	}
}
