-- 011_create_school_applications.up.sql
-- 模块02 — 学校与租户管理：入驻申请记录表
-- 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

CREATE TABLE school_applications (
    id                      BIGINT       PRIMARY KEY,
    school_name             VARCHAR(100) NOT NULL,
    school_code             VARCHAR(50)  NOT NULL,
    school_address          VARCHAR(200),
    school_website          VARCHAR(200),
    school_logo_url         VARCHAR(500),
    contact_name            VARCHAR(50)  NOT NULL,
    contact_phone           VARCHAR(20)  NOT NULL,
    contact_email           VARCHAR(100),
    contact_title           VARCHAR(100),
    status                  SMALLINT     NOT NULL DEFAULT 1,
    reviewer_id             BIGINT,
    reviewed_at             TIMESTAMP,
    reject_reason           VARCHAR(500),
    school_id               BIGINT,
    previous_application_id BIGINT,
    created_at              TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_school_applications_status ON school_applications(status);
CREATE INDEX idx_school_applications_contact_phone ON school_applications(contact_phone);
CREATE INDEX idx_school_applications_created_at ON school_applications(created_at);

COMMENT ON TABLE school_applications IS '入驻申请记录表';
COMMENT ON COLUMN school_applications.status IS '状态：1待审核 2已通过 3已拒绝';
