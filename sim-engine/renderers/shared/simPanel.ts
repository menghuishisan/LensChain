/**
 * simPanel.ts — SimEngine 顶层门面：编排 WS + 多场景 SceneView + 状态。
 *
 * 设计原则（不向 UI 妥协）：
 *   • API 由后端协议（06.md）驱动：sceneCode + RenderState 是核心模型。
 *   • SimPanel 不关心 UI 网格 / localStorage / 教师标注图层 —— 上层关切。
 *   • 用注册 (registerScene) 而非回调注入 (factory) 完成场景元信息接入。
 *   • 用类型化事件 (onSchemaInvalidated / onLinkTrigger / onSceneStateChange) 代替
 *     原始 WS 帧订阅 —— 不泄漏传输层抽象。
 *   • 不写兑底：未注册的场景收到 render 抛错；未连接发送 action 抛错。
 */

import { SceneView } from "./sceneView.js";
import { WSClient } from "./wsClient.js";
import { applyEnvelope, appendTimelineEvent, createRenderState, eventPayloadToTimeline } from "./stateCache.js";
import { mapInteractionDefinition } from "./interactionManager.js";
import type { ResolvedLayout } from "./layoutSolver.js";
import type {
  ActionMessagePayload,
  ControlAckPayload,
  ControlMessagePayload,
  EventMessagePayload,
  InteractionDefinition,
  JsonObject,
  LinkTrigger,
  RenderEnvelope,
  RenderState,
  SceneCategory,
  SimAction,
  SimPanelOptions,
  TimeControlCommand,
  TimeControlMode,
  WebSocketMessage,
} from "./types.js";

/** 场景元信息：在 connect 前必须 registerScene 注册。 */
export interface SceneConfig {
  sceneCode: string;
  title: string;
  category: SceneCategory;
  timeControlMode: TimeControlMode;
}

export type Unsubscribe = () => void;

export class SimPanel {
  private opts: SimPanelOptions;
  private ws: WSClient | null = null;
  private configs = new Map<string, SceneConfig>();
  private states = new Map<string, RenderState>();
  private views = new Map<string, SceneView>();

  private connectionListeners = new Set<(connected: boolean) => void>();
  private stateListeners = new Set<(sceneCode: string, state: RenderState) => void>();
  private schemaInvalidatedListeners = new Set<(sceneCode: string) => void>();
  private linkTriggerListeners = new Set<(trigger: LinkTrigger) => void>();
  private sceneEventListeners = new Set<(sceneCode: string, payload: EventMessagePayload, tick: number) => void>();
  private controlAckListeners = new Set<(ack: ControlAckPayload) => void>();

  /** event 消息计数器 (用于 TimelineEvent.id seq)。 */
  private eventSeq = 0;

  constructor(opts: SimPanelOptions) {
    this.opts = opts;
  }

  // ============================================================
  // 场景注册（必须在 connect 之前 / attachScene 之前）
  // ============================================================

  registerScene(config: SceneConfig): void {
    if (this.configs.has(config.sceneCode)) {
      throw new Error(`SimPanel.registerScene: scene "${config.sceneCode}" 已注册`);
    }
    this.configs.set(config.sceneCode, config);
    // 立即创建空 state（便于 attachScene 在 connect 前就能挂上 canvas）
    this.states.set(config.sceneCode, createRenderState(config));
  }

  unregisterScene(sceneCode: string): void {
    this.detachScene(sceneCode);
    this.configs.delete(sceneCode);
    this.states.delete(sceneCode);
  }

  // ============================================================
  // 连接生命周期
  // ============================================================

  async connect(): Promise<void> {
    if (this.ws) {
      throw new Error("SimPanel.connect: 已连接，先 disconnect()");
    }
    const ws = new WSClient({
      urlProvider: this.opts.urlProvider,
      onOpen: () => this.notifyConnection(true),
      onClose: () => this.notifyConnection(false),
      onError: (err) => { console.error("[SimPanel] WS error", err); },
      onMessage: (msg) => this.handleMessage(msg),
    });
    this.ws = ws;
    await ws.connect();
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    for (const v of this.views.values()) v.dispose();
    this.views.clear();
    this.notifyConnection(false);
  }

  isConnected(): boolean {
    return this.ws?.isConnected() ?? false;
  }

  // ============================================================
  // 订阅（类型化事件，不暴露原始 WS 帧）
  // ============================================================

  onConnectionChange(cb: (connected: boolean) => void): Unsubscribe {
    this.connectionListeners.add(cb);
    return () => { this.connectionListeners.delete(cb); };
  }

  onSceneStateChange(cb: (sceneCode: string, state: RenderState) => void): Unsubscribe {
    this.stateListeners.add(cb);
    return () => { this.stateListeners.delete(cb); };
  }

  onSchemaInvalidated(cb: (sceneCode: string) => void): Unsubscribe {
    this.schemaInvalidatedListeners.add(cb);
    return () => { this.schemaInvalidatedListeners.delete(cb); };
  }

  onLinkTrigger(cb: (trigger: LinkTrigger) => void): Unsubscribe {
    this.linkTriggerListeners.add(cb);
    return () => { this.linkTriggerListeners.delete(cb); };
  }

  onSceneEvent(cb: (sceneCode: string, payload: EventMessagePayload, tick: number) => void): Unsubscribe {
    this.sceneEventListeners.add(cb);
    return () => { this.sceneEventListeners.delete(cb); };
  }

  onControlAck(cb: (ack: ControlAckPayload) => void): Unsubscribe {
    this.controlAckListeners.add(cb);
    return () => { this.controlAckListeners.delete(cb); };
  }

  // ============================================================
  // Canvas 绑定（UI mount 后调用）
  // ============================================================

  attachScene(sceneCode: string, canvas: HTMLCanvasElement): void {
    if (!this.configs.has(sceneCode)) {
      throw new Error(`SimPanel.attachScene: scene "${sceneCode}" 未注册`);
    }
    if (this.views.has(sceneCode)) {
      throw new Error(`SimPanel.attachScene: scene "${sceneCode}" 已绑定，先 detachScene`);
    }
    const view = new SceneView({ canvas });
    this.views.set(sceneCode, view);
    const state = this.states.get(sceneCode);
    if (state && state.envelope.primitives.length > 0) {
      view.setState(state);
      view.requestRender();
    }
  }

  detachScene(sceneCode: string): void {
    const view = this.views.get(sceneCode);
    if (view) {
      view.dispose();
      this.views.delete(sceneCode);
    }
  }

  // ============================================================
  // 查询
  // ============================================================

  getSceneState(sceneCode: string): RenderState | null {
    return this.states.get(sceneCode) ?? null;
  }

  /** 输出指定场景画布的 PNG dataURL（截图）。未绑定 canvas 返回 null。 */
  captureScene(sceneCode: string): string | null {
    const view = this.views.get(sceneCode);
    return view ? view.toDataURL() : null;
  }

  /**
   * 返回指定场景最近一次解算的 ResolvedLayout。
   *
   * 供前端 React 层定位 DOM 浮层（tooltip / label / annotation / math_pipeline /
   * code_block / math_formula / register_row / error_overlay）使用：可从 positions 查
   * 到 anchor primitive 的画布像素位，再把浮层钉在旁边。
   *
   * 返回 null 表示该场景还没绑 canvas 或尚未绘制过一帧。
   */
  getResolvedLayout(sceneCode: string): ResolvedLayout | null {
    return this.views.get(sceneCode)?.getResolvedLayout() ?? null;
  }

  // ============================================================
  // 发送：Action / 时间控制
  // ============================================================

  dispatchAction(action: SimAction): void {
    this.requireConnected();
    if (!this.configs.has(action.sceneCode)) {
      throw new Error(`SimPanel.dispatchAction: scene "${action.sceneCode}" 未注册`);
    }
    const payload: ActionMessagePayload = {
      action_code: action.actionCode,
      params: action.params,
    };
    if (action.actorId ?? this.opts.actorId) payload.actor_id = action.actorId ?? this.opts.actorId!;
    if (action.userRole ?? this.opts.userRole) payload.user_role = action.userRole ?? this.opts.userRole!;
    this.ws!.send({
      type: "action",
      scene_code: action.sceneCode,
      tick: this.states.get(action.sceneCode)?.tick ?? 0,
      timestamp: Date.now(),
      payload: payload as unknown as JsonObject,
    });
  }

  /**
   * 发送协议 §7.4 的 step_back 消息。仅 single-scene process 模式场景有效。
   * 该消息类型独立于 action，不用魔术 actionCode 绕走。
   */
  sendStepBack(sceneCode: string): void {
    this.requireConnected();
    if (!this.configs.has(sceneCode)) {
      throw new Error(`SimPanel.sendStepBack: scene "${sceneCode}" 未注册`);
    }
    this.ws!.send({
      type: "step_back",
      scene_code: sceneCode,
      tick: this.states.get(sceneCode)?.tick ?? 0,
      timestamp: Date.now(),
      payload: {},
    });
  }

  sendTimeControl(command: TimeControlCommand, value?: number): void {
    this.requireConnected();
    const payload: ControlMessagePayload = { command };
    if (typeof value === "number") payload.value = value;
    this.ws!.send({
      type: "control",
      tick: 0,
      timestamp: Date.now(),
      payload: payload as unknown as JsonObject,
    });
    // 本地 SceneView 跟随：play/pause/reset 驱动 RAF 与 microStep
    if (command === "play") for (const v of this.views.values()) v.play();
    else if (command === "pause") for (const v of this.views.values()) v.pause(performance.now());
    else if (command === "reset") for (const v of this.views.values()) v.reset();
    else if (command === "set_speed" && typeof value === "number") {
      for (const v of this.views.values()) v.setSpeed(value);
    }
  }

  // ============================================================
  // 内部：WS 消息分派
  // ============================================================

  private handleMessage(msg: WebSocketMessage): void {
    switch (msg.type) {
      case "render": return this.handleRender(msg);
      case "schema_invalidated": return this.handleSchemaInvalidated(msg);
      case "event": return this.handleSceneEvent(msg);
      case "control_ack": return this.handleControlAck(msg);
      // 客户端 → 后端 (action/control/step_back) 不会出现在入站消息；走到 default 即协议违规。
      default:
        throw new Error(`SimPanel: 非法入站消息类型 "${msg.type}"`);
    }
  }

  private handleRender(msg: WebSocketMessage): void {
    const sceneCode = msg.scene_code;
    if (!sceneCode) {
      throw new Error(`SimPanel.handleRender: render 消息缺 scene_code (tick=${msg.tick})`);
    }
    const config = this.configs.get(sceneCode);
    if (!config) {
      throw new Error(`SimPanel.handleRender: 收到未注册场景 "${sceneCode}" 的 render`);
    }
    const envelope = msg.payload as unknown as RenderEnvelope;
    const prev = this.states.get(sceneCode) ?? createRenderState(config);
    const next = applyEnvelope(prev, envelope, msg.tick);
    this.states.set(sceneCode, next);
    const view = this.views.get(sceneCode);
    if (view) {
      view.setState(next);
      view.requestRender(); // rAF 节流：连续 render 消息合并为单帧
    }
    for (const cb of this.stateListeners) cb(sceneCode, next);

    // link_triggers 内嵌于 envelope；逐个派发给订阅者
    const triggers = envelope.link_triggers ?? [];
    for (const trigger of triggers) {
      for (const cb of this.linkTriggerListeners) cb(trigger);
    }
  }

  private handleSchemaInvalidated(msg: WebSocketMessage): void {
    const sceneCode = msg.scene_code;
    if (!sceneCode) {
      throw new Error("SimPanel.handleSchemaInvalidated: 缺 scene_code");
    }
    for (const cb of this.schemaInvalidatedListeners) cb(sceneCode);
  }

  private handleSceneEvent(msg: WebSocketMessage): void {
    const sceneCode = msg.scene_code;
    if (!sceneCode) {
      throw new Error("SimPanel.handleSceneEvent: 缺 scene_code");
    }
    const config = this.configs.get(sceneCode);
    if (!config) {
      throw new Error(`SimPanel.handleSceneEvent: 未注册场景 "${sceneCode}"`);
    }
    const payload = msg.payload as unknown as EventMessagePayload;
    if (typeof payload.event !== "string") {
      throw new Error(`SimPanel.handleSceneEvent: payload 缺 event 字段`);
    }
    const prev = this.states.get(sceneCode) ?? createRenderState(config);
    const timeline = eventPayloadToTimeline(payload, msg.tick, this.eventSeq++);
    const next = appendTimelineEvent(prev, timeline);
    this.states.set(sceneCode, next);
    for (const cb of this.stateListeners) cb(sceneCode, next);
    for (const cb of this.sceneEventListeners) cb(sceneCode, payload, msg.tick);
  }

  private handleControlAck(msg: WebSocketMessage): void {
    const payload = msg.payload as unknown as ControlAckPayload;
    if (typeof payload.command !== "string" || typeof payload.success !== "boolean") {
      throw new Error(`SimPanel.handleControlAck: payload 形式不符 ControlAckPayload`);
    }
    for (const cb of this.controlAckListeners) cb(payload);
  }

  private notifyConnection(connected: boolean): void {
    for (const cb of this.connectionListeners) cb(connected);
  }

  private requireConnected(): void {
    if (!this.ws || !this.ws.isConnected()) {
      throw new Error("SimPanel: WebSocket 未连接");
    }
  }
}

// ============================================================
// 工厂 + 重导出
// ============================================================

/** 默认 SimPanel 工厂。重构后无需 domain registry 注入。 */
export function createDefaultSimPanel(options: SimPanelOptions): SimPanel {
  return new SimPanel(options);
}

export { createRenderState } from "./stateCache.js";
export { mapInteractionDefinition };
export type { InteractionDefinition };
export type { ResolvedLayout, ResolvedLane, ResolvedPosition } from "./layoutSolver.js";
