-- 模块07 通知与消息
-- 文档依据：
-- 1. docs/modules/07-通知与消息/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 站内信、系统公告、模板、偏好
-- 2. 公告阅读状态与通知数据落库
-- 不负责：
-- 1. 各业务模块的通知触发逻辑
-- 2. WebSocket 与 Redis 推送层

-- notifications：站内信主表。
CREATE TABLE notifications (
    id BIGINT PRIMARY KEY,
    receiver_id BIGINT NOT NULL,
    school_id BIGINT NULL,
    category SMALLINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    source_module VARCHAR(20) NOT NULL,
    source_id BIGINT NULL,
    source_type VARCHAR(50) NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMP NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_notifications_receiver_id FOREIGN KEY (receiver_id) REFERENCES users(id),
    CONSTRAINT fk_notifications_school_id FOREIGN KEY (school_id) REFERENCES schools(id)
);
CREATE INDEX idx_notifications_receiver_id ON notifications(receiver_id);
CREATE INDEX idx_notifications_receiver_read ON notifications(receiver_id, is_read);
CREATE INDEX idx_notifications_receiver_category ON notifications(receiver_id, category);
CREATE INDEX idx_notifications_school_id ON notifications(school_id);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
CREATE INDEX idx_notifications_event_type ON notifications(event_type);

-- system_announcements：系统公告表。
CREATE TABLE system_announcements (
    id BIGINT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    published_by BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    is_pinned BOOLEAN NOT NULL DEFAULT TRUE,
    published_at TIMESTAMP NULL,
    scheduled_at TIMESTAMP NULL,
    unpublished_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_system_announcements_published_by FOREIGN KEY (published_by) REFERENCES users(id)
);
CREATE INDEX idx_system_announcements_status ON system_announcements(status);
CREATE INDEX idx_system_announcements_published_at ON system_announcements(published_at);

-- notification_templates：消息模板表。
CREATE TABLE notification_templates (
    id BIGINT PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    category SMALLINT NOT NULL,
    title_template VARCHAR(200) NOT NULL,
    content_template TEXT NOT NULL,
    variables JSONB NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_notification_templates_event_type ON notification_templates(event_type);

-- user_notification_preferences：用户通知偏好表。
CREATE TABLE user_notification_preferences (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    category SMALLINT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_notification_preferences_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_user_notification_prefs ON user_notification_preferences(user_id, category);

-- announcement_read_status：公告阅读状态表。
CREATE TABLE announcement_read_status (
    id BIGINT PRIMARY KEY,
    announcement_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    read_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_announcement_read_status_announcement_id FOREIGN KEY (announcement_id) REFERENCES system_announcements(id),
    CONSTRAINT fk_announcement_read_status_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_announcement_read ON announcement_read_status(announcement_id, user_id);
CREATE INDEX idx_announcement_read_user ON announcement_read_status(user_id);
