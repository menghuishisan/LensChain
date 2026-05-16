// renderers/index.ts — 对外发布的包入口（@lenschain/sim-engine-renderers）。
// 仅 re-export shared/ 桶，不再有独立 domains / createDefaultRegistry。
// 默认 SimPanel 工厂 createDefaultSimPanel 从 ./shared/simPanel.js 导出。
export * from "./shared/index.js";
