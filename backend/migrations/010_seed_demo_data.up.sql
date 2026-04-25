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

-- ---------------------------------------------------------------------------
-- 02. 角色
-- ---------------------------------------------------------------------------

INSERT INTO roles (id, code, name, description, is_system, created_at, updated_at)
VALUES
    (910000000000000101, 'teacher', '教师', '课程与实验组织者', TRUE, NOW(), NOW()),
    (910000000000000102, 'student', '学生', '课程学习与实验参与者', TRUE, NOW(), NOW()),
    (910000000000000103, 'school_admin', '学校管理员', '学校租户管理者', TRUE, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 03. 用户
-- ---------------------------------------------------------------------------

-- 统一密码说明：
-- 1. 所有 demo 账号密码统一为：LensChain2026
-- 2. 密码已使用 bcrypt 预先生成。

INSERT INTO users (
    id, phone, password_hash, name, school_id, student_no, status,
    is_first_login, is_school_admin, token_valid_after, created_at, updated_at
)
VALUES
    (
        910000000000001001,
        '13761284539',
        '$2a$12$pQ5UWnKy..G/gRIP6aD74uW4IOCgiyhH9pq71qaLY/xenvCCnCqPm',
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
        '$2a$12$pQ5UWnKy..G/gRIP6aD74uW4IOCgiyhH9pq71qaLY/xenvCCnCqPm',
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
        '$2a$12$pQ5UWnKy..G/gRIP6aD74uW4IOCgiyhH9pq71qaLY/xenvCCnCqPm',
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
        '$2a$12$pQ5UWnKy..G/gRIP6aD74uW4IOCgiyhH9pq71qaLY/xenvCCnCqPm',
        '许知遥',
        910000000000000001,
        'BC240302',
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
    (910000000000002102, 910000000000001102, 'zhiyao.xu@jhdt.edu.cn', '区块链工程学院', '区块链工程', '链工 2403', 2024, 2, 2024, '偏链节点运维方向。', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, created_at)
VALUES
    (910000000000003001, 910000000000001001, 910000000000000101, NOW()),
    (910000000000003002, 910000000000001002, 910000000000000103, NOW()),
    (910000000000003101, 910000000000001101, 910000000000000102, NOW()),
    (910000000000003102, 910000000000001102, 910000000000000102, NOW())
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 04. 镜像分类 / 镜像 / 镜像版本
-- ---------------------------------------------------------------------------

INSERT INTO image_categories (id, name, code, description, sort_order, created_at, updated_at)
VALUES
    (910000000000004001, '基础开发环境', 'base', '开发环境与工具基础镜像', 1, NOW(), NOW()),
    (910000000000004002, '链节点', 'chain-nodes', '链节点与协议运行镜像', 2, NOW(), NOW())
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
    1,
    FALSE,
    'B3C9H7Q2',
    NOW(),
    NOW() + INTERVAL '120 days',
    2.0,
    60,
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
    )
ON CONFLICT (id) DO NOTHING;
