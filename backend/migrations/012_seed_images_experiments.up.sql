-- 补充镜像库与多镜像组编排实验模板种子数据。
-- 目标：
-- 1. 添加 Fabric 生态、通用中间件与工具类镜像
-- 2. 添加 geth v1.13 历史版本
-- 3. 添加 topology_mode=2 (单人多节点) Fabric 网络搭建实验
-- 4. 添加 topology_mode=1 (多容器) EVM 全栈 DApp 开发实验
-- 5. 添加 topology_mode=3 (多人协作) Fabric 多组织组网实验（含角色定义）
-- 6. 为新模板添加容器、检查点与课程关联
--
-- 使用方式：在 010、011 之后执行。
-- 约定：所有 ID 使用固定值，与 010 系列延续编号。

-- =====================================================================
-- 01. 添加镜像 — 链节点 / 中间件 / 工具 / 基础开发环境
-- =====================================================================

INSERT INTO images (
    id, category_id, name, display_name, description, ecosystem, source_type, status,
    default_ports, default_env_vars, default_volumes, typical_companions, required_dependencies,
    resource_recommendation, documentation_url, usage_count, created_at, updated_at
)
VALUES
    -- ---------- 链节点 ----------
    (
        910000000000005004,
        910000000000004002,
        'fabric-peer',
        'Hyperledger Fabric Peer',
        'Fabric 网络中的 Peer 节点，负责背书、提交与状态管理。',
        'fabric',
        1,
        1,
        '[{"port":7051,"protocol":"tcp","name":"gRPC"},{"port":7053,"protocol":"tcp","name":"Event"}]'::jsonb,
        '[{"key":"CORE_PEER_ID","value":"peer0.org1.example.com","desc":"Peer 节点 ID","conditions":[]}]'::jsonb,
        '[{"path":"/var/hyperledger/production","desc":"Peer 数据持久化"}]'::jsonb,
        '{"required":[{"image":"fabric-orderer","reason":"排序服务"}],"recommended":[{"image":"couchdb","reason":"状态数据库（替代 LevelDB）"},{"image":"fabric-ca","reason":"证书颁发"}],"optional":[{"image":"fabric-explorer","reason":"Fabric 区块链浏览器"},{"image":"fabric-tools","reason":"Fabric CLI 工具"}]}'::jsonb,
        '["fabric-orderer"]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"10Gi"}'::jsonb,
        '/docs/images/fabric-peer',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005005,
        910000000000004002,
        'fabric-orderer',
        'Hyperledger Fabric Orderer',
        'Fabric 排序节点，负责交易排序与出块。',
        'fabric',
        1,
        1,
        '[{"port":7050,"protocol":"tcp","name":"gRPC"}]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/var/hyperledger/production/orderer","desc":"Orderer 数据持久化"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"fabric-peer","reason":"Peer 节点"}],"optional":[{"image":"fabric-explorer","reason":"Fabric 浏览器"}]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"10Gi"}'::jsonb,
        '/docs/images/fabric-orderer',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005006,
        910000000000004002,
        'fabric-ca',
        'Hyperledger Fabric CA',
        'Fabric 证书颁发服务，管理组织身份与加密材料。',
        'fabric',
        1,
        1,
        '[{"port":7054,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/etc/hyperledger/fabric-ca-server","desc":"CA 配置和证书数据"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"fabric-peer","reason":"Peer 节点需要 CA 颁发证书"}],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.25","memory":"256Mi","disk":"1Gi"}'::jsonb,
        '/docs/images/fabric-ca',
        0,
        NOW(),
        NOW()
    ),
    -- ---------- 中间件 ----------
    (
        910000000000005007,
        910000000000004003,
        'couchdb',
        'CouchDB',
        '文档数据库，Fabric Peer 可用 CouchDB 替代 LevelDB 作为状态存储。',
        'fabric',
        1,
        1,
        '[{"port":5984,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/opt/couchdb/data","desc":"数据持久化"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"fabric-peer","reason":"Fabric Peer 使用 CouchDB 作为状态数据库"}],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.25","memory":"256Mi","disk":"5Gi"}'::jsonb,
        '/docs/images/couchdb',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005008,
        910000000000004003,
        'postgres',
        'PostgreSQL',
        '关系型数据库，用于 Blockscout、Fabric Explorer 等工具的数据存储。',
        'general',
        1,
        1,
        '[{"port":5432,"protocol":"tcp","name":"PostgreSQL"}]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/var/lib/postgresql/data","desc":"数据持久化"}]'::jsonb,
        '{"required":[],"recommended":[],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"10Gi"}'::jsonb,
        '/docs/images/postgres',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005009,
        910000000000004003,
        'redis',
        'Redis',
        '内存缓存数据库，用于会话缓存与消息中间件场景教学。',
        'general',
        1,
        1,
        '[{"port":6379,"protocol":"tcp","name":"Redis"}]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/data","desc":"数据持久化"}]'::jsonb,
        '{"required":[],"recommended":[],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.25","memory":"256Mi","disk":"2Gi"}'::jsonb,
        '/docs/images/redis',
        0,
        NOW(),
        NOW()
    ),
    -- ---------- 工具 ----------
    (
        910000000000005010,
        910000000000004004,
        'code-server',
        'VS Code Web IDE',
        '基于浏览器的 VS Code 在线编辑器，可嵌入实验环境直接编码。',
        'general',
        1,
        1,
        '[{"port":8080,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[{"key":"PASSWORD","value":"","desc":"访问密码（空则免密）","conditions":[]}]'::jsonb,
        '[{"path":"/home/coder/project","desc":"项目工作目录"}]'::jsonb,
        '{"required":[],"recommended":[],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"5Gi"}'::jsonb,
        '/docs/images/code-server',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005011,
        910000000000004004,
        'remix-ide',
        'Remix IDE',
        'Solidity 在线开发环境，可直接连接链节点进行合约部署调试。',
        'ethereum',
        1,
        1,
        '[{"port":8080,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '{"required":[],"recommended":[{"image":"geth","reason":"本地以太坊节点"},{"image":"ganache","reason":"轻量级 EVM 模拟器"}],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"2Gi"}'::jsonb,
        '/docs/images/remix-ide',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005012,
        910000000000004004,
        'fabric-tools',
        'Fabric CLI Tools',
        'Hyperledger Fabric CLI 工具集，包含 peer、configtxgen 等命令行工具。',
        'fabric',
        1,
        1,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '{"required":[],"recommended":[{"image":"fabric-peer","reason":"操作 Fabric 网络"}],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.25","memory":"256Mi","disk":"2Gi"}'::jsonb,
        '/docs/images/fabric-tools',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005013,
        910000000000004004,
        'fabric-explorer',
        'Hyperledger Explorer',
        'Fabric 区块链浏览器，可视化展示通道、区块与交易。',
        'fabric',
        1,
        1,
        '[{"port":8080,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '{"required":[{"image":"fabric-peer","reason":"需要连接 Fabric Peer 节点"},{"image":"postgres","reason":"数据存储"}],"recommended":[],"optional":[]}'::jsonb,
        '["fabric-peer","postgres"]'::jsonb,
        '{"cpu":"0.5","memory":"512Mi","disk":"5Gi"}'::jsonb,
        '/docs/images/fabric-explorer',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005014,
        910000000000004004,
        'xterm-server',
        'Web Terminal',
        'Web 终端服务，为实验容器提供浏览器内嵌终端访问。',
        'general',
        1,
        1,
        '[{"port":3000,"protocol":"tcp","name":"HTTP"}]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '{"required":[],"recommended":[],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.25","memory":"256Mi","disk":"1Gi"}'::jsonb,
        '/docs/images/xterm-server',
        0,
        NOW(),
        NOW()
    ),
    -- ---------- 基础开发环境 ----------
    (
        910000000000005015,
        910000000000004001,
        'go-dev',
        'Go Development Workspace',
        '面向 Fabric 链码与 Go 区块链应用的开发工作空间。',
        'fabric',
        1,
        1,
        '[]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/home/developer/project","desc":"项目工作目录"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"fabric-peer","reason":"Fabric 网络"}],"optional":[{"image":"fabric-tools","reason":"Fabric CLI"}]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"1Gi","disk":"5Gi"}'::jsonb,
        '/docs/images/go-dev',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005016,
        910000000000004001,
        'dapp-dev',
        'DApp Development Workspace',
        '面向去中心化应用前端开发的工作空间，含 ethers.js、web3.js 等前端库。',
        'ethereum',
        1,
        1,
        '[]'::jsonb,
        '[]'::jsonb,
        '[{"path":"/home/developer/project","desc":"项目工作目录"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"geth","reason":"本地以太坊节点"},{"image":"ganache","reason":"轻量级 EVM 模拟器"}],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"1Gi","disk":"5Gi"}'::jsonb,
        '/docs/images/dapp-dev',
        0,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 04. 添加镜像版本
-- =====================================================================

INSERT INTO image_versions (
    id, image_id, version, registry_url, min_cpu, min_memory, min_disk, is_default, status, created_at, updated_at
)
VALUES
    -- geth v1.13（教学兼容旧版）
    (
        910000000000006004,
        910000000000005002,
        '1.13',
        'registry.lianjing.com/chain-nodes/geth:v1.13.15',
        '250m',
        '512Mi',
        '10Gi',
        FALSE,
        1,
        NOW(),
        NOW()
    ),
    -- fabric-peer v2.5
    (
        910000000000006005,
        910000000000005004,
        '2.5',
        'registry.lianjing.com/chain-nodes/fabric-peer:v2.5',
        '250m',
        '256Mi',
        '10Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- fabric-orderer v2.5
    (
        910000000000006006,
        910000000000005005,
        '2.5',
        'registry.lianjing.com/chain-nodes/fabric-orderer:v2.5',
        '250m',
        '256Mi',
        '10Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- fabric-ca v1.5
    (
        910000000000006007,
        910000000000005006,
        '1.5',
        'registry.lianjing.com/chain-nodes/fabric-ca:v1.5',
        '100m',
        '128Mi',
        '1Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- couchdb v3.3
    (
        910000000000006008,
        910000000000005007,
        '3.3',
        'registry.lianjing.com/middleware/couchdb:v3.3',
        '100m',
        '128Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- postgres v15
    (
        910000000000006009,
        910000000000005008,
        '15',
        'registry.lianjing.com/middleware/postgres:v15',
        '250m',
        '256Mi',
        '10Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- redis v7
    (
        910000000000006010,
        910000000000005009,
        '7',
        'registry.lianjing.com/middleware/redis:v7-alpine',
        '100m',
        '128Mi',
        '2Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- code-server v4.89
    (
        910000000000006011,
        910000000000005010,
        '4.89',
        'registry.lianjing.com/tools/code-server:v4.89.1',
        '250m',
        '256Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- remix-ide latest
    (
        910000000000006012,
        910000000000005011,
        'latest',
        'registry.lianjing.com/tools/remix-ide:latest',
        '250m',
        '256Mi',
        '2Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- fabric-tools v2.5
    (
        910000000000006013,
        910000000000005012,
        '2.5',
        'registry.lianjing.com/tools/fabric-tools:v2.5',
        '100m',
        '128Mi',
        '2Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- fabric-explorer v1.1
    (
        910000000000006014,
        910000000000005013,
        '1.1',
        'registry.lianjing.com/tools/fabric-explorer:v1.1.8',
        '250m',
        '256Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- xterm-server v1.0
    (
        910000000000006015,
        910000000000005014,
        '1.0',
        'registry.lianjing.com/tools/xterm-server:v1.0.0',
        '100m',
        '128Mi',
        '1Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- go-dev v1.0
    (
        910000000000006016,
        910000000000005015,
        '1.0',
        'registry.lianjing.com/base/go-dev:v1.0.0',
        '250m',
        '512Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    -- dapp-dev v1.0
    (
        910000000000006017,
        910000000000005016,
        '1.0',
        'registry.lianjing.com/base/dapp-dev:v1.0.0',
        '250m',
        '512Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 05. 添加实验模板（覆盖 topology_mode 2 / 1-多容器 / 3）
-- =====================================================================

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, status, created_at, updated_at
)
VALUES
    -- 模板 5：Fabric 单人多节点网络搭建（topology_mode=2）
    -- 一个学生独立搭建完整的 Fabric 网络：CA → Orderer → CouchDB → Peer → CLI 工具 → 开发环境
    (
        910000000000008005,
        910000000000000001,
        910000000000001001,
        'Fabric 单人多节点网络搭建实验',
        '学生独立搭建一个包含 CA、Orderer、Peer、CouchDB 和 CLI 的完整 Fabric 网络。',
        '理解 Fabric 网络拓扑与各组件职责，掌握单人多节点环境编排能力。',
        '1. 启动 CA 服务并注册组织身份\n2. 启动 Orderer 排序节点\n3. 启动 CouchDB 状态数据库\n4. 启动 Peer 节点并加入通道\n5. 使用 CLI 工具部署链码\n6. 在开发环境中调用链码并验证',
        2,
        2,
        1,
        100,
        120,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 6：EVM 全栈 DApp 开发实验（topology_mode=1，多容器）
    -- 4 个容器协同：geth 节点 + blockscout 浏览器 + dapp-dev 开发环境 + remix-ide
    (
        910000000000008006,
        910000000000000001,
        910000000000001001,
        'EVM 全栈 DApp 开发实验',
        '学生在多容器环境中完成从合约编写、部署到 DApp 前端联调的全流程。',
        '理解 EVM 全栈开发工作流，熟悉节点、浏览器、IDE 与前端开发环境的协同关系。',
        '1. 启动 geth 本地开发链\n2. 在 Remix IDE 中编写并部署合约\n3. 在 Blockscout 中确认部署结果\n4. 在 DApp 开发环境中编写前端并连接合约',
        2,
        1,
        1,
        100,
        90,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 7：Fabric 多人协作组网实验（topology_mode=3）
    -- 多名学生分别扮演 Org1 管理员、Org2 管理员、Orderer 运维，协作搭建多组织网络
    (
        910000000000008007,
        910000000000000002,
        910000000000001202,
        'Fabric 多人协作组网实验',
        '多名学生分角色协作，搭建包含两个组织和排序服务的 Fabric 多组织网络。',
        '理解多组织协作模式下的证书管理、通道创建与跨组织交易流程。',
        '1. 共享 CA 启动并为各组织生成证书\n2. Orderer 运维角色启动排序节点并创建通道\n3. Org1 管理员启动 Peer + CouchDB 并加入通道\n4. Org2 管理员启动 Peer + CouchDB 并加入通道\n5. 通过共享 CLI 工具完成链码部署\n6. 各组织使用开发环境调用链码并验证跨组织交易',
        2,
        3,
        1,
        100,
        120,
        30,
        1,
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 06. 多人协作角色定义（模板 7）
-- =====================================================================

INSERT INTO template_roles (id, template_id, role_name, description, max_members, sort_order, created_at, updated_at)
VALUES
    (910000000000020001, 910000000000008007, 'Org1 管理员', '负责 Org1 的 Peer 节点、CouchDB 和组织证书管理。', 1, 1, NOW(), NOW()),
    (910000000000020002, 910000000000008007, 'Org2 管理员', '负责 Org2 的 Peer 节点、CouchDB 和组织证书管理。', 1, 2, NOW(), NOW()),
    (910000000000020003, 910000000000008007, 'Orderer 运维', '负责排序节点启动、通道创建与维护。', 1, 3, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 07. 模板容器配置
-- =====================================================================
--
-- deployment_scope 枚举：
--   1 = 实例独占（每个学生一份）
--   2 = 共享基础设施（全班共享一份）
--
-- 模板 5（Fabric 单人多节点，topology_mode=2）
-- 所有容器 deployment_scope=1，每个学生独立拥有完整网络
-- 启动顺序：CA(1) → Orderer(2) + CouchDB(2) → Peer(3) → fabric-tools(4) → go-dev(5)

INSERT INTO template_containers (
    id, template_id, image_version_id, container_name, deployment_scope, role_id,
    env_vars, ports, volumes, cpu_limit, memory_limit, depends_on, startup_order,
    is_primary, sort_order, created_at, updated_at
)
VALUES
    -- ---------- 模板 5：Fabric 单人多节点 ----------
    (
        910000000000009008,
        910000000000008005,
        910000000000006007,  -- fabric-ca v1.5
        'fabric-ca',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":7054,"service_port":7054,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        1,
        FALSE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000009009,
        910000000000008005,
        910000000000006006,  -- fabric-orderer v2.5
        'fabric-orderer',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":7050,"service_port":7050,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[{"container_name":"fabric-ca"}]'::jsonb,
        2,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009010,
        910000000000008005,
        910000000000006008,  -- couchdb v3.3
        'couchdb',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":5984,"service_port":5984,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        2,
        FALSE,
        3,
        NOW(),
        NOW()
    ),
    (
        910000000000009011,
        910000000000008005,
        910000000000006005,  -- fabric-peer v2.5
        'fabric-peer',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":7051,"service_port":7051,"protocol":"tcp"},{"container_port":7053,"service_port":7053,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[{"container_name":"fabric-orderer"},{"container_name":"couchdb"}]'::jsonb,
        3,
        FALSE,
        4,
        NOW(),
        NOW()
    ),
    (
        910000000000009012,
        910000000000008005,
        910000000000006013,  -- fabric-tools v2.5
        'fabric-tools',
        1,
        NULL,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[{"container_name":"fabric-peer"}]'::jsonb,
        4,
        FALSE,
        5,
        NOW(),
        NOW()
    ),
    (
        910000000000009013,
        910000000000008005,
        910000000000006016,  -- go-dev v1.0
        'go-dev',
        1,
        NULL,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"fabric-peer"}]'::jsonb,
        5,
        TRUE,
        6,
        NOW(),
        NOW()
    ),

    -- ---------- 模板 6：EVM 全栈 DApp（topology_mode=1，多容器） ----------
    -- 启动顺序：geth(1) → blockscout(2) + remix-ide(2) → dapp-dev(3)
    (
        910000000000009014,
        910000000000008006,
        910000000000006002,  -- geth v1.14
        'geth',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8545,"service_port":8545,"protocol":"tcp"},{"container_port":8546,"service_port":8546,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[]'::jsonb,
        1,
        FALSE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000009015,
        910000000000008006,
        910000000000006003,  -- blockscout v6.3
        'blockscout',
        1,
        NULL,
        '[{"key":"ETHEREUM_JSONRPC_HTTP_URL","value":"http://geth:8545","desc":"EVM 节点 RPC 地址","conditions":[]}]'::jsonb,
        '[{"container_port":4000,"service_port":4000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"geth"}]'::jsonb,
        2,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009016,
        910000000000008006,
        910000000000006012,  -- remix-ide latest
        'remix-ide',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8080,"service_port":8080,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[]'::jsonb,
        2,
        FALSE,
        3,
        NOW(),
        NOW()
    ),
    (
        910000000000009017,
        910000000000008006,
        910000000000006017,  -- dapp-dev v1.0
        'dapp-dev',
        1,
        NULL,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"geth"}]'::jsonb,
        3,
        TRUE,
        4,
        NOW(),
        NOW()
    ),

    -- ---------- 模板 7：Fabric 多人协作（topology_mode=3） ----------
    -- 共享容器（role_id=NULL）：fabric-ca, fabric-tools, go-dev
    -- 角色专属容器（role_id 绑定）：
    --   Org1 管理员 → peer-org1 + couchdb-org1
    --   Org2 管理员 → peer-org2 + couchdb-org2
    --   Orderer 运维 → orderer
    -- 启动顺序：CA(1) → Orderer(2) + CouchDB×2(2) → Peer×2(3) → tools(4) → go-dev(5)

    -- 共享 fabric-ca
    (
        910000000000009018,
        910000000000008007,
        910000000000006007,  -- fabric-ca v1.5
        'shared-ca',
        2,
        NULL,
        '[]'::jsonb,
        '[{"container_port":7054,"service_port":7054,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        1,
        FALSE,
        1,
        NOW(),
        NOW()
    ),
    -- Orderer 运维 → orderer
    (
        910000000000009019,
        910000000000008007,
        910000000000006006,  -- fabric-orderer v2.5
        'orderer',
        1,
        910000000000020003,  -- role: Orderer 运维
        '[]'::jsonb,
        '[{"container_port":7050,"service_port":7050,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[{"container_name":"shared-ca"}]'::jsonb,
        2,
        TRUE,
        2,
        NOW(),
        NOW()
    ),
    -- Org1 管理员 → couchdb-org1
    (
        910000000000009020,
        910000000000008007,
        910000000000006008,  -- couchdb v3.3
        'couchdb-org1',
        1,
        910000000000020001,  -- role: Org1 管理员
        '[]'::jsonb,
        '[{"container_port":5984,"service_port":5984,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        2,
        FALSE,
        3,
        NOW(),
        NOW()
    ),
    -- Org1 管理员 → peer-org1
    (
        910000000000009021,
        910000000000008007,
        910000000000006005,  -- fabric-peer v2.5
        'peer-org1',
        1,
        910000000000020001,  -- role: Org1 管理员
        '[{"key":"CORE_PEER_ID","value":"peer0.org1.example.com","desc":"Peer 节点 ID","conditions":[]}]'::jsonb,
        '[{"container_port":7051,"service_port":7051,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[{"container_name":"orderer"},{"container_name":"couchdb-org1"}]'::jsonb,
        3,
        TRUE,
        4,
        NOW(),
        NOW()
    ),
    -- Org2 管理员 → couchdb-org2
    (
        910000000000009022,
        910000000000008007,
        910000000000006008,  -- couchdb v3.3
        'couchdb-org2',
        1,
        910000000000020002,  -- role: Org2 管理员
        '[]'::jsonb,
        '[{"container_port":5984,"service_port":5984,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        2,
        FALSE,
        5,
        NOW(),
        NOW()
    ),
    -- Org2 管理员 → peer-org2
    (
        910000000000009023,
        910000000000008007,
        910000000000006005,  -- fabric-peer v2.5
        'peer-org2',
        1,
        910000000000020002,  -- role: Org2 管理员
        '[{"key":"CORE_PEER_ID","value":"peer0.org2.example.com","desc":"Peer 节点 ID","conditions":[]}]'::jsonb,
        '[{"container_port":7051,"service_port":7051,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[{"container_name":"orderer"},{"container_name":"couchdb-org2"}]'::jsonb,
        3,
        TRUE,
        6,
        NOW(),
        NOW()
    ),
    -- 共享 fabric-tools
    (
        910000000000009024,
        910000000000008007,
        910000000000006013,  -- fabric-tools v2.5
        'shared-tools',
        2,
        NULL,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[{"container_name":"peer-org1"},{"container_name":"peer-org2"}]'::jsonb,
        4,
        FALSE,
        7,
        NOW(),
        NOW()
    ),
    -- 共享 go-dev 开发环境
    (
        910000000000009025,
        910000000000008007,
        910000000000006016,  -- go-dev v1.0
        'go-dev',
        2,
        NULL,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"peer-org1"},{"container_name":"peer-org2"}]'::jsonb,
        5,
        FALSE,
        8,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 08. 模板检查点
-- =====================================================================

INSERT INTO template_checkpoints (
    id, template_id, title, description, check_type, script_content, script_language,
    target_container, assertion_config, score, scope, sort_order, created_at, updated_at
)
VALUES
    -- 模板 5 检查点
    (
        910000000000010005,
        910000000000008005,
        'Fabric CA 启动与证书生成验证',
        '验证 CA 服务正常启动，组织根证书与管理员证书生成成功。',
        2,
        NULL, NULL, NULL, NULL,
        30,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010006,
        910000000000008005,
        'Peer 节点入网与通道加入验证',
        '验证 Peer 已连接 Orderer 并成功加入指定通道。',
        2,
        NULL, NULL, NULL, NULL,
        40,
        1,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000010007,
        910000000000008005,
        '链码部署与调用验证',
        '验证链码在 Peer 上安装、实例化并能被成功调用。',
        2,
        NULL, NULL, NULL, NULL,
        30,
        1,
        3,
        NOW(),
        NOW()
    ),
    -- 模板 6 检查点
    (
        910000000000010008,
        910000000000008006,
        '合约部署与浏览器查看验证',
        '合约部署到 geth 后可在 Blockscout 中查看交易和合约地址。',
        2,
        NULL, NULL, NULL, NULL,
        40,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010009,
        910000000000008006,
        'DApp 前端与合约交互验证',
        'DApp 前端能够通过 ethers.js/web3.js 连接 geth 节点并调用合约方法。',
        2,
        NULL, NULL, NULL, NULL,
        60,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 7 检查点
    (
        910000000000010010,
        910000000000008007,
        '多组织网络联通验证',
        '各组织 Peer 成功加入同一通道，Orderer 排序服务正常，CA 证书互认。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        2,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010011,
        910000000000008007,
        '跨组织链码调用验证',
        '任一组织发起的交易能被对方组织的 Peer 背书并提交。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        2,
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 09. 课程与实验关联
-- =====================================================================

-- 为课程 1（区块链工程实践导论）添加新章节
INSERT INTO chapters (id, course_id, title, description, sort_order, created_at, updated_at)
VALUES
    (
        910000000000007103,
        910000000000007001,
        'Fabric 网络搭建与多容器编排',
        '介绍 Fabric 多节点网络拓扑与 EVM 全栈 DApp 开发环境编排。',
        3,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- 为课程 3（链上数据分析实践）添加新章节
INSERT INTO chapters (id, course_id, title, description, sort_order, created_at, updated_at)
VALUES
    (
        910000000000007302,
        910000000000007003,
        '多人协作与跨组织交易',
        '通过 Fabric 多组织协作实验理解跨组织交易流程与证书管理。',
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- 课时与实验关联
INSERT INTO lessons (
    id, chapter_id, course_id, title, content_type, content, experiment_id, sort_order, estimated_minutes, created_at, updated_at
)
VALUES
    (
        910000000000011004,
        910000000000007103,
        910000000000007001,
        'Fabric 单人多节点网络搭建',
        3,
        '课时聚焦 Fabric CA + Orderer + Peer + CouchDB 的完整编排与链码操作。',
        910000000000008005,
        1,
        90,
        NOW(),
        NOW()
    ),
    (
        910000000000011005,
        910000000000007103,
        910000000000007001,
        'EVM 全栈 DApp 开发实操',
        3,
        '课时聚焦 geth + Blockscout + Remix + DApp 开发环境的多容器协同。',
        910000000000008006,
        2,
        60,
        NOW(),
        NOW()
    ),
    (
        910000000000011302,
        910000000000007302,
        910000000000007003,
        'Fabric 多人协作组网',
        3,
        '课时聚焦多角色分工下的 Fabric 多组织网络搭建与跨组织交易。',
        910000000000008007,
        1,
        90,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- 课程实验关联
INSERT INTO course_experiments (id, course_id, experiment_id, title, sort_order, created_at)
VALUES
    (
        910000000000012005,
        910000000000007001,
        910000000000008005,
        '课程实验：Fabric 单人多节点网络搭建',
        3,
        NOW()
    ),
    (
        910000000000012006,
        910000000000007001,
        910000000000008006,
        '课程实验：EVM 全栈 DApp 开发',
        4,
        NOW()
    ),
    (
        910000000000012007,
        910000000000007003,
        910000000000008007,
        '课程实验：Fabric 多人协作组网',
        2,
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 10. 标签（便于前端筛选）
-- =====================================================================

INSERT INTO tags (id, name, category, color, is_system, created_at)
VALUES
    (910000000000030001, 'Ethereum',  'ecosystem', '#627EEA', TRUE, NOW()),
    (910000000000030002, 'Fabric',    'ecosystem', '#2D9CDB', TRUE, NOW()),
    (910000000000030003, '多节点',    'topology',  '#F2994A', TRUE, NOW()),
    (910000000000030004, '多人协作',  'topology',  '#EB5757', TRUE, NOW()),
    (910000000000030005, '共享基础设施', 'topology', '#6FCF97', TRUE, NOW()),
    (910000000000030006, '全栈开发',  'topic',     '#9B51E0', TRUE, NOW()),
    (910000000000030007, '网络搭建',  'topic',     '#F2C94C', TRUE, NOW()),
    (910000000000030008, '安全分析',  'topic',     '#FF6B6B', TRUE, NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO template_tags (id, template_id, tag_id, created_at)
VALUES
    -- 模板 1（以太坊本地开发）
    (910000000000031001, 910000000000008001, 910000000000030001, NOW()),
    -- 模板 2（共享链基础设施）
    (910000000000031002, 910000000000008002, 910000000000030001, NOW()),
    (910000000000031003, 910000000000008002, 910000000000030005, NOW()),
    -- 模板 3（漏洞分析）
    (910000000000031004, 910000000000008003, 910000000000030001, NOW()),
    (910000000000031005, 910000000000008003, 910000000000030008, NOW()),
    -- 模板 4（链上数据）
    (910000000000031006, 910000000000008004, 910000000000030001, NOW()),
    (910000000000031007, 910000000000008004, 910000000000030005, NOW()),
    -- 模板 5（Fabric 单人多节点）
    (910000000000031008, 910000000000008005, 910000000000030002, NOW()),
    (910000000000031009, 910000000000008005, 910000000000030003, NOW()),
    (910000000000031010, 910000000000008005, 910000000000030007, NOW()),
    -- 模板 6（EVM 全栈 DApp）
    (910000000000031011, 910000000000008006, 910000000000030001, NOW()),
    (910000000000031012, 910000000000008006, 910000000000030006, NOW()),
    -- 模板 7（Fabric 多人协作）
    (910000000000031013, 910000000000008007, 910000000000030002, NOW()),
    (910000000000031014, 910000000000008007, 910000000000030004, NOW()),
    (910000000000031015, 910000000000008007, 910000000000030007, NOW())
ON CONFLICT (id) DO NOTHING;
