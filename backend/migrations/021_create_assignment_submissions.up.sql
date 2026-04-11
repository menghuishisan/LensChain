-- 021_create_assignment_submissions.up.sql
-- 模块03 — 课程与教学：学生提交表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE assignment_submissions (
    id                      BIGINT       PRIMARY KEY,
    assignment_id           BIGINT       NOT NULL,
    student_id              BIGINT       NOT NULL,
    submission_no           INT          NOT NULL,
    status                  SMALLINT     NOT NULL DEFAULT 1,
    total_score             DECIMAL(6,2),
    is_late                 BOOLEAN      NOT NULL DEFAULT FALSE,
    late_days               INT,
    score_before_deduction  DECIMAL(6,2),
    score_after_deduction   DECIMAL(6,2),
    graded_by               BIGINT,
    graded_at               TIMESTAMP,
    teacher_comment         TEXT,
    submitted_at            TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submissions_assignment_id ON assignment_submissions(assignment_id);
CREATE INDEX idx_submissions_student_id ON assignment_submissions(student_id);
CREATE UNIQUE INDEX uk_submissions_assignment_student_no ON assignment_submissions(assignment_id, student_id, submission_no);
CREATE INDEX idx_submissions_status ON assignment_submissions(status);

COMMENT ON TABLE assignment_submissions IS '学生提交表';
COMMENT ON COLUMN assignment_submissions.status IS '状态：1已提交 2待批改 3已批改';
