-- 043_create_sim_link_group_scenes.up.sql
-- 模块04 — 实验环境：联动组场景关联表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.10节

CREATE TABLE sim_link_group_scenes (
    id             BIGINT      PRIMARY KEY,
    link_group_id  BIGINT      NOT NULL REFERENCES sim_link_groups(id),
    scenario_id    BIGINT      NOT NULL REFERENCES sim_scenarios(id),
    role_in_group  VARCHAR(50) NULL,
    sort_order     INT         NOT NULL DEFAULT 0,
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_link_group_scenes ON sim_link_group_scenes(link_group_id, scenario_id);
CREATE INDEX idx_link_group_scenes_scenario_id ON sim_link_group_scenes(scenario_id);
