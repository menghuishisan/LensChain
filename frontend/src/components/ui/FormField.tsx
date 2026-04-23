// FormField.tsx
// 基础表单字段组件，统一 label、说明、错误提示和必填标记。

import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

/**
 * FormField 组件属性。
 */
export interface FormFieldProps {
  id?: string;
  label: string;
  required?: boolean;
  description?: string;
  error?: string;
  children: ReactNode;
  className?: string;
}

/**
 * FormField 基础表单字段组件。
 */
export function FormField({ id, label, required = false, description, error, children, className }: FormFieldProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <label htmlFor={id} className="text-sm font-semibold text-foreground">
        {label}
        {required ? <span className="ml-1 text-destructive">*</span> : null}
      </label>
      {children}
      {description && !error ? <p className="text-xs leading-5 text-muted-foreground">{description}</p> : null}
      {error ? <p className="text-xs leading-5 text-destructive">{error}</p> : null}
    </div>
  );
}
