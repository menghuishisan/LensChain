// teacherSummary.ts
// SimEngine 教师监控摘要解析
// 负责把后端推送的教师摘要统一转换成前端内部唯一使用的标准结构，并基于预览态重建真实缩略渲染。

import type { JsonObject, Primitive, RenderContext, RenderState, SceneCategory, TeacherSummaryPayload, WebSocketMessage } from "./types.js";
import type { DomainRenderer } from "./domainRenderer.js";
import { FallbackRenderer } from "./fallbackRenderer.js";
import { DomainRendererRegistry } from "./registry.js";
import { asArray, asNumber, asObject, asString, deepClone, isJsonObject } from "./utils.js";

/**
 * extractTeacherSummaryPayload 从 event 类型的 WebSocket 消息中提取教师监控摘要。
 * 新协议下教师摘要通过 event 消息推送，payload.event === "teacher_summary"。
 */
export function extractTeacherSummaryPayload(message: WebSocketMessage): TeacherSummaryPayload | undefined {
  if (message.type !== "event") {
    return undefined;
  }

  const payload = asOptionalObject(message.payload);
  if (!payload || asString(payload.event) !== "teacher_summary") {
    return undefined;
  }
  const data = asOptionalObject(payload.data);
  if (!data) {
    return undefined;
  }
  if (typeof data.session_id !== "string" || typeof data.instance_id !== "string") {
    return undefined;
  }

  const summary: TeacherSummaryPayload = {
    sessionId: asString(data.session_id),
    instanceId: asString(data.instance_id),
    tick: asNumber(data.tick, message.tick),
    speed: asNumber(data.speed, 1),
    activeSceneCodes: asStringArray(data.active_scene_codes),
    linkGroupCodes: asStringArray(data.link_group_codes),
    collectionRunning: Boolean(data.collection_running),
    updatedAt: normalizeTimestamp(data.updated_at)
  };
  const previewSceneCode = asOptionalString(data.preview_scene_code);
  const previewState = asOptionalObject(data.preview_state_json);
  const lastAction = asOptionalString(data.last_action);
  if (previewSceneCode !== undefined) {
    summary.previewSceneCode = previewSceneCode;
  }
  if (previewState !== undefined) {
    summary.previewState = previewState;
  }
  if (lastAction !== undefined) {
    summary.lastAction = lastAction;
  }
  return summary;
}

/**
 * buildTeacherPreviewState 从教师摘要中的 preview_state_json 重建标准渲染状态。
 */
export function buildTeacherPreviewState(summary: TeacherSummaryPayload | undefined): RenderState | undefined {
  if (!summary?.previewSceneCode || !summary.previewState) {
    return undefined;
  }

  const previewState = deepClone(summary.previewState);
  const extra = asOptionalObject(previewState.extra);
  if (!extra) {
    return undefined;
  }

  const category = asOptionalString(extra.category) as SceneCategory | undefined;
  const timeControlMode = resolveTimeControlMode(extra.time_control_mode);
  if (!category || !timeControlMode) {
    return undefined;
  }

  const primitives = asArray(previewState.primitives ?? []) as unknown as Primitive[];

  const state: RenderState = {
    sceneCode: summary.previewSceneCode,
    title: asString(extra.title, summary.previewSceneCode),
    category,
    timeControlMode,
    tick: summary.tick,
    envelope: { primitives, micro_steps: [] }
  };
  const metrics = asArray(extra.metrics);
  const tooltip = asArray(extra.tooltip);
  const timeline = asArray(extra.timeline);
  const linked = typeof extra.linked === "boolean" ? extra.linked : undefined;
  const linkGroupName = asOptionalString(extra.link_group_name);
  if (metrics.length > 0) {
    state.metrics = metrics as NonNullable<RenderState["metrics"]>;
  }
  if (tooltip.length > 0) {
    state.tooltip = tooltip as NonNullable<RenderState["tooltip"]>;
  }
  if (timeline.length > 0) {
    state.timeline = timeline as NonNullable<RenderState["timeline"]>;
  }
  if (linked !== undefined) {
    state.linked = linked;
  }
  if (linkGroupName !== undefined) {
    state.linkGroupName = linkGroupName;
  }
  return state;
}

/**
 * renderTeacherSummaryThumbnail 使用教师摘要中的预览态在前端重渲染真实缩略图。
 */
export function renderTeacherSummaryThumbnail(
  summary: TeacherSummaryPayload | undefined,
  registry: DomainRendererRegistry,
  options?: {
    width?: number;
    height?: number;
  }
): string | undefined {
  const state = buildTeacherPreviewState(summary);
  if (!state) {
    return undefined;
  }

  const width = Math.max(1, Math.round(options?.width ?? 320));
  const height = Math.max(1, Math.round(options?.height ?? 180));
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const renderer = resolveRenderer(registry, state.category);
  const theme = renderer.getTheme().theme;
  const context: RenderContext = {
    surface: { canvas },
    width,
    height,
    viewport: {
      zoom: 1,
      offsetX: 0,
      offsetY: 0
    },
    now: performance.now(),
    speed: 1,
    theme
  };
  renderer.render(state, context);
  return canvas.toDataURL("image/png");
}

function resolveRenderer(registry: DomainRendererRegistry, category: RenderState["category"]): DomainRenderer {
  if (registry.has(category)) {
    return registry.get(category);
  }
  return new FallbackRenderer(category, "当前领域未注册对应渲染器");
}

function asOptionalObject(value: unknown): JsonObject | undefined {
  return isJsonObject(value as never) ? asObject(value) : undefined;
}

function asOptionalString(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string");
}

function normalizeTimestamp(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Date.parse(value);
    if (!Number.isNaN(parsed)) {
      return parsed;
    }
  }
  return 0;
}

function resolveTimeControlMode(value: unknown): RenderState["timeControlMode"] | undefined {
  const mode = asOptionalString(value);
  if (mode === "process" || mode === "reactive" || mode === "continuous") {
    return mode;
  }
  return undefined;
}
