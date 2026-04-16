import type {
  InteractionEvent,
  RenderConfig,
  RenderContext,
  RenderMetric,
  RenderState,
  SceneCategory,
  SimAction,
  TimelineEvent
} from "./types.js";
import { DEFAULT_THEME, asArray, asNumber, asObject, asString, clamp, lerp } from "./utils.js";

/**
 * BaseCanvasRenderer 提供领域渲染器共享的画布绘制能力。
 */
export abstract class BaseCanvasRenderer {
  /**
   * MAX_VISIBLE_NODES 定义单场景默认最大可直接渲染节点数。
   */
  protected static readonly MAX_VISIBLE_NODES = 100;

  /**
   * MAX_VISIBLE_FLOW_ITEMS 定义高频流动画面默认最大并发元素数。
   */
  protected static readonly MAX_VISIBLE_FLOW_ITEMS = 50;

  /**
   * domain 标识当前渲染器所属领域。
   */
  public abstract readonly domain: SceneCategory;

  /**
   * supportedAlgorithms 声明当前领域支持的算法类型。
   */
  public abstract readonly supportedAlgorithms: readonly string[];

  /**
   * getRenderConfig 返回算法类型对应的渲染配置。
   */
  public abstract getRenderConfig(algorithmType: string): RenderConfig;

  /**
   * render 执行领域画面绘制。
   */
  public abstract render(state: RenderState, context: RenderContext): void;

  /**
   * handleInteraction 根据交互事件构造标准操作。
   */
  public handleInteraction(event: InteractionEvent, state: RenderState): SimAction | null {
    if (event.type !== "click") {
      return null;
    }
    const action = state.schema?.actions.find((item) => item.trigger === "click");
    if (!action) {
      return null;
    }
    return {
      sceneCode: state.sceneCode,
      actionCode: action.actionCode,
      params: {
        x: event.x ?? 0,
        y: event.y ?? 0,
        resource_id: event.targetId ?? ""
      }
    };
  }

  /**
   * interpolate 在两帧之间做线性插值。
   */
  public interpolate(previous: RenderState, next: RenderState, progress: number): RenderState {
    const current = progress >= 1 ? next : previous;
    const renderData = asObject(current.renderData);
    const interpolated = { ...current };
    if (typeof renderData.progress === "number") {
      interpolated.renderData = {
        ...renderData,
        progress: lerp(
          asNumber(asObject(previous.renderData).progress, 0),
          asNumber(asObject(next.renderData).progress, 0),
          clamp(progress, 0, 1)
        )
      };
    }
    return interpolated;
  }

  /**
   * beginFrame 初始化当前帧画布。
   */
  protected beginFrame(context: RenderContext, title: string, subtitle?: string): CanvasRenderingContext2D {
    const canvasContext = context.surface.canvas.getContext("2d");
    if (!canvasContext) {
      throw new Error("画布上下文不可用");
    }
    canvasContext.setTransform(1, 0, 0, 1, 0, 0);
    canvasContext.clearRect(0, 0, context.width, context.height);
    canvasContext.fillStyle = DEFAULT_THEME.background;
    canvasContext.fillRect(0, 0, context.width, context.height);
    this.drawGrid(canvasContext, context.width, context.height);
    this.drawTitle(canvasContext, title, subtitle);
    return canvasContext;
  }

  /**
   * drawTitle 绘制场景标题与副标题。
   */
  protected drawTitle(ctx: CanvasRenderingContext2D, title: string, subtitle?: string): void {
    ctx.fillStyle = DEFAULT_THEME.foreground;
    ctx.font = "600 18px sans-serif";
    ctx.fillText(title, 20, 28);
    if (subtitle) {
      ctx.fillStyle = DEFAULT_THEME.muted;
      ctx.font = "12px sans-serif";
      ctx.fillText(subtitle, 20, 48);
    }
  }

  /**
   * drawGrid 绘制轻量背景网格。
   */
  protected drawGrid(ctx: CanvasRenderingContext2D, width: number, height: number): void {
    ctx.save();
    ctx.strokeStyle = DEFAULT_THEME.grid;
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

  /**
   * drawMetricCards 在画面顶部绘制指标卡片。
   */
  protected drawMetricCards(ctx: CanvasRenderingContext2D, metrics: RenderMetric[]): void {
    metrics.forEach((metric, index) => {
      const x = 20 + index * 150;
      ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
      ctx.strokeStyle = this.resolveTone(metric.tone);
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.roundRect(x, 62, 136, 54, 12);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = DEFAULT_THEME.muted;
      ctx.font = "11px sans-serif";
      ctx.fillText(metric.label, x + 12, 82);
      ctx.fillStyle = DEFAULT_THEME.foreground;
      ctx.font = "600 16px sans-serif";
      ctx.fillText(metric.value, x + 12, 102);
    });
  }

  /**
   * drawTooltip 绘制悬停详情提示框。
   */
  protected drawTooltip(
    ctx: CanvasRenderingContext2D,
    items: Array<{ label: string; value: string }>,
    x: number,
    y: number
  ): void {
    if (items.length === 0) {
      return;
    }
    const width = 220;
    const height = 20 + items.length * 20;
    ctx.fillStyle = "rgba(9, 17, 31, 0.94)";
    ctx.strokeStyle = DEFAULT_THEME.accent;
    ctx.beginPath();
    ctx.roundRect(x, y, width, height, 12);
    ctx.fill();
    ctx.stroke();
    items.forEach((item, index) => {
      const lineY = y + 22 + index * 18;
      ctx.fillStyle = DEFAULT_THEME.muted;
      ctx.font = "11px sans-serif";
      ctx.fillText(item.label, x + 12, lineY);
      ctx.fillStyle = DEFAULT_THEME.foreground;
      ctx.fillText(item.value, x + 92, lineY);
    });
  }

  /**
   * drawChangedKeys 绘制变化字段高亮摘要。
   */
  protected drawChangedKeys(ctx: CanvasRenderingContext2D, changedKeys: string[], width: number): void {
    if (changedKeys.length === 0) {
      return;
    }
    ctx.fillStyle = "rgba(18, 29, 48, 0.94)";
    ctx.strokeStyle = DEFAULT_THEME.warning;
    ctx.beginPath();
    ctx.roundRect(width - 260, 206, 240, 96, 16);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = DEFAULT_THEME.foreground;
    ctx.font = "600 13px sans-serif";
    ctx.fillText("状态变化高亮", width - 244, 230);
    ctx.font = "11px sans-serif";
    changedKeys.slice(0, 4).forEach((item, index) => {
      ctx.fillText(item, width - 244, 252 + index * 16);
    });
  }

  /**
   * drawAnnotations 绘制文本标注。
   */
  protected drawAnnotations(ctx: CanvasRenderingContext2D, state: RenderState): void {
    (state.annotations ?? []).forEach((annotation) => {
      ctx.fillStyle = annotation.color ?? DEFAULT_THEME.warning;
      ctx.beginPath();
      ctx.roundRect(annotation.x, annotation.y, Math.max(88, annotation.text.length * 12), 24, 10);
      ctx.fill();
      ctx.fillStyle = "#09111f";
      ctx.font = "12px sans-serif";
      ctx.fillText(annotation.text, annotation.x + 10, annotation.y + 16);
    });
  }

  /**
   * drawTimeline 绘制事件时间线。
   */
  protected drawTimeline(ctx: CanvasRenderingContext2D, width: number, height: number, events: TimelineEvent[]): void {
    const baseY = height - 48;
    ctx.strokeStyle = DEFAULT_THEME.grid;
    ctx.beginPath();
    ctx.moveTo(20, baseY);
    ctx.lineTo(width - 20, baseY);
    ctx.stroke();
    events.slice(-6).forEach((event, index) => {
      const x = 40 + index * ((width - 80) / 5);
      ctx.fillStyle = this.resolveTone(event.tone);
      ctx.beginPath();
      ctx.arc(x, baseY, 6, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = DEFAULT_THEME.foreground;
      ctx.font = "11px sans-serif";
      ctx.fillText(event.title, x - 18, baseY + 22);
    });
  }

  /**
   * drawBadge 绘制联动或状态角标。
   */
  protected drawBadge(
    ctx: CanvasRenderingContext2D,
    text: string,
    tone: "info" | "success" | "warning" | "danger" = "info"
  ): void {
    const width = Math.max(72, text.length * 14);
    ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
    ctx.strokeStyle = this.resolveTone(tone);
    ctx.beginPath();
    ctx.roundRect(20, 126, width, 28, 999);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = DEFAULT_THEME.foreground;
    ctx.font = "12px sans-serif";
    ctx.fillText(text, 32, 145);
  }

  /**
   * drawListPanel 绘制通用列表信息卡。
   */
  protected drawListPanel(
    ctx: CanvasRenderingContext2D,
    title: string,
    items: Array<{ label: string; value: string }>,
    x: number,
    y: number,
    width: number,
    height: number
  ): void {
    ctx.fillStyle = "rgba(18, 29, 48, 0.94)";
    ctx.strokeStyle = DEFAULT_THEME.grid;
    ctx.beginPath();
    ctx.roundRect(x, y, width, height, 16);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = DEFAULT_THEME.foreground;
    ctx.font = "600 14px sans-serif";
    ctx.fillText(title, x + 14, y + 22);
    ctx.font = "12px sans-serif";
    items.forEach((item, index) => {
      const lineY = y + 48 + index * 20;
      ctx.fillStyle = DEFAULT_THEME.muted;
      ctx.fillText(item.label, x + 14, lineY);
      ctx.fillStyle = DEFAULT_THEME.foreground;
      ctx.fillText(item.value, x + width - 120, lineY);
    });
  }

  /**
   * drawProcessStages 绘制过程化场景的阶段轨道与当前阶段高亮。
   */
  protected drawProcessStages(
    ctx: CanvasRenderingContext2D,
    stages: string[],
    activeIndex: number,
    progress: number,
    x: number,
    y: number,
    width: number
  ): void {
    if (stages.length === 0) {
      return;
    }
    const gap = width / Math.max(1, stages.length - 1);
    ctx.strokeStyle = DEFAULT_THEME.grid;
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + width, y);
    ctx.stroke();
    stages.forEach((stage, index) => {
      const dotX = x + gap * index;
      const isActive = index === activeIndex;
      const isCompleted = index < activeIndex;
      ctx.fillStyle = isCompleted ? DEFAULT_THEME.success : isActive ? DEFAULT_THEME.accent : "#24344e";
      ctx.beginPath();
      ctx.arc(dotX, y, isActive ? 10 : 8, 0, Math.PI * 2);
      ctx.fill();
      if (isActive) {
        ctx.strokeStyle = DEFAULT_THEME.warning;
        ctx.beginPath();
        ctx.arc(dotX, y, 14, -Math.PI / 2, -Math.PI / 2 + Math.PI * 2 * clamp(progress, 0, 1));
        ctx.stroke();
      }
      ctx.fillStyle = DEFAULT_THEME.foreground;
      ctx.font = "11px sans-serif";
      ctx.fillText(stage, dotX - Math.max(18, stage.length * 3), y + 28);
    });
  }

  /**
   * drawComparisonPanel 绘制文档要求的攻防或参数 A/B 对比视图。
   */
  protected drawComparisonPanel(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    width: number,
    height: number,
    leftTitle: string,
    rightTitle: string,
    leftItems: Array<{ label: string; value: string }>,
    rightItems: Array<{ label: string; value: string }>
  ): void {
    const columnWidth = (width - 18) / 2;
    this.drawListPanel(ctx, leftTitle, leftItems, x, y, columnWidth, height);
    this.drawListPanel(ctx, rightTitle, rightItems, x + columnWidth + 18, y, columnWidth, height);
  }

  /**
   * drawAggregationNotice 在高负载降级时提示已聚合显示。
   */
  protected drawAggregationNotice(
    ctx: CanvasRenderingContext2D,
    label: string,
    total: number,
    visible: number,
    x: number,
    y: number
  ): void {
    if (total <= visible) {
      return;
    }
    ctx.fillStyle = "rgba(18, 29, 48, 0.92)";
    ctx.strokeStyle = DEFAULT_THEME.warning;
    ctx.beginPath();
    ctx.roundRect(x, y, 224, 34, 12);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = DEFAULT_THEME.foreground;
    ctx.font = "11px sans-serif";
    ctx.fillText(`${label} 已聚合显示 ${visible}/${total}`, x + 12, y + 21);
  }

  /**
   * renderOverlaySVG 在提供 overlay 容器时渲染 SVG 辅助图层。
   */
  protected renderOverlaySVG(context: RenderContext, svgContent: string): void {
    const overlay = context.surface.overlay;
    if (!overlay) {
      return;
    }
    overlay.innerHTML = [
      `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${context.width} ${context.height}" width="${context.width}" height="${context.height}">`,
      svgContent,
      "</svg>"
    ].join("");
  }

  /**
   * clearOverlaySVG 清空上一帧可能残留的 SVG 图层。
   */
  protected clearOverlaySVG(context: RenderContext): void {
    if (context.surface.overlay) {
      context.surface.overlay.innerHTML = "";
    }
  }

  /**
   * resolveTone 将语义色映射为实际颜色。
   */
  protected resolveTone(tone: string | undefined): string {
    switch (tone) {
      case "success":
        return DEFAULT_THEME.success;
      case "warning":
        return DEFAULT_THEME.warning;
      case "danger":
        return DEFAULT_THEME.danger;
      default:
        return DEFAULT_THEME.accent;
    }
  }

  /**
   * nodePoints 读取状态中的节点数组。
   */
  protected nodePoints(state: RenderState): Array<{ id: string; x: number; y: number; label: string; status: string }> {
    return asArray<unknown>(asObject(state.renderData).nodes)
      .slice(0, BaseCanvasRenderer.MAX_VISIBLE_NODES)
      .map((node, index) => {
      const nodeObject = asObject(node);
      return {
        id: asString(nodeObject.id, `node-${index + 1}`),
        x: asNumber(nodeObject.x, 120 + index * 96),
        y: asNumber(nodeObject.y, 220),
        label: asString(nodeObject.label, asString(nodeObject.id, `节点${index + 1}`)),
        status: asString(nodeObject.status, "normal")
      };
      });
  }
}
