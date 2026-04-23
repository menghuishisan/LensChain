// utils.ts
// 前端公共工具函数，当前提供 Tailwind className 合并能力。

import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * 合并 Tailwind CSS 类名，并处理条件 className。
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
