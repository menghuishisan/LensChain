import type { JsonObject, MicroStep, Primitive, RenderState, SceneSnapshot, TimelineEvent, WebSocketMessage } from "./types.js";
import { asArray, asObject, asString, deepClone } from "./utils.js";

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
   * 新协议仅处理 "render" 和 "event" 两种服务端推送。
   */
  public applyMessage(baseState: RenderState, message: WebSocketMessage): RenderState {
    const previous = this.states.get(baseState.sceneCode);
    let nextState = deepClone(baseState);

    if (message.type === "render") {
      const meta = this.normalizeEnvelope(message.payload);
      nextState = {
        ...(previous ?? baseState),
        tick: message.tick,
        envelope: {
          primitives: (asArray(message.payload.primitives) ?? []) as Primitive[],
          micro_steps: (asArray(message.payload.micro_steps) ?? []) as MicroStep[]
        },
        changedKeys: meta.changedKeys ?? []
      };
      nextState = this.applyEnvelopeState(nextState, meta);
    } else if (message.type === "event" && previous) {
      nextState = {
        ...previous,
        timeline: [...(previous.timeline ?? []), this.toTimelineEvent(message)],
        tick: message.tick
      };
    } else if (previous) {
      nextState = {
        ...previous,
        tick: message.tick
      };
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
   * normalizeEnvelope 从 render 载荷中提取元数据字段。
   */
  private normalizeEnvelope(payload: JsonObject): {
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
    const meta = asObject(payload.meta);
    const timeControlMode = this.resolveTimeControlMode(meta.time_control_mode);
    const changedKeys = asArray<unknown>(payload.changed_keys).map((item) => asString(item)).filter(Boolean);
    const result: {
      title?: string;
      timeControlMode?: RenderState["timeControlMode"];
      linked?: boolean;
      linkGroupName?: string;
      metrics?: RenderState["metrics"];
      tooltip?: RenderState["tooltip"];
      timeline?: RenderState["timeline"];
      schema?: RenderState["schema"];
      changedKeys?: string[];
    } = {};

    const title = asString(meta.title);
    if (title) {
      result.title = title;
    }
    if (timeControlMode !== undefined) {
      result.timeControlMode = timeControlMode;
    }
    if (typeof meta.linked === "boolean") {
      result.linked = meta.linked;
    }
    const linkGroupName = asString(meta.link_group_name);
    if (linkGroupName) {
      result.linkGroupName = linkGroupName;
    }
    const metrics = asArray(meta.metrics) as NonNullable<RenderState["metrics"]>;
    if (metrics.length > 0) {
      result.metrics = metrics;
    }
    const tooltip = asArray(meta.tooltip) as NonNullable<RenderState["tooltip"]>;
    if (tooltip.length > 0) {
      result.tooltip = tooltip;
    }
    const timeline = asArray(meta.timeline) as NonNullable<RenderState["timeline"]>;
    if (timeline.length > 0) {
      result.timeline = timeline;
    }
    const schema = this.resolveSchema(meta.schema);
    if (schema !== undefined) {
      result.schema = schema;
    }
    if (changedKeys.length > 0) {
      result.changedKeys = changedKeys;
    }
    return result;
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
