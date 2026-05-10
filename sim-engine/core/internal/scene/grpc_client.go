// 模块：sim-engine/core/internal/scene
// 文件职责：将 ScenarioClient 接口适配到 gRPC SimScenarioService 远端调用。
// 协议依据：proto/lenschain/sim_scenario/v1/sim_scenario.proto。

package scene

import (
	"context"
	"fmt"
	"strings"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
	"google.golang.org/grpc"
)

// grpcScenarioClient 负责将 Core 调用转发到远端场景容器。
// 实例由 K8sOrchestrator 在按需启动 Pod 后构造并注入连接。
type grpcScenarioClient struct {
	conn   *grpc.ClientConn
	client simscenariov1.SimScenarioServiceClient
}

// Close 释放 gRPC 连接（仅在编排器允许时调用，连接归编排器持有）。
func (c *grpcScenarioClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Meta 读取远端场景容器上报的元信息。
func (c *grpcScenarioClient) Meta(ctx context.Context) (Meta, error) {
	resp, err := c.client.GetMeta(ctx, &simscenariov1.GetMetaRequest{})
	if err != nil {
		return Meta{}, err
	}
	meta := resp.GetMeta()
	if meta == nil {
		return Meta{}, fmt.Errorf("场景容器未返回元信息")
	}
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
		CustomRendererPackage:   meta.GetCustomRendererPackage(),
		SupportedLinkGroupCodes: append([]string(nil), meta.GetSupportedLinkGroupCodes()...),
	}, nil
}

// InteractionSchema 获取场景对外暴露的全部 ActionDef。
func (c *grpcScenarioClient) InteractionSchema(ctx context.Context) (InteractionDefinition, error) {
	resp, err := c.client.GetInteractionSchema(ctx, &simscenariov1.GetInteractionSchemaRequest{})
	if err != nil {
		return InteractionDefinition{}, err
	}
	def := resp.GetDefinition()
	if def == nil {
		return InteractionDefinition{}, fmt.Errorf("场景容器未返回 InteractionDefinition")
	}

	actions := make([]ActionDef, 0, len(def.GetActions()))
	for _, action := range def.GetActions() {
		fields := make([]FieldDef, 0, len(action.GetFields()))
		for _, field := range action.GetFields() {
			fields = append(fields, FieldDef{
				Name:        field.GetName(),
				Type:        fieldTypeString(field.GetType()),
				Label:       field.GetLabel(),
				Required:    field.GetRequired(),
				DefaultJSON: cloneBytes(field.GetDefaultJson()),
				MinJSON:     cloneBytes(field.GetMinJson()),
				MaxJSON:     cloneBytes(field.GetMaxJson()),
				StepJSON:    cloneBytes(field.GetStepJson()),
				OptionsJSON: cloneBytes(field.GetOptionsJson()),
				OptionsFrom: field.GetOptionsFrom(),
			})
		}
		actions = append(actions, ActionDef{
			ActionCode:         action.GetActionCode(),
			Label:              action.GetLabel(),
			Description:        action.GetDescription(),
			Category:           actionCategoryString(action.GetCategory()),
			Trigger:            actionTriggerString(action.GetTrigger()),
			Fields:             fields,
			Roles:              append([]string(nil), action.GetRoles()...),
			CooldownMs:         int(action.GetCooldownMs()),
			WritesOwnedFields: append([]string(nil), action.GetWritesOwnedFields()...),
			LinkOwnerFields:    append([]string(nil), action.GetLinkOwnerFields()...),
			HybridChannel:      hybridChannelString(action.GetHybridChannel()),
			ContainerCmd:       action.GetContainerCmd(),
		})
	}
	return InteractionDefinition{
		SceneCode:     def.GetSceneCode(),
		SchemaVersion: def.GetSchemaVersion(),
		Actions:       actions,
	}, nil
}

// Init 调用远端场景容器初始化场景。
func (c *grpcScenarioClient) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	resp, err := c.client.Init(ctx, &simscenariov1.InitRequest{
		SessionId:       req.SessionID,
		SceneCode:       req.SceneCode,
		InstanceId:      req.InstanceID,
		StudentId:       req.StudentID,
		Seed:            req.Seed,
		ParamsJson:      cloneBytes(req.ParamsJSON),
		SharedStateJson: cloneBytes(req.SharedStateJSON),
	})
	if err != nil {
		return InitResult{}, err
	}
	return InitResult{
		SceneCode:           resp.GetSceneCode(),
		Tick:                resp.GetTick(),
		SceneStateJSON:      cloneBytes(resp.GetSceneStateJson()),
		RenderEnvelopeJSON:  cloneBytes(resp.GetRenderEnvelopeJson()),
		SharedStateDiffJSON: cloneBytes(resp.GetSharedStateDiffJson()),
	}, nil
}

// Step 调用远端场景容器推进一个 tick。
func (c *grpcScenarioClient) Step(ctx context.Context, req StepRequest) (StepResult, error) {
	resp, err := c.client.Step(ctx, &simscenariov1.StepRequest{
		SessionId:             req.SessionID,
		SceneCode:             req.SceneCode,
		Tick:                  req.Tick,
		SceneStateJson:        cloneBytes(req.SceneStateJSON),
		SharedStateJson:       cloneBytes(req.SharedStateJSON),
		IncomingLinkTriggers:  toProtoTriggers(req.IncomingLinkTriggers),
	})
	if err != nil {
		return StepResult{}, err
	}
	return StepResult{
		SceneCode:           resp.GetSceneCode(),
		Tick:                resp.GetTick(),
		SceneStateJSON:      cloneBytes(resp.GetSceneStateJson()),
		RenderEnvelopeJSON:  cloneBytes(resp.GetRenderEnvelopeJson()),
		SharedStateDiffJSON: cloneBytes(resp.GetSharedStateDiffJson()),
	}, nil
}

// HandleAction 转发交互请求给场景容器。
func (c *grpcScenarioClient) HandleAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	resp, err := c.client.HandleAction(ctx, &simscenariov1.HandleActionRequest{
		SessionId:       req.SessionID,
		SceneCode:       req.SceneCode,
		Tick:            req.Tick,
		SceneStateJson:  cloneBytes(req.SceneStateJSON),
		SharedStateJson: cloneBytes(req.SharedStateJSON),
		ActionCode:      req.ActionCode,
		ParamsJson:      cloneBytes(req.ParamsJSON),
		ActorId:         req.ActorID,
		UserRole:        req.UserRole,
	})
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		SceneCode:           resp.GetSceneCode(),
		Tick:                resp.GetTick(),
		Success:             resp.GetSuccess(),
		ErrorMessage:        resp.GetErrorMessage(),
		SceneStateJSON:      cloneBytes(resp.GetSceneStateJson()),
		RenderEnvelopeJSON:  cloneBytes(resp.GetRenderEnvelopeJson()),
		SharedStateDiffJSON: cloneBytes(resp.GetSharedStateDiffJson()),
	}, nil
}

// Health 查询场景容器健康状态。
func (c *grpcScenarioClient) Health(ctx context.Context) (HealthStatus, error) {
	resp, err := c.client.HealthCheck(ctx, &simscenariov1.HealthCheckRequest{})
	if err != nil {
		return HealthStatus{}, err
	}
	return HealthStatus{
		Status:      healthStatusString(resp.GetStatus()),
		Message:     resp.GetMessage(),
		CheckedAtMS: resp.GetCheckedAtMs(),
	}, nil
}

// =====================================================================
// proto ↔ scene 类型转换
// =====================================================================

func toProtoTriggers(items []LinkTriggerRef) []*simscenariov1.LinkTriggerRef {
	if len(items) == 0 {
		return nil
	}
	out := make([]*simscenariov1.LinkTriggerRef, 0, len(items))
	for _, item := range items {
		out = append(out, &simscenariov1.LinkTriggerRef{
			Id:             item.ID,
			SourceScene:    item.SourceScene,
			SourceAction:   item.SourceAction,
			LinkGroup:      item.LinkGroup,
			ChangedFields:  append([]string(nil), item.ChangedFields...),
			PayloadJson:    cloneBytes(item.PayloadJSON),
			TimestampMs:    item.TimestampMS,
			SourceAnchorId: item.SourceAnchorID,
			TargetAnchorId: item.TargetAnchorID,
		})
	}
	return out
}

func timeControlModeString(mode simscenariov1.TimeControlMode) string {
	switch mode {
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_PROCESS:
		return "process"
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_REACTIVE:
		return "reactive"
	case simscenariov1.TimeControlMode_TIME_CONTROL_MODE_CONTINUOUS:
		return "continuous"
	}
	return ""
}

func dataSourceModeString(mode simscenariov1.DataSourceMode) string {
	switch mode {
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION:
		return "simulation"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION:
		return "collection"
	case simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL:
		return "dual"
	}
	return ""
}

func actionCategoryString(c simscenariov1.ActionCategory) string {
	switch c {
	case simscenariov1.ActionCategory_ACTION_CATEGORY_PARAM_TUNE:
		return "param_tune"
	case simscenariov1.ActionCategory_ACTION_CATEGORY_ATTACK_INJECT:
		return "attack_inject"
	case simscenariov1.ActionCategory_ACTION_CATEGORY_PRIMARY:
		return "primary"
	case simscenariov1.ActionCategory_ACTION_CATEGORY_OBSERVE:
		return "observe"
	}
	return ""
}

func actionTriggerString(t simscenariov1.ActionTrigger) string {
	switch t {
	case simscenariov1.ActionTrigger_ACTION_TRIGGER_SUBMIT:
		return "submit"
	case simscenariov1.ActionTrigger_ACTION_TRIGGER_IMMEDIATE:
		return "immediate"
	case simscenariov1.ActionTrigger_ACTION_TRIGGER_HOLD:
		return "hold"
	}
	return ""
}

func fieldTypeString(t simscenariov1.FieldType) string {
	switch t {
	case simscenariov1.FieldType_FIELD_TYPE_STRING:
		return "string"
	case simscenariov1.FieldType_FIELD_TYPE_NUMBER:
		return "number"
	case simscenariov1.FieldType_FIELD_TYPE_BOOLEAN:
		return "boolean"
	case simscenariov1.FieldType_FIELD_TYPE_SELECT:
		return "select"
	case simscenariov1.FieldType_FIELD_TYPE_ENUM:
		return "enum"
	case simscenariov1.FieldType_FIELD_TYPE_RANGE:
		return "range"
	case simscenariov1.FieldType_FIELD_TYPE_JSON:
		return "json"
	case simscenariov1.FieldType_FIELD_TYPE_MULTI_SELECT:
		return "multi_select"
	}
	return ""
}

func hybridChannelString(c simscenariov1.HybridChannel) string {
	switch c {
	case simscenariov1.HybridChannel_HYBRID_CHANNEL_SIM:
		return "sim"
	case simscenariov1.HybridChannel_HYBRID_CHANNEL_CONTAINER:
		return "container"
	}
	return ""
}

func healthStatusString(status simscenariov1.HealthStatus) string {
	switch status {
	case simscenariov1.HealthStatus_HEALTH_STATUS_SERVING:
		return "serving"
	case simscenariov1.HealthStatus_HEALTH_STATUS_NOT_SERVING:
		return "not_serving"
	case simscenariov1.HealthStatus_HEALTH_STATUS_STARTING:
		return "starting"
	}
	return ""
}
