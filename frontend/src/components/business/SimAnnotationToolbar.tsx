'use client';

/**
 * SimAnnotationToolbar — 教师标注工具栏（06.2 §7.1）。
 *
 * 浮动 width-48px，左侧主画布外。仅教师可见，由顶部 [✏️] 切换 toggle。
 * 工具：箭头 / 圆圈 / 方框 / 自由笔 / 文字 / 橡皮擦 / 清空。
 *
 * 本组件只发出"已选中工具"事件；具体在 canvas 上的绘制 / 提交 annotation
 * 原语由上层（SimSceneSlot）配合 sendAction('add_annotation') 完成。
 */

import { ArrowUpRight, Circle, Eraser, PencilLine, Square, Type, X } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { cn } from '@/lib/utils';

export type AnnotationTool = 'arrow' | 'circle' | 'box' | 'pen' | 'text' | 'eraser';

export interface SimAnnotationToolbarProps {
  visible: boolean;
  currentTool: AnnotationTool | null;
  onSelectTool: (tool: AnnotationTool | null) => void;
  onClear: () => void;
  className?: string;
}

const TOOL_ITEMS: { tool: AnnotationTool; label: string; Icon: typeof ArrowUpRight }[] = [
  { tool: 'arrow', label: '箭头', Icon: ArrowUpRight },
  { tool: 'circle', label: '圆圈', Icon: Circle },
  { tool: 'box', label: '方框', Icon: Square },
  { tool: 'pen', label: '自由笔', Icon: PencilLine },
  { tool: 'text', label: '文字', Icon: Type },
  { tool: 'eraser', label: '橡皮擦', Icon: Eraser },
];

export function SimAnnotationToolbar(props: SimAnnotationToolbarProps) {
  const { visible, currentTool, onSelectTool, onClear, className } = props;

  if (!visible) return null;

  return (
    <TooltipProvider>
      <div className={cn(
        'flex w-12 flex-col items-center gap-1 rounded-md border border-border bg-card p-1 shadow-md',
        className,
      )}>
        {TOOL_ITEMS.map(({ tool, label, Icon }) => (
          <Tooltip key={tool}>
            <TooltipTrigger asChild>
              <Button
                size="sm"
                variant={currentTool === tool ? 'primary' : 'ghost'}
                onClick={() => onSelectTool(currentTool === tool ? null : tool)}
                className="h-8 w-8 p-0"
              >
                <Icon className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">{label}</TooltipContent>
          </Tooltip>
        ))}
        <div className="my-1 h-px w-6 bg-border" />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button size="sm" variant="ghost" onClick={onClear} className="h-8 w-8 p-0 text-destructive">
              <X className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">清空所有标注</TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}
