// simPanel.ts 负责多场景仿真面板的数据通道、布局与交互协调。
import type { DomainRenderer } from "./domainRenderer.js";
import type {
  InteractionAction,
  InteractionInputMap,
  InteractionSchema,
  PanelLayoutItem,
  RenderState,
  SceneSummary,
  SimAction,
  SimPanelOptions,
  TeacherSummaryPayload,
  WebSocketMessage
} from "./types.js";
import { InteractionManager } from "./interactionManager.js";
import { InteractionPanelModel } from "./interactionPanel.js";
import { LayoutController } from "./layoutController.js";
import { PanelLayoutStore } from "./panelLayout.js";
import { DomainRendererRegistry } from "./registry.js";
import { StateCache } from "./stateCache.js";
import { extractTeacherSummaryPayload } from "./teacherSummary.js";
import { TimelineControl } from "./timelineControl.js";
import { WSClient } from "./wsClient.js";
import { SceneView } from "./sceneView.js";
import { FallbackRenderer } from "./fallbackRenderer.js";

/**
 * SimPanel 管理仿真面板内的多场景视图、布局和数据通道。
 */
export class SimPanel {
  private static readonly MAX_SCENES_PER_PAGE = 6;

  private readonly registry = new DomainRendererRegistry();
  private readonly stateCache = new StateCache();
  private readonly messageListeners = new Set<(message: WebSocketMessage) => void>();
  private readonly views = new Map<string, SceneView>();
  private readonly wsClient: WSClient;
  private readonly interactionManager = new InteractionManager();
  private readonly interactionPanel = new InteractionPanelModel();
  private readonly timelineControl = new TimelineControl();
  private readonly layoutStore: PanelLayoutStore;
  private readonly layoutController: LayoutController;
  private unsubscribeWS?: () => void;
  private readonly sceneUnsubscribes = new Map<string, () => void>();
  private scenePage = 1;

  /**
   * constructor 初始化仿真面板控制器。
   */
  public constructor(private readonly options: SimPanelOptions) {
    this.wsClient = new WSClient(
      `${options.endpoint.replace(/\/$/, "")}/${options.sessionId}?token=${options.token}`
    );
    this.layoutStore = new PanelLayoutStore(
      options.layoutStorageKey ?? `sim-engine-layout:${options.sessionId}`
    );
    this.layoutController = new LayoutController(
      this.mergeInitialLayout(options.initialLayout ?? [], this.layoutStore.load())
    );
  }

  /**
   * registerRenderer 注册一个领域渲染器。
   */
  public registerRenderer(renderer: DomainRenderer): void {
    this.registry.register(renderer);
  }

  /**
   * registerAll 批量注册多个领域渲染器。
   */
  public registerAll(renderers: Iterable<DomainRenderer>): void {
    for (const renderer of renderers) {
      this.registerRenderer(renderer);
    }
  }

  /**
   * attachScene 将场景画布挂到面板控制器中。
   */
  public attachScene(
    sceneCode: string,
    category: RenderState["category"],
    canvas: HTMLCanvasElement,
    overlay?: HTMLElement
  ): void {
    this.detachScene(sceneCode);
    const layout = this.findLayout(sceneCode);
    const renderer = this.registry.has(category)
      ? this.registry.get(category)
      : new FallbackRenderer(category, "当前领域未注册对应渲染器");
    const surface = overlay === undefined ? { canvas } : { canvas, overlay };
    const view = new SceneView(surface, renderer, layout);
    const unsubscribe = this.interactionManager.bind(
      canvas,
      renderer,
      () => {
        const state = this.stateCache.get(sceneCode);
        if (!state) {
          throw new Error(`场景状态不存在: ${sceneCode}`);
        }
        return state;
      },
      (action) => this.sendAction(action)
    );
    this.views.set(sceneCode, view);
    this.sceneUnsubscribes.set(sceneCode, unsubscribe);
    const state = this.stateCache.get(sceneCode);
    if (state) {
      view.render(state);
    }
  }

  /**
   * detachScene 卸载单个场景视图和交互监听。
   */
  public detachScene(sceneCode: string): void {
    this.sceneUnsubscribes.get(sceneCode)?.();
    this.sceneUnsubscribes.delete(sceneCode);
    this.views.delete(sceneCode);
  }

  /**
   * connect 建立与 Core 的 WebSocket 数据通道。
   */
  public connect(stateResolver: (message: WebSocketMessage) => RenderState): void {
    this.unsubscribeWS = this.wsClient.subscribe((message) => {
      for (const listener of this.messageListeners) {
        listener(message);
      }
      try {
        const baseState = stateResolver(message);
        const state = this.stateCache.applyMessage(baseState, message);
        const view = this.views.get(state.sceneCode);
        view?.setState(state);
      } catch {
        if (message.scene_code) {
          const state = this.stateCache.get(message.scene_code);
          if (state) {
            this.views.get(message.scene_code)?.render(state);
          }
        }
      }
    });
    this.wsClient.connect();
  }

  /**
   * subscribeMessages 订阅来自 Core 的原始协议消息。
   */
  public subscribeMessages(listener: (message: WebSocketMessage) => void): () => void {
    this.messageListeners.add(listener);
    return () => {
      this.messageListeners.delete(listener);
    };
  }

  /**
   * subscribeConnectionStatus 订阅数据通道连接状态变化。
   */
  public subscribeConnectionStatus(listener: (connected: boolean) => void): () => void {
    return this.wsClient.subscribeStatus(listener);
  }

  /**
   * disconnect 关闭面板数据通道。
   */
  public disconnect(): void {
    this.unsubscribeWS?.();
    delete this.unsubscribeWS;
    this.wsClient.disconnect();
  }

  /**
   * setState 手动写入一个场景状态。
   */
  public setState(state: RenderState): void {
    this.stateCache.applyMessage(state, {
      type: "state_full",
      scene_code: state.sceneCode,
      tick: state.tick,
      timestamp: Date.now(),
      payload: state.renderData
    });
    this.views.get(state.sceneCode)?.setState(state);
  }

  /**
   * saveLayout 保存当前布局配置。
   */
  public saveLayout(layout: PanelLayoutItem[]): void {
    layout.forEach((item) => this.layoutController.upsert(item));
    this.layoutStore.save(this.layoutController.list());
  }

  /**
   * moveScene 调整场景面板位置并持久化。
   */
  public moveScene(sceneCode: string, x: number, y: number): void {
    this.layoutController.move(sceneCode, x, y);
    this.layoutStore.save(this.layoutController.list());
  }

  /**
   * resizeScene 调整场景面板尺寸并持久化。
   */
  public resizeScene(sceneCode: string, w: number, h: number): void {
    this.layoutController.resize(sceneCode, w, h);
    this.layoutStore.save(this.layoutController.list());
  }

  /**
   * toggleSceneFullscreen 切换单个场景的全屏状态。
   */
  public toggleSceneFullscreen(sceneCode: string): void {
    this.layoutController.toggleFullscreen(sceneCode);
    this.layoutStore.save(this.layoutController.list());
  }

  /**
   * annotateScene 为指定场景增加文本标注。
   */
  public annotateScene(sceneCode: string, text: string, x: number, y: number): void {
    const view = this.views.get(sceneCode);
    view?.addAnnotation(text, x, y);
  }

  /**
   * removeAnnotation 删除指定场景中的单个标注。
   */
  public removeAnnotation(sceneCode: string, annotationID: string): void {
    this.views.get(sceneCode)?.removeAnnotation(annotationID);
  }

  /**
   * clearAnnotations 清空指定场景中的全部标注。
   */
  public clearAnnotations(sceneCode: string): void {
    this.views.get(sceneCode)?.clearAnnotations();
  }

  /**
   * captureScene 导出指定场景截图。
   */
  public captureScene(sceneCode: string): string {
    const view = this.views.get(sceneCode);
    if (!view) {
      throw new Error(`场景视图不存在: ${sceneCode}`);
    }
    return view.capture();
  }

  /**
   * recordScene 启动指定场景录制。
   */
  public recordScene(sceneCode: string): MediaRecorder {
    const view = this.views.get(sceneCode);
    if (!view) {
      throw new Error(`场景视图不存在: ${sceneCode}`);
    }
    return view.record();
  }

  /**
   * getInteractionSchema 返回场景当前交互 schema。
   */
  public getInteractionSchema(sceneCode: string): InteractionSchema | undefined {
    return this.stateCache.get(sceneCode)?.schema;
  }

  /**
   * setInteractionSchema 将控制面返回的场景交互 schema 注入缓存并刷新视图。
   */
  public setInteractionSchema(sceneCode: string, schema: InteractionSchema): void {
    const state = this.stateCache.hydrateSchema(sceneCode, schema);
    if (state) {
      this.views.get(sceneCode)?.setState(state);
    }
  }

  /**
   * listInteractionActions 返回场景的动态交互动作定义。
   */
  public listInteractionActions(sceneCode: string): InteractionAction[] {
    return this.interactionPanel.listActions(this.getInteractionSchema(sceneCode));
  }

  /**
   * createInteractionDefaults 生成某个动作的默认表单值。
   */
  public createInteractionDefaults(sceneCode: string, actionCode: string): InteractionInputMap {
    const action = this.findInteractionAction(sceneCode, actionCode);
    return this.interactionPanel.createDefaults(action);
  }

  /**
   * submitInteraction 根据 schema 动态构造并发送操作请求。
   */
  public submitInteraction(sceneCode: string, actionCode: string, input: InteractionInputMap): void {
    const action = this.findInteractionAction(sceneCode, actionCode);
    this.sendAction(this.interactionPanel.buildAction(sceneCode, action, input));
  }

  /**
   * getTimeControlMode 返回场景当前时间控制模式。
   */
  public getTimeControlMode(sceneCode: string): RenderState["timeControlMode"] | undefined {
    return this.stateCache.get(sceneCode)?.timeControlMode;
  }

  /**
   * getAvailableControls 返回场景当前可展示的时间控制项。
   */
  public getAvailableControls(sceneCode: string) {
    const mode = this.getTimeControlMode(sceneCode);
    return mode ? this.timelineControl.getControls(mode) : [];
  }

  /**
   * getSceneState 返回当前场景状态快照。
   */
  public getSceneState(sceneCode: string): RenderState | undefined {
    return this.views.get(sceneCode)?.getState() ?? this.stateCache.get(sceneCode);
  }

  /**
   * captureSceneThumbnail 导出教师监控缩略图。
   */
  public captureSceneThumbnail(sceneCode: string, maxWidth = 320, maxHeight = 180): string {
    const view = this.views.get(sceneCode);
    if (!view) {
      throw new Error(`场景视图不存在: ${sceneCode}`);
    }
    return view.captureThumbnail(maxWidth, maxHeight);
  }

  /**
   * getTeacherSummary 从摘要消息中读取教师监控载荷。
   */
  public getTeacherSummary(message: WebSocketMessage): TeacherSummaryPayload | undefined {
    return extractTeacherSummaryPayload(message);
  }

  public listSceneSummaries(includeThumbnail = false): SceneSummary[] {
    return this.stateCache.list().map((state) => {
      const summary: SceneSummary = {
        sceneCode: state.sceneCode,
        title: state.title,
        algorithmType: state.algorithmType,
        timeControlMode: state.timeControlMode,
        tick: state.tick,
        linked: state.linked ?? false,
        metricCount: state.metrics?.length ?? 0,
        eventCount: state.timeline?.length ?? 0,
        changedKeyCount: state.changedKeys?.length ?? 0
      };
      if (state.linkGroupName !== undefined) {
        summary.linkGroupName = state.linkGroupName;
      }
      if (includeThumbnail && this.views.has(state.sceneCode)) {
        summary.thumbnail = this.captureSceneThumbnail(state.sceneCode);
      }
      return summary;
    });
  }

  /**
   * setScenePage 设置超过 6 个场景时的分页页码。
   */
  public setScenePage(page: number): void {
    this.scenePage = Math.max(1, Math.floor(page));
  }

  /**
   * getVisibleSceneCodes 返回当前页应展示的场景编码。
   */
  public getVisibleSceneCodes(): string[] {
    const sceneCodes = this.stateCache.list().map((state) => state.sceneCode);
    const start = (this.scenePage - 1) * SimPanel.MAX_SCENES_PER_PAGE;
    return sceneCodes.slice(start, start + SimPanel.MAX_SCENES_PER_PAGE);
  }

  /**
   * getScenePageCount 返回当前状态下的场景分页总数。
   */
  public getScenePageCount(): number {
    return Math.max(1, Math.ceil(this.stateCache.list().length / SimPanel.MAX_SCENES_PER_PAGE));
  }

  /**
   * getLayout 返回当前面板布局快照。
   */
  public getLayout(): PanelLayoutItem[] {
    return this.layoutController.list();
  }

  /**
   * sendAction 发送场景交互。
   */
  public sendAction(action: SimAction): void {
    const payload: SimAction = { ...action };
    if (this.options.actorId !== undefined) {
      payload.actorId = this.options.actorId;
    }
    if (this.options.roleKey !== undefined) {
      payload.roleKey = this.options.roleKey;
    }
    this.wsClient.sendAction(payload);
  }

  /**
   * sendControl 发送时间控制指令。
   */
  public sendControl(command: Parameters<WSClient["sendControl"]>[0], value?: number): void {
    this.wsClient.sendControl(command, value);
  }

  /**
   * rewindTo 发送指定 tick 回退指令。
   */
  public rewindTo(targetTick: number): void {
    this.wsClient.sendRewind(targetTick);
  }

  /**
   * findLayout 读取指定场景的布局，如未配置则给出默认值。
   */
  private findLayout(sceneCode: string): PanelLayoutItem {
    const hit = this.layoutController.list().find((item) => item.sceneCode === sceneCode);
    return hit ?? { sceneCode, x: 0, y: 0, w: 6, h: 4 };
  }

  /**
   * mergeInitialLayout 合并模板初始布局和用户本地持久化布局。
   */
  private mergeInitialLayout(initialLayout: PanelLayoutItem[], storedLayout: PanelLayoutItem[]): PanelLayoutItem[] {
    const merged = new Map<string, PanelLayoutItem>();
    initialLayout.forEach((item) => merged.set(item.sceneCode, { ...item }));
    storedLayout.forEach((item) => merged.set(item.sceneCode, { ...item }));
    return Array.from(merged.values());
  }

  /**
   * findInteractionAction 按动作编码读取场景交互定义。
   */
  private findInteractionAction(sceneCode: string, actionCode: string): InteractionAction {
    const action = this.listInteractionActions(sceneCode).find((item) => item.actionCode === actionCode);
    if (!action) {
      throw new Error(`场景未声明交互动作: ${sceneCode}/${actionCode}`);
    }
    return action;
  }
}
