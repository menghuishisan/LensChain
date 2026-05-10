// createDefaultRegistry.ts
// SimEngine 默认渲染器注册表装配
// 负责装配平台内置 8 个领域渲染器，并在同一个注册表上叠加自定义渲染器包。

import {
  AttackSecurityRenderer,
  ConsensusRenderer,
  CryptographyRenderer,
  DataStructureRenderer,
  EconomicRenderer,
  GenericPrimitiveRenderer,
  NodeNetworkRenderer,
  SmartContractRenderer,
  TransactionRenderer
} from "./domains.js";
import { DomainRendererRegistry, registerRendererPackage, type DomainRenderer, type RendererPackage } from "./shared/index.js";

/**
 * CreateDefaultRegistryOptions 定义默认注册表可选的扩展输入。
 */
export interface CreateDefaultRegistryOptions {
  renderers?: Iterable<DomainRenderer>;
  packages?: Iterable<RendererPackage>;
}

/**
 * createDefaultRegistry 创建包含 8 个领域渲染器的默认注册中心。
 */
export function createDefaultRegistry(options: CreateDefaultRegistryOptions = {}): DomainRendererRegistry {
  const registry = new DomainRendererRegistry();
  registry.register(new NodeNetworkRenderer());
  registry.register(new ConsensusRenderer());
  registry.register(new CryptographyRenderer());
  registry.register(new DataStructureRenderer());
  registry.register(new TransactionRenderer());
  registry.register(new SmartContractRenderer());
  registry.register(new AttackSecurityRenderer());
  registry.register(new EconomicRenderer());
  registry.register(new GenericPrimitiveRenderer());

  for (const renderer of options.renderers ?? []) {
    registry.register(renderer);
  }
  for (const rendererPackage of options.packages ?? []) {
    registerRendererPackage(registry, rendererPackage);
  }
  return registry;
}
