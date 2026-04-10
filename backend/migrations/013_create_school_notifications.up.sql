-- 013_create_school_notifications.up.sql
-- 模块02 — 学校与租户管理：学校通知记录表
-- 对照 docs/modules/02-学校与租户管理/02-数据库设计.md

CREATE TABLE school_notifications (
    id           BIGINT       PRIMARY KEY,
    school_id    BIGINT       NOT NULL,
    type         SMALLINT     NOT NULL,
    title        VARCHAR(200) NOT NULL,
    content      TEXT         NOT NULL,
    is_sent      BOOLEAN      NOT NULL DEFAULT FALSE,
    sent_at      TIMESTAMP,
    target_phone VARCHAR(20),
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_school_notifications_school_id ON school_notifications(school_id);
CREATE INDEX idx_school_notifications_type ON school_notifications(type);
CREATE INDEX idx_school_notifications_is_sent ON school_notifications(is_sent);

COMMENT ON TABLE school_notifications IS '学校通知记录表';
COMMENT ON COLUMN school_notifications.type IS '通知类型：1到期提醒 2缓冲期通知 3冻结通知 4审核通过 5审核拒绝';
