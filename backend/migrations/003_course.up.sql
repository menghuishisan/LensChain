-- 模块03 课程与教学
-- 文档依据：
-- 1. docs/modules/03-课程与教学/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 课程、章节、课时、选课
-- 2. 作业、提交、答题记录
-- 3. 公告、讨论、评价、进度、成绩配置
-- 不负责：
-- 1. 实验模板主数据，相关字段仅保留外键引用
-- 2. 学期与成绩审核主数据，相关字段仅保留外键引用

-- courses：课程主表。
CREATE TABLE courses (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    teacher_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    cover_url VARCHAR(500) NULL,
    course_type SMALLINT NOT NULL,
    difficulty SMALLINT NOT NULL DEFAULT 1,
    topic VARCHAR(50) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    invite_code VARCHAR(10) NULL,
    start_at TIMESTAMP NULL,
    end_at TIMESTAMP NULL,
    credits DECIMAL(3,1) NULL,
    semester_id BIGINT NULL,
    max_students INT NULL,
    cloned_from_id BIGINT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_courses_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_courses_teacher_id FOREIGN KEY (teacher_id) REFERENCES users(id),
    CONSTRAINT fk_courses_cloned_from_id FOREIGN KEY (cloned_from_id) REFERENCES courses(id)
);
CREATE INDEX idx_courses_school_id ON courses(school_id);
CREATE INDEX idx_courses_teacher_id ON courses(teacher_id);
CREATE INDEX idx_courses_status ON courses(status);
CREATE INDEX idx_courses_course_type ON courses(course_type);
CREATE INDEX idx_courses_is_shared ON courses(is_shared) WHERE is_shared = TRUE;
CREATE UNIQUE INDEX uk_courses_invite_code ON courses(invite_code) WHERE invite_code IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_courses_start_at ON courses(start_at);
CREATE INDEX idx_courses_end_at ON courses(end_at);
CREATE INDEX idx_courses_semester_id ON courses(semester_id);

-- chapters：课程章节表。
CREATE TABLE chapters (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_chapters_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_chapters_course_id ON chapters(course_id);
CREATE INDEX idx_chapters_sort_order ON chapters(course_id, sort_order);

-- lessons：课时表。
CREATE TABLE lessons (
    id BIGINT PRIMARY KEY,
    chapter_id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content_type SMALLINT NOT NULL,
    content TEXT NULL,
    video_url VARCHAR(500) NULL,
    video_duration INT NULL,
    experiment_id BIGINT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    estimated_minutes INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_lessons_chapter_id FOREIGN KEY (chapter_id) REFERENCES chapters(id),
    CONSTRAINT fk_lessons_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_lessons_chapter_id ON lessons(chapter_id);
CREATE INDEX idx_lessons_course_id ON lessons(course_id);
CREATE INDEX idx_lessons_sort_order ON lessons(chapter_id, sort_order);

-- lesson_attachments：课时附件表。
CREATE TABLE lesson_attachments (
    id BIGINT PRIMARY KEY,
    lesson_id BIGINT NOT NULL,
    file_name VARCHAR(200) NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_lesson_attachments_lesson_id FOREIGN KEY (lesson_id) REFERENCES lessons(id)
);
CREATE INDEX idx_lesson_attachments_lesson_id ON lesson_attachments(lesson_id);

-- course_enrollments：选课记录表。
CREATE TABLE course_enrollments (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    join_method SMALLINT NOT NULL,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    removed_at TIMESTAMP NULL,
    CONSTRAINT fk_course_enrollments_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_enrollments_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_enrollments_course_student ON course_enrollments(course_id, student_id) WHERE removed_at IS NULL;
CREATE INDEX idx_enrollments_student_id ON course_enrollments(student_id);

-- assignments：作业/测验表。
CREATE TABLE assignments (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    chapter_id BIGINT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    assignment_type SMALLINT NOT NULL,
    total_score DECIMAL(6,2) NOT NULL,
    deadline_at TIMESTAMP NOT NULL,
    max_submissions INT NOT NULL DEFAULT 1,
    late_policy SMALLINT NOT NULL DEFAULT 1,
    late_deduction_per_day DECIMAL(5,2) NULL,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_assignments_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_assignments_chapter_id FOREIGN KEY (chapter_id) REFERENCES chapters(id)
);
CREATE INDEX idx_assignments_course_id ON assignments(course_id);
CREATE INDEX idx_assignments_chapter_id ON assignments(chapter_id);
CREATE INDEX idx_assignments_deadline_at ON assignments(deadline_at);

-- assignment_questions：作业题目表。
CREATE TABLE assignment_questions (
    id BIGINT PRIMARY KEY,
    assignment_id BIGINT NOT NULL,
    question_type SMALLINT NOT NULL,
    title TEXT NOT NULL,
    options JSONB NULL,
    correct_answer TEXT NULL,
    reference_answer TEXT NULL,
    score DECIMAL(6,2) NOT NULL,
    judge_config JSONB NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_assignment_questions_assignment_id FOREIGN KEY (assignment_id) REFERENCES assignments(id)
);
CREATE INDEX idx_assignment_questions_assignment_id ON assignment_questions(assignment_id);

-- assignment_submissions：学生提交表。
CREATE TABLE assignment_submissions (
    id BIGINT PRIMARY KEY,
    assignment_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    submission_no INT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    total_score DECIMAL(6,2) NULL,
    is_late BOOLEAN NOT NULL DEFAULT FALSE,
    late_days INT NULL,
    score_before_deduction DECIMAL(6,2) NULL,
    score_after_deduction DECIMAL(6,2) NULL,
    graded_by BIGINT NULL,
    graded_at TIMESTAMP NULL,
    teacher_comment TEXT NULL,
    submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_assignment_submissions_assignment_id FOREIGN KEY (assignment_id) REFERENCES assignments(id),
    CONSTRAINT fk_assignment_submissions_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_assignment_submissions_graded_by FOREIGN KEY (graded_by) REFERENCES users(id)
);
CREATE INDEX idx_submissions_assignment_id ON assignment_submissions(assignment_id);
CREATE INDEX idx_submissions_student_id ON assignment_submissions(student_id);
CREATE UNIQUE INDEX uk_submissions_assignment_student_no ON assignment_submissions(assignment_id, student_id, submission_no);
CREATE INDEX idx_submissions_status ON assignment_submissions(status);

-- assignment_drafts：学生作答草稿表。
CREATE TABLE assignment_drafts (
    id BIGINT PRIMARY KEY,
    assignment_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    saved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_assignment_drafts_assignment_id FOREIGN KEY (assignment_id) REFERENCES assignments(id),
    CONSTRAINT fk_assignment_drafts_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_assignment_drafts_assignment_student ON assignment_drafts(assignment_id, student_id);
CREATE INDEX idx_assignment_drafts_student_id ON assignment_drafts(student_id);
CREATE INDEX idx_assignment_drafts_saved_at ON assignment_drafts(saved_at);

-- submission_answers：答题记录表。
CREATE TABLE submission_answers (
    id BIGINT PRIMARY KEY,
    submission_id BIGINT NOT NULL,
    question_id BIGINT NOT NULL,
    answer_content TEXT NULL,
    answer_file_url VARCHAR(500) NULL,
    is_correct BOOLEAN NULL,
    score DECIMAL(6,2) NULL,
    teacher_comment TEXT NULL,
    auto_judge_result JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_submission_answers_submission_id FOREIGN KEY (submission_id) REFERENCES assignment_submissions(id),
    CONSTRAINT fk_submission_answers_question_id FOREIGN KEY (question_id) REFERENCES assignment_questions(id)
);
CREATE INDEX idx_submission_answers_submission_id ON submission_answers(submission_id);
CREATE INDEX idx_submission_answers_question_id ON submission_answers(question_id);

-- learning_progresses：学习进度表。
CREATE TABLE learning_progresses (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    lesson_id BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    video_progress INT NULL,
    study_duration INT NOT NULL DEFAULT 0,
    completed_at TIMESTAMP NULL,
    last_accessed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_learning_progresses_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_learning_progresses_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_learning_progresses_lesson_id FOREIGN KEY (lesson_id) REFERENCES lessons(id)
);
CREATE UNIQUE INDEX uk_learning_progress ON learning_progresses(course_id, student_id, lesson_id);
CREATE INDEX idx_learning_progresses_student_id ON learning_progresses(student_id);
CREATE INDEX idx_learning_progresses_status ON learning_progresses(status);

-- course_schedules：课程表。
CREATE TABLE course_schedules (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    day_of_week SMALLINT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    location VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_course_schedules_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_course_schedules_course_id ON course_schedules(course_id);

-- course_announcements：课程公告表。
CREATE TABLE course_announcements (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    teacher_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_course_announcements_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_announcements_teacher_id FOREIGN KEY (teacher_id) REFERENCES users(id)
);
CREATE INDEX idx_course_announcements_course_id ON course_announcements(course_id);

-- course_discussions：讨论帖表。
CREATE TABLE course_discussions (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    reply_count INT NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    last_replied_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_course_discussions_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_discussions_author_id FOREIGN KEY (author_id) REFERENCES users(id)
);
CREATE INDEX idx_course_discussions_course_id ON course_discussions(course_id);
CREATE INDEX idx_course_discussions_author_id ON course_discussions(author_id);
CREATE INDEX idx_course_discussions_is_pinned ON course_discussions(course_id, is_pinned);

-- discussion_replies：讨论回复表。
CREATE TABLE discussion_replies (
    id BIGINT PRIMARY KEY,
    discussion_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    reply_to_id BIGINT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_discussion_replies_discussion_id FOREIGN KEY (discussion_id) REFERENCES course_discussions(id),
    CONSTRAINT fk_discussion_replies_author_id FOREIGN KEY (author_id) REFERENCES users(id),
    CONSTRAINT fk_discussion_replies_reply_to_id FOREIGN KEY (reply_to_id) REFERENCES discussion_replies(id)
);
CREATE INDEX idx_discussion_replies_discussion_id ON discussion_replies(discussion_id);

-- discussion_likes：讨论点赞表。
CREATE TABLE discussion_likes (
    id BIGINT PRIMARY KEY,
    discussion_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_discussion_likes_discussion_id FOREIGN KEY (discussion_id) REFERENCES course_discussions(id),
    CONSTRAINT fk_discussion_likes_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_discussion_likes ON discussion_likes(discussion_id, user_id);

-- course_evaluations：课程评价表。
CREATE TABLE course_evaluations (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    rating SMALLINT NOT NULL,
    comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_course_evaluations_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_evaluations_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_course_evaluations ON course_evaluations(course_id, student_id);
CREATE INDEX idx_course_evaluations_course_id ON course_evaluations(course_id);

-- course_grade_configs：成绩权重配置表。
CREATE TABLE course_grade_configs (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_course_grade_configs_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE UNIQUE INDEX uk_course_grade_configs_course_id ON course_grade_configs(course_id);

-- course_experiments：课程独立实验关联表。
CREATE TABLE course_experiments (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    experiment_id BIGINT NOT NULL,
    title VARCHAR(200) NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_course_experiments_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_course_experiments_course_id ON course_experiments(course_id);
CREATE UNIQUE INDEX uk_course_experiments ON course_experiments(course_id, experiment_id);

-- course_grade_overrides：课程成绩调整记录表。
CREATE TABLE course_grade_overrides (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    weighted_total DECIMAL(6,2) NOT NULL,
    final_score DECIMAL(6,2) NOT NULL,
    adjust_reason VARCHAR(200) NOT NULL,
    adjusted_by BIGINT NOT NULL,
    adjusted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_course_grade_overrides_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_course_grade_overrides_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_course_grade_overrides_adjusted_by FOREIGN KEY (adjusted_by) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_course_grade_overrides_course_student ON course_grade_overrides(course_id, student_id);
CREATE INDEX idx_course_grade_overrides_course_id ON course_grade_overrides(course_id);
CREATE INDEX idx_course_grade_overrides_student_id ON course_grade_overrides(student_id);
