-- 003_create_roles.up.sql
-- 模块01 — 用户与认证：创建 roles 角色表 + 种子数据
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS roles (
    id          BIGINT       PRIMARY KEY,                              -- 雪花算法ID
    code        VARCHAR(50)  NOT NULL UNIQUE,                          -- 角色编码（如 super_admin）
    name        VARCHAR(50)  NOT NULL,                                 -- 角色显示名称
    description VARCHAR(200) NULL,                                     -- 角色描述
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,                   -- 是否系统预设角色（不可删除）
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),                   -- 创建时间
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()                    -- 更新时间
);

-- 预设角色种子数据（4条）
INSERT INTO roles (id, code, name, description, is_system) VALUES
    (1, 'super_admin',  '超级管理员', '平台最高权限，管理所有学校和系统配置', TRUE),
    (2, 'school_admin', '学校管理员', '管理本校用户、课程、实验等资源',       TRUE),
    (3, 'teacher',      '教师',       '创建课程、布置作业、管理实验',         TRUE),
    (4, 'student',      '学生',       '学习课程、完成实验、参加竞赛',         TRUE);

COMMENT ON TABLE roles IS '角色表';
COMMENT ON COLUMN roles.is_system IS '系统预设角色不可删除';
