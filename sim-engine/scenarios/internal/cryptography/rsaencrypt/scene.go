package rsaencrypt

import (
	"fmt"
	"math"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造 RSA 加密解密场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "rsa-encrypt",
		Title:        "RSA 加密解密",
		Phase:        "选择质数",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1000,
		TotalTicks:   10,
		Stages:       []string{"选择质数", "计算模数", "加密", "解密"},
		Nodes: []framework.Node{
			{ID: "prime-p", Label: "Prime-p", Status: "active", Role: "rsa", X: 120, Y: 120},
			{ID: "prime-q", Label: "Prime-q", Status: "normal", Role: "rsa", X: 120, Y: 280},
			{ID: "cipher", Label: "Cipher", Status: "normal", Role: "rsa", X: 340, Y: 120},
			{ID: "plaintext", Label: "Plaintext", Status: "normal", Role: "rsa", X: 340, Y: 280},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化 RSA 参数、明文和密文。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := rsaModel{P: 11, Q: 13, E: 7, Plaintext: 42}
	populateDerived(&model)
	return rebuildState(state, model, "选择质数")
}

// Step 推进质数选择、模数计算、加密和解密。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "选择质数"))
	populateDerived(&model)
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("RSA 流程进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"keys": map[string]any{
				"public_key": fmt.Sprintf("n=%d,e=%d", model.N, model.E),
			},
			"hashes": map[string]any{
				"primary":   fmt.Sprintf("%d", model.Plaintext),
				"signature": fmt.Sprintf("%d", model.Ciphertext),
				"verified":  model.Decrypted == model.Plaintext,
			},
		},
	}, nil
}

// HandleAction 使用新的明文重新执行加密。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Plaintext = int(framework.NumberValue(input.Params["plaintext"], 42))
	populateDerived(&model)
	if err := rebuildState(state, model, "加密"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "重新加密", fmt.Sprintf("已对明文 %d 重新加密。", model.Plaintext), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"hashes": map[string]any{
				"primary":   fmt.Sprintf("%d", model.Plaintext),
				"signature": fmt.Sprintf("%d", model.Ciphertext),
				"verified":  model.Decrypted == model.Plaintext,
			},
		},
	}, nil
}

// BuildRenderState 输出 RSA 参数、密文和解密结果。
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

// SyncSharedState 在密码学验证组共享哈希与验证结果变化后重建 RSA 场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedRSAState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// rsaModel 保存 RSA 质数、密钥和加解密结果。
type rsaModel struct {
	P          int `json:"p"`
	Q          int `json:"q"`
	E          int `json:"e"`
	D          int `json:"d"`
	N          int `json:"n"`
	Phi        int `json:"phi"`
	Plaintext  int `json:"plaintext"`
	Ciphertext int `json:"ciphertext"`
	Decrypted  int `json:"decrypted"`
	KeySize    int `json:"key_size"`
}

// rebuildState 将 RSA 模型映射为节点、消息和指标。
func rebuildState(state *framework.SceneState, model rsaModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[nodeIndexForPhase(state.PhaseIndex)].Status = "active"
	if phase == "解密" {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "prime-modulus", Label: fmt.Sprintf("n=%d", model.N), Kind: "digest", Status: phase, SourceID: "prime-p", TargetID: "cipher"},
		{ID: "cipher-plain", Label: fmt.Sprintf("%d", model.Ciphertext), Kind: "digest", Status: phase, SourceID: "cipher", TargetID: "plaintext"},
	}
	state.Metrics = []framework.Metric{
		{Key: "p", Label: "p", Value: fmt.Sprintf("%d", model.P), Tone: "info"},
		{Key: "q", Label: "q", Value: fmt.Sprintf("%d", model.Q), Tone: "warning"},
		{Key: "keysize", Label: "密钥位数", Value: fmt.Sprintf("%d", model.KeySize), Tone: "info"},
		{Key: "cipher", Label: "密文", Value: fmt.Sprintf("%d", model.Ciphertext), Tone: "success"},
		{Key: "decrypt", Label: "解密结果", Value: fmt.Sprintf("%d", model.Decrypted), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "n", Value: fmt.Sprintf("%d", model.N)},
		{Label: "phi", Value: fmt.Sprintf("%d", model.Phi)},
		{Label: "d", Value: fmt.Sprintf("%d", model.D)},
		{Label: "密钥位数", Value: fmt.Sprintf("%d", model.KeySize)},
	}
	state.Data = map[string]any{
		"phase_name":  phase,
		"rsa_encrypt": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟 RSA 中质数选择、模数计算、加密与解密过程。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复 RSA 模型。
func decodeModel(state *framework.SceneState) rsaModel {
	entry, ok := state.Data["rsa_encrypt"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["rsa_encrypt"].(rsaModel); ok {
			return typed
		}
		model := rsaModel{P: 11, Q: 13, E: 7, Plaintext: 42}
		populateDerived(&model)
		return model
	}
	model := rsaModel{
		P:         int(framework.NumberValue(entry["p"], 11)),
		Q:         int(framework.NumberValue(entry["q"], 13)),
		E:         int(framework.NumberValue(entry["e"], 7)),
		Plaintext: int(framework.NumberValue(entry["plaintext"], 42)),
		KeySize:   int(framework.NumberValue(entry["key_size"], 16)),
	}
	populateDerived(&model)
	return model
}

// applySharedRSAState 将密码学验证组共享哈希结果映射到 RSA 场景。
func applySharedRSAState(model *rsaModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if hashes, ok := sharedState["hashes"].(map[string]any); ok {
		if primary, ok := hashes["primary"].(string); ok && primary != "" {
			model.Plaintext = len(primary)
			populateDerived(model)
		}
	}
}

// populateDerived 计算 RSA 派生参数和加解密结果。
func populateDerived(model *rsaModel) {
	model.N = model.P * model.Q
	model.Phi = (model.P - 1) * (model.Q - 1)
	model.D = modularInverse(model.E, model.Phi)
	model.Ciphertext = modExp(model.Plaintext, model.E, model.N)
	model.Decrypted = modExp(model.Ciphertext, model.D, model.N)
	if model.KeySize == 0 {
		model.KeySize = len(fmt.Sprintf("%b", model.N))
	}
}

// modularInverse 计算模反元素。
func modularInverse(e int, phi int) int {
	for candidate := 1; candidate < phi; candidate++ {
		if (e*candidate)%phi == 1 {
			return candidate
		}
	}
	return 1
}

// modExp 计算模幂。
func modExp(base int, exp int, mod int) int {
	result := 1
	current := base % mod
	power := exp
	for power > 0 {
		if power%2 == 1 {
			result = int(math.Mod(float64(result*current), float64(mod)))
		}
		current = int(math.Mod(float64(current*current), float64(mod)))
		power /= 2
	}
	return result
}

// nodeIndexForPhase 返回当前四节点图中的可用高亮索引。
func nodeIndexForPhase(phaseIndex int) int {
	if phaseIndex > 3 {
		return 3
	}
	if phaseIndex < 0 {
		return 0
	}
	return phaseIndex
}

// nextPhase 返回 RSA 流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "选择质数":
		return "计算模数"
	case "计算模数":
		return "加密"
	case "加密":
		return "解密"
	default:
		return "选择质数"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "选择质数":
		return 0
	case "计算模数":
		return 1
	case "加密":
		return 2
	case "解密":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回 RSA 阶段色调。
func toneByPhase(phase string) string {
	if phase == "解密" {
		return "success"
	}
	return "info"
}
