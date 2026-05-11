// 模块：sim-engine/scenarios/internal/cryptography/ecdsasign
// 文件职责：CRY-03 ECDSA（secp256k1）签名场景的完整实现。
//
// SSOT 依据：06.md §4.3.3 / §3.2 / §6.2 / §6.3 / §8.3；AGENTS.md §0 + §6。
//
// 算法实现：从零自实现 secp256k1 椭圆曲线 + ECDSA 签名 / 验证算法，仅依赖
// Go 标准库 math/big 和复用同子树的 sha256hash.Sum256。**不引入** crypto/elliptic、
// crypto/ecdsa 或任何第三方 secp256k1 库。包含：
//
//   · secp256k1 参数（p / a=0 / b=7 / Gx / Gy / n）—— 比特币 / 以太坊曲线
//   · 仿射坐标点加（P + Q）+ 点倍（2P），含恒等元 O 与对称点处理
//   · 二进制展开标量乘法（double-and-add，从最高位开始）
//   · 模逆 secpP / secpN（费马小定理 ModInverse）
//   · ECDSA 签名：z = SHA-256(m) → 选 k → R = k·G → r = R.x mod n；
//                  s = k^(-1) · (z + r·d) mod n
//   · ECDSA 验证：u1 = z·s^(-1); u2 = r·s^(-1); P = u1·G + u2·Q；
//                  r ?= P.x mod n
//
// 教学决策：
//   - 流水线 P4：keygen → message → hash → sign → verify
//   - 公钥 / 签名 (r, s) / 验证中间值（u1, u2, P.x mod n）全部展示

package ecdsasign

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	fw "github.com/lenschain/sim-engine/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
)

// =====================================================================
// 元信息
// =====================================================================

const (
	sceneCode     = "ecdsa-sign"
	schemaVersion = "v1.0.0"
	algorithmType = "ecdsa-secp256k1"

	linkGroupCryptoVerify = "crypto-verify-group"
	linkOwnerSubtree      = "signatures.ecdsa"

	defaultMessage    = "hello LensChain"
	defaultPrivateKey = "1" // 教学默认 d=1 → Q=G（直观）
	defaultNonceK     = "2" // 教学默认 k=2，避免实际签名变成 r=0
)

var pipelineNodeIDs = []string{
	"phase-keygen", "phase-message", "phase-hash", "phase-sign", "phase-verify",
}
var phaseLabels = []string{"密钥派生", "消息输入", "SHA-256 → z", "签名 (r, s)"}

// =====================================================================
// secp256k1 参数（256-bit；比特币 / 以太坊 / Solana 通用曲线）
// =====================================================================

func bigFromHex(s string) *big.Int {
	z := new(big.Int)
	z.SetString(strings.ReplaceAll(s, " ", ""), 16)
	return z
}

var (
	secpP = bigFromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F")
	secpA = big.NewInt(0)
	secpB = big.NewInt(7)
	secpN = bigFromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")
	secpG = ecPoint{
		X:        bigFromHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798"),
		Y:        bigFromHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8"),
		Infinity: false,
	}
)

// =====================================================================
// 椭圆曲线点 + 仿射运算（自实现）
// =====================================================================

// ecPoint 仿射坐标点；Infinity=true 表示无穷远点（恒等元 O）。
type ecPoint struct {
	X, Y     *big.Int
	Infinity bool
}

// pointInfinity 返回恒等元 O。
func pointInfinity() ecPoint { return ecPoint{Infinity: true} }

// mod 把 z 映射到 [0, m)。
func mod(z, m *big.Int) *big.Int {
	r := new(big.Int).Mod(z, m)
	if r.Sign() < 0 {
		r.Add(r, m)
	}
	return r
}

// modInv 计算 z 关于 m 的模逆（费马小定理：m 必须是素数）。
func modInv(z, m *big.Int) *big.Int {
	return new(big.Int).ModInverse(mod(z, m), m)
}

// pointEqual 两点是否相等。
func pointEqual(p, q ecPoint) bool {
	if p.Infinity || q.Infinity {
		return p.Infinity && q.Infinity
	}
	return p.X.Cmp(q.X) == 0 && p.Y.Cmp(q.Y) == 0
}

// pointNeg 返回 -P（仿射对称点：-y mod p）。
func pointNeg(p ecPoint) ecPoint {
	if p.Infinity {
		return p
	}
	return ecPoint{X: new(big.Int).Set(p.X), Y: mod(new(big.Int).Neg(p.Y), secpP)}
}

// pointAdd 计算 P + Q（仿射坐标，secp256k1）。
func pointAdd(p, q ecPoint) ecPoint {
	if p.Infinity {
		return q
	}
	if q.Infinity {
		return p
	}
	if p.X.Cmp(q.X) == 0 {
		// P.y = -Q.y → P + Q = O
		sum := new(big.Int).Add(p.Y, q.Y)
		if mod(sum, secpP).Sign() == 0 {
			return pointInfinity()
		}
		// P == Q → 调用倍加
		return pointDouble(p)
	}
	// λ = (q.y - p.y) / (q.x - p.x) mod p
	num := new(big.Int).Sub(q.Y, p.Y)
	den := new(big.Int).Sub(q.X, p.X)
	lambda := new(big.Int).Mul(num, modInv(den, secpP))
	lambda = mod(lambda, secpP)
	// x_r = λ² - p.x - q.x
	xr := new(big.Int).Mul(lambda, lambda)
	xr.Sub(xr, p.X)
	xr.Sub(xr, q.X)
	xr = mod(xr, secpP)
	// y_r = λ(p.x - x_r) - p.y
	yr := new(big.Int).Sub(p.X, xr)
	yr.Mul(yr, lambda)
	yr.Sub(yr, p.Y)
	yr = mod(yr, secpP)
	return ecPoint{X: xr, Y: yr}
}

// pointDouble 计算 2P（secp256k1，a=0 简化：λ = 3x²/(2y) mod p）。
func pointDouble(p ecPoint) ecPoint {
	if p.Infinity || p.Y.Sign() == 0 {
		return pointInfinity()
	}
	// λ = (3x² + a) / (2y) = 3x² / (2y) since a=0
	num := new(big.Int).Mul(p.X, p.X)
	num.Mul(num, big.NewInt(3))
	den := new(big.Int).Mul(p.Y, big.NewInt(2))
	lambda := new(big.Int).Mul(num, modInv(den, secpP))
	lambda = mod(lambda, secpP)
	// x_r = λ² - 2x
	xr := new(big.Int).Mul(lambda, lambda)
	xr.Sub(xr, new(big.Int).Mul(p.X, big.NewInt(2)))
	xr = mod(xr, secpP)
	// y_r = λ(x - x_r) - y
	yr := new(big.Int).Sub(p.X, xr)
	yr.Mul(yr, lambda)
	yr.Sub(yr, p.Y)
	yr = mod(yr, secpP)
	return ecPoint{X: xr, Y: yr}
}

// scalarMul 计算 k·P（double-and-add，从最高位开始）。
func scalarMul(k *big.Int, p ecPoint) ecPoint {
	r := pointInfinity()
	kk := mod(k, secpN)
	if kk.Sign() == 0 {
		return r
	}
	for i := kk.BitLen() - 1; i >= 0; i-- {
		r = pointDouble(r)
		if kk.Bit(i) == 1 {
			r = pointAdd(r, p)
		}
	}
	return r
}

// onCurve 验证点是否在 secp256k1 上（y² ≡ x³ + 7 mod p）。
func onCurve(p ecPoint) bool {
	if p.Infinity {
		return true
	}
	lhs := mod(new(big.Int).Mul(p.Y, p.Y), secpP)
	rhs := new(big.Int).Mul(p.X, p.X)
	rhs.Mul(rhs, p.X)
	rhs.Add(rhs, secpB)
	rhs = mod(rhs, secpP)
	return lhs.Cmp(rhs) == 0
}

// =====================================================================
// ECDSA 签名 / 验证
// =====================================================================

// ecdsaSign 给定私钥 d、消息哈希 z、临时密钥 k，返回 (r, s)。
func ecdsaSign(d, z, k *big.Int) (r, s *big.Int, err error) {
	dd := mod(d, secpN)
	if dd.Sign() == 0 {
		return nil, nil, errors.New("私钥 d=0 非法")
	}
	kk := mod(k, secpN)
	if kk.Sign() == 0 {
		return nil, nil, errors.New("nonce k=0 非法")
	}
	R := scalarMul(kk, secpG)
	if R.Infinity {
		return nil, nil, errors.New("k·G = O，请换 k")
	}
	r = mod(R.X, secpN)
	if r.Sign() == 0 {
		return nil, nil, errors.New("r=0，请换 k")
	}
	// s = k^(-1) (z + r d) mod n
	rd := new(big.Int).Mul(r, dd)
	zrd := new(big.Int).Add(mod(z, secpN), rd)
	s = new(big.Int).Mul(modInv(kk, secpN), zrd)
	s = mod(s, secpN)
	if s.Sign() == 0 {
		return nil, nil, errors.New("s=0，请换 k")
	}
	return r, s, nil
}

// ecdsaVerify 给定公钥 Q、消息哈希 z、签名 (r, s)，返回是否有效 + 中间值。
type verifyTrace struct {
	U1      *big.Int
	U2      *big.Int
	Px      *big.Int
	PxModN  *big.Int
	OnCurve bool
}

func ecdsaVerify(Q ecPoint, z, r, s *big.Int) (bool, verifyTrace) {
	tr := verifyTrace{}
	if Q.Infinity || !onCurve(Q) {
		tr.OnCurve = false
		return false, tr
	}
	tr.OnCurve = true
	rr := mod(r, secpN)
	ss := mod(s, secpN)
	if rr.Sign() == 0 || ss.Sign() == 0 || rr.Cmp(secpN) >= 0 || ss.Cmp(secpN) >= 0 {
		return false, tr
	}
	w := modInv(ss, secpN)
	tr.U1 = mod(new(big.Int).Mul(mod(z, secpN), w), secpN)
	tr.U2 = mod(new(big.Int).Mul(rr, w), secpN)
	P := pointAdd(scalarMul(tr.U1, secpG), scalarMul(tr.U2, Q))
	if P.Infinity {
		return false, tr
	}
	tr.Px = new(big.Int).Set(P.X)
	tr.PxModN = mod(P.X, secpN)
	return tr.PxModN.Cmp(rr) == 0, tr
}

// hashMessage 计算 SHA-256(m) → z，作为大整数（big-endian 解释）。
func hashMessage(m string) *big.Int {
	h := sha256hash.Sum256([]byte(m))
	return new(big.Int).SetBytes(h[:])
}

// =====================================================================
// 场景内部状态
// =====================================================================

type snapState struct {
	PrivateKey  string // 大整数 dec / hex 字符串
	NonceK      string
	Message     string
	PublicKeyX  string // 大写 hex
	PublicKeyY  string
	MessageHash string // z 的 hex
	SignatureR  string
	SignatureS  string
	Verified    bool
	TamperedR   bool
	WrongPubKey bool
	LastError   string
}

func loadState(s *fw.SceneState) snapState {
	if s == nil || s.Data == nil {
		return snapState{
			PrivateKey: defaultPrivateKey,
			NonceK:     defaultNonceK,
			Message:    defaultMessage,
		}
	}
	d := s.Data
	return snapState{
		PrivateKey:  fw.MapStr(d, "private_key", defaultPrivateKey),
		NonceK:      fw.MapStr(d, "nonce_k", defaultNonceK),
		Message:     fw.MapStr(d, "message", defaultMessage),
		PublicKeyX:  fw.MapStr(d, "public_key_x", ""),
		PublicKeyY:  fw.MapStr(d, "public_key_y", ""),
		MessageHash: fw.MapStr(d, "message_hash", ""),
		SignatureR:  fw.MapStr(d, "signature_r", ""),
		SignatureS:  fw.MapStr(d, "signature_s", ""),
		Verified:    fw.MapBool(d, "verified", false),
		TamperedR:   fw.MapBool(d, "tampered_r", false),
		WrongPubKey: fw.MapBool(d, "wrong_pubkey", false),
		LastError:   fw.MapStr(d, "last_error", ""),
	}
}

func saveState(s *fw.SceneState, st snapState) {
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	s.Data["private_key"] = st.PrivateKey
	s.Data["nonce_k"] = st.NonceK
	s.Data["message"] = st.Message
	s.Data["public_key_x"] = st.PublicKeyX
	s.Data["public_key_y"] = st.PublicKeyY
	s.Data["message_hash"] = st.MessageHash
	s.Data["signature_r"] = st.SignatureR
	s.Data["signature_s"] = st.SignatureS
	s.Data["verified"] = st.Verified
	s.Data["tampered_r"] = st.TamperedR
	s.Data["wrong_pubkey"] = st.WrongPubKey
	s.Data["last_error"] = st.LastError
}

// parseBigInt 接受 hex（前缀 0x 可选）或十进制字符串。
func parseBigInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	z := new(big.Int)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		_, ok := z.SetString(s[2:], 16)
		if !ok {
			return nil, fmt.Errorf("无法解析 hex: %s", s)
		}
		return z, nil
	}
	if _, ok := z.SetString(s, 10); !ok {
		// 尝试 hex（不带前缀）
		if _, ok2 := z.SetString(s, 16); ok2 {
			return z, nil
		}
		return nil, fmt.Errorf("无法解析数值: %s", s)
	}
	return z, nil
}

// regenerate 根据 PrivateKey / NonceK / Message 重新执行一遍 keygen + sign + verify。
func (st *snapState) regenerate() {
	st.LastError = ""
	d, err := parseBigInt(st.PrivateKey)
	if err != nil {
		st.LastError = "私钥解析失败: " + err.Error()
		return
	}
	dd := mod(d, secpN)
	if dd.Sign() == 0 {
		st.LastError = "私钥不能为 0"
		return
	}
	Q := scalarMul(dd, secpG)
	if Q.Infinity {
		st.LastError = "公钥退化为无穷远点"
		return
	}
	if !st.WrongPubKey {
		st.PublicKeyX = fmt.Sprintf("%064x", Q.X)
		st.PublicKeyY = fmt.Sprintf("%064x", Q.Y)
	}
	z := hashMessage(st.Message)
	st.MessageHash = fmt.Sprintf("%064x", z)
	k, err := parseBigInt(st.NonceK)
	if err != nil {
		st.LastError = "nonce 解析失败: " + err.Error()
		return
	}
	r, s, err := ecdsaSign(dd, z, k)
	if err != nil {
		st.LastError = err.Error()
		return
	}
	st.SignatureR = fmt.Sprintf("%064x", r)
	st.SignatureS = fmt.Sprintf("%064x", s)
	if st.TamperedR {
		// 篡改 r：+1 mod n
		rTamp := mod(new(big.Int).Add(r, big.NewInt(1)), secpN)
		st.SignatureR = fmt.Sprintf("%064x", rTamp)
		r = rTamp
	}
	verifyQ := Q
	if st.WrongPubKey {
		// 用公钥 Q' = (d+1)·G 验证（错误公钥）
		verifyQ = scalarMul(mod(new(big.Int).Add(dd, big.NewInt(1)), secpN), secpG)
	}
	valid, _ := ecdsaVerify(verifyQ, z, r, s)
	st.Verified = valid
}

// =====================================================================
// 场景定义
// =====================================================================

func Definition() fw.Definition {
	return fw.Definition{
		Code:                sceneCode,
		Name:                "ECDSA 签名（secp256k1）",
		Description:         "演示 secp256k1 椭圆曲线上的密钥对生成、签名、验证、篡改检测",
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
			"signatures.ecdsa.public_key_x",
			"signatures.ecdsa.public_key_y",
			"signatures.ecdsa.signature_r",
			"signatures.ecdsa.signature_s",
			"signatures.ecdsa.message_hash",
			"signatures.ecdsa.verified",
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
			"private_key": defaultPrivateKey,
			"nonce_k":     defaultNonceK,
			"message":     defaultMessage,
		},
	}
}

func interactionDefinition() fw.InteractionDefinition {
	return fw.InteractionDefinition{
		SceneCode:     sceneCode,
		SchemaVersion: schemaVersion,
		Actions: []fw.ActionDef{
			{
				ActionCode: "set_private_key", Label: "设置私钥",
				Description:   "设置 256-bit 私钥 d（十进制或 0x 前缀 hex），自动派生公钥 Q=dG",
				Category:      fw.ActionParamTune, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "private_key", Type: fw.FieldString, Label: "私钥 d", Required: true, Default: defaultPrivateKey},
				},
				WritesOwnedFields: []string{"signatures.ecdsa.public_key_x", "signatures.ecdsa.public_key_y"},
				LinkOwnerFields:   []string{"signatures.ecdsa.public_key_x", "signatures.ecdsa.public_key_y"},
			},
			{
				ActionCode: "sign_message", Label: "签名消息",
				Description:   "z = SHA-256(message)；选 nonce k；输出 (r, s)",
				Category:      fw.ActionPrimary, Trigger: fw.TriggerSubmit,
				Roles:         []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType: fw.InterveneState,
				Fields: []fw.FieldDef{
					{Name: "message", Type: fw.FieldString, Label: "消息", Required: true, Default: defaultMessage},
					{Name: "nonce_k", Type: fw.FieldString, Label: "nonce k", Required: true, Default: defaultNonceK},
				},
				WritesOwnedFields: []string{
					"signatures.ecdsa.signature_r",
					"signatures.ecdsa.signature_s",
					"signatures.ecdsa.message_hash",
				},
				LinkOwnerFields: []string{
					"signatures.ecdsa.signature_r",
					"signatures.ecdsa.signature_s",
					"signatures.ecdsa.message_hash",
				},
			},
			{
				ActionCode: "verify", Label: "验证签名",
				Description:     "用公钥 Q 与 (r, s) 验证 message",
				Category:        fw.ActionObserve, Trigger: fw.TriggerImmediate,
				Roles:           []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:   fw.InterveneHint,
				LinkOwnerFields: []string{"signatures.ecdsa.verified"},
			},
			{
				ActionCode: "tamper_signature", Label: "篡改签名",
				Description: "对 r 加 1 mod n，演示验证失败",
				Category:    fw.ActionAttackInject, Trigger: fw.TriggerImmediate,
				Roles:              []fw.UserRole{fw.RoleStudent, fw.RoleTeacher},
				InterveneType:     fw.InterveneAttack,
				WritesOwnedFields: []string{"signatures.ecdsa.signature_r"},
				LinkOwnerFields:   []string{"signatures.ecdsa.signature_r"},
			},
			{
				ActionCode: "wrong_public_key", Label: "用错误公钥验证",
				Description:   "用 Q' = (d+1)·G 验证，演示密钥不匹配",
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
	st.regenerate()
	saveState(state, st)
	state.Phase = "signed"
	env := buildEnvelope(st, "init", "ECDSA 初始化（默认 d=1, k=2）", true)
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
	case "set_private_key":
		pk := fw.MapStr(in.Params, "private_key", defaultPrivateKey)
		st.PrivateKey = pk
		st.WrongPubKey = false
		st.TamperedR = false
		st.regenerate()
		if st.LastError != "" {
			return fw.ActionOutput{Success: false, ErrorMessage: st.LastError}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "set_private_key", "派生新公钥 Q = d·G", true)
		appendKeygenMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "sign_message":
		st.Message = fw.MapStr(in.Params, "message", defaultMessage)
		st.NonceK = fw.MapStr(in.Params, "nonce_k", defaultNonceK)
		st.TamperedR = false
		st.WrongPubKey = false
		st.regenerate()
		if st.LastError != "" {
			return fw.ActionOutput{Success: false, ErrorMessage: st.LastError}, nil
		}
		saveState(state, st)
		out.Render = buildEnvelope(st, "sign_message", "签名生成 (r, s)", false)
		appendSignMicroSteps(&out.Render)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "verify":
		st.regenerate()
		saveState(state, st)
		summary := "签名有效 ✓"
		if !st.Verified {
			summary = "签名无效 ✗"
		}
		out.Render = buildEnvelope(st, "verify", summary, false)
		appendVerifyMicroSteps(&out.Render, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "tamper_signature":
		st.TamperedR = true
		st.regenerate()
		saveState(state, st)
		out.Render = buildEnvelope(st, "tamper_signature", "r 已被篡改 → 验证应失败", false)
		appendVerifyMicroSteps(&out.Render, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
		return out, nil

	case "wrong_public_key":
		st.WrongPubKey = true
		st.regenerate()
		saveState(state, st)
		out.Render = buildEnvelope(st, "wrong_public_key", "使用错误公钥 Q'=(d+1)·G → 验证失败", false)
		appendVerifyMicroSteps(&out.Render, st.Verified)
		out.SharedStateDiff = ownerDiff(st)
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
		st = snapState{PrivateKey: defaultPrivateKey, NonceK: defaultNonceK, Message: defaultMessage}
		st.regenerate()
		saveState(state, st)
		out.Render = buildEnvelope(st, "reset", "已重置", true)
		return out, nil
	}

	return fw.ActionOutput{Success: false, ErrorMessage: "未知 ActionCode: " + in.ActionCode}, errors.New("unknown action")
}

// =====================================================================
// RenderEnvelope 构造
// =====================================================================

func activePhase(st snapState) int {
	switch {
	case st.PublicKeyX == "":
		return 0
	case st.MessageHash == "":
		return 2
	case st.SignatureR == "":
		return 3
	default:
		return 4
	}
}

func buildEnvelope(st snapState, reason, summary string, fullSnapshot bool) fw.RenderEnvelope {
	active := activePhase(st)
	prims := make([]fw.Primitive, 0, 30)

	// 1) 流水线（5 节点）
	prims = append(prims, fw.PrimStack("pipeline", pipelineNodeIDs, "horizontal"))
	for i, id := range pipelineNodeIDs {
		status := "normal"
		if i == active {
			status = "active"
		}
		role := []string{"keygen", "message", "hash", "sign", "verify"}[i]
		label := []string{"Q = d·G", "消息 m", "z = SHA-256(m)", "(r, s)", "Verify"}[i]
		prims = append(prims, fw.PrimNode(id, label, status, role))
	}
	for i := 0; i < len(pipelineNodeIDs)-1; i++ {
		anim := ""
		if i == active-1 {
			anim = "flow"
		}
		prims = append(prims, fw.PrimEdge(fmt.Sprintf("edge-%d-%d", i, i+1), pipelineNodeIDs[i], pipelineNodeIDs[i+1], "solid", anim))
	}

	// 2) 4 阶段进度
	prog := 0.25 * float64(active)
	if active >= 4 {
		prog = 1.0
	}
	prims = append(prims, fw.PrimPhaseProgress("phase-progress", phaseLabels, minInt(active, len(phaseLabels)-1), prog))

	// 3) 验证状态指示（pulse + glow）
	verifyColor := "info"
	if st.SignatureR != "" {
		if st.Verified {
			verifyColor = "success"
		} else {
			verifyColor = "danger"
		}
	}
	prims = append(prims, fw.PrimGlow("glow-verify", pipelineNodeIDs[4], verifyColor, 0.9))

	// 4) 公式
	prims = append(prims, fw.PrimMathFormula("formula-keygen", `Q = d \cdot G`, false))
	prims = append(prims, fw.PrimMathFormula("formula-sign",
		`r = (k \cdot G)_x \bmod n,\ \ s = k^{-1}(z + r \cdot d) \bmod n`, false))
	prims = append(prims, fw.PrimMathFormula("formula-verify",
		`u_1 = z s^{-1},\ u_2 = r s^{-1};\ \ (u_1 G + u_2 Q)_x \stackrel{?}{=} r \pmod n`, false))

	// 5) 曲线常量 + 私钥 / 公钥 / 签名 / 验证 6 个 code_block
	prims = append(prims, fw.PrimCodeBlock("cb-curve",
		fmt.Sprintf("曲线: secp256k1\np = %s\nn = %s\nG.x = %s\nG.y = %s",
			fmt.Sprintf("0x%064x", secpP), fmt.Sprintf("0x%064x", secpN),
			fmt.Sprintf("0x%064x", secpG.X), fmt.Sprintf("0x%064x", secpG.Y)),
		"text", nil, 6))
	prims = append(prims, fw.PrimCodeBlock("cb-private-key",
		fmt.Sprintf("私钥 d = %s", st.PrivateKey), "text", nil, 2))
	prims = append(prims, fw.PrimCodeBlock("cb-public-key",
		fmt.Sprintf("公钥 Q.x = %s\n公钥 Q.y = %s", st.PublicKeyX, st.PublicKeyY),
		"text", nil, 4))
	prims = append(prims, fw.PrimCodeBlock("cb-message",
		fmt.Sprintf("消息: %s\nSHA-256(m) = %s\nnonce k = %s", st.Message, st.MessageHash, st.NonceK),
		"text", nil, 4))
	sigLines := []string{
		fmt.Sprintf("r = %s", st.SignatureR),
		fmt.Sprintf("s = %s", st.SignatureS),
	}
	if st.TamperedR {
		sigLines = append(sigLines, "⚠ r 已被 +1 篡改")
	}
	prims = append(prims, fw.PrimCodeBlock("cb-signature", strings.Join(sigLines, "\n"), "text", nil, 4))

	verifyLines := []string{}
	if st.SignatureR != "" {
		// 用当前公钥重新验证一次得到中间 trace
		Q := ecPoint{X: bigFromHex(st.PublicKeyX), Y: bigFromHex(st.PublicKeyY)}
		z := bigFromHex(st.MessageHash)
		r := bigFromHex(st.SignatureR)
		s := bigFromHex(st.SignatureS)
		_, tr := ecdsaVerify(Q, z, r, s)
		if tr.U1 != nil {
			verifyLines = append(verifyLines,
				fmt.Sprintf("u1 = z·s⁻¹ = %s", fmt.Sprintf("%064x", tr.U1)),
				fmt.Sprintf("u2 = r·s⁻¹ = %s", fmt.Sprintf("%064x", tr.U2)),
				fmt.Sprintf("(u1·G + u2·Q).x = %s", fmt.Sprintf("%064x", tr.Px)),
				fmt.Sprintf("(...) mod n = %s", fmt.Sprintf("%064x", tr.PxModN)),
				fmt.Sprintf("r            = %s", st.SignatureR),
			)
		}
		if st.Verified {
			verifyLines = append(verifyLines, "", "✓ 等于 r → 签名有效")
		} else {
			verifyLines = append(verifyLines, "", "✗ 不等于 r → 签名无效")
		}
	} else {
		verifyLines = append(verifyLines, "（尚未签名）")
	}
	prims = append(prims, fw.PrimCodeBlock("cb-verify", strings.Join(verifyLines, "\n"), "text", nil, 8))

	// 6) 动效
	prims = append(prims, fw.PrimPulse("pulse-verify", "cb-verify", verifyColor, 1500))
	prims = append(prims, fw.PrimGlow("glow-active", pipelineNodeIDs[active], "info", 0.8))

	// 7) 联动徽章
	prims = append(prims, fw.PrimLinkIndicator("link-crypto", linkGroupCryptoVerify, "idle", ""))

	// 8) 错误指示
	if st.LastError != "" {
		prims = append(prims, fw.PrimErrorOverlay("err", "warning", "ECDSA 错误", st.LastError, "scene", "请检查输入", true))
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
		"curve":        "secp256k1",
		"private_key":  st.PrivateKey,
		"nonce_k":      st.NonceK,
		"message":      st.Message,
		"message_hash": st.MessageHash,
		"public_key_x": st.PublicKeyX,
		"public_key_y": st.PublicKeyY,
		"signature_r":  st.SignatureR,
		"signature_s":  st.SignatureS,
		"verified":     st.Verified,
		"tampered_r":   st.TamperedR,
		"wrong_pubkey": st.WrongPubKey,
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
		{ID: "kg-1", Label: "读取私钥 d", DurationMs: 400, HighlightIDs: []string{"cb-private-key"}, ParentPhase: "keygen"},
		{ID: "kg-2", Label: "标量乘法 Q = d·G（double-and-add）", DurationMs: 700, HighlightIDs: []string{"formula-keygen", "cb-curve"}, FirePrimitives: []string{"glow-active"}},
		{ID: "kg-3", Label: "派生公钥 (Qx, Qy)", DurationMs: 500, HighlightIDs: []string{"cb-public-key"}, IsLinkTrigger: true},
	}
}

func appendSignMicroSteps(env *fw.RenderEnvelope) {
	env.MicroSteps = []fw.MicroStep{
		{ID: "sn-1", Label: "z = SHA-256(message)", DurationMs: 500, HighlightIDs: []string{"cb-message", pipelineNodeIDs[2]}},
		{ID: "sn-2", Label: "R = k·G → r = R.x mod n", DurationMs: 600, HighlightIDs: []string{"formula-sign"}, FirePrimitives: []string{"glow-active"}},
		{ID: "sn-3", Label: "s = k⁻¹(z + r·d) mod n", DurationMs: 600, HighlightIDs: []string{"cb-signature"}, IsLinkTrigger: true},
	}
}

func appendVerifyMicroSteps(env *fw.RenderEnvelope, ok bool) {
	tail := "签名有效 ✓"
	if !ok {
		tail = "签名无效 ✗"
	}
	env.MicroSteps = []fw.MicroStep{
		{ID: "vf-1", Label: "u1 = z·s⁻¹, u2 = r·s⁻¹", DurationMs: 500, HighlightIDs: []string{"formula-verify", "cb-verify"}},
		{ID: "vf-2", Label: "P = u1·G + u2·Q", DurationMs: 600, HighlightIDs: []string{"cb-verify"}},
		{ID: "vf-3", Label: tail, DurationMs: 600, HighlightIDs: []string{"cb-verify", "glow-verify"}, FirePrimitives: []string{"pulse-verify"}, IsLinkTrigger: true},
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
		ID:             "ecdsa-publish-" + fmt.Sprint(env.IsFullSnapshot),
		SourceScene:    sceneCode,
		SourceAction:   "publish_signature",
		LinkGroup:      linkGroupCryptoVerify,
		ChangedFields:  []string{"signatures.ecdsa.signature_r", "signatures.ecdsa.signature_s"},
		Payload:        map[string]any{"r": st.SignatureR, "s": st.SignatureS},
		SourceAnchorID: "ecdsa-output-anchor",
		TargetAnchorID: "verifier-input-anchor",
	})
	env.ChangedKeys = append(env.ChangedKeys, "signatures.ecdsa.signature_r", "signatures.ecdsa.signature_s")
	env.Data["link_owner_subtree"] = linkOwnerSubtree
}

func ownerDiff(st snapState) map[string]any {
	return map[string]any{
		"signatures": map[string]any{
			"ecdsa": map[string]any{
				"curve":        "secp256k1",
				"public_key_x": st.PublicKeyX,
				"public_key_y": st.PublicKeyY,
				"message":      st.Message,
				"message_hash": st.MessageHash,
				"signature_r":  st.SignatureR,
				"signature_s":  st.SignatureS,
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
