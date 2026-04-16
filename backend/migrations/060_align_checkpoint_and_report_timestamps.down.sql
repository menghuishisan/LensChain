-- 060_align_checkpoint_and_report_timestamps.down.sql
-- 模块04 — 实验环境：回滚检查点结果与实验报告时间字段对齐

ALTER TABLE checkpoint_results
    DROP COLUMN IF EXISTS checked_at;

ALTER TABLE experiment_reports
    DROP COLUMN IF EXISTS submitted_at;
