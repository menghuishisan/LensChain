-- 059_align_snapshot_and_operation_log_schema.up.sql
-- 模块04 — 实验环境：补齐快照大小与终端命令截断输出字段
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.18 / 2.19节

ALTER TABLE instance_snapshots
    ADD COLUMN IF NOT EXISTS snapshot_size BIGINT NULL;

DELETE FROM instance_snapshots
WHERE snapshot_data_url IS NULL
   OR BTRIM(snapshot_data_url) = '';

ALTER TABLE instance_snapshots
    ALTER COLUMN snapshot_data_url SET NOT NULL;

ALTER TABLE instance_operation_logs
    ADD COLUMN IF NOT EXISTS command_output TEXT NULL;
