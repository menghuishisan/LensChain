-- 017_create_lesson_attachments.up.sql
-- 模块03 — 课程与教学：课时附件表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE lesson_attachments (
    id         BIGINT       PRIMARY KEY,
    lesson_id  BIGINT       NOT NULL,
    file_name  VARCHAR(200) NOT NULL,
    file_url   VARCHAR(500) NOT NULL,
    file_size  BIGINT       NOT NULL,
    file_type  VARCHAR(50)  NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lesson_attachments_lesson_id ON lesson_attachments(lesson_id);

COMMENT ON TABLE lesson_attachments IS '课时附件表';
