// SchoolStatusBadge.tsx
// 模块02学校状态和入驻申请状态徽标，集中映射后端枚举显示。

import { Badge } from "@/components/ui/Badge";
import { getEnumText } from "@/lib/enum";
import type { SchoolApplicationStatus, SchoolStatus } from "@/types/school";

/**
 * 学校状态文本。
 */
export const SCHOOL_STATUS_TEXT: Record<SchoolStatus, string> = {
  1: "待审核",
  2: "已激活",
  3: "缓冲期",
  4: "已冻结",
  5: "已注销",
  6: "已拒绝",
};

/**
 * 入驻申请状态文本。
 */
export const SCHOOL_APPLICATION_STATUS_TEXT: Record<SchoolApplicationStatus, string> = {
  1: "待审核",
  2: "已通过",
  3: "已拒绝",
};

/**
 * SchoolStatusBadge 组件属性。
 */
export interface SchoolStatusBadgeProps {
  status: SchoolStatus;
  text?: string;
}

/**
 * SchoolStatusBadge 学校状态徽标。
 */
export function SchoolStatusBadge({ status, text }: SchoolStatusBadgeProps) {
  const variant = status === 2 ? "success" : status === 1 || status === 3 ? "warning" : status === 4 || status === 6 ? "destructive" : "secondary";
  return <Badge variant={variant}>{text ?? getEnumText(SCHOOL_STATUS_TEXT, status)}</Badge>;
}

/**
 * SchoolApplicationStatusBadge 入驻申请状态徽标。
 */
export function SchoolApplicationStatusBadge({ status, text }: { status: SchoolApplicationStatus; text?: string }) {
  const variant = status === 1 ? "warning" : status === 2 ? "success" : "destructive";
  return <Badge variant={variant}>{text ?? getEnumText(SCHOOL_APPLICATION_STATUS_TEXT, status)}</Badge>;
}
