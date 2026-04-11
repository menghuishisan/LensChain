-- 020_create_assignment_questions.up.sql
-- 模块03 — 课程与教学：题目表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE assignment_questions (
    id               BIGINT       PRIMARY KEY,
    assignment_id    BIGINT       NOT NULL,
    question_type    SMALLINT     NOT NULL,
    title            TEXT         NOT NULL,
    options          JSONB,
    correct_answer   TEXT,
    reference_answer TEXT,
    score            DECIMAL(6,2) NOT NULL,
    judge_config     JSONB,
    sort_order       INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assignment_questions_assignment_id ON assignment_questions(assignment_id);

COMMENT ON TABLE assignment_questions IS '题目表';
COMMENT ON COLUMN assignment_questions.question_type IS '题型：1单选 2多选 3判断 4填空 5简答 6编程 7实验报告';
COMMENT ON COLUMN assignment_questions.options IS '选项（客观题）：[{"key":"A","text":"选项A"},...]';
