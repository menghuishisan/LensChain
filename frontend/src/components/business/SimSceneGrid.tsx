'use client';

// SimSceneGrid.tsx
// 多场景响应式布局（06.2 §三）。
// 三种布局：
//   - grid：1=单列全宽；2=横排 2 列；3=横排 3 列；4=2×2；≥5 强制降级 focus；
//   - focus：左主右缩略图列表（width 200px），缩略图 100×60；主区 ≥ 720×480；
//   - carousel：单画布 + 左右切换 + 底部指示器。
// 每个 slot 委托给 SimSceneSlot 组件渲染（场景头 + Canvas + 可选 InteractionForm）。
// 文档 §3.5 尺寸约束在本组件统一控制：min-h（480/360）、min-w（800/600/500）。

import { useCallback, useEffect, useState } from 'react';
import { ChevronLeft, ChevronRight, LayoutGrid } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/utils';
import { SimSceneSlot, type SceneSlotConfig } from '@/components/business/SimSceneSlot';
import type { JsonObject, SimLayoutMode } from '@/types/experiment';

export type { SceneSlotConfig };

export interface SimSceneGridProps {
  scenes: SceneSlotConfig[];
  layout: SimLayoutMode;
  instanceID?: string;
  userRole?: string;
  connected: boolean;
  /**
   * 是否启用每场景独立的 InteractionForm（文档 §5.6）。
   * 单场景模式由 SimEnginePanel 显式控制（A 单仿真：在画布下方折叠面板，本组件渲染）。
   */
  enableInteractionForm?: boolean;
  attachScene: (code: string, cat: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (code: string) => void;
  redrawScene: (code: string) => void;
  captureScene: (code: string) => string | null;
  onSubmitInteraction: (sceneCode: string, actionCode: string, params: JsonObject) => void;
  /** 父级请求强制切换布局（用于 focus 模式"返回 grid"按钮触发外部状态更新）。 */
  onLayoutChange?: (layout: SimLayoutMode) => void;
  className?: string;
}

/**
 * 按场景数返回 grid 模式下单元格的最小高度（文档 §3.5）。
 * 1/2/3 场景：480；2×2 网格（4 场景）：360。
 */
function gridCellMinHeight(sceneCount: number): number {
  return sceneCount <= 3 ? 480 : 360;
}

/**
 * 按场景数返回 grid 模式下单元格的最小宽度（文档 §3.5）。
 * 1 场景：800；2 场景：600；3 场景：500；4 场景（2×2）：500。
 */
function gridCellMinWidth(sceneCount: number): number {
  if (sceneCount <= 1) return 800;
  if (sceneCount === 2) return 600;
  return 500;
}

/**
 * SimSceneGrid 管理多场景画布的排列（06.2 §3.1-3.5）。
 */
export function SimSceneGrid({
  scenes,
  layout,
  instanceID,
  userRole,
  connected,
  enableInteractionForm = true,
  attachScene,
  detachScene,
  redrawScene,
  captureScene,
  onSubmitInteraction,
  onLayoutChange,
  className,
}: SimSceneGridProps) {
  const [focusIndex, setFocusIndex] = useState(0);

  // 当场景数量变化导致索引越界时重置。
  useEffect(() => {
    if (focusIndex >= scenes.length) setFocusIndex(0);
  }, [scenes.length, focusIndex]);

  const renderSlot = useCallback(
    (
      scene: SceneSlotConfig,
      opts: { minHeight: number; minWidth?: number; showForm: boolean; className?: string },
    ) => (
      <SimSceneSlot
        key={scene.sceneCode}
        scene={scene}
        instanceID={instanceID}
        userRole={userRole}
        connected={connected}
        minHeight={opts.minHeight}
        showInteractionForm={enableInteractionForm && opts.showForm}
        attachScene={attachScene}
        detachScene={detachScene}
        redrawScene={redrawScene}
        captureScene={captureScene}
        onSubmitInteraction={onSubmitInteraction}
        className={opts.className}
        style={opts.minWidth ? { minWidth: `${opts.minWidth}px` } : undefined}
      />
    ),
    [
      instanceID,
      userRole,
      connected,
      enableInteractionForm,
      attachScene,
      detachScene,
      redrawScene,
      captureScene,
      onSubmitInteraction,
    ],
  );

  if (scenes.length === 0) return null;

  // ─── grid 模式 ─────────────────────────────────────
  // 文档 §3.1：1 全宽 / 2 横排 / 3 横排 / 4 → 2×2；≥5 由 SimEnginePanel 在 defaultLayout 降级 focus。
  if (layout === 'grid') {
    const colsClass =
      scenes.length === 1
        ? 'grid-cols-1'
        : scenes.length === 2
          ? 'grid-cols-2'
          : scenes.length === 3
            ? 'grid-cols-3'
            : 'grid-cols-2'; // 4 场景 → 2×2
    const cellMinH = gridCellMinHeight(scenes.length);
    const cellMinW = gridCellMinWidth(scenes.length);
    // 文档 §3.5 规定每单元格的最小高度（480 / 360）是硬性下限，而不是"等分父高度后的妥协值"。
    // 因此显式使用 auto-rows-[Npx] 让行高按 cell 最小高度起算；当父容器高度不足时由外层
    // overflow-auto 接管纵向滚动，而不是把画布压扁导致内容裁剪。
    // arbitrary value 必须是字面量字符串，Tailwind JIT 才能识别，所以用静态映射表。
    const autoRowsClass = scenes.length === 4 ? 'auto-rows-[360px]' : 'auto-rows-[480px]';
    return (
      <div className={cn('grid gap-4 w-full', colsClass, autoRowsClass, className)}>
        {scenes.map((s) =>
          renderSlot(s, {
            minHeight: cellMinH,
            minWidth: cellMinW,
            showForm: true,
            // min-h-0 解除 flex 子项的默认 min-content；min-w 通过 inline style 传入。
            className: 'min-h-0',
          }),
        )}
      </div>
    );
  }

  // ─── focus 模式 ────────────────────────────────────
  // 文档 §3.2：主区 ≥ 720×480；缩略图列宽 200px、单个缩略图 100×60；
  // 触发跨画布弧线 M8 时缩略图仍保持因果连线（视觉简化，本组件用文字 + 状态点表示）。
  if (layout === 'focus') {
    const focused = scenes[focusIndex] ?? scenes[0];
    return (
      <div className={cn('flex gap-3 h-full', className)}>
        {/* 主聚焦区 */}
        <div className="flex-1 min-w-[720px]">
          {renderSlot(focused, {
            minHeight: 480,
            showForm: true,
            className: 'h-full',
          })}
        </div>

        {/* 右侧缩略图列表 */}
        {scenes.length > 1 && (
          <div className="w-[200px] shrink-0 flex flex-col gap-2">
            {/* 返回 grid 按钮（文档 §3.2：顶部 [返回 grid] 按钮） */}
            {onLayoutChange && scenes.length <= 4 && (
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-xs gap-1 justify-start"
                onClick={() => onLayoutChange('grid')}
              >
                <LayoutGrid className="h-3.5 w-3.5" />返回 grid
              </Button>
            )}
            <div className="flex-1 overflow-y-auto flex flex-col gap-2">
              {scenes.map((s, i) => (
                <button
                  key={s.sceneCode}
                  type="button"
                  className={cn(
                    'rounded-lg border p-2 text-left transition-colors hover:bg-accent shrink-0',
                    // 缩略图尺寸 100×60（文档 §3.2 / §3.5）
                    'w-full min-h-[60px]',
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
          </div>
        )}
      </div>
    );
  }

  // ─── carousel 模式 ─────────────────────────────────
  // 文档 §3.3：当前展示场景下方显示 InteractionForm；左右箭头切换。
  const current = scenes[focusIndex] ?? scenes[0];
  return (
    <div className={cn('relative h-full', className)}>
      {renderSlot(current, {
        minHeight: 480,
        showForm: true,
        className: 'h-full w-full',
      })}

      {scenes.length > 1 && (
        <>
          <Button
            variant="outline"
            size="sm"
            className="absolute left-2 top-1/2 -translate-y-1/2 h-8 w-8 p-0 rounded-full opacity-80 hover:opacity-100 z-10"
            onClick={() => setFocusIndex((focusIndex - 1 + scenes.length) % scenes.length)}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8 p-0 rounded-full opacity-80 hover:opacity-100 z-10"
            onClick={() => setFocusIndex((focusIndex + 1) % scenes.length)}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex gap-1 z-10">
            {scenes.map((s, i) => (
              <button
                key={s.sceneCode}
                type="button"
                aria-label={`切换到 ${s.title}`}
                className={cn(
                  'h-1.5 w-6 rounded-full transition-colors',
                  i === focusIndex ? 'bg-primary' : 'bg-muted-foreground/30',
                )}
                onClick={() => setFocusIndex(i)}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}
