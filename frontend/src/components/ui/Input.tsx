// Input.tsx
// 基础输入框组件，统一表单控件视觉与焦点状态。

import { forwardRef, type InputHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

/**
 * Input 组件属性。
 */
export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  hasError?: boolean;
}

/**
 * Input 基础输入框组件。
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(({ className, hasError = false, ...props }, ref) => (
  <input
    ref={ref}
    className={cn(
      "h-10 w-full rounded-lg border bg-background/80 px-3 text-sm text-foreground shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60",
      hasError ? "border-destructive focus:border-destructive focus:ring-destructive/20" : "border-input",
      className,
    )}
    {...props}
  />
));

Input.displayName = "Input";
