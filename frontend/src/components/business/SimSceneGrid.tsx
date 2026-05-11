'use client';

// SimSceneGrid.tsx
// 多场景响应式布局（06.2 §三）。
// 支持三种布局模式：grid（网格平铺）、focus（焦点+侧边缩略图）、carousel（轮播）。
// 每个场景 slot 内部挂载 SimSceneCanvas + 场景头部信息栏。

import { useCallback, useState } from 'react';
import { Camera, ChevronLeft, ChevronRight, Maximize2, Minimize2 } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/utils';
import { SimSceneCanvas } from '@/components/business/SimSceneCanvas';
import type { SimLayoutMode } from '@/types/experiment';

/** 场景配置项。 */
export interface SceneSlotConfig {
  sceneCode: string;
  category: string;
  title: string;
  hasState: boolean;
}

export interface SimSceneGridProps {
  scenes: SceneSlotConfig[];
  layout: SimLayoutMode;
  onLayoutChange?: (layout: SimLayoutMode) => void;
  attachScene: (code: string, cat: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (code: string) => void;
  redrawScene: (code: string) => void;
  captureScene: (code: string) => string | null;
  className?: string;
}

/**
 * SimSceneGrid 管理多场景画布的排列（06.2 §3.1-3.3）。
 * grid: sceneCount≤4 时 2-col 网格
 * focus: 一主一副，侧边缩略图列表切换焦点
 * carousel: 单场景轮播，左右箭头切换
 */
export function SimSceneGrid({
  scenes,
  layout,
  attachScene,
  detachScene,
  redrawScene,
  captureScene,
  className,
}: SimSceneGridProps) {
  const [focusIndex, setFocusIndex] = useState(0);
  const [fullscreenCode, setFullscreenCode] = useState<string | null>(null);

  const handleScreenshot = useCallback(
    (code: string) => {
      const dataUrl = captureScene(code);
      if (dataUrl) {
        const a = document.createElement('a');
        a.href = dataUrl;
        a.download = `${code}-screenshot.png`;
        a.click();
      }
    },
    [captureScene],
  );

  if (scenes.length === 0) return null;

  const renderSceneSlot = (scene: SceneSlotConfig, sizing: string) => {
    const isFullscreen = fullscreenCode === scene.sceneCode;
    return (
      <div
        key={scene.sceneCode}
        className={cn(
          'rounded-xl border bg-card overflow-hidden flex flex-col',
          isFullscreen && 'fixed inset-4 z-50 bg-background',
          sizing,
        )}
      >
        {/* 场景头部 */}
        <div className="flex items-center justify-between px-3 py-2 border-b">
          <div className="min-w-0">
            <p className="text-sm font-medium truncate">{scene.title}</p>
            <p className="text-xs text-muted-foreground">{scene.category}</p>
          </div>
          <div className="flex gap-1 shrink-0">
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => handleScreenshot(scene.sceneCode)}>
              <Camera className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0"
              onClick={() => setFullscreenCode(isFullscreen ? null : scene.sceneCode)}
            >
              {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </div>

        {/* Canvas 区域 */}
        <SimSceneCanvas
          sceneCode={scene.sceneCode}
          category={scene.category}
          attachScene={attachScene}
          detachScene={detachScene}
          redrawScene={redrawScene}
          hasState={scene.hasState}
          className="flex-1"
        />
      </div>
    );
  };

  // ─── grid 模式 ─────────────────────────────────────
  // 文档 06.2 §3.1：1 场景全宽、2 场景 1×2、3 场景 1×3、4 场景 2×2。
  // 注意：grid 容器宽度由父级（已扣除外层导航 + 侧栏）决定，与 window 宽度不一致，
  // 所以这里不能用 `lg:` 这种 viewport 断点（会因为内层容器达不到 1024px 而退回单列堆叠）。
  // 直接给出场景数对应的列数即可。
  if (layout === 'grid') {
    const colsClass =
      scenes.length === 1
        ? 'grid-cols-1'
        : scenes.length === 2
          ? 'grid-cols-2'
          : scenes.length === 3
            ? 'grid-cols-3'
            : 'grid-cols-2'; // 4 场景 → 2×2（行由 grid 自动换行）
    return <div className={cn('grid gap-4', colsClass, className)}>{scenes.map((s) => renderSceneSlot(s, ''))}</div>;
  }

  // ─── focus 模式 ────────────────────────────────────
  if (layout === 'focus') {
    const focused = scenes[focusIndex] ?? scenes[0];
    return (
      <div className={cn('flex gap-3', className)}>
        {/* 主画布 */}
        <div className="flex-1">{renderSceneSlot(focused, 'h-full')}</div>

        {/* 侧边缩略图列表 */}
        {scenes.length > 1 && (
          <div className="w-40 flex flex-col gap-2 shrink-0">
            {scenes.map((s, i) => (
              <button
                key={s.sceneCode}
                className={cn(
                  'rounded-lg border p-2 text-left transition-colors hover:bg-accent',
                  i === focusIndex && 'border-primary bg-primary/5',
                )}
                onClick={() => setFocusIndex(i)}
              >
                <p className="text-xs font-medium truncate">{s.title}</p>
                <Badge variant="outline" className="mt-1 text-[10px]">
                  {s.category}
                </Badge>
              </button>
            ))}
          </div>
        )}
      </div>
    );
  }

  // ─── carousel 模式 ─────────────────────────────────
  const current = scenes[focusIndex] ?? scenes[0];
  return (
    <div className={cn('relative', className)}>
      {renderSceneSlot(current, 'w-full')}

      {scenes.length > 1 && (
        <>
          <Button
            variant="outline"
            size="sm"
            className="absolute left-2 top-1/2 -translate-y-1/2 h-8 w-8 p-0 rounded-full opacity-80 hover:opacity-100"
            onClick={() => setFocusIndex((focusIndex - 1 + scenes.length) % scenes.length)}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8 p-0 rounded-full opacity-80 hover:opacity-100"
            onClick={() => setFocusIndex((focusIndex + 1) % scenes.length)}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex gap-1">
            {scenes.map((s, i) => (
              <button
                key={s.sceneCode}
                className={cn('h-1.5 w-6 rounded-full transition-colors', i === focusIndex ? 'bg-primary' : 'bg-muted-foreground/30')}
                onClick={() => setFocusIndex(i)}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}
