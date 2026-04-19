-- 模块08 系统管理与监控
-- 文档依据：
-- 1. docs/modules/08-系统管理与监控/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 系统配置与配置变更日志
-- 2. 告警规则、告警事件、平台统计、备份记录
-- 不负责：
-- 1. 外部模块日志写入
-- 2. 监控采集执行器本身

-- system_configs：系统配置表。
CREATE TABLE system_configs (
    id BIGINT PRIMARY KEY,
    config_group VARCHAR(50) NOT NULL,
    config_key VARCHAR(100) NOT NULL,
    config_value TEXT NOT NULL,
    value_type SMALLINT NOT NULL DEFAULT 1,
    description VARCHAR(200) NULL,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by BIGINT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_system_configs_updated_by FOREIGN KEY (updated_by) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_system_configs_group_key ON system_configs(config_group, config_key);
CREATE INDEX idx_system_configs_group ON system_configs(config_group);

-- config_change_logs：配置变更记录表。
CREATE TABLE config_change_logs (
    id BIGINT PRIMARY KEY,
    config_group VARCHAR(50) NOT NULL,
    config_key VARCHAR(100) NOT NULL,
    old_value TEXT NULL,
    new_value TEXT NOT NULL,
    changed_by BIGINT NOT NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip VARCHAR(45) NOT NULL,
    CONSTRAINT fk_config_change_logs_changed_by FOREIGN KEY (changed_by) REFERENCES users(id)
);
CREATE INDEX idx_config_change_logs_group_key ON config_change_logs(config_group, config_key);
CREATE INDEX idx_config_change_logs_changed_by ON config_change_logs(changed_by);
CREATE INDEX idx_config_change_logs_changed_at ON config_change_logs(changed_at);

-- alert_rules：告警规则表。
CREATE TABLE alert_rules (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NULL,
    alert_type SMALLINT NOT NULL,
    level SMALLINT NOT NULL DEFAULT 2,
    condition JSONB NOT NULL,
    silence_period INT NOT NULL DEFAULT 1800,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_alert_rules_created_by FOREIGN KEY (created_by) REFERENCES users(id)
);
CREATE INDEX idx_alert_rules_alert_type ON alert_rules(alert_type);
CREATE INDEX idx_alert_rules_is_enabled ON alert_rules(is_enabled);

-- alert_events：告警事件表。
CREATE TABLE alert_events (
    id BIGINT PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    level SMALLINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    detail JSONB NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    handled_by BIGINT NULL,
    handled_at TIMESTAMP NULL,
    handle_note TEXT NULL,
    triggered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_alert_events_rule_id FOREIGN KEY (rule_id) REFERENCES alert_rules(id),
    CONSTRAINT fk_alert_events_handled_by FOREIGN KEY (handled_by) REFERENCES users(id)
);
CREATE INDEX idx_alert_events_rule_id ON alert_events(rule_id);
CREATE INDEX idx_alert_events_level ON alert_events(level);
CREATE INDEX idx_alert_events_status ON alert_events(status);
CREATE INDEX idx_alert_events_triggered_at ON alert_events(triggered_at);

-- platform_statistics：平台统计日表。
CREATE TABLE platform_statistics (
    id BIGINT PRIMARY KEY,
    stat_date DATE NOT NULL,
    active_users INT NOT NULL DEFAULT 0,
    new_users INT NOT NULL DEFAULT 0,
    total_users INT NOT NULL DEFAULT 0,
    total_schools INT NOT NULL DEFAULT 0,
    total_courses INT NOT NULL DEFAULT 0,
    active_courses INT NOT NULL DEFAULT 0,
    total_experiments INT NOT NULL DEFAULT 0,
    total_competitions INT NOT NULL DEFAULT 0,
    active_competitions INT NOT NULL DEFAULT 0,
    storage_used_gb DECIMAL(10,2) NOT NULL DEFAULT 0,
    api_request_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_platform_statistics_date ON platform_statistics(stat_date);

-- backup_records：备份记录表。
CREATE TABLE backup_records (
    id BIGINT PRIMARY KEY,
    backup_type SMALLINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    file_path VARCHAR(500) NULL,
    file_size BIGINT NULL,
    database_name VARCHAR(100) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    error_message TEXT NULL,
    triggered_by BIGINT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_backup_records_triggered_by FOREIGN KEY (triggered_by) REFERENCES users(id)
);
CREATE INDEX idx_backup_records_status ON backup_records(status);
CREATE INDEX idx_backup_records_started_at ON backup_records(started_at);
