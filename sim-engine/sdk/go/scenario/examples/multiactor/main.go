// 模块：sim-engine/sdk/go/scenario/examples/multiactor
// 文件职责：教师 L3 自定义场景示例 — 演示多 actor 共享会话与 actor 桶。
//
// 演示能力点（详 AGENTS.md §0.7.3）：
//   1. Definition.SupportsMultiActor = true
//   2. ActorID 缺失视为错误
//   3. EnsureActorBucket 在 SceneState.Data["actor_states"] 维护每个 actor 的私有状态
//   4. 教师专属 ActionDef（reset_actor）写入全局桶，不分 actor
//
// 运行方式：
//
//	SCENARIO_LISTEN_ADDR=:50100 go run ./examples/multiactor

package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/sdk/go/scenario"
)

const sceneCode = "demo-multiactor"

func definition() scenario.Definition {
	return scenario.Definition{
		Code:            sceneCode,
		Name:            "示例：多 actor 共享会话",
		Description:     "演示 SupportsMultiActor + EnsureActorBucket 维护每个学生的私有状态。",
		Category:        scenario.CategoryGeneric,
		AlgorithmType:   "demo-multiactor",
		Version:         "v1.0.0",
		TimeControlMode: scenario.TimeControlReactive,
		DataSourceMode:  scenario.DataSourceSimulation,
		ExtensionLevel:  scenario.ExtensionL3,

		LinkGroupVersion:   "v0.5.0",
		OwnedFieldPaths:    []string{"demo.multiactor.global_count"},
		SupportsMultiActor: true,

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
						ActionCode: "increment_self",
						Label:      "本人计数 +1",
						Category:   scenario.ActionPrimary,
						Trigger:    scenario.TriggerImmediate,
						Roles:      scenario.RolesStudentOnly,
						Reversible: true,
					},
					{
						ActionCode:    "reset_all",
						Label:         "重置所有学生计数",
						Category:      scenario.ActionPrimary,
						Trigger:       scenario.TriggerImmediate,
						Roles:         scenario.RolesTeacherOnly,
						Reversible:    true,
						InterveneType: scenario.InterveneReset,
					},
					fw.BroadcastHintAction(),
				},
			}
		},
		Init: func(state *scenario.SceneState, _ scenario.InitInput) (scenario.RenderEnvelope, error) {
			return scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt("hint", 0.5, 0.5, "click '本人计数 +1'", ""),
				},
				IsFullSnapshot: true,
			}, nil
		},
		Step: func(state *scenario.SceneState, _ scenario.StepInput) (scenario.StepOutput, error) {
			return scenario.StepOutput{Render: scenario.RenderEnvelope{ChangedKeys: []string{"display_round"}}}, nil
		},
		HandleAction: handleAction,
	}
}

func handleAction(state *scenario.SceneState, in scenario.ActionInput) (scenario.ActionOutput, error) {
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}
	switch in.ActionCode {
	case "increment_self":
		// SupportsMultiActor=true 时 ActorID 不能为空
		if in.ActorID == "" {
			return scenario.ActionOutput{Success: false, ErrorMessage: "multi-actor scene requires ActorID"}, nil
		}
		bucket := fw.EnsureActorBucket(state, in.ActorID)
		count, _ := bucket["count"].(float64)
		count++
		bucket["count"] = count
		state.Tick++
		return scenario.ActionOutput{
			Success: true,
			Render: scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt("hint", 0.5, 0.5, "actor "+in.ActorID, ""),
				},
				ChangedKeys: []string{"actor_states." + in.ActorID + ".count"},
			},
		}, nil
	case "reset_all":
		// 教师动作写入全局桶（不分 actor）。
		if state.Data == nil {
			state.Data = map[string]any{}
		}
		state.Data[fw.ActorStatesKey] = map[string]any{}
		state.Tick++
		return scenario.ActionOutput{
			Success: true,
			Render: scenario.RenderEnvelope{
				ChangedKeys: []string{fw.ActorStatesKey},
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
		log.Fatalf("multiactor example 启动失败: %v", err)
	}
}
