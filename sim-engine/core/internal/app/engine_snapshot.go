package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lenschain/sim-engine/core/internal/scene"
	"github.com/lenschain/sim-engine/core/internal/simcore"
)

// =====================================================================
// 快照
// =====================================================================

// CreateSnapshot 为会话创建持久化快照记录。
//
// 锁策略（生产事故复盘后的固化做法）：
//
//   - **绝不**在持 e.mu.Lock 时调用 e.store.Save。MinIO/S3 I/O 一次抖动就能让锁
//     永久不释放，导致 ControlTime / advanceRunnableSessions 全部排队，前端
//     pause/step/reset 像被吞。
//
//   - **绝不**在持 e.mu 任何级别时调用 e.publishEvent / e.currentTick 等会再次拿
//     e.mu 的方法。Go 的 sync.RWMutex 不可重入，自家持锁再 RLock = 同 goroutine
//     死锁。本函数早期版本就是这么把 sim-engine 卡死的。
//
// 实际流程：① 短时持 RLock 读快照元数据；② 锁外做 I/O；③ 短时持 Lock 写
// updatedAt；④ 锁外发布事件。
func (e *Engine) CreateSnapshot(sessionID string, description string) (Snapshot, error) {
	// ① 读取运行时元数据（尽量短）。
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	store := e.store
	e.mu.RUnlock()
	if !ok {
		return Snapshot{}, errors.New("session runtime not found")
	}
	if store == nil {
		return Snapshot{}, errors.New("snapshot store is not configured")
	}

	snapshotID, err := newSnapshotID()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{
		SnapshotID:  snapshotID,
		SessionID:   sessionID,
		Description: description,
		Tick:        runtime.clock.Tick(),
		CreatedAt:   time.Now().UTC(),
	}
	payload := SnapshotPayload{
		SessionID: sessionID,
		Tick:      snapshot.Tick,
		// buildSnapshotScenes 走 scene.Manager，自带 mu；不需要 e.mu 保护。
		Scenes: e.buildSnapshotScenes(sessionID),
	}

	// ② I/O 必须在锁外——store.Save 内部已带 ctx 超时（snapshot_store.go）。
	objectURL, err := store.Save(snapshotID, payload)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.ObjectURL = objectURL

	// ③ 短时持 Lock 写元数据。
	e.mu.Lock()
	if rt, ok := e.runtimes[sessionID]; ok {
		rt.updatedAt = time.Now().UTC()
	}
	e.mu.Unlock()

	// ④ publishEvent 走 hub.Publish + currentTick(自带 RLock)，必须在锁外。
	e.publishEvent(sessionID, "snapshot_created", map[string]any{
		"snapshot_id":   snapshot.SnapshotID,
		"snapshot_type": "manual",
		"tick":          snapshot.Tick,
	})
	return snapshot, nil
}

// RestoreSnapshot 校验快照存在并恢复到快照 tick。
//
// 快照查找走 SnapshotStore（MinIO/S3）唯一权威源；跨会话恢复天然合法（CreateSnapshot
// 已通过 e.store.Save 持久化）。容器回收路径与本函数解耦。
func (e *Engine) RestoreSnapshot(sessionID string, snapshotID string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if e.store == nil {
		return errors.New("snapshot store is not configured")
	}
	payload, err := e.store.Load(snapshotID)
	if err != nil {
		return fmt.Errorf("加载快照 %s 失败: %w", snapshotID, err)
	}
	// 时钟重设到目标 tick：从 0 一步步前进（用 step 保持模式校验）。
	runtime.clock.Reset()
	for runtime.clock.Tick() < payload.Tick {
		if err := runtime.clock.Step(); err != nil {
			return err
		}
	}
	// 恢复每个场景的 SceneState 与 RenderEnvelope。
	for _, sceneSnapshot := range payload.Scenes {
		e.scenes.RestoreState(sessionID, sceneSnapshot.SceneCode, scene.State{
			SceneCode:          sceneSnapshot.SceneCode,
			Tick:               payload.Tick,
			SceneStateJSON:     cloneBytes(sceneSnapshot.SceneStateJSON),
			RenderEnvelopeJSON: cloneBytes(sceneSnapshot.RenderEnvelopeJSON),
		})
		e.publishRender(sessionID, sceneSnapshot.SceneCode, payload.Tick, sceneSnapshot.RenderEnvelopeJSON)
	}
	// 恢复联动组共享状态。
	e.syncLinkStatesFromSnapshot(sessionID, payload.Scenes)

	if snapshotJSON, snapshotErr := e.buildTickSnapshotJSON(sessionID); snapshotErr == nil {
		e.mu.Lock()
		rt := e.runtimes[sessionID]
		rt.lastSnapshotState = cloneBytes(snapshotJSON)
		rt.lastAutoAdvanceAt = time.Now().UTC()
		e.mu.Unlock()
	}
	e.publishEvent(sessionID, "snapshot_restored", map[string]any{
		"snapshot_id":   snapshotID,
		"snapshot_type": "restore",
		"tick":          payload.Tick,
	})
	return nil
}

// StartAutoSnapshotLoop 按固定周期为所有会话创建持久化快照。
func (e *Engine) StartAutoSnapshotLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, sessionID := range e.sessionIDs() {
				_, _ = e.CreateSnapshot(sessionID, "auto")
			}
		}
	}
}

// =====================================================================
// 内部 — 状态摘要 / 快照栈
// =====================================================================

// buildSnapshotScenes 构造会话完整快照所需的场景载荷。
func (e *Engine) buildSnapshotScenes(sessionID string) []SnapshotScene {
	runtimes := e.scenes.ListBySession(sessionID)
	result := make([]SnapshotScene, 0, len(runtimes))
	for _, runtimeRef := range runtimes {
		sharedState := e.sharedStateForScene(sessionID, runtimeRef.Config.SceneCode)
		result = append(result, SnapshotScene{
			SceneCode:          runtimeRef.Config.SceneCode,
			SceneStateJSON:     cloneBytes(runtimeRef.State.SceneStateJSON),
			RenderEnvelopeJSON: cloneBytes(runtimeRef.State.RenderEnvelopeJSON),
			SharedStateJSON:    sharedState,
		})
	}
	return result
}

// buildSceneStateJSON 构造会话当前场景状态摘要 JSON。
func (e *Engine) buildSceneStateJSON(sessionID string) []byte {
	states := make([]simcore.SceneStateSnapshot, 0)
	for _, runtimeRef := range e.scenes.ListBySession(sessionID) {
		states = append(states, simcore.SceneStateSnapshot{
			SceneCode:       runtimeRef.Config.SceneCode,
			Tick:            runtimeRef.State.Tick,
			RenderStateJSON: cloneBytes(runtimeRef.State.RenderEnvelopeJSON),
		})
	}
	return e.stateMgr.BuildSceneSummary(states)
}

// recordTickSnapshot 按文档策略记录 tick 快照。
// 每 tick 增量 diff，每 50 tick 关键帧，最多 1000 tick 滚动。
func (e *Engine) recordTickSnapshot(sessionID string) error {
	currentJSON, err := e.buildTickSnapshotJSON(sessionID)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok || runtime.snapshotStack == nil {
		return nil
	}
	diffJSON, err := e.stateMgr.BuildDiff(runtime.lastSnapshotState, currentJSON)
	if err != nil {
		return err
	}
	var keyframeJSON []byte
	if runtime.clock.Tick()%50 == 0 {
		keyframeJSON = cloneBytes(currentJSON)
	}
	runtime.snapshotStack.Save(runtime.clock.Tick(), keyframeJSON, diffJSON)
	runtime.lastSnapshotState = cloneBytes(currentJSON)
	return nil
}

// buildTickSnapshotJSON 构造当前 tick 的完整状态 JSON。
func (e *Engine) buildTickSnapshotJSON(sessionID string) ([]byte, error) {
	states := make([]simcore.SceneStateSnapshot, 0)
	for _, runtimeRef := range e.scenes.ListBySession(sessionID) {
		sharedJSON := e.sharedStateForScene(sessionID, runtimeRef.Config.SceneCode)
		states = append(states, simcore.SceneStateSnapshot{
			SceneCode:       runtimeRef.Config.SceneCode,
			Tick:            runtimeRef.State.Tick,
			StateJSON:       cloneBytes(runtimeRef.State.SceneStateJSON),
			RenderStateJSON: cloneBytes(runtimeRef.State.RenderEnvelopeJSON),
			SharedStateJSON: sharedJSON,
		})
	}
	return e.stateMgr.BuildTickSnapshot(sessionID, e.currentTick(sessionID), states)
}

// restoreTickState 将某个 tick 的完整状态 JSON 恢复回会话运行时。
//
// 解码 BuildTickSnapshot 的结果（map[scene_code] => {state_json, render_state_json, shared_state_json}），
// 通过 scene.Manager.RestoreState 写回，并广播 render 帧。
func (e *Engine) restoreTickState(sessionID string, targetTick int64, snapshotJSON []byte) error {
	var payload struct {
		SessionID string `json:"session_id"`
		Tick      int64  `json:"tick"`
		Scenes    map[string]struct {
			SceneCode       string          `json:"scene_code"`
			Tick            int64           `json:"tick"`
			StateJSON       json.RawMessage `json:"state_json"`
			RenderStateJSON json.RawMessage `json:"render_state_json"`
			SharedStateJSON json.RawMessage `json:"shared_state_json"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(snapshotJSON, &payload); err != nil {
		return err
	}
	rebuiltScenes := make([]SnapshotScene, 0, len(payload.Scenes))
	for sceneCode, sceneSnapshot := range payload.Scenes {
		e.scenes.RestoreState(sessionID, sceneCode, scene.State{
			SceneCode:          sceneCode,
			Tick:               targetTick,
			SceneStateJSON:     cloneBytes(sceneSnapshot.StateJSON),
			RenderEnvelopeJSON: cloneBytes(sceneSnapshot.RenderStateJSON),
		})
		e.publishRender(sessionID, sceneCode, targetTick, sceneSnapshot.RenderStateJSON)
		rebuiltScenes = append(rebuiltScenes, SnapshotScene{
			SceneCode:          sceneCode,
			SceneStateJSON:     cloneBytes(sceneSnapshot.StateJSON),
			RenderEnvelopeJSON: cloneBytes(sceneSnapshot.RenderStateJSON),
			SharedStateJSON:    cloneBytes(sceneSnapshot.SharedStateJSON),
		})
	}
	e.syncLinkStatesFromSnapshot(sessionID, rebuiltScenes)
	return nil
}

// RecoverLatestTickSnapshot 用最近一次 tick 快照恢复当前会话状态。
func (e *Engine) RecoverLatestTickSnapshot(sessionID string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if len(runtime.lastSnapshotState) == 0 {
		return nil
	}
	return e.restoreTickState(sessionID, runtime.clock.Tick(), cloneBytes(runtime.lastSnapshotState))
}

// resetSession 将会话恢复到 tick=0 的初始化状态。
func (e *Engine) resetSession(sessionID string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if len(runtime.initialSnapshot) == 0 {
		return errors.New("initial snapshot is not available")
	}
	if err := e.restoreTickState(sessionID, 0, cloneBytes(runtime.initialSnapshot)); err != nil {
		return err
	}
	e.mu.Lock()
	rt := e.runtimes[sessionID]
	rt.clock.Reset()
	rt.lastSnapshotState = cloneBytes(rt.initialSnapshot)
	rt.snapshotStack = simcore.NewSnapshotStack(1000, 50)
	rt.snapshotStack.Save(0, cloneBytes(rt.initialSnapshot), cloneBytes(rt.initialSnapshot))
	rt.lastAutoAdvanceAt = time.Now().UTC()
	rt.pendingLinkTrigs = make(map[string][]scene.LinkTriggerRef)
	// 联动组也重置为初始状态。
	for _, spec := range rt.linkGroups {
		_ = e.linker.ResetGroup(e.linkRuntimeCode(sessionID, spec.Code), spec.InitialState)
	}
	e.mu.Unlock()
	return nil
}
