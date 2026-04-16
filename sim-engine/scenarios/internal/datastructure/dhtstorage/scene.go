package dhtstorage

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

const (
	// replicaCount 表示写入 DHT 时默认生成的副本数。
	replicaCount = 3
)

// DefaultState 构造分布式存储 DHT 场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "dht-storage",
		Title:        "分布式存储 DHT",
		Phase:        "键映射",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1200,
		TotalTicks:   14,
		Stages:       []string{"键映射", "副本分片", "路由查找"},
		Nodes: []framework.Node{
			{ID: "node-a", Label: "Node-1", Status: "active", Role: "storage", X: 180, Y: 70},
			{ID: "node-b", Label: "Node-2", Status: "normal", Role: "storage", X: 420, Y: 70},
			{ID: "node-c", Label: "Node-3", Status: "normal", Role: "storage", X: 570, Y: 220},
			{ID: "node-d", Label: "Node-4", Status: "normal", Role: "storage", X: 420, Y: 370},
			{ID: "node-e", Label: "Node-5", Status: "normal", Role: "storage", X: 180, Y: 370},
			{ID: "node-f", Label: "Node-6", Status: "normal", Role: "storage", X: 30, Y: 220},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 DHT 环空间、键哈希和路由目标。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := storageModel{
		RingOrder:    defaultRing(),
		ActiveKey:    "doc-1",
		KeyHash:      hashToSlot("doc-1"),
		PrimaryNode:  "node-b",
		ReplicaNodes: []string{"node-b", "node-c", "node-d"},
		RoutePath:    []string{"node-a", "node-b"},
		Redundancy:   replicaCount,
	}
	return rebuildState(state, model, "键映射")
}

// Step 推进键映射、副本分片和路由查找过程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "键映射"))
	switch phase {
	case "副本分片":
		model.ReplicaNodes = selectReplicas(model.RingOrder, model.PrimaryNode)
	case "路由查找":
		model.RoutePath = buildRoute(model.RingOrder, "node-a", model.PrimaryNode)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("DHT 进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology": map[string]any{
				"dht_ring":     encodeRing(model.RingOrder),
				"route_path":   append([]string(nil), model.RoutePath...),
				"primary_node": model.PrimaryNode,
			},
			"storage": map[string]any{
				model.ActiveKey: map[string]any{
					"hash_slot":  model.KeyHash,
					"replicas":   append([]string(nil), model.ReplicaNodes...),
					"redundancy": model.Redundancy,
				},
			},
		},
	}, nil
}

// HandleAction 写入新键并重新计算主节点、副本和路由。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.ActiveKey = framework.StringValue(input.Params["key"], "doc-1")
	model.KeyHash = hashToSlot(model.ActiveKey)
	model.PrimaryNode = locatePrimary(model.RingOrder, model.KeyHash)
	model.ReplicaNodes = selectReplicas(model.RingOrder, model.PrimaryNode)
	model.RoutePath = buildRoute(model.RingOrder, "node-a", model.PrimaryNode)
	if err := rebuildState(state, model, "键映射"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "写入键值", fmt.Sprintf("已将键 %s 映射到 DHT 环。", model.ActiveKey), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"storage": map[string]any{
				model.ActiveKey: map[string]any{
					"hash_slot": model.KeyHash,
					"primary":   model.PrimaryNode,
					"replicas":  append([]string(nil), model.ReplicaNodes...),
				},
			},
		},
	}, nil
}

// BuildRenderState 输出 DHT 环分布、路由路径和副本布局。
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

// SyncSharedState 在网络基础组共享拓扑与存储状态变化后重建 DHT 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedDHTState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// ringNode 表示 DHT 环上的单个逻辑节点。
type ringNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Position int    `json:"position"`
}

// storageModel 保存键映射、副本分布和查找路径。
type storageModel struct {
	RingOrder    []ringNode `json:"ring_order"`
	ActiveKey    string     `json:"active_key"`
	KeyHash      int        `json:"key_hash"`
	PrimaryNode  string     `json:"primary_node"`
	ReplicaNodes []string   `json:"replica_nodes"`
	RoutePath    []string   `json:"route_path"`
	Redundancy   int        `json:"redundancy"`
}

// rebuildState 将 DHT 模型映射为环节点、数据流与内嵌指标。
func rebuildState(state *framework.SceneState, model storageModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	replicaSet := make(map[string]struct{}, len(model.ReplicaNodes))
	for _, replica := range model.ReplicaNodes {
		replicaSet[replica] = struct{}{}
	}
	for index := range state.Nodes {
		node := &state.Nodes[index]
		node.Status = "normal"
		node.Load = 0
		node.Attributes = map[string]any{
			"slot": ringPosition(model.RingOrder, node.ID),
		}
		if node.ID == model.PrimaryNode {
			node.Status = "active"
			node.Load = float64(model.KeyHash)
		}
		if _, ok := replicaSet[node.ID]; ok {
			node.Status = "warning"
			node.Load += 18
		}
		if contains(model.RoutePath, node.ID) {
			node.Status = "success"
			node.Load += 24
		}
	}
	state.Messages = buildMessages(model, phase)
	state.Metrics = []framework.Metric{
		{Key: "key", Label: "当前键", Value: model.ActiveKey, Tone: "info"},
		{Key: "slot", Label: "哈希槽位", Value: fmt.Sprintf("%d/255", model.KeyHash), Tone: "warning"},
		{Key: "primary", Label: "主节点", Value: model.PrimaryNode, Tone: "success"},
		{Key: "replicas", Label: "副本数", Value: fmt.Sprintf("%d", len(model.ReplicaNodes)), Tone: "info"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "副本节点", Value: strings.Join(model.ReplicaNodes, ", ")},
		{Label: "路由路径", Value: strings.Join(model.RoutePath, " -> ")},
		{Label: "冗余级别", Value: fmt.Sprintf("%d", model.Redundancy)},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"dht_storage": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟键在 DHT 环上的哈希映射、副本放置和逐跳路由查找。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 DHT 模型。
func decodeModel(state *framework.SceneState) storageModel {
	entry, ok := state.Data["dht_storage"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["dht_storage"].(storageModel); ok {
			return typed
		}
		return storageModel{
			RingOrder:    defaultRing(),
			ActiveKey:    "doc-1",
			KeyHash:      hashToSlot("doc-1"),
			PrimaryNode:  "node-b",
			ReplicaNodes: []string{"node-b", "node-c", "node-d"},
			RoutePath:    []string{"node-a", "node-b"},
			Redundancy:   replicaCount,
		}
	}
	return storageModel{
		RingOrder:    decodeRing(entry["ring_order"]),
		ActiveKey:    framework.StringValue(entry["active_key"], "doc-1"),
		KeyHash:      int(framework.NumberValue(entry["key_hash"], float64(hashToSlot("doc-1")))),
		PrimaryNode:  framework.StringValue(entry["primary_node"], "node-b"),
		ReplicaNodes: framework.ToStringSlice(entry["replica_nodes"]),
		RoutePath:    framework.ToStringSlice(entry["route_path"]),
		Redundancy:   int(framework.NumberValue(entry["redundancy"], replicaCount)),
	}
}

// applySharedDHTState 将网络基础组共享拓扑与存储布局映射回 DHT 模型。
func applySharedDHTState(model *storageModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if topology, ok := sharedState["topology"].(map[string]any); ok {
		if routePath := framework.ToStringSlice(topology["route_path"]); len(routePath) > 0 {
			model.RoutePath = routePath
		}
		if primaryNode, ok := topology["primary_node"].(string); ok && strings.TrimSpace(primaryNode) != "" {
			model.PrimaryNode = primaryNode
		}
	}
	if storage, ok := sharedState["storage"].(map[string]any); ok {
		if keyState, ok := storage[model.ActiveKey].(map[string]any); ok {
			if hashSlot, ok := keyState["hash_slot"]; ok {
				model.KeyHash = int(framework.NumberValue(hashSlot, float64(model.KeyHash)))
			}
			if replicas := framework.ToStringSlice(keyState["replicas"]); len(replicas) > 0 {
				model.ReplicaNodes = replicas
			}
		}
	}
}

// defaultRing 返回默认 DHT 环顺序。
func defaultRing() []ringNode {
	return []ringNode{
		{ID: "node-a", Label: "Node-1", Position: 28},
		{ID: "node-b", Label: "Node-2", Position: 87},
		{ID: "node-c", Label: "Node-3", Position: 126},
		{ID: "node-d", Label: "Node-4", Position: 173},
		{ID: "node-e", Label: "Node-5", Position: 214},
		{ID: "node-f", Label: "Node-6", Position: 242},
	}
}

// hashToSlot 将键哈希到 0-255 的环空间。
func hashToSlot(key string) int {
	sum := sha1.Sum([]byte(key))
	return int(sum[0])
}

// locatePrimary 在环上查找负责当前槽位的主节点。
func locatePrimary(ring []ringNode, slot int) string {
	sorted := append([]ringNode(nil), ring...)
	sort.Slice(sorted, func(i int, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})
	for _, node := range sorted {
		if slot <= node.Position {
			return node.ID
		}
	}
	return sorted[0].ID
}

// selectReplicas 选择主节点开始的连续副本集合。
func selectReplicas(ring []ringNode, primary string) []string {
	index := ringIndex(ring, primary)
	if index < 0 {
		return []string{primary}
	}
	result := make([]string, 0, replicaCount)
	for offset := 0; offset < replicaCount; offset++ {
		result = append(result, ring[(index+offset)%len(ring)].ID)
	}
	return result
}

// buildRoute 构造从起点节点到目标节点的逐跳路径。
func buildRoute(ring []ringNode, from string, to string) []string {
	start := ringIndex(ring, from)
	target := ringIndex(ring, to)
	if start < 0 || target < 0 {
		return []string{from, to}
	}
	result := []string{ring[start].ID}
	index := start
	for index != target {
		index = (index + 1) % len(ring)
		result = append(result, ring[index].ID)
	}
	return result
}

// buildMessages 生成键映射、副本复制和路由查找消息。
func buildMessages(model storageModel, phase string) []framework.Message {
	messages := []framework.Message{
		{ID: "hash-map", Label: fmt.Sprintf("%s@%d", model.ActiveKey, model.KeyHash), Kind: "pointer", Status: phase, SourceID: "node-a", TargetID: model.PrimaryNode},
	}
	for index, replica := range model.ReplicaNodes {
		messages = append(messages, framework.Message{
			ID:       fmt.Sprintf("replica-%d", index),
			Label:    fmt.Sprintf("Replica-%d", index+1),
			Kind:     "pointer",
			Status:   phase,
			SourceID: model.PrimaryNode,
			TargetID: replica,
			Attributes: map[string]any{
				"replica": true,
			},
		})
	}
	if phase == "路由查找" {
		for index := 0; index < len(model.RoutePath)-1; index++ {
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("route-%d", index),
				Label:    fmt.Sprintf("Hop-%d", index+1),
				Kind:     "pointer",
				Status:   "route",
				SourceID: model.RoutePath[index],
				TargetID: model.RoutePath[index+1],
			})
		}
	}
	return messages
}

// encodeRing 将环顺序转为前端友好的映射。
func encodeRing(ring []ringNode) []map[string]any {
	result := make([]map[string]any, 0, len(ring))
	for _, node := range ring {
		result = append(result, map[string]any{
			"id":       node.ID,
			"label":    node.Label,
			"position": node.Position,
		})
	}
	return result
}

// decodeRing 从通用 JSON 列表恢复环节点顺序。
func decodeRing(value any) []ringNode {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]ringNode); ok {
			return append([]ringNode(nil), typed...)
		}
		return defaultRing()
	}
	result := make([]ringNode, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, ringNode{
			ID:       framework.StringValue(entry["id"], ""),
			Label:    framework.StringValue(entry["label"], ""),
			Position: int(framework.NumberValue(entry["position"], 0)),
		})
	}
	if len(result) == 0 {
		return defaultRing()
	}
	return result
}

// nextPhase 返回 DHT 场景的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "键映射":
		return "副本分片"
	case "副本分片":
		return "路由查找"
	default:
		return "键映射"
	}
}

// phaseIndex 将阶段映射到时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "键映射":
		return 0
	case "副本分片":
		return 1
	case "路由查找":
		return 2
	default:
		return 0
	}
}

// toneByPhase 返回阶段对应的色调。
func toneByPhase(phase string) string {
	if phase == "路由查找" {
		return "success"
	}
	if phase == "副本分片" {
		return "warning"
	}
	return "info"
}

// ringIndex 返回指定节点在环顺序中的下标。
func ringIndex(ring []ringNode, nodeID string) int {
	for index, node := range ring {
		if node.ID == nodeID {
			return index
		}
	}
	return -1
}

// ringPosition 返回节点在环上的槽位。
func ringPosition(ring []ringNode, nodeID string) int {
	for _, node := range ring {
		if node.ID == nodeID {
			return node.Position
		}
	}
	return -1
}

// contains 判断字符串切片中是否包含指定值。
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

