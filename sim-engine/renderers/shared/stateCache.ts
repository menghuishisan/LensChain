import type { JsonObject, RenderState, SceneSnapshot, TimelineEvent, WebSocketMessage } from "./types.js";
import { asArray, asObject, asString, deepClone, deepMerge, flattenChangedKeys } from "./utils.js";

/**
 * StateCache 保存场景完整状态与最多 1000 个历史快照。
 */
export class StateCache {
  private readonly states = new Map<string, RenderState>();
  private readonly history = new Map<string, SceneSnapshot[]>();

  /**
   * constructor 初始化状态缓存器。
   */
  public constructor(private readonly maxTicks = 1000) {}

  /**
   * get 返回某个场景的最新完整状态。
   */
  public get(sceneCode: string): RenderState | undefined {
    const state = this.states.get(sceneCode);
    return state ? deepClone(state) : undefined;
  }

  /**
   * list 返回全部最新场景状态。
   */
  public list(): RenderState[] {
    return Array.from(this.states.values()).map((state) => deepClone(state));
  }

  /**
   * hydrateSchema 将场景交互 schema 注入当前缓存状态。
   */
  public hydrateSchema(sceneCode: string, schema: NonNullable<RenderState["schema"]>): RenderState | undefined {
    const previous = this.states.get(sceneCode);
    if (!previous) {
      return undefined;
    }
    const nextState: RenderState = {
      ...previous,
      schema: deepClone(schema)
    };
    this.states.set(sceneCode, deepClone(nextState));
    this.pushHistory(sceneCode, nextState);
    return nextState;
  }

  /**
   * applyMessage 将数据通道消息合并到状态缓存。
   */
  public applyMessage(baseState: RenderState, message: WebSocketMessage): RenderState {
    const previous = this.states.get(baseState.sceneCode);
    let nextState = deepClone(baseState);
    const envelope = this.normalizeEnvelope(message.payload);
    if (message.type === "state_diff" && previous) {
      nextState = {
        ...previous,
        tick: message.tick,
        renderData: deepMerge(previous.renderData as JsonObject, envelope.renderData),
        changedKeys: envelope.changedKeys ?? flattenChangedKeys(envelope.renderData)
      };
      nextState = this.applyEnvelopeState(nextState, envelope);
    }
    if (message.type === "state_full") {
      nextState = {
        ...baseState,
        tick: message.tick,
        renderData: deepClone(envelope.renderData),
        changedKeys: []
      };
      nextState = this.applyEnvelopeState(nextState, envelope);
    }
    if (message.type === "event" && previous) {
      nextState = {
        ...previous,
        timeline: [...(previous.timeline ?? []), this.toTimelineEvent(message)],
        tick: message.tick
      };
    }
    if (message.type === "teacher_summary" && previous) {
      nextState = {
        ...previous,
        tick: message.tick
      };
    }
    if (message.type === "link_update" && previous) {
      nextState = {
        ...previous,
        linked: true,
        tick: message.tick,
        changedKeys: this.resolveLinkChangedKeys(message.payload)
      };
      nextState = this.applyEnvelopeState(nextState, envelope);
    }
    this.states.set(baseState.sceneCode, deepClone(nextState));
    this.pushHistory(baseState.sceneCode, nextState);
    return nextState;
  }

  /**
   * applyEnvelopeState 将标准 envelope 中的顶层元数据映射回 RenderState。
   */
  private applyEnvelopeState(
    state: RenderState,
    envelope: {
      title?: string;
      timeControlMode?: RenderState["timeControlMode"];
      linked?: boolean;
      linkGroupName?: string;
      metrics?: RenderState["metrics"];
      tooltip?: RenderState["tooltip"];
      timeline?: RenderState["timeline"];
      schema?: RenderState["schema"];
      changedKeys?: string[];
    }
  ): RenderState {
    const nextState: RenderState = {
      ...state
    };
    if (envelope.title !== undefined) {
      nextState.title = envelope.title;
    }
    if (envelope.timeControlMode !== undefined) {
      nextState.timeControlMode = envelope.timeControlMode;
    }
    if (envelope.linked !== undefined) {
      nextState.linked = envelope.linked;
    }
    if (envelope.linkGroupName !== undefined) {
      nextState.linkGroupName = envelope.linkGroupName;
    }
    if (envelope.metrics !== undefined) {
      nextState.metrics = deepClone(envelope.metrics);
    }
    if (envelope.tooltip !== undefined) {
      nextState.tooltip = deepClone(envelope.tooltip);
    }
    if (envelope.timeline !== undefined) {
      nextState.timeline = deepClone(envelope.timeline);
    }
    if (envelope.schema !== undefined) {
      nextState.schema = deepClone(envelope.schema);
    }
    if (envelope.changedKeys !== undefined) {
      nextState.changedKeys = [...envelope.changedKeys];
    }
    return nextState;
  }

  /**
   * normalizeEnvelope 将 Core 推送的完整 render envelope 提取为标准状态字段。
   */
  private normalizeEnvelope(payload: JsonObject): {
    renderData: JsonObject;
    title?: string;
    timeControlMode?: RenderState["timeControlMode"];
    linked?: boolean;
    linkGroupName?: string;
    metrics?: RenderState["metrics"];
    tooltip?: RenderState["tooltip"];
    timeline?: RenderState["timeline"];
    schema?: RenderState["schema"];
    changedKeys?: string[];
  } {
    const renderData = deepClone(payload);
    const extra = asObject(renderData.extra);
    delete renderData.extra;

    const timeControlMode = this.resolveTimeControlMode(extra.time_control_mode);
    const changedKeys = asArray<unknown>(renderData.changed_keys).map((item) => asString(item)).filter(Boolean);
    const envelope: {
      renderData: JsonObject;
      title?: string;
      timeControlMode?: RenderState["timeControlMode"];
      linked?: boolean;
      linkGroupName?: string;
      metrics?: RenderState["metrics"];
      tooltip?: RenderState["tooltip"];
      timeline?: RenderState["timeline"];
      schema?: RenderState["schema"];
      changedKeys?: string[];
    } = {
      renderData
    };

    const title = asString(extra.title);
    if (title) {
      envelope.title = title;
    }
    if (timeControlMode !== undefined) {
      envelope.timeControlMode = timeControlMode;
    }
    if (typeof extra.linked === "boolean") {
      envelope.linked = extra.linked;
    }
    const linkGroupName = asString(extra.link_group_name);
    if (linkGroupName) {
      envelope.linkGroupName = linkGroupName;
    }
    const metrics = asArray(extra.metrics) as NonNullable<RenderState["metrics"]>;
    if (metrics.length > 0) {
      envelope.metrics = metrics;
    }
    const tooltip = asArray(extra.tooltip) as NonNullable<RenderState["tooltip"]>;
    if (tooltip.length > 0) {
      envelope.tooltip = tooltip;
    }
    const timeline = asArray(extra.timeline) as NonNullable<RenderState["timeline"]>;
    if (timeline.length > 0) {
      envelope.timeline = timeline;
    }
    const schema = this.resolveSchema(extra.schema);
    if (schema !== undefined) {
      envelope.schema = schema;
    }
    if (changedKeys.length > 0) {
      envelope.changedKeys = changedKeys;
    }
    return envelope;
  }

  /**
   * resolveTimeControlMode 校验并返回允许的时间模式。
   */
  private resolveTimeControlMode(value: unknown): RenderState["timeControlMode"] | undefined {
    const mode = asString(value);
    if (mode === "process" || mode === "reactive" || mode === "continuous") {
      return mode;
    }
    return undefined;
  }

  /**
   * resolveSchema 校验并返回场景交互 schema。
   */
  private resolveSchema(value: unknown): RenderState["schema"] | undefined {
    const schema = asObject(value);
    if (typeof schema.sceneCode !== "string" || !Array.isArray(schema.actions)) {
      return undefined;
    }
    return schema as unknown as RenderState["schema"];
  }

  /**
   * rewindTo 查找指定 tick 的状态快照。
   */
  public rewindTo(sceneCode: string, targetTick: number): RenderState | undefined {
    const snapshots = this.history.get(sceneCode) ?? [];
    const hit = [...snapshots].reverse().find((snapshot) => snapshot.tick <= targetTick);
    return hit ? deepClone(hit.state) : undefined;
  }

  /**
   * toTimelineEvent 将消息转换为时间线项。
   */
  private toTimelineEvent(message: WebSocketMessage) {
    const event: TimelineEvent = {
      id: `${message.scene_code ?? "scene"}-${message.tick}-${message.timestamp}`,
      tick: message.tick,
      title: asString(message.payload.title, asString(message.payload.event, "事件")),
      description: asString(message.payload.description, JSON.stringify(message.payload))
    };
    const tone = this.resolveTimelineTone(message.payload.tone);
    if (tone !== undefined) {
      event.tone = tone;
    }
    return event;
  }

  /**
   * resolveLinkChangedKeys 读取后端在联动协议中显式声明的 changed_keys。
   */
  private resolveLinkChangedKeys(payload: JsonObject): string[] {
    return asArray<unknown>(payload.changed_keys).map((item) => asString(item)).filter(Boolean);
  }

  /**
   * resolveTimelineTone 将后端返回的事件语义收敛到前端允许的 tone 枚举。
   */
  private resolveTimelineTone(value: unknown): TimelineEvent["tone"] | undefined {
    const tone = asString(value);
    if (tone === "neutral" || tone === "info" || tone === "success" || tone === "warning" || tone === "danger") {
      return tone;
    }
    return undefined;
  }

  /**
   * pushHistory 追加单次场景快照并裁剪历史长度。
   */
  private pushHistory(sceneCode: string, state: RenderState): void {
    const snapshots = this.history.get(sceneCode) ?? [];
    snapshots.push({ tick: state.tick, state: deepClone(state) });
    while (snapshots.length > this.maxTicks) {
      snapshots.shift();
    }
    this.history.set(sceneCode, snapshots);
  }
}
