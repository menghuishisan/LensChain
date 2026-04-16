-- 044_create_template_sim_scenes.up.sql
-- 模块04 — 实验环境：模板仿真场景配置表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.11节

CREATE TABLE template_sim_scenes (
    id                 BIGINT    PRIMARY KEY,
    template_id        BIGINT    NOT NULL REFERENCES experiment_templates(id),
    scenario_id        BIGINT    NOT NULL REFERENCES sim_scenarios(id),
    link_group_id      BIGINT    NULL REFERENCES sim_link_groups(id),
    config             JSONB     NULL,
    layout_position    JSONB     NULL,
    data_source_config JSONB     NULL,
    sort_order         INT       NOT NULL DEFAULT 0,
    created_at         TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_template_sim_scenes_template_id ON template_sim_scenes(template_id);
CREATE INDEX idx_template_sim_scenes_scenario_id ON template_sim_scenes(scenario_id);
