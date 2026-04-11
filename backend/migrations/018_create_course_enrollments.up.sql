-- 018_create_course_enrollments.up.sql
-- 模块03 — 课程与教学：选课记录表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE course_enrollments (
    id          BIGINT   PRIMARY KEY,
    course_id   BIGINT   NOT NULL,
    student_id  BIGINT   NOT NULL,
    join_method SMALLINT NOT NULL,
    joined_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    removed_at  TIMESTAMP
);

CREATE UNIQUE INDEX uk_enrollments_course_student ON course_enrollments(course_id, student_id) WHERE removed_at IS NULL;
CREATE INDEX idx_enrollments_student_id ON course_enrollments(student_id);

COMMENT ON TABLE course_enrollments IS '选课记录表';
COMMENT ON COLUMN course_enrollments.join_method IS '加入方式：1教师指定 2邀请码';
