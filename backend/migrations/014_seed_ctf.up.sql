-- 014_seed_ctf.up.sql
-- CTF 竞赛种子数据（统一管理）。
-- 包含：
-- 1. 解题赛（competition_type=1）：2 场竞赛 + 4 道题目 + 队伍 + 报名
-- 2. 攻防对抗赛（competition_type=2）：1 场竞赛 + 3 道题目 + 分组 + 回合 + 攻防记录
--
-- 使用方式：在 010、011 之后执行。
-- 依赖：users、schools 等基础数据（来自 010）。

-- =====================================================================
-- 01. 解题赛题目
-- =====================================================================

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

-- =====================================================================
-- 02. 攻防赛题目（flag_type=3 链上状态验证，base_score 由 Token 机制决定）
-- =====================================================================

INSERT INTO challenges (
    id, title, description, category, difficulty, base_score, flag_type,
    runtime_mode, author_id, school_id, status, is_public, created_at, updated_at
)
VALUES
    (920000000000021005, '可重入 Vault 攻防',
     '一个存在重入漏洞的 Vault 合约。攻击方需利用漏洞窃取 Token，防守方需在限时内提交修复补丁。',
     'contract', 2, 0, 3, 1,
     910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW()),
    (920000000000021006, '权限后门攻防',
     '合约中隐藏了一个后门函数允许任意地址提取资金。攻击方需找到并利用后门，防守方需移除后门并保留核心功能。',
     'contract', 2, 0, 3, 1,
     910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW()),
    (920000000000021007, 'Flashloan 价格操纵',
     '一个 DeFi 价格预言机存在闪电贷操纵漏洞。攻击方操纵价格窃取资金，防守方修复预言机逻辑。',
     'contract', 3, 0, 3, 1,
     910000000000001001, 910000000000000001, 2, FALSE, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 03. 解题赛竞赛
-- =====================================================================

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
        1, 2,
        910000000000000001,
        910000000000001001,
        2, 3, 1, 30,
        3,
        NOW() - INTERVAL '14 days',
        NOW() - INTERVAL '3 days',
        NOW() - INTERVAL '2 days',
        NOW() + INTERVAL '5 days',
        NOW() + INTERVAL '4 days',
        '{"scoring_mode":"dynamic","initial_score":1000,"min_score":100,"decay_factor":20}'::jsonb,
        '1. 每队 1-3 人\n2. 不得使用自动化攻击工具\n3. Flag 提交频率限制 60 秒\n4. 最终排名以总分为准',
        NOW(), NOW()
    ),
    (
        920000000000020002,
        '全平台区块链 CTF 公开赛',
        '面向所有学校的跨校公开赛，涵盖合约安全、链上数据分析与 DeFi 协议审计。',
        1, 1,
        NULL,
        910000000000001900,
        2, 4, 1, 50,
        2,
        NOW() - INTERVAL '5 days',
        NOW() + INTERVAL '2 days',
        NOW() + INTERVAL '3 days',
        NOW() + INTERVAL '10 days',
        NOW() + INTERVAL '9 days',
        '{"scoring_mode":"dynamic","initial_score":1000,"min_score":100,"decay_factor":15}'::jsonb,
        '1. 每队 1-4 人\n2. 跨校组队允许\n3. 禁止共享 Flag\n4. 最终排名以总分为准',
        NOW(), NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 04. 攻防对抗赛竞赛
-- =====================================================================

INSERT INTO competitions (
    id, title, description, competition_type, scope, school_id, created_by,
    team_mode, max_team_size, min_team_size, max_teams, status,
    registration_start_at, registration_end_at, start_at, end_at, freeze_at,
    ad_config, rules, created_at, updated_at
)
VALUES
    (
        920000000000020003,
        '江海杯·智能合约攻防对抗赛',
        '校内攻防对抗赛，每支队伍拥有独立链环境，在限时回合中攻击对手合约漏洞并修补己方合约。',
        2, 2,
        910000000000000001,
        910000000000001001,
        2, 3, 2, 16,
        3,
        NOW() - INTERVAL '10 days',
        NOW() - INTERVAL '2 days',
        NOW() - INTERVAL '1 day',
        NOW() + INTERVAL '6 days',
        NOW() + INTERVAL '5 days',
        '{
            "round_count": 5,
            "attack_duration_minutes": 30,
            "defense_duration_minutes": 20,
            "initial_token": 10000,
            "attack_reward_token": 500,
            "defense_reward_token": 300,
            "first_blood_bonus": 200,
            "first_patch_bonus": 150,
            "steal_ratio": 0.1,
            "loss_ratio": 0.05
        }'::jsonb,
        '1. 每队 2-3 人\n2. 每回合分攻击与防守两阶段\n3. 攻击成功窃取目标队伍 Token\n4. 提交有效修复补丁可获得防守奖励\n5. 最终以 Token 余额排名',
        NOW(), NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 05. 解题赛 — 竞赛题目关联
-- =====================================================================

INSERT INTO competition_challenges (id, competition_id, challenge_id, sort_order, created_at)
VALUES
    -- 江海杯解题赛
    (920000000000022001, 920000000000020001, 920000000000021001, 1, NOW()),
    (920000000000022002, 920000000000020001, 920000000000021002, 2, NOW()),
    (920000000000022003, 920000000000020001, 920000000000021003, 3, NOW()),
    -- 全平台公开赛
    (920000000000022004, 920000000000020002, 920000000000021001, 1, NOW()),
    (920000000000022005, 920000000000020002, 920000000000021004, 2, NOW()),
    -- 攻防对抗赛
    (920000000000022006, 920000000000020003, 920000000000021005, 1, NOW()),
    (920000000000022007, 920000000000020003, 920000000000021006, 2, NOW()),
    (920000000000022008, 920000000000020003, 920000000000021007, 3, NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 06. 攻防赛分组（ad_groups）
-- =====================================================================

INSERT INTO ad_groups (id, competition_id, group_name, status, created_at, updated_at)
VALUES
    (920000000000026001, 920000000000020003, 'A 组', 2, NOW(), NOW()),
    (920000000000026002, 920000000000020003, 'B 组', 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 07. 团队（解题赛 + 攻防赛共用 teams 表）
-- =====================================================================

INSERT INTO teams (
    id, competition_id, name, captain_id, invite_code, status,
    ad_group_id, token_balance,
    created_at, updated_at
)
VALUES
    -- 解题赛队伍
    (920000000000023001, 920000000000020001, '链上先锋队', 910000000000001101, 'TEAM01A', 1, NULL, NULL, NOW(), NOW()),
    (920000000000023002, 920000000000020001, '合约猎人组', 910000000000001102, 'TEAM02B', 1, NULL, NULL, NOW(), NOW()),
    (920000000000023003, 920000000000020002, '跨校联合战队', 910000000000001101, 'TEAM03C', 1, NULL, NULL, NOW(), NOW()),
    -- 攻防赛队伍（A 组 3 支队伍）
    (920000000000023004, 920000000000020003, '漏洞猎手',   910000000000001101, 'AD01AA', 2, 920000000000026001, 10000, NOW(), NOW()),
    (920000000000023005, 920000000000020003, '安全壁垒',   910000000000001102, 'AD02BB', 2, 920000000000026001, 10000, NOW(), NOW()),
    (920000000000023006, 920000000000020003, '链盾战队',   910000000000001203, 'AD03CC', 2, 920000000000026001, 10000, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 08. 团队成员
-- =====================================================================

INSERT INTO team_members (id, team_id, student_id, role, joined_at)
VALUES
    -- 解题赛
    (920000000000024001, 920000000000023001, 910000000000001101, 1, NOW()),   -- 学生1 队长·链上先锋队
    (920000000000024003, 920000000000023002, 910000000000001102, 1, NOW()),   -- 学生2 队长·合约猎人组
    (920000000000024004, 920000000000023003, 910000000000001101, 1, NOW()),   -- 学生1 队长·跨校联合战队
    (920000000000024005, 920000000000023003, 910000000000001203, 2, NOW()),   -- 学生3 成员·跨校联合战队
    -- 攻防赛
    (920000000000024006, 920000000000023004, 910000000000001101, 1, NOW()),   -- 学生1 队长·漏洞猎手
    (920000000000024007, 920000000000023004, 910000000000001102, 2, NOW()),   -- 学生2 成员·漏洞猎手
    (920000000000024008, 920000000000023005, 910000000000001102, 1, NOW()),   -- 学生2 队长·安全壁垒（不同竞赛）
    (920000000000024009, 920000000000023005, 910000000000001101, 2, NOW()),   -- 学生1 成员·安全壁垒（不同竞赛）
    (920000000000024010, 920000000000023006, 910000000000001203, 1, NOW()),   -- 学生3 队长·链盾战队
    (920000000000024011, 920000000000023006, 910000000000001102, 2, NOW())    -- 学生2 成员·链盾战队（不同竞赛）
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 09. 竞赛报名
-- =====================================================================

INSERT INTO competition_registrations (id, competition_id, team_id, registered_by, status, created_at)
VALUES
    -- 解题赛报名（status: 1=已报名）
    (920000000000025001, 920000000000020001, 920000000000023001, 910000000000001101, 1, NOW() - INTERVAL '10 days'),
    (920000000000025002, 920000000000020001, 920000000000023002, 910000000000001102, 1, NOW() - INTERVAL '9 days'),
    (920000000000025003, 920000000000020002, 920000000000023003, 910000000000001101, 1, NOW() - INTERVAL '3 days'),
    -- 攻防赛报名（status: 1=已报名）
    (920000000000025004, 920000000000020003, 920000000000023004, 910000000000001101, 1, NOW() - INTERVAL '5 days'),
    (920000000000025005, 920000000000020003, 920000000000023005, 910000000000001102, 1, NOW() - INTERVAL '5 days'),
    (920000000000025006, 920000000000020003, 920000000000023006, 910000000000001203, 1, NOW() - INTERVAL '4 days')
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 10. 攻防赛队伍链（ad_team_chains）
-- =====================================================================

INSERT INTO ad_team_chains (
    id, competition_id, group_id, team_id,
    chain_rpc_url, chain_ws_url, deployed_contracts,
    current_patch_version, status, created_at, updated_at
)
VALUES
    (920000000000027001, 920000000000020003, 920000000000026001, 920000000000023004,
     'http://ad-chain-team4:8545', 'ws://ad-chain-team4:8546',
     '{"ReentrancyVault":"0x5FbDB2315678afecb367f032d93F642f64180aa3","BackdoorWallet":"0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512","FlashLoanOracle":"0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"}'::jsonb,
     0, 2, NOW(), NOW()),
    (920000000000027002, 920000000000020003, 920000000000026001, 920000000000023005,
     'http://ad-chain-team5:8545', 'ws://ad-chain-team5:8546',
     '{"ReentrancyVault":"0x5FbDB2315678afecb367f032d93F642f64180aa3","BackdoorWallet":"0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512","FlashLoanOracle":"0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"}'::jsonb,
     0, 2, NOW(), NOW()),
    (920000000000027003, 920000000000020003, 920000000000026001, 920000000000023006,
     'http://ad-chain-team6:8545', 'ws://ad-chain-team6:8546',
     '{"ReentrancyVault":"0x5FbDB2315678afecb367f032d93F642f64180aa3","BackdoorWallet":"0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512","FlashLoanOracle":"0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"}'::jsonb,
     0, 2, NOW(), NOW())
ON CONFLICT (competition_id, team_id) DO NOTHING;

-- =====================================================================
-- 11. 攻防赛回合（ad_rounds）— 已完成 2 轮，第 3 轮进行中
-- =====================================================================

INSERT INTO ad_rounds (
    id, competition_id, group_id, round_number, phase,
    attack_start_at, attack_end_at, defense_start_at, defense_end_at,
    settlement_start_at, settlement_end_at, settlement_result,
    created_at, updated_at
)
VALUES
    -- 第 1 轮（已完成）
    (920000000000028001, 920000000000020003, 920000000000026001, 1, 4,
     NOW() - INTERVAL '1 day',
     NOW() - INTERVAL '1 day' + INTERVAL '30 minutes',
     NOW() - INTERVAL '1 day' + INTERVAL '30 minutes',
     NOW() - INTERVAL '1 day' + INTERVAL '50 minutes',
     NOW() - INTERVAL '1 day' + INTERVAL '50 minutes',
     NOW() - INTERVAL '1 day' + INTERVAL '55 minutes',
     '{"total_attacks":6,"successful_attacks":3,"patches_submitted":4,"patches_accepted":3}'::jsonb,
     NOW(), NOW()),
    -- 第 2 轮（已完成）
    (920000000000028002, 920000000000020003, 920000000000026001, 2, 4,
     NOW() - INTERVAL '20 hours',
     NOW() - INTERVAL '20 hours' + INTERVAL '30 minutes',
     NOW() - INTERVAL '20 hours' + INTERVAL '30 minutes',
     NOW() - INTERVAL '20 hours' + INTERVAL '50 minutes',
     NOW() - INTERVAL '20 hours' + INTERVAL '50 minutes',
     NOW() - INTERVAL '20 hours' + INTERVAL '55 minutes',
     '{"total_attacks":8,"successful_attacks":4,"patches_submitted":5,"patches_accepted":4}'::jsonb,
     NOW(), NOW()),
    -- 第 3 轮（攻击阶段进行中）
    (920000000000028003, 920000000000020003, 920000000000026001, 3, 1,
     NOW() - INTERVAL '10 minutes',
     NULL, NULL, NULL, NULL, NULL, NULL,
     NOW(), NOW())
ON CONFLICT (competition_id, group_id, round_number) DO NOTHING;

-- =====================================================================
-- 12. 攻防赛攻击记录（ad_attacks）
-- =====================================================================

INSERT INTO ad_attacks (
    id, competition_id, round_id, attacker_team_id, target_team_id,
    challenge_id, attack_tx_data, is_successful, assertion_results,
    token_reward, is_first_blood, created_at
)
VALUES
    -- 第 1 轮攻击
    (920000000000029001, 920000000000020003, 920000000000028001,
     920000000000023004, 920000000000023005, 920000000000021005,
     '0x2e1a7d4d000000000000000000000000000000000000000000000000000000000001',
     TRUE,
     '[{"type":"balance_check","target":"ReentrancyVault","operator":"lt","expected":"1000","actual":"0","passed":true}]'::jsonb,
     500, TRUE, NOW() - INTERVAL '1 day' + INTERVAL '12 minutes'),
    (920000000000029002, 920000000000020003, 920000000000028001,
     920000000000023005, 920000000000023006, 920000000000021006,
     '0x3ccfd60b',
     TRUE,
     '[{"type":"balance_check","target":"BackdoorWallet","operator":"eq","expected":"0","actual":"0","passed":true}]'::jsonb,
     500, TRUE, NOW() - INTERVAL '1 day' + INTERVAL '18 minutes'),
    (920000000000029003, 920000000000020003, 920000000000028001,
     920000000000023006, 920000000000023004, 920000000000021007,
     '0x38ed1739',
     FALSE, NULL, 0, FALSE, NOW() - INTERVAL '1 day' + INTERVAL '25 minutes'),
    -- 第 2 轮攻击
    (920000000000029004, 920000000000020003, 920000000000028002,
     920000000000023004, 920000000000023006, 920000000000021005,
     '0x2e1a7d4d000000000000000000000000000000000000000000000000000000000001',
     TRUE,
     '[{"type":"balance_check","target":"ReentrancyVault","operator":"lt","expected":"1000","actual":"0","passed":true}]'::jsonb,
     500, FALSE, NOW() - INTERVAL '20 hours' + INTERVAL '8 minutes'),
    (920000000000029005, 920000000000020003, 920000000000028002,
     920000000000023006, 920000000000023004, 920000000000021006,
     '0x3ccfd60b',
     TRUE,
     '[{"type":"balance_check","target":"BackdoorWallet","operator":"eq","expected":"0","actual":"0","passed":true}]'::jsonb,
     500, FALSE, NOW() - INTERVAL '20 hours' + INTERVAL '15 minutes'),
    (920000000000029006, 920000000000020003, 920000000000028002,
     920000000000023005, 920000000000023004, 920000000000021007,
     '0x38ed1739',
     TRUE,
     '[{"type":"balance_check","target":"FlashLoanOracle","operator":"lt","expected":"5000","actual":"0","passed":true}]'::jsonb,
     500, TRUE, NOW() - INTERVAL '20 hours' + INTERVAL '22 minutes')
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 13. 攻防赛防守记录（ad_defenses）
-- =====================================================================

INSERT INTO ad_defenses (
    id, competition_id, round_id, team_id, challenge_id,
    patch_source_code, is_accepted, functionality_passed, vulnerability_fixed,
    is_first_patch, token_reward, rejection_reason, created_at
)
VALUES
    -- 第 1 轮防守
    (920000000000029101, 920000000000020003, 920000000000028001,
     920000000000023005, 920000000000021005,
     'function withdraw(uint256 amount) external { require(balances[msg.sender] >= amount); balances[msg.sender] -= amount; totalDeposited -= amount; (bool s,) = msg.sender.call{value: amount}(""); require(s); }',
     TRUE, TRUE, TRUE, TRUE, 300 + 150, NULL,
     NOW() - INTERVAL '1 day' + INTERVAL '35 minutes'),
    (920000000000029102, 920000000000020003, 920000000000028001,
     920000000000023006, 920000000000021006,
     'function emergencyWithdraw() external onlyOwner { payable(owner).transfer(address(this).balance); }',
     TRUE, TRUE, TRUE, TRUE, 300 + 150, NULL,
     NOW() - INTERVAL '1 day' + INTERVAL '40 minutes'),
    (920000000000029103, 920000000000020003, 920000000000028001,
     920000000000023004, 920000000000021007,
     'function getPrice() public view returns (uint256) { return twapOracle.consult(token, 1e18); }',
     FALSE, TRUE, FALSE, FALSE, 0, '补丁未能有效修复价格操纵漏洞，仍可通过单区块内大额交易影响价格。',
     NOW() - INTERVAL '1 day' + INTERVAL '45 minutes'),
    -- 第 2 轮防守
    (920000000000029104, 920000000000020003, 920000000000028002,
     920000000000023004, 920000000000021007,
     'function getPrice() public view returns (uint256) { require(block.number > lastUpdateBlock + MIN_BLOCKS); return chainlinkOracle.latestAnswer(); }',
     TRUE, TRUE, TRUE, FALSE, 300, NULL,
     NOW() - INTERVAL '20 hours' + INTERVAL '38 minutes'),
    (920000000000029105, 920000000000020003, 920000000000028002,
     920000000000023004, 920000000000021006,
     'function emergencyWithdraw() external { require(msg.sender == owner, "Not owner"); payable(owner).transfer(address(this).balance); }',
     TRUE, TRUE, TRUE, TRUE, 300 + 150, NULL,
     NOW() - INTERVAL '20 hours' + INTERVAL '42 minutes')
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 14. Token 流水账（ad_token_ledger）
-- =====================================================================

INSERT INTO ad_token_ledger (
    id, competition_id, group_id, round_id, team_id,
    change_type, amount, balance_after,
    related_attack_id, related_defense_id, description, created_at
)
VALUES
    -- 初始化 Token（3 支队伍各 10000）
    (920000000000029201, 920000000000020003, 920000000000026001, NULL, 920000000000023004,
     1, 10000, 10000, NULL, NULL, '初始化 Token', NOW() - INTERVAL '1 day'),
    (920000000000029202, 920000000000020003, 920000000000026001, NULL, 920000000000023005,
     1, 10000, 10000, NULL, NULL, '初始化 Token', NOW() - INTERVAL '1 day'),
    (920000000000029203, 920000000000020003, 920000000000026001, NULL, 920000000000023006,
     1, 10000, 10000, NULL, NULL, '初始化 Token', NOW() - INTERVAL '1 day'),

    -- 第 1 轮：漏洞猎手攻击安全壁垒成功 (+500 攻击奖励, +1000 窃取)
    (920000000000029204, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023004,
     3, 500, 10500, 920000000000029001, NULL, '攻击奖励：可重入 Vault 攻防', NOW() - INTERVAL '1 day' + INTERVAL '12 minutes'),
    (920000000000029205, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023004,
     2, 1000, 11500, 920000000000029001, NULL, '攻击窃取：可重入 Vault 攻防', NOW() - INTERVAL '1 day' + INTERVAL '12 minutes'),
    (920000000000029206, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023004,
     7, 200, 11700, 920000000000029001, NULL, 'First Blood 奖励', NOW() - INTERVAL '1 day' + INTERVAL '12 minutes'),
    -- 安全壁垒被攻击 (-500)
    (920000000000029207, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023005,
     4, -500, 9500, 920000000000029001, NULL, '被攻击扣除：可重入 Vault', NOW() - INTERVAL '1 day' + INTERVAL '12 minutes'),

    -- 第 1 轮：安全壁垒攻击链盾成功 (+500 攻击奖励)
    (920000000000029208, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023005,
     3, 500, 10000, 920000000000029002, NULL, '攻击奖励：权限后门攻防', NOW() - INTERVAL '1 day' + INTERVAL '18 minutes'),
    (920000000000029209, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023005,
     7, 200, 10200, 920000000000029002, NULL, 'First Blood 奖励', NOW() - INTERVAL '1 day' + INTERVAL '18 minutes'),
    -- 链盾被攻击 (-500)
    (920000000000029210, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023006,
     4, -500, 9500, 920000000000029002, NULL, '被攻击扣除：权限后门', NOW() - INTERVAL '1 day' + INTERVAL '18 minutes'),

    -- 第 1 轮防守奖励：安全壁垒修复重入 (+300 + 150 首补丁)
    (920000000000029211, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023005,
     5, 300, 10500, NULL, 920000000000029101, '防守奖励：修复可重入 Vault', NOW() - INTERVAL '1 day' + INTERVAL '35 minutes'),
    (920000000000029212, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023005,
     6, 150, 10650, NULL, 920000000000029101, '首补丁奖励', NOW() - INTERVAL '1 day' + INTERVAL '35 minutes'),
    -- 链盾修复后门 (+300 + 150 首补丁)
    (920000000000029213, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023006,
     5, 300, 9800, NULL, 920000000000029102, '防守奖励：修复权限后门', NOW() - INTERVAL '1 day' + INTERVAL '40 minutes'),
    (920000000000029214, 920000000000020003, 920000000000026001, 920000000000028001, 920000000000023006,
     6, 150, 9950, NULL, 920000000000029102, '首补丁奖励', NOW() - INTERVAL '1 day' + INTERVAL '40 minutes')
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 15. 排行榜快照
-- =====================================================================

INSERT INTO leaderboard_snapshots (
    id, competition_id, team_id, rank, score, solve_count, last_solve_at,
    is_frozen, snapshot_at, created_at
)
VALUES
    -- 解题赛排行榜（江海杯）
    (920000000000029301, 920000000000020001, 920000000000023001, 1, 1300, 2, NOW() - INTERVAL '3 days', FALSE, NOW() - INTERVAL '1 day', NOW()),
    (920000000000029302, 920000000000020001, 920000000000023002, 2, 500, 1, NOW() - INTERVAL '4 days', FALSE, NOW() - INTERVAL '1 day', NOW()),
    -- 攻防赛排行榜（A 组，第 1 轮后）
    (920000000000029303, 920000000000020003, 920000000000023004, 1, 11700, NULL, NULL, FALSE, NOW() - INTERVAL '23 hours', NOW()),
    (920000000000029304, 920000000000020003, 920000000000023005, 2, 10650, NULL, NULL, FALSE, NOW() - INTERVAL '23 hours', NOW()),
    (920000000000029305, 920000000000020003, 920000000000023006, 3, 9950, NULL, NULL, FALSE, NOW() - INTERVAL '23 hours', NOW())
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 16. 解题赛提交记录
-- =====================================================================

INSERT INTO submissions (
    id, competition_id, challenge_id, team_id, student_id,
    submission_type, content, is_correct, score_awarded, is_first_blood,
    created_at
)
VALUES
    -- 链上先锋队 解题
    (920000000000029401, 920000000000020001, 920000000000021001, 920000000000023001, 910000000000001101,
     1, 'flag{reentrancy_solved_2026}', TRUE, 500, TRUE, NOW() - INTERVAL '4 days'),
    (920000000000029402, 920000000000020001, 920000000000021002, 920000000000023001, 910000000000001101,
     1, 'flag{overflow_bypassed_2026}', TRUE, 800, TRUE, NOW() - INTERVAL '3 days'),
    -- 合约猎人组 解题
    (920000000000029403, 920000000000020001, 920000000000021001, 920000000000023002, 910000000000001102,
     1, 'flag{reentrancy_solved_2026}', TRUE, 500, FALSE, NOW() - INTERVAL '4 days' + INTERVAL '2 hours'),
    -- 错误提交
    (920000000000029404, 920000000000020001, 920000000000021003, 920000000000023001, 910000000000001101,
     1, 'flag{wrong_flag}', FALSE, 0, FALSE, NOW() - INTERVAL '3 days' + INTERVAL '1 hour')
ON CONFLICT (id) DO NOTHING;

-- =====================================================================
-- 17. CTF 资源配额
-- =====================================================================

INSERT INTO ctf_resource_quotas (
    id, competition_id, max_cpu, max_memory, max_storage, max_namespaces,
    used_cpu, used_memory, used_storage, current_namespaces,
    created_at, updated_at
)
VALUES
    (920000000000029501, 920000000000020001, '8000m', '16Gi', '50Gi', 30, '2400m', '4Gi', '8Gi', 6, NOW(), NOW()),
    (920000000000029502, 920000000000020002, '16000m', '32Gi', '100Gi', 50, '0', '0', '0', 0, NOW(), NOW()),
    (920000000000029503, 920000000000020003, '12000m', '24Gi', '60Gi', 20, '3600m', '6Gi', '12Gi', 6, NOW(), NOW())
ON CONFLICT (competition_id) DO NOTHING;

-- =====================================================================
-- 18. 竞赛公告
-- =====================================================================

INSERT INTO announcements (
    id, competition_id, challenge_id, title, content,
    announcement_type, published_by, created_at
)
VALUES
    (920000000000029601, 920000000000020001, NULL,
     '竞赛开始公告', '江海杯·智能合约安全挑战赛已正式开赛！祝各位选手取得好成绩。',
     1, 910000000000001001, NOW() - INTERVAL '2 days'),
    (920000000000029602, 920000000000020003, NULL,
     '攻防赛规则说明', '每回合攻击阶段 30 分钟，防守阶段 20 分钟。攻击成功可窃取目标队伍 Token，提交有效补丁获得防守奖励。',
     3, 910000000000001001, NOW() - INTERVAL '1 day'),
    (920000000000029603, 920000000000020003, 920000000000021005,
     '题目勘误：可重入 Vault', '补充说明：Vault 合约初始部署时会由平台注入 1 ETH 作为初始资金。',
     2, 910000000000001001, NOW() - INTERVAL '22 hours')
ON CONFLICT (id) DO NOTHING;
