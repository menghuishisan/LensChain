// 模块：sim-engine/scenarios/internal/datastructure/blockinternal
// 文件职责：DS-02 区块内部结构（header + body + 三 Merkle root）场景的完整实现。
//
// SSOT 依据：06.md §4.4.2 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现以太坊风格区块的内部结构（零外部依赖）：
//   · Header: prev_hash / tx_root / state_root / receipts_root / ts / difficulty /
//             nonce / miner / gas_used / gas_limit
//   · Body: transactions[] + receipts[]
//   · tx_root / receipts_root：用 SHA-256 二叉 Merkle 树（sha256hash.Sum256）
//   · state_root：基于已应用 tx 的状态映射 + Merkle 化
//   · header_hash = SHA-256(serialize(header))
//   · 篡改演示：篡改某 tx 后不重算 root → verify 失败

package blockinternal

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

const (
	sceneCode     = "block-internal"
	schemaVersion = "v1.0.0"
	algorithmType = "block-internal"

	maxTxs = 16

	linkGroupBlockchainIntegr = "blockchain-integrity-group"
	linkOwnerSubtree          = "chain.block_internal"
)

type tx struct {
	ID    string
	From  string
	To    string
	Value int64
	Gas   int
	Nonce int
}

type receipt struct {
	TxID    string
	Success bool
	GasUsed int
	Logs    int
}

// merkleRoot 用 SHA-256 自底向上构造 Merkle 树根（奇数节点 duplicate）。
func merkleRoot(items [][]byte) [32]byte {
	if len(items) == 0 {
		return [32]byte{}
	}
	level := make([][32]byte, len(items))
	for i, b := range items {
		level[i] = sha256hash.Sum256(b)
	}
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			var buf [64]byte
			copy(buf[:32], left[:])
			copy(buf[32:], right[:])
			next = append(next, sha256hash.Sum256(buf[:]))
		}
		level = next
	}
	return level[0]
}

// serializeTx 序列化 tx 为字节流。
func serializeTx(t tx) []byte {
	buf := []byte(t.ID + "|" + t.From + "|" + t.To)
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], uint64(t.Value))
	buf = append(buf, v[:]...)
	var g [4]byte
	binary.BigEndian.PutUint32(g[:], uint32(t.Gas))
	buf = append(buf, g[:]...)
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(t.Nonce))
	buf = append(buf, n[:]...)
	return buf
}

// serializeReceipt 序列化 receipt。
func serializeReceipt(r receipt) []byte {
	buf := []byte(r.TxID)
	if r.Success {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	var g [4]byte
	binary.BigEndian.PutUint32(g[:], uint32(r.GasUsed))
	buf = append(buf, g[:]...)
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(r.Logs))
	buf = append(buf, l[:]...)
	return buf
}

// computeStateRoot 基于"账户余额"映射构造 state Merkle root（教学：simple kv root）。
func computeStateRoot(state map[string]int64) [32]byte {
	keys := make([]string, 0, len(state))
	for k := range state {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([][]byte, 0, len(keys))
	for _, k := range keys {
		buf := []byte(k)
		var v [8]byte
		binary.BigEndian.PutUint64(v[:], uint64(state[k]))
		buf = append(buf, v[:]...)
		items = append(items, buf)
	}
	if len(items) == 0 {
		return [32]byte{}
	}
	return merkleRoot(items)
}

// applyTxToState 把 tx 应用到 state（from 扣 value+gas，to 加 value）。
func applyTxToState(state map[string]int64, t tx) (success bool, gasUsed int) {
	cost := t.Value + int64(t.Gas)
	if state[t.From] < cost {
		return false, t.Gas / 2 // 失败也扣半 gas
	}
	state[t.From] -= cost
	state[t.To] += t.Value
	return true, t.Gas
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	PrevHash     string
	Timestamp    uint32
	Difficulty   int
	Nonce        uint64
	Miner        string
	GasLimit     int
	Txs          []tx
	Receipts     []receipt
	State        map[string]int64
	TxRoot       string
	ReceiptsRoot string
	StateRoot    string
	HeaderHash   string
	GasUsed      int
	Tampered     bool
	TamperedTxID string
	LastError    string
}

func defaultSnapState() snapState {
	return snapState{
		PrevHash:   strings.Repeat("0", 64),
		Timestamp:  1700000000,
		Difficulty: 100,
		Miner:      "alice",
		GasLimit:   1000,
		State: map[string]int64{
			"alice":   1000,
			"bob":     500,
			"carol":   500,
			"dave":    100,
			"genesis": 10000,
		},
	}
}

// computeAllRoots 重算 tx_root / receipts_root / state_root / header_hash。
func (st *snapState) computeAllRoots() {
	txItems := make([][]byte, len(st.Txs))
	for i, t := range st.Txs {
		txItems[i] = serializeTx(t)
	}
	tr := merkleRoot(txItems)
	st.TxRoot = hex.EncodeToString(tr[:])

	rcItems := make([][]byte, len(st.Receipts))
	for i, r := range st.Receipts {
		rcItems[i] = serializeReceipt(r)
	}
	rr := merkleRoot(rcItems)
	st.ReceiptsRoot = hex.EncodeToString(rr[:])

	sr := computeStateRoot(st.State)
	st.StateRoot = hex.EncodeToString(sr[:])

	st.HeaderHash = hex.EncodeToString(st.computeHeaderHash())
}

// computeHeaderHash 序列化 header 后 SHA-256。
func (st snapState) computeHeaderHash() []byte {
	buf := []byte{}
	prev, _ := hex.DecodeString(st.PrevHash)
	if len(prev) < 32 {
		prev = append(prev, make([]byte, 32-len(prev))...)
	}
	buf = append(buf, prev[:32]...)
	tr, _ := hex.DecodeString(st.TxRoot)
	if len(tr) < 32 {
		tr = append(tr, make([]byte, 32-len(tr))...)
	}
	buf = append(buf, tr[:32]...)
	rr, _ := hex.DecodeString(st.ReceiptsRoot)
	if len(rr) < 32 {
		rr = append(rr, make([]byte, 32-len(rr))...)
	}
	buf = append(buf, rr[:32]...)
	sr, _ := hex.DecodeString(st.StateRoot)
	if len(sr) < 32 {
		sr = append(sr, make([]byte, 32-len(sr))...)
	}
	buf = append(buf, sr[:32]...)
	var ts [4]byte
	binary.BigEndian.PutUint32(ts[:], st.Timestamp)
	buf = append(buf, ts[:]...)
	var diff [4]byte
	binary.BigEndian.PutUint32(diff[:], uint32(st.Difficulty))
	buf = append(buf, diff[:]...)
	var nonce [8]byte
	binary.BigEndian.PutUint64(nonce[:], st.Nonce)
	buf = append(buf, nonce[:]...)
	buf = append(buf, []byte(st.Miner)...)
	var gu [4]byte
	binary.BigEndian.PutUint32(gu[:], uint32(st.GasUsed))
	buf = append(buf, gu[:]...)
	var gl [4]byte
	binary.BigEndian.PutUint32(gl[:], uint32(st.GasLimit))
	buf = append(buf, gl[:]...)
	h := sha256hash.Sum256(buf)
	return h[:]
}

// addTxAndApply 添加 tx，应用到 state，生成 receipt。
func (st *snapState) addTxAndApply(t tx) error {
	if len(st.Txs) >= maxTxs {
		return fmt.Errorf("tx 数 ≥ %d", maxTxs)
	}
	if st.GasUsed+t.Gas > st.GasLimit {
		return fmt.Errorf("超出 gas_limit (%d + %d > %d)", st.GasUsed, t.Gas, st.GasLimit)
	}
	success, gasUsed := applyTxToState(st.State, t)
	st.Txs = append(st.Txs, t)
	st.Receipts = append(st.Receipts, receipt{
		TxID: t.ID, Success: success, GasUsed: gasUsed, Logs: 0,
	})
	st.GasUsed += gasUsed
	st.computeAllRoots()
	return nil
}

// verifyIntegrity 重算 root 与字段比较；篡改 tx 后字段不变 → 重算的 root 与字段不一致。
type integrityViolation struct {
	Field    string
	Expected string
	Actual   string
}

func (st snapState) verifyIntegrity() []integrityViolation {
	violations := []integrityViolation{}

	txItems := make([][]byte, len(st.Txs))
	for i, t := range st.Txs {
		txItems[i] = serializeTx(t)
	}
	expectedTxRoot := merkleRoot(txItems)
	expectedTxRootHex := hex.EncodeToString(expectedTxRoot[:])
	if expectedTxRootHex != st.TxRoot {
		violations = append(violations, integrityViolation{
			Field: "tx_root", Expected: expectedTxRootHex, Actual: st.TxRoot,
		})
	}

	rcItems := make([][]byte, len(st.Receipts))
	for i, r := range st.Receipts {
		rcItems[i] = serializeReceipt(r)
	}
	expectedRR := merkleRoot(rcItems)
	expectedRRHex := hex.EncodeToString(expectedRR[:])
	if expectedRRHex != st.ReceiptsRoot {
		violations = append(violations, integrityViolation{
			Field: "receipts_root", Expected: expectedRRHex, Actual: st.ReceiptsRoot,
		})
	}

	expectedSR := computeStateRoot(st.State)
	expectedSRHex := hex.EncodeToString(expectedSR[:])
	if expectedSRHex != st.StateRoot {
		violations = append(violations, integrityViolation{
			Field: "state_root", Expected: expectedSRHex, Actual: st.StateRoot,
		})
	}

	expectedHH := hex.EncodeToString(st.computeHeaderHash())
	if expectedHH != st.HeaderHash {
		violations = append(violations, integrityViolation{
			Field: "header_hash", Expected: expectedHH, Actual: st.HeaderHash,
		})
	}
	return violations
}

// =====================================================================
// 持久化
// =====================================================================

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		st := defaultSnapState()
		st.computeAllRoots()
		return st
	}
	d := s.Data
	st := snapState{
		PrevHash:     fw.MapStr(d, "prev_hash", strings.Repeat("0", 64)),
		Timestamp:    uint32(fw.MapInt(d, "ts", 1700000000)),
		Difficulty:   fw.MapInt(d, "difficulty", 100),
		Nonce:        uint64(fw.MapInt(d, "nonce", 0)),
		Miner:        fw.MapStr(d, "miner", "alice"),
		GasLimit:     fw.MapInt(d, "gas_limit", 1000),
		GasUsed:      fw.MapInt(d, "gas_used", 0),
		TxRoot:       fw.MapStr(d, "tx_root", ""),
		ReceiptsRoot: fw.MapStr(d, "receipts_root", ""),
		StateRoot:    fw.MapStr(d, "state_root", ""),
		HeaderHash:   fw.MapStr(d, "header_hash", ""),
		Tampered:     fw.MapBool(d, "tampered", false),
		TamperedTxID: fw.MapStr(d, "tampered_tx_id", ""),
		LastError:    fw.MapStr(d, "last_error", ""),
		State:        map[string]int64{},
	}
	if txsAny, ok := d["txs"].([]any); ok {
		for _, tAny := range txsAny {
			if tm, ok := tAny.(map[string]any); ok {
				st.Txs = append(st.Txs, tx{
					ID:    fw.MapStr(tm, "id", ""),
					From:  fw.MapStr(tm, "from", ""),
					To:    fw.MapStr(tm, "to", ""),
					Value: int64(fw.MapInt(tm, "value", 0)),
					Gas:   fw.MapInt(tm, "gas", 0),
					Nonce: fw.MapInt(tm, "nonce", 0),
				})
			}
		}
	}
	if rcsAny, ok := d["receipts"].([]any); ok {
		for _, rAny := range rcsAny {
			if rm, ok := rAny.(map[string]any); ok {
				st.Receipts = append(st.Receipts, receipt{
					TxID:    fw.MapStr(rm, "tx_id", ""),
					Success: fw.MapBool(rm, "success", false),
					GasUsed: fw.MapInt(rm, "gas_used", 0),
					Logs:    fw.MapInt(rm, "logs", 0),
				})
			}
		}
	}
	if stAny, ok := d["state"].(map[string]any); ok {
		for k, v := range stAny {
			st.State[k] = int64(intFromAny(v))
		}
	}
	if len(st.State) == 0 {
		st.State = defaultSnapState().State
	}
	return st
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["prev_hash"] = st.PrevHash
	s.Data["ts"] = int(st.Timestamp)
	s.Data["difficulty"] = st.Difficulty
	s.Data["nonce"] = int(st.Nonce)
	s.Data["miner"] = st.Miner
	s.Data["gas_limit"] = st.GasLimit
	s.Data["gas_used"] = st.GasUsed
	s.Data["tx_root"] = st.TxRoot
	s.Data["receipts_root"] = st.ReceiptsRoot
	s.Data["state_root"] = st.StateRoot
	s.Data["header_hash"] = st.HeaderHash
	s.Data["tampered"] = st.Tampered
	s.Data["tampered_tx_id"] = st.TamperedTxID
	s.Data["last_error"] = st.LastError
	txsAny := make([]any, len(st.Txs))
	for i, t := range st.Txs {
		txsAny[i] = map[string]any{
			"id": t.ID, "from": t.From, "to": t.To,
			"value": int(t.Value), "gas": t.Gas, "nonce": t.Nonce,
		}
	}
	s.Data["txs"] = txsAny
	rcsAny := make([]any, len(st.Receipts))
	for i, r := range st.Receipts {
		rcsAny[i] = map[string]any{
			"tx_id": r.TxID, "success": r.Success,
			"gas_used": r.GasUsed, "logs": r.Logs,
		}
	}
	s.Data["receipts"] = rcsAny
	stAny := map[string]any{}
	for k, v := range st.State {
		stAny[k] = int(v)
	}
	s.Data["state"] = stAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "区块内部结构",
		Description:         "演示以太坊风格区块的 header / body / 三 Merkle root（tx / state / receipts）+ 篡改检测",
		Category:            fw.CategoryDataStructure,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupBlockchainIntegr},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"chain.block_internal.tx_root",
			"chain.block_internal.state_root",
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
				ActionCode: "set_header", Label: "设置区块头字段",
				Category: fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "miner", Type: fw.FieldString, Label: "矿工", Required: true, Default: "alice"},
					{Name: "difficulty", Type: fw.FieldNumber, Label: "难度", Required: true, Default: 100, Min: 1, Step: 1},
					{Name: "gas_limit", Type: fw.FieldNumber, Label: "Gas Limit", Required: true, Default: 1000, Min: 100, Step: 100},
				},
			},
			{
				ActionCode: "add_tx", Label: "添加交易",
				Description: "把 tx 加入 body 并应用到 state，更新 receipt + 三 root",
				Category:    fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "alice"},
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "bob"},
					{Name: "value", Type: fw.FieldNumber, Label: "value", Required: true, Default: 100, Min: 0, Step: 1},
					{Name: "gas", Type: fw.FieldNumber, Label: "gas", Required: true, Default: 50, Min: 1, Step: 1},
				},
				WritesOwnedFields: []string{"chain.block_internal.tx_root", "chain.block_internal.state_root"},
				LinkOwnerFields:   []string{"chain.block_internal.tx_root", "chain.block_internal.state_root"},
			},
			{
				ActionCode: "compute_roots", Label: "重算所有 root",
				Category: fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
			},
			{
				ActionCode: "verify_integrity", Label: "验证完整性",
				Description: "重算 root 与存储字段比较",
				Category:    fw.ActionObserve, Trigger: fw.TriggerImmediate,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				LinkOwnerFields: []string{"chain.block_internal.integrity"},
			},
			{
				ActionCode: "tamper_tx", Label: "篡改交易",
				Description: "把指定 tx 的 value 改成新值（不重算 root），演示完整性破坏",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles: []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				Fields: []fw.FieldDef{
					{Name: "tx_id", Type: fw.FieldString, Label: "tx ID", Required: true, Default: "tx0"},
					{Name: "new_value", Type: fw.FieldNumber, Label: "新 value", Required: true, Default: 99999, Min: 0, Step: 1},
				},
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
	st.computeAllRoots()
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "Block 初始化（空 body，5 账户）", true)
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
	case "set_header":
		st.Miner = fw.MapStr(in.Params, "miner", "alice")
		st.Difficulty = fw.MapInt(in.Params, "difficulty", 100)
		st.GasLimit = fw.MapInt(in.Params, "gas_limit", 1000)
		st.computeAllRoots()
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_header", "header 已更新 → 重算 header_hash", false)
		return out, nil

	case "add_tx":
		from := fw.MapStr(in.Params, "from", "alice")
		to := fw.MapStr(in.Params, "to", "bob")
		value := int64(fw.MapInt(in.Params, "value", 100))
		gas := fw.MapInt(in.Params, "gas", 50)
		t := tx{
			ID:   fmt.Sprintf("tx%d", len(st.Txs)),
			From: from, To: to, Value: value, Gas: gas,
			Nonce: len(st.Txs),
		}
		if err := st.addTxAndApply(t); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		r := st.Receipts[len(st.Receipts)-1]
		summary := fmt.Sprintf("%s = %s → %s (value=%d, gas=%d) ", t.ID, from, to, value, gas)
		if r.Success {
			summary += "✓"
		} else {
			summary += "✗ 余额不足"
		}
		out.Render = buildEnvelope(st, "add_tx", summary, false)
		appendAddTxMicroSteps(&out.Render, t.ID, r.Success)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "compute_roots":
		st.computeAllRoots()
		saveState(state, st)
		out.Render = buildEnvelope(st, "compute_roots", "三 root + header_hash 已重算", false)
		appendComputeMicroSteps(&out.Render)
		return out, nil

	case "verify_integrity":
		viol := st.verifyIntegrity()
		summary := "✓ 完整性通过"
		if len(viol) > 0 {
			fields := []string{}
			for _, v := range viol {
				fields = append(fields, v.Field)
			}
			summary = fmt.Sprintf("✗ 完整性破坏：%s", strings.Join(fields, ", "))
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "verify_integrity", summary, false)
		appendVerifyMicroSteps(&out.Render, len(viol) == 0, viol)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "tamper_tx":
		txID := fw.MapStr(in.Params, "tx_id", "tx0")
		newVal := int64(fw.MapInt(in.Params, "new_value", 99999))
		idx := -1
		for i, t := range st.Txs {
			if t.ID == txID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "未找到 tx: " + txID}, nil
		}
		st.Txs[idx].Value = newVal
		st.Tampered = true
		st.TamperedTxID = txID
		saveState(state, st)
		out.Render = buildEnvelope(st, "tamper_tx",
			fmt.Sprintf("⚠ 已篡改 %s.value=%d（root 未重算 → verify 会失败）", txID, newVal), false)
		appendTamperMicroSteps(&out.Render, txID)
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
		st.computeAllRoots()
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

	// 1) header / body 垂直布局
	prims = append(prims, fw.PrimStack("layout", []string{"header-row", "body-row"}, "vertical"))

	// 2) header 节点（横排：header / tx_root / state_root / receipts_root / hash）
	headerIDs := []string{"node-prev", "node-tx-root", "node-state-root", "node-receipts-root", "node-header-hash"}
	prims = append(prims, fw.PrimStack("header-row", headerIDs, "horizontal"))
	prims = append(prims, fw.PrimNode("node-prev", "prev_hash\n"+truncStr(st.PrevHash, 12), "normal", "header-field"))
	prims = append(prims, fw.PrimNode("node-tx-root", "tx_root\n"+truncStr(st.TxRoot, 12), "active", "merkle-root"))
	prims = append(prims, fw.PrimNode("node-state-root", "state_root\n"+truncStr(st.StateRoot, 12), "active", "merkle-root"))
	prims = append(prims, fw.PrimNode("node-receipts-root", "receipts_root\n"+truncStr(st.ReceiptsRoot, 12), "active", "merkle-root"))
	prims = append(prims, fw.PrimNode("node-header-hash", "header_hash\n"+truncStr(st.HeaderHash, 12), "active", "header-hash"))
	for i := 0; i < len(headerIDs)-1; i++ {
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-h-%d", i), headerIDs[i], headerIDs[i+1], "solid", "flow"))
	}

	// 3) body 节点（txs / receipts）
	prims = append(prims, fw.PrimStack("body-row", []string{"tx-list", "receipts-list", "state-list"}, "horizontal"))
	prims = append(prims, fw.PrimNode("tx-list", fmt.Sprintf("Transactions\n%d 笔", len(st.Txs)), "normal", "tx-list"))
	prims = append(prims, fw.PrimNode("receipts-list", fmt.Sprintf("Receipts\n%d 条", len(st.Receipts)), "normal", "receipts-list"))
	prims = append(prims, fw.PrimNode("state-list", fmt.Sprintf("State\n%d 账户", len(st.State)), "normal", "state-list"))

	// 4) tx_list → tx_root, state_list → state_root, receipts_list → receipts_root（边）
	prims = append(prims, fw.PrimEdge("edge-tx", "tx-list", "node-tx-root", "solid", "flow"))
	prims = append(prims, fw.PrimEdge("edge-state", "state-list", "node-state-root", "solid", "flow"))
	prims = append(prims, fw.PrimEdge("edge-rc", "receipts-list", "node-receipts-root", "solid", "flow"))

	// 5) header 摘要
	prims = append(prims, fw.PrimCodeBlock("cb-header",
		fmt.Sprintf("prev_hash      = %s\ntx_root        = %s\nstate_root     = %s\nreceipts_root  = %s\nts             = %d\ndifficulty     = %d\nnonce          = %d\nminer          = %s\ngas_used       = %d\ngas_limit      = %d\nheader_hash    = %s",
			st.PrevHash, st.TxRoot, st.StateRoot, st.ReceiptsRoot,
			st.Timestamp, st.Difficulty, st.Nonce, st.Miner,
			st.GasUsed, st.GasLimit, st.HeaderHash),
		"text", nil, 12))

	// 6) tx 列表
	if len(st.Txs) > 0 {
		txLines := []string{"id    from    to      value   gas   ok"}
		for i, t := range st.Txs {
			ok := "✓"
			if i < len(st.Receipts) && !st.Receipts[i].Success {
				ok = "✗"
			}
			tag := ""
			if st.Tampered && t.ID == st.TamperedTxID {
				tag = " ⚠TAMPERED"
			}
			txLines = append(txLines, fmt.Sprintf("%-5s %-7s %-7s %-7d %-5d %s%s",
				t.ID, t.From, t.To, t.Value, t.Gas, ok, tag))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-txs", strings.Join(txLines, "\n"), "text", nil, 14))
	}

	// 7) state 表
	stKeys := make([]string, 0, len(st.State))
	for k := range st.State {
		stKeys = append(stKeys, k)
	}
	sort.Strings(stKeys)
	stLines := []string{"账户       余额"}
	for _, k := range stKeys {
		stLines = append(stLines, fmt.Sprintf("  %-9s %d", k, st.State[k]))
	}
	prims = append(prims, fw.PrimCodeBlock("cb-state", strings.Join(stLines, "\n"), "text", nil, 10))

	// 8) 完整性验证结果
	viol := st.verifyIntegrity()
	if len(viol) > 0 {
		violLines := []string{"完整性违规："}
		for _, v := range viol {
			violLines = append(violLines, fmt.Sprintf("  %s:\n    expected = %s\n    actual   = %s",
				v.Field, v.Expected, v.Actual))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-violations", strings.Join(violLines, "\n"), "text", nil, 12))
	}

	// 9) 公式
	prims = append(prims, fw.PrimMathFormula("formula-roots",
		`\text{tx\_root} = \mathrm{Merkle}(\text{txs});\ \text{state\_root} = \mathrm{Merkle}(\text{state});\ \text{header\_hash} = H(\text{header})`, false))

	// 10) gas 进度条
	prims = append(prims, fw.PrimProgressBar("gas-progress", float64(st.GasUsed), float64(st.GasLimit),
		fmt.Sprintf("Gas %d/%d", st.GasUsed, st.GasLimit)))

	// 11) 动效
	prims = append(prims, fw.PrimGlow("glow-header", "node-header-hash", "info", 0.8))
	if st.Tampered {
		prims = append(prims, fw.PrimShake("shake-tampered", "cb-violations", 0.4, 700))
	}
	if len(viol) > 0 {
		prims = append(prims, fw.PrimPulse("pulse-violation", "cb-violations", "danger", 1500))
	} else {
		prims = append(prims, fw.PrimPulse("pulse-ok", "node-header-hash", "success", 1800))
	}

	// 12) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-integ", linkGroupBlockchainIntegr, "idle", ""))

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Block 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	viol := st.verifyIntegrity()
	d := map[string]any{
		"prev_hash":     st.PrevHash,
		"tx_root":       st.TxRoot,
		"state_root":    st.StateRoot,
		"receipts_root": st.ReceiptsRoot,
		"header_hash":   st.HeaderHash,
		"miner":         st.Miner,
		"difficulty":    st.Difficulty,
		"gas_used":      st.GasUsed,
		"gas_limit":     st.GasLimit,
		"tx_count":      len(st.Txs),
		"receipt_count": len(st.Receipts),
		"state_keys":    len(st.State),
		"tampered":      st.Tampered,
		"integrity":     len(viol) == 0,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendAddTxMicroSteps(env *fw.RenderEnvelope, txID string, ok bool) {
	tail := "更新 receipt 为成功"
	if !ok {
		tail = "更新 receipt 为失败（余额不足）"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "at-1", Label: "tx 加入 body 列表", DurationMs: 400, HighlightIDs: []string{"cb-txs", "tx-list"}},
		{ID: "at-2", Label: "应用 tx 到 state（balance）", DurationMs: 400, HighlightIDs: []string{"cb-state", "state-list"}, FirePrimitives: []string{"glow-header"}},
		{ID: "at-3", Label: tail, DurationMs: 400, HighlightIDs: []string{"receipts-list"}},
		{ID: "at-4", Label: "重算 tx_root / state_root / receipts_root / header_hash", DurationMs: 500, HighlightIDs: []string{"formula-roots", "cb-header"}, FirePrimitives: []string{"pulse-ok"}, IsLinkTrigger: true},
	}
}

func appendComputeMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "cr-1", Label: "Merkle 化 txs[]", DurationMs: 400, HighlightIDs: []string{"node-tx-root"}},
		{ID: "cr-2", Label: "Merkle 化 state map", DurationMs: 400, HighlightIDs: []string{"node-state-root"}},
		{ID: "cr-3", Label: "Merkle 化 receipts[]", DurationMs: 400, HighlightIDs: []string{"node-receipts-root"}},
		{ID: "cr-4", Label: "header_hash = SHA-256(header)", DurationMs: 500, HighlightIDs: []string{"node-header-hash"}, FirePrimitives: []string{"pulse-ok"}, IsLinkTrigger: true},
	}
}

func appendVerifyMicroSteps(env *fw.RenderEnvelope, ok bool, viol []integrityViolation) {
	tail := "✓ 重算 root 与字段一致"
	if !ok {
		tail = fmt.Sprintf("✗ %d 个字段不一致", len(viol))
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "vf-1", Label: "用当前 body 重算 tx_root / state_root / receipts_root", DurationMs: 500, HighlightIDs: []string{"formula-roots"}},
		{ID: "vf-2", Label: "重算 header_hash", DurationMs: 400, HighlightIDs: []string{"node-header-hash"}},
		{ID: "vf-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-violations", "cb-header"}, FirePrimitives: []string{"pulse-ok"}, IsLinkTrigger: true},
	}
}

func appendTamperMicroSteps(env *fw.RenderEnvelope, txID string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "tm-1", Label: "篡改 " + txID + ".value（不重算 root）", DurationMs: 400, HighlightIDs: []string{"cb-txs"}, FirePrimitives: []string{"shake-tampered"}},
		{ID: "tm-2", Label: "tx_root 字段保持旧值", DurationMs: 400, HighlightIDs: []string{"node-tx-root"}},
		{ID: "tm-3", Label: "verify_integrity 时 → 重算 ≠ 字段 → 失败", DurationMs: 500, HighlightIDs: []string{"cb-violations"}, IsLinkTrigger: true},
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
		ID:             "block-internal-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_block_internal",
		LinkGroup:      linkGroupBlockchainIntegr,
		ChangedFields:  []string{"chain.block_internal.tx_root", "chain.block_internal.state_root"},
		Payload:        map[string]any{"tx_count": len(st.Txs)},
		SourceAnchorID: "block-internal-anchor",
		TargetAnchorID: "integrity-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "chain.block_internal.tx_root", "chain.block_internal.state_root")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	viol := st.verifyIntegrity()
	return map[string]any{
		"chain": map[string]any{
			"block_internal": map[string]any{
				"tx_root":       st.TxRoot,
				"state_root":    st.StateRoot,
				"receipts_root": st.ReceiptsRoot,
				"header_hash":   st.HeaderHash,
				"tx_count":      len(st.Txs),
				"gas_used":      st.GasUsed,
				"gas_limit":     st.GasLimit,
				"integrity":     len(viol) == 0,
				"tampered":      st.Tampered,
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

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
