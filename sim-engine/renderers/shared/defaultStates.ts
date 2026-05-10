import type { InteractionSchema, RenderEnvelope, RenderState, SceneCategory, TimeControlMode } from "./types.js";

/**
 * createRenderState 创建标准渲染状态对象（新协议：envelope-based）。
 */
export function createRenderState(input: {
  sceneCode: string;
  title: string;
  category: SceneCategory;
  timeControlMode: TimeControlMode;
  tick?: number;
  totalTicks?: number;
  linked?: boolean;
  linkGroupName?: string;
  envelope?: RenderEnvelope;
  metrics?: RenderState["metrics"];
  tooltip?: RenderState["tooltip"];
  timeline?: RenderState["timeline"];
  changedKeys?: string[];
  schema?: InteractionSchema;
}): RenderState {
  const state: RenderState = {
    sceneCode: input.sceneCode,
    title: input.title,
    category: input.category,
    timeControlMode: input.timeControlMode,
    tick: input.tick ?? 0,
    envelope: input.envelope ?? { primitives: [], micro_steps: [] },
    metrics: input.metrics ?? [],
    tooltip: input.tooltip ?? [],
    timeline: input.timeline ?? [],
    changedKeys: input.changedKeys ?? []
  };
  if (input.totalTicks !== undefined) {
    state.totalTicks = input.totalTicks;
  }
  if (input.linked !== undefined) {
    state.linked = input.linked;
  }
  if (input.linkGroupName !== undefined) {
    state.linkGroupName = input.linkGroupName;
  }
  if (input.schema !== undefined) {
    state.schema = input.schema;
  }
  return state;
}
