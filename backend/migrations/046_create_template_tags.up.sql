-- 046_create_template_tags.up.sql
-- 模块04 — 实验环境：模板标签关联表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.13节

CREATE TABLE template_tags (
    id          BIGINT    PRIMARY KEY,
    template_id BIGINT    NOT NULL REFERENCES experiment_templates(id),
    tag_id      BIGINT    NOT NULL REFERENCES tags(id),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_template_tags ON template_tags(template_id, tag_id);
CREATE INDEX idx_template_tags_tag_id ON template_tags(tag_id);
