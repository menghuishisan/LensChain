-- 022_create_submission_answers.up.sql
-- 模块03 — 课程与教学：答题记录表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE submission_answers (
    id                BIGINT       PRIMARY KEY,
    submission_id     BIGINT       NOT NULL,
    question_id       BIGINT       NOT NULL,
    answer_content    TEXT,
    answer_file_url   VARCHAR(500),
    is_correct        BOOLEAN,
    score             DECIMAL(6,2),
    teacher_comment   TEXT,
    auto_judge_result JSONB,
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submission_answers_submission_id ON submission_answers(submission_id);
CREATE INDEX idx_submission_answers_question_id ON submission_answers(question_id);

COMMENT ON TABLE submission_answers IS '答题记录表';
