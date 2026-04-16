-- 045_create_tags.up.sql
-- 模块04 — 实验环境：标签表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.12节

CREATE TABLE tags (
    id         BIGINT      PRIMARY KEY,
    name       VARCHAR(50) NOT NULL,
    category   VARCHAR(20) NOT NULL,
    is_system  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_tags_name_category ON tags(name, category);
