package scenario

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
)

// Server 将场景接口适配为生成后的远程过程调用服务端实现。
type Server struct {
	simscenariov1.UnimplementedSimScenarioServiceServer
	scenario Scenario
	mu       sync.RWMutex
	state    State
	meta     *Meta
}

// NewServer 创建场景远程过程调用服务适配器。
func NewServer(scenario Scenario) *Server {
	return &Server{scenario: scenario}
}

// Init 初始化场景。
func (s *Server) Init(ctx context.Context, req *simscenariov1.InitRequest) (*simscenariov1.InitResponse, error) {
	state, err := s.scenario.Init(ctx, InitRequest{
		SceneCode:        req.GetSceneCode(),
		InstanceID:       req.GetInstanceId(),
		StudentID:        req.GetStudentId(),
		Seed:             req.GetSeed(),
		SessionID:        req.GetSessionId(),
		ParamsJSON:       req.GetParamsJson(),
		InitialStateJSON: req.GetInitialStateJson(),
		SharedStateJSON:  req.GetSharedStateJson(),
	})
	if err != nil {
		return nil, err
	}
	s.storeState(state)
	return &simscenariov1.InitResponse{
		SceneCode:       req.GetSceneCode(),
		Tick:            state.Tick,
		StateJson:       state.StateJSON,
		RenderStateJson: state.RenderStateJSON,
		SharedStateJson: state.SharedStateJSON,
	}, nil
}

// Step 推进一个仿真时钟步。
func (s *Server) Step(ctx context.Context, req *simscenariov1.StepRequest) (*simscenariov1.StepResponse, error) {
	result, err := s.scenario.Step(ctx, StepRequest{
		SessionID:       req.GetSessionId(),
		SceneCode:       req.GetSceneCode(),
		Tick:            req.GetTick(),
		StateJSON:       req.GetStateJson(),
		SharedStateJSON: req.GetSharedStateJson(),
	})
	if err != nil {
		return nil, err
	}
	s.storeState(State{
		Tick:            result.Tick,
		StateJSON:       result.StateJSON,
		RenderStateJSON: result.RenderStateJSON,
	})
	return &simscenariov1.StepResponse{
		SceneCode:           req.GetSceneCode(),
		Tick:                result.Tick,
		StateJson:           result.StateJSON,
		RenderStateJson:     result.RenderStateJSON,
		Events:              toProtoEvents(result.Events),
		SharedStateDiffJson: result.SharedStateDiffJSON,
	}, nil
}

// HandleAction 处理场景交互。未声明交互能力的场景会被直接拒绝，避免以伪成功响应掩盖契约缺失。
func (s *Server) HandleAction(ctx context.Context, req *simscenariov1.HandleActionRequest) (*simscenariov1.HandleActionResponse, error) {
	interactive, ok := s.scenario.(InteractiveScenario)
	if !ok {
		return nil, errors.New("当前场景未声明交互能力，请实现 InteractiveScenario 接口")
	}

	result, err := interactive.HandleAction(ctx, ActionRequest{
		SessionID:       req.GetSessionId(),
		SceneCode:       req.GetSceneCode(),
		ActionCode:      req.GetActionCode(),
		ParamsJSON:      req.GetParamsJson(),
		StateJSON:       req.GetStateJson(),
		SharedStateJSON: req.GetSharedStateJson(),
		Tick:            req.GetTick(),
		ActorID:         req.GetActorId(),
		RoleKey:         req.GetRoleKey(),
	})
	if err != nil {
		return nil, err
	}
	s.storeState(State{
		Tick:            req.GetTick(),
		StateJSON:       result.StateJSON,
		RenderStateJSON: result.RenderStateJSON,
	})

	return &simscenariov1.HandleActionResponse{
		SceneCode:           req.GetSceneCode(),
		Tick:                req.GetTick(),
		Success:             result.Success,
		ErrorMessage:        result.ErrorMessage,
		StateJson:           result.StateJSON,
		RenderStateJson:     result.RenderStateJSON,
		Events:              toProtoEvents(result.Events),
		SharedStateDiffJson: result.SharedStateDiff,
	}, nil
}

// GetRenderState 返回场景当前缓存的可视化状态。
func (s *Server) GetRenderState(ctx context.Context, req *simscenariov1.GetRenderStateRequest) (*simscenariov1.GetRenderStateResponse, error) {
	meta, err := s.cachedMeta(ctx)
	if err != nil {
		return nil, err
	}
	state := s.loadState()
	if provider, ok := s.scenario.(RenderStateProvider); ok {
		state, err = provider.RenderState(ctx, RenderStateRequest{
			SessionID:       req.GetSessionId(),
			SceneCode:       req.GetSceneCode(),
			Tick:            req.GetTick(),
			StateJSON:       req.GetStateJson(),
			SharedStateJSON: req.GetSharedStateJson(),
		})
		if err != nil {
			return nil, err
		}
		s.storeState(state)
	} else {
		if len(req.GetStateJson()) > 0 {
			state.StateJSON = req.GetStateJson()
		}
		if req.GetTick() != 0 {
			state.Tick = req.GetTick()
		}
	}
	metricsJSON, events := extractRenderSupplements(state)
	return &simscenariov1.GetRenderStateResponse{
		SceneCode:       req.GetSceneCode(),
		Category:        string(meta.Category),
		AlgorithmType:   meta.AlgorithmType,
		Tick:            state.Tick,
		StateJson:       state.StateJSON,
		RenderStateJson: state.RenderStateJSON,
		MetricsJson:     metricsJSON,
		Events:          events,
	}, nil
}

// GetInteractionSchema 返回交互面板定义。未声明 schema 的场景会被直接拒绝，保证协议契约完整。
func (s *Server) GetInteractionSchema(ctx context.Context, _ *simscenariov1.GetInteractionSchemaRequest) (*simscenariov1.GetInteractionSchemaResponse, error) {
	provider, ok := s.scenario.(InteractionSchemaProvider)
	if !ok {
		return nil, errors.New("当前场景未声明交互面板，请实现 InteractionSchemaProvider 接口")
	}

	schema, err := provider.InteractionSchema(ctx)
	if err != nil {
		return nil, err
	}
	if err := ValidateInteractionSchema(schema); err != nil {
		return nil, err
	}

	return &simscenariov1.GetInteractionSchemaResponse{
		SceneCode: schema.SceneCode,
		Actions:   toProtoActions(schema.Actions),
	}, nil
}

// GetMeta 返回场景元信息。
func (s *Server) GetMeta(ctx context.Context, _ *simscenariov1.GetMetaRequest) (*simscenariov1.GetMetaResponse, error) {
	meta, err := s.cachedMeta(ctx)
	if err != nil {
		return nil, err
	}
	return &simscenariov1.GetMetaResponse{Meta: toProtoMeta(meta)}, nil
}

// HealthCheck 返回场景服务健康状态。
func (s *Server) HealthCheck(context.Context, *simscenariov1.HealthCheckRequest) (*simscenariov1.HealthCheckResponse, error) {
	return &simscenariov1.HealthCheckResponse{
		Status:      simscenariov1.HealthStatus_HEALTH_STATUS_SERVING,
		Message:     "服务正常",
		CheckedAtMs: time.Now().UTC().UnixMilli(),
	}, nil
}

// cachedMeta 读取或缓存场景元信息。
func (s *Server) cachedMeta(ctx context.Context) (Meta, error) {
	s.mu.RLock()
	if s.meta != nil {
		meta := *s.meta
		s.mu.RUnlock()
		return meta, nil
	}
	s.mu.RUnlock()

	meta, err := s.scenario.Meta(ctx)
	if err != nil {
		return Meta{}, err
	}
	if err := ValidateMeta(meta); err != nil {
		return Meta{}, err
	}
	s.mu.Lock()
	s.meta = &meta
	s.mu.Unlock()
	return meta, nil
}

// storeState 更新服务端缓存的最新场景状态。
func (s *Server) storeState(state State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = State{
		Tick:            state.Tick,
		StateJSON:       cloneBytes(state.StateJSON),
		RenderStateJSON: cloneBytes(state.RenderStateJSON),
		SharedStateJSON: cloneBytes(state.SharedStateJSON),
	}
}

// loadState 读取服务端缓存的最新场景状态副本。
func (s *Server) loadState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return State{
		Tick:            s.state.Tick,
		StateJSON:       cloneBytes(s.state.StateJSON),
		RenderStateJSON: cloneBytes(s.state.RenderStateJSON),
		SharedStateJSON: cloneBytes(s.state.SharedStateJSON),
	}
}

// cloneBytes 复制状态字节切片，避免共享底层数组。
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

// toProtoMeta 将场景元信息转换为协议结构。
func toProtoMeta(meta Meta) *simscenariov1.ScenarioMeta {
	return &simscenariov1.ScenarioMeta{
		Code:                    meta.Code,
		Name:                    meta.Name,
		Category:                string(meta.Category),
		AlgorithmType:           meta.AlgorithmType,
		Description:             meta.Description,
		Version:                 meta.Version,
		TimeControlMode:         toProtoTimeControlMode(meta.TimeControlMode),
		DataSourceMode:          toProtoDataSourceMode(meta.DataSourceMode),
		DefaultParamsJson:       meta.DefaultParams,
		DefaultStateJson:        meta.DefaultState,
		SupportedLinkGroupCodes: append([]string(nil), meta.SupportedLinkGroupCodes...),
	}
}

// extractRenderSupplements 从完整状态中提取指标与事件，补齐协议返回。
func extractRenderSupplements(state State) ([]byte, []*simscenariov1.SimEvent) {
	if len(state.StateJSON) == 0 {
		return nil, nil
	}
	var payload struct {
		Metrics  any `json:"metrics"`
		Timeline []struct {
			ID          string `json:"id"`
			Tick        int64  `json:"tick"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Tone        string `json:"tone"`
		} `json:"timeline"`
	}
	if err := json.Unmarshal(state.StateJSON, &payload); err != nil {
		return nil, nil
	}
	metricsJSON, err := json.Marshal(payload.Metrics)
	if err != nil {
		metricsJSON = nil
	}
	events := make([]*simscenariov1.SimEvent, 0, len(payload.Timeline))
	for _, item := range payload.Timeline {
		eventPayload, err := json.Marshal(map[string]any{
			"title":       item.Title,
			"description": item.Description,
			"tone":        item.Tone,
		})
		if err != nil {
			continue
		}
		events = append(events, &simscenariov1.SimEvent{
			EventId:     item.ID,
			EventType:   "timeline",
			Tick:        item.Tick,
			TimestampMs: time.Now().UTC().UnixMilli(),
			PayloadJson: eventPayload,
		})
	}
	return metricsJSON, events
}

// toProtoTimeControlMode 将时间模式转换为协议枚举。
func toProtoTimeControlMode(mode TimeControlMode) simscenariov1.TimeControlMode {
	switch mode {
	case TimeControlModeProcess:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_PROCESS
	case TimeControlModeReactive:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_REACTIVE
	case TimeControlModeContinuous:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_CONTINUOUS
	default:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_UNSPECIFIED
	}
}

// toProtoDataSourceMode 将数据源模式转换为协议枚举。
func toProtoDataSourceMode(mode DataSourceMode) simscenariov1.DataSourceMode {
	switch mode {
	case DataSourceModeSimulation:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION
	case DataSourceModeCollection:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION
	case DataSourceModeDual:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL
	default:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_UNSPECIFIED
	}
}

// toProtoActions 将交互动作集合转换为协议结构。
func toProtoActions(actions []InteractionAction) []*simscenariov1.InteractionAction {
	result := make([]*simscenariov1.InteractionAction, 0, len(actions))
	for _, action := range actions {
		result = append(result, &simscenariov1.InteractionAction{
			ActionCode:   action.ActionCode,
			Label:        action.Label,
			Description:  action.Description,
			Trigger:      toProtoTrigger(action.Trigger),
			Fields:       toProtoFields(action.Fields),
			UiSchemaJson: action.UISchemaJSON,
		})
	}
	return result
}

// toProtoFields 将交互字段集合转换为协议结构。
func toProtoFields(fields []InteractionField) []*simscenariov1.InteractionField {
	result := make([]*simscenariov1.InteractionField, 0, len(fields))
	for _, field := range fields {
		result = append(result, &simscenariov1.InteractionField{
			Key:            field.Key,
			Label:          field.Label,
			Type:           toProtoFieldType(field.Type),
			Required:       field.Required,
			DefaultValue:   field.DefaultValue,
			Options:        toProtoOptions(field.Options),
			ValidationJson: field.ValidationJSON,
		})
	}
	return result
}

// toProtoOptions 将交互选项集合转换为协议结构。
func toProtoOptions(options []InteractionOption) []*simscenariov1.InteractionOption {
	result := make([]*simscenariov1.InteractionOption, 0, len(options))
	for _, option := range options {
		result = append(result, &simscenariov1.InteractionOption{
			Value: option.Value,
			Label: option.Label,
		})
	}
	return result
}

// toProtoFieldType 将字段类型转换为协议枚举。
func toProtoFieldType(fieldType InteractionFieldType) simscenariov1.InteractionFieldType {
	switch fieldType {
	case InteractionFieldTypeString:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_STRING
	case InteractionFieldTypeNumber:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NUMBER
	case InteractionFieldTypeBoolean:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_BOOLEAN
	case InteractionFieldTypeSelect:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_SELECT
	case InteractionFieldTypeNodeRef:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NODE_REF
	case InteractionFieldTypeRange:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_RANGE
	case InteractionFieldTypeJSON:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_JSON
	default:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_UNSPECIFIED
	}
}

// toProtoTrigger 将交互触发器转换为协议枚举。
func toProtoTrigger(trigger InteractionTrigger) simscenariov1.InteractionTrigger {
	switch trigger {
	case InteractionTriggerClick:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CLICK
	case InteractionTriggerFormSubmit:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_FORM_SUBMIT
	case InteractionTriggerDrag:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_DRAG
	case InteractionTriggerCanvasSelect:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CANVAS_SELECT
	default:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_UNSPECIFIED
	}
}

// toProtoEvents 将场景事件集合转换为协议结构。
func toProtoEvents(events []Event) []*simscenariov1.SimEvent {
	result := make([]*simscenariov1.SimEvent, 0, len(events))
	for _, event := range events {
		result = append(result, &simscenariov1.SimEvent{
			EventId:     event.EventID,
			EventType:   event.EventType,
			SceneCode:   event.SceneCode,
			Tick:        event.Tick,
			TimestampMs: event.TimestampMS,
			PayloadJson: event.PayloadJSON,
		})
	}
	return result
}
