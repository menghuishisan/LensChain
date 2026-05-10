// 模块：sim-engine/sdk/go/scenario
// 文件职责：把场景作者声明的 framework.Definition 适配为 sdk.Scenario，让平台内部场景与
//          外部场景共用同一套 gRPC 适配（sdk.Server）。
// 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §6.4。
//
// 核心规则：场景容器无状态，所有跨 Step 状态通过 SceneState 序列化往返；
//          RuntimeScenario 不持有任何运行时状态。

package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	fw "github.com/lenschain/sim-engine/framework"
)

// RuntimeScenario 把 framework.Definition 适配为 sdk.Scenario。
type RuntimeScenario struct {
	def Definition
}

// NewRuntimeScenario 校验 Definition 必填项后返回 sdk.Scenario 适配器。
//
// 全量协议校验（含 semver 版本、TimeControlMode 与 DataSourceMode 合法性、ExtensionLevel、
// Interaction 中必备的 broadcast_hint / 教师 ActionDef、WritesOwnedFields ⊆ OwnedFieldPaths 等）
// 由 framework.ValidateDefinition 完成（详 AGENTS.md §0.7.1 C23）。
func NewRuntimeScenario(def Definition) (*RuntimeScenario, error) {
	if err := fw.ValidateDefinition(def); err != nil {
		return nil, err
	}
	return &RuntimeScenario{def: def}, nil
}

// Meta 实现 Scenario.Meta：把 Definition 元信息封装为 sdk.Meta。
func (r *RuntimeScenario) Meta(context.Context) (Meta, error) {
	defaultParams := map[string]any{}
	if r.def.DefaultParams != nil {
		defaultParams = r.def.DefaultParams()
	}
	paramsJSON, err := json.Marshal(defaultParams)
	if err != nil {
		return Meta{}, fmt.Errorf("编码默认参数失败: %w", err)
	}
	stateJSON, err := json.Marshal(r.def.DefaultState())
	if err != nil {
		return Meta{}, fmt.Errorf("编码默认状态失败: %w", err)
	}
	return Meta{
		Code:                    r.def.Code,
		Name:                    r.def.Name,
		Description:             r.def.Description,
		Category:                r.def.Category,
		AlgorithmType:           r.def.AlgorithmType,
		Version:                 r.def.Version,
		TimeControlMode:         r.def.TimeControlMode,
		DataSourceMode:          r.def.DataSourceMode,
		DefaultParams:           paramsJSON,
		DefaultState:            stateJSON,
		SupportedLinkGroupCodes: append([]string(nil), r.def.SupportedLinkGroups...),

		// v0.5 新增（详 AGENTS.md §0.7.1 C10 / C29 / C37）。
		ExtensionLevel:     r.def.ExtensionLevel,
		LinkGroupVersion:   r.def.LinkGroupVersion,
		SupportsMultiActor: r.def.SupportsMultiActor,
		OwnedFieldPaths:    append([]string(nil), r.def.OwnedFieldPaths...),
	}, nil
}

// InteractionSchema 实现 Scenario.InteractionSchema：把 Definition.Interaction() 转换为协议 InteractionDefinition。
func (r *RuntimeScenario) InteractionSchema(context.Context) (InteractionDefinition, error) {
	def := r.def.Interaction()
	if strings.TrimSpace(def.SceneCode) == "" {
		def.SceneCode = r.def.Code
	}
	if strings.TrimSpace(def.SchemaVersion) == "" {
		def.SchemaVersion = r.def.Version
	}
	return def, nil
}

// Init 实现 Scenario.Init：解码 params/shared，调用 Definition.Init，编码返回。
func (r *RuntimeScenario) Init(_ context.Context, req InitRequest) (InitResult, error) {
	state := r.def.DefaultState()
	state.SceneCode = r.def.Code
	if state.StartedAt == 0 {
		state.StartedAt = time.Now().UTC().UnixMilli()
	}
	if req.Seed != 0 {
		state.Seed = req.Seed
	} else if state.Seed == 0 {
		state.Seed = time.Now().UTC().UnixNano()
	}

	params, err := decodeMap(req.ParamsJSON)
	if err != nil {
		return InitResult{}, fmt.Errorf("解码 params 失败: %w", err)
	}
	if r.def.DefaultParams != nil {
		params = fw.MergeMap(r.def.DefaultParams(), params)
	}
	shared, err := decodeMap(req.SharedStateJSON)
	if err != nil {
		return InitResult{}, fmt.Errorf("解码 shared_state 失败: %w", err)
	}

	envelope, err := r.def.Init(&state, InitInput{
		SessionID:   req.SessionID,
		InstanceID:  req.InstanceID,
		StudentID:   req.StudentID,
		Seed:        state.Seed,
		Params:      params,
		SharedState: shared,
	})
	if err != nil {
		return InitResult{}, err
	}
	envelope.IsFullSnapshot = true

	stateJSON, envelopeJSON, err := encodeStateAndEnvelope(state, envelope)
	if err != nil {
		return InitResult{}, err
	}
	return InitResult{
		Tick:               state.Tick,
		SceneStateJSON:     stateJSON,
		RenderEnvelopeJSON: envelopeJSON,
	}, nil
}

// Step 实现 Scenario.Step：解码状态，调用 Definition.Step 推进，编码返回。
func (r *RuntimeScenario) Step(_ context.Context, req StepRequest) (StepResult, error) {
	state, err := r.decodeSceneState(req.SceneStateJSON)
	if err != nil {
		return StepResult{}, err
	}
	state.Tick = req.Tick
	shared, err := decodeMap(req.SharedStateJSON)
	if err != nil {
		return StepResult{}, fmt.Errorf("解码 shared_state 失败: %w", err)
	}

	output, err := r.def.Step(&state, StepInput{
		Tick:                     req.Tick,
		SharedState:              shared,
		IncomingLinkTriggers:     append([]LinkTrigger(nil), req.IncomingLinkTriggers...),
		IncomingContainerMetrics: append([]ContainerMetric(nil), req.IncomingContainerMetrics...),
	})
	if err != nil {
		return StepResult{}, err
	}

	stateJSON, envelopeJSON, err := encodeStateAndEnvelope(state, output.Render)
	if err != nil {
		return StepResult{}, err
	}
	diffJSON, err := encodeMap(output.SharedStateDiff)
	if err != nil {
		return StepResult{}, err
	}
	return StepResult{
		Tick:                state.Tick,
		SceneStateJSON:      stateJSON,
		RenderEnvelopeJSON:  envelopeJSON,
		SharedStateDiffJSON: diffJSON,
	}, nil
}

// HandleAction 实现 Scenario.HandleAction：解码状态与参数，调用 Definition.HandleAction，编码返回。
func (r *RuntimeScenario) HandleAction(_ context.Context, req ActionRequest) (ActionResult, error) {
	state, err := r.decodeSceneState(req.SceneStateJSON)
	if err != nil {
		return ActionResult{}, err
	}
	if req.Tick != 0 {
		state.Tick = req.Tick
	}
	params, err := decodeMap(req.ParamsJSON)
	if err != nil {
		return ActionResult{}, fmt.Errorf("解码 params 失败: %w", err)
	}
	shared, err := decodeMap(req.SharedStateJSON)
	if err != nil {
		return ActionResult{}, fmt.Errorf("解码 shared_state 失败: %w", err)
	}

	output, err := r.def.HandleAction(&state, ActionInput{
		Tick:        state.Tick,
		ActionCode:  req.ActionCode,
		Params:      params,
		ActorID:     req.ActorID,
		UserRole:    req.UserRole,
		SharedState: shared,
	})
	if err != nil {
		return ActionResult{}, err
	}

	stateJSON, envelopeJSON, err := encodeStateAndEnvelope(state, output.Render)
	if err != nil {
		return ActionResult{}, err
	}
	diffJSON, err := encodeMap(output.SharedStateDiff)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		Success:             output.Success,
		ErrorMessage:        output.ErrorMessage,
		Tick:                state.Tick,
		SceneStateJSON:      stateJSON,
		RenderEnvelopeJSON:  envelopeJSON,
		SharedStateDiffJSON: diffJSON,
	}, nil
}

// =====================================================================
// 内部工具
// =====================================================================

// decodeSceneState 反序列化 SceneState；空字节回退为 Definition.DefaultState()。
func (r *RuntimeScenario) decodeSceneState(raw []byte) (SceneState, error) {
	if len(raw) == 0 {
		return r.def.DefaultState(), nil
	}
	var state SceneState
	if err := json.Unmarshal(raw, &state); err != nil {
		return SceneState{}, fmt.Errorf("解码 scene_state 失败: %w", err)
	}
	return state, nil
}

// decodeMap 反序列化 map[string]any；空字节返回空 map。
func decodeMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

// encodeMap 序列化 map[string]any；空 map 返回 nil。
func encodeMap(value map[string]any) ([]byte, error) {
	if len(value) == 0 {
		return nil, nil
	}
	return json.Marshal(value)
}

// encodeStateAndEnvelope 同时序列化 SceneState 与 RenderEnvelope。
func encodeStateAndEnvelope(state SceneState, env RenderEnvelope) ([]byte, []byte, error) {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, nil, fmt.Errorf("编码 scene_state 失败: %w", err)
	}
	envelopeJSON, err := json.Marshal(env)
	if err != nil {
		return nil, nil, fmt.Errorf("编码 render_envelope 失败: %w", err)
	}
	return stateJSON, envelopeJSON, nil
}
