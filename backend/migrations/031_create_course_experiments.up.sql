-- 031_create_course_experiments.up.sql
-- 模块03 — 课程与教学：课程独立实验关联表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_experiments (
    id            BIGINT       PRIMARY KEY,
    course_id     BIGINT       NOT NULL,
    experiment_id BIGINT       NOT NULL,
    title         VARCHAR(200),
    sort_order    INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_experiments_course_id ON course_experiments(course_id);
CREATE UNIQUE INDEX uk_course_experiments ON course_experiments(course_id, experiment_id);

COMMENT ON TABLE course_experiments IS '课程独立实验关联表';
COMMENT ON COLUMN course_experiments.title IS '在课程中的显示标题（可覆盖实验原标题）';
