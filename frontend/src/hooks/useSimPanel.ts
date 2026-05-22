/**
 * useSimPanel.ts — SimPanel 渲染编排 hook（06.2 §一-§六）。
 *
 * 职责：
 *   • 创建并连接 SimPanel；
 *   • 注册 scenes、订阅事件，把后端 RenderState / TimelineEvent / LinkTrigger
 *     映射成 React state 给上层组件消费；
 *   • 暴露 attach/detach/dispatch 给画布与控件层。
 *
 * 注意：
 *   • panel 用 useState 而非 useRef——子组件 SceneCanvas 的 effect 早于父
 *     useEffect，必须靠 panel 状态变化触发 attach 重跑（参考前次实现的踩坑笔记）。
 *   • urlProvider 走 apiClient.ensureFreshAccessToken：每次重连自带 refresh，
 *     避免 access token 过期 → 401 死循环。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  createDefaultSimPanel,
  type ControlAckPayload,
  type EventMessagePayload,
  type JsonObject as EngineJsonObject,
  type LinkTrigger,
  type RenderState,
  type ResolvedLayout,
  type SceneCategory,
  type SimAction,
  type SimPanel,
  type TimeControlCommand,
  type TimeControlMode,
} from '@lenschain/sim-engine-renderers';
import { useAuthStore } from '@/stores/authStore';
import { apiClient } from '@/lib/api-client';
import type { SimTimeControlMode } from '@/types/experiment';

/** 单个场景配置：上层把模板里的 sim_scenes 映射成此结构传入。 */
export interface SimSceneConfig {
  sceneCode: string;
  title: string;
  /** 06.md 9 大领域 + generic 教师扩展。 */
  category: SceneCategory;
  /** SimEngine 时间控制模式（snake_case 与 camelCase 等价）。 */
  timeControlMode: SimTimeControlMode;
}

/** useSimPanel 输入。 */
export interface UseSimPanelOptions {
  /** SimEngine WS 会话 ID（来自 instance.sim_session_id）。 */
  sessionId: string;
  /** 场景列表；元素顺序即默认主区排列顺序。 */
  scenes: readonly SimSceneConfig[];
  /** 用户角色，传给协议 actor_id / user_role。 */
  userRole?: 'student' | 'teacher';
  /** 教师监控视角传 actor_id（学生侧无需）。 */
  actorId?: string;
}

/** useSimPanel 返回。 */
export interface UseSimPanelReturn {
  panel: SimPanel | null;
  connected: boolean;
  /** 各场景最新 RenderState；key=sceneCode。 */
  sceneStates: ReadonlyMap<string, RenderState>;
  /** 最近一次 control_ack（供 ControlBar 显示等待态 / 错误回滚）。 */
  lastControlAck: ControlAckPayload | null;
  /** 累积 link triggers（cross-canvas-overlay 消费）。 */
  linkTriggers: readonly LinkTrigger[];
  /** 累积 scene events（侧栏 timeline / 教师监控可消费）。 */
  sceneEvents: readonly { sceneCode: string; payload: EventMessagePayload; tick: number }[];
  /** 累积 session-scoped events（envelope 不带 scene_code 的 `event` 消息，
   * teacher_broadcast / link_update / snapshot_* / scene_runtime_failure 等）。
   * 供 SimEnginePanel 跳 toast / sidebar 通知区 / 刷 SharedState 使用。 */
  sessionEvents: readonly { payload: EventMessagePayload; tick: number }[];

  // ── canvas 绑定（子组件 mount 时调用） ──
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;

  /** 查询某场景最近一次 layoutSolver 解算结果（用于跨画布弧线 anchor 定位）。 */
  getResolvedLayout: (sceneCode: string) => ResolvedLayout | null;

  // ── 操作 ──
  dispatchAction: (action: SimAction) => void;
  sendTimeControl: (command: TimeControlCommand, value?: number) => void;
  sendStepBack: (sceneCode: string) => void;
  captureScene: (sceneCode: string) => string | null;
}

/** SimSceneConfig.timeControlMode → SimEngine TimeControlMode（同义重命名）。 */
function toEngineTimeControlMode(m: SimTimeControlMode): TimeControlMode {
  return m;
}

export function useSimPanel(options: UseSimPanelOptions): UseSimPanelReturn {
  const { sessionId, scenes, userRole, actorId } = options;
  const accessToken = useAuthStore(s => s.accessToken);

  const [panel, setPanel] = useState<SimPanel | null>(null);
  const [connected, setConnected] = useState(false);
  const [sceneStates, setSceneStates] = useState<Map<string, RenderState>>(new Map());
  const [lastControlAck, setLastControlAck] = useState<ControlAckPayload | null>(null);
  const [linkTriggers, setLinkTriggers] = useState<LinkTrigger[]>([]);
  const [sceneEvents, setSceneEvents] = useState<{ sceneCode: string; payload: EventMessagePayload; tick: number }[]>([]);
  const [sessionEvents, setSessionEvents] = useState<{ payload: EventMessagePayload; tick: number }[]>([]);

  // 跟踪场景配置，避免 effect 因数组身份频繁变化重连
  const scenesRef = useRef(scenes);
  useEffect(() => { scenesRef.current = scenes; }, [scenes]);

  useEffect(() => {
    if (!sessionId) return;
    if (!accessToken) return; // 仅作为"已登录"门闸，token 实际值由 urlProvider 现取

    const rawBase = process.env.NEXT_PUBLIC_WS_BASE_URL ?? '';
    if (rawBase.length === 0) return;
    const wsBase = rawBase
      .replace(/^https?:/, (proto) => (proto === 'https:' ? 'wss:' : 'ws:'))
      .replace(/\/$/, '');
    const endpoint = `${wsBase}/ws/sim-engine`;

    const urlProvider = async (): Promise<string> => {
      const token = await apiClient.ensureFreshAccessToken();
      if (!token) return '';
      return `${endpoint}/${sessionId}?token=${token}`;
    };

    const instance = createDefaultSimPanel({
      sessionId,
      urlProvider,
      ...(actorId ? { actorId } : {}),
      ...(userRole ? { userRole } : {}),
    });

    // 注册全部场景
    for (const cfg of scenesRef.current) {
      instance.registerScene({
        sceneCode: cfg.sceneCode,
        title: cfg.title,
        category: cfg.category,
        timeControlMode: toEngineTimeControlMode(cfg.timeControlMode),
      });
    }

    setPanel(instance);

    const offConn = instance.onConnectionChange(setConnected);
    const offState = instance.onSceneStateChange((sceneCode, state) => {
      setSceneStates(prev => {
        const next = new Map(prev);
        next.set(sceneCode, state);
        return next;
      });
    });
    const offLink = instance.onLinkTrigger((trigger) => {
      setLinkTriggers(prev => [...prev, trigger]);
    });
    const offEvent = instance.onSceneEvent((sceneCode, payload, tick) => {
      setSceneEvents(prev => [...prev, { sceneCode, payload, tick }]);
    });
    const offSession = instance.onSessionEvent((payload, tick) => {
      setSessionEvents(prev => [...prev, { payload, tick }]);
    });
    const offAck = instance.onControlAck((ack) => {
      setLastControlAck(ack);
    });

    void instance.connect();

    return () => {
      offConn();
      offState();
      offLink();
      offEvent();
      offSession();
      offAck();
      instance.disconnect();
      setPanel(null);
      setConnected(false);
      setSceneStates(new Map());
      setLastControlAck(null);
      setLinkTriggers([]);
      setSceneEvents([]);
      setSessionEvents([]);
    };
  }, [sessionId, accessToken, userRole, actorId]);

  const attachScene = useCallback((sceneCode: string, canvas: HTMLCanvasElement) => {
    panel?.attachScene(sceneCode, canvas);
  }, [panel]);

  const detachScene = useCallback((sceneCode: string) => {
    panel?.detachScene(sceneCode);
  }, [panel]);

  const getResolvedLayout = useCallback((sceneCode: string) => {
    return panel?.getResolvedLayout(sceneCode) ?? null;
  }, [panel]);

  const dispatchAction = useCallback((action: SimAction) => {
    panel?.dispatchAction(action);
  }, [panel]);

  const sendTimeControl = useCallback((command: TimeControlCommand, value?: number) => {
    panel?.sendTimeControl(command, value);
  }, [panel]);

  const sendStepBack = useCallback((sceneCode: string) => {
    panel?.sendStepBack(sceneCode);
  }, [panel]);

  const captureScene = useCallback((sceneCode: string): string | null => {
    return panel?.captureScene(sceneCode) ?? null;
  }, [panel]);

  return useMemo<UseSimPanelReturn>(() => ({
    panel,
    connected,
    sceneStates,
    lastControlAck,
    linkTriggers,
    sceneEvents,
    sessionEvents,
    attachScene,
    detachScene,
    getResolvedLayout,
    dispatchAction,
    sendTimeControl,
    sendStepBack,
    captureScene,
  }), [
    panel, connected, sceneStates, lastControlAck, linkTriggers, sceneEvents, sessionEvents,
    attachScene, detachScene, getResolvedLayout, dispatchAction, sendTimeControl, sendStepBack, captureScene,
  ]);
}

/** 便利：构造 immediate / submit ActionDef 的 SimAction（用于 InteractionForm 提交）。 */
export function buildSimAction(input: {
  sceneCode: string;
  actionCode: string;
  params: EngineJsonObject;
  actorId?: string;
  userRole?: 'student' | 'teacher';
}): SimAction {
  const action: SimAction = {
    sceneCode: input.sceneCode,
    actionCode: input.actionCode,
    params: input.params,
  };
  if (input.actorId) action.actorId = input.actorId;
  if (input.userRole) action.userRole = input.userRole;
  return action;
}
