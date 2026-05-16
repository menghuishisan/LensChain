'use client';

/**
 * SimTopBar — 顶部面包屑与工具栏（06.2 §1.1, 1.2）。
 *
 * 始终显示。工具按钮（均为 Lucide 图标）：Camera 截图 / Maximize2 全屏 / Settings2 教师干预。
 * 教师干预按钮仅 isTeacher 时显示。紧凑模式会多一个 PanelRight 按钮用于弹出 sidebar。
 */

import { Camera, Maximize2, Minimize2, PanelRight, Settings2 } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { cn } from '@/lib/utils';

export interface SimTopBarProps {
  /** 面包屑分段，从左至右。例：['实验 X', '仿真', 'PoW 挖矿']。 */
  breadcrumb: readonly string[];
  isFullscreen: boolean;
  isTeacher: boolean;
  onCapture: () => void;
  onToggleFullscreen: () => void;
  onOpenIntervention?: () => void;
  /** 紧凑模式下切换 sidebar popover；仅在紧凑模式下传入。 */
  onToggleSidebar?: () => void;
  className?: string;
}

export function SimTopBar(props: SimTopBarProps) {
  const { breadcrumb, isFullscreen, isTeacher, onCapture, onToggleFullscreen, onOpenIntervention, onToggleSidebar, className } = props;

  return (
    <TooltipProvider>
      <div className={cn(
        'flex h-12 items-center justify-between border-b border-border bg-card px-3',
        className,
      )}>
        <div className="flex items-center gap-1 text-sm text-foreground">
          {breadcrumb.map((seg, i) => (
            <span key={i} className="flex items-center gap-1">
              {i > 0 && <span className="text-muted-foreground">/</span>}
              <span className={cn(
                i === breadcrumb.length - 1 ? 'font-semibold' : 'text-muted-foreground',
              )}>{seg}</span>
            </span>
          ))}
        </div>

        <div className="flex items-center gap-1">
          {onToggleSidebar && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button size="sm" variant="ghost" onClick={onToggleSidebar} className="h-8 w-8 p-0">
                  <PanelRight className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>侧栏面板</TooltipContent>
            </Tooltip>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button size="sm" variant="ghost" onClick={onCapture} className="h-8 w-8 p-0">
                <Camera className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>截图当前画布</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button size="sm" variant="ghost" onClick={onToggleFullscreen} className="h-8 w-8 p-0">
                {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{isFullscreen ? '退出全屏' : '全屏'}</TooltipContent>
          </Tooltip>

          {isTeacher && onOpenIntervention && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button size="sm" variant="ghost" onClick={onOpenIntervention} className="h-8 w-8 p-0">
                  <Settings2 className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>教师干预</TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>
    </TooltipProvider>
  );
}
