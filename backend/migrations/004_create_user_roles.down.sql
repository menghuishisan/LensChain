-- 004_create_user_roles.down.sql
-- 回滚：删除 user_roles 用户角色关联表

DROP TABLE IF EXISTS user_roles;
