// 模块：sim-engine/scenarios/catalog
// 文件职责：聚合 8 类目下所有场景包的 Definition()。
//
// 该文件仅 import 已实现的场景子包并按类目调用其 Definition()，不写任何业务逻辑。
// 接入新场景的固定流程：
//   1. 在 `scenarios/internal/<category>/<scene>/scene.go` 中实现 `framework.Definition`。
//   2. 在本文件 import 该子包。
//   3. 在 collectAll 内对应类目分组追加 `<pkg>.Definition()` 到返回列表。
//   4. 与 `backend/migrations/013_seed_sim_scenarios.up.sql` 元信息核对一致。

package catalog

import (
	"github.com/lenschain/sim-engine/framework"

	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/doublespend"
	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/fiftyoneattack"
	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/integeroverflow"
	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/pbftbyzantine"
	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/reentrancyattack"
	"github.com/lenschain/sim-engine/scenarios/internal/attacksecurity/selfishmining"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/dposvoting"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/pbftconsensus"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/posvalidator"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/powmining"
	"github.com/lenschain/sim-engine/scenarios/internal/consensus/raftelection"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/ecdsasign"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/keccak256hash"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/merkletree"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/rsaencrypt"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/sha256hash"
	"github.com/lenschain/sim-engine/scenarios/internal/cryptography/zkpbasic"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/blockchainstructure"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/blockinternal"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/bloomfilter"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/dhtstorage"
	"github.com/lenschain/sim-engine/scenarios/internal/datastructure/mpttrie"
	"github.com/lenschain/sim-engine/scenarios/internal/economic/defiliquidity"
	"github.com/lenschain/sim-engine/scenarios/internal/economic/gasmarket"
	"github.com/lenschain/sim-engine/scenarios/internal/economic/governancevoting"
	"github.com/lenschain/sim-engine/scenarios/internal/economic/posstaking"
	"github.com/lenschain/sim-engine/scenarios/internal/economic/tokeneconomics"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/blocksync"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/gossippropagation"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/networkpartition"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/nodeloadbalance"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/p2pdiscovery"
	"github.com/lenschain/sim-engine/scenarios/internal/nodenetwork/txbroadcast"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractdeployment"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractinteraction"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/contractstatemachine"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/evmexecution"
	"github.com/lenschain/sim-engine/scenarios/internal/smartcontract/statechannel"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/crosschainbridge"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/gascalculation"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/tokentransfer"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/txlifecycle"
	"github.com/lenschain/sim-engine/scenarios/internal/transaction/txorderingmev"
)

// collectAll 聚合 43 内置场景的 Definition。
//
// 每个场景包必须导出 `Definition() framework.Definition`；
// 此处按 06.md §四 的领域顺序排列：
//
//	§4.1 节点与网络（6） §4.2 共识过程（5） §4.3 密码学运算（6）
//	§4.4 数据结构（5）   §4.5 交易生命周期（5） §4.6 智能合约（5）
//	§4.7 攻击与安全（6） §4.8 经济模型（5）
//
// 全部 43 个场景已完整实现并注册；NewRegistry 期望 len(collectAll()) == 43。
// 后续若新增场景，遵循"接入新场景的固定流程"：先在对应类目子目录创建 Definition()，
// 再在本文件按字典序追加 import 与 append。
func collectAll() []framework.Definition {
	all := make([]framework.Definition, 0, 43)

	// §4.1 节点与网络（6）：p2pdiscovery / gossippropagation / networkpartition /
	// txbroadcast / blocksync / nodeloadbalance
	all = append(all, p2pdiscovery.Definition())
	all = append(all, gossippropagation.Definition())
	all = append(all, networkpartition.Definition())
	all = append(all, txbroadcast.Definition())
	all = append(all, blocksync.Definition())
	all = append(all, nodeloadbalance.Definition())

	// §4.2 共识过程（5）：powmining / posvalidator / pbftconsensus / raftelection / dposvoting
	all = append(all, powmining.Definition())
	all = append(all, posvalidator.Definition())
	all = append(all, raftelection.Definition())
	all = append(all, dposvoting.Definition())
	all = append(all, pbftconsensus.Definition())

	// §4.3 密码学运算（6）：sha256hash / keccak256hash / ecdsasign / rsaencrypt /
	// merkletree / zkpbasic
	all = append(all, sha256hash.Definition())
	all = append(all, keccak256hash.Definition())
	all = append(all, merkletree.Definition())
	all = append(all, ecdsasign.Definition())
	all = append(all, rsaencrypt.Definition())
	all = append(all, zkpbasic.Definition())

	// §4.4 数据结构（5）：blockchainstructure / blockinternal / mpttrie / bloomfilter / dhtstorage
	all = append(all, blockchainstructure.Definition())
	all = append(all, blockinternal.Definition())
	all = append(all, mpttrie.Definition())
	all = append(all, bloomfilter.Definition())
	all = append(all, dhtstorage.Definition())

	// §4.5 交易生命周期（5）：txlifecycle / gascalculation / tokentransfer /
	// txorderingmev / crosschainbridge
	all = append(all, txlifecycle.Definition())
	all = append(all, gascalculation.Definition())
	all = append(all, tokentransfer.Definition())
	all = append(all, txorderingmev.Definition())
	all = append(all, crosschainbridge.Definition())

	// §4.6 智能合约（5）：contractstatemachine / evmexecution / contractinteraction /
	// contractdeployment / statechannel
	all = append(all, contractstatemachine.Definition())
	all = append(all, evmexecution.Definition())
	all = append(all, contractinteraction.Definition())
	all = append(all, contractdeployment.Definition())
	all = append(all, statechannel.Definition())

	// §4.7 攻击与安全（6）：fiftyoneattack / doublespend / pbftbyzantine /
	// reentrancyattack / integeroverflow / selfishmining
	all = append(all, fiftyoneattack.Definition())
	all = append(all, doublespend.Definition())
	all = append(all, pbftbyzantine.Definition())
	all = append(all, reentrancyattack.Definition())
	all = append(all, integeroverflow.Definition())
	all = append(all, selfishmining.Definition())

	// §4.8 经济模型（5）：tokeneconomics / posstaking / governancevoting /
	// defiliquidity / gasmarket
	all = append(all, tokeneconomics.Definition())
	all = append(all, posstaking.Definition())
	all = append(all, governancevoting.Definition())
	all = append(all, defiliquidity.Definition())
	all = append(all, gasmarket.Definition())

	return all
}
