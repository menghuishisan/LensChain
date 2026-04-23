// system.ts
// 模块08系统管理与监控 service：统一审计、全局配置、告警、运维仪表盘、平台统计与数据备份接口。

import { apiClient } from "@/lib/api-client";
import type { ID } from "@/types/api";
import type {
  BatchUpdateSystemConfigsRequest,
  CreateSystemAlertRuleRequest,
  HandleSystemAlertEventRequest,
  SystemAlertEventDetail,
  SystemAlertEventListParams,
  SystemAlertEventListResponse,
  SystemAlertRuleDetail,
  SystemAlertRuleListParams,
  SystemAlertRuleListResponse,
  SystemAuditExportParams,
  SystemAuditLogListResponse,
  SystemAuditListParams,
  SystemBackupConfig,
  SystemBackupListParams,
  SystemBackupListResponse,
  SystemConfigChangeLogListParams,
  SystemConfigChangeLogListResponse,
  SystemConfigGroup,
  SystemConfigGroupResponse,
  SystemConfigListResponse,
  SystemDashboardHealthResponse,
  SystemDashboardRealtimeResponse,
  SystemDashboardResourcesResponse,
  SystemSchoolActivityRankingResponse,
  SystemStatisticsOverviewResponse,
  SystemStatisticsTrendParams,
  SystemStatisticsTrendResponse,
  ToggleSystemAlertRuleRequest,
  TriggerSystemBackupResponse,
  UpdateSystemAlertRuleRequest,
  UpdateSystemBackupConfigRequest,
  UpdateSystemConfigRequest,
} from "@/types/system";

/**
 * listAuditLogs 对应 GET /api/v1/system/audit/logs，用于统一审计日志分页查询。
 */
export function listAuditLogs(params: SystemAuditListParams) {
  return apiClient.get<SystemAuditLogListResponse>("/system/audit/logs", { query: params });
}

/**
 * exportAuditLogs 对应 GET /api/v1/system/audit/logs/export，用于按当前筛选条件导出审计日志。
 */
export function exportAuditLogs(params: SystemAuditExportParams) {
  return apiClient.download("/system/audit/logs/export", { query: params });
}

/**
 * getSystemConfigs 对应 GET /api/v1/system/configs，用于获取全部配置分组。
 */
export function getSystemConfigs() {
  return apiClient.get<SystemConfigListResponse>("/system/configs", { query: {} });
}

/**
 * getConfigGroup 对应 GET /api/v1/system/configs/:group，用于读取单个配置分组。
 */
export function getConfigGroup(group: SystemConfigGroup) {
  return apiClient.get<SystemConfigGroupResponse>(`/system/configs/${group}`, { query: {} });
}

/**
 * updateSystemConfig 对应 PUT /api/v1/system/configs/:group/:key，用于更新单个配置项。
 */
export function updateSystemConfig(group: SystemConfigGroup, key: string, payload: UpdateSystemConfigRequest) {
  return apiClient.put<null>(`/system/configs/${group}/${key}`, payload);
}

/**
 * batchUpdateSystemConfigs 对应 PUT /api/v1/system/configs/:group，用于批量更新同一分组配置。
 */
export function batchUpdateSystemConfigs(group: SystemConfigGroup, payload: BatchUpdateSystemConfigsRequest) {
  return apiClient.put<null>(`/system/configs/${group}`, payload);
}

/**
 * listConfigChangeLogs 对应 GET /api/v1/system/configs/change-logs，用于查询配置变更历史。
 */
export function listConfigChangeLogs(params: SystemConfigChangeLogListParams) {
  return apiClient.get<SystemConfigChangeLogListResponse>("/system/configs/change-logs", { query: params });
}

/**
 * createAlertRule 对应 POST /api/v1/system/alert-rules，用于创建告警规则。
 */
export function createAlertRule(payload: CreateSystemAlertRuleRequest) {
  return apiClient.post<SystemAlertRuleDetail>("/system/alert-rules", payload);
}

/**
 * listAlertRules 对应 GET /api/v1/system/alert-rules，用于获取告警规则列表。
 */
export function listAlertRules(params: SystemAlertRuleListParams) {
  return apiClient.get<SystemAlertRuleListResponse>("/system/alert-rules", { query: params });
}

/**
 * getAlertRule 对应 GET /api/v1/system/alert-rules/:id，用于获取告警规则详情。
 */
export function getAlertRule(ruleID: ID) {
  return apiClient.get<SystemAlertRuleDetail>(`/system/alert-rules/${ruleID}`, { query: {} });
}

/**
 * updateAlertRule 对应 PUT /api/v1/system/alert-rules/:id，用于更新告警规则。
 */
export function updateAlertRule(ruleID: ID, payload: UpdateSystemAlertRuleRequest) {
  return apiClient.put<null>(`/system/alert-rules/${ruleID}`, payload);
}

/**
 * toggleAlertRule 对应 PATCH /api/v1/system/alert-rules/:id/toggle，用于启用或禁用规则。
 */
export function toggleAlertRule(ruleID: ID, payload: ToggleSystemAlertRuleRequest) {
  return apiClient.patch<null>(`/system/alert-rules/${ruleID}/toggle`, payload);
}

/**
 * deleteAlertRule 对应 DELETE /api/v1/system/alert-rules/:id，用于删除告警规则。
 */
export function deleteAlertRule(ruleID: ID) {
  return apiClient.delete<null>(`/system/alert-rules/${ruleID}`);
}

/**
 * listAlertEvents 对应 GET /api/v1/system/alert-events，用于查询告警事件列表。
 */
export function listAlertEvents(params: SystemAlertEventListParams) {
  return apiClient.get<SystemAlertEventListResponse>("/system/alert-events", { query: params });
}

/**
 * getAlertEvent 对应 GET /api/v1/system/alert-events/:id，用于获取告警事件详情。
 */
export function getAlertEvent(eventID: ID) {
  return apiClient.get<SystemAlertEventDetail>(`/system/alert-events/${eventID}`, { query: {} });
}

/**
 * handleAlertEvent 对应 POST /api/v1/system/alert-events/:id/handle，用于处理告警事件。
 */
export function handleAlertEvent(eventID: ID, payload: HandleSystemAlertEventRequest) {
  return apiClient.post<null>(`/system/alert-events/${eventID}/handle`, payload);
}

/**
 * ignoreAlertEvent 对应 POST /api/v1/system/alert-events/:id/ignore，用于忽略告警事件。
 */
export function ignoreAlertEvent(eventID: ID, payload: HandleSystemAlertEventRequest) {
  return apiClient.post<null>(`/system/alert-events/${eventID}/ignore`, payload);
}

/**
 * getDashboardHealth 对应 GET /api/v1/system/dashboard/health，用于读取平台健康状态。
 */
export function getDashboardHealth() {
  return apiClient.get<SystemDashboardHealthResponse>("/system/dashboard/health", { query: {} });
}

/**
 * getDashboardResources 对应 GET /api/v1/system/dashboard/resources，用于读取资源使用情况。
 */
export function getDashboardResources() {
  return apiClient.get<SystemDashboardResourcesResponse>("/system/dashboard/resources", { query: {} });
}

/**
 * getDashboardRealtime 对应 GET /api/v1/system/dashboard/realtime，用于读取实时指标与最近告警。
 */
export function getDashboardRealtime() {
  return apiClient.get<SystemDashboardRealtimeResponse>("/system/dashboard/realtime", { query: {} });
}

/**
 * getStatisticsOverview 对应 GET /api/v1/system/statistics/overview，用于读取平台统计总览。
 */
export function getStatisticsOverview() {
  return apiClient.get<SystemStatisticsOverviewResponse>("/system/statistics/overview", { query: {} });
}

/**
 * getStatisticsTrend 对应 GET /api/v1/system/statistics/trend，用于读取趋势图数据。
 */
export function getStatisticsTrend(params: SystemStatisticsTrendParams) {
  return apiClient.get<SystemStatisticsTrendResponse>("/system/statistics/trend", { query: params });
}

/**
 * getSystemSchoolRanking 对应 GET /api/v1/system/statistics/schools，用于读取学校活跃度排行。
 */
export function getSystemSchoolRanking() {
  return apiClient.get<SystemSchoolActivityRankingResponse>("/system/statistics/schools", { query: {} });
}

/**
 * triggerBackup 对应 POST /api/v1/system/backups/trigger，用于手动触发备份。
 */
export function triggerBackup() {
  return apiClient.post<TriggerSystemBackupResponse>("/system/backups/trigger");
}

/**
 * listBackups 对应 GET /api/v1/system/backups，用于读取备份历史列表。
 */
export function listBackups(params: SystemBackupListParams) {
  return apiClient.get<SystemBackupListResponse>("/system/backups", { query: params });
}

/**
 * downloadBackup 对应 GET /api/v1/system/backups/:id/download，用于下载备份文件。
 */
export function downloadBackup(backupID: ID) {
  return apiClient.download(`/system/backups/${backupID}/download`);
}

/**
 * getBackupConfig 对应 GET /api/v1/system/backups/config，用于读取备份配置。
 */
export function getBackupConfig() {
  return apiClient.get<SystemBackupConfig>("/system/backups/config", { query: {} });
}

/**
 * updateBackupConfig 对应 PUT /api/v1/system/backups/config，用于更新备份配置。
 */
export function updateBackupConfig(payload: UpdateSystemBackupConfigRequest) {
  return apiClient.put<SystemBackupConfig>("/system/backups/config", payload);
}
