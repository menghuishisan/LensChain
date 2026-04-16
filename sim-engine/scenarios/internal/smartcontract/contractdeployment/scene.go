package contractdeployment

import (
	"fmt"
	"strings"

	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// DefaultState 构造合约部署流程场景的初始状态。
func DefaultState() framework.SceneState {
	return framework.SceneState{
		SceneCode:    "contract-deployment",
		Title:        "合约部署流程",
		Phase:        "字节码准备",
		PhaseIndex:   0,
		Progress:     0,
		StepDuration: 1400,
		TotalTicks:   12,
		Stages:       []string{"字节码准备", "构造函数执行", "状态初始化", "地址生成"},
		Nodes: []framework.Node{
			{ID: "bytecode", Label: "Bytecode", Status: "active", Role: "deploy", X: 110, Y: 200},
			{ID: "constructor", Label: "Constructor", Status: "normal", Role: "deploy", X: 280, Y: 100},
			{ID: "storage", Label: "Storage", Status: "normal", Role: "deploy", X: 280, Y: 300},
			{ID: "address", Label: "Address", Status: "normal", Role: "deploy", X: 500, Y: 200},
		},
		ChangedKeys: []string{"nodes", "data", "metrics"},
		Data:        map[string]any{},
		Extra:       map[string]any{},
	}
}

// Init 初始化部署字节码、构造参数和地址推导结果。
func Init(state *framework.SceneState, _ framework.InitInput) error {
	model := deploymentModel{
		Bytecode:        "6080604052",
		ConstructorArgs: []string{"owner=teacher", "supply=1000"},
		Nonce:           1,
		StorageSnapshot: map[string]string{"owner": "", "supply": "0"},
		Deployer:        "teacher-wallet",
	}
	populateDerived(&model)
	return rebuildState(state, model, "字节码准备")
}

// Step 推进字节码准备、构造函数执行、状态初始化和地址生成。
func Step(state *framework.SceneState, _ framework.StepInput) (framework.StepOutput, error) {
	model := decodeModel(state)
	phase := nextPhase(framework.StringValue(state.Data["phase_name"], "字节码准备"))
	switch phase {
	case "构造函数执行":
		model.StorageSnapshot["owner"] = "teacher"
		model.GasUsed = 53000
	case "状态初始化":
		model.StorageSnapshot["supply"] = "1000"
		model.GasUsed = 91000
	case "地址生成":
		model.ContractAddress = deriveAddress(model.Bytecode, model.Nonce)
		model.GasUsed = 121000
	}
	if err := rebuildState(state, model, phase); err != nil {
		return framework.StepOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, phase, fmt.Sprintf("合约部署进入%s阶段。", phase), toneByPhase(phase))
	return framework.StepOutput{
		Events: []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"storage": map[string]any{
					"owner":          model.StorageSnapshot["owner"],
					"supply":         model.StorageSnapshot["supply"],
					"init_code_hash": model.InitCodeHash,
					"gas_used":       model.GasUsed,
				},
				"address": model.ContractAddress,
			},
			"call_stack": map[string]any{
				"type":   "create",
				"stack":  []string{"Deployer", "Constructor", "Storage"},
				"target": "Address",
			},
		},
	}, nil
}

// HandleAction 使用新的 nonce 重新生成部署地址。
func HandleAction(state *framework.SceneState, input framework.ActionInput) (framework.ActionOutput, error) {
	model := decodeModel(state)
	model.Nonce = int(framework.NumberValue(input.Params["nonce"], float64(model.Nonce+1)))
	if deployer := framework.StringValue(input.Params["deployer"], ""); deployer != "" {
		model.Deployer = deployer
	}
	populateDerived(&model)
	if err := rebuildState(state, model, "地址生成"); err != nil {
		return framework.ActionOutput{}, err
	}
	event := framework.NewEvent(state.SceneCode, state.Tick, "重新部署", fmt.Sprintf("已使用 nonce=%d 重新部署。", model.Nonce), "info")
	return framework.ActionOutput{
		Success: true,
		Events:  []framework.TimelineEvent{event},
		SharedDiff: map[string]any{
			"contract_state": map[string]any{
				"address": model.ContractAddress,
				"storage": map[string]any{
					"init_code_hash": model.InitCodeHash,
					"gas_used":       model.GasUsed,
				},
			},
			"call_stack": map[string]any{
				"type":   "redeploy",
				"stack":  []string{"Deployer", "Constructor", "Address"},
				"target": "Address",
			},
		},
	}, nil
}

// BuildRenderState 输出字节码、构造参数、存储和地址。
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

// SyncSharedState 在合约安全组共享合约状态变化后重建部署场景。
func SyncSharedState(state *framework.SceneState, sharedState map[string]any) error {
	model := decodeModel(state)
	applySharedDeploymentState(&model, sharedState)
	return rebuildState(state, model, framework.StringValue(state.Data["phase_name"], state.Phase))
}

// deploymentModel 保存部署字节码、初始化存储和合约地址。
type deploymentModel struct {
	Bytecode        string            `json:"bytecode"`
	ConstructorArgs []string          `json:"constructor_args"`
	Nonce           int               `json:"nonce"`
	StorageSnapshot map[string]string `json:"storage_snapshot"`
	ContractAddress string            `json:"contract_address"`
	InitCodeHash    string            `json:"init_code_hash"`
	Deployer        string            `json:"deployer"`
	GasUsed         int               `json:"gas_used"`
}

// rebuildState 将部署模型映射为节点、部署消息和指标。
func rebuildState(state *framework.SceneState, model deploymentModel, phase string) error {
	state.Phase = phase
	state.PhaseIndex = phaseIndex(phase)
	state.Progress = framework.NextProgress(state.Tick, state.TotalTicks)
	for index := range state.Nodes {
		state.Nodes[index].Status = "normal"
	}
	state.Nodes[nodeIndexForPhase(state.PhaseIndex)].Status = "active"
	if phase == "地址生成" {
		state.Nodes[3].Status = "success"
	}
	state.Messages = []framework.Message{
		{ID: "bytecode-deploy", Label: framework.Abbreviate(model.Bytecode, 12), Kind: "call", Status: phase, SourceID: "bytecode", TargetID: "constructor"},
		{ID: "storage-address", Label: framework.Abbreviate(model.ContractAddress, 12), Kind: "call", Status: phase, SourceID: "storage", TargetID: "address"},
	}
	state.Metrics = []framework.Metric{
		{Key: "nonce", Label: "部署 Nonce", Value: fmt.Sprintf("%d", model.Nonce), Tone: "info"},
		{Key: "owner", Label: "Owner", Value: model.StorageSnapshot["owner"], Tone: "warning"},
		{Key: "supply", Label: "Supply", Value: model.StorageSnapshot["supply"], Tone: "success"},
		{Key: "gas", Label: "部署 Gas", Value: fmt.Sprintf("%d", model.GasUsed), Tone: "warning"},
		{Key: "address", Label: "合约地址", Value: framework.Abbreviate(model.ContractAddress, 12), Tone: toneByPhase(phase)},
	}
	state.Tooltip = []framework.TooltipEntry{
		{Label: "Bytecode", Value: model.Bytecode},
		{Label: "InitCodeHash", Value: model.InitCodeHash},
		{Label: "Deployer", Value: model.Deployer},
		{Label: "Args", Value: strings.Join(model.ConstructorArgs, ", ")},
		{Label: "Address", Value: model.ContractAddress},
	}
	state.Data = map[string]any{
		"phase_name":          phase,
		"contract_deployment": model,
	}
	state.Extra = map[string]any{
		"description": "该场景真实模拟合约部署中的字节码准备、构造函数执行、状态初始化和地址生成。",
	}
	state.ChangedKeys = []string{"nodes", "messages", "metrics", "data", "tooltip"}
	return nil
}

// decodeModel 从通用 JSON 状态恢复部署模型。
func decodeModel(state *framework.SceneState) deploymentModel {
	entry, ok := state.Data["contract_deployment"].(map[string]any)
	if !ok {
		if typed, ok := state.Data["contract_deployment"].(deploymentModel); ok {
			return typed
		}
		model := deploymentModel{
			Bytecode:        "6080604052",
			ConstructorArgs: []string{"owner=teacher", "supply=1000"},
			Nonce:           1,
			StorageSnapshot: map[string]string{"owner": "", "supply": "0"},
			Deployer:        "teacher-wallet",
		}
		populateDerived(&model)
		return model
	}
	model := deploymentModel{
		Bytecode:        framework.StringValue(entry["bytecode"], "6080604052"),
		ConstructorArgs: framework.ToStringSlice(entry["constructor_args"]),
		Nonce:           int(framework.NumberValue(entry["nonce"], 1)),
		StorageSnapshot: decodeStringMap(entry["storage_snapshot"]),
		ContractAddress: framework.StringValue(entry["contract_address"], ""),
		InitCodeHash:    framework.StringValue(entry["init_code_hash"], ""),
		Deployer:        framework.StringValue(entry["deployer"], "teacher-wallet"),
		GasUsed:         int(framework.NumberValue(entry["gas_used"], 0)),
	}
	populateDerived(&model)
	return model
}

// applySharedDeploymentState 将共享合约状态映射回部署模型。
func applySharedDeploymentState(model *deploymentModel, sharedState map[string]any) {
	if len(sharedState) == 0 {
		return
	}
	if contractState, ok := sharedState["contract_state"].(map[string]any); ok {
		if address, ok := contractState["address"].(string); ok && strings.TrimSpace(address) != "" {
			model.ContractAddress = address
		}
		if storage, ok := contractState["storage"].(map[string]any); ok {
			for key, value := range storage {
				model.StorageSnapshot[key] = scalarText(value, model.StorageSnapshot[key])
			}
			if initCodeHash, ok := storage["init_code_hash"].(string); ok && strings.TrimSpace(initCodeHash) != "" {
				model.InitCodeHash = initCodeHash
			}
			if gasUsed, ok := storage["gas_used"]; ok {
				model.GasUsed = int(framework.NumberValue(gasUsed, float64(model.GasUsed)))
			}
		}
	}
}

// populateDerived 补齐派生地址。
func populateDerived(model *deploymentModel) {
	if model.Deployer == "" {
		model.Deployer = "teacher-wallet"
	}
	model.InitCodeHash = framework.HashText(model.Bytecode + "|" + strings.Join(model.ConstructorArgs, "|"))
	if model.ContractAddress == "" {
		model.ContractAddress = deriveAddress(model.Bytecode, model.Nonce)
	}
	if model.GasUsed == 0 {
		model.GasUsed = 42000 + len(model.ConstructorArgs)*7000 + len(model.Bytecode)*4
	}
}

// deriveAddress 根据字节码和 nonce 生成稳定地址。
func deriveAddress(bytecode string, nonce int) string {
	return "0x" + framework.HashText(fmt.Sprintf("%s|%d", bytecode, nonce))[:40]
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

// nextPhase 返回部署流程的下一阶段。
func nextPhase(phase string) string {
	switch phase {
	case "字节码准备":
		return "构造函数执行"
	case "构造函数执行":
		return "状态初始化"
	case "状态初始化":
		return "地址生成"
	default:
		return "字节码准备"
	}
}

// phaseIndex 将阶段映射为时间线索引。
func phaseIndex(phase string) int {
	switch phase {
	case "字节码准备":
		return 0
	case "构造函数执行":
		return 1
	case "状态初始化":
		return 2
	case "地址生成":
		return 3
	default:
		return 0
	}
}

// toneByPhase 返回部署阶段色调。
func toneByPhase(phase string) string {
	if phase == "地址生成" {
		return "success"
	}
	return "info"
}

// decodeStringMap 恢复字符串映射。
func decodeStringMap(value any) map[string]string {
	entry, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]string); ok {
			return typed
		}
		return map[string]string{"owner": "", "supply": "0"}
	}
	result := make(map[string]string, len(entry))
	for key, raw := range entry {
		result[key] = framework.StringValue(raw, "")
	}
	return result
}

// scalarText 将共享状态中的标量值统一转换为字符串，兼容数字与布尔输入。
func scalarText(value any, fallback string) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fallback
	}
}
