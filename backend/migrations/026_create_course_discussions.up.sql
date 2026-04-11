-- 026_create_course_discussions.up.sql
-- 模块03 — 课程与教学：讨论帖表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_discussions (
    id              BIGINT       PRIMARY KEY,
    course_id       BIGINT       NOT NULL,
    author_id       BIGINT       NOT NULL,
    title           VARCHAR(200) NOT NULL,
    content         TEXT         NOT NULL,
    is_pinned       BOOLEAN      NOT NULL DEFAULT FALSE,
    reply_count     INT          NOT NULL DEFAULT 0,
    like_count      INT          NOT NULL DEFAULT 0,
    last_replied_at TIMESTAMP,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_course_discussions_course_id ON course_discussions(course_id);
CREATE INDEX idx_course_discussions_author_id ON course_discussions(author_id);
CREATE INDEX idx_course_discussions_is_pinned ON course_discussions(course_id, is_pinned);

COMMENT ON TABLE course_discussions IS '讨论帖表';
