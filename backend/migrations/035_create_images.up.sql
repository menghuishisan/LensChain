-- 035_create_images.up.sql
-- 模块04 — 实验环境：镜像主表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.2节

CREATE TABLE images (
    id                      BIGINT        PRIMARY KEY,
    category_id             BIGINT        NOT NULL REFERENCES image_categories(id),
    name                    VARCHAR(100)  NOT NULL,
    display_name            VARCHAR(100)  NOT NULL,
    description             TEXT          NULL,
    icon_url                VARCHAR(500)  NULL,
    ecosystem               VARCHAR(50)   NULL,
    source_type             SMALLINT      NOT NULL DEFAULT 1,
    uploaded_by             BIGINT        NULL,
    school_id               BIGINT        NULL,
    status                  SMALLINT      NOT NULL DEFAULT 1,
    review_comment          VARCHAR(500)  NULL,
    reviewed_by             BIGINT        NULL,
    reviewed_at             TIMESTAMP     NULL,
    default_ports           JSONB         NULL,
    default_env_vars        JSONB         NULL,
    default_volumes         JSONB         NULL,
    typical_companions      JSONB         NULL,
    required_dependencies   JSONB         NULL,
    resource_recommendation JSONB         NULL,
    documentation_url       VARCHAR(500)  NULL,
    usage_count             INT           NOT NULL DEFAULT 0,
    created_at              TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMP     NULL
);

CREATE INDEX idx_images_category_id ON images(category_id);
CREATE INDEX idx_images_ecosystem ON images(ecosystem);
CREATE INDEX idx_images_source_type ON images(source_type);
CREATE INDEX idx_images_status ON images(status);
CREATE INDEX idx_images_school_id ON images(school_id);
CREATE INDEX idx_images_uploaded_by ON images(uploaded_by);
