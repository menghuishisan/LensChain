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
-- 注：images / image_versions 数据来自 deploy/images/<category>/<name>/manifest.yaml，
-- 通过 cmd/seed-manifests CLI（或 admin API POST /api/v1/admin/images/sync）灌入。
-- init-db 脚本会在本 seed 执行前调用 CLI 完成同步。
-- 本文件下方的 template_containers 通过 (image_name, version) 子查询关联 image_version，
-- 不再硬编码 image_version_id。


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
        E'1. 启动 CA 服务并注册组织身份\n2. 启动 Orderer 排序节点\n3. 启动 CouchDB 状态数据库\n4. 启动 Peer 节点并加入通道\n5. 使用 CLI 工具部署链码\n6. 在开发环境中调用链码并验证',
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
        E'1. 启动 geth 本地开发链\n2. 在 Remix IDE 中编写并部署合约\n3. 在 Blockscout 中确认部署结果\n4. 在 DApp 开发环境中编写前端并连接合约',
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
        E'1. 共享 CA 启动并为各组织生成证书\n2. Orderer 运维角色启动排序节点并创建通道\n3. Org1 管理员启动 Peer + CouchDB 并加入通道\n4. Org2 管理员启动 Peer + CouchDB 并加入通道\n5. 通过共享 CLI 工具完成链码部署\n6. 各组织使用开发环境调用链码并验证跨组织交易',
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-ca' AND iv.version = '1.5'),  -- fabric-ca v1.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-orderer' AND iv.version = '2.5'),  -- fabric-orderer v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'couchdb' AND iv.version = '3.3'),  -- couchdb v3.3
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-peer' AND iv.version = '2.5'),  -- fabric-peer v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-tools' AND iv.version = '2.5'),  -- fabric-tools v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'go-dev' AND iv.version = '1.0'),  -- go-dev v1.0
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'geth' AND iv.version = '1.14'),  -- geth v1.14
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
    -- Blockscout 是 Phoenix(Elixir) 应用，启动时强制要求 DATABASE_URL 与 SECRET_KEY_BASE，
    -- 否则容器立即 Completed（exit 0）。manifest required_dependencies 已声明依赖 postgres，
    -- 这里把 env / depends_on 与依赖容器一并补齐。
    (
        910000000000009015,
        910000000000008006,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'blockscout' AND iv.version = '6.3'),  -- blockscout v6.3
        'blockscout',
        1,
        NULL,
        '[{"key":"ETHEREUM_JSONRPC_HTTP_URL","value":"http://geth:8545","desc":"EVM 节点 RPC 地址","conditions":[]},{"key":"ETHEREUM_JSONRPC_VARIANT","value":"geth","desc":"上游节点类型","conditions":[]},{"key":"DATABASE_URL","value":"postgresql://postgres:postgres@postgres:5432/blockscout","desc":"Postgres 连接串","conditions":[]},{"key":"ECTO_USE_SSL","value":"false","desc":"教学环境关闭 TLS","conditions":[]},{"key":"SECRET_KEY_BASE","value":"VTIzNDU2Nzg5MGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVoxMjM0NTY3ODkw","desc":"Phoenix 会话签名密钥（教学环境固定）","conditions":[]}]'::jsonb,
        '[{"container_port":4000,"service_port":4000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"geth"},{"container_name":"postgres"}]'::jsonb,
        2,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 6 的 Postgres 容器：blockscout 持久化所需，与 geth 同 startup_order=1 并行启动。
    (
        910000000000009033,
        910000000000008006,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'postgres' AND iv.version = '15'),
        'postgres',
        1,
        NULL,
        '[{"key":"POSTGRES_PASSWORD","value":"postgres","desc":"Postgres 超级用户密码","conditions":[]},{"key":"POSTGRES_DB","value":"blockscout","desc":"Blockscout 使用的数据库","conditions":[]},{"key":"POSTGRES_USER","value":"postgres","desc":"Postgres 超级用户","conditions":[]}]'::jsonb,
        '[{"container_port":5432,"service_port":5432,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[]'::jsonb,
        1,
        FALSE,
        5,
        NOW(),
        NOW()
    ),
    (
        910000000000009016,
        910000000000008006,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'remix-ide' AND iv.version = 'latest'),  -- remix-ide latest
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'dapp-dev' AND iv.version = '1.0'),  -- dapp-dev v1.0
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-ca' AND iv.version = '1.5'),  -- fabric-ca v1.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-orderer' AND iv.version = '2.5'),  -- fabric-orderer v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'couchdb' AND iv.version = '3.3'),  -- couchdb v3.3
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-peer' AND iv.version = '2.5'),  -- fabric-peer v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'couchdb' AND iv.version = '3.3'),  -- couchdb v3.3
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-peer' AND iv.version = '2.5'),  -- fabric-peer v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-tools' AND iv.version = '2.5'),  -- fabric-tools v2.5
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
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'go-dev' AND iv.version = '1.0'),  -- go-dev v1.0
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

-- ---------------------------------------------------------------------
-- 07b. 终端工具容器：为所有真实环境/混合实验模板（experiment_type ∈ {2,3}）
--      统一注入一个 xterm-server 容器，对齐文档 §2.16 终端约束。
--      deployment_scope=1（实例独享，每位学生独立终端，即便其他容器走共享基础设施）
--      role_id=NULL（多人协作模板下所有角色共享）
--      startup_order=99（始终最后启动，等待主容器就绪）
-- ---------------------------------------------------------------------

INSERT INTO template_containers (
    id, template_id, image_version_id, container_name, deployment_scope, role_id,
    env_vars, ports, volumes, cpu_limit, memory_limit, depends_on, startup_order,
    is_primary, sort_order, created_at, updated_at
)
VALUES
    -- 模板 1（以太坊本地开发与部署实践）
    (
        910000000000009026,
        910000000000008001,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),  -- xterm-server v1.0
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 2（共享链基础设施上的 DApp 部署）
    (
        910000000000009027,
        910000000000008002,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 3（智能合约漏洞分析）
    (
        910000000000009028,
        910000000000008003,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 4（链上数据索引与浏览）
    (
        910000000000009029,
        910000000000008004,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 5（Fabric 单人多节点）
    (
        910000000000009030,
        910000000000008005,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 6（EVM 全栈 DApp）
    (
        910000000000009031,
        910000000000008006,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
    ),
    -- 模板 7（Fabric 多人协作）
    (
        910000000000009032,
        910000000000008007,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1, NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m', '128Mi',
        '[]'::jsonb,
        99, FALSE, 99, NOW(), NOW()
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

-- =====================================================================
-- 11. 补充工具类实验模板（使用各种工具镜像）
-- =====================================================================

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, status, created_at, updated_at
)
VALUES
    -- 模板 8：Code-Server 在线开发实验
    (
        910000000000008008,
        910000000000000001,
        910000000000001001,
        'Code-Server 在线开发实验',
        '使用 Code-Server 在线 IDE 完成 Go 语言区块链节点开发。',
        '掌握在线 IDE 的使用，理解区块链节点的基本开发流程。',
        E'1. 启动 Code-Server 容器\n2. 创建 Go 项目并初始化\n3. 编写简单的 P2P 节点代码\n4. 编译并运行测试',
        2,
        1,
        1,
        100,
        60,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 9：Remix-IDE 合约开发实验
    (
        910000000000008009,
        910000000000000001,
        910000000000001001,
        'Remix-IDE 合约开发实验',
        '使用 Remix-IDE 完成 ERC-20 代币合约的开发与测试。',
        '熟悉 Remix-IDE 的各项功能，掌握 Solidity 合约开发基础。',
        E'1. 启动 Remix-IDE 容器\n2. 编写 ERC-20 代币合约\n3. 使用 Remix 编译器编译\n4. 部署到测试网络并测试',
        2,
        1,
        1,
        100,
        45,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 10：Fabric-Explorer 区块链浏览器实验
    (
        910000000000008010,
        910000000000000001,
        910000000000001001,
        'Fabric-Explorer 区块链浏览器实验',
        '部署并配置 Fabric 区块链浏览器，观察 Fabric 网络的交易和区块。',
        '理解 Fabric 网络的数据结构，掌握区块链浏览器的部署与使用。',
        E'1. 启动 Fabric 网络节点\n2. 部署 Fabric-Explorer\n3. 配置 Explorer 连接网络\n4. 观察区块和交易数据',
        2,
        1,
        1,
        100,
        75,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 11：Xterm-Server 终端操作实验
    (
        910000000000008011,
        910000000000000001,
        910000000000001001,
        'Xterm-Server 终端操作实验',
        '通过 Web 终端完成区块链节点的命令行操作。',
        '掌握区块链节点的命令行工具，熟悉常用 CLI 命令。',
        E'1. 启动 xterm-server 容器\n2. 连接到区块链节点\n3. 使用 CLI 查询节点状态\n4. 发送测试交易',
        2,
        1,
        1,
        100,
        30,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 12：Redis 缓存实验
    (
        910000000000008012,
        910000000000000001,
        910000000000001001,
        'Redis 缓存实验',
        '使用 Redis 作为区块链节点的缓存层，提升查询性能。',
        '理解 Redis 在区块链系统中的应用，掌握缓存策略设计。',
        E'1. 启动 Redis 容器\n2. 配置节点连接 Redis\n3. 实现区块数据缓存\n4. 测试缓存命中率',
        2,
        1,
        1,
        100,
        45,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 13：Postgres 数据持久化实验
    (
        910000000000008013,
        910000000000000001,
        910000000000001001,
        'Postgres 数据持久化实验',
        '使用 PostgreSQL 存储区块链交易和状态数据。',
        '理解区块链数据的持久化方案，掌握数据库设计。',
        E'1. 启动 Postgres 容器\n2. 设计数据库表结构\n3. 实现交易数据同步\n4. 查询并分析链上数据',
        2,
        1,
        1,
        100,
        60,
        30,
        1,
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 12. 新增模板的容器配置
-- =====================================================================

INSERT INTO template_containers (
    id, template_id, image_version_id, container_name, deployment_scope, role_id,
    env_vars, ports, volumes, cpu_limit, memory_limit, depends_on, startup_order,
    is_primary, sort_order, created_at, updated_at
)
VALUES
    -- 模板 8：Code-Server
    (
        910000000000009034,
        910000000000008008,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'code-server' AND iv.version = '4.89'),
        'code-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8080,"service_port":8080,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[]'::jsonb,
        1,
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000009035,
        910000000000008008,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[]'::jsonb,
        99,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 9：Remix-IDE
    (
        910000000000009036,
        910000000000008009,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'remix-ide' AND iv.version = 'latest'),
        'remix-ide',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8080,"service_port":8080,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[]'::jsonb,
        1,
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000009037,
        910000000000008009,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[]'::jsonb,
        99,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 10：Fabric-Explorer（需要先启动 Fabric 网络）
    (
        910000000000009038,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-ca' AND iv.version = '1.5'),
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
        910000000000009039,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-orderer' AND iv.version = '2.5'),
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
        910000000000009040,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'couchdb' AND iv.version = '3.3'),
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
        910000000000009041,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-peer' AND iv.version = '2.5'),
        'fabric-peer',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":7051,"service_port":7051,"protocol":"tcp"}]'::jsonb,
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
        910000000000009042,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'fabric-explorer' AND iv.version = '1.1'),
        'fabric-explorer',
        1,
        NULL,
        '[{"key":"DISCOVERY_AS_LOCALHOST","value":"false","desc":"使用容器网络","conditions":[]},{"key":"LOG_LEVEL_APP","value":"info","desc":"日志级别","conditions":[]}]'::jsonb,
        '[{"container_port":8080,"service_port":8080,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"fabric-peer"}]'::jsonb,
        4,
        TRUE,
        5,
        NOW(),
        NOW()
    ),
    (
        910000000000009043,
        910000000000008010,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[]'::jsonb,
        99,
        FALSE,
        6,
        NOW(),
        NOW()
    ),
    -- 模板 11：Xterm-Server（单容器）
    (
        910000000000009044,
        910000000000008011,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'geth' AND iv.version = '1.14'),
        'geth',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8545,"service_port":8545,"protocol":"tcp"}]'::jsonb,
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
        910000000000009045,
        910000000000008011,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[{"container_name":"geth"}]'::jsonb,
        2,
        TRUE,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 12：Redis 缓存
    (
        910000000000009046,
        910000000000008012,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'geth' AND iv.version = '1.14'),
        'geth',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8545,"service_port":8545,"protocol":"tcp"}]'::jsonb,
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
        910000000000009047,
        910000000000008012,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'redis' AND iv.version = '7'),
        'redis',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":6379,"service_port":6379,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '250m',
        '256Mi',
        '[]'::jsonb,
        1,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009048,
        910000000000008012,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[{"container_name":"geth"},{"container_name":"redis"}]'::jsonb,
        2,
        TRUE,
        3,
        NOW(),
        NOW()
    ),
    -- 模板 13：Postgres 数据持久化
    (
        910000000000009049,
        910000000000008013,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'geth' AND iv.version = '1.14'),
        'geth',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":8545,"service_port":8545,"protocol":"tcp"}]'::jsonb,
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
        910000000000009050,
        910000000000008013,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'postgres' AND iv.version = '15'),
        'postgres',
        1,
        NULL,
        '[{"key":"POSTGRES_PASSWORD","value":"postgres","desc":"Postgres 超级用户密码","conditions":[]},{"key":"POSTGRES_DB","value":"blockchain","desc":"区块链数据库","conditions":[]},{"key":"POSTGRES_USER","value":"postgres","desc":"Postgres 超级用户","conditions":[]}]'::jsonb,
        '[{"container_port":5432,"service_port":5432,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '512Mi',
        '[]'::jsonb,
        1,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009051,
        910000000000008013,
        (SELECT iv.id FROM image_versions iv JOIN images i ON iv.image_id = i.id WHERE i.name = 'xterm-server' AND iv.version = '1.0'),
        'xterm-server',
        1,
        NULL,
        '[]'::jsonb,
        '[{"container_port":3000,"service_port":3000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '100m',
        '128Mi',
        '[{"container_name":"geth"},{"container_name":"postgres"}]'::jsonb,
        2,
        TRUE,
        3,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 13. 新增模板的检查点
-- =====================================================================

INSERT INTO template_checkpoints (
    id, template_id, title, description, check_type, script_content, script_language,
    target_container, assertion_config, score, scope, sort_order, created_at, updated_at
)
VALUES
    -- 模板 8 检查点
    (
        910000000000010012,
        910000000000008008,
        'Code-Server 环境验证',
        '验证 Code-Server 正常启动并可访问。',
        2,
        NULL, NULL, NULL, NULL,
        30,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010013,
        910000000000008008,
        'Go 项目编译测试',
        '验证 Go 项目能够正常编译运行。',
        2,
        NULL, NULL, NULL, NULL,
        70,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 9 检查点
    (
        910000000000010014,
        910000000000008009,
        'Remix-IDE 环境验证',
        '验证 Remix-IDE 正常启动并可访问。',
        2,
        NULL, NULL, NULL, NULL,
        40,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010015,
        910000000000008009,
        'ERC-20 合约编译部署',
        '验证 ERC-20 合约能够成功编译并部署。',
        2,
        NULL, NULL, NULL, NULL,
        60,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 10 检查点
    (
        910000000000010016,
        910000000000008010,
        'Fabric 网络启动验证',
        '验证 Fabric 网络节点正常启动并连接。',
        2,
        NULL, NULL, NULL, NULL,
        40,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010017,
        910000000000008010,
        'Fabric-Explorer 数据展示',
        '验证 Explorer 能够正常显示 Fabric 网络数据。',
        2,
        NULL, NULL, NULL, NULL,
        60,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 11 检查点
    (
        910000000000010018,
        910000000000008011,
        '终端连接验证',
        '验证能够通过 Web 终端连接到节点。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010019,
        910000000000008011,
        'CLI 命令执行验证',
        '验证能够通过 CLI 查询节点状态并发送交易。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 12 检查点
    (
        910000000000010020,
        910000000000008012,
        'Redis 连接验证',
        '验证节点能够正常连接到 Redis。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010021,
        910000000000008012,
        '缓存功能验证',
        '验证区块数据能够正常缓存并命中。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        2,
        NOW(),
        NOW()
    ),
    -- 模板 13 检查点
    (
        910000000000010022,
        910000000000008013,
        'Postgres 连接验证',
        '验证节点能够正常连接到 PostgreSQL。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010023,
        910000000000008013,
        '数据持久化验证',
        '验证交易数据能够正常同步到数据库。',
        2,
        NULL, NULL, NULL, NULL,
        50,
        1,
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 14. 新增标签
-- =====================================================================

INSERT INTO tags (id, name, category, color, is_system, created_at)
VALUES
    (910000000000030009, '在线开发', 'topic', '#8B5CF6', TRUE, NOW()),
    (910000000000030010, 'IDE', 'tool', '#EC4899', TRUE, NOW()),
    (910000000000030011, '缓存', 'topic', '#F59E0B', TRUE, NOW()),
    (910000000000030012, '数据库', 'topic', '#10B981', TRUE, NOW()),
    (910000000000030013, '终端', 'tool', '#6366F1', TRUE, NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 15. 新增模板的标签关联
-- =====================================================================

INSERT INTO template_tags (id, template_id, tag_id, created_at)
VALUES
    -- 模板 8（Code-Server）
    (910000000000031016, 910000000000008008, 910000000000030009, NOW()),
    (910000000000031017, 910000000000008008, 910000000000030010, NOW()),
    -- 模板 9（Remix-IDE）
    (910000000000031018, 910000000000008009, 910000000000030009, NOW()),
    (910000000000031019, 910000000000008009, 910000000000030010, NOW()),
    (910000000000031020, 910000000000008009, 910000000000030001, NOW()),
    -- 模板 10（Fabric-Explorer）
    (910000000000031021, 910000000000008010, 910000000000030002, NOW()),
    (910000000000031022, 910000000000008010, 910000000000030007, NOW()),
    -- 模板 11（Xterm-Server）
    (910000000000031023, 910000000000008011, 910000000000030013, NOW()),
    (910000000000031024, 910000000000008011, 910000000000030001, NOW()),
    -- 模板 12（Redis）
    (910000000000031025, 910000000000008012, 910000000000030011, NOW()),
    (910000000000031026, 910000000000008012, 910000000000030001, NOW()),
    -- 模板 13（Postgres）
    (910000000000031027, 910000000000008013, 910000000000030012, NOW()),
    (910000000000031028, 910000000000008013, 910000000000030001, NOW())
ON CONFLICT (id) DO NOTHING;
