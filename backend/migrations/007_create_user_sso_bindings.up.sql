-- 007_create_user_sso_bindings.up.sql
-- 模块01 — 用户与认证：创建 user_sso_bindings SSO绑定记录表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS user_sso_bindings (
    id           BIGINT      PRIMARY KEY,                              -- 雪花算法ID
    user_id      BIGINT      NOT NULL,                                 -- 用户ID
    school_id    BIGINT      NOT NULL,                                 -- 学校ID
    sso_provider VARCHAR(20) NOT NULL,                                 -- SSO协议类型：cas / oauth2
    sso_user_id  VARCHAR(100) NOT NULL,                                -- SSO系统中的用户标识（学号/工号）
    bound_at     TIMESTAMP   NOT NULL DEFAULT NOW(),                   -- 绑定时间
    last_login_at TIMESTAMP  NULL                                      -- 最后一次SSO登录时间
);

-- 索引（2个）
CREATE UNIQUE INDEX uk_sso_bindings_school_sso_user ON user_sso_bindings(school_id, sso_user_id);
CREATE INDEX idx_sso_bindings_user_id ON user_sso_bindings(user_id);

COMMENT ON TABLE user_sso_bindings IS 'SSO绑定记录表';
COMMENT ON COLUMN user_sso_bindings.sso_provider IS 'SSO协议类型：cas / oauth2';
