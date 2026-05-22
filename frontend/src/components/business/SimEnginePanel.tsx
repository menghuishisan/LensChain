'use client';

/**
 * SimEnginePanel — SimEngine 顶层 Tab 容器（06.2 §一）。
 *
 * 三段式布局：TopBar / 主画布区+侧栏 / ControlBar。
 * 教师附加层（标注工具栏 / 共享状态抽屉 / 干预抽屉）按 role 切组件，不开两套路由。
 *
 * 父组件契约（与历史一致，便于 ExperimentInstancePanel 直接消费）：
 *   sessionID / instanceID / scenes / experimentType / userRole / className
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useQueries, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  CheckCircle2,
  Columns3,
  Database,
  Focus as FocusIcon,
  LayoutGrid,
  PauseCircle,
  Pencil,
} from 'lucide-react';
import {
  mapInteractionDefinition,
  type InteractionDefinition,
  type InteractionSchema,
  type JsonObject,
  type SceneCategory,
} from '@lenschain/sim-engine-renderers';
import { Card, CardContent } from '@/components/ui/Card';
import { cn } from '@/lib/utils';
import { apiClient } from '@/lib/api-client';
import { useSimPanel, type SimSceneConfig } from '@/hooks/useSimPanel';
import { useSimMode, defaultLayout, DEFAULT_SPEED } from '@/hooks/useSimMode';
import { useSimSchemaInvalidation } from '@/hooks/useSimInteraction';
import { SimTopBar } from '@/components/business/SimTopBar';
import { SimControlBar } from '@/components/business/SimControlBar';
import { SimSceneGrid } from '@/components/business/SimSceneGrid';
import { SimSidebar } from '@/components/business/SimSidebar';
import { SimSharedStatePanel } from '@/components/business/SimSharedStatePanel';
import { SimCrossCanvasOverlay, type AnchorPosition } from '@/components/business/SimCrossCanvasOverlay';
import {
  SimTeacherInterventionPanel,
  type InterventionPayload,
} from '@/components/business/SimTeacherInterventionPanel';
import { SimAnnotationToolbar, type AnnotationTool } from '@/components/business/SimAnnotationToolbar';
import { SimInteractionForm } from '@/components/business/SimInteractionForm';
import { getInteractionSchema } from '@/services/experiment';
import type { ExperimentType, SimLayoutMode, SimSharedStateGroup, SimTimeControlMode } from '@/types/experiment';
import type { ID } from '@/types/api';

/** 父组件喂入的精简场景结构（来自 sim_scenes JOIN scenario）。 */
export interface SimEnginePanelScene {
  id: ID;
  scenario: {
    code: string;
    name: string;
    category: string;
    time_control_mode: SimTimeControlMode;
  } | null;
}

export interface SimEnginePanelProps {
  sessionID: string;
  instanceID: ID;
  scenes: readonly SimEnginePanelScene[];
  experimentType: ExperimentType;
  userRole: 'student' | 'teacher';
  className?: string;
}

const SCHEMA_QUERY_KEY = (instanceID: ID, sceneCode: string) =>
  ['sim-interaction-schema', instanceID, sceneCode] as const;
const CACHE_24H = 24 * 60 * 60 * 1000;

export function SimEnginePanel(props: SimEnginePanelProps) {
  const { sessionID, instanceID, scenes, experimentType, userRole, className } = props;
  const queryClient = useQueryClient();

  // ── 1. 把后端 sim_scenes 形态映射成 SimSceneConfig ──
  const sceneConfigs = useMemo<SimSceneConfig[]>(
    () => scenes
      .filter((s): s is SimEnginePanelScene & { scenario: NonNullable<SimEnginePanelScene['scenario']> } => s.scenario !== null)
      .map(s => ({
        sceneCode: s.scenario.code,
        title: s.scenario.name,
        category: s.scenario.category as SceneCategory,
        timeControlMode: s.scenario.time_control_mode,
      })),
    [scenes],
  );

  // ── 2. 启 SimPanel ──
  const {
    panel, connected, sceneStates, lastControlAck, linkTriggers,
    attachScene: rawAttachScene, detachScene: rawDetachScene,
    getResolvedLayout, dispatchAction, sendTimeControl, sendStepBack, captureScene,
  } = useSimPanel({
    sessionId: sessionID,
    scenes: sceneConfigs,
    userRole,
  });

  useSimSchemaInvalidation(panel, instanceID);

  // canvas 引用记录（供跨画布弧线的 anchor 解析使用）
  const canvasRefs = useRef<Map<string, HTMLCanvasElement>>(new Map());
  const attachScene = useCallback((sceneCode: string, canvas: HTMLCanvasElement) => {
    canvasRefs.current.set(sceneCode, canvas);
    rawAttachScene(sceneCode, canvas);
  }, [rawAttachScene]);
  const detachScene = useCallback((sceneCode: string) => {
    canvasRefs.current.delete(sceneCode);
    rawDetachScene(sceneCode);
  }, [rawDetachScene]);

  // 跨画布 anchor 坐标解析：联动模式下为 SimCrossCanvasOverlay 提供。
  // canvas DOM 视口另表示 + 该原语在 ResolvedLayout 中的像素偏移。
  const getAnchorPosition = useCallback((sceneCode: string, anchorId: string): AnchorPosition | null => {
    const canvas = canvasRefs.current.get(sceneCode);
    if (!canvas) return null;
    const layout = getResolvedLayout(sceneCode);
    const pos = layout?.positions.get(anchorId);
    if (!pos) return null;
    const rect = canvas.getBoundingClientRect();
    // ResolvedLayout 是 canvas 内部逻辑像素坐标；DPR 由 sceneView 含在 canvas.width 中，
    // CSS 尺寸 = canvas.clientWidth，按比例换算。
    const sx = canvas.clientWidth / Math.max(canvas.width, 1);
    const sy = canvas.clientHeight / Math.max(canvas.height, 1);
    return { x: rect.left + pos.x * sx, y: rect.top + pos.y * sy };
  }, [getResolvedLayout]);

  // ── 3. 模式判定：任一场景 RenderState.linked=true 或活跃期间收到过 link_trigger 即视为联动 ──
  const hasActiveLinkGroup = useMemo(() => {
    if (linkTriggers.length > 0) return true;
    for (const [, st] of sceneStates) {
      if (st.linked) return true;
    }
    return false;
  }, [linkTriggers, sceneStates]);

  const mode = useSimMode({
    experimentType,
    sceneCount: sceneConfigs.length,
    hasActiveLinkGroup,
  });

  // ── 4. 从 linkTriggers 派生 SharedState 视图（按 link_group 聚合，同名字段取最近一次变更）。
  // 本身就是协议语义：LinkTrigger.changed_fields + payload 即为最新 SharedState 增量。
  const sharedStateGroups = useMemo<SimSharedStateGroup[]>(() => {
    const groupMap = new Map<string, Map<string, { value: unknown; owner: string }>>();
    for (const t of linkTriggers) {
      let fields = groupMap.get(t.link_group);
      if (!fields) {
        fields = new Map();
        groupMap.set(t.link_group, fields);
      }
      for (const fname of t.changed_fields) {
        const val = (t.payload as Record<string, unknown>)[fname];
        fields.set(fname, { value: val, owner: t.source_scene });
      }
    }
    const out: SimSharedStateGroup[] = [];
    for (const [groupName, fields] of groupMap) {
      out.push({
        link_group_id: groupName,
        link_group_name: groupName,
        fields: Array.from(fields, ([field_name, { value, owner }]) => ({
          field_name,
          value,
          owner_scene: owner,
          owner_scene_label: owner,
        })),
      });
    }
    return out;
  }, [linkTriggers]);

  // ── 5. 拉每个场景的 InteractionSchema（并行 useQueries） ──
  const schemaQueries = useQueries({
    queries: sceneConfigs.map(cfg => ({
      queryKey: SCHEMA_QUERY_KEY(instanceID, cfg.sceneCode),
      queryFn: async () => {
        const raw = await getInteractionSchema(instanceID, cfg.sceneCode);
        return mapInteractionDefinition(raw as unknown as InteractionDefinition);
      },
      enabled: Boolean(instanceID) && Boolean(cfg.sceneCode),
      staleTime: CACHE_24H,
      gcTime: CACHE_24H,
    })),
  });

  const schemas = useMemo<Map<string, InteractionSchema | null>>(() => {
    const m = new Map<string, InteractionSchema | null>();
    sceneConfigs.forEach((cfg, i) => {
      m.set(cfg.sceneCode, schemaQueries[i]?.data ?? null);
    });
    return m;
  }, [sceneConfigs, schemaQueries]);

  // ── 5. 视图状态，与 URL ?layout / ?focus 双向同步（50 路由、设计稿 §8）──
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const urlLayout = searchParams?.get('layout') as SimLayoutMode | null;
  const urlFocus = searchParams?.get('focus');
  const [layoutMode, setLayoutMode] = useState<SimLayoutMode>(() => {
    if (urlLayout === 'grid' || urlLayout === 'focus' || urlLayout === 'carousel') return urlLayout;
    return defaultLayout(sceneConfigs.length, typeof window === 'undefined' ? 1920 : window.innerWidth);
  });
  const [focusedSceneCode, setFocusedSceneCode] = useState<string | null>(() => {
    if (urlFocus && sceneConfigs.some(s => s.sceneCode === urlFocus)) return urlFocus;
    return sceneConfigs[0]?.sceneCode ?? null;
  });
  // URL 反向写：状态变更 → router.replace（不走历史）。仅 layout/focus 变更时触发。
  useEffect(() => {
    if (!pathname) return;
    const params = new URLSearchParams(searchParams?.toString() ?? '');
    params.set('layout', layoutMode);
    if (focusedSceneCode) params.set('focus', focusedSceneCode); else params.delete('focus');
    const next = `${pathname}?${params.toString()}`;
    const current = `${pathname}?${searchParams?.toString() ?? ''}`;
    if (next !== current) router.replace(next, { scroll: false });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [layoutMode, focusedSceneCode]);

  const [isFullscreen, setIsFullscreen] = useState(false);
  const [annotationOpen, setAnnotationOpen] = useState(false);
  const [annotationTool, setAnnotationTool] = useState<AnnotationTool | null>(null);
  const [sharedStateOpen, setSharedStateOpen] = useState(false);
  const [interventionOpen, setInterventionOpen] = useState(false);
  /** 紧凑模式 Sidebar popover 开关（设计稿 §⑧）。 */
  const [sidebarPopoverOpen, setSidebarPopoverOpen] = useState(false);

  // ── 6. 时间控制本地状态 ──
  const [isRunning, setIsRunning] = useState(false);
  const [currentSpeed, setCurrentSpeed] = useState(DEFAULT_SPEED);
  // 当前 tick 取首场景（或 focused）作为代表
  const repTick = useMemo(() => {
    const code = focusedSceneCode ?? sceneConfigs[0]?.sceneCode;
    if (!code) return 0;
    return sceneStates.get(code)?.tick ?? 0;
  }, [focusedSceneCode, sceneConfigs, sceneStates]);

  // 时间控制 UI 用首场景 timeControlMode（多场景默认对齐第一个）
  const repTimeMode = sceneConfigs[0]?.timeControlMode ?? 'process';

  // ── 7. 时间控制回调 ──
  const handlePlay = useCallback(() => { sendTimeControl('play'); setIsRunning(true); }, [sendTimeControl]);
  const handlePause = useCallback(() => { sendTimeControl('pause'); setIsRunning(false); }, [sendTimeControl]);
  const handleStep = useCallback(() => { sendTimeControl('step'); }, [sendTimeControl]);
  const handleReset = useCallback(() => { sendTimeControl('reset'); setIsRunning(false); }, [sendTimeControl]);
  const handleSpeedChange = useCallback((s: number) => {
    sendTimeControl('set_speed', s);
    setCurrentSpeed(s);
  }, [sendTimeControl]);
  const handleStepBack = useCallback(() => {
    if (focusedSceneCode) sendStepBack(focusedSceneCode);
    else if (sceneConfigs[0]) sendStepBack(sceneConfigs[0].sceneCode);
  }, [focusedSceneCode, sceneConfigs, sendStepBack]);

  // control_ack 回滚：如果服务端 nack pause/play，撤回本地按钮态
  useEffect(() => {
    if (!lastControlAck || lastControlAck.success) return;
    if (lastControlAck.command === 'play') setIsRunning(false);
    else if (lastControlAck.command === 'pause') setIsRunning(true);
  }, [lastControlAck]);

  // ── FSM 5 态派生（设计稿 ⑥）。
  // error    ：已连接过 + 现不连接 + WS 存在过 -> 异常断开
  // finished ：任何场景 totalTicks 存在且已走到末尾
  // running  ：isRunning=true
  // paused   ：!isRunning 且任一场景 tick > 0
  // idle     ：默认态。
  const everConnectedRef = useRef(false);
  if (connected) everConnectedRef.current = true;
  const fsmState = useMemo<'idle' | 'running' | 'paused' | 'finished' | 'error'>(() => {
    if (everConnectedRef.current && !connected) return 'error';
    let anyTickPositive = false;
    let allFinished = sceneStates.size > 0;
    for (const [, st] of sceneStates) {
      if (st.tick > 0) anyTickPositive = true;
      if (st.totalTicks == null || st.tick < st.totalTicks) allFinished = false;
    }
    if (allFinished) return 'finished';
    if (isRunning) return 'running';
    if (anyTickPositive) return 'paused';
    return 'idle';
  }, [connected, sceneStates, isRunning]);

  // ── 8. ActionDef 提交 ──
  const handleSubmitAction = useCallback((sceneCode: string, actionCode: string, params: JsonObject) => {
    dispatchAction({ sceneCode, actionCode, params });
  }, [dispatchAction]);

  // ── 9. 截图 ──
  const handleCaptureFocused = useCallback(() => {
    const code = focusedSceneCode ?? sceneConfigs[0]?.sceneCode;
    if (!code) return;
    const dataUrl = captureScene(code);
    if (!dataUrl) return;
    const a = document.createElement('a');
    a.href = dataUrl;
    a.download = `sim-${code}-${Date.now()}.png`;
    a.click();
  }, [focusedSceneCode, sceneConfigs, captureScene]);

  // ── 10. 教师干预（POST /api/v1/teacher/experiments/:id/intervene） ──
  const handleIntervene = useCallback(async (payload: InterventionPayload) => {
    // 前端 InterventionType 与 backend internal/model/enum/experiment.go 中的 InterveneType*
    // 常量严格同名，无需映射（跨端奇名是冗余兼容层，已刪除）。
    const body: {
      action_code: string;
      params?: Record<string, unknown>;
      target_session_ids?: string[];
    } = { action_code: payload.type };
    if (payload.data && Object.keys(payload.data).length > 0) {
      body.params = payload.data;
    }
    // kick_student / push_step 等需要 target_session_ids；若 payload.data 已含 student_id
    // 由 services 层扩展（此处保持纯 REST 直连）。
    try {
      await apiClient.post<null>(`/teacher/experiments/${instanceID}/intervene`, body);
      // 干预成功后强制刷一次 schema（教师 push_step 可能改变 reactive_to_event 集合）
      sceneConfigs.forEach(cfg => {
        void queryClient.invalidateQueries({ queryKey: SCHEMA_QUERY_KEY(instanceID, cfg.sceneCode) });
      });
    } catch (err) {
      // 上报到全局 toast 由父层接管；这里只在控制台留 trace 方便排查。
      // eslint-disable-next-line no-console
      console.error('[SimEnginePanel] intervene failed', err);
    }
  }, [instanceID, sceneConfigs, queryClient]);

  // ── 11. 渲染 ──
  if (sceneConfigs.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="p-6 text-center text-sm text-muted-foreground">
          当前实验未配置仿真场景。
        </CardContent>
      </Card>
    );
  }

  const activeSceneTitle = sceneConfigs.find(s => s.sceneCode === focusedSceneCode)?.title
    ?? sceneConfigs[0]?.title
    ?? '';
  const breadcrumb = [
    '实验',
    '仿真',
    sceneConfigs.length > 1 ? `${activeSceneTitle} · 共 ${sceneConfigs.length}` : activeSceneTitle,
  ];

  // 收集所有场景的 metrics / timeline 聚合到侧栏
  const aggMetrics = useMemo(() => {
    const out: { label: string; value: string; tone?: 'info' | 'success' | 'warning' | 'danger' }[] = [];
    for (const [, st] of sceneStates) {
      if (st.metrics) out.push(...st.metrics);
    }
    return out;
  }, [sceneStates]);

  const aggTimeline = useMemo(() => {
    const out: NonNullable<ReturnType<typeof sceneStates.get>>['timeline'] = [];
    for (const [, st] of sceneStates) {
      if (st.timeline) (out as typeof out & unknown[]).push(...st.timeline);
    }
    return (out ?? []).sort((a, b) => a.tick - b.tick);
  }, [sceneStates]);

  // ── 紧凑响应式降级（设计稿 ⑧： panel <600w 划为 compact）。
  // 主体从横排（canvas | sidebar）改为竖排（canvas / sidebar），sidebar 由 360px 调为 100% 且限高。
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [compact, setCompact] = useState(false);
  useEffect(() => {
    const el = containerRef.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver((entries) => {
      for (const e of entries) {
        setCompact(e.contentRect.width < 600);
      }
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    <div
      ref={containerRef}
      className={cn('relative flex h-full flex-col overflow-hidden rounded-md border border-border bg-background', className)}
      data-compact={compact || undefined}
    >
      <SimTopBar
        breadcrumb={breadcrumb}
        isFullscreen={isFullscreen}
        isTeacher={userRole === 'teacher'}
        onCapture={handleCaptureFocused}
        onToggleFullscreen={() => setIsFullscreen(v => !v)}
        {...(userRole === 'teacher' ? { onOpenIntervention: () => setInterventionOpen(true) } : {})}
        {...(compact ? { onToggleSidebar: () => setSidebarPopoverOpen(v => !v) } : {})}
      />

      {/* 主体：左教师工具栏 + 主区 + 右侧栏； compact 模式下 sidebar 切为顶部 popover */}
      <div className="relative flex flex-1 overflow-hidden">
        {userRole === 'teacher' && (
          <div className="absolute left-2 top-2 z-20">
            <button
              type="button"
              onClick={() => setAnnotationOpen(v => !v)}
              className="mb-1 inline-flex items-center gap-1 rounded-md border border-border bg-card px-2 py-1 text-xs hover:bg-muted"
            >
              <Pencil className="h-3 w-3" /> 标注
            </button>
            <SimAnnotationToolbar
              visible={annotationOpen}
              currentTool={annotationTool}
              onSelectTool={setAnnotationTool}
              onClear={() => {
                // 通过 dispatchAction 让后端清空标注；具体 actionCode 由场景实现侧约定
                if (focusedSceneCode) {
                  dispatchAction({ sceneCode: focusedSceneCode, actionCode: 'clear_annotations', params: {} });
                }
              }}
            />
          </div>
        )}

        {/* 主区：垂直 = mode toolbar (shrink-0) + 画布栈 (flex-1 min-h-0)。
            必须用 flex flex-col 显式分配高度——以前是非 flex 块流，toolbar 与
            `h-full` 画布栈相加超出父高度，被 overflow-hidden 裁掉的同时画布栈
            内部仍按"父 100%"算高，导致 4 场景 grid 时每格 canvas 高度被压到负
            值，layoutSolver 抛 `主区计算后非正`。 */}
        <div className="flex flex-1 min-h-0 flex-col overflow-hidden">
          <div className="flex shrink-0 items-center justify-between border-b border-border bg-muted/20 px-2 py-1 text-[11px] text-muted-foreground">
            <span>模式：{mode}</span>
            <div className="flex items-center gap-1">
              {(['grid', 'focus', 'carousel'] as const).map(v => (
                <button
                  key={v}
                  type="button"
                  onClick={() => setLayoutMode(v)}
                  className={cn(
                    'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px]',
                    layoutMode === v ? 'bg-primary text-primary-foreground' : 'bg-card hover:bg-muted',
                  )}
                >
                  {v === 'grid' ? <LayoutGrid className="h-3 w-3" /> : v === 'focus' ? <FocusIcon className="h-3 w-3" /> : <Columns3 className="h-3 w-3" />}
                  {v === 'grid' ? '网格' : v === 'focus' ? '焦点' : '轮播'}
                </button>
              ))}
              <span className={cn('ml-2', connected ? 'text-emerald-600' : 'text-destructive')}>
                {connected ? '● 已连接' : '○ 未连接'}
              </span>
              {mode === 'linkage' && (
                <button
                  type="button"
                  onClick={() => setSharedStateOpen(v => !v)}
                  className={cn(
                    'ml-1 inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] hover:bg-muted',
                    sharedStateOpen ? 'bg-primary/15 text-primary' : 'bg-card',
                  )}
                >
                  <Database className="h-3 w-3" /> 共享状态
                </button>
              )}
            </div>
          </div>
          <div className="relative flex flex-1 min-h-0 flex-col">
            {/* 跨画布 M8 弧线（仅联动 / hybrid 模式叠加；pointer-events:none 透传交互） */}
            {(mode === 'linkage' || mode === 'hybrid') && (
              <SimCrossCanvasOverlay
                triggers={linkTriggers}
                getAnchorPosition={getAnchorPosition}
              />
            )}
            <div className="flex-1 min-h-0">
              <SimSceneGrid
                scenes={sceneConfigs}
                layoutMode={layoutMode}
                focusedSceneCode={focusedSceneCode}
                setFocusedSceneCode={(code) => { setFocusedSceneCode(code); if (layoutMode === 'grid') setLayoutMode('focus'); }}
                sceneStates={sceneStates}
                attachScene={attachScene}
                detachScene={detachScene}
                hideSlotHeader={sceneConfigs.length === 1}
                onCaptureScene={(code) => {
                  const data = captureScene(code);
                  if (!data) return;
                  const a = document.createElement('a');
                  a.href = data;
                  a.download = `sim-${code}-${Date.now()}.png`;
                  a.click();
                }}
              />
            </div>
            {/* 统一 InteractionForm：仅 single 或 focus 模式下渲染（对齐设计稿 ①/③-2/⑤）。 */}
            {(sceneConfigs.length === 1 || layoutMode === 'focus') && (() => {
              const activeCode = sceneConfigs.length === 1
                ? sceneConfigs[0]!.sceneCode
                : (focusedSceneCode ?? sceneConfigs[0]!.sceneCode);
              const activeSchema = schemas.get(activeCode) ?? null;
              return activeSchema && activeSchema.actions.length > 0 ? (
                <div className="shrink-0 border-t border-border bg-muted/10">
                  <SimInteractionForm
                    sceneCode={activeCode}
                    schema={activeSchema}
                    userRole={userRole}
                    onSubmit={handleSubmitAction}
                  />
                </div>
              ) : null;
            })()}
          </div>
        </div>

        {/* 右侧 sidebar / SharedState —— 仅非紧凑模式内联 */}
        {!compact && (
          mode === 'linkage' && sharedStateOpen ? (
            <SimSharedStatePanel groups={sharedStateGroups} />
          ) : (
            <SimSidebar
              mode={mode}
              metrics={aggMetrics}
              timeline={aggTimeline ?? []}
              linkTriggers={linkTriggers}
              {...(sceneConfigs.length === 1 ? {
                contentPrimitives: (sceneStates.get(sceneConfigs[0]!.sceneCode)?.envelope.primitives ?? [])
                  .filter(p => p.type === 'code_block' || p.type === 'math_formula' || p.type === 'progress_bar' || p.type === 'risk_gauge'),
              } : {})}
            />
          )
        )}
      </div>

      {/* 紧凑模式 sidebar 顶部 popover（设计稿 §⑧）。click 遮罩关闭。 */}
      {compact && sidebarPopoverOpen && (
        <>
          <div className="absolute inset-0 z-40 bg-background/40" onClick={() => setSidebarPopoverOpen(false)} />
          <div className="absolute right-2 top-12 z-50 max-h-[60vh] w-[280px] overflow-hidden rounded-md border border-border bg-card shadow-lg">
            {mode === 'linkage' && sharedStateOpen ? (
              <SimSharedStatePanel groups={sharedStateGroups} className="w-full" />
            ) : (
              <SimSidebar
                mode={mode}
                metrics={aggMetrics}
                timeline={aggTimeline ?? []}
                linkTriggers={linkTriggers}
                {...(sceneConfigs.length === 1 ? {
                  contentPrimitives: (sceneStates.get(sceneConfigs[0]!.sceneCode)?.envelope.primitives ?? [])
                    .filter(p => p.type === 'code_block' || p.type === 'math_formula' || p.type === 'progress_bar' || p.type === 'risk_gauge'),
                } : {})}
                className="w-full border-l-0"
              />
            )}
          </div>
        </>
      )}

      {/* reactive 模式整段隐藏 ControlBar（设计稿 ⑥）。 */}
      {repTimeMode !== 'reactive' && (
        <SimControlBar
          mode={mode}
          timeMode={repTimeMode}
          isRunning={isRunning}
          currentSpeed={currentSpeed}
          currentTick={repTick}
          {...(mode === 'linkage' ? { linkBadgeText: '联动同步' } : {})}
          onPlay={handlePlay}
          onPause={handlePause}
          onStep={handleStep}
          onStepBack={handleStepBack}
          onReset={handleReset}
          onSpeedChange={handleSpeedChange}
        />
      )}

      {/* paused / error / finished banner（设计稿 ⑦-1 / ⑦-2） */}
      {fsmState === 'paused' && (
        <div className="pointer-events-none absolute left-1/2 top-14 z-40 inline-flex -translate-x-1/2 items-center gap-1.5 rounded border border-yellow-500 bg-yellow-500/15 px-4 py-1 text-xs font-medium text-yellow-700 shadow">
          <PauseCircle className="h-3.5 w-3.5" /> 已暂停 · Tick {repTick}
        </div>
      )}
      {fsmState === 'error' && (
        <div className="absolute inset-x-0 top-14 z-40 mx-auto inline-flex w-fit items-center gap-1.5 rounded border border-destructive bg-destructive/15 px-4 py-1 text-xs font-medium text-destructive shadow">
          <AlertTriangle className="h-3.5 w-3.5" /> SimEngine 连接中断 · 请检查后端容器，点击重置按钮恢复
        </div>
      )}
      {fsmState === 'finished' && (
        <div className="pointer-events-none absolute right-4 top-14 z-40 inline-flex items-center gap-1.5 rounded border border-primary bg-primary/15 px-3 py-1 text-xs font-medium text-primary shadow">
          <CheckCircle2 className="h-3.5 w-3.5" /> 已完成
        </div>
      )}
      <SimTeacherInterventionPanel
        open={interventionOpen}
        onOpenChange={setInterventionOpen}
        onIntervene={handleIntervene}
      />
    </div>
  );
}
