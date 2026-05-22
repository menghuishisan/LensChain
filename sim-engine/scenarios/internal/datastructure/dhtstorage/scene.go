// 模块：sim-engine/scenarios/internal/datastructure/dhtstorage
// 文件职责：DS-05 DHT 存储（Kademlia STORE / FIND_VALUE）场景的完整实现。
//
// SSOT 依据：06.md §4.4.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Kademlia DHT 存储层（基于 P2P discovery 思路扩展）：
//   · 节点 ID = SHA-256(seed)[:idBytes]，XOR 距离衡量"近"
//   · STORE(key, value)：把 (key, value) 存到与 hash(key) 距离最近的 R 个节点
//   · FIND_VALUE(key)：从所有持有副本的节点中选距离最近的返回
//   · TTL 过期：每个副本在 ttl ticks 后过期；过期前可重新发布
//   · 节点离线：副本数 < R 时数据可能丢失（演示冗余必要性）

package dhtstorage

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

const (
	sceneCode     = "dht-storage"
	schemaVersion = "v1.0.0"
	algorithmType = "kademlia-dht-storage"

	defaultIDBits    = 32
	defaultNodeCount = 8
	defaultR         = 3 // 复制因子
	defaultTTL       = 10
	maxNodeCount     = 16
	maxKeys          = 16

	linkGroupNetworkBase = "network-base-group"
	linkOwnerSubtree     = "datastructure.dht"
)

// =====================================================================
// ID 与 XOR 距离
// =====================================================================

func makeID(seed string, idBits int) []byte {
	h := sha256hash.Sum256([]byte(seed))
	return append([]byte{}, h[:(idBits+7)/8]...)
}

func xorDistance(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func distLess(a, b []byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func leadingZeros(d []byte) int {
	for i, b := range d {
		if b == 0 {
			continue
		}
		return i*8 + bits.LeadingZeros8(b)
	}
	return len(d) * 8
}

// =====================================================================
// 节点 / 数据
// =====================================================================

type valueEntry struct {
	Value      string
	StoredTick int
	Expires    int
}

type dhtNode struct {
	Label  string
	ID     []byte
	IsDown bool
	Store  map[string]valueEntry // key (hex) → value entry
}

type findResult struct {
	Key      string
	KeyHash  []byte
	Found    bool
	FromNode string
	Value    string
	Replicas []string // 持有该 key 的所有节点 label
	Expired  bool
}

type snapState struct {
	IDBits    int
	R         int
	TTL       int
	Tick      int
	Nodes     []dhtNode
	Keys      []string
	LastFind  *findResult
	LastError string
}

func defaultSnapState() snapState {
	st := snapState{
		IDBits: defaultIDBits,
		R:      defaultR,
		TTL:    defaultTTL,
	}
	for i := 0; i < defaultNodeCount; i++ {
		label := fmt.Sprintf("n%d", i)
		st.Nodes = append(st.Nodes, dhtNode{
			Label: label,
			ID:    makeID(label+"-seed", st.IDBits),
			Store: map[string]valueEntry{},
		})
	}
	return st
}

// keyToHash 把 key 字符串映射到与节点 ID 相同长度的字节数组。
func (st snapState) keyToHash(key string) []byte {
	h := sha256hash.Sum256([]byte(key))
	return append([]byte{}, h[:(st.IDBits+7)/8]...)
}

// closestNodes 返回距离 keyHash 最近的 m 个非 down 节点（按 label）。
func (st snapState) closestNodes(keyHash []byte, m int) []*dhtNode {
	type idxDist struct {
		Idx  int
		Dist []byte
	}
	cands := []idxDist{}
	for i := range st.Nodes {
		if st.Nodes[i].IsDown {
			continue
		}
		cands = append(cands, idxDist{Idx: i, Dist: xorDistance(st.Nodes[i].ID, keyHash)})
	}
	sort.Slice(cands, func(a, b int) bool {
		return distLess(cands[a].Dist, cands[b].Dist)
	})
	if len(cands) > m {
		cands = cands[:m]
	}
	out := make([]*dhtNode, 0, len(cands))
	for _, c := range cands {
		out = append(out, &st.Nodes[c.Idx])
	}
	return out
}

// store 把 (key, value) 存到 R 个最近节点。
func (st *snapState) store(key, value string) []string {
	keyHash := st.keyToHash(key)
	keyHex := hex.EncodeToString(keyHash)
	closest := st.closestNodes(keyHash, st.R)
	stored := []string{}
	for _, n := range closest {
		n.Store[keyHex] = valueEntry{
			Value: value, StoredTick: st.Tick, Expires: st.Tick + st.TTL,
		}
		stored = append(stored, n.Label)
	}
	keyExists := false
	for _, k := range st.Keys {
		if k == key {
			keyExists = true
			break
		}
	}
	if !keyExists {
		st.Keys = append(st.Keys, key)
	}
	return stored
}

// findValue 从所有持有该 key 的节点中返回最近未过期的 value。
func (st snapState) findValue(key string) findResult {
	keyHash := st.keyToHash(key)
	keyHex := hex.EncodeToString(keyHash)
	res := findResult{Key: key, KeyHash: keyHash}
	holders := []*dhtNode{}
	for i := range st.Nodes {
		if st.Nodes[i].IsDown {
			continue
		}
		if e, has := st.Nodes[i].Store[keyHex]; has {
			res.Replicas = append(res.Replicas, st.Nodes[i].Label)
			if e.Expires > st.Tick {
				holders = append(holders, &st.Nodes[i])
			} else {
				res.Expired = true
			}
		}
	}
	if len(holders) == 0 {
		res.Found = false
		return res
	}
	sort.Slice(holders, func(i, j int) bool {
		return distLess(xorDistance(holders[i].ID, keyHash), xorDistance(holders[j].ID, keyHash))
	})
	res.Found = true
	res.FromNode = holders[0].Label
	res.Value = holders[0].Store[keyHex].Value
	return res
}

// republish 重新发布所有 key（重置 TTL）。
func (st *snapState) republish() (int, []string) {
	count := 0
	updated := []string{}
	for _, key := range st.Keys {
		// 找到任一未过期持有副本
		keyHash := st.keyToHash(key)
		keyHex := hex.EncodeToString(keyHash)
		var existingValue string
		found := false
		for _, n := range st.Nodes {
			if n.IsDown {
				continue
			}
			if e, has := n.Store[keyHex]; has && e.Expires > st.Tick {
				existingValue = e.Value
				found = true
				break
			}
		}
		if !found {
			continue
		}
		nodes := st.store(key, existingValue)
		count++
		updated = append(updated, fmt.Sprintf("%s→{%s}", key, strings.Join(nodes, ",")))
	}
	return count, updated
}

// expireExpired 模拟 tick 推进时清理过期副本。
func (st *snapState) expireExpired() int {
	count := 0
	for i := range st.Nodes {
		for k, e := range st.Nodes[i].Store {
			if e.Expires <= st.Tick {
				delete(st.Nodes[i].Store, k)
				count++
			}
		}
	}
	return count
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
		IDBits:    fw.MapInt(d, "id_bits", defaultIDBits),
		R:         fw.MapInt(d, "r", defaultR),
		TTL:       fw.MapInt(d, "ttl", defaultTTL),
		Tick:      fw.MapInt(d, "tick", 0),
		LastError: fw.MapStr(d, "last_error", ""),
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				idHex := fw.MapStr(nm, "id", "")
				id, _ := hex.DecodeString(idHex)
				n := dhtNode{
					Label:  fw.MapStr(nm, "label", ""),
					ID:     id,
					IsDown: fw.MapBool(nm, "down", false),
					Store:  map[string]valueEntry{},
				}
				if stAny, ok := nm["store"].(map[string]any); ok {
					for k, v := range stAny {
						if vm, ok := v.(map[string]any); ok {
							n.Store[k] = valueEntry{
								Value:      fw.MapStr(vm, "value", ""),
								StoredTick: fw.MapInt(vm, "stored", 0),
								Expires:    fw.MapInt(vm, "expires", 0),
							}
						}
					}
				}
				st.Nodes = append(st.Nodes, n)
			}
		}
	}
	if len(st.Nodes) == 0 {
		return defaultSnapState()
	}
	if keysAny, ok := d["keys"].([]any); ok {
		for _, k := range keysAny {
			if s, ok := k.(string); ok {
				st.Keys = append(st.Keys, s)
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["id_bits"] = st.IDBits
	s.Data["r"] = st.R
	s.Data["ttl"] = st.TTL
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	nodesAny := make([]any, len(st.Nodes))
	for i, n := range st.Nodes {
		store := map[string]any{}
		for k, v := range n.Store {
			store[k] = map[string]any{
				"value": v.Value, "stored": v.StoredTick, "expires": v.Expires,
			}
		}
		nodesAny[i] = map[string]any{
			"label": n.Label, "id": hex.EncodeToString(n.ID),
			"down": n.IsDown, "store": store,
		}
	}
	s.Data["nodes"] = nodesAny
	keysAny := make([]any, len(st.Keys))
	for i, k := range st.Keys {
		keysAny[i] = k
	}
	s.Data["keys"] = keysAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "DHT 分布式存储",
		Description:         "演示 Kademlia DHT STORE / FIND_VALUE：复制因子 R + TTL + 重新发布 + 节点离线导致数据丢失",
		Category:            fw.CategoryDataStructure,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupNetworkBase},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"datastructure.dht.key_count",
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
				ActionCode: "set_params", Label: "设置 DHT 参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "r", Type: fw.FieldNumber, Label: "复制因子 R", Required: true, Default: defaultR, Min: 1, Max: 8, Step: 1},
					{Name: "ttl", Type: fw.FieldNumber, Label: "TTL (ticks)", Required: true, Default: defaultTTL, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "store", Label: "存储 (key, value)",
				Description:   "存到与 hash(key) 距离最近的 R 个节点",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "key", Type: fw.FieldString, Label: "key", Required: true, Default: "alice"},
					{Name: "value", Type: fw.FieldString, Label: "value", Required: true, Default: "0xCAFE"},
				},
				WritesOwnedFields: []string{"datastructure.dht.key_count"},
				LinkOwnerFields:   []string{"datastructure.dht.key_count"},
			},
			{
				ActionCode: "find_value", Label: "查询 key",
				Category:      fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "key", Type: fw.FieldString, Label: "key", Required: true, Default: "alice"},
				},
				LinkOwnerFields: []string{"datastructure.dht.last_find_result"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Description:   "tick++，过期副本自动清理",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode: "step_n_ticks", Label: "推进 N tick",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 5, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "republish", Label: "重新发布所有 key",
				Description:   "对每个尚未全过期的 key 重新执行 STORE",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
			},
			{
				ActionCode: "crash_node", Label: "节点离线",
				Description:   "副本数 < R 时数据可能丢失",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "label", Type: fw.FieldString, Label: "节点 label", Required: true, Default: "n0"},
				},
			},
			{
				ActionCode: "recover_node", Label: "节点恢复",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
				Fields: []fw.FieldDef{
					{Name: "label", Type: fw.FieldString, Label: "节点 label", Required: true, Default: "n0"},
				},
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
	env := buildEnvelope(st, "init", "DHT Storage 初始化（8 节点 R=3 TTL=10）", true)
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
		st.R = fw.MapInt(in.Params, "r", defaultR)
		st.TTL = fw.MapInt(in.Params, "ttl", defaultTTL)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", fmt.Sprintf("R=%d TTL=%d", st.R, st.TTL), false)
		return out, nil

	case "store":
		key := fw.MapStr(in.Params, "key", "alice")
		value := fw.MapStr(in.Params, "value", "0xCAFE")
		if len(st.Keys) >= maxKeys {
			exists := false
			for _, k := range st.Keys {
				if k == key {
					exists = true
					break
				}
			}
			if !exists {
				return fw.ActionOutput{Success: false, ErrorMessage: fmt.Sprintf("key 数 ≥ %d", maxKeys)}, nil
			}
		}
		nodes := st.store(key, value)
		saveState(state, st)
		out.Render = buildEnvelope(st, "store",
			fmt.Sprintf("STORE(%s, %s) → %d 副本: [%s]", key, value, len(nodes), strings.Join(nodes, ", ")), false)
		appendStoreMicroSteps(&out.Render, key, len(nodes))
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "find_value":
		key := fw.MapStr(in.Params, "key", "alice")
		res := st.findValue(key)
		st.LastFind = &res
		saveState(state, st)
		summary := fmt.Sprintf("FIND_VALUE(%s) → not found", key)
		if res.Found {
			summary = fmt.Sprintf("FIND_VALUE(%s) → \"%s\" from %s (replicas=%d)", key, res.Value, res.FromNode, len(res.Replicas))
		}
		out.Render = buildEnvelope(st, "find_value", summary, false)
		appendFindMicroSteps(&out.Render, key, res.Found)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_tick":
		st.Tick++
		expired := st.expireExpired()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_tick", fmt.Sprintf("tick=%d 清理过期副本 %d 个", st.Tick, expired), false)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 5)
		expiredTotal := 0
		for i := 0; i < n; i++ {
			st.Tick++
			expiredTotal += st.expireExpired()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n_ticks",
			fmt.Sprintf("推进 %d tick → 累计清理 %d 副本", n, expiredTotal), false)
		return out, nil

	case "republish":
		count, _ := st.republish()
		saveState(state, st)
		out.Render = buildEnvelope(st, "republish", fmt.Sprintf("重新发布 %d 个 key", count), false)
		appendRepublishMicroSteps(&out.Render, count)
		return out, nil

	case "crash_node":
		label := fw.MapStr(in.Params, "label", "n0")
		for i := range st.Nodes {
			if st.Nodes[i].Label == label {
				st.Nodes[i].IsDown = true
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "crash_node", label+" 离线", false)
		appendCrashMicroSteps(&out.Render, label)
		return out, nil

	case "recover_node":
		label := fw.MapStr(in.Params, "label", "n0")
		for i := range st.Nodes {
			if st.Nodes[i].Label == label {
				st.Nodes[i].IsDown = false
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "recover_node", label+" 恢复", false)
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

	// 1) 节点环（按 st.Nodes 顺序声明 ring 成员）
	nodeRingIDs := make([]string, len(st.Nodes))
	for i, n := range st.Nodes {
		nodeRingIDs[i] = "node-" + n.Label
	}
	prims = append(prims, fw.PrimRingLayout("node-ring", nodeRingIDs))
	holdersSet := map[string]bool{}
	if st.LastFind != nil {
		for _, h := range st.LastFind.Replicas {
			holdersSet[h] = true
		}
	}
	for _, n := range st.Nodes {
		role := "node"
		status := "normal"
		if n.IsDown {
			status = "error"
			role = "down"
		} else if holdersSet[n.Label] {
			status = "active"
			role = "replica"
		} else if len(n.Store) > 0 {
			role = "holder"
		}
		label := fmt.Sprintf("%s\nstore=%d\nid=%s", n.Label, len(n.Store), hex.EncodeToString(n.ID))
		prims = append(prims, fw.PrimNode("node-"+n.Label, label, status, role))
	}

	// 2) heat_map：节点 × key 持有矩阵
	if len(st.Keys) > 0 {
		cells := make([]map[string]any, 0, len(st.Nodes)*len(st.Keys))
		for i, n := range st.Nodes {
			for j, key := range st.Keys {
				keyHex := hex.EncodeToString(st.keyToHash(key))
				val := 0
				color := "muted"
				if e, has := n.Store[keyHex]; has {
					val = 1
					if e.Expires > st.Tick {
						color = "success"
					} else {
						color = "warning"
					}
				}
				if n.IsDown {
					color = "danger"
				}
				cells = append(cells, map[string]any{
					"row": i, "col": j, "value": val, "color_role": color,
				})
			}
		}
		prims = append(prims, fw.PrimHeatMap("storage-matrix", len(st.Nodes), len(st.Keys), cells))
	}

	// 3) 公式
	prims = append(prims, fw.PrimMathFormula("formula-store",
		`\text{STORE}(k, v):\ \text{closest}_R(\mathrm{hash}(k))\ \text{节点存 } v;\ \ \text{TTL}\ \text{后过期}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-find",
		`\text{FIND\_VALUE}(k):\ \text{any holder of}\ k\ \text{未过期} \to \text{return}`, false))

	// 4) 状态
	totalReplicas := 0
	for _, n := range st.Nodes {
		totalReplicas += len(n.Store)
	}
	activeNodes := 0
	for _, n := range st.Nodes {
		if !n.IsDown {
			activeNodes++
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d\nID 位长 = %d\nR (复制因子) = %d\nTTL = %d\n节点 = %d (active=%d)\nkey 数 = %d\n副本总数 = %d",
			st.Tick, st.IDBits, st.R, st.TTL,
			len(st.Nodes), activeNodes, len(st.Keys), totalReplicas),
		"text", nil, 8))

	// 5) keys 表
	if len(st.Keys) > 0 {
		keyRows := []string{"key       hash             holders"}
		for _, key := range st.Keys {
			keyHex := hex.EncodeToString(st.keyToHash(key))
			holders := []string{}
			for _, n := range st.Nodes {
				if e, has := n.Store[keyHex]; has && e.Expires > st.Tick && !n.IsDown {
					holders = append(holders, n.Label)
				}
			}
			status := "ok"
			if len(holders) < st.R {
				status = "DEGRADED"
			}
			if len(holders) == 0 {
				status = "LOST"
			}
			keyRows = append(keyRows, fmt.Sprintf("%-9s %s [%s] %s",
				key, keyHex[:14], strings.Join(holders, ","), status))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-keys", strings.Join(keyRows, "\n"), "text", nil, 12))
	}

	// 6) 节点详情
	nodeRows := []string{"label  id           down  keys  cpl→hash"}
	for _, n := range st.Nodes {
		cplStr := ""
		if st.LastFind != nil {
			cplStr = fmt.Sprintf("cpl=%d", leadingZeros(xorDistance(n.ID, st.LastFind.KeyHash)))
		}
		nodeRows = append(nodeRows, fmt.Sprintf("%-5s  %s   %-5v  %-4d  %s",
			n.Label, hex.EncodeToString(n.ID), n.IsDown, len(n.Store), cplStr))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-nodes", strings.Join(nodeRows, "\n"), "text", nil, 12))

	// 7) 上次查询
	if st.LastFind != nil {
		queryLines := []string{
			fmt.Sprintf("last_find = %s", st.LastFind.Key),
			fmt.Sprintf("hash      = %s", hex.EncodeToString(st.LastFind.KeyHash)),
			fmt.Sprintf("found     = %v", st.LastFind.Found),
			fmt.Sprintf("from      = %s", st.LastFind.FromNode),
			fmt.Sprintf("value     = %s", st.LastFind.Value),
			fmt.Sprintf("replicas  = [%s]", strings.Join(st.LastFind.Replicas, ", ")),
		}
		if st.LastFind.Expired {
			queryLines = append(queryLines, "⚠ 部分副本已过期")
		}
		prims = append(prims, fw.PrimCodeBlock("cb-query", strings.Join(queryLines, "\n"), "text", nil, 10))
	}

	// 8) 进度条：副本充足度
	if len(st.Keys) > 0 {
		ok := 0
		for _, key := range st.Keys {
			keyHex := hex.EncodeToString(st.keyToHash(key))
			holders := 0
			for _, n := range st.Nodes {
				if e, has := n.Store[keyHex]; has && e.Expires > st.Tick && !n.IsDown {
					holders++
				}
			}
			if holders >= st.R {
				ok++
			}
		}
		prims = append(prims, fw.PrimProgressBar("redundancy", float64(ok), float64(len(st.Keys)),
			fmt.Sprintf("满副本 key %d/%d", ok, len(st.Keys))))
	}

	// 9) 动效
	for _, n := range st.Nodes {
		if n.IsDown {
			prims = append(prims, fw.PrimGlow("glow-down-"+n.Label, "node-"+n.Label, "danger", 0.7))
		}
		if holdersSet[n.Label] {
			prims = append(prims, fw.PrimGlow("glow-replica-"+n.Label, "node-"+n.Label, "info", 0.7))
		}
	}
	if st.LastFind != nil && st.LastFind.Found {
		prims = append(prims, fw.PrimBurst("burst-found", "node-"+st.LastFind.FromNode, "success",
			int64(st.Tick), 800))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "DHT 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	totalReplicas := 0
	for _, n := range st.Nodes {
		totalReplicas += len(n.Store)
	}
	d := map[string]any{
		"id_bits":       st.IDBits,
		"r":             st.R,
		"ttl":           st.TTL,
		"tick":          st.Tick,
		"node_count":    len(st.Nodes),
		"key_count":     len(st.Keys),
		"replica_count": totalReplicas,
	}
	if st.LastFind != nil {
		d["last_find_result"] = st.LastFind.Found
		d["last_find_key"] = st.LastFind.Key
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendStoreMicroSteps(env *fw.RenderEnvelope, key string, n int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "st-1", Label: fmt.Sprintf("hash(%s) → 256-bit", key), DurationMs: 400, HighlightIDs: []string{"formula-store"}},
		{ID: "st-2", Label: fmt.Sprintf("找最近 R 个节点（%d 个）", n), DurationMs: 500, HighlightIDs: []string{"node-ring", "cb-nodes"}},
		{ID: "st-3", Label: "在每个节点的 store 中保存 (key, value)", DurationMs: 500, HighlightIDs: []string{"storage-matrix"}, IsLinkTrigger: true},
	}
}

func appendFindMicroSteps(env *fw.RenderEnvelope, key string, found bool) {
	tail := "未找到（无副本或全过期）"
	if found {
		tail = "✓ 返回最近未过期副本的 value"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "fd-1", Label: fmt.Sprintf("hash(%s)", key), DurationMs: 400, HighlightIDs: []string{"formula-find"}},
		{ID: "fd-2", Label: "查询所有持有副本的节点", DurationMs: 500, HighlightIDs: []string{"storage-matrix"}},
		{ID: "fd-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-query"}, FirePrimitives: []string{"burst-found"}, IsLinkTrigger: true},
	}
}

func appendRepublishMicroSteps(env *fw.RenderEnvelope, count int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "rp-1", Label: "枚举所有 key", DurationMs: 400, HighlightIDs: []string{"cb-keys"}},
		{ID: "rp-2", Label: "对每个未全过期 key 重 STORE", DurationMs: 500, HighlightIDs: []string{"storage-matrix"}},
		{ID: "rp-3", Label: fmt.Sprintf("已重新发布 %d 个 key", count), DurationMs: 400, HighlightIDs: []string{"redundancy"}, IsLinkTrigger: true},
	}
}

func appendCrashMicroSteps(env *fw.RenderEnvelope, label string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "cr-1", Label: label + " 离线", DurationMs: 400, HighlightIDs: []string{"node-" + label}, FirePrimitives: []string{"glow-down-" + label}},
		{ID: "cr-2", Label: "其副本不再可达 → 部分 key 副本数 < R", DurationMs: 500, HighlightIDs: []string{"redundancy", "cb-keys"}},
		{ID: "cr-3", Label: "可触发 republish 修复或等下次 STORE 自然恢复", DurationMs: 500, HighlightIDs: []string{"formula-store"}, IsLinkTrigger: true},
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
		ID:             "dht-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_dht",
		LinkGroup:      linkGroupNetworkBase,
		ChangedFields:  []string{"datastructure.dht.key_count"},
		Payload:        map[string]any{"key_count": len(st.Keys)},
		SourceAnchorID: "dht-output-anchor",
		TargetAnchorID: "network-base-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "datastructure.dht.key_count")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	totalReplicas := 0
	for _, n := range st.Nodes {
		totalReplicas += len(n.Store)
	}
	return map[string]any{
		"datastructure": map[string]any{
			"dht": map[string]any{
				"id_bits":       st.IDBits,
				"r":             st.R,
				"ttl":           st.TTL,
				"node_count":    len(st.Nodes),
				"key_count":     len(st.Keys),
				"replica_count": totalReplicas,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

