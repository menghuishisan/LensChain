// instance_service_checkpoint.go
// 模块04 — 实验环境：检查点执行与评分共享逻辑
// 统一处理个人/组级检查点的执行、落库与结果镜像，避免多处各自维护检查点规则

package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/assertion"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// groupCheckpointMemberResult 表示组级检查点在单个组员实例上的验证结果。
type groupCheckpointMemberResult struct {
	StudentID    int64   `json:"student_id,string"`
	InstanceID   int64   `json:"instance_id,string"`
	Container    string  `json:"container,omitempty"`
	Passed       bool    `json:"passed"`
	ExitCode     *int    `json:"exit_code,omitempty"`
	CheckOutput  string  `json:"check_output,omitempty"`
	SessionID    string  `json:"session_id,omitempty"`
	Score        float64 `json:"score,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// checkpointExecutionState 表示一次检查点验证在当前作用域下的统一执行结果。
type checkpointExecutionState struct {
	Targets         []*entity.ExperimentInstance
	IsPassed        bool
	Score           *float64
	CheckOutput     *string
	AssertionResult json.RawMessage
}

const defaultCheckpointExecTimeout = 10 * time.Second

// executeCheckpoint 执行单个检查点验证。
// 个人检查点只作用于当前实例，组级检查点会在组内所有已就绪实例上执行并将结果镜像给所有组员。
func (s *instanceService) executeCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) *entity.CheckpointResult {
	state := s.runCheckpoint(ctx, instance, cp)
	now := time.Now()
	results := s.persistCheckpointState(ctx, cp, state, now)
	return pickCheckpointResult(results, instance.ID)
}

// persistManualCheckpointScore 按检查点作用域写入教师手动评分结果。
// 组级手动评分会同步写入组内所有实例，保证所有组员看到统一结果。
func (s *instanceService) persistManualCheckpointScore(ctx context.Context, instance *entity.ExperimentInstance, checkpoint *entity.TemplateCheckpoint, teacherID int64, score float64, comment *string, gradedAt time.Time) error {
	targets := s.resolveCheckpointTargetInstances(ctx, instance, checkpoint)
	if len(targets) == 0 {
		targets = []*entity.ExperimentInstance{instance}
	}
	passed := score >= checkpoint.Score
	for _, target := range targets {
		result, err := s.checkResultRepo.GetByInstanceAndCheckpoint(ctx, target.ID, checkpoint.ID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			scoreCopy := score
			create := &entity.CheckpointResult{
				ID:             snowflake.Generate(),
				InstanceID:     target.ID,
				CheckpointID:   checkpoint.ID,
				StudentID:      target.StudentID,
				IsPassed:       checkpointBoolPtr(passed),
				Score:          &scoreCopy,
				TeacherComment: comment,
				GradedBy:       &teacherID,
				GradedAt:       &gradedAt,
				CheckedAt:      gradedAt,
				CreatedAt:      gradedAt,
				UpdatedAt:      gradedAt,
			}
			if createErr := s.checkResultRepo.Create(ctx, create); createErr != nil {
				return createErr
			}
			continue
		}
		fields := map[string]interface{}{
			"is_passed":       passed,
			"score":           score,
			"teacher_comment": comment,
			"graded_by":       teacherID,
			"graded_at":       gradedAt,
			"checked_at":      gradedAt,
			"updated_at":      gradedAt,
		}
		if updateErr := s.checkResultRepo.UpdateFields(ctx, result.ID, fields); updateErr != nil {
			return updateErr
		}
	}
	return nil
}

// runCheckpoint 根据检查点范围执行验证并返回统一结果。
func (s *instanceService) runCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) *checkpointExecutionState {
	targets := s.resolveCheckpointTargetInstances(ctx, instance, cp)
	if len(targets) == 0 {
		targets = []*entity.ExperimentInstance{instance}
	}

	if cp.Scope == enum.CheckpointScopeGroup {
		return s.runGroupCheckpoint(ctx, instance, cp, targets)
	}
	return s.runPersonalCheckpoint(ctx, instance, cp)
}

// runPersonalCheckpoint 执行个人检查点。
func (s *instanceService) runPersonalCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) *checkpointExecutionState {
	state := &checkpointExecutionState{
		Targets:  []*entity.ExperimentInstance{instance},
		IsPassed: false,
	}

	switch cp.CheckType {
	case enum.CheckTypeScript:
		passed, output, detail := s.validateScriptCheckpoint(ctx, instance, cp)
		state.IsPassed = passed
		state.CheckOutput = &output
		state.AssertionResult = detail
	case enum.CheckTypeSimAssert:
		passed, output, detail := s.validateSimCheckpoint(ctx, instance, cp)
		state.IsPassed = passed
		state.CheckOutput = &output
		state.AssertionResult = detail
	}

	if state.IsPassed {
		score := cp.Score
		state.Score = &score
	}
	return state
}

// runGroupCheckpoint 执行组级检查点。
// 仅当组内所有成员实例都处于运行中时才会执行真实验证，否则返回不可验证结果并同步到已启动组员。
func (s *instanceService) runGroupCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint, targets []*entity.ExperimentInstance) *checkpointExecutionState {
	state := &checkpointExecutionState{
		Targets:  targets,
		IsPassed: false,
	}

	if instance.GroupID == nil {
		output := "组级检查点要求实例属于实验分组"
		state.CheckOutput = &output
		state.AssertionResult = mustMarshalJSON(map[string]interface{}{
			"scope":   "group",
			"message": output,
		})
		return state
	}

	members, err := s.groupMemberRepo.ListByGroupID(ctx, *instance.GroupID)
	if err != nil {
		output := fmt.Sprintf("获取分组成员失败: %v", err)
		state.CheckOutput = &output
		state.AssertionResult = mustMarshalJSON(map[string]interface{}{
			"scope":   "group",
			"message": output,
		})
		return state
	}

	instancesByStudent := make(map[int64]*entity.ExperimentInstance, len(targets))
	for _, target := range targets {
		instancesByStudent[target.StudentID] = target
	}

	memberResults := make([]groupCheckpointMemberResult, 0, len(members))
	notReady := make([]string, 0)
	for _, member := range members {
		memberResult := groupCheckpointMemberResult{StudentID: member.StudentID}
		target := instancesByStudent[member.StudentID]
		if target == nil {
			memberResult.ErrorMessage = "未启动实验实例"
			notReady = append(notReady, strconv.FormatInt(member.StudentID, 10))
		} else {
			memberResult.InstanceID = target.ID
			if target.Status != enum.InstanceStatusRunning {
				memberResult.ErrorMessage = fmt.Sprintf("实例状态为%s", enum.GetInstanceStatusText(target.Status))
				notReady = append(notReady, strconv.FormatInt(member.StudentID, 10))
			}
		}
		memberResults = append(memberResults, memberResult)
	}
	if len(notReady) > 0 {
		output := "组级检查点需所有组员实例处于运行中后方可验证"
		state.CheckOutput = &output
		state.AssertionResult = mustMarshalJSON(map[string]interface{}{
			"scope":    "group",
			"message":  output,
			"students": notReady,
			"members":  memberResults,
		})
		return state
	}

	switch cp.CheckType {
	case enum.CheckTypeScript:
		passed, output, detail := s.validateGroupScriptCheckpoint(ctx, cp, targets)
		state.IsPassed = passed
		state.CheckOutput = &output
		state.AssertionResult = detail
	case enum.CheckTypeSimAssert:
		passed, output, detail := s.validateGroupSimCheckpoint(ctx, cp, targets)
		state.IsPassed = passed
		state.CheckOutput = &output
		state.AssertionResult = detail
	}

	if state.IsPassed {
		score := cp.Score
		state.Score = &score
	}
	return state
}

// resolveCheckpointTargetInstances 解析检查点需要同步结果的实例集合。
func (s *instanceService) resolveCheckpointTargetInstances(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) []*entity.ExperimentInstance {
	if cp.Scope != enum.CheckpointScopeGroup || instance.GroupID == nil {
		return []*entity.ExperimentInstance{instance}
	}
	members, err := s.groupMemberRepo.ListByGroupID(ctx, *instance.GroupID)
	if err != nil || len(members) == 0 {
		return []*entity.ExperimentInstance{instance}
	}
	instances, err := s.instanceRepo.ListByGroupID(ctx, *instance.GroupID)
	if err != nil || len(instances) == 0 {
		return []*entity.ExperimentInstance{instance}
	}
	latestByStudent := buildLatestInstanceByStudent(instances)
	targets := make([]*entity.ExperimentInstance, 0, len(members))
	for _, member := range members {
		if item := latestByStudent[member.StudentID]; item != nil {
			targets = append(targets, item)
		}
	}
	if len(targets) == 0 {
		return []*entity.ExperimentInstance{instance}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].ID < targets[j].ID
	})
	return targets
}

// validateScriptCheckpoint 在单个实例目标容器内执行脚本检查点。
func (s *instanceService) validateScriptCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) (bool, string, json.RawMessage) {
	result := groupCheckpointMemberResult{
		StudentID:  instance.StudentID,
		InstanceID: instance.ID,
	}
	if cp.TargetContainer != nil {
		result.Container = *cp.TargetContainer
	}

	switch {
	case cp.ScriptContent == nil || cp.TargetContainer == nil:
		result.ErrorMessage = "检查点脚本或目标容器未配置"
	case instance.Namespace == nil || *instance.Namespace == "":
		result.ErrorMessage = "实验实例未分配命名空间"
	default:
		podName := fmt.Sprintf("%s-%s", *instance.Namespace, *cp.TargetContainer)
		scriptCtx, cancel := context.WithTimeout(ctx, defaultCheckpointExecTimeout)
		defer cancel()
		execResult, err := s.k8sSvc.ExecInPod(scriptCtx, *instance.Namespace, podName, *cp.TargetContainer, *cp.ScriptContent)
		if err != nil {
			if errors.Is(scriptCtx.Err(), context.DeadlineExceeded) {
				result.ErrorMessage = "验证超时"
			} else {
				result.ErrorMessage = err.Error()
			}
		} else {
			result.Passed = execResult.ExitCode == 0
			result.CheckOutput = execResult.Stdout
			result.Score = cp.Score
			exitCode := execResult.ExitCode
			result.ExitCode = &exitCode
			if execResult.Stderr != "" {
				result.ErrorMessage = execResult.Stderr
			}
		}
	}

	output := result.CheckOutput
	if result.ErrorMessage != "" {
		if output != "" {
			output = fmt.Sprintf("%s\n%s", output, result.ErrorMessage)
		} else {
			output = result.ErrorMessage
		}
	}
	if output == "" {
		output = "检查点未通过"
	}

	return result.Passed, output, mustMarshalJSON(map[string]interface{}{
		"scope":   "personal",
		"members": []groupCheckpointMemberResult{result},
	})
}

// validateSimCheckpoint 在单个实例上执行 SimEngine 状态断言检查点。
func (s *instanceService) validateSimCheckpoint(ctx context.Context, instance *entity.ExperimentInstance, cp *entity.TemplateCheckpoint) (bool, string, json.RawMessage) {
	result := groupCheckpointMemberResult{
		StudentID:  instance.StudentID,
		InstanceID: instance.ID,
	}
	if instance.SimSessionID != nil {
		result.SessionID = *instance.SimSessionID
	}

	switch {
	case cp.AssertionConfig == nil:
		result.ErrorMessage = "SimEngine 状态断言未配置"
	case instance.SimSessionID == nil || *instance.SimSessionID == "":
		result.ErrorMessage = "实验实例未绑定 SimEngine 会话"
	default:
		state, err := s.simEngineSvc.GetSessionState(ctx, *instance.SimSessionID)
		if err != nil {
			result.ErrorMessage = err.Error()
		} else {
			passed, evalResult := evaluateSimAssertion(json.RawMessage(cp.AssertionConfig), state.SceneState)
			result.Passed = passed
			result.CheckOutput = formatSimAssertionOutput(evalResult)
			if !passed && evalResult.Reason != "" && result.ErrorMessage == "" {
				result.ErrorMessage = evalResult.Reason
			}
			if passed {
				result.Score = cp.Score
			}
		}
	}

	output := result.CheckOutput
	if result.ErrorMessage != "" {
		if output != "" {
			output = fmt.Sprintf("%s\n%s", output, result.ErrorMessage)
		} else {
			output = result.ErrorMessage
		}
	}
	if output == "" {
		output = "状态断言未通过"
	}

	return result.Passed, output, mustMarshalJSON(map[string]interface{}{
		"scope":   "personal",
		"members": []groupCheckpointMemberResult{result},
	})
}

// validateGroupScriptCheckpoint 在组内所有实例目标容器执行脚本检查点。
func (s *instanceService) validateGroupScriptCheckpoint(ctx context.Context, cp *entity.TemplateCheckpoint, targets []*entity.ExperimentInstance) (bool, string, json.RawMessage) {
	execTargets, err := s.resolveGroupScriptExecutionTargets(ctx, cp, targets)
	if err != nil {
		output := err.Error()
		return false, output, mustMarshalJSON(map[string]interface{}{
			"scope":   "group",
			"message": output,
		})
	}

	memberResults := make([]groupCheckpointMemberResult, 0, len(execTargets))
	allPassed := true
	for _, target := range execTargets {
		passed, _, detail := s.validateScriptCheckpoint(ctx, target, cp)
		member := extractFirstCheckpointMember(detail, target)
		memberResults = append(memberResults, member)
		if !passed {
			allPassed = false
		}
	}
	output := "组级检查点验证通过"
	if !allPassed {
		output = string(mustMarshalJSON(map[string]interface{}{
			"scope":   "group",
			"members": memberResults,
		}))
	}
	if output == "" {
		output = "组级检查点验证未通过，存在组员实例未满足条件"
	}
	return allPassed, output, mustMarshalJSON(map[string]interface{}{
		"scope":   "group",
		"members": memberResults,
	})
}

// resolveGroupScriptExecutionTargets 根据目标容器的角色归属筛选实际执行组级脚本检查点的实例。
// 角色专属容器只能在拥有该角色的实例上执行，避免将同一目标容器错误地广播到所有组员实例。
func (s *instanceService) resolveGroupScriptExecutionTargets(
	ctx context.Context,
	cp *entity.TemplateCheckpoint,
	targets []*entity.ExperimentInstance,
) ([]*entity.ExperimentInstance, error) {
	if cp == nil || cp.TargetContainer == nil || *cp.TargetContainer == "" || len(targets) == 0 {
		return targets, nil
	}

	templateContainer, err := s.templateContainerRepo.GetByTemplateAndName(ctx, targets[0].TemplateID, *cp.TargetContainer)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("组级检查点目标容器不存在")
		}
		return nil, err
	}
	if templateContainer.RoleID == nil {
		return targets, nil
	}

	if targets[0].GroupID == nil {
		return nil, fmt.Errorf("组级检查点要求实例属于实验分组")
	}
	members, err := s.groupMemberRepo.ListByGroupID(ctx, *targets[0].GroupID)
	if err != nil {
		return nil, err
	}
	memberRoleByStudent := make(map[int64]int64, len(members))
	for _, member := range members {
		if member == nil || member.RoleID == nil {
			continue
		}
		memberRoleByStudent[member.StudentID] = *member.RoleID
	}

	filtered := make([]*entity.ExperimentInstance, 0, len(targets))
	for _, target := range targets {
		if target == nil {
			continue
		}
		if memberRoleByStudent[target.StudentID] == *templateContainer.RoleID {
			filtered = append(filtered, target)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("组级检查点目标容器未部署到任何组员实例")
	}
	return filtered, nil
}

// validateGroupSimCheckpoint 在组内所有 SimEngine 会话上执行状态断言。
func (s *instanceService) validateGroupSimCheckpoint(ctx context.Context, cp *entity.TemplateCheckpoint, targets []*entity.ExperimentInstance) (bool, string, json.RawMessage) {
	memberResults := make([]groupCheckpointMemberResult, 0, len(targets))
	allPassed := true
	for _, target := range targets {
		passed, _, detail := s.validateSimCheckpoint(ctx, target, cp)
		member := extractFirstCheckpointMember(detail, target)
		memberResults = append(memberResults, member)
		if !passed {
			allPassed = false
		}
	}
	output := "组级检查点验证通过"
	if !allPassed {
		output = string(mustMarshalJSON(map[string]interface{}{
			"scope":   "group",
			"members": memberResults,
		}))
	}
	if output == "" {
		output = "组级检查点验证未通过，存在组员仿真状态未满足断言"
	}
	return allPassed, output, mustMarshalJSON(map[string]interface{}{
		"scope":   "group",
		"members": memberResults,
	})
}

// persistCheckpointState 将检查点执行结果写入所有目标实例。
func (s *instanceService) persistCheckpointState(ctx context.Context, cp *entity.TemplateCheckpoint, state *checkpointExecutionState, checkedAt time.Time) []*entity.CheckpointResult {
	targets := state.Targets
	if len(targets) == 0 {
		return nil
	}

	results := make([]*entity.CheckpointResult, 0, len(targets))
	for _, target := range targets {
		result, err := s.checkResultRepo.GetByInstanceAndCheckpoint(ctx, target.ID, cp.ID)
		if err == nil {
			fields := map[string]interface{}{
				"is_passed":        state.IsPassed,
				"score":            derefFloat64(state.Score),
				"check_output":     state.CheckOutput,
				"assertion_result": state.AssertionResult,
				"checked_at":       checkedAt,
				"updated_at":       checkedAt,
			}
			if state.Score == nil {
				fields["score"] = nil
			}
			_ = s.checkResultRepo.UpdateFields(ctx, result.ID, fields)
			result.IsPassed = checkpointBoolPtr(state.IsPassed)
			result.Score = cloneFloat64Ptr(state.Score)
			result.CheckOutput = cloneStringPtr(state.CheckOutput)
			result.AssertionResult = cloneDatatypesJSON(state.AssertionResult)
			result.CheckedAt = checkedAt
			result.UpdatedAt = checkedAt
			results = append(results, result)
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		scoreCopy := cloneFloat64Ptr(state.Score)
		outputCopy := cloneStringPtr(state.CheckOutput)
		create := &entity.CheckpointResult{
			ID:              snowflake.Generate(),
			InstanceID:      target.ID,
			CheckpointID:    cp.ID,
			StudentID:       target.StudentID,
			IsPassed:        checkpointBoolPtr(state.IsPassed),
			Score:           scoreCopy,
			CheckOutput:     outputCopy,
			AssertionResult: cloneDatatypesJSON(state.AssertionResult),
			CheckedAt:       checkedAt,
			CreatedAt:       checkedAt,
			UpdatedAt:       checkedAt,
		}
		_ = s.checkResultRepo.Create(ctx, create)
		results = append(results, create)
	}
	return results
}

// simAssertionConfig 解析后的 SimEngine 状态断言配置。
//
// 契约：docs/modules/04-实验环境/02-数据库设计.md §2.6 template_checkpoints.assertion_config
//
//	{
//	  "scene_code": "<必填，定位会话内的目标场景>",
//	  "conditions": [
//	    {"path": "$.x.y", "operator": "<eq|ne|gt|gte|lt|lte|contains>", "value": <any>, "description": "..."}
//	  ],
//	  "require_all": true   // 可选，缺省 true
//	}
type simAssertionConfig struct {
	SceneCode  string                  `json:"scene_code"`
	Conditions []simAssertionCondition `json:"conditions"`
	RequireAll *bool                   `json:"require_all"`
}

type simAssertionCondition struct {
	Path        string          `json:"path"`
	Operator    string          `json:"operator"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description,omitempty"`
}

type simAssertionConditionResult struct {
	Path        string `json:"path"`
	Operator    string `json:"operator"`
	Description string `json:"description,omitempty"`
	Passed      bool   `json:"passed"`
	Reason      string `json:"reason,omitempty"`
}

type simAssertionResult struct {
	Passed     bool                          `json:"passed"`
	Reason     string                        `json:"reason,omitempty"`
	Conditions []simAssertionConditionResult `json:"conditions,omitempty"`
}

// sessionStateEnvelope 与 sim-engine simcore.StateManager.BuildSceneSummary 输出 JSON 结构对齐。
type sessionStateEnvelope struct {
	Scenes []sceneStateSummary `json:"scenes"`
}

type sceneStateSummary struct {
	SceneCode       string          `json:"scene_code"`
	Tick            int64           `json:"tick"`
	RenderStateJSON json.RawMessage `json:"render_state_json"`
}

// evaluateSimAssertion 按文档 DSL 评估 SimEngine 状态断言。
//
// 评估流程：
//  1. 解析 assertion_config，校验 scene_code / conditions 必填；
//  2. 解析会话状态 envelope，按 scene_code 定位目标场景；
//  3. 解码场景 render_state_json，并把 wrapper 的 tick / scene_code 注入根字段，
//     这样 `$.tick` 这种最常用的进度断言无需场景容器在 RenderEnvelope 里重复写 tick；
//  4. 每条 condition 通过 pkg/assertion 走 JSONPath 取值 + operator 比较，
//     按 require_all 汇总（默认 true）；
//  5. 返回 (passed, result)，result 包含逐条条件的详细原因，便于学生看到失败点。
//
// 算子集合 / JSONPath 语法见 backend/internal/pkg/assertion，与 CTF 模块共享同一套基础设施。
func evaluateSimAssertion(assertionConfig json.RawMessage, sceneState json.RawMessage) (bool, simAssertionResult) {
	if len(assertionConfig) == 0 {
		return false, simAssertionResult{Reason: "断言配置为空"}
	}
	var cfg simAssertionConfig
	if err := json.Unmarshal(assertionConfig, &cfg); err != nil {
		return false, simAssertionResult{Reason: fmt.Sprintf("断言配置解析失败: %v", err)}
	}
	cfg.SceneCode = strings.TrimSpace(cfg.SceneCode)
	if cfg.SceneCode == "" {
		return false, simAssertionResult{Reason: "断言缺少 scene_code"}
	}
	if len(cfg.Conditions) == 0 {
		return false, simAssertionResult{Reason: "断言缺少 conditions"}
	}
	if len(sceneState) == 0 {
		return false, simAssertionResult{Reason: "SimEngine 会话状态为空"}
	}

	var envelope sessionStateEnvelope
	if err := json.Unmarshal(sceneState, &envelope); err != nil {
		return false, simAssertionResult{Reason: fmt.Sprintf("会话状态解析失败: %v", err)}
	}
	scene, ok := findSceneByCode(envelope.Scenes, cfg.SceneCode)
	if !ok {
		return false, simAssertionResult{Reason: fmt.Sprintf("会话未激活场景 %q", cfg.SceneCode)}
	}
	context := map[string]any{}
	if len(scene.RenderStateJSON) > 0 {
		if err := json.Unmarshal(scene.RenderStateJSON, &context); err != nil {
			return false, simAssertionResult{Reason: fmt.Sprintf("场景渲染态解析失败: %v", err)}
		}
	}
	context["tick"] = scene.Tick
	context["scene_code"] = scene.SceneCode

	requireAll := true
	if cfg.RequireAll != nil {
		requireAll = *cfg.RequireAll
	}

	results := make([]simAssertionConditionResult, 0, len(cfg.Conditions))
	passedCount := 0
	for _, condition := range cfg.Conditions {
		item := evaluateSimAssertionCondition(condition, context)
		results = append(results, item)
		if item.Passed {
			passedCount++
		} else if requireAll {
			break
		}
	}

	overall := simAssertionResult{Conditions: results}
	if requireAll {
		overall.Passed = passedCount == len(cfg.Conditions)
		if !overall.Passed {
			overall.Reason = "存在未通过的条件"
		}
	} else {
		overall.Passed = passedCount > 0
		if !overall.Passed {
			overall.Reason = "所有条件均未通过"
		}
	}
	return overall.Passed, overall
}

// findSceneByCode 在场景摘要列表里按 code 精确匹配。
func findSceneByCode(scenes []sceneStateSummary, code string) (sceneStateSummary, bool) {
	for _, s := range scenes {
		if s.SceneCode == code {
			return s, true
		}
	}
	return sceneStateSummary{}, false
}

// evaluateSimAssertionCondition 评估单条断言条件。
func evaluateSimAssertionCondition(condition simAssertionCondition, context map[string]any) simAssertionConditionResult {
	result := simAssertionConditionResult{
		Path:        condition.Path,
		Operator:    condition.Operator,
		Description: condition.Description,
	}
	if condition.Path == "" {
		result.Reason = "条件缺少 path"
		return result
	}
	if !assertion.IsValidOperator(condition.Operator) {
		result.Reason = fmt.Sprintf("不支持的算子 %q", condition.Operator)
		return result
	}
	pathExpr, err := assertion.Compile(condition.Path)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	actual, err := pathExpr.Lookup(context)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	var expected any
	if len(condition.Value) > 0 {
		if err := json.Unmarshal(condition.Value, &expected); err != nil {
			result.Reason = fmt.Sprintf("expected value 解析失败: %v", err)
			return result
		}
	}
	passed, reason := assertion.Compare(actual, condition.Operator, expected)
	result.Passed = passed
	if !passed {
		result.Reason = reason
	}
	return result
}

// formatSimAssertionOutput 把评估结果格式化成结构化 JSON 字符串作为检查点输出，
// 便于前端检查点面板展示通过/未通过条件列表。失败兜底为可读 reason。
func formatSimAssertionOutput(result simAssertionResult) string {
	data, err := json.Marshal(result)
	if err != nil {
		if result.Reason != "" {
			return result.Reason
		}
		return "状态断言评估失败"
	}
	return string(data)
}

// extractFirstCheckpointMember 从断言详情中提取第一个组员结果。
func extractFirstCheckpointMember(detail json.RawMessage, sourceInstance *entity.ExperimentInstance) groupCheckpointMemberResult {
	member := groupCheckpointMemberResult{
		StudentID:  sourceInstance.StudentID,
		InstanceID: sourceInstance.ID,
	}
	if detail == nil {
		return member
	}
	var payload struct {
		Members []groupCheckpointMemberResult `json:"members"`
	}
	if err := json.Unmarshal(detail, &payload); err != nil || len(payload.Members) == 0 {
		return member
	}
	return payload.Members[0]
}

// pickCheckpointResult 从镜像结果中优先挑出当前实例对应的检查点结果。
func pickCheckpointResult(results []*entity.CheckpointResult, instanceID int64) *entity.CheckpointResult {
	for _, result := range results {
		if result.InstanceID == instanceID {
			return result
		}
	}
	if len(results) > 0 {
		return results[0]
	}
	return &entity.CheckpointResult{
		ID:         snowflake.Generate(),
		InstanceID: instanceID,
		IsPassed:   checkpointBoolPtr(false),
	}
}

// mustMarshalJSON 将结构体编码为 JSON；编码失败时返回空对象，避免影响主流程。
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

// cloneFloat64Ptr 复制浮点数指针，避免多个结果共享同一地址。
func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}

// cloneStringPtr 复制字符串指针，避免多个结果共享同一地址。
func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}

// cloneDatatypesJSON 复制 JSONB 内容，避免多个结果共享底层切片。
func cloneDatatypesJSON(v json.RawMessage) datatypes.JSON {
	if v == nil {
		return nil
	}
	clone := make(datatypes.JSON, len(v))
	copy(clone, v)
	return clone
}

// checkpointBoolPtr 返回布尔值指针，便于与实体层可空字段对齐。
func checkpointBoolPtr(v bool) *bool {
	return &v
}

// derefFloat64 解引用浮点数指针；空值时返回 0，仅用于构造更新字段。
func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
