// 模块：sim-engine/scenarios/internal/cryptography/rsaencrypt
// 文件职责：CRY-04 RSA 加密 / 解密 / 签名 / 验证场景的完整实现。
//
// SSOT 依据：06.md §4.3.4 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：自实现完整 RSA 算法，零依赖 crypto/rsa 等任何第三方库；只用
// 标准库 math/big、rand.New(rand.NewSource) 确定性 RNG、复用 sha256hash.Sum256。包含：
//
//   · Miller-Rabin 素性测试（自实现，多轮）
//   · 随机奇数试除筛 + Miller-Rabin 生成 N-bit 素数 p / q
//   · n = p·q；φ(n) = (p-1)(q-1)；公钥 e（默认 65537）；
//     私钥 d = e^(-1) mod φ(n)（math/big.ModInverse 仅 GCD 算法，自实现可证明无依赖）
//   · 加密 c = m^e mod n；解密 m = c^d mod n（math/big.Exp 是模幂硬件无关算法）
//   · 教学签名：s = SHA-256(m)^d mod n；验证 SHA-256(m) ?= s^e mod n
//   · 攻击演示：暴力枚举 √n 寻找因子（仅在 n 较小时教学有效）
//
// 教学决策：
//   - 流水线 P4：keygen → encrypt → decrypt → sign → verify
//   - 全部展示：p, q, n, φ, e, d, m, c, decrypted_m, signature, verify_result

package rsaencrypt

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
	sceneCode     = "rsa-encrypt"
	schemaVersion = "v1.0.0"
	algorithmType = "rsa"

	defaultBits      = 64 // 教学位长（每个素数 64-bit → n ≈ 128-bit），保证生成 / 演示快速
	maxBits          = 256
	minBits          = 16
	defaultE         = 65537
	defaultMessage   = "42"
	millerRabinTries = 10

	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "encryptions.rsa"
)

var pipelineNodeIDs = []string{
	"phase-keygen", "phase-encrypt", "phase-decrypt", "phase-sign", "phase-verify",
}
var phaseLabels = []string{"密钥生成", "加密 c=mᵉ", "解密 m=cᵈ", "签名 s=H(m)ᵈ"}

// =====================================================================
// Miller-Rabin 素性测试（自实现）
// =====================================================================

// millerRabin 用 Miller-Rabin 算法判断 n 是否为素数（rounds 轮独立测试）。
// 详细原理（FIPS 186-4 附录 C）：把 n-1 = 2^r·d；选随机 a；若 aᵈ ≡ 1 (mod n) 或
// 存在 0 ≤ s < r 使 a^(2ˢ·d) ≡ -1 (mod n) → 可能素数；否则合数。
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
	// 试除小素数加速（避免大量 Miller-Rabin 回合用于明显合数）
	smallPrimes := []int64{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
	for _, sp := range smallPrimes {
		spB := big.NewInt(sp)
		if n.Cmp(spB) == 0 {
			return true
		}
		if new(big.Int).Mod(n, spB).Sign() == 0 {
			return false
		}
	}
	// 分解 n-1 = 2^r · d
	d := new(big.Int).Sub(n, big.NewInt(1))
	r := 0
	for d.Bit(0) == 0 {
		d.Rsh(d, 1)
		r++
	}
	nMinus1 := new(big.Int).Sub(n, big.NewInt(1))
	nMinus3 := new(big.Int).Sub(n, big.NewInt(3))
	for i := 0; i < rounds; i++ {
		// a ∈ [2, n-2]
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

// generatePrime 生成 bits 位的素数（教学：bits 不大时几毫秒可完成）。
func generatePrime(bits int, rng *rand.Rand) *big.Int {
	for {
		// 随机 bits 位奇数（最高位与最低位置 1）
		buf := make([]byte, (bits+7)/8)
		for i := range buf {
			buf[i] = byte(rng.Intn(256))
		}
		// 强制最高 bits-1 位为 1（保证位长），最低位为 1（保证奇）
		buf[0] |= 0x80
		buf[len(buf)-1] |= 0x01
		// 多余位清零
		extra := uint(len(buf)*8 - bits)
		buf[0] &= byte(0xFF >> extra)
		buf[0] |= byte(0x80 >> extra)
		p := new(big.Int).SetBytes(buf)
		if millerRabin(p, millerRabinTries, rng) {
			return p
		}
	}
}

// =====================================================================
// RSA 算法
// =====================================================================

type rsaKey struct {
	N   *big.Int // 模数
	E   *big.Int // 公钥指数
	D   *big.Int // 私钥指数
	P   *big.Int // 素数 1
	Q   *big.Int // 素数 2
	Phi *big.Int // φ(n)
}

// generateKeyPair 生成 bits 位 RSA 密钥对（每个素数 bits 位 → n 大约 2·bits 位）。
func generateKeyPair(bits int, eVal int, seed int64) (rsaKey, error) {
	if bits < minBits || bits > maxBits {
		return rsaKey{}, fmt.Errorf("bits 越界 [%d,%d]", minBits, maxBits)
	}
	rng := rand.New(rand.NewSource(seed))
	for retry := 0; retry < 8; retry++ {
		p := generatePrime(bits, rng)
		q := generatePrime(bits, rng)
		if p.Cmp(q) == 0 {
			continue
		}
		n := new(big.Int).Mul(p, q)
		phi := new(big.Int).Mul(
			new(big.Int).Sub(p, big.NewInt(1)),
			new(big.Int).Sub(q, big.NewInt(1)),
		)
		e := big.NewInt(int64(eVal))
		// 必须 gcd(e, φ) = 1
		gcd := new(big.Int).GCD(nil, nil, e, phi)
		if gcd.Cmp(big.NewInt(1)) != 0 {
			continue
		}
		d := new(big.Int).ModInverse(e, phi)
		if d == nil {
			continue
		}
		return rsaKey{N: n, E: e, D: d, P: p, Q: q, Phi: phi}, nil
	}
	return rsaKey{}, errors.New("RSA 密钥生成多次失败，请换 seed 或 bits")
}

// encrypt c = m^e mod n。
func (k rsaKey) encrypt(m *big.Int) *big.Int {
	return new(big.Int).Exp(m, k.E, k.N)
}

// decrypt m = c^d mod n。
func (k rsaKey) decrypt(c *big.Int) *big.Int {
	return new(big.Int).Exp(c, k.D, k.N)
}

// sign s = H(m)^d mod n（教学版 RSA 签名，无 PKCS#1 padding）。
func (k rsaKey) sign(m string) *big.Int {
	h := sha256hash.Sum256([]byte(m))
	hashInt := new(big.Int).SetBytes(h[:])
	hashInt.Mod(hashInt, k.N) // 缩到 [0, n)，便于小 n 教学
	return new(big.Int).Exp(hashInt, k.D, k.N)
}

// verify SHA-256(m) ?= s^e mod n。
func (k rsaKey) verify(m string, s *big.Int) bool {
	h := sha256hash.Sum256([]byte(m))
	hashInt := new(big.Int).SetBytes(h[:])
	hashInt.Mod(hashInt, k.N)
	recovered := new(big.Int).Exp(s, k.E, k.N)
	return recovered.Cmp(hashInt) == 0
}

// factorN 暴力试除 n（教学：仅当 n 较小时可成功；演示 RSA 安全性依赖大数分解难题）。
// 返回 (p, q)，若超时未找到则 (nil, nil)。
func factorN(n *big.Int, maxIters int) (*big.Int, *big.Int, int) {
	// 从 2 起按奇数试除（除 2 外）。
	if new(big.Int).Mod(n, big.NewInt(2)).Sign() == 0 {
		return big.NewInt(2), new(big.Int).Quo(n, big.NewInt(2)), 1
	}
	i := big.NewInt(3)
	two := big.NewInt(2)
	iterations := 0
	for iterations < maxIters {
		iterations++
		sq := new(big.Int).Mul(i, i)
		if sq.Cmp(n) > 0 {
			return nil, nil, iterations
		}
		if new(big.Int).Mod(n, i).Sign() == 0 {
			return new(big.Int).Set(i), new(big.Int).Quo(n, i), iterations
		}
		i.Add(i, two)
	}
	return nil, nil, iterations
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	Bits        int
	Seed        int64
	N           string
	E           string
	D           string
	P           string
	Q           string
	Phi         string
	Message     string
	Ciphertext  string
	Decrypted   string
	Signature   string
	Verified    bool
	FactorP     string
	FactorQ     string
	FactorIters int
	LastError   string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{Bits: defaultBits, Seed: 1, Message: defaultMessage, E: fmt.Sprintf("%d", defaultE)}
	}
	d := s.Data
	return snapState{
		Bits:        fw.MapInt(d, "bits", defaultBits),
		Seed:        int64(fw.MapInt(d, "seed", 1)),
		N:           fw.MapStr(d, "n", ""),
		E:           fw.MapStr(d, "e", fmt.Sprintf("%d", defaultE)),
		D:           fw.MapStr(d, "d", ""),
		P:           fw.MapStr(d, "p", ""),
		Q:           fw.MapStr(d, "q", ""),
		Phi:         fw.MapStr(d, "phi", ""),
		Message:     fw.MapStr(d, "message", defaultMessage),
		Ciphertext:  fw.MapStr(d, "ciphertext", ""),
		Decrypted:   fw.MapStr(d, "decrypted", ""),
		Signature:   fw.MapStr(d, "signature", ""),
		Verified:    fw.MapBool(d, "verified", false),
		FactorP:     fw.MapStr(d, "factor_p", ""),
		FactorQ:     fw.MapStr(d, "factor_q", ""),
		FactorIters: fw.MapInt(d, "factor_iters", 0),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["bits"] = st.Bits
	s.Data["seed"] = st.Seed
	s.Data["n"] = st.N
	s.Data["e"] = st.E
	s.Data["d"] = st.D
	s.Data["p"] = st.P
	s.Data["q"] = st.Q
	s.Data["phi"] = st.Phi
	s.Data["message"] = st.Message
	s.Data["ciphertext"] = st.Ciphertext
	s.Data["decrypted"] = st.Decrypted
	s.Data["signature"] = st.Signature
	s.Data["verified"] = st.Verified
	s.Data["factor_p"] = st.FactorP
	s.Data["factor_q"] = st.FactorQ
	s.Data["factor_iters"] = st.FactorIters
	s.Data["last_error"] = st.LastError
}

// loadKey 从快照状态重建 rsaKey（用于 encrypt/decrypt/sign/verify）。
func (st snapState) loadKey() (rsaKey, error) {
	if st.N == "" || st.D == "" {
		return rsaKey{}, errors.New("RSA 密钥未生成")
	}
	n, _ := new(big.Int).SetString(st.N, 10)
	e, _ := new(big.Int).SetString(st.E, 10)
	d, _ := new(big.Int).SetString(st.D, 10)
	p, _ := new(big.Int).SetString(st.P, 10)
	q, _ := new(big.Int).SetString(st.Q, 10)
	phi, _ := new(big.Int).SetString(st.Phi, 10)
	if n == nil || e == nil || d == nil {
		return rsaKey{}, errors.New("密钥反序列化失败")
	}
	return rsaKey{N: n, E: e, D: d, P: p, Q: q, Phi: phi}, nil
}

// regenerateKeys 重新生成 RSA 密钥对，刷新所有派生字段。
func (st *snapState) regenerateKeys() {
	st.LastError = ""
	eVal := defaultE
	if v, err := strconvAtoi(st.E); err == nil {
		eVal = v
	}
	k, err := generateKeyPair(st.Bits, eVal, st.Seed)
	if err != nil {
		st.LastError = err.Error()
		return
	}
	st.N = k.N.String()
	st.E = k.E.String()
	st.D = k.D.String()
	st.P = k.P.String()
	st.Q = k.Q.String()
	st.Phi = k.Phi.String()
	st.Ciphertext = ""
	st.Decrypted = ""
	st.Signature = ""
	st.Verified = false
	st.FactorP = ""
	st.FactorQ = ""
	st.FactorIters = 0
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "RSA 加密 / 签名",
		Description:         "演示 RSA：素数生成 → 密钥派生 → 加解密 → 签名验证 → 大数分解攻击",
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
			"encryptions.rsa.n",
			"encryptions.rsa.e",
			"encryptions.rsa.message",
			"encryptions.rsa.ciphertext",
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
			"bits":    defaultBits,
			"seed":    1,
			"message": defaultMessage,
			"e":       fmt.Sprintf("%d", defaultE),
		},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "generate_keys", Label: "生成密钥对",
				Description:   "用 Miller-Rabin 生成 p, q（每个 bits 位），派生 (n, e, d)",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "bits", Type: fw.FieldNumber, Label: "每素数位长", Required: true, Default: defaultBits, Min: minBits, Max: maxBits, Step: 8},
					{Name: "seed", Type: fw.FieldNumber, Label: "随机种子（可复现）", Required: true, Default: 1, Min: 0, Step: 1},
					{Name: "e", Type: fw.FieldString, Label: "公钥指数 e", Required: true, Default: fmt.Sprintf("%d", defaultE)},
				},
				WritesOwnedFields: []string{"encryptions.rsa.n", "encryptions.rsa.e"},
				LinkOwnerFields:   []string{"encryptions.rsa.n", "encryptions.rsa.e"},
			},
			{
				ActionCode: "encrypt", Label: "加密消息",
				Description:   "c = m^e mod n（消息 m 须 < n；教学允许十进制或 hex）",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "message", Type: fw.FieldString, Label: "明文 m（十进制）", Required: true, Default: defaultMessage},
				},
				WritesOwnedFields: []string{"encryptions.rsa.message", "encryptions.rsa.ciphertext"},
				LinkOwnerFields:   []string{"encryptions.rsa.message", "encryptions.rsa.ciphertext"},
			},
			{
				ActionCode: "decrypt", Label: "解密密文",
				Description:   "m = c^d mod n",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
			},
			{
				ActionCode: "sign", Label: "签名消息",
				Description:     "s = H(m)^d mod n（教学版，无 PKCS#1 padding）",
				Category:        fw.ActionPrimary, Trigger: fw.TriggerImmediate,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:   fw.InterveneState,
				LinkOwnerFields: []string{"encryptions.rsa.signature"},
			},
			{
				ActionCode: "verify", Label: "验证签名",
				Description:     "H(m) ?= s^e mod n",
				Category:        fw.ActionObserve, Trigger: fw.TriggerImmediate,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:   fw.InterveneHint,
				LinkOwnerFields: []string{"encryptions.rsa.verified"},
			},
			{
				ActionCode: "factor_attack", Label: "暴力分解 n（攻击）",
				Description:   "演示 RSA 安全性依赖大数分解难题；n 较小时可成功",
				Category:      fw.ActionAttackInject, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneAttack,
				Fields: []fw.FieldDef{
					{Name: "max_iters", Type: fw.FieldNumber, Label: "最大试除轮数", Required: true, Default: 1000000, Min: 1000, Max: 100000000, Step: 1000},
				},
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
	st.regenerateKeys()
	saveState(state, st)
	state.Phase = "keygen"
	env := buildEnvelope(st, "init", "RSA 初始化（默认 64-bit×2 → n ≈ 128-bit）", true)
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
	case "generate_keys":
		st.Bits = fw.MapInt(in.Params, "bits", defaultBits)
		st.Seed = int64(fw.MapInt(in.Params, "seed", 1))
		st.E = fw.MapStr(in.Params, "e", fmt.Sprintf("%d", defaultE))
		st.regenerateKeys()
		if st.LastError != "" {
			return fw.ActionOutput{Success: false, ErrorMessage: st.LastError}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "generate_keys", fmt.Sprintf("生成新密钥（%d-bit×2）", st.Bits), true)
		appendKeygenMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "encrypt":
		k, err := st.loadKey()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		msg := fw.MapStr(in.Params, "message", defaultMessage)
		st.Message = msg
		m, ok := new(big.Int).SetString(msg, 10)
		if !ok {
			return fw.ActionOutput{Success: false, ErrorMessage: "明文必须是十进制整数"}, nil
		}
		if m.Cmp(k.N) >= 0 {
			return fw.ActionOutput{Success: false, ErrorMessage: "明文 m 必须 < n"}, nil
		}
		c := k.encrypt(m)
		st.Ciphertext = c.String()
		st.Decrypted = ""
		saveState(state, st)
		out.Render = buildEnvelope(st, "encrypt", "已加密 c = m^e mod n", false)
		appendEncryptMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "decrypt":
		k, err := st.loadKey()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		if st.Ciphertext == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "尚无密文，请先 encrypt"}, nil
		}
		c, _ := new(big.Int).SetString(st.Ciphertext, 10)
		m := k.decrypt(c)
		st.Decrypted = m.String()
		saveState(state, st)
		out.Render = buildEnvelope(st, "decrypt", "已解密 m = c^d mod n", false)
		appendDecryptMicroSteps(&out.Render)
		return out, nil

	case "sign":
		k, err := st.loadKey()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		s := k.sign(st.Message)
		st.Signature = s.String()
		st.Verified = false
		saveState(state, st)
		out.Render = buildEnvelope(st, "sign", "已签名 s = H(m)^d mod n", false)
		appendSignMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "verify":
		k, err := st.loadKey()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		if st.Signature == "" {
			return fw.ActionOutput{Success: false, ErrorMessage: "尚无签名，请先 sign"}, nil
		}
		s, _ := new(big.Int).SetString(st.Signature, 10)
		st.Verified = k.verify(st.Message, s)
		saveState(state, st)
		summary := "签名有效 ✓"
		if !st.Verified {
			summary = "签名无效 ✗"
		}
		out.Render = buildEnvelope(st, "verify", summary, false)
		appendVerifyMicroSteps(&out.Render, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "factor_attack":
		k, err := st.loadKey()
		if err != nil {
			return fw.ActionOutput{Success: false, ErrorMessage: err.Error()}, nil
		}
		maxIters := fw.MapInt(in.Params, "max_iters", 1000000)
		p, q, iters := factorN(k.N, maxIters)
		st.FactorIters = iters
		if p != nil && q != nil {
			st.FactorP = p.String()
			st.FactorQ = q.String()
		} else {
			st.FactorP = ""
			st.FactorQ = ""
		}
		saveState(state, st)
		summary := fmt.Sprintf("试除 %d 轮：未找到因子（n 太大）", iters)
		if p != nil {
			summary = fmt.Sprintf("试除 %d 轮：分解成功 → p, q 推回 → 私钥泄漏", iters)
		}
		out.Render = buildEnvelope(st, "factor_attack", summary, false)
		appendFactorMicroSteps(&out.Render, p != nil)
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
		st = snapState{Bits: defaultBits, Seed: 1, Message: defaultMessage, E: fmt.Sprintf("%d", defaultE)}
		st.regenerateKeys()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置并重新生成密钥", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func activePhase(st snapState) int {
	switch {
	case st.N == "":
		return 0
	case st.Ciphertext == "":
		return 1
	case st.Decrypted == "":
		return 2
	case st.Signature == "":
		return 3
	default:
		return 4
	}
}

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	active := activePhase(st)
	prims := make([]fw.Primitive, 0, 30)

	// 1) 流水线
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))
	for i, id := range pipelineNodeIDs {
		status := "normal"
		if i == active {
			status = "active"
		}
		role := []string{"keygen", "encrypt", "decrypt", "sign", "verify"}[i]
		label := []string{"密钥生成", "加密", "解密", "签名", "验证"}[i]
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		anim := ""
		if i == active-1 {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-%d-%d", i, i+1), pipelineNodeIDs[i], pipelineNodeIDs[i+1], "solid", anim))
	}

	prog := 0.25 * float64(active)
	if active >= 4 {
		prog = 1.0
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, minInt(active, len(phaseLabels)-1), prog))

	// 2) 公式
	prims = append(prims, fw.PrimMathFormula("formula-keygen",
		`n = p \cdot q,\ \ \varphi(n) = (p-1)(q-1),\ \ d \equiv e^{-1} \pmod{\varphi(n)}`, false))
	prims = append(prims, fw.PrimMathFormula("formula-enc-dec",
		`c = m^e \bmod n,\ \ m = c^d \bmod n`, false))
	prims = append(prims, fw.PrimMathFormula("formula-sign",
		`s = H(m)^d \bmod n,\ \ \mathrm{verify}: H(m) \stackrel{?}{=} s^e \bmod n`, false))

	// 3) 密钥参数 code_block
	prims = append(prims, fw.PrimCodeBlock("cb-primes",
		fmt.Sprintf("p = %s\n\nq = %s\n\nbits = %d, seed = %d", st.P, st.Q, st.Bits, st.Seed),
		"text", nil, 6))
	prims = append(prims, fw.PrimCodeBlock("cb-public",
		fmt.Sprintf("公钥 (n, e):\nn = %s\ne = %s", st.N, st.E),
		"text", nil, 4))
	prims = append(prims, fw.PrimCodeBlock("cb-private",
		fmt.Sprintf("私钥 d:\nd = %s\nφ(n) = %s", st.D, st.Phi),
		"text", nil, 4))

	// 4) 加解密 / 签名 / 验证
	encDecLines := []string{
		fmt.Sprintf("明文 m   = %s", st.Message),
		fmt.Sprintf("密文 c   = %s", st.Ciphertext),
		fmt.Sprintf("解密 m'  = %s", st.Decrypted),
	}
	if st.Decrypted != "" {
		if st.Decrypted == st.Message {
			encDecLines = append(encDecLines, "✓ m' = m → 解密成功")
		} else {
			encDecLines = append(encDecLines, "✗ m' ≠ m")
		}
	}
	prims = append(prims, fw.PrimCodeBlock("cb-encdec", strings.Join(encDecLines, "\n"), "text", nil, 6))

	sigLines := []string{fmt.Sprintf("H(m) = SHA-256(\"%s\")", st.Message)}
	if st.Signature != "" {
		sigLines = append(sigLines, fmt.Sprintf("签名 s = %s", st.Signature))
		if st.Verified {
			sigLines = append(sigLines, "", "✓ H(m) ≡ s^e mod n → 签名有效")
		} else {
			sigLines = append(sigLines, "", "（未验证或验证失败）")
		}
	} else {
		sigLines = append(sigLines, "（尚未签名）")
	}
	prims = append(prims, fw.PrimCodeBlock("cb-sig", strings.Join(sigLines, "\n"), "text", nil, 6))

	// 5) 攻击区
	if st.FactorIters > 0 {
		atkLines := []string{fmt.Sprintf("试除轮数: %d", st.FactorIters)}
		if st.FactorP != "" {
			atkLines = append(atkLines,
				fmt.Sprintf("还原 p = %s", st.FactorP),
				fmt.Sprintf("还原 q = %s", st.FactorQ),
				"⚠ 由 (p, q) 可立即推得 d → RSA 密钥泄漏")
		} else {
			atkLines = append(atkLines, "未找到因子（n 太大 / 试除轮数不够）",
				"→ 这正是 RSA 安全性的来源：大数分解难")
		}
		prims = append(prims, fw.PrimCodeBlock("cb-attack", strings.Join(atkLines, "\n"), "text", nil, 6))
	}

	// 6) 动效
	prims = append(prims, fw.PrimGlow("glow-active", pipelineNodeIDs[active], "info", 0.8))
	verifyColor := "info"
	if st.Verified {
		verifyColor = "success"
	} else if st.Signature != "" {
		verifyColor = "warning"
	}
	prims = append(prims, fw.PrimPulse("pulse-verify", "cb-sig", verifyColor, 1500))

	// 7) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))

	// 8) 错误
	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "RSA 错误", st.LastError, "scene", "请检查参数", true))
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
		"bits":         st.Bits,
		"seed":         st.Seed,
		"n":            st.N,
		"e":            st.E,
		"d":            st.D,
		"phi":          st.Phi,
		"message":      st.Message,
		"ciphertext":   st.Ciphertext,
		"decrypted":    st.Decrypted,
		"signature":    st.Signature,
		"verified":     st.Verified,
		"factor_p":     st.FactorP,
		"factor_q":     st.FactorQ,
		"factor_iters": st.FactorIters,
	}
	if summary != "" {
		d["summary"] = summary
	}
	return d
}

// =====================================================================
// MicroStep 模板
// =====================================================================

func appendKeygenMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "kg-1", Label: "随机 + Miller-Rabin 生成 p", DurationMs: 600, HighlightIDs: []string{"cb-primes"}, FirePrimitives: []string{"glow-active"}, ParentPhase: "keygen"},
		{ID: "kg-2", Label: "随机 + Miller-Rabin 生成 q", DurationMs: 600, HighlightIDs: []string{"cb-primes"}},
		{ID: "kg-3", Label: "n = p·q,  φ = (p-1)(q-1)", DurationMs: 500, HighlightIDs: []string{"formula-keygen", "cb-public"}},
		{ID: "kg-4", Label: "d ≡ e⁻¹ (mod φ)", DurationMs: 500, HighlightIDs: []string{"cb-private"}, IsLinkTrigger: true},
	}
}

func appendEncryptMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "en-1", Label: "读取明文 m（须 < n）", DurationMs: 400, HighlightIDs: []string{"cb-encdec"}},
		{ID: "en-2", Label: "模幂 c = m^e mod n", DurationMs: 600, HighlightIDs: []string{"formula-enc-dec"}, FirePrimitives: []string{"glow-active"}},
		{ID: "en-3", Label: "输出密文", DurationMs: 400, HighlightIDs: []string{"cb-encdec"}, IsLinkTrigger: true},
	}
}

func appendDecryptMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "de-1", Label: "用私钥 d 解密", DurationMs: 400, HighlightIDs: []string{"cb-private"}, FirePrimitives: []string{"glow-active"}},
		{ID: "de-2", Label: "模幂 m = c^d mod n", DurationMs: 600, HighlightIDs: []string{"formula-enc-dec"}},
		{ID: "de-3", Label: "校验 m' = m", DurationMs: 400, HighlightIDs: []string{"cb-encdec"}},
	}
}

func appendSignMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sn-1", Label: "H(m) = SHA-256(m)", DurationMs: 400, HighlightIDs: []string{"cb-sig"}},
		{ID: "sn-2", Label: "签名 s = H(m)^d mod n", DurationMs: 600, HighlightIDs: []string{"formula-sign"}, FirePrimitives: []string{"glow-active"}},
		{ID: "sn-3", Label: "输出签名", DurationMs: 400, HighlightIDs: []string{"cb-sig"}, IsLinkTrigger: true},
	}
}

func appendVerifyMicroSteps(env *fw.RenderEnvelope, ok bool) {
	tail := "签名有效 ✓"
	if !ok {
		tail = "签名无效 ✗"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "vf-1", Label: "用公钥 e 计算 s^e mod n", DurationMs: 500, HighlightIDs: []string{"formula-sign"}},
		{ID: "vf-2", Label: "与 H(m) 比较", DurationMs: 500, HighlightIDs: []string{"cb-sig"}, FirePrimitives: []string{"pulse-verify"}},
		{ID: "vf-3", Label: tail, DurationMs: 400, HighlightIDs: []string{"cb-sig"}, IsLinkTrigger: true},
	}
}

func appendFactorMicroSteps(env *fw.RenderEnvelope, success bool) {
	tail := "未找到因子 → RSA 安全"
	if success {
		tail = "找到 p, q → RSA 私钥泄漏"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "atk-1", Label: "从 i=3 起按奇数试除 n", DurationMs: 400, HighlightIDs: []string{"cb-attack"}, FirePrimitives: []string{"glow-active"}},
		{ID: "atk-2", Label: "i² > n 时停止", DurationMs: 400, HighlightIDs: []string{"cb-attack"}},
		{ID: "atk-3", Label: tail, DurationMs: 600, HighlightIDs: []string{"cb-attack"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
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
		ID:             "rsa-publish",
		SourceScene:    sceneCode,
		SourceAction:   "publish_keys",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"encryptions.rsa.n", "encryptions.rsa.ciphertext"},
		Payload:        map[string]any{"n": st.N, "ciphertext": st.Ciphertext},
		SourceAnchorID: "rsa-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "encryptions.rsa.n", "encryptions.rsa.ciphertext")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"encryptions": map[string]any{
			"rsa": map[string]any{
				"n":          st.N,
				"e":          st.E,
				"bits":       st.Bits,
				"message":    st.Message,
				"ciphertext": st.Ciphertext,
				"signature":  st.Signature,
				"verified":   st.Verified,
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

// strconvAtoi 简化的字符串到整数（避免引入 strconv 包并保持本地化）。
func strconvAtoi(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	sign := 1
	i := 0
	if s[0] == '-' {
		sign = -1
		i = 1
	} else if s[0] == '+' {
		i = 1
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("非法字符 %c", c)
		}
		n = n*10 + int(c-'0')
	}
	return n * sign, nil
}
