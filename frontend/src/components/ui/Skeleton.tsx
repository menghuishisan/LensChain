// Skeleton.tsx
// 骨架屏基础组件，提供可复用的占位块，用于数据加载前的视觉反馈。
// 包含基础占位块、卡片骨架、表格骨架、列表骨架、网格骨架和页面头部骨架。

import { cn } from "@/lib/utils";

// ────────────────── 列宽权重，让表格每行列宽随机变化更逼真 ──────────────────
const COL_WIDTHS = ["w-full", "w-4/5", "w-3/5", "w-2/5", "w-3/4", "w-1/2"] as const;

/**
 * Skeleton 基础骨架占位块属性。
 */
export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {}

/**
 * Skeleton 基础骨架占位块，渲染一个带脉冲动画的灰色矩形。
 */
export function Skeleton({ className, ...props }: SkeletonProps) {
  return <div className={cn("animate-pulse rounded-lg bg-muted", className)} {...props} />;
}

/**
 * SkeletonCard 属性。
 */
export interface SkeletonCardProps {
  className?: string;
}

/**
 * SkeletonCard 卡片骨架屏，模拟一张含标题、描述和内容行的卡片。
 */
export function SkeletonCard({ className }: SkeletonCardProps) {
  return (
    <div className={cn("rounded-xl border border-border bg-card/70 p-5 space-y-4", className)}>
      <div className="space-y-2">
        <Skeleton className="h-5 w-2/5" />
        <Skeleton className="h-4 w-3/5" />
      </div>
      <div className="space-y-3">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-4/5" />
        <Skeleton className="h-4 w-3/4" />
      </div>
    </div>
  );
}

/**
 * SkeletonTable 属性。
 */
export interface SkeletonTableProps {
  rows?: number;
  cols?: number;
  className?: string;
}

/**
 * SkeletonTable 表格骨架屏，模拟一个含表头和多行数据的表格。
 * 各行列宽交替变化，让视觉效果更贴近真实数据。
 */
export function SkeletonTable({ rows = 5, cols = 4, className }: SkeletonTableProps) {
  return (
    <div className={cn("rounded-xl border border-border bg-card/70 overflow-hidden", className)}>
      <div className="flex gap-4 border-b border-border bg-muted/40 px-4 py-3">
        {Array.from({ length: cols }).map((_, ci) => (
          <Skeleton key={ci} className={cn("h-4", ci === 0 ? "w-1/4" : "flex-1")} />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, ri) => (
        <div key={ri} className="flex gap-4 border-b border-border/50 px-4 py-3.5 last:border-0">
          {Array.from({ length: cols }).map((_, ci) => (
            <Skeleton
              key={ci}
              className={cn("h-4", ci === 0 ? "w-1/4" : "flex-1", COL_WIDTHS[(ri + ci) % COL_WIDTHS.length])}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

/**
 * SkeletonList 属性。
 */
export interface SkeletonListProps {
  count?: number;
  className?: string;
}

/**
 * SkeletonList 列表骨架屏，模拟多条带图标和操作按钮的列表项。
 */
export function SkeletonList({ count = 3, className }: SkeletonListProps) {
  return (
    <div className={cn("space-y-3", className)}>
      {Array.from({ length: count }).map((_, index) => (
        <div key={index} className="flex items-center gap-4 rounded-xl border border-border bg-card/70 p-4">
          <Skeleton className="h-10 w-10 shrink-0 rounded-lg" />
          <div className="flex-1 space-y-2">
            <Skeleton className={cn("h-4", COL_WIDTHS[index % 3])} />
            <Skeleton className={cn("h-3", COL_WIDTHS[(index + 2) % COL_WIDTHS.length])} />
          </div>
          <Skeleton className="h-8 w-16 rounded-md" />
        </div>
      ))}
    </div>
  );
}

/**
 * SkeletonGrid 属性。
 */
export interface SkeletonGridProps {
  count?: number;
  className?: string;
}

/**
 * SkeletonGrid 网格卡片骨架屏，模拟课程/竞赛/实验卡片网格。
 * 卡片顶部使用微透明主题色占位，底部模拟标题、描述和标签。
 */
export function SkeletonGrid({ count = 6, className }: SkeletonGridProps) {
  return (
    <div className={cn("grid gap-4 md:grid-cols-2 lg:grid-cols-3", className)}>
      {Array.from({ length: count }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border bg-card/70 overflow-hidden">
          <div className="h-32 animate-pulse bg-[linear-gradient(135deg,hsl(var(--muted)),hsl(var(--primary)/0.08))]" />
          <div className="p-4 space-y-3">
            <Skeleton className={cn("h-5", COL_WIDTHS[index % 3])} />
            <Skeleton className={cn("h-4", COL_WIDTHS[(index + 1) % COL_WIDTHS.length])} />
            <div className="flex gap-2">
              <Skeleton className="h-6 w-14 rounded-full" />
              <Skeleton className="h-6 w-14 rounded-full" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * SkeletonHero 属性。
 */
export interface SkeletonHeroProps {
  className?: string;
}

/**
 * SkeletonHero 页面头部骨架屏，模拟带渐变背景的 hero 区域。
 * 适用于带大标题和描述文字的页面顶部加载状态。
 */
export function SkeletonHero({ className }: SkeletonHeroProps) {
  return (
    <div
      className={cn(
        "rounded-3xl border border-border/70 p-6 space-y-3",
        "bg-[linear-gradient(135deg,hsl(var(--primary)/0.12),hsl(var(--primary)/0.06))]",
        className,
      )}
    >
      <Skeleton className="h-4 w-24 bg-primary/10" />
      <Skeleton className="h-8 w-2/5 bg-primary/10" />
      <Skeleton className="h-4 w-3/5 bg-primary/10" />
    </div>
  );
}
