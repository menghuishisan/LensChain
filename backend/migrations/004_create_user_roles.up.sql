-- 004_create_user_roles.up.sql
-- 模块01 — 用户与认证：创建 user_roles 用户角色关联表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS user_roles (
    id         BIGINT    PRIMARY KEY,                                  -- 雪花算法ID
    user_id    BIGINT    NOT NULL,                                     -- 用户ID
    role_id    BIGINT    NOT NULL,                                     -- 角色ID
    created_at TIMESTAMP NOT NULL DEFAULT NOW()                        -- 创建时间
);

-- 索引（2个）
CREATE UNIQUE INDEX uk_user_roles_user_role ON user_roles(user_id, role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

COMMENT ON TABLE user_roles IS '用户角色关联表';
