// 模块：sim-engine/framework
// 文件职责：协议契约校验工具集。
// 协议依据：AGENTS.md §0.7.1 C22 / C23 / C34；06.md §6.2 / §6.3 / §3.10.1。
//
// 三个公开校验入口：
//
//	ValidateDefinition(def)            场景上架前自检（sdk.NewRuntimeScenario 调用）
//	ValidateActionParams(action, p)    场景作者在 HandleAction 内部按 ActionDef.Fields 校验入参
//	ValidateMicroStepDurations(steps)  确保 MicroStep.DurationMs ≥ 200ms（详 §3.10.1）

package framework

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// =====================================================================
// ValidateDefinition：场景定义自检
// =====================================================================

// semverPattern 严格 semver `vX.Y.Z`（与 §0.7.1 C31 对齐）。
var semverPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// ValidateDefinition 校验 Definition 是否满足平台上架要求（详 AGENTS.md §6.2）。
//
// 任一项不达标返回带语义错误信息；通过则返回 nil。sdk.NewRuntimeScenario 在场景容器启动时
// 调用本函数；若失败则进程立刻退出（避免错误的场景被 K8s 健康检查拉起）。
func ValidateDefinition(def Definition) error {
	if strings.TrimSpace(def.Code) == "" {
		return errors.New("Definition.Code 不能为空")
	}
	if strings.TrimSpace(def.Name) == "" {
		return errors.New("Definition.Name 不能为空")
	}
	if strings.TrimSpace(def.AlgorithmType) == "" {
		return errors.New("Definition.AlgorithmType 不能为空")
	}
	if !semverPattern.MatchString(def.Version) {
		return fmt.Errorf("Definition.Version 必须为严格 semver vX.Y.Z，当前 %q", def.Version)
	}
	if !validTimeControlMode(def.TimeControlMode) {
		return fmt.Errorf("Definition.TimeControlMode 不合法 %q", def.TimeControlMode)
	}
	if !validDataSourceMode(def.DataSourceMode) {
		return fmt.Errorf("Definition.DataSourceMode 不合法 %q", def.DataSourceMode)
	}
	if !validCategory(def.Category) {
		return fmt.Errorf("Definition.Category 不合法 %q", def.Category)
	}
	if !validExtensionLevel(def.ExtensionLevel) {
		return fmt.Errorf("Definition.ExtensionLevel 必须为 L1/L2/L3，当前 %q", def.ExtensionLevel)
	}
	if def.DefaultParams == nil {
		return errors.New("Definition.DefaultParams 必填（即使返回空 map）")
	}
	if def.DefaultState == nil {
		return errors.New("Definition.DefaultState 必填")
	}
	if def.Interaction == nil {
		return errors.New("Definition.Interaction 必填")
	}
	if def.Init == nil {
		return errors.New("Definition.Init 必填")
	}
	if def.Step == nil {
		return errors.New("Definition.Step 必填")
	}
	if def.HandleAction == nil {
		return errors.New("Definition.HandleAction 必填")
	}

	// 交互定义子项校验
	interaction := def.Interaction()
	if err := validateInteraction(def, interaction); err != nil {
		return err
	}
	return nil
}

// validateInteraction 校验 InteractionDefinition 与 Definition 的耦合点。
func validateInteraction(def Definition, in InteractionDefinition) error {
	owned := make(map[string]struct{}, len(def.OwnedFieldPaths))
	for _, p := range def.OwnedFieldPaths {
		owned[p] = struct{}{}
	}

	hasBroadcastHint := false
	hasTeacherAction := false
	for idx, action := range in.Actions {
		if strings.TrimSpace(action.ActionCode) == "" {
			return fmt.Errorf("ActionDef[%d].ActionCode 不能为空", idx)
		}
		if strings.TrimSpace(action.Label) == "" {
			return fmt.Errorf("ActionDef[%d].Label 不能为空（action_code=%s）", idx, action.ActionCode)
		}
		if !validActionCategory(action.Category) {
			return fmt.Errorf("ActionDef[%s].Category 不合法 %q", action.ActionCode, action.Category)
		}
		if !validActionTrigger(action.Trigger) {
			return fmt.Errorf("ActionDef[%s].Trigger 不合法 %q", action.ActionCode, action.Trigger)
		}
		// FieldDef 校验
		for fidx, f := range action.Fields {
			if strings.TrimSpace(f.Name) == "" {
				return fmt.Errorf("ActionDef[%s].Fields[%d].Name 不能为空", action.ActionCode, fidx)
			}
			if !validFieldType(f.Type) {
				return fmt.Errorf("ActionDef[%s].Fields[%s].Type 不合法 %q", action.ActionCode, f.Name, f.Type)
			}
		}
		// WritesOwnedFields 必须 ⊆ Definition.OwnedFieldPaths
		if len(def.OwnedFieldPaths) > 0 {
			for _, w := range action.WritesOwnedFields {
				if _, ok := owned[w]; !ok {
					return fmt.Errorf("ActionDef[%s].WritesOwnedFields 含非 owner 字段 %q（不在 Definition.OwnedFieldPaths 中）", action.ActionCode, w)
				}
			}
		}
		// 教师动作必填 InterveneType
		isTeacher := containsRole(action.Roles, RoleTeacher)
		if isTeacher {
			hasTeacherAction = true
			if action.InterveneType == "" {
				return fmt.Errorf("ActionDef[%s] 角色含 teacher 但未填 InterveneType（详 §10.9）", action.ActionCode)
			}
			if !validInterveneType(action.InterveneType) {
				return fmt.Errorf("ActionDef[%s].InterveneType 不合法 %q", action.ActionCode, action.InterveneType)
			}
		}
		// container-channel ActionDef 必须 Reversible: false
		if action.HybridChannel == HybridChannelContainer && action.Reversible {
			return fmt.Errorf("ActionDef[%s] HybridChannel=container 必须 Reversible=false（详 §0.7.7）", action.ActionCode)
		}
		if action.ActionCode == BroadcastHintActionCode {
			hasBroadcastHint = true
		}
	}

	if !hasBroadcastHint {
		return fmt.Errorf("InteractionDefinition 必须包含 broadcast_hint ActionDef（详 §0.7.5），可调用 fw.BroadcastHintAction() 获取标准实现")
	}
	if !hasTeacherAction {
		return errors.New("InteractionDefinition 必须至少包含 1 个 RoleTeacher-only 的 ActionDef（详 §0.7.5）")
	}
	return nil
}

// =====================================================================
// ValidateActionParams：场景作者在 HandleAction 内部使用
// =====================================================================

// ValidateActionParams 按 ActionDef.Fields 校验 params：
//   - Required 字段缺失 → 错误
//   - 类型不匹配 → 错误（仅类型 family 校验：number/boolean/string/json，不强校验枚举值）
//
// 通过返回 nil。错误信息含字段名，便于教师调试。
func ValidateActionParams(action ActionDef, params map[string]any) error {
	for _, f := range action.Fields {
		val, present := params[f.Name]
		if !present {
			if f.Required && f.Default == nil {
				return fmt.Errorf("字段 %s 必填", f.Name)
			}
			continue
		}
		if val == nil {
			if f.Required {
				return fmt.Errorf("字段 %s 不可为 null", f.Name)
			}
			continue
		}
		if err := checkFieldType(f, val); err != nil {
			return fmt.Errorf("字段 %s: %w", f.Name, err)
		}
	}
	return nil
}

// checkFieldType 仅做类型 family 校验。
func checkFieldType(f FieldDef, val any) error {
	switch f.Type {
	case FieldString, FieldSelect, FieldEnum:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("应为 string，实际 %T", val)
		}
	case FieldNumber, FieldRange:
		switch val.(type) {
		case float64, float32, int, int32, int64:
			return nil
		default:
			return fmt.Errorf("应为 number，实际 %T", val)
		}
	case FieldBoolean:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("应为 boolean，实际 %T", val)
		}
	case FieldMultiSelect:
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("应为 array，实际 %T", val)
		}
	case FieldJSON:
		// 任意类型放行
		return nil
	}
	return nil
}

// =====================================================================
// ValidateMicroStepDurations：动画时长检查（§3.10.1）
// =====================================================================

// MinMicroStepDurationMs 是单个 MicroStep 的最小时长（含动画 buffer）。
const MinMicroStepDurationMs = 200

// ValidateMicroStepDurations 校验 MicroStep 时长是否满足前端动画调度的最小预算。
//
// 教学动画与同步要求每步 ≥ 200ms，确保学生能看清状态变化；过短会导致快闪。
// 场景作者可在 Init/Step/HandleAction 输出 RenderEnvelope 之后调用本函数自检。
func ValidateMicroStepDurations(steps []MicroStep) error {
	for i, s := range steps {
		if s.DurationMs < MinMicroStepDurationMs {
			return fmt.Errorf("MicroStep[%d] (%s) duration_ms=%d 小于最小 %dms（详 §3.10.1）",
				i, s.ID, s.DurationMs, MinMicroStepDurationMs)
		}
	}
	return nil
}

// =====================================================================
// 内部工具
// =====================================================================

func containsRole(roles []UserRole, target UserRole) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func validCategory(c SceneCategory) bool {
	switch c {
	case CategoryNodeNetwork, CategoryConsensus, CategoryCryptography, CategoryDataStructure,
		CategoryTransaction, CategorySmartContract, CategoryAttackSecurity, CategoryEconomic, CategoryGeneric:
		return true
	}
	return false
}

func validTimeControlMode(m TimeControlMode) bool {
	switch m {
	case TimeControlProcess, TimeControlReactive, TimeControlContinuous:
		return true
	}
	return false
}

func validDataSourceMode(m DataSourceMode) bool {
	switch m {
	case DataSourceSimulation, DataSourceCollection, DataSourceDual:
		return true
	}
	return false
}

func validActionCategory(c ActionCategory) bool {
	switch c {
	case ActionParamTune, ActionAttackInject, ActionPrimary, ActionObserve:
		return true
	}
	return false
}

func validActionTrigger(t ActionTrigger) bool {
	switch t {
	case TriggerSubmit, TriggerImmediate, TriggerHold:
		return true
	}
	return false
}

func validFieldType(t FieldType) bool {
	switch t {
	case FieldString, FieldNumber, FieldBoolean, FieldSelect,
		FieldEnum, FieldRange, FieldJSON, FieldMultiSelect:
		return true
	}
	return false
}

func validExtensionLevel(l ExtensionLevel) bool {
	switch l {
	case ExtensionL1, ExtensionL2, ExtensionL3:
		return true
	}
	return false
}

func validInterveneType(t InterveneType) bool {
	switch t {
	case InterveneHint, InterveneFault, InterveneAttack, IntervenePhase, InterveneTopology,
		InterveneState, InterveneReset, InterveneEpoch, InterveneRevert, InterveneFreeze:
		return true
	}
	return false
}
