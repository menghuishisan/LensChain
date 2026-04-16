import type { DomainRenderer } from "./domainRenderer.js";
import type { PanelLayoutItem, RenderState, RenderSurface } from "./types.js";
import { AnimationScheduler } from "./animationScheduler.js";
import { AnnotationStore, captureCanvas, startCanvasRecording } from "./mediaToolkit.js";
import { FallbackRenderer } from "./fallbackRenderer.js";
import { ViewportController } from "./viewportController.js";

/**
 * SceneView 管理单个场景面板的渲染生命周期。
 */
export class SceneView {
  private readonly viewport = new ViewportController();
  private readonly scheduler = new AnimationScheduler();
  private readonly annotations = new AnnotationStore();
  private currentState?: RenderState;
  private readonly fallbackRenderer: FallbackRenderer;

  /**
   * constructor 初始化场景视图。
   */
  public constructor(
    public readonly surface: RenderSurface,
    public readonly renderer: DomainRenderer,
    public readonly layout: PanelLayoutItem
  ) {
    this.fallbackRenderer = new FallbackRenderer(renderer.domain, "当前场景渲染异常，已切换到失败提示视图");
  }

  /**
   * setState 设置场景状态并触发渲染。
   */
  public setState(state: RenderState): void {
    const previous = this.currentState;
    this.currentState = state;
    if (!previous) {
      this.render(state);
      return;
    }
    this.scheduler.start(`scene-${state.sceneCode}`, 240, (progress) => {
      try {
        const frame = this.renderer.interpolate(previous, state, progress);
        this.render(frame);
      } catch {
        this.renderFailure(state);
      }
    });
  }

  /**
   * render 使用当前渲染器执行绘制。
   */
  public render(state: RenderState): void {
    this.currentState = {
      ...state,
      annotations: this.annotations.list()
    };
    try {
      this.renderer.render(this.currentState, this.createContext());
    } catch {
      this.renderFailure(this.currentState);
    }
  }

  /**
   * zoom 调整视图缩放。
   */
  public zoom(delta: number): void {
    this.viewport.zoom(delta);
    if (this.currentState) {
      this.render(this.currentState);
    }
  }

  /**
   * capture 导出当前画面截图。
   */
  public capture(): string {
    return captureCanvas(this.surface.canvas);
  }

  /**
   * captureThumbnail 导出教师监控卡片使用的缩略图。
   */
  public captureThumbnail(maxWidth = 320, maxHeight = 180): string {
    const source = this.surface.canvas;
    const ratio = Math.min(maxWidth / source.width, maxHeight / source.height, 1);
    const canvas = document.createElement("canvas");
    canvas.width = Math.max(1, Math.round(source.width * ratio));
    canvas.height = Math.max(1, Math.round(source.height * ratio));
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("缩略图画布上下文不可用");
    }
    ctx.drawImage(source, 0, 0, canvas.width, canvas.height);
    return canvas.toDataURL("image/png");
  }

  /**
   * record 启动画布录制。
   */
  public record(): MediaRecorder {
    return startCanvasRecording(this.surface.canvas);
  }

  /**
   * addAnnotation 为当前场景增加文本标注。
   */
  public addAnnotation(text: string, x: number, y: number): void {
    this.annotations.add(text, x, y);
    if (this.currentState) {
      this.render(this.currentState);
    }
  }

  /**
   * removeAnnotation 删除指定标注。
   */
  public removeAnnotation(id: string): void {
    this.annotations.remove(id);
    if (this.currentState) {
      this.render(this.currentState);
    }
  }

  /**
   * clearAnnotations 清空当前场景全部标注。
   */
  public clearAnnotations(): void {
    this.annotations.clear();
    if (this.currentState) {
      this.render(this.currentState);
    }
  }

  /**
   * getState 返回当前场景状态快照。
   */
  public getState(): RenderState | undefined {
    return this.currentState ? { ...this.currentState } : undefined;
  }

  /**
   * createContext 生成渲染器执行上下文。
   */
  private createContext() {
    return {
      surface: this.surface,
      width: this.surface.canvas.width,
      height: this.surface.canvas.height,
      viewport: this.viewport.getState(),
      now: performance.now()
    };
  }

  /**
   * renderFailure 将单场景渲染故障隔离到失败提示视图。
   */
  private renderFailure(state: RenderState): void {
    this.fallbackRenderer.render(state, this.createContext());
  }
}
