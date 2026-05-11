// 模块：sim-engine/scenarios/internal/nodenetwork/p2pdiscovery
// 文件职责：NET-01 P2P 节点发现（Kademlia DHT）场景的完整实现。
//
// SSOT 依据：06.md §4.1.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现 Kademlia 节点发现协议（Petar Maymounkov & David Mazières 2002）；
// 零第三方网络库；复用 sha256hash.Sum256 派生节点 ID（160-bit 简化为 bits 位）：
//
//   · 节点 ID = SHA-256(seed_str)[:idBytes]，截断到指定 bit 长度（默认 32-bit 教学版）
//   · XOR 距离：d(a, b) = a XOR b（按字节异或，前导零越多越"近"）
//   · K-buckets：每个节点维护 (idBits) 个桶，bucket[i] 保存与本节点 XOR 距离前导零位 = i
//                的已知节点（每桶上限 k，默认 k=4）；新节点入桶时 LRU 替换或满则丢弃
//   · FIND_NODE 协议：查询 target → 从本地 closest α 个节点开始 → 它们返回各自最近 α 个 →
//                合并去重、保留最 closest k 个 → 重复直到没有更近节点
//   · PING / PONG：节点存活探测（教学：直接读 down 标志判定）
//
// 教学决策：
//   - graph_layout 力导向布局展示节点拓扑
//   - heat_map 展示某节点的 k-buckets 占用（行 = bucket index，列 = bucket 内位置）
//   - 当前查询路径用 verify_path_highlight 高亮
//   - 节点 status：normal / active=查询中 / error=down

package p2pdiscovery

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "p2p-discovery"
	schemaVersion = "v1.0.0"
	algorithmType = "kademlia"

	defaultIDBits    = 32 // 教学版 ID 长度（实际 Kad 是 160 bit）
	defaultK         = 4  // bucket 容量
	defaultAlpha     = 3  // 并发查询度
	defaultNodeCount = 8
	maxNodeCount     = 16
	maxLookupHistory = 16

	linkGroupNetworkBase = "network-base-group"
	linkOwnerSubtree     = "network.p2p"
)

// =====================================================================
// 节点 ID 与 XOR 距离
// =====================================================================

// nodeID 紧凑节点 ID（最多 idBytes 字节，对应 idBits 位）。
type nodeID []byte

// makeID 把字符串种子派生为 idBits 位 ID（教学：取 SHA-256 前 idBytes 字节）。
func makeID(seed string, idBits int) nodeID {
	h := sha256hash.Sum256([]byte(seed))
	idBytes := (idBits + 7) / 8
	return append(nodeID{}, h[:idBytes]...)
}

// xorDistance 返回 a XOR b 的字节切片。
func xorDistance(a, b nodeID) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// commonPrefixLen 计算 XOR 结果的前导零位数（即 a 与 b 的共同前缀长度，越大 = 越近）。
func commonPrefixLen(d []byte) int {
	for i, b := range d {
		if b == 0 {
			continue
		}
		return i*8 + bits.LeadingZeros8(b)
	}
	return len(d) * 8
}

// distLess 比较两个 XOR 距离 a < b（按字节序大端）。
func distLess(a, b []byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// =====================================================================
// 节点 / k-buckets
// =====================================================================

type peer struct {
	ID nodeID
}

type kademliaNode struct {
	Label   string // 教学用 ID（"n0", "n1", ...）
	ID      nodeID
	IsDown  bool
	Buckets [][]peer // buckets[i] 存放 commonPrefix == i 的 peer
}

func newKademliaNode(label string, id nodeID, idBits int) kademliaNode {
	return kademliaNode{
		Label:   label,
		ID:      id,
		Buckets: make([][]peer, idBits),
	}
}

// addPeer 把新发现的 peer 加入对应 bucket（去重 + 容量上限）。
func (n *kademliaNode) addPeer(other peer, k int) {
	if string(other.ID) == string(n.ID) {
		return
	}
	cpl := commonPrefixLen(xorDistance(n.ID, other.ID))
	if cpl >= len(n.Buckets) {
		cpl = len(n.Buckets) - 1
	}
	bucket := n.Buckets[cpl]
	for _, p := range bucket {
		if string(p.ID) == string(other.ID) {
			return // 已存在
		}
	}
	if len(bucket) >= k {
		// 简化 LRU：丢弃新节点（生产环境应 PING 最旧节点决定）
		return
	}
	n.Buckets[cpl] = append(bucket, other)
}

// closestPeers 返回本地已知节点中，与 target 最近的 m 个。
func (n kademliaNode) closestPeers(target nodeID, m int) []peer {
	all := []peer{}
	for _, b := range n.Buckets {
		all = append(all, b...)
	}
	sort.Slice(all, func(i, j int) bool {
		return distLess(xorDistance(all[i].ID, target), xorDistance(all[j].ID, target))
	})
	if len(all) > m {
		all = all[:m]
	}
	return all
}

// totalKnownPeers 已知 peer 总数。
func (n kademliaNode) totalKnownPeers() int {
	total := 0
	for _, b := range n.Buckets {
		total += len(b)
	}
	return total
}

// =====================================================================
// 网络（多个节点 + 全局拓扑）
// =====================================================================

type lookupStep struct {
	Round    int
	From     string // 发起查询的节点 label
	Queried  []string
	Returned []string // 返回的更近节点 label
}

type lookupResult struct {
	Initiator  string
	TargetID   string
	Steps      []lookupStep
	FoundPeer  string // 找到的最近节点 label
	HopCount   int
	Successful bool
}

type networkState struct {
	IDBits      int
	K           int
	Alpha       int
	Nodes       []kademliaNode
	LastLookup  *lookupResult
	BootstrapID string // bootstrap 节点 label
	LastError   string
}

func defaultNetwork() networkState {
	ns := networkState{
		IDBits: defaultIDBits,
		K:      defaultK,
		Alpha:  defaultAlpha,
	}
	for i := 0; i < defaultNodeCount; i++ {
		label := fmt.Sprintf("n%d", i)
		ns.Nodes = append(ns.Nodes, newKademliaNode(label, makeID(label+"-seed", ns.IDBits), ns.IDBits))
	}
	ns.BootstrapID = "n0"
	// 全连通初始（每节点都知道 bootstrap）
	for i := range ns.Nodes {
		if ns.Nodes[i].Label == ns.BootstrapID {
			continue
		}
		var bn *kademliaNode
		for j := range ns.Nodes {
			if ns.Nodes[j].Label == ns.BootstrapID {
				bn = &ns.Nodes[j]
				break
			}
		}
		if bn != nil {
			ns.Nodes[i].addPeer(peer{ID: bn.ID}, ns.K)
			bn.addPeer(peer{ID: ns.Nodes[i].ID}, ns.K)
		}
	}
	return ns
}

// findNodeByLabel 按 label 查找节点指针。
func (ns *networkState) findNodeByLabel(label string) *kademliaNode {
	for i := range ns.Nodes {
		if ns.Nodes[i].Label == label {
			return &ns.Nodes[i]
		}
	}
	return nil
}

// labelByID 按 ID 字节查找节点 label。
func (ns networkState) labelByID(id nodeID) string {
	for _, n := range ns.Nodes {
		if string(n.ID) == string(id) {
			return n.Label
		}
	}
	return "?"
}

// runLookup 在 initiator 节点上执行 FIND_NODE(target)：
//   - 选 α 个最近未查询过的本地 peer 作为待查询集
//   - 它们各自返回 closest α 个；合并入 short-list
//   - 直到没有更近节点出现，停止；返回 short-list 头 = 找到的节点
func (ns *networkState) runLookup(initiatorLabel string, targetID nodeID, maxRounds int) lookupResult {
	res := lookupResult{
		Initiator: initiatorLabel,
		TargetID:  hex.EncodeToString(targetID),
	}
	initiator := ns.findNodeByLabel(initiatorLabel)
	if initiator == nil || initiator.IsDown {
		res.Successful = false
		return res
	}
	visited := map[string]bool{}
	visited[string(initiator.ID)] = true

	// short-list：当前认为最近的节点列表
	shortList := initiator.closestPeers(targetID, ns.K)

	for round := 0; round < maxRounds; round++ {
		// 取前 α 个未查询节点
		toQuery := []peer{}
		for _, p := range shortList {
			if visited[string(p.ID)] {
				continue
			}
			toQuery = append(toQuery, p)
			if len(toQuery) >= ns.Alpha {
				break
			}
		}
		if len(toQuery) == 0 {
			break
		}
		step := lookupStep{Round: round, From: initiatorLabel}
		newDiscovered := []peer{}
		for _, p := range toQuery {
			step.Queried = append(step.Queried, ns.labelByID(p.ID))
			visited[string(p.ID)] = true
			// 找到 p 节点的本地视图，请它返回 closest α
			var responder *kademliaNode
			for i := range ns.Nodes {
				if string(ns.Nodes[i].ID) == string(p.ID) && !ns.Nodes[i].IsDown {
					responder = &ns.Nodes[i]
					break
				}
			}
			if responder == nil {
				continue
			}
			returned := responder.closestPeers(targetID, ns.Alpha)
			for _, r := range returned {
				step.Returned = append(step.Returned, ns.labelByID(r.ID))
				newDiscovered = append(newDiscovered, r)
				// 顺便把 responder 与 returned 都加入 initiator 的 buckets（教学：信息传播）
				initiator.addPeer(r, ns.K)
			}
			initiator.addPeer(p, ns.K)
		}
		res.Steps = append(res.Steps, step)
		// 合并 newDiscovered 到 shortList，按距离排序，截断到 K
		merged := append([]peer{}, shortList...)
		merged = append(merged, newDiscovered...)
		// 去重
		seen := map[string]bool{}
		dedup := []peer{}
		for _, p := range merged {
			if seen[string(p.ID)] {
				continue
			}
			seen[string(p.ID)] = true
			dedup = append(dedup, p)
		}
		sort.Slice(dedup, func(i, j int) bool {
			return distLess(xorDistance(dedup[i].ID, targetID), xorDistance(dedup[j].ID, targetID))
		})
		if len(dedup) > ns.K {
			dedup = dedup[:ns.K]
		}
		// 检查是否还有更近的；如果新的 head 和上轮 head 一样且都已 visited，停止
		if len(dedup) > 0 && len(shortList) > 0 &&
			string(dedup[0].ID) == string(shortList[0].ID) &&
			visited[string(dedup[0].ID)] {
			shortList = dedup
			break
		}
		shortList = dedup
	}
	res.HopCount = len(res.Steps)
	if len(shortList) > 0 {
		res.FoundPeer = ns.labelByID(shortList[0].ID)
		res.Successful = true
	}
	return res
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) networkState {
	if s == nil || s.Data == nil {
		return defaultNetwork()
	}
	d := s.Data
	ns := networkState{
		IDBits:      fw.MapInt(d, "id_bits", defaultIDBits),
		K:           fw.MapInt(d, "k", defaultK),
		Alpha:       fw.MapInt(d, "alpha", defaultAlpha),
		BootstrapID: fw.MapStr(d, "bootstrap_id", "n0"),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				idHex := fw.MapStr(nm, "id", "")
				idBytes, _ := hex.DecodeString(idHex)
				node := newKademliaNode(fw.MapStr(nm, "label", ""), idBytes, ns.IDBits)
				node.IsDown = fw.MapBool(nm, "down", false)
				if bsAny, ok := nm["buckets"].([]any); ok {
					for i, bAny := range bsAny {
						if i >= len(node.Buckets) {
							break
						}
						if peersAny, ok := bAny.([]any); ok {
							for _, pAny := range peersAny {
								if pHex, ok := pAny.(string); ok {
									if pBytes, err := hex.DecodeString(pHex); err == nil {
										node.Buckets[i] = append(node.Buckets[i], peer{ID: pBytes})
									}
								}
							}
						}
					}
				}
				ns.Nodes = append(ns.Nodes, node)
			}
		}
	}
	if len(ns.Nodes) == 0 {
		return defaultNetwork()
	}
	if luAny, ok := d["last_lookup"].(map[string]any); ok {
		ns.LastLookup = &lookupResult{
			Initiator:  fw.MapStr(luAny, "initiator", ""),
			TargetID:   fw.MapStr(luAny, "target_id", ""),
			FoundPeer:  fw.MapStr(luAny, "found_peer", ""),
			HopCount:   fw.MapInt(luAny, "hop_count", 0),
			Successful: fw.MapBool(luAny, "successful", false),
		}
		if stepsAny, ok := luAny["steps"].([]any); ok {
			for _, sAny := range stepsAny {
				if sm, ok := sAny.(map[string]any); ok {
					st := lookupStep{
						Round: fw.MapInt(sm, "round", 0),
						From:  fw.MapStr(sm, "from", ""),
					}
					if qAny, ok := sm["queried"].([]any); ok {
						for _, q := range qAny {
							if s, ok := q.(string); ok {
								st.Queried = append(st.Queried, s)
							}
						}
					}
					if rAny, ok := sm["returned"].([]any); ok {
						for _, r := range rAny {
							if s, ok := r.(string); ok {
								st.Returned = append(st.Returned, s)
							}
						}
					}
					ns.LastLookup.Steps = append(ns.LastLookup.Steps, st)
				}
			}
		}
	}
	return ns
}

func saveState(s *fw.SceneState, ns networkState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["id_bits"] = ns.IDBits
	s.Data["k"] = ns.K
	s.Data["alpha"] = ns.Alpha
	s.Data["bootstrap_id"] = ns.BootstrapID
	s.Data["last_error"] = ns.LastError
	nodesAny := make([]any, len(ns.Nodes))
	for i, n := range ns.Nodes {
		buckets := make([]any, len(n.Buckets))
		for bi, b := range n.Buckets {
			peersAny := make([]any, len(b))
			for pi, p := range b {
				peersAny[pi] = hex.EncodeToString(p.ID)
			}
			buckets[bi] = peersAny
		}
		nodesAny[i] = map[string]any{
			"label":   n.Label,
			"id":      hex.EncodeToString(n.ID),
			"down":    n.IsDown,
			"buckets": buckets,
		}
	}
	s.Data["nodes"] = nodesAny
	if ns.LastLookup != nil {
		stepsAny := make([]any, len(ns.LastLookup.Steps))
		for i, st := range ns.LastLookup.Steps {
			qAny := make([]any, len(st.Queried))
			for j, q := range st.Queried {
				qAny[j] = q
			}
			rAny := make([]any, len(st.Returned))
			for j, r := range st.Returned {
				rAny[j] = r
			}
			stepsAny[i] = map[string]any{
				"round":    st.Round,
				"from":     st.From,
				"queried":  qAny,
				"returned": rAny,
			}
		}
		s.Data["last_lookup"] = map[string]any{
			"initiator":  ns.LastLookup.Initiator,
			"target_id":  ns.LastLookup.TargetID,
			"found_peer": ns.LastLookup.FoundPeer,
			"hop_count":  ns.LastLookup.HopCount,
			"successful": ns.LastLookup.Successful,
			"steps":      stepsAny,
		}
	}
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "P2P 节点发现（Kademlia DHT）",
		Description:         "演示 Kademlia XOR 距离 + k-buckets + FIND_NODE 递归查找；含节点加入 / 离线 / 过期逐出",
		Category:            fw.CategoryNodeNetwork,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupNetworkBase},

		// v0.5 协议字段。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"network.p2p.node_count",
			"network.p2p.last_hop_count",
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
		Data:      map[string]any{},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_params", Label: "设置 DHT 参数",
				Description:   "调整 ID 位长 / bucket 容量 k / 并发度 α",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "id_bits", Type: fw.FieldNumber, Label: "ID 位长", Required: true, Default: defaultIDBits, Min: 8, Max: 64, Step: 8},
					{Name: "k", Type: fw.FieldNumber, Label: "bucket 容量 k", Required: true, Default: defaultK, Min: 2, Max: 16, Step: 1},
					{Name: "alpha", Type: fw.FieldNumber, Label: "并发查询度 α", Required: true, Default: defaultAlpha, Min: 1, Max: 8, Step: 1},
				},
			},
			{
				ActionCode: "join_node", Label: "新节点加入",
				Description:   "新节点 PING bootstrap → 触发 FIND_NODE(self) 填充本地 buckets",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "label", Type: fw.FieldString, Label: "节点 label", Required: true, Default: "n8"},
				},
				WritesOwnedFields: []string{"network.p2p.node_count"},
				LinkOwnerFields:   []string{"network.p2p.node_count"},
			},
			{
				ActionCode: "lookup_node", Label: "FIND_NODE 查找",
				Description:   "在 initiator 上发起 FIND_NODE(target_label)",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "initiator", Type: fw.FieldString, Label: "发起节点 label", Required: true, Default: "n0"},
					{Name: "target", Type: fw.FieldString, Label: "目标节点 label", Required: true, Default: "n7"},
				},
				WritesOwnedFields: []string{"network.p2p.last_hop_count"},
				LinkOwnerFields:   []string{"network.p2p.last_hop_count"},
			},
			{
				ActionCode: "ping_node", Label: "PING 检测",
				Description:   "标记节点 down，演示 bucket 中失效节点逐出",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "label", Type: fw.FieldString, Label: "节点 label", Required: true, Default: "n3"},
				},
			},
			{
				ActionCode: "recover_node", Label: "节点恢复",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "label", Type: fw.FieldString, Label: "节点 label", Required: true, Default: "n3"},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			// §0.7.4 混合实验：add_peer 走 geth admin_addPeer。
			{
				ActionCode: "add_peer", Label: "加入真链 peer（容器通道）",
				Description: "调 geth admin_addPeer 接入真实节点",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  "geth attach --exec 'admin.addPeer(\"{{enode_url}}\")' http://geth:8545",
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "enode_url", Type: fw.FieldString, Label: "enode://...", Required: true, Default: ""},
				},
			},
			{
				ActionCode:    "teacher_partition_inject",
				Label:         "教师注入拓扑变更",
				Description:   "仅教师可用，注入拓扑变更用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneTopology,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师注入拓扑变更"},
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
	ns := loadState(state)
	saveState(state, ns)
	state.Phase = "ready"
	env := buildEnvelope(ns, "init", "Kademlia 初始化（8 节点全连通到 bootstrap）", true)
	publishOwnerSubtree(&env, ns)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	ns := loadState(state)
	state.Tick = in.Tick
	env := buildEnvelope(ns, "tick", "", false)
	return fw.StepOutput{Render: env}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	ns := loadState(state)
	out := fw.ActionOutput{Success: true}

	switch in.ActionCode {
	case "set_params":
		ns.IDBits = fw.MapInt(in.Params, "id_bits", defaultIDBits)
		ns.K = fw.MapInt(in.Params, "k", defaultK)
		ns.Alpha = fw.MapInt(in.Params, "alpha", defaultAlpha)
		// 重置网络（参数变化使旧 buckets 不兼容）
		bs := ns.BootstrapID
		ns = defaultNetwork()
		ns.IDBits, ns.K, ns.Alpha, ns.BootstrapID = fw.MapInt(in.Params, "id_bits", defaultIDBits), fw.MapInt(in.Params, "k", defaultK), fw.MapInt(in.Params, "alpha", defaultAlpha), bs
		// 重新派生节点 ID（按新 idBits）
		ns.Nodes = nil
		for i := 0; i < defaultNodeCount; i++ {
			label := fmt.Sprintf("n%d", i)
			ns.Nodes = append(ns.Nodes, newKademliaNode(label, makeID(label+"-seed", ns.IDBits), ns.IDBits))
		}
		bn := ns.findNodeByLabel(ns.BootstrapID)
		if bn != nil {
			for i := range ns.Nodes {
				if ns.Nodes[i].Label != ns.BootstrapID {
					ns.Nodes[i].addPeer(peer{ID: bn.ID}, ns.K)
					bn.addPeer(peer{ID: ns.Nodes[i].ID}, ns.K)
				}
			}
		}
		saveState(state, ns)
		out.Render = buildEnvelope(ns, "set_params",
			fmt.Sprintf("DHT 参数：id_bits=%d, k=%d, α=%d", ns.IDBits, ns.K, ns.Alpha), true)
		return out, nil

	case "join_node":
		label := fw.MapStr(in.Params, "label", "n8")
		if ns.findNodeByLabel(label) != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: label + " 已存在"}, nil
		}
		if len(ns.Nodes) >= maxNodeCount {
			return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("节点数已达上限 %d", maxNodeCount)}, nil
		}
		newNode := newKademliaNode(label, makeID(label+"-seed", ns.IDBits), ns.IDBits)
		bn := ns.findNodeByLabel(ns.BootstrapID)
		if bn != nil {
			newNode.addPeer(peer{ID: bn.ID}, ns.K)
		}
		ns.Nodes = append(ns.Nodes, newNode)
		// 让新节点对自己 ID 做 FIND_NODE 来填充 buckets
		lr := ns.runLookup(label, ns.Nodes[len(ns.Nodes)-1].ID, 8)
		ns.LastLookup = &lr
		saveState(state, ns)
		out.Render = buildEnvelope(ns, "join_node",
			fmt.Sprintf("%s 加入网络（FIND_NODE(self) %d 跳）", label, lr.HopCount), true)
		appendJoinMicroSteps(&out.Render, label)
		out.SharedStateDiff = ownerDiff(ns)
		return out, nil

	case "lookup_node":
		init := fw.MapStr(in.Params, "initiator", "n0")
		target := fw.MapStr(in.Params, "target", "n7")
		if ns.findNodeByLabel(init) == nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到 initiator: " + init}, nil
		}
		targetNode := ns.findNodeByLabel(target)
		if targetNode == nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到 target: " + target}, nil
		}
		lr := ns.runLookup(init, targetNode.ID, 8)
		ns.LastLookup = &lr
		saveState(state, ns)
		summary := fmt.Sprintf("%s → FIND_NODE(%s)：%d 跳，找到 %s", init, target, lr.HopCount, lr.FoundPeer)
		out.Render = buildEnvelope(ns, "lookup_node", summary, false)
		appendLookupMicroSteps(&out.Render, init, target, lr.HopCount)
		out.SharedStateDiff = ownerDiff(ns)
		return out, nil

	case "ping_node":
		label := fw.MapStr(in.Params, "label", "n3")
		n := ns.findNodeByLabel(label)
		if n == nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到节点: " + label}, nil
		}
		n.IsDown = true
		saveState(state, ns)
		out.Render = buildEnvelope(ns, "ping_node", label+" PING 失败 → 标记 down", false)
		return out, nil

	case "recover_node":
		label := fw.MapStr(in.Params, "label", "n3")
		n := ns.findNodeByLabel(label)
		if n == nil {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到节点: " + label}, nil
		}
		n.IsDown = false
		saveState(state, ns)
		out.Render = buildEnvelope(ns, "recover_node", label+" 恢复在线", false)
		return out, nil

	case "teacher_partition_inject":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师注入拓扑变更"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-topo-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		ns = defaultNetwork()
		saveState(state, ns)
		out.Render = buildEnvelope(ns, "reset", "已重置（8 节点 + bootstrap）", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(ns networkState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 40)

	// 1) graph_layout 力导向（节点 + 已知 peer 边）
	nodeIDs := []string{}
	for _, n := range ns.Nodes {
		nodeIDs = append(nodeIDs, "node-"+n.Label)
	}
	edgeIDs := []string{}
	prims = append(prims, fw.PrimGraphLayout("topology", "force", nodeIDs, edgeIDs))

	// 2) 节点
	pathSet := map[string]bool{}
	if ns.LastLookup != nil {
		pathSet[ns.LastLookup.Initiator] = true
		for _, st := range ns.LastLookup.Steps {
			for _, q := range st.Queried {
				pathSet[q] = true
			}
		}
		if ns.LastLookup.FoundPeer != "" {
			pathSet[ns.LastLookup.FoundPeer] = true
		}
	}
	for _, n := range ns.Nodes {
		status := "normal"
		role := "peer"
		if n.IsDown {
			status = "error"
			role = "down"
		} else if n.Label == ns.BootstrapID {
			status = "active"
			role = "bootstrap"
		} else if pathSet[n.Label] {
			status = "active"
			role = "lookup-path"
		}
		label := fmt.Sprintf("%s\nid=%s\npeers=%d", n.Label, hex.EncodeToString(n.ID), n.totalKnownPeers())
		prims = append(prims, fw.PrimNode("node-"+n.Label, label, status, role))
	}

	// 3) 节点之间的"已知 peer"边（对称性：a 知道 b 即画一条）
	drawn := map[string]bool{}
	for _, n := range ns.Nodes {
		for _, b := range n.Buckets {
			for _, p := range b {
				other := ns.labelByID(p.ID)
				if other == "?" {
					continue
				}
				key := edgeKey(n.Label, other)
				if drawn[key] {
					continue
				}
				drawn[key] = true
				style := "solid"
				anim := ""
				if pathSet[n.Label] && pathSet[other] {
					anim = "flow"
				}
				prims = append(prims, fw.PrimEdge(
					"edge-"+key,
					"node-"+n.Label, "node-"+other, style, anim))
			}
		}
	}

	// 4) k-buckets 热力图（取 bootstrap 节点的 buckets）
	bn := ns.findNodeByLabel(ns.BootstrapID)
	if bn != nil {
		cells := make([]map[string]any, 0, len(bn.Buckets)*ns.K)
		for i, b := range bn.Buckets {
			for j := 0; j < ns.K; j++ {
				val := 0
				if j < len(b) {
					val = 1
				}
				color := "muted"
				if val > 0 {
					color = "info"
				}
				cells = append(cells, map[string]any{
					"row": i, "col": j, "value": val, "color_role": color,
				})
			}
		}
		prims = append(prims, fw.PrimHeatMap("buckets-heatmap", len(bn.Buckets), ns.K, cells))
	}

	// 5) 查找路径高亮
	if ns.LastLookup != nil && len(ns.LastLookup.Steps) > 0 {
		ids := []string{"node-" + ns.LastLookup.Initiator}
		for _, st := range ns.LastLookup.Steps {
			for _, q := range st.Queried {
				ids = append(ids, "node-"+q)
			}
		}
		prims = append(prims, fw.PrimVerifyPathHighlight("lookup-path", ids))
	}

	// 6) 公式
	prims = append(prims, fw.PrimMathFormula("formula-xor",
		`d(a, b) = a \oplus b;\quad \mathrm{bucket}_i:\ \mathrm{cpl}(a, b) = i`, false))

	// 7) DHT 参数
	prims = append(prims, fw.PrimCodeBlock("cb-params",
		fmt.Sprintf("ID 位长 = %d\nbucket 容量 k = %d\n并发度 α = %d\nbootstrap = %s\n节点总数 = %d",
			ns.IDBits, ns.K, ns.Alpha, ns.BootstrapID, len(ns.Nodes)),
		"text", nil, 6))

	// 8) 节点表
	rows := []string{"label  id           down   peers  buckets-fill"}
	for _, n := range ns.Nodes {
		fill := ""
		for _, b := range n.Buckets {
			fill += fmt.Sprintf("%d", len(b))
		}
		rows = append(rows, fmt.Sprintf("%-5s  %-12s %-6v %-5d  %s",
			n.Label, hex.EncodeToString(n.ID), n.IsDown, n.totalKnownPeers(), fill))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-nodes", strings.Join(rows, "\n"), "text", nil, 16))

	// 9) 查找步骤
	if ns.LastLookup != nil {
		stepLines := []string{
			fmt.Sprintf("发起 = %s", ns.LastLookup.Initiator),
			fmt.Sprintf("target = %s", ns.LastLookup.TargetID),
			fmt.Sprintf("hops = %d", ns.LastLookup.HopCount),
			fmt.Sprintf("found = %s", ns.LastLookup.FoundPeer),
			"",
		}
		for _, s := range ns.LastLookup.Steps {
			stepLines = append(stepLines,
				fmt.Sprintf("Round %d: %s 询问 [%s] → 返回 [%s]",
					s.Round, s.From, strings.Join(s.Queried, ", "), strings.Join(s.Returned, ", ")))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-lookup", strings.Join(stepLines, "\n"), "text", nil, 16))
	}

	// 10) 动效
	if ns.BootstrapID != "" {
		prims = append(prims, fw.PrimGlow("glow-bootstrap", "node-"+ns.BootstrapID, "info", 0.7))
	}
	if ns.LastLookup != nil && ns.LastLookup.FoundPeer != "" {
		prims = append(prims, fw.PrimBurst("burst-found", "node-"+ns.LastLookup.FoundPeer, "success",
			int64(ns.LastLookup.HopCount), 800))
	}
	for _, n := range ns.Nodes {
		if n.IsDown {
			prims = append(prims, fw.PrimGlow("glow-down-"+n.Label, "node-"+n.Label, "danger", 0.7))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))

	if ns.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "P2P 错误", ns.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(ns, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "p2p.node_count", Value: len(ns.Nodes), TargetPrimitive: "cb-nodes", TargetParam: "count"},
		},
	}
}

func buildSidePanelData(ns networkState, summary string) map[string]any {
	d := map[string]any{
		"id_bits":      ns.IDBits,
		"k":            ns.K,
		"alpha":        ns.Alpha,
		"bootstrap_id": ns.BootstrapID,
		"node_count":   len(ns.Nodes),
		"active_nodes": countActive(ns),
		"down_nodes":   countDown(ns),
	}
	if ns.LastLookup != nil {
		d["last_initiator"] = ns.LastLookup.Initiator
		d["last_target"] = ns.LastLookup.TargetID
		d["last_hop_count"] = ns.LastLookup.HopCount
		d["last_found"] = ns.LastLookup.FoundPeer
		d["last_successful"] = ns.LastLookup.Successful
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

func countActive(ns networkState) int {
	n := 0
	for _, x := range ns.Nodes {
		if !x.IsDown {
			n++
		}
	}
	return n
}

func countDown(ns networkState) int {
	n := 0
	for _, x := range ns.Nodes {
		if x.IsDown {
			n++
		}
	}
	return n
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendJoinMicroSteps(env *fw.RenderEnvelope, label string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "jn-1", Label: label + " 派生节点 ID = SHA-256(seed)", DurationMs: 400, HighlightIDs: []string{"cb-params"}},
		{ID: "jn-2", Label: "PING bootstrap → 加入对方 bucket", DurationMs: 500, HighlightIDs: []string{"node-" + label, "buckets-heatmap"}, FirePrimitives: []string{"glow-bootstrap"}},
		{ID: "jn-3", Label: "FIND_NODE(self) 填充本地 buckets", DurationMs: 600, HighlightIDs: []string{"cb-lookup", "topology"}, IsLinkTrigger: true},
	}
}

func appendLookupMicroSteps(env *fw.RenderEnvelope, init, target string, hops int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "lk-1", Label: init + " 选择本地最近的 α 个 peer", DurationMs: 400, HighlightIDs: []string{"node-" + init, "formula-xor"}},
		{ID: "lk-2", Label: "对每个 peer 发 FIND_NODE(target)", DurationMs: 500, HighlightIDs: []string{"lookup-path"}},
		{ID: "lk-3", Label: "合并返回的更近节点 → 短列表", DurationMs: 500, HighlightIDs: []string{"cb-lookup"}},
		{ID: "lk-4", Label: fmt.Sprintf("递归 %d 跳 → 找到 %s", hops, target), DurationMs: 600, HighlightIDs: []string{"node-" + target}, FirePrimitives: []string{"burst-found"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

// lookupHopCount 取最近一次 lookup 的 hop 数；无 lookup 时返回 0。
func lookupHopCount(ns networkState) int {
	if ns.LastLookup == nil {
		return 0
	}
	return ns.LastLookup.HopCount
}

func publishOwnerSubtree(env *fw.RenderEnvelope, ns networkState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	// LinkTrigger 带锚点（§0.7.1 C18）。
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "p2p-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_topology",
		LinkGroup:      linkGroupNetworkBase,
		ChangedFields:  []string{"network.p2p.node_count", "network.p2p.last_hop_count"},
		Payload: map[string]any{
			"node_count":     len(ns.Nodes),
			"last_hop_count": lookupHopCount(ns),
		},
		SourceAnchorID: "p2p-topology-anchor",
		TargetAnchorID: "network-base-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.p2p.node_count", "network.p2p.last_hop_count")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(ns networkState) map[string]any {
	hops := 0
	if ns.LastLookup != nil {
		hops = ns.LastLookup.HopCount
	}
	return map[string]any{
		"network": map[string]any{
			"p2p": map[string]any{
				"protocol":       "kademlia",
				"id_bits":        ns.IDBits,
				"k":              ns.K,
				"alpha":          ns.Alpha,
				"node_count":     len(ns.Nodes),
				"active_nodes":   countActive(ns),
				"last_hop_count": hops,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func edgeKey(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}
