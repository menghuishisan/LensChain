package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/lenschain/sim-engine/core/internal/simcore"
	"github.com/lenschain/sim-engine/core/internal/ws"
)

// =====================================================================
// 状态变更工具
// =====================================================================

// setRuntimeStatus 更新会话运行状态。
func (e *Engine) setRuntimeStatus(sessionID string, status string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return
	}
	runtime.status = status
	runtime.updatedAt = time.Now().UTC()
}

// markRuntimeAction 更新会话最近一次动作和更新时间。
func (e *Engine) markRuntimeAction(sessionID string, action string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return
	}
	runtime.lastAction = action
	runtime.updatedAt = time.Now().UTC()
}

// markSceneReady 更新场景就绪状态和错误信息。
func (e *Engine) markSceneReady(sessionID string, sceneCode string, ready bool, errMessage string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return
	}
	if runtime.sceneReady == nil {
		runtime.sceneReady = make(map[string]bool)
	}
	if runtime.sceneErrors == nil {
		runtime.sceneErrors = make(map[string]string)
	}
	runtime.sceneReady[sceneCode] = ready
	if ready {
		delete(runtime.sceneErrors, sceneCode)
		if !containsSceneCode(runtime.activeSceneCodes, sceneCode) {
			runtime.activeSceneCodes = append(runtime.activeSceneCodes, sceneCode)
		}
	} else if errMessage != "" {
		runtime.sceneErrors[sceneCode] = errMessage
	}
	runtime.updatedAt = time.Now().UTC()
}

// =====================================================================
// WS 推送
// =====================================================================

// publishRender 推送 RenderEnvelope（render 类型）。
func (e *Engine) publishRender(sessionID string, sceneCode string, tick int64, envelopeJSON []byte) {
	if len(envelopeJSON) == 0 {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeRender,
		SceneCode:   sceneCode,
		Tick:        tick,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: cloneBytes(envelopeJSON),
	})
}

// publishEvent 推送通用事件（event 类型）。
func (e *Engine) publishEvent(sessionID string, eventName string, data map[string]any) {
	payload, err := json.Marshal(ws.EventPayload{
		Event: eventName,
		Data:  data,
	})
	if err != nil {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeEvent,
		Tick:        e.currentTick(sessionID),
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: payload,
	})
}

// publishCollectorInterrupted 在采集失败时向前端发送中断事件。
func (e *Engine) publishCollectorInterrupted(sessionID string, cause error) {
	if cause == nil {
		return
	}
	e.publishEvent(sessionID, "collector_interrupted", map[string]any{
		"error_message": cause.Error(),
	})
}

// publishControlAck 向前端广播时间控制命令的执行结果。
func (e *Engine) publishControlAck(sessionID string, command string, err error) {
	ack := ws.ControlAckPayload{Command: command, Success: err == nil}
	if err != nil {
		ack.Error = err.Error()
	}
	data, marshalErr := json.Marshal(ack)
	if marshalErr != nil {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeControlAck,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: data,
	})
}

// =====================================================================
// 辅助函数
// =====================================================================

// currentTick 返回指定会话当前仿真 tick。
func (e *Engine) currentTick(sessionID string) int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return 0
	}
	return runtime.clock.Tick()
}

// sessionIDs 返回当前已注册的全部会话 ID。
func (e *Engine) sessionIDs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]string, 0, len(e.runtimes))
	for sessionID := range e.runtimes {
		result = append(result, sessionID)
	}
	return result
}

// resolveSessionClockMode 根据会话内场景组合推导时钟模式。
func resolveSessionClockMode(modes []simcore.TimeControlMode) simcore.TimeControlMode {
	hasContinuous := false
	for _, mode := range modes {
		if mode == simcore.TimeControlModeProcess {
			return simcore.TimeControlModeProcess
		}
		if mode == simcore.TimeControlModeContinuous {
			hasContinuous = true
		}
	}
	if hasContinuous {
		return simcore.TimeControlModeContinuous
	}
	return simcore.TimeControlModeReactive
}

// containsSceneCode 判断场景列表中是否已包含目标场景。
func containsSceneCode(sceneCodes []string, target string) bool {
	for _, sceneCode := range sceneCodes {
		if sceneCode == target {
			return true
		}
	}
	return false
}

// buildSimTimeSeconds 根据 tick 和速度计算仿真时间摘要。
func buildSimTimeSeconds(tick int64, speed float64) float64 {
	if speed <= 0 {
		return float64(tick)
	}
	return float64(tick) / speed
}

// newSnapshotID 生成快照 ID。
func newSnapshotID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "snap-" + hex.EncodeToString(raw[:]), nil
}

// cloneBytes 复制字节切片，避免共享底层数组。
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

// encodeMapJSON 序列化 map 为 JSON 字节；nil 或空 map 返回 nil。
func encodeMapJSON(value map[string]any) []byte {
	if len(value) == 0 {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}

// mergeSceneParams 将运行时选中的联动组编码注入场景参数。
func mergeSceneParams(paramsJSON []byte, linkGroupCode string) []byte {
	params := make(map[string]any)
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return cloneBytes(paramsJSON)
		}
	}
	if strings.TrimSpace(linkGroupCode) != "" {
		params["link_group_code"] = linkGroupCode
	}
	if len(params) == 0 {
		return nil
	}
	merged, err := json.Marshal(params)
	if err != nil {
		return cloneBytes(paramsJSON)
	}
	return merged
}

// sharedStateJSONForGroup 在 LinkGroups 列表中寻找 groupCode 对应的初始 SharedState JSON。
func sharedStateJSONForGroup(groups []LinkGroupSpec, groupCode string) []byte {
	if groupCode == "" {
		return nil
	}
	for _, spec := range groups {
		if spec.Code != groupCode {
			continue
		}
		return encodeMapJSON(spec.InitialState)
	}
	return nil
}

// flattenJSONPaths 将联动 diff JSON 展平成路径列表。
func flattenJSONPaths(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	paths := make([]string, 0)
	flattenPathMap(payload, "", &paths)
	return paths
}

// flattenPathMap 递归收集 JSON 对象中的字段路径。
func flattenPathMap(payload map[string]any, prefix string, paths *[]string) {
	for key, value := range payload {
		nextPath := key
		if prefix != "" {
			nextPath = prefix + "." + key
		}
		*paths = append(*paths, nextPath)
		child, ok := value.(map[string]any)
		if ok {
			flattenPathMap(child, nextPath, paths)
		}
	}
}
