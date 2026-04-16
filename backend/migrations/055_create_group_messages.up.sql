-- 055_create_group_messages.up.sql
-- 模块04 — 实验环境：组内消息表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.22节

CREATE TABLE group_messages (
    id           BIGINT    PRIMARY KEY,
    group_id     BIGINT    NOT NULL REFERENCES experiment_groups(id),
    sender_id    BIGINT    NOT NULL,
    content      TEXT      NOT NULL,
    message_type SMALLINT  NOT NULL DEFAULT 1,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_group_messages_group_id ON group_messages(group_id);
CREATE INDEX idx_group_messages_created_at ON group_messages(created_at);
