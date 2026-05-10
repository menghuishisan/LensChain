import type {
  InteractionEvent,
  Primitive,
  PrimitiveLayer,
  RenderConfig,
  RenderContext,
  RenderMetric,
  RenderState,
  SceneCategory,
  SimAction,
  TimelineEvent
} from "./types.js";
import { DEFAULT_THEME } from "./utils.js";
import { type DrawEnvironment, type PrimitiveDrawFn, FALLBACK_DRAWER, PRIMITIVE_DRAWER_MAP } from "./primitiveDrawers.js";

/** 层级绘制顺序。 */
const LAYER_ORDER: PrimitiveLayer[] = ["background", "content", "effect", "overlay"];

/**
 * PrimitiveBasedRenderer 原语驱动渲染基类。
 * 通过 dispatch table 绘制全部 47 原语。
 * 领域子类仅需覆盖 getTheme() 提供配色，可选覆盖单个原语的绘制。
 */
export abstract class PrimitiveBasedRenderer {
  /**
   * domain 标识当前渲染器所属领域。
   */
  public abstract readonly domain: SceneCategory;

  /**
   * getTheme 返回领域主题配色。
   */
  public abstract getTheme(): RenderConfig;

  /**
   * drawerOverrides 子类可覆盖部分原语的绘制函数。
   */
  protected drawerOverrides: Partial<Record<string, PrimitiveDrawFn>> = {};

  /**
   * render 执行原语 dispatch 绘制。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title);
    const theme = context.theme ?? this.getTheme().theme;
    const primitives = state.envelope.primitives ?? [];

    const highlightIds = new Set<string>();
    const fireIds = new Set<string>();
    if (state.activeMicroStepId) {
      const step = (state.envelope.micro_steps ?? []).find((s) => s.id === state.activeMicroStepId);
      if (step) {
        (step.highlight_ids ?? []).forEach((id) => highlightIds.add(id));
        (step.fire_primitives ?? []).forEach((id) => fireIds.add(id));
      }
    }

    const env: DrawEnvironment = { ctx, context, theme, highlightIds, fireIds, now: context.now };

    for (const layer of LAYER_ORDER) {
      const layerPrimitives = primitives.filter((p) => p.layer === layer);
      for (const p of layerPrimitives) {
        ctx.save();
        const drawer = this.drawerOverrides[p.type] ?? PRIMITIVE_DRAWER_MAP[p.type] ?? FALLBACK_DRAWER;
        drawer(p, env);
        ctx.restore();
      }
    }

    this.drawMetricCards(ctx, state.metrics ?? [], theme);
    this.drawAnnotations(ctx, state);
    if ((state.timeline?.length ?? 0) > 0) {
      this.drawTimeline(ctx, context.width, context.height, state.timeline ?? [], theme);
    }
    this.drawChangedKeys(ctx, state.changedKeys ?? [], context.width, theme);
    if (state.linked && state.linkGroupName) {
      this.drawBadge(ctx, `联动: ${state.linkGroupName}`, "success", theme);
    }
  }

  /**
   * handleInteraction 根据交互事件构造标准操作。
   * 事件坐标经过视口逆变换后再做命中检测，确保 zoom/pan 状态下正确命中。
   */
  public handleInteraction(event: InteractionEvent, state: RenderState): SimAction | null {
    if (event.type !== "click") return null;
    const primitives = state.envelope.primitives ?? [];
    const viewportEvent = this.applyViewportInverse(event);
    const clicked = primitives.find((p) => p.clickable && p.click_action && this.hitTest(p, viewportEvent));
    if (clicked?.click_action) {
      return {
        sceneCode: state.sceneCode,
        actionCode: clicked.click_action,
        params: { primitive_id: clicked.id, x: viewportEvent.x ?? 0, y: viewportEvent.y ?? 0 }
      };
    }
    const action = state.schema?.actions.find((item) => item.trigger === "immediate");
    if (!action) return null;
    return {
      sceneCode: state.sceneCode,
      actionCode: action.actionCode,
      params: { x: event.x ?? 0, y: event.y ?? 0, resource_id: event.targetId ?? "" }
    };
  }

  /**
   * applyViewportInverse 将浏览器坐标转为画布世界坐标。
   */
  private viewportZoom = 1;
  private viewportOffsetX = 0;
  private viewportOffsetY = 0;

  public setViewport(zoom: number, offsetX: number, offsetY: number): void {
    this.viewportZoom = zoom;
    this.viewportOffsetX = offsetX;
    this.viewportOffsetY = offsetY;
  }

  private applyViewportInverse(event: InteractionEvent): InteractionEvent {
    if (this.viewportZoom === 1 && this.viewportOffsetX === 0 && this.viewportOffsetY === 0) {
      return event;
    }
    const result: InteractionEvent = { sceneCode: event.sceneCode, type: event.type };
    if (event.x != null) result.x = (event.x - this.viewportOffsetX) / this.viewportZoom;
    if (event.y != null) result.y = (event.y - this.viewportOffsetY) / this.viewportZoom;
    if (event.deltaX != null) result.deltaX = event.deltaX;
    if (event.deltaY != null) result.deltaY = event.deltaY;
    if (event.targetId != null) result.targetId = event.targetId;
    return result;
  }

  /**
   * interpolate 在两帧之间做线性过渡（默认直接返回目标帧）。
   */
  public interpolate(_previous: RenderState, next: RenderState, _progress: number): RenderState {
    return next;
  }

  /**
   * beginFrame 初始化当前帧画布（含 DPR 修复）。
   */
  protected beginFrame(context: RenderContext, title: string, subtitle?: string): CanvasRenderingContext2D {
    const canvas = context.surface.canvas;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("画布上下文不可用");
    }
    const dpr = typeof window !== "undefined" ? (window.devicePixelRatio ?? 1) : 1;
    const cssWidth = context.width;
    const cssHeight = context.height;
    if (canvas.width !== Math.round(cssWidth * dpr) || canvas.height !== Math.round(cssHeight * dpr)) {
      canvas.width = Math.round(cssWidth * dpr);
      canvas.height = Math.round(cssHeight * dpr);
      canvas.style.width = `${cssWidth}px`;
      canvas.style.height = `${cssHeight}px`;
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, cssWidth, cssHeight);

    const theme = context.theme ?? this.getTheme().theme;
    ctx.fillStyle = theme.background;
    ctx.fillRect(0, 0, cssWidth, cssHeight);
    this.drawGrid(ctx, cssWidth, cssHeight, theme);
    this.drawTitle(ctx, title, theme, subtitle);
    return ctx;
  }

  private drawTitle(ctx: CanvasRenderingContext2D, title: string, theme: typeof DEFAULT_THEME, subtitle?: string): void {
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 18px sans-serif";
    ctx.fillText(title, 20, 28);
    if (subtitle) {
      ctx.fillStyle = theme.muted;
      ctx.font = "12px sans-serif";
      ctx.fillText(subtitle, 20, 48);
    }
  }

  private drawGrid(ctx: CanvasRenderingContext2D, width: number, height: number, theme: typeof DEFAULT_THEME): void {
    ctx.save();
    ctx.strokeStyle = theme.grid;
    ctx.lineWidth = 1;
    for (let x = 0; x <= width; x += 32) {
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, height);
      ctx.stroke();
    }
    for (let y = 0; y <= height; y += 32) {
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }
    ctx.restore();
  }

  private drawMetricCards(ctx: CanvasRenderingContext2D, metrics: RenderMetric[], theme: typeof DEFAULT_THEME): void {
    metrics.slice(0, 4).forEach((metric, index) => {
      const x = 20 + index * 150;
      ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
      ctx.strokeStyle = this.resolveTone(metric.tone, theme);
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.roundRect(x, 62, 136, 54, 12);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = theme.muted;
      ctx.font = "11px sans-serif";
      ctx.fillText(metric.label, x + 12, 82);
      ctx.fillStyle = theme.foreground;
      ctx.font = "600 16px sans-serif";
      ctx.fillText(metric.value, x + 12, 102);
    });
  }

  private drawChangedKeys(ctx: CanvasRenderingContext2D, changedKeys: string[], width: number, theme: typeof DEFAULT_THEME): void {
    if (changedKeys.length === 0) return;
    ctx.fillStyle = "rgba(18, 29, 48, 0.94)";
    ctx.strokeStyle = theme.warning;
    ctx.beginPath();
    ctx.roundRect(width - 260, 206, 240, 96, 16);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = theme.foreground;
    ctx.font = "600 13px sans-serif";
    ctx.fillText("状态变化高亮", width - 244, 230);
    ctx.font = "11px sans-serif";
    changedKeys.slice(0, 4).forEach((item, index) => {
      ctx.fillText(item, width - 244, 252 + index * 16);
    });
  }

  private drawAnnotations(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const theme = DEFAULT_THEME;
    (state.annotations ?? []).forEach((annotation) => {
      ctx.fillStyle = annotation.color ?? theme.warning;
      ctx.beginPath();
      ctx.roundRect(annotation.x, annotation.y, Math.max(88, annotation.text.length * 12), 24, 10);
      ctx.fill();
      ctx.fillStyle = theme.background;
      ctx.font = "12px sans-serif";
      ctx.fillText(annotation.text, annotation.x + 10, annotation.y + 16);
    });
  }

  private drawTimeline(ctx: CanvasRenderingContext2D, width: number, height: number, events: TimelineEvent[], theme: typeof DEFAULT_THEME): void {
    const baseY = height - 48;
    ctx.strokeStyle = theme.grid;
    ctx.beginPath();
    ctx.moveTo(20, baseY);
    ctx.lineTo(width - 20, baseY);
    ctx.stroke();
    events.slice(-6).forEach((event, index) => {
      const x = 40 + index * ((width - 80) / 5);
      ctx.fillStyle = this.resolveTone(event.tone, theme);
      ctx.beginPath();
      ctx.arc(x, baseY, 6, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = theme.foreground;
      ctx.font = "11px sans-serif";
      ctx.fillText(event.title, x - 18, baseY + 22);
    });
  }

  private drawBadge(ctx: CanvasRenderingContext2D, text: string, tone: string, theme: typeof DEFAULT_THEME): void {
    const width = Math.max(72, text.length * 14);
    ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
    ctx.strokeStyle = this.resolveTone(tone, theme);
    ctx.beginPath();
    ctx.roundRect(20, 126, width, 28, 999);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = theme.foreground;
    ctx.font = "12px sans-serif";
    ctx.fillText(text, 32, 145);
  }

  private static readonly CIRCULAR_HIT_TYPES = new Set([
    "node", "pulse", "burst", "ring", "glow", "curve_point", "ring_layout"
  ]);
  private static readonly RECT_HIT_TYPES = new Set([
    "bar", "area", "grid_cell", "register_row", "mempool_slot", "code_block",
    "horizontal_lane", "stack", "dual_track", "partition_zone", "target_zone",
    "error_overlay", "progress_bar", "math_pipeline"
  ]);

  private hitTest(p: Primitive, event: InteractionEvent): boolean {
    const ex = event.x ?? 0;
    const ey = event.y ?? 0;

    if (PrimitiveBasedRenderer.CIRCULAR_HIT_TYPES.has(p.type)) {
      const cx = this.paramNum(p, "cx", this.paramNum(p, "x", -9999));
      const cy = this.paramNum(p, "cy", this.paramNum(p, "y", -9999));
      const radius = this.paramNum(p, "radius", 24);
      const dx = ex - cx;
      const dy = ey - cy;
      return dx * dx + dy * dy <= radius * radius;
    }

    if (PrimitiveBasedRenderer.RECT_HIT_TYPES.has(p.type)) {
      const rx = this.paramNum(p, "x", -9999);
      const ry = this.paramNum(p, "y", -9999);
      const rw = this.paramNum(p, "width", this.paramNum(p, "size", 32));
      const rh = this.paramNum(p, "height", this.paramNum(p, "size", 32));
      return ex >= rx && ex <= rx + rw && ey >= ry && ey <= ry + rh;
    }

    const fx = this.paramNum(p, "x", this.paramNum(p, "cx", this.paramNum(p, "x1", -9999)));
    const fy = this.paramNum(p, "y", this.paramNum(p, "cy", this.paramNum(p, "y1", -9999)));
    const fallbackRadius = this.paramNum(p, "radius", 20);
    const ddx = ex - fx;
    const ddy = ey - fy;
    return ddx * ddx + ddy * ddy <= fallbackRadius * fallbackRadius;
  }

  private paramNum(p: Primitive, key: string, fallback: number): number {
    const v = p.params[key];
    return typeof v === "number" ? v : fallback;
  }

  private resolveTone(tone: string | undefined, theme: typeof DEFAULT_THEME): string {
    switch (tone) {
      case "success": return theme.success;
      case "warning": return theme.warning;
      case "danger": return theme.danger;
      default: return theme.accent;
    }
  }
}
