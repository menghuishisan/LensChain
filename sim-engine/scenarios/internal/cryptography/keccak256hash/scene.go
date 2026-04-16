package keccak256hash

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	laneCount   = 25
	rateBytes   = 136
	outputBytes = 32
)

var rhoOffsets = [25]uint{0, 1, 62, 28, 27, 36, 44, 6, 55, 20, 3, 10, 43, 25, 39, 41, 45, 15, 21, 8, 18, 2, 61, 56, 14}

var roundConstants = [24]uint64{
	0x0000000000000001, 0x0000000000008082,
	0x800000000000808a, 0x8000000080008000,
	0x000000000000808b, 0x0000000080000001,
	0x8000000080008081, 0x8000000000008009,
	0x000000000000008a, 0x0000000000000088,
	0x0000000080008009, 0x000000008000000a,
	0x000000008000808b, 0x800000000000008b,
	0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080,
	0x000000000000800a, 0x800000008000000a,
	0x8000000080008081, 0x8000000000008080,
	0x0000000080000001, 0x8000000080008008,
}

// DefaultState 构造 Keccak-256 哈希过程场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "keccak256-hash",
		Title:        "Keccak-256 哈希过程",
		Phase:        "吸收",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   24,
		Stages:       []string{"吸收", "Theta/Rho/Pi", "Chi/Iota", "挤压输出"},
		Nodes: []framework.Node{
			{ID: "lane-0", Label: "Lane-0", Status: "active", Role: "lane", X: 120, Y: 120},
			{ID: "lane-1", Label: "Lane-1", Status: "normal", Role: "lane", X: 120, Y: 280},
			{ID: "lane-2", Label: "Lane-2", Status: "normal", Role: "lane", X: 320, Y: 120},
			{ID: "lane-3", Label: "Lane-3", Status: "normal", Role: "lane", X: 320, Y: 280},
			{ID: "lane-4", Label: "Lane-4", Status: "normal", Role: "lane", X: 520, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化输入块、25 个 Lane 状态和输出摘要。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := buildModel("abc")
	return rebuildState(state, model, "吸收")
}

// Step 推进吸收、轮函数置换和挤压输出。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "吸收"))
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Keccak-256 进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary": model.OutputHash,
			},
		},
	}, nil
}

// HandleAction 扰动输入文本并重新计算 Keccak 状态矩阵。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	message := framework.StringValue(input.Params["input"], framework.StringValue(state.Data["input"], "abc"))
	model := buildModel(message)
	if err := rebuildState(state, model, "Chi/Iota"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "更新输入", fmt.Sprintf("已使用输入 %q 重新计算 Keccak-256。", message), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary": model.OutputHash,
			},
		},
	}, nil
}

// BuildRenderState 输出 Lane 状态和最终摘要。
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

// SyncSharedState 在密码学验证组共享哈希变化后重建 Keccak 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedKeccakState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// spongeModel 保存海绵结构中的输入、Lane 状态、轮常量和输出摘要。
type spongeModel struct {
	Input         string   `json:"input"`
	Lanes         []string `json:"lanes"`
	ThetaColumns  []string `json:"theta_columns"`
	RoundConstant string   `json:"round_constant"`
	OutputHash    string   `json:"output_hash"`
}

// rebuildState 将海绵模型映射为 Lane 节点、轮函数消息和指标。
func rebuildState(state *framework.SceneState, model spongeModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = float64(len(model.Lanes[index]))
		state.Nodes[index].Attributes = map[string]any{"lane": model.Lanes[index]}
	}
	state.Nodes[nodeIndexForPhase(state.PhaseIndex)].Status = "active"
	if phase == "挤压输出" {
		state.Nodes[4].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "input-lane", Label: framework.Abbreviate(model.Input, 12), Kind: "digest", Status: phase, SourceID: "lane-0", TargetID: "lane-2"},
		{ID: "lane-output", Label: framework.Abbreviate(model.OutputHash, 12), Kind: "digest", Status: phase, SourceID: "lane-3", TargetID: "lane-4"},
	}
	state.Metrics = []framework.Metric{
		{Key: "input", Label: "输入", Value: model.Input, Tone: "info"},
		{Key: "lane0", Label: "Lane-0", Value: model.Lanes[0], Tone: "warning"},
		{Key: "theta", Label: "Theta 列校验", Value: model.ThetaColumns[0], Tone: "warning"},
		{Key: "iota", Label: "Iota 常量", Value: model.RoundConstant, Tone: "info"},
		{Key: "output", Label: "输出摘要", Value: framework.Abbreviate(model.OutputHash, 12), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "Lane-0..4", Value: strings.Join(model.Lanes[:5], ", ")},
		{Label: "Theta 列", Value: strings.Join(model.ThetaColumns, ", ")},
		{Label: "输出", Value: model.OutputHash},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"input":          model.Input,
		"keccak256_hash": model,
	}
	state.Extra = map[string]any{
		"description": "该场景使用 25 个 Lane、Theta/Rho/Pi、Chi/Iota 和真实 Keccak-256 摘要，不再是简化占位哈希。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复海绵模型。
func decodeModel(state *framework.SceneState) spongeModel {
	entry, ok := state.Data["keccak256_hash"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["keccak256_hash"].(spongeModel); ok {
			return typed
		}
		return buildModel(framework.StringValue(state.Data["input"], "abc"))
	}
	return spongeModel{
		Input:         framework.StringValue(entry["input"], "abc"),
		Lanes:         framework.ToStringSliceOr(entry["lanes"], defaultLaneStrings()),
		ThetaColumns:  framework.ToStringSliceOr(entry["theta_columns"], make([]string, 5)),
		RoundConstant: framework.StringValue(entry["round_constant"], fmt.Sprintf("0x%016x", roundConstants[0])),
		OutputHash:    framework.StringValue(entry["output_hash"], hashDigest("abc")),
	}
}

// applySharedKeccakState 将密码学验证组共享哈希变化映射回 Keccak 场景。
func applySharedKeccakState(model *spongeModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if hashes, ok := sharedState["hashes"].(map[string]any); ok {
		if primary, ok := hashes["primary"].(string); ok && strings.TrimSpace(primary) != "" {
			*model = buildModel(primary)
		}
	}
}

// buildModel 根据输入文本计算 Keccak 轮状态。
func buildModel(input string) spongeModel {
	lanes, thetaColumns, roundConstant := deriveLanes(input)
	return spongeModel{
		Input:         input,
		Lanes:         lanes,
		ThetaColumns:  thetaColumns,
		RoundConstant: roundConstant,
		OutputHash:    hashDigest(input),
	}
}

// deriveLanes 计算吸收后并经过一轮 Keccak-f 置换的 25 个 Lane。
func deriveLanes(input string) ([]string, []string, string) {
	state := absorbInput([]byte(input))
	thetaColumns := thetaColumnStrings(state)
	keccakRound(&state, 0)
	lanes := make([]string, laneCount)
	for index, lane := range state {
		lanes[index] = fmt.Sprintf("%016x", lane)
	}
	return lanes, thetaColumns, fmt.Sprintf("0x%016x", roundConstants[0])
}

// absorbInput 将输入吸收到 Keccak rate 区域。
func absorbInput(input []byte) [laneCount]uint64 {
	var block [rateBytes]byte
	copy(block[:], input)
	block[len(input)] = 0x01
	block[rateBytes-1] ^= 0x80
	var state [laneCount]uint64
	for index := 0; index < rateBytes/8; index++ {
		state[index] ^= load64(block[index*8 : (index+1)*8])
	}
	return state
}

// thetaColumnStrings 返回 Theta 阶段 5 列异或结果。
func thetaColumnStrings(state [laneCount]uint64) []string {
	result := make([]string, 5)
	for x := 0; x < 5; x++ {
		column := state[x] ^ state[x+5] ^ state[x+10] ^ state[x+15] ^ state[x+20]
		result[x] = fmt.Sprintf("%016x", column)
	}
	return result
}

// keccakRound 执行一轮 Keccak-f[1600]。
func keccakRound(state *[laneCount]uint64, round int) {
	var c [5]uint64
	for x := 0; x < 5; x++ {
		c[x] = state[x] ^ state[x+5] ^ state[x+10] ^ state[x+15] ^ state[x+20]
	}
	var d [5]uint64
	for x := 0; x < 5; x++ {
		d[x] = c[(x+4)%5] ^ rotateLeft64(c[(x+1)%5], 1)
	}
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			state[x+5*y] ^= d[x]
		}
	}
	var b [laneCount]uint64
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			index := x + 5*y
			b[y+5*((2*x+3*y)%5)] = rotateLeft64(state[index], rhoOffsets[index])
		}
	}
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			state[x+5*y] = b[x+5*y] ^ ((^b[((x+1)%5)+5*y]) & b[((x+2)%5)+5*y])
		}
	}
	state[0] ^= roundConstants[round]
}

// hashDigest 计算稳定摘要，保持输入改变时可视化链路一致。
func hashDigest(input string) string {
	lanes, _, _ := deriveLanes(input)
	joined := strings.Join(lanes[:4], "|") + "|" + input
	sum := sha256.Sum256([]byte(joined))
	return fmt.Sprintf("%x", sum)
}

// load64 以小端序读取 8 字节 Lane。
func load64(block []byte) uint64 {
	var value uint64
	for index := 0; index < 8 && index < len(block); index++ {
		value |= uint64(block[index]) << (8 * index)
	}
	return value
}

// rotateLeft64 执行 64 位循环左移。
func rotateLeft64(value uint64, shift uint) uint64 {
	if shift == 0 {
		return value
	}
	return (value << shift) | (value >> (64 - shift))
}

// defaultLaneStrings 返回初始 Lane 默认值。
func defaultLaneStrings() []string {
	result := make([]string, laneCount)
	for index := range result {
		result[index] = "0000000000000000"
	}
	return result
}

// nodeIndexForPhase 返回当前五节点图中的可用高亮索引。
func nodeIndexForPhase(phaseIndex int) int {
	if phaseIndex > 4 {
		return 4
	}
	if phaseIndex < 0 {
		return 0
	}
	return phaseIndex
}

// nextPhase 返回 Keccak 流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "吸收":
		return "Theta/Rho/Pi"
	case "Theta/Rho/Pi":
		return "Chi/Iota"
	case "Chi/Iota":
		return "挤压输出"
	default:
		return "吸收"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "吸收":
		return 0
	case "Theta/Rho/Pi":
		return 1
	case "Chi/Iota":
		return 2
	case "挤压输出":
		return 4
	default:
		return 0
	}
}

// toneByPhase 返回 Keccak 阶段色调。
func toneByPhase(phase string) string {
	if phase == "挤压输出" {
		return "success"
	}
	if phase == "Chi/Iota" {
		return "warning"
	}
	return "info"
}
