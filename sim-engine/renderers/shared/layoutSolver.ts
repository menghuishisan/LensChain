/**
 * layoutSolver.ts — 把 47 原语解算成画布像素坐标。
 *
 * 设计依据：
 *   docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.2 / §3.3
 *   docs/modules/04-实验环境/06.2-SimEngine前端设计.md §四
 *   docs/modules/04-实验环境/sim-engine-redesign-proposal.html 核心规则四铁律
 *
 * 算法分 6 Pass：
 *   Pass 1 顶部 HUD：仅 phase_progress 占用顶部 36px 高度。
 *   Pass 2 horizontal_lane：声明语义泳道；无 y 字段时按数量等分主区，
 *          有 y 字段时按逻辑 0~1 → 主区像素。
 *   Pass 3 父布局：ring_layout / matrix_layout / graph_layout / tree_layout / stack
 *          解算容器本身的中心 + 子原语槽位。
 *   Pass 4 子原语就位：node / bar / grid_cell 等按父容器槽位填入位置。
 *   Pass 5 锚定效果：particle_stream / burst / pulse / trail / glow / shake / shift_animation /
 *          external_event_marker / label(anchor_id) / tooltip(anchor_id) → 复制 anchor 位置。
 *   Pass 6 校验：canvas 类原语若仍无坐标，抛错暴露协议违规。
 *
 * 设计原则：
 *   • 不写兑底：未注册类型 / 缺字段 / 引用不存在的 id → 直接抛错。
 *   • 内容驱动尺寸：node radius clamp [16, 28]；horizontal step ≥ 2r+8。
 *   • 文字宽度感知：基于平均字符宽度估算（无 canvas ctx 时用 6.6px/char）。
 *
 * 注：14 个走 DOM / 浮层的原语在本 solver 不分配画布坐标
 *     （code_block / math_formula / register_row / error_overlay / label(无 anchor) /
 *      tooltip / annotation / math_pipeline / 6 个 layout 容器内部不绘制）。
 */

import type {
  JsonObject, JsonValue,
  Primitive, PrimitiveType,
} from "./types.js";
import { asArray, asNumber, asObject, asString, clamp } from "./utils.js";

// ============================================================
// 公开类型
// ============================================================

/** 解算结果中单个原语的像素位置 + 尺寸。 */
export interface ResolvedPosition {
  /** 主锚点 X（像素）。 */
  x: number;
  /** 主锚点 Y（像素）。 */
  y: number;
  /** 矩形宽（像素，可选）。 */
  width?: number;
  /** 矩形高（像素，可选）。 */
  height?: number;
  /** 圆形半径（像素，可选）。 */
  radius?: number;
  /** 文字标签像素宽度（measureText 缓存，可选）。 */
  labelWidth?: number;
}

/** 主区分配给单个 horizontal_lane 的横向带。 */
export interface ResolvedLane {
  id: string;
  /** 带顶 y（像素）。 */
  y: number;
  /** 带高（像素）。 */
  height: number;
  /** 带中线（像素，供单行容器使用）。 */
  midY: number;
  /** 可选文字标签。 */
  label?: string;
}

/** 完整解算结果。 */
export interface ResolvedLayout {
  /** primitive.id → 像素位置。 */
  positions: Map<string, ResolvedPosition>;
  /** HUD 矩形（phase_progress 专用），null 表示本场景无 phase_progress。 */
  hud: { x: number; y: number; width: number; height: number } | null;
  /** 主区矩形（HUD 之下的剩余区域）。 */
  main: { x: number; y: number; width: number; height: number };
  /** 主区内的 horizontal_lane 带。 */
  lanes: ResolvedLane[];
  /** 画布总尺寸。 */
  bounds: { width: number; height: number };
  /** 是否触发紧凑模式（width<600 或 height<360）。 */
  compact: boolean;
}

// ============================================================
// 常量（与 redesign-proposal §四铁律一致）
// ============================================================

/** 顶部 HUD 高度（仅 phase_progress 占用）。 */
const HUD_HEIGHT = 36;
/** 紧凑模式 HUD 高度。 */
const HUD_HEIGHT_COMPACT = 24;
/** 主区上下内边距。 */
const MAIN_PADDING_Y = 16;
/** 主区左右内边距。 */
const MAIN_PADDING_X = 24;
/** node 半径常规 clamp。 */
const NODE_RADIUS_MIN = 16;
const NODE_RADIUS_MAX = 28;
/** node 半径紧凑模式 clamp。 */
const NODE_RADIUS_MIN_COMPACT = 12;
const NODE_RADIUS_MAX_COMPACT = 18;
/** 水平步距下限 = 2r + 此值。 */
const STEP_PADDING = 8;
/** 紧凑判定阈值。 */
const COMPACT_WIDTH = 600;
const COMPACT_HEIGHT = 360;
/** 平均字符像素宽（无 canvas ctx 时的估算值，对应 11px sans-serif）。 */
const AVG_CHAR_WIDTH = 6.6;

/**
 * 不在 canvas 绘制的原语 type（layoutSolver / primitiveRenderer 共同跳过）。
 *
 * 协议依据：
 *   - 06.md §3.2.4 内容类：code_block / math_formula / register_row 由前端 KaTeX /
 *     语法高亮渲染，DOM 浮层；
 *   - 06.md §3.2.5 状态指示类：error_overlay 居中 Modal、progress_bar / risk_gauge
 *     下沉到 SimSceneSlot 的 ▼ 内容类原语 dropdown DOM 渲染；
 *   - 06.md §3.4 浮层：tooltip / annotation 由 React DOM 浮层处理；
 *   - math_pipeline 走浮层。
 *
 * 注：6 个布局容器（horizontal_lane / stack / ring_layout / tree_layout / graph_layout /
 * matrix_layout）虽然自身不画内容，但需要解算 ResolvedPosition 让子原语用容器槽位
 * 定位 → 不放入此集合，由 Pass 3 正常解算。
 */
const NON_CANVAS_TYPES = new Set<PrimitiveType>([
  // DOM 渲染（KaTeX / 语法高亮 / 进度条 / 仪表）—— SimSceneSlot ▼ 下沉
  "code_block", "math_formula", "register_row", "progress_bar", "risk_gauge",
  // DOM 浮层
  "tooltip", "annotation", "math_pipeline", "error_overlay",
]);

// ============================================================
// 主入口
// ============================================================

/**
 * resolveLayout 把 primitives 列表解算成像素坐标。
 * 同一组 primitives + 同一画布尺寸 + 同一 ctx 必须输出确定性结果（pure function）。
 */
export function resolveLayout(
  primitives: readonly Primitive[],
  width: number,
  height: number,
  measureText?: (text: string) => number,
): ResolvedLayout {
  if (!Number.isFinite(width) || width <= 0 || !Number.isFinite(height) || height <= 0) {
    throw new Error(`resolveLayout: 非法画布尺寸 ${width}x${height}`);
  }
  const compact = width < COMPACT_WIDTH || height < COMPACT_HEIGHT;
  const measure = measureText ?? defaultMeasureText;

  // ===== Pass 1: HUD =====
  const hasPhaseProgress = primitives.some(p => p.type === "phase_progress");
  const hudH = compact ? HUD_HEIGHT_COMPACT : HUD_HEIGHT;
  const hud = hasPhaseProgress
    ? { x: 0, y: 0, width, height: hudH }
    : null;
  const mainY = hudH * (hasPhaseProgress ? 1 : 0);
  const main = {
    x: MAIN_PADDING_X,
    y: mainY + MAIN_PADDING_Y,
    width: width - MAIN_PADDING_X * 2,
    height: height - mainY - MAIN_PADDING_Y * 2,
  };
  if (main.width <= 0 || main.height <= 0) {
    // 画布合法（W>0 H>0）但减完 HUD/padding 后无主区可分配。这不是协议违规，
    // 而是运行时尺寸瞬态：典型场景是 ResizeObserver 在父容器挂载/收起时多次回报
    // 中间值（如 height=17px），稳态后下一帧自然回到正常尺寸。返回空布局即可，
    // renderer 这一帧自然不画任何 primitive。后续帧 W/H 一回到合理值就正常解算。
    //
    // 注意：保留上方 width/height ≤ 0 的 throw —— 那是调用方传非法值，属协议违规。
    return {
      positions: new Map(),
      hud,
      main: { x: main.x, y: main.y, width: 0, height: 0 },
      lanes: [],
      bounds: { width, height },
      compact,
    };
  }

  // 索引：id → primitive
  const byId = new Map<string, Primitive>();
  for (const p of primitives) {
    if (byId.has(p.id)) {
      throw new Error(`resolveLayout: primitive id 重复 "${p.id}"`);
    }
    byId.set(p.id, p);
  }

  const positions = new Map<string, ResolvedPosition>();

  // 分配 phase_progress 位置（占满 HUD）
  if (hud) {
    for (const p of primitives) {
      if (p.type === "phase_progress") {
        positions.set(p.id, { x: hud.x, y: hud.y, width: hud.width, height: hud.height });
      }
    }
  }

  // ===== Pass 2: horizontal_lane → 主区垂直带 =====
  const laneInputs = primitives.filter(p => p.type === "horizontal_lane");
  const lanes = allocateLanes(laneInputs, main);
  for (const lane of lanes) {
    positions.set(lane.id, {
      x: main.x, y: lane.y, width: main.width, height: lane.height,
    });
  }

  // ===== Pass 3: 父布局容器 =====
  const containerSlots = new Map<string, Map<string | number, ResolvedPosition>>();

  // 默认主区中心作为单容器场景的回退中心（仅当场景仅一个容器时启用）
  for (const p of primitives) {
    switch (p.type) {
      case "ring_layout": {
        // 06.md §3.2.3：ring_layout.nodes[] 显式声明环上成员 ID 列表，
        // 渲染器按列表顺序从 12 点钟方向顺时针均分 slot。语义对齐 graph_layout.nodes[]。
        const nodes = asArray<string>(p.params.nodes);
        if (nodes.length === 0) {
          throw new Error(`ring_layout "${p.id}": nodes 为空`);
        }
        const ring = resolveRingLayout(p, main, compact, nodes.length);
        const center = readCenter(p, main);
        positions.set(p.id, { x: center.x, y: center.y, radius: ring.radius });
        const slotMap = new Map<string, ResolvedPosition>();
        nodes.forEach((nodeId, i) => slotMap.set(nodeId, ring.points[i]!));
        containerSlots.set(p.id, slotMap);
        break;
      }
      case "matrix_layout": {
        const grid = resolveMatrixLayout(p, main);
        positions.set(p.id, { x: grid.originX, y: grid.originY, width: grid.cols * grid.cellW, height: grid.rows * grid.cellH });
        const cellMap = new Map<string, ResolvedPosition>();
        for (let r = 0; r < grid.rows; r++) {
          for (let c = 0; c < grid.cols; c++) {
            cellMap.set(`${r}:${c}`, {
              x: grid.originX + c * grid.cellW + grid.cellW / 2,
              y: grid.originY + r * grid.cellH + grid.cellH / 2,
              width: grid.cellW, height: grid.cellH,
            });
          }
        }
        containerSlots.set(p.id, cellMap);
        break;
      }
      case "stack": {
        const items = asArray<string>(p.params.items);
        const direction = asString(p.params.direction, "horizontal");
        const r = nodeRadius(compact);
        const stackBox = readStackBox(p, main);
        const slotMap = new Map<string, ResolvedPosition>();
        if (direction === "horizontal") {
          const step = Math.max(2 * r + STEP_PADDING, stackBox.width / Math.max(1, items.length));
          const startX = stackBox.x + step / 2;
          const y = stackBox.y + stackBox.height / 2;
          items.forEach((id, i) => slotMap.set(id, { x: startX + i * step, y, radius: r }));
        } else {
          const step = Math.max(2 * r + STEP_PADDING, stackBox.height / Math.max(1, items.length));
          const startY = stackBox.y + step / 2;
          const x = stackBox.x + stackBox.width / 2;
          items.forEach((id, i) => slotMap.set(id, { x, y: startY + i * step, radius: r }));
        }
        positions.set(p.id, { x: stackBox.x, y: stackBox.y, width: stackBox.width, height: stackBox.height });
        containerSlots.set(p.id, slotMap);
        break;
      }
      case "graph_layout": {
        const algo = asString(p.params.algorithm, "circular");
        const nodes = asArray<string>(p.params.nodes);
        const center = { x: main.x + main.width / 2, y: main.y + main.height / 2 };
        const radius = Math.min(main.width, main.height) * 0.36;
        const slotMap = new Map<string, ResolvedPosition>();
        const r = nodeRadius(compact);
        if (algo === "circular" || algo === "force") {
          // force 退化为 circular（真实力导向交给场景预算好，这里只提供稳定坐标）
          const n = Math.max(1, nodes.length);
          nodes.forEach((id, i) => {
            const angle = (Math.PI * 2 * i) / n - Math.PI / 2;
            slotMap.set(id, {
              x: center.x + radius * Math.cos(angle),
              y: center.y + radius * Math.sin(angle),
              radius: r,
            });
          });
        } else if (algo === "grid") {
          const cols = Math.ceil(Math.sqrt(nodes.length));
          const rows = Math.ceil(nodes.length / cols);
          const cw = main.width / cols, ch = main.height / rows;
          nodes.forEach((id, i) => {
            const c = i % cols, ro = Math.floor(i / cols);
            slotMap.set(id, {
              x: main.x + c * cw + cw / 2,
              y: main.y + ro * ch + ch / 2,
              radius: r,
            });
          });
        } else {
          throw new Error(`graph_layout "${p.id}": 未知 algorithm "${algo}"，必须是 circular|force|grid`);
        }
        positions.set(p.id, { x: center.x, y: center.y, radius });
        containerSlots.set(p.id, slotMap);
        break;
      }
      case "tree_layout": {
        // 简化：按 nodes 字段（若有）等分布在主区，按 BFS 层级排版交给后续迭代。
        // v1 输出居中 + radius=main 宽高较小者的 0.4。
        const center = { x: main.x + main.width / 2, y: main.y + main.height / 2 };
        positions.set(p.id, { x: center.x, y: center.y, width: main.width, height: main.height });
        // tree_layout 的节点位置由场景显式给出（PrimNodeAt）；本 solver 不强行分配。
        containerSlots.set(p.id, new Map());
        break;
      }
      default:
        // 其它 type 在后续 Pass 处理
        break;
    }
  }

  // ===== Pass 4: 子原语就位 =====
  // 4.1 显式 x/y 的原语（PrimNodeAt / PrimBarAt 等）
  for (const p of primitives) {
    if (positions.has(p.id)) continue;
    if (NON_CANVAS_TYPES.has(p.type)) continue;
    const x = readOptNumber(p.params, "x");
    const y = readOptNumber(p.params, "y");
    if (x !== null && y !== null) {
      const pos: ResolvedPosition = {
        x: logicalToPixelX(x, main),
        y: logicalToPixelY(y, main),
      };
      if (p.type === "node") pos.radius = readNodeRadius(p, compact);
      const w = readOptNumber(p.params, "width");
      const h = readOptNumber(p.params, "height");
      if (w !== null) pos.width = logicalSpan(w, main.width);
      if (h !== null) pos.height = logicalSpan(h, main.height);
      positions.set(p.id, pos);
    }
  }

  // 4.2 容器内的子原语（通过 items / nodes / row+col 引用）
  for (const p of primitives) {
    if (positions.has(p.id)) continue;
    if (NON_CANVAS_TYPES.has(p.type)) continue;

    // node 通过被 stack.items[] / ring_layout.nodes[] / graph_layout.nodes[] 显式声明找到容器
    if (p.type === "node") {
      const owner = findOwnerForNode(p.id, primitives);
      if (owner) {
        const slot = lookupNodeSlot(p, owner, containerSlots);
        if (slot) { positions.set(p.id, slot); continue; }
      }
    }
    // grid_cell / mempool_slot 通过 row/col 嵌入父 matrix_layout
    if (p.type === "grid_cell" || p.type === "mempool_slot") {
      const row = asNumber(p.params.row, -1);
      const col = asNumber(p.params.col, -1);
      if (row < 0 || col < 0) {
        throw new Error(`${p.type} "${p.id}": 缺少 row/col 字段`);
      }
      const matrix = findFirstByType(primitives, "matrix_layout");
      if (!matrix) {
        throw new Error(`${p.type} "${p.id}": 找不到父 matrix_layout`);
      }
      const slot = containerSlots.get(matrix.id)?.get(`${row}:${col}`);
      if (!slot) {
        throw new Error(`${p.type} "${p.id}": matrix_layout "${matrix.id}" 无 (${row},${col}) 槽`);
      }
      positions.set(p.id, slot);
      continue;
    }
  }

  // 4.3 target_zone：贯穿主区的水平/垂直虚线
  for (const p of primitives) {
    if (p.type !== "target_zone" || positions.has(p.id)) continue;
    const value = asNumber(p.params.value, NaN);
    if (!Number.isFinite(value)) {
      throw new Error(`target_zone "${p.id}": 缺 value`);
    }
    const axis = asString(p.params.axis, "y");
    // value 是逻辑 0~1（已被场景归一化），按轴投影到主区
    if (axis === "y") {
      positions.set(p.id, {
        x: main.x, y: main.y + (1 - clamp(value, 0, 1)) * main.height,
        width: main.width, height: 0,
      });
    } else if (axis === "x") {
      positions.set(p.id, {
        x: main.x + clamp(value, 0, 1) * main.width, y: main.y,
        width: 0, height: main.height,
      });
    } else {
      throw new Error(`target_zone "${p.id}": axis 必须是 "x" 或 "y"，得到 "${axis}"`);
    }
  }

  // 4.4 ring / pie_chart / time_wheel / sankey_flow / dual_track / vote_matrix / heat_map /
  //     bridge_track / partition_zone / curve_point / verify_path_highlight 等"占据主区或
  //     剩余空间"的复合块：依次纵向堆叠，落在 lane 或主区中。
  //     progress_bar / risk_gauge 已下沉到 SimSceneSlot 的 DOM dropdown，不再参与 canvas
  //     主区分配（详 06.md §3.2.5 + NON_CANVAS_TYPES 注释）。
  const stackableTypes: PrimitiveType[] = [
    "ring", "pie_chart", "time_wheel",
    "sankey_flow", "dual_track", "vote_matrix", "heat_map", "bridge_track",
    "partition_zone", "verify_path_highlight", "curve_point",
    "link_indicator", "external_event_marker",
  ];
  layoutOrphanStackables(primitives, stackableTypes, lanes, main, positions, compact);

  // ===== Pass 4.5: edge 端点 → 中点位置 =====
  // 协议依据：06.md §3.2.1 line 169 —— edge schema 为 (id, from_id, to_id, style?, animation?)，
  // 不含 x/y，位置由 from_id / to_id 派生。drawEdge 自身用端点画线（不依赖 edge.id 的 position），
  // 但 Pass 7 sanity 要求所有 canvas-绘制 primitive 在 positions 中有项；同时 anchor_id 也可能锚到
  // 一条 edge（如 glow 高亮某段链路）。统一以 from/to 中点作为 edge 自身的 ResolvedPosition。
  for (const p of primitives) {
    if (p.type !== "edge" || positions.has(p.id)) continue;
    const fromID = asString(p.params.from_id, "");
    const toID = asString(p.params.to_id, "");
    if (!fromID || !toID) {
      throw new Error(`edge "${p.id}": 缺 from_id 或 to_id`);
    }
    const from = positions.get(fromID);
    const to = positions.get(toID);
    if (!from) {
      throw new Error(`edge "${p.id}": from_id="${fromID}" 未解算（端点不存在或未参与父布局）`);
    }
    if (!to) {
      throw new Error(`edge "${p.id}": to_id="${toID}" 未解算（端点不存在或未参与父布局）`);
    }
    positions.set(p.id, {
      x: (from.x + to.x) / 2,
      y: (from.y + to.y) / 2,
    });
  }

  // ===== Pass 5: 锚定效果 =====
  for (const p of primitives) {
    if (positions.has(p.id)) continue;
    const anchorID = asString(p.params.anchor_id, "");
    if (!anchorID) continue;
    const anchor = positions.get(anchorID);
    if (!anchor) {
      throw new Error(`原语 "${p.id}" (${p.type}) anchor_id="${anchorID}" 引用不存在的 primitive`);
    }
    positions.set(p.id, { ...anchor });
  }

  // shift_animation 用 target_id 而非 anchor_id
  for (const p of primitives) {
    if (positions.has(p.id) || p.type !== "shift_animation") continue;
    const targetID = asString(p.params.target_id, "");
    const anchor = positions.get(targetID);
    if (!anchor) {
      throw new Error(`shift_animation "${p.id}": target_id="${targetID}" 引用不存在的 primitive`);
    }
    positions.set(p.id, { ...anchor });
  }

  // verify_path_highlight 用 node_ids[0] 作为锚（多点高亮由 drawer 自己遍历）
  for (const p of primitives) {
    if (positions.has(p.id) || p.type !== "verify_path_highlight") continue;
    const nodeIds = asArray<string>(p.params.node_ids);
    if (nodeIds.length === 0) {
      throw new Error(`verify_path_highlight "${p.id}": node_ids 为空`);
    }
    const firstId = nodeIds[0]!;
    const first = positions.get(firstId);
    if (!first) {
      throw new Error(`verify_path_highlight "${p.id}": 首个 node_id "${firstId}" 未解析`);
    }
    positions.set(p.id, { ...first });
  }

  // label：anchor_id 模式已在 Pass 5 处理；剩余仅 DOM 浮层用，不入 canvas → 不再分配
  // tooltip / annotation / math_pipeline / code_block / math_formula / register_row / error_overlay 同理

  // ===== Pass 6: 文字宽度量算（仅对有 label 字段的原语） =====
  for (const p of primitives) {
    const pos = positions.get(p.id);
    if (!pos) continue;
    const text = readLabelText(p);
    if (text) pos.labelWidth = measure(text);
  }

  // ===== Pass 7: 校验 =====
  for (const p of primitives) {
    if (positions.has(p.id)) continue;
    if (NON_CANVAS_TYPES.has(p.type)) continue;
    // 6 个布局容器自身：tree_layout / horizontal_lane 已分配；其它都该有位置
    if (p.type === "horizontal_lane") continue;
    throw new Error(`原语 "${p.id}" (${p.type}) 无法解算位置——协议未声明所属容器、未给 anchor_id、未给 x/y`);
  }

  return { positions, hud, main, lanes, bounds: { width, height }, compact };
}

// ============================================================
// 辅助函数
// ============================================================

function allocateLanes(
  laneInputs: readonly Primitive[],
  main: ResolvedLayout["main"],
): ResolvedLane[] {
  if (laneInputs.length === 0) return [];
  const explicit = laneInputs.filter(p => readOptNumber(p.params, "y") !== null);
  const auto = laneInputs.filter(p => readOptNumber(p.params, "y") === null);
  const lanes: ResolvedLane[] = [];

  const buildLane = (p: Primitive, y: number, h: number): ResolvedLane => {
    const label = asString(p.params.label, "");
    const lane: ResolvedLane = { id: p.id, y, height: h, midY: y + h / 2 };
    if (label) lane.label = label;
    return lane;
  };

  // 显式 y 的泳道
  for (const p of explicit) {
    const yLogical = clamp(asNumber(p.params.y, 0), 0, 1);
    const h = main.height / laneInputs.length;
    lanes.push(buildLane(p, main.y + yLogical * main.height - h / 2, h));
  }
  // 自动等分的泳道
  if (auto.length > 0) {
    const remainingH = main.height - lanes.length * (main.height / laneInputs.length);
    const h = remainingH / auto.length;
    let cursor = main.y + lanes.length * (main.height / laneInputs.length);
    for (const p of auto) {
      lanes.push(buildLane(p, cursor, h));
      cursor += h;
    }
  }
  return lanes;
}

function resolveRingLayout(
  p: Primitive,
  main: ResolvedLayout["main"],
  compact: boolean,
  slots: number,
): { points: ResolvedPosition[]; radius: number } {
  if (slots <= 0) {
    throw new Error(`ring_layout "${p.id}": slots 必须 ≥1，得到 ${slots}`);
  }
  const center = readCenter(p, main);
  const explicitR = readOptNumber(p.params, "radius");
  const radius = explicitR !== null
    ? explicitR * Math.min(main.width, main.height)
    : Math.min(main.width, main.height) * (compact ? 0.32 : 0.36);
  const r = nodeRadius(compact);
  const points: ResolvedPosition[] = [];
  for (let i = 0; i < slots; i++) {
    const angle = (Math.PI * 2 * i) / slots - Math.PI / 2;
    points.push({
      x: center.x + radius * Math.cos(angle),
      y: center.y + radius * Math.sin(angle),
      radius: r,
    });
  }
  return { points, radius };
}

function resolveMatrixLayout(
  p: Primitive,
  main: ResolvedLayout["main"],
): { originX: number; originY: number; rows: number; cols: number; cellW: number; cellH: number } {
  const rows = Math.max(1, asNumber(p.params.rows, 0));
  const cols = Math.max(1, asNumber(p.params.cols, 0));
  if (rows <= 0 || cols <= 0) {
    throw new Error(`matrix_layout "${p.id}": rows/cols 必须 ≥1，得到 ${rows}x${cols}`);
  }
  const explicitW = readOptNumber(p.params, "cell_w");
  const explicitH = readOptNumber(p.params, "cell_h");
  const cellW = explicitW !== null ? explicitW * main.width : main.width / cols;
  const cellH = explicitH !== null ? explicitH * main.height : main.height / rows;
  const originX = main.x + (main.width - cellW * cols) / 2;
  const originY = main.y + (main.height - cellH * rows) / 2;
  return { originX, originY, rows, cols, cellW, cellH };
}

function readCenter(p: Primitive, main: ResolvedLayout["main"]): { x: number; y: number } {
  const cx = readOptNumber(p.params, "center_x");
  const cy = readOptNumber(p.params, "center_y");
  return {
    x: cx !== null ? main.x + cx * main.width : main.x + main.width / 2,
    y: cy !== null ? main.y + cy * main.height : main.y + main.height / 2,
  };
}

function readStackBox(
  p: Primitive,
  main: ResolvedLayout["main"],
): { x: number; y: number; width: number; height: number } {
  const x = readOptNumber(p.params, "x");
  if (x !== null) {
    // 显式 x（PrimStackAt），高度仍占主区一带
    return {
      x: main.x + x * main.width,
      y: main.y + main.height * 0.4,
      width: main.width * 0.8,
      height: main.height * 0.2,
    };
  }
  return { x: main.x, y: main.y, width: main.width, height: main.height };
}

function findOwnerForNode(nodeId: string, primitives: readonly Primitive[]): Primitive | null {
  // 06.md §3.2.3：5 个布局容器中 stack / graph_layout / ring_layout 通过显式
  // members 字段（items[] 或 nodes[]）声明子节点；matrix_layout 子原语自带 row+col；
  // tree_layout 子原语沿 root_id+edge 链路推导（本函数不参与）。
  for (const p of primitives) {
    if (p.type === "stack") {
      const items = asArray<string>(p.params.items);
      if (items.includes(nodeId)) return p;
    }
    if (p.type === "graph_layout") {
      const nodes = asArray<string>(p.params.nodes);
      if (nodes.includes(nodeId)) return p;
    }
    if (p.type === "ring_layout") {
      const nodes = asArray<string>(p.params.nodes);
      if (nodes.includes(nodeId)) return p;
    }
  }
  return null;
}

function lookupNodeSlot(
  node: Primitive,
  owner: Primitive,
  containerSlots: Map<string, Map<string | number, ResolvedPosition>>,
): ResolvedPosition | null {
  const slots = containerSlots.get(owner.id);
  if (!slots) return null;
  if (owner.type === "stack" || owner.type === "graph_layout" || owner.type === "ring_layout") {
    return slots.get(node.id) ?? null;
  }
  return null;
}

function layoutOrphanStackables(
  primitives: readonly Primitive[],
  types: readonly PrimitiveType[],
  lanes: ResolvedLane[],
  main: ResolvedLayout["main"],
  positions: Map<string, ResolvedPosition>,
  compact: boolean,
): void {
  const orphans = primitives.filter(p => types.includes(p.type) && !positions.has(p.id));
  if (orphans.length === 0) return;
  // 优先填入未被占用的 lane；不够则纵向均分主区剩余部分
  if (lanes.length >= orphans.length) {
    orphans.forEach((p, i) => {
      const lane = lanes[i]!;
      positions.set(p.id, {
        x: main.x + main.width / 2, y: lane.midY,
        width: main.width, height: lane.height,
      });
    });
    return;
  }
  const stepH = main.height / orphans.length;
  orphans.forEach((p, i) => {
    positions.set(p.id, {
      x: main.x + main.width / 2,
      y: main.y + stepH * (i + 0.5),
      width: main.width,
      height: stepH * 0.8,
    });
  });
  // compact 模式下统一压缩高度
  if (compact) {
    for (const p of orphans) {
      const pos = positions.get(p.id);
      if (pos && pos.height) pos.height *= 0.85;
    }
  }
}

function findFirstByType(primitives: readonly Primitive[], type: PrimitiveType): Primitive | null {
  for (const p of primitives) if (p.type === type) return p;
  return null;
}

function nodeRadius(compact: boolean): number {
  // 内容驱动可在 drawer 端再放大，这里给出基准
  return compact
    ? clamp(NODE_RADIUS_MIN_COMPACT + 2, NODE_RADIUS_MIN_COMPACT, NODE_RADIUS_MAX_COMPACT)
    : clamp(NODE_RADIUS_MIN + 4, NODE_RADIUS_MIN, NODE_RADIUS_MAX);
}

function readNodeRadius(p: Primitive, compact: boolean): number {
  const size = readOptNumber(p.params, "size");
  const min = compact ? NODE_RADIUS_MIN_COMPACT : NODE_RADIUS_MIN;
  const max = compact ? NODE_RADIUS_MAX_COMPACT : NODE_RADIUS_MAX;
  if (size === null) return nodeRadius(compact);
  // size 也是逻辑 0~1 时 * min(main)，否则当像素值
  const px = size > 0 && size <= 1 ? size * 100 : size;
  return clamp(px, min, max);
}

function readLabelText(p: Primitive): string {
  // 几乎所有原语都有 label / text 字段（按 schema）
  const label = p.params["label"];
  if (typeof label === "string" && label) return label;
  const text = p.params["text"];
  if (typeof text === "string" && text) return text;
  return "";
}

function readOptNumber(params: JsonObject, key: string): number | null {
  const v: JsonValue | undefined = params[key];
  if (typeof v === "number" && Number.isFinite(v)) return v;
  return null;
}

function logicalToPixelX(logical: number, main: ResolvedLayout["main"]): number {
  return main.x + clamp(logical, 0, 1) * main.width;
}
function logicalToPixelY(logical: number, main: ResolvedLayout["main"]): number {
  return main.y + clamp(logical, 0, 1) * main.height;
}
function logicalSpan(logical: number, span: number): number {
  return clamp(logical, 0, 1) * span;
}

function defaultMeasureText(text: string): number {
  return text.length * AVG_CHAR_WIDTH;
}

// 防止 unused 警告：asObject 在后续 drawer 大量使用，但 layoutSolver 本身只读基础字段
void asObject;
