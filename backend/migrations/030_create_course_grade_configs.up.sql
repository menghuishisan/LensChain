-- 030_create_course_grade_configs.up.sql
-- 模块03 — 课程与教学：成绩权重配置表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_grade_configs (
    id         BIGINT    PRIMARY KEY,
    course_id  BIGINT    NOT NULL UNIQUE,
    config     JSONB     NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course_grade_configs IS '成绩权重配置表';
COMMENT ON COLUMN course_grade_configs.config IS '权重配置JSON：{"items":[{"assignment_id":"...","name":"...","weight":20}],"total_weight":100}';
