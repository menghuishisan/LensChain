package p2pdiscovery

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 P2P 网络发现与路由场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "p2p-discovery",
		Title:        "P2P 网络发现与路由",
		Phase:        "节点发现",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   18,
		Stages:       []string{"节点发现", "邻居交换", "路由收敛"},
		Nodes: []framework.Node{
			{ID: "peer-a", Label: "Peer-A", Status: "active", Role: "peer", X: 160, Y: 160},
			{ID: "peer-b", Label: "Peer-B", Status: "normal", Role: "peer", X: 330, Y: 120},
			{ID: "peer-c", Label: "Peer-C", Status: "normal", Role: "peer", X: 500, Y: 160},
			{ID: "peer-d", Label: "Peer-D", Status: "normal", Role: "peer", X: 250, Y: 320},
			{ID: "peer-e", Label: "Peer-E", Status: "normal", Role: "peer", X: 430, Y: 320},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 Kademlia 风格的节点标识和路由表。
func Init(state *framework.SceneState, input framework.InitInput) error {
	peers := make([]peerState, 0, len(state.Nodes))
	for _, node := range state.Nodes {
		peers = append(peers, newPeer(node.ID, node.Label))
	}
	peers = applySharedDiscoveryState(peers, input.SharedState)
	return rebuildState(state, convergeRoutes(peers, 2), "节点发现")
}

// Step 推进节点发现、邻居交换和路由收敛过程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	peers := decodePeers(state)
	peers = applySharedDiscoveryState(peers, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "节点发现"))
	switch phase {
	case "节点发现":
		peers = convergeRoutes(peers, 2)
	case "邻居交换":
		peers = convergeRoutes(peers, 3)
	case "路由收敛":
		peers = convergeRoutes(peers, 4)
	}
	if err := rebuildState(state, peers, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("P2P 网络进入%s阶段，路由表按 XOR 距离更新。", phase), "info")
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology":      state.Data["peers"],
			"routing_table": state.Data["routing_tables"],
		},
	}, nil
}

// HandleAction 支持新增节点和手动扰动路由两类交互。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	peers := decodePeers(state)
	phase := framework.StringValue(state.Data["phase_name"], "节点发现")
	switch input.ActionCode {
	case "add_peer":
		label := framework.StringValue(input.Params["peer_label"], fmt.Sprintf("Peer-%d", len(peers)+1))
		peers = append(peers, newPeer(strings.ToLower(strings.ReplaceAll(label, " ", "-")), label))
		phase = "节点发现"
	case "shuffle_route":
		phase = "邻居交换"
	}
	peers = convergeRoutes(peers, 3)
	if err := rebuildState(state, peers, phase); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "更新路由", "路由表已按节点距离重新收敛。", "success")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"topology":      state.Data["peers"],
			"routing_table": state.Data["routing_tables"],
		},
	}, nil
}

// BuildRenderState 返回拓扑、路由表和距离信息。
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

// SyncSharedState 在网络基础组共享状态变化后重建 P2P 发现场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	peers := applySharedDiscoveryState(decodePeers(state), sharedState)
	return rebuildState(state, peers, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// peerState 保存 P2P 节点的短 ID、邻居和 Kademlia bucket 信息。
type peerState struct {
	ID           string        `json:"id"`
	Label        string        `json:"label"`
	NodeID       uint64        `json:"node_id"`
	NodeIDHex    string        `json:"node_id_hex"`
	Neighbors    []string      `json:"neighbors"`
	RouteEntries []routeEntry  `json:"route_entries"`
	Buckets      []bucketState `json:"buckets"`
}

// routeEntry 表示一个路由表条目及其 XOR 距离。
type routeEntry struct {
	PeerID      string `json:"peer_id"`
	Distance    uint64 `json:"distance"`
	DistanceHex string `json:"distance_hex"`
	Bucket      int    `json:"bucket"`
}

// bucketState 表示一个 Kademlia 距离桶中的候选节点。
type bucketState struct {
	Index int      `json:"index"`
	Peers []string `json:"peers"`
}

// newPeer 根据标签生成稳定节点 ID。
func newPeer(id string, label string) peerState {
	hash := sha1.Sum([]byte(label))
	nodeID := binary.BigEndian.Uint64(hash[:8])
	return peerState{
		ID:        id,
		Label:     label,
		NodeID:    nodeID,
		NodeIDHex: hex.EncodeToString(hash[:8]),
	}
}

// convergeRoutes 按 XOR 距离为每个节点选择最近邻并重建 bucket。
func convergeRoutes(peers []peerState, limit int) []peerState {
	for index := range peers {
		entries := make([]routeEntry, 0, len(peers)-1)
		for _, other := range peers {
			if other.ID == peers[index].ID {
				continue
			}
			distance := peers[index].NodeID ^ other.NodeID
			entries = append(entries, routeEntry{
				PeerID:      other.ID,
				Distance:    distance,
				DistanceHex: fmt.Sprintf("%016x", distance),
				Bucket:      bucketIndex(distance),
			})
		}
		sort.Slice(entries, func(left int, right int) bool {
			return entries[left].Distance < entries[right].Distance
		})
		selectedLimit := limit
		if selectedLimit > len(entries) {
			selectedLimit = len(entries)
		}
		selected := entries[:selectedLimit]
		neighbors := make([]string, 0, len(selected))
		for _, entry := range selected {
			neighbors = append(neighbors, entry.PeerID)
		}
		peers[index].Neighbors = neighbors
		peers[index].RouteEntries = selected
		peers[index].Buckets = buildBuckets(selected)
	}
	return peers
}

// rebuildState 将路由表收敛结果转换为节点拓扑和消息流。
func rebuildState(state *framework.SceneState, peers []peerState, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	state.Nodes = make([]framework.Node, 0, len(peers))
	for index, peer := range peers {
		status := "normal"
		if index == 0 {
			status = "active"
		}
		state.Nodes = append(state.Nodes, framework.Node{
			ID:     peer.ID,
			Label:  peer.Label,
			Status: status,
			Role:   "peer",
			X:      160 + float64((index%3)*180),
			Y:      150 + float64((index/3)*150),
			Load:   float64(len(peer.Neighbors) * 20),
			Attributes: map[string]any{
				"node_id":       peer.NodeIDHex,
				"neighbors":     peer.Neighbors,
				"route_entries": peer.RouteEntries,
				"buckets":       peer.Buckets,
			},
		})
	}
	state.Messages = buildMessages(peers, phase)
	state.Metrics = []framework.Metric{
		{Key: "peers", Label: "节点数", Value: fmt.Sprintf("%d", len(peers)), Tone: "info"},
		{Key: "routes", Label: "路由条目", Value: fmt.Sprintf("%d", routeCount(peers)), Tone: "success"},
		{Key: "coverage", Label: "邻居覆盖率", Value: fmt.Sprintf("%.1f%%", coverage(peers)*100), Tone: "warning"},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "路由规则", Value: "Kademlia XOR 距离最近邻"},
	}
	state.Data = map[string]any{
		"phase_name":     phase,
		"peers":          peers,
		"routing_tables": routingTables(peers),
	}
	state.Extra = map[string]any{
		"description": "该场景使用稳定节点 ID、XOR 距离和 bucket 路由表实现 P2P 发现过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodePeers 从 JSON 通用状态恢复 P2P 节点集合。
func decodePeers(state *framework.SceneState) []peerState {
	raw, ok := state.Data["peers"].([]any)
	if !ok {
		if typed, ok := state.Data["peers"].([]peerState); ok {
			return typed
		}
		peers := make([]peerState, 0, len(state.Nodes))
		for _, node := range state.Nodes {
			peers = append(peers, newPeer(node.ID, node.Label))
		}
		return convergeRoutes(peers, 2)
	}
	peers := make([]peerState, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		nodeID := uint64(framework.NumberValue(entry["node_id"], 0))
		peers = append(peers, peerState{
			ID:        framework.StringValue(entry["id"], ""),
			Label:     framework.StringValue(entry["label"], ""),
			NodeID:    nodeID,
			NodeIDHex: framework.StringValue(entry["node_id_hex"], fmt.Sprintf("%016x", nodeID)),
			Neighbors: framework.ToStringSlice(entry["neighbors"]),
		})
	}
	return convergeRoutes(peers, 3)
}

// buildMessages 根据邻居关系生成路由交换消息。
func buildMessages(peers []peerState, phase string) []framework.Message {
	messages := make([]framework.Message, 0)
	for _, peer := range peers {
		for _, neighbor := range peer.Neighbors {
			messages = append(messages, framework.Message{
				ID:       fmt.Sprintf("%s-%s-route", peer.ID, neighbor),
				Label:    "FIND_NODE",
				Kind:     "packet",
				Status:   phase,
				SourceID: peer.ID,
				TargetID: neighbor,
			})
		}
	}
	return messages
}

// buildBuckets 将路由条目按距离桶聚合。
func buildBuckets(entries []routeEntry) []bucketState {
	index := map[int][]string{}
	for _, entry := range entries {
		index[entry.Bucket] = append(index[entry.Bucket], entry.PeerID)
	}
	keys := make([]int, 0, len(index))
	for key := range index {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	result := make([]bucketState, 0, len(keys))
	for _, key := range keys {
		result = append(result, bucketState{Index: key, Peers: index[key]})
	}
	return result
}

// bucketIndex 计算 XOR 距离所在的 Kademlia bucket 编号。
func bucketIndex(distance uint64) int {
	if distance == 0 {
		return 0
	}
	return bits.Len64(distance) - 1
}

// routingTables 构造联动共享状态中的路由表映射。
func routingTables(peers []peerState) map[string]any {
	result := map[string]any{}
	for _, peer := range peers {
		result[peer.ID] = peer.RouteEntries
	}
	return result
}

// routeCount 统计全网路由条目数量。
func routeCount(peers []peerState) int {
	total := 0
	for _, peer := range peers {
		total += len(peer.RouteEntries)
	}
	return total
}

// coverage 计算当前节点对全网候选邻居的覆盖比例。
func coverage(peers []peerState) float64 {
	if len(peers) <= 1 {
		return 1
	}
	possible := len(peers) * (len(peers) - 1)
	return float64(routeCount(peers)) / float64(possible)
}

// nextPhase 返回 P2P 发现过程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "节点发现":
		return "邻居交换"
	case "邻居交换":
		return "路由收敛"
	default:
		return "路由收敛"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "节点发现":
		return 0
	case "邻居交换":
		return 1
	case "路由收敛":
		return 2
	default:
		return 0
	}
}

// applySharedDiscoveryState 将网络基础组中的拓扑与路由共享状态映射回节点发现场景。
func applySharedDiscoveryState(peers []peerState, sharedState map[string]any) []peerState {
	if len(sharedState) == 0 {
		return peers
	}
	if topology, ok := sharedState["topology"].([]any); ok && len(topology) > len(peers) {
		for index := len(peers); index < len(topology); index++ {
			peers = append(peers, newPeer(fmt.Sprintf("peer-linked-%d", index+1), fmt.Sprintf("Peer-Linked-%d", index+1)))
		}
	}
	if routes, ok := sharedState["routing_table"].(map[string]any); ok {
		for index := range peers {
			if entries, ok := routes[peers[index].ID].([]any); ok && len(entries) > 0 {
				neighbors := make([]string, 0, len(entries))
				for _, item := range entries {
					if entry, ok := item.(map[string]any); ok {
						neighborID := framework.StringValue(entry["peer_id"], "")
						if neighborID != "" {
							neighbors = append(neighbors, neighborID)
						}
					}
				}
				if len(neighbors) > 0 {
					peers[index].Neighbors = neighbors
				}
			}
		}
	}
	return peers
}
