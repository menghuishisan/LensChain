package tokeneconomics

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 Token 经济模型场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "token-economics",
		Title:        "Token 经济模型",
		Phase:        "初始分配",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   16,
		Stages:       []string{"初始分配", "线性释放", "通胀调整", "供应观察"},
		Nodes: []framework.Node{
			{ID: "team", Label: "Team", Status: "normal", Role: "allocation", X: 140, Y: 170},
			{ID: "community", Label: "Community", Status: "normal", Role: "allocation", X: 330, Y: 120},
			{ID: "treasury", Label: "Treasury", Status: "normal", Role: "allocation", X: 520, Y: 170},
			{ID: "supply", Label: "Supply", Status: "active", Role: "supply", X: 330, Y: 330},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化总供应量、分配比例和释放节奏。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := economicsModel{
		TotalSupply:    1000000,
		Circulating:    350000,
		InflationRate:  0.04,
		ReleaseStep:    50000,
		Allocations:    []allocation{{Name: "Team", Share: 0.2}, {Name: "Community", Share: 0.45}, {Name: "Treasury", Share: 0.35}},
		ReleasedAmount: 0,
	}
	applySharedEconomicsState(&model, input.SharedState)
	return rebuildState(state, model, "初始分配")
}

// Step 推进分配、释放、通胀与供应观察流程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedEconomicsState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "初始分配"))
	switch phase {
	case "初始分配":
		model.Circulating = 350000
	case "线性释放":
		model.ReleasedAmount += model.ReleaseStep
		model.Circulating += model.ReleaseStep
	case "通胀调整":
		inflation := int(float64(model.Circulating) * model.InflationRate)
		model.Circulating += inflation
		model.TotalSupply += inflation
	case "供应观察":
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Token 经济流程进入%s阶段。", phase), "info")
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"token_supply": map[string]any{
				"total":       model.TotalSupply,
				"circulating": model.Circulating,
				"inflation":   model.InflationRate,
			},
			"stakes": map[string]any{
				"treasury_reserve": reserveAmount(model),
			},
		},
	}, nil
}

// HandleAction 调整年化通胀率并立即反映到模型中。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	rate := framework.NumberValue(input.Params["inflation_rate"], model.InflationRate)
	if rate < 0 {
		rate = 0
	}
	if rate > 0.2 {
		rate = 0.2
	}
	model.InflationRate = rate
	if err := rebuildState(state, model, "通胀调整"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "调整通胀率", fmt.Sprintf("年化通胀率调整为 %.2f%%。", rate*100), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"token_supply": map[string]any{
				"inflation": rate,
			},
		},
	}, nil
}

// BuildRenderState 输出供应量、分配和释放数据。
func BuildRenderState(state framework.SceneState) framework.RenderEnvelope {
	return framework.RenderEnvelope{
		Nodes:       state.Nodes,
		Messages:    state.Messages,
		Stages:      state.Stages,
		ChangedKeys: state.ChangedKeys,
		Phase:       state.Phase,
		PhaseIndex:  state.PhaseIndex,
		Progress:    state.Progress,
		Data:        framework.CloneMap(state.Data),
		Extra:       framework.CloneMap(state.Extra),
	}
}

// SyncSharedState 在 PoS 经济组共享状态变化后重建 Token 经济场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedEconomicsState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// economicsModel 保存供应量、流通量、通胀率和释放计划。
type economicsModel struct {
	TotalSupply    int          `json:"total_supply"`
	Circulating    int          `json:"circulating"`
	InflationRate  float64      `json:"inflation_rate"`
	ReleaseStep    int          `json:"release_step"`
	ReleasedAmount int          `json:"released_amount"`
	Allocations    []allocation `json:"allocations"`
}

// allocation 保存单个分配对象及其占比。
type allocation struct {
	Name  string  `json:"name"`
	Share float64 `json:"share"`
}

// rebuildState 将经济模型转为可视化节点、指标和供应曲线数据。
func rebuildState(state *framework.SceneState, model economicsModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
	}
	for index, allocation := range model.Allocations {
		if index >= len(state.Nodes)-1 {
			break
		}
		state.Nodes[index].Load = allocation.Share * 100
		state.Nodes[index].Attributes = map[string]any{
			"allocation": allocation.Share,
			"amount":     int(float64(model.TotalSupply) * allocation.Share),
		}
	}
	state.Nodes[3].Status = "active"
	state.Nodes[3].Load = float64(model.Circulating) / float64(model.TotalSupply) * 100
	state.Nodes[3].Attributes = map[string]any{
		"total_supply": model.TotalSupply,
		"circulating":  model.Circulating,
		"inflation":    model.InflationRate,
	}
	state.Messages = []framework.Message{
		{ID: "release", Label: "release", Kind: "proposal", Status: phase, SourceID: "treasury", TargetID: "supply"},
	}
	state.Metrics = []framework.Metric{
		{Key: "total", Label: "总供应量", Value: fmt.Sprintf("%d", model.TotalSupply), Tone: "info"},
		{Key: "circulating", Label: "流通量", Value: fmt.Sprintf("%d", model.Circulating), Tone: "success"},
		{Key: "inflation", Label: "通胀率", Value: fmt.Sprintf("%.2f%%", model.InflationRate*100), Tone: "warning"},
		{Key: "released", Label: "累计释放", Value: fmt.Sprintf("%d", model.ReleasedAmount), Tone: "info"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "储备量", Value: fmt.Sprintf("%d", reserveAmount(model))},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"economics":  model,
	}
	state.Extra = map[string]any{
		"description": "该场景实现初始分配、线性释放、通胀调整和流通供应观察。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复经济模型。
func decodeModel(state *framework.SceneState) economicsModel {
	entry, ok := state.Data["economics"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["economics"].(economicsModel); ok {
			return typed
		}
		return economicsModel{TotalSupply: 1000000, Circulating: 350000, InflationRate: 0.04, ReleaseStep: 50000}
	}
	return economicsModel{
		TotalSupply:    int(framework.NumberValue(entry["total_supply"], 1000000)),
		Circulating:    int(framework.NumberValue(entry["circulating"], 350000)),
		InflationRate:  framework.NumberValue(entry["inflation_rate"], 0.04),
		ReleaseStep:    int(framework.NumberValue(entry["release_step"], 50000)),
		ReleasedAmount: int(framework.NumberValue(entry["released_amount"], 0)),
		Allocations:    decodeAllocations(entry["allocations"]),
	}
}

// applySharedEconomicsState 将 PoS 经济组中的质押与治理结果映射回 Token 经济模型。
func applySharedEconomicsState(model *economicsModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if tokenSupply, ok := sharedState["token_supply"].(map[string]any); ok {
		model.TotalSupply = int(framework.NumberValue(tokenSupply["total"], float64(model.TotalSupply)))
		model.Circulating = int(framework.NumberValue(tokenSupply["circulating"], float64(model.Circulating)))
		model.InflationRate = framework.NumberValue(tokenSupply["inflation"], model.InflationRate)
	}
	if stakes, ok := sharedState["stakes"].(map[string]any); ok {
		reserve := int(framework.NumberValue(stakes["treasury_reserve"], float64(reserveAmount(*model))))
		if reserve > 0 {
			model.Circulating = model.TotalSupply - reserve
		}
	}
}

// decodeAllocations 恢复分配结构列表。
func decodeAllocations(value any) []allocation {
	raw, ok := value.([]any)
	if !ok {
		return []allocation{{Name: "Team", Share: 0.2}, {Name: "Community", Share: 0.45}, {Name: "Treasury", Share: 0.35}}
	}
	result := make([]allocation, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, allocation{
			Name:  framework.StringValue(entry["name"], ""),
			Share: framework.NumberValue(entry["share"], 0),
		})
	}
	if len(result) == 0 {
		return []allocation{{Name: "Team", Share: 0.2}, {Name: "Community", Share: 0.45}, {Name: "Treasury", Share: 0.35}}
	}
	return result
}

// reserveAmount 返回当前未流通储备量。
func reserveAmount(model economicsModel) int {
	return model.TotalSupply - model.Circulating
}

// nextPhase 返回经济模型流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "初始分配":
		return "线性释放"
	case "线性释放":
		return "通胀调整"
	case "通胀调整":
		return "供应观察"
	default:
		return "初始分配"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "初始分配":
		return 0
	case "线性释放":
		return 1
	case "通胀调整":
		return 2
	case "供应观察":
		return 3
	default:
		return 0
	}
}
