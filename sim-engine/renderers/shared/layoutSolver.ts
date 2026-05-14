/**
 * 模块：sim-engine/renderers/shared
 * 文件职责：原语布局解算器（Layout Solver）。
 *
 * 设计契约（详 framework/primitive.go:7-13）：
 *   - 后端 scenarios 推送"布局原语 + 无坐标内容原语"或"*At 变体 + 逻辑 0~1 坐标"；
 *   - 前端 PrimitiveBasedRenderer 在 dispatch 之前调用本 solver，
 *     按画布尺寸 + 父布局规则推导每个内容原语的像素坐标（写入 ResolvedLayout）；
 *   - drawer 通过 env.resolvedLayout.get(id) 取最终坐标，0~1 逻辑坐标也在此一并放大为像素。
 *
 * 支持的 6 类布局原语（与 framework/primitive.go 1:1）：
 *   - stack(items, direction)         水平 / 垂直 等距排列子节点；
 *   - ring_layout(slots)              在余下未定位 node 中取前 slots 个布到环上；
 *   - matrix_layout(rows, cols)       按 row/col 索引为 grid_cell 派发坐标；
 *   - horizontal_lane                 多条 lane 等高分割 y 轴；
 *   - tree_layout(root_id, algorithm) 用 edges 自顶向下分层；
 *   - graph_layout(algorithm, nodes)  按 algorithm 推导（circular/grid，force 暂回退 circular）。
 *
 * 三段式扫描（pass）：
 *   Pass A — *At 显式坐标归一化（含 center_x/center_y）；
 *   Pass B — 含 items 列表的布局（stack / graph_layout / tree_layout）；
 *   Pass C — 隐式占位布局（ring_layout / matrix_layout / horizontal_lane）；
 *   Pass D — HUD 类原语（phase_progress / progress_bar / target_zone / risk_gauge）顶部带状自动落位。
 *
 * 注：本 solver 不直接绘制；只产出坐标映射给 drawer 使用，drawer 自行决定外观。
 */

import type { Primitive } from "./types.js";

/** 单个原语解算结果。所有字段单位均为像素。 */
export interface ResolvedLayout {
  x: number;
  y: number;
  /** 节点半径 / 环半径；不含填充。 */
  radius?: number;
  /** 矩形宽度（bar / area / grid_cell 等）。 */
  width?: number;
  /** 矩形高度。 */
  height?: number;
}

/** Solver 内部状态。 */
interface SolverState {
  width: number;
  height: number;
  primitives: Primitive[];
  resolved: Map<string, ResolvedLayout>;
}

/** 内容节点类型（参与 ring_layout 隐式占位）。 */
const CONTENT_NODE_TYPES = new Set<string>([
  "node", "ring", "bar", "mempool_slot", "register_row", "code_block"
]);

/** HUD 风格状态原语（顶部带状自动落位）。 */
const HUD_TYPES = new Set<string>([
  "phase_progress", "progress_bar", "target_zone", "risk_gauge"
]);

/**
 * resolveLayout 主入口。
 * @param primitives RenderEnvelope.primitives 全列表
 * @param width  canvas CSS 像素宽
 * @param height canvas CSS 像素高
 * @returns id → ResolvedLayout 映射
 */
export function resolveLayout(
  primitives: Primitive[],
  width: number,
  height: number,
): Map<string, ResolvedLayout> {
  const resolved = new Map<string, ResolvedLayout>();
  const st: SolverState = { width, height, primitives, resolved };

  passExplicitCoords(st);
  passExplicitItemLayouts(st);
  passImplicitLayouts(st);
  passHud(st);
  passAnchorEffects(st);
  passDomainComposite(st);
  passOrphanContent(st);

  return resolved;
}

// ============================================================
// Pass G：兜底 — 未被任何父布局消费的孤立内容原语
// ============================================================

function passOrphanContent(st: SolverState): void {
  const yTop = st.height * 0.18;
  const yBot = st.height * 0.92;
  const cy = (yTop + yBot) / 2;
  const orphans: Primitive[] = [];
  for (const p of st.primitives) {
    if (st.resolved.has(p.id)) continue;
    if (CONTENT_NODE_TYPES.has(p.type)) orphans.push(p);
  }
  if (orphans.length === 0) return;
  // 多个孤立原语水平错位，避免堆叠到同一点。
  const padX = st.width * 0.15;
  const usableW = st.width - padX * 2;
  const step = orphans.length > 1 ? usableW / (orphans.length - 1) : 0;
  orphans.forEach((p, i) => {
    const x = orphans.length === 1 ? st.width / 2 : padX + step * i;
    // ring / bar 给较大的半径占位，其它给默认 22
    const radius = p.type === "ring" ? Math.min(st.width, yBot - yTop) * 0.18 : 22;
    st.resolved.set(p.id, { x, y: cy, radius });
  });
}

// ============================================================
// Pass E：effect / 标记按 anchor_id 跟随宿主节点
// ============================================================

/** 通过 anchor_id 依附到宿主节点的特效 / 标注类原语。 */
const ANCHORED_TYPES = new Set<string>([
  "burst", "pulse", "glow", "trail", "particle_stream",
  "external_event_marker", "code_marker",
]);

function passAnchorEffects(st: SolverState): void {
  for (const p of st.primitives) {
    if (!ANCHORED_TYPES.has(p.type)) continue;
    if (st.resolved.has(p.id)) continue;
    const anchorId = typeof p.params["anchor_id"] === "string" ? (p.params["anchor_id"] as string) : "";
    if (!anchorId) continue;
    const host = st.resolved.get(anchorId);
    if (!host) continue;
    const entry: ResolvedLayout = { x: host.x, y: host.y };
    if (host.radius !== undefined) entry.radius = host.radius;
    st.resolved.set(p.id, entry);
  }
}

// ============================================================
// Pass F：未定位的域复合原语兜底落到工作区
// ============================================================

/** 以左上角 (x, y) 为锚的矩形域复合原语。 */
const COMPOSITE_RECT_TYPES = new Set<string>([
  "dual_track", "vote_matrix", "sankey_flow", "heat_map",
  "bridge_track", "partition_zone", "mempool_slot",
]);

/** 以中心 (cx, cy) 为锚的圆形域复合原语。 */
const COMPOSITE_CENTER_TYPES = new Set<string>([
  "time_wheel", "pie_chart",
]);

function passDomainComposite(st: SolverState): void {
  const yTop = st.height * 0.18;
  const yBot = st.height * 0.92;
  const padX = st.width * 0.1;
  const usableW = st.width - padX * 2;
  const usableH = yBot - yTop;
  const cx = st.width / 2;
  const cy = (yTop + yBot) / 2;

  for (const p of st.primitives) {
    if (st.resolved.has(p.id)) continue;
    if (COMPOSITE_RECT_TYPES.has(p.type)) {
      const w = usableW * 0.92;
      const h = usableH * 0.7;
      st.resolved.set(p.id, { x: cx - w / 2, y: cy - h / 2, width: w, height: h });
      continue;
    }
    if (COMPOSITE_CENTER_TYPES.has(p.type)) {
      const r = Math.min(usableW, usableH) * 0.28;
      // p_num("cx") / p_num("cy") 会命中 POSITION_X_KEYS / POSITION_Y_KEYS → 读 resolved.x / y
      st.resolved.set(p.id, { x: cx, y: cy, radius: r });
      continue;
    }
  }
}

// ============================================================
// Pass A：*At 显式坐标 — 0~1 逻辑值 × 画布尺寸
// ============================================================

/** 把 0~1 逻辑值放大到对应轴的像素；> 1 视为像素直传。 */
function toPxX(v: number, w: number): number {
  return v >= 0 && v <= 1 ? v * w : v;
}
function toPxY(v: number, h: number): number {
  return v >= 0 && v <= 1 ? v * h : v;
}
function toPxR(v: number, w: number, h: number): number {
  return v >= 0 && v <= 1 ? v * Math.min(w, h) : v;
}

function passExplicitCoords(st: SolverState): void {
  for (const p of st.primitives) {
    const xRaw = p.params["x"];
    const yRaw = p.params["y"];
    const cxRaw = p.params["center_x"];
    const cyRaw = p.params["center_y"];

    let cx: number | undefined;
    let cy: number | undefined;
    if (typeof xRaw === "number" && typeof yRaw === "number") {
      cx = toPxX(xRaw, st.width);
      cy = toPxY(yRaw, st.height);
    } else if (typeof cxRaw === "number" && typeof cyRaw === "number") {
      cx = toPxX(cxRaw, st.width);
      cy = toPxY(cyRaw, st.height);
    }
    if (cx === undefined || cy === undefined) continue;

    const entry: ResolvedLayout = { x: cx, y: cy };

    const sizeMul = p.params["size"];
    if (typeof sizeMul === "number") {
      // PrimNodeAt(size) — 节点半径缩放系数，基准 22px。
      entry.radius = 22 * sizeMul;
    }
    const rRaw = p.params["radius"];
    if (typeof rRaw === "number") entry.radius = toPxR(rRaw, st.width, st.height);
    const wRaw = p.params["width"];
    if (typeof wRaw === "number") entry.width = toPxX(wRaw, st.width);
    const hRaw = p.params["height"];
    if (typeof hRaw === "number") entry.height = toPxY(hRaw, st.height);

    st.resolved.set(p.id, entry);
  }
}

// ============================================================
// Pass B：带 items 列表的布局
// ============================================================

function getIdArr(p: Primitive, key: string): string[] {
  const v = p.params[key];
  if (!Array.isArray(v)) return [];
  return v.filter((s): s is string => typeof s === "string");
}

function passExplicitItemLayouts(st: SolverState): void {
  // 收集所有未解算的布局原语
  const stacks = st.primitives.filter((p) => p.type === "stack" && !st.resolved.has(p.id));
  const graphs = st.primitives.filter((p) => p.type === "graph_layout" && !st.resolved.has(p.id));
  const trees = st.primitives.filter((p) => p.type === "tree_layout" && !st.resolved.has(p.id));

  const horizontalStacks = stacks.filter((p) => p.params["direction"] === "horizontal");
  const verticalStacks = stacks.filter((p) => p.params["direction"] !== "horizontal");

  // 顶部 / 底部留白比例：HUD 区 18% + 控制区 8%；工作区 = 74%
  const yTop = st.height * 0.18;
  const yBot = st.height * 0.92;
  const workH = yBot - yTop;

  // 横向栈：每条占工作区 1/N 高度的带
  const hStackBands: Array<{ cy: number; height: number }> = [];
  const hStackCount = horizontalStacks.length;
  // 工作区拆分：横向栈占顶部 N 条带，其余空间留给环 / 矩阵 / 树
  const hStackAreaH = hStackCount > 0 ? Math.min(workH * 0.4, hStackCount * 100) : 0;
  for (let i = 0; i < hStackCount; i++) {
    const bandH = hStackAreaH / hStackCount;
    const cy = yTop + bandH / 2 + i * bandH;
    hStackBands.push({ cy, height: bandH });
  }

  // 横向栈：items 沿 x 等距分布
  horizontalStacks.forEach((p, i) => {
    const ids = getIdArr(p, "items");
    if (ids.length === 0) return;
    const band = hStackBands[i];
    if (!band) return;
    const padX = st.width * 0.08;
    const usableW = st.width - padX * 2;
    const step = ids.length > 1 ? usableW / (ids.length - 1) : 0;
    // 节点半径：随密度收缩，避免相邻重叠（最少 12 最多 28）
    const nodeR = Math.max(12, Math.min(28, (usableW / Math.max(1, ids.length)) * 0.32));
    ids.forEach((id, j) => {
      const x = ids.length === 1 ? padX + usableW / 2 : padX + step * j;
      if (!st.resolved.has(id)) {
        st.resolved.set(id, { x, y: band.cy, radius: nodeR });
      }
    });
    st.resolved.set(p.id, { x: padX, y: band.cy, width: usableW, height: band.height });
  });

  // 纵向栈：左侧列
  verticalStacks.forEach((p, i) => {
    const ids = getIdArr(p, "items");
    if (ids.length === 0) return;
    const x = st.width * (0.12 + i * 0.16);
    const padY = yTop + 12;
    const usableH = (yBot - 12) - padY;
    const step = ids.length > 1 ? usableH / (ids.length - 1) : 0;
    ids.forEach((id, j) => {
      const y = ids.length === 1 ? padY + usableH / 2 : padY + step * j;
      if (!st.resolved.has(id)) {
        st.resolved.set(id, { x, y, radius: 18 });
      }
    });
    st.resolved.set(p.id, { x, y: padY, width: 16, height: usableH });
  });

  // 工作区剩余空间（环 / 矩阵 / 树共享）
  const ringYStart = yTop + hStackAreaH + 8;
  const ringYEnd = yBot;
  const ringCY = (ringYStart + ringYEnd) / 2;
  const ringMaxR = Math.min(st.width, ringYEnd - ringYStart) * 0.42;

  // 图布局（显式 nodes）：按 algorithm 分发
  graphs.forEach((p, i) => {
    const algo = typeof p.params["algorithm"] === "string" ? (p.params["algorithm"] as string) : "circular";
    const ids = getIdArr(p, "nodes");
    if (ids.length === 0) return;
    const cx = graphs.length === 1 ? st.width / 2 : (st.width * (i + 1)) / (graphs.length + 1);
    if (algo === "grid") {
      const cols = Math.ceil(Math.sqrt(ids.length));
      const rows = Math.ceil(ids.length / cols);
      const cellW = (ringMaxR * 2) / cols;
      const cellH = (ringMaxR * 2) / rows;
      ids.forEach((id, j) => {
        const r = Math.floor(j / cols);
        const c = j % cols;
        const x = cx - ringMaxR + cellW * c + cellW / 2;
        const y = ringCY - ringMaxR + cellH * r + cellH / 2;
        if (!st.resolved.has(id)) {
          st.resolved.set(id, { x, y, radius: Math.min(cellW, cellH) * 0.32 });
        }
      });
    } else {
      // circular / force fallback
      placeOnCircle(st, ids, cx, ringCY, ringMaxR);
    }
    st.resolved.set(p.id, { x: cx, y: ringCY, radius: ringMaxR });
  });

  // 树布局：BFS 分层
  trees.forEach((p) => {
    const rootId = typeof p.params["root_id"] === "string" ? (p.params["root_id"] as string) : "";
    if (!rootId) return;
    const childMap = new Map<string, string[]>();
    for (const cp of st.primitives) {
      if (cp.type !== "edge") continue;
      const f = cp.params["from_id"];
      const t = cp.params["to_id"];
      if (typeof f === "string" && typeof t === "string") {
        const list = childMap.get(f) ?? [];
        list.push(t);
        childMap.set(f, list);
      }
    }
    const layers: string[][] = [[rootId]];
    const seen = new Set<string>([rootId]);
    while (true) {
      const last = layers[layers.length - 1];
      if (!last) break;
      const next: string[] = [];
      for (const id of last) {
        for (const c of childMap.get(id) ?? []) {
          if (!seen.has(c)) {
            seen.add(c);
            next.push(c);
          }
        }
      }
      if (next.length === 0) break;
      layers.push(next);
    }
    const layerH = (ringYEnd - ringYStart) / layers.length;
    layers.forEach((layer, li) => {
      const ly = ringYStart + layerH * li + layerH / 2;
      const layerW = st.width * 0.84;
      const padX = st.width * 0.08;
      const step = layer.length > 1 ? layerW / (layer.length - 1) : 0;
      const nodeR = Math.max(14, 22 - layer.length);
      layer.forEach((id, j) => {
        const x = layer.length === 1 ? st.width / 2 : padX + step * j;
        if (!st.resolved.has(id)) {
          st.resolved.set(id, { x, y: ly, radius: nodeR });
        }
      });
    });
    st.resolved.set(p.id, { x: st.width / 2, y: (ringYStart + ringYEnd) / 2, width: st.width, height: ringYEnd - ringYStart });
  });
}

/** 将 ids 等角分布在 (cx, cy, r) 圆周上。 */
function placeOnCircle(st: SolverState, ids: string[], cx: number, cy: number, r: number): void {
  if (ids.length === 0) return;
  const nodeR = Math.max(12, Math.min(26, (2 * Math.PI * r) / ids.length / 3));
  ids.forEach((id, j) => {
    const angle = (j / ids.length) * Math.PI * 2 - Math.PI / 2;
    const x = cx + r * Math.cos(angle);
    const y = cy + r * Math.sin(angle);
    if (!st.resolved.has(id)) {
      st.resolved.set(id, { x, y, radius: nodeR });
    }
  });
}

// ============================================================
// Pass C：隐式占位布局
// ============================================================

function passImplicitLayouts(st: SolverState): void {
  const rings = st.primitives.filter((p) => p.type === "ring_layout" && !st.resolved.has(p.id));
  const matrices = st.primitives.filter((p) => p.type === "matrix_layout" && !st.resolved.has(p.id));
  const lanes = st.primitives.filter((p) => p.type === "horizontal_lane" && !st.resolved.has(p.id));

  // 计算工作区（与 Pass B 保持一致）
  const hStackCount = st.primitives.filter(
    (p) => p.type === "stack" && p.params["direction"] === "horizontal" && st.resolved.has(p.id),
  ).length;
  const yTop = st.height * 0.18;
  const yBot = st.height * 0.92;
  const workH = yBot - yTop;
  const hStackAreaH = hStackCount > 0 ? Math.min(workH * 0.4, hStackCount * 100) : 0;
  const ringYStart = yTop + hStackAreaH + 8;
  const ringYEnd = yBot;
  const ringCY = (ringYStart + ringYEnd) / 2;
  const ringMaxR = Math.min(st.width, ringYEnd - ringYStart) * 0.38;

  // 隐式 ring_layout：在未解算的 content node 中取前 slots 个
  rings.forEach((p, i) => {
    const slots = typeof p.params["slots"] === "number" ? (p.params["slots"] as number) : 0;
    if (slots <= 0) return;
    const collected: string[] = [];
    for (const cp of st.primitives) {
      if (!CONTENT_NODE_TYPES.has(cp.type)) continue;
      if (st.resolved.has(cp.id)) continue;
      collected.push(cp.id);
      if (collected.length >= slots) break;
    }
    if (collected.length === 0) return;
    const cx = rings.length === 1 ? st.width / 2 : (st.width * (i + 1)) / (rings.length + 1);
    placeOnCircle(st, collected, cx, ringCY, ringMaxR);
    st.resolved.set(p.id, { x: cx, y: ringCY, radius: ringMaxR });
  });

  // 矩阵布局：grid_cell 按 row/col 派发
  matrices.forEach((p) => {
    const rows = typeof p.params["rows"] === "number" ? (p.params["rows"] as number) : 0;
    const cols = typeof p.params["cols"] === "number" ? (p.params["cols"] as number) : 0;
    if (rows <= 0 || cols <= 0) return;
    const padX = st.width * 0.1;
    const usableW = st.width - padX * 2;
    const usableH = ringYEnd - ringYStart;
    const cellW = usableW / cols;
    const cellH = usableH / rows;
    for (const cp of st.primitives) {
      if (cp.type !== "grid_cell") continue;
      if (st.resolved.has(cp.id)) continue;
      const r = typeof cp.params["row"] === "number" ? (cp.params["row"] as number) : 0;
      const c = typeof cp.params["col"] === "number" ? (cp.params["col"] as number) : 0;
      st.resolved.set(cp.id, {
        x: padX + c * cellW + cellW / 2,
        y: ringYStart + r * cellH + cellH / 2,
        width: cellW - 4,
        height: cellH - 4,
      });
    }
    st.resolved.set(p.id, { x: padX, y: ringYStart, width: usableW, height: usableH });
  });

  // 水平泳道：把工作区垂直 N 等分
  if (lanes.length > 0) {
    const laneH = (ringYEnd - ringYStart) / lanes.length;
    lanes.forEach((p, i) => {
      const cy = ringYStart + laneH * i + laneH / 2;
      st.resolved.set(p.id, { x: 0, y: cy, width: st.width, height: laneH });
    });
  }
}

// ============================================================
// Pass D：HUD 类原语自动落位（顶部带状）
// ============================================================

function passHud(st: SolverState): void {
  const huds = st.primitives.filter((p) => HUD_TYPES.has(p.type) && !st.resolved.has(p.id));
  if (huds.length === 0) return;
  const padX = st.width * 0.08;
  const usableW = st.width - padX * 2;
  // HUD 占顶部 18% 区域，按个数堆叠（高度 24px + 8px 间距）
  huds.forEach((p, i) => {
    const y = 18 + i * 32;
    st.resolved.set(p.id, { x: padX, y, width: usableW, height: 24 });
  });
}
