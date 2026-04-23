// system.ts
// 模块08系统管理与监控工具：集中定义枚举选项、审计查询校验、cron 规范和下载辅助方法。

import type {
  SystemAlertEventStatus,
  SystemAlertLevel,
  SystemAlertType,
  SystemAuditSource,
  SystemBackupStatus,
  SystemBackupType,
  SystemDashboardStatus,
  SystemServiceName,
  SystemStatisticsMetric,
  SystemStatisticsPeriod,
  SystemThresholdMetric,
  SystemThresholdOperator,
} from "@/types/system";

/**
 * 文本值选项。
 */
export interface SystemOption<TValue extends string | number | boolean> {
  value: TValue;
  label: string;
  description?: string;
}

/**
 * 系统告警级别徽标变体。
 */
export type SystemAlertLevelBadgeVariant = "secondary" | "outline" | "destructive";

/**
 * 审计来源选项。
 */
export const SYSTEM_AUDIT_SOURCE_OPTIONS: readonly SystemOption<SystemAuditSource | "all">[] = [
  { value: "all", label: "全部", description: "聚合查询所有日志来源" },
  { value: "login", label: "登录日志", description: "仅查询 login_logs" },
  { value: "operation", label: "操作日志", description: "仅查询 operation_logs" },
  { value: "experiment", label: "实验操作日志", description: "仅查询 instance_operation_logs" },
] as const;

/**
 * 告警类型选项。
 */
export const SYSTEM_ALERT_TYPE_OPTIONS: readonly SystemOption<SystemAlertType>[] = [
  { value: 1, label: "阈值告警" },
  { value: 2, label: "事件告警" },
  { value: 3, label: "服务状态告警" },
] as const;

/**
 * 告警级别选项。
 */
export const SYSTEM_ALERT_LEVEL_OPTIONS: readonly SystemOption<SystemAlertLevel>[] = [
  { value: 1, label: "信息" },
  { value: 2, label: "警告" },
  { value: 3, label: "严重" },
  { value: 4, label: "紧急" },
] as const;

/**
 * 告警事件状态选项。
 */
export const SYSTEM_ALERT_EVENT_STATUS_OPTIONS: readonly SystemOption<SystemAlertEventStatus>[] = [
  { value: 1, label: "待处理" },
  { value: 2, label: "已处理" },
  { value: 3, label: "已忽略" },
] as const;

/**
 * 阈值告警指标选项。
 */
export const SYSTEM_THRESHOLD_METRIC_OPTIONS: readonly SystemOption<SystemThresholdMetric>[] = [
  { value: "cpu_usage", label: "CPU使用率" },
  { value: "memory_usage", label: "内存使用率" },
  { value: "storage_usage", label: "存储使用率" },
] as const;

/**
 * 阈值告警运算符选项。
 */
export const SYSTEM_THRESHOLD_OPERATOR_OPTIONS: readonly SystemOption<SystemThresholdOperator>[] = [
  { value: ">", label: ">" },
  { value: ">=", label: ">=" },
  { value: "<", label: "<" },
  { value: "<=", label: "<=" },
  { value: "==", label: "==" },
] as const;

/**
 * 事件告警来源选项。
 */
export const SYSTEM_EVENT_SOURCE_OPTIONS: readonly SystemOption<"login_logs">[] = [
  { value: "login_logs", label: "登录日志" },
] as const;

/**
 * 事件告警分组选项。
 */
export const SYSTEM_EVENT_GROUP_BY_OPTIONS: readonly SystemOption<"ip">[] = [
  { value: "ip", label: "IP地址" },
] as const;

/**
 * 服务状态告警服务名选项。
 */
export const SYSTEM_SERVICE_NAME_OPTIONS: readonly SystemOption<SystemServiceName>[] = [
  { value: "postgresql", label: "PostgreSQL" },
  { value: "redis", label: "Redis" },
  { value: "nats", label: "NATS" },
  { value: "minio", label: "MinIO" },
  { value: "api_server", label: "API Server" },
  { value: "k8s_cluster", label: "K8s Cluster" },
] as const;

/**
 * 平台统计趋势指标选项。
 */
export const SYSTEM_STATISTICS_METRIC_OPTIONS: readonly SystemOption<SystemStatisticsMetric>[] = [
  { value: "active_users", label: "活跃用户" },
  { value: "new_users", label: "新增用户" },
  { value: "experiments", label: "实验启动量" },
  { value: "api_requests", label: "API请求量" },
] as const;

/**
 * 平台统计周期选项。
 */
export const SYSTEM_STATISTICS_PERIOD_OPTIONS: readonly SystemOption<SystemStatisticsPeriod>[] = [
  { value: "7d", label: "7天" },
  { value: "30d", label: "30天" },
  { value: "90d", label: "90天" },
  { value: "365d", label: "365天" },
] as const;

/**
 * 备份状态选项。
 */
export const SYSTEM_BACKUP_STATUS_OPTIONS: readonly SystemOption<SystemBackupStatus>[] = [
  { value: 1, label: "进行中" },
  { value: 2, label: "成功" },
  { value: 3, label: "失败" },
] as const;

/**
 * 备份类型选项。
 */
export const SYSTEM_BACKUP_TYPE_OPTIONS: readonly SystemOption<SystemBackupType>[] = [
  { value: 1, label: "自动备份" },
  { value: 2, label: "手动备份" },
] as const;

/**
 * isValidSystemCronExpression 校验 6 段秒级 cron 表达式。
 */
export function isValidSystemCronExpression(value: string) {
  return normalizeSystemCronExpression(value) !== null;
}

/**
 * normalizeSystemCronExpression 归一化 6 段秒级 cron 表达式，不合法时返回 null。
 */
export function normalizeSystemCronExpression(value: string) {
  const fields = value.trim().split(/\s+/).filter(Boolean);
  if (fields.length !== 6) {
    return null;
  }

  return fields.join(" ");
}

/**
 * toSystemDateTimeLocalValue 将 RFC3339 时间转换为 datetime-local 输入框值。
 */
export function toSystemDateTimeLocalValue(value?: string | Date | null) {
  if (value === null || value === undefined || value === "") {
    return "";
  }

  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60 * 1000);
  return localDate.toISOString().slice(0, 16);
}

/**
 * fromSystemDateTimeLocalValue 将 datetime-local 输入框值转换为 RFC3339 字符串。
 */
export function fromSystemDateTimeLocalValue(value?: string | null) {
  if (value === null || value === undefined || value.trim() === "") {
    return undefined;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toISOString();
}

/**
 * createSystemRecentRange 生成最近 N 天的时间范围默认值。
 */
export function createSystemRecentRange(days = 7, now = new Date()) {
  const endAt = new Date(now);
  const startAt = new Date(now.getTime() - days * 24 * 60 * 60 * 1000);

  return {
    dateFrom: toSystemDateTimeLocalValue(startAt),
    dateTo: toSystemDateTimeLocalValue(endAt),
  };
}

/**
 * validateSystemAuditScope 校验统一审计查询范围。
 * 全来源查询必须携带完整时间范围，且跨度不能超过 30 天。
 */
export function validateSystemAuditScope(
  source: SystemAuditSource | "all",
  dateFrom?: string | null,
  dateTo?: string | null,
) {
  if (source !== "all") {
    return null;
  }

  if (!dateFrom || !dateTo) {
    return "查询全部日志来源时必须指定开始和结束时间";
  }

  const startAt = new Date(dateFrom);
  const endAt = new Date(dateTo);
  if (Number.isNaN(startAt.getTime()) || Number.isNaN(endAt.getTime())) {
    return "时间范围格式错误";
  }

  if (endAt.getTime() - startAt.getTime() > 30 * 24 * 60 * 60 * 1000) {
    return "查询全部日志来源时范围不能超过30天";
  }

  return null;
}

/**
 * formatSystemDuration 将秒数格式化为易读时长。
 */
export function formatSystemDuration(value?: number | null) {
  if (value === null || value === undefined || value < 0) {
    return "—";
  }

  const totalSeconds = Math.floor(value);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts: string[] = [];

  if (hours > 0) {
    parts.push(`${hours}时`);
  }
  if (minutes > 0) {
    parts.push(`${minutes}分`);
  }
  if (seconds > 0 || parts.length === 0) {
    parts.push(`${seconds}秒`);
  }

  return parts.join("");
}

/**
 * getSystemUsageTone 根据资源占用百分比返回界面色阶。
 */
export function getSystemUsageTone(percent: number) {
  if (percent >= 80) {
    return "danger";
  }
  if (percent >= 60) {
    return "warning";
  }
  return "safe";
}

/**
 * getSystemStatusTone 根据服务健康状态返回界面色阶。
 */
export function getSystemStatusTone(status: SystemDashboardStatus) {
  if (status === "unhealthy") {
    return "danger";
  }
  if (status === "unknown") {
    return "warning";
  }
  return "safe";
}

/**
 * getSystemAlertLevelBadgeVariant 返回告警级别对应的 Badge 变体。
 */
export function getSystemAlertLevelBadgeVariant(level: SystemAlertLevel): SystemAlertLevelBadgeVariant {
  if (level === 1) {
    return "secondary";
  }
  if (level === 2) {
    return "outline";
  }
  if (level === 3 || level === 4) {
    return "destructive";
  }
  return "secondary";
}

/**
 * getSystemAlertLevelClassName 返回告警级别的额外样式。
 */
export function getSystemAlertLevelClassName(level: SystemAlertLevel) {
  if (level === 1) {
    return "bg-sky-500/12 text-sky-700 dark:text-sky-300";
  }
  if (level === 2) {
    return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300";
  }
  if (level === 3) {
    return "bg-destructive/12 text-destructive";
  }
  if (level === 4) {
    return "bg-rose-600/15 text-rose-700 ring-1 ring-rose-500/25 dark:text-rose-300";
  }
  return "";
}

/**
 * getSystemAlertEventStatusClassName 返回告警事件状态样式。
 */
export function getSystemAlertEventStatusClassName(status: SystemAlertEventStatus) {
  if (status === 1) {
    return "bg-destructive/12 text-destructive";
  }
  if (status === 2) {
    return "bg-emerald-500/12 text-emerald-700 dark:text-emerald-300";
  }
  return "bg-muted text-muted-foreground";
}

/**
 * getSystemStatusClassName 返回服务状态附加样式。
 */
export function getSystemStatusClassName(status: SystemDashboardStatus) {
  if (status === "unhealthy") {
    return "bg-destructive/12 text-destructive";
  }
  if (status === "unknown") {
    return "bg-amber-500/12 text-amber-700 dark:text-amber-300";
  }
  return "bg-emerald-500/12 text-emerald-700 dark:text-emerald-300";
}

/**
 * getSystemUsageBarClassName 返回资源条颜色。
 */
export function getSystemUsageBarClassName(percent: number) {
  const tone = getSystemUsageTone(percent);
  if (tone === "danger") {
    return "bg-gradient-to-r from-rose-500 to-red-500";
  }
  if (tone === "warning") {
    return "bg-gradient-to-r from-amber-400 to-orange-500";
  }
  return "bg-gradient-to-r from-emerald-400 to-teal-500";
}

/**
 * saveDownloadedFile 将后端返回的 Blob 保存为浏览器下载文件。
 */
export function saveDownloadedFile(blob: Blob, filename: string) {
  if (typeof window === "undefined") {
    return;
  }

  const objectURL = window.URL.createObjectURL(blob);
  const link = window.document.createElement("a");
  link.href = objectURL;
  link.download = filename;
  window.document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(objectURL);
}
