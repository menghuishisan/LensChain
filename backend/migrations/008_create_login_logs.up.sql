-- 008_create_login_logs.up.sql
-- 模块01 — 用户与认证：创建 login_logs 登录日志表
-- 审计日志红线：只插入，不更新，不删除
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS login_logs (
    id           BIGINT       PRIMARY KEY,                             -- 雪花算法ID
    user_id      BIGINT       NOT NULL,                                -- 用户ID（不设FK，避免删除用户后日志丢失）
    action       SMALLINT     NOT NULL,                                -- 操作类型：1登录成功 2登录失败 3登出 4被踢下线 5账号锁定
    login_method SMALLINT     NULL,                                    -- 登录方式：1密码 2SSO-CAS 3SSO-OAuth2
    ip           VARCHAR(45)  NOT NULL,                                -- 客户端IP
    user_agent   VARCHAR(500) NULL,                                    -- 浏览器UA
    fail_reason  VARCHAR(200) NULL,                                    -- 失败原因（仅失败时记录）
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW()                   -- 记录时间
);

-- 索引（3个）
CREATE INDEX idx_login_logs_user_id ON login_logs(user_id);
CREATE INDEX idx_login_logs_created_at ON login_logs(created_at);
CREATE INDEX idx_login_logs_action ON login_logs(action);

COMMENT ON TABLE login_logs IS '登录日志表（审计日志，只插入不更新不删除）';
COMMENT ON COLUMN login_logs.action IS '操作类型：1登录成功 2登录失败 3登出 4被踢下线 5账号锁定';
COMMENT ON COLUMN login_logs.login_method IS '登录方式：1密码 2SSO-CAS 3SSO-OAuth2';
