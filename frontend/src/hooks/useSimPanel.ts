'use client';

// useSimPanel.ts
// SimEngine 仿真面板 Hook
// 管理 SimPanel 实例生命周期，桥接 sim-engine 渲染器包与 React 组件

import { useEffect, useRef, useCallback, useState } from 'react';
import {
  createDefaultSimPanel,
  createRenderState,
  type SimPanel,
  type RenderState,
  type PanelLayoutItem,
  type ControlDescriptor,
  type InteractionAction,
  type InteractionInputMap,
  type WebSocketMessage,
  type SimAction,
  type TimeControlCommand,
  type SceneCategory,
} from '@lenschain/sim-engine-renderers';
import { useAuthStore } from '@/stores/authStore';

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
  attachScene: (sceneCode: string, category: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (sceneCode: string) => void;
  sendControl: (command: TimeControlCommand, value?: number) => void;
  sendAction: (action: SimAction) => void;
  rewindTo: (targetTick: number) => void;
  getControls: (sceneCode: string) => ControlDescriptor[];
  getInteractionActions: (sceneCode: string) => InteractionAction[];
  submitInteraction: (sceneCode: string, actionCode: string, inputs: InteractionInputMap) => void;
  saveLayout: () => void;
  captureScene: (sceneCode: string) => string | null;
}

/**
 * SimEngine 仿真面板 Hook
 * 创建并管理 SimPanel 渲染器实例，自动处理 WebSocket 连接
 */
export function useSimPanel(options: UseSimPanelOptions): UseSimPanelReturn {
  const { sessionId, scenes, initialLayout, layoutStorageKey } = options;
  const accessToken = useAuthStore(s => s.accessToken);
  const panelRef = useRef<SimPanel | null>(null);
  const [connected, setConnected] = useState(false);
  const [sceneStates, setSceneStates] = useState<Map<string, RenderState>>(new Map());
  const [layout, setLayout] = useState<PanelLayoutItem[]>(initialLayout ?? []);

  // 通过 ref 持有 sceneCode → 配置映射表，避免 scenes 引用变化导致 WebSocket 重连
  const sceneConfigMapRef = useRef(new Map<string, SimSceneConfig>());
  useEffect(() => {
    const map = new Map<string, SimSceneConfig>();
    scenes.forEach(s => map.set(s.sceneCode, s));
    sceneConfigMapRef.current = map;
  }, [scenes]);

  useEffect(() => {
    if (!sessionId) return;

    // 项目规约：所有实时连接必须基于 NEXT_PUBLIC_WS_BASE_URL（指向后端 :8080），见 src/lib/ws-url.ts。
    // 之前用 window.location.host 会拿到 Next.js dev 服务器（:3000），无 WS 路由，浏览器握手立即失败
    // 且不会在 DevTools WS 面板出现，是 SimEngine 显示"未连接"的真正根因。
    // SimPanel 内部会自动拼接 `/${sessionId}?token=${token}`（renderers/shared/simPanel.ts），
    // 因此 endpoint 必须只传到 `/ws/sim-engine`（不含 sessionId、不含 token）。
    const rawBase = process.env.NEXT_PUBLIC_WS_BASE_URL ?? '';
    if (rawBase.length === 0) {
      // 环境变量缺失时直接放弃连接，避免回退到错误 host 静默失败
      return;
    }
    const wsBase = rawBase.replace(/^https?:/, (proto) => (proto === 'https:' ? 'wss:' : 'ws:')).replace(/\/$/, '');
    const endpoint = `${wsBase}/ws/sim-engine`;

    const panel = createDefaultSimPanel({
      sessionId,
      token: accessToken ?? '',
      endpoint,
      initialLayout: initialLayout ?? [],
      layoutStorageKey: layoutStorageKey ?? `sim-layout-${sessionId}`,
    });

    panelRef.current = panel;

    // 订阅连接状态，跟踪真实 WebSocket open/close
    const unsubscribeStatus = panel.subscribeConnectionStatus(setConnected);

    // 订阅消息以跟踪状态变化
    panel.subscribeMessages((message: WebSocketMessage) => {
      if (message.type === 'state_diff' || message.type === 'state_full') {
        const code = message.scene_code;
        if (code) {
          const state = panel.getSceneState(code);
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

    // 连接 WebSocket，stateResolver 将消息转为 RenderState
    panel.connect((message: WebSocketMessage): RenderState => {
      const sceneCode = message.scene_code ?? '';
      const config = sceneConfigMapRef.current.get(sceneCode);
      return createRenderState({
        sceneCode,
        title: config?.title ?? '',
        category: (config?.category ?? 'node_network') as SceneCategory,
        algorithmType: config?.algorithmType ?? '',
        timeControlMode: 'reactive',
        tick: typeof message.tick === 'number' ? message.tick : 0,
        renderData: (message.payload ?? {}) as RenderState['renderData'],
      });
    });

    return () => {
      unsubscribeStatus();
      panel.disconnect();
      panelRef.current = null;
      setConnected(false);
    };
  }, [sessionId, accessToken]);

  const attachScene = useCallback((sceneCode: string, category: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => {
    panelRef.current?.attachScene(sceneCode, category, canvas, overlay);
  }, []);

  const detachScene = useCallback((sceneCode: string) => {
    panelRef.current?.detachScene(sceneCode);
  }, []);

  const sendControl = useCallback((command: TimeControlCommand, value?: number) => {
    panelRef.current?.sendControl(command, value);
  }, []);

  const sendAction = useCallback((action: SimAction) => {
    panelRef.current?.sendAction(action);
  }, []);

  const rewindTo = useCallback((targetTick: number) => {
    panelRef.current?.rewindTo(targetTick);
  }, []);

  const getControls = useCallback((sceneCode: string): ControlDescriptor[] => {
    return panelRef.current?.getAvailableControls(sceneCode) ?? [];
  }, []);

  const getInteractionActions = useCallback((sceneCode: string): InteractionAction[] => {
    return panelRef.current?.listInteractionActions(sceneCode) ?? [];
  }, []);

  const submitInteraction = useCallback((sceneCode: string, actionCode: string, inputs: InteractionInputMap) => {
    panelRef.current?.submitInteraction(sceneCode, actionCode, inputs);
  }, []);

  const saveLayoutCb = useCallback(() => {
    const currentLayout = panelRef.current?.getLayout() ?? [];
    panelRef.current?.saveLayout(currentLayout);
  }, []);

  const captureScene = useCallback((sceneCode: string): string | null => {
    return panelRef.current?.captureScene(sceneCode) ?? null;
  }, []);

  return {
    panel: panelRef.current,
    connected,
    sceneStates,
    layout,
    attachScene,
    detachScene,
    sendControl,
    sendAction,
    rewindTo,
    getControls,
    getInteractionActions,
    submitInteraction,
    saveLayout: saveLayoutCb,
    captureScene,
  };
}
