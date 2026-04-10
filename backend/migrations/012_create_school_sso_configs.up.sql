-- 012_create_school_sso_configs.up.sql
-- 模块02 — 学校与租户管理：SSO配置表
-- 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

CREATE TABLE school_sso_configs (
    id         BIGINT       PRIMARY KEY,
    school_id  BIGINT       NOT NULL,
    provider   VARCHAR(20)  NOT NULL,
    is_enabled BOOLEAN      NOT NULL DEFAULT FALSE,
    is_tested  BOOLEAN      NOT NULL DEFAULT FALSE,
    config     JSONB        NOT NULL DEFAULT '{}',
    tested_at  TIMESTAMP,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_by BIGINT
);

CREATE UNIQUE INDEX uk_school_sso_configs_school_id ON school_sso_configs(school_id);

COMMENT ON TABLE school_sso_configs IS 'SSO配置表';
COMMENT ON COLUMN school_sso_configs.provider IS 'SSO协议类型：cas / oauth2';
COMMENT ON COLUMN school_sso_configs.config IS 'SSO配置参数（JSON格式），client_secret加密存储';
