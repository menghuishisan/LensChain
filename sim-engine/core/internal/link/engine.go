// 模块：sim-engine/core/internal/link
// 文件职责：LinkEngine — 联动组共享状态空间的 owner-based 写权限校验与跨场景事件分发。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §8.3 / §8.4 / §8.5 / §8.7。
//
// 关键约束：
// 1. 每个 SharedState 字段必须在 LinkGroup 的 schema 中显式声明唯一 owner 场景；
//    非 owner 写入一律拒绝（OwnerViolationError），并记录审计。
// 2. 同一 LinkGroup 的 schema 版本一旦发布即不可变（schema 不可演进，§8.7）；
//    重复注册同 code+version 但 schema 不同将被拒绝。
// 3. fan-out：源场景写入后，向同组其他成员构造 LinkTrigger 事件。
// 4. 不在本层做 Step 推进；只做状态合并与事件构造。

package link

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// FieldSchema 是 LinkGroup shared_state_schema 中一个字段的元描述。
type FieldSchema struct {
	// Name 字段名。
	Name string
	// Type JSON 类型描述（"int" / "string" / "bool" / "string[]" / "object" 等，仅记录用）。
	Type string
	// Owner 唯一可写场景 code；该字段值只能由该场景的 SharedStateDiff 写入。
	Owner string
}

// Group 表示一个联动组的不可变定义。
type Group struct {
	Code    string
	Version string
	// Members 联动组成员场景 code 列表。
	Members []string
	// Fields shared_state_schema：字段名 → 字段元信息。
	Fields map[string]FieldSchema
	// ForceClockSync 是否强制成员时钟同步（详 §9.3）。
	ForceClockSync bool
}

// LinkTrigger 是 fan-out 给其他成员的联动事件（与 framework / sdk 的 LinkTrigger 对齐）。
type LinkTrigger struct {
	ID             string         `json:"id"`
	SourceScene    string         `json:"source_scene"`
	SourceAction   string         `json:"source_action,omitempty"`
	LinkGroup      string         `json:"link_group"`
	ChangedFields  []string       `json:"changed_fields"`
	Payload        map[string]any `json:"payload"`
	Timestamp      int64          `json:"ts"`
	SourceAnchorID string         `json:"source_anchor_id,omitempty"`
	TargetAnchorID string         `json:"target_anchor_id,omitempty"`
}

// FanOut 是一次 ApplyDiff 的结果：合并后的最新共享状态 + 给每个接收方的 LinkTrigger。
type FanOut struct {
	GroupCode    string
	GroupVersion string
	// SourceScene 触发本次 fan-out 的源场景。
	SourceScene string
	// State 当前合并后的完整共享状态（深拷贝，调用方可自由读写）。
	State map[string]any
	// Triggers 构造给每个接收方的 LinkTrigger 数组（key=接收方 scene_code）。
	Triggers map[string]LinkTrigger
}

// OwnerViolationError 表示非 owner 场景试图写入字段。
type OwnerViolationError struct {
	GroupCode string
	Field     string
	Writer    string
	Owner     string
}

// Error 实现 error 接口。
func (e OwnerViolationError) Error() string {
	return fmt.Sprintf("LinkGroup %s 字段 %s 只能由 owner 场景 %s 写入，被 %s 拒绝",
		e.GroupCode, e.Field, e.Owner, e.Writer)
}

// SchemaConflictError 表示同 code+version 但 schema 不同（违反 §8.7 不可演进）。
type SchemaConflictError struct {
	GroupCode string
	Version   string
	Detail    string
}

// Error 实现 error 接口。
func (e SchemaConflictError) Error() string {
	return fmt.Sprintf("LinkGroup %s@%s schema 冲突：%s（schema 一旦发布不可演进，详 §8.7）",
		e.GroupCode, e.Version, e.Detail)
}

// =====================================================================
// Engine
// =====================================================================

// Engine 管理多个联动组的共享状态空间与事件分发。
type Engine struct {
	mu     sync.RWMutex
	groups map[string]Group          // key: groupCode
	states map[string]map[string]any // key: groupCode → 当前共享状态
}

// NewEngine 创建联动引擎。
func NewEngine() *Engine {
	return &Engine{
		groups: make(map[string]Group),
		states: make(map[string]map[string]any),
	}
}

// RegisterGroup 注册联动组及其初始共享状态。
//
// 规则：
//  1. 同 code 不同 version：允许，并行存在（旧实验锁定旧版本）；
//     当前实现以 code 作为 key（同一 Engine 实例只承载某个版本），
//     上层（Session）按 link_group_id+version 选择 Engine 命名空间。
//  2. 同 code 同 version 重复注册：要求 schema 完全一致，否则返回 SchemaConflictError。
func (e *Engine) RegisterGroup(group Group, initial map[string]any) error {
	if group.Code == "" {
		return errors.New("LinkGroup.Code 不能为空")
	}
	if group.Version == "" {
		return errors.New("LinkGroup.Version 不能为空")
	}
	for fieldName, fs := range group.Fields {
		if fs.Owner == "" {
			return fmt.Errorf("LinkGroup %s 字段 %s 缺少 owner 声明", group.Code, fieldName)
		}
		if !containsString(group.Members, fs.Owner) {
			return fmt.Errorf("LinkGroup %s 字段 %s owner %s 不在成员列表中", group.Code, fieldName, fs.Owner)
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if existing, ok := e.groups[group.Code]; ok {
		if existing.Version != group.Version {
			return fmt.Errorf("LinkGroup %s 已注册版本 %s，与新版本 %s 冲突；请按 §8.7 走新版本路径",
				group.Code, existing.Version, group.Version)
		}
		if !schemaEqual(existing, group) {
			return SchemaConflictError{
				GroupCode: group.Code,
				Version:   group.Version,
				Detail:    "已存在的 schema 与新注册的 schema 不一致",
			}
		}
		return nil
	}

	e.groups[group.Code] = cloneGroup(group)
	e.states[group.Code] = cloneMap(initial)
	return nil
}

// SharedState 返回联动组当前共享状态的深拷贝。
func (e *Engine) SharedState(groupCode string) (map[string]any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state, ok := e.states[groupCode]
	if !ok {
		return nil, false
	}
	return cloneMap(state), true
}

// ApplyDiff 校验 owner 后将 diff 合并入共享状态，并构造 fan-out。
//
// 规则：
//  1. diff 中每个 top-level 字段必须出现在 group.Fields 中，且 owner == sourceScene；
//     否则返回 OwnerViolationError（不应用 diff，整体拒绝）。
//  2. 合并后构造 LinkTrigger 给除 sourceScene 之外的成员。
//  3. payload 携带本次写入的字段子集（payload[field] = newValue）。
func (e *Engine) ApplyDiff(groupCode string, sourceScene string, sourceAction string,
	diff map[string]any, sourceAnchorID string, anchorMapping map[string]string) (FanOut, error) {

	e.mu.Lock()
	defer e.mu.Unlock()

	group, ok := e.groups[groupCode]
	if !ok {
		return FanOut{}, fmt.Errorf("LinkGroup %s 未注册", groupCode)
	}

	// 1) owner 校验
	changed := make([]string, 0, len(diff))
	for field := range diff {
		schema, declared := group.Fields[field]
		if !declared {
			return FanOut{}, fmt.Errorf("LinkGroup %s 不包含字段 %s（schema 未声明）", groupCode, field)
		}
		if schema.Owner != sourceScene {
			return FanOut{}, OwnerViolationError{
				GroupCode: groupCode,
				Field:     field,
				Writer:    sourceScene,
				Owner:     schema.Owner,
			}
		}
		changed = append(changed, field)
	}
	sort.Strings(changed)

	// 2) 合并到当前状态
	current := e.states[groupCode]
	if current == nil {
		current = make(map[string]any)
	}
	for _, field := range changed {
		current[field] = diff[field]
	}
	e.states[groupCode] = current

	// 3) 构造 fan-out triggers
	timestamp := time.Now().UTC().UnixMilli()
	triggers := make(map[string]LinkTrigger, len(group.Members))
	payload := make(map[string]any, len(changed))
	for _, field := range changed {
		payload[field] = diff[field]
	}
	for _, member := range group.Members {
		if member == sourceScene {
			continue
		}
		targetAnchor := ""
		if anchorMapping != nil {
			targetAnchor = anchorMapping[member]
		}
		triggers[member] = LinkTrigger{
			ID:             fmt.Sprintf("%s-%s-%d", groupCode, sourceScene, timestamp),
			SourceScene:    sourceScene,
			SourceAction:   sourceAction,
			LinkGroup:      groupCode,
			ChangedFields:  append([]string(nil), changed...),
			Payload:        cloneMap(payload),
			Timestamp:      timestamp,
			SourceAnchorID: sourceAnchorID,
			TargetAnchorID: targetAnchor,
		}
	}

	return FanOut{
		GroupCode:    groupCode,
		GroupVersion: group.Version,
		SourceScene:  sourceScene,
		State:        cloneMap(current),
		Triggers:     triggers,
	}, nil
}

// ApplyDiffJSON 是 ApplyDiff 的 JSON 字节包装，便于 gRPC 使用。
func (e *Engine) ApplyDiffJSON(groupCode string, sourceScene string, sourceAction string,
	diffJSON []byte, sourceAnchorID string, anchorMapping map[string]string) (FanOut, error) {

	if len(diffJSON) == 0 {
		return FanOut{}, errors.New("diffJSON 为空")
	}
	var diff map[string]any
	if err := json.Unmarshal(diffJSON, &diff); err != nil {
		return FanOut{}, fmt.Errorf("解码 diff 失败: %w", err)
	}
	return e.ApplyDiff(groupCode, sourceScene, sourceAction, diff, sourceAnchorID, anchorMapping)
}

// ResetGroup 将联动组共享状态重置为指定初始值；用于实验 reset 控制。
func (e *Engine) ResetGroup(groupCode string, initial map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.groups[groupCode]; !ok {
		return fmt.Errorf("LinkGroup %s 未注册", groupCode)
	}
	e.states[groupCode] = cloneMap(initial)
	return nil
}

// SetForceClockSync 修改联动组的强制时钟同步标记。
//
// 仅供教师干预 unlock_link_sync 使用（详 06.md §14.5）。
func (e *Engine) SetForceClockSync(groupCode string, enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.groups[groupCode]
	if !ok {
		return
	}
	g.ForceClockSync = enabled
	e.groups[groupCode] = g
}

// DeleteGroup 删除指定联动组及其共享状态（会话结束时调用）。
func (e *Engine) DeleteGroup(groupCode string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.groups, groupCode)
	delete(e.states, groupCode)
}

// GroupInfo 返回某个联动组的元信息（不含状态）。
func (e *Engine) GroupInfo(groupCode string) (Group, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	g, ok := e.groups[groupCode]
	if !ok {
		return Group{}, false
	}
	return cloneGroup(g), true
}

// =====================================================================
// 内部工具
// =====================================================================

func cloneGroup(g Group) Group {
	fields := make(map[string]FieldSchema, len(g.Fields))
	for k, v := range g.Fields {
		fields[k] = v
	}
	return Group{
		Code:           g.Code,
		Version:        g.Version,
		Members:        append([]string(nil), g.Members...),
		Fields:         fields,
		ForceClockSync: g.ForceClockSync,
	}
}

func schemaEqual(a Group, b Group) bool {
	if len(a.Members) != len(b.Members) || len(a.Fields) != len(b.Fields) {
		return false
	}
	am := append([]string(nil), a.Members...)
	bm := append([]string(nil), b.Members...)
	sort.Strings(am)
	sort.Strings(bm)
	for i := range am {
		if am[i] != bm[i] {
			return false
		}
	}
	for name, fs := range a.Fields {
		bfs, ok := b.Fields[name]
		if !ok || bfs.Owner != fs.Owner || bfs.Type != fs.Type {
			return false
		}
	}
	return a.ForceClockSync == b.ForceClockSync
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var copied map[string]any
	if err := json.Unmarshal(data, &copied); err != nil {
		return map[string]any{}
	}
	return copied
}
