/**
 * stateCache.ts — 单场景 RenderState 的累积与查询。
 *
 * 职责：
 *   • createRenderState(input) 构造初始 RenderState（前端启动 / 切场景调用）。
 *   • applyEnvelope(state, envelope) 将后端推送的 RenderEnvelope 累积到 state 上。
 *     - 全量快照：替换 primitives / micro_steps / link_triggers。
 *     - 增量：按 changed_keys 合并；primitives 按 id 合并新增/更新。
 *   • applyMetricsAndTimeline 由 EventMessage 写侧栏数据时调用。
 *
 * 不写兑底：未声明 is_full_snapshot=true 时必须给 changed_keys，否则报错。
 */

import type {
  Annotation,
  ContainerMetric,
  EventMessagePayload,
  InteractionSchema,
  JsonValue,
  Primitive,
  RenderEnvelope,
  RenderMetric,
  RenderState,
  SceneCategory,
  TimelineEvent,
  TimeControlMode,
} from "./types.js";

export interface CreateRenderStateInput {
  sceneCode: string;
  title: string;
  category: SceneCategory;
  timeControlMode: TimeControlMode;
}

/**
 * 构造空骨架状态。
 *
 * 严格语义：tick=0，envelope.primitives=[]。
 * 首帧来自 WS 时调用方应：state = applyEnvelope(emptyState, firstEnvelope, msg.tick)。
 * 协议要求首帧必须是 is_full_snapshot=true，否则 applyEnvelope 会因缺 changed_keys 抛错。
 */
export function createRenderState(input: CreateRenderStateInput): RenderState {
  return {
    sceneCode: input.sceneCode,
    title: input.title,
    category: input.category,
    timeControlMode: input.timeControlMode,
    tick: 0,
    envelope: { primitives: [] },
    metrics: [],
    timeline: [],
    annotations: [],
    linked: false,
  };
}

/**
 * applyEnvelope 把后端推送的 envelope 合并进 state；返回新 state（不可变更新）。
 *
 * 规则：
 *   • is_full_snapshot=true → 直接替换 primitives / micro_steps / link_triggers。
 *   • is_full_snapshot=false → 按 changed_keys 合并：
 *     - "primitives" 在 changed_keys 中：用新 primitives 数组按 id 合并到旧数组（同 id 替换、新 id 追加、本帧未出现的旧 id 保留）。
 *     - "micro_steps" / "link_triggers" 同理。
 *     - "container_data" 累积追加。
 *   • changed_keys 为空且非全量 → 抛错（协议要求二选一）。
 */
export function applyEnvelope(state: RenderState, envelope: RenderEnvelope, tick: number): RenderState {
  if (envelope.is_full_snapshot) {
    const fullMerged: RenderEnvelope = {
      primitives: applyContainerMetricsToPrimitives(envelope.primitives, envelope.container_data ?? []),
      micro_steps: envelope.micro_steps ?? [],
      link_triggers: envelope.link_triggers ?? [],
      container_data: envelope.container_data ?? [],
      changed_keys: envelope.changed_keys ?? [],
      is_full_snapshot: true,
      data: envelope.data ?? {},
    };
    return {
      ...state,
      tick,
      envelope: fullMerged,
      changedKeys: envelope.changed_keys ?? [],
    };
  }

  const changedKeys = envelope.changed_keys;
  if (!changedKeys || changedKeys.length === 0) {
    throw new Error(
      `applyEnvelope: 增量推送必须填 changed_keys（is_full_snapshot=false）；场景 "${state.sceneCode}" tick=${tick}`,
    );
  }

  const merged: RenderEnvelope = { ...state.envelope };

  if (changedKeys.includes("primitives") && envelope.primitives.length > 0) {
    merged.primitives = mergeById(state.envelope.primitives, envelope.primitives);
  }
  if (changedKeys.includes("micro_steps") && envelope.micro_steps) {
    merged.micro_steps = envelope.micro_steps;
  }
  if (changedKeys.includes("link_triggers") && envelope.link_triggers) {
    merged.link_triggers = [...(state.envelope.link_triggers ?? []), ...envelope.link_triggers];
  }
  if (envelope.container_data && envelope.container_data.length > 0) {
    merged.container_data = [...(state.envelope.container_data ?? []), ...envelope.container_data];
    // 同步将容器采集指标 patch 到目标原语 params（协议 §10）
    merged.primitives = applyContainerMetricsToPrimitives(
      merged.primitives ?? state.envelope.primitives,
      envelope.container_data,
    );
  }
  merged.changed_keys = changedKeys;
  merged.is_full_snapshot = false;
  if (envelope.data) merged.data = { ...(state.envelope.data ?? {}), ...envelope.data };

  return { ...state, tick, envelope: merged, changedKeys };
}

/**
 * 将 ContainerMetric 会制到原语 params：按 (target_primitive, target_param) 定位并赋值。
 *
 * 不带目标的条目留给侧栏指标卡使用，不改 primitives。
 * 指向不存在的 primitive 会抛错——场景表明宣告了采集变量却未输出对应原语是协议 bug。
 */
function applyContainerMetricsToPrimitives(
  primitives: readonly Primitive[],
  metrics: readonly ContainerMetric[],
): Primitive[] {
  const targeted = metrics.filter(m => m.target_primitive && m.target_param);
  if (targeted.length === 0) return [...primitives];
  // 同 id 多条取最新（在本批内后到为最新）
  const byTarget = new Map<string, ContainerMetric>();
  for (const m of targeted) byTarget.set(`${m.target_primitive}::${m.target_param}`, m);
  const ids = new Set(targeted.map(m => m.target_primitive!));
  return primitives.map(p => {
    if (!ids.has(p.id)) return p;
    const nextParams = { ...p.params };
    for (const m of targeted) {
      if (m.target_primitive !== p.id) continue;
      nextParams[m.target_param!] = m.value as JsonValue;
    }
    return { ...p, params: nextParams };
  });
  void byTarget;
}

/**
 * 将 event 消息转为 TimelineEvent，按约定映射：
 *   title = payload.event
 *   description = payload.data.description (if string)
 *   tone = payload.data.tone (if matches)
 *   id = `evt-${tick}-${event}-${seq}`
 */
export function eventPayloadToTimeline(
  payload: EventMessagePayload,
  tick: number,
  seq: number,
): TimelineEvent {
  const desc = payload.data?.description;
  const tone = payload.data?.tone;
  const evt: TimelineEvent = {
    id: `evt-${tick}-${payload.event}-${seq}`,
    tick,
    title: payload.event,
  };
  if (typeof desc === "string") evt.description = desc;
  if (typeof tone === "string" && (
    tone === "neutral" || tone === "info" || tone === "success" || tone === "warning" || tone === "danger"
  )) {
    evt.tone = tone;
  }
  return evt;
}

/** 按 id 合并 primitives：新数组覆盖同 id，未出现的旧 id 保留。 */
function mergeById<T extends { id: string }>(oldItems: readonly T[], newItems: readonly T[]): T[] {
  const map = new Map<string, T>();
  for (const it of oldItems) map.set(it.id, it);
  for (const it of newItems) map.set(it.id, it);
  return Array.from(map.values());
}

export function applyMetrics(state: RenderState, metrics: readonly RenderMetric[]): RenderState {
  return { ...state, metrics: [...metrics] };
}

export function appendTimelineEvent(state: RenderState, event: TimelineEvent): RenderState {
  return { ...state, timeline: [...(state.timeline ?? []), event] };
}

export function appendAnnotation(state: RenderState, annotation: Annotation): RenderState {
  return { ...state, annotations: [...(state.annotations ?? []), annotation] };
}

export function removeAnnotation(state: RenderState, id: string): RenderState {
  return { ...state, annotations: (state.annotations ?? []).filter(a => a.id !== id) };
}

export function setSchema(state: RenderState, schema: InteractionSchema): RenderState {
  return { ...state, schema };
}

export function setLinkInfo(state: RenderState, linked: boolean, linkGroupName?: string): RenderState {
  const next: RenderState = { ...state, linked };
  if (linkGroupName !== undefined) next.linkGroupName = linkGroupName;
  return next;
}
