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
  redrawScene: (sceneCode: string) => void;
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
  // panel 必须用 state 而不是 ref：子组件 SceneCanvas 的 useEffect 在父组件 useEffect 之前执行
  // （React 对 effect 的提交顺序是子先父后），如果只用 ref，第一次 attachScene 调用时 ref
  // 仍是 null，调用变为 no-op，导致场景永不绑定 SceneView，Canvas 永远不绘制。
  // 用 state 后 panel 创建会触发重渲，所有 useCallback 依赖 panel 后随之更新，
  // 子组件 useEffect 因依赖变更重跑一次 attachScene，这次 panel 已就绪。
  const [panel, setPanel] = useState<SimPanel | null>(null);
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

    const instance = createDefaultSimPanel({
      sessionId,
      token: accessToken ?? '',
      endpoint,
      initialLayout: initialLayout ?? [],
      layoutStorageKey: layoutStorageKey ?? `sim-layout-${sessionId}`,
    });

    setPanel(instance);

    // 订阅连接状态，跟踪真实 WebSocket open/close
    const unsubscribeStatus = instance.subscribeConnectionStatus(setConnected);

    // 订阅消息以跟踪状态变化
    instance.subscribeMessages((message: WebSocketMessage) => {
      if (message.type === 'state_diff' || message.type === 'state_full') {
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

    // 连接 WebSocket，stateResolver 将消息转为 RenderState
    instance.connect((message: WebSocketMessage): RenderState => {
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
      instance.disconnect();
      setPanel(null);
      setConnected(false);
    };
  }, [sessionId, accessToken]);

  // 所有暴露给消费者的回调都依赖 panel state，panel 创建后回调身份变化，
  // 子组件 useEffect（依赖回调）会重跑一次 attachScene，确保场景在 panel 就绪后绑定。
  const attachScene = useCallback((sceneCode: string, category: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => {
    panel?.attachScene(sceneCode, category, canvas, overlay);
  }, [panel]);

  const detachScene = useCallback((sceneCode: string) => {
    panel?.detachScene(sceneCode);
  }, [panel]);

  // redrawScene 用于画布尺寸变化后强制以当前缓存状态重绘一帧。
  // 不修改 stateCache，仅触发 SceneView.render 重新走渲染管线。
  const redrawScene = useCallback((sceneCode: string) => {
    panel?.redrawScene(sceneCode);
  }, [panel]);

  const sendControl = useCallback((command: TimeControlCommand, value?: number) => {
    panel?.sendControl(command, value);
  }, [panel]);

  const sendAction = useCallback((action: SimAction) => {
    panel?.sendAction(action);
  }, [panel]);

  const rewindTo = useCallback((targetTick: number) => {
    panel?.rewindTo(targetTick);
  }, [panel]);

  const getControls = useCallback((sceneCode: string): ControlDescriptor[] => {
    return panel?.getAvailableControls(sceneCode) ?? [];
  }, [panel]);

  const getInteractionActions = useCallback((sceneCode: string): InteractionAction[] => {
    return panel?.listInteractionActions(sceneCode) ?? [];
  }, [panel]);

  const submitInteraction = useCallback((sceneCode: string, actionCode: string, inputs: InteractionInputMap) => {
    panel?.submitInteraction(sceneCode, actionCode, inputs);
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
    attachScene,
    detachScene,
    redrawScene,
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
