package zkpbasic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造零知识证明原理场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "zkp-basic",
		Title:        "零知识证明原理",
		Phase:        "承诺",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   12,
		Stages:       []string{"承诺", "挑战", "响应", "验证通过"},
		Nodes: []framework.Node{
			{ID: "prover", Label: "Prover", Status: "active", Role: "zkp", X: 140, Y: 220},
			{ID: "commitment", Label: "Commitment", Status: "normal", Role: "zkp", X: 330, Y: 120},
			{ID: "challenge", Label: "Challenge", Status: "normal", Role: "zkp", X: 330, Y: 320},
			{ID: "verifier", Label: "Verifier", Status: "normal", Role: "zkp", X: 540, Y: 220},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化证明者秘密、随机数和验证状态。
func Init(state *framework.SceneState, input framework.InitInput) error {
	secret := framework.StringValue(input.Params["secret"], "s1")
	model := proofModel{
		Secret:         secret,
		Witness:        "knowledge-of-discrete-log",
		Randomness:     "r-12",
		Commitment:     buildCommitment(secret, "r-12"),
		Challenge:      "000",
		Response:       "pending",
		Verified:       false,
		VerifierTrust:  0.32,
		TranscriptHash: "",
		ProofSize:      len(secret) + len("r-12") + 32,
	}
	model.TranscriptHash = buildTranscript(model)
	return rebuildState(state, model, "承诺")
}

// Step 推进承诺、挑战、响应和验证流程。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "承诺"))
	switch phase {
	case "挑战":
		model.Challenge = buildChallenge(model.Commitment)
		model.VerifierTrust = 0.56
	case "响应":
		model.Response = buildResponse(model.Secret, model.Randomness, model.Challenge)
		model.VerifierTrust = 0.78
	case "验证通过":
		model.Verified = verifyTranscript(model)
		if model.Verified {
			model.VerifierTrust = 0.96
		} else {
			model.VerifierTrust = 0.18
		}
	}
	model.TranscriptHash = buildTranscript(model)
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("零知识证明进入%s阶段。", phase), toneByVerification(model.Verified, phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
	}, nil
}

// HandleAction 更换秘密值并重新生成证明 transcript。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Secret = framework.StringValue(input.Params["secret"], "s1")
	model.Randomness = fmt.Sprintf("r-%d", len(model.Secret)*7)
	model.Commitment = buildCommitment(model.Secret, model.Randomness)
	model.Challenge = "000"
	model.Response = "pending"
	model.Verified = false
	model.VerifierTrust = 0.3
	model.ProofSize = len(model.Secret) + len(model.Randomness) + 32
	model.TranscriptHash = buildTranscript(model)
	if err := rebuildState(state, model, "承诺"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "更换秘密", fmt.Sprintf("证明者切换秘密为 %s。", model.Secret), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
	}, nil
}

// BuildRenderState 输出证明 transcript、消息流和校验状态。
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

// SyncSharedState 在密码学验证组共享哈希变化后重建零知识证明场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedZKPState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// proofModel 保存证明者和验证者之间的 transcript。
type proofModel struct {
	Secret         string  `json:"secret"`
	Witness        string  `json:"witness"`
	Randomness     string  `json:"randomness"`
	Commitment     string  `json:"commitment"`
	Challenge      string  `json:"challenge"`
	Response       string  `json:"response"`
	Verified       bool    `json:"verified"`
	VerifierTrust  float64 `json:"verifier_trust"`
	TranscriptHash string  `json:"transcript_hash"`
	ProofSize      int     `json:"proof_size"`
}

// rebuildState 将 transcript 模型映射为可视化节点、消息与指标。
func rebuildState(state *framework.SceneState, model proofModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Load = 0
		state.Nodes[index].Attributes = nil
	}
	state.Nodes[0].Status = "active"
	state.Nodes[0].Load = float64(len(model.Secret))
	state.Nodes[0].Attributes = map[string]any{"witness": model.Witness}
	state.Nodes[1].Status = "warning"
	state.Nodes[1].Load = float64(len(model.Commitment))
	state.Nodes[1].Attributes = map[string]any{"commitment": model.Commitment}
	state.Nodes[2].Status = challengeStatus(phase)
	state.Nodes[2].Load = float64(len(model.Challenge))
	state.Nodes[2].Attributes = map[string]any{"challenge": model.Challenge}
	state.Nodes[3].Status = verifierStatus(model, phase)
	state.Nodes[3].Load = model.VerifierTrust * 100
	state.Nodes[3].Attributes = map[string]any{"verified": model.Verified}
	state.Messages = []framework.Message{
		{ID: "commit-message", Label: framework.Abbreviate(model.Commitment, 14), Kind: "digest", Status: phase, SourceID: "prover", TargetID: "commitment"},
		{ID: "challenge-message", Label: model.Challenge, Kind: "digest", Status: phase, SourceID: "verifier", TargetID: "challenge"},
		{ID: "response-message", Label: framework.Abbreviate(model.Response, 14), Kind: "digest", Status: phase, SourceID: "prover", TargetID: "verifier"},
	}
	state.Metrics = []framework.Metric{
		{Key: "secret", Label: "秘密标签", Value: model.Secret, Tone: "info"},
		{Key: "challenge", Label: "挑战值", Value: model.Challenge, Tone: challengeTone(phase)},
		{Key: "proof_size", Label: "证明大小", Value: fmt.Sprintf("%d", model.ProofSize), Tone: "warning"},
		{Key: "verified", Label: "验证结果", Value: framework.BoolText(model.Verified), Tone: toneByVerification(model.Verified, phase)},
		{Key: "trust", Label: "验证者信心", Value: fmt.Sprintf("%.0f%%", model.VerifierTrust*100), Tone: toneByVerification(model.Verified, phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "承诺", Value: model.Commitment},
		{Label: "响应", Value: model.Response},
		{Label: "Transcript", Value: model.TranscriptHash},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"zkp_basic":  model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟承诺、挑战、响应与验证四段式零知识证明交互。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复证明 transcript。
func decodeModel(state *framework.SceneState) proofModel {
	entry, ok := state.Data["zkp_basic"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["zkp_basic"].(proofModel); ok {
			return typed
		}
		model := proofModel{
			Secret:        "s1",
			Witness:       "knowledge-of-discrete-log",
			Randomness:    "r-12",
			Commitment:    buildCommitment("s1", "r-12"),
			Challenge:     "000",
			Response:      "pending",
			VerifierTrust: 0.32,
		}
		model.TranscriptHash = buildTranscript(model)
		return model
	}
	return proofModel{
		Secret:         framework.StringValue(entry["secret"], "s1"),
		Witness:        framework.StringValue(entry["witness"], "knowledge-of-discrete-log"),
		Randomness:     framework.StringValue(entry["randomness"], "r-12"),
		Commitment:     framework.StringValue(entry["commitment"], ""),
		Challenge:      framework.StringValue(entry["challenge"], "000"),
		Response:       framework.StringValue(entry["response"], "pending"),
		Verified:       framework.BoolValue(entry["verified"], false),
		VerifierTrust:  framework.NumberValue(entry["verifier_trust"], 0.32),
		TranscriptHash: framework.StringValue(entry["transcript_hash"], ""),
		ProofSize:      int(framework.NumberValue(entry["proof_size"], 36)),
	}
}

// applySharedZKPState 将密码学验证组共享哈希变化映射回证明 transcript。
func applySharedZKPState(model *proofModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if hashes, ok := sharedState["hashes"].(map[string]any); ok {
		if primary, ok := hashes["primary"].(string); ok && strings.TrimSpace(primary) != "" {
			model.Challenge = strings.ToUpper(primary[:minInt(3, len(primary))])
			if model.Response != "pending" {
				model.Verified = verifyTranscript(*model)
			}
			model.TranscriptHash = buildTranscript(*model)
		}
	}
}

// minInt 返回两个整数中的较小值。
func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

// buildCommitment 生成承诺值，体现“知道秘密但不暴露秘密”的语义。
func buildCommitment(secret string, randomness string) string {
	sum := sha256.Sum256([]byte(secret + "|" + randomness))
	return hex.EncodeToString(sum[:])
}

// buildChallenge 根据承诺构造验证者挑战。
func buildChallenge(commitment string) string {
	if len(commitment) < 6 {
		return "111"
	}
	return strings.ToUpper(commitment[2:5])
}

// buildResponse 生成针对挑战的响应摘要。
func buildResponse(secret string, randomness string, challenge string) string {
	sum := sha256.Sum256([]byte(secret + ":" + randomness + ":" + challenge))
	return hex.EncodeToString(sum[:])
}

// verifyTranscript 校验证明 transcript 是否自洽。
func verifyTranscript(model proofModel) bool {
	expected := buildResponse(model.Secret, model.Randomness, model.Challenge)
	return model.Commitment == buildCommitment(model.Secret, model.Randomness) && expected == model.Response
}

// buildTranscript 生成 transcript 哈希，便于可视化跟踪。
func buildTranscript(model proofModel) string {
	sum := sha256.Sum256([]byte(model.Commitment + "|" + model.Challenge + "|" + model.Response))
	return hex.EncodeToString(sum[:])
}

// nextPhase 返回零知识证明的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "承诺":
		return "挑战"
	case "挑战":
		return "响应"
	case "响应":
		return "验证通过"
	default:
		return "承诺"
	}
}

// phaseIndex 将阶段映射到时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "承诺":
		return 0
	case "挑战":
		return 1
	case "响应":
		return 2
	case "验证通过":
		return 3
	default:
		return 0
	}
}

// verifierStatus 返回验证节点的状态。
func verifierStatus(model proofModel, phase string) string {
	if phase == "验证通过" && model.Verified {
		return "success"
	}
	if phase == "响应" || phase == "挑战" {
		return "warning"
	}
	return "normal"
}

// challengeStatus 返回挑战节点状态。
func challengeStatus(phase string) string {
	if phase == "挑战" {
		return "active"
	}
	if phase == "响应" || phase == "验证通过" {
		return "warning"
	}
	return "normal"
}

// challengeTone 返回挑战指标色调。
func challengeTone(phase string) string {
	if phase == "挑战" || phase == "响应" {
		return "warning"
	}
	return "info"
}

// toneByVerification 返回验证结果对应的事件色调。
func toneByVerification(verified bool, phase string) string {
	if phase == "验证通过" {
		if verified {
			return "success"
		}
		return "warning"
	}
	if phase == "响应" {
		return "info"
	}
	return "warning"
}
