/**
 * sceneView.ts — 单场景画布视图。
 *
 * 职责：
 *   • 绑定 canvas + RenderState；驱动 RAF；每帧调用 renderFrame。
 *   • 持有 MicroStepScheduler 提供 highlight / fire 集合。
 *   • setState() 接收外部 (SimPanel) 推送的新 RenderState。
 *   • 自装 ResizeObserver：canvas 尺寸变化自动 setSize + 重绘。
 *   • requestRender() RAF 节流：多次连续 setState 合并为单帧绘制。
 *   • getResolvedLayout() 暴露最近一次解算结果，供前端 DOM 浮层定位。
 *
 * 不写兑底：画布无 2d 上下文 / state 未设置 → 抛错。
 */

import { AnimationScheduler } from "./animationScheduler.js";
import { MicroStepScheduler } from "./microStepScheduler.js";
import { renderFrame } from "./primitiveRenderer.js";
import type { ResolvedLayout } from "./layoutSolver.js";
import { themeForCategory } from "./theme.js";
import type { RenderState } from "./types.js";

export interface SceneViewOptions {
  canvas: HTMLCanvasElement;
}

export class SceneView {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private state: RenderState | null = null;
  private microSteps = new MicroStepScheduler();
  private scheduler = new AnimationScheduler();
  private width = 0;
  private height = 0;
  private dpr = typeof window !== "undefined" ? window.devicePixelRatio || 1 : 1;
  private lastLayout: ResolvedLayout | null = null;
  private resizeObserver: ResizeObserver | null = null;
  private rafPending: number | null = null;

  constructor(opts: SceneViewOptions) {
    this.canvas = opts.canvas;
    const ctx = opts.canvas.getContext("2d");
    if (!ctx) {
      throw new Error("SceneView: canvas.getContext('2d') 返回 null");
    }
    this.ctx = ctx;
    this.setSize(opts.canvas.clientWidth, opts.canvas.clientHeight);
    this.installResizeObserver();
  }

  setState(state: RenderState): void {
    const prevSteps = this.state?.envelope.micro_steps ?? [];
    const nextSteps = state.envelope.micro_steps ?? [];
    if (prevSteps !== nextSteps) {
      this.microSteps.load(nextSteps);
    }
    this.state = state;
  }

  setSize(cssWidth: number, cssHeight: number): void {
    if (cssWidth <= 0 || cssHeight <= 0) return;
    if (cssWidth === this.width && cssHeight === this.height) return;
    this.width = cssWidth;
    this.height = cssHeight;
    this.canvas.width = Math.floor(cssWidth * this.dpr);
    this.canvas.height = Math.floor(cssHeight * this.dpr);
    this.canvas.style.width = `${cssWidth}px`;
    this.canvas.style.height = `${cssHeight}px`;
    this.ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
    this.lastLayout = null; // 失效旧布局缓存
  }

  play(): void {
    if (this.scheduler.isRunning()) return;
    this.scheduler.start((now) => {
      this.microSteps.play(now);
      this.renderOnce(now);
    });
  }

  pause(now: number): void {
    this.microSteps.pause(now);
    this.scheduler.stop();
    this.renderOnce(now);
  }

  reset(): void {
    this.microSteps.reset();
    if (this.state) this.renderOnce(performance.now());
  }

  setSpeed(multiplier: number): void {
    this.microSteps.setSpeed(multiplier);
  }

  /**
   * 请求一次合并渲染。多次 requestRender 在同一 rAF 内只触发一次实际绘制。
   *
   * 适用：外部（SimPanel）连续 setState 后调用，避免每次都跑 layoutSolver。
   * 注意：play 模式下 RAF 循环已驱动重绘，此函数仍安全（合并为下一帧首次）。
   */
  requestRender(): void {
    if (this.rafPending !== null) return;
    this.rafPending = requestAnimationFrame((now) => {
      this.rafPending = null;
      if (this.state) this.renderOnce(now);
    });
  }

  /** 同步渲染一帧（不节流）。外部更建议使用 requestRender。 */
  renderOnce(now: number): void {
    if (!this.state) {
      throw new Error("SceneView.renderOnce: state 未设置");
    }
    const status = this.microSteps.advance(now);
    this.lastLayout = renderFrame({
      ctx: this.ctx,
      width: this.width,
      height: this.height,
      theme: themeForCategory(this.state.category),
      primitives: this.state.envelope.primitives,
      highlightIds: status.activeHighlightIds,
      fireIds: status.activeFireIds,
      now,
      tick: this.state.tick,
    });
  }

  /** 返回最近一次解算的 ResolvedLayout（供前端 React DOM 浮层定位用）。 */
  getResolvedLayout(): ResolvedLayout | null {
    return this.lastLayout;
  }

  /** 输出当前画布的 PNG dataURL（截图）。 */
  toDataURL(): string {
    return this.canvas.toDataURL("image/png");
  }

  dispose(): void {
    this.scheduler.stop();
    if (this.rafPending !== null) {
      cancelAnimationFrame(this.rafPending);
      this.rafPending = null;
    }
    if (this.resizeObserver) {
      this.resizeObserver.disconnect();
      this.resizeObserver = null;
    }
    this.state = null;
    this.lastLayout = null;
  }

  private installResizeObserver(): void {
    if (typeof ResizeObserver === "undefined") return;
    this.resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const cr = entry.contentRect;
        this.setSize(cr.width, cr.height);
        this.requestRender();
      }
    });
    this.resizeObserver.observe(this.canvas);
  }
}
