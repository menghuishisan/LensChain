// 模块：sim-engine/scenarios/internal/economic/governancevoting
// 文件职责：ECO-03 治理投票场景的完整实现。
//
// SSOT 依据：06.md §4.8.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现完整 DAO 治理投票协议（零外部依赖）。
//
//   1. 提案生命周期：PENDING → ACTIVE → SUCCEEDED / DEFEATED → QUEUED → EXECUTED / CANCELED
//      · PENDING       : 提交后等 votingDelay 进入 ACTIVE
//      · ACTIVE        : 接受投票直到 votingPeriod 结束
//      · SUCCEEDED     : 通过 quorum + 多数 yes
//      · DEFEATED      : 未达 quorum 或多数 no
//      · QUEUED        : 进入 timelock 队列，等 timelockDelay 后可执行
//      · EXECUTED      : 实际执行
//      · CANCELED      : 提案者主动撤销（仅 PENDING/ACTIVE）
//
//   2. 三种投票权重模式：
//      · token_weighted  : 1 token = 1 vote
//      · quadratic       : voting_power = sqrt(token_balance)
//      · one_person_one_vote : 每地址 1 票（与 quadratic 类似但不依赖余额）
//
//   3. 委托投票：
//      · delegate(from, to)：把 from 的投票权全部委托给 to
//      · 投票时使用"被委托后总票权"
//
//   4. 治理参数：
//      · proposalThreshold : 创建提案最低 token
//      · quorum            : 达 quorum 比例（教学：4%）
//      · votingDelay       : 1 epoch
//      · votingPeriod      : 5 epoch
//      · timelockDelay     : 2 epoch
//
//   5. 教学指标：
//      · totalProposals / executedCount / defeatedCount / canceledCount
//      · whaleConcentration : 最大持有者票权占比
//      · sybil-attack 演示：1 person → N 个零余额地址投票（quadratic 失效）

package governancevoting

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
)

const (
	sceneCode     = "governance-voting"
	schemaVersion = "v1.0.0"
	algorithmType = "dao-governance"

	weightTokenWeighted = "token_weighted"
	weightQuadratic     = "quadratic"
	weightOPOV          = "one_person_one_vote"

	stateProp_Pending   = "PENDING"
	stateProp_Active    = "ACTIVE"
	stateProp_Succeeded = "SUCCEEDED"
	stateProp_Defeated  = "DEFEATED"
	stateProp_Queued    = "QUEUED"
	stateProp_Executed  = "EXECUTED"
	stateProp_Canceled  = "CANCELED"

	defaultProposalThreshold = 100.0
	defaultQuorumRate        = 0.04
	defaultVotingDelay       = 1
	defaultVotingPeriod      = 5
	defaultTimelockDelay     = 2

	linkGroupPosEcon  = "pos-economy-group"
	linkOwnerSubtree  = "economic.governance"
)

// =====================================================================
// 数据结构
// =====================================================================

type voter struct {
	Addr          string
	TokenBalance  float64
	DelegatedTo   string   // 空 = 自投
	DelegatedFrom []string // 反向引用
}

type voteRecord struct {
	Voter   string
	Support string // yes / no / abstain
	Weight  float64
	Reason  string
}

type proposal struct {
	ID             int
	Proposer       string
	Title          string
	Description    string
	WeightMode     string
	State          string
	StartEpoch     int
	EndEpoch       int
	QueuedEpoch    int
	ExecutionEpoch int
	YesWeight      float64
	NoWeight       float64
	AbstainWeight  float64
	Quorum         float64
	Records        []voteRecord
	CanceledNote   string
}

type govEvent struct {
	Epoch int
	Tick  int
	Kind  string
	Note  string
}

type snapState struct {
	Voters         map[string]*voter
	Proposals      []*proposal
	NextProposalID int
	Epoch          int
	Tick           int
	WeightMode     string

	ProposalThreshold float64
	QuorumRate        float64
	VotingDelay       int
	VotingPeriod      int
	TimelockDelay     int

	ExecutedCount int
	DefeatedCount int
	CanceledCount int
	Events        []govEvent
	LastError     string
}

func defaultSnapState() snapState {
	st := snapState{
		Voters:            map[string]*voter{},
		WeightMode:        weightTokenWeighted,
		ProposalThreshold: defaultProposalThreshold,
		QuorumRate:        defaultQuorumRate,
		VotingDelay:       defaultVotingDelay,
		VotingPeriod:      defaultVotingPeriod,
		TimelockDelay:     defaultTimelockDelay,
	}
	st.Voters["alice"] = &voter{Addr: "alice", TokenBalance: 10000}
	st.Voters["bob"] = &voter{Addr: "bob", TokenBalance: 5000}
	st.Voters["carol"] = &voter{Addr: "carol", TokenBalance: 1000}
	st.Voters["dave"] = &voter{Addr: "dave", TokenBalance: 500}
	st.Voters["whale"] = &voter{Addr: "whale", TokenBalance: 50000}
	return st
}

// votingPower 根据 weightMode 计算单个 voter 的票权（不含委托）。
func (st snapState) votingPower(addr string) float64 {
	v, ok := st.Voters[addr]
	if !ok {
		return 0
	}
	switch st.WeightMode {
	case weightTokenWeighted:
		return v.TokenBalance
	case weightQuadratic:
		return math.Sqrt(v.TokenBalance)
	case weightOPOV:
		if v.TokenBalance > 0 {
			return 1
		}
		return 0
	}
	return 0
}

// effectivePower 计算 addr 的有效票权（自身 + 所有委托给他的）。
func (st snapState) effectivePower(addr string) float64 {
	v, ok := st.Voters[addr]
	if !ok {
		return 0
	}
	if v.DelegatedTo != "" {
		// 已委托给别人 → 自己 0
		return 0
	}
	power := st.votingPower(addr)
	for _, fromAddr := range v.DelegatedFrom {
		power += st.votingPower(fromAddr)
	}
	return power
}

// totalVotingSupply 全网票权（用于 quorum）。
func (st snapState) totalVotingSupply() float64 {
	s := 0.0
	for addr := range st.Voters {
		s += st.votingPower(addr)
	}
	return s
}

// =====================================================================
// 操作
// =====================================================================

// delegate from → to。
func (st *snapState) delegate(from, to string) error {
	f, ok1 := st.Voters[from]
	t, ok2 := st.Voters[to]
	if !ok1 || !ok2 {
		return fmt.Errorf("voter 不存在: %s/%s", from, to)
	}
	if from == to {
		return errors.New("不能委托给自己")
	}
	// 撤销旧委托
	if f.DelegatedTo != "" {
		old := st.Voters[f.DelegatedTo]
		if old != nil {
			old.DelegatedFrom = removeAddr(old.DelegatedFrom, from)
		}
	}
	f.DelegatedTo = to
	t.DelegatedFrom = append(t.DelegatedFrom, from)
	st.recordEvent("delegate", fmt.Sprintf("%s → %s", from, to))
	return nil
}

func removeAddr(arr []string, target string) []string {
	out := []string{}
	for _, a := range arr {
		if a != target {
			out = append(out, a)
		}
	}
	return out
}

// submitProposal 创建提案。
func (st *snapState) submitProposal(proposer, title, desc string) (*proposal, error) {
	v, ok := st.Voters[proposer]
	if !ok {
		return nil, fmt.Errorf("proposer %s 不存在", proposer)
	}
	if v.TokenBalance < st.ProposalThreshold {
		return nil, fmt.Errorf("token %.2f < threshold %.2f", v.TokenBalance, st.ProposalThreshold)
	}
	st.NextProposalID++
	p := &proposal{
		ID:       st.NextProposalID,
		Proposer: proposer, Title: title, Description: desc,
		WeightMode: st.WeightMode,
		State:      stateProp_Pending,
		StartEpoch: st.Epoch + st.VotingDelay,
		EndEpoch:   st.Epoch + st.VotingDelay + st.VotingPeriod,
		Quorum:     st.totalVotingSupply() * st.QuorumRate,
	}
	st.Proposals = append(st.Proposals, p)
	st.recordEvent("submit", fmt.Sprintf("#%d %s by %s", p.ID, title, proposer))
	return p, nil
}

// cast 投票。support ∈ {yes, no, abstain}。
func (st *snapState) cast(propID int, voterAddr, support, reason string) error {
	p := st.findProp(propID)
	if p == nil {
		return fmt.Errorf("提案 #%d 不存在", propID)
	}
	if p.State != stateProp_Active {
		return fmt.Errorf("提案 #%d 当前 %s 不可投票", propID, p.State)
	}
	if support != "yes" && support != "no" && support != "abstain" {
		return errors.New("support 必须是 yes/no/abstain")
	}
	v, ok := st.Voters[voterAddr]
	if !ok {
		return fmt.Errorf("voter %s 不存在", voterAddr)
	}
	if v.DelegatedTo != "" {
		return fmt.Errorf("%s 已委托给 %s，不能直接投票", voterAddr, v.DelegatedTo)
	}
	// 防重复
	for _, r := range p.Records {
		if r.Voter == voterAddr {
			return errors.New("已投票")
		}
	}
	w := st.effectivePower(voterAddr)
	rec := voteRecord{Voter: voterAddr, Support: support, Weight: w, Reason: reason}
	p.Records = append(p.Records, rec)
	switch support {
	case "yes":
		p.YesWeight += w
	case "no":
		p.NoWeight += w
	case "abstain":
		p.AbstainWeight += w
	}
	st.recordEvent("cast",
		fmt.Sprintf("#%d %s votes %s weight=%.4f", propID, voterAddr, support, w))
	return nil
}

// cancel 提案者撤销提案（仅 PENDING/ACTIVE）。
func (st *snapState) cancel(propID int, byAddr string) error {
	p := st.findProp(propID)
	if p == nil {
		return fmt.Errorf("提案 #%d 不存在", propID)
	}
	if p.Proposer != byAddr {
		return fmt.Errorf("仅 proposer 可撤销")
	}
	if p.State != stateProp_Pending && p.State != stateProp_Active {
		return fmt.Errorf("当前 %s 不可 cancel", p.State)
	}
	p.State = stateProp_Canceled
	p.CanceledNote = "by proposer at epoch " + fmt.Sprintf("%d", st.Epoch)
	st.CanceledCount++
	st.recordEvent("cancel", fmt.Sprintf("#%d by %s", propID, byAddr))
	return nil
}

// queueExecute 把 SUCCEEDED 的提案放入 timelock 队列。
func (st *snapState) queueExecute(propID int) error {
	p := st.findProp(propID)
	if p == nil {
		return fmt.Errorf("提案 #%d 不存在", propID)
	}
	if p.State != stateProp_Succeeded {
		return fmt.Errorf("当前 %s 不可 queue", p.State)
	}
	p.State = stateProp_Queued
	p.QueuedEpoch = st.Epoch
	p.ExecutionEpoch = st.Epoch + st.TimelockDelay
	st.recordEvent("queue", fmt.Sprintf("#%d 排入 timelock 至 epoch %d", propID, p.ExecutionEpoch))
	return nil
}

// execute 真正执行（仅 timelock 到期后）。
func (st *snapState) execute(propID int) error {
	p := st.findProp(propID)
	if p == nil {
		return fmt.Errorf("提案 #%d 不存在", propID)
	}
	if p.State != stateProp_Queued {
		return fmt.Errorf("当前 %s 不可 execute", p.State)
	}
	if st.Epoch < p.ExecutionEpoch {
		return fmt.Errorf("timelock 未到（now=%d, exec=%d）", st.Epoch, p.ExecutionEpoch)
	}
	p.State = stateProp_Executed
	st.ExecutedCount++
	st.recordEvent("execute", fmt.Sprintf("#%d EXECUTED at epoch %d", propID, st.Epoch))
	return nil
}

// advanceEpoch 推进 1 epoch；处理状态转移。
func (st *snapState) advanceEpoch() {
	st.Tick++
	st.Epoch++
	for _, p := range st.Proposals {
		switch p.State {
		case stateProp_Pending:
			if st.Epoch >= p.StartEpoch {
				p.State = stateProp_Active
				st.recordEvent("transition", fmt.Sprintf("#%d Pending → Active", p.ID))
			}
		case stateProp_Active:
			if st.Epoch >= p.EndEpoch {
				st.tallyProposal(p)
			}
		}
	}
}

// tallyProposal 截止后统计结果。
func (st *snapState) tallyProposal(p *proposal) {
	totalWeight := p.YesWeight + p.NoWeight + p.AbstainWeight
	if totalWeight < p.Quorum {
		p.State = stateProp_Defeated
		st.DefeatedCount++
		st.recordEvent("transition",
			fmt.Sprintf("#%d Active → Defeated（quorum 不足 %.4f<%.4f）", p.ID, totalWeight, p.Quorum))
		return
	}
	if p.YesWeight > p.NoWeight {
		p.State = stateProp_Succeeded
		st.recordEvent("transition",
			fmt.Sprintf("#%d Active → Succeeded（yes %.4f > no %.4f）", p.ID, p.YesWeight, p.NoWeight))
	} else {
		p.State = stateProp_Defeated
		st.DefeatedCount++
		st.recordEvent("transition",
			fmt.Sprintf("#%d Active → Defeated（yes %.4f ≤ no %.4f）", p.ID, p.YesWeight, p.NoWeight))
	}
}

func (st *snapState) findProp(id int) *proposal {
	for _, p := range st.Proposals {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (st *snapState) recordEvent(kind, note string) {
	st.Events = append(st.Events, govEvent{Epoch: st.Epoch, Tick: st.Tick, Kind: kind, Note: note})
	if len(st.Events) > 64 {
		st.Events = st.Events[len(st.Events)-64:]
	}
}

// distributeTokens 教学：给某地址注入 token（用于 sybil 攻击演示）。
func (st *snapState) distributeTokens(addr string, amount float64) {
	if _, ok := st.Voters[addr]; !ok {
		st.Voters[addr] = &voter{Addr: addr}
	}
	st.Voters[addr].TokenBalance += amount
}

// sybilAttack 演示：1 个 attacker 把自己的 token 平分给 N 个新建地址；
// 在 quadratic 模式下 voting_power 急剧扩大。
func (st *snapState) sybilAttack(from string, n int) error {
	v, ok := st.Voters[from]
	if !ok {
		return fmt.Errorf("%s 不存在", from)
	}
	if v.TokenBalance <= 0 {
		return errors.New("没有 token 可分")
	}
	share := v.TokenBalance / float64(n+1) // attacker 自己保留 1 份
	for i := 1; i <= n; i++ {
		addr := fmt.Sprintf("sybil-%s-%d", from, i)
		st.Voters[addr] = &voter{Addr: addr, TokenBalance: share}
	}
	v.TokenBalance = share
	st.recordEvent("sybil", fmt.Sprintf("%s 把 token 平分给 %d 个 sybil（教学）", from, n))
	return nil
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
		Voters:            map[string]*voter{},
		WeightMode:        fw.MapStr(d, "weight_mode", weightTokenWeighted),
		ProposalThreshold: floatOr(d, "threshold", defaultProposalThreshold),
		QuorumRate:        floatOr(d, "quorum_rate", defaultQuorumRate),
		VotingDelay:       fw.MapInt(d, "voting_delay", defaultVotingDelay),
		VotingPeriod:      fw.MapInt(d, "voting_period", defaultVotingPeriod),
		TimelockDelay:     fw.MapInt(d, "timelock_delay", defaultTimelockDelay),
		Epoch:             fw.MapInt(d, "epoch", 0),
		Tick:              fw.MapInt(d, "tick", 0),
		NextProposalID:    fw.MapInt(d, "next_id", 0),
		ExecutedCount:     fw.MapInt(d, "exec_cnt", 0),
		DefeatedCount:     fw.MapInt(d, "def_cnt", 0),
		CanceledCount:     fw.MapInt(d, "can_cnt", 0),
		LastError:         fw.MapStr(d, "last_error", ""),
	}
	if vAny, ok := d["voters"].(map[string]any); ok {
		for addr, x := range vAny {
			if m, ok := x.(map[string]any); ok {
				v := &voter{Addr: addr,
					TokenBalance: floatOr(m, "balance", 0),
					DelegatedTo:  fw.MapStr(m, "deleg_to", ""),
				}
				if dfAny, ok := m["deleg_from"].([]any); ok {
					for _, y := range dfAny {
						if s, ok := y.(string); ok {
							v.DelegatedFrom = append(v.DelegatedFrom, s)
						}
					}
				}
				st.Voters[addr] = v
			}
		}
	}
	if len(st.Voters) == 0 {
		return defaultSnapState()
	}
	if pAny, ok := d["proposals"].([]any); ok {
		for _, x := range pAny {
			if m, ok := x.(map[string]any); ok {
				p := &proposal{
					ID:       fw.MapInt(m, "id", 0),
					Proposer: fw.MapStr(m, "proposer", ""),
					Title:    fw.MapStr(m, "title", ""), Description: fw.MapStr(m, "desc", ""),
					WeightMode: fw.MapStr(m, "wm", ""),
					State:      fw.MapStr(m, "state", stateProp_Pending),
					StartEpoch: fw.MapInt(m, "start", 0), EndEpoch: fw.MapInt(m, "end", 0),
					QueuedEpoch: fw.MapInt(m, "queued", 0), ExecutionEpoch: fw.MapInt(m, "exec", 0),
					YesWeight: floatOr(m, "yes", 0), NoWeight: floatOr(m, "no", 0),
					AbstainWeight: floatOr(m, "abs", 0),
					Quorum:        floatOr(m, "quorum", 0),
					CanceledNote:  fw.MapStr(m, "cancel_note", ""),
				}
				if rAny, ok := m["records"].([]any); ok {
					for _, y := range rAny {
						if rm, ok := y.(map[string]any); ok {
							p.Records = append(p.Records, voteRecord{
								Voter:   fw.MapStr(rm, "v", ""),
								Support: fw.MapStr(rm, "s", ""),
								Weight:  floatOr(rm, "w", 0),
								Reason:  fw.MapStr(rm, "r", ""),
							})
						}
					}
				}
				st.Proposals = append(st.Proposals, p)
			}
		}
	}
	if eAny, ok := d["events"].([]any); ok {
		for _, x := range eAny {
			if m, ok := x.(map[string]any); ok {
				st.Events = append(st.Events, govEvent{
					Epoch: fw.MapInt(m, "epoch", 0), Tick: fw.MapInt(m, "tick", 0),
					Kind: fw.MapStr(m, "kind", ""), Note: fw.MapStr(m, "note", ""),
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
	s.Data["weight_mode"] = st.WeightMode
	s.Data["threshold"] = st.ProposalThreshold
	s.Data["quorum_rate"] = st.QuorumRate
	s.Data["voting_delay"] = st.VotingDelay
	s.Data["voting_period"] = st.VotingPeriod
	s.Data["timelock_delay"] = st.TimelockDelay
	s.Data["epoch"] = st.Epoch
	s.Data["tick"] = st.Tick
	s.Data["next_id"] = st.NextProposalID
	s.Data["exec_cnt"] = st.ExecutedCount
	s.Data["def_cnt"] = st.DefeatedCount
	s.Data["can_cnt"] = st.CanceledCount
	s.Data["last_error"] = st.LastError
	vAny := map[string]any{}
	for addr, v := range st.Voters {
		df := make([]any, len(v.DelegatedFrom))
		for i, a := range v.DelegatedFrom {
			df[i] = a
		}
		vAny[addr] = map[string]any{
			"balance": v.TokenBalance, "deleg_to": v.DelegatedTo, "deleg_from": df,
		}
	}
	s.Data["voters"] = vAny
	pAny := make([]any, len(st.Proposals))
	for i, p := range st.Proposals {
		recs := make([]any, len(p.Records))
		for j, r := range p.Records {
			recs[j] = map[string]any{"v": r.Voter, "s": r.Support, "w": r.Weight, "r": r.Reason}
		}
		pAny[i] = map[string]any{
			"id": p.ID, "proposer": p.Proposer,
			"title": p.Title, "desc": p.Description,
			"wm": p.WeightMode, "state": p.State,
			"start": p.StartEpoch, "end": p.EndEpoch,
			"queued": p.QueuedEpoch, "exec": p.ExecutionEpoch,
			"yes": p.YesWeight, "no": p.NoWeight, "abs": p.AbstainWeight,
			"quorum": p.Quorum, "records": recs, "cancel_note": p.CanceledNote,
		}
	}
	s.Data["proposals"] = pAny
	eAny := make([]any, len(st.Events))
	for i, ev := range st.Events {
		eAny[i] = map[string]any{
			"epoch": ev.Epoch, "tick": ev.Tick,
			"kind": ev.Kind, "note": ev.Note,
		}
	}
	s.Data["events"] = eAny
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code: sceneCode, Name: "DAO 治理投票",
		Description:         "演示提案 lifecycle + 三种权重模式（token / quadratic / OPOV）+ 委托投票 + timelock + sybil 攻击",
		Category:            fw.CategoryEconomic,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlReactive,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{},

		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths:    []string{},

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
		SceneCode: sceneCode, SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_params", Label: "治理参数",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "weight_mode", Type: fw.FieldEnum, Label: "权重模式", Required: true, Default: weightTokenWeighted,
						Options: []any{weightTokenWeighted, weightQuadratic, weightOPOV}},
					{Name: "threshold", Type: fw.FieldNumber, Label: "提案门槛 token", Required: true, Default: defaultProposalThreshold, Min: 0, Step: 50},
					{Name: "quorum_rate", Type: fw.FieldNumber, Label: "Quorum rate", Required: true, Default: defaultQuorumRate, Min: 0, Max: 1, Step: 0.01},
					{Name: "voting_delay", Type: fw.FieldNumber, Label: "VotingDelay", Required: true, Default: defaultVotingDelay, Min: 0, Step: 1},
					{Name: "voting_period", Type: fw.FieldNumber, Label: "VotingPeriod", Required: true, Default: defaultVotingPeriod, Min: 1, Step: 1},
					{Name: "timelock_delay", Type: fw.FieldNumber, Label: "TimelockDelay", Required: true, Default: defaultTimelockDelay, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "submit", Label: "创建提案",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "proposer", Type: fw.FieldString, Label: "proposer", Required: true, Default: "alice"},
					{Name: "title", Type: fw.FieldString, Label: "title", Required: true, Default: "Increase block reward"},
					{Name: "description", Type: fw.FieldString, Label: "description", Required: false, Default: "Bump from 50 to 75"},
				},
			},
			{
				ActionCode: "cast", Label: "投票",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "prop_id", Type: fw.FieldNumber, Label: "提案 ID", Required: true, Default: 1, Min: 1, Step: 1},
					{Name: "voter", Type: fw.FieldString, Label: "voter", Required: true, Default: "bob"},
					{Name: "support", Type: fw.FieldEnum, Label: "support", Required: true, Default: "yes",
						Options: []any{"yes", "no", "abstain"}},
					{Name: "reason", Type: fw.FieldString, Label: "reason", Required: false, Default: ""},
				},
			},
			{
				ActionCode: "delegate", Label: "委托投票权",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "carol"},
					{Name: "to", Type: fw.FieldString, Label: "to", Required: true, Default: "bob"},
				},
			},
			{
				ActionCode: "advance_epoch", Label: "推进 epoch",
				Description:   "处理 PENDING→ACTIVE / ACTIVE→SUCCEEDED·DEFEATED",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneEpoch,
			},
			{
				ActionCode: "advance_n", Label: "推进 N epoch",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneEpoch,
				Fields: []fw.FieldDef{
					{Name: "n", Type: fw.FieldNumber, Label: "n", Required: true, Default: 5, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "queue", Label: "排入 timelock",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "prop_id", Type: fw.FieldNumber, Label: "提案 ID", Required: true, Default: 1, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "execute", Label: "执行",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.IntervenePhase,
				Fields: []fw.FieldDef{
					{Name: "prop_id", Type: fw.FieldNumber, Label: "提案 ID", Required: true, Default: 1, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "cancel", Label: "撤销提案",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "prop_id", Type: fw.FieldNumber, Label: "提案 ID", Required: true, Default: 1, Min: 1, Step: 1},
					{Name: "by", Type: fw.FieldString, Label: "proposer", Required: true, Default: "alice"},
				},
			},
			{
				ActionCode: "distribute_tokens", Label: "调整 token 余额",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "addr", Type: fw.FieldString, Label: "地址", Required: true, Default: "newbie"},
					{Name: "amount", Type: fw.FieldNumber, Label: "amount", Required: true, Default: 100, Step: 50},
				},
			},
			{
				ActionCode: "sybil_attack", Label: "Sybil 攻击演示",
				Description:   "把 from 的 token 平分给 N 个 sybil 地址",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "from", Type: fw.FieldString, Label: "from", Required: true, Default: "whale"},
					{Name: "n", Type: fw.FieldNumber, Label: "sybil 数量", Required: true, Default: 50, Min: 1, Step: 1},
				},
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_force_epoch",
				Label:         "教师强制纪元推进",
				Description:   "仅教师可用，强制纪元推进用于教学演示",
				Category:      fw.ActionAttackInject,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师强制纪元推进"},
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
	env := buildEnvelope(st, "init", "DAO 5 voters 初始化（token-weighted）", true)
	publishOwnerSubtree(&env, st)
	return env, nil
}

func stepScene(state *fw.SceneState, in fw.StepInput) (fw.StepOutput, error) {
	st := loadState(state)
	state.Tick = in.Tick
	return fw.StepOutput{Render: buildEnvelope(st, "tick", "", false)}, nil
}

func handleAction(state *fw.SceneState, in fw.ActionInput) (fw.ActionOutput, error) {
	_ = fw.EnsureActorBucket(state, in.ActorID)
	if out, ok := fw.HandleBroadcastHint(state, in); ok {
		return out, nil
	}

	st := loadState(state)
	out := fw.ActionOutput{Success: true}
	st.Tick++

	switch in.ActionCode {
	case "set_params":
		st.WeightMode = fw.MapStr(in.Params, "weight_mode", weightTokenWeighted)
		st.ProposalThreshold = floatOr(in.Params, "threshold", defaultProposalThreshold)
		st.QuorumRate = floatOr(in.Params, "quorum_rate", defaultQuorumRate)
		st.VotingDelay = fw.MapInt(in.Params, "voting_delay", defaultVotingDelay)
		st.VotingPeriod = fw.MapInt(in.Params, "voting_period", defaultVotingPeriod)
		st.TimelockDelay = fw.MapInt(in.Params, "timelock_delay", defaultTimelockDelay)
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_params", "参数已更新", false)
		return out, nil

	case "submit":
		proposer := fw.MapStr(in.Params, "proposer", "alice")
		title := fw.MapStr(in.Params, "title", "Increase block reward")
		desc := fw.MapStr(in.Params, "description", "")
		p, err := st.submitProposal(proposer, title, desc)
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "submit",
			fmt.Sprintf("✓ 创建 #%d %s", p.ID, p.Title), false)
		appendSubmitMicroSteps(&out.Render, p.ID)
		return out, nil

	case "cast":
		id := fw.MapInt(in.Params, "prop_id", 1)
		v := fw.MapStr(in.Params, "voter", "bob")
		s := fw.MapStr(in.Params, "support", "yes")
		r := fw.MapStr(in.Params, "reason", "")
		if err := st.cast(id, v, s, r); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "cast",
			fmt.Sprintf("#%d %s votes %s", id, v, s), false)
		appendCastMicroSteps(&out.Render, id, v, s)
		return out, nil

	case "delegate":
		f := fw.MapStr(in.Params, "from", "carol")
		t := fw.MapStr(in.Params, "to", "bob")
		if err := st.delegate(f, t); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "delegate",
			fmt.Sprintf("%s 委托给 %s", f, t), false)
		appendDelegateMicroSteps(&out.Render, f, t)
		return out, nil

	case "advance_epoch":
		st.advanceEpoch()
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_epoch",
			fmt.Sprintf("epoch=%d", st.Epoch), false)
		appendAdvanceMicroSteps(&out.Render, st.Epoch)
		return out, nil

	case "advance_n":
		n := fw.MapInt(in.Params, "n", 5)
		for i := 0; i < n; i++ {
			st.advanceEpoch()
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "advance_n",
			fmt.Sprintf("推进 %d epoch → epoch=%d", n, st.Epoch), false)
		appendAdvanceMicroSteps(&out.Render, st.Epoch)
		return out, nil

	case "queue":
		id := fw.MapInt(in.Params, "prop_id", 1)
		if err := st.queueExecute(id); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "queue",
			fmt.Sprintf("#%d → QUEUED", id), false)
		appendQueueMicroSteps(&out.Render)
		return out, nil

	case "execute":
		id := fw.MapInt(in.Params, "prop_id", 1)
		if err := st.execute(id); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "execute",
			fmt.Sprintf("#%d EXECUTED", id), false)
		appendExecuteMicroSteps(&out.Render)
		return out, nil

	case "cancel":
		id := fw.MapInt(in.Params, "prop_id", 1)
		by := fw.MapStr(in.Params, "by", "alice")
		if err := st.cancel(id, by); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "cancel",
			fmt.Sprintf("#%d CANCELED", id), false)
		return out, nil

	case "distribute_tokens":
		addr := fw.MapStr(in.Params, "addr", "newbie")
		amt := floatOr(in.Params, "amount", 100)
		st.distributeTokens(addr, amt)
		saveState(state, st)
		out.Render = buildEnvelope(st, "distribute_tokens",
			fmt.Sprintf("%s += %.2f", addr, amt), false)
		return out, nil

	case "sybil_attack":
		from := fw.MapStr(in.Params, "from", "whale")
		n := fw.MapInt(in.Params, "n", 50)
		if err := st.sybilAttack(from, n); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "sybil_attack",
			fmt.Sprintf("⚠ %s 创建 %d 个 sybil（quadratic 模式下投票力被放大）", from, n), false)
		appendSybilMicroSteps(&out.Render, n)
		return out, nil

	case "teacher_force_epoch":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师强制纪元推进"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-epoch-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
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
	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode"}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	prims := make([]fw.Primitive, 0, 60)

	// 1) 提案状态机流水线
	phases := []string{stateProp_Pending, stateProp_Active, stateProp_Succeeded, stateProp_Queued, stateProp_Executed}
	pIDs := []string{"ph-pending", "ph-active", "ph-succeeded", "ph-queued", "ph-executed"}
	prims = append(prims, fw.PrimStack("phase-stack", pIDs, "horizontal"))
	curPhase := -1
	if len(st.Proposals) > 0 {
		last := st.Proposals[len(st.Proposals)-1]
		for i, p := range phases {
			if p == last.State {
				curPhase = i
			}
		}
	}
	for i, p := range phases {
		role := strings.ToLower(p)
		status := "normal"
		if i == curPhase {
			status = "active"
		}
		prims = append(prims, fw.PrimNode(pIDs[i], p, status, role))
	}
	for i := 0; i < 4; i++ {
		anim := ""
		if i < curPhase {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("ph-edge-%d", i), pIDs[i], pIDs[i+1], "solid", anim))
	}

	// 2) Voter 节点（环形）
	vAddrs := []string{}
	for k := range st.Voters {
		vAddrs = append(vAddrs, k)
	}
	sort.Strings(vAddrs)
	voterRingIDs := make([]string, len(vAddrs))
	for i, a := range vAddrs {
		voterRingIDs[i] = "v-" + a
	}
	prims = append(prims, fw.PrimRingLayout("voter-ring", voterRingIDs))
	for _, a := range vAddrs {
		v := st.Voters[a]
		role := "voter"
		if strings.HasPrefix(a, "sybil-") {
			role = "voter-sybil"
		}
		if v.DelegatedTo != "" {
			role = "voter-delegated"
		}
		label := fmt.Sprintf("%s\nbal=%.0f\npower=%.2f", a, v.TokenBalance, st.effectivePower(a))
		if v.DelegatedTo != "" {
			label += "\n→ " + v.DelegatedTo
		}
		prims = append(prims, fw.PrimNode("v-"+a, label, "active", role))
	}

	// 3) 委托关系边
	for _, a := range vAddrs {
		v := st.Voters[a]
		if v.DelegatedTo != "" {
			prims = append(prims, fw.PrimEdge("de-"+a, "v-"+a, "v-"+v.DelegatedTo, "dashed", "flow"))
		}
	}

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-power",
		`\text{token: } P=B;\quad \text{quadratic: } P=\sqrt{B};\quad \text{OPOV: } P=\mathbb{1}_{B>0}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-quorum",
		`\text{Quorum} = \text{TotalSupply}_P \times r_q;\quad \text{Pass iff totalVotes} \ge Q \land \text{yes} > \text{no}`, false))

	// 5) 状态参数
	prims = append(prims, fw.PrimCodeBlock("cb-status",
		fmt.Sprintf("epoch = %d\nweightMode = %s  threshold=%.0f  quorumRate=%.4f\nvotingDelay = %d  votingPeriod = %d  timelockDelay = %d\ntotalVotingSupply = %.4f\nproposals = %d  executed = %d  defeated = %d  canceled = %d",
			st.Epoch,
			st.WeightMode, st.ProposalThreshold, st.QuorumRate,
			st.VotingDelay, st.VotingPeriod, st.TimelockDelay,
			st.totalVotingSupply(),
			len(st.Proposals), st.ExecutedCount, st.DefeatedCount, st.CanceledCount),
		"text", nil, 10))

	// 6) Voter 表
	if len(vAddrs) > 0 {
		vLines := []string{"address           balance    voting_power(eff)  delegated_to"}
		for _, a := range vAddrs {
			v := st.Voters[a]
			vLines = append(vLines, fmt.Sprintf("  %-16s  %-9.2f  %-17.4f  %s",
				a, v.TokenBalance, st.effectivePower(a), v.DelegatedTo))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-voters", strings.Join(vLines, "\n"), "text", nil, 16))
	}

	// 7) 提案表
	if len(st.Proposals) > 0 {
		pLines := []string{"id  state       proposer  title                              yes        no         abst       quorum     start  end  exec"}
		for _, p := range st.Proposals {
			pLines = append(pLines, fmt.Sprintf("  %-3d %-10s  %-9s  %-32s  %-9.4f  %-9.4f  %-9.4f  %-9.4f  %-5d  %-3d  %d",
				p.ID, p.State, p.Proposer, truncate(p.Title, 32),
				p.YesWeight, p.NoWeight, p.AbstainWeight, p.Quorum,
				p.StartEpoch, p.EndEpoch, p.ExecutionEpoch))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-proposals", strings.Join(pLines, "\n"), "text", nil, 14))
	}

	// 8) 当前提案的投票饼图（最新提案）
	if len(st.Proposals) > 0 {
		p := st.Proposals[len(st.Proposals)-1]
		prims = append(prims, fw.PrimPieChart("vote-pie", []map[string]any{
			{"label": "Yes", "value": p.YesWeight, "color_role": "success"},
			{"label": "No", "value": p.NoWeight, "color_role": "danger"},
			{"label": "Abstain", "value": p.AbstainWeight, "color_role": "info"},
		}))
	}

	// 9) 进度条：proposal quorum
	if len(st.Proposals) > 0 {
		p := st.Proposals[len(st.Proposals)-1]
		total := p.YesWeight + p.NoWeight + p.AbstainWeight
		prims = append(prims, fw.PrimProgressBar("bar-quorum",
			total, p.Quorum,
			fmt.Sprintf("Proposal #%d total %.4f / quorum %.4f", p.ID, total, p.Quorum)))
	}

	// 10) 事件日志
	if len(st.Events) > 0 {
		eLines := []string{"epoch tick   kind          note"}
		startIdx := 0
		if len(st.Events) > 16 {
			startIdx = len(st.Events) - 16
		}
		for _, ev := range st.Events[startIdx:] {
			eLines = append(eLines, fmt.Sprintf("  %-5d %-5d  %-12s  %s",
				ev.Epoch, ev.Tick, ev.Kind, ev.Note))
		}
		prims = append(prims, fw.PrimCodeBlock("cb-events", strings.Join(eLines, "\n"), "text", nil, 18))
	}

	// 11) 动效
	if len(st.Proposals) > 0 {
		last := st.Proposals[len(st.Proposals)-1]
		switch last.State {
		case stateProp_Executed:
			prims = append(prims, fw.PrimBurst("burst-exec", "ph-executed", "success", int64(last.ID), 700))
		case stateProp_Defeated:
			prims = append(prims, fw.PrimShake("shake-def", "vote-pie", 0.5, 800))
		case stateProp_Active:
			prims = append(prims, fw.PrimGlow("glow-active", "ph-active", "info", 0.7))
		case stateProp_Queued:
			prims = append(prims, fw.PrimPulse("pulse-queue", "ph-queued", "warning", 1500))
		}
	}

	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "Governance 错误", st.LastError, "scene", "请检查参数", true))
	}

	return fw.RenderEnvelope{
		Primitives:     prims,
		IsFullSnapshot: fullSnapshot,
		ChangedKeys:    []string{reason},
		Data:           buildSidePanelData(st, summary),
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func buildSidePanelData(st snapState, summary string) map[string]any {
	d := map[string]any{
		"epoch":               st.Epoch,
		"weight_mode":         st.WeightMode,
		"proposals":           len(st.Proposals),
		"executed":            st.ExecutedCount,
		"defeated":            st.DefeatedCount,
		"canceled":            st.CanceledCount,
		"voters":              len(st.Voters),
		"total_voting_supply": st.totalVotingSupply(),
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep
// =====================================================================

func appendSubmitMicroSteps(env *fw.RenderEnvelope, id int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "检查 proposalThreshold", DurationMs: 400, HighlightIDs: []string{"cb-status"}},
		{ID: "s-2", Label: fmt.Sprintf("创建 #%d → PENDING", id), DurationMs: 500, HighlightIDs: []string{"ph-pending", "cb-proposals"}, IsLinkTrigger: true},
	}
}

func appendCastMicroSteps(env *fw.RenderEnvelope, id int, voter, support string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "c-1", Label: "计算 voter 有效票权（含 delegation）", DurationMs: 400, HighlightIDs: []string{"formula-power"}},
		{ID: "c-2", Label: fmt.Sprintf("#%d %s 投 %s", id, voter, support), DurationMs: 500, HighlightIDs: []string{"vote-pie", "cb-proposals", "bar-quorum"}, IsLinkTrigger: true},
	}
}

func appendDelegateMicroSteps(env *fw.RenderEnvelope, from, to string) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "d-1", Label: fmt.Sprintf("%s.delegatedTo = %s", from, to), DurationMs: 400, HighlightIDs: []string{"v-" + from, "v-" + to}},
		{ID: "d-2", Label: "更新有效票权", DurationMs: 400, HighlightIDs: []string{"cb-voters"}, IsLinkTrigger: true},
	}
}

func appendAdvanceMicroSteps(env *fw.RenderEnvelope, epoch int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "a-1", Label: fmt.Sprintf("epoch → %d", epoch), DurationMs: 400, HighlightIDs: []string{"cb-status"}},
		{ID: "a-2", Label: "处理 PENDING→ACTIVE / ACTIVE→tally", DurationMs: 500, HighlightIDs: []string{"phase-stack", "formula-quorum"}, IsLinkTrigger: true},
	}
}

func appendQueueMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "q-1", Label: "进入 timelock 队列", DurationMs: 400, HighlightIDs: []string{"ph-queued"}, FirePrimitives: []string{"pulse-queue"}, IsLinkTrigger: true},
	}
}

func appendExecuteMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "e-1", Label: "等待 TimelockDelay 到期", DurationMs: 400, HighlightIDs: []string{"ph-queued"}},
		{ID: "e-2", Label: "EXECUTED", DurationMs: 500, HighlightIDs: []string{"ph-executed"}, FirePrimitives: []string{"burst-exec"}, IsLinkTrigger: true},
	}
}

func appendSybilMicroSteps(env *fw.RenderEnvelope, n int) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sy-1", Label: fmt.Sprintf("创建 %d 个 sybil 地址", n), DurationMs: 400, HighlightIDs: []string{"voter-ring"}},
		{ID: "sy-2", Label: "在 quadratic 模式下：每 sybil sqrt(B/n) → ∑sqrt(B/n) ≫ sqrt(B)", DurationMs: 600, HighlightIDs: []string{"formula-power", "cb-voters"}, IsLinkTrigger: true},
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
		ID:             "governance-update",
		SourceScene:    sceneCode,
		SourceAction:   "cast_vote",
		LinkGroup:      linkGroupPosEcon,
		ChangedFields:  []string{"economic.governance.active_proposals"},
		Payload:        map[string]any{"active_proposals": len(st.Proposals)},
		SourceAnchorID: "governance-anchor",
		TargetAnchorID: "econ-group-anchor",
	})
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

// =====================================================================
// 工具
// =====================================================================

func floatOr(m map[string]any, k string, def float64) float64 {
	if m == nil {
		return def
	}
	if v, ok := m[k]; ok {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		case int64:
			return float64(t)
		}
	}
	return def
}
