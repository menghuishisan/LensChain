-- 060_align_checkpoint_and_report_timestamps.up.sql
-- 模块04 — 实验环境：统一检查点结果与实验报告时间字段语义
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.17、2.24节

ALTER TABLE checkpoint_results
    ADD COLUMN IF NOT EXISTS checked_at TIMESTAMP;

UPDATE checkpoint_results
SET checked_at = COALESCE(checked_at, graded_at, updated_at, created_at, NOW());

ALTER TABLE checkpoint_results
    ALTER COLUMN checked_at SET NOT NULL,
    ALTER COLUMN checked_at SET DEFAULT NOW();

DELETE FROM checkpoint_results older
USING checkpoint_results newer
WHERE older.instance_id = newer.instance_id
  AND older.checkpoint_id = newer.checkpoint_id
  AND (
      COALESCE(older.checked_at, older.graded_at, older.updated_at, older.created_at) <
      COALESCE(newer.checked_at, newer.graded_at, newer.updated_at, newer.created_at)
      OR (
          COALESCE(older.checked_at, older.graded_at, older.updated_at, older.created_at) =
          COALESCE(newer.checked_at, newer.graded_at, newer.updated_at, newer.created_at)
          AND older.id < newer.id
      )
  );

DROP INDEX IF EXISTS uk_checkpoint_results;
CREATE UNIQUE INDEX uk_checkpoint_results ON checkpoint_results(instance_id, checkpoint_id);

ALTER TABLE experiment_reports
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMP;

UPDATE experiment_reports
SET submitted_at = COALESCE(submitted_at, created_at, NOW());

ALTER TABLE experiment_reports
    ALTER COLUMN submitted_at SET NOT NULL,
    ALTER COLUMN submitted_at SET DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_experiment_reports_student_id ON experiment_reports(student_id);
