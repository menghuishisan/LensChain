// Textarea.tsx
// 基础多行输入组件，用于表单长文本输入。

import { forwardRef, type TextareaHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

/**
 * Textarea 组件属性。
 */
export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  hasError?: boolean;
}

/**
 * Textarea 基础多行输入组件。
 */
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, hasError = false, rows = 4, ...props }, ref) => (
    <textarea
      ref={ref}
      rows={rows}
      className={cn(
        "w-full rounded-lg border bg-background/80 px-3 py-2 text-sm text-foreground shadow-sm outline-none transition placeholder:text-muted-foreground focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60",
        hasError ? "border-destructive focus:border-destructive focus:ring-destructive/20" : "border-input",
        className,
      )}
      {...props}
    />
  ),
);

Textarea.displayName = "Textarea";
