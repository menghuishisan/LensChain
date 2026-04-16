import type { ViewportState } from "./types.js";
import { clamp } from "./utils.js";

/**
 * ViewportController 管理场景画布的平移与缩放。
 */
export class ViewportController {
  private state: ViewportState = { zoom: 1, offsetX: 0, offsetY: 0 };

  /**
   * getState 返回当前视口状态。
   */
  public getState(): ViewportState {
    return { ...this.state };
  }

  /**
   * pan 对画布执行平移。
   */
  public pan(deltaX: number, deltaY: number): void {
    this.state = {
      ...this.state,
      offsetX: this.state.offsetX + deltaX,
      offsetY: this.state.offsetY + deltaY
    };
  }

  /**
   * zoom 对画布执行缩放。
   */
  public zoom(delta: number): void {
    this.state = {
      ...this.state,
      zoom: clamp(this.state.zoom + delta, 0.5, 2.5)
    };
  }

  /**
   * reset 将视口恢复到默认值。
   */
  public reset(): void {
    this.state = { zoom: 1, offsetX: 0, offsetY: 0 };
  }
}
