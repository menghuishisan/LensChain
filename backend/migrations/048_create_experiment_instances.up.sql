-- 048_create_experiment_instances.up.sql
-- 模块04 — 实验环境：实验实例表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.15节

CREATE TABLE experiment_instances (
    id              BIGINT        PRIMARY KEY,
    template_id     BIGINT        NOT NULL REFERENCES experiment_templates(id),
    student_id      BIGINT        NOT NULL,
    school_id       BIGINT        NOT NULL,
    course_id       BIGINT        NULL,
    lesson_id       BIGINT        NULL,
    assignment_id   BIGINT        NULL,
    group_id        BIGINT        NULL,
    experiment_type SMALLINT      NOT NULL,
    status          SMALLINT      NOT NULL DEFAULT 1,
    attempt_no      INT           NOT NULL DEFAULT 1,
    namespace       VARCHAR(100)  NULL,
    access_url      VARCHAR(500)  NULL,
    total_score     DECIMAL(6,2)  NULL,
    auto_score      DECIMAL(6,2)  NULL,
    manual_score    DECIMAL(6,2)  NULL,
    started_at      TIMESTAMP     NULL,
    paused_at       TIMESTAMP     NULL,
    submitted_at    TIMESTAMP     NULL,
    destroyed_at    TIMESTAMP     NULL,
    last_active_at  TIMESTAMP     NULL,
    error_message   TEXT          NULL,
    sim_session_id  VARCHAR(100)  NULL,
    created_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_experiment_instances_template_id ON experiment_instances(template_id);
CREATE INDEX idx_experiment_instances_student_id ON experiment_instances(student_id);
CREATE INDEX idx_experiment_instances_school_id ON experiment_instances(school_id);
CREATE INDEX idx_experiment_instances_course_id ON experiment_instances(course_id);
CREATE INDEX idx_experiment_instances_group_id ON experiment_instances(group_id);
CREATE INDEX idx_experiment_instances_status ON experiment_instances(status);
