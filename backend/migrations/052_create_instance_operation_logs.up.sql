-- 052_create_instance_operation_logs.up.sql
-- 模块04 — 实验环境：实例操作日志表（只插入不更新不删除）
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.19节

CREATE TABLE instance_operation_logs (
    id               BIGINT       PRIMARY KEY,
    instance_id      BIGINT       NOT NULL REFERENCES experiment_instances(id),
    student_id       BIGINT       NOT NULL,
    action           VARCHAR(50)  NOT NULL,
    target_container VARCHAR(100) NULL,
    target_scene     VARCHAR(100) NULL,
    command          TEXT         NULL,
    detail           JSONB        NULL,
    client_ip        VARCHAR(50)  NULL,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instance_operation_logs_instance_id ON instance_operation_logs(instance_id);
CREATE INDEX idx_instance_operation_logs_student_id ON instance_operation_logs(student_id);
CREATE INDEX idx_instance_operation_logs_action ON instance_operation_logs(action);
CREATE INDEX idx_instance_operation_logs_created_at ON instance_operation_logs(created_at);
