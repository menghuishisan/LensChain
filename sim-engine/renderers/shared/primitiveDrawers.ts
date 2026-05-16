/**
 * primitiveDrawers.ts — 33 个 canvas 原语的绘制函数 (Map dispatch)。
 *
 * 设计依据：
 *   docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.2
 *   docs/modules/04-实验环境/sim-engine-redesign-proposal.html §核心规则四铁律
 *
 * 原则：
 *   • 仅 33 canvas 原语注册到 PRIMITIVE_DRAWER_MAP；14 个 DOM/浮层/容器类型不在此 map。
 *   • drawer 接收 ResolvedPosition (已由 layoutSolver 算好像素)，drawer 不再做坐标计算。
 *   • 不写兑底：未知 type / 缺字段 / 引用不存在的 anchor → 抛错（layoutSolver 已校验大部分）。
 *   • 字段名严格对齐 framework/primitive.go schema (id/label/status/role/anchor_id/...)
 */

import type { JsonObject, JsonValue, Primitive, PrimitiveType, RendererTheme } from "./types.js";
import { asArray, asNumber, asString, clamp } from "./utils.js";
import type { ResolvedLayout, ResolvedPosition } from "./layoutSolver.js";

// ============================================================
// 绘制环境
// ============================================================

/** 绘制时上下文（每帧由 primitiveRenderer 注入）。 */
export interface DrawEnvironment {
  ctx: CanvasRenderingContext2D;
  theme: RendererTheme;
  /** 当前活跃高亮的 primitive id 集合（由 MicroStep.highlight_ids 注入）。 */
  highlightIds: ReadonlySet<string>;
  /** 当前帧应触发的一次性效果 id 集合（MicroStep.fire_primitives）。 */
  fireIds: ReadonlySet<string>;
  /** 当前帧高精度时间 (ms)，用于动画相位计算。 */
  now: number;
  /** 当前 tick（用于 burst.fired_at_tick 幂等）。 */
  tick: number;
  /** layoutSolver 输出。 */
  resolvedLayout: ResolvedLayout;
}

/** drawer 函数签名。 */
export type PrimitiveDrawer = (p: Primitive, env: DrawEnvironment) => void;

// ============================================================
// 颜色 / 文字 helper
// ============================================================

/** 角色 → 主题色映射（与 06.md §3.2 color_role 词表对齐）。 */
function roleColor(role: string, theme: RendererTheme): string {
  switch (role) {
    case "primary":
    case "honest":
    case "ok":
    case "success":
      return theme.success;
    case "attack":
    case "byzantine":
    case "danger":
    case "error":
      return theme.danger;
    case "warning":
    case "pending":
      return theme.warning;
    case "accent":
    case "active":
      return theme.accent;
    case "muted":
    case "idle":
      return theme.muted;
    default:
      return theme.accent;
  }
}

/** 状态 → 透明度 / 边框样式。 */
function statusStyle(status: string): { alpha: number; stroke: "solid" | "dashed" } {
  switch (status) {
    case "active": return { alpha: 1.0, stroke: "solid" };
    case "pending": return { alpha: 0.65, stroke: "dashed" };
    case "idle": return { alpha: 0.45, stroke: "solid" };
    case "error": return { alpha: 1.0, stroke: "solid" };
    case "done": return { alpha: 0.85, stroke: "solid" };
    default: return { alpha: 1.0, stroke: "solid" };
  }
}

function withAlpha(color: string, alpha: number): string {
  // 简单 hex → rgba 转换；非 hex 直接返回（rgba()/hsl() 等已含 alpha）
  if (color.startsWith("#") && (color.length === 7 || color.length === 4)) {
    const hex = color.length === 4
      ? "#" + color[1]!.repeat(2) + color[2]!.repeat(2) + color[3]!.repeat(2)
      : color;
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return `rgba(${r},${g},${b},${alpha})`;
  }
  return color;
}

function drawText(
  ctx: CanvasRenderingContext2D,
  text: string, x: number, y: number,
  opts: { color?: string; size?: number; weight?: number; align?: CanvasTextAlign; baseline?: CanvasTextBaseline } = {},
): void {
  const { color = "#e5f2ff", size = 11, weight = 400, align = "center", baseline = "middle" } = opts;
  ctx.save();
  ctx.fillStyle = color;
  ctx.font = `${weight} ${size}px ui-monospace, "SF Mono", monospace`;
  ctx.textAlign = align;
  ctx.textBaseline = baseline;
  ctx.fillText(text, x, y);
  ctx.restore();
}

function requirePos(p: Primitive, env: DrawEnvironment): ResolvedPosition {
  const pos = env.resolvedLayout.positions.get(p.id);
  if (!pos) {
    throw new Error(`drawer: primitive "${p.id}" (${p.type}) 缺解算位置`);
  }
  return pos;
}

function requireAnchorPos(anchorId: string, env: DrawEnvironment, owner: Primitive): ResolvedPosition {
  if (!anchorId) {
    throw new Error(`drawer: ${owner.type} "${owner.id}" 缺 anchor_id`);
  }
  const pos = env.resolvedLayout.positions.get(anchorId);
  if (!pos) {
    throw new Error(`drawer: ${owner.type} "${owner.id}" anchor_id="${anchorId}" 未解算`);
  }
  return pos;
}

// ============================================================
// 几何类（8）
// ============================================================

const drawNode: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const r = pos.radius ?? 22;
  const role = asString(p.params.role, "primary");
  const status = asString(p.params.status, "active");
  const label = asString(p.params.label, "");
  const color = roleColor(role, env.theme);
  const style = statusStyle(status);
  const highlighted = env.highlightIds.has(p.id);

  const { ctx } = env;
  // halo for highlighted
  if (highlighted) {
    ctx.save();
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, r + 8, 0, Math.PI * 2);
    ctx.strokeStyle = env.theme.accent;
    ctx.lineWidth = 2;
    ctx.shadowBlur = 12;
    ctx.shadowColor = env.theme.accent;
    ctx.stroke();
    ctx.restore();
  }
  // fill disc
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, r, 0, Math.PI * 2);
  ctx.fillStyle = withAlpha(color, style.alpha * 0.35);
  ctx.fill();
  // outline
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  if (style.stroke === "dashed") ctx.setLineDash([4, 3]);
  ctx.stroke();
  ctx.setLineDash([]);
  // label inside
  if (label) {
    drawText(ctx, label, pos.x, pos.y, { color: env.theme.foreground, size: r >= 22 ? 11 : 10, weight: 600 });
  }
};

const drawEdge: PrimitiveDrawer = (p, env) => {
  const fromId = asString(p.params.from_id, "");
  const toId = asString(p.params.to_id, "");
  const from = env.resolvedLayout.positions.get(fromId);
  const to = env.resolvedLayout.positions.get(toId);
  if (!from || !to) {
    throw new Error(`edge "${p.id}": from_id="${fromId}" 或 to_id="${toId}" 未解算`);
  }
  const style = asString(p.params.style, "solid");
  const animation = asString(p.params.animation, "");
  const { ctx } = env;
  ctx.strokeStyle = withAlpha(env.theme.muted, 0.65);
  ctx.lineWidth = 1.5;
  if (style === "dashed") ctx.setLineDash([6, 4]);
  ctx.beginPath();
  ctx.moveTo(from.x, from.y);
  ctx.lineTo(to.x, to.y);
  ctx.stroke();
  ctx.setLineDash([]);
  // arrow head when animation=flow
  if (animation === "flow") {
    const angle = Math.atan2(to.y - from.y, to.x - from.x);
    const ax = to.x - Math.cos(angle) * (to.radius ?? 22);
    const ay = to.y - Math.sin(angle) * (to.radius ?? 22);
    ctx.beginPath();
    ctx.moveTo(ax, ay);
    ctx.lineTo(ax - Math.cos(angle - 0.35) * 8, ay - Math.sin(angle - 0.35) * 8);
    ctx.lineTo(ax - Math.cos(angle + 0.35) * 8, ay - Math.sin(angle + 0.35) * 8);
    ctx.closePath();
    ctx.fillStyle = env.theme.accent;
    ctx.fill();
  }
};

const drawBar: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const colorRole = asString(p.params.color_role, "primary");
  const heightLogical = asNumber(p.params.height, 0.5);
  const widthLogical = asNumber(p.params.width, 0.1);
  const color = roleColor(colorRole, env.theme);
  const w = pos.width ?? widthLogical * 200;
  const h = clamp(heightLogical, 0, 1) * (pos.height ?? 80);
  const { ctx } = env;
  ctx.fillStyle = withAlpha(color, env.highlightIds.has(p.id) ? 1 : 0.7);
  ctx.fillRect(pos.x - w / 2, pos.y - h / 2, w, h);
  const label = asString(p.params.label, "");
  if (label) drawText(ctx, label, pos.x, pos.y + h / 2 + 12, { color: env.theme.muted, size: 10 });
};

const drawCurve: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const points = asArray<JsonObject>(p.params.points);
  if (points.length < 2) return;
  const style = asString(p.params.style, "solid");
  const w = pos.width ?? env.resolvedLayout.main.width;
  const h = pos.height ?? env.resolvedLayout.main.height;
  const left = pos.x - w / 2;
  const top = pos.y - h / 2;
  // points 为算法坐标系（通常 0~1），归一化到当前区域
  const { ctx } = env;
  ctx.strokeStyle = env.theme.accent;
  ctx.lineWidth = 2;
  if (style === "dashed") ctx.setLineDash([5, 4]);
  ctx.beginPath();
  points.forEach((pt, i) => {
    const px = asNumber(pt.x, 0);
    const py = asNumber(pt.y, 0);
    const x = left + clamp(px, 0, 1) * w;
    const y = top + (1 - clamp(py, 0, 1)) * h;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  ctx.stroke();
  ctx.setLineDash([]);
};

const drawPolygon: PrimitiveDrawer = (p, env) => {
  const vertices = asArray<JsonObject>(p.params.vertices);
  if (vertices.length < 3) return;
  const pos = env.resolvedLayout.positions.get(p.id);
  const main = env.resolvedLayout.main;
  const ox = pos?.x ?? main.x + main.width / 2;
  const oy = pos?.y ?? main.y + main.height / 2;
  const span = Math.min(main.width, main.height);
  const fill = asString(p.params.fill, "");
  const stroke = asString(p.params.stroke, "");
  const { ctx } = env;
  ctx.beginPath();
  vertices.forEach((v, i) => {
    const x = ox + asNumber(v.x, 0) * span;
    const y = oy + asNumber(v.y, 0) * span;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  ctx.closePath();
  if (fill) { ctx.fillStyle = fill; ctx.fill(); }
  if (stroke) { ctx.strokeStyle = stroke; ctx.lineWidth = 1.5; ctx.stroke(); }
};

const drawArea: PrimitiveDrawer = (p, env) => {
  const points = asArray<JsonObject>(p.params.points);
  if (points.length < 2) return;
  const main = env.resolvedLayout.main;
  const { ctx } = env;
  const grad = ctx.createLinearGradient(main.x, main.y, main.x, main.y + main.height);
  grad.addColorStop(0, withAlpha(env.theme.accent, 0.35));
  grad.addColorStop(1, withAlpha(env.theme.accent, 0));
  ctx.fillStyle = grad;
  ctx.beginPath();
  points.forEach((pt, i) => {
    const x = main.x + clamp(asNumber(pt.x, 0), 0, 1) * main.width;
    const y = main.y + (1 - clamp(asNumber(pt.y, 0), 0, 1)) * main.height;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  ctx.lineTo(main.x + main.width, main.y + main.height);
  ctx.lineTo(main.x, main.y + main.height);
  ctx.closePath();
  ctx.fill();
};

const drawGridCell: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const colorRole = asString(p.params.color_role, "muted");
  const color = roleColor(colorRole, env.theme);
  const w = pos.width ?? 48;
  const h = pos.height ?? 32;
  const { ctx } = env;
  ctx.fillStyle = withAlpha(color, env.highlightIds.has(p.id) ? 0.9 : 0.55);
  ctx.fillRect(pos.x - w / 2 + 2, pos.y - h / 2 + 2, w - 4, h - 4);
  ctx.strokeStyle = withAlpha(color, 0.9);
  ctx.lineWidth = 1;
  ctx.strokeRect(pos.x - w / 2 + 2, pos.y - h / 2 + 2, w - 4, h - 4);
  const value = p.params.value;
  if (value !== null && value !== undefined) {
    drawText(ctx, String(value), pos.x, pos.y, { color: env.theme.foreground, size: 10, weight: 600 });
  }
};

const drawRing: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const total = Math.max(1, asNumber(p.params.total, 1));
  const current = clamp(asNumber(p.params.current, 0), 0, total);
  const r = pos.radius ?? 32;
  const { ctx } = env;
  // 底环
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, r, 0, Math.PI * 2);
  ctx.strokeStyle = withAlpha(env.theme.muted, 0.2);
  ctx.lineWidth = 6;
  ctx.stroke();
  // 进度
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, r, -Math.PI / 2, -Math.PI / 2 + (Math.PI * 2 * current) / total);
  ctx.strokeStyle = env.theme.accent;
  ctx.lineWidth = 6;
  ctx.lineCap = "round";
  ctx.stroke();
  ctx.lineCap = "butt";
  // 文字
  drawText(ctx, `${current}/${total}`, pos.x, pos.y, { color: env.theme.foreground, size: 12, weight: 600 });
  const label = asString(p.params.label, "");
  if (label) drawText(ctx, label, pos.x, pos.y + r + 14, { color: env.theme.muted, size: 10 });
};

// ============================================================
// 动效类（7）— 全部需要 anchor_id
// ============================================================

const drawParticleStream: PrimitiveDrawer = (p, env) => {
  const anchor = requireAnchorPos(asString(p.params.anchor_id, ""), env, p);
  const color = roleColor(asString(p.params.color_role, "accent"), env.theme);
  const rate = asNumber(p.params.rate, 20);
  const lifetime = asNumber(p.params.lifetime_ms, 1200);
  const direction = asString(p.params.direction, "out");
  const { ctx } = env;
  // 简化：用 now+rate 决定 N 个粒子的相位
  const count = Math.min(12, Math.max(4, Math.floor(rate / 4)));
  for (let i = 0; i < count; i++) {
    const phase = ((env.now / lifetime) + i / count) % 1;
    const distance = phase * 40;
    const angle = (i / count) * Math.PI * 2;
    const dx = direction === "in" ? -distance : distance;
    const x = anchor.x + Math.cos(angle) * dx;
    const y = anchor.y + Math.sin(angle) * dx;
    ctx.beginPath();
    ctx.arc(x, y, 2, 0, Math.PI * 2);
    ctx.fillStyle = withAlpha(color, 1 - phase);
    ctx.fill();
  }
};

const drawBurst: PrimitiveDrawer = (p, env) => {
  const anchor = requireAnchorPos(asString(p.params.anchor_id, ""), env, p);
  const firedAt = asNumber(p.params.fired_at_tick, -1);
  if (firedAt < 0 || env.tick - firedAt > 1) return; // 仅在触发后短时间内显示
  const color = asString(p.params.color, env.theme.warning);
  const duration = asNumber(p.params.duration_ms, 600);
  const progress = Math.min(1, (env.now % duration) / duration);
  const r = 8 + progress * 40;
  const { ctx } = env;
  ctx.beginPath();
  ctx.arc(anchor.x, anchor.y, r, 0, Math.PI * 2);
  ctx.strokeStyle = withAlpha(color, 1 - progress);
  ctx.lineWidth = 3;
  ctx.stroke();
};

const drawPulse: PrimitiveDrawer = (p, env) => {
  const anchor = requireAnchorPos(asString(p.params.anchor_id, ""), env, p);
  const color = roleColor(asString(p.params.color_role, "accent"), env.theme);
  const period = asNumber(p.params.period_ms, 1200);
  const phase = (env.now % period) / period;
  const r = (anchor.radius ?? 22) + 6 + phase * 12;
  const { ctx } = env;
  ctx.beginPath();
  ctx.arc(anchor.x, anchor.y, r, 0, Math.PI * 2);
  ctx.strokeStyle = withAlpha(color, 0.7 * (1 - phase));
  ctx.lineWidth = 2;
  ctx.stroke();
};

const drawTrail: PrimitiveDrawer = (p, env) => {
  const points = asArray<JsonObject>(p.params.points);
  if (points.length < 2) return;
  const main = env.resolvedLayout.main;
  const duration = asNumber(p.params.duration_ms, 1500);
  const fade = asNumber(p.params.fade_ms, 500);
  const elapsed = env.now % (duration + fade);
  const alpha = elapsed < duration ? 1 : Math.max(0, 1 - (elapsed - duration) / fade);
  const { ctx } = env;
  ctx.strokeStyle = withAlpha(env.theme.accent, alpha * 0.85);
  ctx.lineWidth = 2;
  ctx.setLineDash([4, 3]);
  ctx.beginPath();
  points.forEach((pt, i) => {
    const x = main.x + clamp(asNumber(pt.x, 0), 0, 1) * main.width;
    const y = main.y + clamp(asNumber(pt.y, 0), 0, 1) * main.height;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  ctx.stroke();
  ctx.setLineDash([]);
};

const drawGlow: PrimitiveDrawer = (p, env) => {
  const anchor = requireAnchorPos(asString(p.params.anchor_id, ""), env, p);
  const color = roleColor(asString(p.params.color_role, "accent"), env.theme);
  const intensity = clamp(asNumber(p.params.intensity, 0.6), 0, 1);
  const r = (anchor.radius ?? 22) + 16;
  const { ctx } = env;
  const grad = ctx.createRadialGradient(anchor.x, anchor.y, anchor.radius ?? 22, anchor.x, anchor.y, r);
  grad.addColorStop(0, withAlpha(color, intensity * 0.6));
  grad.addColorStop(1, withAlpha(color, 0));
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.arc(anchor.x, anchor.y, r, 0, Math.PI * 2);
  ctx.fill();
};

const drawShake: PrimitiveDrawer = (p, env) => {
  // shake 不画自己；只是为 anchor 的下一帧加位移。这里画一个短暂红圈表示触发。
  const anchor = requireAnchorPos(asString(p.params.anchor_id, ""), env, p);
  const duration = asNumber(p.params.duration_ms, 400);
  const phase = (env.now % duration) / duration;
  if (phase > 0.8) return;
  const { ctx } = env;
  ctx.strokeStyle = withAlpha(env.theme.danger, 1 - phase);
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(anchor.x, anchor.y, (anchor.radius ?? 22) + 4, 0, Math.PI * 2);
  ctx.stroke();
};

const drawShiftAnimation: PrimitiveDrawer = (p, env) => {
  // 同上：实际位移由布局生成下一帧呈现；这里仅画方向箭头表示动作
  const target = requireAnchorPos(asString(p.params.target_id, ""), env, p);
  const direction = asString(p.params.direction, "right");
  const distance = asNumber(p.params.distance, 30);
  const dx = direction === "right" ? distance : direction === "left" ? -distance : 0;
  const dy = direction === "down" ? distance : direction === "up" ? -distance : 0;
  const { ctx } = env;
  ctx.strokeStyle = env.theme.accent;
  ctx.lineWidth = 1.5;
  ctx.setLineDash([3, 3]);
  ctx.beginPath();
  ctx.moveTo(target.x, target.y);
  ctx.lineTo(target.x + dx, target.y + dy);
  ctx.stroke();
  ctx.setLineDash([]);
};

// ============================================================
// 状态指示类（7 canvas，error_overlay 走 DOM）
// ============================================================

const drawPhaseProgress: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const phases = asArray<string>(p.params.phases);
  const current = asNumber(p.params.current_index, 0);
  const progress = clamp(asNumber(p.params.progress, 0), 0, 1);
  if (phases.length === 0) return;
  const { ctx } = env;
  const padding = 16;
  const segW = (pos.width! - padding * 2) / phases.length;
  const y = pos.y + pos.height! / 2;
  phases.forEach((label, i) => {
    const cx = pos.x + padding + segW * (i + 0.5);
    const done = i < current;
    const active = i === current;
    // dot
    ctx.beginPath();
    ctx.arc(cx, y, 4, 0, Math.PI * 2);
    ctx.fillStyle = done ? env.theme.success : active ? env.theme.accent : env.theme.muted;
    ctx.fill();
    // label
    drawText(ctx, label, cx, y + 12, {
      color: done ? env.theme.success : active ? env.theme.accent : env.theme.muted,
      size: 10,
    });
    // connector
    if (i < phases.length - 1) {
      ctx.strokeStyle = done ? env.theme.success : withAlpha(env.theme.muted, 0.3);
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(cx + 6, y);
      ctx.lineTo(cx + segW - 6, y);
      ctx.stroke();
    }
  });
  // 当前段填充进度
  if (current >= 0 && current < phases.length) {
    const cx = pos.x + padding + segW * (current + 0.5);
    ctx.fillStyle = withAlpha(env.theme.accent, 0.6);
    ctx.fillRect(cx, y - 1, (segW - 12) * progress, 2);
  }
};

const drawProgressBar: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const value = asNumber(p.params.value, 0);
  const max = Math.max(1, asNumber(p.params.max, 1));
  const ratio = clamp(value / max, 0, 1);
  const label = asString(p.params.label, "");
  const w = pos.width ?? 160;
  const h = 8;
  const { ctx } = env;
  if (label) drawText(ctx, label, pos.x, pos.y - 10, { color: env.theme.muted, size: 10, align: "center" });
  ctx.fillStyle = withAlpha(env.theme.muted, 0.2);
  ctx.fillRect(pos.x - w / 2, pos.y, w, h);
  ctx.fillStyle = env.theme.accent;
  ctx.fillRect(pos.x - w / 2, pos.y, w * ratio, h);
};

const drawTargetZone: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const label = asString(p.params.label, "");
  const axis = asString(p.params.axis, "y");
  const { ctx } = env;
  ctx.strokeStyle = env.theme.warning;
  ctx.lineWidth = 1.5;
  ctx.setLineDash([6, 4]);
  ctx.beginPath();
  if (axis === "y") {
    ctx.moveTo(pos.x, pos.y);
    ctx.lineTo(pos.x + pos.width!, pos.y);
  } else {
    ctx.moveTo(pos.x, pos.y);
    ctx.lineTo(pos.x, pos.y + pos.height!);
  }
  ctx.stroke();
  ctx.setLineDash([]);
  if (label) {
    if (axis === "y") drawText(ctx, label, pos.x + 8, pos.y - 8, { color: env.theme.warning, size: 10, align: "left" });
    else drawText(ctx, label, pos.x + 8, pos.y + 10, { color: env.theme.warning, size: 10, align: "left", baseline: "top" });
  }
};

const drawLinkIndicator: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const status = asString(p.params.status, "idle");
  const linkGroup = asString(p.params.link_group, "");
  const color = status === "active" ? env.theme.accent : status === "recent" ? env.theme.warning : env.theme.muted;
  const { ctx } = env;
  ctx.fillStyle = withAlpha(color, 0.25);
  ctx.strokeStyle = color;
  ctx.lineWidth = 1.5;
  const w = 64, h = 22;
  const x = pos.x - w / 2, y = pos.y - h / 2;
  ctx.beginPath();
  // rounded rect
  const r = 11;
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();
  drawText(ctx, `🔗 ${linkGroup || "link"}`, pos.x, pos.y, { color, size: 10, weight: 600 });
};

const drawExternalEventMarker: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const label = asString(p.params.label, "ext");
  const fadeMs = asNumber(p.params.fade_ms, 1500);
  const phase = (env.now % fadeMs) / fadeMs;
  const alpha = 1 - phase;
  const { ctx } = env;
  ctx.fillStyle = withAlpha(env.theme.warning, alpha * 0.85);
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, 6, 0, Math.PI * 2);
  ctx.fill();
  drawText(ctx, label, pos.x + 10, pos.y, { color: env.theme.warning, size: 10, align: "left" });
};

const drawVerifyPathHighlight: PrimitiveDrawer = (p, env) => {
  const nodeIds = asArray<string>(p.params.node_ids);
  const { ctx } = env;
  ctx.strokeStyle = env.theme.success;
  ctx.lineWidth = 2.5;
  for (let i = 0; i < nodeIds.length; i++) {
    const id = nodeIds[i]!;
    const pos = env.resolvedLayout.positions.get(id);
    if (!pos) continue;
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, (pos.radius ?? 22) + 4, 0, Math.PI * 2);
    ctx.stroke();
    if (i < nodeIds.length - 1) {
      const next = env.resolvedLayout.positions.get(nodeIds[i + 1]!);
      if (next) {
        ctx.beginPath();
        ctx.moveTo(pos.x, pos.y);
        ctx.lineTo(next.x, next.y);
        ctx.stroke();
      }
    }
  }
};

const drawRiskGauge: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const value = clamp(asNumber(p.params.value, 0), 0, 1);
  const r = 30;
  const { ctx } = env;
  // arc 底
  ctx.beginPath();
  ctx.arc(pos.x, pos.y + 10, r, Math.PI, 0);
  ctx.strokeStyle = withAlpha(env.theme.muted, 0.3);
  ctx.lineWidth = 6;
  ctx.stroke();
  // 当前
  const color = value > 0.66 ? env.theme.danger : value > 0.33 ? env.theme.warning : env.theme.success;
  ctx.beginPath();
  ctx.arc(pos.x, pos.y + 10, r, Math.PI, Math.PI + Math.PI * value);
  ctx.strokeStyle = color;
  ctx.lineWidth = 6;
  ctx.lineCap = "round";
  ctx.stroke();
  ctx.lineCap = "butt";
  drawText(ctx, `${Math.round(value * 100)}%`, pos.x, pos.y + 12, { color, size: 12, weight: 700 });
};

// ============================================================
// 领域复合类（11）
// ============================================================

const drawVoteMatrix: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const rows = Math.max(1, asNumber(p.params.rows, 1));
  const cols = Math.max(1, asNumber(p.params.cols, 1));
  const cells = asArray<JsonObject>(p.params.cells);
  const w = pos.width ?? 200, h = pos.height ?? 120;
  const cw = w / cols, ch = h / rows;
  const left = pos.x - w / 2, top = pos.y - h / 2;
  const { ctx } = env;
  // 背景网格
  ctx.strokeStyle = withAlpha(env.theme.muted, 0.2);
  ctx.lineWidth = 1;
  for (let r = 0; r <= rows; r++) {
    ctx.beginPath();
    ctx.moveTo(left, top + r * ch);
    ctx.lineTo(left + w, top + r * ch);
    ctx.stroke();
  }
  for (let c = 0; c <= cols; c++) {
    ctx.beginPath();
    ctx.moveTo(left + c * cw, top);
    ctx.lineTo(left + c * cw, top + h);
    ctx.stroke();
  }
  // 单元
  for (const cell of cells) {
    const r = asNumber(cell.row, 0);
    const c = asNumber(cell.col, 0);
    const role = asString(cell.color_role, "muted");
    ctx.fillStyle = withAlpha(roleColor(role, env.theme), 0.7);
    ctx.fillRect(left + c * cw + 2, top + r * ch + 2, cw - 4, ch - 4);
    const value = cell.value;
    if (value !== null && value !== undefined) {
      drawText(ctx, String(value), left + c * cw + cw / 2, top + r * ch + ch / 2, {
        color: env.theme.foreground, size: 10, weight: 600,
      });
    }
  }
};

const drawDualTrack: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const tracks = asArray<JsonObject>(p.params.tracks);
  if (tracks.length === 0) return;
  const w = pos.width ?? 400, h = pos.height ?? 80;
  const trackH = h / tracks.length;
  const { ctx } = env;
  tracks.forEach((t, ti) => {
    const lane = asString(t.lane, "");
    const label = asString(t.label, lane);
    const blocks = asArray<JsonObject>(t.blocks);
    const color = roleColor(lane === "attack" ? "danger" : "primary", env.theme);
    const y = pos.y - h / 2 + ti * trackH + trackH / 2;
    drawText(ctx, label, pos.x - w / 2 - 4, y, { color, size: 10, align: "right" });
    // 区块
    const bw = Math.min(40, (w - 20) / Math.max(1, blocks.length));
    blocks.forEach((b, i) => {
      const x = pos.x - w / 2 + 20 + i * (bw + 4);
      ctx.fillStyle = withAlpha(color, 0.65);
      ctx.fillRect(x, y - 10, bw, 20);
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.strokeRect(x, y - 10, bw, 20);
      const bLabel = asString(b.label, "");
      if (bLabel) drawText(ctx, bLabel, x + bw / 2, y, { color: env.theme.foreground, size: 9, weight: 600 });
    });
  });
};

const drawTimeWheel: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const slots = Math.max(1, asNumber(p.params.slots, 1));
  const current = asNumber(p.params.current_slot, 0);
  const r = pos.radius ?? 40;
  const { ctx } = env;
  ctx.strokeStyle = withAlpha(env.theme.muted, 0.4);
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, r, 0, Math.PI * 2);
  ctx.stroke();
  for (let i = 0; i < slots; i++) {
    const angle = (Math.PI * 2 * i) / slots - Math.PI / 2;
    const x = pos.x + r * Math.cos(angle);
    const y = pos.y + r * Math.sin(angle);
    ctx.beginPath();
    ctx.arc(x, y, 4, 0, Math.PI * 2);
    ctx.fillStyle = i === current ? env.theme.accent : env.theme.muted;
    ctx.fill();
  }
  drawText(ctx, `${current}/${slots}`, pos.x, pos.y, { color: env.theme.foreground, size: 11, weight: 600 });
};

const drawPieChart: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const segments = asArray<JsonObject>(p.params.segments);
  if (segments.length === 0) return;
  const total = segments.reduce((s, seg) => s + asNumber(seg.value, 0), 0) || 1;
  const r = pos.radius ?? 40;
  const { ctx } = env;
  let angle = -Math.PI / 2;
  for (const seg of segments) {
    const v = asNumber(seg.value, 0);
    const role = asString(seg.color_role, "accent");
    const sweep = (v / total) * Math.PI * 2;
    ctx.beginPath();
    ctx.moveTo(pos.x, pos.y);
    ctx.arc(pos.x, pos.y, r, angle, angle + sweep);
    ctx.closePath();
    ctx.fillStyle = withAlpha(roleColor(role, env.theme), 0.75);
    ctx.fill();
    angle += sweep;
  }
};

const drawSankeyFlow: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const flows = asArray<JsonObject>(p.params.flows);
  const w = pos.width ?? 300, h = pos.height ?? 80;
  const left = pos.x - w / 2, top = pos.y - h / 2;
  const { ctx } = env;
  flows.forEach((f, i) => {
    const value = asNumber(f.value, 1);
    const thickness = clamp(value, 1, 20);
    const y = top + (i + 0.5) * (h / Math.max(1, flows.length));
    ctx.strokeStyle = withAlpha(env.theme.accent, 0.55);
    ctx.lineWidth = thickness;
    ctx.beginPath();
    ctx.moveTo(left, y);
    ctx.bezierCurveTo(left + w * 0.4, y, left + w * 0.6, y, left + w, y);
    ctx.stroke();
  });
  ctx.lineWidth = 1;
};

const drawHeatMap: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const rows = Math.max(1, asNumber(p.params.rows, 1));
  const cols = Math.max(1, asNumber(p.params.cols, 1));
  const cells = asArray<JsonObject>(p.params.cells);
  const w = pos.width ?? 200, h = pos.height ?? 120;
  const cw = w / cols, ch = h / rows;
  const left = pos.x - w / 2, top = pos.y - h / 2;
  const { ctx } = env;
  for (const c of cells) {
    const r = asNumber(c.row, 0);
    const co = asNumber(c.col, 0);
    const v = clamp(asNumber(c.value, 0), 0, 1);
    ctx.fillStyle = withAlpha(env.theme.danger, v);
    ctx.fillRect(left + co * cw, top + r * ch, cw - 1, ch - 1);
  }
};

const drawMempoolSlot: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const txId = asString(p.params.tx_id, "");
  const label = asString(p.params.label, txId.slice(0, 6));
  const w = pos.width ?? 48, h = pos.height ?? 24;
  const { ctx } = env;
  ctx.fillStyle = withAlpha(env.theme.accent, 0.25);
  ctx.fillRect(pos.x - w / 2 + 1, pos.y - h / 2 + 1, w - 2, h - 2);
  ctx.strokeStyle = env.theme.accent;
  ctx.lineWidth = 1;
  ctx.strokeRect(pos.x - w / 2 + 1, pos.y - h / 2 + 1, w - 2, h - 2);
  drawText(ctx, label, pos.x, pos.y, { color: env.theme.foreground, size: 9 });
};

const drawBridgeTrack: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const left = asArray<JsonObject>(p.params.left_chain);
  const right = asArray<JsonObject>(p.params.right_chain);
  const w = pos.width ?? 400, h = pos.height ?? 80;
  const ox = pos.x - w / 2, oy = pos.y - h / 2;
  const { ctx } = env;
  // 左链
  drawChainRow(ctx, env.theme.accent, ox + 10, oy + h * 0.25, w * 0.35, left);
  // 右链
  drawChainRow(ctx, env.theme.success, ox + w * 0.55, oy + h * 0.75, w * 0.35, right);
  // 桥梁连接
  ctx.strokeStyle = withAlpha(env.theme.warning, 0.6);
  ctx.lineWidth = 1.5;
  ctx.setLineDash([4, 3]);
  ctx.beginPath();
  ctx.moveTo(ox + w * 0.45, oy + h * 0.25);
  ctx.lineTo(ox + w * 0.55, oy + h * 0.75);
  ctx.stroke();
  ctx.setLineDash([]);
};

function drawChainRow(
  ctx: CanvasRenderingContext2D, color: string,
  x: number, y: number, width: number, blocks: JsonObject[],
): void {
  if (blocks.length === 0) return;
  const bw = Math.min(32, width / blocks.length);
  blocks.forEach((b, i) => {
    const bx = x + i * (bw + 3);
    ctx.fillStyle = withAlpha(color, 0.55);
    ctx.fillRect(bx, y - 10, bw, 20);
    ctx.strokeStyle = color;
    ctx.lineWidth = 1;
    ctx.strokeRect(bx, y - 10, bw, 20);
    const lbl = asString((b as JsonObject).label as JsonValue, "");
    if (lbl) drawText(ctx, lbl.slice(0, 4), bx + bw / 2, y, { color: "#e5f2ff", size: 8 });
  });
}

const drawCodeMarker: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const line = asNumber(p.params.line_number, 0);
  const label = asString(p.params.label, "PC");
  const { ctx } = env;
  ctx.fillStyle = env.theme.warning;
  // 三角箭头指向代码块行（实际代码块在 DOM，这里只画一个浮标作为可视提示）
  ctx.beginPath();
  ctx.moveTo(pos.x - 8, pos.y);
  ctx.lineTo(pos.x, pos.y - 6);
  ctx.lineTo(pos.x, pos.y + 6);
  ctx.closePath();
  ctx.fill();
  drawText(ctx, `${label} L${line}`, pos.x + 6, pos.y, { color: env.theme.warning, size: 10, align: "left" });
};

const drawPartitionZone: PrimitiveDrawer = (p, env) => {
  const vertices = asArray<JsonObject>(p.params.vertices);
  if (vertices.length < 3) return;
  const main = env.resolvedLayout.main;
  const { ctx } = env;
  ctx.fillStyle = withAlpha(env.theme.danger, 0.12);
  ctx.strokeStyle = withAlpha(env.theme.danger, 0.55);
  ctx.lineWidth = 1.5;
  ctx.setLineDash([5, 4]);
  ctx.beginPath();
  vertices.forEach((v, i) => {
    const x = main.x + clamp(asNumber(v.x, 0), 0, 1) * main.width;
    const y = main.y + clamp(asNumber(v.y, 0), 0, 1) * main.height;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  ctx.closePath();
  ctx.fill();
  ctx.stroke();
  ctx.setLineDash([]);
  const label = asString(p.params.label, "");
  if (label && vertices[0]) {
    const x = main.x + clamp(asNumber(vertices[0].x, 0), 0, 1) * main.width;
    const y = main.y + clamp(asNumber(vertices[0].y, 0), 0, 1) * main.height;
    drawText(ctx, label, x + 6, y - 6, { color: env.theme.danger, size: 10, align: "left" });
  }
};

const drawCurvePoint: PrimitiveDrawer = (p, env) => {
  const pos = requirePos(p, env);
  const label = asString(p.params.label, "");
  const { ctx } = env;
  ctx.fillStyle = env.theme.accent;
  ctx.beginPath();
  ctx.arc(pos.x, pos.y, 4, 0, Math.PI * 2);
  ctx.fill();
  if (label) drawText(ctx, label, pos.x + 6, pos.y - 6, { color: env.theme.accent, size: 10, align: "left" });
};

// ============================================================
// 注册表（仅 33 canvas 原语）
// ============================================================

/** 画在 canvas 的 33 个原语 type。 */
export const CANVAS_PRIMITIVE_TYPES: ReadonlySet<PrimitiveType> = new Set<PrimitiveType>([
  // 几何 8
  "node", "edge", "bar", "curve", "polygon", "area", "grid_cell", "ring",
  // 动效 7
  "particle_stream", "burst", "pulse", "trail", "glow", "shake", "shift_animation",
  // 状态 7（不含 error_overlay）
  "phase_progress", "progress_bar", "target_zone", "link_indicator",
  "external_event_marker", "verify_path_highlight", "risk_gauge",
  // 领域 11
  "vote_matrix", "dual_track", "time_wheel", "pie_chart", "sankey_flow", "heat_map",
  "mempool_slot", "bridge_track", "code_marker", "partition_zone", "curve_point",
]);

/** 33 个 canvas drawer 注册表；renderer 严格只用此表查找。 */
export const PRIMITIVE_DRAWER_MAP: ReadonlyMap<PrimitiveType, PrimitiveDrawer> = new Map<PrimitiveType, PrimitiveDrawer>([
  // 几何
  ["node", drawNode],
  ["edge", drawEdge],
  ["bar", drawBar],
  ["curve", drawCurve],
  ["polygon", drawPolygon],
  ["area", drawArea],
  ["grid_cell", drawGridCell],
  ["ring", drawRing],
  // 动效
  ["particle_stream", drawParticleStream],
  ["burst", drawBurst],
  ["pulse", drawPulse],
  ["trail", drawTrail],
  ["glow", drawGlow],
  ["shake", drawShake],
  ["shift_animation", drawShiftAnimation],
  // 状态
  ["phase_progress", drawPhaseProgress],
  ["progress_bar", drawProgressBar],
  ["target_zone", drawTargetZone],
  ["link_indicator", drawLinkIndicator],
  ["external_event_marker", drawExternalEventMarker],
  ["verify_path_highlight", drawVerifyPathHighlight],
  ["risk_gauge", drawRiskGauge],
  // 领域
  ["vote_matrix", drawVoteMatrix],
  ["dual_track", drawDualTrack],
  ["time_wheel", drawTimeWheel],
  ["pie_chart", drawPieChart],
  ["sankey_flow", drawSankeyFlow],
  ["heat_map", drawHeatMap],
  ["mempool_slot", drawMempoolSlot],
  ["bridge_track", drawBridgeTrack],
  ["code_marker", drawCodeMarker],
  ["partition_zone", drawPartitionZone],
  ["curve_point", drawCurvePoint],
]);
