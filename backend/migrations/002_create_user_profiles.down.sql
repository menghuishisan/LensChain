-- 002_create_user_profiles.down.sql
-- 回滚：删除 user_profiles 用户扩展信息表

DROP TABLE IF EXISTS user_profiles;
