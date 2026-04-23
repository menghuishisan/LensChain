// enum.ts
// 枚举文本映射基础工具，避免在页面和组件中重复散落状态文案。

/**
 * 枚举文本映射表。
 */
export type EnumTextMap<TKey extends string | number> = Partial<Record<TKey, string>>;

/**
 * 获取枚举展示文本，未知值返回兜底文本。
 */
export function getEnumText<TKey extends string | number>(
  map: EnumTextMap<TKey>,
  value: TKey | null | undefined,
  fallback = "未知",
) {
  if (value === null || value === undefined) {
    return fallback;
  }

  return map[value] ?? fallback;
}

/**
 * 创建稳定的枚举文本映射函数。
 */
export function createEnumTextMapper<TKey extends string | number>(map: EnumTextMap<TKey>, fallback = "未知") {
  return (value: TKey | null | undefined) => getEnumText(map, value, fallback);
}
