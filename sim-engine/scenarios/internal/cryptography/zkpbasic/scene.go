// 模块：sim-engine/scenarios/internal/cryptography/zkpbasic
// 文件职责：CRY-06 零知识证明基础（Schnorr 协议）场景的完整实现。
//
// SSOT 依据：06.md §4.3.6 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现 Schnorr 离散对数零知识证明（DLOG ZKP），含 Fiat-Shamir 非
// 交互变体。零依赖加密第三方库；仅使用 math/big、rand.New(rand.NewSource) 确定性 RNG、复用 sha256hash.Sum256：
//
//   公开参数（教学版安全素数群）：
//     · p：安全素数（p = 2q + 1，q 也是素数）；
//     · q：p 的奇素因子（生成元 g 的阶）；
//     · g：阶为 q 的生成元（验证 g^q ≡ 1 mod p）。
//
//   秘密 / 公开：
//     · 秘密 x ∈ [1, q-1]；公开 y = g^x mod p（Prover 给 Verifier）。
//
//   协议（3 步 Sigma 协议）：
//     1. Commit：Prover 选随机 r ∈ [1, q-1]，发送 t = g^r mod p；
//     2. Challenge：Verifier 发送随机 c ∈ [0, q-1]；
//     3. Response：Prover 发送 s = (r + c·x) mod q；
//     4. Verify：g^s mod p ?= (t · y^c) mod p。
//
//   Fiat-Shamir 非交互化（教学）：
//     · c = SHA-256(g || y || t) mod q（不再需要交互式 Challenge）。
//
//   零知识属性演示：
//     · forge_attempt：不知 x 时随机选 s' / c' 伪造 → 验证必失败；
//     · 同样 (g, y) 上 Prover 可对不同消息生成无穷多组合法证明，但都不泄漏 x。

package zkpbasic

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand" // used via rand.New(rand.NewSource) only — deterministic
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "zkp-basic"
	schemaVersion = "v1.0.0"
	algorithmType = "schnorr-zkp"

	defaultBits    = 32
	maxBits        = 64
	minBits        = 16
	defaultSecretX = "7"
	defaultR       = "5"

	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "proofs.zkp"

	millerRabinTries = 12
)

var pipelineNodeIDs = []string{
	"phase-params", "phase-secret", "phase-commit", "phase-challenge", "phase-response", "phase-verify",
}
var phaseLabels = []string{"参数 p,q,g", "y = gˣ", "Commit t = gʳ", "Challenge c", "Response s"}

// =====================================================================
// Miller-Rabin 素性测试（与 rsaencrypt 同算法，独立实现避免跨包暴露内部 API）
// =====================================================================

func millerRabin(n *big.Int, rounds int, rng *rand.Rand) bool {
	if n.Cmp(big.NewInt(2)) < 0 {
		return false
	}
	if n.Cmp(big.NewInt(2)) == 0 || n.Cmp(big.NewInt(3)) == 0 {
		return true
	}
	if n.Bit(0) == 0 {
		return false
	}
	smallPrimes := []int64{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
	for _, sp := range smallPrimes {
		spB := big.NewInt(sp)
		if n.Cmp(spB) == 0 {
			return true
		}
		if new(big.Int).Mod(n, spB).Sign() == 0 {
			return false
		}
	}
	d := new(big.Int).Sub(n, big.NewInt(1))
	r := 0
	for d.Bit(0) == 0 {
		d.Rsh(d, 1)
		r++
	}
	nMinus1 := new(big.Int).Sub(n, big.NewInt(1))
	nMinus3 := new(big.Int).Sub(n, big.NewInt(3))
	for i := 0; i < rounds; i++ {
		if nMinus3.Sign() <= 0 {
			return false
		}
		a := new(big.Int).Rand(rng, nMinus3)
		a.Add(a, big.NewInt(2))
		x := new(big.Int).Exp(a, d, n)
		if x.Cmp(big.NewInt(1)) == 0 || x.Cmp(nMinus1) == 0 {
			continue
		}
		composite := true
		for j := 0; j < r-1; j++ {
			x = new(big.Int).Exp(x, big.NewInt(2), n)
			if x.Cmp(nMinus1) == 0 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

// =====================================================================
// 安全素数群参数生成（自实现）
// =====================================================================

// generateSafePrime 生成安全素数 p（p = 2q+1，q 也是素数）。
// 教学位长（≤ 64）下几毫秒可完成；不适合生产。
func generateSafePrime(bits int, rng *rand.Rand) (p, q *big.Int) {
	for {
		// 先生成 (bits-1) 位的奇素数 q
		buf := make([]byte, (bits-1+7)/8)
		for i := range buf {
			buf[i] = byte(rng.Intn(256))
		}
		// 强制最高位为 1（保证位长）+ 最低位为 1（奇）
		extra := uint(len(buf)*8 - (bits - 1))
		buf[0] &= byte(0xFF >> extra)
		buf[0] |= byte(0x80 >> extra)
		buf[len(buf)-1] |= 0x01
		qCand := new(big.Int).SetBytes(buf)
		if !millerRabin(qCand, millerRabinTries, rng) {
			continue
		}
		// p = 2q + 1
		pCand := new(big.Int).Lsh(qCand, 1)
		pCand.Add(pCand, big.NewInt(1))
		if pCand.BitLen() != bits {
			continue
		}
		if millerRabin(pCand, millerRabinTries, rng) {
			return pCand, qCand
		}
	}
}

// findGenerator 在安全素数群 (p, q) 中找到阶为 q 的生成元 g。
// 由于 p = 2q+1 → 群 Z_p* 阶为 2q；阶 q 子群的元素恰为二次剩余。
// g = h² mod p（h 任取 1~p-1 且 h ≠ 1 / p-1），则 g 是阶为 q 的生成元（除非 h ∈ {1, p-1}）。
func findGenerator(p, q *big.Int, rng *rand.Rand) *big.Int {
	pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
	for {
		h := new(big.Int).Rand(rng, pMinus1) // ∈ [0, p-2]
		h.Add(h, big.NewInt(1))              // ∈ [1, p-1]
		if h.Cmp(big.NewInt(1)) == 0 || h.Cmp(pMinus1) == 0 {
			continue
		}
		g := new(big.Int).Exp(h, big.NewInt(2), p)
		if g.Cmp(big.NewInt(1)) == 0 {
			continue
		}
		// 验证 g 阶为 q：g^q ≡ 1 mod p
		if new(big.Int).Exp(g, q, p).Cmp(big.NewInt(1)) == 0 {
			return g
		}
	}
}

// =====================================================================
// Schnorr 协议
// =====================================================================

type schnorrParams struct {
	P, Q, G *big.Int
}

// commit Prover：选随机 r → t = g^r mod p。
func (s schnorrParams) commit(r *big.Int) *big.Int {
	return new(big.Int).Exp(s.G, r, s.P)
}

// fiatShamirChallenge 非交互式挑战：c = SHA-256(g || y || t) mod q。
func (s schnorrParams) fiatShamirChallenge(y, t *big.Int) *big.Int {
	buf := []byte{}
	buf = append(buf, s.G.Bytes()...)
	buf = append(buf, y.Bytes()...)
	buf = append(buf, t.Bytes()...)
	h := sha256hash.Sum256(buf)
	c := new(big.Int).SetBytes(h[:])
	return c.Mod(c, s.Q)
}

// response Prover：s = (r + c·x) mod q。
func (s schnorrParams) response(r, c, x *big.Int) *big.Int {
	prod := new(big.Int).Mul(c, x)
	sum := new(big.Int).Add(r, prod)
	return sum.Mod(sum, s.Q)
}

// verify Verifier：g^s ?= (t · y^c) mod p。
func (s schnorrParams) verify(y, t, c, sResp *big.Int) bool {
	lhs := new(big.Int).Exp(s.G, sResp, s.P)
	yc := new(big.Int).Exp(y, c, s.P)
	rhs := new(big.Int).Mul(t, yc)
	rhs.Mod(rhs, s.P)
	return lhs.Cmp(rhs) == 0
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Bits      int
	Seed      int64
	P         string
	Q         string
	G         string
	X         string // 秘密
	Y         string // 公开 y = g^x
	R         string // commit 随机数
	T         string // commitment
	C         string // challenge
	S         string // response
	Verified  bool
	ForgeS    string
	ForgeC    string
	ForgeOK   bool
	LastError string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{Bits: defaultBits, Seed: 1, X: defaultSecretX, R: defaultR}
	}
	d := s.Data
	return snapState{
		Bits:      fw.MapInt(d, "bits", defaultBits),
		Seed:      int64(fw.MapInt(d, "seed", 1)),
		P:         fw.MapStr(d, "p", ""),
		Q:         fw.MapStr(d, "q", ""),
		G:         fw.MapStr(d, "g", ""),
		X:         fw.MapStr(d, "x", defaultSecretX),
		Y:         fw.MapStr(d, "y", ""),
		R:         fw.MapStr(d, "r", defaultR),
		T:         fw.MapStr(d, "t", ""),
		C:         fw.MapStr(d, "c", ""),
		S:         fw.MapStr(d, "s", ""),
		Verified:  fw.MapBool(d, "verified", false),
		ForgeS:    fw.MapStr(d, "forge_s", ""),
		ForgeC:    fw.MapStr(d, "forge_c", ""),
		ForgeOK:   fw.MapBool(d, "forge_ok", false),
		LastError: fw.MapStr(d, "last_error", ""),
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["bits"] = st.Bits
	s.Data["seed"] = st.Seed
	s.Data["p"] = st.P
	s.Data["q"] = st.Q
	s.Data["g"] = st.G
	s.Data["x"] = st.X
	s.Data["y"] = st.Y
	s.Data["r"] = st.R
	s.Data["t"] = st.T
	s.Data["c"] = st.C
	s.Data["s"] = st.S
	s.Data["verified"] = st.Verified
	s.Data["forge_s"] = st.ForgeS
	s.Data["forge_c"] = st.ForgeC
	s.Data["forge_ok"] = st.ForgeOK
	s.Data["last_error"] = st.LastError
}

// loadParams 从快照重建 schnorrParams。
func (st snapState) loadParams() (schnorrParams, error) {
	p, _ := new(big.Int).SetString(st.P, 10)
	q, _ := new(big.Int).SetString(st.Q, 10)
	g, _ := new(big.Int).SetString(st.G, 10)
	if p == nil || q == nil || g == nil {
		return schnorrParams{}, errors.New("公开参数 (p, q, g) 未生成")
	}
	return schnorrParams{P: p, Q: q, G: g}, nil
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "零知识证明基础（Schnorr）",
		Description:         "演示离散对数零知识证明：Commit → Challenge → Response → Verify；含 Fiat-Shamir 与伪造攻击",
		Category:            fw.CategoryCryptography,
		AlgorithmType:       algorithmType,
		Version:             schemaVersion,
		TimeControlMode:     fw.TimeControlProcess,
		DataSourceMode:      fw.DataSourceSimulation,
		SupportedLinkGroups: []string{linkGroupCryptoVerify},

		// v0.5 协议字段。
		ExtensionLevel:     fw.ExtensionL1,
		LinkGroupVersion:   "v0.5.0",
		SupportsMultiActor: false,
		OwnedFieldPaths: []string{
			"proofs.zkp.public_y",
			"proofs.zkp.commitment_t",
			"proofs.zkp.challenge_c",
			"proofs.zkp.response_s",
		},

		DefaultParams: func() map[string]any { return map[string]any{} },
		DefaultState:  defaultState,
		Interaction:   interactionDefinition,
		Init:                initScene,
		Step:                stepScene,
		HandleAction:        handleAction,
	}
}

func defaultState() fw.SceneState {
	return fw.SceneState{
		SceneCode: sceneCode,
		Tick:      0,
		Phase:     "ready",
		Data: map[string]any{
			"bits": defaultBits,
			"seed": 1,
			"x":    defaultSecretX,
			"r":    defaultR,
		},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "generate_params", Label: "生成公开参数",
				Description:   "Miller-Rabin 生成安全素数 p=2q+1 与阶为 q 的生成元 g",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "bits", Type: fw.FieldNumber, Label: "p 位长", Required: true, Default: defaultBits, Min: minBits, Max: maxBits, Step: 8},
					{Name: "seed", Type: fw.FieldNumber, Label: "随机种子", Required: true, Default: 1, Min: 0, Step: 1},
				},
			},
			{
				ActionCode: "set_secret", Label: "设置秘密 x",
				Description:   "派生 y = g^x mod p（公开给 Verifier）",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "x", Type: fw.FieldString, Label: "秘密 x（十进制）", Required: true, Default: defaultSecretX},
				},
				WritesOwnedFields: []string{"proofs.zkp.public_y"},
				LinkOwnerFields:   []string{"proofs.zkp.public_y"},
			},
			{
				ActionCode: "prove_interactive", Label: "交互式证明（3 步）",
				Description:   "Prover 选 r 计算 t；Verifier 给 c；Prover 给 s",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "r", Type: fw.FieldString, Label: "Prover 随机 r", Required: true, Default: defaultR},
					{Name: "c", Type: fw.FieldString, Label: "Verifier 挑战 c", Required: true, Default: "3"},
				},
				WritesOwnedFields: []string{
					"proofs.zkp.commitment_t",
					"proofs.zkp.challenge_c",
					"proofs.zkp.response_s",
				},
				LinkOwnerFields: []string{
					"proofs.zkp.commitment_t",
					"proofs.zkp.challenge_c",
					"proofs.zkp.response_s",
					"proofs.zkp.verified",
				},
			},
			{
				ActionCode: "prove_fiat_shamir", Label: "Fiat-Shamir 非交互证明",
				Description:   "c = SHA-256(g||y||t) mod q；自动一次性生成完整证明",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "r", Type: fw.FieldString, Label: "Prover 随机 r", Required: true, Default: defaultR},
				},
				LinkOwnerFields: []string{"proofs.zkp.verified"},
			},
			{
				ActionCode: "forge_attempt", Label: "伪造证明（不知 x）",
				Description:   "随机猜 s 与 c → 验证必失败（演示 ZKP 健全性 soundness）",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
			},
			{
				ActionCode: "reset", Label: "重置",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneReset,
			},
			{
				ActionCode:    "teacher_set_demo_input",
				Label:         "教师设置演示输入",
				Description:   "仅教师可用，设置演示输入用于教学展示",
				Category:      fw.ActionParamTune,
				Trigger:       fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleTeacher},
				InterveneType: fw.InterveneHint,
				Fields: []fw.FieldDef{
					{Name: "description", Type: fw.FieldString, Label: "描述", Required: false, Default: "教师设置演示输入"},
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
	if err := generateAndSetParams(&st); err != nil {
		st.LastError = err.Error()
	} else {
		setSecretAndDerive(&st, st.X)
	}
	saveState(state, st)
	state.Phase = "ready"
	env := buildEnvelope(st, "init", "ZKP 初始化（默认 32-bit 安全素数）", true)
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
	case "generate_params":
		st.Bits = fw.MapInt(in.Params, "bits", defaultBits)
		st.Seed = int64(fw.MapInt(in.Params, "seed", 1))
		if err := generateAndSetParams(&st); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		setSecretAndDerive(&st, st.X)
		st.T = ""
		st.C = ""
		st.S = ""
		st.Verified = false
		saveState(state, st)
		out.Render = buildEnvelope(st, "generate_params", fmt.Sprintf("参数已生成（p 位长 = %d）", st.Bits), true)
		appendParamsMicroSteps(&out.Render)
		return out, nil

	case "set_secret":
		x := fw.MapStr(in.Params, "x", defaultSecretX)
		if err := setSecretAndDerive(&st, x); err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		st.T = ""
		st.C = ""
		st.S = ""
		st.Verified = false
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_secret", "派生公开 y = g^x mod p", false)
		appendSecretMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "prove_interactive":
		params, err := st.loadParams()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		rStr := fw.MapStr(in.Params, "r", defaultR)
		cStr := fw.MapStr(in.Params, "c", "3")
		r, ok1 := new(big.Int).SetString(rStr, 10)
		c, ok2 := new(big.Int).SetString(cStr, 10)
		if !ok1 || !ok2 {
			return fw.ActionOutput{Success: false, ErrorMessage: "r / c 必须是十进制整数"}, nil
		}
		r.Mod(r, params.Q)
		c.Mod(c, params.Q)
		st.R = r.String()
		t := params.commit(r)
		st.T = t.String()
		st.C = c.String()
		x, _ := new(big.Int).SetString(st.X, 10)
		s := params.response(r, c, x)
		st.S = s.String()
		y, _ := new(big.Int).SetString(st.Y, 10)
		st.Verified = params.verify(y, t, c, s)
		saveState(state, st)
		summary := fmt.Sprintf("交互式证明：%s", verifyTag(st.Verified))
		out.Render = buildEnvelope(st, "prove_interactive", summary, false)
		appendProveMicroSteps(&out.Render, false, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "prove_fiat_shamir":
		params, err := st.loadParams()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		rStr := fw.MapStr(in.Params, "r", defaultR)
		r, ok := new(big.Int).SetString(rStr, 10)
		if !ok {
			return fw.ActionOutput{Success: false, ErrorMessage: "r 必须是十进制整数"}, nil
		}
		r.Mod(r, params.Q)
		st.R = r.String()
		t := params.commit(r)
		st.T = t.String()
		y, _ := new(big.Int).SetString(st.Y, 10)
		c := params.fiatShamirChallenge(y, t)
		st.C = c.String()
		x, _ := new(big.Int).SetString(st.X, 10)
		s := params.response(r, c, x)
		st.S = s.String()
		st.Verified = params.verify(y, t, c, s)
		saveState(state, st)
		summary := fmt.Sprintf("Fiat-Shamir 证明：%s", verifyTag(st.Verified))
		out.Render = buildEnvelope(st, "prove_fiat_shamir", summary, false)
		appendProveMicroSteps(&out.Render, true, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "forge_attempt":
		params, err := st.loadParams()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		// 不知 x 时随机猜 s 和 c，构造 t 满足 g^s ≡ t·y^c：
		// 等价于 t = g^s · y^(-c) mod p（这样验证一定成功，但 c 与 t 顺序错了 → 现实 verifier 先 commit 再发 c → 攻击失败）
		// 为了演示真实攻击：先随机猜 s, c → 然后看 verify 失败（攻击者无法控制 t）。
		rng := rand.New(rand.NewSource(st.Seed + 1000))
		fakeS := new(big.Int).Rand(rng, params.Q)
		fakeC := new(big.Int).Rand(rng, params.Q)
		// 用现有 t（如果有），或随机生成一个 t（攻击者无 r 也只能任选一个 t）
		fakeT := new(big.Int).Rand(rng, params.P)
		if fakeT.Sign() == 0 {
			fakeT.SetInt64(1)
		}
		y, _ := new(big.Int).SetString(st.Y, 10)
		st.ForgeS = fakeS.String()
		st.ForgeC = fakeC.String()
		st.ForgeOK = params.verify(y, fakeT, fakeC, fakeS)
		saveState(state, st)
		summary := "随机伪造：验证失败（健全性保证）"
		if st.ForgeOK {
			summary = "随机伪造：极小概率验证通过（≤ 1/q）"
		}
		out.Render = buildEnvelope(st, "forge_attempt", summary, false)
		appendForgeMicroSteps(&out.Render, st.ForgeOK)
		return out, nil

	case "teacher_set_demo_input":
		if in.UserRole != fw.RoleTeacher {
			return fw.ActionOutput{Success: false, ErrorMessage: "仅教师可调用"}, nil
		}
		desc, _ := in.Params["description"].(string)
		if desc == "" {
			desc = "教师设置演示输入"
		}
		annot := fw.PrimAnnotation(fmt.Sprintf("teacher-hint-%d", state.Tick), "text", "teacher", 5000, map[string]any{"x": 0.5, "y": 0.1}, nil, desc)
		return fw.ActionOutput{
			Success: true,
			Render: fw.RenderEnvelope{
				Primitives:  []fw.Primitive{annot},
				ChangedKeys: []string{"teacher_action"},
			},
		}, nil

	case "reset":
		st = snapState{Bits: defaultBits, Seed: 1, X: defaultSecretX, R: defaultR}
		_ = generateAndSetParams(&st)
		setSecretAndDerive(&st, st.X)
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// generateAndSetParams 使用当前 Bits / Seed 生成 (p, q, g)。
func generateAndSetParams(st *snapState) error {
	if st.Bits < minBits || st.Bits > maxBits {
		return fmt.Errorf("bits 越界 [%d,%d]", minBits, maxBits)
	}
	rng := rand.New(rand.NewSource(st.Seed))
	p, q := generateSafePrime(st.Bits, rng)
	g := findGenerator(p, q, rng)
	st.P = p.String()
	st.Q = q.String()
	st.G = g.String()
	return nil
}

// setSecretAndDerive 设置 x 并派生 y = g^x mod p。
func setSecretAndDerive(st *snapState, xStr string) error {
	params, err := st.loadParams()
	if err != nil {
		return err
	}
	x, ok := new(big.Int).SetString(xStr, 10)
	if !ok {
		return errors.New("x 必须是十进制整数")
	}
	x.Mod(x, params.Q)
	if x.Sign() == 0 {
		return errors.New("x 不能为 0 mod q")
	}
	y := new(big.Int).Exp(params.G, x, params.P)
	st.X = x.String()
	st.Y = y.String()
	return nil
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func activePhase(st snapState) int {
	switch {
	case st.P == "":
		return 0
	case st.Y == "":
		return 1
	case st.T == "":
		return 2
	case st.C == "":
		return 3
	case st.S == "":
		return 4
	default:
		return 5
	}
}

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	active := activePhase(st)
	prims := make([]fw.Primitive, 0, 30)

	// 1) 流水线 6 节点
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))
	for i, id := range pipelineNodeIDs {
		status := "normal"
		if i == active {
			status = "active"
		}
		role := []string{"params", "secret", "commit", "challenge", "response", "verify"}[i]
		label := []string{"参数", "y=gˣ", "t=gʳ", "c", "s", "Verify"}[i]
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		anim := ""
		if i == active-1 {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-%d-%d", i, i+1),
			pipelineNodeIDs[i], pipelineNodeIDs[i+1], "solid", anim))
	}

	prog := float64(active) / 5.0
	if active >= 5 {
		prog = 1.0
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, minInt(active, len(phaseLabels)-1), prog))

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-protocol",
		`\text{Commit: } t = g^r \bmod p;\ \text{Challenge: } c \in [0,q);\ \text{Response: } s = r + c\cdot x \bmod q`, false))
	prims = append(prims, fw.PrimMathFormula("formula-verify",
		`\text{Verify: } g^s \stackrel{?}{=} t \cdot y^c \pmod p`, false))
	prims = append(prims, fw.PrimMathFormula("formula-fs",
		`\text{Fiat-Shamir: } c = \mathrm{SHA256}(g\,\|\,y\,\|\,t) \bmod q`, false))

	// 3) 公开参数
	prims = append(prims, fw.PrimCodeBlock("cb-params",
		fmt.Sprintf("p = %s\nq = %s\ng = %s\nbits = %d, seed = %d", st.P, st.Q, st.G, st.Bits, st.Seed),
		"text", nil, 6))

	// 4) 秘密 / 公开
	prims = append(prims, fw.PrimCodeBlock("cb-secret",
		fmt.Sprintf("秘密 x（仅 Prover）:\nx = %s\n\n公开 y（公开给 Verifier）:\ny = g^x mod p\n  = %s", st.X, st.Y),
		"text", nil, 6))

	// 5) 协议过程
	protoLines := []string{
		fmt.Sprintf("r (Prover 选)         = %s", st.R),
		fmt.Sprintf("t = g^r mod p         = %s", st.T),
		fmt.Sprintf("c (Verifier / FS)     = %s", st.C),
		fmt.Sprintf("s = r + c·x mod q     = %s", st.S),
	}
	if st.S != "" {
		// 计算验证两侧具体值
		params, err := st.loadParams()
		if err == nil {
			y, _ := new(big.Int).SetString(st.Y, 10)
			t, _ := new(big.Int).SetString(st.T, 10)
			c, _ := new(big.Int).SetString(st.C, 10)
			s, _ := new(big.Int).SetString(st.S, 10)
			lhs := new(big.Int).Exp(params.G, s, params.P)
			yc := new(big.Int).Exp(y, c, params.P)
			rhs := new(big.Int).Mul(t, yc)
			rhs.Mod(rhs, params.P)
			protoLines = append(protoLines,
				"",
				fmt.Sprintf("g^s mod p           = %s", lhs.String()),
				fmt.Sprintf("(t · y^c) mod p     = %s", rhs.String()),
			)
			if st.Verified {
				protoLines = append(protoLines, "✓ 两侧相等 → 证明有效（且零知识：未泄漏 x）")
			} else {
				protoLines = append(protoLines, "✗ 两侧不等 → 证明无效")
			}
		}
	} else {
		protoLines = append(protoLines, "（尚未发起证明）")
	}
	prims = append(prims, fw.PrimCodeBlock("cb-protocol", strings.Join(protoLines, "\n"), "text", nil, 12))

	// 6) 伪造攻击演示
	if st.ForgeS != "" {
		atkLines := []string{
			"攻击者尝试（不知 x）：",
			fmt.Sprintf("  随机 s = %s", st.ForgeS),
			fmt.Sprintf("  随机 c = %s", st.ForgeC),
			"",
		}
		if st.ForgeOK {
			atkLines = append(atkLines, "⚠ 极小概率验证通过（≤ 1/q）")
		} else {
			atkLines = append(atkLines, "✓ 验证失败 → ZKP 健全性保证攻击不可行")
		}
		prims = append(prims, fw.PrimCodeBlock("cb-forge", strings.Join(atkLines, "\n"), "text", nil, 6))
	}

	// 7) 动效
	prims = append(prims, fw.PrimGlow("glow-active", pipelineNodeIDs[active], "info", 0.8))
	verifyColor := "info"
	if st.S != "" {
		if st.Verified {
			verifyColor = "success"
		} else {
			verifyColor = "danger"
		}
	}
	prims = append(prims, fw.PrimPulse("pulse-verify", "cb-protocol", verifyColor, 1500))

	// 8) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))

	// 9) 错误
	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "ZKP 错误", st.LastError, "scene", "请检查输入", true))
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
		"bits":     st.Bits,
		"seed":     st.Seed,
		"p":        st.P,
		"q":        st.Q,
		"g":        st.G,
		"y":        st.Y,
		"t":        st.T,
		"c":        st.C,
		"s":        st.S,
		"verified": st.Verified,
		"forge_ok": st.ForgeOK,
		"protocol": "schnorr",
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendParamsMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "p-1", Label: "Miller-Rabin 找 q（素数）", DurationMs: 600, HighlightIDs: []string{"cb-params"}, FirePrimitives: []string{"glow-active"}, ParentPhase: "setup"},
		{ID: "p-2", Label: "p = 2q + 1（安全素数）", DurationMs: 500, HighlightIDs: []string{"cb-params"}},
		{ID: "p-3", Label: "找阶为 q 的生成元 g", DurationMs: 500, HighlightIDs: []string{"cb-params"}},
	}
}

func appendSecretMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "s-1", Label: "Prover 设秘密 x", DurationMs: 400, HighlightIDs: []string{"cb-secret"}},
		{ID: "s-2", Label: "派生 y = g^x mod p", DurationMs: 500, HighlightIDs: []string{"cb-secret"}, FirePrimitives: []string{"glow-active"}},
		{ID: "s-3", Label: "公开 y 给 Verifier", DurationMs: 400, HighlightIDs: []string{"cb-secret"}, IsLinkTrigger: true},
	}
}

func appendProveMicroSteps(env *fw.RenderEnvelope, fiatShamir bool, ok bool) {
	verb := "Verifier 选随机 c"
	if fiatShamir {
		verb = "Fiat-Shamir: c = SHA256(g||y||t) mod q"
	}
	tail := "g^s = t·y^c → 证明有效 ✓"
	if !ok {
		tail = "验证失败 ✗"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "pv-1", Label: "Prover 选随机 r → 计算 t = g^r mod p", DurationMs: 500, HighlightIDs: []string{"cb-protocol", "formula-protocol"}, FirePrimitives: []string{"glow-active"}},
		{ID: "pv-2", Label: verb, DurationMs: 500, HighlightIDs: []string{"cb-protocol", "formula-fs"}},
		{ID: "pv-3", Label: "Prover 计算 s = r + c·x mod q", DurationMs: 500, HighlightIDs: []string{"cb-protocol"}},
		{ID: "pv-4", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-protocol", "formula-verify"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
	}
}

func appendForgeMicroSteps(env *fw.RenderEnvelope, ok bool) {
	tail := "验证失败 → 健全性保证（攻击者无 x 不能伪造）"
	if ok {
		tail = "极小概率通过 → 概率 ≤ 1/q"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "fg-1", Label: "攻击者随机猜 s, c", DurationMs: 500, HighlightIDs: []string{"cb-forge"}, FirePrimitives: []string{"glow-active"}},
		{ID: "fg-2", Label: "Verifier 用同样验证规则检查", DurationMs: 500, HighlightIDs: []string{"formula-verify", "cb-forge"}},
		{ID: "fg-3", Label: tail, DurationMs: 500, HighlightIDs: []string{"cb-forge"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
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
		ID:             "zkp-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_proof",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"proofs.zkp.public_y", "proofs.zkp.commitment_t", "proofs.zkp.response_s"},
		Payload:        map[string]any{"y": st.Y, "t": st.T, "s": st.S},
		SourceAnchorID: "zkp-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "proofs.zkp.public_y", "proofs.zkp.commitment_t")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"proofs": map[string]any{
			"zkp": map[string]any{
				"protocol":     "schnorr",
				"public_y":     st.Y,
				"commitment_t": st.T,
				"challenge_c":  st.C,
				"response_s":   st.S,
				"verified":     st.Verified,
			},
		},
	}
}

// =====================================================================
// 工具
// =====================================================================

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func verifyTag(ok bool) string {
	if ok {
		return "✓ 证明有效"
	}
	return "✗ 证明无效"
}
