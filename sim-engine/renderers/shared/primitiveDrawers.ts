import type { Primitive, PrimitiveType, RenderContext, RendererTheme } from "./types.js";
import { asNumber, asString, clamp } from "./utils.js";

/**
 * DrawEnvironment 渲染器绘制原语时可访问的环境。
 */
export interface DrawEnvironment {
  ctx: CanvasRenderingContext2D;
  context: RenderContext;
  theme: RendererTheme;
  highlightIds: Set<string>;
  fireIds: Set<string>;
  now: number;
}

/**
 * PrimitiveDrawFn 单个原语绘制函数签名。
 */
export type PrimitiveDrawFn = (p: Primitive, env: DrawEnvironment) => void;

// ============================================================
// 工具函数
// ============================================================

function p_num(p: Primitive, key: string, fallback = 0): number {
  const v = p.params[key];
  return typeof v === "number" ? v : fallback;
}

function p_str(p: Primitive, key: string, fallback = ""): string {
  const v = p.params[key];
  return typeof v === "string" ? v : fallback;
}

function p_color(p: Primitive, key: string, env: DrawEnvironment): string {
  const v = p.params[key];
  return typeof v === "string" && v.length > 0 ? v : env.theme.accent;
}

function p_arr(p: Primitive, key: string): unknown[] {
  const v = p.params[key];
  return Array.isArray(v) ? v : [];
}

function isHighlighted(p: Primitive, env: DrawEnvironment): boolean {
  return env.highlightIds.has(p.id);
}

function highlightGlow(p: Primitive, env: DrawEnvironment, overrideColor?: string, blur = 16): void {
  const isFire = env.fireIds.has(p.id);
  const isHigh = isHighlighted(p, env);
  if (!isHigh && !isFire) return;
  
  // Fire状态给非常强的白/蓝脉冲，Highlight给警告色
  env.ctx.shadowColor = isFire ? "#ffffff" : (overrideColor || env.theme.warning);
  env.ctx.shadowBlur = isFire ? blur * 1.5 : blur;
}

function resetGlow(env: DrawEnvironment): void {
  env.ctx.shadowColor = "transparent";
  env.ctx.shadowBlur = 0;
}

// 辅助方法：Hex转rgba
function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16) || 0;
  const g = parseInt(hex.slice(3, 5), 16) || 0;
  const b = parseInt(hex.slice(5, 7), 16) || 0;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// ============================================================
// 几何类（8）
// ============================================================

function drawNode(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x", 100);
  const y = p_num(p, "y", 100);
  const radius = p_num(p, "radius", 24);
  const label = p_str(p, "label");
  const status = p_str(p, "status", "normal");
  const baseColor = status === "fault" ? theme.danger
    : status === "leader" ? theme.success
    : status === "active" ? theme.accent
    : p_color(p, "color", env);

  // 极客风外发光层
  ctx.save();
  highlightGlow(p, env, baseColor, 20);
  
  // 节点渐变底色
  const grad = ctx.createRadialGradient(x - radius * 0.3, y - radius * 0.3, radius * 0.1, x, y, radius);
  grad.addColorStop(0, hexToRgba(baseColor, 0.9));
  grad.addColorStop(1, hexToRgba(baseColor, 0.4));
  
  ctx.fillStyle = grad;
  ctx.strokeStyle = baseColor;
  ctx.lineWidth = 2;
  
  ctx.beginPath();
  ctx.arc(x, y, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.stroke();
  
  // 高亮呼吸环
  if (status === "leader" || status === "active") {
    const pulse = 1 + 0.1 * Math.sin(env.now * 0.003);
    ctx.strokeStyle = hexToRgba(baseColor, 0.4 * pulse);
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.arc(x, y, radius + 6 * pulse, 0, Math.PI * 2);
    ctx.stroke();
  }
  
  ctx.restore();

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 12px sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(label, x, y + radius + 16);
    ctx.textAlign = "start";
    ctx.textBaseline = "alphabetic";
  }
}

function drawEdge(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme, now } = env;
  const x1 = p_num(p, "x1");
  const y1 = p_num(p, "y1");
  const x2 = p_num(p, "x2");
  const y2 = p_num(p, "y2");
  const width = p_num(p, "width", 2);
  const dashed = p.params["dashed"] === true;
  const color = p_color(p, "color", env);
  const label = p_str(p, "label");

  ctx.save();
  highlightGlow(p, env, color, 12);
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  
  // 赛博流光虚线
  if (dashed) {
    ctx.setLineDash([8, 6]);
    ctx.lineDashOffset = -(now * 0.05) % 14;
  }
  
  ctx.beginPath();
  ctx.moveTo(x1, y1);
  ctx.lineTo(x2, y2);
  ctx.stroke();
  
  if (dashed) {
    ctx.setLineDash([]);
    ctx.lineDashOffset = 0;
  }
  ctx.restore();

  if (label) {
    const mx = (x1 + x2) / 2;
    const my = (y1 + y2) / 2;
    
    // 标签背景胶囊
    ctx.font = "11px sans-serif";
    const tw = ctx.measureText(label).width;
    ctx.fillStyle = "rgba(9, 17, 31, 0.8)";
    ctx.beginPath();
    ctx.roundRect(mx - tw/2 - 4, my - 10, tw + 8, 18, 4);
    ctx.fill();
    
    ctx.fillStyle = theme.muted;
    ctx.textAlign = "center";
    ctx.fillText(label, mx, my + 3);
    ctx.textAlign = "start";
  }
}

function drawBar(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 32);
  const height = p_num(p, "height", 100);
  const value = clamp(p_num(p, "value", 1), 0, 1);
  const color = p_color(p, "color", env);
  const label = p_str(p, "label");

  const barH = height * value;
  
  // 玻璃态槽槽底
  ctx.fillStyle = "rgba(18, 29, 48, 0.6)";
  ctx.strokeStyle = "rgba(143, 163, 191, 0.2)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 4);
  ctx.fill();
  ctx.stroke();

  ctx.save();
  highlightGlow(p, env, color, 12);
  
  // 渐变柱子
  const grad = ctx.createLinearGradient(x, y + height, x, y + height - barH);
  grad.addColorStop(0, hexToRgba(color, 0.6));
  grad.addColorStop(1, color);
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.roundRect(x, y + height - barH, width, barH, [0, 0, 4, 4]);
  ctx.fill();
  
  // 顶部高光线
  if (barH > 2) {
    ctx.fillStyle = "rgba(255,255,255,0.4)";
    ctx.fillRect(x, y + height - barH, width, 2);
  }
  ctx.restore();

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "11px sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(label, x + width / 2, y + height + 16);
    ctx.textAlign = "start";
  }
}

function drawCurve(p: Primitive, env: DrawEnvironment): void {
  const { ctx } = env;
  const points = p_arr(p, "points");
  const color = p_color(p, "color", env);
  const width = p_num(p, "width", 2);

  if (points.length < 2) return;
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.beginPath();
  for (let i = 0; i < points.length; i++) {
    const pt = points[i] as { x?: number; y?: number } | undefined;
    if (!pt) continue;
    const px = typeof pt.x === "number" ? pt.x : 0;
    const py = typeof pt.y === "number" ? pt.y : 0;
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.stroke();
}

function drawPolygon(p: Primitive, env: DrawEnvironment): void {
  const { ctx } = env;
  const points = p_arr(p, "points");
  const color = p_color(p, "fill", env);
  const stroke = p_str(p, "stroke");

  if (points.length < 3) return;
  ctx.fillStyle = color;
  ctx.beginPath();
  for (let i = 0; i < points.length; i++) {
    const pt = points[i] as { x?: number; y?: number } | undefined;
    if (!pt) continue;
    const px = typeof pt.x === "number" ? pt.x : 0;
    const py = typeof pt.y === "number" ? pt.y : 0;
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.closePath();
  ctx.fill();
  if (stroke) {
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 1;
    ctx.stroke();
  }
}

function drawArea(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 120);
  const height = p_num(p, "height", 80);
  const color = p_color(p, "color", env);
  const opacity = p_num(p, "opacity", 0.15);
  const label = p_str(p, "label");

  ctx.save();
  highlightGlow(p, env, color, 12);
  
  // 赛博分区背景
  const grad = ctx.createLinearGradient(x, y, x, y + height);
  grad.addColorStop(0, hexToRgba(color, opacity));
  grad.addColorStop(1, hexToRgba(color, opacity * 0.3));
  
  ctx.fillStyle = grad;
  ctx.strokeStyle = hexToRgba(color, 0.8);
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 8);
  ctx.fill();
  ctx.stroke();
  
  // 角落装饰框
  const decoL = 12;
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.beginPath();
  // 左上
  ctx.moveTo(x, y + decoL); ctx.lineTo(x, y); ctx.lineTo(x + decoL, y);
  // 右上
  ctx.moveTo(x + width - decoL, y); ctx.lineTo(x + width, y); ctx.lineTo(x + width, y + decoL);
  // 左下
  ctx.moveTo(x, y + height - decoL); ctx.lineTo(x, y + height); ctx.lineTo(x + decoL, y + height);
  // 右下
  ctx.moveTo(x + width - decoL, y + height); ctx.lineTo(x + width, y + height); ctx.lineTo(x + width, y + height - decoL);
  ctx.stroke();

  ctx.restore();

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 11px sans-serif";
    ctx.fillText(label, x + 8, y + 18);
  }
}

function drawGridCell(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const size = p_num(p, "size", 32);
  const color = p_color(p, "color", env);
  const label = p_str(p, "label");

  ctx.save();
  highlightGlow(p, env, color, 12);
  
  // 单元格赛博格栅效果
  ctx.fillStyle = hexToRgba(color, 0.15);
  ctx.fillRect(x, y, size, size);
  
  // 边框加亮
  ctx.strokeStyle = hexToRgba(color, 0.8);
  ctx.lineWidth = 1;
  ctx.strokeRect(x, y, size, size);
  
  // 四角装饰小方块
  ctx.fillStyle = color;
  ctx.fillRect(x, y, 3, 3);
  ctx.fillRect(x + size - 3, y, 3, 3);
  ctx.fillRect(x, y + size - 3, 3, 3);
  ctx.fillRect(x + size - 3, y + size - 3, 3, 3);
  
  ctx.restore();

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 10px monospace";
    ctx.textAlign = "center";
    ctx.fillText(label, x + size / 2, y + size / 2 + 4);
    ctx.textAlign = "start";
  }
}

function drawRing(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const radius = p_num(p, "radius", 60);
  const startAngle = p_num(p, "start_angle", 0);
  const endAngle = p_num(p, "end_angle", Math.PI * 2);
  const color = p_color(p, "color", env);
  const width = p_num(p, "width", 8);

  ctx.save();
  highlightGlow(p, env, color, 12);
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.lineCap = "round";
  
  // 环状动态光泽
  const grad = ctx.createConicGradient(now * 0.002, cx, cy);
  grad.addColorStop(0, color);
  grad.addColorStop(0.5, hexToRgba(color, 0.2));
  grad.addColorStop(1, color);
  
  ctx.strokeStyle = grad;
  ctx.beginPath();
  ctx.arc(cx, cy, radius, startAngle, endAngle);
  ctx.stroke();
  ctx.restore();
}

// ============================================================
// 动效类（7）
// ============================================================

function drawParticleStream(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const x1 = p_num(p, "x1");
  const y1 = p_num(p, "y1");
  const x2 = p_num(p, "x2");
  const y2 = p_num(p, "y2");
  const color = p_color(p, "color", env);
  const count = p_num(p, "count", 8);
  const speed = p_num(p, "speed", 1);

  const dx = x2 - x1;
  const dy = y2 - y1;
  
  ctx.save();
  ctx.shadowColor = color;
  ctx.shadowBlur = 8;
  ctx.fillStyle = color;
  
  for (let i = 0; i < count; i++) {
    const t = ((now * speed * 0.001 + i / count) % 1);
    // 使用 easeOutQuad 缓动，产生粒子在结尾减速并淡出的效果
    const eased = t * (2 - t);
    const px = x1 + dx * eased;
    const py = y1 + dy * eased;
    
    ctx.globalAlpha = 1 - Math.pow(eased, 2);
    ctx.beginPath();
    ctx.arc(px, py, 3 - eased * 1.5, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.restore();
}

function drawBurst(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const color = p_color(p, "color", env);
  const maxRadius = p_num(p, "radius", 40);
  const startTime = p_num(p, "start_time", 0);
  const duration = p_num(p, "duration", 600);

  const elapsed = now - startTime;
  if (elapsed < 0 || elapsed > duration) return;
  const progress = elapsed / duration;
  
  // easeOutQuart 缓动，爆炸感
  const eased = 1 - Math.pow(1 - progress, 4);
  const r = maxRadius * eased;

  ctx.save();
  ctx.strokeStyle = color;
  ctx.fillStyle = hexToRgba(color, 0.2);
  ctx.shadowColor = color;
  ctx.shadowBlur = 15;
  ctx.lineWidth = 3 * (1 - progress);
  ctx.globalAlpha = 1 - progress;
  
  ctx.beginPath();
  ctx.arc(cx, cy, r, 0, Math.PI * 2);
  ctx.fill();
  ctx.stroke();
  ctx.restore();
}

function drawPulse(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const color = p_color(p, "color", env);
  const radius = p_num(p, "radius", 30);
  const period = p_num(p, "period", 1200);

  const phase = (now % period) / period;
  
  // 核心
  ctx.fillStyle = color;
  ctx.globalAlpha = 0.5 + 0.5 * Math.sin(phase * Math.PI * 2);
  ctx.beginPath();
  ctx.arc(cx, cy, radius * 0.8, 0, Math.PI * 2);
  ctx.fill();
  
  // 脉冲波纹
  ctx.save();
  ctx.strokeStyle = color;
  ctx.shadowColor = color;
  ctx.shadowBlur = 10;
  const ripplePhase = (now % (period * 1.5)) / (period * 1.5);
  ctx.globalAlpha = 1 - ripplePhase;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(cx, cy, radius * (0.8 + 0.8 * ripplePhase), 0, Math.PI * 2);
  ctx.stroke();
  ctx.restore();
}

function drawTrail(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const points = p_arr(p, "points");
  const color = p_color(p, "color", env);
  const duration = p_num(p, "duration", 1500);
  const startTime = p_num(p, "start_time", 0);

  const elapsed = now - startTime;
  if (elapsed < 0 || points.length < 2) return;
  const progress = clamp(elapsed / duration, 0, 1);
  
  const totalLength = points.length - 1;
  const currentIndex = progress * totalLength;
  const tailLength = Math.max(1, totalLength * 0.3); // 尾巴长度占 30%

  ctx.save();
  ctx.shadowColor = color;
  ctx.shadowBlur = 10;
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  
  for (let i = 1; i <= Math.ceil(currentIndex) && i < points.length; i++) {
    const ptPrev = points[i-1] as { x?: number; y?: number };
    const ptCurr = points[i] as { x?: number; y?: number };
    if (!ptPrev || !ptCurr) continue;
    
    // 计算当前线段相对于尾巴的透明度和粗细
    const age = currentIndex - i;
    if (age > tailLength) continue;
    
    const intensity = 1 - (age / tailLength);
    
    ctx.strokeStyle = hexToRgba(color, intensity);
    ctx.lineWidth = 1 + 3 * intensity;
    ctx.beginPath();
    ctx.moveTo(typeof ptPrev.x === "number" ? ptPrev.x : 0, typeof ptPrev.y === "number" ? ptPrev.y : 0);
    ctx.lineTo(typeof ptCurr.x === "number" ? ptCurr.x : 0, typeof ptCurr.y === "number" ? ptCurr.y : 0);
    ctx.stroke();
  }
  ctx.restore();
}

function drawGlow(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const radius = p_num(p, "radius", 20);
  const color = p_color(p, "color", env);
  const intensity = p_num(p, "intensity", 0.6);

  ctx.save();
  const pulse = 1 + 0.1 * Math.sin(now * 0.003); // 微呼吸
  
  const grad = ctx.createRadialGradient(cx, cy, 0, cx, cy, radius * pulse);
  grad.addColorStop(0, hexToRgba(color, intensity));
  grad.addColorStop(0.4, hexToRgba(color, intensity * 0.5));
  grad.addColorStop(1, hexToRgba(color, 0));

  ctx.fillStyle = grad;
  ctx.globalCompositeOperation = "screen"; // 叠加提亮层
  ctx.beginPath();
  ctx.arc(cx, cy, radius * pulse, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
}

function drawShake(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const targetId = p_str(p, "target_id");
  const amplitude = p_num(p, "amplitude", 4);
  const duration = p_num(p, "duration", 400);
  const startTime = p_num(p, "start_time", 0);

  const elapsed = now - startTime;
  if (elapsed < 0 || elapsed > duration) return;
  const decay = 1 - elapsed / duration;
  const offsetX = Math.sin(elapsed * 0.05) * amplitude * decay;
  const offsetY = Math.cos(elapsed * 0.07) * amplitude * decay * 0.5;
  ctx.translate(offsetX, offsetY);
  void targetId;
}

function drawShiftAnimation(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const x1 = p_num(p, "from_x");
  const y1 = p_num(p, "from_y");
  const x2 = p_num(p, "to_x", 100);
  const y2 = p_num(p, "to_y", 100);
  const duration = p_num(p, "duration", 500);
  const startTime = p_num(p, "start_time", 0);
  const color = p_color(p, "color", env);
  const radius = p_num(p, "radius", 8);

  const elapsed = now - startTime;
  if (elapsed < 0) return;
  const t = clamp(elapsed / duration, 0, 1);
  const eased = t * (2 - t);
  const px = x1 + (x2 - x1) * eased;
  const py = y1 + (y2 - y1) * eased;

  ctx.save();
  ctx.shadowColor = color;
  ctx.shadowBlur = 12;
  ctx.fillStyle = color;
  ctx.globalAlpha = t < 1 ? 1 : 0;
  
  // 主体
  ctx.beginPath();
  ctx.arc(px, py, radius, 0, Math.PI * 2);
  ctx.fill();
  
  // 运动拖影
  if (t > 0 && t < 1) {
    ctx.globalAlpha = 0.5;
    ctx.beginPath();
    ctx.arc(px - (x2-x1)*0.05, py - (y2-y1)*0.05, radius * 0.8, 0, Math.PI * 2);
    ctx.fill();
  }
  ctx.restore();
}

// ============================================================
// 布局类（6）
// ============================================================

function drawHorizontalLane(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 600);
  const height = p_num(p, "height", 48);
  const label = p_str(p, "label");

  ctx.fillStyle = "rgba(18, 29, 48, 0.4)";
  ctx.fillRect(x, y, width, height);
  ctx.strokeStyle = theme.grid;
  ctx.strokeRect(x, y, width, height);

  if (label) {
    ctx.fillStyle = theme.muted;
    ctx.font = "11px sans-serif";
    ctx.fillText(label, x + 8, y + height / 2 + 4);
  }
}

function drawStack(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 120);
  const items = p_arr(p, "items");
  const itemHeight = p_num(p, "item_height", 28);

  items.forEach((item, i) => {
    const iy = y + i * (itemHeight + 2);
    const label = typeof item === "string" ? item : asString((item as Record<string, unknown>)?.label);
    ctx.fillStyle = "rgba(18, 29, 48, 0.7)";
    ctx.fillRect(x, iy, width, itemHeight);
    ctx.strokeStyle = theme.grid;
    ctx.strokeRect(x, iy, width, itemHeight);
    if (label) {
      ctx.fillStyle = theme.foreground;
      ctx.font = "11px sans-serif";
      ctx.fillText(label, x + 6, iy + itemHeight / 2 + 4);
    }
  });
}

function drawRingLayout(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const cx = p_num(p, "cx", 300);
  const cy = p_num(p, "cy", 300);
  const radius = p_num(p, "radius", 100);
  const count = p_num(p, "count", 6);

  ctx.strokeStyle = theme.grid;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.arc(cx, cy, radius, 0, Math.PI * 2);
  ctx.stroke();

  for (let i = 0; i < count; i++) {
    const angle = (i / count) * Math.PI * 2 - Math.PI / 2;
    const nx = cx + radius * Math.cos(angle);
    const ny = cy + radius * Math.sin(angle);
    ctx.fillStyle = theme.accent;
    ctx.beginPath();
    ctx.arc(nx, ny, 6, 0, Math.PI * 2);
    ctx.fill();
  }
}

function drawTreeLayout(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x", 100);
  const y = p_num(p, "y", 60);
  const levels = p_num(p, "levels", 3);
  const hGap = p_num(p, "h_gap", 80);
  const vGap = p_num(p, "v_gap", 60);

  for (let level = 0; level < levels; level++) {
    const nodesInLevel = Math.pow(2, level);
    const levelWidth = (nodesInLevel - 1) * hGap;
    const startX = x - levelWidth / 2;
    for (let n = 0; n < nodesInLevel; n++) {
      const nx = startX + n * hGap;
      const ny = y + level * vGap;
      ctx.fillStyle = level === 0 ? theme.accent : theme.muted;
      ctx.beginPath();
      ctx.arc(nx, ny, 8, 0, Math.PI * 2);
      ctx.fill();
      if (level > 0) {
        const parentIdx = Math.floor(n / 2);
        const parentNodesInLevel = Math.pow(2, level - 1);
        const parentWidth = (parentNodesInLevel - 1) * hGap;
        const parentStartX = x - parentWidth / 2;
        const px = parentStartX + parentIdx * hGap;
        const py = y + (level - 1) * vGap;
        ctx.strokeStyle = theme.grid;
        ctx.beginPath();
        ctx.moveTo(px, py + 8);
        ctx.lineTo(nx, ny - 8);
        ctx.stroke();
      }
    }
  }
}

function drawGraphLayout(p: Primitive, env: DrawEnvironment): void {
  drawRingLayout(p, env);
}

function drawMatrixLayout(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const rows = p_num(p, "rows", 4);
  const cols = p_num(p, "cols", 4);
  const cellSize = p_num(p, "cell_size", 32);
  const gap = p_num(p, "gap", 2);

  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      const cx = x + c * (cellSize + gap);
      const cy = y + r * (cellSize + gap);
      ctx.fillStyle = "rgba(18, 29, 48, 0.5)";
      ctx.fillRect(cx, cy, cellSize, cellSize);
      ctx.strokeStyle = theme.grid;
      ctx.strokeRect(cx, cy, cellSize, cellSize);
    }
  }
}

// ============================================================
// 数据展示类（7）
// ============================================================

function drawLabel(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const text = p_str(p, "text");
  const fontSize = p_num(p, "font_size", 14);
  const color = p_color(p, "color", env);
  const bold = p.params["bold"] === true;

  ctx.fillStyle = color === env.theme.accent ? theme.foreground : color;
  ctx.font = `${bold ? "600 " : ""}${fontSize}px sans-serif`;
  ctx.fillText(text, x, y);
}

function drawTooltip(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const items = p_arr(p, "items");

  if (items.length === 0) return;
  const width = 220;
  const height = 24 + items.length * 20;
  
  ctx.save();
  // 悬浮玻璃态背景
  ctx.fillStyle = "rgba(10, 18, 30, 0.85)";
  ctx.shadowColor = "rgba(0, 0, 0, 0.5)";
  ctx.shadowBlur = 10;
  ctx.shadowOffsetY = 4;
  
  // 赛博边框
  ctx.strokeStyle = hexToRgba(theme.accent, 0.5);
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 6);
  ctx.fill();
  ctx.stroke();

  // 顶部装饰线
  ctx.fillStyle = theme.accent;
  ctx.beginPath();
  ctx.roundRect(x, y, width, 4, [6, 6, 0, 0]);
  ctx.fill();

  items.forEach((item, i) => {
    const row = item as { label?: string; value?: string } | undefined;
    if (!row) return;
    const ly = y + 20 + i * 20;
    ctx.fillStyle = theme.muted;
    ctx.font = "11px sans-serif";
    ctx.fillText(asString(row.label), x + 12, ly);
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 11px monospace";
    ctx.fillText(asString(row.value), x + 110, ly);
  });
  ctx.restore();
}

function drawAnnotation(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const text = p_str(p, "text");
  const color = p_color(p, "color", env);

  ctx.font = "600 12px sans-serif";
  const tw = ctx.measureText(text).width;
  const w = Math.max(60, tw + 20);
  
  ctx.save();
  ctx.shadowColor = color;
  ctx.shadowBlur = 8;
  
  // 梯形/斜切标签底板
  ctx.fillStyle = color;
  ctx.globalAlpha = 0.9;
  ctx.beginPath();
  ctx.moveTo(x + 6, y);
  ctx.lineTo(x + w, y);
  ctx.lineTo(x + w - 6, y + 24);
  ctx.lineTo(x, y + 24);
  ctx.closePath();
  ctx.fill();
  
  ctx.globalAlpha = 1;
  ctx.fillStyle = theme.background; // 黑色文字
  ctx.textAlign = "center";
  ctx.fillText(text, x + w / 2, y + 16);
  ctx.restore();
}

function drawRegisterRow(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 280);
  const label = p_str(p, "label");
  const value = p_str(p, "value");

  ctx.save();
  highlightGlow(p, env, theme.accent, 12);
  
  // 玻璃态微渐变背景
  const grad = ctx.createLinearGradient(x, y, x, y + 24);
  grad.addColorStop(0, "rgba(24, 38, 62, 0.9)");
  grad.addColorStop(1, "rgba(14, 22, 38, 0.9)");
  
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.roundRect(x, y, width, 24, 4);
  ctx.fill();
  
  ctx.strokeStyle = "rgba(143, 163, 191, 0.25)";
  ctx.lineWidth = 1;
  ctx.stroke();

  ctx.fillStyle = theme.muted;
  ctx.font = "11px monospace";
  ctx.fillText(label, x + 10, y + 16);
  
  ctx.fillStyle = theme.foreground;
  ctx.font = "600 12px monospace";
  ctx.fillText(value, x + width * 0.4, y + 16);
  
  // 分隔线
  ctx.strokeStyle = "rgba(143, 163, 191, 0.15)";
  ctx.beginPath();
  ctx.moveTo(x + width * 0.38, y + 4);
  ctx.lineTo(x + width * 0.38, y + 20);
  ctx.stroke();
  
  ctx.restore();
}

function drawMathPipeline(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const steps = p_arr(p, "steps");
  const stepWidth = p_num(p, "step_width", 100);

  steps.forEach((step, i) => {
    const sx = x + i * (stepWidth + 20);
    const label = typeof step === "string" ? step : asString((step as Record<string, unknown>)?.label);
    ctx.fillStyle = "rgba(18, 29, 48, 0.85)";
    ctx.strokeStyle = theme.accent;
    ctx.beginPath();
    ctx.roundRect(sx, y, stepWidth, 36, 8);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = theme.foreground;
    ctx.font = "12px sans-serif";
    ctx.fillText(label, sx + 8, y + 22);
    if (i < steps.length - 1) {
      ctx.strokeStyle = theme.muted;
      ctx.beginPath();
      ctx.moveTo(sx + stepWidth + 4, y + 18);
      ctx.lineTo(sx + stepWidth + 16, y + 18);
      ctx.stroke();
    }
  });
}

function drawCodeBlock(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 300);
  const lines = p_arr(p, "lines");
  const lineHeight = 18;
  const height = Math.max(40, lines.length * lineHeight + 16);

  ctx.save();
  highlightGlow(p, env, theme.accent, 12);
  
  // 赛博终端背景
  ctx.fillStyle = "rgba(5, 10, 20, 0.95)";
  ctx.strokeStyle = "rgba(143, 163, 191, 0.2)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 6);
  ctx.fill();
  ctx.stroke();

  // 顶部装饰条（模拟终端标题栏）
  ctx.fillStyle = "rgba(24, 38, 62, 0.8)";
  ctx.beginPath();
  ctx.roundRect(x, y, width, 12, [6, 6, 0, 0]);
  ctx.fill();
  
  // 三个小红绿灯
  ctx.fillStyle = theme.danger; ctx.beginPath(); ctx.arc(x + 10, y + 6, 3, 0, Math.PI*2); ctx.fill();
  ctx.fillStyle = theme.warning; ctx.beginPath(); ctx.arc(x + 20, y + 6, 3, 0, Math.PI*2); ctx.fill();
  ctx.fillStyle = theme.success; ctx.beginPath(); ctx.arc(x + 30, y + 6, 3, 0, Math.PI*2); ctx.fill();

  ctx.fillStyle = theme.foreground;
  ctx.font = "12px monospace";
  lines.forEach((line, i) => {
    const text = typeof line === "string" ? line : String(line ?? "");
    // 行号
    ctx.fillStyle = theme.muted;
    ctx.globalAlpha = 0.5;
    ctx.fillText(String(i + 1).padStart(2, ' '), x + 8, y + 24 + i * lineHeight);
    
    // 代码高亮简化版
    ctx.globalAlpha = 1;
    ctx.fillStyle = text.includes("return") || text.includes("function") ? theme.accent : theme.foreground;
    ctx.fillText(text, x + 32, y + 24 + i * lineHeight);
  });
  ctx.restore();
}

function drawMathFormula(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const formula = p_str(p, "formula");

  ctx.fillStyle = theme.foreground;
  ctx.font = "italic 16px serif";
  ctx.fillText(formula, x, y);
}

// ============================================================
// 状态指示类（8）
// ============================================================

function drawPhaseProgress(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 300);
  const phases = p_arr(p, "phases");
  const activeIndex = p_num(p, "active_index", 0);
  const progress = p_num(p, "progress", 0);

  if (phases.length === 0) return;
  const gap = width / Math.max(1, phases.length - 1);

  // 极客感底层线
  ctx.strokeStyle = "rgba(143, 163, 191, 0.2)";
  ctx.lineWidth = 4;
  ctx.lineCap = "round";
  ctx.beginPath();
  ctx.moveTo(x, y);
  ctx.lineTo(x + width, y);
  ctx.stroke();

  // 进度发光线
  if (activeIndex > 0 || progress > 0) {
    const currentProgressX = x + gap * (activeIndex + clamp(progress, 0, 1));
    ctx.save();
    ctx.strokeStyle = theme.accent;
    ctx.shadowColor = theme.accent;
    ctx.shadowBlur = 10;
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(Math.min(currentProgressX, x + width), y);
    ctx.stroke();
    ctx.restore();
  }

  phases.forEach((phase, i) => {
    const dotX = x + gap * i;
    const isActive = i === activeIndex;
    const isCompleted = i < activeIndex;
    const label = typeof phase === "string" ? phase : asString((phase as Record<string, unknown>)?.label);
    
    ctx.save();
    if (isCompleted) {
      ctx.fillStyle = theme.success;
      ctx.shadowColor = theme.success;
      ctx.shadowBlur = 8;
    } else if (isActive) {
      ctx.fillStyle = theme.accent;
      ctx.shadowColor = theme.accent;
      ctx.shadowBlur = 12;
    } else {
      ctx.fillStyle = "rgba(36, 52, 78, 1)";
      ctx.strokeStyle = theme.muted;
      ctx.lineWidth = 2;
    }

    ctx.beginPath();
    ctx.arc(dotX, y, isActive ? 8 : 6, 0, Math.PI * 2);
    ctx.fill();
    if (!isActive && !isCompleted) ctx.stroke();
    
    // 活跃状态的呼吸外圈
    if (isActive) {
      const pulse = 1 + 0.2 * Math.sin(env.now * 0.005);
      ctx.strokeStyle = hexToRgba(theme.accent, 0.5 * (2 - pulse));
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(dotX, y, 10 * pulse, 0, Math.PI * 2);
      ctx.stroke();
    }
    ctx.restore();

    if (label) {
      ctx.fillStyle = isActive ? theme.foreground : theme.muted;
      ctx.font = isActive ? "600 12px sans-serif" : "11px sans-serif";
      ctx.textAlign = "center";
      ctx.fillText(label, dotX, y + 24);
      ctx.textAlign = "start";
    }
  });
}

function drawProgressBar(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 200);
  const height = p_num(p, "height", 12);
  const value = clamp(p_num(p, "value", 0), 0, 1);
  const color = p_color(p, "color", env);

  ctx.save();
  // 槽底
  ctx.fillStyle = "rgba(18, 29, 48, 0.8)";
  ctx.strokeStyle = "rgba(143, 163, 191, 0.2)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, height / 2);
  ctx.fill();
  ctx.stroke();
  
  // 进度条发光
  if (value > 0) {
    ctx.shadowColor = color;
    ctx.shadowBlur = 8;
    const grad = ctx.createLinearGradient(x, y, x + width * value, y);
    grad.addColorStop(0, hexToRgba(color, 0.5));
    grad.addColorStop(1, color);
    
    ctx.fillStyle = grad;
    ctx.beginPath();
    ctx.roundRect(x, y, width * value, height, height / 2);
    ctx.fill();
  }
  ctx.restore();
}

function drawTargetZone(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme, now } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 60);
  const height = p_num(p, "height", 60);
  const hit = p.params["hit"] === true;

  const color = hit ? theme.success : theme.warning;
  
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  
  // 边框呼吸流动
  ctx.setLineDash([8, 8]);
  ctx.lineDashOffset = -(now * 0.05);
  ctx.strokeRect(x, y, width, height);
  ctx.setLineDash([]);
  
  // 四角瞄准星
  const m = 6;
  ctx.beginPath();
  ctx.moveTo(x, y + m); ctx.lineTo(x, y); ctx.lineTo(x + m, y);
  ctx.moveTo(x + width - m, y); ctx.lineTo(x + width, y); ctx.lineTo(x + width, y + m);
  ctx.moveTo(x, y + height - m); ctx.lineTo(x, y + height); ctx.lineTo(x + m, y + height);
  ctx.moveTo(x + width - m, y + height); ctx.lineTo(x + width, y + height); ctx.lineTo(x + width, y + height - m);
  ctx.stroke();

  if (hit) {
    ctx.fillStyle = color;
    ctx.globalAlpha = 0.15 + 0.05 * Math.sin(now * 0.01);
    ctx.fillRect(x, y, width, height);
  }
  ctx.restore();
}

function drawLinkIndicator(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const active = p.params["active"] !== false;
  const label = p_str(p, "label", "联动");

  const color = active ? theme.success : theme.muted;
  ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
  ctx.strokeStyle = color;
  ctx.beginPath();
  ctx.roundRect(x, y, Math.max(60, label.length * 12 + 16), 24, 12);
  ctx.fill();
  ctx.stroke();
  ctx.fillStyle = theme.foreground;
  ctx.font = "11px sans-serif";
  ctx.fillText(label, x + 8, y + 16);
}

function drawExternalEventMarker(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const label = p_str(p, "label", "外部事件");

  ctx.fillStyle = theme.warning;
  ctx.beginPath();
  ctx.moveTo(x, y - 10);
  ctx.lineTo(x + 8, y + 6);
  ctx.lineTo(x - 8, y + 6);
  ctx.closePath();
  ctx.fill();
  ctx.fillStyle = theme.foreground;
  ctx.font = "10px sans-serif";
  ctx.fillText(label, x - 20, y + 20);
}

function drawErrorOverlay(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme, now } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 200);
  const height = p_num(p, "height", 40);
  const message = p_str(p, "message", "错误");

  ctx.save();
  // 红色闪烁光晕
  const pulse = 1 + 0.3 * Math.sin(now * 0.008);
  ctx.shadowColor = theme.danger;
  ctx.shadowBlur = 12 * pulse;
  
  ctx.fillStyle = hexToRgba(theme.danger, 0.15 + 0.05 * Math.sin(now * 0.01));
  ctx.strokeStyle = theme.danger;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 4);
  ctx.fill();
  ctx.stroke();
  
  // 警告斜纹背景 (模拟警示条)
  ctx.save();
  ctx.clip();
  ctx.strokeStyle = hexToRgba(theme.danger, 0.3);
  ctx.lineWidth = 2;
  const offset = (now * 0.02) % 10;
  for (let i = -10; i < width + height; i += 10) {
    ctx.beginPath();
    ctx.moveTo(x + i + offset, y);
    ctx.lineTo(x + i - height + offset, y + height);
    ctx.stroke();
  }
  ctx.restore();

  // 文字发光
  ctx.shadowColor = "transparent";
  ctx.fillStyle = theme.danger;
  ctx.font = "600 12px sans-serif";
  ctx.fillText(message, x + 10, y + height / 2 + 4);
  ctx.restore();
}

function drawVerifyPathHighlight(p: Primitive, env: DrawEnvironment): void {
  const { ctx, now } = env;
  const points = p_arr(p, "points");
  const valid = p.params["valid"] !== false;
  const color = valid ? env.theme.success : env.theme.danger;

  if (points.length < 2) return;
  
  ctx.save();
  ctx.shadowColor = color;
  ctx.shadowBlur = 12;
  ctx.strokeStyle = color;
  ctx.lineWidth = 4;
  
  // 验证路径流光
  ctx.setLineDash([12, 8]);
  ctx.lineDashOffset = -(now * 0.08);
  
  ctx.beginPath();
  for (let i = 0; i < points.length; i++) {
    const pt = points[i] as { x?: number; y?: number } | undefined;
    if (!pt) continue;
    const px = typeof pt.x === "number" ? pt.x : 0;
    const py = typeof pt.y === "number" ? pt.y : 0;
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.stroke();
  
  ctx.setLineDash([]);
  ctx.restore();
}

function drawRiskGauge(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const cx = p_num(p, "cx", 100);
  const cy = p_num(p, "cy", 100);
  const radius = p_num(p, "radius", 40);
  const value = clamp(p_num(p, "value", 0.5), 0, 1);

  const startAngle = Math.PI * 0.8;
  const endAngle = Math.PI * 2.2;
  
  // 赛博仪表盘背景
  ctx.strokeStyle = "rgba(18, 29, 48, 0.8)";
  ctx.lineWidth = 10;
  ctx.lineCap = "round";
  ctx.beginPath();
  ctx.arc(cx, cy, radius, startAngle, endAngle);
  ctx.stroke();
  
  // 刻度线
  ctx.strokeStyle = theme.grid;
  ctx.lineWidth = 2;
  const ticks = 10;
  for (let i = 0; i <= ticks; i++) {
    const angle = startAngle + (endAngle - startAngle) * (i / ticks);
    ctx.beginPath();
    ctx.moveTo(cx + (radius - 12) * Math.cos(angle), cy + (radius - 12) * Math.sin(angle));
    ctx.lineTo(cx + (radius - 6) * Math.cos(angle), cy + (radius - 6) * Math.sin(angle));
    ctx.stroke();
  }

  const color = value > 0.75 ? theme.danger : value > 0.4 ? theme.warning : theme.success;
  const valueAngle = startAngle + (endAngle - startAngle) * value;
  
  // 发光指针条
  ctx.save();
  ctx.strokeStyle = color;
  ctx.shadowColor = color;
  ctx.shadowBlur = 12;
  ctx.lineWidth = 8;
  ctx.lineCap = "round";
  ctx.beginPath();
  if (value > 0) {
    ctx.arc(cx, cy, radius, startAngle, valueAngle);
    ctx.stroke();
  }
  ctx.restore();

  ctx.fillStyle = color;
  ctx.font = "600 16px sans-serif";
  ctx.textAlign = "center";
  ctx.fillText(`${Math.round(value * 100)}%`, cx, cy + 6);
  ctx.textAlign = "start";
}

// ============================================================
// 领域复合类（11）
// ============================================================

function drawVoteMatrix(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const rows = p_arr(p, "rows");
  const cols = p_arr(p, "cols");
  const votes = p_arr(p, "votes");
  const cellSize = p_num(p, "cell_size", 24);

  cols.forEach((col, ci) => {
    const label = typeof col === "string" ? col : `C${ci}`;
    ctx.fillStyle = theme.muted;
    ctx.font = "10px sans-serif";
    ctx.fillText(label, x + 60 + ci * (cellSize + 2), y - 4);
  });

  rows.forEach((row, ri) => {
    const label = typeof row === "string" ? row : `R${ri}`;
    ctx.fillStyle = theme.muted;
    ctx.font = "10px sans-serif";
    ctx.fillText(label, x, y + 12 + ri * (cellSize + 2));
    const rowVotes = (Array.isArray(votes[ri]) ? votes[ri] : []) as unknown[];
    (rowVotes as unknown[]).forEach((v: unknown, ci: number) => {
      const cx = x + 60 + ci * (cellSize + 2);
      const cy = y + ri * (cellSize + 2);
      ctx.fillStyle = v === true || v === 1 ? theme.success : v === false || v === 0 ? theme.danger : theme.grid;
      ctx.fillRect(cx, cy, cellSize, cellSize);
    });
  });
}

function drawDualTrack(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 400);
  const height = p_num(p, "height", 200);
  const leftLabel = p_str(p, "left_label", "Track A");
  const rightLabel = p_str(p, "right_label", "Track B");

  const half = width / 2 - 8;
  ctx.fillStyle = "rgba(18, 29, 48, 0.5)";
  ctx.fillRect(x, y, half, height);
  ctx.fillRect(x + half + 16, y, half, height);
  ctx.strokeStyle = theme.grid;
  ctx.strokeRect(x, y, half, height);
  ctx.strokeRect(x + half + 16, y, half, height);

  ctx.fillStyle = theme.foreground;
  ctx.font = "600 12px sans-serif";
  ctx.fillText(leftLabel, x + 8, y + 18);
  ctx.fillText(rightLabel, x + half + 24, y + 18);
}

function drawTimeWheel(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme, now } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const radius = p_num(p, "radius", 60);
  const segments = p_num(p, "segments", 12);
  const activeSegment = p_num(p, "active_segment", 0);

  for (let i = 0; i < segments; i++) {
    const start = (i / segments) * Math.PI * 2 - Math.PI / 2;
    const end = ((i + 1) / segments) * Math.PI * 2 - Math.PI / 2;
    ctx.fillStyle = i === activeSegment ? theme.accent : "rgba(18, 29, 48, 0.6)";
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.arc(cx, cy, radius, start, end);
    ctx.closePath();
    ctx.fill();
    ctx.strokeStyle = theme.grid;
    ctx.stroke();
  }
  void now;
}

function drawPieChart(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const cx = p_num(p, "cx", 200);
  const cy = p_num(p, "cy", 200);
  const radius = p_num(p, "radius", 60);
  const slices = p_arr(p, "slices");

  const colors = [theme.accent, theme.success, theme.warning, theme.danger, theme.muted, "#9b59b6", "#e67e22", "#1abc9c"];
  let startAngle = -Math.PI / 2;
  const total = slices.reduce<number>((sum, s) => sum + asNumber((s as Record<string, unknown>)?.value), 0);
  if (total <= 0) return;

  slices.forEach((slice, i) => {
    const s = slice as Record<string, unknown> | undefined;
    if (!s) return;
    const value = asNumber(s.value);
    const angle = (value / total) * Math.PI * 2;
    ctx.fillStyle = colors[i % colors.length] ?? theme.accent;
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.arc(cx, cy, radius, startAngle, startAngle + angle);
    ctx.closePath();
    ctx.fill();
    startAngle += angle;
  });
}

function drawSankeyFlow(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 300);
  const flows = p_arr(p, "flows");

  flows.forEach((flow, i) => {
    const f = flow as { from_y?: number; to_y?: number; thickness?: number; color?: string } | undefined;
    if (!f) return;
    const fy = y + asNumber(f.from_y, i * 40);
    const ty = y + asNumber(f.to_y, i * 40);
    const thickness = asNumber(f.thickness, 12);
    const color = typeof f.color === "string" ? f.color : theme.accent;

    ctx.fillStyle = color;
    ctx.globalAlpha = 0.6;
    ctx.beginPath();
    ctx.moveTo(x, fy);
    ctx.bezierCurveTo(x + width * 0.4, fy, x + width * 0.6, ty, x + width, ty);
    ctx.lineTo(x + width, ty + thickness);
    ctx.bezierCurveTo(x + width * 0.6, ty + thickness, x + width * 0.4, fy + thickness, x, fy + thickness);
    ctx.closePath();
    ctx.fill();
    ctx.globalAlpha = 1;
  });
}

function drawHeatMap(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const rows = p_num(p, "rows", 4);
  const cols = p_num(p, "cols", 6);
  const cellSize = p_num(p, "cell_size", 24);
  const data = p_arr(p, "data");

  for (let r = 0; r < rows; r++) {
    const rowData = (Array.isArray(data[r]) ? data[r] : []) as number[];
    for (let c = 0; c < cols; c++) {
      const value = clamp(typeof rowData[c] === "number" ? rowData[c] as number : 0, 0, 1);
      const cx = x + c * (cellSize + 1);
      const cy = y + r * (cellSize + 1);
      const r_ = Math.round(255 * value);
      const b_ = Math.round(255 * (1 - value));
      ctx.fillStyle = `rgb(${r_}, 40, ${b_})`;
      ctx.fillRect(cx, cy, cellSize, cellSize);
    }
  }
  void theme;
}

function drawMempoolSlot(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 48);
  const height = p_num(p, "height", 28);
  const label = p_str(p, "label");
  const priority = p_str(p, "priority", "normal");

  const color = priority === "high" ? theme.warning : priority === "low" ? theme.muted : theme.accent;
  
  ctx.save();
  highlightGlow(p, env, color, 12);
  
  ctx.fillStyle = hexToRgba(color, 0.15);
  ctx.strokeStyle = color;
  ctx.lineWidth = 1;
  
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, 4);
  ctx.fill();
  ctx.stroke();
  
  // 右上角小角标
  ctx.beginPath();
  ctx.moveTo(x + width - 8, y);
  ctx.lineTo(x + width, y);
  ctx.lineTo(x + width, y + 8);
  ctx.closePath();
  ctx.fillStyle = color;
  ctx.fill();

  ctx.restore();

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 10px monospace";
    ctx.textAlign = "center";
    ctx.fillText(label, x + width / 2, y + height / 2 + 3);
    ctx.textAlign = "start";
  }
}

function drawBridgeTrack(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 400);
  const leftLabel = p_str(p, "left_chain", "Chain A");
  const rightLabel = p_str(p, "right_chain", "Chain B");

  ctx.strokeStyle = theme.accent;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(x, y);
  ctx.lineTo(x + width, y);
  ctx.stroke();

  ctx.fillStyle = theme.foreground;
  ctx.font = "12px sans-serif";
  ctx.fillText(leftLabel, x, y - 8);
  ctx.textAlign = "end";
  ctx.fillText(rightLabel, x + width, y - 8);
  ctx.textAlign = "start";
}

function drawCodeMarker(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const line = p_num(p, "line", 1);
  const marker = p_str(p, "marker", "▶");
  const color = p_color(p, "color", env);

  ctx.fillStyle = color;
  ctx.font = "14px monospace";
  ctx.fillText(`${marker} L${line}`, x, y);
  void theme;
}

function drawPartitionZone(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme, now } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const width = p_num(p, "width", 150);
  const height = p_num(p, "height", 150);
  const label = p_str(p, "label", "分区");

  ctx.save();
  ctx.strokeStyle = theme.danger;
  ctx.lineWidth = 2;
  
  // 流动虚线框
  ctx.setLineDash([10, 10]);
  ctx.lineDashOffset = -(now * 0.02);
  ctx.strokeRect(x, y, width, height);
  ctx.setLineDash([]);
  
  // 闪烁背景
  const alpha = 0.04 + 0.03 * Math.sin(now * 0.005);
  ctx.fillStyle = hexToRgba(theme.danger, alpha);
  ctx.fillRect(x, y, width, height);
  
  // 左上角名称标签底色
  ctx.fillStyle = theme.danger;
  ctx.beginPath();
  ctx.moveTo(x, y);
  ctx.lineTo(x + 60, y);
  ctx.lineTo(x + 50, y + 20);
  ctx.lineTo(x, y + 20);
  ctx.closePath();
  ctx.fill();

  ctx.fillStyle = theme.background; // 文字反色
  ctx.font = "600 11px sans-serif";
  ctx.fillText(label, x + 8, y + 14);
  
  ctx.restore();
}

function drawCurvePoint(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x");
  const y = p_num(p, "y");
  const radius = p_num(p, "radius", 5);
  const color = p_color(p, "color", env);
  const label = p_str(p, "label");

  highlightGlow(p, env);
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.arc(x, y, radius, 0, Math.PI * 2);
  ctx.fill();
  resetGlow(env);

  if (label) {
    ctx.fillStyle = theme.foreground;
    ctx.font = "10px sans-serif";
    ctx.fillText(label, x + radius + 4, y + 4);
  }
}

// ============================================================
// Fallback
// ============================================================

function drawFallback(p: Primitive, env: DrawEnvironment): void {
  const { ctx, theme } = env;
  const x = p_num(p, "x", 20);
  const y = p_num(p, "y", 20);

  ctx.fillStyle = theme.muted;
  ctx.font = "11px sans-serif";
  ctx.fillText(`[未知原语: ${p.type}] id=${p.id}`, x, y);
}

// ============================================================
// 注册表
// ============================================================

/**
 * PRIMITIVE_DRAWER_MAP 全部 47 原语的绘制函数注册表。
 */
export const PRIMITIVE_DRAWER_MAP: Record<PrimitiveType, PrimitiveDrawFn> = {
  // 几何类
  node: drawNode,
  edge: drawEdge,
  bar: drawBar,
  curve: drawCurve,
  polygon: drawPolygon,
  area: drawArea,
  grid_cell: drawGridCell,
  ring: drawRing,
  // 动效类
  particle_stream: drawParticleStream,
  burst: drawBurst,
  pulse: drawPulse,
  trail: drawTrail,
  glow: drawGlow,
  shake: drawShake,
  shift_animation: drawShiftAnimation,
  // 布局类
  horizontal_lane: drawHorizontalLane,
  stack: drawStack,
  ring_layout: drawRingLayout,
  tree_layout: drawTreeLayout,
  graph_layout: drawGraphLayout,
  matrix_layout: drawMatrixLayout,
  // 数据展示类
  label: drawLabel,
  tooltip: drawTooltip,
  annotation: drawAnnotation,
  register_row: drawRegisterRow,
  math_pipeline: drawMathPipeline,
  code_block: drawCodeBlock,
  math_formula: drawMathFormula,
  // 状态指示类
  phase_progress: drawPhaseProgress,
  progress_bar: drawProgressBar,
  target_zone: drawTargetZone,
  link_indicator: drawLinkIndicator,
  external_event_marker: drawExternalEventMarker,
  error_overlay: drawErrorOverlay,
  verify_path_highlight: drawVerifyPathHighlight,
  risk_gauge: drawRiskGauge,
  // 领域复合类
  vote_matrix: drawVoteMatrix,
  dual_track: drawDualTrack,
  time_wheel: drawTimeWheel,
  pie_chart: drawPieChart,
  sankey_flow: drawSankeyFlow,
  heat_map: drawHeatMap,
  mempool_slot: drawMempoolSlot,
  bridge_track: drawBridgeTrack,
  code_marker: drawCodeMarker,
  partition_zone: drawPartitionZone,
  curve_point: drawCurvePoint,
};

/**
 * FALLBACK_DRAWER 未知原语类型的兜底绘制函数。
 */
export const FALLBACK_DRAWER: PrimitiveDrawFn = drawFallback;
