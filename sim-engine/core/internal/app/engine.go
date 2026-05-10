// 模块：sim-engine/core/internal/app
// 文件职责：SimEngine Core 编排器 — 类型定义、会话生命周期、交互调度、协作校验。
// 时间控制 → engine_time.go / 快照 → engine_snapshot.go / 联动 → engine_link.go
// 教师干预与采集 → engine_teacher.go / 工具与 WS 推送 → engine_util.go
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §6 / §7 / §8 / §9。
//
// 关键设计：
// 1. 仅四种 WS 下行类型：render / event / control_ack / schema_invalidated（详 06.md §7.3）。
//    其余原有 state_full / state_diff / link_update / snapshot / teacher_summary 全部折叠：
//    - render 承载 RenderEnvelope（含 is_full_snapshot 标志）
//    - event 承载所有非渲染事件（用 payload.event 字段区分）
// 2. 时间回退：仅 step_back 单步（限单场景 process 模式 + 非联动 + 非混合，详 §5.2）。
// 3. 联动组通过 owner-based link.Engine：场景 Step/Action 返回 SharedStateDiff，
//    Engine 校验 owner 后 fan-out 给非源场景，下一次 Step 通过 IncomingLinkTriggers 注入。
// 4. 场景容器无状态：state 与 envelope 由 Core 持有；scene.State{SceneStateJSON, RenderEnvelopeJSON}。
// 5. 快照内容：所有 scene state + envelope + 各 link group shared state；恢复时整体回放。

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lenschain/sim-engine/core/internal/collector"
	"github.com/lenschain/sim-engine/core/internal/link"
	"github.com/lenschain/sim-engine/core/internal/scene"
	"github.com/lenschain/sim-engine/core/internal/session"
	"github.com/lenschain/sim-engine/core/internal/simcore"
	"github.com/lenschain/sim-engine/core/internal/ws"
)

// =====================================================================
// 对外请求 / 响应类型
// =====================================================================

// SceneConfig 是启动会话时单个场景的配置。
type SceneConfig struct {
	SceneCode            string
	LinkGroupCode        string // 该场景所属联动组 code（可空，单场景）
	ParamsJSON           []byte
	DataSourceConfigJSON []byte
	DataSourceMode       string
	// ContainerImageURL 场景算法容器镜像，按需启动 Pod 的依据。
	ContainerImageURL string
	// 资源请求覆盖（为空走 orchestrator 默认值）。
	ResourceRequestCPU    string
	ResourceRequestMemory string
}

// LinkFieldSpec 是联动组中一个 SharedState 字段的元描述（与 link.FieldSchema 1:1）。
type LinkFieldSpec struct {
	Name  string
	Type  string
	Owner string // 唯一可写场景 code
}

// LinkGroupSpec 是会话启动时显式声明的联动组（含 schema 与初始状态）。
//
// 上游（backend）必须从 sim_link_groups + sim_link_group_scenes 表组装该结构后透传到 Core；
// Core 不会从其它任何来源推断 schema（owner 模型必须显式）。
type LinkGroupSpec struct {
	Code           string
	Version        string
	Members        []string
	Fields         []LinkFieldSpec
	ForceClockSync bool
	InitialState   map[string]any
}

// StartSessionRequest 是 Core 内部启动会话请求。
type StartSessionRequest struct {
	InstanceID        string
	StudentID         string
	Seed              int64
	LinkageEnabled    bool
	SessionConfigJSON []byte
	Scenes            []SceneConfig
	LinkGroups        []LinkGroupSpec
}

// StartSessionResult 是会话启动结果。
type StartSessionResult struct {
	SessionID        string
	ActiveSceneCodes []string
}

// SessionState 是 Core 暴露给控制面的会话状态摘要。
type SessionState struct {
	SessionID        string
	InstanceID       string
	Status           string
	Tick             int64
	SimTimeSeconds   float64
	Speed            float64
	ActiveSceneCodes []string
	LinkGroupCodes   []string
	SceneStateJSON   []byte
	LastAction       string
	UpdatedAt        time.Time
}

// SessionBinding 表示 WebSocket 鉴权所需的会话归属信息。
type SessionBinding struct {
	SessionID  string
	InstanceID string
	StudentID  string
}

// TeacherSummary 是教师监控面板消费的会话摘要。
type TeacherSummary struct {
	SessionID            string          `json:"session_id"`
	InstanceID           string          `json:"instance_id"`
	Tick                 int64           `json:"tick"`
	Speed                float64         `json:"speed"`
	ActiveSceneCodes     []string        `json:"active_scene_codes"`
	LinkGroupCodes       []string        `json:"link_group_codes"`
	CollectionRunning    bool            `json:"collection_running"`
	PreviewSceneCode     string          `json:"preview_scene_code"`
	PreviewEnvelopeJSON  json.RawMessage `json:"preview_envelope_json,omitempty"`
	LastAction           string          `json:"last_action"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// InteractionResult 是 ActionDef 调用结果。
type InteractionResult struct {
	Success            bool
	ErrorMessage       string
	RenderEnvelopeJSON []byte
}

// InteractionSchemaResult 是 InteractionDefinition 查询结果。
type InteractionSchemaResult struct {
	SceneCode     string
	SchemaVersion string
	Actions       []scene.ActionDef
}

// Snapshot 是 Core 内部快照记录。
type Snapshot struct {
	SnapshotID  string
	SessionID   string
	Description string
	Tick        int64
	ObjectURL   string
	CreatedAt   time.Time
}

// =====================================================================
// 内部 runtime
// =====================================================================

// runtime 保存单个仿真会话在 Core 内部的运行时状态。
type runtime struct {
	opMu              *sync.Mutex
	clock             *simcore.Clock
	eventBus          *simcore.EventBus
	snapshotStack     *simcore.SnapshotStack
	lastSnapshotState []byte
	initialSnapshot   []byte
	lastAutoAdvanceAt time.Time
	instanceID        string
	studentID         string
	seed              int64
	status            string
	activeSceneCodes  []string
	linkageEnabled    bool
	sceneConfigs      map[string]SceneConfig
	sceneReady        map[string]bool
	sceneErrors       map[string]string
	linkGroupCodes    []string
	linkGroups        map[string]LinkGroupSpec
	pendingLinkTrigs  map[string][]scene.LinkTriggerRef // 按场景 code 缓存的下一 Step 待注入联动事件
	collaboration     collaborationPolicy
	lastAction        string
	updatedAt         time.Time
}

// =====================================================================
// Engine
// =====================================================================

// Engine 编排会话、场景和仿真时钟。
type Engine struct {
	mu       sync.RWMutex
	sessions *session.Manager
	scenes   *scene.Manager
	linker   *link.Engine
	collect  *collector.Manager
	hub      *ws.Hub
	store    SnapshotStore
	stateMgr *simcore.StateManager
	runtimes map[string]*runtime
}

// NewEngine 创建 Core 编排器。
func NewEngine(orch scene.Orchestrator) *Engine {
	return &Engine{
		sessions: session.NewManager(),
		scenes:   scene.NewManager(orch),
		linker:   link.NewEngine(),
		collect:  collector.NewManager(),
		hub:      ws.NewHub(),
		stateMgr: simcore.NewStateManager(),
		runtimes: make(map[string]*runtime),
	}
}

// SetSnapshotStore 为 Core 注入自定义快照存储实现。
func (e *Engine) SetSnapshotStore(store SnapshotStore) {
	if store == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store = store
}

// =====================================================================
// 会话生命周期
// =====================================================================

// StartSession 创建会话并启动所有配置的场景。
func (e *Engine) StartSession(ctx context.Context, req StartSessionRequest) (StartSessionResult, error) {
	sceneSessionConfigs := make([]session.SceneConfig, 0, len(req.Scenes))
	for _, item := range req.Scenes {
		sceneSessionConfigs = append(sceneSessionConfigs, session.SceneConfig{SceneCode: item.SceneCode})
	}
	created, err := e.sessions.Create(session.CreateRequest{
		InstanceID: req.InstanceID,
		StudentID:  req.StudentID,
		Scenes:     sceneSessionConfigs,
	})
	if err != nil {
		return StartSessionResult{}, err
	}

	sceneIndex := make(map[string]SceneConfig, len(req.Scenes))
	activeSceneCodes := make([]string, 0, len(req.Scenes))
	sceneReady := make(map[string]bool, len(req.Scenes))
	sceneErrors := make(map[string]string)
	sceneModes := make([]simcore.TimeControlMode, 0, len(req.Scenes))

	// 各场景无依赖关系，并行启动。
	var scenesMu sync.Mutex
	var wg sync.WaitGroup
	for i := range req.Scenes {
		item := req.Scenes[i]
		item.ParamsJSON = mergeSceneParams(item.ParamsJSON, item.LinkGroupCode)
		scenesMu.Lock()
		sceneIndex[item.SceneCode] = item
		scenesMu.Unlock()

		wg.Add(1)
		go func(item SceneConfig) {
			defer wg.Done()
			started, startErr := e.scenes.Start(ctx, scene.Config{
				SessionID:             created.SessionID,
				SceneCode:             item.SceneCode,
				ParamsJSON:            item.ParamsJSON,
				SharedStateJSON:       sharedStateJSONForGroup(req.LinkGroups, item.LinkGroupCode),
				ContainerImageURL:     item.ContainerImageURL,
				ResourceRequestCPU:    item.ResourceRequestCPU,
				ResourceRequestMemory: item.ResourceRequestMemory,
			}, req.InstanceID, req.StudentID, req.Seed)

			scenesMu.Lock()
			defer scenesMu.Unlock()
			if startErr != nil {
				sceneErrors[item.SceneCode] = startErr.Error()
				log.Printf("[StartSession] session=%s scene=%s start failed: %v", created.SessionID, item.SceneCode, startErr)
				return
			}
			if started.Meta.TimeControlMode != "" {
				sceneModes = append(sceneModes, simcore.TimeControlMode(started.Meta.TimeControlMode))
			}
			activeSceneCodes = append(activeSceneCodes, item.SceneCode)
			sceneReady[item.SceneCode] = true
		}(item)
	}
	wg.Wait()

	if len(activeSceneCodes) == 0 {
		_ = e.sessions.Destroy(created.SessionID)
		failedCodes := make([]string, 0, len(sceneErrors))
		for code := range sceneErrors {
			failedCodes = append(failedCodes, code)
		}
		sort.Strings(failedCodes)
		return StartSessionResult{}, fmt.Errorf("所有场景启动失败 (%s)，请联系运维查看 sim-engine 日志", strings.Join(failedCodes, ", "))
	}

	// 注册联动组。
	registeredGroups, err := e.registerLinkGroups(created.SessionID, req)
	if err != nil {
		e.cleanupFailedSessionStart(created.SessionID, nil)
		return StartSessionResult{}, err
	}
	mode := resolveSessionClockMode(sceneModes)
	collaboration, err := parseCollaborationPolicy(req.SessionConfigJSON)
	if err != nil {
		e.cleanupFailedSessionStart(created.SessionID, registeredGroups)
		return StartSessionResult{}, err
	}

	linkGroupsByCode := make(map[string]LinkGroupSpec, len(req.LinkGroups))
	for _, spec := range req.LinkGroups {
		linkGroupsByCode[spec.Code] = spec
	}

	e.mu.Lock()
	e.runtimes[created.SessionID] = &runtime{
		opMu:              &sync.Mutex{},
		clock:             simcore.NewClock(mode),
		eventBus:          simcore.NewEventBus(),
		snapshotStack:     simcore.NewSnapshotStack(1000, 50),
		lastAutoAdvanceAt: time.Now().UTC(),
		instanceID:        req.InstanceID,
		studentID:         req.StudentID,
		seed:              req.Seed,
		status:            session.StatusRunning,
		activeSceneCodes:  append([]string(nil), activeSceneCodes...),
		linkageEnabled:    req.LinkageEnabled,
		sceneConfigs:      sceneIndex,
		sceneReady:        sceneReady,
		sceneErrors:       sceneErrors,
		linkGroupCodes:    registeredGroups,
		linkGroups:        linkGroupsByCode,
		pendingLinkTrigs:  make(map[string][]scene.LinkTriggerRef),
		collaboration:     collaboration,
		updatedAt:         time.Now().UTC(),
	}
	e.mu.Unlock()

	// 推送各场景首帧 RenderEnvelope（is_full_snapshot=true 由场景容器在 Init 时已置位）。
	for _, runtimeRef := range e.scenes.ListBySession(created.SessionID) {
		e.publishRender(created.SessionID, runtimeRef.Config.SceneCode, runtimeRef.State.Tick, runtimeRef.State.RenderEnvelopeJSON)
	}
	_ = e.recordTickSnapshot(created.SessionID)
	if snapshotJSON, snapshotErr := e.buildTickSnapshotJSON(created.SessionID); snapshotErr == nil {
		e.mu.Lock()
		currentRuntime := e.runtimes[created.SessionID]
		currentRuntime.initialSnapshot = cloneBytes(snapshotJSON)
		currentRuntime.lastSnapshotState = cloneBytes(snapshotJSON)
		e.runtimes[created.SessionID] = currentRuntime
		e.mu.Unlock()
	}

	return StartSessionResult{
		SessionID:        created.SessionID,
		ActiveSceneCodes: activeSceneCodes,
	}, nil
}

// DestroySession 销毁会话运行时。
func (e *Engine) DestroySession(sessionID string) error {
	if err := e.sessions.Destroy(sessionID); err != nil {
		return err
	}
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if ok {
		for _, groupCode := range runtime.linkGroupCodes {
			e.linker.DeleteGroup(e.linkRuntimeCode(sessionID, groupCode))
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.runtimes, sessionID)
	e.collect.Delete(sessionID)
	return e.scenes.DestroySession(sessionID)
}

// SessionState 返回会话状态摘要。
func (e *Engine) SessionState(sessionID string) (SessionState, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return SessionState{}, false
	}
	return SessionState{
		SessionID:        sessionID,
		InstanceID:       runtime.instanceID,
		Status:           runtime.status,
		Tick:             runtime.clock.Tick(),
		SimTimeSeconds:   buildSimTimeSeconds(runtime.clock.Tick(), runtime.clock.Speed()),
		Speed:            runtime.clock.Speed(),
		ActiveSceneCodes: append([]string(nil), runtime.activeSceneCodes...),
		LinkGroupCodes:   append([]string(nil), runtime.linkGroupCodes...),
		SceneStateJSON:   e.buildSceneStateJSON(sessionID),
		LastAction:       runtime.lastAction,
		UpdatedAt:        runtime.updatedAt,
	}, true
}

// SessionBinding 返回 WebSocket 鉴权需要的会话归属信息。
func (e *Engine) SessionBinding(sessionID string) (SessionBinding, bool) {
	sessionState, ok := e.sessions.Get(sessionID)
	if !ok {
		return SessionBinding{}, false
	}
	return SessionBinding{
		SessionID:  sessionState.SessionID,
		InstanceID: sessionState.InstanceID,
		StudentID:  sessionState.StudentID,
	}, true
}

// SessionExists 判断会话是否存在。
func (e *Engine) SessionExists(sessionID string) bool {
	_, ok := e.sessions.Get(sessionID)
	return ok
}

// SubscribeMessages 订阅会话消息。
func (e *Engine) SubscribeMessages(sessionID string) ws.Subscription {
	return e.hub.Subscribe(sessionID)
}

// CurrentMessages 返回会话当前完整状态（用于断线重连首帧）。
//
// 重发所有场景缓存的 RenderEnvelope 作为 render 消息。
// 场景启动失败时附带 event 消息说明原因，避免学生看到空白画布。
func (e *Engine) CurrentMessages(sessionID string) []ws.Message {
	runtimes := e.scenes.ListBySession(sessionID)
	result := make([]ws.Message, 0, len(runtimes))
	for _, runtimeRef := range runtimes {
		result = append(result, ws.Message{
			Type:        ws.MessageTypeRender,
			SceneCode:   runtimeRef.Config.SceneCode,
			Tick:        runtimeRef.State.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: cloneBytes(runtimeRef.State.RenderEnvelopeJSON),
		})
	}
	e.mu.RLock()
	sessionRuntime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if ok {
		for sceneCode, reason := range sessionRuntime.sceneErrors {
			payload, err := json.Marshal(ws.EventPayload{
				Event: "scene_load_failed",
				Data: map[string]any{
					"scene_code":    sceneCode,
					"error_message": reason,
				},
			})
			if err != nil {
				continue
			}
			result = append(result, ws.Message{
				Type:        ws.MessageTypeEvent,
				SceneCode:   sceneCode,
				TimestampMS: time.Now().UTC().UnixMilli(),
				PayloadJSON: payload,
			})
		}
	}
	return result
}

// =====================================================================
// 交互
// =====================================================================

// SendInteraction 将交互操作路由到指定场景。
func (e *Engine) SendInteraction(
	ctx context.Context,
	sessionID string,
	sceneCode string,
	actionCode string,
	paramsJSON []byte,
	interactionCtx InteractionContext,
) (InteractionResult, error) {
	unlock, err := e.lockRuntimeOperation(sessionID)
	if err != nil {
		return InteractionResult{}, err
	}
	defer unlock()

	if err := e.validateInteraction(sessionID, sceneCode, actionCode, paramsJSON, interactionCtx); err != nil {
		return InteractionResult{}, err
	}

	sharedStateJSON := e.sharedStateForScene(sessionID, sceneCode)
	result, err := e.scenes.HandleAction(ctx, sessionID, sceneCode, actionCode, paramsJSON, sharedStateJSON, scene.ActionContext{
		ActorID:  interactionCtx.ActorID,
		UserRole: interactionCtx.UserRole,
	})
	if err != nil {
		e.handleSceneRuntimeFailure(sessionID, sceneCode, err)
		return InteractionResult{}, err
	}
	e.markRuntimeAction(sessionID, actionCode)
	e.applyLinkDiff(sessionID, sceneCode, actionCode, result.SharedStateDiffJSON)
	if err := e.recordTickSnapshot(sessionID); err != nil {
		return InteractionResult{}, err
	}
	e.publishRender(sessionID, sceneCode, e.currentTick(sessionID), result.RenderEnvelopeJSON)
	return InteractionResult{
		Success:            result.Success,
		ErrorMessage:       result.ErrorMessage,
		RenderEnvelopeJSON: result.RenderEnvelopeJSON,
	}, nil
}

// GetInteractionSchema 获取指定场景的交互面板定义。
func (e *Engine) GetInteractionSchema(ctx context.Context, sessionID string, sceneCode string) (InteractionSchemaResult, error) {
	def, err := e.scenes.InteractionSchema(ctx, sessionID, sceneCode)
	if err != nil {
		return InteractionSchemaResult{}, err
	}
	return InteractionSchemaResult{
		SceneCode:     def.SceneCode,
		SchemaVersion: def.SchemaVersion,
		Actions:       def.Actions,
	}, nil
}
// =====================================================================
// 内部 — 校验与多人协作
// =====================================================================

// lockRuntimeOperation 获取会话级串行操作锁。
func (e *Engine) lockRuntimeOperation(sessionID string) (func(), error) {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || runtime.opMu == nil {
		return nil, errors.New("session runtime not found")
	}
	runtime.opMu.Lock()
	return runtime.opMu.Unlock, nil
}

// validateInteraction 校验交互是否满足会话级多人协作权限规则。
func (e *Engine) validateInteraction(sessionID string, sceneCode string, actionCode string, paramsJSON []byte, ctx InteractionContext) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	return runtime.collaboration.Validate(sceneCode, actionCode, paramsJSON, ctx)
}
