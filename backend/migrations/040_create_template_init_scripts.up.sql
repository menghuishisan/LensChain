-- 040_create_template_init_scripts.up.sql
-- 模块04 — 实验环境：初始化脚本表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.7节

CREATE TABLE template_init_scripts (
    id               BIGINT       PRIMARY KEY,
    template_id      BIGINT       NOT NULL REFERENCES experiment_templates(id),
    target_container VARCHAR(100) NOT NULL,
    script_content   TEXT         NOT NULL,
    script_language  VARCHAR(20)  NOT NULL DEFAULT 'bash',
    execution_order  INT          NOT NULL DEFAULT 0,
    timeout          INT          NOT NULL DEFAULT 300,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_template_init_scripts_template_id ON template_init_scripts(template_id);
