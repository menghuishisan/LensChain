-- 039_create_template_checkpoints.up.sql
-- 模块04 — 实验环境：检查点定义表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.6节

CREATE TABLE template_checkpoints (
    id               BIGINT       PRIMARY KEY,
    template_id      BIGINT       NOT NULL REFERENCES experiment_templates(id),
    title            VARCHAR(200) NOT NULL,
    description      TEXT         NULL,
    check_type       SMALLINT     NOT NULL,
    script_content   TEXT         NULL,
    script_language  VARCHAR(20)  NULL,
    target_container VARCHAR(100) NULL,
    assertion_config JSONB        NULL,
    score            DECIMAL(6,2) NOT NULL,
    scope            SMALLINT     NOT NULL DEFAULT 1,
    sort_order       INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_template_checkpoints_template_id ON template_checkpoints(template_id);
