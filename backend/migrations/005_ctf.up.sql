-- 模块05 CTF竞赛
-- 文档依据：
-- 1. docs/modules/05-CTF竞赛/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 竞赛、题目、模板、审核、预验证
-- 2. 团队、报名、提交、排行榜、公告
-- 3. 攻防赛分组、回合、攻击、防守、Token流水
-- 不负责：
-- 1. 竞赛期间 Redis 排行榜缓存
-- 2. 题目环境实际编排执行器

-- competitions：竞赛主表。
CREATE TABLE competitions (
    id BIGINT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    banner_url VARCHAR(500) NULL,
    competition_type SMALLINT NOT NULL,
    scope SMALLINT NOT NULL DEFAULT 1,
    school_id BIGINT NULL,
    created_by BIGINT NOT NULL,
    team_mode SMALLINT NOT NULL DEFAULT 1,
    max_team_size INT NOT NULL DEFAULT 1,
    min_team_size INT NOT NULL DEFAULT 1,
    max_teams INT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    registration_start_at TIMESTAMP NULL,
    registration_end_at TIMESTAMP NULL,
    start_at TIMESTAMP NULL,
    end_at TIMESTAMP NULL,
    freeze_at TIMESTAMP NULL,
    jeopardy_config JSONB NULL,
    ad_config JSONB NULL,
    rules TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_competitions_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_competitions_created_by FOREIGN KEY (created_by) REFERENCES users(id)
);
CREATE INDEX idx_competitions_competition_type ON competitions(competition_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_competitions_scope ON competitions(scope) WHERE deleted_at IS NULL;
CREATE INDEX idx_competitions_school_id ON competitions(school_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_competitions_status ON competitions(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_competitions_created_by ON competitions(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_competitions_start_at ON competitions(start_at) WHERE deleted_at IS NULL;

-- challenge_templates：参数化模板库表。
CREATE TABLE challenge_templates (
    id BIGINT PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(100) NOT NULL,
    description TEXT NULL,
    vulnerability_type VARCHAR(100) NOT NULL,
    base_source_code TEXT NOT NULL,
    base_assertions JSONB NOT NULL,
    base_setup_transactions JSONB NOT NULL,
    parameters JSONB NOT NULL,
    variants JSONB NULL,
    reference_events JSONB NULL,
    difficulty_range JSONB NOT NULL,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_challenge_templates_code ON challenge_templates(code);
CREATE INDEX idx_challenge_templates_vulnerability_type ON challenge_templates(vulnerability_type);

-- challenges：题目主表。
CREATE TABLE challenges (
    id BIGINT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(20) NOT NULL,
    difficulty SMALLINT NOT NULL,
    base_score INT NOT NULL,
    flag_type SMALLINT NOT NULL DEFAULT 1,
    static_flag VARCHAR(500) NULL,
    dynamic_flag_secret VARCHAR(200) NULL,
    runtime_mode SMALLINT NOT NULL DEFAULT 1,
    chain_config JSONB NULL,
    setup_transactions JSONB NULL,
    source_path SMALLINT NULL,
    swc_id VARCHAR(20) NULL,
    template_id BIGINT NULL,
    template_params JSONB NULL,
    environment_config JSONB NULL,
    attachment_urls JSONB NULL,
    author_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_challenges_template_id FOREIGN KEY (template_id) REFERENCES challenge_templates(id),
    CONSTRAINT fk_challenges_author_id FOREIGN KEY (author_id) REFERENCES users(id),
    CONSTRAINT fk_challenges_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE INDEX idx_challenges_category ON challenges(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_difficulty ON challenges(difficulty) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_flag_type ON challenges(flag_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_author_id ON challenges(author_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_school_id ON challenges(school_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_status ON challenges(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_challenges_runtime_mode ON challenges(runtime_mode) WHERE deleted_at IS NULL;

-- challenge_contracts：题目合约表。
CREATE TABLE challenge_contracts (
    id BIGINT PRIMARY KEY,
    challenge_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    source_code TEXT NOT NULL,
    abi JSONB NOT NULL,
    bytecode TEXT NOT NULL,
    constructor_args JSONB NULL,
    deploy_order INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_challenge_contracts_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id)
);
CREATE INDEX idx_challenge_contracts_challenge_id ON challenge_contracts(challenge_id);

-- challenge_assertions：验证断言表。
CREATE TABLE challenge_assertions (
    id BIGINT PRIMARY KEY,
    challenge_id BIGINT NOT NULL,
    assertion_type VARCHAR(30) NOT NULL,
    target VARCHAR(200) NOT NULL,
    operator VARCHAR(10) NOT NULL,
    expected_value TEXT NOT NULL,
    description VARCHAR(500) NULL,
    extra_params JSONB NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_challenge_assertions_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id)
);
CREATE INDEX idx_challenge_assertions_challenge_id ON challenge_assertions(challenge_id);

-- challenge_reviews：题目审核记录表。
CREATE TABLE challenge_reviews (
    id BIGINT PRIMARY KEY,
    challenge_id BIGINT NOT NULL,
    reviewer_id BIGINT NOT NULL,
    action SMALLINT NOT NULL,
    comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_challenge_reviews_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id),
    CONSTRAINT fk_challenge_reviews_reviewer_id FOREIGN KEY (reviewer_id) REFERENCES users(id)
);
CREATE INDEX idx_challenge_reviews_challenge_id ON challenge_reviews(challenge_id);
CREATE INDEX idx_challenge_reviews_reviewer_id ON challenge_reviews(reviewer_id);

-- challenge_verifications：题目预验证记录表。
CREATE TABLE challenge_verifications (
    id BIGINT PRIMARY KEY,
    challenge_id BIGINT NOT NULL,
    initiated_by BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    step_results JSONB NOT NULL DEFAULT '[]',
    poc_content TEXT NULL,
    poc_language VARCHAR(20) NULL,
    environment_id VARCHAR(200) NULL,
    error_message TEXT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_challenge_verifications_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id),
    CONSTRAINT fk_challenge_verifications_initiated_by FOREIGN KEY (initiated_by) REFERENCES users(id)
);
CREATE INDEX idx_challenge_verifications_challenge_id ON challenge_verifications(challenge_id);
CREATE INDEX idx_challenge_verifications_status ON challenge_verifications(status);

-- competition_challenges：竞赛题目关联表。
CREATE TABLE competition_challenges (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    challenge_id BIGINT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    current_score INT NULL,
    solve_count INT NOT NULL DEFAULT 0,
    first_blood_team_id BIGINT NULL,
    first_blood_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_competition_challenges_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_competition_challenges_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id)
);
CREATE UNIQUE INDEX uk_competition_challenges ON competition_challenges(competition_id, challenge_id);
CREATE INDEX idx_competition_challenges_challenge_id ON competition_challenges(challenge_id);

-- ad_groups：攻防赛分组表。
CREATE TABLE ad_groups (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    group_name VARCHAR(100) NOT NULL,
    namespace VARCHAR(100) NULL,
    judge_chain_url VARCHAR(500) NULL,
    judge_contract_address VARCHAR(100) NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_groups_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id)
);
CREATE INDEX idx_ad_groups_competition_id ON ad_groups(competition_id);
CREATE INDEX idx_ad_groups_status ON ad_groups(status);

-- teams：参赛团队表。
CREATE TABLE teams (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    captain_id BIGINT NOT NULL,
    invite_code VARCHAR(20) NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    ad_group_id BIGINT NULL,
    token_balance INT NULL,
    final_rank INT NULL,
    total_score INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_teams_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_teams_captain_id FOREIGN KEY (captain_id) REFERENCES users(id),
    CONSTRAINT fk_teams_ad_group_id FOREIGN KEY (ad_group_id) REFERENCES ad_groups(id)
);
CREATE INDEX idx_teams_competition_id ON teams(competition_id);
CREATE INDEX idx_teams_captain_id ON teams(captain_id);
CREATE UNIQUE INDEX uk_teams_invite_code ON teams(invite_code) WHERE invite_code IS NOT NULL;
CREATE INDEX idx_teams_ad_group_id ON teams(ad_group_id) WHERE ad_group_id IS NOT NULL;
CREATE INDEX idx_teams_status ON teams(status);

-- team_members：团队成员表。
CREATE TABLE team_members (
    id BIGINT PRIMARY KEY,
    team_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    role SMALLINT NOT NULL DEFAULT 2,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_team_members_team_id FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT fk_team_members_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_team_members ON team_members(team_id, student_id);
CREATE INDEX idx_team_members_student_id ON team_members(student_id);

-- competition_registrations：竞赛报名表。
CREATE TABLE competition_registrations (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    registered_by BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_competition_registrations_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_competition_registrations_team_id FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT fk_competition_registrations_registered_by FOREIGN KEY (registered_by) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_competition_registrations ON competition_registrations(competition_id, team_id);
CREATE INDEX idx_competition_registrations_team_id ON competition_registrations(team_id);

-- submissions：提交记录表。
CREATE TABLE submissions (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    challenge_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    submission_type SMALLINT NOT NULL,
    content TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    score_awarded INT NULL,
    is_first_blood BOOLEAN NOT NULL DEFAULT FALSE,
    assertion_results JSONB NULL,
    error_message TEXT NULL,
    namespace VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_submissions_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_submissions_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id),
    CONSTRAINT fk_submissions_team_id FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT fk_submissions_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE INDEX idx_ctf_submissions_competition_id ON submissions(competition_id);
CREATE INDEX idx_ctf_submissions_challenge_id ON submissions(challenge_id);
CREATE INDEX idx_ctf_submissions_team_id ON submissions(team_id);
CREATE INDEX idx_ctf_submissions_student_id ON submissions(student_id);
CREATE INDEX idx_ctf_submissions_is_correct ON submissions(is_correct) WHERE is_correct = TRUE;
CREATE INDEX idx_ctf_submissions_created_at ON submissions(created_at);

-- ad_rounds：攻防赛回合表。
CREATE TABLE ad_rounds (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    round_number INT NOT NULL,
    phase SMALLINT NOT NULL DEFAULT 1,
    attack_start_at TIMESTAMP NULL,
    attack_end_at TIMESTAMP NULL,
    defense_start_at TIMESTAMP NULL,
    defense_end_at TIMESTAMP NULL,
    settlement_start_at TIMESTAMP NULL,
    settlement_end_at TIMESTAMP NULL,
    settlement_result JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_rounds_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_ad_rounds_group_id FOREIGN KEY (group_id) REFERENCES ad_groups(id)
);
CREATE UNIQUE INDEX uk_ad_rounds ON ad_rounds(competition_id, group_id, round_number);
CREATE INDEX idx_ad_rounds_group_id ON ad_rounds(group_id);
CREATE INDEX idx_ad_rounds_phase ON ad_rounds(phase);

-- ad_attacks：攻防赛攻击记录表。
CREATE TABLE ad_attacks (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    round_id BIGINT NOT NULL,
    attacker_team_id BIGINT NOT NULL,
    target_team_id BIGINT NOT NULL,
    challenge_id BIGINT NOT NULL,
    attack_tx_data TEXT NOT NULL,
    is_successful BOOLEAN NOT NULL DEFAULT FALSE,
    assertion_results JSONB NULL,
    token_reward INT NULL,
    exploit_count INT NULL,
    is_first_blood BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_attacks_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_ad_attacks_round_id FOREIGN KEY (round_id) REFERENCES ad_rounds(id),
    CONSTRAINT fk_ad_attacks_attacker_team_id FOREIGN KEY (attacker_team_id) REFERENCES teams(id),
    CONSTRAINT fk_ad_attacks_target_team_id FOREIGN KEY (target_team_id) REFERENCES teams(id),
    CONSTRAINT fk_ad_attacks_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id)
);
CREATE INDEX idx_ad_attacks_competition_id ON ad_attacks(competition_id);
CREATE INDEX idx_ad_attacks_round_id ON ad_attacks(round_id);
CREATE INDEX idx_ad_attacks_attacker_team_id ON ad_attacks(attacker_team_id);
CREATE INDEX idx_ad_attacks_target_team_id ON ad_attacks(target_team_id);
CREATE INDEX idx_ad_attacks_challenge_id ON ad_attacks(challenge_id);
CREATE INDEX idx_ad_attacks_is_successful ON ad_attacks(is_successful) WHERE is_successful = TRUE;

-- ad_defenses：攻防赛防守记录表。
CREATE TABLE ad_defenses (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    round_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    challenge_id BIGINT NOT NULL,
    patch_source_code TEXT NOT NULL,
    is_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    functionality_passed BOOLEAN NULL,
    vulnerability_fixed BOOLEAN NULL,
    is_first_patch BOOLEAN NOT NULL DEFAULT FALSE,
    token_reward INT NULL,
    rejection_reason TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_defenses_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_ad_defenses_round_id FOREIGN KEY (round_id) REFERENCES ad_rounds(id),
    CONSTRAINT fk_ad_defenses_team_id FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT fk_ad_defenses_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id)
);
CREATE INDEX idx_ad_defenses_competition_id ON ad_defenses(competition_id);
CREATE INDEX idx_ad_defenses_round_id ON ad_defenses(round_id);
CREATE INDEX idx_ad_defenses_team_id ON ad_defenses(team_id);
CREATE INDEX idx_ad_defenses_challenge_id ON ad_defenses(challenge_id);

-- ad_token_ledger：Token流水账表。
CREATE TABLE ad_token_ledger (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    round_id BIGINT NULL,
    team_id BIGINT NOT NULL,
    change_type SMALLINT NOT NULL,
    amount INT NOT NULL,
    balance_after INT NOT NULL,
    related_attack_id BIGINT NULL,
    related_defense_id BIGINT NULL,
    description VARCHAR(500) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_token_ledger_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_ad_token_ledger_group_id FOREIGN KEY (group_id) REFERENCES ad_groups(id),
    CONSTRAINT fk_ad_token_ledger_round_id FOREIGN KEY (round_id) REFERENCES ad_rounds(id),
    CONSTRAINT fk_ad_token_ledger_team_id FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT fk_ad_token_ledger_related_attack_id FOREIGN KEY (related_attack_id) REFERENCES ad_attacks(id),
    CONSTRAINT fk_ad_token_ledger_related_defense_id FOREIGN KEY (related_defense_id) REFERENCES ad_defenses(id)
);
CREATE INDEX idx_ad_token_ledger_competition_id ON ad_token_ledger(competition_id);
CREATE INDEX idx_ad_token_ledger_group_id ON ad_token_ledger(group_id);
CREATE INDEX idx_ad_token_ledger_team_id ON ad_token_ledger(team_id);
CREATE INDEX idx_ad_token_ledger_round_id ON ad_token_ledger(round_id);
CREATE INDEX idx_ad_token_ledger_created_at ON ad_token_ledger(created_at);

-- leaderboard_snapshots：排行榜快照表。
CREATE TABLE leaderboard_snapshots (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    rank INT NOT NULL,
    score INT NOT NULL,
    solve_count INT NULL,
    last_solve_at TIMESTAMP NULL,
    is_frozen BOOLEAN NOT NULL DEFAULT FALSE,
    snapshot_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_leaderboard_snapshots_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_leaderboard_snapshots_team_id FOREIGN KEY (team_id) REFERENCES teams(id)
);
CREATE INDEX idx_leaderboard_snapshots_competition_id ON leaderboard_snapshots(competition_id);
CREATE INDEX idx_leaderboard_snapshots_team_id ON leaderboard_snapshots(team_id);
CREATE INDEX idx_leaderboard_snapshots_snapshot_at ON leaderboard_snapshots(snapshot_at);
CREATE INDEX idx_leaderboard_snapshots_rank ON leaderboard_snapshots(competition_id, snapshot_at, rank);

-- announcements：竞赛公告表。
CREATE TABLE announcements (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    challenge_id BIGINT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    announcement_type SMALLINT NOT NULL DEFAULT 1,
    published_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_announcements_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_announcements_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id),
    CONSTRAINT fk_announcements_published_by FOREIGN KEY (published_by) REFERENCES users(id)
);
CREATE INDEX idx_announcements_competition_id ON announcements(competition_id);
CREATE INDEX idx_announcements_challenge_id ON announcements(challenge_id) WHERE challenge_id IS NOT NULL;
CREATE INDEX idx_announcements_created_at ON announcements(created_at);

-- ctf_resource_quotas：CTF资源配额表。
CREATE TABLE ctf_resource_quotas (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    max_cpu VARCHAR(20) NULL,
    max_memory VARCHAR(20) NULL,
    max_storage VARCHAR(20) NULL,
    max_namespaces INT NULL,
    used_cpu VARCHAR(20) NOT NULL DEFAULT '0',
    used_memory VARCHAR(20) NOT NULL DEFAULT '0',
    used_storage VARCHAR(20) NOT NULL DEFAULT '0',
    current_namespaces INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ctf_resource_quotas_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id)
);
CREATE UNIQUE INDEX uk_ctf_resource_quotas_competition ON ctf_resource_quotas(competition_id);

-- challenge_environments：题目环境实例表。
CREATE TABLE challenge_environments (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    challenge_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    namespace VARCHAR(100) NOT NULL,
    chain_rpc_url VARCHAR(500) NULL,
    container_status JSONB NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    started_at TIMESTAMP NULL,
    destroyed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_challenge_environments_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_challenge_environments_challenge_id FOREIGN KEY (challenge_id) REFERENCES challenges(id),
    CONSTRAINT fk_challenge_environments_team_id FOREIGN KEY (team_id) REFERENCES teams(id)
);
CREATE INDEX idx_challenge_environments_competition_id ON challenge_environments(competition_id);
CREATE INDEX idx_challenge_environments_challenge_id ON challenge_environments(challenge_id);
CREATE INDEX idx_challenge_environments_team_id ON challenge_environments(team_id);
CREATE INDEX idx_challenge_environments_status ON challenge_environments(status);
CREATE UNIQUE INDEX uk_challenge_environments ON challenge_environments(competition_id, challenge_id, team_id) WHERE status != 5;

-- ad_team_chains：攻防赛队伍链表。
CREATE TABLE ad_team_chains (
    id BIGINT PRIMARY KEY,
    competition_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    chain_rpc_url VARCHAR(500) NULL,
    chain_ws_url VARCHAR(500) NULL,
    deployed_contracts JSONB NULL,
    current_patch_version INT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ad_team_chains_competition_id FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT fk_ad_team_chains_group_id FOREIGN KEY (group_id) REFERENCES ad_groups(id),
    CONSTRAINT fk_ad_team_chains_team_id FOREIGN KEY (team_id) REFERENCES teams(id)
);
CREATE UNIQUE INDEX uk_ad_team_chains ON ad_team_chains(competition_id, team_id);
CREATE INDEX idx_ad_team_chains_group_id ON ad_team_chains(group_id);
CREATE INDEX idx_ad_team_chains_status ON ad_team_chains(status);
