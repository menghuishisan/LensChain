// 模块：sim-engine/sdk/go/scenario/examples/link
// 文件职责：教师 L3 自定义场景示例 — 演示 owner-based 联动写入 + LinkTrigger 锚点。
//
// 演示能力点（详 AGENTS.md §0.7.6 / §8.3 / §8.5.2）：
//   1. Definition.OwnedFieldPaths：声明本场景拥有的 owner 字段
//   2. ActionDef.WritesOwnedFields：声明本动作会写入哪些 owner 字段（必须 ⊆ OwnedFieldPaths）
//   3. ActionOutput.SharedStateDiff：实际写入 owner 字段
//   4. RenderEnvelope.LinkTriggers：含 SourceAnchorID/TargetAnchorID 用于跨画布弧线
//
// 运行方式：
//
//	SCENARIO_LISTEN_ADDR=:50100 go run ./examples/link

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

const (
	sceneCode    = "demo-link"
	ownerHashKey = "demo.link.last_hash"
)

func definition() scenario.Definition {
	return scenario.Definition{
		Code:            sceneCode,
		Name:            "示例：联动写入与锚点",
		Description:     "演示 owner 字段写入 + LinkTrigger 锚点。",
		Category:        scenario.CategoryGeneric,
		AlgorithmType:   "demo-link",
		Version:         "v1.0.0",
		TimeControlMode: scenario.TimeControlReactive,
		DataSourceMode:  scenario.DataSourceSimulation,
		ExtensionLevel:  scenario.ExtensionL3,

		LinkGroupVersion: "v0.5.0",
		OwnedFieldPaths:  []string{ownerHashKey},

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
						ActionCode:        "publish_hash",
						Label:             "发布哈希",
						Category:          scenario.ActionPrimary,
						Trigger:           scenario.TriggerSubmit,
						Roles:             scenario.RolesAll,
						Reversible:        true,
						WritesOwnedFields: []string{ownerHashKey},
						LinkOwnerFields:   []string{ownerHashKey},
						Fields: []scenario.FieldDef{
							{Name: "hash_hex", Type: scenario.FieldString, Label: "哈希(hex)", Required: true, Default: "abc123"},
						},
					},
					fw.BroadcastHintAction(),
				},
			}
		},
		Init: func(state *scenario.SceneState, _ scenario.InitInput) (scenario.RenderEnvelope, error) {
			return scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt("hint", 0.5, 0.5, "click 'publish_hash' to write owner field", ""),
				},
				IsFullSnapshot: true,
			}, nil
		},
		Step: func(state *scenario.SceneState, _ scenario.StepInput) (scenario.StepOutput, error) {
			return scenario.StepOutput{
				Render: scenario.RenderEnvelope{ChangedKeys: []string{"display_round"}},
			}, nil
		},
		HandleAction: handleAction,
	}
}

func handleAction(state *scenario.SceneState, in scenario.ActionInput) (scenario.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID) // §0.7.3 多 actor 桶
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}
	switch in.ActionCode {
	case "publish_hash":
		hash, _ := in.Params["hash_hex"].(string)
		if hash == "" {
			return scenario.ActionOutput{Success: false, ErrorMessage: "hash_hex 必填"}, nil
		}
		state.Tick++
		anchorID := fmt.Sprintf("hash-anchor-%d", state.Tick)
		return scenario.ActionOutput{
			Success: true,
			Render: scenario.RenderEnvelope{
				Primitives: []scenario.Primitive{
					scenario.PrimLabelAt(anchorID, 0.5, 0.5, "hash="+hash, ""),
				},
				LinkTriggers: []scenario.LinkTrigger{{
					ID:             fmt.Sprintf("lt-%d", state.Tick),
					SourceScene:    sceneCode,
					SourceAction:   "publish_hash",
					LinkGroup:      "crypto-verify-group",
					ChangedFields:  []string{ownerHashKey},
					Payload:        map[string]any{"hash_hex": hash},
					SourceAnchorID: anchorID,
					TargetAnchorID: "verifier-input",
				}},
				ChangedKeys: []string{ownerHashKey},
			},
			SharedStateDiff: map[string]any{
				ownerHashKey: hash,
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
		log.Fatalf("link example 启动失败: %v", err)
	}
}
