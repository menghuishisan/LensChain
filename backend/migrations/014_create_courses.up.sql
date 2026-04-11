-- 014_create_courses.up.sql
-- 模块03 — 课程与教学：课程主表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE courses (
    id              BIGINT       PRIMARY KEY,
    school_id       BIGINT       NOT NULL,
    teacher_id      BIGINT       NOT NULL,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    cover_url       VARCHAR(500),
    course_type     SMALLINT     NOT NULL,
    difficulty      SMALLINT     NOT NULL DEFAULT 1,
    topic           VARCHAR(50)  NOT NULL,
    status          SMALLINT     NOT NULL DEFAULT 1,
    is_shared       BOOLEAN      NOT NULL DEFAULT FALSE,
    invite_code     VARCHAR(10),
    start_at        TIMESTAMP,
    end_at          TIMESTAMP,
    max_students    INT,
    cloned_from_id  BIGINT,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_courses_school_id ON courses(school_id);
CREATE INDEX idx_courses_teacher_id ON courses(teacher_id);
CREATE INDEX idx_courses_status ON courses(status);
CREATE INDEX idx_courses_course_type ON courses(course_type);
CREATE INDEX idx_courses_is_shared ON courses(is_shared) WHERE is_shared = TRUE;
CREATE UNIQUE INDEX uk_courses_invite_code ON courses(invite_code) WHERE invite_code IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_courses_start_at ON courses(start_at);
CREATE INDEX idx_courses_end_at ON courses(end_at);

COMMENT ON TABLE courses IS '课程主表';
COMMENT ON COLUMN courses.course_type IS '类型：1理论 2实验 3混合 4项目实战';
COMMENT ON COLUMN courses.difficulty IS '难度：1入门 2进阶 3高级 4研究';
COMMENT ON COLUMN courses.status IS '状态：1草稿 2已发布 3进行中 4已结束 5已归档';
