// LoadingState.tsx
// 通用加载状态组件，用于页面和局部区域加载反馈。

import { LoaderCircle } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * LoadingState 组件属性。
 */
export interface LoadingStateProps {
  title?: string;
  description?: string;
  className?: string;
}

/**
 * LoadingState 通用加载状态组件。
 */
export function LoadingState({ title = "正在加载", description = "请稍候，链镜正在准备数据。", className }: LoadingStateProps) {
  return (
    <div className={cn("flex min-h-48 flex-col items-center justify-center rounded-xl border border-border bg-card/70 p-8 text-center", className)}>
      <LoaderCircle className="mb-4 h-8 w-8 animate-spin text-primary" />
      <h2 className="font-display text-lg font-semibold">{title}</h2>
      <p className="mt-2 text-sm text-muted-foreground">{description}</p>
    </div>
  );
}
