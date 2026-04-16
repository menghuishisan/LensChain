-- 037_create_experiment_templates.up.sql
-- 模块04 — 实验环境：实验模板主表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.4节

CREATE TABLE experiment_templates (
    id              BIGINT        PRIMARY KEY,
    school_id       BIGINT        NOT NULL,
    teacher_id      BIGINT        NOT NULL,
    title           VARCHAR(200)  NOT NULL,
    description     TEXT          NULL,
    objectives      TEXT          NULL,
    instructions    TEXT          NULL,
    references      TEXT          NULL,
    experiment_type SMALLINT      NOT NULL DEFAULT 2,
    topology_mode   SMALLINT      NULL,
    judge_mode      SMALLINT      NOT NULL DEFAULT 1,
    auto_weight     DECIMAL(5,2)  NULL,
    manual_weight   DECIMAL(5,2)  NULL,
    total_score     DECIMAL(6,2)  NOT NULL DEFAULT 100,
    max_duration    INT           NULL,
    idle_timeout    INT           NOT NULL DEFAULT 30,
    cpu_limit       VARCHAR(20)   NULL,
    memory_limit    VARCHAR(20)   NULL,
    disk_limit      VARCHAR(20)   NULL,
    score_strategy  SMALLINT      NOT NULL DEFAULT 1,
    is_shared       BOOLEAN       NOT NULL DEFAULT FALSE,
    cloned_from_id  BIGINT        NULL,
    status          SMALLINT      NOT NULL DEFAULT 1,
    sim_layout      JSONB         NULL,
    k8s_config      JSONB         NULL,
    network_config  JSONB         NULL,
    created_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP     NULL
);

CREATE INDEX idx_experiment_templates_school_id ON experiment_templates(school_id);
CREATE INDEX idx_experiment_templates_teacher_id ON experiment_templates(teacher_id);
CREATE INDEX idx_experiment_templates_experiment_type ON experiment_templates(experiment_type);
CREATE INDEX idx_experiment_templates_topology_mode ON experiment_templates(topology_mode);
CREATE INDEX idx_experiment_templates_status ON experiment_templates(status);
CREATE INDEX idx_experiment_templates_is_shared ON experiment_templates(is_shared) WHERE is_shared = TRUE;
