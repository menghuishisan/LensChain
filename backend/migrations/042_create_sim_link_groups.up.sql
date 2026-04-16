-- 042_create_sim_link_groups.up.sql
-- 模块04 — 实验环境：联动组定义表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.9节

CREATE TABLE sim_link_groups (
    id                  BIGINT       PRIMARY KEY,
    name                VARCHAR(100) NOT NULL,
    code                VARCHAR(100) NOT NULL,
    description         TEXT         NULL,
    shared_state_schema JSONB        NOT NULL,
    created_at          TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_sim_link_groups_code ON sim_link_groups(code);
