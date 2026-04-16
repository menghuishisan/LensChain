-- 059_align_snapshot_and_operation_log_schema.down.sql
-- 回滚快照大小与终端命令截断输出字段

ALTER TABLE instance_operation_logs
    DROP COLUMN IF EXISTS command_output;

ALTER TABLE instance_snapshots
    ALTER COLUMN snapshot_data_url DROP NOT NULL;

ALTER TABLE instance_snapshots
    DROP COLUMN IF EXISTS snapshot_size;
