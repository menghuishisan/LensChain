"use client";

// ConfirmDialog.tsx
// 通用确认弹窗组件，用于危险操作二次确认。

import type { ReactNode } from "react";

import { Button, type ButtonProps } from "@/components/ui/Button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/Dialog";

/**
 * ConfirmDialog 组件属性。
 */
export interface ConfirmDialogProps {
  title: string;
  description: string;
  trigger?: ReactNode;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: ButtonProps["variant"];
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onConfirm: () => void;
}

/**
 * ConfirmDialog 通用确认弹窗组件。
 */
export function ConfirmDialog({
  title,
  description,
  trigger,
  confirmText = "确认",
  cancelText = "取消",
  confirmVariant = "destructive",
  open,
  onOpenChange,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">{cancelText}</Button>
          </DialogClose>
          <DialogClose asChild>
            <Button variant={confirmVariant} onClick={onConfirm}>
              {confirmText}
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
