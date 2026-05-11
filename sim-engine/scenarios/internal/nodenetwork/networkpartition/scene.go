// 模块：sim-engine/scenarios/internal/nodenetwork/networkpartition
// 文件职责：NET-03 网络分区（CAP 定理演示）场景的完整实现。
//
// SSOT 依据：06.md §4.1.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现网络分区演示（零外部依赖）：
//   · N 节点维护 KV 存储（key → (value, version, writer, tick, partition)）
//   · CP 模式：写入须 quorum > N/2 副本可达，否则拒绝
//   · AP 模式：每分区独立接受写，恢复时 LWW (last-write-wins) 合并冲突
//   · 跨分区消息丢弃，DroppedCount 累计
//   · 恢复后 Conflicts 列表展示哪些 key 在不同分区写入了不同值

package networkpartition

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "network-partition"
	schemaVersion = "v1.0.0"
	algorithmType = "cap-partition"

	defaultNodeCount = 6
	maxNodeCount     = 12

	modeCP = "cp"
	modeAP = "ap"

	linkGroupNetworkBase = "network-base-group"
	linkOwnerSubtree     = "network.partition"
)

type kvEntry struct {
	Key       string
	Value     string
	Version   int
	Writer    string
	WriteTick int
	Partition int
}

type pnode struct {
	ID        string
	Partition int
	IsDown    bool
	KV        map[string]kvEntry
}

type conflictRecord struct {
	Key      string
	Versions []kvEntry
	Winner   string
}

type snapState struct {
	Nodes         []pnode
	Mode          string
	Partitioned   bool
	Tick          int
	GlobalVersion int
	DroppedCount  int
	Conflicts     []conflictRecord
	EventLog      []string
	LastError     string
}

func defaultSnapState() snapState {
	st := snapState{Mode: modeAP}
	for i := 0; i < defaultNodeCount; i++ {
		st.Nodes = append(st.Nodes, pnode{ID: fmt.Sprintf("n%d", i), KV: map[string]kvEntry{}})
	}
	return st
}

func activeCount(nodes []pnode) int {
	n := 0
	for _, x := range nodes {
		if !x.IsDown {
			n++
		}
	}
	return n
}

func (st snapState) majority() int { return activeCount(st.Nodes)/2 + 1 }

// partitionMembers 返回某分区内非 down 节点的指针。
func (st *snapState) partitionMembers(p int) []*pnode {
	out := []*pnode{}
	for i := range st.Nodes {
		if st.Nodes[i].IsDown {
			continue
		}
		if st.Partitioned && st.Nodes[i].Partition != p {
			continue
		}
		out = append(out, &st.Nodes[i])
	}
	return out
}

// writeKV 在节点 nid 上写入 (key, value)，按 CP / AP 规则处理。
func (st *snapState) writeKV(nid, key, value string) error {
	idx := -1
	for i, n := range st.Nodes {
		if n.ID == nid {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("未找到节点: %s", nid)
	}
	if st.Nodes[idx].IsDown {
		return fmt.Errorf("%s 已离线", nid)
	}
	myP := st.Nodes[idx].Partition
	members := st.partitionMembers(myP)
	if st.Mode == modeCP && st.Partitioned && len(members) < st.majority() {
		st.DroppedCount++
		st.pushEvent(fmt.Sprintf("⚠ CP：%s 所在 P%d 副本 %d < majority %d → 拒绝", nid, myP, len(members), st.majority()))
		return fmt.Errorf("CP 模式拒绝：分区 %d 副本不足", myP)
	}
	st.GlobalVersion++
	entry := kvEntry{Key: key, Value: value, Version: st.GlobalVersion, Writer: nid, WriteTick: st.Tick, Partition: myP}
	for _, m := range members {
		old, has := m.KV[key]
		if has && old.Version >= entry.Version {
			continue
		}
		m.KV[key] = entry
	}
	st.pushEvent(fmt.Sprintf("%s @P%d 写 %s=%s (v=%d) → %d 副本", nid, myP, key, value, entry.Version, len(members)))
	return nil
}

// applyPartition 把节点重新划分到分区。
func (st *snapState) applyPartition(splitAt int) {
	st.Partitioned = true
	for i := range st.Nodes {
		if i < splitAt {
			st.Nodes[i].Partition = 1
		} else {
			st.Nodes[i].Partition = 2
		}
	}
	st.pushEvent(fmt.Sprintf("⚠ 网络分区：前 %d → P1，其余 → P2", splitAt))
}

// recoverPartition 恢复并合并：LWW 选 winner，记录冲突。
func (st *snapState) recoverPartition() {
	if !st.Partitioned {
		return
	}
	allVersions := map[string][]kvEntry{}
	for _, n := range st.Nodes {
		if n.IsDown {
			continue
		}
		for k, e := range n.KV {
			seen := false
			for _, ex := range allVersions[k] {
				if ex.Version == e.Version && ex.Writer == e.Writer {
					seen = true
					break
				}
			}
			if !seen {
				allVersions[k] = append(allVersions[k], e)
			}
		}
	}
	st.Conflicts = nil
	merged := map[string]kvEntry{}
	for k, versions := range allVersions {
		sort.Slice(versions, func(a, b int) bool {
			if versions[a].WriteTick != versions[b].WriteTick {
				return versions[a].WriteTick > versions[b].WriteTick
			}
			if versions[a].Version != versions[b].Version {
				return versions[a].Version > versions[b].Version
			}
			return versions[a].Writer < versions[b].Writer
		})
		merged[k] = versions[0]
		distinct := map[string]bool{}
		for _, v := range versions {
			distinct[v.Value] = true
		}
		if len(distinct) > 1 {
			st.Conflicts = append(st.Conflicts, conflictRecord{Key: k, Versions: versions, Winner: versions[0].Writer})
		}
	}
	st.Partitioned = false
	for i := range st.Nodes {
		st.Nodes[i].Partition = 0
		st.Nodes[i].KV = map[string]kvEntry{}
		for k, v := range merged {
			st.Nodes[i].KV[k] = v
		}
	}
	st.pushEvent(fmt.Sprintf("✓ 恢复：合并 %d 键，冲突 %d", len(merged), len(st.Conflicts)))
}

func (st *snapState) pushEvent(evt string) {
	st.EventLog = append(st.EventLog, fmt.Sprintf("[t=%d] %s", st.Tick, evt))
	if len(st.EventLog) > 24 {
		st.EventLog = st.EventLog[len(st.EventLog)-24:]
	}
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
		Mode:          fw.MapStr(d, "mode", modeAP),
		Partitioned:   fw.MapBool(d, "partitioned", false),
		Tick:          fw.MapInt(d, "tick", 0),
		GlobalVersion: fw.MapInt(d, "global_version", 0),
		DroppedCount:  fw.MapInt(d, "dropped_count", 0),
		LastError:     fw.MapStr(d, "last_error", ""),
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				n := pnode{
					ID:        fw.MapStr(nm, "id", ""),
					Partition: fw.MapInt(nm, "partition", 0),
					IsDown:    fw.MapBool(nm, "down", false),
					KV:        map[string]kvEntry{},
				}
				if kvAny, ok := nm["kv"].(map[string]any); ok {
					for k, v := range kvAny {
						if vm, ok := v.(map[string]any); ok {
							n.KV[k] = kvEntry{
								Key: k, Value: fw.MapStr(vm, "value", ""),
								Version: fw.MapInt(vm, "version", 0), Writer: fw.MapStr(vm, "writer", ""),
								WriteTick: fw.MapInt(vm, "tick", 0), Partition: fw.MapInt(vm, "partition", 0),
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
	if cAny, ok := d["conflicts"].([]any); ok {
		for _, cV := range cAny {
			if cm, ok := cV.(map[string]any); ok {
				cr := conflictRecord{Key: fw.MapStr(cm, "key", ""), Winner: fw.MapStr(cm, "winner", "")}
				if vsAny, ok := cm["versions"].([]any); ok {
					for _, vAny := range vsAny {
						if vm, ok := vAny.(map[string]any); ok {
							cr.Versions = append(cr.Versions, kvEntry{
								Key: fw.MapStr(vm, "key", ""), Value: fw.MapStr(vm, "value", ""),
								Version: fw.MapInt(vm, "version", 0), Writer: fw.MapStr(vm, "writer", ""),
								WriteTick: fw.MapInt(vm, "tick", 0), Partition: fw.MapInt(vm, "partition", 0),
							})
						}
					}
				}
				st.Conflicts = append(st.Conflicts, cr)
			}
		}
	}
	if logAny, ok := d["event_log"].([]any); ok {
		for _, e := range logAny {
			if s, ok := e.(string); ok {
				st.EventLog = append(st.EventLog, s)
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["mode"] = st.Mode
	s.Data["partitioned"] = st.Partitioned
	s.Data["tick"] = st.Tick
	s.Data["global_version"] = st.GlobalVersion
	s.Data["dropped_count"] = st.DroppedCount
	s.Data["last_error"] = st.LastError
	nodesAny := make([]any, len(st.Nodes))
	for i, n := range st.Nodes {
		kv := map[string]any{}
		for k, e := range n.KV {
			kv[k] = map[string]any{
				"value": e.Value, "version": e.Version,
				"writer": e.Writer, "tick": e.WriteTick, "partition": e.Partition,
			}
		}
		nodesAny[i] = map[string]any{
			"id": n.ID, "partition": n.Partition, "down": n.IsDown, "kv": kv,
		}
	}
	s.Data["nodes"] = nodesAny
	confAny := make([]any, len(st.Conflicts))
	for i, c := range st.Conflicts {
		vs := make([]any, len(c.Versions))
		for j, v := range c.Versions {
			vs[j] = map[string]any{
				"key": v.Key, "value": v.Value, "version": v.Version,
				"writer": v.Writer, "tick": v.WriteTick, "partition": v.Partition,
			}
		}
		confAny[i] = map[string]any{"key": c.Key, "winner": c.Winner, "versions": vs}
	}
	s.Data["conflicts"] = confAny
	logAny := make([]any, len(st.EventLog))
	for i, e := range st.EventLog {
		logAny[i] = e
	}
	s.Data["event_log"] = logAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "网络分区（CAP 演示）",
		Description:         "演示网络分区下 CP / AP 模式的写可用性 + 跨分区消息丢弃 + 恢复时冲突合并 (LWW)",
		Category:            fw.CategoryNodeNetwork,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupNetworkBase},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"network.partition.partitioned",
			"network.partition.global_version",
			"network.partition.conflicts_count",
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
				ActionCode: "set_mode", Label: "设置 CAP 模式",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "mode", Type: fw.FieldEnum, Label: "CP=拒绝少数派写 / AP=接受所有写",
						Required: true, Default: modeAP, Options: []any{modeCP, modeAP}},
				},
			},
			{
				ActionCode: "create_partition", Label: "创建网络分区",
				Description:   "把前 split_at 节点放到 P1，其余 P2",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "split_at", Type: fw.FieldNumber, Label: "前 N 节点入 P1", Required: true,
						Default: defaultNodeCount / 2, Min: 1, Max: maxNodeCount - 1, Step: 1},
				},
				WritesOwnedFields: []string{"network.partition.partitioned"},
				LinkOwnerFields:   []string{"network.partition.partitioned"},
			},
			{
				ActionCode: "write_kv", Label: "在节点写入 KV",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n0"},
					{Name: "key", Type: fw.FieldString, Label: "key", Required: true, Default: "x"},
					{Name: "value", Type: fw.FieldString, Label: "value", Required: true, Default: "1"},
				},
				WritesOwnedFields: []string{"network.partition.global_version"},
				LinkOwnerFields:   []string{"network.partition.global_version"},
			},
			{
				ActionCode: "recover", Label: "恢复网络（合并冲突）",
				Description: "LWW 合并所有分区 KV，记录 distinct value 冲突",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"network.partition.conflicts_count"},
				LinkOwnerFields:   []string{"network.partition.conflicts_count"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
			},
			{
				ActionCode: "crash_node", Label: "节点离线",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n2"}},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
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
	st := loadState(state)
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "Network Partition 初始化（6 节点，AP 模式）", true)
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
	case "set_mode":
		st.Mode = fw.MapStr(in.Params, "mode", modeAP)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_mode", "切换为 "+st.Mode+" 模式", false)
		return out, nil

	case "create_partition":
		split := fw.MapInt(in.Params, "split_at", defaultNodeCount/2)
		if split < 1 || split >= len(st.Nodes) {
			return fw.ActionOutput{Success: false, ErrorMessage: "split_at 越界"}, nil
		}
		st.applyPartition(split)
		saveState(state, st)
		out.Render = buildEnvelope(st, "create_partition", fmt.Sprintf("分区生效：P1=%d 节点 / P2=%d 节点", split, len(st.Nodes)-split), false)
		appendPartitionMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "write_kv":
		nid := fw.MapStr(in.Params, "node_id", "n0")
		key := fw.MapStr(in.Params, "key", "x")
		value := fw.MapStr(in.Params, "value", "1")
		if err := st.writeKV(nid, key, value); err != nil {
			saveState(state, st)
			out.Render = buildEnvelope(st, "write_kv", err.Error(), false)
			appendWriteMicroSteps(&out.Render, false, st.Mode)
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error(), Render: out.Render}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "write_kv", fmt.Sprintf("%s.%s=%s 写入成功", nid, key, value), false)
		appendWriteMicroSteps(&out.Render, true, st.Mode)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "recover":
		if !st.Partitioned {
			return fw.ActionOutput{Success: false, ErrorMessage: "当前未分区"}, nil
		}
		st.recoverPartition()
		saveState(state, st)
		out.Render = buildEnvelope(st, "recover", fmt.Sprintf("已恢复（冲突 %d）", len(st.Conflicts)), false)
		appendRecoverMicroSteps(&out.Render, len(st.Conflicts))
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_tick":
		st.Tick++
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_tick", fmt.Sprintf("tick=%d", st.Tick), false)
		return out, nil

	case "crash_node":
		nid := fw.MapStr(in.Params, "node_id", "n2")
		for i := range st.Nodes {
			if st.Nodes[i].ID == nid {
				st.Nodes[i].IsDown = true
				st.pushEvent(nid + " 离线")
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "crash_node", nid+" 离线", false)
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
	prims := make([]fw.Primitive, 0, 30)

	// 1) graph_layout 拓扑
	nodeIDs := []string{}
	for _, n := range st.Nodes {
		nodeIDs = append(nodeIDs, "node-"+n.ID)
	}
	prims = append(prims, fw.PrimGraphLayout("topology", "force", nodeIDs, nil))

	// 2) 节点
	for _, n := range st.Nodes {
		status := "normal"
		role := "node"
		if n.IsDown {
			status = "error"
			role = "down"
		} else if st.Partitioned {
			role = fmt.Sprintf("partition-%d", n.Partition)
			if n.Partition == 1 {
				status = "active"
			} else {
				status = "warning"
			}
		}
		label := fmt.Sprintf("%s\nP%d\nkeys=%d", n.ID, n.Partition, len(n.KV))
		prims = append(prims, fw.PrimNode("node-"+n.ID, label, status, role))
	}

	// 3) 同分区节点之间的边（dashed if 跨分区）
	for i := 0; i < len(st.Nodes); i++ {
		for j := i + 1; j < len(st.Nodes); j++ {
			a, b := st.Nodes[i], st.Nodes[j]
			style := "solid"
			if st.Partitioned && a.Partition != b.Partition {
				style = "dashed"
			}
			prims = append(prims, fw.PrimEdge(
				fmt.Sprintf("edge-%s-%s", a.ID, b.ID),
				"node-"+a.ID, "node-"+b.ID, style, ""))
		}
	}

	// 4) 分区遮罩
	if st.Partitioned {
		prims = append(prims, fw.PrimPartitionZone("zone-1",
			[]map[string]float64{{"x": 0.0, "y": 0.0}, {"x": 0.5, "y": 0.0}, {"x": 0.5, "y": 1.0}, {"x": 0.0, "y": 1.0}},
			"P1"))
		prims = append(prims, fw.PrimPartitionZone("zone-2",
			[]map[string]float64{{"x": 0.5, "y": 0.0}, {"x": 1.0, "y": 0.0}, {"x": 1.0, "y": 1.0}, {"x": 0.5, "y": 1.0}},
			"P2"))
	}

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-cap",
		`\text{CAP}: \text{C} \land \text{A} \land \text{P}\ \text{至多三选二}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-cp",
		`\text{CP}:\ \text{write\_ok} \iff |\text{partition}| \ge \lceil N/2 \rceil + 1`, false))

	// 6) 关键参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("mode = %s\npartitioned = %v\nN = %d (active=%d)\nmajority = %d\nglobal_version = %d\ndropped = %d\nconflicts = %d",
			st.Mode, st.Partitioned, len(st.Nodes), activeCount(st.Nodes),
			st.majority(), st.GlobalVersion, st.DroppedCount, len(st.Conflicts)),
		"text", nil, 8))

	// 7) 节点 KV 视图
	for _, n := range st.Nodes {
		if n.IsDown {
			continue
		}
	}
	rows := []string{"node  P  keys"}
	for _, n := range st.Nodes {
		kvStr := ""
		for k, e := range n.KV {
			if kvStr != "" {
				kvStr += ", "
			}
			kvStr += fmt.Sprintf("%s=%s(v%d)", k, e.Value, e.Version)
		}
		flag := ""
		if n.IsDown {
			flag = " [DOWN]"
		}
		rows = append(rows, fmt.Sprintf("%-4s  %d  %s%s", n.ID, n.Partition, kvStr, flag))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-kv", strings.Join(rows, "\n"), "text", nil, 14))

	// 8) 冲突列表
	if len(st.Conflicts) > 0 {
		conflictLines := []string{"恢复后检测到的冲突（LWW 已选 winner）："}
		for _, c := range st.Conflicts {
			conflictLines = append(conflictLines, fmt.Sprintf("  %s → winner=%s (v=%d, val=%s)",
				c.Key, c.Winner, c.Versions[0].Version, c.Versions[0].Value))
			for _, v := range c.Versions[1:] {
				conflictLines = append(conflictLines, fmt.Sprintf("    loser: %s @P%d v=%d val=%s",
					v.Writer, v.Partition, v.Version, v.Value))
			}
		}
		prims = append(prims, fw.PrimCodeBlock("cb-conflicts", strings.Join(conflictLines, "\n"), "text", nil, 14))
	}

	// 9) 事件
	if len(st.EventLog) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(st.EventLog, "\n"), "text", nil, 16))
	}

	// 10) 动效
	for _, n := range st.Nodes {
		if n.IsDown {
			prims = append(prims, fw.PrimGlow("glow-down-"+n.ID, "node-"+n.ID, "danger", 0.7))
		}
	}
	if st.Partitioned {
		prims = append(prims, fw.PrimPulse("pulse-partition", "zone-1", "warning", 1500))
	}
	if len(st.Conflicts) > 0 {
		prims = append(prims, fw.PrimBurst("burst-conflict", "cb-conflicts", "danger",
			int64(st.Tick), 800))
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "网络分区错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"mode":            st.Mode,
		"partitioned":     st.Partitioned,
		"node_count":      len(st.Nodes),
		"active_nodes":    activeCount(st.Nodes),
		"majority":        st.majority(),
		"global_version":  st.GlobalVersion,
		"dropped_count":   st.DroppedCount,
		"conflicts_count": len(st.Conflicts),
		"tick":            st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendPartitionMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "pn-1", Label: "节点划入 P1 / P2", DurationMs: 400, HighlightIDs: []string{"topology"}},
		{ID: "pn-2", Label: "跨分区消息丢弃", DurationMs: 400, HighlightIDs: []string{"zone-1", "zone-2"}, FirePrimitives: []string{"pulse-partition"}},
		{ID: "pn-3", Label: "等待写入 / 恢复 demonstrate CAP 取舍", DurationMs: 400, HighlightIDs: []string{"formula-cap"}, IsLinkTrigger: true},
	}
}

func appendWriteMicroSteps(env *fw.RenderEnvelope, ok bool, mode string) {
	tail := "写入失败（CP 拒绝少数派）"
	if ok {
		tail = "写入成功，复制到分区内副本"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "wr-1", Label: "客户端发起写入", DurationMs: 400, HighlightIDs: []string{"cb-status"}},
		{ID: "wr-2", Label: "检查所在分区可达副本数", DurationMs: 400, HighlightIDs: []string{"formula-cp", "cb-status"}},
		{ID: "wr-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-kv"}, IsLinkTrigger: true},
	}
}

func appendRecoverMicroSteps(env *fw.RenderEnvelope, conflicts int) {
	tail := "无冲突，状态已统一"
	if conflicts > 0 {
		tail = fmt.Sprintf("⚠ %d 个冲突按 LWW 解决", conflicts)
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "rc-1", Label: "收集所有分区的 KV 版本", DurationMs: 500, HighlightIDs: []string{"cb-kv"}},
		{ID: "rc-2", Label: "按 (tick, version, writer) 排序选 winner", DurationMs: 500, HighlightIDs: []string{"cb-conflicts"}},
		{ID: "rc-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-status", "cb-kv"}, FirePrimitives: []string{"burst-conflict"}, IsLinkTrigger: true},
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
		ID:             "partition-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_partition_state",
		LinkGroup:      linkGroupNetworkBase,
		ChangedFields:  []string{"network.partition.partitioned"},
		Payload:        map[string]any{"partitioned": st.Partitioned},
		SourceAnchorID: "partition-output-anchor",
		TargetAnchorID: "network-base-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.partition.partitioned")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"network": map[string]any{
			"partition": map[string]any{
				"mode":            st.Mode,
				"partitioned":     st.Partitioned,
				"node_count":      len(st.Nodes),
				"global_version":  st.GlobalVersion,
				"dropped_count":   st.DroppedCount,
				"conflicts_count": len(st.Conflicts),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

