-- 057_create_experiment_reports.up.sql
-- 模块04 — 实验环境：实验报告表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.24节

CREATE TABLE experiment_reports (
    id          BIGINT        PRIMARY KEY,
    instance_id BIGINT        NOT NULL REFERENCES experiment_instances(id),
    student_id  BIGINT        NOT NULL,
    content     TEXT          NULL,
    file_url    VARCHAR(500)  NULL,
    file_name   VARCHAR(200)  NULL,
    file_size   BIGINT        NULL,
    submitted_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_experiment_reports_instance_id ON experiment_reports(instance_id);
CREATE INDEX idx_experiment_reports_student_id ON experiment_reports(student_id);
CREATE UNIQUE INDEX uk_experiment_reports_instance ON experiment_reports(instance_id, student_id);
