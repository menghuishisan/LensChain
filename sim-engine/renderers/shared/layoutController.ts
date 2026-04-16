import type { PanelLayoutItem } from "./types.js";

/**
 * LayoutController 管理场景面板的拖拽、缩放与全屏布局。
 */
export class LayoutController {
  private layout: PanelLayoutItem[];
  private readonly previousLayout = new Map<string, PanelLayoutItem>();

  /**
   * constructor 初始化布局控制器。
   */
  public constructor(initialLayout: PanelLayoutItem[]) {
    this.layout = [...initialLayout];
  }

  /**
   * list 返回当前布局快照。
   */
  public list(): PanelLayoutItem[] {
    return this.layout.map((item) => ({ ...item }));
  }

  /**
   * upsert 写入或更新单个场景布局。
   */
  public upsert(item: PanelLayoutItem): void {
    const normalized = this.normalize(item);
    const index = this.layout.findIndex((current) => current.sceneCode === item.sceneCode);
    if (index >= 0) {
      this.layout[index] = normalized;
      return;
    }
    this.layout.push(normalized);
  }

  /**
   * move 调整场景面板位置。
   */
  public move(sceneCode: string, x: number, y: number): void {
    this.layout = this.layout.map((item) =>
      item.sceneCode === sceneCode ? this.normalize({ ...item, x, y }) : item
    );
  }

  /**
   * resize 调整场景面板尺寸。
   */
  public resize(sceneCode: string, w: number, h: number): void {
    this.layout = this.layout.map((item) =>
      item.sceneCode === sceneCode ? this.normalize({ ...item, w, h }) : item
    );
  }

  /**
   * toggleFullscreen 切换单个场景全屏状态。
   */
  public toggleFullscreen(sceneCode: string): void {
    const target = this.layout.find((item) => item.sceneCode === sceneCode);
    if (!target) {
      return;
    }
    const willEnterFullscreen = !target.fullscreen;
    if (willEnterFullscreen) {
      this.previousLayout.set(sceneCode, { ...target });
    }
    this.layout = this.layout.map((item) => {
      if (item.sceneCode !== sceneCode) {
        return { ...item, fullscreen: false };
      }
      if (willEnterFullscreen) {
        return { ...item, fullscreen: true, x: 0, y: 0, w: 12, h: Math.max(6, item.h) };
      }
      const previous = this.previousLayout.get(sceneCode);
      this.previousLayout.delete(sceneCode);
      return previous ? this.normalize({ ...previous, fullscreen: false }) : this.normalize({ ...item, fullscreen: false });
    });
  }

  /**
   * normalize 将布局限制在 12 列栅格范围内。
   */
  private normalize(item: PanelLayoutItem): PanelLayoutItem {
    const width = Math.min(12, Math.max(1, Math.round(item.w)));
    const x = Math.min(11, Math.max(0, Math.round(item.x)));
    const normalizedX = Math.min(x, 12 - width);
    return {
      ...item,
      x: normalizedX,
      y: Math.max(0, Math.round(item.y)),
      w: width,
      h: Math.max(1, Math.round(item.h))
    };
  }
}
