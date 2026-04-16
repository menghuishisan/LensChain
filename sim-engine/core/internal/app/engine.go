// Package app 编排 SimEngine Core 会话、时钟、联动与采集运行时。
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// SceneConfig 是启动会话时传入的场景配置。
type SceneConfig struct {
	SceneCode            string
	LinkGroupCode        string
	ParamsJSON           []byte
	InitialStateJSON     []byte
	DataSourceConfigJSON []byte
	DataSourceMode       string
	SharedStateJSON      []byte
}

// StartSessionRequest 是 Core 内部启动会话请求。
type StartSessionRequest struct {
	InstanceID        string
	StudentID         string
	LinkageEnabled    bool
	SessionConfigJSON []byte
	Scenes            []SceneConfig
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
	SessionID         string          `json:"session_id"`
	InstanceID        string          `json:"instance_id"`
	Tick              int64           `json:"tick"`
	Speed             float64         `json:"speed"`
	ActiveSceneCodes  []string        `json:"active_scene_codes"`
	LinkGroupCodes    []string        `json:"link_group_codes"`
	CollectionRunning bool            `json:"collection_running"`
	PreviewSceneCode  string          `json:"preview_scene_code"`
	PreviewStateJSON  json.RawMessage `json:"preview_state_json,omitempty"`
	LastAction        string          `json:"last_action"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// InteractionResult 是场景交互操作结果。
type InteractionResult struct {
	Success         bool
	ErrorMessage    string
	RenderStateJSON []byte
}

// InteractionSchemaResult 是场景交互面板定义查询结果。
type InteractionSchemaResult struct {
	SceneCode string
	Actions   []scene.InteractionAction
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

// runtime 保存单个仿真会话在 Core 内部的运行时状态。
type runtime struct {
	opMu              *sync.Mutex
	clock             *simcore.Clock
	eventBus          *simcore.EventBus
	snapshots         map[string]Snapshot
	snapshotStack     *simcore.SnapshotStack
	lastSnapshotState []byte
	initialSnapshot   []byte
	lastAutoAdvanceAt time.Time
	instanceID        string
	status            string
	activeSceneCodes  []string
	linkageEnabled    bool
	sceneConfigs      map[string]SceneConfig
	sceneReady        map[string]bool
	sceneErrors       map[string]string
	linkGroupCodes    []string
	collaboration     collaborationPolicy
	lastAction        string
	updatedAt         time.Time
}

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
	runtimes map[string]runtime
}

// NewEngine 创建 Core 编排器。
func NewEngine(factory scene.ClientFactory) *Engine {
	return &Engine{
		sessions: session.NewManager(),
		scenes:   scene.NewManager(factory),
		linker:   link.NewEngine(),
		collect:  collector.NewManager(),
		hub:      ws.NewHub(),
		stateMgr: simcore.NewStateManager(),
		runtimes: make(map[string]runtime),
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

// StartSession 创建会话并启动所有配置的场景。
func (e *Engine) StartSession(ctx context.Context, req StartSessionRequest) (StartSessionResult, error) {
	sceneConfigs := make([]session.SceneConfig, 0, len(req.Scenes))
	for _, item := range req.Scenes {
		sceneConfigs = append(sceneConfigs, session.SceneConfig{SceneCode: item.SceneCode})
	}

	created, err := e.sessions.Create(session.CreateRequest{
		InstanceID: req.InstanceID,
		StudentID:  req.StudentID,
		Scenes:     sceneConfigs,
	})
	if err != nil {
		return StartSessionResult{}, err
	}

	sceneIndex := make(map[string]SceneConfig, len(req.Scenes))
	activeSceneCodes := make([]string, 0, len(req.Scenes))
	sceneReady := make(map[string]bool, len(req.Scenes))
	sceneErrors := make(map[string]string)
	sceneModes := make([]simcore.TimeControlMode, 0, len(req.Scenes))
	for _, item := range req.Scenes {
		item.ParamsJSON = mergeSceneParams(item.ParamsJSON, item.LinkGroupCode)
		sceneIndex[item.SceneCode] = item
		started, err := e.scenes.Start(ctx, scene.Config{
			SessionID:        created.SessionID,
			SceneCode:        item.SceneCode,
			ParamsJSON:       item.ParamsJSON,
			InitialStateJSON: item.InitialStateJSON,
			SharedStateJSON:  item.SharedStateJSON,
		})
		if err != nil {
			sceneErrors[item.SceneCode] = err.Error()
			continue
		}
		if started.Meta.TimeControlMode != "" {
			sceneModes = append(sceneModes, simcore.TimeControlMode(started.Meta.TimeControlMode))
		}
		activeSceneCodes = append(activeSceneCodes, item.SceneCode)
		sceneReady[item.SceneCode] = true
	}
	if len(activeSceneCodes) == 0 {
		_ = e.sessions.Destroy(created.SessionID)
		return StartSessionResult{}, errors.New("all scenes failed to start")
	}

	linkGroupCodes, err := e.registerLinkGroups(created.SessionID, req)
	if err != nil {
		e.cleanupFailedSessionStart(created.SessionID, nil)
		return StartSessionResult{}, err
	}
	mode := resolveSessionClockMode(sceneModes)
	collaboration, err := parseCollaborationPolicy(req.SessionConfigJSON)
	if err != nil {
		e.cleanupFailedSessionStart(created.SessionID, linkGroupCodes)
		return StartSessionResult{}, err
	}

	e.mu.Lock()
	e.runtimes[created.SessionID] = runtime{
		opMu:              &sync.Mutex{},
		clock:             simcore.NewClock(mode),
		eventBus:          simcore.NewEventBus(),
		snapshotStack:     simcore.NewSnapshotStack(1000, 50),
		lastAutoAdvanceAt: time.Now().UTC(),
		instanceID:        req.InstanceID,
		status:            session.StatusRunning,
		activeSceneCodes:  append([]string(nil), activeSceneCodes...),
		linkageEnabled:    req.LinkageEnabled,
		sceneConfigs:      sceneIndex,
		sceneReady:        sceneReady,
		sceneErrors:       sceneErrors,
		linkGroupCodes:    linkGroupCodes,
		collaboration:     collaboration,
		updatedAt:         time.Now().UTC(),
	}
	e.mu.Unlock()

	for _, runtime := range e.scenes.ListBySession(created.SessionID) {
		e.hub.Publish(created.SessionID, ws.Message{
			Type:        ws.MessageTypeStateFull,
			SceneCode:   runtime.Config.SceneCode,
			Tick:        runtime.State.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: cloneBytes(runtime.State.RenderStateJSON),
		})
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

// ControlTime 执行会话级时间控制指令。
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
	case "rewind_to":
		targetTick, err := parseTargetTick(paramsJSON)
		if err != nil {
			e.publishControlAck(sessionID, command, err)
			return err
		}
		err = e.rewindToTick(sessionID, targetTick)
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
	default:
		err := errors.New("unsupported time control command")
		e.publishControlAck(sessionID, command, err)
		return err
	}
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

// parseTargetTick 从控制参数中解析目标 tick。
func parseTargetTick(paramsJSON []byte) (int64, error) {
	var params struct {
		TargetTick *int64 `json:"target_tick"`
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return 0, err
	}
	if params.TargetTick == nil {
		return 0, errors.New("target_tick is required")
	}
	return *params.TargetTick, nil
}

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

	previousRuntime, _ := e.scenes.Get(sessionID, sceneCode)
	sharedStateJSON := e.sharedStateForScene(sessionID, sceneCode)
	result, err := e.scenes.HandleAction(ctx, sessionID, sceneCode, actionCode, paramsJSON, sharedStateJSON, scene.ActionContext{
		ActorID: interactionCtx.ActorID,
		RoleKey: interactionCtx.RoleKey,
	})
	if err != nil {
		e.handleSceneRuntimeFailure(sessionID, sceneCode, err)
		return InteractionResult{}, err
	}
	e.markRuntimeAction(sessionID, actionCode)
	e.applyLinkDiff(sessionID, sceneCode, result.SharedStateDiffJSON, sharedStateJSON)
	e.appendSceneEvents(sessionID, result.Events)
	if err := e.recordTickSnapshot(sessionID); err != nil {
		return InteractionResult{}, err
	}
	diffJSON, err := e.buildRenderStateDiff(previousRuntime.State.RenderStateJSON, result.RenderStateJSON)
	if err != nil {
		return InteractionResult{}, err
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeStateDiff,
		SceneCode:   sceneCode,
		Tick:        e.currentTick(sessionID),
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: diffJSON,
	})
	return InteractionResult{
		Success:         result.Success,
		ErrorMessage:    result.ErrorMessage,
		RenderStateJSON: result.RenderStateJSON,
	}, nil
}

// GetInteractionSchema 获取指定场景的交互面板定义。
func (e *Engine) GetInteractionSchema(ctx context.Context, sessionID string, sceneCode string) (InteractionSchemaResult, error) {
	schema, err := e.scenes.InteractionSchema(ctx, sessionID, sceneCode)
	if err != nil {
		return InteractionSchemaResult{}, err
	}
	return InteractionSchemaResult{
		SceneCode: schema.SceneCode,
		Actions:   schema.Actions,
	}, nil
}

// CreateSnapshot 为会话创建持久化快照记录。
func (e *Engine) CreateSnapshot(sessionID string, description string) (Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok {
		return Snapshot{}, errors.New("session runtime not found")
	}
	if e.store == nil {
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
		Scenes:    e.buildSnapshotScenes(sessionID),
	}
	objectURL, err := e.store.Save(snapshotID, payload)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.ObjectURL = objectURL
	if runtime.snapshots == nil {
		runtime.snapshots = make(map[string]Snapshot)
	}
	runtime.snapshots[snapshotID] = snapshot
	runtime.updatedAt = time.Now().UTC()
	e.runtimes[sessionID] = runtime
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeSnapshot,
		Tick:        snapshot.Tick,
		TimestampMS: snapshot.CreatedAt.UnixMilli(),
		PayloadJSON: []byte(`{"snapshot_id":"` + snapshot.SnapshotID + `","snapshot_type":"manual"}`),
	})
	return snapshot, nil
}

// RestoreSnapshot 校验快照存在并恢复到快照 tick。
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
	snapshot, ok := runtime.snapshots[snapshotID]
	if !ok {
		return errors.New("snapshot not found")
	}
	payload, err := e.store.Load(snapshotID)
	if err != nil {
		return err
	}
	if err := runtime.clock.Rewind(payload.Tick); err != nil && payload.Tick != runtime.clock.Tick() {
		runtime.clock.Reset()
		for runtime.clock.Tick() < payload.Tick {
			if err := runtime.clock.Step(); err != nil {
				return err
			}
		}
	}
	for _, sceneSnapshot := range payload.Scenes {
		e.scenes.RestoreState(sessionID, sceneSnapshot.SceneCode, scene.State{
			SceneCode:       sceneSnapshot.SceneCode,
			Tick:            payload.Tick,
			StateJSON:       cloneBytes(sceneSnapshot.StateJSON),
			RenderStateJSON: cloneBytes(sceneSnapshot.RenderStateJSON),
			SharedStateJSON: cloneBytes(sceneSnapshot.SharedStateJSON),
		})
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeStateFull,
			SceneCode:   sceneSnapshot.SceneCode,
			Tick:        payload.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: cloneBytes(sceneSnapshot.RenderStateJSON),
		})
	}
	restoredScenes := make(map[string]struct {
		StateJSON       []byte `json:"state_json"`
		RenderStateJSON []byte `json:"render_state_json"`
		SharedStateJSON []byte `json:"shared_state_json"`
	}, len(payload.Scenes))
	for _, sceneSnapshot := range payload.Scenes {
		restoredScenes[sceneSnapshot.SceneCode] = struct {
			StateJSON       []byte `json:"state_json"`
			RenderStateJSON []byte `json:"render_state_json"`
			SharedStateJSON []byte `json:"shared_state_json"`
		}{
			StateJSON:       cloneBytes(sceneSnapshot.StateJSON),
			RenderStateJSON: cloneBytes(sceneSnapshot.RenderStateJSON),
			SharedStateJSON: cloneBytes(sceneSnapshot.SharedStateJSON),
		}
	}
	e.syncLinkStatesFromScenes(sessionID, restoredScenes)
	if snapshotJSON, snapshotErr := e.buildTickSnapshotJSON(sessionID); snapshotErr == nil {
		e.mu.Lock()
		runtime = e.runtimes[sessionID]
		runtime.lastSnapshotState = cloneBytes(snapshotJSON)
		runtime.lastAutoAdvanceAt = time.Now().UTC()
		e.runtimes[sessionID] = runtime
		e.mu.Unlock()
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeSnapshot,
		Tick:        snapshot.Tick,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: []byte(`{"snapshot_id":"` + snapshotID + `","snapshot_type":"restore"}`),
	})
	return nil
}

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
	e.runtimes[sessionID] = runtime
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
	e.runtimes[sessionID] = runtime
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
	e.runtimes[sessionID] = runtime
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
		previousRuntime, _ := e.scenes.Get(sessionID, sceneCode)
		state, injectErr := e.scenes.InjectCollectionPatch(sessionID, sceneCode, patch.PatchJSON)
		if injectErr != nil {
			_ = e.collect.RecordError(sessionID, injectErr)
			e.publishCollectorInterrupted(sessionID, injectErr)
			return injectErr
		}
		diffJSON, diffErr := e.buildRenderStateDiff(previousRuntime.State.RenderStateJSON, state.RenderStateJSON)
		if diffErr != nil {
			return diffErr
		}
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeStateDiff,
			SceneCode:   sceneCode,
			Tick:        e.currentTick(sessionID),
			TimestampMS: patch.TimestampMS,
			PayloadJSON: diffJSON,
		})
	}
	if err := e.recordTickSnapshot(sessionID); err != nil {
		return err
	}
	return nil
}

// lockRuntimeOperation 获取会话级串行操作锁，确保多人协作操作按服务端顺序执行。
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
		SessionID:         sessionID,
		InstanceID:        sessionState.InstanceID,
		Tick:              sessionState.Tick,
		Speed:             sessionState.Speed,
		ActiveSceneCodes:  sessionState.ActiveSceneCodes,
		LinkGroupCodes:    sessionState.LinkGroupCodes,
		CollectionRunning: e.collectorRunning(sessionID),
		PreviewSceneCode:  preview.SceneCode,
		PreviewStateJSON:  json.RawMessage(cloneBytes(preview.RenderStateJSON)),
		LastAction:        sessionState.LastAction,
		UpdatedAt:         sessionState.UpdatedAt,
	}, true
}

// PublishTeacherSummaries 将所有会话摘要推送到各自消息总线，供教师监控端订阅。
func (e *Engine) PublishTeacherSummaries() {
	sessionIDs := e.sessionIDs()
	for _, sessionID := range sessionIDs {
		summary, ok := e.BuildTeacherSummary(sessionID)
		if !ok {
			continue
		}
		payload, err := json.Marshal(summary)
		if err != nil {
			continue
		}
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeTeacherSummary,
			Tick:        summary.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: payload,
		})
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

// SubscribeMessages 订阅会话消息。
func (e *Engine) SubscribeMessages(sessionID string) ws.Subscription {
	return e.hub.Subscribe(sessionID)
}

// SessionExists 判断会话是否存在。
func (e *Engine) SessionExists(sessionID string) bool {
	_, ok := e.sessions.Get(sessionID)
	return ok
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

// CurrentMessages 返回会话当前完整状态消息。
func (e *Engine) CurrentMessages(sessionID string) []ws.Message {
	runtimes := e.scenes.ListBySession(sessionID)
	result := make([]ws.Message, 0, len(runtimes))
	for _, runtime := range runtimes {
		result = append(result, ws.Message{
			Type:        ws.MessageTypeStateFull,
			SceneCode:   runtime.Config.SceneCode,
			Tick:        runtime.State.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: cloneBytes(runtime.State.RenderStateJSON),
		})
	}
	e.mu.RLock()
	sessionRuntime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if ok {
		for sceneCode, reason := range sessionRuntime.sceneErrors {
			payload, err := json.Marshal(map[string]any{
				"event":         "scene_load_failed",
				"scene_code":    sceneCode,
				"error_message": reason,
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

// stepScenes 推进会话内所有场景一个 tick，并广播状态变更。
func (e *Engine) stepScenes(ctx context.Context, sessionID string, tick int64) error {
	for _, runtime := range e.scenes.ListBySession(sessionID) {
		if !shouldAdvanceScene(runtime.Meta.TimeControlMode) {
			continue
		}
		previousRenderStateJSON := cloneBytes(runtime.State.RenderStateJSON)
		sharedStateJSON := e.sharedStateForScene(sessionID, runtime.Config.SceneCode)
		result, err := e.scenes.Step(ctx, sessionID, runtime.Config.SceneCode, tick, sharedStateJSON)
		if err != nil {
			e.handleSceneRuntimeFailure(sessionID, runtime.Config.SceneCode, err)
			return err
		}
		e.appendSceneEvents(sessionID, result.Events)
		e.applyLinkDiff(sessionID, runtime.Config.SceneCode, result.SharedStateDiffJSON, sharedStateJSON)
		diffJSON, err := e.buildRenderStateDiff(previousRenderStateJSON, result.RenderStateJSON)
		if err != nil {
			return err
		}
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeStateDiff,
			SceneCode:   runtime.Config.SceneCode,
			Tick:        result.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: diffJSON,
		})
		e.publishSceneEvents(sessionID, result.Events)
	}
	return nil
}

// buildRenderStateDiff 计算两次渲染状态之间的增量载荷。
func (e *Engine) buildRenderStateDiff(previousJSON []byte, currentJSON []byte) ([]byte, error) {
	return e.stateMgr.BuildDiff(previousJSON, currentJSON)
}

// shouldAdvanceScene 判断当前场景是否允许被会话时钟推进。
func shouldAdvanceScene(mode string) bool {
	switch simcore.TimeControlMode(mode) {
	case simcore.TimeControlModeProcess, simcore.TimeControlModeContinuous:
		return true
	default:
		return false
	}
}

// handleSceneRuntimeFailure 在场景容器异常后尝试自动重启并恢复最近快照状态。
func (e *Engine) handleSceneRuntimeFailure(sessionID string, sceneCode string, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, restartErr := e.scenes.Restart(ctx, sessionID, sceneCode)
	recoveryErr := error(nil)
	if restartErr != nil {
		e.setRuntimeStatus(sessionID, session.StatusError)
	} else {
		recoveryErr = e.RecoverLatestTickSnapshot(sessionID)
		if recoveryErr != nil {
			e.setRuntimeStatus(sessionID, session.StatusError)
		}
		e.markSceneReady(sessionID, sceneCode, true, "")
	}
	payload := map[string]any{
		"event":      "scene_runtime_failure",
		"scene_code": sceneCode,
		"error":      cause.Error(),
		"recovered":  restartErr == nil && recoveryErr == nil,
	}
	if restartErr != nil {
		payload["recovery_error"] = restartErr.Error()
	} else if recoveryErr != nil {
		payload["recovery_error"] = recoveryErr.Error()
	}
	if data, err := json.Marshal(payload); err == nil {
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeEvent,
			SceneCode:   sceneCode,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: data,
		})
	}
}

// publishCollectorInterrupted 在采集失败时向前端发送中断事件。
func (e *Engine) publishCollectorInterrupted(sessionID string, cause error) {
	if cause == nil {
		return
	}
	data, err := json.Marshal(map[string]any{
		"event":         "collector_interrupted",
		"error_message": cause.Error(),
	})
	if err != nil {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeEvent,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: data,
	})
}

// publishControlAck 向前端广播时间控制命令的执行结果。
func (e *Engine) publishControlAck(sessionID string, command string, err error) {
	payload := map[string]any{
		"command": command,
		"success": err == nil,
	}
	if err != nil {
		payload["error_message"] = err.Error()
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeControlAck,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: data,
	})
}

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

// registerLinkGroups 根据会话请求注册联动组共享状态空间。
func (e *Engine) registerLinkGroups(sessionID string, req StartSessionRequest) ([]string, error) {
	if !req.LinkageEnabled {
		return nil, nil
	}

	groups := make(map[string][]string)
	initialStates := make(map[string][]byte)
	for _, item := range req.Scenes {
		if item.LinkGroupCode == "" {
			continue
		}
		if err := e.validateSceneLinkGroup(sessionID, item.SceneCode, item.LinkGroupCode); err != nil {
			return nil, err
		}
		groups[item.LinkGroupCode] = append(groups[item.LinkGroupCode], item.SceneCode)
		if initialStates[item.LinkGroupCode] == nil && len(item.SharedStateJSON) > 0 {
			initialStates[item.LinkGroupCode] = cloneBytes(item.SharedStateJSON)
		}
	}

	registered := make([]string, 0, len(groups))
	for groupCode, sceneCodes := range groups {
		if err := e.linker.RegisterGroup(link.Group{
			Code:       e.linkRuntimeCode(sessionID, groupCode),
			SceneCodes: sceneCodes,
			StateJSON:  initialStates[groupCode],
		}); err != nil {
			for _, registeredGroupCode := range registered {
				e.linker.DeleteGroup(e.linkRuntimeCode(sessionID, registeredGroupCode))
			}
			return nil, err
		}
		registered = append(registered, groupCode)
	}
	return registered, nil
}

// validateSceneLinkGroup 校验场景声明的联动组是否与元信息兼容。
func (e *Engine) validateSceneLinkGroup(sessionID string, sceneCode string, linkGroupCode string) error {
	runtime, ok := e.scenes.Get(sessionID, sceneCode)
	if !ok {
		return nil
	}
	for _, supportedGroupCode := range runtime.Meta.SupportedLinkGroupCodes {
		if supportedGroupCode == linkGroupCode {
			return nil
		}
	}
	return errors.New("scene " + sceneCode + " 不支持联动组 " + linkGroupCode)
}

// cleanupFailedSessionStart 回滚启动过程中已创建的会话、联动组和场景运行时，避免残留脏状态。
func (e *Engine) cleanupFailedSessionStart(sessionID string, linkGroupCodes []string) {
	for _, groupCode := range linkGroupCodes {
		e.linker.DeleteGroup(e.linkRuntimeCode(sessionID, groupCode))
	}
	_ = e.scenes.DestroySession(sessionID)
	e.collect.Delete(sessionID)
	_ = e.sessions.Destroy(sessionID)
}

// sharedStateForScene 返回某个场景当前可见的共享状态。
func (e *Engine) sharedStateForScene(sessionID string, sceneCode string) []byte {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return nil
	}

	config, ok := runtime.sceneConfigs[sceneCode]
	if !ok || config.LinkGroupCode == "" {
		return nil
	}
	if !e.isLinkGroupReady(runtime, config.LinkGroupCode) {
		return nil
	}
	sharedState, ok := e.linker.SharedState(e.linkRuntimeCode(sessionID, config.LinkGroupCode))
	if !ok {
		return nil
	}
	return sharedState
}

// applyLinkDiff 将场景返回的共享状态 diff 合并进联动组状态，并广播联动更新。
func (e *Engine) applyLinkDiff(sessionID string, sourceScene string, diffJSON []byte, previousSharedState []byte) {
	if len(diffJSON) == 0 {
		return
	}

	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return
	}

	sceneConfig, ok := runtime.sceneConfigs[sourceScene]
	if !ok || sceneConfig.LinkGroupCode == "" {
		return
	}
	if !e.isLinkGroupReady(runtime, sceneConfig.LinkGroupCode) {
		return
	}

	update, err := e.linker.ApplyDiff(e.linkRuntimeCode(sessionID, sceneConfig.LinkGroupCode), sourceScene, diffJSON)
	if err != nil {
		return
	}
	e.scenes.UpdateSharedState(sessionID, sourceScene, update.StateJSON)
	for _, affectedScene := range update.AffectedScenes {
		e.scenes.UpdateSharedState(sessionID, affectedScene, update.StateJSON)
	}
	e.refreshAffectedLinkedScenes(sessionID, update.AffectedScenes, update.StateJSON)

	payload, err := json.Marshal(map[string]any{
		"link_group_code": sceneConfig.LinkGroupCode,
		"source_scene":    update.SourceScene,
		"affected_scenes": update.AffectedScenes,
		"changed_keys":    flattenJSONPaths(diffJSON),
		"previous_state":  json.RawMessage(previousSharedState),
		"state":           json.RawMessage(update.StateJSON),
	})
	if err != nil {
		return
	}

	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeLinkUpdate,
		SceneCode:   sourceScene,
		Tick:        e.currentTick(sessionID),
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: payload,
	})
}

// refreshAffectedLinkedScenes 在共享状态变更后主动刷新受影响场景，满足自然联动刷新要求。
func (e *Engine) refreshAffectedLinkedScenes(sessionID string, sceneCodes []string, sharedStateJSON []byte) {
	if len(sceneCodes) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, sceneCode := range sceneCodes {
		previousRuntime, ok := e.scenes.Get(sessionID, sceneCode)
		if !ok {
			continue
		}
		state, err := e.scenes.RefreshRenderState(ctx, sessionID, sceneCode, sharedStateJSON)
		if err != nil {
			continue
		}
		diffJSON, err := e.buildRenderStateDiff(previousRuntime.State.RenderStateJSON, state.RenderStateJSON)
		if err != nil {
			continue
		}
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeStateDiff,
			SceneCode:   sceneCode,
			Tick:        state.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: diffJSON,
		})
	}
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
	e.runtimes[sessionID] = runtime
}

// appendSceneEvents 将场景事件写入会话事件总线。
func (e *Engine) appendSceneEvents(sessionID string, events []scene.Event) {
	if len(events) == 0 {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	runtime, ok := e.runtimes[sessionID]
	if !ok || runtime.eventBus == nil {
		return
	}
	busEvents := make([]simcore.Event, 0, len(events))
	for _, event := range events {
		busEvents = append(busEvents, simcore.Event{
			EventID:     event.EventID,
			EventType:   event.EventType,
			SceneCode:   event.SceneCode,
			Tick:        event.Tick,
			TimestampMS: event.TimestampMS,
			PayloadJSON: cloneBytes(event.PayloadJSON),
		})
	}
	runtime.eventBus.Append(busEvents)
	runtime.updatedAt = time.Now().UTC()
	e.runtimes[sessionID] = runtime
}

// publishSceneEvents 将场景事件广播给前端。
func (e *Engine) publishSceneEvents(sessionID string, events []scene.Event) {
	for _, event := range events {
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeEvent,
			SceneCode:   event.SceneCode,
			Tick:        event.Tick,
			TimestampMS: event.TimestampMS,
			PayloadJSON: cloneBytes(event.PayloadJSON),
		})
	}
}

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
	e.runtimes[sessionID] = runtime
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
	e.runtimes[sessionID] = runtime
}

// isLinkGroupReady 判断联动组中的全部场景是否都已就绪。
func (e *Engine) isLinkGroupReady(runtime runtime, groupCode string) bool {
	if groupCode == "" {
		return false
	}
	for sceneCode, config := range runtime.sceneConfigs {
		if config.LinkGroupCode != groupCode {
			continue
		}
		if !runtime.sceneReady[sceneCode] {
			return false
		}
	}
	return true
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

// teacherPreview 表示教师监控卡片展示的默认预览内容。
type teacherPreview struct {
	SceneCode       string
	RenderStateJSON []byte
}

// buildTeacherPreview 为教师监控卡片选择一个默认缩略图场景。
func (e *Engine) buildTeacherPreview(sessionID string) teacherPreview {
	runtimes := e.scenes.ListBySession(sessionID)
	if len(runtimes) == 0 {
		return teacherPreview{}
	}
	renderStateJSON := cloneBytes(runtimes[0].State.RenderStateJSON)
	if len(renderStateJSON) == 0 {
		renderStateJSON = cloneBytes(runtimes[0].State.StateJSON)
	}
	return teacherPreview{
		SceneCode:       runtimes[0].Config.SceneCode,
		RenderStateJSON: renderStateJSON,
	}
}

// collectorRunning 判断指定会话的采集器当前是否处于运行状态。
func (e *Engine) collectorRunning(sessionID string) bool {
	session, ok := e.collect.Get(sessionID)
	return ok && session.Running
}

// buildSnapshotScenes 构造会话完整快照所需的场景载荷。
func (e *Engine) buildSnapshotScenes(sessionID string) []SnapshotScene {
	runtimes := e.scenes.ListBySession(sessionID)
	result := make([]SnapshotScene, 0, len(runtimes))
	for _, runtime := range runtimes {
		result = append(result, SnapshotScene{
			SceneCode:       runtime.Config.SceneCode,
			StateJSON:       cloneBytes(runtime.State.StateJSON),
			RenderStateJSON: cloneBytes(runtime.State.RenderStateJSON),
			SharedStateJSON: cloneBytes(runtime.State.SharedStateJSON),
		})
	}
	return result
}

// buildSceneStateJSON 构造会话当前场景状态摘要 JSON。
func (e *Engine) buildSceneStateJSON(sessionID string) []byte {
	states := make([]simcore.SceneStateSnapshot, 0)
	for _, runtime := range e.scenes.ListBySession(sessionID) {
		states = append(states, simcore.SceneStateSnapshot{
			SceneCode:       runtime.Config.SceneCode,
			Tick:            runtime.State.Tick,
			RenderStateJSON: cloneBytes(runtime.State.RenderStateJSON),
		})
	}
	return e.stateMgr.BuildSceneSummary(states)
}

// recordTickSnapshot 按文档策略记录 tick 快照。
// 每个 tick 保存增量 diff，每 50 tick 保存关键帧，并最多保留最近 1000 tick。
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
	keyframeTick := runtime.clock.Tick()
	if runtime.clock.Tick()%50 == 0 {
		keyframeJSON = cloneBytes(currentJSON)
	}
	runtime.snapshotStack.Save(runtime.clock.Tick(), keyframeJSON, diffJSON)
	runtime.lastSnapshotState = cloneBytes(currentJSON)
	e.runtimes[sessionID] = runtime
	if keyframeJSON != nil {
		e.publishKeyframeSnapshot(sessionID, keyframeTick)
	}
	return nil
}

// publishKeyframeSnapshot 广播自动关键帧快照通知，便于前端同步回退锚点信息。
func (e *Engine) publishKeyframeSnapshot(sessionID string, tick int64) {
	payload, err := json.Marshal(map[string]any{
		"snapshot_id":   fmt.Sprintf("keyframe-%s-%d", sessionID, tick),
		"snapshot_type": "keyframe",
	})
	if err != nil {
		return
	}
	e.hub.Publish(sessionID, ws.Message{
		Type:        ws.MessageTypeSnapshot,
		Tick:        tick,
		TimestampMS: time.Now().UTC().UnixMilli(),
		PayloadJSON: payload,
	})
}

// rewindToTick 使用快照栈恢复到目标 tick。
func (e *Engine) rewindToTick(sessionID string, targetTick int64) error {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return errors.New("session runtime not found")
	}
	if runtime.snapshotStack == nil {
		return runtime.clock.Rewind(targetTick)
	}

	keyframe, ok := runtime.snapshotStack.NearestKeyframe(targetTick)
	if !ok {
		return errors.New("snapshot keyframe not found")
	}
	restoredJSON := cloneBytes(keyframe.StateJSON)
	for _, diff := range runtime.snapshotStack.DiffsAfter(keyframe.Tick, targetTick) {
		restoredJSON, _ = e.stateMgr.MergeDiff(restoredJSON, diff.DiffJSON)
	}
	if err := e.restoreTickState(sessionID, targetTick, restoredJSON); err != nil {
		return err
	}

	e.mu.Lock()
	runtime = e.runtimes[sessionID]
	runtime.clock.Reset()
	for runtime.clock.Tick() < targetTick {
		if err := runtime.clock.Step(); err != nil {
			e.mu.Unlock()
			return err
		}
	}
	runtime.lastSnapshotState = cloneBytes(restoredJSON)
	runtime.lastAutoAdvanceAt = time.Now().UTC()
	e.runtimes[sessionID] = runtime
	e.mu.Unlock()
	return nil
}

// buildTickSnapshotJSON 构造当前 tick 的完整状态 JSON。
func (e *Engine) buildTickSnapshotJSON(sessionID string) ([]byte, error) {
	states := make([]simcore.SceneStateSnapshot, 0)
	for _, runtime := range e.scenes.ListBySession(sessionID) {
		states = append(states, simcore.SceneStateSnapshot{
			SceneCode:       runtime.Config.SceneCode,
			Tick:            runtime.State.Tick,
			StateJSON:       cloneBytes(runtime.State.StateJSON),
			RenderStateJSON: cloneBytes(runtime.State.RenderStateJSON),
			SharedStateJSON: cloneBytes(runtime.State.SharedStateJSON),
		})
	}
	return e.stateMgr.BuildTickSnapshot(sessionID, e.currentTick(sessionID), states)
}

// restoreTickState 将某个 tick 的完整状态 JSON 恢复回会话运行时。
func (e *Engine) restoreTickState(sessionID string, targetTick int64, snapshotJSON []byte) error {
	var payload struct {
		SessionID string `json:"session_id"`
		Tick      int64  `json:"tick"`
		Scenes    map[string]struct {
			StateJSON       []byte `json:"state_json"`
			RenderStateJSON []byte `json:"render_state_json"`
			SharedStateJSON []byte `json:"shared_state_json"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(snapshotJSON, &payload); err != nil {
		return err
	}
	for sceneCode, sceneState := range payload.Scenes {
		e.scenes.RestoreState(sessionID, sceneCode, scene.State{
			SceneCode:       sceneCode,
			Tick:            targetTick,
			StateJSON:       cloneBytes(sceneState.StateJSON),
			RenderStateJSON: cloneBytes(sceneState.RenderStateJSON),
			SharedStateJSON: cloneBytes(sceneState.SharedStateJSON),
		})
		e.hub.Publish(sessionID, ws.Message{
			Type:        ws.MessageTypeStateFull,
			SceneCode:   sceneCode,
			Tick:        targetTick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: cloneBytes(sceneState.RenderStateJSON),
		})
	}
	e.syncLinkStatesFromScenes(sessionID, payload.Scenes)
	return nil
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
func (e *Engine) currentSceneStepDuration(sessionID string, mode simcore.TimeControlMode) time.Duration {
	if mode != simcore.TimeControlModeProcess && mode != simcore.TimeControlModeContinuous {
		return time.Second
	}

	baseMS := int64(1000)
	for _, runtime := range e.scenes.ListBySession(sessionID) {
		if simcore.TimeControlMode(runtime.Meta.TimeControlMode) != mode {
			continue
		}
		var snapshot struct {
			StepDuration int64 `json:"step_duration_ms"`
		}
		if err := json.Unmarshal(runtime.State.StateJSON, &snapshot); err == nil && snapshot.StepDuration > 0 {
			baseMS = snapshot.StepDuration
			break
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
	e.runtimes[sessionID] = runtime
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
	runtime = e.runtimes[sessionID]
	runtime.clock.Reset()
	runtime.lastSnapshotState = cloneBytes(runtime.initialSnapshot)
	runtime.snapshotStack = simcore.NewSnapshotStack(1000, 50)
	runtime.snapshotStack.Save(0, cloneBytes(runtime.initialSnapshot), cloneBytes(runtime.initialSnapshot))
	runtime.lastAutoAdvanceAt = time.Now().UTC()
	e.runtimes[sessionID] = runtime
	e.mu.Unlock()
	return nil
}

// syncLinkStatesFromScenes 根据已恢复的场景共享状态重建联动组共享状态。
func (e *Engine) syncLinkStatesFromScenes(sessionID string, scenes map[string]struct {
	StateJSON       []byte `json:"state_json"`
	RenderStateJSON []byte `json:"render_state_json"`
	SharedStateJSON []byte `json:"shared_state_json"`
}) {
	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok || !runtime.linkageEnabled {
		return
	}

	groupStates := make(map[string][]byte)
	for sceneCode, sceneState := range scenes {
		config, configOK := runtime.sceneConfigs[sceneCode]
		if !configOK || config.LinkGroupCode == "" || len(sceneState.SharedStateJSON) == 0 {
			continue
		}
		if groupStates[config.LinkGroupCode] == nil {
			groupStates[config.LinkGroupCode] = cloneBytes(sceneState.SharedStateJSON)
		}
	}
	for groupCode, stateJSON := range groupStates {
		_ = e.linker.ReplaceState(e.linkRuntimeCode(sessionID, groupCode), stateJSON)
	}
}

// linkRuntimeCode 返回某个会话内部使用的联动组唯一键。
func (e *Engine) linkRuntimeCode(sessionID string, groupCode string) string {
	return sessionID + "::" + groupCode
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

// mergeSceneParams 将运行时选中的联动组编码注入场景参数，保证场景内部只有一条联动配置来源。
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

// flattenJSONPaths 将联动 diff JSON 展平成路径列表，供前端按文档高亮变更字段。
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
