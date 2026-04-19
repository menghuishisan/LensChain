-- 模块02 学校与租户管理
-- 文档依据：
-- 1. docs/modules/02-学校与租户管理/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 学校租户主表与入驻申请表
-- 2. 学校级 SSO 配置与学校通知表
-- 3. 为模块01补充 school_id 外键约束
-- 不负责：
-- 1. 用户角色与认证日志表
-- 2. 课程、实验、竞赛、成绩、通知、系统管理相关表

-- schools：学校租户主表，记录平台入驻学校的基础信息与授权状态。
CREATE TABLE schools (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    logo_url VARCHAR(500) NULL,
    address VARCHAR(200) NULL,
    website VARCHAR(200) NULL,
    description TEXT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    license_start_at TIMESTAMP NULL,
    license_end_at TIMESTAMP NULL,
    frozen_at TIMESTAMP NULL,
    frozen_reason VARCHAR(200) NULL,
    contact_name VARCHAR(50) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    contact_email VARCHAR(100) NULL,
    contact_title VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL,
    deleted_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX uk_schools_name ON schools(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_schools_code ON schools(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_schools_status ON schools(status);
CREATE INDEX idx_schools_license_end_at ON schools(license_end_at);
CREATE INDEX idx_schools_contact_phone ON schools(contact_phone);
COMMENT ON TABLE schools IS '学校租户主表，记录平台入驻学校的基础信息与授权状态。';
COMMENT ON COLUMN schools.status IS '学校状态：1待审核 2已激活 3缓冲期 4已冻结 5已注销 6已拒绝。';
COMMENT ON COLUMN schools.deleted_at IS '软删除时间，学校注销时设置。';

-- school_applications：学校入驻申请表，记录申请、审核与复提交流程。
CREATE TABLE school_applications (
    id BIGINT PRIMARY KEY,
    school_name VARCHAR(100) NOT NULL,
    school_code VARCHAR(50) NOT NULL,
    school_address VARCHAR(200) NULL,
    school_website VARCHAR(200) NULL,
    school_logo_url VARCHAR(500) NULL,
    contact_name VARCHAR(50) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    contact_email VARCHAR(100) NULL,
    contact_title VARCHAR(100) NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    reviewer_id BIGINT NULL,
    reviewed_at TIMESTAMP NULL,
    reject_reason VARCHAR(500) NULL,
    school_id BIGINT NULL,
    previous_application_id BIGINT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_school_applications_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_school_applications_previous_application_id FOREIGN KEY (previous_application_id) REFERENCES school_applications(id)
);
CREATE INDEX idx_school_applications_status ON school_applications(status);
CREATE INDEX idx_school_applications_contact_phone ON school_applications(contact_phone);
CREATE INDEX idx_school_applications_created_at ON school_applications(created_at);
COMMENT ON TABLE school_applications IS '学校入驻申请表，记录学校申请、审核与复提交流程。';
COMMENT ON COLUMN school_applications.status IS '申请状态：1待审核 2已通过 3已拒绝。';
COMMENT ON COLUMN school_applications.previous_application_id IS '重新申请时关联的上一次申请记录。';

-- school_sso_configs：学校级 SSO 配置表，保存学校接入参数与测试状态。
CREATE TABLE school_sso_configs (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    provider VARCHAR(20) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    is_tested BOOLEAN NOT NULL DEFAULT FALSE,
    config JSONB NOT NULL,
    tested_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by BIGINT NULL,
    CONSTRAINT fk_school_sso_configs_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE UNIQUE INDEX uk_school_sso_configs_school_id ON school_sso_configs(school_id);
COMMENT ON TABLE school_sso_configs IS '学校单点登录配置表，保存学校级 SSO 接入参数与测试状态。';
COMMENT ON COLUMN school_sso_configs.provider IS 'SSO 协议类型：cas / oauth2。';
COMMENT ON COLUMN school_sso_configs.config IS 'SSO 配置参数，敏感字段需要应用层加密存储。';

-- school_notifications：学校通知记录表，保存到期提醒、冻结通知等发送流水。
CREATE TABLE school_notifications (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    type SMALLINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    is_sent BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at TIMESTAMP NULL,
    target_phone VARCHAR(20) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_school_notifications_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE INDEX idx_school_notifications_school_id ON school_notifications(school_id);
CREATE INDEX idx_school_notifications_type ON school_notifications(type);
CREATE INDEX idx_school_notifications_is_sent ON school_notifications(is_sent);
COMMENT ON TABLE school_notifications IS '学校通知记录表，保存学校相关短信或提醒发送流水。';
COMMENT ON COLUMN school_notifications.type IS '通知类型：1到期提醒 2缓冲期通知 3冻结通知 4审核通过 5审核拒绝。';
COMMENT ON COLUMN school_notifications.is_sent IS '是否已经发送。';

-- 为模块01中的 school_id 字段补充跨模块外键约束。
ALTER TABLE users ADD CONSTRAINT fk_users_school_id FOREIGN KEY (school_id) REFERENCES schools(id);
ALTER TABLE user_sso_bindings ADD CONSTRAINT fk_user_sso_bindings_school_id FOREIGN KEY (school_id) REFERENCES schools(id);
