-- 058_add_sim_websocket_url_to_experiment_instances.up.sql
-- 模块04 — 实验环境：为实验实例补充 SimEngine WebSocket 上游地址

ALTER TABLE experiment_instances
    ADD COLUMN sim_websocket_url VARCHAR(500) NULL;
