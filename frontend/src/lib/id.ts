// id.ts
// 雪花 ID 字符串处理工具，避免前端以 number 长期承载后端 ID。

import type { ID } from "@/types/api";

/**
 * 可标准化为 ID 的输入类型。
 */
export type IDInput = string | number | bigint | null | undefined;

/**
 * 将输入标准化为字符串 ID，空值返回 null。
 */
export function normalizeID(value: IDInput): ID | null {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === "number" && !Number.isFinite(value)) {
    return null;
  }

  const normalized = String(value).trim();
  return normalized.length > 0 ? normalized : null;
}

/**
 * 获取必需 ID，缺失时抛出明确错误。
 */
export function ensureID(value: IDInput): ID {
  const normalized = normalizeID(value);
  if (normalized === null) {
    throw new Error("ID不能为空");
  }

  return normalized;
}

/**
 * 按字符串语义比较两个 ID。
 */
export function isSameID(left: IDInput, right: IDInput) {
  const normalizedLeft = normalizeID(left);
  const normalizedRight = normalizeID(right);

  return normalizedLeft !== null && normalizedLeft === normalizedRight;
}
