// 模块：sim-engine/core/internal/scene
// 文件职责：场景运行时生命周期管理 + 场景算法容器调用契约（Manager / ScenarioClient / Orchestrator）。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §6.1 / §6.4 / §6.5。
//
// 关键设计：
// 1. 场景容器无状态：每次 Step / HandleAction 输入完整 SceneState，输出新 SceneState + RenderEnvelope。
// 2. Core 持有最新 RenderEnvelope；客户端断线重连时由 Manager 直接重发缓存（is_full_snapshot=true），
//    不再通过 GetRenderState rpc 拉取（该接口已废弃）。
// 3. Step 接收 Core LinkEngine fan-out 的 IncomingLinkTriggers；场景在本 tick 中可视化这些事件。
// 4. ActionDef / FieldDef 与 06.md §6.3 协议对齐（含 category / roles / cooldown /
//    writes_owned_fields / link_owner_fields / hybrid_channel / container_cmd 等字段）。

package scene

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// =====================================================================
// 配置 / 元信息 / 状态
// =====================================================================

// Config 是启动单个场景运行时所需的配置。
type Config struct {
	SessionID  string
	SceneCode  string
	ParamsJSON []byte
	// SharedStateJSON 联动共享状态当前快照（仅联动组场景）。
	SharedStateJSON []byte
	// ContainerImageURL 场景算法容器镜像地址（来自 backend 透传的 sim_scenarios.container_image_url）。
	ContainerImageURL string
	// ResourceRequestCPU / ResourceRequestMemory 容器资源请求，为空时使用 orchestrator 默认值。
	ResourceRequestCPU    string
	ResourceRequestMemory string
}

// Meta 是场景元信息。
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
	CustomRendererPackage   string
	SupportedLinkGroupCodes []string
}

// HealthStatus 表示场景容器健康状态。
type HealthStatus struct {
	Status      string
	Message     string
	CheckedAtMS int64
}

// State 表示某场景在某 tick 的状态快照。
//
// SceneStateJSON：场景算法内部状态（Core 持有，下次调用回传给容器）。
// RenderEnvelopeJSON：最近一次 RenderEnvelope（含 primitives / micro_steps / link_triggers /
// container_data；首帧 IsFullSnapshot=true）。
type State struct {
	SceneCode          string
	Tick               int64
	SceneStateJSON     []byte
	RenderEnvelopeJSON []byte
}

// LinkTriggerRef 是 Core 透传给场景的联动事件引用（与 link.LinkTrigger 同义）。
type LinkTriggerRef struct {
	ID             string
	SourceScene    string
	SourceAction   string
	LinkGroup      string
	ChangedFields  []string
	PayloadJSON    []byte
	TimestampMS    int64
	SourceAnchorID string
	TargetAnchorID string
}

// =====================================================================
// 请求 / 响应
// =====================================================================

// InitRequest 场景初始化请求。
type InitRequest struct {
	SessionID       string
	SceneCode       string
	InstanceID      string
	StudentID       string
	Seed            int64
	ParamsJSON      []byte
	SharedStateJSON []byte
}

// InitResult 场景初始化结果。
type InitResult struct {
	SceneCode           string
	Tick                int64
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// StepRequest 场景 tick 推进请求。
type StepRequest struct {
	SessionID            string
	SceneCode            string
	Tick                 int64
	SceneStateJSON       []byte
	SharedStateJSON      []byte
	IncomingLinkTriggers []LinkTriggerRef
}

// StepResult 场景 tick 推进结果。
type StepResult struct {
	SceneCode           string
	Tick                int64
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// ActionRequest 场景交互请求。
type ActionRequest struct {
	SessionID       string
	SceneCode       string
	Tick            int64
	SceneStateJSON  []byte
	SharedStateJSON []byte
	ActionCode      string
	ParamsJSON      []byte
	ActorID         string
	UserRole        string
}

// ActionResult 场景交互结果。
type ActionResult struct {
	SceneCode           string
	Tick                int64
	Success             bool
	ErrorMessage        string
	SceneStateJSON      []byte
	RenderEnvelopeJSON  []byte
	SharedStateDiffJSON []byte
}

// ActionContext 调用 ActionDef 时附带的操作者身份。
type ActionContext struct {
	ActorID  string
	UserRole string
}

// =====================================================================
// 交互定义（与 06.md §6.3 协议对齐）
// =====================================================================

// FieldDef 单个输入字段定义。
type FieldDef struct {
	Name        string
	Type        string
	Label       string
	Required    bool
	DefaultJSON []byte
	MinJSON     []byte
	MaxJSON     []byte
	StepJSON    []byte
	OptionsJSON []byte
	OptionsFrom string
}

// ActionDef 单个 ActionDef。
type ActionDef struct {
	ActionCode         string
	Label              string
	Description        string
	Category           string
	Trigger            string
	Fields             []FieldDef
	Roles              []string
	CooldownMs         int
	WritesOwnedFields []string
	LinkOwnerFields   []string
	HybridChannel      string
	ContainerCmd       string
}

// InteractionDefinition 场景对外暴露的全部 ActionDef。
type InteractionDefinition struct {
	SceneCode     string
	SchemaVersion string
	Actions       []ActionDef
}

// =====================================================================
// 接口契约
// =====================================================================

// ScenarioClient 是 Core 调用场景算法容器的最小接口。
type ScenarioClient interface {
	Meta(ctx context.Context) (Meta, error)
	InteractionSchema(ctx context.Context) (InteractionDefinition, error)
	Init(ctx context.Context, req InitRequest) (InitResult, error)
	Step(ctx context.Context, req StepRequest) (StepResult, error)
	HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error)
	Health(ctx context.Context) (HealthStatus, error)
	Close() error
}

// Orchestrator 是 Manager 唯一依赖的场景算法容器编排能力契约。
// 唯一生产实现：K8sOrchestrator（k8s_orchestrator.go）。
type Orchestrator interface {
	StartScene(ctx context.Context, config Config) (ScenarioClient, error)
	EvictScene(ctx context.Context, sessionID, sceneCode string) error
	DestroySession(ctx context.Context, sessionID string) error
}

// =====================================================================
// Runtime / Manager
// =====================================================================

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
	orch     Orchestrator
	runtimes map[string]Runtime
}

// NewManager 创建场景管理器。
func NewManager(orch Orchestrator) *Manager {
	if orch == nil {
		panic("scene.NewManager: orchestrator is required")
	}
	return &Manager{
		orch:     orch,
		runtimes: make(map[string]Runtime),
	}
}

// Start 启动场景并保存运行时（包含首帧 RenderEnvelope）。
func (m *Manager) Start(ctx context.Context, config Config, instanceID string, studentID string, seed int64) (Runtime, error) {
	if config.SceneCode == "" {
		return Runtime{}, errors.New("scene code is required")
	}

	client, err := m.orch.StartScene(ctx, config)
	if err != nil {
		return Runtime{}, err
	}
	meta, err := client.Meta(ctx)
	if err != nil {
		return Runtime{}, err
	}
	result, err := client.Init(ctx, InitRequest{
		SessionID:       config.SessionID,
		SceneCode:       config.SceneCode,
		InstanceID:      instanceID,
		StudentID:       studentID,
		Seed:            seed,
		ParamsJSON:      cloneBytes(config.ParamsJSON),
		SharedStateJSON: cloneBytes(config.SharedStateJSON),
	})
	if err != nil {
		return Runtime{}, err
	}

	runtime := Runtime{
		Config: config,
		Meta:   meta,
		State: State{
			SceneCode:          result.SceneCode,
			Tick:               result.Tick,
			SceneStateJSON:     cloneBytes(result.SceneStateJSON),
			RenderEnvelopeJSON: cloneBytes(result.RenderEnvelopeJSON),
		},
		client: client,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimes[runtimeKey(config.SessionID, config.SceneCode)] = runtime
	return runtime, nil
}

// Get 返回场景运行时副本。
func (m *Manager) Get(sessionID string, sceneCode string) (Runtime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	return runtime, ok
}

// CurrentEnvelope 返回当前场景缓存的 RenderEnvelope JSON（用于断线重连首帧）。
func (m *Manager) CurrentEnvelope(sessionID string, sceneCode string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	if !ok {
		return nil, false
	}
	return cloneBytes(runtime.State.RenderEnvelopeJSON), true
}

// Step 推进指定场景一个 tick；incoming 为 LinkEngine fan-out 给本场景的事件。
func (m *Manager) Step(ctx context.Context, sessionID string, sceneCode string, tick int64,
	sharedStateJSON []byte, incoming []LinkTriggerRef) (StepResult, error) {

	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return StepResult{}, errors.New("scene runtime not found")
	}

	result, err := runtime.client.Step(ctx, StepRequest{
		SessionID:            runtime.Config.SessionID,
		SceneCode:            sceneCode,
		Tick:                 tick,
		SceneStateJSON:       cloneBytes(runtime.State.SceneStateJSON),
		SharedStateJSON:      cloneBytes(sharedStateJSON),
		IncomingLinkTriggers: cloneTriggers(incoming),
	})
	if err != nil {
		return StepResult{}, err
	}

	runtime.State.Tick = result.Tick
	if result.SceneCode != "" {
		runtime.State.SceneCode = result.SceneCode
	}
	if result.SceneStateJSON != nil {
		runtime.State.SceneStateJSON = cloneBytes(result.SceneStateJSON)
	}
	if result.RenderEnvelopeJSON != nil {
		runtime.State.RenderEnvelopeJSON = cloneBytes(result.RenderEnvelopeJSON)
	}

	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return result, nil
}

// HandleAction 转发交互请求给场景容器。
func (m *Manager) HandleAction(ctx context.Context, sessionID string, sceneCode string,
	actionCode string, paramsJSON []byte, sharedStateJSON []byte, actionCtx ActionContext) (ActionResult, error) {

	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return ActionResult{}, errors.New("scene runtime not found")
	}

	result, err := runtime.client.HandleAction(ctx, ActionRequest{
		SessionID:       runtime.Config.SessionID,
		SceneCode:       sceneCode,
		Tick:            runtime.State.Tick,
		SceneStateJSON:  cloneBytes(runtime.State.SceneStateJSON),
		SharedStateJSON: cloneBytes(sharedStateJSON),
		ActionCode:      actionCode,
		ParamsJSON:      cloneBytes(paramsJSON),
		ActorID:         actionCtx.ActorID,
		UserRole:        actionCtx.UserRole,
	})
	if err != nil {
		return ActionResult{}, err
	}

	if result.SceneStateJSON != nil {
		runtime.State.SceneStateJSON = cloneBytes(result.SceneStateJSON)
	}
	if result.RenderEnvelopeJSON != nil {
		runtime.State.RenderEnvelopeJSON = cloneBytes(result.RenderEnvelopeJSON)
	}
	if result.Tick != 0 {
		runtime.State.Tick = result.Tick
	}
	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return result, nil
}

// InteractionSchema 返回指定场景的 InteractionDefinition。
func (m *Manager) InteractionSchema(ctx context.Context, sessionID string, sceneCode string) (InteractionDefinition, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return InteractionDefinition{}, errors.New("scene runtime not found")
	}
	return runtime.client.InteractionSchema(ctx)
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

// Restart 驱逐旧 Pod 与连接，启动新 Pod，并以最近 SceneState 重新 Init。
func (m *Manager) Restart(ctx context.Context, sessionID string, sceneCode string,
	instanceID string, studentID string, seed int64, sharedStateJSON []byte) (Runtime, error) {

	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(sessionID, sceneCode)]
	m.mu.RUnlock()
	if !ok {
		return Runtime{}, errors.New("scene runtime not found")
	}

	if runtime.client != nil {
		_ = runtime.client.Close()
	}
	if err := m.orch.EvictScene(ctx, sessionID, sceneCode); err != nil {
		return Runtime{}, err
	}

	client, err := m.orch.StartScene(ctx, runtime.Config)
	if err != nil {
		return Runtime{}, err
	}
	meta, err := client.Meta(ctx)
	if err != nil {
		return Runtime{}, err
	}
	result, err := client.Init(ctx, InitRequest{
		SessionID:       runtime.Config.SessionID,
		SceneCode:       runtime.Config.SceneCode,
		InstanceID:      instanceID,
		StudentID:       studentID,
		Seed:            seed,
		ParamsJSON:      cloneBytes(runtime.Config.ParamsJSON),
		SharedStateJSON: cloneBytes(sharedStateJSON),
	})
	if err != nil {
		return Runtime{}, err
	}

	runtime.client = client
	runtime.Meta = meta
	if result.SceneStateJSON != nil {
		runtime.State.SceneStateJSON = cloneBytes(result.SceneStateJSON)
	}
	if result.RenderEnvelopeJSON != nil {
		runtime.State.RenderEnvelopeJSON = cloneBytes(result.RenderEnvelopeJSON)
	}
	if result.Tick != 0 {
		runtime.State.Tick = result.Tick
	}

	m.mu.Lock()
	m.runtimes[runtimeKey(sessionID, sceneCode)] = runtime
	m.mu.Unlock()
	return runtime, nil
}

// ListBySession 返回会话内所有场景运行时副本。
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

// RestoreState 用快照中的场景状态覆盖当前运行时状态（用于实例恢复）。
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

// DestroySession 销毁指定会话下的全部场景运行时并回调 Orchestrator。
func (m *Manager) DestroySession(sessionID string) error {
	m.mu.Lock()
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
	m.mu.Unlock()

	if err := m.orch.DestroySession(context.Background(), sessionID); err != nil && closeErr == nil {
		closeErr = err
	}
	return closeErr
}

// InjectCollectionPatch 将采集事件补丁合并进指定场景的 SceneState 与 RenderEnvelope。
//
// 本方法仅做纯 JSON 深度合并；不调用场景容器；用于混合实验把 ContainerMetric 注入到
// 当前缓存的 RenderEnvelope（典型路径：补 RenderEnvelope.container_data[]）。
func (m *Manager) InjectCollectionPatch(sessionID string, sceneCode string, patchJSON []byte) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := runtimeKey(sessionID, sceneCode)
	runtime, ok := m.runtimes[key]
	if !ok {
		return State{}, errors.New("scene runtime not found")
	}

	stateJSON, err := mergeJSONObjects(runtime.State.SceneStateJSON, patchJSON)
	if err != nil {
		return State{}, err
	}
	envelopeJSON, err := mergeJSONObjects(runtime.State.RenderEnvelopeJSON, patchJSON)
	if err != nil {
		return State{}, err
	}

	runtime.State.SceneStateJSON = stateJSON
	runtime.State.RenderEnvelopeJSON = envelopeJSON
	m.runtimes[key] = runtime
	return runtime.State, nil
}

// =====================================================================
// 内部工具
// =====================================================================

func runtimeKey(sessionID string, sceneCode string) string {
	return sessionID + "::" + sceneCode
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

func cloneTriggers(items []LinkTriggerRef) []LinkTriggerRef {
	if len(items) == 0 {
		return nil
	}
	out := make([]LinkTriggerRef, len(items))
	for i, item := range items {
		out[i] = item
		out[i].ChangedFields = append([]string(nil), item.ChangedFields...)
		out[i].PayloadJSON = cloneBytes(item.PayloadJSON)
	}
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
