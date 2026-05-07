// LoadingState.tsx
// 通用加载状态组件，用于页面和局部区域加载反馈。
// 支持 variant 属性切换骨架屏样式：spinner（旋转图标）、card（默认）、table、list、grid、hero。

import { LoaderCircle } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  SkeletonCard,
  SkeletonGrid,
  SkeletonHero,
  SkeletonList,
  SkeletonTable,
} from "@/components/ui/Skeleton";

/**
 * LoadingState 组件属性。
 */
export interface LoadingStateProps {
  title?: string;
  description?: string;
  className?: string;
  /** 加载样式变体。spinner 为旋转图标，hero 为页面头部，其他为对应骨架屏。 */
  variant?: "spinner" | "card" | "table" | "list" | "grid" | "hero";
}

/**
 * LoadingState 通用加载状态组件。
 * 默认展示骨架屏卡片，传入 variant="spinner" 使用旋转图标。
 */
export function LoadingState({
  title = "正在加载",
  description = "请稍候，链镜正在准备数据。",
  className,
  variant = "card",
}: LoadingStateProps) {
  if (variant === "spinner") {
    return (
      <div className={cn("flex min-h-48 flex-col items-center justify-center rounded-xl border border-border bg-card/70 p-8 text-center", className)}>
        <LoaderCircle className="mb-4 h-8 w-8 animate-spin text-primary" />
        <h2 className="font-display text-lg font-semibold">{title}</h2>
        <p className="mt-2 text-sm text-muted-foreground">{description}</p>
      </div>
    );
  }

  if (variant === "hero") {
    return (
      <div className={cn("space-y-5", className)}>
        <SkeletonHero />
        <SkeletonCard />
      </div>
    );
  }

  if (variant === "table") {
    return (
      <div className={cn("space-y-4", className)}>
        <SkeletonTable />
      </div>
    );
  }

  if (variant === "list") {
    return (
      <div className={cn("space-y-4", className)}>
        <SkeletonList />
      </div>
    );
  }

  if (variant === "grid") {
    return (
      <div className={cn("space-y-5", className)}>
        <SkeletonHero />
        <SkeletonGrid />
      </div>
    );
  }

  return (
    <div className={cn("space-y-4", className)}>
      <SkeletonCard />
    </div>
  );
}
