package catalog

import (
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/crosschainbridge"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/gascalculation"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/tokentransfer"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/txlifecycle"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/txorderingmev"
)

// transactionDefinitions 返回交易领域的 5 个场景定义。
func transactionDefinitions() []framework.Definition {
	return []framework.Definition{
		buildDefinition(SceneTemplate{
			Code:            "tx-lifecycle",
			Name:            "交易生命周期",
			Description:     "展示创建、签名、广播、打包和确认全流程。",
			CategoryCode:    "transaction",
			AlgorithmType:   "tx-lifecycle",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"blockchain-integrity-group", "tx-processing-group"},
			Profile:         StepProfile{Stages: []string{"创建", "签名", "广播", "内存池", "打包", "确认"}, TotalTicks: 18, StepDuration: 1400},
			BaseNodeLabels:  []string{"Wallet", "Signer", "Peer", "Mempool", "Miner"},
			BaseNodeRole:    "tx",
			Actions: []ActionSpec{
				{ActionCode: "create_tx", Label: "创建交易", Description: "创建一笔新的链上交易。", Trigger: "form_submit", FieldKey: "tx", FieldLabel: "交易标识", FieldType: "string", DefaultValue: "tx-1"},
			},
			DefaultState:  txlifecycle.DefaultState,
			InitHandler:   txlifecycle.Init,
			StepHandler:   txlifecycle.Step,
			ActionHandler: txlifecycle.HandleAction,
			SyncHandler:   txlifecycle.SyncSharedState,
			RenderBuilder: txlifecycle.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "gas-calculation",
			Name:            "Gas 计算与优化",
			Description:     "展示操作码 Gas 瀑布图和优化建议。",
			CategoryCode:    "transaction",
			AlgorithmType:   "gas-calculation",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"tx-processing-group"},
			Profile:         StepProfile{Stages: []string{"Opcode 分析", "Gas 汇总", "优化建议"}, TotalTicks: 10, StepDuration: 900},
			BaseNodeLabels:  []string{"Opcode", "Gas", "Limit", "Advice"},
			BaseNodeRole:    "gas",
			Actions: []ActionSpec{
				{ActionCode: "switch_opcode", Label: "切换操作码", Description: "切换执行路径查看 Gas 变化。", Trigger: "click", FieldKey: "opcode", FieldLabel: "操作码", FieldType: "string", DefaultValue: "SSTORE"},
			},
			DefaultState:  gascalculation.DefaultState,
			InitHandler:   gascalculation.Init,
			StepHandler:   gascalculation.Step,
			ActionHandler: gascalculation.HandleAction,
			SyncHandler:   gascalculation.SyncSharedState,
			RenderBuilder: gascalculation.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "token-transfer",
			Name:            "Token 转账流转",
			Description:     "展示账户余额变化和 ERC-20 事件日志。",
			CategoryCode:    "transaction",
			AlgorithmType:   "token-transfer",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"tx-processing-group"},
			Profile:         StepProfile{Stages: []string{"扣减余额", "事件广播", "接收到账"}, TotalTicks: 12, StepDuration: 1300},
			BaseNodeLabels:  []string{"Sender", "Token", "Receiver"},
			BaseNodeRole:    "account",
			Actions: []ActionSpec{
				{ActionCode: "transfer_token", Label: "执行转账", Description: "发起一次新的 Token 转账。", Trigger: "form_submit", FieldKey: "amount", FieldLabel: "转账数量", FieldType: "number", DefaultValue: "10"},
			},
			DefaultState:  tokentransfer.DefaultState,
			InitHandler:   tokentransfer.Init,
			StepHandler:   tokentransfer.Step,
			ActionHandler: tokentransfer.HandleAction,
			SyncHandler:   tokentransfer.SyncSharedState,
			RenderBuilder: tokentransfer.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "tx-ordering-mev",
			Name:            "交易排序与 MEV",
			Description:     "展示内存池排序、矿工优先级和三明治攻击。",
			CategoryCode:    "transaction",
			AlgorithmType:   "tx-ordering-mev",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"tx-processing-group"},
			Profile:         StepProfile{Stages: []string{"交易入池", "按费率排序", "夹击插单", "打包确认"}, TotalTicks: 16, StepDuration: 1400},
			BaseNodeLabels:  []string{"User-Tx", "Bot-Tx", "Miner"},
			BaseNodeRole:    "mempool",
			Actions: []ActionSpec{
				{ActionCode: "inject_mev_bot", Label: "注入 MEV Bot", Description: "向交易池中注入抢跑交易。", Trigger: "click", FieldKey: "bot", FieldLabel: "机器人标识", FieldType: "string", DefaultValue: "bot-1"},
			},
			DefaultState:  txorderingmev.DefaultState,
			InitHandler:   txorderingmev.Init,
			StepHandler:   txorderingmev.Step,
			ActionHandler: txorderingmev.HandleAction,
			SyncHandler:   txorderingmev.SyncSharedState,
			RenderBuilder: txorderingmev.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "cross-chain-bridge",
			Name:            "跨链桥接通信",
			Description:     "展示锁定、证明、中继和目标链铸造。",
			CategoryCode:    "transaction",
			AlgorithmType:   "cross-chain-bridge",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"源链锁定", "证明生成", "中继提交", "目标链铸造"}, TotalTicks: 16, StepDuration: 1500},
			BaseNodeLabels:  []string{"Source", "Relay", "Target"},
			BaseNodeRole:    "bridge",
			Actions: []ActionSpec{
				{ActionCode: "bridge_asset", Label: "桥接资产", Description: "发起一笔跨链桥接操作。", Trigger: "form_submit", FieldKey: "asset", FieldLabel: "资产名称", FieldType: "string", DefaultValue: "USDT"},
			},
			DefaultState:  crosschainbridge.DefaultState,
			InitHandler:   crosschainbridge.Init,
			StepHandler:   crosschainbridge.Step,
			ActionHandler: crosschainbridge.HandleAction,
			RenderBuilder: crosschainbridge.BuildRenderState,
		}),
	}
}
