-- 006_create_role_permissions.down.sql
-- 回滚：删除 role_permissions 角色权限关联表

DROP TABLE IF EXISTS role_permissions;
