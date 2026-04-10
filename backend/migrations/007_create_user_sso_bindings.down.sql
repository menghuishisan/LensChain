-- 007_create_user_sso_bindings.down.sql
-- 回滚：删除 user_sso_bindings SSO绑定记录表

DROP TABLE IF EXISTS user_sso_bindings;
