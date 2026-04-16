package scene

import (
	"context"
	"errors"
	"fmt"
	"strings"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewGRPCClientFactory 根据场景编码与远程过程调用地址映射创建客户端工厂。
func NewGRPCClientFactory(endpoints map[string]string) ClientFactory {
	copied := make(map[string]string, len(endpoints))
	for sceneCode, endpoint := range endpoints {
		copied[strings.TrimSpace(sceneCode)] = strings.TrimSpace(endpoint)
	}

	return func(config Config) (ScenarioClient, error) {
		endpoint := copied[strings.TrimSpace(config.SceneCode)]
		if endpoint == "" {
			return nil, fmt.Errorf("scene endpoint is not configured: %s", config.SceneCode)
		}

		conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}

		return &grpcScenarioClient{
			conn:   conn,
			client: simscenariov1.NewSimScenarioServiceClient(conn),
		}, nil
	}
}

// ParseEndpointsConfig 解析环境变量中的场景地址配置。
func ParseEndpointsConfig(raw string) (map[string]string, error) {
	result := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return result, errors.New("scene endpoints config is required")
	}

	items := strings.Split(raw, ",")
	for _, item := range items {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid scene endpoint config item: %s", item)
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result, nil
}

// grpcScenarioClient 负责将 Core 的场景调用转发到远端场景容器。
type grpcScenarioClient struct {
	conn   *grpc.ClientConn
	client simscenariov1.SimScenarioServiceClient
}

// Close 释放场景容器 gRPC 连接。
func (c *grpcScenarioClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Meta 读取远端场景容器上报的元信息。
func (c *grpcScenarioClient) Meta(ctx context.Context) (Meta, error) {
	response, err := c.client.GetMeta(ctx, &simscenariov1.GetMetaRequest{})
	if err != nil {
		return Meta{}, err
	}

	meta := response.GetMeta()
	return Meta{
		Code:                    meta.GetCode(),
		Name:                    meta.GetName(),
		Category:                strings.TrimSpace(meta.GetCategory()),
		AlgorithmType:           meta.GetAlgorithmType(),
		Description:             meta.GetDescription(),
		Version:                 meta.GetVersion(),
		TimeControlMode:         timeControlModeString(meta.GetTimeControlMode()),
		DataSourceMode:          dataSourceModeString(meta.GetDataSourceMode()),
		DefaultParamsJSON:       cloneBytes(meta.GetDefaultParamsJson()),
		DefaultStateJSON:        cloneBytes(meta.GetDefaultStateJson()),
		SupportedLinkGroupCodes: append([]string(nil), meta.GetSupportedLinkGroupCodes()...),
	}, nil
}

// Init 调用远端场景容器完成初始化。
func (c *grpcScenarioClient) Init(ctx context.Context, req InitRequest) (State, error) {
	response, err := c.client.Init(ctx, &simscenariov1.InitRequest{
		SessionId:        req.SessionID,
		SceneCode:        req.SceneCode,
		ParamsJson:       req.ParamsJSON,
		InitialStateJson: req.InitialStateJSON,
		SharedStateJson:  req.SharedStateJSON,
	})
	if err != nil {
		return State{}, err
	}

	return State{
		SceneCode:       response.GetSceneCode(),
		Tick:            response.GetTick(),
		StateJSON:       response.GetStateJson(),
		RenderStateJSON: response.GetRenderStateJson(),
		SharedStateJSON: response.GetSharedStateJson(),
	}, nil
}

// Step 调用远端场景容器推进一个仿真时钟步。
func (c *grpcScenarioClient) Step(ctx context.Context, req StepRequest) (StepResult, error) {
	response, err := c.client.Step(ctx, &simscenariov1.StepRequest{
		SessionId:       req.SessionID,
		SceneCode:       req.SceneCode,
		StateJson:       req.StateJSON,
		Tick:            req.Tick,
		SharedStateJson: req.SharedStateJSON,
	})
	if err != nil {
		return StepResult{}, err
	}

	return StepResult{
		SceneCode:           response.GetSceneCode(),
		Tick:                response.GetTick(),
		StateJSON:           response.GetStateJson(),
		RenderStateJSON:     response.GetRenderStateJson(),
		Events:              toSceneEvents(response.GetEvents()),
		SharedStateDiffJSON: response.GetSharedStateDiffJson(),
	}, nil
}

// HandleAction 将交互请求转发给远端场景容器处理。
func (c *grpcScenarioClient) HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	response, err := c.client.HandleAction(ctx, &simscenariov1.HandleActionRequest{
		SessionId:       req.SessionID,
		SceneCode:       req.SceneCode,
		StateJson:       req.StateJSON,
		ActionCode:      req.ActionCode,
		ParamsJson:      req.ParamsJSON,
		Tick:            req.Tick,
		SharedStateJson: req.SharedStateJSON,
		ActorId:         req.ActorID,
		RoleKey:         req.RoleKey,
	})
	if err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		Tick:                response.GetTick(),
		Success:             response.GetSuccess(),
		ErrorMessage:        response.GetErrorMessage(),
		StateJSON:           response.GetStateJson(),
		RenderStateJSON:     response.GetRenderStateJson(),
		Events:              toSceneEvents(response.GetEvents()),
		SharedStateDiffJSON: response.GetSharedStateDiffJson(),
	}, nil
}

// RenderState 获取场景当前可渲染状态。
func (c *grpcScenarioClient) RenderState(ctx context.Context, sessionID string, sceneCode string, stateJSON []byte, tick int64, sharedStateJSON []byte) (RenderState, error) {
	response, err := c.client.GetRenderState(ctx, &simscenariov1.GetRenderStateRequest{
		SessionId:       sessionID,
		SceneCode:       sceneCode,
		StateJson:       stateJSON,
		Tick:            tick,
		SharedStateJson: sharedStateJSON,
	})
	if err != nil {
		return RenderState{}, err
	}
	category, err := resolveRenderStateCategory(response.GetCategory())
	if err != nil {
		return RenderState{}, err
	}

	return RenderState{
		SceneCode:       response.GetSceneCode(),
		Category:        category,
		AlgorithmType:   response.GetAlgorithmType(),
		Tick:            response.GetTick(),
		StateJSON:       response.GetStateJson(),
		RenderStateJSON: response.GetRenderStateJson(),
		MetricsJSON:     response.GetMetricsJson(),
		Events:          toSceneEvents(response.GetEvents()),
	}, nil
}

// resolveRenderStateCategory 要求运行态协议显式返回字符串领域编码，避免继续保留旧的枚举或别名链路。
func resolveRenderStateCategory(category string) (string, error) {
	normalized := strings.TrimSpace(category)
	if normalized == "" {
		return "", fmt.Errorf("render state category is required")
	}
	return normalized, nil
}

// Health 查询场景容器健康状态。
func (c *grpcScenarioClient) Health(ctx context.Context) (HealthStatus, error) {
	response, err := c.client.HealthCheck(ctx, &simscenariov1.HealthCheckRequest{})
	if err != nil {
		return HealthStatus{}, err
	}
	return HealthStatus{
		Status:      healthStatusString(response.GetStatus()),
		Message:     response.GetMessage(),
		CheckedAtMS: response.GetCheckedAtMs(),
	}, nil
}

// InteractionSchema 获取场景专属交互面板定义。
func (c *grpcScenarioClient) InteractionSchema(ctx context.Context) (InteractionSchema, error) {
	response, err := c.client.GetInteractionSchema(ctx, &simscenariov1.GetInteractionSchemaRequest{})
	if err != nil {
		return InteractionSchema{}, err
	}

	actions := make([]InteractionAction, 0, len(response.GetActions()))
	for _, action := range response.GetActions() {
		fields := make([]InteractionField, 0, len(action.GetFields()))
		for _, field := range action.GetFields() {
			options := make([]InteractionFieldOption, 0, len(field.GetOptions()))
			for _, option := range field.GetOptions() {
				options = append(options, InteractionFieldOption{
					Value: option.GetValue(),
					Label: option.GetLabel(),
				})
			}
			fields = append(fields, InteractionField{
				Key:            field.GetKey(),
				Label:          field.GetLabel(),
				Type:           interactionFieldTypeString(field.GetType()),
				Required:       field.GetRequired(),
				DefaultValue:   field.GetDefaultValue(),
				Options:        options,
				ValidationJSON: field.GetValidationJson(),
			})
		}
		actions = append(actions, InteractionAction{
			ActionCode:   action.GetActionCode(),
			Label:        action.GetLabel(),
			Description:  action.GetDescription(),
			Trigger:      interactionTriggerString(action.GetTrigger()),
			Fields:       fields,
			UISchemaJSON: action.GetUiSchemaJson(),
		})
	}

	return InteractionSchema{
		SceneCode: response.GetSceneCode(),
		Actions:   actions,
	}, nil
}

// timeControlModeString 将协议时间模式转换为内部字符串编码。
func timeControlModeString(mode simscenariov1.TimeControlMode) string {
	switch mode {
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_PROCESS:
		return "process"
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_REACTIVE:
		return "reactive"
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_CONTINUOUS:
		return "continuous"
	default:
		return ""
	}
}

// dataSourceModeString 将协议数据源模式转换为内部字符串编码。
func dataSourceModeString(mode simscenariov1.DataSourceMode) string {
	switch mode {
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION:
		return "simulation"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION:
		return "collection"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL:
		return "dual"
	default:
		return ""
	}
}

// interactionFieldTypeString 将协议字段类型转换为字符串编码。
func interactionFieldTypeString(fieldType simscenariov1.InteractionFieldType) string {
	switch fieldType {
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_STRING:
		return "string"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NUMBER:
		return "number"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_BOOLEAN:
		return "boolean"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_SELECT:
		return "select"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_NODE_REF:
		return "node_ref"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_RANGE:
		return "range"
	case simscenariov1.InteractionFieldType_INTERACTION_FIELD_TYPE_JSON:
		return "json"
	default:
		return ""
	}
}

// interactionTriggerString 将协议触发器转换为字符串编码。
func interactionTriggerString(trigger simscenariov1.InteractionTrigger) string {
	switch trigger {
	case simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CLICK:
		return "click"
	case simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_FORM_SUBMIT:
		return "form_submit"
	case simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_DRAG:
		return "drag"
	case simscenariov1.InteractionTrigger_INTERACTION_TRIGGER_CANVAS_SELECT:
		return "canvas_select"
	default:
		return ""
	}
}

// healthStatusString 将协议健康状态转换为内部字符串编码。
func healthStatusString(status simscenariov1.HealthStatus) string {
	switch status {
	case simscenariov1.HealthStatus_HEALTH_STATUS_SERVING:
		return "serving"
	case simscenariov1.HealthStatus_HEALTH_STATUS_NOT_SERVING:
		return "not_serving"
	case simscenariov1.HealthStatus_HEALTH_STATUS_STARTING:
		return "starting"
	default:
		return ""
	}
}

// toSceneEvents 将协议事件集合转换为 Core 内部事件结构。
func toSceneEvents(events []*simscenariov1.SimEvent) []Event {
	result := make([]Event, 0, len(events))
	for _, event := range events {
		result = append(result, Event{
			EventID:     event.GetEventId(),
			EventType:   event.GetEventType(),
			SceneCode:   event.GetSceneCode(),
			Tick:        event.GetTick(),
			TimestampMS: event.GetTimestampMs(),
			PayloadJSON: event.GetPayloadJson(),
		})
	}
	return result
}
