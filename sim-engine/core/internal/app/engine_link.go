package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/lenschain/sim-engine/core/internal/link"
	"github.com/lenschain/sim-engine/core/internal/scene"
	"github.com/lenschain/sim-engine/core/internal/session"
	"github.com/lenschain/sim-engine/core/internal/simcore"
)

// =====================================================================
// 场景推进与联动
// =====================================================================

// stepScenes 推进会话内所有场景一个 tick，并广播渲染。
func (e *Engine) stepScenes(ctx context.Context, sessionID string, tick int64) error {
	for _, runtimeRef := range e.scenes.ListBySession(sessionID) {
		if !shouldAdvanceScene(runtimeRef.Meta.TimeControlMode) {
			continue
		}
		sharedStateJSON := e.sharedStateForScene(sessionID, runtimeRef.Config.SceneCode)
		incoming := e.popPendingLinkTriggers(sessionID, runtimeRef.Config.SceneCode)
		result, err := e.scenes.Step(ctx, sessionID, runtimeRef.Config.SceneCode, tick, sharedStateJSON, incoming)
		if err != nil {
			e.handleSceneRuntimeFailure(sessionID, runtimeRef.Config.SceneCode, err)
			return err
		}
		e.applyLinkDiff(sessionID, runtimeRef.Config.SceneCode, "", result.SharedStateDiffJSON)
		e.publishRender(sessionID, runtimeRef.Config.SceneCode, result.Tick, result.RenderEnvelopeJSON)

		// 记录场景步进事件到 EventBus。
		e.recordSceneStepEvent(sessionID, runtimeRef.Config.SceneCode, tick)
	}
	return nil
}

// recordSceneStepEvent 将场景步进事件写入会话 EventBus。
func (e *Engine) recordSceneStepEvent(sessionID string, sceneCode string, tick int64) {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || runtime.eventBus == nil {
		return
	}
	runtime.eventBus.Append([]simcore.Event{{
		EventID:     fmt.Sprintf("step-%s-%d", sceneCode, tick),
		EventType:   "scene_step",
		SceneCode:   sceneCode,
		Tick:        tick,
		TimestampMS: time.Now().UTC().UnixMilli(),
	}})
}

// applyLinkDiff 将场景返回的共享状态 diff 通过 link.Engine 校验 owner 后 fan-out。
//
// 受影响的接收方场景的 LinkTrigger 暂存到 runtime.pendingLinkTrigs，下一次 Step 时注入。
// 同时广播一条 event 消息（event=link_update）便于前端联动可视化（M2 / M8）。
func (e *Engine) applyLinkDiff(sessionID string, sourceScene string, sourceAction string, diffJSON []byte) {
	if len(diffJSON) == 0 {
		return
	}
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return
	}
	sceneConfig, ok := runtime.sceneConfigs[sourceScene]
	if !ok || sceneConfig.LinkGroupCode == "" {
		return
	}
	if !e.isLinkGroupReady(runtime, sceneConfig.LinkGroupCode) {
		return
	}
	groupRuntimeCode := e.linkRuntimeCode(sessionID, sceneConfig.LinkGroupCode)
	fanOut, err := e.linker.ApplyDiffJSON(groupRuntimeCode, sourceScene, sourceAction, diffJSON, "", nil)
	if err != nil {
		e.publishEvent(sessionID, "link_owner_violation", map[string]any{
			"link_group":    sceneConfig.LinkGroupCode,
			"source_scene":  sourceScene,
			"source_action": sourceAction,
			"error":         err.Error(),
		})
		return
	}

	// 把 fan-out triggers 暂存到接收方场景的 pendingLinkTrigs。
	e.mu.Lock()
	runtime = e.runtimes[sessionID]
	for receiverScene, trigger := range fanOut.Triggers {
		runtime.pendingLinkTrigs[receiverScene] = append(runtime.pendingLinkTrigs[receiverScene], scene.LinkTriggerRef{
			ID:             trigger.ID,
			SourceScene:    trigger.SourceScene,
			SourceAction:   trigger.SourceAction,
			LinkGroup:      trigger.LinkGroup,
			ChangedFields:  append([]string(nil), trigger.ChangedFields...),
			PayloadJSON:    encodeMapJSON(trigger.Payload),
			TimestampMS:    trigger.Timestamp,
			SourceAnchorID: trigger.SourceAnchorID,
			TargetAnchorID: trigger.TargetAnchorID,
		})
	}
	e.mu.Unlock()

	receivers := make([]string, 0, len(fanOut.Triggers))
	for receiver := range fanOut.Triggers {
		receivers = append(receivers, receiver)
	}
	sort.Strings(receivers)
	e.publishEvent(sessionID, "link_update", map[string]any{
		"link_group":      sceneConfig.LinkGroupCode,
		"source_scene":    sourceScene,
		"source_action":   sourceAction,
		"changed_fields":  flattenJSONPaths(diffJSON),
		"affected_scenes": receivers,
		"shared_state":    fanOut.State,
	})
}

// popPendingLinkTriggers 取出并清空指定场景下的待处理联动事件。
func (e *Engine) popPendingLinkTriggers(sessionID string, sceneCode string) []scene.LinkTriggerRef {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return nil
	}
	if runtime.pendingLinkTrigs == nil {
		return nil
	}
	triggers := runtime.pendingLinkTrigs[sceneCode]
	delete(runtime.pendingLinkTrigs, sceneCode)
	return triggers
}

// shouldAdvanceScene 判断当前场景是否允许被会话时钟推进。
func shouldAdvanceScene(mode string) bool {
	switch simcore.TimeControlMode(mode) {
	case simcore.TimeControlModeProcess, simcore.TimeControlModeContinuous:
		return true
	default:
		return false
	}
}

// handleSceneRuntimeFailure 在场景容器异常后尝试自动重启并恢复最近快照状态。
func (e *Engine) handleSceneRuntimeFailure(sessionID string, sceneCode string, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return
	}

	sharedStateJSON := e.sharedStateForScene(sessionID, sceneCode)
	_, restartErr := e.scenes.Restart(ctx, sessionID, sceneCode, runtime.instanceID, runtime.studentID, runtime.seed, sharedStateJSON)
	recoveryErr := error(nil)
	if restartErr != nil {
		e.setRuntimeStatus(sessionID, session.StatusError)
	} else {
		recoveryErr = e.RecoverLatestTickSnapshot(sessionID)
		if recoveryErr != nil {
			e.setRuntimeStatus(sessionID, session.StatusError)
		}
		e.markSceneReady(sessionID, sceneCode, true, "")
	}
	payload := map[string]any{
		"scene_code": sceneCode,
		"error":      cause.Error(),
		"recovered":  restartErr == nil && recoveryErr == nil,
	}
	if restartErr != nil {
		payload["recovery_error"] = restartErr.Error()
	} else if recoveryErr != nil {
		payload["recovery_error"] = recoveryErr.Error()
	}
	e.publishEvent(sessionID, "scene_runtime_failure", payload)
}

// =====================================================================
// 联动组注册与共享状态
// =====================================================================

// registerLinkGroups 根据会话请求注册联动组共享状态空间。
func (e *Engine) registerLinkGroups(sessionID string, req StartSessionRequest) ([]string, error) {
	if !req.LinkageEnabled || len(req.LinkGroups) == 0 {
		return nil, nil
	}
	registered := make([]string, 0, len(req.LinkGroups))
	for _, spec := range req.LinkGroups {
		fields := make(map[string]link.FieldSchema, len(spec.Fields))
		for _, field := range spec.Fields {
			fields[field.Name] = link.FieldSchema{
				Name:  field.Name,
				Type:  field.Type,
				Owner: field.Owner,
			}
		}
		err := e.linker.RegisterGroup(link.Group{
			Code:           e.linkRuntimeCode(sessionID, spec.Code),
			Version:        spec.Version,
			Members:        append([]string(nil), spec.Members...),
			Fields:         fields,
			ForceClockSync: spec.ForceClockSync,
		}, spec.InitialState)
		if err != nil {
			for _, registeredCode := range registered {
				e.linker.DeleteGroup(e.linkRuntimeCode(sessionID, registeredCode))
			}
			return nil, fmt.Errorf("注册联动组 %s 失败: %w", spec.Code, err)
		}
		registered = append(registered, spec.Code)
	}
	return registered, nil
}

// cleanupFailedSessionStart 回滚启动过程中已创建的会话、联动组和场景运行时。
func (e *Engine) cleanupFailedSessionStart(sessionID string, linkGroupCodes []string) {
	for _, groupCode := range linkGroupCodes {
		e.linker.DeleteGroup(e.linkRuntimeCode(sessionID, groupCode))
	}
	_ = e.scenes.DestroySession(sessionID)
	e.collect.Delete(sessionID)
	_ = e.sessions.Destroy(sessionID)
}

// sharedStateForScene 返回某个场景当前可见的共享状态 JSON（合并视图）。
func (e *Engine) sharedStateForScene(sessionID string, sceneCode string) []byte {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return nil
	}
	config, ok := runtime.sceneConfigs[sceneCode]
	if !ok || config.LinkGroupCode == "" {
		return nil
	}
	if !e.isLinkGroupReady(runtime, config.LinkGroupCode) {
		return nil
	}
	state, ok := e.linker.SharedState(e.linkRuntimeCode(sessionID, config.LinkGroupCode))
	if !ok {
		return nil
	}
	return encodeMapJSON(state)
}

// isLinkGroupReady 判断联动组中的全部场景是否都已就绪。
func (e *Engine) isLinkGroupReady(runtime *runtime, groupCode string) bool {
	if groupCode == "" {
		return false
	}
	for sceneCode, config := range runtime.sceneConfigs {
		if config.LinkGroupCode != groupCode {
			continue
		}
		if !runtime.sceneReady[sceneCode] {
			return false
		}
	}
	return true
}

// linkRuntimeCode 返回某个会话内部使用的联动组唯一键。
func (e *Engine) linkRuntimeCode(sessionID string, groupCode string) string {
	return sessionID + "::" + groupCode
}

// syncLinkStatesFromSnapshot 从快照场景中重建联动组共享状态（取首个非空 SharedStateJSON）。
func (e *Engine) syncLinkStatesFromSnapshot(sessionID string, scenes []SnapshotScene) {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return
	}
	groupStates := make(map[string]map[string]any)
	for _, sceneSnapshot := range scenes {
		config, configOK := runtime.sceneConfigs[sceneSnapshot.SceneCode]
		if !configOK || config.LinkGroupCode == "" || len(sceneSnapshot.SharedStateJSON) == 0 {
			continue
		}
		if _, exists := groupStates[config.LinkGroupCode]; exists {
			continue
		}
		var state map[string]any
		if err := json.Unmarshal(sceneSnapshot.SharedStateJSON, &state); err != nil {
			continue
		}
		groupStates[config.LinkGroupCode] = state
	}
	for groupCode, state := range groupStates {
		_ = e.linker.ResetGroup(e.linkRuntimeCode(sessionID, groupCode), state)
	}
}

// ForceSetLinkState 教师干预接口：强制覆写指定联动组的共享状态字段（绕过 owner 校验）。
//
// 仅供 force_link_state 教师干预使用（详 06.md §14.5）。
// 写入后向所有成员场景广播 link_update 事件。
func (e *Engine) ForceSetLinkState(sessionID string, linkGroupCode string, fieldsJSON []byte) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if !runtime.linkageEnabled {
		return errors.New("linkage is not enabled for this session")
	}

	groupRuntimeCode := e.linkRuntimeCode(sessionID, linkGroupCode)
	currentState, stateOK := e.linker.SharedState(groupRuntimeCode)
	if !stateOK {
		return fmt.Errorf("联动组 %s 未注册", linkGroupCode)
	}

	var override map[string]any
	if err := json.Unmarshal(fieldsJSON, &override); err != nil {
		return fmt.Errorf("force_link_state fields JSON 格式错误: %w", err)
	}
	for k, v := range override {
		currentState[k] = v
	}
	if err := e.linker.ResetGroup(groupRuntimeCode, currentState); err != nil {
		return err
	}

	changedFields := make([]string, 0, len(override))
	for k := range override {
		changedFields = append(changedFields, k)
	}
	sort.Strings(changedFields)

	e.publishEvent(sessionID, "link_update", map[string]any{
		"link_group":      linkGroupCode,
		"source_scene":    "__teacher__",
		"source_action":   "force_link_state",
		"changed_fields":  changedFields,
		"affected_scenes": runtime.activeSceneCodes,
		"shared_state":    currentState,
	})
	log.Printf("[ForceSetLinkState] session=%s group=%s fields=%v", sessionID, linkGroupCode, changedFields)
	return nil
}

// UnlockLinkSync 教师干预接口：临时解除指定联动组的时钟强制同步（详 06.md §14.5）。
//
// 解除后成员场景可独立控制时钟。此操作不可逆（需重新注册联动组恢复同步）。
func (e *Engine) UnlockLinkSync(sessionID string, linkGroupCode string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if !runtime.linkageEnabled {
		return errors.New("linkage is not enabled for this session")
	}

	groupRuntimeCode := e.linkRuntimeCode(sessionID, linkGroupCode)
	groupInfo, groupOK := e.linker.GroupInfo(groupRuntimeCode)
	if !groupOK {
		return fmt.Errorf("联动组 %s 未注册", linkGroupCode)
	}
	if !groupInfo.ForceClockSync {
		return nil // 已经是非强制同步
	}

	// 更新 link.Engine 中的 ForceClockSync 标记。
	e.linker.SetForceClockSync(groupRuntimeCode, false)

	e.publishEvent(sessionID, "link_sync_unlocked", map[string]any{
		"link_group": linkGroupCode,
		"source":     "__teacher__",
	})
	log.Printf("[UnlockLinkSync] session=%s group=%s", sessionID, linkGroupCode)
	return nil
}
