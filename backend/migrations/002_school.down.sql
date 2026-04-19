-- 模块02 学校与租户管理回滚
ALTER TABLE IF EXISTS user_sso_bindings DROP CONSTRAINT IF EXISTS fk_user_sso_bindings_school_id;
ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_users_school_id;
DROP TABLE IF EXISTS school_notifications;
DROP TABLE IF EXISTS school_sso_configs;
DROP TABLE IF EXISTS school_applications;
DROP TABLE IF EXISTS schools;
