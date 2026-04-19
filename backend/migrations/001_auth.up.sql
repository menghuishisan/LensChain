-- 模块01 用户与认证
-- 文档依据：
-- 1. docs/modules/01-用户与认证/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 用户主表、扩展资料表、角色权限表
-- 2. SSO 绑定表、登录日志表、操作日志表
-- 不负责：
-- 1. 学校主表及学校级配置
-- 2. 课程、实验、竞赛、成绩、通知、系统管理相关表

-- roles：系统角色表，定义平台内置角色与学校可分配角色。
CREATE TABLE roles (
    id BIGINT PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(200) NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE roles IS '系统角色表，定义平台内置角色与学校可分配角色。';
COMMENT ON COLUMN roles.code IS '角色编码，如 super_admin、teacher。';
COMMENT ON COLUMN roles.is_system IS '是否系统预设角色，系统角色不可删除。';

-- permissions：权限点表，定义后端接口与功能操作的细粒度授权项。
CREATE TABLE permissions (
    id BIGINT PRIMARY KEY,
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    description VARCHAR(200) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_permissions_module ON permissions(module);
COMMENT ON TABLE permissions IS '权限点表，定义后端接口与功能操作的细粒度授权项。';
COMMENT ON COLUMN permissions.module IS '权限所属模块，如 auth、course。';

-- users：用户主表，记录登录凭证、基础身份和租户归属。
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    phone VARCHAR(20) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,
    school_id BIGINT NOT NULL,
    student_no VARCHAR(50) NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    is_first_login BOOLEAN NOT NULL DEFAULT TRUE,
    is_school_admin BOOLEAN NOT NULL DEFAULT FALSE,
    login_fail_count SMALLINT NOT NULL DEFAULT 0,
    locked_until TIMESTAMP NULL,
    token_valid_after TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP NULL,
    last_login_ip VARCHAR(45) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL,
    deleted_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX uk_users_phone ON users(phone) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_users_school_student_no ON users(school_id, student_no) WHERE deleted_at IS NULL AND student_no IS NOT NULL;
CREATE INDEX idx_users_school_id ON users(school_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_name ON users(name);
COMMENT ON TABLE users IS '用户主表，记录登录凭证、基础身份和租户归属。';
COMMENT ON COLUMN users.school_id IS '所属学校ID，在模块02迁移中补充外键约束。';
COMMENT ON COLUMN users.status IS '账号状态：1正常 2禁用 3归档。';
COMMENT ON COLUMN users.is_first_login IS '是否首次登录，首次登录需要强制修改密码。';
COMMENT ON COLUMN users.is_school_admin IS '是否兼任学校管理员。';
COMMENT ON COLUMN users.token_valid_after IS 'Token 生效时间基线，早于该时间签发的访问令牌全部失效。';

-- user_profiles：用户扩展资料表，保存头像、院系、班级等补充信息。
CREATE TABLE user_profiles (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    avatar_url VARCHAR(500) NULL,
    nickname VARCHAR(50) NULL,
    email VARCHAR(100) NULL,
    college VARCHAR(100) NULL,
    major VARCHAR(100) NULL,
    class_name VARCHAR(50) NULL,
    enrollment_year SMALLINT NULL,
    education_level SMALLINT NULL,
    grade SMALLINT NULL,
    remark TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_profiles_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX idx_user_profiles_college ON user_profiles(college);
CREATE INDEX idx_user_profiles_education_level ON user_profiles(education_level);
COMMENT ON TABLE user_profiles IS '用户扩展资料表，保存头像、院系、班级等补充信息。';
COMMENT ON COLUMN user_profiles.education_level IS '学业层次：1专科 2本科 3硕士 4博士。';

-- user_roles：用户角色关联表，建立用户与角色的多对多授权关系。
CREATE TABLE user_roles (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id)
);
CREATE UNIQUE INDEX uk_user_roles_user_role ON user_roles(user_id, role_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
COMMENT ON TABLE user_roles IS '用户角色关联表，建立用户与角色的多对多授权关系。';

-- role_permissions：角色权限关联表，建立角色与权限点的多对多关系。
CREATE TABLE role_permissions (
    id BIGINT PRIMARY KEY,
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_role_permissions_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT fk_role_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(id)
);
CREATE UNIQUE INDEX uk_role_permissions ON role_permissions(role_id, permission_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);
COMMENT ON TABLE role_permissions IS '角色权限关联表，建立角色与权限点的多对多关系。';

-- user_sso_bindings：用户 SSO 绑定表，记录平台账号与学校身份提供方账号的映射关系。
CREATE TABLE user_sso_bindings (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    sso_provider VARCHAR(20) NOT NULL,
    sso_user_id VARCHAR(100) NOT NULL,
    bound_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP NULL,
    CONSTRAINT fk_user_sso_bindings_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_sso_bindings_school_sso_user ON user_sso_bindings(school_id, sso_user_id);
CREATE INDEX idx_sso_bindings_user_id ON user_sso_bindings(user_id);
COMMENT ON TABLE user_sso_bindings IS '用户 SSO 绑定表，记录平台账号与学校身份提供方账号的映射关系。';
COMMENT ON COLUMN user_sso_bindings.school_id IS '所属学校ID，在模块02迁移中补充外键约束。';
COMMENT ON COLUMN user_sso_bindings.sso_provider IS 'SSO 协议类型：cas / oauth2。';

-- login_logs：登录日志表，记录登录成功、失败、登出和锁定等认证事件。
CREATE TABLE login_logs (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    action SMALLINT NOT NULL,
    login_method SMALLINT NULL,
    ip VARCHAR(45) NOT NULL,
    user_agent VARCHAR(500) NULL,
    fail_reason VARCHAR(200) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_login_logs_user_id ON login_logs(user_id);
CREATE INDEX idx_login_logs_created_at ON login_logs(created_at);
CREATE INDEX idx_login_logs_action ON login_logs(action);
COMMENT ON TABLE login_logs IS '登录日志表，记录登录成功、失败、登出和锁定等认证事件。';
COMMENT ON COLUMN login_logs.action IS '操作类型：1登录成功 2登录失败 3登出 4被踢下线 5账号锁定。';
COMMENT ON COLUMN login_logs.login_method IS '登录方式：1密码 2SSO-CAS 3SSO-OAuth2。';

-- operation_logs：操作日志表，记录用户管理相关关键操作，供审计和问题追踪使用。
CREATE TABLE operation_logs (
    id BIGINT PRIMARY KEY,
    operator_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id BIGINT NULL,
    detail JSONB NULL,
    ip VARCHAR(45) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_operation_logs_operator_id ON operation_logs(operator_id);
CREATE INDEX idx_operation_logs_action ON operation_logs(action);
CREATE INDEX idx_operation_logs_target ON operation_logs(target_type, target_id);
CREATE INDEX idx_operation_logs_created_at ON operation_logs(created_at);
COMMENT ON TABLE operation_logs IS '操作日志表，记录用户管理相关关键操作。';
COMMENT ON COLUMN operation_logs.operator_id IS '操作人ID。';
COMMENT ON COLUMN operation_logs.detail IS '操作详情，保存变更前后数据快照或批量导入结果。';
