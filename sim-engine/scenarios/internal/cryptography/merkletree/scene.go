package merkletree

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 Merkle 树构建验证场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "merkle-tree",
		Title:        "Merkle 树构建验证",
		Phase:        "叶子哈希",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 900,
		TotalTicks:   16,
		Stages:       []string{"叶子哈希", "两两合并", "根节点输出", "路径验证"},
		Nodes: []framework.Node{
			{ID: "leaf-1", Label: "Leaf-1", Status: "normal", Role: "leaf", X: 80, Y: 280},
			{ID: "leaf-2", Label: "Leaf-2", Status: "normal", Role: "leaf", X: 220, Y: 280},
			{ID: "leaf-3", Label: "Leaf-3", Status: "normal", Role: "leaf", X: 360, Y: 280},
			{ID: "leaf-4", Label: "Leaf-4", Status: "normal", Role: "leaf", X: 500, Y: 280},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化四个叶子节点及其原始数据。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := treeModel{
		Leaves: []leafValue{
			{ID: "leaf-1", Data: "tx-a"},
			{ID: "leaf-2", Data: "tx-b"},
			{ID: "leaf-3", Data: "tx-c"},
			{ID: "leaf-4", Data: "tx-d"},
		},
	}
	applySharedTreeState(&model, input.SharedState, state.LinkGroup)
	recalculateTree(&model)
	return rebuildState(state, model, "叶子哈希")
}

// Step 推进叶子哈希、父节点合并、根输出和验证路径检查。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedTreeState(&model, input.SharedState, state.LinkGroup)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "叶子哈希"))
	switch phase {
	case "两两合并":
		recalculateParents(&model)
	case "根节点输出":
		recalculateRoot(&model)
	case "路径验证":
		model.Verified = verifyPath(model)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("Merkle 树进入%s阶段。", phase), toneByVerification(model.Verified, phase))
	return framework.StepOutput{
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// HandleAction 允许篡改单个叶子数据并重新计算整棵树。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	index := int(framework.NumberValue(input.Params["leaf"], 0))
	if index < 0 || index >= len(model.Leaves) {
		index = 0
	}
	model.Leaves[index].Data = model.Leaves[index].Data + "-tampered"
	recalculateTree(&model)
	model.Verified = false
	if err := rebuildState(state, model, "路径验证"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "篡改叶子", fmt.Sprintf("已篡改 %s 的数据。", model.Leaves[index].ID), "warning")
	return framework.ActionOutput{
		Success:    true,
		Events:     []framework.TimelineEvent{event},
		SharedDiff: buildSharedDiff(state.LinkGroup, model),
	}, nil
}

// BuildRenderState 输出叶子哈希、父层哈希和根验证状态。
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

// SyncSharedState 在共享树状态变化后重建 Merkle 树场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedTreeState(&model, sharedState, state.LinkGroup)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// treeModel 保存叶子、父层哈希、根哈希和验证结果。
type treeModel struct {
	Leaves     []leafValue `json:"leaves"`
	ParentHash []string    `json:"parent_hash"`
	RootHash   string      `json:"root_hash"`
	Verified   bool        `json:"verified"`
}

// leafValue 保存单个叶子节点的数据与哈希值。
type leafValue struct {
	ID   string `json:"id"`
	Data string `json:"data"`
	Hash string `json:"hash"`
}

// rebuildState 将 Merkle 树模型转换为渲染所需的节点和指标。
func rebuildState(state *framework.SceneState, model treeModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
		state.Nodes[index].Attributes = map[string]any{
			"hash": model.Leaves[index].Hash,
		}
		if phase == "叶子哈希" || phase == "路径验证" {
			state.Nodes[index].Status = "active"
		}
	}
	state.Messages = []framework.Message{
		{ID: "pair-left", Label: framework.Abbreviate(model.ParentHashAt(0), 12), Kind: "digest", Status: phase, SourceID: "leaf-1", TargetID: "leaf-2"},
		{ID: "pair-right", Label: framework.Abbreviate(model.ParentHashAt(1), 12), Kind: "digest", Status: phase, SourceID: "leaf-3", TargetID: "leaf-4"},
		{ID: "root", Label: framework.Abbreviate(model.RootHash, 12), Kind: "digest", Status: verificationStatus(model.Verified, phase), SourceID: "leaf-2", TargetID: "leaf-3"},
	}
	state.Metrics = []framework.Metric{
		{Key: "root", Label: "Merkle Root", Value: framework.Abbreviate(model.RootHash, 12), Tone: toneByVerification(model.Verified, phase)},
		{Key: "leaf_count", Label: "叶子数量", Value: fmt.Sprintf("%d", len(model.Leaves)), Tone: "info"},
		{Key: "parent_count", Label: "父节点数量", Value: fmt.Sprintf("%d", len(model.ParentHash)), Tone: "warning"},
		{Key: "verified", Label: "路径验证", Value: framework.BoolLabel(model.Verified, "通过", "失败"), Tone: toneByVerification(model.Verified, phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "当前阶段", Value: phase},
		{Label: "根哈希", Value: model.RootHash},
		{Label: "验证结果", Value: framework.BoolLabel(model.Verified, "通过", "失败")},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"merkle_tree": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 Merkle 树中叶子哈希、父节点合并、根输出和验证路径检查。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// ParentHashAt 返回指定位置的父节点哈希。
func (m treeModel) ParentHashAt(index int) string {
	if index < 0 || index >= len(m.ParentHash) {
		return ""
	}
	return m.ParentHash[index]
}

// decodeModel 从通用 JSON 状态恢复 Merkle 树模型。
func decodeModel(state *framework.SceneState) treeModel {
	entry, ok := state.Data["merkle_tree"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["merkle_tree"].(treeModel); ok {
			return typed
		}
		model := treeModel{
			Leaves: []leafValue{
				{ID: "leaf-1", Data: "tx-a"},
				{ID: "leaf-2", Data: "tx-b"},
				{ID: "leaf-3", Data: "tx-c"},
				{ID: "leaf-4", Data: "tx-d"},
			},
		}
		recalculateTree(&model)
		return model
	}
	model := treeModel{
		Leaves:     decodeLeaves(entry["leaves"]),
		ParentHash: framework.ToStringSlice(entry["parent_hash"]),
		RootHash:   framework.StringValue(entry["root_hash"], ""),
		Verified:   framework.BoolValue(entry["verified"], false),
	}
	if model.RootHash == "" {
		recalculateTree(&model)
	}
	return model
}

// recalculateTree 重新计算整棵树的叶子哈希、父层哈希和根哈希。
func recalculateTree(model *treeModel) {
	for index := range model.Leaves {
		model.Leaves[index].Hash = framework.HashText(model.Leaves[index].Data)
	}
	recalculateParents(model)
	recalculateRoot(model)
	model.Verified = verifyPath(*model)
}

// recalculateParents 重新计算父层哈希。
func recalculateParents(model *treeModel) {
	model.ParentHash = make([]string, 0, len(model.Leaves)/2)
	for index := 0; index < len(model.Leaves); index += 2 {
		right := index + 1
		if right >= len(model.Leaves) {
			right = index
		}
		model.ParentHash = append(model.ParentHash, framework.HashText(model.Leaves[index].Hash+model.Leaves[right].Hash))
	}
}

// recalculateRoot 基于父层哈希重新计算根哈希。
func recalculateRoot(model *treeModel) {
	if len(model.ParentHash) == 0 {
		model.RootHash = ""
		return
	}
	if len(model.ParentHash) == 1 {
		model.RootHash = model.ParentHash[0]
		return
	}
	model.RootHash = framework.HashText(model.ParentHash[0] + model.ParentHash[1])
}

// verifyPath 校验叶子到根的路径是否仍然一致。
func verifyPath(model treeModel) bool {
	if len(model.Leaves) < 2 || len(model.ParentHash) < 2 {
		return false
	}
	left := framework.HashText(model.Leaves[0].Hash + model.Leaves[1].Hash)
	right := framework.HashText(model.Leaves[2].Hash + model.Leaves[3].Hash)
	root := framework.HashText(left + right)
	return left == model.ParentHash[0] && right == model.ParentHash[1] && root == model.RootHash
}

// decodeLeaves 恢复叶子节点切片。
func decodeLeaves(value any) []leafValue {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]leafValue); ok {
			return append([]leafValue(nil), typed...)
		}
		return []leafValue{
			{ID: "leaf-1", Data: "tx-a"},
			{ID: "leaf-2", Data: "tx-b"},
			{ID: "leaf-3", Data: "tx-c"},
			{ID: "leaf-4", Data: "tx-d"},
		}
	}
	result := make([]leafValue, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, leafValue{
			ID:   framework.StringValue(entry["id"], ""),
			Data: framework.StringValue(entry["data"], ""),
			Hash: framework.StringValue(entry["hash"], ""),
		})
	}
	if len(result) == 0 {
		return []leafValue{
			{ID: "leaf-1", Data: "tx-a"},
			{ID: "leaf-2", Data: "tx-b"},
			{ID: "leaf-3", Data: "tx-c"},
			{ID: "leaf-4", Data: "tx-d"},
		}
	}
	return result
}

// nextPhase 返回 Merkle 树的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "叶子哈希":
		return "两两合并"
	case "两两合并":
		return "根节点输出"
	case "根节点输出":
		return "路径验证"
	default:
		return "叶子哈希"
	}
}

// phaseIndex 将阶段映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "叶子哈希":
		return 0
	case "两两合并":
		return 1
	case "根节点输出":
		return 2
	case "路径验证":
		return 3
	default:
		return 0
	}
}

// toneByVerification 根据验证结果选择色调。
func toneByVerification(verified bool, phase string) string {
	if phase == "路径验证" && verified {
		return "success"
	}
	if phase == "路径验证" && !verified {
		return "warning"
	}
	return "info"
}

// verificationStatus 返回消息链路的验证状态。
func verificationStatus(verified bool, phase string) string {
	if phase != "路径验证" {
		return phase
	}
	if verified {
		return "verified"
	}
	return "tampered"
}

// buildSharedDiff 按联动组输出 Merkle 场景允许共享的状态键。
func buildSharedDiff(linkGroup string, model treeModel) map[string]any {
	switch linkGroup {
	case "crypto-verify-group":
		return map[string]any{
			"hashes": map[string]any{
				"root":     model.RootHash,
				"verified": model.Verified,
				"leaves":   leafHashes(model),
			},
			"keys": map[string]any{
				"proof_target": firstLeafData(model),
			},
			"tree": map[string]any{
				"leaves":      model.Leaves,
				"parent_hash": model.ParentHash,
				"root_hash":   model.RootHash,
				"verified":    model.Verified,
			},
		}
	case "blockchain-integrity-group":
		return map[string]any{
			"transactions": transactionPayload(model),
			"merkle_root":  model.RootHash,
		}
	default:
		return nil
	}
}

// leafHashes 提取叶子哈希数组，供密码学验证组共享。
func leafHashes(model treeModel) []string {
	result := make([]string, 0, len(model.Leaves))
	for _, leaf := range model.Leaves {
		result = append(result, leaf.Hash)
	}
	return result
}

// firstLeafData 返回第一个叶子原文，用于和签名/密钥验证场景联动。
func firstLeafData(model treeModel) string {
	if len(model.Leaves) == 0 {
		return ""
	}
	return model.Leaves[0].Data
}

// transactionPayload 将叶子数据映射为区块链完整性组中的交易集合。
func transactionPayload(model treeModel) map[string]any {
	result := make(map[string]any, len(model.Leaves))
	for _, leaf := range model.Leaves {
		result[leaf.ID] = map[string]any{
			"payload": leaf.Data,
			"hash":    leaf.Hash,
		}
	}
	return result
}

// applySharedTreeState 将当前联动组的共享状态映射回 Merkle 树。
func applySharedTreeState(model *treeModel, sharedState map[string]any, linkGroup string) {
	if len(sharedState) == 0 {
		return
	}
	switch linkGroup {
	case "crypto-verify-group":
		if tree, ok := sharedState["tree"].(map[string]any); ok {
			model.Leaves = decodeLeaves(tree["leaves"])
			model.ParentHash = framework.ToStringSlice(tree["parent_hash"])
			model.RootHash = framework.StringValue(tree["root_hash"], model.RootHash)
			model.Verified = framework.BoolValue(tree["verified"], model.Verified)
		}
		if hashes, ok := sharedState["hashes"].(map[string]any); ok {
			model.Verified = framework.BoolValue(hashes["verified"], model.Verified)
			if sharedRoot := framework.StringValue(hashes["root"], ""); sharedRoot != "" {
				model.RootHash = sharedRoot
			}
		}
		if keys, ok := sharedState["keys"].(map[string]any); ok {
			if proofTarget := framework.StringValue(keys["proof_target"], ""); proofTarget != "" && len(model.Leaves) > 0 {
				model.Leaves[0].Data = proofTarget
			}
		}
	case "blockchain-integrity-group":
		if transactions, ok := sharedState["transactions"].(map[string]any); ok && len(transactions) > 0 {
			model.Leaves = leavesFromTransactions(transactions)
		}
		if sharedRoot := framework.StringValue(sharedState["merkle_root"], ""); sharedRoot != "" {
			model.RootHash = sharedRoot
		}
	}
}

// leavesFromTransactions 将完整性组的交易索引映射为叶子节点。
func leavesFromTransactions(transactions map[string]any) []leafValue {
	result := make([]leafValue, 0, len(transactions))
	for key, raw := range transactions {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, leafValue{
			ID:   key,
			Data: framework.StringValue(entry["payload"], framework.StringValue(entry["hash"], key)),
			Hash: framework.StringValue(entry["hash"], ""),
		})
	}
	if len(result) == 0 {
		return defaultLeaves()
	}
	for len(result) < 4 {
		index := len(result) + 1
		result = append(result, leafValue{
			ID:   fmt.Sprintf("leaf-%d", index),
			Data: fmt.Sprintf("padding-%d", index),
		})
	}
	return result[:4]
}

// defaultLeaves 返回 Merkle 树的默认四个叶子，确保渲染结构稳定。
func defaultLeaves() []leafValue {
	return []leafValue{
		{ID: "leaf-1", Data: "tx-a"},
		{ID: "leaf-2", Data: "tx-b"},
		{ID: "leaf-3", Data: "tx-c"},
		{ID: "leaf-4", Data: "tx-d"},
	}
}
