"use client";

// page.tsx
// 课程评价页，对应模块03课程评价功能。

import { EvaluationPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";
import { useAuth } from "@/hooks/useAuth";

/**
 * CourseEvaluationsPage 课程评价页。
 */
export default function CourseEvaluationsPage({ params }: { params: { id: string } }) {
  const { roles } = useAuth();
  const evaluationRole = roles.includes("teacher") ? "teacher" : "student";
  return <PermissionGate allowedRoles={["student", "teacher"]}><EvaluationPanel courseID={params.id} role={evaluationRole} /></PermissionGate>;
}
