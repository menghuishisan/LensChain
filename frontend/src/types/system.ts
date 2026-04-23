// system.ts
// 模块08系统管理与监控类型定义：统一审计、全局配置、告警、仪表盘、统计与数据备份。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/**
 * JSON 原始值。
 */
export type SystemJSONPrimitive = string | number | boolean | null;

/**
 * JSON 值。
 */
export type SystemJSONValue = SystemJSONPrimitive | SystemJSONObject | SystemJSONValue[];

/**
 * JSON 对象。
 */
export interface SystemJSONObject {
  [key: string]: SystemJSONValue;
}

/**
 * 审计日志来源：登录、操作、实验。
 */
export type SystemAuditSource = "login" | "operation" | "experiment";

/**
 * 审计日志导出格式：Excel 或 CSV。
 */
export type SystemAuditExportFormat = "excel" | "csv";

/**
 * 统一审计查询参数。
 */
export interface SystemAuditListParams extends QueryParams {
  page?: number;
  page_size?: number;
  source?: SystemAuditSource;
  keyword?: string;
  operator_id?: ID;
  action?: string;
  date_from?: string;
  date_to?: string;
  ip?: string;
}

/**
 * 审计日志导出参数。
 */
export interface SystemAuditExportParams extends SystemAuditListParams {
  format?: SystemAuditExportFormat;
}

/**
 * 审计日志目标对象。
 */
export interface SystemAuditLogTarget {
  type: string;
  id: ID | null;
}

/**
 * 审计来源统计。
 */
export interface SystemAuditSourceCounts {
  login: number;
  operation: number;
  experiment: number;
}

/**
 * 审计日志列表项。
 */
export interface SystemAuditLogItem {
  id: ID;
  source: SystemAuditSource;
  source_text: string;
  operator_id: ID | null;
  operator_name: string | null;
  action: string;
  action_text: string;
  target: SystemAuditLogTarget | null;
  detail: SystemJSONValue;
  ip: string | null;
  user_agent: string | null;
  created_at: string;
}

/**
 * 审计日志列表响应。
 */
export interface SystemAuditLogListResponse extends PaginatedData<SystemAuditLogItem> {
  source_counts: SystemAuditSourceCounts;
}

/**
 * 系统配置值类型：1字符串 2数字 3布尔 4JSON。
 */
export type SystemConfigValueType = 1 | 2 | 3 | 4;

/**
 * 系统配置分组：平台、存储、安全、备份。
 */
export type SystemConfigGroup = "platform" | "storage" | "security" | "backup";

/**
 * 系统配置项。
 */
export interface SystemConfigItem {
  key: string;
  value: string;
  value_type: SystemConfigValueType;
  description: string;
  is_sensitive: boolean;
  updated_at: string;
}

/**
 * 系统配置分组项。
 */
export interface SystemConfigGroupItem {
  group: SystemConfigGroup;
  group_text: string;
  configs: SystemConfigItem[];
}

/**
 * 系统配置列表响应。
 */
export interface SystemConfigListResponse {
  groups: SystemConfigGroupItem[];
}

/**
 * 单个配置分组响应。
 */
export interface SystemConfigGroupResponse {
  group: SystemConfigGroupItem;
}

/**
 * 单个配置更新请求。
 */
export interface UpdateSystemConfigRequest {
  value: string;
  updated_at: string;
}

/**
 * 批量配置更新项。
 */
export interface BatchUpdateSystemConfigItem {
  key: string;
  value: string;
  updated_at: string;
}

/**
 * 批量配置更新请求。
 */
export interface BatchUpdateSystemConfigsRequest {
  configs: BatchUpdateSystemConfigItem[];
}

/**
 * 配置变更记录查询参数。
 */
export interface SystemConfigChangeLogListParams extends QueryParams {
  page?: number;
  page_size?: number;
  config_group?: SystemConfigGroup;
  config_key?: string;
  date_from?: string;
  date_to?: string;
}

/**
 * 配置变更记录项。
 */
export interface SystemConfigChangeLogItem {
  id: ID;
  config_group: SystemConfigGroup;
  config_key: string;
  old_value: string | null;
  new_value: string;
  changed_by: ID;
  changed_by_name: string;
  changed_at: string;
  ip: string;
}

/**
 * 配置变更记录响应。
 */
export type SystemConfigChangeLogListResponse = PaginatedData<SystemConfigChangeLogItem>;

/**
 * 告警类型：1阈值告警 2事件告警 3服务状态告警。
 */
export type SystemAlertType = 1 | 2 | 3;

/**
 * 告警级别：1信息 2警告 3严重 4紧急。
 */
export type SystemAlertLevel = 1 | 2 | 3 | 4;

/**
 * 告警事件状态：1待处理 2已处理 3已忽略。
 */
export type SystemAlertEventStatus = 1 | 2 | 3;

/**
 * 阈值告警指标。
 */
export type SystemThresholdMetric = "cpu_usage" | "memory_usage" | "storage_usage";

/**
 * 阈值告警运算符。
 */
export type SystemThresholdOperator = ">" | ">=" | "<" | "<=" | "==";

/**
 * 事件告警来源。
 */
export type SystemEventSource = "login_logs";

/**
 * 事件告警分组字段。
 */
export type SystemEventGroupBy = "ip";

/**
 * 服务状态告警服务名。
 */
export type SystemServiceName = "postgresql" | "redis" | "nats" | "minio" | "api_server" | "k8s_cluster";

/**
 * 事件告警过滤值。
 */
export type SystemAlertEventFilterValue = string | number | boolean | null;

/**
 * 告警规则条件。
 * 对齐后端 DTO，未使用字段保持为空。
 */
export interface SystemAlertRuleCondition {
  metric?: SystemThresholdMetric;
  operator?: SystemThresholdOperator;
  value?: number;
  duration?: number;
  event_source?: SystemEventSource;
  event_filter?: Record<string, SystemAlertEventFilterValue>;
  group_by?: SystemEventGroupBy;
  count_threshold?: number;
  time_window?: number;
  service_name?: SystemServiceName;
  check_url?: string;
  check_interval?: number;
  fail_threshold?: number;
}

/**
 * 创建告警规则请求。
 */
export interface CreateSystemAlertRuleRequest {
  name: string;
  description?: string | null;
  alert_type: SystemAlertType;
  level: SystemAlertLevel;
  condition: SystemAlertRuleCondition;
  silence_period?: number;
}

/**
 * 更新告警规则请求。
 */
export interface UpdateSystemAlertRuleRequest {
  name?: string;
  description?: string | null;
  alert_type?: SystemAlertType;
  level?: SystemAlertLevel;
  condition?: SystemAlertRuleCondition;
  silence_period?: number;
}

/**
 * 告警规则启用/禁用请求。
 */
export interface ToggleSystemAlertRuleRequest {
  is_enabled: boolean;
}

/**
 * 告警规则列表查询参数。
 */
export interface SystemAlertRuleListParams extends QueryParams {
  page?: number;
  page_size?: number;
  alert_type?: SystemAlertType;
  level?: SystemAlertLevel;
  is_enabled?: boolean;
}

/**
 * 告警规则列表项。
 */
export interface SystemAlertRuleItem {
  id: ID;
  name: string;
  description: string | null;
  alert_type: SystemAlertType;
  alert_type_text: string;
  level: SystemAlertLevel;
  level_text: string;
  condition: SystemAlertRuleCondition;
  silence_period: number;
  is_enabled: boolean;
  created_at?: string | null;
}

/**
 * 告警规则详情。
 */
export type SystemAlertRuleDetail = SystemAlertRuleItem;

/**
 * 告警规则列表响应。
 */
export type SystemAlertRuleListResponse = PaginatedData<SystemAlertRuleItem>;

/**
 * 告警事件样本日志。
 */
export interface SystemAlertEventSampleEvent {
  user_id?: ID | null;
  fail_reason?: string | null;
  created_at: string;
}

/**
 * 告警事件详情结构。
 */
export interface SystemAlertEventDetailData {
  metric?: SystemThresholdMetric;
  current_value?: number;
  threshold?: number;
  duration_seconds?: number;
  node?: string;
  event_source?: "login_logs" | "service_health";
  group_value?: string;
  event_count?: number;
  time_window?: number;
  sample_events?: SystemAlertEventSampleEvent[];
}

/**
 * 告警事件列表项。
 */
export interface SystemAlertEventItem {
  id: ID;
  rule_id: ID;
  rule_name: string;
  level: SystemAlertLevel;
  level_text: string;
  title: string;
  detail: SystemAlertEventDetailData;
  status: SystemAlertEventStatus;
  status_text: string;
  triggered_at: string;
}

/**
 * 告警事件状态统计。
 */
export interface SystemAlertEventStatusCounts {
  pending: number;
  handled: number;
  ignored: number;
}

/**
 * 告警事件列表查询参数。
 */
export interface SystemAlertEventListParams extends QueryParams {
  page?: number;
  page_size?: number;
  rule_id?: ID;
  level?: SystemAlertLevel;
  status?: SystemAlertEventStatus;
  date_from?: string;
  date_to?: string;
}

/**
 * 告警事件列表响应。
 */
export interface SystemAlertEventListResponse extends PaginatedData<SystemAlertEventItem> {
  status_counts: SystemAlertEventStatusCounts;
}

/**
 * 告警事件详情。
 */
export interface SystemAlertEventDetail {
  id: ID;
  rule_id: ID;
  rule_name: string;
  level: SystemAlertLevel;
  level_text: string;
  title: string;
  detail: SystemAlertEventDetailData;
  status: SystemAlertEventStatus;
  status_text: string;
  handled_by?: ID | null;
  handled_by_name?: string | null;
  handled_at?: string | null;
  handle_note?: string | null;
  triggered_at: string;
}

/**
 * 处理或忽略告警请求。
 */
export interface HandleSystemAlertEventRequest {
  handle_note: string;
}

/**
 * 仪表盘整体健康状态。
 */
export type SystemDashboardStatus = "healthy" | "unhealthy" | "unknown";

/**
 * 服务连接摘要。
 */
export interface SystemDashboardServiceConnections {
  active: number;
  max: number;
}

/**
 * 仪表盘服务健康项。
 */
export interface SystemDashboardServiceHealth {
  name: string;
  status: SystemDashboardStatus;
  latency_ms: number;
  uptime?: string;
  connections?: SystemDashboardServiceConnections;
  memory_used_mb?: number;
  messages_in_queue?: number;
  storage_used_gb?: number;
  nodes?: number;
  pods_running?: number;
}

/**
 * 仪表盘健康响应。
 */
export interface SystemDashboardHealthResponse {
  overall_status: SystemDashboardStatus;
  services: SystemDashboardServiceHealth[];
  last_check_at: string;
}

/**
 * CPU 资源使用情况。
 */
export interface SystemDashboardCPUResource {
  usage_percent: number;
  cores_total: number;
  cores_used: number;
}

/**
 * 内存资源使用情况。
 */
export interface SystemDashboardMemoryResource {
  usage_percent: number;
  total_gb: number;
  used_gb: number;
}

/**
 * 存储资源使用情况。
 */
export interface SystemDashboardStorageResource {
  usage_percent: number;
  total_gb: number;
  used_gb: number;
}

/**
 * K8s 资源使用情况。
 */
export interface SystemDashboardK8sResource {
  nodes: number;
  pods_total: number;
  pods_running: number;
  pods_pending: number;
  namespaces: number;
}

/**
 * 仪表盘资源使用响应。
 */
export interface SystemDashboardResourcesResponse {
  cpu: SystemDashboardCPUResource;
  memory: SystemDashboardMemoryResource;
  storage: SystemDashboardStorageResource;
  k8s: SystemDashboardK8sResource;
}

/**
 * 仪表盘最近告警项。
 */
export interface SystemDashboardRecentAlert {
  id: ID;
  title: string;
  level: SystemAlertLevel;
  triggered_at: string;
}

/**
 * 仪表盘实时指标响应。
 */
export interface SystemDashboardRealtimeResponse {
  online_users: number;
  active_experiments: number;
  active_competitions: number;
  api_requests_per_minute: number;
  pending_alerts: number;
  recent_alerts: SystemDashboardRecentAlert[];
}

/**
 * 统计趋势指标。
 */
export type SystemStatisticsMetric = "active_users" | "new_users" | "experiments" | "api_requests";

/**
 * 统计趋势周期。
 */
export type SystemStatisticsPeriod = "7d" | "30d" | "90d" | "365d";

/**
 * 平台统计今日摘要。
 */
export interface SystemStatisticsTodaySummary {
  active_users: number;
  new_users: number;
  experiments_started: number;
  api_requests: number;
}

/**
 * 平台统计总览响应。
 */
export interface SystemStatisticsOverviewResponse {
  total_users: number;
  total_schools: number;
  total_courses: number;
  total_experiments: number;
  total_competitions: number;
  today: SystemStatisticsTodaySummary;
}

/**
 * 平台趋势查询参数。
 */
export interface SystemStatisticsTrendParams extends QueryParams {
  metric: SystemStatisticsMetric;
  period?: SystemStatisticsPeriod;
}

/**
 * 平台趋势点位。
 */
export interface SystemStatisticsTrendPoint {
  date: string;
  value: number;
}

/**
 * 平台趋势响应。
 */
export interface SystemStatisticsTrendResponse {
  metric: SystemStatisticsMetric;
  period: SystemStatisticsPeriod;
  data_points: SystemStatisticsTrendPoint[];
}

/**
 * 学校活跃度排行项。
 */
export interface SystemSchoolActivityRankingItem {
  rank: number;
  school_id: ID;
  school_name: string;
  active_users: number;
  total_users: number;
  activity_score: number;
}

/**
 * 学校活跃度排行响应。
 */
export interface SystemSchoolActivityRankingResponse {
  list: SystemSchoolActivityRankingItem[];
}

/**
 * 备份类型：1自动 2手动。
 */
export type SystemBackupType = 1 | 2;

/**
 * 备份状态：1进行中 2成功 3失败。
 */
export type SystemBackupStatus = 1 | 2 | 3;

/**
 * 手动触发备份响应。
 */
export interface TriggerSystemBackupResponse {
  id: ID;
  backup_type: SystemBackupType;
  status: SystemBackupStatus;
  status_text: string;
  started_at: string;
}

/**
 * 备份列表查询参数。
 */
export interface SystemBackupListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: SystemBackupStatus;
}

/**
 * 备份列表项。
 */
export interface SystemBackupItem {
  id: ID;
  backup_type: SystemBackupType;
  backup_type_text: string;
  status: SystemBackupStatus;
  status_text: string;
  database_name: string;
  file_size: number | null;
  file_size_text: string | null;
  started_at: string;
  completed_at: string | null;
  duration_seconds: number | null;
  error_message?: string | null;
}

/**
 * 备份配置响应。
 */
export interface SystemBackupConfig {
  auto_enabled: boolean;
  cron: string;
  retention_count: number;
  auto_enabled_updated_at: string;
  cron_updated_at: string;
  retention_count_updated_at: string;
}

/**
 * 备份列表响应。
 */
export interface SystemBackupListResponse extends PaginatedData<SystemBackupItem> {
  backup_config: SystemBackupConfig;
}

/**
 * 更新备份配置请求。
 */
export interface UpdateSystemBackupConfigRequest {
  auto_enabled: boolean;
  cron: string;
  retention_count: number;
  auto_enabled_updated_at: string;
  cron_updated_at: string;
  retention_count_updated_at: string;
}
