package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// StepProfile 描述过程化、响应式和持续运行场景的节奏与阶段。
type StepProfile struct {
	Stages       []string
	TotalTicks   int64
	StepDuration int64
}

// ActionSpec 描述一个场景专属交互动作。
type ActionSpec struct {
	ActionCode   string
	Label        string
	Description  string
	Trigger      string
	Fields       []framework.InteractionFieldDefinition
	DefaultValue string
	FieldType    string
	FieldKey     string
	FieldLabel   string
}

// SceneTemplate 描述单个场景定义的文档驱动配置。
type SceneTemplate struct {
	Code            string
	Name            string
	Description     string
	CategoryCode    string
	AlgorithmType   string
	Version         string
	TimeControlMode string
	DataSourceMode  string
	LinkGroups      []string
	Profile         StepProfile
	BaseNodeLabels  []string
	BaseNodeRole    string
	Actions         []ActionSpec
	MetricsBuilder  func(state framework.SceneState) []framework.Metric
	DataBuilder     func(state framework.SceneState) map[string]any
	ActionHandler   func(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error)
	DefaultState    func() framework.SceneState
	InitHandler     func(state *framework.SceneState, input framework.InitInput) error
	StepHandler     func(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error)
	SyncHandler     func(state *framework.SceneState, sharedState map[string]any) error
	RenderBuilder   func(state framework.SceneState) framework.RenderEnvelope
}

// buildDefinition 将模板转换为可执行的场景定义。
func buildDefinition(template SceneTemplate) framework.Definition {
	validateTemplate(template)
	return framework.Definition{
		Code:            template.Code,
		Name:            template.Name,
		Description:     template.Description,
		CategoryCode:    template.CategoryCode,
		AlgorithmType:   template.AlgorithmType,
		Version:         template.Version,
		TimeControlMode: template.TimeControlMode,
		DataSourceMode:  template.DataSourceMode,
		DefaultParams: map[string]any{
			"scene_code":      template.Code,
			"algorithm_type":  template.AlgorithmType,
			"time_control":    template.TimeControlMode,
			"step_duration":   template.Profile.StepDuration,
			"total_ticks":     template.Profile.TotalTicks,
			"default_actions": len(template.Actions),
		},
		DefaultState: func() framework.SceneState {
			return template.DefaultState()
		},
		SupportedLinks: template.LinkGroups,
		Interaction: func() framework.InteractionDefinition {
			return buildInteractionDefinition(template)
		},
		Init: func(state *framework.SceneState, input framework.InitInput) error {
			state.Seed = input.Seed
			if len(input.InitialState) > 0 {
				state.Extra = framework.MergeMap(state.Extra, input.InitialState)
			}
			state.LinkGroup = resolveLinkGroupCode(input.Params)
			state.Linked = strings.TrimSpace(state.LinkGroup) != "" && len(input.SharedState) > 0
			return template.InitHandler(state, input)
		},
		Step: func(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
			state.Linked = strings.TrimSpace(state.LinkGroup) != "" && len(input.SharedState) > 0
			return template.StepHandler(state, input)
		},
		HandleAction: func(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
			return template.ActionHandler(state, input)
		},
		SyncSharedState: func(state *framework.SceneState, sharedState map[string]any) error {
			if template.SyncHandler == nil {
				return nil
			}
			return template.SyncHandler(state, sharedState)
		},
		BuildRenderState: func(state framework.SceneState) framework.RenderEnvelope {
			return template.RenderBuilder(state)
		},
	}
}

// validateTemplate 校验内置场景必须显式声明自己的处理器，避免隐式兜底偏离文档。
func validateTemplate(template SceneTemplate) {
	required := map[string]any{
		"DefaultState":  template.DefaultState,
		"InitHandler":   template.InitHandler,
		"StepHandler":   template.StepHandler,
		"ActionHandler": template.ActionHandler,
		"RenderBuilder": template.RenderBuilder,
	}
	for name, handler := range required {
		if handler == nil {
			panic(fmt.Sprintf("场景 %s 缺少必需处理器 %s", template.Code, name))
		}
	}
	if strings.TrimSpace(template.Code) == "" || strings.TrimSpace(template.Name) == "" {
		panic("场景必须声明 code 和 name")
	}
	if strings.TrimSpace(template.CategoryCode) == "" || strings.TrimSpace(template.AlgorithmType) == "" {
		panic(fmt.Sprintf("场景 %s 必须声明 category 和 algorithm_type", template.Code))
	}
	if len(template.LinkGroups) > 0 {
		linkGroups := slices.Clone(template.LinkGroups)
		slices.Sort(linkGroups)
		for _, linkGroup := range linkGroups {
			if strings.TrimSpace(linkGroup) == "" {
				panic(fmt.Sprintf("场景 %s 的联动组编码不能为空", template.Code))
			}
		}
		if template.SyncHandler == nil {
			panic(fmt.Sprintf("场景 %s 声明了联动组但缺少 SyncHandler", template.Code))
		}
	}
	switch strings.TrimSpace(template.TimeControlMode) {
	case "process", "reactive", "continuous":
	default:
		panic(fmt.Sprintf("场景 %s 的 time_control_mode 非法: %s", template.Code, template.TimeControlMode))
	}
	switch strings.TrimSpace(template.DataSourceMode) {
	case "simulation", "collection", "dual":
	default:
		panic(fmt.Sprintf("场景 %s 的 data_source_mode 非法: %s", template.Code, template.DataSourceMode))
	}
}

// buildInteractionDefinition 将模板动作转换为交互 schema。
func buildInteractionDefinition(template SceneTemplate) framework.InteractionDefinition {
	actions := make([]framework.InteractionActionDefinition, 0, len(template.Actions))
	for _, action := range template.Actions {
		fields := action.Fields
		if len(fields) == 0 {
			fields = []framework.InteractionFieldDefinition{
				{
					Key:          action.FieldKey,
					Label:        action.FieldLabel,
					Type:         action.FieldType,
					Required:     true,
					DefaultValue: action.DefaultValue,
				},
			}
		}
		actions = append(actions, framework.InteractionActionDefinition{
			ActionCode:  action.ActionCode,
			Label:       action.Label,
			Description: action.Description,
			Trigger:     action.Trigger,
			Fields:      fields,
		})
	}
	return framework.InteractionDefinition{
		SceneCode: template.Code,
		Actions:   actions,
	}
}

// resolveLinkGroupCode 从初始化参数中提取本次运行实际选中的联动组编码。
func resolveLinkGroupCode(params map[string]any) string {
	if params == nil {
		return ""
	}
	value, ok := params["link_group_code"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
