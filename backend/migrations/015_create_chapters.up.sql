-- 015_create_chapters.up.sql
-- 模块03 — 课程与教学：章节表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE chapters (
    id          BIGINT       PRIMARY KEY,
    course_id   BIGINT       NOT NULL,
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE INDEX idx_chapters_course_id ON chapters(course_id);
CREATE INDEX idx_chapters_sort_order ON chapters(course_id, sort_order);

COMMENT ON TABLE chapters IS '章节表';
