-- 013_seed_sim_scenarios.up.sql
-- 补充仿真场景库、联动组与纯仿真/混合实验模板种子数据。
-- 目标：
-- 1. 添加 43 个内置仿真场景（8 大领域），与 sim-engine/scenarios 目录一一对应
-- 2. 添加 9 个内置联动组及其场景关联（教师扩展场景可申请加入或新建联动组，详见 06 文档 §11）
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
--   current_version, created_at, updated_at

INSERT INTO sim_scenarios (
    id, name, code, category, description,
    source_type, status,
    algorithm_type, time_control_mode, data_source_mode,
    default_params, interaction_schema, default_size, delivery_phase,
    current_version, created_at, updated_at
)
VALUES
-- ---- 节点与网络 (node_network) — 6 个 ----
(
    920000000000001001, 'P2P 网络发现与路由', 'p2p-discovery', 'node_network',
    '展示节点发现、邻居路由表收敛和拓扑稳定过程。',
    1, 1, 'p2p-discovery', 'continuous', 1,
    '{"scene_code":"p2p-discovery","algorithm_type":"p2p-discovery","time_control":"continuous","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"p2p-discovery","schema_version":"1.0","actions":[{"action_code":"add_peer","label":"新增节点","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"peer_label","label":"节点名称","type":"string","required":true,"default":"Peer-X"}]},{"action_code":"shuffle_route","label":"扰动路由","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"目标节点","type":"string","required":true,"default":"Peer-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001002, 'Gossip 消息传播', 'gossip-propagation', 'node_network',
    '展示消息从源节点向全网扩散的覆盖过程。',
    1, 1, 'gossip-propagation', 'continuous', 1,
    '{"scene_code":"gossip-propagation","algorithm_type":"gossip-propagation","time_control":"continuous","step_duration":1200,"total_ticks":15}'::jsonb,
    '{"scene_code":"gossip-propagation","schema_version":"1.0","actions":[{"action_code":"seed_message","label":"投放消息","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"message_label","label":"消息标签","type":"string","required":true,"default":"TX-001"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001003, '网络分区与恢复', 'network-partition', 'node_network',
    '展示网络隔离边界形成和恢复同步流程。',
    1, 1, 'network-partition', 'process', 1,
    '{"scene_code":"network-partition","algorithm_type":"network-partition","time_control":"process","step_duration":1600,"total_ticks":20}'::jsonb,
    '{"scene_code":"network-partition","schema_version":"1.0","actions":[{"action_code":"cut_link","label":"切断链路","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"目标链路","type":"string","required":true,"default":"A-B"}]},{"action_code":"restore_link","label":"恢复链路","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"目标链路","type":"string","required":true,"default":"A-B"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001004, '交易广播与打包', 'tx-broadcast', 'node_network',
    '展示交易广播、矿工收集和内存池填充。',
    1, 1, 'tx-broadcast', 'process', 1,
    '{"scene_code":"tx-broadcast","algorithm_type":"tx-broadcast","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"tx-broadcast","schema_version":"1.0","actions":[{"action_code":"broadcast_tx","label":"发起交易","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"tx_label","label":"交易标识","type":"string","required":true,"default":"tx-1001"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001005, '区块同步与传播', 'block-sync', 'node_network',
    '展示新区块传播、高度追平和分叉检测。',
    1, 1, 'block-sync', 'process', 1,
    '{"scene_code":"block-sync","algorithm_type":"block-sync","time_control":"process","step_duration":1500,"total_ticks":20}'::jsonb,
    '{"scene_code":"block-sync","schema_version":"1.0","actions":[{"action_code":"inject_block","label":"注入新区块","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"block_height","label":"区块高度","type":"number","required":true,"default":"10"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001006, '节点负载均衡', 'node-load-balance', 'node_network',
    '展示网络请求在节点间的路由与迁移。',
    1, 1, 'node-load-balance', 'continuous', 1,
    '{"scene_code":"node-load-balance","algorithm_type":"node-load-balance","time_control":"continuous","step_duration":1200,"total_ticks":14}'::jsonb,
    '{"scene_code":"node-load-balance","schema_version":"1.0","actions":[{"action_code":"shift_traffic","label":"迁移流量","category":"param_tune","trigger":"hold","roles":["student"],"fields":[{"name":"resource_id","label":"目标节点","type":"string","required":true,"default":"LB-2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 共识机制 (consensus) — 5 个 ----
(
    920000000000001007, 'PoW 挖矿竞争', 'pow-mining', 'consensus',
    '展示算力竞争、Nonce 搜索和新区块出块过程。',
    1, 1, 'pow-mining', 'continuous', 1,
    '{"scene_code":"pow-mining","algorithm_type":"pow-mining","time_control":"continuous","step_duration":1500,"total_ticks":24}'::jsonb,
    '{"scene_code":"pow-mining","schema_version":"1.0","actions":[{"action_code":"adjust_hashrate","label":"调整算力","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"resource_id","label":"矿工标识","type":"string","required":true,"default":"miner-a"},{"name":"hashrate","label":"算力倍率","type":"number","required":true,"default":"1.2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001008, 'PoS 验证者选举', 'pos-validator', 'consensus',
    '展示质押权重、随机选举与 Epoch 轮转。',
    1, 1, 'pos-validator', 'process', 1,
    '{"scene_code":"pos-validator","algorithm_type":"pos-validator","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"pos-validator","schema_version":"1.0","actions":[{"action_code":"delegate_stake","label":"追加质押","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"resource_id","label":"验证者标识","type":"string","required":true,"default":"validator-a"},{"name":"stake","label":"质押数量","type":"number","required":true,"default":"100"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001009, 'PBFT 三阶段共识', 'pbft-consensus', 'consensus',
    '展示 Pre-prepare、Prepare、Commit 三阶段与视图切换。',
    1, 1, 'pbft-consensus', 'process', 1,
    '{"scene_code":"pbft-consensus","algorithm_type":"pbft-consensus","time_control":"process","step_duration":1600,"total_ticks":25}'::jsonb,
    '{"scene_code":"pbft-consensus","schema_version":"1.0","actions":[{"action_code":"inject_byzantine_node","label":"注入拜占庭节点","category":"attack_inject","trigger":"submit","roles":["student"],"fields":[{"name":"resource_id","label":"节点标识","type":"string","required":true,"default":"Replica-1"}]},{"action_code":"trigger_view_change","label":"触发视图切换","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"view","label":"视图编号","type":"number","required":true,"default":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001010, 'Raft 领导选举', 'raft-election', 'consensus',
    '展示超时、拉票、Leader 产生和日志复制。',
    1, 1, 'raft-election', 'process', 1,
    '{"scene_code":"raft-election","algorithm_type":"raft-election","time_control":"process","step_duration":1500,"total_ticks":20}'::jsonb,
    '{"scene_code":"raft-election","schema_version":"1.0","actions":[{"action_code":"fail_leader","label":"宕掉 Leader","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"Leader 标识","type":"string","required":true,"default":"Node-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001011, 'DPoS 委托投票', 'dpos-voting', 'consensus',
    '展示投票权重流向、超级节点排名和轮次出块。',
    1, 1, 'dpos-voting', 'process', 1,
    '{"scene_code":"dpos-voting","algorithm_type":"dpos-voting","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"dpos-voting","schema_version":"1.0","actions":[{"action_code":"reassign_vote","label":"重新投票","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"resource_id","label":"代表标识","type":"string","required":true,"default":"Delegate-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 密码学 (cryptography) — 6 个 ----
(
    920000000000001012, 'SHA-256 哈希过程', 'sha256-hash', 'cryptography',
    '展示消息分块、填充和 64 轮压缩函数的状态演化。',
    1, 1, 'sha256-hash', 'reactive', 1,
    '{"scene_code":"sha256-hash","algorithm_type":"sha256-hash","time_control":"reactive","step_duration":900,"total_ticks":64}'::jsonb,
    '{"scene_code":"sha256-hash","schema_version":"1.0","actions":[{"action_code":"mutate_input","label":"修改输入","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"input","label":"输入内容","type":"string","required":true,"default":"abc"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001013, 'Keccak-256 哈希过程', 'keccak256-hash', 'cryptography',
    '展示海绵结构吸收、置换和挤压输出。',
    1, 1, 'keccak256-hash', 'reactive', 1,
    '{"scene_code":"keccak256-hash","algorithm_type":"keccak256-hash","time_control":"reactive","step_duration":900,"total_ticks":24}'::jsonb,
    '{"scene_code":"keccak256-hash","schema_version":"1.0","actions":[{"action_code":"toggle_lane","label":"切换 Lane","category":"observe","trigger":"immediate","roles":["student"],"fields":[{"name":"lane","label":"Lane 索引","type":"number","required":true,"default":"0"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001014, 'ECDSA 签名验签', 'ecdsa-sign', 'cryptography',
    '展示密钥对、随机数生成、签名和验签流程。',
    1, 1, 'ecdsa-sign', 'reactive', 1,
    '{"scene_code":"ecdsa-sign","algorithm_type":"ecdsa-sign","time_control":"reactive","step_duration":1000,"total_ticks":12}'::jsonb,
    '{"scene_code":"ecdsa-sign","schema_version":"1.0","actions":[{"action_code":"sign_message","label":"重新签名","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"message","label":"消息内容","type":"string","required":true,"default":"hello"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001015, 'RSA 加密解密', 'rsa-encrypt', 'cryptography',
    '展示大数模幂运算和加密、解密、验签关系。',
    1, 1, 'rsa-encrypt', 'reactive', 1,
    '{"scene_code":"rsa-encrypt","algorithm_type":"rsa-encrypt","time_control":"reactive","step_duration":1000,"total_ticks":10}'::jsonb,
    '{"scene_code":"rsa-encrypt","schema_version":"1.0","actions":[{"action_code":"encrypt_plaintext","label":"重新加密","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"plaintext","label":"明文","type":"string","required":true,"default":"42"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001016, 'Merkle 树构建验证', 'merkle-tree', 'cryptography',
    '展示叶子哈希、逐层合并和验证路径。',
    1, 1, 'merkle-tree', 'reactive', 1,
    '{"scene_code":"merkle-tree","algorithm_type":"merkle-tree","time_control":"reactive","step_duration":900,"total_ticks":16}'::jsonb,
    '{"scene_code":"merkle-tree","schema_version":"1.0","actions":[{"action_code":"tamper_leaf","label":"篡改叶子","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"leaf","label":"叶子索引","type":"number","required":true,"default":"0"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001017, '零知识证明原理', 'zkp-basic', 'cryptography',
    '展示承诺、挑战、响应三步交互。',
    1, 1, 'zkp-basic', 'process', 1,
    '{"scene_code":"zkp-basic","algorithm_type":"zkp-basic","time_control":"process","step_duration":1400,"total_ticks":12}'::jsonb,
    '{"scene_code":"zkp-basic","schema_version":"1.0","actions":[{"action_code":"change_secret","label":"更换秘密","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"secret","label":"秘密值","type":"string","required":true,"default":"s1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 数据结构 (data_structure) — 5 个 ----
(
    920000000000001018, '区块链结构与分叉', 'blockchain-structure', 'data_structure',
    '展示主链、分叉链和最长链选择。',
    1, 1, 'blockchain-structure', 'reactive', 1,
    '{"scene_code":"blockchain-structure","algorithm_type":"blockchain-structure","time_control":"reactive","step_duration":900,"total_ticks":14}'::jsonb,
    '{"scene_code":"blockchain-structure","schema_version":"1.0","actions":[{"action_code":"append_block","label":"追加区块","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"branch","label":"目标分支","type":"string","required":true,"default":"main"}]},{"action_code":"fork_chain","label":"制造分叉","category":"attack_inject","trigger":"submit","roles":["student"],"fields":[{"name":"height","label":"分叉高度","type":"number","required":true,"default":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001019, '区块内部结构', 'block-internal', 'data_structure',
    '展示区块头字段、交易体和 Merkle 根关系。',
    1, 1, 'block-internal', 'reactive', 1,
    '{"scene_code":"block-internal","algorithm_type":"block-internal","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"block-internal","schema_version":"1.0","actions":[{"action_code":"expand_field","label":"展开字段","category":"observe","trigger":"immediate","roles":["student"],"fields":[{"name":"field","label":"字段名","type":"string","required":true,"default":"header"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001020, '状态树（MPT）', 'mpt-trie', 'data_structure',
    '展示路径查找、节点展开和状态更新传播。',
    1, 1, 'mpt-trie', 'reactive', 1,
    '{"scene_code":"mpt-trie","algorithm_type":"mpt-trie","time_control":"reactive","step_duration":900,"total_ticks":12}'::jsonb,
    '{"scene_code":"mpt-trie","schema_version":"1.0","actions":[{"action_code":"update_account","label":"更新账户","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"account","label":"账户键","type":"string","required":true,"default":"0xabc"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001021, '布隆过滤器', 'bloom-filter', 'data_structure',
    '展示位数组、多哈希映射和误判统计。',
    1, 1, 'bloom-filter', 'reactive', 1,
    '{"scene_code":"bloom-filter","algorithm_type":"bloom-filter","time_control":"reactive","step_duration":900,"total_ticks":8}'::jsonb,
    '{"scene_code":"bloom-filter","schema_version":"1.0","actions":[{"action_code":"query_key","label":"查询键","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"key","label":"元素键","type":"string","required":true,"default":"alice"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001022, '分布式存储 DHT', 'dht-storage', 'data_structure',
    '展示环形空间、键值映射和分片分布。',
    1, 1, 'dht-storage', 'continuous', 1,
    '{"scene_code":"dht-storage","algorithm_type":"dht-storage","time_control":"continuous","step_duration":1200,"total_ticks":14}'::jsonb,
    '{"scene_code":"dht-storage","schema_version":"1.0","actions":[{"action_code":"store_key","label":"存储键值","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"key","label":"键名","type":"string","required":true,"default":"doc-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 交易 (transaction) — 5 个 ----
(
    920000000000001023, '交易生命周期', 'tx-lifecycle', 'transaction',
    '展示创建、签名、广播、打包和确认全流程。',
    1, 1, 'tx-lifecycle', 'process', 1,
    '{"scene_code":"tx-lifecycle","algorithm_type":"tx-lifecycle","time_control":"process","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"tx-lifecycle","schema_version":"1.0","actions":[{"action_code":"create_tx","label":"创建交易","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"tx","label":"交易标识","type":"string","required":true,"default":"tx-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001024, 'Gas 计算与优化', 'gas-calculation', 'transaction',
    '展示操作码 Gas 瀑布图和优化建议。',
    1, 1, 'gas-calculation', 'reactive', 1,
    '{"scene_code":"gas-calculation","algorithm_type":"gas-calculation","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"gas-calculation","schema_version":"1.0","actions":[{"action_code":"switch_opcode","label":"切换操作码","category":"observe","trigger":"immediate","roles":["student"],"fields":[{"name":"opcode","label":"操作码","type":"string","required":true,"default":"SSTORE"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001025, 'Token 转账流转', 'token-transfer', 'transaction',
    '展示账户余额变化和 ERC-20 事件日志。',
    1, 1, 'token-transfer', 'process', 1,
    '{"scene_code":"token-transfer","algorithm_type":"token-transfer","time_control":"process","step_duration":1300,"total_ticks":12}'::jsonb,
    '{"scene_code":"token-transfer","schema_version":"1.0","actions":[{"action_code":"transfer_token","label":"执行转账","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"amount","label":"转账数量","type":"number","required":true,"default":"10"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001026, '交易排序与 MEV', 'tx-ordering-mev', 'transaction',
    '展示内存池排序、矿工优先级和三明治攻击。',
    1, 1, 'tx-ordering-mev', 'process', 1,
    '{"scene_code":"tx-ordering-mev","algorithm_type":"tx-ordering-mev","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"tx-ordering-mev","schema_version":"1.0","actions":[{"action_code":"inject_mev_bot","label":"注入 MEV Bot","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"bot","label":"机器人标识","type":"string","required":true,"default":"bot-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001027, '跨链桥接通信', 'cross-chain-bridge', 'transaction',
    '展示锁定、证明、中继和目标链铸造。',
    1, 1, 'cross-chain-bridge', 'process', 1,
    '{"scene_code":"cross-chain-bridge","algorithm_type":"cross-chain-bridge","time_control":"process","step_duration":1500,"total_ticks":16}'::jsonb,
    '{"scene_code":"cross-chain-bridge","schema_version":"1.0","actions":[{"action_code":"bridge_asset","label":"桥接资产","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"asset","label":"资产名称","type":"string","required":true,"default":"USDT"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 智能合约 (smart_contract) — 5 个 ----
(
    920000000000001028, '智能合约状态机', 'contract-state-machine', 'smart_contract',
    '展示状态转换、事件触发和存储读写。',
    1, 1, 'contract-state-machine', 'reactive', 1,
    '{"scene_code":"contract-state-machine","algorithm_type":"contract-state-machine","time_control":"reactive","step_duration":1000,"total_ticks":10}'::jsonb,
    '{"scene_code":"contract-state-machine","schema_version":"1.0","actions":[{"action_code":"fire_event","label":"触发事件","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"event","label":"事件名","type":"string","required":true,"default":"activate"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001029, 'EVM 执行步进', 'evm-execution', 'smart_contract',
    '展示操作码执行、栈内存变化和存储写回。',
    1, 1, 'evm-execution', 'process', 1,
    '{"scene_code":"evm-execution","algorithm_type":"evm-execution","time_control":"process","step_duration":1300,"total_ticks":20}'::jsonb,
    '{"scene_code":"evm-execution","schema_version":"1.0","actions":[{"action_code":"step_opcode","label":"执行单步","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"opcode","label":"操作码","type":"string","required":true,"default":"SLOAD"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001030, '合约间调用', 'contract-interaction', 'smart_contract',
    '展示 call、delegatecall 和上下文切换。',
    1, 1, 'contract-interaction', 'process', 1,
    '{"scene_code":"contract-interaction","algorithm_type":"contract-interaction","time_control":"process","step_duration":1300,"total_ticks":14}'::jsonb,
    '{"scene_code":"contract-interaction","schema_version":"1.0","actions":[{"action_code":"invoke_delegatecall","label":"触发 delegatecall","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"目标合约","type":"string","required":true,"default":"Library"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001031, '合约部署流程', 'contract-deployment', 'smart_contract',
    '展示字节码、构造函数和部署地址推导。',
    1, 1, 'contract-deployment', 'process', 1,
    '{"scene_code":"contract-deployment","algorithm_type":"contract-deployment","time_control":"process","step_duration":1400,"total_ticks":12}'::jsonb,
    '{"scene_code":"contract-deployment","schema_version":"1.0","actions":[{"action_code":"redeploy_contract","label":"重新部署","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"nonce","label":"部署 nonce","type":"number","required":true,"default":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001032, '状态通道', 'state-channel', 'smart_contract',
    '展示链上开通、链下更新、争议和关闭流程。',
    1, 1, 'state-channel', 'process', 1,
    '{"scene_code":"state-channel","algorithm_type":"state-channel","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"state-channel","schema_version":"1.0","actions":[{"action_code":"submit_dispute","label":"提交争议","category":"primary","trigger":"immediate","roles":["student"],"fields":[{"name":"proof","label":"争议证明","type":"string","required":true,"default":"proof-1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 攻击与安全 (attack_security) — 6 个 ----
(
    920000000000001033, '51% 算力攻击', '51-percent-attack', 'attack_security',
    '展示诚实链与攻击链竞赛和链重组。',
    1, 1, '51-percent-attack', 'process', 1,
    '{"scene_code":"51-percent-attack","algorithm_type":"51-percent-attack","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"51-percent-attack","schema_version":"1.0","actions":[{"action_code":"boost_attacker_hashrate","label":"提升攻击算力","category":"attack_inject","trigger":"submit","roles":["student"],"fields":[{"name":"ratio","label":"算力比例","type":"range","required":true,"default":"0.55"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001034, '双花攻击', 'double-spend', 'attack_security',
    '展示冲突交易、确认数竞赛和商家风险。',
    1, 1, 'double-spend', 'process', 1,
    '{"scene_code":"double-spend","algorithm_type":"double-spend","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"double-spend","schema_version":"1.0","actions":[{"action_code":"send_conflict_tx","label":"发送冲突交易","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"tx","label":"交易标识","type":"string","required":true,"default":"conflict-tx"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001035, 'PBFT 拜占庭攻击', 'pbft-byzantine', 'attack_security',
    '展示异常副本、消息偏差和容错阈值变化。',
    1, 1, 'pbft-byzantine', 'process', 1,
    '{"scene_code":"pbft-byzantine","algorithm_type":"pbft-byzantine","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"pbft-byzantine","schema_version":"1.0","actions":[{"action_code":"forge_vote","label":"伪造投票","category":"attack_inject","trigger":"submit","roles":["student"],"fields":[{"name":"resource_id","label":"副本标识","type":"string","required":true,"default":"Replica-2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001036, '重入攻击', 'reentrancy-attack', 'attack_security',
    '展示递归调用栈和余额被盗过程。',
    1, 1, 'reentrancy-attack', 'process', 1,
    '{"scene_code":"reentrancy-attack","algorithm_type":"reentrancy-attack","time_control":"process","step_duration":1500,"total_ticks":18}'::jsonb,
    '{"scene_code":"reentrancy-attack","schema_version":"1.0","actions":[{"action_code":"trigger_reentrancy","label":"触发重入","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"depth","label":"重入深度","type":"number","required":true,"default":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001037, '整数溢出攻击', 'integer-overflow', 'attack_security',
    '展示数值回绕、临界点和 SafeMath 对比。',
    1, 1, 'integer-overflow', 'reactive', 1,
    '{"scene_code":"integer-overflow","algorithm_type":"integer-overflow","time_control":"reactive","step_duration":900,"total_ticks":10}'::jsonb,
    '{"scene_code":"integer-overflow","schema_version":"1.0","actions":[{"action_code":"add_value","label":"增加数值","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"delta","label":"增量","type":"number","required":true,"default":"1"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001038, '自私挖矿', 'selfish-mining', 'attack_security',
    '展示私有链积累、公开策略和收益对比。',
    1, 1, 'selfish-mining', 'process', 1,
    '{"scene_code":"selfish-mining","algorithm_type":"selfish-mining","time_control":"process","step_duration":1400,"total_ticks":16}'::jsonb,
    '{"scene_code":"selfish-mining","schema_version":"1.0","actions":[{"action_code":"publish_private_chain","label":"公开私链","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"blocks","label":"区块数量","type":"number","required":true,"default":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
-- ---- 经济模型 (economic) — 5 个 ----
(
    920000000000001039, 'Token 经济模型', 'token-economics', 'economic',
    '展示供应量曲线、分配比例和释放节奏。',
    1, 1, 'token-economics', 'reactive', 1,
    '{"scene_code":"token-economics","algorithm_type":"token-economics","time_control":"reactive","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"token-economics","schema_version":"1.0","actions":[{"action_code":"switch_release_model","label":"切换释放模型","category":"param_tune","trigger":"submit","roles":["student"],"fields":[{"name":"inflation_rate","label":"年化通胀率","type":"number","required":true,"default":"0.04"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001040, 'PoS 质押经济', 'pos-staking', 'economic',
    '展示质押量、收益率和 Slash 事件。',
    1, 1, 'pos-staking', 'continuous', 1,
    '{"scene_code":"pos-staking","algorithm_type":"pos-staking","time_control":"continuous","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"pos-staking","schema_version":"1.0","actions":[{"action_code":"apply_slash","label":"执行 Slash","category":"attack_inject","trigger":"immediate","roles":["student"],"fields":[{"name":"resource_id","label":"验证者标识","type":"string","required":true,"default":"Validator-A"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001041, '链上治理投票', 'governance-voting', 'economic',
    '展示提案生命周期、投票权重和法定人数。',
    1, 1, 'governance-voting', 'process', 1,
    '{"scene_code":"governance-voting","algorithm_type":"governance-voting","time_control":"process","step_duration":1400,"total_ticks":18}'::jsonb,
    '{"scene_code":"governance-voting","schema_version":"1.0","actions":[{"action_code":"cast_vote","label":"投票","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"choice","label":"投票选项","type":"string","required":true,"default":"yes"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001042, 'DeFi 流动性池', 'defi-liquidity', 'economic',
    '展示 AMM 曲线、滑点和无常损失。',
    1, 1, 'defi-liquidity', 'reactive', 1,
    '{"scene_code":"defi-liquidity","algorithm_type":"defi-liquidity","time_control":"reactive","step_duration":900,"total_ticks":12}'::jsonb,
    '{"scene_code":"defi-liquidity","schema_version":"1.0","actions":[{"action_code":"swap_asset","label":"执行兑换","category":"primary","trigger":"submit","roles":["student"],"fields":[{"name":"amount","label":"兑换数量","type":"number","required":true,"default":"20"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
),
(
    920000000000001043, 'Gas 费市场（EIP-1559）', 'gas-market', 'economic',
    '展示基础费、区块利用率和燃烧量变化。',
    1, 1, 'gas-market', 'continuous', 1,
    '{"scene_code":"gas-market","algorithm_type":"gas-market","time_control":"continuous","step_duration":1200,"total_ticks":16}'::jsonb,
    '{"scene_code":"gas-market","schema_version":"1.0","actions":[{"action_code":"spike_demand","label":"提升需求","category":"param_tune","trigger":"immediate","roles":["student"],"fields":[{"name":"demand","label":"需求倍率","type":"number","required":true,"default":"2"}]}]}'::jsonb,
    '{"width":600,"height":400}'::jsonb, 1,
    'v1.0.0', NOW(), NOW()
)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

-- ---------------------------------------------------------------------
-- 内置场景算法容器镜像填充
-- ---------------------------------------------------------------------
-- 平台 43 个内置场景共享同一个运行时镜像 scenarios/runtime:v1.0.0，
-- 由 deploy/docker/scenario.Dockerfile 单次构建。
-- SimEngine SceneManager（K8sOrchestrator）创建场景 Pod 时注入
-- SCENE_CODE 环境变量来分发不同场景，无需每个场景独立打包。
-- 自定义场景在教师上传审核通过后单独写入自己的 container_image_url，
-- 不在此处赋值。
UPDATE sim_scenarios
SET container_image_url = 'registry.lianjing.com/scenarios/runtime:v1.0.0',
    updated_at = NOW()
WHERE source_type = 1
  AND (container_image_url IS NULL OR container_image_url = '');

-- =====================================================================
-- 02. 联动组 — 9 组
-- =====================================================================

INSERT INTO sim_link_groups (id, name, code, version, category, description, shared_state_schema, created_at, updated_at)
VALUES
(920000000000002001, 'PoW 攻击联动组', 'pow-attack-group', '1.0.0', 'attack',
 'PoW 挖矿、51% 攻击、区块同步、区块链结构之间共享算力与链状态。',
 '{"total_hashrate":{"type":"number","owner":"pow-mining"},"honest_hashrate":{"type":"number","owner":"pow-mining"},"attacker_hashrate":{"type":"number","owner":"51-percent-attack"},"chain_height":{"type":"integer","owner":"block-sync"},"fork_height":{"type":"integer","owner":"blockchain-structure"}}'::jsonb,
 NOW(), NOW()),
(920000000000002002, 'PBFT 攻击联动组', 'pbft-attack-group', '1.0.0', 'attack',
 'PBFT 共识、拜占庭攻击、网络分区之间共享副本状态与视图编号。',
 '{"view_number":{"type":"integer","owner":"pbft-consensus"},"byzantine_nodes":{"type":"array","items":{"type":"string"},"owner":"pbft-byzantine"},"partition_active":{"type":"boolean","owner":"network-partition"}}'::jsonb,
 NOW(), NOW()),
(920000000000002003, 'Raft 容错联动组', 'raft-fault-group', '1.0.0', 'consensus',
 'Raft 选举、网络分区、区块同步之间共享 Leader 与任期状态。',
 '{"leader":{"type":"string","owner":"raft-election"},"term":{"type":"integer","owner":"raft-election"},"partition_active":{"type":"boolean","owner":"network-partition"}}'::jsonb,
 NOW(), NOW()),
(920000000000002004, '网络基础联动组', 'network-base-group', '1.0.0', 'network',
 'P2P 发现、Gossip 传播、负载均衡之间共享节点拓扑。',
 '{"peer_count":{"type":"integer","owner":"p2p-discovery"},"topology":{"type":"object","owner":"p2p-discovery"},"coverage_ratio":{"type":"number","owner":"gossip-propagation"}}'::jsonb,
 NOW(), NOW()),
(920000000000002005, '密码学验证联动组', 'crypto-verify-group', '1.0.0', 'crypto',
 'SHA-256、ECDSA、Merkle 树之间共享哈希与签名状态。',
 '{"current_hash":{"type":"string","owner":"sha256-hash"},"signature":{"type":"string","owner":"ecdsa-sign"},"verified":{"type":"boolean","owner":"merkle-tree"}}'::jsonb,
 NOW(), NOW()),
(920000000000002006, '区块链完整性联动组', 'blockchain-integrity-group', '1.0.0', 'blockchain-integrity',
 '区块链结构、区块内部结构、Merkle 树、交易生命周期之间共享链数据完整性状态。',
 '{"merkle_root":{"type":"string","owner":"merkle-tree"},"block_hash":{"type":"string","owner":"block-internal"},"chain_valid":{"type":"boolean","owner":"blockchain-structure"}}'::jsonb,
 NOW(), NOW()),
(920000000000002007, '交易处理联动组', 'tx-processing-group', '1.0.0', 'economic',
 'Gas 计算、Token 转账、MEV、交易生命周期之间共享交易与 Gas 状态。',
 '{"gas_price":{"type":"number","owner":"gas-calculation"},"mempool_size":{"type":"integer","owner":"tx-lifecycle"},"pending_txs":{"type":"array","items":{"type":"string"},"owner":"tx-ordering-mev"}}'::jsonb,
 NOW(), NOW()),
(920000000000002008, 'PoS 经济联动组', 'pos-economy-group', '1.0.0', 'economic',
 'PoS 验证者、质押经济、Token 经济、治理投票之间共享质押与供应量状态。',
 '{"total_stake":{"type":"number","owner":"pos-validator"},"inflation_rate":{"type":"number","owner":"token-economics"},"active_validators":{"type":"integer","owner":"pos-staking"}}'::jsonb,
 NOW(), NOW()),
(920000000000002009, '合约安全联动组', 'contract-security-group', '1.0.0', 'attack',
 '状态机、EVM 执行、重入攻击、整数溢出之间共享合约运行时状态。',
 '{"current_state":{"type":"string","owner":"contract-state-machine"},"call_depth":{"type":"integer","owner":"evm-execution"},"balance":{"type":"number","owner":"reentrancy-attack"}}'::jsonb,
 NOW(), NOW())
ON CONFLICT (code, version) DO NOTHING;

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
    E'1. 进入仿真面板，观察 PoW 挖矿竞争动画\n2. 切换到 PoS 验证者选举场景，对比出块方式\n3. 在 PBFT 场景中注入拜占庭节点，观察容错\n4. 总结四种共识机制的优劣',
    1, NULL, 1, 100, 45, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008002,
    910000000000000001,
    910000000000001001,
    '密码学基础可视化实验',
    '通过可视化仿真理解哈希、签名、Merkle 树和零知识证明的工作原理。',
    '掌握 SHA-256 雪崩效应、ECDSA 签名验签流程、Merkle 树验证路径。',
    E'1. 在 SHA-256 场景中修改输入，观察雪崩效应\n2. 在 ECDSA 场景中完成签名和验签\n3. 在 Merkle 树场景中篡改叶子，观察验证路径变化\n4. 完成检查点验证',
    1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008003,
    910000000000000001,
    910000000000001001,
    '交易与 Gas 机制仿真实验',
    '通过可视化仿真理解交易生命周期、Gas 计算和 MEV 现象。',
    '理解交易从创建到确认的完整流程，掌握 Gas 机制和 MEV 对交易排序的影响。',
    E'1. 在交易生命周期场景中观察一笔交易的完整旅程\n2. 在 Gas 计算场景中对比不同操作码的消耗\n3. 在 MEV 场景中注入抢跑机器人，观察三明治攻击\n4. 完成检查点',
    1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
),
(
    920000000000008004,
    910000000000000002,
    910000000000001202,
    '区块链攻防安全仿真实验',
    '通过可视化仿真理解 51% 攻击、双花、重入攻击等安全威胁。',
    '理解各种攻击向量的原理与防御措施，培养安全意识。',
    E'1. 在 51% 攻击场景中调整算力比例，观察链重组\n2. 在重入攻击场景中触发递归调用，观察余额被盗\n3. 在整数溢出场景中增加数值到临界点\n4. 总结防御方案',
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
    E'1. 在 Remix IDE 中编写一个简单的存储合约\n2. 编译并部署到本地 geth 节点\n3. 切换到仿真面板，在 EVM 执行步进场景中跟踪操作码\n4. 在合约状态机场景中观察状态变迁\n5. 完成检查点验证',
    3, 1, 1, 100, 90, 30, 1, FALSE, 2, NOW(), NOW()
),
(
    920000000000008006,
    910000000000000001,
    910000000000001001,
    'PoW 挖矿与链观察混合实验',
    '在真实 geth 节点上进行挖矿操作，同时通过仿真面板可视化观察 PoW 过程和区块同步。',
    '将真实链节点操作与仿真可视化结合，理解挖矿竞争和区块传播的实际过程。',
    E'1. 启动 geth 开发节点，开启挖矿\n2. 通过区块浏览器观察出块情况\n3. 切换到仿真面板，在 PoW 挖矿场景中观察 Nonce 搜索动画\n4. 在区块同步场景中观察新区块传播\n5. 对比真实链数据与仿真数据',
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
 '{"scene_params":{"link_group_code":"contract-security-group"},"data_source_mode":3}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009017, 920000000000008005, 920000000000001028, 920000000000002009,
 '{"scene_params":{"link_group_code":"contract-security-group"},"data_source_mode":3}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),

-- PoW 挖矿混合实验（8006）→ 2 个仿真场景
(920000000000009018, 920000000000008006, 920000000000001007, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"},"data_source_mode":3}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009019, 920000000000008006, 920000000000001005, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"},"data_source_mode":3}'::jsonb,
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
    cpu_limit, memory_limit, depends_on,
    is_primary, created_at, updated_at
)
VALUES
-- 混合实验 8005: geth + remix-ide
(
    920000000000009101, 920000000000008005, (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'solidity-dev' AND iv.version = '1.0'),
    'geth', 1,
    '[]'::jsonb,
    '[{"port":8545,"protocol":"tcp","name":"HTTP-RPC"},{"port":30303,"protocol":"tcp","name":"P2P"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '[]'::jsonb, FALSE, NOW(), NOW()
),
(
    920000000000009102, 920000000000008005, (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'remix-ide' AND iv.version = 'latest'),
    'remix-ide', 1,
    '[{"name":"REMIX_URL","value":"http://geth:8545","desc":"RPC 地址"}]'::jsonb,
    '[{"port":8080,"protocol":"tcp","name":"Web UI"}]'::jsonb,
    '[]'::jsonb,
    '300m', '512Mi', '["geth"]'::jsonb, TRUE, NOW(), NOW()
),
-- 混合实验 8006: geth + blockscout
(
    920000000000009103, 920000000000008006, (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'solidity-dev' AND iv.version = '1.0'),
    'geth', 1,
    '[]'::jsonb,
    '[{"port":8545,"protocol":"tcp","name":"HTTP-RPC"},{"port":30303,"protocol":"tcp","name":"P2P"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '[]'::jsonb, FALSE, NOW(), NOW()
),
(
    920000000000009104, 920000000000008006, (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'blockscout' AND iv.version = '6.3'),
    'blockscout', 1,
    '[{"name":"ETHEREUM_JSONRPC_HTTP_URL","value":"http://geth:8545","desc":"EVM 节点 RPC 地址"}]'::jsonb,
    '[{"port":4000,"protocol":"tcp","name":"Web UI"}]'::jsonb,
    '[]'::jsonb,
    '500m', '1Gi', '["geth"]'::jsonb, TRUE, NOW(), NOW()
)
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 09. 检查点
-- =====================================================================

-- check_type: 1=脚本验证 / 2=手动评分 / 3=SimEngine 状态断言
-- 脚本断言走 script_content + script_language + target_container 列；
-- SimEngine 断言走 assertion_config 列，DSL 见
--   docs/modules/04-实验环境/02-数据库设计.md §2.6
--   {scene_code, conditions:[{path, operator, value}], require_all}
-- 算子集合：eq / ne / gt / gte / lt / lte / contains（与 CTF 模块一致）
INSERT INTO template_checkpoints (
    id, template_id, title, description,
    check_type, script_content, script_language, target_container,
    assertion_config, score, scope, sort_order,
    created_at, updated_at
)
VALUES
-- ---- 共识机制可视化对比（8001）— 4 个 SimEngine 状态断言 ----
(920000000000010001, 920000000000008001, '推进 PoW 出块', '在 PoW 场景中至少推进 10 个 Tick，观察算力竞赛与新区块产生。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pow-mining","conditions":[{"path":"$.tick","operator":"gte","value":10,"description":"至少推进 10 个 tick"}],"require_all":true}'::jsonb,
 25, 1, 1, NOW(), NOW()),
(920000000000010002, 920000000000008001, '推进 PBFT 至 Commit 阶段', '在 PBFT 场景中至少推进 15 个 Tick，达到 Commit 阶段。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pbft-consensus","conditions":[{"path":"$.tick","operator":"gte","value":15,"description":"至少 15 个 tick"},{"path":"$.phase_index","operator":"gte","value":2,"description":"达到 Commit（phase_index>=2）"}],"require_all":true}'::jsonb,
 25, 1, 2, NOW(), NOW()),
(920000000000010003, 920000000000008001, '推进 Raft 选举完成', '在 Raft 场景中至少推进 12 个 Tick，完成 Leader 选举与日志复制阶段。',
 3, NULL, NULL, NULL,
 '{"scene_code":"raft-election","conditions":[{"path":"$.tick","operator":"gte","value":12},{"path":"$.phase_index","operator":"gte","value":2,"description":"进入日志复制阶段"}],"require_all":true}'::jsonb,
 25, 1, 3, NOW(), NOW()),
(920000000000010004, 920000000000008001, '推进 PoS 验证者轮转', '在 PoS 场景中至少推进 10 个 Tick，观察一次完整 Epoch 轮转。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pos-validator","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 25, 1, 4, NOW(), NOW()),

-- ---- 密码学基础（8002）— 4 个 SimEngine 状态断言 ----
(920000000000010005, 920000000000008002, '完成 SHA-256 计算', '在 SHA-256 场景中至少进行 3 次输入修改，观察雪崩效应。',
 3, NULL, NULL, NULL,
 '{"scene_code":"sha256-hash","conditions":[{"path":"$.tick","operator":"gte","value":3,"description":"至少 3 次 mutate_input"}],"require_all":true}'::jsonb,
 25, 1, 1, NOW(), NOW()),
(920000000000010006, 920000000000008002, '完成 ECDSA 签名', '在 ECDSA 场景中至少推进 8 个 Tick，覆盖签名与验签流程。',
 3, NULL, NULL, NULL,
 '{"scene_code":"ecdsa-sign","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 25, 1, 2, NOW(), NOW()),
(920000000000010007, 920000000000008002, '完成 Merkle 树构建与篡改', '在 Merkle 树场景中至少推进 8 个 Tick，触发一次叶子篡改与验证路径失效。',
 3, NULL, NULL, NULL,
 '{"scene_code":"merkle-tree","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 25, 1, 3, NOW(), NOW()),
(920000000000010008, 920000000000008002, '完成零知识证明交互', '在 ZKP 场景中至少推进 10 个 Tick，覆盖承诺-挑战-响应三阶段。',
 3, NULL, NULL, NULL,
 '{"scene_code":"zkp-basic","conditions":[{"path":"$.tick","operator":"gte","value":10},{"path":"$.phase_index","operator":"gte","value":2,"description":"完成响应阶段"}],"require_all":true}'::jsonb,
 25, 1, 4, NOW(), NOW()),

-- ---- 交易与 Gas（8003）— 3 个 SimEngine 状态断言 ----
(920000000000010009, 920000000000008003, '追踪交易生命周期', '在交易生命周期场景中至少推进 12 个 Tick，覆盖创建到确认全流程。',
 3, NULL, NULL, NULL,
 '{"scene_code":"tx-lifecycle","conditions":[{"path":"$.tick","operator":"gte","value":12}],"require_all":true}'::jsonb,
 34, 1, 1, NOW(), NOW()),
(920000000000010010, 920000000000008003, '分析 Gas 消耗', '在 Gas 计算场景中至少进行 5 次操作码切换，对比消耗差异。',
 3, NULL, NULL, NULL,
 '{"scene_code":"gas-calculation","conditions":[{"path":"$.tick","operator":"gte","value":5}],"require_all":true}'::jsonb,
 33, 1, 2, NOW(), NOW()),
(920000000000010011, 920000000000008003, '观察 MEV 攻击', '在 MEV 场景中至少推进 10 个 Tick，观察抢跑机器人对排序的影响。',
 3, NULL, NULL, NULL,
 '{"scene_code":"tx-ordering-mev","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 33, 1, 3, NOW(), NOW()),

-- ---- 攻防安全（8004）— 4 个 SimEngine 状态断言 ----
(920000000000010012, 920000000000008004, '执行 51% 攻击', '在 51% 攻击场景中至少推进 12 个 Tick，观察链重组。',
 3, NULL, NULL, NULL,
 '{"scene_code":"51-percent-attack","conditions":[{"path":"$.tick","operator":"gte","value":12}],"require_all":true}'::jsonb,
 25, 1, 1, NOW(), NOW()),
(920000000000010013, 920000000000008004, '触发双花攻击', '在双花场景中至少推进 10 个 Tick，发送冲突交易并观察确认。',
 3, NULL, NULL, NULL,
 '{"scene_code":"double-spend","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 25, 1, 2, NOW(), NOW()),
(920000000000010014, 920000000000008004, '触发重入攻击', '在重入攻击场景中至少推进 10 个 Tick，观察资金被清空。',
 3, NULL, NULL, NULL,
 '{"scene_code":"reentrancy-attack","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 25, 1, 3, NOW(), NOW()),
(920000000000010015, 920000000000008004, '观察整数溢出', '在整数溢出场景中至少推进 8 个 Tick，触发回绕。',
 3, NULL, NULL, NULL,
 '{"scene_code":"integer-overflow","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 25, 1, 4, NOW(), NOW()),

-- ---- EVM 混合实验（8005）— 1 个脚本 + 2 个 SimEngine 状态断言 ----
(920000000000010016, 920000000000008005, '验证 geth RPC 可达', '在 geth 容器内通过 JSON-RPC 调用 eth_blockNumber，要求返回非零结果。',
 1,
 E'#!/bin/sh\nset -e\nresp=$(curl -sS -X POST http://localhost:8545 -H ''Content-Type: application/json'' -d ''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'')\necho "$resp"\necho "$resp" | grep -q ''"result"''\n',
 'bash', 'geth',
 NULL,
 40, 1, 1, NOW(), NOW()),
(920000000000010017, 920000000000008005, '推进 EVM 执行步进', '在 EVM 执行场景中至少推进 10 个 Tick，覆盖一段操作码序列。',
 3, NULL, NULL, NULL,
 '{"scene_code":"evm-execution","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 30, 1, 2, NOW(), NOW()),
(920000000000010018, 920000000000008005, '触发合约状态迁移', '在合约状态机场景中至少触发 3 次事件，覆盖状态迁移。',
 3, NULL, NULL, NULL,
 '{"scene_code":"contract-state-machine","conditions":[{"path":"$.tick","operator":"gte","value":3}],"require_all":true}'::jsonb,
 30, 1, 3, NOW(), NOW()),

-- ---- PoW 混合实验（8006）— 1 个脚本 + 2 个 SimEngine 状态断言 ----
(920000000000010019, 920000000000008006, '验证 geth 出块', '在 geth 容器内调用 eth_blockNumber，要求当前区块高度大于 0。',
 1,
 E'#!/bin/sh\nset -e\nresp=$(curl -sS -X POST http://localhost:8545 -H ''Content-Type: application/json'' -d ''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'')\necho "$resp"\nhex=$(echo "$resp" | sed -n ''s/.*"result":"\\([^"]*\\)".*/\\1/p'')\ntest -n "$hex"\ntest "$hex" != "0x0"\n',
 'bash', 'geth',
 NULL,
 30, 1, 1, NOW(), NOW()),
(920000000000010020, 920000000000008006, '推进 PoW 仿真', '在 PoW 挖矿仿真场景中至少推进 15 个 Tick。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pow-mining","conditions":[{"path":"$.tick","operator":"gte","value":15}],"require_all":true}'::jsonb,
 35, 1, 2, NOW(), NOW()),
(920000000000010021, 920000000000008006, '观察区块同步', '在区块同步场景中至少推进 10 个 Tick，观察新区块传播。',
 3, NULL, NULL, NULL,
 '{"scene_code":"block-sync","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 35, 1, 3, NOW(), NOW())
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

-- =====================================================================
-- 12. 补充单个/两个仿真场景的实验模板（experiment_type=1）
-- =====================================================================

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, is_shared, status, created_at, updated_at
)
VALUES
    -- 模板 8007：PoW 挖矿单场景实验
    (
        920000000000008007,
        910000000000000001,
        910000000000001001,
        'PoW 挖矿仿真实验',
        '通过可视化仿真深入理解 PoW 挖矿机制，包括算力竞争和新区块产生。',
        '掌握 PoW 共识的核心原理，理解算力对挖矿成功概率的影响。',
        E'1. 进入 PoW 挖矿仿真场景\n2. 调整不同矿工的算力参数\n3. 观察 Nonce 搜索动画和出块过程\n4. 完成检查点验证',
        1, NULL, 1, 100, 30, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8008：SHA-256 哈希单场景实验
    (
        920000000000008008,
        910000000000000001,
        910000000000001001,
        'SHA-256 哈希仿真实验',
        '通过可视化仿真理解 SHA-256 哈希算法的工作原理和雪崩效应。',
        '掌握 SHA-256 的分块处理、填充和压缩函数状态演化过程。',
        E'1. 进入 SHA-256 仿真场景\n2. 修改输入内容观察哈希值变化\n3. 观察雪崩效应（微小输入变化导致哈希值完全不同）\n4. 完成检查点验证',
        1, NULL, 1, 100, 25, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8009：ECDSA 签名单场景实验
    (
        920000000000008009,
        910000000000000001,
        910000000000001001,
        'ECDSA 签名仿真实验',
        '通过可视化仿真理解 ECDSA 签名和验签的完整流程。',
        '掌握椭圆曲线密钥对生成、随机数作用和签名验证过程。',
        E'1. 进入 ECDSA 签名仿真场景\n2. 观察密钥对生成过程\n3. 完成消息签名和验签\n4. 完成检查点验证',
        1, NULL, 1, 100, 25, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8010：P2P 网络发现单场景实验
    (
        920000000000008010,
        910000000000000001,
        910000000000001001,
        'P2P 网络发现仿真实验',
        '通过可视化仿真理解 P2P 网络的节点发现和路由表收敛过程。',
        '掌握 P2P 网络的拓扑发现、邻居维护和路由表更新机制。',
        E'1. 进入 P2P 网络发现仿真场景\n2. 添加新节点观察网络拓扑变化\n3. 观察路由表收敛过程\n4. 完成检查点验证',
        1, NULL, 1, 100, 30, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8011：交易生命周期单场景实验
    (
        920000000000008011,
        910000000000000001,
        910000000000001001,
        '交易生命周期仿真实验',
        '通过可视化仿真理解交易从创建到确认的完整生命周期。',
        '掌握交易的创建、签名、广播、打包和确认全流程。',
        E'1. 进入交易生命周期仿真场景\n2. 创建一笔新交易\n3. 观察交易在网络中的传播和打包过程\n4. 完成检查点验证',
        1, NULL, 1, 100, 30, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8012：51% 攻击单场景实验
    (
        920000000000008012,
        910000000000000001,
        910000000000001001,
        '51% 攻击仿真实验',
        '通过可视化仿真理解 51% 算力攻击的原理和链重组过程。',
        '理解算力优势如何导致链重组，以及双花攻击的实现方式。',
        E'1. 进入 51% 攻击仿真场景\n2. 调整攻击者算力比例\n3. 观察诚实链与攻击链的竞争\n4. 观察链重组和双花成功\n5. 完成检查点验证',
        1, NULL, 1, 100, 35, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8013：PoW + 区块同步双场景实验
    (
        920000000000008013,
        910000000000000001,
        910000000000001001,
        'PoW 挖矿与区块同步仿真实验',
        '通过双场景联动理解挖矿出块与区块传播的协同过程。',
        '掌握新区块产生后的网络传播和节点同步机制。',
        E'1. 进入 PoW 挖矿场景，观察出块过程\n2. 切换到区块同步场景，观察新区块传播\n3. 理解两个场景的联动关系\n4. 完成检查点验证',
        1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8014：SHA-256 + ECDSA 双场景实验
    (
        920000000000008014,
        910000000000000001,
        910000000000001001,
        '哈希与签名组合仿真实验',
        '通过双场景联动理解哈希和签名在区块链中的协同作用。',
        '掌握交易哈希计算和数字签名的完整流程。',
        E'1. 进入 SHA-256 场景，计算交易哈希\n2. 切换到 ECDSA 场景，对哈希值进行签名\n3. 理解哈希与签名的联动关系\n4. 完成检查点验证',
        1, NULL, 1, 100, 35, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8015：PBFT 共识单场景实验
    (
        920000000000008015,
        910000000000000001,
        910000000000001001,
        'PBFT 三阶段共识仿真实验',
        '通过可视化仿真理解 PBFT 的 Pre-prepare、Prepare、Commit 三阶段共识。',
        '掌握 PBFT 的容错机制和视图切换流程。',
        E'1. 进入 PBFT 共识仿真场景\n2. 观察三阶段消息传递\n3. 注入拜占庭节点观察容错\n4. 触发视图切换\n5. 完成检查点验证',
        1, NULL, 1, 100, 40, 30, 1, TRUE, 2, NOW(), NOW()
    ),
    -- 模板 8016：Gas 计算单场景实验
    (
        920000000000008016,
        910000000000000001,
        910000000000001001,
        'Gas 计算与优化仿真实验',
        '通过可视化仿真理解 EVM Gas 计算机制和优化策略。',
        '掌握不同操作码的 Gas 消耗和优化方法。',
        E'1. 进入 Gas 计算仿真场景\n2. 切换不同操作码观察 Gas 消耗\n3. 理解 Gas 瀑布图\n4. 完成检查点验证',
        1, NULL, 1, 100, 25, 30, 1, TRUE, 2, NOW(), NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 13. 新增单场景/双场景模板的 template_sim_scenes
-- =====================================================================

INSERT INTO template_sim_scenes (id, template_id, scenario_id, link_group_id, config, layout_position, sort_order, created_at, updated_at)
VALUES
-- 模板 8007：PoW 挖矿单场景
(920000000000009020, 920000000000008007, 920000000000001007, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8008：SHA-256 单场景
(920000000000009021, 920000000000008008, 920000000000001012, NULL,
 '{}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8009：ECDSA 签名单场景
(920000000000009022, 920000000000008009, 920000000000001014, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8010：P2P 网络发现单场景
(920000000000009023, 920000000000008010, 920000000000001001, 920000000000002004,
 '{"scene_params":{"link_group_code":"network-base-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8011：交易生命周期单场景
(920000000000009024, 920000000000008011, 920000000000001023, 920000000000002007,
 '{"scene_params":{"link_group_code":"tx-processing-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8012：51% 攻击单场景
(920000000000009025, 920000000000008012, 920000000000001033, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8013：PoW + 区块同步双场景
(920000000000009026, 920000000000008013, 920000000000001007, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009027, 920000000000008013, 920000000000001005, 920000000000002001,
 '{"scene_params":{"link_group_code":"pow-attack-group"}}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),

-- 模板 8014：SHA-256 + ECDSA 双场景
(920000000000009028, 920000000000008014, 920000000000001012, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),
(920000000000009029, 920000000000008014, 920000000000001014, 920000000000002005,
 '{"scene_params":{"link_group_code":"crypto-verify-group"}}'::jsonb,
 '{"row":0,"col":1}'::jsonb, 2, NOW(), NOW()),

-- 模板 8015：PBFT 共识单场景
(920000000000009030, 920000000000008015, 920000000000001009, 920000000000002002,
 '{"scene_params":{"link_group_code":"pbft-attack-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW()),

-- 模板 8016：Gas 计算单场景
(920000000000009031, 920000000000008016, 920000000000001024, 920000000000002007,
 '{"scene_params":{"link_group_code":"tx-processing-group"}}'::jsonb,
 '{"row":0,"col":0}'::jsonb, 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 14. 新增模板的检查点
-- =====================================================================

INSERT INTO template_checkpoints (
    id, template_id, title, description, check_type, script_content, script_language,
    target_container, assertion_config, score, scope, sort_order, created_at, updated_at
)
VALUES
-- 模板 8007 检查点
(920000000000010022, 920000000000008007, '推进 PoW 出块', '在 PoW 场景中至少推进 15 个 Tick，观察算力竞赛与新区块产生。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pow-mining","conditions":[{"path":"$.tick","operator":"gte","value":15}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8008 检查点
(920000000000010023, 920000000000008008, '完成 SHA-256 计算', '在 SHA-256 场景中至少进行 3 次输入修改，观察雪崩效应。',
 3, NULL, NULL, NULL,
 '{"scene_code":"sha256-hash","conditions":[{"path":"$.tick","operator":"gte","value":3}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8009 检查点
(920000000000010024, 920000000000008009, '完成 ECDSA 签名', '在 ECDSA 场景中至少推进 8 个 Tick，覆盖签名与验签流程。',
 3, NULL, NULL, NULL,
 '{"scene_code":"ecdsa-sign","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8010 检查点
(920000000000010025, 920000000000008010, '添加节点并观察收敛', '在 P2P 网络发现场景中至少添加 3 个节点，观察路由表收敛。',
 3, NULL, NULL, NULL,
 '{"scene_code":"p2p-discovery","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8011 检查点
(920000000000010026, 920000000000008011, '完成交易生命周期', '在交易生命周期场景中至少推进 15 个 Tick，观察交易从创建到确认。',
 3, NULL, NULL, NULL,
 '{"scene_code":"tx-lifecycle","conditions":[{"path":"$.tick","operator":"gte","value":15}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8012 检查点
(920000000000010027, 920000000000008012, '触发 51% 攻击成功', '在 51% 攻击场景中调整算力比例至 55% 以上，观察链重组。',
 3, NULL, NULL, NULL,
 '{"scene_code":"51-percent-attack","conditions":[{"path":"$.attacker_hashrate_ratio","operator":"gte","value":0.55}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8013 检查点
(920000000000010028, 920000000000008013, '推进 PoW 出块与区块同步', '在 PoW 场景推进 10 Tick，在区块同步场景推进 8 Tick。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pow-mining","conditions":[{"path":"$.tick","operator":"gte","value":10}],"require_all":true}'::jsonb,
 50, 1, 1, NOW(), NOW()),
(920000000000010029, 920000000000008013, '观察区块传播', '在区块同步场景中观察新区块传播过程。',
 3, NULL, NULL, NULL,
 '{"scene_code":"block-sync","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 50, 1, 2, NOW(), NOW()),

-- 模板 8014 检查点
(920000000000010030, 920000000000008014, '完成哈希计算', '在 SHA-256 场景中完成至少 2 次哈希计算。',
 3, NULL, NULL, NULL,
 '{"scene_code":"sha256-hash","conditions":[{"path":"$.tick","operator":"gte","value":2}],"require_all":true}'::jsonb,
 50, 1, 1, NOW(), NOW()),
(920000000000010031, 920000000000008014, '完成签名验证', '在 ECDSA 场景中完成签名与验签。',
 3, NULL, NULL, NULL,
 '{"scene_code":"ecdsa-sign","conditions":[{"path":"$.tick","operator":"gte","value":8}],"require_all":true}'::jsonb,
 50, 1, 2, NOW(), NOW()),

-- 模板 8015 检查点
(920000000000010032, 920000000000008015, '推进 PBFT 至 Commit 阶段', '在 PBFT 场景中至少推进 15 个 Tick，达到 Commit 阶段。',
 3, NULL, NULL, NULL,
 '{"scene_code":"pbft-consensus","conditions":[{"path":"$.tick","operator":"gte","value":15},{"path":"$.phase_index","operator":"gte","value":2}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW()),

-- 模板 8016 检查点
(920000000000010033, 920000000000008016, '切换操作码观察 Gas', '在 Gas 计算场景中切换至少 3 种不同操作码。',
 3, NULL, NULL, NULL,
 '{"scene_code":"gas-calculation","conditions":[{"path":"$.tick","operator":"gte","value":3}],"require_all":true}'::jsonb,
 100, 1, 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 15. 新增模板的标签关联
-- =====================================================================

INSERT INTO template_tags (id, template_id, tag_id, created_at)
VALUES
-- 模板 8007（PoW 挖矿）
(920000000000012019, 920000000000008007, 920000000000011001, NOW()),
(920000000000012020, 920000000000008007, 920000000000011003, NOW()),
(920000000000012021, 920000000000008007, 920000000000011008, NOW()),
-- 模板 8008（SHA-256）
(920000000000012022, 920000000000008008, 920000000000011001, NOW()),
(920000000000012023, 920000000000008008, 920000000000011004, NOW()),
(920000000000012024, 920000000000008008, 920000000000011008, NOW()),
-- 模板 8009（ECDSA）
(920000000000012025, 920000000000008009, 920000000000011001, NOW()),
(920000000000012026, 920000000000008009, 920000000000011004, NOW()),
(920000000000012027, 920000000000008009, 920000000000011008, NOW()),
-- 模板 8010（P2P 网络发现）
(920000000000012028, 920000000000008010, 920000000000011001, NOW()),
(920000000000012029, 920000000000008010, 920000000000011008, NOW()),
-- 模板 8011（交易生命周期）
(920000000000012030, 920000000000008011, 920000000000011001, NOW()),
(920000000000012031, 920000000000008011, 920000000000011005, NOW()),
(920000000000012032, 920000000000008011, 920000000000011008, NOW()),
-- 模板 8012（51% 攻击）
(920000000000012033, 920000000000008012, 920000000000011001, NOW()),
(920000000000012034, 920000000000008012, 920000000000011006, NOW()),
(920000000000012035, 920000000000008012, 920000000000011008, NOW()),
-- 模板 8013（PoW + 区块同步）
(920000000000012036, 920000000000008013, 920000000000011001, NOW()),
(920000000000012037, 920000000000008013, 920000000000011003, NOW()),
(920000000000012038, 920000000000008013, 920000000000011008, NOW()),
-- 模板 8014（SHA-256 + ECDSA）
(920000000000012039, 920000000000008014, 920000000000011001, NOW()),
(920000000000012040, 920000000000008014, 920000000000011004, NOW()),
(920000000000012041, 920000000000008014, 920000000000011008, NOW()),
-- 模板 8015（PBFT）
(920000000000012042, 920000000000008015, 920000000000011001, NOW()),
(920000000000012043, 920000000000008015, 920000000000011003, NOW()),
(920000000000012044, 920000000000008015, 920000000000011008, NOW()),
-- 模板 8016（Gas 计算）
(920000000000012045, 920000000000008016, 920000000000011001, NOW()),
(920000000000012046, 920000000000008016, 920000000000011005, NOW()),
(920000000000012047, 920000000000008016, 920000000000011008, NOW())
ON CONFLICT (template_id, tag_id) DO NOTHING;

-- =====================================================================
-- 16. 新增课程课时与实验关联
-- =====================================================================

INSERT INTO lessons (id, chapter_id, course_id, title, content_type, experiment_id, sort_order, estimated_minutes, created_at, updated_at)
VALUES
(920000000000013104, 920000000000013001, 910000000000007001, '4.4 PoW 挖矿仿真', 4, 920000000000008007, 4, 30, NOW(), NOW()),
(920000000000013105, 920000000000013001, 910000000000007001, '4.5 密码学基础单场景', 4, 920000000000008008, 5, 25, NOW(), NOW()),
(920000000000013106, 920000000000013001, 910000000000007001, '4.6 P2P 网络发现仿真', 4, 920000000000008010, 6, 30, NOW(), NOW()),
(920000000000013107, 920000000000013001, 910000000000007001, '4.7 交易生命周期仿真', 4, 920000000000008011, 7, 30, NOW(), NOW()),
(920000000000013108, 920000000000013001, 910000000000007001, '4.8 攻防安全仿真', 4, 920000000000008012, 8, 35, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO course_experiments (id, course_id, experiment_id, title, sort_order, created_at)
VALUES
(920000000000014004, 910000000000007001, 920000000000008007, 'PoW 挖矿仿真', 4, NOW()),
(920000000000014005, 910000000000007001, 920000000000008008, 'SHA-256 哈希仿真', 5, NOW()),
(920000000000014006, 910000000000007001, 920000000000008009, 'ECDSA 签名仿真', 6, NOW()),
(920000000000014007, 910000000000007001, 920000000000008010, 'P2P 网络发现仿真', 7, NOW()),
(920000000000014008, 910000000000007001, 920000000000008011, '交易生命周期仿真', 8, NOW()),
(920000000000014009, 910000000000007001, 920000000000008012, '51% 攻击仿真', 9, NOW()),
(920000000000014010, 910000000000007001, 920000000000008013, 'PoW 挖矿与区块同步', 10, NOW()),
(920000000000014011, 910000000000007001, 920000000000008014, '哈希与签名组合', 11, NOW()),
(920000000000014012, 910000000000007001, 920000000000008015, 'PBFT 共识仿真', 12, NOW()),
(920000000000014013, 910000000000007001, 920000000000008016, 'Gas 计算仿真', 13, NOW())
ON CONFLICT (id) DO NOTHING;
