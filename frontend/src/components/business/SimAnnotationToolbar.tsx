'use client';

// SimAnnotationToolbar.tsx
// 教师标注工具栏（06.2 §7.1）。
// 左侧浮动竖排，48px 宽。含箭头/圆圈/方框/自由笔/文字/橡皮擦/清空。
// 仅教师可见，TopBar [标注] 按钮 toggle。
// 全班学生同步可见（通过 annotation 原语下发）。

import { useState, useCallback } from 'react';
import { ArrowUpRight, Circle, Eraser, Pencil, Square, Trash2, Type } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/Tooltip';
import { cn } from '@/lib/utils';

/** 标注工具类型。 */
export type AnnotationTool = 'arrow' | 'circle' | 'rect' | 'freehand' | 'text' | 'eraser';

export interface SimAnnotationToolbarProps {
  /** 选中的工具通过回调通知父组件（SimEnginePanel）以便在 canvas overlay 上执行绑定。 */
  onToolChange: (tool: AnnotationTool | null) => void;
  /** 清空所有标注。 */
  onClearAll: () => void;
  className?: string;
}

const TOOLS: { tool: AnnotationTool; icon: React.ElementType; label: string }[] = [
  { tool: 'arrow', icon: ArrowUpRight, label: '箭头' },
  { tool: 'circle', icon: Circle, label: '圆圈' },
  { tool: 'rect', icon: Square, label: '方框' },
  { tool: 'freehand', icon: Pencil, label: '自由笔' },
  { tool: 'text', icon: Type, label: '文字' },
  { tool: 'eraser', icon: Eraser, label: '橡皮擦' },
];

/**
 * SimAnnotationToolbar 教师标注工具栏（06.2 §7.1）。
 * 左侧浮动竖排，宽 48px。通过 onToolChange 回调驱动 canvas overlay 标注逻辑。
 */
export function SimAnnotationToolbar({ onToolChange, onClearAll, className }: SimAnnotationToolbarProps) {
  const [activeTool, setActiveTool] = useState<AnnotationTool | null>(null);

  const selectTool = useCallback(
    (tool: AnnotationTool) => {
      const next = activeTool === tool ? null : tool;
      setActiveTool(next);
      onToolChange(next);
    },
    [activeTool, onToolChange],
  );

  const clearAll = useCallback(() => {
    setActiveTool(null);
    onToolChange(null);
    onClearAll();
  }, [onToolChange, onClearAll]);

  return (
    <div
      className={cn(
        'absolute left-2 top-1/2 -translate-y-1/2 z-30 flex w-12 flex-col items-center gap-1 rounded-xl border border-border bg-card/95 p-1.5 shadow-lg backdrop-blur-sm',
        className,
      )}
    >
      {TOOLS.map(({ tool, icon: Icon, label }) => (
        <Tooltip key={tool}>
          <TooltipTrigger asChild>
            <Button
              variant={activeTool === tool ? 'secondary' : 'ghost'}
              size="sm"
              className={cn('h-9 w-9 p-0', activeTool === tool && 'ring-2 ring-primary/40')}
              onClick={() => selectTool(tool)}
            >
              <Icon className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right" className="text-xs">
            {label}
          </TooltipContent>
        </Tooltip>
      ))}

      {/* 清空全部 */}
      <div className="my-0.5 h-px w-6 bg-border" />
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="sm" className="h-9 w-9 p-0 text-destructive hover:text-destructive" onClick={clearAll}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right" className="text-xs">
          清空标注
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
