// grade.ts
// 模块06运行时枚举映射与下载工具，集中管理审核、申诉、预警状态展示。

import type { GradeAppealStatus, GradeReviewStatus, AcademicWarningStatus, AcademicWarningType } from "@/types/grade";

/**
 * 成绩审核状态选项。
 */
export const GRADE_REVIEW_STATUS_OPTIONS: Array<{ value: GradeReviewStatus; label: string }> = [
  { value: 1, label: "未提交" },
  { value: 2, label: "待审核" },
  { value: 3, label: "已通过" },
  { value: 4, label: "已驳回" },
];

/**
 * 成绩申诉状态选项。
 */
export const GRADE_APPEAL_STATUS_OPTIONS: Array<{ value: GradeAppealStatus; label: string }> = [
  { value: 1, label: "待处理" },
  { value: 2, label: "已同意" },
  { value: 3, label: "已驳回" },
];

/**
 * 学业预警状态选项。
 */
export const ACADEMIC_WARNING_STATUS_OPTIONS: Array<{ value: AcademicWarningStatus; label: string }> = [
  { value: 1, label: "待处理" },
  { value: 2, label: "已处理" },
  { value: 3, label: "已解除" },
];

/**
 * 学业预警类型选项。
 */
export const ACADEMIC_WARNING_TYPE_OPTIONS: Array<{ value: AcademicWarningType; label: string }> = [
  { value: 1, label: "低GPA" },
  { value: 2, label: "连续挂科" },
];

/**
 * getGradeReviewStatusVariant 获取成绩审核状态的徽标样式。
 */
export function getGradeReviewStatusVariant(status: GradeReviewStatus) {
  if (status === 3) {
    return "success" as const;
  }
  if (status === 4) {
    return "destructive" as const;
  }
  return "warning" as const;
}

/**
 * getGradeAppealStatusVariant 获取成绩申诉状态的徽标样式。
 */
export function getGradeAppealStatusVariant(status: GradeAppealStatus) {
  if (status === 2) {
    return "success" as const;
  }
  if (status === 3) {
    return "destructive" as const;
  }
  return "warning" as const;
}

/**
 * getAcademicWarningStatusVariant 获取学业预警状态的徽标样式。
 */
export function getAcademicWarningStatusVariant(status: AcademicWarningStatus) {
  if (status === 2 || status === 3) {
    return "success" as const;
  }
  return "destructive" as const;
}

/**
 * downloadBlob 统一处理成绩单下载等后端文件流。
 */
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
