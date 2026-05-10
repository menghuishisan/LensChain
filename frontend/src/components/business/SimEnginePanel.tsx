'use client';

// SimEnginePanel.tsx
// SimEngine 仿真面板主编排组件（06.2 §一）。
// 三段式布局：TopBar / MainCanvas+Sidebar / ControlBar + InteractionForm。
// 编排 useSimPanel、useSimMode、useSimInteraction、useSimSchemaInvalidation hook。
// 按模式（single/comparison/linkage/hybrid）和布局（grid/focus/carousel）渲染场景画布。

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, CardContent } from '@/components/ui/Card';
import { cn } from '@/lib/utils';
import { useSimPanel, type SimSceneConfig } from '@/hooks/useSimPanel';
import { useSimMode } from '@/hooks/useSimMode';
import { useSimInteraction, useSimSchemaInvalidation } from '@/hooks/useSimInteraction';
import { SimTopBar } from '@/components/business/SimTopBar';
import { SimSceneGrid, type SceneSlotConfig } from '@/components/business/SimSceneGrid';
import { SimControlBar } from '@/components/business/SimControlBar';
import { SimInteractionForm } from '@/components/business/SimInteractionForm';
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
  const firstTimeControlMode = scenes[0]?.scenario?.time_control_mode ?? 'reactive';

  const [viewportWidth] = useState(() => (typeof window !== 'undefined' ? window.innerWidth : 1440));
  const [layoutOverride, setLayoutOverride] = useState<SimLayoutMode | null>(null);
  const [interventionOpen, setInterventionOpen] = useState(false);
  const [annotationVisible, setAnnotationVisible] = useState(false);
  const [, setActiveAnnotationTool] = useState<AnnotationTool | null>(null);
  const isTeacher = userRole === 'teacher';

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
    timeControlMode: firstTimeControlMode,
  });

  const activeLayout = layoutOverride ?? simMode.layout;

  // schema_invalidated 信号处理
  useSimSchemaInvalidation(instanceID ?? '', sim.latestMessage);

  // 第一个场景的交互 schema（单场景时直接渲染，多场景时各场景分别获取）
  const firstScene = sceneConfigs[0];
  const firstInteraction = useSimInteraction({
    instanceID: instanceID ?? '',
    sceneCode: firstScene?.sceneCode ?? '',
    enabled: !!instanceID && !!firstScene,
    userRole,
  });

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

  // ─── 空态 ─────────────────────────────────────────
  if (!sessionID) {
    return (
      <Card className={cn('border-dashed', className)}>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-sm text-muted-foreground">可视化内容还在准备中，请稍后刷新查看。</p>
        </CardContent>
      </Card>
    );
  }

  // ─── 三段式布局 ───────────────────────────────────
  return (
    <Card className={cn('overflow-hidden flex flex-col', className)}>
      {/* §1.1 TopBar */}
      <SimTopBar
        mode={simMode.mode}
        layout={activeLayout}
        connected={sim.connected}
        sceneCount={sceneConfigs.length}
        onLayoutChange={setLayoutOverride}
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

        <div className="flex-1 p-4 overflow-auto">
          <SimSceneGrid
            scenes={sceneSlots}
            layout={activeLayout}
            attachScene={sim.attachScene}
            detachScene={sim.detachScene}
            redrawScene={sim.redrawScene}
            captureScene={sim.captureScene}
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
          timeControlMode={firstTimeControlMode}
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

      {/* §五 InteractionForm */}
      <SimInteractionForm
        actions={firstInteraction.actions}
        connected={sim.connected}
        onSubmit={(actionCode, params) =>
          firstScene && sim.submitInteraction(firstScene.sceneCode, actionCode, params)
        }
      />

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
