-- 025_create_course_announcements.up.sql
-- 模块03 — 课程与教学：课程公告表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_announcements (
    id         BIGINT       PRIMARY KEY,
    course_id  BIGINT       NOT NULL,
    teacher_id BIGINT       NOT NULL,
    title      VARCHAR(200) NOT NULL,
    content    TEXT         NOT NULL,
    is_pinned  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_course_announcements_course_id ON course_announcements(course_id);

COMMENT ON TABLE course_announcements IS '课程公告表';
