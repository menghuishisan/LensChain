-- 041_create_sim_scenarios.up.sql
-- 模块04 — 实验环境：仿真场景库表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.8节

CREATE TABLE sim_scenarios (
    id                   BIGINT       PRIMARY KEY,
    name                 VARCHAR(100) NOT NULL,
    code                 VARCHAR(100) NOT NULL,
    category             VARCHAR(50)  NOT NULL,
    description          TEXT         NULL,
    icon_url             VARCHAR(500) NULL,
    thumbnail_url        VARCHAR(500) NULL,
    source_type          SMALLINT     NOT NULL DEFAULT 1,
    uploaded_by          BIGINT       NULL,
    school_id            BIGINT       NULL,
    status               SMALLINT     NOT NULL DEFAULT 1,
    review_comment       VARCHAR(500) NULL,
    reviewed_by          BIGINT       NULL,
    reviewed_at          TIMESTAMP    NULL,
    algorithm_type       VARCHAR(100) NOT NULL,
    time_control_mode    VARCHAR(20)  NOT NULL DEFAULT 'process',
    container_image_url  VARCHAR(500) NULL,
    container_image_size BIGINT       NULL,
    default_params       JSONB        NULL,
    interaction_schema   JSONB        NULL,
    data_source_mode     SMALLINT     NOT NULL DEFAULT 1,
    default_size         JSONB        NULL,
    delivery_phase       SMALLINT     NOT NULL DEFAULT 1,
    version              VARCHAR(50)  NOT NULL DEFAULT '1.0.0',
    created_at           TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMP    NULL
);

CREATE UNIQUE INDEX uk_sim_scenarios_code ON sim_scenarios(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_sim_scenarios_category ON sim_scenarios(category);
CREATE INDEX idx_sim_scenarios_source_type ON sim_scenarios(source_type);
CREATE INDEX idx_sim_scenarios_status ON sim_scenarios(status);
CREATE INDEX idx_sim_scenarios_algorithm_type ON sim_scenarios(algorithm_type);
CREATE INDEX idx_sim_scenarios_delivery_phase ON sim_scenarios(delivery_phase);
CREATE INDEX idx_sim_scenarios_time_control_mode ON sim_scenarios(time_control_mode);
