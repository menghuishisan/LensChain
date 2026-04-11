-- 029_create_course_evaluations.up.sql
-- 模块03 — 课程与教学：课程评价表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_evaluations (
    id         BIGINT    PRIMARY KEY,
    course_id  BIGINT    NOT NULL,
    student_id BIGINT    NOT NULL,
    rating     SMALLINT  NOT NULL,
    comment    TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_course_evaluations ON course_evaluations(course_id, student_id);
CREATE INDEX idx_course_evaluations_course_id ON course_evaluations(course_id);

COMMENT ON TABLE course_evaluations IS '课程评价表';
COMMENT ON COLUMN course_evaluations.rating IS '评分：1-5星';
