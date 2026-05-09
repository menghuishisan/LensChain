"use client";

// Sheet.tsx
// 侧边滑出抽屉组件，基于 Radix Dialog 封装。
// 从右侧滑出，用于不离开当前页面的参考内容展示。

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { forwardRef, type ComponentPropsWithoutRef, type ElementRef, type HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

/**
 * Sheet 根组件。
 */
export const Sheet = DialogPrimitive.Root;

/**
 * SheetTrigger 触发器。
 */
export const SheetTrigger = DialogPrimitive.Trigger;

/**
 * SheetClose 关闭器。
 */
export const SheetClose = DialogPrimitive.Close;

/**
 * SheetPortal 传送门。
 */
export const SheetPortal = DialogPrimitive.Portal;

/**
 * SheetOverlay 遮罩层。
 */
export const SheetOverlay = forwardRef<
  ElementRef<typeof DialogPrimitive.Overlay>,
  ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    className={cn(
      "fixed inset-0 z-50 bg-slate-950/40 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      className,
    )}
    {...props}
  />
));

SheetOverlay.displayName = DialogPrimitive.Overlay.displayName;

/**
 * SheetContent 从右侧滑出的内容面板。
 */
export const SheetContent = forwardRef<
  ElementRef<typeof DialogPrimitive.Content>,
  ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & { width?: string }
>(({ className, children, width = "w-[420px]", ...props }, ref) => (
  <SheetPortal>
    <SheetOverlay />
    <DialogPrimitive.Content
      ref={ref}
      className={cn(
        "fixed inset-y-0 right-0 z-50 flex flex-col border-l border-border bg-card text-card-foreground shadow-panel",
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right",
        "duration-200",
        width,
        className,
      )}
      {...props}
    >
      {children}
      <DialogPrimitive.Close className="absolute right-4 top-3 rounded-full p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground focus:outline-none focus:ring-2 focus:ring-ring">
        <X className="h-4 w-4" />
        <span className="sr-only">关闭</span>
      </DialogPrimitive.Close>
    </DialogPrimitive.Content>
  </SheetPortal>
));

SheetContent.displayName = "SheetContent";

/**
 * SheetHeader 抽屉标题区域。
 */
export function SheetHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("space-y-1 border-b border-border px-5 py-3", className)} {...props} />;
}

/**
 * SheetTitle 抽屉标题。
 */
export const SheetTitle = forwardRef<
  ElementRef<typeof DialogPrimitive.Title>,
  ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title ref={ref} className={cn("font-display text-base font-semibold", className)} {...props} />
));

SheetTitle.displayName = DialogPrimitive.Title.displayName;

/**
 * SheetDescription 抽屉描述。
 */
export const SheetDescription = forwardRef<
  ElementRef<typeof DialogPrimitive.Description>,
  ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description ref={ref} className={cn("text-sm text-muted-foreground", className)} {...props} />
));

SheetDescription.displayName = DialogPrimitive.Description.displayName;
