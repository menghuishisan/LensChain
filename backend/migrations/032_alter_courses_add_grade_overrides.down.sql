-- 032_alter_courses_add_grade_overrides.down.sql
-- 模块03 — 课程与教学：回滚课程学分/学期字段与成绩调整记录表
-- 删除 course_grade_overrides 表，并移除 courses.credits、courses.semester_id

DROP INDEX IF EXISTS idx_course_grade_overrides_student_id;
DROP INDEX IF EXISTS idx_course_grade_overrides_course_id;
DROP INDEX IF EXISTS uk_course_grade_overrides_course_student;

DROP TABLE IF EXISTS course_grade_overrides;

DROP INDEX IF EXISTS idx_courses_semester_id;

ALTER TABLE courses
DROP COLUMN IF EXISTS semester_id,
DROP COLUMN IF EXISTS credits;
