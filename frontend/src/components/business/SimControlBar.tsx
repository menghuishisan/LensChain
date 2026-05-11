'use client';

// SimControlBar.tsx
// SimEngine 时间控制栏（06.2 §四）。
// 根据 TimeControlMode 动态渲染按钮集，含速度切换、Tick 显示和联动徽章。

import { useState } from 'react';
import { Pause, Play, RotateCcw, SkipBack, SkipForward } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { cn } from '@/lib/utils';
import { SPEED_OPTIONS } from '@/hooks/useSimMode';
import type { SimControlCommand, SimMode, SimTimeControlMode } from '@/types/experiment';

export interface SimControlBarProps {
  timeControlMode: SimTimeControlMode;
  mode: SimMode;
  tick: number;
  playing: boolean;
  speed: number;
  connected: boolean;
  canStepBack: boolean;
  stepBackTooltip: string | null;
  onControl: (command: SimControlCommand, value?: number) => void;
  onStepBack: () => void;
  className?: string;
}

/**
 * SimControlBar 时间控制栏（h-10，06.2 §1.2 ControlBar 区域）。
 * reactive 模式不渲染（showTimeControl=false 时上层不挂载本组件）。
 */
export function SimControlBar({
  timeControlMode,
  mode,
  tick,
  playing,
  speed,
  connected,
  canStepBack,
  stepBackTooltip,
  onControl,
  onStepBack,
  className,
}: SimControlBarProps) {
  const [currentSpeed, setCurrentSpeed] = useState(speed);

  const handleSpeedChange = (s: number) => {
    setCurrentSpeed(s);
    onControl('set_speed', s);
  };

  return (
    <TooltipProvider delayDuration={200}>
    <div className={cn('flex items-center gap-2 h-10 px-4 border-t bg-muted/30', className)}>
      {timeControlMode === 'process' && (
        <>
          {playing ? (
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('pause')}>
              <Pause className="h-3.5 w-3.5" />
              暂停
            </Button>
          ) : (
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('play')}>
              <Play className="h-3.5 w-3.5" />
              播放
            </Button>
          )}
          <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('step')}>
            <SkipForward className="h-3.5 w-3.5" />
            单步
          </Button>
          {canStepBack ? (
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={onStepBack}>
              <SkipBack className="h-3.5 w-3.5" />
              回退
            </Button>
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled>
                    <SkipBack className="h-3.5 w-3.5" />
                    回退
                  </Button>
                </span>
              </TooltipTrigger>
              {stepBackTooltip && <TooltipContent>{stepBackTooltip}</TooltipContent>}
            </Tooltip>
          )}
          <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('reset')}>
            <RotateCcw className="h-3.5 w-3.5" />
            重置
          </Button>
        </>
      )}

      {timeControlMode === 'continuous' && (
        <>
          {playing ? (
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('pause')}>
              <Pause className="h-3.5 w-3.5" />
              暂停
            </Button>
          ) : (
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs" disabled={!connected} onClick={() => onControl('resume')}>
              <Play className="h-3.5 w-3.5" />
              恢复
            </Button>
          )}
        </>
      )}

      {/* 速度切换（process / continuous 共用）。 */}
      <div className="flex items-center gap-1 ml-2">
        <span className="text-xs text-muted-foreground">速度:</span>
        {SPEED_OPTIONS.map((s) => (
          <Button
            key={s}
            variant={currentSpeed === s ? 'secondary' : 'ghost'}
            size="sm"
            className="h-6 px-2 text-xs"
            disabled={!connected}
            onClick={() => handleSpeedChange(s)}
          >
            {s}x
          </Button>
        ))}
      </div>

      {/* Tick 显示 + 联动徽章。 */}
      <div className="ml-auto flex items-center gap-2">
        {mode === 'linkage' && (
          <span className="inline-flex items-center gap-1 rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-300">
            联动同步
          </span>
        )}
        <span className="text-xs text-muted-foreground">第 {tick} 步</span>
      </div>
    </div>
    </TooltipProvider>
  );
}
