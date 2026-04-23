// format.ts
// 通用格式化工具，集中处理日期、文件大小、数字、分数和 GPA 展示。

const EMPTY_TEXT = "—";

function normalizeNumber(value: number): number | null {
  return Number.isFinite(value) ? value : null;
}

function trimTrailingZeros(value: string): string {
  return value.replace(/\.0+$/, "").replace(/(\.\d*?)0+$/, "$1");
}

function roundToPrecision(value: number, precision: number) {
  const factor = 10 ** precision;
  return Math.round((value + Number.EPSILON) * factor) / factor;
}

/**
 * 将 ISO/RFC3339 时间格式化为本地日期时间。
 */
export function formatDateTime(
  value: string | number | Date | null | undefined,
  locale = "zh-CN",
  options: Intl.DateTimeFormatOptions = {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  },
) {
  if (value === null || value === undefined || value === "") {
    return EMPTY_TEXT;
  }

  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return EMPTY_TEXT;
  }

  return new Intl.DateTimeFormat(locale, options).format(date);
}

/**
 * 将字节数格式化为可读文件大小。
 */
export function formatFileSize(bytes: number) {
  const normalized = normalizeNumber(bytes);
  if (normalized === null || normalized < 0) {
    return EMPTY_TEXT;
  }

  const units = ["B", "KB", "MB", "GB", "TB"] as const;
  let size = normalized;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  const precision = unitIndex === 0 ? 0 : 1;
  return `${trimTrailingZeros(size.toFixed(precision))} ${units[unitIndex]}`;
}

/**
 * 格式化普通数字，默认使用千分位。
 */
export function formatNumber(value: number, locale = "en-US") {
  const normalized = normalizeNumber(value);
  if (normalized === null) {
    return EMPTY_TEXT;
  }

  return new Intl.NumberFormat(locale).format(normalized);
}

/**
 * 格式化分数，默认保留两位小数。
 */
export function formatScore(value: number, precision = 2) {
  const normalized = normalizeNumber(value);
  if (normalized === null) {
    return EMPTY_TEXT;
  }

  return trimTrailingZeros(roundToPrecision(normalized, precision).toFixed(precision));
}

/**
 * 格式化 GPA，默认保留两位小数。
 */
export function formatGPA(value: number, precision = 2) {
  const normalized = normalizeNumber(value);
  if (normalized === null) {
    return EMPTY_TEXT;
  }

  return roundToPrecision(normalized, precision).toFixed(precision);
}
