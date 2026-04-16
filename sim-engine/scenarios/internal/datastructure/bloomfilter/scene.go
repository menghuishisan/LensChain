package bloomfilter

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const bloomSize = 12

// DefaultState 构造布隆过滤器场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "bloom-filter",
		Title:        "布隆过滤器",
		Phase:        "哈希定位",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   8,
		Stages:       []string{"哈希定位", "置位", "查询判断"},
		Nodes: []framework.Node{
			{ID: "hash-1", Label: "Hash-1", Status: "active", Role: "hash", X: 120, Y: 120},
			{ID: "hash-2", Label: "Hash-2", Status: "normal", Role: "hash", X: 120, Y: 200},
			{ID: "hash-3", Label: "Hash-3", Status: "normal", Role: "hash", X: 120, Y: 280},
			{ID: "bitset", Label: "BitSet", Status: "normal", Role: "hash", X: 380, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化位数组、已插入元素和最近查询结果。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := bloomModel{
		Key:          "alice",
		InsertedKeys: []string{"alice", "bob"},
		Bits:         make([]bool, bloomSize),
	}
	for _, key := range model.InsertedKeys {
		setBits(model.Bits, key)
	}
	model.LastPositions = positionsFor(model.Key)
	model.MightContain = queryBits(model.Bits, model.Key)
	return rebuildState(state, model, "哈希定位")
}

// Step 推进哈希定位、置位和查询判断。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "哈希定位"))
	switch phase {
	case "置位":
		setBits(model.Bits, model.Key)
	case "查询判断":
		model.MightContain = queryBits(model.Bits, model.Key)
	}
	model.LastPositions = positionsFor(model.Key)
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("布隆过滤器进入%s阶段。", phase), toneByQuery(model.MightContain, phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"positions": model.LastPositions,
				"bitset":    bitsToInts(model.Bits),
			},
		},
	}, nil
}

// HandleAction 查询新的元素并更新最近命中结果。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Key = framework.StringValue(input.Params["key"], "alice")
	model.LastPositions = positionsFor(model.Key)
	model.MightContain = queryBits(model.Bits, model.Key)
	if err := rebuildState(state, model, "查询判断"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "查询键", fmt.Sprintf("已查询元素 %s。", model.Key), toneByQuery(model.MightContain, "查询判断"))
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"positions": model.LastPositions,
			},
		},
	}, nil
}

// BuildRenderState 输出哈希位置、位数组和查询结果。
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

// bloomModel 保存位数组、已插入元素和最近查询结果。
type bloomModel struct {
	Key           string   `json:"key"`
	InsertedKeys  []string `json:"inserted_keys"`
	Bits          []bool   `json:"bits"`
	LastPositions []int    `json:"last_positions"`
	MightContain  bool     `json:"might_contain"`
}

// rebuildState 将布隆过滤器模型映射为节点状态、哈希消息和指标。
func rebuildState(state *framework.SceneState, model bloomModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	state.Nodes[3].Load = float64(countBits(model.Bits))
	state.Messages = []framework.Message{
		{ID: "h1", Label: fmt.Sprintf("%d", model.LastPositions[0]), Kind: "pointer", Status: phase, SourceID: "hash-1", TargetID: "bitset"},
		{ID: "h2", Label: fmt.Sprintf("%d", model.LastPositions[1]), Kind: "pointer", Status: phase, SourceID: "hash-2", TargetID: "bitset"},
		{ID: "h3", Label: fmt.Sprintf("%d", model.LastPositions[2]), Kind: "pointer", Status: phase, SourceID: "hash-3", TargetID: "bitset"},
	}
	state.Metrics = []framework.Metric{
		{Key: "key", Label: "查询键", Value: model.Key, Tone: "info"},
		{Key: "positions", Label: "命中位置", Value: formatPositions(model.LastPositions), Tone: "warning"},
		{Key: "ones", Label: "置位数量", Value: fmt.Sprintf("%d", countBits(model.Bits)), Tone: "success"},
		{Key: "contain", Label: "可能存在", Value: framework.BoolText(model.MightContain), Tone: toneByQuery(model.MightContain, phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "BitSet", Value: formatBits(model.Bits)},
		{Label: "Inserted", Value: strings.Join(model.InsertedKeys, ", ")},
		{Label: "Stage", Value: phase},
	}
	state.Data = map[string]any{
		"phase_name":   phase,
		"bloom_filter": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟布隆过滤器中的多哈希定位、置位和查询判断。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复布隆过滤器模型。
func decodeModel(state *framework.SceneState) bloomModel {
	entry, ok := state.Data["bloom_filter"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["bloom_filter"].(bloomModel); ok {
			return typed
		}
		model := bloomModel{Key: "alice", InsertedKeys: []string{"alice", "bob"}, Bits: make([]bool, bloomSize)}
		for _, key := range model.InsertedKeys {
			setBits(model.Bits, key)
		}
		model.LastPositions = positionsFor(model.Key)
		model.MightContain = queryBits(model.Bits, model.Key)
		return model
	}
	return bloomModel{
		Key:           framework.StringValue(entry["key"], "alice"),
		InsertedKeys:  framework.ToStringSliceOr(entry["inserted_keys"], []string{"alice", "bob"}),
		Bits:          framework.ToBoolSliceOr(entry["bits"], make([]bool, bloomSize)),
		LastPositions: framework.ToIntSliceOr(entry["last_positions"], []int{0, 0, 0}),
		MightContain:  framework.BoolValue(entry["might_contain"], false),
	}
}

// positionsFor 计算三个哈希位置。
func positionsFor(key string) []int {
	return []int{
		hashAt(key, 17),
		hashAt(key, 31),
		hashAt(key, 47),
	}
}

// setBits 根据键对位数组置位。
func setBits(bits []bool, key string) {
	for _, position := range positionsFor(key) {
		bits[position] = true
	}
}

// queryBits 判断某个键是否可能存在。
func queryBits(bits []bool, key string) bool {
	for _, position := range positionsFor(key) {
		if !bits[position] {
			return false
		}
	}
	return true
}

// hashAt 计算带种子的 FNV 位置。
func hashAt(key string, seed uint32) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d:%s", seed, key)))
	return int(hasher.Sum32() % bloomSize)
}

// nextPhase 返回布隆过滤器下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "哈希定位":
		return "置位"
	case "置位":
		return "查询判断"
	default:
		return "哈希定位"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "哈希定位":
		return 0
	case "置位":
		return 1
	case "查询判断":
		return 3
	default:
		return 0
	}
}

// toneByQuery 返回查询结果对应的色调。
func toneByQuery(mightContain bool, phase string) string {
	if phase == "查询判断" && mightContain {
		return "warning"
	}
	if phase == "查询判断" {
		return "success"
	}
	return "info"
}

// countBits 统计位数组中为真的数量。
func countBits(bits []bool) int {
	total := 0
	for _, bit := range bits {
		if bit {
			total++
		}
	}
	return total
}

// bitsToInts 将布尔数组转换为 0/1 数组。
func bitsToInts(bits []bool) []int {
	result := make([]int, 0, len(bits))
	for _, bit := range bits {
		if bit {
			result = append(result, 1)
			continue
		}
		result = append(result, 0)
	}
	return result
}

// formatPositions 格式化哈希位置。
func formatPositions(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

// formatBits 格式化位数组。
func formatBits(bits []bool) string {
	parts := make([]string, 0, len(bits))
	for _, bit := range bits {
		if bit {
			parts = append(parts, "1")
		} else {
			parts = append(parts, "0")
		}
	}
	return strings.Join(parts, "")
}
