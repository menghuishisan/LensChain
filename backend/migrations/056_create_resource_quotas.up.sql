-- 056_create_resource_quotas.up.sql
-- 模块04 — 实验环境：资源配额表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.23节

CREATE TABLE resource_quotas (
    id              BIGINT       PRIMARY KEY,
    quota_level     SMALLINT     NOT NULL,
    school_id       BIGINT       NOT NULL,
    course_id       BIGINT       NULL,
    max_cpu         VARCHAR(20)  NOT NULL DEFAULT '0',
    max_memory      VARCHAR(20)  NOT NULL DEFAULT '0',
    max_storage     VARCHAR(20)  NOT NULL DEFAULT '0',
    max_concurrency INT          NOT NULL DEFAULT 0,
    max_per_student INT          NOT NULL DEFAULT 2,
    used_cpu        VARCHAR(20)  NOT NULL DEFAULT '0',
    used_memory     VARCHAR(20)  NOT NULL DEFAULT '0',
    used_storage    VARCHAR(20)  NOT NULL DEFAULT '0',
    used_concurrency INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resource_quotas_school_id ON resource_quotas(school_id);
CREATE INDEX idx_resource_quotas_course_id ON resource_quotas(course_id);
CREATE UNIQUE INDEX uk_resource_quotas_school ON resource_quotas(school_id) WHERE quota_level = 1 AND course_id IS NULL;
CREATE UNIQUE INDEX uk_resource_quotas_course ON resource_quotas(school_id, course_id) WHERE quota_level = 2 AND course_id IS NOT NULL;
