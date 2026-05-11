// 模块：sim-engine/scenarios/internal/consensus/powmining
// 文件职责：CON-01 工作量证明（PoW）挖矿场景的完整实现。
//
// SSOT 依据：06.md §4.2.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现 PoW 挖矿循环（比特币风格），零第三方加密库；复用
// scenarios/internal/cryptography/sha256hash.Sum256 做区块头双 SHA-256：
//
//   · 区块头序列化：prev_hash || merkle_root || timestamp || difficulty || nonce
//   · header_hash = SHA-256(SHA-256(header))（双 SHA-256，比特币习惯）
//   · 难度比较：将 hash 解读为大端 256-bit 整数，比对 target = 2^(256-difficulty)
//     —— 即"前导零比特数 ≥ difficulty" 等价于"hash 整数值 ≤ target"
//   · 挖矿：从 nonce=0 递增，找到满足条件的 nonce 即出块
//   · 难度调整：按当前 hash rate / 期望区块时间，每 N 块调整一次
//   · 51% 攻击：双轨链对照 — 诚实链与攻击链并行挖矿，攻击链超过诚实链时
//     展示"链重组"，演示双花成立的条件
//
// 教学决策（双轨对照 + 流水线）：
//   - PrimDualTrack 双轨链对比（honest vs attack）
//   - PrimRing 64 位前导零目标进度（当前 hash 接近 target 的程度）
//   - PrimTargetZone 难度阈值线
//   - PrimRiskGauge 算力占比仪表

package powmining

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "pow-mining"
	schemaVersion = "v1.0.0"
	algorithmType = "pow-bitcoin"

	defaultDifficulty = 16 // 前导零比特数（教学用：4 ≈ 16 次尝试，16 ≈ 65k 次）
	maxDifficulty     = 32
	minDifficulty     = 4
	defaultStepLimit  = 50000 // 单次 step_attempts 最大尝试数
	maxBlockHistory   = 8

	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkGroupPowAttack        = "pow-attack-group"
	linkOwnerSubtree          = "blocks.pow"
)

var pipelineNodeIDs = []string{
	"phase-template", "phase-nonce", "phase-hash", "phase-target", "phase-block",
}
var phaseLabels = []string{"区块头模板", "递增 Nonce", "双 SHA-256", "比对难度"}

// =====================================================================
// 区块结构
// =====================================================================

// blockHeader 比特币风格区块头（教学版精简：80 字节 ≈ 真实 80 字节）。
type blockHeader struct {
	PrevHash   [32]byte
	MerkleRoot [32]byte
	Timestamp  uint32
	Difficulty uint32 // 前导零比特数（教学：直接存比特数，不用 nBits 编码）
	Nonce      uint64
}

// serialize 把区块头按固定布局序列化为字节流。
func (h blockHeader) serialize() []byte {
	buf := make([]byte, 0, 80)
	buf = append(buf, h.PrevHash[:]...)
	buf = append(buf, h.MerkleRoot[:]...)
	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], h.Timestamp)
	buf = append(buf, ts[:]...)
	var diff [4]byte
	binary.BigEndian.PutUint32(diff[:], h.Difficulty)
	buf = append(buf, diff[:]...)
	var nonce [8]byte
	binary.BigEndian.PutUint64(nonce[:], h.Nonce)
	buf = append(buf, nonce[:]...)
	return buf
}

// hash 计算 double-SHA256(header)。
func (h blockHeader) hash() [32]byte {
	first := sha256hash.Sum256(h.serialize())
	return sha256hash.Sum256(first[:])
}

// minedBlock 完整挖到的区块。
type minedBlock struct {
	Height int
	Header blockHeader
	Hash   [32]byte
}

// targetFromDifficulty 计算 difficulty 对应的 256-bit target（hash ≤ target 即满足）。
// target = 2^(256 - difficulty) - 1（更精确一点：2^(256-difficulty)，但用 -1 避免溢出比较）
func targetFromDifficulty(difficulty uint32) *big.Int {
	if difficulty == 0 {
		// difficulty=0 没有任何要求，target = 2^256 - 1
		return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	}
	return new(big.Int).Lsh(big.NewInt(1), 256-uint(difficulty))
}

// hashAsBigInt 把 32 字节 hash 视作大端无符号 256-bit 整数。
func hashAsBigInt(h [32]byte) *big.Int {
	return new(big.Int).SetBytes(h[:])
}

// leadingZeroBits 计算 hash 的前导零比特数（用于教学进度展示）。
func leadingZeroBits(h [32]byte) int {
	for i, b := range h {
		if b == 0 {
			continue
		}
		// 当前字节非零，统计该字节内前导零
		zeros := 0
		for mask := byte(0x80); mask != 0; mask >>= 1 {
			if b&mask != 0 {
				break
			}
			zeros++
		}
		return i*8 + zeros
	}
	return 256
}

// mineN 从给定 startNonce 开始尝试 maxAttempts 次，返回 (找到的 nonce, hash, 实际尝试次数, 是否找到)。
func mineN(template blockHeader, startNonce uint64, maxAttempts int, difficulty uint32) (uint64, [32]byte, int, bool) {
	target := targetFromDifficulty(difficulty)
	h := template
	for i := 0; i < maxAttempts; i++ {
		h.Nonce = startNonce + uint64(i)
		hash := h.hash()
		if hashAsBigInt(hash).Cmp(target) <= 0 {
			return h.Nonce, hash, i + 1, true
		}
	}
	return startNonce + uint64(maxAttempts) - 1, h.hash(), maxAttempts, false
}

// =====================================================================
// 场景内部状态
// =====================================================================

type chainState struct {
	Blocks    []minedBlock
	NextNonce uint64
	Attempts  int
	LastHash  [32]byte
	BestZeros int // 当前模板尝试中遇到的最佳前导零数（演示"逼近"）
}

type snapState struct {
	Difficulty       int
	StepLimit        int
	TxData           string
	Honest           chainState
	Attack           chainState
	AttackerHashRate int // 0..100，攻击者算力占比百分比
	AttackEnabled    bool
	Reorged          bool
	LastError        string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		Difficulty:       fw.MapInt(d, "difficulty", defaultDifficulty),
		StepLimit:        fw.MapInt(d, "step_limit", defaultStepLimit),
		TxData:           fw.MapStr(d, "tx_data", "tx-default"),
		AttackerHashRate: fw.MapInt(d, "attacker_hashrate", 30),
		AttackEnabled:    fw.MapBool(d, "attack_enabled", false),
		Reorged:          fw.MapBool(d, "reorged", false),
		LastError:        fw.MapStr(d, "last_error", ""),
	}
	loadChain(&st.Honest, d, "honest_")
	loadChain(&st.Attack, d, "attack_")
	return st
}

func loadChain(c *chainState, d map[string]any, prefix string) {
	c.NextNonce = uint64(fw.MapInt(d, prefix+"next_nonce", 0))
	c.Attempts = fw.MapInt(d, prefix+"attempts", 0)
	c.BestZeros = fw.MapInt(d, prefix+"best_zeros", 0)
	if blocksAny, ok := d[prefix+"blocks"].([]any); ok {
		for _, b := range blocksAny {
			if m, ok := b.(map[string]any); ok {
				blk := minedBlock{
					Height: fw.MapInt(m, "height", 0),
					Header: blockHeader{
						Timestamp:  uint32(fw.MapInt(m, "ts", 0)),
						Difficulty: uint32(fw.MapInt(m, "difficulty", 0)),
						Nonce:      uint64(fw.MapInt(m, "nonce", 0)),
					},
				}
				if s := fw.MapStr(m, "prev_hash", ""); s != "" {
					if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
						copy(blk.Header.PrevHash[:], b)
					}
				}
				if s := fw.MapStr(m, "hash", ""); s != "" {
					if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
						copy(blk.Hash[:], b)
					}
				}
				c.Blocks = append(c.Blocks, blk)
			}
		}
	}
	if s := fw.MapStr(d, prefix+"last_hash", ""); s != "" {
		if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
			copy(c.LastHash[:], b)
		}
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["difficulty"] = st.Difficulty
	s.Data["step_limit"] = st.StepLimit
	s.Data["tx_data"] = st.TxData
	s.Data["attacker_hashrate"] = st.AttackerHashRate
	s.Data["attack_enabled"] = st.AttackEnabled
	s.Data["reorged"] = st.Reorged
	s.Data["last_error"] = st.LastError
	saveChain(s.Data, "honest_", st.Honest)
	saveChain(s.Data, "attack_", st.Attack)
}

func saveChain(d map[string]any, prefix string, c chainState) {
	d[prefix+"next_nonce"] = c.NextNonce
	d[prefix+"attempts"] = c.Attempts
	d[prefix+"best_zeros"] = c.BestZeros
	d[prefix+"last_hash"] = hex.EncodeToString(c.LastHash[:])
	blocksAny := make([]any, len(c.Blocks))
	for i, blk := range c.Blocks {
		blocksAny[i] = map[string]any{
			"height":     blk.Height,
			"prev_hash":  hex.EncodeToString(blk.Header.PrevHash[:]),
			"ts":         blk.Header.Timestamp,
			"difficulty": blk.Header.Difficulty,
			"nonce":      blk.Header.Nonce,
			"hash":       hex.EncodeToString(blk.Hash[:]),
		}
	}
	d[prefix+"blocks"] = blocksAny
}

func defaultSnapState() snapState {
	return snapState{
		Difficulty:       defaultDifficulty,
		StepLimit:        defaultStepLimit,
		TxData:           "tx-default",
		AttackerHashRate: 30,
	}
}

// currentTemplate 根据链尾构造下一个区块模板（merkle 用 tx_data 单交易简化）。
func (st snapState) currentTemplate(c chainState, height int) blockHeader {
	var prev [32]byte
	if len(c.Blocks) > 0 {
		prev = c.Blocks[len(c.Blocks)-1].Hash
	}
	merkle := sha256hash.Sum256([]byte(st.TxData))
	return blockHeader{
		PrevHash:   prev,
		MerkleRoot: merkle,
		Timestamp:  uint32(1700000000 + height),
		Difficulty: uint32(st.Difficulty),
		Nonce:      0,
	}
}

// stepMineChain 推进单条链 n 次 nonce 尝试；找到则追加新块并重置 NextNonce/Attempts/BestZeros。
func stepMineChain(st *snapState, c *chainState, n int) (found bool, newBlock *minedBlock, hash [32]byte) {
	tmpl := st.currentTemplate(*c, len(c.Blocks))
	tmpl.Nonce = c.NextNonce
	nonce, hash, attempts, found := mineN(tmpl, c.NextNonce, n, uint32(st.Difficulty))
	c.Attempts += attempts
	c.LastHash = hash
	zeros := leadingZeroBits(hash)
	if zeros > c.BestZeros {
		c.BestZeros = zeros
	}
	if found {
		blk := minedBlock{
			Height: len(c.Blocks),
			Header: tmpl,
			Hash:   hash,
		}
		blk.Header.Nonce = nonce
		c.Blocks = append(c.Blocks, blk)
		if len(c.Blocks) > maxBlockHistory {
			c.Blocks = c.Blocks[len(c.Blocks)-maxBlockHistory:]
		}
		c.NextNonce = 0
		c.Attempts = 0
		c.BestZeros = 0
		return true, &blk, hash
	}
	c.NextNonce = nonce + 1
	return false, nil, hash
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "工作量证明（PoW 挖矿）",
		Description:         "演示 PoW 区块头双 SHA-256 + 难度比对 + 51% 攻击 + 链重组",
		Category:            fw.CategoryConsensus,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupBlockchainIntegr, linkGroupPowAttack},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"blocks.pow.tip_hash",
			"blocks.pow.height",
			"blocks.pow.attack_height",
		},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultStateFw,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultStateFw() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data: map[string]any{
			"difficulty":        defaultDifficulty,
			"step_limit":        defaultStepLimit,
			"tx_data":           "tx-default",
			"attacker_hashrate": 30,
		},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_params", Label: "设置挖矿参数",
				Description: "设置难度（前导零比特数）+ 单次尝试上限 + 交易载荷",
				Category:    fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "difficulty", Type: fw.FieldNumber, Label: "难度（前导零比特数）", Required: true, Default: defaultDifficulty, Min: minDifficulty, Max: maxDifficulty, Step: 1},
					{Name: "step_limit", Type: fw.FieldNumber, Label: "单次 step 最大尝试", Required: true, Default: defaultStepLimit, Min: 1000, Max: 5000000, Step: 1000},
					{Name: "tx_data", Type: fw.FieldString, Label: "交易载荷（用于 merkle）", Required: false, Default: "tx-default"},
				},
			},
			{
				ActionCode: "step_mining", Label: "推进挖矿",
				Description: "执行 step_limit 次 nonce 尝试；找到目标即出块",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"blocks.pow.tip_hash", "blocks.pow.height"},
				LinkOwnerFields:   []string{"blocks.pow.tip_hash", "blocks.pow.height"},
			},
			{
				ActionCode: "mine_until_block", Label: "挖到下一块",
				Description: "持续尝试直到找到下一个有效区块（最多 1000 万次）",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"blocks.pow.tip_hash", "blocks.pow.height"},
				LinkOwnerFields:   []string{"blocks.pow.tip_hash", "blocks.pow.height"},
			},
			{
				ActionCode: "enable_51_attack", Label: "启用 51% 攻击",
				Description: "启动攻击链并行挖矿；攻击者算力占比可调",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "hashrate_pct", Type: fw.FieldNumber, Label: "攻击者算力占比 %", Required: true, Default: 30, Min: 1, Max: 99, Step: 1},
				},
				WritesOwnedFields: []string{"blocks.pow.attack_height"},
				LinkOwnerFields:   []string{"blocks.pow.attack_height"},
			},
			{
				ActionCode: "step_attack_round", Label: "推进 1 攻防回合",
				Description: "按算力比例推进诚实链与攻击链，演示链重组",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			// §0.7.4 混合实验：submit_real_block 走 geth eth_submitWork。
			{
				ActionCode: "submit_real_block", Label: "提交真链区块（容器通道）",
				Description:   "调 geth eth_submitWork 提交真实挖矿结果",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_submitWork","params":["{{nonce}}","{{header}}","{{mix}}"],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "nonce", Type: fw.FieldString, Label: "nonce (hex)", Required: true, Default: "0x0000000000000000"},
					{Name: "header", Type: fw.FieldString, Label: "header hash (hex)", Required: true, Default: ""},
					{Name: "mix", Type: fw.FieldString, Label: "mix digest (hex)", Required: true, Default: ""},
				},
			},
			{
				ActionCode:    "teacher_inject_fault",
				Label:         "教师注入故障",
				Description:   "仅教师可用，注入故障用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneFault,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "故障描述", Required: false, Default: "教师注入故障"},
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
	state.Seed = in.Seed
	st := loadState(state)
	saveState(state, st)
	state.Phase = "mining"
	env := buildEnvelope(st, "init", "PoW 初始化（创世空链）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", "", false)
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
	case "set_params":
		st.Difficulty = fw.MapInt(in.Params, "difficulty", defaultDifficulty)
		st.StepLimit = fw.MapInt(in.Params, "step_limit", defaultStepLimit)
		st.TxData = fw.MapStr(in.Params, "tx_data", "tx-default")
		// 改难度 / tx 后重置当前模板进度（已挖到的块不动）
		st.Honest.NextNonce = 0
		st.Honest.Attempts = 0
		st.Honest.BestZeros = 0
		st.Attack.NextNonce = 0
		st.Attack.Attempts = 0
		st.Attack.BestZeros = 0
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", fmt.Sprintf("难度 = %d, step = %d", st.Difficulty, st.StepLimit), false)
		return out, nil

	case "step_mining":
		found, blk, _ := stepMineChain(&st, &st.Honest, st.StepLimit)
		saveState(state, st)
		summary := fmt.Sprintf("已尝试 %d 次（最佳前导零=%d）", st.Honest.Attempts, st.Honest.BestZeros)
		if found {
			summary = fmt.Sprintf("挖到第 %d 块，nonce = %d", blk.Height, blk.Header.Nonce)
		}
		out.Render = buildEnvelope(st, "step_mining", summary, false)
		appendStepMicroSteps(&out.Render, found)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_until_block":
		// 限定最多 1000 万次尝试以避免无限循环
		batch := 100000
		var blk *minedBlock
		found := false
		total := 0
		for total < 10000000 {
			ok, b, _ := stepMineChain(&st, &st.Honest, batch)
			total += batch
			if ok {
				blk = b
				found = true
				break
			}
		}
		saveState(state, st)
		summary := fmt.Sprintf("尝试 %d 次未找到", total)
		if found && blk != nil {
			summary = fmt.Sprintf("挖到第 %d 块（共尝试 %d 次），nonce = %d", blk.Height, total, blk.Header.Nonce)
		}
		out.Render = buildEnvelope(st, "mine_until_block", summary, false)
		appendStepMicroSteps(&out.Render, found)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "enable_51_attack":
		st.AttackEnabled = true
		st.AttackerHashRate = fw.MapInt(in.Params, "hashrate_pct", 30)
		// 攻击链从诚实链当前末端分叉（教学：直接复制 blocks）
		st.Attack.Blocks = append([]minedBlock{}, st.Honest.Blocks...)
		st.Attack.NextNonce = 0
		st.Attack.Attempts = 0
		st.Attack.BestZeros = 0
		st.Reorged = false
		saveState(state, st)
		out.Render = buildEnvelope(st, "enable_51_attack",
			fmt.Sprintf("启用 51%% 攻击（攻击算力 %d%%）", st.AttackerHashRate), false)
		appendAttackInitMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_attack_round":
		if !st.AttackEnabled {
			return fw.ActionOutput{Success: false, ErrorMessage: "请先 enable_51_attack"}, nil
		}
		// 按算力比例分配尝试预算
		honestBudget := st.StepLimit * (100 - st.AttackerHashRate) / 100
		attackBudget := st.StepLimit * st.AttackerHashRate / 100
		if honestBudget < 100 {
			honestBudget = 100
		}
		if attackBudget < 100 {
			attackBudget = 100
		}
		stepMineChain(&st, &st.Honest, honestBudget)
		stepMineChain(&st, &st.Attack, attackBudget)
		// 检测重组
		if len(st.Attack.Blocks) > len(st.Honest.Blocks) {
			st.Reorged = true
		}
		saveState(state, st)
		summary := fmt.Sprintf("诚实=%d 块, 攻击=%d 块", len(st.Honest.Blocks), len(st.Attack.Blocks))
		if st.Reorged {
			summary += "  ⚠ 攻击链反超 → 链重组成功"
		}
		out.Render = buildEnvelope(st, "step_attack_round", summary, false)
		appendAttackRoundMicroSteps(&out.Render, st.Reorged)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "teacher_inject_fault":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师注入故障"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-fault-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		st = defaultSnapState()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置（清空所有链）", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 30)

	// 1) 流水线 5 阶段
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))
	for i, id := range pipelineNodeIDs {
		role := []string{"template", "nonce", "hash", "target", "block"}[i]
		label := []string{"区块头", "Nonce++", "双 SHA-256", "≤ Target?", "新区块"}[i]
		status := "active"
		if i < 4 {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-%d-%d", i, i+1),
			pipelineNodeIDs[i], pipelineNodeIDs[i+1], "solid", "flow"))
	}

	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, 0, float64(st.Honest.BestZeros)/float64(st.Difficulty)))

	// 2) 当前 hash 前导零进度环（最大值 = difficulty，≥ 即出块）
	prims = append(prims, fw.PrimRing("ring-zeros", st.Difficulty, st.Honest.BestZeros,
		fmt.Sprintf("最佳 %d / %d 前导零", st.Honest.BestZeros, st.Difficulty)))

	// 3) 难度阈值线（target_zone）
	prims = append(prims, fw.PrimTargetZone("target-line", float64(st.Difficulty),
		fmt.Sprintf("难度阈值 (≥ %d 前导零)", st.Difficulty), "y"))

	// 4) 双轨链对比（DualTrack 领域复合原语）
	tracks := []map[string]any{
		{
			"lane":   "honest",
			"label":  "诚实链",
			"blocks": chainBlocksForTrack(st.Honest, false),
		},
	}
	if st.AttackEnabled {
		tracks = append(tracks, map[string]any{
			"lane":   "attack",
			"label":  "攻击链 (51%)",
			"blocks": chainBlocksForTrack(st.Attack, true),
		})
	}
	prims = append(prims, fw.PrimDualTrack("dual-track", tracks))

	// 5) 算力占比仪表（risk_gauge）
	if st.AttackEnabled {
		prims = append(prims, fw.PrimRiskGauge("hashrate-gauge",
			float64(st.AttackerHashRate),
			[]map[string]any{
				{"from": 0.0, "to": 33.0, "color": "success"},
				{"from": 33.0, "to": 50.0, "color": "warning"},
				{"from": 50.0, "to": 100.0, "color": "danger"},
			},
		))
	}

	// 6) 公式
	prims = append(prims, fw.PrimMathFormula("formula-pow",
		`H = \mathrm{SHA256}(\mathrm{SHA256}(\mathrm{header})),\quad H \le 2^{256-\mathrm{difficulty}}`, false))

	// 7) 区块头详情
	tmpl := st.currentTemplate(st.Honest, len(st.Honest.Blocks))
	headerLines := []string{
		fmt.Sprintf("prev_hash    = %s", hex.EncodeToString(tmpl.PrevHash[:])),
		fmt.Sprintf("merkle_root  = %s", hex.EncodeToString(tmpl.MerkleRoot[:])),
		fmt.Sprintf("timestamp    = %d", tmpl.Timestamp),
		fmt.Sprintf("difficulty   = %d", tmpl.Difficulty),
		fmt.Sprintf("nonce(尝试)  = %d", st.Honest.NextNonce),
	}
	prims = append(prims, fw.PrimCodeBlock("cb-header", strings.Join(headerLines, "\n"), "text", nil, 6))

	// 8) 当前 hash
	prims = append(prims, fw.PrimCodeBlock("cb-hash",
		fmt.Sprintf("当前 hash = %s\n前导零比特 = %d / %d",
			hex.EncodeToString(st.Honest.LastHash[:]), leadingZeroBits(st.Honest.LastHash), st.Difficulty),
		"text", nil, 4))

	// 9) 链历史（已出块）
	chainLines := []string{fmt.Sprintf("诚实链高度 = %d", len(st.Honest.Blocks))}
	for _, b := range st.Honest.Blocks {
		chainLines = append(chainLines, fmt.Sprintf("  #%d nonce=%d  hash=%s...",
			b.Height, b.Header.Nonce, hex.EncodeToString(b.Hash[:])[:16]))
	}
	if st.AttackEnabled {
		chainLines = append(chainLines, "", fmt.Sprintf("攻击链高度 = %d", len(st.Attack.Blocks)))
		for _, b := range st.Attack.Blocks {
			chainLines = append(chainLines, fmt.Sprintf("  #%d nonce=%d  hash=%s...",
				b.Height, b.Header.Nonce, hex.EncodeToString(b.Hash[:])[:16]))
		}
		if st.Reorged {
			chainLines = append(chainLines, "", "⚠ 攻击链已反超 → 双花成立")
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-chain", strings.Join(chainLines, "\n"), "text", nil, 16))

	// 10) 动效
	prims = append(prims, fw.PrimGlow("glow-mining", pipelineNodeIDs[2], "info", 0.8))
	prims = append(prims, fw.PrimPulse("pulse-target", "target-line", "warning", 1500))
	if len(st.Honest.Blocks) > 0 {
		prims = append(prims, fw.PrimBurst("burst-block", pipelineNodeIDs[4], "success",
			int64(len(st.Honest.Blocks)), 800))
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-attack", linkGroupPowAttack, "idle", ""))

	// 12) 错误
	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "PoW 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "pow.block_height", Value: len(st.Honest.Blocks), TargetPrimitive: "cb-chain", TargetParam: "height"},
		},
	}
}

func chainBlocksForTrack(c chainState, attack bool) []map[string]any {
	out := make([]map[string]any, 0, len(c.Blocks))
	for _, b := range c.Blocks {
		role := "honest-block"
		if attack {
			role = "attack-block"
		}
		out = append(out, map[string]any{
			"id":    fmt.Sprintf("blk-%s-%d", role, b.Height),
			"label": fmt.Sprintf("#%d", b.Height),
			"hash":  hex.EncodeToString(b.Hash[:])[:12],
			"nonce": b.Header.Nonce,
		})
	}
	return out
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"difficulty":        st.Difficulty,
		"step_limit":        st.StepLimit,
		"tx_data":           st.TxData,
		"honest_height":     len(st.Honest.Blocks),
		"honest_attempts":   st.Honest.Attempts,
		"honest_best_zeros": st.Honest.BestZeros,
		"honest_tip":        hex.EncodeToString(st.Honest.LastHash[:]),
		"attack_enabled":    st.AttackEnabled,
		"attacker_hashrate": st.AttackerHashRate,
		"attack_height":     len(st.Attack.Blocks),
		"reorged":           st.Reorged,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendStepMicroSteps(env *fw.RenderEnvelope, found bool) {
	tail := "未达到难度，继续尝试"
	if found {
		tail = "✓ hash ≤ target → 出块"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "mn-1", Label: "构造区块头模板", DurationMs: 300, HighlightIDs: []string{"cb-header", pipelineNodeIDs[0]}, ParentPhase: "mining"},
		{ID: "mn-2", Label: "递增 nonce", DurationMs: 300, HighlightIDs: []string{pipelineNodeIDs[1]}, FirePrimitives: []string{"glow-mining"}},
		{ID: "mn-3", Label: "double-SHA256(header)", DurationMs: 400, HighlightIDs: []string{"formula-pow", "cb-hash", pipelineNodeIDs[2]}},
		{ID: "mn-4", Label: "对比难度阈值", DurationMs: 300, HighlightIDs: []string{"target-line", "ring-zeros", pipelineNodeIDs[3]}, FirePrimitives: []string{"pulse-target"}},
		{ID: "mn-5", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-chain", pipelineNodeIDs[4]}, FirePrimitives: []string{"burst-block"}, IsLinkTrigger: true},
	}
}

func appendAttackInitMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "atk-1", Label: "攻击者从诚实链分叉", DurationMs: 500, HighlightIDs: []string{"dual-track"}, FirePrimitives: []string{"glow-mining"}},
		{ID: "atk-2", Label: "并行启动攻击链挖矿", DurationMs: 500, HighlightIDs: []string{"hashrate-gauge"}},
		{ID: "atk-3", Label: "等待算力倾斜 → 链重组", DurationMs: 600, HighlightIDs: []string{"dual-track"}, IsLinkTrigger: true},
	}
}

func appendAttackRoundMicroSteps(env *fw.RenderEnvelope, reorged bool) {
	tail := "诚实链仍领先"
	if reorged {
		tail = "⚠ 攻击链反超 → 双花成立"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "ar-1", Label: "诚实矿工尝试 N 次", DurationMs: 400, HighlightIDs: []string{"dual-track"}},
		{ID: "ar-2", Label: "攻击者按比例尝试 N 次", DurationMs: 400, HighlightIDs: []string{"hashrate-gauge"}, FirePrimitives: []string{"glow-mining"}},
		{ID: "ar-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-chain"}, FirePrimitives: []string{"burst-block"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "pow-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_block",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"blocks.pow.tip_hash", "blocks.pow.height"},
		Payload:        map[string]any{"height": len(st.Honest.Blocks)},
		SourceAnchorID: "pow-tip-anchor",
		TargetAnchorID: "chain-tip-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "blocks.pow.tip_hash", "blocks.pow.height")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	tip := ""
	if len(st.Honest.Blocks) > 0 {
		tip = hex.EncodeToString(st.Honest.Blocks[len(st.Honest.Blocks)-1].Hash[:])
	}
	return map[string]any{
		"blocks": map[string]any{
			"pow": map[string]any{
				"height":            len(st.Honest.Blocks),
				"tip_hash":          tip,
				"difficulty":        st.Difficulty,
				"attack_height":     len(st.Attack.Blocks),
				"attack_enabled":    st.AttackEnabled,
				"attacker_hashrate": st.AttackerHashRate,
				"reorged":           st.Reorged,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

