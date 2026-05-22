// 模块：sim-engine/scenarios/internal/nodenetwork/nodeloadbalance
// 文件职责：NET-06 节点负载均衡场景的完整实现。
//
// SSOT 依据：06.md §4.1.6 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 4 种负载均衡策略（零外部依赖）：
//   · random：哈希(client_key||tick)取模选节点；
//   · round-robin：tick 轮转；
//   · least-loaded：选 in-flight 最小的节点（平局取 ID 字典序）；
//   · consistent-hash：每节点取 hash 后映射到 [0, 2^32) 环上；
//                       客户端 hash → 顺时针找最近节点（节点变动只影响 1/N）。
//
// 每 tick：
//   · 客户端按 RPS 提交请求，按策略路由
//   · 节点 in-flight 增 1
//   · 上一 tick 进入的请求若 in_flight ≤ capacity 则完成（in_flight 减 1, processed 增 1），
//     否则记入 rejected
//   · 节点离线：路由跳过；consistent-hash 重映射到下一节点

package nodeloadbalance

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "node-load-balance"
	schemaVersion = "v1.0.0"
	algorithmType = "load-balance"

	defaultBackendCount = 5
	maxBackendCount     = 12

	strategyRandom     = "random"
	strategyRR         = "round-robin"
	strategyLeast      = "least-loaded"
	strategyConsistent = "consistent-hash"

	linkGroupNetworkBase = "network-base-group"
	linkOwnerSubtree     = "network.load_balance"
)

type backend struct {
	ID             string
	Capacity       int
	InFlight       int
	TotalProcessed int
	TotalRejected  int
	IsDown         bool
	HashRingPos    uint32 // consistent-hash 环上位置
}

type pendingReq struct {
	ClientKey string
	BackendID string
	StartTick int
	Rejected  bool
}

type snapState struct {
	Backends  []backend
	Strategy  string
	RPS       int // 每 tick 入队的请求数
	Tick      int
	RRCounter int
	Pending   []pendingReq // 上一 tick 进入仍未完成的（教学：1 tick 处理时延）
	History   []pendingReq // 最近 32 个完成 / 拒绝
	LastError string
}

func defaultSnapState() snapState {
	st := snapState{
		Strategy: strategyLeast,
		RPS:      8,
	}
	for i := 0; i < defaultBackendCount; i++ {
		id := fmt.Sprintf("b%d", i)
		st.Backends = append(st.Backends, backend{
			ID: id, Capacity: 4,
			HashRingPos: hashKeyToRingPos(id),
		})
	}
	return st
}

// hashKeyToRingPos 用 SHA-256 前 4 字节大端序作为 32-bit 环上位置。
func hashKeyToRingPos(key string) uint32 {
	// 简化：直接用 FNV-like，但为了"零依赖且确定性"用 SHA-256
	// 这里复用 sha256（教学：与 consistent-hashing 论文 Karger et al. 1997 一致）
	// 但为了避免循环依赖，本场景独立计算 32-bit hash：
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h
}

// activeCount 未 down 的 backend 数。
func (st snapState) activeCount() int {
	n := 0
	for _, b := range st.Backends {
		if !b.IsDown {
			n++
		}
	}
	return n
}

// pickBackend 按 strategy 路由 client_key 到某个 backend；返回索引。
func (st snapState) pickBackend(clientKey string) int {
	N := len(st.Backends)
	if N == 0 || st.activeCount() == 0 {
		return -1
	}
	// 排除 down
	candidates := []int{}
	for i := range st.Backends {
		if !st.Backends[i].IsDown {
			candidates = append(candidates, i)
		}
	}
	switch st.Strategy {
	case strategyRandom:
		// 用 hash(client_key || tick) 选
		var buf []byte
		buf = append(buf, []byte(clientKey)...)
		var tb [8]byte
		binary.BigEndian.PutUint64(tb[:], uint64(st.Tick))
		buf = append(buf, tb[:]...)
		h := hashKeyToRingPos(string(buf))
		return candidates[int(h)%len(candidates)]

	case strategyRR:
		// 用 RRCounter mod 候选
		return candidates[st.RRCounter%len(candidates)]

	case strategyLeast:
		// 选 InFlight 最小的；平局取 ID 字典序
		best := candidates[0]
		for _, c := range candidates[1:] {
			a, b := st.Backends[best], st.Backends[c]
			if b.InFlight < a.InFlight || (b.InFlight == a.InFlight && b.ID < a.ID) {
				best = c
			}
		}
		return best

	case strategyConsistent:
		// 把 candidates 按 HashRingPos 排序，找 ≥ hash(client_key) 的第一个；环式 wrap
		clientHash := hashKeyToRingPos(clientKey)
		ring := append([]int{}, candidates...)
		sort.Slice(ring, func(i, j int) bool {
			return st.Backends[ring[i]].HashRingPos < st.Backends[ring[j]].HashRingPos
		})
		for _, idx := range ring {
			if st.Backends[idx].HashRingPos >= clientHash {
				return idx
			}
		}
		return ring[0] // wrap
	}
	return candidates[0]
}

// stepTick 推进 1 tick：完成上一批 pending，进入本批新请求。
func (st *snapState) stepTick() {
	st.Tick++

	// 1) 处理上一批 pending：in_flight ≤ capacity → 完成；否则拒绝
	completedThisTick := []pendingReq{}
	for _, p := range st.Pending {
		idx := -1
		for i, b := range st.Backends {
			if b.ID == p.BackendID {
				idx = i
				break
			}
		}
		if idx < 0 {
			p.Rejected = true
			completedThisTick = append(completedThisTick, p)
			continue
		}
		if st.Backends[idx].IsDown {
			st.Backends[idx].TotalRejected++
			p.Rejected = true
			completedThisTick = append(completedThisTick, p)
			st.Backends[idx].InFlight--
			if st.Backends[idx].InFlight < 0 {
				st.Backends[idx].InFlight = 0
			}
			continue
		}
		st.Backends[idx].TotalProcessed++
		st.Backends[idx].InFlight--
		if st.Backends[idx].InFlight < 0 {
			st.Backends[idx].InFlight = 0
		}
		completedThisTick = append(completedThisTick, p)
	}
	st.History = append(st.History, completedThisTick...)
	if len(st.History) > 32 {
		st.History = st.History[len(st.History)-32:]
	}

	// 2) 入队本 tick 新请求
	st.Pending = nil
	for k := 0; k < st.RPS; k++ {
		clientKey := fmt.Sprintf("c%d-t%d", k, st.Tick)
		idx := st.pickBackend(clientKey)
		if idx < 0 {
			st.History = append(st.History, pendingReq{ClientKey: clientKey, Rejected: true, StartTick: st.Tick})
			continue
		}
		// 检查 capacity 过载
		if st.Backends[idx].InFlight >= st.Backends[idx].Capacity {
			st.Backends[idx].TotalRejected++
			st.History = append(st.History, pendingReq{
				ClientKey: clientKey, BackendID: st.Backends[idx].ID,
				StartTick: st.Tick, Rejected: true,
			})
			continue
		}
		st.Backends[idx].InFlight++
		st.Pending = append(st.Pending, pendingReq{
			ClientKey: clientKey, BackendID: st.Backends[idx].ID, StartTick: st.Tick,
		})
		if st.Strategy == strategyRR {
			st.RRCounter++
		}
	}
}

// totalProcessed / totalRejected 统计。
func (st snapState) totalProcessed() int {
	n := 0
	for _, b := range st.Backends {
		n += b.TotalProcessed
	}
	return n
}

func (st snapState) totalRejected() int {
	n := 0
	for _, b := range st.Backends {
		n += b.TotalRejected
	}
	return n
}

// loadVariance 节点负载（in_flight / capacity）的方差，衡量均衡度。
func (st snapState) loadVariance() float64 {
	N := st.activeCount()
	if N == 0 {
		return 0
	}
	mean := 0.0
	for _, b := range st.Backends {
		if b.IsDown {
			continue
		}
		mean += float64(b.InFlight) / float64(b.Capacity)
	}
	mean /= float64(N)
	variance := 0.0
	for _, b := range st.Backends {
		if b.IsDown {
			continue
		}
		x := float64(b.InFlight)/float64(b.Capacity) - mean
		variance += x * x
	}
	return variance / float64(N)
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
		Strategy:  fw.MapStr(d, "strategy", strategyLeast),
		RPS:       fw.MapInt(d, "rps", 8),
		Tick:      fw.MapInt(d, "tick", 0),
		RRCounter: fw.MapInt(d, "rr_counter", 0),
		LastError: fw.MapStr(d, "last_error", ""),
	}
	if bsAny, ok := d["backends"].([]any); ok {
		for _, bAny := range bsAny {
			if bm, ok := bAny.(map[string]any); ok {
				st.Backends = append(st.Backends, backend{
					ID:             fw.MapStr(bm, "id", ""),
					Capacity:       fw.MapInt(bm, "capacity", 4),
					InFlight:       fw.MapInt(bm, "in_flight", 0),
					TotalProcessed: fw.MapInt(bm, "processed", 0),
					TotalRejected:  fw.MapInt(bm, "rejected", 0),
					IsDown:         fw.MapBool(bm, "down", false),
					HashRingPos:    uint32(fw.MapInt(bm, "ring_pos", 0)),
				})
			}
		}
	}
	if len(st.Backends) == 0 {
		return defaultSnapState()
	}
	if pAny, ok := d["pending"].([]any); ok {
		for _, p := range pAny {
			if pm, ok := p.(map[string]any); ok {
				st.Pending = append(st.Pending, pendingReq{
					ClientKey: fw.MapStr(pm, "client", ""),
					BackendID: fw.MapStr(pm, "backend", ""),
					StartTick: fw.MapInt(pm, "start", 0),
				})
			}
		}
	}
	if hAny, ok := d["history"].([]any); ok {
		for _, h := range hAny {
			if hm, ok := h.(map[string]any); ok {
				st.History = append(st.History, pendingReq{
					ClientKey: fw.MapStr(hm, "client", ""),
					BackendID: fw.MapStr(hm, "backend", ""),
					StartTick: fw.MapInt(hm, "start", 0),
					Rejected:  fw.MapBool(hm, "rejected", false),
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
	s.Data["strategy"] = st.Strategy
	s.Data["rps"] = st.RPS
	s.Data["tick"] = st.Tick
	s.Data["rr_counter"] = st.RRCounter
	s.Data["last_error"] = st.LastError
	bsAny := make([]any, len(st.Backends))
	for i, b := range st.Backends {
		bsAny[i] = map[string]any{
			"id": b.ID, "capacity": b.Capacity, "in_flight": b.InFlight,
			"processed": b.TotalProcessed, "rejected": b.TotalRejected,
			"down": b.IsDown, "ring_pos": int(b.HashRingPos),
		}
	}
	s.Data["backends"] = bsAny
	pAny := make([]any, len(st.Pending))
	for i, p := range st.Pending {
		pAny[i] = map[string]any{
			"client": p.ClientKey, "backend": p.BackendID, "start": p.StartTick,
		}
	}
	s.Data["pending"] = pAny
	hAny := make([]any, len(st.History))
	for i, h := range st.History {
		hAny[i] = map[string]any{
			"client": h.ClientKey, "backend": h.BackendID,
			"start": h.StartTick, "rejected": h.Rejected,
		}
	}
	s.Data["history"] = hAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "节点负载均衡",
		Description:         "演示 4 种 LB 策略（random / round-robin / least-loaded / consistent-hash）+ 过载拒绝 + 一致性哈希环",
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
			"network.lb.dispatched",
			"network.lb.rejected",
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
				ActionCode: "set_params", Label: "设置 LB 参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "strategy", Type: fw.FieldEnum, Label: "策略", Required: true, Default: strategyLeast,
						Options: []any{strategyRandom, strategyRR, strategyLeast, strategyConsistent}},
					{Name: "rps", Type: fw.FieldNumber, Label: "每 tick 请求数", Required: true, Default: 8, Min: 1, Max: 64, Step: 1},
				},
				LinkOwnerFields: []string{"network.load_balance.strategy"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.IntervenePhase,
				WritesOwnedFields: []string{"network.lb.dispatched"},
				LinkOwnerFields:   []string{"network.lb.dispatched"},
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
				ActionCode: "set_capacity", Label: "设置节点容量",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "backend_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "b0"},
					{Name: "capacity", Type: fw.FieldNumber, Label: "容量", Required: true, Default: 4, Min: 1, Max: 32, Step: 1},
				},
			},
			{
				ActionCode: "crash_backend", Label: "节点崩溃",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{{Name: "backend_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "b1"}},
			},
			{
				ActionCode: "recover_backend", Label: "节点恢复",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{{Name: "backend_id", Type: fw.FieldString, Label: "节点 ID", Required: true, Default: "b1"}},
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
	env := buildEnvelope(st, "init", "Load Balance 初始化（5 节点 / least-loaded / RPS=8）", true)
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
		st.Strategy = fw.MapStr(in.Params, "strategy", strategyLeast)
		st.RPS = fw.MapInt(in.Params, "rps", 8)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("strategy=%s  rps=%d", st.Strategy, st.RPS), false)
		return out, nil

	case "step_tick":
		st.stepTick()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_tick",
			fmt.Sprintf("tick=%d processed=%d rejected=%d", st.Tick, st.totalProcessed(), st.totalRejected()), false)
		appendStepMicroSteps(&out.Render, st.Strategy)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.stepTick()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n_ticks",
			fmt.Sprintf("推进 %d tick → processed=%d rejected=%d", n, st.totalProcessed(), st.totalRejected()), false)
		appendStepMicroSteps(&out.Render, st.Strategy)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "set_capacity":
		bid := fw.MapStr(in.Params, "backend_id", "b0")
		cap := fw.MapInt(in.Params, "capacity", 4)
		for i := range st.Backends {
			if st.Backends[i].ID == bid {
				st.Backends[i].Capacity = cap
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_capacity", fmt.Sprintf("%s capacity=%d", bid, cap), false)
		return out, nil

	case "crash_backend":
		bid := fw.MapStr(in.Params, "backend_id", "b1")
		for i := range st.Backends {
			if st.Backends[i].ID == bid {
				st.Backends[i].IsDown = true
				st.Backends[i].InFlight = 0
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "crash_backend", bid+" 崩溃", false)
		appendCrashMicroSteps(&out.Render, bid)
		return out, nil

	case "recover_backend":
		bid := fw.MapStr(in.Params, "backend_id", "b1")
		for i := range st.Backends {
			if st.Backends[i].ID == bid {
				st.Backends[i].IsDown = false
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "recover_backend", bid+" 恢复", false)
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

	// 1) 客户端 → LB → backends 流水线
	stackIDs := []string{"client", "lb-router", "backends-stack"}
	prims = append(prims, fw.PrimStack("layout", stackIDs, "horizontal"))
	prims = append(prims, fw.PrimNode("client", "客户端\nRPS="+fmt.Sprintf("%d", st.RPS), "active", "client"))
	prims = append(prims, fw.PrimNode("lb-router", "LB 路由\n"+st.Strategy, "active", "router"))

	// 2) backends 节点
	backendIDs := []string{}
	for _, b := range st.Backends {
		backendIDs = append(backendIDs, "be-"+b.ID)
	}
	prims = append(prims, fw.PrimStack("backends-stack", backendIDs, "vertical"))
	for _, b := range st.Backends {
		status := "normal"
		role := "backend"
		if b.IsDown {
			status = "error"
			role = "down"
		} else if b.InFlight >= b.Capacity {
			status = "warning"
			role = "overloaded"
		} else if b.InFlight > 0 {
			status = "active"
			role = "busy"
		}
		fillPct := 100 * b.InFlight / max(b.Capacity, 1)
		label := fmt.Sprintf("%s\n%d/%d (%d%%)\nproc=%d rej=%d",
			b.ID, b.InFlight, b.Capacity, fillPct, b.TotalProcessed, b.TotalRejected)
		prims = append(prims, fw.PrimNode("be-"+b.ID, label, status, role))
	}

	// 3) client → router 边
	prims = append(prims, fw.PrimEdge("edge-cli-lb", "client", "lb-router", "solid", "flow"))

	// 4) router → backend 边（只画当前 tick 收到请求的节点）
	usedBackends := map[string]bool{}
	for _, p := range st.Pending {
		usedBackends[p.BackendID] = true
	}
	for _, b := range st.Backends {
		anim := ""
		if usedBackends[b.ID] {
			anim = "flow"
		}
		style := "solid"
		if b.IsDown {
			style = "dashed"
		}
		prims = append(prims, fw.PrimEdge(
			"edge-lb-"+b.ID, "lb-router", "be-"+b.ID, style, anim))
	}

	// 5) 一致性哈希环（仅 strategy=consistent-hash 时）
	if st.Strategy == strategyConsistent {
		// 用 ring_layout 显式声明 N 个环上节点（按 st.Backends 顺序）
		ringNodeIDs := make([]string, len(st.Backends))
		for i, b := range st.Backends {
			ringNodeIDs[i] = "ring-" + b.ID
		}
		prims = append(prims, fw.PrimRingLayout("hash-ring", ringNodeIDs))
		for _, b := range st.Backends {
			role := "ring-node"
			status := "normal"
			if b.IsDown {
				status = "error"
			}
			label := fmt.Sprintf("%s\nring=%d", b.ID, b.HashRingPos)
			prims = append(prims, fw.PrimNode("ring-"+b.ID, label, status, role))
		}
	}

	// 6) 公式
	prims = append(prims, fw.PrimMathFormula("formula-strategy",
		`\text{least-loaded}: i^* = \arg\min_i \text{InFlight}_i;\quad \text{consistent-hash}: i^* = \min\{i: \mathrm{ring}_i \ge h(\text{key})\}`, false))

	// 7) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("strategy = %s\nRPS = %d\nbackend = %d (active=%d)\nprocessed total = %d\nrejected total = %d\nload variance = %.4f\ntick = %d",
			st.Strategy, st.RPS, len(st.Backends), st.activeCount(),
			st.totalProcessed(), st.totalRejected(), st.loadVariance(), st.Tick),
		"text", nil, 8))

	// 8) 节点详情
	rows := []string{"id   in_flight/cap   processed   rejected   ring_pos     status"}
	for _, b := range st.Backends {
		statusStr := "active"
		if b.IsDown {
			statusStr = "DOWN"
		} else if b.InFlight >= b.Capacity {
			statusStr = "OVERLOAD"
		}
		rows = append(rows, fmt.Sprintf("%-3s  %d/%-7d   %-9d   %-8d   %-10d   %s",
			b.ID, b.InFlight, b.Capacity, b.TotalProcessed, b.TotalRejected, b.HashRingPos, statusStr))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-backends", strings.Join(rows, "\n"), "text", nil, 12))

	// 9) 节点负载条形图
	for _, b := range st.Backends {
		colorRole := "info"
		if b.IsDown {
			colorRole = "danger"
		} else if b.InFlight >= b.Capacity {
			colorRole = "warning"
		}
		prims = append(prims, fw.PrimBar("bar-"+b.ID,
			float64(b.InFlight)/float64(max(b.Capacity, 1)),
			0, colorRole, b.ID))
	}

	// 10) 历史请求
	if len(st.History) > 0 {
		histLines := []string{"最近 16 条请求："}
		startIdx := 0
		if len(st.History) > 16 {
			startIdx = len(st.History) - 16
		}
		for _, h := range st.History[startIdx:] {
			tag := "ok"
			if h.Rejected {
				tag = "REJECTED"
			}
			histLines = append(histLines, fmt.Sprintf("  t=%d  %s → %s  [%s]",
				h.StartTick, h.ClientKey, h.BackendID, tag))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-history", strings.Join(histLines, "\n"), "text", nil, 16))
	}

	// 11) 动效
	for _, b := range st.Backends {
		if b.InFlight >= b.Capacity && !b.IsDown {
			prims = append(prims, fw.PrimGlow("glow-overload-"+b.ID, "be-"+b.ID, "warning", 0.8))
			prims = append(prims, fw.PrimShake("shake-overload-"+b.ID, "be-"+b.ID, 0.3, 600))
		}
		if b.IsDown {
			prims = append(prims, fw.PrimGlow("glow-down-"+b.ID, "be-"+b.ID, "danger", 0.7))
		}
	}
	prims = append(prims, fw.PrimPulse("pulse-router", "lb-router", "info", 1500))

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "LB 错误", st.LastError, "scene", "请检查参数", true))
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
		"strategy":        st.Strategy,
		"rps":             st.RPS,
		"tick":            st.Tick,
		"backend_count":   len(st.Backends),
		"active_backends": st.activeCount(),
		"total_processed": st.totalProcessed(),
		"total_rejected":  st.totalRejected(),
		"load_variance":   st.loadVariance(),
		"rr_counter":      st.RRCounter,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendStepMicroSteps(env *fw.RenderEnvelope, strategy string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "ls-1", Label: "上一批 in_flight 完成", DurationMs: 400, HighlightIDs: []string{"cb-backends"}},
		{ID: "ls-2", Label: "客户端发起 RPS 个新请求", DurationMs: 400, HighlightIDs: []string{"client", "lb-router"}, FirePrimitives: []string{"pulse-router"}},
		{ID: "ls-3", Label: strategy + " 路由 → backend", DurationMs: 500, HighlightIDs: []string{"formula-strategy", "cb-history"}, IsLinkTrigger: true},
	}
}

func appendCrashMicroSteps(env *fw.RenderEnvelope, bid string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "cr-1", Label: bid + " 崩溃", DurationMs: 400, HighlightIDs: []string{"be-" + bid}, FirePrimitives: []string{"glow-down-" + bid}},
		{ID: "cr-2", Label: "请求重路由到剩余节点", DurationMs: 500, HighlightIDs: []string{"cb-backends", "lb-router"}},
		{ID: "cr-3", Label: "consistent-hash 仅 1/N 请求重映射", DurationMs: 500, HighlightIDs: []string{"hash-ring"}, IsLinkTrigger: true},
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
		ID:             "lb-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_lb_state",
		LinkGroup:      linkGroupNetworkBase,
		ChangedFields:  []string{"network.lb.dispatched"},
		Payload:        map[string]any{"dispatched": st.totalProcessed(), "rejected": st.totalRejected()},
		SourceAnchorID: "lb-output-anchor",
		TargetAnchorID: "network-base-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.lb.dispatched")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"network": map[string]any{
			"load_balance": map[string]any{
				"strategy":        st.Strategy,
				"rps":             st.RPS,
				"backend_count":   len(st.Backends),
				"active_backends": st.activeCount(),
				"total_processed": st.totalProcessed(),
				"total_rejected":  st.totalRejected(),
				"load_variance":   st.loadVariance(),
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
