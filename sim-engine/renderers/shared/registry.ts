// registry.ts 维护按领域编码查找共享渲染器的注册表。
import type { DomainRenderer } from "./domainRenderer.js";
import type { SceneCategory } from "./types.js";

/**
 * DomainRendererRegistry 管理领域渲染器注册与查找。
 */
export class DomainRendererRegistry {
  private readonly renderers = new Map<SceneCategory, DomainRenderer>();

  /**
   * register 注册一个领域渲染器。
   */
  public register(renderer: DomainRenderer): void {
    this.renderers.set(renderer.domain, renderer);
  }

  /**
   * get 根据领域编码获取渲染器。
   */
  public get(domain: SceneCategory): DomainRenderer {
    const renderer = this.renderers.get(domain);
    if (!renderer) {
      throw new Error(`未注册领域渲染器: ${domain}`);
    }
    return renderer;
  }

  /**
   * has 判断某个领域是否已注册。
   */
  public has(domain: SceneCategory): boolean {
    return this.renderers.has(domain);
  }

  /**
   * list 返回全部已注册渲染器。
   */
  public list(): DomainRenderer[] {
    return Array.from(this.renderers.values());
  }
}
