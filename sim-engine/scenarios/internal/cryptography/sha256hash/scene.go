package sha256hash

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/bits"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

var (
	// sha256Init 是 SHA-256 压缩函数的初始哈希向量。
	sha256Init = [8]uint32{
		0x6a09e667,
		0xbb67ae85,
		0x3c6ef372,
		0xa54ff53a,
		0x510e527f,
		0x9b05688c,
		0x1f83d9ab,
		0x5be0cd19,
	}
	// sha256K 是 64 轮压缩函数使用的常量表。
	sha256K = [64]uint32{
		0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
		0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
		0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
		0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
		0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
		0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
		0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
		0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
		0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
		0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
		0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
		0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
		0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
		0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
		0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
		0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
	}
)

// DefaultState 构造 SHA-256 场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "sha256-hash",
		Title:        "SHA-256 哈希过程",
		Phase:        "消息分块",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   64,
		Stages:       []string{"消息分块", "消息填充", "压缩轮函数", "哈希输出"},
		Nodes: []framework.Node{
			{ID: "sha-input", Label: "Input", Status: "active", Role: "message", X: 80, Y: 180},
			{ID: "sha-padding", Label: "Padding", Status: "normal", Role: "padding", X: 220, Y: 180},
			{ID: "sha-schedule", Label: "Schedule", Status: "normal", Role: "schedule", X: 380, Y: 180},
			{ID: "sha-round", Label: "Round", Status: "normal", Role: "round", X: 520, Y: 180},
			{ID: "sha-output", Label: "Digest", Status: "normal", Role: "digest", X: 680, Y: 180},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 SHA-256 场景，并完成首轮真实压缩函数预计算。
func Init(state *framework.SceneState, input framework.InitInput) error {
	message := framework.StringValue(input.Params["input"], "abc")
	if hashes, ok := input.SharedState["hashes"].(map[string]any); ok {
		message = framework.StringValue(hashes["primary"], message)
	}
	return rebuildState(state, message, 0)
}

// Step 以真实 64 轮压缩函数结果驱动 round 指针推进。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	currentRound := int(framework.NumberValue(state.Data["current_round"], 0))
	message := framework.StringValue(state.Data["input"], "abc")
	if hashes, ok := input.SharedState["hashes"].(map[string]any); ok {
		message = framework.StringValue(hashes["primary"], message)
	}
	if currentRound < 63 {
		currentRound++
	}
	if err := rebuildState(state, message, currentRound); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(
		state.SceneCode,
		state.Tick,
		fmt.Sprintf("第 %d 轮压缩", currentRound+1),
		fmt.Sprintf("W[%d]=%08x，状态寄存器完成一次真实更新。", currentRound, uint32(framework.NumberValue(state.Data["current_word"], 0))),
		"warning",
	)
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary": state.Data["digest"],
			},
		},
	}, nil
}

// HandleAction 允许教师或学生替换输入并重新计算完整哈希过程。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	message := framework.StringValue(input.Params["input"], "abc")
	if err := rebuildState(state, message, 0); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "重新计算哈希", fmt.Sprintf("输入已切换为 %q，并重新执行 SHA-256。", message), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary": state.Data["digest"],
			},
		},
	}, nil
}

// BuildRenderState 将真实 SHA-256 中间结果打包给渲染层。
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

// SyncSharedState 在密码学联动共享摘要变化后重建 SHA-256 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	message := framework.StringValue(state.Data["input"], "hello blockchain")
	if hashes, ok := sharedState["hashes"].(map[string]any); ok {
		message = framework.StringValue(hashes["primary"], message)
	}
	return rebuildState(state, message, int(framework.NumberValue(state.Data["current_round"], 0)))
}

// rebuildState 根据输入字符串和当前轮次重建可视化状态。
func rebuildState(state *framework.SceneState, message string, currentRound int) error {
	detail := buildHashDetail([]byte(message))
	if currentRound < 0 {
		currentRound = 0
	}
	if currentRound >= len(detail.Rounds) {
		currentRound = len(detail.Rounds) - 1
	}
	activeRound := detail.Rounds[currentRound]
	state.PhaseIndex, state.Phase = phaseByRound(currentRound)
	state.Progress = float64(currentRound+1) / 64.0
	state.Messages = []framework.Message{
		{ID: "msg-input", Label: "原始消息", Kind: "digest", Status: state.Phase, SourceID: "sha-input", TargetID: "sha-padding"},
		{ID: "msg-padding", Label: "填充后块", Kind: "digest", Status: state.Phase, SourceID: "sha-padding", TargetID: "sha-schedule"},
		{ID: "msg-schedule", Label: fmt.Sprintf("W[%d]", currentRound), Kind: "digest", Status: state.Phase, SourceID: "sha-schedule", TargetID: "sha-round"},
		{ID: "msg-output", Label: "最终摘要", Kind: "digest", Status: state.Phase, SourceID: "sha-round", TargetID: "sha-output"},
	}
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[state.PhaseIndex+1].Status = "active"
	state.Metrics = []framework.Metric{
		{Key: "round", Label: "当前轮次", Value: fmt.Sprintf("%d / 64", currentRound+1), Tone: "warning"},
		{Key: "blocks", Label: "消息块数", Value: fmt.Sprintf("%d", len(detail.Blocks)), Tone: "info"},
		{Key: "digest", Label: "哈希输出", Value: detail.Digest[:16] + "...", Tone: "success"},
		{Key: "avalanche", Label: "雪崩差异位数", Value: fmt.Sprintf("%d bit", detail.AvalancheBits), Tone: "info"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "输入", Value: detail.Input},
		{Label: "当前轮常量", Value: fmt.Sprintf("%08x", activeRound.K)},
		{Label: "当前消息字", Value: fmt.Sprintf("%08x", activeRound.W)},
		{Label: "最终摘要", Value: detail.Digest},
	}
	state.Data = map[string]any{
		"input":             detail.Input,
		"input_hex":         detail.InputHex,
		"padded_hex":        detail.PaddedHex,
		"message_blocks":    detail.Blocks,
		"schedule_words":    detail.Schedule,
		"rounds":            detail.Rounds,
		"current_round":     currentRound,
		"current_word":      activeRound.W,
		"current_constant":  activeRound.K,
		"digest":            detail.Digest,
		"mutated_input":     detail.MutatedInput,
		"mutated_digest":    detail.MutatedDigest,
		"avalanche_bits":    detail.AvalancheBits,
		"working_registers": activeRound.Registers(),
	}
	state.Extra = map[string]any{
		"description": "该场景使用真实 SHA-256 填充、消息调度与 64 轮压缩函数，不是占位动画。",
	}
	state.ChangedKeys = []string{"metrics", "messages", "data", "nodes", "tooltip"}
	return nil
}

// hashDetail 汇总一次真实 SHA-256 计算需要输出给渲染层的全部中间结果。
type hashDetail struct {
	Input         string
	InputHex      string
	PaddedHex     string
	Blocks        []string
	Schedule      []string
	Rounds        []roundState
	Digest        string
	MutatedInput  string
	MutatedDigest string
	AvalancheBits int
}

// roundState 保存单轮压缩函数结束后的寄存器快照。
type roundState struct {
	Round int    `json:"round"`
	W     uint32 `json:"w"`
	K     uint32 `json:"k"`
	A     uint32 `json:"a"`
	B     uint32 `json:"b"`
	C     uint32 `json:"c"`
	D     uint32 `json:"d"`
	E     uint32 `json:"e"`
	F     uint32 `json:"f"`
	G     uint32 `json:"g"`
	H     uint32 `json:"h"`
	T1    uint32 `json:"t1"`
	T2    uint32 `json:"t2"`
}

// Registers 返回渲染层更容易直接消费的寄存器映射。
func (r roundState) Registers() map[string]any {
	return map[string]any{
		"a": fmt.Sprintf("%08x", r.A),
		"b": fmt.Sprintf("%08x", r.B),
		"c": fmt.Sprintf("%08x", r.C),
		"d": fmt.Sprintf("%08x", r.D),
		"e": fmt.Sprintf("%08x", r.E),
		"f": fmt.Sprintf("%08x", r.F),
		"g": fmt.Sprintf("%08x", r.G),
		"h": fmt.Sprintf("%08x", r.H),
	}
}

// buildHashDetail 计算真实 SHA-256 中间过程，并提取首个消息块的 64 轮细节。
func buildHashDetail(message []byte) hashDetail {
	padded := padMessage(message)
	blocks := splitBlocks(padded)
	state := sha256Init
	rounds := make([]roundState, 0, 64)
	scheduleWords := make([]string, 0, 64)
	for blockIndex, block := range blocks {
		w := scheduleBlock(block)
		if blockIndex == 0 {
			for _, word := range w {
				scheduleWords = append(scheduleWords, fmt.Sprintf("%08x", word))
			}
		}
		a, b, c, d := state[0], state[1], state[2], state[3]
		e, f, g, h := state[4], state[5], state[6], state[7]
		for i := 0; i < 64; i++ {
			s1 := bits.RotateLeft32(e, -6) ^ bits.RotateLeft32(e, -11) ^ bits.RotateLeft32(e, -25)
			ch := (e & f) ^ (^e & g)
			t1 := h + s1 + ch + sha256K[i] + w[i]
			s0 := bits.RotateLeft32(a, -2) ^ bits.RotateLeft32(a, -13) ^ bits.RotateLeft32(a, -22)
			maj := (a & b) ^ (a & c) ^ (b & c)
			t2 := s0 + maj
			h = g
			g = f
			f = e
			e = d + t1
			d = c
			c = b
			b = a
			a = t1 + t2
			if blockIndex == 0 {
				rounds = append(rounds, roundState{
					Round: i, W: w[i], K: sha256K[i], A: a, B: b, C: c, D: d, E: e, F: f, G: g, H: h, T1: t1, T2: t2,
				})
			}
		}
		state[0] += a
		state[1] += b
		state[2] += c
		state[3] += d
		state[4] += e
		state[5] += f
		state[6] += g
		state[7] += h
	}
	digest := encodeDigest(state)
	mutatedInput := mutateInput(message)
	mutatedDigest := hashDigest(mutatedInput)
	return hashDetail{
		Input:         string(message),
		InputHex:      hex.EncodeToString(message),
		PaddedHex:     hex.EncodeToString(padded),
		Blocks:        encodeBlocks(blocks),
		Schedule:      scheduleWords,
		Rounds:        rounds,
		Digest:        digest,
		MutatedInput:  string(mutatedInput),
		MutatedDigest: mutatedDigest,
		AvalancheBits: hammingDistanceHex(digest, mutatedDigest),
	}
}

// phaseByRound 根据当前轮次选择前端阶段标签。
func phaseByRound(round int) (int, string) {
	switch {
	case round == 0:
		return 0, "消息分块"
	case round < 16:
		return 1, "消息填充"
	case round < 63:
		return 2, "压缩轮函数"
	default:
		return 3, "哈希输出"
	}
}

// padMessage 按 SHA-256 规范追加 1 bit、零填充和消息长度。
func padMessage(message []byte) []byte {
	bitLen := uint64(len(message) * 8)
	padded := append([]byte{}, message...)
	padded = append(padded, 0x80)
	for (len(padded)+8)%64 != 0 {
		padded = append(padded, 0x00)
	}
	lenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBytes, bitLen)
	return append(padded, lenBytes...)
}

// splitBlocks 将填充后的消息按 512 bit 切分为消息块。
func splitBlocks(padded []byte) [][]byte {
	blocks := make([][]byte, 0, len(padded)/64)
	for offset := 0; offset < len(padded); offset += 64 {
		blocks = append(blocks, append([]byte{}, padded[offset:offset+64]...))
	}
	return blocks
}

// scheduleBlock 生成单个消息块的 64 个消息调度字。
func scheduleBlock(block []byte) [64]uint32 {
	var schedule [64]uint32
	for i := 0; i < 16; i++ {
		schedule[i] = binary.BigEndian.Uint32(block[i*4 : (i+1)*4])
	}
	for i := 16; i < 64; i++ {
		s0 := bits.RotateLeft32(schedule[i-15], -7) ^ bits.RotateLeft32(schedule[i-15], -18) ^ (schedule[i-15] >> 3)
		s1 := bits.RotateLeft32(schedule[i-2], -17) ^ bits.RotateLeft32(schedule[i-2], -19) ^ (schedule[i-2] >> 10)
		schedule[i] = schedule[i-16] + s0 + schedule[i-7] + s1
	}
	return schedule
}

// encodeDigest 将最终 8 个工作字拼接为十六进制摘要。
func encodeDigest(state [8]uint32) string {
	builder := strings.Builder{}
	for _, word := range state {
		builder.WriteString(fmt.Sprintf("%08x", word))
	}
	return builder.String()
}

// encodeBlocks 将消息块转成渲染层更适合展示的十六进制文本。
func encodeBlocks(blocks [][]byte) []string {
	result := make([]string, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, hex.EncodeToString(block))
	}
	return result
}

// mutateInput 对输入做最小扰动，用于生成雪崩效应对比样本。
func mutateInput(message []byte) []byte {
	if len(message) == 0 {
		return []byte{0x01}
	}
	result := append([]byte{}, message...)
	result[len(result)-1] ^= 0x01
	return result
}

// hashDigest 使用同一套真实压缩函数流程计算扰动后的摘要。
func hashDigest(message []byte) string {
	padded := padMessage(message)
	blocks := splitBlocks(padded)
	state := sha256Init
	for _, block := range blocks {
		w := scheduleBlock(block)
		a, b, c, d := state[0], state[1], state[2], state[3]
		e, f, g, h := state[4], state[5], state[6], state[7]
		for i := 0; i < 64; i++ {
			s1 := bits.RotateLeft32(e, -6) ^ bits.RotateLeft32(e, -11) ^ bits.RotateLeft32(e, -25)
			ch := (e & f) ^ (^e & g)
			t1 := h + s1 + ch + sha256K[i] + w[i]
			s0 := bits.RotateLeft32(a, -2) ^ bits.RotateLeft32(a, -13) ^ bits.RotateLeft32(a, -22)
			maj := (a & b) ^ (a & c) ^ (b & c)
			t2 := s0 + maj
			h = g
			g = f
			f = e
			e = d + t1
			d = c
			c = b
			b = a
			a = t1 + t2
		}
		state[0] += a
		state[1] += b
		state[2] += c
		state[3] += d
		state[4] += e
		state[5] += f
		state[6] += g
		state[7] += h
	}
	return encodeDigest(state)
}

// hammingDistanceHex 统计两个摘要之间的比特差异数。
func hammingDistanceHex(left string, right string) int {
	leftBytes, err := hex.DecodeString(left)
	if err != nil {
		return 0
	}
	rightBytes, err := hex.DecodeString(right)
	if err != nil {
		return 0
	}
	total := 0
	for index := range leftBytes {
		total += bits.OnesCount8(leftBytes[index] ^ rightBytes[index])
	}
	return total
}
