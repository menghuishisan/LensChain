-- 032_alter_courses_add_grade_overrides.up.sql
-- 模块03 — 课程与教学：补充课程学分/学期字段与成绩调整记录表
-- 新增 courses.credits、courses.semester_id，并创建 course_grade_overrides 持久化教师手动调分结果

ALTER TABLE courses
ADD COLUMN IF NOT EXISTS credits DECIMAL(3,1) NULL,
ADD COLUMN IF NOT EXISTS semester_id BIGINT NULL;

CREATE INDEX IF NOT EXISTS idx_courses_semester_id ON courses(semester_id);

CREATE TABLE IF NOT EXISTS course_grade_overrides (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    weighted_total DECIMAL(6,2) NOT NULL,
    final_score DECIMAL(6,2) NOT NULL,
    adjust_reason VARCHAR(200) NOT NULL,
    adjusted_by BIGINT NOT NULL,
    adjusted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_course_grade_overrides_course_student
    ON course_grade_overrides(course_id, student_id);

CREATE INDEX IF NOT EXISTS idx_course_grade_overrides_course_id
    ON course_grade_overrides(course_id);

CREATE INDEX IF NOT EXISTS idx_course_grade_overrides_student_id
    ON course_grade_overrides(student_id);
