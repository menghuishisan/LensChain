-- 038_create_template_containers.up.sql
-- 模块04 — 实验环境：模板容器配置表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.5节

CREATE TABLE template_containers (
    id               BIGINT       PRIMARY KEY,
    template_id      BIGINT       NOT NULL REFERENCES experiment_templates(id),
    image_version_id BIGINT       NOT NULL REFERENCES image_versions(id),
    container_name   VARCHAR(100) NOT NULL,
    role_id          BIGINT       NULL,
    env_vars         JSONB        NULL,
    ports            JSONB        NULL,
    volumes          JSONB        NULL,
    cpu_limit        VARCHAR(20)  NULL,
    memory_limit     VARCHAR(20)  NULL,
    depends_on       JSONB        NULL,
    startup_order    INT          NOT NULL DEFAULT 0,
    is_primary       BOOLEAN      NOT NULL DEFAULT FALSE,
    sort_order       INT          NOT NULL DEFAULT 0,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_template_containers_template_id ON template_containers(template_id);
CREATE INDEX idx_template_containers_image_version_id ON template_containers(image_version_id);
