import type { PanelLayoutItem } from "./types.js";

/**
 * PanelLayoutStore 负责保存与恢复场景面板布局。
 */
export class PanelLayoutStore {
  /**
   * constructor 初始化布局存储器。
   */
  public constructor(private readonly storageKey: string) {}

  /**
   * save 保存布局配置到本地存储。
   */
  public save(layout: PanelLayoutItem[]): void {
    if (!this.hasStorage()) {
      return;
    }
    localStorage.setItem(this.storageKey, JSON.stringify(layout));
  }

  /**
   * load 读取本地存储中的布局配置。
   */
  public load(): PanelLayoutItem[] {
    if (!this.hasStorage()) {
      return [];
    }
    const raw = localStorage.getItem(this.storageKey);
    if (!raw) {
      return [];
    }
    try {
      return JSON.parse(raw) as PanelLayoutItem[];
    } catch {
      return [];
    }
  }

  /**
   * hasStorage 判断当前运行环境是否可用浏览器本地存储。
   */
  private hasStorage(): boolean {
    return typeof window !== "undefined" && typeof localStorage !== "undefined";
  }
}
