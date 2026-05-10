// 模块：sim-engine/scenarios/internal/transaction/txlifecycle
// 文件职责：TX-01 交易生命周期场景的完整实现。
//
// SSOT 依据：06.md §4.5.1 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现交易 5 阶段状态机（零外部依赖）：
//   · created → mempool → included → confirmed → finalized
//   · 每 BlockInterval tick 自动出块 / 也可手动 mine_block
//   · included 后每多一个块 confirmations++；≥ ConfirmationThreshold 进入 confirmed
//   · ≥ FinalityThreshold 进入 finalized（不可逆）
//   · drop_tx：主动丢弃 mempool 中的 tx
//   · replace_tx：用相同 nonce + 更高 gas 替换 mempool 中已有 tx

package txlifecycle

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "tx-lifecycle"
	schemaVersion = "v1.0.0"
	algorithmType = "tx-lifecycle"

	defaultConfirmationThreshold = 6
	defaultFinalityThreshold     = 12
	defaultBlockInterval         = 3 // 每 N tick 自动出块
	defaultMaxBlockSize          = 4 // 每块最多打包 N 笔

	maxTxs = 32

	phaseCreated   = "created"
	phaseMempool   = "mempool"
	phaseIncluded  = "included"
	phaseConfirmed = "confirmed"
	phaseFinalized = "finalized"
	phaseDropped   = "dropped"
	phaseReplaced  = "replaced"

	linkGroupTxProc  = "tx-processing-group"
	linkOwnerSubtree = "tx.lifecycle"
)

var phaseList = []string{phaseCreated, phaseMempool, phaseIncluded, phaseConfirmed, phaseFinalized}

type tx struct {
	ID            string
	From          string
	To            string
	Value         int64
	Gas           int
	Nonce         int
	Phase         string
	CreatedAt     int
	MempoolAt     int
	IncludedAt    int // 进入 block 的 tick
	BlockHeight   int // included block height
	Confirmations int // 当前确认数
	ConfirmedAt   int
	FinalizedAt   int
	DroppedAt     int
	ReplacedBy    string
}

type bblock struct {
	Height int
	Tick   int
	TxIDs  []string
}

type snapState struct {
	Tick                  int
	BlockHeight           int
	BlockInterval         int
	ConfirmationThreshold int
	FinalityThreshold     int
	MaxBlockSize          int
	Txs                   []tx
	Blocks                []bblock
	NextNonce             map[string]int // from → next nonce
	LastError             string
}

func defaultSnapState() snapState {
	return snapState{
		BlockInterval:         defaultBlockInterval,
		ConfirmationThreshold: defaultConfirmationThreshold,
		FinalityThreshold:     defaultFinalityThreshold,
		MaxBlockSize:          defaultMaxBlockSize,
		NextNonce:             map[string]int{},
	}
}

// findTx 按 ID 找 tx 指针。
func (st *snapState) findTx(id string) *tx {
	for i := range st.Txs {
		if st.Txs[i].ID == id {
			return &st.Txs[i]
		}
	}
	return nil
}

// submitTx 创建新 tx 进入 mempool（一步完成 created → mempool）。
func (st *snapState) submitTx(from, to string, value int64, gas int) (*tx, error) {
	if len(st.Txs) >= maxTxs {
		return nil, fmt.Errorf("tx 总数 ≥ %d", maxTxs)
	}
	t := tx{
		ID:   fmt.Sprintf("tx%d", len(st.Txs)),
		From: from, To: to, Value: value, Gas: gas,
		Nonce:     st.NextNonce[from],
		Phase:     phaseMempool,
		CreatedAt: st.Tick,
		MempoolAt: st.Tick,
	}
	st.NextNonce[from]++
	st.Txs = append(st.Txs, t)
	return &st.Txs[len(st.Txs)-1], nil
}

// mineBlock 把 mempool 中按 gas 降序的 N 个 tx 打包入新区块。
func (st *snapState) mineBlock() (bblock, error) {
	candidates := []int{}
	for i, t := range st.Txs {
		if t.Phase == phaseMempool {
			candidates = append(candidates, i)
		}
	}
	sort.Slice(candidates, func(a, b int) bool {
		return st.Txs[candidates[a]].Gas > st.Txs[candidates[b]].Gas
	})
	if len(candidates) > st.MaxBlockSize {
		candidates = candidates[:st.MaxBlockSize]
	}
	blk := bblock{Height: st.BlockHeight, Tick: st.Tick}
	st.BlockHeight++
	for _, idx := range candidates {
		st.Txs[idx].Phase = phaseIncluded
		st.Txs[idx].IncludedAt = st.Tick
		st.Txs[idx].BlockHeight = blk.Height
		blk.TxIDs = append(blk.TxIDs, st.Txs[idx].ID)
	}
	st.Blocks = append(st.Blocks, blk)
	if len(st.Blocks) > 32 {
		st.Blocks = st.Blocks[len(st.Blocks)-32:]
	}
	// 推进所有 included tx 的 confirmations
	st.advanceConfirmations()
	return blk, nil
}

// advanceConfirmations 每出一个新块，所有 included 的 tx confirmations++。
func (st *snapState) advanceConfirmations() {
	for i := range st.Txs {
		t := &st.Txs[i]
		switch t.Phase {
		case phaseIncluded, phaseConfirmed:
			t.Confirmations = st.BlockHeight - t.BlockHeight
			if t.Phase == phaseIncluded && t.Confirmations >= st.ConfirmationThreshold {
				t.Phase = phaseConfirmed
				t.ConfirmedAt = st.Tick
			}
			if t.Phase == phaseConfirmed && t.Confirmations >= st.FinalityThreshold {
				t.Phase = phaseFinalized
				t.FinalizedAt = st.Tick
			}
		}
	}
}

// stepTick 推进 1 tick，按 BlockInterval 自动出块。
func (st *snapState) stepTick() (mined bool) {
	st.Tick++
	if st.Tick%st.BlockInterval == 0 {
		st.mineBlock()
		mined = true
	}
	return
}

// dropTx 把 mempool 中的 tx 标记为 dropped。
func (st *snapState) dropTx(id string) error {
	t := st.findTx(id)
	if t == nil {
		return fmt.Errorf("未找到 tx: %s", id)
	}
	if t.Phase != phaseMempool {
		return fmt.Errorf("%s 不在 mempool（当前 %s）", id, t.Phase)
	}
	t.Phase = phaseDropped
	t.DroppedAt = st.Tick
	return nil
}

// replaceTx 在相同 (from, nonce) 上提交更高 gas 的新 tx，原 tx 标 replaced。
func (st *snapState) replaceTx(from string, nonce int, newGas int) (*tx, error) {
	var oldT *tx
	for i := range st.Txs {
		if st.Txs[i].From == from && st.Txs[i].Nonce == nonce && st.Txs[i].Phase == phaseMempool {
			oldT = &st.Txs[i]
			break
		}
	}
	if oldT == nil {
		return nil, fmt.Errorf("(%s, nonce=%d) 在 mempool 中未找到", from, nonce)
	}
	if newGas <= oldT.Gas {
		return nil, fmt.Errorf("新 gas %d 必须 > 原 gas %d", newGas, oldT.Gas)
	}
	if len(st.Txs) >= maxTxs {
		return nil, fmt.Errorf("tx 总数 ≥ %d", maxTxs)
	}
	newT := tx{
		ID:   fmt.Sprintf("tx%d", len(st.Txs)),
		From: from, To: oldT.To, Value: oldT.Value,
		Gas: newGas, Nonce: nonce,
		Phase:     phaseMempool,
		CreatedAt: st.Tick,
		MempoolAt: st.Tick,
	}
	st.Txs = append(st.Txs, newT)
	oldT.Phase = phaseReplaced
	oldT.ReplacedBy = newT.ID
	return &st.Txs[len(st.Txs)-1], nil
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
		Tick:                  fw.MapInt(d, "tick", 0),
		BlockHeight:           fw.MapInt(d, "block_height", 0),
		BlockInterval:         fw.MapInt(d, "block_interval", defaultBlockInterval),
		ConfirmationThreshold: fw.MapInt(d, "conf_th", defaultConfirmationThreshold),
		FinalityThreshold:     fw.MapInt(d, "fin_th", defaultFinalityThreshold),
		MaxBlockSize:          fw.MapInt(d, "max_block_size", defaultMaxBlockSize),
		LastError:             fw.MapStr(d, "last_error", ""),
		NextNonce:             map[string]int{},
	}
	if txsAny, ok := d["txs"].([]any); ok {
		for _, tAny := range txsAny {
			if tm, ok := tAny.(map[string]any); ok {
				st.Txs = append(st.Txs, tx{
					ID:            fw.MapStr(tm, "id", ""),
					From:          fw.MapStr(tm, "from", ""),
					To:            fw.MapStr(tm, "to", ""),
					Value:         int64(fw.MapInt(tm, "value", 0)),
					Gas:           fw.MapInt(tm, "gas", 0),
					Nonce:         fw.MapInt(tm, "nonce", 0),
					Phase:         fw.MapStr(tm, "phase", ""),
					CreatedAt:     fw.MapInt(tm, "created", 0),
					MempoolAt:     fw.MapInt(tm, "mempool_at", 0),
					IncludedAt:    fw.MapInt(tm, "included_at", 0),
					BlockHeight:   fw.MapInt(tm, "block", 0),
					Confirmations: fw.MapInt(tm, "confirmations", 0),
					ConfirmedAt:   fw.MapInt(tm, "confirmed_at", 0),
					FinalizedAt:   fw.MapInt(tm, "finalized_at", 0),
					DroppedAt:     fw.MapInt(tm, "dropped_at", 0),
					ReplacedBy:    fw.MapStr(tm, "replaced_by", ""),
				})
			}
		}
	}
	if bsAny, ok := d["blocks"].([]any); ok {
		for _, bAny := range bsAny {
			if bm, ok := bAny.(map[string]any); ok {
				blk := bblock{
					Height: fw.MapInt(bm, "h", 0),
					Tick:   fw.MapInt(bm, "tick", 0),
				}
				if txAny, ok := bm["txs"].([]any); ok {
					for _, t := range txAny {
						if s, ok := t.(string); ok {
							blk.TxIDs = append(blk.TxIDs, s)
						}
					}
				}
				st.Blocks = append(st.Blocks, blk)
			}
		}
	}
	if nnAny, ok := d["next_nonce"].(map[string]any); ok {
		for k, v := range nnAny {
			st.NextNonce[k] = intFromAny(v)
		}
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["tick"] = st.Tick
	s.Data["block_height"] = st.BlockHeight
	s.Data["block_interval"] = st.BlockInterval
	s.Data["conf_th"] = st.ConfirmationThreshold
	s.Data["fin_th"] = st.FinalityThreshold
	s.Data["max_block_size"] = st.MaxBlockSize
	s.Data["last_error"] = st.LastError
	txsAny := make([]any, len(st.Txs))
	for i, t := range st.Txs {
		txsAny[i] = map[string]any{
			"id": t.ID, "from": t.From, "to": t.To,
			"value": int(t.Value), "gas": t.Gas, "nonce": t.Nonce,
			"phase": t.Phase, "created": t.CreatedAt, "mempool_at": t.MempoolAt,
			"included_at": t.IncludedAt, "block": t.BlockHeight,
			"confirmations": t.Confirmations, "confirmed_at": t.ConfirmedAt,
			"finalized_at": t.FinalizedAt, "dropped_at": t.DroppedAt,
			"replaced_by": t.ReplacedBy,
		}
	}
	s.Data["txs"] = txsAny
	bsAny := make([]any, len(st.Blocks))
	for i, b := range st.Blocks {
		txs := make([]any, len(b.TxIDs))
		for j, id := range b.TxIDs {
			txs[j] = id
		}
		bsAny[i] = map[string]any{"h": b.Height, "tick": b.Tick, "txs": txs}
	}
	s.Data["blocks"] = bsAny
	nnAny := map[string]any{}
	for k, v := range st.NextNonce {
		nnAny[k] = v
	}
	s.Data["next_nonce"] = nnAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "交易生命周期",
		Description:         "演示 tx 5 阶段：created → mempool → included → confirmed → finalized；含 drop / replace / 自动出块",
		Category:            fw.CategoryTransaction,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupTxProc},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"tx.lifecycle.mempool_count",
			"tx.lifecycle.block_height",
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
				ActionCode: "set_params", Label: "设置生命周期参数",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "block_interval", Type: fw.FieldNumber, Label: "出块间隔 (tick)", Required: true, Default: defaultBlockInterval, Min: 1, Max: 10, Step: 1},
					{Name: "conf_th", Type: fw.FieldNumber, Label: "确认门槛", Required: true, Default: defaultConfirmationThreshold, Min: 1, Max: 30, Step: 1},
					{Name: "fin_th", Type: fw.FieldNumber, Label: "终结门槛", Required: true, Default: defaultFinalityThreshold, Min: 2, Max: 100, Step: 1},
					{Name: "max_block_size", Type: fw.FieldNumber, Label: "每块最多 tx 数", Required: true, Default: defaultMaxBlockSize, Min: 1, Max: 16, Step: 1},
				},
			},
			{
				ActionCode: "submit_tx", Label: "提交交易",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "alice"},
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "bob"},
					{Name: "value", Type: fw.FieldNumber, Label: "value", Required: true, Default: 100, Min: 1, Step: 1},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: 50, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"tx.lifecycle.mempool_count"},
				LinkOwnerFields:   []string{"tx.lifecycle.mempool_count"},
			},
			{
				ActionCode: "mine_block", Label: "立即出块",
				Description: "把 mempool 按 gas 降序前 N 笔打包",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				WritesOwnedFields: []string{"tx.lifecycle.block_height"},
				LinkOwnerFields:   []string{"tx.lifecycle.block_height"},
			},
			{
				ActionCode: "step_tick", Label: "推进 1 tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode: "step_n_ticks", Label: "推进 N tick",
				Category: fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "tick 数", Required: true, Default: 12, Min: 1, Max: 100, Step: 1},
				},
			},
			{
				ActionCode: "drop_tx", Label: "丢弃 mempool 中的 tx",
				Category: fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "tx_id", Type: fw.FieldString, Label: "tx ID", Required: true, Default: "tx0"},
				},
			},
			{
				ActionCode: "replace_tx", Label: "RBF 替换交易",
				Description: "用相同 (from, nonce) 但更高 gas 替换",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "alice"},
					{Name: "nonce", Type: fw.FieldNumber, Label: "nonce", Required: true, Default: 0, Min: 0, Step: 1},
					{Name: "new_gas", Type: fw.FieldNumber, Label: "新 gas（须 > 原 gas）", Required: true, Default: 200, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode:    "teacher_freeze_mempool",
				Label:         "教师冻结内存池",
				Description:   "仅教师可用，冻结内存池用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师冻结内存池"},
				},
			},
			{
				ActionCode:    "send_real_tx",
				Label:         "发送真实交易",
				Description:   "调 geth eth_sendRawTransaction 发送签名交易",
				Category:      fw.ActionPrimary,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				HybridChannel: fw.HybridChannelContainer,
				ContainerCmd:  `curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["{{raw_tx}}"],"id":1}' http://geth:8545`,
				Reversible:    false,
				Fields: []fw.FieldDef{
					{Name: "raw_tx", Type: fw.FieldString, Label: "signed raw tx (hex)", Required: true, Default: "0x"},
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
	env := buildEnvelope(st, "init", "Tx Lifecycle 初始化（block_interval=3）", true)
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
		st.BlockInterval = fw.MapInt(in.Params, "block_interval", defaultBlockInterval)
		st.ConfirmationThreshold = fw.MapInt(in.Params, "conf_th", defaultConfirmationThreshold)
		st.FinalityThreshold = fw.MapInt(in.Params, "fin_th", defaultFinalityThreshold)
		st.MaxBlockSize = fw.MapInt(in.Params, "max_block_size", defaultMaxBlockSize)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params",
			fmt.Sprintf("interval=%d conf=%d fin=%d max=%d",
				st.BlockInterval, st.ConfirmationThreshold, st.FinalityThreshold, st.MaxBlockSize), false)
		return out, nil

	case "submit_tx":
		from := fw.MapStr(in.Params, "from", "alice")
		to := fw.MapStr(in.Params, "to", "bob")
		value := int64(fw.MapInt(in.Params, "value", 100))
		gas := fw.MapInt(in.Params, "gas", 50)
		t, err := st.submitTx(from, to, value, gas)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "submit_tx",
			fmt.Sprintf("%s = %s → %s (val=%d, gas=%d) 进入 mempool", t.ID, from, to, value, gas), false)
		appendSubmitMicroSteps(&out.Render, t.ID)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "mine_block":
		blk, err := st.mineBlock()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "mine_block",
			fmt.Sprintf("出块 #%d 含 %d 笔 tx", blk.Height, len(blk.TxIDs)), false)
		appendMineMicroSteps(&out.Render, blk.Height, len(blk.TxIDs))
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_tick":
		mined := st.stepTick()
		saveState(state, st)
		summary := fmt.Sprintf("tick=%d", st.Tick)
		if mined {
			summary += fmt.Sprintf(" (auto-mine #%d)", st.BlockHeight-1)
		}
		out.Render = buildEnvelope(st, "step_tick", summary, false)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "step_n_ticks":
		n := fw.MapInt(in.Params, "n", 12)
		mined := 0
		for i := 0; i < n; i++ {
			if st.stepTick() {
				mined++
			}
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "step_n_ticks",
			fmt.Sprintf("推进 %d tick → 自动出块 %d 次", n, mined), false)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "drop_tx":
		id := fw.MapStr(in.Params, "tx_id", "tx0")
		if err := st.dropTx(id); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "drop_tx", id+" 被丢弃", false)
		appendDropMicroSteps(&out.Render, id)
		return out, nil

	case "replace_tx":
		from := fw.MapStr(in.Params, "from", "alice")
		nonce := fw.MapInt(in.Params, "nonce", 0)
		newGas := fw.MapInt(in.Params, "new_gas", 200)
		newT, err := st.replaceTx(from, nonce, newGas)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "replace_tx",
			fmt.Sprintf("RBF: 用 %s (gas=%d) 替换 (from=%s, nonce=%d)", newT.ID, newGas, from, nonce), false)
		appendReplaceMicroSteps(&out.Render, newT.ID)
		return out, nil

	case "teacher_freeze_mempool":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师冻结内存池"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-freeze-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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

	// 1) 5 阶段流水线
	phaseIDs := make([]string, len(phaseList))
	for i, p := range phaseList {
		phaseIDs[i] = "phase-" + p
	}
	prims = append(prims, fw.PrimStack("phase-stack", phaseIDs, "horizontal"))

	// 统计每个阶段的 tx 数
	counts := map[string]int{}
	for _, t := range st.Txs {
		counts[t.Phase]++
	}
	for i, p := range phaseList {
		role := p
		status := "normal"
		if counts[p] > 0 {
			status = "active"
		}
		label := fmt.Sprintf("%s\n%d", p, counts[p])
		prims = append(prims, fw.PrimNode(phaseIDs[i], label, status, role))
	}
	for i := 0; i < len(phaseList)-1; i++ {
		anim := "flow"
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ph-edge-%d", i), phaseIDs[i], phaseIDs[i+1], "solid", anim))
	}

	// 2) phase_progress
	maxPhase := 0
	for i, p := range phaseList {
		if counts[p] > 0 {
			maxPhase = i
		}
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseList, maxPhase, float64(maxPhase)/float64(len(phaseList)-1)))

	// 3) 公式
	prims = append(prims, fw.PrimMathFormula("formula-conf",
		`\text{confirmations} = h_{\text{tip}} - h_{\text{tx}};\quad \text{confirmed} \iff c \ge T_c`, false))

	// 4) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("tick = %d\nblock_height = %d\nblock_interval = %d\nconf_threshold = %d\nfin_threshold = %d\nmax_block_size = %d\ntx 总数 = %d",
			st.Tick, st.BlockHeight, st.BlockInterval,
			st.ConfirmationThreshold, st.FinalityThreshold,
			st.MaxBlockSize, len(st.Txs)),
		"text", nil, 8))

	// 5) tx 表
	rows := []string{"id    from   to     val   gas   nonce  phase       block  conf"}
	startIdx := 0
	if len(st.Txs) > 16 {
		startIdx = len(st.Txs) - 16
	}
	for _, t := range st.Txs[startIdx:] {
		rows = append(rows, fmt.Sprintf("%-5s %-6s %-6s %-5d %-5d %-6d %-11s %-6d %d",
			t.ID, t.From, t.To, t.Value, t.Gas, t.Nonce,
			t.Phase, t.BlockHeight, t.Confirmations))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-txs", strings.Join(rows, "\n"), "text", nil, 18))

	// 6) 区块表
	if len(st.Blocks) > 0 {
		bRows := []string{"#    tick  txs"}
		startBI := 0
		if len(st.Blocks) > 12 {
			startBI = len(st.Blocks) - 12
		}
		for _, b := range st.Blocks[startBI:] {
			bRows = append(bRows, fmt.Sprintf("%-4d %-5d [%s]", b.Height, b.Tick, strings.Join(b.TxIDs, ", ")))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-blocks", strings.Join(bRows, "\n"), "text", nil, 14))
	}

	// 7) 进度条：mempool / confirmed / finalized
	prims = append(prims, fw.PrimProgressBar("bar-mempool", float64(counts[phaseMempool]), float64(maxInt(counts[phaseMempool], 4)),
		fmt.Sprintf("mempool %d", counts[phaseMempool])))
	prims = append(prims, fw.PrimProgressBar("bar-confirmed", float64(counts[phaseConfirmed]+counts[phaseFinalized]), float64(len(st.Txs)),
		fmt.Sprintf("confirmed+finalized %d", counts[phaseConfirmed]+counts[phaseFinalized])))

	// 8) 动效
	for i, p := range phaseList {
		if counts[p] > 0 {
			color := "info"
			if p == phaseFinalized {
				color = "success"
			} else if p == phaseConfirmed {
				color = "success"
			} else if p == phaseMempool {
				color = "warning"
			}
			prims = append(prims, fw.PrimGlow("glow-"+p, phaseIDs[i], color, 0.7))
		}
	}
	if st.Tick > 0 && st.Tick%st.BlockInterval == 0 {
		prims = append(prims, fw.PrimBurst("burst-block", phaseIDs[2], "success", int64(st.BlockHeight), 700))
	}

	// 9) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-tx", linkGroupTxProc, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Tx 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
		ContainerData: []fw.ContainerMetric{
			{SourceContainer: "geth", MetricKey: "tx.pending_count", Value: len(st.Txs), TargetPrimitive: "cb-mempool", TargetParam: "count"},
		},
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	counts := map[string]int{}
	for _, t := range st.Txs {
		counts[t.Phase]++
	}
	d := map[string]any{
		"tick":            st.Tick,
		"block_height":    st.BlockHeight,
		"block_interval":  st.BlockInterval,
		"conf_threshold":  st.ConfirmationThreshold,
		"fin_threshold":   st.FinalityThreshold,
		"total_tx":        len(st.Txs),
		"mempool_count":   counts[phaseMempool],
		"included_count":  counts[phaseIncluded],
		"confirmed_count": counts[phaseConfirmed],
		"finalized_count": counts[phaseFinalized],
		"dropped_count":   counts[phaseDropped],
		"replaced_count":  counts[phaseReplaced],
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendSubmitMicroSteps(env *fw.RenderEnvelope, txID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sb-1", Label: txID + " 进入 created", DurationMs: 400, HighlightIDs: []string{"phase-created"}},
		{ID: "sb-2", Label: "广播到 mempool", DurationMs: 400, HighlightIDs: []string{"phase-mempool", "bar-mempool"}, FirePrimitives: []string{"glow-mempool"}},
		{ID: "sb-3", Label: "等待出块", DurationMs: 400, HighlightIDs: []string{"cb-txs"}, IsLinkTrigger: true},
	}
}

func appendMineMicroSteps(env *fw.RenderEnvelope, h, n int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "mn-1", Label: "mempool 按 gas 降序选 N 笔", DurationMs: 400, HighlightIDs: []string{"phase-mempool", "cb-txs"}},
		{ID: "mn-2", Label: fmt.Sprintf("打包入块 #%d", h), DurationMs: 500, HighlightIDs: []string{"phase-included", "cb-blocks"}, FirePrimitives: []string{"burst-block"}},
		{ID: "mn-3", Label: "所有已 included tx confirmations++", DurationMs: 500, HighlightIDs: []string{"formula-conf", "cb-status"}, IsLinkTrigger: true},
	}
}

func appendDropMicroSteps(env *fw.RenderEnvelope, id string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "dr-1", Label: id + " 被节点 / 用户主动丢弃", DurationMs: 400, HighlightIDs: []string{"cb-txs"}},
		{ID: "dr-2", Label: "phase 转 dropped，不会再被打包", DurationMs: 500, HighlightIDs: []string{"phase-mempool"}},
	}
}

func appendReplaceMicroSteps(env *fw.RenderEnvelope, newID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "rp-1", Label: "提交相同 (from, nonce) 但 gas 更高的新 tx", DurationMs: 400, HighlightIDs: []string{"cb-txs"}},
		{ID: "rp-2", Label: "原 tx 标 replaced，新 tx 进入 mempool", DurationMs: 500, HighlightIDs: []string{"phase-mempool", "bar-mempool"}, FirePrimitives: []string{"glow-mempool"}},
		{ID: "rp-3", Label: newID + " 排在 mempool 前面（gas 更高）", DurationMs: 400, HighlightIDs: []string{"cb-txs"}, IsLinkTrigger: true},
	}
}

// =====================================================================
// 联动
// =====================================================================

func txLifecyclePendingCount(st snapState) int {
	cnt := 0
	for _, t := range st.Txs {
		if t.Phase == "mempool" {
			cnt++
		}
	}
	return cnt
}

func publishOwnerSubtree(env *fw.RenderEnvelope, st snapState) {
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	env.LinkTriggers = append(env.LinkTriggers, fw.LinkTrigger{
		ID:             "tx-lifecycle-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_tx",
		LinkGroup:      linkGroupTxProc,
		ChangedFields:  []string{"tx.lifecycle.mempool_count", "tx.lifecycle.block_height"},
		Payload:        map[string]any{"mempool": txLifecyclePendingCount(st), "height": st.BlockHeight},
		SourceAnchorID: "tx-lifecycle-anchor",
		TargetAnchorID: "tx-proc-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "tx.lifecycle.mempool_count", "tx.lifecycle.block_height")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	counts := map[string]int{}
	for _, t := range st.Txs {
		counts[t.Phase]++
	}
	return map[string]any{
		"tx": map[string]any{
			"lifecycle": map[string]any{
				"tick":            st.Tick,
				"block_height":    st.BlockHeight,
				"total_tx":        len(st.Txs),
				"mempool_count":   counts[phaseMempool],
				"included_count":  counts[phaseIncluded],
				"confirmed_count": counts[phaseConfirmed],
				"finalized_count": counts[phaseFinalized],
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
