'use client';

// SimEnginePanel.tsx
// SimEngine 仿真面板主编排组件（06.2 §一）。
// 三段式布局：TopBar / MainCanvas+Sidebar / ControlBar。
// InteractionForm 由 SimSceneGrid 内每个 SimSceneSlot 自管（§5.6 文档要求每场景独立）。
// 编排 useSimPanel、useSimMode、useSimSchemaInvalidation hook；
// schema 拉取下沉到 SimSceneSlot 内的 useSimInteraction，本组件不再聚合。

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, CardContent } from '@/components/ui/Card';
import { cn } from '@/lib/utils';
import { useSimPanel, type SimSceneConfig } from '@/hooks/useSimPanel';
import { useSimMode } from '@/hooks/useSimMode';
import { useSimSchemaInvalidation } from '@/hooks/useSimInteraction';
import { SimTopBar } from '@/components/business/SimTopBar';
import { SimSceneGrid, type SceneSlotConfig } from '@/components/business/SimSceneGrid';
import { SimControlBar } from '@/components/business/SimControlBar';
import { SimSidebar, type MetricItem, type LinkIndicatorItem } from '@/components/business/SimSidebar';
import { SimSharedStatePanel } from '@/components/business/SimSharedStatePanel';
import { SimTeacherInterventionPanel } from '@/components/business/SimTeacherInterventionPanel';
import { SimAnnotationToolbar, type AnnotationTool } from '@/components/business/SimAnnotationToolbar';
import { SimCrossCanvasOverlay, type LinkArc } from '@/components/business/SimCrossCanvasOverlay';
import type { ExperimentType, SimLayoutMode, SimSharedStateGroup, SimTimeControlMode } from '@/types/experiment';

/** SimEnginePanel 外部可见场景信息。 */
export interface SceneInfo {
  scenario?: {
    code?: string;
    name?: string;
    category?: string;
    time_control_mode?: SimTimeControlMode;
  } | null;
  scene_id?: string;
  link_group_id?: string | null;
  link_group_name?: string | null;
}

function getSceneCode(scene: SceneInfo): string {
  return scene.scenario?.code ?? scene.scene_id ?? 'unknown';
}

export interface SimEnginePanelProps {
  sessionID: string | null;
  instanceID?: string;
  scenes: SceneInfo[];
  experimentType?: ExperimentType;
  userRole?: string;
  onLayoutChange?: (layout: unknown) => void;
  className?: string;
}

/**
 * SimEnginePanel 三段式仿真面板（06.2 §一）。
 *
 * 自动判定运行模式（single/comparison/linkage/hybrid），
 * 根据场景数选择布局（grid/focus/carousel），
 * 渲染时间控制栏（process/continuous 模式）和交互表单（ActionDef 驱动）。
 */
export function SimEnginePanel({
  sessionID,
  instanceID,
  scenes,
  experimentType = 1,
  userRole = 'student',
  className,
}: SimEnginePanelProps) {
  // ─── 场景配置构建 ─────────────────────────────────
  const sceneConfigs: SimSceneConfig[] = useMemo(
    () =>
      scenes.map((s) => ({
        sceneCode: getSceneCode(s),
        category: s.scenario?.category ?? 'node_network',
        algorithmType: s.scenario?.code ?? '',
        title: s.scenario?.name ?? '未命名场景',
      })),
    [scenes],
  );

  // ─── 核心 Hook ────────────────────────────────────
  const sim = useSimPanel({
    sessionId: String(sessionID ?? ''),
    scenes: sceneConfigs,
    layoutStorageKey: `sim-layout-${sessionID}`,
  });

  const hasActiveLinkGroup = scenes.some((s) => s.link_group_id != null);
  // 与 sim-engine 后端 `resolveSessionClockMode` 保持一致（engine_util.go:157）：
  // 任一场景为 process → 会话 process；否则任一 continuous → continuous；否则 reactive。
  // 仅取 scenes[0] 会在多模式混合模板（如 pow-mining=continuous + pbft=process）时
  // 导致前端按 continuous 渲染、发送 `resume`，被后端按 process 拒绝（unsupported），
  // 时钟永远无法启动、步数停在 0。
  const sessionTimeControlMode: SimTimeControlMode = (() => {
    let hasContinuous = false;
    for (const s of scenes) {
      const mode = s.scenario?.time_control_mode;
      if (mode === 'process') return 'process';
      if (mode === 'continuous') hasContinuous = true;
    }
    return hasContinuous ? 'continuous' : 'reactive';
  })();

  // 文档 §3.4：viewport < 1280 → 自动降级为 focus；必须随窗口缩放联动，
  // 因此挂载后绑定 resize（rAF 节流），而不是 useState 一次性快照。
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth : 1440,
  );
  useEffect(() => {
    if (typeof window === 'undefined') return;
    let raf = 0;
    const onResize = () => {
      if (raf) return;
      raf = window.requestAnimationFrame(() => {
        raf = 0;
        setViewportWidth(window.innerWidth);
      });
    };
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      if (raf) window.cancelAnimationFrame(raf);
    };
  }, []);
  const [layoutOverride, setLayoutOverride] = useState<SimLayoutMode | null>(null);
  const [interventionOpen, setInterventionOpen] = useState(false);
  const [annotationVisible, setAnnotationVisible] = useState(false);
  const [, setActiveAnnotationTool] = useState<AnnotationTool | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);
  const isTeacher = userRole === 'teacher';

  // 单一场景引用（仅用于 ControlBar 的 stepBack/Reset、Sidebar 的 micro_steps 等“面板全局级”读取）。
  // 交互表单 schema 由每个 SimSceneSlot 内部 useSimInteraction 自己拉，本处不再需要。
  const firstScene = sceneConfigs[0];

  // 面板级全屏（§1.1 ⬛）：调用浏览器 Fullscreen API，与场景级全屏（SimSceneSlot 内 fixed）不交叉。
  const handleFullscreenToggle = useCallback(() => {
    const el = cardRef.current;
    if (!el) return;
    if (document.fullscreenElement) {
      void document.exitFullscreen();
    } else {
      void el.requestFullscreen();
    }
  }, []);

  // 跟踪浏览器全屏状态变化（ESC 退出也能反映）。
  useEffect(() => {
    const onChange = () => setIsFullscreen(document.fullscreenElement === cardRef.current);
    document.addEventListener('fullscreenchange', onChange);
    return () => document.removeEventListener('fullscreenchange', onChange);
  }, []);

  // 顶部 📷 截图：截主场景画布（多场景时取第一个，focus/carousel 也应由 slot 内部 📷 处理）。
  const handleTopScreenshot = useCallback(() => {
    if (!firstScene) return;
    const dataUrl = sim.captureScene(firstScene.sceneCode);
    if (!dataUrl) return;
    const a = document.createElement('a');
    a.href = dataUrl;
    a.download = `${firstScene.sceneCode}-screenshot.png`;
    a.click();
  }, [firstScene, sim]);

  // 标注工具切换：通过 SimPanel 下发 annotation 原语设置当前绘制模式
  const handleAnnotationToolChange = useCallback((tool: AnnotationTool | null) => {
    setActiveAnnotationTool(tool);
    if (sim.panel) {
      sim.panel.sendAction({
        sceneCode: '__annotation__',
        actionCode: 'set_tool',
        params: { tool: tool ?? 'none' },
      });
    }
  }, [sim.panel]);

  // 清空所有标注：通过 SimPanel 下发 clear_all 指令
  const handleAnnotationClear = useCallback(() => {
    setActiveAnnotationTool(null);
    if (sim.panel) {
      sim.panel.sendAction({
        sceneCode: '__annotation__',
        actionCode: 'clear_all',
        params: {},
      });
    }
  }, [sim.panel]);

  const simMode = useSimMode({
    experimentType,
    sceneCount: sceneConfigs.length,
    hasActiveLinkGroup,
    viewportWidth,
    timeControlMode: sessionTimeControlMode,
  });

  const activeLayout = layoutOverride ?? simMode.layout;

  // schema_invalidated 信号处理
  useSimSchemaInvalidation(instanceID ?? '', sim.latestMessage);

  // ─── 状态派生 ─────────────────────────────────────
  const firstState = firstScene ? sim.sceneStates.get(firstScene.sceneCode) : undefined;
  const currentTick = firstState?.tick ?? 0;

  // 从 control_ack 消息推导播放状态
  const [playing, setPlaying] = useState(false);
  useEffect(() => {
    const msg = sim.latestMessage;
    if (!msg || msg.type !== 'control_ack') return;
    const ack = msg.payload as { command?: string; success?: boolean };
    if (!ack.success) return;
    if (ack.command === 'play' || ack.command === 'resume') setPlaying(true);
    else if (ack.command === 'pause' || ack.command === 'reset') setPlaying(false);
  }, [sim.latestMessage]);

  const sceneSlots: SceneSlotConfig[] = sceneConfigs.map((sc) => ({
    sceneCode: sc.sceneCode,
    category: sc.category,
    title: sc.title,
    hasState: sim.sceneStates.has(sc.sceneCode),
  }));

  const metrics: MetricItem[] = useMemo(() => {
    if (!firstState?.metrics) return [];
    return firstState.metrics.map((m) => ({ label: m.label, value: m.value }));
  }, [firstState?.metrics]);

  const linkIndicators: LinkIndicatorItem[] = useMemo(() => {
    return scenes
      .filter((s) => s.link_group_id)
      .map((s) => ({
        linkGroup: s.link_group_id!,
        status: 'idle' as const,
        label: s.link_group_name ?? s.link_group_id!,
      }));
  }, [scenes]);

  const microSteps = firstState?.envelope?.micro_steps ?? [];

  // 从 WS render 消息的 container_data.shared_state 累积联动 SharedState 字段
  const sharedStateRef = useRef(new Map<string, SimSharedStateGroup>());
  const [sharedStateGroups, setSharedStateGroups] = useState<SimSharedStateGroup[]>([]);
  useEffect(() => {
    const msg = sim.latestMessage;
    if (!msg || msg.type !== 'render') return;
    const payload = msg.payload as { container_data?: { shared_state?: Record<string, unknown> } };
    const sharedData = payload.container_data?.shared_state;
    if (!sharedData) return;

    const sceneCode = msg.scene_code ?? '';
    const sceneInfo = scenes.find((s) => getSceneCode(s) === sceneCode);
    const linkGroupId = sceneInfo?.link_group_id;
    if (!linkGroupId) return;

    const groupMap = sharedStateRef.current;
    let group = groupMap.get(linkGroupId);
    if (!group) {
      group = {
        link_group_id: linkGroupId,
        link_group_name: sceneInfo?.link_group_name ?? linkGroupId,
        fields: [],
      };
      groupMap.set(linkGroupId, group);
    }
    for (const [fieldName, value] of Object.entries(sharedData)) {
      const existing = group.fields.find((f) => f.field_name === fieldName);
      if (existing) {
        existing.value = value;
      } else {
        group.fields.push({
          field_name: fieldName,
          value,
          owner_scene: sceneCode,
          owner_scene_label: sceneInfo?.scenario?.name ?? sceneCode,
        });
      }
    }
    setSharedStateGroups(Array.from(groupMap.values()).map((g) => ({ ...g, fields: [...g.fields] })));
  }, [sim.latestMessage, scenes]);

  // 从 WS link_triggers 消息实时推导跨画布弧线坐标
  const [linkArcs, setLinkArcs] = useState<LinkArc[]>([]);
  const mainCanvasRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (simMode.mode !== 'linkage' || sceneConfigs.length <= 1) return;
    const msg = sim.latestMessage;
    if (!msg || msg.type !== 'render') return;
    const envelope = msg.payload as { link_triggers?: Array<{
      id: string;
      source_scene: string;
      link_group: string;
      source_anchor_id?: string;
      target_anchor_id?: string;
      changed_fields?: string[];
      ts: number;
    }> };
    const triggers = envelope.link_triggers;
    if (!triggers || triggers.length === 0) return;

    const container = mainCanvasRef.current;
    if (!container) return;
    const containerRect = container.getBoundingClientRect();

    const newArcs: LinkArc[] = triggers.map((t) => {
      // 尝试从 DOM 中查找 anchor 元素的几何中心
      const sourceEl = t.source_anchor_id ? container.querySelector(`[data-anchor-id="${t.source_anchor_id}"]`) : null;
      const targetEl = t.target_anchor_id ? container.querySelector(`[data-anchor-id="${t.target_anchor_id}"]`) : null;
      const sourceRect = sourceEl?.getBoundingClientRect();
      const targetRect = targetEl?.getBoundingClientRect();

      return {
        id: t.id,
        sourceX: sourceRect ? sourceRect.left + sourceRect.width / 2 - containerRect.left : 100,
        sourceY: sourceRect ? sourceRect.top + sourceRect.height / 2 - containerRect.top : 100,
        targetX: targetRect ? targetRect.left + targetRect.width / 2 - containerRect.left : 300,
        targetY: targetRect ? targetRect.top + targetRect.height / 2 - containerRect.top : 100,
        colorType: (t.link_group?.includes('attack') ? 'attack'
          : t.link_group?.includes('network') ? 'network'
          : t.link_group?.includes('crypto') ? 'crypto'
          : t.link_group?.includes('economic') ? 'economic'
          : t.link_group?.includes('consensus') ? 'consensus'
          : 'blockchain-integrity') as LinkArc['colorType'],
        label: t.changed_fields?.join(', '),
        createdAt: Date.now(),
      };
    });

    setLinkArcs((prev) => [...prev, ...newArcs]);
  }, [sim.latestMessage, simMode.mode, sceneConfigs.length]);

  // ─── 空态 ──────────────────────────────────────
  if (!sessionID) {
    return (
      <Card className={cn('border-dashed h-full', className)}>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-sm text-muted-foreground">可视化内容还在准备中，请稍后刷新查看。</p>
        </CardContent>
      </Card>
    );
  }

  // ─── 三段式布局 ──────────────────────────────────
  // h-full 是关键：Card 填满父容器高度，使内部 flex-1 画布区拿到明确高度，避免
  // “画布薗充、ControlBar/InteractionForm 被推出视口”问题。选 bg-background 避免全屏时透明背景。
  return (
    <Card ref={cardRef} className={cn('overflow-hidden flex flex-col h-full bg-background', className)}>
      {/* §1.1 TopBar */}
      <SimTopBar
        mode={simMode.mode}
        layout={activeLayout}
        connected={sim.connected}
        sceneCount={sceneConfigs.length}
        isFullscreen={isFullscreen}
        onLayoutChange={setLayoutOverride}
        onScreenshot={handleTopScreenshot}
        onFullscreenToggle={handleFullscreenToggle}
        onIntervene={isTeacher ? () => setInterventionOpen(true) : undefined}
        onAnnotationToggle={isTeacher ? () => setAnnotationVisible((v) => !v) : undefined}
        annotationActive={annotationVisible}
      />

      {/* §1.1 MainCanvas + Sidebar */}
      <div ref={mainCanvasRef} className="relative flex flex-1 min-h-0">
        {/* §7.1 教师标注工具栏（左侧浮动） */}
        {isTeacher && annotationVisible && (
          <SimAnnotationToolbar
            onToolChange={handleAnnotationToolChange}
            onClearAll={handleAnnotationClear}
          />
        )}

        {/* §6.2 联动跨画布弧线（仅 linkage 模式同屏多场景） */}
        {simMode.mode === 'linkage' && sceneConfigs.length > 1 && (
          <SimCrossCanvasOverlay arcs={linkArcs} />
        )}

        {/* 画布区：只该区域内部 overflow-auto，InteractionForm 以外的外层保持 overflow-hidden，
            这样 ControlBar 依然钉在底部，交互表单随 slot 在各场景下方不需页面滚动。 */}
        <div className="flex-1 p-4 overflow-auto min-w-0">
          <SimSceneGrid
            scenes={sceneSlots}
            layout={activeLayout}
            instanceID={instanceID}
            userRole={userRole}
            connected={sim.connected}
            attachScene={sim.attachScene}
            detachScene={sim.detachScene}
            redrawScene={sim.redrawScene}
            captureScene={sim.captureScene}
            onSubmitInteraction={(code, action, params) => sim.submitInteraction(code, action, params)}
            onLayoutChange={setLayoutOverride}
          />
        </div>

        <SimSidebar
          mode={simMode.mode}
          metrics={metrics}
          linkIndicators={linkIndicators}
          containerHealth={[]}
          microSteps={microSteps}
          activeMicroStepId={firstState?.activeMicroStepId}
        />
      </div>

      {/* §6.3 SharedState 联动面板（仅 linkage 模式） */}
      {simMode.showSharedStatePanel && <SimSharedStatePanel groups={sharedStateGroups} />}

      {/* §四 ControlBar（reactive 模式不渲染） */}
      {simMode.showTimeControl && (
        <SimControlBar
          timeControlMode={sessionTimeControlMode}
          mode={simMode.mode}
          tick={currentTick}
          playing={playing}
          speed={1}
          connected={sim.connected}
          canStepBack={simMode.canStepBack}
          stepBackTooltip={simMode.stepBackTooltip}
          onControl={(cmd, val) => sim.sendSimControl(cmd, val)}
          onStepBack={() => firstScene && sim.stepBack(firstScene.sceneCode)}
        />
      )}

      {/* §五 InteractionForm 已下沉到 SimSceneSlot 内部，本处不再渲染全局 form。 */}

      {/* §七.2 教师干预抽屉面板 */}
      {isTeacher && instanceID && (
        <SimTeacherInterventionPanel
          experimentID={instanceID}
          open={interventionOpen}
          onClose={() => setInterventionOpen(false)}
        />
      )}
    </Card>
  );
}
