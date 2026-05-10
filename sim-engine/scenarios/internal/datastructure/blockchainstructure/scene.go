// 模块：sim-engine/scenarios/internal/datastructure/blockchainstructure
// 文件职责：DS-01 区块链结构（区块 + 链头链 + 分叉重组）场景的完整实现。
//
// SSOT 依据：06.md §4.4.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现区块链数据结构（零外部依赖）：
//   · Block：height + prev_hash + tx_root + ts + difficulty + nonce + miner → hash = SHA-256(serialized)
//   · 主链：按 hash 索引的有向链，每个 block 的 prev_hash 必须 == 前一块 hash；
//   · 分叉：可在任意 height 创建分叉块，与主链共存；
//   · 重组：当某分叉 total_diff > 主链 total_diff 时，切换主链；
//   · 完整性验证：从 tip 回溯到创世，每对 (block, prev) 验证 prev_hash 一致。

package blockchainstructure

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

const (
	sceneCode     = "blockchain-structure"
	schemaVersion = "v1.0.0"
	algorithmType = "blockchain"

	defaultDifficulty = 100
	maxBlocks         = 64

	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkOwnerSubtree          = "chain.struct"
)

type block struct {
	Height     int
	PrevHash   string // hex
	TxRoot     string // hex（简化：直接用 tx_count 派生）
	Timestamp  uint32
	Difficulty int
	Nonce      uint64
	Miner      string
	Hash       string // hex
}

// computeBlockHash 用 SHA-256 计算区块头哈希。
func computeBlockHash(b block) string {
	buf := make([]byte, 0, 128)
	prevBytes, _ := hex.DecodeString(b.PrevHash)
	if len(prevBytes) < 32 {
		prevBytes = append(prevBytes, make([]byte, 32-len(prevBytes))...)
	}
	buf = append(buf, prevBytes[:32]...)
	trBytes, _ := hex.DecodeString(b.TxRoot)
	if len(trBytes) < 32 {
		trBytes = append(trBytes, make([]byte, 32-len(trBytes))...)
	}
	buf = append(buf, trBytes[:32]...)
	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], b.Timestamp)
	buf = append(buf, ts[:]...)
	var diff [4]byte
	binary.BigEndian.PutUint32(diff[:], uint32(b.Difficulty))
	buf = append(buf, diff[:]...)
	var nonce [8]byte
	binary.BigEndian.PutUint64(nonce[:], b.Nonce)
	buf = append(buf, nonce[:]...)
	buf = append(buf, []byte(b.Miner)...)
	var height [4]byte
	binary.BigEndian.PutUint32(height[:], uint32(b.Height))
	buf = append(buf, height[:]...)
	h := sha256hash.Sum256(buf)
	return hex.EncodeToString(h[:])
}

// computeTxRoot 教学版：直接 SHA-256(tx_count || miner || nonce)。
func computeTxRoot(txCount int, miner string, nonce uint64) string {
	buf := []byte(miner)
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], uint32(txCount))
	buf = append(buf, c[:]...)
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], nonce)
	buf = append(buf, n[:]...)
	h := sha256hash.Sum256(buf)
	return hex.EncodeToString(h[:])
}

type chain struct {
	BlocksByHash   map[string]block
	BlocksByHeight map[int][]string // height → hash 列表（含分叉）
	MainTip        string           // 当前主链 tip hash
	GenesisHash    string
}

func newChain() chain {
	c := chain{
		BlocksByHash:   map[string]block{},
		BlocksByHeight: map[int][]string{},
	}
	// 创世块
	g := block{
		Height: 0, PrevHash: strings.Repeat("0", 64),
		TxRoot:    computeTxRoot(0, "genesis", 0),
		Timestamp: 1700000000, Difficulty: defaultDifficulty, Nonce: 0, Miner: "genesis",
	}
	g.Hash = computeBlockHash(g)
	c.BlocksByHash[g.Hash] = g
	c.BlocksByHeight[0] = []string{g.Hash}
	c.MainTip = g.Hash
	c.GenesisHash = g.Hash
	return c
}

// addBlockOnTop 在 prevHash 之上挖新块；如果 prevHash == MainTip 则延续主链；否则形成分叉。
func (c *chain) addBlockOnTop(prevHash string, miner string, txCount int, ts uint32, nonce uint64) (block, error) {
	prev, ok := c.BlocksByHash[prevHash]
	if !ok {
		return block{}, fmt.Errorf("找不到 prev hash: %s", prevHash)
	}
	b := block{
		Height: prev.Height + 1, PrevHash: prevHash,
		TxRoot:    computeTxRoot(txCount, miner, nonce),
		Timestamp: ts, Difficulty: defaultDifficulty, Nonce: nonce, Miner: miner,
	}
	b.Hash = computeBlockHash(b)
	if _, exists := c.BlocksByHash[b.Hash]; exists {
		return block{}, errors.New("hash 碰撞（请换 nonce）")
	}
	if len(c.BlocksByHash) >= maxBlocks {
		return block{}, fmt.Errorf("区块数 ≥ %d", maxBlocks)
	}
	c.BlocksByHash[b.Hash] = b
	c.BlocksByHeight[b.Height] = append(c.BlocksByHeight[b.Height], b.Hash)
	return b, nil
}

// totalDiff 从 tip 回溯到 genesis 累加难度。
func (c chain) totalDiff(tipHash string) int {
	td := 0
	cur := tipHash
	for {
		b, ok := c.BlocksByHash[cur]
		if !ok {
			return td
		}
		td += b.Difficulty
		if b.Height == 0 {
			return td
		}
		cur = b.PrevHash
	}
}

// chainFromTip 返回从 genesis 到 tip 的有序链。
func (c chain) chainFromTip(tipHash string) []block {
	out := []block{}
	cur := tipHash
	for {
		b, ok := c.BlocksByHash[cur]
		if !ok {
			break
		}
		out = append([]block{b}, out...)
		if b.Height == 0 {
			break
		}
		cur = b.PrevHash
	}
	return out
}

// allTips 找出所有"非任何块的 prev"的块（即叶子）。
func (c chain) allTips() []string {
	parents := map[string]bool{}
	for _, b := range c.BlocksByHash {
		if b.Height > 0 {
			parents[b.PrevHash] = true
		}
	}
	out := []string{}
	for h := range c.BlocksByHash {
		if !parents[h] {
			out = append(out, h)
		}
	}
	sort.Strings(out)
	return out
}

// pickMainTip 选 totalDiff 最大的 tip 作为主链 tip。
func (c *chain) pickMainTip() {
	tips := c.allTips()
	if len(tips) == 0 {
		return
	}
	best := tips[0]
	bestTD := c.totalDiff(best)
	for _, t := range tips[1:] {
		td := c.totalDiff(t)
		if td > bestTD {
			best = t
			bestTD = td
		}
	}
	c.MainTip = best
}

// verifyChain 从 tip 回溯，每对 (block, prev) 验证 prev_hash 与 hash 一致。
func (c chain) verifyChain(tipHash string) (bool, []string) {
	violations := []string{}
	cur := tipHash
	for {
		b, ok := c.BlocksByHash[cur]
		if !ok {
			violations = append(violations, "block missing: "+cur)
			return false, violations
		}
		// 重新计算 hash 验证未被篡改
		if computeBlockHash(b) != b.Hash {
			violations = append(violations, fmt.Sprintf("h=%d hash mismatch", b.Height))
		}
		if b.Height == 0 {
			break
		}
		prev, ok := c.BlocksByHash[b.PrevHash]
		if !ok {
			violations = append(violations, fmt.Sprintf("h=%d prev missing", b.Height))
			return false, violations
		}
		if prev.Height != b.Height-1 {
			violations = append(violations, fmt.Sprintf("h=%d prev height mismatch", b.Height))
		}
		cur = b.PrevHash
	}
	return len(violations) == 0, violations
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Chain      chain
	NextNonce  uint64
	Tampered   bool
	TamperedAt int
	LastError  string
}

func defaultSnapState() snapState {
	return snapState{Chain: newChain()}
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		Chain: chain{
			BlocksByHash:   map[string]block{},
			BlocksByHeight: map[int][]string{},
			MainTip:        fw.MapStr(d, "main_tip", ""),
			GenesisHash:    fw.MapStr(d, "genesis_hash", ""),
		},
		NextNonce:  uint64(fw.MapInt(d, "next_nonce", 0)),
		Tampered:   fw.MapBool(d, "tampered", false),
		TamperedAt: fw.MapInt(d, "tampered_at", -1),
		LastError:  fw.MapStr(d, "last_error", ""),
	}
	if bsAny, ok := d["blocks"].([]any); ok {
		for _, bAny := range bsAny {
			if bm, ok := bAny.(map[string]any); ok {
				b := block{
					Height:     fw.MapInt(bm, "height", 0),
					PrevHash:   fw.MapStr(bm, "prev", ""),
					TxRoot:     fw.MapStr(bm, "tx_root", ""),
					Timestamp:  uint32(fw.MapInt(bm, "ts", 0)),
					Difficulty: fw.MapInt(bm, "diff", defaultDifficulty),
					Nonce:      uint64(fw.MapInt(bm, "nonce", 0)),
					Miner:      fw.MapStr(bm, "miner", ""),
					Hash:       fw.MapStr(bm, "hash", ""),
				}
				st.Chain.BlocksByHash[b.Hash] = b
				st.Chain.BlocksByHeight[b.Height] = append(st.Chain.BlocksByHeight[b.Height], b.Hash)
			}
		}
	}
	if len(st.Chain.BlocksByHash) == 0 {
		return defaultSnapState()
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["main_tip"] = st.Chain.MainTip
	s.Data["genesis_hash"] = st.Chain.GenesisHash
	s.Data["next_nonce"] = st.NextNonce
	s.Data["tampered"] = st.Tampered
	s.Data["tampered_at"] = st.TamperedAt
	s.Data["last_error"] = st.LastError
	bs := make([]any, 0, len(st.Chain.BlocksByHash))
	for _, b := range st.Chain.BlocksByHash {
		bs = append(bs, map[string]any{
			"height": b.Height, "prev": b.PrevHash, "tx_root": b.TxRoot,
			"ts": int(b.Timestamp), "diff": b.Difficulty, "nonce": int(b.Nonce),
			"miner": b.Miner, "hash": b.Hash,
		})
	}
	s.Data["blocks"] = bs
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "区块链结构",
		Description:         "演示区块链 prev_hash 链 + 分叉 + 累计难度选主链 + 完整性验证 + 篡改检测",
		Category:            fw.CategoryDataStructure,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupBlockchainIntegr},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"chain.struct.tip_hash",
			"chain.struct.height",
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
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: "ready", Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "mine_block", Label: "挖新块（在主链尾）",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "miner", Type: fw.FieldString, Label: "矿工", Required: true, Default: "alice"},
					{Name: "tx_count", Type: fw.FieldNumber, Label: "交易数", Required: true, Default: 3, Min: 0, Max: 100, Step: 1},
				},
				WritesOwnedFields: []string{"chain.struct.tip_hash", "chain.struct.height"},
				LinkOwnerFields:   []string{"chain.struct.tip_hash", "chain.struct.height"},
			},
			{
				ActionCode: "mine_fork_block", Label: "在指定 height 分叉",
				Description: "找 height 处的某块作为 prev，挖一个分叉块",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "fork_at_height", Type: fw.FieldNumber, Label: "分叉 height", Required: true, Default: 1, Min: 0, Step: 1},
					{Name: "miner", Type: fw.FieldString, Label: "矿工", Required: true, Default: "fork-miner"},
				},
				WritesOwnedFields: []string{"chain.struct.tip_hash"},
				LinkOwnerFields:   []string{"chain.struct.tip_hash"},
			},
			{
				ActionCode: "extend_fork", Label: "延伸分叉链",
				Description: "选 totalDiff 第二大的 tip 延伸 1 块",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "miner", Type: fw.FieldString, Label: "矿工", Required: true, Default: "fork-miner"},
				},
			},
			{
				ActionCode: "reorg", Label: "重组主链",
				Description: "重新选 totalDiff 最大的 tip 作为主链",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"chain.struct.tip_hash", "chain.struct.height"},
				LinkOwnerFields:   []string{"chain.struct.tip_hash", "chain.struct.height"},
			},
			{
				ActionCode: "verify_chain", Label: "验证主链完整性",
				Category: fw.ActionObserve, Trigger: fw.TriggerImmediate,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				LinkOwnerFields: []string{"chain.struct.integrity"},
			},
			{
				ActionCode: "tamper_block", Label: "篡改区块",
				Description: "修改某块的 miner 字段（不重新计算 hash），演示完整性破坏",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "height", Type: fw.FieldNumber, Label: "目标 height", Required: true, Default: 1, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_inject_corruption",
				Label:         "教师注入数据损坏",
				Description:   "仅教师可用，注入数据损坏用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneFault,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师注入数据损坏"},
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
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "Blockchain 初始化（仅创世块）", true)
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
	case "mine_block":
		miner := fw.MapStr(in.Params, "miner", "alice")
		tx := fw.MapInt(in.Params, "tx_count", 3)
		st.NextNonce++
		ts := uint32(1700000000 + len(st.Chain.BlocksByHash))
		b, err := st.Chain.addBlockOnTop(st.Chain.MainTip, miner, tx, ts, st.NextNonce)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.Chain.MainTip = b.Hash
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_block", fmt.Sprintf("挖出 #%d hash=%s...", b.Height, b.Hash[:12]), false)
		appendMineMicroSteps(&out.Render, b.Height)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_fork_block":
		h := fw.MapInt(in.Params, "fork_at_height", 1)
		miner := fw.MapStr(in.Params, "miner", "fork-miner")
		hashes, ok := st.Chain.BlocksByHeight[h]
		if !ok || len(hashes) == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("height %d 不存在", h)}, nil
		}
		// 取该 height 的第一个块作为分叉 prev（应该用其 prev 实现真正分叉）
		// 真正分叉：取 height-1 的某块作为 prev
		if h == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "无法在创世块分叉"}, nil
		}
		prevHashes, ok := st.Chain.BlocksByHeight[h-1]
		if !ok || len(prevHashes) == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "找不到 prev"}, nil
		}
		st.NextNonce++
		ts := uint32(1700000000 + len(st.Chain.BlocksByHash) + 1000)
		b, err := st.Chain.addBlockOnTop(prevHashes[0], miner, 0, ts, st.NextNonce)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_fork_block",
			fmt.Sprintf("分叉块 h=%d hash=%s...（未切换主链）", b.Height, b.Hash[:12]), false)
		appendForkMicroSteps(&out.Render, b.Height)
		return out, nil

	case "extend_fork":
		miner := fw.MapStr(in.Params, "miner", "fork-miner")
		// 找非主链 tip 中 totalDiff 最大的
		tips := st.Chain.allTips()
		if len(tips) < 2 {
			return fw.ActionOutput{Success: false, ErrorMessage: "目前无分叉链"}, nil
		}
		var altTip string
		altTD := -1
		for _, t := range tips {
			if t == st.Chain.MainTip {
				continue
			}
			td := st.Chain.totalDiff(t)
			if td > altTD {
				altTD = td
				altTip = t
			}
		}
		if altTip == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "无可延伸的分叉"}, nil
		}
		st.NextNonce++
		ts := uint32(1700000000 + len(st.Chain.BlocksByHash) + 2000)
		b, err := st.Chain.addBlockOnTop(altTip, miner, 1, ts, st.NextNonce)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "extend_fork", fmt.Sprintf("分叉链 +1 块（h=%d）", b.Height), false)
		return out, nil

	case "reorg":
		oldTip := st.Chain.MainTip
		st.Chain.pickMainTip()
		saveState(state, st)
		summary := "主链未变"
		if st.Chain.MainTip != oldTip {
			summary = fmt.Sprintf("⚠ 主链重组：%s... → %s...", oldTip[:12], st.Chain.MainTip[:12])
		}
		out.Render = buildEnvelope(st, "reorg", summary, false)
		appendReorgMicroSteps(&out.Render, st.Chain.MainTip != oldTip)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "verify_chain":
		ok, viol := st.Chain.verifyChain(st.Chain.MainTip)
		summary := "✓ 链完整"
		if !ok {
			summary = fmt.Sprintf("✗ 完整性破坏：%d 个违规", len(viol))
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "verify_chain", summary, false)
		appendVerifyMicroSteps(&out.Render, ok, viol)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "tamper_block":
		h := fw.MapInt(in.Params, "height", 1)
		hashes, ok := st.Chain.BlocksByHeight[h]
		if !ok || len(hashes) == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "height 不存在"}, nil
		}
		// 篡改：改 miner 字段，不重算 hash
		oldHash := hashes[0]
		b := st.Chain.BlocksByHash[oldHash]
		b.Miner = "TAMPERED-" + b.Miner
		st.Chain.BlocksByHash[oldHash] = b
		st.Tampered = true
		st.TamperedAt = h
		saveState(state, st)
		out.Render = buildEnvelope(st, "tamper_block",
			fmt.Sprintf("⚠ 已篡改 #%d 的 miner（hash 未重算 → verify 会失败）", h), false)
		appendTamperMicroSteps(&out.Render, h)
		return out, nil

	case "teacher_inject_corruption":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师注入数据损坏"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-corrupt-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 40)

	// 1) 树形布局：从 genesis 展开
	prims = append(prims, fw.PrimTreeLayout("chain-tree", "blk-"+st.Chain.GenesisHash, "top-down"))

	// 2) 主链节点集合
	mainChain := st.Chain.chainFromTip(st.Chain.MainTip)
	mainSet := map[string]bool{}
	for _, b := range mainChain {
		mainSet[b.Hash] = true
	}

	// 3) 所有节点
	for _, b := range st.Chain.BlocksByHash {
		role := "block"
		status := "normal"
		if b.Hash == st.Chain.GenesisHash {
			role = "genesis"
			status = "active"
		} else if b.Hash == st.Chain.MainTip {
			role = "tip"
			status = "active"
		} else if mainSet[b.Hash] {
			role = "main-block"
		} else {
			role = "fork-block"
			status = "warning"
		}
		if st.Tampered && b.Height == st.TamperedAt {
			status = "error"
		}
		label := fmt.Sprintf("h=%d\n%s\n%s", b.Height, b.Miner, b.Hash[:12])
		prims = append(prims, fw.PrimNode("blk-"+b.Hash, label, status, role))
	}

	// 4) 边（child → parent）
	for _, b := range st.Chain.BlocksByHash {
		if b.Height == 0 {
			continue
		}
		anim := ""
		if mainSet[b.Hash] {
			anim = "flow"
		}
		style := "solid"
		if !mainSet[b.Hash] {
			style = "dashed"
		}
		prims = append(prims, fw.PrimEdge(
			fmt.Sprintf("edge-%s-%s", b.Hash[:8], b.PrevHash[:8]),
			"blk-"+b.PrevHash, "blk-"+b.Hash, style, anim))
	}

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-chain",
		`B_i.\mathrm{prev\_hash} = H(B_{i-1});\quad \text{main} = \arg\max_t \sum_{B \in \text{path}(t)} \mathrm{diff}_B`, false))

	// 6) 状态参数
	mainTD := st.Chain.totalDiff(st.Chain.MainTip)
	mainHeight := 0
	if t, ok := st.Chain.BlocksByHash[st.Chain.MainTip]; ok {
		mainHeight = t.Height
	}
	tips := st.Chain.allTips()
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("总块数 = %d\n主链高度 = %d\n主链 tip = %s...\n主链 total_diff = %d\ntips 数 = %d (含主链)\n篡改 = %v",
			len(st.Chain.BlocksByHash), mainHeight, truncStr(st.Chain.MainTip, 16),
			mainTD, len(tips), st.Tampered),
		"text", nil, 8))

	// 7) 主链表
	mainLines := []string{"主链区块列表："}
	for _, b := range mainChain {
		mainLines = append(mainLines, fmt.Sprintf("  h=%d  miner=%-12s  hash=%s",
			b.Height, b.Miner, b.Hash[:24]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-main", strings.Join(mainLines, "\n"), "text", nil, 14))

	// 8) tips 表
	tipLines := []string{fmt.Sprintf("所有 tips（%d 个）：", len(tips))}
	for _, t := range tips {
		td := st.Chain.totalDiff(t)
		role := "fork"
		if t == st.Chain.MainTip {
			role = "MAIN"
		}
		tipB := st.Chain.BlocksByHash[t]
		tipLines = append(tipLines, fmt.Sprintf("  [%s] h=%d td=%d hash=%s",
			role, tipB.Height, td, t[:24]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-tips", strings.Join(tipLines, "\n"), "text", nil, 8))

	// 9) 动效
	prims = append(prims, fw.PrimGlow("glow-tip", "blk-"+st.Chain.MainTip, "success", 0.9))
	prims = append(prims, fw.PrimGlow("glow-genesis", "blk-"+st.Chain.GenesisHash, "info", 0.6))
	if st.Tampered {
		prims = append(prims, fw.PrimShake("shake-tampered", "cb-status", 0.4, 600))
	}
	prims = append(prims, fw.PrimPulse("pulse-tip", "blk-"+st.Chain.MainTip, "success", 1500))

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Chain 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	mainTD := st.Chain.totalDiff(st.Chain.MainTip)
	mainHeight := 0
	if t, ok := st.Chain.BlocksByHash[st.Chain.MainTip]; ok {
		mainHeight = t.Height
	}
	d := map[string]any{
		"total_blocks":    len(st.Chain.BlocksByHash),
		"main_height":     mainHeight,
		"tip_hash":        st.Chain.MainTip,
		"genesis_hash":    st.Chain.GenesisHash,
		"main_total_diff": mainTD,
		"tip_count":       len(st.Chain.allTips()),
		"tampered":        st.Tampered,
		"tampered_at":     st.TamperedAt,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendMineMicroSteps(env *fw.RenderEnvelope, h int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "mn-1", Label: fmt.Sprintf("准备 #%d 区块头（prev_hash=tip）", h), DurationMs: 400, HighlightIDs: []string{"cb-main"}, ParentPhase: "append"},
		{ID: "mn-2", Label: "计算 SHA-256 (header)", DurationMs: 400, HighlightIDs: []string{"formula-chain"}, FirePrimitives: []string{"pulse-tip"}},
		{ID: "mn-3", Label: "追加到主链 → 更新 tip", DurationMs: 400, HighlightIDs: []string{"chain-tree", "glow-tip"}, IsLinkTrigger: true},
	}
}

func appendForkMicroSteps(env *fw.RenderEnvelope, h int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "fk-1", Label: fmt.Sprintf("找 height %d-1 的某块作为 prev", h), DurationMs: 400, HighlightIDs: []string{"chain-tree"}},
		{ID: "fk-2", Label: "构造分叉块（prev != 主链 tip）", DurationMs: 400, HighlightIDs: []string{"cb-tips"}},
		{ID: "fk-3", Label: "存入 chain，主链不变（td 不够）", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

func appendReorgMicroSteps(env *fw.RenderEnvelope, changed bool) {
	tail := "主链未变"
	if changed {
		tail = "⚠ 切换到 td 更大的 tip"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "rg-1", Label: "枚举所有 tips", DurationMs: 400, HighlightIDs: []string{"cb-tips"}},
		{ID: "rg-2", Label: "计算每条链 totalDiff", DurationMs: 400, HighlightIDs: []string{"formula-chain"}},
		{ID: "rg-3", Label: tail, DurationMs: 400, HighlightIDs: []string{"glow-tip"}, FirePrimitives: []string{"pulse-tip"}, IsLinkTrigger: true},
	}
}

func appendVerifyMicroSteps(env *fw.RenderEnvelope, ok bool, viol []string) {
	tail := "✓ 全部 prev_hash 一致"
	if !ok {
		tail = fmt.Sprintf("✗ %d 个违规：%s", len(viol), strings.Join(viol, "; "))
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "vf-1", Label: "从 tip 回溯到 genesis", DurationMs: 500, HighlightIDs: []string{"chain-tree", "cb-main"}},
		{ID: "vf-2", Label: "每对 (B_i, B_{i-1}) 验证 prev_hash + 重算 hash", DurationMs: 500, HighlightIDs: []string{"formula-chain"}},
		{ID: "vf-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-status"}, FirePrimitives: []string{"pulse-tip"}, IsLinkTrigger: true},
	}
}

func appendTamperMicroSteps(env *fw.RenderEnvelope, h int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tm-1", Label: fmt.Sprintf("篡改 #%d 的 miner 字段", h), DurationMs: 400, HighlightIDs: []string{"chain-tree"}, FirePrimitives: []string{"shake-tampered"}},
		{ID: "tm-2", Label: "未重算 hash → 当前 hash 与 SHA-256(header) 不一致", DurationMs: 500, HighlightIDs: []string{"cb-main"}},
		{ID: "tm-3", Label: "verify_chain 时 → 检测出 hash mismatch", DurationMs: 500, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

// chainTipHeight 取主链 tip 高度。
func chainTipHeight(st snapState) int {
	if t, ok := st.Chain.BlocksByHash[st.Chain.MainTip]; ok {
		return t.Height
	}
	return 0
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "chain-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_chain",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"chain.struct.tip_hash", "chain.struct.height"},
		Payload:        map[string]any{"height": chainTipHeight(st)},
		SourceAnchorID: "chain-tip-anchor",
		TargetAnchorID: "integrity-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "chain.struct.tip_hash", "chain.struct.height")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	mainHeight := 0
	if t, ok := st.Chain.BlocksByHash[st.Chain.MainTip]; ok {
		mainHeight = t.Height
	}
	return map[string]any{
		"chain": map[string]any{
			"struct": map[string]any{
				"height":       mainHeight,
				"tip_hash":     st.Chain.MainTip,
				"genesis_hash": st.Chain.GenesisHash,
				"total_blocks": len(st.Chain.BlocksByHash),
				"tip_count":    len(st.Chain.allTips()),
				"tampered":     st.Tampered,
				"integrity":    !st.Tampered,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
