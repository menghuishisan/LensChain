// Badge.tsx
// 基础徽标组件，用于状态、角色和未读数展示。

import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

const badgeVariants = cva("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold", {
  variants: {
    variant: {
      default: "bg-primary/12 text-primary",
      secondary: "bg-secondary text-secondary-foreground",
      outline: "border border-border bg-background text-foreground",
      destructive: "bg-destructive/12 text-destructive",
      success: "bg-emerald-500/12 text-emerald-700 dark:text-emerald-300",
      warning: "bg-amber-500/12 text-amber-700 dark:text-amber-300",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

/**
 * Badge 组件属性。
 */
export interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

/**
 * Badge 基础徽标组件。
 *
 * 使用 forwardRef 暴露底层 span 的 ref，便于 Radix `<TooltipTrigger asChild>` /
 * `<PopoverTrigger asChild>` 之类把 ref 透传给 Badge（否则 React 会抛
 * "Function components cannot be given refs" 警告）。
 */
export const Badge = forwardRef<HTMLSpanElement, BadgeProps>(function Badge(
  { className, variant, ...props },
  ref,
) {
  return <span ref={ref} className={cn(badgeVariants({ variant }), className)} {...props} />;
});

Badge.displayName = "Badge";
