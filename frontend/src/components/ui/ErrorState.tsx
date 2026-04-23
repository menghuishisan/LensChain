// ErrorState.tsx
// 通用错误状态组件，用于页面错误边界和局部请求失败展示。

import { TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

/**
 * ErrorState 组件属性。
 */
export interface ErrorStateProps {
  title?: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

/**
 * ErrorState 通用错误状态组件。
 */
export function ErrorState({ title = "加载失败", description, action, className }: ErrorStateProps) {
  return (
    <div className={cn("rounded-xl border border-destructive/30 bg-destructive/6 p-6 text-center", className)}>
      <div className="mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-destructive/12 text-destructive">
        <TriangleAlert className="h-6 w-6" />
      </div>
      <h2 className="font-display text-lg font-semibold text-destructive">{title}</h2>
      {description ? <p className="mt-2 text-sm leading-6 text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
