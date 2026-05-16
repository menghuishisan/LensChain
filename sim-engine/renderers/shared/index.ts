// shared/index.ts — SimEngine 渲染层桶导出。
// 重构后只保留 12 个文件，无 domain renderer / package / fallback 等过度抽象。
// 依赖顺序（自下而上）：
//   types → utils/theme → layoutSolver/primitiveDrawers
//   → primitiveRenderer → animationScheduler/microStepScheduler
//   → stateCache/interactionManager/wsClient → sceneView → simPanel.

export * from "./types.js";
export * from "./utils.js";
export * from "./theme.js";
export * from "./layoutSolver.js";
export * from "./primitiveDrawers.js";
export * from "./primitiveRenderer.js";
export * from "./animationScheduler.js";
export * from "./microStepScheduler.js";
export * from "./stateCache.js";
export * from "./interactionManager.js";
export * from "./wsClient.js";
export * from "./sceneView.js";
export * from "./simPanel.js";
