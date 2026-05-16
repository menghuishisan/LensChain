'use client';

/**
 * SimSceneGrid — 多场景主区布局（重构版）。
 *
 * 三种 layoutMode：
 *   • grid     : 1×N / 2×2 网格，场景 ≤4 用此模式（HTML 行 122-125）
 *   • focus    : 主区一个 + 右侧缩略图竖排（HTML 行 1033-1041）
 *   • carousel : 单个 + ◀ ▶ 翻页（HTML 行 1077-1100）
 *
 * 不再向 slot 注入 schema / userRole / onSubmit —— 这些在 SimEnginePanel 顶层处理；
 * 单个 slot 只负责画布 + 内容类原语 dropdown。
 */

import { useMemo, useState } from 'react';
import { ChevronLeft, ChevronRight, LayoutGrid } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/utils';
import { SimSceneSlot } from '@/components/business/SimSceneSlot';
import type { RenderState } from '@lenschain/sim-engine-renderers';
import type { SimLayoutMode } from '@/types/experiment';
import type { SimSceneConfig } from '@/hooks/useSimPanel';

export interface SimSceneGridProps {
  scenes: readonly SimSceneConfig[];
  layoutMode: SimLayoutMode;
  /** focus 模式被聚焦的 sceneCode；null = 默认首场景。 */
  focusedSceneCode: string | null;
  setFocusedSceneCode: (sceneCode: string) => void;
  sceneStates: ReadonlyMap<string, RenderState>;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  onCaptureScene?: (sceneCode: string) => void;
  /** single 模式 slot header 隐藏（让位 TopBar 表达标题）。 */
  hideSlotHeader?: boolean;
  className?: string;
}

export function SimSceneGrid(props: SimSceneGridProps) {
  const {
    scenes, layoutMode, focusedSceneCode, setFocusedSceneCode,
    sceneStates, attachScene, detachScene, onCaptureScene, hideSlotHeader, className,
  } = props;

  if (scenes.length === 0) {
    return <div className={cn('flex items-center justify-center text-sm text-muted-foreground', className)}>无场景</div>;
  }

  if (layoutMode === 'carousel') {
    return (
      <CarouselLayout
        scenes={scenes}
        sceneStates={sceneStates}
        attachScene={attachScene}
        detachScene={detachScene}
        {...(onCaptureScene ? { onCaptureScene } : {})}
        className={className}
      />
    );
  }

  if (layoutMode === 'focus') {
    const focusCode = focusedSceneCode ?? scenes[0]!.sceneCode;
    return (
      <FocusLayout
        scenes={scenes}
        focusedSceneCode={focusCode}
        setFocusedSceneCode={setFocusedSceneCode}
        sceneStates={sceneStates}
        attachScene={attachScene}
        detachScene={detachScene}
        {...(onCaptureScene ? { onCaptureScene } : {})}
        className={className}
      />
    );
  }

  return (
    <GridLayout
      scenes={scenes}
      sceneStates={sceneStates}
      attachScene={attachScene}
      detachScene={detachScene}
      onFocus={setFocusedSceneCode}
      {...(onCaptureScene ? { onCaptureScene } : {})}
      hideSlotHeader={Boolean(hideSlotHeader)}
      className={className}
    />
  );
}

// ────────────────────────────────────────────────────────
// grid
// ────────────────────────────────────────────────────────

function GridLayout(props: {
  scenes: readonly SimSceneConfig[];
  sceneStates: ReadonlyMap<string, RenderState>;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  onFocus: (sceneCode: string) => void;
  onCaptureScene?: (sceneCode: string) => void;
  hideSlotHeader: boolean;
  className?: string;
}) {
  const { scenes, sceneStates, attachScene, detachScene, onFocus, onCaptureScene, hideSlotHeader, className } = props;

  const gridClass = useMemo(() => {
    const n = scenes.length;
    if (n === 1) return 'grid grid-cols-1';
    if (n === 2) return 'grid grid-cols-2';
    if (n === 3) return 'grid grid-cols-3';
    if (n === 4) return 'grid grid-cols-2 grid-rows-2';
    return 'grid grid-cols-3';
  }, [scenes.length]);

  return (
    <div className={cn('h-full gap-2 p-2', gridClass, className)}>
      {scenes.map(s => (
        <SimSceneSlot
          key={s.sceneCode}
          sceneCode={s.sceneCode}
          title={s.title}
          state={sceneStates.get(s.sceneCode) ?? null}
          attachScene={attachScene}
          detachScene={detachScene}
          {...(scenes.length > 1 ? { onFocus } : {})}
          {...(onCaptureScene ? { onCapture: onCaptureScene } : {})}
          hideHeader={hideSlotHeader && scenes.length === 1}
        />
      ))}
    </div>
  );
}

// ────────────────────────────────────────────────────────
// focus
// ────────────────────────────────────────────────────────

function FocusLayout(props: {
  scenes: readonly SimSceneConfig[];
  focusedSceneCode: string;
  setFocusedSceneCode: (sceneCode: string) => void;
  sceneStates: ReadonlyMap<string, RenderState>;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  onCaptureScene?: (sceneCode: string) => void;
  className?: string;
}) {
  const { scenes, focusedSceneCode, setFocusedSceneCode, sceneStates, attachScene, detachScene, onCaptureScene, className } = props;

  const focused = scenes.find(s => s.sceneCode === focusedSceneCode) ?? scenes[0]!;
  const others = scenes.filter(s => s.sceneCode !== focused.sceneCode);

  return (
    <div className={cn('flex h-full gap-2 p-2', className)}>
      <div className="flex-1">
        <SimSceneSlot
          sceneCode={focused.sceneCode}
          title={focused.title}
          state={sceneStates.get(focused.sceneCode) ?? null}
          attachScene={attachScene}
          detachScene={detachScene}
          {...(onCaptureScene ? { onCapture: onCaptureScene } : {})}
        />
      </div>
      <aside className="flex w-[160px] flex-col gap-2 overflow-y-auto">
        <Button
          size="sm"
          variant="outline"
          onClick={() => setFocusedSceneCode(scenes[0]!.sceneCode)}
          className="h-7 gap-1 text-xs"
        >
          <LayoutGrid className="h-3 w-3" /> 返回网格
        </Button>
        {others.map(s => (
          <button
            key={s.sceneCode}
            type="button"
            onClick={() => setFocusedSceneCode(s.sceneCode)}
            className="flex flex-col gap-1 rounded border border-border bg-card p-1 text-left transition hover:border-primary"
          >
            <div className="aspect-[5/3] w-full overflow-hidden rounded">
              <SimSceneSlot
                sceneCode={s.sceneCode}
                title=""
                state={sceneStates.get(s.sceneCode) ?? null}
                attachScene={attachScene}
                detachScene={detachScene}
                hideHeader
              />
            </div>
            <span className="truncate text-[10px] text-muted-foreground">{s.title}</span>
          </button>
        ))}
      </aside>
    </div>
  );
}

// ────────────────────────────────────────────────────────
// carousel
// ────────────────────────────────────────────────────────

function CarouselLayout(props: {
  scenes: readonly SimSceneConfig[];
  sceneStates: ReadonlyMap<string, RenderState>;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  onCaptureScene?: (sceneCode: string) => void;
  className?: string;
}) {
  const { scenes, sceneStates, attachScene, detachScene, onCaptureScene, className } = props;
  const [index, setIndex] = useState(0);

  const cur = scenes[Math.min(index, scenes.length - 1)]!;
  const prev = () => setIndex(i => (i - 1 + scenes.length) % scenes.length);
  const next = () => setIndex(i => (i + 1) % scenes.length);

  return (
    <div className={cn('flex h-full flex-col items-stretch gap-2 p-2', className)}>
      <div className="flex flex-1 items-stretch gap-2">
        <Button size="sm" variant="ghost" onClick={prev} className="h-auto self-stretch px-1"><ChevronLeft className="h-5 w-5" /></Button>
        <div className="flex-1">
          <SimSceneSlot
            sceneCode={cur.sceneCode}
            title={cur.title}
            state={sceneStates.get(cur.sceneCode) ?? null}
            attachScene={attachScene}
            detachScene={detachScene}
            {...(onCaptureScene ? { onCapture: onCaptureScene } : {})}
          />
        </div>
        <Button size="sm" variant="ghost" onClick={next} className="h-auto self-stretch px-1"><ChevronRight className="h-5 w-5" /></Button>
      </div>
      <div className="flex justify-center gap-1">
        {scenes.map((s, i) => (
          <button
            key={s.sceneCode}
            type="button"
            aria-label={s.title}
            onClick={() => setIndex(i)}
            className={cn(
              'h-2 w-2 rounded-full transition-colors',
              i === index ? 'bg-primary' : 'bg-muted',
            )}
          />
        ))}
      </div>
      <p className="text-center text-[11px] text-muted-foreground">{index + 1} / {scenes.length} · {cur.title}</p>
    </div>
  );
}
