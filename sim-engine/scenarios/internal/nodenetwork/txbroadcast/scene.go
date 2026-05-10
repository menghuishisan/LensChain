// 模块：sim-engine/scenarios/internal/nodenetwork/txbroadcast
// 文件职责：NET-04 交易广播（mempool 泛洪 + 去重）场景的完整实现。
//
// SSOT 依据：06.md §4.1.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现交易广播泛洪协议（比特币 / 以太坊风格 INV/GETDATA 简化版）：
//   · N 节点，每节点维护 mempool（tx_id → mempoolEntry）
//   · 客户端 submit_tx 在某节点 → 加入 mempool 并标记需广播
//   · 每 tick：每节点把"未广播给 peer P 过的 tx"广播给所有 peer P
//   · 接收方先 dedupe（mempool 已有则丢弃，统计 duplicate）；否则纳入 mempool
//   · mempool 容量上限：达到时驱逐 gas_price 最低的 tx
//   · spam_attack：注入 N 个 0-gas tx 演示 mempool 洪水
//   · 统计：每 tx 的 broadcast_delay = (max(收到 tick) - submit_tick)

package txbroadcast

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "tx-broadcast"
	schemaVersion = "v1.0.0"
	algorithmType = "tx-flood"

	defaultNodeCount    = 8
	defaultMempoolLimit = 32
	maxNodeCount        = 16
	maxTxHistory        = 64

	linkGroupNetworkBase = "network-base-group"
	linkGroupTxProc      = "tx-processing-group"
	linkOwnerSubtree     = "network.tx_broadcast"
)

type tx struct {
	ID         string
	Submitter  string
	GasPrice   int
	SubmitTick int
}

type mempoolEntry struct {
	Tx           tx
	ReceivedTick int
}

type bnode struct {
	ID      string
	IsDown  bool
	Mempool map[string]mempoolEntry
	// "已广播给 peer P 的 tx 集合"（用于避免重复广播）
	Sent map[string]map[string]bool // peer_id → tx_id → true
}

type tickEvent struct {
	Tick   int
	Sender string
	Recv   string
	TxID   string
	Dup    bool
}

type snapState struct {
	Nodes        []bnode
	MempoolLimit int
	Tick         int
	TotalTx      int  // 已 submit 的总 tx 数
	TotalMsgs    int  // 总广播消息数
	DupMsgs      int  // 重复消息数
	TxHistory    []tx // 最近的 tx 列表
	Events       []tickEvent
	LastError    string
}

func defaultSnapState() snapState {
	st := snapState{MempoolLimit: defaultMempoolLimit}
	for i := 0; i < defaultNodeCount; i++ {
		st.Nodes = append(st.Nodes, bnode{
			ID:      fmt.Sprintf("n%d", i),
			Mempool: map[string]mempoolEntry{},
			Sent:    map[string]map[string]bool{},
		})
	}
	return st
}

func activeCount(nodes []bnode) int {
	n := 0
	for _, x := range nodes {
		if !x.IsDown {
			n++
		}
	}
	return n
}

// submitTx 在节点 nid 提交一条 tx，加入其 mempool 并标记 broadcast。
func (st *snapState) submitTx(nid string, gasPrice int) (tx, error) {
	idx := -1
	for i, n := range st.Nodes {
		if n.ID == nid {
			idx = i
			break
		}
	}
	if idx < 0 {
		return tx{}, fmt.Errorf("未找到节点: %s", nid)
	}
	if st.Nodes[idx].IsDown {
		return tx{}, fmt.Errorf("%s 已离线", nid)
	}
	st.TotalTx++
	t := tx{
		ID:         fmt.Sprintf("tx%d", st.TotalTx),
		Submitter:  nid,
		GasPrice:   gasPrice,
		SubmitTick: st.Tick,
	}
	st.addToMempool(idx, t, st.Tick)
	st.TxHistory = append(st.TxHistory, t)
	if len(st.TxHistory) > maxTxHistory {
		st.TxHistory = st.TxHistory[len(st.TxHistory)-maxTxHistory:]
	}
	return t, nil
}

// addToMempool 添加 tx，必要时驱逐 gas 最低的。
func (st *snapState) addToMempool(idx int, t tx, tick int) {
	mp := st.Nodes[idx].Mempool
	if len(mp) >= st.MempoolLimit {
		// 驱逐 gas_price 最低
		var lowID string
		var lowGas = 1 << 30
		for id, e := range mp {
			if e.Tx.GasPrice < lowGas {
				lowGas = e.Tx.GasPrice
				lowID = id
			}
		}
		// 新 tx 比最低 gas 还低则丢弃自身
		if t.GasPrice <= lowGas {
			return
		}
		delete(mp, lowID)
	}
	mp[t.ID] = mempoolEntry{Tx: t, ReceivedTick: tick}
}

// broadcastStep 每 tick 推进：每个节点把"对每个 peer 都未发送过的 tx"广播给该 peer。
func (st *snapState) broadcastStep() {
	st.Tick++
	for i := range st.Nodes {
		s := &st.Nodes[i]
		if s.IsDown {
			continue
		}
		for j := range st.Nodes {
			if i == j {
				continue
			}
			r := &st.Nodes[j]
			if r.IsDown {
				continue
			}
			if s.Sent[r.ID] == nil {
				s.Sent[r.ID] = map[string]bool{}
			}
			// 遍历自己 mempool 中尚未发给 r 的 tx
			toSend := []tx{}
			for id, e := range s.Mempool {
				if !s.Sent[r.ID][id] {
					toSend = append(toSend, e.Tx)
				}
			}
			// 按 GasPrice 降序广播（教学：高 gas 优先）
			sort.Slice(toSend, func(a, b int) bool { return toSend[a].GasPrice > toSend[b].GasPrice })
			for _, t := range toSend {
				st.TotalMsgs++
				dup := false
				if _, has := r.Mempool[t.ID]; has {
					dup = true
					st.DupMsgs++
				} else {
					st.addToMempool(j, t, st.Tick)
					if r.Sent[s.ID] == nil {
						r.Sent[s.ID] = map[string]bool{}
					}
					// r 收到 t 后已知 s 也有；mark Sent 避免回广播
					r.Sent[s.ID][t.ID] = true
				}
				s.Sent[r.ID][t.ID] = true
				st.recordEvent(s.ID, r.ID, t.ID, dup)
			}
		}
	}
}

func (st *snapState) recordEvent(sender, recv, txID string, dup bool) {
	st.Events = append(st.Events, tickEvent{
		Tick: st.Tick, Sender: sender, Recv: recv, TxID: txID, Dup: dup,
	})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

// coverage 返回 tx 在网络中的覆盖率（活跃节点中已知比例）+ broadcast_delay。
func (st snapState) coverage(txID string, submitTick int) (float64, int, int, int) {
	active := 0
	has := 0
	maxTick := 0
	for _, n := range st.Nodes {
		if n.IsDown {
			continue
		}
		active++
		if e, ok := n.Mempool[txID]; ok {
			has++
			if e.ReceivedTick > maxTick {
				maxTick = e.ReceivedTick
			}
		}
	}
	delay := -1
	if has == active && active > 0 {
		delay = maxTick - submitTick
	}
	if active == 0 {
		return 0, 0, 0, -1
	}
	return float64(has) / float64(active), has, active, delay
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
		MempoolLimit: fw.MapInt(d, "mempool_limit", defaultMempoolLimit),
		Tick:         fw.MapInt(d, "tick", 0),
		TotalTx:      fw.MapInt(d, "total_tx", 0),
		TotalMsgs:    fw.MapInt(d, "total_msgs", 0),
		DupMsgs:      fw.MapInt(d, "dup_msgs", 0),
		LastError:    fw.MapStr(d, "last_error", ""),
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				n := bnode{
					ID:      fw.MapStr(nm, "id", ""),
					IsDown:  fw.MapBool(nm, "down", false),
					Mempool: map[string]mempoolEntry{},
					Sent:    map[string]map[string]bool{},
				}
				if mpAny, ok := nm["mempool"].(map[string]any); ok {
					for k, v := range mpAny {
						if vm, ok := v.(map[string]any); ok {
							n.Mempool[k] = mempoolEntry{
								Tx: tx{
									ID:         k,
									Submitter:  fw.MapStr(vm, "submitter", ""),
									GasPrice:   fw.MapInt(vm, "gas", 0),
									SubmitTick: fw.MapInt(vm, "submit_tick", 0),
								},
								ReceivedTick: fw.MapInt(vm, "recv_tick", 0),
							}
						}
					}
				}
				if sntAny, ok := nm["sent"].(map[string]any); ok {
					for peer, txs := range sntAny {
						n.Sent[peer] = map[string]bool{}
						if listAny, ok := txs.([]any); ok {
							for _, x := range listAny {
								if s, ok := x.(string); ok {
									n.Sent[peer][s] = true
								}
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
	if histAny, ok := d["tx_history"].([]any); ok {
		for _, hAny := range histAny {
			if hm, ok := hAny.(map[string]any); ok {
				st.TxHistory = append(st.TxHistory, tx{
					ID:         fw.MapStr(hm, "id", ""),
					Submitter:  fw.MapStr(hm, "submitter", ""),
					GasPrice:   fw.MapInt(hm, "gas", 0),
					SubmitTick: fw.MapInt(hm, "submit_tick", 0),
				})
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, evAny := range eAny {
			if em, ok := evAny.(map[string]any); ok {
				st.Events = append(st.Events, tickEvent{
					Tick: fw.MapInt(em, "tick", 0), Sender: fw.MapStr(em, "sender", ""),
					Recv: fw.MapStr(em, "recv", ""), TxID: fw.MapStr(em, "tx", ""),
					Dup: fw.MapBool(em, "dup", false),
				})
			}
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["mempool_limit"] = st.MempoolLimit
	s.Data["tick"] = st.Tick
	s.Data["total_tx"] = st.TotalTx
	s.Data["total_msgs"] = st.TotalMsgs
	s.Data["dup_msgs"] = st.DupMsgs
	s.Data["last_error"] = st.LastError
	nodesAny := make([]any, len(st.Nodes))
	for i, n := range st.Nodes {
		mp := map[string]any{}
		for k, e := range n.Mempool {
			mp[k] = map[string]any{
				"submitter":   e.Tx.Submitter,
				"gas":         e.Tx.GasPrice,
				"submit_tick": e.Tx.SubmitTick,
				"recv_tick":   e.ReceivedTick,
			}
		}
		sent := map[string]any{}
		for peer, txs := range n.Sent {
			list := []any{}
			for k := range txs {
				list = append(list, k)
			}
			sent[peer] = list
		}
		nodesAny[i] = map[string]any{
			"id": n.ID, "down": n.IsDown, "mempool": mp, "sent": sent,
		}
	}
	s.Data["nodes"] = nodesAny
	histAny := make([]any, len(st.TxHistory))
	for i, t := range st.TxHistory {
		histAny[i] = map[string]any{
			"id": t.ID, "submitter": t.Submitter, "gas": t.GasPrice, "submit_tick": t.SubmitTick,
		}
	}
	s.Data["tx_history"] = histAny
	eAny := make([]any, len(st.Events))
	for i, e := range st.Events {
		eAny[i] = map[string]any{
			"tick": e.Tick, "sender": e.Sender, "recv": e.Recv, "tx": e.TxID, "dup": e.Dup,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "交易广播（mempool 泛洪）",
		Description:         "演示比特币风格 INV/GETDATA 泛洪 + mempool 去重 + gas 优先 + spam 攻击",
		Category:            fw.CategoryNodeNetwork,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupNetworkBase, linkGroupTxProc},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"network.tx_broadcast.total_tx",
			"network.tx_broadcast.tick",
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
				ActionCode: "set_params", Label: "设置广播参数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "mempool_limit", Type: fw.FieldNumber, Label: "mempool 容量", Required: true,
						Default: defaultMempoolLimit, Min: 4, Max: 256, Step: 4},
				},
			},
			{
				ActionCode: "submit_tx", Label: "提交交易",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "提交节点", Required: true, Default: "n0"},
					{Name: "gas_price", Type: fw.FieldNumber, Label: "Gas Price", Required: true, Default: 100, Min: 0, Step: 1},
				},
				WritesOwnedFields: []string{"network.tx_broadcast.total_tx"},
				LinkOwnerFields:   []string{"network.tx_broadcast.total_tx"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"network.tx_broadcast.tick"},
				LinkOwnerFields:   []string{"network.tx_broadcast.tick"},
			},
			{
				ActionCode: "step_n_ticks", Label: "推进 N tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 5, Min: 1, Max: 50, Step: 1},
				},
			},
			{
				ActionCode: "spam_attack", Label: "Spam 攻击（注入垃圾交易）",
				Description: "在指定节点上一次性提交 N 个 0-gas tx",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n0"},
					{Name: "count", Type: fw.FieldNumber, Label: "垃圾 tx 数", Required: true, Default: 16, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "crash_node", Label: "节点离线",
				Category: fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:  []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n2"}},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
	env := buildEnvelope(st, "init", "Tx Broadcast 初始化（8 节点，mempool=32）", true)
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
		st.MempoolLimit = fw.MapInt(in.Params, "mempool_limit", defaultMempoolLimit)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", fmt.Sprintf("mempool_limit=%d", st.MempoolLimit), false)
		return out, nil

	case "submit_tx":
		nid := fw.MapStr(in.Params, "node_id", "n0")
		gas := fw.MapInt(in.Params, "gas_price", 100)
		t, err := st.submitTx(nid, gas)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "submit_tx", fmt.Sprintf("%s 提交 %s (gas=%d)", nid, t.ID, gas), false)
		appendSubmitMicroSteps(&out.Render, nid, t.ID)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_tick":
		st.broadcastStep()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_tick", fmt.Sprintf("tick=%d total_msgs=%d dup=%d", st.Tick, st.TotalMsgs, st.DupMsgs), false)
		appendStepMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.broadcastStep()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n_ticks",
			fmt.Sprintf("推进 %d tick → msgs=%d dup_rate=%.0f%%", n, st.TotalMsgs, dupRate(st)*100), false)
		appendStepMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "spam_attack":
		nid := fw.MapStr(in.Params, "node_id", "n0")
		cnt := fw.MapInt(in.Params, "count", 16)
		spawned := 0
		for i := 0; i < cnt; i++ {
			if _, err := st.submitTx(nid, 0); err == nil {
				spawned++
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "spam_attack",
			fmt.Sprintf("%s 提交 %d 笔垃圾 tx（gas=0）", nid, spawned), false)
		appendSpamMicroSteps(&out.Render, nid, spawned)
		return out, nil

	case "crash_node":
		nid := fw.MapStr(in.Params, "node_id", "n2")
		for i := range st.Nodes {
			if st.Nodes[i].ID == nid {
				st.Nodes[i].IsDown = true
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

func dupRate(st snapState) float64 {
	if st.TotalMsgs == 0 {
		return 0
	}
	return float64(st.DupMsgs) / float64(st.TotalMsgs)
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 30)

	// 1) graph_layout
	nodeIDs := []string{}
	for _, n := range st.Nodes {
		nodeIDs = append(nodeIDs, "node-"+n.ID)
	}
	prims = append(prims, fw.PrimGraphLayout("topology", "force", nodeIDs, nil))

	// 2) 节点
	activeSet := map[string]bool{}
	for _, e := range st.Events {
		if e.Tick == st.Tick {
			activeSet[e.Sender] = true
			activeSet[e.Recv] = true
		}
	}
	for _, n := range st.Nodes {
		status := "normal"
		role := "node"
		if n.IsDown {
			status = "error"
			role = "down"
		} else if activeSet[n.ID] {
			status = "active"
			role = "broadcasting"
		}
		// mempool 占用率
		fillPct := 100 * len(n.Mempool) / st.MempoolLimit
		label := fmt.Sprintf("%s\nmempool=%d/%d (%d%%)", n.ID, len(n.Mempool), st.MempoolLimit, fillPct)
		prims = append(prims, fw.PrimNode("node-"+n.ID, label, status, role))
	}

	// 3) 当前 tick 消息流
	for _, e := range st.Events {
		if e.Tick != st.Tick {
			continue
		}
		style := "solid"
		anim := "flow"
		if e.Dup {
			style = "dashed"
			anim = ""
		}
		eid := fmt.Sprintf("evt-%d-%s-%s-%s", e.Tick, e.Sender, e.Recv, e.TxID)
		prims = append(prims, fw.PrimEdge(eid, "node-"+e.Sender, "node-"+e.Recv, style, anim))
	}

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-flood",
		`\text{每 tick: }\forall (s, r),\ s\to r:\ \mathrm{Mempool}_s \setminus \mathrm{Sent}_{s,r}`, false))

	// 5) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d\nN = %d (active=%d)\nmempool_limit = %d\ntotal_tx = %d\ntotal_msgs = %d\nduplicate = %d (%.0f%%)",
			st.Tick, len(st.Nodes), activeCount(st.Nodes), st.MempoolLimit,
			st.TotalTx, st.TotalMsgs, st.DupMsgs, dupRate(st)*100),
		"text", nil, 8))

	// 6) 进度条：dup_rate / mempool 平均占用
	prims = append(prims, fw.PrimProgressBar("dup-rate", dupRate(st)*100, 100,
		fmt.Sprintf("重复率 %.0f%%", dupRate(st)*100)))

	// 7) tx 列表 + 覆盖率 + 延迟
	if len(st.TxHistory) > 0 {
		txLines := []string{"tx_id  submitter  gas  submit_t  cov     delay"}
		startIdx := 0
		if len(st.TxHistory) > 16 {
			startIdx = len(st.TxHistory) - 16
		}
		for _, t := range st.TxHistory[startIdx:] {
			c, has, total, delay := st.coverage(t.ID, t.SubmitTick)
			delayStr := "..."
			if delay >= 0 {
				delayStr = fmt.Sprintf("%d", delay)
			}
			txLines = append(txLines, fmt.Sprintf("%-5s  %-8s  %-3d  %-7d   %d/%d (%.0f%%) %s",
				t.ID, t.Submitter, t.GasPrice, t.SubmitTick, has, total, c*100, delayStr))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-tx", strings.Join(txLines, "\n"), "text", nil, 16))
	}

	// 8) heat_map mempool 占用（行=节点，列=最近 16 个 tx）
	if len(st.TxHistory) > 0 {
		txCols := st.TxHistory
		if len(txCols) > 16 {
			txCols = txCols[len(txCols)-16:]
		}
		cells := make([]map[string]any, 0, len(st.Nodes)*len(txCols))
		for i, n := range st.Nodes {
			for j, t := range txCols {
				val := 0
				color := "muted"
				if _, has := n.Mempool[t.ID]; has {
					val = 1
					color = "info"
				}
				if n.IsDown {
					color = "danger"
				}
				cells = append(cells, map[string]any{
					"row": i, "col": j, "value": val, "color_role": color,
				})
			}
		}
		prims = append(prims, fw.PrimHeatMap("mempool-heatmap", len(st.Nodes), len(txCols), cells))
	}

	// 9) 事件
	if len(st.Events) > 0 {
		evLines := []string{"消息流（最近 16）："}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		evs := append([]tickEvent{}, st.Events[startIdx:]...)
		sort.Slice(evs, func(i, j int) bool { return evs[i].Tick < evs[j].Tick })
		for _, e := range evs {
			tag := ""
			if e.Dup {
				tag = " [DUP]"
			}
			evLines = append(evLines, fmt.Sprintf("  t=%d  %s → %s  %s%s",
				e.Tick, e.Sender, e.Recv, e.TxID, tag))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(evLines, "\n"), "text", nil, 16))
	}

	// 10) 动效
	for _, n := range st.Nodes {
		if n.IsDown {
			prims = append(prims, fw.PrimGlow("glow-down-"+n.ID, "node-"+n.ID, "danger", 0.7))
		} else if activeSet[n.ID] {
			prims = append(prims, fw.PrimPulse("pulse-active-"+n.ID, "node-"+n.ID, "info", 1200))
		}
	}

	// 11) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "广播错误", st.LastError, "scene", "请检查参数", true))
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
		"tick":          st.Tick,
		"node_count":    len(st.Nodes),
		"active_nodes":  activeCount(st.Nodes),
		"mempool_limit": st.MempoolLimit,
		"total_tx":      st.TotalTx,
		"total_msgs":    st.TotalMsgs,
		"dup_msgs":      st.DupMsgs,
		"dup_rate":      dupRate(st),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendSubmitMicroSteps(env *fw.RenderEnvelope, nid, txID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sb-1", Label: nid + " 创建 " + txID, DurationMs: 400, HighlightIDs: []string{"node-" + nid}, FirePrimitives: []string{"pulse-active-" + nid}},
		{ID: "sb-2", Label: "写入本地 mempool", DurationMs: 400, HighlightIDs: []string{"mempool-heatmap"}},
		{ID: "sb-3", Label: "标记需广播给所有 peer", DurationMs: 400, HighlightIDs: []string{"cb-events", "formula-flood"}, IsLinkTrigger: true},
	}
}

func appendStepMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "st-1", Label: "每节点对每 peer 广播 mempool \\ Sent", DurationMs: 400, HighlightIDs: []string{"formula-flood"}},
		{ID: "st-2", Label: "接收方 dedupe 后纳入 mempool", DurationMs: 500, HighlightIDs: []string{"cb-events", "mempool-heatmap"}},
		{ID: "st-3", Label: "更新 dup_rate / 覆盖率统计", DurationMs: 400, HighlightIDs: []string{"dup-rate", "cb-tx"}, IsLinkTrigger: true},
	}
}

func appendSpamMicroSteps(env *fw.RenderEnvelope, nid string, count int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sp-1", Label: fmt.Sprintf("攻击者从 %s 注入 %d 笔 0-gas tx", nid, count), DurationMs: 500, HighlightIDs: []string{"node-" + nid}, FirePrimitives: []string{"pulse-active-" + nid}},
		{ID: "sp-2", Label: "mempool 容量耗尽 → 驱逐 + 拒绝", DurationMs: 500, HighlightIDs: []string{"mempool-heatmap", "cb-status"}},
		{ID: "sp-3", Label: "正常 tx 被边缘化 / 延迟", DurationMs: 500, HighlightIDs: []string{"cb-tx", "dup-rate"}, IsLinkTrigger: true},
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
		ID:             "txbroadcast-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_mempool",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"network.tx_broadcast.total_tx"},
		Payload:        map[string]any{"total_tx": st.TotalTx},
		SourceAnchorID: "txbroadcast-output-anchor",
		TargetAnchorID: "tx-mempool-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.tx_broadcast.total_tx")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"network": map[string]any{
			"tx_broadcast": map[string]any{
				"tick":          st.Tick,
				"node_count":    len(st.Nodes),
				"total_tx":      st.TotalTx,
				"total_msgs":    st.TotalMsgs,
				"dup_msgs":      st.DupMsgs,
				"dup_rate":      dupRate(st),
				"mempool_limit": st.MempoolLimit,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

