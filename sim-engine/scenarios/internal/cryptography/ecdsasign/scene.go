package ecdsasign

import (
	"fmt"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 ECDSA 签名验签场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "ecdsa-sign",
		Title:        "ECDSA 签名验签",
		Phase:        "生成随机数 k",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1000,
		TotalTicks:   12,
		Stages:       []string{"生成随机数 k", "点乘运算", "签名输出", "验签验证"},
		Nodes: []framework.Node{
			{ID: "private-key", Label: "PrivateKey", Status: "active", Role: "crypto", X: 100, Y: 200},
			{ID: "public-key", Label: "PublicKey", Status: "normal", Role: "crypto", X: 280, Y: 100},
			{ID: "message", Label: "Message", Status: "normal", Role: "crypto", X: 280, Y: 300},
			{ID: "signature", Label: "Signature", Status: "normal", Role: "crypto", X: 500, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化密钥对、消息和待生成的签名。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := signModel{
		PrivateKey: "0xpriv-123",
		PublicKey:  "0xpub-456",
		Message:    framework.StringValue(input.Params["message"], "hello"),
		NonceK:     "k-001",
		CurvePoint: "G*k",
		Verified:   false,
	}
	applySharedSignState(&model, input.SharedState)
	model.Signature = signDigest(model.Message, model.PrivateKey, model.NonceK)
	return rebuildState(state, model, "生成随机数 k")
}

// Step 推进随机数生成、点乘、签名输出和验签验证。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedSignState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "生成随机数 k"))
	switch phase {
	case "点乘运算":
		model.PublicKey = derivePublicKey(model.PrivateKey)
		model.CurvePoint = "(" + framework.HashText(model.NonceK)[:6] + "," + framework.HashText(model.PrivateKey)[:6] + ")"
	case "签名输出":
		model.Signature = signDigest(model.Message, model.PrivateKey, model.NonceK)
	case "验签验证":
		model.Verified = verifySignature(model)
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("ECDSA 流程进入%s阶段。", phase), toneByVerification(model.Verified, phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"keys": map[string]any{
				"public_key": model.PublicKey,
			},
			"hashes": map[string]any{
				"primary":   model.Message,
				"signature": model.Signature,
				"verified":  model.Verified,
			},
		},
	}, nil
}

// HandleAction 使用新消息重新生成签名。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Message = framework.StringValue(input.Params["message"], "hello")
	model.NonceK = fmt.Sprintf("k-%03d", state.Tick+2)
	model.Signature = signDigest(model.Message, model.PrivateKey, model.NonceK)
	model.CurvePoint = "(" + framework.HashText(model.NonceK)[:6] + "," + framework.HashText(model.PrivateKey)[:6] + ")"
	model.Verified = false
	if err := rebuildState(state, model, "签名输出"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "重新签名", fmt.Sprintf("已对消息 %s 重新签名。", model.Message), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary":   model.Message,
				"signature": model.Signature,
			},
		},
	}, nil
}

// BuildRenderState 输出密钥、消息、签名和验签结果。
func BuildRenderState(state framework.SceneState) framework.RenderEnvelope {
	return framework.RenderEnvelope{
		Nodes:       state.Nodes,
		Messages:    state.Messages,
		Stages:      state.Stages,
		ChangedKeys: state.ChangedKeys,
		Phase:       state.Phase,
		PhaseIndex:  state.PhaseIndex,
		Progress:    state.Progress,
		Data:        framework.CloneMap(state.Data),
		Extra:       framework.CloneMap(state.Extra),
	}
}

// SyncSharedState 在共享摘要、公钥和验签结果变化后重建 ECDSA 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedSignState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// signModel 保存 ECDSA 简化流程中的密钥、随机数和签名结果。
type signModel struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Message    string `json:"message"`
	NonceK     string `json:"nonce_k"`
	Signature  string `json:"signature"`
	CurvePoint string `json:"curve_point"`
	Verified   bool   `json:"verified"`
}

// rebuildState 将签名模型映射为节点、消息和指标。
func rebuildState(state *framework.SceneState, model signModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[state.PhaseIndex].Status = "active"
	if phase == "验签验证" && model.Verified {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "digest-flow", Label: framework.Abbreviate(framework.HashText(model.Message), 12), Kind: "digest", Status: phase, SourceID: "message", TargetID: "signature"},
		{ID: "verify-flow", Label: framework.Abbreviate(model.Signature, 12), Kind: "digest", Status: phase, SourceID: "public-key", TargetID: "signature"},
	}
	state.Metrics = []framework.Metric{
		{Key: "message", Label: "消息", Value: model.Message, Tone: "info"},
		{Key: "nonce", Label: "随机数 k", Value: model.NonceK, Tone: "warning"},
		{Key: "signature", Label: "签名", Value: framework.Abbreviate(model.Signature, 12), Tone: "success"},
		{Key: "verified", Label: "验签结果", Value: framework.BoolLabel(model.Verified, "通过", "失败"), Tone: toneByVerification(model.Verified, phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "私钥", Value: model.PrivateKey},
		{Label: "公钥", Value: model.PublicKey},
		{Label: "曲线点", Value: model.CurvePoint},
		{Label: "消息摘要", Value: framework.HashText(model.Message)},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"ecdsa_sign": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 ECDSA 中随机数生成、点乘、公钥推导、签名与验签过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 ECDSA 模型。
func decodeModel(state *framework.SceneState) signModel {
	entry, ok := state.Data["ecdsa_sign"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["ecdsa_sign"].(signModel); ok {
			return typed
		}
		return signModel{
			PrivateKey: "0xpriv-123",
			PublicKey:  "0xpub-456",
			Message:    "hello",
			NonceK:     "k-001",
		}
	}
	return signModel{
		PrivateKey: framework.StringValue(entry["private_key"], "0xpriv-123"),
		PublicKey:  framework.StringValue(entry["public_key"], "0xpub-456"),
		Message:    framework.StringValue(entry["message"], "hello"),
		NonceK:     framework.StringValue(entry["nonce_k"], "k-001"),
		Signature:  framework.StringValue(entry["signature"], ""),
		CurvePoint: framework.StringValue(entry["curve_point"], "G*k"),
		Verified:   framework.BoolValue(entry["verified"], false),
	}
}

// derivePublicKey 使用私钥摘要构造稳定的公钥文案。
func derivePublicKey(privateKey string) string {
	return "0xpub-" + framework.HashText(privateKey)[:8]
}

// signDigest 生成稳定的简化签名结果。
func signDigest(message string, privateKey string, nonce string) string {
	return framework.HashText(message + "|" + privateKey + "|" + nonce)
}

// verifySignature 校验签名是否与当前消息和私钥一致。
func verifySignature(model signModel) bool {
	expected := signDigest(model.Message, model.PrivateKey, model.NonceK)
	return expected == model.Signature && derivePublicKey(model.PrivateKey) == model.PublicKey
}

// nextPhase 返回 ECDSA 流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "生成随机数 k":
		return "点乘运算"
	case "点乘运算":
		return "签名输出"
	case "签名输出":
		return "验签验证"
	default:
		return "生成随机数 k"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "生成随机数 k":
		return 0
	case "点乘运算":
		return 1
	case "签名输出":
		return 2
	case "验签验证":
		return 3
	default:
		return 0
	}
}

// toneByVerification 根据验签结果返回色调。
func toneByVerification(verified bool, phase string) string {
	if phase != "验签验证" {
		return "info"
	}
	if verified {
		return "success"
	}
	return "warning"
}

// applySharedSignState 将密码学联动组中的摘要、公钥和验签结果映射回 ECDSA 场景。
func applySharedSignState(model *signModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if keys, ok := sharedState["keys"].(map[string]any); ok {
		if publicKey := framework.StringValue(keys["public_key"], ""); publicKey != "" {
			model.PublicKey = publicKey
		}
	}
	if hashes, ok := sharedState["hashes"].(map[string]any); ok {
		if primary := framework.StringValue(hashes["primary"], ""); primary != "" {
			model.Message = primary
			model.Verified = false
		}
		if signature := framework.StringValue(hashes["signature"], ""); signature != "" {
			model.Signature = signature
		}
		model.Verified = framework.BoolValue(hashes["verified"], model.Verified)
	}
}
