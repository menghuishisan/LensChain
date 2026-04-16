-- 034_create_image_categories.up.sql
-- 模块04 — 实验环境：镜像分类表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.1节

CREATE TABLE image_categories (
    id          BIGINT       PRIMARY KEY,
    name        VARCHAR(50)  NOT NULL,
    code        VARCHAR(50)  NOT NULL UNIQUE,
    description VARCHAR(200) NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- 预设分类数据
INSERT INTO image_categories (id, name, code, description, sort_order) VALUES
(1, '链节点镜像', 'chain_node', '区块链节点运行环境镜像', 1),
(2, '中间件镜像', 'middleware', '区块链中间件和工具链镜像', 2),
(3, '工具镜像', 'tool', '开发工具和辅助工具镜像', 3),
(4, '环境基础镜像', 'base_env', '基础操作系统和运行环境镜像', 4);
