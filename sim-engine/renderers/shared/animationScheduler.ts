/**
 * animationScheduler.ts — RAF 帧循环调度。
 *
 * 职责：
 *   • start(tick) 启动 requestAnimationFrame 循环，每帧调 tick(now)。
 *   • stop() 取消。
 *   • 不持有任何渲染状态——状态由调用方（SceneView）管理。
 */

export type FrameTick = (now: number) => void;

export class AnimationScheduler {
  private rafId: number | null = null;
  private tick: FrameTick | null = null;

  start(tick: FrameTick): void {
    if (this.rafId !== null) {
      throw new Error("AnimationScheduler.start: 已在运行，先 stop()");
    }
    this.tick = tick;
    const loop = (now: number): void => {
      if (this.tick) {
        this.tick(now);
        this.rafId = requestAnimationFrame(loop);
      }
    };
    this.rafId = requestAnimationFrame(loop);
  }

  stop(): void {
    if (this.rafId !== null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
    }
    this.tick = null;
  }

  isRunning(): boolean {
    return this.rafId !== null;
  }
}
