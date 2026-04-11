-- 016_create_lessons.up.sql
-- 模块03 — 课程与教学：课时表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE lessons (
    id                BIGINT       PRIMARY KEY,
    chapter_id        BIGINT       NOT NULL,
    course_id         BIGINT       NOT NULL,
    title             VARCHAR(200) NOT NULL,
    content_type      SMALLINT     NOT NULL,
    content           TEXT,
    video_url         VARCHAR(500),
    video_duration    INT,
    experiment_id     BIGINT,
    sort_order        INT          NOT NULL DEFAULT 0,
    estimated_minutes INT,
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMP
);

CREATE INDEX idx_lessons_chapter_id ON lessons(chapter_id);
CREATE INDEX idx_lessons_course_id ON lessons(course_id);
CREATE INDEX idx_lessons_sort_order ON lessons(chapter_id, sort_order);

COMMENT ON TABLE lessons IS '课时表';
COMMENT ON COLUMN lessons.content_type IS '内容类型：1视频 2图文 3附件 4实验';
