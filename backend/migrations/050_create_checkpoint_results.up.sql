-- 050_create_checkpoint_results.up.sql
-- 模块04 — 实验环境：检查点结果表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.17节

CREATE TABLE checkpoint_results (
    id              BIGINT        PRIMARY KEY,
    instance_id     BIGINT        NOT NULL REFERENCES experiment_instances(id),
    checkpoint_id   BIGINT        NOT NULL REFERENCES template_checkpoints(id),
    student_id      BIGINT        NOT NULL,
    is_passed       BOOLEAN       NOT NULL DEFAULT FALSE,
    score           DECIMAL(6,2)  NULL,
    check_output    TEXT          NULL,
    assertion_result JSONB        NULL,
    teacher_comment VARCHAR(500)  NULL,
    graded_by       BIGINT        NULL,
    graded_at       TIMESTAMP     NULL,
    checked_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkpoint_results_instance_id ON checkpoint_results(instance_id);
CREATE INDEX idx_checkpoint_results_checkpoint_id ON checkpoint_results(checkpoint_id);
CREATE INDEX idx_checkpoint_results_student_id ON checkpoint_results(student_id);
CREATE UNIQUE INDEX uk_checkpoint_results ON checkpoint_results(instance_id, checkpoint_id);
