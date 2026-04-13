-- 033_add_users_token_valid_after.down.sql
-- 模块01 — 用户与认证：回滚用户Token生效时间基线字段
-- 删除 users.token_valid_after 字段

ALTER TABLE users
DROP COLUMN IF EXISTS token_valid_after;
