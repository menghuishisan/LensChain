-- 036_create_image_versions.up.sql
-- 模块04 — 实验环境：镜像版本表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.3节

CREATE TABLE image_versions (
    id           BIGINT       PRIMARY KEY,
    image_id     BIGINT       NOT NULL REFERENCES images(id),
    version      VARCHAR(50)  NOT NULL,
    registry_url VARCHAR(500) NOT NULL,
    image_size   BIGINT       NULL,
    digest       VARCHAR(200) NULL,
    min_cpu      VARCHAR(20)  NULL,
    min_memory   VARCHAR(20)  NULL,
    min_disk     VARCHAR(20)  NULL,
    is_default   BOOLEAN      NOT NULL DEFAULT FALSE,
    status       SMALLINT     NOT NULL DEFAULT 1,
    scan_result  JSONB        NULL,
    scanned_at   TIMESTAMP    NULL,
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_image_versions_image_id ON image_versions(image_id);
CREATE UNIQUE INDEX uk_image_versions_image_version ON image_versions(image_id, version);
