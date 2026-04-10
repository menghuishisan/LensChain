-- 009_create_operation_logs.up.sql
-- 模块01 — 用户与认证：创建 operation_logs 操作日志表
-- 审计日志红线：只插入，不更新，不删除
-- 对照 docs/modules/01-用户与认证/02-数据库设计.md

CREATE TABLE IF NOT EXISTS operation_logs (
    id          BIGINT      PRIMARY KEY,                               -- 雪花算法ID
    operator_id BIGINT      NOT NULL,                                  -- 操作人ID
    action      VARCHAR(50) NOT NULL,                                  -- 操作类型（如 import_students, reset_password）
    target_type VARCHAR(50) NOT NULL,                                  -- 操作对象类型（如 user）
    target_id   BIGINT      NULL,                                      -- 操作对象ID
    detail      JSONB       NULL,                                      -- 操作详情（变更前后的数据快照）
    ip          VARCHAR(45) NOT NULL,                                  -- 操作人IP
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW()                     -- 操作时间
);

-- 索引（4个）
CREATE INDEX idx_operation_logs_operator_id ON operation_logs(operator_id);
CREATE INDEX idx_operation_logs_action ON operation_logs(action);
CREATE INDEX idx_operation_logs_target ON operation_logs(target_type, target_id);
CREATE INDEX idx_operation_logs_created_at ON operation_logs(created_at);

COMMENT ON TABLE operation_logs IS '操作日志表（审计日志，只插入不更新不删除）';
