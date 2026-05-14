'use client';

// SimTopBar.tsx
// SimEngine 顶部工具栏（06.2 §1.1 TopBar 区域）。
// 包含场景标题、连接状态、模式 Badge、布局切换和全局操作按钮。

import { Camera, Grid2X2, Layers, LayoutGrid, Maximize2, Minimize2, Pencil, Settings2, Wifi, WifiOff } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/utils';
import type { SimLayoutMode, SimMode } from '@/types/experiment';

/** 模式 → 中文标签 + Badge 色彩。 */
const MODE_META: Record<SimMode, { label: string; className: string }> = {
  single: { label: '单场景', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' },
  comparison: { label: '对照模式', className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' },
  linkage: { label: '联动模式', className: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300' },
  hybrid: { label: '混合实验', className: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' },
};

/** 布局图标。 */
const LAYOUT_OPTIONS: { value: SimLayoutMode; icon: React.ElementType; label: string }[] = [
  { value: 'grid', icon: Grid2X2, label: '网格' },
  { value: 'focus', icon: Layers, label: '焦点' },
  { value: 'carousel', icon: LayoutGrid, label: '轮播' },
];

export interface SimTopBarProps {
  title?: string;
  mode: SimMode;
  layout: SimLayoutMode;
  connected: boolean;
  sceneCount: number;
  /** 当前是否处于浏览器全屏态（外部受控）。 */
  isFullscreen?: boolean;
  onLayoutChange: (layout: SimLayoutMode) => void;
  /** 顶部 📷：截图当前主场景。文档 §1.1。 */
  onScreenshot?: () => void;
  /** 顶部 ⛶：整个 SimEnginePanel 进入/退出浏览器全屏。文档 §1.1。 */
  onFullscreenToggle?: () => void;
  onIntervene?: () => void;
  onAnnotationToggle?: () => void;
  annotationActive?: boolean;
  className?: string;
}

/**
 * SimTopBar 渲染 SimEngine 面板顶部栏（06.2 §1.1）。
 */
export function SimTopBar({
  title = '可视化仿真',
  mode,
  layout,
  connected,
  sceneCount,
  isFullscreen = false,
  onLayoutChange,
  onScreenshot,
  onFullscreenToggle,
  onIntervene,
  onAnnotationToggle,
  annotationActive,
  className,
}: SimTopBarProps) {
  const modeMeta = MODE_META[mode];

  return (
    <div className={cn('flex items-center justify-between px-4 h-10 border-b bg-muted/30', className)}>
      <div className="flex items-center gap-2 min-w-0">
        <h3 className="text-sm font-medium truncate">{title}</h3>
        <Badge className={cn('text-[10px] px-1.5 py-0 h-5', modeMeta.className)}>
          {modeMeta.label}
        </Badge>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        {/* 布局切换（≥2 场景时显示） */}
        {sceneCount >= 2 && (
          <div className="flex items-center gap-0.5 rounded-md border p-0.5">
            {LAYOUT_OPTIONS.map(({ value, icon: Icon, label }) => (
              <Button
                key={value}
                variant={layout === value ? 'secondary' : 'ghost'}
                size="sm"
                className="h-6 w-6 p-0"
                title={label}
                onClick={() => onLayoutChange(value)}
              >
                <Icon className="h-3.5 w-3.5" />
              </Button>
            ))}
          </div>
        )}

        {/* §7.1 教师标注工具开关 */}
        {onAnnotationToggle && (
          <Button variant={annotationActive ? 'secondary' : 'outline'} size="sm" className="h-6 text-xs gap-1" onClick={onAnnotationToggle}>
            <Pencil className="h-3 w-3" />标注
          </Button>
        )}

        {/* §1.1 📷 截图主场景 */}
        {onScreenshot && (
          <Button variant="ghost" size="sm" className="h-6 w-6 p-0" title="截图主场景" onClick={onScreenshot}>
            <Camera className="h-3.5 w-3.5" />
          </Button>
        )}

        {/* §1.1 ⛶ 面板级全屏 */}
        {onFullscreenToggle && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            title={isFullscreen ? '退出全屏' : '面板全屏'}
            onClick={onFullscreenToggle}
          >
            {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </Button>
        )}

        {/* §7.2 教师干预按钮 */}
        {onIntervene && (
          <Button variant="outline" size="sm" className="h-6 text-xs gap-1" onClick={onIntervene}>
            <Settings2 className="h-3 w-3" />干预
          </Button>
        )}

        {/* 连接状态 */}
        <Badge variant={connected ? 'outline' : 'destructive'} className="text-xs gap-1 h-6">
          {connected ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
          {connected ? '已连接' : '未连接'}
        </Badge>
      </div>
    </div>
  );
}
