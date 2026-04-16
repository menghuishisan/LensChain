package catalog

import (
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/dposvoting"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/pbftconsensus"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/posvalidator"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/powmining"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/raftelection"
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// consensusDefinitions 返回共识领域的 5 个场景定义。
func consensusDefinitions() []framework.Definition {
	return []framework.Definition{
		buildDefinition(SceneTemplate{
			Code:            "pow-mining",
			Name:            "PoW 挖矿竞争",
			Description:     "展示算力竞争、Nonce 搜索和新区块出块过程。",
			CategoryCode:    "consensus",
			AlgorithmType:   "pow-mining",
			Version:         "v1.0.0",
			TimeControlMode: "continuous",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"pow-attack-group"},
			Profile:         StepProfile{Stages: []string{"分配算力", "Nonce 搜索", "命中目标", "出块广播"}, TotalTicks: 24, StepDuration: 1500},
			BaseNodeLabels:  []string{"Miner-A", "Miner-B", "Miner-C", "Miner-D"},
			BaseNodeRole:    "miner",
			Actions: []ActionSpec{
				{
					ActionCode:  "adjust_hashrate",
					Label:       "调整算力",
					Description: "动态调整某个矿工的算力。",
					Trigger:     "form_submit",
					Fields: []framework.InteractionFieldDefinition{
						{Key: "resource_id", Label: "矿工标识", Type: "node_ref", Required: true, DefaultValue: "miner-a"},
						{Key: "hashrate", Label: "算力倍率", Type: "number", Required: true, DefaultValue: "1.2"},
					},
				},
			},
			DefaultState:  powmining.DefaultState,
			InitHandler:   powmining.Init,
			StepHandler:   powmining.Step,
			ActionHandler: powmining.HandleAction,
			SyncHandler:   powmining.SyncSharedState,
			RenderBuilder: powmining.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "pos-validator",
			Name:            "PoS 验证者选举",
			Description:     "展示质押权重、随机选举与 Epoch 轮转。",
			CategoryCode:    "consensus",
			AlgorithmType:   "pos-validator",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"pos-economy-group"},
			Profile:         StepProfile{Stages: []string{"质押汇总", "随机抽样", "Epoch 轮转", "奖励结算"}, TotalTicks: 18, StepDuration: 1500},
			BaseNodeLabels:  []string{"Validator-A", "Validator-B", "Validator-C", "Validator-D"},
			BaseNodeRole:    "validator",
			Actions: []ActionSpec{
				{
					ActionCode:  "delegate_stake",
					Label:       "追加质押",
					Description: "调整验证者权重。",
					Trigger:     "form_submit",
					Fields: []framework.InteractionFieldDefinition{
						{Key: "resource_id", Label: "验证者标识", Type: "node_ref", Required: true, DefaultValue: "validator-a"},
						{Key: "stake", Label: "质押数量", Type: "number", Required: true, DefaultValue: "100"},
					},
				},
			},
			DefaultState:  posvalidator.DefaultState,
			InitHandler:   posvalidator.Init,
			StepHandler:   posvalidator.Step,
			ActionHandler: posvalidator.HandleAction,
			SyncHandler:   posvalidator.SyncSharedState,
			RenderBuilder: posvalidator.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "pbft-consensus",
			Name:            "PBFT 三阶段共识",
			Description:     "展示 Pre-prepare、Prepare、Commit 三阶段与视图切换。",
			CategoryCode:    "consensus",
			AlgorithmType:   "pbft-consensus",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"pbft-attack-group"},
			Profile:         StepProfile{Stages: []string{"Pre-prepare", "Prepare", "Commit", "Checkpoint", "View Change"}, TotalTicks: 25, StepDuration: 1600},
			BaseNodeLabels:  []string{"Replica-0", "Replica-1", "Replica-2", "Replica-3"},
			BaseNodeRole:    "replica",
			Actions: []ActionSpec{
				{ActionCode: "inject_byzantine_node", Label: "注入拜占庭节点", Description: "将某个副本切换为异常节点。", Trigger: "form_submit", FieldKey: "resource_id", FieldLabel: "节点标识", FieldType: "node_ref", DefaultValue: "Replica-1"},
				{ActionCode: "trigger_view_change", Label: "触发视图切换", Description: "模拟主节点超时后进行视图切换。", Trigger: "click", FieldKey: "view", FieldLabel: "视图编号", FieldType: "number", DefaultValue: "1"},
			},
			DefaultState:  pbftconsensus.DefaultState,
			InitHandler:   pbftconsensus.Init,
			StepHandler:   pbftconsensus.Step,
			ActionHandler: pbftconsensus.HandleAction,
			SyncHandler:   pbftconsensus.SyncSharedState,
			RenderBuilder: pbftconsensus.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "raft-election",
			Name:            "Raft 领导选举",
			Description:     "展示超时、拉票、Leader 产生和日志复制。",
			CategoryCode:    "consensus",
			AlgorithmType:   "raft-election",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"raft-fault-group"},
			Profile:         StepProfile{Stages: []string{"Follower 超时", "Candidate 拉票", "Leader 当选", "日志复制"}, TotalTicks: 20, StepDuration: 1500},
			BaseNodeLabels:  []string{"Node-A", "Node-B", "Node-C", "Node-D", "Node-E"},
			BaseNodeRole:    "raft",
			Actions: []ActionSpec{
				{ActionCode: "fail_leader", Label: "宕掉 Leader", Description: "模拟当前 Leader 故障。", Trigger: "click", FieldKey: "resource_id", FieldLabel: "Leader 标识", FieldType: "node_ref", DefaultValue: "Node-A"},
			},
			DefaultState:  raftelection.DefaultState,
			InitHandler:   raftelection.Init,
			StepHandler:   raftelection.Step,
			ActionHandler: raftelection.HandleAction,
			SyncHandler:   raftelection.SyncSharedState,
			RenderBuilder: raftelection.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "dpos-voting",
			Name:            "DPoS 委托投票",
			Description:     "展示投票权重流向、超级节点排名和轮次出块。",
			CategoryCode:    "consensus",
			AlgorithmType:   "dpos-voting",
			Version:         "v1.0.0",
			TimeControlMode: "process",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"委托投票", "权重汇总", "超级节点排名", "轮次出块"}, TotalTicks: 18, StepDuration: 1500},
			BaseNodeLabels:  []string{"Delegate-1", "Delegate-2", "Delegate-3", "Delegate-4", "Delegate-5"},
			BaseNodeRole:    "delegate",
			Actions: []ActionSpec{
				{ActionCode: "reassign_vote", Label: "重新投票", Description: "将票权重新委托给目标代表。", Trigger: "form_submit", FieldKey: "resource_id", FieldLabel: "代表标识", FieldType: "node_ref", DefaultValue: "Delegate-1"},
			},
			DefaultState:  dposvoting.DefaultState,
			InitHandler:   dposvoting.Init,
			StepHandler:   dposvoting.Step,
			ActionHandler: dposvoting.HandleAction,
			RenderBuilder: dposvoting.BuildRenderState,
		}),
	}
}
