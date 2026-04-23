// EmptyState.tsx
// 通用空状态组件，用于列表、面板和无数据说明页面。

import { Inbox } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

/**
 * EmptyState 组件属性。
 */
export interface EmptyStateProps {
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

/**
 * EmptyState 通用空状态组件。
 */
export function EmptyState({ title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card/70 p-10 text-center", className)}>
      <div className="mb-4 rounded-full bg-primary/10 p-3 text-primary">
        <Inbox className="h-7 w-7" />
      </div>
      <h2 className="font-display text-xl font-semibold">{title}</h2>
      {description ? <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">{description}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
