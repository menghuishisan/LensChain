-- 001_create_users.up.sql
-- 模块01 — 用户与认证：创建 users 用户主表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS users (
    id              BIGINT       PRIMARY KEY,                          -- 雪花算法ID
    phone           VARCHAR(20)  NOT NULL,                             -- 手机号（全局唯一标识）
    password_hash   VARCHAR(255) NOT NULL,                             -- 密码哈希（bcrypt）
    name            VARCHAR(50)  NOT NULL,                             -- 真实姓名
    school_id       BIGINT       NOT NULL,                             -- 所属学校ID
    student_no      VARCHAR(50)  NULL,                                 -- 学号/工号（校内唯一）
    status          SMALLINT     NOT NULL DEFAULT 1,                   -- 账号状态：1正常 2禁用 3归档
    is_first_login  BOOLEAN      NOT NULL DEFAULT TRUE,                -- 是否首次登录（需强制改密）
    is_school_admin BOOLEAN      NOT NULL DEFAULT FALSE,               -- 是否兼任学校管理员
    login_fail_count SMALLINT    NOT NULL DEFAULT 0,                   -- 连续登录失败次数
    locked_until    TIMESTAMP    NULL,                                 -- 锁定截止时间，NULL表示未锁定
    last_login_at   TIMESTAMP    NULL,                                 -- 最后登录时间
    last_login_ip   VARCHAR(45)  NULL,                                 -- 最后登录IP
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),               -- 创建时间
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),               -- 更新时间
    created_by      BIGINT       NULL,                                 -- 创建人ID
    deleted_at      TIMESTAMP    NULL                                  -- 软删除时间
);

-- 索引（5个）
CREATE UNIQUE INDEX uk_users_phone ON users(phone) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_users_school_student_no ON users(school_id, student_no) WHERE deleted_at IS NULL AND student_no IS NOT NULL;
CREATE INDEX idx_users_school_id ON users(school_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_name ON users(name);

COMMENT ON TABLE users IS '用户主表';
COMMENT ON COLUMN users.status IS '账号状态：1正常 2禁用 3归档';
COMMENT ON COLUMN users.is_first_login IS '是否首次登录（需强制改密）';
COMMENT ON COLUMN users.is_school_admin IS '是否兼任学校管理员';
