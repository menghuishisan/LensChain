// 模块：sim-engine/sdk/go/scenario
// 文件职责：把 sdk.Scenario 接口适配为 gRPC SimScenarioService 服务端实现。
// 协议依据：proto/lenschain/sim_scenario/v1/sim_scenario.proto。
//
// 本文件只做 sdk 类型 ↔ proto 类型转换，不写任何场景算法、不持有运行时状态。

package scenario

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
)

// Server 把 sdk.Scenario 适配为 gRPC SimScenarioService 服务端。
type Server struct {
	simscenariov1.UnimplementedSimScenarioServiceServer
	scenario Scenario
}

// NewServer 创建 sdk.Scenario 的 gRPC 适配器。
func NewServer(s Scenario) *Server {
	return &Server{scenario: s}
}

// =====================================================================
// gRPC 服务方法（sdk → proto 适配）
// =====================================================================

// GetMeta 实现 gRPC GetMeta：调用 Scenario.Meta 并转换为 proto.ScenarioMeta。
func (s *Server) GetMeta(ctx context.Context, _ *simscenariov1.GetMetaRequest) (*simscenariov1.GetMetaResponse, error) {
	if s.scenario == nil {
		return nil, errors.New("scenario 未注入")
	}
	meta, err := s.scenario.Meta(ctx)
	if err != nil {
		return nil, err
	}
	if err := ValidateMeta(meta); err != nil {
		return nil, fmt.Errorf("场景元信息校验失败: %w", err)
	}
	return &simscenariov1.GetMetaResponse{Meta: toProtoMeta(meta)}, nil
}

// GetInteractionSchema 实现 gRPC GetInteractionSchema：调用 Scenario.InteractionSchema。
func (s *Server) GetInteractionSchema(ctx context.Context, _ *simscenariov1.GetInteractionSchemaRequest) (*simscenariov1.GetInteractionSchemaResponse, error) {
	def, err := s.scenario.InteractionSchema(ctx)
	if err != nil {
		return nil, err
	}
	if err := ValidateInteractionDefinition(def); err != nil {
		return nil, fmt.Errorf("交互定义校验失败: %w", err)
	}
	return &simscenariov1.GetInteractionSchemaResponse{Definition: toProtoInteractionDefinition(def)}, nil
}

// Init 实现 gRPC Init：调用 Scenario.Init 并返回首帧。
func (s *Server) Init(ctx context.Context, req *simscenariov1.InitRequest) (*simscenariov1.InitResponse, error) {
	result, err := s.scenario.Init(ctx, InitRequest{
		SessionID:       req.GetSessionId(),
		SceneCode:       req.GetSceneCode(),
		InstanceID:      req.GetInstanceId(),
		StudentID:       req.GetStudentId(),
		Seed:            req.GetSeed(),
		ParamsJSON:      req.GetParamsJson(),
		SharedStateJSON: req.GetSharedStateJson(),
	})
	if err != nil {
		return nil, err
	}
	return &simscenariov1.InitResponse{
		SceneCode:           req.GetSceneCode(),
		Tick:                result.Tick,
		SceneStateJson:      result.SceneStateJSON,
		RenderEnvelopeJson:  result.RenderEnvelopeJSON,
		SharedStateDiffJson: result.SharedStateDiffJSON,
	}, nil
}

// Step 实现 gRPC Step：推进单 tick。
func (s *Server) Step(ctx context.Context, req *simscenariov1.StepRequest) (*simscenariov1.StepResponse, error) {
	incoming := fromProtoLinkTriggers(req.GetIncomingLinkTriggers())
	metrics := fromProtoContainerMetrics(req.GetIncomingContainerMetrics())
	result, err := s.scenario.Step(ctx, StepRequest{
		SessionID:                req.GetSessionId(),
		SceneCode:                req.GetSceneCode(),
		Tick:                     req.GetTick(),
		SceneStateJSON:           req.GetSceneStateJson(),
		SharedStateJSON:          req.GetSharedStateJson(),
		IncomingLinkTriggers:     incoming,
		IncomingContainerMetrics: metrics,
	})
	if err != nil {
		return nil, err
	}
	return &simscenariov1.StepResponse{
		SceneCode:           req.GetSceneCode(),
		Tick:                result.Tick,
		SceneStateJson:      result.SceneStateJSON,
		RenderEnvelopeJson:  result.RenderEnvelopeJSON,
		SharedStateDiffJson: result.SharedStateDiffJSON,
	}, nil
}

// HandleAction 实现 gRPC HandleAction：处理交互。
func (s *Server) HandleAction(ctx context.Context, req *simscenariov1.HandleActionRequest) (*simscenariov1.HandleActionResponse, error) {
	result, err := s.scenario.HandleAction(ctx, ActionRequest{
		SessionID:       req.GetSessionId(),
		SceneCode:       req.GetSceneCode(),
		Tick:            req.GetTick(),
		SceneStateJSON:  req.GetSceneStateJson(),
		SharedStateJSON: req.GetSharedStateJson(),
		ActionCode:      req.GetActionCode(),
		ParamsJSON:      req.GetParamsJson(),
		ActorID:         req.GetActorId(),
		UserRole:        UserRole(req.GetUserRole()),
	})
	if err != nil {
		return nil, err
	}
	return &simscenariov1.HandleActionResponse{
		SceneCode:           req.GetSceneCode(),
		Tick:                result.Tick,
		Success:             result.Success,
		ErrorMessage:        result.ErrorMessage,
		SceneStateJson:      result.SceneStateJSON,
		RenderEnvelopeJson:  result.RenderEnvelopeJSON,
		SharedStateDiffJson: result.SharedStateDiffJSON,
	}, nil
}

// HealthCheck 实现 gRPC HealthCheck：默认返回 SERVING。
func (s *Server) HealthCheck(context.Context, *simscenariov1.HealthCheckRequest) (*simscenariov1.HealthCheckResponse, error) {
	return &simscenariov1.HealthCheckResponse{
		Status:      simscenariov1.HealthStatus_HEALTH_STATUS_SERVING,
		Message:     "serving",
		CheckedAtMs: time.Now().UTC().UnixMilli(),
	}, nil
}

// =====================================================================
// proto ↔ sdk 类型转换
// =====================================================================

func toProtoMeta(m Meta) *simscenariov1.ScenarioMeta {
	return &simscenariov1.ScenarioMeta{
		Code:                    m.Code,
		Name:                    m.Name,
		Category:                string(m.Category),
		AlgorithmType:           m.AlgorithmType,
		Description:             m.Description,
		Version:                 m.Version,
		TimeControlMode:         toProtoTimeControlMode(m.TimeControlMode),
		DataSourceMode:          toProtoDataSourceMode(m.DataSourceMode),
		DefaultParamsJson:       m.DefaultParams,
		DefaultStateJson:        m.DefaultState,
		CustomRendererPackage:   m.CustomRendererPackage,
		SupportedLinkGroupCodes: append([]string(nil), m.SupportedLinkGroupCodes...),

		// v0.5 新增（详 AGENTS.md §0.7.1 C10 / C29 / C37）。
		ExtensionLevel:     string(m.ExtensionLevel),
		LinkGroupVersion:   m.LinkGroupVersion,
		SupportsMultiActor: m.SupportsMultiActor,
		OwnedFieldPaths:    append([]string(nil), m.OwnedFieldPaths...),
	}
}

func toProtoTimeControlMode(m TimeControlMode) simscenariov1.TimeControlMode {
	switch m {
	case TimeControlProcess:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_PROCESS
	case TimeControlReactive:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_REACTIVE
	case TimeControlContinuous:
		return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_CONTINUOUS
	}
	return simscenariov1.TimeControlMode_TIME_CONTROL_MODE_UNSPECIFIED
}

func toProtoDataSourceMode(m DataSourceMode) simscenariov1.DataSourceMode {
	switch m {
	case DataSourceSimulation:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION
	case DataSourceCollection:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION
	case DataSourceDual:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL
	}
	return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_UNSPECIFIED
}

func toProtoActionCategory(c ActionCategory) simscenariov1.ActionCategory {
	switch c {
	case ActionParamTune:
		return simscenariov1.ActionCategory_ACTION_CATEGORY_PARAM_TUNE
	case ActionAttackInject:
		return simscenariov1.ActionCategory_ACTION_CATEGORY_ATTACK_INJECT
	case ActionPrimary:
		return simscenariov1.ActionCategory_ACTION_CATEGORY_PRIMARY
	case ActionObserve:
		return simscenariov1.ActionCategory_ACTION_CATEGORY_OBSERVE
	}
	return simscenariov1.ActionCategory_ACTION_CATEGORY_UNSPECIFIED
}

func toProtoActionTrigger(t ActionTrigger) simscenariov1.ActionTrigger {
	switch t {
	case TriggerSubmit:
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_SUBMIT
	case TriggerImmediate:
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_IMMEDIATE
	case TriggerHold:
		return simscenariov1.ActionTrigger_ACTION_TRIGGER_HOLD
	}
	return simscenariov1.ActionTrigger_ACTION_TRIGGER_UNSPECIFIED
}

func toProtoFieldType(t FieldType) simscenariov1.FieldType {
	switch t {
	case FieldString:
		return simscenariov1.FieldType_FIELD_TYPE_STRING
	case FieldNumber:
		return simscenariov1.FieldType_FIELD_TYPE_NUMBER
	case FieldBoolean:
		return simscenariov1.FieldType_FIELD_TYPE_BOOLEAN
	case FieldSelect:
		return simscenariov1.FieldType_FIELD_TYPE_SELECT
	case FieldEnum:
		return simscenariov1.FieldType_FIELD_TYPE_ENUM
	case FieldRange:
		return simscenariov1.FieldType_FIELD_TYPE_RANGE
	case FieldJSON:
		return simscenariov1.FieldType_FIELD_TYPE_JSON
	case FieldMultiSelect:
		return simscenariov1.FieldType_FIELD_TYPE_MULTI_SELECT
	}
	return simscenariov1.FieldType_FIELD_TYPE_UNSPECIFIED
}

func toProtoHybridChannel(c HybridChannel) simscenariov1.HybridChannel {
	switch c {
	case HybridChannelSim:
		return simscenariov1.HybridChannel_HYBRID_CHANNEL_SIM
	case HybridChannelContainer:
		return simscenariov1.HybridChannel_HYBRID_CHANNEL_CONTAINER
	}
	return simscenariov1.HybridChannel_HYBRID_CHANNEL_UNSPECIFIED
}

func toProtoInteractionDefinition(def InteractionDefinition) *simscenariov1.InteractionDefinition {
	actions := make([]*simscenariov1.ActionDef, 0, len(def.Actions))
	for _, action := range def.Actions {
		fields := make([]*simscenariov1.FieldDef, 0, len(action.Fields))
		for _, field := range action.Fields {
			fields = append(fields, &simscenariov1.FieldDef{
				Name:        field.Name,
				Type:        toProtoFieldType(field.Type),
				Label:       field.Label,
				Required:    field.Required,
				DefaultJson: marshalOrNil(field.Default),
				MinJson:     marshalOrNil(field.Min),
				MaxJson:     marshalOrNil(field.Max),
				StepJson:    marshalOrNil(field.Step),
				OptionsJson: marshalOrNil(field.Options),
				OptionsFrom: field.OptionsFrom,
			})
		}
		roles := make([]string, 0, len(action.Roles))
		for _, r := range action.Roles {
			roles = append(roles, string(r))
		}
		actions = append(actions, &simscenariov1.ActionDef{
			ActionCode:        action.ActionCode,
			Label:             action.Label,
			Description:       action.Description,
			Category:          toProtoActionCategory(action.Category),
			Trigger:           toProtoActionTrigger(action.Trigger),
			Fields:            fields,
			Roles:             roles,
			CooldownMs:        int32(action.CooldownMs),
			LinkOwnerFields:   append([]string(nil), action.LinkOwnerFields...),
			WritesOwnedFields: append([]string(nil), action.WritesOwnedFields...),
			Reversible:        action.Reversible,
			InterveneType:     string(action.InterveneType),
			HybridChannel:     toProtoHybridChannel(action.HybridChannel),
			ContainerCmd:      action.ContainerCmd,
		})
	}
	return &simscenariov1.InteractionDefinition{
		SceneCode:     def.SceneCode,
		SchemaVersion: def.SchemaVersion,
		Actions:       actions,
	}
}

// fromProtoLinkTriggers 把 Core 透传的联动事件转为 framework.LinkTrigger。
func fromProtoLinkTriggers(items []*simscenariov1.LinkTriggerRef) []LinkTrigger {
	if len(items) == 0 {
		return nil
	}
	result := make([]LinkTrigger, 0, len(items))
	for _, item := range items {
		var payload map[string]any
		if len(item.GetPayloadJson()) > 0 {
			if err := json.Unmarshal(item.GetPayloadJson(), &payload); err != nil {
				payload = nil
			}
		}
		result = append(result, LinkTrigger{
			ID:             item.GetId(),
			SourceScene:    item.GetSourceScene(),
			SourceAction:   item.GetSourceAction(),
			LinkGroup:      item.GetLinkGroup(),
			ChangedFields:  append([]string(nil), item.GetChangedFields()...),
			Payload:        payload,
			Timestamp:      item.GetTimestampMs(),
			SourceAnchorID: item.GetSourceAnchorId(),
			TargetAnchorID: item.GetTargetAnchorId(),
		})
	}
	return result
}

// fromProtoContainerMetrics 把 Core 注入的容器指标转为 framework.ContainerMetric。
//
// proto 把 value 字段以 JSON 字节传递（兼容任意类型），本函数解码为 any；
// 解码失败时 value 留空字符串以保持 Step 调用不中断（场景按需校验）。
func fromProtoContainerMetrics(items []*simscenariov1.ContainerMetric) []ContainerMetric {
	if len(items) == 0 {
		return nil
	}
	result := make([]ContainerMetric, 0, len(items))
	for _, item := range items {
		var value any
		if raw := item.GetValueJson(); len(raw) > 0 {
			if err := json.Unmarshal(raw, &value); err != nil {
				value = ""
			}
		}
		result = append(result, ContainerMetric{
			SourceContainer: item.GetSourceContainer(),
			MetricKey:       item.GetMetricKey(),
			Value:           value,
			Timestamp:       item.GetTimestampMs(),
			TargetPrimitive: item.GetTargetPrimitive(),
			TargetParam:     item.GetTargetParam(),
		})
	}
	return result
}

// marshalOrNil 把任意值序列化为 JSON 字节；nil 输入返回 nil。
//
// 用于 FieldDef 的 default / min / max / step / options 等任意类型字段统一以 JSON 字节传输。
func marshalOrNil(value any) []byte {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}
