// 模块：sim-engine/scenarios/internal/datastructure/bloomfilter
// 文件职责：DS-04 布隆过滤器场景的完整实现。
//
// SSOT 依据：06.md §4.4.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现布隆过滤器（Bloom Filter）：
//   · 位数组 bits[m]：m 位（默认 64）
//   · k 个独立 hash 函数：h_i(x) = SHA-256(i_byte || x) → 前 4 字节大端整数 mod m
//   · add(x)：对每个 i ∈ [0, k)，把 bits[h_i(x)] 置 1
//   · contains(x)：所有 h_i(x) 位都为 1 才返回 true（假阳可能，假阴不可能）
//   · FPR 理论：p = (1 - e^(-kn/m))^k，当 k = (m/n)·ln 2 时最优
//   · FPR 实测：用未插入的 sample 集统计假阳率
//
// 教学决策：
//   - heat_map 展示位数组（行 = 8，列 = m/8）
//   - 当前查询时高亮 k 个被检查的位
//   - bar 显示已插入数 / FPR

package bloomfilter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

const (
	sceneCode     = "bloom-filter"
	schemaVersion = "v1.0.0"
	algorithmType = "bloom-filter"

	defaultM = 64
	defaultK = 3
	maxM     = 256
	maxK     = 8

	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "datastructure.bloom"
)

// hashAt 计算第 i 个 hash 函数对 item 的输出（mod m）。
func hashAt(i int, item string, m int) int {
	buf := []byte{byte(i)}
	buf = append(buf, []byte(item)...)
	h := sha256hash.Sum256(buf)
	v := binary.BigEndian.Uint32(h[:4])
	return int(v % uint32(m))
}

// =====================================================================
// 状态
// =====================================================================

type checkResult struct {
	Item       string
	Indices    []int  // k 个被检查位
	BitsState  []bool // 每个位的当前状态
	FoundTrue  bool   // 全 1 → 报告"存在"
	ActuallyIn bool   // 真实是否曾插入
	IsFP       bool   // 假阳：FoundTrue && !ActuallyIn
}

type snapState struct {
	M           int
	K           int
	Bits        []bool   // 位数组
	Inserted    []string // 已插入的 item（顺序保留）
	InsertedSet map[string]bool
	LastCheck   *checkResult
	FPRSample   int // 用于实测 FPR 的样本数
	FPRHits     int // 假阳数
	LastError   string
}

func defaultSnapState() snapState {
	return snapState{
		M:           defaultM,
		K:           defaultK,
		Bits:        make([]bool, defaultM),
		InsertedSet: map[string]bool{},
	}
}

// addItem 把 item 插入 bloom。
func (st *snapState) addItem(item string) {
	for i := 0; i < st.K; i++ {
		idx := hashAt(i, item, st.M)
		st.Bits[idx] = true
	}
	if !st.InsertedSet[item] {
		st.Inserted = append(st.Inserted, item)
		st.InsertedSet[item] = true
	}
}

// containsItem 查询 item 是否可能在集合中；返回详细 trace。
func (st snapState) containsItem(item string) checkResult {
	res := checkResult{Item: item, ActuallyIn: st.InsertedSet[item], FoundTrue: true}
	for i := 0; i < st.K; i++ {
		idx := hashAt(i, item, st.M)
		state := st.Bits[idx]
		res.Indices = append(res.Indices, idx)
		res.BitsState = append(res.BitsState, state)
		if !state {
			res.FoundTrue = false
		}
	}
	res.IsFP = res.FoundTrue && !res.ActuallyIn
	return res
}

// theoreticalFPR 计算给定 (m, k, n) 下的理论 FPR。
func theoreticalFPR(m, k, n int) float64 {
	if m == 0 || k == 0 || n == 0 {
		return 0
	}
	x := -float64(k*n) / float64(m)
	return math.Pow(1-math.Exp(x), float64(k))
}

// optimalK 给定 (m, n) 计算最优 k = (m/n) * ln 2。
func optimalK(m, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(m) * math.Ln2 / float64(n)
}

// bitsCount 已置 1 的位数。
func (st snapState) bitsCount() int {
	c := 0
	for _, b := range st.Bits {
		if b {
			c++
		}
	}
	return c
}

// runFPRSample 用 n 个未插入样本检测假阳率。
func (st *snapState) runFPRSample(n int) {
	st.FPRSample = n
	st.FPRHits = 0
	for i := 0; i < n; i++ {
		probe := fmt.Sprintf("not-inserted-%d", i)
		if st.InsertedSet[probe] {
			continue
		}
		res := st.containsItem(probe)
		if res.FoundTrue {
			st.FPRHits++
		}
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
		M:           fw.MapInt(d, "m", defaultM),
		K:           fw.MapInt(d, "k", defaultK),
		FPRSample:   fw.MapInt(d, "fpr_sample", 0),
		FPRHits:     fw.MapInt(d, "fpr_hits", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
		InsertedSet: map[string]bool{},
	}
	st.Bits = make([]bool, st.M)
	if bAny, ok := d["bits"].([]any); ok {
		for i, v := range bAny {
			if i >= st.M {
				break
			}
			st.Bits[i] = boolFromAny(v)
		}
	}
	if insAny, ok := d["inserted"].([]any); ok {
		for _, v := range insAny {
			if s, ok := v.(string); ok {
				st.Inserted = append(st.Inserted, s)
				st.InsertedSet[s] = true
			}
		}
	}
	if lcAny, ok := d["last_check"].(map[string]any); ok {
		lc := &checkResult{
			Item:       fw.MapStr(lcAny, "item", ""),
			FoundTrue:  fw.MapBool(lcAny, "found", false),
			ActuallyIn: fw.MapBool(lcAny, "in_set", false),
			IsFP:       fw.MapBool(lcAny, "is_fp", false),
		}
		if iAny, ok := lcAny["indices"].([]any); ok {
			for _, v := range iAny {
				lc.Indices = append(lc.Indices, intFromAny(v))
			}
		}
		if bAny, ok := lcAny["bits_state"].([]any); ok {
			for _, v := range bAny {
				lc.BitsState = append(lc.BitsState, boolFromAny(v))
			}
		}
		st.LastCheck = lc
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["m"] = st.M
	s.Data["k"] = st.K
	s.Data["fpr_sample"] = st.FPRSample
	s.Data["fpr_hits"] = st.FPRHits
	s.Data["last_error"] = st.LastError
	bAny := make([]any, len(st.Bits))
	for i, b := range st.Bits {
		bAny[i] = b
	}
	s.Data["bits"] = bAny
	insAny := make([]any, len(st.Inserted))
	for i, v := range st.Inserted {
		insAny[i] = v
	}
	s.Data["inserted"] = insAny
	if st.LastCheck != nil {
		iAny := make([]any, len(st.LastCheck.Indices))
		for i, v := range st.LastCheck.Indices {
			iAny[i] = v
		}
		bsAny := make([]any, len(st.LastCheck.BitsState))
		for i, v := range st.LastCheck.BitsState {
			bsAny[i] = v
		}
		s.Data["last_check"] = map[string]any{
			"item":       st.LastCheck.Item,
			"found":      st.LastCheck.FoundTrue,
			"in_set":     st.LastCheck.ActuallyIn,
			"is_fp":      st.LastCheck.IsFP,
			"indices":    iAny,
			"bits_state": bsAny,
		}
	}
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "布隆过滤器（Bloom Filter）",
		Description:         "演示 m bits + k SHA-256 hash 的概率集合：add / contains / 假阳率 / 最优 k 估算",
		Category:            fw.CategoryDataStructure,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"datastructure.bloom.fpr_actual",
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
				ActionCode: "set_params", Label: "设置 (m, k)",
				Description: "重置位数组并设置位数 m / hash 函数数 k",
				Category:    fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "m", Type: fw.FieldNumber, Label: "位数 m", Required: true, Default: defaultM, Min: 8, Max: maxM, Step: 8},
					{Name: "k", Type: fw.FieldNumber, Label: "hash 函数数 k", Required: true, Default: defaultK, Min: 1, Max: maxK, Step: 1},
				},
			},
			{
				ActionCode: "add", Label: "插入元素",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "item", Type: fw.FieldString, Label: "元素", Required: true, Default: "alice"},
				},
				WritesOwnedFields: []string{"datastructure.bloom.fpr_actual"},
				LinkOwnerFields:   []string{"datastructure.bloom.fpr_actual"},
			},
			{
				ActionCode: "bulk_add", Label: "批量插入",
				Description: "插入 N 个 item-0, item-1, ..., item-(N-1)",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "数量", Required: true, Default: 8, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "contains", Label: "查询元素",
				Description: "返回 \"可能存在\"（k 位全 1）/ \"绝对不存在\"（≥1 位为 0）",
				Category:    fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "item", Type: fw.FieldString, Label: "查询元素", Required: true, Default: "bob"},
				},
				LinkOwnerFields: []string{"datastructure.bloom.last_query"},
			},
			{
				ActionCode: "measure_fpr", Label: "实测假阳率（FPR）",
				Description: "用 N 个未插入样本统计假阳频率",
				Category:    fw.ActionObserve, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "samples", Type: fw.FieldNumber, Label: "样本数", Required: true, Default: 100, Min: 10, Max: 10000, Step: 10},
				},
				LinkOwnerFields: []string{"datastructure.bloom.fpr_actual"},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
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
	env := buildEnvelope(st, "init", "Bloom Filter 初始化（m=64, k=3）", true)
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
		m := fw.MapInt(in.Params, "m", defaultM)
		k := fw.MapInt(in.Params, "k", defaultK)
		st = snapState{M: m, K: k, Bits: make([]bool, m), InsertedSet: map[string]bool{}}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", fmt.Sprintf("重置：m=%d k=%d", m, k), true)
		return out, nil

	case "add":
		item := fw.MapStr(in.Params, "item", "")
		if item == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "item 不能为空"}, nil
		}
		st.addItem(item)
		saveState(state, st)
		out.Render = buildEnvelope(st, "add", fmt.Sprintf("已插入 %s", item), false)
		appendAddMicroSteps(&out.Render, item)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "bulk_add":
		n := fw.MapInt(in.Params, "n", 8)
		for i := 0; i < n; i++ {
			st.addItem(fmt.Sprintf("item-%d", i))
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "bulk_add", fmt.Sprintf("批量插入 %d 个", n), false)
		appendBulkAddMicroSteps(&out.Render, n)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "contains":
		item := fw.MapStr(in.Params, "item", "")
		res := st.containsItem(item)
		st.LastCheck = &res
		saveState(state, st)
		summary := "✓ 可能存在"
		if !res.FoundTrue {
			summary = "✗ 绝对不存在"
		}
		if res.IsFP {
			summary += "（实际未插入 → 假阳）"
		}
		out.Render = buildEnvelope(st, "contains", summary, false)
		appendContainsMicroSteps(&out.Render, item, res.FoundTrue, res.IsFP)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "measure_fpr":
		n := fw.MapInt(in.Params, "samples", 100)
		st.runFPRSample(n)
		saveState(state, st)
		fprActual := 0.0
		if n > 0 {
			fprActual = float64(st.FPRHits) / float64(n)
		}
		fprTheo := theoreticalFPR(st.M, st.K, len(st.Inserted))
		summary := fmt.Sprintf("实测 %d/%d=%.2f%% / 理论 %.2f%%",
			st.FPRHits, n, fprActual*100, fprTheo*100)
		out.Render = buildEnvelope(st, "measure_fpr", summary, false)
		appendFPRMicroSteps(&out.Render, fprActual, fprTheo)
		out.SharedStateDiff = ownerDiff(st)
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
	prims := make([]fw.Primitive, 0, 30)

	// 1) heat_map 位数组
	cols := 8
	rows := (st.M + cols - 1) / cols
	cells := make([]map[string]any, 0, st.M)
	checkSet := map[int]bool{}
	if st.LastCheck != nil {
		for _, idx := range st.LastCheck.Indices {
			checkSet[idx] = true
		}
	}
	for i := 0; i < st.M; i++ {
		row := i / cols
		col := i % cols
		val := 0
		color := "muted"
		if st.Bits[i] {
			val = 1
			color = "info"
		}
		if checkSet[i] {
			if st.Bits[i] {
				color = "success"
			} else {
				color = "danger"
			}
		}
		cells = append(cells, map[string]any{
			"row": row, "col": col, "value": val, "color_role": color,
		})
	}
	prims = append(prims, fw.PrimHeatMap("bits-array", rows, cols, cells))

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-fpr",
		`p \approx \left(1 - e^{-kn/m}\right)^k;\quad k_{\mathrm{opt}} = \frac{m}{n}\ln 2`, false))
	prims = append(prims, fw.PrimMathFormula("formula-hash",
		`h_i(x) = \mathrm{SHA256}(i \,\|\, x)[0..3] \bmod m`, false))

	// 3) 状态参数
	bitsCnt := st.bitsCount()
	fillRatio := 0.0
	if st.M > 0 {
		fillRatio = float64(bitsCnt) / float64(st.M)
	}
	fprTheo := theoreticalFPR(st.M, st.K, len(st.Inserted))
	kOpt := optimalK(st.M, len(st.Inserted))
	fprActual := 0.0
	if st.FPRSample > 0 {
		fprActual = float64(st.FPRHits) / float64(st.FPRSample)
	}
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("m = %d  k = %d  n = %d\n位填充率 = %d/%d = %.1f%%\n理论 FPR = %.4f%% (= %.4f)\n最优 k = %.2f\n实测 FPR = %d/%d = %.2f%%",
			st.M, st.K, len(st.Inserted),
			bitsCnt, st.M, fillRatio*100,
			fprTheo*100, fprTheo, kOpt,
			st.FPRHits, st.FPRSample, fprActual*100),
		"text", nil, 8))

	// 4) 已插入元素表
	insRows := []string{fmt.Sprintf("已插入 %d 个：", len(st.Inserted))}
	startIdx := 0
	if len(st.Inserted) > 16 {
		startIdx = len(st.Inserted) - 16
		insRows = append(insRows, "  …")
	}
	for _, item := range st.Inserted[startIdx:] {
		// 显示该 item 的 k 个位
		positions := []string{}
		for i := 0; i < st.K; i++ {
			positions = append(positions, fmt.Sprintf("%d", hashAt(i, item, st.M)))
		}
		insRows = append(insRows, fmt.Sprintf("  %s → bits[%s]", item, strings.Join(positions, ",")))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-inserted", strings.Join(insRows, "\n"), "text", nil, 14))

	// 5) 上次查询详情
	if st.LastCheck != nil {
		lc := st.LastCheck
		queryLines := []string{
			fmt.Sprintf("query = %q", lc.Item),
			fmt.Sprintf("k=%d 个 hash → 位索引 = %v", st.K, lc.Indices),
		}
		for i, idx := range lc.Indices {
			state := "0"
			if i < len(lc.BitsState) && lc.BitsState[i] {
				state = "1"
			}
			queryLines = append(queryLines, fmt.Sprintf("  h_%d(%q) = %d → bits[%d] = %s",
				i, lc.Item, idx, idx, state))
		}
		queryLines = append(queryLines, "")
		if lc.FoundTrue {
			queryLines = append(queryLines, "✓ 所有位 = 1 → 报告\"可能存在\"")
		} else {
			queryLines = append(queryLines, "✗ ≥1 位 = 0 → 报告\"绝对不存在\"")
		}
		queryLines = append(queryLines, fmt.Sprintf("  实际是否在集合：%v", lc.ActuallyIn))
		if lc.IsFP {
			queryLines = append(queryLines, "  ⚠ 假阳！")
		}
		prims = append(prims, fw.PrimCodeBlock("cb-query", strings.Join(queryLines, "\n"), "text", nil, 14))
	}

	// 6) 进度条：填充率 / FPR
	prims = append(prims, fw.PrimProgressBar("fill-progress", fillRatio*100, 100,
		fmt.Sprintf("位填充率 %.0f%%", fillRatio*100)))
	if st.FPRSample > 0 {
		prims = append(prims, fw.PrimProgressBar("fpr-progress", fprActual*100, 100,
			fmt.Sprintf("实测 FPR %.2f%%", fprActual*100)))
	}

	// 7) 风险仪表（FPR 风险）
	prims = append(prims, fw.PrimRiskGauge("fpr-gauge", fprTheo*100,
		[]map[string]any{
			{"from": 0.0, "to": 5.0, "color": "success"},
			{"from": 5.0, "to": 20.0, "color": "warning"},
			{"from": 20.0, "to": 100.0, "color": "danger"},
		},
	))

	// 8) 动效
	if st.LastCheck != nil {
		col := "info"
		if st.LastCheck.IsFP {
			col = "danger"
		} else if st.LastCheck.FoundTrue {
			col = "success"
		} else {
			col = "warning"
		}
		prims = append(prims, fw.PrimPulse("pulse-query", "cb-query", col, 1500))
		if st.LastCheck.IsFP {
			prims = append(prims, fw.PrimShake("shake-fp", "cb-query", 0.4, 700))
		}
	}

	// 9) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Bloom 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	bitsCnt := st.bitsCount()
	fprTheo := theoreticalFPR(st.M, st.K, len(st.Inserted))
	fprActual := 0.0
	if st.FPRSample > 0 {
		fprActual = float64(st.FPRHits) / float64(st.FPRSample)
	}
	d := map[string]any{
		"m":           st.M,
		"k":           st.K,
		"n":           len(st.Inserted),
		"bits_set":    bitsCnt,
		"fill_ratio":  float64(bitsCnt) / float64(st.M),
		"fpr_theory":  fprTheo,
		"fpr_actual":  fprActual,
		"fpr_samples": st.FPRSample,
		"fpr_hits":    st.FPRHits,
		"optimal_k":   optimalK(st.M, len(st.Inserted)),
	}
	if st.LastCheck != nil {
		d["last_query"] = st.LastCheck.Item
		d["last_found"] = st.LastCheck.FoundTrue
		d["last_is_fp"] = st.LastCheck.IsFP
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendAddMicroSteps(env *fw.RenderEnvelope, item string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "ad-1", Label: "对元素计算 k 个 hash", DurationMs: 400, HighlightIDs: []string{"formula-hash"}},
		{ID: "ad-2", Label: "把对应 k 位置 1", DurationMs: 500, HighlightIDs: []string{"bits-array", "fill-progress"}},
		{ID: "ad-3", Label: "更新填充率与 FPR 估算", DurationMs: 400, HighlightIDs: []string{"cb-status", "fpr-gauge"}, IsLinkTrigger: true},
	}
}

func appendBulkAddMicroSteps(env *fw.RenderEnvelope, n int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "bk-1", Label: fmt.Sprintf("批量插入 %d 个 item", n), DurationMs: 500, HighlightIDs: []string{"cb-inserted"}},
		{ID: "bk-2", Label: "位填充率随 n 增长", DurationMs: 500, HighlightIDs: []string{"fill-progress"}},
		{ID: "bk-3", Label: "FPR 上升", DurationMs: 500, HighlightIDs: []string{"fpr-gauge", "formula-fpr"}, IsLinkTrigger: true},
	}
}

func appendContainsMicroSteps(env *fw.RenderEnvelope, item string, found, isFP bool) {
	tail := "✗ 绝对不存在"
	if found {
		tail = "✓ 可能存在"
	}
	if isFP {
		tail += " ⚠ 假阳"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "ct-1", Label: fmt.Sprintf("计算 k=%d 个 hash(%s)", 0, item), DurationMs: 400, HighlightIDs: []string{"formula-hash"}},
		{ID: "ct-2", Label: "检查 k 位是否全 1", DurationMs: 500, HighlightIDs: []string{"bits-array", "cb-query"}},
		{ID: "ct-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-query"}, FirePrimitives: []string{"pulse-query", "shake-fp"}, IsLinkTrigger: true},
	}
}

func appendFPRMicroSteps(env *fw.RenderEnvelope, fprActual, fprTheo float64) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "fp-1", Label: "用未插入样本依次 contains 检测", DurationMs: 500, HighlightIDs: []string{"bits-array"}},
		{ID: "fp-2", Label: fmt.Sprintf("统计假阳数 / 样本总数 = %.2f%%", fprActual*100), DurationMs: 500, HighlightIDs: []string{"fpr-progress"}},
		{ID: "fp-3", Label: fmt.Sprintf("与理论 %.2f%% 对比", fprTheo*100), DurationMs: 500, HighlightIDs: []string{"formula-fpr", "fpr-gauge"}, IsLinkTrigger: true},
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
		ID:             "bloom-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_bloom",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"datastructure.bloom.fpr_actual"},
		Payload:        map[string]any{"m": st.M, "k": st.K, "n": len(st.Inserted)},
		SourceAnchorID: "bloom-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "datastructure.bloom.fpr_actual")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	fprActual := 0.0
	if st.FPRSample > 0 {
		fprActual = float64(st.FPRHits) / float64(st.FPRSample)
	}
	d := map[string]any{
		"m":          st.M,
		"k":          st.K,
		"n":          len(st.Inserted),
		"bits_set":   st.bitsCount(),
		"fpr_theory": theoreticalFPR(st.M, st.K, len(st.Inserted)),
		"fpr_actual": fprActual,
	}
	if st.LastCheck != nil {
		d["last_query"] = st.LastCheck.Item
		d["last_found"] = st.LastCheck.FoundTrue
	}
	return map[string]any{
		"datastructure": map[string]any{
			"bloom": d,
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

func boolFromAny(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
