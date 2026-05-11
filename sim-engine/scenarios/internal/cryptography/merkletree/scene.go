// 模块：sim-engine/scenarios/internal/cryptography/merkletree
// 文件职责：CRY-05 Merkle Tree 场景的完整实现。
//
// SSOT 依据：06.md §4.3.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现二叉 Merkle 树（自底向上构造 + 包含证明），哈希函数复用
// scenarios/internal/cryptography/sha256hash.Sum256（同 internal 子树兄弟包，零外部依赖）：
//   · 叶子哈希：H(leaf_bytes)；
//   · 内部节点：H(left || right)；
//   · 奇数节点的最后一个复制自身配对（Bitcoin 风格）；
//   · 包含证明：从叶子向上提取兄弟节点序列 + 左右标记；
//   · 验证：用兄弟逐层 H 重算路径，与根比较。
//
// 教学决策（树形布局）：
//   - tree_layout 顶向下展示完整 Merkle 树；
//   - 叶子节点输入文本可见；内部节点显示 hash 前 8 hex；
//   - 选定叶子后，verify_path_highlight 高亮证明路径；
//   - 篡改叶子触发"根变化"对比，演示完整性。

package merkletree

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "merkle-tree"
	schemaVersion = "v1.0.0"
	algorithmType = "merkle"
	maxLeaves     = 32

	linkGroupCryptoVerify     = "crypto-verify-group"
	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkOwnerSubtree          = "proofs.merkle"
)

var defaultLeaves = []string{"tx-0", "tx-1", "tx-2", "tx-3", "tx-4", "tx-5", "tx-6", "tx-7"}

// =====================================================================
// Merkle 树构造（基于 sha256hash.Sum256）
// =====================================================================

// hashLeaf 计算叶子哈希 H(leaf)。
func hashLeaf(leaf string) [32]byte { return sha256hash.Sum256([]byte(leaf)) }

// hashPair 计算父节点 H(left || right)，左右各 32 字节。
func hashPair(left, right [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], left[:])
	copy(buf[32:], right[:])
	return sha256hash.Sum256(buf[:])
}

// merkleTree 完整的 Merkle 树（按层存储；levels[0]=叶子层；levels[end]=根）。
type merkleTree struct {
	Leaves []string
	Levels [][][32]byte // levels[i] = 第 i 层所有节点哈希
}

// buildMerkleTree 从叶子构造完整树（奇数节点最后一个复制自身配对，Bitcoin 风格）。
func buildMerkleTree(leaves []string) merkleTree {
	t := merkleTree{Leaves: append([]string{}, leaves...)}
	if len(leaves) == 0 {
		return t
	}
	level := make([][32]byte, len(leaves))
	for i, leaf := range leaves {
		level[i] = hashLeaf(leaf)
	}
	t.Levels = append(t.Levels, level)
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			var right [32]byte
			if i+1 < len(level) {
				right = level[i+1]
			} else {
				right = left // 奇数节点：复制最后一个
			}
			next = append(next, hashPair(left, right))
		}
		t.Levels = append(t.Levels, next)
		level = next
	}
	return t
}

// Root 返回树的根哈希；空树返回零值。
func (t merkleTree) Root() [32]byte {
	if len(t.Levels) == 0 {
		return [32]byte{}
	}
	return t.Levels[len(t.Levels)-1][0]
}

// proofStep 包含证明的单步：兄弟哈希 + 左右标记（true=兄弟在右）。
type proofStep struct {
	Sibling        [32]byte
	SiblingOnRight bool
	LevelIndex     int // 0=叶子层
}

// generateProof 生成第 leafIdx 个叶子的包含证明（从底向上的兄弟序列）。
func (t merkleTree) generateProof(leafIdx int) ([]proofStep, error) {
	if leafIdx < 0 || leafIdx >= len(t.Leaves) {
		return nil, fmt.Errorf("leaf_index %d 越界 [0,%d)", leafIdx, len(t.Leaves))
	}
	steps := make([]proofStep, 0, len(t.Levels)-1)
	idx := leafIdx
	for lvl := 0; lvl < len(t.Levels)-1; lvl++ {
		level := t.Levels[lvl]
		var sibling [32]byte
		var siblingOnRight bool
		if idx%2 == 0 {
			// 我是左孩子，兄弟在右
			if idx+1 < len(level) {
				sibling = level[idx+1]
			} else {
				sibling = level[idx] // 奇数节点 duplicate 自己
			}
			siblingOnRight = true
		} else {
			sibling = level[idx-1]
			siblingOnRight = false
		}
		steps = append(steps, proofStep{Sibling: sibling, SiblingOnRight: siblingOnRight, LevelIndex: lvl})
		idx /= 2
	}
	return steps, nil
}

// verifyProof 用证明从叶子重算，结果应等于 root。
func verifyProof(leaf string, proof []proofStep, root [32]byte) (bool, [][32]byte) {
	cur := hashLeaf(leaf)
	intermediate := make([][32]byte, 0, len(proof)+1)
	intermediate = append(intermediate, cur)
	for _, st := range proof {
		if st.SiblingOnRight {
			cur = hashPair(cur, st.Sibling)
		} else {
			cur = hashPair(st.Sibling, cur)
		}
		intermediate = append(intermediate, cur)
	}
	return cur == root, intermediate
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Leaves       []string
	TargetIndex  int
	TamperedIdx  int
	OriginalRoot string // 篡改前的根（演示完整性破坏）
	LastError    string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{Leaves: append([]string{}, defaultLeaves...), TargetIndex: 0, TamperedIdx: -1}
	}
	d := s.Data
	st := snapState{
		TargetIndex:  fw.MapInt(d, "target_index", 0),
		TamperedIdx:  fw.MapInt(d, "tampered_idx", -1),
		OriginalRoot: fw.MapStr(d, "original_root", ""),
	}
	if leaves, ok := d["leaves"].([]any); ok {
		for _, v := range leaves {
			if s, ok := v.(string); ok {
				st.Leaves = append(st.Leaves, s)
			}
		}
	}
	if len(st.Leaves) == 0 {
		st.Leaves = append([]string{}, defaultLeaves...)
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = make(map[string]any, 6)
	}
	leavesAny := make([]any, len(st.Leaves))
	for i, l := range st.Leaves {
		leavesAny[i] = l
	}
	s.Data["leaves"] = leavesAny
	s.Data["target_index"] = st.TargetIndex
	s.Data["tampered_idx"] = st.TamperedIdx
	s.Data["original_root"] = st.OriginalRoot
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "Merkle 树",
		Description:         "演示 Merkle 树自底向上构造、包含证明生成与验证、篡改检测",
		Category:            fw.CategoryCryptography,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify, linkGroupBlockchainIntegr},

		// v0.5 协议字段。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"proofs.merkle.root_hex",
			"proofs.merkle.leaf_count",
			"proofs.merkle.target_index",
			"proofs.merkle.proof_path",
			"proofs.merkle.tampered",
		},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultState,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultState() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data:      map[string]any{"target_index": 0, "tampered_idx": -1},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_leaves", Label: "设置叶子",
				Description:   "用逗号分隔的字符串作为叶子重建 Merkle 树",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "leaves_csv", Type: fw.FieldString, Label: "逗号分隔叶子列表", Required: true,
						Default: strings.Join(defaultLeaves, ",")},
				},
				WritesOwnedFields: []string{"proofs.merkle.root_hex", "proofs.merkle.leaf_count"},
				LinkOwnerFields:   []string{"proofs.merkle.root_hex", "proofs.merkle.leaf_count"},
			},
			{
				ActionCode: "select_leaf", Label: "选择叶子查看证明",
				Category:      fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "leaf_index", Type: fw.FieldNumber, Label: "叶子索引", Required: true, Default: 0, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"proofs.merkle.target_index", "proofs.merkle.proof_path"},
				LinkOwnerFields:   []string{"proofs.merkle.target_index", "proofs.merkle.proof_path"},
			},
			{
				ActionCode: "add_leaf", Label: "追加叶子",
				Description:   "追加一个新叶子并重建树（≤ 32）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "leaf", Type: fw.FieldString, Label: "新叶子内容", Required: true, Default: "tx-new"},
				},
			},
			{
				ActionCode: "tamper_leaf", Label: "篡改叶子",
				Description:   "修改某叶子内容，演示完整性破坏",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "leaf_index", Type: fw.FieldNumber, Label: "目标叶子索引", Required: true, Default: 0, Min: 0, Step: 1},
					{Name: "new_value", Type: fw.FieldString, Label: "新内容", Required: true, Default: "tampered"},
				},
				WritesOwnedFields: []string{"proofs.merkle.root_hex", "proofs.merkle.tampered"},
				LinkOwnerFields:   []string{"proofs.merkle.root_hex", "proofs.merkle.tampered"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
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
	saveState(state, st)
	state.Phase = "built"
	env := buildEnvelope(st, "init", "Merkle 树初构（叶子=8 默认）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(st, "tick", fmt.Sprintf("叶子=%d", len(st.Leaves)), false)
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
	case "set_leaves":
		csv := fw.MapStr(in.Params, "leaves_csv", "")
		if csv == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "leaves_csv 不能为空"}, nil
		}
		parts := strings.Split(csv, ",")
		leaves := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				leaves = append(leaves, p)
			}
		}
		if len(leaves) == 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "至少 1 个叶子"}, nil
		}
		if len(leaves) > maxLeaves {
			return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("叶子数 ≤ %d", maxLeaves)}, nil
		}
		st.Leaves = leaves
		st.TargetIndex = 0
		st.TamperedIdx = -1
		st.OriginalRoot = ""
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_leaves", fmt.Sprintf("重建 Merkle 树（叶子=%d）", len(leaves)), true)
		appendBuildMicroSteps(&out.Render, len(leaves))
		publishOwnerSubtree(&out.Render, st)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "select_leaf":
		idx := fw.MapInt(in.Params, "leaf_index", 0)
		if idx < 0 || idx >= len(st.Leaves) {
			return fw.ActionOutput{Success: false, ErrorMessage: "leaf_index 越界"}, nil
		}
		st.TargetIndex = idx
		saveState(state, st)
		out.Render = buildEnvelope(st, "select_leaf", fmt.Sprintf("生成叶子 #%d 的包含证明", idx), false)
		appendProofMicroSteps(&out.Render, idx)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "add_leaf":
		if len(st.Leaves) >= maxLeaves {
			return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("叶子已达上限 %d", maxLeaves)}, nil
		}
		newLeaf := fw.MapStr(in.Params, "leaf", "tx-new")
		st.Leaves = append(st.Leaves, newLeaf)
		st.TargetIndex = len(st.Leaves) - 1
		saveState(state, st)
		out.Render = buildEnvelope(st, "add_leaf", fmt.Sprintf("追加叶子 → 树重建（叶子=%d）", len(st.Leaves)), true)
		appendBuildMicroSteps(&out.Render, len(st.Leaves))
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "tamper_leaf":
		idx := fw.MapInt(in.Params, "leaf_index", 0)
		if idx < 0 || idx >= len(st.Leaves) {
			return fw.ActionOutput{Success: false, ErrorMessage: "leaf_index 越界"}, nil
		}
		newVal := fw.MapStr(in.Params, "new_value", "tampered")
		// 记录篡改前的根
		oldTree := buildMerkleTree(st.Leaves)
		oldRoot := oldTree.Root()
		st.OriginalRoot = hex.EncodeToString(oldRoot[:])
		st.Leaves[idx] = newVal
		st.TamperedIdx = idx
		saveState(state, st)
		out.Render = buildEnvelope(st, "tamper_leaf", fmt.Sprintf("篡改叶子 #%d → 根哈希变化", idx), false)
		appendTamperMicroSteps(&out.Render, idx)
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
		st.Leaves = append([]string{}, defaultLeaves...)
		st.TargetIndex = 0
		st.TamperedIdx = -1
		st.OriginalRoot = ""
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "重置为默认 8 叶子", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	tree := buildMerkleTree(st.Leaves)
	root := tree.Root()
	rootHex := hex.EncodeToString(root[:])

	// 生成证明 + 验证（教学：始终展示当前 target_index 的证明）
	proof, _ := tree.generateProof(st.TargetIndex)
	leafForProof := ""
	if st.TargetIndex < len(st.Leaves) {
		leafForProof = st.Leaves[st.TargetIndex]
	}
	proofValid, intermediate := verifyProof(leafForProof, proof, root)

	prims := make([]fw.Primitive, 0, 64)

	// 1) 树根 ID = "node-L<level>-<index>"
	rootID := fmt.Sprintf("node-L%d-0", len(tree.Levels)-1)
	prims = append(prims, fw.PrimTreeLayout("merkle-tree", rootID, "top-down"))

	// 2) 所有节点（按层 / 按索引）
	for lvl := len(tree.Levels) - 1; lvl >= 0; lvl-- {
		for idx, h := range tree.Levels[lvl] {
			id := fmt.Sprintf("node-L%d-%d", lvl, idx)
			short := hex.EncodeToString(h[:])[:12]
			role := "internal"
			status := "normal"
			label := short
			if lvl == 0 {
				role = "leaf"
				if idx < len(st.Leaves) {
					label = fmt.Sprintf("[%d] %s\n%s", idx, st.Leaves[idx], short)
				}
				if idx == st.TargetIndex {
					status = "active"
				}
				if idx == st.TamperedIdx {
					status = "warning"
				}
			} else if lvl == len(tree.Levels)-1 {
				role = "root"
				label = fmt.Sprintf("ROOT\n%s", short)
				status = "active"
			}
			prims = append(prims, fw.PrimNode(id, label, status, role))
		}
	}

	// 3) 父子边
	for lvl := 0; lvl < len(tree.Levels)-1; lvl++ {
		for idx := range tree.Levels[lvl] {
			parentIdx := idx / 2
			parentID := fmt.Sprintf("node-L%d-%d", lvl+1, parentIdx)
			childID := fmt.Sprintf("node-L%d-%d", lvl, idx)
			prims = append(prims, fw.PrimEdge(
				fmt.Sprintf("edge-%s-%s", parentID, childID),
				parentID, childID, "solid", ""))
		}
	}

	// 4) 证明路径高亮
	proofPathIDs := make([]string, 0, len(proof)+1)
	pathIdx := st.TargetIndex
	proofPathIDs = append(proofPathIDs, fmt.Sprintf("node-L0-%d", pathIdx))
	for lvl := 0; lvl < len(tree.Levels)-1; lvl++ {
		pathIdx /= 2
		proofPathIDs = append(proofPathIDs, fmt.Sprintf("node-L%d-%d", lvl+1, pathIdx))
	}
	prims = append(prims, fw.PrimVerifyPathHighlight("proof-path", proofPathIDs))

	// 5) 兄弟节点列表（code_block）
	siblingsLines := make([]string, 0, len(proof))
	siblingsLines = append(siblingsLines, fmt.Sprintf("叶子 #%d = %s", st.TargetIndex, leafForProof))
	for _, ps := range proof {
		side := "left"
		if ps.SiblingOnRight {
			side = "right"
		}
		siblingsLines = append(siblingsLines, fmt.Sprintf("L%d 兄弟(%s) = %s", ps.LevelIndex, side, hex.EncodeToString(ps.Sibling[:])[:16]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-proof", strings.Join(siblingsLines, "\n"), "text", nil, 16))

	// 6) 中间哈希（每层验证结果）
	intLines := []string{"验证中间哈希："}
	for i, h := range intermediate {
		intLines = append(intLines, fmt.Sprintf("  step %d: %s", i, hex.EncodeToString(h[:])[:24]))
	}
	intLines = append(intLines, "")
	if proofValid {
		intLines = append(intLines, "✓ 与根匹配 → 证明有效")
	} else {
		intLines = append(intLines, "✗ 与根不匹配 → 证明无效")
	}
	prims = append(prims, fw.PrimCodeBlock("cb-verify", strings.Join(intLines, "\n"), "text", nil, 12))

	// 7) 根哈希
	prims = append(prims, fw.PrimCodeBlock("cb-root", "Merkle Root = "+rootHex, "text", nil, 2))

	// 8) 篡改对比
	if st.OriginalRoot != "" && st.TamperedIdx >= 0 {
		tampLines := []string{
			fmt.Sprintf("篡改叶子: #%d", st.TamperedIdx),
			fmt.Sprintf("原 Root: %s", st.OriginalRoot),
			fmt.Sprintf("新 Root: %s", rootHex),
			"⚠ 根哈希变化 → 数据完整性被破坏",
		}
		prims = append(prims, fw.PrimCodeBlock("cb-tamper", strings.Join(tampLines, "\n"), "text", nil, 6))
	}

	// 9) 公式
	prims = append(prims, fw.PrimMathFormula("formula",
		`H_{\mathrm{parent}} = \mathrm{SHA256}(H_{\mathrm{left}}\,\|\,H_{\mathrm{right}})`, false))

	// 10) 进度条（构造完成度，整树构造一次性完成 → 始终满）
	prims = append(prims, fw.PrimProgressBar("build-progress", float64(len(tree.Levels)), float64(len(tree.Levels)), "构造层数"))

	// 11) glow 当前选中叶子 + root
	leafID := fmt.Sprintf("node-L0-%d", st.TargetIndex)
	prims = append(prims, fw.PrimGlow("glow-target", leafID, "info", 0.7))
	prims = append(prims, fw.PrimGlow("glow-root", rootID, "success", 0.9))

	// 12) pulse 验证结果
	pulseColor := "success"
	if !proofValid {
		pulseColor = "danger"
	}
	prims = append(prims, fw.PrimPulse("pulse-verify", "cb-verify", pulseColor, 1500))

	// 13) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Merkle Tree 错误", st.LastError, "scene", "请检查输入", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, tree, rootHex, proofValid, summary),
	}
}

func buildSidePanelData(st snapState, tree merkleTree, rootHex string, proofValid bool, summary string) map[string]any {
	d := map[string]any{
		"leaf_count":    len(st.Leaves),
		"tree_levels":   len(tree.Levels),
		"target_index":  st.TargetIndex,
		"tampered_idx":  st.TamperedIdx,
		"root_hex":      rootHex,
		"proof_valid":   proofValid,
		"original_root": st.OriginalRoot,
	}
	if summary != "" {
		d["summary"] = summary
	}
	if st.TamperedIdx >= 0 && st.OriginalRoot != "" {
		d["root_changed"] = (rootHex != st.OriginalRoot)
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendBuildMicroSteps(env *fw.RenderEnvelope, leafCount int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "build-1", Label: fmt.Sprintf("叶子哈希（%d 个 SHA-256）", leafCount), DurationMs: 600,
			HighlightIDs: []string{"merkle-tree"}, FirePrimitives: []string{"glow-target"}, ParentPhase: "build"},
		{ID: "build-2", Label: "自底向上配对哈希", DurationMs: 800,
			HighlightIDs: []string{"merkle-tree", "formula"}},
		{ID: "build-3", Label: "得到 Merkle Root", DurationMs: 600,
			HighlightIDs: []string{"cb-root", "glow-root"}, FirePrimitives: []string{"glow-root"}, IsLinkTrigger: true},
	}
}

func appendProofMicroSteps(env *fw.RenderEnvelope, leafIdx int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "proof-1", Label: fmt.Sprintf("提取叶子 #%d 的兄弟序列", leafIdx), DurationMs: 500,
			HighlightIDs: []string{fmt.Sprintf("node-L0-%d", leafIdx), "cb-proof"}, FirePrimitives: []string{"glow-target"}},
		{ID: "proof-2", Label: "用兄弟逐层向上重算 SHA-256", DurationMs: 800,
			HighlightIDs: []string{"proof-path", "cb-verify"}},
		{ID: "proof-3", Label: "与 Root 比较 → 证明有效", DurationMs: 600,
			HighlightIDs: []string{"cb-root", "cb-verify"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
	}
}

func appendTamperMicroSteps(env *fw.RenderEnvelope, idx int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tamp-1", Label: fmt.Sprintf("篡改叶子 #%d 内容", idx), DurationMs: 500,
			HighlightIDs: []string{fmt.Sprintf("node-L0-%d", idx)}, FirePrimitives: []string{"glow-target"}},
		{ID: "tamp-2", Label: "重新自底向上哈希所有内部节点", DurationMs: 700,
			HighlightIDs: []string{"merkle-tree"}},
		{ID: "tamp-3", Label: "新 Root ≠ 原 Root → 完整性破坏", DurationMs: 700,
			HighlightIDs: []string{"cb-tamper", "cb-root"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	// 计算根哈希（snapState 不缓存，从叶子重建）。
	tree := buildMerkleTree(st.Leaves)
	root := tree.Root()
	rootHex := hex.EncodeToString(root[:])
	// LinkTrigger 带锚点（§0.7.1 C18）。
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "merkle-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_root",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"proofs.merkle.root_hex"},
		Payload:        map[string]any{"root_hex": rootHex, "leaf_count": len(st.Leaves)},
		SourceAnchorID: "merkle-root-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "proofs.merkle.root_hex")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	tree := buildMerkleTree(st.Leaves)
	root := tree.Root()
	proof, _ := tree.generateProof(st.TargetIndex)
	proofPathHex := make([]string, 0, len(proof))
	for _, s := range proof {
		side := "L"
		if s.SiblingOnRight {
			side = "R"
		}
		proofPathHex = append(proofPathHex, side+":"+hex.EncodeToString(s.Sibling[:]))
	}
	return map[string]any{
		"proofs": map[string]any{
			"merkle": map[string]any{
				"root_hex":     hex.EncodeToString(root[:]),
				"leaf_count":   len(st.Leaves),
				"target_index": st.TargetIndex,
				"proof_path":   proofPathHex,
				"tampered":     st.TamperedIdx >= 0,
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
