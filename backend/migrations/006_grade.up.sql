-- 模块06 评测与成绩
-- 文档依据：
-- 1. docs/modules/06-评测与成绩/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 学期、等级映射、成绩审核
-- 2. 学期成绩汇总、申诉、预警、成绩单记录
-- 不负责：
-- 1. 课程成绩的原始采集，原始分项数据仍来自模块03
-- 2. 通知发送，通知由模块07处理

-- semesters：学期表。
CREATE TABLE semesters (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    name VARCHAR(50) NOT NULL,
    code VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_semesters_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE INDEX idx_semesters_school_id ON semesters(school_id);
CREATE UNIQUE INDEX uk_semesters_school_code ON semesters(school_id, code) WHERE deleted_at IS NULL;
CREATE INDEX idx_semesters_is_current ON semesters(school_id, is_current);

-- grade_level_configs：等级映射配置表。
CREATE TABLE grade_level_configs (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    level_name VARCHAR(10) NOT NULL,
    min_score DECIMAL(5,2) NOT NULL,
    max_score DECIMAL(5,2) NOT NULL,
    gpa_point DECIMAL(3,2) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_grade_level_configs_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE INDEX idx_grade_level_configs_school_id ON grade_level_configs(school_id);
CREATE UNIQUE INDEX uk_grade_level_configs_school_level ON grade_level_configs(school_id, level_name);

-- grade_reviews：成绩审核表。
CREATE TABLE grade_reviews (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    semester_id BIGINT NOT NULL,
    submitted_by BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    submit_note TEXT NULL,
    submitted_at TIMESTAMP NULL,
    reviewed_by BIGINT NULL,
    reviewed_at TIMESTAMP NULL,
    review_comment TEXT NULL,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    locked_at TIMESTAMP NULL,
    unlocked_by BIGINT NULL,
    unlocked_at TIMESTAMP NULL,
    unlock_reason TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_grade_reviews_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_grade_reviews_semester_id FOREIGN KEY (semester_id) REFERENCES semesters(id),
    CONSTRAINT fk_grade_reviews_submitted_by FOREIGN KEY (submitted_by) REFERENCES users(id),
    CONSTRAINT fk_grade_reviews_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id),
    CONSTRAINT fk_grade_reviews_unlocked_by FOREIGN KEY (unlocked_by) REFERENCES users(id)
);
CREATE INDEX idx_grade_reviews_school_id ON grade_reviews(school_id);
CREATE INDEX idx_grade_reviews_semester_id ON grade_reviews(semester_id);
CREATE INDEX idx_grade_reviews_status ON grade_reviews(status);
CREATE UNIQUE INDEX uk_grade_reviews_course_semester ON grade_reviews(course_id, semester_id);
CREATE INDEX idx_grade_reviews_submitted_by ON grade_reviews(submitted_by);

-- student_semester_grades：学生学期成绩汇总表。
CREATE TABLE student_semester_grades (
    id BIGINT PRIMARY KEY,
    student_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    semester_id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    final_score DECIMAL(6,2) NOT NULL,
    grade_level VARCHAR(10) NOT NULL,
    gpa_point DECIMAL(3,2) NOT NULL,
    credits DECIMAL(3,1) NOT NULL,
    is_adjusted BOOLEAN NOT NULL DEFAULT FALSE,
    review_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_student_semester_grades_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_student_semester_grades_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_student_semester_grades_semester_id FOREIGN KEY (semester_id) REFERENCES semesters(id),
    CONSTRAINT fk_student_semester_grades_review_id FOREIGN KEY (review_id) REFERENCES grade_reviews(id)
);
CREATE UNIQUE INDEX uk_student_semester_grades_unique ON student_semester_grades(student_id, semester_id, course_id);
CREATE INDEX idx_ssg_school_id ON student_semester_grades(school_id);
CREATE INDEX idx_ssg_semester_id ON student_semester_grades(semester_id);
CREATE INDEX idx_ssg_student_id ON student_semester_grades(student_id);
CREATE INDEX idx_ssg_course_id ON student_semester_grades(course_id);
CREATE INDEX idx_ssg_grade_level ON student_semester_grades(grade_level);

-- grade_appeals：成绩申诉表。
CREATE TABLE grade_appeals (
    id BIGINT PRIMARY KEY,
    student_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    semester_id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    grade_id BIGINT NOT NULL,
    original_score DECIMAL(6,2) NOT NULL,
    appeal_reason TEXT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    handled_by BIGINT NULL,
    handled_at TIMESTAMP NULL,
    new_score DECIMAL(6,2) NULL,
    handle_comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_grade_appeals_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_grade_appeals_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_grade_appeals_semester_id FOREIGN KEY (semester_id) REFERENCES semesters(id),
    CONSTRAINT fk_grade_appeals_grade_id FOREIGN KEY (grade_id) REFERENCES student_semester_grades(id),
    CONSTRAINT fk_grade_appeals_handled_by FOREIGN KEY (handled_by) REFERENCES users(id)
);
CREATE INDEX idx_grade_appeals_student_id ON grade_appeals(student_id);
CREATE INDEX idx_grade_appeals_course_id ON grade_appeals(course_id);
CREATE INDEX idx_grade_appeals_status ON grade_appeals(status);
CREATE UNIQUE INDEX uk_grade_appeals_student_course_semester ON grade_appeals(student_id, course_id, semester_id);
CREATE INDEX idx_grade_appeals_school_id ON grade_appeals(school_id);

-- academic_warnings：学业预警表。
CREATE TABLE academic_warnings (
    id BIGINT PRIMARY KEY,
    student_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    semester_id BIGINT NOT NULL,
    warning_type SMALLINT NOT NULL,
    detail JSONB NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    handled_by BIGINT NULL,
    handled_at TIMESTAMP NULL,
    handle_note TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_academic_warnings_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_academic_warnings_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_academic_warnings_semester_id FOREIGN KEY (semester_id) REFERENCES semesters(id),
    CONSTRAINT fk_academic_warnings_handled_by FOREIGN KEY (handled_by) REFERENCES users(id)
);
CREATE INDEX idx_academic_warnings_student_id ON academic_warnings(student_id);
CREATE INDEX idx_academic_warnings_school_id ON academic_warnings(school_id);
CREATE INDEX idx_academic_warnings_semester_id ON academic_warnings(semester_id);
CREATE INDEX idx_academic_warnings_status ON academic_warnings(status);

-- warning_configs：学业预警配置表。
CREATE TABLE warning_configs (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    gpa_threshold DECIMAL(3,2) NOT NULL DEFAULT 2.00,
    fail_count_threshold INT NOT NULL DEFAULT 2,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_warning_configs_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE UNIQUE INDEX uk_warning_configs_school_id ON warning_configs(school_id);

-- transcript_records：成绩单生成记录表。
CREATE TABLE transcript_records (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    generated_by BIGINT NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    include_semesters JSONB NOT NULL,
    generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_transcript_records_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_transcript_records_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_transcript_records_generated_by FOREIGN KEY (generated_by) REFERENCES users(id)
);
CREATE INDEX idx_transcript_records_student_id ON transcript_records(student_id);
CREATE INDEX idx_transcript_records_school_id ON transcript_records(school_id);

-- 为模块03补充跨模块外键约束。
ALTER TABLE courses
    ADD CONSTRAINT fk_courses_semester_id FOREIGN KEY (semester_id) REFERENCES semesters(id);

ALTER TABLE grade_reviews
    ADD CONSTRAINT fk_grade_reviews_course_id FOREIGN KEY (course_id) REFERENCES courses(id);

ALTER TABLE student_semester_grades
    ADD CONSTRAINT fk_student_semester_grades_course_id FOREIGN KEY (course_id) REFERENCES courses(id);

ALTER TABLE grade_appeals
    ADD CONSTRAINT fk_grade_appeals_course_id FOREIGN KEY (course_id) REFERENCES courses(id);
