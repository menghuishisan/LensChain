'use client';

// useSimPanel.ts
// SimEngine 仿真面板 Hook（06.2 §一 / API v3.1 §4.1）。
// 管理 SimPanel 渲染器实例的生命周期，桥接 sim-engine-renderers 包与 React 业务组件。
// 消费者：SimTopBar / SimSceneGrid / SimSceneSlot / SimControlBar / SimSidebar 等。
// 注：InteractionForm 的 schema 拉取已下沉到 SimSceneSlot 内的 useSimInteraction，本 hook 不再聚合。

import { useEffect, useRef, useCallback, useState } from 'react';
import {
  createDefaultSimPanel,
  createRenderState,
  type SimPanel,
  type RenderState,
  type PanelLayoutItem,
  type WebSocketMessage,
  type SimAction,
  type TimeControlCommand,
  type SceneCategory,
  type InteractionInputMap,
} from '@lenschain/sim-engine-renderers';
import { useAuthStore } from '@/stores/authStore';
import { apiClient } from '@/lib/api-client';
import type { JsonObject, SimControlCommand, SimWSMessage } from '@/types/experiment';

export interface SimSceneConfig {
  sceneCode: string;
  category: string;
  algorithmType: string;
  title: string;
}

export interface UseSimPanelOptions {
  sessionId: string;
  scenes: SimSceneConfig[];
  initialLayout?: PanelLayoutItem[];
  layoutStorageKey?: string;
}

export interface UseSimPanelReturn {
  panel: SimPanel | null;
  connected: boolean;
  sceneStates: Map<string, RenderState>;
  layout: PanelLayoutItem[];
  latestMessage: SimWSMessage | null;
  attachScene: (sceneCode: string, category: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (sceneCode: string) => void;
  redrawScene: (sceneCode: string) => void;
  sendControl: (command: TimeControlCommand, value?: number) => void;
  sendSimControl: (command: SimControlCommand, value?: number) => void;
  sendAction: (action: SimAction) => void;
  stepBack: (sceneCode: string) => void;
  submitInteraction: (sceneCode: string, actionCode: string, params: JsonObject) => void;
  saveLayout: () => void;
  captureScene: (sceneCode: string) => string | null;
}

/**
 * useSimPanel 创建并管理 SimPanel 渲染器实例。
 *
 * 设计要点：
 * - panel 必须用 state 而不是 ref：子组件 SceneCanvas 的 useEffect 在父组件 useEffect 之前执行
 *   （React 对 effect 的提交顺序是子先父后），如果只用 ref，第一次 attachScene 调用时 ref
 *   仍是 null，调用变为 no-op，导致场景永不绑定 SceneView，Canvas 永远不绘制。
 *   用 state 后 panel 创建会触发重渲，所有 useCallback 依赖 panel 后随之更新，
 *   子组件 useEffect 因依赖变更重跑一次 attachScene，这次 panel 已就绪。
 * - latestMessage 暴露最新 WS 消息供 useSimSchemaInvalidation 等上层 hook 消费。
 */
export function useSimPanel(options: UseSimPanelOptions): UseSimPanelReturn {
  const { sessionId, scenes, initialLayout, layoutStorageKey } = options;
  const accessToken = useAuthStore(s => s.accessToken);

  const [panel, setPanel] = useState<SimPanel | null>(null);
  const [connected, setConnected] = useState(false);
  const [sceneStates, setSceneStates] = useState<Map<string, RenderState>>(new Map());
  const [layout, setLayout] = useState<PanelLayoutItem[]>(initialLayout ?? []);
  const [latestMessage, setLatestMessage] = useState<SimWSMessage | null>(null);

  const sceneConfigMapRef = useRef(new Map<string, SimSceneConfig>());
  useEffect(() => {
    const map = new Map<string, SimSceneConfig>();
    scenes.forEach(s => map.set(s.sceneCode, s));
    sceneConfigMapRef.current = map;
  }, [scenes]);

  useEffect(() => {
    if (!sessionId) return;
    // accessToken 仅作为"是否已登录"的 gate；URL 里实际使用的 token 由 urlProvider 通过
    // apiClient.ensureFreshAccessToken() 每次拨号前拿最新值（HTTP 双 token 同源刷新）。
    if (!accessToken) return;

    // 项目规约：所有实时连接必须基于 NEXT_PUBLIC_WS_BASE_URL（指向后端 :8080），见 src/lib/ws-url.ts。
    const rawBase = process.env.NEXT_PUBLIC_WS_BASE_URL ?? '';
    if (rawBase.length === 0) return;
    const wsBase = rawBase.replace(/^https?:/, (proto) => (proto === 'https:' ? 'wss:' : 'ws:')).replace(/\/$/, '');
    const endpoint = `${wsBase}/ws/sim-engine`;

    // urlProvider 在每次 WS 拨号 / 重连前被 SimPanel 调用：
    //   - 先走 apiClient.ensureFreshAccessToken()——若 access_token 距过期 < 5min
    //     会复用 HTTP 同款双 token 无感刷新拿新 token；
    //   - 再用最新 token 拼 URL。
    //
    // 直接读 Zustand 的 accessToken 是错的：组件渲染时绑定的 accessToken 会随时间
    // 过期，重连仍会带过期 token 触发 401 死循环（之前 sim-engine WS 死循环 401 即此因）。
    const urlProvider = async () => {
      const token = await apiClient.ensureFreshAccessToken();
      if (!token) return '';
      return `${endpoint}/${sessionId}?token=${token}`;
    };

    const instance = createDefaultSimPanel({
      sessionId,
      urlProvider,
      initialLayout: initialLayout ?? [],
      layoutStorageKey: layoutStorageKey ?? `sim-layout-${sessionId}`,
    });

    setPanel(instance);

    const unsubscribeStatus = instance.subscribeConnectionStatus(setConnected);

    // 订阅消息：render 类型触发 sceneStates 更新，所有消息暴露给上层。
    instance.subscribeMessages((message: WebSocketMessage) => {
      setLatestMessage(message as unknown as SimWSMessage);

      if (message.type === 'render') {
        const code = message.scene_code;
        if (code) {
          const state = instance.getSceneState(code);
          if (state) {
            setSceneStates(prev => {
              const next = new Map(prev);
              next.set(code, state);
              return next;
            });
          }
        }
      }
    });

    instance.connect((message: WebSocketMessage): RenderState => {
      const sceneCode = message.scene_code ?? '';
      const config = sceneConfigMapRef.current.get(sceneCode);
      return createRenderState({
        sceneCode,
        title: config?.title ?? '',
        category: (config?.category ?? 'node_network') as SceneCategory,
        timeControlMode: 'reactive',
        tick: typeof message.tick === 'number' ? message.tick : 0,
        envelope: (message.payload ?? { primitives: [] }) as unknown as RenderState['envelope'],
      });
    });

    return () => {
      unsubscribeStatus();
      instance.disconnect();
      setPanel(null);
      setConnected(false);
      setLatestMessage(null);
    };
  }, [sessionId, accessToken]);

  const attachScene = useCallback((sceneCode: string, category: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => {
    panel?.attachScene(sceneCode, category as SceneCategory, canvas, overlay);
  }, [panel]);

  const detachScene = useCallback((sceneCode: string) => {
    panel?.detachScene(sceneCode);
  }, [panel]);

  const redrawScene = useCallback((sceneCode: string) => {
    panel?.redrawScene(sceneCode);
  }, [panel]);

  const sendControl = useCallback((command: TimeControlCommand, value?: number) => {
    panel?.sendControl(command, value);
  }, [panel]);

  const sendSimControl = useCallback((command: SimControlCommand, value?: number) => {
    panel?.sendControl(command as TimeControlCommand, value);
  }, [panel]);

  const sendAction = useCallback((action: SimAction) => {
    panel?.sendAction(action);
  }, [panel]);

  const stepBack = useCallback((sceneCode: string) => {
    panel?.sendAction({ sceneCode, actionCode: '__step_back__', params: {} });
  }, [panel]);

  const submitInteraction = useCallback((sceneCode: string, actionCode: string, params: JsonObject) => {
    panel?.submitInteraction(sceneCode, actionCode, params as unknown as InteractionInputMap);
  }, [panel]);

  const saveLayoutCb = useCallback(() => {
    const currentLayout = panel?.getLayout() ?? [];
    panel?.saveLayout(currentLayout);
  }, [panel]);

  const captureScene = useCallback((sceneCode: string): string | null => {
    return panel?.captureScene(sceneCode) ?? null;
  }, [panel]);

  return {
    panel,
    connected,
    sceneStates,
    layout,
    latestMessage,
    attachScene,
    detachScene,
    redrawScene,
    sendControl,
    sendSimControl,
    sendAction,
    stepBack,
    submitInteraction,
    saveLayout: saveLayoutCb,
    captureScene,
  };
}
