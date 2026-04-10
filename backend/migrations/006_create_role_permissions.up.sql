-- 006_create_role_permissions.up.sql
-- 模块01 — 用户与认证：创建 role_permissions 角色权限关联表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS role_permissions (
    id            BIGINT    PRIMARY KEY,                               -- 雪花算法ID
    role_id       BIGINT    NOT NULL,                                  -- 角色ID
    permission_id BIGINT    NOT NULL,                                  -- 权限ID
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()                     -- 创建时间
);

-- 索引（2个）
CREATE UNIQUE INDEX uk_role_permissions ON role_permissions(role_id, permission_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

COMMENT ON TABLE role_permissions IS '角色权限关联表';
