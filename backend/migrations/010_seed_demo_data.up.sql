-- Demo seed data for LensChain.
-- 目标：
-- 1. 提供一套可重复导入的可信演示 / 联调数据
-- 2. 避免“张三李四 / 13800138000”这类测试味很重的数据
-- 3. 覆盖登录、课程、实验模板、共享基础设施模板与选课链路
--
-- 使用方式：
-- 1. 在空库初始化完成后执行本文件
-- 2. 若需要重置，可先清理对应业务数据，再重新执行本文件
--
-- 约定：
-- 1. 本文件只插入 demo 业务数据，不删除任何现有业务数据
-- 2. 所有 ID 使用固定值，便于联调和前后端说明文档引用

-- ---------------------------------------------------------------------------
-- 01. 学校
-- ---------------------------------------------------------------------------

INSERT INTO schools (
    id, name, code, logo_url, address, website, description, status,
    license_start_at, license_end_at, contact_name, contact_phone, contact_email, contact_title,
    created_at, updated_at
)
VALUES (
    910000000000000001,
    '江海数字科技学院',
    'jianghai-digital-tech',
    NULL,
    '江苏省苏州市工业园区星海路 18 号',
    'https://www.jhdt.edu.cn',
    '面向区块链工程与数字系统安全课程的演示学校数据。',
    2,
    NOW(),
    NOW() + INTERVAL '365 days',
    '周闻笙',
    '13761284539',
    'operations@jhdt.edu.cn',
    '实验教学平台主管',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO schools (
    id, name, code, logo_url, address, website, description, status,
    license_start_at, license_end_at, contact_name, contact_phone, contact_email, contact_title,
    created_at, updated_at
)
VALUES (
    910000000000000002,
    '云浦工程实验学院',
    'yunpu-engineering-labs',
    NULL,
    '浙江省杭州市滨江区启智路 66 号',
    'https://www.ype.edu.cn',
    '侧重智能合约工程、链上数据分析与课程实训的演示学校。',
    2,
    NOW(),
    NOW() + INTERVAL '240 days',
    '孟时越',
    '13577120468',
    'teaching@ype.edu.cn',
    '教学平台主管',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 02. 角色
-- ---------------------------------------------------------------------------

INSERT INTO roles (id, code, name, description, is_system, created_at, updated_at)
VALUES
    (910000000000000101, 'teacher', '教师', '课程与实验组织者', TRUE, NOW(), NOW()),
    (910000000000000102, 'student', '学生', '课程学习与实验参与者', TRUE, NOW(), NOW()),
    (910000000000000103, 'school_admin', '学校管理员', '学校租户管理者', TRUE, NOW(), NOW()),
    (910000000000000104, 'super_admin', '超级管理员', '平台级系统管理者', TRUE, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 03. 用户
-- ---------------------------------------------------------------------------

-- 统一密码说明：
-- 1. 所有 demo 账号密码统一为：LensChain2026
-- 2. 密码已使用 bcrypt 预先生成。
-- 3. bcrypt hash = $2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC

INSERT INTO users (
    id, phone, password_hash, name, school_id, student_no, status,
    is_first_login, is_school_admin, token_valid_after, created_at, updated_at
)
VALUES
    (
        910000000000001001,
        '13761284539',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '沈砚秋',
        910000000000000001,
        'TCH2026001',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001002,
        '13651827461',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '顾承礼',
        910000000000000001,
        'ADM2026001',
        1,
        FALSE,
        TRUE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001101,
        '13927461582',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '林叙安',
        910000000000000001,
        'BC240301',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001102,
        '13872945160',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '许知遥',
        910000000000000001,
        'BC240302',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001201,
        '13577120468',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '孟时越',
        910000000000000002,
        'ADM2026201',
        1,
        FALSE,
        TRUE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001202,
        '13622054819',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '姚见川',
        910000000000000002,
        'TCH2026201',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001203,
        '13790811624',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '宋观澜',
        910000000000000002,
        'BC240901',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001204,
        '13816650372',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '杜霁青',
        910000000000000002,
        'BC240902',
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    ),
    (
        910000000000001900,
        '13988001234',
        '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
        '平台演示管理员',
        0,
        NULL,
        1,
        FALSE,
        FALSE,
        NOW(),
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_profiles (
    id, user_id, email, college, major, class_name, enrollment_year, education_level, grade, remark, created_at, updated_at
)
VALUES
    (910000000000002001, 910000000000001001, 'yanqiu.shen@jhdt.edu.cn', '区块链工程学院', '区块链工程', NULL, NULL, NULL, NULL, '负责链上实验课程与实验环境维护。', NOW(), NOW()),
    (910000000000002002, 910000000000001002, 'chengli.gu@jhdt.edu.cn', '信息化建设中心', '教育技术', NULL, NULL, NULL, NULL, '学校管理员演示账号。', NOW(), NOW()),
    (910000000000002101, 910000000000001101, 'xuan.lin@jhdt.edu.cn', '区块链工程学院', '区块链工程', '链工 2403', 2024, 2, 2024, '偏合约安全方向。', NOW(), NOW()),
    (910000000000002102, 910000000000001102, 'zhiyao.xu@jhdt.edu.cn', '区块链工程学院', '区块链工程', '链工 2403', 2024, 2, 2024, '偏链节点运维方向。', NOW(), NOW()),
    (910000000000002201, 910000000000001201, 'shiyue.meng@ype.edu.cn', '工程实验中心', '教育技术', NULL, NULL, NULL, NULL, '第二所学校管理员演示账号。', NOW(), NOW()),
    (910000000000002202, 910000000000001202, 'jianchuan.yao@ype.edu.cn', '智能合约学院', '智能合约工程', NULL, NULL, NULL, NULL, '负责云浦学院课程与实验设计。', NOW(), NOW()),
    (910000000000002203, 910000000000001203, 'guanlan.song@ype.edu.cn', '智能合约学院', '智能合约工程', '合约 2409', 2024, 2, 2024, '偏链上安全审计方向。', NOW(), NOW()),
    (910000000000002204, 910000000000001204, 'jiqing.du@ype.edu.cn', '链上数据学院', '区块链数据工程', '数据 2409', 2024, 2, 2024, '偏数据索引与 The Graph 场景。', NOW(), NOW()),
    (910000000000002900, 910000000000001900, 'platform-admin@lenschain.local', '平台运营中心', '平台治理', NULL, NULL, NULL, NULL, '超级管理员演示账号。', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, created_at)
VALUES
    (910000000000003001, 910000000000001001, 910000000000000101, NOW()),
    (9100000000000030011, 910000000000001002, 910000000000000101, NOW()),
    (910000000000003002, 910000000000001002, 910000000000000103, NOW()),
    (9100000000000032011, 910000000000001201, 910000000000000101, NOW()),
    (910000000000003101, 910000000000001101, 910000000000000102, NOW()),
    (910000000000003102, 910000000000001102, 910000000000000102, NOW()),
    (910000000000003201, 910000000000001201, 910000000000000103, NOW()),
    (910000000000003202, 910000000000001202, 910000000000000101, NOW()),
    (910000000000003203, 910000000000001203, 910000000000000102, NOW()),
    (910000000000003204, 910000000000001204, 910000000000000102, NOW()),
    (910000000000003900, 910000000000001900, 910000000000000104, NOW())
ON CONFLICT (id) DO NOTHING;

UPDATE users
SET password_hash = '$2a$12$JN672ikmHzqJMmr1HLBthe3Qtznkj9jlsHu2DhW4pgRvSYggmV2iC',
    updated_at = NOW()
WHERE id IN (
    910000000000001001,
    910000000000001002,
    910000000000001101,
    910000000000001102,
    910000000000001201,
    910000000000001202,
    910000000000001203,
    910000000000001204,
    910000000000001900
);

-- ---------------------------------------------------------------------------
-- 03A. 学校 SSO 配置
-- ---------------------------------------------------------------------------

INSERT INTO school_sso_configs (
    id, school_id, provider, is_enabled, is_tested, config, tested_at, created_at, updated_at, updated_by
)
VALUES
    (
        910000000000003501,
        910000000000000001,
        'cas',
        TRUE,
        TRUE,
        '{
          "cas_server_url": "https://sso.jhdt.edu.cn/cas",
          "cas_service_url": "http://localhost:3000/auth/sso/callback",
          "cas_version": "3.0",
          "user_id_attribute": "studentNo"
        }'::jsonb,
        NOW(),
        NOW(),
        NOW(),
        910000000000001002
    ),
    (
        910000000000003502,
        910000000000000002,
        'oauth2',
        TRUE,
        TRUE,
        '{
          "authorize_url": "https://passport.ype.edu.cn/oauth2/authorize",
          "token_url": "https://passport.ype.edu.cn/oauth2/token",
          "userinfo_url": "https://passport.ype.edu.cn/oauth2/userinfo",
          "client_id": "lenschain-demo-ype",
          "client_secret": "******",
          "redirect_uri": "http://localhost:3000/auth/sso/callback",
          "scope": "openid profile",
          "user_id_attribute": "student_no"
        }'::jsonb,
        NOW(),
        NOW(),
        NOW(),
        910000000000001201
    )
ON CONFLICT (school_id) DO UPDATE
SET provider = EXCLUDED.provider,
    is_enabled = EXCLUDED.is_enabled,
    is_tested = EXCLUDED.is_tested,
    config = EXCLUDED.config,
    tested_at = EXCLUDED.tested_at,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by;

-- ---------------------------------------------------------------------------
-- 04. 镜像分类 / 镜像 / 镜像版本
-- ---------------------------------------------------------------------------

INSERT INTO image_categories (id, name, code, description, sort_order, created_at, updated_at)
VALUES
    (910000000000004001, '基础开发环境', 'base', '开发环境与工具基础镜像', 1, NOW(), NOW()),
    (910000000000004002, '链节点', 'chain-nodes', '链节点与协议运行镜像', 2, NOW(), NOW()),
    (910000000000004003, '区块链中间件', 'middleware', '部署、索引与链上调试中间件镜像', 3, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO images (
    id, category_id, name, display_name, description, ecosystem, source_type, status,
    default_ports, default_env_vars, default_volumes, typical_companions, required_dependencies,
    resource_recommendation, documentation_url, usage_count, created_at, updated_at
)
VALUES
    (
        910000000000005001,
        910000000000004001,
        'solidity-dev',
        'Solidity Development Workspace',
        '用于智能合约编写、编译与调试的开发工作空间。',
        'ethereum',
        1,
        1,
        '[]'::jsonb,
        '[{"key":"SOLC_VERSION","value":"0.8.25","desc":"Solidity 编译器版本","conditions":[]}]'::jsonb,
        '[{"path":"/home/developer/project","desc":"项目工作目录"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"geth","reason":"连接本地开发链进行部署与调试"}],"optional":[{"image":"blockscout","reason":"查看区块和交易"}]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"1Gi","disk":"5Gi"}'::jsonb,
        '/docs/images/solidity-dev',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005002,
        910000000000004002,
        'geth',
        'Go-Ethereum Node',
        '用于教学演示和 DApp 联调的本地以太坊节点。',
        'ethereum',
        1,
        1,
        '[{"port":8545,"protocol":"tcp","name":"HTTP-RPC"},{"port":8546,"protocol":"tcp","name":"WebSocket-RPC"},{"port":30303,"protocol":"tcp","name":"P2P"}]'::jsonb,
        '[{"key":"GETH_NETWORK","value":"dev","desc":"运行网络","conditions":[]}]'::jsonb,
        '[{"path":"/root/.ethereum","desc":"链数据目录"}]'::jsonb,
        '{"required":[],"recommended":[{"image":"solidity-dev","reason":"与开发工作空间配合完成合约部署"}],"optional":[{"image":"blockscout","reason":"区块浏览器"}]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"1Gi","disk":"10Gi"}'::jsonb,
        '/docs/images/geth',
        0,
        NOW(),
        NOW()
    ),
    (
        910000000000005003,
        910000000000004003,
        'blockscout',
        'Blockscout Explorer',
        '用于教学环境的区块浏览器，便于学生观察交易与区块状态。',
        'ethereum',
        1,
        1,
        '[{"port":4000,"protocol":"tcp","name":"Web UI"}]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '{"required":[{"image":"geth","reason":"依赖链节点提供区块与交易数据"}],"recommended":[],"optional":[]}'::jsonb,
        '[]'::jsonb,
        '{"cpu":"0.5","memory":"1Gi","disk":"8Gi"}'::jsonb,
        '/docs/images/blockscout',
        0,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO image_versions (
    id, image_id, version, registry_url, min_cpu, min_memory, min_disk, is_default, status, created_at, updated_at
)
VALUES
    (
        910000000000006001,
        910000000000005001,
        '1.0',
        'registry.lianjing.com/base/solidity-dev:v1.0.0',
        '250m',
        '512Mi',
        '5Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000006002,
        910000000000005002,
        '1.14',
        'registry.lianjing.com/chain-nodes/geth:v1.14.0',
        '250m',
        '512Mi',
        '10Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000006003,
        910000000000005003,
        '6.8',
        'registry.lianjing.com/middleware/blockscout:v6.8.0',
        '500m',
        '1Gi',
        '8Gi',
        TRUE,
        1,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 05. 课程、章节、课时与选课
-- ---------------------------------------------------------------------------

INSERT INTO courses (
    id, school_id, teacher_id, title, description, course_type, difficulty, topic,
    status, is_shared, invite_code, start_at, end_at, credits, max_students, created_at, updated_at
)
VALUES (
    910000000000007001,
    910000000000000001,
    910000000000001001,
    '区块链工程实践导论',
    '围绕节点部署、合约开发、共享链基础设施与实验协作展开的实践课程。',
    1,
    2,
    'blockchain-engineering',
    3,
    FALSE,
    'B3C9H7',
    NOW() - INTERVAL '7 days',
    NOW() + INTERVAL '120 days',
    2.0,
    60,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO courses (
    id, school_id, teacher_id, title, description, course_type, difficulty, topic,
    status, is_shared, invite_code, start_at, end_at, credits, max_students, created_at, updated_at
)
VALUES
    (
        910000000000007002,
        910000000000000001,
        910000000000001001,
        '智能合约安全基础',
        '围绕 Solidity 开发规范、常见漏洞模式和攻防思维展开的课程。',
        1,
        3,
        'smart-contract-security',
        3,
        TRUE,
        'C7S9Q2',
        NOW() - INTERVAL '7 days',
        NOW() + INTERVAL '90 days',
        2.5,
        80,
        NOW(),
        NOW()
    ),
    (
        910000000000007003,
        910000000000000002,
        910000000000001202,
        '链上数据分析实践',
        '面向事件索引、区块浏览与链上数据查询的课程演示数据。',
        1,
        2,
        'onchain-data-analytics',
        3,
        FALSE,
        'D4A8N1',
        NOW() - INTERVAL '7 days',
        NOW() + INTERVAL '100 days',
        2.0,
        70,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO chapters (id, course_id, title, description, sort_order, created_at, updated_at)
VALUES
    (
        910000000000007101,
        910000000000007001,
        '链上环境与开发工作流',
        '介绍教学用链节点、开发工作区与共享基础设施实验的使用方式。',
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000007102,
        910000000000007001,
        '共享基础设施协作实验',
        '聚焦共享链节点、浏览器、中间件之间的协同关系。',
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000007201,
        910000000000007002,
        '漏洞模式与修复策略',
        '从教学案例中理解常见智能合约漏洞及修复方案。',
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000007301,
        910000000000007003,
        '事件索引与区块浏览',
        '围绕链上事件、浏览器和索引工具构建数据分析基础。',
        1,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 06. 实验模板
-- ---------------------------------------------------------------------------

INSERT INTO experiment_templates (
    id, school_id, teacher_id, title, description, objectives, instructions,
    experiment_type, topology_mode, judge_mode, total_score, max_duration, idle_timeout,
    score_strategy, status, created_at, updated_at
)
VALUES
    (
        910000000000008001,
        910000000000000001,
        910000000000001001,
        '以太坊本地开发与部署实践',
        '学生在个人开发环境中完成基础合约编写与部署。',
        '熟悉本地开发工作流，理解合约编译、部署与调用过程。',
        '1. 进入开发环境\n2. 编译合约\n3. 连接本地链节点完成部署',
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
    (
        910000000000008002,
        910000000000000001,
        910000000000001001,
        '共享链基础设施上的 DApp 部署实验',
        '全班共享一套以太坊链节点，每位学生拥有独立开发工作区。',
        '理解共享链基础设施与个人开发环境之间的关系，并完成一次真实的合约部署。',
        '1. 平台先准备共享链节点\n2. 学生启动个人开发环境\n3. 通过共享 RPC 地址部署并验证合约',
        2,
        4,
        1,
        100,
        90,
        30,
        1,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000008003,
        910000000000000001,
        910000000000001001,
        '智能合约漏洞分析实验',
        '学生基于示例合约识别漏洞、修复并提交分析结论。',
        '理解常见漏洞模式，形成安全修复与验证意识。',
        '1. 阅读示例合约\n2. 识别漏洞\n3. 编写修复方案\n4. 完成验证',
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
    (
        910000000000008004,
        910000000000000002,
        910000000000001202,
        '链上数据索引与浏览实验',
        '通过共享链节点与浏览器观察交易、事件与合约状态。',
        '理解链上数据从节点到浏览器的呈现路径。',
        '1. 启动区块浏览器\n2. 部署合约\n3. 查询事件和交易数据',
        2,
        4,
        1,
        100,
        80,
        30,
        1,
        2,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO template_containers (
    id, template_id, image_version_id, container_name, deployment_scope,
    env_vars, ports, volumes, cpu_limit, memory_limit, depends_on, startup_order,
    is_primary, sort_order, created_at, updated_at
)
VALUES
    (
        910000000000009001,
        910000000000008001,
        910000000000006001,
        'solidity-workspace',
        1,
        '[]'::jsonb,
        '[]'::jsonb,
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
        910000000000009002,
        910000000000008002,
        910000000000006002,
        'shared-geth',
        2,
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
        910000000000009003,
        910000000000008002,
        910000000000006001,
        'student-dev',
        1,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[]'::jsonb,
        2,
        TRUE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009004,
        910000000000008003,
        910000000000006001,
        'security-lab',
        1,
        '[]'::jsonb,
        '[]'::jsonb,
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
        910000000000009005,
        910000000000008004,
        910000000000006002,
        'data-shared-geth',
        2,
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
        910000000000009006,
        910000000000008004,
        910000000000006003,
        'data-blockscout',
        2,
        '[]'::jsonb,
        '[{"container_port":4000,"service_port":4000,"protocol":"tcp"}]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[{"container_name":"data-shared-geth"}]'::jsonb,
        2,
        FALSE,
        2,
        NOW(),
        NOW()
    ),
    (
        910000000000009007,
        910000000000008004,
        910000000000006001,
        'data-student-dev',
        1,
        '[]'::jsonb,
        '[]'::jsonb,
        '[]'::jsonb,
        '500m',
        '1Gi',
        '[]'::jsonb,
        3,
        TRUE,
        3,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO template_checkpoints (
    id, template_id, title, description, check_type, script_content, script_language,
    target_container, assertion_config, score, scope, sort_order, created_at, updated_at
)
VALUES
    (
        910000000000010001,
        910000000000008001,
        '完成开发环境自检',
        '学生在个人开发环境中确认编译器与工具链可用。',
        2,
        NULL,
        NULL,
        NULL,
        NULL,
        100,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010002,
        910000000000008002,
        '完成共享链部署',
        '学生能够连接共享基础设施上的链节点并完成一次部署。',
        2,
        NULL,
        NULL,
        NULL,
        NULL,
        100,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010003,
        910000000000008003,
        '识别漏洞并完成修复说明',
        '学生需要提交漏洞分析与修复思路。',
        2,
        NULL,
        NULL,
        NULL,
        NULL,
        100,
        1,
        1,
        NOW(),
        NOW()
    ),
    (
        910000000000010004,
        910000000000008004,
        '完成链上浏览验证',
        '学生能够在浏览器中定位部署结果、事件和交易详情。',
        2,
        NULL,
        NULL,
        NULL,
        NULL,
        100,
        1,
        1,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 07. 课时与实验关联
-- ---------------------------------------------------------------------------

INSERT INTO lessons (
    id, chapter_id, course_id, title, content_type, content, experiment_id, sort_order, estimated_minutes, created_at, updated_at
)
VALUES
    (
        910000000000011001,
        910000000000007101,
        910000000000007001,
        '个人开发环境实操',
        3,
        '课时聚焦个人开发工作区的工具链验证与基本部署流程。',
        910000000000008001,
        1,
        45,
        NOW(),
        NOW()
    ),
    (
        910000000000011002,
        910000000000007101,
        910000000000007001,
        '共享链基础设施部署',
        3,
        '课时聚焦共享链节点 + 独立开发工作区的协同实验。',
        910000000000008002,
        2,
        60,
        NOW(),
        NOW()
    ),
    (
        910000000000011003,
        910000000000007102,
        910000000000007001,
        '共享链浏览器观察',
        3,
        '课时聚焦区块浏览器、共享链节点与开发环境的协作观测。',
        910000000000008004,
        3,
        50,
        NOW(),
        NOW()
    ),
    (
        910000000000011201,
        910000000000007201,
        910000000000007002,
        '漏洞修复实操',
        3,
        '课时聚焦智能合约漏洞识别与修复实验。',
        910000000000008003,
        1,
        55,
        NOW(),
        NOW()
    ),
    (
        910000000000011301,
        910000000000007301,
        910000000000007003,
        '区块浏览与事件索引',
        3,
        '课时聚焦事件检索、浏览器验证与链上数据理解。',
        910000000000008004,
        1,
        60,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO course_experiments (id, course_id, experiment_id, title, sort_order, created_at)
VALUES
    (
        910000000000012001,
        910000000000007001,
        910000000000008001,
        '课程实验：个人开发环境实操',
        1,
        NOW()
    ),
    (
        910000000000012002,
        910000000000007001,
        910000000000008002,
        '课程实验：共享链基础设施部署',
        2,
        NOW()
    ),
    (
        910000000000012003,
        910000000000007002,
        910000000000008003,
        '课程实验：智能合约漏洞分析',
        1,
        NOW()
    ),
    (
        910000000000012004,
        910000000000007003,
        910000000000008004,
        '课程实验：链上数据索引与浏览',
        1,
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO course_enrollments (id, course_id, student_id, join_method, joined_at)
VALUES
    (
        910000000000013001,
        910000000000007001,
        910000000000001101,
        1,
        NOW()
    ),
    (
        910000000000013002,
        910000000000007001,
        910000000000001102,
        1,
        NOW()
    ),
    (
        910000000000013003,
        910000000000007002,
        910000000000001101,
        1,
        NOW()
    ),
    (
        910000000000013004,
        910000000000007002,
        910000000000001102,
        1,
        NOW()
    ),
    (
        910000000000013005,
        910000000000007003,
        910000000000001203,
        1,
        NOW()
    ),
    (
        910000000000013006,
        910000000000007003,
        910000000000001204,
        1,
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 08. 学期、成绩与预警
-- ---------------------------------------------------------------------------

INSERT INTO semesters (
    id, school_id, name, code, start_date, end_date, is_current, created_at, updated_at
)
VALUES
    (
        910000000000014001,
        910000000000000001,
        '2026 春季学期',
        '2026-SPR',
        DATE '2026-02-20',
        DATE '2026-07-10',
        TRUE,
        NOW(),
        NOW()
    ),
    (
        910000000000014002,
        910000000000000002,
        '2026 春季学期',
        '2026-SPR',
        DATE '2026-02-24',
        DATE '2026-07-12',
        TRUE,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

UPDATE courses
SET semester_id = 910000000000014001,
    updated_at = NOW()
WHERE id IN (910000000000007001, 910000000000007002);

UPDATE courses
SET semester_id = 910000000000014002,
    updated_at = NOW()
WHERE id = 910000000000007003;

INSERT INTO grade_level_configs (
    id, school_id, level_name, min_score, max_score, gpa_point, sort_order, created_at, updated_at
)
VALUES
    (910000000000015001, 910000000000000001, 'A', 90, 100, 4.00, 1, NOW(), NOW()),
    (910000000000015002, 910000000000000001, 'B', 80, 89.99, 3.00, 2, NOW(), NOW()),
    (910000000000015003, 910000000000000001, 'C', 70, 79.99, 2.00, 3, NOW(), NOW()),
    (910000000000015004, 910000000000000001, 'D', 60, 69.99, 1.00, 4, NOW(), NOW()),
    (910000000000015005, 910000000000000001, 'F', 0, 59.99, 0.00, 5, NOW(), NOW()),
    (910000000000015101, 910000000000000002, 'A', 90, 100, 4.00, 1, NOW(), NOW()),
    (910000000000015102, 910000000000000002, 'B', 80, 89.99, 3.00, 2, NOW(), NOW()),
    (910000000000015103, 910000000000000002, 'C', 70, 79.99, 2.00, 3, NOW(), NOW()),
    (910000000000015104, 910000000000000002, 'D', 60, 69.99, 1.00, 4, NOW(), NOW()),
    (910000000000015105, 910000000000000002, 'F', 0, 59.99, 0.00, 5, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO warning_configs (
    id, school_id, gpa_threshold, fail_count_threshold, is_enabled, created_at, updated_at
)
VALUES
    (910000000000015201, 910000000000000001, 2.00, 2, TRUE, NOW(), NOW()),
    (910000000000015202, 910000000000000002, 2.20, 2, TRUE, NOW(), NOW())
ON CONFLICT (school_id) DO NOTHING;

INSERT INTO grade_reviews (
    id, course_id, school_id, semester_id, submitted_by, status, submit_note, submitted_at,
    reviewed_by, reviewed_at, review_comment, is_locked, locked_at, created_at, updated_at
)
VALUES
    (
        910000000000016001,
        910000000000007001,
        910000000000000001,
        910000000000014001,
        910000000000001001,
        2,
        '课程成绩已提交审核。',
        NOW() - INTERVAL '3 days',
        910000000000001002,
        NOW() - INTERVAL '2 days',
        '成绩审核通过。',
        TRUE,
        NOW() - INTERVAL '2 days',
        NOW(),
        NOW()
    ),
    (
        910000000000016002,
        910000000000007003,
        910000000000000002,
        910000000000014002,
        910000000000001202,
        2,
        '链上数据课程成绩已完成审核。',
        NOW() - INTERVAL '1 day',
        910000000000001201,
        NOW() - INTERVAL '12 hours',
        '审核通过。',
        TRUE,
        NOW() - INTERVAL '12 hours',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO student_semester_grades (
    id, student_id, school_id, semester_id, course_id, final_score, grade_level, gpa_point,
    credits, is_adjusted, review_id, created_at, updated_at
)
VALUES
    (910000000000017001, 910000000000001101, 910000000000000001, 910000000000014001, 910000000000007001, 92.0, 'A', 4.00, 2.0, FALSE, 910000000000016001, NOW(), NOW()),
    (910000000000017002, 910000000000001102, 910000000000000001, 910000000000014001, 910000000000007001, 84.0, 'B', 3.00, 2.0, FALSE, 910000000000016001, NOW(), NOW()),
    (910000000000017003, 910000000000001203, 910000000000000002, 910000000000014002, 910000000000007003, 88.0, 'B', 3.00, 2.0, FALSE, 910000000000016002, NOW(), NOW()),
    (910000000000017004, 910000000000001204, 910000000000000002, 910000000000014002, 910000000000007003, 76.0, 'C', 2.00, 2.0, FALSE, 910000000000016002, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO academic_warnings (
    id, student_id, school_id, semester_id, warning_type, detail, status, handled_by, handled_at, handle_note, created_at, updated_at
)
VALUES
    (
        910000000000018001,
        910000000000001204,
        910000000000000002,
        910000000000014002,
        1,
        '{"gpa":2.0,"threshold":2.2,"reason":"当前 GPA 接近预警阈值"}'::jsonb,
        1,
        NULL,
        NULL,
        NULL,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 09. 通知与公告
-- ---------------------------------------------------------------------------

INSERT INTO system_announcements (
    id, title, content, published_by, status, is_pinned, published_at, created_at, updated_at
)
VALUES
    (
        910000000000019001,
        '平台演示环境已就绪',
        '当前演示环境已预置课程、实验、成绩与 SSO 示例数据，可直接进行联调体验。',
        910000000000001900,
        2,
        TRUE,
        NOW() - INTERVAL '1 day',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO notifications (
    id, receiver_id, school_id, category, event_type, title, content, source_module, source_id, source_type,
    is_read, read_at, is_deleted, created_at
)
VALUES
    (
        910000000000020001,
        910000000000001101,
        910000000000000001,
        2,
        'assignment.published',
        '新实验课时已开放',
        '课程《区块链工程实践导论》新增了共享链基础设施相关学习内容，请尽快查看。',
        'course',
        910000000000007001,
        'course',
        FALSE,
        NULL,
        FALSE,
        NOW() - INTERVAL '6 hours'
    ),
    (
        910000000000020002,
        910000000000001204,
        910000000000000002,
        5,
        'grade.academic_warning',
        '学业预警提醒',
        '您的当前 GPA 接近预警阈值，请关注最近课程成绩并及时和教师沟通。',
        'grade',
        910000000000018001,
        'academic_warning',
        FALSE,
        NULL,
        FALSE,
        NOW() - INTERVAL '3 hours'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_notification_preferences (
    id, user_id, category, is_enabled, created_at, updated_at
)
VALUES
    (910000000000021001, 910000000000001101, 2, TRUE, NOW(), NOW()),
    (910000000000021002, 910000000000001102, 2, TRUE, NOW(), NOW()),
    (910000000000021003, 910000000000001203, 5, TRUE, NOW(), NOW()),
    (910000000000021004, 910000000000001204, 5, TRUE, NOW(), NOW())
ON CONFLICT (user_id, category) DO NOTHING;
