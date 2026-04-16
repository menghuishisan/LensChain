// createDefaultRegistry.ts
// SimEngine 默认渲染器注册表装配
// 负责装配平台内置 8 个领域渲染器，并在同一个注册表上叠加自定义渲染器包。

import { AttackSecurityRenderer } from "./attack-security/index.js";
import { ConsensusRenderer } from "./consensus/index.js";
import { CryptographyRenderer } from "./cryptography/index.js";
import { DataStructureRenderer } from "./data-structure/index.js";
import { EconomicRenderer } from "./economic/index.js";
import { NodeNetworkRenderer } from "./node-network/index.js";
import { SmartContractRenderer } from "./smart-contract/index.js";
import { DomainRendererRegistry, registerRendererPackage, type DomainRenderer, type RendererPackage } from "./shared/index.js";
import { TransactionRenderer } from "./transaction/index.js";

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

  for (const renderer of options.renderers ?? []) {
    registry.register(renderer);
  }
  for (const rendererPackage of options.packages ?? []) {
    registerRendererPackage(registry, rendererPackage);
  }
  return registry;
}
