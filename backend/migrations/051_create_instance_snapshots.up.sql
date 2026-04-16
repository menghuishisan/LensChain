-- 051_create_instance_snapshots.up.sql
-- 模块04 — 实验环境：实例快照表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.18节

CREATE TABLE instance_snapshots (
    id                BIGINT       PRIMARY KEY,
    instance_id       BIGINT       NOT NULL REFERENCES experiment_instances(id),
    snapshot_type     SMALLINT     NOT NULL,
    snapshot_data_url VARCHAR(500) NOT NULL,
    container_states  JSONB        NULL,
    sim_engine_state  JSONB        NULL,
    description       VARCHAR(200) NULL,
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instance_snapshots_instance_id ON instance_snapshots(instance_id);
