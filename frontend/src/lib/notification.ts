// notification.ts
// 模块07通知运行时工具：分类映射、来源跳转、安全预览和下载后的本地处理。

import type { NotificationCategory } from "@/types/notification";

/**
 * 通知分类标签。
 */
export const NOTIFICATION_CATEGORY_OPTIONS: Array<{ value: NotificationCategory; label: string }> = [
  { value: 1, label: "系统" },
  { value: 2, label: "课程" },
  { value: 3, label: "实验" },
  { value: 4, label: "竞赛" },
  { value: 5, label: "成绩" },
];

/**
 * getNotificationCategoryVariant 获取通知分类徽标样式。
 */
export function getNotificationCategoryVariant(category: NotificationCategory) {
  if (category === 1) {
    return "secondary" as const;
  }
  if (category === 5) {
    return "success" as const;
  }
  return "outline" as const;
}

/**
 * stripHtmlToText 安全移除模板预览或公告内容中的 HTML 标签，避免直接渲染不可信 HTML。
 */
export function stripHtmlToText(html: string) {
  return html
    .replace(/<script[\s\S]*?>[\s\S]*?<\/script>/gi, "")
    .replace(/<style[\s\S]*?>[\s\S]*?<\/style>/gi, "")
    .replace(/<[^>]+>/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

/**
 * resolveNotificationSourceHref 解析消息来源跳转目标。
 */
export function resolveNotificationSourceHref(sourceType: string | null, sourceID: string | null) {
  if (!sourceType || !sourceID) {
    return null;
  }
  if (sourceType === "assignment") {
    return `/student/assignments/${sourceID}`;
  }
  if (sourceType === "experiment") {
    return `/student/experiment-instances/${sourceID}`;
  }
  if (sourceType === "competition") {
    return `/ctf/${sourceID}`;
  }
  if (sourceType === "grade_review") {
    return "/student/grades";
  }
  if (sourceType === "grade_appeal") {
    return "/student/grades/appeals";
  }
  if (sourceType === "academic_warning") {
    return "/student/grades/gpa";
  }
  return null;
}
