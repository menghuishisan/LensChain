-- 049_create_instance_containers.up.sql
-- 模块04 — 实验环境：实例容器表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.16节

CREATE TABLE instance_containers (
    id                    BIGINT       PRIMARY KEY,
    instance_id           BIGINT       NOT NULL REFERENCES experiment_instances(id),
    template_container_id BIGINT       NOT NULL REFERENCES template_containers(id),
    container_name        VARCHAR(100) NOT NULL,
    pod_name              VARCHAR(200) NULL,
    internal_ip           VARCHAR(50)  NULL,
    status                SMALLINT     NOT NULL DEFAULT 1,
    cpu_usage             VARCHAR(20)  NULL,
    memory_usage          VARCHAR(20)  NULL,
    created_at            TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instance_containers_instance_id ON instance_containers(instance_id);
