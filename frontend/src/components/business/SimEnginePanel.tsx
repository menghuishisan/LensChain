'use client';

// SimEnginePanel.tsx
// SimEngine 可视化仿真面板
// 集成领域渲染器实现 Canvas 绘制，支持时间控制、场景交互、栅格布局

import { useEffect, useRef, useState } from 'react';
import { Activity, Camera, Maximize2, Minimize2, Pause, Play, RotateCcw, SkipForward, Wifi, WifiOff } from 'lucide-react';
import { useSimPanel } from '@/hooks/useSimPanel';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { cn } from '@/lib/utils';
import type { ID } from '@/types/api';
import type { RenderState, ControlDescriptor, InteractionAction, TimeControlCommand, InteractionInputValue } from '@lenschain/sim-engine-renderers';

/** SimEnginePanel 接受的最小场景输入类型。 */
export interface SimSceneInput {
  id: ID;
  scenario?: { code?: string; category?: string; name?: string; time_control_mode?: string } | null;
}

export interface SimEnginePanelProps {
  sessionID: ID;
  scenes: SimSceneInput[];
  onLayoutChange?: (layouts: Array<{ scene_id: ID; layout_position: Record<string, unknown> }>) => void;
  className?: string;
}

function getSceneCode(scene: SimSceneInput) {
  return scene.scenario?.code ?? String(scene.id);
}

/**
 * SimEngine 可视化仿真面板
 * 使用领域渲染器在 Canvas 上绘制仿真动画
 */
export function SimEnginePanel({ sessionID, scenes, onLayoutChange, className }: SimEnginePanelProps) {
  const sceneConfigs = scenes.map(s => ({
    sceneCode: getSceneCode(s),
    category: s.scenario?.category ?? 'node_network',
    algorithmType: s.scenario?.code ?? '',
    title: s.scenario?.name ?? '未命名场景',
  }));

  const sim = useSimPanel({
    sessionId: String(sessionID),
    scenes: sceneConfigs,
    layoutStorageKey: `sim-layout-${sessionID}`,
  });

  const [fullscreenScene, setFullscreenScene] = useState<string | null>(null);

  if (!sessionID) {
    return (
      <Card className={cn('border-amber-400/20 bg-slate-950 text-white', className)}>
        <CardContent className="p-6">
          <p className="text-sm text-amber-100">可视化内容还在准备中，请稍后刷新查看。</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn('overflow-hidden border-cyan-500/20 bg-slate-950 text-white', className)}>
      <CardHeader className="border-b border-white/10 py-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-sm text-white">
            <Activity className="h-4 w-4 text-cyan-300" />
            SimEngine 可视化仿真
          </CardTitle>
          <Badge variant={sim.connected ? 'default' : 'destructive'} className="text-xs gap-1">
            {sim.connected ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
            {sim.connected ? '已连接' : '未连接'}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4 p-4">
        <div className="grid gap-4 lg:grid-cols-2">
          {sceneConfigs.map(scene => {
            const state = sim.sceneStates.get(scene.sceneCode);
            const isFullscreen = fullscreenScene === scene.sceneCode;

            return (
              <div
                key={scene.sceneCode}
                className={cn(
                  'rounded-xl border border-white/10 bg-gradient-to-br from-slate-900 to-cyan-950/40 overflow-hidden',
                  isFullscreen && 'fixed inset-4 z-50 bg-slate-900',
                )}
              >
                {/* 场景头部 */}
                <div className="flex items-center justify-between px-3 py-2 border-b border-white/10">
                  <div>
                    <p className="text-sm font-medium">{scene.title}</p>
                    <p className="text-xs text-slate-400">{scene.category}</p>
                  </div>
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0 text-slate-300 hover:bg-white/10"
                      onClick={() => {
                        const dataUrl = sim.captureScene(scene.sceneCode);
                        if (dataUrl) {
                          const a = document.createElement('a');
                          a.href = dataUrl;
                          a.download = `${scene.sceneCode}-screenshot.png`;
                          a.click();
                        }
                      }}
                    >
                      <Camera className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0 text-slate-300 hover:bg-white/10"
                      onClick={() => setFullscreenScene(isFullscreen ? null : scene.sceneCode)}
                    >
                      {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
                    </Button>
                  </div>
                </div>

                {/* Canvas 渲染区域 */}
                <SceneCanvas
                  sceneCode={scene.sceneCode}
                  category={scene.category}
                  attachScene={sim.attachScene}
                  detachScene={sim.detachScene}
                  state={state}
                  isFullscreen={isFullscreen}
                />

                {/* 指标卡片 */}
                {state?.metrics && state.metrics.length > 0 && (
                  <div className="flex flex-wrap gap-2 px-3 py-2 border-t border-white/10">
                    {state.metrics.map((m, i) => (
                      <div key={i} className="rounded bg-white/5 px-2 py-1 text-xs">
                        <span className="text-slate-400">{m.label}:</span>{' '}
                        <span className="font-medium text-cyan-200">{m.value}</span>
                      </div>
                    ))}
                  </div>
                )}

                {/* 时间控制器 */}
                <TimeController
                  state={state}
                  controls={sim.getControls(scene.sceneCode)}
                  onControl={sim.sendControl}
                  connected={sim.connected}
                />

                {/* 场景交互面板 */}
                <InteractionForm
                  actions={sim.getInteractionActions(scene.sceneCode)}
                  onSubmit={(actionCode, inputs) => sim.submitInteraction(scene.sceneCode, actionCode, inputs)}
                  connected={sim.connected}
                />
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * 场景 Canvas 渲染组件
 */
function SceneCanvas({
  sceneCode,
  category,
  attachScene,
  detachScene,
  state,
  isFullscreen,
}: {
  sceneCode: string;
  category: string;
  attachScene: (code: string, cat: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (code: string) => void;
  state: RenderState | undefined;
  isFullscreen: boolean;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const attachedRef = useRef(false);

  useEffect(() => {
    if (!canvasRef.current || attachedRef.current) return;

    attachScene(sceneCode, category, canvasRef.current, svgRef.current as unknown as HTMLElement ?? undefined);
    attachedRef.current = true;

    return () => {
      if (attachedRef.current) {
        detachScene(sceneCode);
        attachedRef.current = false;
      }
    };
  }, [sceneCode, category, attachScene, detachScene]);

  const height = isFullscreen ? 'h-[calc(100vh-200px)]' : 'h-64';

  return (
    <div className={cn('relative', height)}>
      <canvas ref={canvasRef} className="absolute inset-0 w-full h-full" />
      <svg ref={svgRef} className="absolute inset-0 w-full h-full pointer-events-none" xmlns="http://www.w3.org/2000/svg" />
      {!state && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/50">
          <div className="text-center">
            <div className="mx-auto h-12 w-12 rounded-full border border-cyan-300/30 bg-cyan-300/10 animate-pulse" />
            <p className="mt-2 text-xs text-slate-400">等待仿真数据...</p>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * 时间控制器组件
 * 根据时间控制模式动态渲染按钮
 */
function TimeController({
  state,
  controls,
  onControl,
  connected,
}: {
  state: RenderState | undefined;
  controls: ControlDescriptor[];
  onControl: (command: TimeControlCommand, value?: number) => void;
  connected: boolean;
}) {
  const [speed, setSpeed] = useState(1);

  if (!controls.length) return null;

  const iconMap: Record<string, React.ReactNode> = {
    play: <Play className="h-3.5 w-3.5" />,
    pause: <Pause className="h-3.5 w-3.5" />,
    step: <SkipForward className="h-3.5 w-3.5" />,
    reset: <RotateCcw className="h-3.5 w-3.5" />,
    resume: <Play className="h-3.5 w-3.5" />,
  };

  const speedOptions = [0.5, 1, 1.5, 2];

  return (
    <div className="flex flex-wrap items-center gap-2 px-3 py-2 border-t border-white/10">
      {controls.map(ctrl => (
        <Button
          key={ctrl.command}
          variant="outline"
          size="sm"
          className="h-7 text-xs border-white/15 bg-white/5 text-white hover:bg-white/10"
          disabled={!connected || !ctrl.enabled}
          onClick={() => onControl(ctrl.command)}
        >
          {iconMap[ctrl.command] ?? null}
          {ctrl.label}
        </Button>
      ))}

      {controls.some(c => c.command === 'set_speed' || c.command === 'play' || c.command === 'resume') && (
        <div className="flex items-center gap-1 ml-2">
          <span className="text-xs text-slate-400">速度:</span>
          {speedOptions.map(s => (
            <Button
              key={s}
              variant={speed === s ? 'secondary' : 'ghost'}
              size="sm"
              className="h-6 px-2 text-xs"
              disabled={!connected}
              onClick={() => { setSpeed(s); onControl('set_speed', s); }}
            >
              {s}x
            </Button>
          ))}
        </div>
      )}

      {state && (
        <span className="ml-auto text-xs text-slate-500">
          Tick: {state.tick ?? 0}
        </span>
      )}
    </div>
  );
}

/**
 * 场景交互面板
 * 根据 schema 动态生成操作表单
 */
function InteractionForm({
  actions,
  onSubmit,
  connected,
}: {
  actions: InteractionAction[];
  onSubmit: (actionCode: string, inputs: Record<string, InteractionInputValue>) => void;
  connected: boolean;
}) {
  const [inputs, setInputs] = useState<Record<string, Record<string, InteractionInputValue>>>({});

  if (!actions.length) return null;

  return (
    <div className="px-3 py-2 border-t border-white/10 space-y-2">
      <p className="text-xs font-medium text-slate-300">场景操作</p>
      <div className="flex flex-wrap gap-2">
        {actions.map(action => {
          const hasFields = action.fields && action.fields.length > 0;
          const actionInputs = inputs[action.actionCode] ?? {};

          return (
            <div key={action.actionCode} className="flex items-center gap-1">
              {hasFields && action.fields.map(field => (
                <Input
                  key={field.key}
                  className="h-7 w-24 text-xs border-white/10 bg-white/5 text-white"
                  placeholder={field.label}
                  value={String(actionInputs[field.key] ?? '')}
                  onChange={e => setInputs(prev => ({
                    ...prev,
                    [action.actionCode]: { ...prev[action.actionCode], [field.key]: e.target.value },
                  }))}
                />
              ))}
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-xs border-cyan-300/20 bg-cyan-300/5 text-cyan-100 hover:bg-cyan-300/10"
                disabled={!connected}
                onClick={() => onSubmit(action.actionCode, actionInputs)}
              >
                {action.label}
              </Button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
