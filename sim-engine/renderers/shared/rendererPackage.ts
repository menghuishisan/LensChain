// rendererPackage.ts
// SimEngine 渲染器包扩展定义
// 负责为外部 npm 渲染器包提供统一注册入口，复用现有注册表而不是再维护第二套装配逻辑。

import type { DomainRenderer } from "./domainRenderer.js";
import { DomainRendererRegistry } from "./registry.js";

/**
 * RendererPackage 描述一个可注册到平台中的自定义渲染器包。
 */
export interface RendererPackage {
  packageName: string;
  renderers: Iterable<DomainRenderer>;
}

/**
 * registerRendererPackage 将一个渲染器包中的全部渲染器注册到现有注册表。
 */
export function registerRendererPackage(registry: DomainRendererRegistry, rendererPackage: RendererPackage): void {
  for (const renderer of rendererPackage.renderers) {
    registry.register(renderer);
  }
}
