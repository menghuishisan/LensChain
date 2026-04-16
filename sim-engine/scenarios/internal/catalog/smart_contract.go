package catalog

import (
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractdeployment"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractinteraction"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractstatemachine"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/evmexecution"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/statechannel"
)

// smartContractDefinitions 返回智能合约领域的 5 个场景定义。
func smartContractDefinitions() []framework.Definition {
	return []framework.Definition{
		buildDefinition(SceneTemplate{
			Code:            "contract-state-machine",
			Name:            "智能合约状态机",
			Description:     "展示状态转换、事件触发和存储读写。",
			CategoryCode:    "smart_contract",
			AlgorithmType:   "contract-state-machine",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"contract-security-group"},
			Profile:         StepProfile{Stages: []string{"状态触发", "状态迁移", "存储更新"}, TotalTicks: 10, StepDuration: 1000},
			BaseNodeLabels:  []string{"Created", "Active", "Paused", "Closed"},
			BaseNodeRole:    "state",
			Actions: []ActionSpec{
				{ActionCode: "fire_event", Label: "触发事件", Description: "触发状态迁移事件。", Trigger: "click", FieldKey: "event", FieldLabel: "事件名", FieldType: "string", DefaultValue: "activate"},
			},
			DefaultState:  contractstatemachine.DefaultState,
			InitHandler:   contractstatemachine.Init,
			StepHandler:   contractstatemachine.Step,
			ActionHandler: contractstatemachine.HandleAction,
			SyncHandler:   contractstatemachine.SyncSharedState,
			RenderBuilder: contractstatemachine.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "evm-execution",
			Name:            "EVM 执行步进",
			Description:     "展示操作码执行、栈内存变化和存储写回。",
			CategoryCode:    "smart_contract",
			AlgorithmType:   "evm-execution",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"contract-security-group"},
			Profile:         StepProfile{Stages: []string{"取指", "执行", "栈更新", "存储写回"}, TotalTicks: 20, StepDuration: 1300},
			BaseNodeLabels:  []string{"PC", "Stack", "Memory", "Storage"},
			BaseNodeRole:    "vm",
			Actions: []ActionSpec{
				{ActionCode: "step_opcode", Label: "执行单步", Description: "手动推动一个操作码。", Trigger: "click", FieldKey: "opcode", FieldLabel: "操作码", FieldType: "string", DefaultValue: "SLOAD"},
			},
			DefaultState:  evmexecution.DefaultState,
			InitHandler:   evmexecution.Init,
			StepHandler:   evmexecution.Step,
			ActionHandler: evmexecution.HandleAction,
			SyncHandler:   evmexecution.SyncSharedState,
			RenderBuilder: evmexecution.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "contract-interaction",
			Name:            "合约间调用",
			Description:     "展示 call、delegatecall 和上下文切换。",
			CategoryCode:    "smart_contract",
			AlgorithmType:   "contract-interaction",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"进入调用", "上下文切换", "返回值传播"}, TotalTicks: 14, StepDuration: 1300},
			BaseNodeLabels:  []string{"Caller", "Library", "Vault", "Receiver"},
			BaseNodeRole:    "contract",
			Actions: []ActionSpec{
				{ActionCode: "invoke_delegatecall", Label: "触发 delegatecall", Description: "切换到 delegatecall 调用链路。", Trigger: "click", FieldKey: "resource_id", FieldLabel: "目标合约", FieldType: "string", DefaultValue: "Library"},
			},
			DefaultState:  contractinteraction.DefaultState,
			InitHandler:   contractinteraction.Init,
			StepHandler:   contractinteraction.Step,
			ActionHandler: contractinteraction.HandleAction,
			RenderBuilder: contractinteraction.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "contract-deployment",
			Name:            "合约部署流程",
			Description:     "展示字节码、构造函数和部署地址推导。",
			CategoryCode:    "smart_contract",
			AlgorithmType:   "contract-deployment",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"字节码准备", "构造函数执行", "状态初始化", "地址生成"}, TotalTicks: 12, StepDuration: 1400},
			BaseNodeLabels:  []string{"Bytecode", "Constructor", "Storage", "Address"},
			BaseNodeRole:    "deploy",
			Actions: []ActionSpec{
				{ActionCode: "redeploy_contract", Label: "重新部署", Description: "使用新的 nonce 重新部署合约。", Trigger: "form_submit", FieldKey: "nonce", FieldLabel: "部署 nonce", FieldType: "number", DefaultValue: "1"},
			},
			DefaultState:  contractdeployment.DefaultState,
			InitHandler:   contractdeployment.Init,
			StepHandler:   contractdeployment.Step,
			ActionHandler: contractdeployment.HandleAction,
			RenderBuilder: contractdeployment.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "state-channel",
			Name:            "状态通道",
			Description:     "展示链上开通、链下更新、争议和关闭流程。",
			CategoryCode:    "smart_contract",
			AlgorithmType:   "state-channel",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"开通通道", "链下更新", "争议提交", "通道关闭"}, TotalTicks: 16, StepDuration: 1400},
			BaseNodeLabels:  []string{"Party-A", "Party-B", "Adjudicator"},
			BaseNodeRole:    "channel",
			Actions: []ActionSpec{
				{ActionCode: "submit_dispute", Label: "提交争议", Description: "向链上提交一笔争议。", Trigger: "click", FieldKey: "proof", FieldLabel: "争议证明", FieldType: "string", DefaultValue: "proof-1"},
			},
			DefaultState:  statechannel.DefaultState,
			InitHandler:   statechannel.Init,
			StepHandler:   statechannel.Step,
			ActionHandler: statechannel.HandleAction,
			RenderBuilder: statechannel.BuildRenderState,
		}),
	}
}
