import { createDefaultRegistry } from "./createDefaultRegistry.js";
import { SimPanel, type SimPanelOptions } from "./shared/index.js";

/**
 * createDefaultSimPanel 创建并装配默认 8 个领域渲染器的仿真面板。
 */
export function createDefaultSimPanel(options: SimPanelOptions): SimPanel {
  const panel = new SimPanel(options);
  panel.registerAll(createDefaultRegistry().list());
  return panel;
}
