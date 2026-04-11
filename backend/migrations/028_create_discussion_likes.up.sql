-- 028_create_discussion_likes.up.sql
-- 模块03 — 课程与教学：讨论点赞表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE discussion_likes (
    id            BIGINT    PRIMARY KEY,
    discussion_id BIGINT    NOT NULL,
    user_id       BIGINT    NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_discussion_likes ON discussion_likes(discussion_id, user_id);

COMMENT ON TABLE discussion_likes IS '讨论点赞表';
