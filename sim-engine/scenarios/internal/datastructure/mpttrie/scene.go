// 模块：sim-engine/scenarios/internal/datastructure/mpttrie
// 文件职责：DS-03 修改型 Merkle Patricia Trie（MPT）场景的完整实现。
//
// SSOT 依据：06.md §4.4.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以太坊风格 MPT，零外部依赖；复用 sha256hash.Sum256 计算节点 hash：
//
//   · 4 节点类型：
//     · empty / null
//     · leaf: { path: nibbles, value }
//     · extension: { path: nibbles, child: ref }
//     · branch: { children[16]: ref, value: string }
//   · 路径表示：把 key 字符串映射到 nibbles（每字符 1 nibble，0..15）
//   · put(key, value) 递归：
//     · 空节点 → leaf
//     · leaf  → 比较 path：完全相同则更新；否则分裂为 branch（可能加 extension 前缀）
//     · ext   → 比较 path：完全包含则递归；否则分裂
//     · branch → 路径首 nibble 索引到子节点递归；空路径则设 branch.value
//   · get(key) 递归遍历返回 value
//   · root hash = SHA-256(serialize(root_node))，子节点 hash 在 serialize 中递归引用
//
// 教学：用 hex 字符串作为 key（每字符 1 nibble）以便清晰可读。

package mpttrie

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

const (
	sceneCode     = "mpt-trie"
	schemaVersion = "v1.0.0"
	algorithmType = "mpt"

	maxKeys = 32

	linkGroupCryptoVerify     = "crypto-verify-group"
	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkOwnerSubtree          = "datastructure.mpt"
)

// =====================================================================
// MPT 节点
// =====================================================================

type nodeType int

const (
	nodeEmpty nodeType = iota
	nodeLeaf
	nodeExtension
	nodeBranch
)

func (t nodeType) String() string {
	return []string{"empty", "leaf", "extension", "branch"}[t]
}

// mptNode 是 MPT 节点的统一结构（按 Type 区分含义）。
type mptNode struct {
	Type     nodeType
	Path     []byte       // nibbles（leaf / extension）
	Value    string       // leaf / branch.value
	Children [16]*mptNode // branch.children
}

// keyToNibbles 把字符串转换为 nibble 序列（每字符 1 nibble，仅支持 0-9 / a-f）。
func keyToNibbles(key string) []byte {
	out := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		var n byte
		switch {
		case c >= '0' && c <= '9':
			n = c - '0'
		case c >= 'a' && c <= 'f':
			n = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			n = c - 'A' + 10
		default:
			n = c & 0xf // fallback
		}
		out = append(out, n)
	}
	return out
}

// nibblesToHex 把 nibble 序列回到 hex 字符串。
func nibblesToHex(ns []byte) string {
	out := make([]byte, len(ns))
	for i, n := range ns {
		if n < 10 {
			out[i] = '0' + n
		} else {
			out[i] = 'a' + (n - 10)
		}
	}
	return string(out)
}

// commonPrefix 返回 a 与 b 的最长公共前缀长度。
func commonPrefix(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// =====================================================================
// MPT 操作（递归）
// =====================================================================

// put 在 node 上插入 (path, value)；返回新的子树根。
func put(node *mptNode, path []byte, value string) *mptNode {
	// 1) 空节点
	if node == nil || node.Type == nodeEmpty {
		return &mptNode{Type: nodeLeaf, Path: append([]byte{}, path...), Value: value}
	}
	// 2) leaf
	if node.Type == nodeLeaf {
		cp := commonPrefix(node.Path, path)
		// 完全相同 → 更新 value
		if cp == len(node.Path) && cp == len(path) {
			return &mptNode{Type: nodeLeaf, Path: append([]byte{}, node.Path...), Value: value}
		}
		// 分裂：构造 branch，把原 leaf + 新 leaf 各自下放
		branch := &mptNode{Type: nodeBranch}
		// 原 leaf 剩余路径
		oldRest := node.Path[cp:]
		if len(oldRest) == 0 {
			branch.Value = node.Value
		} else {
			branch.Children[oldRest[0]] = &mptNode{
				Type: nodeLeaf, Path: append([]byte{}, oldRest[1:]...), Value: node.Value,
			}
		}
		// 新 leaf 剩余路径
		newRest := path[cp:]
		if len(newRest) == 0 {
			branch.Value = value
		} else {
			branch.Children[newRest[0]] = &mptNode{
				Type: nodeLeaf, Path: append([]byte{}, newRest[1:]...), Value: value,
			}
		}
		// 公共前缀 → extension 包裹
		if cp > 0 {
			return &mptNode{Type: nodeExtension, Path: append([]byte{}, node.Path[:cp]...), Children: [16]*mptNode{0: branch}}
		}
		return branch
	}
	// 3) extension
	if node.Type == nodeExtension {
		cp := commonPrefix(node.Path, path)
		// path 完全包含 extension.path → 递归到 child
		if cp == len(node.Path) {
			child := node.Children[0]
			newChild := put(child, path[cp:], value)
			return &mptNode{Type: nodeExtension, Path: append([]byte{}, node.Path...), Children: [16]*mptNode{0: newChild}}
		}
		// 否则分裂
		branch := &mptNode{Type: nodeBranch}
		// 原 extension 剩余
		oldRest := node.Path[cp:]
		if len(oldRest) == 1 {
			// 直接把 child 挂到 branch.children[oldRest[0]]
			branch.Children[oldRest[0]] = node.Children[0]
		} else {
			// extension 剩余 ≥ 2 nibble → 缩短的 extension 放到 branch
			branch.Children[oldRest[0]] = &mptNode{
				Type: nodeExtension, Path: append([]byte{}, oldRest[1:]...),
				Children: [16]*mptNode{0: node.Children[0]},
			}
		}
		// 新 path 剩余
		newRest := path[cp:]
		if len(newRest) == 0 {
			branch.Value = value
		} else {
			branch.Children[newRest[0]] = &mptNode{
				Type: nodeLeaf, Path: append([]byte{}, newRest[1:]...), Value: value,
			}
		}
		if cp > 0 {
			return &mptNode{Type: nodeExtension, Path: append([]byte{}, node.Path[:cp]...), Children: [16]*mptNode{0: branch}}
		}
		return branch
	}
	// 4) branch
	if node.Type == nodeBranch {
		newNode := *node // 浅复制（教学版可接受）
		if len(path) == 0 {
			newNode.Value = value
			return &newNode
		}
		idx := path[0]
		newNode.Children[idx] = put(node.Children[idx], path[1:], value)
		return &newNode
	}
	return node
}

// get 在 node 上查找 path 对应的 value；找不到返回 ("", false)。
func get(node *mptNode, path []byte) (string, bool) {
	if node == nil || node.Type == nodeEmpty {
		return "", false
	}
	switch node.Type {
	case nodeLeaf:
		if len(node.Path) != len(path) {
			return "", false
		}
		for i := range node.Path {
			if node.Path[i] != path[i] {
				return "", false
			}
		}
		return node.Value, true
	case nodeExtension:
		if len(path) < len(node.Path) {
			return "", false
		}
		for i := range node.Path {
			if node.Path[i] != path[i] {
				return "", false
			}
		}
		return get(node.Children[0], path[len(node.Path):])
	case nodeBranch:
		if len(path) == 0 {
			if node.Value != "" {
				return node.Value, true
			}
			return "", false
		}
		return get(node.Children[path[0]], path[1:])
	}
	return "", false
}

// nodeHash 计算节点 hash（递归：含子节点 hash）。
func nodeHash(node *mptNode) [32]byte {
	if node == nil || node.Type == nodeEmpty {
		return sha256hash.Sum256([]byte{})
	}
	buf := []byte{byte(node.Type)}
	switch node.Type {
	case nodeLeaf:
		buf = append(buf, byte(len(node.Path)))
		buf = append(buf, node.Path...)
		buf = append(buf, []byte(node.Value)...)
	case nodeExtension:
		buf = append(buf, byte(len(node.Path)))
		buf = append(buf, node.Path...)
		ch := nodeHash(node.Children[0])
		buf = append(buf, ch[:]...)
	case nodeBranch:
		for i := 0; i < 16; i++ {
			ch := nodeHash(node.Children[i])
			buf = append(buf, ch[:]...)
		}
		buf = append(buf, []byte(node.Value)...)
	}
	return sha256hash.Sum256(buf)
}

// proofPath 返回从 root 到 path 终点的节点序列（hash[]）。
func proofPath(node *mptNode, path []byte) [][32]byte {
	out := [][32]byte{}
	if node == nil || node.Type == nodeEmpty {
		return out
	}
	out = append(out, nodeHash(node))
	switch node.Type {
	case nodeLeaf:
		return out
	case nodeExtension:
		if len(path) < len(node.Path) {
			return out
		}
		for i := range node.Path {
			if node.Path[i] != path[i] {
				return out
			}
		}
		return append(out, proofPath(node.Children[0], path[len(node.Path):])...)
	case nodeBranch:
		if len(path) == 0 {
			return out
		}
		return append(out, proofPath(node.Children[path[0]], path[1:])...)
	}
	return out
}

// flattenNodes 把 trie 所有节点扁平化为列表（含 ID 路径），用于可视化。
type flatNode struct {
	ID       string // "n0", "n1", ...
	Type     nodeType
	PathHex  string
	Value    string
	ChildIDs []string // branch: 16 children；extension: 1 child
	Hash     string
}

func flatten(root *mptNode) []flatNode {
	out := []flatNode{}
	idCounter := 0
	var visit func(n *mptNode) string
	visit = func(n *mptNode) string {
		if n == nil || n.Type == nodeEmpty {
			return ""
		}
		myID := fmt.Sprintf("n%d", idCounter)
		idCounter++
		fn := flatNode{
			ID:      myID,
			Type:    n.Type,
			PathHex: nibblesToHex(n.Path),
			Value:   n.Value,
		}
		h := nodeHash(n)
		fn.Hash = hex.EncodeToString(h[:])
		// 暂存 node 自身位置；children 后处理
		idx := len(out)
		out = append(out, fn)
		switch n.Type {
		case nodeExtension:
			cid := visit(n.Children[0])
			out[idx].ChildIDs = []string{cid}
		case nodeBranch:
			cids := make([]string, 16)
			for i := 0; i < 16; i++ {
				cids[i] = visit(n.Children[i])
			}
			out[idx].ChildIDs = cids
		}
		return myID
	}
	visit(root)
	return out
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Keys      []string // 已插入的 key 列表（顺序保留）
	Values    map[string]string
	Root      *mptNode
	LastQuery string
	LastFound bool
	LastValue string
	Tampered  bool
	LastError string
}

func defaultSnapState() snapState {
	st := snapState{Values: map[string]string{}}
	// 默认插入几个 key 演示
	defaults := [][2]string{
		{"a711", "alice"},
		{"a755", "bob"},
		{"a77d", "carol"},
		{"b123", "dave"},
	}
	for _, kv := range defaults {
		st.Root = put(st.Root, keyToNibbles(kv[0]), kv[1])
		st.Keys = append(st.Keys, kv[0])
		st.Values[kv[0]] = kv[1]
	}
	return st
}

// =====================================================================
// 持久化（仅保存 Keys/Values，每次重建 trie）
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return defaultSnapState()
	}
	d := s.Data
	st := snapState{
		Values:    map[string]string{},
		LastQuery: fw.MapStr(d, "last_query", ""),
		LastFound: fw.MapBool(d, "last_found", false),
		LastValue: fw.MapStr(d, "last_value", ""),
		Tampered:  fw.MapBool(d, "tampered", false),
		LastError: fw.MapStr(d, "last_error", ""),
	}
	if keysAny, ok := d["keys"].([]any); ok {
		for _, k := range keysAny {
			if s, ok := k.(string); ok {
				st.Keys = append(st.Keys, s)
			}
		}
	}
	if valsAny, ok := d["values"].(map[string]any); ok {
		for k, v := range valsAny {
			if s, ok := v.(string); ok {
				st.Values[k] = s
			}
		}
	}
	if len(st.Keys) == 0 {
		return defaultSnapState()
	}
	// 重建 trie
	for _, k := range st.Keys {
		st.Root = put(st.Root, keyToNibbles(k), st.Values[k])
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["last_query"] = st.LastQuery
	s.Data["last_found"] = st.LastFound
	s.Data["last_value"] = st.LastValue
	s.Data["tampered"] = st.Tampered
	s.Data["last_error"] = st.LastError
	keysAny := make([]any, len(st.Keys))
	for i, k := range st.Keys {
		keysAny[i] = k
	}
	s.Data["keys"] = keysAny
	valsAny := map[string]any{}
	for k, v := range st.Values {
		valsAny[k] = v
	}
	s.Data["values"] = valsAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "Merkle Patricia Trie",
		Description:         "演示 MPT 4 节点类型（empty/leaf/extension/branch）+ put/get + root hash + 包含证明",
		Category:            fw.CategoryDataStructure,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify, linkGroupBlockchainIntegr},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"datastructure.mpt.root_hash",
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
				ActionCode: "put", Label: "插入键值对",
				Description:   "key 必须是 hex 字符串（每字符 = 1 nibble，0-9 a-f）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "key", Type: fw.FieldString, Label: "key (hex)", Required: true, Default: "a777"},
					{Name: "value", Type: fw.FieldString, Label: "value", Required: true, Default: "eve"},
				},
				WritesOwnedFields: []string{"datastructure.mpt.root_hash"},
				LinkOwnerFields:   []string{"datastructure.mpt.root_hash"},
			},
			{
				ActionCode: "get", Label: "查询键值",
				Category:      fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "key", Type: fw.FieldString, Label: "key (hex)", Required: true, Default: "a711"},
				},
				LinkOwnerFields: []string{"datastructure.mpt.last_query"},
			},
			{
				ActionCode: "compute_root", Label: "重算 root hash",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode: "tamper_value", Label: "篡改 value",
				Description:   "改某 key 的 value，重建 trie → root 变化",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "key", Type: fw.FieldString, Label: "key", Required: true, Default: "a711"},
					{Name: "new_value", Type: fw.FieldString, Label: "新 value", Required: true, Default: "tampered"},
				},
				WritesOwnedFields: []string{"datastructure.mpt.root_hash"},
				LinkOwnerFields:   []string{"datastructure.mpt.root_hash"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
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
	env := buildEnvelope(st, "init", "MPT 初始化（4 默认 key）", true)
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
	case "put":
		key := fw.MapStr(in.Params, "key", "")
		value := fw.MapStr(in.Params, "value", "")
		if key == "" || !isHexKey(key) {
			return fw.ActionOutput{Success: false, ErrorMessage: "key 必须为 hex 字符串"}, nil
		}
		if len(st.Keys) >= maxKeys {
			if _, has := st.Values[key]; !has {
				return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("key 数量 ≥ %d", maxKeys)}, nil
			}
		}
		_, exists := st.Values[key]
		st.Root = put(st.Root, keyToNibbles(key), value)
		st.Values[key] = value
		if !exists {
			st.Keys = append(st.Keys, key)
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "put", fmt.Sprintf("put(%s, %s)", key, value), false)
		appendPutMicroSteps(&out.Render, key, exists)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "get":
		key := fw.MapStr(in.Params, "key", "")
		v, found := get(st.Root, keyToNibbles(key))
		st.LastQuery = key
		st.LastFound = found
		st.LastValue = v
		saveState(state, st)
		summary := fmt.Sprintf("get(%s) → not found", key)
		if found {
			summary = fmt.Sprintf("get(%s) → \"%s\"", key, v)
		}
		out.Render = buildEnvelope(st, "get", summary, false)
		appendGetMicroSteps(&out.Render, key, found)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "compute_root":
		saveState(state, st)
		h := nodeHash(st.Root)
		out.Render = buildEnvelope(st, "compute_root", "root hash = "+hex.EncodeToString(h[:])[:24]+"...", false)
		return out, nil

	case "tamper_value":
		key := fw.MapStr(in.Params, "key", "")
		nv := fw.MapStr(in.Params, "new_value", "tampered")
		if _, has := st.Values[key]; !has {
			return fw.ActionOutput{Success: false, ErrorMessage: "key 不存在: " + key}, nil
		}
		oldRoot := nodeHash(st.Root)
		st.Values[key] = nv
		// 重建 trie
		st.Root = nil
		for _, k := range st.Keys {
			st.Root = put(st.Root, keyToNibbles(k), st.Values[k])
		}
		newRoot := nodeHash(st.Root)
		st.Tampered = true
		saveState(state, st)
		out.Render = buildEnvelope(st, "tamper_value",
			fmt.Sprintf("篡改 %s → root 从 %s... 变为 %s...",
				key, hex.EncodeToString(oldRoot[:])[:12], hex.EncodeToString(newRoot[:])[:12]), false)
		appendTamperMicroSteps(&out.Render, key)
		out.SharedStateDiff = ownerDiff(st)
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

func isHexKey(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(s) > 0
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 30)

	flat := flatten(st.Root)

	// 1) tree 布局
	rootID := ""
	if len(flat) > 0 {
		rootID = "mpt-" + flat[0].ID
	}
	prims = append(prims, fw.PrimTreeLayout("mpt-tree", rootID, "top-down"))

	// 2) 节点
	for _, fn := range flat {
		role := fn.Type.String()
		status := "normal"
		switch fn.Type {
		case nodeBranch:
			status = "active"
			role = "branch"
		case nodeExtension:
			role = "extension"
		case nodeLeaf:
			role = "leaf"
		}
		var label string
		switch fn.Type {
		case nodeLeaf:
			label = fmt.Sprintf("LEAF\npath=%s\n=\"%s\"", fn.PathHex, fn.Value)
		case nodeExtension:
			label = fmt.Sprintf("EXT\npath=%s", fn.PathHex)
		case nodeBranch:
			cnt := 0
			for _, c := range fn.ChildIDs {
				if c != "" {
					cnt++
				}
			}
			vstr := ""
			if fn.Value != "" {
				vstr = fmt.Sprintf("\n.v=\"%s\"", fn.Value)
			}
			label = fmt.Sprintf("BRANCH\nchildren=%d/16%s", cnt, vstr)
		}
		prims = append(prims, fw.PrimNode("mpt-"+fn.ID, label, status, role))
	}

	// 3) 父子边
	for _, fn := range flat {
		switch fn.Type {
		case nodeExtension:
			if len(fn.ChildIDs) > 0 && fn.ChildIDs[0] != "" {
				prims = append(prims, fw.PrimEdge(
					"edge-"+fn.ID+"-"+fn.ChildIDs[0],
					"mpt-"+fn.ID, "mpt-"+fn.ChildIDs[0], "solid", ""))
			}
		case nodeBranch:
			for i, cid := range fn.ChildIDs {
				if cid == "" {
					continue
				}
				prims = append(prims, fw.PrimEdge(
					fmt.Sprintf("edge-%s-%d-%s", fn.ID, i, cid),
					"mpt-"+fn.ID, "mpt-"+cid, "solid", ""))
			}
		}
	}

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-hash",
		`H(\text{leaf}) = \mathrm{SHA256}(\text{type}\,\|\,\text{path}\,\|\,\text{value});\ \ H(\text{branch}) = \mathrm{SHA256}(\text{type}\,\|\,H_0\,\|\,\dots\,\|\,H_{15}\,\|\,v)`, false))

	// 5) root hash
	rootH := nodeHash(st.Root)
	rootHex := hex.EncodeToString(rootH[:])
	prims = append(prims, fw.PrimCodeBlock("cb-root", "root_hash = "+rootHex, "text", nil, 4))

	// 6) keys 表
	rows := []string{"key       value           path nibbles"}
	sortedKeys := append([]string{}, st.Keys...)
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		rows = append(rows, fmt.Sprintf("%-9s %-15s [%s]", k, st.Values[k], nibblesToHex(keyToNibbles(k))))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-keys", strings.Join(rows, "\n"), "text", nil, 12))

	// 7) 节点列表
	nodeRows := []string{fmt.Sprintf("共 %d 节点：", len(flat))}
	for _, fn := range flat {
		nodeRows = append(nodeRows, fmt.Sprintf("  %s [%s] path=%s val=%q hash=%s",
			fn.ID, fn.Type.String(), fn.PathHex, fn.Value, fn.Hash[:16]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-nodes", strings.Join(nodeRows, "\n"), "text", nil, 14))

	// 8) 上次查询
	if st.LastQuery != "" {
		queryNibbles := keyToNibbles(st.LastQuery)
		proof := proofPath(st.Root, queryNibbles)
		queryLines := []string{
			fmt.Sprintf("last_query = %s", st.LastQuery),
			fmt.Sprintf("found      = %v", st.LastFound),
			fmt.Sprintf("value      = %s", st.LastValue),
			fmt.Sprintf("nibbles    = [%s]", nibblesToHex(queryNibbles)),
			"",
			fmt.Sprintf("proof path （%d 层）：", len(proof)),
		}
		for i, h := range proof {
			queryLines = append(queryLines, fmt.Sprintf("  L%d hash = %s", i, hex.EncodeToString(h[:])[:24]))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-query", strings.Join(queryLines, "\n"), "text", nil, 14))
	}

	// 9) 动效
	prims = append(prims, fw.PrimGlow("glow-root", "mpt-"+flat[0].ID, "success", 0.9))
	if st.LastFound {
		prims = append(prims, fw.PrimPulse("pulse-found", "cb-query", "success", 1500))
	} else if st.LastQuery != "" {
		prims = append(prims, fw.PrimPulse("pulse-notfound", "cb-query", "warning", 1500))
	}
	if st.Tampered {
		prims = append(prims, fw.PrimShake("shake-tampered", "cb-root", 0.4, 700))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "MPT 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, flat, rootHex, summary),
	}
}

func buildSidePanelData(st snapState, flat []flatNode, rootHex, summary string) map[string]any {
	branchCnt, leafCnt, extCnt := 0, 0, 0
	for _, fn := range flat {
		switch fn.Type {
		case nodeBranch:
			branchCnt++
		case nodeLeaf:
			leafCnt++
		case nodeExtension:
			extCnt++
		}
	}
	d := map[string]any{
		"key_count":       len(st.Keys),
		"node_count":      len(flat),
		"branch_count":    branchCnt,
		"leaf_count":      leafCnt,
		"extension_count": extCnt,
		"root_hash":       rootHex,
		"last_query":      st.LastQuery,
		"last_found":      st.LastFound,
		"last_value":      st.LastValue,
		"tampered":        st.Tampered,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendPutMicroSteps(env *fw.RenderEnvelope, key string, exists bool) {
	tail := "新建 leaf 节点"
	if exists {
		tail = "更新已有 key 的 value"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "p-1", Label: fmt.Sprintf("把 key %q 转成 nibble 序列", key), DurationMs: 400, HighlightIDs: []string{"cb-keys"}},
		{ID: "p-2", Label: "递归遍历到分裂点", DurationMs: 500, HighlightIDs: []string{"mpt-tree"}, FirePrimitives: []string{"glow-root"}},
		{ID: "p-3", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-nodes"}},
		{ID: "p-4", Label: "重算 root_hash", DurationMs: 400, HighlightIDs: []string{"cb-root", "formula-hash"}, IsLinkTrigger: true},
	}
}

func appendGetMicroSteps(env *fw.RenderEnvelope, key string, found bool) {
	tail := "未找到（路径中断）"
	if found {
		tail = "✓ 找到 leaf 的 value"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "g-1", Label: fmt.Sprintf("把 key %q 转成 nibble 序列", key), DurationMs: 400, HighlightIDs: []string{"cb-query"}},
		{ID: "g-2", Label: "从 root 沿 nibble 路径递归", DurationMs: 500, HighlightIDs: []string{"mpt-tree"}},
		{ID: "g-3", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-query"}, FirePrimitives: []string{"pulse-found", "pulse-notfound"}, IsLinkTrigger: true},
	}
}

func appendTamperMicroSteps(env *fw.RenderEnvelope, key string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tm-1", Label: "篡改 " + key + " 的 value", DurationMs: 400, HighlightIDs: []string{"cb-keys"}, FirePrimitives: []string{"shake-tampered"}},
		{ID: "tm-2", Label: "重建 trie 后 leaf hash 变化", DurationMs: 500, HighlightIDs: []string{"mpt-tree"}},
		{ID: "tm-3", Label: "向上递归改变所有祖先 hash → root 变化", DurationMs: 500, HighlightIDs: []string{"cb-root", "formula-hash"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

// mptRootHex 返回当前 trie root hash 的 hex 字符串。
func mptRootHex(st snapState) string {
	h := nodeHash(st.Root)
	return hex.EncodeToString(h[:])
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "mpt-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_mpt",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"datastructure.mpt.root_hash"},
		Payload:        map[string]any{"root_hash": mptRootHex(st), "key_count": len(st.Keys)},
		SourceAnchorID: "mpt-root-anchor",
		TargetAnchorID: "integrity-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "datastructure.mpt.root_hash")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	rootH := nodeHash(st.Root)
	return map[string]any{
		"datastructure": map[string]any{
			"mpt": map[string]any{
				"root_hash":  hex.EncodeToString(rootH[:]),
				"key_count":  len(st.Keys),
				"last_query": st.LastQuery,
				"last_found": st.LastFound,
				"tampered":   st.Tampered,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

