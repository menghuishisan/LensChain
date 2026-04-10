-- 005_create_permissions.up.sql
-- 模块01 — 用户与认证：创建 permissions 权限表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS permissions (
    id          BIGINT       PRIMARY KEY,                              -- 雪花算法ID
    code        VARCHAR(100) NOT NULL UNIQUE,                          -- 权限编码（如 user:import）
    name        VARCHAR(100) NOT NULL,                                 -- 权限显示名称
    module      VARCHAR(50)  NOT NULL,                                 -- 所属模块（如 user, course）
    description VARCHAR(200) NULL,                                     -- 权限描述
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()                    -- 创建时间
);

-- 索引（1个）
CREATE INDEX idx_permissions_module ON permissions(module);

COMMENT ON TABLE permissions IS '权限表';
