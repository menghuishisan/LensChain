package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/lenschain/sim-engine/core/internal/collector"
)

// =====================================================================
// 教师监控
// =====================================================================

// BuildTeacherSummary 生成单个会话的教师监控摘要。
func (e *Engine) BuildTeacherSummary(sessionID string) (TeacherSummary, bool) {
	e.mu.RLock()
	_, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return TeacherSummary{}, false
	}
	sessionState, ok := e.SessionState(sessionID)
	if !ok {
		return TeacherSummary{}, false
	}
	preview := e.buildTeacherPreview(sessionID)
	return TeacherSummary{
		SessionID:           sessionID,
		InstanceID:          sessionState.InstanceID,
		Tick:                sessionState.Tick,
		Speed:               sessionState.Speed,
		ActiveSceneCodes:    sessionState.ActiveSceneCodes,
		LinkGroupCodes:      sessionState.LinkGroupCodes,
		CollectionRunning:   e.collectorRunning(sessionID),
		PreviewSceneCode:    preview.SceneCode,
		PreviewEnvelopeJSON: json.RawMessage(cloneBytes(preview.RenderEnvelopeJSON)),
		LastAction:          sessionState.LastAction,
		UpdatedAt:           sessionState.UpdatedAt,
	}, true
}

// PublishTeacherSummaries 将所有会话摘要推送到各自消息总线（用 event 类型 + payload.event=teacher_summary）。
func (e *Engine) PublishTeacherSummaries() {
	for _, sessionID := range e.sessionIDs() {
		summary, ok := e.BuildTeacherSummary(sessionID)
		if !ok {
			continue
		}
		summaryJSON, err := json.Marshal(summary)
		if err != nil {
			continue
		}
		var summaryMap map[string]any
		if err := json.Unmarshal(summaryJSON, &summaryMap); err != nil {
			continue
		}
		e.publishEvent(sessionID, "teacher_summary", summaryMap)
	}
}

// StartTeacherSummaryLoop 按固定周期持续推送教师监控摘要。
func (e *Engine) StartTeacherSummaryLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.PublishTeacherSummaries()
		}
	}
}

// teacherPreview 表示教师监控卡片展示的默认预览内容。
type teacherPreview struct {
	SceneCode          string
	RenderEnvelopeJSON []byte
}

// buildTeacherPreview 为教师监控卡片选择一个默认缩略图场景。
func (e *Engine) buildTeacherPreview(sessionID string) teacherPreview {
	runtimes := e.scenes.ListBySession(sessionID)
	if len(runtimes) == 0 {
		return teacherPreview{}
	}
	return teacherPreview{
		SceneCode:          runtimes[0].Config.SceneCode,
		RenderEnvelopeJSON: cloneBytes(runtimes[0].State.RenderEnvelopeJSON),
	}
}

// collectorRunning 判断指定会话的采集器当前是否处于运行状态。
func (e *Engine) collectorRunning(sessionID string) bool {
	session, ok := e.collect.Get(sessionID)
	return ok && session.Running
}

// =====================================================================
// 数据采集
// =====================================================================

// StartDataCollection 标记会话采集通道已启动。
func (e *Engine) StartDataCollection(sessionID string, configJSON []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return errors.New("session runtime not found")
	}
	mode := "collection"
	for _, cfg := range runtime.sceneConfigs {
		if strings.TrimSpace(cfg.DataSourceMode) == "dual" {
			mode = "dual"
			break
		}
	}
	if err := e.collect.Start(sessionID, mode, configJSON); err != nil {
		return err
	}
	runtime.updatedAt = time.Now().UTC()
	return nil
}

// StopDataCollection 标记会话采集通道已停止。
func (e *Engine) StopDataCollection(sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return errors.New("session runtime not found")
	}
	if err := e.collect.Stop(sessionID); err != nil {
		return err
	}
	runtime.updatedAt = time.Now().UTC()
	return nil
}

// InjectCollectionEvent 将 Collector Agent 的标准化事件按场景配置注入仿真状态。
func (e *Engine) InjectCollectionEvent(sessionID string, event collector.Event) error {
	unlock, err := e.lockRuntimeOperation(sessionID)
	if err != nil {
		return err
	}
	defer unlock()

	e.mu.Lock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		e.mu.Unlock()
		return errors.New("session runtime not found")
	}
	collectorSession, collectorOK := e.collect.Get(sessionID)
	if !collectorOK || !collectorSession.Running {
		e.mu.Unlock()
		return errors.New("data collection is not running")
	}
	runtime.updatedAt = time.Now().UTC()
	runtime.lastAction = "collector:" + event.DataType
	e.mu.Unlock()

	patch, err := collector.Normalize(event)
	if err != nil {
		_ = e.collect.RecordError(sessionID, err)
		e.publishCollectorInterrupted(sessionID, err)
		return err
	}
	_ = e.collect.RecordEvent(sessionID, event)
	affectedScenes, err := e.resolveCollectionScenes(sessionID, collectorSession.ConfigJSON, event)
	if err != nil {
		_ = e.collect.RecordError(sessionID, err)
		e.publishCollectorInterrupted(sessionID, err)
		return err
	}
	for _, sceneCode := range affectedScenes {
		state, injectErr := e.scenes.InjectCollectionPatch(sessionID, sceneCode, patch.PatchJSON)
		if injectErr != nil {
			_ = e.collect.RecordError(sessionID, injectErr)
			e.publishCollectorInterrupted(sessionID, injectErr)
			return injectErr
		}
		e.publishRender(sessionID, sceneCode, e.currentTick(sessionID), state.RenderEnvelopeJSON)
	}
	if err := e.recordTickSnapshot(sessionID); err != nil {
		return err
	}
	return nil
}

// =====================================================================
// 教师干预 — 完整实现（对齐 06.md §14.5 八种 action_code）
// =====================================================================

// TeacherBroadcastMessage 向指定会话广播教师消息（broadcast_message）。
func (e *Engine) TeacherBroadcastMessage(sessionID string, message string) error {
	e.mu.RLock()
	_, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	e.publishEvent(sessionID, "teacher_broadcast", map[string]any{
		"message": message,
		"source":  "__teacher__",
	})
	return nil
}

// TeacherForceStep 教师干预：帮学生跨过卡点，强制推进指定会话的指定场景一个 tick。
func (e *Engine) TeacherForceStep(ctx context.Context, sessionID string, sceneCode string) error {
	unlock, err := e.lockRuntimeOperation(sessionID)
	if err != nil {
		return err
	}
	defer unlock()

	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}

	if err := runtime.clock.Step(); err != nil {
		return err
	}
	tick := runtime.clock.Tick()
	sharedStateJSON := e.sharedStateForScene(sessionID, sceneCode)
	incoming := e.popPendingLinkTriggers(sessionID, sceneCode)
	result, err := e.scenes.Step(ctx, sessionID, sceneCode, tick, sharedStateJSON, incoming)
	if err != nil {
		e.handleSceneRuntimeFailure(sessionID, sceneCode, err)
		return err
	}
	e.applyLinkDiff(sessionID, sceneCode, "teacher_force_step", result.SharedStateDiffJSON)
	e.publishRender(sessionID, sceneCode, result.Tick, result.RenderEnvelopeJSON)

	if err := e.recordTickSnapshot(sessionID); err != nil {
		return err
	}
	e.markRuntimeAction(sessionID, "teacher_force_step")
	e.publishEvent(sessionID, "teacher_intervention", map[string]any{
		"action":     "force_step",
		"scene_code": sceneCode,
		"tick":       tick,
	})
	log.Printf("[TeacherForceStep] session=%s scene=%s tick=%d", sessionID, sceneCode, tick)
	return nil
}

// TeacherKickStudent 教师干预：踢出违规学生（销毁其会话）。
func (e *Engine) TeacherKickStudent(sessionID string, reason string) error {
	e.mu.RLock()
	_, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}

	// 先广播踢出事件让前端收到通知。
	e.publishEvent(sessionID, "teacher_kick", map[string]any{
		"reason": reason,
		"source": "__teacher__",
	})

	// 然后销毁会话。
	if err := e.DestroySession(sessionID); err != nil {
		log.Printf("[TeacherKickStudent] destroy session %s failed: %v", sessionID, err)
		return err
	}
	log.Printf("[TeacherKickStudent] session=%s reason=%s", sessionID, reason)
	return nil
}
