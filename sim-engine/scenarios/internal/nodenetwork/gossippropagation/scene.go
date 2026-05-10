// 模块：sim-engine/scenarios/internal/nodenetwork/gossippropagation
// 文件职责：NET-02 Gossip 消息传播场景的完整实现。
//
// SSOT 依据：06.md §4.1.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现 epidemic-style gossip 协议三种模式（Demers et al. 1987）；
// 复用 sha256hash.Sum256 派生确定性"伪随机" peer 选择，零外部网络库：
//
//   · push：每 tick，每个有消息 m 的节点随机选 fanout 个 peer，对方还没有 m 时复制过去；
//   · pull：每 tick，每个节点随机选 fanout 个 peer，把对方有而自己没有的消息拉回；
//   · push-pull：每 tick 同时执行 push 和 pull（最快收敛，带宽 2 倍）；
//   · 节点离线：跳过该节点的发送 / 接收；
//   · 网络丢包率：每条消息按概率丢弃（教学：用确定性 hash 代替）。
//
// 教学决策：
//   - graph_layout 力导向展示节点拓扑
//   - 节点 status：normal / active=已收到当前消息 / error=离线
//   - heat_map 可视化每节点已知消息集合（行=节点，列=消息）
//   - 收敛进度条（覆盖率随 tick 增长）

package gossippropagation

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "gossip-propagation"
	schemaVersion = "v1.0.0"
	algorithmType = "gossip-epidemic"

	defaultNodeCount = 10
	maxNodeCount     = 20
	defaultFanout    = 3
	maxMessages      = 8
	defaultLossPctBp = 0 // 默认 0% 丢包

	modePush     = "push"
	modePull     = "pull"
	modePushPull = "push-pull"

	linkGroupNetworkBase = "network-base-group"
	linkOwnerSubtree     = "network.gossip"
)

// =====================================================================
// 节点 / 消息
// =====================================================================

type gnode struct {
	ID     string
	Known  map[string]int // message_id → first_received_tick
	IsDown bool
}

type message struct {
	ID         string
	Origin     string
	OriginTick int
	Content    string
}

type tickEvent struct {
	Tick   int
	Sender string
	Recv   string
	MsgID  string
	Mode   string
	Lost   bool
}

type snapState struct {
	Nodes       []gnode
	Messages    []message
	Mode        string
	Fanout      int
	LossPctBp   int // 10000 = 100%
	Tick        int
	Events      []tickEvent
	ConvergedAt map[string]int // 每条消息覆盖 100% 时的 tick；未收敛 = -1
	LastError   string
}

func defaultSnapState() snapState {
	st := snapState{
		Mode:        modePushPull,
		Fanout:      defaultFanout,
		LossPctBp:   defaultLossPctBp,
		ConvergedAt: map[string]int{},
	}
	for i := 0; i < defaultNodeCount; i++ {
		st.Nodes = append(st.Nodes, gnode{
			ID:    fmt.Sprintf("n%d", i),
			Known: map[string]int{},
		})
	}
	return st
}

// publishMessage 节点 originID 在 tick 创建消息 m。
func (st *snapState) publishMessage(originID, content string) (message, error) {
	if len(st.Messages) >= maxMessages {
		return message{}, fmt.Errorf("消息数已达上限 %d", maxMessages)
	}
	idx := -1
	for i, n := range st.Nodes {
		if n.ID == originID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return message{}, fmt.Errorf("未找到节点: %s", originID)
	}
	if st.Nodes[idx].IsDown {
		return message{}, fmt.Errorf("%s 已离线，不能 publish", originID)
	}
	mid := fmt.Sprintf("m%d", len(st.Messages))
	m := message{ID: mid, Origin: originID, OriginTick: st.Tick, Content: content}
	st.Messages = append(st.Messages, m)
	st.Nodes[idx].Known[mid] = st.Tick
	st.ConvergedAt[mid] = -1
	return m, nil
}

// pickPeers 给定节点 i 在 tick t 下的 fanout 个 peer 索引（确定性伪随机）。
// 不选自己 / 不选 down 节点；用 SHA-256(tick || nodeID || nonce) 派生序列。
func (st snapState) pickPeers(selfIdx int, tickAdj int) []int {
	N := len(st.Nodes)
	if N <= 1 {
		return nil
	}
	// 候选索引
	cands := []int{}
	for j := 0; j < N; j++ {
		if j == selfIdx || st.Nodes[j].IsDown {
			continue
		}
		cands = append(cands, j)
	}
	if len(cands) == 0 {
		return nil
	}
	picked := []int{}
	for nonce := 0; len(picked) < st.Fanout && len(picked) < len(cands); nonce++ {
		buf := make([]byte, 0, 24)
		buf = append(buf, []byte(st.Nodes[selfIdx].ID)...)
		var tickBytes [8]byte
		binary.BigEndian.PutUint64(tickBytes[:], uint64(tickAdj+nonce*1000))
		buf = append(buf, tickBytes[:]...)
		h := sha256hash.Sum256(buf)
		idx := int(binary.BigEndian.Uint32(h[:4])) % len(cands)
		if idx < 0 {
			idx += len(cands)
		}
		c := cands[idx]
		dup := false
		for _, p := range picked {
			if p == c {
				dup = true
				break
			}
		}
		if !dup {
			picked = append(picked, c)
		}
		if nonce > 100 {
			break
		}
	}
	return picked
}

// isLost 用确定性 hash 模拟丢包：hash(tick||sender||recv||msg) mod 10000 < LossPctBp 即丢。
func (st snapState) isLost(tickAdj int, sender, recv, msg string) bool {
	if st.LossPctBp <= 0 {
		return false
	}
	buf := []byte(sender + "|" + recv + "|" + msg)
	var tb [8]byte
	binary.BigEndian.PutUint64(tb[:], uint64(tickAdj))
	buf = append(buf, tb[:]...)
	h := sha256hash.Sum256(buf)
	r := int(binary.BigEndian.Uint32(h[:4])) % 10000
	if r < 0 {
		r += 10000
	}
	return r < st.LossPctBp
}

// stepGossip 推进 1 tick：按 Mode 执行所有节点的 push / pull / push-pull。
func (st *snapState) stepGossip() {
	st.Tick++
	N := len(st.Nodes)
	// 注意：所有节点同步更新（基于 step 开始时的 Known 快照）
	snapshot := make([]map[string]int, N)
	for i, n := range st.Nodes {
		snapshot[i] = make(map[string]int, len(n.Known))
		for k, v := range n.Known {
			snapshot[i][k] = v
		}
	}
	for i := range st.Nodes {
		if st.Nodes[i].IsDown {
			continue
		}
		peers := st.pickPeers(i, st.Tick*1000+i)
		for _, j := range peers {
			if st.Mode == modePush || st.Mode == modePushPull {
				// push: i 把自己有 j 没有的消息发给 j
				for mid := range snapshot[i] {
					if _, has := snapshot[j][mid]; has {
						continue
					}
					lost := st.isLost(st.Tick, st.Nodes[i].ID, st.Nodes[j].ID, mid)
					st.recordEvent(st.Nodes[i].ID, st.Nodes[j].ID, mid, modePush, lost)
					if !lost {
						if _, has := st.Nodes[j].Known[mid]; !has {
							st.Nodes[j].Known[mid] = st.Tick
						}
					}
				}
			}
			if st.Mode == modePull || st.Mode == modePushPull {
				// pull: i 把 j 有自己没有的消息拉回
				for mid := range snapshot[j] {
					if _, has := snapshot[i][mid]; has {
						continue
					}
					lost := st.isLost(st.Tick, st.Nodes[j].ID, st.Nodes[i].ID, mid)
					st.recordEvent(st.Nodes[j].ID, st.Nodes[i].ID, mid, modePull, lost)
					if !lost {
						if _, has := st.Nodes[i].Known[mid]; !has {
							st.Nodes[i].Known[mid] = st.Tick
						}
					}
				}
			}
		}
	}
	// 检查每条消息是否覆盖 100%（仅活跃节点）
	for _, m := range st.Messages {
		if st.ConvergedAt[m.ID] >= 0 {
			continue
		}
		all := true
		active := 0
		hasIt := 0
		for _, n := range st.Nodes {
			if n.IsDown {
				continue
			}
			active++
			if _, ok := n.Known[m.ID]; ok {
				hasIt++
			} else {
				all = false
			}
		}
		_ = active
		_ = hasIt
		if all && active > 0 {
			st.ConvergedAt[m.ID] = st.Tick
		}
	}
}

func (st *snapState) recordEvent(sender, recv, mid, mode string, lost bool) {
	st.Events = append(st.Events, tickEvent{
		Tick: st.Tick, Sender: sender, Recv: recv, MsgID: mid, Mode: mode, Lost: lost,
	})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

// coverage 计算给定消息的当前覆盖率（活跃节点中已知比例）。
func (st snapState) coverage(mid string) (float64, int, int) {
	active := 0
	has := 0
	for _, n := range st.Nodes {
		if n.IsDown {
			continue
		}
		active++
		if _, ok := n.Known[mid]; ok {
			has++
		}
	}
	if active == 0 {
		return 0, 0, 0
	}
	return float64(has) / float64(active), has, active
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
		Mode:        fw.MapStr(d, "mode", modePushPull),
		Fanout:      fw.MapInt(d, "fanout", defaultFanout),
		LossPctBp:   fw.MapInt(d, "loss_pct_bp", defaultLossPctBp),
		Tick:        fw.MapInt(d, "tick", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
		ConvergedAt: map[string]int{},
	}
	if nodesAny, ok := d["nodes"].([]any); ok {
		for _, nAny := range nodesAny {
			if nm, ok := nAny.(map[string]any); ok {
				n := gnode{
					ID:     fw.MapStr(nm, "id", ""),
					IsDown: fw.MapBool(nm, "down", false),
					Known:  map[string]int{},
				}
				if kAny, ok := nm["known"].(map[string]any); ok {
					for k, v := range kAny {
						n.Known[k] = intFromAny(v)
					}
				}
				st.Nodes = append(st.Nodes, n)
			}
		}
	}
	if len(st.Nodes) == 0 {
		return defaultSnapState()
	}
	if msgsAny, ok := d["messages"].([]any); ok {
		for _, mAny := range msgsAny {
			if mm, ok := mAny.(map[string]any); ok {
				st.Messages = append(st.Messages, message{
					ID:         fw.MapStr(mm, "id", ""),
					Origin:     fw.MapStr(mm, "origin", ""),
					OriginTick: fw.MapInt(mm, "origin_tick", 0),
					Content:    fw.MapStr(mm, "content", ""),
				})
			}
		}
	}
	if cAny, ok := d["converged_at"].(map[string]any); ok {
		for k, v := range cAny {
			st.ConvergedAt[k] = intFromAny(v)
		}
	}
	if evAny, ok := d["events"].([]any); ok {
		for _, eAny := range evAny {
			if em, ok := eAny.(map[string]any); ok {
				st.Events = append(st.Events, tickEvent{
					Tick:   fw.MapInt(em, "tick", 0),
					Sender: fw.MapStr(em, "sender", ""),
					Recv:   fw.MapStr(em, "recv", ""),
					MsgID:  fw.MapStr(em, "mid", ""),
					Mode:   fw.MapStr(em, "mode", ""),
					Lost:   fw.MapBool(em, "lost", false),
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
	s.Data["mode"] = st.Mode
	s.Data["fanout"] = st.Fanout
	s.Data["loss_pct_bp"] = st.LossPctBp
	s.Data["tick"] = st.Tick
	s.Data["last_error"] = st.LastError
	nodesAny := make([]any, len(st.Nodes))
	for i, n := range st.Nodes {
		known := map[string]any{}
		for k, v := range n.Known {
			known[k] = v
		}
		nodesAny[i] = map[string]any{
			"id":    n.ID,
			"down":  n.IsDown,
			"known": known,
		}
	}
	s.Data["nodes"] = nodesAny
	msgsAny := make([]any, len(st.Messages))
	for i, m := range st.Messages {
		msgsAny[i] = map[string]any{
			"id":          m.ID,
			"origin":      m.Origin,
			"origin_tick": m.OriginTick,
			"content":     m.Content,
		}
	}
	s.Data["messages"] = msgsAny
	cAny := map[string]any{}
	for k, v := range st.ConvergedAt {
		cAny[k] = v
	}
	s.Data["converged_at"] = cAny
	evAny := make([]any, len(st.Events))
	for i, e := range st.Events {
		evAny[i] = map[string]any{
			"tick": e.Tick, "sender": e.Sender, "recv": e.Recv,
			"mid": e.MsgID, "mode": e.Mode, "lost": e.Lost,
		}
	}
	s.Data["events"] = evAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "Gossip 消息传播",
		Description:         "演示 epidemic-style gossip 三种模式（push / pull / push-pull）+ 丢包 + 收敛统计",
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
			"network.gossip.tick",
			"network.gossip.coverage",
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
				ActionCode: "set_params", Label: "设置 Gossip 参数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "mode", Type: fw.FieldEnum, Label: "模式", Required: true, Default: modePushPull,
						Options: []any{modePush, modePull, modePushPull}},
					{Name: "fanout", Type: fw.FieldNumber, Label: "fanout（每节点选 N 个 peer）", Required: true, Default: defaultFanout, Min: 1, Max: 8, Step: 1},
					{Name: "loss_pct_bp", Type: fw.FieldNumber, Label: "丢包率（万分位）", Required: true, Default: defaultLossPctBp, Min: 0, Max: 5000, Step: 100},
				},
			},
			{
				ActionCode: "publish_message", Label: "节点发布消息",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "origin", Type: fw.FieldString, Label: "源节点", Required: true, Default: "n0"},
					{Name: "content", Type: fw.FieldString, Label: "消息内容", Required: true, Default: "hello"},
				},
				WritesOwnedFields: []string{"network.gossip.coverage"},
				LinkOwnerFields:   []string{"network.gossip.coverage"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"network.gossip.tick", "network.gossip.coverage"},
				LinkOwnerFields:   []string{"network.gossip.tick", "network.gossip.coverage"},
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
				ActionCode: "crash_node", Label: "节点离线",
				Category: fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n2"},
				},
			},
			{
				ActionCode: "recover_node", Label: "节点恢复",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "node_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "n2"},
				},
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
	env := buildEnvelope(st, "init", "Gossip 初始化（10 节点，push-pull 模式）", true)
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
		st.Mode = fw.MapStr(in.Params, "mode", modePushPull)
		st.Fanout = fw.MapInt(in.Params, "fanout", defaultFanout)
		st.LossPctBp = fw.MapInt(in.Params, "loss_pct_bp", defaultLossPctBp)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("mode=%s fanout=%d loss=%.1f%%", st.Mode, st.Fanout, float64(st.LossPctBp)/100), false)
		return out, nil

	case "publish_message":
		origin := fw.MapStr(in.Params, "origin", "n0")
		content := fw.MapStr(in.Params, "content", "hello")
		m, err := st.publishMessage(origin, content)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "publish_message",
			fmt.Sprintf("%s 发布消息 %s='%s'", origin, m.ID, content), false)
		appendPublishMicroSteps(&out.Render, origin, m.ID)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_tick":
		st.stepGossip()
		saveState(state, st)
		summary := fmt.Sprintf("tick=%d %s", st.Tick, briefCoverage(st))
		out.Render = buildEnvelope(st, "step_tick", summary, false)
		appendStepMicroSteps(&out.Render, st.Mode)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.stepGossip()
		}
		saveState(state, st)
		summary := fmt.Sprintf("推进 %d tick → %s", n, briefCoverage(st))
		out.Render = buildEnvelope(st, "step_n_ticks", summary, false)
		appendStepMicroSteps(&out.Render, st.Mode)
		out.SharedStateDiff = ownerDiff(st)
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
		out.Render = buildEnvelope(st, "crash_node", nid+" 已离线", false)
		return out, nil

	case "recover_node":
		nid := fw.MapStr(in.Params, "node_id", "n2")
		for i := range st.Nodes {
			if st.Nodes[i].ID == nid {
				st.Nodes[i].IsDown = false
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "recover_node", nid+" 恢复", false)
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

func briefCoverage(st snapState) string {
	if len(st.Messages) == 0 {
		return "无消息"
	}
	parts := []string{}
	for _, m := range st.Messages {
		c, has, total := st.coverage(m.ID)
		conv := ""
		if t, ok := st.ConvergedAt[m.ID]; ok && t >= 0 {
			conv = fmt.Sprintf(" (在 t=%d 收敛)", t)
		}
		parts = append(parts, fmt.Sprintf("%s: %d/%d (%.0f%%)%s", m.ID, has, total, c*100, conv))
	}
	return strings.Join(parts, "; ")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 40)

	// 1) graph_layout 拓扑
	nodeIDs := []string{}
	for _, n := range st.Nodes {
		nodeIDs = append(nodeIDs, "node-"+n.ID)
	}
	prims = append(prims, fw.PrimGraphLayout("topology", "force", nodeIDs, nil))

	// 2) 节点
	// 当前 tick 的事件中涉及的节点高亮
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
			role = "gossiping"
		}
		label := fmt.Sprintf("%s\nknown=%d", n.ID, len(n.Known))
		prims = append(prims, fw.PrimNode("node-"+n.ID, label, status, role))
	}

	// 3) 当前 tick 的消息流边
	for _, e := range st.Events {
		if e.Tick != st.Tick {
			continue
		}
		style := "solid"
		anim := "flow"
		if e.Lost {
			style = "dashed"
			anim = ""
		}
		eid := fmt.Sprintf("evt-%d-%s-%s-%s", e.Tick, e.Sender, e.Recv, e.MsgID)
		prims = append(prims, fw.PrimEdge(eid,
			"node-"+e.Sender, "node-"+e.Recv, style, anim))
	}

	// 4) heat_map：节点 × 消息 占有矩阵
	if len(st.Messages) > 0 {
		cells := make([]map[string]any, 0, len(st.Nodes)*len(st.Messages))
		for i, n := range st.Nodes {
			for j, m := range st.Messages {
				val := 0
				color := "muted"
				if _, has := n.Known[m.ID]; has {
					val = 1
					color = "success"
					if n.Known[m.ID] == st.Tick {
						color = "info" // 本 tick 新收到
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
		prims = append(prims, fw.PrimHeatMap("known-matrix", len(st.Nodes), len(st.Messages), cells))
	}

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-conv",
		`\text{push-pull 期望收敛}\ \mathcal{O}(\log_{1+f} N)`, false))

	// 6) 关键参数
	prims = append(prims, fw.PrimCodeBlock("cb-params",
		fmt.Sprintf("mode = %s\nfanout = %d\nloss = %.1f%%\ntick = %d\n节点 = %d (active=%d)\n消息 = %d",
			st.Mode, st.Fanout, float64(st.LossPctBp)/100, st.Tick, len(st.Nodes), countActive(st), len(st.Messages)),
		"text", nil, 6))

	// 7) 消息覆盖率
	covLines := []string{"消息覆盖率："}
	convergedCount := 0
	for _, m := range st.Messages {
		c, has, total := st.coverage(m.ID)
		conv := "未收敛"
		if t, ok := st.ConvergedAt[m.ID]; ok && t >= 0 {
			conv = fmt.Sprintf("收敛 t=%d (用时 %d tick)", t, t-m.OriginTick)
			convergedCount++
		}
		covLines = append(covLines, fmt.Sprintf("  %s [%s] %d/%d (%.0f%%)  %s",
			m.ID, m.Origin, has, total, c*100, conv))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-coverage", strings.Join(covLines, "\n"), "text", nil, 12))

	// 8) 进度条：所有消息的平均覆盖率
	if len(st.Messages) > 0 {
		totalCov := 0.0
		for _, m := range st.Messages {
			c, _, _ := st.coverage(m.ID)
			totalCov += c
		}
		avgCov := totalCov / float64(len(st.Messages))
		prims = append(prims, fw.PrimProgressBar("convergence-progress", avgCov*100, 100,
			fmt.Sprintf("平均覆盖率 %.0f%%", avgCov*100)))
	}

	// 9) 事件日志
	if len(st.Events) > 0 {
		evLines := []string{"消息流（最近 16 条）："}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		// 按 tick 升序展示
		evs := append([]tickEvent{}, st.Events[startIdx:]...)
		sort.Slice(evs, func(i, j int) bool { return evs[i].Tick < evs[j].Tick })
		for _, e := range evs {
			lost := ""
			if e.Lost {
				lost = " ✗LOST"
			}
			evLines = append(evLines, fmt.Sprintf("  t=%d  %s → %s  %s [%s]%s",
				e.Tick, e.Sender, e.Recv, e.MsgID, e.Mode, lost))
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

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Gossip 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	totalCov := 0.0
	for _, m := range st.Messages {
		c, _, _ := st.coverage(m.ID)
		totalCov += c
	}
	avgCov := 0.0
	if len(st.Messages) > 0 {
		avgCov = totalCov / float64(len(st.Messages))
	}
	convergedMsgs := []string{}
	for _, m := range st.Messages {
		if t, ok := st.ConvergedAt[m.ID]; ok && t >= 0 {
			convergedMsgs = append(convergedMsgs, m.ID)
		}
	}
	d := map[string]any{
		"mode":           st.Mode,
		"fanout":         st.Fanout,
		"loss_pct_bp":    st.LossPctBp,
		"tick":           st.Tick,
		"node_count":     len(st.Nodes),
		"active_nodes":   countActive(st),
		"message_count":  len(st.Messages),
		"avg_coverage":   avgCov,
		"converged_msgs": convergedMsgs,
		"event_count":    len(st.Events),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

func countActive(st snapState) int {
	n := 0
	for _, x := range st.Nodes {
		if !x.IsDown {
			n++
		}
	}
	return n
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendPublishMicroSteps(env *fw.RenderEnvelope, origin, mid string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "pb-1", Label: origin + " 创建新消息 " + mid, DurationMs: 400, HighlightIDs: []string{"node-" + origin, "cb-coverage"}, FirePrimitives: []string{"pulse-active-" + origin}},
		{ID: "pb-2", Label: "本节点 known 集合纳入 " + mid, DurationMs: 400, HighlightIDs: []string{"known-matrix"}},
		{ID: "pb-3", Label: "等待 step_tick 推进传播", DurationMs: 400, HighlightIDs: []string{"cb-params"}, IsLinkTrigger: true},
	}
}

func appendStepMicroSteps(env *fw.RenderEnvelope, mode string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "st-1", Label: "每个节点选 fanout 个 peer", DurationMs: 400, HighlightIDs: []string{"topology", "formula-conv"}},
		{ID: "st-2", Label: mode + ": 同步消息（含丢包）", DurationMs: 500, HighlightIDs: []string{"cb-events", "known-matrix"}},
		{ID: "st-3", Label: "更新 known 集合 + 检查覆盖率", DurationMs: 400, HighlightIDs: []string{"convergence-progress", "cb-coverage"}, IsLinkTrigger: true},
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
		ID:             "gossip-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_coverage",
		LinkGroup:      linkGroupNetworkBase,
		ChangedFields:  []string{"network.gossip.coverage"},
		Payload:        map[string]any{"tick": st.Tick},
		SourceAnchorID: "gossip-output-anchor",
		TargetAnchorID: "network-base-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.gossip.coverage")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	totalCov := 0.0
	for _, m := range st.Messages {
		c, _, _ := st.coverage(m.ID)
		totalCov += c
	}
	avgCov := 0.0
	if len(st.Messages) > 0 {
		avgCov = totalCov / float64(len(st.Messages))
	}
	return map[string]any{
		"network": map[string]any{
			"gossip": map[string]any{
				"mode":          st.Mode,
				"fanout":        st.Fanout,
				"tick":          st.Tick,
				"node_count":    len(st.Nodes),
				"message_count": len(st.Messages),
				"avg_coverage":  avgCov,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}
