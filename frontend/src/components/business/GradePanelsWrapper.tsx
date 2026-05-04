// GradePanelsWrapper.tsx
// 模块06页面包装组件，负责角色门禁和对应页面面板选择。

import Link from "next/link";

import { PermissionGate } from "@/components/business/PermissionGate";
import { buttonClassName } from "@/components/ui/Button";
import {
  AdminGradeAnalyticsPanel,
  AdminGradeReviewsPanel,
  AdminLevelConfigsPanel,
  AdminSemestersPanel,
  AdminWarningConfigPanel,
  AdminWarningsPanel,
  StudentGpaPanel,
  StudentGradeAppealsPanel,
  StudentGradesPanel,
  StudentTranscriptsPanel,
  SuperGradeAnalyticsPanel,
  TeacherGradeAnalyticsPanel,
  TeacherGradeAppealsPanel,
  TeacherGradeReviewsPanel,
} from "@/components/business/GradePanels";
import type { ID } from "@/types/api";

/**
 * GradePanelsStudentWrapper 学生端成绩页面包装组件。
 */
export function GradePanelsStudentWrapper({ page }: { page: "grades" | "gpa" | "appeals" | "transcripts" }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      {page === "grades" ? <StudentGradesPanel /> : null}
      {page === "gpa" ? <StudentGpaPanel /> : null}
      {page === "appeals" ? <StudentGradeAppealsPanel /> : null}
      {page === "transcripts" ? <StudentTranscriptsPanel /> : null}
    </PermissionGate>
  );
}

/**
 * GradePanelsTeacherWrapper 教师端成绩页面包装组件。
 */
export function GradePanelsTeacherWrapper({ page, courseID }: { page: "reviews" | "appeals" | "analytics"; courseID?: ID }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <div className="space-y-5">
        <div className="flex flex-wrap gap-2">
          <Link className={buttonClassName({ variant: "outline" })} href="/teacher/grades/reviews">成绩审核</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/teacher/grades/appeals">申诉处理</Link>
        </div>
        {page === "reviews" ? <TeacherGradeReviewsPanel /> : null}
        {page === "appeals" ? <TeacherGradeAppealsPanel /> : null}
        {page === "analytics" && courseID ? <TeacherGradeAnalyticsPanel courseID={courseID} /> : null}
      </div>
    </PermissionGate>
  );
}

/**
 * GradePanelsAdminWrapper 学校管理员端成绩页面包装组件。
 */
export function GradePanelsAdminWrapper({ page, reviewID }: { page: "semesters" | "level-configs" | "reviews" | "review-detail" | "warnings" | "warning-configs" | "analytics"; reviewID?: ID }) {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <div className="space-y-5">
        <div className="flex flex-wrap gap-2">
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/semesters">学期管理</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/level-configs">等级映射</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/reviews">成绩审核</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/warnings">学业预警</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/warning-configs">预警配置</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/admin/grades/analytics">全校分析</Link>
        </div>
        {page === "semesters" ? <AdminSemestersPanel /> : null}
        {page === "level-configs" ? <AdminLevelConfigsPanel /> : null}
        {page === "reviews" ? <AdminGradeReviewsPanel /> : null}
        {page === "review-detail" && reviewID ? <AdminGradeReviewsPanel reviewID={reviewID} /> : null}
        {page === "warnings" ? <AdminWarningsPanel /> : null}
        {page === "warning-configs" ? <AdminWarningConfigPanel /> : null}
        {page === "analytics" ? <AdminGradeAnalyticsPanel /> : null}
      </div>
    </PermissionGate>
  );
}

/**
 * GradePanelsSuperWrapper 超级管理员成绩总览包装组件。
 */
export function GradePanelsSuperWrapper() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SuperGradeAnalyticsPanel />
    </PermissionGate>
  );
}
