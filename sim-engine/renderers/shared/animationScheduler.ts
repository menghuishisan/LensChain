/**
 * AnimationScheduler 负责管理场景过渡动画调度。
 */
export class AnimationScheduler {
  private readonly handles = new Map<string, number>();

  /**
   * start 启动一次补间动画。
   */
  public start(
    key: string,
    durationMS: number,
    onFrame: (progress: number) => void,
    onDone?: () => void
  ): void {
    this.stop(key);
    const startedAt = performance.now();
    const step = (now: number) => {
      const progress = Math.min(1, (now - startedAt) / durationMS);
      onFrame(progress);
      if (progress >= 1) {
        this.handles.delete(key);
        onDone?.();
        return;
      }
      this.handles.set(key, requestAnimationFrame(step));
    };
    this.handles.set(key, requestAnimationFrame(step));
  }

  /**
   * stop 停止指定动画。
   */
  public stop(key: string): void {
    const handle = this.handles.get(key);
    if (handle !== undefined) {
      cancelAnimationFrame(handle);
      this.handles.delete(key);
    }
  }

  /**
   * clear 停止全部动画。
   */
  public clear(): void {
    for (const key of this.handles.keys()) {
      this.stop(key);
    }
  }
}
