-- 047_create_template_roles.up.sql
-- 模块04 — 实验环境：多人实验角色定义表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.14节

CREATE TABLE template_roles (
    id          BIGINT       PRIMARY KEY,
    template_id BIGINT       NOT NULL REFERENCES experiment_templates(id),
    role_name   VARCHAR(50)  NOT NULL,
    description VARCHAR(200) NULL,
    max_members INT          NOT NULL DEFAULT 1,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_template_roles_template_id ON template_roles(template_id);
