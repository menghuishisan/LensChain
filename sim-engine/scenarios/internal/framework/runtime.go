package framework

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sdkscenario "github.com/lenschain/sim-engine/sdk/go/scenario"
)

// RuntimeScenario 将框架定义适配为 SDK 场景实现。
type RuntimeScenario struct {
	definition Definition
}

// NewRuntimeScenario 根据场景定义创建一个可直接挂到 gRPC 服务的场景实现。
func NewRuntimeScenario(definition Definition) (*RuntimeScenario, error) {
	if strings.TrimSpace(definition.Code) == "" {
		return nil, errors.New("场景编码不能为空")
	}
	if definition.DefaultState == nil {
		return nil, errors.New("必须提供默认状态构造函数")
	}
	if definition.Interaction == nil {
		return nil, errors.New("必须提供场景交互定义构造函数")
	}
	if definition.BuildRenderState == nil {
		return nil, errors.New("必须提供渲染状态构造函数")
	}
	return &RuntimeScenario{definition: definition}, nil
}

// Meta 返回场景元信息。
func (r *RuntimeScenario) Meta(context.Context) (sdkscenario.Meta, error) {
	defaultStateJSON, err := Encode(r.definition.DefaultState())
	if err != nil {
		return sdkscenario.Meta{}, err
	}
	meta := sdkscenario.Meta{
		Code:                    r.definition.Code,
		Name:                    r.definition.Name,
		Category:                sdkscenario.Category(r.definition.CategoryCode),
		AlgorithmType:           r.definition.AlgorithmType,
		Description:             r.definition.Description,
		Version:                 r.definition.Version,
		TimeControlMode:         sdkscenario.TimeControlMode(r.definition.TimeControlMode),
		DataSourceMode:          sdkscenario.DataSourceMode(r.definition.DataSourceMode),
		DefaultState:            defaultStateJSON,
		SupportedLinkGroupCodes: append([]string(nil), r.definition.SupportedLinks...),
	}
	defaultParams, err := Encode(r.definition.DefaultParams)
	if err != nil {
		return sdkscenario.Meta{}, err
	}
	meta.DefaultParams = defaultParams
	return meta, sdkscenario.ValidateMeta(meta)
}

// Init 初始化场景状态。
func (r *RuntimeScenario) Init(_ context.Context, req sdkscenario.InitRequest) (sdkscenario.State, error) {
	state := r.definition.DefaultState()
	state.SceneCode = r.definition.Code
	if strings.TrimSpace(req.SceneCode) != "" {
		state.SceneCode = req.SceneCode
	}
	state.Title = r.definition.Name
	state.StartedAt = time.Now().UTC()
	state.Seed = time.Now().UTC().UnixNano()
	if req.Seed != 0 {
		state.Seed = req.Seed
	}

	params, err := DecodeMap(req.ParamsJSON)
	if err != nil {
		return sdkscenario.State{}, err
	}
	initialState, err := DecodeMap(req.InitialStateJSON)
	if err != nil {
		return sdkscenario.State{}, err
	}
	sharedState, err := DecodeMap(req.SharedStateJSON)
	if err != nil {
		return sdkscenario.State{}, err
	}

	if r.definition.Init != nil {
		if err := r.definition.Init(&state, InitInput{
			SessionID:    req.SessionID,
			Params:       MergeMap(r.definition.DefaultParams, params),
			InitialState: initialState,
			SharedState:  sharedState,
			Seed:         state.Seed,
		}); err != nil {
			return sdkscenario.State{}, err
		}
	}
	encoded, err := r.encodeState(state)
	if err != nil {
		return sdkscenario.State{}, err
	}
	encoded.SharedStateJSON, err = Encode(sharedState)
	if err != nil {
		return sdkscenario.State{}, err
	}
	return encoded, nil
}

// Step 推进一步场景仿真状态。
func (r *RuntimeScenario) Step(_ context.Context, req sdkscenario.StepRequest) (sdkscenario.StepResult, error) {
	state, err := r.decodeState(req.StateJSON)
	if err != nil {
		return sdkscenario.StepResult{}, err
	}
	state.Tick = req.Tick + 1
	sharedState, err := DecodeMap(req.SharedStateJSON)
	if err != nil {
		return sdkscenario.StepResult{}, err
	}

	output := StepOutput{}
	if r.definition.Step != nil {
		output, err = r.definition.Step(&state, StepInput{SharedState: sharedState})
		if err != nil {
			return sdkscenario.StepResult{}, err
		}
	}
	encoded, err := r.encodeState(state)
	if err != nil {
		return sdkscenario.StepResult{}, err
	}
	diffJSON, err := Encode(output.SharedDiff)
	if err != nil {
		return sdkscenario.StepResult{}, err
	}
	events, err := r.toSDKEvents(output.Events)
	if err != nil {
		return sdkscenario.StepResult{}, err
	}
	return sdkscenario.StepResult{
		Tick:                encoded.Tick,
		StateJSON:           encoded.StateJSON,
		RenderStateJSON:     encoded.RenderStateJSON,
		Events:              events,
		SharedStateDiffJSON: diffJSON,
	}, nil
}

// HandleAction 处理前端交互。
func (r *RuntimeScenario) HandleAction(_ context.Context, req sdkscenario.ActionRequest) (sdkscenario.ActionResult, error) {
	state, err := r.decodeState(req.StateJSON)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}
	params, err := DecodeMap(req.ParamsJSON)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}
	sharedState, err := DecodeMap(req.SharedStateJSON)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}

	result := ActionOutput{Success: true}
	if r.definition.HandleAction != nil {
		result, err = r.definition.HandleAction(&state, ActionInput{
			ActionCode:  req.ActionCode,
			Params:      params,
			ActorID:     req.ActorID,
			RoleKey:     req.RoleKey,
			SharedState: sharedState,
		})
		if err != nil {
			return sdkscenario.ActionResult{}, err
		}
	}
	encoded, err := r.encodeState(state)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}
	events, err := r.toSDKEvents(result.Events)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}
	diffJSON, err := Encode(result.SharedDiff)
	if err != nil {
		return sdkscenario.ActionResult{}, err
	}
	return sdkscenario.ActionResult{
		Success:         result.Success,
		ErrorMessage:    result.Error,
		StateJSON:       encoded.StateJSON,
		RenderStateJSON: encoded.RenderStateJSON,
		Events:          events,
		SharedStateDiff: diffJSON,
	}, nil
}

// RenderState 基于当前完整状态与共享状态重新计算可渲染状态。
func (r *RuntimeScenario) RenderState(_ context.Context, req sdkscenario.RenderStateRequest) (sdkscenario.State, error) {
	state, err := r.decodeState(req.StateJSON)
	if err != nil {
		return sdkscenario.State{}, err
	}
	if req.Tick > 0 {
		state.Tick = req.Tick
	}
	sharedState, err := DecodeMap(req.SharedStateJSON)
	if err != nil {
		return sdkscenario.State{}, err
	}
	if r.definition.SyncSharedState != nil {
		if err := r.definition.SyncSharedState(&state, sharedState); err != nil {
			return sdkscenario.State{}, err
		}
	}
	encoded, err := r.encodeState(state)
	if err != nil {
		return sdkscenario.State{}, err
	}
	encoded.SharedStateJSON, err = Encode(sharedState)
	if err != nil {
		return sdkscenario.State{}, err
	}
	return encoded, nil
}

// InteractionSchema 返回场景专属交互定义。
func (r *RuntimeScenario) InteractionSchema(context.Context) (sdkscenario.InteractionSchema, error) {
	definition := r.definition.Interaction()
	actions := make([]sdkscenario.InteractionAction, 0, len(definition.Actions))
	for _, action := range definition.Actions {
		fields := make([]sdkscenario.InteractionField, 0, len(action.Fields))
		for _, field := range action.Fields {
			options := make([]sdkscenario.InteractionOption, 0, len(field.Options))
			for _, option := range field.Options {
				options = append(options, sdkscenario.InteractionOption{
					Value: option.Value,
					Label: option.Label,
				})
			}
			fields = append(fields, sdkscenario.InteractionField{
				Key:            field.Key,
				Label:          field.Label,
				Type:           sdkscenario.InteractionFieldType(field.Type),
				Required:       field.Required,
				DefaultValue:   field.DefaultValue,
				Options:        options,
				ValidationJSON: field.ValidationJSON,
			})
		}
		actions = append(actions, sdkscenario.InteractionAction{
			ActionCode:   action.ActionCode,
			Label:        action.Label,
			Description:  action.Description,
			Trigger:      sdkscenario.InteractionTrigger(action.Trigger),
			Fields:       fields,
			UISchemaJSON: action.UISchemaJSON,
		})
	}
	schema := sdkscenario.InteractionSchema{
		SceneCode: definition.SceneCode,
		Actions:   actions,
	}
	return schema, sdkscenario.ValidateInteractionSchema(schema)
}

// encodeState 编码完整状态和可渲染状态。
func (r *RuntimeScenario) encodeState(state SceneState) (sdkscenario.State, error) {
	stateJSON, err := Encode(state)
	if err != nil {
		return sdkscenario.State{}, err
	}
	renderEnvelope := r.definition.BuildRenderState(state)
	renderEnvelope.Extra = enrichRenderEnvelopeExtra(state, r.definition, renderEnvelope.Extra)
	renderJSON, err := Encode(renderEnvelope)
	if err != nil {
		return sdkscenario.State{}, err
	}
	return sdkscenario.State{
		Tick:            state.Tick,
		StateJSON:       stateJSON,
		RenderStateJSON: renderJSON,
	}, nil
}

// enrichRenderEnvelopeExtra 为渲染层补齐标准元数据，避免各场景重复拼装通用字段。
func enrichRenderEnvelopeExtra(state SceneState, definition Definition, extra map[string]any) map[string]any {
	enriched := CloneMap(extra)
	enriched["title"] = state.Title
	enriched["category"] = definition.CategoryCode
	enriched["algorithm_type"] = definition.AlgorithmType
	enriched["time_control_mode"] = definition.TimeControlMode
	enriched["metrics"] = state.Metrics
	enriched["tooltip"] = state.Tooltip
	enriched["timeline"] = state.Timeline
	enriched["linked"] = state.Linked
	if state.LinkGroup != "" {
		enriched["link_group_name"] = state.LinkGroup
	}
	return enriched
}

// decodeState 解码场景内部状态。
func (r *RuntimeScenario) decodeState(raw []byte) (SceneState, error) {
	if len(raw) == 0 {
		return r.definition.DefaultState(), nil
	}
	var state SceneState
	if err := decodeInto(raw, &state); err != nil {
		return SceneState{}, fmt.Errorf("解码场景状态失败: %w", err)
	}
	return state, nil
}

// toSDKEvents 将前端时间线事件转换为 SDK 事件。
func (r *RuntimeScenario) toSDKEvents(events []TimelineEvent) ([]sdkscenario.Event, error) {
	result := make([]sdkscenario.Event, 0, len(events))
	for _, event := range events {
		payload, err := Encode(map[string]any{
			"title":       event.Title,
			"description": event.Description,
			"tone":        event.Tone,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, sdkscenario.Event{
			EventID:     event.ID,
			EventType:   "timeline",
			SceneCode:   r.definition.Code,
			Tick:        event.Tick,
			TimestampMS: time.Now().UTC().UnixMilli(),
			PayloadJSON: payload,
		})
	}
	return result, nil
}

// decodeInto 将 JSON 直接解码到目标结构。
func decodeInto(raw []byte, target any) error {
	if len(raw) == 0 {
		return nil
	}
	return jsonUnmarshal(raw, target)
}
