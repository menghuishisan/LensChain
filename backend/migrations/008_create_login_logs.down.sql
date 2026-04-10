-- 008_create_login_logs.down.sql
-- 回滚：删除 login_logs 登录日志表

DROP TABLE IF EXISTS login_logs;
