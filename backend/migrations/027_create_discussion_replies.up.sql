-- 027_create_discussion_replies.up.sql
-- 模块03 — 课程与教学：讨论回复表
-- 对照 docs/modules/03-课程与教学/02-数据库设计.md

CREATE TABLE discussion_replies (
    id            BIGINT    PRIMARY KEY,
    discussion_id BIGINT    NOT NULL,
    author_id     BIGINT    NOT NULL,
    content       TEXT      NOT NULL,
    reply_to_id   BIGINT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMP
);

CREATE INDEX idx_discussion_replies_discussion_id ON discussion_replies(discussion_id);

COMMENT ON TABLE discussion_replies IS '讨论回复表';
COMMENT ON COLUMN discussion_replies.reply_to_id IS '回复某条回复的ID（楼中楼）';
