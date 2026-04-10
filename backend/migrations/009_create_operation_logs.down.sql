-- 009_create_operation_logs.down.sql
-- 回滚：删除 operation_logs 操作日志表

DROP TABLE IF EXISTS operation_logs;
