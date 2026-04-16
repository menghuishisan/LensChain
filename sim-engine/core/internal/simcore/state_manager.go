package simcore

import "encoding/json"

// SceneStateSnapshot 表示某个场景当前的完整状态快照。
type SceneStateSnapshot struct {
	SceneCode       string `json:"scene_code"`
	Tick            int64  `json:"tick"`
	StateJSON       []byte `json:"state_json"`
	RenderStateJSON []byte `json:"render_state_json"`
	SharedStateJSON []byte `json:"shared_state_json"`
}

// StateManager 负责状态摘要、完整快照和增量 diff 的统一构造。
type StateManager struct{}

// NewStateManager 创建状态管理器。
func NewStateManager() *StateManager {
	return &StateManager{}
}

// BuildSceneSummary 构造会话当前场景状态摘要 JSON。
func (m *StateManager) BuildSceneSummary(states []SceneStateSnapshot) []byte {
	payload := struct {
		Scenes []struct {
			SceneCode       string `json:"scene_code"`
			Tick            int64  `json:"tick"`
			RenderStateJSON []byte `json:"render_state_json"`
		} `json:"scenes"`
	}{
		Scenes: make([]struct {
			SceneCode       string `json:"scene_code"`
			Tick            int64  `json:"tick"`
			RenderStateJSON []byte `json:"render_state_json"`
		}, 0, len(states)),
	}

	for _, state := range states {
		payload.Scenes = append(payload.Scenes, struct {
			SceneCode       string `json:"scene_code"`
			Tick            int64  `json:"tick"`
			RenderStateJSON []byte `json:"render_state_json"`
		}{
			SceneCode:       state.SceneCode,
			Tick:            state.Tick,
			RenderStateJSON: cloneSnapshotBytes(state.RenderStateJSON),
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
}

// BuildTickSnapshot 构造某个 tick 的完整状态 JSON。
func (m *StateManager) BuildTickSnapshot(sessionID string, tick int64, states []SceneStateSnapshot) ([]byte, error) {
	payload := struct {
		SessionID string                        `json:"session_id"`
		Tick      int64                         `json:"tick"`
		Scenes    map[string]SceneStateSnapshot `json:"scenes"`
	}{
		SessionID: sessionID,
		Tick:      tick,
		Scenes:    make(map[string]SceneStateSnapshot, len(states)),
	}

	for _, state := range states {
		payload.Scenes[state.SceneCode] = SceneStateSnapshot{
			SceneCode:       state.SceneCode,
			Tick:            state.Tick,
			StateJSON:       cloneSnapshotBytes(state.StateJSON),
			RenderStateJSON: cloneSnapshotBytes(state.RenderStateJSON),
			SharedStateJSON: cloneSnapshotBytes(state.SharedStateJSON),
		}
	}

	return json.Marshal(payload)
}

// BuildDiff 计算两个完整状态 JSON 之间的增量 diff。
func (m *StateManager) BuildDiff(previousJSON []byte, currentJSON []byte) ([]byte, error) {
	if len(previousJSON) == 0 {
		return cloneSnapshotBytes(currentJSON), nil
	}

	var previous map[string]any
	var current map[string]any
	if err := json.Unmarshal(previousJSON, &previous); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(currentJSON, &current); err != nil {
		return nil, err
	}
	diff := diffJSONObjects(previous, current)
	return json.Marshal(diff)
}

// MergeDiff 将增量 diff 合并回完整状态 JSON。
func (m *StateManager) MergeDiff(baseJSON []byte, diffJSON []byte) ([]byte, error) {
	if len(baseJSON) == 0 {
		return cloneSnapshotBytes(diffJSON), nil
	}
	var base map[string]any
	var diff map[string]any
	if err := json.Unmarshal(baseJSON, &base); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(diffJSON, &diff); err != nil {
		return nil, err
	}
	mergeSnapshotMap(base, diff)
	return json.Marshal(base)
}

// diffJSONObjects 递归比较两个 JSON 对象并生成差异对象。
func diffJSONObjects(previous map[string]any, current map[string]any) map[string]any {
	result := make(map[string]any)
	for key, currentValue := range current {
		previousValue, exists := previous[key]
		if !exists {
			result[key] = currentValue
			continue
		}
		previousMap, previousIsMap := previousValue.(map[string]any)
		currentMap, currentIsMap := currentValue.(map[string]any)
		if previousIsMap && currentIsMap {
			child := diffJSONObjects(previousMap, currentMap)
			if len(child) > 0 {
				result[key] = child
			}
			continue
		}
		if !jsonValueEqual(previousValue, currentValue) {
			result[key] = currentValue
		}
	}
	for key := range previous {
		if _, exists := current[key]; !exists {
			result[key] = nil
		}
	}
	return result
}

// mergeSnapshotMap 对快照 map 做递归合并。
func mergeSnapshotMap(base map[string]any, diff map[string]any) {
	for key, diffValue := range diff {
		if diffValue == nil {
			delete(base, key)
			continue
		}
		baseMap, baseIsMap := base[key].(map[string]any)
		diffMap, diffIsMap := diffValue.(map[string]any)
		if baseIsMap && diffIsMap {
			mergeSnapshotMap(baseMap, diffMap)
			continue
		}
		base[key] = diffValue
	}
}

// jsonValueEqual 通过 JSON 编码比较两个值是否语义相等。
func jsonValueEqual(left any, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}
