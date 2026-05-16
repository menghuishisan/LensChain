/**
 * primitiveRenderer.ts — 单次帧的原语绘制调度器。
 *
 * 职责：
 *   1. 调用 layoutSolver 把 47 原语解算成像素位置。
 *   2. 按 Layer 顺序遍历，对 33 canvas 原语 dispatch 到 drawer。
 *   3. 14 个 DOM/浮层/容器类型被显式过滤掉（CANVAS_PRIMITIVE_TYPES 检查）。
 *   4. 教师 annotation 浮层数据不在此处理（由 React SimAnnotationLayer 接管）。
 *
 * 不写兑底：未注册的 canvas type → 抛错（说明协议被破坏）。
 */

import type { Primitive, PrimitiveLayer, RendererTheme } from "./types.js";
import { resolveLayout, type ResolvedLayout } from "./layoutSolver.js";
import {
  CANVAS_PRIMITIVE_TYPES, PRIMITIVE_DRAWER_MAP,
  type DrawEnvironment,
} from "./primitiveDrawers.js";

/** Layer 绘制顺序（先入底）。 */
const LAYER_ORDER: readonly PrimitiveLayer[] = ["background", "content", "effect", "overlay"];

/** 单帧渲染输入。 */
export interface FrameInput {
  ctx: CanvasRenderingContext2D;
  width: number;
  height: number;
  theme: RendererTheme;
  primitives: readonly Primitive[];
  /** 当前活跃高亮 primitive id（来自 MicroStep.highlight_ids）。 */
  highlightIds?: ReadonlySet<string>;
  /** 当前帧触发的一次性效果 id（来自 MicroStep.fire_primitives）。 */
  fireIds?: ReadonlySet<string>;
  /** 当前帧高精度时间 ms（RAF.now）。 */
  now: number;
  /** 当前 tick。 */
  tick: number;
}

/**
 * renderFrame 渲染一帧。
 * 调用方负责 ctx.clearRect / 背景填充；本函数只画原语。
 */
export function renderFrame(input: FrameInput): ResolvedLayout {
  const { ctx, width, height, theme, primitives, now, tick } = input;
  const highlightIds = input.highlightIds ?? new Set<string>();
  const fireIds = input.fireIds ?? new Set<string>();

  // 背景
  ctx.save();
  ctx.fillStyle = theme.background;
  ctx.fillRect(0, 0, width, height);
  ctx.restore();

  const resolvedLayout = resolveLayout(
    primitives, width, height,
    (text) => ctx.measureText(text).width,
  );

  const env: DrawEnvironment = {
    ctx, theme, highlightIds, fireIds, now, tick, resolvedLayout,
  };

  for (const layer of LAYER_ORDER) {
    for (const p of primitives) {
      if (p.layer !== layer) continue;
      if (!CANVAS_PRIMITIVE_TYPES.has(p.type)) continue; // 14 DOM/浮层/容器跳过
      const drawer = PRIMITIVE_DRAWER_MAP.get(p.type);
      if (!drawer) {
        // CANVAS_PRIMITIVE_TYPES 与 PRIMITIVE_DRAWER_MAP 必须严格同步；
        // 走到这里说明开发期遗漏注册——抛错暴露问题。
        throw new Error(`primitiveRenderer: canvas 原语 "${p.type}" 未注册 drawer`);
      }
      ctx.save();
      try {
        drawer(p, env);
      } finally {
        ctx.restore();
      }
    }
  }
  return resolvedLayout;
}
