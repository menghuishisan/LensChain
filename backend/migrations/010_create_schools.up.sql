-- 010_create_schools.up.sql
-- 模块02 — 学校与租户管理：学校主表
-- 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

CREATE TABLE schools (
    id              BIGINT       PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(50)  NOT NULL,
    logo_url        VARCHAR(500),
    address         VARCHAR(200),
    website         VARCHAR(200),
    description     TEXT,
    status          SMALLINT     NOT NULL DEFAULT 1,
    license_start_at TIMESTAMP,
    license_end_at  TIMESTAMP,
    frozen_at       TIMESTAMP,
    frozen_reason   VARCHAR(200),
    contact_name    VARCHAR(50)  NOT NULL,
    contact_phone   VARCHAR(20)  NOT NULL,
    contact_email   VARCHAR(100),
    contact_title   VARCHAR(100),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    created_by      BIGINT,
    deleted_at      TIMESTAMP
);

-- 部分唯一索引（软删除后不占用唯一性）
CREATE UNIQUE INDEX uk_schools_name ON schools(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_schools_code ON schools(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_schools_status ON schools(status);
CREATE INDEX idx_schools_license_end_at ON schools(license_end_at);
CREATE INDEX idx_schools_contact_phone ON schools(contact_phone);

COMMENT ON TABLE schools IS '学校主表';
COMMENT ON COLUMN schools.status IS '状态：1待审核 2已激活 3缓冲期 4已冻结 5已注销 6已拒绝';
