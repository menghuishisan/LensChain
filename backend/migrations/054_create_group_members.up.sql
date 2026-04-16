-- 054_create_group_members.up.sql
-- 模块04 — 实验环境：分组成员表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.21节

CREATE TABLE group_members (
    id          BIGINT    PRIMARY KEY,
    group_id    BIGINT    NOT NULL REFERENCES experiment_groups(id),
    student_id  BIGINT    NOT NULL,
    role_id     BIGINT    NULL REFERENCES template_roles(id),
    instance_id BIGINT    NULL,
    joined_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_group_members ON group_members(group_id, student_id);
CREATE INDEX idx_group_members_student_id ON group_members(student_id);
