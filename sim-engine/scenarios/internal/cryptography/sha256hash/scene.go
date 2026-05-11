// 模块：sim-engine/scenarios/internal/cryptography/sha256hash
// 文件职责：CRY-01 SHA-256 哈希函数仿真场景的完整实现。
//
// SSOT 依据：
//   - docs/modules/04-实验环境/06-可视化仿真引擎设计.md §4.3.1（SHA-256 场景需求）
//     §3.2（47 原语 schema）/ §3.10.4（动画调度器）/ §6.2（RenderEnvelope）
//     §6.3（ActionDef trigger/category 三枚举）/ §8.3（owner-based 联动）
//   - SIM_ENGINE_SCENE_AUDIT.md / SIM_ENGINE_REWORK_PROPOSAL.md（场景拼装演示要求）
//   - sim-engine/AGENTS.md §0（防摇摆基线）/ §6（教学决策层规范）
//
// 算法实现：
//   - 严格按 RFC 6234 / FIPS 180-4 实现 SHA-256：
//     · 8 个 IV 初值 sha256Init[0..7]
//     · 64 个轮常量 sha256K[0..63]
//     · 真填充：尾部 1-bit + 0-pad + 64-bit big-endian 长度，使总长 ≡ 0 (mod 512)
//     · 真消息调度 W[0..63]：W[0..15]=块 32-bit 大端字；
//                          W[t]=σ₁(W[t-2])+W[t-7]+σ₀(W[t-15])+W[t-16]
//     · 真 64 轮压缩：T₁=h+Σ₁(e)+Ch(e,f,g)+K[t]+W[t]；T₂=Σ₀(a)+Maj(a,b,c)
//     · 雪崩统计：翻转输入指定 bit → 重新执行 SHA-256 → popcount(原 ⊕ 翻转) 256 位差异
//
// 教学决策层职责（详 AGENTS.md §0.2 判别试纸）：
//   - 输出 RenderEnvelope（流水线型 P4 布局：stack horizontal）+ 语义槽位（status/role/color_role）
//   - 不写像素坐标 / 颜色 RGB / 动画曲线（这些由 renderers 皮肤决定）
//   - SceneState 完整算法状态前端不可见；RenderEnvelope.Data 主动暴露 12 个侧栏指标
//   - 联动写入 hashes.sha256.* owner 子树（v0.5 owner-based）

package sha256hash

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"

	fw "github.com/lenschain/sim-engine/framework"
)

// =====================================================================
// 场景元信息常量
// =====================================================================

const (
	sceneCode      = "sha256-hash"
	schemaVersion  = "v1.0.0"
	algorithmType  = "sha256"
	totalRounds    = 64
	digestBits     = 256
	defaultMessage = "abc"

	// owner-based 联动子树根（详 AGENTS.md §0.6 + 06.md §8.3）
	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "hashes.sha256"

	// 阶段索引（与下方 phaseLabels / pipelineNodeIDs 对齐）
	phaseIdxInput    = 0
	phaseIdxPadding  = 1
	phaseIdxSchedule = 2
	phaseIdxCompress = 3
	phaseIdxOutput   = 4
)

// 流水线 5 节点的稳定 ID（跨 tick 不变，保证前端原语 diff 正确）。
var pipelineNodeIDs = []string{
	"phase-input", "phase-padding", "phase-schedule", "phase-compress", "phase-output",
}

// 4 阶段进度条 phases 标签（与 phaseIdx* 索引对齐，但仅 4 阶段，最末态由 progress=1 表达）。
var phaseLabels = []string{
	"消息分块", "比特填充", "调度字 W[64]", "64 轮压缩",
}

// SHA-256 IV 初值（FIPS 180-4 §5.3.3）。
var sha256Init = [8]uint32{
	0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
	0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
}

// SHA-256 64 轮常量 K[0..63]（FIPS 180-4 §4.2.2）。
var sha256K = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

// =====================================================================
// SHA-256 算法实现（真完整，零依赖 crypto/sha256）
// =====================================================================

// 6 个位运算辅助函数（FIPS 180-4 §4.1.2）。
func ch(x, y, z uint32) uint32  { return (x & y) ^ (^x & z) }
func maj(x, y, z uint32) uint32 { return (x & y) ^ (x & z) ^ (y & z) }
func bigSigma0(x uint32) uint32 {
	return bits.RotateLeft32(x, -2) ^ bits.RotateLeft32(x, -13) ^ bits.RotateLeft32(x, -22)
}
func bigSigma1(x uint32) uint32 {
	return bits.RotateLeft32(x, -6) ^ bits.RotateLeft32(x, -11) ^ bits.RotateLeft32(x, -25)
}
func smallSigma0(x uint32) uint32 {
	return bits.RotateLeft32(x, -7) ^ bits.RotateLeft32(x, -18) ^ (x >> 3)
}
func smallSigma1(x uint32) uint32 {
	return bits.RotateLeft32(x, -17) ^ bits.RotateLeft32(x, -19) ^ (x >> 10)
}

// padMessage 严格按 RFC 6234 §4.1 对消息做填充，返回 512-bit 倍数字节。
// 步骤：尾部 1 个 0x80 → 补 0x00 至 (len mod 64 == 56) → 末尾 8 字节 big-endian 比特长度。
func padMessage(msg []byte) []byte {
	bitLen := uint64(len(msg)) * 8
	out := make([]byte, 0, len(msg)+1+64)
	out = append(out, msg...)
	out = append(out, 0x80)
	for len(out)%64 != 56 {
		out = append(out, 0x00)
	}
	var bitLenBytes [8]byte
	binary.BigEndian.PutUint64(bitLenBytes[:], bitLen)
	out = append(out, bitLenBytes[:]...)
	return out
}

// scheduleBlock 按 §6.2 生成 64 字调度数组 W[0..63]（一个 512-bit 块）。
func scheduleBlock(block []byte) [64]uint32 {
	var w [64]uint32
	for i := 0; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(block[i*4 : i*4+4])
	}
	for t := 16; t < 64; t++ {
		w[t] = smallSigma1(w[t-2]) + w[t-7] + smallSigma0(w[t-15]) + w[t-16]
	}
	return w
}

// compressBlock 按 §6.2 对当前 hash 做 64 轮压缩，返回新 hash。
// roundCallback 可选，用于演示模式下对每一轮发布 a..h 寄存器快照。
func compressBlock(h0 [8]uint32, w [64]uint32, roundCallback func(t int, a, b, c, d, e, f, g, h uint32, t1, t2 uint32)) [8]uint32 {
	a, b, c, d, e, f, g, h := h0[0], h0[1], h0[2], h0[3], h0[4], h0[5], h0[6], h0[7]
	for t := 0; t < 64; t++ {
		t1 := h + bigSigma1(e) + ch(e, f, g) + sha256K[t] + w[t]
		t2 := bigSigma0(a) + maj(a, b, c)
		if roundCallback != nil {
			roundCallback(t, a, b, c, d, e, f, g, h, t1, t2)
		}
		h = g
		g = f
		f = e
		e = d + t1
		d = c
		c = b
		b = a
		a = t1 + t2
	}
	return [8]uint32{h0[0] + a, h0[1] + b, h0[2] + c, h0[3] + d, h0[4] + e, h0[5] + f, h0[6] + g, h0[7] + h}
}

// Sum256 是 sha256Full 的导出别名，供同 scenarios/internal 子树下的兄弟场景包
// （如 merkletree、blockchainstructure）复用 SHA-256 算法，避免代码重复。
//
// 该 export 仅供 scenarios 内部使用（受 internal 包规则约束，外部包无法 import），
// 不构成对教师 SDK 的稳定 API；教师场景应使用 sdk/go/scenario 暴露的工具或自行引入哈希。
func Sum256(msg []byte) [32]byte { return sha256Full(msg) }

// sha256Full 计算消息的完整 SHA-256 摘要（多块迭代 + 最终值串接）。
func sha256Full(msg []byte) [32]byte {
	padded := padMessage(msg)
	hh := sha256Init
	for off := 0; off < len(padded); off += 64 {
		w := scheduleBlock(padded[off : off+64])
		hh = compressBlock(hh, w, nil)
	}
	var digest [32]byte
	for i, v := range hh {
		binary.BigEndian.PutUint32(digest[i*4:], v)
	}
	return digest
}

// avalancheBits 统计两个 32 字节摘要的差异比特数（256 bit）。
func avalancheBits(a, b [32]byte) int {
	cnt := 0
	for i := 0; i < 32; i++ {
		cnt += bits.OnesCount8(a[i] ^ b[i])
	}
	return cnt
}

// flipBit 翻转字节切片的指定 bit（0 = 第 0 字节最低位；按位 little-endian 内字节序）。
func flipBit(in []byte, bitIndex int) []byte {
	if bitIndex < 0 || bitIndex >= len(in)*8 {
		return append([]byte{}, in...)
	}
	out := append([]byte{}, in...)
	byteIdx := bitIndex / 8
	bitInByte := uint(bitIndex % 8)
	out[byteIdx] ^= 1 << bitInByte
	return out
}

// =====================================================================
// 场景内部状态（仅写入 SceneState.Data；前端不可见，详 AGENTS.md §0.4）
// =====================================================================

// snapState 序列化算法状态到 SceneState.Data。
type snapState struct {
	Input         string `json:"input"`
	MutatedInput  string `json:"mutated_input,omitempty"`
	BitFlipped    int    `json:"bit_flipped"`
	CurrentRound  int    `json:"current_round"` // 真实算法轮次：0=未开始 / 1..64=已完成 t-1
	DisplayRound  int    `json:"display_round"` // 学生侧显示的展示轮次（同上，便于皮肤展示）
	Regs          [8]uint32
	W             [64]uint32
	Digest        string `json:"digest"` // 完整摘要（执行完所有轮次后）
	MutatedDigest string `json:"mutated_digest,omitempty"`
	AvalancheBits int    `json:"avalanche_bits"`
	PaddedBlocks  int    `json:"padded_blocks"`
	LastError     string `json:"last_error,omitempty"`
}

// loadState 从 SceneState 解码业务状态；缺省时返回零值。
func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{Input: defaultMessage}
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
		PaddedBlocks:  fw.MapInt(d, "padded_blocks", 0),
	}
	if regs, ok := d["regs"].([]any); ok && len(regs) == 8 {
		for i, v := range regs {
			st.Regs[i] = uint32(toFloat(v))
		}
	}
	if ws, ok := d["w"].([]any); ok && len(ws) == 64 {
		for i, v := range ws {
			st.W[i] = uint32(toFloat(v))
		}
	}
	return st
}

// saveState 把业务状态回写到 SceneState.Data。
func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = make(map[string]any, 16)
	}
	s.Data["input"] = st.Input
	s.Data["mutated_input"] = st.MutatedInput
	s.Data["bit_flipped"] = st.BitFlipped
	s.Data["current_round"] = st.CurrentRound
	s.Data["display_round"] = st.DisplayRound
	s.Data["digest"] = st.Digest
	s.Data["mutated_digest"] = st.MutatedDigest
	s.Data["avalanche_bits"] = st.AvalancheBits
	s.Data["padded_blocks"] = st.PaddedBlocks
	regs := make([]any, 8)
	for i, v := range st.Regs {
		regs[i] = float64(v)
	}
	s.Data["regs"] = regs
	ws := make([]any, 64)
	for i, v := range st.W {
		ws[i] = float64(v)
	}
	s.Data["w"] = ws
}

// recompute 基于 Input / MutatedInput 重算所有派生量（摘要 / 雪崩 / 当前块 W / 寄存器初值）。
// 默认 CurrentRound 复位为 0；调用方按需推进。
func (st *snapState) recompute() {
	if st.Input == "" {
		st.Input = defaultMessage
	}
	digest := sha256Full([]byte(st.Input))
	st.Digest = hex.EncodeToString(digest[:])
	if st.MutatedInput != "" {
		mdigest := sha256Full([]byte(st.MutatedInput))
		st.MutatedDigest = hex.EncodeToString(mdigest[:])
		st.AvalancheBits = avalancheBits(digest, mdigest)
	} else {
		st.MutatedDigest = ""
		st.AvalancheBits = 0
	}
	padded := padMessage([]byte(st.Input))
	st.PaddedBlocks = len(padded) / 64
	// 当前展示用的 W / 初始寄存器：取第 0 块（多块场景可拓展）。
	if st.PaddedBlocks > 0 {
		st.W = scheduleBlock(padded[:64])
	}
	st.Regs = sha256Init
	st.CurrentRound = 0
	st.DisplayRound = 0
}

// stepOneRound 把"已完成轮次"推进 1（最多 64）。
// 内部对第 0 块从头执行 t = CurrentRound 这一轮，得到推进后的 a..h 快照。
func (st *snapState) stepOneRound() {
	if st.CurrentRound >= totalRounds {
		return
	}
	regs := sha256Init
	target := st.CurrentRound + 1
	round := 0
	regs = compressBlock(regs, st.W, func(t int, a, b, c, d, e, f, g, h uint32, t1, t2 uint32) {
		if t+1 == target {
			// 抓取这一轮"开始时"的快照写入寄存器（演示用）。
			st.Regs = [8]uint32{a, b, c, d, e, f, g, h}
			round = t
		}
	})
	_ = regs
	_ = round
	st.CurrentRound = target
	st.DisplayRound = target
}

// =====================================================================
// 场景定义（实现 framework.Definition 5 个钩子）
// =====================================================================

// Definition 返回 CRY-01 SHA-256 场景的完整 framework.Definition。
// 注册路径：scenarios/internal/catalog/definitions.go §4.3 密码学运算。
func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "SHA-256 哈希函数",
		Description:         "演示 SHA-256 标准实现：消息分块、填充、64 轮压缩、雪崩效应",
		Category:            fw.CategoryCryptography,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess, // 学生触发 → 才推进
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify},

		// v0.5 协议字段（详 AGENTS.md §0.7.1 C10 / C29 / C37 / C_SupportsMultiActor）。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"hashes.sha256.input",
			"hashes.sha256.digest_hex",
			"hashes.sha256.mutated_digest_hex",
			"hashes.sha256.avalanche_bits",
		},

		DefaultParams: func() map[string]any { return map[string]any{"input": defaultMessage} },
		DefaultState:  defaultState,
		Interaction:   interactionDefinition,
		Init:          initScene,
		Step:          stepScene,
		HandleAction:  handleAction,
	}
}

// defaultState 构造场景初始 SceneState（CurrentRound=0，待 Init 调 recompute 填充派生量）。
func defaultState() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data:      map[string]any{"input": defaultMessage, "bit_flipped": -1},
	}
}

// interactionDefinition 返回 5 个 ActionDef（详 AGENTS.md §0.6 三 trigger 枚举）。
func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode:  "set_input",
				Label:       "设置输入消息",
				Description: "重置场景并使用新输入计算 SHA-256",
				Category:    fw.ActionParamTune,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "message", Type: fw.FieldString, Label: "消息（UTF-8）", Required: true, Default: defaultMessage},
				},
				WritesOwnedFields: []string{"hashes.sha256.input", "hashes.sha256.digest_hex"},
				LinkOwnerFields:   []string{"hashes.sha256.input", "hashes.sha256.digest_hex"},
			},
			{
				ActionCode:  "step_round",
				Label:       "推进 1 轮",
				Description: "执行下一轮 SHA-256 压缩函数（共 64 轮）",
				Category:    fw.ActionPrimary,
				Trigger:       fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode:  "step_to_round",
				Label:       "跳到指定轮",
				Description: "跳转到第 N 轮压缩（0~64）",
				Category:    fw.ActionObserve,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "target_round", Type: fw.FieldNumber, Label: "目标轮次", Required: true, Default: 0, Min: 0, Max: totalRounds, Step: 1},
				},
			},
			{
				ActionCode:  "mutate_input_bit",
				Label:       "翻转输入位",
				Description: "翻转输入消息指定 bit，演示雪崩效应",
				Category:    fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "bit_index", Type: fw.FieldNumber, Label: "Bit 位号（0=输入第 0 字节最低位）", Required: true, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{
					"hashes.sha256.mutated_digest_hex",
					"hashes.sha256.avalanche_bits",
				},
				LinkOwnerFields: []string{
					"hashes.sha256.mutated_digest_hex",
					"hashes.sha256.avalanche_bits",
				},
			},
			{
				ActionCode:    "reset",
				Label:         "重置",
				Description:   "回到第 0 轮，清除翻转输入",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			// §0.7.5 教师专属 broadcast_hint（标准实现，所有场景必备）。
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

// initScene 首帧 RenderEnvelope。计算输入摘要 + 寄存器初值，发布 hashes.sha256.* owner 子树。
func initScene(state *fw.SceneState, in fw.InitInput) (fw.RenderEnvelope, error) {
	st := loadState(state)
	if v, ok := in.Params["input"].(string); ok && v != "" {
		st.Input = v
	}
	st.MutatedInput = ""
	st.BitFlipped = -1
	st.recompute()
	saveState(state, st)
	state.Phase = "compressing"

	env := buildEnvelope(st, "init", "首帧：消息分块 → 填充 → 调度字 → 等待压缩", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

// stepScene reactive 模式下不做时间推进；返回当前完整快照供前端订阅 / 教师监控。
func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", fmt.Sprintf("当前轮 %d / %d", st.DisplayRound, totalRounds), false)
	return fw.StepOutput{Render: env}, nil
}

// handleAction 路由所有 ActionDef，更新状态并构造 RenderEnvelope（含 MicroSteps）。
//
// 入口先做两件协议级事：
//  1. EnsureActorBucket：保证多 actor 桶就绪（详 AGENTS.md §0.7.3）
//  2. HandleBroadcastHint：拦截教师广播提示，输出 annotation 原语
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
		st.recompute()
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_input", "重新计算 SHA-256（已重置到第 0 轮）", true)
		appendSetInputMicroSteps(&out.Render)
		publishOwnerSubtree(&out.Render, st)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_round":
		if st.CurrentRound >= totalRounds {
			out.Render = buildEnvelope(st, "step_round", "已到第 64 轮（终态），请使用 reset 或 set_input", false)
			return out, nil
		}
		st.stepOneRound()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_round", fmt.Sprintf("推进到第 %d 轮", st.CurrentRound), false)
		appendStepRoundMicroSteps(&out.Render, st.CurrentRound)
		return out, nil

	case "step_to_round":
		target := fw.MapInt(in.Params, "target_round", 0)
		if target < 0 || target > totalRounds {
			return fw.ActionOutput{Success: false, ErrorMessage: "target_round 越界 [0,64]"}, nil
		}
		st.CurrentRound = 0
		for st.CurrentRound < target {
			st.stepOneRound()
		}
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
		st.Regs = sha256Init
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置到第 0 轮", false)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造：流水线型 P4 布局 + 12 字段 Data
// =====================================================================

// activePhase 根据当前轮次决定流水线高亮的阶段。
func activePhase(st snapState) int {
	switch {
	case st.Input == "":
		return phaseIdxInput
	case st.PaddedBlocks == 0:
		return phaseIdxPadding
	case st.CurrentRound == 0:
		return phaseIdxSchedule
	case st.CurrentRound < totalRounds:
		return phaseIdxCompress
	default:
		return phaseIdxOutput
	}
}

// buildEnvelope 装配 RenderEnvelope（流水线型 P4：stack horizontal + 5 节点 + 4 边）。
// reason 仅作内部诊断；fullSnapshot 为 true 时声明为快照帧。
func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	active := activePhase(st)
	prims := make([]fw.Primitive, 0, 32)

	// 1) 流水线水平 stack（教学决策：5 阶段从左到右，无绝对坐标，由前端响应式排列）
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))

	// 2) 5 个流水线节点（默认无坐标，参与父 stack 推导位置）
	for i, id := range pipelineNodeIDs {
		status := "normal"
		if i == active {
			status = "active"
		}
		role := []string{"input", "padding", "schedule", "compress", "output"}[i]
		label := []string{"输入", "填充", "调度 W[64]", "64 轮压缩", "256-bit 摘要"}[i]
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}

	// 3) 4 条数据流边（节点间）
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		anim := ""
		if i == active-1 || (i == active && active == phaseIdxCompress) {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(
			fmt.Sprintf("edge-%d-%d", i, i+1),
			pipelineNodeIDs[i], pipelineNodeIDs[i+1],
			"solid", anim,
		))
	}

	// 4) 4 阶段进度条
	phaseProgress := 0.0
	switch active {
	case phaseIdxPadding:
		phaseProgress = 0.25
	case phaseIdxSchedule:
		phaseProgress = 0.5
	case phaseIdxCompress:
		phaseProgress = 0.5 + 0.5*float64(st.CurrentRound)/float64(totalRounds)
	case phaseIdxOutput:
		phaseProgress = 1.0
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, minInt(active, len(phaseLabels)-1), phaseProgress))

	// 5) 64 轮进度环（PrimRing 默认无坐标，由前端 HUD 区域布局）
	prims = append(prims, fw.PrimRing("ring-rounds", totalRounds, st.DisplayRound, fmt.Sprintf("Round %d / %d", st.DisplayRound, totalRounds)))

	// 6) a..h 寄存器组
	regHex := make([]string, 8)
	for i, v := range st.Regs {
		regHex[i] = fmt.Sprintf("%08x", v)
	}
	highlight := -1
	if active == phaseIdxCompress {
		highlight = 0 // 压缩中：a 寄存器活跃
	}
	prims = append(prims, fw.PrimRegisterRow("reg-row",
		[]string{"a", "b", "c", "d", "e", "f", "g", "h"}, regHex, highlight))

	// 7) 压缩函数公式（LaTeX × 2）
	prims = append(prims, fw.PrimMathFormula("formula-t1",
		`T_1 = h + \Sigma_1(e) + \mathrm{Ch}(e,f,g) + K_t + W_t`, false))
	prims = append(prims, fw.PrimMathFormula("formula-t2",
		`T_2 = \Sigma_0(a) + \mathrm{Maj}(a,b,c)`, false))

	// 8) 6 个代码块（输入 / 填充 / 调度 / 当前 W·K / 摘要 / 雪崩对比）
	prims = append(prims, fw.PrimCodeBlock("cb-input",
		fmt.Sprintf("%s\n(%d 字节 / %d 比特)", st.Input, len([]byte(st.Input)), len([]byte(st.Input))*8),
		"text", nil, 6))
	prims = append(prims, fw.PrimCodeBlock("cb-padded",
		hexBlocks(padMessage([]byte(st.Input))), "text", nil, 8))
	prims = append(prims, fw.PrimCodeBlock("cb-schedule",
		formatScheduleHex(st.W), "text", nil, 8))
	wkLines := []int{}
	if st.CurrentRound > 0 && st.CurrentRound <= totalRounds {
		wkLines = []int{0}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-current-w-k",
		fmt.Sprintf("t = %d\nW[t] = %08x\nK[t] = %08x",
			maxInt(st.CurrentRound-1, 0),
			st.W[clampRound(st.CurrentRound)],
			sha256K[clampRound(st.CurrentRound)]),
		"text", wkLines, 4))
	prims = append(prims, fw.PrimCodeBlock("cb-digest", st.Digest, "text", nil, 4))
	prims = append(prims, fw.PrimCodeBlock("cb-avalanche",
		formatAvalanche(st.Digest, st.MutatedDigest, st.AvalancheBits, st.BitFlipped),
		"text", nil, 6))

	// 9) shift_animation：演示 a..h 轮转（每次推进 1 轮时被 MicroStep fire）
	prims = append(prims, fw.PrimShiftAnimation("anim-reg-shift", "reg-row", "right", 1.0, 600))

	// 10) pulse：当前块 W[t]/K[t] 同步呼吸
	prims = append(prims, fw.PrimPulse("pulse-wk", "cb-current-w-k", "info", 1200))

	// 11) 高亮当前阶段节点
	if active >= 0 && active < len(pipelineNodeIDs) {
		prims = append(prims, fw.PrimGlow("glow-active", pipelineNodeIDs[active], "info", 0.8))
	}

	// 12) 联动徽章（owner 场景，显示 idle，仅作 group 标识）
	prims = append(prims, fw.PrimLinkIndicator("link-crypto-verify", linkGroupCryptoVerify, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "SHA-256 错误", st.LastError, "scene", "请检查输入", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

// buildSidePanelData 12 字段（与 AGENTS.md §0.4 SidePanel key-value 约定一致）。
func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"input":          st.Input,
		"input_bits":     len([]byte(st.Input)) * 8,
		"current_round":  st.CurrentRound,
		"display_round":  st.DisplayRound,
		"total_rounds":   totalRounds,
		"current_W":      fmt.Sprintf("%08x", st.W[clampRound(st.CurrentRound)]),
		"current_K":      fmt.Sprintf("%08x", sha256K[clampRound(st.CurrentRound)]),
		"digest":         st.Digest,
		"mutated_input":  st.MutatedInput,
		"mutated_digest": st.MutatedDigest,
		"avalanche_bits": st.AvalancheBits,
		"padded_blocks":  st.PaddedBlocks,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroSteps 模板（教学节奏，详 06.md §3.10.4 + AGENTS.md §0.5）
// =====================================================================

// appendSetInputMicroSteps reactive 模式下，set_input 操作的 5 个教学子步。
func appendSetInputMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "set-1", Label: "重置状态", DurationMs: 400, HighlightIDs: []string{pipelineNodeIDs[phaseIdxInput], "cb-input"}, ParentPhase: "padding"},
		{ID: "set-2", Label: "比特填充至 512 倍数", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxPadding], "cb-padded"}, FirePrimitives: []string{"glow-active"}, ParentPhase: "padding"},
		{ID: "set-3", Label: "生成 64 个调度字 W[t]", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxSchedule], "cb-schedule"}, ParentPhase: "schedule"},
		{ID: "set-4", Label: "执行 64 轮压缩", DurationMs: 800, HighlightIDs: []string{pipelineNodeIDs[phaseIdxCompress], "reg-row", "ring-rounds"}, FirePrimitives: []string{"anim-reg-shift", "pulse-wk"}, ParentPhase: "compression"},
		{ID: "set-5", Label: "输出 256-bit 摘要", DurationMs: 600, HighlightIDs: []string{pipelineNodeIDs[phaseIdxOutput], "cb-digest"}, IsLinkTrigger: true, ParentPhase: "output"},
	}
}

// appendStepRoundMicroSteps step_round 操作的 3 个教学子步（一轮压缩的展开）。
func appendStepRoundMicroSteps(env *fw.RenderEnvelope, round int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: fmt.Sprintf("r%d-1", round), Label: fmt.Sprintf("Round %d 取 W[t] / K[t]", round-1), DurationMs: 400, HighlightIDs: []string{"cb-current-w-k", "ring-rounds"}, FirePrimitives: []string{"pulse-wk"}, ParentPhase: "compression"},
		{ID: fmt.Sprintf("r%d-2", round), Label: "应用压缩函数 T₁ / T₂", DurationMs: 500, HighlightIDs: []string{"formula-t1", "formula-t2", "reg-row"}, ParentPhase: "compression"},
		{ID: fmt.Sprintf("r%d-3", round), Label: "更新 a..h 寄存器（h→g→...→a）", DurationMs: 500, HighlightIDs: []string{"reg-row"}, FirePrimitives: []string{"anim-reg-shift"}, ParentPhase: "compression"},
	}
}

// appendMutateMicroSteps mutate_input_bit 操作的 3 个教学子步（雪崩演示）。
func appendMutateMicroSteps(env *fw.RenderEnvelope, bitIdx, avalanche int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "mut-1", Label: fmt.Sprintf("翻转输入第 %d 位", bitIdx), DurationMs: 500, HighlightIDs: []string{"cb-input"}, FirePrimitives: []string{"glow-active"}, ParentPhase: "avalanche"},
		{ID: "mut-2", Label: "重新执行 SHA-256（多块迭代）", DurationMs: 800, HighlightIDs: []string{pipelineNodeIDs[phaseIdxCompress], "ring-rounds"}, FirePrimitives: []string{"anim-reg-shift"}, ParentPhase: "avalanche"},
		{ID: "mut-3", Label: fmt.Sprintf("差异比特 %d / %d（雪崩效应）", avalanche, digestBits), DurationMs: 700, HighlightIDs: []string{"cb-avalanche", "cb-digest"}, IsLinkTrigger: true, ParentPhase: "avalanche"},
	}
}

// =====================================================================
// 联动（owner-based 子树发布；详 AGENTS.md §0.6 + 06.md §8.3）
// =====================================================================

// publishOwnerSubtree 在 RenderEnvelope.Data 中带上 link_state 摘要（仅供 SidePanel 调试展示），
// 真正的 SharedState 写入通过 ownerDiff() 返回到 SharedStateDiff。
func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	// §6.2 LinkTrigger：声明本场景对 hashes.sha256.* owner 子树的写入；
	// SourceAnchorID/TargetAnchorID 用于 M8 跨画布弧线起终点定位（§6.2 / §0.7.1 C18）。
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             fmt.Sprintf("sha256-publish-r%d", st.DisplayRound),
		SourceScene:    sceneCode,
		SourceAction:   "publish_digest",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"hashes.sha256.digest_hex", "hashes.sha256.avalanche_bits"},
		Payload:        map[string]any{"digest_hex": st.Digest, "avalanche_bits": st.AvalancheBits},
		SourceAnchorID: "sha256-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys,
		"hashes.sha256.digest_hex", "hashes.sha256.avalanche_bits")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

// ownerDiff 构造本场景在 crypto-verify-group 的 owner 子树（v0.5 owner-based 嵌套子 map）。
func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"hashes": map[string]any{
			"sha256": map[string]any{
				"input":              st.Input,
				"digest_hex":         st.Digest,
				"mutated_digest_hex": st.MutatedDigest,
				"avalanche_bits":     st.AvalancheBits,
				"current_round":      st.CurrentRound,
				"updated_at_tick":    0,
			},
		},
	}
}

// =====================================================================
// 工具函数（仅本文件用，纯字符串 / 数字 / hex 处理）
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// clampRound 钳制到 [0, 63]，避免 W[64] 越界。
func clampRound(t int) int {
	if t < 0 {
		return 0
	}
	if t >= totalRounds {
		return totalRounds - 1
	}
	return t
}

// hexBlocks 把字节切片按 16 字节一行格式化为 hex 字符串（最多 8 行避免过长）。
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
		if i/16 >= 7 && i+16 < len(b) {
			out += "\n…"
			break
		}
	}
	return out
}

// formatScheduleHex 把 W[0..63] 按 8 列展开（共 8 行）。
func formatScheduleHex(w [64]uint32) string {
	out := ""
	for i := 0; i < 64; i += 8 {
		if i > 0 {
			out += "\n"
		}
		for j := 0; j < 8; j++ {
			if j > 0 {
				out += " "
			}
			out += fmt.Sprintf("%08x", w[i+j])
		}
	}
	return out
}

// formatAvalanche 把雪崩对比结果格式化为 4 行：原摘要 / 翻转后摘要 / 差异比特数 / 翻转位号。
func formatAvalanche(orig, mutated string, bits, flippedBit int) string {
	if mutated == "" {
		return "（未触发雪崩演示，使用 mutate_input_bit）"
	}
	return fmt.Sprintf("原  : %s\n翻转: %s\n翻转位: %d\n差异: %d / %d 比特", orig, mutated, flippedBit, bits, digestBits)
}
