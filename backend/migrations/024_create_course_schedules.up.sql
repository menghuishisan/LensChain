-- 024_create_course_schedules.up.sql
-- 模块03 — 课程与教学：课程表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_schedules (
    id          BIGINT       PRIMARY KEY,
    course_id   BIGINT       NOT NULL,
    day_of_week SMALLINT     NOT NULL,
    start_time  TIME         NOT NULL,
    end_time    TIME         NOT NULL,
    location    VARCHAR(100),
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_course_schedules_course_id ON course_schedules(course_id);

COMMENT ON TABLE course_schedules IS '课程表';
COMMENT ON COLUMN course_schedules.day_of_week IS '星期几：1-7（周一到周日）';
