// Package scene 管理场景算法运行时与容器调用适配。
package scene

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// Config 是启动单个场景运行时所需的配置。
type Config struct {
	SessionID        string
	SceneCode        string
	ParamsJSON       []byte
	InitialStateJSON []byte
	SharedStateJSON  []byte
}

// Meta 是场景元信息，Core 用它匹配渲染器和时间模式。
type Meta struct {
	Code                    string
	Name                    string
	Category                string
	AlgorithmType           string
	Description             string
	Version                 string
	TimeControlMode         string
	DataSourceMode          string
	DefaultParamsJSON       []byte
	DefaultStateJSON        []byte
	SupportedLinkGroupCodes []string
}

// RenderState 表示场景容器按需返回的可渲染状态。
type RenderState struct {
	SceneCode     string
	Category      string
	AlgorithmType string
	Tick          int64
	StateJSON     []byte
	RenderStateJSON []byte
	MetricsJSON   []byte
	Events        []Event
}

// HealthStatus 表示场景容器健康状态。
type HealthStatus struct {
	Status      string
	Message     string
	CheckedAtMS int64
}

// State 是场景完整状态和前端可渲染状态。
type State struct {
	SceneCode       string
	Tick            int64
	StateJSON       []byte
	RenderStateJSON []byte
	SharedStateJSON []byte
}

// InitRequest 是场景初始化请求。
type InitRequest struct {
	SessionID        string
	SceneCode        string
	ParamsJSON       []byte
	InitialStateJSON []byte
	SharedStateJSON  []byte
}

// StepRequest 是场景 tick 推进请求。
type StepRequest struct {
	SessionID       string
	SceneCode       string
	Tick            int64
	StateJSON       []byte
	SharedStateJSON []byte
}

// StepResult 是场景 tick 推进结果。
type StepResult struct {
	SceneCode           string
	Tick                int64
	StateJSON           []byte
	RenderStateJSON     []byte
	Events              []Event
	SharedStateDiffJSON []byte
}

// ActionRequest 是场景交互请求。
type ActionRequest struct {
	SessionID       string
	SceneCode       string
	StateJSON       []byte
	ActionCode      string
	ParamsJSON      []byte
	SharedStateJSON []byte
	Tick            int64
	ActorID         string
	RoleKey         string
}

// ActionResult 是场景交互结果。
type ActionResult struct {
	Tick                int64
	Success             bool
	ErrorMessage        string
	StateJSON           []byte
	RenderStateJSON     []byte
	Events              []Event
	SharedStateDiffJSON []byte
}

// Event 表示场景算法产生的过程事件。
type Event struct {
	EventID     string
	EventType   string
	SceneCode   string
	Tick        int64
	TimestampMS int64
	PayloadJSON []byte
}

// InteractionFieldOption 表示交互字段选项。
type InteractionFieldOption struct {
	Value string
	Label string
}

// InteractionField 表示场景交互面板的输入字段定义。
type InteractionField struct {
	Key            string
	Label          string
	Type           string
	Required       bool
	DefaultValue   string
	Options        []InteractionFieldOption
	ValidationJSON []byte
}

// InteractionAction 表示一个场景可执行交互动作。
type InteractionAction struct {
	ActionCode   string
	Label        string
	Description  string
	Trigger      string
	Fields       []InteractionField
	UISchemaJSON []byte
}

// InteractionSchema 是场景专属交互面板定义。
type InteractionSchema struct {
	SceneCode string
	Actions   []InteractionAction
}

// ActionContext 表示一次交互请求附带的操作者身份信息。
type ActionContext struct {
	ActorID string
	RoleKey string
}

// ScenarioClient 是 Core 调用场景算法容器的最小接口。
type ScenarioClient interface {
	Meta(ctx context.Context) (Meta, error)
	Init(ctx context.Context, req InitRequest) (State, error)
	Step(ctx context.Context, req StepRequest) (StepResult, error)
	HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error)
	InteractionSchema(ctx context.Context) (InteractionSchema, error)
	RenderState(ctx context.Context, sessionID string, sceneCode string, stateJSON []byte, tick int64, sharedStateJSON []byte) (RenderState, error)
	Health(ctx context.Context) (HealthStatus, error)
	Close() error
}

// ClientFactory 根据场景配置创建算法容器客户端。
type ClientFactory func(config Config) (ScenarioClient, error)

// Runtime 表示一个已启动的场景运行时。
type Runtime struct {
	Config Config
	Meta   Meta
	State  State
	client ScenarioClient
}

// Manager 管理场景运行时生命周期。
type Manager struct {
	mu       sync.RWMutex
	factory  ClientFactory
	runtimes map[string]Runtime
}

// NewManager 创建场景管理器。
func NewManager(factory ClientFactory) *Manager {
	return &Manager{
		factory:  factory,
		runtimes: make(map[string]Runtime),
	}
}

// Start 启动场景并保存运行时。
func (m *Manager) Start(ctx context.Context, config Config) (Runtime, error) {
	if config.SceneCode == "" {
		return Runtime{}, errors.New("scene code is required")
	}
	if m.factory == nil {
		return Runtime{}, errors.New("scene client factory is required")
	}

	client, err := m.factory(config)
	if err != nil {
		return Runtime{}, err
	}
	meta, err := client.Meta(ctx)
	if err != nil {
		return Runtime{}, err
	}
	state, err := client.Init(ctx, InitRequest{
		SessionID:        config.SessionID,
		SceneCode:        config.SceneCode,
		ParamsJSON:       cloneBytes(config.ParamsJSON),
		InitialStateJSON: cloneBytes(config.InitialStateJSON),
		SharedStateJSON:  cloneBytes(config.SharedStateJSON),
	})
	if err != nil {
		return Runtime{}, err
	}

	runtime := Runtime{
		Config: config,
		Meta:   meta,
		State:  state,
		client: client,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimes[runtimeKey(config.SessionID, config.SceneCode)] = runtime
	return runtime, nil
}

// Get 获取场景运行时。
func (m *Manager) Get(sessionID string, sceneCode string) (Runtime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	return runtime, ok
}

// Step 推进指定场景一个 tick。
func (m *Manager) Step(ctx context.Context, sessionID string, sceneCode string, tick int64, sharedStateJSON []byte) (StepResult, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return StepResult{}, errors.New("scene runtime not found")
	}

	result, err := runtime.client.Step(ctx, StepRequest{
		SessionID:       runtime.Config.SessionID,
		SceneCode:       sceneCode,
		Tick:            tick,
		StateJSON:       cloneBytes(runtime.State.StateJSON),
		SharedStateJSON: cloneBytes(sharedStateJSON),
	})
	if err != nil {
		return StepResult{}, err
	}

	runtime.State.Tick = result.Tick
	runtime.State.SharedStateJSON = cloneBytes(sharedStateJSON)
	if result.SceneCode != "" {
		runtime.State.SceneCode = result.SceneCode
	}
	if result.StateJSON != nil {
		runtime.State.StateJSON = cloneBytes(result.StateJSON)
	}
	if result.RenderStateJSON != nil {
		runtime.State.RenderStateJSON = cloneBytes(result.RenderStateJSON)
	}

	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return result, nil
}

// HandleAction 将场景专属交互转发给对应场景运行时。
func (m *Manager) HandleAction(
	ctx context.Context,
	sessionID string,
	sceneCode string,
	actionCode string,
	paramsJSON []byte,
	sharedStateJSON []byte,
	actionCtx ActionContext,
) (ActionResult, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return ActionResult{}, errors.New("scene runtime not found")
	}

	result, err := runtime.client.HandleAction(ctx, ActionRequest{
		SessionID:       runtime.Config.SessionID,
		SceneCode:       sceneCode,
		StateJSON:       cloneBytes(runtime.State.StateJSON),
		ActionCode:      actionCode,
		ParamsJSON:      cloneBytes(paramsJSON),
		SharedStateJSON: cloneBytes(sharedStateJSON),
		Tick:            runtime.State.Tick,
		ActorID:         actionCtx.ActorID,
		RoleKey:         actionCtx.RoleKey,
	})
	if err != nil {
		return ActionResult{}, err
	}

	if result.StateJSON != nil || result.RenderStateJSON != nil {
		if result.StateJSON != nil {
			runtime.State.StateJSON = cloneBytes(result.StateJSON)
		}
		if result.RenderStateJSON != nil {
			runtime.State.RenderStateJSON = cloneBytes(result.RenderStateJSON)
		}
		runtime.State.Tick = result.Tick
		runtime.State.SharedStateJSON = cloneBytes(sharedStateJSON)
		m.mu.Lock()
		m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
		m.mu.Unlock()
	}
	return result, nil
}

// InteractionSchema 返回指定场景的交互面板定义。
func (m *Manager) InteractionSchema(ctx context.Context, sessionID string, sceneCode string) (InteractionSchema, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return InteractionSchema{}, errors.New("scene runtime not found")
	}
	return runtime.client.InteractionSchema(ctx)
}

// RenderState 按需查询场景当前可渲染状态。
func (m *Manager) RenderState(ctx context.Context, sessionID string, sceneCode string) (RenderState, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return RenderState{}, errors.New("scene runtime not found")
	}
	return runtime.client.RenderState(ctx, sessionID, sceneCode, cloneBytes(runtime.State.StateJSON), runtime.State.Tick, cloneBytes(runtime.State.SharedStateJSON))
}

// RefreshRenderState 基于最新共享状态重新计算场景状态与渲染态。
func (m *Manager) RefreshRenderState(ctx context.Context, sessionID string, sceneCode string, sharedStateJSON []byte) (State, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return State{}, errors.New("scene runtime not found")
	}

	renderState, err := runtime.client.RenderState(
		ctx,
		sessionID,
		sceneCode,
		cloneBytes(runtime.State.StateJSON),
		runtime.State.Tick,
		cloneBytes(sharedStateJSON),
	)
	if err != nil {
		return State{}, err
	}

	runtime.State.Tick = renderState.Tick
	runtime.State.StateJSON = cloneBytes(renderState.StateJSON)
	runtime.State.RenderStateJSON = cloneBytes(renderState.RenderStateJSON)
	runtime.State.SharedStateJSON = cloneBytes(sharedStateJSON)

	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return runtime.State, nil
}

// HealthCheck 查询场景容器健康状态。
func (m *Manager) HealthCheck(ctx context.Context, sessionID string, sceneCode string) (HealthStatus, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return HealthStatus{}, errors.New("scene runtime not found")
	}
	return runtime.client.Health(ctx)
}

// Restart 重新创建场景客户端，并尽量用最近状态恢复该场景。
func (m *Manager) Restart(ctx context.Context, sessionID string, sceneCode string) (Runtime, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return Runtime{}, errors.New("scene runtime not found")
	}
	if m.factory == nil {
		return Runtime{}, errors.New("scene client factory is required")
	}

	client, err := m.factory(runtime.Config)
	if err != nil {
		return Runtime{}, err
	}
	meta, err := client.Meta(ctx)
	if err != nil {
		return Runtime{}, err
	}
	state, err := client.Init(ctx, InitRequest{
		SessionID:        runtime.Config.SessionID,
		SceneCode:        runtime.Config.SceneCode,
		ParamsJSON:       cloneBytes(runtime.Config.ParamsJSON),
		InitialStateJSON: cloneBytes(runtime.State.StateJSON),
		SharedStateJSON:  cloneBytes(runtime.State.SharedStateJSON),
	})
	if err != nil {
		return Runtime{}, err
	}

	runtime.client = client
	runtime.Meta = meta
	if len(state.StateJSON) > 0 || len(state.RenderStateJSON) > 0 {
		runtime.State = state
	}

	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return runtime, nil
}

// ListBySession 返回会话内所有场景运行时快照。
func (m *Manager) ListBySession(sessionID string) []Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Runtime, 0)
	prefix := sessionID + "::"
	for key, runtime := range m.runtimes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, runtime)
		}
	}
	return result
}

// RestoreState 用快照中的场景状态覆盖当前运行时状态。
func (m *Manager) RestoreState(sessionID string, sceneCode string, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := runtimeKey(sessionID, sceneCode)
	runtime, ok := m.runtimes[key]
	if !ok {
		return
	}
	runtime.State = state
	m.runtimes[key] = runtime
}

// DestroySession 销毁指定会话下的全部场景运行时，并关闭关联客户端连接。
func (m *Manager) DestroySession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var closeErr error
	prefix := sessionID + "::"
	for key, runtime := range m.runtimes {
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		if runtime.client != nil {
			if err := runtime.client.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		delete(m.runtimes, key)
	}
	return closeErr
}

// UpdateSharedState 用当前共享状态覆盖场景运行时，确保容器重启时按最新联动状态恢复。
func (m *Manager) UpdateSharedState(sessionID string, sceneCode string, sharedStateJSON []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := runtimeKey(sessionID, sceneCode)
	runtime, ok := m.runtimes[key]
	if !ok {
		return
	}
	runtime.State.SharedStateJSON = cloneBytes(sharedStateJSON)
	m.runtimes[key] = runtime
}

// InjectCollectionPatch 将采集事件补丁合并进指定场景的状态与渲染状态。
func (m *Manager) InjectCollectionPatch(sessionID string, sceneCode string, patchJSON []byte) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := runtimeKey(sessionID, sceneCode)
	runtime, ok := m.runtimes[key]
	if !ok {
		return State{}, errors.New("scene runtime not found")
	}

	stateJSON, err := mergeJSONObjects(runtime.State.StateJSON, patchJSON)
	if err != nil {
		return State{}, err
	}
	renderJSON, err := mergeJSONObjects(runtime.State.RenderStateJSON, patchJSON)
	if err != nil {
		return State{}, err
	}

	runtime.State.StateJSON = stateJSON
	runtime.State.RenderStateJSON = renderJSON
	m.runtimes[key] = runtime
	return runtime.State, nil
}

// runtimeKey 生成会话内唯一的场景运行时键。
func runtimeKey(sessionID string, sceneCode string) string {
	return sessionID + "::" + sceneCode
}

// cloneBytes 复制场景状态字节切片。
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

// mergeJSONObjects 将补丁 JSON 深度合并到原始 JSON。
func mergeJSONObjects(baseJSON []byte, patchJSON []byte) ([]byte, error) {
	var base map[string]any
	var patch map[string]any

	if len(baseJSON) == 0 {
		base = make(map[string]any)
	} else if err := json.Unmarshal(baseJSON, &base); err != nil {
		return nil, err
	}
	if len(patchJSON) == 0 {
		return json.Marshal(base)
	}
	if err := json.Unmarshal(patchJSON, &patch); err != nil {
		return nil, err
	}

	deepMerge(base, patch)
	return json.Marshal(base)
}

// deepMerge 对 JSON 对象执行递归深度合并。
func deepMerge(base map[string]any, patch map[string]any) {
	for key, patchValue := range patch {
		baseMap, baseIsMap := base[key].(map[string]any)
		patchMap, patchIsMap := patchValue.(map[string]any)
		if baseIsMap && patchIsMap {
			deepMerge(baseMap, patchMap)
			continue
		}
		base[key] = patchValue
	}
}
