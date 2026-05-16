'use client';

/**
 * SimControlBar — 时间控件栏（06.2 §四）。
 *
 * 三模式控件清单：
 *   process    : ▶ ⏸ ⏭ ⏮ + 速度切换 + ⟲
 *   continuous : ⏸/恢复 + 速度切换
 *   reactive   : 不显示（操作即响应）
 *
 * ⏮ 按钮可见性见 §4.2：仅 single + process 启用，B/C/D 置灰带 tooltip。
 */

import { Lock, Pause, Play, RotateCcw, SkipBack, SkipForward, Zap } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';
import { SPEED_OPTIONS, stepBackVisibility } from '@/hooks/useSimMode';
import type { SimMode, SimTimeControlMode } from '@/types/experiment';

export interface SimControlBarProps {
  mode: SimMode;
  timeMode: SimTimeControlMode;
  isRunning: boolean;
  currentSpeed: number;
  currentTick: number;
  /** C 联动 / D 混合时显示徽章。 */
  linkBadgeText?: string;
  onPlay: () => void;
  onPause: () => void;
  onStep: () => void;
  onStepBack: () => void;
  onReset: () => void;
  onSpeedChange: (speed: number) => void;
  className?: string;
}

export function SimControlBar(props: SimControlBarProps) {
  const {
    mode, timeMode, isRunning, currentSpeed, currentTick, linkBadgeText,
    onPlay, onPause, onStep, onStepBack, onReset, onSpeedChange, className,
  } = props;

  // reactive 模式整段不渲染（06.2 §4.1 / 设计稿 ⑤）；父级在该模式下应直接跳过本组件。
  if (timeMode === 'reactive') return null;

  const stepBack = stepBackVisibility(mode, timeMode);
  const supportsStep = timeMode === 'process'; // continuous 不支持单步前进

  return (
    <TooltipProvider>
      <div className={cn(
        'flex h-10 items-center gap-2 border-t border-border bg-card px-3',
        className,
      )}>
        {/* 播放 / 暂停 */}
        {isRunning ? (
          <Button size="sm" variant="outline" onClick={onPause} className="h-7 gap-1 text-xs">
            <Pause className="h-3.5 w-3.5" /> 暂停
          </Button>
        ) : (
          <Button size="sm" variant="primary" onClick={onPlay} className="h-7 gap-1 text-xs">
            <Play className="h-3.5 w-3.5" /> 播放
          </Button>
        )}

        {/* 单步前进（仅 process） */}
        {supportsStep && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button size="sm" variant="outline" onClick={onStep} className="h-7 w-7 p-0">
                <SkipForward className="h-3.5 w-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>单步前进</TooltipContent>
          </Tooltip>
        )}

        {/* 单步回退 */}
        {stepBack.visible && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!stepBack.enabled}
                  onClick={() => stepBack.enabled && onStepBack()}
                  className="h-7 w-7 p-0"
                >
                  <SkipBack className="h-3.5 w-3.5" />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{stepBack.reason ?? '单步回退'}</TooltipContent>
          </Tooltip>
        )}

        {/* 速度切换 */}
        <div className="ml-2 flex items-center gap-1 text-xs text-muted-foreground">
          <span>速度</span>
          {SPEED_OPTIONS.map(opt => (
            <Button
              key={opt.value}
              size="sm"
              variant={Math.abs(currentSpeed - opt.value) < 1e-6 ? 'primary' : 'outline'}
              onClick={() => onSpeedChange(opt.value)}
              className="h-6 px-2 text-xs"
            >
              {opt.label}
            </Button>
          ))}
        </div>

        {/* 重置 */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button size="sm" variant="outline" onClick={onReset} className="ml-2 h-7 w-7 p-0">
              <RotateCcw className="h-3.5 w-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>重置</TooltipContent>
        </Tooltip>

        {/* 右侧 Tick + 徽章；联动模式显示"时钟同步锁" */}
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          {mode === 'linkage' && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="outline" className="gap-1 border-purple-400/40 text-purple-300">
                  <Lock className="h-3 w-3" /> 时钟同步
                </Badge>
              </TooltipTrigger>
              <TooltipContent>联动组内速率/暂停/重置全员同步</TooltipContent>
            </Tooltip>
          )}
          <span>Tick {currentTick}</span>
          {linkBadgeText && (
            <Badge variant="secondary" className="gap-1">
              <Zap className="h-3 w-3" />
              {linkBadgeText}
            </Badge>
          )}
        </div>
      </div>
    </TooltipProvider>
  );
}
