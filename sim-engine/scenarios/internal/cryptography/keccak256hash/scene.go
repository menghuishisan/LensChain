// 模块：sim-engine/scenarios/internal/cryptography/keccak256hash
// 文件职责：CRY-02 Keccak-256（以太坊 keccak256）哈希函数仿真场景的完整实现。
//
// SSOT 依据：06.md §4.3.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Keccak-f[1600] 24 轮置换 + sponge 构造，零依赖外部加密库：
//   · 状态：5×5 lane，每 lane 64 bit（共 1600 bit）；
//   · 容量 c=512 bit / 速率 r=1088 bit（136 字节）；
//   · 24 轮，每轮 5 步：θ / ρ / π / χ / ι（详 FIPS 202 §3.2）；
//   · 24 个轮常量 RC + 25 个旋转偏移 ρ-offsets；
//   · 填充：以太坊 keccak256 风格 multi-rate padding（首字节 0x01，尾字节 OR 0x80），
//     与 NIST SHA-3 标准的 0x06 padding 不同 —— 这是以太坊 / Solidity / EVM 选择。
//
// 教学决策（流水线型 P4）：
//   - 5 阶段流水线：input → padding → absorb → keccak-f(24 round) → output
//   - 真实 sponge 状态可视化（5×5 lane 的 16 进制 / 当前轮 RC / 当前 ρ offsets）

package keccak256hash

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"

	fw "github.com/lenschain/sim-engine/framework"
)

// =====================================================================
// 元信息常量
// =====================================================================

const (
	sceneCode      = "keccak256-hash"
	schemaVersion  = "v1.0.0"
	algorithmType  = "keccak256"
	rate           = 136 // r = 1088 bit
	capacity       = 64  // c = 512 bit
	totalRounds    = 24
	digestBits     = 256
	defaultMessage = "hello"

	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "hashes.keccak256"

	phaseIdxInput   = 0
	phaseIdxPadding = 1
	phaseIdxAbsorb  = 2
	phaseIdxPermute = 3
	phaseIdxOutput  = 4
)

var pipelineNodeIDs = []string{
	"phase-input", "phase-padding", "phase-absorb", "phase-permute", "phase-output",
}
var phaseLabels = []string{"消息输入", "0x01..0x80 填充", "Sponge 吸收", "Keccak-f 24 轮置换"}

// 24 个轮常量 RC[0..23]（FIPS 202 §3.2.5）。
var rcConstants = [24]uint64{
	0x0000000000000001, 0x0000000000008082, 0x800000000000808A, 0x8000000080008000,
	0x000000000000808B, 0x0000000080000001, 0x8000000080008081, 0x8000000000008009,
	0x000000000000008A, 0x0000000000000088, 0x0000000080008009, 0x000000008000000A,
	0x000000008000808B, 0x800000000000008B, 0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080, 0x000000000000800A, 0x800000008000000A,
	0x8000000080008081, 0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
}

// ρ 旋转偏移量（按 [x][y] 索引，FIPS 202 §3.2.2 表 2）。
var rotationOffsets = [5][5]int{
	{0, 36, 3, 41, 18},
	{1, 44, 10, 45, 2},
	{62, 6, 43, 15, 61},
	{28, 55, 25, 21, 56},
	{27, 20, 39, 8, 14},
}

// =====================================================================
// Keccak-f[1600] 自实现（24 轮 / 5 步 / 5×5 lane）
// =====================================================================

// keccakState 5×5 个 64-bit lane，共 1600 bit 状态。
type keccakState [5][5]uint64

// keccakF 执行 Keccak-f[1600] 全部 24 轮（θ → ρ → π → χ → ι）。
// roundCallback 用于教学演示：每轮结束抓取状态快照。
func keccakF(s *keccakState, roundCallback func(round int, snapshot keccakState)) {
	for r := 0; r < totalRounds; r++ {
		// θ：列奇偶性扩散
		var c [5]uint64
		for x := 0; x < 5; x++ {
			c[x] = s[x][0] ^ s[x][1] ^ s[x][2] ^ s[x][3] ^ s[x][4]
		}
		var d [5]uint64
		for x := 0; x < 5; x++ {
			d[x] = c[(x+4)%5] ^ bits.RotateLeft64(c[(x+1)%5], 1)
		}
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				s[x][y] ^= d[x]
			}
		}
		// ρ + π：旋转 + 重排
		var b keccakState
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				b[y][(2*x+3*y)%5] = bits.RotateLeft64(s[x][y], rotationOffsets[x][y])
			}
		}
		// χ：行内非线性变换
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				s[x][y] = b[x][y] ^ ((^b[(x+1)%5][y]) & b[(x+2)%5][y])
			}
		}
		// ι：异或轮常量
		s[0][0] ^= rcConstants[r]

		if roundCallback != nil {
			roundCallback(r, *s)
		}
	}
}

// absorbBlock 把 rate 字节块按 little-endian 8 字节 lane 吸收（XOR 入状态）。
func absorbBlock(s *keccakState, block []byte) {
	for i := 0; i < rate/8; i++ {
		x := i % 5
		y := i / 5
		var lane uint64
		for j := 0; j < 8; j++ {
			lane |= uint64(block[i*8+j]) << uint(8*j)
		}
		s[x][y] ^= lane
	}
}

// padKeccak256 对消息做以太坊 keccak256 风格填充：尾部追加 0x01 + 0x00... + 末字节 OR 0x80，
// 总长 ≡ 0 (mod rate)。
func padKeccak256(msg []byte) []byte {
	out := make([]byte, ((len(msg)/rate)+1)*rate)
	copy(out, msg)
	out[len(msg)] = 0x01
	out[len(out)-1] |= 0x80
	return out
}

// Sum256 是 keccak256 的导出别名，供同 scenarios/internal 子树下的兄弟场景包
// （如 evmexecution）复用以太坊风格 Keccak-256 算法，避免代码重复。
//
// 该 export 仅供 scenarios 内部使用（受 internal 包规则约束），
// 不构成对教师 SDK 的稳定 API。
func Sum256(msg []byte) [32]byte { return keccak256(msg) }

// keccak256 计算消息的 Keccak-256 摘要（以太坊风格）。
func keccak256(msg []byte) [32]byte {
	var state keccakState
	padded := padKeccak256(msg)
	for off := 0; off < len(padded); off += rate {
		absorbBlock(&state, padded[off:off+rate])
		keccakF(&state, nil)
	}
	// Squeeze：取前 32 字节（little-endian lane → byte stream）。
	var out [32]byte
	for i := 0; i < 32; i++ {
		laneIdx := i / 8
		x := laneIdx % 5
		y := laneIdx / 5
		out[i] = byte(state[x][y] >> uint(8*(i%8)))
	}
	return out
}

// keccak256WithRoundSnapshots 跑完整哈希并记录每轮状态（仅末块 24 轮）。
func keccak256WithRoundSnapshots(msg []byte) ([32]byte, [25]keccakState) {
	var state keccakState
	padded := padKeccak256(msg)
	totalBlocks := len(padded) / rate
	var roundStates [25]keccakState // [0]=吸收后未置换；[1..24]=每轮后
	for off := 0; off < len(padded); off += rate {
		absorbBlock(&state, padded[off:off+rate])
		if off == (totalBlocks-1)*rate {
			roundStates[0] = state
			keccakF(&state, func(r int, snapshot keccakState) {
				roundStates[r+1] = snapshot
			})
		} else {
			keccakF(&state, nil)
		}
	}
	var out [32]byte
	for i := 0; i < 32; i++ {
		laneIdx := i / 8
		x := laneIdx % 5
		y := laneIdx / 5
		out[i] = byte(state[x][y] >> uint(8*(i%8)))
	}
	return out, roundStates
}

// avalancheBits 256-bit 摘要的差异比特数。
func avalancheBits(a, b [32]byte) int {
	cnt := 0
	for i := 0; i < 32; i++ {
		cnt += bits.OnesCount8(a[i] ^ b[i])
	}
	return cnt
}

// flipBit 翻转字节切片指定 bit。
func flipBit(in []byte, bitIndex int) []byte {
	if bitIndex < 0 || bitIndex >= len(in)*8 {
		return append([]byte{}, in...)
	}
	out := append([]byte{}, in...)
	out[bitIndex/8] ^= 1 << uint(bitIndex%8)
	return out
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Input         string
	MutatedInput  string
	BitFlipped    int
	CurrentRound  int // 0..24（0=吸收后/未置换；24=完成）
	DisplayRound  int
	StateHex      [25]string // 25 lane 的当前快照 hex
	Digest        string
	MutatedDigest string
	AvalancheBits int
	PaddedBytes   int
	RoundStates   [25]keccakState
	LastError     string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{Input: defaultMessage, BitFlipped: -1}
	}
	d := s.Data
	st := snapState{
		Input:         fw.MapStr(d, "input", defaultMessage),
		MutatedInput:  fw.MapStr(d, "mutated_input", ""),
		BitFlipped:    fw.MapInt(d, "bit_flipped", -1),
		CurrentRound:  fw.MapInt(d, "current_round", 0),
		DisplayRound:  fw.MapInt(d, "display_round", 0),
		Digest:        fw.MapStr(d, "digest", ""),
		MutatedDigest: fw.MapStr(d, "mutated_digest", ""),
		AvalancheBits: fw.MapInt(d, "avalanche_bits", 0),
		PaddedBytes:   fw.MapInt(d, "padded_bytes", 0),
	}
	// state lane / round states 不通过 Data 持久化（体积大，每次 recompute 重建）
	st.recompute()
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = make(map[string]any, 12)
	}
	s.Data["input"] = st.Input
	s.Data["mutated_input"] = st.MutatedInput
	s.Data["bit_flipped"] = st.BitFlipped
	s.Data["current_round"] = st.CurrentRound
	s.Data["display_round"] = st.DisplayRound
	s.Data["digest"] = st.Digest
	s.Data["mutated_digest"] = st.MutatedDigest
	s.Data["avalanche_bits"] = st.AvalancheBits
	s.Data["padded_bytes"] = st.PaddedBytes
}

// recompute 基于 Input / MutatedInput 重算摘要、雪崩、24 轮状态快照。
func (st *snapState) recompute() {
	if st.Input == "" {
		st.Input = defaultMessage
	}
	digest, rs := keccak256WithRoundSnapshots([]byte(st.Input))
	st.Digest = hex.EncodeToString(digest[:])
	st.RoundStates = rs
	if st.MutatedInput != "" {
		mdigest := keccak256([]byte(st.MutatedInput))
		st.MutatedDigest = hex.EncodeToString(mdigest[:])
		st.AvalancheBits = avalancheBits(digest, mdigest)
	} else {
		st.MutatedDigest = ""
		st.AvalancheBits = 0
	}
	st.PaddedBytes = len(padKeccak256([]byte(st.Input)))
	st.refreshStateHex()
}

// refreshStateHex 把当前轮的 5×5 lane 转成 25 个 hex 字符串供前端展示。
func (st *snapState) refreshStateHex() {
	idx := clampRound(st.CurrentRound)
	s := st.RoundStates[idx]
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			st.StateHex[y*5+x] = fmt.Sprintf("%016x", s[x][y])
		}
	}
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "Keccak-256 哈希函数（以太坊）",
		Description:         "演示以太坊 keccak256：sponge 构造 + 24 轮 Keccak-f[1600] 置换 + 雪崩效应",
		Category:            fw.CategoryCryptography,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify},

		// v0.5 协议字段（详 AGENTS.md §0.7.1 C10 / C29 / C37）。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"hashes.keccak256.input",
			"hashes.keccak256.digest_hex",
			"hashes.keccak256.mutated_digest_hex",
			"hashes.keccak256.avalanche_bits",
		},

		DefaultParams: func() map[string]any { return map[string]any{"input": defaultMessage} },
		DefaultState:  defaultState,
		Interaction:   interactionDefinition,
		Init:          initScene,
		Step:          stepScene,
		HandleAction:  handleAction,
	}
}

func defaultState() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data:      map[string]any{"input": defaultMessage, "bit_flipped": -1},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_input", Label: "设置输入消息",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "message", Type: fw.FieldString, Label: "消息（UTF-8）", Required: true, Default: defaultMessage},
				},
				WritesOwnedFields: []string{"hashes.keccak256.input", "hashes.keccak256.digest_hex"},
				LinkOwnerFields:   []string{"hashes.keccak256.input", "hashes.keccak256.digest_hex"},
			},
			{
				ActionCode: "step_round", Label: "推进 1 轮置换",
				Description: "执行下一轮 Keccak-f[1600]（共 24 轮 5 步）",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode: "step_to_round", Label: "跳到指定轮",
				Category: fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "target_round", Type: fw.FieldNumber, Label: "目标轮次", Required: true, Default: 0, Min: 0, Max: totalRounds, Step: 1},
				},
			},
			{
				ActionCode: "mutate_input_bit", Label: "翻转输入位",
				Description: "演示 Keccak-256 的雪崩效应",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "bit_index", Type: fw.FieldNumber, Label: "Bit 位号", Required: true, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{
					"hashes.keccak256.mutated_digest_hex",
					"hashes.keccak256.avalanche_bits",
				},
				LinkOwnerFields: []string{
					"hashes.keccak256.mutated_digest_hex",
					"hashes.keccak256.avalanche_bits",
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_set_demo_input",
				Label:         "教师设置演示输入",
				Description:   "仅教师可用，设置演示输入用于教学展示",
				Category:      fw.ActionParamTune,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师设置演示输入"},
				},
			},
			fw.BroadcastHintAction(),
		},
	}
}

// =====================================================================
// 钩子
// =====================================================================

func initScene(state *fw.SceneState, in fw.InitInput) (fw.RenderEnvelope, error) {
	st := loadState(state)
	if v, ok := in.Params["input"].(string); ok && v != "" {
		st.Input = v
	}
	st.MutatedInput = ""
	st.BitFlipped = -1
	st.CurrentRound = 0
	st.DisplayRound = 0
	st.recompute()
	saveState(state, st)
	state.Phase = "permuting"

	env := buildEnvelope(st, "init", "首帧：消息分块 → 填充 → 吸收 → 等待置换", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", fmt.Sprintf("Round %d / %d", st.DisplayRound, totalRounds), false)
	return fw.StepOutput{Render: env}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "set_input":
		msg := fw.MapStr(in.Params, "message", defaultMessage)
		if msg == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "message 不能为空"}, nil
		}
		st.Input = msg
		st.MutatedInput = ""
		st.BitFlipped = -1
		st.CurrentRound = 0
		st.DisplayRound = 0
		st.recompute()
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_input", "重新计算 Keccak-256（已重置到第 0 轮）", true)
		appendSetInputMicroSteps(&out.Render)
		publishOwnerSubtree(&out.Render, st)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_round":
		if st.CurrentRound >= totalRounds {
			out.Render = buildEnvelope(st, "step_round", "已到第 24 轮（终态）", false)
			return out, nil
		}
		st.CurrentRound++
		st.DisplayRound = st.CurrentRound
		st.refreshStateHex()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_round", fmt.Sprintf("推进到第 %d 轮（θ→ρ→π→χ→ι）", st.CurrentRound), false)
		appendStepRoundMicroSteps(&out.Render, st.CurrentRound)
		return out, nil

	case "step_to_round":
		target := fw.MapInt(in.Params, "target_round", 0)
		if target < 0 || target > totalRounds {
			return fw.ActionOutput{Success: false, ErrorMessage: "target_round 越界 [0,24]"}, nil
		}
		st.CurrentRound = target
		st.DisplayRound = target
		st.refreshStateHex()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_to_round", fmt.Sprintf("跳转到第 %d 轮", target), false)
		return out, nil

	case "mutate_input_bit":
		idx := fw.MapInt(in.Params, "bit_index", 0)
		if idx < 0 || idx >= len([]byte(st.Input))*8 {
			return fw.ActionOutput{Success: false, ErrorMessage: "bit_index 超出输入比特长度"}, nil
		}
		st.MutatedInput = string(flipBit([]byte(st.Input), idx))
		st.BitFlipped = idx
		st.recompute()
		saveState(state, st)
		out.Render = buildEnvelope(st, "mutate_input_bit", fmt.Sprintf("翻转第 %d 位 → 雪崩 %d/256 比特", idx, st.AvalancheBits), false)
		appendMutateMicroSteps(&out.Render, idx, st.AvalancheBits)
		publishOwnerSubtree(&out.Render, st)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "teacher_set_demo_input":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师设置演示输入"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-hint-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		st.MutatedInput = ""
		st.BitFlipped = -1
		st.CurrentRound = 0
		st.DisplayRound = 0
		st.refreshStateHex()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置到第 0 轮", false)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func activePhase(st snapState) int {
	switch {
	case st.Input == "":
		return phaseIdxInput
	case st.PaddedBytes == 0:
		return phaseIdxPadding
	case st.CurrentRound == 0:
		return phaseIdxAbsorb
	case st.CurrentRound < totalRounds:
		return phaseIdxPermute
	default:
		return phaseIdxOutput
	}
}

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	active := activePhase(st)
	prims := make([]fw.Primitive, 0, 40)

	// 1) 流水线
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))

	// 2) 5 节点
	for i, id := range pipelineNodeIDs {
		status := "normal"
		if i == active {
			status = "active"
		}
		role := []string{"input", "padding", "absorb", "permute", "output"}[i]
		label := []string{"输入", "填充 0x01..0x80", "Sponge 吸收", "Keccak-f 24 轮", "256-bit 摘要"}[i]
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}

	// 3) 4 边
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		anim := ""
		if i == active-1 || (i == active && active == phaseIdxPermute) {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-%d-%d", i, i+1), pipelineNodeIDs[i], pipelineNodeIDs[i+1], "solid", anim))
	}

	// 4) 阶段进度
	phaseProgress := 0.0
	switch active {
	case phaseIdxPadding:
		phaseProgress = 0.25
	case phaseIdxAbsorb:
		phaseProgress = 0.5
	case phaseIdxPermute:
		phaseProgress = 0.5 + 0.5*float64(st.CurrentRound)/float64(totalRounds)
	case phaseIdxOutput:
		phaseProgress = 1.0
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, minInt(active, len(phaseLabels)-1), phaseProgress))

	// 5) 24 轮进度环
	prims = append(prims, fw.PrimRing("ring-rounds", totalRounds, st.DisplayRound, fmt.Sprintf("Round %d / %d", st.DisplayRound, totalRounds)))

	// 6) 5×5 lane 矩阵布局（默认无单元格尺寸，由前端决定）
	prims = append(prims, fw.PrimMatrixLayout("state-matrix", 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			cellID := fmt.Sprintf("lane-%d-%d", x, y)
			role := "lane"
			if x == 0 && y == 0 {
				role = "lane-rc" // RC 异或目标
			}
			prims = append(prims, fw.PrimGridCell(cellID, y, x, st.StateHex[y*5+x], role))
		}
	}

	// 7) 5 步阶段标签（θ ρ π χ ι）
	prims = append(prims, fw.PrimMathFormula("step-formulas",
		`\theta:\ C[x]=\bigoplus_y A[x,y];\ \rho:\ \mathrm{rot}\ \mathrm{by}\ r[x,y];\ \pi:\ \mathrm{permute};\ \chi:\ A\oplus(\bar A_{x{+}1}\!\wedge A_{x{+}2});\ \iota:\ A_{0,0}\oplus RC[r]`, false))

	// 8) 当前轮 RC（轮常量）
	rcIdx := clampRound(st.CurrentRound) - 1
	if rcIdx < 0 {
		rcIdx = 0
	}
	prims = append(prims, fw.PrimCodeBlock("cb-rc",
		fmt.Sprintf("RC[%d] = %016x", rcIdx, rcConstants[rcIdx]),
		"text", nil, 2))

	// 9) 输入 / 填充 / 摘要 / 雪崩
	prims = append(prims, fw.PrimCodeBlock("cb-input",
		fmt.Sprintf("%s\n(%d 字节 / %d 比特)", st.Input, len([]byte(st.Input)), len([]byte(st.Input))*8),
		"text", nil, 6))
	prims = append(prims, fw.PrimCodeBlock("cb-padded",
		hexBlocks(padKeccak256([]byte(st.Input))), "text", nil, 6))
	prims = append(prims, fw.PrimCodeBlock("cb-digest", st.Digest, "text", nil, 4))
	prims = append(prims, fw.PrimCodeBlock("cb-avalanche",
		formatAvalanche(st.Digest, st.MutatedDigest, st.AvalancheBits, st.BitFlipped),
		"text", nil, 6))

	// 10) 动效
	prims = append(prims, fw.PrimPulse("pulse-rc", "cb-rc", "info", 1200))
	prims = append(prims, fw.PrimShiftAnimation("anim-state-shift", "state-matrix", "right", 0.2, 600))

	// 11) glow 当前阶段
	if active >= 0 && active < len(pipelineNodeIDs) {
		prims = append(prims, fw.PrimGlow("glow-active", pipelineNodeIDs[active], "info", 0.8))
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto-verify", linkGroupCryptoVerify, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Keccak-256 错误", st.LastError, "scene", "请检查输入", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"input":          st.Input,
		"input_bits":     len([]byte(st.Input)) * 8,
		"current_round":  st.CurrentRound,
		"display_round":  st.DisplayRound,
		"total_rounds":   totalRounds,
		"rate_bytes":     rate,
		"capacity_bits":  capacity * 8,
		"digest":         st.Digest,
		"mutated_input":  st.MutatedInput,
		"mutated_digest": st.MutatedDigest,
		"avalanche_bits": st.AvalancheBits,
		"padded_bytes":   st.PaddedBytes,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendSetInputMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "set-1", Label: "重置 1600-bit 状态", DurationMs: 400, HighlightIDs: []string{pipelineNodeIDs[phaseIdxInput], "cb-input"}, ParentPhase: "absorb"},
		{ID: "set-2", Label: "Keccak 风格填充 (0x01...0x80)", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxPadding], "cb-padded"}, FirePrimitives: []string{"glow-active"}},
		{ID: "set-3", Label: "Sponge 吸收（XOR 进 5×5 lane）", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxAbsorb], "state-matrix"}, FirePrimitives: []string{"anim-state-shift"}},
		{ID: "set-4", Label: "Keccak-f 24 轮置换", DurationMs: 800, HighlightIDs: []string{pipelineNodeIDs[phaseIdxPermute], "ring-rounds"}, FirePrimitives: []string{"pulse-rc"}},
		{ID: "set-5", Label: "Squeeze 输出 256-bit 摘要", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxOutput], "cb-digest"}, IsLinkTrigger: true},
	}
}

func appendStepRoundMicroSteps(env *fw.RenderEnvelope, round int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: fmt.Sprintf("r%d-1", round), Label: fmt.Sprintf("Round %d-θ：列奇偶扩散", round), DurationMs: 350, HighlightIDs: []string{"state-matrix", "step-formulas"}},
		{ID: fmt.Sprintf("r%d-2", round), Label: fmt.Sprintf("Round %d-ρπ：旋转 + 重排", round), DurationMs: 350, HighlightIDs: []string{"state-matrix"}, FirePrimitives: []string{"anim-state-shift"}},
		{ID: fmt.Sprintf("r%d-3", round), Label: fmt.Sprintf("Round %d-χ：行内非线性", round), DurationMs: 350, HighlightIDs: []string{"state-matrix"}},
		{ID: fmt.Sprintf("r%d-4", round), Label: fmt.Sprintf("Round %d-ι：异或 RC", round), DurationMs: 400, HighlightIDs: []string{"lane-0-0", "cb-rc"}, FirePrimitives: []string{"pulse-rc"}},
	}
}

func appendMutateMicroSteps(env *fw.RenderEnvelope, bitIdx, av int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "mut-1", Label: fmt.Sprintf("翻转输入第 %d 位", bitIdx), DurationMs: 500, HighlightIDs: []string{"cb-input"}, FirePrimitives: []string{"glow-active"}},
		{ID: "mut-2", Label: "重新执行 24 轮 Keccak-f", DurationMs: 800, HighlightIDs: []string{pipelineNodeIDs[phaseIdxPermute], "ring-rounds", "state-matrix"}, FirePrimitives: []string{"anim-state-shift"}},
		{ID: "mut-3", Label: fmt.Sprintf("差异比特 %d / %d（雪崩效应）", av, digestBits), DurationMs: 700, HighlightIDs: []string{"cb-avalanche", "cb-digest"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	// LinkTrigger 带锚点（§0.7.1 C18）。
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             fmt.Sprintf("keccak256-publish-r%d", st.DisplayRound),
		SourceScene:    sceneCode,
		SourceAction:   "publish_digest",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"hashes.keccak256.digest_hex", "hashes.keccak256.avalanche_bits"},
		Payload:        map[string]any{"digest_hex": st.Digest, "avalanche_bits": st.AvalancheBits},
		SourceAnchorID: "keccak256-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "hashes.keccak256.digest_hex", "hashes.keccak256.avalanche_bits")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"hashes": map[string]any{
			"keccak256": map[string]any{
				"input":              st.Input,
				"digest_hex":         st.Digest,
				"mutated_digest_hex": st.MutatedDigest,
				"avalanche_bits":     st.AvalancheBits,
				"current_round":      st.CurrentRound,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func toFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case uint32:
		return float64(t)
	case uint64:
		return float64(t)
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampRound(t int) int {
	if t < 0 {
		return 0
	}
	if t > totalRounds {
		return totalRounds
	}
	return t
}

func hexBlocks(b []byte) string {
	out := ""
	for i := 0; i < len(b); i += 16 {
		end := i + 16
		if end > len(b) {
			end = len(b)
		}
		if i > 0 {
			out += "\n"
		}
		out += hex.EncodeToString(b[i:end])
		if i/16 >= 5 && i+16 < len(b) {
			out += "\n…"
			break
		}
	}
	return out
}

func formatAvalanche(orig, mutated string, av, flippedBit int) string {
	if mutated == "" {
		return "（未触发雪崩演示，使用 mutate_input_bit）"
	}
	return fmt.Sprintf("原  : %s\n翻转: %s\n翻转位: %d\n差异: %d / %d 比特", orig, mutated, flippedBit, av, digestBits)
}
