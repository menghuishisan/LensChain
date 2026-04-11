-- 023_create_learning_progresses.up.sql
-- 模块03 — 课程与教学：学习进度表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE learning_progresses (
    id               BIGINT    PRIMARY KEY,
    course_id        BIGINT    NOT NULL,
    student_id       BIGINT    NOT NULL,
    lesson_id        BIGINT    NOT NULL,
    status           SMALLINT  NOT NULL DEFAULT 1,
    video_progress   INT,
    study_duration   INT       NOT NULL DEFAULT 0,
    completed_at     TIMESTAMP,
    last_accessed_at TIMESTAMP,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_learning_progress ON learning_progresses(course_id, student_id, lesson_id);
CREATE INDEX idx_learning_progresses_student_id ON learning_progresses(student_id);
CREATE INDEX idx_learning_progresses_status ON learning_progresses(status);

COMMENT ON TABLE learning_progresses IS '学习进度表';
COMMENT ON COLUMN learning_progresses.status IS '状态：1未开始 2进行中 3已完成';
