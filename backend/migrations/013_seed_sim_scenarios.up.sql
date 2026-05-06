-- 013_seed_sim_scenarios.up.sql
-- 补充仿真场景库、联动组与纯仿真/混合实验模板种子数据。
-- 目标：
-- 1. 添加 43 个内置仿真场景（8 大领域），与 sim-engine/scenarios 目录一一对应
-- 2. 添加 7 个联动组及其场景关联
-- 3. 添加 4 个纯仿真实验模板（experiment_type=1）
-- 4. 添加 2 个混合实验模板（experiment_type=3，含容器 + 仿真场景）
-- 5. 为新模板添加 template_sim_scenes、检查点、标签与课程关联
--
-- 使用方式：在 010、011、012 之后执行。
-- 约定：所有 ID 使用固定值，与 010/012 系列延续编号。

-- =====================================================================
-- 01. 仿真场景库 — 43 个内置场景
-- =====================================================================
-- sim_scenarios 列:
--   id, name, code, category, description,
--   source_type (1=平台内置), status (1=正常),
--   algorithm_type, time_control_mode, data_source_mode (1=仿真,2=采集,3=双模式),
--   default_params, interaction_schema, default_size, delivery_phase (1=已就绪),
--   version, created_at, updated_at

INSERT INTO sim_scenarios (
    id, name, code, category, description,
    source_type, status,
    algorithm_type, time_control_mode, data_source_mode,
    default_params, interaction_schema, default_size, delivery_phase,
    version, created_at, updated_at
)
VALUES
-- ---- 节点与网络 (node_network) — 6 个 ----
(
    920000000000001001, 'P2P 网络发现与路由', 'p2p-discovery', 'node_network',
    '展示节点发现、邻居路由表收敛和拓扑稳定过程。',
    1, 1, 'p2p-discovery', 'continuous', 1,
    '{"scene_code":"p2p-discovery","algorithm_type":"p2p-discovery","time_control":"continuous","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"p2p-discovery","actions":[{"action_code":"add_peer","label":"新增节点","trigger":"form_submit","fields":[{"key":"peer_label","label":"节点名称","type":"string","required":true,"default_value":"Peer-X"}]},{"action_code":"shuffle_route","label":"扰动路由","trigger":"click","fields":[{"key":"resource_id","label":"目标节点","type":"node_ref","required":true,"default_value":"Peer-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001002, 'Gossip 消息传播', 'gossip-propagation', 'node_network',
    '展示消息从源节点向全网扩散的覆盖过程。',
    1, 1, 'gossip-propagation', 'continuous', 1,
    '{"scene_code":"gossip-propagation","algorithm_type":"gossip-propagation","time_control":"continuous","step_duration":1200,"total_ticks":15}'::jsonb,
    '{"scene_code":"gossip-propagation","actions":[{"action_code":"seed_message","label":"投放消息","trigger":"form_submit","fields":[{"key":"message_label","label":"消息标签","type":"string","required":true,"default_value":"TX-001"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001003, '网络分区与恢复', 'network-partition', 'node_network',
    '展示网络隔离边界形成和恢复同步流程。',
    1, 1, 'network-partition', 'process', 1,
    '{"scene_code":"network-partition","algorithm_type":"network-partition","time_control":"process","step_duration":1600,"total_ticks":20}'::jsonb,
    '{"scene_code":"network-partition","actions":[{"action_code":"cut_link","label":"切断链路","trigger":"click","fields":[{"key":"resource_id","label":"目标链路","type":"string","required":true,"default_value":"A-B"}]},{"action_code":"restore_link","label":"恢复链路","trigger":"click","fields":[{"key":"resource_id","label":"目标链路","type":"string","required":true,"default_value":"A-B"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001004, '交易广播与打包', 'tx-broadcast', 'node_network',
    '展示交易广播、矿工收集和内存池填充。',
    1, 1, 'tx-broadcast', 'process', 1,
    '{"scene_code":"tx-broadcast","algorithm_type":"tx-broadcast","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"tx-broadcast","actions":[{"action_code":"broadcast_tx","label":"发起交易","trigger":"form_submit","fields":[{"key":"tx_label","label":"交易标识","type":"string","required":true,"default_value":"tx-1001"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001005, '区块同步与传播', 'block-sync', 'node_network',
    '展示新区块传播、高度追平和分叉检测。',
    1, 1, 'block-sync', 'process', 1,
    '{"scene_code":"block-sync","algorithm_type":"block-sync","time_control":"process","step_duration":1500,"total_ticks":20}'::jsonb,
    '{"scene_code":"block-sync","actions":[{"action_code":"inject_block","label":"注入新区块","trigger":"click","fields":[{"key":"block_height","label":"区块高度","type":"number","required":true,"default_value":"10"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001006, '节点负载均衡', 'node-load-balance', 'node_network',
    '展示网络请求在节点间的路由与迁移。',
    1, 1, 'node-load-balance', 'continuous', 1,
    '{"scene_code":"node-load-balance","algorithm_type":"node-load-balance","time_control":"continuous","step_duration":1200,"total_ticks":14}'::jsonb,
    '{"scene_code":"node-load-balance","actions":[{"action_code":"shift_traffic","label":"迁移流量","trigger":"drag","fields":[{"key":"resource_id","label":"目标节点","type":"node_ref","required":true,"default_value":"LB-2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 共识机制 (consensus) — 5 个 ----
(
    920000000000001007, 'PoW 挖矿竞争', 'pow-mining', 'consensus',
    '展示算力竞争、Nonce 搜索和新区块出块过程。',
    1, 1, 'pow-mining', 'continuous', 1,
    '{"scene_code":"pow-mining","algorithm_type":"pow-mining","time_control":"continuous","step_duration":1500,"total_ticks":24}'::jsonb,
    '{"scene_code":"pow-mining","actions":[{"action_code":"adjust_hashrate","label":"调整算力","trigger":"form_submit","fields":[{"key":"resource_id","label":"矿工标识","type":"node_ref","required":true,"default_value":"miner-a"},{"key":"hashrate","label":"算力倍率","type":"number","required":true,"default_value":"1.2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001008, 'PoS 验证者选举', 'pos-validator', 'consensus',
    '展示质押权重、随机选举与 Epoch 轮转。',
    1, 1, 'pos-validator', 'process', 1,
    '{"scene_code":"pos-validator","algorithm_type":"pos-validator","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"pos-validator","actions":[{"action_code":"delegate_stake","label":"追加质押","trigger":"form_submit","fields":[{"key":"resource_id","label":"验证者标识","type":"node_ref","required":true,"default_value":"validator-a"},{"key":"stake","label":"质押数量","type":"number","required":true,"default_value":"100"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001009, 'PBFT 三阶段共识', 'pbft-consensus', 'consensus',
    '展示 Pre-prepare、Prepare、Commit 三阶段与视图切换。',
    1, 1, 'pbft-consensus', 'process', 1,
    '{"scene_code":"pbft-consensus","algorithm_type":"pbft-consensus","time_control":"process","step_duration":1600,"total_ticks":25}'::jsonb,
    '{"scene_code":"pbft-consensus","actions":[{"action_code":"inject_byzantine_node","label":"注入拜占庭节点","trigger":"form_submit","fields":[{"key":"resource_id","label":"节点标识","type":"node_ref","required":true,"default_value":"Replica-1"}]},{"action_code":"trigger_view_change","label":"触发视图切换","trigger":"click","fields":[{"key":"view","label":"视图编号","type":"number","required":true,"default_value":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001010, 'Raft 领导选举', 'raft-election', 'consensus',
    '展示超时、拉票、Leader 产生和日志复制。',
    1, 1, 'raft-election', 'process', 1,
    '{"scene_code":"raft-election","algorithm_type":"raft-election","time_control":"process","step_duration":1500,"total_ticks":20}'::jsonb,
    '{"scene_code":"raft-election","actions":[{"action_code":"fail_leader","label":"宕掉 Leader","trigger":"click","fields":[{"key":"resource_id","label":"Leader 标识","type":"node_ref","required":true,"default_value":"Node-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001011, 'DPoS 委托投票', 'dpos-voting', 'consensus',
    '展示投票权重流向、超级节点排名和轮次出块。',
    1, 1, 'dpos-voting', 'process', 1,
    '{"scene_code":"dpos-voting","algorithm_type":"dpos-voting","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"dpos-voting","actions":[{"action_code":"reassign_vote","label":"重新投票","trigger":"form_submit","fields":[{"key":"resource_id","label":"代表标识","type":"node_ref","required":true,"default_value":"Delegate-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 密码学 (cryptography) — 6 个 ----
(
    920000000000001012, 'SHA-256 哈希过程', 'sha256-hash', 'cryptography',
    '展示消息分块、填充和 64 轮压缩函数的状态演化。',
    1, 1, 'sha256-hash', 'reactive', 1,
    '{"scene_code":"sha256-hash","algorithm_type":"sha256-hash","time_control":"reactive","step_duration":900,"total_ticks":64}'::jsonb,
    '{"scene_code":"sha256-hash","actions":[{"action_code":"mutate_input","label":"修改输入","trigger":"form_submit","fields":[{"key":"input","label":"输入内容","type":"string","required":true,"default_value":"abc"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001013, 'Keccak-256 哈希过程', 'keccak256-hash', 'cryptography',
    '展示海绵结构吸收、置换和挤压输出。',
    1, 1, 'keccak256-hash', 'reactive', 1,
    '{"scene_code":"keccak256-hash","algorithm_type":"keccak256-hash","time_control":"reactive","step_duration":900,"total_ticks":24}'::jsonb,
    '{"scene_code":"keccak256-hash","actions":[{"action_code":"toggle_lane","label":"切换 Lane","trigger":"click","fields":[{"key":"lane","label":"Lane 索引","type":"number","required":true,"default_value":"0"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001014, 'ECDSA 签名验签', 'ecdsa-sign', 'cryptography',
    '展示密钥对、随机数生成、签名和验签流程。',
    1, 1, 'ecdsa-sign', 'reactive', 1,
    '{"scene_code":"ecdsa-sign","algorithm_type":"ecdsa-sign","time_control":"reactive","step_duration":1000,"total_ticks":12}'::jsonb,
    '{"scene_code":"ecdsa-sign","actions":[{"action_code":"sign_message","label":"重新签名","trigger":"form_submit","fields":[{"key":"message","label":"消息内容","type":"string","required":true,"default_value":"hello"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001015, 'RSA 加密解密', 'rsa-encrypt', 'cryptography',
    '展示大数模幂运算和加密、解密、验签关系。',
    1, 1, 'rsa-encrypt', 'reactive', 1,
    '{"scene_code":"rsa-encrypt","algorithm_type":"rsa-encrypt","time_control":"reactive","step_duration":1000,"total_ticks":10}'::jsonb,
    '{"scene_code":"rsa-encrypt","actions":[{"action_code":"encrypt_plaintext","label":"重新加密","trigger":"form_submit","fields":[{"key":"plaintext","label":"明文","type":"string","required":true,"default_value":"42"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001016, 'Merkle 树构建验证', 'merkle-tree', 'cryptography',
    '展示叶子哈希、逐层合并和验证路径。',
    1, 1, 'merkle-tree', 'reactive', 1,
    '{"scene_code":"merkle-tree","algorithm_type":"merkle-tree","time_control":"reactive","step_duration":900,"total_ticks":16}'::jsonb,
    '{"scene_code":"merkle-tree","actions":[{"action_code":"tamper_leaf","label":"篡改叶子","trigger":"click","fields":[{"key":"leaf","label":"叶子索引","type":"number","required":true,"default_value":"0"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001017, '零知识证明原理', 'zkp-basic', 'cryptography',
    '展示承诺、挑战、响应三步交互。',
    1, 1, 'zkp-basic', 'process', 1,
    '{"scene_code":"zkp-basic","algorithm_type":"zkp-basic","time_control":"process","step_duration":1400,"total_ticks":12}'::jsonb,
    '{"scene_code":"zkp-basic","actions":[{"action_code":"change_secret","label":"更换秘密","trigger":"form_submit","fields":[{"key":"secret","label":"秘密值","type":"string","required":true,"default_value":"s1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 数据结构 (data_structure) — 5 个 ----
(
    920000000000001018, '区块链结构与分叉', 'blockchain-structure', 'data_structure',
    '展示主链、分叉链和最长链选择。',
    1, 1, 'blockchain-structure', 'reactive', 1,
    '{"scene_code":"blockchain-structure","algorithm_type":"blockchain-structure","time_control":"reactive","step_duration":900,"total_ticks":14}'::jsonb,
    '{"scene_code":"blockchain-structure","actions":[{"action_code":"append_block","label":"追加区块","trigger":"click","fields":[{"key":"branch","label":"目标分支","type":"string","required":true,"default_value":"main"}]},{"action_code":"fork_chain","label":"制造分叉","trigger":"form_submit","fields":[{"key":"height","label":"分叉高度","type":"number","required":true,"default_value":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001019, '区块内部结构', 'block-internal', 'data_structure',
    '展示区块头字段、交易体和 Merkle 根关系。',
    1, 1, 'block-internal', 'reactive', 1,
    '{"scene_code":"block-internal","algorithm_type":"block-internal","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"block-internal","actions":[{"action_code":"expand_field","label":"展开字段","trigger":"click","fields":[{"key":"field","label":"字段名","type":"string","required":true,"default_value":"header"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001020, '状态树（MPT）', 'mpt-trie', 'data_structure',
    '展示路径查找、节点展开和状态更新传播。',
    1, 1, 'mpt-trie', 'reactive', 1,
    '{"scene_code":"mpt-trie","algorithm_type":"mpt-trie","time_control":"reactive","step_duration":900,"total_ticks":12}'::jsonb,
    '{"scene_code":"mpt-trie","actions":[{"action_code":"update_account","label":"更新账户","trigger":"form_submit","fields":[{"key":"account","label":"账户键","type":"string","required":true,"default_value":"0xabc"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001021, '布隆过滤器', 'bloom-filter', 'data_structure',
    '展示位数组、多哈希映射和误判统计。',
    1, 1, 'bloom-filter', 'reactive', 1,
    '{"scene_code":"bloom-filter","algorithm_type":"bloom-filter","time_control":"reactive","step_duration":900,"total_ticks":8}'::jsonb,
    '{"scene_code":"bloom-filter","actions":[{"action_code":"query_key","label":"查询键","trigger":"form_submit","fields":[{"key":"key","label":"元素键","type":"string","required":true,"default_value":"alice"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001022, '分布式存储 DHT', 'dht-storage', 'data_structure',
    '展示环形空间、键值映射和分片分布。',
    1, 1, 'dht-storage', 'continuous', 1,
    '{"scene_code":"dht-storage","algorithm_type":"dht-storage","time_control":"continuous","step_duration":1200,"total_ticks":14}'::jsonb,
    '{"scene_code":"dht-storage","actions":[{"action_code":"store_key","label":"存储键值","trigger":"form_submit","fields":[{"key":"key","label":"键名","type":"string","required":true,"default_value":"doc-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 交易 (transaction) — 5 个 ----
(
    920000000000001023, '交易生命周期', 'tx-lifecycle', 'transaction',
    '展示创建、签名、广播、打包和确认全流程。',
    1, 1, 'tx-lifecycle', 'process', 1,
    '{"scene_code":"tx-lifecycle","algorithm_type":"tx-lifecycle","time_control":"process","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"tx-lifecycle","actions":[{"action_code":"create_tx","label":"创建交易","trigger":"form_submit","fields":[{"key":"tx","label":"交易标识","type":"string","required":true,"default_value":"tx-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001024, 'Gas 计算与优化', 'gas-calculation', 'transaction',
    '展示操作码 Gas 瀑布图和优化建议。',
    1, 1, 'gas-calculation', 'reactive', 1,
    '{"scene_code":"gas-calculation","algorithm_type":"gas-calculation","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"gas-calculation","actions":[{"action_code":"switch_opcode","label":"切换操作码","trigger":"click","fields":[{"key":"opcode","label":"操作码","type":"string","required":true,"default_value":"SSTORE"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001025, 'Token 转账流转', 'token-transfer', 'transaction',
    '展示账户余额变化和 ERC-20 事件日志。',
    1, 1, 'token-transfer', 'process', 1,
    '{"scene_code":"token-transfer","algorithm_type":"token-transfer","time_control":"process","step_duration":1300,"total_ticks":12}'::jsonb,
    '{"scene_code":"token-transfer","actions":[{"action_code":"transfer_token","label":"执行转账","trigger":"form_submit","fields":[{"key":"amount","label":"转账数量","type":"number","required":true,"default_value":"10"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001026, '交易排序与 MEV', 'tx-ordering-mev', 'transaction',
    '展示内存池排序、矿工优先级和三明治攻击。',
    1, 1, 'tx-ordering-mev', 'process', 1,
    '{"scene_code":"tx-ordering-mev","algorithm_type":"tx-ordering-mev","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"tx-ordering-mev","actions":[{"action_code":"inject_mev_bot","label":"注入 MEV Bot","trigger":"click","fields":[{"key":"bot","label":"机器人标识","type":"string","required":true,"default_value":"bot-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001027, '跨链桥接通信', 'cross-chain-bridge', 'transaction',
    '展示锁定、证明、中继和目标链铸造。',
    1, 1, 'cross-chain-bridge', 'process', 1,
    '{"scene_code":"cross-chain-bridge","algorithm_type":"cross-chain-bridge","time_control":"process","step_duration":1500,"total_ticks":16}'::jsonb,
    '{"scene_code":"cross-chain-bridge","actions":[{"action_code":"bridge_asset","label":"桥接资产","trigger":"form_submit","fields":[{"key":"asset","label":"资产名称","type":"string","required":true,"default_value":"USDT"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 智能合约 (smart_contract) — 5 个 ----
(
    920000000000001028, '智能合约状态机', 'contract-state-machine', 'smart_contract',
    '展示状态转换、事件触发和存储读写。',
    1, 1, 'contract-state-machine', 'reactive', 1,
    '{"scene_code":"contract-state-machine","algorithm_type":"contract-state-machine","time_control":"reactive","step_duration":1000,"total_ticks":10}'::jsonb,
    '{"scene_code":"contract-state-machine","actions":[{"action_code":"fire_event","label":"触发事件","trigger":"click","fields":[{"key":"event","label":"事件名","type":"string","required":true,"default_value":"activate"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001029, 'EVM 执行步进', 'evm-execution', 'smart_contract',
    '展示操作码执行、栈内存变化和存储写回。',
    1, 1, 'evm-execution', 'process', 1,
    '{"scene_code":"evm-execution","algorithm_type":"evm-execution","time_control":"process","step_duration":1300,"total_ticks":20}'::jsonb,
    '{"scene_code":"evm-execution","actions":[{"action_code":"step_opcode","label":"执行单步","trigger":"click","fields":[{"key":"opcode","label":"操作码","type":"string","required":true,"default_value":"SLOAD"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001030, '合约间调用', 'contract-interaction', 'smart_contract',
    '展示 call、delegatecall 和上下文切换。',
    1, 1, 'contract-interaction', 'process', 1,
    '{"scene_code":"contract-interaction","algorithm_type":"contract-interaction","time_control":"process","step_duration":1300,"total_ticks":14}'::jsonb,
    '{"scene_code":"contract-interaction","actions":[{"action_code":"invoke_delegatecall","label":"触发 delegatecall","trigger":"click","fields":[{"key":"resource_id","label":"目标合约","type":"string","required":true,"default_value":"Library"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001031, '合约部署流程', 'contract-deployment', 'smart_contract',
    '展示字节码、构造函数和部署地址推导。',
    1, 1, 'contract-deployment', 'process', 1,
    '{"scene_code":"contract-deployment","algorithm_type":"contract-deployment","time_control":"process","step_duration":1400,"total_ticks":12}'::jsonb,
    '{"scene_code":"contract-deployment","actions":[{"action_code":"redeploy_contract","label":"重新部署","trigger":"form_submit","fields":[{"key":"nonce","label":"部署 nonce","type":"number","required":true,"default_value":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001032, '状态通道', 'state-channel', 'smart_contract',
    '展示链上开通、链下更新、争议和关闭流程。',
    1, 1, 'state-channel', 'process', 1,
    '{"scene_code":"state-channel","algorithm_type":"state-channel","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"state-channel","actions":[{"action_code":"submit_dispute","label":"提交争议","trigger":"click","fields":[{"key":"proof","label":"争议证明","type":"string","required":true,"default_value":"proof-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 攻击与安全 (attack_security) — 6 个 ----
(
    920000000000001033, '51% 算力攻击', '51-percent-attack', 'attack_security',
    '展示诚实链与攻击链竞赛和链重组。',
    1, 1, '51-percent-attack', 'process', 1,
    '{"scene_code":"51-percent-attack","algorithm_type":"51-percent-attack","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"51-percent-attack","actions":[{"action_code":"boost_attacker_hashrate","label":"提升攻击算力","trigger":"form_submit","fields":[{"key":"ratio","label":"算力比例","type":"range","required":true,"default_value":"0.55"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001034, '双花攻击', 'double-spend', 'attack_security',
    '展示冲突交易、确认数竞赛和商家风险。',
    1, 1, 'double-spend', 'process', 1,
    '{"scene_code":"double-spend","algorithm_type":"double-spend","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"double-spend","actions":[{"action_code":"send_conflict_tx","label":"发送冲突交易","trigger":"click","fields":[{"key":"tx","label":"交易标识","type":"string","required":true,"default_value":"conflict-tx"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001035, 'PBFT 拜占庭攻击', 'pbft-byzantine', 'attack_security',
    '展示异常副本、消息偏差和容错阈值变化。',
    1, 1, 'pbft-byzantine', 'process', 1,
    '{"scene_code":"pbft-byzantine","algorithm_type":"pbft-byzantine","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"pbft-byzantine","actions":[{"action_code":"forge_vote","label":"伪造投票","trigger":"form_submit","fields":[{"key":"resource_id","label":"副本标识","type":"node_ref","required":true,"default_value":"Replica-2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001036, '重入攻击', 'reentrancy-attack', 'attack_security',
    '展示递归调用栈和余额被盗过程。',
    1, 1, 'reentrancy-attack', 'process', 1,
    '{"scene_code":"reentrancy-attack","algorithm_type":"reentrancy-attack","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"reentrancy-attack","actions":[{"action_code":"trigger_reentrancy","label":"触发重入","trigger":"click","fields":[{"key":"depth","label":"重入深度","type":"number","required":true,"default_value":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001037, '整数溢出攻击', 'integer-overflow', 'attack_security',
    '展示数值回绕、临界点和 SafeMath 对比。',
    1, 1, 'integer-overflow', 'reactive', 1,
    '{"scene_code":"integer-overflow","algorithm_type":"integer-overflow","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"integer-overflow","actions":[{"action_code":"add_value","label":"增加数值","trigger":"form_submit","fields":[{"key":"delta","label":"增量","type":"number","required":true,"default_value":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001038, '自私挖矿', 'selfish-mining', 'attack_security',
    '展示私有链积累、公开策略和收益对比。',
    1, 1, 'selfish-mining', 'process', 1,
    '{"scene_code":"selfish-mining","algorithm_type":"selfish-mining","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"selfish-mining","actions":[{"action_code":"publish_private_chain","label":"公开私链","trigger":"click","fields":[{"key":"blocks","label":"区块数量","type":"number","required":true,"default_value":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 经济模型 (economic) — 5 个 ----
(
    920000000000001039, 'Token 经济模型', 'token-economics', 'economic',
    '展示供应量曲线、分配比例和释放节奏。',
    1, 1, 'token-economics', 'reactive', 1,
    '{"scene_code":"token-economics","algorithm_type":"token-economics","time_control":"reactive","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"token-economics","actions":[{"action_code":"switch_release_model","label":"切换释放模型","trigger":"form_submit","fields":[{"key":"inflation_rate","label":"年化通胀率","type":"number","required":true,"default_value":"0.04"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001040, 'PoS 质押经济', 'pos-staking', 'economic',
    '展示质押量、收益率和 Slash 事件。',
    1, 1, 'pos-staking', 'continuous', 1,
    '{"scene_code":"pos-staking","algorithm_type":"pos-staking","time_control":"continuous","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"pos-staking","actions":[{"action_code":"apply_slash","label":"执行 Slash","trigger":"click","fields":[{"key":"resource_id","label":"验证者标识","type":"node_ref","required":true,"default_value":"Validator-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001041, '链上治理投票', 'governance-voting', 'economic',
    '展示提案生命周期、投票权重和法定人数。',
    1, 1, 'governance-voting', 'process', 1,
    '{"scene_code":"governance-voting","algorithm_type":"governance-voting","time_control":"process","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"governance-voting","actions":[{"action_code":"cast_vote","label":"投票","trigger":"form_submit","fields":[{"key":"choice","label":"投票选项","type":"string","required":true,"default_value":"yes"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001042, 'DeFi 流动性池', 'defi-liquidity', 'economic',
    '展示 AMM 曲线、滑点和无常损失。',
    1, 1, 'defi-liquidity', 'reactive', 1,
    '{"scene_code":"defi-liquidity","algorithm_type":"defi-liquidity","time_control":"reactive","step_duration":900,"total_ticks":12}'::jsonb,
    '{"scene_code":"defi-liquidity","actions":[{"action_code":"swap_asset","label":"执行兑换","trigger":"form_submit","fields":[{"key":"amount","label":"兑换数量","type":"number","required":true,"default_value":"20"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001043, 'Gas 费市场（EIP-1559）', 'gas-market', 'economic',
    '展示基础费、区块利用率和燃烧量变化。',
    1, 1, 'gas-market', 'continuous', 1,
    '{"scene_code":"gas-market","algorithm_type":"gas-market","time_control":"continuous","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"gas-market","actions":[{"action_code":"spike_demand","label":"提升需求","trigger":"click","fields":[{"key":"demand","label":"需求倍率","type":"number","required":true,"default_value":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

-- =====================================================================
-- 02. 联动组 — 7 组
-- =====================================================================

INSERT INTO sim_link_groups (id, name, code, description, shared_state_schema, created_at, updated_at)
VALUES
(920000000000002001, 'PoW 攻击联动组', 'pow-attack-group',
 'PoW 挖矿、51% 攻击、区块同步、区块链结构之间共享算力与链状态。',
 '{"type":"object","properties":{"total_hashrate":{"type":"number"},"honest_hashrate":{"type":"number"},"attacker_hashrate":{"type":"number"},"chain_height":{"type":"integer"},"fork_height":{"type":"integer"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002002, 'PBFT 攻击联动组', 'pbft-attack-group',
 'PBFT 共识、拜占庭攻击、网络分区之间共享副本状态与视图编号。',
 '{"type":"object","properties":{"view_number":{"type":"integer"},"byzantine_nodes":{"type":"array","items":{"type":"string"}},"partition_active":{"type":"boolean"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002003, 'Raft 容错联动组', 'raft-fault-group',
 'Raft 选举、网络分区、区块同步之间共享 Leader 与任期状态。',
 '{"type":"object","properties":{"leader":{"type":"string"},"term":{"type":"integer"},"partition_active":{"type":"boolean"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002004, '网络基础联动组', 'network-base-group',
 'P2P 发现、Gossip 传播、负载均衡之间共享节点拓扑。',
 '{"type":"object","properties":{"peer_count":{"type":"integer"},"topology":{"type":"object"},"coverage_ratio":{"type":"number"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002005, '密码学验证联动组', 'crypto-verify-group',
 'SHA-256、ECDSA、Merkle 树之间共享哈希与签名状态。',
 '{"type":"object","properties":{"current_hash":{"type":"string"},"signature":{"type":"string"},"verified":{"type":"boolean"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002006, '区块链完整性联动组', 'blockchain-integrity-group',
 '区块链结构、区块内部结构、Merkle 树、交易生命周期之间共享链数据完整性状态。',
 '{"type":"object","properties":{"merkle_root":{"type":"string"},"block_hash":{"type":"string"},"chain_valid":{"type":"boolean"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002007, '交易处理联动组', 'tx-processing-group',
 'Gas 计算、Token 转账、MEV、交易生命周期之间共享交易与 Gas 状态。',
 '{"type":"object","properties":{"gas_price":{"type":"number"},"mempool_size":{"type":"integer"},"pending_txs":{"type":"array","items":{"type":"string"}}}}'::jsonb,
 NOW(), NOW()),
(920000000000002008, 'PoS 经济联动组', 'pos-economy-group',
 'PoS 验证者、质押经济、Token 经济、治理投票之间共享质押与供应量状态。',
 '{"type":"object","properties":{"total_stake":{"type":"number"},"inflation_rate":{"type":"number"},"active_validators":{"type":"integer"}}}'::jsonb,
 NOW(), NOW()),
(920000000000002009, '合约安全联动组', 'contract-security-group',
 '状态机、EVM 执行、重入攻击、整数溢出之间共享合约运行时状态。',
 '{"type":"object","properties":{"current_state":{"type":"string"},"call_depth":{"type":"integer"},"balance":{"type":"number"}}}'::jsonb,
 NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- =====================================================================
-- 03. 联动组场景关联
-- =====================================================================

INSERT INTO sim_link_group_scenes (id, link_group_id, scenario_id, role_in_group, sort_order, created_at)
VALUES
-- pow-attack-group: pow-mining, 51-percent-attack, block-sync, blockchain-structure
(920000000000003001, 920000000000002001, 920000000000001007, 'mining', 1, NOW()),
(920000000000003002, 920000000000002001, 920000000000001033, 'attack', 2, NOW()),
(920000000000003003, 920000000000002001, 920000000000001005, 'sync', 3, NOW()),
(920000000000003004, 920000000000002001, 920000000000001018, 'structure', 4, NOW()),
-- pbft-attack-group: pbft-consensus, pbft-byzantine, network-partition
(920000000000003005, 920000000000002002, 920000000000001009, 'consensus', 1, NOW()),
(920000000000003006, 920000000000002002, 920000000000001035, 'attack', 2, NOW()),
(920000000000003007, 920000000000002002, 920000000000001003, 'partition', 3, NOW()),
-- raft-fault-group: raft-election, network-partition, block-sync
(920000000000003008, 920000000000002003, 920000000000001010, 'leader', 1, NOW()),
(920000000000003009, 920000000000002003, 920000000000001003, 'partition', 2, NOW()),
(920000000000003010, 920000000000002003, 920000000000001005, 'sync', 3, NOW()),
-- network-base-group: p2p-discovery, gossip-propagation, node-load-balance
(920000000000003011, 920000000000002004, 920000000000001001, 'discovery', 1, NOW()),
(920000000000003012, 920000000000002004, 920000000000001002, 'propagation', 2, NOW()),
(920000000000003013, 920000000000002004, 920000000000001006, 'balance', 3, NOW()),
-- crypto-verify-group: sha256-hash, ecdsa-sign, merkle-tree
(920000000000003014, 920000000000002005, 920000000000001012, 'hash', 1, NOW()),
(920000000000003015, 920000000000002005, 920000000000001014, 'sign', 2, NOW()),
(920000000000003016, 920000000000002005, 920000000000001016, 'tree', 3, NOW()),
-- blockchain-integrity-group: blockchain-structure, block-internal, merkle-tree, tx-lifecycle
(920000000000003017, 920000000000002006, 920000000000001018, 'chain', 1, NOW()),
(920000000000003018, 920000000000002006, 920000000000001019, 'block', 2, NOW()),
(920000000000003019, 920000000000002006, 920000000000001016, 'merkle', 3, NOW()),
(920000000000003020, 920000000000002006, 920000000000001023, 'tx', 4, NOW()),
-- tx-processing-group: gas-calculation, token-transfer, tx-ordering-mev, tx-lifecycle
(920000000000003021, 920000000000002007, 920000000000001024, 'gas', 1, NOW()),
(920000000000003022, 920000000000002007, 920000000000001025, 'transfer', 2, NOW()),
(920000000000003023, 920000000000002007, 920000000000001026, 'mev', 3, NOW()),
(920000000000003024, 920000000000002007, 920000000000001023, 'lifecycle', 4, NOW()),
-- pos-economy-group: pos-validator, pos-staking, token-economics, governance-voting
(920000000000003025, 920000000000002008, 920000000000001008, 'validator', 1, NOW()),
(920000000000003026, 920000000000002008, 920000000000001040, 'staking', 2, NOW()),
(920000000000003027, 920000000000002008, 920000000000001039, 'tokenomics', 3, NOW()),
(920000000000003028, 920000000000002008, 920000000000001041, 'governance', 4, NOW()),
-- contract-security-group: contract-state-machine, evm-execution, reentrancy-attack, integer-overflow
(920000000000003029, 920000000000002009, 920000000000001028, 'state', 1, NOW()),
(920000000000003030, 920000000000002009, 920000000000001029, 'evm', 2, NOW()),
(920000000000003031, 920000000000002009, 920000000000001036, 'reentrancy', 3, NOW()),
(920000000000003032, 920000000000002009, 920000000000001037, 'overflow', 4, NOW())
ON CONFLICT (link_group_id, scenario_id) DO NOTHING;

-- =====================================================================
-- 04. 纯仿真实验模板 (experiment_type=1) — 4 个
-- =====================================================================
-- school_id=910000000000000001 (学校A), teacher_id=910000000000001001 (教师)

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, is_shared, status, created_at, updated_at
)
VALUES
(
    920000000000008001,
    910000000000000001,
    910000000000001001,
    '共识机制可视化对比实验',
    '通过可视化仿真对比 PoW、PoS、PBFT、Raft 四种共识机制的工作原理与性能差异。',
    '理解主流共识算法的运行过程，对比不同共识在吞吐量、延迟、容错性方面的差异。',
    '1. 进入仿真面板，观察 PoW 挖矿竞争动画\n2. 切换到 PoS 验证者选举场景，对比出块方式\n3. 在 PBFT 场景中注入拜占庭节点，观察容错\n4. 总结四种共识机制的优劣',
    1, NULL, 1, 100, 45, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008002,
    910000000000000001,
    910000000000001001,
    '密码学基础可视化实验',
    '通过可视化仿真理解哈希、签名、Merkle 树和零知识证明的工作原理。',
    '掌握 SHA-256 雪崩效应、ECDSA 签名验签流程、Merkle 树验证路径。',
    '1. 在 SHA-256 场景中修改输入，观察雪崩效应\n2. 在 ECDSA 场景中完成签名和验签\n3. 在 Merkle 树场景中篡改叶子，观察验证路径变化\n4. 完成检查点验证',
    1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008003,
    910000000000000001,
    910000000000001001,
    '交易与 Gas 机制仿真实验',
    '通过可视化仿真理解交易生命周期、Gas 计算和 MEV 现象。',
    '理解交易从创建到确认的完整流程，掌握 Gas 机制和 MEV 对交易排序的影响。',
    '1. 在交易生命周期场景中观察一笔交易的完整旅程\n2. 在 Gas 计算场景中对比不同操作码的消耗\n3. 在 MEV 场景中注入抢跑机器人，观察三明治攻击\n4. 完成检查点',
    1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008004,
    910000000000000002,
    910000000000001202,
    '区块链攻防安全仿真实验',
    '通过可视化仿真理解 51% 攻击、双花、重入攻击等安全威胁。',
    '理解各种攻击向量的原理与防御措施，培养安全意识。',
    '1. 在 51% 攻击场景中调整算力比例，观察链重组\n2. 在重入攻击场景中触发递归调用，观察余额被盗\n3. 在整数溢出场景中增加数值到临界点\n4. 总结防御方案',
    1, NULL, 1, 100, 50, 30, 1, TRUE, 2, NOW(), NOW()
)
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 05. 混合实验模板 (experiment_type=3) — 2 个
-- =====================================================================

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, is_shared, status, created_at, updated_at
)
VALUES
(
    920000000000008005,
    910000000000000001,
    910000000000001001,
    'EVM 合约开发与执行可视化混合实验',
    '在真实 EVM 开发环境中编写合约，同时通过仿真面板观察 EVM 执行步进和合约状态变化。',
    '将真实合约开发与 EVM 内部原理可视化结合，深入理解编译-部署-执行全链路。',
    '1. 在 Remix IDE 中编写一个简单的存储合约\n2. 编译并部署到本地 geth 节点\n3. 切换到仿真面板，在 EVM 执行步进场景中跟踪操作码\n4. 在合约状态机场景中观察状态变迁\n5. 完成检查点验证',
    3, 1, 1, 100, 90, 30, 1, FALSE, 2, NOW(), NOW()
),
(
    920000000000008006,
    910000000000000001,
    910000000000001001,
    'PoW 挖矿与链观察混合实验',
    '在真实 geth 节点上进行挖矿操作，同时通过仿真面板可视化观察 PoW 过程和区块同步。',
    '将真实链节点操作与仿真可视化结合，理解挖矿竞争和区块传播的实际过程。',
    '1. 启动 geth 开发节点，开启挖矿\n2. 通过区块浏览器观察出块情况\n3. 切换到仿真面板，在 PoW 挖矿场景中观察 Nonce 搜索动画\n4. 在区块同步场景中观察新区块传播\n5. 对比真实链数据与仿真数据',
    3, 1, 1, 100, 90, 30, 1, FALSE, 2, NOW(), NOW()
)
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 06. 纯仿真模板的 template_sim_scenes（场景关联）
-- =====================================================================

INSERT INTO template_sim_scenes (id, template_id, scenario_id, link_group_id, config, layout_position, sort_order, created_at, updated_at)
VALUES
-- 共识机制可视化对比实验（8001）→ 4 个共识场景
(920000000000009001, 920000000000008001, 920000000000001007, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009002, 920000000000008001, 920000000000001008, 920000000000002008,
 '{"scene_params":{"link_group_code":"pos-economy-group"}}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),
(920000000000009003, 920000000000008001, 920000000000001009, 920000000000002002,
 '{"scene_params":{"link_group_code":"pbft-attack-group"}}'::jsonb,
 '{"row":1,"col":0}'::jsonb, 3, NOW(), NOW()),
(920000000000009004, 920000000000008001, 920000000000001010, 920000000000002003,
 '{"scene_params":{"link_group_code":"raft-fault-group"}}'::jsonb,
 '{"row":1,"col":1}'::jsonb, 4, NOW(), NOW()),

-- 密码学基础可视化实验（8002）→ 4 个密码学场景
(920000000000009005, 920000000000008002, 920000000000001012, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009006, 920000000000008002, 920000000000001014, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),
(920000000000009007, 920000000000008002, 920000000000001016, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":1,"col":0}'::jsonb, 3, NOW(), NOW()),
(920000000000009008, 920000000000008002, 920000000000001017, NULL,
 '{}'::jsonb,
 '{"row":1,"col":1}'::jsonb, 4, NOW(), NOW()),

-- 交易与 Gas 机制仿真实验（8003）→ 3 个交易场景
(920000000000009009, 920000000000008003, 920000000000001023, 920000000000002007,
 '{"scene_params":{"link_group_code":"tx-processing-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009010, 920000000000008003, 920000000000001024, 920000000000002007,
 '{"scene_params":{"link_group_code":"tx-processing-group"}}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),
(920000000000009011, 920000000000008003, 920000000000001026, 920000000000002007,
 '{"scene_params":{"link_group_code":"tx-processing-group"}}'::jsonb,
 '{"row":1,"col":0}'::jsonb, 3, NOW(), NOW()),

-- 区块链攻防安全仿真实验（8004）→ 4 个安全场景
(920000000000009012, 920000000000008004, 920000000000001033, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009013, 920000000000008004, 920000000000001034, NULL,
 '{}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),
(920000000000009014, 920000000000008004, 920000000000001036, 920000000000002009,
 '{"scene_params":{"link_group_code":"contract-security-group"}}'::jsonb,
 '{"row":1,"col":0}'::jsonb, 3, NOW(), NOW()),
(920000000000009015, 920000000000008004, 920000000000001037, 920000000000002009,
 '{"scene_params":{"link_group_code":"contract-security-group"}}'::jsonb,
 '{"row":1,"col":1}'::jsonb, 4, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 07. 混合模板的 template_sim_scenes
-- =====================================================================

INSERT INTO template_sim_scenes (id, template_id, scenario_id, link_group_id, config, layout_position, sort_order, created_at, updated_at)
VALUES
-- EVM 合约开发混合实验（8005）→ 2 个仿真场景
(920000000000009016, 920000000000008005, 920000000000001029, 920000000000002009,
 '{"scene_params":{"link_group_code":"contract-security-group"},"data_source_mode":"dual"}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009017, 920000000000008005, 920000000000001028, 920000000000002009,
 '{"scene_params":{"link_group_code":"contract-security-group"},"data_source_mode":"dual"}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),

-- PoW 挖矿混合实验（8006）→ 2 个仿真场景
(920000000000009018, 920000000000008006, 920000000000001007, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"},"data_source_mode":"dual"}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009019, 920000000000008006, 920000000000001005, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"},"data_source_mode":"dual"}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 08. 混合模板的容器配置
-- =====================================================================
-- 混合实验 8005 (EVM 合约+仿真) 使用 geth + remix-ide 两个容器
-- 引用 012 中已有的 image_version_id

INSERT INTO template_containers (
    id, template_id, image_version_id, container_name,
    deployment_scope, env_vars, ports, volumes,
    cpu_limit, memory_limit, depends_on, startup_order,
    is_primary, created_at, updated_at
)
VALUES
-- 混合实验 8005: geth + remix-ide
(
    920000000000009101, 920000000000008005, 910000000000006001,
    'geth', 1,
    '[]'::jsonb,
    '[{"port":8545,"protocol":"tcp","name":"HTTP-RPC"},{"port":30303,"protocol":"tcp","name":"P2P"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '[]'::jsonb, 1, FALSE, NOW(), NOW()
),
(
    920000000000009102, 920000000000008005, 910000000000006012,
    'remix-ide', 1,
    '[{"key":"REMIX_URL","value":"http://geth:8545","desc":"RPC 地址"}]'::jsonb,
    '[{"port":8080,"protocol":"tcp","name":"Web UI"}]'::jsonb,
    '[]'::jsonb,
    '300m', '512Mi', '["geth"]'::jsonb, 2, TRUE, NOW(), NOW()
),
-- 混合实验 8006: geth + blockscout
(
    920000000000009103, 920000000000008006, 910000000000006001,
    'geth', 1,
    '[]'::jsonb,
    '[{"port":8545,"protocol":"tcp","name":"HTTP-RPC"},{"port":30303,"protocol":"tcp","name":"P2P"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '[]'::jsonb, 1, FALSE, NOW(), NOW()
),
(
    920000000000009104, 920000000000008006, 910000000000006003,
    'blockscout', 1,
    '[{"key":"ETHEREUM_JSONRPC_HTTP_URL","value":"http://geth:8545","desc":"EVM 节点 RPC 地址"}]'::jsonb,
    '[{"port":4000,"protocol":"tcp","name":"Web UI"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '["geth"]'::jsonb, 2, TRUE, NOW(), NOW()
)
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 09. 检查点
-- =====================================================================

INSERT INTO template_checkpoints (
    id, template_id, title, description,
    check_type, assertion_config, score, sort_order,
    created_at, updated_at
)
VALUES
-- 共识机制可视化对比（8001）— check_type: 3=SimEngine状态断言
(920000000000010001, 920000000000008001, '观察 PoW 出块', '在 PoW 场景中推进至少 10 个 Tick，观察一次成功出块。',
 3, '{"scene_code":"pow-mining","condition":"tick >= 10"}'::jsonb, 25, 1, NOW(), NOW()),
(920000000000010002, 920000000000008001, '注入拜占庭节点', '在 PBFT 场景中触发注入拜占庭节点交互。',
 3, '{"scene_code":"pbft-consensus","action_code":"inject_byzantine_node"}'::jsonb, 25, 2, NOW(), NOW()),
(920000000000010003, 920000000000008001, '触发 Raft Leader 故障', '在 Raft 场景中让 Leader 宕机并观察重新选举。',
 3, '{"scene_code":"raft-election","action_code":"fail_leader"}'::jsonb, 25, 3, NOW(), NOW()),
(920000000000010004, 920000000000008001, '完成四种共识对比', '将四个共识场景都推进至结束状态。',
 3, '{"all_scenes_completed":true}'::jsonb, 25, 4, NOW(), NOW()),

-- 密码学基础（8002）
(920000000000010005, 920000000000008002, '观察雪崩效应', '在 SHA-256 场景中修改输入，对比哈希变化。',
 3, '{"scene_code":"sha256-hash","action_code":"mutate_input"}'::jsonb, 25, 1, NOW(), NOW()),
(920000000000010006, 920000000000008002, '完成 ECDSA 签名', '在 ECDSA 场景中完成一次签名和验签流程。',
 3, '{"scene_code":"ecdsa-sign","condition":"tick >= 12"}'::jsonb, 25, 2, NOW(), NOW()),
(920000000000010007, 920000000000008002, '篡改 Merkle 叶子', '篡改一个叶子并观察验证路径失效。',
 3, '{"scene_code":"merkle-tree","action_code":"tamper_leaf"}'::jsonb, 25, 3, NOW(), NOW()),
(920000000000010008, 920000000000008002, '完成零知识证明交互', '在 ZKP 场景中完成承诺-挑战-响应全流程。',
 3, '{"scene_code":"zkp-basic","condition":"tick >= 12"}'::jsonb, 25, 4, NOW(), NOW()),

-- 交易与 Gas（8003）
(920000000000010009, 920000000000008003, '追踪交易生命周期', '在交易生命周期场景中创建交易并观察完整流程。',
 3, '{"scene_code":"tx-lifecycle","action_code":"create_tx"}'::jsonb, 34, 1, NOW(), NOW()),
(920000000000010010, 920000000000008003, '分析 Gas 消耗', '在 Gas 计算场景中切换操作码对比消耗差异。',
 3, '{"scene_code":"gas-calculation","action_code":"switch_opcode"}'::jsonb, 33, 2, NOW(), NOW()),
(920000000000010011, 920000000000008003, '观察 MEV 攻击', '在 MEV 场景中注入抢跑机器人并观察排序变化。',
 3, '{"scene_code":"tx-ordering-mev","action_code":"inject_mev_bot"}'::jsonb, 33, 3, NOW(), NOW()),

-- 攻防安全（8004）
(920000000000010012, 920000000000008004, '执行 51% 攻击', '提升攻击者算力至 55%，观察链重组。',
 3, '{"scene_code":"51-percent-attack","action_code":"boost_attacker_hashrate"}'::jsonb, 25, 1, NOW(), NOW()),
(920000000000010013, 920000000000008004, '触发双花攻击', '发送冲突交易，观察商家交易被替换。',
 3, '{"scene_code":"double-spend","action_code":"send_conflict_tx"}'::jsonb, 25, 2, NOW(), NOW()),
(920000000000010014, 920000000000008004, '触发重入攻击', '执行重入攻击并观察余额被清空。',
 3, '{"scene_code":"reentrancy-attack","action_code":"trigger_reentrancy"}'::jsonb, 25, 3, NOW(), NOW()),
(920000000000010015, 920000000000008004, '观察整数溢出', '增加数值到临界点并观察回绕现象。',
 3, '{"scene_code":"integer-overflow","action_code":"add_value"}'::jsonb, 25, 4, NOW(), NOW()),

-- EVM 混合实验（8005）— check_type: 1=脚本验证, 3=SimEngine断言
(920000000000010016, 920000000000008005, '部署合约到 geth', '在 Remix IDE 中编译并部署合约到本地 geth 节点。',
 1, '{"target_container":"remix-ide","command":"curl -s http://geth:8545 | jq .result","expected":"0x"}'::jsonb, 40, 1, NOW(), NOW()),
(920000000000010017, 920000000000008005, '跟踪 EVM 执行', '在仿真面板的 EVM 执行步进场景中至少推进 10 步。',
 3, '{"scene_code":"evm-execution","condition":"tick >= 10"}'::jsonb, 30, 2, NOW(), NOW()),
(920000000000010018, 920000000000008005, '观察合约状态变化', '在合约状态机场景中触发至少一次状态迁移。',
 3, '{"scene_code":"contract-state-machine","action_code":"fire_event"}'::jsonb, 30, 3, NOW(), NOW()),

-- PoW 混合实验（8006）
(920000000000010019, 920000000000008006, '启动 geth 挖矿', '连接 geth 节点并确认出块。',
 1, '{"target_container":"blockscout","command":"curl -s http://geth:8545 -X POST -H \"Content-Type:application/json\" -d ''{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}''","expected":"0x"}'::jsonb, 30, 1, NOW(), NOW()),
(920000000000010020, 920000000000008006, '观察 PoW 仿真', '在 PoW 挖矿仿真场景中推进至少 15 个 Tick。',
 3, '{"scene_code":"pow-mining","condition":"tick >= 15"}'::jsonb, 35, 2, NOW(), NOW()),
(920000000000010021, 920000000000008006, '对比真实与仿真区块同步', '在区块同步场景中观察至少一次新区块传播。',
 3, '{"scene_code":"block-sync","condition":"tick >= 10"}'::jsonb, 35, 3, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 10. 标签
-- =====================================================================

INSERT INTO tags (id, name, category, color, is_system, created_at)
VALUES
(920000000000011001, '纯仿真', 'experiment_type', '#8B5CF6', TRUE, NOW()),
(920000000000011002, '混合实验', 'experiment_type', '#F59E0B', TRUE, NOW()),
(920000000000011003, '共识机制', 'topic', '#3B82F6', TRUE, NOW()),
(920000000000011004, '密码学', 'topic', '#10B981', TRUE, NOW()),
(920000000000011005, '交易机制', 'topic', '#6366F1', TRUE, NOW()),
(920000000000011006, '攻防安全', 'topic', '#EF4444', TRUE, NOW()),
(920000000000011007, 'EVM', 'ecosystem', '#627EEA', TRUE, NOW()),
(920000000000011008, '可视化', 'feature', '#06B6D4', TRUE, NOW())
ON CONFLICT (name, category) DO NOTHING;

INSERT INTO template_tags (id, template_id, tag_id, created_at)
VALUES
-- 共识机制可视化实验
(920000000000012001, 920000000000008001, 920000000000011001, NOW()),
(920000000000012002, 920000000000008001, 920000000000011003, NOW()),
(920000000000012003, 920000000000008001, 920000000000011008, NOW()),
-- 密码学基础实验
(920000000000012004, 920000000000008002, 920000000000011001, NOW()),
(920000000000012005, 920000000000008002, 920000000000011004, NOW()),
(920000000000012006, 920000000000008002, 920000000000011008, NOW()),
-- 交易与 Gas 实验
(920000000000012007, 920000000000008003, 920000000000011001, NOW()),
(920000000000012008, 920000000000008003, 920000000000011005, NOW()),
(920000000000012009, 920000000000008003, 920000000000011008, NOW()),
-- 攻防安全实验
(920000000000012010, 920000000000008004, 920000000000011001, NOW()),
(920000000000012011, 920000000000008004, 920000000000011006, NOW()),
(920000000000012012, 920000000000008004, 920000000000011008, NOW()),
-- EVM 混合实验
(920000000000012013, 920000000000008005, 920000000000011002, NOW()),
(920000000000012014, 920000000000008005, 920000000000011007, NOW()),
(920000000000012015, 920000000000008005, 920000000000011008, NOW()),
-- PoW 混合实验
(920000000000012016, 920000000000008006, 920000000000011002, NOW()),
(920000000000012017, 920000000000008006, 920000000000011003, NOW()),
(920000000000012018, 920000000000008006, 920000000000011008, NOW())
ON CONFLICT (template_id, tag_id) DO NOTHING;

-- =====================================================================
-- 11. 课程章节、课时与实验关联
-- =====================================================================
-- 给 010 中已有的课程 (910000000000007001) 添加仿真实验章节

INSERT INTO chapters (id, course_id, title, description, sort_order, created_at, updated_at)
VALUES
(920000000000013001, 910000000000007001, '第四章 区块链原理可视化', '通过仿真实验理解区块链核心原理。', 4, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- content_type=4 实验课时，experiment_id 指向 experiment_templates
INSERT INTO lessons (id, chapter_id, course_id, title, content_type, experiment_id, sort_order, estimated_minutes, created_at, updated_at)
VALUES
(920000000000013101, 920000000000013001, 910000000000007001, '4.1 共识机制仿真', 4, 920000000000008001, 1, 45, NOW(), NOW()),
(920000000000013102, 920000000000013001, 910000000000007001, '4.2 密码学基础仿真', 4, 920000000000008002, 2, 40, NOW(), NOW()),
(920000000000013103, 920000000000013001, 910000000000007001, '4.3 交易与 Gas 仿真', 4, 920000000000008003, 3, 40, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO course_experiments (id, course_id, experiment_id, title, sort_order, created_at)
VALUES
(920000000000014001, 910000000000007001, 920000000000008001, '共识机制可视化对比', 1, NOW()),
(920000000000014002, 910000000000007001, 920000000000008002, '密码学基础可视化', 2, NOW()),
(920000000000014003, 910000000000007001, 920000000000008003, '交易与Gas机制仿真', 3, NOW())
ON CONFLICT (id) DO NOTHING;
