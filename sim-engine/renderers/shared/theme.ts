/**
 * theme.ts — 按场景 category 选择主题色板。
 *
 * 设计决策：
 * • 每个 SceneCategory 只是一份配色，渲染逻辑与 category 解耦。
 * • SceneCategory 是闭集（types.ts 定义），本表必须覆盖全部枚举值。
 * • <strong>不写兑底</strong>：未注册的 category 直接抛错，暴露问题不隐藏。
 *
 * 文档：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.5
 */

import type { RendererTheme, SceneCategory } from "./types.js";

/** 9 类目（8 内置 + 1 教师扩展 generic）的主题映射，必须覆盖全枚举。 */
const THEME_BY_CATEGORY: Record<SceneCategory, RendererTheme> = {
  consensus: {
    background: "#0d1222",
    foreground: "#f4f7fb",
    accent: "#c084fc",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(192,132,252,0.18)",
  },
  cryptography: {
    background: "#0a1628",
    foreground: "#f4f7fb",
    accent: "#67e8f9",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(103,232,249,0.18)",
  },
  node_network: {
    background: "#0b1527",
    foreground: "#f4f7fb",
    accent: "#38bdf8",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(56,189,248,0.18)",
  },
  data_structure: {
    background: "#0c1425",
    foreground: "#f4f7fb",
    accent: "#94a3b8",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#94a3b8",
    grid: "rgba(148,163,184,0.18)",
  },
  transaction: {
    background: "#0d1220",
    foreground: "#f4f7fb",
    accent: "#fbbf24",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(251,191,36,0.18)",
  },
  smart_contract: {
    background: "#0e1424",
    foreground: "#f4f7fb",
    accent: "#f97316",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(249,115,22,0.18)",
  },
  attack_security: {
    background: "#120e1c",
    foreground: "#f4f7fb",
    accent: "#f87171",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(248,113,113,0.20)",
  },
  economic: {
    background: "#0d1422",
    foreground: "#f4f7fb",
    accent: "#facc15",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#93a4bb",
    grid: "rgba(250,204,21,0.18)",
  },
  generic: {
    background: "#0c1220",
    foreground: "#e5f2ff",
    accent: "#2fb3ff",
    success: "#4ade80",
    warning: "#fbbf24",
    danger: "#f87171",
    muted: "#8fa3bf",
    grid: "rgba(47,179,255,0.18)",
  },
};

/**
 * 按 category 取主题。
 * SceneCategory 类型是 9 值闭集 (TS 编译期即保证)；运行期又走 unknown branch 只能是协议被破坏，直接抛错。
 */
export function themeForCategory(category: SceneCategory): RendererTheme {
  const t = THEME_BY_CATEGORY[category];
  if (!t) {
    throw new Error(`themeForCategory: 未注册的 SceneCategory "${category}"，协议被破坏`);
  }
  return t;
}

/** 全部主题（仅供调试 / 文档生成使用）。 */
export const ALL_THEMES = THEME_BY_CATEGORY;
