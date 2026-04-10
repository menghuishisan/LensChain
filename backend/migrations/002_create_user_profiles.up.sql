-- 002_create_user_profiles.up.sql
-- 模块01 — 用户与认证：创建 user_profiles 用户扩展信息表
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS user_profiles (
    id              BIGINT       PRIMARY KEY,                          -- 雪花算法ID
    user_id         BIGINT       NOT NULL,                             -- 关联用户ID
    avatar_url      VARCHAR(500) NULL,                                 -- 头像URL
    nickname        VARCHAR(50)  NULL,                                 -- 昵称
    email           VARCHAR(100) NULL,                                 -- 邮箱
    college         VARCHAR(100) NULL,                                 -- 学院
    major           VARCHAR(100) NULL,                                 -- 专业
    class_name      VARCHAR(50)  NULL,                                 -- 班级
    enrollment_year SMALLINT     NULL,                                 -- 入学年份
    education_level SMALLINT     NULL,                                 -- 学业层次：1本科 2硕士 3博士
    grade           SMALLINT     NULL,                                 -- 年级
    remark          TEXT         NULL,                                 -- 备注
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),               -- 创建时间
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW()                -- 更新时间
);

-- 索引（3个）
CREATE UNIQUE INDEX uk_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX idx_user_profiles_college ON user_profiles(college);
CREATE INDEX idx_user_profiles_education_level ON user_profiles(education_level);

COMMENT ON TABLE user_profiles IS '用户扩展信息表';
COMMENT ON COLUMN user_profiles.education_level IS '学业层次：1本科 2硕士 3博士';
