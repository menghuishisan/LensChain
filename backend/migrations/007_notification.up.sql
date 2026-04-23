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

-- notification_templates：系统预置模板数据。
INSERT INTO notification_templates (id, event_type, category, title_template, content_template, variables, is_enabled)
VALUES
    (700000000000000001, 'assignment.published', 2, '新作业发布', '课程《{course_name}》发布了新作业《{assignment_name}》，截止时间：{deadline}。', '[{"name":"course_name","description":"课程名称","required":true},{"name":"assignment_name","description":"作业名称","required":true},{"name":"deadline","description":"截止时间","required":false}]'::jsonb, TRUE),
    (700000000000000002, 'assignment.deadline_reminder', 2, '作业截止提醒', '作业《{assignment_name}》将于{hours}小时后截止，请尽快提交。', '[{"name":"assignment_name","description":"作业名称","required":true},{"name":"hours","description":"剩余小时数","required":true}]'::jsonb, TRUE),
    (700000000000000003, 'assignment.graded', 2, '作业批改完成', '您的作业《{assignment_name}》已批改，得分：{score}。', '[{"name":"assignment_name","description":"作业名称","required":true},{"name":"score","description":"分数","required":true}]'::jsonb, TRUE),
    (700000000000000004, 'experiment.published', 3, '新实验发布', '课程《{course_name}》发布了新实验《{experiment_name}》。', '[{"name":"course_name","description":"课程名称","required":true},{"name":"experiment_name","description":"实验名称","required":true}]'::jsonb, TRUE),
    (700000000000000005, 'experiment.expiring', 3, '实验环境即将超时', '您的实验环境将于{minutes}分钟后到期，请及时保存进度。', '[{"name":"minutes","description":"剩余分钟数","required":true}]'::jsonb, TRUE),
    (700000000000000006, 'experiment.graded', 3, '实验评分完成', '实验《{experiment_name}》已评分，得分：{score}。', '[{"name":"experiment_name","description":"实验名称","required":true},{"name":"score","description":"分数","required":true}]'::jsonb, TRUE),
    (700000000000000007, 'competition.published', 4, '新竞赛发布', '新竞赛《{competition_name}》开始报名，报名截止：{deadline}。', '[{"name":"competition_name","description":"竞赛名称","required":true},{"name":"deadline","description":"报名截止时间","required":false}]'::jsonb, TRUE),
    (700000000000000008, 'competition.registration_reminder', 4, '竞赛报名提醒', '竞赛《{competition_name}》将于{deadline}截止报名，请及时组队报名。', '[{"name":"competition_name","description":"竞赛名称","required":true},{"name":"deadline","description":"报名截止时间","required":true}]'::jsonb, TRUE),
    (700000000000000009, 'competition.starting_reminder', 4, '竞赛即将开始', '竞赛《{competition_name}》将于{start_time}开始，请准时参赛。', '[{"name":"competition_name","description":"竞赛名称","required":true},{"name":"start_time","description":"开始时间","required":true}]'::jsonb, TRUE),
    (700000000000000010, 'grade.review_approved', 5, '成绩已发布', '课程《{course_name}》{semester_name}成绩已发布，请查看。', '[{"name":"course_name","description":"课程名称","required":true},{"name":"semester_name","description":"学期名称","required":false}]'::jsonb, TRUE),
    (700000000000000011, 'grade.review_rejected', 5, '成绩审核已驳回', '课程《{course_name}》成绩审核已驳回，原因：{reason}', '[{"name":"course_name","description":"课程名称","required":true},{"name":"reason","description":"驳回原因","required":false}]'::jsonb, TRUE),
    (700000000000000012, 'grade.appeal_handled', 5, '成绩申诉处理完成', '课程《{course_name}》的成绩申诉已处理：{result}。', '[{"name":"course_name","description":"课程名称","required":true},{"name":"result","description":"处理结果","required":true}]'::jsonb, TRUE),
    (700000000000000013, 'grade.academic_warning', 5, '学业预警', '您的GPA({gpa})低于预警线({threshold})，请关注学业表现。', '[{"name":"gpa","description":"当前GPA","required":true},{"name":"threshold","description":"预警阈值","required":true}]'::jsonb, TRUE),
    (700000000000000014, 'system.maintenance', 1, '系统维护通知', '平台将于{time}进行维护，预计持续{duration}，届时将暂停服务。', '[{"name":"time","description":"维护时间","required":true},{"name":"duration","description":"维护时长","required":false}]'::jsonb, TRUE),
    (700000000000000015, 'system.alert.triggered', 1, '系统告警：{rule_name}', '告警级别：{level_text}；告警规则：{rule_name}；告警标题：{title}；触发详情：{detail}', '[{"name":"level_text","description":"告警级别","required":true},{"name":"rule_name","description":"告警规则名称","required":true},{"name":"title","description":"告警标题","required":true},{"name":"detail","description":"告警详情","required":true}]'::jsonb, TRUE);
