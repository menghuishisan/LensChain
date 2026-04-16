-- 058_add_sim_websocket_url_to_experiment_instances.down.sql
-- 模块04 — 实验环境：回滚实验实例 SimEngine WebSocket 上游地址字段

ALTER TABLE experiment_instances
    DROP COLUMN IF EXISTS sim_websocket_url;
