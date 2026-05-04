-- 011_seed_demo_supplement.up.sql
-- 补充种子数据，覆盖 010 未涉及的前端页面所需的业务数据。
-- 确保每个端（学生、教师、学校管理员、超级管理员）的所有列表页面都有可显示的数据。
-- 确保数据隔离正确：学校1和学校2的数据不交叉。

-- ===========================================================================
-- 用户与 ID 索引（沿用 010 中的固定 ID）
-- ===========================================================================
-- 学校1：江海数字科技学院 910000000000000001
-- 学校2：云浦工程实验学院 910000000000000002
-- 教师1（学校1）：沈砚秋 910000000000001001
-- 校管1（学校1）：顾承礼 910000000000001002  (teacher + school_admin)
-- 学生1（学校1）：林叙安 910000000000001101
-- 学生2（学校1）：许知遥 910000000000001102
-- 校管2（学校2）：孟时越 910000000000001201  (teacher + school_admin)
-- 教师2（学校2）：姚见川 910000000000001202
-- 学生3（学校2）：宋观澜 910000000000001203
-- 学生4（学校2）：杜霁青 910000000000001204
-- 超管：平台演示管理员  910000000000001900
-- 课程1（学校1·沈砚秋）：区块链工程实践导论 910000000000007001
-- 课程2（学校1·沈砚秋）：智能合约安全基础   910000000000007002
-- 课程3（学校2·姚见川）：链上数据分析实践   910000000000007003
-- 章节：910000000000007101/7102/7201/7301
-- 课时：910000000000011001/11002/11003/11201/11301
-- 学期1（学校1）：910000000000014001
-- 学期2（学校2）：910000000000014002

-- ===========================================================================
-- S1. 课程教学补充数据
-- ===========================================================================

-- S1.1 作业（每门课各1个）
INSERT INTO assignments (
    id, course_id, chapter_id, title, description, assignment_type, total_score,
    deadline_at, max_submissions, late_policy, late_deduction_per_day, is_published, sort_order, created_at, updated_at
)
VALUES
    (
        920000000000001001,
        910000000000007001,
        910000000000007101,
        '链上环境配置报告',
        '完成链节点安装与连通性验证，提交配置截图与过程说明。',
        1,
        100.00,
        NOW() + INTERVAL '14 days',
        2,
        2,
        5.00,
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        920000000000001002,
        910000000000007002,
        910000000000007201,
        '漏洞识别练习',
        '阅读给定的智能合约代码，标注存在的安全隐患，并写出修复方案。',
        1,
        100.00,
        NOW() + INTERVAL '10 days',
        1,
        1,
        NULL,
        TRUE,
        1,
        NOW(),
        NOW()
    ),
    (
        920000000000001003,
        910000000000007003,
        910000000000007301,
        '事件索引实操作业',
        '使用区块浏览器查询指定合约的 Transfer 事件，并提交截图与分析。',
        1,
        100.00,
        NOW() + INTERVAL '12 days',
        2,
        2,
        3.00,
        TRUE,
        1,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- S1.2 作业题目
INSERT INTO assignment_questions (
    id, assignment_id, question_type, title, options, correct_answer, reference_answer, score, judge_config, sort_order, created_at, updated_at
)
VALUES
    (920000000000002001, 920000000000001001, 3, '请描述你在搭建本地链节点过程中遇到的关键步骤和问题。', NULL, NULL, '应包含节点启动命令、端口验证、区块高度确认等关键环节。', 60.00, NULL, 1, NOW(), NOW()),
    (920000000000002002, 920000000000001001, 1, '以太坊本地开发链默认的 HTTP-RPC 端口是？', '["8545","3000","8080","30303"]'::jsonb, '8545', NULL, 40.00, NULL, 2, NOW(), NOW()),
    (920000000000002003, 920000000000001002, 3, '指出下列合约中的安全漏洞类型，并给出修复方案。', NULL, NULL, '应识别出重入漏洞，并采用 Checks-Effects-Interactions 模式修复。', 100.00, NULL, 1, NOW(), NOW()),
    (920000000000002004, 920000000000001003, 3, '截图展示使用 Blockscout 查询到的 Transfer 事件列表，并分析事件参数含义。', NULL, NULL, '应展示 from、to、value 字段及其在 ERC-20 标准中的含义。', 100.00, NULL, 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S1.3 学生提交（学校1学生提交了课程1的作业）
INSERT INTO assignment_submissions (
    id, assignment_id, student_id, submission_no, status, total_score,
    is_late, graded_by, graded_at, teacher_comment, submitted_at
)
VALUES
    (920000000000003001, 920000000000001001, 910000000000001101, 1, 3, 88.00, FALSE, 910000000000001001, NOW() - INTERVAL '1 day', '流程描述清晰，端口验证完整。', NOW() - INTERVAL '3 days'),
    (920000000000003002, 920000000000001001, 910000000000001102, 1, 3, 75.00, FALSE, 910000000000001001, NOW() - INTERVAL '1 day', '节点启动步骤遗漏了区块高度确认。', NOW() - INTERVAL '2 days')
ON CONFLICT (id) DO NOTHING;

-- S1.4 答题记录
INSERT INTO submission_answers (
    id, submission_id, question_id, answer_content, is_correct, score, teacher_comment, created_at, updated_at
)
VALUES
    (920000000000004001, 920000000000003001, 920000000000002001, '首先使用 geth --dev 启动本地链，验证 8545 端口可达后确认区块高度递增。过程中遇到防火墙阻止问题，通过允许端口通行解决。', NULL, 50.00, '描述完整。', NOW(), NOW()),
    (920000000000004002, 920000000000003001, 920000000000002002, '8545', TRUE, 38.00, NULL, NOW(), NOW()),
    (920000000000004003, 920000000000003002, 920000000000002001, '执行 geth --dev 命令启动本地开发链，确认 RPC 端口可用。', NULL, 40.00, '缺少区块高度确认步骤。', NOW(), NOW()),
    (920000000000004004, 920000000000003002, 920000000000002002, '8545', TRUE, 35.00, NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S1.5 学习进度
INSERT INTO learning_progresses (
    id, course_id, student_id, lesson_id, status, video_progress, study_duration,
    completed_at, last_accessed_at, created_at, updated_at
)
VALUES
    (920000000000005001, 910000000000007001, 910000000000001101, 910000000000011001, 3, NULL, 2400, NOW() - INTERVAL '5 days', NOW() - INTERVAL '1 day', NOW(), NOW()),
    (920000000000005002, 910000000000007001, 910000000000001101, 910000000000011002, 2, NULL, 1200, NULL, NOW() - INTERVAL '2 hours', NOW(), NOW()),
    (920000000000005003, 910000000000007001, 910000000000001102, 910000000000011001, 3, NULL, 1800, NOW() - INTERVAL '4 days', NOW() - INTERVAL '3 days', NOW(), NOW()),
    (920000000000005004, 910000000000007001, 910000000000001102, 910000000000011002, 1, NULL, 300, NULL, NOW() - INTERVAL '6 hours', NOW(), NOW()),
    (920000000000005005, 910000000000007003, 910000000000001203, 910000000000011301, 3, NULL, 3000, NOW() - INTERVAL '2 days', NOW() - INTERVAL '12 hours', NOW(), NOW()),
    (920000000000005006, 910000000000007003, 910000000000001204, 910000000000011301, 2, NULL, 900, NULL, NOW() - INTERVAL '1 day', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S1.6 课程表
INSERT INTO course_schedules (id, course_id, day_of_week, start_time, end_time, location, created_at)
VALUES
    (920000000000006001, 910000000000007001, 1, '08:30', '10:10', '教A-301', NOW()),
    (920000000000006002, 910000000000007001, 3, '14:00', '15:40', '实验楼-B205', NOW()),
    (920000000000006003, 910000000000007002, 2, '10:20', '12:00', '教A-405', NOW()),
    (920000000000006004, 910000000000007003, 4, '08:30', '10:10', '工程楼-C102', NOW()),
    (920000000000006005, 910000000000007003, 5, '14:00', '15:40', '工程楼-实验室E', NOW())
ON CONFLICT (id) DO NOTHING;

-- S1.7 课程公告
INSERT INTO course_announcements (id, course_id, teacher_id, title, content, is_pinned, created_at, updated_at)
VALUES
    (920000000000007001, 910000000000007001, 910000000000001001, '第二章实验开放通知', '共享链基础设施实验已开放，请同学们在本周内完成实验启动和基础任务。实验时长90分钟，注意保存快照。', TRUE, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
    (920000000000007002, 910000000000007001, 910000000000001001, '作业截止提醒', '链上环境配置报告作业将于两周后截止，请务必按时提交。晚交每天扣5分。', FALSE, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    (920000000000007003, 910000000000007002, 910000000000001001, '漏洞分析实验注意事项', '请同学们在进入实验前先阅读课时说明，了解目标合约的基本逻辑。', FALSE, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
    (920000000000007004, 910000000000007003, 910000000000001202, '链上数据课程开课通知', '本课程围绕事件索引与区块浏览展开，请先安装好 MetaMask 浏览器插件。', TRUE, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- S1.8 讨论帖
INSERT INTO course_discussions (
    id, course_id, author_id, title, content, is_pinned, reply_count, like_count, last_replied_at, created_at, updated_at
)
VALUES
    (920000000000008001, 910000000000007001, 910000000000001101, 'geth dev 模式启动后无法连接 RPC', '按照课时步骤执行 geth --dev 后，curl localhost:8545 返回 connection refused。操作系统是 Ubuntu 22.04，请问还需要额外配置吗？', FALSE, 2, 1, NOW() - INTERVAL '12 hours', NOW() - INTERVAL '2 days', NOW()),
    (920000000000008002, 910000000000007001, 910000000000001102, '共享链实验中如何确认合约部署成功？', '实验步骤要求"通过共享 RPC 部署合约"，部署后怎么验证合约确实上链了？', FALSE, 1, 0, NOW() - INTERVAL '6 hours', NOW() - INTERVAL '1 day', NOW()),
    (920000000000008003, 910000000000007003, 910000000000001203, 'Blockscout 页面显示 "no transactions"', '明明已经发送了交易，但浏览器一直显示空。是索引延迟还是配置问题？', FALSE, 1, 1, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '18 hours', NOW())
ON CONFLICT (id) DO NOTHING;

-- S1.9 讨论回复
INSERT INTO discussion_replies (id, discussion_id, author_id, content, reply_to_id, created_at)
VALUES
    (920000000000009001, 920000000000008001, 910000000000001001, '请检查是否添加了 --http --http.addr 0.0.0.0 参数。默认情况下 geth 只监听 127.0.0.1，容器环境需绑定全部地址。', NULL, NOW() - INTERVAL '1 day'),
    (920000000000009002, 920000000000008001, 910000000000001101, '加上参数后解决了，谢谢老师！', 920000000000009001, NOW() - INTERVAL '12 hours'),
    (920000000000009003, 920000000000008002, 910000000000001001, '可以使用 eth_getCode 调用查询合约地址，如果返回非 0x 就说明部署成功。也可以通过 Blockscout 浏览器查看。', NULL, NOW() - INTERVAL '6 hours'),
    (920000000000009004, 920000000000008003, 910000000000001202, 'Blockscout 有几秒钟的索引延迟，请等待30秒后刷新。如果仍然为空，检查浏览器连接的 RPC 地址是否与交易发送的链节点一致。', NULL, NOW() - INTERVAL '3 hours')
ON CONFLICT (id) DO NOTHING;

-- S1.10 讨论点赞
INSERT INTO discussion_likes (id, discussion_id, user_id, created_at)
VALUES
    (920000000000009101, 920000000000008001, 910000000000001102, NOW() - INTERVAL '1 day'),
    (920000000000009102, 920000000000008003, 910000000000001204, NOW() - INTERVAL '6 hours')
ON CONFLICT (discussion_id, user_id) DO NOTHING;

-- S1.11 课程评价
INSERT INTO course_evaluations (id, course_id, student_id, rating, comment, created_at, updated_at)
VALUES
    (920000000000010001, 910000000000007001, 910000000000001101, 5, '实验环境很流畅，共享链部署环节非常有启发性。', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    (920000000000010002, 910000000000007001, 910000000000001102, 4, '课程内容不错，但希望课时说明能更详细一些。', NOW() - INTERVAL '12 hours', NOW() - INTERVAL '12 hours'),
    (920000000000010003, 910000000000007003, 910000000000001203, 5, '区块浏览器实验很直观，对理解链上数据有很大帮助。', NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours')
ON CONFLICT (course_id, student_id) DO NOTHING;

-- S1.12 成绩权重配置
INSERT INTO course_grade_configs (id, course_id, config, created_at, updated_at)
VALUES
    (920000000000011001, 910000000000007001, '{"weights":[{"source":"assignment","weight":30},{"source":"experiment","weight":50},{"source":"participation","weight":20}]}'::jsonb, NOW(), NOW()),
    (920000000000011002, 910000000000007002, '{"weights":[{"source":"assignment","weight":40},{"source":"experiment","weight":40},{"source":"participation","weight":20}]}'::jsonb, NOW(), NOW()),
    (920000000000011003, 910000000000007003, '{"weights":[{"source":"assignment","weight":35},{"source":"experiment","weight":45},{"source":"participation","weight":20}]}'::jsonb, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ===========================================================================
-- S2. CTF 竞赛模块
-- ===========================================================================

-- S2.1 竞赛
INSERT INTO competitions (
    id, title, description, competition_type, scope, school_id, created_by,
    team_mode, max_team_size, min_team_size, max_teams, status,
    registration_start_at, registration_end_at, start_at, end_at, freeze_at,
    jeopardy_config, rules, created_at, updated_at
)
VALUES
    (
        920000000000020001,
        '江海杯·智能合约安全挑战赛',
        '面向江海学院学生的校内解题赛，涵盖重入、整数溢出、权限控制等常见智能合约漏洞类型。',
        1,
        2,
        910000000000000001,
        910000000000001001,
        2,
        3,
        1,
        30,
        3,
        NOW() - INTERVAL '14 days',
        NOW() - INTERVAL '3 days',
        NOW() - INTERVAL '2 days',
        NOW() + INTERVAL '5 days',
        NOW() + INTERVAL '4 days',
        '{"scoring_mode":"dynamic","initial_score":1000,"min_score":100,"decay_factor":20}'::jsonb,
        '1. 每队 1-3 人\n2. 不得使用自动化攻击工具\n3. Flag 提交频率限制 60 秒\n4. 最终排名以总分为准',
        NOW(),
        NOW()
    ),
    (
        920000000000020002,
        '全平台区块链 CTF 公开赛',
        '面向所有学校的跨校公开赛，涵盖合约安全、链上数据分析与 DeFi 协议审计。',
        1,
        1,
        NULL,
        910000000000001900,
        2,
        4,
        1,
        50,
        2,
        NOW() - INTERVAL '5 days',
        NOW() + INTERVAL '2 days',
        NOW() + INTERVAL '3 days',
        NOW() + INTERVAL '10 days',
        NOW() + INTERVAL '9 days',
        '{"scoring_mode":"dynamic","initial_score":1000,"min_score":100,"decay_factor":15}'::jsonb,
        '1. 每队 1-4 人\n2. 跨校组队允许\n3. 禁止共享 Flag\n4. 最终排名以总分为准',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- S2.2 题目
INSERT INTO challenges (
    id, title, description, category, difficulty, base_score, flag_type,
    static_flag, runtime_mode, author_id, school_id, status, is_public, created_at, updated_at
)
VALUES
    (920000000000021001, '新手重入', '一个存在经典提款重入漏洞的合约，请完成攻击使 solved 变为 true。', 'pwn', 1, 500, 1, 'flag{reentrancy_solved_2026}', 2, 910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW()),
    (920000000000021002, '整数溢出陷阱', '利用 unchecked 块中的整数溢出来绕过余额检查。', 'pwn', 2, 800, 1, 'flag{overflow_bypassed_2026}', 2, 910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW()),
    (920000000000021003, '权限提升挑战', '找到合约中的权限校验缺陷，将自己设置为 owner。', 'pwn', 2, 700, 1, 'flag{owner_escalated_2026}', 2, 910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW()),
    (920000000000021004, '链上密文解密', '合约中存储了一段加密 Flag，通过分析 storage slot 恢复明文。', 'reverse', 3, 1000, 1, 'flag{storage_exposed_2026}', 1, 910000000000001202, 910000000000000002, 2, TRUE, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S2.3 竞赛题目关联
INSERT INTO competition_challenges (id, competition_id, challenge_id, sort_order, created_at)
VALUES
    (920000000000022001, 920000000000020001, 920000000000021001, 1, NOW()),
    (920000000000022002, 920000000000020001, 920000000000021002, 2, NOW()),
    (920000000000022003, 920000000000020001, 920000000000021003, 3, NOW()),
    (920000000000022004, 920000000000020002, 920000000000021001, 1, NOW()),
    (920000000000022005, 920000000000020002, 920000000000021004, 2, NOW())
ON CONFLICT (id) DO NOTHING;

-- S2.4 团队
INSERT INTO teams (
    id, competition_id, name, captain_id, invite_code, status, created_at, updated_at
)
VALUES
    (920000000000023001, 920000000000020001, '链上先锋队', 910000000000001101, 'TEAM01A', 1, NOW(), NOW()),
    (920000000000023002, 920000000000020001, '合约猎人组', 910000000000001102, 'TEAM02B', 1, NOW(), NOW()),
    (920000000000023003, 920000000000020002, '跨校联合战队', 910000000000001101, 'TEAM03C', 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S2.5 团队成员
-- 注意：同一竞赛中一名学生只能属于一支队伍
INSERT INTO team_members (id, team_id, student_id, role, joined_at)
VALUES
    (920000000000024001, 920000000000023001, 910000000000001101, 1, NOW()),   -- 学生1 队长·链上先锋队（竞赛1）
    (920000000000024003, 920000000000023002, 910000000000001102, 1, NOW()),   -- 学生2 队长·合约猎人组（竞赛1）
    (920000000000024004, 920000000000023003, 910000000000001101, 1, NOW()),   -- 学生1 队长·跨校联合战队（竞赛2，不同竞赛可重复参赛）
    (920000000000024005, 920000000000023003, 910000000000001203, 2, NOW())    -- 学生3 成员·跨校联合战队（竞赛2）
ON CONFLICT (id) DO NOTHING;

-- S2.6 竞赛报名
INSERT INTO competition_registrations (id, competition_id, team_id, registered_by, status, created_at)
VALUES
    (920000000000025001, 920000000000020001, 920000000000023001, 910000000000001101, 2, NOW() - INTERVAL '10 days'),
    (920000000000025002, 920000000000020001, 920000000000023002, 910000000000001102, 2, NOW() - INTERVAL '9 days'),
    (920000000000025003, 920000000000020002, 920000000000023003, 910000000000001101, 1, NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- ===========================================================================
-- S3. 日志数据（登录日志、操作日志）
-- ===========================================================================

-- S3.1 登录日志
INSERT INTO login_logs (
    id, user_id, action, login_method, ip, user_agent, fail_reason, created_at
)
VALUES
    (920000000000030001, 910000000000001101, 1, 1, '10.0.1.101', 'Mozilla/5.0 Chrome/125 Windows', NULL, NOW() - INTERVAL '2 days'),
    (920000000000030002, 910000000000001102, 1, 1, '10.0.1.102', 'Mozilla/5.0 Chrome/125 macOS', NULL, NOW() - INTERVAL '1 day'),
    (920000000000030003, 910000000000001001, 1, 1, '10.0.1.10', 'Mozilla/5.0 Firefox/126 Linux', NULL, NOW() - INTERVAL '12 hours'),
    (920000000000030004, 910000000000001002, 1, 1, '10.0.1.11', 'Mozilla/5.0 Chrome/125 Windows', NULL, NOW() - INTERVAL '8 hours'),
    (920000000000030005, 910000000000001203, 1, 1, '10.0.2.201', 'Mozilla/5.0 Safari/17 macOS', NULL, NOW() - INTERVAL '6 hours'),
    (920000000000030006, 910000000000001204, 1, 1, '10.0.2.202', 'Mozilla/5.0 Chrome/125 Windows', NULL, NOW() - INTERVAL '4 hours'),
    (920000000000030007, 910000000000001202, 1, 1, '10.0.2.10', 'Mozilla/5.0 Chrome/125 Linux', NULL, NOW() - INTERVAL '3 hours'),
    (920000000000030008, 910000000000001201, 1, 1, '10.0.2.11', 'Mozilla/5.0 Firefox/126 Windows', NULL, NOW() - INTERVAL '2 hours'),
    (920000000000030009, 910000000000001900, 1, 1, '192.168.1.100', 'Mozilla/5.0 Chrome/125 macOS', NULL, NOW() - INTERVAL '1 hour'),
    (920000000000030010, 910000000000001101, 1, 1, '10.0.1.101', 'Mozilla/5.0 Chrome/125 Windows', NULL, NOW() - INTERVAL '30 minutes'),
    -- 一次失败的登录（用学生1的user_id记录）
    (920000000000030011, 910000000000001101, 2, 1, '10.0.99.99', 'Mozilla/5.0 Chrome/125 Windows', '密码错误', NOW() - INTERVAL '5 hours')
ON CONFLICT (id) DO NOTHING;

-- S3.2 操作日志
INSERT INTO operation_logs (
    id, operator_id, action, target_type, target_id, detail, ip, created_at
)
VALUES
    (920000000000031001, 910000000000001002, 'user.create', 'user', 910000000000001101, '{"name":"林叙安","role":"student"}'::jsonb, '10.0.1.11', NOW() - INTERVAL '7 days'),
    (920000000000031002, 910000000000001001, 'course.publish', 'course', 910000000000007001, '{"title":"区块链工程实践导论"}'::jsonb, '10.0.1.10', NOW() - INTERVAL '6 days'),
    (920000000000031003, 910000000000001001, 'experiment.template.publish', 'experiment_template', 910000000000008001, '{"title":"以太坊本地开发与部署实践"}'::jsonb, '10.0.1.10', NOW() - INTERVAL '5 days'),
    (920000000000031004, 910000000000001002, 'grade.review.approve', 'grade_review', 910000000000016001, '{"course":"区块链工程实践导论","action":"审核通过"}'::jsonb, '10.0.1.11', NOW() - INTERVAL '2 days'),
    (920000000000031005, 910000000000001201, 'user.import', 'user', NULL, '{"count":2,"method":"excel"}'::jsonb, '10.0.2.11', NOW() - INTERVAL '8 days'),
    (920000000000031006, 910000000000001202, 'course.publish', 'course', 910000000000007003, '{"title":"链上数据分析实践"}'::jsonb, '10.0.2.10', NOW() - INTERVAL '6 days'),
    (920000000000031007, 910000000000001900, 'school.approve', 'school', 910000000000000001, '{"school":"江海数字科技学院","action":"入驻审核通过"}'::jsonb, '192.168.1.100', NOW() - INTERVAL '14 days'),
    (920000000000031008, 910000000000001900, 'system.config.update', 'system_config', NULL, '{"key":"session_timeout_hours","old":"24","new":"12"}'::jsonb, '192.168.1.100', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO NOTHING;

-- ===========================================================================
-- S4. 系统管理与监控
-- ===========================================================================

-- S4.1 告警规则
INSERT INTO alert_rules (
    id, name, description, alert_type, level, condition, silence_period, is_enabled, created_by, created_at, updated_at
)
VALUES
    (920000000000040001, 'CPU使用率过高', '当任意节点 CPU 使用率超过 85% 时触发告警。', 1, 2, '{"metric":"cpu_usage","operator":">","threshold":85,"unit":"percent"}'::jsonb, 1800, TRUE, 910000000000001900, NOW(), NOW()),
    (920000000000040002, '内存使用率过高', '当任意节点内存使用率超过 90% 时触发紧急告警。', 1, 3, '{"metric":"memory_usage","operator":">","threshold":90,"unit":"percent"}'::jsonb, 900, TRUE, 910000000000001900, NOW(), NOW()),
    (920000000000040003, '实验实例创建失败', '当实验实例创建连续失败超过 5 次时触发告警。', 2, 2, '{"metric":"instance_create_fail_count","operator":">=","threshold":5,"window_seconds":300}'::jsonb, 3600, TRUE, 910000000000001900, NOW(), NOW()),
    (920000000000040004, '数据库连接池耗尽', '当数据库可用连接数降至 0 时触发紧急告警。', 3, 3, '{"metric":"db_available_connections","operator":"<=","threshold":0}'::jsonb, 600, TRUE, 910000000000001900, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S4.2 告警事件
INSERT INTO alert_events (
    id, rule_id, level, title, detail, status, handled_by, handled_at, handle_note, triggered_at, created_at
)
VALUES
    (920000000000041001, 920000000000040001, 2, 'worker-03 CPU 使用率 87%', '{"node":"worker-03","cpu_usage":87.2,"timestamp":"2026-05-03T14:22:00Z"}'::jsonb, 2, 910000000000001900, NOW() - INTERVAL '6 hours', '已排查，为批量实验启动导致，已自动回落。', NOW() - INTERVAL '8 hours', NOW() - INTERVAL '8 hours'),
    (920000000000041002, 920000000000040002, 3, 'worker-01 内存使用率 93%', '{"node":"worker-01","memory_usage":93.1,"timestamp":"2026-05-03T16:05:00Z"}'::jsonb, 1, NULL, NULL, NULL, NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'),
    (920000000000041003, 920000000000040003, 2, '实验创建连续失败 6 次', '{"recent_failures":6,"window":"5分钟","last_error":"资源配额不足"}'::jsonb, 2, 910000000000001900, NOW() - INTERVAL '1 hour', '配额已调整，恢复正常。', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours')
ON CONFLICT (id) DO NOTHING;

-- S4.3 平台统计（最近7天）
INSERT INTO platform_statistics (
    id, stat_date, active_users, new_users, total_users, total_schools,
    total_courses, active_courses, total_experiments, total_competitions,
    active_competitions, storage_used_gb, api_request_count, created_at
)
VALUES
    (920000000000042001, CURRENT_DATE - INTERVAL '6 days', 6, 0, 9, 2, 3, 3, 4, 2, 1, 12.50, 4200, NOW()),
    (920000000000042002, CURRENT_DATE - INTERVAL '5 days', 7, 0, 9, 2, 3, 3, 4, 2, 1, 12.80, 5100, NOW()),
    (920000000000042003, CURRENT_DATE - INTERVAL '4 days', 5, 0, 9, 2, 3, 3, 4, 2, 1, 13.10, 3800, NOW()),
    (920000000000042004, CURRENT_DATE - INTERVAL '3 days', 8, 0, 9, 2, 3, 3, 4, 2, 2, 13.50, 6200, NOW()),
    (920000000000042005, CURRENT_DATE - INTERVAL '2 days', 7, 0, 9, 2, 3, 3, 4, 2, 2, 13.80, 5500, NOW()),
    (920000000000042006, CURRENT_DATE - INTERVAL '1 day', 9, 0, 9, 2, 3, 3, 4, 2, 2, 14.20, 7800, NOW()),
    (920000000000042007, CURRENT_DATE, 4, 0, 9, 2, 3, 3, 4, 2, 2, 14.30, 2100, NOW())
ON CONFLICT (stat_date) DO NOTHING;

-- S4.4 备份记录
INSERT INTO backup_records (
    id, backup_type, status, file_path, file_size, database_name,
    started_at, completed_at, error_message, triggered_by, created_at
)
VALUES
    (920000000000043001, 1, 2, '/backups/2026-05-01_020000_lenschain.sql.gz', 524288000, 'lenschain', NOW() - INTERVAL '3 days 2 hours', NOW() - INTERVAL '3 days 1 hour 45 minutes', NULL, NULL, NOW() - INTERVAL '3 days'),
    (920000000000043002, 1, 2, '/backups/2026-05-02_020000_lenschain.sql.gz', 531456000, 'lenschain', NOW() - INTERVAL '2 days 2 hours', NOW() - INTERVAL '2 days 1 hour 48 minutes', NULL, NULL, NOW() - INTERVAL '2 days'),
    (920000000000043003, 1, 2, '/backups/2026-05-03_020000_lenschain.sql.gz', 538624000, 'lenschain', NOW() - INTERVAL '1 day 2 hours', NOW() - INTERVAL '1 day 1 hour 50 minutes', NULL, NULL, NOW() - INTERVAL '1 day'),
    (920000000000043004, 2, 2, '/backups/manual_2026-05-03_superadmin.sql.gz', 540000000, 'lenschain', NOW() - INTERVAL '12 hours', NOW() - INTERVAL '11 hours 45 minutes', NULL, 910000000000001900, NOW() - INTERVAL '12 hours')
ON CONFLICT (id) DO NOTHING;

-- S4.5 配置变更记录
INSERT INTO config_change_logs (
    id, config_group, config_key, old_value, new_value, changed_by, changed_at, ip
)
VALUES
    (920000000000044001, 'security', 'session_timeout_hours', '24', '12', 910000000000001900, NOW() - INTERVAL '10 days', '192.168.1.100'),
    (920000000000044002, 'security', 'max_login_fail_count', '5', '3', 910000000000001900, NOW() - INTERVAL '8 days', '192.168.1.100'),
    (920000000000044003, 'backup', 'backup_retention_count', '30', '45', 910000000000001900, NOW() - INTERVAL '5 days', '192.168.1.100')
ON CONFLICT (id) DO NOTHING;

-- ===========================================================================
-- S5. 入驻申请、资源配额、通知模板
-- ===========================================================================

-- S5.1 入驻申请（含已通过和待审核各1条）
INSERT INTO school_applications (
    id, school_name, school_code, school_address, school_website,
    contact_name, contact_phone, contact_email, contact_title,
    status, reviewer_id, reviewed_at, reject_reason, school_id, created_at
)
VALUES
    (
        920000000000050001,
        '江海数字科技学院',
        'jianghai-digital-tech',
        '江苏省苏州市工业园区星海路 18 号',
        'https://www.jhdt.edu.cn',
        '周闻笙',
        '13761284539',
        'operations@jhdt.edu.cn',
        '实验教学平台主管',
        2,
        910000000000001900,
        NOW() - INTERVAL '30 days',
        NULL,
        910000000000000001,
        NOW() - INTERVAL '35 days'
    ),
    (
        920000000000050002,
        '明远理工大学',
        'mingyuan-tech',
        '广东省深圳市南山区科技南路 88 号',
        'https://www.mytech.edu.cn',
        '陈景行',
        '13512345678',
        'jingxing.chen@mytech.edu.cn',
        '教务处信息化主管',
        1,
        NULL,
        NULL,
        NULL,
        NULL,
        NOW() - INTERVAL '2 days'
    ),
    (
        920000000000050003,
        '北辰信息学院',
        'beichen-info',
        '北京市海淀区学院路 12 号',
        'https://www.bcinfo.edu.cn',
        '赵清河',
        '13698765432',
        'qinghe.zhao@bcinfo.edu.cn',
        '实验中心主任',
        3,
        910000000000001900,
        NOW() - INTERVAL '5 days',
        '提交材料中学校办学资质证明缺失，请补充后重新申请。',
        NULL,
        NOW() - INTERVAL '10 days'
    )
ON CONFLICT (id) DO NOTHING;

-- S5.2 资源配额（学校级 + 课程级）
INSERT INTO resource_quotas (
    id, quota_level, school_id, course_id,
    max_cpu, max_memory, max_storage, max_concurrency, max_per_student,
    used_cpu, used_memory, used_storage, current_concurrency,
    created_at, updated_at
)
VALUES
    (920000000000051001, 1, 910000000000000001, NULL, '40', '80Gi', '500Gi', 30, 2, '8', '16Gi', '120Gi', 5, NOW(), NOW()),
    (920000000000051002, 2, 910000000000000001, 910000000000007001, '20', '40Gi', '200Gi', 15, 2, '4', '8Gi', '60Gi', 3, NOW(), NOW()),
    (920000000000051003, 1, 910000000000000002, NULL, '30', '60Gi', '400Gi', 20, 2, '3', '6Gi', '40Gi', 2, NOW(), NOW()),
    (920000000000051004, 2, 910000000000000002, 910000000000007003, '15', '30Gi', '150Gi', 10, 2, '2', '4Gi', '30Gi', 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- S5.3 通知模板
INSERT INTO notification_templates (
    id, event_type, category, title_template, content_template, variables, is_enabled, created_at, updated_at
)
VALUES
    (920000000000052001, 'assignment.published', 2, '新作业发布：{{assignment_title}}', '课程《{{course_title}}》发布了新作业"{{assignment_title}}"，截止时间为 {{deadline}}。请及时完成并提交。', '["assignment_title","course_title","deadline"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052002, 'experiment.ready', 3, '实验环境已就绪', '您在课程《{{course_title}}》中启动的实验"{{template_title}}"已准备就绪，请前往实验操作页面开始操作。', '["course_title","template_title"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052003, 'grade.published', 5, '成绩已发布：{{course_title}}', '课程《{{course_title}}》的期末成绩已发布，最终得分 {{score}} 分，等级 {{level}}。', '["course_title","score","level"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052004, 'grade.academic_warning', 5, '学业预警提醒', '您在 {{semester}} 学期的 GPA 为 {{gpa}}，低于预警阈值 {{threshold}}。请关注学业情况。', '["semester","gpa","threshold"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052005, 'ctf.registration.approved', 4, '竞赛报名已通过', '您的队伍"{{team_name}}"已成功报名参加"{{competition_title}}"竞赛。比赛将于 {{start_time}} 开始。', '["team_name","competition_title","start_time"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052006, 'school.license.expiring', 1, '学校授权即将到期', '学校"{{school_name}}"的平台授权将于 {{expire_date}} 到期，请及时续约。', '["school_name","expire_date"]'::jsonb, TRUE, NOW(), NOW()),
    (920000000000052007, 'system.alert', 1, '系统告警：{{alert_title}}', '{{alert_detail}}（告警级别：{{alert_level}}）', '["alert_title","alert_detail","alert_level"]'::jsonb, TRUE, NOW(), NOW())
ON CONFLICT (event_type) DO NOTHING;

-- ===========================================================================
-- S6. 补充：成绩申诉（学生端成绩申诉页需要）
-- ===========================================================================

INSERT INTO grade_appeals (
    id, student_id, school_id, semester_id, course_id, grade_id, original_score, appeal_reason, status,
    handled_by, handled_at, new_score, handle_comment, created_at, updated_at
)
VALUES
    (920000000000060001, 910000000000001102, 910000000000000001, 910000000000014001, 910000000000007001,
     910000000000017002, 84.0,
     '实验报告中的操作历史截图因环境重置丢失，导致扣分过多，申请复核。',
     1, NULL, NULL, NULL, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    (920000000000060002, 910000000000001204, 910000000000000002, 910000000000014002, 910000000000007003,
     910000000000017004, 76.0,
     '事件索引查询部分因浏览器延迟未能截图，但实际已完成操作，请老师复核。',
     2, 910000000000001202, NOW() - INTERVAL '6 hours', 82.0, '经核实操作日志，确认操作完成，成绩已调整至82分。',
     NOW() - INTERVAL '2 days', NOW() - INTERVAL '6 hours')
ON CONFLICT (id) DO NOTHING;

-- ===========================================================================
-- S7. 补充通知数据（确保每种角色收件箱都有数据可展示）
-- ===========================================================================
-- 010 中仅有 2 条通知：学生1（课程通知）和学生4（成绩预警），
-- 教师、校管、超管收件箱为空，无法测试。以下按角色补充。

INSERT INTO notifications (
    id, receiver_id, school_id, category, event_type, title, content,
    source_module, source_id, source_type, is_read, read_at, is_deleted, created_at
)
VALUES
    -- 教师1（沈砚秋）：学生提交作业通知
    (920000000000070001, 910000000000001001, 910000000000000001, 2, 'assignment.submitted',
     '学生提交了作业', '学生 林叙安 提交了课程《区块链工程实践导论》的作业"链上环境配置报告"，请及时批改。',
     'course', 920000000000003001, 'assignment_submission',
     FALSE, NULL, FALSE, NOW() - INTERVAL '3 days'),

    -- 教师1（沈砚秋）：成绩申诉通知
    (920000000000070002, 910000000000001001, 910000000000000001, 5, 'grade.appeal.created',
     '收到成绩申诉', '学生 许知遥 对课程《区块链工程实践导论》的成绩提出了申诉，请查看并处理。',
     'grade', 920000000000060001, 'grade_appeal',
     FALSE, NULL, FALSE, NOW() - INTERVAL '1 day'),

    -- 教师1（沈砚秋）：实验环境告警（已读）
    (920000000000070003, 910000000000001001, 910000000000000001, 3, 'experiment.instance.timeout',
     '实验即将超时', '学生 林叙安 在"以太坊本地开发与部署实践"实验中已运行超过 85 分钟，即将达到时长上限。',
     'experiment', NULL, 'experiment_instance',
     TRUE, NOW() - INTERVAL '4 hours', FALSE, NOW() - INTERVAL '5 hours'),

    -- 校管1（顾承礼）：系统通知
    (920000000000070004, 910000000000001002, 910000000000000001, 1, 'system.maintenance',
     '平台维护通知', '平台将于本周日凌晨 2:00-4:00 进行系统维护，届时部分功能可能暂时不可用。',
     'system', NULL, 'system',
     FALSE, NULL, FALSE, NOW() - INTERVAL '2 days'),

    -- 校管1（顾承礼）：成绩审核完成（已读）
    (920000000000070005, 910000000000001002, 910000000000000001, 5, 'grade.review.completed',
     '成绩审核已完成', '课程《区块链工程实践导论》的期末成绩审核已完成，共审核 2 名学生。',
     'grade', 910000000000016001, 'grade_review',
     TRUE, NOW() - INTERVAL '1 day', FALSE, NOW() - INTERVAL '2 days'),

    -- 校管1（顾承礼）：新用户导入通知（已读）
    (920000000000070006, 910000000000001002, 910000000000000001, 1, 'user.import.completed',
     '用户批量导入完成', '本次导入共处理 2 条记录，成功 2 条，失败 0 条。',
     'auth', NULL, 'user_import',
     TRUE, NOW() - INTERVAL '5 days', FALSE, NOW() - INTERVAL '6 days'),

    -- 教师2（姚见川·学校2）：学生作业提交
    (920000000000070007, 910000000000001202, 910000000000000002, 2, 'assignment.submitted',
     '学生提交了作业', '学生 宋观澜 提交了课程《链上数据分析实践》的作业"事件索引实操作业"，请及时批改。',
     'course', 920000000000001003, 'assignment_submission',
     FALSE, NULL, FALSE, NOW() - INTERVAL '2 days'),

    -- 校管2（孟时越·学校2）：资源配额告警
    (920000000000070008, 910000000000001201, 910000000000000002, 1, 'system.quota.warning',
     '资源配额即将耗尽', '学校"云浦工程实验学院"的实验并发配额使用率已达 90%，请考虑申请扩容。',
     'system', NULL, 'resource_quota',
     FALSE, NULL, FALSE, NOW() - INTERVAL '1 day'),

    -- 超管：学校入驻申请
    (920000000000070009, 910000000000001900, NULL, 1, 'school.application.created',
     '新学校入驻申请', '学校"明远理工大学"提交了平台入驻申请，请及时审核。',
     'school', 920000000000050002, 'school_application',
     FALSE, NULL, FALSE, NOW() - INTERVAL '2 days'),

    -- 超管：告警事件
    (920000000000070010, 910000000000001900, NULL, 1, 'system.alert.triggered',
     '系统告警：worker-01 内存使用率过高', '节点 worker-01 内存使用率达到 93.1%，触发紧急告警规则，请立即排查。',
     'system', 920000000000041002, 'alert_event',
     FALSE, NULL, FALSE, NOW() - INTERVAL '4 hours'),

    -- 超管：备份完成（已读）
    (920000000000070011, 910000000000001900, NULL, 1, 'system.backup.completed',
     '数据库备份完成', '自动备份任务已完成，备份文件大小 513MB，存储路径 /backups/2026-05-03_020000_lenschain.sql.gz。',
     'system', 920000000000043003, 'backup_record',
     TRUE, NOW() - INTERVAL '12 hours', FALSE, NOW() - INTERVAL '1 day'),

    -- 学生1（林叙安）：作业批改完成
    (920000000000070012, 910000000000001101, 910000000000000001, 2, 'assignment.graded',
     '作业已批改', '您在课程《区块链工程实践导论》提交的作业"链上环境配置报告"已批改完成，得分 88 分。',
     'course', 920000000000003001, 'assignment_submission',
     FALSE, NULL, FALSE, NOW() - INTERVAL '1 day'),

    -- 学生1（林叙安）：竞赛报名通过（已读）
    (920000000000070013, 910000000000001101, 910000000000000001, 4, 'ctf.registration.approved',
     '竞赛报名已通过', '您的队伍"链上先锋队"已成功报名参加"江海杯·智能合约安全挑战赛"。',
     'ctf', 920000000000025001, 'competition_registration',
     TRUE, NOW() - INTERVAL '8 days', FALSE, NOW() - INTERVAL '9 days'),

    -- 学生2（许知遥）：实验环境就绪
    (920000000000070014, 910000000000001102, 910000000000000001, 3, 'experiment.ready',
     '实验环境已就绪', '您在课程《区块链工程实践导论》中启动的实验"以太坊本地开发与部署实践"已准备就绪。',
     'experiment', NULL, 'experiment_instance',
     FALSE, NULL, FALSE, NOW() - INTERVAL '6 hours'),

    -- 学生3（宋观澜·学校2）：课程公告
    (920000000000070015, 910000000000001203, 910000000000000002, 2, 'course.announcement',
     '课程公告', '课程《链上数据分析实践》发布了新公告："链上数据课程开课通知"，请查看。',
     'course', 920000000000007004, 'course_announcement',
     FALSE, NULL, FALSE, NOW() - INTERVAL '5 days'),

    -- 学生3（宋观澜·学校2）：竞赛即将开始
    (920000000000070016, 910000000000001203, 910000000000000002, 4, 'ctf.competition.starting',
     '竞赛即将开始', '"全平台区块链 CTF 公开赛"将于 3 天后正式开赛，请确保队伍已完成准备。',
     'ctf', 920000000000020002, 'competition',
     FALSE, NULL, FALSE, NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;
