'use client';

// SimSidebar.tsx
// SimEngine 右侧折叠侧边栏（06.2 §1.1 Sidebar 区域）。
// 按顺序展示：实时指标、联动指示器、容器健康状态（混合模式）、微步骤时间线。

import { Activity, ArrowRight, Box, Link2, Timer } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { SimLinkIndicatorStatus, SimMicroStep, SimMode } from '@/types/experiment';

/** 指标条目。 */
export interface MetricItem {
  label: string;
  value: string | number;
}

/** 联动指示器条目。 */
export interface LinkIndicatorItem {
  linkGroup: string;
  status: SimLinkIndicatorStatus;
  label: string;
}

/** 容器状态条目。 */
export interface ContainerHealthItem {
  name: string;
  status: 'running' | 'starting' | 'error' | 'stopped';
  cpu?: string;
  memory?: string;
}

export interface SimSidebarProps {
  mode: SimMode;
  metrics: MetricItem[];
  linkIndicators: LinkIndicatorItem[];
  containerHealth: ContainerHealthItem[];
  microSteps: SimMicroStep[];
  activeMicroStepId?: string;
  collapsed?: boolean;
  onToggle?: () => void;
  className?: string;
}

const LINK_STATUS_STYLE: Record<SimLinkIndicatorStatus, string> = {
  idle: 'bg-gray-300 dark:bg-gray-600',
  active: 'bg-green-500 animate-pulse',
  recent: 'bg-amber-400',
};

const CONTAINER_STATUS_STYLE: Record<string, string> = {
  running: 'bg-green-500',
  starting: 'bg-amber-400 animate-pulse',
  error: 'bg-red-500',
  stopped: 'bg-gray-400',
};

/**
 * SimSidebar 右侧信息面板（06.2 §1.1）。
 * 在 collapsed 状态下缩为窄条（仅图标），展开时完整显示。
 */
export function SimSidebar({
  mode,
  metrics,
  linkIndicators,
  containerHealth,
  microSteps,
  activeMicroStepId,
  collapsed = false,
  onToggle,
  className,
}: SimSidebarProps) {
  if (collapsed) {
    return (
      <div className={cn('w-10 border-l flex flex-col items-center gap-3 py-3 cursor-pointer', className)} onClick={onToggle}>
        <Activity className="h-4 w-4 text-muted-foreground" />
        {linkIndicators.length > 0 && <Link2 className="h-4 w-4 text-muted-foreground" />}
        {containerHealth.length > 0 && <Box className="h-4 w-4 text-muted-foreground" />}
        {microSteps.length > 0 && <Timer className="h-4 w-4 text-muted-foreground" />}
      </div>
    );
  }

  return (
    <div className={cn('w-56 border-l overflow-y-auto', className)}>
      {/* 实时指标 */}
      {metrics.length > 0 && (
        <SidebarSection icon={Activity} title="实时指标">
          <div className="space-y-1">
            {metrics.map((m, i) => (
              <div key={i} className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">{m.label}</span>
                <span className="font-medium">{m.value}</span>
              </div>
            ))}
          </div>
        </SidebarSection>
      )}

      {/* 联动指示器（linkage 模式） */}
      {(mode === 'linkage' || mode === 'hybrid') && linkIndicators.length > 0 && (
        <SidebarSection icon={Link2} title="联动状态">
          <div className="space-y-1.5">
            {linkIndicators.map((li) => (
              <div key={li.linkGroup} className="flex items-center gap-2 text-xs">
                <span className={cn('h-2 w-2 rounded-full shrink-0', LINK_STATUS_STYLE[li.status])} />
                <span className="truncate">{li.label}</span>
              </div>
            ))}
          </div>
        </SidebarSection>
      )}

      {/* 容器健康（混合模式） */}
      {mode === 'hybrid' && containerHealth.length > 0 && (
        <SidebarSection icon={Box} title="容器状态">
          <div className="space-y-1.5">
            {containerHealth.map((c) => (
              <div key={c.name} className="flex items-center gap-2 text-xs">
                <span className={cn('h-2 w-2 rounded-full shrink-0', CONTAINER_STATUS_STYLE[c.status] ?? 'bg-gray-400')} />
                <span className="truncate flex-1">{c.name}</span>
                {c.cpu && <span className="text-muted-foreground">{c.cpu}</span>}
              </div>
            ))}
          </div>
        </SidebarSection>
      )}

      {/* 微步骤时间线 */}
      {microSteps.length > 0 && (
        <SidebarSection icon={Timer} title="微步骤">
          <div className="space-y-1">
            {microSteps.map((step) => {
              const isActive = step.id === activeMicroStepId;
              return (
                <div
                  key={step.id}
                  className={cn(
                    'flex items-center gap-2 rounded px-2 py-1 text-xs transition-colors',
                    isActive && 'bg-primary/10 text-primary font-medium',
                  )}
                >
                  <ArrowRight className={cn('h-3 w-3 shrink-0', isActive ? 'text-primary' : 'text-muted-foreground')} />
                  <span className="truncate">{step.label}</span>
                  <span className="ml-auto text-muted-foreground shrink-0">{step.duration_ms}ms</span>
                </div>
              );
            })}
          </div>
        </SidebarSection>
      )}
    </div>
  );
}

/** SidebarSection 通用侧边栏区块。 */
function SidebarSection({
  icon: Icon,
  title,
  children,
}: {
  icon: React.ElementType;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="border-b px-3 py-3">
      <div className="flex items-center gap-1.5 mb-2">
        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
      </div>
      {children}
    </div>
  );
}
