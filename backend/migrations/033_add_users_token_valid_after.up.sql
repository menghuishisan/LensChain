-- 033_add_users_token_valid_after.up.sql
-- 模块01 — 用户与认证：新增用户Token生效时间基线字段
-- 用于单设备登录踢下线、账号禁用/归档、学校冻结/注销等场景的统一Access Token失效校验

ALTER TABLE users
ADD COLUMN IF NOT EXISTS token_valid_after TIMESTAMP NOT NULL DEFAULT NOW();
