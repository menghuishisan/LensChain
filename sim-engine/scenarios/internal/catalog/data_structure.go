package catalog

import (
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/blockchainstructure"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/blockinternal"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/bloomfilter"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/dhtstorage"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/mpttrie"
	"github.com/lenschain/sim-engine/scenarios/internal/framework"
)

// dataStructureDefinitions 返回数据结构领域的 5 个场景定义。
func dataStructureDefinitions() []framework.Definition {
	return []framework.Definition{
		buildDefinition(SceneTemplate{
			Code:            "blockchain-structure",
			Name:            "区块链结构与分叉",
			Description:     "展示主链、分叉链和最长链选择。",
			CategoryCode:    "data_structure",
			AlgorithmType:   "blockchain-structure",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"pow-attack-group", "blockchain-integrity-group"},
			Profile:         StepProfile{Stages: []string{"主链增长", "出现分叉", "最长链选择"}, TotalTicks: 14, StepDuration: 900},
			BaseNodeLabels:  []string{"Block-0", "Block-1", "Block-2", "Fork-A", "Fork-B"},
			BaseNodeRole:    "block",
			Actions: []ActionSpec{
				{ActionCode: "append_block", Label: "追加区块", Description: "向当前链尾追加新区块。", Trigger: "click", FieldKey: "branch", FieldLabel: "目标分支", FieldType: "string", DefaultValue: "main"},
				{ActionCode: "fork_chain", Label: "制造分叉", Description: "从指定高度制造一条分叉链。", Trigger: "form_submit", FieldKey: "height", FieldLabel: "分叉高度", FieldType: "number", DefaultValue: "1"},
			},
			DefaultState:  blockchainstructure.DefaultState,
			InitHandler:   blockchainstructure.Init,
			StepHandler:   blockchainstructure.Step,
			ActionHandler: blockchainstructure.HandleAction,
			SyncHandler:   blockchainstructure.SyncSharedState,
			RenderBuilder: blockchainstructure.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "block-internal",
			Name:            "区块内部结构",
			Description:     "展示区块头字段、交易体和 Merkle 根关系。",
			CategoryCode:    "data_structure",
			AlgorithmType:   "block-internal",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			LinkGroups:      []string{"blockchain-integrity-group"},
			Profile:         StepProfile{Stages: []string{"展开 Header", "展开 Body", "计算 Merkle Root"}, TotalTicks: 10, StepDuration: 900},
			BaseNodeLabels:  []string{"Header", "Body", "Nonce", "MerkleRoot"},
			BaseNodeRole:    "field",
			Actions: []ActionSpec{
				{ActionCode: "expand_field", Label: "展开字段", Description: "展开区块内部字段查看细节。", Trigger: "click", FieldKey: "field", FieldLabel: "字段名", FieldType: "string", DefaultValue: "header"},
			},
			DefaultState:  blockinternal.DefaultState,
			InitHandler:   blockinternal.Init,
			StepHandler:   blockinternal.Step,
			ActionHandler: blockinternal.HandleAction,
			SyncHandler:   blockinternal.SyncSharedState,
			RenderBuilder: blockinternal.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "mpt-trie",
			Name:            "状态树（MPT）",
			Description:     "展示路径查找、节点展开和状态更新传播。",
			CategoryCode:    "data_structure",
			AlgorithmType:   "mpt-trie",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"路径定位", "分支展开", "哈希回写"}, TotalTicks: 12, StepDuration: 900},
			BaseNodeLabels:  []string{"Root", "Branch-A", "Branch-B", "Leaf-X", "Leaf-Y"},
			BaseNodeRole:    "trie",
			Actions: []ActionSpec{
				{ActionCode: "update_account", Label: "更新账户", Description: "修改账户状态并触发树更新。", Trigger: "form_submit", FieldKey: "account", FieldLabel: "账户键", FieldType: "string", DefaultValue: "0xabc"},
			},
			DefaultState:  mpttrie.DefaultState,
			InitHandler:   mpttrie.Init,
			StepHandler:   mpttrie.Step,
			ActionHandler: mpttrie.HandleAction,
			RenderBuilder: mpttrie.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "bloom-filter",
			Name:            "布隆过滤器",
			Description:     "展示位数组、多哈希映射和误判统计。",
			CategoryCode:    "data_structure",
			AlgorithmType:   "bloom-filter",
			Version:         "v1.0.0",
			TimeControlMode: "reactive",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"哈希定位", "置位", "查询判断"}, TotalTicks: 8, StepDuration: 900},
			BaseNodeLabels:  []string{"Hash-1", "Hash-2", "Hash-3", "BitSet"},
			BaseNodeRole:    "hash",
			Actions: []ActionSpec{
				{ActionCode: "query_key", Label: "查询键", Description: "查询一个元素是否可能存在。", Trigger: "form_submit", FieldKey: "key", FieldLabel: "元素键", FieldType: "string", DefaultValue: "alice"},
			},
			DefaultState:  bloomfilter.DefaultState,
			InitHandler:   bloomfilter.Init,
			StepHandler:   bloomfilter.Step,
			ActionHandler: bloomfilter.HandleAction,
			RenderBuilder: bloomfilter.BuildRenderState,
		}),
		buildDefinition(SceneTemplate{
			Code:            "dht-storage",
			Name:            "分布式存储 DHT",
			Description:     "展示环形空间、键值映射和分片分布。",
			CategoryCode:    "data_structure",
			AlgorithmType:   "dht-storage",
			Version:         "v1.0.0",
			TimeControlMode: "continuous",
			DataSourceMode:  "simulation",
			Profile:         StepProfile{Stages: []string{"键映射", "副本分片", "路由查找"}, TotalTicks: 14, StepDuration: 1200},
			BaseNodeLabels:  []string{"Node-1", "Node-2", "Node-3", "Node-4", "Node-5", "Node-6"},
			BaseNodeRole:    "storage",
			Actions: []ActionSpec{
				{ActionCode: "store_key", Label: "存储键值", Description: "向 DHT 中写入一个新键。", Trigger: "form_submit", FieldKey: "key", FieldLabel: "键名", FieldType: "string", DefaultValue: "doc-1"},
			},
			DefaultState:  dhtstorage.DefaultState,
			InitHandler:   dhtstorage.Init,
			StepHandler:   dhtstorage.Step,
			ActionHandler: dhtstorage.HandleAction,
			RenderBuilder: dhtstorage.BuildRenderState,
		}),
	}
}
