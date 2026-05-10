// 模块：sim-engine/scenarios/internal/nodenetwork/blocksync
// 文件职责：NET-05 区块同步（Header-First / Body 并行下载 / 分叉重组）场景的完整实现。
//
// SSOT 依据：06.md §4.1.5 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以太坊 fast-sync / 比特币 IBD 风格的两阶段同步：
//   · Phase 1: header-first — 先下载所有 headers（每 tick 下载 ParallelHeaders 个）
//   · Phase 2: body-download — 多 peer 并行下载 bodies（每 tick 下载 ParallelBodies 个）
//   · 高度选取：从 best peer 开始，逐个对齐到其 tip
//   · 分叉重组：peer 提供的 chain 总难度 > 自己当前链时，回滚到分叉点 + 切换
//   · 同步速率：blocks/tick，含 stale block 监控

package blocksync

import (
	"errors"
	"fmt"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "block-sync"
	schemaVersion = "v1.0.0"
	algorithmType = "block-sync"

	defaultPeerCount       = 3
	defaultNetworkTip      = 64
	defaultParallelHeaders = 8
	defaultParallelBodies  = 4
	maxNetworkTip          = 256

	phaseHeaders = "headers"
	phaseBodies  = "bodies"
	phaseSynced  = "synced"

	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkGroupNetworkBase      = "network-base-group"
	linkOwnerSubtree          = "network.block_sync"
)

type peerInfo struct {
	ID        string
	TipHeight int
	TotalDiff int
	IsDown    bool
}

type snapState struct {
	NetworkTip        int // 网络中最长链高度
	NetworkTotalDiff  int // 网络最长链总难度
	LocalHeight       int
	HeadersDownloaded int // 已下载 headers 高度（从 0 到 NetworkTip）
	BodiesDownloaded  int // 已下载 bodies 高度
	Peers             []peerInfo
	Phase             string
	ParallelHeaders   int
	ParallelBodies    int
	Tick              int
	ReorgCount        int
	StaleBlocks       int
	EventLog          []string
	LastError         string
}

func defaultSnapState() snapState {
	st := snapState{
		NetworkTip:       defaultNetworkTip,
		NetworkTotalDiff: defaultNetworkTip * 100,
		Phase:            phaseHeaders,
		ParallelHeaders:  defaultParallelHeaders,
		ParallelBodies:   defaultParallelBodies,
	}
	for i := 0; i < defaultPeerCount; i++ {
		st.Peers = append(st.Peers, peerInfo{
			ID:        fmt.Sprintf("p%d", i),
			TipHeight: defaultNetworkTip,
			TotalDiff: defaultNetworkTip * 100,
		})
	}
	return st
}

func (st snapState) bestPeer() *peerInfo {
	var best *peerInfo
	for i := range st.Peers {
		p := &st.Peers[i]
		if p.IsDown {
			continue
		}
		if best == nil || p.TotalDiff > best.TotalDiff {
			best = p
		}
	}
	return best
}

func (st snapState) syncProgress() float64 {
	if st.NetworkTip == 0 {
		return 0
	}
	return float64(st.LocalHeight) / float64(st.NetworkTip)
}

// stepSync 推进 1 tick 的同步过程。
func (st *snapState) stepSync() {
	st.Tick++
	best := st.bestPeer()
	if best == nil {
		st.pushEvent("⚠ 无可用 peer")
		return
	}
	if st.NetworkTip < best.TipHeight {
		st.NetworkTip = best.TipHeight
	}
	if st.NetworkTotalDiff < best.TotalDiff {
		st.NetworkTotalDiff = best.TotalDiff
	}
	switch st.Phase {
	case phaseHeaders:
		// 每 tick 下载 ParallelHeaders 个
		want := st.HeadersDownloaded + st.ParallelHeaders
		if want > st.NetworkTip {
			want = st.NetworkTip
		}
		st.HeadersDownloaded = want
		st.pushEvent(fmt.Sprintf("Headers 下载至 %d / %d", st.HeadersDownloaded, st.NetworkTip))
		if st.HeadersDownloaded >= st.NetworkTip {
			st.Phase = phaseBodies
			st.pushEvent("✓ Headers 完成 → 进入 Bodies 阶段")
		}
	case phaseBodies:
		// 每 tick 下载 ParallelBodies 个 body（在 [BodiesDownloaded+1, +ParallelBodies] 区间）
		want := st.BodiesDownloaded + st.ParallelBodies
		if want > st.HeadersDownloaded {
			want = st.HeadersDownloaded
		}
		// 模拟少量 stale block（每 16 tick 一个）
		if st.Tick%16 == 0 && want > st.BodiesDownloaded {
			st.StaleBlocks++
		}
		st.BodiesDownloaded = want
		st.LocalHeight = want
		st.pushEvent(fmt.Sprintf("Bodies 下载至 %d / %d", st.BodiesDownloaded, st.NetworkTip))
		if st.BodiesDownloaded >= st.NetworkTip {
			st.Phase = phaseSynced
			st.pushEvent("✓ 同步完成（local = network tip）")
		}
	case phaseSynced:
		// 已同步：检查是否有更新的网络 tip（peer tip > local）
		if best.TipHeight > st.LocalHeight {
			st.Phase = phaseHeaders
			st.pushEvent("发现新区块 → 增量同步")
		}
	}
}

// extendNetworkTip 模拟网络出新块。
func (st *snapState) extendNetworkTip(blocks int) {
	st.NetworkTip += blocks
	st.NetworkTotalDiff += blocks * 100
	for i := range st.Peers {
		if !st.Peers[i].IsDown {
			st.Peers[i].TipHeight = st.NetworkTip
			st.Peers[i].TotalDiff = st.NetworkTotalDiff
		}
	}
	st.pushEvent(fmt.Sprintf("网络新增 %d 块 → tip = %d", blocks, st.NetworkTip))
	if st.LocalHeight < st.NetworkTip {
		st.Phase = phaseHeaders
	}
}

// triggerReorg 模拟分叉重组：某 peer 提供的链总难度 > 当前 → 回滚到分叉点。
func (st *snapState) triggerReorg(forkPoint, newTotalDiff int) {
	if forkPoint < 0 || forkPoint >= st.LocalHeight {
		st.pushEvent("⚠ fork_point 越界")
		return
	}
	rollback := st.LocalHeight - forkPoint
	st.LocalHeight = forkPoint
	st.HeadersDownloaded = forkPoint
	st.BodiesDownloaded = forkPoint
	st.NetworkTotalDiff = newTotalDiff
	st.NetworkTip = forkPoint + (newTotalDiff-forkPoint*100)/100
	if st.NetworkTip < forkPoint+1 {
		st.NetworkTip = forkPoint + 1
	}
	st.ReorgCount++
	st.Phase = phaseHeaders
	st.pushEvent(fmt.Sprintf("⚠ 重组：从 %d 回滚 %d 块 → 新 tip = %d (难度=%d)",
		forkPoint, rollback, st.NetworkTip, newTotalDiff))
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
		NetworkTip:        fw.MapInt(d, "network_tip", defaultNetworkTip),
		NetworkTotalDiff:  fw.MapInt(d, "network_total_diff", defaultNetworkTip*100),
		LocalHeight:       fw.MapInt(d, "local_height", 0),
		HeadersDownloaded: fw.MapInt(d, "headers", 0),
		BodiesDownloaded:  fw.MapInt(d, "bodies", 0),
		Phase:             fw.MapStr(d, "phase", phaseHeaders),
		ParallelHeaders:   fw.MapInt(d, "par_headers", defaultParallelHeaders),
		ParallelBodies:    fw.MapInt(d, "par_bodies", defaultParallelBodies),
		Tick:              fw.MapInt(d, "tick", 0),
		ReorgCount:        fw.MapInt(d, "reorg_count", 0),
		StaleBlocks:       fw.MapInt(d, "stale_blocks", 0),
		LastError:         fw.MapStr(d, "last_error", ""),
	}
	if peersAny, ok := d["peers"].([]any); ok {
		for _, pAny := range peersAny {
			if pm, ok := pAny.(map[string]any); ok {
				st.Peers = append(st.Peers, peerInfo{
					ID:        fw.MapStr(pm, "id", ""),
					TipHeight: fw.MapInt(pm, "tip", 0),
					TotalDiff: fw.MapInt(pm, "diff", 0),
					IsDown:    fw.MapBool(pm, "down", false),
				})
			}
		}
	}
	if len(st.Peers) == 0 {
		return defaultSnapState()
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
	s.Data["network_tip"] = st.NetworkTip
	s.Data["network_total_diff"] = st.NetworkTotalDiff
	s.Data["local_height"] = st.LocalHeight
	s.Data["headers"] = st.HeadersDownloaded
	s.Data["bodies"] = st.BodiesDownloaded
	s.Data["phase"] = st.Phase
	s.Data["par_headers"] = st.ParallelHeaders
	s.Data["par_bodies"] = st.ParallelBodies
	s.Data["tick"] = st.Tick
	s.Data["reorg_count"] = st.ReorgCount
	s.Data["stale_blocks"] = st.StaleBlocks
	s.Data["last_error"] = st.LastError
	peersAny := make([]any, len(st.Peers))
	for i, p := range st.Peers {
		peersAny[i] = map[string]any{
			"id": p.ID, "tip": p.TipHeight, "diff": p.TotalDiff, "down": p.IsDown,
		}
	}
	s.Data["peers"] = peersAny
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
		Code: sceneCode, Name: "区块同步（Header-First）",
		Description:         "演示 Header-First 同步 + Body 并行下载 + 分叉重组",
		Category:            fw.CategoryNodeNetwork,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupBlockchainIntegr, linkGroupNetworkBase},

		// v0.5 协议字段。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"network.block_sync.local_height",
			"network.block_sync.reorg_count",
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
	return fw.SceneState{SceneCode: sceneCode, Tick: 0, Phase: phaseHeaders, Data: map[string]any{}}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_params", Label: "设置同步参数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "network_tip", Type: fw.FieldNumber, Label: "网络 tip 高度", Required: true,
						Default: defaultNetworkTip, Min: 8, Max: maxNetworkTip, Step: 8},
					{Name: "par_headers", Type: fw.FieldNumber, Label: "headers 并行度", Required: true,
						Default: defaultParallelHeaders, Min: 1, Max: 32, Step: 1},
					{Name: "par_bodies", Type: fw.FieldNumber, Label: "bodies 并行度", Required: true,
						Default: defaultParallelBodies, Min: 1, Max: 16, Step: 1},
				},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"network.block_sync.local_height"},
				LinkOwnerFields:   []string{"network.block_sync.local_height"},
			},
			{
				ActionCode: "step_n_ticks", Label: "推进 N tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 5, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "extend_network", Label: "网络出新块",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "blocks", Type: fw.FieldNumber, Label: "新增块数", Required: true, Default: 8, Min: 1, Max: 64, Step: 1},
				},
			},
			{
				ActionCode: "trigger_reorg", Label: "触发分叉重组",
				Description: "回滚到 fork_point，切换到难度 new_total_diff 的新链",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "fork_point", Type: fw.FieldNumber, Label: "分叉点高度", Required: true, Default: 30, Min: 0, Step: 1},
					{Name: "new_total_diff", Type: fw.FieldNumber, Label: "新链总难度", Required: true, Default: 7000, Min: 100, Step: 100},
				},
				WritesOwnedFields: []string{"network.block_sync.reorg_count"},
				LinkOwnerFields:   []string{"network.block_sync.reorg_count"},
			},
			{
				ActionCode: "crash_peer", Label: "Peer 离线",
				Category: fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:  []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{{Name: "peer_id", Type: fw.FieldString, Label: "Peer ID", Required: true, Default: "p0"}},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			// §0.7.4 混合实验：fetch_block 走 geth eth_getBlockByNumber。
			{
				ActionCode:  "fetch_block", Label: "拉取真链区块（容器通道）",
				Description: "调 geth eth_getBlockByNumber 获取真实链上区块",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:        []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["{{block_tag}}",false],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "block_tag", Type: fw.FieldString, Label: "block tag", Required: true, Default: "latest"},
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
	st := loadState(state)
	saveState(state, st)
	state.Phase = st.Phase
	env := buildEnvelope(st, "init", "Block Sync 初始化（network_tip=64, 3 peers）", true)
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
		newTip := fw.MapInt(in.Params, "network_tip", defaultNetworkTip)
		st.ParallelHeaders = fw.MapInt(in.Params, "par_headers", defaultParallelHeaders)
		st.ParallelBodies = fw.MapInt(in.Params, "par_bodies", defaultParallelBodies)
		if newTip != st.NetworkTip {
			st.NetworkTip = newTip
			st.NetworkTotalDiff = newTip * 100
			for i := range st.Peers {
				st.Peers[i].TipHeight = newTip
				st.Peers[i].TotalDiff = st.NetworkTotalDiff
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("network_tip=%d  par_headers=%d  par_bodies=%d", st.NetworkTip, st.ParallelHeaders, st.ParallelBodies), false)
		return out, nil

	case "step_tick":
		st.stepSync()
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_tick",
			fmt.Sprintf("tick=%d phase=%s local=%d/%d", st.Tick, st.Phase, st.LocalHeight, st.NetworkTip), false)
		appendStepMicroSteps(&out.Render, st.Phase)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.stepSync()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n_ticks",
			fmt.Sprintf("推进 %d tick → %d/%d (%.0f%%)", n, st.LocalHeight, st.NetworkTip, st.syncProgress()*100), false)
		appendStepMicroSteps(&out.Render, st.Phase)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "extend_network":
		blocks := fw.MapInt(in.Params, "blocks", 8)
		st.extendNetworkTip(blocks)
		saveState(state, st)
		out.Render = buildEnvelope(st, "extend_network", fmt.Sprintf("网络新增 %d 块", blocks), false)
		return out, nil

	case "trigger_reorg":
		fp := fw.MapInt(in.Params, "fork_point", 30)
		td := fw.MapInt(in.Params, "new_total_diff", 7000)
		st.triggerReorg(fp, td)
		saveState(state, st)
		out.Render = buildEnvelope(st, "trigger_reorg",
			fmt.Sprintf("从 %d 重组（新难度=%d）", fp, td), false)
		appendReorgMicroSteps(&out.Render, fp)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "crash_peer":
		pid := fw.MapStr(in.Params, "peer_id", "p0")
		for i := range st.Peers {
			if st.Peers[i].ID == pid {
				st.Peers[i].IsDown = true
				st.pushEvent(pid + " 离线")
				break
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "crash_peer", pid+" 离线", false)
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

	// 1) 阶段流水线（headers → bodies → synced）
	phaseIDs := []string{"phase-headers", "phase-bodies", "phase-synced"}
	prims = append(prims, fw.PrimStack("phase-stack", phaseIDs, "horizontal"))
	for i, id := range phaseIDs {
		role := []string{"headers", "bodies", "synced"}[i]
		label := []string{"Headers 下载", "Bodies 下载", "已同步"}[i]
		status := "normal"
		if string(id[6:]) == st.Phase {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}
	for i := 0; i < 2; i++ {
		anim := ""
		if i == 0 && (st.Phase == phaseBodies || st.Phase == phaseSynced) {
			anim = "flow"
		}
		if i == 1 && st.Phase == phaseSynced {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-ph-%d", i), phaseIDs[i], phaseIDs[i+1], "solid", anim))
	}

	prims = append(prims, fw.PrimPhaseProgress("phase-progress",
		[]string{"Headers", "Bodies", "Synced"},
		map[string]int{phaseHeaders: 0, phaseBodies: 1, phaseSynced: 2}[st.Phase],
		st.syncProgress()))

	// 2) 同步进度环（local_height vs network_tip）
	prims = append(prims, fw.PrimRing("ring-sync", st.NetworkTip, st.LocalHeight,
		fmt.Sprintf("Local %d / Tip %d", st.LocalHeight, st.NetworkTip)))

	// 3) headers / bodies 进度条
	prims = append(prims, fw.PrimProgressBar("progress-headers", float64(st.HeadersDownloaded), float64(st.NetworkTip),
		fmt.Sprintf("Headers %d/%d", st.HeadersDownloaded, st.NetworkTip)))
	prims = append(prims, fw.PrimProgressBar("progress-bodies", float64(st.BodiesDownloaded), float64(st.NetworkTip),
		fmt.Sprintf("Bodies %d/%d", st.BodiesDownloaded, st.NetworkTip)))

	// 4) Peers
	peerIDs := []string{}
	for _, p := range st.Peers {
		peerIDs = append(peerIDs, "peer-"+p.ID)
	}
	prims = append(prims, fw.PrimStack("peers-stack", peerIDs, "horizontal"))
	best := st.bestPeer()
	for _, p := range st.Peers {
		role := "peer"
		status := "normal"
		if p.IsDown {
			status = "error"
			role = "down"
		} else if best != nil && p.ID == best.ID {
			status = "active"
			role = "best-peer"
		}
		label := fmt.Sprintf("%s\ntip=%d\ndiff=%d", p.ID, p.TipHeight, p.TotalDiff)
		prims = append(prims, fw.PrimNode("peer-"+p.ID, label, status, role))
	}

	// 5) 公式
	prims = append(prims, fw.PrimMathFormula("formula-sync",
		`\text{best peer} = \arg\max_p \mathrm{TotalDiff}_p;\quad \text{reorg if}\ \mathrm{Diff}_{\text{new}} > \mathrm{Diff}_{\text{cur}}`, false))

	// 6) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("phase = %s\nlocal_height = %d / network_tip = %d (%.0f%%)\nheaders = %d, bodies = %d\npar_headers = %d, par_bodies = %d\nreorg_count = %d, stale = %d\ntick = %d",
			st.Phase, st.LocalHeight, st.NetworkTip, st.syncProgress()*100,
			st.HeadersDownloaded, st.BodiesDownloaded,
			st.ParallelHeaders, st.ParallelBodies,
			st.ReorgCount, st.StaleBlocks, st.Tick),
		"text", nil, 8))

	// 7) Peer 表
	rows := []string{"id   tip   diff   status"}
	for _, p := range st.Peers {
		statusStr := "active"
		if p.IsDown {
			statusStr = "DOWN"
		}
		if best != nil && p.ID == best.ID {
			statusStr = "BEST"
		}
		rows = append(rows, fmt.Sprintf("%-3s  %-4d  %-5d  %s", p.ID, p.TipHeight, p.TotalDiff, statusStr))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-peers", strings.Join(rows, "\n"), "text", nil, 8))

	// 8) 事件
	if len(st.EventLog) > 0 {
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(st.EventLog, "\n"), "text", nil, 16))
	}

	// 9) 动效
	prims = append(prims, fw.PrimGlow("glow-phase", "phase-"+st.Phase, "info", 0.8))
	if st.Phase != phaseSynced {
		prims = append(prims, fw.PrimPulse("pulse-sync", "ring-sync", "info", 1500))
	} else {
		prims = append(prims, fw.PrimBurst("burst-synced", "ring-sync", "success", int64(st.Tick), 800))
	}
	if st.ReorgCount > 0 {
		prims = append(prims, fw.PrimShake("shake-reorg", "cb-status", 0.3, 600))
	}

	// 10) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))
	prims = append(prims, fw.PrimLinkIndicator("link-net", linkGroupNetworkBase, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Block Sync 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "sync.peer_count", Value: len(st.Peers), TargetPrimitive: "cb-peers", TargetParam: "count"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	bestID := ""
	if b := st.bestPeer(); b != nil {
		bestID = b.ID
	}
	d := map[string]any{
		"phase":              st.Phase,
		"local_height":       st.LocalHeight,
		"network_tip":        st.NetworkTip,
		"network_total_diff": st.NetworkTotalDiff,
		"headers":            st.HeadersDownloaded,
		"bodies":             st.BodiesDownloaded,
		"sync_progress":      st.syncProgress(),
		"par_headers":        st.ParallelHeaders,
		"par_bodies":         st.ParallelBodies,
		"reorg_count":        st.ReorgCount,
		"stale_blocks":       st.StaleBlocks,
		"best_peer":          bestID,
		"tick":               st.Tick,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendStepMicroSteps(env *fw.RenderEnvelope, phase string) {
	switch phase {
	case phaseHeaders:
		env.MicroSteps = []fw.MicroStep{
			{ID: "h-1", Label: "选最佳 peer（最高 TotalDiff）", DurationMs: 400, HighlightIDs: []string{"formula-sync", "cb-peers"}},
			{ID: "h-2", Label: "并行下载 headers", DurationMs: 500, HighlightIDs: []string{"progress-headers"}, FirePrimitives: []string{"pulse-sync"}},
			{ID: "h-3", Label: "headers 完成 → 进入 bodies 阶段", DurationMs: 400, HighlightIDs: []string{"phase-bodies"}, IsLinkTrigger: true},
		}
	case phaseBodies:
		env.MicroSteps = []fw.MicroStep{
			{ID: "b-1", Label: "headers 已就绪", DurationMs: 400, HighlightIDs: []string{"progress-headers"}},
			{ID: "b-2", Label: "多 peer 并行下载 bodies", DurationMs: 500, HighlightIDs: []string{"progress-bodies", "peers-stack"}, FirePrimitives: []string{"pulse-sync"}},
			{ID: "b-3", Label: "推进 local_height", DurationMs: 400, HighlightIDs: []string{"ring-sync"}, IsLinkTrigger: true},
		}
	case phaseSynced:
		env.MicroSteps = []fw.MicroStep{
			{ID: "s-1", Label: "local = network tip", DurationMs: 400, HighlightIDs: []string{"phase-synced"}, FirePrimitives: []string{"burst-synced"}},
			{ID: "s-2", Label: "等待新区块到达", DurationMs: 400, HighlightIDs: []string{"cb-status"}, IsLinkTrigger: true},
		}
	}
}

func appendReorgMicroSteps(env *fw.RenderEnvelope, fp int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "r-1", Label: "检测到更高难度的分叉链", DurationMs: 400, HighlightIDs: []string{"formula-sync"}, FirePrimitives: []string{"shake-reorg"}},
		{ID: "r-2", Label: fmt.Sprintf("回滚到 fork_point = %d", fp), DurationMs: 500, HighlightIDs: []string{"ring-sync", "cb-status"}},
		{ID: "r-3", Label: "切换到新链 → 重新同步", DurationMs: 500, HighlightIDs: []string{"phase-headers"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	// LinkTrigger 带锚点（§0.7.1 C18）。
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "blocksync-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_height",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"network.block_sync.local_height", "network.block_sync.reorg_count"},
		Payload:        map[string]any{"local_height": st.LocalHeight, "reorg_count": st.ReorgCount},
		SourceAnchorID: "blocksync-output-anchor",
		TargetAnchorID: "chain-tip-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "network.block_sync.local_height", "network.block_sync.reorg_count")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"network": map[string]any{
			"block_sync": map[string]any{
				"phase":         st.Phase,
				"local_height":  st.LocalHeight,
				"network_tip":   st.NetworkTip,
				"sync_progress": st.syncProgress(),
				"reorg_count":   st.ReorgCount,
				"stale_blocks":  st.StaleBlocks,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

