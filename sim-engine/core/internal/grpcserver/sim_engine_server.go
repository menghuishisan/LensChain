package grpcserver

import (
	"context"
	"strings"

	simenginev1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_engine/v1"
	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"

	"github.com/lenschain/sim-engine/core/internal/app"
	"github.com/lenschain/sim-engine/core/internal/scene"
)

// SimEngineServer 将内部编排器适配为远程过程调用控制面服务。
type SimEngineServer struct {
	simenginev1.UnimplementedSimEngineServiceServer
	engine     *app.Engine
	publicBase string
}

// NewSimEngineServer 创建 SimEngine 远程过程调用控制面服务。
func NewSimEngineServer(engine *app.Engine, publicBase string) *SimEngineServer {
	return &SimEngineServer{
		engine:     engine,
		publicBase: strings.TrimRight(publicBase, "/"),
	}
}

// CreateSession 创建仿真会话。
func (s *SimEngineServer) CreateSession(ctx context.Context, req *simenginev1.CreateSessionRequest) (*simenginev1.CreateSessionResponse, error) {
	scenes := make([]app.SceneConfig, 0, len(req.GetScenes()))
	for _, item := range req.GetScenes() {
		scenes = append(scenes, app.SceneConfig{
			SceneCode:            item.GetSceneCode(),
			LinkGroupCode:        item.GetLinkGroupCode(),
			ParamsJSON:           item.GetParamsJson(),
			InitialStateJSON:     item.GetInitialStateJson(),
			DataSourceConfigJSON: item.GetDataSourceConfigJson(),
			DataSourceMode:       dataSourceModeString(item.GetDataSourceMode()),
			SharedStateJSON:      item.GetSharedStateJson(),
		})
	}

	started, err := s.engine.StartSession(ctx, app.StartSessionRequest{
		InstanceID:        req.GetInstanceId(),
		StudentID:         req.GetStudentId(),
		LinkageEnabled:    req.GetLinkageEnabled(),
		SessionConfigJSON: req.GetSessionConfigJson(),
		Scenes:            scenes,
	})
	if err != nil {
		return nil, err
	}

	return &simenginev1.CreateSessionResponse{
		SessionId:        started.SessionID,
		InstanceId:       req.GetInstanceId(),
		Status:           simenginev1.SessionStatus_SESSION_STATUS_RUNNING,
		WebsocketUrl:     s.websocketURL(started.SessionID),
		ActiveSceneCodes: started.ActiveSceneCodes,
	}, nil
}

// dataSourceModeString 将协议数据源模式转换为内部字符串编码。
func dataSourceModeString(mode simscenariov1.DataSourceMode) string {
	switch mode {
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION:
		return "collection"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL:
		return "dual"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION:
		return "simulation"
	default:
		return ""
	}
}

// GetSessionState 获取会话状态摘要。
func (s *SimEngineServer) GetSessionState(_ context.Context, req *simenginev1.GetSessionStateRequest) (*simenginev1.GetSessionStateResponse, error) {
	state, ok := s.engine.SessionState(req.GetSessionId())
	if !ok {
		return &simenginev1.GetSessionStateResponse{
			SessionId: req.GetSessionId(),
			Status:    simenginev1.SessionStatus_SESSION_STATUS_ERROR,
		}, nil
	}
	return &simenginev1.GetSessionStateResponse{
		SessionId:        req.GetSessionId(),
		InstanceId:       state.InstanceID,
		Status:           toProtoSessionStatus(state.Status),
		SimTick:          state.Tick,
		SimTimeSeconds:   state.SimTimeSeconds,
		Speed:            state.Speed,
		ActiveSceneCodes: state.ActiveSceneCodes,
		LinkGroupCodes:   state.LinkGroupCodes,
		SceneStateJson:   state.SceneStateJSON,
		LastAction:       state.LastAction,
		UpdatedAtMs:      state.UpdatedAt.UnixMilli(),
	}, nil
}

// toProtoSessionStatus 将内部会话状态转换为协议枚举。
func toProtoSessionStatus(status string) simenginev1.SessionStatus {
	switch status {
	case "paused":
		return simenginev1.SessionStatus_SESSION_STATUS_PAUSED
	case "error":
		return simenginev1.SessionStatus_SESSION_STATUS_ERROR
	case "stopped":
		return simenginev1.SessionStatus_SESSION_STATUS_STOPPED
	case "creating":
		return simenginev1.SessionStatus_SESSION_STATUS_CREATING
	default:
		return simenginev1.SessionStatus_SESSION_STATUS_RUNNING
	}
}

// ControlTime 控制仿真时钟。
func (s *SimEngineServer) ControlTime(_ context.Context, req *simenginev1.ControlTimeRequest) (*simenginev1.ControlTimeResponse, error) {
	command := commandString(req.GetCommand())
	if command == "" {
		return &simenginev1.ControlTimeResponse{
			Success:      false,
			ErrorMessage: "unsupported time control command",
		}, nil
	}
	err := s.engine.ControlTime(req.GetSessionId(), command, req.GetParamsJson())
	if err != nil {
		return &simenginev1.ControlTimeResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.ControlTimeResponse{Success: true}, nil
}

// DestroySession 销毁会话。
func (s *SimEngineServer) DestroySession(_ context.Context, req *simenginev1.DestroySessionRequest) (*simenginev1.DestroySessionResponse, error) {
	if err := s.engine.DestroySession(req.GetSessionId()); err != nil {
		return &simenginev1.DestroySessionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.DestroySessionResponse{Success: true}, nil
}

// SendInteraction 处理交互。
func (s *SimEngineServer) SendInteraction(ctx context.Context, req *simenginev1.SendInteractionRequest) (*simenginev1.SendInteractionResponse, error) {
	result, err := s.engine.SendInteraction(
		ctx,
		req.GetSessionId(),
		req.GetSceneCode(),
		req.GetActionCode(),
		req.GetParamsJson(),
		app.InteractionContext{
			ActorID: req.GetActorId(),
			RoleKey: req.GetRoleKey(),
		},
	)
	if err != nil {
		return &simenginev1.SendInteractionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.SendInteractionResponse{
		Success:      result.Success,
		ErrorMessage: result.ErrorMessage,
		DataJson:     result.RenderStateJSON,
	}, nil
}

// GetInteractionSchema 返回场景专属交互面板定义。
func (s *SimEngineServer) GetInteractionSchema(ctx context.Context, req *simenginev1.GetInteractionSchemaRequest) (*simenginev1.GetInteractionSchemaResponse, error) {
	schema, err := s.engine.GetInteractionSchema(ctx, req.GetSessionId(), req.GetSceneCode())
	if err != nil {
		return nil, err
	}
	return &simenginev1.GetInteractionSchemaResponse{
		SceneCode: schema.SceneCode,
		Actions:   toProtoInteractionActions(schema.Actions),
	}, nil
}

// CreateSnapshot 创建快照。
func (s *SimEngineServer) CreateSnapshot(_ context.Context, req *simenginev1.CreateSnapshotRequest) (*simenginev1.CreateSnapshotResponse, error) {
	snapshot, err := s.engine.CreateSnapshot(req.GetSessionId(), req.GetDescription())
	if err != nil {
		return nil, err
	}
	return &simenginev1.CreateSnapshotResponse{
		SnapshotId:   snapshot.SnapshotID,
		SessionId:    snapshot.SessionID,
		SnapshotType: "manual",
		SimTick:      snapshot.Tick,
		ObjectUrl:    snapshot.ObjectURL,
		CreatedAtMs:  snapshot.CreatedAt.UnixMilli(),
	}, nil
}

// RestoreSnapshot 恢复快照。
func (s *SimEngineServer) RestoreSnapshot(_ context.Context, req *simenginev1.RestoreSnapshotRequest) (*simenginev1.RestoreSnapshotResponse, error) {
	if err := s.engine.RestoreSnapshot(req.GetSessionId(), req.GetSnapshotId()); err != nil {
		return &simenginev1.RestoreSnapshotResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.RestoreSnapshotResponse{Success: true}, nil
}

// StartDataCollection 启动 Collector 采集。
func (s *SimEngineServer) StartDataCollection(_ context.Context, req *simenginev1.StartDataCollectionRequest) (*simenginev1.StartDataCollectionResponse, error) {
	if err := s.engine.StartDataCollection(req.GetSessionId(), req.GetCollectorConfigJson()); err != nil {
		return &simenginev1.StartDataCollectionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.StartDataCollectionResponse{Success: true}, nil
}

// StopDataCollection 停止 Collector 采集。
func (s *SimEngineServer) StopDataCollection(_ context.Context, req *simenginev1.StopDataCollectionRequest) (*simenginev1.StopDataCollectionResponse, error) {
	if err := s.engine.StopDataCollection(req.GetSessionId()); err != nil {
		return &simenginev1.StopDataCollectionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.StopDataCollectionResponse{Success: true}, nil
}

// websocketURL 构造会话数据通道地址。
func (s *SimEngineServer) websocketURL(sessionID string) string {
	return s.publicBase + "/api/v1/ws/sim-engine/" + sessionID
}

// commandString 将协议时间控制枚举转换为内部命令字符串。
func commandString(command simenginev1.TimeControlCommand) string {
	switch command {
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_PLAY:
		return "play"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_PAUSE:
		return "pause"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_STEP:
		return "step"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_SET_SPEED:
		return "set_speed"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_RESET:
		return "reset"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_REWIND:
		return ""
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_RESUME:
		return "resume"
	default:
		return ""
	}
}

// toProtoInteractionActions 将内部交互动作定义转换为协议结构。
func toProtoInteractionActions(actions []scene.InteractionAction) []*simscenariov1.InteractionAction {
	result := make([]*simscenariov1.InteractionAction, 0, len(actions))
	for _, action := range actions {
		fields := make([]*simscenariov1.InteractionField, 0, len(action.Fields))
		for _, field := range action.Fields {
			options := make([]*simscenariov1.InteractionOption, 0, len(field.Options))
			for _, option := range field.Options {
				options = append(options, &simscenariov1.InteractionOption{
					Value: option.Value,
					Label: option.Label,
				})
			}
			fields = append(fields, &simscenariov1.InteractionField{
				Key:            field.Key,
				Label:          field.Label,
				Type:           toProtoFieldType(field.Type),
				Required:       field.Required,
				DefaultValue:   field.DefaultValue,
				Options:        options,
				ValidationJson: field.ValidationJSON,
			})
		}
		result = append(result, &simscenariov1.InteractionAction{
			ActionCode:   action.ActionCode,
			Label:        action.Label,
			Description:  action.Description,
			Trigger:      toProtoTrigger(action.Trigger),
			Fields:       fields,
			UiSchemaJson: action.UISchemaJSON,
		})
	}
	return result
}

// toProtoFieldType 将内部字段类型编码转换为协议枚举。
func toProtoFieldType(fieldType string) simscenariov1.InteractionFieldType {
	switch fieldType {
	case "string":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_STRING
	case "number":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NUMBER
	case "boolean":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_BOOLEAN
	case "select":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_SELECT
	case "node_ref":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NODE_REF
	case "range":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_RANGE
	case "json":
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_JSON
	default:
		return simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_UNSPECIFIED
	}
}

// toProtoTrigger 将内部触发器编码转换为协议枚举。
func toProtoTrigger(trigger string) simscenariov1.InteractionTrigger {
	switch trigger {
	case "click":
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CLICK
	case "form_submit":
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_FORM_SUBMIT
	case "drag":
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_DRAG
	case "canvas_select":
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CANVAS_SELECT
	default:
		return simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_UNSPECIFIED
	}
}
