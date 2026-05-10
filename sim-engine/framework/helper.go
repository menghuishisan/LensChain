// 模块：sim-engine/framework
// 文件职责：场景作者通用助手 — broadcast_hint 标准实现 + 多 actor 桶工具。
// 协议依据：AGENTS.md §0.7.3 多 actor 桶；§0.7.5 / §0.7.8 教师 broadcast_hint 规范。
//
// 设计要点：
// 1. 本文件不引入任何业务逻辑，只封装"所有场景共用的协议级动作"。
// 2. 场景使用方式：在 Interaction.Actions 列表中追加 BroadcastHintAction()，
//    在 HandleAction case 列表中调用 HandleBroadcastHint() 处理。
// 3. ActorBucket / EnsureActorBucket 用于多 actor 场景把不同学生的私有状态隔离在 SceneState.Data["actor_states"] 桶。

package framework

import (
	"fmt"
)

// =====================================================================
// broadcast_hint 教师广播提示 — 所有场景必备的 RoleTeacher-only 动作
// =====================================================================

// BroadcastHintActionCode 是教师广播提示动作的全局动作编码。
const BroadcastHintActionCode = "broadcast_hint"

// BroadcastHintAction 返回标准 broadcast_hint ActionDef，所有场景必须把它追加到 Interaction.Actions。
//
// 该动作语义（详 AGENTS.md §0.7.8）：
//   - Roles: RolesTeacherOnly（仅教师可调用）
//   - Reversible: true（仅写本地 SceneState，无外部副作用，可参与 process 模式回退）
//   - InterveneType: InterveneHint（写 teacher_intervene_logs.intervene_type=hint）
//   - 单字段 text：教师输入提示文字
func BroadcastHintAction() ActionDef {
	return ActionDef{
		ActionCode:    BroadcastHintActionCode,
		Label:         "广播教学提示",
		Description:   "教师向当前场景所有学生面板推送一条教学提示；前端按 owner_role=teacher 用红色边框 + 全屏可见样式渲染。",
		Category:      ActionPrimary,
		Trigger:       TriggerSubmit,
		Roles:         RolesTeacherOnly,
		Reversible:    true,
		InterveneType: InterveneHint,
		Fields: []FieldDef{
			{
				Name:     "text",
				Type:     FieldString,
				Label:    "提示文字",
				Required: true,
				Default:  "请关注当前步骤",
			},
		},
	}
}

// HandleBroadcastHint 是 broadcast_hint ActionCode 的标准处理。
//
// 用法：
//
//	func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
//	    if out, ok := fw.HandleBroadcastHint(state, in); ok {
//	        return out, nil
//	    }
//	    // ... 其它 case
//	}
//
// 返回值 ok == false 表示本次调用不是 broadcast_hint，调用方应继续处理其它 ActionCode。
// 角色校验失败、参数缺失等错误都体现在返回的 ActionOutput.Success / ErrorMessage 中。
//
// 输出 RenderEnvelope 包含一个 annotation 原语，owner_role 标为 "teacher"。
// 前端皮肤层据此区分教师标注（红框，全屏）与学生标注（小标注）。
func HandleBroadcastHint(state *SceneState, in ActionInput) (ActionOutput, bool) {
	if in.ActionCode != BroadcastHintActionCode {
		return ActionOutput{}, false
	}
	if in.UserRole != RoleTeacher {
		return ActionOutput{
			Success:      false,
			ErrorMessage: "broadcast_hint 仅教师可调用",
		}, true
	}
	text, _ := in.Params["text"].(string)
	if text == "" {
		text = "（教师未填写提示文字）"
	}
	tick := int64(0)
	if state != nil {
		tick = state.Tick
	}
	annot := PrimAnnotation(
		fmt.Sprintf("teacher-hint-%d", tick),
		"text",
		"teacher",
		8000,
		map[string]any{"x": 0.5, "y": 0.08},
		map[string]any{"emphasis": "high"},
		text,
	)
	return ActionOutput{
		Success: true,
		Render: RenderEnvelope{
			Primitives:     []Primitive{annot},
			IsFullSnapshot: false,
			ChangedKeys:    []string{"teacher_hint"},
		},
	}, true
}

// =====================================================================
// 多 actor 桶（详 AGENTS.md §0.7.3）
// =====================================================================

// ActorStatesKey 是 SceneState.Data 中保存所有 actor 私有状态的固定 key。
const ActorStatesKey = "actor_states"

// DefaultActorID 是单人场景或 ActorID 缺失时使用的回退桶 key。
const DefaultActorID = "default"

// EnsureActorBucket 取出（必要时初始化）指定 actorID 对应的私有状态桶。
//
// 路由规则（详 AGENTS.md §0.7.3）：
//  1. actorID == ""：回退到 "default"（单人场景向后兼容）
//  2. SceneState.Data 不存在 actor_states 时自动创建
//
// 返回的 map 是 SceneState 内部桶的实时引用 — 直接修改即可，无需回写。
//
// 多 actor 场景应在 HandleAction 进入时第一行调用本函数；单人场景调用本函数也是无害的
// （只会在 default 桶下分配一次空 map），并能让 §0.7.1 C5（in.ActorID grep）天然命中。
func EnsureActorBucket(state *SceneState, actorID string) map[string]any {
	if state == nil {
		return map[string]any{}
	}
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	if actorID == "" {
		actorID = DefaultActorID
	}
	rawBuckets, ok := state.Data[ActorStatesKey].(map[string]any)
	if !ok {
		rawBuckets = map[string]any{}
		state.Data[ActorStatesKey] = rawBuckets
	}
	bucket, ok := rawBuckets[actorID].(map[string]any)
	if !ok {
		bucket = map[string]any{}
		rawBuckets[actorID] = bucket
	}
	return bucket
}

