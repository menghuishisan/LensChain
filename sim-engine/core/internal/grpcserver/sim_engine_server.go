// 模块：sim-engine/core/internal/grpcserver
// 文件职责：将 SimEngine Core 内部 app.Engine 适配为 SimEngineService gRPC 服务端。
// 协议依据：proto/lenschain/sim_engine/v1/sim_engine.proto；
//          docs/modules/04-实验环境/03-API接口设计.md §四 / 06-可视化仿真引擎设计.md §6 / §7。

package grpcserver

import (
	"context"
	"encoding/json"
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

// =====================================================================
// 会话生命周期
// =====================================================================

// CreateSession 创建仿真会话。
func (s *SimEngineServer) CreateSession(ctx context.Context, req *simenginev1.CreateSessionRequest) (*simenginev1.CreateSessionResponse, error) {
	scenes := make([]app.SceneConfig, 0, len(req.GetScenes()))
	for _, item := range req.GetScenes() {
		scenes = append(scenes, app.SceneConfig{
			SceneCode:             item.GetSceneCode(),
			LinkGroupCode:         item.GetLinkGroupCode(),
			ParamsJSON:            item.GetParamsJson(),
			DataSourceConfigJSON:  item.GetDataSourceConfigJson(),
			DataSourceMode:        dataSourceModeString(item.GetDataSourceMode()),
			ContainerImageURL:     item.GetContainerImageUrl(),
			ResourceRequestCPU:    item.GetResourceRequestCpu(),
			ResourceRequestMemory: item.GetResourceRequestMemory(),
		})
	}

	// LinkGroups 元数据由 backend 通过 session_config_json 透传：
	//
	//   {
	//     "link_groups": [{
	//       "code": "pbft-attack-group",
	//       "version": "1.0.0",
	//       "members": ["pbft-consensus", "pbft-byzantine", "network-partition"],
	//       "fields": [{"name":"view_number","type":"int","owner":"pbft-consensus"}, ...],
	//       "force_clock_sync": true,
	//       "initial_state": {"view_number": 0, "byzantine_nodes": [], "partition_active": false}
	//     }],
	//     "collaboration": { ... }
	//   }
	//
	// 与 collaboration policy 共享同一份 session_config_json。
	linkGroups, err := parseLinkGroupsFromConfig(req.GetSessionConfigJson())
	if err != nil {
		return nil, err
	}

	started, err := s.engine.StartSession(ctx, app.StartSessionRequest{
		InstanceID:        req.GetInstanceId(),
		StudentID:         req.GetStudentId(),
		LinkageEnabled:    req.GetLinkageEnabled(),
		SessionConfigJSON: req.GetSessionConfigJson(),
		Scenes:            scenes,
		LinkGroups:        linkGroups,
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

// DestroySession 销毁会话。
func (s *SimEngineServer) DestroySession(_ context.Context, req *simenginev1.DestroySessionRequest) (*simenginev1.DestroySessionResponse, error) {
	if err := s.engine.DestroySession(req.GetSessionId()); err != nil {
		return &simenginev1.DestroySessionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.DestroySessionResponse{Success: true}, nil
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

// =====================================================================
// 时间控制
// =====================================================================

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

// =====================================================================
// 交互
// =====================================================================

// SendInteraction 处理交互。
func (s *SimEngineServer) SendInteraction(ctx context.Context, req *simenginev1.SendInteractionRequest) (*simenginev1.SendInteractionResponse, error) {
	result, err := s.engine.SendInteraction(
		ctx,
		req.GetSessionId(),
		req.GetSceneCode(),
		req.GetActionCode(),
		req.GetParamsJson(),
		app.InteractionContext{
			ActorID:  req.GetActorId(),
			UserRole: req.GetUserRole(),
		},
	)
	if err != nil {
		return &simenginev1.SendInteractionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	return &simenginev1.SendInteractionResponse{
		Success:            result.Success,
		ErrorMessage:       result.ErrorMessage,
		RenderEnvelopeJson: result.RenderEnvelopeJSON,
	}, nil
}

// GetInteractionSchema 返回场景专属交互面板定义（含 schema_version）。
func (s *SimEngineServer) GetInteractionSchema(ctx context.Context, req *simenginev1.GetInteractionSchemaRequest) (*simenginev1.GetInteractionSchemaResponse, error) {
	schema, err := s.engine.GetInteractionSchema(ctx, req.GetSessionId(), req.GetSceneCode())
	if err != nil {
		return nil, err
	}
	return &simenginev1.GetInteractionSchemaResponse{
		Definition: &simscenariov1.InteractionDefinition{
			SceneCode:     schema.SceneCode,
			SchemaVersion: schema.SchemaVersion,
			Actions:       toProtoActions(schema.Actions),
		},
	}, nil
}

// =====================================================================
// 快照
// =====================================================================

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

// =====================================================================
// 数据采集
// =====================================================================

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

// =====================================================================
// 教师干预（完整实现 — 对齐 06.md §14.5 八种 action_code）
// =====================================================================

// PublishTeacherIntervention 接收 backend 转发的教师干预指令。
//
// 完整支持 8 种 action_code（详 06.md §14.5 教师跨学生干预 action）：
//   - broadcast_message: 向全班推送文字提示
//   - pause_all:         暂停全班学生所有场景
//   - resume_all:        恢复全班学生所有场景
//   - force_step:        强制推进指定学生的指定场景一个 tick
//   - force_reset:       强制重做指定学生的指定场景
//   - force_link_state:  强制覆写指定 LinkGroup 共享状态字段
//   - unlock_link_sync:  临时解除指定 LinkGroup 的时钟强制同步
//   - kick_student:      踢出违规学生（销毁会话）
func (s *SimEngineServer) PublishTeacherIntervention(ctx context.Context, req *simenginev1.PublishTeacherInterventionRequest) (*simenginev1.PublishTeacherInterventionResponse, error) {
	affected := make([]string, 0, len(req.GetTargetSessionIds()))
	switch req.GetActionCode() {
	case "pause_all":
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.ControlTime(sid, "pause", nil); err == nil {
				affected = append(affected, sid)
			}
		}
	case "resume_all":
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.ControlTime(sid, "resume", nil); err == nil {
				affected = append(affected, sid)
			}
		}
	case "force_reset":
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.ControlTime(sid, "reset", nil); err == nil {
				affected = append(affected, sid)
			}
		}
	case "broadcast_message":
		message := extractTeacherMessage(req.GetParamsJson())
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.TeacherBroadcastMessage(sid, message); err == nil {
				affected = append(affected, sid)
			}
		}
	case "force_step":
		sceneCode := extractTeacherSceneCode(req.GetParamsJson())
		if sceneCode == "" {
			return &simenginev1.PublishTeacherInterventionResponse{
				Success:      false,
				ErrorMessage: "force_step requires params_json.scene_code",
			}, nil
		}
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.TeacherForceStep(ctx, sid, sceneCode); err == nil {
				affected = append(affected, sid)
			}
		}
	case "force_link_state":
		linkGroupCode := extractTeacherLinkGroupCode(req.GetParamsJson())
		fieldsJSON := extractTeacherFieldsJSON(req.GetParamsJson())
		if linkGroupCode == "" || len(fieldsJSON) == 0 {
			return &simenginev1.PublishTeacherInterventionResponse{
				Success:      false,
				ErrorMessage: "force_link_state requires params_json.link_group_code and params_json.fields",
			}, nil
		}
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.ForceSetLinkState(sid, linkGroupCode, fieldsJSON); err == nil {
				affected = append(affected, sid)
			}
		}
	case "unlock_link_sync":
		linkGroupCode := extractTeacherLinkGroupCode(req.GetParamsJson())
		if linkGroupCode == "" {
			return &simenginev1.PublishTeacherInterventionResponse{
				Success:      false,
				ErrorMessage: "unlock_link_sync requires params_json.link_group_code",
			}, nil
		}
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.UnlockLinkSync(sid, linkGroupCode); err == nil {
				affected = append(affected, sid)
			}
		}
	case "kick_student":
		reason := extractTeacherReason(req.GetParamsJson())
		for _, sid := range req.GetTargetSessionIds() {
			if err := s.engine.TeacherKickStudent(sid, reason); err == nil {
				affected = append(affected, sid)
			}
		}
	default:
		return &simenginev1.PublishTeacherInterventionResponse{
			Success:      false,
			ErrorMessage: "unsupported teacher intervention action: " + req.GetActionCode(),
		}, nil
	}
	return &simenginev1.PublishTeacherInterventionResponse{
		Success:            true,
		AffectedSessionIds: affected,
	}, nil
}

// =====================================================================
// 教师干预参数提取
// =====================================================================

// extractTeacherMessage 从教师干预参数中提取消息文本。
func extractTeacherMessage(paramsJSON []byte) string {
	if len(paramsJSON) == 0 {
		return ""
	}
	var p struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return ""
	}
	return strings.TrimSpace(p.Message)
}

// extractTeacherSceneCode 从教师干预参数中提取目标场景编码。
func extractTeacherSceneCode(paramsJSON []byte) string {
	if len(paramsJSON) == 0 {
		return ""
	}
	var p struct {
		SceneCode string `json:"scene_code"`
	}
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return ""
	}
	return strings.TrimSpace(p.SceneCode)
}

// extractTeacherLinkGroupCode 从教师干预参数中提取联动组编码。
func extractTeacherLinkGroupCode(paramsJSON []byte) string {
	if len(paramsJSON) == 0 {
		return ""
	}
	var p struct {
		LinkGroupCode string `json:"link_group_code"`
	}
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return ""
	}
	return strings.TrimSpace(p.LinkGroupCode)
}

// extractTeacherFieldsJSON 从教师干预参数中提取要覆写的字段 JSON。
func extractTeacherFieldsJSON(paramsJSON []byte) []byte {
	if len(paramsJSON) == 0 {
		return nil
	}
	var p struct {
		Fields json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return nil
	}
	if len(p.Fields) == 0 || string(p.Fields) == "null" {
		return nil
	}
	return p.Fields
}

// extractTeacherReason 从教师干预参数中提取踢出原因。
func extractTeacherReason(paramsJSON []byte) string {
	if len(paramsJSON) == 0 {
		return ""
	}
	var p struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return ""
	}
	return strings.TrimSpace(p.Reason)
}

// =====================================================================
// 内部工具
// =====================================================================

// websocketURL 构造会话数据通道地址。
func (s *SimEngineServer) websocketURL(sessionID string) string {
	return s.publicBase + "/api/v1/ws/sim-engine/" + sessionID
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
	}
	return ""
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
	}
	return simenginev1.SessionStatus_SESSION_STATUS_RUNNING
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
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_RESUME:
		return "resume"
	case simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_STEP_BACK:
		return "step_back"
	}
	return ""
}

// toProtoActions 将 ActionDef 列表转换为 proto 结构（与新协议字段全字段对齐）。
func toProtoActions(actions []scene.ActionDef) []*simscenariov1.ActionDef {
	result := make([]*simscenariov1.ActionDef, 0, len(actions))
	for _, action := range actions {
		fields := make([]*simscenariov1.FieldDef, 0, len(action.Fields))
		for _, field := range action.Fields {
			fields = append(fields, &simscenariov1.FieldDef{
				Name:        field.Name,
				Type:        toProtoFieldType(field.Type),
				Label:       field.Label,
				Required:    field.Required,
				DefaultJson: field.DefaultJSON,
				MinJson:     field.MinJSON,
				MaxJson:     field.MaxJSON,
				StepJson:    field.StepJSON,
				OptionsJson: field.OptionsJSON,
				OptionsFrom: field.OptionsFrom,
			})
		}
		result = append(result, &simscenariov1.ActionDef{
			ActionCode:         action.ActionCode,
			Label:              action.Label,
			Description:        action.Description,
			Category:           toProtoActionCategory(action.Category),
			Trigger:            toProtoActionTrigger(action.Trigger),
			Fields:             fields,
			Roles:              append([]string(nil), action.Roles...),
			CooldownMs:         int32(action.CooldownMs),
			WritesOwnedFields: append([]string(nil), action.WritesOwnedFields...),
			LinkOwnerFields:    append([]string(nil), action.LinkOwnerFields...),
			HybridChannel:      toProtoHybridChannel(action.HybridChannel),
			ContainerCmd:       action.ContainerCmd,
		})
	}
	return result
}

// toProtoFieldType 将 FieldDef.Type 字符串转换为 proto 枚举（含新增 enum / multi_select）。
func toProtoFieldType(fieldType string) simscenariov1.FieldType {
	switch fieldType {
	case "string":
		return simscenariov1.FieldType_FIELD_TYPE_STRING
	case "number":
		return simscenariov1.FieldType_FIELD_TYPE_NUMBER
	case "boolean":
		return simscenariov1.FieldType_FIELD_TYPE_BOOLEAN
	case "select":
		return simscenariov1.FieldType_FIELD_TYPE_SELECT
	case "enum":
		return simscenariov1.FieldType_FIELD_TYPE_ENUM
	case "range":
		return simscenariov1.FieldType_FIELD_TYPE_RANGE
	case "json":
		return simscenariov1.FieldType_FIELD_TYPE_JSON
	case "multi_select":
		return simscenariov1.FieldType_FIELD_TYPE_MULTI_SELECT
	}
	return simscenariov1.FieldType_FIELD_TYPE_UNSPECIFIED
}

// toProtoActionCategory 将 ActionDef.Category 字符串转换为 proto 枚举。
func toProtoActionCategory(category string) simscenariov1.ActionCategory {
	switch category {
	case "param_tune":
		return simscenariov1.ActionCategory_ACTION_CATEGORY_PARAM_TUNE
	case "attack_inject":
		return simscenariov1.ActionCategory_ACTION_CATEGORY_ATTACK_INJECT
	case "primary":
		return simscenariov1.ActionCategory_ACTION_CATEGORY_PRIMARY
	case "observe":
		return simscenariov1.ActionCategory_ACTION_CATEGORY_OBSERVE
	}
	return simscenariov1.ActionCategory_ACTION_CATEGORY_UNSPECIFIED
}

// toProtoActionTrigger 将 ActionDef.Trigger 字符串转换为 proto 枚举。
func toProtoActionTrigger(trigger string) simscenariov1.ActionTrigger {
	switch trigger {
	case "submit":
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_SUBMIT
	case "immediate":
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_IMMEDIATE
	case "hold":
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_HOLD
	}
	return simscenariov1.ActionTrigger_ACTION_TRIGGER_UNSPECIFIED
}

// toProtoHybridChannel 将 ActionDef.HybridChannel 字符串转换为 proto 枚举。
func toProtoHybridChannel(channel string) simscenariov1.HybridChannel {
	switch channel {
	case "sim":
		return simscenariov1.HybridChannel_HYBRID_CHANNEL_SIM
	case "container":
		return simscenariov1.HybridChannel_HYBRID_CHANNEL_CONTAINER
	}
	return simscenariov1.HybridChannel_HYBRID_CHANNEL_UNSPECIFIED
}

// =====================================================================
// session_config_json → app.LinkGroupSpec 解析
// =====================================================================

// linkGroupsConfig 描述 session_config_json 中 link_groups 入口。
type linkGroupsConfig struct {
	LinkGroups []linkGroupConfigItem `json:"link_groups"`
}

// linkGroupConfigItem 描述单个联动组的元数据（含 schema 字段与初始状态）。
type linkGroupConfigItem struct {
	Code           string                  `json:"code"`
	Version        string                  `json:"version"`
	Members        []string                `json:"members"`
	Fields         []linkFieldConfigItem   `json:"fields"`
	ForceClockSync bool                    `json:"force_clock_sync"`
	InitialState   map[string]any          `json:"initial_state"`
}

// linkFieldConfigItem 描述单个 SharedState 字段的 owner 声明。
type linkFieldConfigItem struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Owner string `json:"owner"`
}

// parseLinkGroupsFromConfig 从 session_config_json 中提取联动组定义。
//
// 缺省（无 link_groups 字段）返回 nil；JSON 不合法时返回错误。
func parseLinkGroupsFromConfig(configJSON []byte) ([]app.LinkGroupSpec, error) {
	if len(configJSON) == 0 {
		return nil, nil
	}
	var parsed linkGroupsConfig
	if err := json.Unmarshal(configJSON, &parsed); err != nil {
		return nil, err
	}
	result := make([]app.LinkGroupSpec, 0, len(parsed.LinkGroups))
	for _, item := range parsed.LinkGroups {
		fields := make([]app.LinkFieldSpec, 0, len(item.Fields))
		for _, field := range item.Fields {
			fields = append(fields, app.LinkFieldSpec{
				Name:  field.Name,
				Type:  field.Type,
				Owner: field.Owner,
			})
		}
		result = append(result, app.LinkGroupSpec{
			Code:           item.Code,
			Version:        item.Version,
			Members:        append([]string(nil), item.Members...),
			Fields:         fields,
			ForceClockSync: item.ForceClockSync,
			InitialState:   item.InitialState,
		})
	}
	return result, nil
}
