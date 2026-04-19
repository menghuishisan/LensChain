-- 模块06 评测与成绩回滚

ALTER TABLE IF EXISTS grade_appeals DROP CONSTRAINT IF EXISTS fk_grade_appeals_course_id;
ALTER TABLE IF EXISTS student_semester_grades DROP CONSTRAINT IF EXISTS fk_student_semester_grades_course_id;
ALTER TABLE IF EXISTS grade_reviews DROP CONSTRAINT IF EXISTS fk_grade_reviews_course_id;
ALTER TABLE IF EXISTS courses DROP CONSTRAINT IF EXISTS fk_courses_semester_id;

DROP TABLE IF EXISTS transcript_records;
DROP TABLE IF EXISTS warning_configs;
DROP TABLE IF EXISTS academic_warnings;
DROP TABLE IF EXISTS grade_appeals;
DROP TABLE IF EXISTS student_semester_grades;
DROP TABLE IF EXISTS grade_reviews;
DROP TABLE IF EXISTS grade_level_configs;
DROP TABLE IF EXISTS semesters;
