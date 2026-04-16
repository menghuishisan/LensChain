// teacherSummary.ts
// SimEngine 教师监控摘要解析
// 负责把后端推送的教师摘要统一转换成前端内部唯一使用的标准结构，并基于预览态重建真实缩略渲染。

import type { JsonObject, RenderContext, RenderState, TeacherSummaryPayload, WebSocketMessage } from "./types.js";
import type { DomainRenderer } from "./domainRenderer.js";
import { FallbackRenderer } from "./fallbackRenderer.js";
import { DomainRendererRegistry } from "./registry.js";
import { asArray, asNumber, asObject, asString, deepClone, isJsonObject } from "./utils.js";

/**
 * extractTeacherSummaryPayload 从 WebSocket 消息中提取并标准化教师监控摘要。
 */
export function extractTeacherSummaryPayload(message: WebSocketMessage): TeacherSummaryPayload | undefined {
  if (message.type !== "teacher_summary") {
    return undefined;
  }

  const payload = asOptionalObject(message.payload);
  if (!payload) {
    return undefined;
  }
  if (typeof payload.session_id !== "string" || typeof payload.instance_id !== "string") {
    return undefined;
  }

  const summary: TeacherSummaryPayload = {
    sessionId: asString(payload.session_id),
    instanceId: asString(payload.instance_id),
    tick: asNumber(payload.tick, message.tick),
    speed: asNumber(payload.speed, 1),
    activeSceneCodes: asStringArray(payload.active_scene_codes),
    linkGroupCodes: asStringArray(payload.link_group_codes),
    collectionRunning: Boolean(payload.collection_running),
    updatedAt: normalizeTimestamp(payload.updated_at)
  };
  const previewSceneCode = asOptionalString(payload.preview_scene_code);
  const previewState = asOptionalObject(payload.preview_state_json);
  const lastAction = asOptionalString(payload.last_action);
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

  const category = asOptionalString(extra.category);
  const algorithmType = asOptionalString(extra.algorithm_type);
  const timeControlMode = resolveTimeControlMode(extra.time_control_mode);
  if (!category || !algorithmType || !timeControlMode) {
    return undefined;
  }

  delete previewState.extra;

  const state: RenderState = {
    sceneCode: summary.previewSceneCode,
    title: asString(extra.title, summary.previewSceneCode),
    category,
    algorithmType,
    timeControlMode,
    tick: summary.tick,
    renderData: previewState
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
  const context: RenderContext = {
    surface: { canvas },
    width,
    height,
    viewport: {
      zoom: 1,
      offsetX: 0,
      offsetY: 0
    },
    now: performance.now()
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
