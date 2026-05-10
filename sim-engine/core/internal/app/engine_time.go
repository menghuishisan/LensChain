package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lenschain/sim-engine/core/internal/session"
	"github.com/lenschain/sim-engine/core/internal/simcore"
)

// =====================================================================
// 时间控制
// =====================================================================

// ControlTime 执行会话级时间控制指令（详 06.md §7.4）。
//
// 命令集合：play / pause / step / set_speed / reset / resume / step_back。
// 注意：step_back 一般通过独立 WS 消息进入（详 §7.4），但本接口同样兜底支持。
func (e *Engine) ControlTime(sessionID string, command string, paramsJSON []byte) error {
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

	switch command {
	case "play":
		err := runtime.clock.Play()
		if err == nil {
			e.touchAutoAdvance(sessionID)
			e.setRuntimeStatus(sessionID, session.StatusRunning)
		}
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	case "pause":
		err := runtime.clock.Pause()
		if err == nil {
			_, err = e.CreateSnapshot(sessionID, "pause")
		}
		if err == nil {
			e.setRuntimeStatus(sessionID, session.StatusPaused)
		}
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	case "step":
		if err := runtime.clock.Step(); err != nil {
			e.publishControlAck(sessionID, command, err)
			return err
		}
		if err := e.stepScenes(context.Background(), sessionID, runtime.clock.Tick()); err != nil {
			e.publishControlAck(sessionID, command, err)
			return err
		}
		if err := e.recordTickSnapshot(sessionID); err != nil {
			e.publishControlAck(sessionID, command, err)
			return err
		}
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, nil)
		return nil
	case "set_speed":
		speed, err := parseSpeed(paramsJSON)
		if err != nil {
			e.publishControlAck(sessionID, command, err)
			return err
		}
		err = runtime.clock.SetSpeed(speed)
		if err == nil {
			e.touchAutoAdvance(sessionID)
		}
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	case "reset":
		err := e.resetSession(sessionID)
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	case "resume":
		err := runtime.clock.Resume()
		if err == nil {
			e.touchAutoAdvance(sessionID)
			e.setRuntimeStatus(sessionID, session.StatusRunning)
		}
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	case "step_back":
		err := e.stepBackOnce(sessionID)
		e.markRuntimeAction(sessionID, command)
		e.publishControlAck(sessionID, command, err)
		return err
	default:
		err := fmt.Errorf("unsupported time control command: %s", command)
		e.publishControlAck(sessionID, command, err)
		return err
	}
}

// StepBack 是 WS 顶层 step_back 消息的入口（与 ControlTime("step_back", nil) 等价）。
func (e *Engine) StepBack(sessionID string) error {
	return e.ControlTime(sessionID, "step_back", nil)
}

// stepBackOnce 校验上下文后单步回退（详 06.md §5.2 ⏮ 限制范围）。
//
// 仅在以下条件全部满足时允许：
//   - 单场景（activeSceneCodes 仅 1 个）
//   - process 模式
//   - 未启用联动（linkageEnabled=false）
//   - 未启用混合实验采集（collector not running）
func (e *Engine) stepBackOnce(sessionID string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if len(runtime.activeSceneCodes) != 1 {
		return errors.New("step_back_not_allowed_in_multi_scene")
	}
	if runtime.linkageEnabled {
		return errors.New("step_back_not_allowed_in_linkage")
	}
	if e.collectorRunning(sessionID) {
		return errors.New("step_back_not_allowed_in_hybrid")
	}

	if err := runtime.clock.StepBack(); err != nil {
		return err
	}
	targetTick := runtime.clock.Tick()
	if err := e.restoreFromSnapshotStack(sessionID, targetTick); err != nil {
		return err
	}
	return nil
}

// restoreFromSnapshotStack 从 tick 快照栈恢复指定 tick 的状态并重发 render 帧。
func (e *Engine) restoreFromSnapshotStack(sessionID string, targetTick int64) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || runtime.snapshotStack == nil {
		return errors.New("snapshot stack 不可用")
	}
	keyframe, ok := runtime.snapshotStack.NearestKeyframe(targetTick)
	if !ok {
		return errors.New("buffer_exhausted")
	}
	restoredJSON := cloneBytes(keyframe.StateJSON)
	for _, diff := range runtime.snapshotStack.DiffsAfter(keyframe.Tick, targetTick) {
		merged, err := e.stateMgr.MergeDiff(restoredJSON, diff.DiffJSON)
		if err != nil {
			return err
		}
		restoredJSON = merged
	}
	return e.restoreTickState(sessionID, targetTick, restoredJSON)
}

// =====================================================================
// 自动推进
// =====================================================================

// StartClockLoop 按场景节奏自动推进 process 和 continuous 模式会话。
func (e *Engine) StartClockLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.advanceRunnableSessions(ctx)
		}
	}
}

// advanceRunnableSessions 推进所有处于自动播放状态的会话。
func (e *Engine) advanceRunnableSessions(ctx context.Context) {
	for _, sessionID := range e.sessionIDs() {
		unlock, err := e.lockRuntimeOperation(sessionID)
		if err != nil {
			continue
		}
		_ = e.advanceSession(ctx, sessionID)
		unlock()
	}
}

// advanceSession 在达到下一次推进时机时推进单个会话。
func (e *Engine) advanceSession(ctx context.Context, sessionID string) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || runtime.clock == nil || !runtime.clock.IsRunning() {
		return nil
	}
	now := time.Now().UTC()
	stepInterval := e.currentSceneStepDuration(sessionID, runtime.clock.Mode())
	if now.Sub(runtime.lastAutoAdvanceAt) < stepInterval {
		return nil
	}
	if err := runtime.clock.Advance(); err != nil {
		return err
	}
	if err := e.stepScenes(ctx, sessionID, runtime.clock.Tick()); err != nil {
		return err
	}
	if err := e.recordTickSnapshot(sessionID); err != nil {
		return err
	}
	e.touchAutoAdvance(sessionID)
	return nil
}

// currentSceneStepDuration 返回当前会话自动推进一次的间隔。
//
// 默认 1000ms / tick，按 clock.Speed() 缩放。
// 多场景时取所有匹配场景中最小的 step_duration_ms，确保最快节拍的场景不会被拖慢。
func (e *Engine) currentSceneStepDuration(sessionID string, mode simcore.TimeControlMode) time.Duration {
	if mode != simcore.TimeControlModeProcess && mode != simcore.TimeControlModeContinuous {
		return time.Second
	}
	baseMS := int64(1000)
	for _, runtimeRef := range e.scenes.ListBySession(sessionID) {
		if simcore.TimeControlMode(runtimeRef.Meta.TimeControlMode) != mode {
			continue
		}
		var snapshot struct {
			Data struct {
				StepDuration int64 `json:"step_duration_ms"`
			} `json:"data"`
		}
		if err := json.Unmarshal(runtimeRef.State.SceneStateJSON, &snapshot); err == nil && snapshot.Data.StepDuration > 0 {
			if snapshot.Data.StepDuration < baseMS {
				baseMS = snapshot.Data.StepDuration
			}
		}
	}
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || runtime.clock == nil || runtime.clock.Speed() <= 0 {
		return time.Duration(baseMS) * time.Millisecond
	}
	return time.Duration(float64(baseMS)/runtime.clock.Speed()) * time.Millisecond
}

// touchAutoAdvance 刷新会话自动推进节拍起点。
func (e *Engine) touchAutoAdvance(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return
	}
	runtime.lastAutoAdvanceAt = time.Now().UTC()
}

// parseSpeed 从控制参数中解析仿真速度。
func parseSpeed(paramsJSON []byte) (float64, error) {
	var params struct {
		Value float64 `json:"value"`
	}
	if len(paramsJSON) == 0 {
		return 0, errors.New("control value is required")
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return 0, err
	}
	if params.Value == 0 {
		return 0, errors.New("control value is required")
	}
	return params.Value, nil
}
