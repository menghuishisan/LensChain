package reentrancyattack

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造重入攻击场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "reentrancy-attack",
		Title:        "重入攻击",
		Phase:        "调用 withdraw",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1500,
		TotalTicks:   18,
		Stages:       []string{"调用 withdraw", "回调重入", "余额清空", "修复对比"},
		Nodes: []framework.Node{
			{ID: "vault", Label: "Vault", Status: "active", Role: "contract", X: 150, Y: 200},
			{ID: "attacker", Label: "Attacker", Status: "normal", Role: "contract", X: 340, Y: 120},
			{ID: "fallback", Label: "Fallback", Status: "normal", Role: "function", X: 520, Y: 200},
			{ID: "balance", Label: "Balance", Status: "normal", Role: "state", X: 340, Y: 340},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化存在漏洞的金库余额和调用栈。
func Init(state *framework.SceneState, input framework.InitInput) error {
	model := attackModel{VaultBalance: 100, AttackerBalance: 0, UserCredit: 40, MaxDepth: 3, Protected: false}
	applySharedContractState(&model, input.SharedState)
	return rebuildState(state, model, "调用 withdraw")
}

// Step 推进重入调用、余额转移和修复对比过程。
func Step(state *framework.SceneState, input framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	applySharedContractState(&model, input.SharedState)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "调用 withdraw"))
	switch phase {
	case "调用 withdraw":
		model.CallStack = []string{"Vault.withdraw(attacker)"}
	case "回调重入":
		if !model.Protected && model.Depth < model.MaxDepth && model.VaultBalance >= model.UserCredit {
			model.Depth++
			model.CallStack = append(model.CallStack, fmt.Sprintf("Attacker.fallback #%d", model.Depth))
			model.VaultBalance -= model.UserCredit
			model.AttackerBalance += model.UserCredit
		}
	case "余额清空":
		for !model.Protected && model.Depth < model.MaxDepth && model.VaultBalance >= model.UserCredit {
			model.Depth++
			model.CallStack = append(model.CallStack, fmt.Sprintf("Attacker.fallback #%d", model.Depth))
			model.VaultBalance -= model.UserCredit
			model.AttackerBalance += model.UserCredit
		}
	case "修复对比":
		model.Protected = true
		model.FixedTrace = []string{"Checks", "Effects", "Interactions"}
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("攻击流程进入%s阶段。", phase), eventTone(model))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": state.Data["contract_state"],
			"call_stack":     state.Data["call_stack"],
			"balances":       state.Data["balances"],
		},
	}, nil
}

// HandleAction 触发指定深度的重入攻击。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.MaxDepth = int(framework.NumberValue(input.Params["depth"], 3))
	model.Protected = false
	model.Depth = 0
	model.CallStack = []string{"Vault.withdraw(attacker)"}
	for model.Depth < model.MaxDepth && model.VaultBalance >= model.UserCredit {
		model.Depth++
		model.CallStack = append(model.CallStack, fmt.Sprintf("Attacker.fallback #%d", model.Depth))
		model.VaultBalance -= model.UserCredit
		model.AttackerBalance += model.UserCredit
	}
	if err := rebuildState(state, model, "余额清空"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "触发重入", fmt.Sprintf("攻击递归深度达到 %d。", model.Depth), "warning")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": state.Data["contract_state"],
			"call_stack":     state.Data["call_stack"],
			"balances":       state.Data["balances"],
		},
	}, nil
}

// BuildRenderState 输出攻击调用栈、余额变化和修复对比状态。
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

// SyncSharedState 在共享调用栈、余额和合约状态变化后重建重入攻击场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedContractState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// attackModel 保存重入攻击中的合约余额、攻击深度和调用栈。
type attackModel struct {
	VaultBalance    int      `json:"vault_balance"`
	AttackerBalance int      `json:"attacker_balance"`
	UserCredit      int      `json:"user_credit"`
	Depth           int      `json:"depth"`
	MaxDepth        int      `json:"max_depth"`
	Protected       bool     `json:"protected"`
	CallStack       []string `json:"call_stack"`
	FixedTrace      []string `json:"fixed_trace"`
}

// rebuildState 将攻击模型转换成可视化节点、调用消息和指标。
func rebuildState(state *framework.SceneState, model attackModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
		state.Nodes[index].Attributes = map[string]any{}
	}
	setNodeStatus(state, "vault", vaultStatus(model))
	setNodeStatus(state, "attacker", "attack")
	setNodeStatus(state, "fallback", fallbackStatus(model))
	setNodeStatus(state, "balance", balanceStatus(model))
	state.Messages = buildMessages(model, phase)
	state.Metrics = []framework.Metric{
		{Key: "vault_balance", Label: "金库余额", Value: fmt.Sprintf("%d", model.VaultBalance), Tone: balanceTone(model.VaultBalance)},
		{Key: "attacker_balance", Label: "攻击者余额", Value: fmt.Sprintf("%d", model.AttackerBalance), Tone: "warning"},
		{Key: "depth", Label: "重入深度", Value: fmt.Sprintf("%d / %d", model.Depth, model.MaxDepth), Tone: "warning"},
		{Key: "protected", Label: "防护状态", Value: protectedText(model.Protected), Tone: protectedTone(model.Protected)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "阶段", Value: phase},
		{Label: "修复策略", Value: strings.Join(model.FixedTrace, " -> ")},
	}
	state.Data = map[string]any{
		"phase_name": phase,
		"contract_state": map[string]any{
			"status":      contractStatus(model),
			"last_event":  lastContractEvent(model),
			"protected":   model.Protected,
			"user_credit": model.UserCredit,
			"storage": map[string]any{
				"vault_balance":    model.VaultBalance,
				"attacker_balance": model.AttackerBalance,
				"reentry_depth":    model.Depth,
			},
		},
		"call_stack": map[string]any{
			"pc":        model.Depth,
			"opcode":    callStackOpcode(model),
			"stack":     callStackDepthValues(model),
			"frames":    model.CallStack,
			"depth":     model.Depth,
			"reenter":   model.Depth > 0,
			"protected": model.Protected,
		},
		"balances": map[string]any{
			"vault":    model.VaultBalance,
			"attacker": model.AttackerBalance,
		},
		"attack_model": model,
	}
	state.Extra = map[string]any{
		"description": "该场景实现 withdraw 外部调用前未更新余额导致的递归重入，以及 Checks-Effects-Interactions 修复对比。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复攻击模型。
func decodeModel(state *framework.SceneState) attackModel {
	entry, ok := state.Data["attack_model"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["attack_model"].(attackModel); ok {
			return typed
		}
		return attackModel{VaultBalance: 100, AttackerBalance: 0, UserCredit: 40, MaxDepth: 3}
	}
	return attackModel{
		VaultBalance:    int(framework.NumberValue(entry["vault_balance"], 100)),
		AttackerBalance: int(framework.NumberValue(entry["attacker_balance"], 0)),
		UserCredit:      int(framework.NumberValue(entry["user_credit"], 40)),
		Depth:           int(framework.NumberValue(entry["depth"], 0)),
		MaxDepth:        int(framework.NumberValue(entry["max_depth"], 3)),
		Protected:       framework.BoolValue(entry["protected"], false),
		CallStack:       framework.ToStringSlice(entry["call_stack"]),
		FixedTrace:      framework.ToStringSlice(entry["fixed_trace"]),
	}
}

// applySharedContractState 将合约安全组中的共享状态映射回重入攻击模型。
func applySharedContractState(model *attackModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		model.Protected = framework.BoolValue(contractState["protected"], model.Protected)
		model.UserCredit = int(framework.NumberValue(contractState["user_credit"], float64(model.UserCredit)))
		if status := framework.StringValue(contractState["status"], ""); status == "closed" {
			model.Protected = true
		}
	}
	if balances, ok := sharedState["balances"].(map[string]any); ok {
		model.VaultBalance = int(framework.NumberValue(balances["vault"], float64(model.VaultBalance)))
		model.AttackerBalance = int(framework.NumberValue(balances["attacker"], float64(model.AttackerBalance)))
	}
	if callStack, ok := sharedState["call_stack"].(map[string]any); ok {
		if stack := framework.ToStringSlice(callStack["frames"]); len(stack) > 0 {
			model.CallStack = stack
			model.Depth = len(stack) - 1
		} else if stack := framework.ToStringSlice(callStack["stack"]); len(stack) > 0 {
			model.CallStack = stack
			model.Depth = len(stack)
		}
	}
}

// buildMessages 根据调用栈生成合约调用流。
func buildMessages(model attackModel, phase string) []framework.Message {
	messages := []framework.Message{
		{ID: "withdraw-call", Label: "withdraw()", Kind: "attack", Status: phase, SourceID: "attacker", TargetID: "vault"},
	}
	for index := 0; index < model.Depth; index++ {
		messages = append(messages, framework.Message{
			ID:       fmt.Sprintf("fallback-%d", index+1),
			Label:    "fallback reenter",
			Kind:     "attack",
			Status:   phase,
			SourceID: "vault",
			TargetID: "fallback",
		})
	}
	if model.Protected {
		messages = append(messages, framework.Message{ID: "fix", Label: "CEI Guard", Kind: "attack", Status: phase, SourceID: "vault", TargetID: "balance"})
	}
	return messages
}

// setNodeStatus 更新指定节点的展示状态。
func setNodeStatus(state *framework.SceneState, nodeID string, status string) {
	for index := range state.Nodes {
		if state.Nodes[index].ID == nodeID {
			state.Nodes[index].Status = status
		}
	}
}

// nextPhase 返回重入攻击流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "调用 withdraw":
		return "回调重入"
	case "回调重入":
		return "余额清空"
	case "余额清空":
		return "修复对比"
	default:
		return "调用 withdraw"
	}
}

// phaseIndex 将阶段名称映射为前端时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "调用 withdraw":
		return 0
	case "回调重入":
		return 1
	case "余额清空":
		return 2
	case "修复对比":
		return 3
	default:
		return 0
	}
}

// vaultStatus 返回金库节点状态。
func vaultStatus(model attackModel) string {
	if model.Protected {
		return "protected"
	}
	if model.VaultBalance <= 0 {
		return "drained"
	}
	return "warning"
}

// fallbackStatus 返回 fallback 节点状态。
func fallbackStatus(model attackModel) string {
	if model.Depth > 0 {
		return "active"
	}
	return "normal"
}

// balanceStatus 返回余额节点状态。
func balanceStatus(model attackModel) string {
	if model.VaultBalance <= 0 {
		return "fault"
	}
	return "warning"
}

// balanceTone 根据金库余额选择指标色调。
func balanceTone(balance int) string {
	if balance <= 0 {
		return "warning"
	}
	return "info"
}

// protectedText 返回防护状态文案。
func protectedText(flag bool) string {
	if flag {
		return "已启用"
	}
	return "未启用"
}

// protectedTone 返回防护状态指标色调。
func protectedTone(flag bool) string {
	if flag {
		return "success"
	}
	return "warning"
}

// eventTone 根据攻击模型选择事件色调。
func eventTone(model attackModel) string {
	if model.Protected {
		return "success"
	}
	return "warning"
}

// contractStatus 返回攻击对合约状态机联动面板可见的状态。
func contractStatus(model attackModel) string {
	if model.Protected {
		return "paused"
	}
	if model.Depth > 0 {
		return "active"
	}
	return "created"
}

// lastContractEvent 返回当前攻击阶段对状态机可见的最近事件。
func lastContractEvent(model attackModel) string {
	if model.Protected {
		return "pause"
	}
	if model.Depth > 0 {
		return "activate"
	}
	return "deploy"
}

// callStackOpcode 返回当前调用栈在执行面板中的主操作码标签。
func callStackOpcode(model attackModel) string {
	if model.Protected {
		return "SSTORE"
	}
	if model.Depth > 0 {
		return "CALL"
	}
	return "SLOAD"
}

// callStackDepthValues 将调用深度转换为执行栈可直接展示的整数切片。
func callStackDepthValues(model attackModel) []int {
	values := make([]int, 0, len(model.CallStack))
	for index := range model.CallStack {
		values = append(values, index+1)
	}
	return values
}
