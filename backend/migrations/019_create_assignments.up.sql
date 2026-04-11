-- 019_create_assignments.up.sql
-- 模块03 — 课程与教学：作业/测验表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE assignments (
    id                      BIGINT        PRIMARY KEY,
    course_id               BIGINT        NOT NULL,
    chapter_id              BIGINT,
    title                   VARCHAR(200)  NOT NULL,
    description             TEXT,
    assignment_type         SMALLINT      NOT NULL,
    total_score             DECIMAL(6,2)  NOT NULL,
    deadline_at             TIMESTAMP,
    max_submissions         INT           NOT NULL DEFAULT 1,
    late_policy             SMALLINT      NOT NULL DEFAULT 1,
    late_deduction_per_day  DECIMAL(5,2),
    is_published            BOOLEAN       NOT NULL DEFAULT FALSE,
    sort_order              INT           NOT NULL DEFAULT 0,
    created_at              TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMP
);

CREATE INDEX idx_assignments_course_id ON assignments(course_id);
CREATE INDEX idx_assignments_chapter_id ON assignments(chapter_id);
CREATE INDEX idx_assignments_deadline_at ON assignments(deadline_at);

COMMENT ON TABLE assignments IS '作业/测验表';
COMMENT ON COLUMN assignments.assignment_type IS '类型：1作业 2测验';
COMMENT ON COLUMN assignments.late_policy IS '迟交策略：1不允许 2允许扣分 3允许不扣分';
