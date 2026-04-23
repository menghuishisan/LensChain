"use client";

// Toast.tsx
// 全局 Toast 反馈组件，基于 Radix Toast 提供成功、普通和错误提示。

import * as ToastPrimitive from "@radix-ui/react-toast";
import { createContext, type ReactNode, useContext, useState } from "react";

import { cn } from "@/lib/utils";

/**
 * Toast 提示类型。
 */
export type ToastVariant = "default" | "success" | "destructive";

/**
 * Toast 展示参数。
 */
export interface ToastInput {
  title: string;
  description?: string;
  variant?: ToastVariant;
}

interface ToastMessage extends ToastInput {
  id: string;
  variant: ToastVariant;
}

interface ToastContextValue {
  showToast: (input: ToastInput) => string;
  dismissToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

function createToastID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function getToastClassName(variant: ToastVariant) {
  if (variant === "success") {
    return "border-emerald-500/30 bg-emerald-50 text-emerald-950 dark:bg-emerald-950 dark:text-emerald-50";
  }

  if (variant === "destructive") {
    return "border-destructive/30 bg-destructive/10 text-destructive";
  }

  return "border-border bg-card text-card-foreground";
}

/**
 * ToastProvider 全局提示容器。
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const dismissToast = (id: string) => {
    setToasts((current) => current.filter((toastItem) => toastItem.id !== id));
  };

  const showToast = (input: ToastInput) => {
    const id = createToastID();
    setToasts((current) => [...current, { ...input, id, variant: input.variant ?? "default" }]);
    return id;
  };

  return (
    <ToastContext.Provider value={{ showToast, dismissToast }}>
      <ToastPrimitive.Provider swipeDirection="right">
        {children}
        {toasts.map((toastItem) => (
          <ToastPrimitive.Root
            key={toastItem.id}
            className={cn(
              "grid w-full gap-1 rounded-xl border p-4 shadow-panel data-[state=open]:animate-fade-up sm:max-w-sm",
              getToastClassName(toastItem.variant),
            )}
            onOpenChange={(open) => {
              if (!open) {
                dismissToast(toastItem.id);
              }
            }}
          >
            <ToastPrimitive.Title className="text-sm font-semibold">{toastItem.title}</ToastPrimitive.Title>
            {toastItem.description ? (
              <ToastPrimitive.Description className="text-sm opacity-80">{toastItem.description}</ToastPrimitive.Description>
            ) : null}
          </ToastPrimitive.Root>
        ))}
        <ToastViewport />
      </ToastPrimitive.Provider>
    </ToastContext.Provider>
  );
}

/**
 * ToastViewport 全局提示视口。
 */
export function ToastViewport() {
  return <ToastPrimitive.Viewport className="fixed right-4 top-4 z-[100] flex w-[calc(100%-2rem)] flex-col gap-3 sm:w-auto" />;
}

/**
 * useToast 获取全局 Toast 操作。
 */
export function useToast() {
  const context = useContext(ToastContext);
  if (context === null) {
    throw new Error("useToast 必须在 ToastProvider 内使用");
  }

  return context;
}
