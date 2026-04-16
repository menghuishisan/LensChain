import type { JsonObject, JsonValue, RendererTheme } from "./types.js";

/**
 * DEFAULT_THEME 提供共享默认主题。
 */
export const DEFAULT_THEME: RendererTheme = {
  background: "#09111f",
  foreground: "#e5f2ff",
  accent: "#2fb3ff",
  success: "#25c48a",
  warning: "#ffb020",
  danger: "#ff5d73",
  muted: "#8fa3bf",
  grid: "rgba(143, 163, 191, 0.18)"
};

/**
 * deepClone 深复制值，避免共享引用。
 */
export function deepClone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

/**
 * deepMerge 按对象语义合并状态增量。
 */
export function deepMerge(base: JsonObject, patch: JsonObject): JsonObject {
  const result = deepClone(base);
  for (const [key, value] of Object.entries(patch)) {
    if (value === null) {
      delete result[key];
      continue;
    }
    const current = result[key];
    if (isJsonObject(current) && isJsonObject(value)) {
      result[key] = deepMerge(current, value);
      continue;
    }
    result[key] = deepClone(value);
  }
  return result;
}

/**
 * flattenChangedKeys 将嵌套差异对象展平成路径列表。
 */
export function flattenChangedKeys(value: JsonObject, prefix = ""): string[] {
  const result: string[] = [];
  for (const [key, child] of Object.entries(value)) {
    const nextPath = prefix ? `${prefix}.${key}` : key;
    result.push(nextPath);
    if (isJsonObject(child)) {
      result.push(...flattenChangedKeys(child, nextPath));
    }
  }
  return result;
}

/**
 * isJsonObject 判断值是否为 JSON 对象。
 */
export function isJsonObject(value: JsonValue | undefined): value is JsonObject {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * clamp 将数值限制在指定范围内。
 */
export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/**
 * lerp 计算两值之间的线性插值。
 */
export function lerp(from: number, to: number, progress: number): number {
  return from + (to - from) * progress;
}

/**
 * randomID 生成渲染层本地标识。
 */
export function randomID(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

/**
 * asArray 将未知值转换为数组。
 */
export function asArray<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

/**
 * asObject 将未知值转换为对象。
 */
export function asObject(value: unknown): JsonObject {
  return isJsonObject(value as JsonValue) ? (value as JsonObject) : {};
}

/**
 * asNumber 将未知值转换为数值。
 */
export function asNumber(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

/**
 * asString 将未知值转换为字符串。
 */
export function asString(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}
