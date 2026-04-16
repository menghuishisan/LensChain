package link

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"
)

// Group 表示一组共享底层状态空间的仿真场景。
type Group struct {
	Code       string
	SceneCodes []string
	StateJSON  []byte
}

// Update 表示联动状态传播结果。
type Update struct {
	SourceScene    string
	AffectedScenes []string
	StateJSON      []byte
}

// Engine 管理联动组共享状态空间。
type Engine struct {
	mu     sync.RWMutex
	groups map[string]Group
	states map[string][]byte
}

// NewEngine 创建联动引擎。
func NewEngine() *Engine {
	return &Engine{
		groups: make(map[string]Group),
		states: make(map[string][]byte),
	}
}

// RegisterGroup 注册联动组及其初始共享状态。
func (e *Engine) RegisterGroup(group Group) error {
	if group.Code == "" {
		return errors.New("link group code is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.groups[group.Code] = group
	e.states[group.Code] = cloneBytes(group.StateJSON)
	return nil
}

// ApplyDiff 合并共享状态 diff，并返回受影响场景。
func (e *Engine) ApplyDiff(groupCode string, sourceScene string, diffJSON []byte) (Update, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	group, ok := e.groups[groupCode]
	if !ok {
		return Update{}, errors.New("link group not found")
	}

	merged, err := mergeJSONObjects(e.states[groupCode], diffJSON)
	if err != nil {
		return Update{}, err
	}
	e.states[groupCode] = merged

	affected := make([]string, 0, len(group.SceneCodes))
	for _, sceneCode := range group.SceneCodes {
		if sceneCode != sourceScene {
			affected = append(affected, sceneCode)
		}
	}
	sort.Strings(affected)

	return Update{
		SourceScene:    sourceScene,
		AffectedScenes: affected,
		StateJSON:      cloneBytes(merged),
	}, nil
}

// SharedState 返回联动组当前共享状态。
func (e *Engine) SharedState(groupCode string) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state, ok := e.states[groupCode]
	return cloneBytes(state), ok
}

// ReplaceState 用指定状态覆盖联动组当前共享状态。
func (e *Engine) ReplaceState(groupCode string, stateJSON []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.groups[groupCode]; !ok {
		return errors.New("link group not found")
	}
	e.states[groupCode] = cloneBytes(stateJSON)
	return nil
}

// DeleteGroup 删除指定联动组及其共享状态。
func (e *Engine) DeleteGroup(groupCode string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.groups, groupCode)
	delete(e.states, groupCode)
}

// mergeJSONObjects 将 diff JSON 深度合并到共享状态 JSON。
func mergeJSONObjects(baseJSON []byte, diffJSON []byte) ([]byte, error) {
	var base map[string]any
	var diff map[string]any

	if len(baseJSON) == 0 {
		base = make(map[string]any)
	} else if err := json.Unmarshal(baseJSON, &base); err != nil {
		return nil, err
	}

	if len(diffJSON) == 0 {
		return json.Marshal(base)
	}
	if err := json.Unmarshal(diffJSON, &diff); err != nil {
		return nil, err
	}

	deepMerge(base, diff)
	return json.Marshal(base)
}

// deepMerge 对 map 进行递归深度合并。
func deepMerge(base map[string]any, diff map[string]any) {
	for key, diffValue := range diff {
		baseChild, baseIsMap := base[key].(map[string]any)
		diffChild, diffIsMap := diffValue.(map[string]any)
		if baseIsMap && diffIsMap {
			deepMerge(baseChild, diffChild)
			continue
		}
		base[key] = diffValue
	}
}

// cloneBytes 复制共享状态字节切片。
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
